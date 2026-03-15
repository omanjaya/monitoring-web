package domain

import (
	"encoding/json"
	"time"
)

// AuditLog represents an audit/activity log entry
type AuditLog struct {
	ID           int64           `db:"id" json:"id"`
	UserID       NullInt64   `db:"user_id" json:"user_id"`
	Username     NullString  `db:"username" json:"username"`
	Action       string          `db:"action" json:"action"`
	ResourceType string          `db:"resource_type" json:"resource_type"`
	ResourceID   NullInt64   `db:"resource_id" json:"resource_id"`
	Details      json.RawMessage `db:"details" json:"details"`
	IPAddress    NullString  `db:"ip_address" json:"ip_address"`
	UserAgent    NullString  `db:"user_agent" json:"user_agent"`
	CreatedAt    time.Time       `db:"created_at" json:"created_at"`
}

// AuditFilter holds filter parameters for querying audit logs
type AuditFilter struct {
	UserID       *int64  `json:"user_id"`
	Action       *string `json:"action"`
	ResourceType *string `json:"resource_type"`
	Page         int     `json:"page"`
	Limit        int     `json:"limit"`
}
