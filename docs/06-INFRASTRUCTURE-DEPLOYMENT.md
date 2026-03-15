# Dokumen Infrastructure & Deployment

## Monitoring Website Pemerintah Provinsi Bali

---

## 1. Infrastructure Overview

### 1.1 Architecture Diagram

```
                                    INTERNET
                                        │
                                        ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                              FIREWALL / WAF                                    │
│                           (Optional: Cloudflare)                               │
└───────────────────────────────────────┬───────────────────────────────────────┘
                                        │
                                        ▼
┌───────────────────────────────────────────────────────────────────────────────┐
│                              REVERSE PROXY                                     │
│                                 (Nginx)                                        │
│                                                                                │
│   • SSL Termination                                                            │
│   • Load Balancing                                                             │
│   • Static File Serving                                                        │
│   • Rate Limiting                                                              │
└───────────────────────────────────────┬───────────────────────────────────────┘
                                        │
            ┌───────────────────────────┼───────────────────────────┐
            │                           │                           │
            ▼                           ▼                           ▼
┌───────────────────┐      ┌───────────────────┐      ┌───────────────────┐
│    WEB SERVER     │      │    WEB SERVER     │      │   WORKER SERVICE  │
│   (Container 1)   │      │   (Container 2)   │      │    (Container)    │
│                   │      │                   │      │                   │
│  • REST API       │      │  • REST API       │      │  • Scheduler      │
│  • Dashboard      │      │  • Dashboard      │      │  • Monitor Jobs   │
│  • Auth           │      │  • Auth           │      │  • Notifier       │
│                   │      │                   │      │                   │
│  Port: 8080       │      │  Port: 8081       │      │  No exposed port  │
└─────────┬─────────┘      └─────────┬─────────┘      └─────────┬─────────┘
          │                          │                          │
          └──────────────────────────┼──────────────────────────┘
                                     │
            ┌────────────────────────┼────────────────────────┐
            │                        │                        │
            ▼                        ▼                        ▼
┌───────────────────┐      ┌───────────────────┐      ┌───────────────────┐
│      MySQL        │      │      Redis        │      │   Telegram API    │
│   (Container)     │      │   (Container)     │      │   (External)      │
│                   │      │                   │      │                   │
│  • Primary Data   │      │  • Cache          │      │  • Notifications  │
│  • Port: 3306     │      │  • Job Queue      │      │                   │
│                   │      │  • Rate Limit     │      │                   │
│                   │      │  • Port: 6379     │      │                   │
└───────────────────┘      └───────────────────┘      └───────────────────┘
```

### 1.2 Minimum Server Requirements

#### Single Server Setup (Small Scale: < 100 websites)

| Resource | Minimum | Recommended |
|----------|---------|-------------|
| CPU | 2 vCPU | 4 vCPU |
| RAM | 4 GB | 8 GB |
| Storage | 50 GB SSD | 100 GB SSD |
| Network | 100 Mbps | 1 Gbps |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

#### Multi-Server Setup (Large Scale: 100+ websites)

| Server | Specs | Purpose |
|--------|-------|---------|
| App Server 1 | 4 vCPU, 8GB RAM | Web Server |
| App Server 2 | 4 vCPU, 8GB RAM | Web Server |
| Worker Server | 4 vCPU, 8GB RAM | Background Jobs |
| DB Server | 4 vCPU, 16GB RAM | MySQL |
| Cache Server | 2 vCPU, 4GB RAM | Redis |

---

## 2. Docker Configuration

### 2.1 Dockerfile (Application)

```dockerfile
# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./cmd/worker

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Set timezone
ENV TZ=Asia/Makassar

# Copy binaries from builder
COPY --from=builder /app/server .
COPY --from=builder /app/worker .
COPY --from=builder /app/web ./web
COPY --from=builder /app/migrations ./migrations

# Create non-root user
RUN adduser -D -g '' appuser
USER appuser

EXPOSE 8080

CMD ["./server"]
```

### 2.2 Docker Compose

```yaml
version: '3.8'

services:
  # ===================
  # Application Services
  # ===================

  web:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: monitoring-web
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - APP_ENV=production
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=monitoring_website
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./logs:/app/logs
    networks:
      - monitoring-network
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s

  worker:
    build:
      context: .
      dockerfile: Dockerfile
    container_name: monitoring-worker
    restart: unless-stopped
    command: ["./worker"]
    environment:
      - APP_ENV=production
      - DB_HOST=mysql
      - DB_PORT=3306
      - DB_USER=${DB_USER}
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=monitoring_website
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - ./logs:/app/logs
    networks:
      - monitoring-network

  # ===================
  # Database Services
  # ===================

  mysql:
    image: mysql:8.0
    container_name: monitoring-mysql
    restart: unless-stopped
    environment:
      - MYSQL_ROOT_PASSWORD=${DB_ROOT_PASSWORD}
      - MYSQL_DATABASE=monitoring_website
      - MYSQL_USER=${DB_USER}
      - MYSQL_PASSWORD=${DB_PASSWORD}
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
      - ./migrations:/docker-entrypoint-initdb.d:ro
      - ./mysql.cnf:/etc/mysql/conf.d/custom.cnf:ro
    networks:
      - monitoring-network
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "root", "-p${DB_ROOT_PASSWORD}"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s

  redis:
    image: redis:7-alpine
    container_name: monitoring-redis
    restart: unless-stopped
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    networks:
      - monitoring-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  # ===================
  # Reverse Proxy
  # ===================

  nginx:
    image: nginx:alpine
    container_name: monitoring-nginx
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - ./web/static:/var/www/static:ro
      - certbot_www:/var/www/certbot:ro
    depends_on:
      - web
    networks:
      - monitoring-network

  # ===================
  # Monitoring (Optional)
  # ===================

  # prometheus:
  #   image: prom/prometheus:latest
  #   container_name: monitoring-prometheus
  #   volumes:
  #     - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro
  #     - prometheus_data:/prometheus
  #   ports:
  #     - "9090:9090"
  #   networks:
  #     - monitoring-network

volumes:
  mysql_data:
  redis_data:
  certbot_www:
  # prometheus_data:

networks:
  monitoring-network:
    driver: bridge
```

### 2.3 Nginx Configuration

```nginx
# /nginx/nginx.conf

user nginx;
worker_processes auto;
error_log /var/log/nginx/error.log warn;
pid /var/run/nginx.pid;

events {
    worker_connections 1024;
    use epoll;
    multi_accept on;
}

http {
    include /etc/nginx/mime.types;
    default_type application/octet-stream;

    # Logging
    log_format main '$remote_addr - $remote_user [$time_local] "$request" '
                    '$status $body_bytes_sent "$http_referer" '
                    '"$http_user_agent" "$http_x_forwarded_for"';
    access_log /var/log/nginx/access.log main;

    # Performance
    sendfile on;
    tcp_nopush on;
    tcp_nodelay on;
    keepalive_timeout 65;
    types_hash_max_size 2048;

    # Gzip
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types text/plain text/css text/xml application/json application/javascript
               application/xml application/rss+xml application/atom+xml image/svg+xml;

    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Rate Limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=login_limit:10m rate=5r/m;

    # Upstream
    upstream app_servers {
        least_conn;
        server web:8080 weight=1 max_fails=3 fail_timeout=30s;
        # server web2:8080 weight=1 max_fails=3 fail_timeout=30s;
        keepalive 32;
    }

    # HTTP to HTTPS redirect
    server {
        listen 80;
        server_name monitoring.diskominfos.baliprov.go.id;

        location /.well-known/acme-challenge/ {
            root /var/www/certbot;
        }

        location / {
            return 301 https://$server_name$request_uri;
        }
    }

    # HTTPS Server
    server {
        listen 443 ssl http2;
        server_name monitoring.diskominfos.baliprov.go.id;

        # SSL Configuration
        ssl_certificate /etc/nginx/ssl/fullchain.pem;
        ssl_certificate_key /etc/nginx/ssl/privkey.pem;
        ssl_session_timeout 1d;
        ssl_session_cache shared:SSL:50m;
        ssl_session_tickets off;

        # Modern SSL configuration
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
        ssl_prefer_server_ciphers off;

        # HSTS
        add_header Strict-Transport-Security "max-age=63072000" always;

        # Static files
        location /static/ {
            alias /var/www/static/;
            expires 30d;
            add_header Cache-Control "public, immutable";
        }

        # API endpoints
        location /api/ {
            limit_req zone=api_limit burst=20 nodelay;

            proxy_pass http://app_servers;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header Connection "";

            proxy_connect_timeout 30s;
            proxy_send_timeout 30s;
            proxy_read_timeout 30s;
        }

        # Login endpoint (stricter rate limit)
        location /api/v1/auth/login {
            limit_req zone=login_limit burst=5 nodelay;

            proxy_pass http://app_servers;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Health check (no rate limit)
        location /health {
            proxy_pass http://app_servers;
            proxy_http_version 1.1;
        }

        # Dashboard and other pages
        location / {
            proxy_pass http://app_servers;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

### 2.4 MySQL Configuration

```ini
# /mysql.cnf

[mysqld]
# Character set
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci

# InnoDB settings
innodb_buffer_pool_size = 1G
innodb_log_file_size = 256M
innodb_flush_log_at_trx_commit = 2
innodb_flush_method = O_DIRECT

# Query cache (disabled in MySQL 8.0+)
# query_cache_type = 0

# Connection settings
max_connections = 200
wait_timeout = 600
interactive_timeout = 600

# Slow query log
slow_query_log = 1
slow_query_log_file = /var/lib/mysql/slow.log
long_query_time = 2

# Binary log for replication (optional)
# log_bin = mysql-bin
# server_id = 1

[client]
default-character-set = utf8mb4
```

---

## 3. Environment Variables

### 3.1 .env File Template

```bash
# .env

# Application
APP_ENV=production
APP_DEBUG=false
APP_SECRET_KEY=your-256-bit-secret-key-here

# Database
DB_HOST=mysql
DB_PORT=3306
DB_USER=monitoring_user
DB_PASSWORD=strong-password-here
DB_ROOT_PASSWORD=root-strong-password-here
DB_NAME=monitoring_website

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# Telegram
TELEGRAM_BOT_TOKEN=1234567890:ABCdefGHIjklMNOpqrsTUVwxyz
TELEGRAM_CHAT_ID=-1001234567890

# JWT
JWT_SECRET=your-jwt-secret-key
JWT_EXPIRY=86400

# Monitoring Settings
UPTIME_CHECK_INTERVAL=5
CONTENT_SCAN_INTERVAL=30
HTTP_TIMEOUT=30
```

---

## 4. Deployment Steps

### 4.1 Initial Server Setup

```bash
#!/bin/bash
# scripts/setup-server.sh

# Update system
sudo apt update && sudo apt upgrade -y

# Install required packages
sudo apt install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git \
    ufw

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Add current user to docker group
sudo usermod -aG docker $USER

# Configure firewall
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Create application directory
sudo mkdir -p /opt/monitoring-website
sudo chown $USER:$USER /opt/monitoring-website

echo "Server setup complete. Please log out and log back in for docker group changes to take effect."
```

### 4.2 Deploy Application

```bash
#!/bin/bash
# scripts/deploy.sh

set -e

APP_DIR="/opt/monitoring-website"
REPO_URL="git@github.com:diskominfos-bali/monitoring-website.git"

echo "Starting deployment..."

# Navigate to app directory
cd $APP_DIR

# Pull latest code (or clone if first time)
if [ -d ".git" ]; then
    echo "Pulling latest changes..."
    git pull origin main
else
    echo "Cloning repository..."
    git clone $REPO_URL .
fi

# Copy environment file if not exists
if [ ! -f ".env" ]; then
    cp .env.example .env
    echo "Please edit .env file with your configuration"
    exit 1
fi

# Build and start containers
echo "Building and starting containers..."
docker-compose build --no-cache
docker-compose up -d

# Run database migrations
echo "Running database migrations..."
docker-compose exec web ./server migrate

# Check status
echo "Checking service status..."
docker-compose ps

echo "Deployment complete!"
echo "Application is available at https://monitoring.diskominfos.baliprov.go.id"
```

### 4.3 SSL Certificate Setup (Let's Encrypt)

```bash
#!/bin/bash
# scripts/setup-ssl.sh

DOMAIN="monitoring.diskominfos.baliprov.go.id"
EMAIL="admin@diskominfos.baliprov.go.id"

# Install certbot
sudo apt install -y certbot

# Stop nginx temporarily
docker-compose stop nginx

# Get certificate
sudo certbot certonly --standalone \
    -d $DOMAIN \
    --email $EMAIL \
    --agree-tos \
    --no-eff-email

# Copy certificates to nginx ssl directory
sudo mkdir -p ./nginx/ssl
sudo cp /etc/letsencrypt/live/$DOMAIN/fullchain.pem ./nginx/ssl/
sudo cp /etc/letsencrypt/live/$DOMAIN/privkey.pem ./nginx/ssl/
sudo chown -R $USER:$USER ./nginx/ssl

# Start nginx
docker-compose start nginx

# Setup auto-renewal cron job
(crontab -l 2>/dev/null; echo "0 3 * * * certbot renew --quiet && docker-compose -f /opt/monitoring-website/docker-compose.yml restart nginx") | crontab -

echo "SSL certificate setup complete!"
```

---

## 5. Backup Strategy

### 5.1 Database Backup Script

```bash
#!/bin/bash
# scripts/backup-db.sh

BACKUP_DIR="/opt/backups/mysql"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="monitoring_website_${DATE}.sql.gz"
RETENTION_DAYS=30

# Create backup directory
mkdir -p $BACKUP_DIR

# Create backup
docker-compose exec -T mysql mysqldump \
    -u root \
    -p${DB_ROOT_PASSWORD} \
    --single-transaction \
    --routines \
    --triggers \
    monitoring_website | gzip > $BACKUP_DIR/$BACKUP_FILE

# Remove old backups
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup created: $BACKUP_DIR/$BACKUP_FILE"

# Optional: Upload to remote storage (S3, GCS, etc.)
# aws s3 cp $BACKUP_DIR/$BACKUP_FILE s3://your-bucket/backups/
```

### 5.2 Backup Cron Schedule

```bash
# Add to crontab
# Daily database backup at 2 AM
0 2 * * * /opt/monitoring-website/scripts/backup-db.sh >> /var/log/backup.log 2>&1

# Weekly full system backup at Sunday 3 AM
0 3 * * 0 /opt/monitoring-website/scripts/backup-full.sh >> /var/log/backup.log 2>&1
```

---

## 6. Monitoring & Logging

### 6.1 Log Rotation

```bash
# /etc/logrotate.d/monitoring-website

/opt/monitoring-website/logs/*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 root root
    sharedscripts
    postrotate
        docker-compose -f /opt/monitoring-website/docker-compose.yml kill -s USR1 web worker
    endscript
}
```

### 6.2 Health Check Script

```bash
#!/bin/bash
# scripts/health-check.sh

HEALTH_URL="http://localhost:8080/health"
TELEGRAM_BOT_TOKEN="your-bot-token"
TELEGRAM_CHAT_ID="your-chat-id"

response=$(curl -s -o /dev/null -w "%{http_code}" $HEALTH_URL)

if [ "$response" != "200" ]; then
    message="⚠️ Monitoring System Health Check Failed!%0A%0AStatus: $response%0ATime: $(date)"
    curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/sendMessage?chat_id=$TELEGRAM_CHAT_ID&text=$message"
fi
```

---

## 7. Scaling Considerations

### 7.1 Horizontal Scaling

```yaml
# docker-compose.scale.yml

version: '3.8'

services:
  web:
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M

  worker:
    deploy:
      replicas: 2
      resources:
        limits:
          cpus: '2'
          memory: 2G
```

### 7.2 Database Replication (Optional)

```yaml
# MySQL Master-Slave setup for read scaling
mysql-master:
  image: mysql:8.0
  environment:
    - MYSQL_ROOT_PASSWORD=${DB_ROOT_PASSWORD}
  command: --server-id=1 --log-bin=mysql-bin --binlog-do-db=monitoring_website

mysql-slave:
  image: mysql:8.0
  environment:
    - MYSQL_ROOT_PASSWORD=${DB_ROOT_PASSWORD}
  command: --server-id=2 --relay-log=relay-log --read-only=1
```

---

## 8. Disaster Recovery

### 8.1 Recovery Procedure

1. **Database Recovery**
   ```bash
   # Restore from backup
   gunzip < backup.sql.gz | docker-compose exec -T mysql mysql -u root -p monitoring_website
   ```

2. **Application Recovery**
   ```bash
   # Pull and redeploy
   cd /opt/monitoring-website
   git pull origin main
   docker-compose up -d --build
   ```

3. **Full Server Recovery**
   - Provision new server
   - Run setup-server.sh
   - Clone repository
   - Restore database backup
   - Configure DNS

### 8.2 Recovery Time Objective (RTO)

| Scenario | RTO |
|----------|-----|
| Container crash | < 1 minute (auto-restart) |
| Database corruption | < 30 minutes |
| Server failure | < 2 hours |
| Complete disaster | < 4 hours |
