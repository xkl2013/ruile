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

func (h *ServiceSpaceHandler) ListSubjects(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	page, pageSize, valid := parseServiceSpacePagination(c)
	if !valid {
		return
	}
	subjects, total, err := h.service.ListSubjects(
		c.Request.Context(), tenantID, userID, c.Param("service_id"),
		c.Query("subject_type"), page, pageSize,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items": subjects, "total": total, "page": page, "page_size": pageSize,
	}})
}

func (h *ServiceSpaceHandler) GetSubject(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	subject, err := h.service.GetSubject(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("subject_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": subject})
}

func (h *ServiceSpaceHandler) CreateSubject(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSubjectCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service subject request").WithDetails(err.Error()))
		return
	}
	subject, err := h.service.CreateSubject(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": subject})
}

func (h *ServiceSpaceHandler) UpdateSubject(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSubjectUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service subject update request").WithDetails(err.Error()))
		return
	}
	subject, err := h.service.UpdateSubject(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("subject_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": subject})
}

func (h *ServiceSpaceHandler) DeleteSubject(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteSubject(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("subject_id"),
	); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListReminderStatuses(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	statuses, err := h.service.ListReminderStatuses(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Query("include_disabled") == "true",
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": statuses})
}

func (h *ServiceSpaceHandler) CreateReminderStatus(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderStatusCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reminder status request").WithDetails(err.Error()))
		return
	}
	status, err := h.service.CreateReminderStatus(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": status})
}

func (h *ServiceSpaceHandler) UpdateReminderStatus(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderStatusUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reminder status update request").WithDetails(err.Error()))
		return
	}
	status, err := h.service.UpdateReminderStatus(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("status_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": status})
}

func (h *ServiceSpaceHandler) DeleteReminderStatus(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteReminderStatus(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("status_id"),
	); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListReminderStatusTransitions(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	transitions, err := h.service.ListReminderStatusTransitions(
		c.Request.Context(), tenantID, userID, c.Param("service_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": transitions})
}

func (h *ServiceSpaceHandler) ReplaceReminderStatusTransitions(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderStatusTransitionReplaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reminder status transitions request").WithDetails(err.Error()))
		return
	}
	transitions, err := h.service.ReplaceReminderStatusTransitions(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": transitions})
}

func (h *ServiceSpaceHandler) ListReminders(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	page, pageSize, valid := parseServiceSpacePagination(c)
	if !valid {
		return
	}
	reminders, total, err := h.service.ListReminders(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Query("status"), page, pageSize,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"items": reminders, "total": total, "page": page, "page_size": pageSize,
	}})
}

func (h *ServiceSpaceHandler) GetReminder(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	reminder, err := h.service.GetReminder(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceSpaceHandler) CreateReminder(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service reminder request").WithDetails(err.Error()))
		return
	}
	reminder, err := h.service.CreateReminder(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": reminder})
}

func (h *ServiceSpaceHandler) UpdateReminder(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service reminder update request").WithDetails(err.Error()))
		return
	}
	reminder, err := h.service.UpdateReminder(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceSpaceHandler) DeleteReminder(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteReminder(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"),
	); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListReminderAssignees(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	assignees, err := h.service.ListReminderAssignees(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignees})
}

func (h *ServiceSpaceHandler) ReplaceReminderAssignees(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderAssigneeReplaceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reminder assignees request").WithDetails(err.Error()))
		return
	}
	assignees, err := h.service.ReplaceReminderAssignees(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": assignees})
}

func (h *ServiceSpaceHandler) ListReminderComments(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	comments, err := h.service.ListReminderComments(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": comments})
}

func (h *ServiceSpaceHandler) AddReminderComment(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceReminderCommentCreateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid reminder comment request").WithDetails(err.Error()))
		return
	}
	comment, err := h.service.AddReminderComment(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"), input,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": comment})
}

func (h *ServiceSpaceHandler) DeleteReminderComment(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteReminderComment(
		c.Request.Context(), tenantID, userID, c.Param("service_id"),
		c.Param("reminder_id"), c.Param("comment_id"),
	); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *ServiceSpaceHandler) ListReminderHistory(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	history, err := h.service.ListReminderHistory(
		c.Request.Context(), tenantID, userID, c.Param("service_id"), c.Param("reminder_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": history})
}

func (h *ServiceSpaceHandler) ListTemplates(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	templates, err := h.service.ListTemplates(c.Request.Context(), tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": templates})
}

func (h *ServiceSpaceHandler) ApplyTemplate(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceTemplateApplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service template apply request").WithDetails(err.Error()))
		return
	}
	if input.TemplateKey == "" {
		input.TemplateKey = c.Param("template_key")
	}
	service, err := h.service.ApplyTemplate(c.Request.Context(), tenantID, userID, input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) PreviewBlueprint(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceBlueprintPreviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid blueprint preview request").WithDetails(err.Error()))
		return
	}
	blueprint, err := h.service.PreviewBlueprint(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": blueprint})
}

func (h *ServiceSpaceHandler) PreviewStandaloneBlueprint(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceBlueprintPreviewInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid blueprint preview request").WithDetails(err.Error()))
		return
	}
	blueprint, err := h.service.PreviewBlueprint(c.Request.Context(), tenantID, userID, "", input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": blueprint})
}

func (h *ServiceSpaceHandler) GetBlueprint(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	blueprint, err := h.service.GetBlueprint(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": blueprint})
}

func (h *ServiceSpaceHandler) ConfirmBlueprint(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceBlueprintConfirmInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid blueprint confirm request").WithDetails(err.Error()))
		return
	}
	service, err := h.service.ConfirmBlueprint(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": service})
}

func (h *ServiceSpaceHandler) GetProfile(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	profile, err := h.service.GetProfile(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceSpaceHandler) UpdateProfile(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceProfileUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid service profile request").WithDetails(err.Error()))
		return
	}
	profile, err := h.service.UpdateProfile(c.Request.Context(), tenantID, userID, c.Param("service_id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceSpaceHandler) GetSummary(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	summary, err := h.service.GetSummary(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func (h *ServiceSpaceHandler) RefreshSummary(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	summary, err := h.service.RefreshSummary(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}

func (h *ServiceSpaceHandler) GetRuntimeContext(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	runtimeContext, err := h.service.ResolveRuntimeContext(c.Request.Context(), tenantID, userID, c.Param("service_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": runtimeContext})
}

func (h *ServiceSpaceHandler) ListContextSources(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	sources, err := h.service.ListContextSources(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": sources})
}

func (h *ServiceSpaceHandler) ImportContextSource(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceContextSourceImportInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid context source request").WithDetails(err.Error()))
		return
	}
	if strings.TrimSpace(input.SourceType) != types.ServiceContextSourceTypeOrganizeOutput {
		c.Error(apperrors.NewBadRequestError("unsupported context source type"))
		return
	}
	source, err := h.service.ImportOrganizeOutput(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		input.SourceID,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "data": source})
}

func (h *ServiceSpaceHandler) DeleteContextSource(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if err := h.service.DeleteContextSource(
		c.Request.Context(),
		tenantID,
		userID,
		c.Param("service_id"),
		c.Param("source_id"),
	); err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
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

func (h *ServiceSpaceHandler) UpdateArtifactLifecycle(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.ServiceSpaceArtifactLifecycleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid artifact lifecycle request").WithDetails(err.Error()))
		return
	}
	artifact, err := h.service.UpdateArtifactLifecycle(
		c.Request.Context(), tenantID, userID, c.Param("service_id"),
		c.Param("artifact_id"), input.Lifecycle, input.IdempotencyKey,
	)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": artifact})
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
		stderrors.Is(err, appsvc.ErrServiceSpaceContextSourceNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceReminderNotFound),
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
		stderrors.Is(err, appsvc.ErrServiceSpaceBlueprintNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceBlueprintConflict),
		stderrors.Is(err, appsvc.ErrServiceSpaceTemplateNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceTemplateNotAllowed),
		stderrors.Is(err, appsvc.ErrServiceSpaceInvalidLifecycle),
		stderrors.Is(err, appsvc.ErrServiceSpaceKBOutOfScope),
		stderrors.Is(err, appsvc.ErrServiceSpaceSubjectInvalid),
		stderrors.Is(err, appsvc.ErrServiceSpaceSubjectParent),
		stderrors.Is(err, appsvc.ErrServiceSpaceStatusInvalid),
		stderrors.Is(err, appsvc.ErrServiceSpaceStatusInUse),
		stderrors.Is(err, appsvc.ErrServiceSpaceLastStatus),
		stderrors.Is(err, appsvc.ErrServiceSpaceDuplicateStatusKey),
		stderrors.Is(err, appsvc.ErrServiceSpaceTransitionInvalid),
		stderrors.Is(err, appsvc.ErrServiceSpaceContextSourceInvalid),
		stderrors.Is(err, appsvc.ErrServiceSpaceContextSourceNotReady),
		stderrors.Is(err, appsvc.ErrServiceSpaceContextSourceAssigned),
		stderrors.Is(err, appsvc.ErrServiceSpaceReminderInvalid),
		stderrors.Is(err, appsvc.ErrServiceSpaceReminderTransition),
		stderrors.Is(err, appsvc.ErrServiceSpaceReminderParent),
		stderrors.Is(err, appsvc.ErrServiceSpaceReminderDepth),
		stderrors.Is(err, appsvc.ErrServiceSpaceProfileSchemaInvalid),
		stderrors.Is(err, appsvc.ErrAgentRunInvalidRequest):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceSpaceSubjectNotFound),
		stderrors.Is(err, appsvc.ErrServiceSpaceStatusNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
