package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

type AuditRepository struct {
	db *sqlx.DB
}

func NewAuditRepository(db *sqlx.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create inserts a new audit log entry
func (r *AuditRepository) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (user_id, username, action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query,
		log.UserID,
		log.Username,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		log.Details,
		log.IPAddress,
		log.UserAgent,
	)
	return err
}

// GetAll retrieves audit logs with filtering and pagination.
// Returns the logs, total count, and any error.
func (r *AuditRepository) GetAll(ctx context.Context, filter domain.AuditFilter) ([]domain.AuditLog, int, error) {
	var conditions []string
	var args []interface{}

	if filter.UserID != nil {
		conditions = append(conditions, "user_id = ?")
		args = append(args, *filter.UserID)
	}
	if filter.Action != nil && *filter.Action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, *filter.Action)
	}
	if filter.ResourceType != nil && *filter.ResourceType != "" {
		conditions = append(conditions, "resource_type = ?")
		args = append(args, *filter.ResourceType)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	// Defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	offset := (filter.Page - 1) * filter.Limit

	dataQuery := fmt.Sprintf(
		"SELECT * FROM audit_logs %s ORDER BY created_at DESC LIMIT ? OFFSET ?",
		whereClause,
	)
	args = append(args, filter.Limit, offset)

	var logs []domain.AuditLog
	if err := r.db.SelectContext(ctx, &logs, dataQuery, args...); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
