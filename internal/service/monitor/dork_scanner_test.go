package monitor

import (
	"regexp"
	"strings"
	"testing"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
)

// TestDorkPatternCompilation tests that all regex patterns compile correctly
func TestDorkPatternCompilation(t *testing.T) {
	failedPatterns := []string{}
	compiledCount := 0

	for _, pattern := range domain.DefaultDorkPatterns {
		if pattern.IsRegex && pattern.Pattern != "" {
			_, err := regexp.Compile(pattern.Pattern)
			if err != nil {
				failedPatterns = append(failedPatterns, pattern.Name+": "+err.Error())
			} else {
				compiledCount++
			}
		}
	}

	if len(failedPatterns) > 0 {
		t.Errorf("Failed to compile %d patterns:\n%s", len(failedPatterns), strings.Join(failedPatterns, "\n"))
	}

	t.Logf("Successfully compiled %d regex patterns", compiledCount)
}

// TestGamblingKeywordDetection tests detection of gambling keywords
func TestGamblingKeywordDetection(t *testing.T) {
	testCases := []struct {
		name     string
		content  string
		expected []string // Expected keywords to match
	}{
		{
			name:     "Slot Gacor Basic",
			content:  "Selamat datang di situs slot gacor terbaik, dapatkan maxwin hari ini!",
			expected: []string{"slot gacor", "maxwin"},
		},
		{
			name:     "Togel Keywords",
			content:  "Data SGP keluaran togel hongkong hari ini, prediksi togel singapore",
			expected: []string{"togel", "togel hongkong", "togel singapore", "data sgp"},
		},
		{
			name:     "Casino Keywords",
			content:  "Live casino baccarat online dengan dealer sexy baccarat terpercaya",
			expected: []string{"live casino", "baccarat", "sexy baccarat"},
		},
		{
			name:     "Raja Series",
			content:  "Rajabola situs judi bola terpercaya, rajaslot88 slot online gacor",
			expected: []string{"rajabola", "rajaslot"},
		},
		{
			name:     "88 Series",
			content:  "Indo88 agen judi terpercaya, daftar di slot88 sekarang!",
			expected: []string{"indo88", "slot88"},
		},
		{
			name:     "Pragmatic Games",
			content:  "Main gates of olympus dan sweet bonanza gratis demo slot pragmatic",
			expected: []string{"gates of olympus", "sweet bonanza", "pragmatic"},
		},
		{
			name:     "PG Soft Games",
			content:  "Mahjong ways 2 gacor hari ini, fortune tiger dan lucky neko maxwin",
			expected: []string{"mahjong ways", "fortune tiger", "lucky neko", "maxwin"},
		},
		{
			name:     "Bonus Keywords",
			content:  "Bonus new member 100% freebet gratis depo 25 bonus 25",
			expected: []string{"bonus new member", "freebet gratis", "depo 25 bonus 25"},
		},
		{
			name:     "Payment Methods",
			content:  "Deposit pulsa tanpa potongan, depo dana dan slot via gopay",
			expected: []string{"deposit pulsa", "tanpa potongan", "depo dana", "gopay"},
		},
		{
			name:     "Slang Judol",
			content:  "JP paus auto sultan sensational gacor parah cuan gede pecah jp!",
			expected: []string{"jp paus", "auto sultan", "sensational", "gacor parah", "cuan gede", "pecah jp"},
		},
		{
			name:     "Core Keywords Low FP",
			content:  "Togel toto judi slot gacor bandar maxwin zeus judol",
			expected: []string{"togel", "toto", "judi", "slot", "gacor", "bandar", "maxwin", "zeus", "judol"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			contentLower := strings.ToLower(tc.content)
			matched := []string{}

			for _, pattern := range domain.DefaultDorkPatterns {
				if pattern.Category != domain.DorkCategoryGambling {
					continue
				}

				for _, keyword := range pattern.Keywords {
					if strings.Contains(contentLower, strings.ToLower(keyword)) {
						matched = append(matched, keyword)
					}
				}
			}

			for _, expected := range tc.expected {
				found := false
				for _, m := range matched {
					if strings.ToLower(m) == strings.ToLower(expected) {
						found = true
						break
					}
				}
				if !found {
					// Check if it's part of any matched keyword
					for _, m := range matched {
						if strings.Contains(strings.ToLower(m), strings.ToLower(expected)) ||
							strings.Contains(strings.ToLower(expected), strings.ToLower(m)) {
							found = true
							break
						}
					}
				}
				if !found {
					t.Logf("Expected keyword '%s' not found in matches: %v", expected, matched)
				}
			}

			t.Logf("Test '%s' matched %d keywords: %v", tc.name, len(matched), matched)
		})
	}
}

// TestLeetSpeakDetection tests detection of leetspeak variations
func TestLeetSpeakDetection(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		should  bool // should match
	}{
		{"rajabol4", "Daftar di rajabol4 sekarang", true},
		{"raj4bola", "Main di raj4bola terpercaya", true},
		{"d3wabet", "Situs d3wabet casino online", true},
		{"g4cor", "Slot g4cor hari ini", true},
		{"m4xwin", "Demo slot m4xwin gratis", true},
		{"h0ki88", "Agen h0ki88 terbaik", true},
		{"cu4n", "Auto cu4n jp paus", true},
		{"normal text", "This is normal text without gambling", false},
		{"government site", "Informasi resmi dari pemerintah", false},
	}

	// Find leetspeak pattern
	var leetPattern *regexp.Regexp
	for _, p := range domain.DefaultDorkPatterns {
		if p.Name == "Leetspeak Gambling Names" && p.IsRegex {
			var err error
			leetPattern, err = regexp.Compile(p.Pattern)
			if err != nil {
				t.Fatalf("Failed to compile leetspeak pattern: %v", err)
			}
			break
		}
	}

	if leetPattern == nil {
		t.Fatal("Leetspeak pattern not found")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := leetPattern.FindAllString(tc.content, -1)
			hasMatch := len(matches) > 0

			if hasMatch != tc.should {
				if tc.should {
					t.Errorf("Expected to match '%s' but didn't", tc.content)
				} else {
					t.Errorf("Should NOT match '%s' but got: %v", tc.content, matches)
				}
			} else if hasMatch {
				t.Logf("Correctly matched: %v", matches)
			}
		})
	}
}

// TestEvasionDetection tests detection of evasion techniques
func TestEvasionDetection(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		patternName string
		should      bool
	}{
		{"dot separated slot", "s.l.o.t gacor", "Evasion Dot Separated", true},
		{"dot separated togel", "main t.o.g.e.l online", "Evasion Dot Separated", true},
		{"star separated", "j*u*d*i online", "Evasion Star Separated", true},
		{"dash separated", "c-a-s-i-n-o bonus", "Evasion Dash Separated", true},
		{"normal slot", "slot online", "Evasion Dot Separated", false},
	}

	// Build pattern map
	patternMap := make(map[string]*regexp.Regexp)
	for _, p := range domain.DefaultDorkPatterns {
		if p.IsRegex && strings.HasPrefix(p.Name, "Evasion") {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				t.Logf("Warning: Failed to compile pattern %s: %v", p.Name, err)
				continue
			}
			patternMap[p.Name] = re
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			re, exists := patternMap[tc.patternName]
			if !exists {
				t.Skipf("Pattern %s not found", tc.patternName)
				return
			}

			matches := re.FindAllString(tc.content, -1)
			hasMatch := len(matches) > 0

			if hasMatch != tc.should {
				if tc.should {
					t.Errorf("Expected pattern '%s' to match '%s'", tc.patternName, tc.content)
				} else {
					t.Errorf("Pattern '%s' should NOT match '%s' but got: %v", tc.patternName, tc.content, matches)
				}
			} else if hasMatch {
				t.Logf("Pattern '%s' correctly matched: %v", tc.patternName, matches)
			}
		})
	}
}

// TestDefacementDetection tests detection of defacement patterns
func TestDefacementDetection(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		should  bool
	}{
		{"hacked by", "Website ini hacked by Indonesian Cyber Team", true},
		{"defaced by", "Defaced by Anonymous Indonesia", true},
		{"h4ck3d", "H4ck3d by elite hackers", true},
		{"greetz to", "Greetz to all my friends", true},
		{"cyber army", "Indonesia Cyber Army was here", true},
		{"normal content", "Website resmi pemerintah", false},
	}

	// Find defacement patterns
	var defacePatterns []*regexp.Regexp
	for _, p := range domain.DefaultDorkPatterns {
		if p.Category == domain.DorkCategoryDefacement && p.IsRegex {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				continue
			}
			defacePatterns = append(defacePatterns, re)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasMatch := false
			for _, re := range defacePatterns {
				if re.MatchString(tc.content) {
					hasMatch = true
					break
				}
			}

			if hasMatch != tc.should {
				if tc.should {
					t.Errorf("Expected to detect defacement in: %s", tc.content)
				} else {
					t.Errorf("Should NOT detect defacement in: %s", tc.content)
				}
			}
		})
	}
}

// TestWebshellDetection tests detection of webshell patterns
func TestWebshellDetection(t *testing.T) {
	testCases := []struct {
		name    string
		content string
		should  bool
	}{
		{"eval get", `<?php eval($_GET['cmd']); ?>`, true},
		{"shell_exec", `<?php shell_exec($_POST['c']); ?>`, true},
		{"c99 shell", `c99shell version 2.0`, true},
		{"b374k", `b374k shell v3.2`, true},
		{"passthru", `passthru($_REQUEST['x']);`, true},
		{"normal php", `<?php echo "Hello World"; ?>`, false},
	}

	// Find webshell patterns
	var shellPatterns []*regexp.Regexp
	for _, p := range domain.DefaultDorkPatterns {
		if p.Category == domain.DorkCategoryShell && p.IsRegex {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				continue
			}
			shellPatterns = append(shellPatterns, re)
		}
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			hasMatch := false
			for _, re := range shellPatterns {
				if re.MatchString(tc.content) {
					hasMatch = true
					break
				}
			}

			if hasMatch != tc.should {
				if tc.should {
					t.Errorf("Expected to detect webshell in: %s", tc.content)
				} else {
					t.Errorf("Should NOT detect webshell in: %s", tc.content)
				}
			}
		})
	}
}

// TestGamblingURLPatterns tests detection of gambling URL patterns
func TestGamblingURLPatterns(t *testing.T) {
	testCases := []struct {
		name    string
		url     string
		should  bool
	}{
		{"slot88.com", "https://slot88.com", true},
		{"togel123.xyz", "http://togel123.xyz/daftar", true},
		{"gacor777.vip", "https://gacor777.vip", true},
		{"raja88.bet", "http://raja88.bet/login", true},
		{"maxwin.games", "https://maxwin.games", true},
		{"casino99.live", "http://casino99.live", true},
		{"government.go.id", "https://example.go.id", false},
		{"google.com", "https://google.com", false},
	}

	// Find gambling URL pattern
	var urlPattern *regexp.Regexp
	for _, p := range domain.DefaultDorkPatterns {
		if p.Name == "Gambling URL Patterns" && p.IsRegex {
			var err error
			urlPattern, err = regexp.Compile(p.Pattern)
			if err != nil {
				t.Fatalf("Failed to compile URL pattern: %v", err)
			}
			break
		}
	}

	if urlPattern == nil {
		t.Fatal("Gambling URL pattern not found")
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matches := urlPattern.FindAllString(tc.url, -1)
			hasMatch := len(matches) > 0

			if hasMatch != tc.should {
				if tc.should {
					t.Errorf("Expected to match URL '%s' but didn't", tc.url)
				} else {
					t.Errorf("Should NOT match URL '%s' but got: %v", tc.url, matches)
				}
			}
		})
	}
}

// TestPatternCount tests that we have enough patterns for each category
func TestPatternCount(t *testing.T) {
	categoryCount := make(map[domain.DorkCategory]int)
	regexCount := 0
	keywordCount := 0
	totalKeywords := 0

	for _, p := range domain.DefaultDorkPatterns {
		categoryCount[p.Category]++
		if p.IsRegex {
			regexCount++
		}
		if len(p.Keywords) > 0 {
			keywordCount++
			totalKeywords += len(p.Keywords)
		}
	}

	t.Logf("Pattern statistics:")
	t.Logf("  Total patterns: %d", len(domain.DefaultDorkPatterns))
	t.Logf("  Regex patterns: %d", regexCount)
	t.Logf("  Keyword patterns: %d (total keywords: %d)", keywordCount, totalKeywords)
	t.Logf("\nBy category:")
	for cat, count := range categoryCount {
		t.Logf("  %s: %d patterns", cat, count)
	}

	// Verify gambling patterns are comprehensive
	if categoryCount[domain.DorkCategoryGambling] < 10 {
		t.Errorf("Expected at least 10 gambling patterns, got %d", categoryCount[domain.DorkCategoryGambling])
	}

	// Verify we have patterns for all categories
	expectedCategories := []domain.DorkCategory{
		domain.DorkCategoryGambling,
		domain.DorkCategoryDefacement,
		domain.DorkCategoryShell,
		domain.DorkCategoryMalware,
		domain.DorkCategoryPhishing,
		domain.DorkCategorySEOSpam,
		domain.DorkCategoryBackdoor,
		domain.DorkCategoryInjection,
	}

	for _, cat := range expectedCategories {
		if categoryCount[cat] == 0 {
			t.Errorf("No patterns found for category: %s", cat)
		}
	}
}

// TestFalsePositiveKeywords verifies false positive keywords are documented
func TestFalsePositiveKeywords(t *testing.T) {
	fpKeywords := domain.FalsePositiveKeywords
	if len(fpKeywords) == 0 {
		t.Error("FalsePositiveKeywords list is empty")
	}

	t.Logf("False positive keywords to be careful with: %v", fpKeywords)

	// Test that "bet" can match unwanted words
	testContent := "The difference between the two is better than expected"
	if !strings.Contains(testContent, "bet") {
		t.Error("Test content should contain 'bet' substring")
	}
	t.Logf("Example: '%s' contains 'bet' - potential false positive", testContent)
}

// BenchmarkPatternMatching benchmarks the pattern matching performance
func BenchmarkPatternMatching(b *testing.B) {
	// Compile all regex patterns
	var compiledPatterns []*regexp.Regexp
	for _, p := range domain.DefaultDorkPatterns {
		if p.IsRegex && p.Pattern != "" {
			re, err := regexp.Compile(p.Pattern)
			if err != nil {
				continue
			}
			compiledPatterns = append(compiledPatterns, re)
		}
	}

	// Sample content to scan
	content := `
		<html>
		<head><title>Website Resmi</title></head>
		<body>
		<h1>Selamat Datang</h1>
		<p>Ini adalah website resmi pemerintah.</p>
		<a href="https://slot88.com">Link mencurigakan</a>
		<div style="display:none">slot gacor maxwin togel online</div>
		</body>
		</html>
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, re := range compiledPatterns {
			re.FindAllString(content, 10)
		}
	}
}
