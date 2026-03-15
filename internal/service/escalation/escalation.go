package escalation

import (
	"context"
	"fmt"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type EscalationService struct {
	cfg              *config.Config
	escalationRepo   *mysql.EscalationRepository
	alertRepo        *mysql.AlertRepository
	websiteRepo      *mysql.WebsiteRepository
	telegramNotifier *notifier.TelegramNotifier
	emailNotifier    *notifier.EmailNotifier
	webhookNotifier  *notifier.WebhookNotifier
}

func NewEscalationService(
	cfg *config.Config,
	escalationRepo *mysql.EscalationRepository,
	alertRepo *mysql.AlertRepository,
	websiteRepo *mysql.WebsiteRepository,
	telegramNotifier *notifier.TelegramNotifier,
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
) *EscalationService {
	return &EscalationService{
		cfg:              cfg,
		escalationRepo:   escalationRepo,
		alertRepo:        alertRepo,
		websiteRepo:      websiteRepo,
		telegramNotifier: telegramNotifier,
		emailNotifier:    emailNotifier,
		webhookNotifier:  webhookNotifier,
	}
}

// ProcessEscalations checks all active alerts and escalates as needed
func (s *EscalationService) ProcessEscalations(ctx context.Context) error {
	logger.Info().Msg("Starting escalation processing")

	// Get default policy
	policy, err := s.escalationRepo.GetDefaultPolicy(ctx)
	if err != nil {
		return fmt.Errorf("failed to get default policy: %w", err)
	}
	if policy == nil {
		logger.Info().Msg("No active escalation policy found")
		return nil
	}

	// Get alerts that need escalation check
	alerts, err := s.escalationRepo.GetAlertsForEscalation(ctx)
	if err != nil {
		return fmt.Errorf("failed to get alerts for escalation: %w", err)
	}

	logger.Info().Int("alert_count", len(alerts)).Msg("Processing alerts for escalation")

	for _, alert := range alerts {
		if err := s.processAlertEscalation(ctx, &alert, policy); err != nil {
			logger.Error().
				Err(err).
				Int64("alert_id", alert.ID).
				Msg("Failed to process alert escalation")
		}
	}

	return nil
}

func (s *EscalationService) processAlertEscalation(ctx context.Context, alert *domain.AlertWithEscalation, policy *domain.EscalationPolicy) error {
	// Calculate time since alert was created
	minutesSinceAlert := int(time.Since(alert.CreatedAt).Minutes())

	// Get applicable rules for this alert
	rules, err := s.escalationRepo.GetRulesForEscalation(
		ctx,
		policy.ID,
		alert.Severity,
		int(alert.EscalationLevel),
		minutesSinceAlert,
	)
	if err != nil {
		return err
	}

	if len(rules) == 0 {
		// Check for repeat notifications at current level
		return s.checkRepeatNotification(ctx, alert, policy)
	}

	// Process the next applicable rule (escalate)
	rule := rules[0] // Get the first applicable rule
	return s.escalateAlert(ctx, alert, &rule)
}

func (s *EscalationService) checkRepeatNotification(ctx context.Context, alert *domain.AlertWithEscalation, policy *domain.EscalationPolicy) error {
	if alert.EscalationLevel == 0 {
		return nil
	}

	// Find the current level rule
	var currentRule *domain.EscalationRule
	for _, r := range policy.Rules {
		if int(r.Level) == int(alert.EscalationLevel) && r.Severity == alert.Severity {
			currentRule = &r
			break
		}
	}

	if currentRule == nil || currentRule.RepeatInterval == 0 {
		return nil
	}

	// Check if it's time for repeat notification
	if alert.LastEscalatedAt == nil {
		return nil
	}

	minutesSinceLastEscalation := int(time.Since(*alert.LastEscalatedAt).Minutes())
	if minutesSinceLastEscalation < currentRule.RepeatInterval {
		return nil
	}

	// Check if max repeat reached
	if currentRule.MaxRepeat > 0 {
		escalationCount, err := s.escalationRepo.GetEscalationCount(ctx, alert.ID, int(currentRule.Level))
		if err != nil {
			return err
		}
		if escalationCount >= currentRule.MaxRepeat {
			return nil
		}
	}

	// Send repeat notification
	return s.sendEscalationNotifications(ctx, alert, currentRule, true)
}

func (s *EscalationService) escalateAlert(ctx context.Context, alert *domain.AlertWithEscalation, rule *domain.EscalationRule) error {
	logger.Info().
		Int64("alert_id", alert.ID).
		Int("from_level", int(alert.EscalationLevel)).
		Int("to_level", int(rule.Level)).
		Msg("Escalating alert")

	// Send notifications
	if err := s.sendEscalationNotifications(ctx, alert, rule, false); err != nil {
		return err
	}

	// Update alert escalation status
	newCount := alert.EscalationCount + 1
	if err := s.escalationRepo.UpdateAlertEscalation(ctx, alert.ID, int(rule.Level), newCount); err != nil {
		return err
	}

	return nil
}

func (s *EscalationService) sendEscalationNotifications(ctx context.Context, alert *domain.AlertWithEscalation, rule *domain.EscalationRule, isRepeat bool) error {
	// Get website info for the message
	website, _ := s.websiteRepo.GetByID(ctx, alert.WebsiteID)
	websiteName := "Unknown"
	websiteURL := ""
	if website != nil {
		websiteName = website.Name
		websiteURL = website.URL
	}

	// Build escalation message
	actionType := "ESCALATED"
	if isRepeat {
		actionType = "REMINDER"
	}

	message := fmt.Sprintf(
		"🚨 ALERT %s - Level %d\n\n"+
			"Website: %s\n"+
			"URL: %s\n"+
			"Severity: %s\n"+
			"Type: %s\n"+
			"Title: %s\n"+
			"Message: %s\n\n"+
			"Alert Created: %s\n"+
			"Escalation Level: %d\n"+
			"Escalation Count: %d\n\n"+
			"Please acknowledge or resolve this alert.",
		actionType, rule.Level,
		websiteName,
		websiteURL,
		alert.Severity,
		alert.Type,
		alert.Title,
		alert.Message,
		alert.CreatedAt.Format("2006-01-02 15:04:05"),
		rule.Level,
		alert.EscalationCount+1,
	)

	// Send to configured channels
	for _, channel := range rule.NotifyChannels {
		var err error
		var recipient string

		switch channel {
		case domain.EscalationChannelTelegram:
			if s.telegramNotifier != nil {
				for _, chatID := range s.cfg.Telegram.ChatIDs {
					recipient = chatID
					err = s.telegramNotifier.SendRawMessage(ctx, chatID, message)
					s.recordEscalationHistory(ctx, alert.ID, rule.ID, rule.Level, channel, recipient, err)
				}
			}

		case domain.EscalationChannelEmail:
			if s.emailNotifier != nil {
				for _, email := range s.cfg.Email.Recipients {
					recipient = email
					subject := fmt.Sprintf("[%s] Alert Escalation - %s", alert.Severity, websiteName)
					err = s.emailNotifier.SendRawEmail(ctx, email, subject, message)
					s.recordEscalationHistory(ctx, alert.ID, rule.ID, rule.Level, channel, recipient, err)
				}
			}

		case domain.EscalationChannelWebhook:
			if s.webhookNotifier != nil {
				for _, url := range s.cfg.Webhook.URLs {
					recipient = url
					err = s.webhookNotifier.SendEscalation(ctx, url, alert, rule)
					s.recordEscalationHistory(ctx, alert.ID, rule.ID, rule.Level, channel, recipient, err)
				}
			}
		}

		// Also send to specific contacts for this rule
		for _, contact := range rule.NotifyContacts {
			if contact.Channel != channel || !contact.IsActive {
				continue
			}

			recipient = contact.Value
			switch channel {
			case domain.EscalationChannelTelegram:
				if s.telegramNotifier != nil {
					err = s.telegramNotifier.SendRawMessage(ctx, contact.Value, message)
				}
			case domain.EscalationChannelEmail:
				if s.emailNotifier != nil {
					subject := fmt.Sprintf("[%s] Alert Escalation - %s", alert.Severity, websiteName)
					err = s.emailNotifier.SendRawEmail(ctx, contact.Value, subject, message)
				}
			}
			s.recordEscalationHistory(ctx, alert.ID, rule.ID, rule.Level, channel, recipient, err)
		}
	}

	return nil
}

func (s *EscalationService) recordEscalationHistory(ctx context.Context, alertID int64, ruleID int64, level domain.EscalationLevel, channel domain.EscalationChannel, recipient string, err error) {
	status := "sent"
	var errorMsg domain.NullString

	if err != nil {
		status = "failed"
		errorMsg = domain.NewNullString(err.Error())
	}

	history := &domain.EscalationHistory{
		AlertID:      alertID,
		RuleID:       ruleID,
		Level:        level,
		Channel:      channel,
		Recipient:    recipient,
		Status:       status,
		ErrorMessage: errorMsg,
	}

	if _, err := s.escalationRepo.CreateHistory(ctx, history); err != nil {
		logger.Error().Err(err).Msg("Failed to record escalation history")
	}
}

// GetPolicies returns all escalation policies
func (s *EscalationService) GetPolicies(ctx context.Context) ([]domain.EscalationPolicy, error) {
	return s.escalationRepo.GetAllPolicies(ctx)
}

// GetPolicy returns a specific policy by ID
func (s *EscalationService) GetPolicy(ctx context.Context, id int64) (*domain.EscalationPolicy, error) {
	return s.escalationRepo.GetPolicyByID(ctx, id)
}

// CreatePolicy creates a new escalation policy
func (s *EscalationService) CreatePolicy(ctx context.Context, p *domain.EscalationPolicyCreate) (int64, error) {
	return s.escalationRepo.CreatePolicy(ctx, p)
}

// UpdatePolicy updates an existing policy
func (s *EscalationService) UpdatePolicy(ctx context.Context, id int64, p *domain.EscalationPolicyCreate) error {
	return s.escalationRepo.UpdatePolicy(ctx, id, p)
}

// DeletePolicy deletes a policy
func (s *EscalationService) DeletePolicy(ctx context.Context, id int64) error {
	return s.escalationRepo.DeletePolicy(ctx, id)
}

// CreateRule creates a new escalation rule
func (s *EscalationService) CreateRule(ctx context.Context, rule *domain.EscalationRuleCreate) (int64, error) {
	return s.escalationRepo.CreateRule(ctx, rule)
}

// DeleteRule deletes an escalation rule
func (s *EscalationService) DeleteRule(ctx context.Context, id int64) error {
	return s.escalationRepo.DeleteRule(ctx, id)
}

// GetHistory returns escalation history
func (s *EscalationService) GetHistory(ctx context.Context, filter domain.EscalationFilter) ([]domain.EscalationHistory, int, error) {
	return s.escalationRepo.GetHistory(ctx, filter)
}

// GetSummary returns escalation summary
func (s *EscalationService) GetSummary(ctx context.Context) (*domain.EscalationSummary, error) {
	return s.escalationRepo.GetEscalationSummary(ctx)
}
