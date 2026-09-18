package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type usageBillingRecorder struct {
	interfaces.UsageBillingService
	starts   []types.BillingUsageStartRequest
	settles  []types.BillingModelUsage
	released []string
}

func (r *usageBillingRecorder) BeginModelUsage(
	_ context.Context,
	req types.BillingUsageStartRequest,
) (*types.BillingUsageHandle, error) {
	r.starts = append(r.starts, req)
	return &types.BillingUsageHandle{Request: req, Mode: "observe"}, nil
}

func (r *usageBillingRecorder) SettleModelUsage(
	_ context.Context,
	_ *types.BillingUsageHandle,
	usage types.BillingModelUsage,
) (*types.TenantUsageLedger, error) {
	r.settles = append(r.settles, usage)
	return nil, nil
}

func (r *usageBillingRecorder) ReleaseModelUsage(
	_ context.Context,
	handle *types.BillingUsageHandle,
	_ string,
) error {
	r.released = append(r.released, handle.Request.RefNo)
	return nil
}

type billingTestChat struct{}

func (billingTestChat) Chat(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (*types.ChatResponse, error) {
	return &types.ChatResponse{
		Content: "ok",
		Usage: types.TokenUsage{
			PromptTokens:     12,
			CachedTokens:     2,
			CompletionTokens: 5,
		},
	}, nil
}

func (billingTestChat) ChatStream(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	out := make(chan types.StreamResponse)
	close(out)
	return out, nil
}

func (billingTestChat) GetModelName() string { return "test-chat" }
func (billingTestChat) GetModelID() string   { return "model-chat" }

type billingTestEmbedder struct{}

func (billingTestEmbedder) Embed(context.Context, string) ([]float32, error) {
	return []float32{1, 2}, nil
}

func (billingTestEmbedder) BatchEmbed(
	_ context.Context,
	texts []string,
) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for index := range texts {
		out[index] = []float32{float32(index)}
	}
	return out, nil
}

func (billingTestEmbedder) BatchEmbedWithPool(
	ctx context.Context,
	_ embedding.Embedder,
	texts []string,
) ([][]float32, error) {
	return billingTestEmbedder{}.BatchEmbed(ctx, texts)
}

func (billingTestEmbedder) GetModelName() string { return "test-embedding" }
func (billingTestEmbedder) GetDimensions() int   { return 2 }
func (billingTestEmbedder) GetModelID() string   { return "model-embedding" }

type billingTestTool struct{}

func (billingTestTool) Name() string                { return "test_tool" }
func (billingTestTool) Description() string         { return "test" }
func (billingTestTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }
func (billingTestTool) Execute(context.Context, json.RawMessage) (*types.ToolResult, error) {
	return &types.ToolResult{
		Success: true,
		Data:    map[string]any{"count": 3},
	}, nil
}

func billingTestContext() context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	return context.WithValue(ctx, types.UserIDContextKey, "user-7")
}

func TestBillingChatUsesExplicitServiceCodeAndBypass(t *testing.T) {
	recorder := &usageBillingRecorder{}
	model := wrapBillingChat(billingTestChat{}, recorder)
	ctx := types.WithBillingServiceCode(billingTestContext(), "knowledge.summary")

	if _, err := model.Chat(ctx, []chat.Message{{Role: "user", Content: "hello"}}, nil); err != nil {
		t.Fatal(err)
	}
	if len(recorder.starts) != 1 {
		t.Fatalf("starts=%d, want 1", len(recorder.starts))
	}
	if got := recorder.starts[0].ServiceCode; got != "knowledge.summary" {
		t.Fatalf("service=%q, want knowledge.summary", got)
	}
	if len(recorder.settles) != 1 || recorder.settles[0].InputTokens != 12 || recorder.settles[0].OutputTokens != 5 {
		t.Fatalf("settles=%+v, want provider usage", recorder.settles)
	}

	if _, err := model.Chat(types.WithUsageBillingBypass(ctx), []chat.Message{{Role: "user", Content: "skip"}}, nil); err != nil {
		t.Fatal(err)
	}
	if len(recorder.starts) != 1 {
		t.Fatalf("bypassed call recorded %d starts, want 1", len(recorder.starts))
	}
}

func TestBillingEmbedderKeepsOwnServiceCode(t *testing.T) {
	recorder := &usageBillingRecorder{}
	model := wrapBillingEmbedder(billingTestEmbedder{}, recorder)
	ctx := types.WithBillingServiceCode(billingTestContext(), "agent.run")

	if _, err := model.Embed(ctx, "需要向量化的内容"); err != nil {
		t.Fatal(err)
	}
	if len(recorder.starts) != 1 {
		t.Fatalf("starts=%d, want 1", len(recorder.starts))
	}
	request := recorder.starts[0]
	if request.ServiceCode != "knowledge.embedding" {
		t.Fatalf("service=%q, want knowledge.embedding", request.ServiceCode)
	}
	if request.ModelID != "model-embedding" || request.EstimatedUsage.ServiceUnits <= 0 {
		t.Fatalf("request=%+v, want embedding model and positive token units", request)
	}
}

func TestBillingToolUsesServicePricingWithoutModel(t *testing.T) {
	recorder := &usageBillingRecorder{}
	tool := wrapBillingTool(billingTestTool{}, recorder, "web_search.query")

	if _, err := tool.Execute(billingTestContext(), json.RawMessage(`{"query":"billing"}`)); err != nil {
		t.Fatal(err)
	}
	if len(recorder.starts) != 1 {
		t.Fatalf("starts=%d, want 1", len(recorder.starts))
	}
	request := recorder.starts[0]
	if request.ServiceCode != "web_search.query" || request.ModelID != "" || request.ModelKey != "" {
		t.Fatalf("request=%+v, want service-only usage", request)
	}
	if len(recorder.settles) != 1 || recorder.settles[0].ServiceUnits != 3 {
		t.Fatalf("settles=%+v, want 3 result units", recorder.settles)
	}
}
