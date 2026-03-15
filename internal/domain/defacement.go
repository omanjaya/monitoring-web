package domain

import "time"

type DefacementSource string

const (
	DefacementSourceZoneH    DefacementSource = "zone_h"
	DefacementSourceZoneXSec DefacementSource = "zone_xsec"
)

type DefacementIncident struct {
	ID             int64            `db:"id" json:"id"`
	WebsiteID      int64            `db:"website_id" json:"website_id"`
	WebsiteName    string           `json:"website_name,omitempty"`
	WebsiteURL     string           `json:"website_url,omitempty"`
	Source         DefacementSource `db:"source" json:"source"`
	SourceID       string           `db:"source_id" json:"source_id,omitempty"`
	DefacedURL     string           `db:"defaced_url" json:"defaced_url"`
	Attacker       string           `db:"attacker" json:"attacker,omitempty"`
	Team           string           `db:"team" json:"team,omitempty"`
	DefacedAt      *time.Time       `db:"defaced_at" json:"defaced_at,omitempty"`
	MirrorURL      string           `db:"mirror_url" json:"mirror_url,omitempty"`
	IsAcknowledged bool             `db:"is_acknowledged" json:"is_acknowledged"`
	AcknowledgedAt *time.Time       `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
	AcknowledgedBy string           `db:"acknowledged_by" json:"acknowledged_by,omitempty"`
	Notes          string           `db:"notes" json:"notes,omitempty"`
	CreatedAt      time.Time        `db:"created_at" json:"created_at"`
}

type DefacementScan struct {
	ID           int64            `db:"id" json:"id"`
	Source       DefacementSource `db:"source" json:"source"`
	Status       string           `db:"status" json:"status"`
	TotalChecked int              `db:"total_checked" json:"total_checked"`
	NewIncidents int              `db:"new_incidents" json:"new_incidents"`
	StartedAt    time.Time        `db:"started_at" json:"started_at"`
	CompletedAt  *time.Time       `db:"completed_at" json:"completed_at,omitempty"`
	ErrorMessage string           `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time        `db:"created_at" json:"created_at"`
}

type DefacementStats struct {
	TotalIncidents      int                         `json:"total_incidents"`
	UnacknowledgedCount int                         `json:"unacknowledged_count"`
	WebsitesAffected    int                         `json:"websites_affected"`
	BySource            map[DefacementSource]int    `json:"by_source"`
	RecentIncidents     []DefacementIncident        `json:"recent_incidents,omitempty"`
	LastScanAt          *time.Time                  `json:"last_scan_at,omitempty"`
}
