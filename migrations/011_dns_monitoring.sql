-- Migration 011: DNS monitoring - store DNS scan results for periodic checks

CREATE TABLE IF NOT EXISTS dns_scans (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    website_id BIGINT UNSIGNED NOT NULL,
    domain_name VARCHAR(255) NOT NULL,
    has_spf BOOLEAN DEFAULT FALSE,
    has_dmarc BOOLEAN DEFAULT FALSE,
    spf_record TEXT,
    dmarc_record TEXT,
    nameservers JSON,
    mx_records JSON,
    dns_records JSON,
    subdomains JSON,
    subdomain_count INT DEFAULT 0,
    scan_duration_ms INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,
    INDEX idx_dns_scans_website (website_id),
    INDEX idx_dns_scans_domain (domain_name),
    INDEX idx_dns_scans_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
