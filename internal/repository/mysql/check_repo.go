package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type CheckRepository struct {
	db *sqlx.DB
}

func NewCheckRepository(db *sqlx.DB) *CheckRepository {
	return &CheckRepository{db: db}
}

// CreateCheck saves a new uptime check result
func (r *CheckRepository) CreateCheck(ctx context.Context, c *domain.Check) (int64, error) {
	query := `
		INSERT INTO checks (website_id, status_code, response_time, status, error_message, error_type, content_length)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		c.WebsiteID, c.StatusCode, c.ResponseTime, c.Status,
		c.ErrorMessage, c.ErrorType, c.ContentLength,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetChecksByWebsiteID returns check history for a website
func (r *CheckRepository) GetChecksByWebsiteID(ctx context.Context, websiteID int64, limit int) ([]domain.Check, error) {
	var checks []domain.Check

	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT * FROM checks
		WHERE website_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`

	err := r.db.SelectContext(ctx, &checks, query, websiteID, limit)
	if err != nil {
		return nil, err
	}

	return checks, nil
}

// GetRecentChecks returns the most recent checks for a website
func (r *CheckRepository) GetRecentChecks(ctx context.Context, websiteID int64, limit int) ([]domain.Check, error) {
	return r.GetChecksByWebsiteID(ctx, websiteID, limit)
}

// GetLatestCheck returns the latest check for a website
func (r *CheckRepository) GetLatestCheck(ctx context.Context, websiteID int64) (*domain.Check, error) {
	var check domain.Check
	query := `SELECT * FROM checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &check, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &check, nil
}

// GetUptimeStats calculates uptime statistics for a period
func (r *CheckRepository) GetUptimeStats(ctx context.Context, websiteID int64, since time.Time) (*domain.UptimeStats, error) {
	query := `
		SELECT
			COUNT(*) as total_checks,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status IN ('down', 'timeout', 'error') THEN 1 ELSE 0 END) as down_count,
			AVG(response_time) as avg_response_time,
			MIN(response_time) as min_response_time,
			MAX(response_time) as max_response_time
		FROM checks
		WHERE website_id = ? AND checked_at >= ?
	`

	var result struct {
		TotalChecks     int             `db:"total_checks"`
		UpCount         int             `db:"up_count"`
		DownCount       int             `db:"down_count"`
		AvgResponseTime sql.NullFloat64 `db:"avg_response_time"`
		MinResponseTime sql.NullInt32   `db:"min_response_time"`
		MaxResponseTime sql.NullInt32   `db:"max_response_time"`
	}

	err := r.db.GetContext(ctx, &result, query, websiteID, since)
	if err != nil {
		return nil, err
	}

	stats := &domain.UptimeStats{
		WebsiteID:   websiteID,
		TotalChecks: result.TotalChecks,
		UpCount:     result.UpCount,
		DownCount:   result.DownCount,
	}

	if result.TotalChecks > 0 {
		stats.UptimePercentage = float64(result.UpCount) * 100 / float64(result.TotalChecks)
	}
	if result.AvgResponseTime.Valid {
		stats.AvgResponseTime = result.AvgResponseTime.Float64
	}
	if result.MinResponseTime.Valid {
		stats.MinResponseTime = int(result.MinResponseTime.Int32)
	}
	if result.MaxResponseTime.Valid {
		stats.MaxResponseTime = int(result.MaxResponseTime.Int32)
	}

	return stats, nil
}

// GetResponseTimePercentiles calculates response time percentiles for a website over a given number of hours
func (r *CheckRepository) GetResponseTimePercentiles(ctx context.Context, websiteID int64, hours int) (*domain.ResponseTimePercentiles, error) {
	// First get basic stats (avg, min, max, count)
	statsQuery := `
		SELECT
			COALESCE(AVG(response_time), 0) as avg_rt,
			COALESCE(MIN(response_time), 0) as min_rt,
			COALESCE(MAX(response_time), 0) as max_rt,
			COUNT(*) as count
		FROM checks
		WHERE website_id = ? AND checked_at >= DATE_SUB(NOW(), INTERVAL ? HOUR) AND response_time > 0
	`

	var stats struct {
		Avg   float64 `db:"avg_rt"`
		Min   float64 `db:"min_rt"`
		Max   float64 `db:"max_rt"`
		Count int     `db:"count"`
	}

	err := r.db.GetContext(ctx, &stats, statsQuery, websiteID, hours)
	if err != nil {
		return nil, err
	}

	result := &domain.ResponseTimePercentiles{
		Avg:   stats.Avg,
		Min:   stats.Min,
		Max:   stats.Max,
		Count: stats.Count,
	}

	if stats.Count == 0 {
		return result, nil
	}

	// Calculate percentiles using MySQL CTE with ROW_NUMBER
	percentileQuery := `
		WITH ordered AS (
			SELECT response_time,
				ROW_NUMBER() OVER (ORDER BY response_time) as rn,
				COUNT(*) OVER () as total
			FROM checks
			WHERE website_id = ? AND checked_at >= DATE_SUB(NOW(), INTERVAL ? HOUR) AND response_time > 0
		)
		SELECT
			COALESCE(MAX(CASE WHEN rn = CEIL(total * 0.50) THEN response_time END), 0) as p50,
			COALESCE(MAX(CASE WHEN rn = CEIL(total * 0.95) THEN response_time END), 0) as p95,
			COALESCE(MAX(CASE WHEN rn = CEIL(total * 0.99) THEN response_time END), 0) as p99
		FROM ordered
	`

	var percentiles struct {
		P50 float64 `db:"p50"`
		P95 float64 `db:"p95"`
		P99 float64 `db:"p99"`
	}

	err = r.db.GetContext(ctx, &percentiles, percentileQuery, websiteID, hours)
	if err != nil {
		// If CTE not supported (MySQL < 8.0), fall back to basic stats only
		result.P50 = stats.Avg
		result.P95 = stats.Max
		result.P99 = stats.Max
		return result, nil
	}

	result.P50 = percentiles.P50
	result.P95 = percentiles.P95
	result.P99 = percentiles.P99

	return result, nil
}

// CreateSSLCheck saves a new SSL check result
func (r *CheckRepository) CreateSSLCheck(ctx context.Context, s *domain.SSLCheck) (int64, error) {
	query := `
		INSERT INTO ssl_checks (website_id, is_valid, issuer, subject, valid_from, valid_until, days_until_expiry, protocol, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		s.WebsiteID, s.IsValid, s.Issuer, s.Subject,
		s.ValidFrom, s.ValidUntil, s.DaysUntilExpiry,
		s.Protocol, s.ErrorMessage,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestSSLCheck returns the latest SSL check for a website
func (r *CheckRepository) GetLatestSSLCheck(ctx context.Context, websiteID int64) (*domain.SSLCheck, error) {
	var check domain.SSLCheck
	query := `SELECT * FROM ssl_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &check, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &check, nil
}

// CreateContentScan saves a new content scan result
func (r *CheckRepository) CreateContentScan(ctx context.Context, s *domain.ContentScan) (int64, error) {
	query := `
		INSERT INTO content_scans (website_id, is_clean, scan_type, findings, page_title, page_hash, keywords_found, iframes_found, redirects_found)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		s.WebsiteID, s.IsClean, s.ScanType, s.Findings,
		s.PageTitle, s.PageHash, s.KeywordsFound,
		s.IframesFound, s.RedirectsFound,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestContentScan returns the latest content scan for a website
func (r *CheckRepository) GetLatestContentScan(ctx context.Context, websiteID int64) (*domain.ContentScan, error) {
	var scan domain.ContentScan
	query := `SELECT * FROM content_scans WHERE website_id = ? ORDER BY scanned_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &scan, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &scan, nil
}

// GetContentScansByWebsiteID returns content scan history
func (r *CheckRepository) GetContentScansByWebsiteID(ctx context.Context, websiteID int64, limit int) ([]domain.ContentScan, error) {
	var scans []domain.ContentScan

	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT * FROM content_scans
		WHERE website_id = ?
		ORDER BY scanned_at DESC
		LIMIT ?
	`

	err := r.db.SelectContext(ctx, &scans, query, websiteID, limit)
	if err != nil {
		return nil, err
	}

	return scans, nil
}

// CleanupOldChecks removes checks older than specified days
func (r *CheckRepository) CleanupOldChecks(ctx context.Context, days int) error {
	query := `DELETE FROM checks WHERE checked_at < DATE_SUB(NOW(), INTERVAL ? DAY)`
	_, err := r.db.ExecContext(ctx, query, days)
	return err
}

// CleanupOldData removes old records from a specified table
func (r *CheckRepository) CleanupOldData(ctx context.Context, tableName string, days int) (int64, error) {
	// Map table names to their date columns
	dateColumns := map[string]string{
		"checks":                 "checked_at",
		"ssl_checks":             "checked_at",
		"content_scans":          "scanned_at",
		"security_header_checks": "checked_at",
	}

	dateCol, ok := dateColumns[tableName]
	if !ok {
		return 0, nil
	}

	query := "DELETE FROM " + tableName + " WHERE " + dateCol + " < DATE_SUB(NOW(), INTERVAL ? DAY)"
	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetTableRowCount returns the row count for a table
func (r *CheckRepository) GetTableRowCount(ctx context.Context, tableName string) (int64, error) {
	var count int64
	query := "SELECT COUNT(*) FROM " + tableName
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

// GetOldestRecordDate returns the oldest record date in a table
func (r *CheckRepository) GetOldestRecordDate(ctx context.Context, tableName, dateColumn string) (*time.Time, error) {
	var oldest time.Time
	query := "SELECT MIN(" + dateColumn + ") FROM " + tableName
	err := r.db.GetContext(ctx, &oldest, query)
	if err != nil {
		return nil, err
	}
	if oldest.IsZero() {
		return nil, nil
	}
	return &oldest, nil
}

// GetHourlyChartData retrieves hourly aggregated data for charts
func (r *CheckRepository) GetHourlyChartData(ctx context.Context, websiteID int64, hours int) (*domain.UptimeChartData, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	// Get hourly status data
	statusQuery := `
		SELECT
			DATE_FORMAT(checked_at, '%Y-%m-%d %H:00:00') as timestamp,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status IN ('down', 'timeout', 'error') THEN 1 ELSE 0 END) as down_count
		FROM checks
		WHERE website_id = ? AND checked_at >= ?
		GROUP BY DATE_FORMAT(checked_at, '%Y-%m-%d %H:00:00')
		ORDER BY timestamp ASC
	`

	type statusResult struct {
		Timestamp string `db:"timestamp"`
		UpCount   int    `db:"up_count"`
		DownCount int    `db:"down_count"`
	}

	var statusResults []statusResult
	err := r.db.SelectContext(ctx, &statusResults, statusQuery, websiteID, since)
	if err != nil {
		return nil, err
	}

	// Get hourly response time data
	responseTimeQuery := `
		SELECT
			DATE_FORMAT(checked_at, '%Y-%m-%d %H:00:00') as timestamp,
			COALESCE(AVG(response_time), 0) as avg_response_time,
			COALESCE(MIN(response_time), 0) as min_response_time,
			COALESCE(MAX(response_time), 0) as max_response_time
		FROM checks
		WHERE website_id = ? AND checked_at >= ? AND response_time IS NOT NULL
		GROUP BY DATE_FORMAT(checked_at, '%Y-%m-%d %H:00:00')
		ORDER BY timestamp ASC
	`

	type responseTimeResult struct {
		Timestamp       string  `db:"timestamp"`
		AvgResponseTime float64 `db:"avg_response_time"`
		MinResponseTime int     `db:"min_response_time"`
		MaxResponseTime int     `db:"max_response_time"`
	}

	var responseTimeResults []responseTimeResult
	err = r.db.SelectContext(ctx, &responseTimeResults, responseTimeQuery, websiteID, since)
	if err != nil {
		return nil, err
	}

	// Calculate overall uptime
	var totalUp, totalDown int
	dataPoints := make([]domain.ChartDataPoint, 0, len(statusResults))
	for _, sr := range statusResults {
		ts, _ := time.Parse("2006-01-02 15:04:05", sr.Timestamp)
		status := "up"
		if sr.DownCount > sr.UpCount {
			status = "down"
		} else if sr.DownCount > 0 {
			status = "degraded"
		}
		dataPoints = append(dataPoints, domain.ChartDataPoint{
			Timestamp: ts,
			Status:    status,
			UpCount:   sr.UpCount,
			DownCount: sr.DownCount,
		})
		totalUp += sr.UpCount
		totalDown += sr.DownCount
	}

	responseTimes := make([]domain.ResponseTimePoint, 0, len(responseTimeResults))
	for _, rtr := range responseTimeResults {
		ts, _ := time.Parse("2006-01-02 15:04:05", rtr.Timestamp)
		responseTimes = append(responseTimes, domain.ResponseTimePoint{
			Timestamp:       ts,
			AvgResponseTime: rtr.AvgResponseTime,
			MinResponseTime: rtr.MinResponseTime,
			MaxResponseTime: rtr.MaxResponseTime,
		})
	}

	uptimePercentage := float64(0)
	if totalUp+totalDown > 0 {
		uptimePercentage = float64(totalUp) * 100 / float64(totalUp+totalDown)
	}

	return &domain.UptimeChartData{
		WebsiteID:        websiteID,
		Period:           fmt.Sprintf("%dh", hours),
		DataPoints:       dataPoints,
		ResponseTimes:    responseTimes,
		UptimePercentage: uptimePercentage,
	}, nil
}

// GetDailyChartData retrieves daily aggregated data for charts (for longer periods)
func (r *CheckRepository) GetDailyChartData(ctx context.Context, websiteID int64, days int) (*domain.UptimeChartData, error) {
	since := time.Now().AddDate(0, 0, -days)

	// Get daily status data
	statusQuery := `
		SELECT
			DATE(checked_at) as timestamp,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status IN ('down', 'timeout', 'error') THEN 1 ELSE 0 END) as down_count
		FROM checks
		WHERE website_id = ? AND checked_at >= ?
		GROUP BY DATE(checked_at)
		ORDER BY timestamp ASC
	`

	type statusResult struct {
		Timestamp string `db:"timestamp"`
		UpCount   int    `db:"up_count"`
		DownCount int    `db:"down_count"`
	}

	var statusResults []statusResult
	err := r.db.SelectContext(ctx, &statusResults, statusQuery, websiteID, since)
	if err != nil {
		return nil, err
	}

	// Get daily response time data
	responseTimeQuery := `
		SELECT
			DATE(checked_at) as timestamp,
			COALESCE(AVG(response_time), 0) as avg_response_time,
			COALESCE(MIN(response_time), 0) as min_response_time,
			COALESCE(MAX(response_time), 0) as max_response_time
		FROM checks
		WHERE website_id = ? AND checked_at >= ? AND response_time IS NOT NULL
		GROUP BY DATE(checked_at)
		ORDER BY timestamp ASC
	`

	type responseTimeResult struct {
		Timestamp       string  `db:"timestamp"`
		AvgResponseTime float64 `db:"avg_response_time"`
		MinResponseTime int     `db:"min_response_time"`
		MaxResponseTime int     `db:"max_response_time"`
	}

	var responseTimeResults []responseTimeResult
	err = r.db.SelectContext(ctx, &responseTimeResults, responseTimeQuery, websiteID, since)
	if err != nil {
		return nil, err
	}

	// Calculate overall uptime
	var totalUp, totalDown int
	dataPoints := make([]domain.ChartDataPoint, 0, len(statusResults))
	for _, sr := range statusResults {
		ts, _ := time.Parse("2006-01-02", sr.Timestamp)
		status := "up"
		if sr.DownCount > sr.UpCount {
			status = "down"
		} else if sr.DownCount > 0 {
			status = "degraded"
		}
		dataPoints = append(dataPoints, domain.ChartDataPoint{
			Timestamp: ts,
			Status:    status,
			UpCount:   sr.UpCount,
			DownCount: sr.DownCount,
		})
		totalUp += sr.UpCount
		totalDown += sr.DownCount
	}

	responseTimes := make([]domain.ResponseTimePoint, 0, len(responseTimeResults))
	for _, rtr := range responseTimeResults {
		ts, _ := time.Parse("2006-01-02", rtr.Timestamp)
		responseTimes = append(responseTimes, domain.ResponseTimePoint{
			Timestamp:       ts,
			AvgResponseTime: rtr.AvgResponseTime,
			MinResponseTime: rtr.MinResponseTime,
			MaxResponseTime: rtr.MaxResponseTime,
		})
	}

	uptimePercentage := float64(0)
	if totalUp+totalDown > 0 {
		uptimePercentage = float64(totalUp) * 100 / float64(totalUp+totalDown)
	}

	return &domain.UptimeChartData{
		WebsiteID:        websiteID,
		Period:           fmt.Sprintf("%dd", days),
		DataPoints:       dataPoints,
		ResponseTimes:    responseTimes,
		UptimePercentage: uptimePercentage,
	}, nil
}

// GetPublicUptimeHistory retrieves public uptime history for a website (simplified for public display)
func (r *CheckRepository) GetPublicUptimeHistory(ctx context.Context, websiteID int64, days int) (*domain.PublicUptimeHistory, error) {
	since := time.Now().AddDate(0, 0, -days)

	// Get daily aggregated data
	query := `
		SELECT
			DATE(checked_at) as date,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status IN ('down', 'timeout', 'error') THEN 1 ELSE 0 END) as down_count,
			COUNT(*) as total_count
		FROM checks
		WHERE website_id = ? AND checked_at >= ?
		GROUP BY DATE(checked_at)
		ORDER BY date ASC
	`

	type dailyResult struct {
		Date       string `db:"date"`
		UpCount    int    `db:"up_count"`
		DownCount  int    `db:"down_count"`
		TotalCount int    `db:"total_count"`
	}

	var results []dailyResult
	err := r.db.SelectContext(ctx, &results, query, websiteID, since)
	if err != nil {
		return nil, err
	}

	// Calculate overall uptime and build data points
	var totalUp, totalDown int
	dataPoints := make([]domain.UptimeHistoryPoint, 0, len(results))

	for _, r := range results {
		uptime := float64(0)
		status := "operational"

		if r.TotalCount > 0 {
			uptime = float64(r.UpCount) * 100 / float64(r.TotalCount)
			if r.DownCount > r.UpCount {
				status = "down"
			} else if r.DownCount > 0 {
				status = "degraded"
			}
		}

		dataPoints = append(dataPoints, domain.UptimeHistoryPoint{
			Date:   r.Date,
			Status: status,
			Uptime: uptime,
		})

		totalUp += r.UpCount
		totalDown += r.DownCount
	}

	overallUptime := float64(0)
	if totalUp+totalDown > 0 {
		overallUptime = float64(totalUp) * 100 / float64(totalUp+totalDown)
	}

	period := fmt.Sprintf("%dd", days)

	return &domain.PublicUptimeHistory{
		ServiceID:  websiteID,
		Period:     period,
		Uptime:     overallUptime,
		DataPoints: dataPoints,
	}, nil
}

// GetServiceUptimePercentage calculates the uptime percentage for a specific service over a period
func (r *CheckRepository) GetServiceUptimePercentage(ctx context.Context, websiteID int64, days int) (float64, error) {
	since := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) as up_count,
			COUNT(*) as total_count
		FROM checks
		WHERE website_id = ? AND checked_at >= ?
	`

	var result struct {
		UpCount    int `db:"up_count"`
		TotalCount int `db:"total_count"`
	}

	err := r.db.GetContext(ctx, &result, query, websiteID, since)
	if err != nil {
		return 0, err
	}

	if result.TotalCount == 0 {
		return 100, nil // No checks means we assume 100% uptime
	}

	return float64(result.UpCount) * 100 / float64(result.TotalCount), nil
}

// CreateSecurityHeaderCheck saves a new security header check result
func (r *CheckRepository) CreateSecurityHeaderCheck(ctx context.Context, check *domain.SecurityHeaderCheck) (int64, error) {
	query := `
		INSERT INTO security_header_checks (website_id, score, grade, headers, findings)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		check.WebsiteID, check.Score, check.Grade, check.Headers, check.Findings,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestSecurityHeaderCheck returns the latest security header check for a website
func (r *CheckRepository) GetLatestSecurityHeaderCheck(ctx context.Context, websiteID int64) (*domain.SecurityHeaderCheck, error) {
	var check domain.SecurityHeaderCheck
	query := `SELECT * FROM security_header_checks WHERE website_id = ? ORDER BY checked_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &check, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &check, nil
}

// GetSecurityHeaderHistory returns security header check history
func (r *CheckRepository) GetSecurityHeaderHistory(ctx context.Context, websiteID int64, limit int) ([]domain.SecurityHeaderCheck, error) {
	var checks []domain.SecurityHeaderCheck

	if limit <= 0 {
		limit = 30
	}

	query := `
		SELECT * FROM security_header_checks
		WHERE website_id = ?
		ORDER BY checked_at DESC
		LIMIT ?
	`

	err := r.db.SelectContext(ctx, &checks, query, websiteID, limit)
	if err != nil {
		return nil, err
	}

	return checks, nil
}

// GetSecurityStats returns overall security statistics
func (r *CheckRepository) GetSecurityStats(ctx context.Context) (*domain.SecurityStats, error) {
	// Get latest check for each website
	query := `
		SELECT
			COUNT(DISTINCT website_id) as total_websites,
			COALESCE(AVG(score), 0) as avg_score
		FROM security_header_checks shc
		WHERE shc.checked_at = (
			SELECT MAX(shc2.checked_at)
			FROM security_header_checks shc2
			WHERE shc2.website_id = shc.website_id
		)
	`

	var result struct {
		TotalWebsites int     `db:"total_websites"`
		AvgScore      float64 `db:"avg_score"`
	}

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, err
	}

	// Get grade distribution
	gradeQuery := `
		SELECT grade, COUNT(*) as count
		FROM security_header_checks shc
		WHERE shc.checked_at = (
			SELECT MAX(shc2.checked_at)
			FROM security_header_checks shc2
			WHERE shc2.website_id = shc.website_id
		)
		GROUP BY grade
	`

	type gradeResult struct {
		Grade string `db:"grade"`
		Count int    `db:"count"`
	}

	var grades []gradeResult
	err = r.db.SelectContext(ctx, &grades, gradeQuery)
	if err != nil {
		return nil, err
	}

	distribution := make(map[string]int)
	for _, g := range grades {
		distribution[g.Grade] = g.Count
	}

	// Get SSL statistics from websites table
	sslQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN ssl_valid = TRUE THEN 1 ELSE 0 END), 0) as ssl_valid,
			COALESCE(SUM(CASE WHEN ssl_valid = TRUE AND ssl_expiry_date IS NOT NULL AND ssl_expiry_date <= DATE_ADD(NOW(), INTERVAL 30 DAY) THEN 1 ELSE 0 END), 0) as ssl_expiring_soon
		FROM websites
		WHERE is_active = TRUE
	`

	var sslResult struct {
		SSLValid        int `db:"ssl_valid"`
		SSLExpiringSoon int `db:"ssl_expiring_soon"`
	}

	err = r.db.GetContext(ctx, &sslResult, sslQuery)
	if err != nil {
		// Log error but don't fail the entire stats
		sslResult.SSLValid = 0
		sslResult.SSLExpiringSoon = 0
	}

	// Count websites with low security scores (missing important headers)
	missingHeadersQuery := `
		SELECT COUNT(*) as count
		FROM security_header_checks shc
		WHERE shc.checked_at = (
			SELECT MAX(shc2.checked_at)
			FROM security_header_checks shc2
			WHERE shc2.website_id = shc.website_id
		)
		AND score < 50
	`

	var missingHeaders int
	err = r.db.GetContext(ctx, &missingHeaders, missingHeadersQuery)
	if err != nil {
		missingHeaders = 0
	}

	return &domain.SecurityStats{
		TotalWebsites:     result.TotalWebsites,
		AverageScore:      result.AvgScore,
		GradeDistribution: distribution,
		SSLValid:          sslResult.SSLValid,
		SSLExpiringSoon:   sslResult.SSLExpiringSoon,
		MissingHeaders:    missingHeaders,
	}, nil
}

// ==================== Vulnerability Scan Methods ====================

// CreateVulnerabilityScan saves a new vulnerability scan result
func (r *CheckRepository) CreateVulnerabilityScan(ctx context.Context, scan *domain.VulnerabilityScan) (int64, error) {
	query := `
		INSERT INTO vulnerability_scans (
			website_id, scan_type, total_checks, vulnerabilities_found,
			critical_count, high_count, medium_count, low_count, info_count,
			risk_score, risk_level, cms_detected, cms_version, server_info, findings
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		scan.WebsiteID, scan.ScanType, scan.TotalChecks, scan.VulnerabilitiesFound,
		scan.CriticalCount, scan.HighCount, scan.MediumCount, scan.LowCount, scan.InfoCount,
		scan.RiskScore, scan.RiskLevel, scan.CMSDetected, scan.CMSVersion, scan.ServerInfo, scan.Findings,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetLatestVulnerabilityScan returns the latest vulnerability scan for a website
func (r *CheckRepository) GetLatestVulnerabilityScan(ctx context.Context, websiteID int64) (*domain.VulnerabilityScan, error) {
	var scan domain.VulnerabilityScan
	query := `SELECT * FROM vulnerability_scans WHERE website_id = ? ORDER BY scanned_at DESC LIMIT 1`

	err := r.db.GetContext(ctx, &scan, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &scan, nil
}

// GetVulnerabilityScans returns vulnerability scan history for a website
func (r *CheckRepository) GetVulnerabilityScans(ctx context.Context, websiteID int64, limit int) ([]domain.VulnerabilityScan, error) {
	var scans []domain.VulnerabilityScan

	if limit <= 0 {
		limit = 50
	}

	query := `SELECT * FROM vulnerability_scans WHERE website_id = ? ORDER BY scanned_at DESC LIMIT ?`

	err := r.db.SelectContext(ctx, &scans, query, websiteID, limit)
	if err != nil {
		return nil, err
	}

	return scans, nil
}

// GetVulnerabilityStats returns overall vulnerability statistics
func (r *CheckRepository) GetVulnerabilityStats(ctx context.Context) (*domain.VulnerabilityStats, error) {
	query := `
		SELECT
			COUNT(DISTINCT vs.website_id) as total_scans,
			COALESCE(SUM(CASE WHEN vs.risk_level IN ('critical', 'high') THEN 1 ELSE 0 END), 0) as vulnerable_websites,
			COALESCE(SUM(CASE WHEN vs.risk_level = 'safe' THEN 1 ELSE 0 END), 0) as secure_websites,
			COALESCE(SUM(vs.critical_count), 0) as critical_vulns,
			COALESCE(SUM(vs.high_count), 0) as high_vulns,
			COALESCE(SUM(vs.medium_count), 0) as medium_vulns,
			COALESCE(SUM(vs.low_count), 0) as low_vulns,
			COALESCE(AVG(vs.risk_score), 0) as avg_risk_score
		FROM vulnerability_scans vs
		WHERE vs.scanned_at = (
			SELECT MAX(vs2.scanned_at)
			FROM vulnerability_scans vs2
			WHERE vs2.website_id = vs.website_id
		)
	`

	var result struct {
		TotalScans         int     `db:"total_scans"`
		VulnerableWebsites int     `db:"vulnerable_websites"`
		SecureWebsites     int     `db:"secure_websites"`
		CriticalVulns      int     `db:"critical_vulns"`
		HighVulns          int     `db:"high_vulns"`
		MediumVulns        int     `db:"medium_vulns"`
		LowVulns           int     `db:"low_vulns"`
		AvgRiskScore       float64 `db:"avg_risk_score"`
	}

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, err
	}

	return &domain.VulnerabilityStats{
		TotalScans:         result.TotalScans,
		VulnerableWebsites: result.VulnerableWebsites,
		SecureWebsites:     result.SecureWebsites,
		CriticalVulns:      result.CriticalVulns,
		HighVulns:          result.HighVulns,
		MediumVulns:        result.MediumVulns,
		LowVulns:           result.LowVulns,
		AverageRiskScore:   result.AvgRiskScore,
	}, nil
}

// GetWebsiteVulnerabilityStatus returns vulnerability status for all websites
func (r *CheckRepository) GetWebsiteVulnerabilityStatus(ctx context.Context) ([]domain.WebsiteVulnerabilityStatus, error) {
	// First check if vulnerability_scans table exists
	var tableExists int
	checkQuery := `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'vulnerability_scans'`
	r.db.GetContext(ctx, &tableExists, checkQuery)

	if tableExists == 0 {
		// Return website list with default values if table doesn't exist
		fallbackQuery := `
			SELECT
				id as website_id,
				name as website_name,
				url,
				0 as risk_score,
				'unknown' as risk_level,
				0 as critical_count,
				0 as high_count,
				0 as vulnerabilities_found,
				'' as cms_detected,
				'' as cms_version,
				created_at as last_scan_at
			FROM websites
			WHERE is_active = 1
			ORDER BY name ASC
		`
		var statuses []domain.WebsiteVulnerabilityStatus
		err := r.db.SelectContext(ctx, &statuses, fallbackQuery)
		return statuses, err
	}

	query := `
		SELECT
			w.id as website_id,
			w.name as website_name,
			w.url,
			COALESCE(vs.risk_score, 0) as risk_score,
			COALESCE(vs.risk_level, 'unknown') as risk_level,
			COALESCE(vs.critical_count, 0) as critical_count,
			COALESCE(vs.high_count, 0) as high_count,
			COALESCE(vs.vulnerabilities_found, 0) as vulnerabilities_found,
			COALESCE(vs.cms_detected, '') as cms_detected,
			COALESCE(vs.cms_version, '') as cms_version,
			COALESCE(vs.scanned_at, w.created_at) as last_scan_at
		FROM websites w
		LEFT JOIN vulnerability_scans vs ON w.id = vs.website_id
			AND vs.scanned_at = (
				SELECT MAX(vs2.scanned_at)
				FROM vulnerability_scans vs2
				WHERE vs2.website_id = w.id
			)
		WHERE w.is_active = 1
		ORDER BY vs.risk_score DESC, w.name ASC
	`

	var statuses []domain.WebsiteVulnerabilityStatus
	err := r.db.SelectContext(ctx, &statuses, query)
	if err != nil {
		return nil, err
	}

	return statuses, nil
}

// ==================== Dashboard Trends Methods ====================

// GetOverallUptimeTrend returns daily uptime percentages for the last N days
func (r *CheckRepository) GetOverallUptimeTrend(ctx context.Context, days int) ([]domain.UptimeTrendPoint, error) {
	since := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(checked_at) as date,
			ROUND(
				SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) * 100.0 / COUNT(*),
				2
			) as uptime_percentage
		FROM checks
		WHERE checked_at >= ?
		GROUP BY DATE(checked_at)
		ORDER BY date ASC
	`

	var results []domain.UptimeTrendPoint
	err := r.db.SelectContext(ctx, &results, query, since)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetOverallResponseTimeTrend returns daily average response times for the last N days
func (r *CheckRepository) GetOverallResponseTimeTrend(ctx context.Context, days int) ([]domain.ResponseTimeTrendPoint, error) {
	since := time.Now().AddDate(0, 0, -days)

	query := `
		SELECT
			DATE(checked_at) as date,
			ROUND(AVG(response_time), 0) as avg_response_time
		FROM checks
		WHERE checked_at >= ? AND response_time IS NOT NULL AND response_time > 0
		GROUP BY DATE(checked_at)
		ORDER BY date ASC
	`

	var results []domain.ResponseTimeTrendPoint
	err := r.db.SelectContext(ctx, &results, query, since)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// GetStatusDistribution returns current status distribution of all websites
func (r *CheckRepository) GetStatusDistribution(ctx context.Context) (*domain.StatusDistribution, error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) as up_count,
			COALESCE(SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END), 0) as down_count,
			COALESCE(SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END), 0) as degraded_count
		FROM websites
		WHERE is_active = 1
	`

	var result struct {
		UpCount       int `db:"up_count"`
		DownCount     int `db:"down_count"`
		DegradedCount int `db:"degraded_count"`
	}

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, err
	}

	return &domain.StatusDistribution{
		Up:       result.UpCount,
		Down:     result.DownCount,
		Degraded: result.DegradedCount,
	}, nil
}

// GetDashboardTrends returns all trend data for dashboard charts
func (r *CheckRepository) GetDashboardTrends(ctx context.Context, days int) (*domain.DashboardTrends, error) {
	uptimeTrend, err := r.GetOverallUptimeTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	responseTimes, err := r.GetOverallResponseTimeTrend(ctx, days)
	if err != nil {
		return nil, err
	}

	statusDist, err := r.GetStatusDistribution(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.DashboardTrends{
		UptimeTrend:        uptimeTrend,
		ResponseTimes:      responseTimes,
		StatusDistribution: *statusDist,
	}, nil
}
