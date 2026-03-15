# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Website monitoring application for Pemerintah Provinsi Bali (baliprov.go.id). Monitors uptime, SSL certificates, content (gambling/defacement detection), security headers, vulnerabilities, and Google dork patterns. Sends alerts via Telegram, email, and webhooks.

## Commands

```bash
# Development (hot reload via Air)
make dev

# Build & run
make build          # builds binary: ./monitoring-website
make run            # build + run

# Tests
make test           # go test -v ./...

# Docker
docker-compose up -d        # start all services
docker-compose down         # stop
make docker-rebuild         # full rebuild

# Dependencies
make deps           # go mod download + tidy

# Database migrations (manual, sequential SQL files)
mysql -u root -p monitoring_website < migrations/001_initial_schema.sql
```

## Architecture

Go application using **clean architecture** with Gin HTTP framework and MySQL (via sqlx).

### Layer structure

- **`cmd/server/main.go`** — Entry point. Wires all dependencies manually (no DI container). Creates repos → services → handlers → router → scheduler, then starts HTTP server with graceful shutdown.
- **`internal/config/`** — Viper-based config from `config.yaml` with env var overrides (`VIPER_KEY_REPLACER: . → _`).
- **`internal/domain/`** — Domain models/structs (website, alert, check, user, escalation, dork, vulnerability, etc.). No business logic.
- **`internal/repository/mysql/`** — All database access. One repo per domain entity (`website_repo.go`, `alert_repo.go`, `check_repo.go`, etc.).
- **`internal/service/`** — Business logic, organized by feature:
  - `monitor/` — uptime, SSL, content scanner, security headers, vulnerability scanner, dork scanner
  - `notifier/` — telegram, email, webhook notification senders
  - `auth/` — JWT authentication
  - `website/` — website CRUD + OPD management
  - `alert/`, `summary/`, `report/`, `escalation/`, `cleanup/`, `settings/`
- **`internal/handler/`** — HTTP handlers (one per feature) + `router.go` (all route definitions)
- **`internal/handler/middleware/`** — Auth (JWT) and CORS middleware
- **`internal/scheduler/`** — Cron-based scheduler (robfig/cron) for periodic monitoring tasks
- **`pkg/logger/`** — Zerolog-based structured logger
- **`web/`** — Static files and HTML templates (TailwindCSS + vanilla JS frontend)

### Key patterns

- All dependencies are injected via constructors in `cmd/server/main.go`
- Cron expressions use 6-field format (with seconds): `"0 */5 * * * *"`
- Config loads from `config.yaml` (gitignored) with fallback to env vars and defaults
- Migrations are plain SQL files in `migrations/`, applied manually in order
- Public status page routes (`/status/*`) require no auth; all `/api/*` routes (except login) require JWT

### Configuration

Two config sources: `config.yaml` (primary) and `.env` (for Docker). Copy from `*.example` files. Key sections: server, database, telegram, email, webhook, jwt, monitoring, scheduler, keywords.
