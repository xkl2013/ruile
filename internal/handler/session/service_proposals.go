package session

import (
	"context"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const serviceFactProposalHistoryLimit = 8

func (h *Handler) emitServiceFactProposal(
	streamCtx *sseStreamContext,
	reqCtx *qaRequestContext,
	sourceID string,
) {
	if h == nil || h.serviceSpace == nil || streamCtx == nil || reqCtx == nil || reqCtx.session == nil {
		return
	}
	serviceID := strings.TrimSpace(reqCtx.session.ServiceID)
	if serviceID == "" {
		return
	}
	userID := strings.TrimSpace(reqCtx.session.UserID)
	if userID == "" {
		userID = types.SessionOwnerIDFromContext(reqCtx.ctx)
	}
	if userID == "" {
		return
	}
	ctx := context.WithoutCancel(streamCtx.asyncCtx)
	proposalText := h.buildServiceFactProposalText(
		ctx,
		reqCtx,
		streamCtx.assistantMessage.ID,
	)
	proposal, err := h.serviceSpace.PreviewFactProposal(
		ctx,
		reqCtx.session.TenantID,
		userID,
		serviceID,
		types.ServiceFactProposalPreviewInput{
			SessionID:          reqCtx.sessionID,
			AssistantMessageID: streamCtx.assistantMessage.ID,
			Text:               proposalText,
			SourceType:         "chat_message",
			SourceID:           sourceID,
		},
	)
	if err != nil {
		logger.Warnf(ctx, "service fact proposal preview failed: service_id=%s session_id=%s err=%v", serviceID, reqCtx.sessionID, err)
		return
	}
	if proposal == nil {
		return
	}
	if err := h.streamManager.AppendEvent(ctx, reqCtx.sessionID, streamCtx.assistantMessage.ID, interfaces.StreamEvent{
		ID:        proposal.ID,
		Type:      types.ResponseTypeServiceUpdateProposal,
		Content:   "检测到可更新的服务档案信息",
		Done:      true,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"proposal": proposal,
		},
	}); err != nil {
		logger.Warnf(ctx, "service fact proposal stream event failed: service_id=%s proposal_id=%s err=%v", serviceID, proposal.ID, err)
	}
}

func (h *Handler) buildServiceFactProposalText(
	ctx context.Context,
	reqCtx *qaRequestContext,
	currentAssistantMessageID string,
) string {
	if reqCtx == nil {
		return ""
	}
	query := strings.TrimSpace(reqCtx.query)
	if h == nil || h.messageService == nil || reqCtx.session == nil ||
		!isServiceFactWriteConfirmation(query) {
		return query
	}
	historyCtx := context.WithValue(ctx, types.TenantIDContextKey, reqCtx.session.TenantID)
	messages, err := h.messageService.GetRecentMessagesBySession(
		historyCtx,
		reqCtx.sessionID,
		serviceFactProposalHistoryLimit,
	)
	if err != nil {
		logger.Warnf(ctx, "service fact proposal history lookup failed: session_id=%s err=%v", reqCtx.sessionID, err)
		return query
	}
	return composeServiceFactProposalText(query, messages, currentAssistantMessageID)
}

func composeServiceFactProposalText(
	query string,
	messages []*types.Message,
	currentAssistantMessageID string,
) string {
	query = strings.TrimSpace(query)
	if !isServiceFactWriteConfirmation(query) {
		return query
	}
	for index := len(messages) - 1; index >= 0; index-- {
		message := messages[index]
		if message == nil || message.Role != "assistant" || !message.IsCompleted ||
			message.ID == currentAssistantMessageID {
			continue
		}
		contextParts := make([]string, 0, 2)
		if userEvidence := previousTurnUserEvidence(messages, index, message.RequestID); userEvidence != "" {
			contextParts = append(contextParts, "上一轮用户提供的信息：\n"+userEvidence)
		}
		if assistantCandidates := extractConfirmedServiceFactLines(message.Content); assistantCandidates != "" {
			contextParts = append(contextParts, "上一轮助手列出的候选字段：\n"+assistantCandidates)
		}
		if len(contextParts) == 0 {
			return query
		}
		return query + "\n\n用户确认写入的上一轮候选信息：\n" + strings.Join(contextParts, "\n")
	}
	return query
}

func previousTurnUserEvidence(
	messages []*types.Message,
	assistantIndex int,
	requestID string,
) string {
	evidence := make([]string, 0)
	for index := assistantIndex - 1; index >= 0; index-- {
		message := messages[index]
		if message == nil {
			continue
		}
		if requestID != "" && message.RequestID != requestID {
			if len(evidence) > 0 {
				break
			}
			continue
		}
		if message.Role != "user" {
			continue
		}
		content := strings.TrimSpace(message.Content)
		if content != "" {
			evidence = append(evidence, content)
		}
		if requestID == "" {
			break
		}
	}
	for left, right := 0, len(evidence)-1; left < right; left, right = left+1, right-1 {
		evidence[left], evidence[right] = evidence[right], evidence[left]
	}
	return strings.Join(evidence, "\n")
}

func isServiceFactWriteConfirmation(text string) bool {
	normalized := strings.NewReplacer(
		" ", "",
		"\t", "",
		"\n", "",
		"。", "",
		"，", "",
		",", "",
		"！", "",
		"!", "",
		"？", "",
		"?", "",
	).Replace(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	for _, marker := range []string{
		"为什么", "怎么", "没有更新", "没更新", "未更新",
		"是否更新", "不要更新", "先别更新", "取消更新",
	} {
		if strings.Contains(normalized, marker) {
			return false
		}
	}
	if normalized == "确认" {
		return true
	}
	for _, marker := range []string{
		"更新档案", "确认更新", "确认写入", "写入档案",
		"保存到档案", "同步档案", "按这个更新", "按这些更新",
		"就按这个更新", "就按这些更新", "可以更新",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func extractConfirmedServiceFactLines(content string) string {
	content = strings.NewReplacer(
		"\r\n", "\n",
		"\r", "\n",
		"；", "\n",
		";", "\n",
	).Replace(content)
	lines := make([]string, 0)
	for _, rawLine := range strings.Split(content, "\n") {
		line := trimServiceFactListPrefix(rawLine)
		if line == "" || strings.ContainsAny(line, "？?") ||
			containsServiceFactPromptMarker(line) {
			continue
		}
		label, value, ok := splitServiceFactCandidateLine(line)
		if !ok {
			continue
		}
		lines = append(lines, label+"："+value)
		if len(lines) >= 20 {
			break
		}
	}
	return strings.Join(lines, "\n")
}

func trimServiceFactListPrefix(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimLeft(line, "-*•· ")
	for index, character := range line {
		if character < '0' || character > '9' {
			if index > 0 && (character == '.' || character == '、') {
				line = strings.TrimSpace(line[index+len(string(character)):])
			}
			break
		}
	}
	return strings.TrimSpace(line)
}

func containsServiceFactPromptMarker(line string) bool {
	for _, marker := range []string{
		"例如", "比如", "请补充", "需要补充", "还需要", "待补充",
		"可补充", "建议补充",
	} {
		if strings.Contains(line, marker) {
			return true
		}
	}
	return false
}

func splitServiceFactCandidateLine(line string) (string, string, bool) {
	separatorIndex := -1
	separatorSize := 0
	for _, separator := range []string{"：", ":", "＝", "="} {
		if index := strings.Index(line, separator); index >= 0 &&
			(separatorIndex < 0 || index < separatorIndex) {
			separatorIndex = index
			separatorSize = len(separator)
		}
	}
	if separatorIndex <= 0 {
		return "", "", false
	}
	label := strings.Trim(strings.TrimSpace(line[:separatorIndex]), "*_`# ")
	value := strings.Trim(strings.TrimSpace(line[separatorIndex+separatorSize:]), "*_` ")
	if label == "" || value == "" || len(label) > 40 || len(value) > 500 {
		return "", "", false
	}
	switch value {
	case "待补充", "未知", "未提供", "暂无":
		return "", "", false
	default:
		return label, value, true
	}
}
