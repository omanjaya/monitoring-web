package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/handler"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/scheduler"
	alertservice "github.com/diskominfos-bali/monitoring-website/internal/service/alert"
	"github.com/diskominfos-bali/monitoring-website/internal/service/auth"
	"github.com/diskominfos-bali/monitoring-website/internal/service/cleanup"
	"github.com/diskominfos-bali/monitoring-website/internal/service/escalation"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
	"github.com/diskominfos-bali/monitoring-website/internal/service/report"
	"github.com/diskominfos-bali/monitoring-website/internal/service/settings"
	"github.com/diskominfos-bali/monitoring-website/internal/service/summary"
	"github.com/diskominfos-bali/monitoring-website/internal/service/website"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

func main() {
	// Initialize logger
	logger.Init()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	logger.Info().Str("mode", cfg.Server.Mode).Msg("Configuration loaded")

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Warn().Err(err).Msg("Configuration validation warnings")
	}

	// Connect to database
	db, err := mysql.Connect(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	logger.Info().Msg("Database connected")

	// Run database migrations
	if err := mysql.RunMigrations(db); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Initialize repositories
	websiteRepo := mysql.NewWebsiteRepository(db)
	checkRepo := mysql.NewCheckRepository(db)
	alertRepo := mysql.NewAlertRepository(db)
	userRepo := mysql.NewUserRepository(db)
	keywordRepo := mysql.NewKeywordRepository(db)
	opdRepo := mysql.NewOPDRepository(db)
	maintenanceRepo := mysql.NewMaintenanceRepository(db)
	escalationRepo := mysql.NewEscalationRepository(db)
	dorkRepo := mysql.NewDorkRepository(db)
	settingsRepo := mysql.NewSettingsRepository(db)
	auditRepo := mysql.NewAuditRepository(db)
	dnsRepo := mysql.NewDNSRepository(db)
	defacementRepo := mysql.NewDefacementRepository(db)

	// Initialize services
	authService := auth.NewService(cfg, userRepo)
	websiteService := website.NewService(cfg, websiteRepo, checkRepo, alertRepo, opdRepo)
	uptimeMonitor := monitor.NewUptimeMonitor(cfg, websiteRepo, checkRepo, alertRepo)
	sslChecker := monitor.NewSSLChecker(cfg, websiteRepo, checkRepo, alertRepo)
	dorkScanner := monitor.NewDorkScanner(cfg, websiteRepo, dorkRepo, alertRepo, settingsRepo)
	contentScanner := monitor.NewContentScanner(cfg, websiteRepo, checkRepo, alertRepo, keywordRepo, dorkScanner)
	securityChecker := monitor.NewSecurityHeadersChecker(cfg, websiteRepo, checkRepo, alertRepo)
	telegramNotifier := notifier.NewTelegramNotifier(cfg, alertRepo)
	emailNotifier := notifier.NewEmailNotifier(cfg, alertRepo)
	webhookNotifier := notifier.NewWebhookNotifier(cfg, alertRepo)
	// Initialize digest service for notification batching (if enabled)
	var digestService *notifier.DigestService
	if cfg.Notification.DigestEnabled {
		digestService = notifier.NewDigestService(cfg, telegramNotifier, emailNotifier, webhookNotifier)
		digestService.Start()
		logger.Info().Msg("Notification digest mode enabled")
	}

	// Initialize alert service with digest support
	_ = alertservice.NewService(cfg, alertRepo, websiteRepo, telegramNotifier, emailNotifier, webhookNotifier, digestService)

	summaryService := summary.NewService(websiteRepo, checkRepo, alertRepo)
	cleanupService := cleanup.NewCleanupService(cfg, checkRepo, alertRepo)
	reportService := report.NewReportService(cfg, websiteRepo, checkRepo, alertRepo)
	escalationService := escalation.NewEscalationService(cfg, escalationRepo, alertRepo, websiteRepo, telegramNotifier, emailNotifier, webhookNotifier)
	vulnScanner := monitor.NewVulnerabilityScanner(cfg, websiteRepo, checkRepo, alertRepo)
	dnsScanner := monitor.NewDNSScanner(cfg, websiteRepo, dnsRepo)
	defacementScanner := monitor.NewDefacementArchiveScanner(cfg, websiteRepo, defacementRepo, alertRepo)
	settingsService := settings.NewService(cfg, settingsRepo)

	// Initialize scheduler
	sched := scheduler.NewScheduler(cfg, uptimeMonitor, sslChecker, contentScanner, dorkScanner, vulnScanner, dnsScanner, securityChecker, defacementScanner, telegramNotifier, emailNotifier, webhookNotifier, summaryService, cleanupService, escalationService, settingsRepo)

	// Initialize handlers
	healthHandler := handler.NewHealthHandler(db)
	authHandler := handler.NewAuthHandler(authService)
	websiteHandler := handler.NewWebsiteHandler(websiteService, dnsScanner, dnsRepo)
	alertHandler := handler.NewAlertHandler(alertRepo)
	dashboardHandler := handler.NewDashboardHandler(websiteService, alertRepo, checkRepo)
	adminHandler := handler.NewAdminHandler(sched, telegramNotifier, emailNotifier, webhookNotifier, websiteRepo, settingsRepo)
	keywordHandler := handler.NewKeywordHandler(keywordRepo)
	statusHandler := handler.NewStatusHandler(cfg, websiteRepo, checkRepo, alertRepo, maintenanceRepo)
	maintenanceHandler := handler.NewMaintenanceHandler(maintenanceRepo)
	securityHandler := handler.NewSecurityHandler(securityChecker, checkRepo, websiteRepo)
	reportHandler := handler.NewReportHandler(reportService)
	escalationHandler := handler.NewEscalationHandler(escalationService)
	dorkHandler := handler.NewDorkHandler(dorkScanner, dorkRepo, settingsRepo)
	vulnerabilityHandler := handler.NewVulnerabilityHandler(vulnScanner, checkRepo, websiteRepo)
	defacementHandler := handler.NewDefacementHandler(defacementScanner, defacementRepo)
	settingsHandler := handler.NewSettingsHandler(settingsService, sched)
	userHandler := handler.NewUserHandler(userRepo)
	auditHandler := handler.NewAuditHandler(auditRepo)

	// Setup router
	router := handler.NewRouter(healthHandler, authHandler, websiteHandler, alertHandler, dashboardHandler, adminHandler, keywordHandler, statusHandler, maintenanceHandler, securityHandler, reportHandler, escalationHandler, dorkHandler, vulnerabilityHandler, defacementHandler, settingsHandler, userHandler, auditHandler, authService, auditRepo, cfg)

	// Create Gin engine
	if cfg.Server.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(gin.Recovery())

	// Custom logger middleware
	engine.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Msg("Request")
	})

	// Setup routes
	router.Setup(engine)

	// Start scheduler
	if err := sched.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start scheduler")
	}

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: engine,
	}

	// Graceful shutdown
	go func() {
		logger.Info().Str("addr", addr).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down server...")

	// Stop digest service (flushes remaining alerts)
	if digestService != nil {
		digestService.Stop()
	}

	// Stop scheduler
	sched.Stop()

	// Shutdown HTTP server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error().Err(err).Msg("Server shutdown error")
	}

	logger.Info().Msg("Server exited")
}
