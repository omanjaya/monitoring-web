package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	ai "github.com/diskominfos-bali/monitoring-website/internal/service/ai"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type DorkHandler struct {
	scanner      *monitor.DorkScanner
	dorkRepo     *mysql.DorkRepository
	settingsRepo *mysql.SettingsRepository
}

func NewDorkHandler(scanner *monitor.DorkScanner, dorkRepo *mysql.DorkRepository, settingsRepo *mysql.SettingsRepository) *DorkHandler {
	return &DorkHandler{
		scanner:      scanner,
		dorkRepo:     dorkRepo,
		settingsRepo: settingsRepo,
	}
}

// GetOverallStats returns overall dork detection statistics
func (h *DorkHandler) GetOverallStats(c *gin.Context) {
	stats, err := h.dorkRepo.GetOverallStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}

	// Include AI verification status
	aiStatus := gin.H{"enabled": false, "provider": "", "model": ""}
	if h.settingsRepo != nil {
		if aiSettings, err := h.settingsRepo.GetAISettings(c.Request.Context()); err == nil {
			aiStatus["enabled"] = aiSettings.Enabled
			aiStatus["provider"] = aiSettings.Provider
			aiStatus["model"] = aiSettings.Model
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": stats, "ai_status": aiStatus})
}

// GetWebsiteStats returns dork detection statistics for a specific website
func (h *DorkHandler) GetWebsiteStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid website ID"})
		return
	}

	stats, err := h.dorkRepo.GetDetectionStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get statistics"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ListPatterns returns all dork patterns
func (h *DorkHandler) ListPatterns(c *gin.Context) {
	filter := domain.DorkPatternFilter{}

	if category := c.Query("category"); category != "" {
		filter.Category = domain.DorkCategory(category)
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = domain.DorkSeverity(severity)
	}
	if active := c.Query("is_active"); active != "" {
		isActive := active == "true"
		filter.IsActive = &isActive
	}

	patterns, err := h.dorkRepo.ListPatterns(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list patterns"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":  patterns,
		"total": len(patterns),
	})
}

// CreatePattern creates a new custom dork pattern
func (h *DorkHandler) CreatePattern(c *gin.Context) {
	var req struct {
		Name        string              `json:"name" binding:"required"`
		Category    domain.DorkCategory `json:"category" binding:"required"`
		Pattern     string              `json:"pattern" binding:"required"`
		PatternType string              `json:"pattern_type"`
		Severity    domain.DorkSeverity `json:"severity"`
		Description string              `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pattern := &domain.DorkPattern{
		Name:        req.Name,
		Category:    req.Category,
		Pattern:     req.Pattern,
		PatternType: req.PatternType,
		Severity:    req.Severity,
		Description: req.Description,
		IsActive:    true,
		IsDefault:   false,
	}

	if pattern.PatternType == "" {
		pattern.PatternType = "regex"
	}
	if pattern.Severity == "" {
		pattern.Severity = domain.DorkSeverityMedium
	}

	if err := h.dorkRepo.CreatePattern(c.Request.Context(), pattern); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pattern"})
		return
	}

	// Reload patterns in scanner
	h.scanner.ReloadPatterns(c.Request.Context())

	c.JSON(http.StatusCreated, pattern)
}

// UpdatePattern updates an existing pattern
func (h *DorkHandler) UpdatePattern(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pattern ID"})
		return
	}

	var req struct {
		Name        string              `json:"name"`
		Category    domain.DorkCategory `json:"category"`
		Pattern     string              `json:"pattern"`
		PatternType string              `json:"pattern_type"`
		Severity    domain.DorkSeverity `json:"severity"`
		Description string              `json:"description"`
		IsActive    *bool               `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.dorkRepo.GetPattern(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pattern not found"})
		return
	}

	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Category != "" {
		existing.Category = req.Category
	}
	if req.Pattern != "" {
		existing.Pattern = req.Pattern
	}
	if req.PatternType != "" {
		existing.PatternType = req.PatternType
	}
	if req.Severity != "" {
		existing.Severity = req.Severity
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.dorkRepo.UpdatePattern(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pattern"})
		return
	}

	// Reload patterns in scanner
	h.scanner.ReloadPatterns(c.Request.Context())

	c.JSON(http.StatusOK, existing)
}

// DeletePattern deletes a custom pattern (not default patterns)
func (h *DorkHandler) DeletePattern(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid pattern ID"})
		return
	}

	existing, err := h.dorkRepo.GetPattern(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pattern not found"})
		return
	}

	if existing.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete default patterns"})
		return
	}

	if err := h.dorkRepo.DeletePattern(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete pattern"})
		return
	}

	// Reload patterns in scanner
	h.scanner.ReloadPatterns(c.Request.Context())

	c.JSON(http.StatusOK, gin.H{"message": "Pattern deleted"})
}

// ListDetections returns detections with filters
func (h *DorkHandler) ListDetections(c *gin.Context) {
	filter := domain.DorkDetectionFilter{
		Limit:  50,
		Offset: 0,
	}

	if websiteID := c.Query("website_id"); websiteID != "" {
		id, _ := strconv.ParseInt(websiteID, 10, 64)
		filter.WebsiteID = id
	}
	if category := c.Query("category"); category != "" {
		filter.Category = domain.DorkCategory(category)
	}
	if severity := c.Query("severity"); severity != "" {
		filter.Severity = domain.DorkSeverity(severity)
	}
	if resolved := c.Query("is_resolved"); resolved != "" {
		isResolved := resolved == "true"
		filter.IsResolved = &isResolved
	}
	if limit := c.Query("limit"); limit != "" {
		l, _ := strconv.Atoi(limit)
		if l > 0 && l <= 100 {
			filter.Limit = l
		}
	}
	if offset := c.Query("offset"); offset != "" {
		o, _ := strconv.Atoi(offset)
		if o >= 0 {
			filter.Offset = o
		}
	}

	detections, total, err := h.dorkRepo.ListDetections(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list detections"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":   detections,
		"total":  total,
		"limit":  filter.Limit,
		"offset": filter.Offset,
	})
}

// GetDetection returns a specific detection
func (h *DorkHandler) GetDetection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid detection ID"})
		return
	}

	detection, err := h.dorkRepo.GetDetection(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Detection not found"})
		return
	}
	c.JSON(http.StatusOK, detection)
}

// ResolveDetection marks a detection as resolved
func (h *DorkHandler) ResolveDetection(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid detection ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	// Get user from context
	resolvedBy := "admin"
	if user, exists := c.Get("user"); exists {
		if u, ok := user.(*domain.User); ok {
			resolvedBy = u.Username
		}
	}

	if err := h.dorkRepo.MarkAsResolved(c.Request.Context(), id, resolvedBy, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve detection"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Detection resolved"})
}

// MarkFalsePositive marks a detection as false positive
func (h *DorkHandler) MarkFalsePositive(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid detection ID"})
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&req)

	if err := h.dorkRepo.MarkAsFalsePositive(c.Request.Context(), id, req.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark as false positive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Detection marked as false positive"})
}

// ListScanResults returns scan results for a website
func (h *DorkHandler) ListScanResults(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid website ID"})
		return
	}

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	results, err := h.dorkRepo.ListScanResults(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list scan results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(results),
	})
}

// GetScanResult returns a specific scan result with detections
func (h *DorkHandler) GetScanResult(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid scan result ID"})
		return
	}

	result, err := h.dorkRepo.GetScanResult(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Scan result not found"})
		return
	}

	detections, err := h.dorkRepo.GetDetectionsByScanResult(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get detections"})
		return
	}

	result.Detections = detections

	c.JSON(http.StatusOK, result)
}

// TriggerScan triggers a dork scan for a specific website
func (h *DorkHandler) TriggerScan(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid website ID"})
		return
	}

	var req struct {
		ScanType   string                `json:"scan_type"`
		Categories []domain.DorkCategory `json:"categories"`
	}
	c.ShouldBindJSON(&req)

	if req.ScanType == "" {
		req.ScanType = "quick"
	}

	// Run scan in background
	go func() {
		h.scanner.ScanWebsite(context.Background(), id, req.ScanType, req.Categories)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message":    "Scan started",
		"website_id": id,
		"scan_type":  req.ScanType,
	})
}

// TriggerAllScans triggers dork scans for all websites
func (h *DorkHandler) TriggerAllScans(c *gin.Context) {
	go func() {
		h.scanner.ScanAllWebsites(context.Background())
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Scan started for all websites",
	})
}

// GetWebsiteSettings returns dork settings for a website
func (h *DorkHandler) GetWebsiteSettings(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid website ID"})
		return
	}

	settings, err := h.dorkRepo.GetWebsiteSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateWebsiteSettings updates dork settings for a website
func (h *DorkHandler) UpdateWebsiteSettings(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid website ID"})
		return
	}

	var req struct {
		IsEnabled         *bool                 `json:"is_enabled"`
		ScanFrequency     string                `json:"scan_frequency"`
		ScanDepth         int                   `json:"scan_depth"`
		MaxPages          int                   `json:"max_pages"`
		CategoriesEnabled []domain.DorkCategory `json:"categories_enabled"`
		ExcludedPaths     []string              `json:"excluded_paths"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settings, err := h.dorkRepo.GetWebsiteSettings(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get settings"})
		return
	}

	if req.IsEnabled != nil {
		settings.IsEnabled = *req.IsEnabled
	}
	if req.ScanFrequency != "" {
		settings.ScanFrequency = req.ScanFrequency
	}
	if req.ScanDepth > 0 {
		settings.ScanDepth = req.ScanDepth
	}
	if req.MaxPages > 0 {
		settings.MaxPages = req.MaxPages
	}
	if req.CategoriesEnabled != nil {
		settings.CategoriesEnabled = req.CategoriesEnabled
	}
	if req.ExcludedPaths != nil {
		settings.ExcludedPaths = req.ExcludedPaths
	}

	if err := h.dorkRepo.SaveWebsiteSettings(c.Request.Context(), settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// ClearAllDetections removes all detections and scan results
func (h *DorkHandler) ClearAllDetections(c *gin.Context) {
	deleted, err := h.dorkRepo.ClearAllDetections(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear detections: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "All detections cleared",
		"deleted": deleted,
	})
}

// VerifyAllWithAI runs AI verification on all unverified detections
func (h *DorkHandler) VerifyAllWithAI(c *gin.Context) {
	ctx := c.Request.Context()

	// Load AI settings from DB
	aiSettings, err := h.settingsRepo.GetAISettings(ctx)
	if err != nil || !aiSettings.Enabled || aiSettings.APIKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "AI verification belum dikonfigurasi. Aktifkan di Pengaturan → AI Verification."})
		return
	}

	// Get unverified detections
	detections, err := h.dorkRepo.GetUnverifiedDetections(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil deteksi"})
		return
	}

	if len(detections) == 0 {
		c.JSON(http.StatusOK, gin.H{"message": "Tidak ada deteksi yang perlu diverifikasi", "verified": 0, "false_positives": 0})
		return
	}

	// Create AI verifier
	aiCfg := &config.AIConfig{
		Enabled:  aiSettings.Enabled,
		Provider: aiSettings.Provider,
		APIKey:   aiSettings.APIKey,
		Model:    aiSettings.Model,
	}
	verifier := ai.NewVerifier(aiCfg)

	// Group detections by website for better AI context
	type websiteGroup struct {
		name       string
		url        string
		detections []domain.DorkDetection
	}
	groups := make(map[int64]*websiteGroup)
	for _, d := range detections {
		g, ok := groups[d.WebsiteID]
		if !ok {
			g = &websiteGroup{name: d.WebsiteName, url: d.WebsiteURL}
			groups[d.WebsiteID] = g
		}
		g.detections = append(g.detections, d)
	}

	totalVerified := 0
	totalFalsePositives := 0

	for _, group := range groups {
		// Convert to AI detection format
		aiDetections := make([]ai.Detection, len(group.detections))
		for i, d := range group.detections {
			aiDetections[i] = ai.Detection{
				PatternName:    d.PatternName,
				Category:       string(d.Category),
				MatchedContent: d.MatchedContent,
				Context:        d.Context,
				URL:            d.URL,
				Confidence:     d.Confidence,
			}
		}

		// Process in batches of 20
		const batchSize = 20
		for start := 0; start < len(aiDetections); start += batchSize {
			end := start + batchSize
			if end > len(aiDetections) {
				end = len(aiDetections)
			}

			results, err := verifier.VerifyDetections(ctx, group.name, group.url, aiDetections[start:end])
			if err != nil {
				logger.Warn().Err(err).Str("website", group.name).Msg("AI verification failed for batch")
				// Mark batch as verified even if AI fails (so we don't retry endlessly)
				for i := start; i < end; i++ {
					h.dorkRepo.MarkDetectionAIVerified(ctx, group.detections[i].ID)
				}
				totalVerified += end - start
				continue
			}

			falsePositiveIdx := make(map[int]string)
			for _, r := range results {
				if r.IsFalsePositive && r.Confidence >= 0.7 {
					falsePositiveIdx[r.Index] = r.Reason
				}
			}

			for i := start; i < end; i++ {
				localIdx := i - start
				if reason, isFP := falsePositiveIdx[localIdx]; isFP {
					h.dorkRepo.MarkAsFalsePositiveByAI(ctx, group.detections[i].ID, reason)
					totalFalsePositives++
				} else {
					h.dorkRepo.MarkDetectionAIVerified(ctx, group.detections[i].ID)
				}
				totalVerified++
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "AI verification selesai",
		"verified":        totalVerified,
		"false_positives": totalFalsePositives,
	})
}

// GetCategories returns all available dork categories
func (h *DorkHandler) GetCategories(c *gin.Context) {
	categories := []gin.H{
		{"id": "gambling", "name": "Gambling (Judol)", "description": "Deteksi konten judi online, slot, togel"},
		{"id": "defacement", "name": "Defacement", "description": "Deteksi website yang di-deface/hack"},
		{"id": "malware", "name": "Malware", "description": "Deteksi script malware dan cryptominer"},
		{"id": "phishing", "name": "Phishing", "description": "Deteksi halaman phishing"},
		{"id": "seo_spam", "name": "SEO Spam", "description": "Deteksi spam SEO dan doorway pages"},
		{"id": "webshell", "name": "Webshell", "description": "Deteksi webshell (c99, r57, b374k)"},
		{"id": "backdoor", "name": "Backdoor", "description": "Deteksi backdoor PHP"},
		{"id": "injection", "name": "Injection", "description": "Deteksi SQL injection dan XSS"},
	}
	c.JSON(http.StatusOK, gin.H{"data": categories})
}
