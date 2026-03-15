-- Defacement archive monitoring (Zone-H, Zone-XSEC)
CREATE TABLE IF NOT EXISTS defacement_incidents (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    website_id BIGINT UNSIGNED NOT NULL,
    source VARCHAR(20) NOT NULL COMMENT 'zone_h or zone_xsec',
    source_id VARCHAR(100) DEFAULT NULL COMMENT 'ID from source site',
    defaced_url VARCHAR(500) NOT NULL,
    attacker VARCHAR(200) DEFAULT NULL,
    team VARCHAR(200) DEFAULT NULL,
    defaced_at DATETIME DEFAULT NULL,
    mirror_url VARCHAR(500) DEFAULT NULL,
    is_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at DATETIME DEFAULT NULL,
    acknowledged_by VARCHAR(100) DEFAULT NULL,
    notes TEXT DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_source_url (source, defaced_url),
    INDEX idx_website (website_id),
    INDEX idx_source (source),
    INDEX idx_defaced_at (defaced_at),
    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS defacement_scans (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    source VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'running',
    total_checked INT NOT NULL DEFAULT 0,
    new_incidents INT NOT NULL DEFAULT 0,
    started_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME DEFAULT NULL,
    error_message TEXT DEFAULT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
