package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type UptimeMonitor struct {
	cfg          *config.Config
	websiteRepo  *mysql.WebsiteRepository
	checkRepo    *mysql.CheckRepository
	alertRepo    *mysql.AlertRepository
	httpClient   *http.Client
}

func NewUptimeMonitor(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *UptimeMonitor {
	// Create HTTP client with custom transport
	// InsecureSkipVerify: true for uptime checks because:
	// 1. SSL validity is checked separately by SSLChecker
	// 2. Many government websites use self-signed or internal CA certificates
	// 3. Uptime check purpose is to verify if website is reachable, not SSL validity
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		ForceAttemptHTTP2:   cfg.Monitoring.EnableHTTP2,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Monitoring.HTTPTimeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &UptimeMonitor{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
		httpClient:  client,
	}
}

// CheckWebsite performs an uptime check on a single website
func (m *UptimeMonitor) CheckWebsite(ctx context.Context, website *domain.Website) (*domain.Check, error) {
	check := &domain.Check{
		WebsiteID: website.ID,
	}

	startTime := time.Now()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", website.URL, nil)
	if err != nil {
		check.Status = domain.CheckStatusError
		check.ErrorMessage = domain.NewNullString(err.Error())
		check.ErrorType = domain.NewNullString("request_error")
		return m.saveCheck(ctx, check, website)
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	// Perform request
	resp, err := m.httpClient.Do(req)
	responseTime := int(time.Since(startTime).Milliseconds())
	check.ResponseTime = domain.NewNullInt32(int32(responseTime))

	if err != nil {
		check.Status = m.classifyError(err)
		check.ErrorMessage = domain.NewNullString(err.Error())
		check.ErrorType = domain.NewNullString(string(check.Status))
		return m.saveCheck(ctx, check, website)
	}
	defer resp.Body.Close()

	// Record status code
	check.StatusCode = domain.NewNullInt32(int32(resp.StatusCode))
	check.ContentLength = domain.NewNullInt32(int32(resp.ContentLength))

	// Capture HTTP protocol version (e.g. "HTTP/1.1", "HTTP/2.0")
	check.Protocol = resp.Proto

	// Resolve IPv4 and IPv6 addresses for the domain
	if m.cfg.Monitoring.EnableIPv6 {
		parsedURL, parseErr := url.Parse(website.URL)
		if parseErr == nil {
			ipv4, ipv6 := m.resolveAddresses(ctx, parsedURL.Hostname())
			check.IPv4Addresses = ipv4
			check.IPv6Addresses = ipv6
			check.SupportsIPv6 = len(ipv6) > 0
		}
	}

	logger.Debug().
		Str("website", website.Name).
		Str("protocol", check.Protocol).
		Bool("ipv6", check.SupportsIPv6).
		Msg("Uptime check completed")

	// Determine status (using per-website thresholds if set)
	check.Status = m.classifyStatusWithThresholds(resp.StatusCode, responseTime, website)

	return m.saveCheck(ctx, check, website)
}

func (m *UptimeMonitor) classifyError(err error) domain.CheckStatus {
	errStr := err.Error()

	if contains(errStr, "timeout") || contains(errStr, "deadline exceeded") {
		return domain.CheckStatusTimeout
	}
	if contains(errStr, "no such host") || contains(errStr, "dns") {
		return domain.CheckStatusError
	}
	if contains(errStr, "connection refused") || contains(errStr, "connection reset") {
		return domain.CheckStatusDown
	}

	return domain.CheckStatusError
}

func (m *UptimeMonitor) classifyStatus(statusCode int, responseTime int) domain.CheckStatus {
	return m.classifyStatusWithThresholds(statusCode, responseTime, nil)
}

// classifyStatusWithThresholds determines the status with optional per-website threshold overrides
func (m *UptimeMonitor) classifyStatusWithThresholds(statusCode int, responseTime int, website *domain.Website) domain.CheckStatus {
	// Determine thresholds: per-website overrides global config
	warningThreshold := m.cfg.Monitoring.ResponseTimeWarning
	criticalThreshold := m.cfg.Monitoring.ResponseTimeCritical
	if website != nil {
		if website.ResponseTimeWarning.Valid {
			warningThreshold = int(website.ResponseTimeWarning.Int32)
		}
		if website.ResponseTimeCritical.Valid {
			criticalThreshold = int(website.ResponseTimeCritical.Int32)
		}
	}

	// Check if response is too slow
	if responseTime > criticalThreshold {
		return domain.CheckStatusDegraded
	}

	// Check HTTP status code
	if statusCode >= 200 && statusCode < 400 {
		if responseTime > warningThreshold {
			return domain.CheckStatusDegraded
		}
		return domain.CheckStatusUp
	}

	if statusCode >= 500 {
		return domain.CheckStatusDown
	}

	if statusCode >= 400 {
		return domain.CheckStatusError
	}

	return domain.CheckStatusUp
}

func (m *UptimeMonitor) saveCheck(ctx context.Context, check *domain.Check, website *domain.Website) (*domain.Check, error) {
	// Save check to database
	id, err := m.checkRepo.CreateCheck(ctx, check)
	if err != nil {
		logger.Error().Err(err).Int64("website_id", website.ID).Msg("Failed to save check")
		return nil, err
	}
	check.ID = id

	// Update website status
	statusCode := 0
	if check.StatusCode.Valid {
		statusCode = int(check.StatusCode.Int32)
	}
	responseTime := 0
	if check.ResponseTime.Valid {
		responseTime = int(check.ResponseTime.Int32)
	}

	var websiteStatus domain.WebsiteStatus
	switch check.Status {
	case domain.CheckStatusUp:
		websiteStatus = domain.StatusUp
	case domain.CheckStatusDown, domain.CheckStatusError:
		websiteStatus = domain.StatusDown
	case domain.CheckStatusTimeout, domain.CheckStatusDegraded:
		websiteStatus = domain.StatusDegraded
	default:
		websiteStatus = domain.StatusUnknown
	}

	err = m.websiteRepo.UpdateStatus(ctx, website.ID, websiteStatus, statusCode, responseTime)
	if err != nil {
		logger.Error().Err(err).Int64("website_id", website.ID).Msg("Failed to update website status")
	}

	// Check if we need to create/resolve alerts
	m.handleAlerts(ctx, check, website)

	return check, nil
}

func (m *UptimeMonitor) handleAlerts(ctx context.Context, check *domain.Check, website *domain.Website) {
	previousStatus := website.Status

	// Get all unresolved alert types for this website to enable grouping
	unresolvedTypes, _ := m.alertRepo.GetUnresolvedAlertTypes(ctx, website.ID)
	unresolvedTypeSet := make(map[string]bool)
	for _, t := range unresolvedTypes {
		unresolvedTypeSet[t] = true
	}

	// Website timed out (slow) — treat as degraded, not down
	if check.Status == domain.CheckStatusTimeout && previousStatus == domain.StatusUp {
		if !unresolvedTypeSet[string(domain.AlertTypeSlowResponse)] {
			alert := &domain.AlertCreate{
				WebsiteID: website.ID,
				Type:      domain.AlertTypeSlowResponse,
				Severity:  domain.SeverityWarning,
				Title:     fmt.Sprintf("Website TIMEOUT (Lambat): %s", website.Name),
				Message:   fmt.Sprintf("Website %s tidak merespons dalam batas waktu (timeout). Kemungkinan server sangat lambat.", website.URL),
				Context: map[string]interface{}{
					"error_type": check.ErrorType.String,
				},
			}
			if _, err := m.alertRepo.Create(ctx, alert); err != nil {
				logger.Error().Err(err).Msg("Failed to create TIMEOUT alert")
			} else {
				logger.Info().Str("website", website.Name).Msg("TIMEOUT/slow alert created")
			}
		}
		return
	}

	// Website went DOWN
	if (check.Status == domain.CheckStatusDown || check.Status == domain.CheckStatusError) &&
		previousStatus == domain.StatusUp {

		// Check for existing unresolved DOWN alert to avoid duplicates
		if unresolvedTypeSet[string(domain.AlertTypeDown)] {
			logger.Debug().Int64("website_id", website.ID).Str("type", "down").Msg("Skipping duplicate alert")
			return
		}

		// Check cooldown: don't re-alert within 15 minutes of a resolved DOWN alert
		hasCooldown, _ := m.alertRepo.HasRecentResolvedAlert(ctx, website.ID, domain.AlertTypeDown, 15)
		if hasCooldown {
			logger.Debug().Int64("website_id", website.ID).Str("type", "down").Msg("Skipping alert - within cooldown period")
			return
		}

		// Create DOWN alert
		alert := &domain.AlertCreate{
			WebsiteID: website.ID,
			Type:      domain.AlertTypeDown,
			Severity:  domain.SeverityCritical,
			Title:     fmt.Sprintf("Website DOWN: %s", website.Name),
			Message:   fmt.Sprintf("Website %s tidak dapat diakses. Error: %s", website.URL, check.ErrorMessage.String),
			Context: map[string]interface{}{
				"status_code":   check.StatusCode.Int32,
				"response_time": check.ResponseTime.Int32,
				"error_type":    check.ErrorType.String,
			},
		}

		_, err := m.alertRepo.Create(ctx, alert)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create DOWN alert")
		} else {
			logger.Info().Str("website", website.Name).Msg("DOWN alert created")
		}
	}

	// Website came back UP
	if check.Status == domain.CheckStatusUp && previousStatus == domain.StatusDown {
		// Resolve ALL existing DOWN and SLOW_RESPONSE alerts for this website
		m.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeDown, "Website kembali online secara otomatis")
		m.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeSlowResponse, "Website kembali normal secara otomatis")

		// Create UP alert (info) and immediately resolve it — it's just a notification record
		if !unresolvedTypeSet[string(domain.AlertTypeUp)] {
			alert := &domain.AlertCreate{
				WebsiteID: website.ID,
				Type:      domain.AlertTypeUp,
				Severity:  domain.SeverityInfo,
				Title:     fmt.Sprintf("Website UP: %s", website.Name),
				Message:   fmt.Sprintf("Website %s kembali online. Response time: %dms", website.URL, check.ResponseTime.Int32),
				Context: map[string]interface{}{
					"status_code":   check.StatusCode.Int32,
					"response_time": check.ResponseTime.Int32,
				},
			}
			if newID, err := m.alertRepo.Create(ctx, alert); err == nil {
				// Auto-resolve UP alerts — they are informational notifications, not incidents
				m.alertRepo.Resolve(ctx, newID, 0, "Auto-resolved: UP alert adalah notifikasi informatif")
			}
		}
		logger.Info().Str("website", website.Name).Msg("Website is back UP")
	}

	// Website response normalized (Degraded → Up)
	if check.Status == domain.CheckStatusUp && previousStatus == domain.StatusDegraded {
		m.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeSlowResponse, "Website kembali normal secara otomatis")
	}

	// Slow response warning
	if check.Status == domain.CheckStatusDegraded && previousStatus == domain.StatusUp {
		// Don't create SLOW_RESPONSE if site is already DOWN - slow response is expected
		if unresolvedTypeSet[string(domain.AlertTypeDown)] {
			logger.Debug().Int64("website_id", website.ID).Msg("Skipping slow_response alert - site has unresolved DOWN alert")
			return
		}

		// Determine effective threshold for alert context
		effectiveWarning := m.cfg.Monitoring.ResponseTimeWarning
		if website.ResponseTimeWarning.Valid {
			effectiveWarning = int(website.ResponseTimeWarning.Int32)
		}

		// Check for existing unresolved slow response alert to avoid duplicates
		if unresolvedTypeSet[string(domain.AlertTypeSlowResponse)] {
			logger.Debug().Int64("website_id", website.ID).Str("type", "slow_response").Msg("Skipping duplicate alert")
			return
		}

		// Check cooldown: don't re-alert within 15 minutes of a resolved slow_response alert
		hasCooldown, _ := m.alertRepo.HasRecentResolvedAlert(ctx, website.ID, domain.AlertTypeSlowResponse, 15)
		if hasCooldown {
			logger.Debug().Int64("website_id", website.ID).Str("type", "slow_response").Msg("Skipping alert - within cooldown period")
			return
		}

		alert := &domain.AlertCreate{
			WebsiteID: website.ID,
			Type:      domain.AlertTypeSlowResponse,
			Severity:  domain.SeverityWarning,
			Title:     fmt.Sprintf("Slow Response: %s", website.Name),
			Message:   fmt.Sprintf("Website %s merespons lambat: %dms", website.URL, check.ResponseTime.Int32),
			Context: map[string]interface{}{
				"response_time": check.ResponseTime.Int32,
				"threshold":     effectiveWarning,
			},
		}

		m.alertRepo.Create(ctx, alert)
	}
}

// CheckAllWebsites checks all active websites with rate limiting
// Rate limited to 10 websites per minute to avoid IP bans
func (m *UptimeMonitor) CheckAllWebsites(ctx context.Context) error {
	websites, err := m.websiteRepo.GetActive(ctx)
	if err != nil {
		return err
	}

	totalWebsites := len(websites)
	logger.Info().Int("count", totalWebsites).Msg("Starting uptime check for all websites (rate limited: 10/min)")

	// Rate limiting: 10 websites per minute = 1 website every 6 seconds
	rateLimitPerMinute := m.cfg.Monitoring.RateLimitPerMinute
	if rateLimitPerMinute <= 0 {
		rateLimitPerMinute = 10 // default 10 per minute
	}

	delayBetweenChecks := time.Minute / time.Duration(rateLimitPerMinute)

	// Estimate total time
	estimatedMinutes := float64(totalWebsites) / float64(rateLimitPerMinute)
	logger.Info().
		Float64("estimated_minutes", estimatedMinutes).
		Int("rate_per_minute", rateLimitPerMinute).
		Msg("Estimated completion time")

	// Process websites sequentially with rate limiting
	for i, website := range websites {
		select {
		case <-ctx.Done():
			logger.Warn().Int("checked", i).Int("total", totalWebsites).Msg("Check cancelled")
			return ctx.Err()
		default:
			// Log progress every 10 websites
			if i > 0 && i%10 == 0 {
				logger.Info().
					Int("checked", i).
					Int("total", totalWebsites).
					Int("remaining", totalWebsites-i).
					Msg("Check progress")
			}

			_, err := m.CheckWebsite(ctx, &website)
			if err != nil {
				logger.Error().Err(err).Str("url", website.URL).Msg("Check failed")
			}

			// Apply rate limiting delay (except for the last website)
			if i < totalWebsites-1 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(delayBetweenChecks):
					// Continue to next website
				}
			}
		}
	}

	logger.Info().Int("count", totalWebsites).Msg("Completed uptime check for all websites")
	return nil
}

// resolveAddresses resolves both IPv4 and IPv6 addresses for a hostname
func (m *UptimeMonitor) resolveAddresses(ctx context.Context, host string) (ipv4 []string, ipv6 []string) {
	// Use a short timeout for DNS resolution to avoid blocking the check
	resolveCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(resolveCtx, host)
	if err != nil {
		logger.Debug().Err(err).Str("host", host).Msg("Failed to resolve addresses")
		return nil, nil
	}

	for _, ip := range ips {
		if ip.IP.To4() != nil {
			ipv4 = append(ipv4, ip.IP.String())
		} else {
			ipv6 = append(ipv6, ip.IP.String())
		}
	}
	return ipv4, ipv6
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
