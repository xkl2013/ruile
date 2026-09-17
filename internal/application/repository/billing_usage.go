package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/types"
)

var ErrInsufficientBillingCredits = errors.New("billing: insufficient credits")

func (r *billingRepository) ListModelPrices(ctx context.Context) ([]*types.BillingModelPrice, error) {
	var rows []*types.BillingModelPrice
	err := r.db.WithContext(ctx).
		Order("model_key ASC, version DESC").
		Find(&rows).Error
	return rows, err
}

func (r *billingRepository) CreateModelPriceVersion(
	ctx context.Context,
	input types.BillingModelPriceInput,
) (*types.BillingModelPrice, error) {
	modelKey := strings.TrimSpace(input.ModelKey)
	if modelKey == "" {
		return nil, errors.New("billing: model_key is required")
	}
	mode := strings.ToLower(strings.TrimSpace(input.PricingMode))
	if mode == "" {
		mode = "token"
	}
	switch mode {
	case "token", "call", "duration":
	default:
		return nil, fmt.Errorf("billing: pricing_mode %q is not supported in step 2A", mode)
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = types.BillingStatusActive
	}
	if status != types.BillingStatusActive && status != "disabled" {
		return nil, fmt.Errorf("billing: invalid model price status %q", status)
	}
	rates := []int64{
		input.InputNanoUSDPerMTokens,
		input.OutputNanoUSDPerMTokens,
		input.CacheReadNanoUSDPerMTokens,
		input.CacheWriteNanoUSDPerMTokens,
		input.CallNanoUSDPerCall,
		input.DurationNanoUSDPerSecond,
	}
	for _, rate := range rates {
		if rate < 0 {
			return nil, errors.New("billing: model prices must be non-negative")
		}
	}
	multiplier := input.ModelMultiplierPPM
	if multiplier <= 0 {
		multiplier = 1_000_000
	}
	effectiveAt := time.Now().UTC()
	if input.EffectiveAt != nil {
		effectiveAt = input.EffectiveAt.UTC()
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(effectiveAt) {
		return nil, errors.New("billing: expires_at must be after effective_at")
	}

	var created *types.BillingModelPrice
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVersion int
		if err := tx.Model(&types.BillingModelPrice{}).
			Where("model_key = ?", modelKey).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}
		snapshot, err := json.Marshal(map[string]any{
			"model_key":                        modelKey,
			"provider":                         strings.TrimSpace(input.Provider),
			"pricing_mode":                     mode,
			"input_nanousd_per_m_tokens":       input.InputNanoUSDPerMTokens,
			"output_nanousd_per_m_tokens":      input.OutputNanoUSDPerMTokens,
			"cache_read_nanousd_per_m_tokens":  input.CacheReadNanoUSDPerMTokens,
			"cache_write_nanousd_per_m_tokens": input.CacheWriteNanoUSDPerMTokens,
			"call_nanousd_per_call":            input.CallNanoUSDPerCall,
			"duration_nanousd_per_second":      input.DurationNanoUSDPerSecond,
			"model_multiplier_ppm":             multiplier,
			"version":                          maxVersion + 1,
			"effective_at":                     effectiveAt,
		})
		if err != nil {
			return err
		}
		created = &types.BillingModelPrice{
			ID:                          uuid.NewString(),
			ModelKey:                    modelKey,
			Provider:                    strings.TrimSpace(input.Provider),
			PricingMode:                 mode,
			InputNanoUSDPerMTokens:      input.InputNanoUSDPerMTokens,
			OutputNanoUSDPerMTokens:     input.OutputNanoUSDPerMTokens,
			CacheReadNanoUSDPerMTokens:  input.CacheReadNanoUSDPerMTokens,
			CacheWriteNanoUSDPerMTokens: input.CacheWriteNanoUSDPerMTokens,
			CallNanoUSDPerCall:          input.CallNanoUSDPerCall,
			DurationNanoUSDPerSecond:    input.DurationNanoUSDPerSecond,
			TieredPricingJSON:           types.JSON([]byte("{}")),
			ModelMultiplierPPM:          multiplier,
			Version:                     maxVersion + 1,
			EffectiveAt:                 effectiveAt,
			ExpiresAt:                   input.ExpiresAt,
			Status:                      status,
			SnapshotJSON:                types.JSON(snapshot),
		}
		return tx.Create(created).Error
	})
	return created, err
}

func (r *billingRepository) GetActiveModelPrice(
	ctx context.Context,
	modelKeys ...string,
) (*types.BillingModelPrice, error) {
	now := time.Now().UTC()
	seen := make(map[string]struct{}, len(modelKeys))
	for _, raw := range modelKeys {
		key := strings.TrimSpace(raw)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		var price types.BillingModelPrice
		err := r.db.WithContext(ctx).
			Where(
				"model_key = ? AND status = ? AND effective_at <= ? AND (expires_at IS NULL OR expires_at > ?)",
				key,
				types.BillingStatusActive,
				now,
				now,
			).
			Order("version DESC").
			First(&price).Error
		if err == nil {
			return &price, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	return nil, nil
}

func billingPeriod(subscription *types.TenantSubscription, now time.Time) (time.Time, time.Time) {
	if subscription != nil && subscription.CurrentPeriodStart != nil && subscription.CurrentPeriodEnd != nil &&
		subscription.CurrentPeriodEnd.After(*subscription.CurrentPeriodStart) {
		return subscription.CurrentPeriodStart.UTC(), subscription.CurrentPeriodEnd.UTC()
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func (r *billingRepository) CreateUsageReservation(
	ctx context.Context,
	handle *types.BillingUsageHandle,
	estimatedBaseCostNanoUSD int64,
	estimatedBilledPointMicros int64,
) (*types.TenantUsageReservation, error) {
	if handle == nil || handle.Plan == nil || handle.Subscription == nil {
		return nil, errors.New("billing: subscription context is required")
	}
	var result *types.TenantUsageReservation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.TenantUsageReservation
		err := tx.Where(
			"tenant_id = ? AND ref_no = ?",
			handle.Request.TenantID,
			handle.Request.RefNo,
		).First(&existing).Error
		if err == nil {
			result = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var account types.TenantCreditAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ?", handle.Request.TenantID).
			First(&account).Error; err != nil {
			return fmt.Errorf("billing: load credit account: %w", err)
		}

		now := time.Now().UTC()
		periodStart, periodEnd := billingPeriod(handle.Subscription, now)
		var usedPeriod int64
		if err := tx.Model(&types.TenantUsageLedger{}).
			Where(
				"tenant_id = ? AND status = ? AND billing_at >= ? AND billing_at < ?",
				handle.Request.TenantID,
				"settled",
				periodStart,
				periodEnd,
			).
			Select("COALESCE(SUM(period_covered_point_micros), 0)").
			Scan(&usedPeriod).Error; err != nil {
			return err
		}
		var reservedPeriod, reservedBalance int64
		if err := tx.Model(&types.TenantUsageReservation{}).
			Where(
				"tenant_id = ? AND status = ? AND expires_at > ?",
				handle.Request.TenantID,
				"active",
				now,
			).
			Select(`
				COALESCE(SUM(reserved_period_point_micros), 0),
				COALESCE(SUM(reserved_balance_point_micros), 0)
			`).
			Row().
			Scan(&reservedPeriod, &reservedBalance); err != nil {
			return err
		}

		availablePeriod := handle.Plan.IncludedPointMicros - usedPeriod - reservedPeriod
		if availablePeriod < 0 {
			availablePeriod = 0
		}
		availableBalance := account.BalancePointMicros - reservedBalance
		if availableBalance < 0 {
			availableBalance = 0
		}
		if estimatedBilledPointMicros > availablePeriod+availableBalance {
			return ErrInsufficientBillingCredits
		}
		periodReserve := min(estimatedBilledPointMicros, availablePeriod)
		balanceReserve := estimatedBilledPointMicros - periodReserve
		periodStartCopy, periodEndCopy := periodStart, periodEnd
		result = &types.TenantUsageReservation{
			ID:                         uuid.NewString(),
			TenantID:                   handle.Request.TenantID,
			ActorUserID:                handle.Request.ActorUserID,
			UsageScope:                 handle.UsageScope,
			RefNo:                      handle.Request.RefNo,
			ModelKey:                   handle.Request.ModelKey,
			EstimatedBaseCostNanoUSD:   estimatedBaseCostNanoUSD,
			EstimatedBilledPointMicros: estimatedBilledPointMicros,
			ReservedPeriodPointMicros:  periodReserve,
			ReservedBalancePointMicros: balanceReserve,
			PeriodLimitPointMicros:     handle.Plan.IncludedPointMicros,
			PeriodStartAt:              &periodStartCopy,
			PeriodEndAt:                &periodEndCopy,
			Status:                     "active",
			ExpiresAt:                  now.Add(15 * time.Minute),
		}
		if handle.Price != nil {
			result.PricingID = handle.Price.ID
		}
		return tx.Create(result).Error
	})
	return result, err
}

func (r *billingRepository) ReleaseUsageReservation(
	ctx context.Context,
	tenantID uint64,
	refNo, failureCode string,
) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&types.TenantUsageReservation{}).
		Where("tenant_id = ? AND ref_no = ? AND status = ?", tenantID, refNo, "active").
		Updates(map[string]any{
			"status":       "released",
			"released_at":  now,
			"failure_code": strings.TrimSpace(failureCode),
			"updated_at":   now,
		}).Error
}

func (r *billingRepository) SettleUsage(
	ctx context.Context,
	handle *types.BillingUsageHandle,
	ledger *types.TenantUsageLedger,
) (*types.TenantUsageLedger, error) {
	if handle == nil || ledger == nil {
		return nil, errors.New("billing: usage handle and ledger are required")
	}
	var result *types.TenantUsageLedger
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.TenantUsageLedger
		err := tx.Where(
			"tenant_id = ? AND ref_no = ?",
			ledger.TenantID,
			ledger.RefNo,
		).First(&existing).Error
		if err == nil {
			result = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if handle.Mode != "enforce" {
			if err := tx.Create(ledger).Error; err != nil {
				return err
			}
			result = ledger
			return nil
		}

		var reservation types.TenantUsageReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND ref_no = ?", ledger.TenantID, ledger.RefNo).
			First(&reservation).Error; err != nil {
			return fmt.Errorf("billing: load usage reservation: %w", err)
		}
		if reservation.Status == "settled" && reservation.UsageLedgerID != "" {
			if err := tx.Where("id = ?", reservation.UsageLedgerID).First(&existing).Error; err != nil {
				return err
			}
			result = &existing
			return nil
		}
		if reservation.Status != "active" {
			return fmt.Errorf("billing: reservation is %s", reservation.Status)
		}

		var account types.TenantCreditAccount
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ?", ledger.TenantID).
			First(&account).Error; err != nil {
			return fmt.Errorf("billing: load credit account: %w", err)
		}
		periodStart, periodEnd := billingPeriod(handle.Subscription, time.Now().UTC())
		var usedPeriod int64
		if err := tx.Model(&types.TenantUsageLedger{}).
			Where(
				"tenant_id = ? AND status = ? AND billing_at >= ? AND billing_at < ?",
				ledger.TenantID,
				"settled",
				periodStart,
				periodEnd,
			).
			Select("COALESCE(SUM(period_covered_point_micros), 0)").
			Scan(&usedPeriod).Error; err != nil {
			return err
		}
		var otherReservedPeriod, otherReservedBalance int64
		if err := tx.Model(&types.TenantUsageReservation{}).
			Where(
				"tenant_id = ? AND ref_no <> ? AND status = ? AND expires_at > ?",
				ledger.TenantID,
				ledger.RefNo,
				"active",
				time.Now().UTC(),
			).
			Select(`
				COALESCE(SUM(reserved_period_point_micros), 0),
				COALESCE(SUM(reserved_balance_point_micros), 0)
			`).
			Row().
			Scan(&otherReservedPeriod, &otherReservedBalance); err != nil {
			return err
		}
		availablePeriod := handle.Plan.IncludedPointMicros - usedPeriod - otherReservedPeriod
		if availablePeriod < 0 {
			availablePeriod = 0
		}
		availableBalance := account.BalancePointMicros - otherReservedBalance
		if availableBalance < 0 {
			availableBalance = 0
		}
		periodCharge := min(ledger.BilledPointMicros, availablePeriod)
		balanceCharge := ledger.BilledPointMicros - periodCharge
		if balanceCharge > availableBalance {
			now := time.Now().UTC()
			ledger.Status = "reconciliation"
			ledger.FailureCode = "actual_cost_exceeds_available_credits"
			ledger.PeriodCoveredPointMicros = 0
			ledger.BalanceChargedPointMicros = 0
			balanceAfter := account.BalancePointMicros
			ledger.BalanceAfterPointMicros = &balanceAfter
			if err := tx.Create(ledger).Error; err != nil {
				return err
			}
			if err := tx.Model(&reservation).Updates(map[string]any{
				"status":            "reconciliation",
				"usage_ledger_id":   ledger.ID,
				"reconciliation_at": now,
				"failure_code":      ledger.FailureCode,
				"updated_at":        now,
			}).Error; err != nil {
				return err
			}
			result = ledger
			return nil
		}

		ledger.Status = "settled"
		ledger.PeriodCoveredPointMicros = periodCharge
		ledger.BalanceChargedPointMicros = balanceCharge
		balanceAfter := account.BalancePointMicros - balanceCharge
		ledger.BalanceAfterPointMicros = &balanceAfter
		if balanceCharge > 0 {
			if err := tx.Model(&types.TenantCreditAccount{}).
				Where("id = ? AND version = ?", account.ID, account.Version).
				Updates(map[string]any{
					"balance_point_micros": balanceAfter,
					"version":              account.Version + 1,
					"updated_at":           time.Now().UTC(),
				}).Error; err != nil {
				return err
			}
			transaction := &types.TenantCreditTransaction{
				ID:                 uuid.NewString(),
				TenantID:           ledger.TenantID,
				AccountID:          account.ID,
				Type:               "usage_debit",
				AmountPointMicros:  -balanceCharge,
				BalancePointMicros: balanceAfter,
				RefNo:              "usage-debit:" + ledger.ID,
				Description:        "Model usage settlement",
			}
			if err := tx.Create(transaction).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(ledger).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&reservation).Updates(map[string]any{
			"status":          "settled",
			"usage_ledger_id": ledger.ID,
			"settled_at":      now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}
		result = ledger
		return nil
	})
	return result, err
}

func (r *billingRepository) ListUsageLedgers(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingUsageLedgerSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).
		Table("tenant_usage_ledgers").
		Select("tenant_usage_ledgers.*, tenants.name AS tenant_name").
		Joins("JOIN tenants ON tenants.id = tenant_usage_ledgers.tenant_id AND tenants.deleted_at IS NULL")
	if tenantID > 0 {
		query = query.Where("tenant_usage_ledgers.tenant_id = ?", tenantID)
	}
	var rows []*types.BillingUsageLedgerSummary
	err := query.
		Order("tenant_usage_ledgers.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}
