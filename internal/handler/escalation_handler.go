package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/diskominfos-bali/monitoring-website/internal/domain"
	"github.com/diskominfos-bali/monitoring-website/internal/service/escalation"
)

type EscalationHandler struct {
	escalationService *escalation.EscalationService
}

func NewEscalationHandler(escalationService *escalation.EscalationService) *EscalationHandler {
	return &EscalationHandler{
		escalationService: escalationService,
	}
}

// ListPolicies returns all escalation policies
// GET /api/escalation/policies
func (h *EscalationHandler) ListPolicies(c *gin.Context) {
	policies, err := h.escalationService.GetPolicies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": policies})
}

// GetPolicy returns a specific policy by ID
// GET /api/escalation/policies/:id
func (h *EscalationHandler) GetPolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	policy, err := h.escalationService.GetPolicy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if policy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Policy not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": policy})
}

// CreatePolicy creates a new escalation policy
// POST /api/escalation/policies
func (h *EscalationHandler) CreatePolicy(c *gin.Context) {
	var req domain.EscalationPolicyCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.escalationService.CreatePolicy(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Policy created successfully"})
}

// UpdatePolicy updates an existing policy
// PUT /api/escalation/policies/:id
func (h *EscalationHandler) UpdatePolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	var req domain.EscalationPolicyCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.escalationService.UpdatePolicy(c.Request.Context(), id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Policy updated successfully"})
}

// DeletePolicy deletes a policy
// DELETE /api/escalation/policies/:id
func (h *EscalationHandler) DeletePolicy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid policy ID"})
		return
	}

	if err := h.escalationService.DeletePolicy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Policy deleted successfully"})
}

// CreateRule creates a new escalation rule
// POST /api/escalation/rules
func (h *EscalationHandler) CreateRule(c *gin.Context) {
	var req domain.EscalationRuleCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.escalationService.CreateRule(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "Rule created successfully"})
}

// DeleteRule deletes an escalation rule
// DELETE /api/escalation/rules/:id
func (h *EscalationHandler) DeleteRule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid rule ID"})
		return
	}

	if err := h.escalationService.DeleteRule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

// GetHistory returns escalation history
// GET /api/escalation/history
func (h *EscalationHandler) GetHistory(c *gin.Context) {
	var filter domain.EscalationFilter

	if alertID := c.Query("alert_id"); alertID != "" {
		id, _ := strconv.ParseInt(alertID, 10, 64)
		filter.AlertID = &id
	}
	if level := c.Query("level"); level != "" {
		l, _ := strconv.Atoi(level)
		filter.Level = &l
	}
	if status := c.Query("status"); status != "" {
		filter.Status = &status
	}

	if page := c.Query("page"); page != "" {
		filter.Page, _ = strconv.Atoi(page)
	}
	if limit := c.Query("limit"); limit != "" {
		filter.Limit, _ = strconv.Atoi(limit)
	}

	history, total, err := h.escalationService.GetHistory(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"history": history,
		"total":   total,
		"page":    filter.Page,
		"limit":   filter.Limit,
	})
}

// GetSummary returns escalation summary
// GET /api/escalation/summary
func (h *EscalationHandler) GetSummary(c *gin.Context) {
	summary, err := h.escalationService.GetSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": summary})
}

// TriggerEscalation manually triggers escalation processing
// POST /api/escalation/trigger
func (h *EscalationHandler) TriggerEscalation(c *gin.Context) {
	if err := h.escalationService.ProcessEscalations(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Escalation processing triggered successfully"})
}
