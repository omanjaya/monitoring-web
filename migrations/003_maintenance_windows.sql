-- Migration: Create maintenance_windows table
-- Version: 003
-- Created: 2025-01-22

CREATE TABLE IF NOT EXISTS maintenance_windows (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    website_id BIGINT UNSIGNED NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NULL,
    status ENUM('scheduled', 'in_progress', 'completed', 'cancelled') NOT NULL DEFAULT 'scheduled',
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    created_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET DEFAULT,
    INDEX idx_maintenance_status (status),
    INDEX idx_maintenance_times (start_time, end_time),
    INDEX idx_maintenance_website (website_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Add comment
ALTER TABLE maintenance_windows COMMENT = 'Stores scheduled maintenance windows for websites';
