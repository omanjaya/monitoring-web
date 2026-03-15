package monitor

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/ai"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type DorkScanner struct {
	cfg          *config.Config
	websiteRepo  *mysql.WebsiteRepository
	dorkRepo     *mysql.DorkRepository
	alertRepo    *mysql.AlertRepository
	settingsRepo *mysql.SettingsRepository
	aiVerifier   *ai.Verifier
	httpClient   *http.Client
	patterns     []domain.DorkPattern
	compiledRe   map[int64]*regexp.Regexp
	mu           sync.RWMutex
}

func NewDorkScanner(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	dorkRepo *mysql.DorkRepository,
	alertRepo *mysql.AlertRepository,
	settingsRepo *mysql.SettingsRepository,
) *DorkScanner {
	scanner := &DorkScanner{
		cfg:          cfg,
		websiteRepo:  websiteRepo,
		dorkRepo:     dorkRepo,
		alertRepo:    alertRepo,
		settingsRepo: settingsRepo,
		aiVerifier:   ai.NewVerifier(&cfg.AI),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		patterns:   domain.DefaultDorkPatterns,
		compiledRe: make(map[int64]*regexp.Regexp),
	}

	// Compile regex patterns
	scanner.compilePatterns()

	return scanner
}

func (s *DorkScanner) compilePatterns() {
	for i, p := range s.patterns {
		if p.IsRegex && p.Pattern != "" {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				logger.Warn().Err(err).Str("pattern", p.Name).Msg("Failed to compile regex pattern")
				continue
			}
			s.compiledRe[int64(i)] = re
		}
	}
	logger.Info().Int("compiled", len(s.compiledRe)).Msg("Dork patterns compiled")
}

// ScanWebsite performs a comprehensive dork scan on a website
func (s *DorkScanner) ScanWebsite(ctx context.Context, websiteID int64, scanType string, categories []domain.DorkCategory) (*domain.DorkScanResult, error) {
	startTime := time.Now()

	// Get website
	website, err := s.websiteRepo.GetByID(ctx, websiteID)
	if err != nil || website == nil {
		return nil, fmt.Errorf("website not found: %w", err)
	}

	result := &domain.DorkScanResult{
		WebsiteID:         websiteID,
		WebsiteName:       website.Name,
		WebsiteURL:        website.URL,
		ScanType:          scanType,
		Status:            "running",
		CategoriesScanned: categories,
		Detections:        make([]domain.DorkDetection, 0),
		StartedAt:         &startTime,
		ScannedAt:         time.Now(),
		CreatedAt:         time.Now(),
	}

	// Determine which patterns to use
	patterns := s.getPatternsByCategories(categories)
	result.TotalPatterns = len(patterns)
	pagesScanned := 1

	// Fetch main page
	content, err := s.fetchPage(ctx, website.URL)
	if err != nil {
		logger.Warn().Err(err).Str("url", website.URL).Msg("Failed to fetch main page")
		result.Status = "failed"
		result.ErrorMessage = err.Error()
		now := time.Now()
		result.CompletedAt = &now
		return result, nil
	}

	// Scan main page
	mainPageDetections := s.scanContent(content, website.URL, patterns)
	result.Detections = append(result.Detections, mainPageDetections...)

	// If full scan, crawl additional pages
	if scanType == "full" {
		additionalURLs := s.extractLinks(content, website.URL)
		for _, pageURL := range additionalURLs {
			select {
			case <-ctx.Done():
				break
			default:
				pageContent, err := s.fetchPage(ctx, pageURL)
				if err != nil {
					continue
				}
				pagesScanned++
				pageDetections := s.scanContent(pageContent, pageURL, patterns)
				result.Detections = append(result.Detections, pageDetections...)
			}
		}
	}

	// AI-powered false positive verification (check DB settings first, fallback to config)
	if len(result.Detections) > 0 {
		s.refreshAIVerifier(ctx)
		if s.aiVerifier.IsEnabled() {
			originalCount := len(result.Detections)
			result.Detections = s.verifyWithAI(ctx, website.Name, website.URL, result.Detections)
			result.AIFilteredCount = originalCount - len(result.Detections)
			// Mark surviving detections as AI-verified
			for i := range result.Detections {
				result.Detections[i].AIVerified = true
			}
		}
	}

	// Calculate statistics
	result.TotalPagesScanned = pagesScanned
	result.TotalDetections = len(result.Detections)
	result.MatchedPatterns = len(result.Detections)
	result.ThreatLevel = s.calculateThreatLevel(result.Detections)
	result.ScanDuration = time.Since(startTime).Milliseconds()
	result.Status = "completed"
	now := time.Now()
	result.CompletedAt = &now

	// Count by severity
	for _, d := range result.Detections {
		switch d.Severity {
		case domain.DorkSeverityCritical:
			result.CriticalCount++
		case domain.DorkSeverityHigh:
			result.HighCount++
		case domain.DorkSeverityMedium:
			result.MediumCount++
		case domain.DorkSeverityLow:
			result.LowCount++
		}
	}

	// Save result to database
	if s.dorkRepo != nil {
		if err := s.dorkRepo.CreateScanResult(ctx, result); err != nil {
			logger.Error().Err(err).Msg("Failed to save dork scan result")
		} else {
			// Save detections
			for i := range result.Detections {
				result.Detections[i].ScanResultID = result.ID
				result.Detections[i].WebsiteID = websiteID
			}
			if err := s.dorkRepo.CreateDetections(ctx, result.Detections); err != nil {
				logger.Error().Err(err).Msg("Failed to save dork detections")
			}
			// Update last scan time
			s.dorkRepo.UpdateLastScan(ctx, websiteID)
		}
	}

	// Create alerts for critical/high detections
	s.createAlerts(ctx, website, result)

	return result, nil
}

// ReloadPatterns reloads patterns from the database
func (s *DorkScanner) ReloadPatterns(ctx context.Context) error {
	if s.dorkRepo == nil {
		return nil
	}

	patterns, err := s.dorkRepo.GetActivePatterns(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to reload dork patterns")
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing compiled patterns
	s.compiledRe = make(map[int64]*regexp.Regexp)

	// Merge with default patterns
	s.patterns = domain.DefaultDorkPatterns

	// Add database patterns
	for _, p := range patterns {
		// Check if pattern already exists by name
		exists := false
		for _, dp := range s.patterns {
			if dp.Name == p.Name {
				exists = true
				break
			}
		}
		if !exists {
			s.patterns = append(s.patterns, p)
		}
	}

	// Recompile regex patterns
	for i, p := range s.patterns {
		if p.IsRegex && p.Pattern != "" {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				logger.Warn().Err(err).Str("pattern", p.Name).Msg("Failed to compile regex pattern")
				continue
			}
			s.compiledRe[int64(i)] = re
		}
	}

	logger.Info().Int("total_patterns", len(s.patterns)).Int("compiled_regex", len(s.compiledRe)).Msg("Dork patterns reloaded")
	return nil
}

// ScanAllWebsites scans all active websites
func (s *DorkScanner) ScanAllWebsites(ctx context.Context) error {
	websites, _, err := s.websiteRepo.GetAll(ctx, domain.WebsiteFilter{IsActive: boolPtr(true)})
	if err != nil {
		return err
	}

	logger.Info().Int("count", len(websites)).Msg("Starting dork scan for all websites")

	// Use semaphore for concurrency control
	maxConcurrent := s.cfg.Monitoring.MaxConcurrentChecks
	if maxConcurrent == 0 {
		maxConcurrent = 10
	}

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, website := range websites {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			wg.Add(1)
			sem <- struct{}{}

			go func(w domain.Website) {
				defer wg.Done()
				defer func() { <-sem }()

				_, err := s.ScanWebsite(ctx, w.ID, "quick", nil)
				if err != nil {
					logger.Error().Err(err).Int64("website_id", w.ID).Msg("Dork scan failed")
				}
			}(website)
		}
	}

	wg.Wait()
	return nil
}

func (s *DorkScanner) fetchPage(ctx context.Context, pageURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", pageURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Limit read size to 5MB
	limitedReader := io.LimitReader(resp.Body, 5*1024*1024)
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (s *DorkScanner) scanContent(content, pageURL string, patterns []domain.DorkPattern) []domain.DorkDetection {
	var detections []domain.DorkDetection
	contentLower := strings.ToLower(content)

	for i, pattern := range patterns {
		if !pattern.IsActive {
			continue
		}

		if pattern.IsRegex && pattern.Pattern != "" {
			// Use compiled regex
			s.mu.RLock()
			re, exists := s.compiledRe[int64(i)]
			s.mu.RUnlock()

			if exists {
				matches := re.FindAllString(content, 10) // Limit to 10 matches per pattern
				for _, match := range matches {
					ctx := getContext(content, match, 150)
					// Check for false positive before adding detection
					if isFalsePositive(match, ctx, pattern.Category) {
						logger.Debug().
							Str("pattern", pattern.Name).
							Str("match", truncateString(match, 50)).
							Str("url", pageURL).
							Msg("Skipping false positive detection")
						continue
					}
					detection := domain.DorkDetection{
						PatternID:      pattern.ID,
						Category:       pattern.Category,
						PatternName:    pattern.Name,
						Severity:       pattern.Severity,
						URL:            pageURL,
						MatchedContent: truncateString(match, 500),
						MatchedText:    truncateString(match, 500),
						Location:       pageURL,
						Context:        ctx,
						Confidence:     0.9,
						DetectedAt:     time.Now(),
						CreatedAt:      time.Now(),
					}
					detections = append(detections, detection)
				}
			}
		} else if len(pattern.Keywords) > 0 {
			// Keyword matching
			for _, keyword := range pattern.Keywords {
				keywordLower := strings.ToLower(keyword)
				// Use word boundary matching to prevent "war" matching "warm", "2d" matching "2days", etc.
				idx := findWordBoundaryMatch(contentLower, keywordLower)
				if idx == -1 {
					continue
				}
				{
					actualMatch := content[idx : idx+len(keyword)]
					ctx := getContext(content, actualMatch, 150)

					// Check for false positive before adding detection
					if isFalsePositive(keyword, ctx, pattern.Category) {
						logger.Debug().
							Str("pattern", pattern.Name).
							Str("keyword", keyword).
							Str("url", pageURL).
							Msg("Skipping false positive keyword detection")
						continue
					}

					detection := domain.DorkDetection{
						PatternID:      pattern.ID,
						Category:       pattern.Category,
						PatternName:    pattern.Name,
						Severity:       pattern.Severity,
						URL:            pageURL,
						MatchedContent: keyword,
						MatchedText:    keyword,
						Location:       pageURL,
						Context:        ctx,
						Confidence:     0.85,
						DetectedAt:     time.Now(),
						CreatedAt:      time.Now(),
					}
					detections = append(detections, detection)
				}
			}
		}
	}


	// Apply minimum match threshold to reduce false positives:
	// - If only 1 non-critical pattern matched, require confidence > 0.9
	// - If only keyword matches (no regex), require at least 2 different pattern matches
	if len(detections) > 0 {
		detections = applyDetectionThreshold(detections)
	}

	return detections
}

// isFalsePositive checks if a match is likely a false positive based on surrounding context.
// Government websites often contain legitimate uses of words like "transfer", "deposit",
// "bonus", "menang", etc. in fiscal/administrative contexts.
func isFalsePositive(matchedContent string, context string, patternCategory domain.DorkCategory) bool {
	ctx := strings.ToLower(context)
	matched := strings.ToLower(matchedContent)

	// Only apply false positive filtering to gambling and phishing categories
	// Defacement, webshell, malware, backdoor, injection patterns are unlikely false positives
	if patternCategory != domain.DorkCategoryGambling &&
		patternCategory != domain.DorkCategoryPhishing &&
		patternCategory != domain.DorkCategorySEOSpam {
		return false
	}

	// Government/legitimate context indicators (Indonesian government website terms)
	legitimateContexts := []string{
		// Fiscal/budget terms
		"anggaran", "apbd", "apbn", "belanja", "pendapatan",
		"transfer daerah", "dana transfer", "dana desa", "dana alokasi",
		"realisasi anggaran", "laporan keuangan", "neraca", "fiskal",
		// HR/personnel terms
		"bonus demografi", "bonus asn", "bonus pegawai", "tunjangan",
		"gaji pegawai", "insentif kinerja", "remunerasi",
		// Banking/finance (legitimate)
		"deposit tabungan", "bank deposit", "deposito", "tabungan",
		"transfer bank", "transfer dana", "transfer gaji",
		"rekening pemerintah", "bendahara",
		// Education/games
		"game theory", "game edukasi", "permainan edukasi", "bermain peran",
		"bermain anak", "taman bermain", "game tradisional",
		// Sports (legitimate)
		"sportif", "olahraga", "pertandingan", "kompetisi",
		"juara", "pemenang lomba", "pemenang sayembara",
		"turnamen", "kejuaraan",
		// News/article context
		"perjudian ilegal", "razia judi", "penertiban judi",
		"bahaya judi", "sosialisasi anti judi", "pencegahan judi",
		"berita", "artikel", "press release", "siaran pers",
		// Government services
		"layanan publik", "pelayanan masyarakat",
		"pendaftaran", "registrasi", "administrasi",
		// WhatsApp/Telegram for legitimate government contact
		"kontak kami", "hubungi kami", "pusat informasi", "call center",
		"pengaduan", "helpdesk", "pusat bantuan",
	}

	for _, legit := range legitimateContexts {
		if strings.Contains(ctx, legit) {
			return true
		}
	}

	// Check for short ambiguous keywords that need extra context validation
	ambiguousKeywords := map[string]bool{
		"menang": true, "kaya": true, "senang": true, "gembira": true,
		"bangga": true, "panen": true, "hoki": true,
		"win": true, "war": true, "bos": true, "cuan": true,
		"2d": true, "3d": true, "4d": true,
		"situs resmi": true, "situs terpercaya": true,
		"bonus": true, "deposit": true, "transfer": true,
		"game": true, "play": true, "bet": true,
		"livechat 24 jam": true, "cs 24 jam": true, "customer service 24 jam": true,
		"online 24 jam": true, "layanan 24 jam": true,
	}

	if ambiguousKeywords[matched] {
		// These short words are only suspicious in gambling-specific context
		// Check if gambling-specific context words are nearby
		gamblingContext := []string{
			"slot", "togel", "casino", "judi", "taruhan", "betting",
			"gacor", "maxwin", "jackpot", "deposit pulsa", "rtp",
			"bandar", "scatter", "spin", "daftar sekarang",
		}
		hasGamblingContext := false
		for _, gc := range gamblingContext {
			if strings.Contains(ctx, gc) {
				hasGamblingContext = true
				break
			}
		}
		if !hasGamblingContext {
			return true // Ambiguous word without gambling context = likely false positive
		}
	}

	return false
}

// applyDetectionThreshold filters out low-confidence detections when there aren't
// enough corroborating signals. This prevents a single ambiguous keyword match
// on a large page from triggering an alert.
func applyDetectionThreshold(detections []domain.DorkDetection) []domain.DorkDetection {
	// Count unique pattern names to see how many different patterns matched
	uniquePatterns := make(map[string]bool)
	hasCriticalSeverity := false
	hasRegexMatch := false

	for _, d := range detections {
		uniquePatterns[d.PatternName] = true
		if d.Severity == domain.DorkSeverityCritical {
			hasCriticalSeverity = true
		}
		if d.Confidence >= 0.9 {
			hasRegexMatch = true
		}
	}

	// If at least 2 different patterns matched, or any critical severity, or regex match: keep all
	if len(uniquePatterns) >= 2 || hasCriticalSeverity || hasRegexMatch {
		return detections
	}

	// Single non-critical keyword-only match: reduce confidence and filter
	var filtered []domain.DorkDetection
	for _, d := range detections {
		// Lower confidence for single-pattern keyword matches
		d.Confidence = d.Confidence * 0.5
		// Only keep if confidence is still meaningful (> 0.3)
		if d.Confidence > 0.3 {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func (s *DorkScanner) extractLinks(content, baseURL string) []string {
	var links []string
	seen := make(map[string]bool)

	base, err := url.Parse(baseURL)
	if err != nil {
		return links
	}

	doc, err := html.Parse(strings.NewReader(content))
	if err != nil {
		return links
	}

	var extract func(*html.Node)
	extract = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := attr.Val

					// Skip empty, javascript, mailto, tel links
					if href == "" || strings.HasPrefix(href, "javascript:") ||
						strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") ||
						strings.HasPrefix(href, "#") {
						continue
					}

					// Parse and resolve URL
					linkURL, err := url.Parse(href)
					if err != nil {
						continue
					}

					resolved := base.ResolveReference(linkURL)

					// Only include same-domain links
					if resolved.Host == base.Host {
						fullURL := resolved.String()
						if !seen[fullURL] && len(links) < 20 { // Limit to 20 pages
							seen[fullURL] = true
							links = append(links, fullURL)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extract(c)
		}
	}

	extract(doc)
	return links
}

func (s *DorkScanner) getPatternsByCategories(categories []domain.DorkCategory) []domain.DorkPattern {
	if len(categories) == 0 {
		return s.patterns
	}

	categoryMap := make(map[domain.DorkCategory]bool)
	for _, c := range categories {
		categoryMap[c] = true
	}

	var filtered []domain.DorkPattern
	for _, p := range s.patterns {
		if categoryMap[p.Category] {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func (s *DorkScanner) calculateThreatLevel(detections []domain.DorkDetection) domain.DorkSeverity {
	if len(detections) == 0 {
		return domain.DorkSeverityLow
	}

	// Return highest severity found
	hasCritical := false
	hasHigh := false
	hasMedium := false

	for _, d := range detections {
		switch d.Severity {
		case domain.DorkSeverityCritical:
			hasCritical = true
		case domain.DorkSeverityHigh:
			hasHigh = true
		case domain.DorkSeverityMedium:
			hasMedium = true
		}
	}

	if hasCritical {
		return domain.DorkSeverityCritical
	}
	if hasHigh {
		return domain.DorkSeverityHigh
	}
	if hasMedium {
		return domain.DorkSeverityMedium
	}
	return domain.DorkSeverityLow
}

func (s *DorkScanner) createAlerts(ctx context.Context, website *domain.Website, result *domain.DorkScanResult) {
	if len(result.Detections) == 0 {
		// Scan clean — resolve any existing dork-related alerts
		s.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeJudolDetected, "Scan bersih: konten judol tidak lagi terdeteksi")
		s.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeDefacement, "Scan bersih: defacement tidak lagi terdeteksi")
		s.alertRepo.ResolveAllByType(ctx, website.ID, domain.AlertTypeSecurityIssue, "Scan bersih: ancaman keamanan tidak lagi terdeteksi")
		return
	}

	// Group detections by category
	categoryDetections := make(map[domain.DorkCategory][]domain.DorkDetection)
	for _, d := range result.Detections {
		categoryDetections[d.Category] = append(categoryDetections[d.Category], d)
	}

	// Get all unresolved alert types for this website to enable grouping
	unresolvedTypes, _ := s.alertRepo.GetUnresolvedAlertTypes(ctx, website.ID)
	unresolvedTypeSet := make(map[string]bool)
	for _, t := range unresolvedTypes {
		unresolvedTypeSet[t] = true
	}

	for category, detections := range categoryDetections {
		// Determine alert type and severity
		var alertType domain.AlertType
		var severity domain.AlertSeverity
		var title string

		switch category {
		case domain.DorkCategoryGambling:
			alertType = domain.AlertTypeJudolDetected
			severity = domain.SeverityCritical
			title = "Konten Judi Online (Judol) Terdeteksi"
		case domain.DorkCategoryDefacement:
			alertType = domain.AlertTypeDefacement
			severity = domain.SeverityCritical
			title = "Website Defacement Terdeteksi"
		case domain.DorkCategoryShell:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityCritical
			title = "Webshell Terdeteksi"
		case domain.DorkCategoryMalware:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityCritical
			title = "Malware/Cryptominer Terdeteksi"
		case domain.DorkCategoryPhishing:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityWarning
			title = "Indikasi Phishing Terdeteksi"
		case domain.DorkCategorySEOSpam:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityWarning
			title = "SEO Spam Injection Terdeteksi"
		case domain.DorkCategoryBackdoor:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityCritical
			title = "Backdoor Terdeteksi"
		case domain.DorkCategoryInjection:
			alertType = domain.AlertTypeSecurityIssue
			severity = domain.SeverityWarning
			title = "Code Injection Terdeteksi"
		default:
			continue
		}

		// Build message with detection details
		var msgBuilder strings.Builder
		msgBuilder.WriteString(fmt.Sprintf("Ditemukan %d deteksi kategori %s:\n\n", len(detections), category))

		for i, d := range detections {
			if i >= 5 { // Limit to 5 examples
				msgBuilder.WriteString(fmt.Sprintf("\n... dan %d deteksi lainnya", len(detections)-5))
				break
			}
			msgBuilder.WriteString(fmt.Sprintf("- %s: %s\n", d.PatternName, truncateString(d.MatchedText, 100)))
		}

		// Check if similar unresolved alert already exists
		if unresolvedTypeSet[string(alertType)] {
			logger.Debug().
				Int64("website_id", website.ID).
				Str("type", string(alertType)).
				Msg("Skipping duplicate dork alert - unresolved alert exists")
			continue
		}

		// Check cooldown: don't re-alert within 15 minutes of a resolved alert
		hasCooldown, _ := s.alertRepo.HasRecentResolvedAlert(ctx, website.ID, alertType, 15)
		if hasCooldown {
			logger.Debug().
				Int64("website_id", website.ID).
				Str("type", string(alertType)).
				Msg("Skipping dork alert - within cooldown period after resolution")
			continue
		}

		// Create new alert
		alertCreate := &domain.AlertCreate{
			WebsiteID: website.ID,
			Type:      alertType,
			Severity:  severity,
			Title:     title,
			Message:   msgBuilder.String(),
			Context: map[string]interface{}{
				"scan_type":        result.ScanType,
				"detection_count":  len(detections),
				"category":         category,
				"threat_level":     result.ThreatLevel,
				"sample_detection": detections[0].MatchedText,
			},
		}

		if _, err := s.alertRepo.Create(ctx, alertCreate); err != nil {
			logger.Error().Err(err).Msg("Failed to create dork detection alert")
		}
	}
}

// GetPatterns returns all configured patterns
func (s *DorkScanner) GetPatterns() []domain.DorkPattern {
	return s.patterns
}

// GetPatternsByCategory returns patterns for a specific category
func (s *DorkScanner) GetPatternsByCategory(category domain.DorkCategory) []domain.DorkPattern {
	var filtered []domain.DorkPattern
	for _, p := range s.patterns {
		if p.Category == category {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// AddCustomPattern adds a custom pattern
func (s *DorkScanner) AddCustomPattern(pattern domain.DorkPattern) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if pattern.IsRegex && pattern.Pattern != "" {
		re, err := regexp.Compile(pattern.Pattern)
		if err != nil {
			return fmt.Errorf("invalid regex pattern: %w", err)
		}
		s.compiledRe[int64(len(s.patterns))] = re
	}

	s.patterns = append(s.patterns, pattern)
	return nil
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func getContext(content, match string, contextLen int) string {
	idx := strings.Index(content, match)
	if idx == -1 {
		return ""
	}

	start := idx - contextLen
	if start < 0 {
		start = 0
	}

	end := idx + len(match) + contextLen
	if end > len(content) {
		end = len(content)
	}

	return strings.TrimSpace(content[start:end])
}

// findWordBoundaryMatch finds a keyword in content ensuring it's at word boundaries.
// For multi-word keywords (containing spaces), it uses simple substring match.
// For single-word keywords, it checks that characters before/after are not alphanumeric,
// preventing "war" from matching inside "warm" or "2d" matching inside "2days".
func findWordBoundaryMatch(contentLower, keywordLower string) int {
	// Multi-word keywords (e.g. "slot gacor") are specific enough — use substring match
	if strings.Contains(keywordLower, " ") {
		return strings.Index(contentLower, keywordLower)
	}

	// For single-word keywords, scan all occurrences and check word boundaries
	searchFrom := 0
	for searchFrom < len(contentLower) {
		idx := strings.Index(contentLower[searchFrom:], keywordLower)
		if idx == -1 {
			return -1
		}
		absIdx := searchFrom + idx
		endIdx := absIdx + len(keywordLower)

		// Check character before match
		boundaryBefore := absIdx == 0 || !isAlphanumeric(contentLower[absIdx-1])
		// Check character after match
		boundaryAfter := endIdx >= len(contentLower) || !isAlphanumeric(contentLower[endIdx])

		if boundaryBefore && boundaryAfter {
			return absIdx
		}

		// Move past this occurrence and keep searching
		searchFrom = absIdx + 1
	}
	return -1
}

func isAlphanumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

func boolPtr(b bool) *bool {
	return &b
}

// refreshAIVerifier loads AI settings from database and updates the verifier
func (s *DorkScanner) refreshAIVerifier(ctx context.Context) {
	if s.settingsRepo == nil {
		return
	}
	dbSettings, err := s.settingsRepo.GetAISettings(ctx)
	if err != nil {
		return
	}
	// Override config with DB settings if they exist
	aiCfg := config.AIConfig{
		Enabled:  dbSettings.Enabled,
		Provider: dbSettings.Provider,
		APIKey:   dbSettings.APIKey,
		Model:    dbSettings.Model,
	}
	// Fallback to config file values if DB is empty
	if aiCfg.APIKey == "" && s.cfg.AI.APIKey != "" {
		aiCfg = s.cfg.AI
	}
	s.aiVerifier = ai.NewVerifier(&aiCfg)
}

// verifyWithAI sends detections to AI for false positive verification.
// Batches detections (max 20 per API call) and filters out confirmed false positives.
func (s *DorkScanner) verifyWithAI(ctx context.Context, websiteName, websiteURL string, detections []domain.DorkDetection) []domain.DorkDetection {
	const batchSize = 20

	logger.Info().
		Int("detection_count", len(detections)).
		Str("website", websiteName).
		Msg("Starting AI verification of dork detections")

	// Convert to AI detection format
	aiDetections := make([]ai.Detection, len(detections))
	for i, d := range detections {
		aiDetections[i] = ai.Detection{
			PatternName:    d.PatternName,
			Category:       string(d.Category),
			MatchedContent: d.MatchedContent,
			Context:        d.Context,
			URL:            d.URL,
			Confidence:     d.Confidence,
		}
	}

	// Process in batches
	falsePositives := make(map[int]bool)
	for start := 0; start < len(aiDetections); start += batchSize {
		end := start + batchSize
		if end > len(aiDetections) {
			end = len(aiDetections)
		}

		batch := aiDetections[start:end]
		results, err := s.aiVerifier.VerifyDetections(ctx, websiteName, websiteURL, batch)
		if err != nil {
			logger.Warn().Err(err).Msg("AI verification failed for batch, keeping all detections")
			continue
		}

		for _, r := range results {
			globalIdx := start + r.Index
			if globalIdx < len(detections) && r.IsFalsePositive && r.Confidence >= 0.7 {
				falsePositives[globalIdx] = true
				logger.Info().
					Int("index", globalIdx).
					Str("pattern", detections[globalIdx].PatternName).
					Str("matched", truncateString(detections[globalIdx].MatchedContent, 50)).
					Str("reason", r.Reason).
					Msg("AI flagged as false positive")
			}
		}
	}

	if len(falsePositives) == 0 {
		logger.Info().Msg("AI verification: no false positives found")
		return detections
	}

	// Filter out false positives
	var verified []domain.DorkDetection
	for i, d := range detections {
		if !falsePositives[i] {
			verified = append(verified, d)
		}
	}

	logger.Info().
		Int("original", len(detections)).
		Int("false_positives", len(falsePositives)).
		Int("verified", len(verified)).
		Msg("AI verification completed")

	return verified
}
