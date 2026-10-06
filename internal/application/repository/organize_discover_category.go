package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
)

func (r *organizeRepository) ListDiscoverCategories(
	ctx context.Context,
) ([]*types.OrganizeDiscoverCategoryRecord, error) {
	var categories []*types.OrganizeDiscoverCategoryRecord
	err := r.db.WithContext(ctx).
		Where("status = ?", types.OrganizeDiscoverCategoryStatusEnabled).
		Order("sort_order ASC").
		Order("created_at ASC").
		Find(&categories).Error
	return categories, err
}

func (r *organizeRepository) ListAdminDiscoverCategories(
	ctx context.Context,
	query types.OrganizeDiscoverCategoryQuery,
) ([]*types.OrganizeDiscoverCategoryRecord, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeDiscoverCategoryRecord{})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		dbq = applyOrganizeKeyword(dbq, keyword, "key", "label", "description")
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		dbq = dbq.Where("status = ?", status)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var categories []*types.OrganizeDiscoverCategoryRecord
	if err := dbq.
		Order("sort_order ASC").
		Order("updated_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&categories).Error; err != nil {
		return nil, 0, err
	}
	return categories, total, nil
}

func (r *organizeRepository) CreateDiscoverCategory(
	ctx context.Context,
	category *types.OrganizeDiscoverCategoryRecord,
) error {
	return r.db.WithContext(ctx).Create(category).Error
}

func (r *organizeRepository) UpdateDiscoverCategory(
	ctx context.Context,
	category *types.OrganizeDiscoverCategoryRecord,
) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeDiscoverCategoryRecord{}).
		Where("id = ?", category.ID).
		Select("label", "description", "sort_order", "status", "updated_at").
		Updates(category).Error
}

func (r *organizeRepository) getDiscoverCategory(
	ctx context.Context,
	key string,
) (*types.OrganizeDiscoverCategoryRecord, error) {
	var category types.OrganizeDiscoverCategoryRecord
	err := r.db.WithContext(ctx).Where("key = ?", strings.TrimSpace(key)).First(&category).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &category, err
}
