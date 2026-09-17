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

type agentRunRepository struct {
	db *gorm.DB
}

func NewAgentRunRepository(db *gorm.DB) interfaces.AgentRunRepository {
	return &agentRunRepository{db: db}
}

func (r *agentRunRepository) Create(ctx context.Context, run *types.AgentRun) error {
	return r.db.WithContext(ctx).Create(run).Error
}

func (r *agentRunRepository) GetByID(ctx context.Context, id string) (*types.AgentRun, error) {
	var run types.AgentRun
	err := r.db.WithContext(ctx).Where("id = ?", strings.TrimSpace(id)).First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}

func (r *agentRunRepository) GetByIDForUser(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentRun, error) {
	var run types.AgentRun
	err := r.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND id = ?", tenantID, strings.TrimSpace(userID), strings.TrimSpace(id)).
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}

func (r *agentRunRepository) FindActiveByIdempotency(
	ctx context.Context,
	tenantID uint64,
	userID, runType, key string,
) (*types.AgentRun, error) {
	var run types.AgentRun
	err := r.db.WithContext(ctx).
		Where(
			"tenant_id = ? AND user_id = ? AND run_type = ? AND idempotency_key = ? AND status IN ?",
			tenantID,
			strings.TrimSpace(userID),
			strings.TrimSpace(runType),
			strings.TrimSpace(key),
			[]string{types.AgentRunStatusQueued, types.AgentRunStatusRunning},
		).
		Order("created_at DESC").
		First(&run).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &run, err
}

func (r *agentRunRepository) SetTaskID(ctx context.Context, id, taskID string) error {
	return r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ?", strings.TrimSpace(id)).
		Update("task_id", strings.TrimSpace(taskID)).Error
}

func (r *agentRunRepository) Claim(ctx context.Context, id string, startedAt time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStatusQueued).
		Updates(map[string]any{
			"status":     types.AgentRunStatusRunning,
			"started_at": startedAt.UTC(),
			"attempt":    gorm.Expr("attempt + ?", 1),
		})
	return result.RowsAffected == 1, result.Error
}

func (r *agentRunRepository) MarkRetry(ctx context.Context, id, code, message string) error {
	return r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStatusRunning).
		Updates(map[string]any{
			"status":        types.AgentRunStatusQueued,
			"error_code":    strings.TrimSpace(code),
			"error_message": strings.TrimSpace(message),
		}).Error
}

func (r *agentRunRepository) MarkSucceeded(
	ctx context.Context,
	id string,
	profileID string,
	result types.JSONMap,
	finishedAt time.Time,
) (bool, error) {
	updates := map[string]any{
		"status":        types.AgentRunStatusSucceeded,
		"result":        result,
		"error_code":    "",
		"error_message": "",
		"finished_at":   finishedAt.UTC(),
	}
	if strings.TrimSpace(profileID) != "" {
		updates["profile_id"] = strings.TrimSpace(profileID)
	}
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStatusRunning).
		Updates(updates)
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) MarkFailed(
	ctx context.Context,
	id, code, message string,
	finishedAt time.Time,
) (bool, error) {
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status IN ?", strings.TrimSpace(id), []string{
			types.AgentRunStatusQueued,
			types.AgentRunStatusRunning,
		}).
		Updates(map[string]any{
			"status":        types.AgentRunStatusFailed,
			"error_code":    strings.TrimSpace(code),
			"error_message": strings.TrimSpace(message),
			"finished_at":   finishedAt.UTC(),
		})
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) MarkFailedWithResult(
	ctx context.Context,
	id string,
	result types.JSONMap,
	code, message string,
	finishedAt time.Time,
) (bool, error) {
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status IN ?", strings.TrimSpace(id), []string{
			types.AgentRunStatusQueued,
			types.AgentRunStatusRunning,
		}).
		Updates(map[string]any{
			"status":        types.AgentRunStatusFailed,
			"result":        result,
			"error_code":    strings.TrimSpace(code),
			"error_message": strings.TrimSpace(message),
			"finished_at":   finishedAt.UTC(),
		})
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) CancelQueued(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
	finishedAt time.Time,
) (bool, error) {
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where(
			"tenant_id = ? AND user_id = ? AND id = ? AND status = ?",
			tenantID,
			strings.TrimSpace(userID),
			strings.TrimSpace(id),
			types.AgentRunStatusQueued,
		).
		Updates(map[string]any{
			"status":      types.AgentRunStatusCancelled,
			"finished_at": finishedAt.UTC(),
		})
	return update.RowsAffected == 1, update.Error
}
