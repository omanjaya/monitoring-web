package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
)

type MaintenanceHandler struct {
	maintenanceRepo *mysql.MaintenanceRepository
}

func NewMaintenanceHandler(maintenanceRepo *mysql.MaintenanceRepository) *MaintenanceHandler {
	return &MaintenanceHandler{
		maintenanceRepo: maintenanceRepo,
	}
}

// ListMaintenance retrieves all maintenance windows
// @Summary List maintenance windows
// @Tags Maintenance
// @Security BearerAuth
// @Produce json
// @Param website_id query int false "Filter by Website ID"
// @Param status query string false "Filter by status"
// @Param include_past query bool false "Include past maintenance"
// @Param page query int false "Page number"
// @Param limit query int false "Items per page"
// @Success 200 {object} map[string]interface{}
// @Router /api/maintenance [get]
func (h *MaintenanceHandler) ListMaintenance(c *gin.Context) {
	var filter domain.MaintenanceFilter

	if websiteID := c.Query("website_id"); websiteID != "" {
		id, _ := strconv.ParseInt(websiteID, 10, 64)
		filter.WebsiteID = &id
	}
	if status := c.Query("status"); status != "" {
		s := domain.MaintenanceStatus(status)
		filter.Status = &s
	}
	if includePast := c.Query("include_past"); includePast == "true" {
		filter.IncludePast = true
	}
	if page := c.Query("page"); page != "" {
		filter.Page, _ = strconv.Atoi(page)
	}
	if limit := c.Query("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}

	mws, total, err := h.maintenanceRepo.GetAll(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  mws,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}

// GetMaintenance retrieves a single maintenance window
// @Summary Get maintenance window detail
// @Tags Maintenance
// @Security BearerAuth
// @Produce json
// @Param id path int true "Maintenance ID"
// @Success 200 {object} domain.MaintenanceWindow
// @Router /api/maintenance/{id} [get]
func (h *MaintenanceHandler) GetMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	mw, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	if mw == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Maintenance window tidak ditemukan",
		})
		return
	}

	c.JSON(http.StatusOK, mw)
}

// CreateMaintenance creates a new maintenance window
// @Summary Create maintenance window
// @Tags Maintenance
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param input body domain.MaintenanceWindowCreate true "Maintenance data"
// @Success 201 {object} domain.MaintenanceWindow
// @Router /api/maintenance [post]
func (h *MaintenanceHandler) CreateMaintenance(c *gin.Context) {
	var input domain.MaintenanceWindowCreate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		userID = int64(0)
	}

	id, err := h.maintenanceRepo.Create(c.Request.Context(), &input, userID.(int64))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	mw, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusCreated, mw)
}

// UpdateMaintenance updates a maintenance window
// @Summary Update maintenance window
// @Tags Maintenance
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Maintenance ID"
// @Param input body domain.MaintenanceWindowUpdate true "Maintenance data"
// @Success 200 {object} domain.MaintenanceWindow
// @Router /api/maintenance/{id} [put]
func (h *MaintenanceHandler) UpdateMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	var input domain.MaintenanceWindowUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Data tidak valid: " + err.Error(),
		})
		return
	}

	// Check if exists
	existing, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Maintenance window tidak ditemukan",
		})
		return
	}

	if err := h.maintenanceRepo.Update(c.Request.Context(), id, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	mw, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusOK, mw)
}

// DeleteMaintenance deletes a maintenance window
// @Summary Delete maintenance window
// @Tags Maintenance
// @Security BearerAuth
// @Param id path int true "Maintenance ID"
// @Success 200 {object} map[string]string
// @Router /api/maintenance/{id} [delete]
func (h *MaintenanceHandler) DeleteMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Check if exists
	existing, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Maintenance window tidak ditemukan",
		})
		return
	}

	if err := h.maintenanceRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menghapus maintenance window",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Maintenance window berhasil dihapus",
	})
}

// CancelMaintenance cancels a scheduled maintenance window
// @Summary Cancel maintenance window
// @Tags Maintenance
// @Security BearerAuth
// @Param id path int true "Maintenance ID"
// @Success 200 {object} domain.MaintenanceWindow
// @Router /api/maintenance/{id}/cancel [post]
func (h *MaintenanceHandler) CancelMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Check if exists
	existing, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Maintenance window tidak ditemukan",
		})
		return
	}

	if existing.Status == domain.MaintenanceCompleted || existing.Status == domain.MaintenanceCancelled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maintenance window sudah selesai atau dibatalkan",
		})
		return
	}

	status := string(domain.MaintenanceCancelled)
	if err := h.maintenanceRepo.Update(c.Request.Context(), id, &domain.MaintenanceWindowUpdate{
		Status: &status,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal membatalkan maintenance window",
		})
		return
	}

	mw, _ := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, mw)
}

// CompleteMaintenance marks a maintenance window as completed
// @Summary Complete maintenance window
// @Tags Maintenance
// @Security BearerAuth
// @Param id path int true "Maintenance ID"
// @Success 200 {object} domain.MaintenanceWindow
// @Router /api/maintenance/{id}/complete [post]
func (h *MaintenanceHandler) CompleteMaintenance(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "ID tidak valid",
		})
		return
	}

	// Check if exists
	existing, err := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}
	if existing == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Maintenance window tidak ditemukan",
		})
		return
	}

	if existing.Status == domain.MaintenanceCompleted || existing.Status == domain.MaintenanceCancelled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Maintenance window sudah selesai atau dibatalkan",
		})
		return
	}

	status := string(domain.MaintenanceCompleted)
	if err := h.maintenanceRepo.Update(c.Request.Context(), id, &domain.MaintenanceWindowUpdate{
		Status: &status,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal menyelesaikan maintenance window",
		})
		return
	}

	mw, _ := h.maintenanceRepo.GetByID(c.Request.Context(), id)
	c.JSON(http.StatusOK, mw)
}

// GetCurrentMaintenance returns currently active maintenance windows
// @Summary Get current maintenance
// @Tags Maintenance
// @Security BearerAuth
// @Produce json
// @Success 200 {array} domain.MaintenanceWindow
// @Router /api/maintenance/current [get]
func (h *MaintenanceHandler) GetCurrentMaintenance(c *gin.Context) {
	mws, err := h.maintenanceRepo.GetCurrentMaintenance(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusOK, mws)
}

// GetUpcomingMaintenance returns scheduled future maintenance windows
// @Summary Get upcoming maintenance
// @Tags Maintenance
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Number of items (default: 5)"
// @Success 200 {array} domain.MaintenanceWindow
// @Router /api/maintenance/upcoming [get]
func (h *MaintenanceHandler) GetUpcomingMaintenance(c *gin.Context) {
	limit := 5
	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	mws, err := h.maintenanceRepo.GetUpcomingMaintenance(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil data maintenance",
		})
		return
	}

	c.JSON(http.StatusOK, mws)
}
