package chatpipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// syncEventBus is a thread-safe recorder; the stream plugin emits from a
// background goroutine so the test must guard concurrent appends.
type syncEventBus struct {
	mu     sync.Mutex
	events []types.Event
}

func (b *syncEventBus) On(types.EventType, types.EventHandler) {}

func (b *syncEventBus) Emit(_ context.Context, evt types.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, evt)
	return nil
}

func (b *syncEventBus) finalAnswerContents() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []string
	for _, evt := range b.events {
		if evt.Type != types.EventType(event.EventAgentFinalAnswer) {
			continue
		}
		if data, ok := evt.Data.(event.AgentFinalAnswerData); ok {
			out = append(out, data.Content)
		}
	}
	return out
}

// openStreamChat returns a buffered channel preloaded with chunks and never
// closes it, so the stream plugin blocks on the channel until ctx is cancelled
// — deterministically exercising the ctx.Done() branch.
type openStreamChat struct {
	chunks []types.StreamResponse
}

func (m *openStreamChat) Chat(context.Context, []chat.Message, *chat.ChatOptions) (*types.ChatResponse, error) {
	return nil, nil
}

func (m *openStreamChat) ChatStream(
	context.Context, []chat.Message, *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	ch := make(chan types.StreamResponse, len(m.chunks))
	for _, c := range m.chunks {
		ch <- c
	}
	return ch, nil // intentionally left open
}

func (m *openStreamChat) GetModelName() string { return "mock" }
func (m *openStreamChat) GetModelID() string   { return "mock" }

type controlledStreamChat struct {
	ch <-chan types.StreamResponse
}

func (m *controlledStreamChat) Chat(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (*types.ChatResponse, error) {
	return nil, nil
}

func (m *controlledStreamChat) ChatStream(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return m.ch, nil
}

func (m *controlledStreamChat) GetModelName() string { return "mock" }
func (m *controlledStreamChat) GetModelID() string   { return "mock" }

type streamBillingRecorder struct {
	interfaces.UsageBillingService
	settled chan types.BillingModelUsage
	release chan string
}

func (s *streamBillingRecorder) BeginModelUsage(
	_ context.Context,
	request types.BillingUsageStartRequest,
) (*types.BillingUsageHandle, error) {
	return &types.BillingUsageHandle{Request: request, Mode: "observe"}, nil
}

func (s *streamBillingRecorder) SettleModelUsage(
	_ context.Context,
	_ *types.BillingUsageHandle,
	usage types.BillingModelUsage,
) (*types.TenantUsageLedger, error) {
	s.settled <- usage
	return nil, nil
}

func (s *streamBillingRecorder) ReleaseModelUsage(
	_ context.Context,
	_ *types.BillingUsageHandle,
	failureCode string,
) error {
	s.release <- failureCode
	return nil
}

// stubModelService only needs GetChatModel; the rest is unused for this test.
type stubModelService struct {
	interfaces.ModelService
	model chat.Chat
}

func (s *stubModelService) GetChatModel(context.Context, string) (chat.Chat, error) {
	return s.model, nil
}

// TestStreamFlushesHeldAliasOnCancel verifies that when the request is cancelled
// mid-stream, the decoder's held-back alias suffix is flushed (emitted) rather
// than silently dropped. Without the ctx.Done() flush, "res://0" would be lost.
func TestStreamFlushesHeldAliasOnCancel(t *testing.T) {
	const ref = "resource://AbCdEfGhIjKlMnOpQrStUv"
	bus := &syncEventBus{}
	model := &openStreamChat{chunks: []types.StreamResponse{
		// Ends with a partial alias prefix ("res://0"), so the stream decoder
		// holds it back waiting for the rest that never arrives before cancel.
		{ResponseType: types.ResponseTypeAnswer, Content: "hello res://0"},
	}}

	chatManage := &types.ChatManage{}
	chatManage.SessionID = "sess-cancel"
	chatManage.UserContent = ref // seeds the registry so res://0001 becomes a known alias
	chatManage.EventBus = bus

	ctx, cancel := context.WithCancel(context.Background())
	plugin := &PluginChatCompletionStream{modelService: &stubModelService{model: model}}
	require.Nil(t, plugin.OnEvent(ctx, types.CHAT_COMPLETION_STREAM, chatManage, func() *PluginError { return nil }))

	// Wait until the pre-hold content has been emitted, then cancel.
	require.Eventually(t, func() bool {
		for _, c := range bus.finalAnswerContents() {
			if c == "hello " {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)

	cancel()

	// After cancel, the held "res://0" suffix must be flushed as a final-answer chunk.
	require.Eventually(t, func() bool {
		for _, c := range bus.finalAnswerContents() {
			if c == "res://0" {
				return true
			}
		}
		return false
	}, 2*time.Second, 5*time.Millisecond)
}

func TestStreamBillingWaitsForFinalUsageAfterDone(t *testing.T) {
	responseChan := make(chan types.StreamResponse, 2)
	billing := &streamBillingRecorder{
		settled: make(chan types.BillingModelUsage, 1),
		release: make(chan string, 1),
	}
	bus := &syncEventBus{}
	chatManage := &types.ChatManage{}
	chatManage.SessionID = "sess-billing"
	chatManage.UserContent = "hello"
	chatManage.EventBus = bus

	plugin := &PluginChatCompletionStream{
		modelService: &stubModelService{model: &controlledStreamChat{ch: responseChan}},
		usageBilling: billing,
	}
	require.Nil(t, plugin.OnEvent(
		context.Background(),
		types.CHAT_COMPLETION_STREAM,
		chatManage,
		func() *PluginError { return nil },
	))

	responseChan <- types.StreamResponse{
		ResponseType: types.ResponseTypeAnswer,
		Content:      "answer",
		Done:         true,
	}
	require.Eventually(t, func() bool {
		return len(bus.finalAnswerContents()) == 1
	}, 2*time.Second, 5*time.Millisecond)

	select {
	case usage := <-billing.settled:
		t.Fatalf("billing settled before final usage arrived: %+v", usage)
	default:
	}
	select {
	case failureCode := <-billing.release:
		t.Fatalf("billing released before final usage arrived: %s", failureCode)
	default:
	}

	responseChan <- types.StreamResponse{
		ResponseType: types.ResponseTypeAnswer,
		Done:         true,
		Usage: &types.TokenUsage{
			PromptTokens:     12,
			CompletionTokens: 3,
			TotalTokens:      15,
		},
	}
	close(responseChan)

	select {
	case usage := <-billing.settled:
		require.Equal(t, int64(12), usage.InputTokens)
		require.Equal(t, int64(3), usage.OutputTokens)
	case <-time.After(2 * time.Second):
		t.Fatal("billing was not settled after the final usage arrived")
	}
	select {
	case failureCode := <-billing.release:
		t.Fatalf("billing unexpectedly released: %s", failureCode)
	default:
	}
}
