package domain

import (
	"time"
)

type WebsiteStatus string

const (
	StatusUp       WebsiteStatus = "up"
	StatusDown     WebsiteStatus = "down"
	StatusDegraded WebsiteStatus = "degraded"
	StatusUnknown  WebsiteStatus = "unknown"
)

type Website struct {
	ID          int64      `db:"id" json:"id"`
	OPDID       NullInt64  `db:"opd_id" json:"opd_id,omitempty"`
	URL         string     `db:"url" json:"url"`
	Name        string     `db:"name" json:"name"`
	Description NullString `db:"description" json:"description,omitempty"`

	// Monitoring settings
	IsActive      bool `db:"is_active" json:"is_active"`
	CheckInterval int  `db:"check_interval" json:"check_interval"` // minutes
	Timeout       int  `db:"timeout" json:"timeout"`               // seconds

	// Current status (denormalized)
	Status           WebsiteStatus `db:"status" json:"status"`
	LastStatusCode   NullInt32     `db:"last_status_code" json:"last_status_code,omitempty"`
	LastResponseTime NullInt32     `db:"last_response_time" json:"last_response_time,omitempty"`
	LastCheckedAt    NullTime      `db:"last_checked_at" json:"last_checked_at,omitempty"`

	// SSL info
	SSLValid      NullBool `db:"ssl_valid" json:"ssl_valid,omitempty"`
	SSLExpiryDate NullTime `db:"ssl_expiry_date" json:"ssl_expiry_date,omitempty"`

	// Content scan info
	ContentClean bool     `db:"content_clean" json:"content_clean"`
	LastScanAt   NullTime `db:"last_scan_at" json:"last_scan_at,omitempty"`

	// Security info
	SecurityScore     NullInt32  `db:"security_score" json:"security_score,omitempty"`
	SecurityGrade     NullString `db:"security_grade" json:"security_grade,omitempty"`
	SecurityCheckedAt NullTime   `db:"security_checked_at" json:"security_checked_at,omitempty"`

	// Vulnerability info
	VulnRiskScore NullInt32  `db:"vuln_risk_score" json:"vuln_risk_score,omitempty"`
	VulnRiskLevel NullString `db:"vuln_risk_level" json:"vuln_risk_level,omitempty"`
	VulnLastScan  NullTime   `db:"vuln_last_scan" json:"vuln_last_scan,omitempty"`

	// Per-website response time thresholds (overrides global config)
	ResponseTimeWarning  NullInt32 `db:"response_time_warning" json:"response_time_warning,omitempty"`
	ResponseTimeCritical NullInt32 `db:"response_time_critical" json:"response_time_critical,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`

	// Relations (optional, loaded separately)
	OPD *OPD `db:"-" json:"opd,omitempty"`
}

// WebsiteCreate is the input for creating a new website
type WebsiteCreate struct {
	URL                  string `json:"url" binding:"required,url"`
	Name                 string `json:"name" binding:"required"`
	Description          string `json:"description"`
	OPDID                *int64 `json:"opd_id"`
	CheckInterval        int    `json:"check_interval"`
	Timeout              int    `json:"timeout"`
	IsActive             bool   `json:"is_active"`
	ResponseTimeWarning  *int   `json:"response_time_warning,omitempty"`
	ResponseTimeCritical *int   `json:"response_time_critical,omitempty"`
}

// WebsiteUpdate is the input for updating a website
type WebsiteUpdate struct {
	Name                 *string `json:"name"`
	Description          *string `json:"description"`
	OPDID                *int64  `json:"opd_id"`
	CheckInterval        *int    `json:"check_interval"`
	Timeout              *int    `json:"timeout"`
	IsActive             *bool   `json:"is_active"`
	ResponseTimeWarning  *int    `json:"response_time_warning,omitempty"`
	ResponseTimeCritical *int    `json:"response_time_critical,omitempty"`
}

// BulkWebsiteAction is the input for performing bulk actions on websites
type BulkWebsiteAction struct {
	IDs    []int64 `json:"ids" binding:"required,min=1"`
	Action string  `json:"action" binding:"required,oneof=enable disable delete"`
}

// WebsiteFilter for querying websites
type WebsiteFilter struct {
	Status       *WebsiteStatus
	OPDID        *int64
	IsActive     *bool
	ContentClean *bool
	Search       string
	Page         int
	Limit        int
}

// OPD represents Organisasi Perangkat Daerah
type OPD struct {
	ID           int64      `db:"id" json:"id"`
	Name         string     `db:"name" json:"name"`
	Code         string     `db:"code" json:"code"`
	ContactEmail NullString `db:"contact_email" json:"contact_email,omitempty"`
	ContactPhone NullString `db:"contact_phone" json:"contact_phone,omitempty"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
}
