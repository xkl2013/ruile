package service

import (
	"context"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/asr"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/models/embedding"
	"github.com/Tencent/WeKnora/internal/models/rerank"
	"github.com/Tencent/WeKnora/internal/models/vlm"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const modelBillingFinalizeTimeout = 10 * time.Second

func estimatedTextTokens(values ...string) int64 {
	var runes int64
	for _, value := range values {
		runes += int64(utf8.RuneCountInString(value))
	}
	if runes <= 0 {
		return 0
	}
	return (runes + 3) / 4
}

func beginDecoratedModelUsage(
	ctx context.Context,
	billing interfaces.UsageBillingService,
	serviceCode, modelID, modelKey string,
	usage types.BillingModelUsage,
) (*types.BillingUsageHandle, error) {
	if billing == nil || types.UsageBillingBypassed(ctx) {
		return nil, nil
	}
	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, nil
	}
	actorUserID, _ := types.UserIDFromContext(ctx)
	if actorUserID == "" {
		actorUserID = fmt.Sprintf("system-%d", tenantID)
	}
	source := "web"
	if types.IsBackgroundTask(ctx) {
		source = "worker"
	}
	source = types.BillingSourceFromContext(ctx, source)
	return billing.BeginModelUsage(ctx, types.BillingUsageStartRequest{
		TenantID:       tenantID,
		ActorUserID:    actorUserID,
		RefNo:          fmt.Sprintf("%s:%d:%s", serviceCode, tenantID, uuid.NewString()),
		Source:         source,
		ServiceCode:    serviceCode,
		ModelID:        modelID,
		ModelKey:       modelKey,
		EstimatedUsage: usage,
	})
}

func settleDecoratedModelUsage(
	parent context.Context,
	billing interfaces.UsageBillingService,
	handle *types.BillingUsageHandle,
	usage types.BillingModelUsage,
) {
	if billing == nil || handle == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), modelBillingFinalizeTimeout)
	defer cancel()
	if _, err := billing.SettleModelUsage(ctx, handle, usage); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": handle.Request.TenantID,
			"ref_no":    handle.Request.RefNo,
		})
	}
}

func releaseDecoratedModelUsage(
	parent context.Context,
	billing interfaces.UsageBillingService,
	handle *types.BillingUsageHandle,
) {
	if billing == nil || handle == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), modelBillingFinalizeTimeout)
	defer cancel()
	if err := billing.ReleaseModelUsage(ctx, handle, "model_call_failed"); err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": handle.Request.TenantID,
			"ref_no":    handle.Request.RefNo,
		})
	}
}

type billingEmbedder struct {
	inner   embedding.Embedder
	billing interfaces.UsageBillingService
}

func wrapBillingEmbedder(
	inner embedding.Embedder,
	billing interfaces.UsageBillingService,
) embedding.Embedder {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingEmbedder{inner: inner, billing: billing}
}

func (b *billingEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	values, err := b.batch(ctx, []string{text}, func() ([][]float32, error) {
		value, err := b.inner.Embed(ctx, text)
		return [][]float32{value}, err
	})
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, nil
	}
	return values[0], nil
}

func (b *billingEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	return b.batch(ctx, texts, func() ([][]float32, error) {
		return b.inner.BatchEmbed(ctx, texts)
	})
}

func (b *billingEmbedder) BatchEmbedWithPool(
	ctx context.Context,
	_ embedding.Embedder,
	texts []string,
) ([][]float32, error) {
	return b.batch(ctx, texts, func() ([][]float32, error) {
		return b.inner.BatchEmbedWithPool(ctx, b.inner, texts)
	})
}

func (b *billingEmbedder) batch(
	ctx context.Context,
	texts []string,
	call func() ([][]float32, error),
) ([][]float32, error) {
	tokenCount := estimatedTextTokens(texts...)
	usage := types.BillingModelUsage{
		InputTokens:  tokenCount,
		CallCount:    1,
		ServiceUnits: tokenCount,
	}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		"knowledge.embedding",
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return nil, err
	}
	values, err := call()
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, err
	}
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return values, nil
}

func (b *billingEmbedder) GetModelName() string { return b.inner.GetModelName() }
func (b *billingEmbedder) GetDimensions() int   { return b.inner.GetDimensions() }
func (b *billingEmbedder) GetModelID() string   { return b.inner.GetModelID() }

type billingReranker struct {
	inner   rerank.Reranker
	billing interfaces.UsageBillingService
}

func wrapBillingReranker(
	inner rerank.Reranker,
	billing interfaces.UsageBillingService,
) rerank.Reranker {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingReranker{inner: inner, billing: billing}
}

func (b *billingReranker) Rerank(
	ctx context.Context,
	query string,
	documents []string,
) ([]rerank.RankResult, error) {
	allText := append([]string{query}, documents...)
	tokenCount := estimatedTextTokens(allText...)
	usage := types.BillingModelUsage{
		InputTokens:  tokenCount,
		CallCount:    1,
		ServiceUnits: int64(len(documents)),
	}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		"retrieval.rerank",
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return nil, err
	}
	results, err := b.inner.Rerank(ctx, query, documents)
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, err
	}
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return results, nil
}

func (b *billingReranker) GetModelName() string { return b.inner.GetModelName() }
func (b *billingReranker) GetModelID() string   { return b.inner.GetModelID() }

type billingVLM struct {
	inner              vlm.VLM
	billing            interfaces.UsageBillingService
	defaultServiceCode string
}

func wrapBillingVLM(
	inner vlm.VLM,
	billing interfaces.UsageBillingService,
	defaultServiceCode string,
) vlm.VLM {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingVLM{
		inner:              inner,
		billing:            billing,
		defaultServiceCode: defaultServiceCode,
	}
}

func (b *billingVLM) Predict(
	ctx context.Context,
	images [][]byte,
	prompt string,
) (string, error) {
	inputTokens := estimatedTextTokens(prompt)
	usage := types.BillingModelUsage{
		InputTokens:  inputTokens,
		OutputTokens: 1024,
		CallCount:    1,
		ServiceUnits: int64(len(images)),
	}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		b.defaultServiceCode,
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return "", err
	}
	value, err := b.inner.Predict(ctx, images, prompt)
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return "", err
	}
	usage.OutputTokens = estimatedTextTokens(value)
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return value, nil
}

func (b *billingVLM) GetModelName() string { return b.inner.GetModelName() }
func (b *billingVLM) GetModelID() string   { return b.inner.GetModelID() }

type billingASR struct {
	inner   asr.ASR
	billing interfaces.UsageBillingService
}

func wrapBillingASR(inner asr.ASR, billing interfaces.UsageBillingService) asr.ASR {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingASR{inner: inner, billing: billing}
}

func (b *billingASR) Transcribe(
	ctx context.Context,
	audioBytes []byte,
	fileName string,
) (*asr.TranscriptionResult, error) {
	usage := types.BillingModelUsage{CallCount: 1}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		"file.asr",
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return nil, err
	}
	result, err := b.inner.Transcribe(ctx, audioBytes, fileName)
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, err
	}
	if result != nil {
		usage.OutputTokens = estimatedTextTokens(result.Text)
		if len(result.Segments) > 0 {
			last := result.Segments[len(result.Segments)-1]
			if last.End > 0 {
				usage.ServiceUnits = int64(last.End + 0.999)
			}
		}
	}
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return result, nil
}

func (b *billingASR) GetModelName() string { return b.inner.GetModelName() }
func (b *billingASR) GetModelID() string   { return b.inner.GetModelID() }

type billingChat struct {
	inner   chat.Chat
	billing interfaces.UsageBillingService
}

func wrapBillingChat(inner chat.Chat, billing interfaces.UsageBillingService) chat.Chat {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingChat{inner: inner, billing: billing}
}

func estimateChatMessages(messages []chat.Message) int64 {
	values := make([]string, 0, len(messages)*2)
	for _, message := range messages {
		values = append(values, message.Content)
		for _, part := range message.MultiContent {
			values = append(values, part.Text)
		}
	}
	return estimatedTextTokens(values...)
}

func estimatedChatOutput(opts *chat.ChatOptions) int64 {
	if opts != nil {
		if opts.MaxCompletionTokens > 0 {
			return int64(opts.MaxCompletionTokens)
		}
		if opts.MaxTokens > 0 {
			return int64(opts.MaxTokens)
		}
	}
	return 4096
}

func (b *billingChat) Chat(
	ctx context.Context,
	messages []chat.Message,
	opts *chat.ChatOptions,
) (*types.ChatResponse, error) {
	serviceCode := types.BillingServiceCodeFromContext(ctx, "chat.completion")
	usage := types.BillingModelUsage{
		InputTokens:  estimateChatMessages(messages),
		OutputTokens: estimatedChatOutput(opts),
		CallCount:    1,
	}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		serviceCode,
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now()
	response, err := b.inner.Chat(ctx, messages, opts)
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, err
	}
	if response != nil {
		usage.InputTokens = int64(response.Usage.PromptTokens)
		usage.CachedTokens = int64(response.Usage.CachedTokens)
		usage.OutputTokens = int64(response.Usage.CompletionTokens)
	}
	usage.DurationMillis = time.Since(startedAt).Milliseconds()
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return response, nil
}

func (b *billingChat) ChatStream(
	ctx context.Context,
	messages []chat.Message,
	opts *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	if types.UsageBillingBypassed(ctx) {
		return b.inner.ChatStream(ctx, messages, opts)
	}
	serviceCode := types.BillingServiceCodeFromContext(ctx, "chat.completion")
	usage := types.BillingModelUsage{
		InputTokens:  estimateChatMessages(messages),
		OutputTokens: estimatedChatOutput(opts),
		CallCount:    1,
	}
	handle, err := beginDecoratedModelUsage(
		ctx,
		b.billing,
		serviceCode,
		b.inner.GetModelID(),
		b.inner.GetModelName(),
		usage,
	)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now()
	upstream, err := b.inner.ChatStream(ctx, messages, opts)
	if err != nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, err
	}
	if upstream == nil {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return nil, nil
	}
	output := make(chan types.StreamResponse)
	go func() {
		defer close(output)
		var finalUsage *types.TokenUsage
		for item := range upstream {
			if item.Usage != nil {
				copy := *item.Usage
				finalUsage = &copy
			}
			select {
			case output <- item:
			case <-ctx.Done():
				releaseDecoratedModelUsage(ctx, b.billing, handle)
				return
			}
		}
		if finalUsage != nil {
			usage.InputTokens = int64(finalUsage.PromptTokens)
			usage.CachedTokens = int64(finalUsage.CachedTokens)
			usage.OutputTokens = int64(finalUsage.CompletionTokens)
		}
		usage.DurationMillis = time.Since(startedAt).Milliseconds()
		settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	}()
	return output, nil
}

func (b *billingChat) GetModelName() string { return b.inner.GetModelName() }
func (b *billingChat) GetModelID() string   { return b.inner.GetModelID() }
