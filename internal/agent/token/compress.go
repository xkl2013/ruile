package token

import (
	"github.com/Tencent/WeKnora/internal/models/chat"
)

// DefaultContextThresholdRatio is the ratio of context window usage that triggers compression.
// When message tokens exceed MaxContextTokens * threshold, old messages are trimmed.
const DefaultContextThresholdRatio = 0.8

// CompressContext trims the oldest logical message groups to bring total token
// count below the threshold. It preserves the system prompt and the current
// user query. When the current turn itself is the only large part of the
// context, older assistant/tool groups from that turn may also be removed, but
// tool_call/tool_result groups are never split.
//
// currentTokens is the caller's best estimate of the current context size.
func CompressContext(
	messages []chat.Message,
	estimator *Estimator,
	maxTokens int,
	currentTokens int,
) []chat.Message {
	if maxTokens <= 0 || len(messages) <= 2 {
		return messages
	}

	threshold := int(float64(maxTokens) * DefaultContextThresholdRatio)
	if currentTokens <= threshold {
		return messages
	}

	systemMsg := messages[0]

	// Find the current user query — the last message with role "user".
	lastUserIdx := -1
	for i := len(messages) - 1; i >= 1; i-- {
		if messages[i].Role == "user" {
			lastUserIdx = i
			break
		}
	}
	if lastUserIdx < 0 {
		return messages
	}

	history := messages[1:lastUserIdx]
	tail := messages[lastUserIdx:]

	tokensToFree := currentTokens - threshold
	freed := 0
	removeHistoryUpTo := 0
	historyGroups := groupToolMessages(history)
	for i, group := range historyGroups {
		groupTokens := 0
		for _, msg := range group {
			groupTokens += estimator.EstimateMessage(&msg)
		}
		freed += groupTokens
		removeHistoryUpTo = i + 1
		if freed >= tokensToFree {
			break
		}
	}

	remaining := make([]chat.Message, 0, len(messages))
	remaining = append(remaining, systemMsg)
	for i := removeHistoryUpTo; i < len(historyGroups); i++ {
		remaining = append(remaining, historyGroups[i]...)
	}

	// A long ReAct turn can contain many assistant/tool rounds after the
	// current user message. If history did not free enough space, trim the
	// oldest complete tool groups inside the current turn while keeping the
	// user query itself and the newest tool context.
	if freed < tokensToFree && len(tail) > 1 {
		tailGroups := groupToolMessages(tail[1:])
		removeTailUpTo := 0
		for i, group := range tailGroups {
			groupTokens := 0
			for _, msg := range group {
				groupTokens += estimator.EstimateMessage(&msg)
			}
			freed += groupTokens
			removeTailUpTo = i + 1
			if freed >= tokensToFree {
				break
			}
		}
		remaining = append(remaining, tail[0])
		for i := removeTailUpTo; i < len(tailGroups); i++ {
			remaining = append(remaining, tailGroups[i]...)
		}
		return remaining
	}
	remaining = append(remaining, tail...)

	return remaining
}

// groupToolMessages groups middle messages into logical units:
//   - An assistant message with tool_calls + its corresponding tool result messages = one group
//   - A standalone message (user, assistant without tool_calls) = one group
//
// This ensures tool_call/tool_result pairs are never split during compression.
func groupToolMessages(messages []chat.Message) [][]chat.Message {
	var groups [][]chat.Message
	i := 0
	for i < len(messages) {
		msg := messages[i]

		// If this is an assistant message with tool_calls, group it with following tool results
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			group := []chat.Message{msg}
			i++
			// Collect all following tool result messages
			for i < len(messages) && messages[i].Role == "tool" {
				group = append(group, messages[i])
				i++
			}
			groups = append(groups, group)
		} else {
			groups = append(groups, []chat.Message{msg})
			i++
		}
	}
	return groups
}
