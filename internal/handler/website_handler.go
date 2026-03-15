package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
	"github.com/diskominfos-bali/monitoring-website/internal/service/website"
)

type WebsiteHandler struct {
	websiteService *website.Service
	dnsScanner     *monitor.DNSScanner
	dnsRepo        *mysql.DNSRepository
}

func NewWebsiteHandler(websiteService *website.Service, dnsScanner *monitor.DNSScanner, dnsRepo *mysql.DNSRepository) *WebsiteHandler {
	return &WebsiteHandler{
		websiteService: websiteService,
		dnsScanner:     dnsScanner,
		dnsRepo:        dnsRepo,
	}
}

// ListWebsites retrieves all websites with filtering
// @Summary List websites
// @Tags Websites
// @Security BearerAuth
// @Produce json
// @Param status query string false "Filter by status"
// @Param opd_id query int false "Filter by OPD ID"
// @Param search query string false "Search by name or URL"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/websites [get]
func (h *WebsiteHandler) ListWebsites(c *gin.Context) {
	var filter domain.WebsiteFilter

	if status := c.Query("status"); status != "" {
		s := domain.WebsiteStatus(status)
		filter.Status = &s
	}
	if opdID := c.Query("opd_id"); opdID != "" {
		id, _ := strconv.ParseInt(opdID, 10, 64)
		filter.OPDID = &id
	}
	if search := c.Query("search"); search != "" {
		filter.Search = search
	}
	if page := c.Query("page"); page != "" {
		filter.Page, _ = strconv.Atoi(page)
	}
	if limit := c.Query("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}

	websites, total, err := h.websiteService.ListWebsites(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data website",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  websites,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

// GetWebsite retrieves a single website
// @Summary Get website detail
// @Tags Websites
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} domain.Website
// @Router /api/websites/{id} [get]
func (h *WebsiteHandler) GetWebsite(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	website, err := h.websiteService.GetWebsite(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data website",
		})
		return
	}
	if website == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Website tidak ditemukan",
		})
		return
	}

	// Get additional data
	uptimeStats, _ := h.websiteService.GetUptimeStats(c.Request.Context(), id, 24)
	recentChecks, _ := h.websiteService.GetRecentChecks(c.Request.Context(), id, 10)
	sslCheck, _ := h.websiteService.GetLatestSSLCheck(c.Request.Context(), id)
	contentScan, _ := h.websiteService.GetLatestContentScan(c.Request.Context(), id)

	c.JSON(http.StatusOK, gin.H{
		"website":       website,
		"uptime_stats":  uptimeStats,
		"recent_checks": recentChecks,
		"ssl_check":     sslCheck,
		"content_scan":  contentScan,
	})
}

// CreateWebsite creates a new website
// @Summary Create website
// @Tags Websites
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.WebsiteCreate true "Website data"
// @Success 201 {object} domain.Website
// @Router /api/websites [post]
func (h *WebsiteHandler) CreateWebsite(c *gin.Context) {
	var input domain.WebsiteCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	website, err := h.websiteService.CreateWebsite(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, website)
}

// UpdateWebsite updates a website
// @Summary Update website
// @Tags Websites
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Website ID"
// @Param input body domain.WebsiteUpdate true "Website data"
// @Success 200 {object} domain.Website
// @Router /api/websites/{id} [put]
func (h *WebsiteHandler) UpdateWebsite(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	var input domain.WebsiteUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	website, err := h.websiteService.UpdateWebsite(c.Request.Context(), id, &input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, website)
}

// DeleteWebsite deletes a website
// @Summary Delete website
// @Tags Websites
// @Security BearerAuth
// @Param id path int true "Website ID"
// @Success 200 {object} map[string]string
// @Router /api/websites/{id} [delete]
func (h *WebsiteHandler) DeleteWebsite(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	if err := h.websiteService.DeleteWebsite(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Website berhasil dihapus",
	})
}

// BulkImport imports multiple websites
// @Summary Bulk import websites
// @Tags Websites
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body []domain.WebsiteCreate true "List of websites"
// @Success 200 {object} website.BulkImportResult
// @Router /api/websites/bulk [post]
func (h *WebsiteHandler) BulkImport(c *gin.Context) {
	var input []domain.WebsiteCreate
	if err := json.NewDecoder(c.Request.Body).Decode(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	if len(input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data kosong",
		})
		return
	}

	result, err := h.websiteService.BulkImportWebsites(c.Request.Context(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengimport website",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// BulkAction performs bulk actions on multiple websites
// @Summary Bulk action on websites
// @Tags Websites
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.BulkWebsiteAction true "Bulk action data"
// @Success 200 {object} map[string]interface{}
// @Router /api/websites/bulk-action [post]
func (h *WebsiteHandler) BulkAction(c *gin.Context) {
	var input domain.BulkWebsiteAction
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	affected, err := h.websiteService.BulkAction(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal melakukan bulk action: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":    gin.H{"affected": affected},
		"message": fmt.Sprintf("Bulk %s berhasil untuk %d website", input.Action, affected),
	})
}

// GetUptimeHistory retrieves uptime history for a website with chart data
// @Summary Get uptime history with chart data
// @Tags Websites
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Param hours query int false "Hours to look back (default: 24, max: 720 for 30 days)"
// @Success 200 {object} map[string]interface{}
// @Router /api/websites/{id}/uptime [get]
func (h *WebsiteHandler) GetUptimeHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	hours := 24
	if hr := c.Query("hours"); hr != "" {
		hours, _ = strconv.Atoi(hr)
	}

	// Limit to max 30 days
	if hours > 720 {
		hours = 720
	}
	if hours < 1 {
		hours = 24
	}

	// Get stats summary
	stats, err := h.websiteService.GetUptimeStats(c.Request.Context(), id, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data uptime",
		})
		return
	}

	// Get chart data
	chartData, err := h.websiteService.GetUptimeChartData(c.Request.Context(), id, hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data chart",
		})
		return
	}

	// Get recent checks for the detail table
	recentChecks, _ := h.websiteService.GetRecentChecks(c.Request.Context(), id, 50)

	c.JSON(http.StatusOK, gin.H{
		"stats":         stats,
		"chart_data":    chartData,
		"recent_checks": recentChecks,
	})
}

// GetWebsiteMetrics retrieves response time percentile metrics for a website
// @Summary Get website performance metrics (percentiles)
// @Tags Websites
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} domain.WebsiteMetrics
// @Router /api/websites/{id}/metrics [get]
func (h *WebsiteHandler) GetWebsiteMetrics(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	metrics, err := h.websiteService.GetWebsiteMetrics(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data metrik: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": metrics,
	})
}

// ListOPDs retrieves all OPDs
// @Summary List OPDs
// @Tags OPD
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.OPD
// @Router /api/opd [get]
func (h *WebsiteHandler) ListOPDs(c *gin.Context) {
	opds, err := h.websiteService.ListOPDs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data OPD",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": opds})
}

// CreateOPD creates a new OPD
// @Summary Create OPD
// @Tags OPD
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.OPD true "OPD data"
// @Success 201 {object} domain.OPD
// @Router /api/opd [post]
func (h *WebsiteHandler) CreateOPD(c *gin.Context) {
	var input domain.OPD
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	opd, err := h.websiteService.CreateOPD(c.Request.Context(), &input)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": opd})
}

// BulkImportOPD imports multiple OPDs at once
// @Summary Bulk import OPDs
// @Tags OPD
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body []domain.OPD true "List of OPDs"
// @Success 200 {object} map[string]interface{}
// @Router /api/opd/bulk [post]
func (h *WebsiteHandler) BulkImportOPD(c *gin.Context) {
	var input []domain.OPD
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	if len(input) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data kosong",
		})
		return
	}

	type BulkError struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}

	var created []string
	var skipped []string
	var failed []BulkError

	// Get existing OPDs to check duplicates by code
	existingOPDs, _ := h.websiteService.ListOPDs(c.Request.Context())
	existingCodes := make(map[string]bool)
	for _, opd := range existingOPDs {
		existingCodes[opd.Code] = true
	}

	for _, opd := range input {
		if opd.Name == "" || opd.Code == "" {
			failed = append(failed, BulkError{Name: opd.Name, Error: "nama dan kode harus diisi"})
			continue
		}

		opd.Code = strings.ToUpper(opd.Code)

		if existingCodes[opd.Code] {
			skipped = append(skipped, opd.Name)
			continue
		}

		_, err := h.websiteService.CreateOPD(c.Request.Context(), &opd)
		if err != nil {
			failed = append(failed, BulkError{Name: opd.Name, Error: err.Error()})
			continue
		}

		created = append(created, opd.Name)
		existingCodes[opd.Code] = true
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"created": created,
			"skipped": skipped,
			"failed":  failed,
		},
	})
}

// GetDNSScan performs a DNS scan for a website and returns the results
// @Summary DNS scan for a website
// @Tags Websites
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} domain.DNSScanResult
// @Router /api/websites/{id}/dns [get]
func (h *WebsiteHandler) GetDNSScan(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID tidak valid"})
		return
	}

	w, err := h.websiteService.GetWebsite(c.Request.Context(), id)
	if err != nil || w == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Website tidak ditemukan"})
		return
	}

	result, err := h.dnsScanner.ScanDNS(c.Request.Context(), w)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DNS scan gagal: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

// GetDNSSummary returns the latest DNS scan for all domains
// @Summary Get DNS summary for all domains
// @Tags DNS
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/dns/summary [get]
func (h *WebsiteHandler) GetDNSSummary(c *gin.Context) {
	scans, err := h.dnsRepo.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data DNS: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": scans})
}
