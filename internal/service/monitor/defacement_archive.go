package monitor

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type DefacementArchiveScanner struct {
	cfg            *config.Config
	websiteRepo    *mysql.WebsiteRepository
	defacementRepo *mysql.DefacementRepository
	alertRepo      *mysql.AlertRepository
	httpClient     *http.Client
}

func NewDefacementArchiveScanner(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	defacementRepo *mysql.DefacementRepository,
	alertRepo *mysql.AlertRepository,
) *DefacementArchiveScanner {
	return &DefacementArchiveScanner{
		cfg:            cfg,
		websiteRepo:    websiteRepo,
		defacementRepo: defacementRepo,
		alertRepo:      alertRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ScanAll checks all monitored domains against defacement archives
func (s *DefacementArchiveScanner) ScanAll(ctx context.Context) error {
	websites, err := s.websiteRepo.GetActive(ctx)
	if err != nil {
		return fmt.Errorf("failed to get websites: %w", err)
	}

	// Extract unique base domains
	domains := s.extractDomains(websites)
	logger.Info().Int("domains", len(domains)).Msg("Starting defacement archive scan")

	// Scan Zone-XSEC (most reliable, no captcha)
	s.scanZoneXSec(ctx, domains, websites)

	return nil
}

// extractDomains gets unique base domains from websites
func (s *DefacementArchiveScanner) extractDomains(websites []domain.Website) []string {
	seen := make(map[string]bool)
	var domains []string

	for _, w := range websites {
		parsed, err := url.Parse(w.URL)
		if err != nil {
			continue
		}
		host := parsed.Hostname()

		// Get base domain (e.g. "baliprov.go.id" from "disdikpora.baliprov.go.id")
		baseDomain := getBaseDomain(host)
		if !seen[baseDomain] {
			seen[baseDomain] = true
			domains = append(domains, baseDomain)
		}
	}
	return domains
}

// getBaseDomain extracts the registerable domain
func getBaseDomain(host string) string {
	parts := strings.Split(host, ".")

	// Handle .go.id, .co.id, .or.id etc.
	if len(parts) >= 3 {
		suffix2 := parts[len(parts)-2] + "." + parts[len(parts)-1]
		if suffix2 == "go.id" || suffix2 == "co.id" || suffix2 == "or.id" || suffix2 == "ac.id" {
			if len(parts) >= 4 {
				return parts[len(parts)-3] + "." + suffix2
			}
			return host
		}
	}

	// Standard TLD
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "." + parts[len(parts)-1]
	}
	return host
}

// scanZoneXSec scrapes Zone-XSEC for defacement records
func (s *DefacementArchiveScanner) scanZoneXSec(ctx context.Context, domains []string, websites []domain.Website) {
	scan := &domain.DefacementScan{
		Source: domain.DefacementSourceZoneXSec,
		Status: "running",
	}
	s.defacementRepo.CreateScan(ctx, scan)

	totalChecked := 0
	newIncidents := 0

	for _, baseDomain := range domains {
		select {
		case <-ctx.Done():
			s.defacementRepo.CompleteScan(ctx, scan.ID, totalChecked, newIncidents, "context cancelled")
			return
		default:
		}

		searchURL := fmt.Sprintf("https://zone-xsec.com/search/q=%s", url.PathEscape(baseDomain))
		incidents, err := s.fetchZoneXSec(ctx, searchURL, baseDomain)
		if err != nil {
			logger.Warn().Err(err).Str("domain", baseDomain).Msg("Failed to fetch Zone-XSEC")
			continue
		}

		totalChecked += len(incidents)

		for _, inc := range incidents {
			// Match to monitored website
			websiteID := s.matchWebsite(inc.DefacedURL, websites)
			if websiteID == 0 {
				continue
			}
			inc.WebsiteID = websiteID

			// Check if already exists
			exists, _ := s.defacementRepo.ExistsIncident(ctx, string(inc.Source), inc.DefacedURL)
			if exists {
				continue
			}

			if err := s.defacementRepo.CreateIncident(ctx, &inc); err == nil && inc.ID > 0 {
				newIncidents++
				logger.Warn().
					Str("url", inc.DefacedURL).
					Str("attacker", inc.Attacker).
					Str("source", "zone_xsec").
					Msg("New defacement incident found")

				// Create alert
				s.createAlert(ctx, websiteID, &inc)
			}
		}

		// Rate limit: wait between requests
		time.Sleep(2 * time.Second)
	}

	s.defacementRepo.CompleteScan(ctx, scan.ID, totalChecked, newIncidents, "")
	logger.Info().Int("checked", totalChecked).Int("new", newIncidents).Msg("Zone-XSEC scan completed")
}

// fetchZoneXSec parses the Zone-XSEC search results page
func (s *DefacementArchiveScanner) fetchZoneXSec(ctx context.Context, searchURL, baseDomain string) ([]domain.DefacementIncident, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Max 1MB
	if err != nil {
		return nil, err
	}

	return s.parseZoneXSec(string(body), baseDomain)
}

// parseZoneXSec extracts defacement data from Zone-XSEC HTML
// Table structure: <td>date</td><td>attacker</td><td>team</td><td>country_flag</td><td>url</td><td>mirror_link</td>
func (s *DefacementArchiveScanner) parseZoneXSec(html, baseDomain string) ([]domain.DefacementIncident, error) {
	var incidents []domain.DefacementIncident

	rowRe := regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	tdRe := regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	mirrorRe := regexp.MustCompile(`<a[^>]*href="(/mirror/id/\d+)"`)

	rows := rowRe.FindAllStringSubmatch(html, -1)
	for _, row := range rows {
		content := row[1]

		// Check if this row contains our domain
		if !strings.Contains(strings.ToLower(content), strings.ToLower(baseDomain)) {
			continue
		}

		tds := tdRe.FindAllStringSubmatch(content, -1)
		if len(tds) < 5 {
			continue
		}

		// Column 0: date
		dateStr := strings.TrimSpace(stripTags(tds[0][1]))
		defacedAt := parseDate(dateStr)

		// Column 1: attacker
		attacker := strings.TrimSpace(stripTags(tds[1][1]))

		// Column 2: team
		team := strings.TrimSpace(stripTags(tds[2][1]))

		// Column 3: country flag (skip)

		// Column 4: defaced URL
		defacedURL := strings.TrimSpace(stripTags(tds[4][1]))

		// Column 5: mirror link (if exists)
		mirrorURL := ""
		if len(tds) >= 6 {
			if m := mirrorRe.FindStringSubmatch(tds[5][1]); m != nil {
				mirrorURL = "https://zone-xsec.com" + m[1]
			}
		}

		if defacedURL == "" {
			continue
		}

		// Normalize URL
		if !strings.HasPrefix(defacedURL, "http") {
			defacedURL = "https://" + defacedURL
		}

		inc := domain.DefacementIncident{
			Source:     domain.DefacementSourceZoneXSec,
			DefacedURL: defacedURL,
			Attacker:   attacker,
			Team:       team,
			DefacedAt:  defacedAt,
			MirrorURL:  mirrorURL,
		}
		incidents = append(incidents, inc)
	}

	return incidents, nil
}

// matchWebsite finds the monitored website that matches a defaced URL
func (s *DefacementArchiveScanner) matchWebsite(defacedURL string, websites []domain.Website) int64 {
	parsed, err := url.Parse(defacedURL)
	if err != nil {
		return 0
	}
	defacedHost := strings.ToLower(parsed.Hostname())

	for _, w := range websites {
		wParsed, err := url.Parse(w.URL)
		if err != nil {
			continue
		}
		wHost := strings.ToLower(wParsed.Hostname())

		// Exact match or subdomain match
		if defacedHost == wHost || strings.HasSuffix(defacedHost, "."+wHost) {
			return w.ID
		}
	}

	// Fallback: match any website with same base domain
	defacedBase := getBaseDomain(defacedHost)
	for _, w := range websites {
		wParsed, _ := url.Parse(w.URL)
		if wParsed == nil {
			continue
		}
		wBase := getBaseDomain(wParsed.Hostname())
		if defacedBase == wBase {
			return w.ID
		}
	}
	return 0
}

func (s *DefacementArchiveScanner) createAlert(ctx context.Context, websiteID int64, inc *domain.DefacementIncident) {
	if s.alertRepo == nil {
		return
	}

	title := fmt.Sprintf("Defacement terdeteksi: %s", inc.DefacedURL)
	msg := fmt.Sprintf("Attacker: %s", inc.Attacker)
	if inc.Team != "" {
		msg += fmt.Sprintf(" (team: %s)", inc.Team)
	}
	msg += fmt.Sprintf(" — Sumber: %s", inc.Source)
	if inc.DefacedAt != nil {
		msg += fmt.Sprintf(" — Tanggal: %s", inc.DefacedAt.Format("2006-01-02"))
	}

	alert := &domain.AlertCreate{
		WebsiteID: websiteID,
		Type:      "defacement_archive",
		Severity:  "critical",
		Title:     title,
		Message:   msg,
	}

	if _, err := s.alertRepo.Create(ctx, alert); err != nil {
		logger.Error().Err(err).Msg("Failed to create defacement alert")
	}
}

// Utility functions

func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

func parseDate(s string) *time.Time {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05",
		"02/01/2006",
		"01/02/2006",
		"Jan 2, 2006",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return &t
		}
	}
	return nil
}
