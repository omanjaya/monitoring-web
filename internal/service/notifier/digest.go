package notifier

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

// DigestEntry represents a single alert entry in the digest queue.
type DigestEntry struct {
	Alert   *domain.Alert
	Website *domain.Website
}

// DigestService batches multiple alert notifications into periodic digest messages
// instead of sending one notification per alert.
type DigestService struct {
	cfg              *config.Config
	mu               sync.Mutex
	pending          []DigestEntry
	telegramNotifier *TelegramNotifier
	emailNotifier    *EmailNotifier
	webhookNotifier  *WebhookNotifier
	ticker           *time.Ticker
	done             chan struct{}
}

// NewDigestService creates a new DigestService.
func NewDigestService(
	cfg *config.Config,
	telegramNotifier *TelegramNotifier,
	emailNotifier *EmailNotifier,
	webhookNotifier *WebhookNotifier,
) *DigestService {
	return &DigestService{
		cfg:              cfg,
		pending:          make([]DigestEntry, 0),
		telegramNotifier: telegramNotifier,
		emailNotifier:    emailNotifier,
		webhookNotifier:  webhookNotifier,
		done:             make(chan struct{}),
	}
}

// Start begins the digest ticker that periodically flushes queued alerts.
func (ds *DigestService) Start() {
	interval := ds.cfg.Notification.DigestInterval
	if interval <= 0 {
		interval = 15
	}
	ds.ticker = time.NewTicker(time.Duration(interval) * time.Minute)

	go func() {
		for {
			select {
			case <-ds.ticker.C:
				ds.Flush(context.Background())
			case <-ds.done:
				return
			}
		}
	}()

	logger.Info().Int("interval_minutes", interval).Msg("Notification digest service started")
}

// Stop stops the digest ticker and flushes remaining alerts.
func (ds *DigestService) Stop() {
	if ds.ticker != nil {
		ds.ticker.Stop()
	}
	close(ds.done)
	// Flush remaining alerts before shutting down
	ds.Flush(context.Background())
}

// Add adds an alert to the digest queue. Critical alerts bypass the digest
// and are sent immediately.
func (ds *DigestService) Add(alert *domain.Alert, website *domain.Website) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Critical alerts bypass digest and send immediately
	if alert.Severity == domain.SeverityCritical {
		go ds.sendImmediate(context.Background(), alert, website)
		return
	}

	ds.pending = append(ds.pending, DigestEntry{Alert: alert, Website: website})
	logger.Debug().Int64("alert_id", alert.ID).Msg("Alert added to digest queue")
}

// Flush sends all pending alerts as a single digest notification.
func (ds *DigestService) Flush(ctx context.Context) {
	ds.mu.Lock()
	if len(ds.pending) == 0 {
		ds.mu.Unlock()
		return
	}
	entries := make([]DigestEntry, len(ds.pending))
	copy(entries, ds.pending)
	ds.pending = ds.pending[:0]
	ds.mu.Unlock()

	// Check quiet hours — re-queue if within quiet window
	if ds.isQuietHours() {
		logger.Info().Int("count", len(entries)).Msg("Digest skipped during quiet hours, re-queuing")
		ds.mu.Lock()
		ds.pending = append(entries, ds.pending...)
		ds.mu.Unlock()
		return
	}

	logger.Info().Int("count", len(entries)).Msg("Sending notification digest")
	ds.sendDigest(ctx, entries)
}

func (ds *DigestService) isQuietHours() bool {
	start := ds.cfg.Notification.QuietHoursStart
	end := ds.cfg.Notification.QuietHoursEnd
	if start == "" || end == "" {
		return false
	}

	now := time.Now()
	startTime, err1 := time.Parse("15:04", start)
	endTime, err2 := time.Parse("15:04", end)
	if err1 != nil || err2 != nil {
		return false
	}

	currentMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startTime.Hour()*60 + startTime.Minute()
	endMinutes := endTime.Hour()*60 + endTime.Minute()

	if startMinutes > endMinutes {
		// Overnight quiet hours (e.g., 22:00 - 07:00)
		return currentMinutes >= startMinutes || currentMinutes < endMinutes
	}
	return currentMinutes >= startMinutes && currentMinutes < endMinutes
}

func (ds *DigestService) sendImmediate(ctx context.Context, alert *domain.Alert, website *domain.Website) {
	if ds.cfg.Telegram.Enabled {
		if err := ds.telegramNotifier.SendAlert(ctx, alert, website); err != nil {
			logger.Error().Err(err).Msg("Failed to send immediate telegram alert")
		}
	}
	if ds.cfg.Email.Enabled && ds.emailNotifier != nil {
		if err := ds.emailNotifier.SendAlert(ctx, alert, website); err != nil {
			logger.Error().Err(err).Msg("Failed to send immediate email alert")
		}
	}
	if ds.cfg.Webhook.Enabled && ds.webhookNotifier != nil {
		if err := ds.webhookNotifier.SendAlert(ctx, alert, website); err != nil {
			logger.Error().Err(err).Msg("Failed to send immediate webhook alert")
		}
	}
}

func (ds *DigestService) sendDigest(ctx context.Context, entries []DigestEntry) {
	if ds.cfg.Telegram.Enabled {
		ds.sendTelegramDigest(ctx, entries)
	}
	if ds.cfg.Email.Enabled && ds.emailNotifier != nil {
		ds.sendEmailDigest(ctx, entries)
	}
	if ds.cfg.Webhook.Enabled && ds.webhookNotifier != nil {
		ds.sendWebhookDigest(ctx, entries)
	}
}

// --- Telegram digest ---

func (ds *DigestService) sendTelegramDigest(ctx context.Context, entries []DigestEntry) {
	// Group by severity
	var critical, warning, info []DigestEntry
	for _, e := range entries {
		switch e.Alert.Severity {
		case domain.SeverityCritical:
			critical = append(critical, e)
		case domain.SeverityWarning:
			warning = append(warning, e)
		default:
			info = append(info, e)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📋 <b>Alert Digest</b>\n📊 <b>%d</b> alert baru\n\n", len(entries)))

	if len(critical) > 0 {
		sb.WriteString(fmt.Sprintf("🔴 <b>Critical (%d)</b>\n", len(critical)))
		for _, e := range critical {
			name := "Unknown"
			if e.Website != nil {
				name = e.Website.Name
			}
			sb.WriteString(fmt.Sprintf("  • %s — %s\n", name, e.Alert.Title))
		}
		sb.WriteString("\n")
	}
	if len(warning) > 0 {
		sb.WriteString(fmt.Sprintf("🟡 <b>Warning (%d)</b>\n", len(warning)))
		for _, e := range warning {
			name := "Unknown"
			if e.Website != nil {
				name = e.Website.Name
			}
			sb.WriteString(fmt.Sprintf("  • %s — %s\n", name, e.Alert.Title))
		}
		sb.WriteString("\n")
	}
	if len(info) > 0 {
		sb.WriteString(fmt.Sprintf("ℹ️ <b>Info (%d)</b>\n", len(info)))
		for _, e := range info {
			name := "Unknown"
			if e.Website != nil {
				name = e.Website.Name
			}
			sb.WriteString(fmt.Sprintf("  • %s — %s\n", name, e.Alert.Title))
		}
	}

	sb.WriteString(fmt.Sprintf("\n🕐 %s", time.Now().Format("02 Jan 2006 15:04 WIB")))

	text := sb.String()
	for _, chatID := range ds.cfg.Telegram.ChatIDs {
		if err := ds.telegramNotifier.sendMessage(chatID, text); err != nil {
			logger.Error().Err(err).Str("chat_id", chatID).Msg("Failed to send telegram digest")
		}
	}
}

// --- Email digest ---

func (ds *DigestService) sendEmailDigest(ctx context.Context, entries []DigestEntry) {
	// Group by severity
	var critical, warning, info []DigestEntry
	for _, e := range entries {
		switch e.Alert.Severity {
		case domain.SeverityCritical:
			critical = append(critical, e)
		case domain.SeverityWarning:
			warning = append(warning, e)
		default:
			info = append(info, e)
		}
	}

	subject := fmt.Sprintf("[DIGEST] %d Alert Baru - %s", len(entries), time.Now().Format("02 Jan 2006 15:04"))

	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #007bff; color: white; padding: 20px; text-align: center; border-radius: 5px 5px 0 0; }
        .content { background-color: #f8f9fa; padding: 20px; border: 1px solid #dee2e6; }
        .section-title { margin-top: 15px; font-weight: bold; padding: 8px; border-radius: 3px; }
        .section-critical { background-color: #dc3545; color: white; }
        .section-warning { background-color: #ffc107; color: #333; }
        .section-info { background-color: #17a2b8; color: white; }
        table { width: 100%; border-collapse: collapse; margin-top: 10px; margin-bottom: 15px; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #dee2e6; font-size: 13px; }
        th { background-color: #e9ecef; }
        .footer { padding: 15px; text-align: center; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>ALERT DIGEST</h1>
            <p>`)
	sb.WriteString(fmt.Sprintf("%d alert baru — %s", len(entries), time.Now().Format("02 January 2006 15:04 WIB")))
	sb.WriteString(`</p>
        </div>
        <div class="content">
`)

	writeSection := func(title, cssClass string, items []DigestEntry) {
		if len(items) == 0 {
			return
		}
		sb.WriteString(fmt.Sprintf(`            <div class="section-title %s">%s (%d)</div>
            <table>
                <tr><th>Website</th><th>Alert</th><th>Waktu</th></tr>
`, cssClass, title, len(items)))
		for _, e := range items {
			name := "Unknown"
			if e.Website != nil {
				name = e.Website.Name
			}
			sb.WriteString(fmt.Sprintf("                <tr><td>%s</td><td>%s</td><td>%s</td></tr>\n",
				name, e.Alert.Title, e.Alert.CreatedAt.Format("15:04:05")))
		}
		sb.WriteString("            </table>\n")
	}

	writeSection("Critical", "section-critical", critical)
	writeSection("Warning", "section-warning", warning)
	writeSection("Info", "section-info", info)

	sb.WriteString(`        </div>
        <div class="footer">
            Monitoring Website - Diskominfos Provinsi Bali<br>
            Email digest ini dikirim secara otomatis.
        </div>
    </div>
</body>
</html>`)

	for _, recipient := range ds.cfg.Email.Recipients {
		if err := ds.emailNotifier.sendEmail(recipient, subject, sb.String()); err != nil {
			logger.Error().Err(err).Str("recipient", recipient).Msg("Failed to send email digest")
		}
	}
}

// --- Webhook digest ---

// DigestAlertPayload is a single alert entry in the webhook digest payload.
type DigestAlertPayload struct {
	AlertID   int64  `json:"alert_id"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
	Website   string `json:"website"`
	URL       string `json:"url"`
}

func (ds *DigestService) sendWebhookDigest(ctx context.Context, entries []DigestEntry) {
	alerts := make([]DigestAlertPayload, 0, len(entries))
	for _, e := range entries {
		websiteName := ""
		websiteURL := ""
		if e.Website != nil {
			websiteName = e.Website.Name
			websiteURL = e.Website.URL
		}
		alerts = append(alerts, DigestAlertPayload{
			AlertID:   e.Alert.ID,
			Type:      string(e.Alert.Type),
			Severity:  string(e.Alert.Severity),
			Title:     e.Alert.Title,
			Message:   e.Alert.Message,
			CreatedAt: e.Alert.CreatedAt.Format(time.RFC3339),
			Website:   websiteName,
			URL:       websiteURL,
		})
	}

	payload := map[string]interface{}{
		"event":     "alert.digest",
		"timestamp": time.Now().Format(time.RFC3339),
		"count":     len(entries),
		"alerts":    alerts,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to marshal webhook digest payload")
		return
	}

	for _, webhookURL := range ds.cfg.Webhook.URLs {
		if err := ds.webhookNotifier.sendRawWebhook(webhookURL, body); err != nil {
			logger.Error().Err(err).Str("url", webhookURL).Msg("Failed to send webhook digest")
		}
	}
}
