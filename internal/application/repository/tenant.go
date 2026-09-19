package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrTenantNotFound         = errors.New("tenant not found")
	ErrTenantHasKnowledgeBase = errors.New("tenant has associated knowledge bases")
)

// tenantRepository implements tenant repository interface
type tenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *gorm.DB) interfaces.TenantRepository {
	return &tenantRepository{db: db}
}

// CreateTenant creates tenant
func (r *tenantRepository) CreateTenant(ctx context.Context, tenant *types.Tenant) error {
	return r.db.WithContext(ctx).Create(tenant).Error
}

// GetTenantByID gets tenant by ID
func (r *tenantRepository) GetTenantByID(ctx context.Context, id uint64) (*types.Tenant, error) {
	var tenant types.Tenant
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}
	return &tenant, nil
}

// GetTenantByProvisioningKey returns the tenant previously created for an
// idempotent provisioning command. A missing key is a normal lookup miss and
// returns (nil, nil), allowing the caller to proceed with the create path.
func (r *tenantRepository) GetTenantByProvisioningKey(ctx context.Context, key string) (*types.Tenant, error) {
	var tenant types.Tenant
	if err := r.db.WithContext(ctx).
		Where("provisioning_key = ?", key).
		First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &tenant, nil
}

// GetTenantsByIDs batches GetTenantByID with a single IN-list query.
// Returns a map keyed by tenant ID; missing rows are simply absent from
// the map (no error). An empty input slice short-circuits to an empty map
// without hitting the database.
func (r *tenantRepository) GetTenantsByIDs(ctx context.Context, ids []uint64) (map[uint64]*types.Tenant, error) {
	if len(ids) == 0 {
		return map[uint64]*types.Tenant{}, nil
	}
	var tenants []*types.Tenant
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&tenants).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]*types.Tenant, len(tenants))
	for _, t := range tenants {
		if t != nil {
			out[t.ID] = t
		}
	}
	return out, nil
}

// ListTenants lists all tenants
func (r *tenantRepository) ListTenants(ctx context.Context) ([]*types.Tenant, error) {
	var tenants []*types.Tenant
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&tenants).Error; err != nil {
		return nil, err
	}
	return tenants, nil
}

// SearchTenants searches tenants with pagination and filters
func (r *tenantRepository) SearchTenants(ctx context.Context, keyword string, tenantID uint64, page, pageSize int) ([]*types.Tenant, int64, error) {
	var tenants []*types.Tenant
	var total int64

	query := r.db.WithContext(ctx).Model(&types.Tenant{})

	// Build search conditions
	if tenantID > 0 && keyword != "" {
		escaped := escapeLikeKeyword(keyword)
		query = query.Where("id = ? OR name LIKE ? OR description LIKE ?", tenantID, "%"+escaped+"%", "%"+escaped+"%")
	} else if tenantID > 0 {
		query = query.Where("id = ?", tenantID)
	} else if keyword != "" {
		escaped := escapeLikeKeyword(keyword)
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+escaped+"%", "%"+escaped+"%")
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Apply pagination
	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query = query.Offset(offset).Limit(pageSize)
	}

	// Order by created_at DESC
	query = query.Order("created_at DESC")

	// Execute query
	if err := query.Find(&tenants).Error; err != nil {
		return nil, 0, err
	}

	return tenants, total, nil
}

// UpdateTenant updates tenant.
func (r *tenantRepository) UpdateTenant(ctx context.Context, tenant *types.Tenant) error {
	return r.db.WithContext(ctx).Model(&types.Tenant{}).Where("id = ?", tenant.ID).Updates(tenant).Error
}

// DeleteTenant soft-deletes the tenant and every active membership row
// for that tenant in one transaction. Without the membership purge,
// /auth/me still lists the defunct tenant (name lookup fails → UI shows
// "#<id>").
func (r *tenantRepository) DeleteTenant(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ?", id).Delete(&types.TenantMember{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).Delete(&types.Tenant{}).Error
	})
}

func (r *tenantRepository) AdjustStorageUsed(ctx context.Context, tenantID uint64, delta int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant types.Tenant
		// 使用悲观锁确保并发安全
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&tenant, tenantID).Error; err != nil {
			return err
		}

		tenant.StorageUsed += delta
		// 保存更新并验证业务规则
		if tenant.StorageUsed < 0 {
			logger.Errorf(ctx, "tenant storage used is negative %d: %d", tenant.ID, tenant.StorageUsed)
			tenant.StorageUsed = 0
		}

		return tx.Save(&tenant).Error
	})
}

func normalizeStorageMetadata(metadata types.JSON) types.JSON {
	if len(metadata) == 0 {
		return types.JSON([]byte("{}"))
	}
	return metadata
}

func (r *tenantRepository) ReserveStorage(
	ctx context.Context,
	reservation *types.TenantStorageReservation,
) (*types.TenantStorageReservation, error) {
	if reservation == nil || reservation.TenantID == 0 {
		return nil, errors.New("storage: tenant and reservation are required")
	}
	reservation.RefNo = strings.TrimSpace(reservation.RefNo)
	reservation.Operation = strings.TrimSpace(reservation.Operation)
	if reservation.RefNo == "" || reservation.Operation == "" {
		return nil, errors.New("storage: ref_no and operation are required")
	}
	if reservation.RequestedBytes < 0 {
		return nil, errors.New("storage: requested bytes must be non-negative")
	}
	now := time.Now().UTC()
	if reservation.ID == "" {
		reservation.ID = uuid.NewString()
	}
	if !reservation.ExpiresAt.After(now) {
		reservation.ExpiresAt = now.Add(time.Hour)
	}
	reservation.Status = types.StorageReservationStatusReserved
	reservation.MetadataJSON = normalizeStorageMetadata(reservation.MetadataJSON)

	var persisted types.TenantStorageReservation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant types.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&tenant, reservation.TenantID).Error; err != nil {
			return err
		}
		if err := tx.Model(&types.TenantStorageReservation{}).
			Where(
				"tenant_id = ? AND status = ? AND expires_at <= ?",
				reservation.TenantID,
				types.StorageReservationStatusReserved,
				now,
			).
			Updates(map[string]any{
				"status":       types.StorageReservationStatusExpired,
				"released_at":  now,
				"failure_code": "expired",
				"updated_at":   now,
			}).Error; err != nil {
			return err
		}

		err := tx.Where(
			"tenant_id = ? AND ref_no = ?",
			reservation.TenantID,
			reservation.RefNo,
		).First(&persisted).Error
		switch {
		case err == nil && persisted.Status == types.StorageReservationStatusCommitted:
			return nil
		case err == nil && persisted.Status == types.StorageReservationStatusReserved &&
			persisted.ExpiresAt.After(now):
			return nil
		case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		var activeReserved int64
		if err := tx.Model(&types.TenantStorageReservation{}).
			Where(
				"tenant_id = ? AND status = ? AND expires_at > ? AND ref_no <> ?",
				reservation.TenantID,
				types.StorageReservationStatusReserved,
				now,
				reservation.RefNo,
			).
			Select("COALESCE(SUM(requested_bytes), 0)").
			Scan(&activeReserved).Error; err != nil {
			return err
		}
		if tenant.StorageQuota > 0 &&
			tenant.StorageUsed+activeReserved+reservation.RequestedBytes > tenant.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}

		if err == nil {
			if updateErr := tx.Model(&types.TenantStorageReservation{}).
				Where("id = ?", persisted.ID).
				Updates(map[string]any{
					"actor_user_id":   reservation.ActorUserID,
					"operation":       reservation.Operation,
					"requested_bytes": reservation.RequestedBytes,
					"actual_bytes":    0,
					"status":          types.StorageReservationStatusReserved,
					"expires_at":      reservation.ExpiresAt,
					"committed_at":    nil,
					"released_at":     nil,
					"failure_code":    "",
					"metadata_json":   reservation.MetadataJSON,
					"updated_at":      now,
				}).Error; updateErr != nil {
				return updateErr
			}
			return tx.Where("id = ?", persisted.ID).First(&persisted).Error
		}
		if createErr := tx.Create(reservation).Error; createErr != nil {
			return createErr
		}
		persisted = *reservation
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &persisted, nil
}

func (r *tenantRepository) CommitStorageReservation(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	actualBytes int64,
	metadata types.JSON,
) (*types.TenantStorageTransaction, error) {
	refNo = strings.TrimSpace(refNo)
	if tenantID == 0 || refNo == "" {
		return nil, errors.New("storage: tenant_id and ref_no are required")
	}
	if actualBytes < 0 {
		return nil, errors.New("storage: actual bytes must be non-negative")
	}
	metadata = normalizeStorageMetadata(metadata)
	now := time.Now().UTC()
	var result types.TenantStorageTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where("tenant_id = ? AND ref_no = ?", tenantID, refNo).
			First(&result).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var tenant types.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&tenant, tenantID).Error; err != nil {
			return err
		}
		var reservation types.TenantStorageReservation
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("tenant_id = ? AND ref_no = ?", tenantID, refNo).
			First(&reservation).Error; err != nil {
			return err
		}
		if reservation.Status != types.StorageReservationStatusReserved {
			return fmt.Errorf("storage: reservation %s is %s", refNo, reservation.Status)
		}

		var otherReserved int64
		if err := tx.Model(&types.TenantStorageReservation{}).
			Where(
				"tenant_id = ? AND status = ? AND expires_at > ? AND ref_no <> ?",
				tenantID,
				types.StorageReservationStatusReserved,
				now,
				refNo,
			).
			Select("COALESCE(SUM(requested_bytes), 0)").
			Scan(&otherReserved).Error; err != nil {
			return err
		}
		if tenant.StorageQuota > 0 &&
			tenant.StorageUsed+otherReserved+actualBytes > tenant.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
		tenant.StorageUsed += actualBytes
		if err := tx.Model(&types.Tenant{}).
			Where("id = ?", tenant.ID).
			Update("storage_used", tenant.StorageUsed).Error; err != nil {
			return err
		}

		result = types.TenantStorageTransaction{
			ID:                    uuid.NewString(),
			TenantID:              tenantID,
			ActorUserID:           reservation.ActorUserID,
			ReservationID:         reservation.ID,
			RefNo:                 refNo,
			Operation:             reservation.Operation,
			AmountBytes:           actualBytes,
			StorageUsedAfterBytes: tenant.StorageUsed,
			MetadataJSON:          metadata,
			CreatedAt:             now,
		}
		if err := tx.Create(&result).Error; err != nil {
			return err
		}
		return tx.Model(&types.TenantStorageReservation{}).
			Where("id = ?", reservation.ID).
			Updates(map[string]any{
				"actual_bytes":  actualBytes,
				"status":        types.StorageReservationStatusCommitted,
				"committed_at":  now,
				"metadata_json": metadata,
				"updated_at":    now,
			}).Error
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *tenantRepository) ReleaseStorageReservation(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	failureCode string,
) error {
	refNo = strings.TrimSpace(refNo)
	if tenantID == 0 || refNo == "" {
		return nil
	}
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.TenantStorageReservation{}).
		Where(
			"tenant_id = ? AND ref_no = ? AND status = ?",
			tenantID,
			refNo,
			types.StorageReservationStatusReserved,
		).
		Updates(map[string]any{
			"status":       types.StorageReservationStatusReleased,
			"released_at":  now,
			"failure_code": strings.TrimSpace(failureCode),
			"updated_at":   now,
		}).Error
}

func (r *tenantRepository) RecordStorageTransaction(
	ctx context.Context,
	transaction *types.TenantStorageTransaction,
) (*types.TenantStorageTransaction, error) {
	if transaction == nil || transaction.TenantID == 0 {
		return nil, errors.New("storage: tenant and transaction are required")
	}
	transaction.RefNo = strings.TrimSpace(transaction.RefNo)
	transaction.Operation = strings.TrimSpace(transaction.Operation)
	if transaction.RefNo == "" || transaction.Operation == "" {
		return nil, errors.New("storage: ref_no and operation are required")
	}
	transaction.MetadataJSON = normalizeStorageMetadata(transaction.MetadataJSON)
	if transaction.ID == "" {
		transaction.ID = uuid.NewString()
	}
	if transaction.CreatedAt.IsZero() {
		transaction.CreatedAt = time.Now().UTC()
	}

	var persisted types.TenantStorageTransaction
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Where(
			"tenant_id = ? AND ref_no = ?",
			transaction.TenantID,
			transaction.RefNo,
		).First(&persisted).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		var tenant types.Tenant
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&tenant, transaction.TenantID).Error; err != nil {
			return err
		}
		if transaction.AmountBytes > 0 && tenant.StorageQuota > 0 {
			now := time.Now().UTC()
			var activeReserved int64
			if err := tx.Model(&types.TenantStorageReservation{}).
				Where(
					"tenant_id = ? AND status = ? AND expires_at > ?",
					transaction.TenantID,
					types.StorageReservationStatusReserved,
					now,
				).
				Select("COALESCE(SUM(requested_bytes), 0)").
				Scan(&activeReserved).Error; err != nil {
				return err
			}
			if tenant.StorageUsed+activeReserved+transaction.AmountBytes > tenant.StorageQuota {
				return types.NewStorageQuotaExceededError()
			}
		}
		nextUsed := tenant.StorageUsed + transaction.AmountBytes
		if nextUsed < 0 {
			nextUsed = 0
		}
		appliedAmount := nextUsed - tenant.StorageUsed
		if err := tx.Model(&types.Tenant{}).
			Where("id = ?", tenant.ID).
			Update("storage_used", nextUsed).Error; err != nil {
			return err
		}
		transaction.AmountBytes = appliedAmount
		transaction.StorageUsedAfterBytes = nextUsed
		if err := tx.Create(transaction).Error; err != nil {
			return err
		}
		persisted = *transaction
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &persisted, nil
}

func (r *tenantRepository) ListStorageTransactions(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.TenantStorageTransaction, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if tenantID > 0 {
		query = query.Where("tenant_id = ?", tenantID)
	}
	var rows []*types.TenantStorageTransaction
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// BulkSetStorageQuota writes quotaBytes to storage_quota for every
// tenant in one statement. We don't WHERE-filter (the action is
// "apply globally"), so the affected count equals the row count of
// the tenants table.
//
// No transaction here: the operation is a single statement and we
// don't want to hold a long lock just to update a single column. If
// a concurrent CreateTenant lands in the middle, the new row gets
// the new default via the system-setting resolver in the handler —
// no risk of the new tenant being skipped.
func (r *tenantRepository) BulkSetStorageQuota(ctx context.Context, quotaBytes int64) (int64, error) {
	res := r.db.WithContext(ctx).
		Model(&types.Tenant{}).
		Where("1 = 1"). // GORM refuses unconditional UPDATEs without an explicit WHERE
		Update("storage_quota", quotaBytes)
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
