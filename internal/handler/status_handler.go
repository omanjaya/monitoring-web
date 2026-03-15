package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/config"
	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
)

type StatusHandler struct {
	cfg             *config.Config
	websiteRepo     *mysql.WebsiteRepository
	checkRepo       *mysql.CheckRepository
	alertRepo       *mysql.AlertRepository
	maintenanceRepo *mysql.MaintenanceRepository
}

func NewStatusHandler(
	cfg *config.Config,
	websiteRepo *mysql.WebsiteRepository,
	checkRepo *mysql.CheckRepository,
	alertRepo *mysql.AlertRepository,
	maintenanceRepo *mysql.MaintenanceRepository,
) *StatusHandler {
	return &StatusHandler{
		cfg:             cfg,
		websiteRepo:     websiteRepo,
		checkRepo:       checkRepo,
		alertRepo:       alertRepo,
		maintenanceRepo: maintenanceRepo,
	}
}

// GetStatusOverview returns the public status page overview
// @Summary Get public status overview
// @Tags Public Status
// @Produce json
// @Success 200 {object} domain.PublicStatusOverview
// @Router /status [get]
func (h *StatusHandler) GetStatusOverview(c *gin.Context) {
	overview, err := h.websiteRepo.GetPublicStatusOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data status",
		})
		return
	}

	// Get all services
	services, err := h.websiteRepo.GetPublicServiceStatuses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data layanan",
		})
		return
	}

	// Get uptime percentage for each service (last 30 days)
	for i := range services {
		uptime, err := h.checkRepo.GetServiceUptimePercentage(c.Request.Context(), services[i].ID, 30)
		if err == nil {
			services[i].UptimePercentage = uptime
		}
	}

	overview.Services = services
	overview.LastUpdated = time.Now()

	c.JSON(http.StatusOK, overview)
}

// GetAllServices returns all services status
// @Summary Get all services status
// @Tags Public Status
// @Produce json
// @Success 200 {array} domain.PublicServiceStatus
// @Router /status/services [get]
func (h *StatusHandler) GetAllServices(c *gin.Context) {
	services, err := h.websiteRepo.GetPublicServiceStatuses(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data layanan",
		})
		return
	}

	// Get uptime percentage for each service
	for i := range services {
		uptime, err := h.checkRepo.GetServiceUptimePercentage(c.Request.Context(), services[i].ID, 30)
		if err == nil {
			services[i].UptimePercentage = uptime
		}
	}

	c.JSON(http.StatusOK, services)
}

// GetServiceStatus returns a single service status
// @Summary Get single service status
// @Tags Public Status
// @Produce json
// @Param id path int true "Service ID"
// @Success 200 {object} domain.PublicServiceStatus
// @Router /status/services/{id} [get]
func (h *StatusHandler) GetServiceStatus(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	service, err := h.websiteRepo.GetPublicServiceStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data layanan",
		})
		return
	}

	if service == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Layanan tidak ditemukan",
		})
		return
	}

	// Get uptime percentage
	uptime, err := h.checkRepo.GetServiceUptimePercentage(c.Request.Context(), id, 30)
	if err == nil {
		service.UptimePercentage = uptime
	}

	c.JSON(http.StatusOK, service)
}

// GetServiceHistory returns uptime history for a service
// @Summary Get service uptime history
// @Tags Public Status
// @Produce json
// @Param id path int true "Service ID"
// @Param days query int false "Number of days (default: 30, max: 90)"
// @Success 200 {object} domain.PublicUptimeHistory
// @Router /status/services/{id}/history [get]
func (h *StatusHandler) GetServiceHistory(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Verify service exists and is active
	service, err := h.websiteRepo.GetPublicServiceStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data layanan",
		})
		return
	}
	if service == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Layanan tidak ditemukan",
		})
		return
	}

	// Parse days parameter
	days := 30
	if d := c.Query("days"); d != "" {
		days, _ = strconv.Atoi(d)
	}
	if days < 1 {
		days = 30
	}
	if days > 90 {
		days = 90
	}

	history, err := h.checkRepo.GetPublicUptimeHistory(c.Request.Context(), id, days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data riwayat",
		})
		return
	}

	history.ServiceName = service.Name

	c.JSON(http.StatusOK, history)
}

// GetRecentIncidents returns recent public incidents
// @Summary Get recent incidents
// @Tags Public Status
// @Produce json
// @Param limit query int false "Number of incidents (default: 10)"
// @Success 200 {array} domain.PublicIncident
// @Router /status/incidents [get]
func (h *StatusHandler) GetRecentIncidents(c *gin.Context) {
	limit := 10
	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if limit < 1 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// Get recent alerts and convert to public incidents
	filter := domain.AlertFilter{
		Limit: limit,
	}
	alerts, _, err := h.alertRepo.GetAll(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data insiden",
		})
		return
	}

	incidents := make([]gin.H, 0, len(alerts))
	for _, alert := range alerts {
		status := "investigating"
		if alert.IsAcknowledged {
			status = "identified"
		}
		if alert.IsResolved {
			status = "resolved"
		}

		impact := "minor"
		switch alert.Severity {
		case "critical":
			impact = "critical"
		case "warning":
			impact = "major"
		}

		// Get website name from relation if available
		serviceName := ""
		if alert.Website != nil {
			serviceName = alert.Website.Name
		}

		incident := gin.H{
			"id":           alert.ID,
			"title":        alert.Title,
			"status":       status,
			"impact":       impact,
			"message":      alert.Message,
			"service_name": serviceName,
			"created_at":   alert.CreatedAt,
		}

		if alert.IsResolved && alert.ResolvedAt.Valid {
			incident["resolved_at"] = alert.ResolvedAt.Time
		}

		incidents = append(incidents, incident)
	}

	c.JSON(http.StatusOK, incidents)
}

// GetStatusSummary returns a simplified status summary (for embedding)
// @Summary Get status summary for embedding
// @Tags Public Status
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /status/summary [get]
func (h *StatusHandler) GetStatusSummary(c *gin.Context) {
	overview, err := h.websiteRepo.GetPublicStatusOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data status",
		})
		return
	}

	// Simple summary for badges/widgets
	c.JSON(http.StatusOK, gin.H{
		"status":        overview.SystemStatus,
		"uptime":        overview.OverallUptime,
		"total":         overview.TotalWebsites,
		"operational":   overview.OperationalCnt,
		"degraded":      overview.DegradedCnt,
		"down":          overview.DownCnt,
		"last_updated":  time.Now(),
	})
}

// GetPublicMaintenance returns scheduled and active maintenance windows for public display
// @Summary Get public maintenance schedule
// @Tags Public Status
// @Produce json
// @Success 200 {array} domain.PublicMaintenanceWindow
// @Router /status/maintenance [get]
func (h *StatusHandler) GetPublicMaintenance(c *gin.Context) {
	maintenance, err := h.maintenanceRepo.GetPublicMaintenance(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusOK, maintenance)
}

// GetStatusBadge returns an SVG badge for embedding
// @Summary Get status badge SVG
// @Tags Public Status
// @Produce image/svg+xml
// @Success 200 {string} string "SVG badge"
// @Router /status/badge [get]
func (h *StatusHandler) GetStatusBadge(c *gin.Context) {
	overview, err := h.websiteRepo.GetPublicStatusOverview(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "Error")
		return
	}

	// Generate SVG badge
	statusText := "Operational"
	statusColor := "#4ade80" // green

	switch overview.SystemStatus {
	case "degraded":
		statusText = "Degraded"
		statusColor = "#fbbf24" // yellow
	case "partial_outage":
		statusText = "Partial Outage"
		statusColor = "#f97316" // orange
	case "major_outage":
		statusText = "Major Outage"
		statusColor = "#ef4444" // red
	}

	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="150" height="20">
		<linearGradient id="b" x2="0" y2="100%">
			<stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
			<stop offset="1" stop-opacity=".1"/>
		</linearGradient>
		<clipPath id="a">
			<rect width="150" height="20" rx="3" fill="#fff"/>
		</clipPath>
		<g clip-path="url(#a)">
			<rect width="50" height="20" fill="#555"/>
			<rect x="50" width="100" height="20" fill="` + statusColor + `"/>
			<rect width="150" height="20" fill="url(#b)"/>
		</g>
		<g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
			<text x="25" y="14">status</text>
			<text x="100" y="14">` + statusText + `</text>
		</g>
	</svg>`

	c.Header("Content-Type", "image/svg+xml")
	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.String(http.StatusOK, svg)
}
