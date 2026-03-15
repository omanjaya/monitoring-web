-- Monitoring Website - Initial Schema
-- Version: 1.0.0

-- ============================================
-- Table: opd (Organisasi Perangkat Daerah)
-- ============================================
CREATE TABLE IF NOT EXISTS opd (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(255) NOT NULL COMMENT 'Nama OPD lengkap',
    code            VARCHAR(50) NOT NULL UNIQUE COMMENT 'Kode OPD (singkatan)',
    contact_email   VARCHAR(255) NULL COMMENT 'Email PIC OPD',
    contact_phone   VARCHAR(20) NULL COMMENT 'No HP PIC OPD',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_opd_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: websites
-- ============================================
CREATE TABLE IF NOT EXISTS websites (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    opd_id          BIGINT UNSIGNED NULL COMMENT 'FK ke tabel opd',
    url             VARCHAR(500) NOT NULL COMMENT 'URL lengkap website',
    name            VARCHAR(255) NOT NULL COMMENT 'Nama website',
    description     TEXT NULL COMMENT 'Deskripsi website',

    -- Monitoring settings
    is_active       BOOLEAN DEFAULT TRUE COMMENT 'Apakah website aktif dimonitor',
    check_interval  INT DEFAULT 5 COMMENT 'Interval check dalam menit',
    timeout         INT DEFAULT 30 COMMENT 'Timeout dalam detik',

    -- Current status (denormalized for performance)
    status          ENUM('up', 'down', 'degraded', 'unknown') DEFAULT 'unknown',
    last_status_code INT NULL COMMENT 'HTTP status code terakhir',
    last_response_time INT NULL COMMENT 'Response time terakhir (ms)',
    last_checked_at TIMESTAMP NULL COMMENT 'Waktu check terakhir',

    -- SSL info (denormalized)
    ssl_valid       BOOLEAN NULL,
    ssl_expiry_date DATE NULL,

    -- Content scan info (denormalized)
    content_clean   BOOLEAN DEFAULT TRUE COMMENT 'Apakah konten bersih dari judol',
    last_scan_at    TIMESTAMP NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    FOREIGN KEY (opd_id) REFERENCES opd(id) ON DELETE SET NULL,

    INDEX idx_websites_status (status),
    INDEX idx_websites_active (is_active),
    INDEX idx_websites_opd (opd_id),
    INDEX idx_websites_content_clean (content_clean),
    UNIQUE INDEX idx_websites_url (url(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: checks (Uptime Check History)
-- ============================================
CREATE TABLE IF NOT EXISTS checks (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- Check results
    status_code     INT NULL COMMENT 'HTTP status code',
    response_time   INT NULL COMMENT 'Response time dalam ms',
    status          ENUM('up', 'down', 'degraded', 'timeout', 'error') NOT NULL,

    -- Error info
    error_message   TEXT NULL COMMENT 'Pesan error jika gagal',
    error_type      VARCHAR(50) NULL COMMENT 'Tipe error: dns, connection, timeout, ssl',

    -- Response info
    content_length  INT NULL COMMENT 'Ukuran response body',

    checked_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,

    INDEX idx_checks_website (website_id),
    INDEX idx_checks_status (status),
    INDEX idx_checks_time (checked_at),
    INDEX idx_checks_website_time (website_id, checked_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: ssl_checks
-- ============================================
CREATE TABLE IF NOT EXISTS ssl_checks (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- SSL certificate info
    is_valid        BOOLEAN NOT NULL,
    issuer          VARCHAR(255) NULL COMMENT 'Certificate issuer',
    subject         VARCHAR(255) NULL COMMENT 'Certificate subject',
    valid_from      DATETIME NULL,
    valid_until     DATETIME NULL,
    days_until_expiry INT NULL COMMENT 'Hari sampai expired',

    -- SSL details
    protocol        VARCHAR(20) NULL COMMENT 'TLS version',

    -- Error info
    error_message   TEXT NULL,

    checked_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,

    INDEX idx_ssl_website (website_id),
    INDEX idx_ssl_expiry (days_until_expiry),
    INDEX idx_ssl_time (checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: content_scans
-- ============================================
CREATE TABLE IF NOT EXISTS content_scans (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- Scan results
    is_clean        BOOLEAN NOT NULL COMMENT 'TRUE jika tidak ditemukan masalah',
    scan_type       ENUM('full', 'quick') DEFAULT 'quick',

    -- Findings (JSON array of found issues)
    findings        JSON NULL COMMENT 'Array of findings',

    -- Page info
    page_title      VARCHAR(500) NULL,
    page_hash       CHAR(64) NULL COMMENT 'SHA256 hash of page content',

    -- Statistics
    keywords_found  INT DEFAULT 0 COMMENT 'Jumlah keyword ditemukan',
    iframes_found   INT DEFAULT 0 COMMENT 'Jumlah iframe mencurigakan',
    redirects_found INT DEFAULT 0 COMMENT 'Jumlah redirect mencurigakan',

    scanned_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,

    INDEX idx_content_website (website_id),
    INDEX idx_content_clean (is_clean),
    INDEX idx_content_time (scanned_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: alerts
-- ============================================
CREATE TABLE IF NOT EXISTS alerts (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- Alert info
    type            ENUM('down', 'up', 'ssl_expiring', 'ssl_expired', 'judol_detected', 'defacement', 'slow_response') NOT NULL,
    severity        ENUM('info', 'warning', 'critical') NOT NULL,
    title           VARCHAR(255) NOT NULL,
    message         TEXT NOT NULL,

    -- Alert context (JSON for flexible data)
    context         JSON NULL COMMENT 'Additional context data',

    -- Resolution
    is_resolved     BOOLEAN DEFAULT FALSE,
    resolved_at     TIMESTAMP NULL,
    resolved_by     BIGINT UNSIGNED NULL COMMENT 'User ID yang resolve',
    resolution_note TEXT NULL,

    -- Acknowledgement
    is_acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP NULL,
    acknowledged_by BIGINT UNSIGNED NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,

    INDEX idx_alerts_website (website_id),
    INDEX idx_alerts_type (type),
    INDEX idx_alerts_severity (severity),
    INDEX idx_alerts_resolved (is_resolved),
    INDEX idx_alerts_time (created_at DESC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: notifications
-- ============================================
CREATE TABLE IF NOT EXISTS notifications (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    alert_id        BIGINT UNSIGNED NOT NULL,

    -- Notification info
    channel         ENUM('telegram', 'email', 'webhook') NOT NULL,
    recipient       VARCHAR(255) NOT NULL COMMENT 'Chat ID / Email',

    -- Status
    status          ENUM('pending', 'sent', 'failed') DEFAULT 'pending',
    sent_at         TIMESTAMP NULL,
    error_message   TEXT NULL,

    -- Retry info
    retry_count     INT DEFAULT 0,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,

    INDEX idx_notif_alert (alert_id),
    INDEX idx_notif_status (status),
    INDEX idx_notif_channel (channel)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: keywords
-- ============================================
CREATE TABLE IF NOT EXISTS keywords (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    keyword         VARCHAR(255) NOT NULL,
    category        ENUM('gambling', 'defacement', 'malware', 'phishing', 'porn', 'custom') NOT NULL,
    is_regex        BOOLEAN DEFAULT FALSE COMMENT 'Apakah keyword adalah regex pattern',
    is_active       BOOLEAN DEFAULT TRUE,
    weight          INT DEFAULT 1 COMMENT 'Bobot untuk scoring (1-10)',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_keyword_unique (keyword, category),
    INDEX idx_keyword_category (category),
    INDEX idx_keyword_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: users
-- ============================================
CREATE TABLE IF NOT EXISTS users (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    -- Auth info
    username        VARCHAR(50) NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,

    -- Profile
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(20) NULL,

    -- Role (Phase 1: hanya super_admin)
    role            VARCHAR(50) NOT NULL DEFAULT 'super_admin',

    -- Status
    is_active       BOOLEAN DEFAULT TRUE,
    last_login_at   TIMESTAMP NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_users_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Table: settings
-- ============================================
CREATE TABLE IF NOT EXISTS settings (
    `key`           VARCHAR(100) PRIMARY KEY,
    `value`         TEXT NOT NULL,
    `type`          ENUM('string', 'int', 'bool', 'json') DEFAULT 'string',
    description     VARCHAR(500) NULL,

    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ============================================
-- Insert Default Data
-- ============================================

-- Default admin user (password: admin123 - GANTI SEGERA!)
-- Hash generated with bcrypt cost 12
INSERT INTO users (username, email, password_hash, full_name, role) VALUES
('admin', 'admin@diskominfos.baliprov.go.id', '$2a$10$b2cFlY0MqKUZErfozZnYLOzbWsp8tLWQxpqLXlLzvt2.BKjvvCjW2', 'Administrator', 'super_admin')
ON DUPLICATE KEY UPDATE username = username;

-- Default keywords (gambling)
INSERT IGNORE INTO keywords (keyword, category, weight) VALUES
('slot gacor', 'gambling', 10),
('judi online', 'gambling', 10),
('togel', 'gambling', 10),
('casino online', 'gambling', 10),
('poker online', 'gambling', 9),
('sbobet', 'gambling', 9),
('pragmatic', 'gambling', 8),
('joker123', 'gambling', 8),
('maxwin', 'gambling', 8),
('slot online', 'gambling', 8),
('scatter', 'gambling', 7),
('jackpot', 'gambling', 6),
('rtp slot', 'gambling', 8),
('bocoran slot', 'gambling', 9),
('demo slot', 'gambling', 5),
('bandar togel', 'gambling', 10),
('live casino', 'gambling', 9),
('deposit pulsa', 'gambling', 7);

-- Default keywords (defacement)
INSERT IGNORE INTO keywords (keyword, category, weight) VALUES
('hacked by', 'defacement', 10),
('defaced by', 'defacement', 10),
('owned by', 'defacement', 8),
('greetz to', 'defacement', 9),
('cyber army', 'defacement', 7);

-- Default settings
INSERT INTO settings (`key`, `value`, `type`, description) VALUES
('uptime_check_interval', '5', 'int', 'Interval uptime check dalam menit'),
('content_scan_interval', '30', 'int', 'Interval content scan dalam menit'),
('ssl_check_interval', '1440', 'int', 'Interval SSL check dalam menit'),
('http_timeout', '30', 'int', 'HTTP request timeout dalam detik'),
('response_time_warning', '3000', 'int', 'Threshold warning response time (ms)'),
('response_time_critical', '10000', 'int', 'Threshold critical response time (ms)'),
('telegram_enabled', 'true', 'bool', 'Enable Telegram notifications')
ON DUPLICATE KEY UPDATE `key` = `key`;

-- Sample OPD data
INSERT IGNORE INTO opd (name, code, contact_email) VALUES
('Dinas Komunikasi Informatika dan Statistik', 'DISKOMINFOS', 'admin@diskominfos.baliprov.go.id'),
('Dinas Kesehatan', 'DINKES', 'admin@dinkes.baliprov.go.id'),
('Badan Perencanaan Pembangunan Daerah', 'BAPPEDA', 'admin@bappeda.baliprov.go.id');
