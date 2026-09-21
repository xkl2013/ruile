package handler

import (
	"errors"
	"net/http"
	"strings"

	appsvc "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ExpertPackageHandler struct {
	service   interfaces.ExpertPackageService
	agentRuns interfaces.AgentRunService
}

func NewExpertPackageHandler(
	service interfaces.ExpertPackageService,
	agentRuns interfaces.AgentRunService,
) *ExpertPackageHandler {
	return &ExpertPackageHandler{service: service, agentRuns: agentRuns}
}

func (h *ExpertPackageHandler) ImportArchive(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(
		c.Writer,
		c.Request.Body,
		appsvc.ExpertPackageMaxArchiveBytes+1024*1024,
	)
	file, err := c.FormFile("file")
	if err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert package upload").WithDetails(err.Error()))
		return
	}
	version, err := h.service.ImportPackageArchive(c.Request.Context(), tenantID, userID, file)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": version})
}

// Import keeps the normalized JSON importer available for internal debugging.
// The administrator UI uses ImportArchive instead.
func (h *ExpertPackageHandler) Import(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertPackageImportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert package request").WithDetails(err.Error()))
		return
	}
	version, err := h.service.ImportPackage(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": version})
}

func (h *ExpertPackageHandler) List(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	packages, err := h.service.ListPackages(c.Request.Context(), tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": packages})
}

func (h *ExpertPackageHandler) ListPublished(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	experts, err := h.service.ListPublishedExperts(c.Request.Context(), tenantID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": experts})
}

func (h *ExpertPackageHandler) RoutePublished(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertAgentTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert route request").WithDetails(err.Error()))
		return
	}
	decision, err := h.agentRuns.RoutePublishedExpert(
		c.Request.Context(),
		tenantID,
		input.Prompt,
		input.ModelID,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": decision})
}

func (h *ExpertPackageHandler) Get(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	pkg, err := h.service.GetPackage(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": pkg})
}

func (h *ExpertPackageHandler) Publish(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.PublishVersion(c.Request.Context(), tenantID, userID, c.Param("id"), c.Param("version_id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ExpertPackageHandler) Bind(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.AgentBindingInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid agent binding request").WithDetails(err.Error()))
		return
	}
	binding, err := h.service.BindAgent(c.Request.Context(), tenantID, userID, c.Param("id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": binding})
}

func (h *ExpertPackageHandler) ListBindings(c *gin.Context) {
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	bindings, err := h.service.ListBindings(c.Request.Context(), tenantID, strings.TrimSpace(c.Query("profile_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": bindings})
}

func (h *ExpertPackageHandler) TestRun(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertAgentTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert test request").WithDetails(err.Error()))
		return
	}
	input.PackageID = c.Param("id")
	input.DefinitionID = c.Param("definition_id")
	run, err := h.agentRuns.EnqueueExpertTest(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ExpertPackageHandler) PublishedRun(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertAgentTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert run request").WithDetails(err.Error()))
		return
	}
	input.PackageID = c.Param("package_id")
	input.DefinitionID = c.Param("definition_id")
	input.ProfileID = ""
	input.RouteMode = types.AgentRouteModeManual
	run, err := h.agentRuns.EnqueuePublishedExpertRun(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ExpertPackageHandler) PublishedAutoRun(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertAgentTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert auto-route request").WithDetails(err.Error()))
		return
	}
	input.PackageID = ""
	input.DefinitionID = ""
	input.ProfileID = ""
	input.RouteMode = types.AgentRouteModeAuto
	run, err := h.agentRuns.EnqueuePublishedExpertAutoRun(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ExpertPackageHandler) PublishedFollowUp(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertFollowUpInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid expert follow-up request").WithDetails(err.Error()))
		return
	}
	input.ParentRunID = c.Param("run_id")
	input.Mode = "explanation"
	run, err := h.agentRuns.EnqueuePublishedExpertFollowUp(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ExpertPackageHandler) handleError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, appsvc.ErrExpertPackageNotFound) || errors.Is(err, appsvc.ErrExpertPackageVersionNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case errors.Is(err, appsvc.ErrExpertPackageInvalidInput) ||
		errors.Is(err, appsvc.ErrExpertPackageVersionExists) ||
		errors.Is(err, appsvc.ErrExpertPackageNotPublished) ||
		errors.Is(err, appsvc.ErrAgentRunInvalidRequest) ||
		errors.Is(err, appsvc.ErrAgentRunNoExpertMatch) ||
		errors.Is(err, appsvc.ErrAgentRunRouteConfirm):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
