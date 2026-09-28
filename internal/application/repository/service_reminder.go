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

func (r *serviceSpaceRepository) ListReminders(
	ctx context.Context,
	tenantID uint64,
	serviceID, status string,
	page, pageSize int,
) ([]*types.ServiceReminder, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&types.ServiceReminder{}).
		Where("tenant_id = ? AND service_id = ? AND deleted_at IS NULL", tenantID, strings.TrimSpace(serviceID))
	if status = strings.TrimSpace(status); status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var reminders []*types.ServiceReminder
	err := query.
		Order("CASE WHEN due_at IS NULL THEN 1 ELSE 0 END ASC").
		Order("due_at ASC").
		Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&reminders).Error
	return reminders, total, err
}

func (r *serviceSpaceRepository) GetReminder(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) (*types.ServiceReminder, error) {
	var reminder types.ServiceReminder
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		First(&reminder).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &reminder, err
}

func (r *serviceSpaceRepository) CreateReminder(
	ctx context.Context,
	reminder *types.ServiceReminder,
) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

func (r *serviceSpaceRepository) UpdateReminder(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
	fields map[string]any,
) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now().UTC()
	return r.db.WithContext(ctx).
		Model(&types.ServiceReminder{}).
		Where("tenant_id = ? AND service_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		Updates(fields).Error
}

func (r *serviceSpaceRepository) DeleteReminder(
	ctx context.Context,
	tenantID uint64,
	serviceID, reminderID string,
) error {
	return r.db.WithContext(ctx).
		Where("tenant_id = ? AND service_id = ? AND id = ? AND deleted_at IS NULL",
			tenantID, strings.TrimSpace(serviceID), strings.TrimSpace(reminderID)).
		Delete(&types.ServiceReminder{}).Error
}

var _ interfaces.ServiceSpaceRepository = (*serviceSpaceRepository)(nil)
