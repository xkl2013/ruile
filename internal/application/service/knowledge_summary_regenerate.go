package service

import (
	"context"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

func (s *knowledgeService) RegenerateKnowledgeSummary(
	ctx context.Context,
	knowledgeID string,
	mode types.KnowledgeSummaryRegenerateMode,
) (*types.Knowledge, error) {
	resolvedMode, err := normalizeSummaryRegenerateMode(mode)
	if err != nil {
		return nil, err
	}

	tenantID := ctx.Value(types.TenantIDContextKey).(uint64)
	knowledge, err := s.repo.GetKnowledgeByIDOnly(ctx, knowledgeID)
	if err != nil {
		return nil, err
	}
	if knowledge == nil {
		return nil, errors.NewNotFoundError("知识不存在")
	}
	if knowledge.TenantID != tenantID {
		return nil, errors.NewForbiddenError("no access to this knowledge")
	}

	switch knowledge.ParseStatus {
	case types.ParseStatusPending, types.ParseStatusProcessing, types.ParseStatusFinalizing, types.ParseStatusDeleting, types.ParseStatusCancelled:
		return nil, errors.NewConflictError("当前文档正在处理中，请稍后再试")
	}

	if knowledge.SummaryStatus == types.SummaryStatusPending || knowledge.SummaryStatus == types.SummaryStatusProcessing {
		return nil, errors.NewConflictError("摘要正在生成中，请稍后再试")
	}

	kb, err := s.kbService.GetKnowledgeBaseByID(ctx, knowledge.KnowledgeBaseID)
	if err != nil {
		return nil, err
	}
	if kb == nil {
		return nil, errors.NewNotFoundError("知识库不存在")
	}

	summaryChunks, err := listSummaryInputChunks(ctx, s.chunkService, knowledge.ID)
	if err != nil {
		return nil, err
	}
	summaryContent := buildSummaryInputContent(ctx, s.chunkRepo, knowledge.TenantID, summaryChunks)
	summaryReady := strings.TrimSpace(summaryContent) != "" &&
		checkSufficientSummaryContent(ctx, knowledge.ID, summaryContent) == nil

	logger.Infof(ctx, "[KnowledgeSummary] Regenerate request for %s mode=%s summary_ready=%t", knowledge.ID, resolvedMode, summaryReady)

	if resolvedMode == types.KnowledgeSummaryRegenerateModeAuto && !summaryReady {
		if knowledge.IsManual() {
			return nil, errors.NewBadRequestError("当前文件缺少可用于摘要的内容，请先重新解析")
		}
		currentOverrides := buildCurrentReparseOverrides(kb, knowledge)
		return s.ReparseKnowledge(ctx, knowledge.ID, currentOverrides)
	}

	if !summaryReady {
		return nil, errors.NewBadRequestError("当前文件缺少可用于摘要的内容，请先重新解析")
	}

	now := time.Now()
	prevSummaryStatus := knowledge.SummaryStatus
	prevUpdatedAt := knowledge.UpdatedAt
	if err := s.repo.UpdateKnowledgeColumns(ctx, knowledge.ID, map[string]interface{}{
		"summary_status": types.SummaryStatusPending,
		"updated_at":     now,
	}); err != nil {
		return nil, err
	}

	lang, _ := types.LanguageFromContext(ctx)
	taskPayload := types.SummaryGenerationPayload{
		TenantID:        knowledge.TenantID,
		KnowledgeBaseID: knowledge.KnowledgeBaseID,
		KnowledgeID:     knowledge.ID,
		Language:        lang,
		Attempt:         s.tracker().LatestAttempt(ctx, knowledge.ID),
	}
	if !enqueueSummaryGenerationTask(ctx, s.task, taskPayload) {
		revertErr := s.repo.UpdateKnowledgeColumns(ctx, knowledge.ID, map[string]interface{}{
			"summary_status": prevSummaryStatus,
			"updated_at":     prevUpdatedAt,
		})
		if revertErr != nil {
			logger.Warnf(ctx, "[KnowledgeSummary] Failed to revert summary status for %s after enqueue failure: %v", knowledge.ID, revertErr)
		}
		return nil, errors.NewInternalServerError("failed to enqueue summary generation task")
	}

	knowledge.SummaryStatus = types.SummaryStatusPending
	knowledge.UpdatedAt = now
	return knowledge, nil
}

func normalizeSummaryRegenerateMode(mode types.KnowledgeSummaryRegenerateMode) (types.KnowledgeSummaryRegenerateMode, error) {
	switch mode {
	case "", types.KnowledgeSummaryRegenerateModeAuto:
		return types.KnowledgeSummaryRegenerateModeAuto, nil
	case types.KnowledgeSummaryRegenerateModeSummaryOnly:
		return types.KnowledgeSummaryRegenerateModeSummaryOnly, nil
	default:
		return "", errors.NewBadRequestError("无效的摘要重生成模式")
	}
}

func buildCurrentReparseOverrides(kb *types.KnowledgeBase, knowledge *types.Knowledge) *types.KnowledgeProcessOverrides {
	if kb == nil {
		return nil
	}
	overrides := effectiveProcessConfigToOverrides(ResolveProcessConfig(kb, nil))
	if knowledge == nil {
		return overrides
	}
	stored, err := knowledge.ProcessOverrides()
	if err != nil || stored == nil || len(stored.ParserEngineOverrides) == 0 {
		return overrides
	}
	overrides.ParserEngineOverrides = cloneStringMap(stored.ParserEngineOverrides)
	return overrides
}

func cloneStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
