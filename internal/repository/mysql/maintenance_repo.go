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

type MaintenanceRepository struct {
	db *sqlx.DB
}

func NewMaintenanceRepository(db *sqlx.DB) *MaintenanceRepository {
	return &MaintenanceRepository{db: db}
}

// Create creates a new maintenance window
func (r *MaintenanceRepository) Create(ctx context.Context, m *domain.MaintenanceWindowCreate, userID int64) (int64, error) {
	startTime, err := time.Parse(time.RFC3339, m.StartTime)
	if err != nil {
		return 0, fmt.Errorf("invalid start_time format: %w", err)
	}

	endTime, err := time.Parse(time.RFC3339, m.EndTime)
	if err != nil {
		return 0, fmt.Errorf("invalid end_time format: %w", err)
	}

	if endTime.Before(startTime) {
		return 0, fmt.Errorf("end_time must be after start_time")
	}

	// Determine initial status
	status := domain.MaintenanceScheduled
	if time.Now().After(startTime) && time.Now().Before(endTime) {
		status = domain.MaintenanceInProgress
	}

	query := `
		INSERT INTO maintenance_windows (website_id, title, description, status, start_time, end_time, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query,
		m.WebsiteID, m.Title, m.Description, status, startTime, endTime, userID,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetByID retrieves a maintenance window by ID
func (r *MaintenanceRepository) GetByID(ctx context.Context, id int64) (*domain.MaintenanceWindow, error) {
	var mw domain.MaintenanceWindow
	query := `
		SELECT
			mw.*,
			COALESCE(w.name, 'Semua Layanan') as website_name,
			COALESCE(u.full_name, '') as creator_name
		FROM maintenance_windows mw
		LEFT JOIN websites w ON mw.website_id = w.id
		LEFT JOIN users u ON mw.created_by = u.id
		WHERE mw.id = ?
	`

	err := r.db.GetContext(ctx, &mw, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &mw, nil
}

// GetAll retrieves all maintenance windows with filtering
func (r *MaintenanceRepository) GetAll(ctx context.Context, filter domain.MaintenanceFilter) ([]domain.MaintenanceWindow, int, error) {
	var mws []domain.MaintenanceWindow
	var total int

	// Build query
	baseQuery := `FROM maintenance_windows mw
		LEFT JOIN websites w ON mw.website_id = w.id
		LEFT JOIN users u ON mw.created_by = u.id
		WHERE 1=1`
	var conditions []string
	var args []interface{}

	if filter.WebsiteID != nil {
		conditions = append(conditions, "(mw.website_id = ? OR mw.website_id IS NULL)")
		args = append(args, *filter.WebsiteID)
	}
	if filter.Status != nil {
		conditions = append(conditions, "mw.status = ?")
		args = append(args, *filter.Status)
	}
	if !filter.IncludePast {
		conditions = append(conditions, "(mw.end_time >= NOW() OR mw.status = 'in_progress')")
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

	// Pagination
	limit := 20
	if filter.Limit > 0 && filter.Limit <= 100 {
		limit = filter.Limit
	}
	offset := 0
	if filter.Page > 1 {
		offset = (filter.Page - 1) * limit
	}

	// Get data
	dataQuery := fmt.Sprintf(`
		SELECT
			mw.*,
			COALESCE(w.name, 'Semua Layanan') as website_name,
			COALESCE(u.full_name, '') as creator_name
		%s ORDER BY mw.start_time DESC LIMIT ? OFFSET ?`, baseQuery)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &mws, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return mws, total, nil
}

// Update updates a maintenance window
func (r *MaintenanceRepository) Update(ctx context.Context, id int64, m *domain.MaintenanceWindowUpdate) error {
	var sets []string
	var args []interface{}

	if m.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *m.Title)
	}
	if m.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *m.Description)
	}
	if m.StartTime != nil {
		startTime, err := time.Parse(time.RFC3339, *m.StartTime)
		if err != nil {
			return fmt.Errorf("invalid start_time format: %w", err)
		}
		sets = append(sets, "start_time = ?")
		args = append(args, startTime)
	}
	if m.EndTime != nil {
		endTime, err := time.Parse(time.RFC3339, *m.EndTime)
		if err != nil {
			return fmt.Errorf("invalid end_time format: %w", err)
		}
		sets = append(sets, "end_time = ?")
		args = append(args, endTime)
	}
	if m.Status != nil {
		sets = append(sets, "status = ?")
		args = append(args, *m.Status)
	}

	if len(sets) == 0 {
		return nil
	}

	query := fmt.Sprintf("UPDATE maintenance_windows SET %s WHERE id = ?", strings.Join(sets, ", "))
	args = append(args, id)

	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Delete deletes a maintenance window
func (r *MaintenanceRepository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM maintenance_windows WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// GetActiveForWebsite checks if there's an active maintenance window for a website
func (r *MaintenanceRepository) GetActiveForWebsite(ctx context.Context, websiteID int64) (*domain.MaintenanceWindow, error) {
	var mw domain.MaintenanceWindow
	query := `
		SELECT * FROM maintenance_windows
		WHERE (website_id = ? OR website_id IS NULL)
		AND status IN ('scheduled', 'in_progress')
		AND start_time <= NOW()
		AND end_time >= NOW()
		ORDER BY website_id DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &mw, query, websiteID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &mw, nil
}

// GetUpcomingMaintenance returns scheduled maintenance windows
func (r *MaintenanceRepository) GetUpcomingMaintenance(ctx context.Context, limit int) ([]domain.MaintenanceWindow, error) {
	var mws []domain.MaintenanceWindow
	query := `
		SELECT
			mw.*,
			COALESCE(w.name, 'Semua Layanan') as website_name
		FROM maintenance_windows mw
		LEFT JOIN websites w ON mw.website_id = w.id
		WHERE mw.status = 'scheduled' AND mw.start_time > NOW()
		ORDER BY mw.start_time ASC
		LIMIT ?
	`

	err := r.db.SelectContext(ctx, &mws, query, limit)
	if err != nil {
		return nil, err
	}

	return mws, nil
}

// GetCurrentMaintenance returns currently active maintenance windows
func (r *MaintenanceRepository) GetCurrentMaintenance(ctx context.Context) ([]domain.MaintenanceWindow, error) {
	var mws []domain.MaintenanceWindow
	query := `
		SELECT
			mw.*,
			COALESCE(w.name, 'Semua Layanan') as website_name
		FROM maintenance_windows mw
		LEFT JOIN websites w ON mw.website_id = w.id
		WHERE (mw.status = 'in_progress' OR
			(mw.status = 'scheduled' AND mw.start_time <= NOW() AND mw.end_time >= NOW()))
		ORDER BY mw.start_time ASC
	`

	err := r.db.SelectContext(ctx, &mws, query)
	if err != nil {
		return nil, err
	}

	return mws, nil
}

// UpdateStatuses updates maintenance window statuses based on time
func (r *MaintenanceRepository) UpdateStatuses(ctx context.Context) error {
	// Update scheduled -> in_progress
	_, err := r.db.ExecContext(ctx, `
		UPDATE maintenance_windows
		SET status = 'in_progress'
		WHERE status = 'scheduled' AND start_time <= NOW() AND end_time > NOW()
	`)
	if err != nil {
		return err
	}

	// Update in_progress -> completed
	_, err = r.db.ExecContext(ctx, `
		UPDATE maintenance_windows
		SET status = 'completed'
		WHERE status IN ('scheduled', 'in_progress') AND end_time <= NOW()
	`)
	return err
}

// GetPublicMaintenance returns maintenance info for public status page
func (r *MaintenanceRepository) GetPublicMaintenance(ctx context.Context) ([]domain.PublicMaintenanceWindow, error) {
	query := `
		SELECT
			mw.id,
			mw.title,
			COALESCE(mw.description, '') as description,
			mw.status,
			mw.start_time,
			mw.end_time,
			COALESCE(w.name, '') as service_name
		FROM maintenance_windows mw
		LEFT JOIN websites w ON mw.website_id = w.id
		WHERE mw.status IN ('scheduled', 'in_progress')
		AND mw.end_time >= NOW()
		ORDER BY mw.start_time ASC
	`

	var results []struct {
		ID          int64     `db:"id"`
		Title       string    `db:"title"`
		Description string    `db:"description"`
		Status      string    `db:"status"`
		StartTime   time.Time `db:"start_time"`
		EndTime     time.Time `db:"end_time"`
		ServiceName string    `db:"service_name"`
	}

	err := r.db.SelectContext(ctx, &results, query)
	if err != nil {
		return nil, err
	}

	mws := make([]domain.PublicMaintenanceWindow, 0, len(results))
	for _, r := range results {
		status := "scheduled"
		if r.Status == "in_progress" || (r.Status == "scheduled" && time.Now().After(r.StartTime)) {
			status = "in_progress"
		}

		mws = append(mws, domain.PublicMaintenanceWindow{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Status:      status,
			StartTime:   r.StartTime,
			EndTime:     r.EndTime,
			ServiceName: r.ServiceName,
		})
	}

	return mws, nil
}
