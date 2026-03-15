package notifier

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type WebhookNotifier struct {
	cfg        *config.Config
	httpClient *http.Client
	alertRepo  *mysql.AlertRepository
}

func NewWebhookNotifier(cfg *config.Config, alertRepo *mysql.AlertRepository) *WebhookNotifier {
	timeout := cfg.Webhook.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	return &WebhookNotifier{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		alertRepo: alertRepo,
	}
}

// WebhookPayload represents the payload sent to webhook endpoints
type WebhookPayload struct {
	Event      string             `json:"event"`
	Timestamp  time.Time          `json:"timestamp"`
	Alert      *AlertPayload      `json:"alert,omitempty"`
	Website    *WebsitePayload    `json:"website,omitempty"`
	Summary    *SummaryPayload    `json:"summary,omitempty"`
	Escalation *EscalationPayload `json:"escalation,omitempty"`
}

type AlertPayload struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Severity  string `json:"severity"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	CreatedAt string `json:"created_at"`
}

type WebsitePayload struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	ResponseTime int    `json:"response_time,omitempty"`
	OPDName      string `json:"opd_name,omitempty"`
}

type SummaryPayload struct {
	Date             string  `json:"date"`
	TotalWebsites    int     `json:"total_websites"`
	WebsitesUp       int     `json:"websites_up"`
	WebsitesDown     int     `json:"websites_down"`
	WebsitesDegraded int     `json:"websites_degraded"`
	CriticalAlerts   int     `json:"critical_alerts"`
	WarningAlerts    int     `json:"warning_alerts"`
	InfoAlerts       int     `json:"info_alerts"`
	AvgResponseTime  int     `json:"avg_response_time"`
	UptimePercentage float64 `json:"uptime_percentage"`
	JudolDetected    int     `json:"judol_detected"`
}

type EscalationPayload struct {
	Level           int `json:"level"`
	DelayMinutes    int `json:"delay_minutes"`
	EscalationCount int `json:"escalation_count"`
}

// SendAlert sends an alert notification via webhook
func (w *WebhookNotifier) SendAlert(ctx context.Context, alert *domain.Alert, website *domain.Website) error {
	if !w.cfg.Webhook.Enabled {
		logger.Debug().Msg("Webhook notifications disabled")
		return nil
	}

	payload := WebhookPayload{
		Event:     "alert.created",
		Timestamp: time.Now(),
		Alert: &AlertPayload{
			ID:        alert.ID,
			Type:      string(alert.Type),
			Severity:  string(alert.Severity),
			Title:     alert.Title,
			Message:   alert.Message,
			CreatedAt: alert.CreatedAt.Format(time.RFC3339),
		},
	}

	if website != nil {
		responseTime := 0
		if website.LastResponseTime.Valid {
			responseTime = int(website.LastResponseTime.Int32)
		}
		opdName := ""
		if website.OPD != nil {
			opdName = website.OPD.Name
		}
		payload.Website = &WebsitePayload{
			ID:           website.ID,
			Name:         website.Name,
			URL:          website.URL,
			Status:       string(website.Status),
			ResponseTime: responseTime,
			OPDName:      opdName,
		}
	}

	// Send to all configured webhook URLs
	for _, webhookURL := range w.cfg.Webhook.URLs {
		// Create notification record
		notification := &domain.Notification{
			AlertID:   alert.ID,
			Channel:   "webhook",
			Recipient: webhookURL,
			Status:    "pending",
		}

		notifID, err := w.alertRepo.CreateNotification(ctx, notification)
		if err != nil {
			logger.Error().Err(err).Str("url", webhookURL).Msg("Failed to create webhook notification record")
			continue
		}

		// Send webhook with retry
		err = retryWithBackoff("webhook", 3, func() error {
			return w.sendWebhook(webhookURL, payload)
		})
		if err != nil {
			logger.Error().Err(err).Str("url", webhookURL).Msg("Failed to send webhook after retries")
			w.alertRepo.UpdateNotificationStatus(ctx, notifID, "failed", err.Error())
			continue
		}

		w.alertRepo.UpdateNotificationStatus(ctx, notifID, "sent", "")
		logger.Info().Str("url", webhookURL).Int64("alert_id", alert.ID).Msg("Webhook notification sent")
	}

	return nil
}

func (w *WebhookNotifier) sendWebhook(url string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MonitoringWebsite-Diskominfos-Bali/1.0")
	req.Header.Set("X-Webhook-Event", payload.Event)

	// Add signature if secret key is configured
	if w.cfg.Webhook.SecretKey != "" {
		signature := w.computeSignature(body)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func (w *WebhookNotifier) computeSignature(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(w.cfg.Webhook.SecretKey))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// sendRawWebhook sends a pre-marshaled JSON payload to a webhook URL.
func (w *WebhookNotifier) sendRawWebhook(url string, body []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "MonitoringWebsite-Diskominfos-Bali/1.0")
	req.Header.Set("X-Webhook-Event", "alert.digest")

	// Add signature if secret key is configured
	if w.cfg.Webhook.SecretKey != "" {
		signature := w.computeSignature(body)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// SendTestMessage sends a test webhook to verify configuration
func (w *WebhookNotifier) SendTestMessage(ctx context.Context) error {
	if !w.cfg.Webhook.Enabled {
		return fmt.Errorf("webhook notifications are disabled")
	}

	payload := WebhookPayload{
		Event:     "test",
		Timestamp: time.Now(),
		Alert: &AlertPayload{
			ID:        0,
			Type:      "test",
			Severity:  "info",
			Title:     "Test Webhook",
			Message:   "Webhook notification berhasil dikonfigurasi!",
			CreatedAt: time.Now().Format(time.RFC3339),
		},
	}

	for _, webhookURL := range w.cfg.Webhook.URLs {
		if err := w.sendWebhook(webhookURL, payload); err != nil {
			return fmt.Errorf("failed to send to %s: %w", webhookURL, err)
		}
	}

	return nil
}

// SendDailySummary sends daily monitoring summary via webhook
func (w *WebhookNotifier) SendDailySummary(ctx context.Context, summary *DailySummary) error {
	if !w.cfg.Webhook.Enabled {
		return nil
	}

	payload := WebhookPayload{
		Event:     "daily_summary",
		Timestamp: time.Now(),
		Summary: &SummaryPayload{
			Date:             time.Now().Format("2006-01-02"),
			TotalWebsites:    summary.TotalWebsites,
			WebsitesUp:       summary.WebsitesUp,
			WebsitesDown:     summary.WebsitesDown,
			WebsitesDegraded: summary.WebsitesDegraded,
			CriticalAlerts:   summary.CriticalAlerts,
			WarningAlerts:    summary.WarningAlerts,
			InfoAlerts:       summary.InfoAlerts,
			AvgResponseTime:  summary.AvgResponseTime,
			UptimePercentage: summary.UptimePercentage,
			JudolDetected:    summary.JudolDetected,
		},
	}

	for _, webhookURL := range w.cfg.Webhook.URLs {
		if err := w.sendWebhook(webhookURL, payload); err != nil {
			logger.Error().Err(err).Str("url", webhookURL).Msg("Failed to send daily summary webhook")
		}
	}

	return nil
}

// SendAlertResolved sends notification when an alert is resolved
func (w *WebhookNotifier) SendAlertResolved(ctx context.Context, alert *domain.Alert, website *domain.Website, note string) error {
	if !w.cfg.Webhook.Enabled {
		return nil
	}

	payload := WebhookPayload{
		Event:     "alert.resolved",
		Timestamp: time.Now(),
		Alert: &AlertPayload{
			ID:        alert.ID,
			Type:      string(alert.Type),
			Severity:  string(alert.Severity),
			Title:     alert.Title,
			Message:   note,
			CreatedAt: alert.CreatedAt.Format(time.RFC3339),
		},
	}

	if website != nil {
		responseTime := 0
		if website.LastResponseTime.Valid {
			responseTime = int(website.LastResponseTime.Int32)
		}
		opdName := ""
		if website.OPD != nil {
			opdName = website.OPD.Name
		}
		payload.Website = &WebsitePayload{
			ID:           website.ID,
			Name:         website.Name,
			URL:          website.URL,
			Status:       string(website.Status),
			ResponseTime: responseTime,
			OPDName:      opdName,
		}
	}

	for _, webhookURL := range w.cfg.Webhook.URLs {
		if err := w.sendWebhook(webhookURL, payload); err != nil {
			logger.Error().Err(err).Str("url", webhookURL).Msg("Failed to send resolved webhook")
		}
	}

	return nil
}

// SendEscalation sends escalation data to a specific webhook URL
func (w *WebhookNotifier) SendEscalation(ctx context.Context, url string, alert *domain.AlertWithEscalation, rule *domain.EscalationRule) error {
	if !w.cfg.Webhook.Enabled {
		return nil
	}

	payload := WebhookPayload{
		Event:     "alert.escalated",
		Timestamp: time.Now(),
		Alert: &AlertPayload{
			ID:        alert.ID,
			Type:      string(alert.Type),
			Severity:  string(alert.Severity),
			Title:     alert.Title,
			Message:   alert.Message,
			CreatedAt: alert.CreatedAt.Format(time.RFC3339),
		},
		Escalation: &EscalationPayload{
			Level:           int(rule.Level),
			DelayMinutes:    rule.DelayMinutes,
			EscalationCount: alert.EscalationCount + 1,
		},
	}

	return w.sendWebhook(url, payload)
}
