package service

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type subscriptionService struct {
	tenants interfaces.TenantRepository
	billing interfaces.BillingRepository
	policy  interfaces.BillingPolicyService
}

func NewSubscriptionService(
	tenants interfaces.TenantRepository,
	billing interfaces.BillingRepository,
	policy interfaces.BillingPolicyService,
) interfaces.SubscriptionService {
	return &subscriptionService{
		tenants: tenants,
		billing: billing,
		policy:  policy,
	}
}

func (s *subscriptionService) EnsureTenantBilling(ctx context.Context, tenant *types.Tenant) error {
	if s.billing == nil {
		return errors.New("billing repository is not configured")
	}
	return s.billing.EnsureTenantBilling(ctx, tenant)
}

func (s *subscriptionService) GetOverview(ctx context.Context, tenantID uint64) (*types.BillingOverview, error) {
	tenant, err := s.tenants.GetTenantByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	subscription, plan, err := s.billing.GetCurrentSubscription(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	account, err := s.billing.GetCreditAccount(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if subscription == nil || plan == nil || account == nil {
		logger.Warnf(ctx, "[billing] missing billing records for tenant=%d; running idempotent repair", tenantID)
		if err := s.billing.EnsureTenantBilling(ctx, tenant); err != nil {
			return nil, err
		}
		subscription, plan, err = s.billing.GetCurrentSubscription(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		account, err = s.billing.GetCreditAccount(ctx, tenantID)
		if err != nil {
			return nil, err
		}
	}
	if subscription == nil || plan == nil || account == nil {
		return nil, errors.New("billing overview is incomplete after repair")
	}
	periodStart, periodEnd := billingOverviewPeriod(subscription, time.Now().UTC())
	periodUsed, err := s.billing.GetPeriodUsedPointMicros(ctx, tenantID, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	periodRemaining := plan.IncludedPointMicros - periodUsed
	if periodRemaining < 0 {
		periodRemaining = 0
	}

	spaceType := types.SpaceTypeLegacy
	if tenant.SpaceType != nil {
		spaceType = *tenant.SpaceType
	}
	unlimited := tenant.StorageQuota <= 0
	remaining := int64(0)
	percent := float64(0)
	status := "ok"
	if !unlimited {
		remaining = tenant.StorageQuota - tenant.StorageUsed
		if remaining < 0 {
			remaining = 0
		}
		if tenant.StorageQuota > 0 {
			percent = math.Max(0, float64(tenant.StorageUsed)/float64(tenant.StorageQuota)*100)
		}
		switch {
		case tenant.StorageUsed >= tenant.StorageQuota:
			status = "exceeded"
		case percent >= 80:
			status = "warning"
		}
	}

	policy := s.policy.RuntimePolicy(ctx)
	compatibilityMode := plan.Code == types.BillingPlanLegacyCompat || subscription.Status == types.BillingStatusLegacy
	if policy.EnforcementMode == "observe" {
		logger.Infof(
			ctx,
			"[billing][observe] tenant=%d space_type=%s plan=%s subscription=%s compatibility=%t enabled=%t",
			tenantID,
			spaceType,
			plan.Code,
			subscription.Status,
			compatibilityMode,
			policy.Enabled,
		)
	}

	return &types.BillingOverview{
		TenantID:   tenant.ID,
		TenantName: tenant.Name,
		SpaceType:  spaceType,
		Policy:     policy,
		Plan: types.BillingOverviewPlan{
			Code:                 plan.Code,
			Name:                 plan.Name,
			Edition:              plan.Edition,
			Status:               plan.Status,
			IncludedStorageBytes: plan.IncludedStorageBytes,
			IncludedPointMicros:  plan.IncludedPointMicros,
		},
		Subscription: types.BillingOverviewSubscription{
			Status:             subscription.Status,
			BillingInterval:    subscription.BillingInterval,
			CurrentPeriodStart: subscription.CurrentPeriodStart,
			CurrentPeriodEnd:   subscription.CurrentPeriodEnd,
			Source:             subscription.Source,
		},
		Storage: types.BillingOverviewStorage{
			UsedBytes:      tenant.StorageUsed,
			QuotaBytes:     tenant.StorageQuota,
			RemainingBytes: remaining,
			UsagePercent:   percent,
			Unlimited:      unlimited,
			Status:         status,
			Visible:        true,
		},
		Credits: types.BillingOverviewCredits{
			BalancePointMicros:         account.BalancePointMicros,
			PeriodPointMicros:          plan.IncludedPointMicros,
			PeriodUsedPointMicros:      periodUsed,
			PeriodRemainingPointMicros: periodRemaining,
			Visible:                    true,
		},
		CompatibilityMode: compatibilityMode,
	}, nil
}

func billingOverviewPeriod(subscription *types.TenantSubscription, now time.Time) (time.Time, time.Time) {
	if subscription != nil &&
		subscription.CurrentPeriodStart != nil &&
		subscription.CurrentPeriodEnd != nil &&
		subscription.CurrentPeriodEnd.After(*subscription.CurrentPeriodStart) {
		return subscription.CurrentPeriodStart.UTC(), subscription.CurrentPeriodEnd.UTC()
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func (s *subscriptionService) ListPlans(ctx context.Context) ([]*types.BillingPlan, error) {
	return s.billing.ListPlans(ctx)
}

func (s *subscriptionService) ListPrices(ctx context.Context) ([]*types.BillingPrice, error) {
	return s.billing.ListPrices(ctx)
}

func (s *subscriptionService) ListSubscriptions(
	ctx context.Context,
) ([]*types.BillingSubscriptionSummary, error) {
	return s.billing.ListSubscriptions(ctx)
}

func (s *subscriptionService) ListCurrentSubscriptionsByTenantIDs(
	ctx context.Context,
	tenantIDs []uint64,
) ([]*types.BillingSubscriptionSummary, error) {
	return s.billing.ListCurrentSubscriptionsByTenantIDs(ctx, tenantIDs)
}

func (s *subscriptionService) ListCreditAccounts(
	ctx context.Context,
) ([]*types.BillingCreditAccountSummary, error) {
	return s.billing.ListCreditAccounts(ctx)
}
