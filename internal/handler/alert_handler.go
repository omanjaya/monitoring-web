package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/handler/middleware"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
)

type AlertHandler struct {
	alertRepo *mysql.AlertRepository
}

func NewAlertHandler(alertRepo *mysql.AlertRepository) *AlertHandler {
	return &AlertHandler{
		alertRepo: alertRepo,
	}
}

// ListAlerts retrieves all alerts with filtering
// @Summary List alerts
// @Tags Alerts
// @Security BearerAuth
// @Produce json
// @Param website_id query int false "Filter by website ID"
// @Param type query string false "Filter by type"
// @Param severity query string false "Filter by severity"
// @Param is_resolved query bool false "Filter by resolved status"
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/alerts [get]
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	var filter domain.AlertFilter

	if websiteID := c.Query("website_id"); websiteID != "" {
		id, _ := strconv.ParseInt(websiteID, 10, 64)
		filter.WebsiteID = &id
	}
	if alertType := c.Query("type"); alertType != "" {
		t := domain.AlertType(alertType)
		filter.Type = &t
	}
	if severity := c.Query("severity"); severity != "" {
		s := domain.AlertSeverity(severity)
		filter.Severity = &s
	}
	if isResolved := c.Query("is_resolved"); isResolved != "" {
		resolved := isResolved == "true"
		filter.IsResolved = &resolved
	}
	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			filter.StartDate = &t
		}
	}
	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			t = t.Add(24*time.Hour - time.Second) // End of day
			filter.EndDate = &t
		}
	}
	if page := c.Query("page"); page != "" {
		filter.Page, _ = strconv.Atoi(page)
	}
	if limit := c.Query("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}

	alerts, total, err := h.alertRepo.GetAll(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  alerts,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

// GetActiveAlerts retrieves active (unresolved) alerts
// @Summary Get active alerts
// @Tags Alerts
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.Alert
// @Router /api/alerts/active [get]
func (h *AlertHandler) GetActiveAlerts(c *gin.Context) {
	alerts, err := h.alertRepo.GetActiveAlerts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data alert aktif",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alerts})
}

// GetAlertSummary retrieves alert summary
// @Summary Get alert summary
// @Tags Alerts
// @Security BearerAuth
// @Produce json
// @Success 200 {object} domain.AlertSummary
// @Router /api/alerts/summary [get]
func (h *AlertHandler) GetAlertSummary(c *gin.Context) {
	summary, err := h.alertRepo.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil summary alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": summary,
	})
}

// GetAlert retrieves a single alert
// @Summary Get alert detail
// @Tags Alerts
// @Security BearerAuth
// @Produce json
// @Param id path int true "Alert ID"
// @Success 200 {object} domain.Alert
// @Router /api/alerts/{id} [get]
func (h *AlertHandler) GetAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	alert, err := h.alertRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data alert",
		})
		return
	}
	if alert == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Alert tidak ditemukan",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": alert})
}

// AcknowledgeAlert acknowledges an alert
// @Summary Acknowledge alert
// @Tags Alerts
// @Security BearerAuth
// @Param id path int true "Alert ID"
// @Success 200 {object} map[string]string
// @Router /api/alerts/{id}/acknowledge [post]
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	userID := middleware.GetUserID(c)

	if err := h.alertRepo.Acknowledge(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal acknowledge alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert berhasil di-acknowledge",
	})
}

type BulkResolveInput struct {
	IDs  []int64 `json:"ids"  binding:"required,min=1"`
	Note string  `json:"note"`
}

type ResolveAlertInput struct {
	Note string `json:"note"`
}

// ResolveAlert resolves an alert
// @Summary Resolve alert
// @Tags Alerts
// @Security BearerAuth
// @Accept json
// @Param id path int true "Alert ID"
// @Param input body ResolveAlertInput true "Resolution note"
// @Success 200 {object} map[string]string
// @Router /api/alerts/{id}/resolve [post]
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	var input ResolveAlertInput
	c.ShouldBindJSON(&input)

	userID := middleware.GetUserID(c)

	if err := h.alertRepo.Resolve(c.Request.Context(), id, userID, input.Note); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal resolve alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Alert berhasil di-resolve",
	})
}

// BulkResolveAlerts resolves multiple alerts in one request
// @Summary Bulk resolve alerts
// @Tags Alerts
// @Security BearerAuth
// @Accept json
// @Param input body BulkResolveInput true "List of alert IDs and optional note"
// @Success 200 {object} map[string]interface{}
// @Router /api/alerts/bulk-resolve [post]
func (h *AlertHandler) BulkResolveAlerts(c *gin.Context) {
	var input BulkResolveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	userID := middleware.GetUserID(c)

	affected, err := h.alertRepo.BulkResolve(c.Request.Context(), input.IDs, userID, input.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal bulk resolve alert",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "Alert berhasil di-resolve",
		"affected": affected,
	})
}
