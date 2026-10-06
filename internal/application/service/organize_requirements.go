package service

import (
	"context"
	"sort"
	"strings"

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

	memoryIDs, err := s.selectOrganizeJobMemoryIDs(ctx, tenantID, userID, input.MemoryIDs)
	if err != nil {
		return nil, err
	}
	ambiguities := make([]string, 0, 2)
	suggestions := make([]string, 0, 3)
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
	if len(memoryIDs) == 0 {
		ambiguities = append(ambiguities, "没有指定记忆，将使用当前配置可见范围内的记忆")
	}
	return &types.OrganizeRequirementPreview{
		ConfigID:         config.ID,
		TemplateKey:      template.Key,
		TemplateName:     template.Name,
		Scene:            template.Scene,
		NormalizedText:   text,
		MemoryIDs:        types.StringArray(memoryIDs),
		Ambiguities:      ambiguities,
		Suggestions:      uniqueStrings(suggestions),
		NeedConfirmation: len(ambiguities) > 0,
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
	return s.CreateJob(ctx, tenantID, userID, types.OrganizeJobInput{
		ConfigID:     preview.ConfigID,
		MemoryIDs:    preview.MemoryIDs,
		ModelID:      input.ModelID,
		Requirement:  preview.NormalizedText,
		AllowPartial: input.AllowPartial,
	})
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
