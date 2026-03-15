-- Migration 010: Add per-website response time threshold customization
-- Allows overriding global response_time_warning and response_time_critical per website

ALTER TABLE websites
    ADD COLUMN response_time_warning INT DEFAULT NULL COMMENT 'Custom warning threshold in ms (overrides global)',
    ADD COLUMN response_time_critical INT DEFAULT NULL COMMENT 'Custom critical threshold in ms (overrides global)';
