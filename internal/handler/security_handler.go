package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
	"github.com/diskominfos-bali/monitoring-website/pkg/logger"
)

type SecurityHandler struct {
	securityChecker *monitor.SecurityHeadersChecker
	checkRepo       *mysql.CheckRepository
	websiteRepo     *mysql.WebsiteRepository
}

func NewSecurityHandler(
	securityChecker *monitor.SecurityHeadersChecker,
	checkRepo *mysql.CheckRepository,
	websiteRepo *mysql.WebsiteRepository,
) *SecurityHandler {
	return &SecurityHandler{
		securityChecker: securityChecker,
		checkRepo:       checkRepo,
		websiteRepo:     websiteRepo,
	}
}

// GetSecurityStats returns overall security statistics
// @Summary Get security statistics
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.SecurityStats
// @Router /api/security/stats [get]
func (h *SecurityHandler) GetSecurityStats(c *gin.Context) {
	stats, err := h.checkRepo.GetSecurityStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil statistik keamanan",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// GetWebsiteSecurity returns security check result for a website
// @Summary Get website security check result
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} domain.SecurityHeaderCheck
// @Router /api/security/websites/{id} [get]
func (h *SecurityHandler) GetWebsiteSecurity(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	check, err := h.checkRepo.GetLatestSecurityHeaderCheck(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data keamanan",
		})
		return
	}

	if check == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Data keamanan tidak ditemukan. Jalankan pemeriksaan keamanan terlebih dahulu.",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": check})
}

// GetWebsiteSecurityHistory returns security check history for a website
// @Summary Get website security check history
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Param limit query int false "Number of items (default: 30)"
// @Success 200 {array} domain.SecurityHeaderCheck
// @Router /api/security/websites/{id}/history [get]
func (h *SecurityHandler) GetWebsiteSecurityHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	limit := 30
	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	checks, err := h.checkRepo.GetSecurityHeaderHistory(c.Request.Context(), id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil riwayat keamanan",
		})
		return
	}

	c.JSON(http.StatusOK, checks)
}

// TriggerSecurityCheck triggers a security check for a specific website
// @Summary Trigger security check for a website
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Param id path int true "Website ID"
// @Success 200 {object} domain.SecurityHeaderCheck
// @Router /api/security/websites/{id}/check [post]
func (h *SecurityHandler) TriggerSecurityCheck(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Get website
	website, err := h.websiteRepo.GetByID(c.Request.Context(), id)
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

	// Run security check
	check, err := h.securityChecker.CheckWebsite(c.Request.Context(), website)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal melakukan pemeriksaan keamanan: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, check)
}

// TriggerAllSecurityChecks triggers security check for all websites
// @Summary Trigger security check for all websites
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/security/check-all [post]
func (h *SecurityHandler) TriggerAllSecurityChecks(c *gin.Context) {
	go func() {
		ctx := context.Background()
		if err := h.securityChecker.CheckAllWebsites(ctx); err != nil {
			logger.Error().Err(err).Msg("Security check-all failed")
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"message": "Pemeriksaan keamanan dimulai untuk semua website",
	})
}

// GetSecuritySummary returns security summary for all websites
// @Summary Get security summary for all websites
// @Tags Security
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/security/summary [get]
func (h *SecurityHandler) GetSecuritySummary(c *gin.Context) {
	// Get all websites with security scores
	websites, _, err := h.websiteRepo.GetAll(c.Request.Context(), domain.WebsiteFilter{
		Limit: 100,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data website",
		})
		return
	}

	var summaries []gin.H
	for _, w := range websites {
		check, _ := h.checkRepo.GetLatestSecurityHeaderCheck(c.Request.Context(), w.ID)

		summary := gin.H{
			"id":             w.ID,
			"name":           w.Name,
			"url":            w.URL,
			"ssl_valid":      w.SSLValid.Bool && w.SSLValid.Valid,
			"security_score": 0,
			"grade":          "F",
		}

		// Calculate days until SSL expiry
		if w.SSLExpiryDate.Valid {
			daysUntil := int(w.SSLExpiryDate.Time.Sub(time.Now()).Hours() / 24)
			summary["ssl_days_until_expiry"] = daysUntil
		}

		if check != nil {
			summary["security_score"] = check.Score
			summary["grade"] = check.Grade
			summary["last_checked_at"] = check.CheckedAt
		}

		summaries = append(summaries, summary)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summaries,
	})
}
