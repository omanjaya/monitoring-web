package report

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-pdf/fpdf"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type ReportService struct {
	cfg         *config.Config
	websiteRepo *mysql.WebsiteRepository
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
}

func NewReportService(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *ReportService {
	return &ReportService{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
	}
}

// GenerateReport generates a report based on the request
func (s *ReportService) GenerateReport(ctx context.Context, req *domain.ReportRequest, username string) ([]byte, *domain.ReportMetadata, error) {
	metadata := &domain.ReportMetadata{
		ID:          uuid.New().String(),
		Type:        req.Type,
		Format:      req.Format,
		GeneratedAt: time.Now(),
		GeneratedBy: username,
		Period: domain.ReportPeriod{
			StartDate: req.StartDate,
			EndDate:   req.EndDate,
			Days:      int(req.EndDate.Sub(req.StartDate).Hours() / 24),
		},
	}

	var data []byte
	var err error

	switch req.Type {
	case domain.ReportTypeUptime:
		data, err = s.generateUptimeReport(ctx, req, metadata)
	case domain.ReportTypeSSL:
		data, err = s.generateSSLReport(ctx, req, metadata)
	case domain.ReportTypeSecurity:
		data, err = s.generateSecurityReport(ctx, req, metadata)
	case domain.ReportTypeAlerts:
		data, err = s.generateAlertsReport(ctx, req, metadata)
	case domain.ReportTypeContentScan:
		data, err = s.generateContentScanReport(ctx, req, metadata)
	case domain.ReportTypeComprehensive:
		data, err = s.generateComprehensiveReport(ctx, req, metadata)
	default:
		return nil, nil, fmt.Errorf("unsupported report type: %s", req.Type)
	}

	if err != nil {
		return nil, nil, err
	}

	metadata.FileSize = int64(len(data))
	return data, metadata, nil
}

func (s *ReportService) generateUptimeReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "Uptime Monitoring Report"
	metadata.Description = "Website uptime and response time statistics"

	// Get website stats
	websites, _, err := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	if err != nil {
		return nil, err
	}

	// Filter by website IDs if specified
	if len(req.WebsiteIDs) > 0 {
		filtered := make([]domain.Website, 0)
		for _, w := range websites {
			for _, id := range req.WebsiteIDs {
				if w.ID == id {
					filtered = append(filtered, w)
					break
				}
			}
		}
		websites = filtered
	}

	// Build report data
	reportData := &domain.UptimeReportData{
		Metadata:     *metadata,
		WebsiteStats: make([]domain.WebsiteUptimeStats, 0),
	}

	var totalUptime float64
	var totalResponseTime float64
	var totalChecks int64
	var bestUptime float64 = -1
	var worstUptime float64 = 101

	for _, w := range websites {
		// Get uptime stats for each website
		stats, err := s.checkRepo.GetUptimeStats(ctx, w.ID, req.StartDate)
		if err != nil {
			logger.Warn().Err(err).Int64("website_id", w.ID).Msg("Failed to get uptime stats")
			continue
		}

		uptimePercent := 100.0
		if stats.TotalChecks > 0 {
			uptimePercent = float64(stats.UpCount) / float64(stats.TotalChecks) * 100
		}

		websiteStat := domain.WebsiteUptimeStats{
			WebsiteID:       w.ID,
			WebsiteName:     w.Name,
			URL:             w.URL,
			OPDName:         "", // OPD lookup not implemented
			UptimePercent:   uptimePercent,
			TotalChecks:     int64(stats.TotalChecks),
			SuccessChecks:   int64(stats.UpCount),
			FailedChecks:    int64(stats.DownCount),
			AvgResponseTime: stats.AvgResponseTime,
			MinResponseTime: float64(stats.MinResponseTime),
			MaxResponseTime: float64(stats.MaxResponseTime),
			CurrentStatus:   string(w.Status),
		}
		reportData.WebsiteStats = append(reportData.WebsiteStats, websiteStat)

		totalUptime += uptimePercent
		totalResponseTime += stats.AvgResponseTime
		totalChecks += int64(stats.TotalChecks)

		if uptimePercent > bestUptime {
			bestUptime = uptimePercent
			reportData.Summary.BestPerforming = w.Name
		}
		if uptimePercent < worstUptime {
			worstUptime = uptimePercent
			reportData.Summary.WorstPerforming = w.Name
		}
	}

	// Calculate summary
	websiteCount := len(reportData.WebsiteStats)
	if websiteCount > 0 {
		reportData.Summary.TotalWebsites = websiteCount
		reportData.Summary.AverageUptime = totalUptime / float64(websiteCount)
		reportData.Summary.AverageResponseTime = totalResponseTime / float64(websiteCount)
		reportData.Summary.TotalChecks = totalChecks
	}

	// Generate output based on format
	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("uptime_report_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateUptimeExcel(reportData)
	case domain.ReportFormatCSV:
		metadata.FileName = fmt.Sprintf("uptime_report_%s.csv", time.Now().Format("20060102_150405"))
		return s.generateUptimeCSV(reportData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("uptime_report_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateUptimePDF(reportData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateUptimeExcel(data *domain.UptimeReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Summary sheet
	f.SetSheetName("Sheet1", "Summary")
	f.SetCellValue("Summary", "A1", "Uptime Monitoring Report")
	f.SetCellValue("Summary", "A3", "Report Period:")
	f.SetCellValue("Summary", "B3", fmt.Sprintf("%s - %s", data.Metadata.Period.StartDate.Format("2006-01-02"), data.Metadata.Period.EndDate.Format("2006-01-02")))
	f.SetCellValue("Summary", "A4", "Generated At:")
	f.SetCellValue("Summary", "B4", data.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))
	f.SetCellValue("Summary", "A5", "Generated By:")
	f.SetCellValue("Summary", "B5", data.Metadata.GeneratedBy)

	f.SetCellValue("Summary", "A7", "Summary Statistics")
	f.SetCellValue("Summary", "A8", "Total Websites:")
	f.SetCellValue("Summary", "B8", data.Summary.TotalWebsites)
	f.SetCellValue("Summary", "A9", "Average Uptime:")
	f.SetCellValue("Summary", "B9", fmt.Sprintf("%.2f%%", data.Summary.AverageUptime))
	f.SetCellValue("Summary", "A10", "Total Checks:")
	f.SetCellValue("Summary", "B10", data.Summary.TotalChecks)
	f.SetCellValue("Summary", "A11", "Avg Response Time:")
	f.SetCellValue("Summary", "B11", fmt.Sprintf("%.2f ms", data.Summary.AverageResponseTime))
	f.SetCellValue("Summary", "A12", "Best Performing:")
	f.SetCellValue("Summary", "B12", data.Summary.BestPerforming)
	f.SetCellValue("Summary", "A13", "Worst Performing:")
	f.SetCellValue("Summary", "B13", data.Summary.WorstPerforming)

	// Website Details sheet
	f.NewSheet("Website Details")
	headers := []string{"Website Name", "URL", "OPD", "Uptime %", "Total Checks", "Success", "Failed", "Avg Response (ms)", "Min Response (ms)", "Max Response (ms)", "Status"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Website Details", cell, h)
	}

	for i, ws := range data.WebsiteStats {
		row := i + 2
		f.SetCellValue("Website Details", fmt.Sprintf("A%d", row), ws.WebsiteName)
		f.SetCellValue("Website Details", fmt.Sprintf("B%d", row), ws.URL)
		f.SetCellValue("Website Details", fmt.Sprintf("C%d", row), ws.OPDName)
		f.SetCellValue("Website Details", fmt.Sprintf("D%d", row), fmt.Sprintf("%.2f", ws.UptimePercent))
		f.SetCellValue("Website Details", fmt.Sprintf("E%d", row), ws.TotalChecks)
		f.SetCellValue("Website Details", fmt.Sprintf("F%d", row), ws.SuccessChecks)
		f.SetCellValue("Website Details", fmt.Sprintf("G%d", row), ws.FailedChecks)
		f.SetCellValue("Website Details", fmt.Sprintf("H%d", row), fmt.Sprintf("%.2f", ws.AvgResponseTime))
		f.SetCellValue("Website Details", fmt.Sprintf("I%d", row), fmt.Sprintf("%.2f", ws.MinResponseTime))
		f.SetCellValue("Website Details", fmt.Sprintf("J%d", row), fmt.Sprintf("%.2f", ws.MaxResponseTime))
		f.SetCellValue("Website Details", fmt.Sprintf("K%d", row), ws.CurrentStatus)
	}

	// Auto-fit columns
	for i := 1; i <= len(headers); i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth("Website Details", col, col, 15)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateUptimeCSV(data *domain.UptimeReportData) ([]byte, error) {
	var buf bytes.Buffer

	// Header
	buf.WriteString("Website Name,URL,OPD,Uptime %,Total Checks,Success,Failed,Avg Response (ms),Min Response (ms),Max Response (ms),Status\n")

	for _, ws := range data.WebsiteStats {
		buf.WriteString(fmt.Sprintf("\"%s\",\"%s\",\"%s\",%.2f,%d,%d,%d,%.2f,%.2f,%.2f,%s\n",
			ws.WebsiteName, ws.URL, ws.OPDName, ws.UptimePercent,
			ws.TotalChecks, ws.SuccessChecks, ws.FailedChecks,
			ws.AvgResponseTime, ws.MinResponseTime, ws.MaxResponseTime,
			ws.CurrentStatus))
	}

	return buf.Bytes(), nil
}

func (s *ReportService) generateSSLReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "SSL Certificate Report"
	metadata.Description = "SSL certificate status and expiration details"

	websites, _, err := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	if err != nil {
		return nil, err
	}

	reportData := &domain.SSLReportData{
		Metadata:     *metadata,
		Certificates: make([]domain.SSLCertDetails, 0),
	}

	for _, w := range websites {
		sslCheck, err := s.checkRepo.GetLatestSSLCheck(ctx, w.ID)
		if err != nil || sslCheck == nil {
			reportData.Summary.NoCertificate++
			continue
		}

		daysToExpiry := 0
		if sslCheck.ValidUntil.Valid {
			daysToExpiry = int(sslCheck.ValidUntil.Time.Sub(time.Now()).Hours() / 24)
		}
		status := "valid"
		if daysToExpiry <= 0 {
			status = "expired"
			reportData.Summary.Expired++
		} else if daysToExpiry <= 30 {
			status = "expiring_soon"
			reportData.Summary.ExpiringSoon++
		} else {
			reportData.Summary.ValidCertificates++
		}

		// Get OPD name if available
		opdName := ""
		if w.OPD != nil {
			opdName = w.OPD.Name
		}

		// Extract values from sql.Null types
		issuer := ""
		if sslCheck.Issuer.Valid {
			issuer = sslCheck.Issuer.String
		}
		var validFrom, validUntil time.Time
		if sslCheck.ValidFrom.Valid {
			validFrom = sslCheck.ValidFrom.Time
		}
		if sslCheck.ValidUntil.Valid {
			validUntil = sslCheck.ValidUntil.Time
		}

		cert := domain.SSLCertDetails{
			WebsiteID:    w.ID,
			WebsiteName:  w.Name,
			URL:          w.URL,
			OPDName:      opdName,
			Issuer:       issuer,
			ValidFrom:    validFrom,
			ValidUntil:   validUntil,
			DaysToExpiry: daysToExpiry,
			Status:       status,
			Grade:        "", // SSLCheck doesn't have Grade field
		}
		reportData.Certificates = append(reportData.Certificates, cert)
	}

	reportData.Summary.TotalWebsites = len(websites)

	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("ssl_report_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateSSLExcel(reportData)
	case domain.ReportFormatCSV:
		metadata.FileName = fmt.Sprintf("ssl_report_%s.csv", time.Now().Format("20060102_150405"))
		return s.generateSSLCSV(reportData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("ssl_report_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateSSLPDF(reportData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateSSLExcel(data *domain.SSLReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Summary sheet
	f.SetSheetName("Sheet1", "Summary")
	f.SetCellValue("Summary", "A1", "SSL Certificate Report")
	f.SetCellValue("Summary", "A3", "Generated At:")
	f.SetCellValue("Summary", "B3", data.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))

	f.SetCellValue("Summary", "A5", "Summary Statistics")
	f.SetCellValue("Summary", "A6", "Total Websites:")
	f.SetCellValue("Summary", "B6", data.Summary.TotalWebsites)
	f.SetCellValue("Summary", "A7", "Valid Certificates:")
	f.SetCellValue("Summary", "B7", data.Summary.ValidCertificates)
	f.SetCellValue("Summary", "A8", "Expiring Soon (30 days):")
	f.SetCellValue("Summary", "B8", data.Summary.ExpiringSoon)
	f.SetCellValue("Summary", "A9", "Expired:")
	f.SetCellValue("Summary", "B9", data.Summary.Expired)
	f.SetCellValue("Summary", "A10", "No Certificate:")
	f.SetCellValue("Summary", "B10", data.Summary.NoCertificate)

	// Certificates sheet
	f.NewSheet("Certificates")
	headers := []string{"Website Name", "URL", "OPD", "Issuer", "Valid From", "Valid Until", "Days to Expiry", "Status", "Grade"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Certificates", cell, h)
	}

	for i, cert := range data.Certificates {
		row := i + 2
		f.SetCellValue("Certificates", fmt.Sprintf("A%d", row), cert.WebsiteName)
		f.SetCellValue("Certificates", fmt.Sprintf("B%d", row), cert.URL)
		f.SetCellValue("Certificates", fmt.Sprintf("C%d", row), cert.OPDName)
		f.SetCellValue("Certificates", fmt.Sprintf("D%d", row), cert.Issuer)
		f.SetCellValue("Certificates", fmt.Sprintf("E%d", row), cert.ValidFrom.Format("2006-01-02"))
		f.SetCellValue("Certificates", fmt.Sprintf("F%d", row), cert.ValidUntil.Format("2006-01-02"))
		f.SetCellValue("Certificates", fmt.Sprintf("G%d", row), cert.DaysToExpiry)
		f.SetCellValue("Certificates", fmt.Sprintf("H%d", row), cert.Status)
		f.SetCellValue("Certificates", fmt.Sprintf("I%d", row), cert.Grade)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateSSLCSV(data *domain.SSLReportData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("Website Name,URL,OPD,Issuer,Valid From,Valid Until,Days to Expiry,Status,Grade\n")

	for _, cert := range data.Certificates {
		buf.WriteString(fmt.Sprintf("\"%s\",\"%s\",\"%s\",\"%s\",%s,%s,%d,%s,%s\n",
			cert.WebsiteName, cert.URL, cert.OPDName, cert.Issuer,
			cert.ValidFrom.Format("2006-01-02"), cert.ValidUntil.Format("2006-01-02"),
			cert.DaysToExpiry, cert.Status, cert.Grade))
	}

	return buf.Bytes(), nil
}

func (s *ReportService) generateSecurityReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "Security Headers Report"
	metadata.Description = "Website security header analysis"

	websites, _, err := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	if err != nil {
		return nil, err
	}

	reportData := &domain.SecurityReportData{
		Metadata:        *metadata,
		WebsiteSecurity: make([]domain.WebsiteSecuritySummary, 0),
	}

	var totalScore float64

	for _, w := range websites {
		secCheck, err := s.checkRepo.GetLatestSecurityHeaderCheck(ctx, w.ID)
		if err != nil || secCheck == nil {
			continue
		}

		// Get OPD name if available
		opdName := ""
		if w.OPD != nil {
			opdName = w.OPD.Name
		}

		ws := domain.WebsiteSecuritySummary{
			WebsiteID:      w.ID,
			WebsiteName:    w.Name,
			URL:            w.URL,
			OPDName:        opdName,
			SecurityScore:  secCheck.Score,
			SecurityGrade:  secCheck.Grade,
			HeadersPresent: []string{}, // Not directly available in SecurityHeaderCheck
			HeadersMissing: []string{}, // Not directly available in SecurityHeaderCheck
		}
		reportData.WebsiteSecurity = append(reportData.WebsiteSecurity, ws)
		totalScore += float64(secCheck.Score)

		// Count grades
		switch secCheck.Grade {
		case "A", "A+":
			reportData.Summary.GradeACount++
		case "B":
			reportData.Summary.GradeBCount++
		case "C":
			reportData.Summary.GradeCCount++
		case "D":
			reportData.Summary.GradeDCount++
		default:
			reportData.Summary.GradeFCount++
		}
	}

	reportData.Summary.TotalWebsites = len(reportData.WebsiteSecurity)
	if reportData.Summary.TotalWebsites > 0 {
		reportData.Summary.AverageScore = totalScore / float64(reportData.Summary.TotalWebsites)
	}

	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("security_report_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateSecurityExcel(reportData)
	case domain.ReportFormatCSV:
		metadata.FileName = fmt.Sprintf("security_report_%s.csv", time.Now().Format("20060102_150405"))
		return s.generateSecurityCSV(reportData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("security_report_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateSecurityPDF(reportData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateSecurityExcel(data *domain.SecurityReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Summary sheet
	f.SetSheetName("Sheet1", "Summary")
	f.SetCellValue("Summary", "A1", "Security Headers Report")
	f.SetCellValue("Summary", "A3", "Generated At:")
	f.SetCellValue("Summary", "B3", data.Metadata.GeneratedAt.Format("2006-01-02 15:04:05"))

	f.SetCellValue("Summary", "A5", "Summary Statistics")
	f.SetCellValue("Summary", "A6", "Total Websites:")
	f.SetCellValue("Summary", "B6", data.Summary.TotalWebsites)
	f.SetCellValue("Summary", "A7", "Average Score:")
	f.SetCellValue("Summary", "B7", fmt.Sprintf("%.1f", data.Summary.AverageScore))
	f.SetCellValue("Summary", "A8", "Grade A:")
	f.SetCellValue("Summary", "B8", data.Summary.GradeACount)
	f.SetCellValue("Summary", "A9", "Grade B:")
	f.SetCellValue("Summary", "B9", data.Summary.GradeBCount)
	f.SetCellValue("Summary", "A10", "Grade C:")
	f.SetCellValue("Summary", "B10", data.Summary.GradeCCount)
	f.SetCellValue("Summary", "A11", "Grade D:")
	f.SetCellValue("Summary", "B11", data.Summary.GradeDCount)
	f.SetCellValue("Summary", "A12", "Grade F:")
	f.SetCellValue("Summary", "B12", data.Summary.GradeFCount)
	f.SetCellValue("Summary", "A13", "Most Missing Header:")
	f.SetCellValue("Summary", "B13", data.Summary.MostMissingHeader)

	// Details sheet
	f.NewSheet("Security Details")
	headers := []string{"Website Name", "URL", "OPD", "Score", "Grade", "Headers Present", "Headers Missing"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Security Details", cell, h)
	}

	for i, ws := range data.WebsiteSecurity {
		row := i + 2
		f.SetCellValue("Security Details", fmt.Sprintf("A%d", row), ws.WebsiteName)
		f.SetCellValue("Security Details", fmt.Sprintf("B%d", row), ws.URL)
		f.SetCellValue("Security Details", fmt.Sprintf("C%d", row), ws.OPDName)
		f.SetCellValue("Security Details", fmt.Sprintf("D%d", row), ws.SecurityScore)
		f.SetCellValue("Security Details", fmt.Sprintf("E%d", row), ws.SecurityGrade)
		f.SetCellValue("Security Details", fmt.Sprintf("F%d", row), fmt.Sprintf("%v", ws.HeadersPresent))
		f.SetCellValue("Security Details", fmt.Sprintf("G%d", row), fmt.Sprintf("%v", ws.HeadersMissing))
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateSecurityCSV(data *domain.SecurityReportData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("Website Name,URL,OPD,Score,Grade,Headers Present,Headers Missing\n")

	for _, ws := range data.WebsiteSecurity {
		buf.WriteString(fmt.Sprintf("\"%s\",\"%s\",\"%s\",%d,%s,\"%v\",\"%v\"\n",
			ws.WebsiteName, ws.URL, ws.OPDName,
			ws.SecurityScore, ws.SecurityGrade,
			ws.HeadersPresent, ws.HeadersMissing))
	}

	return buf.Bytes(), nil
}

func (s *ReportService) generateAlertsReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "Alerts Report"
	metadata.Description = "Alert history and analysis"

	// Get alerts within the date range
	filter := domain.AlertFilter{
		StartDate: &req.StartDate,
		EndDate:   &req.EndDate,
		Limit:     -1, // No limit for reports
	}

	alerts, _, err := s.alertRepo.GetAll(ctx, filter)
	if err != nil {
		return nil, err
	}

	reportData := &domain.AlertsReportData{
		Metadata:   *metadata,
		AlertsList: make([]domain.AlertReportItem, 0),
		ByType:     make([]domain.AlertTypeCount, 0),
		BySeverity: make([]domain.AlertSeverityCount, 0),
	}

	typeCount := make(map[string]int)
	severityCount := make(map[string]int)
	var totalResolutionTime float64
	resolvedCount := 0

	for _, a := range alerts {
		// Get website name
		website, _ := s.websiteRepo.GetByID(ctx, a.WebsiteID)
		websiteName := ""
		if website != nil {
			websiteName = website.Name
		}

		item := domain.AlertReportItem{
			ID:          a.ID,
			WebsiteName: websiteName,
			Type:        string(a.Type),
			Severity:    string(a.Severity),
			Title:       a.Title,
			Message:     a.Message,
			CreatedAt:   a.CreatedAt,
			IsResolved:  a.IsResolved,
		}
		if a.ResolvedAt.Valid {
			item.ResolvedAt = &a.ResolvedAt.Time
			resolutionTime := a.ResolvedAt.Time.Sub(a.CreatedAt).Hours()
			totalResolutionTime += resolutionTime
			resolvedCount++
		}
		reportData.AlertsList = append(reportData.AlertsList, item)

		typeCount[string(a.Type)]++
		severityCount[string(a.Severity)]++

		if a.IsResolved {
			reportData.Summary.ResolvedAlerts++
		} else {
			reportData.Summary.UnresolvedAlerts++
		}

		switch a.Severity {
		case domain.SeverityCritical:
			reportData.Summary.CriticalCount++
		case domain.SeverityWarning:
			reportData.Summary.WarningCount++
		case domain.SeverityInfo:
			reportData.Summary.InfoCount++
		}
	}

	reportData.Summary.TotalAlerts = len(alerts)
	if resolvedCount > 0 {
		reportData.Summary.AvgResolutionHours = totalResolutionTime / float64(resolvedCount)
	}

	for t, count := range typeCount {
		reportData.ByType = append(reportData.ByType, domain.AlertTypeCount{Type: t, Count: count})
	}
	for s, count := range severityCount {
		reportData.BySeverity = append(reportData.BySeverity, domain.AlertSeverityCount{Severity: s, Count: count})
	}

	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("alerts_report_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateAlertsExcel(reportData)
	case domain.ReportFormatCSV:
		metadata.FileName = fmt.Sprintf("alerts_report_%s.csv", time.Now().Format("20060102_150405"))
		return s.generateAlertsCSV(reportData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("alerts_report_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateAlertsPDF(reportData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateAlertsExcel(data *domain.AlertsReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Summary sheet
	f.SetSheetName("Sheet1", "Summary")
	f.SetCellValue("Summary", "A1", "Alerts Report")
	f.SetCellValue("Summary", "A3", "Report Period:")
	f.SetCellValue("Summary", "B3", fmt.Sprintf("%s - %s", data.Metadata.Period.StartDate.Format("2006-01-02"), data.Metadata.Period.EndDate.Format("2006-01-02")))

	f.SetCellValue("Summary", "A5", "Summary Statistics")
	f.SetCellValue("Summary", "A6", "Total Alerts:")
	f.SetCellValue("Summary", "B6", data.Summary.TotalAlerts)
	f.SetCellValue("Summary", "A7", "Resolved:")
	f.SetCellValue("Summary", "B7", data.Summary.ResolvedAlerts)
	f.SetCellValue("Summary", "A8", "Unresolved:")
	f.SetCellValue("Summary", "B8", data.Summary.UnresolvedAlerts)
	f.SetCellValue("Summary", "A9", "Critical:")
	f.SetCellValue("Summary", "B9", data.Summary.CriticalCount)
	f.SetCellValue("Summary", "A10", "Warning:")
	f.SetCellValue("Summary", "B10", data.Summary.WarningCount)
	f.SetCellValue("Summary", "A11", "Info:")
	f.SetCellValue("Summary", "B11", data.Summary.InfoCount)
	f.SetCellValue("Summary", "A12", "Avg Resolution Time:")
	f.SetCellValue("Summary", "B12", fmt.Sprintf("%.1f hours", data.Summary.AvgResolutionHours))

	// Alerts list sheet
	f.NewSheet("Alerts")
	headers := []string{"ID", "Website", "Type", "Severity", "Title", "Message", "Created At", "Resolved At", "Status"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Alerts", cell, h)
	}

	for i, alert := range data.AlertsList {
		row := i + 2
		f.SetCellValue("Alerts", fmt.Sprintf("A%d", row), alert.ID)
		f.SetCellValue("Alerts", fmt.Sprintf("B%d", row), alert.WebsiteName)
		f.SetCellValue("Alerts", fmt.Sprintf("C%d", row), alert.Type)
		f.SetCellValue("Alerts", fmt.Sprintf("D%d", row), alert.Severity)
		f.SetCellValue("Alerts", fmt.Sprintf("E%d", row), alert.Title)
		f.SetCellValue("Alerts", fmt.Sprintf("F%d", row), alert.Message)
		f.SetCellValue("Alerts", fmt.Sprintf("G%d", row), alert.CreatedAt.Format("2006-01-02 15:04:05"))
		if alert.ResolvedAt != nil {
			f.SetCellValue("Alerts", fmt.Sprintf("H%d", row), alert.ResolvedAt.Format("2006-01-02 15:04:05"))
		}
		status := "Unresolved"
		if alert.IsResolved {
			status = "Resolved"
		}
		f.SetCellValue("Alerts", fmt.Sprintf("I%d", row), status)
	}

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateAlertsCSV(data *domain.AlertsReportData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("ID,Website,Type,Severity,Title,Message,Created At,Resolved At,Status\n")

	for _, alert := range data.AlertsList {
		resolvedAt := ""
		if alert.ResolvedAt != nil {
			resolvedAt = alert.ResolvedAt.Format("2006-01-02 15:04:05")
		}
		status := "Unresolved"
		if alert.IsResolved {
			status = "Resolved"
		}
		buf.WriteString(fmt.Sprintf("%d,\"%s\",%s,%s,\"%s\",\"%s\",%s,%s,%s\n",
			alert.ID, alert.WebsiteName, alert.Type, alert.Severity,
			alert.Title, alert.Message,
			alert.CreatedAt.Format("2006-01-02 15:04:05"), resolvedAt, status))
	}

	return buf.Bytes(), nil
}

func (s *ReportService) generateContentScanReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "Laporan Content Scan"
	metadata.Description = "Hasil pemindaian konten website (Judol/Defacement)"

	contentScanData, err := s.generateContentScanReportData(ctx, req)
	if err != nil {
		return nil, err
	}
	contentScanData.Metadata = *metadata

	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("content_scan_report_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateContentScanExcel(contentScanData)
	case domain.ReportFormatCSV:
		metadata.FileName = fmt.Sprintf("content_scan_report_%s.csv", time.Now().Format("20060102_150405"))
		return s.generateContentScanCSV(contentScanData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("content_scan_report_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateContentScanPDF(contentScanData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateContentScanExcel(data *domain.ContentScanReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Header style
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Summary sheet
	f.SetSheetName("Sheet1", "Ringkasan")
	f.SetCellValue("Ringkasan", "A1", "Laporan Content Scan (Judol/Defacement)")
	f.SetCellValue("Ringkasan", "A3", "Total Website:")
	f.SetCellValue("Ringkasan", "B3", data.Summary.TotalWebsites)
	f.SetCellValue("Ringkasan", "A4", "Website Bersih:")
	f.SetCellValue("Ringkasan", "B4", data.Summary.CleanWebsites)
	f.SetCellValue("Ringkasan", "A5", "Website Terinfeksi:")
	f.SetCellValue("Ringkasan", "B5", data.Summary.InfectedWebsites)
	f.SetCellValue("Ringkasan", "A6", "Total Keywords Ditemukan:")
	f.SetCellValue("Ringkasan", "B6", data.Summary.TotalKeywords)
	f.SetCellValue("Ringkasan", "A7", "Total Iframes Mencurigakan:")
	f.SetCellValue("Ringkasan", "B7", data.Summary.TotalIframes)
	f.SetCellValue("Ringkasan", "A8", "Total Redirects Mencurigakan:")
	f.SetCellValue("Ringkasan", "B8", data.Summary.TotalRedirects)

	f.SetColWidth("Ringkasan", "A", "A", 25)
	f.SetColWidth("Ringkasan", "B", "B", 15)

	// Detail sheet
	f.NewSheet("Detail Scan")
	headers := []string{"No", "Nama Website", "URL", "OPD", "Status", "Keywords", "Iframes", "Redirects", "Terakhir Scan"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Detail Scan", cell, h)
		f.SetCellStyle("Detail Scan", cell, cell, headerStyle)
	}

	for i, scan := range data.ScanResults {
		row := i + 2
		f.SetCellValue("Detail Scan", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("Detail Scan", fmt.Sprintf("B%d", row), scan.WebsiteName)
		f.SetCellValue("Detail Scan", fmt.Sprintf("C%d", row), scan.URL)
		f.SetCellValue("Detail Scan", fmt.Sprintf("D%d", row), scan.OPDName)
		status := "BERSIH"
		if !scan.IsClean {
			status = "TERINFEKSI"
		}
		f.SetCellValue("Detail Scan", fmt.Sprintf("E%d", row), status)
		f.SetCellValue("Detail Scan", fmt.Sprintf("F%d", row), scan.KeywordsFound)
		f.SetCellValue("Detail Scan", fmt.Sprintf("G%d", row), scan.IframesFound)
		f.SetCellValue("Detail Scan", fmt.Sprintf("H%d", row), scan.RedirectsFound)
		f.SetCellValue("Detail Scan", fmt.Sprintf("I%d", row), scan.LastScanAt)
	}

	f.SetColWidth("Detail Scan", "A", "A", 5)
	f.SetColWidth("Detail Scan", "B", "B", 30)
	f.SetColWidth("Detail Scan", "C", "C", 40)
	f.SetColWidth("Detail Scan", "D", "D", 25)
	f.SetColWidth("Detail Scan", "E", "E", 12)
	f.SetColWidth("Detail Scan", "F", "H", 10)
	f.SetColWidth("Detail Scan", "I", "I", 20)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateContentScanCSV(data *domain.ContentScanReportData) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("No,Nama Website,URL,OPD,Status,Keywords Found,Iframes Found,Redirects Found,Last Scan\n")

	for i, scan := range data.ScanResults {
		status := "BERSIH"
		if !scan.IsClean {
			status = "TERINFEKSI"
		}
		buf.WriteString(fmt.Sprintf("%d,\"%s\",\"%s\",\"%s\",%s,%d,%d,%d,%s\n",
			i+1, scan.WebsiteName, scan.URL, scan.OPDName, status,
			scan.KeywordsFound, scan.IframesFound, scan.RedirectsFound,
			scan.LastScanAt))
	}

	return buf.Bytes(), nil
}

func (s *ReportService) generateComprehensiveReport(ctx context.Context, req *domain.ReportRequest, metadata *domain.ReportMetadata) ([]byte, error) {
	metadata.Title = "Laporan Monitoring Komprehensif"
	metadata.Description = "Laporan lengkap monitoring website: uptime, SSL, security, content scan, dan alerts"

	// Generate individual reports
	uptimeData, _ := s.generateUptimeReportData(ctx, req)
	sslData, _ := s.generateSSLReportData(ctx, req)
	securityData, _ := s.generateSecurityReportData(ctx, req)
	contentScanData, _ := s.generateContentScanReportData(ctx, req)
	alertsData, _ := s.generateAlertsReportData(ctx, req)

	switch req.Format {
	case domain.ReportFormatExcel:
		metadata.FileName = fmt.Sprintf("laporan_monitoring_%s.xlsx", time.Now().Format("20060102_150405"))
		return s.generateComprehensiveExcel(metadata, uptimeData, sslData, securityData, contentScanData, alertsData)
	case domain.ReportFormatPDF:
		metadata.FileName = fmt.Sprintf("laporan_monitoring_%s.pdf", time.Now().Format("20060102_150405"))
		return s.generateComprehensivePDF(metadata, uptimeData, sslData, securityData, contentScanData, alertsData)
	default:
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}
}

func (s *ReportService) generateUptimeReportData(ctx context.Context, req *domain.ReportRequest) (*domain.UptimeReportData, error) {
	websites, _, _ := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	data := &domain.UptimeReportData{WebsiteStats: make([]domain.WebsiteUptimeStats, 0)}

	for _, w := range websites {
		stats, _ := s.checkRepo.GetUptimeStats(ctx, w.ID, req.StartDate)
		if stats == nil {
			continue
		}
		uptimePercent := 100.0
		if stats.TotalChecks > 0 {
			uptimePercent = float64(stats.UpCount) / float64(stats.TotalChecks) * 100
		}
		data.WebsiteStats = append(data.WebsiteStats, domain.WebsiteUptimeStats{
			WebsiteName:     w.Name,
			URL:             w.URL,
			UptimePercent:   uptimePercent,
			AvgResponseTime: stats.AvgResponseTime,
		})
	}
	return data, nil
}

func (s *ReportService) generateSSLReportData(ctx context.Context, req *domain.ReportRequest) (*domain.SSLReportData, error) {
	websites, _, _ := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	data := &domain.SSLReportData{Certificates: make([]domain.SSLCertDetails, 0)}

	for _, w := range websites {
		sslCheck, _ := s.checkRepo.GetLatestSSLCheck(ctx, w.ID)
		if sslCheck == nil {
			continue
		}
		var validUntil time.Time
		daysToExpiry := 0
		if sslCheck.ValidUntil.Valid {
			validUntil = sslCheck.ValidUntil.Time
			daysToExpiry = int(sslCheck.ValidUntil.Time.Sub(time.Now()).Hours() / 24)
		}
		data.Certificates = append(data.Certificates, domain.SSLCertDetails{
			WebsiteName:  w.Name,
			ValidUntil:   validUntil,
			DaysToExpiry: daysToExpiry,
			Grade:        "", // SSLCheck doesn't have Grade field
		})
	}
	return data, nil
}

func (s *ReportService) generateSecurityReportData(ctx context.Context, req *domain.ReportRequest) (*domain.SecurityReportData, error) {
	websites, _, _ := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	data := &domain.SecurityReportData{WebsiteSecurity: make([]domain.WebsiteSecuritySummary, 0)}

	for _, w := range websites {
		secCheck, _ := s.checkRepo.GetLatestSecurityHeaderCheck(ctx, w.ID)
		if secCheck == nil {
			continue
		}
		data.WebsiteSecurity = append(data.WebsiteSecurity, domain.WebsiteSecuritySummary{
			WebsiteName:   w.Name,
			SecurityScore: secCheck.Score,
			SecurityGrade: secCheck.Grade,
		})
	}
	return data, nil
}

func (s *ReportService) generateContentScanReportData(ctx context.Context, req *domain.ReportRequest) (*domain.ContentScanReportData, error) {
	websites, _, _ := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{Limit: -1})
	data := &domain.ContentScanReportData{ScanResults: make([]domain.WebsiteContentScan, 0)}

	for _, w := range websites {
		scan, _ := s.checkRepo.GetLatestContentScan(ctx, w.ID)
		if scan == nil {
			continue
		}

		opdName := ""
		if w.OPD != nil {
			opdName = w.OPD.Name
		}

		status := "clean"
		if !scan.IsClean {
			status = "infected"
			data.Summary.InfectedWebsites++
		} else {
			data.Summary.CleanWebsites++
		}

		pageTitle := ""
		if scan.PageTitle.Valid {
			pageTitle = scan.PageTitle.String
		}

		data.Summary.TotalKeywords += scan.KeywordsFound
		data.Summary.TotalIframes += scan.IframesFound
		data.Summary.TotalRedirects += scan.RedirectsFound

		data.ScanResults = append(data.ScanResults, domain.WebsiteContentScan{
			WebsiteID:      w.ID,
			WebsiteName:    w.Name,
			URL:            w.URL,
			OPDName:        opdName,
			IsClean:        scan.IsClean,
			KeywordsFound:  scan.KeywordsFound,
			IframesFound:   scan.IframesFound,
			RedirectsFound: scan.RedirectsFound,
			PageTitle:      pageTitle,
			Status:         status,
			LastScanAt:     scan.ScannedAt.Format("2006-01-02 15:04:05"),
		})
	}

	data.Summary.TotalWebsites = len(data.ScanResults)
	return data, nil
}

func (s *ReportService) generateAlertsReportData(ctx context.Context, req *domain.ReportRequest) (*domain.AlertsReportData, error) {
	filter := domain.AlertFilter{StartDate: &req.StartDate, EndDate: &req.EndDate, Limit: -1}
	alerts, _, _ := s.alertRepo.GetAll(ctx, filter)

	data := &domain.AlertsReportData{AlertsList: make([]domain.AlertReportItem, 0)}
	for _, a := range alerts {
		website, _ := s.websiteRepo.GetByID(ctx, a.WebsiteID)
		websiteName := ""
		if website != nil {
			websiteName = website.Name
		}
		data.AlertsList = append(data.AlertsList, domain.AlertReportItem{
			WebsiteName: websiteName,
			Type:        string(a.Type),
			Severity:    string(a.Severity),
			Title:       a.Title,
			CreatedAt:   a.CreatedAt,
		})
	}
	data.Summary.TotalAlerts = len(alerts)
	return data, nil
}

func (s *ReportService) generateComprehensiveExcel(metadata *domain.ReportMetadata, uptime *domain.UptimeReportData, ssl *domain.SSLReportData, security *domain.SecurityReportData, contentScan *domain.ContentScanReportData, alerts *domain.AlertsReportData) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	// Define styles
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
		},
	})

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "1F4E79"},
	})

	// ==================== Overview Sheet ====================
	f.SetSheetName("Sheet1", "Ringkasan")
	f.SetCellValue("Ringkasan", "A1", "LAPORAN MONITORING WEBSITE")
	f.SetCellStyle("Ringkasan", "A1", "A1", titleStyle)
	f.SetCellValue("Ringkasan", "A2", "Pemerintah Provinsi Bali - Diskominfos")

	f.SetCellValue("Ringkasan", "A4", "Periode Laporan:")
	f.SetCellValue("Ringkasan", "B4", fmt.Sprintf("%s s/d %s", metadata.Period.StartDate.Format("02 Jan 2006"), metadata.Period.EndDate.Format("02 Jan 2006")))
	f.SetCellValue("Ringkasan", "A5", "Dibuat Pada:")
	f.SetCellValue("Ringkasan", "B5", metadata.GeneratedAt.Format("02 Jan 2006 15:04 WIB"))
	f.SetCellValue("Ringkasan", "A6", "Dibuat Oleh:")
	f.SetCellValue("Ringkasan", "B6", metadata.GeneratedBy)

	// Summary statistics
	f.SetCellValue("Ringkasan", "A8", "RINGKASAN STATISTIK")
	f.SetCellStyle("Ringkasan", "A8", "A8", titleStyle)

	f.SetCellValue("Ringkasan", "A10", "Total Website Dimonitor:")
	f.SetCellValue("Ringkasan", "B10", len(uptime.WebsiteStats))

	f.SetCellValue("Ringkasan", "A11", "Website Bersih (Content Scan):")
	f.SetCellValue("Ringkasan", "B11", contentScan.Summary.CleanWebsites)

	f.SetCellValue("Ringkasan", "A12", "Website Terinfeksi (Judol/Defacement):")
	f.SetCellValue("Ringkasan", "B12", contentScan.Summary.InfectedWebsites)

	f.SetCellValue("Ringkasan", "A13", "Total Alert Periode Ini:")
	f.SetCellValue("Ringkasan", "B13", alerts.Summary.TotalAlerts)

	f.SetColWidth("Ringkasan", "A", "A", 30)
	f.SetColWidth("Ringkasan", "B", "B", 40)

	// ==================== Uptime Sheet ====================
	f.NewSheet("Uptime")
	uptimeHeaders := []string{"No", "Nama Website", "URL", "Uptime (%)", "Avg Response (ms)", "Status"}
	for i, h := range uptimeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Uptime", cell, h)
		f.SetCellStyle("Uptime", cell, cell, headerStyle)
	}
	for i, ws := range uptime.WebsiteStats {
		row := i + 2
		f.SetCellValue("Uptime", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("Uptime", fmt.Sprintf("B%d", row), ws.WebsiteName)
		f.SetCellValue("Uptime", fmt.Sprintf("C%d", row), ws.URL)
		f.SetCellValue("Uptime", fmt.Sprintf("D%d", row), fmt.Sprintf("%.2f%%", ws.UptimePercent))
		f.SetCellValue("Uptime", fmt.Sprintf("E%d", row), fmt.Sprintf("%.0f", ws.AvgResponseTime))
		status := "UP"
		if ws.UptimePercent < 50 {
			status = "DOWN"
		} else if ws.UptimePercent < 90 {
			status = "DEGRADED"
		}
		f.SetCellValue("Uptime", fmt.Sprintf("F%d", row), status)
	}
	f.SetColWidth("Uptime", "A", "A", 5)
	f.SetColWidth("Uptime", "B", "B", 30)
	f.SetColWidth("Uptime", "C", "C", 40)
	f.SetColWidth("Uptime", "D", "F", 15)

	// ==================== SSL Sheet ====================
	f.NewSheet("SSL Certificate")
	sslHeaders := []string{"No", "Nama Website", "Berlaku Hingga", "Sisa Hari", "Status"}
	for i, h := range sslHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("SSL Certificate", cell, h)
		f.SetCellStyle("SSL Certificate", cell, cell, headerStyle)
	}
	for i, cert := range ssl.Certificates {
		row := i + 2
		f.SetCellValue("SSL Certificate", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("SSL Certificate", fmt.Sprintf("B%d", row), cert.WebsiteName)
		f.SetCellValue("SSL Certificate", fmt.Sprintf("C%d", row), cert.ValidUntil.Format("02 Jan 2006"))
		f.SetCellValue("SSL Certificate", fmt.Sprintf("D%d", row), cert.DaysToExpiry)
		status := "Valid"
		if cert.DaysToExpiry <= 0 {
			status = "EXPIRED"
		} else if cert.DaysToExpiry <= 30 {
			status = "Akan Expired"
		}
		f.SetCellValue("SSL Certificate", fmt.Sprintf("E%d", row), status)
	}
	f.SetColWidth("SSL Certificate", "A", "A", 5)
	f.SetColWidth("SSL Certificate", "B", "B", 30)
	f.SetColWidth("SSL Certificate", "C", "E", 15)

	// ==================== Security Sheet ====================
	f.NewSheet("Security Headers")
	secHeaders := []string{"No", "Nama Website", "Skor", "Grade", "Keterangan"}
	for i, h := range secHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Security Headers", cell, h)
		f.SetCellStyle("Security Headers", cell, cell, headerStyle)
	}
	for i, ws := range security.WebsiteSecurity {
		row := i + 2
		f.SetCellValue("Security Headers", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("Security Headers", fmt.Sprintf("B%d", row), ws.WebsiteName)
		f.SetCellValue("Security Headers", fmt.Sprintf("C%d", row), ws.SecurityScore)
		f.SetCellValue("Security Headers", fmt.Sprintf("D%d", row), ws.SecurityGrade)
		keterangan := "Baik"
		if ws.SecurityGrade == "F" {
			keterangan = "Perlu Perbaikan Segera"
		} else if ws.SecurityGrade == "D" || ws.SecurityGrade == "C" {
			keterangan = "Perlu Ditingkatkan"
		}
		f.SetCellValue("Security Headers", fmt.Sprintf("E%d", row), keterangan)
	}
	f.SetColWidth("Security Headers", "A", "A", 5)
	f.SetColWidth("Security Headers", "B", "B", 30)
	f.SetColWidth("Security Headers", "C", "D", 10)
	f.SetColWidth("Security Headers", "E", "E", 25)

	// ==================== Content Scan (Judol) Sheet ====================
	f.NewSheet("Content Scan")
	scanHeaders := []string{"No", "Nama Website", "URL", "Status", "Keywords", "Iframes", "Redirects", "Terakhir Scan"}
	for i, h := range scanHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Content Scan", cell, h)
		f.SetCellStyle("Content Scan", cell, cell, headerStyle)
	}
	for i, scan := range contentScan.ScanResults {
		row := i + 2
		f.SetCellValue("Content Scan", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("Content Scan", fmt.Sprintf("B%d", row), scan.WebsiteName)
		f.SetCellValue("Content Scan", fmt.Sprintf("C%d", row), scan.URL)
		status := "BERSIH"
		if !scan.IsClean {
			status = "TERINFEKSI"
		}
		f.SetCellValue("Content Scan", fmt.Sprintf("D%d", row), status)
		f.SetCellValue("Content Scan", fmt.Sprintf("E%d", row), scan.KeywordsFound)
		f.SetCellValue("Content Scan", fmt.Sprintf("F%d", row), scan.IframesFound)
		f.SetCellValue("Content Scan", fmt.Sprintf("G%d", row), scan.RedirectsFound)
		f.SetCellValue("Content Scan", fmt.Sprintf("H%d", row), scan.LastScanAt)
	}
	f.SetColWidth("Content Scan", "A", "A", 5)
	f.SetColWidth("Content Scan", "B", "B", 30)
	f.SetColWidth("Content Scan", "C", "C", 40)
	f.SetColWidth("Content Scan", "D", "D", 12)
	f.SetColWidth("Content Scan", "E", "G", 10)
	f.SetColWidth("Content Scan", "H", "H", 20)

	// ==================== Alerts Sheet ====================
	f.NewSheet("Alerts")
	alertHeaders := []string{"No", "Website", "Tipe", "Severity", "Judul", "Tanggal"}
	for i, h := range alertHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Alerts", cell, h)
		f.SetCellStyle("Alerts", cell, cell, headerStyle)
	}
	for i, alert := range alerts.AlertsList {
		row := i + 2
		f.SetCellValue("Alerts", fmt.Sprintf("A%d", row), i+1)
		f.SetCellValue("Alerts", fmt.Sprintf("B%d", row), alert.WebsiteName)
		f.SetCellValue("Alerts", fmt.Sprintf("C%d", row), alert.Type)
		f.SetCellValue("Alerts", fmt.Sprintf("D%d", row), alert.Severity)
		f.SetCellValue("Alerts", fmt.Sprintf("E%d", row), alert.Title)
		f.SetCellValue("Alerts", fmt.Sprintf("F%d", row), alert.CreatedAt.Format("02 Jan 2006 15:04"))
	}
	f.SetColWidth("Alerts", "A", "A", 5)
	f.SetColWidth("Alerts", "B", "B", 25)
	f.SetColWidth("Alerts", "C", "D", 12)
	f.SetColWidth("Alerts", "E", "E", 40)
	f.SetColWidth("Alerts", "F", "F", 18)

	// Set active sheet to Overview
	f.SetActiveSheet(0)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateComprehensivePDF(metadata *domain.ReportMetadata, uptime *domain.UptimeReportData, ssl *domain.SSLReportData, security *domain.SecurityReportData, contentScan *domain.ContentScanReportData, alerts *domain.AlertsReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// ==================== Cover Page ====================
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.Ln(30)
	pdf.CellFormat(0, 15, "LAPORAN MONITORING WEBSITE", "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 14)
	pdf.Ln(5)
	pdf.CellFormat(0, 8, "Pemerintah Provinsi Bali", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Dinas Komunikasi, Informatika dan Statistik", "", 1, "C", false, 0, "")

	pdf.Ln(20)
	pdf.SetFont("Helvetica", "", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Periode: %s s/d %s",
		metadata.Period.StartDate.Format("02 January 2006"),
		metadata.Period.EndDate.Format("02 January 2006")), "", 1, "C", false, 0, "")

	pdf.Ln(40)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Dibuat pada: %s", metadata.GeneratedAt.Format("02 January 2006 15:04 WIB")), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Dibuat oleh: %s", metadata.GeneratedBy), "", 1, "C", false, 0, "")

	// ==================== Summary Page ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN EKSEKUTIF")

	// Summary stats
	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	// Create summary table
	summaryData := [][]string{
		{"Total Website Dimonitor", fmt.Sprintf("%d website", len(uptime.WebsiteStats))},
		{"Website Bersih (Content Scan)", fmt.Sprintf("%d website", contentScan.Summary.CleanWebsites)},
		{"Website Terinfeksi", fmt.Sprintf("%d website", contentScan.Summary.InfectedWebsites)},
		{"Total Alert Periode Ini", fmt.Sprintf("%d alert", alerts.Summary.TotalAlerts)},
		{"SSL Akan Expired", fmt.Sprintf("%d sertifikat", ssl.Summary.ExpiringSoon)},
	}

	for _, row := range summaryData {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// ==================== Content Scan (Judol/Defacement) Page ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "HASIL SCAN KONTEN (Judol/Defacement)")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Hasil pemindaian konten website untuk mendeteksi konten judi online (judol) dan defacement. Website yang terinfeksi memerlukan penanganan segera.", "", "L", false)
	pdf.Ln(5)

	// Content scan summary
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(60, 7, "Total Website Dipindai", "1", 0, "L", false, 0, "")
	pdf.CellFormat(30, 7, fmt.Sprintf("%d", contentScan.Summary.TotalWebsites), "1", 1, "C", false, 0, "")

	pdf.CellFormat(60, 7, "Website Bersih", "1", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 128, 0)
	pdf.CellFormat(30, 7, fmt.Sprintf("%d", contentScan.Summary.CleanWebsites), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.CellFormat(60, 7, "Website Terinfeksi", "1", 0, "L", false, 0, "")
	if contentScan.Summary.InfectedWebsites > 0 {
		pdf.SetTextColor(255, 0, 0)
	}
	pdf.CellFormat(30, 7, fmt.Sprintf("%d", contentScan.Summary.InfectedWebsites), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// List infected websites if any
	if contentScan.Summary.InfectedWebsites > 0 {
		pdf.Ln(5)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 7, "Daftar Website Terinfeksi:", "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)

		// Table header
		pdf.SetFillColor(66, 114, 196)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(55, 7, "Nama Website", "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 7, "Keywords", "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 7, "Iframes", "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 7, "Redirects", "1", 0, "C", true, 0, "")
		pdf.CellFormat(57, 7, "URL", "1", 1, "C", true, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 8)

		no := 1
		for _, scan := range contentScan.ScanResults {
			if !scan.IsClean {
				pdf.CellFormat(8, 6, fmt.Sprintf("%d", no), "1", 0, "C", false, 0, "")
				name := scan.WebsiteName
				if len(name) > 25 {
					name = name[:22] + "..."
				}
				pdf.CellFormat(55, 6, name, "1", 0, "L", false, 0, "")
				pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.KeywordsFound), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.IframesFound), "1", 0, "C", false, 0, "")
				pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.RedirectsFound), "1", 0, "C", false, 0, "")
				urlTrunc := scan.URL
				if len(urlTrunc) > 30 {
					urlTrunc = urlTrunc[:27] + "..."
				}
				pdf.CellFormat(57, 6, urlTrunc, "1", 1, "L", false, 0, "")
				no++
			}
		}
	}

	// ==================== Uptime Report Page ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "LAPORAN UPTIME")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Status ketersediaan (uptime) website selama periode laporan.", "", "L", false)
	pdf.Ln(5)

	// Uptime table header
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 7, "Nama Website", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, "Uptime (%)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(35, 7, "Avg Response (ms)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(47, 7, "Status", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	for i, ws := range uptime.WebsiteStats {
		if i >= 30 {
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d website lainnya (lihat lampiran Excel untuk detail lengkap)", len(uptime.WebsiteStats)-30), "", 1, "L", false, 0, "")
			break
		}
		pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		name := ws.WebsiteName
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		pdf.CellFormat(60, 6, name, "1", 0, "L", false, 0, "")

		// Color code uptime
		if ws.UptimePercent >= 99 {
			pdf.SetTextColor(0, 128, 0)
		} else if ws.UptimePercent >= 90 {
			pdf.SetTextColor(255, 165, 0)
		} else {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f%%", ws.UptimePercent), "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)

		pdf.CellFormat(35, 6, fmt.Sprintf("%.0f", ws.AvgResponseTime), "1", 0, "C", false, 0, "")

		status := "UP"
		if ws.UptimePercent < 50 {
			status = "DOWN"
		} else if ws.UptimePercent < 90 {
			status = "DEGRADED"
		}
		pdf.CellFormat(47, 6, status, "1", 1, "C", false, 0, "")
	}

	// ==================== SSL Report Page ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "STATUS SERTIFIKAT SSL")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Status sertifikat SSL/TLS untuk masing-masing website.", "", "L", false)
	pdf.Ln(5)

	// SSL summary
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(50, 7, "Sertifikat Valid", "1", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 128, 0)
	pdf.CellFormat(25, 7, fmt.Sprintf("%d", ssl.Summary.ValidCertificates), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.CellFormat(50, 7, "Akan Expired (30 hari)", "1", 0, "L", false, 0, "")
	if ssl.Summary.ExpiringSoon > 0 {
		pdf.SetTextColor(255, 165, 0)
	}
	pdf.CellFormat(25, 7, fmt.Sprintf("%d", ssl.Summary.ExpiringSoon), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	pdf.CellFormat(50, 7, "Sudah Expired", "1", 0, "L", false, 0, "")
	if ssl.Summary.Expired > 0 {
		pdf.SetTextColor(255, 0, 0)
	}
	pdf.CellFormat(25, 7, fmt.Sprintf("%d", ssl.Summary.Expired), "1", 1, "C", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// List expiring/expired SSL if any
	if ssl.Summary.ExpiringSoon > 0 || ssl.Summary.Expired > 0 {
		pdf.Ln(5)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(0, 7, "Sertifikat Perlu Perhatian:", "", 1, "L", false, 0, "")

		pdf.SetFillColor(66, 114, 196)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(70, 7, "Nama Website", "1", 0, "C", true, 0, "")
		pdf.CellFormat(40, 7, "Berlaku Hingga", "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, "Sisa Hari", "1", 0, "C", true, 0, "")
		pdf.CellFormat(37, 7, "Status", "1", 1, "C", true, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 8)

		no := 1
		for _, cert := range ssl.Certificates {
			if cert.DaysToExpiry <= 30 {
				pdf.CellFormat(8, 6, fmt.Sprintf("%d", no), "1", 0, "C", false, 0, "")
				name := cert.WebsiteName
				if len(name) > 35 {
					name = name[:32] + "..."
				}
				pdf.CellFormat(70, 6, name, "1", 0, "L", false, 0, "")
				pdf.CellFormat(40, 6, cert.ValidUntil.Format("02 Jan 2006"), "1", 0, "C", false, 0, "")

				if cert.DaysToExpiry <= 0 {
					pdf.SetTextColor(255, 0, 0)
					pdf.CellFormat(25, 6, fmt.Sprintf("%d", cert.DaysToExpiry), "1", 0, "C", false, 0, "")
					pdf.CellFormat(37, 6, "EXPIRED", "1", 1, "C", false, 0, "")
				} else {
					pdf.SetTextColor(255, 165, 0)
					pdf.CellFormat(25, 6, fmt.Sprintf("%d", cert.DaysToExpiry), "1", 0, "C", false, 0, "")
					pdf.CellFormat(37, 6, "Akan Expired", "1", 1, "C", false, 0, "")
				}
				pdf.SetTextColor(0, 0, 0)
				no++
			}
		}
	}

	// ==================== Alerts Page ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "RIWAYAT ALERT")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Daftar alert/peringatan yang terjadi selama periode laporan.", "", "L", false)
	pdf.Ln(5)

	// Alert summary
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(50, 7, "Total Alert", "1", 0, "L", false, 0, "")
	pdf.CellFormat(25, 7, fmt.Sprintf("%d", alerts.Summary.TotalAlerts), "1", 1, "C", false, 0, "")

	if len(alerts.AlertsList) > 0 {
		pdf.Ln(5)
		pdf.SetFillColor(66, 114, 196)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(50, 7, "Website", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "Tipe", "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, "Severity", "1", 0, "C", true, 0, "")
		pdf.CellFormat(67, 7, "Tanggal", "1", 1, "C", true, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 8)

		for i, alert := range alerts.AlertsList {
			if i >= 20 {
				pdf.Ln(3)
				pdf.SetFont("Helvetica", "I", 9)
				pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d alert lainnya", len(alerts.AlertsList)-20), "", 1, "L", false, 0, "")
				break
			}
			pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
			name := alert.WebsiteName
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			pdf.CellFormat(50, 6, name, "1", 0, "L", false, 0, "")

			// Translate alert type
			alertTypeID := alert.Type
			switch alert.Type {
			case "judol_detected":
				alertTypeID = "Judol"
			case "defacement":
				alertTypeID = "Defacement"
			case "down":
				alertTypeID = "Down"
			case "ssl_expiring":
				alertTypeID = "SSL Expiring"
			case "ssl_expired":
				alertTypeID = "SSL Expired"
			}
			pdf.CellFormat(30, 6, alertTypeID, "1", 0, "C", false, 0, "")

			// Color code severity
			switch alert.Severity {
			case "critical":
				pdf.SetTextColor(255, 0, 0)
			case "warning":
				pdf.SetTextColor(255, 165, 0)
			default:
				pdf.SetTextColor(0, 128, 0)
			}
			pdf.CellFormat(25, 6, alert.Severity, "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)

			pdf.CellFormat(67, 6, alert.CreatedAt.Format("02 Jan 2006 15:04"), "1", 1, "L", false, 0, "")
		}
	}

	// ==================== Footer/Closing ====================
	pdf.AddPage()
	s.addPDFSection(pdf, "KESIMPULAN DAN REKOMENDASI")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	// Auto-generate recommendations based on data
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Kesimpulan:", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(2)

	// Content issues
	if contentScan.Summary.InfectedWebsites > 0 {
		pdf.SetTextColor(255, 0, 0)
		pdf.MultiCell(0, 5, fmt.Sprintf("- Ditemukan %d website terinfeksi konten judol/defacement yang memerlukan penanganan segera.", contentScan.Summary.InfectedWebsites), "", "L", false)
		pdf.SetTextColor(0, 0, 0)
	} else {
		pdf.SetTextColor(0, 128, 0)
		pdf.MultiCell(0, 5, "- Semua website dalam kondisi bersih dari konten judol dan defacement.", "", "L", false)
		pdf.SetTextColor(0, 0, 0)
	}

	// SSL issues
	if ssl.Summary.ExpiringSoon > 0 || ssl.Summary.Expired > 0 {
		pdf.SetTextColor(255, 165, 0)
		pdf.MultiCell(0, 5, fmt.Sprintf("- Terdapat %d sertifikat SSL yang akan/sudah expired dan perlu diperpanjang.", ssl.Summary.ExpiringSoon+ssl.Summary.Expired), "", "L", false)
		pdf.SetTextColor(0, 0, 0)
	}

	pdf.Ln(5)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 7, "Rekomendasi:", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(2)

	if contentScan.Summary.InfectedWebsites > 0 {
		pdf.MultiCell(0, 5, "1. Segera lakukan pembersihan konten pada website yang terinfeksi.", "", "L", false)
		pdf.MultiCell(0, 5, "2. Lakukan investigasi keamanan untuk menentukan penyebab infeksi.", "", "L", false)
		pdf.MultiCell(0, 5, "3. Tingkatkan keamanan website dengan update CMS dan plugin.", "", "L", false)
	}
	if ssl.Summary.ExpiringSoon > 0 {
		pdf.MultiCell(0, 5, fmt.Sprintf("%d. Perpanjang sertifikat SSL sebelum masa berlaku habis.", contentScan.Summary.InfectedWebsites+1), "", "L", false)
	}

	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	// Output PDF
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (s *ReportService) addPDFSection(pdf *fpdf.Fpdf, title string) {
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.CellFormat(0, 10, title, "", 1, "L", true, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.Ln(2)
}

func (s *ReportService) addPDFCoverPage(pdf *fpdf.Fpdf, title string, metadata *domain.ReportMetadata) {
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 24)
	pdf.Ln(30)
	pdf.CellFormat(0, 15, title, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 14)
	pdf.Ln(5)
	pdf.CellFormat(0, 8, "Pemerintah Provinsi Bali", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, "Dinas Komunikasi, Informatika dan Statistik", "", 1, "C", false, 0, "")

	pdf.Ln(20)
	pdf.SetFont("Helvetica", "", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Periode: %s s/d %s",
		metadata.Period.StartDate.Format("02 January 2006"),
		metadata.Period.EndDate.Format("02 January 2006")), "", 1, "C", false, 0, "")

	pdf.Ln(40)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Dibuat pada: %s", metadata.GeneratedAt.Format("02 January 2006 15:04 WIB")), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Dibuat oleh: %s", metadata.GeneratedBy), "", 1, "C", false, 0, "")
}

func (s *ReportService) generateUptimePDF(data *domain.UptimeReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	s.addPDFCoverPage(pdf, "LAPORAN UPTIME WEBSITE", &data.Metadata)

	// Summary page
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN UPTIME")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	summaryData := [][]string{
		{"Total Website Dimonitor", fmt.Sprintf("%d website", data.Summary.TotalWebsites)},
		{"Rata-rata Uptime", fmt.Sprintf("%.2f%%", data.Summary.AverageUptime)},
		{"Rata-rata Response Time", fmt.Sprintf("%.2f ms", data.Summary.AverageResponseTime)},
		{"Total Pengecekan", fmt.Sprintf("%d kali", data.Summary.TotalChecks)},
		{"Performa Terbaik", data.Summary.BestPerforming},
		{"Performa Terburuk", data.Summary.WorstPerforming},
	}

	for _, row := range summaryData {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// Detail table
	pdf.AddPage()
	s.addPDFSection(pdf, "DETAIL UPTIME WEBSITE")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Status ketersediaan (uptime) dan response time masing-masing website.", "", "L", false)
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(60, 7, "Nama Website", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Uptime (%)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, "Avg Resp (ms)", "1", 0, "C", true, 0, "")
	pdf.CellFormat(57, 7, "Status", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	maxRows := 50
	for i, ws := range data.WebsiteStats {
		if i >= maxRows {
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d website lainnya", len(data.WebsiteStats)-maxRows), "", 1, "L", false, 0, "")
			break
		}
		pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		name := ws.WebsiteName
		if len(name) > 30 {
			name = name[:27] + "..."
		}
		pdf.CellFormat(60, 6, name, "1", 0, "L", false, 0, "")

		// Color code uptime
		if ws.UptimePercent >= 99 {
			pdf.SetTextColor(0, 128, 0)
		} else if ws.UptimePercent >= 90 {
			pdf.SetTextColor(255, 165, 0)
		} else {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(25, 6, fmt.Sprintf("%.2f%%", ws.UptimePercent), "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)

		pdf.CellFormat(30, 6, fmt.Sprintf("%.0f", ws.AvgResponseTime), "1", 0, "C", false, 0, "")

		status := "UP"
		if ws.UptimePercent < 50 {
			status = "DOWN"
			pdf.SetTextColor(255, 0, 0)
		} else if ws.UptimePercent < 90 {
			status = "DEGRADED"
			pdf.SetTextColor(255, 165, 0)
		} else {
			pdf.SetTextColor(0, 128, 0)
		}
		pdf.CellFormat(57, 6, status, "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateSSLPDF(data *domain.SSLReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	s.addPDFCoverPage(pdf, "LAPORAN SERTIFIKAT SSL", &data.Metadata)

	// Summary page
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN SERTIFIKAT SSL")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	summaryRows := [][]string{
		{"Total Website", fmt.Sprintf("%d website", data.Summary.TotalWebsites)},
		{"Sertifikat Valid", fmt.Sprintf("%d", data.Summary.ValidCertificates)},
		{"Akan Expired (30 hari)", fmt.Sprintf("%d", data.Summary.ExpiringSoon)},
		{"Sudah Expired", fmt.Sprintf("%d", data.Summary.Expired)},
		{"Tanpa Sertifikat", fmt.Sprintf("%d", data.Summary.NoCertificate)},
	}

	for _, row := range summaryRows {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// Detail table
	pdf.AddPage()
	s.addPDFSection(pdf, "DETAIL SERTIFIKAT SSL")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Status sertifikat SSL/TLS untuk masing-masing website.", "", "L", false)
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(55, 7, "Nama Website", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 7, "Issuer", "1", 0, "C", true, 0, "")
	pdf.CellFormat(30, 7, "Berlaku Hingga", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Sisa Hari", "1", 0, "C", true, 0, "")
	pdf.CellFormat(27, 7, "Status", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	maxRows := 50
	for i, cert := range data.Certificates {
		if i >= maxRows {
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d sertifikat lainnya", len(data.Certificates)-maxRows), "", 1, "L", false, 0, "")
			break
		}
		pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		name := cert.WebsiteName
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		pdf.CellFormat(55, 6, name, "1", 0, "L", false, 0, "")

		issuer := cert.Issuer
		if len(issuer) > 20 {
			issuer = issuer[:17] + "..."
		}
		pdf.CellFormat(40, 6, issuer, "1", 0, "L", false, 0, "")

		pdf.CellFormat(30, 6, cert.ValidUntil.Format("02 Jan 2006"), "1", 0, "C", false, 0, "")

		// Color code days to expiry
		if cert.DaysToExpiry <= 0 {
			pdf.SetTextColor(255, 0, 0)
		} else if cert.DaysToExpiry <= 30 {
			pdf.SetTextColor(255, 165, 0)
		} else {
			pdf.SetTextColor(0, 128, 0)
		}
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", cert.DaysToExpiry), "1", 0, "C", false, 0, "")

		status := "Valid"
		if cert.DaysToExpiry <= 0 {
			status = "EXPIRED"
		} else if cert.DaysToExpiry <= 30 {
			status = "Akan Expired"
		}
		pdf.CellFormat(27, 6, status, "1", 1, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateSecurityPDF(data *domain.SecurityReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	s.addPDFCoverPage(pdf, "LAPORAN SECURITY HEADERS", &data.Metadata)

	// Summary page
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN SECURITY HEADERS")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	summaryRows := [][]string{
		{"Total Website", fmt.Sprintf("%d website", data.Summary.TotalWebsites)},
		{"Rata-rata Skor", fmt.Sprintf("%.1f", data.Summary.AverageScore)},
		{"Grade A/A+", fmt.Sprintf("%d website", data.Summary.GradeACount)},
		{"Grade B", fmt.Sprintf("%d website", data.Summary.GradeBCount)},
		{"Grade C", fmt.Sprintf("%d website", data.Summary.GradeCCount)},
		{"Grade D", fmt.Sprintf("%d website", data.Summary.GradeDCount)},
		{"Grade F", fmt.Sprintf("%d website", data.Summary.GradeFCount)},
	}

	for _, row := range summaryRows {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// Detail table
	pdf.AddPage()
	s.addPDFSection(pdf, "DETAIL SECURITY HEADERS")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Skor dan grade security headers untuk masing-masing website.", "", "L", false)
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(65, 7, "Nama Website", "1", 0, "C", true, 0, "")
	pdf.CellFormat(25, 7, "Skor", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Grade", "1", 0, "C", true, 0, "")
	pdf.CellFormat(62, 7, "Keterangan", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	maxRows := 50
	for i, ws := range data.WebsiteSecurity {
		if i >= maxRows {
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d website lainnya", len(data.WebsiteSecurity)-maxRows), "", 1, "L", false, 0, "")
			break
		}
		pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		name := ws.WebsiteName
		if len(name) > 32 {
			name = name[:29] + "..."
		}
		pdf.CellFormat(65, 6, name, "1", 0, "L", false, 0, "")
		pdf.CellFormat(25, 6, fmt.Sprintf("%d", ws.SecurityScore), "1", 0, "C", false, 0, "")

		// Color code grade
		switch ws.SecurityGrade {
		case "A", "A+":
			pdf.SetTextColor(0, 128, 0)
		case "B":
			pdf.SetTextColor(0, 128, 0)
		case "C":
			pdf.SetTextColor(255, 165, 0)
		case "D":
			pdf.SetTextColor(255, 165, 0)
		default:
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(20, 6, ws.SecurityGrade, "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)

		keterangan := "Baik"
		if ws.SecurityGrade == "F" {
			keterangan = "Perlu Perbaikan Segera"
		} else if ws.SecurityGrade == "D" || ws.SecurityGrade == "C" {
			keterangan = "Perlu Ditingkatkan"
		}
		pdf.CellFormat(62, 6, keterangan, "1", 1, "L", false, 0, "")
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateAlertsPDF(data *domain.AlertsReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	s.addPDFCoverPage(pdf, "LAPORAN ALERTS", &data.Metadata)

	// Summary page
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN ALERTS")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	summaryRows := [][]string{
		{"Total Alerts", fmt.Sprintf("%d alert", data.Summary.TotalAlerts)},
		{"Alert Terselesaikan", fmt.Sprintf("%d alert", data.Summary.ResolvedAlerts)},
		{"Alert Belum Selesai", fmt.Sprintf("%d alert", data.Summary.UnresolvedAlerts)},
		{"Critical", fmt.Sprintf("%d alert", data.Summary.CriticalCount)},
		{"Warning", fmt.Sprintf("%d alert", data.Summary.WarningCount)},
		{"Info", fmt.Sprintf("%d alert", data.Summary.InfoCount)},
		{"Rata-rata Waktu Resolusi", fmt.Sprintf("%.1f jam", data.Summary.AvgResolutionHours)},
	}

	for _, row := range summaryRows {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// Detail table
	if len(data.AlertsList) > 0 {
		pdf.AddPage()
		s.addPDFSection(pdf, "DAFTAR ALERTS")

		pdf.SetFont("Helvetica", "", 10)
		pdf.Ln(3)
		pdf.MultiCell(0, 5, "Daftar alert/peringatan yang terjadi selama periode laporan.", "", "L", false)
		pdf.Ln(5)

		// Table header
		pdf.SetFillColor(66, 114, 196)
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
		pdf.CellFormat(50, 7, "Website", "1", 0, "C", true, 0, "")
		pdf.CellFormat(30, 7, "Tipe", "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 7, "Severity", "1", 0, "C", true, 0, "")
		pdf.CellFormat(22, 7, "Status", "1", 0, "C", true, 0, "")
		pdf.CellFormat(48, 7, "Tanggal", "1", 1, "C", true, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetFont("Helvetica", "", 8)

		maxRows := 50
		for i, alert := range data.AlertsList {
			if i >= maxRows {
				pdf.Ln(3)
				pdf.SetFont("Helvetica", "I", 9)
				pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d alert lainnya", len(data.AlertsList)-maxRows), "", 1, "L", false, 0, "")
				break
			}
			pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
			name := alert.WebsiteName
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			pdf.CellFormat(50, 6, name, "1", 0, "L", false, 0, "")

			// Translate alert type
			alertTypeID := alert.Type
			switch alert.Type {
			case "judol_detected":
				alertTypeID = "Judol"
			case "defacement":
				alertTypeID = "Defacement"
			case "down":
				alertTypeID = "Down"
			case "ssl_expiring":
				alertTypeID = "SSL Expiring"
			case "ssl_expired":
				alertTypeID = "SSL Expired"
			}
			pdf.CellFormat(30, 6, alertTypeID, "1", 0, "C", false, 0, "")

			// Color code severity
			switch alert.Severity {
			case "critical":
				pdf.SetTextColor(255, 0, 0)
			case "warning":
				pdf.SetTextColor(255, 165, 0)
			default:
				pdf.SetTextColor(0, 128, 0)
			}
			pdf.CellFormat(22, 6, alert.Severity, "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)

			status := "Open"
			if alert.IsResolved {
				status = "Resolved"
				pdf.SetTextColor(0, 128, 0)
			} else {
				pdf.SetTextColor(255, 0, 0)
			}
			pdf.CellFormat(22, 6, status, "1", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)

			pdf.CellFormat(48, 6, alert.CreatedAt.Format("02 Jan 2006 15:04"), "1", 1, "L", false, 0, "")
		}
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ReportService) generateContentScanPDF(data *domain.ContentScanReportData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Cover page
	s.addPDFCoverPage(pdf, "LAPORAN CONTENT SCAN", &data.Metadata)

	// Summary page
	pdf.AddPage()
	s.addPDFSection(pdf, "RINGKASAN CONTENT SCAN (Judol/Defacement)")

	pdf.SetFont("Helvetica", "", 11)
	pdf.Ln(5)

	summaryRows := [][]string{
		{"Total Website Dipindai", fmt.Sprintf("%d website", data.Summary.TotalWebsites)},
		{"Website Bersih", fmt.Sprintf("%d website", data.Summary.CleanWebsites)},
		{"Website Terinfeksi", fmt.Sprintf("%d website", data.Summary.InfectedWebsites)},
		{"Total Keywords Ditemukan", fmt.Sprintf("%d", data.Summary.TotalKeywords)},
		{"Total Iframes Mencurigakan", fmt.Sprintf("%d", data.Summary.TotalIframes)},
		{"Total Redirects Mencurigakan", fmt.Sprintf("%d", data.Summary.TotalRedirects)},
	}

	for _, row := range summaryRows {
		pdf.SetFont("Helvetica", "", 11)
		pdf.CellFormat(80, 8, row[0], "1", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 11)
		pdf.CellFormat(100, 8, row[1], "1", 1, "L", false, 0, "")
	}

	// Detail table
	pdf.AddPage()
	s.addPDFSection(pdf, "DETAIL HASIL SCAN")

	pdf.SetFont("Helvetica", "", 10)
	pdf.Ln(3)
	pdf.MultiCell(0, 5, "Hasil pemindaian konten website untuk mendeteksi konten judi online (judol) dan defacement.", "", "L", false)
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(66, 114, 196)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(8, 7, "No", "1", 0, "C", true, 0, "")
	pdf.CellFormat(50, 7, "Nama Website", "1", 0, "C", true, 0, "")
	pdf.CellFormat(22, 7, "Status", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Keywords", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Iframes", "1", 0, "C", true, 0, "")
	pdf.CellFormat(20, 7, "Redirects", "1", 0, "C", true, 0, "")
	pdf.CellFormat(40, 7, "URL", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 8)

	maxRows := 50
	for i, scan := range data.ScanResults {
		if i >= maxRows {
			pdf.Ln(3)
			pdf.SetFont("Helvetica", "I", 9)
			pdf.CellFormat(0, 6, fmt.Sprintf("... dan %d website lainnya", len(data.ScanResults)-maxRows), "", 1, "L", false, 0, "")
			break
		}
		pdf.CellFormat(8, 6, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		name := scan.WebsiteName
		if len(name) > 25 {
			name = name[:22] + "..."
		}
		pdf.CellFormat(50, 6, name, "1", 0, "L", false, 0, "")

		// Color code status
		if scan.IsClean {
			pdf.SetTextColor(0, 128, 0)
			pdf.CellFormat(22, 6, "BERSIH", "1", 0, "C", false, 0, "")
		} else {
			pdf.SetTextColor(255, 0, 0)
			pdf.CellFormat(22, 6, "TERINFEKSI", "1", 0, "C", false, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)

		pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.KeywordsFound), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.IframesFound), "1", 0, "C", false, 0, "")
		pdf.CellFormat(20, 6, fmt.Sprintf("%d", scan.RedirectsFound), "1", 0, "C", false, 0, "")

		urlTrunc := scan.URL
		if len(urlTrunc) > 22 {
			urlTrunc = urlTrunc[:19] + "..."
		}
		pdf.CellFormat(40, 6, urlTrunc, "1", 1, "L", false, 0, "")
	}

	// Footer
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "I", 9)
	pdf.CellFormat(0, 5, "Laporan ini dibuat secara otomatis oleh Sistem Monitoring Website Pemerintah Provinsi Bali.", "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
