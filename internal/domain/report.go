package domain

import "time"

// ReportType represents the type of report
type ReportType string

const (
	ReportTypeUptime       ReportType = "uptime"
	ReportTypeSSL          ReportType = "ssl"
	ReportTypeContentScan  ReportType = "content_scan"
	ReportTypeSecurity     ReportType = "security"
	ReportTypeAlerts       ReportType = "alerts"
	ReportTypeComprehensive ReportType = "comprehensive"
)

// ReportFormat represents the output format
type ReportFormat string

const (
	ReportFormatPDF   ReportFormat = "pdf"
	ReportFormatExcel ReportFormat = "excel"
	ReportFormatCSV   ReportFormat = "csv"
)

// ReportRequest represents a request to generate a report
type ReportRequest struct {
	Type       ReportType   `json:"type" binding:"required"`
	Format     ReportFormat `json:"format" binding:"required"`
	StartDate  time.Time    `json:"start_date" binding:"required"`
	EndDate    time.Time    `json:"end_date" binding:"required"`
	WebsiteIDs []int64      `json:"website_ids,omitempty"` // Empty means all websites
	OPDID      *int64       `json:"opd_id,omitempty"`      // Filter by OPD
}

// ReportMetadata contains report metadata
type ReportMetadata struct {
	ID          string       `json:"id"`
	Type        ReportType   `json:"type"`
	Format      ReportFormat `json:"format"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	GeneratedAt time.Time    `json:"generated_at"`
	GeneratedBy string       `json:"generated_by"`
	Period      ReportPeriod `json:"period"`
	FileName    string       `json:"file_name"`
	FileSize    int64        `json:"file_size"`
}

// ReportPeriod represents the time period for a report
type ReportPeriod struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Days      int       `json:"days"`
}

// UptimeReportData contains data for uptime reports
type UptimeReportData struct {
	Metadata      ReportMetadata       `json:"metadata"`
	Summary       UptimeReportSummary  `json:"summary"`
	WebsiteStats  []WebsiteUptimeStats `json:"website_stats"`
	DailyTrends   []DailyUptimeTrend   `json:"daily_trends"`
}

// UptimeReportSummary contains summary statistics for uptime report
type UptimeReportSummary struct {
	TotalWebsites       int     `json:"total_websites"`
	AverageUptime       float64 `json:"average_uptime"`
	TotalChecks         int64   `json:"total_checks"`
	TotalDowntime       int64   `json:"total_downtime_minutes"`
	AverageResponseTime float64 `json:"average_response_time_ms"`
	WorstPerforming     string  `json:"worst_performing_website"`
	BestPerforming      string  `json:"best_performing_website"`
}

// WebsiteUptimeStats contains uptime stats for a single website
type WebsiteUptimeStats struct {
	WebsiteID       int64   `json:"website_id"`
	WebsiteName     string  `json:"website_name"`
	URL             string  `json:"url"`
	OPDName         string  `json:"opd_name"`
	UptimePercent   float64 `json:"uptime_percent"`
	TotalChecks     int64   `json:"total_checks"`
	SuccessChecks   int64   `json:"success_checks"`
	FailedChecks    int64   `json:"failed_checks"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	MinResponseTime float64 `json:"min_response_time_ms"`
	MaxResponseTime float64 `json:"max_response_time_ms"`
	CurrentStatus   string  `json:"current_status"`
}

// DailyUptimeTrend represents daily uptime trend data
type DailyUptimeTrend struct {
	Date            string  `json:"date"`
	UptimePercent   float64 `json:"uptime_percent"`
	TotalChecks     int64   `json:"total_checks"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
}

// SSLReportData contains data for SSL reports
type SSLReportData struct {
	Metadata     ReportMetadata    `json:"metadata"`
	Summary      SSLReportSummary  `json:"summary"`
	Certificates []SSLCertDetails  `json:"certificates"`
}

// SSLReportSummary contains summary for SSL report
type SSLReportSummary struct {
	TotalWebsites      int `json:"total_websites"`
	ValidCertificates  int `json:"valid_certificates"`
	ExpiringSoon       int `json:"expiring_soon"` // Within 30 days
	Expired            int `json:"expired"`
	NoCertificate      int `json:"no_certificate"`
	AverageDaysToExpiry int `json:"average_days_to_expiry"`
}

// SSLCertDetails contains SSL certificate details for report
type SSLCertDetails struct {
	WebsiteID     int64     `json:"website_id"`
	WebsiteName   string    `json:"website_name"`
	URL           string    `json:"url"`
	OPDName       string    `json:"opd_name"`
	Issuer        string    `json:"issuer"`
	ValidFrom     time.Time `json:"valid_from"`
	ValidUntil    time.Time `json:"valid_until"`
	DaysToExpiry  int       `json:"days_to_expiry"`
	Status        string    `json:"status"` // valid, expiring_soon, expired, invalid
	Grade         string    `json:"grade"`
}

// SecurityReportData contains data for security reports
type SecurityReportData struct {
	Metadata      ReportMetadata         `json:"metadata"`
	Summary       SecurityReportSummary  `json:"summary"`
	WebsiteSecurity []WebsiteSecuritySummary `json:"website_security"`
}

// SecurityReportSummary contains summary for security report
type SecurityReportSummary struct {
	TotalWebsites     int     `json:"total_websites"`
	AverageScore      float64 `json:"average_score"`
	GradeACount       int     `json:"grade_a_count"`
	GradeBCount       int     `json:"grade_b_count"`
	GradeCCount       int     `json:"grade_c_count"`
	GradeDCount       int     `json:"grade_d_count"`
	GradeFCount       int     `json:"grade_f_count"`
	MostMissingHeader string  `json:"most_missing_header"`
}

// WebsiteSecuritySummary contains security summary for a website
type WebsiteSecuritySummary struct {
	WebsiteID      int64    `json:"website_id"`
	WebsiteName    string   `json:"website_name"`
	URL            string   `json:"url"`
	OPDName        string   `json:"opd_name"`
	SecurityScore  int      `json:"security_score"`
	SecurityGrade  string   `json:"security_grade"`
	HeadersPresent []string `json:"headers_present"`
	HeadersMissing []string `json:"headers_missing"`
}

// AlertsReportData contains data for alerts reports
type AlertsReportData struct {
	Metadata     ReportMetadata       `json:"metadata"`
	Summary      AlertsReportSummary  `json:"summary"`
	AlertsList   []AlertReportItem    `json:"alerts_list"`
	ByType       []AlertTypeCount     `json:"by_type"`
	BySeverity   []AlertSeverityCount `json:"by_severity"`
}

// AlertsReportSummary contains summary for alerts report
type AlertsReportSummary struct {
	TotalAlerts     int     `json:"total_alerts"`
	ResolvedAlerts  int     `json:"resolved_alerts"`
	UnresolvedAlerts int    `json:"unresolved_alerts"`
	CriticalCount   int     `json:"critical_count"`
	WarningCount    int     `json:"warning_count"`
	InfoCount       int     `json:"info_count"`
	AvgResolutionHours float64 `json:"avg_resolution_hours"`
}

// AlertReportItem represents an alert item in report
type AlertReportItem struct {
	ID          int64     `json:"id"`
	WebsiteName string    `json:"website_name"`
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
	ResolvedAt  *time.Time `json:"resolved_at,omitempty"`
	IsResolved  bool      `json:"is_resolved"`
}

// AlertTypeCount represents alert count by type
type AlertTypeCount struct {
	Type  string `json:"type"`
	Count int    `json:"count"`
}

// AlertSeverityCount represents alert count by severity
type AlertSeverityCount struct {
	Severity string `json:"severity"`
	Count    int    `json:"count"`
}

// ContentScanReportData contains data for content scan reports
type ContentScanReportData struct {
	Metadata    ReportMetadata          `json:"metadata"`
	Summary     ContentScanReportSummary `json:"summary"`
	ScanResults []WebsiteContentScan    `json:"scan_results"`
}

// ContentScanReportSummary contains summary for content scan report
type ContentScanReportSummary struct {
	TotalWebsites    int `json:"total_websites"`
	CleanWebsites    int `json:"clean_websites"`
	InfectedWebsites int `json:"infected_websites"`
	TotalKeywords    int `json:"total_keywords_found"`
	TotalIframes     int `json:"total_iframes_found"`
	TotalRedirects   int `json:"total_redirects_found"`
}

// WebsiteContentScan contains content scan results for a website
type WebsiteContentScan struct {
	WebsiteID      int64  `json:"website_id"`
	WebsiteName    string `json:"website_name"`
	URL            string `json:"url"`
	OPDName        string `json:"opd_name"`
	IsClean        bool   `json:"is_clean"`
	KeywordsFound  int    `json:"keywords_found"`
	IframesFound   int    `json:"iframes_found"`
	RedirectsFound int    `json:"redirects_found"`
	PageTitle      string `json:"page_title"`
	Status         string `json:"status"` // clean, infected
	LastScanAt     string `json:"last_scan_at"`
}

// ComprehensiveReportData contains all report data
type ComprehensiveReportData struct {
	Metadata    ReportMetadata         `json:"metadata"`
	Uptime      UptimeReportData       `json:"uptime"`
	SSL         SSLReportData          `json:"ssl"`
	Security    SecurityReportData     `json:"security"`
	ContentScan ContentScanReportData  `json:"content_scan"`
	Alerts      AlertsReportData       `json:"alerts"`
}
