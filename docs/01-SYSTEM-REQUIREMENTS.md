# Dokumen Analisis Sistem dan Requirements

## Monitoring Website Pemerintah Provinsi Bali
**Versi:** 1.0
**Tanggal:** Januari 2026
**Penyusun:** Diskominfos Provinsi Bali

---

## 1. Latar Belakang

Website pemerintah dengan domain `*.baliprov.go.id` merupakan aset digital penting yang harus dijaga ketersediaan dan integritasnya. Ancaman seperti:
- **Downtime** yang mengganggu layanan publik
- **Defacement** yang merusak citra pemerintah
- **Penyusupan konten judi online (judol)** yang marak terjadi
- **Expired SSL certificate** yang menurunkan kepercayaan pengguna

Memerlukan sistem monitoring yang proaktif dan real-time.

---

## 2. Tujuan Sistem

### 2.1 Tujuan Utama
1. Mendeteksi downtime website secara real-time
2. Mendeteksi penyusupan konten judi online (judol)
3. Mendeteksi defacement/perubahan konten tidak sah
4. Monitoring SSL certificate expiry
5. Memberikan notifikasi cepat kepada tim terkait

### 2.2 Tujuan Sekunder
1. Menyediakan dashboard monitoring terpusat
2. Menyimpan historical data untuk analisis
3. Menghasilkan laporan berkala
4. Memudahkan inventory management website

---

## 3. Functional Requirements

### FR-01: Website Management
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-01.1 | Sistem dapat menambah website baru untuk dimonitor | High |
| FR-01.2 | Sistem dapat mengedit informasi website | High |
| FR-01.3 | Sistem dapat menghapus website dari monitoring | High |
| FR-01.4 | Sistem dapat mengimport daftar website dari CSV | Medium |
| FR-01.5 | Sistem dapat mengelompokkan website berdasarkan OPD | Medium |

### FR-02: Uptime Monitoring
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-02.1 | Sistem dapat mengecek status HTTP (2xx, 3xx, 4xx, 5xx) | High |
| FR-02.2 | Sistem dapat mengukur response time | High |
| FR-02.3 | Sistem dapat mendeteksi connection timeout | High |
| FR-02.4 | Sistem dapat mendeteksi DNS resolution failure | High |
| FR-02.5 | Sistem mencatat uptime percentage | Medium |

### FR-03: SSL Monitoring
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-03.1 | Sistem dapat mengecek validitas SSL certificate | High |
| FR-03.2 | Sistem dapat mendeteksi SSL expiry date | High |
| FR-03.3 | Sistem memberikan warning H-30, H-14, H-7 sebelum expired | High |
| FR-03.4 | Sistem mendeteksi SSL issuer dan grade | Low |

### FR-04: Content Monitoring (Judol Detection)
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-04.1 | Sistem dapat scan halaman untuk keyword gambling | High |
| FR-04.2 | Sistem dapat mendeteksi iframe/script mencurigakan | High |
| FR-04.3 | Sistem dapat mendeteksi redirect ke situs judi | High |
| FR-04.4 | Sistem dapat scan meta tags yang dimanipulasi | Medium |
| FR-04.5 | Sistem dapat mendeteksi hidden elements | Medium |
| FR-04.6 | Sistem dapat customizable keyword list | Medium |

### FR-05: Defacement Detection
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-05.1 | Sistem dapat mendeteksi perubahan title yang drastis | High |
| FR-05.2 | Sistem dapat hash comparison untuk detect perubahan | Medium |
| FR-05.3 | Sistem dapat visual comparison (screenshot) | Low |

### FR-06: Notification System
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-06.1 | Sistem mengirim alert via Telegram bot | High |
| FR-06.2 | Sistem mendukung multiple notification channel | Medium |
| FR-06.3 | Sistem memiliki escalation policy | Medium |
| FR-06.4 | Sistem dapat mute notification sementara | Medium |

### FR-07: Dashboard & Reporting
| ID | Requirement | Priority |
|----|-------------|----------|
| FR-07.1 | Dashboard menampilkan status semua website | High |
| FR-07.2 | Dashboard menampilkan statistik uptime | High |
| FR-07.3 | Sistem dapat generate laporan PDF | Medium |
| FR-07.4 | Sistem menyediakan API untuk integrasi | Medium |

---

## 4. Non-Functional Requirements

### NFR-01: Performance
| ID | Requirement | Target |
|----|-------------|--------|
| NFR-01.1 | Monitoring interval minimum | 1 menit |
| NFR-01.2 | Dashboard load time | < 3 detik |
| NFR-01.3 | Concurrent website monitoring | 500+ websites |
| NFR-01.4 | Alert delivery time | < 30 detik |

### NFR-02: Availability
| ID | Requirement | Target |
|----|-------------|--------|
| NFR-02.1 | System uptime | 99.5% |
| NFR-02.2 | Data retention | 1 tahun |

### NFR-03: Security
| ID | Requirement | Target |
|----|-------------|--------|
| NFR-03.1 | Authentication required | Yes |
| NFR-03.2 | HTTPS only | Yes |
| NFR-03.3 | Role-based access control | Yes |
| NFR-03.4 | Audit logging | Yes |

### NFR-04: Scalability
| ID | Requirement | Target |
|----|-------------|--------|
| NFR-04.1 | Horizontal scaling support | Yes |
| NFR-04.2 | Database read replicas | Optional |

---

## 5. User Roles

### 5.1 Phase 1: Single User (Current)

Untuk tahap awal, sistem hanya menggunakan **1 user Super Admin** dengan full access:

- Full access ke semua fitur
- Manage semua website
- System configuration
- View semua alerts dan reports

### 5.2 Phase 2: Multi-Role (Future Development)

Checklist untuk pengembangan role system di masa depan:

```
ROLE SYSTEM CHECKLIST
═════════════════════

☐ Database & Model
  ☐ Tabel roles
  ☐ Tabel permissions
  ☐ Tabel role_permissions (pivot)
  ☐ Update tabel users (tambah role_id)

☐ Backend Implementation
  ☐ Role middleware/guard
  ☐ Permission checker helper
  ☐ RBAC service

☐ Roles yang Direncanakan
  ☐ Super Admin - Full access semua fitur
  ☐ Admin OPD - Manage website OPD sendiri
  ☐ Viewer - Read-only dashboard

☐ UI/Dashboard
  ☐ User management page
  ☐ Role assignment UI
  ☐ Permission matrix view

☐ Features per Role
  Super Admin:
    ☐ Manage all websites
    ☐ Manage users & roles
    ☐ System settings
    ☐ View all alerts
    ☐ Generate all reports

  Admin OPD:
    ☐ Manage own OPD websites
    ☐ View own OPD alerts
    ☐ Resolve own alerts
    ☐ View own reports

  Viewer:
    ☐ View dashboard (read-only)
    ☐ View alerts (read-only)
    ☐ View reports (read-only)
```

---

## 6. Stakeholders

| Stakeholder | Kepentingan |
|-------------|-------------|
| Kepala Diskominfos | Executive summary, laporan bulanan |
| Tim Persandian/Keamanan | Alert real-time, incident response |
| Admin Website OPD | Status website masing-masing |
| Tim NOC | Dashboard monitoring 24/7 |

---

## 7. Assumptions & Constraints

### Assumptions
1. Semua website target dapat diakses dari server monitoring
2. Telegram bot dapat diakses dari server
3. Tim akan merespons alert dalam waktu yang wajar

### Constraints
1. Monitoring dilakukan dari sisi external (black-box)
2. Tidak ada akses ke server internal website target
3. Rate limiting harus diperhatikan agar tidak membebani website target
4. Budget infrastruktur terbatas

---

## 8. Out of Scope

Berikut yang TIDAK termasuk dalam scope proyek ini:
1. Penetration testing otomatis
2. Vulnerability scanning (gunakan tool terpisah)
3. Web Application Firewall (WAF)
4. DDoS protection
5. Log analysis dari server website
6. Code review atau static analysis

---

## 9. Success Criteria

1. ✅ Sistem dapat memonitor 100+ website secara bersamaan
2. ✅ Alert terkirim dalam < 1 menit setelah incident terdeteksi
3. ✅ False positive rate < 5%
4. ✅ Dashboard dapat diakses 24/7
5. ✅ Semua website dengan domain baliprov.go.id termonitor
