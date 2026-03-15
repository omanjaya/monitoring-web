const API_BASE = typeof window !== "undefined" ? "" : (process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080")

class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.name = "ApiError"
    this.status = status
  }
}

function getToken(): string | null {
  if (typeof window === "undefined") return null
  return localStorage.getItem("token")
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...((options.headers as Record<string, string>) || {}),
  }

  if (token) {
    headers["Authorization"] = `Bearer ${token}`
  }

  const res = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  })

  if (res.status === 401) {
    if (typeof window !== "undefined") {
      localStorage.removeItem("token")
      window.location.href = "/login"
    }
    throw new ApiError("Unauthorized", 401)
  }

  const data = await res.json()

  if (!res.ok) {
    throw new ApiError(data.error || "Request failed", res.status)
  }

  return data
}

export const api = {
  get: <T>(endpoint: string) => request<T>(endpoint),
  post: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: "POST", body: JSON.stringify(body) }),
  put: <T>(endpoint: string, body?: unknown) =>
    request<T>(endpoint, { method: "PUT", body: JSON.stringify(body) }),
  delete: <T>(endpoint: string) =>
    request<T>(endpoint, { method: "DELETE" }),
}

// Auth
export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ token: string }>("/api/auth/login", { username, password }),
  me: () => api.get<{ data: User }>("/api/auth/me"),
  changePassword: (current_password: string, new_password: string) =>
    api.put("/api/auth/password", { current_password, new_password }),
}

// Dashboard
export const dashboardApi = {
  overview: () => api.get<{ data: DashboardOverview }>("/api/dashboard"),
  stats: () => api.get<{ data: DashboardStats }>("/api/dashboard/stats"),
  trends: () => api.get<{ data: DashboardTrends }>("/api/dashboard/trends"),
}

// Websites
export const websiteApi = {
  list: (params?: string) => api.get<{ data: Website[]; total: number }>(`/api/websites${params ? `?${params}` : ""}`),
  get: (id: number) => api.get<{ data: Website }>(`/api/websites/${id}`),
  create: (data: WebsiteCreate) => api.post<{ data: { id: number } }>("/api/websites", data),
  update: (id: number, data: Partial<WebsiteCreate>) => api.put(`/api/websites/${id}`, data),
  delete: (id: number) => api.delete(`/api/websites/${id}`),
  uptime: (id: number, hours?: number) => api.get<{ data: UptimeEntry[] }>(`/api/websites/${id}/uptime?hours=${hours || 24}`),
  bulkAction: (ids: number[], action: string) => api.post<{ data: { affected: number } }>("/api/websites/bulk-action", { ids, action }),
  bulkImport: (data: WebsiteCreate[]) => api.post<{ data: BulkImportResult }>("/api/websites/bulk", data),
  metrics: (id: number) => api.get<{ data: WebsiteMetrics }>(`/api/websites/${id}/metrics`),
  dns: (id: number) => api.get<{ data: DNSScanResult }>(`/api/websites/${id}/dns`),
}

// Alerts
export const alertApi = {
  list: (params?: string) => api.get<{ data: Alert[]; total: number }>(`/api/alerts${params ? `?${params}` : ""}`),
  active: () => api.get<{ data: Alert[] }>("/api/alerts/active"),
  summary: () => api.get<{ data: AlertSummary }>("/api/alerts/summary"),
  get: (id: number) => api.get<{ data: Alert }>(`/api/alerts/${id}`),
  acknowledge: (id: number) => api.post(`/api/alerts/${id}/acknowledge`),
  resolve: (id: number, note?: string) => api.post(`/api/alerts/${id}/resolve`, { note }),
  bulkResolve: (ids: number[], note?: string) =>
    api.post<{ affected: number; message: string }>("/api/alerts/bulk-resolve", { ids, note }),
}

// Users
export const userApi = {
  list: () => api.get<{ data: User[] }>("/api/users"),
  create: (data: UserCreate) => api.post("/api/users", data),
  update: (id: number, data: Partial<UserUpdate>) => api.put(`/api/users/${id}`, data),
  delete: (id: number) => api.delete(`/api/users/${id}`),
  resetPassword: (id: number, password: string) => api.post(`/api/users/${id}/reset-password`, { new_password: password }),
}

// Audit
export const auditApi = {
  list: (params?: string) => api.get<{ data: AuditLog[]; total: number }>(`/api/audit-logs${params ? `?${params}` : ""}`),
}

// OPD
export const opdApi = {
  list: () => api.get<{ data: OPD[] }>("/api/opd"),
  create: (data: { name: string; code: string; contact_email?: string; contact_phone?: string }) => api.post("/api/opd", data),
  bulkImport: (data: { name: string; code: string; contact_email?: string; contact_phone?: string }[]) => api.post<{ data: BulkImportResult }>("/api/opd/bulk", data),
}

// Admin
export const adminApi = {
  status: () => api.get<{ data: SystemStatus }>("/api/admin/status"),
  trigger: (type: string) => api.post(`/api/admin/trigger?type=${type}`),
  testTelegram: () => api.post("/api/admin/test-telegram"),
  testEmail: () => api.post("/api/admin/test-email"),
  testWebhook: () => api.post("/api/admin/test-webhook"),
}

// Keywords
export const keywordApi = {
  list: () => api.get<{ data: Keyword[] }>("/api/keywords"),
  create: (data: { keyword: string; category: string; is_regex?: boolean; weight?: number }) => api.post("/api/keywords", data),
  delete: (id: number) => api.delete(`/api/keywords/${id}`),
  bulkImport: (data: { keyword: string; category: string; is_regex?: boolean; weight?: number }[]) => api.post<{ data: BulkImportResult }>("/api/keywords/bulk", data),
}

// Maintenance
export const maintenanceApi = {
  list: () => api.get<{ data: Maintenance[] }>("/api/maintenance"),
  get: (id: number) => api.get<{ data: Maintenance }>(`/api/maintenance/${id}`),
  create: (data: MaintenanceCreate) => api.post("/api/maintenance", data),
  update: (id: number, data: Partial<MaintenanceCreate>) => api.put(`/api/maintenance/${id}`, data),
  delete: (id: number) => api.delete(`/api/maintenance/${id}`),
  cancel: (id: number) => api.post(`/api/maintenance/${id}/cancel`),
  complete: (id: number) => api.post(`/api/maintenance/${id}/complete`),
  current: () => api.get<{ data: Maintenance[] }>("/api/maintenance/current"),
  upcoming: () => api.get<{ data: Maintenance[] }>("/api/maintenance/upcoming"),
}

// Security
export const securityApi = {
  stats: () => api.get<{ data: unknown }>("/api/security/stats"),
  summary: (params?: string) => api.get<{ data: unknown; total: number }>(`/api/security/summary${params ? `?${params}` : ""}`),
  website: (id: number) => api.get<{ data: unknown }>(`/api/security/websites/${id}`),
  history: (id: number, limit?: number) => api.get<{ data: unknown }>(`/api/security/websites/${id}/history?limit=${limit || 10}`),
  check: (id: number) => api.post(`/api/security/websites/${id}/check`),
  checkAll: () => api.post("/api/security/check-all"),
}

// Reports
export const reportApi = {
  types: () => api.get<{ data: unknown }>("/api/reports/types"),
  generate: (data: unknown) => api.post<{ data: unknown }>("/api/reports/generate", data),
  quick: (type: string, period: string) => api.get<{ data: unknown }>(`/api/reports/quick/${type}/${period}`),
  scheduleOptions: () => api.get<{ data: unknown }>("/api/reports/schedule/options"),
}

// Escalation
export const escalationApi = {
  policies: () => api.get<{ data: unknown[] }>("/api/escalation/policies"),
  getPolicy: (id: number) => api.get<{ data: unknown }>(`/api/escalation/policies/${id}`),
  createPolicy: (data: unknown) => api.post("/api/escalation/policies", data),
  updatePolicy: (id: number, data: unknown) => api.put(`/api/escalation/policies/${id}`, data),
  deletePolicy: (id: number) => api.delete(`/api/escalation/policies/${id}`),
  history: () => api.get<{ data: unknown[] }>("/api/escalation/history"),
  summary: () => api.get<{ data: unknown }>("/api/escalation/summary"),
  createRule: (data: { policy_id: number; level: number; delay_minutes: number; notify_channels: string; notify_contacts: string }) =>
    api.post("/api/escalation/rules", data),
  deleteRule: (id: number) => api.delete(`/api/escalation/rules/${id}`),
  trigger: (data: { policy_id: number; alert_id?: number }) => api.post("/api/escalation/trigger", data),
}

// Dork
export const dorkApi = {
  stats: () => api.get<{ data: unknown }>("/api/dork/stats"),
  categories: () => api.get<{ data: DorkCategory[] }>("/api/dork/categories"),
  patterns: () => api.get<{ data: unknown[] }>("/api/dork/patterns"),
  createPattern: (data: { name: string; category: string; pattern: string; pattern_type?: string; severity?: string; description?: string }) =>
    api.post("/api/dork/patterns", data),
  updatePattern: (id: number, data: unknown) => api.put(`/api/dork/patterns/${id}`, data),
  deletePattern: (id: number) => api.delete(`/api/dork/patterns/${id}`),
  detections: (params?: string) => api.get<{ data: unknown[]; total?: number }>(`/api/dork/detections${params ? `?${params}` : ""}`),
  detection: (id: number) => api.get<{ data: DorkDetectionDetail }>(`/api/dork/detections/${id}`),
  resolveDetection: (id: number, notes?: string) => api.post(`/api/dork/detections/${id}/resolve`, { notes }),
  markFalsePositive: (id: number, notes?: string) => api.post(`/api/dork/detections/${id}/false-positive`, { notes }),
  scanAll: () => api.post("/api/dork/scan-all"),
  clearAll: () => api.delete<{ deleted: number }>("/api/dork/detections"),
  verifyAI: () => api.post<{ verified: number; false_positives: number }>("/api/dork/verify-ai"),
  websiteStats: (id: number) => api.get<{ data: unknown }>(`/api/dork/websites/${id}/stats`),
  websiteScans: (id: number) => api.get<{ data: unknown[] }>(`/api/dork/websites/${id}/scans`),
  websiteScan: (id: number) => api.post(`/api/dork/websites/${id}/scan`),
  scanResult: (id: number) => api.get<{ data: unknown }>(`/api/dork/scans/${id}`),
}

// Defacement Archive
export const defacementApi = {
  stats: () => api.get<{ data: unknown }>("/api/defacement/stats"),
  incidents: (params?: string) => api.get<{ data: unknown[]; total?: number }>(`/api/defacement/incidents${params ? `?${params}` : ""}`),
  acknowledge: (id: number, notes?: string) => api.post(`/api/defacement/incidents/${id}/acknowledge`, { notes }),
  scan: () => api.post("/api/defacement/scan"),
}

// Vulnerability
export const vulnApi = {
  stats: () => api.get<{ data: unknown }>("/api/vulnerability/stats"),
  summary: (params?: string) => api.get<{ data: unknown; total: number }>(`/api/vulnerability/summary${params ? `?${params}` : ""}`),
  website: (id: number) => api.get<{ data: unknown }>(`/api/vulnerability/websites/${id}`),
  detail: (id: number) => api.get<{ data: unknown }>(`/api/vulnerability/websites/${id}`),
  history: (id: number, limit?: number) => api.get<{ data: unknown }>(`/api/vulnerability/websites/${id}/history?limit=${limit || 10}`),
  scan: (id: number) => api.post(`/api/vulnerability/websites/${id}/scan`),
  scanAll: () => api.post("/api/vulnerability/scan-all"),
  progress: () => api.get<{ data: unknown }>("/api/vulnerability/progress"),
}

// DNS
export const dnsApi = {
  summary: () => api.get<{ data: DNSScanRecord[] }>("/api/dns/summary"),
}

// Settings
export const settingsApi = {
  notifications: () => api.get<{ data: unknown }>("/api/settings/notifications"),
  updateTelegram: (data: unknown) => api.put("/api/settings/notifications/telegram", data),
  updateEmail: (data: unknown) => api.put("/api/settings/notifications/email", data),
  updateWebhook: (data: unknown) => api.put("/api/settings/notifications/webhook", data),
  digest: () => api.get<{ data: DigestSettings }>("/api/settings/notifications/digest"),
  updateDigest: (data: Partial<DigestSettings>) => api.put<{ data: DigestSettings }>("/api/settings/notifications/digest", data),
  monitoring: () => api.get<{ data: unknown }>("/api/settings/monitoring"),
  updateMonitoring: (data: unknown) => api.put("/api/settings/monitoring", data),
  ai: () => api.get<{ data: AISettings }>("/api/settings/ai"),
  updateAI: (data: Partial<AISettings>) => api.put("/api/settings/ai", data),
}

// Public Status
export const statusApi = {
  overview: () => fetch(`${API_BASE}/status`).then(r => r.json()),
  services: () => fetch(`${API_BASE}/status/services`).then(r => r.json()),
  serviceHistory: (id: number, days?: number) => fetch(`${API_BASE}/status/services/${id}/history?days=${days || 90}`).then(r => r.json()),
  incidents: (limit?: number) => fetch(`${API_BASE}/status/incidents?limit=${limit || 20}`).then(r => r.json()),
  maintenance: () => fetch(`${API_BASE}/status/maintenance`).then(r => r.json()),
}

// Types
export interface User {
  id: number
  username: string
  email: string
  full_name: string
  phone?: string
  role: string
  is_active: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

export interface UserCreate {
  username: string
  email: string
  password: string
  full_name: string
  phone?: string
  role?: string
}

export interface UserUpdate {
  email?: string
  full_name?: string
  phone?: string
  is_active?: boolean
}

export interface Website {
  id: number
  url: string
  name: string
  description?: string
  opd_id?: number
  opd_name?: string
  check_interval: number
  timeout: number
  is_active: boolean
  status: string
  last_status_code?: number
  last_response_time?: number
  last_checked_at?: string
  ssl_valid?: boolean
  ssl_expiry_date?: string
  content_clean?: boolean
  last_scan_at?: string
  security_score?: number
  security_grade?: string
  vuln_risk_score?: number
  vuln_risk_level?: string
  created_at: string
  updated_at: string
}

export interface WebsiteCreate {
  url: string
  name: string
  description?: string
  opd_id?: number
  check_interval?: number
  timeout?: number
  is_active?: boolean
}

export interface UptimeEntry {
  checked_at: string
  status: string
  status_code: number
  response_time: number
}

export interface Alert {
  id: number
  website_id: number
  website_name?: string
  type: string
  severity: string
  title: string
  message: string
  context?: unknown
  is_resolved: boolean
  resolved_at?: string
  resolved_by?: number
  resolution_note?: string
  is_acknowledged: boolean
  acknowledged_at?: string
  acknowledged_by?: number
  created_at: string
}

export interface AlertSummary {
  total_active: number
  critical: number
  warning: number
  info: number
}

export interface DashboardOverview {
  stats: DashboardStats
  recent_alerts: Alert[]
  status_distribution: Record<string, number>
}

export interface DashboardStats {
  total_websites: number
  total_up: number
  total_down: number
  total_degraded: number
  content_issues: number
  ssl_expiring_soon: number
  avg_response_time: number
  overall_uptime: number
}

export interface DashboardTrends {
  response_times: { date: string; avg: number }[]
  uptime_history: { date: string; uptime: number }[]
}

export interface OPD {
  id: number
  name: string
  code: string
  contact_email?: string
  contact_phone?: string
}

export interface Keyword {
  id: number
  keyword: string
  category: string
  is_regex: boolean
  is_active: boolean
  weight: number
}

export interface Maintenance {
  id: number
  title: string
  description?: string
  status: string
  scheduled_start: string
  scheduled_end: string
  actual_start?: string
  actual_end?: string
  created_at: string
}

export interface MaintenanceCreate {
  title: string
  description?: string
  scheduled_start: string
  scheduled_end: string
  website_ids?: number[]
}

export interface SystemStatus {
  status: string
  version: string
  total_websites: number
  monitor_running: boolean
  scheduler: string
  telegram_enabled: boolean
  last_check: string
  server_uptime?: string
  db_status?: string
  active_jobs?: string[]
}

export interface AuditLog {
  id: number
  user_id?: number
  username?: string
  action: string
  resource_type: string
  resource_id?: number
  details?: unknown
  ip_address?: string
  user_agent?: string
  created_at: string
}

export interface BulkImportResult {
  created: string[] | null
  skipped: string[] | null
  failed: { keyword?: string; name?: string; url?: string; error: string }[] | null
}

export interface ResponseTimePercentiles {
  p50: number
  p95: number
  p99: number
  avg: number
  min: number
  max: number
  count: number
}

export interface WebsiteMetrics {
  last_24h: ResponseTimePercentiles
  last_7d: ResponseTimePercentiles
  last_30d: ResponseTimePercentiles
}

export interface DNSRecord {
  type: string
  name: string
  value: string
  ttl: number
}

export interface SubdomainResult {
  subdomain: string
  ip?: string
  status_code?: number
  title?: string
  found_at: string
  source: string
}

export interface DNSScanResult {
  website_id: number
  domain: string
  records: DNSRecord[]
  subdomains: SubdomainResult[]
  nameservers: string[]
  mx_records: string[]
  spf_record?: string
  dmarc_record?: string
  scanned_at: string
  scan_duration_ms: number
}

export interface DNSScanRecord {
  id: number
  website_id: number
  domain_name: string
  has_spf: boolean
  has_dmarc: boolean
  spf_record?: string
  dmarc_record?: string
  nameservers: string[]
  mx_records: string[]
  dns_records: DNSRecord[]
  subdomains: SubdomainResult[]
  subdomain_count: number
  scan_duration_ms: number
  created_at: string
}

export interface AISettings {
  enabled: boolean
  provider: string
  api_key: string
  model: string
}

export interface DigestSettings {
  digest_enabled: boolean
  digest_interval: number
  quiet_hours_start: string
  quiet_hours_end: string
}

export interface DorkCategory {
  name: string
  value: string
  count?: number
  description?: string
}

export interface DorkDetectionDetail {
  id: number
  website_id: number
  website_name?: string
  website_url?: string
  pattern_id: number
  pattern_name?: string
  pattern_matched?: string
  matched_content?: string
  matched_text?: string
  context?: string
  location?: string
  category: string
  severity: string
  confidence?: number
  ai_verified?: boolean
  url?: string
  detected_at: string
  status: string
  is_false_positive: boolean
  is_resolved: boolean
  resolved_at?: string
  resolved_by?: string
  notes?: string
  scan_id?: number
  snippet?: string
}

export { ApiError }
