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

func (r *tenantSkillRepository) Update(ctx context.Context, skill *types.TenantSkill) error {
	if skill == nil {
		return errors.New("skill is required")
	}
	return r.db.WithContext(ctx).
		Model(&types.TenantSkill{}).
		Where("tenant_id = ? AND id = ?", skill.TenantID, strings.TrimSpace(skill.ID)).
		Updates(map[string]any{
			"name":          skill.Name,
			"description":   skill.Description,
			"version":       skill.Version,
			"bundle_path":   skill.BundlePath,
			"bundle_sha256": skill.BundleSHA,
			"source_type":   skill.SourceType,
			"source_ref":    skill.SourceRef,
			"env_vars":      skill.EnvVars,
			"status":        skill.Status,
			"enabled":       skill.Enabled,
			"updated_at":    skill.UpdatedAt,
		}).Error
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

func (r *tenantSkillRepository) SetEnvVars(
	ctx context.Context,
	tenantID uint64,
	id string,
	envVars types.TenantSkillEnvVars,
) error {
	return r.db.WithContext(ctx).
		Model(&types.TenantSkill{}).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Updates(map[string]any{
			"env_vars":   envVars,
			"updated_at": gorm.Expr("CURRENT_TIMESTAMP"),
		}).Error
}

func (r *tenantSkillRepository) Delete(ctx context.Context, tenantID uint64, id string) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		Delete(&types.TenantSkill{}).Error
}

func (r *tenantSkillRepository) CreateInstallJob(ctx context.Context, job *types.TenantSkillInstallJob) error {
	if job == nil {
		return errors.New("install job is required")
	}
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *tenantSkillRepository) UpdateInstallJob(ctx context.Context, job *types.TenantSkillInstallJob) error {
	if job == nil {
		return errors.New("install job is required")
	}
	return r.db.WithContext(ctx).
		Model(&types.TenantSkillInstallJob{}).
		Where("tenant_id = ? AND id = ?", job.TenantID, strings.TrimSpace(job.ID)).
		Updates(map[string]any{
			"source_type":      job.SourceType,
			"source_ref":       job.SourceRef,
			"skill_id":         job.SkillID,
			"status":           job.Status,
			"error":            job.Error,
			"attempt":          job.Attempt,
			"cancel_requested": job.CancelRequested,
			"started_at":       job.StartedAt,
			"finished_at":      job.FinishedAt,
			"updated_at":       job.UpdatedAt,
		}).Error
}

func (r *tenantSkillRepository) GetInstallJob(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillInstallJob, error) {
	var job types.TenantSkillInstallJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *tenantSkillRepository) ListInstallJobs(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.TenantSkillInstallJob, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var jobs []*types.TenantSkillInstallJob
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Limit(limit).
		Find(&jobs).Error
	return jobs, err
}
