package alert

import (
	"context"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type Service struct {
	cfg              *config.Config
	alertRepo        *mysql.AlertRepository
	websiteRepo      *mysql.WebsiteRepository
	telegramNotifier *notifier.TelegramNotifier
	emailNotifier    *notifier.EmailNotifier
	webhookNotifier  *notifier.WebhookNotifier
	digestService    *notifier.DigestService
}

func NewService(
	cfg *config.Config,
	alertRepo *mysql.AlertRepository,
	websiteRepo *mysql.WebsiteRepository,
	telegramNotifier *notifier.TelegramNotifier,
	emailNotifier *notifier.EmailNotifier,
	webhookNotifier *notifier.WebhookNotifier,
	digestService *notifier.DigestService,
) *Service {
	return &Service{
		cfg:              cfg,
		alertRepo:        alertRepo,
		websiteRepo:      websiteRepo,
		telegramNotifier: telegramNotifier,
		emailNotifier:    emailNotifier,
		webhookNotifier:  webhookNotifier,
		digestService:    digestService,
	}
}

// CreateAlert creates a new alert and sends notification
func (s *Service) CreateAlert(ctx context.Context, alertCreate *domain.AlertCreate) (*domain.Alert, error) {
	// Create alert in database
	id, err := s.alertRepo.Create(ctx, alertCreate)
	if err != nil {
		return nil, err
	}

	// Get the created alert
	alert, err := s.alertRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Get website info for notification
	website, _ := s.websiteRepo.GetByID(ctx, alertCreate.WebsiteID)

	// Route notification through digest service if enabled, otherwise send immediately
	if s.digestService != nil && s.cfg.Notification.DigestEnabled {
		s.digestService.Add(alert, website)
	} else {
		// Send notifications to all channels immediately
		go func() {
			ctx := context.Background()

			// Telegram
			if err := s.telegramNotifier.SendAlert(ctx, alert, website); err != nil {
				logger.Error().Err(err).Int64("alert_id", id).Msg("Failed to send Telegram notification")
			}

			// Email
			if s.emailNotifier != nil {
				if err := s.emailNotifier.SendAlert(ctx, alert, website); err != nil {
					logger.Error().Err(err).Int64("alert_id", id).Msg("Failed to send Email notification")
				}
			}

			// Webhook
			if s.webhookNotifier != nil {
				if err := s.webhookNotifier.SendAlert(ctx, alert, website); err != nil {
					logger.Error().Err(err).Int64("alert_id", id).Msg("Failed to send Webhook notification")
				}
			}
		}()
	}

	return alert, nil
}

// AcknowledgeAlert marks an alert as acknowledged
func (s *Service) AcknowledgeAlert(ctx context.Context, alertID int64, userID int64) error {
	return s.alertRepo.Acknowledge(ctx, alertID, userID)
}

// ResolveAlert marks an alert as resolved and sends notification
func (s *Service) ResolveAlert(ctx context.Context, alertID int64, userID int64, note string) error {
	// Get alert before resolving
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return err
	}
	if alert == nil {
		return nil
	}

	// Resolve alert
	if err := s.alertRepo.Resolve(ctx, alertID, userID, note); err != nil {
		return err
	}

	// Get website info
	website, _ := s.websiteRepo.GetByID(ctx, alert.WebsiteID)

	// Send resolution notifications to all channels
	go func() {
		ctx := context.Background()

		// Telegram
		if err := s.telegramNotifier.SendAlertResolved(ctx, alert, website, note); err != nil {
			logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to send Telegram resolution notification")
		}

		// Email
		if s.emailNotifier != nil {
			if err := s.emailNotifier.SendAlertResolved(ctx, alert, website, note); err != nil {
				logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to send Email resolution notification")
			}
		}

		// Webhook
		if s.webhookNotifier != nil {
			if err := s.webhookNotifier.SendAlertResolved(ctx, alert, website, note); err != nil {
				logger.Error().Err(err).Int64("alert_id", alertID).Msg("Failed to send Webhook resolution notification")
			}
		}
	}()

	return nil
}

// GetActiveAlerts returns all unresolved alerts
func (s *Service) GetActiveAlerts(ctx context.Context) ([]domain.Alert, error) {
	return s.alertRepo.GetActiveAlerts(ctx)
}

// GetAlertSummary returns alert statistics
func (s *Service) GetAlertSummary(ctx context.Context) (*domain.AlertSummary, error) {
	return s.alertRepo.GetSummary(ctx)
}

// GetAlerts returns alerts with filtering
func (s *Service) GetAlerts(ctx context.Context, filter domain.AlertFilter) ([]domain.Alert, int, error) {
	return s.alertRepo.GetAll(ctx, filter)
}

// GetAlertByID returns a single alert
func (s *Service) GetAlertByID(ctx context.Context, id int64) (*domain.Alert, error) {
	return s.alertRepo.GetByID(ctx, id)
}
