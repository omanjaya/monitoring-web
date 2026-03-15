package domain

import (
	"time"
)

type MaintenanceStatus string

const (
	MaintenanceScheduled  MaintenanceStatus = "scheduled"
	MaintenanceInProgress MaintenanceStatus = "in_progress"
	MaintenanceCompleted  MaintenanceStatus = "completed"
	MaintenanceCancelled  MaintenanceStatus = "cancelled"
)

// MaintenanceWindow represents a scheduled maintenance window
type MaintenanceWindow struct {
	ID          int64             `db:"id" json:"id"`
	WebsiteID   NullInt64     `db:"website_id" json:"website_id,omitempty"` // NULL = all websites
	Title       string            `db:"title" json:"title"`
	Description NullString    `db:"description" json:"description,omitempty"`
	Status      MaintenanceStatus `db:"status" json:"status"`
	StartTime   time.Time         `db:"start_time" json:"start_time"`
	EndTime     time.Time         `db:"end_time" json:"end_time"`
	CreatedBy   int64             `db:"created_by" json:"created_by"`
	CreatedAt   time.Time         `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `db:"updated_at" json:"updated_at"`

	// Relations (optional)
	WebsiteName string `db:"website_name" json:"website_name,omitempty"`
	CreatorName string `db:"creator_name" json:"creator_name,omitempty"`
}

// MaintenanceWindowCreate is the input for creating a maintenance window
type MaintenanceWindowCreate struct {
	WebsiteID   *int64 `json:"website_id"`
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	StartTime   string `json:"start_time" binding:"required"` // RFC3339 format
	EndTime     string `json:"end_time" binding:"required"`   // RFC3339 format
}

// MaintenanceWindowUpdate is the input for updating a maintenance window
type MaintenanceWindowUpdate struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	StartTime   *string `json:"start_time"`
	EndTime     *string `json:"end_time"`
	Status      *string `json:"status"`
}

// MaintenanceFilter for querying maintenance windows
type MaintenanceFilter struct {
	WebsiteID   *int64
	Status      *MaintenanceStatus
	IncludePast bool
	Page        int
	Limit       int
}

// IsActive checks if the maintenance window is currently active
func (m *MaintenanceWindow) IsActive() bool {
	now := time.Now()
	return m.Status == MaintenanceInProgress ||
		(m.Status == MaintenanceScheduled && now.After(m.StartTime) && now.Before(m.EndTime))
}

// PublicMaintenanceWindow represents maintenance info for public display
type PublicMaintenanceWindow struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	ServiceName string    `json:"service_name,omitempty"` // Empty = all services
}
