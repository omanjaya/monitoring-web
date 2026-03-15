package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type DorkRepository struct {
	db *sqlx.DB
}

func NewDorkRepository(db *sqlx.DB) *DorkRepository {
	return &DorkRepository{db: db}
}

// Pattern methods

func (r *DorkRepository) CreatePattern(ctx context.Context, pattern *domain.DorkPattern) error {
	query := `
		INSERT INTO dork_patterns (name, category, pattern, pattern_type, severity, description, is_active, is_default)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		pattern.Name,
		pattern.Category,
		pattern.Pattern,
		pattern.PatternType,
		pattern.Severity,
		pattern.Description,
		pattern.IsActive,
		pattern.IsDefault,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	pattern.ID = id
	return nil
}

func (r *DorkRepository) GetPattern(ctx context.Context, id int64) (*domain.DorkPattern, error) {
	var pattern domain.DorkPattern
	query := `SELECT id, name, category, pattern, pattern_type, severity, description, is_active, is_default, created_at, updated_at FROM dork_patterns WHERE id = ?`
	err := r.db.GetContext(ctx, &pattern, query, id)
	if err != nil {
		return nil, err
	}
	return &pattern, nil
}

func (r *DorkRepository) ListPatterns(ctx context.Context, filter domain.DorkPatternFilter) ([]domain.DorkPattern, error) {
	query := `SELECT id, name, category, pattern, pattern_type, severity, description, is_active, is_default, created_at, updated_at FROM dork_patterns WHERE 1=1`
	args := []interface{}{}

	if filter.Category != "" {
		query += " AND category = ?"
		args = append(args, filter.Category)
	}

	if filter.Severity != "" {
		query += " AND severity = ?"
		args = append(args, filter.Severity)
	}

	if filter.IsActive != nil {
		query += " AND is_active = ?"
		args = append(args, *filter.IsActive)
	}

	if filter.IsDefault != nil {
		query += " AND is_default = ?"
		args = append(args, *filter.IsDefault)
	}

	query += " ORDER BY category, severity DESC, name"

	var patterns []domain.DorkPattern
	err := r.db.SelectContext(ctx, &patterns, query, args...)
	if err != nil {
		return nil, err
	}
	return patterns, nil
}

func (r *DorkRepository) GetActivePatterns(ctx context.Context) ([]domain.DorkPattern, error) {
	isActive := true
	return r.ListPatterns(ctx, domain.DorkPatternFilter{IsActive: &isActive})
}

func (r *DorkRepository) GetPatternsByCategory(ctx context.Context, category domain.DorkCategory) ([]domain.DorkPattern, error) {
	isActive := true
	return r.ListPatterns(ctx, domain.DorkPatternFilter{
		Category: category,
		IsActive: &isActive,
	})
}

func (r *DorkRepository) UpdatePattern(ctx context.Context, pattern *domain.DorkPattern) error {
	query := `
		UPDATE dork_patterns
		SET name = ?, category = ?, pattern = ?, pattern_type = ?, severity = ?, description = ?, is_active = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		pattern.Name,
		pattern.Category,
		pattern.Pattern,
		pattern.PatternType,
		pattern.Severity,
		pattern.Description,
		pattern.IsActive,
		pattern.ID,
	)
	return err
}

func (r *DorkRepository) DeletePattern(ctx context.Context, id int64) error {
	query := `DELETE FROM dork_patterns WHERE id = ? AND is_default = FALSE`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Scan result methods

func (r *DorkRepository) CreateScanResult(ctx context.Context, result *domain.DorkScanResult) error {
	categoriesJSON, _ := json.Marshal(result.CategoriesScanned)

	query := `
		INSERT INTO dork_scan_results (website_id, scan_type, status, total_pages_scanned, total_detections,
			critical_count, high_count, medium_count, low_count, ai_filtered_count, categories_scanned, started_at, completed_at, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	res, err := r.db.ExecContext(ctx, query,
		result.WebsiteID,
		result.ScanType,
		result.Status,
		result.TotalPagesScanned,
		result.TotalDetections,
		result.CriticalCount,
		result.HighCount,
		result.MediumCount,
		result.LowCount,
		result.AIFilteredCount,
		categoriesJSON,
		result.StartedAt,
		result.CompletedAt,
		result.ErrorMessage,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	result.ID = id
	return nil
}

func (r *DorkRepository) GetScanResult(ctx context.Context, id int64) (*domain.DorkScanResult, error) {
	var result scanResultRow
	query := `
		SELECT id, website_id, scan_type, status, total_pages_scanned, total_detections,
			critical_count, high_count, medium_count, low_count, categories_scanned,
			started_at, completed_at, error_message, created_at
		FROM dork_scan_results WHERE id = ?
	`
	err := r.db.GetContext(ctx, &result, query, id)
	if err != nil {
		return nil, err
	}
	return result.toDomain(), nil
}

type scanResultRow struct {
	ID                int64          `db:"id"`
	WebsiteID         int64          `db:"website_id"`
	ScanType          string         `db:"scan_type"`
	Status            string         `db:"status"`
	TotalPagesScanned int            `db:"total_pages_scanned"`
	TotalDetections   int            `db:"total_detections"`
	CriticalCount     int            `db:"critical_count"`
	HighCount         int            `db:"high_count"`
	MediumCount       int            `db:"medium_count"`
	LowCount          int            `db:"low_count"`
	CategoriesScanned []byte         `db:"categories_scanned"`
	StartedAt         sql.NullTime   `db:"started_at"`
	CompletedAt       sql.NullTime   `db:"completed_at"`
	ErrorMessage      sql.NullString `db:"error_message"`
	CreatedAt         time.Time      `db:"created_at"`
}

func (r *scanResultRow) toDomain() *domain.DorkScanResult {
	result := &domain.DorkScanResult{
		ID:                r.ID,
		WebsiteID:         r.WebsiteID,
		ScanType:          r.ScanType,
		Status:            r.Status,
		TotalPagesScanned: r.TotalPagesScanned,
		TotalDetections:   r.TotalDetections,
		CriticalCount:     r.CriticalCount,
		HighCount:         r.HighCount,
		MediumCount:       r.MediumCount,
		LowCount:          r.LowCount,
		CreatedAt:         r.CreatedAt,
	}

	if r.StartedAt.Valid {
		result.StartedAt = &r.StartedAt.Time
	}
	if r.CompletedAt.Valid {
		result.CompletedAt = &r.CompletedAt.Time
	}
	if r.ErrorMessage.Valid {
		result.ErrorMessage = r.ErrorMessage.String
	}
	if len(r.CategoriesScanned) > 0 {
		json.Unmarshal(r.CategoriesScanned, &result.CategoriesScanned)
	}

	return result
}

func (r *DorkRepository) UpdateScanResult(ctx context.Context, result *domain.DorkScanResult) error {
	categoriesJSON, _ := json.Marshal(result.CategoriesScanned)

	query := `
		UPDATE dork_scan_results
		SET status = ?, total_pages_scanned = ?, total_detections = ?,
			critical_count = ?, high_count = ?, medium_count = ?, low_count = ?,
			categories_scanned = ?, completed_at = ?, error_message = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query,
		result.Status,
		result.TotalPagesScanned,
		result.TotalDetections,
		result.CriticalCount,
		result.HighCount,
		result.MediumCount,
		result.LowCount,
		categoriesJSON,
		result.CompletedAt,
		result.ErrorMessage,
		result.ID,
	)
	return err
}

func (r *DorkRepository) ListScanResults(ctx context.Context, websiteID int64, limit int) ([]domain.DorkScanResult, error) {
	query := `
		SELECT id, website_id, scan_type, status, total_pages_scanned, total_detections,
			critical_count, high_count, medium_count, low_count, categories_scanned,
			started_at, completed_at, error_message, created_at
		FROM dork_scan_results
		WHERE website_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`

	var rows []scanResultRow
	err := r.db.SelectContext(ctx, &rows, query, websiteID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]domain.DorkScanResult, len(rows))
	for i, row := range rows {
		results[i] = *row.toDomain()
	}
	return results, nil
}

func (r *DorkRepository) GetLatestScanResult(ctx context.Context, websiteID int64) (*domain.DorkScanResult, error) {
	results, err := r.ListScanResults(ctx, websiteID, 1)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, sql.ErrNoRows
	}
	return &results[0], nil
}

// Detection methods

func (r *DorkRepository) CreateDetection(ctx context.Context, detection *domain.DorkDetection) error {
	query := `
		INSERT INTO dork_detections (scan_result_id, website_id, pattern_id, pattern_name, category,
			severity, url, matched_content, context, confidence, ai_verified, is_false_positive, is_resolved)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var patternID *int64
	if detection.PatternID > 0 {
		patternID = &detection.PatternID
	}

	result, err := r.db.ExecContext(ctx, query,
		detection.ScanResultID,
		detection.WebsiteID,
		patternID,
		detection.PatternName,
		detection.Category,
		detection.Severity,
		detection.URL,
		detection.MatchedContent,
		detection.Context,
		detection.Confidence,
		detection.AIVerified,
		detection.IsFalsePositive,
		detection.IsResolved,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	detection.ID = id
	return nil
}

func (r *DorkRepository) CreateDetections(ctx context.Context, detections []domain.DorkDetection) error {
	if len(detections) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO dork_detections (scan_result_id, website_id, pattern_id, pattern_name, category,
			severity, url, matched_content, context, confidence, ai_verified, is_false_positive, is_resolved)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for i := range detections {
		d := &detections[i]
		var patternID *int64
		if d.PatternID > 0 {
			patternID = &d.PatternID
		}

		result, err := stmt.ExecContext(ctx,
			d.ScanResultID,
			d.WebsiteID,
			patternID,
			d.PatternName,
			d.Category,
			d.Severity,
			d.URL,
			d.MatchedContent,
			d.Context,
			d.Confidence,
			d.AIVerified,
			d.IsFalsePositive,
			d.IsResolved,
		)
		if err != nil {
			return err
		}
		id, _ := result.LastInsertId()
		d.ID = id
	}

	return tx.Commit()
}

func (r *DorkRepository) GetDetection(ctx context.Context, id int64) (*domain.DorkDetection, error) {
	var detection detectionRow
	query := `
		SELECT d.id, d.scan_result_id, d.website_id, w.name as website_name, w.url as website_url,
			d.pattern_id, d.pattern_name, d.category, d.severity,
			d.url, d.matched_content, d.context, d.confidence, d.ai_verified, d.is_false_positive, d.is_resolved,
			d.resolved_at, d.resolved_by, d.notes, d.created_at
		FROM dork_detections d LEFT JOIN websites w ON d.website_id = w.id WHERE d.id = ?
	`
	err := r.db.GetContext(ctx, &detection, query, id)
	if err != nil {
		return nil, err
	}
	return detection.toDomain(), nil
}

type detectionRow struct {
	ID              int64          `db:"id"`
	ScanResultID    int64          `db:"scan_result_id"`
	WebsiteID       int64          `db:"website_id"`
	WebsiteName     sql.NullString `db:"website_name"`
	WebsiteURL      sql.NullString `db:"website_url"`
	PatternID       sql.NullInt64  `db:"pattern_id"`
	PatternName     string         `db:"pattern_name"`
	Category        string         `db:"category"`
	Severity        string         `db:"severity"`
	URL             string         `db:"url"`
	MatchedContent  sql.NullString `db:"matched_content"`
	Context         sql.NullString `db:"context"`
	Confidence      float64        `db:"confidence"`
	AIVerified      bool           `db:"ai_verified"`
	IsFalsePositive bool           `db:"is_false_positive"`
	IsResolved      bool           `db:"is_resolved"`
	ResolvedAt      sql.NullTime   `db:"resolved_at"`
	ResolvedBy      sql.NullString `db:"resolved_by"`
	Notes           sql.NullString `db:"notes"`
	CreatedAt       time.Time      `db:"created_at"`
}

func (d *detectionRow) toDomain() *domain.DorkDetection {
	detection := &domain.DorkDetection{
		ID:              d.ID,
		ScanResultID:    d.ScanResultID,
		WebsiteID:       d.WebsiteID,
		PatternName:     d.PatternName,
		Category:        domain.DorkCategory(d.Category),
		Severity:        domain.DorkSeverity(d.Severity),
		URL:             d.URL,
		Confidence:      d.Confidence,
		AIVerified:      d.AIVerified,
		IsFalsePositive: d.IsFalsePositive,
		IsResolved:      d.IsResolved,
		CreatedAt:       d.CreatedAt,
	}

	if d.WebsiteName.Valid {
		detection.WebsiteName = d.WebsiteName.String
	}
	if d.WebsiteURL.Valid {
		detection.WebsiteURL = d.WebsiteURL.String
	}
	if d.PatternID.Valid {
		detection.PatternID = d.PatternID.Int64
	}
	if d.MatchedContent.Valid {
		detection.MatchedContent = d.MatchedContent.String
	}
	if d.Context.Valid {
		detection.Context = d.Context.String
	}
	if d.ResolvedAt.Valid {
		detection.ResolvedAt = &d.ResolvedAt.Time
	}
	if d.ResolvedBy.Valid {
		detection.ResolvedBy = d.ResolvedBy.String
	}
	if d.Notes.Valid {
		detection.Notes = d.Notes.String
	}
	// Use created_at as detected_at since they are the same (row is created at detection time)
	detection.DetectedAt = d.CreatedAt

	return detection
}

func (r *DorkRepository) ListDetections(ctx context.Context, filter domain.DorkDetectionFilter) ([]domain.DorkDetection, int64, error) {
	baseQuery := `FROM dork_detections d LEFT JOIN websites w ON d.website_id = w.id WHERE 1=1`
	args := []interface{}{}

	if filter.WebsiteID > 0 {
		baseQuery += " AND d.website_id = ?"
		args = append(args, filter.WebsiteID)
	}

	if filter.ScanResultID > 0 {
		baseQuery += " AND d.scan_result_id = ?"
		args = append(args, filter.ScanResultID)
	}

	if filter.Category != "" {
		baseQuery += " AND d.category = ?"
		args = append(args, filter.Category)
	}

	if filter.Severity != "" {
		baseQuery += " AND d.severity = ?"
		args = append(args, filter.Severity)
	}

	if filter.IsResolved != nil {
		baseQuery += " AND d.is_resolved = ?"
		args = append(args, *filter.IsResolved)
	}

	if filter.IsFalsePositive != nil {
		baseQuery += " AND d.is_false_positive = ?"
		args = append(args, *filter.IsFalsePositive)
	}

	// Count total
	var total int64
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// Get data
	selectQuery := `
		SELECT d.id, d.scan_result_id, d.website_id, w.name as website_name, w.url as website_url,
			d.pattern_id, d.pattern_name, d.category, d.severity,
			d.url, d.matched_content, d.context, d.confidence, d.ai_verified, d.is_false_positive, d.is_resolved,
			d.resolved_at, d.resolved_by, d.notes, d.created_at ` + baseQuery + ` ORDER BY d.created_at DESC`

	if filter.Limit > 0 {
		selectQuery += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		selectQuery += " OFFSET ?"
		args = append(args, filter.Offset)
	}

	var rows []detectionRow
	err = r.db.SelectContext(ctx, &rows, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	detections := make([]domain.DorkDetection, len(rows))
	for i, row := range rows {
		detections[i] = *row.toDomain()
	}

	return detections, total, nil
}

func (r *DorkRepository) GetDetectionsByScanResult(ctx context.Context, scanResultID int64) ([]domain.DorkDetection, error) {
	detections, _, err := r.ListDetections(ctx, domain.DorkDetectionFilter{
		ScanResultID: scanResultID,
	})
	return detections, err
}

func (r *DorkRepository) GetUnresolvedDetections(ctx context.Context, websiteID int64) ([]domain.DorkDetection, error) {
	isResolved := false
	isFalsePositive := false
	detections, _, err := r.ListDetections(ctx, domain.DorkDetectionFilter{
		WebsiteID:       websiteID,
		IsResolved:      &isResolved,
		IsFalsePositive: &isFalsePositive,
	})
	return detections, err
}

func (r *DorkRepository) MarkAsResolved(ctx context.Context, id int64, resolvedBy string, notes string) error {
	query := `
		UPDATE dork_detections
		SET is_resolved = TRUE, resolved_at = NOW(), resolved_by = ?, notes = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, resolvedBy, notes, id)
	return err
}

func (r *DorkRepository) MarkAsFalsePositive(ctx context.Context, id int64, notes string) error {
	query := `
		UPDATE dork_detections
		SET is_false_positive = TRUE, is_resolved = TRUE, resolved_at = NOW(), notes = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, notes, id)
	return err
}

// GetUnverifiedDetections returns all active detections not yet verified by AI
func (r *DorkRepository) GetUnverifiedDetections(ctx context.Context) ([]domain.DorkDetection, error) {
	query := `
		SELECT d.id, d.scan_result_id, d.website_id, w.name as website_name, w.url as website_url,
			d.pattern_id, d.pattern_name, d.category, d.severity,
			d.url, d.matched_content, d.context, d.confidence, d.ai_verified, d.is_false_positive, d.is_resolved,
			d.resolved_at, d.resolved_by, d.notes, d.created_at
		FROM dork_detections d LEFT JOIN websites w ON d.website_id = w.id
		WHERE d.ai_verified = FALSE AND d.is_resolved = FALSE AND d.is_false_positive = FALSE
		ORDER BY d.created_at DESC
	`
	var rows []detectionRow
	if err := r.db.SelectContext(ctx, &rows, query); err != nil {
		return nil, err
	}
	detections := make([]domain.DorkDetection, len(rows))
	for i, row := range rows {
		detections[i] = *row.toDomain()
	}
	return detections, nil
}

// MarkDetectionAIVerified marks a detection as AI-verified
func (r *DorkRepository) MarkDetectionAIVerified(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE dork_detections SET ai_verified = TRUE WHERE id = ?`, id)
	return err
}

// MarkAsFalsePositiveByAI marks a detection as false positive by AI verification
func (r *DorkRepository) MarkAsFalsePositiveByAI(ctx context.Context, id int64, reason string) error {
	query := `
		UPDATE dork_detections
		SET is_false_positive = TRUE, ai_verified = TRUE, notes = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, "AI: "+reason, id)
	return err
}

// Website settings methods

func (r *DorkRepository) GetWebsiteSettings(ctx context.Context, websiteID int64) (*domain.WebsiteDorkSettings, error) {
	var settings settingsRow
	query := `
		SELECT id, website_id, is_enabled, scan_frequency, scan_depth, max_pages,
			categories_enabled, excluded_paths, last_scan_at, next_scan_at, created_at, updated_at
		FROM website_dork_settings WHERE website_id = ?
	`
	err := r.db.GetContext(ctx, &settings, query, websiteID)
	if err == sql.ErrNoRows {
		// Return default settings
		return &domain.WebsiteDorkSettings{
			WebsiteID:     websiteID,
			IsEnabled:     true,
			ScanFrequency: "daily",
			ScanDepth:     3,
			MaxPages:      50,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return settings.toDomain(), nil
}

type settingsRow struct {
	ID                int64          `db:"id"`
	WebsiteID         int64          `db:"website_id"`
	IsEnabled         bool           `db:"is_enabled"`
	ScanFrequency     string         `db:"scan_frequency"`
	ScanDepth         int            `db:"scan_depth"`
	MaxPages          int            `db:"max_pages"`
	CategoriesEnabled []byte         `db:"categories_enabled"`
	ExcludedPaths     []byte         `db:"excluded_paths"`
	LastScanAt        sql.NullTime   `db:"last_scan_at"`
	NextScanAt        sql.NullTime   `db:"next_scan_at"`
	CreatedAt         time.Time      `db:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at"`
}

func (s *settingsRow) toDomain() *domain.WebsiteDorkSettings {
	settings := &domain.WebsiteDorkSettings{
		ID:            s.ID,
		WebsiteID:     s.WebsiteID,
		IsEnabled:     s.IsEnabled,
		ScanFrequency: s.ScanFrequency,
		ScanDepth:     s.ScanDepth,
		MaxPages:      s.MaxPages,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}

	if s.LastScanAt.Valid {
		settings.LastScanAt = &s.LastScanAt.Time
	}
	if s.NextScanAt.Valid {
		settings.NextScanAt = &s.NextScanAt.Time
	}
	if len(s.CategoriesEnabled) > 0 {
		json.Unmarshal(s.CategoriesEnabled, &settings.CategoriesEnabled)
	}
	if len(s.ExcludedPaths) > 0 {
		json.Unmarshal(s.ExcludedPaths, &settings.ExcludedPaths)
	}

	return settings
}

func (r *DorkRepository) SaveWebsiteSettings(ctx context.Context, settings *domain.WebsiteDorkSettings) error {
	categoriesJSON, _ := json.Marshal(settings.CategoriesEnabled)
	pathsJSON, _ := json.Marshal(settings.ExcludedPaths)

	query := `
		INSERT INTO website_dork_settings (website_id, is_enabled, scan_frequency, scan_depth, max_pages,
			categories_enabled, excluded_paths, last_scan_at, next_scan_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			is_enabled = VALUES(is_enabled),
			scan_frequency = VALUES(scan_frequency),
			scan_depth = VALUES(scan_depth),
			max_pages = VALUES(max_pages),
			categories_enabled = VALUES(categories_enabled),
			excluded_paths = VALUES(excluded_paths),
			last_scan_at = VALUES(last_scan_at),
			next_scan_at = VALUES(next_scan_at)
	`

	_, err := r.db.ExecContext(ctx, query,
		settings.WebsiteID,
		settings.IsEnabled,
		settings.ScanFrequency,
		settings.ScanDepth,
		settings.MaxPages,
		categoriesJSON,
		pathsJSON,
		settings.LastScanAt,
		settings.NextScanAt,
	)
	return err
}

func (r *DorkRepository) UpdateLastScan(ctx context.Context, websiteID int64) error {
	query := `
		UPDATE website_dork_settings
		SET last_scan_at = NOW()
		WHERE website_id = ?
	`
	_, err := r.db.ExecContext(ctx, query, websiteID)
	return err
}

// Statistics methods

func (r *DorkRepository) GetDetectionStats(ctx context.Context, websiteID int64) (*domain.DorkDetectionStats, error) {
	stats := &domain.DorkDetectionStats{
		WebsiteID:        websiteID,
		ByCategory:       make(map[domain.DorkCategory]int),
		BySeverity:       make(map[domain.DorkSeverity]int),
	}

	// Get total detections
	var totalRow struct {
		Total      int `db:"total"`
		Unresolved int `db:"unresolved"`
	}
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN is_resolved = FALSE AND is_false_positive = FALSE THEN 1 ELSE 0 END), 0) as unresolved
		FROM dork_detections WHERE website_id = ?
	`
	if err := r.db.GetContext(ctx, &totalRow, query, websiteID); err != nil {
		return nil, err
	}
	stats.TotalDetections = totalRow.Total
	stats.UnresolvedCount = totalRow.Unresolved

	// Get by category
	var categoryRows []struct {
		Category string `db:"category"`
		Count    int    `db:"count"`
	}
	query = `SELECT category, COUNT(*) as count FROM dork_detections WHERE website_id = ? GROUP BY category`
	if err := r.db.SelectContext(ctx, &categoryRows, query, websiteID); err != nil {
		return nil, err
	}
	for _, row := range categoryRows {
		stats.ByCategory[domain.DorkCategory(row.Category)] = row.Count
	}

	// Get by severity
	var severityRows []struct {
		Severity string `db:"severity"`
		Count    int    `db:"count"`
	}
	query = `SELECT severity, COUNT(*) as count FROM dork_detections WHERE website_id = ? GROUP BY severity`
	if err := r.db.SelectContext(ctx, &severityRows, query, websiteID); err != nil {
		return nil, err
	}
	for _, row := range severityRows {
		stats.BySeverity[domain.DorkSeverity(row.Severity)] = row.Count
	}

	// Get last scan
	var lastScan sql.NullTime
	query = `SELECT MAX(created_at) FROM dork_scan_results WHERE website_id = ?`
	if err := r.db.GetContext(ctx, &lastScan, query, websiteID); err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if lastScan.Valid {
		stats.LastScanAt = &lastScan.Time
	}

	return stats, nil
}

func (r *DorkRepository) GetOverallStats(ctx context.Context) (*domain.DorkOverallStats, error) {
	stats := &domain.DorkOverallStats{
		ByCategory: make(map[domain.DorkCategory]int),
		BySeverity: make(map[domain.DorkSeverity]int),
	}

	// Get total stats
	var totalRow struct {
		TotalScans       int `db:"total_scans"`
		TotalDetections  int `db:"total_detections"`
		TotalUnresolved  int `db:"total_unresolved"`
		WebsitesAffected int `db:"websites_affected"`
	}
	query := `
		SELECT
			(SELECT COUNT(*) FROM dork_scan_results) as total_scans,
			(SELECT COUNT(*) FROM dork_detections) as total_detections,
			(SELECT COUNT(*) FROM dork_detections WHERE is_resolved = FALSE AND is_false_positive = FALSE) as total_unresolved,
			(SELECT COUNT(DISTINCT website_id) FROM dork_detections WHERE is_resolved = FALSE AND is_false_positive = FALSE) as websites_affected
	`
	if err := r.db.GetContext(ctx, &totalRow, query); err != nil {
		return nil, err
	}
	stats.TotalScans = totalRow.TotalScans
	stats.TotalDetections = totalRow.TotalDetections
	stats.UnresolvedCount = totalRow.TotalUnresolved
	stats.WebsitesAffected = totalRow.WebsitesAffected

	// Get by category
	var categoryRows []struct {
		Category string `db:"category"`
		Count    int    `db:"count"`
	}
	query = `SELECT category, COUNT(*) as count FROM dork_detections WHERE is_resolved = FALSE GROUP BY category`
	if err := r.db.SelectContext(ctx, &categoryRows, query); err != nil {
		return nil, err
	}
	for _, row := range categoryRows {
		stats.ByCategory[domain.DorkCategory(row.Category)] = row.Count
	}

	// Get by severity
	var severityRows []struct {
		Severity string `db:"severity"`
		Count    int    `db:"count"`
	}
	query = `SELECT severity, COUNT(*) as count FROM dork_detections WHERE is_resolved = FALSE GROUP BY severity`
	if err := r.db.SelectContext(ctx, &severityRows, query); err != nil {
		return nil, err
	}
	for _, row := range severityRows {
		stats.BySeverity[domain.DorkSeverity(row.Severity)] = row.Count
	}

	return stats, nil
}

// ClearAllDetections removes all detections and scan results
func (r *DorkRepository) ClearAllDetections(ctx context.Context) (int64, error) {
	// Delete detections first (FK constraint)
	result, err := r.db.ExecContext(ctx, "DELETE FROM dork_detections")
	if err != nil {
		return 0, err
	}
	deleted, _ := result.RowsAffected()

	// Delete scan results
	r.db.ExecContext(ctx, "DELETE FROM dork_scan_results")

	return deleted, nil
}

// Cleanup method
func (r *DorkRepository) CleanupOldResults(ctx context.Context, olderThan time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, "DELETE FROM dork_scan_results WHERE created_at < ?", olderThan)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
