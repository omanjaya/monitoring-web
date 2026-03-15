package summary

import (
	"context"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/notifier"
)

type Service struct {
	websiteRepo *mysql.WebsiteRepository
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
}

func NewService(
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
) *Service {
	return &Service{
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
	}
}

// GenerateDailySummary creates a summary of the past 24 hours
func (s *Service) GenerateDailySummary(ctx context.Context) (*notifier.DailySummary, error) {
	summary := &notifier.DailySummary{}

	// Get website stats
	stats, err := s.websiteRepo.GetDashboardStats(ctx)
	if err != nil {
		return nil, err
	}

	summary.TotalWebsites = stats.TotalWebsites
	summary.WebsitesUp = stats.TotalUp
	summary.WebsitesDown = stats.TotalDown
	summary.WebsitesDegraded = stats.TotalDegraded
	summary.AvgResponseTime = int(stats.AvgResponseTime)

	// Calculate uptime percentage
	if stats.TotalWebsites > 0 {
		summary.UptimePercentage = float64(stats.TotalUp) / float64(stats.TotalWebsites) * 100
	}

	// Get alert summary
	alertSummary, err := s.alertRepo.GetSummary(ctx)
	if err != nil {
		return nil, err
	}

	summary.CriticalAlerts = alertSummary.Critical
	summary.WarningAlerts = alertSummary.Warning
	summary.InfoAlerts = alertSummary.Info

	// Count judol detected (websites with content_clean = false)
	filter := domain.WebsiteFilter{}
	websites, _, _ := s.websiteRepo.GetAll(ctx, filter)
	for _, w := range websites {
		if !w.ContentClean {
			summary.JudolDetected++
		}
	}

	return summary, nil
}
