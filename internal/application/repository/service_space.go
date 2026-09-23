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

type serviceSpaceRepository struct {
	db *gorm.DB
}

func NewServiceSpaceRepository(db *gorm.DB) interfaces.ServiceSpaceRepository {
	return &serviceSpaceRepository{db: db}
}

func (r *serviceSpaceRepository) Create(
	ctx context.Context,
	service *types.ServiceSpace,
	owner *types.ServiceSpaceMember,
	experts []*types.ServiceExpertBinding,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if !service.IsDefault {
			var existing int64
			if err := tx.Model(&types.ServiceSpace{}).
				Where("tenant_id = ? AND owner_user_id = ?", service.TenantID, service.OwnerUserID).
				Count(&existing).Error; err != nil {
				return err
			}
			service.IsDefault = existing == 0
		}
		if service.IsDefault {
			if err := tx.Model(&types.ServiceSpace{}).
				Where("tenant_id = ? AND owner_user_id = ? AND is_default = ?", service.TenantID, service.OwnerUserID, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(service).Error; err != nil {
			return err
		}
		owner.ServiceID = service.ID
		if err := tx.Create(owner).Error; err != nil {
			return err
		}
		for _, expert := range experts {
			expert.ServiceID = service.ID
		}
		if len(experts) > 0 {
			if err := tx.Create(&experts).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *serviceSpaceRepository) ListAccessible(
	ctx context.Context,
	tenantID uint64,
	userID string,
	includeArchived bool,
) ([]*types.ServiceSpaceView, error) {
	rows := make([]*types.ServiceSpaceView, 0)
	query := r.db.WithContext(ctx).
		Table("services AS s").
		Select(`s.*,
			COALESCE(sm.role, CASE WHEN s.owner_user_id = ? THEN ? ELSE ? END) AS role,
			(SELECT COUNT(*) FROM service_members mc
			 WHERE mc.service_id = s.id AND mc.status = ? AND mc.deleted_at IS NULL) AS member_count`,
			userID,
			types.ServiceMemberRoleOwner,
			types.ServiceMemberRoleViewer,
			types.ServiceMemberStatusActive,
		).
		Joins(`LEFT JOIN service_members sm
			ON sm.service_id = s.id AND sm.user_id = ? AND sm.status = ? AND sm.deleted_at IS NULL`,
			userID, types.ServiceMemberStatusActive).
		Where("s.tenant_id = ? AND s.deleted_at IS NULL", tenantID).
		Where("(s.owner_user_id = ? OR sm.id IS NOT NULL OR s.visibility = ?)", userID, types.ServiceSpaceVisibilityTenant)
	if !includeArchived {
		query = query.Where("s.state <> ?", types.ServiceSpaceStateArchived)
	}
	err := query.Order("s.is_default DESC, s.updated_at DESC").Scan(&rows).Error
	return rows, err
}

func (r *serviceSpaceRepository) GetAccessible(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID string,
) (*types.ServiceSpaceView, error) {
	var row types.ServiceSpaceView
	err := r.db.WithContext(ctx).
		Table("services AS s").
		Select(`s.*,
			COALESCE(sm.role, CASE WHEN s.owner_user_id = ? THEN ? ELSE ? END) AS role,
			(SELECT COUNT(*) FROM service_members mc
			 WHERE mc.service_id = s.id AND mc.status = ? AND mc.deleted_at IS NULL) AS member_count`,
			userID,
			types.ServiceMemberRoleOwner,
			types.ServiceMemberRoleViewer,
			types.ServiceMemberStatusActive,
		).
		Joins(`LEFT JOIN service_members sm
			ON sm.service_id = s.id AND sm.user_id = ? AND sm.status = ? AND sm.deleted_at IS NULL`,
			userID, types.ServiceMemberStatusActive).
		Where("s.tenant_id = ? AND s.id = ? AND s.deleted_at IS NULL", tenantID, strings.TrimSpace(serviceID)).
		Where("(s.owner_user_id = ? OR sm.id IS NOT NULL OR s.visibility = ?)", userID, types.ServiceSpaceVisibilityTenant).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

func (r *serviceSpaceRepository) Update(
	ctx context.Context,
	service *types.ServiceSpace,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.ServiceSpace{}).
		Where("tenant_id = ? AND id = ?", service.TenantID, service.ID).
		Updates(fields).Error
}

func (r *serviceSpaceRepository) SetDefault(ctx context.Context, tenantID uint64, userID, serviceID string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&types.ServiceSpace{}).
			Where("tenant_id = ? AND owner_user_id = ? AND is_default = ?", tenantID, userID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&types.ServiceSpace{}).
			Where("tenant_id = ? AND id = ?", tenantID, serviceID).
			Updates(map[string]any{"is_default": true, "updated_at": time.Now().UTC()}).Error
	})
}

func (r *serviceSpaceRepository) Delete(ctx context.Context, tenantID uint64, serviceID string) error {
	return r.db.WithContext(ctx).Where("tenant_id = ? AND id = ?", tenantID, serviceID).
		Delete(&types.ServiceSpace{}).Error
}

func (r *serviceSpaceRepository) ListMembers(
	ctx context.Context,
	tenantID uint64,
	serviceID, status string,
) ([]*types.ServiceSpaceMember, error) {
	var members []*types.ServiceSpaceMember
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Order("CASE role WHEN 'owner' THEN 1 WHEN 'admin' THEN 2 WHEN 'editor' THEN 3 ELSE 4 END").
		Order("created_at ASC").
		Find(&members).Error
	return members, err
}

func (r *serviceSpaceRepository) GetMember(
	ctx context.Context,
	tenantID uint64,
	serviceID, userID string,
) (*types.ServiceSpaceMember, error) {
	var member types.ServiceSpaceMember
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND user_id = ?", tenantID, serviceID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &member, err
}

func (r *serviceSpaceRepository) UpsertMember(ctx context.Context, member *types.ServiceSpaceMember) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing types.ServiceSpaceMember
		err := tx.Unscoped().
			Where("tenant_id = ? AND service_id = ? AND user_id = ?", member.TenantID, member.ServiceID, member.UserID).
			First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(member).Error
		}
		if err != nil {
			return err
		}
		return tx.Unscoped().Model(&types.ServiceSpaceMember{}).
			Where("id = ?", existing.ID).
			Updates(map[string]any{
				"role":       member.Role,
				"status":     types.ServiceMemberStatusActive,
				"invited_by": member.InvitedBy,
				"joined_at":  now,
				"left_at":    nil,
				"updated_at": now,
				"deleted_at": nil,
			}).Error
	})
}

func (r *serviceSpaceRepository) UpdateMemberRole(
	ctx context.Context,
	tenantID uint64,
	serviceID, userID, role string,
) error {
	return r.db.WithContext(ctx).Model(&types.ServiceSpaceMember{}).
		Where("tenant_id = ? AND service_id = ? AND user_id = ? AND status = ?", tenantID, serviceID, userID, types.ServiceMemberStatusActive).
		Updates(map[string]any{"role": role, "updated_at": time.Now().UTC()}).Error
}

func (r *serviceSpaceRepository) MarkMemberLeft(
	ctx context.Context,
	tenantID uint64,
	serviceID, userID string,
) error {
	now := time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.ServiceSpaceMember{}).
		Where("tenant_id = ? AND service_id = ? AND user_id = ? AND status = ?", tenantID, serviceID, userID, types.ServiceMemberStatusActive).
		Updates(map[string]any{"status": types.ServiceMemberStatusLeft, "left_at": now, "updated_at": now}).Error
}

func (r *serviceSpaceRepository) ListExperts(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) ([]*types.ServiceExpertBinding, error) {
	var experts []*types.ServiceExpertBinding
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("display_order ASC, created_at ASC").
		Find(&experts).Error
	return experts, err
}

func (r *serviceSpaceRepository) ReplaceExperts(
	ctx context.Context,
	tenantID uint64,
	serviceID, operatorUserID string,
	experts []*types.ServiceExpertBinding,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
			Delete(&types.ServiceExpertBinding{}).Error; err != nil {
			return err
		}
		for _, expert := range experts {
			expert.TenantID = tenantID
			expert.ServiceID = serviceID
			expert.CreatedBy = operatorUserID
			expert.UpdatedBy = operatorUserID
		}
		if len(experts) == 0 {
			return nil
		}
		return tx.Create(&experts).Error
	})
}

func (r *serviceSpaceRepository) CreateSession(ctx context.Context, session *types.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *serviceSpaceRepository) GetSession(
	ctx context.Context,
	tenantID uint64,
	serviceID, sessionID string,
) (*types.Session, error) {
	var session types.Session
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, sessionID).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &session, err
}

func (r *serviceSpaceRepository) ListSessions(
	ctx context.Context,
	tenantID uint64,
	serviceID, keyword string,
	page, pageSize int,
) ([]*types.Session, int64, error) {
	query := r.db.WithContext(ctx).Model(&types.Session{}).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID)
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		query = query.Where("LOWER(title) LIKE ?", "%"+strings.ToLower(keyword)+"%")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var sessions []*types.Session
	err := query.Order("is_pinned DESC").
		Order("pinned_at DESC").
		Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&sessions).Error
	return sessions, total, err
}

func (r *serviceSpaceRepository) UpdateSession(
	ctx context.Context,
	tenantID uint64,
	serviceID, sessionID string,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.Session{}).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, sessionID).
		Updates(fields).Error
}

func (r *serviceSpaceRepository) SetSessionPinned(
	ctx context.Context,
	tenantID uint64,
	serviceID, sessionID string,
	pinned bool,
) error {
	fields := map[string]any{"is_pinned": pinned, "updated_at": time.Now().UTC()}
	if pinned {
		fields["pinned_at"] = time.Now().UTC()
	} else {
		fields["pinned_at"] = nil
	}
	return r.UpdateSession(ctx, tenantID, serviceID, sessionID, fields)
}

func (r *serviceSpaceRepository) DeleteSession(
	ctx context.Context,
	tenantID uint64,
	serviceID, sessionID string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, sessionID).
		Delete(&types.Session{}).Error
}

func (r *serviceSpaceRepository) UpsertArtifacts(ctx context.Context, artifacts []*types.ServiceArtifact) error {
	if len(artifacts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, artifact := range artifacts {
			if err := tx.Model(&types.ServiceArtifact{}).
				Where("tenant_id = ? AND service_id = ? AND artifact_id = ? AND is_current = ?",
					artifact.TenantID, artifact.ServiceID, artifact.ArtifactID, true).
				Update("is_current", false).Error; err != nil {
				return err
			}
			var existing int64
			if err := tx.Model(&types.ServiceArtifact{}).
				Where("tenant_id = ? AND service_id = ? AND version_id = ?", artifact.TenantID, artifact.ServiceID, artifact.VersionID).
				Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				if err := tx.Create(artifact).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *serviceSpaceRepository) ListArtifacts(
	ctx context.Context,
	tenantID uint64,
	serviceID, lifecycle string,
	page, pageSize int,
) ([]*types.ServiceArtifact, int64, error) {
	query := r.db.WithContext(ctx).Model(&types.ServiceArtifact{}).
		Where("tenant_id = ? AND service_id = ? AND is_current = ?", tenantID, serviceID, true)
	if lifecycle = strings.TrimSpace(lifecycle); lifecycle != "" {
		query = query.Where("lifecycle = ?", lifecycle)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var artifacts []*types.ServiceArtifact
	err := query.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&artifacts).Error
	return artifacts, total, err
}

func (r *serviceSpaceRepository) GetArtifact(
	ctx context.Context,
	tenantID uint64,
	serviceID, artifactID string,
	version int,
) (*types.ServiceArtifact, error) {
	var artifact types.ServiceArtifact
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND artifact_id = ?", tenantID, serviceID, artifactID)
	if version > 0 {
		query = query.Where("version = ?", version)
	} else {
		query = query.Where("is_current = ?", true)
	}
	err := query.Order("version DESC").First(&artifact).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &artifact, err
}

func (r *serviceSpaceRepository) CountOverview(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) (sessions, artifacts, members int64, err error) {
	if err = r.db.WithContext(ctx).Model(&types.Session{}).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Count(&sessions).Error; err != nil {
		return
	}
	if err = r.db.WithContext(ctx).Model(&types.ServiceArtifact{}).
		Where("tenant_id = ? AND service_id = ? AND is_current = ?", tenantID, serviceID, true).
		Count(&artifacts).Error; err != nil {
		return
	}
	err = r.db.WithContext(ctx).Model(&types.ServiceSpaceMember{}).
		Where("tenant_id = ? AND service_id = ? AND status = ?", tenantID, serviceID, types.ServiceMemberStatusActive).
		Count(&members).Error
	return
}
