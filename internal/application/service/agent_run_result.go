package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

func buildDailyReportAgentResult(report *types.ServiceDailyReport) (types.AgentResultV1, error) {
	if report == nil {
		return types.AgentResultV1{}, errors.New("daily report is required")
	}
	structured := structuredDailyReportFromServiceReport(report)
	content, err := structured.ToJSONMap()
	if err != nil {
		return types.AgentResultV1{}, fmt.Errorf("encode structured daily report: %w", err)
	}
	return types.AgentResultV1{
		SchemaVersion: types.AgentResultSchemaV1,
		Decision: types.AgentResultDecisionV1{
			ShouldCreateCard: false,
			Confidence:       1,
			Reason:           "日报已汇总当前服务事项和来源记忆。",
		},
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:    types.AgentArtifactKindReport,
				Role:    types.AgentArtifactRolePrimary,
				Title:   report.Title,
				Format:  types.StructuredReportFormatV1,
				Content: content,
			},
		},
		Evidence: agentEvidenceFromIDs(report.SourceMemoryIDs, "evidence"),
	}, nil
}

func buildMemoryExtractionAgentResult(
	memoryID string,
	extraction *types.ServiceMemoryExtraction,
) (types.AgentResultV1, string, error) {
	if extraction == nil {
		return types.AgentResultV1{}, "", errors.New("memory extraction is required")
	}
	evidence := agentEvidenceFromMemoryExtraction(memoryID, extraction)
	if !extraction.Generated {
		return types.AgentResultV1{
			SchemaVersion: types.AgentResultSchemaV1,
			Decision: types.AgentResultDecisionV1{
				ShouldCreateCard: false,
				Confidence:       0,
				Reason:           firstNonEmpty(strings.TrimSpace(extraction.Reason), "当前记忆不需要生成服务卡片。"),
			},
			Artifacts: []types.AgentArtifactResultV1{},
			Evidence:  evidence,
		}, "", nil
	}
	if extraction.Reminder == nil {
		return types.AgentResultV1{}, "", errors.New("generated memory extraction is missing a service reminder")
	}
	reminder := extraction.Reminder
	structured := structuredReportFromServiceReminder(reminder)
	content, err := structured.ToJSONMap()
	if err != nil {
		return types.AgentResultV1{}, "", fmt.Errorf("encode service reminder report: %w", err)
	}
	return types.AgentResultV1{
		SchemaVersion: types.AgentResultSchemaV1,
		Decision: types.AgentResultDecisionV1{
			ShouldCreateCard: true,
			Confidence:       reminder.Confidence,
			Reason:           firstNonEmpty(strings.TrimSpace(reminder.AssistReason), strings.TrimSpace(extraction.Reason), "识别到需要跟进的服务事项。"),
		},
		Card: &types.ServiceCardV1{
			SchemaVersion: types.ServiceCardSchemaV1,
			Title:         reminder.Title,
			Summary:       reminder.Summary,
			NextAction:    reminder.NextAction,
		},
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:    types.AgentArtifactKindReport,
				Role:    types.AgentArtifactRolePrimary,
				Title:   firstNonEmpty(reminder.Title+"服务分析", "服务分析"),
				Format:  types.StructuredReportFormatV1,
				Content: content,
			},
		},
		Evidence: evidence,
	}, reminder.ProfileID, nil
}

func structuredDailyReportFromServiceReport(report *types.ServiceDailyReport) types.StructuredReportV1 {
	if report.StructuredReport != nil {
		return *report.StructuredReport
	}
	facts := []string{
		fmt.Sprintf("本次汇总 %d 个服务动作，覆盖 %d 个服务对象。", report.ActionCount, report.SubjectCount),
		fmt.Sprintf("关联 %d 条来源记忆，证据完整率 %d%%。", report.SourceMemoryCount, report.EvidenceCompleteRate),
	}
	risks := []string{"当前未识别高风险服务事项。"}
	if report.HighRiskCount > 0 {
		risks = []string{fmt.Sprintf("当前有 %d 个高风险服务事项，需要优先核实和推进。", report.HighRiskCount)}
	}
	missing := []string{"当前没有明确的知识补齐项。"}
	if report.KnowledgeGapCount > 0 {
		missing = []string{fmt.Sprintf("存在 %d 项知识补齐建议，需在后续服务中补充事实或材料。", report.KnowledgeGapCount)}
	}
	recommendedActions := append([]string{}, report.Chips...)
	if len(recommendedActions) == 0 {
		recommendedActions = []string{"查看日报中的行动闭环，确认最高优先级的待办事项。"}
	}
	evidenceItems := evidenceItemsFromIDs(report.SourceMemoryIDs)
	return types.StructuredReportV1{
		Format:           types.StructuredReportFormatV1,
		Title:            report.Title,
		ExecutiveSummary: firstNonEmpty(report.Summary, "日报已生成，请根据行动闭环确认下一步服务动作。"),
		Sections: []types.StructuredReportSection{
			{Type: "facts", Title: "已知事实", Items: types.StringArray(facts)},
			{Type: "analysis", Title: "判断", Content: firstNonEmpty(report.Summary, "本报告基于已生成的服务事项和授权记忆汇总。")},
			{Type: "risks", Title: "风险", Items: types.StringArray(risks)},
			{Type: "missing_information", Title: "待确认信息", Items: types.StringArray(missing)},
			{Type: "recommended_actions", Title: "建议动作", Items: types.StringArray(recommendedActions)},
			{Type: "talk_track", Title: "执行参考", Content: "优先完成高风险或临近截止的事项，完成后在服务空间回写处理结果。"},
			{Type: "evidence", Title: "证据来源", Items: types.StringArray(evidenceItems)},
		},
		EvidenceRefs: types.StringArray(cleanAgentEvidenceIDs(report.SourceMemoryIDs)),
	}
}

func structuredReportFromServiceReminder(reminder *types.ServiceReminder) types.StructuredReportV1 {
	facts := []string{
		"当前阶段：" + firstNonEmpty(reminder.Stage, "待确认"),
		"服务渠道：" + firstNonEmpty(reminder.Channel, "待确认"),
	}
	if reminder.DecisionRole != "" {
		facts = append(facts, "关联角色："+reminder.DecisionRole)
	}
	risks := []string{"当前没有明确风险标签。"}
	if reminder.RiskLabel != "" {
		risks = []string{"风险/关注：" + reminder.RiskLabel}
	}
	recommendedActions := nonBlankStrings(reminder.NextAction, reminder.PrimaryAction)
	if len(recommendedActions) == 0 {
		recommendedActions = []string{"先确认服务对象和当前情况，再补充下一步动作。"}
	}
	return types.StructuredReportV1{
		Format:           types.StructuredReportFormatV1,
		Title:            firstNonEmpty(reminder.Title+"服务分析", "服务分析"),
		ExecutiveSummary: firstNonEmpty(reminder.Summary, "已识别服务事项，请核实后执行下一步动作。"),
		Sections: []types.StructuredReportSection{
			{Type: "facts", Title: "已知事实", Items: types.StringArray(facts)},
			{Type: "analysis", Title: "判断", Content: firstNonEmpty(reminder.AssistReason, reminder.Summary)},
			{Type: "risks", Title: "风险", Items: types.StringArray(risks)},
			{Type: "missing_information", Title: "待确认信息", Items: types.StringArray([]string{
				firstNonEmpty(reminder.WriteBackDraft, "完成下一步后补充处理结果和客户反馈。"),
			})},
			{Type: "recommended_actions", Title: "建议动作", Items: types.StringArray(recommendedActions)},
			{Type: "talk_track", Title: "沟通参考", Content: firstNonEmpty(reminder.ReplyDraft, reminder.NextAction)},
			{Type: "evidence", Title: "证据来源", Items: types.StringArray(evidenceItemsFromReminder(reminder))},
		},
		EvidenceRefs: types.StringArray(cleanAgentEvidenceIDs(reminder.SourceMemoryIDs)),
	}
}

func agentEvidenceFromMemoryExtraction(
	memoryID string,
	extraction *types.ServiceMemoryExtraction,
) []types.AgentEvidenceRefV1 {
	result := make([]types.AgentEvidenceRefV1, 0)
	seen := map[string]bool{}
	add := func(id, relation, excerpt string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		result = append(result, types.AgentEvidenceRefV1{
			SourceType: "memory",
			SourceID:   id,
			Relation:   relation,
			Excerpt:    trimMax(strings.TrimSpace(excerpt), 1000),
		})
	}
	if extraction != nil && extraction.Reminder != nil {
		for _, item := range extraction.Reminder.MemoryEvidence {
			relation := "evidence"
			if item.ID == memoryID {
				relation = "trigger"
			}
			add(item.ID, relation, firstNonEmpty(item.Summary, item.Title))
		}
		for _, id := range extraction.Reminder.SourceMemoryIDs {
			relation := "evidence"
			if id == memoryID {
				relation = "trigger"
			}
			add(id, relation, extraction.Reminder.Summary)
		}
	}
	add(memoryID, "trigger", "")
	return result
}

func agentEvidenceFromIDs(ids []string, relation string) []types.AgentEvidenceRefV1 {
	result := make([]types.AgentEvidenceRefV1, 0, len(ids))
	for _, id := range cleanAgentEvidenceIDs(ids) {
		result = append(result, types.AgentEvidenceRefV1{
			SourceType: "memory",
			SourceID:   id,
			Relation:   relation,
		})
	}
	return result
}

func evidenceItemsFromReminder(reminder *types.ServiceReminder) []string {
	if reminder == nil {
		return []string{"暂无关联记忆证据。"}
	}
	items := make([]string, 0, len(reminder.MemoryEvidence))
	seen := map[string]bool{}
	for _, evidence := range reminder.MemoryEvidence {
		if strings.TrimSpace(evidence.ID) == "" || seen[evidence.ID] {
			continue
		}
		seen[evidence.ID] = true
		items = append(items, firstNonEmpty(evidence.Title, evidence.Summary, evidence.ID))
	}
	if len(items) == 0 {
		items = evidenceItemsFromIDs(reminder.SourceMemoryIDs)
	}
	return items
}

func evidenceItemsFromIDs(ids []string) []string {
	cleaned := cleanAgentEvidenceIDs(ids)
	if len(cleaned) == 0 {
		return []string{"暂无关联记忆证据。"}
	}
	items := make([]string, 0, len(cleaned))
	for _, id := range cleaned {
		items = append(items, "记忆："+id)
	}
	return items
}

func cleanAgentEvidenceIDs(ids []string) []string {
	result := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	return result
}

func nonBlankStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
