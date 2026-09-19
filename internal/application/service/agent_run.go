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
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

var (
	ErrAgentRunNotFound         = errors.New("agent run not found")
	ErrAgentRunCannotCancel     = errors.New("agent run can only be cancelled while queued")
	ErrAgentRunNotWaitingInput  = errors.New("agent run is not waiting for input")
	ErrAgentRunCannotRegenerate = errors.New("agent run can only be regenerated after it reaches a terminal state")
	ErrAgentRunInvalidRequest   = errors.New("invalid agent run request")
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
	source := "request"
	if run.TriggerType == "regenerate" {
		source = "regenerate"
	}
	if err := s.repo.CreateInputRevision(ctx, &types.AgentRunInputRevision{
		RunID:  run.ID,
		Source: source,
		Input:  cloneAgentRunJSONMap(run.Input),
	}); err != nil {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorInvalidInput, "failed to persist initial input revision", time.Now())
		return nil, fmt.Errorf("save initial agent run input: %w", err)
	}
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunQueued, types.JSONMap{
		"runId":    run.ID,
		"threadId": run.ID,
		"status":   run.Status,
		"runType":  run.RunType,
		"agentRef": run.AgentRef,
	})

	if err := s.enqueueRunTask(ctx, run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *agentRunService) enqueueRunTask(ctx context.Context, run *types.AgentRun) error {
	payload, err := json.Marshal(types.AgentRunTaskPayload{RunID: run.ID, TenantID: run.TenantID})
	if err != nil {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorInvalidInput, "failed to encode agent run payload", time.Now())
		return fmt.Errorf("encode agent run task payload: %w", err)
	}
	taskID := secutils.GenerateTaskID("agent_run", run.TenantID, run.ID)
	queue, ok := types.QueueForTaskType(types.TypeAgentRunExecute)
	if !ok {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorEnqueueFailed, "agent queue is not configured", time.Now())
		return errors.New("agent queue is not configured")
	}
	if _, err := s.taskClient.Enqueue(
		asynq.NewTask(types.TypeAgentRunExecute, payload),
		asynq.TaskID(taskID),
		asynq.Queue(queue),
		asynq.MaxRetry(agentRunMaxRetry),
		asynq.Timeout(agentRunTimeout),
	); err != nil {
		_, _ = s.repo.MarkFailed(ctx, run.ID, types.AgentRunErrorEnqueueFailed, truncateAgentRunError(err), time.Now())
		return fmt.Errorf("enqueue agent run: %w", err)
	}
	if err := s.repo.SetTaskID(ctx, run.ID, taskID); err != nil {
		return fmt.Errorf("save agent run task id: %w", err)
	}
	run.TaskID = taskID
	return nil
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

func (s *agentRunService) GetAgentRunQuality(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) (types.JSONMap, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	return cloneAgentRunJSONMap(run.Quality), nil
}

func (s *agentRunService) ListAgentRunSteps(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) ([]*types.AgentRunStep, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSteps(ctx, run.ID)
}

func (s *agentRunService) SubmitAgentRunAnswers(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
	answerInput types.AgentRunAnswersInput,
) (*types.AgentRun, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if run.Status != types.AgentRunStatusWaitingInput {
		return nil, ErrAgentRunNotWaitingInput
	}
	if len(answerInput.Answers) == 0 {
		return nil, ErrAgentRunInvalidRequest
	}
	questions, err := decodeExpertIntakeInteraction(run.Interaction)
	if err != nil {
		return nil, ErrAgentRunInvalidRequest
	}
	if err := validateExpertIntakeAnswers(questions, answerInput.Answers); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAgentRunInvalidRequest, err)
	}

	var expertInput types.ExpertAgentTestInput
	if err := decodeAgentRunInput(run.Input, &expertInput); err != nil {
		return nil, fmt.Errorf("decode waiting expert run: %w", err)
	}
	mergedAnswers := cloneAgentRunJSONMap(expertInput.Answers)
	for key, value := range answerInput.Answers {
		mergedAnswers[key] = value
	}
	expertInput.Answers = mergedAnswers
	nextInput, err := agentRunJSONMap(expertInput)
	if err != nil {
		return nil, fmt.Errorf("encode expert answers: %w", err)
	}
	if err := s.repo.CreateInputRevision(ctx, &types.AgentRunInputRevision{
		RunID:  run.ID,
		Source: "user_answers",
		Input:  cloneAgentRunJSONMap(nextInput),
	}); err != nil {
		return nil, fmt.Errorf("save expert answer revision: %w", err)
	}
	snapshot := &types.AgentRequirementSnapshot{
		RunID: run.ID,
		Values: types.JSONMap{
			"prompt":  expertInput.Prompt,
			"answers": cloneAgentRunJSONMap(mergedAnswers),
		},
		Assumptions: types.StringArray{},
		Missing:     types.StringArray{},
	}
	if err := s.repo.CreateRequirementSnapshot(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("save expert requirement snapshot: %w", err)
	}
	now := time.Now().UTC()
	resumed, err := s.repo.ResumeWaiting(
		ctx,
		tenantID,
		userID,
		run.ID,
		nextInput,
		snapshot.ID,
		now,
	)
	if err != nil {
		return nil, err
	}
	if !resumed {
		return nil, ErrAgentRunNotWaitingInput
	}
	run, err = s.repo.GetByID(ctx, run.ID)
	if err != nil {
		return nil, err
	}
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunResumed, types.JSONMap{
		"runId":    run.ID,
		"threadId": run.ID,
		"status":   types.AgentRunStatusQueued,
		"phase":    types.AgentRunPhasePlanning,
		"answers":  cloneAgentRunJSONMap(answerInput.Answers),
	})
	if err := s.enqueueRunTask(ctx, run); err != nil {
		return nil, err
	}
	return s.GetAgentRun(ctx, tenantID, userID, run.ID)
}

func (s *agentRunService) RegenerateAgentRun(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
	regenerateInput types.AgentRunRegenerateInput,
) (*types.AgentRun, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	switch run.Status {
	case types.AgentRunStatusSucceeded, types.AgentRunStatusFailed:
	default:
		return nil, ErrAgentRunCannotRegenerate
	}
	if run.RunType != types.AgentRunTypeExpertAgentTest {
		return nil, fmt.Errorf("%w: only expert runs are supported", ErrAgentRunInvalidRequest)
	}
	var input types.ExpertAgentTestInput
	if err := decodeAgentRunInput(run.Input, &input); err != nil {
		return nil, fmt.Errorf("decode expert run for regeneration: %w", err)
	}
	input.Feedback = strings.TrimSpace(regenerateInput.Feedback)
	if utf8.RuneCountInString(input.Feedback) > expertAgentTestMaxPromptRunes {
		return nil, ErrAgentRunInvalidRequest
	}
	nextInput, err := agentRunJSONMap(input)
	if err != nil {
		return nil, fmt.Errorf("encode expert regeneration request: %w", err)
	}
	child := &types.AgentRun{
		ParentRunID:  run.ID,
		RunType:      run.RunType,
		AgentRef:     run.AgentRef,
		AgentVersion: run.AgentVersion,
		ProfileID:    run.ProfileID,
		TriggerType:  "regenerate",
		TriggerID:    run.TriggerID,
		Input:        nextInput,
	}
	return s.enqueue(ctx, tenantID, userID, child, fmt.Sprintf(
		"regenerate:%s:%s",
		run.ID,
		uuid.NewString(),
	))
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
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunCancelled, types.JSONMap{
		"runId":    run.ID,
		"threadId": run.ID,
		"status":   types.AgentRunStatusCancelled,
	})
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
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunStarted, types.JSONMap{
		"runId":    run.ID,
		"threadId": run.ID,
		"status":   types.AgentRunStatusRunning,
		"phase":    run.Phase,
		"attempt":  run.Attempt + 1,
	})
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeReasoningStart, types.JSONMap{
		"runId": run.ID,
		"title": "Agent 执行过程",
	})
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeReasoningMessageStart, types.JSONMap{
		"runId":     run.ID,
		"messageId": "reasoning-" + run.ID,
	})

	result, compatibility, profileID, err := s.execute(ctx, run)
	if err != nil {
		var waiting agentRunWaitingInputError
		if errors.As(err, &waiting) {
			_, updateErr := s.repo.MarkWaitingInput(ctx, run.ID, waiting.interaction)
			if updateErr == nil {
				s.emitWaitingInputEvents(ctx, run.ID, waiting.interaction)
			}
			return updateErr
		}
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
		failed, err := s.repo.MarkFailedWithResult(
			ctx,
			runID,
			resultMap,
			types.AgentRunErrorInvalidOutput,
			truncateAgentRunError(errors.New(message)),
			time.Now(),
		)
		if err == nil && failed {
			s.emitRunError(ctx, runID, types.AgentRunErrorInvalidOutput, message)
		}
		return err
	}
	succeeded, err := s.repo.MarkSucceeded(ctx, runID, profileID, resultMap, time.Now())
	if err == nil && succeeded {
		s.emitRunCompletionEvents(ctx, runID, profileID, result)
	}
	return err
}

func (s *agentRunService) recordExecutionFailure(ctx context.Context, runID string, err error) error {
	message := truncateAgentRunError(err)
	var invalidOutput invalidAgentRunOutputError
	if errors.As(err, &invalidOutput) {
		failed, updateErr := s.repo.MarkFailed(ctx, runID, types.AgentRunErrorInvalidOutput, message, time.Now())
		if updateErr == nil && failed {
			s.emitRunError(ctx, runID, types.AgentRunErrorInvalidOutput, message)
		}
		return updateErr
	}
	var permanent permanentAgentRunError
	if errors.As(err, &permanent) || agentRunFinalAttempt(ctx) {
		failed, updateErr := s.repo.MarkFailed(ctx, runID, types.AgentRunErrorExecutionFailed, message, time.Now())
		if updateErr != nil {
			return updateErr
		}
		if failed {
			s.emitRunError(ctx, runID, types.AgentRunErrorExecutionFailed, message)
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
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeRunError, types.JSONMap{
		"runId":     runID,
		"status":    types.AgentRunStatusQueued,
		"errorCode": types.AgentRunErrorExecutionFailed,
		"message":   message,
		"retrying":  true,
	})
	return err
}

func (s *agentRunService) ListAgentRunEvents(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
	afterSequence int64,
	limit int,
) ([]*types.AgentRunEvent, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	return s.repo.ListEvents(ctx, run.ID, afterSequence, limit)
}

func (s *agentRunService) emitRunEvent(
	ctx context.Context,
	runID, eventType string,
	payload types.JSONMap,
) {
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(eventType) == "" {
		return
	}
	if err := s.repo.CreateEvent(ctx, &types.AgentRunEvent{
		RunID:     runID,
		EventType: eventType,
		Payload:   cloneAgentRunJSONMap(payload),
	}); err != nil {
		// Event persistence must not turn a completed agent result into a
		// failed task. The run state remains authoritative and the error is
		// visible in server logs for operational repair.
		logger.Warnf(ctx, "failed to persist agent run event (run=%s type=%s): %v", runID, eventType, err)
	}
}

func (s *agentRunService) emitRunError(ctx context.Context, runID, code, message string) {
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeRunError, types.JSONMap{
		"runId":     runID,
		"status":    types.AgentRunStatusFailed,
		"errorCode": code,
		"message":   message,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningMessageEnd, types.JSONMap{
		"runId":     runID,
		"messageId": "reasoning-" + runID,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningEnd, types.JSONMap{
		"runId": runID,
	})
}

func (s *agentRunService) emitWaitingInputEvents(
	ctx context.Context,
	runID string,
	interaction types.JSONMap,
) {
	toolCallID := "ask_user_" + runID
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeToolCallStart, types.JSONMap{
		"runId":           runID,
		"toolCallId":      toolCallID,
		"toolCallName":    "ask_user",
		"parentMessageId": runID,
	})
	raw, err := json.Marshal(interaction)
	if err == nil {
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeToolCallArgs, types.JSONMap{
			"runId":      runID,
			"toolCallId": toolCallID,
			"delta":      string(raw),
		})
	}
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeRunWaitingInput, types.JSONMap{
		"runId":       runID,
		"threadId":    runID,
		"status":      types.AgentRunStatusWaitingInput,
		"toolCallId":  toolCallID,
		"interaction": cloneAgentRunJSONMap(interaction),
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningMessageEnd, types.JSONMap{
		"runId":     runID,
		"messageId": "reasoning-" + runID,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningEnd, types.JSONMap{
		"runId": runID,
	})
}

func (s *agentRunService) emitRunCompletionEvents(
	ctx context.Context,
	runID, profileID string,
	result types.AgentResultV1,
) {
	summary := strings.TrimSpace(result.Decision.Reason)
	if result.Card != nil {
		summary = firstNonEmpty(strings.TrimSpace(result.Card.Summary), summary)
	}
	if summary != "" {
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeTextMessageStart, types.JSONMap{
			"runId":     runID,
			"messageId": runID,
			"role":      "assistant",
		})
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeTextMessageDelta, types.JSONMap{
			"runId":     runID,
			"messageId": runID,
			"delta":     summary,
		})
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeTextMessageEnd, types.JSONMap{
			"runId":     runID,
			"messageId": runID,
		})
	}
	files := make([]map[string]any, 0, len(result.Artifacts))
	for _, artifact := range result.Artifacts {
		files = append(files, map[string]any{
			"name":   artifact.Title,
			"kind":   artifact.Kind,
			"format": artifact.Format,
		})
	}
	if len(files) > 0 {
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeActivitySnapshot, types.JSONMap{
			"runId":        runID,
			"messageId":    runID,
			"activityType": "artifacts",
			"replace":      true,
			"content": types.JSONMap{
				"title": "本轮产物",
				"files": files,
			},
		})
	}
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningMessageEnd, types.JSONMap{
		"runId":     runID,
		"messageId": "reasoning-" + runID,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningEnd, types.JSONMap{
		"runId": runID,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeRunFinished, types.JSONMap{
		"runId":     runID,
		"threadId":  runID,
		"status":    types.AgentRunStatusSucceeded,
		"profileId": profileID,
		"phase":     types.AgentRunPhaseCompleted,
	})
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

func cloneAgentRunJSONMap(value types.JSONMap) types.JSONMap {
	if value == nil {
		return types.JSONMap{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return types.JSONMap{}
	}
	var cloned types.JSONMap
	if json.Unmarshal(raw, &cloned) != nil {
		return types.JSONMap{}
	}
	return cloned
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

type agentRunWaitingInputError struct {
	interaction types.JSONMap
}

func (e agentRunWaitingInputError) Error() string { return "agent run is waiting for user input" }
