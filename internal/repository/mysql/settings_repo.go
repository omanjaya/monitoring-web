package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type SettingsRepository struct {
	db *sqlx.DB
}

func NewSettingsRepository(db *sqlx.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key
func (r *SettingsRepository) Get(ctx context.Context, key string) (*domain.Setting, error) {
	var setting domain.Setting
	query := "SELECT `key`, `value`, `type`, COALESCE(description, '') as description, updated_at FROM settings WHERE `key` = ?"

	err := r.db.GetContext(ctx, &setting, query, key)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &setting, nil
}

// GetAll retrieves all settings
func (r *SettingsRepository) GetAll(ctx context.Context) ([]domain.Setting, error) {
	var settings []domain.Setting
	query := "SELECT `key`, `value`, `type`, COALESCE(description, '') as description, updated_at FROM settings ORDER BY `key`"

	err := r.db.SelectContext(ctx, &settings, query)
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// GetByPrefix retrieves all settings with a specific key prefix
func (r *SettingsRepository) GetByPrefix(ctx context.Context, prefix string) ([]domain.Setting, error) {
	var settings []domain.Setting
	query := "SELECT `key`, `value`, `type`, COALESCE(description, '') as description, updated_at FROM settings WHERE `key` LIKE ? ORDER BY `key`"

	err := r.db.SelectContext(ctx, &settings, query, prefix+"%")
	if err != nil {
		return nil, err
	}

	return settings, nil
}

// Set creates or updates a setting
func (r *SettingsRepository) Set(ctx context.Context, key, value string, settingType domain.SettingType, description string) error {
	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	_, err := r.db.ExecContext(ctx, query, key, value, settingType, description)
	return err
}

// SetMultiple updates multiple settings at once
func (r *SettingsRepository) SetMultiple(ctx context.Context, settings map[string]string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := "UPDATE settings SET `value` = ?, updated_at = NOW() WHERE `key` = ?"

	for key, value := range settings {
		_, err := tx.ExecContext(ctx, query, value, key)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Delete removes a setting by key
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	query := "DELETE FROM settings WHERE `key` = ?"
	_, err := r.db.ExecContext(ctx, query, key)
	return err
}

// GetString retrieves a string setting value
func (r *SettingsRepository) GetString(ctx context.Context, key string) (string, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", nil
	}
	return setting.Value, nil
}

// GetBool retrieves a boolean setting value
func (r *SettingsRepository) GetBool(ctx context.Context, key string) (bool, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return false, err
	}
	if setting == nil {
		return false, nil
	}
	return setting.Value == "true" || setting.Value == "1", nil
}

// GetInt retrieves an integer setting value
func (r *SettingsRepository) GetInt(ctx context.Context, key string) (int, error) {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	if setting == nil {
		return 0, nil
	}
	val, _ := strconv.Atoi(setting.Value)
	return val, nil
}

// GetJSON retrieves a JSON setting value and unmarshals it
func (r *SettingsRepository) GetJSON(ctx context.Context, key string, dest interface{}) error {
	setting, err := r.Get(ctx, key)
	if err != nil {
		return err
	}
	if setting == nil || setting.Value == "" {
		return nil
	}
	return json.Unmarshal([]byte(setting.Value), dest)
}

// SetString sets a string setting value
func (r *SettingsRepository) SetString(ctx context.Context, key, value, description string) error {
	return r.Set(ctx, key, value, domain.SettingTypeString, description)
}

// SetBool sets a boolean setting value
func (r *SettingsRepository) SetBool(ctx context.Context, key string, value bool, description string) error {
	strVal := "false"
	if value {
		strVal = "true"
	}
	return r.Set(ctx, key, strVal, domain.SettingTypeBool, description)
}

// SetInt sets an integer setting value
func (r *SettingsRepository) SetInt(ctx context.Context, key string, value int, description string) error {
	return r.Set(ctx, key, strconv.Itoa(value), domain.SettingTypeInt, description)
}

// SetJSON sets a JSON setting value
func (r *SettingsRepository) SetJSON(ctx context.Context, key string, value interface{}, description string) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.Set(ctx, key, string(data), domain.SettingTypeJSON, description)
}

// GetNotificationSettings retrieves all notification settings
func (r *SettingsRepository) GetNotificationSettings(ctx context.Context) (*domain.NotificationSettings, error) {
	settings := &domain.NotificationSettings{}

	// Telegram settings
	settings.Telegram.Enabled, _ = r.GetBool(ctx, "telegram.enabled")
	settings.Telegram.BotToken, _ = r.GetString(ctx, "telegram.bot_token")
	r.GetJSON(ctx, "telegram.chat_ids", &settings.Telegram.ChatIDs)
	if settings.Telegram.ChatIDs == nil {
		settings.Telegram.ChatIDs = []string{}
	}

	// Email settings
	settings.Email.Enabled, _ = r.GetBool(ctx, "email.enabled")
	settings.Email.SMTPHost, _ = r.GetString(ctx, "email.smtp_host")
	settings.Email.SMTPPort, _ = r.GetInt(ctx, "email.smtp_port")
	if settings.Email.SMTPPort == 0 {
		settings.Email.SMTPPort = 587
	}
	settings.Email.Username, _ = r.GetString(ctx, "email.username")
	settings.Email.Password, _ = r.GetString(ctx, "email.password")
	settings.Email.From, _ = r.GetString(ctx, "email.from")
	settings.Email.FromName, _ = r.GetString(ctx, "email.from_name")
	settings.Email.UseTLS, _ = r.GetBool(ctx, "email.use_tls")
	r.GetJSON(ctx, "email.recipients", &settings.Email.Recipients)
	if settings.Email.Recipients == nil {
		settings.Email.Recipients = []string{}
	}

	// Webhook settings
	settings.Webhook.Enabled, _ = r.GetBool(ctx, "webhook.enabled")
	settings.Webhook.SecretKey, _ = r.GetString(ctx, "webhook.secret_key")
	r.GetJSON(ctx, "webhook.urls", &settings.Webhook.URLs)
	if settings.Webhook.URLs == nil {
		settings.Webhook.URLs = []string{}
	}

	// Digest settings
	settings.Digest.Enabled, _ = r.GetBool(ctx, "notification.digest_enabled")
	settings.Digest.Interval, _ = r.GetInt(ctx, "notification.digest_interval")
	if settings.Digest.Interval == 0 {
		settings.Digest.Interval = 15
	}
	settings.Digest.QuietHoursStart, _ = r.GetString(ctx, "notification.quiet_hours_start")
	settings.Digest.QuietHoursEnd, _ = r.GetString(ctx, "notification.quiet_hours_end")

	return settings, nil
}

// GetMonitoringSettings retrieves all monitoring settings
func (r *SettingsRepository) GetMonitoringSettings(ctx context.Context) (*domain.MonitoringSettings, error) {
	settings := &domain.MonitoringSettings{}

	settings.UptimeInterval, _ = r.GetInt(ctx, "monitoring.uptime_interval")
	if settings.UptimeInterval == 0 {
		settings.UptimeInterval = 5
	}

	settings.SSLInterval, _ = r.GetInt(ctx, "monitoring.ssl_interval")
	if settings.SSLInterval == 0 {
		settings.SSLInterval = 360 // 6 hours
	}

	settings.ContentInterval, _ = r.GetInt(ctx, "monitoring.content_interval")
	if settings.ContentInterval == 0 {
		settings.ContentInterval = 60
	}

	settings.SecurityInterval, _ = r.GetInt(ctx, "monitoring.security_interval")
	if settings.SecurityInterval == 0 {
		settings.SecurityInterval = 1440 // 24 hours
	}

	settings.VulnInterval, _ = r.GetInt(ctx, "monitoring.vuln_interval")
	if settings.VulnInterval == 0 {
		settings.VulnInterval = 1440
	}

	settings.DorkInterval, _ = r.GetInt(ctx, "monitoring.dork_interval")
	if settings.DorkInterval == 0 {
		settings.DorkInterval = 60
	}

	settings.DNSInterval, _ = r.GetInt(ctx, "monitoring.dns_interval")
	if settings.DNSInterval == 0 {
		settings.DNSInterval = 720 // 12 hours
	}

	settings.ResponseTimeThreshold, _ = r.GetInt(ctx, "monitoring.response_time_threshold")
	if settings.ResponseTimeThreshold == 0 {
		settings.ResponseTimeThreshold = 5000
	}

	settings.SSLExpiryWarningDays, _ = r.GetInt(ctx, "monitoring.ssl_expiry_warning_days")
	if settings.SSLExpiryWarningDays == 0 {
		settings.SSLExpiryWarningDays = 30
	}

	settings.DataRetentionDays, _ = r.GetInt(ctx, "monitoring.data_retention_days")
	if settings.DataRetentionDays == 0 {
		settings.DataRetentionDays = 90
	}

	return settings, nil
}

// SaveTelegramSettings saves Telegram notification settings
func (r *SettingsRepository) SaveTelegramSettings(ctx context.Context, settings *domain.TelegramSettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.Enabled != nil {
		val := "false"
		if *settings.Enabled {
			val = "true"
		}
		_, err := tx.ExecContext(ctx, query, "telegram.enabled", val, "bool", "Enable Telegram notifications")
		if err != nil {
			return err
		}
	}

	if settings.BotToken != nil {
		_, err := tx.ExecContext(ctx, query, "telegram.bot_token", *settings.BotToken, "string", "Telegram bot token")
		if err != nil {
			return err
		}
	}

	if settings.ChatIDs != nil {
		data, _ := json.Marshal(settings.ChatIDs)
		_, err := tx.ExecContext(ctx, query, "telegram.chat_ids", string(data), "json", "Telegram chat IDs")
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveEmailSettings saves email notification settings
func (r *SettingsRepository) SaveEmailSettings(ctx context.Context, settings *domain.EmailSettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.Enabled != nil {
		val := "false"
		if *settings.Enabled {
			val = "true"
		}
		if _, err := tx.ExecContext(ctx, query, "email.enabled", val, "bool", "Enable email notifications"); err != nil {
			return err
		}
	}

	if settings.SMTPHost != nil {
		if _, err := tx.ExecContext(ctx, query, "email.smtp_host", *settings.SMTPHost, "string", "SMTP server host"); err != nil {
			return err
		}
	}

	if settings.SMTPPort != nil {
		if _, err := tx.ExecContext(ctx, query, "email.smtp_port", strconv.Itoa(*settings.SMTPPort), "int", "SMTP server port"); err != nil {
			return err
		}
	}

	if settings.Username != nil {
		if _, err := tx.ExecContext(ctx, query, "email.username", *settings.Username, "string", "SMTP username"); err != nil {
			return err
		}
	}

	if settings.Password != nil {
		if _, err := tx.ExecContext(ctx, query, "email.password", *settings.Password, "string", "SMTP password"); err != nil {
			return err
		}
	}

	if settings.From != nil {
		if _, err := tx.ExecContext(ctx, query, "email.from", *settings.From, "string", "From email address"); err != nil {
			return err
		}
	}

	if settings.FromName != nil {
		if _, err := tx.ExecContext(ctx, query, "email.from_name", *settings.FromName, "string", "From name"); err != nil {
			return err
		}
	}

	if settings.UseTLS != nil {
		val := "false"
		if *settings.UseTLS {
			val = "true"
		}
		if _, err := tx.ExecContext(ctx, query, "email.use_tls", val, "bool", "Use TLS for SMTP"); err != nil {
			return err
		}
	}

	if settings.Recipients != nil {
		data, _ := json.Marshal(settings.Recipients)
		if _, err := tx.ExecContext(ctx, query, "email.recipients", string(data), "json", "Email recipients"); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveWebhookSettings saves webhook notification settings
func (r *SettingsRepository) SaveWebhookSettings(ctx context.Context, settings *domain.WebhookSettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.Enabled != nil {
		val := "false"
		if *settings.Enabled {
			val = "true"
		}
		if _, err := tx.ExecContext(ctx, query, "webhook.enabled", val, "bool", "Enable webhook notifications"); err != nil {
			return err
		}
	}

	if settings.SecretKey != nil {
		if _, err := tx.ExecContext(ctx, query, "webhook.secret_key", *settings.SecretKey, "string", "Webhook secret key"); err != nil {
			return err
		}
	}

	if settings.URLs != nil {
		data, _ := json.Marshal(settings.URLs)
		if _, err := tx.ExecContext(ctx, query, "webhook.urls", string(data), "json", "Webhook URLs"); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// SaveDigestSettings saves notification digest settings
func (r *SettingsRepository) SaveDigestSettings(ctx context.Context, settings *domain.DigestSettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.Enabled != nil {
		val := "false"
		if *settings.Enabled {
			val = "true"
		}
		tx.ExecContext(ctx, query, "notification.digest_enabled", val, "bool", "Enable notification digest")
	}

	if settings.Interval != nil {
		tx.ExecContext(ctx, query, "notification.digest_interval", strconv.Itoa(*settings.Interval), "int", "Digest interval (minutes)")
	}

	if settings.QuietHoursStart != nil {
		tx.ExecContext(ctx, query, "notification.quiet_hours_start", *settings.QuietHoursStart, "string", "Quiet hours start time")
	}

	if settings.QuietHoursEnd != nil {
		tx.ExecContext(ctx, query, "notification.quiet_hours_end", *settings.QuietHoursEnd, "string", "Quiet hours end time")
	}

	return tx.Commit()
}

// SaveMonitoringSettings saves monitoring configuration settings
func (r *SettingsRepository) SaveMonitoringSettings(ctx context.Context, settings *domain.MonitoringSettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.UptimeInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.uptime_interval", strconv.Itoa(*settings.UptimeInterval), "int", "Uptime check interval (minutes)")
	}

	if settings.SSLInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.ssl_interval", strconv.Itoa(*settings.SSLInterval), "int", "SSL check interval (minutes)")
	}

	if settings.ContentInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.content_interval", strconv.Itoa(*settings.ContentInterval), "int", "Content scan interval (minutes)")
	}

	if settings.SecurityInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.security_interval", strconv.Itoa(*settings.SecurityInterval), "int", "Security check interval (minutes)")
	}

	if settings.VulnInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.vuln_interval", strconv.Itoa(*settings.VulnInterval), "int", "Vulnerability scan interval (minutes)")
	}

	if settings.DorkInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.dork_interval", strconv.Itoa(*settings.DorkInterval), "int", "Dork scan interval (minutes)")
	}

	if settings.DNSInterval != nil {
		tx.ExecContext(ctx, query, "monitoring.dns_interval", strconv.Itoa(*settings.DNSInterval), "int", "DNS scan interval (minutes, 0=disabled)")
	}

	if settings.ResponseTimeThreshold != nil {
		tx.ExecContext(ctx, query, "monitoring.response_time_threshold", strconv.Itoa(*settings.ResponseTimeThreshold), "int", "Response time threshold (ms)")
	}

	if settings.SSLExpiryWarningDays != nil {
		tx.ExecContext(ctx, query, "monitoring.ssl_expiry_warning_days", strconv.Itoa(*settings.SSLExpiryWarningDays), "int", "SSL expiry warning days")
	}

	if settings.DataRetentionDays != nil {
		tx.ExecContext(ctx, query, "monitoring.data_retention_days", strconv.Itoa(*settings.DataRetentionDays), "int", "Data retention days")
	}

	return tx.Commit()
}

// GetAISettings retrieves AI verification settings
func (r *SettingsRepository) GetAISettings(ctx context.Context) (*domain.AISettings, error) {
	settings := &domain.AISettings{
		Provider: "groq",
		Model:    "",
	}

	if val, err := r.GetBool(ctx, "ai.enabled"); err == nil {
		settings.Enabled = val
	}
	if val, err := r.GetString(ctx, "ai.provider"); err == nil && val != "" {
		settings.Provider = val
	}
	if val, err := r.GetString(ctx, "ai.api_key"); err == nil {
		settings.APIKey = val
	}
	if val, err := r.GetString(ctx, "ai.model"); err == nil {
		settings.Model = val
	}

	return settings, nil
}

// SaveAISettings saves AI verification settings
func (r *SettingsRepository) SaveAISettings(ctx context.Context, settings *domain.AISettingsUpdate) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO settings (` + "`key`" + `, ` + "`value`" + `, ` + "`type`" + `, description, updated_at)
		VALUES (?, ?, ?, ?, NOW())
		ON DUPLICATE KEY UPDATE ` + "`value`" + ` = VALUES(` + "`value`" + `), updated_at = NOW()
	`

	if settings.Enabled != nil {
		val := "false"
		if *settings.Enabled {
			val = "true"
		}
		if _, err := tx.ExecContext(ctx, query, "ai.enabled", val, "bool", "Enable AI verification for dork detection"); err != nil {
			return err
		}
	}

	if settings.Provider != nil {
		if _, err := tx.ExecContext(ctx, query, "ai.provider", *settings.Provider, "string", "AI provider (groq, mistral, anthropic)"); err != nil {
			return err
		}
	}

	if settings.APIKey != nil {
		if _, err := tx.ExecContext(ctx, query, "ai.api_key", *settings.APIKey, "string", "AI provider API key"); err != nil {
			return err
		}
	}

	if settings.Model != nil {
		if _, err := tx.ExecContext(ctx, query, "ai.model", *settings.Model, "string", "AI model name"); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetLastUpdated returns the last update timestamp for settings
func (r *SettingsRepository) GetLastUpdated(ctx context.Context) (*time.Time, error) {
	var lastUpdated time.Time
	query := "SELECT MAX(updated_at) FROM settings"

	err := r.db.GetContext(ctx, &lastUpdated, query)
	if err != nil {
		return nil, err
	}

	return &lastUpdated, nil
}
