package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type SSLChecker struct {
	cfg         *config.Config
	websiteRepo *mysql.WebsiteRepository
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
}

func NewSSLChecker(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *SSLChecker {
	return &SSLChecker{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
	}
}

// CheckSSL performs SSL certificate check on a website
func (s *SSLChecker) CheckSSL(ctx context.Context, website *domain.Website) (*domain.SSLCheck, error) {
	sslCheck := &domain.SSLCheck{
		WebsiteID: website.ID,
	}

	// Parse URL to get host
	parsedURL, err := url.Parse(website.URL)
	if err != nil {
		sslCheck.IsValid = false
		sslCheck.ErrorMessage = domain.NewNullString(err.Error())
		return s.saveSSLCheck(ctx, sslCheck, website)
	}

	// Only check HTTPS
	if parsedURL.Scheme != "https" {
		sslCheck.IsValid = false
		sslCheck.ErrorMessage = domain.NewNullString("Website does not use HTTPS")
		return s.saveSSLCheck(ctx, sslCheck, website)
	}

	host := parsedURL.Host
	hostname := parsedURL.Hostname() // without port
	if parsedURL.Port() == "" {
		host = host + ":443"
	}

	dialer := &net.Dialer{
		Timeout: time.Duration(s.cfg.Monitoring.HTTPTimeout) * time.Second,
	}

	// First try with strict verification
	conn, err := tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         hostname,
	})

	chainValid := true
	if err != nil {
		// If chain verification fails, try again without verification to get cert info
		// This handles cases where server doesn't send intermediate certificates
		if strings.Contains(err.Error(), "unknown authority") || strings.Contains(err.Error(), "certificate signed by") {
			chainValid = false
			conn, err = tls.DialWithDialer(dialer, "tcp", host, &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         hostname,
			})
		}

		if err != nil {
			sslCheck.IsValid = false
			sslCheck.ErrorMessage = domain.NewNullString(err.Error())
			return s.saveSSLCheck(ctx, sslCheck, website)
		}
	}
	defer conn.Close()

	// Get certificate info
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		sslCheck.IsValid = false
		sslCheck.ErrorMessage = domain.NewNullString("No certificates found")
		return s.saveSSLCheck(ctx, sslCheck, website)
	}

	cert := certs[0]

	// Extract certificate info
	sslCheck.Issuer = domain.NewNullString(cert.Issuer.CommonName)
	sslCheck.Subject = domain.NewNullString(cert.Subject.CommonName)
	sslCheck.ValidFrom = domain.NewNullTime(cert.NotBefore)
	sslCheck.ValidUntil = domain.NewNullTime(cert.NotAfter)
	sslCheck.Protocol = domain.NewNullString(tlsVersionString(conn.ConnectionState().Version))

	// Calculate days until expiry
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)
	sslCheck.DaysUntilExpiry = domain.NewNullInt32(int32(daysUntilExpiry))

	// Check validity
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		sslCheck.IsValid = false
		sslCheck.ErrorMessage = domain.NewNullString("Certificate is expired or not yet valid")
		return s.saveSSLCheck(ctx, sslCheck, website)
	}

	// Check if certificate matches hostname
	if err := cert.VerifyHostname(hostname); err != nil {
		sslCheck.IsValid = false
		sslCheck.ErrorMessage = domain.NewNullString("Certificate does not match hostname: " + err.Error())
		return s.saveSSLCheck(ctx, sslCheck, website)
	}

	// Certificate is valid
	sslCheck.IsValid = true

	// Add warning if chain verification failed but cert is otherwise valid
	if !chainValid {
		sslCheck.ErrorMessage = domain.NewNullString("Warning: Server doesn't send intermediate certificate (chain incomplete)")
		logger.Warn().Str("website", website.Name).Msg("SSL chain incomplete but certificate is valid")
	}

	return s.saveSSLCheck(ctx, sslCheck, website)
}

func (s *SSLChecker) saveSSLCheck(ctx context.Context, sslCheck *domain.SSLCheck, website *domain.Website) (*domain.SSLCheck, error) {
	// Save SSL check to database
	id, err := s.checkRepo.CreateSSLCheck(ctx, sslCheck)
	if err != nil {
		logger.Error().Err(err).Int64("website_id", website.ID).Msg("Failed to save SSL check")
		return nil, err
	}
	sslCheck.ID = id

	// Update website SSL info
	var expiryDate *string
	if sslCheck.ValidUntil.Valid {
		dateStr := sslCheck.ValidUntil.Time.Format("2006-01-02")
		expiryDate = &dateStr
	}
	s.websiteRepo.UpdateSSLInfo(ctx, website.ID, sslCheck.IsValid, expiryDate)

	// Handle alerts
	s.handleAlerts(ctx, sslCheck, website)

	return sslCheck, nil
}

func (s *SSLChecker) handleAlerts(ctx context.Context, sslCheck *domain.SSLCheck, website *domain.Website) {
	// SSL is expired
	if !sslCheck.IsValid {
		exists, _ := s.alertRepo.HasRecentUnresolvedAlert(ctx, website.ID, domain.AlertTypeSSLExpired)
		if !exists {
			alert := &domain.AlertCreate{
				WebsiteID: website.ID,
				Type:      domain.AlertTypeSSLExpired,
				Severity:  domain.SeverityCritical,
				Title:     fmt.Sprintf("SSL Expired: %s", website.Name),
				Message:   fmt.Sprintf("SSL certificate untuk %s tidak valid. Error: %s", website.URL, sslCheck.ErrorMessage.String),
				Context: map[string]interface{}{
					"issuer": sslCheck.Issuer.String,
				},
			}
			s.alertRepo.Create(ctx, alert)
			logger.Warn().Str("website", website.Name).Msg("SSL expired alert created")
		}
		return
	}

	// Resolve existing expired alerts if SSL is now valid
	existingExpiredAlert, _ := s.alertRepo.GetLatestAlertByType(ctx, website.ID, domain.AlertTypeSSLExpired)
	if existingExpiredAlert != nil && !existingExpiredAlert.IsResolved {
		s.alertRepo.Resolve(ctx, existingExpiredAlert.ID, 0, "SSL certificate sudah valid")
	}

	// Check for expiring soon (30, 14, 7 days)
	if sslCheck.DaysUntilExpiry.Valid {
		days := int(sslCheck.DaysUntilExpiry.Int32)

		warningThresholds := []int{30, 14, 7}
		for _, threshold := range warningThresholds {
			if days <= threshold && days > 0 {
				// Check if we already have an unresolved alert (deduplication)
				existsExpiring, _ := s.alertRepo.HasRecentUnresolvedAlert(ctx, website.ID, domain.AlertTypeSSLExpiring)
				if existsExpiring {
					logger.Debug().Int64("website_id", website.ID).Str("type", "ssl_expiring").Msg("Skipping duplicate alert")
					break
				}

				severity := domain.SeverityWarning
				if days <= 7 {
					severity = domain.SeverityCritical
				}

				alert := &domain.AlertCreate{
					WebsiteID: website.ID,
					Type:      domain.AlertTypeSSLExpiring,
					Severity:  severity,
					Title:     fmt.Sprintf("SSL Expiring Soon: %s", website.Name),
					Message:   fmt.Sprintf("SSL certificate untuk %s akan expired dalam %d hari (%s)", website.URL, days, sslCheck.ValidUntil.Time.Format("02 Jan 2006")),
					Context: map[string]interface{}{
						"days_until_expiry": days,
						"expiry_date":       sslCheck.ValidUntil.Time.Format("2006-01-02"),
						"issuer":            sslCheck.Issuer.String,
					},
				}
				s.alertRepo.Create(ctx, alert)
				logger.Warn().Str("website", website.Name).Int("days", days).Msg("SSL expiring alert created")
				break
			}
		}
	}
}

// CheckAllWebsites checks SSL for all active websites with concurrent limiting
func (s *SSLChecker) CheckAllWebsites(ctx context.Context) error {
	websites, err := s.websiteRepo.GetActive(ctx)
	if err != nil {
		return err
	}

	logger.Info().Int("count", len(websites)).Msg("Starting SSL check for all websites")

	// Use concurrent limiting
	maxConcurrent := s.cfg.Monitoring.MaxConcurrentChecks
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // default
	}

	// Create semaphore channel
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, website := range websites {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Only check HTTPS websites
			parsedURL, err := url.Parse(website.URL)
			if err != nil || parsedURL.Scheme != "https" {
				continue
			}

			wg.Add(1)
			// Acquire semaphore
			sem <- struct{}{}

			go func(w domain.Website) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				_, err := s.CheckSSL(ctx, &w)
				if err != nil {
					logger.Error().Err(err).Str("url", w.URL).Msg("SSL check failed")
				}
			}(website)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return nil
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return "Unknown"
	}
}
