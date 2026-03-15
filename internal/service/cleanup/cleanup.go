package cleanup

import (
	"context"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type CleanupService struct {
	cfg         *config.Config
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
}

func NewCleanupService(
	cfg *config.Config,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *CleanupService {
	return &CleanupService{
		cfg:       cfg,
		checkRepo: checkRepo,
		alertRepo: alertRepo,
	}
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	ChecksDeleted        int64     `json:"checks_deleted"`
	SSLChecksDeleted     int64     `json:"ssl_checks_deleted"`
	ContentScansDeleted  int64     `json:"content_scans_deleted"`
	SecurityChecksDeleted int64    `json:"security_checks_deleted"`
	AlertsDeleted        int64     `json:"alerts_deleted"`
	CleanedAt            time.Time `json:"cleaned_at"`
}

// RunCleanup performs database cleanup based on retention settings
func (s *CleanupService) RunCleanup(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{
		CleanedAt: time.Now(),
	}

	// Get retention days from config, default to 90 days
	retentionDays := 90
	if s.cfg.Database.RetentionDays > 0 {
		retentionDays = s.cfg.Database.RetentionDays
	}

	logger.Info().Int("retention_days", retentionDays).Msg("Starting database cleanup")

	// Clean up old checks
	checksDeleted, err := s.checkRepo.CleanupOldData(ctx, "checks", retentionDays)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup checks")
	} else {
		result.ChecksDeleted = checksDeleted
		logger.Info().Int64("deleted", checksDeleted).Msg("Cleaned up old checks")
	}

	// Clean up old SSL checks
	sslChecksDeleted, err := s.checkRepo.CleanupOldData(ctx, "ssl_checks", retentionDays)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup SSL checks")
	} else {
		result.SSLChecksDeleted = sslChecksDeleted
		logger.Info().Int64("deleted", sslChecksDeleted).Msg("Cleaned up old SSL checks")
	}

	// Clean up old content scans
	contentScansDeleted, err := s.checkRepo.CleanupOldData(ctx, "content_scans", retentionDays)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup content scans")
	} else {
		result.ContentScansDeleted = contentScansDeleted
		logger.Info().Int64("deleted", contentScansDeleted).Msg("Cleaned up old content scans")
	}

	// Clean up old security header checks
	securityChecksDeleted, err := s.checkRepo.CleanupOldData(ctx, "security_header_checks", retentionDays)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup security header checks")
	} else {
		result.SecurityChecksDeleted = securityChecksDeleted
		logger.Info().Int64("deleted", securityChecksDeleted).Msg("Cleaned up old security header checks")
	}

	// Clean up old resolved alerts (keep for 30 days only)
	alertRetentionDays := 30
	alertsDeleted, err := s.alertRepo.CleanupOldAlerts(ctx, alertRetentionDays)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to cleanup alerts")
	} else {
		result.AlertsDeleted = alertsDeleted
		logger.Info().Int64("deleted", alertsDeleted).Msg("Cleaned up old alerts")
	}

	logger.Info().
		Int64("checks_deleted", result.ChecksDeleted).
		Int64("ssl_checks_deleted", result.SSLChecksDeleted).
		Int64("content_scans_deleted", result.ContentScansDeleted).
		Int64("security_checks_deleted", result.SecurityChecksDeleted).
		Int64("alerts_deleted", result.AlertsDeleted).
		Msg("Database cleanup completed")

	return result, nil
}

// GetDatabaseStats returns current database statistics
func (s *CleanupService) GetDatabaseStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get table row counts
	tables := []string{"checks", "ssl_checks", "content_scans", "security_header_checks", "alerts", "websites"}
	for _, table := range tables {
		count, err := s.checkRepo.GetTableRowCount(ctx, table)
		if err != nil {
			logger.Warn().Err(err).Str("table", table).Msg("Failed to get row count")
			continue
		}
		stats[table+"_count"] = count
	}

	// Get oldest data dates
	oldestCheck, err := s.checkRepo.GetOldestRecordDate(ctx, "checks", "checked_at")
	if err == nil && oldestCheck != nil {
		stats["oldest_check"] = oldestCheck
	}

	oldestAlert, err := s.checkRepo.GetOldestRecordDate(ctx, "alerts", "created_at")
	if err == nil && oldestAlert != nil {
		stats["oldest_alert"] = oldestAlert
	}

	return stats, nil
}
