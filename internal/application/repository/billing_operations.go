package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/Tencent/WeKnora/internal/types"
)

const (
	manualBillingProvider      = "manual"
	manualBillingPaymentMethod = "system_admin"
)

var currentSubscriptionStatuses = []string{
	types.BillingStatusActive,
	types.BillingStatusTrialing,
	types.BillingStatusLegacy,
}

func normalizePurchaseItemInput(
	input types.BillingPurchaseItemInput,
) (types.BillingPurchaseItemInput, error) {
	input.Code = strings.TrimSpace(input.Code)
	input.ItemType = strings.ToLower(strings.TrimSpace(input.ItemType))
	input.EditionScope = strings.ToLower(strings.TrimSpace(input.EditionScope))
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))

	if input.Code == "" || len(input.Code) > 64 {
		return input, errors.New("billing: purchase item code is required and must be at most 64 characters")
	}
	if input.Name == "" || len(input.Name) > 128 {
		return input, errors.New("billing: purchase item name is required and must be at most 128 characters")
	}
	if len(input.Description) > 512 {
		return input, errors.New("billing: purchase item description must be at most 512 characters")
	}
	switch input.ItemType {
	case types.BillingPurchaseItemTypeTopup, types.BillingPurchaseItemTypeStorageAddon:
	default:
		return input, fmt.Errorf("billing: unsupported purchase item type %q", input.ItemType)
	}
	if input.EditionScope == "" {
		input.EditionScope = types.BillingEditionScopeAll
	}
	switch input.EditionScope {
	case types.BillingEditionScopeAll, types.BillingEditionScopePersonal, types.BillingEditionScopeEnterprise:
	default:
		return input, fmt.Errorf("billing: unsupported edition scope %q", input.EditionScope)
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.Currency != "CNY" {
		return input, errors.New("billing: only CNY purchase items are supported")
	}
	if input.AmountCents < 0 {
		return input, errors.New("billing: purchase item amount_cents must be non-negative")
	}
	if input.DurationDays < 0 {
		return input, errors.New("billing: purchase item duration_days must be non-negative")
	}
	// The offline operations flow has no automatic expiry sweeper. Do not
	// allow operators to configure a duration that would silently leave quota
	// available after the entitlement expires.
	if input.DurationDays != 0 {
		return input, errors.New("billing: time-limited purchase items are not available while online payments are disabled")
	}
	if input.Status == "" {
		input.Status = types.BillingPurchaseItemStatusActive
	}
	if input.Status != types.BillingPurchaseItemStatusActive &&
		input.Status != types.BillingPurchaseItemStatusDisabled {
		return input, fmt.Errorf("billing: unsupported purchase item status %q", input.Status)
	}
	if len(input.MetadataJSON) == 0 {
		input.MetadataJSON = types.JSON([]byte("{}"))
	}
	if !json.Valid(input.MetadataJSON) {
		return input, errors.New("billing: purchase item metadata_json must be valid JSON")
	}

	switch input.ItemType {
	case types.BillingPurchaseItemTypeTopup:
		if input.CreditPointMicros <= 0 || input.StorageQuotaBytes != 0 {
			return input, errors.New("billing: a topup item requires positive credits and zero storage")
		}
	case types.BillingPurchaseItemTypeStorageAddon:
		if input.StorageQuotaBytes <= 0 || input.CreditPointMicros != 0 {
			return input, errors.New("billing: a storage addon requires positive storage and zero credits")
		}
	}
	return input, nil
}

func purchaseItemFromInput(
	id string,
	input types.BillingPurchaseItemInput,
) *types.BillingPurchaseItem {
	return &types.BillingPurchaseItem{
		ID:                id,
		Code:              input.Code,
		ItemType:          input.ItemType,
		EditionScope:      input.EditionScope,
		Name:              input.Name,
		Description:       input.Description,
		Currency:          input.Currency,
		AmountCents:       input.AmountCents,
		CreditPointMicros: input.CreditPointMicros,
		StorageQuotaBytes: input.StorageQuotaBytes,
		DurationDays:      input.DurationDays,
		Status:            input.Status,
		SortOrder:         input.SortOrder,
		MetadataJSON:      input.MetadataJSON,
	}
}

func (r *billingRepository) ListPurchaseItems(
	ctx context.Context,
) ([]*types.BillingPurchaseItem, error) {
	var rows []*types.BillingPurchaseItem
	err := r.db.WithContext(ctx).
		Order("sort_order ASC, created_at ASC, code ASC").
		Find(&rows).Error
	return rows, err
}

func (r *billingRepository) CreatePurchaseItem(
	ctx context.Context,
	input types.BillingPurchaseItemInput,
) (*types.BillingPurchaseItem, error) {
	input, err := normalizePurchaseItemInput(input)
	if err != nil {
		return nil, err
	}
	row := purchaseItemFromInput(uuid.NewString(), input)
	if err := r.db.WithContext(ctx).Create(row).Error; err != nil {
		return nil, err
	}
	return row, nil
}

func (r *billingRepository) UpdatePurchaseItem(
	ctx context.Context,
	id string,
	input types.BillingPurchaseItemInput,
) (*types.BillingPurchaseItem, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, errors.New("billing: purchase item ID is required")
	}
	input, err := normalizePurchaseItemInput(input)
	if err != nil {
		return nil, err
	}
	var result types.BillingPurchaseItem
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.BillingPurchaseItem
		if err := tx.Where("id = ?", id).First(&existing).Error; err != nil {
			return err
		}
		updates := purchaseItemFromInput(existing.ID, input)
		if err := tx.Model(&types.BillingPurchaseItem{}).
			Where("id = ?", id).
			Updates(map[string]any{
				"code":                updates.Code,
				"item_type":           updates.ItemType,
				"edition_scope":       updates.EditionScope,
				"name":                updates.Name,
				"description":         updates.Description,
				"currency":            updates.Currency,
				"amount_cents":        updates.AmountCents,
				"credit_point_micros": updates.CreditPointMicros,
				"storage_quota_bytes": updates.StorageQuotaBytes,
				"duration_days":       updates.DurationDays,
				"status":              updates.Status,
				"sort_order":          updates.SortOrder,
				"metadata_json":       updates.MetadataJSON,
				"updated_at":          time.Now().UTC(),
			}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).First(&result).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *billingRepository) ListPaymentOrders(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.BillingPaymentOrderSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).
		Table("billing_payment_orders").
		Select(`
			billing_payment_orders.*,
			tenants.name AS tenant_name,
			COALESCE(billing_plans.name, '') AS plan_name,
			COALESCE(billing_purchase_items.name, '') AS item_name
		`).
		Joins("JOIN tenants ON tenants.id = billing_payment_orders.tenant_id AND tenants.deleted_at IS NULL").
		Joins("LEFT JOIN billing_plans ON billing_plans.id = billing_payment_orders.plan_id").
		Joins("LEFT JOIN billing_purchase_items ON billing_purchase_items.id = billing_payment_orders.item_id")
	if tenantID > 0 {
		query = query.Where("billing_payment_orders.tenant_id = ?", tenantID)
	}
	var rows []*types.BillingPaymentOrderSummary
	err := query.Order("billing_payment_orders.created_at DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

func (r *billingRepository) GetPaymentOrder(
	ctx context.Context,
	tenantID uint64,
	orderNo string,
) (*types.BillingPaymentOrderSummary, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, errors.New("billing: order_no is required")
	}
	query := r.db.WithContext(ctx).
		Table("billing_payment_orders").
		Select(`
			billing_payment_orders.*,
			tenants.name AS tenant_name,
			COALESCE(billing_plans.name, '') AS plan_name,
			COALESCE(billing_purchase_items.name, '') AS item_name
		`).
		Joins("JOIN tenants ON tenants.id = billing_payment_orders.tenant_id AND tenants.deleted_at IS NULL").
		Joins("LEFT JOIN billing_plans ON billing_plans.id = billing_payment_orders.plan_id").
		Joins("LEFT JOIN billing_purchase_items ON billing_purchase_items.id = billing_payment_orders.item_id").
		Where("billing_payment_orders.order_no = ?", orderNo)
	if tenantID > 0 {
		query = query.Where("billing_payment_orders.tenant_id = ?", tenantID)
	}
	var row types.BillingPaymentOrderSummary
	if err := query.First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func validateManualOrderType(orderType string) (string, error) {
	orderType = strings.ToLower(strings.TrimSpace(orderType))
	switch orderType {
	case types.BillingOrderTypeTopup,
		types.BillingOrderTypeStorageAddon,
		types.BillingOrderTypeManualContract:
		return orderType, nil
	default:
		return "", fmt.Errorf("billing: unsupported manual order type %q", orderType)
	}
}

func validateItemScope(tenant *types.Tenant, scope string) error {
	if scope == types.BillingEditionScopeAll {
		return nil
	}
	if tenant == nil || tenant.SpaceType == nil {
		return errors.New("billing: purchase item cannot be applied to an unclassified workspace")
	}
	switch scope {
	case types.BillingEditionScopePersonal:
		if *tenant.SpaceType == types.SpaceTypePersonal {
			return nil
		}
	case types.BillingEditionScopeEnterprise:
		if *tenant.SpaceType == types.SpaceTypeOrganization {
			return nil
		}
	}
	return errors.New("billing: purchase item is not available for this workspace type")
}

func manualBillingPeriod(
	billingInterval string,
	periodDays int,
	now time.Time,
) (string, time.Time, time.Time, string, error) {
	billingInterval = strings.ToLower(strings.TrimSpace(billingInterval))
	if billingInterval == "" {
		billingInterval = "manual"
	}
	switch billingInterval {
	case "trial", "month", "year", "manual", "none":
	default:
		return "", time.Time{}, time.Time{}, "", fmt.Errorf("billing: unsupported billing interval %q", billingInterval)
	}
	if periodDays <= 0 {
		switch billingInterval {
		case "trial":
			periodDays = 14
		case "month":
			periodDays = 31
		case "year":
			periodDays = 366
		default:
			periodDays = 365
		}
	}
	if periodDays > 3660 {
		return "", time.Time{}, time.Time{}, "", errors.New("billing: manual contract period_days must be at most 3660")
	}
	status := types.BillingStatusActive
	if billingInterval == "trial" {
		status = types.BillingStatusTrialing
	}
	return billingInterval, now, now.AddDate(0, 0, periodDays), status, nil
}

func (r *billingRepository) CreateManualPaymentOrder(
	ctx context.Context,
	input types.BillingManualOrderInput,
) (*types.BillingPaymentOrder, error) {
	if input.TenantID == 0 {
		return nil, errors.New("billing: tenant_id is required")
	}
	orderType, err := validateManualOrderType(input.OrderType)
	if err != nil {
		return nil, err
	}
	input.ItemID = strings.TrimSpace(input.ItemID)
	input.PlanID = strings.TrimSpace(input.PlanID)
	input.Description = strings.TrimSpace(input.Description)
	input.ActorUserID = strings.TrimSpace(input.ActorUserID)
	if input.AmountCents < 0 {
		return nil, errors.New("billing: manual order amount_cents must be non-negative")
	}
	if len(input.Description) > 512 {
		return nil, errors.New("billing: manual order description must be at most 512 characters")
	}
	if input.Description == "" {
		input.Description = "SystemAdmin manual billing operation"
	}

	now := time.Now().UTC()
	orderNo := "manual-" + uuid.NewString()
	var created types.BillingPaymentOrder
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant types.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ?", input.TenantID).
			First(&tenant).Error; err != nil {
			return err
		}

		var (
			item         *types.BillingPurchaseItem
			plan         *types.BillingPlan
			creditMicros = input.CreditPointMicros
			storageBytes = input.StorageQuotaBytes
			amountCents  = input.AmountCents
			interval     string
			periodStart  *time.Time
			periodEnd    *time.Time
			status       = types.BillingOrderStatusPaid
		)
		if input.ItemID != "" {
			var found types.BillingPurchaseItem
			if err := tx.Where("id = ?", input.ItemID).First(&found).Error; err != nil {
				return err
			}
			if found.Status != types.BillingPurchaseItemStatusActive {
				return errors.New("billing: purchase item is disabled")
			}
			if found.ItemType != orderType {
				return errors.New("billing: purchase item type does not match manual order type")
			}
			if err := validateItemScope(&tenant, found.EditionScope); err != nil {
				return err
			}
			item = &found
			creditMicros = found.CreditPointMicros
			storageBytes = found.StorageQuotaBytes
			amountCents = found.AmountCents
		}

		switch orderType {
		case types.BillingOrderTypeTopup:
			if creditMicros == 0 || storageBytes != 0 {
				return errors.New("billing: manual topup requires a non-zero credit adjustment and zero storage")
			}
			var account types.TenantCreditAccount
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("tenant_id = ?", tenant.ID).
				First(&account).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				account = types.TenantCreditAccount{
					ID:       uuid.NewString(),
					TenantID: tenant.ID,
				}
				if err := tx.Create(&account).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
			nextBalance := account.BalancePointMicros + creditMicros
			if (creditMicros > 0 && nextBalance < account.BalancePointMicros) || nextBalance < 0 {
				return ErrInsufficientBillingCredits
			}
			if err := tx.Model(&types.TenantCreditAccount{}).
				Where("id = ?", account.ID).
				Updates(map[string]any{
					"balance_point_micros": nextBalance,
					"version":              account.Version + 1,
					"updated_at":           now,
				}).Error; err != nil {
				return err
			}
			if err := tx.Create(&types.TenantCreditTransaction{
				ID:                 uuid.NewString(),
				TenantID:           tenant.ID,
				AccountID:          account.ID,
				Type:               "manual_adjustment",
				AmountPointMicros:  creditMicros,
				BalancePointMicros: nextBalance,
				RefNo:              orderNo,
				Description:        input.Description,
			}).Error; err != nil {
				return err
			}
		case types.BillingOrderTypeStorageAddon:
			if storageBytes == 0 || creditMicros != 0 {
				return errors.New("billing: manual storage adjustment requires non-zero storage and zero credits")
			}
			if (storageBytes > 0 && tenant.StorageQuota > math.MaxInt64-storageBytes) ||
				(storageBytes < 0 && tenant.StorageQuota < math.MinInt64-storageBytes) {
				return errors.New("billing: storage adjustment exceeds supported range")
			}
			nextQuota := tenant.StorageQuota + storageBytes
			if nextQuota <= 0 || nextQuota < tenant.StorageUsed {
				return errors.New("billing: storage quota cannot be lower than current storage used")
			}
			if err := tx.Model(&types.Tenant{}).
				Where("id = ?", tenant.ID).
				Updates(map[string]any{
					"storage_quota": nextQuota,
					"updated_at":    now,
				}).Error; err != nil {
				return err
			}
			metadata, err := json.Marshal(map[string]any{
				"quota_before_bytes": tenant.StorageQuota,
				"quota_after_bytes":  nextQuota,
				"source":             "manual_order",
				"description":        input.Description,
			})
			if err != nil {
				return err
			}
			if err := tx.Create(&types.TenantStorageTransaction{
				ID:                    uuid.NewString(),
				TenantID:              tenant.ID,
				ActorUserID:           input.ActorUserID,
				RefNo:                 orderNo,
				Operation:             "storage_quota_adjustment",
				AmountBytes:           0,
				StorageUsedAfterBytes: tenant.StorageUsed,
				MetadataJSON:          types.JSON(metadata),
			}).Error; err != nil {
				return err
			}
			if storageBytes > 0 {
				if err := tx.Create(&types.TenantStorageAddonGrant{
					ID:                uuid.NewString(),
					TenantID:          tenant.ID,
					OrderNo:           orderNo,
					ItemID:            input.ItemID,
					StorageQuotaBytes: storageBytes,
					Status:            types.BillingStatusActive,
					MetadataJSON:      types.JSON(metadata),
				}).Error; err != nil {
					return err
				}
			}
		case types.BillingOrderTypeManualContract:
			if input.PlanID == "" {
				return errors.New("billing: manual contract requires plan_id")
			}
			if creditMicros != 0 || storageBytes != 0 || input.ItemID != "" {
				return errors.New("billing: manual contract only accepts plan and contract period fields")
			}
			var found types.BillingPlan
			if err := tx.Where("id = ?", input.PlanID).First(&found).Error; err != nil {
				return err
			}
			if found.Status != types.BillingStatusActive {
				return errors.New("billing: target plan is not active")
			}
			if tenant.SpaceType == nil || found.SpaceType != *tenant.SpaceType {
				return errors.New("billing: target plan does not match workspace type")
			}
			plan = &found
			var subscriptionStatus string
			interval, start, end, subscriptionStatus, err := manualBillingPeriod(
				input.BillingInterval,
				input.PeriodDays,
				now,
			)
			if err != nil {
				return err
			}
			periodStart = &start
			periodEnd = &end
			if err := tx.Model(&types.TenantSubscription{}).
				Where("tenant_id = ? AND status IN ?", tenant.ID, currentSubscriptionStatuses).
				Updates(map[string]any{
					"status":     types.BillingStatusCanceled,
					"updated_at": now,
				}).Error; err != nil {
				return err
			}
			snapshot, err := json.Marshal(map[string]any{
				"source":                 "manual_contract",
				"order_no":               orderNo,
				"plan_code":              plan.Code,
				"included_storage_bytes": plan.IncludedStorageBytes,
				"included_point_micros":  plan.IncludedPointMicros,
				"contract_description":   input.Description,
			})
			if err != nil {
				return err
			}
			if err := tx.Create(&types.TenantSubscription{
				ID:                 uuid.NewString(),
				TenantID:           tenant.ID,
				PlanID:             plan.ID,
				Status:             subscriptionStatus,
				BillingInterval:    interval,
				CurrentPeriodStart: periodStart,
				CurrentPeriodEnd:   periodEnd,
				PriceSnapshot:      types.JSON(snapshot),
				Source:             "manual_contract",
			}).Error; err != nil {
				return err
			}
			// A plan guarantees its included storage but must never silently
			// remove manually assigned or purchased capacity.
			if tenant.StorageQuota > 0 && plan.IncludedStorageBytes > tenant.StorageQuota {
				metadata, err := json.Marshal(map[string]any{
					"quota_before_bytes": tenant.StorageQuota,
					"quota_after_bytes":  plan.IncludedStorageBytes,
					"source":             "manual_contract",
					"plan_code":          plan.Code,
				})
				if err != nil {
					return err
				}
				if err := tx.Model(&types.Tenant{}).
					Where("id = ?", tenant.ID).
					Updates(map[string]any{
						"storage_quota": plan.IncludedStorageBytes,
						"updated_at":    now,
					}).Error; err != nil {
					return err
				}
				if err := tx.Create(&types.TenantStorageTransaction{
					ID:                    uuid.NewString(),
					TenantID:              tenant.ID,
					ActorUserID:           input.ActorUserID,
					RefNo:                 orderNo + ":plan-storage",
					Operation:             "contract_storage_quota",
					AmountBytes:           0,
					StorageUsedAfterBytes: tenant.StorageUsed,
					MetadataJSON:          types.JSON(metadata),
				}).Error; err != nil {
					return err
				}
			}
		}

		snapshot, err := json.Marshal(map[string]any{
			"source":      "system_admin_manual",
			"description": input.Description,
			"item_code": func() string {
				if item == nil {
					return ""
				}
				return item.Code
			}(),
			"plan_code": func() string {
				if plan == nil {
					return ""
				}
				return plan.Code
			}(),
		})
		if err != nil {
			return err
		}
		created = types.BillingPaymentOrder{
			ID:                  uuid.NewString(),
			OrderNo:             orderNo,
			OrderType:           orderType,
			TenantID:            tenant.ID,
			ActorUserID:         input.ActorUserID,
			PlanID:              input.PlanID,
			ItemID:              input.ItemID,
			Provider:            manualBillingProvider,
			PaymentMethod:       manualBillingPaymentMethod,
			Status:              status,
			Currency:            "CNY",
			AmountCents:         amountCents,
			CreditPointMicros:   creditMicros,
			StorageQuotaBytes:   storageBytes,
			BillingInterval:     interval,
			Cycles:              1,
			PaidAt:              &now,
			ProviderPayloadJSON: types.JSON([]byte("{}")),
			NotifySnapshotJSON:  types.JSON([]byte("{}")),
			SnapshotJSON:        types.JSON(snapshot),
		}
		return tx.Create(&created).Error
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}
