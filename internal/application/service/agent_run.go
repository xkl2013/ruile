package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
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
	ErrAgentRunCannotCancel     = errors.New("agent run is already finished and cannot be cancelled")
	ErrAgentRunNotWaitingInput  = errors.New("agent run is not waiting for input")
	ErrAgentRunCannotRegenerate = errors.New("agent run can only be regenerated after it reaches a terminal state")
	ErrAgentRunInvalidRequest   = errors.New("invalid agent run request")
	ErrAgentRunNoExpertMatch    = errors.New("no published expert matched the request")
	ErrAgentRunRouteConfirm     = errors.New("expert route requires confirmation")
	ErrAgentRunRouteModel       = errors.New("expert route model failed")
	ErrAgentRunInvalidScope     = errors.New("invalid agent run scope")
)

const (
	agentRunMaxRetry = 2
	agentRunTimeout  = 10 * time.Minute
)

type agentRunService struct {
	repo            interfaces.AgentRunRepository
	serviceSpace    interfaces.ServiceSpaceService
	expertPackages  interfaces.ExpertPackageRepository
	modelService    interfaces.ModelService
	taskClient      interfaces.TaskEnqueuer
	taskCanceller   interfaces.TaskCanceller
	fileService     interfaces.FileService
	resourceCatalog interfaces.ResourceCatalog
	threadIDCache   sync.Map
}

func NewAgentRunService(
	repo interfaces.AgentRunRepository,
	serviceSpace interfaces.ServiceSpaceService,
	expertPackages interfaces.ExpertPackageRepository,
	modelService interfaces.ModelService,
	taskClient interfaces.TaskEnqueuer,
	taskCanceller interfaces.TaskCanceller,
	fileService interfaces.FileService,
	resourceCatalog interfaces.ResourceCatalog,
) interfaces.AgentRunService {
	return &agentRunService{
		repo:            repo,
		serviceSpace:    serviceSpace,
		expertPackages:  expertPackages,
		modelService:    modelService,
		taskClient:      taskClient,
		taskCanceller:   taskCanceller,
		fileService:     fileService,
		resourceCatalog: resourceCatalog,
	}
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
	s.cacheAgentRunThread(run)
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
		"threadId": firstNonEmpty(run.ThreadID, run.ID),
		"status":   run.Status,
		"runType":  run.RunType,
		"agentRef": run.AgentRef,
	})
	if route, ok := run.Input["routing_decision"].(map[string]any); ok && len(route) > 0 {
		payload := cloneAgentRunJSONMap(route)
		payload["runId"] = run.ID
		payload["threadId"] = firstNonEmpty(run.ThreadID, run.ID)
		s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeExpertRouting, payload)
	}

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
	if err := validateAgentRunScope(tenantID, userID); err != nil {
		return nil, err
	}
	run, err := s.repo.GetByIDForUser(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrAgentRunNotFound
	}
	if strings.TrimSpace(run.ServiceID) != "" {
		return nil, ErrAgentRunNotFound
	}
	run, err = s.recoverTimedOutAgentRun(ctx, run)
	if err != nil {
		return nil, err
	}
	return run, nil
}

func (s *agentRunService) GetAgentRunForService(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, id string,
) (*types.AgentRun, error) {
	if err := validateAgentRunScope(tenantID, userID); err != nil {
		return nil, err
	}
	serviceID = strings.TrimSpace(serviceID)
	if serviceID == "" || s.serviceSpace == nil {
		return nil, ErrAgentRunInvalidRequest
	}
	if _, err := s.serviceSpace.Authorize(
		ctx,
		tenantID,
		userID,
		serviceID,
		types.ServiceMemberRoleViewer,
		false,
	); err != nil {
		return nil, err
	}
	run, err := s.repo.GetByIDForService(ctx, tenantID, serviceID, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrAgentRunNotFound
	}
	return s.recoverTimedOutAgentRun(ctx, run)
}

func validateAgentRunScope(tenantID uint64, userID string) error {
	if tenantID == 0 || strings.TrimSpace(userID) == "" {
		return ErrAgentRunInvalidScope
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func (s *agentRunService) recoverTimedOutAgentRun(
	ctx context.Context,
	run *types.AgentRun,
) (*types.AgentRun, error) {
	if run == nil || run.Status != types.AgentRunStatusRunning || run.StartedAt == nil {
		return run, nil
	}
	now := time.Now().UTC()
	if run.StartedAt.After(now.Add(-agentRunTimeout)) {
		return run, nil
	}
	timedOut, err := s.repo.MarkTimedOut(ctx, run.ID, now.Add(-agentRunTimeout), now)
	if err != nil {
		return nil, err
	}
	if !timedOut {
		return s.repo.GetByID(ctx, run.ID)
	}
	message := "Agent 运行超过服务端执行时限，已自动终止。"
	s.emitRunFailure(ctx, run.ID, types.AgentRunErrorTimedOut, message, types.AgentRunStatusFailed, true, false)
	s.forgetAgentRunThread(run.ID)
	return s.repo.GetByID(ctx, run.ID)
}

func (s *agentRunService) ListAgentThreadRuns(
	ctx context.Context,
	tenantID uint64,
	userID, id string,
) ([]*types.AgentRun, error) {
	run, err := s.GetAgentRun(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	threadID := firstNonEmpty(strings.TrimSpace(run.ThreadID), run.ID)
	return s.repo.ListByThreadForUser(ctx, tenantID, userID, threadID)
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

func (s *agentRunService) ListAgentRunStepsForService(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, id string,
) ([]*types.AgentRunStep, error) {
	run, err := s.GetAgentRunForService(ctx, tenantID, userID, serviceID, id)
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
		"runId":   run.ID,
		"status":  types.AgentRunStatusQueued,
		"phase":   types.AgentRunPhasePlanning,
		"answers": cloneAgentRunJSONMap(answerInput.Answers),
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
		ServiceID:    run.ServiceID,
		ThreadID:     firstNonEmpty(run.ThreadID, run.ID),
		ParentRunID:  run.ID,
		RunType:      run.RunType,
		AgentRef:     run.AgentRef,
		AgentVersion: run.AgentVersion,
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
	if run.Status != types.AgentRunStatusQueued &&
		run.Status != types.AgentRunStatusRunning &&
		run.Status != types.AgentRunStatusWaitingInput {
		return nil, ErrAgentRunCannotCancel
	}
	cancelled, err := s.repo.CancelActive(ctx, tenantID, userID, run.ID, time.Now())
	if err != nil {
		return nil, err
	}
	if !cancelled {
		return nil, ErrAgentRunCannotCancel
	}
	if s.taskCanceller != nil && strings.TrimSpace(run.TaskID) != "" {
		queue, _ := types.QueueForTaskType(types.TypeAgentRunExecute)
		if _, cancelErr := s.taskCanceller.CancelTask(ctx, queue, run.TaskID); cancelErr != nil {
			logger.Warnf(ctx, "failed to cancel agent task (run=%s task=%s): %v", run.ID, run.TaskID, cancelErr)
		}
	}
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunCancelled, types.JSONMap{
		"runId":    run.ID,
		"threadId": firstNonEmpty(run.ThreadID, run.ID),
		"status":   types.AgentRunStatusCancelled,
	})
	s.forgetAgentRunThread(run.ID)
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
	startedAt := time.Now().UTC()
	claimed, err := s.repo.Claim(ctx, run.ID, startedAt)
	if err != nil {
		return err
	}
	if !claimed {
		return nil
	}
	run.Status = types.AgentRunStatusRunning
	run.StartedAt = &startedAt
	run.Attempt++
	s.cacheAgentRunThread(run)
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeRunStarted, types.JSONMap{
		"runId":           run.ID,
		"threadId":        firstNonEmpty(run.ThreadID, run.ID),
		"status":          types.AgentRunStatusRunning,
		"phase":           run.Phase,
		"attempt":         run.Attempt,
		"queuedAt":        run.QueuedAt,
		"startedAt":       startedAt,
		"queueDurationMs": elapsedMilliseconds(run.QueuedAt, startedAt),
	})
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeReasoningStart, types.JSONMap{
		"runId": run.ID,
		"title": "Agent 执行过程",
	})
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeReasoningMessageStart, types.JSONMap{
		"runId":     run.ID,
		"messageId": "reasoning-" + run.ID,
	})

	result, compatibility, _, err := s.execute(ctx, run)
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
	return s.completeAgentRun(ctx, run, result, compatibility)
}

func (s *agentRunService) execute(
	ctx context.Context,
	run *types.AgentRun,
) (types.AgentResultV1, types.JSONMap, string, error) {
	switch run.RunType {
	case types.AgentRunTypeExpertAgentTest:
		return s.executeExpertAgentTest(ctx, run)
	case types.AgentRunTypeExpertFollowUp:
		return s.executeExpertFollowUp(ctx, run)
	default:
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: fmt.Errorf("unsupported agent run type %q", run.RunType)}
	}
}

func (s *agentRunService) completeAgentRun(
	ctx context.Context,
	run *types.AgentRun,
	result types.AgentResultV1,
	compatibility types.JSONMap,
) error {
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return errors.New("agent run is required to complete")
	}
	runID := run.ID
	if err := s.inheritParentArtifactLineage(ctx, run, &result); err != nil {
		return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: err})
	}
	result = types.NormalizeAgentResultV1(result, runID, time.Now().UTC())
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
			s.emitRunFailure(ctx, runID, types.AgentRunErrorInvalidOutput, message, types.AgentRunStatusFailed, false, false)
		}
		return err
	}
	if run.RunType == types.AgentRunTypeExpertAgentTest {
		if err := s.persistExpertReportArtifacts(ctx, run, &result); err != nil {
			return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: err})
		}
		resultMap, err = result.ToJSONMap()
		if err != nil {
			return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: fmt.Errorf("encode persisted agent result: %w", err)})
		}
		resultMap["validation"] = validationMap
		for key, value := range compatibility {
			resultMap[key] = value
		}
	}
	if strings.TrimSpace(run.ServiceID) != "" && s.serviceSpace != nil {
		if err := s.serviceSpace.IndexRunArtifacts(ctx, run, result.Artifacts); err != nil {
			return s.recordExecutionFailure(ctx, runID, permanentAgentRunError{err: fmt.Errorf("index service artifacts: %w", err)})
		}
	}
	finishedAt := time.Now().UTC()
	succeeded, err := s.repo.MarkSucceeded(ctx, runID, resultMap, finishedAt)
	if err == nil && succeeded {
		totalDurationMs := int64(0)
		if run.StartedAt != nil {
			totalDurationMs = elapsedMilliseconds(*run.StartedAt, finishedAt)
		}
		s.emitRunCompletionEvents(ctx, runID, result, totalDurationMs)
		s.forgetAgentRunThread(runID)
	}
	return err
}

// inheritParentArtifactLineage keeps regeneration attached to the same
// logical artifact while allowing the content version and resource to change.
// A missing or legacy parent artifact is tolerated; the child then starts a
// new logical artifact identity.
func (s *agentRunService) inheritParentArtifactLineage(
	ctx context.Context,
	run *types.AgentRun,
	result *types.AgentResultV1,
) error {
	if run == nil || result == nil || strings.TrimSpace(run.ParentRunID) == "" || s.repo == nil {
		return nil
	}
	parent, err := s.repo.GetByID(ctx, run.ParentRunID)
	if err != nil {
		return fmt.Errorf("load parent agent run for artifact lineage: %w", err)
	}
	if parent == nil {
		return fmt.Errorf("parent agent run %q not found", run.ParentRunID)
	}
	parentArtifacts := decodeAgentRunArtifacts(parent.Result["artifacts"])
	if len(parentArtifacts) == 0 || len(result.Artifacts) == 0 {
		return nil
	}

	used := make([]bool, len(parentArtifacts))
	for index := range result.Artifacts {
		parentIndex := matchAgentArtifactLineage(parentArtifacts, used, result.Artifacts[index])
		if parentIndex < 0 {
			continue
		}
		parentArtifact := parentArtifacts[parentIndex]
		if strings.TrimSpace(parentArtifact.ID) == "" {
			continue
		}
		used[parentIndex] = true
		child := &result.Artifacts[index]
		child.ID = parentArtifact.ID
		child.Version = parentArtifact.Version + 1
		if child.Version <= 1 {
			child.Version = 2
		}
		if child.Metadata == nil {
			child.Metadata = types.JSONMap{}
		}
		if parentArtifact.VersionID != "" {
			child.Metadata["parent_version_id"] = parentArtifact.VersionID
		}
		child.Metadata["parent_run_id"] = run.ParentRunID
	}
	return nil
}

func decodeAgentRunArtifacts(value any) []types.AgentArtifactResultV1 {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var artifacts []types.AgentArtifactResultV1
	if err := json.Unmarshal(raw, &artifacts); err != nil {
		return nil
	}
	return artifacts
}

func matchAgentArtifactLineage(
	parentArtifacts []types.AgentArtifactResultV1,
	used []bool,
	child types.AgentArtifactResultV1,
) int {
	for _, exactKind := range []bool{true, false} {
		for index, parent := range parentArtifacts {
			if used[index] || parent.Role != child.Role {
				continue
			}
			if exactKind && parent.Kind != child.Kind {
				continue
			}
			return index
		}
	}
	return -1
}

// persistExpertReportArtifacts turns the platform report contract into a
// durable HTML object. The FileService may point to local disk, OSS, COS, S3,
// or another configured provider, so the execution workflow stays storage
// agnostic.
func (s *agentRunService) persistExpertReportArtifacts(
	ctx context.Context,
	run *types.AgentRun,
	result *types.AgentResultV1,
) error {
	if s.fileService == nil || result == nil {
		return nil
	}
	supportingArtifacts := make([]types.AgentArtifactResultV1, 0, len(result.Artifacts))
	artifactCount := len(result.Artifacts)
	for index := 0; index < artifactCount; index++ {
		artifact := &result.Artifacts[index]
		if artifact.Kind != types.AgentArtifactKindReport ||
			artifact.Format != types.StructuredReportFormatV1 ||
			len(artifact.Content) == 0 {
			continue
		}
		report, err := types.DecodeStructuredReportV1(artifact.Content)
		if err != nil {
			return fmt.Errorf("decode expert report artifact %d: %w", index, err)
		}
		rendered, err := renderStructuredReportHTML(*report)
		if err != nil {
			return fmt.Errorf("render expert report artifact %d: %w", index, err)
		}
		markdown, err := renderStructuredReportMarkdown(*report)
		if err != nil {
			return fmt.Errorf("render expert report markdown artifact %d: %w", index, err)
		}
		fileBase := expertArtifactFileBase(firstNonEmpty(artifact.Title, report.Title, "专家报告"))
		fileName := fileBase + ".html"
		resourceRef, err := s.fileService.SaveBytes(
			ctx,
			[]byte(rendered),
			run.TenantID,
			fileName,
			false,
		)
		if err != nil {
			return fmt.Errorf("persist expert report artifact %d: %w", index, err)
		}
		if s.resourceCatalog != nil {
			if _, isStableResource := types.ParseResourcePath(resourceRef); isStableResource {
				if err := s.resourceCatalog.Bind(ctx, resourceRef, "agent_run", run.ID, "artifact"); err != nil {
					_ = s.fileService.DeleteFile(ctx, resourceRef)
					return fmt.Errorf("bind expert report artifact %d: %w", index, err)
				}
			}
		}
		artifact.MimeType = "text/html"
		artifact.OriginalName = fileName
		artifact.SizeBytes = int64(len([]byte(rendered)))
		artifact.ResourceRef = resourceRef
		artifact.Downloadable = true
		artifact.Previewable = true
		artifact.Metadata["rendered_from"] = types.StructuredReportFormatV1
		artifact.Metadata["representation"] = "html"

		markdownID := uuid.NewString()
		markdownName := fileBase + ".md"
		markdownRef, err := s.fileService.SaveBytes(
			ctx,
			[]byte(markdown),
			run.TenantID,
			markdownName,
			false,
		)
		if err != nil {
			_ = s.fileService.DeleteFile(ctx, resourceRef)
			return fmt.Errorf("persist expert report markdown artifact %d: %w", index, err)
		}
		if s.resourceCatalog != nil {
			if _, isStableResource := types.ParseResourcePath(markdownRef); isStableResource {
				if err := s.resourceCatalog.Bind(ctx, markdownRef, "agent_run", run.ID, "artifact"); err != nil {
					_ = s.fileService.DeleteFile(ctx, markdownRef)
					_ = s.fileService.DeleteFile(ctx, resourceRef)
					return fmt.Errorf("bind expert report markdown artifact %d: %w", index, err)
				}
			}
		}
		supportingArtifacts = append(supportingArtifacts, types.AgentArtifactResultV1{
			ID:           markdownID,
			Kind:         types.AgentArtifactKindText,
			Role:         types.AgentArtifactRoleSupporting,
			Title:        firstNonEmpty(artifact.Title, report.Title, "专家报告") + "（Markdown 原稿）",
			Format:       "markdown",
			MimeType:     "text/markdown; charset=utf-8",
			OriginalName: markdownName,
			SizeBytes:    int64(len([]byte(markdown))),
			ResourceRef:  markdownRef,
			RunID:        run.ID,
			Lifecycle:    types.AgentArtifactLifecycleTemporary,
			Previewable:  true,
			Downloadable: true,
			Metadata: types.JSONMap{
				"representation":     "markdown",
				"source_artifact_id": artifact.ID,
			},
		})
	}
	result.Artifacts = append(result.Artifacts, supportingArtifacts...)
	*result = types.NormalizeAgentResultV1(*result, run.ID, time.Now().UTC())
	return nil
}

func expertArtifactFileBase(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 32 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return -1
		}
		return r
	}, value)
	value = strings.Trim(value, ". ")
	if value == "" {
		value = "专家报告"
	}
	return trimExpertRunes(value, 80)
}

func (s *agentRunService) recordExecutionFailure(ctx context.Context, runID string, err error) error {
	message := truncateAgentRunError(err)
	var invalidOutput invalidAgentRunOutputError
	if errors.As(err, &invalidOutput) {
		failed, updateErr := s.repo.MarkFailed(ctx, runID, types.AgentRunErrorInvalidOutput, message, time.Now())
		if updateErr == nil && failed {
			s.emitRunFailure(ctx, runID, types.AgentRunErrorInvalidOutput, message, types.AgentRunStatusFailed, false, false)
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
			s.emitRunFailure(ctx, runID, types.AgentRunErrorExecutionFailed, message, types.AgentRunStatusFailed, false, false)
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
	s.emitRunFailure(ctx, runID, types.AgentRunErrorExecutionFailed, message, types.AgentRunStatusQueued, true, true)
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

func (s *agentRunService) ListAgentRunEventsForService(
	ctx context.Context,
	tenantID uint64,
	userID, serviceID, id string,
	afterSequence int64,
	limit int,
) ([]*types.AgentRunEvent, error) {
	run, err := s.GetAgentRunForService(ctx, tenantID, userID, serviceID, id)
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
		Payload:   s.withRunThreadID(ctx, runID, payload),
	}); err != nil {
		// Event persistence must not turn a completed agent result into a
		// failed task. The run state remains authoritative and the error is
		// visible in server logs for operational repair.
		logger.Warnf(ctx, "failed to persist agent run event (run=%s type=%s): %v", runID, eventType, err)
	}
}

func (s *agentRunService) withRunThreadID(
	ctx context.Context,
	runID string,
	payload types.JSONMap,
) types.JSONMap {
	enriched := cloneAgentRunJSONMap(payload)
	if strings.TrimSpace(asString(enriched["threadId"])) != "" {
		return enriched
	}
	if cached, ok := s.threadIDCache.Load(runID); ok {
		if threadID := strings.TrimSpace(asString(cached)); threadID != "" {
			enriched["threadId"] = threadID
			return enriched
		}
	}
	run, err := s.repo.GetByID(ctx, runID)
	if err == nil && run != nil {
		threadID := firstNonEmpty(strings.TrimSpace(run.ThreadID), run.ID)
		s.threadIDCache.Store(runID, threadID)
		enriched["threadId"] = threadID
	}
	return enriched
}

func (s *agentRunService) cacheAgentRunThread(run *types.AgentRun) {
	if run == nil || strings.TrimSpace(run.ID) == "" {
		return
	}
	s.threadIDCache.Store(run.ID, firstNonEmpty(strings.TrimSpace(run.ThreadID), run.ID))
}

func (s *agentRunService) forgetAgentRunThread(runID string) {
	s.threadIDCache.Delete(strings.TrimSpace(runID))
}

func (s *agentRunService) emitRunError(ctx context.Context, runID, code, message string) {
	s.emitRunFailure(ctx, runID, code, message, types.AgentRunStatusFailed, false, false)
}

func (s *agentRunService) emitRunFailure(
	ctx context.Context,
	runID, code, message, status string,
	retryable, automaticRetry bool,
) {
	payload := types.JSONMap{
		"runId":          runID,
		"status":         status,
		"errorCode":      code,
		"message":        message,
		"retryable":      retryable,
		"automaticRetry": automaticRetry,
		"retrying":       automaticRetry,
		"terminal":       !automaticRetry,
	}
	if run, err := s.repo.GetByID(ctx, runID); err == nil && run != nil {
		payload["phase"] = firstNonEmpty(strings.TrimSpace(run.Phase), "execution")
		payload["failureStage"] = firstNonEmpty(strings.TrimSpace(run.Phase), "execution")
		payload["attempt"] = run.Attempt
		payload["threadId"] = firstNonEmpty(strings.TrimSpace(run.ThreadID), run.ID)
	}
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeRunError, payload)
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningMessageEnd, types.JSONMap{
		"runId":     runID,
		"messageId": "reasoning-" + runID,
	})
	s.emitRunEvent(ctx, runID, types.AgentRunEventTypeReasoningEnd, types.JSONMap{
		"runId": runID,
	})
	s.forgetAgentRunThread(runID)
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
	runID string,
	result types.AgentResultV1,
	totalDurationMs int64,
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
		s.emitRunEvent(ctx, runID, types.AgentRunEventTypeArtifactVersionCreated, types.JSONMap{
			"runId":        runID,
			"artifactId":   artifact.ID,
			"versionId":    artifact.VersionID,
			"version":      artifact.Version,
			"kind":         artifact.Kind,
			"role":         artifact.Role,
			"title":        artifact.Title,
			"lifecycle":    artifact.Lifecycle,
			"mimeType":     artifact.MimeType,
			"resourceRef":  artifact.ResourceRef,
			"previewable":  artifact.Previewable,
			"downloadable": artifact.Downloadable,
			"createdAt":    artifact.CreatedAt,
		})
		files = append(files, map[string]any{
			"id":           artifact.ID,
			"versionId":    artifact.VersionID,
			"version":      artifact.Version,
			"runId":        artifact.RunID,
			"name":         artifact.Title,
			"kind":         artifact.Kind,
			"format":       artifact.Format,
			"mimeType":     artifact.MimeType,
			"originalName": artifact.OriginalName,
			"resourceRef":  artifact.ResourceRef,
			"lifecycle":    artifact.Lifecycle,
			"previewable":  artifact.Previewable,
			"downloadable": artifact.Downloadable,
			"shareable":    artifact.Shareable,
			"createdAt":    artifact.CreatedAt,
			"metadata":     artifact.Metadata,
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
		"runId":           runID,
		"status":          types.AgentRunStatusSucceeded,
		"phase":           types.AgentRunPhaseCompleted,
		"totalDurationMs": totalDurationMs,
	})
}

func elapsedMilliseconds(start, end time.Time) int64 {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
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
