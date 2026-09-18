package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type billingTool struct {
	inner       types.Tool
	billing     interfaces.UsageBillingService
	serviceCode string
}

func wrapBillingTool(
	inner types.Tool,
	billing interfaces.UsageBillingService,
	serviceCode string,
) types.Tool {
	if inner == nil || billing == nil {
		return inner
	}
	return &billingTool{
		inner:       inner,
		billing:     billing,
		serviceCode: serviceCode,
	}
}

func (b *billingTool) Name() string                { return b.inner.Name() }
func (b *billingTool) Description() string         { return b.inner.Description() }
func (b *billingTool) Parameters() json.RawMessage { return b.inner.Parameters() }

func (b *billingTool) Execute(
	ctx context.Context,
	args json.RawMessage,
) (*types.ToolResult, error) {
	usage := types.BillingModelUsage{CallCount: 1, ServiceUnits: 1}
	billingCtx := types.WithBillingServiceCode(ctx, b.serviceCode)
	handle, err := beginDecoratedModelUsage(
		billingCtx,
		b.billing,
		b.serviceCode,
		"",
		"",
		usage,
	)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now()
	result, err := b.inner.Execute(ctx, args)
	if err != nil || result == nil || !result.Success {
		releaseDecoratedModelUsage(ctx, b.billing, handle)
		return result, err
	}
	usage.DurationMillis = time.Since(startedAt).Milliseconds()
	if b.serviceCode == "web_search.query" {
		usage.ServiceUnits = toolResultCount(result)
	}
	settleDecoratedModelUsage(ctx, b.billing, handle, usage)
	return result, nil
}

func toolResultCount(result *types.ToolResult) int64 {
	if result == nil || result.Data == nil {
		return 1
	}
	switch value := result.Data["count"].(type) {
	case int:
		return max(int64(value), 1)
	case int32:
		return max(int64(value), 1)
	case int64:
		return max(value, 1)
	case float32:
		return max(int64(value), 1)
	case float64:
		return max(int64(value), 1)
	default:
		return 1
	}
}
