package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type billingRepository struct {
	db *gorm.DB
}

func NewBillingRepository(db *gorm.DB) interfaces.BillingRepository {
	return &billingRepository{db: db}
}

func defaultPlanForTenant(tenant *types.Tenant) (planCode, status string) {
	if tenant != nil && tenant.SpaceType != nil {
		switch *tenant.SpaceType {
		case types.SpaceTypePersonal:
			return types.BillingPlanPersonalFree, types.BillingStatusActive
		case types.SpaceTypeOrganization:
			return types.BillingPlanEnterpriseTeam, types.BillingStatusActive
		}
	}
	return types.BillingPlanLegacyCompat, types.BillingStatusLegacy
}

func (r *billingRepository) EnsureTenantBilling(ctx context.Context, tenant *types.Tenant) error {
	if tenant == nil || tenant.ID == 0 {
		return errors.New("billing: tenant is required")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		planCode, subscriptionStatus := defaultPlanForTenant(tenant)
		var plan types.BillingPlan
		if err := tx.Where("code = ?", planCode).First(&plan).Error; err != nil {
			return fmt.Errorf("billing: default plan %q is unavailable: %w", planCode, err)
		}

		initialBalance := tenant.EnterpriseCredits
		if initialBalance < 0 {
			initialBalance = 0
		}
		initialBalance *= types.PointMicrosPerPoint

		account := &types.TenantCreditAccount{
			ID:                 uuid.NewString(),
			TenantID:           tenant.ID,
			BalancePointMicros: initialBalance,
		}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_id"}},
			DoNothing: true,
		}).Create(account).Error; err != nil {
			return fmt.Errorf("billing: create credit account: %w", err)
		}
		persistedAccount := &types.TenantCreditAccount{}
		if err := tx.Where("tenant_id = ?", tenant.ID).First(persistedAccount).Error; err != nil {
			return fmt.Errorf("billing: load credit account: %w", err)
		}
		account = persistedAccount

		if account.BalancePointMicros > 0 {
			transaction := &types.TenantCreditTransaction{
				ID:                 uuid.NewString(),
				TenantID:           tenant.ID,
				AccountID:          account.ID,
				Type:               "legacy_import",
				AmountPointMicros:  account.BalancePointMicros,
				BalancePointMicros: account.BalancePointMicros,
				RefNo:              fmt.Sprintf("legacy-enterprise-credits:%d", tenant.ID),
				Description:        "Imported from tenants.enterprise_credits during billing initialization",
			}
			if err := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "ref_no"}},
				DoNothing: true,
			}).Create(transaction).Error; err != nil {
				return fmt.Errorf("billing: create opening credit transaction: %w", err)
			}
		}

		var currentCount int64
		if err := tx.Model(&types.TenantSubscription{}).
			Where("tenant_id = ? AND status IN ?", tenant.ID, []string{
				types.BillingStatusActive,
				types.BillingStatusTrialing,
				types.BillingStatusLegacy,
			}).
			Count(&currentCount).Error; err != nil {
			return fmt.Errorf("billing: count subscriptions: %w", err)
		}
		if currentCount > 0 {
			return nil
		}

		snapshot, err := json.Marshal(map[string]any{
			"source":                    "tenant_creation",
			"storage_quota_bytes":       tenant.StorageQuota,
			"legacy_enterprise_credits": tenant.EnterpriseCredits,
		})
		if err != nil {
			return fmt.Errorf("billing: create subscription snapshot: %w", err)
		}
		subscription := &types.TenantSubscription{
			ID:              uuid.NewString(),
			TenantID:        tenant.ID,
			PlanID:          plan.ID,
			Status:          subscriptionStatus,
			BillingInterval: "none",
			PriceSnapshot:   types.JSON(snapshot),
			Source:          "tenant_creation",
		}
		if err := tx.Create(subscription).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				return nil
			}
			return fmt.Errorf("billing: create subscription: %w", err)
		}
		return nil
	})
}

func (r *billingRepository) GetCurrentSubscription(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantSubscription, *types.BillingPlan, error) {
	var subscription types.TenantSubscription
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND status IN ?", tenantID, []string{
			types.BillingStatusActive,
			types.BillingStatusTrialing,
			types.BillingStatusLegacy,
		}).
		Order("created_at DESC").
		First(&subscription).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, nil
		}
		return nil, nil, err
	}

	var plan types.BillingPlan
	if err := r.db.WithContext(ctx).Where("id = ?", subscription.PlanID).First(&plan).Error; err != nil {
		return nil, nil, err
	}
	return &subscription, &plan, nil
}

func (r *billingRepository) GetCreditAccount(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantCreditAccount, error) {
	var account types.TenantCreditAccount
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

func (r *billingRepository) ListPlans(ctx context.Context) ([]*types.BillingPlan, error) {
	var rows []*types.BillingPlan
	err := r.db.WithContext(ctx).Order("is_public DESC, created_at ASC, code ASC").Find(&rows).Error
	return rows, err
}

func (r *billingRepository) ListPrices(ctx context.Context) ([]*types.BillingPrice, error) {
	var rows []*types.BillingPrice
	err := r.db.WithContext(ctx).Order("created_at ASC, code ASC").Find(&rows).Error
	return rows, err
}

func (r *billingRepository) ListSubscriptions(
	ctx context.Context,
) ([]*types.BillingSubscriptionSummary, error) {
	var rows []*types.BillingSubscriptionSummary
	err := r.db.WithContext(ctx).
		Table("tenant_subscriptions").
		Select(`
			tenant_subscriptions.*,
			tenants.name AS tenant_name,
			COALESCE(tenants.space_type, 'legacy') AS space_type,
			billing_plans.code AS plan_code,
			billing_plans.name AS plan_name
		`).
		Joins("JOIN tenants ON tenants.id = tenant_subscriptions.tenant_id AND tenants.deleted_at IS NULL").
		Joins("JOIN billing_plans ON billing_plans.id = tenant_subscriptions.plan_id").
		Order("tenant_subscriptions.created_at DESC").
		Scan(&rows).Error
	return rows, err
}

func (r *billingRepository) ListCurrentSubscriptionsByTenantIDs(
	ctx context.Context,
	tenantIDs []uint64,
) ([]*types.BillingSubscriptionSummary, error) {
	if len(tenantIDs) == 0 {
		return []*types.BillingSubscriptionSummary{}, nil
	}

	var candidates []*types.BillingSubscriptionSummary
	err := r.db.WithContext(ctx).
		Table("tenant_subscriptions").
		Select(`
			tenant_subscriptions.*,
			tenants.name AS tenant_name,
			COALESCE(tenants.space_type, 'legacy') AS space_type,
			billing_plans.code AS plan_code,
			billing_plans.name AS plan_name
		`).
		Joins("JOIN tenants ON tenants.id = tenant_subscriptions.tenant_id AND tenants.deleted_at IS NULL").
		Joins("JOIN billing_plans ON billing_plans.id = tenant_subscriptions.plan_id").
		Where("tenant_subscriptions.tenant_id IN ?", tenantIDs).
		Where("tenant_subscriptions.status IN ?", []string{
			types.BillingStatusActive,
			types.BillingStatusTrialing,
			types.BillingStatusLegacy,
		}).
		Order("tenant_subscriptions.tenant_id ASC, tenant_subscriptions.created_at DESC").
		Scan(&candidates).Error
	if err != nil {
		return nil, err
	}

	rows := make([]*types.BillingSubscriptionSummary, 0, len(tenantIDs))
	seen := make(map[uint64]struct{}, len(tenantIDs))
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if _, exists := seen[candidate.TenantID]; exists {
			continue
		}
		seen[candidate.TenantID] = struct{}{}
		rows = append(rows, candidate)
	}
	return rows, nil
}

func (r *billingRepository) ListCreditAccounts(
	ctx context.Context,
) ([]*types.BillingCreditAccountSummary, error) {
	var rows []*types.BillingCreditAccountSummary
	err := r.db.WithContext(ctx).
		Table("tenant_credit_accounts").
		Select(`
			tenant_credit_accounts.*,
			tenants.name AS tenant_name,
			COALESCE(tenants.space_type, 'legacy') AS space_type
		`).
		Joins("JOIN tenants ON tenants.id = tenant_credit_accounts.tenant_id AND tenants.deleted_at IS NULL").
		Order("tenant_credit_accounts.created_at DESC").
		Scan(&rows).Error
	return rows, err
}
