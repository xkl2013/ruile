package handler

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	appsvc "github.com/Tencent/WeKnora/internal/application/service"
	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	service     interfaces.ServiceService
	agentRuns   interfaces.AgentRunService
	fileService interfaces.FileService
}

func NewServiceHandler(
	svc interfaces.ServiceService,
	agentRuns interfaces.AgentRunService,
	fileService interfaces.FileService,
) *ServiceHandler {
	return &ServiceHandler{service: svc, agentRuns: agentRuns, fileService: fileService}
}

func serviceScope(c *gin.Context) (uint64, string, bool) {
	tenantID := c.GetUint64(types.TenantIDContextKey.String())
	if tenantID == 0 {
		c.Error(apperrors.NewUnauthorizedError("workspace ID not found"))
		return 0, "", false
	}
	userID := strings.TrimSpace(c.GetString(types.UserIDContextKey.String()))
	if userID == "" {
		c.Error(apperrors.NewUnauthorizedError("user ID not found"))
		return 0, "", false
	}
	return tenantID, userID, true
}

func (h *ServiceHandler) GetBootstrap(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.service.GetBootstrap(ctx, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) Refresh(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.service.RefreshUserService(ctx, tenantID, userID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) ExtractMemory(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	data, err := h.agentRuns.EnqueueMemoryExtraction(ctx, tenantID, userID, c.Param("memory_id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": data})
}

func (h *ServiceHandler) ListDailyReports(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.dailyReportListQuery(c)
	if !ok {
		return
	}
	reports, total, err := h.service.ListDailyReports(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(reports, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetDailyReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	report, err := h.service.GetDailyReport(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": report})
}

func (h *ServiceHandler) RenderDailyReportHTML(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	rendered, err := h.service.RenderDailyReportHTML(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(rendered))
}

func (h *ServiceHandler) GenerateDailyReport(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceDailyReportInput
	if err := c.ShouldBindJSON(&req); err != nil && !stderrors.Is(err, io.EOF) {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	if req.Range == "" {
		req.Range = strings.TrimSpace(c.Query("range"))
	}
	if req.Date == "" {
		req.Date = strings.TrimSpace(c.Query("date"))
	}
	if req.Timezone == "" {
		req.Timezone = strings.TrimSpace(c.Query("timezone"))
	}
	if req.Trigger == "" {
		req.Trigger = strings.TrimSpace(c.Query("trigger"))
	}
	run, err := h.agentRuns.EnqueueDailyReport(ctx, tenantID, userID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ServiceHandler) GetAgentRun(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	run, err := h.agentRuns.GetAgentRun(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ServiceHandler) PreviewAgentRunArtifact(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	if h.fileService == nil {
		c.Status(http.StatusNotFound)
		return
	}
	run, err := h.agentRuns.GetAgentRun(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}

	rawArtifacts, ok := run.Result["artifacts"]
	if !ok {
		c.Status(http.StatusNotFound)
		return
	}
	raw, err := json.Marshal(rawArtifacts)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	var artifacts []types.AgentArtifactResultV1
	if err := json.Unmarshal(raw, &artifacts); err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	artifactID := strings.TrimSpace(c.Param("artifact_id"))
	var artifact *types.AgentArtifactResultV1
	for index := range artifacts {
		if artifacts[index].ID == artifactID {
			artifact = &artifacts[index]
			break
		}
	}
	if artifact == nil || strings.TrimSpace(artifact.ResourceRef) == "" {
		c.Status(http.StatusNotFound)
		return
	}
	reader, err := h.fileService.GetFile(ctx, artifact.ResourceRef)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer reader.Close()

	contentType := artifact.MimeType
	if contentType == "" {
		contentType = "text/html; charset=utf-8"
	}
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "inline")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.Warnf(ctx, "failed to stream agent artifact preview: run_id=%s artifact_id=%s err=%v",
			run.ID, artifact.ID, err)
	}
}

func (h *ServiceHandler) GetAgentRunQuality(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	quality, err := h.agentRuns.GetAgentRunQuality(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": quality})
}

// StreamAgentRunEvents replays durable AgentRun events after the supplied
// sequence and keeps polling until the run reaches a terminal state. The
// response data is flattened into the AG-UI shape used by the TDesign Chat
// renderer, while the database keeps the original payload envelope.
func (h *ServiceHandler) StreamAgentRunEvents(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	afterSequence := int64(0)
	if raw := strings.TrimSpace(c.Query("after")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			c.Error(apperrors.NewBadRequestError("after must be a non-negative integer"))
			return
		}
		afterSequence = parsed
	} else if raw := strings.TrimSpace(c.GetHeader("Last-Event-ID")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			c.Error(apperrors.NewBadRequestError("Last-Event-ID must be a non-negative integer"))
			return
		}
		afterSequence = parsed
	}
	limit := 100
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			c.Error(apperrors.NewBadRequestError("limit must be a positive integer"))
			return
		}
		if parsed > 500 {
			parsed = 500
		}
		limit = parsed
	}

	runID := strings.TrimSpace(c.Param("id"))
	run, err := h.agentRuns.GetAgentRun(ctx, tenantID, userID, runID)
	if err != nil {
		h.handleError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Writer.Flush()

	pollTicker := time.NewTicker(300 * time.Millisecond)
	defer pollTicker.Stop()
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		events, listErr := h.agentRuns.ListAgentRunEvents(
			ctx,
			tenantID,
			userID,
			runID,
			afterSequence,
			limit,
		)
		if listErr != nil {
			_ = writeAgentRunSSE(c, map[string]any{
				"type":      types.AgentRunEventTypeRunError,
				"runId":     runID,
				"errorCode": "agent_run_event_stream_failed",
				"message":   listErr.Error(),
			})
			return
		}
		for _, event := range events {
			if event == nil {
				continue
			}
			if err := writeAgentRunEventSSE(c, event); err != nil {
				return
			}
			afterSequence = event.Sequence
		}

		if len(events) == 0 && isTerminalAgentRunStatus(run.Status) {
			return
		}
		if len(events) > 0 && isTerminalAgentRunStatus(run.Status) {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-pollTicker.C:
			run, err = h.agentRuns.GetAgentRun(ctx, tenantID, userID, runID)
			if err != nil {
				return
			}
		case <-heartbeatTicker.C:
			if err := writeAgentRunSSE(c, map[string]any{
				"type":   "PING",
				"runId":  runID,
				"status": run.Status,
			}); err != nil {
				return
			}
		}
	}
}

func writeAgentRunEventSSE(c *gin.Context, event *types.AgentRunEvent) error {
	data := map[string]any{
		"type":      event.EventType,
		"id":        event.ID,
		"runId":     event.RunID,
		"sequence":  event.Sequence,
		"createdAt": event.CreatedAt,
	}
	for key, value := range event.Payload {
		data[key] = value
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "id: %d\n", event.Sequence); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func writeAgentRunSSE(c *gin.Context, data map[string]any) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", raw); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func isTerminalAgentRunStatus(status string) bool {
	switch status {
	case types.AgentRunStatusWaitingInput,
		types.AgentRunStatusSucceeded,
		types.AgentRunStatusFailed,
		types.AgentRunStatusCancelled:
		return true
	default:
		return false
	}
}

func (h *ServiceHandler) ListAgentRunSteps(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	steps, err := h.agentRuns.ListAgentRunSteps(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": steps})
}

func (h *ServiceHandler) SubmitAgentRunAnswers(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.AgentRunAnswersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid agent run answers").WithDetails(err.Error()))
		return
	}
	run, err := h.agentRuns.SubmitAgentRunAnswers(ctx, tenantID, userID, c.Param("id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ServiceHandler) RegenerateAgentRun(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.AgentRunRegenerateInput
	if err := c.ShouldBindJSON(&input); err != nil && !stderrors.Is(err, io.EOF) {
		c.Error(apperrors.NewBadRequestError("invalid agent run regeneration request").WithDetails(err.Error()))
		return
	}
	run, err := h.agentRuns.RegenerateAgentRun(ctx, tenantID, userID, c.Param("id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *ServiceHandler) CancelAgentRun(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	run, err := h.agentRuns.CancelAgentRun(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *ServiceHandler) ListCustomerSpaces(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.customerSpaceListQuery(c)
	if !ok {
		return
	}
	spaces, total, err := h.service.ListCustomerSpaces(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(spaces, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetCustomerSpace(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	space, err := h.service.GetCustomerSpace(ctx, tenantID, userID, c.Param("id"), strings.TrimSpace(c.Query("profile_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": space})
}

func (h *ServiceHandler) ListAgentTemplates(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": h.service.ListAgentTemplates(c.Request.Context())})
}

func (h *ServiceHandler) ListReminders(c *gin.Context) {
	ctx := c.Request.Context()
	query, ok := h.listQuery(c)
	if !ok {
		return
	}
	query.Status = strings.TrimSpace(c.Query("status"))
	query.AgentDomain = strings.TrimSpace(c.Query("agent_domain"))
	query.ProfileID = strings.TrimSpace(c.Query("profile_id"))
	query.MemoryID = strings.TrimSpace(c.Query("memory_id"))
	reminders, total, err := h.service.ListReminders(ctx, query)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, listPayload(reminders, total, query.Page, query.PageSize))
}

func (h *ServiceHandler) GetReminder(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	reminder, err := h.service.GetReminder(ctx, tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceHandler) UpdateReminderStatus(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceReminderStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	reminder, err := h.service.UpdateReminderStatus(ctx, tenantID, userID, c.Param("id"), req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": reminder})
}

func (h *ServiceHandler) ListActionDrafts(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	drafts, err := h.service.ListActionDrafts(ctx, tenantID, userID, strings.TrimSpace(c.Query("reminder_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": drafts})
}

func (h *ServiceHandler) CreateActionDraft(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.AgentActionDraftInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.CreateActionDraft(ctx, tenantID, userID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ServiceHandler) UpdateActionDraftStatus(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.AgentActionDraftStatusInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	draft, err := h.service.UpdateActionDraftStatus(ctx, tenantID, userID, c.Param("id"), req.Status)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": draft})
}

func (h *ServiceHandler) ListWorkProfiles(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	profiles, err := h.service.ListWorkProfiles(ctx, tenantID, strings.TrimSpace(c.Query("user_id")))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profiles})
}

func (h *ServiceHandler) CreateWorkProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceWorkProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	profile, err := h.service.CreateWorkProfile(ctx, tenantID, operatorUserID, req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceHandler) UpdateWorkProfile(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.ServiceWorkProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	profile, err := h.service.UpdateWorkProfile(ctx, tenantID, operatorUserID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": profile})
}

func (h *ServiceHandler) ListAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, _, ok := serviceScope(c)
	if !ok {
		return
	}
	settings, err := h.service.ListAgentSettings(ctx, tenantID, c.Param("id"), strings.TrimSpace(c.Query("enabled")) == "true")
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

func (h *ServiceHandler) ReplaceAgentSettings(c *gin.Context) {
	ctx := c.Request.Context()
	tenantID, operatorUserID, ok := serviceScope(c)
	if !ok {
		return
	}
	var req types.WorkProfileAgentSettingsInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid request body").WithDetails(err.Error()))
		return
	}
	settings, err := h.service.ReplaceAgentSettings(ctx, tenantID, operatorUserID, c.Param("id"), req)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": settings})
}

func (h *ServiceHandler) listQuery(c *gin.Context) (types.ServiceListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Keyword:  keyword,
		Page:     page,
		PageSize: pageSize,
	}, true
}

func (h *ServiceHandler) dailyReportListQuery(c *gin.Context) (types.ServiceDailyReportListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceDailyReportListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceDailyReportListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceDailyReportListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: strings.TrimSpace(c.Query("profile_id")),
		Range:     strings.TrimSpace(c.Query("range")),
		Keyword:   keyword,
		Page:      page,
		PageSize:  pageSize,
	}, true
}

func (h *ServiceHandler) customerSpaceListQuery(c *gin.Context) (types.ServiceCustomerSpaceListQuery, bool) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return types.ServiceCustomerSpaceListQuery{}, false
	}
	page, pageSize, ok := parseServicePagination(c)
	if !ok {
		return types.ServiceCustomerSpaceListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("q"))
	if keyword == "" {
		keyword = strings.TrimSpace(c.Query("keyword"))
	}
	return types.ServiceCustomerSpaceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: strings.TrimSpace(c.Query("profile_id")),
		Keyword:   keyword,
		Page:      page,
		PageSize:  pageSize,
	}, true
}

func parseServicePagination(c *gin.Context) (int, int, bool) {
	page := 1
	pageSize := 20
	if raw := strings.TrimSpace(c.Query("page")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			c.Error(apperrors.NewBadRequestError("page must be a positive integer"))
			return 0, 0, false
		}
		page = parsed
	}
	if raw := strings.TrimSpace(c.Query("page_size")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
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

func (h *ServiceHandler) handleError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, appsvc.ErrServiceInvalidScope):
		c.Error(apperrors.NewUnauthorizedError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case stderrors.Is(err, appsvc.ErrServiceProfileNameRequired),
		stderrors.Is(err, appsvc.ErrServiceInvalidProfileState),
		stderrors.Is(err, appsvc.ErrServiceInvalidAgentDomain),
		stderrors.Is(err, appsvc.ErrServiceInvalidReportRange),
		stderrors.Is(err, appsvc.ErrServiceInvalidReportDate),
		stderrors.Is(err, appsvc.ErrServiceInvalidStatus),
		stderrors.Is(err, appsvc.ErrServiceProfileNotConfigured),
		stderrors.Is(err, appsvc.ErrAgentRunInvalidRequest),
		stderrors.Is(err, appsvc.ErrAgentRunCannotCancel),
		stderrors.Is(err, appsvc.ErrAgentRunNotWaitingInput),
		stderrors.Is(err, appsvc.ErrAgentRunCannotRegenerate):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	case stderrors.Is(err, appsvc.ErrAgentRunNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	default:
		logger.ErrorWithFields(c.Request.Context(), err, nil)
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
