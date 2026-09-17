package chatpipeline

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type capturingUsageBilling struct {
	interfaces.UsageBillingService
	request types.BillingUsageStartRequest
}

func (s *capturingUsageBilling) BeginModelUsage(
	_ context.Context,
	request types.BillingUsageStartRequest,
) (*types.BillingUsageHandle, error) {
	s.request = request
	return &types.BillingUsageHandle{Request: request, Mode: "observe"}, nil
}

func TestBeginChatModelBillingUsesSessionTenantNotRetrievalTenant(t *testing.T) {
	usageBilling := &capturingUsageBilling{}
	chatManage := &types.ChatManage{
		PipelineRequest: types.PipelineRequest{
			TenantID:  99,
			UserID:    "user-12",
			SessionID: "session-12",
		},
		PipelineContext: types.PipelineContext{MessageID: "message-12"},
	}
	ctx := context.WithValue(context.Background(), types.SessionTenantIDContextKey, uint64(12))
	model := &openStreamChat{}
	_, err := beginChatModelBilling(
		ctx,
		usageBilling,
		types.CHAT_COMPLETION_STREAM,
		chatManage,
		model,
		[]chat.Message{{Role: "user", Content: "hello"}},
		&chat.ChatOptions{MaxCompletionTokens: 100},
	)
	if err != nil {
		t.Fatal(err)
	}
	if usageBilling.request.TenantID != 12 {
		t.Fatalf("tenant_id=%d, want session tenant 12", usageBilling.request.TenantID)
	}
	if usageBilling.request.EstimatedUsage.OutputTokens != 100 {
		t.Fatalf("estimated output=%d, want 100", usageBilling.request.EstimatedUsage.OutputTokens)
	}
}
