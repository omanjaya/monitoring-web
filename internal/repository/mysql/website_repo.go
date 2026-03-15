package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type WebsiteRepository struct {
	db *sqlx.DB
}

func NewWebsiteRepository(db *sqlx.DB) *WebsiteRepository {
	return &WebsiteRepository{db: db}
}

func (r *WebsiteRepository) Create(ctx context.Context, w *domain.WebsiteCreate) (int64, error) {
	query := `
		INSERT INTO websites (url, name, description, opd_id, check_interval, timeout, is_active, response_time_warning, response_time_critical)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	checkInterval := 5
	if w.CheckInterval > 0 {
		checkInterval = w.CheckInterval
	}
	timeout := 30
	if w.Timeout > 0 {
		timeout = w.Timeout
	}

	result, err := r.db.ExecContext(ctx, query,
		w.URL, w.Name, w.Description, w.OPDID,
		checkInterval, timeout, w.IsActive,
		w.ResponseTimeWarning, w.ResponseTimeCritical,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *WebsiteRepository) GetByID(ctx context.Context, id int64) (*domain.Website, error) {
	var website domain.Website
	query := `SELECT * FROM websites WHERE id = ?`

	err := r.db.GetContext(ctx, &website, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &website, nil
}

func (r *WebsiteRepository) GetByURL(ctx context.Context, url string) (*domain.Website, error) {
	var website domain.Website
	query := `SELECT * FROM websites WHERE url = ?`

	err := r.db.GetContext(ctx, &website, query, url)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &website, nil
}

func (r *WebsiteRepository) GetAll(ctx context.Context, filter domain.WebsiteFilter) ([]domain.Website, int, error) {
	var websites []domain.Website
	var total int

	// Build query
	baseQuery := `FROM websites w WHERE 1=1`
	var conditions []string
	var args []interface{}

	if filter.Status != nil {
		conditions = append(conditions, "w.status = ?")
		args = append(args, *filter.Status)
	}
	if filter.OPDID != nil {
		conditions = append(conditions, "w.opd_id = ?")
		args = append(args, *filter.OPDID)
	}
	if filter.IsActive != nil {
		conditions = append(conditions, "w.is_active = ?")
		args = append(args, *filter.IsActive)
	}
	if filter.ContentClean != nil {
		conditions = append(conditions, "w.content_clean = ?")
		args = append(args, *filter.ContentClean)
	}
	if filter.Search != "" {
		conditions = append(conditions, "(w.name LIKE ? OR w.url LIKE ?)")
		searchTerm := "%" + filter.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	if len(conditions) > 0 {
		baseQuery += " AND " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Pagination (Limit == -1 means no limit, used by reports)
	if filter.Limit == -1 {
		dataQuery := fmt.Sprintf("SELECT w.* %s ORDER BY w.name ASC", baseQuery)
		err = r.db.SelectContext(ctx, &websites, dataQuery, args...)
	} else {
		limit := 25
		if filter.Limit > 0 && filter.Limit <= 200 {
			limit = filter.Limit
		}
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * limit
		}

		dataQuery := fmt.Sprintf("SELECT w.* %s ORDER BY w.name ASC LIMIT ? OFFSET ?", baseQuery)
		args = append(args, limit, offset)
		err = r.db.SelectContext(ctx, &websites, dataQuery, args...)
	}
	if err != nil {
		return nil, 0, err
	}

	return websites, total, nil
}

func (r *WebsiteRepository) GetActive(ctx context.Context) ([]domain.Website, error) {
	var websites []domain.Website
	query := `SELECT * FROM websites WHERE is_active = TRUE ORDER BY id`

	err := r.db.SelectContext(ctx, &websites, query)
	if err != nil {
		return nil, err
	}

	return websites, nil
}

func (r *WebsiteRepository) Update(ctx context.Context, id int64, w *domain.WebsiteUpdate) error {
	var sets []string
	var args []interface{}

	if w.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *w.Name)
	}
	if w.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *w.Description)
	}
	if w.OPDID != nil {
		sets = append(sets, "opd_id = ?")
		args = append(args, *w.OPDID)
	}
	if w.CheckInterval != nil {
		sets = append(sets, "check_interval = ?")
		args = append(args, *w.CheckInterval)
	}
	if w.Timeout != nil {
		sets = append(sets, "timeout = ?")
		args = append(args, *w.Timeout)
	}
	if w.IsActive != nil {
		sets = append(sets, "is_active = ?")
		args = append(args, *w.IsActive)
	}
	if w.ResponseTimeWarning != nil {
		sets = append(sets, "response_time_warning = ?")
		args = append(args, *w.ResponseTimeWarning)
	}
	if w.ResponseTimeCritical != nil {
		sets = append(sets, "response_time_critical = ?")
		args = append(args, *w.ResponseTimeCritical)
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE websites SET %s WHERE id = ?", strings.Join(sets, ", "))
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *WebsiteRepository) UpdateStatus(ctx context.Context, id int64, status domain.WebsiteStatus, statusCode, responseTime int) error {
	query := `
		UPDATE websites
		SET status = ?, last_status_code = ?, last_response_time = ?, last_checked_at = NOW()
		WHERE id = ?
	`

	_, err := r.db.ExecContext(ctx, query, status, statusCode, responseTime, id)
	return err
}

func (r *WebsiteRepository) UpdateSSLInfo(ctx context.Context, id int64, valid bool, expiryDate *string) error {
	query := `UPDATE websites SET ssl_valid = ?, ssl_expiry_date = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, valid, expiryDate, id)
	return err
}

func (r *WebsiteRepository) UpdateContentStatus(ctx context.Context, id int64, clean bool) error {
	query := `UPDATE websites SET content_clean = ?, last_scan_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, clean, id)
	return err
}

func (r *WebsiteRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM websites WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *WebsiteRepository) GetStats(ctx context.Context) (map[string]int, error) {
	stats := make(map[string]int)

	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END) as up_count,
			SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END) as down_count,
			SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END) as degraded_count,
			SUM(CASE WHEN content_clean = FALSE THEN 1 ELSE 0 END) as content_issue_count,
			SUM(CASE WHEN ssl_valid = FALSE OR ssl_expiry_date <= DATE_ADD(NOW(), INTERVAL 30 DAY) THEN 1 ELSE 0 END) as ssl_issue_count
		FROM websites WHERE is_active = TRUE
	`

	var result struct {
		Total             int `db:"total"`
		UpCount           int `db:"up_count"`
		DownCount         int `db:"down_count"`
		DegradedCount     int `db:"degraded_count"`
		ContentIssueCount int `db:"content_issue_count"`
		SSLIssueCount     int `db:"ssl_issue_count"`
	}

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, err
	}

	stats["total"] = result.Total
	stats["up"] = result.UpCount
	stats["down"] = result.DownCount
	stats["degraded"] = result.DegradedCount
	stats["content_issue"] = result.ContentIssueCount
	stats["ssl_issue"] = result.SSLIssueCount

	return stats, nil
}

func (r *WebsiteRepository) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	query := `
		SELECT
			COUNT(*) as total_websites,
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) as total_up,
			COALESCE(SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END), 0) as total_down,
			COALESCE(SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END), 0) as total_degraded,
			COALESCE(SUM(CASE WHEN content_clean = FALSE THEN 1 ELSE 0 END), 0) as content_issues,
			COALESCE(SUM(CASE WHEN ssl_valid = FALSE OR ssl_expiry_date <= DATE_ADD(NOW(), INTERVAL 30 DAY) THEN 1 ELSE 0 END), 0) as ssl_expiring_soon,
			COALESCE(AVG(last_response_time), 0) as avg_response_time
		FROM websites WHERE is_active = TRUE
	`

	var stats domain.DashboardStats
	err := r.db.GetContext(ctx, &stats, query)
	if err != nil {
		return nil, err
	}

	// Calculate overall uptime percentage
	total := stats.TotalUp + stats.TotalDown + stats.TotalDegraded
	if total > 0 {
		stats.OverallUptime = float64(stats.TotalUp) / float64(total) * 100
	}

	return &stats, nil
}

// GetPublicStatusOverview returns the public status page overview
func (r *WebsiteRepository) GetPublicStatusOverview(ctx context.Context) (*domain.PublicStatusOverview, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'up' THEN 1 ELSE 0 END), 0) as up_count,
			COALESCE(SUM(CASE WHEN status = 'down' THEN 1 ELSE 0 END), 0) as down_count,
			COALESCE(SUM(CASE WHEN status = 'degraded' THEN 1 ELSE 0 END), 0) as degraded_count
		FROM websites WHERE is_active = TRUE
	`

	var result struct {
		Total         int `db:"total"`
		UpCount       int `db:"up_count"`
		DownCount     int `db:"down_count"`
		DegradedCount int `db:"degraded_count"`
	}

	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, err
	}

	// Determine system status
	systemStatus := "operational"
	if result.DownCount > 0 {
		if result.DownCount >= result.Total/2 {
			systemStatus = "major_outage"
		} else {
			systemStatus = "partial_outage"
		}
	} else if result.DegradedCount > 0 {
		systemStatus = "degraded"
	}

	// Calculate overall uptime
	overallUptime := float64(0)
	if result.Total > 0 {
		overallUptime = float64(result.UpCount) * 100 / float64(result.Total)
	}

	return &domain.PublicStatusOverview{
		SystemStatus:   systemStatus,
		OverallUptime:  overallUptime,
		TotalWebsites:  result.Total,
		OperationalCnt: result.UpCount,
		DegradedCnt:    result.DegradedCount,
		DownCnt:        result.DownCount,
	}, nil
}

// GetPublicServiceStatuses returns all services for public status page
func (r *WebsiteRepository) GetPublicServiceStatuses(ctx context.Context) ([]domain.PublicServiceStatus, error) {
	query := `
		SELECT
			id, name, url, status,
			COALESCE(last_response_time, 0) as response_time,
			last_checked_at
		FROM websites
		WHERE is_active = TRUE
		ORDER BY name ASC
	`

	type dbResult struct {
		ID            int64          `db:"id"`
		Name          string         `db:"name"`
		URL           string         `db:"url"`
		Status        string         `db:"status"`
		ResponseTime  int            `db:"response_time"`
		LastCheckedAt sql.NullTime   `db:"last_checked_at"`
	}

	var results []dbResult
	err := r.db.SelectContext(ctx, &results, query)
	if err != nil {
		return nil, err
	}

	services := make([]domain.PublicServiceStatus, 0, len(results))
	for _, r := range results {
		status := "operational"
		switch r.Status {
		case "down", "timeout", "error":
			status = "down"
		case "degraded":
			status = "degraded"
		}

		svc := domain.PublicServiceStatus{
			ID:           r.ID,
			Name:         r.Name,
			URL:          r.URL,
			Status:       status,
			ResponseTime: r.ResponseTime,
		}

		if r.LastCheckedAt.Valid {
			svc.LastChecked = r.LastCheckedAt.Time
		}

		services = append(services, svc)
	}

	return services, nil
}

// UpdateSecurityScore updates the security score for a website
func (r *WebsiteRepository) UpdateSecurityScore(ctx context.Context, id int64, score int, grade string) error {
	query := `UPDATE websites SET security_score = ?, security_grade = ?, security_checked_at = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, score, grade, id)
	return err
}

// GetPublicServiceStatus returns a single service status for public display
func (r *WebsiteRepository) GetPublicServiceStatus(ctx context.Context, id int64) (*domain.PublicServiceStatus, error) {
	query := `
		SELECT
			id, name, url, status,
			COALESCE(last_response_time, 0) as response_time,
			last_checked_at
		FROM websites
		WHERE id = ? AND is_active = TRUE
	`

	type dbResult struct {
		ID            int64          `db:"id"`
		Name          string         `db:"name"`
		URL           string         `db:"url"`
		Status        string         `db:"status"`
		ResponseTime  int            `db:"response_time"`
		LastCheckedAt sql.NullTime   `db:"last_checked_at"`
	}

	var result dbResult
	err := r.db.GetContext(ctx, &result, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	status := "operational"
	switch result.Status {
	case "down", "timeout", "error":
		status = "down"
	case "degraded":
		status = "degraded"
	}

	svc := &domain.PublicServiceStatus{
		ID:           result.ID,
		Name:         result.Name,
		URL:          result.URL,
		Status:       status,
		ResponseTime: result.ResponseTime,
	}

	if result.LastCheckedAt.Valid {
		svc.LastChecked = result.LastCheckedAt.Time
	}

	return svc, nil
}

// UpdateVulnerabilityStatus updates the vulnerability scan status for a website
func (r *WebsiteRepository) UpdateVulnerabilityStatus(ctx context.Context, websiteID int64, riskScore int, riskLevel string) error {
	query := `UPDATE websites SET vuln_risk_score = ?, vuln_risk_level = ?, vuln_last_scan = NOW() WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, riskScore, riskLevel, websiteID)
	return err
}

// BulkUpdateActive updates the is_active field for multiple websites
func (r *WebsiteRepository) BulkUpdateActive(ctx context.Context, ids []int64, isActive bool) (int64, error) {
	query, args, err := sqlx.In("UPDATE websites SET is_active = ? WHERE id IN (?)", isActive, ids)
	if err != nil {
		return 0, err
	}
	query = r.db.Rebind(query)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// BulkDelete deletes multiple websites by IDs
func (r *WebsiteRepository) BulkDelete(ctx context.Context, ids []int64) (int64, error) {
	query, args, err := sqlx.In("DELETE FROM websites WHERE id IN (?)", ids)
	if err != nil {
		return 0, err
	}
	query = r.db.Rebind(query)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// Count returns total number of websites
func (r *WebsiteRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	query := `SELECT COUNT(*) FROM websites`
	err := r.db.GetContext(ctx, &count, query)
	return count, err
}

// GetLastCheckTime returns the most recent check time across all websites
func (r *WebsiteRepository) GetLastCheckTime(ctx context.Context) (*time.Time, error) {
	var lastCheck sql.NullTime
	query := `SELECT MAX(last_checked_at) FROM websites WHERE last_checked_at IS NOT NULL`
	err := r.db.GetContext(ctx, &lastCheck, query)
	if err != nil {
		return nil, err
	}
	if !lastCheck.Valid {
		return nil, nil
	}
	return &lastCheck.Time, nil
}
