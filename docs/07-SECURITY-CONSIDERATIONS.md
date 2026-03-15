# Dokumen Security Considerations

## Monitoring Website Pemerintah Provinsi Bali

---

## 1. Security Overview

### 1.1 Security Goals

| Goal | Description |
|------|-------------|
| **Confidentiality** | Data sensitif (credentials, alerts) hanya dapat diakses oleh yang berwenang |
| **Integrity** | Data monitoring tidak dapat dimanipulasi oleh pihak tidak berwenang |
| **Availability** | Sistem monitoring harus tersedia 24/7 untuk deteksi incident |
| **Non-repudiation** | Semua aksi tercatat dalam audit log |

### 1.2 Threat Model

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            THREAT LANDSCAPE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  EXTERNAL THREATS                       INTERNAL THREATS                    │
│  ─────────────────                      ─────────────────                   │
│  • Unauthorized access                  • Privilege abuse                   │
│  • Brute force attack                   • Data exfiltration                 │
│  • SQL injection                        • Credential sharing                │
│  • XSS attacks                          • Insider threats                   │
│  • CSRF attacks                                                             │
│  • DDoS attacks                         INFRASTRUCTURE THREATS              │
│  • Man-in-the-middle                    ─────────────────────               │
│  • API abuse                            • Server compromise                 │
│                                         • Container escape                  │
│  DATA THREATS                           • Misconfiguration                  │
│  ────────────                           • Outdated software                 │
│  • Data breach                                                              │
│  • Data tampering                                                           │
│  • Credential theft                                                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Authentication & Authorization

### 2.1 Authentication Mechanisms

#### Password Policy

```go
// Password requirements
type PasswordPolicy struct {
    MinLength        int  // Minimum 12 characters
    RequireUppercase bool // At least 1 uppercase
    RequireLowercase bool // At least 1 lowercase
    RequireDigit     bool // At least 1 digit
    RequireSpecial   bool // At least 1 special char
    MaxAge           int  // 90 days expiry
    HistoryCount     int  // Cannot reuse last 5 passwords
}

var DefaultPolicy = PasswordPolicy{
    MinLength:        12,
    RequireUppercase: true,
    RequireLowercase: true,
    RequireDigit:     true,
    RequireSpecial:   true,
    MaxAge:           90,
    HistoryCount:     5,
}
```

#### Password Hashing

```go
import "golang.org/x/crypto/bcrypt"

// Hash password with bcrypt (cost 12)
func HashPassword(password string) (string, error) {
    bytes, err := bcrypt.GenerateFromPassword([]byte(password), 12)
    return string(bytes), err
}

// Verify password
func CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

#### JWT Token Configuration

```go
// JWT settings
type JWTConfig struct {
    SecretKey      string        // 256-bit secret
    AccessExpiry   time.Duration // 15 minutes
    RefreshExpiry  time.Duration // 7 days
    Issuer         string        // "monitoring.diskominfos.baliprov.go.id"
    Algorithm      string        // HS256
}

// Token claims
type Claims struct {
    UserID   int64  `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    OpdID    *int64 `json:"opd_id,omitempty"`
    jwt.RegisteredClaims
}
```

#### Brute Force Protection

```go
// Login rate limiting
var loginLimiter = rate.NewLimiter(rate.Every(time.Minute), 5) // 5 attempts per minute

// Account lockout after failed attempts
const (
    MaxFailedAttempts = 5
    LockoutDuration   = 30 * time.Minute
)

type LoginAttempt struct {
    IP            string
    Username      string
    FailedCount   int
    LastAttempt   time.Time
    LockedUntil   time.Time
}
```

### 2.2 Authorization (RBAC)

```go
// Role definitions
type Role string

const (
    RoleSuperAdmin Role = "super_admin"
    RoleAdminOPD   Role = "admin_opd"
    RoleViewer     Role = "viewer"
)

// Permission matrix
var Permissions = map[Role][]string{
    RoleSuperAdmin: {
        "websites:*",      // Full access to websites
        "alerts:*",        // Full access to alerts
        "users:*",         // Full access to users
        "opd:*",           // Full access to OPD
        "settings:*",      // Full access to settings
        "keywords:*",      // Full access to keywords
        "reports:*",       // Full access to reports
    },
    RoleAdminOPD: {
        "websites:read:own",   // Read own OPD websites
        "websites:create:own", // Create for own OPD
        "websites:update:own", // Update own OPD websites
        "alerts:read:own",     // Read own OPD alerts
        "alerts:resolve:own",  // Resolve own OPD alerts
        "reports:read:own",    // View own OPD reports
    },
    RoleViewer: {
        "websites:read:own", // Read only
        "alerts:read:own",   // Read only
        "reports:read:own",  // Read only
    },
}
```

### 2.3 Session Management

```go
// Session configuration
type SessionConfig struct {
    MaxConcurrentSessions int           // Max 3 sessions per user
    SessionTimeout        time.Duration // 30 minutes inactivity
    AbsoluteTimeout       time.Duration // 8 hours max session
    SecureCookie          bool          // HTTPS only
    HttpOnly              bool          // No JavaScript access
    SameSite              string        // "Strict"
}
```

---

## 3. Data Protection

### 3.1 Data Classification

| Classification | Examples | Protection Level |
|----------------|----------|------------------|
| **Public** | Dashboard stats, uptime % | None |
| **Internal** | Website list, check history | Authentication required |
| **Confidential** | Alert details, scan results | Role-based access |
| **Restricted** | User credentials, API keys | Encryption + strict access |

### 3.2 Encryption

#### Data at Rest

```yaml
# Database encryption
MySQL:
  # Enable encryption at rest
  innodb_encrypt_tables = ON
  innodb_encrypt_log = ON
  innodb_encryption_threads = 4

# Sensitive field encryption in application
Encrypted Fields:
  - users.password_hash (bcrypt)
  - settings.telegram_bot_token (AES-256-GCM)
  - settings.api_keys (AES-256-GCM)
```

#### Data in Transit

```nginx
# TLS 1.2+ only
ssl_protocols TLSv1.2 TLSv1.3;
ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
ssl_prefer_server_ciphers off;

# HSTS
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;
```

### 3.3 Sensitive Data Handling

```go
// Mask sensitive data in logs
func MaskSensitive(data string) string {
    if len(data) <= 4 {
        return "****"
    }
    return data[:2] + strings.Repeat("*", len(data)-4) + data[len(data)-2:]
}

// Never log these fields
var SensitiveFields = []string{
    "password",
    "token",
    "api_key",
    "secret",
    "credential",
}
```

---

## 4. Input Validation & Sanitization

### 4.1 Input Validation Rules

```go
// URL validation
func ValidateURL(url string) error {
    parsed, err := url.Parse(url)
    if err != nil {
        return errors.New("invalid URL format")
    }

    // Must be HTTPS or HTTP
    if parsed.Scheme != "https" && parsed.Scheme != "http" {
        return errors.New("URL must use HTTP or HTTPS")
    }

    // Must have valid host
    if parsed.Host == "" {
        return errors.New("URL must have a valid host")
    }

    // Block private/internal IPs
    if isPrivateIP(parsed.Host) {
        return errors.New("private IP addresses not allowed")
    }

    return nil
}

// Keyword validation (prevent regex injection)
func ValidateKeyword(keyword string, isRegex bool) error {
    if len(keyword) > 255 {
        return errors.New("keyword too long")
    }

    if isRegex {
        _, err := regexp.Compile(keyword)
        if err != nil {
            return errors.New("invalid regex pattern")
        }
    }

    return nil
}
```

### 4.2 SQL Injection Prevention

```go
// ALWAYS use parameterized queries
// GOOD
func GetWebsite(db *sqlx.DB, id int64) (*Website, error) {
    var website Website
    err := db.Get(&website, "SELECT * FROM websites WHERE id = ?", id)
    return &website, err
}

// BAD - Never do this!
// query := fmt.Sprintf("SELECT * FROM websites WHERE id = %d", id)
```

### 4.3 XSS Prevention

```go
// Sanitize HTML in content scan results
import "github.com/microcosm-cc/bluemonday"

var strictPolicy = bluemonday.StrictPolicy()

func SanitizeHTML(input string) string {
    return strictPolicy.Sanitize(input)
}

// In templates, always escape output
// {{ .UnsafeContent | html }}
```

### 4.4 CSRF Protection

```go
// Generate CSRF token
func GenerateCSRFToken() string {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.URLEncoding.EncodeToString(b)
}

// CSRF middleware
func CSRFMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.Method == "GET" || c.Request.Method == "HEAD" {
            c.Next()
            return
        }

        token := c.GetHeader("X-CSRF-Token")
        sessionToken := getSessionCSRF(c)

        if !subtle.ConstantTimeCompare([]byte(token), []byte(sessionToken)) {
            c.AbortWithStatusJSON(403, gin.H{"error": "CSRF token mismatch"})
            return
        }

        c.Next()
    }
}
```

---

## 5. API Security

### 5.1 Rate Limiting

```go
// Rate limit configuration per endpoint
var RateLimits = map[string]rate.Limit{
    "/api/v1/auth/login":    rate.Every(12 * time.Second), // 5/min
    "/api/v1/websites":      rate.Every(600 * time.Millisecond), // 100/min
    "/api/v1/websites/*/check": rate.Every(12 * time.Second), // 5/min per website
    "default":               rate.Every(1 * time.Second), // 60/min
}
```

### 5.2 Request Validation

```go
// Validate request size
const MaxRequestSize = 1 << 20 // 1 MB

// Validate content type
func ValidateContentType(c *gin.Context) error {
    ct := c.ContentType()
    if ct != "application/json" {
        return errors.New("content-type must be application/json")
    }
    return nil
}
```

### 5.3 Security Headers

```go
func SecurityHeaders() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
        c.Next()
    }
}
```

---

## 6. Logging & Audit Trail

### 6.1 Security Event Logging

```go
// Events to log
type SecurityEvent string

const (
    EventLoginSuccess     SecurityEvent = "LOGIN_SUCCESS"
    EventLoginFailed      SecurityEvent = "LOGIN_FAILED"
    EventLogout           SecurityEvent = "LOGOUT"
    EventPasswordChanged  SecurityEvent = "PASSWORD_CHANGED"
    EventUserCreated      SecurityEvent = "USER_CREATED"
    EventUserDeleted      SecurityEvent = "USER_DELETED"
    EventPermissionDenied SecurityEvent = "PERMISSION_DENIED"
    EventSuspiciousActivity SecurityEvent = "SUSPICIOUS_ACTIVITY"
)

// Log structure
type AuditLog struct {
    ID          int64         `json:"id"`
    Timestamp   time.Time     `json:"timestamp"`
    Event       SecurityEvent `json:"event"`
    UserID      *int64        `json:"user_id"`
    Username    string        `json:"username"`
    IPAddress   string        `json:"ip_address"`
    UserAgent   string        `json:"user_agent"`
    Resource    string        `json:"resource"`
    Action      string        `json:"action"`
    Status      string        `json:"status"`
    Details     interface{}   `json:"details"`
}
```

### 6.2 Log Protection

```go
// Log sanitization
func SanitizeLogEntry(entry map[string]interface{}) map[string]interface{} {
    sensitiveKeys := []string{"password", "token", "secret", "key", "credential"}

    for key := range entry {
        for _, sensitive := range sensitiveKeys {
            if strings.Contains(strings.ToLower(key), sensitive) {
                entry[key] = "[REDACTED]"
            }
        }
    }
    return entry
}
```

---

## 7. Infrastructure Security

### 7.1 Container Security

```dockerfile
# Security best practices in Dockerfile

# Use specific version, not latest
FROM golang:1.21-alpine AS builder

# Run as non-root user
RUN adduser -D -g '' appuser
USER appuser

# Don't expose unnecessary ports
EXPOSE 8080

# Use read-only filesystem where possible
# docker-compose: read_only: true

# Limit resources
# docker-compose:
#   deploy:
#     resources:
#       limits:
#         cpus: '1'
#         memory: 1G
```

### 7.2 Network Security

```yaml
# docker-compose network isolation
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true  # No external access

services:
  nginx:
    networks:
      - frontend
      - backend

  web:
    networks:
      - backend  # Only internal network

  mysql:
    networks:
      - backend  # Only internal network
```

### 7.3 Secrets Management

```yaml
# Use Docker secrets (production)
secrets:
  db_password:
    external: true
  telegram_token:
    external: true

services:
  web:
    secrets:
      - db_password
      - telegram_token
    environment:
      - DB_PASSWORD_FILE=/run/secrets/db_password
```

---

## 8. Incident Response

### 8.1 Security Incident Categories

| Severity | Category | Example | Response Time |
|----------|----------|---------|---------------|
| P1 - Critical | Data breach | Database compromised | < 1 hour |
| P1 - Critical | System compromise | Server hacked | < 1 hour |
| P2 - High | Unauthorized access | Multiple failed logins | < 4 hours |
| P3 - Medium | Vulnerability found | Outdated package | < 24 hours |
| P4 - Low | Policy violation | Weak password | < 1 week |

### 8.2 Incident Response Procedure

```
┌─────────────────────────────────────────────────────────────────┐
│                    INCIDENT RESPONSE FLOW                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. DETECTION                                                    │
│     ├── Automated alert                                          │
│     ├── User report                                              │
│     └── Audit log review                                         │
│                    │                                             │
│                    ▼                                             │
│  2. CONTAINMENT                                                  │
│     ├── Isolate affected systems                                 │
│     ├── Block malicious IPs                                      │
│     ├── Disable compromised accounts                             │
│     └── Preserve evidence                                        │
│                    │                                             │
│                    ▼                                             │
│  3. ERADICATION                                                  │
│     ├── Remove malware/backdoors                                 │
│     ├── Patch vulnerabilities                                    │
│     ├── Reset credentials                                        │
│     └── Update security controls                                 │
│                    │                                             │
│                    ▼                                             │
│  4. RECOVERY                                                     │
│     ├── Restore from clean backup                                │
│     ├── Verify system integrity                                  │
│     ├── Monitor for recurrence                                   │
│     └── Gradual service restoration                              │
│                    │                                             │
│                    ▼                                             │
│  5. POST-INCIDENT                                                │
│     ├── Document incident                                        │
│     ├── Root cause analysis                                      │
│     ├── Update procedures                                        │
│     └── Security awareness training                              │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 8.3 Emergency Contacts

```yaml
# Incident response contacts
contacts:
  security_team:
    - name: "Tim Persandian"
      phone: "+62-xxx-xxx-xxxx"
      email: "persandian@diskominfos.baliprov.go.id"

  escalation:
    - level: 1
      name: "On-Call Admin"
      response_time: "15 minutes"
    - level: 2
      name: "Security Lead"
      response_time: "30 minutes"
    - level: 3
      name: "Kepala Bidang"
      response_time: "1 hour"
```

---

## 9. Compliance Checklist

### 9.1 Security Controls Checklist

```
Authentication & Access Control:
☐ Strong password policy enforced
☐ Multi-factor authentication (optional/future)
☐ Account lockout after failed attempts
☐ Session timeout implemented
☐ Role-based access control
☐ Principle of least privilege

Data Protection:
☐ TLS 1.2+ for all connections
☐ Sensitive data encrypted at rest
☐ Database access restricted
☐ Backup encryption enabled
☐ Data retention policy implemented

Application Security:
☐ Input validation on all endpoints
☐ Output encoding/escaping
☐ CSRF protection
☐ Security headers configured
☐ Rate limiting implemented
☐ Error messages don't leak info

Infrastructure Security:
☐ Firewall configured
☐ Non-root container users
☐ Network segmentation
☐ Regular security updates
☐ Vulnerability scanning

Logging & Monitoring:
☐ Security events logged
☐ Audit trail maintained
☐ Log integrity protected
☐ Alerting configured
☐ Log retention policy

Incident Response:
☐ Incident response plan documented
☐ Contact list maintained
☐ Regular backup tested
☐ Recovery procedures documented
```

---

## 10. Security Maintenance

### 10.1 Regular Security Tasks

| Task | Frequency | Responsible |
|------|-----------|-------------|
| Security patches | Weekly | DevOps |
| Dependency updates | Monthly | Developer |
| Access review | Quarterly | Security |
| Penetration testing | Annually | External |
| Password rotation (service accounts) | Quarterly | DevOps |
| SSL certificate renewal | Before expiry | DevOps |
| Backup restoration test | Monthly | DevOps |
| Security awareness training | Annually | HR/Security |

### 10.2 Security Scanning

```bash
# Dependency vulnerability scanning
go list -json -m all | nancy sleuth

# Container image scanning
trivy image monitoring-website:latest

# SAST (Static Application Security Testing)
gosec ./...

# Docker bench security
docker run --rm -it --net host --pid host --userns host --cap-add audit_control \
    -v /var/lib:/var/lib -v /var/run/docker.sock:/var/run/docker.sock \
    docker/docker-bench-security
```
