package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

var (
	ErrOrganizeTemplateRequired = errors.New("template_key is required")
	ErrOrganizeTemplateDisabled = errors.New("organize template is unavailable")
	ErrOrganizeConfigRequired   = errors.New("config_id is required")
	ErrOrganizeInvalidSchedule  = errors.New("invalid organize schedule")
	ErrOrganizeInvalidExpert    = errors.New("organize expert is unavailable")
	ErrOrganizeJobNotRetryable  = errors.New("organize job is not retryable")
	ErrOrganizeJobNotCancelable = errors.New("organize job is not cancelable")
)

const (
	organizeJobMemoryLimit  = 50
	organizeJobPromptBudget = 18000
)

var organizeTodoPattern = regexp.MustCompile(`(?m)^\s*(?:[-*]\s*(?:\[[ xX]?\]\s*)?|[0-9]+[.、]\s*)(?:待办|下一步|行动|跟进|建议)?`)

func (s *organizeService) ListTemplates(
	ctx context.Context,
	tenantID uint64,
	userID string,
) ([]*types.OrganizeTemplate, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	return s.repo.ListTemplates(ctx, tenantID, userID)
}

func (s *organizeService) GetTemplate(
	ctx context.Context,
	tenantID uint64,
	userID string,
	key string,
) (*types.OrganizeTemplate, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrOrganizeTemplateRequired
	}
	template, err := s.repo.GetTemplate(ctx, tenantID, userID, key)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, ErrOrganizeTemplateDisabled
	}
	return template, nil
}

func (s *organizeService) ListExperts(
	ctx context.Context,
	tenantID uint64,
	userID string,
) ([]types.OrganizeExpert, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	if s.expertPackages == nil {
		return []types.OrganizeExpert{}, nil
	}
	published, err := s.expertPackages.ListPublishedExperts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	experts := make([]types.OrganizeExpert, 0, len(published))
	for _, expert := range published {
		if expert == nil {
			continue
		}
		id := strings.TrimSpace(expert.AgentID)
		if id == "" {
			id = strings.TrimSpace(expert.DefinitionID)
		}
		if id == "" {
			continue
		}
		name := strings.TrimSpace(expert.DisplayName)
		if name == "" {
			name = id
		}
		experts = append(experts, types.OrganizeExpert{
			ID:          id,
			Name:        name,
			Description: strings.TrimSpace(expert.Description),
		})
	}
	return experts, nil
}

func (s *organizeService) CreateConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeConfigInput,
) (*types.OrganizeConfig, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	config, err := s.buildOrganizeConfig(ctx, tenantID, userID, "", input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateConfig(ctx, config); err != nil {
		return nil, err
	}
	return s.GetConfig(ctx, tenantID, userID, config.ID)
}

func (s *organizeService) GetConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeConfig, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	config, err := s.repo.GetConfig(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if config == nil {
		return nil, ErrOrganizeNotFound
	}
	return config, nil
}

func (s *organizeService) UpdateConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
	input types.OrganizeConfigInput,
) (*types.OrganizeConfig, error) {
	current, err := s.GetConfig(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	config, err := s.buildOrganizeConfig(ctx, tenantID, userID, current.ID, input)
	if err != nil {
		return nil, err
	}
	config.Status = current.Status
	config.LastRunAt = current.LastRunAt
	config.CreatedAt = current.CreatedAt
	config.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateConfig(ctx, config); err != nil {
		return nil, err
	}
	return s.GetConfig(ctx, tenantID, userID, current.ID)
}

func (s *organizeService) DeleteConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) error {
	if _, err := s.GetConfig(ctx, tenantID, userID, id); err != nil {
		return err
	}
	return s.repo.DeleteConfig(ctx, tenantID, userID, strings.TrimSpace(id))
}

func (s *organizeService) ListConfigs(
	ctx context.Context,
	query types.OrganizeConfigQuery,
) ([]*types.OrganizeConfig, int64, error) {
	if err := validateOrganizeScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Status = strings.TrimSpace(query.Status)
	query.Page, query.PageSize = normalizeOrganizePage(query.Page, query.PageSize)
	return s.repo.ListConfigs(ctx, query)
}

func (s *organizeService) RunConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
	input types.OrganizeJobInput,
) (*types.OrganizeJob, error) {
	input.ConfigID = strings.TrimSpace(id)
	return s.CreateJob(ctx, tenantID, userID, input)
}

func (s *organizeService) CreateJob(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeJobInput,
) (*types.OrganizeJob, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	configID := strings.TrimSpace(input.ConfigID)
	if configID == "" {
		return nil, ErrOrganizeConfigRequired
	}
	config, err := s.GetConfig(ctx, tenantID, userID, configID)
	if err != nil {
		return nil, err
	}
	template, err := s.GetTemplate(ctx, tenantID, userID, config.TemplateKey)
	if err != nil {
		return nil, err
	}

	memoryIDs, err := s.selectOrganizeJobMemoryIDs(ctx, tenantID, userID, input.MemoryIDs)
	if err != nil {
		return nil, err
	}
	experts, err := s.resolveOrganizeExpertSnapshots(ctx, tenantID, config.ExpertIDs)
	if err != nil {
		return nil, err
	}
	requirement := types.JSONMap{
		"config_name":     config.Name,
		"instruction":     config.Instruction,
		"expert_ids":      []string(config.ExpertIDs),
		"experts":         experts,
		"template_name":   template.Name,
		"template_scene":  template.Scene,
		"template_output": template.OutputLabel,
		"template_icon":   template.Icon,
		"template_spec":   template.Spec,
		"requested_text":  strings.TrimSpace(input.Requirement),
		"created_at":      time.Now().UTC().Format(time.RFC3339),
	}
	job := &types.OrganizeJob{
		TenantID:        tenantID,
		UserID:          userID,
		ConfigID:        config.ID,
		TemplateKey:     template.Key,
		TemplateVersion: template.PublishedVersion,
		Status:          types.OrganizeJobStatusQueued,
		Stage:           "queued",
		Progress:        5,
		Requirement:     requirement,
		MemoryIDs:       types.StringArray(memoryIDs),
		ModelID:         strings.TrimSpace(input.ModelID),
		Summary:         fmt.Sprintf("等待整理 %d 条记忆", len(memoryIDs)),
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	if err := s.enqueueOrganizeJob(ctx, job); err != nil {
		now := time.Now().UTC()
		job.Status = types.OrganizeJobStatusFailed
		job.Stage = "enqueue_failed"
		job.Progress = 100
		job.ErrorMessage = err.Error()
		job.FinishedAt = &now
		job.UpdatedAt = now
		_ = s.repo.UpdateJob(ctx, job)
		return nil, err
	}
	return s.GetJob(ctx, tenantID, userID, job.ID)
}

func (s *organizeService) GetJob(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeJob, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	job, err := s.repo.GetJob(ctx, tenantID, userID, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrOrganizeNotFound
	}
	return job, nil
}

func (s *organizeService) ListJobs(
	ctx context.Context,
	query types.OrganizeJobQuery,
) ([]*types.OrganizeJob, int64, error) {
	if err := validateOrganizeScope(query.TenantID, query.UserID); err != nil {
		return nil, 0, err
	}
	query.ConfigID = strings.TrimSpace(query.ConfigID)
	query.Status = strings.TrimSpace(query.Status)
	query.Page, query.PageSize = normalizeOrganizePage(query.Page, query.PageSize)
	return s.repo.ListJobs(ctx, query)
}

func (s *organizeService) RetryJob(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeJob, error) {
	job, err := s.GetJob(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if job.Status != types.OrganizeJobStatusFailed && job.Status != types.OrganizeJobStatusFallback {
		return nil, ErrOrganizeJobNotRetryable
	}
	requirement, _ := job.Requirement["requested_text"].(string)
	return s.CreateJob(ctx, tenantID, userID, types.OrganizeJobInput{
		ConfigID:    job.ConfigID,
		MemoryIDs:   append(types.StringArray(nil), job.MemoryIDs...),
		ModelID:     job.ModelID,
		Requirement: requirement,
	})
}

func (s *organizeService) CancelJob(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
) (*types.OrganizeJob, error) {
	job, err := s.GetJob(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if types.IsTerminalOrganizeJobStatus(job.Status) {
		return nil, ErrOrganizeJobNotCancelable
	}
	now := time.Now().UTC()
	job.Status = types.OrganizeJobStatusCanceled
	job.Stage = "canceled"
	job.Progress = 100
	job.Summary = "任务已取消"
	job.FinishedAt = &now
	job.UpdatedAt = now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, err
	}
	return s.GetJob(ctx, tenantID, userID, job.ID)
}

func (s *organizeService) ProcessOrganizeJob(ctx context.Context, task *asynq.Task) error {
	var payload types.OrganizeJobTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode organize job task: %w", err)
	}
	if payload.TenantID == 0 || strings.TrimSpace(payload.UserID) == "" || strings.TrimSpace(payload.JobID) == "" {
		return errors.New("invalid organize job payload")
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)
	ctx = context.WithValue(ctx, types.UserIDContextKey, payload.UserID)

	job, err := s.repo.GetJob(ctx, payload.TenantID, payload.UserID, payload.JobID)
	if err != nil || job == nil {
		return err
	}
	if types.IsTerminalOrganizeJobStatus(job.Status) {
		return nil
	}
	now := time.Now().UTC()
	job.Status = types.OrganizeJobStatusRunning
	job.Stage = "loading_memories"
	job.Progress = 20
	job.StartedAt = &now
	job.ErrorMessage = ""
	job.UpdatedAt = now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return err
	}

	output, fallback, err := s.executeOrganizeJob(ctx, job)
	if err != nil {
		return s.failOrganizeJob(ctx, job, err)
	}

	latest, err := s.repo.GetJob(ctx, job.TenantID, job.UserID, job.ID)
	if err != nil {
		return err
	}
	if latest == nil || latest.Status == types.OrganizeJobStatusCanceled {
		return nil
	}
	finishedAt := time.Now().UTC()
	job.OutputID = output.ID
	job.Status = types.OrganizeJobStatusCompleted
	if fallback {
		job.Status = types.OrganizeJobStatusFallback
	}
	job.Stage = "completed"
	job.Progress = 100
	job.Summary = output.SourceSummary
	job.Result = types.JSONMap{
		"output_id":        output.ID,
		"memory_count":     len(job.MemoryIDs),
		"conclusion_count": metadataInt(output.Metadata, "conclusion_count"),
		"todo_count":       metadataInt(output.Metadata, "todo_count"),
		"fallback":         fallback,
	}
	job.FinishedAt = &finishedAt
	job.UpdatedAt = finishedAt
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return err
	}

	config, err := s.repo.GetConfig(ctx, job.TenantID, job.UserID, job.ConfigID)
	if err == nil && config != nil {
		config.LastRunAt = &finishedAt
		config.UpdatedAt = finishedAt
		_ = s.repo.UpdateConfig(ctx, config)
	}
	return nil
}

func (s *organizeService) RunDueConfigs(ctx context.Context, now time.Time) error {
	configs, err := s.repo.ListDueConfigs(ctx, now.UTC(), 100)
	if err != nil {
		return err
	}
	var joined error
	for _, config := range configs {
		scheduledFor := config.NextRunAt
		nextRunAt := nextOrganizeRun(config.Schedule, now)
		config.NextRunAt = nextRunAt
		config.UpdatedAt = now.UTC()
		if err := s.repo.UpdateConfig(ctx, config); err != nil {
			joined = errors.Join(joined, err)
			continue
		}
		if scheduledFor == nil {
			continue
		}
		jobCtx := context.WithValue(ctx, types.TenantIDContextKey, config.TenantID)
		jobCtx = context.WithValue(jobCtx, types.UserIDContextKey, config.UserID)
		job, createErr := s.createScheduledOrganizeJob(jobCtx, config, scheduledFor.UTC())
		if createErr != nil {
			joined = errors.Join(joined, createErr)
			continue
		}
		if job != nil {
			logger.Infof(ctx, "[OrganizeScheduler] enqueued config=%s job=%s", config.ID, job.ID)
		}
	}
	return joined
}

func (s *organizeService) buildOrganizeConfig(
	ctx context.Context,
	tenantID uint64,
	userID string,
	id string,
	input types.OrganizeConfigInput,
) (*types.OrganizeConfig, error) {
	name, err := normalizeTitle(input.Name)
	if err != nil {
		return nil, err
	}
	template, err := s.GetTemplate(ctx, tenantID, userID, input.TemplateKey)
	if err != nil {
		return nil, err
	}
	schedule := strings.TrimSpace(input.Schedule)
	if schedule == "" {
		schedule = types.OrganizeScheduleManual
	}
	if !types.IsValidOrganizeSchedule(schedule) {
		return nil, ErrOrganizeInvalidSchedule
	}
	instruction := strings.TrimSpace(input.Instruction)
	if instruction == "" {
		instruction = template.DefaultInstruction
	}
	expertIDs := cleanStringArray(input.ExpertIDs, 20, 128)
	if _, err := s.resolveOrganizeExpertSnapshots(ctx, tenantID, expertIDs); err != nil {
		return nil, err
	}
	return &types.OrganizeConfig{
		ID:          id,
		TenantID:    tenantID,
		UserID:      userID,
		Name:        name,
		TemplateKey: template.Key,
		Instruction: instruction,
		ExpertIDs:   expertIDs,
		Schedule:    schedule,
		Status:      types.OrganizeConfigStatusActive,
		NextRunAt:   nextOrganizeRun(schedule, time.Now()),
		Metadata:    normalizeJSONMap(input.Metadata),
	}, nil
}

func (s *organizeService) selectOrganizeJobMemoryIDs(
	ctx context.Context,
	tenantID uint64,
	userID string,
	requested []string,
) ([]string, error) {
	if len(requested) > 0 {
		return s.validateMemoryIDs(ctx, tenantID, userID, requested)
	}
	memories, _, err := s.repo.ListMemories(ctx, types.OrganizeListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PageSize: organizeJobMemoryLimit,
	})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}
	return ids, nil
}

func (s *organizeService) enqueueOrganizeJob(ctx context.Context, job *types.OrganizeJob) error {
	payload, err := json.Marshal(types.OrganizeJobTaskPayload{
		TenantID: job.TenantID,
		UserID:   job.UserID,
		JobID:    job.ID,
	})
	if err != nil {
		return err
	}
	task := asynq.NewTask(types.TypeOrganizeJobRun, payload)
	if s.taskEnqueuer == nil {
		return s.ProcessOrganizeJob(context.WithoutCancel(ctx), task)
	}
	_, err = s.taskEnqueuer.Enqueue(
		task,
		asynq.Queue(types.QueueAgent),
		asynq.MaxRetry(2),
		asynq.Timeout(15*time.Minute),
		asynq.TaskID("organize-job:"+job.ID),
	)
	return err
}

func (s *organizeService) executeOrganizeJob(
	ctx context.Context,
	job *types.OrganizeJob,
) (*types.OrganizeOutput, bool, error) {
	memories, err := s.repo.ListMemoriesByIDs(ctx, job.TenantID, job.UserID, job.MemoryIDs)
	if err != nil {
		return nil, false, err
	}
	job.Stage = "generating"
	job.Progress = 55
	job.Summary = fmt.Sprintf("正在整理 %d 条记忆", len(memories))
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, false, err
	}

	prompt := buildOrganizeJobPrompt(job, memories)
	hash := sha256.Sum256([]byte(prompt))
	job.PromptHash = hex.EncodeToString(hash[:])
	content, modelID, fallback := s.generateOrganizeJobContent(ctx, job, prompt, memories)
	job.ModelID = modelID
	job.Stage = "saving_output"
	job.Progress = 85
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, false, err
	}

	configName, _ := job.Requirement["config_name"].(string)
	outputLabel, _ := job.Requirement["template_output"].(string)
	icon, _ := job.Requirement["template_icon"].(string)
	if outputLabel == "" {
		outputLabel = "整理结果"
	}
	title := strings.TrimSpace(configName)
	if title == "" {
		title = outputLabel
	}
	title = fmt.Sprintf("%s · %s", title, time.Now().In(organizeScheduleLocation()).Format("2006-01-02"))
	summary := firstOrganizeSummaryLine(content)
	todoCount := countOrganizeTodos(content)
	conclusionCount := countOrganizeConclusions(content)
	fields := types.JSONMap{
		"主题":   strings.TrimSpace(configName),
		"模板":   job.TemplateKey,
		"模板版本": job.TemplateVersion,
		"涉及时间": organizeMemoryDateRange(memories),
		"待办数":  todoCount,
		"来源数":  len(memories),
	}
	citations := types.JSONMap{"memory_refs": organizeMemoryCitations(memories)}
	metadata := types.JSONMap{
		"generated_by":     "organize_job",
		"ai_status":        map[bool]string{true: "fallback", false: "completed"}[fallback],
		"conclusion_count": conclusionCount,
		"todo_count":       todoCount,
		"tags":             organizeTemplateStringList(job.Requirement, "template_spec", "tags"),
		"scene":            stringValue(job.Requirement, "template_scene"),
		"expert_ids":       job.Requirement["expert_ids"],
	}
	output := &types.OrganizeOutput{
		TenantID:        job.TenantID,
		UserID:          job.UserID,
		ConfigID:        job.ConfigID,
		JobID:           job.ID,
		TemplateKey:     job.TemplateKey,
		TemplateVersion: job.TemplateVersion,
		Title:           title,
		OutputType:      outputLabel,
		Content:         content,
		SourceSummary:   summary,
		Status:          types.OrganizeOutputStatusReady,
		Icon:            icon,
		Fields:          fields,
		Citations:       citations,
		Metadata:        metadata,
	}
	if err := s.repo.CreateOutput(ctx, output, job.MemoryIDs); err != nil {
		return nil, false, err
	}
	created, err := s.repo.GetOutput(ctx, job.TenantID, job.UserID, output.ID)
	return created, fallback, err
}

func (s *organizeService) generateOrganizeJobContent(
	ctx context.Context,
	job *types.OrganizeJob,
	prompt string,
	memories []*types.OrganizeMemory,
) (string, string, bool) {
	modelID := strings.TrimSpace(job.ModelID)
	if modelID == "" {
		modelID = s.resolveOrganizeModelID(ctx, types.ModelTypeKnowledgeQA)
	}
	if modelID == "" || s.modelService == nil {
		return fallbackOrganizeJobContent(job, memories, "AI 模型未配置"), modelID, true
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return fallbackOrganizeJobContent(job, memories, "AI 模型不可用"), modelID, true
	}
	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是整理工作台的执行引擎。只能依据输入记忆生成中文 Markdown，必须保留可核验的事实边界，不得编造数据。引用原始记忆时使用 [M1]、[M2] 这样的标记。",
		},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.25,
		MaxTokens:   2600,
		Thinking:    &thinking,
	})
	if err != nil || response == nil || strings.TrimSpace(response.Content) == "" {
		return fallbackOrganizeJobContent(job, memories, "AI 生成失败"), modelID, true
	}
	return strings.TrimSpace(response.Content), modelID, false
}

func (s *organizeService) failOrganizeJob(
	ctx context.Context,
	job *types.OrganizeJob,
	jobErr error,
) error {
	now := time.Now().UTC()
	job.Status = types.OrganizeJobStatusFailed
	job.Stage = "failed"
	job.Progress = 100
	job.ErrorMessage = trimMax(jobErr.Error(), 4000)
	job.Summary = "整理失败"
	job.FinishedAt = &now
	job.UpdatedAt = now
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return errors.Join(jobErr, err)
	}
	return jobErr
}

func (s *organizeService) createScheduledOrganizeJob(
	ctx context.Context,
	config *types.OrganizeConfig,
	scheduledFor time.Time,
) (*types.OrganizeJob, error) {
	template, err := s.GetTemplate(ctx, config.TenantID, config.UserID, config.TemplateKey)
	if err != nil {
		return nil, err
	}
	memoryIDs, err := s.selectOrganizeJobMemoryIDs(ctx, config.TenantID, config.UserID, nil)
	if err != nil {
		return nil, err
	}
	experts, err := s.resolveOrganizeExpertSnapshots(ctx, config.TenantID, config.ExpertIDs)
	if err != nil {
		return nil, err
	}
	job := &types.OrganizeJob{
		TenantID:        config.TenantID,
		UserID:          config.UserID,
		ConfigID:        config.ID,
		TemplateKey:     template.Key,
		TemplateVersion: template.PublishedVersion,
		Status:          types.OrganizeJobStatusQueued,
		Stage:           "queued",
		Progress:        5,
		Requirement: types.JSONMap{
			"config_name":     config.Name,
			"instruction":     config.Instruction,
			"expert_ids":      []string(config.ExpertIDs),
			"experts":         experts,
			"template_name":   template.Name,
			"template_scene":  template.Scene,
			"template_output": template.OutputLabel,
			"template_icon":   template.Icon,
			"template_spec":   template.Spec,
			"scheduled":       true,
		},
		MemoryIDs:    types.StringArray(memoryIDs),
		Summary:      fmt.Sprintf("周期任务等待整理 %d 条记忆", len(memoryIDs)),
		DedupeKey:    fmt.Sprintf("scheduled:%s:%s", config.ID, scheduledFor.UTC().Format(time.RFC3339)),
		ScheduledFor: &scheduledFor,
	}
	if err := s.repo.CreateJob(ctx, job); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") ||
			strings.Contains(strings.ToLower(err.Error()), "duplicate") {
			return nil, nil
		}
		return nil, err
	}
	if err := s.enqueueOrganizeJob(ctx, job); err != nil {
		return nil, err
	}
	return job, nil
}

func normalizeOrganizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = organizeDefaultPage
	}
	if pageSize < 1 {
		pageSize = organizeDefaultPageSize
	}
	if pageSize > organizeMaxPageSize {
		pageSize = organizeMaxPageSize
	}
	return page, pageSize
}

func nextOrganizeRun(schedule string, from time.Time) *time.Time {
	if schedule == "" || schedule == types.OrganizeScheduleManual {
		return nil
	}
	location := organizeScheduleLocation()
	local := from.In(location)
	var next time.Time
	switch schedule {
	case types.OrganizeScheduleDaily:
		next = time.Date(local.Year(), local.Month(), local.Day(), 8, 0, 0, 0, location)
		if !next.After(local) {
			next = next.AddDate(0, 0, 1)
		}
	case types.OrganizeScheduleWeekly:
		days := (int(time.Monday) - int(local.Weekday()) + 7) % 7
		next = time.Date(local.Year(), local.Month(), local.Day()+days, 9, 0, 0, 0, location)
		if !next.After(local) {
			next = next.AddDate(0, 0, 7)
		}
	case types.OrganizeScheduleMonthly:
		next = time.Date(local.Year(), local.Month(), 1, 9, 0, 0, 0, location)
		if !next.After(local) {
			next = next.AddDate(0, 1, 0)
		}
	default:
		return nil
	}
	utc := next.UTC()
	return &utc
}

func organizeScheduleLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.Local
	}
	return location
}

func buildOrganizeJobPrompt(job *types.OrganizeJob, memories []*types.OrganizeMemory) string {
	instruction := stringValue(job.Requirement, "instruction")
	requested := stringValue(job.Requirement, "requested_text")
	spec, _ := json.Marshal(job.Requirement["template_spec"])
	experts, _ := json.Marshal(job.Requirement["experts"])
	var builder strings.Builder
	fmt.Fprintf(&builder, "整理名称：%s\n", stringValue(job.Requirement, "config_name"))
	fmt.Fprintf(&builder, "模板：%s（%s）\n", stringValue(job.Requirement, "template_name"), job.TemplateVersion)
	fmt.Fprintf(&builder, "模板结构：%s\n\n", string(spec))
	if len(experts) > 0 && string(experts) != "null" && string(experts) != "[]" {
		fmt.Fprintf(&builder, "参与专家：%s\n\n", string(experts))
	}
	fmt.Fprintf(&builder, "用户指令：\n%s\n", instruction)
	if requested != "" {
		fmt.Fprintf(&builder, "\n本次补充要求：\n%s\n", requested)
	}
	builder.WriteString("\n输出要求：\n")
	builder.WriteString("- 输出可直接阅读的中文 Markdown。\n")
	builder.WriteString("- 先给主题和核心结论，再给证据、待办或模板要求的结构。\n")
	builder.WriteString("- 每条关键结论必须用 [M1] 形式标注来源；没有依据的字段明确写“记录中未提供”。\n")
	builder.WriteString("- 待办使用 Markdown 清单 `- [ ]`。\n\n")
	builder.WriteString("原始记忆：\n")
	for index, memory := range memories {
		content := strings.TrimSpace(memory.Content)
		if content == "" {
			content = memory.Title
		}
		fmt.Fprintf(
			&builder,
			"\n[M%d] 标题：%s\n类型：%s\n时间：%s\n来源：%s\n内容：%s\n",
			index+1,
			memory.Title,
			memory.Kind,
			memory.OccurredAt.In(organizeScheduleLocation()).Format("2006-01-02 15:04"),
			emptyFallback(memory.Source, "手动记录"),
			sampleRunes(content, 2200, "…"),
		)
		if builder.Len() >= organizeJobPromptBudget {
			break
		}
	}
	if len(memories) == 0 {
		builder.WriteString("\n当前没有可用记忆。请输出缺少输入的说明和后续补充建议，不得生成虚构内容。\n")
	}
	return sampleRunes(builder.String(), organizeJobPromptBudget, "…")
}

func (s *organizeService) resolveOrganizeExpertSnapshots(
	ctx context.Context,
	tenantID uint64,
	expertIDs []string,
) ([]types.JSONMap, error) {
	if len(expertIDs) == 0 {
		return []types.JSONMap{}, nil
	}
	if s.expertPackages == nil {
		return nil, ErrOrganizeInvalidExpert
	}
	published, err := s.expertPackages.ListPublishedExperts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	available := make(map[string]*types.PublishedExpert, len(published)*2)
	for _, expert := range published {
		if expert == nil {
			continue
		}
		if id := strings.TrimSpace(expert.AgentID); id != "" {
			available[id] = expert
		}
		if id := strings.TrimSpace(expert.DefinitionID); id != "" {
			available[id] = expert
		}
	}
	snapshots := make([]types.JSONMap, 0, len(expertIDs))
	for _, expertID := range expertIDs {
		expertID = strings.TrimSpace(expertID)
		expert := available[expertID]
		if expert == nil {
			return nil, fmt.Errorf("%w: %s", ErrOrganizeInvalidExpert, expertID)
		}
		snapshots = append(snapshots, types.JSONMap{
			"id":          expertID,
			"name":        strings.TrimSpace(expert.DisplayName),
			"description": strings.TrimSpace(expert.Description),
			"domain":      strings.TrimSpace(expert.Domain),
			"package":     strings.TrimSpace(expert.PackageDisplayName),
			"version":     strings.TrimSpace(expert.Version),
		})
	}
	return snapshots, nil
}

func fallbackOrganizeJobContent(
	job *types.OrganizeJob,
	memories []*types.OrganizeMemory,
	reason string,
) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# %s\n\n", emptyFallback(stringValue(job.Requirement, "config_name"), "整理结果"))
	fmt.Fprintf(&builder, "> %s，已按原始记忆生成基础整理，未补充输入中不存在的事实。\n\n", reason)
	builder.WriteString("## 01. 原始记忆摘要\n\n")
	if len(memories) == 0 {
		builder.WriteString("当前没有可用记忆，暂时无法形成事实性结论。\n\n")
	} else {
		for index, memory := range memories {
			content := strings.TrimSpace(memory.Content)
			if content == "" {
				content = memory.Title
			}
			fmt.Fprintf(&builder, "- **%s**：%s [M%d]\n", memory.Title, sampleRunes(content, 240, "…"), index+1)
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## 02. 当前可确认的信息\n\n")
	builder.WriteString("以上内容均来自已选记忆；需要模型归纳、数据计算或人工判断的部分尚未完成。\n\n")
	builder.WriteString("## 03. 下一步\n\n")
	builder.WriteString("- [ ] 检查原始记忆是否完整。\n")
	builder.WriteString("- [ ] 配置可用模型后重新执行本次整理。\n")
	return builder.String()
}

func firstOrganizeSummaryLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#>-* "))
		if line != "" {
			return trimMax(line, organizeMaxShortText)
		}
	}
	return "整理已完成"
}

func countOrganizeTodos(content string) int {
	count := strings.Count(content, "- [ ]") + strings.Count(content, "- [x]") + strings.Count(content, "- [X]")
	if count > 0 {
		return count
	}
	return len(organizeTodoPattern.FindAllString(content, -1))
}

func countOrganizeConclusions(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") && !strings.Contains(trimmed, "待办") && !strings.Contains(trimmed, "下一步") {
			count++
		}
	}
	if count == 0 && strings.TrimSpace(content) != "" {
		return 1
	}
	return count
}

func organizeMemoryDateRange(memories []*types.OrganizeMemory) string {
	if len(memories) == 0 {
		return "无可用记忆"
	}
	start := memories[0].OccurredAt
	end := memories[0].OccurredAt
	for _, memory := range memories[1:] {
		if memory.OccurredAt.Before(start) {
			start = memory.OccurredAt
		}
		if memory.OccurredAt.After(end) {
			end = memory.OccurredAt
		}
	}
	location := organizeScheduleLocation()
	if start.In(location).Format("2006-01-02") == end.In(location).Format("2006-01-02") {
		return start.In(location).Format("2006-01-02")
	}
	return start.In(location).Format("2006-01-02") + " 至 " + end.In(location).Format("2006-01-02")
}

func organizeMemoryCitations(memories []*types.OrganizeMemory) []types.JSONMap {
	refs := make([]types.JSONMap, 0, len(memories))
	for index, memory := range memories {
		refs = append(refs, types.JSONMap{
			"label":  fmt.Sprintf("M%d", index+1),
			"id":     memory.ID,
			"kind":   memory.Kind,
			"title":  memory.Title,
			"source": memory.Source,
		})
	}
	return refs
}

func organizeTemplateStringList(root types.JSONMap, objectKey, field string) []string {
	object, ok := root[objectKey].(types.JSONMap)
	if !ok {
		if raw, isMap := root[objectKey].(map[string]any); isMap {
			object = types.JSONMap(raw)
		}
	}
	value := object[field]
	switch items := value.(type) {
	case []string:
		return items
	case types.StringArray:
		return []string(items)
	case []any:
		out := make([]string, 0, len(items))
		for _, item := range items {
			if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		return []string{}
	}
}

func metadataInt(metadata types.JSONMap, key string) int {
	switch value := metadata[key].(type) {
	case int:
		return value
	case int64:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}

func stringValue(values types.JSONMap, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}
