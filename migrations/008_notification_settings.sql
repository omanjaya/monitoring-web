-- Migration: 008_notification_settings.sql
-- Description: Add default notification and monitoring settings to settings table
-- Date: 2026-01-22

-- Insert default notification settings (using IGNORE to prevent duplicate key errors)

-- Telegram settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('telegram.enabled', 'false', 'bool', 'Enable Telegram notifications', NOW()),
('telegram.bot_token', '', 'string', 'Telegram bot token from @BotFather', NOW()),
('telegram.chat_ids', '[]', 'json', 'List of Telegram chat IDs to send notifications', NOW());

-- Email settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('email.enabled', 'false', 'bool', 'Enable email notifications', NOW()),
('email.smtp_host', '', 'string', 'SMTP server hostname', NOW()),
('email.smtp_port', '587', 'int', 'SMTP server port', NOW()),
('email.username', '', 'string', 'SMTP authentication username', NOW()),
('email.password', '', 'string', 'SMTP authentication password', NOW()),
('email.from', '', 'string', 'Sender email address', NOW()),
('email.from_name', 'Monitoring Website', 'string', 'Sender display name', NOW()),
('email.use_tls', 'true', 'bool', 'Use TLS encryption for SMTP', NOW()),
('email.recipients', '[]', 'json', 'List of email recipients', NOW());

-- Webhook settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('webhook.enabled', 'false', 'bool', 'Enable webhook notifications', NOW()),
('webhook.urls', '[]', 'json', 'List of webhook URLs to call', NOW()),
('webhook.secret_key', '', 'string', 'Secret key for webhook authentication', NOW());

-- Monitoring interval settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('monitoring.uptime_interval', '5', 'int', 'Uptime check interval in minutes', NOW()),
('monitoring.ssl_interval', '360', 'int', 'SSL check interval in minutes (default: 6 hours)', NOW()),
('monitoring.content_interval', '60', 'int', 'Content scan interval in minutes', NOW()),
('monitoring.security_interval', '1440', 'int', 'Security header check interval in minutes (default: 24 hours)', NOW()),
('monitoring.vuln_interval', '1440', 'int', 'Vulnerability scan interval in minutes (default: 24 hours)', NOW()),
('monitoring.dork_interval', '60', 'int', 'Dork/defacement scan interval in minutes', NOW());

-- Threshold settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('monitoring.response_time_threshold', '5000', 'int', 'Response time warning threshold in milliseconds', NOW()),
('monitoring.ssl_expiry_warning_days', '30', 'int', 'Days before SSL expiry to send warning', NOW()),
('monitoring.data_retention_days', '90', 'int', 'Days to retain check history data', NOW());

-- System settings
INSERT IGNORE INTO settings (`key`, `value`, `type`, description, updated_at) VALUES
('system.daily_summary_enabled', 'true', 'bool', 'Enable daily summary notifications', NOW()),
('system.daily_summary_time', '08:00', 'string', 'Time to send daily summary (24h format)', NOW());
