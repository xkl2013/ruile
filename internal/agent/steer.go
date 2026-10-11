package agent

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

func sanitizeSteerContent(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return strings.TrimSpace(content)
}

// SetSteerSink wires the optional handler-owned steer queue into this run.
func (e *AgentEngine) SetSteerSink(sink types.SteerSink) {
	e.steerSink = sink
}

// drainSteerMessages persists and appends messages queued while the agent was
// thinking or executing tools. Persistence happens first so failed writes are
// retried at the next round boundary.
func (e *AgentEngine) drainSteerMessages(
	ctx context.Context,
	state *types.AgentState,
	messagesPtr *[]chat.Message,
	sessionID, messageID string,
) int {
	if e.steerSink == nil {
		return 0
	}
	events, _, err := e.steerSink.PollSteer(ctx, sessionID, messageID, 0)
	if err != nil {
		logger.Warnf(ctx, "[Agent] Steer poll failed at round %d: %v", state.CurrentRound+1, err)
		return 0
	}
	injected := 0
	for _, raw := range events {
		content := sanitizeSteerContent(types.MapString(raw, "content"))
		if content == "" {
			continue
		}
		steerID := types.MapString(raw, "id")
		userMessageID := e.steerSink.PersistSteerMessage(
			ctx, sessionID, messageID, steerID, content,
			types.MentionedItemsFromRaw(raw["mentioned_items"]),
			types.MapString(raw, "channel"),
		)
		if userMessageID == "" {
			logger.Warnf(ctx, "[Agent] Steer persist failed for %s; leaving event pending", steerID)
			continue
		}
		*messagesPtr = append(*messagesPtr, chat.Message{
			Role:    "user",
			Content: types.SteerMessageContent(content),
		})
		state.PendingSteerMessages = append(state.PendingSteerMessages, userMessageID)
		_ = e.eventBus.Emit(ctx, event.Event{
			ID:        generateEventID("injected"),
			Type:      event.EventUserMessageInjected,
			SessionID: sessionID,
			Data: event.UserMessageInjectedData{
				SteerID:       steerID,
				Content:       content,
				MessageID:     messageID,
				UserMessageID: userMessageID,
			},
		})
		injected++
	}
	return injected
}
