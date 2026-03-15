# Checklist: Role System Implementation

## Status: Phase 1 (Single User - Super Admin Only)

Saat ini sistem hanya mendukung 1 user dengan role `super_admin`.
Checklist ini untuk tracking implementasi multi-role di masa depan.

---

## Phase 2: Multi-Role System

### Database

- [ ] Buat tabel `roles`
  ```sql
  CREATE TABLE roles (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      name VARCHAR(50) NOT NULL UNIQUE,
      description VARCHAR(255) NULL,
      created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
  );
  ```

- [ ] Buat tabel `permissions`
  ```sql
  CREATE TABLE permissions (
      id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
      name VARCHAR(100) NOT NULL UNIQUE,  -- e.g., 'websites:create', 'alerts:resolve'
      description VARCHAR(255) NULL
  );
  ```

- [ ] Buat tabel pivot `role_permissions`
  ```sql
  CREATE TABLE role_permissions (
      role_id BIGINT UNSIGNED NOT NULL,
      permission_id BIGINT UNSIGNED NOT NULL,
      PRIMARY KEY (role_id, permission_id)
  );
  ```

- [ ] Update tabel `users`
  - [ ] Tambah kolom `opd_id` (FK ke tabel opd)
  - [ ] Update kolom `role` menjadi FK ke tabel roles (atau tetap enum)

- [ ] Insert default roles
  ```sql
  INSERT INTO roles (name, description) VALUES
  ('super_admin', 'Full access ke semua fitur'),
  ('admin_opd', 'Manage website OPD sendiri'),
  ('viewer', 'View only - read dashboard');
  ```

- [ ] Insert default permissions
  ```sql
  INSERT INTO permissions (name, description) VALUES
  ('websites:create', 'Tambah website baru'),
  ('websites:read', 'Lihat daftar website'),
  ('websites:update', 'Edit website'),
  ('websites:delete', 'Hapus website'),
  ('alerts:read', 'Lihat alerts'),
  ('alerts:resolve', 'Resolve alerts'),
  ('users:manage', 'Manage users'),
  ('settings:manage', 'Manage system settings'),
  ('reports:view', 'Lihat reports'),
  ('reports:export', 'Export reports');
  ```

---

### Backend

- [ ] Buat model `Role`
- [ ] Buat model `Permission`
- [ ] Buat repository `RoleRepository`
- [ ] Buat service `RoleService`

- [ ] Buat middleware `RBACMiddleware`
  ```go
  func RequirePermission(permission string) gin.HandlerFunc {
      return func(c *gin.Context) {
          user := GetCurrentUser(c)
          if !user.HasPermission(permission) {
              c.AbortWithStatusJSON(403, gin.H{"error": "Forbidden"})
              return
          }
          c.Next()
      }
  }
  ```

- [ ] Update `UserService`
  - [ ] Method `HasPermission(permission string) bool`
  - [ ] Method `GetPermissions() []string`
  - [ ] Method `BelongsToOPD(opdID int64) bool`

- [ ] Update semua handler untuk cek permission
  - [ ] WebsiteHandler
  - [ ] AlertHandler
  - [ ] ReportHandler
  - [ ] SettingsHandler

---

### API Endpoints

- [ ] `GET /api/v1/roles` - List all roles
- [ ] `POST /api/v1/roles` - Create role
- [ ] `PUT /api/v1/roles/{id}` - Update role
- [ ] `DELETE /api/v1/roles/{id}` - Delete role
- [ ] `GET /api/v1/roles/{id}/permissions` - Get role permissions
- [ ] `PUT /api/v1/roles/{id}/permissions` - Update role permissions

- [ ] Update user endpoints
  - [ ] `POST /api/v1/users` - Tambah field role_id, opd_id
  - [ ] `PUT /api/v1/users/{id}` - Bisa update role

---

### Frontend/Dashboard

- [ ] Halaman User Management
  - [ ] List users
  - [ ] Create user form (dengan pilih role & OPD)
  - [ ] Edit user
  - [ ] Delete user

- [ ] Halaman Role Management (super_admin only)
  - [ ] List roles
  - [ ] Permission matrix editor

- [ ] Update semua halaman untuk hide/show berdasarkan permission
  - [ ] Sidebar menu
  - [ ] Action buttons (edit, delete)
  - [ ] Settings page

- [ ] Filter data berdasarkan OPD untuk admin_opd
  - [ ] Website list: hanya tampilkan OPD sendiri
  - [ ] Alerts: hanya tampilkan OPD sendiri
  - [ ] Reports: hanya OPD sendiri

---

### Testing

- [ ] Unit test untuk RBACMiddleware
- [ ] Unit test untuk UserService.HasPermission
- [ ] Integration test untuk setiap role:
  - [ ] super_admin dapat akses semua
  - [ ] admin_opd hanya akses OPD sendiri
  - [ ] viewer hanya bisa read

---

### Documentation

- [ ] Update API documentation
- [ ] Update user guide
- [ ] Buat dokumentasi role & permission

---

## Permission Matrix (Reference)

| Permission | Super Admin | Admin OPD | Viewer |
|------------|:-----------:|:---------:|:------:|
| websites:create | ✅ | ✅ (own) | ❌ |
| websites:read | ✅ | ✅ (own) | ✅ (own) |
| websites:update | ✅ | ✅ (own) | ❌ |
| websites:delete | ✅ | ❌ | ❌ |
| alerts:read | ✅ | ✅ (own) | ✅ (own) |
| alerts:resolve | ✅ | ✅ (own) | ❌ |
| users:manage | ✅ | ❌ | ❌ |
| settings:manage | ✅ | ❌ | ❌ |
| reports:view | ✅ | ✅ (own) | ✅ (own) |
| reports:export | ✅ | ✅ (own) | ❌ |

**(own)** = hanya untuk website/data yang terkait dengan OPD user tersebut

---

## Timeline Suggestion

| Phase | Task | Priority |
|-------|------|----------|
| 2.1 | Database schema & migrations | High |
| 2.2 | Backend RBAC middleware | High |
| 2.3 | Update existing handlers | High |
| 2.4 | User management UI | Medium |
| 2.5 | Role management UI | Medium |
| 2.6 | Testing | High |
| 2.7 | Documentation | Medium |
