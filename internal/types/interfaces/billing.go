package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

type BillingRepository interface {
	EnsureTenantBilling(ctx context.Context, tenant *types.Tenant) error
	GetCurrentSubscription(ctx context.Context, tenantID uint64) (*types.TenantSubscription, *types.BillingPlan, error)
	GetCreditAccount(ctx context.Context, tenantID uint64) (*types.TenantCreditAccount, error)
	ListPlans(ctx context.Context) ([]*types.BillingPlan, error)
	ListPrices(ctx context.Context) ([]*types.BillingPrice, error)
	ListSubscriptions(ctx context.Context) ([]*types.BillingSubscriptionSummary, error)
	ListCurrentSubscriptionsByTenantIDs(ctx context.Context, tenantIDs []uint64) ([]*types.BillingSubscriptionSummary, error)
	ListCreditAccounts(ctx context.Context) ([]*types.BillingCreditAccountSummary, error)
	ListModelPrices(ctx context.Context) ([]*types.BillingModelPrice, error)
	CreateModelPriceVersion(ctx context.Context, input types.BillingModelPriceInput) (*types.BillingModelPrice, error)
	GetActiveModelPrice(ctx context.Context, modelKeys ...string) (*types.BillingModelPrice, error)
	EnsureTenantBillingPolicy(
		ctx context.Context,
		tenantID uint64,
		defaultMemberMonthlyLimitPointMicros int64,
		actorUserID string,
	) (*types.TenantBillingPolicy, error)
	UpdateTenantBillingPolicy(
		ctx context.Context,
		tenantID uint64,
		defaultMemberMonthlyLimitPointMicros int64,
		memberOveragePolicy string,
		actorUserID string,
	) (*types.TenantBillingPolicy, error)
	EnsureCurrentMemberAllocation(
		ctx context.Context,
		tenantID uint64,
		userID string,
		periodStart time.Time,
		periodEnd time.Time,
		defaultBalancePointMicros int64,
		actorUserID string,
	) (*types.TenantMemberCreditAllocation, error)
	SetCurrentMemberAllocation(
		ctx context.Context,
		tenantID uint64,
		userID string,
		periodStart time.Time,
		periodEnd time.Time,
		allocatedPeriodPointMicros int64,
		allocatedBalancePointMicros int64,
		actorUserID string,
	) (*types.TenantMemberCreditAllocation, error)
	SetCurrentMemberPolicy(
		ctx context.Context,
		tenantID uint64,
		userID string,
		periodStart time.Time,
		periodEnd time.Time,
		limitMode string,
		monthlyLimitPointMicros int64,
		actorUserID string,
	) (*types.TenantMemberCreditAllocation, error)
	CreateUsageReservation(
		ctx context.Context,
		handle *types.BillingUsageHandle,
		estimatedBaseCostNanoUSD int64,
		estimatedBilledPointMicros int64,
	) (*types.TenantUsageReservation, error)
	ReleaseUsageReservation(ctx context.Context, tenantID uint64, refNo, failureCode string) error
	SettleUsage(ctx context.Context, handle *types.BillingUsageHandle, ledger *types.TenantUsageLedger) (*types.TenantUsageLedger, error)
	ListUsageLedgers(ctx context.Context, tenantID uint64, limit int) ([]*types.BillingUsageLedgerSummary, error)
	ListUsageSummaryByActor(ctx context.Context, actorUserIDs []string) ([]*types.BillingActorUsageSummary, error)
	ListMemberAllocations(ctx context.Context, tenantID uint64, at time.Time) ([]*types.TenantMemberCreditAllocationSummary, error)
}

type BillingPolicyService interface {
	RuntimePolicy(ctx context.Context) types.BillingRuntimePolicy
}

type SubscriptionService interface {
	EnsureTenantBilling(ctx context.Context, tenant *types.Tenant) error
	GetOverview(ctx context.Context, tenantID uint64) (*types.BillingOverview, error)
	ListPlans(ctx context.Context) ([]*types.BillingPlan, error)
	ListPrices(ctx context.Context) ([]*types.BillingPrice, error)
	ListSubscriptions(ctx context.Context) ([]*types.BillingSubscriptionSummary, error)
	ListCurrentSubscriptionsByTenantIDs(ctx context.Context, tenantIDs []uint64) ([]*types.BillingSubscriptionSummary, error)
	ListCreditAccounts(ctx context.Context) ([]*types.BillingCreditAccountSummary, error)
}

type UsageBillingService interface {
	BeginModelUsage(ctx context.Context, req types.BillingUsageStartRequest) (*types.BillingUsageHandle, error)
	SettleModelUsage(
		ctx context.Context,
		handle *types.BillingUsageHandle,
		usage types.BillingModelUsage,
	) (*types.TenantUsageLedger, error)
	ReleaseModelUsage(ctx context.Context, handle *types.BillingUsageHandle, failureCode string) error
	ListModelPrices(ctx context.Context) ([]*types.BillingModelPrice, error)
	CreateModelPriceVersion(ctx context.Context, input types.BillingModelPriceInput) (*types.BillingModelPrice, error)
	ListUsageLedgers(ctx context.Context, tenantID uint64, limit int) ([]*types.BillingUsageLedgerSummary, error)
	ListUsageSummaryByActor(ctx context.Context, actorUserIDs []string) ([]*types.BillingActorUsageSummary, error)
	ListMemberAllocations(ctx context.Context, tenantID uint64, at time.Time) ([]*types.TenantMemberCreditAllocationSummary, error)
	GetTenantBillingPolicy(ctx context.Context, tenantID uint64) (*types.TenantBillingPolicy, error)
	UpdateTenantBillingPolicy(
		ctx context.Context,
		tenantID uint64,
		defaultMemberMonthlyLimitPointMicros int64,
		memberOveragePolicy string,
		actorUserID string,
	) (*types.TenantBillingPolicy, error)
	UpdateMemberPolicy(
		ctx context.Context,
		tenantID uint64,
		userID string,
		limitMode string,
		monthlyLimitPointMicros int64,
		actorUserID string,
	) (*types.TenantMemberCreditAllocation, error)
	UpdateMemberAllocation(
		ctx context.Context,
		tenantID uint64,
		userID string,
		allocatedPeriodPointMicros int64,
		allocatedBalancePointMicros int64,
		actorUserID string,
	) (*types.TenantMemberCreditAllocation, error)
}
