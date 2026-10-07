package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *organizeService) executeBatchOrganizeJob(
	ctx context.Context,
	job *types.OrganizeJob,
) (*types.OrganizeOutput, bool, error) {
	batches, err := s.repo.ListJobBatches(ctx, job.TenantID, job.UserID, job.ID)
	if err != nil {
		return nil, false, err
	}
	if len(batches) == 0 {
		return nil, false, errorsNewOrganizeBatch("batch organize job has no batches")
	}

	job.Stage = "processing_batches"
	job.Progress = 20
	job.Summary = fmt.Sprintf("正在处理 0/%d 批记忆", len(batches))
	job.ProcessedCount = 0
	job.FailedCount = job.SelectedCount - job.ReadyCount
	if job.FailedCount < 0 {
		job.FailedCount = 0
	}
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, false, err
	}

	anyFallback := false
	successful := make([]*types.OrganizeJobBatch, 0, len(batches))
	for index, batch := range batches {
		if batch.Status == types.OrganizeJobBatchStatusCompleted ||
			batch.Status == types.OrganizeJobBatchStatusFallback {
			successful = append(successful, batch)
			job.ProcessedCount += len(batch.MemoryIDs)
			if batch.Status == types.OrganizeJobBatchStatusFallback {
				anyFallback = true
			}
			continue
		}
		if batch.Status == types.OrganizeJobBatchStatusCanceled {
			job.FailedCount += len(batch.MemoryIDs)
			continue
		}

		startedAt := time.Now().UTC()
		batch.Status = types.OrganizeJobBatchStatusRunning
		batch.Stage = "generating"
		batch.Progress = 20
		batch.StartedAt = &startedAt
		batch.ErrorMessage = ""
		batch.UpdatedAt = startedAt
		if err := s.repo.UpdateJobBatch(ctx, batch); err != nil {
			return nil, false, err
		}

		memories, loadErr := s.repo.ListMemoriesByIDs(
			ctx,
			job.TenantID,
			job.UserID,
			batch.MemoryIDs,
		)
		if loadErr != nil {
			if err := s.failOrganizeJobBatch(ctx, batch, loadErr); err != nil {
				return nil, false, err
			}
			job.FailedCount += len(batch.MemoryIDs)
			anyFallback = true
			continue
		}

		prompt := buildOrganizeJobPrompt(job, memories)
		hash := sha256.Sum256([]byte(prompt))
		batch.PromptHash = hex.EncodeToString(hash[:])
		content, modelID, fallback := s.generateOrganizeJobContent(ctx, job, prompt, memories)
		job.ModelID = modelID
		content = normalizeOrganizeGeneratedMarkdown(
			content,
			stringValue(job.Requirement, "template_markdown"),
			stringValue(job.Requirement, "config_name"),
			organizeJSONMapValue(job.Requirement["template_spec"]),
		)
		finishedAt := time.Now().UTC()
		batch.Summary = content
		batch.Citations = types.JSONMap{"memory_refs": organizeMemoryCitations(memories)}
		batch.StructuredResult = types.JSONMap{
			"summary":          firstOrganizeSummaryLine(content),
			"conclusion_count": countOrganizeConclusions(content),
			"todo_count":       countOrganizeTodos(content),
			"fallback":         fallback,
		}
		batch.Status = types.OrganizeJobBatchStatusCompleted
		if fallback {
			batch.Status = types.OrganizeJobBatchStatusFallback
			anyFallback = true
		}
		batch.Stage = "completed"
		batch.Progress = 100
		batch.FinishedAt = &finishedAt
		batch.UpdatedAt = finishedAt
		if err := s.repo.UpdateJobBatch(ctx, batch); err != nil {
			return nil, false, err
		}
		successful = append(successful, batch)
		job.ProcessedCount += len(batch.MemoryIDs)
		job.Progress = 20 + ((index + 1) * 55 / len(batches))
		job.Summary = fmt.Sprintf("正在处理 %d/%d 批记忆", index+1, len(batches))
		job.Coverage = organizeJobCoverage(job)
		job.UpdatedAt = finishedAt
		if err := s.repo.UpdateJob(ctx, job); err != nil {
			return nil, false, err
		}
	}

	if len(successful) == 0 {
		return nil, false, errorsNewOrganizeBatch("all organize batches failed")
	}
	job.Stage = "merging"
	job.Progress = 80
	job.Summary = fmt.Sprintf("正在合并 %d 批整理结果", len(successful))
	job.Coverage = organizeJobCoverage(job)
	job.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateJob(ctx, job); err != nil {
		return nil, false, err
	}

	content, fallback := s.mergeOrganizeBatchContent(ctx, job, successful)
	if fallback {
		anyFallback = true
	}
	processedMemoryIDs := make([]string, 0, job.ProcessedCount)
	for _, batch := range successful {
		processedMemoryIDs = append(processedMemoryIDs, batch.MemoryIDs...)
	}
	memories, err := s.repo.ListMemoriesByIDs(ctx, job.TenantID, job.UserID, processedMemoryIDs)
	if err != nil {
		return nil, false, err
	}
	job.Coverage = organizeJobCoverage(job)
	return s.saveOrganizeJobOutput(ctx, job, memories, content, anyFallback)
}

func (s *organizeService) failOrganizeJobBatch(
	ctx context.Context,
	batch *types.OrganizeJobBatch,
	batchErr error,
) error {
	finishedAt := time.Now().UTC()
	batch.Status = types.OrganizeJobBatchStatusFailed
	batch.Stage = "failed"
	batch.Progress = 100
	batch.ErrorMessage = batchErr.Error()
	batch.FinishedAt = &finishedAt
	batch.UpdatedAt = finishedAt
	return s.repo.UpdateJobBatch(ctx, batch)
}

func organizeJobCoverage(job *types.OrganizeJob) types.JSONMap {
	total := job.SelectedCount
	if total == 0 {
		total = len(job.MemoryIDs)
	}
	ratio := 0.0
	if total > 0 {
		ratio = float64(job.ProcessedCount) / float64(total)
	}
	return types.JSONMap{
		"selected":  total,
		"ready":     job.ReadyCount,
		"processed": job.ProcessedCount,
		"failed":    job.FailedCount,
		"ratio":     ratio,
	}
}

func (s *organizeService) mergeOrganizeBatchContent(
	ctx context.Context,
	job *types.OrganizeJob,
	batches []*types.OrganizeJobBatch,
) (string, bool) {
	if len(batches) == 1 {
		return batches[0].Summary, batches[0].Status == types.OrganizeJobBatchStatusFallback
	}
	var builder strings.Builder
	builder.WriteString("请把以下批次整理结果合并为一份最终中文 Markdown。\n")
	builder.WriteString("要求：去除重复事实；保留冲突项并放入“待确认”；保留来源引用；不得补充输入中不存在的事实。\n\n")
	for _, batch := range batches {
		fmt.Fprintf(&builder, "## 批次 %d/%d\n\n%s\n\n", batch.BatchIndex, batch.BatchCount, batch.Summary)
	}
	prompt := builder.String()
	if len([]rune(prompt)) > organizeJobPromptBudget {
		prompt = sampleRunes(prompt, organizeJobPromptBudget, "…")
	}
	modelID := strings.TrimSpace(job.ModelID)
	if modelID == "" {
		modelID = s.resolveOrganizeModelID(ctx, types.ModelTypeKnowledgeQA)
	}
	if modelID == "" || s.modelService == nil {
		return fallbackOrganizeBatchMerge(job, batches, "AI 模型未配置"), true
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return fallbackOrganizeBatchMerge(job, batches, "AI 模型不可用"), true
	}
	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{
			Role:    "system",
			Content: "你是整理工作台的合并引擎。只能合并提供的批次结果，必须去重、保留冲突和来源引用，不得编造事实。",
		},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{Thinking: &thinking})
	if err != nil || response == nil || strings.TrimSpace(response.Content) == "" {
		return fallbackOrganizeBatchMerge(job, batches, "批次合并模型调用失败"), true
	}
	return strings.TrimSpace(response.Content), false
}

func fallbackOrganizeBatchMerge(
	job *types.OrganizeJob,
	batches []*types.OrganizeJobBatch,
	reason string,
) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "# %s\n\n", emptyFallback(stringValue(job.Requirement, "config_name"), "整理结果"))
	fmt.Fprintf(&builder, "> %s，以下内容按批次保留，未静默丢弃已完成结果。\n\n", reason)
	for _, batch := range batches {
		fmt.Fprintf(&builder, "## 批次 %d\n\n%s\n\n", batch.BatchIndex, batch.Summary)
	}
	builder.WriteString("## 覆盖情况\n\n")
	fmt.Fprintf(
		&builder,
		"- 已处理 %d / %d 条记忆\n- 失败或未就绪 %d 条\n",
		job.ProcessedCount,
		job.SelectedCount,
		job.FailedCount,
	)
	return builder.String()
}

func errorsNewOrganizeBatch(message string) error {
	return fmt.Errorf("organize batch: %s", message)
}
