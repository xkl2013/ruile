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

func (r *serviceSpaceRepository) ListTemplates(
	ctx context.Context,
	tenantID uint64,
) ([]*types.ServiceSpaceTemplate, error) {
	var records []*types.ServiceSpaceTemplateRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id IN ? AND status = ? AND deleted_at IS NULL", []uint64{0, tenantID}, string(types.ServiceSpaceTemplateStatusPublished)).
		Order("key ASC, version DESC").
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	latest := make(map[string]*types.ServiceSpaceTemplate)
	for _, record := range records {
		if record == nil {
			continue
		}
		template, convertErr := record.ToTemplate()
		if convertErr != nil {
			return nil, convertErr
		}
		if existing, ok := latest[template.Key]; !ok || template.Version > existing.Version {
			copy := template
			latest[template.Key] = &copy
		}
	}
	result := make([]*types.ServiceSpaceTemplate, 0, len(latest))
	for _, template := range latest {
		result = append(result, template)
	}
	return result, nil
}

func (r *serviceSpaceRepository) GetTemplate(
	ctx context.Context,
	tenantID uint64,
	key string,
	version int,
) (*types.ServiceSpaceTemplate, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id IN ? AND key = ? AND status = ? AND deleted_at IS NULL",
			[]uint64{0, tenantID}, strings.TrimSpace(key), string(types.ServiceSpaceTemplateStatusPublished))
	if version > 0 {
		query = query.Where("version = ?", version)
	} else {
		query = query.Order("version DESC")
	}
	var record types.ServiceSpaceTemplateRecord
	if err := query.First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	template, err := record.ToTemplate()
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (r *serviceSpaceRepository) CreateBlueprint(
	ctx context.Context,
	record *types.ServiceSpaceBlueprintRecord,
) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *serviceSpaceRepository) GetLatestBlueprint(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) (*types.ServiceSpaceBlueprintRecord, error) {
	var record types.ServiceSpaceBlueprintRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND deleted_at IS NULL", tenantID, serviceID).
		Order("version DESC").
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &record, err
}

func (r *serviceSpaceRepository) GetBlueprintByID(
	ctx context.Context,
	tenantID uint64,
	serviceID, blueprintID string,
) (*types.ServiceSpaceBlueprintRecord, error) {
	var record types.ServiceSpaceBlueprintRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, serviceID, blueprintID).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &record, err
}

func (r *serviceSpaceRepository) UpdateBlueprint(
	ctx context.Context,
	record *types.ServiceSpaceBlueprintRecord,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.ServiceSpaceBlueprintRecord{}).
		Where("tenant_id = ? AND id = ?", record.TenantID, record.ID).
		Updates(fields).Error
}

func (r *serviceSpaceRepository) GetTemplateApplicationByIdempotency(
	ctx context.Context,
	tenantID uint64,
	key string,
) (*types.ServiceSpaceTemplateApplicationRecord, error) {
	var record types.ServiceSpaceTemplateApplicationRecord
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND idempotency_key = ?", tenantID, strings.TrimSpace(key)).
		First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &record, err
}

func (r *serviceSpaceRepository) CreateTemplateApplication(
	ctx context.Context,
	record *types.ServiceSpaceTemplateApplicationRecord,
) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *serviceSpaceRepository) GetProfile(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) (*types.ServiceSpaceProfile, error) {
	var profile types.ServiceSpaceProfile
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("version DESC").
		First(&profile).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &profile, err
}

func (r *serviceSpaceRepository) UpsertProfile(
	ctx context.Context,
	profile *types.ServiceSpaceProfile,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.ServiceSpaceProfile{}).
			Where("tenant_id = ? AND service_id = ? AND frozen = ?", profile.TenantID, profile.ServiceID, false).
			Update("frozen", true).Error; err != nil {
			return err
		}
		return tx.Create(profile).Error
	})
}

func (r *serviceSpaceRepository) GetSummary(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) (*types.ServiceSpaceSummary, error) {
	var summary types.ServiceSpaceSummary
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("version DESC").
		First(&summary).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &summary, err
}

func (r *serviceSpaceRepository) UpsertSummary(
	ctx context.Context,
	summary *types.ServiceSpaceSummary,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.ServiceSpaceSummary{}).
			Where("tenant_id = ? AND service_id = ? AND frozen = ?", summary.TenantID, summary.ServiceID, false).
			Update("frozen", true).Error; err != nil {
			return err
		}
		return tx.Create(summary).Error
	})
}

func (r *serviceSpaceRepository) CreateContextSource(
	ctx context.Context,
	source *types.ServiceContextSource,
) error {
	return r.db.WithContext(ctx).Create(source).Error
}

func (r *serviceSpaceRepository) ListContextSources(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) ([]*types.ServiceContextSource, error) {
	var sources []*types.ServiceContextSource
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("created_at DESC").
		Find(&sources).Error
	return sources, err
}

func (r *serviceSpaceRepository) GetContextSourceBySource(
	ctx context.Context,
	tenantID uint64,
	serviceID, sourceType, sourceID string,
) (*types.ServiceContextSource, error) {
	var source types.ServiceContextSource
	err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND service_id = ? AND source_type = ? AND source_id = ?",
			tenantID,
			serviceID,
			strings.TrimSpace(sourceType),
			strings.TrimSpace(sourceID),
		).
		First(&source).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &source, err
}

func (r *serviceSpaceRepository) GetContextSource(
	ctx context.Context,
	tenantID uint64,
	serviceID, sourceID string,
) (*types.ServiceContextSource, error) {
	var source types.ServiceContextSource
	err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND service_id = ? AND id = ?",
			tenantID,
			serviceID,
			strings.TrimSpace(sourceID),
		).
		First(&source).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &source, err
}

func (r *serviceSpaceRepository) DeleteContextSource(
	ctx context.Context,
	tenantID uint64,
	serviceID, sourceID string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, sourceID).
		Delete(&types.ServiceContextSource{}).Error
}

func (r *serviceSpaceRepository) UpdateArtifactLifecycle(
	ctx context.Context,
	tenantID uint64,
	serviceID, artifactID, lifecycle string,
) error {
	return r.db.WithContext(ctx).Model(&types.ServiceArtifact{}).
		Where("tenant_id = ? AND service_id = ? AND artifact_id = ? AND is_current = ?",
			tenantID, serviceID, artifactID, true).
		Updates(map[string]any{"lifecycle": lifecycle, "updated_at": time.Now().UTC()}).Error
}

var _ interfaces.ServiceSpaceRepository = (*serviceSpaceRepository)(nil)
