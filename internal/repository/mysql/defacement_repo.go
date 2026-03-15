package mysql

import (
	"context"
	"database/sql"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/jmoiron/sqlx"
)

type DefacementRepository struct {
	db *sqlx.DB
}

func NewDefacementRepository(db *sqlx.DB) *DefacementRepository {
	return &DefacementRepository{db: db}
}

func (r *DefacementRepository) CreateIncident(ctx context.Context, incident *domain.DefacementIncident) error {
	query := `
		INSERT IGNORE INTO defacement_incidents (website_id, source, source_id, defaced_url, attacker, team, defaced_at, mirror_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		incident.WebsiteID, incident.Source, incident.SourceID,
		incident.DefacedURL, incident.Attacker, incident.Team,
		incident.DefacedAt, incident.MirrorURL,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	incident.ID = id
	return nil
}

func (r *DefacementRepository) GetIncidents(ctx context.Context, websiteID int64, source string, acknowledged *bool, limit, offset int) ([]domain.DefacementIncident, int64, error) {
	baseQuery := `FROM defacement_incidents di LEFT JOIN websites w ON di.website_id = w.id WHERE 1=1`
	args := []interface{}{}

	if websiteID > 0 {
		baseQuery += " AND di.website_id = ?"
		args = append(args, websiteID)
	}
	if source != "" {
		baseQuery += " AND di.source = ?"
		args = append(args, source)
	}
	if acknowledged != nil {
		baseQuery += " AND di.is_acknowledged = ?"
		args = append(args, *acknowledged)
	}

	var total int64
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) "+baseQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := `
		SELECT di.id, di.website_id, w.name as website_name, w.url as website_url,
			di.source, di.source_id, di.defaced_url, di.attacker, di.team,
			di.defaced_at, di.mirror_url, di.is_acknowledged, di.acknowledged_at,
			di.acknowledged_by, di.notes, di.created_at
	` + baseQuery + ` ORDER BY di.defaced_at DESC`

	if limit > 0 {
		selectQuery += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		selectQuery += " OFFSET ?"
		args = append(args, offset)
	}

	var rows []defacementRow
	if err := r.db.SelectContext(ctx, &rows, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	incidents := make([]domain.DefacementIncident, len(rows))
	for i, row := range rows {
		incidents[i] = row.toDomain()
	}
	return incidents, total, nil
}

func (r *DefacementRepository) AcknowledgeIncident(ctx context.Context, id int64, user string, notes string) error {
	query := `UPDATE defacement_incidents SET is_acknowledged = TRUE, acknowledged_at = NOW(), acknowledged_by = ?, notes = ? WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, user, notes, id)
	return err
}

func (r *DefacementRepository) GetStats(ctx context.Context) (*domain.DefacementStats, error) {
	stats := &domain.DefacementStats{
		BySource: make(map[domain.DefacementSource]int),
	}

	var row struct {
		Total          int `db:"total"`
		Unacknowledged int `db:"unacknowledged"`
		Affected       int `db:"affected"`
	}
	err := r.db.GetContext(ctx, &row, `
		SELECT
			(SELECT COUNT(*) FROM defacement_incidents) as total,
			(SELECT COUNT(*) FROM defacement_incidents WHERE is_acknowledged = FALSE) as unacknowledged,
			(SELECT COUNT(DISTINCT website_id) FROM defacement_incidents) as affected
	`)
	if err != nil {
		return nil, err
	}
	stats.TotalIncidents = row.Total
	stats.UnacknowledgedCount = row.Unacknowledged
	stats.WebsitesAffected = row.Affected

	var sourceRows []struct {
		Source string `db:"source"`
		Count  int    `db:"count"`
	}
	if err := r.db.SelectContext(ctx, &sourceRows, `SELECT source, COUNT(*) as count FROM defacement_incidents GROUP BY source`); err == nil {
		for _, s := range sourceRows {
			stats.BySource[domain.DefacementSource(s.Source)] = s.Count
		}
	}

	var lastScan sql.NullTime
	if err := r.db.GetContext(ctx, &lastScan, `SELECT MAX(started_at) FROM defacement_scans WHERE status = 'completed'`); err == nil && lastScan.Valid {
		stats.LastScanAt = &lastScan.Time
	}

	return stats, nil
}

func (r *DefacementRepository) CreateScan(ctx context.Context, scan *domain.DefacementScan) error {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO defacement_scans (source, status) VALUES (?, ?)`,
		scan.Source, scan.Status,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	scan.ID = id
	return nil
}

func (r *DefacementRepository) CompleteScan(ctx context.Context, id int64, totalChecked, newIncidents int, errMsg string) error {
	status := "completed"
	if errMsg != "" {
		status = "failed"
	}
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE defacement_scans SET status = ?, total_checked = ?, new_incidents = ?, completed_at = ?, error_message = ? WHERE id = ?`,
		status, totalChecked, newIncidents, now, errMsg, id,
	)
	return err
}

func (r *DefacementRepository) ExistsIncident(ctx context.Context, source, defacedURL string) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM defacement_incidents WHERE source = ? AND defaced_url = ?`, source, defacedURL)
	return count > 0, err
}

type defacementRow struct {
	ID             int64          `db:"id"`
	WebsiteID      int64          `db:"website_id"`
	WebsiteName    sql.NullString `db:"website_name"`
	WebsiteURL     sql.NullString `db:"website_url"`
	Source         string         `db:"source"`
	SourceID       sql.NullString `db:"source_id"`
	DefacedURL     string         `db:"defaced_url"`
	Attacker       sql.NullString `db:"attacker"`
	Team           sql.NullString `db:"team"`
	DefacedAt      sql.NullTime   `db:"defaced_at"`
	MirrorURL      sql.NullString `db:"mirror_url"`
	IsAcknowledged bool           `db:"is_acknowledged"`
	AcknowledgedAt sql.NullTime   `db:"acknowledged_at"`
	AcknowledgedBy sql.NullString `db:"acknowledged_by"`
	Notes          sql.NullString `db:"notes"`
	CreatedAt      time.Time      `db:"created_at"`
}

func (d *defacementRow) toDomain() domain.DefacementIncident {
	inc := domain.DefacementIncident{
		ID:             d.ID,
		WebsiteID:      d.WebsiteID,
		Source:         domain.DefacementSource(d.Source),
		DefacedURL:     d.DefacedURL,
		IsAcknowledged: d.IsAcknowledged,
		CreatedAt:      d.CreatedAt,
	}
	if d.WebsiteName.Valid {
		inc.WebsiteName = d.WebsiteName.String
	}
	if d.WebsiteURL.Valid {
		inc.WebsiteURL = d.WebsiteURL.String
	}
	if d.SourceID.Valid {
		inc.SourceID = d.SourceID.String
	}
	if d.Attacker.Valid {
		inc.Attacker = d.Attacker.String
	}
	if d.Team.Valid {
		inc.Team = d.Team.String
	}
	if d.DefacedAt.Valid {
		inc.DefacedAt = &d.DefacedAt.Time
	}
	if d.MirrorURL.Valid {
		inc.MirrorURL = d.MirrorURL.String
	}
	if d.AcknowledgedAt.Valid {
		inc.AcknowledgedAt = &d.AcknowledgedAt.Time
	}
	if d.AcknowledgedBy.Valid {
		inc.AcknowledgedBy = d.AcknowledgedBy.String
	}
	if d.Notes.Valid {
		inc.Notes = d.Notes.String
	}
	return inc
}
