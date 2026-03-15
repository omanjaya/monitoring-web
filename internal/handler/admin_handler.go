package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/scheduler"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
)

type AdminHandler struct {
	scheduler        *scheduler.Scheduler
	telegramNotifier *notifier.TelegramNotifier
	emailNotifier    *notifier.EmailNotifier
	webhookNotifier  *notifier.WebhookNotifier
	websiteRepo      *mysql.WebsiteRepository
	settingsRepo     *mysql.SettingsRepository
}

func NewAdminHandler(
	scheduler *scheduler.Scheduler,
	telegramNotifier *notifier.TelegramNotifier,
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	websiteRepo *mysql.WebsiteRepository,
	settingsRepo *mysql.SettingsRepository,
) *AdminHandler {
	return &AdminHandler{
		scheduler:        scheduler,
		telegramNotifier: telegramNotifier,
		emailNotifier:    emailNotifier,
		webhookNotifier:  webhookNotifier,
		websiteRepo:      websiteRepo,
		settingsRepo:     settingsRepo,
	}
}

// TriggerCheck triggers a manual check
// @Summary Trigger manual check
// @Tags Admin
// @Security BearerAuth
// @Param type query string true "Check type (uptime, ssl, content, summary)"
// @Success 200 {object} map[string]string
// @Router /api/admin/trigger [post]
func (h *AdminHandler) TriggerCheck(c *gin.Context) {
	checkType := c.Query("type")
	if checkType == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Parameter type wajib diisi (uptime, ssl, content, summary)",
		})
		return
	}

	validTypes := map[string]bool{
		"uptime":        true,
		"ssl":           true,
		"content":       true,
		"summary":       true,
		"cleanup":       true,
		"escalation":    true,
		"dork":          true,
		"vulnerability": true,
		"dns":           true,
		"security":      true,
		"defacement":    true,
	}

	if !validTypes[checkType] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Type tidak valid. Gunakan: uptime, ssl, content, summary, dns, security, dork, vulnerability, defacement",
		})
		return
	}

	if err := h.scheduler.RunNow(checkType); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menjalankan check: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Check " + checkType + " sedang dijalankan",
	})
}

// GetSchedules returns the current active cron schedules
func (h *AdminHandler) GetSchedules(c *gin.Context) {
	schedules := h.scheduler.GetActiveSchedules()
	c.JSON(http.StatusOK, gin.H{"data": schedules})
}

// TestTelegram sends a test message to Telegram
// @Summary Test Telegram notification
// @Tags Admin
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /api/admin/test-telegram [post]
func (h *AdminHandler) TestTelegram(c *gin.Context) {
	if err := h.telegramNotifier.SendTestMessage(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengirim test message: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test message berhasil dikirim ke Telegram",
	})
}

// TestEmail sends a test email
// @Summary Test Email notification
// @Tags Admin
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /api/admin/test-email [post]
func (h *AdminHandler) TestEmail(c *gin.Context) {
	if h.emailNotifier == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email notifier tidak dikonfigurasi",
		})
		return
	}

	if err := h.emailNotifier.SendTestMessage(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengirim test email: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test email berhasil dikirim",
	})
}

// TestWebhook sends a test webhook
// @Summary Test Webhook notification
// @Tags Admin
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Router /api/admin/test-webhook [post]
func (h *AdminHandler) TestWebhook(c *gin.Context) {
	if h.webhookNotifier == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Webhook notifier tidak dikonfigurasi",
		})
		return
	}

	if err := h.webhookNotifier.SendTestMessage(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengirim test webhook: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Test webhook berhasil dikirim",
	})
}

// GetSystemStatus returns system status
// @Summary Get system status
// @Tags Admin
// @Security BearerAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/status [get]
func (h *AdminHandler) GetSystemStatus(c *gin.Context) {
	ctx := c.Request.Context()

	// Get total websites
	totalWebsites := 0
	if h.websiteRepo != nil {
		if count, err := h.websiteRepo.Count(ctx); err == nil {
			totalWebsites = int(count)
		}
	}

	// Get telegram enabled status from settings
	telegramEnabled := false
	if h.settingsRepo != nil {
		if enabled, err := h.settingsRepo.GetBool(ctx, "telegram.enabled"); err == nil {
			telegramEnabled = enabled
		}
	}

	// Get last check time
	var lastCheck *time.Time
	if h.websiteRepo != nil {
		if lc, err := h.websiteRepo.GetLastCheckTime(ctx); err == nil {
			lastCheck = lc
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"status":           "running",
			"scheduler":        "active",
			"version":          "1.0.0",
			"monitor_running":  true,
			"telegram_enabled": telegramEnabled,
			"total_websites":   totalWebsites,
			"last_check":       lastCheck,
		},
	})
}
