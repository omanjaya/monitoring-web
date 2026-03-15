package domain

import (
	"time"
)

// SettingType represents the type of a setting value
type SettingType string

const (
	SettingTypeString SettingType = "string"
	SettingTypeInt    SettingType = "int"
	SettingTypeBool   SettingType = "bool"
	SettingTypeJSON   SettingType = "json"
)

// Setting represents a single configuration setting
type Setting struct {
	Key         string      `db:"key" json:"key"`
	Value       string      `db:"value" json:"value"`
	Type        SettingType `db:"type" json:"type"`
	Description string      `db:"description" json:"description,omitempty"`
	UpdatedAt   time.Time   `db:"updated_at" json:"updated_at"`
}

// SettingUpdate represents the input for updating a setting
type SettingUpdate struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

// NotificationSettings represents all notification configuration
type NotificationSettings struct {
	Telegram TelegramSettings `json:"telegram"`
	Email    EmailSettings    `json:"email"`
	Webhook  WebhookSettings  `json:"webhook"`
	Digest   DigestSettings   `json:"digest"`
}

// DigestSettings represents notification digest/batching configuration
type DigestSettings struct {
	Enabled        bool   `json:"digest_enabled"`
	Interval       int    `json:"digest_interval"`    // minutes
	QuietHoursStart string `json:"quiet_hours_start"` // e.g. "22:00"
	QuietHoursEnd   string `json:"quiet_hours_end"`   // e.g. "07:00"
}

// DigestSettingsUpdate represents the input for updating digest settings
type DigestSettingsUpdate struct {
	Enabled         *bool   `json:"digest_enabled"`
	Interval        *int    `json:"digest_interval"`
	QuietHoursStart *string `json:"quiet_hours_start"`
	QuietHoursEnd   *string `json:"quiet_hours_end"`
}

// TelegramSettings represents Telegram notification configuration
type TelegramSettings struct {
	Enabled  bool     `json:"enabled"`
	BotToken string   `json:"bot_token"`
	ChatIDs  []string `json:"chat_ids"`
}

// EmailSettings represents email notification configuration
type EmailSettings struct {
	Enabled    bool     `json:"enabled"`
	SMTPHost   string   `json:"smtp_host"`
	SMTPPort   int      `json:"smtp_port"`
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	From       string   `json:"from"`
	FromName   string   `json:"from_name"`
	Recipients []string `json:"recipients"`
	UseTLS     bool     `json:"use_tls"`
}

// WebhookSettings represents webhook notification configuration
type WebhookSettings struct {
	Enabled   bool     `json:"enabled"`
	URLs      []string `json:"urls"`
	SecretKey string   `json:"secret_key"`
}

// MonitoringSettings represents monitoring configuration
type MonitoringSettings struct {
	UptimeInterval     int `json:"uptime_interval"`      // minutes
	SSLInterval        int `json:"ssl_interval"`         // minutes
	ContentInterval    int `json:"content_interval"`     // minutes
	SecurityInterval   int `json:"security_interval"`    // minutes
	VulnInterval       int `json:"vuln_interval"`        // minutes
	DorkInterval       int `json:"dork_interval"`        // minutes
	DNSInterval        int `json:"dns_interval"`         // minutes, 0 = disabled
	ResponseTimeThreshold int `json:"response_time_threshold"` // ms
	SSLExpiryWarningDays  int `json:"ssl_expiry_warning_days"`
	DataRetentionDays     int `json:"data_retention_days"`
}

// TelegramSettingsUpdate represents the input for updating Telegram settings
type TelegramSettingsUpdate struct {
	Enabled  *bool    `json:"enabled"`
	BotToken *string  `json:"bot_token"`
	ChatIDs  []string `json:"chat_ids"`
}

// EmailSettingsUpdate represents the input for updating email settings
type EmailSettingsUpdate struct {
	Enabled    *bool    `json:"enabled"`
	SMTPHost   *string  `json:"smtp_host"`
	SMTPPort   *int     `json:"smtp_port"`
	Username   *string  `json:"username"`
	Password   *string  `json:"password"`
	From       *string  `json:"from"`
	FromName   *string  `json:"from_name"`
	Recipients []string `json:"recipients"`
	UseTLS     *bool    `json:"use_tls"`
}

// WebhookSettingsUpdate represents the input for updating webhook settings
type WebhookSettingsUpdate struct {
	Enabled   *bool    `json:"enabled"`
	URLs      []string `json:"urls"`
	SecretKey *string  `json:"secret_key"`
}

// AISettings represents AI verification configuration
type AISettings struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"` // groq, mistral, anthropic
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
}

// AISettingsUpdate represents the input for updating AI settings
type AISettingsUpdate struct {
	Enabled  *bool   `json:"enabled"`
	Provider *string `json:"provider"`
	APIKey   *string `json:"api_key"`
	Model    *string `json:"model"`
}

// MonitoringSettingsUpdate represents the input for updating monitoring settings
type MonitoringSettingsUpdate struct {
	UptimeInterval        *int `json:"uptime_interval"`
	SSLInterval           *int `json:"ssl_interval"`
	ContentInterval       *int `json:"content_interval"`
	SecurityInterval      *int `json:"security_interval"`
	VulnInterval          *int `json:"vuln_interval"`
	DorkInterval          *int `json:"dork_interval"`
	DNSInterval           *int `json:"dns_interval"`
	ResponseTimeThreshold *int `json:"response_time_threshold"`
	SSLExpiryWarningDays  *int `json:"ssl_expiry_warning_days"`
	DataRetentionDays     *int `json:"data_retention_days"`
}

// DashboardTrends represents trend data for dashboard charts
type DashboardTrends struct {
	UptimeTrend        []UptimeTrendPoint        `json:"uptime_trend"`
	ResponseTimes      []ResponseTimeTrendPoint  `json:"response_times"`
	StatusDistribution StatusDistribution        `json:"status_distribution"`
}

// UptimeTrendPoint represents a single data point in uptime trend
type UptimeTrendPoint struct {
	Date             string  `json:"date" db:"date"`
	UptimePercentage float64 `json:"uptime_percentage" db:"uptime_percentage"`
}

// ResponseTimeTrendPoint represents a single data point in response time trend
type ResponseTimeTrendPoint struct {
	Date            string  `json:"date" db:"date"`
	AvgResponseTime float64 `json:"avg_response_time" db:"avg_response_time"`
}

// StatusDistribution represents the current status distribution
type StatusDistribution struct {
	Up       int `json:"up"`
	Down     int `json:"down"`
	Degraded int `json:"degraded"`
}
