package handler

import (
	"net/http"
	"strconv"

	appservice "github.com/Tencent/WeKnora/internal/application/service"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type AgentDelegationRunHandler struct {
	service interfaces.AgentDelegationRunService
}

func NewAgentDelegationRunHandler(service interfaces.AgentDelegationRunService) *AgentDelegationRunHandler {
	return &AgentDelegationRunHandler{service: service}
}

func (h *AgentDelegationRunHandler) List(c *gin.Context) {
	tenantID, ok := types.TenantIDFromContext(c.Request.Context())
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	runs, err := h.service.List(
		c.Request.Context(),
		tenantID,
		c.Query("parent_session_id"),
		c.Query("member_agent_id"),
		limit,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to list delegated Agent runs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": runs})
}

func (h *AgentDelegationRunHandler) Get(c *gin.Context) {
	tenantID, ok := types.TenantIDFromContext(c.Request.Context())
	if !ok || tenantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "workspace context is required"})
		return
	}
	run, err := h.service.Get(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		if err == appservice.ErrAgentDelegationRunNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "delegated Agent run not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "failed to get delegated Agent run"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}
