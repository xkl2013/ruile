package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
)

func (r *organizeRepository) ListAdminTemplates(
	ctx context.Context,
	query types.OrganizeTemplateAdminQuery,
) ([]*types.OrganizeTemplate, int64, error) {
	dbq := r.db.WithContext(ctx).
		Model(&types.OrganizeTemplate{}).
		Where("tenant_id = ? AND owner_user_id = ? AND scope = ?", 0, "", types.OrganizeTemplateScopePlatform).
		Where("key NOT IN ?", []string{"note_import_meta", "note_audio_transcribe", "output_card_meta"})
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		dbq = applyOrganizeKeyword(dbq, keyword, "key", "name", "description", "default_instruction")
	}
	if scene := strings.TrimSpace(query.Scene); scene != "" {
		dbq = dbq.Where("scene = ?", scene)
	}
	if status := strings.TrimSpace(query.Status); status != "" {
		dbq = dbq.Where("status = ?", status)
	}

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var templates []*types.OrganizeTemplate
	if err := dbq.
		Order("sort_order ASC").
		Order("updated_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&templates).Error; err != nil {
		return nil, 0, err
	}
	return templates, total, nil
}

func (r *organizeRepository) GetAdminTemplate(ctx context.Context, key string) (*types.OrganizeTemplate, error) {
	var template types.OrganizeTemplate
	err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND owner_user_id = ? AND scope = ? AND key = ?",
			0,
			"",
			types.OrganizeTemplateScopePlatform,
			strings.TrimSpace(key),
		).
		First(&template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *organizeRepository) CreateTemplate(ctx context.Context, template *types.OrganizeTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

func (r *organizeRepository) UpdateTemplate(ctx context.Context, template *types.OrganizeTemplate) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeTemplate{}).
		Where(
			"id = ? AND tenant_id = ? AND owner_user_id = ? AND scope = ?",
			template.ID,
			0,
			"",
			types.OrganizeTemplateScopePlatform,
		).
		Select(
			"name",
			"scene",
			"description",
			"output_label",
			"icon",
			"default_instruction",
			"markdown_template",
			"expert_ids",
			"spec",
			"status",
			"published_version",
			"published_at",
			"published_by",
			"validation_result",
			"sort_order",
			"updated_at",
		).
		Updates(template).Error
}

func (r *organizeRepository) ListTemplateVersions(
	ctx context.Context,
	query types.OrganizeTemplateVersionQuery,
) ([]*types.OrganizeTemplateVersion, int64, error) {
	dbq := r.db.WithContext(ctx).
		Model(&types.OrganizeTemplateVersion{}).
		Where("template_key = ?", strings.TrimSpace(query.TemplateKey))

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var versions []*types.OrganizeTemplateVersion
	if err := dbq.
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&versions).Error; err != nil {
		return nil, 0, err
	}
	return versions, total, nil
}

func (r *organizeRepository) CreateTemplateVersion(
	ctx context.Context,
	version *types.OrganizeTemplateVersion,
) error {
	return r.db.WithContext(ctx).Create(version).Error
}
