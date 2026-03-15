# Dokumen Arsitektur Aplikasi

## Monitoring Website Pemerintah Provinsi Bali

---

## 1. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              EXTERNAL LAYER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐                    │
│   │   Browser   │    │  Telegram   │    │  Target     │                    │
│   │   (Admin)   │    │   Users     │    │  Websites   │                    │
│   └──────┬──────┘    └──────┬──────┘    └──────┬──────┘                    │
│          │                  │                  │                            │
└──────────┼──────────────────┼──────────────────┼────────────────────────────┘
           │                  │                  │
           ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            APPLICATION LAYER                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────┐         ┌─────────────────────┐                  │
│   │    WEB SERVER       │         │    WORKER SERVICE   │                  │
│   │    (Gin HTTP)       │         │    (Background)     │                  │
│   │                     │         │                     │                  │
│   │  ┌───────────────┐  │         │  ┌───────────────┐  │                  │
│   │  │  REST API     │  │         │  │  Scheduler    │  │                  │
│   │  │  Handlers     │  │         │  │  (Cron)       │  │                  │
│   │  └───────────────┘  │         │  └───────────────┘  │                  │
│   │  ┌───────────────┐  │         │  ┌───────────────┐  │                  │
│   │  │  Dashboard    │  │         │  │  Monitor      │  │                  │
│   │  │  (HTML/JS)    │  │         │  │  Workers      │  │                  │
│   │  └───────────────┘  │         │  └───────────────┘  │                  │
│   │  ┌───────────────┐  │         │  ┌───────────────┐  │                  │
│   │  │  Auth/RBAC    │  │         │  │  Notifier     │  │                  │
│   │  └───────────────┘  │         │  └───────────────┘  │                  │
│   └─────────────────────┘         └─────────────────────┘                  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
           │                                 │
           ▼                                 ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              SERVICE LAYER                                   │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │   Website    │  │   Uptime     │  │   Content    │  │    SSL       │   │
│   │   Service    │  │   Monitor    │  │   Scanner    │  │   Checker    │   │
│   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                                              │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │  Telegram    │  │   Report     │  │    User      │  │    Alert     │   │
│   │  Notifier    │  │   Generator  │  │   Service    │  │   Service    │   │
│   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            REPOSITORY LAYER                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│   │   Website    │  │    Check     │  │    Alert     │  │    User      │   │
│   │    Repo      │  │    Repo      │  │    Repo      │  │    Repo      │   │
│   └──────────────┘  └──────────────┘  └──────────────┘  └──────────────┘   │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                              DATA LAYER                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│   ┌─────────────────────┐              ┌─────────────────────┐              │
│   │       MySQL         │              │       Redis         │              │
│   │   (Primary Data)    │              │   (Cache/Queue)     │              │
│   │                     │              │                     │              │
│   │  - websites         │              │  - session cache    │              │
│   │  - checks           │              │  - rate limiting    │              │
│   │  - alerts           │              │  - job queue        │              │
│   │  - users            │              │                     │              │
│   │  - audit_logs       │              │                     │              │
│   └─────────────────────┘              └─────────────────────┘              │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Layered Architecture Detail

### 2.1 Project Structure

```
monitoring-website/
├── cmd/
│   ├── server/           # HTTP server entrypoint
│   │   └── main.go
│   └── worker/           # Background worker entrypoint
│       └── main.go
│
├── internal/
│   ├── config/           # Configuration management
│   │   └── config.go
│   │
│   ├── domain/           # Domain/Entity layer (Core Business)
│   │   ├── website.go
│   │   ├── check.go
│   │   ├── alert.go
│   │   └── user.go
│   │
│   ├── repository/       # Data access layer
│   │   ├── interface.go
│   │   ├── mysql/
│   │   │   ├── website_repo.go
│   │   │   ├── check_repo.go
│   │   │   ├── alert_repo.go
│   │   │   └── user_repo.go
│   │   └── redis/
│   │       └── cache_repo.go
│   │
│   ├── service/          # Business logic layer
│   │   ├── website_service.go
│   │   ├── monitor/
│   │   │   ├── uptime_monitor.go
│   │   │   ├── ssl_checker.go
│   │   │   └── content_scanner.go
│   │   ├── notifier/
│   │   │   ├── interface.go
│   │   │   └── telegram.go
│   │   ├── alert_service.go
│   │   ├── user_service.go
│   │   └── report_service.go
│   │
│   ├── handler/          # HTTP handlers (Controller layer)
│   │   ├── website_handler.go
│   │   ├── dashboard_handler.go
│   │   ├── auth_handler.go
│   │   ├── alert_handler.go
│   │   └── api/
│   │       └── v1/
│   │           └── routes.go
│   │
│   ├── middleware/       # HTTP middlewares
│   │   ├── auth.go
│   │   ├── cors.go
│   │   └── logging.go
│   │
│   ├── scheduler/        # Cron job scheduler
│   │   └── scheduler.go
│   │
│   └── worker/           # Background workers
│       ├── monitor_worker.go
│       └── alert_worker.go
│
├── pkg/                  # Shared/reusable packages
│   ├── httpclient/       # HTTP client wrapper
│   │   └── client.go
│   ├── validator/        # Input validation
│   │   └── validator.go
│   └── logger/           # Logging utility
│       └── logger.go
│
├── web/                  # Frontend assets
│   ├── templates/        # Go HTML templates
│   │   ├── layouts/
│   │   │   └── base.html
│   │   ├── dashboard.html
│   │   ├── websites.html
│   │   └── alerts.html
│   └── static/
│       ├── css/
│       ├── js/
│       └── img/
│
├── migrations/           # Database migrations
│   ├── 001_create_websites.sql
│   ├── 002_create_checks.sql
│   └── ...
│
├── scripts/              # Utility scripts
│   ├── setup.sh
│   └── seed.sh
│
├── config.yaml           # Configuration file
├── go.mod
├── go.sum
├── Dockerfile
├── docker-compose.yml
└── Makefile
```

---

## 3. Component Details

### 3.1 Domain Layer (Entities)

Domain layer berisi business entities murni tanpa dependency ke infrastructure.

```go
// domain/website.go
type Website struct {
    ID          int64
    URL         string
    Name        string
    Description string
    OPD         string       // Organisasi Perangkat Daerah
    IsActive    bool
    Status      WebsiteStatus
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type WebsiteStatus string
const (
    StatusUp       WebsiteStatus = "up"
    StatusDown     WebsiteStatus = "down"
    StatusDegraded WebsiteStatus = "degraded"
    StatusUnknown  WebsiteStatus = "unknown"
)
```

### 3.2 Repository Layer

Repository layer mengabstraksi data access. Menggunakan interface agar mudah di-mock untuk testing.

```go
// repository/interface.go
type WebsiteRepository interface {
    Create(ctx context.Context, website *domain.Website) error
    GetByID(ctx context.Context, id int64) (*domain.Website, error)
    GetAll(ctx context.Context, filter WebsiteFilter) ([]domain.Website, error)
    Update(ctx context.Context, website *domain.Website) error
    Delete(ctx context.Context, id int64) error
    GetActiveWebsites(ctx context.Context) ([]domain.Website, error)
}

type CheckRepository interface {
    Create(ctx context.Context, check *domain.Check) error
    GetByWebsiteID(ctx context.Context, websiteID int64, limit int) ([]domain.Check, error)
    GetLatestByWebsiteID(ctx context.Context, websiteID int64) (*domain.Check, error)
}
```

### 3.3 Service Layer

Service layer berisi business logic. Orchestrate repository calls dan implement business rules.

```go
// service/monitor/uptime_monitor.go
type UptimeMonitor struct {
    httpClient  *httpclient.Client
    websiteRepo repository.WebsiteRepository
    checkRepo   repository.CheckRepository
    alertSvc    *AlertService
}

func (m *UptimeMonitor) CheckWebsite(ctx context.Context, website *domain.Website) (*domain.Check, error) {
    // 1. Make HTTP request
    // 2. Record response time, status code
    // 3. Save check result
    // 4. Trigger alert if needed
}
```

### 3.4 Handler Layer

Handler layer menerima HTTP request dan memanggil service layer.

```go
// handler/website_handler.go
type WebsiteHandler struct {
    websiteSvc *service.WebsiteService
}

func (h *WebsiteHandler) GetAll(c *gin.Context) {
    websites, err := h.websiteSvc.GetAll(c.Request.Context())
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    c.JSON(200, websites)
}
```

---

## 4. Data Flow Diagrams

### 4.1 Uptime Monitoring Flow

```
┌─────────┐     ┌───────────┐     ┌─────────────┐     ┌──────────┐
│Scheduler│────▶│  Monitor  │────▶│   Target    │────▶│  Record  │
│  Cron   │     │  Worker   │     │   Website   │     │  Result  │
└─────────┘     └───────────┘     └─────────────┘     └──────────┘
                     │                                      │
                     │         ┌─────────────┐              │
                     │         │   Check     │◀─────────────┘
                     │         │   Status    │
                     │         └──────┬──────┘
                     │                │
                     │          Is Status Changed?
                     │                │
                     │         ┌──────┴──────┐
                     │         │             │
                     │        YES           NO
                     │         │             │
                     │         ▼             ▼
                     │   ┌──────────┐   ┌──────────┐
                     └──▶│  Create  │   │   Done   │
                         │  Alert   │   └──────────┘
                         └────┬─────┘
                              │
                              ▼
                        ┌──────────┐
                        │ Telegram │
                        │   Bot    │
                        └──────────┘
```

### 4.2 Content Scanning Flow

```
┌─────────┐     ┌───────────┐     ┌─────────────┐
│Scheduler│────▶│  Content  │────▶│   Fetch     │
│  Cron   │     │  Scanner  │     │   HTML      │
└─────────┘     └───────────┘     └──────┬──────┘
                                         │
                                         ▼
                              ┌─────────────────────┐
                              │   Parse & Analyze   │
                              │                     │
                              │ - Check keywords    │
                              │ - Check iframes     │
                              │ - Check redirects   │
                              │ - Check meta tags   │
                              └──────────┬──────────┘
                                         │
                                   Found Issue?
                                         │
                              ┌──────────┴──────────┐
                              │                     │
                             YES                   NO
                              │                     │
                              ▼                     ▼
                        ┌──────────┐          ┌──────────┐
                        │  Create  │          │  Update  │
                        │  Alert   │          │  Status  │
                        │(CRITICAL)│          │  (Clean) │
                        └────┬─────┘          └──────────┘
                             │
                             ▼
                       ┌──────────┐
                       │ Telegram │
                       │   Bot    │
                       └──────────┘
```

---

## 5. Technology Stack

### 5.1 Backend

| Component | Technology | Justification |
|-----------|------------|---------------|
| Language | Go 1.21+ | High performance, excellent concurrency |
| Web Framework | Gin | Fast, lightweight, production-ready |
| ORM/Database | sqlx | Lightweight, raw SQL control |
| Config | Viper | Flexible configuration management |
| Scheduler | robfig/cron | Reliable cron scheduling |
| HTTP Client | net/http | Standard library, customizable |
| HTML Parser | goquery | jQuery-like HTML parsing |

### 5.2 Database

| Component | Technology | Justification |
|-----------|------------|---------------|
| Primary DB | MySQL 8.0 | Reliable, familiar, good tooling |
| Cache | Redis | Fast caching, rate limiting |

### 5.3 Frontend

| Component | Technology | Justification |
|-----------|------------|---------------|
| Template | Go HTML/Template | Simple, server-side rendering |
| CSS | Tailwind CSS | Utility-first, fast development |
| JS | Alpine.js | Lightweight reactivity |
| Charts | Chart.js | Simple, beautiful charts |

### 5.4 Infrastructure

| Component | Technology | Justification |
|-----------|------------|---------------|
| Container | Docker | Consistent deployment |
| Orchestration | Docker Compose | Simple multi-container setup |
| Reverse Proxy | Nginx | SSL termination, load balancing |

---

## 6. Communication Patterns

### 6.1 Synchronous (HTTP)
- Dashboard requests
- API calls
- CRUD operations

### 6.2 Asynchronous (Background Workers)
- Website monitoring checks
- Content scanning
- Alert notifications
- Report generation

---

## 7. Error Handling Strategy

```go
// Custom error types
type AppError struct {
    Code    string
    Message string
    Err     error
}

var (
    ErrNotFound      = &AppError{Code: "NOT_FOUND", Message: "Resource not found"}
    ErrUnauthorized  = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized access"}
    ErrValidation    = &AppError{Code: "VALIDATION_ERROR", Message: "Validation failed"}
    ErrInternal      = &AppError{Code: "INTERNAL_ERROR", Message: "Internal server error"}
)
```

---

## 8. Dependency Injection

Menggunakan manual dependency injection untuk simplicity:

```go
func main() {
    // Initialize config
    cfg := config.Load()

    // Initialize database
    db := database.NewMySQL(cfg.Database)

    // Initialize repositories
    websiteRepo := mysql.NewWebsiteRepository(db)
    checkRepo := mysql.NewCheckRepository(db)

    // Initialize services
    websiteSvc := service.NewWebsiteService(websiteRepo)
    monitorSvc := monitor.NewUptimeMonitor(websiteRepo, checkRepo)

    // Initialize handlers
    websiteHandler := handler.NewWebsiteHandler(websiteSvc)

    // Setup routes
    router := gin.Default()
    api.SetupRoutes(router, websiteHandler)

    router.Run()
}
```

---

## 9. Concurrency Model

### 9.1 Worker Pool Pattern

```go
type MonitorWorkerPool struct {
    workers    int
    jobQueue   chan *domain.Website
    resultChan chan *domain.Check
}

func (p *MonitorWorkerPool) Start(ctx context.Context) {
    for i := 0; i < p.workers; i++ {
        go p.worker(ctx, i)
    }
}

func (p *MonitorWorkerPool) worker(ctx context.Context, id int) {
    for {
        select {
        case website := <-p.jobQueue:
            result := p.checkWebsite(website)
            p.resultChan <- result
        case <-ctx.Done():
            return
        }
    }
}
```

### 9.2 Rate Limiting

```go
// Limit requests to target websites
rateLimiter := rate.NewLimiter(rate.Every(time.Second), 10) // 10 req/sec
```

---

## 10. Monitoring & Observability

### 10.1 Logging
- Structured logging dengan zerolog
- Log levels: DEBUG, INFO, WARN, ERROR
- Request logging middleware

### 10.2 Metrics (Optional)
- Prometheus metrics endpoint
- Custom metrics: check duration, alert count, etc.

### 10.3 Health Check
```
GET /health
{
    "status": "healthy",
    "database": "connected",
    "redis": "connected",
    "version": "1.0.0"
}
```
