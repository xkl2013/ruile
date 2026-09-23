package handler

import (
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	appsvc "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ServiceSpaceHandler struct {
	service     interfaces.ServiceSpaceService
	agentRuns   interfaces.AgentRunService
	fileService interfaces.FileService
}

func NewServiceSpaceHandler(
	service interfaces.ServiceSpaceService,
	agentRuns interfaces.AgentRunService,
	fileService interfaces.FileService,
) *ServiceSpaceHandler {
	return &ServiceSpaceHandler{
		service:     service,
		agentRuns:   agentRuns,
		fileService: fileService,
	}
}

func (h *ServiceSpaceHandler) SessionScopeMiddleware() gin.HandlerFunc {
	if h == nil {
		return ServiceSessionScopeMiddleware(nil)
	}
	return ServiceSessionScopeMiddleware(h.service)
}

func (h *ServiceSpaceHandler) List(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	services, err := h.service.List(c.Request.Context(), tenantID, userID, c.Query("include_archived") == "true")
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": services})
}

func (h *ServiceSpaceHandler) Create(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service create request").WithDetails(err.Error()))
		return
	}
	service, err := h.service.Create(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) Get(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	service, err := h.service.Get(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) Update(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service update request").WithDetails(err.Error()))
		return
	}
	service, err := h.service.Update(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) SetState(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	state := strings.TrimSpace(c.Param("state"))
	if state == "" {
		var input struct {
			State string `json:"state"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.Error(apperrors.NewBadRequestError("service state is required"))
			return
		}
		state = strings.TrimSpace(input.State)
	}
	service, err := h.service.SetState(c.Request.Context(), tenantID, userID, c.Param("service_id"), state)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) SetDefault(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.SetDefault(c.Request.Context(), tenantID, userID, c.Param("service_id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) Delete(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), tenantID, userID, c.Param("service_id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) Overview(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	overview, err := h.service.GetOverview(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": overview})
}

func (h *ServiceSpaceHandler) CreateAgentRun(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ExpertAgentTestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service agent run request").WithDetails(err.Error()))
		return
	}
	input.ServiceID = c.Param("service_id")
	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.SessionID == "" {
		c.Error(apperrors.NewBadRequestError("session_id is required"))
		return
	}
	run, err := h.agentRuns.EnqueuePublishedExpertRun(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ServiceSpaceHandler) GetAgentRun(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	run, err := h.agentRuns.GetAgentRunForService(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		c.Param("run_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ServiceSpaceHandler) ListAgentRunSteps(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	steps, err := h.agentRuns.ListAgentRunStepsForService(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		c.Param("run_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": steps})
}

func (h *ServiceSpaceHandler) ListAgentRunEvents(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	after, err := strconv.ParseInt(strings.TrimSpace(c.DefaultQuery("after", "0")), 10, 64)
	if err != nil || after < 0 {
		c.Error(apperrors.NewBadRequestError("after must be a non-negative integer"))
		return
	}
	events, err := h.agentRuns.ListAgentRunEventsForService(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		c.Param("run_id"),
		after,
		100,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": events})
}

func (h *ServiceSpaceHandler) ListSessions(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	page, pageSize, valid := parseServiceSpacePagination(c)
	if !valid {
		return
	}
	sessions, total, err := h.service.ListSessions(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		c.Query("keyword"),
		page,
		pageSize,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items": sessions, "total": total, "page": page, "page_size": pageSize,
	}})
}

func (h *ServiceSpaceHandler) CreateSession(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSessionCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service session request").WithDetails(err.Error()))
		return
	}
	session, err := h.service.CreateSession(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": session})
}

func (h *ServiceSpaceHandler) GetSession(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	session, err := h.service.GetSession(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("session_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": session})
}

func (h *ServiceSpaceHandler) UpdateSession(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSessionUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service session update request").WithDetails(err.Error()))
		return
	}
	session, err := h.service.UpdateSession(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("session_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": session})
}

func (h *ServiceSpaceHandler) SetSessionPinned(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Pinned bool `json:"pinned"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("pinned is required"))
		return
	}
	if err := h.service.SetSessionPinned(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("session_id"), input.Pinned); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) DeleteSession(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteSession(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("session_id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListMembers(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	members, err := h.service.ListMembers(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Query("status"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": members})
}

func (h *ServiceSpaceHandler) AddMember(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceMemberInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service member request").WithDetails(err.Error()))
		return
	}
	member, err := h.service.AddMember(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": member})
}

func (h *ServiceSpaceHandler) UpdateMemberRole(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("member role is required"))
		return
	}
	if err := h.service.UpdateMemberRole(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("member_user_id"), input.Role); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) RemoveMember(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.RemoveMember(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("member_user_id")); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListExperts(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	experts, err := h.service.ListExperts(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": experts})
}

func (h *ServiceSpaceHandler) ReplaceExperts(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input struct {
		Experts []types.ServiceExpertBindingInput `json:"experts"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service experts request").WithDetails(err.Error()))
		return
	}
	experts, err := h.service.ReplaceExperts(c.Request.Context(), tenantID, userID, c.Param("service_id"), input.Experts)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": experts})
}

func (h *ServiceSpaceHandler) ListArtifacts(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	page, pageSize, valid := parseServiceSpacePagination(c)
	if !valid {
		return
	}
	artifacts, total, err := h.service.ListArtifacts(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Query("lifecycle"), page, pageSize)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items": artifacts, "total": total, "page": page, "page_size": pageSize,
	}})
}

func (h *ServiceSpaceHandler) StreamArtifact(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if h.fileService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	version := 0
	if raw := strings.TrimSpace(c.Query("version")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			c.Error(apperrors.NewBadRequestError("version must be a positive integer"))
			return
		}
		version = parsed
	}
	artifact, err := h.service.GetArtifact(c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("artifact_id"), version)
	if err != nil {
		h.handleError(c, err)
		return
	}
	if strings.TrimSpace(artifact.ResourceRef) == "" {
		c.Status(http.StatusNotFound)
		return
	}
	reader, err := h.fileService.GetFile(c.Request.Context(), artifact.ResourceRef)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	contentType := artifact.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	disposition := "inline"
	if c.Param("mode") == "download" {
		disposition = "attachment"
	}
	name := sanitizeDownloadName(firstNonEmptyHandler(artifact.OriginalName, artifact.Title, "artifact"))
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, name))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.Warnf(c.Request.Context(), "failed to stream service artifact: service_id=%s artifact_id=%s err=%v", c.Param("service_id"), artifact.ArtifactID, err)
	}
}

func parseServiceSpacePagination(c *gin.Context) (int, int, bool) {
	page, pageSize := 1, 20
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			c.Error(apperrors.NewBadRequestError("page must be a positive integer"))
			return 0, 0, false
		}
		page = parsed
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			c.Error(apperrors.NewBadRequestError("page_size must be a positive integer"))
			return 0, 0, false
		}
		if parsed > 100 {
			parsed = 100
		}
		pageSize = parsed
	}
	return page, pageSize, true
}

func (h *ServiceSpaceHandler) handleError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, appsvc.ErrServiceSpaceInvalidScope):
		c.Error(apperrors.NewUnauthorizedError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceSpaceNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceSessionNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceArtifactNotFound),
		stderrors.Is(err, appsvc.ErrAgentRunNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceSpaceForbidden):
		c.Error(apperrors.NewForbiddenError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceSpaceArchived),
		stderrors.Is(err, appsvc.ErrServiceSpaceNotActive):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceSpaceNameRequired),
		stderrors.Is(err, appsvc.ErrServiceSpaceInvalidState),
		stderrors.Is(err, appsvc.ErrServiceSpaceExpertRequired),
		stderrors.Is(err, appsvc.ErrServiceSpaceInvalidRole),
		stderrors.Is(err, appsvc.ErrServiceSpaceOwnerImmutable),
		stderrors.Is(err, appsvc.ErrServiceSpaceMemberLimit),
		stderrors.Is(err, appsvc.ErrServiceSpaceDuplicateExpertRef),
		stderrors.Is(err, appsvc.ErrAgentRunInvalidRequest):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
