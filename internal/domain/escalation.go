package domain

import (
	"time"
)

// EscalationLevel represents the escalation level
type EscalationLevel int

const (
	EscalationLevel1 EscalationLevel = 1 // First responder
	EscalationLevel2 EscalationLevel = 2 // Team lead
	EscalationLevel3 EscalationLevel = 3 // Manager
	EscalationLevel4 EscalationLevel = 4 // Director/Emergency
)

// EscalationChannel represents notification channel for escalation
type EscalationChannel string

const (
	EscalationChannelTelegram EscalationChannel = "telegram"
	EscalationChannelEmail    EscalationChannel = "email"
	EscalationChannelWebhook  EscalationChannel = "webhook"
	EscalationChannelSMS      EscalationChannel = "sms"
)

// EscalationPolicy defines when and how to escalate alerts
type EscalationPolicy struct {
	ID          int64          `db:"id" json:"id"`
	Name        string         `db:"name" json:"name"`
	Description NullString `db:"description" json:"description,omitempty"`
	IsActive    bool           `db:"is_active" json:"is_active"`
	IsDefault   bool           `db:"is_default" json:"is_default"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
	Rules       []EscalationRule `json:"rules,omitempty"`
}

// EscalationRule defines a single escalation rule within a policy
type EscalationRule struct {
	ID              int64             `db:"id" json:"id"`
	PolicyID        int64             `db:"policy_id" json:"policy_id"`
	Level           EscalationLevel   `db:"level" json:"level"`
	Severity        AlertSeverity     `db:"severity" json:"severity"`        // Which severity triggers this rule
	DelayMinutes    int               `db:"delay_minutes" json:"delay_minutes"` // Minutes after alert creation
	NotifyChannels  []EscalationChannel `json:"notify_channels"`
	NotifyContacts  []EscalationContact `json:"notify_contacts"`
	RepeatInterval  int               `db:"repeat_interval" json:"repeat_interval"` // Minutes between repeated notifications (0 = no repeat)
	MaxRepeat       int               `db:"max_repeat" json:"max_repeat"`           // Max number of repeats (0 = unlimited)
	CreatedAt       time.Time         `db:"created_at" json:"created_at"`
}

// EscalationContact represents a contact to notify
type EscalationContact struct {
	ID        int64             `db:"id" json:"id"`
	RuleID    int64             `db:"rule_id" json:"rule_id"`
	Channel   EscalationChannel `db:"channel" json:"channel"`
	Value     string            `db:"value" json:"value"` // Email, phone, chat_id, webhook URL
	Name      NullString    `db:"name" json:"name"`   // Contact name
	IsActive  bool              `db:"is_active" json:"is_active"`
	CreatedAt time.Time         `db:"created_at" json:"created_at"`
}

// EscalationHistory tracks escalation events
type EscalationHistory struct {
	ID            int64             `db:"id" json:"id"`
	AlertID       int64             `db:"alert_id" json:"alert_id"`
	RuleID        int64             `db:"rule_id" json:"rule_id"`
	Level         EscalationLevel   `db:"level" json:"level"`
	Channel       EscalationChannel `db:"channel" json:"channel"`
	Recipient     string            `db:"recipient" json:"recipient"`
	Status        string            `db:"status" json:"status"` // sent, failed, pending
	ErrorMessage  NullString    `db:"error_message" json:"error_message,omitempty"`
	EscalatedAt   time.Time         `db:"escalated_at" json:"escalated_at"`
	AcknowledgedAt *time.Time       `db:"acknowledged_at" json:"acknowledged_at,omitempty"`
}

// EscalationPolicyCreate represents data for creating a new policy
type EscalationPolicyCreate struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	IsActive    bool   `json:"is_active"`
	IsDefault   bool   `json:"is_default"`
}

// EscalationRuleCreate represents data for creating a new rule
type EscalationRuleCreate struct {
	PolicyID       int64                `json:"policy_id" binding:"required"`
	Level          EscalationLevel      `json:"level" binding:"required"`
	Severity       AlertSeverity        `json:"severity" binding:"required"`
	DelayMinutes   int                  `json:"delay_minutes" binding:"required,min=0"`
	NotifyChannels []EscalationChannel  `json:"notify_channels" binding:"required"`
	NotifyContacts []EscalationContactCreate `json:"notify_contacts"`
	RepeatInterval int                  `json:"repeat_interval"`
	MaxRepeat      int                  `json:"max_repeat"`
}

// EscalationContactCreate represents data for creating a new contact
type EscalationContactCreate struct {
	Channel  EscalationChannel `json:"channel" binding:"required"`
	Value    string            `json:"value" binding:"required"`
	Name     string            `json:"name"`
	IsActive bool              `json:"is_active"`
}

// EscalationFilter for querying escalation history
type EscalationFilter struct {
	AlertID   *int64     `json:"alert_id"`
	PolicyID  *int64     `json:"policy_id"`
	Level     *int       `json:"level"`
	Status    *string    `json:"status"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}

// EscalationSummary provides overview of escalation status
type EscalationSummary struct {
	TotalPolicies     int `json:"total_policies"`
	ActivePolicies    int `json:"active_policies"`
	TotalRules        int `json:"total_rules"`
	EscalationsToday  int `json:"escalations_today"`
	PendingEscalations int `json:"pending_escalations"`
}

// AlertWithEscalation extends Alert with escalation information
type AlertWithEscalation struct {
	Alert
	EscalationLevel   EscalationLevel `json:"escalation_level"`
	LastEscalatedAt   *time.Time      `json:"last_escalated_at,omitempty"`
	EscalationCount   int             `json:"escalation_count"`
	NextEscalationAt  *time.Time      `json:"next_escalation_at,omitempty"`
}

// Default escalation configuration
var DefaultEscalationConfig = []struct {
	Level        EscalationLevel
	DelayMinutes int
	Description  string
}{
	{EscalationLevel1, 0, "Immediate notification to on-call team"},
	{EscalationLevel2, 15, "Escalate to team lead after 15 minutes"},
	{EscalationLevel3, 30, "Escalate to manager after 30 minutes"},
	{EscalationLevel4, 60, "Emergency escalation after 1 hour"},
}
