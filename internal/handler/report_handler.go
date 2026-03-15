package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/service/report"
)

type ReportHandler struct {
	reportService *report.ReportService
}

func NewReportHandler(reportService *report.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

// GenerateReport generates a report based on the request
// POST /api/reports/generate
func (h *ReportHandler) GenerateReport(c *gin.Context) {
	var req domain.ReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// Validate dates
	if req.EndDate.Before(req.StartDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "End date must be after start date"})
		return
	}

	// Limit date range to 365 days
	if req.EndDate.Sub(req.StartDate).Hours() > 365*24 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Date range cannot exceed 365 days"})
		return
	}

	// Get username from context
	username := "system"
	if u, exists := c.Get("username"); exists {
		username = u.(string)
	}

	data, metadata, err := h.reportService.GenerateReport(c.Request.Context(), &req, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report: " + err.Error()})
		return
	}

	// Set appropriate content type and headers
	contentType := "application/octet-stream"
	switch req.Format {
	case domain.ReportFormatExcel:
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case domain.ReportFormatCSV:
		contentType = "text/csv"
	case domain.ReportFormatPDF:
		contentType = "application/pdf"
	}

	c.Header("Content-Disposition", "attachment; filename="+metadata.FileName)
	c.Header("Content-Type", contentType)
	c.Header("Content-Length", string(rune(len(data))))
	c.Header("X-Report-ID", metadata.ID)
	c.Data(http.StatusOK, contentType, data)
}

// GetReportTypes returns available report types
// GET /api/reports/types
func (h *ReportHandler) GetReportTypes(c *gin.Context) {
	types := []gin.H{
		{
			"type":        "uptime",
			"name":        "Uptime Report",
			"description": "Website uptime and response time statistics",
			"formats":     []string{"pdf", "excel", "csv"},
		},
		{
			"type":        "ssl",
			"name":        "SSL Certificate Report",
			"description": "SSL certificate status and expiration details",
			"formats":     []string{"pdf", "excel", "csv"},
		},
		{
			"type":        "security",
			"name":        "Security Headers Report",
			"description": "Website security header analysis",
			"formats":     []string{"pdf", "excel", "csv"},
		},
		{
			"type":        "alerts",
			"name":        "Alerts Report",
			"description": "Alert history and analysis",
			"formats":     []string{"pdf", "excel", "csv"},
		},
		{
			"type":        "content_scan",
			"name":        "Content Scan Report",
			"description": "Website content scan for gambling/defacement detection",
			"formats":     []string{"pdf", "excel", "csv"},
		},
		{
			"type":        "comprehensive",
			"name":        "Comprehensive Report",
			"description": "Complete monitoring overview including all aspects",
			"formats":     []string{"pdf", "excel"},
		},
	}

	c.JSON(http.StatusOK, gin.H{"data": types})
}

// GetQuickReport generates a quick report for common time periods
// GET /api/reports/quick/:type/:period
func (h *ReportHandler) GetQuickReport(c *gin.Context) {
	reportType := c.Param("type")
	period := c.Param("period")
	format := c.DefaultQuery("format", "excel")

	// Calculate date range based on period
	endDate := time.Now()
	var startDate time.Time

	switch period {
	case "today":
		startDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 0, 0, 0, 0, endDate.Location())
	case "yesterday":
		startDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day()-1, 0, 0, 0, 0, endDate.Location())
		endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day()-1, 23, 59, 59, 0, endDate.Location())
	case "week":
		startDate = endDate.AddDate(0, 0, -7)
	case "month":
		startDate = endDate.AddDate(0, -1, 0)
	case "quarter":
		startDate = endDate.AddDate(0, -3, 0)
	case "year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid period. Use: today, yesterday, week, month, quarter, year"})
		return
	}

	// Validate report type
	var rt domain.ReportType
	switch reportType {
	case "uptime":
		rt = domain.ReportTypeUptime
	case "ssl":
		rt = domain.ReportTypeSSL
	case "security":
		rt = domain.ReportTypeSecurity
	case "alerts":
		rt = domain.ReportTypeAlerts
	case "content_scan":
		rt = domain.ReportTypeContentScan
	case "comprehensive":
		rt = domain.ReportTypeComprehensive
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid report type"})
		return
	}

	// Validate format
	var rf domain.ReportFormat
	switch format {
	case "excel":
		rf = domain.ReportFormatExcel
	case "csv":
		rf = domain.ReportFormatCSV
	case "pdf":
		rf = domain.ReportFormatPDF
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid format. Use: pdf, excel, csv"})
		return
	}

	req := &domain.ReportRequest{
		Type:      rt,
		Format:    rf,
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Get username from context
	username := "system"
	if u, exists := c.Get("username"); exists {
		username = u.(string)
	}

	data, metadata, err := h.reportService.GenerateReport(c.Request.Context(), req, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate report: " + err.Error()})
		return
	}

	// Set appropriate content type
	contentType := "application/octet-stream"
	switch rf {
	case domain.ReportFormatExcel:
		contentType = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	case domain.ReportFormatCSV:
		contentType = "text/csv"
	case domain.ReportFormatPDF:
		contentType = "application/pdf"
	}

	c.Header("Content-Disposition", "attachment; filename="+metadata.FileName)
	c.Header("Content-Type", contentType)
	c.Header("X-Report-ID", metadata.ID)
	c.Data(http.StatusOK, contentType, data)
}

// GetScheduledReportOptions returns options for scheduled reports
// GET /api/reports/schedule/options
func (h *ReportHandler) GetScheduledReportOptions(c *gin.Context) {
	options := gin.H{
		"frequencies": []gin.H{
			{"value": "daily", "label": "Daily"},
			{"value": "weekly", "label": "Weekly"},
			{"value": "monthly", "label": "Monthly"},
		},
		"report_types": []string{"uptime", "ssl", "security", "alerts", "comprehensive"},
		"formats":      []string{"pdf", "excel", "csv"},
		"delivery_methods": []gin.H{
			{"value": "email", "label": "Email"},
			{"value": "download", "label": "Download Link"},
		},
	}

	c.JSON(http.StatusOK, options)
}
