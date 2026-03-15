package monitor

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type ContentScanner struct {
	cfg          *config.Config
	websiteRepo  *mysql.WebsiteRepository
	checkRepo    *mysql.CheckRepository
	alertRepo    *mysql.AlertRepository
	keywordRepo  *mysql.KeywordRepository
	dorkScanner  *DorkScanner
	httpClient   *http.Client
	regexCache   sync.Map // maps string pattern to *regexp.Regexp
}

func NewContentScanner(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
	keywordRepo *mysql.KeywordRepository,
	dorkScanner *DorkScanner,
) *ContentScanner {
	// Create custom transport with:
	// 1. InsecureSkipVerify for self-signed/expired certs
	// 2. HTTP/2 completely disabled to avoid protocol errors with malformed headers
	// 3. TLSNextProto set to empty map to force HTTP/1.1 only
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		ForceAttemptHTTP2:   false,
		TLSNextProto:        make(map[string]func(authority string, c *tls.Conn) http.RoundTripper), // Disable HTTP/2
	}

	client := &http.Client{
		Timeout:   time.Duration(cfg.Monitoring.HTTPTimeout) * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	return &ContentScanner{
		cfg:         cfg,
		websiteRepo: websiteRepo,
		checkRepo:   checkRepo,
		alertRepo:   alertRepo,
		keywordRepo: keywordRepo,
		dorkScanner: dorkScanner,
		httpClient:  client,
	}
}

// getOrCompileRegex returns a cached compiled regex, or compiles and caches it on first use.
func (s *ContentScanner) getOrCompileRegex(pattern string) (*regexp.Regexp, error) {
	if cached, ok := s.regexCache.Load(pattern); ok {
		return cached.(*regexp.Regexp), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}
	actual, _ := s.regexCache.LoadOrStore(pattern, re)
	return actual.(*regexp.Regexp), nil
}

// ScanWebsite performs content scanning on a website
func (s *ContentScanner) ScanWebsite(ctx context.Context, website *domain.Website) (*domain.ContentScan, error) {
	scan := &domain.ContentScan{
		WebsiteID: website.ID,
		ScanType:  "quick",
		IsClean:   true,
	}

	// Fetch the page
	req, err := http.NewRequestWithContext(ctx, "GET", website.URL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read body (limit to 5MB)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, err
	}

	bodyStr := string(bodyBytes)

	// Calculate page hash
	hash := sha256.Sum256(bodyBytes)
	scan.PageHash = domain.NewNullString(hex.EncodeToString(hash[:]))

	// Parse HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(bodyStr))
	if err != nil {
		return nil, err
	}

	// Get page title
	title := doc.Find("title").First().Text()
	scan.PageTitle = domain.NewNullString(strings.TrimSpace(title))

	// Load keywords from database
	keywords, err := s.keywordRepo.GetAll(ctx)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load keywords from DB, using config")
		// Fallback to config keywords
		keywords = s.configToKeywords()
	}

	var findings []domain.ContentFinding

	// 1. Scan for gambling/defacement keywords
	keywordFindings := s.scanKeywords(bodyStr, keywords)
	findings = append(findings, keywordFindings...)
	scan.KeywordsFound = len(keywordFindings)

	// 2. Scan using dork patterns for deeper detection
	dorkFindings := s.scanWithDorkPatterns(bodyStr)
	findings = append(findings, dorkFindings...)
	scan.KeywordsFound += len(dorkFindings)

	// 3. Scan for suspicious iframes
	iframeFindings := s.scanIframes(doc, website.URL)
	findings = append(findings, iframeFindings...)
	scan.IframesFound = len(iframeFindings)

	// 4. Scan for suspicious redirects/meta refresh
	redirectFindings := s.scanRedirects(doc, bodyStr)
	findings = append(findings, redirectFindings...)
	scan.RedirectsFound = len(redirectFindings)

	// 5. Scan meta tags for suspicious content
	metaFindings := s.scanMetaTags(doc, keywords)
	findings = append(findings, metaFindings...)

	// Determine if clean
	if len(findings) > 0 {
		scan.IsClean = false

		// Serialize findings to JSON
		findingsJSON, _ := json.Marshal(findings)
		scan.Findings = domain.NewNullString(string(findingsJSON))
	}

	// Save scan result
	id, err := s.checkRepo.CreateContentScan(ctx, scan)
	if err != nil {
		logger.Error().Err(err).Int64("website_id", website.ID).Msg("Failed to save content scan")
		return nil, err
	}
	scan.ID = id

	// Update website content status
	s.websiteRepo.UpdateContentStatus(ctx, website.ID, scan.IsClean)

	// Handle alerts
	s.handleAlerts(ctx, scan, website, findings)

	return scan, nil
}

func (s *ContentScanner) configToKeywords() []domain.Keyword {
	var keywords []domain.Keyword

	for _, kw := range s.cfg.Keywords.Gambling {
		keywords = append(keywords, domain.Keyword{
			Keyword:  kw,
			Category: "gambling",
			Weight:   8,
			IsActive: true,
		})
	}

	for _, kw := range s.cfg.Keywords.Defacement {
		keywords = append(keywords, domain.Keyword{
			Keyword:  kw,
			Category: "defacement",
			Weight:   10,
			IsActive: true,
		})
	}

	return keywords
}

// safePhraseWhitelist contains Indonesian phrases that contain gambling keywords as substrings
// but are legitimate content that should NOT be flagged as gambling/judol.
var safePhraseWhitelist = []string{
	"kerangka acuan",
	"acuan kerja",
	"acuan teknis",
	"acuan pelaksanaan",
	"hoki respirasi",
	"slotted",
	"time slot",
	"slot waktu",
	"slot anggaran",
	"slot jadwal",
}

// matchesWholeWord checks whether the keyword appears as a whole word in the text,
// using Unicode word boundaries (\b). Falls back to substring match if the regex
// cannot be compiled (e.g. keyword contains special regex characters).
func matchesWholeWord(text, keyword string) bool {
	pattern := `(?i)\b` + regexp.QuoteMeta(keyword) + `\b`
	re, err := regexp.Compile(pattern)
	if err != nil {
		// Fallback: plain case-insensitive substring match
		return strings.Contains(strings.ToLower(text), strings.ToLower(keyword))
	}
	return re.MatchString(text)
}

// containsSafePhrase reports whether the surrounding context of a keyword match
// belongs to a whitelisted safe phrase, meaning it is a false positive.
func containsSafePhrase(text string) bool {
	textLower := strings.ToLower(text)
	for _, safe := range safePhraseWhitelist {
		if strings.Contains(textLower, safe) {
			return true
		}
	}
	return false
}

func (s *ContentScanner) scanKeywords(content string, keywords []domain.Keyword) []domain.ContentFinding {
	var findings []domain.ContentFinding

	for _, kw := range keywords {
		if !kw.IsActive {
			continue
		}

		var found bool
		var snippet string

		if kw.IsRegex {
			re, err := s.getOrCompileRegex("(?i)" + kw.Keyword)
			if err == nil {
				matches := re.FindStringIndex(content)
				if matches != nil {
					found = true
					start := max(0, matches[0]-50)
					end := min(len(content), matches[1]+50)
					snippet = content[start:end]
				}
			}
		} else {
			// Use whole-word matching to avoid false positives like
			// "acuan" → "cuan", "hokirespirasi" → "hoki", etc.
			if matchesWholeWord(content, kw.Keyword) {
				// Find the actual position for snippet extraction
				pattern := `(?i)\b` + regexp.QuoteMeta(kw.Keyword) + `\b`
				if re, err := regexp.Compile(pattern); err == nil {
					if loc := re.FindStringIndex(content); loc != nil {
						start := max(0, loc[0]-50)
						end := min(len(content), loc[1]+50)
						snippet = content[start:end]
						// Skip if the snippet context belongs to a safe phrase
						if containsSafePhrase(snippet) {
							continue
						}
						found = true
					}
				}
			}
		}

		if found {
			findings = append(findings, domain.ContentFinding{
				Type:     "keyword",
				Category: kw.Category, // gambling, defacement
				Value:    kw.Keyword,
				Location: "body",
				Snippet:  cleanSnippet(snippet),
			})
		}
	}

	return findings
}

// scanWithDorkPatterns uses the dork scanner's extensive pattern library to detect
// gambling, defacement, malware, webshell, phishing, and other threats in page content.
// This leverages the 546+ patterns from the dork scanner without duplicating crawling logic.
func (s *ContentScanner) scanWithDorkPatterns(content string) []domain.ContentFinding {
	if s.dorkScanner == nil {
		return nil
	}

	patterns := s.dorkScanner.GetPatterns()
	if len(patterns) == 0 {
		return nil
	}

	var findings []domain.ContentFinding

	// Track already-found values to avoid duplicates with keyword scan
	seen := make(map[string]bool)

	for _, pattern := range patterns {
		if !pattern.IsActive {
			continue
		}

		// Map dork category to content finding category
		category := string(pattern.Category)

		if pattern.IsRegex && pattern.Pattern != "" {
			// Regex pattern matching
			re, err := s.getOrCompileRegex(pattern.Pattern)
			if err != nil {
				continue
			}

			matches := re.FindAllStringIndex(content, 5) // Limit to 5 matches per pattern
			for _, match := range matches {
				matchedText := content[match[0]:match[1]]
				dedupKey := category + ":" + strings.ToLower(matchedText)
				if seen[dedupKey] {
					continue
				}
				seen[dedupKey] = true

				start := max(0, match[0]-50)
				end := min(len(content), match[1]+50)
				snippet := content[start:end]

				findings = append(findings, domain.ContentFinding{
					Type:        "dork_pattern",
					Category:    category,
					Value:       matchedText,
					Location:    "body",
					Snippet:     cleanSnippet(snippet),
					PatternName: pattern.Name,
					Severity:    string(pattern.Severity),
				})
			}
		} else if len(pattern.Keywords) > 0 {
			// Keyword matching from dork patterns — use whole-word match to avoid
			// false positives (e.g. "slot" inside "slotted" or "time slot context").
			for _, keyword := range pattern.Keywords {
				keywordLower := strings.ToLower(keyword)
				dedupKey := category + ":" + keywordLower
				if seen[dedupKey] {
					continue
				}

				if !matchesWholeWord(content, keyword) {
					continue
				}

				// Get snippet around the match position
				wPattern := `(?i)\b` + regexp.QuoteMeta(keyword) + `\b`
				re, err := regexp.Compile(wPattern)
				if err != nil {
					continue
				}
				loc := re.FindStringIndex(content)
				if loc == nil {
					continue
				}

				start := max(0, loc[0]-50)
				end := min(len(content), loc[1]+50)
				snippet := content[start:end]

				// Skip safe whitelisted phrases
				if containsSafePhrase(snippet) {
					continue
				}

				seen[dedupKey] = true

				findings = append(findings, domain.ContentFinding{
					Type:        "dork_pattern",
					Category:    category,
					Value:       keyword,
					Location:    "body",
					Snippet:     cleanSnippet(snippet),
					PatternName: pattern.Name,
					Severity:    string(pattern.Severity),
				})
			}
		}
	}

	if len(findings) > 0 {
		logger.Info().Int("dork_pattern_findings", len(findings)).Msg("Dork pattern scan completed")
	}

	return findings
}

func (s *ContentScanner) scanIframes(doc *goquery.Document, baseURL string) []domain.ContentFinding {
	var findings []domain.ContentFinding

	baseHost := ""
	if parsed, err := url.Parse(baseURL); err == nil {
		baseHost = parsed.Host
	}

	// Suspicious iframe patterns
	suspiciousPatterns := []string{
		"slot", "casino", "judi", "togel", "poker",
		"bet", "gambling", "777", "jackpot",
	}

	doc.Find("iframe").Each(func(i int, sel *goquery.Selection) {
		src, exists := sel.Attr("src")
		if !exists || src == "" {
			return
		}

		// Check if iframe points to external suspicious domain
		if parsed, err := url.Parse(src); err == nil {
			if parsed.Host != "" && parsed.Host != baseHost {
				// Check for suspicious patterns in URL
				srcLower := strings.ToLower(src)
				for _, pattern := range suspiciousPatterns {
					if strings.Contains(srcLower, pattern) {
						findings = append(findings, domain.ContentFinding{
							Type:     "iframe",
							Category: "gambling", // iframe patterns are gambling-related
							Value:    src,
							Location: "body",
							Snippet:  fmt.Sprintf("Suspicious iframe: %s", src),
						})
						break
					}
				}
			}
		}
	})

	return findings
}

func (s *ContentScanner) scanRedirects(doc *goquery.Document, content string) []domain.ContentFinding {
	var findings []domain.ContentFinding

	// Check meta refresh
	doc.Find("meta[http-equiv='refresh']").Each(func(i int, sel *goquery.Selection) {
		contentAttr, _ := sel.Attr("content")
		if strings.Contains(strings.ToLower(contentAttr), "url=") {
			// Extract URL from content
			parts := strings.Split(contentAttr, "url=")
			if len(parts) > 1 {
				redirectURL := strings.TrimSpace(parts[1])
				findings = append(findings, domain.ContentFinding{
					Type:     "redirect",
					Category: "suspicious",
					Value:    redirectURL,
					Location: "meta",
					Snippet:  fmt.Sprintf("Meta refresh redirect to: %s", redirectURL),
				})
			}
		}
	})

	// Check for JavaScript redirects
	jsRedirectPatterns := []string{
		`window\.location\s*=`,
		`location\.href\s*=`,
		`location\.replace\s*\(`,
	}

	for _, pattern := range jsRedirectPatterns {
		re, err := s.getOrCompileRegex(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(content) {
			findings = append(findings, domain.ContentFinding{
				Type:     "redirect",
				Category: "suspicious",
				Value:    pattern,
				Location: "script",
				Snippet:  "JavaScript redirect detected",
			})
		}
	}

	return findings
}

func (s *ContentScanner) scanMetaTags(doc *goquery.Document, keywords []domain.Keyword) []domain.ContentFinding {
	var findings []domain.ContentFinding

	// Check meta description and keywords
	metaTags := []string{"description", "keywords"}

	for _, tag := range metaTags {
		content, _ := doc.Find(fmt.Sprintf("meta[name='%s']", tag)).Attr("content")
		if content == "" {
			continue
		}

		for _, kw := range keywords {
			if !kw.IsActive {
				continue
			}
			// Use whole-word matching for meta tag content too
			if matchesWholeWord(content, kw.Keyword) && !containsSafePhrase(content) {
				findings = append(findings, domain.ContentFinding{
					Type:     "keyword",
					Category: kw.Category, // gambling, defacement
					Value:    kw.Keyword,
					Location: fmt.Sprintf("meta:%s", tag),
					Snippet:  content,
				})
			}
		}
	}

	return findings
}

func (s *ContentScanner) handleAlerts(ctx context.Context, scan *domain.ContentScan, website *domain.Website, findings []domain.ContentFinding) {
	if scan.IsClean {
		// Resolve existing alerts if website is now clean
		// Resolve judol alerts
		existingJudol, _ := s.alertRepo.GetLatestAlertByType(ctx, website.ID, domain.AlertTypeJudolDetected)
		if existingJudol != nil && !existingJudol.IsResolved {
			s.alertRepo.Resolve(ctx, existingJudol.ID, 0, "Konten sudah bersih berdasarkan scan otomatis")
			logger.Info().Str("website", website.Name).Msg("Judol alert resolved - content is clean")
		}
		// Resolve defacement alerts
		existingDefacement, _ := s.alertRepo.GetLatestAlertByType(ctx, website.ID, domain.AlertTypeDefacement)
		if existingDefacement != nil && !existingDefacement.IsResolved {
			s.alertRepo.Resolve(ctx, existingDefacement.ID, 0, "Konten sudah bersih berdasarkan scan otomatis")
			logger.Info().Str("website", website.Name).Msg("Defacement alert resolved - content is clean")
		}
		return
	}

	// Categorize findings by category
	categoryFindings := make(map[string][]domain.ContentFinding)
	for _, f := range findings {
		categoryFindings[f.Category] = append(categoryFindings[f.Category], f)
	}

	// Create gambling/judol alert if needed
	if gf := categoryFindings["gambling"]; len(gf) > 0 {
		s.createCategoryAlert(ctx, website, scan, gf, domain.AlertTypeJudolDetected, "JUDOL", "gambling")
	}

	// Create defacement alert if needed
	if df := categoryFindings["defacement"]; len(df) > 0 {
		s.createCategoryAlert(ctx, website, scan, df, domain.AlertTypeDefacement, "DEFACEMENT", "defacement")
	}

	// Create security alerts for dork-detected categories (webshell, malware, backdoor, injection, phishing, seo_spam)
	securityCategories := map[string]string{
		"webshell":  "WEBSHELL",
		"malware":   "MALWARE",
		"backdoor":  "BACKDOOR",
		"injection": "CODE INJECTION",
		"phishing":  "PHISHING",
		"seo_spam":  "SEO SPAM",
	}
	for cat, label := range securityCategories {
		if cf := categoryFindings[cat]; len(cf) > 0 {
			s.createCategoryAlert(ctx, website, scan, cf, domain.AlertTypeSecurityIssue, label, cat)
		}
	}

	// Create suspicious content alert for remaining uncategorized findings
	var suspiciousFindings []domain.ContentFinding
	knownCategories := map[string]bool{
		"gambling": true, "defacement": true, "webshell": true, "malware": true,
		"backdoor": true, "injection": true, "phishing": true, "seo_spam": true,
	}
	for cat, cf := range categoryFindings {
		if !knownCategories[cat] {
			suspiciousFindings = append(suspiciousFindings, cf...)
		}
	}
	if len(suspiciousFindings) > 0 && len(categoryFindings["gambling"]) == 0 && len(categoryFindings["defacement"]) == 0 {
		s.createCategoryAlert(ctx, website, scan, suspiciousFindings, domain.AlertTypeSecurityIssue, "KONTEN MENCURIGAKAN", "suspicious")
	}
}

func (s *ContentScanner) createCategoryAlert(ctx context.Context, website *domain.Website, scan *domain.ContentScan, findings []domain.ContentFinding, alertType domain.AlertType, label string, category string) {
	// Check if alert already exists (deduplication)
	exists, _ := s.alertRepo.HasRecentUnresolvedAlert(ctx, website.ID, alertType)
	if exists {
		logger.Debug().Int64("website_id", website.ID).Str("type", string(alertType)).Msg("Skipping duplicate alert")
		return
	}

	// Categorize findings with details
	var keywordDetails []map[string]string
	var iframeDetails []map[string]string
	var redirectDetails []map[string]string
	var dorkPatternDetails []map[string]string

	for _, f := range findings {
		detail := map[string]string{
			"value":    f.Value,
			"location": f.Location,
			"snippet":  f.Snippet,
			"category": f.Category,
		}
		if f.PatternName != "" {
			detail["pattern_name"] = f.PatternName
		}
		if f.Severity != "" {
			detail["severity"] = f.Severity
		}
		switch f.Type {
		case "keyword":
			keywordDetails = append(keywordDetails, detail)
		case "iframe":
			iframeDetails = append(iframeDetails, detail)
		case "redirect":
			redirectDetails = append(redirectDetails, detail)
		case "dork_pattern":
			dorkPatternDetails = append(dorkPatternDetails, detail)
		}
	}

	// Create alert message
	categoryMessages := map[string]string{
		"gambling":   "Ditemukan konten judi online (judol) pada website %s.",
		"defacement": "Website %s terdeteksi mengalami defacement.",
		"webshell":   "Ditemukan indikasi webshell pada website %s.",
		"malware":    "Ditemukan indikasi malware/cryptominer pada website %s.",
		"backdoor":   "Ditemukan indikasi backdoor pada website %s.",
		"injection":  "Ditemukan indikasi code injection pada website %s.",
		"phishing":   "Ditemukan indikasi phishing pada website %s.",
		"seo_spam":   "Ditemukan SEO spam injection pada website %s.",
	}
	message, ok := categoryMessages[category]
	if ok {
		message = fmt.Sprintf(message, website.URL)
	} else {
		message = fmt.Sprintf("Ditemukan konten mencurigakan pada website %s.", website.URL)
	}

	// Build context with detailed findings
	alertContext := map[string]interface{}{
		"total_findings": len(findings),
		"page_title":     scan.PageTitle.String,
		"page_hash":      scan.PageHash.String,
		"category":       category,
	}

	if len(keywordDetails) > 0 {
		alertContext["keywords"] = keywordDetails
		message += fmt.Sprintf("\n• %d keyword mencurigakan ditemukan", len(keywordDetails))
	}
	if len(iframeDetails) > 0 {
		alertContext["iframes"] = iframeDetails
		message += fmt.Sprintf("\n• %d iframe mencurigakan ditemukan", len(iframeDetails))
	}
	if len(redirectDetails) > 0 {
		alertContext["redirects"] = redirectDetails
		message += fmt.Sprintf("\n• %d redirect mencurigakan ditemukan", len(redirectDetails))
	}
	if len(dorkPatternDetails) > 0 {
		alertContext["dork_patterns"] = dorkPatternDetails
		message += fmt.Sprintf("\n• %d pola dork terdeteksi", len(dorkPatternDetails))
	}

	alert := &domain.AlertCreate{
		WebsiteID: website.ID,
		Type:      alertType,
		Severity:  domain.SeverityCritical,
		Title:     fmt.Sprintf("%s TERDETEKSI: %s", label, website.Name),
		Message:   message,
		Context:   alertContext,
	}

	_, err := s.alertRepo.Create(ctx, alert)
	if err != nil {
		logger.Error().Err(err).Str("category", category).Msg("Failed to create content alert")
	} else {
		logger.Warn().Str("website", website.Name).Str("category", category).Int("findings", len(findings)).Msgf("%s DETECTED - Alert created", label)
	}
}

// ScanAllWebsites scans all active websites for suspicious content with concurrent limiting
func (s *ContentScanner) ScanAllWebsites(ctx context.Context) error {
	websites, err := s.websiteRepo.GetActive(ctx)
	if err != nil {
		return err
	}

	logger.Info().Int("count", len(websites)).Msg("Starting content scan for all websites")

	// Use concurrent limiting
	maxConcurrent := s.cfg.Monitoring.MaxConcurrentChecks
	if maxConcurrent <= 0 {
		maxConcurrent = 20 // default
	}

	// Create semaphore channel
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for _, website := range websites {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			wg.Add(1)
			// Acquire semaphore
			sem <- struct{}{}

			go func(w domain.Website) {
				defer wg.Done()
				defer func() { <-sem }() // Release semaphore

				_, err := s.ScanWebsite(ctx, &w)
				if err != nil {
					logger.Error().Err(err).Str("url", w.URL).Msg("Content scan failed")
				}
			}(website)
		}
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return nil
}

// Helper functions
func cleanSnippet(s string) string {
	// Remove newlines and excessive whitespace
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.Join(strings.Fields(s), " ")

	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
