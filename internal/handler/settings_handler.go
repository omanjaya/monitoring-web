package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/scheduler"
	"github.com/diskominfos-bali/monitoring-website/internal/service/settings"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type SettingsHandler struct {
	settingsService *settings.Service
	scheduler       *scheduler.Scheduler
}

func NewSettingsHandler(settingsService *settings.Service, sched *scheduler.Scheduler) *SettingsHandler {
	return &SettingsHandler{
		settingsService: settingsService,
		scheduler:       sched,
	}
}

// GetNotificationSettings retrieves notification settings
// @Summary Get notification settings
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.NotificationSettings
// @Router /api/settings/notifications [get]
func (h *SettingsHandler) GetNotificationSettings(c *gin.Context) {
	settings, err := h.settingsService.GetNotificationSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil settings notifikasi",
		})
		return
	}

	// Mask sensitive data for display
	maskedSettings := *settings
	if maskedSettings.Telegram.BotToken != "" {
		maskedSettings.Telegram.BotToken = maskToken(maskedSettings.Telegram.BotToken)
	}
	if maskedSettings.Email.Password != "" {
		maskedSettings.Email.Password = "********"
	}
	if maskedSettings.Webhook.SecretKey != "" {
		maskedSettings.Webhook.SecretKey = maskToken(maskedSettings.Webhook.SecretKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": maskedSettings,
	})
}

// UpdateTelegramSettings updates Telegram notification settings
// @Summary Update Telegram settings
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.TelegramSettingsUpdate true "Telegram settings"
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/notifications/telegram [put]
func (h *SettingsHandler) UpdateTelegramSettings(c *gin.Context) {
	var input domain.TelegramSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateTelegramSettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan settings Telegram",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings Telegram berhasil disimpan",
	})
}

// UpdateEmailSettings updates email notification settings
// @Summary Update email settings
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.EmailSettingsUpdate true "Email settings"
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/notifications/email [put]
func (h *SettingsHandler) UpdateEmailSettings(c *gin.Context) {
	var input domain.EmailSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateEmailSettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan settings email",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings email berhasil disimpan",
	})
}

// UpdateWebhookSettings updates webhook notification settings
// @Summary Update webhook settings
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.WebhookSettingsUpdate true "Webhook settings"
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/notifications/webhook [put]
func (h *SettingsHandler) UpdateWebhookSettings(c *gin.Context) {
	var input domain.WebhookSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateWebhookSettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyimpan settings webhook",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings webhook berhasil disimpan",
	})
}

// GetDigestSettings retrieves notification digest settings
// @Summary Get digest settings
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/notifications/digest [get]
func (h *SettingsHandler) GetDigestSettings(c *gin.Context) {
	settings, err := h.settingsService.GetNotificationSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil settings digest",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings.Digest,
	})
}

// UpdateDigestSettings updates notification digest settings
// @Summary Update digest settings
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.DigestSettingsUpdate true "Digest settings"
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/notifications/digest [put]
func (h *SettingsHandler) UpdateDigestSettings(c *gin.Context) {
	var input domain.DigestSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateDigestSettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pengaturan digest berhasil diperbarui",
	})
}

// GetMonitoringSettings retrieves monitoring settings
// @Summary Get monitoring settings
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.MonitoringSettings
// @Router /api/settings/monitoring [get]
func (h *SettingsHandler) GetMonitoringSettings(c *gin.Context) {
	settings, err := h.settingsService.GetMonitoringSettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil settings monitoring",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": settings,
	})
}

// UpdateMonitoringSettings updates monitoring settings
// @Summary Update monitoring settings
// @Tags Settings
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.MonitoringSettingsUpdate true "Monitoring settings"
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/monitoring [put]
func (h *SettingsHandler) UpdateMonitoringSettings(c *gin.Context) {
	var input domain.MonitoringSettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateMonitoringSettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Reload scheduler with new intervals
	if h.scheduler != nil {
		if err := h.scheduler.ReloadSchedules(); err != nil {
			logger.Error().Err(err).Msg("Failed to reload scheduler after settings update")
		} else {
			logger.Info().Msg("Scheduler reloaded with new monitoring settings")
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings monitoring berhasil disimpan dan scheduler di-reload",
	})
}

// TestTelegram tests Telegram configuration from database settings
// @Summary Test Telegram notification
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/test-telegram [post]
func (h *SettingsHandler) TestTelegram(c *gin.Context) {
	err := h.settingsService.TestTelegramFromDB(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Pesan test Telegram berhasil dikirim",
		"success": true,
	})
}

// TestEmail tests email configuration from database settings
// @Summary Test email notification
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/test-email [post]
func (h *SettingsHandler) TestEmail(c *gin.Context) {
	err := h.settingsService.TestEmailFromDB(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Email test berhasil dikirim",
		"success": true,
	})
}

// TestWebhook tests webhook configuration from database settings
// @Summary Test webhook notification
// @Tags Settings
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/settings/test-webhook [post]
func (h *SettingsHandler) TestWebhook(c *gin.Context) {
	err := h.settingsService.TestWebhookFromDB(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   err.Error(),
			"success": false,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Webhook test berhasil dikirim",
		"success": true,
	})
}

// GetAISettings retrieves AI verification settings
func (h *SettingsHandler) GetAISettings(c *gin.Context) {
	settings, err := h.settingsService.GetAISettings(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil settings AI",
		})
		return
	}

	// Mask API key
	masked := *settings
	if masked.APIKey != "" {
		masked.APIKey = maskToken(masked.APIKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": masked,
	})
}

// UpdateAISettings updates AI verification settings
func (h *SettingsHandler) UpdateAISettings(c *gin.Context) {
	var input domain.AISettingsUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid",
		})
		return
	}

	if err := h.settingsService.UpdateAISettings(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Settings AI berhasil disimpan",
	})
}

// maskToken masks a token for display, showing only first and last 4 characters
func maskToken(token string) string {
	if len(token) <= 8 {
		return "****"
	}
	return token[:4] + "****" + token[len(token)-4:]
}
