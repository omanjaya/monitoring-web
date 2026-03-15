package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/repository/mysql"
)

type AuditHandler struct {
	auditRepo *mysql.AuditRepository
}

func NewAuditHandler(auditRepo *mysql.AuditRepository) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
	}
}

// ListAuditLogs returns paginated audit logs with optional filters
// @Summary List audit logs
// @Tags Audit
// @Security BearerAuth
// @Produce json
// @Param user_id query int false "Filter by user ID"
// @Param action query string false "Filter by action"
// @Param resource_type query string false "Filter by resource type"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{}
// @Router /api/audit-logs [get]
func (h *AuditHandler) ListAuditLogs(c *gin.Context) {
	filter := domain.AuditFilter{
		Page:  1,
		Limit: 20,
	}

	if v := c.Query("user_id"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			filter.UserID = &id
		}
	}
	if v := c.Query("action"); v != "" {
		filter.Action = &v
	}
	if v := c.Query("resource_type"); v != "" {
		filter.ResourceType = &v
	}
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			filter.Page = p
		}
	}
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			filter.Limit = l
		}
	}

	logs, total, err := h.auditRepo.GetAll(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Gagal mengambil audit logs",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  logs,
		"total": total,
		"page":  filter.Page,
		"limit": filter.Limit,
	})
}
