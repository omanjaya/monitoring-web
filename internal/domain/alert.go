package domain

import (
	"time"
)

type AlertType string

const (
	AlertTypeDown           AlertType = "down"
	AlertTypeUp             AlertType = "up"
	AlertTypeSSLExpiring    AlertType = "ssl_expiring"
	AlertTypeSSLExpired     AlertType = "ssl_expired"
	AlertTypeJudolDetected  AlertType = "judol_detected"
	AlertTypeDefacement     AlertType = "defacement"
	AlertTypeSlowResponse   AlertType = "slow_response"
	AlertTypeSecurityIssue  AlertType = "security_issue"
)

type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "info"
	SeverityWarning  AlertSeverity = "warning"
	SeverityCritical AlertSeverity = "critical"
)

// Alert represents a monitoring alert
type Alert struct {
	ID             int64          `db:"id" json:"id"`
	WebsiteID      int64          `db:"website_id" json:"website_id"`
	Type           AlertType      `db:"type" json:"type"`
	Severity       AlertSeverity  `db:"severity" json:"severity"`
	Title          string         `db:"title" json:"title"`
	Message        string         `db:"message" json:"message"`
	Context        NullString `db:"context" json:"context,omitempty"` // JSON

	// Resolution
	IsResolved     bool           `db:"is_resolved" json:"is_resolved"`
	ResolvedAt     NullTime   `db:"resolved_at" json:"resolved_at,omitempty"`
	ResolvedBy     NullInt64  `db:"resolved_by" json:"resolved_by,omitempty"`
	ResolutionNote NullString `db:"resolution_note" json:"resolution_note,omitempty"`

	// Acknowledgement
	IsAcknowledged  bool          `db:"is_acknowledged" json:"is_acknowledged"`
	AcknowledgedAt  NullTime  `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	AcknowledgedBy  NullInt64 `db:"acknowledged_by" json:"acknowledged_by,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`

	// Escalation fields
	EscalationLevel  NullInt32 `db:"escalation_level" json:"escalation_level,omitempty"`
	LastEscalatedAt  NullTime  `db:"last_escalated_at" json:"last_escalated_at,omitempty"`
	EscalationCount  NullInt32 `db:"escalation_count" json:"escalation_count,omitempty"`
	PolicyID         NullInt64 `db:"policy_id" json:"policy_id,omitempty"`

	// Relations
	Website *Website `db:"-" json:"website,omitempty"`
}

// AlertCreate is the input for creating a new alert
type AlertCreate struct {
	WebsiteID int64         `json:"website_id"`
	Type      AlertType     `json:"type"`
	Severity  AlertSeverity `json:"severity"`
	Title     string        `json:"title"`
	Message   string        `json:"message"`
	Context   interface{}   `json:"context,omitempty"`
}

// AlertFilter for querying alerts
type AlertFilter struct {
	WebsiteID  *int64
	Type       *AlertType
	Severity   *AlertSeverity
	IsResolved *bool
	StartDate  *time.Time
	EndDate    *time.Time
	Page       int
	Limit      int
}

// AlertSummary provides a summary of alerts
type AlertSummary struct {
	TotalActive int `db:"total_active" json:"total_active"`
	Critical    int `db:"critical" json:"critical"`
	Warning     int `db:"warning" json:"warning"`
	Info        int `db:"info" json:"info"`
}

// Notification represents a sent notification
type Notification struct {
	ID           int64          `db:"id" json:"id"`
	AlertID      int64          `db:"alert_id" json:"alert_id"`
	Channel      string         `db:"channel" json:"channel"` // telegram, email
	Recipient    string         `db:"recipient" json:"recipient"`
	Status       string         `db:"status" json:"status"` // pending, sent, failed
	SentAt       NullTime   `db:"sent_at" json:"sent_at,omitempty"`
	ErrorMessage NullString `db:"error_message" json:"error_message,omitempty"`
	RetryCount   int            `db:"retry_count" json:"retry_count"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}
