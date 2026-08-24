package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	organizeSproutPromptRuneBudget = 7000
	organizeSproutMaxChips         = 6
	organizeSproutMaxChipRunes     = 16
)

type organizeSproutGeneratedReport struct {
	Title      string
	Summary    string
	OutputHint string
	Chips      []string
	Stage      string
}

func (s *organizeService) CreateSproutReportFromMemory(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.OrganizeSproutFromMemoryInput,
) (*types.OrganizeSproutReport, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	memoryID := strings.TrimSpace(input.MemoryID)
	if memoryID == "" {
		return nil, ErrOrganizeMemoryRequired
	}
	memory, err := s.getMemoryForSprout(ctx, tenantID, userID, memoryID)
	if err != nil {
		return nil, err
	}

	role := types.TenantRoleFromContext(ctx)
	roleConfig := normalizeJSONMap(input.RoleConfig)
	report := buildPendingSproutReportFromMemory(tenantID, userID, memory, role, roleConfig, input.ModelID)
	if err := s.repo.CreateSproutReport(ctx, report, []string{memory.ID}); err != nil {
		return nil, err
	}
	created, err := s.repo.GetSproutReport(ctx, tenantID, userID, report.ID)
	if err != nil {
		return nil, err
	}

	go s.completeSproutReportFromMemory(
		context.WithoutCancel(ctx),
		tenantID,
		userID,
		report.ID,
		memory.ID,
		strings.TrimSpace(input.ModelID),
		role,
		roleConfig,
	)
	return created, nil
}

func buildPendingSproutReportFromMemory(
	tenantID uint64,
	userID string,
	memory *types.OrganizeMemory,
	role types.TenantRole,
	roleConfig types.JSONMap,
	modelID string,
) *types.OrganizeSproutReport {
	title := buildSproutTitle(memory)
	now := time.Now().UTC().Format(time.RFC3339)
	metadata := types.JSONMap{
		"source":             "memory",
		"source_memory_id":   memory.ID,
		"source_memory_kind": memory.Kind,
		"sprout_status":      types.OrganizeSproutStageOrganizing,
		"ai_status":          "pending",
		"role":               string(role),
		"role_config":        roleConfig,
		"created_from":       "memory_menu",
		"started_at":         now,
	}
	if modelID = strings.TrimSpace(modelID); modelID != "" {
		metadata["requested_model_id"] = modelID
	}
	return &types.OrganizeSproutReport{
		TenantID:   tenantID,
		UserID:     userID,
		Title:      title,
		Summary:    buildSproutPendingSummary(memory, role, roleConfig),
		Stage:      types.OrganizeSproutStageOrganizing,
		OutputHint: "AI 发芽报告生成中",
		Chips:      cleanStringArray(buildSproutChips(memory), organizeSproutMaxChips, organizeSproutMaxChipRunes),
		Metadata:   metadata,
	}
}

func (s *organizeService) completeSproutReportFromMemory(
	ctx context.Context,
	tenantID uint64,
	userID string,
	reportID string,
	memoryID string,
	requestedModelID string,
	role types.TenantRole,
	roleConfig types.JSONMap,
) {
	memory, err := s.getMemoryForSprout(ctx, tenantID, userID, memoryID)
	if err != nil {
		logger.Warnf(ctx, "failed to load memory %s for sprout report %s: %v", memoryID, reportID, err)
		return
	}
	generated, modelID, aiStatus := s.generateSproutReportFromMemory(ctx, memory, requestedModelID, role, roleConfig)
	metadata := types.JSONMap{
		"source":             "memory",
		"source_memory_id":   memory.ID,
		"source_memory_kind": memory.Kind,
		"sprout_status":      generated.Stage,
		"ai_status":          aiStatus,
		"ai_model_id":        modelID,
		"role":               string(role),
		"role_config":        roleConfig,
		"created_from":       "memory_menu",
		"completed_at":       time.Now().UTC().Format(time.RFC3339),
	}
	if requestedModelID != "" {
		metadata["requested_model_id"] = requestedModelID
	}
	if _, err := s.UpdateSproutReport(ctx, tenantID, userID, reportID, types.OrganizeSproutReportInput{
		Title:      generated.Title,
		Summary:    generated.Summary,
		Stage:      generated.Stage,
		OutputHint: generated.OutputHint,
		Chips:      cleanStringArray(generated.Chips, organizeSproutMaxChips, organizeSproutMaxChipRunes),
		MemoryIDs:  []string{memory.ID},
		Metadata:   metadata,
	}); err != nil {
		logger.Warnf(ctx, "failed to complete sprout report %s from memory %s: %v", reportID, memory.ID, err)
	}
}

func (s *organizeService) generateSproutReportFromMemory(
	ctx context.Context,
	memory *types.OrganizeMemory,
	requestedModelID string,
	role types.TenantRole,
	roleConfig types.JSONMap,
) (organizeSproutGeneratedReport, string, string) {
	modelID := strings.TrimSpace(requestedModelID)
	if modelID == "" {
		modelID = s.resolveOrganizeModelID(ctx, types.ModelTypeKnowledgeQA)
	}
	if modelID == "" || s.modelService == nil {
		return fallbackSproutReportFromMemory(memory, role, roleConfig, "AI 模型未配置，已生成基础发芽报告，可进入编辑完善。"), modelID, "fallback"
	}
	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return fallbackSproutReportFromMemory(memory, role, roleConfig, "AI 模型不可用，已生成基础发芽报告，可进入编辑完善。"), modelID, "fallback"
	}

	prompt := buildSproutAIPrompt(memory, role, roleConfig)
	thinking := false
	resp, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "system", Content: "你是面向教培和园所经营场景的发芽报告生成助手。输出必须是中文 Markdown，不要输出 JSON。"},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.35,
		MaxTokens:   1800,
		Thinking:    &thinking,
	})
	if err != nil || resp == nil || strings.TrimSpace(resp.Content) == "" {
		return fallbackSproutReportFromMemory(memory, role, roleConfig, "AI 生成失败，已生成基础发芽报告，可进入编辑完善。"), modelID, "fallback"
	}

	return organizeSproutGeneratedReport{
		Title:      buildSproutTitle(memory),
		Summary:    strings.TrimSpace(resp.Content),
		OutputHint: "已生成发芽报告，可继续编辑或用于经营复盘",
		Chips:      buildSproutChips(memory),
		Stage:      types.OrganizeSproutStageFormed,
	}, modelID, "completed"
}

func buildSproutAIPrompt(memory *types.OrganizeMemory, role types.TenantRole, roleConfig types.JSONMap) string {
	memoryText := strings.TrimSpace(memory.Content)
	if memoryText == "" {
		memoryText = memory.Title
	}
	return fmt.Sprintf(`请基于一条记忆生成一份“发芽报告”。

用户角色：%s
角色配置：%s

写作要求：
- 报告要服务当前角色的决策和跟进动作。
- 先用 1 段话概括这条记忆值得发芽的经营价值。
- 至少包含 3 个二级标题，使用“## 01. 标题”格式。
- 每个部分都要包含“🌱 种子”和“✨ Aha 瞬间”两个小段落。
- 最后给出 3 条可执行跟进行动。
- 不要编造具体数字、客户姓名或不存在的事实。

记忆标题：%s
记忆类型：%s
来源：%s
发生时间：%s

记忆内容：
%s`,
		organizeTenantRoleLabel(role),
		organizeRoleConfigText(role, roleConfig),
		memory.Title,
		organizeMemoryKindLabel(memory.Kind),
		emptyFallback(memory.Source, "手动输入"),
		memory.OccurredAt.Format(time.RFC3339),
		sampleRunes(memoryText, organizeSproutPromptRuneBudget, "…"),
	)
}

func fallbackSproutReportFromMemory(
	memory *types.OrganizeMemory,
	role types.TenantRole,
	roleConfig types.JSONMap,
	hint string,
) organizeSproutGeneratedReport {
	title := buildSproutTitle(memory)
	content := strings.TrimSpace(memory.Content)
	if content == "" {
		content = memory.Title
	}
	snippet := sampleRunes(content, 420, "…")
	summary := fmt.Sprintf(`%s

这条记忆已经进入发芽流程。当前报告基于原始记忆生成基础结构，后续可继续用 AI 或人工编辑完善。

## 01. 记忆中的经营信号

> **🌱 种子**
> %s

这条记忆值得被继续拆解，因为它可能连接到招生、家长沟通、课程交付或团队管理中的具体改进动作。

> **✨ Aha 瞬间**
> 发芽的关键不是复述内容，而是把记忆转成当前角色可以跟进的判断和动作。

## 02. 当前角色的关注点

> **🌱 种子**
> 用户角色：%s；角色配置：%s

报告应优先服务该角色的决策场景，聚焦需要确认的事实、可推进的任务和可能存在的风险。

> **✨ Aha 瞬间**
> 同一条记忆对不同角色的价值不同，发芽报告需要把信息转译成角色可执行的下一步。

## 03. 建议跟进行动

1. 补充这条记忆背后的对象、时间和业务场景。
2. 标记它对应的招生、沟通、交付或管理主题。
3. 将可验证的行动拆成负责人、完成时间和预期结果。`,
		hint,
		snippet,
		organizeTenantRoleLabel(role),
		organizeRoleConfigText(role, roleConfig),
	)
	return organizeSproutGeneratedReport{
		Title:      title,
		Summary:    summary,
		OutputHint: hint,
		Chips:      buildSproutChips(memory),
		Stage:      types.OrganizeSproutStageExpandable,
	}
}

func (s *organizeService) getMemoryForSprout(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, ErrOrganizeMemoryRequired
	}
	memory, err := s.repo.GetMemory(ctx, tenantID, userID, id)
	if err != nil {
		return nil, err
	}
	if memory == nil {
		memory, err = s.repo.GetTenantMemory(ctx, tenantID, id)
		if err != nil {
			return nil, err
		}
	}
	if memory == nil {
		return nil, ErrOrganizeNotFound
	}
	return memory, nil
}

func buildSproutPendingSummary(memory *types.OrganizeMemory, role types.TenantRole, roleConfig types.JSONMap) string {
	return fmt.Sprintf(`正在基于「%s」生成发芽报告。

## 01. 发芽任务已创建

> **🌱 种子**
> 这条记忆已经进入发芽流程。

系统会结合当前用户角色（%s）和角色配置（%s）生成经营复盘内容。

> **✨ Aha 瞬间**
> 发芽报告生成后，可在发芽模块继续查看、编辑和沉淀。`,
		memory.Title,
		organizeTenantRoleLabel(role),
		organizeRoleConfigText(role, roleConfig),
	)
}

func buildSproutTitle(memory *types.OrganizeMemory) string {
	title := strings.TrimSpace(memory.Title)
	if title == "" {
		title = "无标题记忆"
	}
	return trimMax(title+" 发芽报告", organizeMaxTitleLength)
}

func buildSproutChips(memory *types.OrganizeMemory) []string {
	chips := []string{"记忆发芽", organizeMemoryKindLabel(memory.Kind)}
	if source := strings.TrimSpace(memory.Source); source != "" {
		chips = append(chips, source)
	}
	chips = append(chips, extractSproutMetadataTags(memory.Metadata)...)
	return chips
}

func extractSproutMetadataTags(metadata types.JSONMap) []string {
	if len(metadata) == 0 {
		return nil
	}
	keys := []string{"tags", "chips", "keywords"}
	out := make([]string, 0)
	for _, key := range keys {
		switch value := metadata[key].(type) {
		case []string:
			out = append(out, value...)
		case types.StringArray:
			out = append(out, []string(value)...)
		case []any:
			for _, item := range value {
				if s, ok := item.(string); ok {
					out = append(out, s)
				}
			}
		case string:
			out = append(out, strings.FieldsFunc(value, func(r rune) bool {
				return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n'
			})...)
		}
	}
	return out
}

func organizeMemoryKindLabel(kind string) string {
	switch kind {
	case types.OrganizeMemoryKindAudio:
		return "录音"
	case types.OrganizeMemoryKindAudioCard:
		return "工牌"
	case types.OrganizeMemoryKindRecord:
		return "记录"
	default:
		return "笔记"
	}
}

func organizeTenantRoleLabel(role types.TenantRole) string {
	switch role {
	case types.TenantRoleOwner:
		return "空间负责人"
	case types.TenantRoleAdmin:
		return "空间管理员"
	case types.TenantRoleContributor:
		return "内容共创者"
	default:
		return "观察者"
	}
}

func organizeRoleConfigText(role types.TenantRole, roleConfig types.JSONMap) string {
	if len(roleConfig) > 0 {
		if b, err := json.Marshal(roleConfig); err == nil {
			return string(b)
		}
	}
	switch role {
	case types.TenantRoleOwner, types.TenantRoleAdmin:
		return "关注经营判断、团队分工、风险控制和落地节奏"
	case types.TenantRoleContributor:
		return "关注内容沉淀、沟通素材和可执行跟进行动"
	default:
		return "关注关键信息、背景理解和可观察信号"
	}
}

func emptyFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
