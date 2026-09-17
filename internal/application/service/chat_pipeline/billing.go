package chatpipeline

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const billingFinalizeTimeout = 10 * time.Second

func estimateMessageTokens(messages []chat.Message) int64 {
	var runes int64
	for _, message := range messages {
		runes += int64(utf8.RuneCountInString(message.Content))
		for _, part := range message.MultiContent {
			runes += int64(utf8.RuneCountInString(part.Text))
		}
	}
	return estimateRuneTokens(runes)
}

func estimateRuneTokens(runes int64) int64 {
	if runes <= 0 {
		return 0
	}
	// Provider tokenizers differ. Four Unicode code points per token is a
	// transport-independent fallback used only when exact provider usage is
	// unavailable.
	return (runes + 3) / 4
}

func estimatedOutputTokens(options *chat.ChatOptions) int64 {
	if options != nil {
		if options.MaxCompletionTokens > 0 {
			return int64(options.MaxCompletionTokens)
		}
		if options.MaxTokens > 0 {
			return int64(options.MaxTokens)
		}
	}
	return 4096
}

func usageRefNo(
	chatManage *types.ChatManage,
	eventType types.EventType,
	tenantID uint64,
	modelID string,
) string {
	messageID := chatManage.MessageID
	if messageID == "" {
		messageID = chatManage.UserMessageID
	}
	if messageID == "" {
		messageID = uuid.NewString()
	}
	return fmt.Sprintf("%s:%d:%s:%s", eventType, tenantID, messageID, modelID)
}

func beginChatModelBilling(
	ctx context.Context,
	usageBilling interfaces.UsageBillingService,
	eventType types.EventType,
	chatManage *types.ChatManage,
	chatModel chat.Chat,
	messages []chat.Message,
	options *chat.ChatOptions,
) (*types.BillingUsageHandle, error) {
	if usageBilling == nil {
		return nil, nil
	}
	billingTenantID := chatManage.TenantID
	if sessionTenantID, ok := types.SessionTenantIDFromContext(ctx); ok {
		billingTenantID = sessionTenantID
	}
	return usageBilling.BeginModelUsage(ctx, types.BillingUsageStartRequest{
		TenantID:    billingTenantID,
		ActorUserID: chatManage.UserID,
		RefNo:       usageRefNo(chatManage, eventType, billingTenantID, chatModel.GetModelID()),
		Source:      "web",
		SessionID:   chatManage.SessionID,
		ServiceCode: "chat.completion",
		ModelID:     chatModel.GetModelID(),
		ModelKey:    chatModel.GetModelName(),
		EstimatedUsage: types.BillingModelUsage{
			InputTokens:  estimateMessageTokens(messages),
			OutputTokens: estimatedOutputTokens(options),
			CallCount:    1,
		},
	})
}

func settleChatModelBilling(
	parent context.Context,
	usageBilling interfaces.UsageBillingService,
	handle *types.BillingUsageHandle,
	usage *types.TokenUsage,
	duration time.Duration,
) {
	if usageBilling == nil || handle == nil || handle.Mode == "off" || usage == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), billingFinalizeTimeout)
	defer cancel()
	_, err := usageBilling.SettleModelUsage(ctx, handle, types.BillingModelUsage{
		InputTokens:    int64(usage.PromptTokens),
		CachedTokens:   int64(usage.CachedTokens),
		OutputTokens:   int64(usage.CompletionTokens),
		CallCount:      1,
		DurationMillis: duration.Milliseconds(),
	})
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": handle.Request.TenantID,
			"ref_no":    handle.Request.RefNo,
		})
	}
}

func releaseChatModelBilling(
	parent context.Context,
	usageBilling interfaces.UsageBillingService,
	handle *types.BillingUsageHandle,
	failureCode string,
) {
	if usageBilling == nil || handle == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), billingFinalizeTimeout)
	defer cancel()
	if err := usageBilling.ReleaseModelUsage(ctx, handle, failureCode); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": handle.Request.TenantID,
			"ref_no":    handle.Request.RefNo,
		})
	}
}
