package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	steerDeliveryInject    = "inject"
	steerDeliveryAfter     = "after"
	steerDataConsumed      = "consumed"
	steerDataUserMessageID = "user_message_id"
)

const (
	maxSteerQueueDepth  = 10
	maxSteerQueryLength = 10000
)

// SteerMessageRequest is the payload for a mid-run user message.
type SteerMessageRequest struct {
	ExpectedAssistantMessageID string                 `json:"expected_assistant_message_id,omitempty"`
	SteerID                    string                 `json:"steer_id,omitempty"`
	Query                      string                 `json:"query" binding:"required"`
	MentionedItems             []MentionedItemRequest `json:"mentioned_items,omitempty"`
	Channel                    string                 `json:"channel,omitempty"`
	Delivery                   string                 `json:"delivery,omitempty"`
}

type steerSink struct {
	ctx               context.Context
	sessionID         string
	requestID         string
	messageService    interfaces.MessageService
	streamManager     interfaces.StreamManager
	mu                sync.Mutex
	lastUserMessageID string
	drainedOffset     int
	injectedIDs       map[string]struct{}
}

func newSteerSink(
	ctx context.Context,
	sessionID, requestID string,
	_ *types.Message,
	messageService interfaces.MessageService,
	streamManager interfaces.StreamManager,
) *steerSink {
	return &steerSink{
		ctx:            ctx,
		sessionID:      sessionID,
		requestID:      requestID,
		messageService: messageService,
		streamManager:  streamManager,
		injectedIDs:    make(map[string]struct{}),
	}
}

func (s *steerSink) PollSteer(
	ctx context.Context, sessionID, messageID string, lastOffset int,
) ([]map[string]interface{}, int, error) {
	// Read from the beginning so a message promoted from "after" to "inject"
	// after a previous poll is still visible to the running engine.
	events, total, err := s.streamManager.GetSteerEvents(ctx, sessionID, messageID, 0)
	if err != nil {
		return nil, lastOffset, err
	}
	out := make([]map[string]interface{}, 0, len(events))
	for _, item := range events {
		if delivery, _ := item.Data["delivery"].(string); delivery == steerDeliveryAfter {
			continue
		}
		if s.hasInjected(item.ID) {
			continue
		}
		if consumed, _ := item.Data[steerDataConsumed].(bool); consumed {
			continue
		}
		s.markInjected(item.ID)
		out = append(out, map[string]interface{}{
			"id":              item.ID,
			"content":         item.Content,
			"mentioned_items": item.Data["mentioned_items"],
			"channel":         item.Data["channel"],
		})
	}
	s.mu.Lock()
	if total > s.drainedOffset {
		s.drainedOffset = total
	}
	s.mu.Unlock()
	return out, total, nil
}

func (s *steerSink) hasInjected(id string) bool {
	if id == "" {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.injectedIDs[id]
	return ok
}

func (s *steerSink) markInjected(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.injectedIDs == nil {
		s.injectedIDs = make(map[string]struct{})
	}
	s.injectedIDs[id] = struct{}{}
}

func (s *steerSink) unmarkInjected(id string) {
	if id == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.injectedIDs, id)
}

func (s *steerSink) PersistSteerMessage(
	ctx context.Context,
	sessionID, messageID, steerID, content string,
	mentionedItems types.MentionedItems,
	channel string,
) string {
	if s.messageService == nil {
		s.unmarkInjected(steerID)
		return ""
	}
	if existing := s.persistedUserMessageID(ctx, sessionID, messageID, steerID); existing != "" {
		s.mu.Lock()
		s.lastUserMessageID = existing
		s.mu.Unlock()
		return existing
	}
	if channel == "" {
		channel = "web"
	}
	msg, err := s.messageService.CreateMessage(ctx, &types.Message{
		SessionID:      sessionID,
		Role:           "user",
		Content:        content,
		RequestID:      s.requestID,
		MentionedItems: mentionedItems,
		CreatedAt:      time.Now(),
		IsCompleted:    true,
		Channel:        channel,
	})
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"session_id": sessionID,
			"steer_id":   steerID,
		})
		s.unmarkInjected(steerID)
		return ""
	}
	updated, err := s.streamManager.UpdateSteerEventData(ctx, sessionID, messageID, steerID,
		map[string]interface{}{
			steerDataConsumed:      true,
			steerDataUserMessageID: msg.ID,
		})
	if err != nil || !updated {
		if err != nil {
			logger.Warnf(ctx, "Failed to mark steer %s consumed: %v", steerID, err)
		}
		_ = s.messageService.DeleteMessage(ctx, sessionID, msg.ID)
		s.unmarkInjected(steerID)
		if existing := s.persistedUserMessageID(ctx, sessionID, messageID, steerID); existing != "" {
			s.mu.Lock()
			s.lastUserMessageID = existing
			s.mu.Unlock()
			return existing
		}
		return ""
	}
	s.mu.Lock()
	s.lastUserMessageID = msg.ID
	s.mu.Unlock()
	return msg.ID
}

func (s *steerSink) persistedUserMessageID(
	ctx context.Context, sessionID, messageID, steerID string,
) string {
	if s.streamManager == nil || steerID == "" {
		return ""
	}
	events, _, err := s.streamManager.GetSteerEvents(ctx, sessionID, messageID, 0)
	if err != nil {
		return ""
	}
	for _, item := range events {
		if item.ID != steerID || item.Data == nil {
			continue
		}
		if userMessageID, ok := item.Data[steerDataUserMessageID].(string); ok {
			return userMessageID
		}
	}
	return ""
}

// InjectedIDs returns the steer IDs already handed to the engine in this run.
func (s *steerSink) InjectedIDs() map[string]struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]struct{}, len(s.injectedIDs))
	for id := range s.injectedIDs {
		out[id] = struct{}{}
	}
	return out
}

func steerEvent(id, query string, mentionedItems types.MentionedItems, channel, delivery string) interfaces.StreamEvent {
	if channel == "" {
		channel = "web"
	}
	return interfaces.StreamEvent{
		ID:        id,
		Type:      types.ResponseTypeSteer,
		Content:   query,
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"steer_id":        id,
			"delivery":        delivery,
			"channel":         channel,
			"mentioned_items": types.MentionedItemsToRaw(mentionedItems),
		},
	}
}

func parseSteerDelivery(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", steerDeliveryAfter:
		return steerDeliveryAfter, nil
	case steerDeliveryInject:
		return steerDeliveryInject, nil
	default:
		return "", fmt.Errorf("invalid delivery %q", value)
	}
}

func (h *Handler) liveAgentRun(ctx context.Context, sessionID string) (string, error) {
	assistantID, _, err := h.streamManager.GetLiveRun(ctx, sessionID)
	if err != nil || assistantID == "" {
		return assistantID, err
	}
	message, err := h.messageService.GetMessage(ctx, sessionID, assistantID)
	if err != nil {
		return "", err
	}
	if message == nil || message.IsCompleted {
		_ = h.streamManager.ClearLiveRun(ctx, sessionID, assistantID)
		return "", nil
	}
	return assistantID, nil
}

func (h *Handler) resolveLiveAgentRun(ctx context.Context, c *gin.Context, sessionID string) (string, bool) {
	assistantID, err := h.liveAgentRun(ctx, sessionID)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{"session_id": sessionID})
		_ = c.Error(errors.NewServiceUnavailableError("Failed to look up running turn"))
		return "", false
	}
	return assistantID, true
}

// SteerMessage queues a message for the active Agent turn. Delivery "inject"
// is consumed at the next ReAct boundary; "after" remains available to the
// client for a follow-up turn after the current run finishes.
func (h *Handler) SteerMessage(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	sessionID := secutils.SanitizeForLog(c.Param("session_id"))
	if sessionID == "" {
		_ = c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}
	if _, err := h.sessionService.GetSession(ctx, sessionID); err != nil {
		_ = c.Error(errors.NewNotFoundError("Session not found"))
		return
	}
	var req SteerMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	query := strings.TrimSpace(req.Query)
	if query == "" {
		_ = c.Error(errors.NewBadRequestError("query must not be empty"))
		return
	}
	if len([]rune(query)) > maxSteerQueryLength {
		_ = c.Error(errors.NewBadRequestError("query too long"))
		return
	}
	delivery, err := parseSteerDelivery(req.Delivery)
	if err != nil {
		_ = c.Error(errors.NewBadRequestError(err.Error()))
		return
	}
	assistantID, ok := h.resolveLiveAgentRun(ctx, c, sessionID)
	if !ok {
		return
	}
	if req.ExpectedAssistantMessageID != "" && assistantID != "" &&
		req.ExpectedAssistantMessageID != assistantID {
		_ = c.Error(errors.NewConflictError("The active turn changed; retry the message"))
		return
	}
	if assistantID == "" {
		c.JSON(200, gin.H{"success": true, "status": "new_run"})
		return
	}

	events, _, err := h.streamManager.GetSteerEvents(ctx, sessionID, assistantID, 0)
	if err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to check steer queue"))
		return
	}
	steerID := req.SteerID
	if steerID == "" {
		steerID = uuid.NewString()
	} else if _, err := uuid.Parse(steerID); err != nil {
		_ = c.Error(errors.NewBadRequestError("invalid steer_id"))
		return
	}
	pending := 0
	for _, item := range events {
		if item.ID == steerID {
			if item.Content != query {
				_ = c.Error(errors.NewConflictError("steer_id already belongs to another message"))
				return
			}
			if consumed, _ := item.Data[steerDataConsumed].(bool); consumed {
				c.JSON(200, gin.H{
					"success": true, "status": "already_injected", "steer_id": steerID,
					"assistant_message_id": assistantID,
				})
				return
			}
			c.JSON(200, gin.H{
				"success": true, "status": "queued", "steer_id": steerID,
				"assistant_message_id": assistantID,
				"delivery":             item.Data["delivery"],
			})
			return
		}
		if consumed, _ := item.Data[steerDataConsumed].(bool); !consumed {
			pending++
		}
	}
	if pending >= maxSteerQueueDepth {
		_ = c.Error(errors.NewBadRequestError("too many queued messages for the running turn"))
		return
	}
	if err := h.streamManager.AppendSteerEvents(ctx, sessionID, assistantID, []interfaces.StreamEvent{
		steerEvent(steerID, query, convertMentionedItems(req.MentionedItems), req.Channel, delivery),
	}); err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to queue message"))
		return
	}
	c.JSON(200, gin.H{
		"success": true, "status": "queued", "steer_id": steerID,
		"delivery": delivery, "assistant_message_id": assistantID,
	})
}

func (h *Handler) PromoteSteerMessage(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	sessionID := secutils.SanitizeForLog(c.Param("session_id"))
	steerID := secutils.SanitizeForLog(c.Param("steer_id"))
	if sessionID == "" || steerID == "" {
		_ = c.Error(errors.NewBadRequestError(errors.ErrInvalidSessionID.Error()))
		return
	}
	assistantID, ok := h.resolveLiveAgentRun(ctx, c, sessionID)
	if !ok {
		return
	}
	if assistantID == "" {
		c.JSON(200, gin.H{"success": true, "status": "new_run"})
		return
	}
	events, _, err := h.streamManager.GetSteerEvents(ctx, sessionID, assistantID, 0)
	if err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to update queued message"))
		return
	}
	for _, item := range events {
		if item.ID != steerID {
			continue
		}
		if consumed, _ := item.Data[steerDataConsumed].(bool); consumed {
			c.JSON(200, gin.H{
				"success": true, "status": "already_injected", "steer_id": steerID,
				"assistant_message_id": assistantID,
			})
			return
		}
		break
	}
	updated, err := h.streamManager.UpdateSteerEventData(ctx, sessionID, assistantID, steerID,
		map[string]interface{}{"delivery": steerDeliveryInject})
	if err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to update queued message"))
		return
	}
	if !updated {
		_ = c.Error(errors.NewNotFoundError("Queued message not found"))
		return
	}
	c.JSON(200, gin.H{
		"success": true, "status": "queued", "steer_id": steerID,
		"delivery": steerDeliveryInject, "assistant_message_id": assistantID,
	})
}

func (h *Handler) ListSteerMessages(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	sessionID := secutils.SanitizeForLog(c.Param("id"))
	if sessionID == "" {
		sessionID = secutils.SanitizeForLog(c.Param("session_id"))
	}
	assistantID, ok := h.resolveLiveAgentRun(ctx, c, sessionID)
	if !ok {
		return
	}
	events := []interfaces.StreamEvent(nil)
	var err error
	active := assistantID != ""
	if assistantID == "" {
		assistantID, events, err = h.findPendingSteerRun(ctx, sessionID)
		if err != nil {
			_ = c.Error(errors.NewInternalServerError("Failed to load queued messages"))
			return
		}
		if assistantID == "" {
			c.JSON(200, gin.H{
				"success": true, "active": false,
				"items": []map[string]interface{}{},
			})
			return
		}
	} else {
		events, _, err = h.streamManager.GetSteerEvents(ctx, sessionID, assistantID, 0)
		if err != nil {
			_ = c.Error(errors.NewInternalServerError("Failed to load queued messages"))
			return
		}
	}
	items := make([]map[string]interface{}, 0, len(events))
	for _, item := range events {
		if consumed, _ := item.Data[steerDataConsumed].(bool); consumed {
			continue
		}
		items = append(items, map[string]interface{}{
			"steer_id":        item.ID,
			"content":         item.Content,
			"delivery":        item.Data["delivery"],
			"mentioned_items": item.Data["mentioned_items"],
		})
	}
	c.JSON(200, gin.H{
		"success":              true,
		"active":               active,
		"assistant_message_id": assistantID,
		"items":                items,
	})
}

func (h *Handler) DeleteSteerMessage(c *gin.Context) {
	ctx := logger.CloneContext(c.Request.Context())
	sessionID := secutils.SanitizeForLog(c.Param("id"))
	if sessionID == "" {
		sessionID = secutils.SanitizeForLog(c.Param("session_id"))
	}
	steerID := secutils.SanitizeForLog(c.Param("steer_id"))
	assistantID, ok := h.resolveLiveAgentRun(ctx, c, sessionID)
	if !ok {
		return
	}
	var err error
	if assistantID == "" {
		assistantID, _, err = h.findPendingSteerRun(ctx, sessionID)
		if err != nil {
			_ = c.Error(errors.NewInternalServerError("Failed to find queued message"))
			return
		}
		if assistantID == "" {
			c.JSON(200, gin.H{"success": true, "status": "gone", "removed": false})
			return
		}
	}
	events, _, err := h.streamManager.GetSteerEvents(ctx, sessionID, assistantID, 0)
	if err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to delete queued message"))
		return
	}
	for _, item := range events {
		if item.ID != steerID {
			continue
		}
		if consumed, _ := item.Data[steerDataConsumed].(bool); consumed {
			c.JSON(200, gin.H{
				"success": true, "status": "already_injected", "removed": false,
				"steer_id": steerID,
			})
			return
		}
		break
	}
	removed, err := h.streamManager.DeleteSteerEvent(ctx, sessionID, assistantID, steerID)
	if err != nil {
		_ = c.Error(errors.NewInternalServerError("Failed to delete queued message"))
		return
	}
	c.JSON(200, gin.H{"success": true, "status": "deleted", "removed": removed, "steer_id": steerID})
}

// findPendingSteerRun provides a short handoff window for a browser that
// reloads after the run completed but before its queued "after" messages were
// sent. The live marker is intentionally absent in this state, so inspect the
// newest assistant rows and return the first queue that still has work.
func (h *Handler) findPendingSteerRun(
	ctx context.Context, sessionID string,
) (string, []interfaces.StreamEvent, error) {
	messages, err := h.messageService.GetRecentMessagesBySession(ctx, sessionID, 100)
	if err != nil {
		return "", nil, err
	}
	for i := len(messages) - 1; i >= 0; i-- {
		message := messages[i]
		if message == nil || message.Role != "assistant" || message.ID == "" {
			continue
		}
		events, _, streamErr := h.streamManager.GetSteerEvents(ctx, sessionID, message.ID, 0)
		if streamErr != nil {
			return "", nil, streamErr
		}
		for _, item := range events {
			if consumed, _ := item.Data[steerDataConsumed].(bool); !consumed {
				return message.ID, events, nil
			}
		}
	}
	return "", nil, nil
}
