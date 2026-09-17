package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/billing/pricing"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type usageBillingService struct {
	billing interfaces.BillingRepository
	policy  interfaces.BillingPolicyService
}

func NewUsageBillingService(
	billing interfaces.BillingRepository,
	policy interfaces.BillingPolicyService,
) interfaces.UsageBillingService {
	return &usageBillingService{billing: billing, policy: policy}
}

func normalizeUsageRequest(req types.BillingUsageStartRequest) types.BillingUsageStartRequest {
	if strings.TrimSpace(req.RefNo) == "" {
		req.RefNo = "chat.completion:" + uuid.NewString()
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = "web"
	}
	if strings.TrimSpace(req.ServiceCode) == "" {
		req.ServiceCode = "chat.completion"
	}
	if req.EstimatedUsage.CallCount <= 0 {
		req.EstimatedUsage.CallCount = 1
	}
	return req
}

func ratesForPrice(price *types.BillingModelPrice) pricing.ModelRates {
	if price == nil {
		return pricing.ModelRates{}
	}
	return pricing.ModelRates{
		InputNanoUSDPerMTokens:      price.InputNanoUSDPerMTokens,
		OutputNanoUSDPerMTokens:     price.OutputNanoUSDPerMTokens,
		CacheReadNanoUSDPerMTokens:  price.CacheReadNanoUSDPerMTokens,
		CacheWriteNanoUSDPerMTokens: price.CacheWriteNanoUSDPerMTokens,
		CallNanoUSDPerCall:          price.CallNanoUSDPerCall,
		DurationNanoUSDPerSecond:    price.DurationNanoUSDPerSecond,
	}
}

func pricingUsage(usage types.BillingModelUsage) pricing.ModelUsage {
	return pricing.ModelUsage{
		InputTokens:     usage.InputTokens,
		CachedTokens:    usage.CachedTokens,
		OutputTokens:    usage.OutputTokens,
		ReasoningTokens: usage.ReasoningTokens,
		CallCount:       usage.CallCount,
		DurationMillis:  usage.DurationMillis,
	}
}

func effectiveMultipliers(
	policy types.BillingRuntimePolicy,
	price *types.BillingModelPrice,
	plan *types.BillingPlan,
) (int64, int64) {
	modelMultiplier := policy.DefaultModelMultiplierPPM
	if modelMultiplier <= 0 {
		modelMultiplier = pricing.MultiplierScale
	}
	if price != nil && price.ModelMultiplierPPM > 0 {
		modelMultiplier = price.ModelMultiplierPPM
	}
	planMultiplier := int64(pricing.MultiplierScale)
	if plan != nil && plan.BillingMultiplierPPM > 0 {
		planMultiplier = plan.BillingMultiplierPPM
	}
	return modelMultiplier, planMultiplier
}

func calculateModelCharge(
	policy types.BillingRuntimePolicy,
	price *types.BillingModelPrice,
	plan *types.BillingPlan,
	usage types.BillingModelUsage,
) (pricing.ModelCharge, error) {
	modelMultiplier, planMultiplier := effectiveMultipliers(policy, price, plan)
	return pricing.CalculateModelCharge(
		ratesForPrice(price),
		pricingUsage(usage),
		policy.PointMicrosPerUSD,
		modelMultiplier,
		planMultiplier,
	)
}

func (s *usageBillingService) BeginModelUsage(
	ctx context.Context,
	req types.BillingUsageStartRequest,
) (*types.BillingUsageHandle, error) {
	req = normalizeUsageRequest(req)
	policy := s.policy.RuntimePolicy(ctx)
	handle := &types.BillingUsageHandle{
		Request: req,
		Mode:    "off",
	}
	if !policy.Enabled || policy.EnforcementMode == "off" {
		return handle, nil
	}
	handle.Mode = policy.EnforcementMode

	subscription, plan, err := s.billing.GetCurrentSubscription(ctx, req.TenantID)
	if err != nil {
		return nil, err
	}
	handle.Subscription = subscription
	handle.Plan = plan
	if plan == nil || subscription == nil {
		if handle.Mode == "enforce" {
			return nil, errors.New("billing: current subscription is required")
		}
		handle.FailureCode = "missing_subscription"
		handle.UsageScope = "workspace_usage"
		return handle, nil
	}
	if plan.SpaceType == types.SpaceTypeOrganization {
		handle.UsageScope = "enterprise_allocated_usage"
		if handle.Mode == "enforce" {
			handle.Mode = "observe"
			handle.FailureCode = "enterprise_allocation_pending"
		}
	} else {
		handle.UsageScope = "personal_usage"
	}

	price, err := s.billing.GetActiveModelPrice(ctx, req.ModelKey, req.ModelID)
	if err != nil {
		return nil, err
	}
	handle.Price = price
	if price == nil {
		handle.FailureCode = "model_price_missing"
		if handle.Mode == "enforce" {
			return nil, fmt.Errorf("billing: no active model price for %q", req.ModelKey)
		}
		return handle, nil
	}

	estimate, err := calculateModelCharge(policy, price, plan, req.EstimatedUsage)
	if err != nil {
		return nil, err
	}
	if handle.Mode == "enforce" {
		reservation, err := s.billing.CreateUsageReservation(
			ctx,
			handle,
			estimate.BaseCostNanoUSD,
			estimate.BilledPointMicros,
		)
		if err != nil {
			if errors.Is(err, repository.ErrInsufficientBillingCredits) {
				return nil, errors.New("billing: insufficient credits for this model call")
			}
			return nil, err
		}
		handle.Reservation = reservation
	}
	return handle, nil
}

func (s *usageBillingService) SettleModelUsage(
	ctx context.Context,
	handle *types.BillingUsageHandle,
	usage types.BillingModelUsage,
) (*types.TenantUsageLedger, error) {
	if handle == nil || handle.Mode == "off" {
		return nil, nil
	}
	if usage.CallCount <= 0 {
		usage.CallCount = 1
	}
	policy := s.policy.RuntimePolicy(ctx)
	charge := pricing.ModelCharge{}
	var err error
	status := "observed"
	failureCode := handle.FailureCode
	if handle.Price == nil {
		status = "unpriced"
		if failureCode == "" {
			failureCode = "model_price_missing"
		}
	} else {
		charge, err = calculateModelCharge(policy, handle.Price, handle.Plan, usage)
		if err != nil {
			return nil, err
		}
		if handle.Mode == "enforce" {
			status = "settled"
		}
	}
	modelMultiplier, planMultiplier := effectiveMultipliers(policy, handle.Price, handle.Plan)
	snapshot, err := json.Marshal(map[string]any{
		"mode":                    handle.Mode,
		"failure_code":            failureCode,
		"price":                   handle.Price,
		"subscription_id":         idOfSubscription(handle.Subscription),
		"plan_code":               codeOfPlan(handle.Plan),
		"point_micros_per_usd":    policy.PointMicrosPerUSD,
		"model_multiplier_ppm":    modelMultiplier,
		"plan_multiplier_ppm":     planMultiplier,
		"estimated_usage":         handle.Request.EstimatedUsage,
		"actual_usage":            usage,
		"uncached_input_tokens":   charge.UncachedInputTokens,
		"input_cost_nanousd":      charge.InputCostNanoUSD,
		"cache_read_cost_nanousd": charge.CacheReadCostNanoUSD,
		"output_cost_nanousd":     charge.OutputCostNanoUSD,
		"call_cost_nanousd":       charge.CallCostNanoUSD,
		"duration_cost_nanousd":   charge.DurationCostNanoUSD,
		"base_cost_nanousd":       charge.BaseCostNanoUSD,
		"rated_cost_nanousd":      charge.RatedCostNanoUSD,
		"billed_point_micros":     charge.BilledPointMicros,
	})
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	ledger := &types.TenantUsageLedger{
		ID:                  uuid.NewString(),
		TenantID:            handle.Request.TenantID,
		ActorUserID:         handle.Request.ActorUserID,
		UsageScope:          handle.UsageScope,
		RefNo:               handle.Request.RefNo,
		Source:              handle.Request.Source,
		SessionID:           handle.Request.SessionID,
		ServiceCode:         handle.Request.ServiceCode,
		ModelID:             handle.Request.ModelID,
		ModelKey:            handle.Request.ModelKey,
		InputTokens:         usage.InputTokens,
		CachedTokens:        usage.CachedTokens,
		OutputTokens:        usage.OutputTokens,
		ReasoningTokens:     usage.ReasoningTokens,
		CallCount:           usage.CallCount,
		DurationMillis:      usage.DurationMillis,
		BaseCostNanoUSD:     charge.BaseCostNanoUSD,
		RatedCostNanoUSD:    charge.RatedCostNanoUSD,
		BilledPointMicros:   charge.BilledPointMicros,
		Status:              status,
		FailureCode:         failureCode,
		BillingAt:           now,
		UsageDate:           now,
		PricingSnapshotJSON: types.JSON(snapshot),
	}
	if handle.Price != nil {
		ledger.Provider = handle.Price.Provider
		ledger.PricingID = handle.Price.ID
		ledger.PricingVersion = handle.Price.Version
	}
	row, err := s.billing.SettleUsage(ctx, handle, ledger)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id": handle.Request.TenantID,
			"ref_no":    handle.Request.RefNo,
		})
	}
	return row, err
}

func idOfSubscription(subscription *types.TenantSubscription) string {
	if subscription == nil {
		return ""
	}
	return subscription.ID
}

func codeOfPlan(plan *types.BillingPlan) string {
	if plan == nil {
		return ""
	}
	return plan.Code
}

func (s *usageBillingService) ReleaseModelUsage(
	ctx context.Context,
	handle *types.BillingUsageHandle,
	failureCode string,
) error {
	if handle == nil || handle.Mode != "enforce" || handle.Reservation == nil {
		return nil
	}
	return s.billing.ReleaseUsageReservation(
		ctx,
		handle.Request.TenantID,
		handle.Request.RefNo,
		failureCode,
	)
}

func (s *usageBillingService) ListModelPrices(ctx context.Context) ([]*types.BillingModelPrice, error) {
	return s.billing.ListModelPrices(ctx)
}

func (s *usageBillingService) CreateModelPriceVersion(
	ctx context.Context,
	input types.BillingModelPriceInput,
) (*types.BillingModelPrice, error) {
	return s.billing.CreateModelPriceVersion(ctx, input)
}

func (s *usageBillingService) ListUsageLedgers(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return s.billing.ListUsageLedgers(ctx, tenantID, limit)
}
