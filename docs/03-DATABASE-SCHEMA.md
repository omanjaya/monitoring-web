# Dokumen Database Schema Design

## Monitoring Website Pemerintah Provinsi Bali

---

## 1. Database Overview

**Database Engine:** MySQL 8.0+
**Character Set:** utf8mb4
**Collation:** utf8mb4_unicode_ci
**Storage Engine:** InnoDB

---

## 2. Entity Relationship Diagram (ERD)

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐        │
│  │     opd      │         │   websites   │         │    users     │        │
│  ├──────────────┤         ├──────────────┤         ├──────────────┤        │
│  │ id           │◀────────│ opd_id       │         │ id           │        │
│  │ name         │    1:N  │ id           │         │ username     │        │
│  │ code         │         │ url          │         │ email        │        │
│  │ contact_email│         │ name         │         │ password     │        │
│  └──────────────┘         │ is_active    │         │ role         │        │
│                           │ status       │         │ opd_id       │────┐   │
│                           └──────┬───────┘         └──────────────┘    │   │
│                                  │                                      │   │
│                           1:N    │                                      │   │
│                                  │                                      │   │
│       ┌──────────────────────────┼──────────────────────────┐          │   │
│       │                          │                          │          │   │
│       ▼                          ▼                          ▼          │   │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐   │   │
│  │   checks     │         │    alerts    │         │  ssl_checks  │   │   │
│  ├──────────────┤         ├──────────────┤         ├──────────────┤   │   │
│  │ id           │         │ id           │         │ id           │   │   │
│  │ website_id   │         │ website_id   │         │ website_id   │   │   │
│  │ status_code  │         │ type         │         │ issuer       │   │   │
│  │ response_time│         │ severity     │         │ valid_from   │   │   │
│  │ checked_at   │         │ message      │         │ valid_until  │   │   │
│  └──────────────┘         │ is_resolved  │         │ days_until   │   │   │
│                           └──────────────┘         └──────────────┘   │   │
│                                                                        │   │
│  ┌──────────────┐         ┌──────────────┐         ┌──────────────┐   │   │
│  │content_scans │         │  keywords    │         │ audit_logs   │   │   │
│  ├──────────────┤         ├──────────────┤         ├──────────────┤   │   │
│  │ id           │         │ id           │         │ id           │   │   │
│  │ website_id   │         │ keyword      │         │ user_id      │◀──┘   │
│  │ is_clean     │         │ category     │         │ action       │       │
│  │ findings     │         │ is_active    │         │ entity_type  │       │
│  │ scanned_at   │         └──────────────┘         │ entity_id    │       │
│  └──────────────┘                                  │ created_at   │       │
│                                                    └──────────────┘       │
│                                                                           │
│  ┌──────────────┐         ┌──────────────┐                               │
│  │notifications │         │   settings   │                               │
│  ├──────────────┤         ├──────────────┤                               │
│  │ id           │         │ key          │                               │
│  │ alert_id     │         │ value        │                               │
│  │ channel      │         │ description  │                               │
│  │ sent_at      │         └──────────────┘                               │
│  │ status       │                                                        │
│  └──────────────┘                                                        │
│                                                                           │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Table Definitions

### 3.1 Table: `opd` (Organisasi Perangkat Daerah)

Menyimpan data OPD/dinas yang memiliki website.

```sql
CREATE TABLE opd (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name            VARCHAR(255) NOT NULL COMMENT 'Nama OPD lengkap',
    code            VARCHAR(50) NOT NULL UNIQUE COMMENT 'Kode OPD (singkatan)',
    contact_email   VARCHAR(255) NULL COMMENT 'Email PIC OPD',
    contact_phone   VARCHAR(20) NULL COMMENT 'No HP PIC OPD',
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_opd_code (code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Sample Data:**
| id | name | code | contact_email |
|----|------|------|---------------|
| 1 | Dinas Komunikasi Informatika dan Statistik | DISKOMINFOS | admin@diskominfos.baliprov.go.id |
| 2 | Dinas Kesehatan | DINKES | admin@dinkes.baliprov.go.id |
| 3 | Badan Perencanaan Pembangunan Daerah | BAPPEDA | admin@bappeda.baliprov.go.id |

---

### 3.2 Table: `websites`

Menyimpan daftar website yang dimonitor.

```sql
CREATE TABLE websites (
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
```

---

### 3.3 Table: `checks` (Uptime Check History)

Menyimpan riwayat pengecekan uptime.

```sql
CREATE TABLE checks (
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

-- Partitioning by month for better performance (optional)
-- ALTER TABLE checks PARTITION BY RANGE (UNIX_TIMESTAMP(checked_at)) (
--     PARTITION p202601 VALUES LESS THAN (UNIX_TIMESTAMP('2026-02-01')),
--     PARTITION p202602 VALUES LESS THAN (UNIX_TIMESTAMP('2026-03-01')),
--     ...
-- );
```

---

### 3.4 Table: `ssl_checks` (SSL Certificate History)

Menyimpan riwayat pengecekan SSL certificate.

```sql
CREATE TABLE ssl_checks (
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
    cipher_suite    VARCHAR(100) NULL,

    -- Error info
    error_message   TEXT NULL,

    checked_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (website_id) REFERENCES websites(id) ON DELETE CASCADE,

    INDEX idx_ssl_website (website_id),
    INDEX idx_ssl_expiry (days_until_expiry),
    INDEX idx_ssl_time (checked_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

### 3.5 Table: `content_scans` (Content Scan History)

Menyimpan riwayat scan konten untuk deteksi judol/defacement.

```sql
CREATE TABLE content_scans (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- Scan results
    is_clean        BOOLEAN NOT NULL COMMENT 'TRUE jika tidak ditemukan masalah',
    scan_type       ENUM('full', 'quick') DEFAULT 'quick',

    -- Findings (JSON array of found issues)
    findings        JSON NULL COMMENT 'Array of findings',
    -- Example: [{"type": "keyword", "keyword": "slot gacor", "location": "body", "snippet": "..."}]

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
```

---

### 3.6 Table: `alerts`

Menyimpan semua alert yang di-generate.

```sql
CREATE TABLE alerts (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    website_id      BIGINT UNSIGNED NOT NULL,

    -- Alert info
    type            ENUM('down', 'up', 'ssl_expiring', 'ssl_expired', 'judol_detected', 'defacement', 'slow_response') NOT NULL,
    severity        ENUM('info', 'warning', 'critical') NOT NULL,
    title           VARCHAR(255) NOT NULL,
    message         TEXT NOT NULL,

    -- Alert context (JSON for flexible data)
    context         JSON NULL COMMENT 'Additional context data',
    -- Example: {"status_code": 500, "response_time": 5000, "keywords_found": ["slot gacor"]}

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
```

---

### 3.7 Table: `notifications`

Menyimpan riwayat notifikasi yang dikirim.

```sql
CREATE TABLE notifications (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    alert_id        BIGINT UNSIGNED NOT NULL,

    -- Notification info
    channel         ENUM('telegram', 'email', 'sms', 'webhook') NOT NULL,
    recipient       VARCHAR(255) NOT NULL COMMENT 'Chat ID / Email / Phone',

    -- Status
    status          ENUM('pending', 'sent', 'failed') DEFAULT 'pending',
    sent_at         TIMESTAMP NULL,
    error_message   TEXT NULL,

    -- Retry info
    retry_count     INT DEFAULT 0,
    next_retry_at   TIMESTAMP NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE,

    INDEX idx_notif_alert (alert_id),
    INDEX idx_notif_status (status),
    INDEX idx_notif_channel (channel)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

### 3.8 Table: `keywords`

Menyimpan daftar keyword untuk deteksi judol/defacement.

```sql
CREATE TABLE keywords (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    keyword         VARCHAR(255) NOT NULL,
    category        ENUM('gambling', 'defacement', 'malware', 'phishing', 'custom') NOT NULL,
    is_regex        BOOLEAN DEFAULT FALSE COMMENT 'Apakah keyword adalah regex pattern',
    is_active       BOOLEAN DEFAULT TRUE,
    weight          INT DEFAULT 1 COMMENT 'Bobot untuk scoring (1-10)',

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_keyword_unique (keyword, category),
    INDEX idx_keyword_category (category),
    INDEX idx_keyword_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

### 3.9 Table: `users`

Menyimpan data user dashboard.

**Phase 1:** Hanya 1 user super admin
**Phase 2:** Multi-user dengan role system (lihat checklist di bawah)

```sql
CREATE TABLE users (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,

    -- Auth info
    username        VARCHAR(50) NOT NULL UNIQUE,
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,

    -- Profile
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(20) NULL,

    -- Role (Phase 1: hanya super_admin)
    -- Phase 2: tambah 'admin_opd', 'viewer' dan relasi ke tabel opd
    role            VARCHAR(50) NOT NULL DEFAULT 'super_admin',

    -- Status
    is_active       BOOLEAN DEFAULT TRUE,
    last_login_at   TIMESTAMP NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    INDEX idx_users_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Default admin user (ganti password setelah install!)
INSERT INTO users (username, email, password_hash, full_name, role) VALUES
('admin', 'admin@diskominfos.baliprov.go.id', '$2a$12$CHANGE_THIS_HASH', 'Administrator', 'super_admin');
```

#### Future: Role System Tables (Phase 2)

```sql
-- Uncomment dan jalankan saat implementasi multi-role

-- CREATE TABLE roles (
--     id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
--     name        VARCHAR(50) NOT NULL UNIQUE,
--     description VARCHAR(255) NULL,
--     created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

-- CREATE TABLE permissions (
--     id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
--     name        VARCHAR(100) NOT NULL UNIQUE,
--     description VARCHAR(255) NULL
-- );

-- CREATE TABLE role_permissions (
--     role_id       BIGINT UNSIGNED NOT NULL,
--     permission_id BIGINT UNSIGNED NOT NULL,
--     PRIMARY KEY (role_id, permission_id),
--     FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
--     FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
-- );

-- ALTER TABLE users ADD COLUMN opd_id BIGINT UNSIGNED NULL;
-- ALTER TABLE users ADD FOREIGN KEY (opd_id) REFERENCES opd(id) ON DELETE SET NULL;
```

---

### 3.10 Table: `audit_logs`

Menyimpan audit trail untuk compliance.

```sql
CREATE TABLE audit_logs (
    id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT UNSIGNED NULL COMMENT 'NULL untuk system actions',

    -- Action info
    action          VARCHAR(50) NOT NULL COMMENT 'create, update, delete, login, etc',
    entity_type     VARCHAR(50) NOT NULL COMMENT 'website, user, alert, etc',
    entity_id       BIGINT UNSIGNED NULL,

    -- Change details
    old_values      JSON NULL COMMENT 'Nilai sebelum perubahan',
    new_values      JSON NULL COMMENT 'Nilai setelah perubahan',

    -- Request info
    ip_address      VARCHAR(45) NULL,
    user_agent      VARCHAR(500) NULL,

    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_audit_user (user_id),
    INDEX idx_audit_entity (entity_type, entity_id),
    INDEX idx_audit_action (action),
    INDEX idx_audit_time (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

### 3.11 Table: `settings`

Menyimpan konfigurasi sistem.

```sql
CREATE TABLE settings (
    `key`           VARCHAR(100) PRIMARY KEY,
    `value`         TEXT NOT NULL,
    `type`          ENUM('string', 'int', 'bool', 'json') DEFAULT 'string',
    description     VARCHAR(500) NULL,

    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**Default Settings:**
```sql
INSERT INTO settings (`key`, `value`, `type`, description) VALUES
('uptime_check_interval', '5', 'int', 'Interval uptime check dalam menit'),
('content_scan_interval', '30', 'int', 'Interval content scan dalam menit'),
('ssl_check_interval', '1440', 'int', 'Interval SSL check dalam menit (default 24 jam)'),
('http_timeout', '30', 'int', 'HTTP request timeout dalam detik'),
('response_time_warning', '3000', 'int', 'Threshold warning response time (ms)'),
('response_time_critical', '10000', 'int', 'Threshold critical response time (ms)'),
('ssl_expiry_warning_days', '30', 'int', 'Warning jika SSL expired dalam X hari'),
('telegram_bot_token', '', 'string', 'Telegram Bot Token'),
('telegram_chat_id', '', 'string', 'Telegram Chat ID untuk alert'),
('max_concurrent_checks', '20', 'int', 'Maksimum concurrent website checks');
```

---

## 4. Database Views

### 4.1 View: `v_website_status`

View untuk dashboard overview.

```sql
CREATE VIEW v_website_status AS
SELECT
    w.id,
    w.url,
    w.name,
    o.name AS opd_name,
    o.code AS opd_code,
    w.status,
    w.last_status_code,
    w.last_response_time,
    w.last_checked_at,
    w.ssl_valid,
    w.ssl_expiry_date,
    w.content_clean,
    w.last_scan_at,
    CASE
        WHEN w.status = 'down' THEN 'critical'
        WHEN w.content_clean = FALSE THEN 'critical'
        WHEN w.ssl_valid = FALSE THEN 'warning'
        WHEN DATEDIFF(w.ssl_expiry_date, CURDATE()) <= 7 THEN 'warning'
        WHEN w.status = 'degraded' THEN 'warning'
        ELSE 'healthy'
    END AS health_status
FROM websites w
LEFT JOIN opd o ON w.opd_id = o.id
WHERE w.is_active = TRUE;
```

### 4.2 View: `v_uptime_stats`

View untuk statistik uptime.

```sql
CREATE VIEW v_uptime_stats AS
SELECT
    website_id,
    DATE(checked_at) AS check_date,
    COUNT(*) AS total_checks,
    SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) AS up_count,
    SUM(CASE WHEN status IN ('down', 'timeout', 'error') THEN 1 ELSE 0 END) AS down_count,
    ROUND(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) AS uptime_percentage,
    AVG(response_time) AS avg_response_time,
    MAX(response_time) AS max_response_time,
    MIN(response_time) AS min_response_time
FROM checks
GROUP BY website_id, DATE(checked_at);
```

---

## 5. Indexes Strategy

### 5.1 Primary Indexes
- Semua tabel memiliki primary key auto-increment

### 5.2 Foreign Key Indexes
- Otomatis dibuat untuk foreign key relationships

### 5.3 Query Optimization Indexes
- `checks(website_id, checked_at DESC)` - untuk query history terbaru
- `alerts(is_resolved, created_at DESC)` - untuk alert aktif
- `websites(status, is_active)` - untuk dashboard filtering

### 5.4 Composite Indexes
```sql
-- Untuk query checks per website dalam range waktu
CREATE INDEX idx_checks_website_time_range ON checks(website_id, checked_at);

-- Untuk query alerts aktif per website
CREATE INDEX idx_alerts_active ON alerts(website_id, is_resolved, created_at DESC);
```

---

## 6. Data Retention Policy

### 6.1 Retention Rules

| Table | Retention Period | Action |
|-------|------------------|--------|
| checks | 90 hari | DELETE |
| ssl_checks | 90 hari | DELETE |
| content_scans | 90 hari | DELETE |
| notifications | 30 hari | DELETE |
| audit_logs | 1 tahun | ARCHIVE |
| alerts | 1 tahun | ARCHIVE |

### 6.2 Cleanup Procedure

```sql
DELIMITER //

CREATE PROCEDURE cleanup_old_data()
BEGIN
    -- Delete old checks (90 days)
    DELETE FROM checks WHERE checked_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

    -- Delete old ssl_checks (90 days)
    DELETE FROM ssl_checks WHERE checked_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

    -- Delete old content_scans (90 days)
    DELETE FROM content_scans WHERE scanned_at < DATE_SUB(NOW(), INTERVAL 90 DAY);

    -- Delete old notifications (30 days)
    DELETE FROM notifications WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
END //

DELIMITER ;

-- Schedule daily cleanup
CREATE EVENT evt_daily_cleanup
ON SCHEDULE EVERY 1 DAY
STARTS CURRENT_DATE + INTERVAL 1 DAY + INTERVAL 3 HOUR
DO CALL cleanup_old_data();
```

---

## 7. Sample Data (Seeder)

```sql
-- Insert default keywords
INSERT INTO keywords (keyword, category, weight) VALUES
-- Gambling keywords
('slot gacor', 'gambling', 10),
('judi online', 'gambling', 10),
('togel', 'gambling', 10),
('casino online', 'gambling', 10),
('poker online', 'gambling', 9),
('sbobet', 'gambling', 9),
('pragmatic', 'gambling', 8),
('joker123', 'gambling', 8),
('maxwin', 'gambling', 8),
('scatter', 'gambling', 7),
('jackpot', 'gambling', 6),
('rtp slot', 'gambling', 8),
('bocoran slot', 'gambling', 9),
('demo slot', 'gambling', 5),
('deposit pulsa', 'gambling', 7),
('bandar togel', 'gambling', 10),
('live casino', 'gambling', 9),

-- Defacement keywords
('hacked by', 'defacement', 10),
('defaced by', 'defacement', 10),
('owned by', 'defacement', 8),
('greetz to', 'defacement', 9),
('cyber army', 'defacement', 7);

-- Insert default super admin
INSERT INTO users (username, email, password_hash, full_name, role) VALUES
('admin', 'admin@diskominfos.baliprov.go.id', '$2a$10$HASH_HERE', 'Administrator', 'super_admin');
```

---

## 8. Migration Files

### Migration 001: Initial Schema

File: `migrations/001_initial_schema.up.sql`

```sql
-- Create all tables in order of dependencies
-- 1. opd
-- 2. websites
-- 3. checks
-- 4. ssl_checks
-- 5. content_scans
-- 6. alerts
-- 7. notifications
-- 8. keywords
-- 9. users
-- 10. audit_logs
-- 11. settings
```

### Migration 001 Down

File: `migrations/001_initial_schema.down.sql`

```sql
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS alerts;
DROP TABLE IF EXISTS content_scans;
DROP TABLE IF EXISTS ssl_checks;
DROP TABLE IF EXISTS checks;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS keywords;
DROP TABLE IF EXISTS settings;
DROP TABLE IF EXISTS websites;
DROP TABLE IF EXISTS opd;
```
