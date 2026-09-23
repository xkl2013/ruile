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

type expertPackageRepository struct{ db *gorm.DB }

func NewExpertPackageRepository(db *gorm.DB) interfaces.ExpertPackageRepository {
	return &expertPackageRepository{db: db}
}

func (r *expertPackageRepository) Import(
	ctx context.Context,
	pkg *types.ExpertPackage,
	version *types.ExpertPackageVersion,
	definitions []*types.AgentDefinitionVersion,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.ExpertPackage
		err := tx.Where("tenant_id = ? AND package_key = ?", pkg.TenantID, pkg.PackageKey).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := tx.Create(pkg).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			pkg.ID = existing.ID
			if err := tx.Model(&types.ExpertPackage{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"display_name":  pkg.DisplayName,
				"description":   pkg.Description,
				"avatar":        pkg.Avatar,
				"source_format": pkg.SourceFormat,
				"source_uri":    pkg.SourceURI,
				"license":       pkg.License,
				"updated_at":    time.Now().UTC(),
			}).Error; err != nil {
				return err
			}
		}
		version.PackageID = pkg.ID
		if err := tx.Create(version).Error; err != nil {
			return err
		}
		for _, definition := range definitions {
			definition.PackageID = pkg.ID
			definition.PackageVersionID = version.ID
		}
		return tx.Create(&definitions).Error
	})
}

func (r *expertPackageRepository) ListPackages(ctx context.Context, tenantID uint64) ([]*types.ExpertPackage, error) {
	var packages []*types.ExpertPackage
	err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("updated_at DESC").
		Find(&packages).Error
	if err != nil {
		return nil, err
	}
	for _, pkg := range packages {
		if err := r.loadPackageRelations(ctx, pkg); err != nil {
			return nil, err
		}
	}
	return packages, nil
}

func (r *expertPackageRepository) ListPublishedExperts(ctx context.Context, tenantID uint64) ([]*types.PublishedExpert, error) {
	var experts []*types.PublishedExpert
	err := r.db.WithContext(ctx).Table("agent_definition_versions AS d").
		Select(`
			d.package_id,
			d.package_version_id,
			p.package_key,
			p.display_name AS package_display_name,
			p.description AS package_description,
			p.avatar,
			v.version AS package_version,
			d.id AS definition_id,
			d.agent_id,
			d.version,
			d.display_name,
			COALESCE(NULLIF(d.description, ''), p.description) AS description,
			d.domain,
			d.output_contract,
			d.skills,
			d.capabilities
		`).
		Joins("JOIN expert_package_versions v ON v.id = d.package_version_id AND v.deleted_at IS NULL").
		Joins("JOIN expert_packages p ON p.id = d.package_id AND p.deleted_at IS NULL").
		Where("p.tenant_id = ? AND d.deleted_at IS NULL AND v.state = ?", tenantID, types.ExpertPackageVersionPublished).
		Order("d.display_name ASC, d.version DESC").
		Find(&experts).Error
	if err != nil {
		return nil, err
	}
	return experts, nil
}

func (r *expertPackageRepository) GetPackage(ctx context.Context, tenantID uint64, id string) (*types.ExpertPackage, error) {
	var pkg types.ExpertPackage
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND id = ?", tenantID, strings.TrimSpace(id)).
		First(&pkg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.loadPackageRelations(ctx, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (r *expertPackageRepository) GetVersion(
	ctx context.Context,
	tenantID uint64,
	packageID, versionID string,
) (*types.ExpertPackageVersion, error) {
	var version types.ExpertPackageVersion
	err := r.db.WithContext(ctx).Table("expert_package_versions AS v").
		Select("v.*").
		Joins("JOIN expert_packages p ON p.id = v.package_id AND p.deleted_at IS NULL").
		Where("p.tenant_id = ? AND v.package_id = ? AND v.id = ?", tenantID, strings.TrimSpace(packageID), strings.TrimSpace(versionID)).
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("package_version_id = ?", version.ID).
		Order("display_name ASC").Find(&version.Definitions).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *expertPackageRepository) GetVersionByPackageKey(
	ctx context.Context,
	tenantID uint64,
	packageKey, versionValue string,
) (*types.ExpertPackageVersion, error) {
	var version types.ExpertPackageVersion
	err := r.db.WithContext(ctx).Table("expert_package_versions AS v").
		Select("v.*").
		Joins("JOIN expert_packages p ON p.id = v.package_id AND p.deleted_at IS NULL").
		Where(
			"p.tenant_id = ? AND p.package_key = ? AND v.version = ? AND v.deleted_at IS NULL",
			tenantID,
			strings.TrimSpace(packageKey),
			strings.TrimSpace(versionValue),
		).
		First(&version).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if err := r.db.WithContext(ctx).Where("package_version_id = ?", version.ID).
		Order("display_name ASC").Find(&version.Definitions).Error; err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *expertPackageRepository) GetPublishedDefinition(
	ctx context.Context,
	tenantID uint64,
	packageID, definitionID string,
) (*types.AgentDefinitionVersion, error) {
	var definition types.AgentDefinitionVersion
	err := r.db.WithContext(ctx).Table("agent_definition_versions AS d").
		Select("d.*").
		Joins("JOIN expert_package_versions v ON v.id = d.package_version_id AND v.deleted_at IS NULL").
		Joins("JOIN expert_packages p ON p.id = d.package_id AND p.deleted_at IS NULL").
		Where(
			"p.tenant_id = ? AND d.package_id = ? AND d.id = ? AND d.deleted_at IS NULL AND v.state = ?",
			tenantID,
			strings.TrimSpace(packageID),
			strings.TrimSpace(definitionID),
			types.ExpertPackageVersionPublished,
		).
		First(&definition).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &definition, err
}

func (r *expertPackageRepository) PublishVersion(
	ctx context.Context,
	tenantID uint64,
	packageID, versionID, actorID string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var version types.ExpertPackageVersion
		if err := tx.Table("expert_package_versions AS v").Select("v.*").
			Joins("JOIN expert_packages p ON p.id = v.package_id AND p.deleted_at IS NULL").
			Where("p.tenant_id = ? AND v.package_id = ? AND v.id = ?", tenantID, strings.TrimSpace(packageID), strings.TrimSpace(versionID)).
			First(&version).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if err := tx.Model(&types.ExpertPackageVersion{}).
			Where("package_id = ? AND id <> ? AND state = ?", version.PackageID, version.ID, types.ExpertPackageVersionPublished).
			Update("state", types.ExpertPackageVersionDeprecated).Error; err != nil {
			return err
		}
		return tx.Model(&types.ExpertPackageVersion{}).Where("id = ?", version.ID).Updates(map[string]any{
			"state":        types.ExpertPackageVersionPublished,
			"published_by": strings.TrimSpace(actorID),
			"published_at": now,
			"updated_at":   now,
		}).Error
	})
}

func (r *expertPackageRepository) loadPackageRelations(ctx context.Context, pkg *types.ExpertPackage) error {
	if err := r.db.WithContext(ctx).Where("package_id = ?", pkg.ID).Order("created_at DESC").Find(&pkg.Versions).Error; err != nil {
		return err
	}
	for _, version := range pkg.Versions {
		if err := r.db.WithContext(ctx).Where("package_version_id = ?", version.ID).
			Order("display_name ASC").Find(&version.Definitions).Error; err != nil {
			return err
		}
	}
	return nil
}
