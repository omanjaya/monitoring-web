package website

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type Service struct {
	cfg         *config.Config
	websiteRepo *mysql.WebsiteRepository
	checkRepo   *mysql.CheckRepository
	alertRepo   *mysql.AlertRepository
	opdRepo     *mysql.OPDRepository
}

func NewService(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
	opdRepo *mysql.OPDRepository,
) *Service {
	return &Service{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
		opdRepo:     opdRepo,
	}
}

// CreateWebsite creates a new website for monitoring
func (s *Service) CreateWebsite(ctx context.Context, input *domain.WebsiteCreate) (*domain.Website, error) {
	// Validate URL
	if err := s.validateURL(input.URL); err != nil {
		return nil, err
	}

	// Check if URL already exists
	existing, err := s.websiteRepo.GetByURL(ctx, input.URL)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("website dengan URL %s sudah terdaftar", input.URL)
	}

	// Create website
	id, err := s.websiteRepo.Create(ctx, input)
	if err != nil {
		return nil, err
	}

	// Get created website
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logger.Info().Str("url", input.URL).Int64("id", id).Msg("Website created")
	return website, nil
}

// GetWebsite retrieves a website by ID
func (s *Service) GetWebsite(ctx context.Context, id int64) (*domain.Website, error) {
	return s.websiteRepo.GetByID(ctx, id)
}

// GetWebsiteByURL retrieves a website by URL
func (s *Service) GetWebsiteByURL(ctx context.Context, websiteURL string) (*domain.Website, error) {
	return s.websiteRepo.GetByURL(ctx, websiteURL)
}

// ListWebsites retrieves websites with filtering
func (s *Service) ListWebsites(ctx context.Context, filter domain.WebsiteFilter) ([]domain.Website, int, error) {
	return s.websiteRepo.GetAll(ctx, filter)
}

// UpdateWebsite updates a website
func (s *Service) UpdateWebsite(ctx context.Context, id int64, input *domain.WebsiteUpdate) (*domain.Website, error) {
	// Check if website exists
	existing, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("website dengan ID %d tidak ditemukan", id)
	}

	// Update website
	if err := s.websiteRepo.Update(ctx, id, input); err != nil {
		return nil, err
	}

	// Get updated website
	website, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	logger.Info().Int64("id", id).Msg("Website updated")
	return website, nil
}

// DeleteWebsite deletes a website
func (s *Service) DeleteWebsite(ctx context.Context, id int64) error {
	// Check if website exists
	existing, err := s.websiteRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("website dengan ID %d tidak ditemukan", id)
	}

	// Delete website
	if err := s.websiteRepo.Delete(ctx, id); err != nil {
		return err
	}

	logger.Info().Int64("id", id).Str("url", existing.URL).Msg("Website deleted")
	return nil
}

// BulkAction performs a bulk action (enable, disable, delete) on multiple websites
func (s *Service) BulkAction(ctx context.Context, action *domain.BulkWebsiteAction) (int64, error) {
	var affected int64
	var err error

	switch action.Action {
	case "enable":
		affected, err = s.websiteRepo.BulkUpdateActive(ctx, action.IDs, true)
	case "disable":
		affected, err = s.websiteRepo.BulkUpdateActive(ctx, action.IDs, false)
	case "delete":
		affected, err = s.websiteRepo.BulkDelete(ctx, action.IDs)
	default:
		return 0, fmt.Errorf("action tidak valid: %s", action.Action)
	}

	if err != nil {
		return 0, err
	}

	logger.Info().
		Str("action", action.Action).
		Int("count", len(action.IDs)).
		Int64("affected", affected).
		Msg("Bulk action completed")

	return affected, nil
}

// GetDashboardStats retrieves dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*domain.DashboardStats, error) {
	return s.websiteRepo.GetDashboardStats(ctx)
}

// GetRecentDownWebsites retrieves recently down websites
func (s *Service) GetRecentDownWebsites(ctx context.Context, limit int) ([]domain.Website, error) {
	status := domain.StatusDown
	filter := domain.WebsiteFilter{
		Status: &status,
		Limit:  limit,
	}
	websites, _, err := s.websiteRepo.GetAll(ctx, filter)
	return websites, err
}

// GetUptimeStats retrieves uptime statistics for a website
func (s *Service) GetUptimeStats(ctx context.Context, websiteID int64, hours int) (*domain.UptimeStats, error) {
	since := time.Now().Add(-time.Duration(hours) * time.Hour)
	stats, err := s.checkRepo.GetUptimeStats(ctx, websiteID, since)
	if err != nil {
		return nil, err
	}
	stats.Period = fmt.Sprintf("%dh", hours)
	return stats, nil
}

// GetRecentChecks retrieves recent checks for a website
func (s *Service) GetRecentChecks(ctx context.Context, websiteID int64, limit int) ([]domain.Check, error) {
	return s.checkRepo.GetRecentChecks(ctx, websiteID, limit)
}

// GetLatestSSLCheck retrieves the latest SSL check for a website
func (s *Service) GetLatestSSLCheck(ctx context.Context, websiteID int64) (*domain.SSLCheck, error) {
	return s.checkRepo.GetLatestSSLCheck(ctx, websiteID)
}

// GetLatestContentScan retrieves the latest content scan for a website
func (s *Service) GetLatestContentScan(ctx context.Context, websiteID int64) (*domain.ContentScan, error) {
	return s.checkRepo.GetLatestContentScan(ctx, websiteID)
}

// GetUptimeChartData retrieves chart data for uptime visualization
func (s *Service) GetUptimeChartData(ctx context.Context, websiteID int64, hours int) (*domain.UptimeChartData, error) {
	// Use hourly data for periods up to 72 hours, daily data for longer periods
	if hours <= 72 {
		return s.checkRepo.GetHourlyChartData(ctx, websiteID, hours)
	}
	// Convert hours to days for daily aggregation
	days := hours / 24
	if days < 1 {
		days = 1
	}
	return s.checkRepo.GetDailyChartData(ctx, websiteID, days)
}

// GetWebsiteMetrics retrieves response time percentile metrics for a website across multiple time periods
func (s *Service) GetWebsiteMetrics(ctx context.Context, websiteID int64) (*domain.WebsiteMetrics, error) {
	website, err := s.websiteRepo.GetByID(ctx, websiteID)
	if err != nil {
		return nil, err
	}
	if website == nil {
		return nil, fmt.Errorf("website dengan ID %d tidak ditemukan", websiteID)
	}

	metrics := &domain.WebsiteMetrics{
		WebsiteID:   websiteID,
		WebsiteName: website.Name,
		Periods:     make(map[string]*domain.ResponseTimePercentiles),
	}

	// Calculate percentiles for 24h, 7d (168h), 30d (720h)
	periods := map[string]int{
		"24h": 24,
		"7d":  168,
		"30d": 720,
	}

	for label, hours := range periods {
		percentiles, err := s.checkRepo.GetResponseTimePercentiles(ctx, websiteID, hours)
		if err != nil {
			logger.Error().Err(err).Str("period", label).Int64("website_id", websiteID).Msg("Failed to get percentiles")
			continue
		}
		metrics.Periods[label] = percentiles
	}

	return metrics, nil
}

// Helper functions

func (s *Service) validateURL(rawURL string) error {
	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("URL tidak valid: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL harus menggunakan http atau https")
	}

	// Check host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL harus memiliki host")
	}

	// Check if it's a baliprov.go.id domain (optional validation)
	if s.cfg.Monitoring.RestrictDomain {
		if !strings.HasSuffix(parsedURL.Host, ".baliprov.go.id") && parsedURL.Host != "baliprov.go.id" {
			return fmt.Errorf("hanya domain baliprov.go.id yang diperbolehkan")
		}
	}

	return nil
}

// OPD Operations

// ListOPDs retrieves all OPDs
func (s *Service) ListOPDs(ctx context.Context) ([]domain.OPD, error) {
	return s.opdRepo.GetAll(ctx)
}

// GetOPD retrieves an OPD by ID
func (s *Service) GetOPD(ctx context.Context, id int64) (*domain.OPD, error) {
	return s.opdRepo.GetByID(ctx, id)
}

// CreateOPD creates a new OPD
func (s *Service) CreateOPD(ctx context.Context, opd *domain.OPD) (*domain.OPD, error) {
	id, err := s.opdRepo.Create(ctx, opd)
	if err != nil {
		return nil, err
	}

	return s.opdRepo.GetByID(ctx, id)
}

// BulkImportWebsites imports multiple websites from a list
func (s *Service) BulkImportWebsites(ctx context.Context, websites []domain.WebsiteCreate) (*BulkImportResult, error) {
	result := &BulkImportResult{}

	for _, input := range websites {
		// Validate URL
		if input.URL == "" {
			result.Failed = append(result.Failed, BulkImportError{
				URL:   input.URL,
				Error: "URL harus diisi",
			})
			continue
		}

		// Auto-prepend https:// if no scheme provided
		if !strings.HasPrefix(input.URL, "http://") && !strings.HasPrefix(input.URL, "https://") {
			input.URL = "https://" + input.URL
		}

		if err := s.validateURL(input.URL); err != nil {
			result.Failed = append(result.Failed, BulkImportError{
				URL:   input.URL,
				Error: err.Error(),
			})
			continue
		}

		// Auto-generate name from URL if empty
		if input.Name == "" {
			if parsed, err := url.Parse(input.URL); err == nil {
				input.Name = parsed.Hostname()
			} else {
				input.Name = input.URL
			}
		}

		// Check if URL already exists
		existing, err := s.websiteRepo.GetByURL(ctx, input.URL)
		if err != nil {
			result.Failed = append(result.Failed, BulkImportError{
				URL:   input.URL,
				Error: err.Error(),
			})
			continue
		}
		if existing != nil {
			result.Skipped = append(result.Skipped, input.URL)
			continue
		}

		// Create website
		_, err = s.websiteRepo.Create(ctx, &input)
		if err != nil {
			result.Failed = append(result.Failed, BulkImportError{
				URL:   input.URL,
				Error: err.Error(),
			})
			continue
		}

		result.Created = append(result.Created, input.URL)
	}

	logger.Info().
		Int("created", len(result.Created)).
		Int("skipped", len(result.Skipped)).
		Int("failed", len(result.Failed)).
		Msg("Bulk import completed")

	return result, nil
}

type BulkImportResult struct {
	Created []string          `json:"created"`
	Skipped []string          `json:"skipped"`
	Failed  []BulkImportError `json:"failed"`
}

type BulkImportError struct {
	URL   string `json:"url"`
	Error string `json:"error"`
}
