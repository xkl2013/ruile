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

func ratesForPrice(
	price *types.BillingModelPrice,
	usage types.BillingModelUsage,
) (pricing.ModelRates, error) {
	if price == nil {
		return pricing.ModelRates{}, nil
	}
	if price.PricingMode == "tiered" {
		return pricing.ResolveTieredModelRates(price.TieredPricingJSON, usage.InputTokens)
	}
	return pricing.ModelRates{
		InputNanoUSDPerMTokens:      price.InputNanoUSDPerMTokens,
		OutputNanoUSDPerMTokens:     price.OutputNanoUSDPerMTokens,
		CacheReadNanoUSDPerMTokens:  price.CacheReadNanoUSDPerMTokens,
		CacheWriteNanoUSDPerMTokens: price.CacheWriteNanoUSDPerMTokens,
		CallNanoUSDPerCall:          price.CallNanoUSDPerCall,
		DurationNanoUSDPerSecond:    price.DurationNanoUSDPerSecond,
	}, nil
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

func usesServicePricing(serviceCode string) bool {
	return strings.TrimSpace(serviceCode) == types.BillingServiceCodeMCPToolCall
}

func effectiveMultipliers(
	policy types.BillingRuntimePolicy,
	price *types.BillingModelPrice,
	servicePrice *types.BillingServicePrice,
	plan *types.BillingPlan,
) (int64, int64, int64) {
	modelMultiplier := policy.DefaultModelMultiplierPPM
	if modelMultiplier <= 0 {
		modelMultiplier = pricing.MultiplierScale
	}
	if price != nil && price.ModelMultiplierPPM > 0 {
		modelMultiplier = price.ModelMultiplierPPM
	}
	serviceMultiplier := policy.DefaultServiceMultiplierPPM
	if serviceMultiplier <= 0 {
		serviceMultiplier = pricing.MultiplierScale
	}
	if servicePrice != nil && servicePrice.ServiceMultiplierPPM > 0 {
		serviceMultiplier = servicePrice.ServiceMultiplierPPM
	}
	planMultiplier := int64(pricing.MultiplierScale)
	if plan != nil && plan.BillingMultiplierPPM > 0 {
		planMultiplier = plan.BillingMultiplierPPM
	}
	return modelMultiplier, serviceMultiplier, planMultiplier
}

func calculateUsageCharge(
	policy types.BillingRuntimePolicy,
	price *types.BillingModelPrice,
	servicePrice *types.BillingServicePrice,
	plan *types.BillingPlan,
	usage types.BillingModelUsage,
) (pricing.CombinedCharge, error) {
	modelRates, err := ratesForPrice(price, usage)
	if err != nil {
		return pricing.CombinedCharge{}, err
	}
	modelMultiplier, serviceMultiplier, planMultiplier := effectiveMultipliers(
		policy,
		price,
		servicePrice,
		plan,
	)
	serviceMode := "call"
	serviceRates := pricing.ServiceRates{}
	if servicePrice != nil {
		serviceMode = servicePrice.PricingMode
		serviceRates = pricing.ServiceRates{
			NanoUSDPerCall: servicePrice.NanoUSDPerCall,
			NanoUSDPerUnit: servicePrice.NanoUSDPerUnit,
		}
	}
	return pricing.CalculateCombinedCharge(
		modelRates,
		pricingUsage(usage),
		modelMultiplier,
		serviceMode,
		serviceRates,
		pricing.ServiceUsage{
			CallCount: usage.CallCount,
			Units:     usage.ServiceUnits,
		},
		serviceMultiplier,
		planMultiplier,
		policy.PointMicrosPerUSD,
	)
}

func requiresModelPrice(req types.BillingUsageStartRequest) bool {
	return strings.TrimSpace(req.ModelKey) != "" || strings.TrimSpace(req.ModelID) != ""
}

func usageBillingPeriod(subscription *types.TenantSubscription, now time.Time) (time.Time, time.Time) {
	if subscription != nil && subscription.CurrentPeriodStart != nil && subscription.CurrentPeriodEnd != nil &&
		subscription.CurrentPeriodEnd.After(*subscription.CurrentPeriodStart) {
		return subscription.CurrentPeriodStart.UTC(), subscription.CurrentPeriodEnd.UTC()
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func effectiveMemberUsagePolicy(
	enterprisePolicy *types.TenantBillingPolicy,
	allocation *types.TenantMemberCreditAllocation,
) (string, int64, string) {
	limitMode := types.MemberLimitModeInherit
	monthlyLimit := int64(0)
	overagePolicy := types.MemberOveragePolicyBlock
	if enterprisePolicy != nil {
		monthlyLimit = max(enterprisePolicy.DefaultMemberMonthlyLimitPointMicros, 0)
		if enterprisePolicy.MemberOveragePolicy == types.MemberOveragePolicyUseEnterpriseBalance {
			overagePolicy = types.MemberOveragePolicyUseEnterpriseBalance
		}
	}
	if allocation == nil {
		return limitMode, monthlyLimit, overagePolicy
	}
	switch allocation.LimitMode {
	case types.MemberLimitModeCustom:
		limitMode = types.MemberLimitModeCustom
		monthlyLimit = max(allocation.MonthlyLimitPointMicros, 0)
	case types.MemberLimitModeUnlimited:
		limitMode = types.MemberLimitModeUnlimited
		monthlyLimit = 0
	}
	switch allocation.OveragePolicy {
	case types.MemberOveragePolicyBlock, types.MemberOveragePolicyUseEnterpriseBalance:
		overagePolicy = allocation.OveragePolicy
	}
	return limitMode, monthlyLimit, overagePolicy
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
		handle.UsageScope = types.BillingUsageScopeEnterprise
		if strings.TrimSpace(req.ActorUserID) == "" {
			handle.FailureCode = "enterprise_actor_missing"
		} else {
			enterprisePolicy, err := s.billing.EnsureTenantBillingPolicy(
				ctx,
				req.TenantID,
				policy.DefaultMemberMonthlyLimitPointMicros,
				req.ActorUserID,
			)
			if err != nil {
				if handle.Mode == "enforce" {
					return nil, err
				}
				handle.FailureCode = "enterprise_policy_unavailable"
			} else {
				handle.EnterprisePolicy = enterprisePolicy
			}
			periodStart, periodEnd := usageBillingPeriod(subscription, time.Now().UTC())
			allocation, err := s.billing.EnsureCurrentMemberAllocation(
				ctx,
				req.TenantID,
				req.ActorUserID,
				periodStart,
				periodEnd,
				policy.DefaultMemberMonthlyLimitPointMicros,
				req.ActorUserID,
			)
			if err != nil {
				if handle.Mode == "enforce" {
					return nil, err
				}
				handle.FailureCode = "enterprise_allocation_unavailable"
			} else {
				handle.Allocation = allocation
				handle.AllocationID = allocation.ID
				handle.MemberLimitMode,
					handle.MemberMonthlyLimitPointMicros,
					handle.MemberOveragePolicy =
					effectiveMemberUsagePolicy(handle.EnterprisePolicy, allocation)
				if allocation.Status != types.BillingStatusActive {
					handle.FailureCode = "enterprise_member_policy_inactive"
				}
			}
		}
		if handle.Mode == "enforce" && handle.FailureCode != "" {
			return nil, fmt.Errorf("billing: enterprise usage policy unavailable: %s", handle.FailureCode)
		}
	} else {
		handle.UsageScope = types.BillingUsageScopePersonal
	}

	var price *types.BillingModelPrice
	if !usesServicePricing(req.ServiceCode) {
		var err error
		price, err = s.billing.GetActiveModelPrice(ctx, req.ModelKey, req.ModelID)
		if err != nil {
			return nil, err
		}
		handle.Price = price
		if price == nil && requiresModelPrice(req) {
			handle.FailureCode = "model_price_missing"
			if handle.Mode == "enforce" {
				return nil, fmt.Errorf("billing: no active model price for %q", req.ModelKey)
			}
		}
	}
	var servicePrice *types.BillingServicePrice
	if usesServicePricing(req.ServiceCode) {
		var err error
		servicePrice, err = s.billing.GetActiveServicePrice(ctx, req.ServiceCode)
		if err != nil {
			return nil, err
		}
		handle.ServicePrice = servicePrice
		if servicePrice == nil {
			if handle.FailureCode == "" {
				handle.FailureCode = "service_price_missing"
			}
			if handle.Mode == "enforce" {
				return nil, fmt.Errorf("billing: no active service price for %q", req.ServiceCode)
			}
		}
	}

	estimate, err := calculateUsageCharge(policy, price, servicePrice, plan, req.EstimatedUsage)
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
	charge := pricing.CombinedCharge{}
	var err error
	status := "observed"
	failureCode := handle.FailureCode
	if (handle.Price == nil && requiresModelPrice(handle.Request)) ||
		(usesServicePricing(handle.Request.ServiceCode) && handle.ServicePrice == nil) {
		status = "unpriced"
		if failureCode == "" {
			if usesServicePricing(handle.Request.ServiceCode) {
				failureCode = "service_price_missing"
			} else {
				failureCode = "model_price_missing"
			}
		}
	} else {
		charge, err = calculateUsageCharge(
			policy,
			handle.Price,
			handle.ServicePrice,
			handle.Plan,
			usage,
		)
		if err != nil {
			return nil, err
		}
		if handle.Mode == "enforce" {
			status = "settled"
		}
	}
	modelMultiplier, serviceMultiplier, planMultiplier := effectiveMultipliers(
		policy,
		handle.Price,
		handle.ServicePrice,
		handle.Plan,
	)
	snapshot, err := json.Marshal(map[string]any{
		"mode":                       handle.Mode,
		"failure_code":               failureCode,
		"price":                      handle.Price,
		"service_price":              handle.ServicePrice,
		"subscription_id":            idOfSubscription(handle.Subscription),
		"plan_code":                  codeOfPlan(handle.Plan),
		"point_micros_per_usd":       policy.PointMicrosPerUSD,
		"model_multiplier_ppm":       modelMultiplier,
		"service_multiplier_ppm":     serviceMultiplier,
		"plan_multiplier_ppm":        planMultiplier,
		"estimated_usage":            handle.Request.EstimatedUsage,
		"actual_usage":               usage,
		"uncached_input_tokens":      charge.Model.UncachedInputTokens,
		"input_cost_nanousd":         charge.Model.InputCostNanoUSD,
		"cache_read_cost_nanousd":    charge.Model.CacheReadCostNanoUSD,
		"output_cost_nanousd":        charge.Model.OutputCostNanoUSD,
		"model_call_cost_nanousd":    charge.Model.CallCostNanoUSD,
		"duration_cost_nanousd":      charge.Model.DurationCostNanoUSD,
		"service_call_cost_nanousd":  charge.Service.CallCostNanoUSD,
		"service_unit_cost_nanousd":  charge.Service.UnitCostNanoUSD,
		"model_rated_cost_nanousd":   charge.Model.RatedCostNanoUSD,
		"service_rated_cost_nanousd": charge.Service.RatedCostNanoUSD,
		"base_cost_nanousd":          charge.BaseCostNanoUSD,
		"rated_cost_nanousd":         charge.RatedCostNanoUSD,
		"billed_point_micros":        charge.BilledPointMicros,
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
		AllocationID:        handle.AllocationID,
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
		ServiceUnits:        usage.ServiceUnits,
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
	if handle.ServicePrice != nil {
		ledger.ServicePricingID = handle.ServicePrice.ID
		ledger.ServicePricingVersion = handle.ServicePrice.Version
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

func (s *usageBillingService) ListServicePrices(
	ctx context.Context,
) ([]*types.BillingServicePrice, error) {
	return s.billing.ListServicePrices(ctx)
}

func (s *usageBillingService) CreateServicePriceVersion(
	ctx context.Context,
	input types.BillingServicePriceInput,
) (*types.BillingServicePrice, error) {
	return s.billing.CreateServicePriceVersion(ctx, input)
}

func (s *usageBillingService) ListUsageLedgers(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return s.billing.ListUsageLedgers(ctx, tenantID, limit)
}

func (s *usageBillingService) ListUsageLedgersByActor(
	ctx context.Context,
	tenantID uint64,
	actorUserID string,
	limit int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return s.billing.ListUsageLedgersByActor(ctx, tenantID, actorUserID, limit)
}

func (s *usageBillingService) ListUsageSummaryByActor(
	ctx context.Context,
	actorUserIDs []string,
) ([]*types.BillingActorUsageSummary, error) {
	return s.billing.ListUsageSummaryByActor(ctx, actorUserIDs)
}

func (s *usageBillingService) ListMemberAllocations(
	ctx context.Context,
	tenantID uint64,
	at time.Time,
) ([]*types.TenantMemberCreditAllocationSummary, error) {
	return s.billing.ListMemberAllocations(ctx, tenantID, at)
}

func (s *usageBillingService) GetTenantBillingPolicy(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantBillingPolicy, error) {
	subscription, plan, err := s.billing.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if subscription == nil || plan == nil {
		return nil, errors.New("billing: current subscription is required")
	}
	if plan.SpaceType != types.SpaceTypeOrganization {
		return nil, errors.New("billing: enterprise policy is available only for enterprise workspaces")
	}
	policy := s.policy.RuntimePolicy(ctx)
	return s.billing.EnsureTenantBillingPolicy(
		ctx,
		tenantID,
		policy.DefaultMemberMonthlyLimitPointMicros,
		"",
	)
}

func (s *usageBillingService) UpdateTenantBillingPolicy(
	ctx context.Context,
	tenantID uint64,
	defaultMemberMonthlyLimitPointMicros int64,
	memberOveragePolicy string,
	actorUserID string,
) (*types.TenantBillingPolicy, error) {
	if _, err := s.GetTenantBillingPolicy(ctx, tenantID); err != nil {
		return nil, err
	}
	switch memberOveragePolicy {
	case types.MemberOveragePolicyBlock, types.MemberOveragePolicyUseEnterpriseBalance:
	default:
		return nil, errors.New("billing: invalid enterprise member overage policy")
	}
	return s.billing.UpdateTenantBillingPolicy(
		ctx,
		tenantID,
		defaultMemberMonthlyLimitPointMicros,
		memberOveragePolicy,
		actorUserID,
	)
}

func (s *usageBillingService) UpdateMemberPolicy(
	ctx context.Context,
	tenantID uint64,
	userID string,
	limitMode string,
	monthlyLimitPointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	subscription, plan, err := s.billing.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if subscription == nil || plan == nil {
		return nil, errors.New("billing: current subscription is required")
	}
	if plan.SpaceType != types.SpaceTypeOrganization {
		return nil, errors.New("billing: member usage policies are available only for enterprise workspaces")
	}
	switch limitMode {
	case types.MemberLimitModeInherit, types.MemberLimitModeCustom, types.MemberLimitModeUnlimited:
	default:
		return nil, errors.New("billing: invalid member limit mode")
	}
	if monthlyLimitPointMicros < 0 {
		return nil, errors.New("billing: member monthly limit must be non-negative")
	}
	periodStart, periodEnd := usageBillingPeriod(subscription, time.Now().UTC())
	return s.billing.SetCurrentMemberPolicy(
		ctx,
		tenantID,
		userID,
		periodStart,
		periodEnd,
		limitMode,
		monthlyLimitPointMicros,
		actorUserID,
	)
}

func (s *usageBillingService) UpdateMemberPolicies(
	ctx context.Context,
	tenantID uint64,
	userIDs []string,
	limitMode string,
	monthlyLimitPointMicros int64,
	actorUserID string,
) ([]*types.TenantMemberCreditAllocation, error) {
	subscription, plan, err := s.billing.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if subscription == nil || plan == nil {
		return nil, errors.New("billing: current subscription is required")
	}
	if plan.SpaceType != types.SpaceTypeOrganization {
		return nil, errors.New("billing: member usage policies are available only for enterprise workspaces")
	}
	switch limitMode {
	case types.MemberLimitModeInherit, types.MemberLimitModeCustom, types.MemberLimitModeUnlimited:
	default:
		return nil, errors.New("billing: invalid member limit mode")
	}
	if monthlyLimitPointMicros < 0 {
		return nil, errors.New("billing: member monthly limit must be non-negative")
	}
	periodStart, periodEnd := usageBillingPeriod(subscription, time.Now().UTC())
	return s.billing.SetCurrentMemberPolicies(
		ctx,
		tenantID,
		userIDs,
		periodStart,
		periodEnd,
		limitMode,
		monthlyLimitPointMicros,
		actorUserID,
	)
}

func (s *usageBillingService) UpdateMemberAllocation(
	ctx context.Context,
	tenantID uint64,
	userID string,
	allocatedPeriodPointMicros int64,
	allocatedBalancePointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	return s.UpdateMemberPolicy(
		ctx,
		tenantID,
		userID,
		types.MemberLimitModeCustom,
		allocatedPeriodPointMicros+allocatedBalancePointMicros,
		actorUserID,
	)
}

func (s *usageBillingService) ListStorageTransactions(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingStorageTransactionSummary, error) {
	return s.billing.ListStorageTransactions(ctx, tenantID, limit)
}
