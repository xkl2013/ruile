package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/hibiken/asynq"
)

var (
	ErrAgentRunNotFound       = errors.New("agent run not found")
	ErrAgentRunCannotCancel   = errors.New("agent run can only be cancelled while queued")
	ErrAgentRunInvalidRequest = errors.New("invalid agent run request")
)

const (
	agentRunMaxRetry = 2
	agentRunTimeout  = 10 * time.Minute
)

type agentRunService struct {
	repo           interfaces.AgentRunRepository
	service        interfaces.ServiceService
	expertPackages interfaces.ExpertPackageRepository
	modelService   interfaces.ModelService
	taskClient     interfaces.TaskEnqueuer
}

func NewAgentRunService(
	repo interfaces.AgentRunRepository,
	service interfaces.ServiceService,
	expertPackages interfaces.ExpertPackageRepository,
	modelService interfaces.ModelService,
	taskClient interfaces.TaskEnqueuer,
) interfaces.AgentRunService {
	return &agentRunService{
		repo:           repo,
		service:        service,
		expertPackages: expertPackages,
		modelService:   modelService,
		taskClient:     taskClient,
	}
}

func (s *agentRunService) EnqueueMemoryExtraction(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID string,
) (*types.AgentRun, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	memoryID = strings.TrimSpace(memoryID)
	if memoryID == "" {
		return nil, ErrAgentRunInvalidRequest
	}
	input := types.JSONMap{"memory_id": memoryID}
	return s.enqueue(ctx, tenantID, userID, &types.AgentRun{
		RunType:      types.AgentRunTypeServiceMemoryExtract,
		AgentRef:     types.BuiltinServiceAssistantID,
		AgentVersion: types.BuiltinServiceAssistantVersion,
		TriggerType:  "memory",
		TriggerID:    memoryID,
		Input:        input,
	}, agentRunIdempotencyKey(tenantID, userID, types.AgentRunTypeServiceMemoryExtract, input))
}

func (s *agentRunService) EnqueueDailyReport(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ServiceDailyReportInput,
) (*types.AgentRun, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	input.Range = strings.TrimSpace(input.Range)
	input.Date = strings.TrimSpace(input.Date)
	input.Timezone = strings.TrimSpace(input.Timezone)
	input.Trigger = strings.TrimSpace(input.Trigger)
	runInput, err := agentRunJSONMap(input)
	if err != nil {
		return nil, fmt.Errorf("encode daily report request: %w", err)
	}
	return s.enqueue(ctx, tenantID, userID, &types.AgentRun{
		RunType:      types.AgentRunTypeServiceDailyReport,
		AgentRef:     types.BuiltinServiceAssistantID,
		AgentVersion: types.BuiltinServiceAssistantVersion,
		TriggerType:  firstNonEmpty(input.Trigger, "user_requested"),
		Input:        runInput,
	}, agentRunIdempotencyKey(tenantID, userID, types.AgentRunTypeServiceDailyReport, runInput))
}

func (s *agentRunService) enqueue(
	ctx context.Context,
	tenantID uint64,
	userID string,
	run *types.AgentRun,
	idempotencyKey string,
) (*types.AgentRun, error) {
	if existing, err := s.repo.FindActiveByIdempotency(ctx, tenantID, userID, run.RunType, idempotencyKey); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	run.TenantID = tenantID
	run.UserID = userID
	run.Status = types.AgentRunStatusQueued
	run.IdempotencyKey = idempotencyKey
	run.QueuedAt = time.Now().UTC()
	if err := s.repo.Create(ctx, run); err != nil {
		existing, lookupErr := s.repo.FindActiveByIdempotency(ctx, tenantID, userID, run.RunType, idempotencyKey)
		if lookupErr == nil && existing != nil {
			return existing, nil
		}
		return nil, err
	}

	payload, err := json.Marshal(types.AgentRunTaskPayload{RunID: run.ID, TenantID: tenantID})
	if err != nil {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorInvalidInput, "failed to encode agent run payload", time.Now())
		return nil, fmt.Errorf("encode agent run task payload: %w", err)
	}
	taskID := secutils.GenerateTaskID("agent_run", tenantID, run.ID)
	queue, ok := types.QueueForTaskType(types.TypeAgentRunExecute)
	if !ok {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorEnqueueFailed, "agent queue is not configured", time.Now())
		return nil, errors.New("agent queue is not configured")
	}
	if _, err := s.taskClient.Enqueue(
		asynq.NewTask(types.TypeAgentRunExecute, payload),
		asynq.TaskID(taskID),
		asynq.Queue(queue),
		asynq.MaxRetry(agentRunMaxRetry),
		asynq.Timeout(agentRunTimeout),
	); err != nil {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorEnqueueFailed, truncateAgentRunError(err), time.Now())
		return nil, fmt.Errorf("enqueue agent run: %w", err)
	}
	if err := s.repo.SetTaskID(ctx, run.ID, taskID); err != nil {
		return nil, fmt.Errorf("save agent run task id: %w", err)
	}
	run.TaskID = taskID
	return run, nil
}

func (s *agentRunService) GetAgentRun(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) (*types.AgentRun, error) {
	if err := validateServiceScope(tenantID, userID); err != nil {
		return nil, err
	}
	run, err := s.repo.GetByIDForUser(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrAgentRunNotFound
	}
	return run, nil
}

func (s *agentRunService) CancelAgentRun(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) (*types.AgentRun, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if run.Status == types.AgentRunStatusCancelled {
		return run, nil
	}
	if run.Status != types.AgentRunStatusQueued {
		return nil, ErrAgentRunCannotCancel
	}
	cancelled, err := s.repo.CancelQueued(ctx, tenantID, userID, run.ID, time.Now())
	if err != nil {
		return nil, err
	}
	if !cancelled {
		return nil, ErrAgentRunCannotCancel
	}
	return s.GetAgentRun(ctx, tenantID, userID, run.ID)
}

func (s *agentRunService) ProcessAgentRun(ctx context.Context, task *asynq.Task) error {
	var payload types.AgentRunTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode agent run task: %w", err)
	}
	if strings.TrimSpace(payload.RunID) == "" || payload.TenantID == 0 {
		return errors.New("agent run task missing scope")
	}
	run, err := s.repo.GetByID(ctx, payload.RunID)
	if err != nil {
		return err
	}
	if run == nil {
		return nil
	}
	if run.TenantID != payload.TenantID {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorInvalidInput, "agent run tenant mismatch", time.Now())
		return nil
	}
	claimed, err := s.repo.Claim(ctx, run.ID, time.Now())
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}

	result, compatibility, profileID, err := s.execute(ctx, run)
	if err != nil {
		return s.recordExecutionFailure(ctx, run.ID, err)
	}
	return s.completeAgentRun(ctx, run.ID, profileID, result, compatibility)
}

func (s *agentRunService) execute(
	ctx context.Context,
	run *types.AgentRun,
) (types.AgentResultV1, types.JSONMap, string, error) {
	switch run.RunType {
	case types.AgentRunTypeServiceDailyReport:
		var input types.ServiceDailyReportInput
		if err := decodeAgentRunInput(run.Input, &input); err != nil {
			return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: fmt.Errorf("decode daily report input: %w", err)}
		}
		report, err := s.service.GenerateDailyReport(ctx, run.TenantID, run.UserID, input)
		if err != nil {
			return types.AgentResultV1{}, nil, "", err
		}
		result, err := buildDailyReportAgentResult(report)
		if err != nil {
			return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
		}
		return result, types.JSONMap{
			"artifact_type":   "service_daily_report",
			"daily_report_id": report.ID,
		}, report.ProfileID, nil
	case types.AgentRunTypeServiceMemoryExtract:
		memoryID, _ := run.Input["memory_id"].(string)
		memoryID = strings.TrimSpace(memoryID)
		if memoryID == "" {
			return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: errors.New("memory_id is required")}
		}
		extraction, err := s.service.ExtractMemory(ctx, run.TenantID, run.UserID, memoryID)
		if err != nil {
			return types.AgentResultV1{}, nil, "", err
		}
		result, profileID, err := buildMemoryExtractionAgentResult(memoryID, extraction)
		if err != nil {
			return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
		}
		compatibility := types.JSONMap{
			"artifact_type": "service_memory_extraction",
			"memory_id":     extraction.MemoryID,
			"generated":     extraction.Generated,
			"reason":        extraction.Reason,
		}
		if extraction.Reminder != nil {
			compatibility["service_reminder_id"] = extraction.Reminder.ID
		}
		return result, compatibility, profileID, nil
	case types.AgentRunTypeExpertAgentTest:
		return s.executeExpertAgentTest(ctx, run)
	default:
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: fmt.Errorf("unsupported agent run type %q", run.RunType)}
	}
}

func (s *agentRunService) completeAgentRun(
	ctx context.Context,
	runID, profileID string,
	result types.AgentResultV1,
	compatibility types.JSONMap,
) error {
	validation := types.ValidateAgentResultV1(result)
	resultMap, err := result.ToJSONMap()
	if err != nil {
		return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: fmt.Errorf("encode agent result: %w", err)})
	}
	validationMap, err := agentRunJSONMap(validation)
	if err != nil {
		return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: fmt.Errorf("encode agent result validation: %w", err)})
	}
	resultMap["validation"] = validationMap
	for key, value := range compatibility {
		resultMap[key] = value
	}
	if !validation.Valid {
		message := "agent result validation failed"
		if len(validation.Errors) > 0 {
			message += ": " + strings.Join(validation.Errors, "; ")
		}
		_, err := s.repo.MarkFailedWithResult(
			ctx,
			runID,
			resultMap,
			types.AgentRunErrorInvalidOutput,
			truncateAgentRunError(errors.New(message)),
			time.Now(),
		)
		return err
	}
	_, err = s.repo.MarkSucceeded(ctx, runID, profileID, resultMap, time.Now())
	return err
}

func (s *agentRunService) recordExecutionFailure(ctx context.Context, runID string, err error) error {
	message := truncateAgentRunError(err)
	var invalidOutput invalidAgentRunOutputError
	if errors.As(err, &invalidOutput) {
		_, updateErr := s.repo.MarkFailed(ctx, runID, types.AgentRunErrorInvalidOutput, message, time.Now())
		return updateErr
	}
	var permanent permanentAgentRunError
	if errors.As(err, &permanent) || agentRunFinalAttempt(ctx) {
		if _, updateErr := s.repo.MarkFailed(ctx, runID, types.AgentRunErrorExecutionFailed, message, time.Now()); updateErr != nil {
			return updateErr
		}
		if errors.As(err, &permanent) {
			return nil
		}
		if _, inWorker := asynq.GetRetryCount(ctx); !inWorker {
			return nil
		}
		return err
	}
	if updateErr := s.repo.MarkRetry(ctx, runID, types.AgentRunErrorExecutionFailed, message); updateErr != nil {
		return updateErr
	}
	return err
}

func agentRunFinalAttempt(ctx context.Context) bool {
	retried, retriedOK := asynq.GetRetryCount(ctx)
	maxRetry, maxRetryOK := asynq.GetMaxRetry(ctx)
	if !retriedOK || !maxRetryOK {
		return true
	}
	return retried >= maxRetry
}

func decodeAgentRunInput(input types.JSONMap, destination any) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, destination)
}

func agentRunJSONMap(value any) (types.JSONMap, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result types.JSONMap
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func agentRunIdempotencyKey(tenantID uint64, userID, runType string, input types.JSONMap) string {
	raw, _ := json.Marshal(input)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s:%s", tenantID, userID, runType, raw)))
	return hex.EncodeToString(sum[:])
}

func truncateAgentRunError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 2000 {
		return message[:2000]
	}
	return message
}

type permanentAgentRunError struct {
	err error
}

func (e permanentAgentRunError) Error() string { return e.err.Error() }

func (e permanentAgentRunError) Unwrap() error { return e.err }
