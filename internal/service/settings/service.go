package settings

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type Service struct {
	cfg          *config.Config
	settingsRepo *mysql.SettingsRepository
}

func NewService(cfg *config.Config, settingsRepo *mysql.SettingsRepository) *Service {
	return &Service{
		cfg:          cfg,
		settingsRepo: settingsRepo,
	}
}

// GetNotificationSettings retrieves notification settings from database
func (s *Service) GetNotificationSettings(ctx context.Context) (*domain.NotificationSettings, error) {
	return s.settingsRepo.GetNotificationSettings(ctx)
}

// GetMonitoringSettings retrieves monitoring settings from database
func (s *Service) GetMonitoringSettings(ctx context.Context) (*domain.MonitoringSettings, error) {
	return s.settingsRepo.GetMonitoringSettings(ctx)
}

// UpdateTelegramSettings updates Telegram notification settings
func (s *Service) UpdateTelegramSettings(ctx context.Context, update *domain.TelegramSettingsUpdate) error {
	return s.settingsRepo.SaveTelegramSettings(ctx, update)
}

// UpdateEmailSettings updates email notification settings
func (s *Service) UpdateEmailSettings(ctx context.Context, update *domain.EmailSettingsUpdate) error {
	return s.settingsRepo.SaveEmailSettings(ctx, update)
}

// UpdateWebhookSettings updates webhook notification settings
func (s *Service) UpdateWebhookSettings(ctx context.Context, update *domain.WebhookSettingsUpdate) error {
	return s.settingsRepo.SaveWebhookSettings(ctx, update)
}

// UpdateDigestSettings updates notification digest settings
func (s *Service) UpdateDigestSettings(ctx context.Context, update *domain.DigestSettingsUpdate) error {
	if update.Interval != nil && *update.Interval < 5 {
		return fmt.Errorf("digest interval harus minimal 5 menit")
	}
	return s.settingsRepo.SaveDigestSettings(ctx, update)
}

// UpdateMonitoringSettings updates monitoring settings
func (s *Service) UpdateMonitoringSettings(ctx context.Context, update *domain.MonitoringSettingsUpdate) error {
	// Validate settings
	if update.UptimeInterval != nil && *update.UptimeInterval < 1 {
		return fmt.Errorf("uptime interval harus minimal 1 menit")
	}
	if update.SSLInterval != nil && *update.SSLInterval < 60 {
		return fmt.Errorf("SSL interval harus minimal 60 menit")
	}
	if update.ContentInterval != nil && *update.ContentInterval < 5 {
		return fmt.Errorf("content interval harus minimal 5 menit")
	}
	if update.ResponseTimeThreshold != nil && *update.ResponseTimeThreshold < 100 {
		return fmt.Errorf("response time threshold harus minimal 100ms")
	}
	if update.DataRetentionDays != nil && *update.DataRetentionDays < 7 {
		return fmt.Errorf("data retention harus minimal 7 hari")
	}
	if update.DNSInterval != nil && *update.DNSInterval != 0 && *update.DNSInterval < 60 {
		return fmt.Errorf("DNS interval harus minimal 60 menit atau 0 untuk menonaktifkan")
	}

	return s.settingsRepo.SaveMonitoringSettings(ctx, update)
}

// GetAISettings retrieves AI verification settings from database
func (s *Service) GetAISettings(ctx context.Context) (*domain.AISettings, error) {
	return s.settingsRepo.GetAISettings(ctx)
}

// UpdateAISettings validates and saves AI verification settings
func (s *Service) UpdateAISettings(ctx context.Context, update *domain.AISettingsUpdate) error {
	if update.Provider != nil {
		valid := map[string]bool{"groq": true, "mistral": true, "anthropic": true}
		if !valid[*update.Provider] {
			return fmt.Errorf("provider tidak valid (pilih: groq, mistral, anthropic)")
		}
	}
	return s.settingsRepo.SaveAISettings(ctx, update)
}

// TestTelegramFromDB tests Telegram configuration using settings from database
func (s *Service) TestTelegramFromDB(ctx context.Context) error {
	settings, err := s.settingsRepo.GetNotificationSettings(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil settings: %w", err)
	}

	if !settings.Telegram.Enabled {
		return fmt.Errorf("Telegram notifications tidak diaktifkan")
	}

	if settings.Telegram.BotToken == "" {
		return fmt.Errorf("Bot token tidak dikonfigurasi")
	}

	if len(settings.Telegram.ChatIDs) == 0 {
		return fmt.Errorf("Tidak ada Chat ID yang dikonfigurasi")
	}

	message := `🧪 <b>TEST MESSAGE</b>

✅ Telegram notification berhasil dikonfigurasi!

<i>Monitoring Website - Diskominfos Bali</i>
<i>` + time.Now().Format("02 Jan 2006 15:04:05") + `</i>`

	// Send test message to all chat IDs
	httpClient := &http.Client{Timeout: 30 * time.Second}

	for _, chatID := range settings.Telegram.ChatIDs {
		url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", settings.Telegram.BotToken)

		payload := map[string]string{
			"chat_id":    chatID,
			"text":       message,
			"parse_mode": "HTML",
		}

		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("gagal membuat request: %w", err)
		}

		resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("gagal mengirim ke chat %s: %w", chatID, err)
		}
		defer resp.Body.Close()

		var result struct {
			OK          bool   `json:"ok"`
			Description string `json:"description,omitempty"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("gagal membaca response: %w", err)
		}

		if !result.OK {
			return fmt.Errorf("Telegram API error: %s", result.Description)
		}

		logger.Info().Str("chat_id", chatID).Msg("Test Telegram message sent successfully")
	}

	return nil
}

// TestEmailFromDB tests email configuration using settings from database
func (s *Service) TestEmailFromDB(ctx context.Context) error {
	settings, err := s.settingsRepo.GetNotificationSettings(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil settings: %w", err)
	}

	if !settings.Email.Enabled {
		return fmt.Errorf("Email notifications tidak diaktifkan")
	}

	if settings.Email.SMTPHost == "" {
		return fmt.Errorf("SMTP host tidak dikonfigurasi")
	}

	if len(settings.Email.Recipients) == 0 {
		return fmt.Errorf("Tidak ada recipients yang dikonfigurasi")
	}

	subject := "Test Email - Monitoring Website"
	body := `<html>
<body>
<h2>🧪 TEST EMAIL</h2>
<p>✅ Email notification berhasil dikonfigurasi!</p>
<hr>
<p><i>Monitoring Website - Diskominfos Bali</i></p>
<p><i>` + time.Now().Format("02 Jan 2006 15:04:05") + `</i></p>
</body>
</html>`

	// Build email headers
	fromName := settings.Email.FromName
	if fromName == "" {
		fromName = "Monitoring Website"
	}

	for _, recipient := range settings.Email.Recipients {
		headers := make(map[string]string)
		headers["From"] = fmt.Sprintf("%s <%s>", fromName, settings.Email.From)
		headers["To"] = recipient
		headers["Subject"] = subject
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/html; charset=\"utf-8\""

		var message strings.Builder
		for k, v := range headers {
			message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
		message.WriteString("\r\n")
		message.WriteString(body)

		auth := smtp.PlainAuth("", settings.Email.Username, settings.Email.Password, settings.Email.SMTPHost)
		addr := fmt.Sprintf("%s:%d", settings.Email.SMTPHost, settings.Email.SMTPPort)

		var sendErr error
		if settings.Email.UseTLS {
			sendErr = s.sendEmailTLS(addr, auth, settings.Email.From, recipient, message.String(), settings.Email.SMTPHost)
		} else {
			sendErr = smtp.SendMail(addr, auth, settings.Email.From, []string{recipient}, []byte(message.String()))
		}

		if sendErr != nil {
			return fmt.Errorf("gagal mengirim ke %s: %w", recipient, sendErr)
		}

		logger.Info().Str("recipient", recipient).Msg("Test email sent successfully")
	}

	return nil
}

func (s *Service) sendEmailTLS(addr string, auth smtp.Auth, from, to, message, host string) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: host,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return err
	}

	if err = client.Mail(from); err != nil {
		return err
	}

	if err = client.Rcpt(to); err != nil {
		return err
	}

	w, err := client.Data()
	if err != nil {
		return err
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return err
	}

	err = w.Close()
	if err != nil {
		return err
	}

	return client.Quit()
}

// TestWebhookFromDB tests webhook configuration using settings from database
func (s *Service) TestWebhookFromDB(ctx context.Context) error {
	settings, err := s.settingsRepo.GetNotificationSettings(ctx)
	if err != nil {
		return fmt.Errorf("gagal mengambil settings: %w", err)
	}

	if !settings.Webhook.Enabled {
		return fmt.Errorf("Webhook notifications tidak diaktifkan")
	}

	if len(settings.Webhook.URLs) == 0 {
		return fmt.Errorf("Tidak ada webhook URLs yang dikonfigurasi")
	}

	payload := map[string]interface{}{
		"type":    "test",
		"message": "Test webhook notification",
		"timestamp": time.Now().Format(time.RFC3339),
		"source":  "Monitoring Website - Diskominfos Bali",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("gagal membuat payload: %w", err)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}

	for _, webhookURL := range settings.Webhook.URLs {
		req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("gagal membuat request untuk %s: %w", webhookURL, err)
		}

		req.Header.Set("Content-Type", "application/json")
		if settings.Webhook.SecretKey != "" {
			req.Header.Set("X-Webhook-Secret", settings.Webhook.SecretKey)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("gagal mengirim ke %s: %w", webhookURL, err)
		}
		resp.Body.Close()

		if resp.StatusCode >= 400 {
			return fmt.Errorf("webhook %s mengembalikan status %d", webhookURL, resp.StatusCode)
		}

		logger.Info().Str("url", webhookURL).Msg("Test webhook sent successfully")
	}

	return nil
}

// GetAllSettings retrieves all settings
func (s *Service) GetAllSettings(ctx context.Context) ([]domain.Setting, error) {
	return s.settingsRepo.GetAll(ctx)
}

// GetSetting retrieves a specific setting by key
func (s *Service) GetSetting(ctx context.Context, key string) (*domain.Setting, error) {
	return s.settingsRepo.Get(ctx, key)
}

// SetSetting updates a specific setting
func (s *Service) SetSetting(ctx context.Context, key, value string, settingType domain.SettingType, description string) error {
	return s.settingsRepo.Set(ctx, key, value, settingType, description)
}
