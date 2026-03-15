# Monitoring Website - Diskominfos Provinsi Bali

Aplikasi monitoring website untuk memantau seluruh website Pemerintah Provinsi Bali (domain baliprov.go.id).

## Fitur Utama

- **Uptime Monitoring**: Pengecekan status website secara berkala (setiap 5 menit)
- **SSL Monitoring**: Pemantauan validitas dan masa berlaku sertifikat SSL
- **Content Scanning**: Deteksi konten mencurigakan (judi online/judol, defacement)
- **Alert System**: Notifikasi real-time melalui Telegram
- **Dashboard**: Antarmuka web untuk monitoring terpusat

## Tech Stack

- **Backend**: Go (Gin Framework)
- **Database**: MySQL 8.0
- **Notification**: Telegram Bot API
- **Frontend**: HTML + TailwindCSS + JavaScript
- **Deployment**: Docker & Docker Compose

## Quick Start

### Prerequisites

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0 (jika tidak pakai Docker)

### Development

1. Clone repository:
```bash
git clone https://github.com/diskominfos-bali/monitoring-website.git
cd monitoring-website
```

2. Copy dan edit konfigurasi:
```bash
cp config.yaml.example config.yaml
cp .env.example .env
# Edit kedua file sesuai kebutuhan
```

3. Jalankan dengan Docker:
```bash
docker-compose up -d
```

4. Atau jalankan manual:
```bash
# Install dependencies
go mod download

# Run migrations
mysql -u root -p monitoring_website < migrations/001_initial_schema.sql

# Run application
go run cmd/server/main.go
```

5. Akses aplikasi di http://localhost:8080

### Membuat Admin User Pertama

Gunakan script untuk membuat user admin pertama:
```bash
# Via API (setelah aplikasi berjalan)
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@baliprov.go.id",
    "password": "SecurePassword123!",
    "full_name": "Administrator"
  }'
```

### Konfigurasi Telegram Bot

1. Buat bot baru via [@BotFather](https://t.me/botfather)
2. Dapatkan token bot
3. Tambahkan bot ke grup/channel
4. Dapatkan chat ID grup/channel
5. Update konfigurasi di `.env` atau `config.yaml`

## Struktur Direktori

```
.
├── cmd/
│   └── server/           # Entry point aplikasi
├── internal/
│   ├── config/           # Konfigurasi
│   ├── domain/           # Domain models
│   ├── handler/          # HTTP handlers
│   ├── repository/       # Database repository
│   ├── scheduler/        # Cron jobs
│   └── service/          # Business logic
├── migrations/           # Database migrations
├── pkg/                  # Shared packages
├── web/
│   ├── static/          # Static files
│   └── templates/       # HTML templates
├── docs/                 # Dokumentasi
├── docker-compose.yml
├── Dockerfile
└── config.yaml.example
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `GET /api/auth/me` - Get current user
- `PUT /api/auth/password` - Change password

### Dashboard
- `GET /api/dashboard` - Dashboard overview
- `GET /api/dashboard/stats` - Statistics

### Websites
- `GET /api/websites` - List websites
- `POST /api/websites` - Create website
- `GET /api/websites/:id` - Get website detail
- `PUT /api/websites/:id` - Update website
- `DELETE /api/websites/:id` - Delete website
- `POST /api/websites/bulk` - Bulk import

### Alerts
- `GET /api/alerts` - List alerts
- `GET /api/alerts/active` - Active alerts
- `POST /api/alerts/:id/acknowledge` - Acknowledge alert
- `POST /api/alerts/:id/resolve` - Resolve alert

## Konfigurasi

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_HOST` | Server host | `0.0.0.0` |
| `SERVER_PORT` | Server port | `8080` |
| `DATABASE_HOST` | MySQL host | `localhost` |
| `DATABASE_PORT` | MySQL port | `3306` |
| `DATABASE_USER` | MySQL user | `root` |
| `DATABASE_PASSWORD` | MySQL password | - |
| `DATABASE_NAME` | MySQL database | `monitoring_website` |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | - |
| `TELEGRAM_CHAT_IDS` | Telegram chat IDs (comma-separated) | - |
| `JWT_SECRET_KEY` | JWT secret key | - |

### Scheduler (Cron)

| Job | Default Schedule | Description |
|-----|-----------------|-------------|
| Uptime Check | Every 5 minutes | Check website availability |
| SSL Check | Every 6 hours | Check SSL certificates |
| Content Scan | Every hour | Scan for suspicious content |
| Daily Summary | 8 AM daily | Send daily report |

## Dokumentasi Lengkap

Lihat folder `docs/` untuk dokumentasi lengkap:
- [System Requirements](docs/01-SYSTEM-REQUIREMENTS.md)
- [Architecture](docs/02-ARCHITECTURE.md)
- [Database Schema](docs/03-DATABASE-SCHEMA.md)
- [API Specification](docs/04-API-SPECIFICATION.md)
- [UI/UX Wireframe](docs/05-UI-UX-WIREFRAME.md)
- [Infrastructure](docs/06-INFRASTRUCTURE-DEPLOYMENT.md)
- [Security](docs/07-SECURITY-CONSIDERATIONS.md)

## License

Copyright © 2024 Dinas Komunikasi, Informatika dan Statistik Provinsi Bali

## Support

Untuk bantuan dan pertanyaan, hubungi:
- Email: diskominfos@baliprov.go.id
