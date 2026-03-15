package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Database     DatabaseConfig     `mapstructure:"database"`
	Telegram     TelegramConfig     `mapstructure:"telegram"`
	Email        EmailConfig        `mapstructure:"email"`
	Webhook      WebhookConfig      `mapstructure:"webhook"`
	Notification NotificationConfig `mapstructure:"notification"`
	Monitoring   MonitoringConfig   `mapstructure:"monitoring"`
	Keywords     KeywordsConfig     `mapstructure:"keywords"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Scheduler    SchedulerConfig    `mapstructure:"scheduler"`
	AI           AIConfig           `mapstructure:"ai"`
}

type AIConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Provider string `mapstructure:"provider"` // "anthropic" or "openai"
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"` // e.g. "claude-haiku-4-5-20251001"
}

type NotificationConfig struct {
	DigestEnabled   bool   `mapstructure:"digest_enabled"`
	DigestInterval  int    `mapstructure:"digest_interval"`   // minutes, default 15
	DigestCron      string `mapstructure:"digest_cron"`       // optional cron expression override
	QuietHoursStart string `mapstructure:"quiet_hours_start"` // e.g. "22:00"
	QuietHoursEnd   string `mapstructure:"quiet_hours_end"`   // e.g. "07:00"
}

type ServerConfig struct {
	Host           string   `mapstructure:"host"`
	Port           int      `mapstructure:"port"`
	Mode           string   `mapstructure:"mode"` // debug, release
	AllowedOrigins []string `mapstructure:"allowed_origins"`
	CSPPolicy      string   `mapstructure:"csp_policy"`
}

type DatabaseConfig struct {
	Host          string `mapstructure:"host"`
	Port          int    `mapstructure:"port"`
	User          string `mapstructure:"user"`
	Password      string `mapstructure:"password"`
	Name          string `mapstructure:"name"`
	RetentionDays int    `mapstructure:"retention_days"` // Days to keep old data (default: 90)
}

func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

type TelegramConfig struct {
	BotToken string   `mapstructure:"bot_token"`
	ChatIDs  []string `mapstructure:"chat_ids"`
	Enabled  bool     `mapstructure:"enabled"`
}

type EmailConfig struct {
	Enabled    bool     `mapstructure:"enabled"`
	SMTPHost   string   `mapstructure:"smtp_host"`
	SMTPPort   int      `mapstructure:"smtp_port"`
	Username   string   `mapstructure:"username"`
	Password   string   `mapstructure:"password"`
	From       string   `mapstructure:"from"`
	FromName   string   `mapstructure:"from_name"`
	Recipients []string `mapstructure:"recipients"`
	UseTLS     bool     `mapstructure:"use_tls"`
}

type WebhookConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	URLs      []string `mapstructure:"urls"`
	SecretKey string   `mapstructure:"secret_key"`
	Timeout   int      `mapstructure:"timeout"` // seconds
}

type MonitoringConfig struct {
	UptimeInterval       int  `mapstructure:"uptime_interval"`        // minutes
	ContentScanInterval  int  `mapstructure:"content_scan_interval"`  // minutes
	SSLCheckInterval     int  `mapstructure:"ssl_check_interval"`     // minutes
	HTTPTimeout          int  `mapstructure:"http_timeout"`           // seconds
	ResponseTimeWarning  int  `mapstructure:"response_time_warning"`  // ms
	ResponseTimeCritical int  `mapstructure:"response_time_critical"` // ms
	MaxConcurrentChecks  int  `mapstructure:"max_concurrent_checks"`
	RateLimitPerMinute   int  `mapstructure:"rate_limit_per_minute"`  // websites per minute (to avoid IP ban)
	RestrictDomain       bool `mapstructure:"restrict_domain"`
	EnableHTTP2          bool `mapstructure:"enable_http2"`           // Enable HTTP/2 support (default: true)
	EnableIPv6           bool `mapstructure:"enable_ipv6"`            // Enable IPv6 resolution checking (default: true)
}

type KeywordsConfig struct {
	Gambling   []string `mapstructure:"gambling"`
	Defacement []string `mapstructure:"defacement"`
}

type JWTConfig struct {
	SecretKey       string `mapstructure:"secret_key"`
	ExpirationHours int    `mapstructure:"expiration_hours"`
}

type SchedulerConfig struct {
	UptimeCheck  string `mapstructure:"uptime_check"`   // Cron expression
	SSLCheck     string `mapstructure:"ssl_check"`      // Cron expression
	ContentScan  string `mapstructure:"content_scan"`   // Cron expression
	DailySummary string `mapstructure:"daily_summary"`  // Cron expression
	Cleanup      string `mapstructure:"cleanup"`        // Cron expression for database cleanup
	DorkScan          string `mapstructure:"dork_scan"`          // Cron expression for dork scan (judol/defacement detection)
	VulnerabilityScan string `mapstructure:"vulnerability_scan"` // Cron expression for vulnerability scan
	DNSScan           string `mapstructure:"dns_scan"`            // Cron expression for DNS scan
	SecurityScan      string `mapstructure:"security_scan"`       // Cron expression for security headers scan
}

// Validate checks the configuration for common issues and returns all validation
// errors joined together. This allows running with warnings in development.
func (c *Config) Validate() error {
	var errs []string

	// Server validation
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be between 1 and 65535, got %d", c.Server.Port))
	}

	// Database validation
	if c.Database.Host == "" {
		errs = append(errs, "database.host must not be empty")
	}
	if c.Database.Name == "" {
		errs = append(errs, "database.name must not be empty")
	}

	// JWT validation
	if c.JWT.SecretKey == "change-this-secret-key-in-production" {
		errs = append(errs, "jwt.secret_key is using the default value; change it for production use")
	}
	if len(c.JWT.SecretKey) < 32 {
		errs = append(errs, fmt.Sprintf("jwt.secret_key must be at least 32 characters, got %d", len(c.JWT.SecretKey)))
	}

	// Monitoring validation
	if c.Monitoring.HTTPTimeout <= 0 {
		errs = append(errs, fmt.Sprintf("monitoring.http_timeout must be > 0, got %d", c.Monitoring.HTTPTimeout))
	}
	if c.Monitoring.MaxConcurrentChecks <= 0 {
		errs = append(errs, fmt.Sprintf("monitoring.max_concurrent_checks must be > 0, got %d", c.Monitoring.MaxConcurrentChecks))
	}

	if len(errs) > 0 {
		return fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return nil
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/app")

	// Environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.allowed_origins", []string{"*"})
	viper.SetDefault("server.csp_policy", "")

	// Database defaults
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 3306)
	viper.SetDefault("database.user", "root")
	viper.SetDefault("database.password", "")
	viper.SetDefault("database.name", "monitoring_website")

	// Telegram defaults
	viper.SetDefault("telegram.enabled", true)

	// Email defaults
	viper.SetDefault("email.enabled", false)
	viper.SetDefault("email.smtp_port", 587)
	viper.SetDefault("email.use_tls", true)
	viper.SetDefault("email.from_name", "Monitoring Website Diskominfos Bali")

	// Webhook defaults
	viper.SetDefault("webhook.enabled", false)
	viper.SetDefault("webhook.timeout", 10)

	// Monitoring defaults
	viper.SetDefault("monitoring.uptime_interval", 5)
	viper.SetDefault("monitoring.content_scan_interval", 30)
	viper.SetDefault("monitoring.ssl_check_interval", 1440) // 24 hours
	viper.SetDefault("monitoring.http_timeout", 30)
	viper.SetDefault("monitoring.response_time_warning", 3000)
	viper.SetDefault("monitoring.response_time_critical", 10000)
	viper.SetDefault("monitoring.max_concurrent_checks", 20)
	viper.SetDefault("monitoring.rate_limit_per_minute", 10) // 10 websites per minute to avoid IP ban
	viper.SetDefault("monitoring.enable_http2", true)
	viper.SetDefault("monitoring.enable_ipv6", true)

	// JWT defaults
	viper.SetDefault("jwt.secret_key", "change-this-secret-key-in-production")
	viper.SetDefault("jwt.expiration_hours", 24)

	// Scheduler defaults (cron with seconds)
	viper.SetDefault("scheduler.uptime_check", "0 */5 * * * *")    // Every 5 minutes
	viper.SetDefault("scheduler.ssl_check", "0 0 */6 * * *")       // Every 6 hours
	viper.SetDefault("scheduler.content_scan", "0 0 * * * *")      // Every hour
	viper.SetDefault("scheduler.daily_summary", "0 0 8 * * *")     // 8 AM daily
	viper.SetDefault("scheduler.cleanup", "0 0 3 * * *")           // 3 AM daily
	viper.SetDefault("scheduler.dork_scan", "0 0 */4 * * *")       // Every 4 hours (judol/defacement detection)
	viper.SetDefault("scheduler.dns_scan", "0 0 */12 * * *")      // Every 12 hours

	// Notification digest defaults
	viper.SetDefault("notification.digest_enabled", false)
	viper.SetDefault("notification.digest_interval", 15)
	viper.SetDefault("notification.quiet_hours_start", "")
	viper.SetDefault("notification.quiet_hours_end", "")

	// Database retention defaults
	viper.SetDefault("database.retention_days", 90) // Keep data for 90 days

	// AI verification defaults
	viper.SetDefault("ai.enabled", false)
	viper.SetDefault("ai.provider", "groq")
	viper.SetDefault("ai.api_key", "")
	viper.SetDefault("ai.model", "") // auto-detect based on provider

	// Default keywords
	viper.SetDefault("keywords.gambling", []string{
		"slot gacor", "slot online", "judi online", "togel", "casino",
		"poker online", "pragmatic", "joker123", "sbobet", "maxwin",
		"scatter", "jackpot", "rtp slot", "bocoran slot", "demo slot",
		"bandar togel", "live casino", "deposit pulsa",
	})
	viper.SetDefault("keywords.defacement", []string{
		"hacked by", "defaced by", "owned by", "greetz to", "cyber army",
	})
}
