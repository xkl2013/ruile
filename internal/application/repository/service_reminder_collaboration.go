package repository

import (
	"context"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
)

func (r *serviceSpaceRepository) ListReminderAssignees(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) ([]*types.ServiceReminderAssignee, error) {
	var rows []*types.ServiceReminderAssignee
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND reminder_id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		Order("CASE role WHEN 'primary' THEN 1 ELSE 2 END").
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *serviceSpaceRepository) ReplaceReminderAssignees(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID, assignedBy string,
	userIDs []string,
) error {
	normalized := make([]string, 0, len(userIDs))
	seen := make(map[string]struct{}, len(userIDs))
	for _, userID := range userIDs {
		userID = strings.TrimSpace(userID)
		if userID == "" {
			continue
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		normalized = append(normalized, userID)
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		if err := tx.Model(&types.ServiceReminderAssignee{}).
			Where("tenant_id = ? AND service_id = ? AND reminder_id = ? AND deleted_at IS NULL",
				tenantID, serviceID, reminderID).
			Updates(map[string]any{"deleted_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		if len(normalized) == 0 {
			return nil
		}
		rows := make([]*types.ServiceReminderAssignee, 0, len(normalized))
		for index, userID := range normalized {
			role := types.ServiceReminderAssigneeRoleMember
			if index == 0 {
				role = types.ServiceReminderAssigneeRolePrimary
			}
			rows = append(rows, &types.ServiceReminderAssignee{
				TenantID: tenantID, ServiceID: serviceID, ReminderID: reminderID,
				UserID: userID, Role: role, AssignedBy: strings.TrimSpace(assignedBy),
			})
		}
		return tx.Create(&rows).Error
	})
}

func (r *serviceSpaceRepository) ListReminderComments(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) ([]*types.ServiceReminderComment, error) {
	var rows []*types.ServiceReminderComment
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND reminder_id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		Order("created_at ASC").
		Find(&rows).Error
	return rows, err
}

func (r *serviceSpaceRepository) CreateReminderComment(
	ctx context.Context,
	comment *types.ServiceReminderComment,
) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

func (r *serviceSpaceRepository) DeleteReminderComment(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID, commentID string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND reminder_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID), strings.TrimSpace(commentID)).
		Delete(&types.ServiceReminderComment{}).Error
}

func (r *serviceSpaceRepository) ListReminderHistory(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) ([]*types.ServiceReminderHistory, error) {
	var rows []*types.ServiceReminderHistory
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND reminder_id = ?",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *serviceSpaceRepository) CreateReminderHistory(
	ctx context.Context,
	history *types.ServiceReminderHistory,
) error {
	return r.db.WithContext(ctx).Create(history).Error
}

func (r *serviceSpaceRepository) ensureReminderExists(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) error {
	var count int64
	err := r.db.WithContext(ctx).Model(&types.ServiceReminder{}).
		Where("tenant_id = ? AND service_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, serviceID, reminderID).
		Count(&count).Error
	if err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

var _ interfaces.ServiceSpaceRepository = (*serviceSpaceRepository)(nil)
