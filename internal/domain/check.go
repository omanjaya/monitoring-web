package domain

import (
	"time"
)

type CheckStatus string

const (
	CheckStatusUp      CheckStatus = "up"
	CheckStatusDown    CheckStatus = "down"
	CheckStatusDegraded CheckStatus = "degraded"
	CheckStatusTimeout CheckStatus = "timeout"
	CheckStatusError   CheckStatus = "error"
)

// Check represents an uptime check result
type Check struct {
	ID           int64          `db:"id" json:"id"`
	WebsiteID    int64          `db:"website_id" json:"website_id"`
	StatusCode   NullInt32  `db:"status_code" json:"status_code,omitempty"`
	ResponseTime NullInt32  `db:"response_time" json:"response_time,omitempty"` // ms
	Status       CheckStatus    `db:"status" json:"status"`
	ErrorMessage NullString `db:"error_message" json:"error_message,omitempty"`
	ErrorType    NullString `db:"error_type" json:"error_type,omitempty"`
	ContentLength NullInt32 `db:"content_length" json:"content_length,omitempty"`
	CheckedAt    time.Time      `db:"checked_at" json:"checked_at"`

	// Protocol and network info (not persisted to DB, populated at check time)
	Protocol      string   `db:"-" json:"protocol,omitempty"`       // e.g. "HTTP/1.1", "HTTP/2.0"
	IPv4Addresses []string `db:"-" json:"ipv4_addresses,omitempty"` // resolved IPv4 addresses
	IPv6Addresses []string `db:"-" json:"ipv6_addresses,omitempty"` // resolved IPv6 addresses
	SupportsIPv6  bool     `db:"-" json:"supports_ipv6,omitempty"`  // true if domain has AAAA records
}

// SSLCheck represents an SSL certificate check result
type SSLCheck struct {
	ID              int64          `db:"id" json:"id"`
	WebsiteID       int64          `db:"website_id" json:"website_id"`
	IsValid         bool           `db:"is_valid" json:"is_valid"`
	Issuer          NullString `db:"issuer" json:"issuer,omitempty"`
	Subject         NullString `db:"subject" json:"subject,omitempty"`
	ValidFrom       NullTime   `db:"valid_from" json:"valid_from,omitempty"`
	ValidUntil      NullTime   `db:"valid_until" json:"valid_until,omitempty"`
	DaysUntilExpiry NullInt32  `db:"days_until_expiry" json:"days_until_expiry,omitempty"`
	Protocol        NullString `db:"protocol" json:"protocol,omitempty"`
	ErrorMessage    NullString `db:"error_message" json:"error_message,omitempty"`
	CheckedAt       time.Time      `db:"checked_at" json:"checked_at"`
}

// ContentScan represents a content scan result
type ContentScan struct {
	ID            int64          `db:"id" json:"id"`
	WebsiteID     int64          `db:"website_id" json:"website_id"`
	IsClean       bool           `db:"is_clean" json:"is_clean"`
	ScanType      string         `db:"scan_type" json:"scan_type"` // quick, full
	Findings      NullString `db:"findings" json:"findings,omitempty"` // JSON
	PageTitle     NullString `db:"page_title" json:"page_title,omitempty"`
	PageHash      NullString `db:"page_hash" json:"page_hash,omitempty"`
	KeywordsFound int            `db:"keywords_found" json:"keywords_found"`
	IframesFound  int            `db:"iframes_found" json:"iframes_found"`
	RedirectsFound int           `db:"redirects_found" json:"redirects_found"`
	ScannedAt     time.Time      `db:"scanned_at" json:"scanned_at"`
}

// ContentFinding represents a single finding in content scan
type ContentFinding struct {
	Type        string `json:"type"`                  // keyword, iframe, redirect, dork_pattern
	Category    string `json:"category"`              // gambling, defacement, suspicious, webshell, malware, etc.
	Value       string `json:"value"`                 // the keyword or URL found
	Location    string `json:"location"`              // body, meta, script, etc.
	Snippet     string `json:"snippet"`               // context around the finding
	PatternName string `json:"pattern_name,omitempty"` // name of the dork pattern that matched (if any)
	Severity    string `json:"severity,omitempty"`     // severity from dork pattern (critical, high, medium, low)
}

// UptimeStats represents uptime statistics
type UptimeStats struct {
	WebsiteID         int64   `json:"website_id"`
	Period            string  `json:"period"` // 24h, 7d, 30d
	UptimePercentage  float64 `json:"uptime_percentage"`
	TotalChecks       int     `json:"total_checks"`
	UpCount           int     `json:"up_count"`
	DownCount         int     `json:"down_count"`
	AvgResponseTime   float64 `json:"avg_response_time"`
	MinResponseTime   int     `json:"min_response_time"`
	MaxResponseTime   int     `json:"max_response_time"`
}

// ResponseTimePercentiles represents response time distribution metrics
type ResponseTimePercentiles struct {
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	P99   float64 `json:"p99"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// WebsiteMetrics represents performance metrics for a website over multiple time periods
type WebsiteMetrics struct {
	WebsiteID   int64                           `json:"website_id"`
	WebsiteName string                          `json:"website_name"`
	Periods     map[string]*ResponseTimePercentiles `json:"periods"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalWebsites     int     `db:"total_websites" json:"total_websites"`
	TotalUp           int     `db:"total_up" json:"total_up"`
	TotalDown         int     `db:"total_down" json:"total_down"`
	TotalDegraded     int     `db:"total_degraded" json:"total_degraded"`
	OverallUptime     float64 `db:"-" json:"overall_uptime"`
	SSLExpiringSoon   int     `db:"ssl_expiring_soon" json:"ssl_expiring_soon"`
	ContentIssues     int     `db:"content_issues" json:"content_issues"`
	ActiveAlerts      int     `db:"-" json:"active_alerts"`
	AvgResponseTime   float64 `db:"avg_response_time" json:"avg_response_time"`
}

// UptimeChartData represents data points for uptime charts
type UptimeChartData struct {
	WebsiteID        int64             `json:"website_id"`
	Period           string            `json:"period"`
	DataPoints       []ChartDataPoint  `json:"data_points"`
	ResponseTimes    []ResponseTimePoint `json:"response_times"`
	UptimePercentage float64           `json:"uptime_percentage"`
}

// ChartDataPoint represents a single data point for status chart
type ChartDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
	UpCount   int       `json:"up_count"`
	DownCount int       `json:"down_count"`
}

// ResponseTimePoint represents a single data point for response time chart
type ResponseTimePoint struct {
	Timestamp       time.Time `json:"timestamp"`
	AvgResponseTime float64   `json:"avg_response_time"`
	MinResponseTime int       `json:"min_response_time"`
	MaxResponseTime int       `json:"max_response_time"`
}
