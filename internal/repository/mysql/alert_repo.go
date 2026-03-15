package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type AlertRepository struct {
	db *sqlx.DB
}

func NewAlertRepository(db *sqlx.DB) *AlertRepository {
	return &AlertRepository{db: db}
}

func (r *AlertRepository) Create(ctx context.Context, a *domain.AlertCreate) (int64, error) {
	query := `
		INSERT INTO alerts (website_id, type, severity, title, message, context)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	var contextJSON []byte
	var err error
	if a.Context != nil {
		contextJSON, err = json.Marshal(a.Context)
		if err != nil {
			return 0, err
		}
	}

	result, err := r.db.ExecContext(ctx, query,
		a.WebsiteID, a.Type, a.Severity, a.Title, a.Message, contextJSON,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (r *AlertRepository) GetByID(ctx context.Context, id int64) (*domain.Alert, error) {
	var alert domain.Alert
	query := `SELECT * FROM alerts WHERE id = ?`

	err := r.db.GetContext(ctx, &alert, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &alert, nil
}

func (r *AlertRepository) GetAll(ctx context.Context, filter domain.AlertFilter) ([]domain.Alert, int, error) {
	var alerts []domain.Alert
	var total int

	// Build query
	baseQuery := `FROM alerts a WHERE 1=1`
	var conditions []string
	var args []interface{}

	if filter.WebsiteID != nil {
		conditions = append(conditions, "a.website_id = ?")
		args = append(args, *filter.WebsiteID)
	}
	if filter.Type != nil {
		conditions = append(conditions, "a.type = ?")
		args = append(args, *filter.Type)
	}
	if filter.Severity != nil {
		conditions = append(conditions, "a.severity = ?")
		args = append(args, *filter.Severity)
	}
	if filter.IsResolved != nil {
		conditions = append(conditions, "a.is_resolved = ?")
		args = append(args, *filter.IsResolved)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "a.created_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "a.created_at <= ?")
		args = append(args, *filter.EndDate)
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
		dataQuery := fmt.Sprintf("SELECT a.* %s ORDER BY a.created_at DESC", baseQuery)
		err = r.db.SelectContext(ctx, &alerts, dataQuery, args...)
	} else {
		limit := 20
		if filter.Limit > 0 && filter.Limit <= 100 {
			limit = filter.Limit
		}
		offset := 0
		if filter.Page > 1 {
			offset = (filter.Page - 1) * limit
		}

		dataQuery := fmt.Sprintf("SELECT a.* %s ORDER BY a.created_at DESC LIMIT ? OFFSET ?", baseQuery)
		args = append(args, limit, offset)
		err = r.db.SelectContext(ctx, &alerts, dataQuery, args...)
	}
	if err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

func (r *AlertRepository) GetActiveAlerts(ctx context.Context) ([]domain.Alert, error) {
	var alerts []domain.Alert
	query := `
		SELECT * FROM alerts
		WHERE is_resolved = FALSE
		ORDER BY
			CASE severity
				WHEN 'critical' THEN 1
				WHEN 'warning' THEN 2
				ELSE 3
			END,
			created_at DESC
	`

	err := r.db.SelectContext(ctx, &alerts, query)
	if err != nil {
		return nil, err
	}

	return alerts, nil
}

func (r *AlertRepository) GetSummary(ctx context.Context) (*domain.AlertSummary, error) {
	query := `
		SELECT
			COUNT(*) as total_active,
			COALESCE(SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END), 0) as critical,
			COALESCE(SUM(CASE WHEN severity = 'warning' THEN 1 ELSE 0 END), 0) as warning,
			COALESCE(SUM(CASE WHEN severity = 'info' THEN 1 ELSE 0 END), 0) as info
		FROM alerts
		WHERE is_resolved = FALSE
	`

	var summary domain.AlertSummary
	err := r.db.GetContext(ctx, &summary, query)
	if err != nil {
		return nil, err
	}

	return &summary, nil
}

func (r *AlertRepository) Acknowledge(ctx context.Context, id int64, userID int64) error {
	query := `
		UPDATE alerts
		SET is_acknowledged = TRUE, acknowledged_at = NOW(), acknowledged_by = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, userID, id)
	return err
}

func (r *AlertRepository) Resolve(ctx context.Context, id int64, userID int64, note string) error {
	query := `
		UPDATE alerts
		SET is_resolved = TRUE, resolved_at = NOW(), resolved_by = ?, resolution_note = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, userID, note, id)
	return err
}

// BulkResolve resolves multiple alerts by ID in a single operation.
// It iterates over the given IDs and calls Resolve for each one.
func (r *AlertRepository) BulkResolve(ctx context.Context, ids []int64, userID int64, note string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// Build IN clause placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+3)
	args = append(args, userID, note)
	for i, id := range ids {
		placeholders[i] = "?"
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		UPDATE alerts
		SET is_resolved = TRUE, resolved_at = NOW(), resolved_by = ?, resolution_note = ?
		WHERE id IN (%s) AND is_resolved = FALSE
	`, strings.Join(placeholders, ", "))

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return int(affected), nil
}

// ResolveAllByType resolves all unresolved alerts of specific type for a website
func (r *AlertRepository) ResolveAllByType(ctx context.Context, websiteID int64, alertType domain.AlertType, note string) error {
	query := `
		UPDATE alerts
		SET is_resolved = TRUE, resolved_at = NOW(), resolved_by = 0, resolution_note = ?
		WHERE website_id = ? AND type = ? AND is_resolved = FALSE
	`
	_, err := r.db.ExecContext(ctx, query, note, websiteID, alertType)
	return err
}

// GetLatestAlertByType gets the latest unresolved alert of specific type for a website
func (r *AlertRepository) GetLatestAlertByType(ctx context.Context, websiteID int64, alertType domain.AlertType) (*domain.Alert, error) {
	var alert domain.Alert
	query := `
		SELECT * FROM alerts
		WHERE website_id = ? AND type = ? AND is_resolved = FALSE
		ORDER BY created_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &alert, query, websiteID, alertType)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &alert, nil
}

// HasRecentUnresolvedAlert checks if there is an existing unresolved alert for the given website and type.
// Checks ALL unresolved alerts, not just the most recent one.
func (r *AlertRepository) HasRecentUnresolvedAlert(ctx context.Context, websiteID int64, alertType domain.AlertType) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM alerts WHERE website_id = ? AND type = ? AND is_resolved = FALSE`
	err := r.db.GetContext(ctx, &count, query, websiteID, alertType)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUnresolvedAlertTypes returns all distinct alert types that are currently unresolved
// for a given website. Used for alert grouping - e.g., don't create SLOW_RESPONSE
// if there's already a DOWN alert.
func (r *AlertRepository) GetUnresolvedAlertTypes(ctx context.Context, websiteID int64) ([]string, error) {
	var types []string
	query := `SELECT DISTINCT type FROM alerts WHERE website_id = ? AND is_resolved = FALSE`
	err := r.db.SelectContext(ctx, &types, query, websiteID)
	if err != nil {
		return nil, err
	}
	return types, nil
}

// HasRecentResolvedAlert checks if there is a recently resolved alert within a cooldown period.
// This prevents alert flapping - after an alert is resolved, the same type of alert
// won't be created again within the cooldown window (in minutes).
func (r *AlertRepository) HasRecentResolvedAlert(ctx context.Context, websiteID int64, alertType domain.AlertType, cooldownMinutes int) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM alerts WHERE website_id = ? AND type = ? AND is_resolved = TRUE AND resolved_at >= DATE_SUB(NOW(), INTERVAL ? MINUTE)`
	err := r.db.GetContext(ctx, &count, query, websiteID, alertType, cooldownMinutes)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// CreateNotification saves a notification record
func (r *AlertRepository) CreateNotification(ctx context.Context, n *domain.Notification) (int64, error) {
	query := `
		INSERT INTO notifications (alert_id, channel, recipient, status)
		VALUES (?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(ctx, query, n.AlertID, n.Channel, n.Recipient, n.Status)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// UpdateNotificationStatus updates notification status
func (r *AlertRepository) UpdateNotificationStatus(ctx context.Context, id int64, status string, errorMsg string) error {
	query := `
		UPDATE notifications
		SET status = ?, sent_at = NOW(), error_message = ?, retry_count = retry_count + 1
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, errorMsg, id)
	return err
}

// CleanupOldAlerts removes old resolved alerts
func (r *AlertRepository) CleanupOldAlerts(ctx context.Context, days int) (int64, error) {
	query := `
		DELETE FROM alerts
		WHERE is_resolved = TRUE
		AND resolved_at < DATE_SUB(NOW(), INTERVAL ? DAY)
	`
	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
