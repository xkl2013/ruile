package handler

import (
	"errors"
	"net/http"

	appservice "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SandboxConfigHandler struct {
	service interfaces.TenantSandboxConfigService
}

func NewSandboxConfigHandler(service interfaces.TenantSandboxConfigService) *SandboxConfigHandler {
	return &SandboxConfigHandler{service: service}
}

func (h *SandboxConfigHandler) List(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	configs, err := h.service.List(c.Request.Context(), tenantID)
	if err != nil {
		c.Error(apperrors.NewInternalServerError("failed to list sandbox configs").WithDetails(err.Error()))
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": configs})
}

func (h *SandboxConfigHandler) Create(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.TenantSandboxConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid sandbox config").WithDetails(err.Error()))
		return
	}
	config, err := h.service.Create(c.Request.Context(), tenantID, &input)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": config})
}

func (h *SandboxConfigHandler) Update(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.TenantSandboxConfig
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid sandbox config").WithDetails(err.Error()))
		return
	}
	config, err := h.service.Update(c.Request.Context(), tenantID, c.Param("id"), &input)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

func (h *SandboxConfigHandler) Activate(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	config, err := h.service.Activate(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": config})
}

func (h *SandboxConfigHandler) writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appservice.ErrTenantSandboxConfigNotFound),
		errors.Is(err, gorm.ErrRecordNotFound):
		c.Error(apperrors.NewNotFoundError("sandbox config not found"))
	case errors.Is(err, appservice.ErrTenantSandboxConfigDisabled):
		c.Error(apperrors.NewBadRequestError("sandbox config is disabled"))
	case errors.Is(err, types.ErrTenantSandboxConfigNameRequired),
		errors.Is(err, types.ErrTenantSandboxConfigModeInvalid),
		errors.Is(err, types.ErrTenantSandboxConfigTimeoutInvalid):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError("sandbox config operation failed").WithDetails(err.Error()))
	}
}
