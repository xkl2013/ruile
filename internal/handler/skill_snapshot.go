package handler

import (
	"errors"
	"net/http"
	"strconv"

	appservice "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SkillSnapshotHandler struct {
	service interfaces.TenantSkillSnapshotService
}

func NewSkillSnapshotHandler(service interfaces.TenantSkillSnapshotService) *SkillSnapshotHandler {
	return &SkillSnapshotHandler{service: service}
}

func (h *SkillSnapshotHandler) List(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	snapshots, err := h.service.List(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("failed to list skill snapshots").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshots})
}

func (h *SkillSnapshotHandler) Create(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	snapshot, err := h.service.Create(c.Request.Context(), tenantID)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("failed to create skill snapshot").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": snapshot})
}

func (h *SkillSnapshotHandler) Activate(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	snapshot, err := h.service.Activate(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshot})
}

func (h *SkillSnapshotHandler) Rollback(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	snapshot, err := h.service.Rollback(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": snapshot})
}

func (h *SkillSnapshotHandler) Verify(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	valid, err := h.service.Verify(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"snapshot_id": c.Param("id"),
		"valid":       valid,
	}})
}

func (h *SkillSnapshotHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appservice.ErrTenantSkillSnapshotNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		c.Error(apperrors.NewNotFoundError("skill snapshot not found"))
	default:
		c.Error(apperrors.NewInternalServerError("skill snapshot operation failed").WithDetails(err.Error()))
	}
}
