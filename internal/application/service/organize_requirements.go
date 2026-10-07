package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

func (s *organizeService) PreviewOrganizeRequirement(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeRequirementInput,
) (*types.OrganizeRequirementPreview, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	text := trimMax(input.Text, 4000)
	if text == "" {
		return nil, ErrOrganizeRequirementTextRequired
	}
	configID := strings.TrimSpace(input.ConfigID)
	if configID == "" {
		return nil, ErrOrganizeConfigRequired
	}
	config, err := s.GetConfig(ctx, tenantID, userID, configID)
	if err != nil {
		return nil, err
	}

	templateKey := strings.TrimSpace(input.TemplateKey)
	if templateKey == "" {
		templateKey = config.TemplateKey
	}
	template, err := s.GetTemplate(ctx, tenantID, userID, templateKey)
	if err != nil {
		return nil, err
	}

	ambiguities := make([]string, 0, 2)
	suggestions := make([]string, 0, 3)
	warnings := make([]string, 0, 3)
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "还是") ||
		strings.Contains(lowerText, "或者") ||
		strings.Contains(lowerText, "都可以") ||
		strings.Contains(lowerText, "不确定") {
		ambiguities = append(ambiguities, "整理目标存在多个可能方向，请确认当前模板是否正确")
	}
	if strings.TrimSpace(input.TemplateKey) == "" {
		suggestions = append(suggestions, template.Key)
	}

	now := time.Now()
	queryPlan, queryAmbiguities, queryWarnings := parseOrganizeMemoryQueryPlan(text, now)
	ambiguities = append(ambiguities, queryAmbiguities...)
	warnings = append(warnings, queryWarnings...)
	var memories []*types.OrganizeMemory
	if len(input.MemoryIDs) > 0 {
		memoryIDs, err := s.validateMemoryIDs(ctx, tenantID, userID, input.MemoryIDs)
		if err != nil {
			return nil, err
		}
		memories, err = s.repo.ListMemoriesByIDs(ctx, tenantID, userID, memoryIDs)
		if err != nil {
			return nil, err
		}
		queryPlan = types.OrganizeMemoryQueryPlan{Timezone: organizeScheduleLocation().String()}
		ambiguities = removeOrganizeAmbiguity(ambiguities, "未识别到明确时间范围，将使用当前配置的默认记忆范围")
	} else if queryPlan.OccurredFrom != "" || queryPlan.OccurredTo != "" || len(queryPlan.Kinds) > 0 {
		query, queryErr := organizeQueryPlanListQuery(tenantID, userID, queryPlan)
		if queryErr != nil {
			return nil, queryErr
		}
		memories, _, err = s.repo.ListMemories(ctx, query)
		if err != nil {
			return nil, err
		}
	} else {
		memoryIDs, selectErr := s.selectOrganizeJobMemoryIDs(ctx, tenantID, userID, nil)
		if selectErr != nil {
			return nil, selectErr
		}
		memories, err = s.repo.ListMemoriesByIDs(ctx, tenantID, userID, memoryIDs)
		if err != nil {
			return nil, err
		}
	}

	readyMemories, unreadyMemories := splitOrganizeMemoriesByReadiness(memories, input.AllowPartial)
	if len(unreadyMemories) > 0 {
		warnings = append(warnings, "部分记忆尚未完成转写或附件处理")
	}
	plans := planOrganizeMemoryBatches(readyMemories)
	memoryIDs := make(types.StringArray, 0, len(memories))
	for _, memory := range memories {
		memoryIDs = append(memoryIDs, memory.ID)
	}
	overlapCount, err := s.repo.CountOutputMemoryOverlap(ctx, tenantID, userID, memoryIDs)
	if err != nil {
		return nil, err
	}
	sampleCount := len(memories)
	if sampleCount > organizePreviewSampleLimit {
		sampleCount = organizePreviewSampleLimit
	}
	samples := make([]types.OrganizeMemoryReference, 0, sampleCount)
	for _, memory := range memories[:sampleCount] {
		samples = append(samples, types.OrganizeMemoryReference{
			ID:     memory.ID,
			Kind:   memory.Kind,
			Title:  memory.Title,
			Source: memory.Source,
		})
	}
	if len(memories) == 0 {
		warnings = append(warnings, "当前查询范围没有符合条件的记忆")
	}
	return &types.OrganizeRequirementPreview{
		ConfigID:            config.ID,
		TemplateKey:         template.Key,
		TemplateName:        template.Name,
		Scene:               template.Scene,
		NormalizedText:      text,
		QueryPlan:           queryPlan,
		MemoryIDs:           memoryIDs,
		SelectedCount:       len(memories),
		ReadyCount:          len(readyMemories),
		UnreadyCount:        len(unreadyMemories),
		OverlapCount:        int(overlapCount),
		EstimatedBatchCount: len(plans),
		SampleMemories:      samples,
		Ambiguities:         uniqueStrings(ambiguities),
		Suggestions:         uniqueStrings(suggestions),
		Warnings:            uniqueStrings(warnings),
		NeedConfirmation:    len(ambiguities) > 0,
	}, nil
}

func (s *organizeService) ConfirmOrganizeRequirement(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeRequirementInput,
) (*types.OrganizeJob, error) {
	preview, err := s.PreviewOrganizeRequirement(ctx, tenantID, userID, input)
	if err != nil {
		return nil, err
	}
	if preview.NeedConfirmation && !input.Confirmed {
		return nil, ErrOrganizeRequirementConfirmationNeeded
	}
	if preview.SelectedCount == 0 {
		return nil, ErrOrganizeMemoryRequired
	}
	memories, err := s.repo.ListMemoriesByIDs(ctx, tenantID, userID, preview.MemoryIDs)
	if err != nil {
		return nil, err
	}
	snapshot := organizeSelectionSnapshot(preview.QueryPlan, preview.NormalizedText, memories, time.Now())
	return s.CreateJob(ctx, tenantID, userID, types.OrganizeJobInput{
		ConfigID:          preview.ConfigID,
		MemoryIDs:         preview.MemoryIDs,
		ModelID:           input.ModelID,
		Requirement:       preview.NormalizedText,
		AllowPartial:      input.AllowPartial,
		BatchPolicy:       "auto",
		SelectionSnapshot: snapshot,
	})
}

func removeOrganizeAmbiguity(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
