# Dokumen API Specification

## Monitoring Website Pemerintah Provinsi Bali

**Base URL:** `https://monitoring.diskominfos.baliprov.go.id/api/v1`
**API Version:** v1

---

## 1. Authentication

### 1.1 Login

Mendapatkan access token untuk authentication.

```
POST /auth/login
```

**Request Body:**
```json
{
    "username": "admin",
    "password": "password123"
}
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "token_type": "Bearer",
        "expires_in": 86400,
        "user": {
            "id": 1,
            "username": "admin",
            "email": "admin@diskominfos.baliprov.go.id",
            "full_name": "Administrator",
            "role": "super_admin"
        }
    }
}
```

**Response (401 Unauthorized):**
```json
{
    "success": false,
    "error": {
        "code": "INVALID_CREDENTIALS",
        "message": "Username atau password salah"
    }
}
```

### 1.2 Refresh Token

```
POST /auth/refresh
```

**Headers:**
```
Authorization: Bearer {refresh_token}
```

### 1.3 Logout

```
POST /auth/logout
```

**Headers:**
```
Authorization: Bearer {access_token}
```

---

## 2. Websites

### 2.1 List All Websites

```
GET /websites
```

**Headers:**
```
Authorization: Bearer {access_token}
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | int | Halaman (default: 1) |
| limit | int | Items per page (default: 20, max: 100) |
| status | string | Filter by status: up, down, degraded, unknown |
| opd_id | int | Filter by OPD ID |
| search | string | Search by name or URL |
| content_clean | bool | Filter by content status |
| sort | string | Sort field: name, status, last_checked_at |
| order | string | Sort order: asc, desc |

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "items": [
            {
                "id": 1,
                "url": "https://diskominfos.baliprov.go.id",
                "name": "Website Diskominfos",
                "opd": {
                    "id": 1,
                    "name": "Dinas Komunikasi Informatika dan Statistik",
                    "code": "DISKOMINFOS"
                },
                "status": "up",
                "last_status_code": 200,
                "last_response_time": 450,
                "last_checked_at": "2026-01-22T10:30:00Z",
                "ssl_valid": true,
                "ssl_expiry_date": "2026-06-15",
                "content_clean": true,
                "is_active": true
            }
        ],
        "pagination": {
            "current_page": 1,
            "total_pages": 5,
            "total_items": 98,
            "items_per_page": 20
        }
    }
}
```

### 2.2 Get Website Detail

```
GET /websites/{id}
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "id": 1,
        "url": "https://diskominfos.baliprov.go.id",
        "name": "Website Diskominfos",
        "description": "Website resmi Diskominfos Provinsi Bali",
        "opd": {
            "id": 1,
            "name": "Dinas Komunikasi Informatika dan Statistik",
            "code": "DISKOMINFOS"
        },
        "settings": {
            "check_interval": 5,
            "timeout": 30,
            "is_active": true
        },
        "current_status": {
            "status": "up",
            "status_code": 200,
            "response_time": 450,
            "checked_at": "2026-01-22T10:30:00Z"
        },
        "ssl": {
            "is_valid": true,
            "issuer": "Let's Encrypt",
            "expiry_date": "2026-06-15",
            "days_until_expiry": 144
        },
        "content": {
            "is_clean": true,
            "last_scan_at": "2026-01-22T10:00:00Z"
        },
        "uptime_stats": {
            "last_24h": 99.8,
            "last_7d": 99.5,
            "last_30d": 99.2
        },
        "created_at": "2025-06-01T00:00:00Z",
        "updated_at": "2026-01-22T10:30:00Z"
    }
}
```

### 2.3 Create Website

```
POST /websites
```

**Request Body:**
```json
{
    "url": "https://dinkes.baliprov.go.id",
    "name": "Website Dinas Kesehatan",
    "description": "Website resmi Dinas Kesehatan Provinsi Bali",
    "opd_id": 2,
    "check_interval": 5,
    "timeout": 30,
    "is_active": true
}
```

**Response (201 Created):**
```json
{
    "success": true,
    "data": {
        "id": 99,
        "url": "https://dinkes.baliprov.go.id",
        "name": "Website Dinas Kesehatan",
        "message": "Website berhasil ditambahkan dan akan mulai dimonitor"
    }
}
```

### 2.4 Update Website

```
PUT /websites/{id}
```

**Request Body:**
```json
{
    "name": "Website Dinas Kesehatan Provinsi Bali",
    "description": "Website resmi Dinkes Bali",
    "check_interval": 3,
    "is_active": true
}
```

### 2.5 Delete Website

```
DELETE /websites/{id}
```

**Response (200 OK):**
```json
{
    "success": true,
    "message": "Website berhasil dihapus dari monitoring"
}
```

### 2.6 Import Websites from CSV

```
POST /websites/import
Content-Type: multipart/form-data
```

**Form Data:**
- `file`: CSV file

**CSV Format:**
```csv
url,name,opd_code,description
https://dinkes.baliprov.go.id,Dinas Kesehatan,DINKES,Website Dinkes
https://bappeda.baliprov.go.id,Bappeda,BAPPEDA,Website Bappeda
```

### 2.7 Manual Check Website

Trigger check manual untuk website tertentu.

```
POST /websites/{id}/check
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "website_id": 1,
        "status": "up",
        "status_code": 200,
        "response_time": 523,
        "checked_at": "2026-01-22T10:35:00Z"
    }
}
```

---

## 3. Checks (History)

### 3.1 Get Check History

```
GET /websites/{id}/checks
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | int | Halaman |
| limit | int | Items per page |
| start_date | string | Filter dari tanggal (ISO 8601) |
| end_date | string | Filter sampai tanggal (ISO 8601) |
| status | string | Filter by status |

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "items": [
            {
                "id": 12345,
                "status": "up",
                "status_code": 200,
                "response_time": 450,
                "checked_at": "2026-01-22T10:30:00Z"
            },
            {
                "id": 12344,
                "status": "up",
                "status_code": 200,
                "response_time": 480,
                "checked_at": "2026-01-22T10:25:00Z"
            }
        ],
        "pagination": {
            "current_page": 1,
            "total_pages": 100,
            "total_items": 2000
        }
    }
}
```

### 3.2 Get Uptime Statistics

```
GET /websites/{id}/uptime
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| period | string | 24h, 7d, 30d, 90d |

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "website_id": 1,
        "period": "7d",
        "uptime_percentage": 99.5,
        "total_checks": 2016,
        "up_count": 2006,
        "down_count": 10,
        "average_response_time": 456,
        "min_response_time": 230,
        "max_response_time": 3200,
        "incidents": [
            {
                "started_at": "2026-01-20T14:30:00Z",
                "ended_at": "2026-01-20T14:45:00Z",
                "duration_minutes": 15,
                "type": "down"
            }
        ]
    }
}
```

---

## 4. Alerts

### 4.1 List Alerts

```
GET /alerts
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| page | int | Halaman |
| limit | int | Items per page |
| status | string | active, resolved, all |
| severity | string | info, warning, critical |
| type | string | down, up, ssl_expiring, judol_detected |
| website_id | int | Filter by website |
| start_date | string | Filter dari tanggal |
| end_date | string | Filter sampai tanggal |

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "items": [
            {
                "id": 456,
                "website": {
                    "id": 1,
                    "name": "Website Diskominfos",
                    "url": "https://diskominfos.baliprov.go.id"
                },
                "type": "judol_detected",
                "severity": "critical",
                "title": "Konten Judi Online Terdeteksi",
                "message": "Ditemukan 3 keyword judi online pada halaman utama",
                "context": {
                    "keywords_found": ["slot gacor", "togel", "judi online"],
                    "scan_url": "https://diskominfos.baliprov.go.id/"
                },
                "is_resolved": false,
                "is_acknowledged": false,
                "created_at": "2026-01-22T09:15:00Z"
            }
        ],
        "pagination": {
            "current_page": 1,
            "total_pages": 3,
            "total_items": 45
        },
        "summary": {
            "total_active": 12,
            "critical": 2,
            "warning": 5,
            "info": 5
        }
    }
}
```

### 4.2 Get Alert Detail

```
GET /alerts/{id}
```

### 4.3 Acknowledge Alert

```
POST /alerts/{id}/acknowledge
```

**Request Body:**
```json
{
    "note": "Sedang ditangani oleh tim IT"
}
```

### 4.4 Resolve Alert

```
POST /alerts/{id}/resolve
```

**Request Body:**
```json
{
    "resolution_note": "Konten judi sudah dibersihkan dan website sudah aman"
}
```

---

## 5. Content Scans

### 5.1 Get Scan History

```
GET /websites/{id}/scans
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "items": [
            {
                "id": 789,
                "is_clean": false,
                "findings": [
                    {
                        "type": "keyword",
                        "keyword": "slot gacor",
                        "location": "body",
                        "snippet": "...main <b>slot gacor</b> sekarang..."
                    },
                    {
                        "type": "iframe",
                        "src": "https://malicious-site.com/embed",
                        "location": "body"
                    }
                ],
                "keywords_found": 2,
                "iframes_found": 1,
                "scanned_at": "2026-01-22T10:00:00Z"
            }
        ]
    }
}
```

### 5.2 Manual Content Scan

```
POST /websites/{id}/scan
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| type | string | quick, full (default: quick) |

---

## 6. SSL Checks

### 6.1 Get SSL History

```
GET /websites/{id}/ssl
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "current": {
            "is_valid": true,
            "issuer": "Let's Encrypt Authority X3",
            "subject": "diskominfos.baliprov.go.id",
            "valid_from": "2026-01-01T00:00:00Z",
            "valid_until": "2026-06-15T23:59:59Z",
            "days_until_expiry": 144,
            "protocol": "TLSv1.3",
            "checked_at": "2026-01-22T06:00:00Z"
        },
        "history": [
            {
                "is_valid": true,
                "days_until_expiry": 145,
                "checked_at": "2026-01-21T06:00:00Z"
            }
        ]
    }
}
```

---

## 7. Dashboard & Statistics

### 7.1 Dashboard Overview

```
GET /dashboard/overview
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "summary": {
            "total_websites": 98,
            "websites_up": 92,
            "websites_down": 3,
            "websites_degraded": 2,
            "websites_unknown": 1,
            "content_compromised": 1,
            "ssl_expiring_soon": 5
        },
        "alerts": {
            "total_active": 12,
            "critical": 2,
            "warning": 5,
            "info": 5
        },
        "uptime": {
            "average_24h": 98.5,
            "average_7d": 99.2,
            "average_30d": 99.5
        },
        "recent_incidents": [
            {
                "website_name": "Website Dinkes",
                "type": "down",
                "started_at": "2026-01-22T09:00:00Z",
                "duration_minutes": 15
            }
        ]
    }
}
```

### 7.2 Get Statistics by OPD

```
GET /dashboard/stats/opd
```

**Response (200 OK):**
```json
{
    "success": true,
    "data": [
        {
            "opd": {
                "id": 1,
                "name": "Diskominfos",
                "code": "DISKOMINFOS"
            },
            "total_websites": 5,
            "websites_up": 5,
            "websites_down": 0,
            "average_uptime": 99.8,
            "content_issues": 0
        }
    ]
}
```

### 7.3 Export Report

```
GET /reports/export
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| format | string | pdf, xlsx, csv |
| period | string | daily, weekly, monthly |
| start_date | string | Tanggal mulai |
| end_date | string | Tanggal akhir |
| opd_id | int | Filter by OPD (optional) |

---

## 8. Keywords Management

### 8.1 List Keywords

```
GET /keywords
```

**Query Parameters:**
| Parameter | Type | Description |
|-----------|------|-------------|
| category | string | gambling, defacement, malware, custom |
| is_active | bool | Filter by active status |

### 8.2 Create Keyword

```
POST /keywords
```

**Request Body:**
```json
{
    "keyword": "bonus deposit",
    "category": "gambling",
    "weight": 7,
    "is_regex": false
}
```

### 8.3 Update Keyword

```
PUT /keywords/{id}
```

### 8.4 Delete Keyword

```
DELETE /keywords/{id}
```

---

## 9. Users Management

### 9.1 List Users

```
GET /users
```

**Note:** Requires `super_admin` role

### 9.2 Create User

```
POST /users
```

**Request Body:**
```json
{
    "username": "admin_dinkes",
    "email": "admin@dinkes.baliprov.go.id",
    "password": "SecurePassword123!",
    "full_name": "Admin Dinas Kesehatan",
    "role": "admin_opd",
    "opd_id": 2
}
```

### 9.3 Update User

```
PUT /users/{id}
```

### 9.4 Delete User

```
DELETE /users/{id}
```

### 9.5 Get Current User Profile

```
GET /users/me
```

### 9.6 Update Password

```
PUT /users/me/password
```

**Request Body:**
```json
{
    "current_password": "old_password",
    "new_password": "NewSecurePassword123!"
}
```

---

## 10. OPD Management

### 10.1 List OPD

```
GET /opd
```

### 10.2 Create OPD

```
POST /opd
```

**Request Body:**
```json
{
    "name": "Dinas Pendidikan",
    "code": "DISDIK",
    "contact_email": "admin@disdik.baliprov.go.id",
    "contact_phone": "0361123456"
}
```

---

## 11. Settings

### 11.1 Get Settings

```
GET /settings
```

**Note:** Requires `super_admin` role

**Response (200 OK):**
```json
{
    "success": true,
    "data": {
        "uptime_check_interval": 5,
        "content_scan_interval": 30,
        "ssl_check_interval": 1440,
        "http_timeout": 30,
        "response_time_warning": 3000,
        "response_time_critical": 10000,
        "ssl_expiry_warning_days": 30,
        "max_concurrent_checks": 20
    }
}
```

### 11.2 Update Settings

```
PUT /settings
```

---

## 12. Error Responses

### Standard Error Format

```json
{
    "success": false,
    "error": {
        "code": "ERROR_CODE",
        "message": "Human readable message",
        "details": {}
    }
}
```

### Error Codes

| HTTP Status | Code | Description |
|-------------|------|-------------|
| 400 | VALIDATION_ERROR | Input validation failed |
| 401 | UNAUTHORIZED | Authentication required |
| 403 | FORBIDDEN | Insufficient permissions |
| 404 | NOT_FOUND | Resource not found |
| 409 | CONFLICT | Resource conflict (duplicate) |
| 422 | UNPROCESSABLE | Business logic error |
| 429 | RATE_LIMITED | Too many requests |
| 500 | INTERNAL_ERROR | Server error |

---

## 13. Webhook (Outgoing)

Sistem dapat mengirim webhook ke external services.

### Webhook Payload

```json
{
    "event": "alert.created",
    "timestamp": "2026-01-22T10:30:00Z",
    "data": {
        "alert_id": 456,
        "type": "down",
        "severity": "critical",
        "website": {
            "id": 1,
            "name": "Website Diskominfos",
            "url": "https://diskominfos.baliprov.go.id"
        },
        "message": "Website tidak dapat diakses"
    }
}
```

### Webhook Events

| Event | Description |
|-------|-------------|
| alert.created | Alert baru dibuat |
| alert.resolved | Alert di-resolve |
| website.down | Website down |
| website.up | Website kembali up |
| content.compromised | Konten terdeteksi masalah |
| ssl.expiring | SSL akan expired |

---

## 14. Rate Limiting

| Endpoint Type | Limit |
|---------------|-------|
| Authentication | 10 req/min |
| Read (GET) | 100 req/min |
| Write (POST/PUT/DELETE) | 30 req/min |
| Manual Check/Scan | 5 req/min per website |

**Response Header:**
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642850400
```
