package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/website"
)

type DashboardHandler struct {
	websiteService *website.Service
	alertRepo      *mysql.AlertRepository
	checkRepo      *mysql.CheckRepository
}

func NewDashboardHandler(websiteService *website.Service, alertRepo *mysql.AlertRepository, checkRepo *mysql.CheckRepository) *DashboardHandler {
	return &DashboardHandler{
		websiteService: websiteService,
		alertRepo:      alertRepo,
		checkRepo:      checkRepo,
	}
}

// GetDashboardStats retrieves dashboard statistics
// @Summary Get dashboard statistics
// @Tags Dashboard
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/dashboard/stats [get]
func (h *DashboardHandler) GetDashboardStats(c *gin.Context) {
	// Get website stats
	stats, err := h.websiteService.GetDashboardStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil statistik dashboard",
		})
		return
	}

	// Get alert summary
	alertSummary, err := h.alertRepo.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil summary alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"website_stats":  stats,
			"alert_summary": alertSummary,
		},
	})
}

// GetDashboardOverview retrieves comprehensive dashboard overview
// @Summary Get dashboard overview
// @Tags Dashboard
// @Security BearerAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/dashboard [get]
func (h *DashboardHandler) GetDashboardOverview(c *gin.Context) {
	ctx := c.Request.Context()

	// Get website stats
	stats, err := h.websiteService.GetDashboardStats(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil statistik",
		})
		return
	}

	// Get active alerts for recent_alerts
	activeAlerts, err := h.alertRepo.GetActiveAlerts(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil alert aktif",
		})
		return
	}

	// Limit active alerts to 10 for dashboard
	if len(activeAlerts) > 10 {
		activeAlerts = activeAlerts[:10]
	}

	// Build status distribution map
	statusDistribution := map[string]int{
		"up":       stats.TotalUp,
		"down":     stats.TotalDown,
		"degraded": stats.TotalDegraded,
		"unknown":  stats.TotalWebsites - stats.TotalUp - stats.TotalDown - stats.TotalDegraded,
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"stats":               stats,
			"recent_alerts":       activeAlerts,
			"status_distribution": statusDistribution,
		},
	})
}

// GetDashboardTrends retrieves trend data for dashboard charts
// @Summary Get dashboard trends
// @Tags Dashboard
// @Security BearerAuth
// @Produce json
// @Param days query int false "Number of days for trend data (default: 7)"
// @Success 200 {object} domain.DashboardTrends
// @Router /api/dashboard/trends [get]
func (h *DashboardHandler) GetDashboardTrends(c *gin.Context) {
	days := 7
	if daysParam := c.Query("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 30 {
			days = d
		}
	}

	trends, err := h.checkRepo.GetDashboardTrends(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data trend",
		})
		return
	}

	// Transform to frontend-expected format
	responseTimes := make([]gin.H, 0, len(trends.ResponseTimes))
	for _, rt := range trends.ResponseTimes {
		responseTimes = append(responseTimes, gin.H{
			"date": rt.Date,
			"avg":  rt.AvgResponseTime,
		})
	}

	uptimeHistory := make([]gin.H, 0, len(trends.UptimeTrend))
	for _, ut := range trends.UptimeTrend {
		uptimeHistory = append(uptimeHistory, gin.H{
			"date":   ut.Date,
			"uptime": ut.UptimePercentage,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"response_times": responseTimes,
			"uptime_history": uptimeHistory,
		},
	})
}
