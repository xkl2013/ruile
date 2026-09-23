package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const expertFollowUpDefaultMode = "explanation"

func (s *agentRunService) EnqueuePublishedExpertFollowUp(
	ctx context.Context,
	tenantID uint64,
	userID string,
	input types.ExpertFollowUpInput,
) (*types.AgentRun, error) {
	if err := validateAgentRunScope(tenantID, userID); err != nil {
		return nil, err
	}
	input.ParentRunID = strings.TrimSpace(input.ParentRunID)
	input.Prompt = strings.TrimSpace(input.Prompt)
	input.Mode = strings.TrimSpace(input.Mode)
	input.ModelID = strings.TrimSpace(input.ModelID)
	input.ServiceID = strings.TrimSpace(input.ServiceID)
	input.SessionID = strings.TrimSpace(input.SessionID)
	if input.Mode == "" {
		input.Mode = expertFollowUpDefaultMode
	}
	if input.ParentRunID == "" || input.Prompt == "" ||
		utf8.RuneCountInString(input.Prompt) > expertAgentTestMaxPromptRunes ||
		input.Mode != expertFollowUpDefaultMode {
		return nil, ErrAgentRunInvalidRequest
	}
	if (input.ServiceID == "") != (input.SessionID == "") {
		return nil, ErrAgentRunInvalidRequest
	}
	var parent *types.AgentRun
	var err error
	if input.ServiceID != "" {
		if s.serviceSpace == nil {
			return nil, errors.New("service workspace runtime is not configured")
		}
		if _, err := s.serviceSpace.Authorize(ctx, tenantID, userID, input.ServiceID, types.ServiceMemberRoleEditor, true); err != nil {
			return nil, err
		}
		if _, err := s.serviceSpace.GetSession(ctx, tenantID, userID, input.ServiceID, input.SessionID); err != nil {
			return nil, err
		}
		parent, err = s.repo.GetByIDForService(ctx, tenantID, input.ServiceID, input.ParentRunID)
	} else {
		parent, err = s.repo.GetByIDForUser(ctx, tenantID, userID, input.ParentRunID)
	}
	if err != nil {
		return nil, err
	}
	if parent == nil {
		return nil, ErrAgentRunNotFound
	}
	if input.ServiceID == "" && strings.TrimSpace(parent.ServiceID) != "" {
		return nil, ErrAgentRunNotFound
	}
	if input.ServiceID != "" && parent.ServiceID != input.ServiceID {
		return nil, ErrAgentRunNotFound
	}
	if parent.Status != types.AgentRunStatusSucceeded ||
		(parent.RunType != types.AgentRunTypeExpertAgentTest && parent.RunType != types.AgentRunTypeExpertFollowUp) {
		return nil, fmt.Errorf("%w: parent expert run must be succeeded", ErrAgentRunInvalidRequest)
	}

	expertInput, err := publishedExpertInputFromRun(parent)
	if err != nil {
		return nil, err
	}
	if input.ModelID == "" {
		input.ModelID = expertInput.ModelID
	}
	input.PackageID = expertInput.PackageID
	input.DefinitionID = expertInput.DefinitionID
	if s.expertPackages == nil {
		return nil, errors.New("expert package repository is not configured")
	}
	definition, err := s.expertPackages.GetPublishedDefinition(
		ctx,
		tenantID,
		input.PackageID,
		input.DefinitionID,
	)
	if err != nil {
		return nil, err
	}
	if definition == nil {
		return nil, ErrExpertPackageNotPublished
	}

	runInput, err := agentRunJSONMap(input)
	if err != nil {
		return nil, fmt.Errorf("encode expert follow-up request: %w", err)
	}
	return s.enqueue(ctx, tenantID, userID, &types.AgentRun{
		ServiceID:    input.ServiceID,
		ThreadID:     firstNonEmpty(input.SessionID, parent.ThreadID, parent.ID),
		ParentRunID:  parent.ID,
		RunType:      types.AgentRunTypeExpertFollowUp,
		AgentRef:     definition.AgentID,
		AgentVersion: definition.Version,
		TriggerType:  "published_expert_follow_up",
		TriggerID:    definition.ID,
		Input:        runInput,
	}, agentRunIdempotencyKey(tenantID, userID, types.AgentRunTypeExpertFollowUp, runInput))
}

func (s *agentRunService) executeExpertFollowUp(
	ctx context.Context,
	run *types.AgentRun,
) (types.AgentResultV1, types.JSONMap, string, error) {
	var input types.ExpertFollowUpInput
	if err := decodeAgentRunInput(run.Input, &input); err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: fmt.Errorf("decode expert follow-up input: %w", err)}
	}
	if s.expertPackages == nil || s.modelService == nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: errors.New("expert follow-up runtime is not configured")}
	}
	definition, err := s.expertPackages.GetPublishedDefinition(
		ctx,
		run.TenantID,
		input.PackageID,
		input.DefinitionID,
	)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	if definition == nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: ErrExpertPackageNotPublished}
	}
	if tools := expertDefinitionStringList(definition.CompiledConfig, "allowed_tools"); len(tools) > 0 {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{
			err: fmt.Errorf("expert follow-up runner does not support tools yet: %s", strings.Join(tools, ", ")),
		}
	}

	tenantCtx := context.WithValue(ctx, types.TenantIDContextKey, run.TenantID)
	modelID, err := s.resolveExpertTestModelID(tenantCtx, definition, input.ModelID)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	chatModel, err := s.modelService.GetChatModel(tenantCtx, modelID)
	if err != nil {
		return types.AgentResultV1{}, nil, "", fmt.Errorf("get expert follow-up model: %w", err)
	}
	skills, err := loadExpertSkillSnapshots(definition)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	parentReport, err := s.loadExpertParentReport(ctx, run)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	threadContext, err := s.renderExpertThreadContext(ctx, run)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}

	step, err := s.startExpertRunStep(ctx, run, types.AgentRunStepTypeDrafting, types.AgentRunPhaseDrafting, modelID, types.JSONMap{
		"mode":          input.Mode,
		"parent_run_id": input.ParentRunID,
		"prompt":        input.Prompt,
	})
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}

	system := strings.TrimSpace(definition.SystemPrompt) + renderExpertSkillContext(skills) + `

你现在处理的是已完成专家结果的解释型追问。
只回答用户当前追问，不生成新的服务卡片、文件或完整方案。
基于上一版报告和已经确认的信息回答，不重复要求用户补充微观细节。
如果报告中的内容是合理假设，请明确标注假设；如果缺少依据，请直接说明限制。
输出普通 Markdown 文本，结构清晰、结论优先，避免空泛复述。`
	user := "上一版专家报告：\n" + firstNonEmpty(parentReport, "上一版报告正文不可用，请基于已确认的上下文回答。") +
		"\n\n同一专家线程的近期交互：\n" + firstNonEmpty(threadContext, "暂无更早的追问记录。") +
		"\n\n专家定义：\n" + definition.SystemPrompt +
		"\n\n用户追问：\n" + input.Prompt
	thinking := false
	response, err := chatModel.Chat(ctx, []chat.Message{
		{Role: "system", Content: strings.TrimSpace(system)},
		{Role: "user", Content: strings.TrimSpace(user)},
	}, &chat.ChatOptions{
		Temperature: 0.2,
		MaxTokens:   expertAgentTestDefaultTokens,
		Thinking:    &thinking,
	})
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return types.AgentResultV1{}, nil, "", fmt.Errorf("expert follow-up: %w", err)
	}
	answer := ""
	if response != nil {
		answer = strings.TrimSpace(response.Content)
	}
	if answer == "" {
		err := errors.New("expert follow-up returned an empty answer")
		_ = s.failExpertRunStep(ctx, step, err)
		return types.AgentResultV1{}, nil, "", invalidAgentRunOutputError{err: err}
	}
	if err := s.completeExpertRunStep(ctx, step, types.JSONMap{
		"mode":    input.Mode,
		"message": answer,
	}); err != nil {
		return types.AgentResultV1{}, nil, "", err
	}

	return types.AgentResultV1{
		SchemaVersion: types.AgentResultSchemaV1,
		Decision: types.AgentResultDecisionV1{
			ShouldCreateCard: false,
			Confidence:       1,
			Reason:           "解释型追问只返回聊天回答，不生成卡片或产物。",
		},
		Artifacts: []types.AgentArtifactResultV1{},
		Evidence:  []types.AgentEvidenceRefV1{},
	}, types.JSONMap{
		"message":        answer,
		"follow_up_mode": input.Mode,
		"parent_run_id":  input.ParentRunID,
		"package_id":     input.PackageID,
		"definition_id":  input.DefinitionID,
		"model_id":       modelID,
	}, "", nil
}

func (s *agentRunService) renderExpertThreadContext(
	ctx context.Context,
	run *types.AgentRun,
) (string, error) {
	if run == nil {
		return "", nil
	}
	threadID := firstNonEmpty(strings.TrimSpace(run.ThreadID), run.ID)
	runs, err := s.repo.ListByThreadForUser(ctx, run.TenantID, run.UserID, threadID)
	if err != nil {
		return "", fmt.Errorf("load expert thread: %w", err)
	}
	if len(runs) > 8 {
		runs = runs[len(runs)-8:]
	}
	var builder strings.Builder
	for _, item := range runs {
		if item == nil || item.ID == run.ID {
			continue
		}
		switch item.RunType {
		case types.AgentRunTypeExpertAgentTest:
			var input types.ExpertAgentTestInput
			if decodeAgentRunInput(item.Input, &input) == nil {
				prompt := strings.TrimSpace(input.Prompt)
				if feedback := strings.TrimSpace(input.Feedback); feedback != "" {
					prompt = firstNonEmpty(prompt, "沿用原需求") + "\n修订要求：" + feedback
				}
				if prompt != "" {
					builder.WriteString("用户任务：")
					builder.WriteString(prompt)
					builder.WriteString("\n")
				}
			}
			if card, ok := item.Result["card"].(map[string]any); ok {
				if summary := strings.TrimSpace(asString(card["summary"])); summary != "" {
					builder.WriteString("已交付摘要：")
					builder.WriteString(trimExpertRunes(summary, 800))
					builder.WriteString("\n")
				}
			}
		case types.AgentRunTypeExpertFollowUp:
			var input types.ExpertFollowUpInput
			if decodeAgentRunInput(item.Input, &input) == nil && strings.TrimSpace(input.Prompt) != "" {
				builder.WriteString("用户追问：")
				builder.WriteString(strings.TrimSpace(input.Prompt))
				builder.WriteString("\n")
			}
		}
		if message, ok := item.Result["message"].(string); ok && strings.TrimSpace(message) != "" {
			builder.WriteString("专家回答：")
			builder.WriteString(trimExpertRunes(message, 1800))
			builder.WriteString("\n")
		}
	}
	return trimExpertRunes(builder.String(), 8000), nil
}

func publishedExpertInputFromRun(run *types.AgentRun) (types.ExpertAgentTestInput, error) {
	if run == nil {
		return types.ExpertAgentTestInput{}, ErrAgentRunNotFound
	}
	switch run.RunType {
	case types.AgentRunTypeExpertAgentTest:
		var input types.ExpertAgentTestInput
		if err := decodeAgentRunInput(run.Input, &input); err != nil {
			return types.ExpertAgentTestInput{}, fmt.Errorf("decode parent expert run input: %w", err)
		}
		return input, nil
	case types.AgentRunTypeExpertFollowUp:
		var input types.ExpertFollowUpInput
		if err := decodeAgentRunInput(run.Input, &input); err != nil {
			return types.ExpertAgentTestInput{}, fmt.Errorf("decode parent expert follow-up input: %w", err)
		}
		return types.ExpertAgentTestInput{
			PackageID:    input.PackageID,
			DefinitionID: input.DefinitionID,
			ModelID:      input.ModelID,
		}, nil
	default:
		return types.ExpertAgentTestInput{}, fmt.Errorf("%w: parent run is not an expert run", ErrAgentRunInvalidRequest)
	}
}
