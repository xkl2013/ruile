package handler

import (
	"context"
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

type AgentRunHandler struct {
	agentRuns   interfaces.AgentRunService
	fileService interfaces.FileService
}

func NewAgentRunHandler(
	agentRuns interfaces.AgentRunService,
	fileService interfaces.FileService,
) *AgentRunHandler {
	return &AgentRunHandler{agentRuns: agentRuns, fileService: fileService}
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

// ServiceSessionScopeMiddleware authorizes service-scoped chat requests before
// the legacy session/message handlers run. The header is deliberately opt-in:
// global chats keep their existing per-user scope, while service chats can be
// shared with active members of the owning service space.
func ServiceSessionScopeMiddleware(service interfaces.ServiceSpaceService) gin.HandlerFunc {
	return func(c *gin.Context) {
		serviceID := strings.TrimSpace(c.GetHeader("X-Service-ID"))
		if service == nil || serviceID == "" {
			c.Next()
			return
		}

		sessionID := strings.TrimSpace(c.Param("session_id"))
		if sessionID == "" {
			sessionID = strings.TrimSpace(c.Param("id"))
		}
		if sessionID == "" {
			c.Next()
			return
		}

		tenantID, userID, ok := serviceScope(c)
		if !ok {
			return
		}
		write := c.Request.Method != http.MethodGet
		minimumRole := types.ServiceMemberRoleViewer
		if write {
			minimumRole = types.ServiceMemberRoleEditor
		}
		if _, err := service.Authorize(c.Request.Context(), tenantID, userID, serviceID, minimumRole, write); err != nil {
			switch {
			case stderrors.Is(err, appsvc.ErrServiceSpaceForbidden):
				c.Error(apperrors.NewForbiddenError(err.Error()))
			case stderrors.Is(err, appsvc.ErrServiceSpaceArchived),
				stderrors.Is(err, appsvc.ErrServiceSpaceNotActive):
				c.Error(apperrors.NewBadRequestError(err.Error()))
			default:
				c.Error(apperrors.NewNotFoundError(err.Error()))
			}
			return
		}
		if _, err := service.GetSession(c.Request.Context(), tenantID, userID, serviceID, sessionID); err != nil {
			c.Error(apperrors.NewNotFoundError(err.Error()))
			return
		}

		ctx := context.WithValue(c.Request.Context(), types.ServiceSessionIDContextKey, serviceID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func (h *AgentRunHandler) Get(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	run, err := h.agentRuns.GetAgentRun(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *AgentRunHandler) ListThread(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	runs, err := h.agentRuns.ListAgentThreadRuns(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": runs})
}

func (h *AgentRunHandler) Quality(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	quality, err := h.agentRuns.GetAgentRunQuality(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": quality})
}

func (h *AgentRunHandler) Steps(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	steps, err := h.agentRuns.ListAgentRunSteps(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": steps})
}

func (h *AgentRunHandler) Answers(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.AgentRunAnswersInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.Error(apperrors.NewBadRequestError("invalid agent run answers").WithDetails(err.Error()))
		return
	}
	run, err := h.agentRuns.SubmitAgentRunAnswers(c.Request.Context(), tenantID, userID, c.Param("id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *AgentRunHandler) Regenerate(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	var input types.AgentRunRegenerateInput
	if err := c.ShouldBindJSON(&input); err != nil && !stderrors.Is(err, io.EOF) {
		c.Error(apperrors.NewBadRequestError("invalid agent run regeneration request").WithDetails(err.Error()))
		return
	}
	run, err := h.agentRuns.RegenerateAgentRun(c.Request.Context(), tenantID, userID, c.Param("id"), input)
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"success": true, "data": run})
}

func (h *AgentRunHandler) Cancel(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	run, err := h.agentRuns.CancelAgentRun(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": run})
}

func (h *AgentRunHandler) StreamEvents(c *gin.Context) {
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
		events, listErr := h.agentRuns.ListAgentRunEvents(ctx, tenantID, userID, runID, afterSequence, 100)
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

func (h *AgentRunHandler) PreviewArtifact(c *gin.Context) {
	h.streamArtifact(c, false)
}

func (h *AgentRunHandler) DownloadArtifact(c *gin.Context) {
	h.streamArtifact(c, true)
}

func (h *AgentRunHandler) streamArtifact(c *gin.Context, download bool) {
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
	artifact := findAgentRunArtifact(run, c.Param("artifact_id"))
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

	contentType := firstNonEmptyHandler(artifact.MimeType, "application/octet-stream")
	disposition := "inline"
	if download {
		disposition = "attachment"
	}
	fileName := firstNonEmptyHandler(artifact.OriginalName, artifact.Title, "agent-artifact")
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf(`%s; filename="%s"`, disposition, sanitizeDownloadName(fileName)))
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "private, max-age=300")
	c.Status(http.StatusOK)
	if _, err := io.Copy(c.Writer, reader); err != nil {
		logger.Warnf(ctx, "failed to stream agent artifact: run_id=%s artifact_id=%s err=%v",
			run.ID, artifact.ID, err)
	}
}

func (h *AgentRunHandler) Diff(c *gin.Context) {
	tenantID, userID, ok := serviceScope(c)
	if !ok {
		return
	}
	current, err := h.agentRuns.GetAgentRun(c.Request.Context(), tenantID, userID, c.Param("id"))
	if err != nil {
		h.handleError(c, err)
		return
	}
	parentID := strings.TrimSpace(current.ParentRunID)
	if parentID == "" {
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
			"changed": false,
			"before":  "",
			"after":   agentRunReportText(current),
		}})
		return
	}
	parent, err := h.agentRuns.GetAgentRun(c.Request.Context(), tenantID, userID, parentID)
	if err != nil {
		h.handleError(c, err)
		return
	}
	before := agentRunReportText(parent)
	after := agentRunReportText(current)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"changed":       before != after,
		"parent_run_id": parent.ID,
		"run_id":        current.ID,
		"before":        before,
		"after":         after,
	}})
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

func findAgentRunArtifact(run *types.AgentRun, artifactID string) *types.AgentArtifactResultV1 {
	if run == nil {
		return nil
	}
	raw, err := json.Marshal(run.Result["artifacts"])
	if err != nil {
		return nil
	}
	var artifacts []types.AgentArtifactResultV1
	if json.Unmarshal(raw, &artifacts) != nil {
		return nil
	}
	artifactID = strings.TrimSpace(artifactID)
	for index := range artifacts {
		if artifacts[index].ID == artifactID {
			return &artifacts[index]
		}
	}
	return nil
}

func agentRunReportText(run *types.AgentRun) string {
	if run == nil {
		return ""
	}
	if value, ok := run.Result["report_markdown"].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	raw, err := json.Marshal(run.Result["artifacts"])
	if err != nil {
		return ""
	}
	var artifacts []types.AgentArtifactResultV1
	if json.Unmarshal(raw, &artifacts) != nil {
		return ""
	}
	for _, artifact := range artifacts {
		if artifact.Role != types.AgentArtifactRolePrimary {
			continue
		}
		if report, err := types.DecodeStructuredReportV1(artifact.Content); err == nil {
			var builder strings.Builder
			builder.WriteString(report.Title)
			builder.WriteString("\n\n")
			builder.WriteString(report.ExecutiveSummary)
			for _, section := range report.Sections {
				builder.WriteString("\n\n## ")
				builder.WriteString(section.Title)
				if strings.TrimSpace(section.Content) != "" {
					builder.WriteString("\n")
					builder.WriteString(section.Content)
				}
				for _, item := range section.Items {
					builder.WriteString("\n- ")
					builder.WriteString(item)
				}
			}
			return strings.TrimSpace(builder.String())
		}
	}
	return ""
}

func sanitizeDownloadName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer(`"`, "", "\r", "", "\n", "", "/", "-", "\\", "-").Replace(value)
	if value == "" {
		return "agent-artifact"
	}
	return value
}

func firstNonEmptyHandler(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *AgentRunHandler) handleError(c *gin.Context, err error) {
	switch {
	case stderrors.Is(err, appsvc.ErrAgentRunInvalidScope):
		c.Error(apperrors.NewUnauthorizedError(err.Error()))
	case stderrors.Is(err, appsvc.ErrAgentRunNotFound):
		c.Error(apperrors.NewNotFoundError(err.Error()))
	case stderrors.Is(err, appsvc.ErrAgentRunInvalidRequest),
		stderrors.Is(err, appsvc.ErrAgentRunCannotCancel),
		stderrors.Is(err, appsvc.ErrAgentRunCannotRegenerate),
		stderrors.Is(err, appsvc.ErrAgentRunNotWaitingInput):
		c.Error(apperrors.NewBadRequestError(err.Error()))
	default:
		c.Error(apperrors.NewInternalServerError(err.Error()))
	}
}
