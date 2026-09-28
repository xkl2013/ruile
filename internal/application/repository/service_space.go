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
		return seedDefaultReminderStatusMachine(tx, service.TenantID, service.ID, service.CreatedBy)
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

func (r *serviceSpaceRepository) ListSubjects(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectType string,
	page, pageSize int,
) ([]*types.ServiceSubject, int64, error) {
	query := r.db.WithContext(ctx).Model(&types.ServiceSubject{}).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID)
	if subjectType = strings.TrimSpace(subjectType); subjectType != "" {
		query = query.Where("subject_type = ?", subjectType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var subjects []*types.ServiceSubject
	err := query.Order("updated_at DESC").
		Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&subjects).Error
	return subjects, total, err
}

func (r *serviceSpaceRepository) GetSubject(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectID string,
) (*types.ServiceSubject, error) {
	var subject types.ServiceSubject
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, subjectID).
		First(&subject).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &subject, err
}

func (r *serviceSpaceRepository) CreateSubject(
	ctx context.Context,
	subject *types.ServiceSubject,
) error {
	return r.db.WithContext(ctx).Create(subject).Error
}

func (r *serviceSpaceRepository) UpdateSubject(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectID string,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).Model(&types.ServiceSubject{}).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, subjectID).
		Updates(fields).Error
}

func (r *serviceSpaceRepository) DeleteSubject(
	ctx context.Context,
	tenantID uint64,
	serviceID, subjectID string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, subjectID).
		Delete(&types.ServiceSubject{}).Error
}

func (r *serviceSpaceRepository) ListReminderStatuses(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
	includeDisabled bool,
) ([]*types.ServiceReminderStatus, error) {
	query := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID)
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	var statuses []*types.ServiceReminderStatus
	err := query.Order("display_order ASC").Order("created_at ASC").Find(&statuses).Error
	return statuses, err
}

func (r *serviceSpaceRepository) GetReminderStatus(
	ctx context.Context,
	tenantID uint64,
	serviceID, statusID string,
) (*types.ServiceReminderStatus, error) {
	var status types.ServiceReminderStatus
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, statusID).
		First(&status).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &status, err
}

func (r *serviceSpaceRepository) CreateReminderStatus(
	ctx context.Context,
	status *types.ServiceReminderStatus,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if status.IsInitial {
			if err := tx.Model(&types.ServiceReminderStatus{}).
				Where("tenant_id = ? AND service_id = ? AND deleted_at IS NULL", status.TenantID, status.ServiceID).
				Update("is_initial", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(status).Error
	})
}

func (r *serviceSpaceRepository) UpdateReminderStatus(
	ctx context.Context,
	tenantID uint64,
	serviceID, statusID string,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if initial, ok := fields["is_initial"].(bool); ok && initial {
			if err := tx.Model(&types.ServiceReminderStatus{}).
				Where("tenant_id = ? AND service_id = ? AND id <> ? AND deleted_at IS NULL", tenantID, serviceID, statusID).
				Update("is_initial", false).Error; err != nil {
				return err
			}
		}
		fields["updated_at"] = time.Now().UTC()
		return tx.Model(&types.ServiceReminderStatus{}).
			Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, statusID).
			Updates(fields).Error
	})
}

func (r *serviceSpaceRepository) DeleteReminderStatus(
	ctx context.Context,
	tenantID uint64,
	serviceID, statusID string,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var status types.ServiceReminderStatus
		if err := tx.Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, statusID).
			First(&status).Error; err != nil {
			return err
		}
		if status.IsInitial {
			var next types.ServiceReminderStatus
			err := tx.Where("tenant_id = ? AND service_id = ? AND id <> ? AND enabled = ? AND deleted_at IS NULL",
				tenantID, serviceID, statusID, true).
				Order("display_order ASC").Order("created_at ASC").First(&next).Error
			if err != nil {
				return err
			}
			if err := tx.Model(&types.ServiceReminderStatus{}).Where("id = ?", next.ID).
				Update("is_initial", true).Error; err != nil {
				return err
			}
		}
		return tx.Where("tenant_id = ? AND service_id = ? AND id = ?", tenantID, serviceID, statusID).
			Delete(&types.ServiceReminderStatus{}).Error
	})
}

func (r *serviceSpaceRepository) ListReminderStatusTransitions(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
) ([]*types.ServiceReminderStatusTransition, error) {
	var transitions []*types.ServiceReminderStatusTransition
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
		Order("created_at ASC").
		Find(&transitions).Error
	return transitions, err
}

func (r *serviceSpaceRepository) ReplaceReminderStatusTransitions(
	ctx context.Context,
	tenantID uint64,
	serviceID string,
	transitions []*types.ServiceReminderStatusTransition,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tenant_id = ? AND service_id = ?", tenantID, serviceID).
			Delete(&types.ServiceReminderStatusTransition{}).Error; err != nil {
			return err
		}
		if len(transitions) == 0 {
			return nil
		}
		return tx.Create(&transitions).Error
	})
}

func (r *serviceSpaceRepository) CreateSession(ctx context.Context, session *types.Session) error {
	return r.db.WithContext(ctx).Create(session).Error
}

type defaultReminderStatusDefinition struct {
	key          string
	label        string
	category     string
	initial      bool
	terminal     bool
	displayOrder int
}

func seedDefaultReminderStatusMachine(
	tx *gorm.DB,
	tenantID uint64,
	serviceID, createdBy string,
) error {
	definitions := []defaultReminderStatusDefinition{
		{key: "candidate", label: "待识别", category: types.ServiceReminderStatusCategoryOpen, initial: true, displayOrder: 10},
		{key: "pending", label: "待处理", category: types.ServiceReminderStatusCategoryOpen, displayOrder: 20},
		{key: "generated", label: "已生成", category: types.ServiceReminderStatusCategoryInProgress, displayOrder: 30},
		{key: "confirmed", label: "已确认", category: types.ServiceReminderStatusCategoryInProgress, displayOrder: 40},
		{key: "completed", label: "已完成", category: types.ServiceReminderStatusCategoryDone, terminal: true, displayOrder: 50},
		{key: "ignored", label: "已忽略", category: types.ServiceReminderStatusCategoryDismissed, terminal: true, displayOrder: 60},
		{key: "snoozed", label: "已延后", category: types.ServiceReminderStatusCategoryInProgress, displayOrder: 70},
		{key: "stale", label: "已过期", category: types.ServiceReminderStatusCategoryOpen, displayOrder: 80},
		{key: "recompute_required", label: "待重新计算", category: types.ServiceReminderStatusCategoryOpen, displayOrder: 90},
	}
	statuses := make(map[string]*types.ServiceReminderStatus, len(definitions))
	for _, definition := range definitions {
		status := &types.ServiceReminderStatus{
			TenantID:     tenantID,
			ServiceID:    serviceID,
			StatusKey:    definition.key,
			Label:        definition.label,
			Category:     definition.category,
			IsInitial:    definition.initial,
			IsTerminal:   definition.terminal,
			DisplayOrder: definition.displayOrder,
			IsSystem:     true,
			Enabled:      true,
			CreatedBy:    createdBy,
		}
		if err := tx.Create(status).Error; err != nil {
			return err
		}
		statuses[definition.key] = status
	}
	edges := [][2]string{
		{"candidate", "pending"},
		{"candidate", "ignored"},
		{"pending", "generated"},
		{"pending", "ignored"},
		{"pending", "snoozed"},
		{"generated", "confirmed"},
		{"generated", "ignored"},
		{"confirmed", "completed"},
		{"confirmed", "ignored"},
		{"snoozed", "pending"},
		{"stale", "pending"},
		{"recompute_required", "pending"},
	}
	transitions := make([]*types.ServiceReminderStatusTransition, 0, len(edges))
	for _, edge := range edges {
		transitions = append(transitions, &types.ServiceReminderStatusTransition{
			TenantID:     tenantID,
			ServiceID:    serviceID,
			FromStatusID: statuses[edge[0]].ID,
			ToStatusID:   statuses[edge[1]].ID,
			AllowedRoles: types.StringArray{},
			Enabled:      true,
		})
	}
	return tx.Create(&transitions).Error
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
