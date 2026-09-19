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

	"github.com/Tencent/WeKnora/internal/types"
)

func (r *billingRepository) ListServicePrices(
	ctx context.Context,
) ([]*types.BillingServicePrice, error) {
	var rows []*types.BillingServicePrice
	err := r.db.WithContext(ctx).
		Where("service_code = ?", types.BillingServiceCodeMCPToolCall).
		Order("service_code ASC, version DESC").
		Find(&rows).Error
	return rows, err
}

func (r *billingRepository) CreateServicePriceVersion(
	ctx context.Context,
	input types.BillingServicePriceInput,
) (*types.BillingServicePrice, error) {
	serviceCode := strings.TrimSpace(input.ServiceCode)
	if serviceCode == "" {
		return nil, errors.New("billing: service_code is required")
	}
	if serviceCode != types.BillingServiceCodeMCPToolCall {
		return nil, fmt.Errorf("billing: only %q supports service pricing", types.BillingServiceCodeMCPToolCall)
	}
	mode := strings.ToLower(strings.TrimSpace(input.PricingMode))
	if mode == "" {
		mode = "call"
	}
	if mode != "call" {
		return nil, fmt.Errorf("billing: service pricing_mode %q is not supported", mode)
	}
	if input.NanoUSDPerCall < 0 || input.NanoUSDPerUnit < 0 {
		return nil, errors.New("billing: service prices must be non-negative")
	}
	multiplier := input.ServiceMultiplierPPM
	if multiplier <= 0 {
		multiplier = 1_000_000
	}
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = types.BillingStatusActive
	}
	if status != types.BillingStatusActive && status != "disabled" {
		return nil, fmt.Errorf("billing: invalid service price status %q", status)
	}
	effectiveAt := time.Now().UTC()
	if input.EffectiveAt != nil {
		effectiveAt = input.EffectiveAt.UTC()
	}
	if input.ExpiresAt != nil && !input.ExpiresAt.After(effectiveAt) {
		return nil, errors.New("billing: expires_at must be after effective_at")
	}

	var created *types.BillingServicePrice
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxVersion int
		if err := tx.Model(&types.BillingServicePrice{}).
			Where("service_code = ?", serviceCode).
			Select("COALESCE(MAX(version), 0)").
			Scan(&maxVersion).Error; err != nil {
			return err
		}
		snapshot, err := json.Marshal(map[string]any{
			"service_code":           serviceCode,
			"service_name":           strings.TrimSpace(input.ServiceName),
			"pricing_mode":           mode,
			"nanousd_per_call":       input.NanoUSDPerCall,
			"nanousd_per_unit":       input.NanoUSDPerUnit,
			"unit_name":              strings.TrimSpace(input.UnitName),
			"service_multiplier_ppm": multiplier,
			"version":                maxVersion + 1,
			"effective_at":           effectiveAt,
		})
		if err != nil {
			return err
		}
		created = &types.BillingServicePrice{
			ID:                   uuid.NewString(),
			ServiceCode:          serviceCode,
			ServiceName:          strings.TrimSpace(input.ServiceName),
			PricingMode:          mode,
			NanoUSDPerCall:       input.NanoUSDPerCall,
			NanoUSDPerUnit:       input.NanoUSDPerUnit,
			UnitName:             strings.TrimSpace(input.UnitName),
			ServiceMultiplierPPM: multiplier,
			Version:              maxVersion + 1,
			EffectiveAt:          effectiveAt,
			ExpiresAt:            input.ExpiresAt,
			Status:               status,
			SnapshotJSON:         types.JSON(snapshot),
		}
		return tx.Create(created).Error
	})
	return created, err
}

func (r *billingRepository) GetActiveServicePrice(
	ctx context.Context,
	serviceCode string,
) (*types.BillingServicePrice, error) {
	serviceCode = strings.TrimSpace(serviceCode)
	if serviceCode != types.BillingServiceCodeMCPToolCall {
		return nil, nil
	}
	now := time.Now().UTC()
	var row types.BillingServicePrice
	err := r.db.WithContext(ctx).
		Where(
			"service_code = ? AND status = ? AND effective_at <= ? AND (expires_at IS NULL OR expires_at > ?)",
			serviceCode,
			types.BillingStatusActive,
			now,
			now,
		).
		Order("version DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *billingRepository) ListStorageTransactions(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingStorageTransactionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).
		Table("tenant_storage_transactions").
		Select("tenant_storage_transactions.*, tenants.name AS tenant_name").
		Joins("JOIN tenants ON tenants.id = tenant_storage_transactions.tenant_id AND tenants.deleted_at IS NULL")
	if tenantID > 0 {
		query = query.Where("tenant_storage_transactions.tenant_id = ?", tenantID)
	}
	var rows []*types.BillingStorageTransactionSummary
	err := query.
		Order("tenant_storage_transactions.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}
