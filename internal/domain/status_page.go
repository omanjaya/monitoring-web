package domain

import "time"

// PublicStatusOverview represents the overall system status for public display
type PublicStatusOverview struct {
	SystemStatus   string              `json:"system_status"` // operational, degraded, partial_outage, major_outage
	OverallUptime  float64             `json:"overall_uptime"`
	TotalWebsites  int                 `json:"total_websites"`
	OperationalCnt int                 `json:"operational_count"`
	DegradedCnt    int                 `json:"degraded_count"`
	DownCnt        int                 `json:"down_count"`
	LastUpdated    time.Time           `json:"last_updated"`
	Services       []PublicServiceStatus `json:"services"`
}

// PublicServiceStatus represents a single service/website status for public display
type PublicServiceStatus struct {
	ID               int64     `json:"id"`
	Name             string    `json:"name"`
	URL              string    `json:"url"`
	Status           string    `json:"status"` // operational, degraded, down
	ResponseTime     int       `json:"response_time,omitempty"`
	UptimePercentage float64   `json:"uptime_percentage"`
	LastChecked      time.Time `json:"last_checked,omitempty"`
}

// PublicIncident represents an incident for public display
type PublicIncident struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Status      string    `json:"status"` // investigating, identified, monitoring, resolved
	Impact      string    `json:"impact"` // none, minor, major, critical
	Message     string    `json:"message"`
	ServiceName string    `json:"service_name"`
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
}

// PublicUptimeHistory represents uptime history for a service
type PublicUptimeHistory struct {
	ServiceID   int64                `json:"service_id"`
	ServiceName string               `json:"service_name"`
	Period      string               `json:"period"` // 24h, 7d, 30d, 90d
	Uptime      float64              `json:"uptime_percentage"`
	DataPoints  []UptimeHistoryPoint `json:"data_points"`
}

// UptimeHistoryPoint represents a single point in uptime history
type UptimeHistoryPoint struct {
	Date   string `json:"date"` // YYYY-MM-DD or YYYY-MM-DD HH:00
	Status string `json:"status"` // operational, degraded, down
	Uptime float64 `json:"uptime_percentage"`
}

// StatusPageConfig represents configuration for the public status page
type StatusPageConfig struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	LogoURL     string `json:"logo_url,omitempty"`
	FaviconURL  string `json:"favicon_url,omitempty"`
	CustomCSS   string `json:"custom_css,omitempty"`
}
