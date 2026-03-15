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

type EscalationRepository struct {
	db *sqlx.DB
}

func NewEscalationRepository(db *sqlx.DB) *EscalationRepository {
	return &EscalationRepository{db: db}
}

// --- Policy Methods ---

// CreatePolicy creates a new escalation policy
func (r *EscalationRepository) CreatePolicy(ctx context.Context, p *domain.EscalationPolicyCreate) (int64, error) {
	// If setting as default, unset other defaults first
	if p.IsDefault {
		_, err := r.db.ExecContext(ctx, "UPDATE escalation_policies SET is_default = FALSE WHERE is_default = TRUE")
		if err != nil {
			return 0, err
		}
	}

	query := `
		INSERT INTO escalation_policies (name, description, is_active, is_default)
		VALUES (?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.IsActive, p.IsDefault)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetPolicyByID retrieves a policy by ID with its rules
func (r *EscalationRepository) GetPolicyByID(ctx context.Context, id int64) (*domain.EscalationPolicy, error) {
	var policy domain.EscalationPolicy
	query := `SELECT * FROM escalation_policies WHERE id = ?`

	err := r.db.GetContext(ctx, &policy, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Get rules for this policy
	rules, err := r.GetRulesByPolicyID(ctx, id)
	if err != nil {
		return nil, err
	}
	policy.Rules = rules

	return &policy, nil
}

// GetDefaultPolicy retrieves the default escalation policy
func (r *EscalationRepository) GetDefaultPolicy(ctx context.Context) (*domain.EscalationPolicy, error) {
	var policy domain.EscalationPolicy
	query := `SELECT * FROM escalation_policies WHERE is_default = TRUE AND is_active = TRUE LIMIT 1`

	err := r.db.GetContext(ctx, &policy, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// Get rules for this policy
	rules, err := r.GetRulesByPolicyID(ctx, policy.ID)
	if err != nil {
		return nil, err
	}
	policy.Rules = rules

	return &policy, nil
}

// GetAllPolicies retrieves all escalation policies
func (r *EscalationRepository) GetAllPolicies(ctx context.Context) ([]domain.EscalationPolicy, error) {
	var policies []domain.EscalationPolicy
	query := `SELECT * FROM escalation_policies ORDER BY is_default DESC, name ASC`

	err := r.db.SelectContext(ctx, &policies, query)
	if err != nil {
		return nil, err
	}

	// Get rules for each policy
	for i := range policies {
		rules, err := r.GetRulesByPolicyID(ctx, policies[i].ID)
		if err != nil {
			return nil, err
		}
		policies[i].Rules = rules
	}

	return policies, nil
}

// UpdatePolicy updates an escalation policy
func (r *EscalationRepository) UpdatePolicy(ctx context.Context, id int64, p *domain.EscalationPolicyCreate) error {
	// If setting as default, unset other defaults first
	if p.IsDefault {
		_, err := r.db.ExecContext(ctx, "UPDATE escalation_policies SET is_default = FALSE WHERE is_default = TRUE AND id != ?", id)
		if err != nil {
			return err
		}
	}

	query := `
		UPDATE escalation_policies
		SET name = ?, description = ?, is_active = ?, is_default = ?, updated_at = NOW()
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, p.Name, p.Description, p.IsActive, p.IsDefault, id)
	return err
}

// DeletePolicy deletes an escalation policy
func (r *EscalationRepository) DeletePolicy(ctx context.Context, id int64) error {
	query := `DELETE FROM escalation_policies WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Rule Methods ---

// CreateRule creates a new escalation rule
func (r *EscalationRepository) CreateRule(ctx context.Context, rule *domain.EscalationRuleCreate) (int64, error) {
	channelsJSON, err := json.Marshal(rule.NotifyChannels)
	if err != nil {
		return 0, err
	}

	query := `
		INSERT INTO escalation_rules (policy_id, level, severity, delay_minutes, notify_channels, repeat_interval, max_repeat)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		rule.PolicyID, rule.Level, rule.Severity, rule.DelayMinutes,
		string(channelsJSON), rule.RepeatInterval, rule.MaxRepeat,
	)
	if err != nil {
		return 0, err
	}

	ruleID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Create contacts for this rule
	for _, contact := range rule.NotifyContacts {
		_, err = r.CreateContact(ctx, ruleID, &contact)
		if err != nil {
			return 0, err
		}
	}

	return ruleID, nil
}

// GetRulesByPolicyID retrieves all rules for a policy
func (r *EscalationRepository) GetRulesByPolicyID(ctx context.Context, policyID int64) ([]domain.EscalationRule, error) {
	var rules []domain.EscalationRule
	query := `SELECT * FROM escalation_rules WHERE policy_id = ? ORDER BY level, severity`

	rows, err := r.db.QueryxContext(ctx, query, policyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var rule domain.EscalationRule
		var channelsJSON string

		err := rows.Scan(
			&rule.ID, &rule.PolicyID, &rule.Level, &rule.Severity,
			&rule.DelayMinutes, &channelsJSON, &rule.RepeatInterval,
			&rule.MaxRepeat, &rule.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Parse channels JSON
		if channelsJSON != "" {
			var channels []domain.EscalationChannel
			if err := json.Unmarshal([]byte(channelsJSON), &channels); err == nil {
				rule.NotifyChannels = channels
			}
		}

		// Get contacts for this rule
		contacts, err := r.GetContactsByRuleID(ctx, rule.ID)
		if err != nil {
			return nil, err
		}
		rule.NotifyContacts = contacts

		rules = append(rules, rule)
	}

	return rules, nil
}

// GetRulesForEscalation gets applicable rules for an alert
func (r *EscalationRepository) GetRulesForEscalation(ctx context.Context, policyID int64, severity domain.AlertSeverity, currentLevel int, minutesSinceAlert int) ([]domain.EscalationRule, error) {
	query := `
		SELECT * FROM escalation_rules
		WHERE policy_id = ?
		AND severity = ?
		AND level > ?
		AND delay_minutes <= ?
		ORDER BY level ASC
	`

	rows, err := r.db.QueryxContext(ctx, query, policyID, severity, currentLevel, minutesSinceAlert)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []domain.EscalationRule
	for rows.Next() {
		var rule domain.EscalationRule
		var channelsJSON string

		err := rows.Scan(
			&rule.ID, &rule.PolicyID, &rule.Level, &rule.Severity,
			&rule.DelayMinutes, &channelsJSON, &rule.RepeatInterval,
			&rule.MaxRepeat, &rule.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if channelsJSON != "" {
			var channels []domain.EscalationChannel
			if err := json.Unmarshal([]byte(channelsJSON), &channels); err == nil {
				rule.NotifyChannels = channels
			}
		}

		contacts, _ := r.GetContactsByRuleID(ctx, rule.ID)
		rule.NotifyContacts = contacts

		rules = append(rules, rule)
	}

	return rules, nil
}

// DeleteRule deletes an escalation rule
func (r *EscalationRepository) DeleteRule(ctx context.Context, id int64) error {
	query := `DELETE FROM escalation_rules WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- Contact Methods ---

// CreateContact creates a new escalation contact
func (r *EscalationRepository) CreateContact(ctx context.Context, ruleID int64, c *domain.EscalationContactCreate) (int64, error) {
	query := `
		INSERT INTO escalation_contacts (rule_id, channel, value, name, is_active)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query, ruleID, c.Channel, c.Value, c.Name, c.IsActive)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// GetContactsByRuleID retrieves contacts for a rule
func (r *EscalationRepository) GetContactsByRuleID(ctx context.Context, ruleID int64) ([]domain.EscalationContact, error) {
	var contacts []domain.EscalationContact
	query := `SELECT * FROM escalation_contacts WHERE rule_id = ? AND is_active = TRUE`

	err := r.db.SelectContext(ctx, &contacts, query, ruleID)
	if err != nil {
		return nil, err
	}
	return contacts, nil
}

// DeleteContact deletes an escalation contact
func (r *EscalationRepository) DeleteContact(ctx context.Context, id int64) error {
	query := `DELETE FROM escalation_contacts WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// --- History Methods ---

// CreateHistory creates an escalation history record
func (r *EscalationRepository) CreateHistory(ctx context.Context, h *domain.EscalationHistory) (int64, error) {
	query := `
		INSERT INTO escalation_history (alert_id, rule_id, level, channel, recipient, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := r.db.ExecContext(ctx, query,
		h.AlertID, h.RuleID, h.Level, h.Channel, h.Recipient, h.Status, h.ErrorMessage,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateHistoryStatus updates the status of an escalation history record
func (r *EscalationRepository) UpdateHistoryStatus(ctx context.Context, id int64, status string, errorMsg string) error {
	query := `
		UPDATE escalation_history
		SET status = ?, error_message = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, status, errorMsg, id)
	return err
}

// GetHistory retrieves escalation history with filters
func (r *EscalationRepository) GetHistory(ctx context.Context, filter domain.EscalationFilter) ([]domain.EscalationHistory, int, error) {
	var history []domain.EscalationHistory
	var total int

	baseQuery := `FROM escalation_history h WHERE 1=1`
	var conditions []string
	var args []interface{}

	if filter.AlertID != nil {
		conditions = append(conditions, "h.alert_id = ?")
		args = append(args, *filter.AlertID)
	}
	if filter.Level != nil {
		conditions = append(conditions, "h.level = ?")
		args = append(args, *filter.Level)
	}
	if filter.Status != nil {
		conditions = append(conditions, "h.status = ?")
		args = append(args, *filter.Status)
	}
	if filter.StartDate != nil {
		conditions = append(conditions, "h.escalated_at >= ?")
		args = append(args, *filter.StartDate)
	}
	if filter.EndDate != nil {
		conditions = append(conditions, "h.escalated_at <= ?")
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
	dataQuery := fmt.Sprintf("SELECT h.* %s ORDER BY h.escalated_at DESC LIMIT ? OFFSET ?", baseQuery)
	args = append(args, limit, offset)

	err = r.db.SelectContext(ctx, &history, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return history, total, nil
}

// GetLastEscalation gets the last escalation for an alert
func (r *EscalationRepository) GetLastEscalation(ctx context.Context, alertID int64) (*domain.EscalationHistory, error) {
	var h domain.EscalationHistory
	query := `
		SELECT * FROM escalation_history
		WHERE alert_id = ?
		ORDER BY escalated_at DESC
		LIMIT 1
	`

	err := r.db.GetContext(ctx, &h, query, alertID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &h, nil
}

// GetEscalationCount gets the count of escalations for an alert at a specific level
func (r *EscalationRepository) GetEscalationCount(ctx context.Context, alertID int64, level int) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM escalation_history WHERE alert_id = ? AND level = ?`
	err := r.db.GetContext(ctx, &count, query, alertID, level)
	return count, err
}

// --- Alert Escalation Methods ---

// GetAlertsForEscalation retrieves alerts that need escalation check
func (r *EscalationRepository) GetAlertsForEscalation(ctx context.Context) ([]domain.AlertWithEscalation, error) {
	query := `
		SELECT a.*,
			COALESCE(a.escalation_level, 0) as escalation_level,
			a.last_escalated_at,
			COALESCE(a.escalation_count, 0) as escalation_count
		FROM alerts a
		WHERE a.is_resolved = FALSE
		AND a.severity IN ('critical', 'warning')
		ORDER BY
			CASE a.severity
				WHEN 'critical' THEN 1
				WHEN 'warning' THEN 2
				ELSE 3
			END,
			a.created_at ASC
	`

	var alerts []domain.AlertWithEscalation
	err := r.db.SelectContext(ctx, &alerts, query)
	if err != nil {
		return nil, err
	}
	return alerts, nil
}

// UpdateAlertEscalation updates the escalation tracking fields on an alert
func (r *EscalationRepository) UpdateAlertEscalation(ctx context.Context, alertID int64, level int, count int) error {
	query := `
		UPDATE alerts
		SET escalation_level = ?, last_escalated_at = NOW(), escalation_count = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, query, level, count, alertID)
	return err
}

// GetEscalationSummary gets summary statistics for escalations
func (r *EscalationRepository) GetEscalationSummary(ctx context.Context) (*domain.EscalationSummary, error) {
	summary := &domain.EscalationSummary{}

	// Total and active policies
	err := r.db.GetContext(ctx, &summary.TotalPolicies,
		`SELECT COUNT(*) FROM escalation_policies`)
	if err != nil {
		return nil, err
	}

	err = r.db.GetContext(ctx, &summary.ActivePolicies,
		`SELECT COUNT(*) FROM escalation_policies WHERE is_active = TRUE`)
	if err != nil {
		return nil, err
	}

	// Total rules
	err = r.db.GetContext(ctx, &summary.TotalRules,
		`SELECT COUNT(*) FROM escalation_rules`)
	if err != nil {
		return nil, err
	}

	// Escalations today
	err = r.db.GetContext(ctx, &summary.EscalationsToday,
		`SELECT COUNT(*) FROM escalation_history WHERE DATE(escalated_at) = CURDATE()`)
	if err != nil {
		return nil, err
	}

	// Pending escalations (alerts needing escalation)
	err = r.db.GetContext(ctx, &summary.PendingEscalations,
		`SELECT COUNT(*) FROM alerts WHERE is_resolved = FALSE AND severity IN ('critical', 'warning')`)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

// CleanupOldHistory removes old escalation history records
func (r *EscalationRepository) CleanupOldHistory(ctx context.Context, days int) (int64, error) {
	query := `
		DELETE FROM escalation_history
		WHERE escalated_at < DATE_SUB(NOW(), INTERVAL ? DAY)
	`
	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
