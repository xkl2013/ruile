package repository

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/types"
)

var ErrInsufficientBillingCredits = errors.New("billing: insufficient credits")

type aggregateTime struct {
	Time  time.Time
	Valid bool
}

func (t *aggregateTime) Scan(value any) error {
	if value == nil {
		t.Valid = false
		t.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		t.Time = v.UTC()
		t.Valid = true
		return nil
	case string:
		return t.scanString(v)
	case []byte:
		return t.scanString(string(v))
	default:
		return fmt.Errorf("unsupported aggregate time type %T", value)
	}
}

func (t aggregateTime) Value() (driver.Value, error) {
	if !t.Valid {
		return nil, nil
	}
	return t.Time, nil
}

func (t *aggregateTime) scanString(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		t.Valid = false
		t.Time = time.Time{}
		return nil
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		t.Time = time.Unix(unix, 0).UTC()
		t.Valid = true
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05",
	}
	var parseErr error
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			t.Time = parsed.UTC()
			t.Valid = true
			return nil
		}
		parseErr = err
	}
	return parseErr
}

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
	case "token", "call", "duration", "tiered":
	default:
		return nil, fmt.Errorf("billing: pricing_mode %q is not supported", mode)
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
	tieredPricing := input.TieredPricingJSON
	if len(tieredPricing) == 0 {
		tieredPricing = types.JSON([]byte("{}"))
	}
	if !json.Valid(tieredPricing) {
		return nil, errors.New("billing: tiered_pricing_json must be valid JSON")
	}
	if mode == "tiered" && string(tieredPricing) == "{}" {
		return nil, errors.New("billing: tiered pricing requires at least one tier")
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
			"tiered_pricing_json":              json.RawMessage(tieredPricing),
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
			TieredPricingJSON:           tieredPricing,
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

func normalizeMemberLimitMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case types.MemberLimitModeCustom:
		return types.MemberLimitModeCustom
	case types.MemberLimitModeUnlimited:
		return types.MemberLimitModeUnlimited
	default:
		return types.MemberLimitModeInherit
	}
}

func normalizeMemberOveragePolicy(policy string) string {
	switch strings.ToLower(strings.TrimSpace(policy)) {
	case types.MemberOveragePolicyUseEnterpriseBalance:
		return types.MemberOveragePolicyUseEnterpriseBalance
	default:
		return types.MemberOveragePolicyBlock
	}
}

func normalizeStoredMemberAllocation(allocation *types.TenantMemberCreditAllocation) {
	if allocation == nil {
		return
	}
	allocation.LimitMode = normalizeMemberLimitMode(allocation.LimitMode)
	if allocation.MonthlyLimitPointMicros < 0 {
		allocation.MonthlyLimitPointMicros = 0
	}
	if allocation.LimitMode == types.MemberLimitModeInherit {
		legacyLimit := allocation.AllocatedPeriodPointMicros + allocation.AllocatedBalancePointMicros
		if legacyLimit > 0 {
			allocation.LimitMode = types.MemberLimitModeCustom
			allocation.MonthlyLimitPointMicros = legacyLimit
		}
	}
	if allocation.LimitMode != types.MemberLimitModeCustom {
		allocation.MonthlyLimitPointMicros = 0
	}
	if strings.TrimSpace(allocation.OveragePolicy) == "" {
		allocation.OveragePolicy = types.MemberOveragePolicyInherit
	}
}

func (r *billingRepository) EnsureTenantBillingPolicy(
	ctx context.Context,
	tenantID uint64,
	defaultMemberMonthlyLimitPointMicros int64,
	actorUserID string,
) (*types.TenantBillingPolicy, error) {
	if tenantID == 0 {
		return nil, errors.New("billing: tenant_id is required for enterprise policy")
	}
	if defaultMemberMonthlyLimitPointMicros < 0 {
		defaultMemberMonthlyLimitPointMicros = 0
	}
	policy := &types.TenantBillingPolicy{
		TenantID:                             tenantID,
		DefaultMemberMonthlyLimitPointMicros: defaultMemberMonthlyLimitPointMicros,
		MemberOveragePolicy:                  types.MemberOveragePolicyBlock,
		UpdatedByUserID:                      strings.TrimSpace(actorUserID),
	}
	if err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}},
		DoNothing: true,
	}).Create(policy).Error; err != nil {
		return nil, err
	}
	var persisted types.TenantBillingPolicy
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&persisted).Error; err != nil {
		return nil, err
	}
	persisted.MemberOveragePolicy = normalizeMemberOveragePolicy(persisted.MemberOveragePolicy)
	if persisted.DefaultMemberMonthlyLimitPointMicros < 0 {
		persisted.DefaultMemberMonthlyLimitPointMicros = 0
	}
	return &persisted, nil
}

func (r *billingRepository) UpdateTenantBillingPolicy(
	ctx context.Context,
	tenantID uint64,
	defaultMemberMonthlyLimitPointMicros int64,
	memberOveragePolicy string,
	actorUserID string,
) (*types.TenantBillingPolicy, error) {
	if tenantID == 0 {
		return nil, errors.New("billing: tenant_id is required for enterprise policy")
	}
	if defaultMemberMonthlyLimitPointMicros < 0 {
		return nil, errors.New("billing: default member monthly limit must be non-negative")
	}
	memberOveragePolicy = normalizeMemberOveragePolicy(memberOveragePolicy)
	if _, err := r.EnsureTenantBillingPolicy(
		ctx,
		tenantID,
		defaultMemberMonthlyLimitPointMicros,
		actorUserID,
	); err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).
		Model(&types.TenantBillingPolicy{}).
		Where("tenant_id = ?", tenantID).
		Updates(map[string]any{
			"default_member_monthly_limit_point_micros": defaultMemberMonthlyLimitPointMicros,
			"member_overage_policy":                     memberOveragePolicy,
			"updated_by_user_id":                        strings.TrimSpace(actorUserID),
			"updated_at":                                time.Now().UTC(),
		}).Error; err != nil {
		return nil, err
	}
	var persisted types.TenantBillingPolicy
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&persisted).Error; err != nil {
		return nil, err
	}
	return &persisted, nil
}

func (r *billingRepository) EnsureCurrentMemberAllocation(
	ctx context.Context,
	tenantID uint64,
	userID string,
	periodStart time.Time,
	periodEnd time.Time,
	defaultBalancePointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	userID = strings.TrimSpace(userID)
	actorUserID = strings.TrimSpace(actorUserID)
	if tenantID == 0 || userID == "" {
		return nil, errors.New("billing: tenant_id and user_id are required for member allocation")
	}
	periodStart = periodStart.UTC()
	periodEnd = periodEnd.UTC()
	if periodStart.IsZero() || !periodEnd.After(periodStart) {
		return nil, errors.New("billing: valid allocation period is required")
	}
	if defaultBalancePointMicros < 0 {
		defaultBalancePointMicros = 0
	}

	limitMode := types.MemberLimitModeInherit
	monthlyLimitPointMicros := int64(0)
	overagePolicy := types.MemberOveragePolicyInherit
	legacyPeriodPointMicros := int64(0)
	legacyBalancePointMicros := int64(0)
	var previous types.TenantMemberCreditAllocation
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("period_start_at DESC").
		First(&previous).Error
	if err == nil {
		normalizeStoredMemberAllocation(&previous)
		limitMode = previous.LimitMode
		monthlyLimitPointMicros = previous.MonthlyLimitPointMicros
		overagePolicy = previous.OveragePolicy
		if limitMode == types.MemberLimitModeCustom {
			legacyPeriodPointMicros = monthlyLimitPointMicros
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	snapshot, err := json.Marshal(map[string]any{
		"source": "auto_default",
		"default_member_monthly_limit_point_micros": defaultBalancePointMicros,
		"limit_mode":                       limitMode,
		"monthly_limit_point_micros":       monthlyLimitPointMicros,
		"default_member_policy_created_at": time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	allocation := &types.TenantMemberCreditAllocation{
		ID:                          uuid.NewString(),
		TenantID:                    tenantID,
		UserID:                      userID,
		PeriodStartAt:               periodStart,
		PeriodEndAt:                 periodEnd,
		AllocatedPeriodPointMicros:  legacyPeriodPointMicros,
		AllocatedBalancePointMicros: legacyBalancePointMicros,
		LimitMode:                   limitMode,
		MonthlyLimitPointMicros:     monthlyLimitPointMicros,
		OveragePolicy:               overagePolicy,
		Status:                      types.BillingStatusActive,
		CreatedByUserID:             actorUserID,
		UpdatedByUserID:             actorUserID,
		SnapshotJSON:                types.JSON(snapshot),
	}
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "user_id"},
			{Name: "period_start_at"},
			{Name: "period_end_at"},
		},
		DoNothing: true,
	}).Create(allocation).Error
	if err != nil {
		return nil, err
	}

	var persisted types.TenantMemberCreditAllocation
	if err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND user_id = ? AND period_start_at = ? AND period_end_at = ?",
			tenantID,
			userID,
			periodStart,
			periodEnd,
		).
		First(&persisted).Error; err != nil {
		return nil, err
	}
	normalizeStoredMemberAllocation(&persisted)
	return &persisted, nil
}

func (r *billingRepository) SetCurrentMemberAllocation(
	ctx context.Context,
	tenantID uint64,
	userID string,
	periodStart time.Time,
	periodEnd time.Time,
	allocatedPeriodPointMicros int64,
	allocatedBalancePointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	userID = strings.TrimSpace(userID)
	actorUserID = strings.TrimSpace(actorUserID)
	if tenantID == 0 || userID == "" {
		return nil, errors.New("billing: tenant_id and user_id are required for member allocation")
	}
	periodStart = periodStart.UTC()
	periodEnd = periodEnd.UTC()
	if periodStart.IsZero() || !periodEnd.After(periodStart) {
		return nil, errors.New("billing: valid allocation period is required")
	}
	if allocatedPeriodPointMicros < 0 || allocatedBalancePointMicros < 0 {
		return nil, errors.New("billing: member allocation must be non-negative")
	}
	snapshot, err := json.Marshal(map[string]any{
		"source":                         "manual",
		"allocated_period_point_micros":  allocatedPeriodPointMicros,
		"allocated_balance_point_micros": allocatedBalancePointMicros,
		"updated_by_user_id":             actorUserID,
		"updated_at":                     time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	allocation := &types.TenantMemberCreditAllocation{
		ID:                          uuid.NewString(),
		TenantID:                    tenantID,
		UserID:                      userID,
		PeriodStartAt:               periodStart,
		PeriodEndAt:                 periodEnd,
		AllocatedPeriodPointMicros:  allocatedPeriodPointMicros,
		AllocatedBalancePointMicros: allocatedBalancePointMicros,
		LimitMode:                   types.MemberLimitModeCustom,
		MonthlyLimitPointMicros:     allocatedPeriodPointMicros + allocatedBalancePointMicros,
		OveragePolicy:               types.MemberOveragePolicyInherit,
		Status:                      types.BillingStatusActive,
		CreatedByUserID:             actorUserID,
		UpdatedByUserID:             actorUserID,
		SnapshotJSON:                types.JSON(snapshot),
	}
	err = r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "user_id"},
			{Name: "period_start_at"},
			{Name: "period_end_at"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"allocated_period_point_micros":  allocatedPeriodPointMicros,
			"allocated_balance_point_micros": allocatedBalancePointMicros,
			"limit_mode":                     types.MemberLimitModeCustom,
			"monthly_limit_point_micros":     allocatedPeriodPointMicros + allocatedBalancePointMicros,
			"overage_policy":                 types.MemberOveragePolicyInherit,
			"status":                         types.BillingStatusActive,
			"updated_by_user_id":             actorUserID,
			"snapshot_json":                  types.JSON(snapshot),
			"updated_at":                     time.Now().UTC(),
		}),
	}).Create(allocation).Error
	if err != nil {
		return nil, err
	}
	var persisted types.TenantMemberCreditAllocation
	if err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND user_id = ? AND period_start_at = ? AND period_end_at = ?",
			tenantID,
			userID,
			periodStart,
			periodEnd,
		).
		First(&persisted).Error; err != nil {
		return nil, err
	}
	normalizeStoredMemberAllocation(&persisted)
	return &persisted, nil
}

func (r *billingRepository) SetCurrentMemberPolicy(
	ctx context.Context,
	tenantID uint64,
	userID string,
	periodStart time.Time,
	periodEnd time.Time,
	limitMode string,
	monthlyLimitPointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	var persisted *types.TenantMemberCreditAllocation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		persisted, err = setCurrentMemberPolicyWithDB(
			tx,
			tenantID,
			userID,
			periodStart,
			periodEnd,
			limitMode,
			monthlyLimitPointMicros,
			actorUserID,
		)
		return err
	})
	return persisted, err
}

func (r *billingRepository) SetCurrentMemberPolicies(
	ctx context.Context,
	tenantID uint64,
	userIDs []string,
	periodStart time.Time,
	periodEnd time.Time,
	limitMode string,
	monthlyLimitPointMicros int64,
	actorUserID string,
) ([]*types.TenantMemberCreditAllocation, error) {
	if len(userIDs) == 0 {
		return nil, errors.New("billing: at least one user_id is required")
	}
	if len(userIDs) > 100 {
		return nil, errors.New("billing: at most 100 member policies can be updated at once")
	}
	seen := make(map[string]struct{}, len(userIDs))
	normalizedUserIDs := make([]string, 0, len(userIDs))
	for _, raw := range userIDs {
		userID := strings.TrimSpace(raw)
		if userID == "" {
			return nil, errors.New("billing: user_id must not be empty")
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		normalizedUserIDs = append(normalizedUserIDs, userID)
	}
	rows := make([]*types.TenantMemberCreditAllocation, 0, len(normalizedUserIDs))
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, userID := range normalizedUserIDs {
			row, err := setCurrentMemberPolicyWithDB(
				tx,
				tenantID,
				userID,
				periodStart,
				periodEnd,
				limitMode,
				monthlyLimitPointMicros,
				actorUserID,
			)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
		return nil
	})
	return rows, err
}

func setCurrentMemberPolicyWithDB(
	db *gorm.DB,
	tenantID uint64,
	userID string,
	periodStart time.Time,
	periodEnd time.Time,
	limitMode string,
	monthlyLimitPointMicros int64,
	actorUserID string,
) (*types.TenantMemberCreditAllocation, error) {
	userID = strings.TrimSpace(userID)
	actorUserID = strings.TrimSpace(actorUserID)
	if tenantID == 0 || userID == "" {
		return nil, errors.New("billing: tenant_id and user_id are required for member policy")
	}
	periodStart = periodStart.UTC()
	periodEnd = periodEnd.UTC()
	if periodStart.IsZero() || !periodEnd.After(periodStart) {
		return nil, errors.New("billing: valid member policy period is required")
	}
	limitMode = normalizeMemberLimitMode(limitMode)
	if monthlyLimitPointMicros < 0 {
		return nil, errors.New("billing: member monthly limit must be non-negative")
	}
	if limitMode != types.MemberLimitModeCustom {
		monthlyLimitPointMicros = 0
	}
	legacyPeriodPointMicros := int64(0)
	if limitMode == types.MemberLimitModeCustom {
		legacyPeriodPointMicros = monthlyLimitPointMicros
	}
	snapshot, err := json.Marshal(map[string]any{
		"source":                     "manual_policy",
		"limit_mode":                 limitMode,
		"monthly_limit_point_micros": monthlyLimitPointMicros,
		"overage_policy":             types.MemberOveragePolicyInherit,
		"updated_by_user_id":         actorUserID,
		"updated_at":                 time.Now().UTC(),
	})
	if err != nil {
		return nil, err
	}
	allocation := &types.TenantMemberCreditAllocation{
		ID:                          uuid.NewString(),
		TenantID:                    tenantID,
		UserID:                      userID,
		PeriodStartAt:               periodStart,
		PeriodEndAt:                 periodEnd,
		AllocatedPeriodPointMicros:  legacyPeriodPointMicros,
		AllocatedBalancePointMicros: 0,
		LimitMode:                   limitMode,
		MonthlyLimitPointMicros:     monthlyLimitPointMicros,
		OveragePolicy:               types.MemberOveragePolicyInherit,
		Status:                      types.BillingStatusActive,
		CreatedByUserID:             actorUserID,
		UpdatedByUserID:             actorUserID,
		SnapshotJSON:                types.JSON(snapshot),
	}
	if err := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "tenant_id"},
			{Name: "user_id"},
			{Name: "period_start_at"},
			{Name: "period_end_at"},
		},
		DoUpdates: clause.Assignments(map[string]any{
			"allocated_period_point_micros":  legacyPeriodPointMicros,
			"allocated_balance_point_micros": 0,
			"limit_mode":                     limitMode,
			"monthly_limit_point_micros":     monthlyLimitPointMicros,
			"overage_policy":                 types.MemberOveragePolicyInherit,
			"status":                         types.BillingStatusActive,
			"updated_by_user_id":             actorUserID,
			"snapshot_json":                  types.JSON(snapshot),
			"updated_at":                     time.Now().UTC(),
		}),
	}).Create(allocation).Error; err != nil {
		return nil, err
	}
	var persisted types.TenantMemberCreditAllocation
	if err := db.
		Where(
			"tenant_id = ? AND user_id = ? AND period_start_at = ? AND period_end_at = ?",
			tenantID,
			userID,
			periodStart,
			periodEnd,
		).
		First(&persisted).Error; err != nil {
		return nil, err
	}
	normalizeStoredMemberAllocation(&persisted)
	return &persisted, nil
}

func billingPeriod(subscription *types.TenantSubscription, now time.Time) (time.Time, time.Time) {
	if subscription != nil && subscription.CurrentPeriodStart != nil && subscription.CurrentPeriodEnd != nil &&
		subscription.CurrentPeriodEnd.After(*subscription.CurrentPeriodStart) {
		return subscription.CurrentPeriodStart.UTC(), subscription.CurrentPeriodEnd.UTC()
	}
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 1, 0)
}

func isEnterpriseUsageScope(scope string) bool {
	return scope == types.BillingUsageScopeEnterprise ||
		scope == types.BillingUsageScopeEnterpriseLegacy
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

		enterpriseUsage := isEnterpriseUsageScope(handle.UsageScope)
		memberPeriodAvailable := int64(^uint64(0) >> 1)
		if enterpriseUsage && handle.MemberLimitMode != types.MemberLimitModeUnlimited {
			var usedByMember int64
			if err := tx.Model(&types.TenantUsageLedger{}).
				Where(
					"tenant_id = ? AND actor_user_id = ? AND status = ? AND billing_at >= ? AND billing_at < ? AND usage_scope IN ?",
					handle.Request.TenantID,
					handle.Request.ActorUserID,
					"settled",
					periodStart,
					periodEnd,
					[]string{types.BillingUsageScopeEnterprise, types.BillingUsageScopeEnterpriseLegacy},
				).
				Select("COALESCE(SUM(billed_point_micros), 0)").
				Scan(&usedByMember).Error; err != nil {
				return err
			}
			var reservedByMember int64
			if err := tx.Model(&types.TenantUsageReservation{}).
				Where(
					"tenant_id = ? AND actor_user_id = ? AND status = ? AND expires_at > ? AND usage_scope IN ?",
					handle.Request.TenantID,
					handle.Request.ActorUserID,
					"active",
					now,
					[]string{types.BillingUsageScopeEnterprise, types.BillingUsageScopeEnterpriseLegacy},
				).
				Select("COALESCE(SUM(estimated_billed_point_micros), 0)").
				Scan(&reservedByMember).Error; err != nil {
				return err
			}
			memberPeriodAvailable =
				handle.MemberMonthlyLimitPointMicros - usedByMember - reservedByMember
			if memberPeriodAvailable < 0 {
				memberPeriodAvailable = 0
			}
		}
		availablePeriod = min(availablePeriod, memberPeriodAvailable)
		periodReserve := min(estimatedBilledPointMicros, availablePeriod)
		balanceReserve := estimatedBilledPointMicros - periodReserve
		if enterpriseUsage &&
			balanceReserve > 0 &&
			handle.MemberOveragePolicy != types.MemberOveragePolicyUseEnterpriseBalance {
			return ErrInsufficientBillingCredits
		}
		if balanceReserve > availableBalance {
			return ErrInsufficientBillingCredits
		}
		periodStartCopy, periodEndCopy := periodStart, periodEnd
		result = &types.TenantUsageReservation{
			ID:                         uuid.NewString(),
			TenantID:                   handle.Request.TenantID,
			ActorUserID:                handle.Request.ActorUserID,
			UsageScope:                 handle.UsageScope,
			AllocationID:               handle.AllocationID,
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
		if handle.ServicePrice != nil {
			result.ServicePricingID = handle.ServicePrice.ID
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

		enterpriseUsage := isEnterpriseUsageScope(handle.UsageScope)
		memberPeriodAvailable := int64(^uint64(0) >> 1)
		if enterpriseUsage && handle.MemberLimitMode != types.MemberLimitModeUnlimited {
			var usedByMember int64
			if err := tx.Model(&types.TenantUsageLedger{}).
				Where(
					"tenant_id = ? AND actor_user_id = ? AND status = ? AND billing_at >= ? AND billing_at < ? AND usage_scope IN ?",
					ledger.TenantID,
					ledger.ActorUserID,
					"settled",
					periodStart,
					periodEnd,
					[]string{types.BillingUsageScopeEnterprise, types.BillingUsageScopeEnterpriseLegacy},
				).
				Select("COALESCE(SUM(billed_point_micros), 0)").
				Scan(&usedByMember).Error; err != nil {
				return err
			}
			var otherReservedByMember int64
			if err := tx.Model(&types.TenantUsageReservation{}).
				Where(
					"tenant_id = ? AND actor_user_id = ? AND ref_no <> ? AND status = ? AND expires_at > ? AND usage_scope IN ?",
					ledger.TenantID,
					ledger.ActorUserID,
					ledger.RefNo,
					"active",
					time.Now().UTC(),
					[]string{types.BillingUsageScopeEnterprise, types.BillingUsageScopeEnterpriseLegacy},
				).
				Select("COALESCE(SUM(estimated_billed_point_micros), 0)").
				Scan(&otherReservedByMember).Error; err != nil {
				return err
			}
			memberPeriodAvailable =
				handle.MemberMonthlyLimitPointMicros - usedByMember - otherReservedByMember
			if memberPeriodAvailable < 0 {
				memberPeriodAvailable = 0
			}
		}
		availablePeriod = min(availablePeriod, memberPeriodAvailable)
		periodCharge := min(ledger.BilledPointMicros, availablePeriod)
		balanceCharge := ledger.BilledPointMicros - periodCharge
		overageBlocked := enterpriseUsage &&
			balanceCharge > 0 &&
			handle.MemberOveragePolicy != types.MemberOveragePolicyUseEnterpriseBalance
		if overageBlocked || balanceCharge > availableBalance {
			now := time.Now().UTC()
			ledger.Status = "reconciliation"
			if overageBlocked {
				ledger.FailureCode = "enterprise_overage_blocked"
			} else {
				ledger.FailureCode = "actual_cost_exceeds_available_credits"
			}
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
	return r.listUsageLedgers(ctx, tenantID, "", limit)
}

func (r *billingRepository) ListUsageLedgersByActor(
	ctx context.Context,
	tenantID uint64,
	actorUserID string,
	limit int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return r.listUsageLedgers(ctx, tenantID, strings.TrimSpace(actorUserID), limit)
}

func (r *billingRepository) listUsageLedgers(
	ctx context.Context,
	tenantID uint64,
	actorUserID string,
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
	if actorUserID != "" {
		query = query.Where("tenant_usage_ledgers.actor_user_id = ?", actorUserID)
	}
	var rows []*types.BillingUsageLedgerSummary
	err := query.
		Order("tenant_usage_ledgers.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

func (r *billingRepository) GetPeriodUsedPointMicros(
	ctx context.Context,
	tenantID uint64,
	periodStart time.Time,
	periodEnd time.Time,
) (int64, error) {
	if tenantID == 0 || periodEnd.IsZero() || !periodEnd.After(periodStart) {
		return 0, nil
	}
	var used int64
	err := r.db.WithContext(ctx).
		Model(&types.TenantUsageLedger{}).
		Where(
			"tenant_id = ? AND status = ? AND billing_at >= ? AND billing_at < ?",
			tenantID,
			"settled",
			periodStart.UTC(),
			periodEnd.UTC(),
		).
		Select("COALESCE(SUM(period_covered_point_micros), 0)").
		Scan(&used).Error
	if err != nil {
		return 0, err
	}
	if used < 0 {
		return 0, nil
	}
	return used, nil
}

func (r *billingRepository) ListUsageSummaryByActor(
	ctx context.Context,
	actorUserIDs []string,
) ([]*types.BillingActorUsageSummary, error) {
	if len(actorUserIDs) == 0 {
		return []*types.BillingActorUsageSummary{}, nil
	}
	seen := make(map[string]struct{}, len(actorUserIDs))
	ids := make([]string, 0, len(actorUserIDs))
	for _, raw := range actorUserIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return []*types.BillingActorUsageSummary{}, nil
	}

	type actorUsageRow struct {
		ActorUserID                 string
		LedgerCount                 int64
		PersonalLedgerCount         int64
		EnterpriseLedgerCount       int64
		InputTokens                 int64
		CachedTokens                int64
		OutputTokens                int64
		ReasoningTokens             int64
		BilledPointMicros           int64
		PersonalBilledPointMicros   int64
		EnterpriseBilledPointMicros int64
		LastBillingAt               aggregateTime `gorm:"column:last_billing_at"`
	}
	var aggregateRows []actorUsageRow
	err := r.db.WithContext(ctx).
		Table("tenant_usage_ledgers").
		Select(`
			actor_user_id,
			COUNT(*) AS ledger_count,
			COALESCE(SUM(CASE WHEN usage_scope = 'personal_usage' THEN 1 ELSE 0 END), 0) AS personal_ledger_count,
			COALESCE(SUM(CASE WHEN usage_scope IN ('enterprise_usage', 'enterprise_allocated_usage') THEN 1 ELSE 0 END), 0) AS enterprise_ledger_count,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COALESCE(SUM(billed_point_micros), 0) AS billed_point_micros,
			COALESCE(SUM(CASE WHEN usage_scope = 'personal_usage' THEN billed_point_micros ELSE 0 END), 0) AS personal_billed_point_micros,
			COALESCE(SUM(CASE WHEN usage_scope IN ('enterprise_usage', 'enterprise_allocated_usage') THEN billed_point_micros ELSE 0 END), 0) AS enterprise_billed_point_micros,
			MAX(billing_at) AS last_billing_at
		`).
		Where("actor_user_id IN ?", ids).
		Group("actor_user_id").
		Scan(&aggregateRows).Error
	if err != nil {
		return nil, err
	}
	rows := make([]*types.BillingActorUsageSummary, 0, len(aggregateRows))
	for _, row := range aggregateRows {
		summary := &types.BillingActorUsageSummary{
			ActorUserID:                 row.ActorUserID,
			LedgerCount:                 row.LedgerCount,
			PersonalLedgerCount:         row.PersonalLedgerCount,
			EnterpriseLedgerCount:       row.EnterpriseLedgerCount,
			InputTokens:                 row.InputTokens,
			CachedTokens:                row.CachedTokens,
			OutputTokens:                row.OutputTokens,
			ReasoningTokens:             row.ReasoningTokens,
			BilledPointMicros:           row.BilledPointMicros,
			PersonalBilledPointMicros:   row.PersonalBilledPointMicros,
			EnterpriseBilledPointMicros: row.EnterpriseBilledPointMicros,
		}
		if row.LastBillingAt.Valid {
			billingAt := row.LastBillingAt.Time
			summary.LastBillingAt = &billingAt
		}
		rows = append(rows, summary)
	}
	return rows, err
}

func (r *billingRepository) ListMemberAllocations(
	ctx context.Context,
	tenantID uint64,
	at time.Time,
) ([]*types.TenantMemberCreditAllocationSummary, error) {
	if tenantID == 0 {
		return []*types.TenantMemberCreditAllocationSummary{}, nil
	}
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	var enterprisePolicy types.TenantBillingPolicy
	policyErr := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		First(&enterprisePolicy).Error
	if policyErr != nil && !errors.Is(policyErr, gorm.ErrRecordNotFound) {
		return nil, policyErr
	}
	if errors.Is(policyErr, gorm.ErrRecordNotFound) {
		enterprisePolicy.MemberOveragePolicy = types.MemberOveragePolicyBlock
	}
	enterprisePolicy.MemberOveragePolicy =
		normalizeMemberOveragePolicy(enterprisePolicy.MemberOveragePolicy)
	var allocations []*types.TenantMemberCreditAllocation
	if err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND status = ? AND period_start_at <= ? AND period_end_at > ?",
			tenantID,
			types.BillingStatusActive,
			at,
			at,
		).
		Order("user_id ASC, period_start_at DESC").
		Find(&allocations).Error; err != nil {
		return nil, err
	}
	if len(allocations) == 0 {
		return []*types.TenantMemberCreditAllocationSummary{}, nil
	}
	allocationIDs := make([]string, 0, len(allocations))
	for _, allocation := range allocations {
		if allocation != nil {
			allocationIDs = append(allocationIDs, allocation.ID)
		}
	}

	type usageRow struct {
		AllocationID    string
		UsedPointMicros int64
		InputTokens     int64
		OutputTokens    int64
		ReasoningTokens int64
		LedgerCount     int64
		LastBillingAt   aggregateTime `gorm:"column:last_billing_at"`
	}
	var usageRows []usageRow
	if err := r.db.WithContext(ctx).
		Table("tenant_usage_ledgers").
		Select(`
			allocation_id,
			COALESCE(SUM(billed_point_micros), 0) AS used_point_micros,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(reasoning_tokens), 0) AS reasoning_tokens,
			COUNT(*) AS ledger_count,
			MAX(billing_at) AS last_billing_at
		`).
		Where("tenant_id = ? AND allocation_id IN ?", tenantID, allocationIDs).
		Group("allocation_id").
		Scan(&usageRows).Error; err != nil {
		return nil, err
	}
	usageByAllocationID := make(map[string]usageRow, len(usageRows))
	for _, row := range usageRows {
		usageByAllocationID[row.AllocationID] = row
	}

	summaries := make([]*types.TenantMemberCreditAllocationSummary, 0, len(allocations))
	for _, allocation := range allocations {
		if allocation == nil {
			continue
		}
		normalizeStoredMemberAllocation(allocation)
		summary := &types.TenantMemberCreditAllocationSummary{
			TenantMemberCreditAllocation: *allocation,
			EffectiveOveragePolicy:       enterprisePolicy.MemberOveragePolicy,
		}
		switch allocation.LimitMode {
		case types.MemberLimitModeCustom:
			summary.EffectiveMonthlyLimitPointMicros = allocation.MonthlyLimitPointMicros
		case types.MemberLimitModeUnlimited:
			summary.EffectiveMonthlyLimitPointMicros = 0
		default:
			summary.EffectiveMonthlyLimitPointMicros =
				enterprisePolicy.DefaultMemberMonthlyLimitPointMicros
		}
		if row, ok := usageByAllocationID[allocation.ID]; ok {
			summary.UsedPointMicros = row.UsedPointMicros
			summary.InputTokens = row.InputTokens
			summary.OutputTokens = row.OutputTokens
			summary.ReasoningTokens = row.ReasoningTokens
			summary.LedgerCount = row.LedgerCount
			if row.LastBillingAt.Valid {
				billingAt := row.LastBillingAt.Time
				summary.LastBillingAt = &billingAt
			}
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}
