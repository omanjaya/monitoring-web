package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type SecurityHeadersChecker struct {
	cfg         *config.Config
	websiteRepo *mysql.WebsiteRepository
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
	httpClient  *http.Client
}

func NewSecurityHeadersChecker(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *SecurityHeadersChecker {
	client := &http.Client{
		Timeout: time.Duration(cfg.Monitoring.HTTPTimeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects, we want to check the actual response
			return http.ErrUseLastResponse
		},
	}

	return &SecurityHeadersChecker{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
		httpClient:  client,
	}
}

// CheckWebsite performs security headers check on a single website
func (s *SecurityHeadersChecker) CheckWebsite(ctx context.Context, website *domain.Website) (*domain.SecurityHeaderCheck, error) {
	result := &domain.SecurityHeaderCheck{
		WebsiteID: website.ID,
		CheckedAt: time.Now(),
	}

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "HEAD", website.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check security headers
	var headerResults []domain.HeaderResult
	var findings []domain.SecurityFinding
	totalScore := 0
	maxScore := 0

	for _, headerDef := range domain.SecurityHeaders {
		maxScore += headerDef.Weight
		headerValue := resp.Header.Get(headerDef.Name)

		hr := domain.HeaderResult{
			Name:        headerDef.Name,
			Present:     headerValue != "",
			Value:       headerValue,
			Expected:    headerDef.Required,
			Description: headerDef.Description,
			Impact:      headerDef.Impact,
			MaxPoints:   headerDef.Weight,
		}

		if headerValue != "" {
			hr.Points = headerDef.Weight
			totalScore += headerDef.Weight

			// Validate specific header values
			validationFindings := s.validateHeaderValue(headerDef.Name, headerValue)
			if len(validationFindings) > 0 {
				findings = append(findings, validationFindings...)
				// Reduce score for misconfigured headers
				hr.Points = hr.Points / 2
				totalScore -= hr.Points / 2
			}
		} else if headerDef.Required {
			// Missing required header
			findings = append(findings, domain.SecurityFinding{
				Type:        "missing_header",
				Severity:    headerDef.Impact,
				Title:       "Missing " + headerDef.Name + " Header",
				Description: headerDef.Description,
				Remedy:      getHeaderRemedy(headerDef.Name),
			})
		}

		headerResults = append(headerResults, hr)
	}

	// Check Server header for information disclosure (not scored, info only)
	serverValue := resp.Header.Get("Server")
	if serverValue != "" {
		// Check if server header reveals version info (e.g., "Apache/2.4.51", "nginx/1.21.0")
		versionPattern := regexp.MustCompile(`/[\d]+\.[\d]+`)
		if versionPattern.MatchString(serverValue) {
			findings = append(findings, domain.SecurityFinding{
				Type:        "server_info_disclosure",
				Severity:    "info",
				Title:       "Server Header Reveals Version Information",
				Description: fmt.Sprintf("Server header discloses software version: %s. This information can help attackers target known vulnerabilities.", serverValue),
				Remedy:      "Remove or obfuscate the Server header to hide version information",
			})
		}
	}

	// Check X-Powered-By header (should NOT be present)
	xPoweredBy := resp.Header.Get("X-Powered-By")
	if xPoweredBy != "" {
		totalScore -= 5
		findings = append(findings, domain.SecurityFinding{
			Type:        "info_disclosure",
			Severity:    "medium",
			Title:       "X-Powered-By Header Present",
			Description: fmt.Sprintf("X-Powered-By header discloses technology: %s. This information aids attackers.", xPoweredBy),
			Remedy:      "Remove the X-Powered-By header from server responses",
		})
		headerResults = append(headerResults, domain.HeaderResult{
			Name:        "X-Powered-By",
			Present:     true,
			Value:       xPoweredBy,
			Expected:    false,
			Description: "Should not be present - reveals technology stack information",
			Impact:      "medium",
			Points:      -5,
			MaxPoints:   0,
		})
	}

	// Calculate final score (0-100)
	if maxScore > 0 {
		result.Score = (totalScore * 100) / maxScore
	}
	if result.Score < 0 {
		result.Score = 0
	} else if result.Score > 100 {
		result.Score = 100
	}
	result.Grade = domain.CalculateGrade(result.Score)

	// Serialize headers and findings
	headersJSON, _ := json.Marshal(headerResults)
	result.Headers = domain.NewNullString(string(headersJSON))

	findingsJSON, _ := json.Marshal(findings)
	result.Findings = domain.NewNullString(string(findingsJSON))

	// Save result
	id, err := s.checkRepo.CreateSecurityHeaderCheck(ctx, result)
	if err != nil {
		logger.Error().Err(err).Int64("website_id", website.ID).Msg("Failed to save security headers check")
		return nil, err
	}
	result.ID = id

	// Update website security score
	s.websiteRepo.UpdateSecurityScore(ctx, website.ID, result.Score, result.Grade)

	// Handle alerts for poor security
	if result.Score < 50 {
		s.createSecurityAlert(ctx, website, result, findings)
	}

	return result, nil
}

// validateHeaderValue checks if header value is correctly configured
func (s *SecurityHeadersChecker) validateHeaderValue(headerName, value string) []domain.SecurityFinding {
	var findings []domain.SecurityFinding

	switch headerName {
	case "Content-Security-Policy":
		if strings.Contains(value, "unsafe-inline") {
			findings = append(findings, domain.SecurityFinding{
				Type:        "weak_csp",
				Severity:    "medium",
				Title:       "CSP allows unsafe-inline",
				Description: "Content-Security-Policy contains 'unsafe-inline' which weakens XSS protection",
				Remedy:      "Remove 'unsafe-inline' and use nonces or hashes instead",
			})
		}
		if strings.Contains(value, "unsafe-eval") {
			findings = append(findings, domain.SecurityFinding{
				Type:        "weak_csp",
				Severity:    "medium",
				Title:       "CSP allows unsafe-eval",
				Description: "Content-Security-Policy contains 'unsafe-eval' which can enable code injection",
				Remedy:      "Remove 'unsafe-eval' from your CSP policy",
			})
		}

	case "Strict-Transport-Security":
		if !strings.Contains(value, "max-age") {
			findings = append(findings, domain.SecurityFinding{
				Type:        "weak_hsts",
				Severity:    "medium",
				Title:       "HSTS missing max-age",
				Description: "Strict-Transport-Security should include max-age directive",
				Remedy:      "Add max-age directive with at least 31536000 (1 year)",
			})
		}
		if strings.Contains(value, "max-age=0") {
			findings = append(findings, domain.SecurityFinding{
				Type:        "weak_hsts",
				Severity:    "high",
				Title:       "HSTS disabled with max-age=0",
				Description: "HSTS is effectively disabled with max-age=0",
				Remedy:      "Set max-age to at least 31536000 (1 year)",
			})
		}

	case "X-Frame-Options":
		valueLower := strings.ToLower(value)
		if valueLower != "deny" && valueLower != "sameorigin" && !strings.HasPrefix(valueLower, "allow-from") {
			findings = append(findings, domain.SecurityFinding{
				Type:        "invalid_xfo",
				Severity:    "medium",
				Title:       "Invalid X-Frame-Options value",
				Description: "X-Frame-Options should be DENY, SAMEORIGIN, or ALLOW-FROM",
				Remedy:      "Set X-Frame-Options to DENY or SAMEORIGIN",
			})
		}
	}

	return findings
}

// getHeaderRemedy returns remedy text for missing header
func getHeaderRemedy(headerName string) string {
	remedies := map[string]string{
		"Content-Security-Policy":          "Add Content-Security-Policy header: default-src 'self'; script-src 'self'",
		"Strict-Transport-Security":        "Add Strict-Transport-Security: max-age=31536000; includeSubDomains",
		"X-Frame-Options":                  "Add X-Frame-Options: DENY or SAMEORIGIN",
		"X-Content-Type-Options":           "Add X-Content-Type-Options: nosniff",
		"X-XSS-Protection":                 "Add X-XSS-Protection: 1; mode=block",
		"Referrer-Policy":                  "Add Referrer-Policy: strict-origin-when-cross-origin",
		"Permissions-Policy":               "Add Permissions-Policy header to control browser features",
		"X-Permitted-Cross-Domain-Policies": "Add X-Permitted-Cross-Domain-Policies: none",
		"Cache-Control":                    "Add Cache-Control: no-store, no-cache, must-revalidate",
		"Cross-Origin-Embedder-Policy":     "Add Cross-Origin-Embedder-Policy: require-corp",
		"Cross-Origin-Opener-Policy":       "Add Cross-Origin-Opener-Policy: same-origin",
		"Cross-Origin-Resource-Policy":     "Add Cross-Origin-Resource-Policy: same-origin",
	}

	if remedy, ok := remedies[headerName]; ok {
		return remedy
	}
	return "Add appropriate " + headerName + " header"
}

// createSecurityAlert creates alert for poor security score
func (s *SecurityHeadersChecker) createSecurityAlert(ctx context.Context, website *domain.Website, result *domain.SecurityHeaderCheck, findings []domain.SecurityFinding) {
	// Check if alert already exists
	existingAlert, _ := s.alertRepo.GetLatestAlertByType(ctx, website.ID, domain.AlertTypeSecurityIssue)
	if existingAlert != nil && !existingAlert.IsResolved {
		return
	}

	var criticalFindings, highFindings int
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			criticalFindings++
		case "high":
			highFindings++
		}
	}

	severity := domain.SeverityWarning
	if criticalFindings > 0 || result.Score < 30 {
		severity = domain.SeverityCritical
	}

	message := "Website memiliki skor keamanan rendah.\n"
	message += fmt.Sprintf("Score: %s (%d/100)\n", result.Grade, result.Score)
	if criticalFindings > 0 {
		message += fmt.Sprintf("Critical issues: %d\n", criticalFindings)
	}
	if highFindings > 0 {
		message += fmt.Sprintf("High issues: %d\n", highFindings)
	}

	alert := &domain.AlertCreate{
		WebsiteID: website.ID,
		Type:      domain.AlertTypeSecurityIssue,
		Severity:  severity,
		Title:     "Keamanan Headers Rendah: " + website.Name,
		Message:   message,
		Context: map[string]interface{}{
			"score":           result.Score,
			"grade":           result.Grade,
			"findings_count":  len(findings),
			"critical_count":  criticalFindings,
			"high_count":      highFindings,
		},
	}

	_, err := s.alertRepo.Create(ctx, alert)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create security alert")
	}
}

// CheckAllWebsites checks security headers for all active websites
func (s *SecurityHeadersChecker) CheckAllWebsites(ctx context.Context) error {
	websites, err := s.websiteRepo.GetActive(ctx)
	if err != nil {
		return err
	}

	logger.Info().Int("count", len(websites)).Msg("Starting security headers check for all websites")

	// Use concurrent limiting
	maxConcurrent := s.cfg.Monitoring.MaxConcurrentChecks
	if maxConcurrent <= 0 {
		maxConcurrent = 10
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, website := range websites {
		// Only check HTTPS websites
		if !strings.HasPrefix(website.URL, "https://") {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			wg.Add(1)
			sem <- struct{}{}

			go func(w domain.Website) {
				defer wg.Done()
				defer func() { <-sem }()

				_, err := s.CheckWebsite(ctx, &w)
				if err != nil {
					logger.Error().Err(err).Str("url", w.URL).Msg("Security headers check failed")
				}
			}(website)
		}
	}

	wg.Wait()
	return nil
}
