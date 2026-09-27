package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

type tenantSkillRepository struct {
	db *gorm.DB
}

func NewTenantSkillRepository(db *gorm.DB) interfaces.TenantSkillRepository {
	return &tenantSkillRepository{db: db}
}

func (r *tenantSkillRepository) Create(ctx context.Context, skill *types.TenantSkill) error {
	return r.db.WithContext(ctx).Create(skill).Error
}

func (r *tenantSkillRepository) List(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error) {
	var skills []*types.TenantSkill
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("name ASC").
		Find(&skills).Error
	return skills, err
}

func (r *tenantSkillRepository) GetByID(ctx context.Context, tenantID uint64, id string) (*types.TenantSkill, error) {
	var skill types.TenantSkill
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&skill).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *tenantSkillRepository) GetByName(ctx context.Context, tenantID uint64, name string) (*types.TenantSkill, error) {
	var skill types.TenantSkill
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND name = ?", tenantID, strings.TrimSpace(name)).
		First(&skill).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &skill, nil
}

func (r *tenantSkillRepository) SetEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error {
	return r.db.WithContext(ctx).
		Model(&types.TenantSkill{}).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Updates(map[string]any{"enabled": enabled}).Error
}

func (r *tenantSkillRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Delete(&types.TenantSkill{}).Error
}
