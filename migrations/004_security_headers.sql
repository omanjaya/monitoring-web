-- Migration: Create security_header_checks table and add security fields to websites
-- Version: 004
-- Created: 2025-01-22

-- Add security fields to websites table
ALTER TABLE websites
ADD COLUMN security_score INT NULL DEFAULT NULL,
ADD COLUMN security_grade VARCHAR(5) NULL DEFAULT NULL,
ADD COLUMN security_checked_at TIMESTAMP NULL DEFAULT NULL;

-- Create security header checks table
CREATE TABLE IF NOT EXISTS security_header_checks (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    website_id BIGINT UNSIGNED NOT NULL,
    score INT NOT NULL DEFAULT 0,
    grade VARCHAR(5) NOT NULL DEFAULT 'F',
    headers JSON NULL,
    findings JSON NULL,
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    INDEX idx_security_website (website_id),
    INDEX idx_security_checked (checked_at),
    INDEX idx_security_score (score)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add comment
ALTER TABLE security_header_checks COMMENT = 'Stores security headers check results for websites';
