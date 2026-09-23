package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/gorm"
)

func (r *organizeRepository) ListTemplates(
	ctx context.Context,
	tenantID uint64,
	userID string,
) ([]*types.OrganizeTemplate, error) {
	var templates []*types.OrganizeTemplate
	err := r.db.WithContext(ctx).
		Where("status = ? AND (scope = ? OR (scope = ? AND tenant_id = ?) OR (scope = ? AND tenant_id = ? AND owner_user_id = ?))",
			types.OrganizeTemplateStatusEnabled,
			types.OrganizeTemplateScopePlatform,
			types.OrganizeTemplateScopeTenant, tenantID,
			types.OrganizeTemplateScopePersonal, tenantID, userID,
		).
		Order("sort_order ASC").
		Order("created_at ASC").
		Find(&templates).Error
	return templates, err
}

func (r *organizeRepository) GetTemplate(
	ctx context.Context,
	tenantID uint64,
	userID string,
	key string,
) (*types.OrganizeTemplate, error) {
	var template types.OrganizeTemplate
	err := r.db.WithContext(ctx).
		Where("key = ? AND status = ? AND (scope = ? OR (scope = ? AND tenant_id = ?) OR (scope = ? AND tenant_id = ? AND owner_user_id = ?))",
			key,
			types.OrganizeTemplateStatusEnabled,
			types.OrganizeTemplateScopePlatform,
			types.OrganizeTemplateScopeTenant, tenantID,
			types.OrganizeTemplateScopePersonal, tenantID, userID,
		).
		Order("CASE scope WHEN 'personal' THEN 1 WHEN 'tenant' THEN 2 ELSE 3 END").
		First(&template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &template, err
}

func (r *organizeRepository) GetTemplateVersion(
	ctx context.Context,
	templateID string,
	version string,
) (*types.OrganizeTemplateVersion, error) {
	var item types.OrganizeTemplateVersion
	err := r.db.WithContext(ctx).
		Where("template_id = ? AND version = ?", templateID, version).
		First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &item, err
}

func (r *organizeRepository) CreateConfig(ctx context.Context, config *types.OrganizeConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *organizeRepository) GetConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeConfig, error) {
	var config types.OrganizeConfig
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		First(&config).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.fillOrganizeConfigs(ctx, tenantID, userID, []*types.OrganizeConfig{&config}); err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *organizeRepository) UpdateConfig(ctx context.Context, config *types.OrganizeConfig) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeConfig{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", config.TenantID, config.UserID, config.ID).
		Select(
			"name",
			"template_key",
			"instruction",
			"expert_ids",
			"schedule",
			"status",
			"next_run_at",
			"last_run_at",
			"metadata",
			"updated_at",
		).
		Updates(config).Error
}

func (r *organizeRepository) DeleteConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		Delete(&types.OrganizeConfig{}).Error
}

func (r *organizeRepository) ListConfigs(
	ctx context.Context,
	query types.OrganizeConfigQuery,
) ([]*types.OrganizeConfig, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeConfig{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.Status != "" {
		dbq = dbq.Where("status = ?", query.Status)
	}
	dbq = applyOrganizeKeyword(dbq, query.Keyword, "name", "instruction", "template_key")

	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var configs []*types.OrganizeConfig
	err := dbq.
		Order("updated_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&configs).Error
	if err != nil {
		return nil, 0, err
	}
	if err := r.fillOrganizeConfigs(ctx, query.TenantID, query.UserID, configs); err != nil {
		return nil, 0, err
	}
	return configs, total, nil
}

func (r *organizeRepository) ListDueConfigs(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]*types.OrganizeConfig, error) {
	if limit <= 0 {
		limit = 100
	}
	var configs []*types.OrganizeConfig
	err := r.db.WithContext(ctx).
		Where("status = ? AND schedule <> ? AND next_run_at IS NOT NULL AND next_run_at <= ?",
			types.OrganizeConfigStatusActive,
			types.OrganizeScheduleManual,
			now,
		).
		Order("next_run_at ASC").
		Limit(limit).
		Find(&configs).Error
	return configs, err
}

func (r *organizeRepository) CreateJob(ctx context.Context, job *types.OrganizeJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *organizeRepository) GetJob(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeJob, error) {
	var job types.OrganizeJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, userID, id).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &job, err
}

func (r *organizeRepository) UpdateJob(ctx context.Context, job *types.OrganizeJob) error {
	return r.db.WithContext(ctx).
		Model(&types.OrganizeJob{}).
		Where("tenant_id = ? AND user_id = ? AND id = ?", job.TenantID, job.UserID, job.ID).
		Select(
			"status",
			"stage",
			"progress",
			"requirement",
			"memory_ids",
			"model_id",
			"prompt_hash",
			"output_id",
			"summary",
			"result",
			"error_message",
			"dedupe_key",
			"scheduled_for",
			"started_at",
			"finished_at",
			"updated_at",
		).
		Updates(job).Error
}

func (r *organizeRepository) ListJobs(
	ctx context.Context,
	query types.OrganizeJobQuery,
) ([]*types.OrganizeJob, int64, error) {
	dbq := r.db.WithContext(ctx).Model(&types.OrganizeJob{}).
		Where("tenant_id = ? AND user_id = ?", query.TenantID, query.UserID)
	if query.ConfigID != "" {
		dbq = dbq.Where("config_id = ?", query.ConfigID)
	}
	if query.Status != "" {
		dbq = dbq.Where("status = ?", query.Status)
	}
	var total int64
	if err := dbq.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var jobs []*types.OrganizeJob
	err := dbq.
		Order("created_at DESC").
		Limit(query.PageSize).
		Offset((query.Page - 1) * query.PageSize).
		Find(&jobs).Error
	return jobs, total, err
}

func (r *organizeRepository) ListMemoriesByIDs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	ids []string,
) ([]*types.OrganizeMemory, error) {
	if len(ids) == 0 {
		return []*types.OrganizeMemory{}, nil
	}
	var memories []*types.OrganizeMemory
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id IN ?", tenantID, userID, ids).
		Order("occurred_at ASC").
		Find(&memories).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[string]*types.OrganizeMemory, len(memories))
	for _, memory := range memories {
		byID[memory.ID] = memory
	}
	ordered := make([]*types.OrganizeMemory, 0, len(memories))
	for _, id := range ids {
		if memory := byID[id]; memory != nil {
			ordered = append(ordered, memory)
		}
	}
	return ordered, nil
}

func (r *organizeRepository) fillOrganizeConfigs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	configs []*types.OrganizeConfig,
) error {
	if len(configs) == 0 {
		return nil
	}
	configIDs := make([]string, 0, len(configs))
	templateKeys := make([]string, 0, len(configs))
	seenTemplateKeys := make(map[string]struct{}, len(configs))
	byID := make(map[string]*types.OrganizeConfig, len(configs))
	for _, config := range configs {
		configIDs = append(configIDs, config.ID)
		byID[config.ID] = config
		if key := strings.TrimSpace(config.TemplateKey); key != "" {
			if _, ok := seenTemplateKeys[key]; !ok {
				seenTemplateKeys[key] = struct{}{}
				templateKeys = append(templateKeys, key)
			}
		}
	}

	var jobs []*types.OrganizeJob
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND config_id IN ?", tenantID, userID, configIDs).
		Order("created_at DESC").
		Find(&jobs).Error; err != nil {
		return err
	}
	for _, job := range jobs {
		config := byID[job.ConfigID]
		if config == nil {
			continue
		}
		config.JobCount++
		if config.LatestJob == nil {
			config.LatestJob = job
		}
	}

	if len(templateKeys) == 0 {
		return nil
	}
	var templates []*types.OrganizeTemplate
	if err := r.db.WithContext(ctx).
		Where("key IN ? AND status = ? AND (scope = ? OR (scope = ? AND tenant_id = ?) OR (scope = ? AND tenant_id = ? AND owner_user_id = ?))",
			templateKeys,
			types.OrganizeTemplateStatusEnabled,
			types.OrganizeTemplateScopePlatform,
			types.OrganizeTemplateScopeTenant, tenantID,
			types.OrganizeTemplateScopePersonal, tenantID, userID,
		).
		Order("CASE scope WHEN 'personal' THEN 1 WHEN 'tenant' THEN 2 ELSE 3 END").
		Find(&templates).Error; err != nil {
		return err
	}
	templatesByKey := make(map[string]*types.OrganizeTemplate, len(templates))
	for _, template := range templates {
		if templatesByKey[template.Key] == nil {
			templatesByKey[template.Key] = template
		}
	}
	for _, config := range configs {
		config.Template = templatesByKey[config.TemplateKey]
	}
	return nil
}
