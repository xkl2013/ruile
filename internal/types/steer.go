package types

import "context"

// SteerMessageContent wraps a mid-run user message with explicit continuation
// guidance for the model. The original text is persisted separately.
func SteerMessageContent(content string) string {
	return "<steer_message>\n" + content + "\n</steer_message>\n<continue_task>\n" +
		"This is guidance for the task in progress. Apply it and continue unfinished work " +
		"unless the user explicitly changes or cancels the task.\n</continue_task>"
}

// SteerSink connects the agent loop to the handler-owned steer queue.
type SteerSink interface {
	PollSteer(ctx context.Context, sessionID, messageID string, lastOffset int) ([]map[string]interface{}, int, error)
	PersistSteerMessage(
		ctx context.Context,
		sessionID, messageID, steerID, content string,
		mentionedItems MentionedItems,
		channel string,
	) string
}
