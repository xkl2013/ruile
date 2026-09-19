package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
			[]string{types.AgentRunStatusQueued, types.AgentRunStatusRunning, types.AgentRunStatusWaitingInput},
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
			"status":      types.AgentRunStatusRunning,
			"interaction": types.JSONMap{},
			"started_at":  startedAt.UTC(),
			"finished_at": nil,
			"attempt":     gorm.Expr("attempt + ?", 1),
		})
	return result.RowsAffected == 1, result.Error
}

func (r *agentRunRepository) UpdatePhase(ctx context.Context, id, phase string) error {
	return r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStatusRunning).
		Update("phase", strings.TrimSpace(phase)).Error
}

func (r *agentRunRepository) SetRequirementSnapshot(ctx context.Context, id, snapshotID string) error {
	return r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ?", strings.TrimSpace(id)).
		Update("requirement_snapshot_id", strings.TrimSpace(snapshotID)).Error
}

func (r *agentRunRepository) MarkWaitingInput(
	ctx context.Context,
	id string,
	interaction types.JSONMap,
) (bool, error) {
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStatusRunning).
		Updates(map[string]any{
			"status":        types.AgentRunStatusWaitingInput,
			"phase":         types.AgentRunPhaseIntake,
			"interaction":   interaction,
			"error_code":    "",
			"error_message": "",
			"finished_at":   nil,
		})
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) ResumeWaiting(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
	input types.JSONMap,
	snapshotID string,
	resumedAt time.Time,
) (bool, error) {
	update := r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where(
			"tenant_id = ? AND user_id = ? AND id = ? AND status = ?",
			tenantID,
			strings.TrimSpace(userID),
			strings.TrimSpace(id),
			types.AgentRunStatusWaitingInput,
		).
		Updates(map[string]any{
			"status":                  types.AgentRunStatusQueued,
			"phase":                   types.AgentRunPhasePlanning,
			"input":                   input,
			"interaction":             types.JSONMap{},
			"requirement_snapshot_id": strings.TrimSpace(snapshotID),
			"resumed_at":              resumedAt.UTC(),
			"queued_at":               resumedAt.UTC(),
			"started_at":              nil,
			"finished_at":             nil,
			"task_id":                 "",
			"error_code":              "",
			"error_message":           "",
		})
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) SetQuality(ctx context.Context, id string, quality types.JSONMap) error {
	return r.db.WithContext(ctx).Model(&types.AgentRun{}).
		Where("id = ?", strings.TrimSpace(id)).
		Update("quality", quality).Error
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
		"phase":         types.AgentRunPhaseCompleted,
		"interaction":   types.JSONMap{},
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
			"interaction":   types.JSONMap{},
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
			"interaction":   types.JSONMap{},
			"result":        result,
			"error_code":    strings.TrimSpace(code),
			"error_message": strings.TrimSpace(message),
			"finished_at":   finishedAt.UTC(),
		})
	return update.RowsAffected == 1, update.Error
}

func (r *agentRunRepository) CreateEvent(ctx context.Context, event *types.AgentRunEvent) error {
	if event == nil {
		return errors.New("agent run event is required")
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		runID := strings.TrimSpace(event.RunID)
		if runID == "" {
			return errors.New("agent run event run_id is required")
		}
		sequence := &types.AgentRunEventSequence{RunID: runID}
		if err := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "run_id"}},
			DoNothing: true,
		}).Create(sequence).Error; err != nil {
			return err
		}

		if err := tx.Model(&types.AgentRunEventSequence{}).
			Where("run_id = ?", runID).
			UpdateColumn("next_sequence", gorm.Expr("next_sequence + ?", 1)).Error; err != nil {
			return err
		}
		if err := tx.Where("run_id = ?", runID).First(sequence).Error; err != nil {
			return err
		}
		event.RunID = runID
		event.Sequence = sequence.NextSequence
		return tx.Create(event).Error
	})
}

func (r *agentRunRepository) ListEvents(
	ctx context.Context,
	runID string,
	afterSequence int64,
	limit int,
) ([]*types.AgentRunEvent, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	var events []*types.AgentRunEvent
	err := r.db.WithContext(ctx).
		Where("run_id = ? AND sequence > ?", strings.TrimSpace(runID), afterSequence).
		Order("sequence ASC").
		Limit(limit).
		Find(&events).Error
	return events, err
}

func (r *agentRunRepository) CreateStep(ctx context.Context, step *types.AgentRunStep) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxSequence int
		if err := tx.Model(&types.AgentRunStep{}).
			Where("run_id = ?", strings.TrimSpace(step.RunID)).
			Select("COALESCE(MAX(sequence), 0)").
			Scan(&maxSequence).Error; err != nil {
			return err
		}
		step.Sequence = maxSequence + 1
		return tx.Create(step).Error
	})
}

func (r *agentRunRepository) CompleteStep(
	ctx context.Context,
	id string,
	output types.JSONMap,
	finishedAt time.Time,
) error {
	return r.db.WithContext(ctx).Model(&types.AgentRunStep{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStepStatusRunning).
		Updates(map[string]any{
			"status":      types.AgentRunStepStatusSucceeded,
			"output":      output,
			"error":       "",
			"finished_at": finishedAt.UTC(),
		}).Error
}

func (r *agentRunRepository) FailStep(
	ctx context.Context,
	id, message string,
	finishedAt time.Time,
) error {
	return r.db.WithContext(ctx).Model(&types.AgentRunStep{}).
		Where("id = ? AND status = ?", strings.TrimSpace(id), types.AgentRunStepStatusRunning).
		Updates(map[string]any{
			"status":      types.AgentRunStepStatusFailed,
			"error":       strings.TrimSpace(message),
			"finished_at": finishedAt.UTC(),
		}).Error
}

func (r *agentRunRepository) ListSteps(ctx context.Context, runID string) ([]*types.AgentRunStep, error) {
	var steps []*types.AgentRunStep
	err := r.db.WithContext(ctx).
		Where("run_id = ?", strings.TrimSpace(runID)).
		Order("sequence ASC").
		Find(&steps).Error
	return steps, err
}

func (r *agentRunRepository) CreateInputRevision(
	ctx context.Context,
	revision *types.AgentRunInputRevision,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxRevision int
		if err := tx.Model(&types.AgentRunInputRevision{}).
			Where("run_id = ?", strings.TrimSpace(revision.RunID)).
			Select("COALESCE(MAX(revision), 0)").
			Scan(&maxRevision).Error; err != nil {
			return err
		}
		revision.Revision = maxRevision + 1
		return tx.Create(revision).Error
	})
}

func (r *agentRunRepository) CreateRequirementSnapshot(
	ctx context.Context,
	snapshot *types.AgentRequirementSnapshot,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var maxRevision int
		if err := tx.Model(&types.AgentRequirementSnapshot{}).
			Where("run_id = ?", strings.TrimSpace(snapshot.RunID)).
			Select("COALESCE(MAX(revision), 0)").
			Scan(&maxRevision).Error; err != nil {
			return err
		}
		snapshot.Revision = maxRevision + 1
		return tx.Create(snapshot).Error
	})
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
