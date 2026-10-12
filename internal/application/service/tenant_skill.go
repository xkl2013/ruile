package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"gorm.io/gorm"
)

const (
	maxTenantSkillArchiveBytes   = 20 << 20
	maxTenantSkillExpandedBytes  = 100 << 20
	maxTenantSkillFileBytes      = 10 << 20
	maxTenantSkillArchiveEntries = 500
	tenantSkillInstallStaleAfter = 10 * time.Minute
	maxTenantSkillEnvVars        = 50
	maxTenantSkillEnvValueBytes  = 8 << 10
)

var tenantSkillEnvNamePattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,127}$`)

var tenantSkillReservedEnvNames = map[string]struct{}{
	"PATH": {}, "HOME": {}, "USER": {}, "SHELL": {},
	"LD_PRELOAD": {}, "LD_LIBRARY_PATH": {},
	"PYTHONPATH": {}, "PYTHONHOME": {}, "NODE_OPTIONS": {},
}

type tenantSkillService struct {
	repo         interfaces.TenantSkillRepository
	fileService  interfaces.FileService
	task         interfaces.TaskEnqueuer
	canceller    interfaces.TaskCanceller
	snapshotRepo interfaces.TenantSkillSnapshotRepository
	installMu    sync.Mutex
	installStop  map[string]context.CancelFunc
}

func NewTenantSkillService(
	repo interfaces.TenantSkillRepository,
	fileService interfaces.FileService,
) interfaces.TenantSkillService {
	return &tenantSkillService{
		repo:        repo,
		fileService: fileService,
		installStop: make(map[string]context.CancelFunc),
	}
}

// NewTenantSkillServiceWithTasks wires Skill installation into the shared
// Asynq/Lite task abstraction. The two-argument constructor remains available
// for focused service tests and local callers that need the legacy fallback.
func NewTenantSkillServiceWithTasks(
	repo interfaces.TenantSkillRepository,
	fileService interfaces.FileService,
	task interfaces.TaskEnqueuer,
	canceller interfaces.TaskCanceller,
	snapshotRepo interfaces.TenantSkillSnapshotRepository,
) interfaces.TenantSkillService {
	return &tenantSkillService{
		repo:         repo,
		fileService:  fileService,
		task:         task,
		canceller:    canceller,
		snapshotRepo: snapshotRepo,
		installStop:  make(map[string]context.CancelFunc),
	}
}

func (s *tenantSkillService) ListTenantSkills(ctx context.Context, tenantID uint64) ([]*types.TenantSkill, error) {
	return s.repo.List(ctx, tenantID)
}

func (s *tenantSkillService) ListGlobalSkills(ctx context.Context) ([]*types.TenantSkill, error) {
	return s.ListTenantSkills(ctx, types.SystemSkillTenantID)
}

func (s *tenantSkillService) UploadGlobalSkill(
	ctx context.Context,
	userID string,
	file *multipart.FileHeader,
) (*types.TenantSkill, error) {
	return s.UploadTenantSkill(ctx, types.SystemSkillTenantID, userID, file)
}

func (s *tenantSkillService) UploadGlobalSkillFromSource(
	ctx context.Context,
	userID, source string,
) (*types.TenantSkill, error) {
	return s.UploadTenantSkillFromSource(ctx, types.SystemSkillTenantID, userID, source)
}

func (s *tenantSkillService) StartGlobalSkillSourceInstall(
	ctx context.Context,
	userID, source string,
) (*types.TenantSkillInstallJob, error) {
	return s.StartTenantSkillSourceInstall(ctx, types.SystemSkillTenantID, userID, source)
}

func (s *tenantSkillService) ListGlobalSkillInstallJobs(
	ctx context.Context,
	limit int,
) ([]*types.TenantSkillInstallJob, error) {
	return s.ListTenantSkillInstallJobs(ctx, types.SystemSkillTenantID, limit)
}

func (s *tenantSkillService) GetGlobalSkillInstallJob(
	ctx context.Context,
	id string,
) (*types.TenantSkillInstallJob, error) {
	return s.GetTenantSkillInstallJob(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) CancelGlobalSkillInstall(
	ctx context.Context,
	id string,
) (*types.TenantSkillInstallJob, error) {
	return s.CancelTenantSkillInstall(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) RetryGlobalSkillInstall(
	ctx context.Context,
	id string,
) (*types.TenantSkillInstallJob, error) {
	return s.RetryTenantSkillInstall(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) SetGlobalSkillEnabled(ctx context.Context, id string, enabled bool) error {
	return s.SetTenantSkillEnabled(ctx, types.SystemSkillTenantID, id, enabled)
}

func (s *tenantSkillService) SetGlobalSkillEnvVars(
	ctx context.Context,
	id string,
	envVars map[string]string,
) error {
	return s.SetTenantSkillEnvVars(ctx, types.SystemSkillTenantID, id, envVars)
}

func (s *tenantSkillService) GetGlobalSkillEnvVars(
	ctx context.Context,
	id string,
) (map[string]string, error) {
	return s.GetTenantSkillEnvVars(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) DeleteGlobalSkill(ctx context.Context, id string) error {
	return s.DeleteTenantSkill(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) ListGlobalSkillFiles(ctx context.Context, id string) ([]string, error) {
	return s.ListTenantSkillFiles(ctx, types.SystemSkillTenantID, id)
}

func (s *tenantSkillService) ReadGlobalSkillFile(
	ctx context.Context,
	id, filePath string,
) (io.ReadCloser, error) {
	return s.ReadTenantSkillFile(ctx, types.SystemSkillTenantID, id, filePath)
}

func (s *tenantSkillService) UploadTenantSkill(
	ctx context.Context,
	tenantID uint64,
	userID string,
	file *multipart.FileHeader,
) (*types.TenantSkill, error) {
	if file == nil {
		return nil, fmt.Errorf("skill archive is required")
	}
	if file.Size <= 0 || file.Size > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive must be between 1 byte and %d MB", maxTenantSkillArchiveBytes>>20)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open skill archive: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, maxTenantSkillArchiveBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read skill archive: %w", err)
	}
	if len(data) > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive is too large")
	}

	metadata, _, err := inspectTenantSkillArchive(data)
	if err != nil {
		return nil, err
	}
	return s.storeTenantSkillArchive(ctx, tenantID, userID, data, metadata, types.TenantSkillSourceUpload, "")
}

func (s *tenantSkillService) UploadTenantSkillFromSource(
	ctx context.Context,
	tenantID uint64,
	userID, source string,
) (*types.TenantSkill, error) {
	parsed, err := parseTenantSkillSource(source)
	if err != nil {
		return nil, err
	}
	data, err := fetchTenantSkillSource(ctx, parsed, nil)
	if err != nil {
		return nil, err
	}
	metadata, _, err := inspectTenantSkillArchive(data)
	if err != nil {
		return nil, err
	}
	return s.storeTenantSkillArchive(ctx, tenantID, userID, data, metadata, parsed.Kind, parsed.Ref)
}

func (s *tenantSkillService) StartTenantSkillSourceInstall(
	ctx context.Context,
	tenantID uint64,
	userID, source string,
) (*types.TenantSkillInstallJob, error) {
	parsed, err := parseTenantSkillSource(source)
	if err != nil {
		return nil, err
	}
	job := &types.TenantSkillInstallJob{
		TenantID:   tenantID,
		CreatedBy:  userID,
		SourceType: parsed.Kind,
		SourceRef:  parsed.Ref,
		Status:     types.TenantSkillInstallStatusPending,
		Attempt:    1,
	}
	if err := s.repo.CreateInstallJob(ctx, job); err != nil {
		return nil, fmt.Errorf("create skill install job: %w", err)
	}
	if s.task != nil {
		if err := s.enqueueTenantSkillInstall(ctx, job); err != nil {
			now := time.Now().UTC()
			job.Status = types.TenantSkillInstallStatusFailed
			job.Error = truncateTenantSkillError(err.Error())
			job.FinishedAt = &now
			job.UpdatedAt = now
			_ = s.repo.UpdateInstallJob(ctx, job)
			return nil, fmt.Errorf("enqueue skill install job: %w", err)
		}
	} else {
		s.launchTenantSkillInstall(job, parsed)
	}
	return job, nil
}

func (s *tenantSkillService) ListTenantSkillInstallJobs(
	ctx context.Context,
	tenantID uint64,
	limit int,
) ([]*types.TenantSkillInstallJob, error) {
	return s.repo.ListInstallJobs(ctx, tenantID, limit)
}

func (s *tenantSkillService) GetTenantSkillInstallJob(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillInstallJob, error) {
	job, err := s.repo.GetInstallJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return job, nil
}

func (s *tenantSkillService) CancelTenantSkillInstall(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillInstallJob, error) {
	job, err := s.GetTenantSkillInstallJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if isTenantSkillInstallTerminal(job.Status) {
		return job, nil
	}
	job.CancelRequested = true
	job.Status = types.TenantSkillInstallStatusCancelling
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
		return nil, fmt.Errorf("cancel skill install job: %w", err)
	}
	s.installMu.Lock()
	cancel := s.installStop[job.ID]
	s.installMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if s.canceller != nil && job.TaskID != "" {
		cancelled, cancelErr := s.canceller.CancelTask(
			ctx,
			types.QueueMaintenance,
			job.TaskID,
		)
		if cancelErr != nil {
			return nil, fmt.Errorf("cancel skill install task: %w", cancelErr)
		}
		if cancelled && job.StartedAt == nil {
			now := time.Now().UTC()
			job.Status = types.TenantSkillInstallStatusCancelled
			job.FinishedAt = &now
			job.UpdatedAt = now
			if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
				return nil, fmt.Errorf("finish cancelled skill install job: %w", err)
			}
		}
	}
	return job, nil
}

func (s *tenantSkillService) RetryTenantSkillInstall(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.TenantSkillInstallJob, error) {
	job, err := s.GetTenantSkillInstallJob(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	retryable := job.Status == types.TenantSkillInstallStatusFailed ||
		job.Status == types.TenantSkillInstallStatusCancelled
	if !retryable && time.Since(job.UpdatedAt) >= tenantSkillInstallStaleAfter &&
		!s.hasActiveTenantSkillInstall(job.ID) {
		retryable = job.Status == types.TenantSkillInstallStatusPending ||
			job.Status == types.TenantSkillInstallStatusRunning ||
			job.Status == types.TenantSkillInstallStatusCancelling
	}
	if !retryable {
		return nil, fmt.Errorf("skill install job can only retry from failed/cancelled or stale inactive state")
	}
	parsed, err := parseTenantSkillSource(job.SourceRef)
	if err != nil {
		return nil, err
	}
	job.SourceType = parsed.Kind
	job.Status = types.TenantSkillInstallStatusPending
	job.Error = ""
	job.CancelRequested = false
	job.SkillID = ""
	job.StartedAt = nil
	job.FinishedAt = nil
	job.Attempt++
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
		return nil, fmt.Errorf("retry skill install job: %w", err)
	}
	if s.task != nil {
		if err := s.enqueueTenantSkillInstall(ctx, job); err != nil {
			return nil, fmt.Errorf("enqueue retried skill install job: %w", err)
		}
	} else {
		s.launchTenantSkillInstall(job, parsed)
	}
	return job, nil
}

func (s *tenantSkillService) enqueueTenantSkillInstall(
	ctx context.Context,
	job *types.TenantSkillInstallJob,
) error {
	if s.task == nil {
		return errors.New("skill install task enqueuer is not configured")
	}
	payload, err := json.Marshal(types.TenantSkillInstallTaskPayload{
		TenantID: job.TenantID,
		JobID:    job.ID,
	})
	if err != nil {
		return fmt.Errorf("encode skill install task: %w", err)
	}
	taskID := fmt.Sprintf("tenant-skill-install-%s-%d", job.ID, job.Attempt)
	job.TaskID = taskID
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
		return fmt.Errorf("save skill install task id: %w", err)
	}
	queue, ok := types.QueueForTaskType(types.TypeTenantSkillInstall)
	if !ok {
		return errors.New("skill install queue is not configured")
	}
	_, err = s.task.Enqueue(
		asynq.NewTask(types.TypeTenantSkillInstall, payload),
		asynq.TaskID(taskID),
		asynq.Queue(queue),
		asynq.MaxRetry(3),
		asynq.Timeout(20*time.Minute),
	)
	if err != nil && !errors.Is(err, asynq.ErrTaskIDConflict) {
		return err
	}
	return nil
}

func (s *tenantSkillService) launchTenantSkillInstall(
	job *types.TenantSkillInstallJob,
	parsed tenantSkillSource,
) {
	workerJob := *job
	if job.StartedAt != nil {
		startedAt := *job.StartedAt
		workerJob.StartedAt = &startedAt
	}
	if job.FinishedAt != nil {
		finishedAt := *job.FinishedAt
		workerJob.FinishedAt = &finishedAt
	}
	runCtx, cancel := context.WithCancel(context.Background())
	s.installMu.Lock()
	s.installStop[job.ID] = cancel
	s.installMu.Unlock()
	go func() {
		defer func() {
			s.installMu.Lock()
			delete(s.installStop, job.ID)
			s.installMu.Unlock()
		}()
		s.runTenantSkillInstall(runCtx, &workerJob, parsed)
	}()
}

func (s *tenantSkillService) runTenantSkillInstall(
	ctx context.Context,
	job *types.TenantSkillInstallJob,
	parsed tenantSkillSource,
) {
	if err := s.executeTenantSkillInstall(ctx, job, parsed); err != nil {
		s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusFailed, truncateTenantSkillError(err.Error()))
	}
}

// Handle implements interfaces.TaskHandler for both the Redis-backed Asynq
// worker and the Lite-mode synchronous executor. Transient failures are
// returned to the queue until the configured retry budget is exhausted.
func (s *tenantSkillService) Handle(ctx context.Context, task *asynq.Task) error {
	if task == nil {
		return errors.New("skill install task is required")
	}
	var payload types.TenantSkillInstallTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode skill install task: %w", err)
	}
	if payload.JobID == "" {
		return errors.New("skill install job id is required")
	}
	job, err := s.repo.GetInstallJob(ctx, payload.TenantID, payload.JobID)
	if err != nil {
		return fmt.Errorf("load skill install job: %w", err)
	}
	if job == nil {
		return gorm.ErrRecordNotFound
	}
	if isTenantSkillInstallTerminal(job.Status) {
		return nil
	}
	parsed, err := parseTenantSkillSource(job.SourceRef)
	if err != nil {
		s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusFailed, truncateTenantSkillError(err.Error()))
		return nil
	}
	if err := s.executeTenantSkillInstall(ctx, job, parsed); err != nil {
		retryCount, retryCountOK := asynq.GetRetryCount(ctx)
		maxRetry, maxRetryOK := asynq.GetMaxRetry(ctx)
		if retryCountOK && maxRetryOK && retryCount < maxRetry &&
			ctx.Err() == nil &&
			!s.tenantSkillInstallCancelled(job.ID, job.TenantID) {
			job.Error = truncateTenantSkillError(err.Error())
			job.UpdatedAt = time.Now().UTC()
			_ = s.repo.UpdateInstallJob(context.Background(), job)
			return err
		}
		if errors.Is(err, context.Canceled) || ctx.Err() != nil ||
			s.tenantSkillInstallCancelled(job.ID, job.TenantID) {
			s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusCancelled, "")
			return nil
		}
		s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusFailed, truncateTenantSkillError(err.Error()))
	}
	return nil
}

func (s *tenantSkillService) executeTenantSkillInstall(
	ctx context.Context,
	job *types.TenantSkillInstallJob,
	parsed tenantSkillSource,
) error {
	now := time.Now().UTC()
	job.Status = types.TenantSkillInstallStatusRunning
	job.StartedAt = &now
	job.UpdatedAt = now
	if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
		return err
	}

	if s.tenantSkillInstallCancelled(job.ID, job.TenantID) {
		s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusCancelled, "")
		return nil
	}
	data, err := fetchTenantSkillSource(ctx, parsed, nil)
	if err == nil && ctx.Err() == nil && !s.tenantSkillInstallCancelled(job.ID, job.TenantID) {
		var metadata *skills.Skill
		metadata, _, err = inspectTenantSkillArchive(data)
		if err == nil {
			var skill *types.TenantSkill
			skill, err = s.storeTenantSkillArchive(ctx, job.TenantID, job.CreatedBy, data, metadata, parsed.Kind, parsed.Ref)
			if err == nil {
				job.SkillID = skill.ID
			}
		}
	}

	if err != nil {
		if errors.Is(err, context.Canceled) || ctx.Err() != nil ||
			s.tenantSkillInstallCancelled(job.ID, job.TenantID) {
			s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusCancelled, "")
			return nil
		}
		return err
	}
	s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusSucceeded, "")
	return nil
}

// RecoverPendingTenantSkillInstalls re-enqueues work left in a non-terminal
// state after a process restart. Running jobs are only recovered after the
// stale threshold so a second application instance does not steal active work.
func (s *tenantSkillService) RecoverPendingTenantSkillInstalls(ctx context.Context) error {
	jobs, err := s.repo.ListInstallJobsByStatus(ctx, []string{
		types.TenantSkillInstallStatusPending,
		types.TenantSkillInstallStatusRunning,
		types.TenantSkillInstallStatusCancelling,
	}, 1000)
	if err != nil {
		return fmt.Errorf("list recoverable skill install jobs: %w", err)
	}
	now := time.Now().UTC()
	for _, job := range jobs {
		if job == nil {
			continue
		}
		if job.Status != types.TenantSkillInstallStatusPending &&
			now.Sub(job.UpdatedAt) < tenantSkillInstallStaleAfter {
			continue
		}
		parsed, parseErr := parseTenantSkillSource(job.SourceRef)
		if parseErr != nil {
			s.finishTenantSkillInstall(job, types.TenantSkillInstallStatusFailed, truncateTenantSkillError(parseErr.Error()))
			continue
		}
		job.Status = types.TenantSkillInstallStatusPending
		job.Error = ""
		job.CancelRequested = false
		job.StartedAt = nil
		job.FinishedAt = nil
		job.UpdatedAt = now
		if err := s.repo.UpdateInstallJob(ctx, job); err != nil {
			return fmt.Errorf("reset stale skill install job %s: %w", job.ID, err)
		}
		if s.task != nil {
			if err := s.enqueueTenantSkillInstall(ctx, job); err != nil {
				return fmt.Errorf("requeue skill install job %s: %w", job.ID, err)
			}
		} else {
			s.launchTenantSkillInstall(job, parsed)
		}
	}
	return nil
}

func (s *tenantSkillService) tenantSkillInstallCancelled(id string, tenantID uint64) bool {
	job, err := s.repo.GetInstallJob(context.Background(), tenantID, id)
	return err == nil && job != nil && job.CancelRequested
}

func (s *tenantSkillService) hasActiveTenantSkillInstall(id string) bool {
	s.installMu.Lock()
	defer s.installMu.Unlock()
	_, ok := s.installStop[id]
	return ok
}

func (s *tenantSkillService) finishTenantSkillInstall(
	job *types.TenantSkillInstallJob,
	status, message string,
) {
	now := time.Now().UTC()
	job.Status = status
	job.Error = message
	job.FinishedAt = &now
	job.UpdatedAt = now
	_ = s.repo.UpdateInstallJob(context.Background(), job)
}

func isTenantSkillInstallTerminal(status string) bool {
	switch status {
	case types.TenantSkillInstallStatusSucceeded,
		types.TenantSkillInstallStatusFailed,
		types.TenantSkillInstallStatusCancelled:
		return true
	default:
		return false
	}
}

func (s *tenantSkillService) storeTenantSkillArchive(
	ctx context.Context,
	tenantID uint64,
	userID string,
	data []byte,
	metadata *skills.Skill,
	sourceType, sourceRef string,
) (*types.TenantSkill, error) {
	if metadata == nil {
		return nil, fmt.Errorf("skill metadata is required")
	}
	sum := sha256.Sum256(data)
	bundleSHA := fmt.Sprintf("%x", sum[:])
	existing, err := s.repo.GetByName(ctx, tenantID, metadata.Name)
	if err != nil {
		return nil, fmt.Errorf("check existing skill: %w", err)
	}

	if existing != nil && existing.BundleSHA == bundleSHA {
		existing.Description = metadata.Description
		existing.Version = skillVersion(metadata)
		existing.SourceType = sourceType
		existing.SourceRef = sourceRef
		existing.Status = types.TenantSkillStatusReady
		existing.UpdatedAt = time.Now().UTC()
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update skill metadata: %w", err)
		}
		return existing, nil
	}

	bundlePath, err := s.fileService.SaveBytes(
		ctx,
		data,
		tenantID,
		fmt.Sprintf("skill-%s.zip", uuid.NewString()),
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("save skill archive: %w", err)
	}

	if existing == nil {
		skill := &types.TenantSkill{
			TenantID:    tenantID,
			CreatedBy:   userID,
			Name:        metadata.Name,
			Description: metadata.Description,
			Version:     skillVersion(metadata),
			BundlePath:  bundlePath,
			BundleSHA:   bundleSHA,
			SourceType:  sourceType,
			SourceRef:   sourceRef,
			Status:      types.TenantSkillStatusReady,
			Enabled:     true,
		}
		if err := s.repo.Create(ctx, skill); err != nil {
			_ = s.fileService.DeleteFile(ctx, bundlePath)
			if errorsIsUnique(err) {
				return nil, fmt.Errorf("skill %q already exists", metadata.Name)
			}
			return nil, fmt.Errorf("save skill metadata: %w", err)
		}
		return skill, nil
	}

	previousBundlePath := existing.BundlePath
	existing.Description = metadata.Description
	existing.Version = skillVersion(metadata)
	existing.BundlePath = bundlePath
	existing.BundleSHA = bundleSHA
	existing.SourceType = sourceType
	existing.SourceRef = sourceRef
	existing.Status = types.TenantSkillStatusReady
	existing.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, existing); err != nil {
		_ = s.fileService.DeleteFile(ctx, bundlePath)
		return nil, fmt.Errorf("update skill metadata: %w", err)
	}
	// Keep the previous archive. v2 Skill snapshots pin BundlePath, so
	// deleting it here would make an older snapshot impossible to activate.
	// A later retention job can remove unreferenced bundles once snapshot
	// garbage collection is implemented.
	_ = previousBundlePath
	return existing, nil
}

func (s *tenantSkillService) SetTenantSkillEnabled(ctx context.Context, tenantID uint64, id string, enabled bool) error {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if skill == nil {
		return gorm.ErrRecordNotFound
	}
	return s.repo.SetEnabled(ctx, tenantID, id, enabled)
}

func (s *tenantSkillService) SetTenantSkillEnvVars(
	ctx context.Context,
	tenantID uint64,
	id string,
	envVars map[string]string,
) error {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if skill == nil {
		return gorm.ErrRecordNotFound
	}
	validated, err := validateTenantSkillEnvVars(envVars)
	if err != nil {
		return err
	}
	return s.repo.SetEnvVars(ctx, tenantID, id, validated)
}

func (s *tenantSkillService) GetTenantSkillEnvVars(
	ctx context.Context,
	tenantID uint64,
	id string,
) (map[string]string, error) {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if skill == nil {
		return nil, gorm.ErrRecordNotFound
	}
	values := make(map[string]string, len(skill.EnvVars))
	for name, value := range skill.EnvVars {
		values[name] = value
	}
	return values, nil
}

func validateTenantSkillEnvVars(input map[string]string) (types.TenantSkillEnvVars, error) {
	if len(input) > maxTenantSkillEnvVars {
		return nil, fmt.Errorf("a skill can configure at most %d environment variables", maxTenantSkillEnvVars)
	}
	out := make(types.TenantSkillEnvVars, len(input))
	for rawName, rawValue := range input {
		name := strings.TrimSpace(rawName)
		if !tenantSkillEnvNamePattern.MatchString(name) {
			return nil, fmt.Errorf("environment variable name %q must be UPPER_SNAKE_CASE", rawName)
		}
		if _, reserved := tenantSkillReservedEnvNames[name]; reserved ||
			strings.HasPrefix(name, "WEKNORA_SKILL_") {
			return nil, fmt.Errorf("environment variable name %q is reserved by the sandbox", name)
		}
		if strings.IndexByte(rawValue, 0) >= 0 {
			return nil, fmt.Errorf("environment variable %q contains a NUL byte", name)
		}
		if len([]byte(rawValue)) > maxTenantSkillEnvValueBytes {
			return nil, fmt.Errorf("environment variable %q exceeds %d bytes", name, maxTenantSkillEnvValueBytes)
		}
		out[name] = rawValue
	}
	return out, nil
}

func (s *tenantSkillService) DeleteTenantSkill(ctx context.Context, tenantID uint64, id string) error {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return err
	}
	if skill == nil {
		return gorm.ErrRecordNotFound
	}
	if err := s.repo.Delete(ctx, tenantID, id); err != nil {
		return err
	}
	if skill.BundlePath != "" {
		_ = s.fileService.DeleteFile(ctx, skill.BundlePath)
	}
	return nil
}

func (s *tenantSkillService) ListTenantSkillFiles(ctx context.Context, tenantID uint64, id string) ([]string, error) {
	root, cleanup, err := s.materializeOne(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	var files []string
	err = filepath.Walk(root, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, filePath)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(files)
	return files, err
}

func (s *tenantSkillService) ReadTenantSkillFile(
	ctx context.Context,
	tenantID uint64,
	id, filePath string,
) (io.ReadCloser, error) {
	root, cleanup, err := s.materializeOne(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	clean := filepath.Clean(filepath.FromSlash(filePath))
	if clean == "." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || clean == ".." {
		cleanup()
		return nil, fmt.Errorf("invalid skill file path")
	}
	full := filepath.Join(root, clean)
	absRoot, _ := filepath.Abs(root)
	absFile, _ := filepath.Abs(full)
	if absFile != absRoot && !strings.HasPrefix(absFile, absRoot+string(os.PathSeparator)) {
		cleanup()
		return nil, fmt.Errorf("skill file path escapes archive")
	}
	file, err := os.Open(absFile)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &cleanupReadCloser{ReadCloser: file, cleanup: cleanup}, nil
}

// ListRuntimeTenantSkills returns the enabled catalog when no v2 snapshot is
// active. Once a v2 snapshot is active, the manifest becomes the runtime
// source of truth: it pins both the selected Skill IDs and their bundle paths,
// so a rollback can use the previous archive instead of the current catalog
// row.
func (s *tenantSkillService) ListRuntimeTenantSkills(
	ctx context.Context,
	tenantID uint64,
) ([]*types.TenantSkill, error) {
	globalSkills, err := s.ListGlobalSkills(ctx)
	if err != nil {
		return nil, err
	}
	tenantSkills, err := s.ListTenantSkills(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	all := append(append([]*types.TenantSkill{}, tenantSkills...), globalSkills...)
	if s.snapshotRepo == nil {
		return runtimeEnabledSkills(all), nil
	}
	active, err := s.snapshotRepo.GetActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if active == nil || snapshotManifestSchema(active.Manifest) != "tenant_skill_snapshot_v2" {
		return runtimeEnabledSkills(all), nil
	}

	entries, err := snapshotManifestEntries(active.Manifest)
	if err != nil {
		return nil, err
	}
	if err := s.ValidateSkillSnapshot(ctx, tenantID, active.Manifest); err != nil {
		return nil, err
	}
	byID := make(map[string]*types.TenantSkill, len(all))
	for _, skill := range all {
		if skill != nil {
			byID[skill.ID] = skill
		}
	}
	result := make([]*types.TenantSkill, 0, len(entries))
	for _, entry := range entries {
		skill := byID[entry.ID]
		if skill == nil {
			return nil, fmt.Errorf("active skill snapshot references missing skill %q", entry.ID)
		}
		if entry.BundlePath == "" || entry.BundleSHA == "" {
			return nil, fmt.Errorf("active skill snapshot entry %q is incomplete", entry.ID)
		}
		pinned := *skill
		pinned.BundlePath = entry.BundlePath
		pinned.BundleSHA = entry.BundleSHA
		pinned.Version = entry.Version
		pinned.Enabled = true
		pinned.Status = types.TenantSkillStatusReady
		result = append(result, &pinned)
	}
	return result, nil
}

func (s *tenantSkillService) ValidateSkillSnapshot(
	ctx context.Context,
	tenantID uint64,
	manifest types.JSONMap,
) error {
	if snapshotManifestSchema(manifest) != "tenant_skill_snapshot_v2" {
		return nil
	}
	entries, err := snapshotManifestEntries(manifest)
	if err != nil {
		return err
	}
	globalSkills, err := s.ListGlobalSkills(ctx)
	if err != nil {
		return err
	}
	tenantSkills, err := s.ListTenantSkills(ctx, tenantID)
	if err != nil {
		return err
	}
	byID := make(map[string]*types.TenantSkill, len(globalSkills)+len(tenantSkills))
	for _, skill := range append(append([]*types.TenantSkill{}, tenantSkills...), globalSkills...) {
		if skill != nil {
			byID[skill.ID] = skill
		}
	}
	for _, entry := range entries {
		skill := byID[entry.ID]
		if skill == nil {
			return fmt.Errorf("skill snapshot references missing skill %q", entry.ID)
		}
		if entry.BundlePath == "" || entry.BundleSHA == "" {
			return fmt.Errorf("skill snapshot entry %q is incomplete", entry.ID)
		}
		pinned := *skill
		pinned.BundlePath = entry.BundlePath
		pinned.BundleSHA = entry.BundleSHA
		if _, err := s.readBundle(ctx, &pinned); err != nil {
			return fmt.Errorf("skill snapshot bundle %q is unavailable: %w", entry.ID, err)
		}
	}
	return nil
}

func (s *tenantSkillService) MaterializeTenantSkillDirs(ctx context.Context, tenantID uint64) ([]string, error) {
	skillsList, err := s.ListRuntimeTenantSkills(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	// Keep legacy tenant-owned Skills usable while making system Skills
	// available to every workspace. A global Skill with the same name wins.
	skillsByName := make(map[string]*types.TenantSkill, len(skillsList))
	for _, skill := range skillsList {
		if skill != nil {
			skillsByName[skill.Name] = skill
		}
	}
	uniqueSkills := make([]*types.TenantSkill, 0, len(skillsByName))
	for _, skill := range skillsByName {
		uniqueSkills = append(uniqueSkills, skill)
	}
	sort.Slice(uniqueSkills, func(i, j int) bool {
		return uniqueSkills[i].Name < uniqueSkills[j].Name
	})
	root := tenantSkillCacheRoot(tenantID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create tenant skill cache: %w", err)
	}
	runtimeRoot, err := os.MkdirTemp(root, "runtime-")
	if err != nil {
		return nil, fmt.Errorf("create tenant skill runtime cache: %w", err)
	}

	materialized := 0
	for _, skill := range uniqueSkills {
		if skill == nil || !skill.Enabled || skill.Status != types.TenantSkillStatusReady {
			continue
		}
		data, err := s.readBundle(ctx, skill)
		if err != nil {
			logger.Warnf(ctx, "Skip tenant skill %s: %v", skill.Name, err)
			continue
		}
		_, archiveRoot, err := inspectTenantSkillArchive(data)
		if err != nil {
			logger.Warnf(ctx, "Skip invalid tenant skill %s: %v", skill.Name, err)
			continue
		}
		if err := extractTenantSkillArchive(data, runtimeRoot, skill.Name, archiveRoot); err != nil {
			logger.Warnf(ctx, "Skip tenant skill %s: %v", skill.Name, err)
			continue
		}
		materialized++
	}
	if materialized == 0 {
		_ = os.RemoveAll(runtimeRoot)
		return nil, nil
	}
	return []string{runtimeRoot}, nil
}

type tenantSkillSnapshotEntry struct {
	ID         string
	Name       string
	Version    string
	BundleSHA  string
	BundlePath string
}

func runtimeEnabledSkills(skills []*types.TenantSkill) []*types.TenantSkill {
	result := make([]*types.TenantSkill, 0, len(skills))
	for _, skill := range skills {
		if skill != nil && skill.Enabled && skill.Status == types.TenantSkillStatusReady {
			result = append(result, skill)
		}
	}
	return result
}

func snapshotManifestSchema(manifest types.JSONMap) string {
	if manifest == nil {
		return ""
	}
	schema, _ := manifest["schema_version"].(string)
	return schema
}

func snapshotManifestEntries(manifest types.JSONMap) ([]tenantSkillSnapshotEntry, error) {
	raw, ok := manifest["skills"].([]any)
	if !ok {
		return nil, errors.New("skill snapshot manifest skills must be an array")
	}
	entries := make([]tenantSkillSnapshotEntry, 0, len(raw))
	for _, item := range raw {
		values, ok := item.(map[string]any)
		if !ok {
			return nil, errors.New("skill snapshot manifest entry is invalid")
		}
		id, _ := values["id"].(string)
		name, _ := values["name"].(string)
		version, _ := values["version"].(string)
		bundleSHA, _ := values["bundle_sha256"].(string)
		bundlePath, _ := values["bundle_path"].(string)
		if id == "" || name == "" {
			return nil, errors.New("skill snapshot manifest entry is incomplete")
		}
		entries = append(entries, tenantSkillSnapshotEntry{
			ID:         id,
			Name:       name,
			Version:    version,
			BundleSHA:  bundleSHA,
			BundlePath: bundlePath,
		})
	}
	return entries, nil
}

func (s *tenantSkillService) materializeOne(ctx context.Context, tenantID uint64, id string) (string, func(), error) {
	skill, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return "", func() {}, err
	}
	if skill == nil {
		return "", func() {}, gorm.ErrRecordNotFound
	}
	data, err := s.readBundle(ctx, skill)
	if err != nil {
		return "", func() {}, err
	}
	_, archiveRoot, err := inspectTenantSkillArchive(data)
	if err != nil {
		return "", func() {}, err
	}
	root, err := os.MkdirTemp("", "weknora-skill-")
	if err != nil {
		return "", func() {}, err
	}
	if err := extractTenantSkillArchive(data, root, "skill", archiveRoot); err != nil {
		_ = os.RemoveAll(root)
		return "", func() {}, err
	}
	return filepath.Join(root, "skill"), func() { _ = os.RemoveAll(root) }, nil
}

func (s *tenantSkillService) readBundle(ctx context.Context, skill *types.TenantSkill) ([]byte, error) {
	reader, err := s.fileService.GetFile(ctx, skill.BundlePath)
	if err != nil {
		return nil, fmt.Errorf("read skill archive: %w", err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxTenantSkillArchiveBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxTenantSkillArchiveBytes {
		return nil, fmt.Errorf("skill archive is too large")
	}
	return data, nil
}

func inspectTenantSkillArchive(data []byte) (*skills.Skill, string, error) {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, "", fmt.Errorf("invalid skill ZIP archive: %w", err)
	}
	if len(archive.File) == 0 || len(archive.File) > maxTenantSkillArchiveEntries {
		return nil, "", fmt.Errorf("skill archive contains an invalid number of files")
	}
	var skillPath string
	for _, file := range archive.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return nil, "", err
		}
		if file.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, "", fmt.Errorf("skill archive cannot contain symlinks")
		}
		if !file.FileInfo().IsDir() && path.Base(clean) == skills.SkillFileName {
			if skillPath != "" {
				return nil, "", fmt.Errorf("skill archive must contain exactly one %s", skills.SkillFileName)
			}
			skillPath = clean
		}
	}
	if skillPath == "" {
		return nil, "", fmt.Errorf("skill archive must contain %s", skills.SkillFileName)
	}
	file, err := findZipFile(archive, skillPath)
	if err != nil {
		return nil, "", err
	}
	content, err := readZipEntry(file, maxTenantSkillFileBytes)
	if err != nil {
		return nil, "", err
	}
	metadata, err := skills.ParseSkillFile(string(content))
	if err != nil {
		return nil, "", fmt.Errorf("invalid %s: %w", skills.SkillFileName, err)
	}
	return metadata, path.Dir(skillPath), nil
}

func extractTenantSkillArchive(data []byte, baseDir, outputName, archiveRoot string) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	target := filepath.Join(baseDir, outputName)
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	var expanded int64
	for _, file := range archive.File {
		clean, err := cleanArchivePath(file.Name)
		if err != nil {
			return err
		}
		if archiveRoot != "." {
			if clean != archiveRoot && !strings.HasPrefix(clean, archiveRoot+"/") {
				continue
			}
			clean = strings.TrimPrefix(strings.TrimPrefix(clean, archiveRoot), "/")
		}
		if clean == "" {
			continue
		}
		info := file.FileInfo()
		if info.IsDir() {
			continue
		}
		if info.Size() > maxTenantSkillFileBytes {
			return fmt.Errorf("skill file %q is too large", clean)
		}
		expanded += info.Size()
		if expanded > maxTenantSkillExpandedBytes {
			return fmt.Errorf("skill archive expands beyond %d MB", maxTenantSkillExpandedBytes>>20)
		}
		targetPath := filepath.Join(target, filepath.FromSlash(clean))
		absTarget, _ := filepath.Abs(target)
		absPath, _ := filepath.Abs(targetPath)
		if absPath != absTarget && !strings.HasPrefix(absPath, absTarget+string(os.PathSeparator)) {
			return fmt.Errorf("skill archive path escapes target")
		}
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return err
		}
		reader, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			reader.Close()
			return err
		}
		written, copyErr := io.CopyN(out, reader, maxTenantSkillFileBytes+1)
		reader.Close()
		closeErr := out.Close()
		if copyErr != nil && copyErr != io.EOF {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if written > maxTenantSkillFileBytes {
			return fmt.Errorf("skill file %q is too large", clean)
		}
	}
	return nil
}

func cleanArchivePath(value string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(value, "\\", "/"))
	if clean == "." || path.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("invalid skill archive path")
	}
	return clean, nil
}

func findZipFile(archive *zip.Reader, name string) (*zip.File, error) {
	for _, file := range archive.File {
		clean, _ := cleanArchivePath(file.Name)
		if clean == name {
			return file, nil
		}
	}
	return nil, fmt.Errorf("skill archive entry not found")
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("skill file is too large")
	}
	return data, nil
}

func tenantSkillCacheRoot(tenantID uint64) string {
	base := os.Getenv("WEKNORA_TENANT_SKILL_CACHE_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), "weknora-tenant-skills")
	}
	return filepath.Join(base, fmt.Sprintf("%d", tenantID))
}

func skillVersion(skill *skills.Skill) string {
	// SKILL.md currently has no required version field. Keep an empty version
	// until the catalog format adds one, while preserving the field for UI/API
	// compatibility with future bundles.
	_ = skill
	return ""
}

func errorsIsUnique(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "unique")
}

type cleanupReadCloser struct {
	io.ReadCloser
	cleanup func()
}

func (r *cleanupReadCloser) Close() error {
	err := r.ReadCloser.Close()
	r.cleanup()
	return err
}

var _ interfaces.TenantSkillService = (*tenantSkillService)(nil)
