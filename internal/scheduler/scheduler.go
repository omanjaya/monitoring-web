package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/cleanup"
	"github.com/diskominfos-bali/monitoring-website/internal/service/escalation"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
	"github.com/diskominfos-bali/monitoring-website/internal/service/summary"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type Scheduler struct {
	cfg                      *config.Config
	cron                     *cron.Cron
	uptimeMonitor            *monitor.UptimeMonitor
	sslChecker               *monitor.SSLChecker
	contentScanner           *monitor.ContentScanner
	dorkScanner              *monitor.DorkScanner
	vulnScanner              *monitor.VulnerabilityScanner
	dnsScanner               *monitor.DNSScanner
	securityChecker          *monitor.SecurityHeadersChecker
	defacementArchiveScanner *monitor.DefacementArchiveScanner
	telegramNotifier         *notifier.TelegramNotifier
	emailNotifier            *notifier.EmailNotifier
	webhookNotifier          *notifier.WebhookNotifier
	summaryService           *summary.Service
	cleanupService           *cleanup.CleanupService
	escalationService        *escalation.EscalationService
	settingsRepo             *mysql.SettingsRepository
	running                  bool
	mu                       sync.Mutex
	runningJobs              sync.Map // tracks which jobs are currently running
}

func NewScheduler(
	cfg *config.Config,
	uptimeMonitor *monitor.UptimeMonitor,
	sslChecker *monitor.SSLChecker,
	contentScanner *monitor.ContentScanner,
	dorkScanner *monitor.DorkScanner,
	vulnScanner *monitor.VulnerabilityScanner,
	dnsScanner *monitor.DNSScanner,
	securityChecker *monitor.SecurityHeadersChecker,
	defacementArchiveScanner *monitor.DefacementArchiveScanner,
	telegramNotifier *notifier.TelegramNotifier,
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	summaryService *summary.Service,
	cleanupService *cleanup.CleanupService,
	escalationService *escalation.EscalationService,
	settingsRepo *mysql.SettingsRepository,
) *Scheduler {
	return &Scheduler{
		cfg:                      cfg,
		cron:                     cron.New(cron.WithSeconds()),
		uptimeMonitor:            uptimeMonitor,
		sslChecker:               sslChecker,
		contentScanner:           contentScanner,
		dorkScanner:              dorkScanner,
		vulnScanner:              vulnScanner,
		dnsScanner:               dnsScanner,
		securityChecker:          securityChecker,
		defacementArchiveScanner: defacementArchiveScanner,
		telegramNotifier:         telegramNotifier,
		emailNotifier:            emailNotifier,
		webhookNotifier:          webhookNotifier,
		summaryService:           summaryService,
		cleanupService:           cleanupService,
		escalationService:        escalationService,
		settingsRepo:             settingsRepo,
	}
}

// minutesToCron converts an interval in minutes to a 6-field cron expression (with seconds).
func minutesToCron(minutes int) string {
	if minutes <= 0 {
		minutes = 5
	}
	if minutes < 60 {
		return fmt.Sprintf("0 */%d * * * *", minutes)
	}
	hours := minutes / 60
	if hours < 24 {
		return fmt.Sprintf("0 0 */%d * * *", hours)
	}
	return "0 0 0 * * *" // daily
}

// getSchedules reads intervals from database settings, falling back to config.yaml defaults.
func (s *Scheduler) getSchedules() (uptime, ssl, content, dork, vuln, dns, security string) {
	// Defaults from config.yaml
	uptime = s.cfg.Scheduler.UptimeCheck
	ssl = s.cfg.Scheduler.SSLCheck
	content = s.cfg.Scheduler.ContentScan
	dork = s.cfg.Scheduler.DorkScan
	vuln = s.cfg.Scheduler.VulnerabilityScan
	dns = s.cfg.Scheduler.DNSScan
	security = s.cfg.Scheduler.SecurityScan

	// Try to read from database settings (override config.yaml)
	if s.settingsRepo != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		monSettings, err := s.settingsRepo.GetMonitoringSettings(ctx)
		if err == nil {
			if monSettings.UptimeInterval > 0 {
				uptime = minutesToCron(monSettings.UptimeInterval)
			}
			if monSettings.SSLInterval > 0 {
				ssl = minutesToCron(monSettings.SSLInterval)
			}
			if monSettings.ContentInterval > 0 {
				content = minutesToCron(monSettings.ContentInterval)
			}
			if monSettings.DorkInterval > 0 {
				dork = minutesToCron(monSettings.DorkInterval)
			}
			if monSettings.VulnInterval > 0 {
				vuln = minutesToCron(monSettings.VulnInterval)
			}
			if monSettings.DNSInterval > 0 {
				dns = minutesToCron(monSettings.DNSInterval)
			}
			if monSettings.SecurityInterval > 0 {
				security = minutesToCron(monSettings.SecurityInterval)
			}
			logger.Info().Msg("Scheduler intervals loaded from database settings")
		} else {
			logger.Warn().Err(err).Msg("Failed to load settings from database, using config.yaml defaults")
		}
	}

	// Apply hardcoded fallbacks if still empty
	if uptime == "" {
		uptime = "0 */5 * * * *"
	}
	if ssl == "" {
		ssl = "0 0 */6 * * *"
	}
	if content == "" {
		content = "0 0 * * * *"
	}
	if dork == "" {
		dork = "0 0 */4 * * *"
	}
	if vuln == "" {
		vuln = "0 0 */12 * * *"
	}
	if dns == "" {
		dns = "0 0 */12 * * *"
	}
	if security == "" {
		security = "0 0 */12 * * *"
	}

	return
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	if err := s.setupCronJobs(); err != nil {
		return err
	}

	s.cron.Start()
	s.running = true

	logger.Info().Msg("Scheduler started")

	// Run initial scans on startup (in background)
	go func() {
		time.Sleep(30 * time.Second) // Wait for server to be fully ready
		logger.Info().Msg("Running initial SSL check on startup")
		s.runSSLCheck()
		logger.Info().Msg("Running initial security headers scan on startup")
		s.runSecurityScan()
		logger.Info().Msg("Running initial DNS scan on startup")
		s.runDNSScan()
	}()

	return nil
}

func (s *Scheduler) setupCronJobs() error {
	uptimeSchedule, sslSchedule, contentSchedule, dorkSchedule, vulnSchedule, dnsSchedule, securitySchedule := s.getSchedules()

	// Uptime check
	_, err := s.cron.AddFunc(uptimeSchedule, s.runUptimeCheck)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule uptime check")
		return err
	}
	logger.Info().Str("schedule", uptimeSchedule).Msg("Uptime check scheduled")

	// SSL check
	_, err = s.cron.AddFunc(sslSchedule, s.runSSLCheck)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule SSL check")
		return err
	}
	logger.Info().Str("schedule", sslSchedule).Msg("SSL check scheduled")

	// Content scan
	_, err = s.cron.AddFunc(contentSchedule, s.runContentScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule content scan")
		return err
	}
	logger.Info().Str("schedule", contentSchedule).Msg("Content scan scheduled")

	// Daily summary - from config only (not interval-based)
	dailySummarySchedule := s.cfg.Scheduler.DailySummary
	if dailySummarySchedule == "" {
		dailySummarySchedule = "0 0 8 * * *"
	}
	_, err = s.cron.AddFunc(dailySummarySchedule, s.runDailySummary)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule daily summary")
		return err
	}
	logger.Info().Str("schedule", dailySummarySchedule).Msg("Daily summary scheduled")

	// Database cleanup - from config only
	cleanupSchedule := s.cfg.Scheduler.Cleanup
	if cleanupSchedule == "" {
		cleanupSchedule = "0 0 3 * * *"
	}
	_, err = s.cron.AddFunc(cleanupSchedule, s.runCleanup)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule database cleanup")
		return err
	}
	logger.Info().Str("schedule", cleanupSchedule).Msg("Database cleanup scheduled")

	// Escalation check - always every 5 minutes
	_, err = s.cron.AddFunc("0 */5 * * * *", s.runEscalation)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule escalation check")
		return err
	}
	logger.Info().Str("schedule", "0 */5 * * * *").Msg("Escalation check scheduled")

	// Dork scan
	_, err = s.cron.AddFunc(dorkSchedule, s.runDorkScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule dork scan")
		return err
	}
	logger.Info().Str("schedule", dorkSchedule).Msg("Dork scan scheduled")

	// Vulnerability scan
	_, err = s.cron.AddFunc(vulnSchedule, s.runVulnerabilityScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule vulnerability scan")
		return err
	}
	logger.Info().Str("schedule", vulnSchedule).Msg("Vulnerability scan scheduled")

	// DNS scan
	_, err = s.cron.AddFunc(dnsSchedule, s.runDNSScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule DNS scan")
		return err
	}
	logger.Info().Str("schedule", dnsSchedule).Msg("DNS scan scheduled")

	// Security headers scan
	_, err = s.cron.AddFunc(securitySchedule, s.runSecurityScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule security scan")
		return err
	}
	logger.Info().Str("schedule", securitySchedule).Msg("Security headers scan scheduled")

	// Defacement archive scan - daily at 6 AM (fixed)
	_, err = s.cron.AddFunc("0 0 6 * * *", s.runDefacementArchiveScan)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule defacement archive scan")
		return err
	}
	logger.Info().Str("schedule", "0 0 6 * * *").Msg("Defacement archive scan scheduled")

	return nil
}

// ReloadSchedules stops the current cron, creates a new one with updated intervals from database, and starts it.
// Called when monitoring settings are updated via the UI.
func (s *Scheduler) ReloadSchedules() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	logger.Info().Msg("Reloading scheduler with updated settings...")

	// Stop current cron (wait for running jobs to finish)
	ctx := s.cron.Stop()
	<-ctx.Done()

	// Create new cron instance
	s.cron = cron.New(cron.WithSeconds())

	// Re-setup all cron jobs with new intervals
	if err := s.setupCronJobs(); err != nil {
		return fmt.Errorf("failed to setup cron jobs: %w", err)
	}

	// Start new cron
	s.cron.Start()

	logger.Info().Msg("Scheduler reloaded successfully")
	return nil
}

// GetActiveSchedules returns the current active cron schedules for display in the UI.
func (s *Scheduler) GetActiveSchedules() map[string]string {
	uptime, ssl, content, dork, vuln, dns, security := s.getSchedules()
	return map[string]string{
		"uptime":        uptime,
		"ssl":           ssl,
		"content":       content,
		"dork":          dork,
		"vulnerability": vuln,
		"dns":           dns,
		"security":      security,
		"daily_summary": s.cfg.Scheduler.DailySummary,
		"cleanup":       s.cfg.Scheduler.Cleanup,
		"escalation":    "0 */5 * * * *",
		"defacement":    "0 0 6 * * *",
	}
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false

	logger.Info().Msg("Scheduler stopped")
}

// tryRunJob attempts to run a job if it's not already running.
// Returns true if the job was executed, false if skipped due to overlap.
func (s *Scheduler) tryRunJob(name string, fn func()) bool {
	if _, loaded := s.runningJobs.LoadOrStore(name, true); loaded {
		logger.Warn().Str("job", name).Msg("Skipping job: previous run still in progress")
		return false
	}
	defer s.runningJobs.Delete(name)
	fn()
	return true
}

func (s *Scheduler) runUptimeCheck() {
	s.tryRunJob("uptime", func() {
		logger.Info().Msg("Starting scheduled uptime check")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()

		if err := s.uptimeMonitor.CheckAllWebsites(ctx); err != nil {
			logger.Error().Err(err).Msg("Uptime check failed")
		} else {
			logger.Info().Msg("Uptime check completed")
		}
	})
}

func (s *Scheduler) runSSLCheck() {
	s.tryRunJob("ssl", func() {
		logger.Info().Msg("Starting scheduled SSL check")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := s.sslChecker.CheckAllWebsites(ctx); err != nil {
			logger.Error().Err(err).Msg("SSL check failed")
		} else {
			logger.Info().Msg("SSL check completed")
		}
	})
}

func (s *Scheduler) runContentScan() {
	s.tryRunJob("content", func() {
		logger.Info().Msg("Starting scheduled content scan")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if err := s.contentScanner.ScanAllWebsites(ctx); err != nil {
			logger.Error().Err(err).Msg("Content scan failed")
		} else {
			logger.Info().Msg("Content scan completed")
		}
	})
}

func (s *Scheduler) runDailySummary() {
	s.tryRunJob("summary", func() {
		logger.Info().Msg("Generating daily summary")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		// Generate summary from database
		summaryData, err := s.summaryService.GenerateDailySummary(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to generate daily summary")
			return
		}

		// Send to all notification channels
		if err := s.telegramNotifier.SendDailySummary(ctx, summaryData); err != nil {
			logger.Error().Err(err).Msg("Failed to send Telegram daily summary")
		} else {
			logger.Info().Msg("Telegram daily summary sent")
		}

		if s.emailNotifier != nil {
			if err := s.emailNotifier.SendDailySummary(ctx, summaryData); err != nil {
				logger.Error().Err(err).Msg("Failed to send Email daily summary")
			} else {
				logger.Info().Msg("Email daily summary sent")
			}
		}

		if s.webhookNotifier != nil {
			if err := s.webhookNotifier.SendDailySummary(ctx, summaryData); err != nil {
				logger.Error().Err(err).Msg("Failed to send Webhook daily summary")
			} else {
				logger.Info().Msg("Webhook daily summary sent")
			}
		}
	})
}

func (s *Scheduler) runCleanup() {
	s.tryRunJob("cleanup", func() {
		logger.Info().Msg("Starting scheduled database cleanup")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if s.cleanupService != nil {
			result, err := s.cleanupService.RunCleanup(ctx)
			if err != nil {
				logger.Error().Err(err).Msg("Database cleanup failed")
			} else {
				logger.Info().
					Int64("checks_deleted", result.ChecksDeleted).
					Int64("alerts_deleted", result.AlertsDeleted).
					Msg("Database cleanup completed")
			}
		}
	})
}

func (s *Scheduler) runEscalation() {
	s.tryRunJob("escalation", func() {
		logger.Info().Msg("Starting scheduled escalation check")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		if s.escalationService != nil {
			if err := s.escalationService.ProcessEscalations(ctx); err != nil {
				logger.Error().Err(err).Msg("Escalation check failed")
			} else {
				logger.Info().Msg("Escalation check completed")
			}
		}
	})
}

func (s *Scheduler) runDorkScan() {
	s.tryRunJob("dork", func() {
		logger.Info().Msg("Starting scheduled dork scan (Judol/Defacement detection)")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()

		if s.dorkScanner != nil {
			if err := s.dorkScanner.ScanAllWebsites(ctx); err != nil {
				logger.Error().Err(err).Msg("Dork scan failed")
			} else {
				logger.Info().Msg("Dork scan completed")
			}
		}
	})
}

func (s *Scheduler) runVulnerabilityScan() {
	s.tryRunJob("vulnerability", func() {
		logger.Info().Msg("Starting scheduled vulnerability scan")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()

		if s.vulnScanner != nil {
			if err := s.vulnScanner.ScanAllWebsites(ctx); err != nil {
				logger.Error().Err(err).Msg("Vulnerability scan failed")
			} else {
				logger.Info().Msg("Vulnerability scan completed")
			}
		}
	})
}

func (s *Scheduler) runDefacementArchiveScan() {
	s.tryRunJob("defacement-archive", func() {
		logger.Info().Msg("Starting defacement archive scan (Zone-H, Zone-XSEC)")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		if s.defacementArchiveScanner != nil {
			if err := s.defacementArchiveScanner.ScanAll(ctx); err != nil {
				logger.Error().Err(err).Msg("Defacement archive scan failed")
			} else {
				logger.Info().Msg("Defacement archive scan completed")
			}
		}
	})
}

func (s *Scheduler) runDNSScan() {
	s.tryRunJob("dns", func() {
		// Check if DNS scan is disabled via settings (dns_interval = 0)
		if s.settingsRepo != nil {
			ctx := context.Background()
			monSettings, err := s.settingsRepo.GetMonitoringSettings(ctx)
			if err == nil && monSettings.DNSInterval <= 0 {
				logger.Info().Msg("DNS scan is disabled in settings (interval = 0), skipping")
				return
			}
		}

		logger.Info().Msg("Starting scheduled DNS scan")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()

		if s.dnsScanner != nil {
			if err := s.dnsScanner.ScanAllWebsites(ctx); err != nil {
				logger.Error().Err(err).Msg("DNS scan failed")
			} else {
				logger.Info().Msg("DNS scan completed")
			}
		}
	})
}

func (s *Scheduler) runSecurityScan() {
	s.tryRunJob("security", func() {
		logger.Info().Msg("Starting scheduled security headers scan")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
		defer cancel()

		if s.securityChecker != nil {
			if err := s.securityChecker.CheckAllWebsites(ctx); err != nil {
				logger.Error().Err(err).Msg("Security headers scan failed")
			} else {
				logger.Info().Msg("Security headers scan completed")
			}
		}
	})
}

// RunNow executes a specific check immediately
func (s *Scheduler) RunNow(checkType string) error {
	switch checkType {
	case "uptime":
		go s.runUptimeCheck()
	case "ssl":
		go s.runSSLCheck()
	case "content":
		go s.runContentScan()
	case "summary":
		go s.runDailySummary()
	case "cleanup":
		go s.runCleanup()
	case "escalation":
		go s.runEscalation()
	case "dork":
		go s.runDorkScan()
	case "vulnerability":
		go s.runVulnerabilityScan()
	case "dns":
		go s.runDNSScan()
	case "security":
		go s.runSecurityScan()
	case "defacement":
		go s.runDefacementArchiveScan()
	default:
		return nil
	}
	return nil
}
