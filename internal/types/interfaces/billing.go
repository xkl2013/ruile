package interfaces

import (
	"context"

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
	CreateUsageReservation(
		ctx context.Context,
		handle *types.BillingUsageHandle,
		estimatedBaseCostNanoUSD int64,
		estimatedBilledPointMicros int64,
	) (*types.TenantUsageReservation, error)
	ReleaseUsageReservation(ctx context.Context, tenantID uint64, refNo, failureCode string) error
	SettleUsage(ctx context.Context, handle *types.BillingUsageHandle, ledger *types.TenantUsageLedger) (*types.TenantUsageLedger, error)
	ListUsageLedgers(ctx context.Context, tenantID uint64, limit int) ([]*types.BillingUsageLedgerSummary, error)
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
}
