package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type TelegramNotifier struct {
	cfg        *config.Config
	httpClient *http.Client
	alertRepo  *mysql.AlertRepository
}

func NewTelegramNotifier(cfg *config.Config, alertRepo *mysql.AlertRepository) *TelegramNotifier {
	return &TelegramNotifier{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		alertRepo: alertRepo,
	}
}

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

type telegramResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendAlert sends an alert notification via Telegram
func (t *TelegramNotifier) SendAlert(ctx context.Context, alert *domain.Alert, website *domain.Website) error {
	if !t.cfg.Telegram.Enabled {
		logger.Debug().Msg("Telegram notifications disabled")
		return nil
	}

	message := t.formatAlertMessage(alert, website)

	// Send to all configured chat IDs
	for _, chatID := range t.cfg.Telegram.ChatIDs {
		// Create notification record
		notification := &domain.Notification{
			AlertID:   alert.ID,
			Channel:   "telegram",
			Recipient: chatID,
			Status:    "pending",
		}

		notifID, err := t.alertRepo.CreateNotification(ctx, notification)
		if err != nil {
			logger.Error().Err(err).Str("chat_id", chatID).Msg("Failed to create notification record")
			continue
		}

		// Send message with retry
		err = retryWithBackoff("telegram", 3, func() error {
			return t.sendMessage(chatID, message)
		})
		if err != nil {
			logger.Error().Err(err).Str("chat_id", chatID).Msg("Failed to send Telegram message after retries")
			t.alertRepo.UpdateNotificationStatus(ctx, notifID, "failed", err.Error())
			continue
		}

		t.alertRepo.UpdateNotificationStatus(ctx, notifID, "sent", "")
		logger.Info().Str("chat_id", chatID).Int64("alert_id", alert.ID).Msg("Telegram notification sent")
	}

	return nil
}

func (t *TelegramNotifier) sendMessage(chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", t.cfg.Telegram.BotToken)

	msg := telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := t.httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var result telegramResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	return nil
}

func (t *TelegramNotifier) formatAlertMessage(alert *domain.Alert, website *domain.Website) string {
	var sb strings.Builder

	// Header with severity emoji
	switch alert.Severity {
	case domain.SeverityCritical:
		sb.WriteString("🚨 <b>CRITICAL ALERT</b> 🚨\n\n")
	case domain.SeverityWarning:
		sb.WriteString("⚠️ <b>WARNING</b> ⚠️\n\n")
	case domain.SeverityInfo:
		sb.WriteString("ℹ️ <b>INFO</b>\n\n")
	}

	// Alert type specific formatting
	switch alert.Type {
	case domain.AlertTypeDown:
		sb.WriteString("🔴 <b>WEBSITE DOWN</b>\n")
	case domain.AlertTypeUp:
		sb.WriteString("🟢 <b>WEBSITE UP</b>\n")
	case domain.AlertTypeSlowResponse:
		sb.WriteString("🟡 <b>SLOW RESPONSE</b>\n")
	case domain.AlertTypeSSLExpired:
		sb.WriteString("🔒 <b>SSL EXPIRED</b>\n")
	case domain.AlertTypeSSLExpiring:
		sb.WriteString("🔒 <b>SSL EXPIRING SOON</b>\n")
	case domain.AlertTypeJudolDetected:
		sb.WriteString("🎰 <b>JUDOL TERDETEKSI</b>\n")
	case domain.AlertTypeDefacement:
		sb.WriteString("💀 <b>DEFACEMENT DETECTED</b>\n")
	}

	sb.WriteString("\n")

	// Website info
	if website != nil {
		sb.WriteString(fmt.Sprintf("📌 <b>Website:</b> %s\n", website.Name))
		sb.WriteString(fmt.Sprintf("🔗 <b>URL:</b> %s\n", website.URL))
		if website.OPD != nil {
			sb.WriteString(fmt.Sprintf("🏢 <b>OPD:</b> %s\n", website.OPD.Name))
		}
	}

	sb.WriteString("\n")

	// Alert details
	sb.WriteString(fmt.Sprintf("📝 <b>%s</b>\n", alert.Title))
	sb.WriteString(fmt.Sprintf("%s\n\n", alert.Message))

	// Timestamp
	sb.WriteString(fmt.Sprintf("🕐 <i>%s</i>", alert.CreatedAt.Format("02 Jan 2006 15:04:05 WIB")))

	return sb.String()
}

// SendTestMessage sends a test message to verify Telegram configuration
func (t *TelegramNotifier) SendTestMessage(ctx context.Context) error {
	if !t.cfg.Telegram.Enabled {
		return fmt.Errorf("telegram notifications are disabled")
	}

	message := `🧪 <b>TEST MESSAGE</b>

✅ Telegram notification berhasil dikonfigurasi!

<i>Monitoring Website - Diskominfos Bali</i>`

	for _, chatID := range t.cfg.Telegram.ChatIDs {
		if err := t.sendMessage(chatID, message); err != nil {
			return fmt.Errorf("failed to send to chat %s: %w", chatID, err)
		}
	}

	return nil
}

// SendDailySummary sends daily monitoring summary
func (t *TelegramNotifier) SendDailySummary(ctx context.Context, summary *DailySummary) error {
	if !t.cfg.Telegram.Enabled {
		return nil
	}

	var sb strings.Builder

	sb.WriteString("📊 <b>LAPORAN HARIAN MONITORING</b>\n")
	sb.WriteString(fmt.Sprintf("📅 %s\n\n", time.Now().Format("02 January 2006")))

	sb.WriteString("<b>Status Website:</b>\n")
	sb.WriteString(fmt.Sprintf("🟢 UP: %d\n", summary.WebsitesUp))
	sb.WriteString(fmt.Sprintf("🔴 DOWN: %d\n", summary.WebsitesDown))
	sb.WriteString(fmt.Sprintf("🟡 Degraded: %d\n", summary.WebsitesDegraded))
	sb.WriteString(fmt.Sprintf("📊 Total: %d\n\n", summary.TotalWebsites))

	sb.WriteString("<b>Alert Summary:</b>\n")
	sb.WriteString(fmt.Sprintf("🚨 Critical: %d\n", summary.CriticalAlerts))
	sb.WriteString(fmt.Sprintf("⚠️ Warning: %d\n", summary.WarningAlerts))
	sb.WriteString(fmt.Sprintf("ℹ️ Info: %d\n\n", summary.InfoAlerts))

	if summary.JudolDetected > 0 {
		sb.WriteString(fmt.Sprintf("🎰 <b>JUDOL TERDETEKSI: %d website</b>\n\n", summary.JudolDetected))
	}

	sb.WriteString(fmt.Sprintf("📈 Avg Response Time: %dms\n", summary.AvgResponseTime))
	sb.WriteString(fmt.Sprintf("⬆️ Uptime: %.2f%%\n\n", summary.UptimePercentage))

	sb.WriteString("<i>Monitoring Website - Diskominfos Bali</i>")

	for _, chatID := range t.cfg.Telegram.ChatIDs {
		if err := t.sendMessage(chatID, sb.String()); err != nil {
			logger.Error().Err(err).Str("chat_id", chatID).Msg("Failed to send daily summary")
		}
	}

	return nil
}

// DailySummary represents daily monitoring statistics
type DailySummary struct {
	TotalWebsites    int
	WebsitesUp       int
	WebsitesDown     int
	WebsitesDegraded int
	CriticalAlerts   int
	WarningAlerts    int
	InfoAlerts       int
	JudolDetected    int
	AvgResponseTime  int
	UptimePercentage float64
}

// SendAlertResolved sends notification when an alert is resolved
func (t *TelegramNotifier) SendAlertResolved(ctx context.Context, alert *domain.Alert, website *domain.Website, note string) error {
	if !t.cfg.Telegram.Enabled {
		return nil
	}

	var sb strings.Builder

	sb.WriteString("✅ <b>ALERT RESOLVED</b>\n\n")

	// Alert type
	switch alert.Type {
	case domain.AlertTypeDown:
		sb.WriteString("🟢 Website kembali online\n")
	case domain.AlertTypeSSLExpired, domain.AlertTypeSSLExpiring:
		sb.WriteString("🔒 SSL certificate sudah diperbarui\n")
	case domain.AlertTypeJudolDetected:
		sb.WriteString("🧹 Konten sudah dibersihkan\n")
	default:
		sb.WriteString("✅ Issue sudah diselesaikan\n")
	}

	sb.WriteString("\n")

	// Website info
	if website != nil {
		sb.WriteString(fmt.Sprintf("📌 <b>Website:</b> %s\n", website.Name))
		sb.WriteString(fmt.Sprintf("🔗 <b>URL:</b> %s\n", website.URL))
	}

	sb.WriteString("\n")

	// Resolution note
	if note != "" {
		sb.WriteString(fmt.Sprintf("📝 <b>Note:</b> %s\n\n", note))
	}

	// Timestamp
	sb.WriteString(fmt.Sprintf("🕐 <i>%s</i>", time.Now().Format("02 Jan 2006 15:04:05 WIB")))

	for _, chatID := range t.cfg.Telegram.ChatIDs {
		if err := t.sendMessage(chatID, sb.String()); err != nil {
			logger.Error().Err(err).Str("chat_id", chatID).Msg("Failed to send resolved notification")
		}
	}

	return nil
}

// SendRawMessage sends a raw message to a specific chat ID (used for escalations)
func (t *TelegramNotifier) SendRawMessage(ctx context.Context, chatID, message string) error {
	if !t.cfg.Telegram.Enabled {
		return nil
	}
	return t.sendMessage(chatID, message)
}
