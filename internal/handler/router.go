package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/handler/middleware"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/auth"
)

type Router struct {
	healthHandler        *HealthHandler
	authHandler          *AuthHandler
	websiteHandler       *WebsiteHandler
	alertHandler         *AlertHandler
	dashboardHandler     *DashboardHandler
	adminHandler         *AdminHandler
	keywordHandler       *KeywordHandler
	statusHandler        *StatusHandler
	maintenanceHandler   *MaintenanceHandler
	securityHandler      *SecurityHandler
	reportHandler        *ReportHandler
	escalationHandler    *EscalationHandler
	dorkHandler          *DorkHandler
	vulnerabilityHandler *VulnerabilityHandler
	defacementHandler    *DefacementHandler
	settingsHandler      *SettingsHandler
	userHandler          *UserHandler
	auditHandler         *AuditHandler
	authService          *auth.Service
	auditRepo            *mysql.AuditRepository
	cfg                  *config.Config
}

func NewRouter(
	healthHandler *HealthHandler,
	authHandler *AuthHandler,
	websiteHandler *WebsiteHandler,
	alertHandler *AlertHandler,
	dashboardHandler *DashboardHandler,
	adminHandler *AdminHandler,
	keywordHandler *KeywordHandler,
	statusHandler *StatusHandler,
	maintenanceHandler *MaintenanceHandler,
	securityHandler *SecurityHandler,
	reportHandler *ReportHandler,
	escalationHandler *EscalationHandler,
	dorkHandler *DorkHandler,
	vulnerabilityHandler *VulnerabilityHandler,
	defacementHandler *DefacementHandler,
	settingsHandler *SettingsHandler,
	userHandler *UserHandler,
	auditHandler *AuditHandler,
	authService *auth.Service,
	auditRepo *mysql.AuditRepository,
	cfg *config.Config,
) *Router {
	return &Router{
		healthHandler:        healthHandler,
		authHandler:          authHandler,
		websiteHandler:       websiteHandler,
		alertHandler:         alertHandler,
		dashboardHandler:     dashboardHandler,
		adminHandler:         adminHandler,
		keywordHandler:       keywordHandler,
		statusHandler:        statusHandler,
		maintenanceHandler:   maintenanceHandler,
		securityHandler:      securityHandler,
		reportHandler:        reportHandler,
		escalationHandler:    escalationHandler,
		dorkHandler:          dorkHandler,
		vulnerabilityHandler: vulnerabilityHandler,
		defacementHandler:    defacementHandler,
		settingsHandler:      settingsHandler,
		userHandler:          userHandler,
		auditHandler:         auditHandler,
		authService:          authService,
		auditRepo:            auditRepo,
		cfg:                  cfg,
	}
}

func (r *Router) Setup(engine *gin.Engine) {
	// Health check
	engine.GET("/health", r.healthHandler.HealthCheck)

	// Security headers middleware
	engine.Use(middleware.SecurityHeadersMiddlewareWithConfig(r.cfg))

	// CORS middleware
	engine.Use(middleware.CORSMiddlewareWithConfig(r.cfg))

	// Public Status Page API (no authentication required)
	status := engine.Group("/status")
	{
		status.GET("", r.statusHandler.GetStatusOverview)
		status.GET("/summary", r.statusHandler.GetStatusSummary)
		status.GET("/badge", r.statusHandler.GetStatusBadge)
		status.GET("/services", r.statusHandler.GetAllServices)
		status.GET("/services/:id", r.statusHandler.GetServiceStatus)
		status.GET("/services/:id/history", r.statusHandler.GetServiceHistory)
		status.GET("/incidents", r.statusHandler.GetRecentIncidents)
		status.GET("/maintenance", r.statusHandler.GetPublicMaintenance)
	}

	// API routes
	api := engine.Group("/api")
	api.Use(middleware.RateLimitMiddleware(100, time.Minute))
	{
		// Public routes
		api.POST("/auth/login", middleware.LoginRateLimitMiddleware(), r.authHandler.Login)

		// Protected routes
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(r.authService))
		protected.Use(middleware.AuditMiddleware(r.auditRepo))
		{
			// Auth
			protected.GET("/auth/me", r.authHandler.GetMe)
			protected.PUT("/auth/password", r.authHandler.ChangePassword)

			// Dashboard
			protected.GET("/dashboard", r.dashboardHandler.GetDashboardOverview)
			protected.GET("/dashboard/stats", r.dashboardHandler.GetDashboardStats)
			protected.GET("/dashboard/trends", r.dashboardHandler.GetDashboardTrends)

			// Websites
			protected.GET("/websites", r.websiteHandler.ListWebsites)
			protected.POST("/websites", r.websiteHandler.CreateWebsite)
			protected.POST("/websites/bulk", r.websiteHandler.BulkImport)
			protected.POST("/websites/bulk-action", r.websiteHandler.BulkAction)
			protected.GET("/websites/:id", r.websiteHandler.GetWebsite)
			protected.PUT("/websites/:id", r.websiteHandler.UpdateWebsite)
			protected.DELETE("/websites/:id", r.websiteHandler.DeleteWebsite)
			protected.GET("/websites/:id/uptime", r.websiteHandler.GetUptimeHistory)
			protected.GET("/websites/:id/metrics", r.websiteHandler.GetWebsiteMetrics)
			protected.GET("/websites/:id/dns", r.websiteHandler.GetDNSScan)

			// DNS
			protected.GET("/dns/summary", r.websiteHandler.GetDNSSummary)

			// OPD
			protected.GET("/opd", r.websiteHandler.ListOPDs)
			protected.POST("/opd", r.websiteHandler.CreateOPD)
			protected.POST("/opd/bulk", r.websiteHandler.BulkImportOPD)

			// Alerts
			protected.GET("/alerts", r.alertHandler.ListAlerts)
			protected.GET("/alerts/active", r.alertHandler.GetActiveAlerts)
			protected.GET("/alerts/summary", r.alertHandler.GetAlertSummary)
			protected.POST("/alerts/bulk-resolve", r.alertHandler.BulkResolveAlerts)
			protected.GET("/alerts/:id", r.alertHandler.GetAlert)
			protected.POST("/alerts/:id/acknowledge", r.alertHandler.AcknowledgeAlert)
			protected.POST("/alerts/:id/resolve", r.alertHandler.ResolveAlert)

			// Admin
			adminRateLimit := middleware.RateLimitMiddleware(30, time.Minute)
			protected.GET("/admin/status", r.adminHandler.GetSystemStatus)
			protected.GET("/admin/schedules", r.adminHandler.GetSchedules)
			protected.POST("/admin/trigger", adminRateLimit, r.adminHandler.TriggerCheck)
			protected.POST("/admin/test-telegram", adminRateLimit, r.adminHandler.TestTelegram)
			protected.POST("/admin/test-email", adminRateLimit, r.adminHandler.TestEmail)
			protected.POST("/admin/test-webhook", adminRateLimit, r.adminHandler.TestWebhook)

			// Keywords
			protected.GET("/keywords", r.keywordHandler.ListKeywords)
			protected.POST("/keywords", r.keywordHandler.CreateKeyword)
			protected.POST("/keywords/bulk", r.keywordHandler.BulkImportKeywords)
			protected.DELETE("/keywords/:id", r.keywordHandler.DeleteKeyword)

			// Maintenance Windows
			protected.GET("/maintenance", r.maintenanceHandler.ListMaintenance)
			protected.GET("/maintenance/current", r.maintenanceHandler.GetCurrentMaintenance)
			protected.GET("/maintenance/upcoming", r.maintenanceHandler.GetUpcomingMaintenance)
			protected.POST("/maintenance", r.maintenanceHandler.CreateMaintenance)
			protected.GET("/maintenance/:id", r.maintenanceHandler.GetMaintenance)
			protected.PUT("/maintenance/:id", r.maintenanceHandler.UpdateMaintenance)
			protected.DELETE("/maintenance/:id", r.maintenanceHandler.DeleteMaintenance)
			protected.POST("/maintenance/:id/cancel", r.maintenanceHandler.CancelMaintenance)
			protected.POST("/maintenance/:id/complete", r.maintenanceHandler.CompleteMaintenance)

			// Security Headers
			protected.GET("/security/stats", r.securityHandler.GetSecurityStats)
			protected.GET("/security/summary", r.securityHandler.GetSecuritySummary)
			protected.GET("/security/websites/:id", r.securityHandler.GetWebsiteSecurity)
			protected.GET("/security/websites/:id/history", r.securityHandler.GetWebsiteSecurityHistory)
			protected.POST("/security/websites/:id/check", adminRateLimit, r.securityHandler.TriggerSecurityCheck)
			protected.POST("/security/check-all", adminRateLimit, r.securityHandler.TriggerAllSecurityChecks)

			// Reports
			protected.GET("/reports/types", r.reportHandler.GetReportTypes)
			protected.POST("/reports/generate", r.reportHandler.GenerateReport)
			protected.GET("/reports/quick/:type/:period", r.reportHandler.GetQuickReport)
			protected.GET("/reports/schedule/options", r.reportHandler.GetScheduledReportOptions)

			// Escalation
			protected.GET("/escalation/policies", r.escalationHandler.ListPolicies)
			protected.GET("/escalation/policies/:id", r.escalationHandler.GetPolicy)
			protected.POST("/escalation/policies", r.escalationHandler.CreatePolicy)
			protected.PUT("/escalation/policies/:id", r.escalationHandler.UpdatePolicy)
			protected.DELETE("/escalation/policies/:id", r.escalationHandler.DeletePolicy)
			protected.POST("/escalation/rules", r.escalationHandler.CreateRule)
			protected.DELETE("/escalation/rules/:id", r.escalationHandler.DeleteRule)
			protected.GET("/escalation/history", r.escalationHandler.GetHistory)
			protected.GET("/escalation/summary", r.escalationHandler.GetSummary)
			protected.POST("/escalation/trigger", adminRateLimit, r.escalationHandler.TriggerEscalation)

			// Dork Monitoring (Judol/Defacement Detection)
			protected.GET("/dork/stats", r.dorkHandler.GetOverallStats)
			protected.GET("/dork/categories", r.dorkHandler.GetCategories)
			protected.GET("/dork/patterns", r.dorkHandler.ListPatterns)
			protected.POST("/dork/patterns", r.dorkHandler.CreatePattern)
			protected.PUT("/dork/patterns/:id", r.dorkHandler.UpdatePattern)
			protected.DELETE("/dork/patterns/:id", r.dorkHandler.DeletePattern)
			protected.GET("/dork/detections", r.dorkHandler.ListDetections)
			protected.GET("/dork/detections/:id", r.dorkHandler.GetDetection)
			protected.POST("/dork/detections/:id/resolve", r.dorkHandler.ResolveDetection)
			protected.POST("/dork/detections/:id/false-positive", r.dorkHandler.MarkFalsePositive)
			protected.GET("/dork/websites/:id/stats", r.dorkHandler.GetWebsiteStats)
			protected.GET("/dork/websites/:id/scans", r.dorkHandler.ListScanResults)
			protected.GET("/dork/websites/:id/settings", r.dorkHandler.GetWebsiteSettings)
			protected.PUT("/dork/websites/:id/settings", r.dorkHandler.UpdateWebsiteSettings)
			protected.POST("/dork/websites/:id/scan", adminRateLimit, r.dorkHandler.TriggerScan)
			protected.GET("/dork/scans/:id", r.dorkHandler.GetScanResult)
			protected.POST("/dork/scan-all", adminRateLimit, r.dorkHandler.TriggerAllScans)
			protected.DELETE("/dork/detections", r.dorkHandler.ClearAllDetections)
			protected.POST("/dork/verify-ai", adminRateLimit, r.dorkHandler.VerifyAllWithAI)

			// Defacement Archive (Zone-H, Zone-XSEC)
			protected.GET("/defacement/stats", r.defacementHandler.GetStats)
			protected.GET("/defacement/incidents", r.defacementHandler.ListIncidents)
			protected.POST("/defacement/incidents/:id/acknowledge", r.defacementHandler.AcknowledgeIncident)
			protected.POST("/defacement/scan", adminRateLimit, r.defacementHandler.TriggerScan)

			// Vulnerability Scanner
			protected.GET("/vulnerability/stats", r.vulnerabilityHandler.GetVulnerabilityStats)
			protected.GET("/vulnerability/summary", r.vulnerabilityHandler.GetVulnerabilitySummary)
			protected.GET("/vulnerability/progress", r.vulnerabilityHandler.GetScanProgress)
			protected.GET("/vulnerability/websites/:id", r.vulnerabilityHandler.GetWebsiteVulnerability)
			protected.GET("/vulnerability/websites/:id/history", r.vulnerabilityHandler.GetWebsiteVulnerabilityHistory)
			protected.POST("/vulnerability/websites/:id/scan", adminRateLimit, r.vulnerabilityHandler.TriggerVulnerabilityScan)
			protected.POST("/vulnerability/scan-all", adminRateLimit, r.vulnerabilityHandler.TriggerAllVulnerabilityScans)

			// Users Management
			protected.GET("/users", r.userHandler.ListUsers)
			protected.POST("/users", r.userHandler.CreateUser)
			protected.PUT("/users/:id", r.userHandler.UpdateUser)
			protected.DELETE("/users/:id", r.userHandler.DeleteUser)
			protected.POST("/users/:id/reset-password", r.userHandler.ResetUserPassword)

			// Audit Logs
			protected.GET("/audit-logs", r.auditHandler.ListAuditLogs)

			// Settings
			protected.GET("/settings/notifications", r.settingsHandler.GetNotificationSettings)
			protected.PUT("/settings/notifications/telegram", r.settingsHandler.UpdateTelegramSettings)
			protected.PUT("/settings/notifications/email", r.settingsHandler.UpdateEmailSettings)
			protected.PUT("/settings/notifications/webhook", r.settingsHandler.UpdateWebhookSettings)
			protected.GET("/settings/notifications/digest", r.settingsHandler.GetDigestSettings)
			protected.PUT("/settings/notifications/digest", r.settingsHandler.UpdateDigestSettings)
			protected.GET("/settings/monitoring", r.settingsHandler.GetMonitoringSettings)
			protected.PUT("/settings/monitoring", r.settingsHandler.UpdateMonitoringSettings)
			protected.POST("/settings/test-telegram", r.settingsHandler.TestTelegram)
			protected.POST("/settings/test-email", r.settingsHandler.TestEmail)
			protected.POST("/settings/test-webhook", r.settingsHandler.TestWebhook)

			// AI Verification Settings
			protected.GET("/settings/ai", r.settingsHandler.GetAISettings)
			protected.PUT("/settings/ai", r.settingsHandler.UpdateAISettings)
		}
	}

	// Serve static files (legacy)
	engine.Static("/static", "./web/static")
}
