package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type tenantSkillSnapshotRepository struct {
	db *gorm.DB
}

func NewTenantSkillSnapshotRepository(db *gorm.DB) interfaces.TenantSkillSnapshotRepository {
	return &tenantSkillSnapshotRepository{db: db}
}

func (r *tenantSkillSnapshotRepository) Create(
	ctx context.Context,
	snapshot *types.TenantSkillSnapshot,
) error {
	if snapshot == nil {
		return errors.New("skill snapshot is required")
	}
	return r.db.WithContext(ctx).Create(snapshot).Error
}

func (r *tenantSkillSnapshotRepository) List(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.TenantSkillSnapshot, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var snapshots []*types.TenantSkillSnapshot
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("generation DESC").
		Limit(limit).
		Find(&snapshots).Error
	return snapshots, err
}

func (r *tenantSkillSnapshotRepository) GetByID(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillSnapshot, error) {
	var snapshot types.TenantSkillSnapshot
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *tenantSkillSnapshotRepository) GetActive(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantSkillSnapshot, error) {
	var snapshot types.TenantSkillSnapshot
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND state = ?", tenantID, types.TenantSkillSnapshotStateActive).
		Order("generation DESC").
		First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (r *tenantSkillSnapshotRepository) Activate(
	ctx context.Context,
	tenantID uint64,
	id string,
	activatedAt, supersededAt time.Time,
) error {
	id = strings.TrimSpace(id)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.TenantSkillSnapshot{}).
			Where("tenant_id = ? AND state = ?", tenantID, types.TenantSkillSnapshotStateActive).
			Updates(map[string]any{
				"state":         types.TenantSkillSnapshotStateSuperseded,
				"superseded_at": supersededAt,
			}).Error; err != nil {
			return err
		}
		result := tx.Model(&types.TenantSkillSnapshot{}).
			Where("tenant_id = ? AND id = ?", tenantID, id).
			Updates(map[string]any{
				"state":        types.TenantSkillSnapshotStateActive,
				"activated_at": activatedAt,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}
