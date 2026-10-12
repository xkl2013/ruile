package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type tenantSandboxConfigRepository struct {
	db *gorm.DB
}

func NewTenantSandboxConfigRepository(db *gorm.DB) interfaces.TenantSandboxConfigRepository {
	return &tenantSandboxConfigRepository{db: db}
}

func (r *tenantSandboxConfigRepository) Create(ctx context.Context, config *types.TenantSandboxConfig) error {
	if config == nil {
		return errors.New("sandbox config is required")
	}
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *tenantSandboxConfigRepository) Update(ctx context.Context, config *types.TenantSandboxConfig) error {
	if config == nil {
		return errors.New("sandbox config is required")
	}
	return r.db.WithContext(ctx).
		Model(&types.TenantSandboxConfig{}).
		Where("tenant_id = ? AND id = ?", config.TenantID, strings.TrimSpace(config.ID)).
		Updates(map[string]any{
			"name":             config.Name,
			"mode":             config.Mode,
			"enabled":          config.Enabled,
			"fallback_enabled": config.FallbackEnabled,
			"timeout_seconds":  config.TimeoutSeconds,
			"docker_image":     config.DockerImage,
			"updated_at":       config.UpdatedAt,
		}).Error
}

func (r *tenantSandboxConfigRepository) List(ctx context.Context, tenantID uint64) ([]*types.TenantSandboxConfig, error) {
	var configs []*types.TenantSandboxConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("active DESC, name ASC").
		Find(&configs).Error
	return configs, err
}

func (r *tenantSandboxConfigRepository) GetByID(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSandboxConfig, error) {
	var config types.TenantSandboxConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *tenantSandboxConfigRepository) GetActive(
	ctx context.Context,
	tenantID uint64,
) (*types.TenantSandboxConfig, error) {
	var config types.TenantSandboxConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND active = ? AND enabled = ?", tenantID, true, true).
		Order("updated_at DESC").
		First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *tenantSandboxConfigRepository) SetActive(ctx context.Context, tenantID uint64, id string) error {
	id = strings.TrimSpace(id)
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.TenantSandboxConfig{}).
			Where("tenant_id = ?", tenantID).
			Update("active", false).Error; err != nil {
			return err
		}
		result := tx.Model(&types.TenantSandboxConfig{}).
			Where("tenant_id = ? AND id = ? AND enabled = ?", tenantID, id, true).
			Updates(map[string]any{"active": true})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *tenantSandboxConfigRepository) SetInactive(ctx context.Context, tenantID uint64, id string) error {
	result := r.db.WithContext(ctx).
		Model(&types.TenantSandboxConfig{}).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Update("active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *tenantSandboxConfigRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	result := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Delete(&types.TenantSandboxConfig{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
