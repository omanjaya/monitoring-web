-- Migration: Add alert escalation support
-- Created: 2025-01-22

-- Escalation policies table
CREATE TABLE IF NOT EXISTS escalation_policies (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Escalation rules table
CREATE TABLE IF NOT EXISTS escalation_rules (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    policy_id BIGINT UNSIGNED NOT NULL,
    level INT NOT NULL DEFAULT 1,
    severity VARCHAR(20) NOT NULL,
    delay_minutes INT NOT NULL DEFAULT 0,
    notify_channels JSON,
    repeat_interval INT DEFAULT 0,
    max_repeat INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (policy_id) REFERENCES escalation_policies(id) ON DELETE CASCADE,
    INDEX idx_policy_level (policy_id, level),
    INDEX idx_severity (severity)
);

-- Escalation contacts table
CREATE TABLE IF NOT EXISTS escalation_contacts (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    rule_id BIGINT UNSIGNED NOT NULL,
    channel VARCHAR(20) NOT NULL,
    value VARCHAR(255) NOT NULL,
    name VARCHAR(100),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (rule_id) REFERENCES escalation_rules(id) ON DELETE CASCADE,
    INDEX idx_rule_channel (rule_id, channel)
);

-- Escalation history table
CREATE TABLE IF NOT EXISTS escalation_history (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    alert_id BIGINT UNSIGNED NOT NULL,
    rule_id BIGINT UNSIGNED NOT NULL,
    level INT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    recipient VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    error_message TEXT,
    escalated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    acknowledged_at TIMESTAMP NULL,
    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,
    FOREIGN KEY (rule_id) REFERENCES escalation_rules(id) ON DELETE CASCADE,
    INDEX idx_alert_level (alert_id, level),
    INDEX idx_status (status),
    INDEX idx_escalated_at (escalated_at)
);

-- Add escalation tracking fields to alerts table
ALTER TABLE alerts
    ADD COLUMN escalation_level INT DEFAULT 0,
    ADD COLUMN last_escalated_at TIMESTAMP NULL,
    ADD COLUMN escalation_count INT DEFAULT 0,
    ADD COLUMN policy_id BIGINT UNSIGNED NULL;

-- Create default escalation policy
INSERT INTO escalation_policies (name, description, is_active, is_default)
VALUES ('Default Policy', 'Default escalation policy for all alerts', TRUE, TRUE);

-- Create default escalation rules for the default policy
INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
SELECT
    p.id,
    1,
    'critical',
    0,
    '["telegram", "email"]',
    15,
    3
FROM escalation_policies p WHERE p.is_default = TRUE;

INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
SELECT
    p.id,
    2,
    'critical',
    15,
    '["telegram", "email"]',
    30,
    2
FROM escalation_policies p WHERE p.is_default = TRUE;

INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
SELECT
    p.id,
    3,
    'critical',
    30,
    '["telegram", "email", "webhook"]',
    60,
    0
FROM escalation_policies p WHERE p.is_default = TRUE;

INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
SELECT
    p.id,
    1,
    'warning',
    5,
    '["telegram"]',
    30,
    2
FROM escalation_policies p WHERE p.is_default = TRUE;

INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
SELECT
    p.id,
    2,
    'warning',
    30,
    '["telegram", "email"]',
    60,
    1
FROM escalation_policies p WHERE p.is_default = TRUE;
