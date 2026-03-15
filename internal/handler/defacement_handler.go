package handler

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
	"github.com/diskominfos-bali/monitoring-website/internal/service/monitor"
)

type DefacementHandler struct {
	scanner        *monitor.DefacementArchiveScanner
	defacementRepo *mysql.DefacementRepository
}

func NewDefacementHandler(scanner *monitor.DefacementArchiveScanner, defacementRepo *mysql.DefacementRepository) *DefacementHandler {
	return &DefacementHandler{
		scanner:        scanner,
		defacementRepo: defacementRepo,
	}
}

// GetStats returns defacement archive statistics
func (h *DefacementHandler) GetStats(c *gin.Context) {
	stats, err := h.defacementRepo.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// ListIncidents returns defacement incidents with filters
func (h *DefacementHandler) ListIncidents(c *gin.Context) {
	var websiteID int64
	if idStr := c.Query("website_id"); idStr != "" {
		websiteID, _ = strconv.ParseInt(idStr, 10, 64)
	}
	source := c.Query("source")

	var acknowledged *bool
	if ack := c.Query("acknowledged"); ack != "" {
		v := ack == "true"
		acknowledged = &v
	}

	limit := 50
	if l := c.Query("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	offset := 0
	if o := c.Query("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}

	incidents, total, err := h.defacementRepo.GetIncidents(c.Request.Context(), websiteID, source, acknowledged, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get incidents"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": incidents, "total": total})
}

// AcknowledgeIncident marks an incident as acknowledged
func (h *DefacementHandler) AcknowledgeIncident(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var body struct {
		Notes string `json:"notes"`
	}
	c.ShouldBindJSON(&body)

	user := "admin"
	if u, exists := c.Get("username"); exists {
		user = u.(string)
	}

	if err := h.defacementRepo.AcknowledgeIncident(c.Request.Context(), id, user, body.Notes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Incident acknowledged"})
}

// TriggerScan triggers a manual defacement archive scan
func (h *DefacementHandler) TriggerScan(c *gin.Context) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()
		h.scanner.ScanAll(ctx)
	}()
	c.JSON(http.StatusOK, gin.H{"message": "Defacement archive scan dimulai"})
}
