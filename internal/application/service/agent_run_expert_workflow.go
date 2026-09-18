package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	agentcore "github.com/Tencent/WeKnora/internal/agent"
	agenttools "github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
)

const (
	expertWorkflowDefaultQualityScore = 85
	expertWorkflowDefaultRevisions    = 2
	expertWorkflowMinReportRunes      = 1200
)

var expertIntakeSchema = json.RawMessage(`{
  "type": "object",
  "required": ["ready", "questions", "assumptions"],
  "properties": {
    "ready": {"type": "boolean"},
    "questions": {
      "type": "array",
      "maxItems": 5,
      "items": {
        "type": "object",
        "required": ["id", "label", "type", "required"],
        "properties": {
          "id": {"type": "string"},
          "label": {"type": "string"},
          "type": {"type": "string", "enum": ["text", "single_choice", "multi_choice", "date", "number"]},
          "required": {"type": "boolean"},
          "options": {"type": "array", "items": {"type": "string"}},
          "description": {"type": "string"}
        }
      }
    },
    "assumptions": {"type": "array", "items": {"type": "string"}}
  }
}`)

var expertPlanSchema = json.RawMessage(`{
  "type": "object",
  "required": ["objective", "requirements", "assumptions", "sections", "checklist"],
  "properties": {
    "objective": {"type": "string"},
    "requirements": {"type": "array", "items": {"type": "string"}},
    "assumptions": {"type": "array", "items": {"type": "string"}},
    "sections": {"type": "array", "items": {"type": "string"}},
    "checklist": {"type": "array", "items": {"type": "string"}}
  }
}`)

var expertQualitySchema = json.RawMessage(`{
  "type": "object",
  "required": ["score", "passed", "summary", "red_lines", "dimensions", "issues"],
  "properties": {
    "score": {"type": "integer", "minimum": 0, "maximum": 100},
    "passed": {"type": "boolean"},
    "summary": {"type": "string"},
    "red_lines": {"type": "array", "items": {"type": "string"}},
    "dimensions": {
      "type": "object",
      "additionalProperties": {"type": "integer", "minimum": 0, "maximum": 100}
    },
    "issues": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["code", "severity", "message", "instruction"],
        "properties": {
          "code": {"type": "string"},
          "severity": {"type": "string", "enum": ["warning", "error", "red_line"]},
          "section": {"type": "string"},
          "message": {"type": "string"},
          "instruction": {"type": "string"}
        }
      }
    }
  }
}`)

type expertIntakeResult struct {
	Ready       bool                         `json:"ready"`
	Questions   []types.ExpertIntakeQuestion `json:"questions"`
	Assumptions []string                     `json:"assumptions"`
}

type expertExecutionPlan struct {
	Objective    string   `json:"objective"`
	Requirements []string `json:"requirements"`
	Assumptions  []string `json:"assumptions"`
	Sections     []string `json:"sections"`
	Checklist    []string `json:"checklist"`
}

type expertQualityIssue struct {
	Code        string `json:"code"`
	Severity    string `json:"severity"`
	Section     string `json:"section,omitempty"`
	Message     string `json:"message"`
	Instruction string `json:"instruction"`
}

type expertQualityAssessment struct {
	Score      int                  `json:"score"`
	Passed     bool                 `json:"passed"`
	Summary    string               `json:"summary"`
	RedLines   []string             `json:"red_lines"`
	Dimensions map[string]int       `json:"dimensions"`
	Issues     []expertQualityIssue `json:"issues"`
}

type expertSkillSnapshot struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA256  string `json:"sha256"`
}

func (s *agentRunService) executeExpertWorkflow(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
) (types.AgentResultV1, types.JSONMap, string, error) {
	skills, err := loadExpertSkillSnapshots(definition)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	skillContext := renderExpertSkillContext(skills)

	intake, err := s.runExpertIntake(ctx, run, input, definition, modelID, chatModel, skillContext)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	if !intake.Ready && len(intake.Questions) > 0 {
		interaction, mapErr := agentRunJSONMap(types.ExpertIntakeInteraction{
			SchemaVersion: "intake_request_v1",
			Questions:     intake.Questions,
		})
		if mapErr != nil {
			return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: mapErr}
		}
		return types.AgentResultV1{}, nil, "", agentRunWaitingInputError{interaction: interaction}
	}

	snapshotID, err := s.ensureExpertRequirementSnapshot(ctx, run, input)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	plan, err := s.runExpertPlanning(ctx, run, input, definition, modelID, chatModel, skillContext)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	report, err := s.runExpertWriter(
		ctx,
		run,
		types.AgentRunStepTypeDrafting,
		types.AgentRunPhaseDrafting,
		input,
		definition,
		modelID,
		chatModel,
		skillContext,
		plan,
		"",
	)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}

	quality, err := s.runExpertQualityReview(ctx, run, input, definition, modelID, chatModel, plan, report)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	for revision := 1; !quality.Passed && revision <= expertMaxRevisionRounds(definition.CompiledConfig); revision++ {
		report, err = s.runExpertWriter(
			ctx,
			run,
			types.AgentRunStepTypeRevising,
			types.AgentRunPhaseRevising,
			input,
			definition,
			modelID,
			chatModel,
			skillContext,
			plan,
			expertRevisionInstructions(quality),
		)
		if err != nil {
			return types.AgentResultV1{}, nil, "", err
		}
		quality, err = s.runExpertQualityReview(ctx, run, input, definition, modelID, chatModel, plan, report)
		if err != nil {
			return types.AgentResultV1{}, nil, "", err
		}
	}
	qualityMap, err := agentRunJSONMap(quality)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	if err := s.repo.SetQuality(ctx, run.ID, qualityMap); err != nil {
		return types.AgentResultV1{}, nil, "", err
	}

	result, err := s.runExpertPackaging(
		ctx,
		run,
		input,
		definition,
		modelID,
		chatModel,
		report,
		quality,
	)
	if err != nil {
		return types.AgentResultV1{}, nil, "", err
	}
	return result, types.JSONMap{
		"artifact_type":           "expert_agent_test",
		"package_id":              input.PackageID,
		"definition_id":           definition.ID,
		"expert_name":             definition.DisplayName,
		"model_id":                modelID,
		"workflow_version":        "expert_workflow_v2",
		"requirement_snapshot_id": snapshotID,
		"loaded_skills":           expertSkillTrace(skills),
		"quality":                 qualityMap,
		"report_markdown":         report,
	}, input.ProfileID, nil
}

func (s *agentRunService) runExpertIntake(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	skillContext string,
) (expertIntakeResult, error) {
	step, err := s.startExpertRunStep(ctx, run, types.AgentRunStepTypeIntake, types.AgentRunPhaseIntake, modelID, types.JSONMap{
		"prompt": input.Prompt,
	})
	if err != nil {
		return expertIntakeResult{}, err
	}
	requiredQuestions := deterministicExpertIntakeQuestions(definition.CompiledConfig, input.Answers)
	if len(requiredQuestions) > 0 {
		intake := expertIntakeResult{
			Ready:       false,
			Questions:   requiredQuestions,
			Assumptions: []string{},
		}
		output, _ := agentRunJSONMap(intake)
		if err := s.completeExpertRunStep(ctx, step, output); err != nil {
			return expertIntakeResult{}, err
		}
		return intake, nil
	}
	system := strings.TrimSpace(definition.SystemPrompt) + skillContext + `

# 睿乐需求澄清
你现在只判断信息是否足以开始正式交付，不生成方案。
结合专家工作方法识别会改变方案结构、预算、时间安排或安全结论的关键变量。
用户已经明确提供的信息不要重复询问。每轮最多 5 个问题，优先提供单选项。
日期、参与规模、场地、活动形式、预算或目标受众等关键条件缺失时，应返回 ready=false。
如果信息足够，返回 ready=true 和空 questions。`
	var intake expertIntakeResult
	err = expertChatJSON(ctx, chatModel, system, "用户请求：\n"+input.Prompt+
		"\n\n用户已补充答案：\n"+expertConfigJSON(input.Answers)+
		"\n\n专家声明的必填字段：\n"+expertConfigJSON(definition.CompiledConfig["required_inputs"]), expertIntakeSchema, &intake,
		expertDefinitionFloat(definition.CompiledConfig, "temperature", 0.2),
		expertDefinitionBool(definition.CompiledConfig, "thinking", false),
		expertAgentTestDefaultTokens)
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return expertIntakeResult{}, fmt.Errorf("expert intake: %w", err)
	}
	intake.Questions = normalizeExpertIntakeQuestions(intake.Questions)
	if len(intake.Questions) == 0 {
		intake.Ready = true
	}
	output, _ := agentRunJSONMap(intake)
	if err := s.completeExpertRunStep(ctx, step, output); err != nil {
		return expertIntakeResult{}, err
	}
	return intake, nil
}

func (s *agentRunService) ensureExpertRequirementSnapshot(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
) (string, error) {
	if strings.TrimSpace(run.RequirementSnapshotID) != "" {
		return run.RequirementSnapshotID, nil
	}
	snapshot := &types.AgentRequirementSnapshot{
		RunID: run.ID,
		Values: types.JSONMap{
			"prompt":  input.Prompt,
			"answers": cloneAgentRunJSONMap(input.Answers),
		},
		Assumptions: types.StringArray{},
		Missing:     types.StringArray{},
	}
	if err := s.repo.CreateRequirementSnapshot(ctx, snapshot); err != nil {
		return "", fmt.Errorf("save expert requirement snapshot: %w", err)
	}
	if err := s.repo.SetRequirementSnapshot(ctx, run.ID, snapshot.ID); err != nil {
		return "", fmt.Errorf("bind expert requirement snapshot: %w", err)
	}
	run.RequirementSnapshotID = snapshot.ID
	return snapshot.ID, nil
}

func (s *agentRunService) runExpertPlanning(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	skillContext string,
) (expertExecutionPlan, error) {
	step, err := s.startExpertRunStep(ctx, run, types.AgentRunStepTypePlanning, types.AgentRunPhasePlanning, modelID, types.JSONMap{
		"prompt":  input.Prompt,
		"answers": cloneAgentRunJSONMap(input.Answers),
	})
	if err != nil {
		return expertExecutionPlan{}, err
	}
	system := strings.TrimSpace(definition.SystemPrompt) + skillContext + `

# 睿乐执行规划
你现在是 Planner。不要生成最终报告，只输出执行计划。
计划必须区分用户确认信息与明确假设，列出交付章节、需要检查的计算和安全红线。
章节应忠实遵守专家定义中的标准结构和交付要求。`
	var plan expertExecutionPlan
	user := "原始请求：\n" + input.Prompt + "\n\n用户补充答案：\n" + expertConfigJSON(input.Answers) +
		"\n\n交付规格：\n" + expertConfigJSON(definition.CompiledConfig["deliverable_spec"])
	err = expertChatJSON(ctx, chatModel, system, user, expertPlanSchema, &plan,
		expertDefinitionFloat(definition.CompiledConfig, "temperature", 0.2),
		expertDefinitionBool(definition.CompiledConfig, "thinking", false),
		expertAgentTestDefaultTokens)
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return expertExecutionPlan{}, fmt.Errorf("expert planning: %w", err)
	}
	if strings.TrimSpace(plan.Objective) == "" {
		plan.Objective = input.Prompt
	}
	output, _ := agentRunJSONMap(plan)
	if err := s.completeExpertRunStep(ctx, step, output); err != nil {
		return expertExecutionPlan{}, err
	}
	return plan, nil
}

func (s *agentRunService) runExpertWriter(
	ctx context.Context,
	run *types.AgentRun,
	stepType, phase string,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	skillContext string,
	plan expertExecutionPlan,
	revisionInstructions string,
) (string, error) {
	stepInput := types.JSONMap{
		"prompt": input.Prompt,
		"plan":   plan,
	}
	if revisionInstructions != "" {
		stepInput["revision_instructions"] = revisionInstructions
	}
	step, err := s.startExpertRunStep(ctx, run, stepType, phase, modelID, stepInput)
	if err != nil {
		return "", err
	}
	systemPrompt := strings.TrimSpace(definition.SystemPrompt) + skillContext + `

# 睿乐专家工作流
你现在是 Writer。需求澄清已经完成，不再向用户提问。
严格执行计划和专家定义，输出一份可直接交付的完整 Markdown 报告。
事实、用户确认信息、假设和待确认项必须明确区分。
人员、时间、预算、数量和安全要求应尽可能具体；不得伪造用户没有提供的事实。
不要输出 agent_result_v1 JSON，Result Packager 会在后续步骤处理。`
	if revisionInstructions != "" {
		systemPrompt += "\n\n# 本轮定向修订\n只修复以下审查问题，同时保留初稿中正确的信息：\n" + revisionInstructions
	}
	if strings.TrimSpace(input.Feedback) != "" {
		systemPrompt += "\n\n# 用户纠偏意见\n本次是对上一版结果的重新生成。必须优先处理以下用户纠偏，同时保留未被纠正的有效内容：\n" + input.Feedback
	}
	query := "原始请求：\n" + input.Prompt +
		"\n\n用户确认信息：\n" + expertConfigJSON(input.Answers) +
		"\n\n执行计划：\n" + expertConfigJSON(plan)
	if strings.TrimSpace(input.Feedback) != "" {
		query += "\n\n用户纠偏意见：\n" + input.Feedback
	}
	thinking := expertDefinitionBool(definition.CompiledConfig, "thinking", true)
	citations := false
	config := &types.AgentConfig{
		MaxIterations:         expertDefinitionInt(definition.CompiledConfig, "max_iterations", 4, 8),
		Temperature:           expertDefinitionFloat(definition.CompiledConfig, "temperature", 0.2),
		SystemPrompt:          systemPrompt,
		UseCustomSystemPrompt: true,
		AllowedTools:          []string{},
		Thinking:              &thinking,
		CitationEnabled:       &citations,
		LLMCallTimeout:        180,
		MaxContextTokens:      120000,
	}
	engine := agentcore.NewAgentEngine(
		config,
		chatModel,
		agenttools.NewToolRegistry(),
		nil,
		nil,
		nil,
		run.ID,
		systemPrompt,
	)
	if engine == nil {
		err := errors.New("failed to initialize AgentEngine")
		_ = s.failExpertRunStep(ctx, step, err)
		return "", err
	}
	state, err := engine.Execute(ctx, run.ID, step.ID, query, nil)
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return "", fmt.Errorf("expert writer: %w", err)
	}
	report := strings.TrimSpace(state.FinalAnswer)
	if report == "" {
		err := errors.New("expert writer returned an empty report")
		_ = s.failExpertRunStep(ctx, step, err)
		return "", invalidAgentRunOutputError{err: err}
	}
	output := types.JSONMap{
		"report_markdown": report,
		"rounds":          state.CurrentRound,
		"agent_steps":     len(state.RoundSteps),
	}
	if err := s.completeExpertRunStep(ctx, step, output); err != nil {
		return "", err
	}
	return report, nil
}

func (s *agentRunService) runExpertQualityReview(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	plan expertExecutionPlan,
	report string,
) (expertQualityAssessment, error) {
	step, err := s.startExpertRunStep(ctx, run, types.AgentRunStepTypeReviewing, types.AgentRunPhaseReviewing, modelID, types.JSONMap{
		"report_runes": utf8.RuneCountInString(report),
	})
	if err != nil {
		return expertQualityAssessment{}, err
	}
	system := `你是独立的专家交付质量审查员，不是报告作者。
按质量规则严格评分，不因为文风流畅而放宽要求。
检查需求覆盖、结构完整、具体可执行、时间/人数/预算一致性、事实与假设区分、安全合规和产物可读性。
发现问题时必须指出位置、严重级别和可以直接执行的修改指令。
伪造证据、未标注关键假设、安全合规违规和越权动作属于红线。`
	user := "专家定义：\n" + definition.SystemPrompt +
		"\n\n用户请求：\n" + input.Prompt +
		"\n\n确认信息：\n" + expertConfigJSON(input.Answers) +
		"\n\n用户纠偏意见：\n" + firstNonEmpty(input.Feedback, "无") +
		"\n\n执行计划：\n" + expertConfigJSON(plan) +
		"\n\n质量规则：\n" + expertConfigJSON(definition.CompiledConfig["quality_rubric"]) +
		"\n\n待审查报告：\n" + report
	var quality expertQualityAssessment
	err = expertChatJSON(ctx, chatModel, system, user, expertQualitySchema, &quality,
		0.1,
		false,
		expertAgentTestDefaultTokens)
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return expertQualityAssessment{}, fmt.Errorf("expert quality review: %w", err)
	}
	normalizeExpertQuality(&quality, definition.CompiledConfig, plan, report)
	output, _ := agentRunJSONMap(quality)
	if err := s.completeExpertRunStep(ctx, step, output); err != nil {
		return expertQualityAssessment{}, err
	}
	return quality, nil
}

func (s *agentRunService) runExpertPackaging(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	report string,
	quality expertQualityAssessment,
) (types.AgentResultV1, error) {
	step, err := s.startExpertRunStep(ctx, run, types.AgentRunStepTypePackaging, types.AgentRunPhasePackaging, modelID, types.JSONMap{
		"quality_score":  quality.Score,
		"quality_passed": quality.Passed,
	})
	if err != nil {
		return types.AgentResultV1{}, err
	}
	system := `你是睿乐 Result Packager，只负责把已经完成的 Markdown 报告转换为 agent_result_v1。
不得删除报告中的关键执行细节。服务卡片只能包含 title、summary、next_action。
详细内容放入唯一的 primary structured_report_v1 报告。
可以将原报告各章节转换成多个 sections；无法归类的正文使用 analysis。
不要编造证据；没有平台证据时 evidence 与 evidence_refs 返回空数组。`
	if !quality.Passed {
		system += "\n质量门禁未通过：decision.should_create_card 必须为 false，省略 card，但仍保留报告供管理员检查。"
	}
	user := "原始请求：\n" + input.Prompt +
		"\n\n质量结论：\n" + expertConfigJSON(quality) +
		"\n\n最终 Markdown 报告：\n" + report
	var result types.AgentResultV1
	err = expertChatJSON(ctx, chatModel, system, user, expertAgentResultSchema, &result,
		0.1,
		false,
		expertAgentTestMaxTokens)
	if err != nil {
		result = fallbackExpertAgentResult(definition, report, quality)
	}
	if !quality.Passed {
		result.Decision.ShouldCreateCard = false
		result.Decision.Confidence = minFloat(result.Decision.Confidence, 0.49)
		result.Decision.Reason = firstNonEmpty(strings.TrimSpace(quality.Summary), "质量门禁未通过，结果仅作为管理员候选草稿。")
		result.Card = nil
	}
	if err := validateExpertTestResultShape(result); err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return types.AgentResultV1{}, invalidAgentRunOutputError{err: err}
	}
	output, _ := result.ToJSONMap()
	if err := s.completeExpertRunStep(ctx, step, output); err != nil {
		return types.AgentResultV1{}, err
	}
	return result, nil
}

func (s *agentRunService) startExpertRunStep(
	ctx context.Context,
	run *types.AgentRun,
	stepType, phase, modelID string,
	input types.JSONMap,
) (*types.AgentRunStep, error) {
	if err := s.repo.UpdatePhase(ctx, run.ID, phase); err != nil {
		return nil, err
	}
	step := &types.AgentRunStep{
		RunID:    run.ID,
		StepType: stepType,
		ModelID:  modelID,
		Input:    input,
	}
	if err := s.repo.CreateStep(ctx, step); err != nil {
		return nil, err
	}
	return step, nil
}

func (s *agentRunService) completeExpertRunStep(
	ctx context.Context,
	step *types.AgentRunStep,
	output types.JSONMap,
) error {
	return s.repo.CompleteStep(ctx, step.ID, output, time.Now().UTC())
}

func (s *agentRunService) failExpertRunStep(
	ctx context.Context,
	step *types.AgentRunStep,
	err error,
) error {
	return s.repo.FailStep(ctx, step.ID, truncateAgentRunError(err), time.Now().UTC())
}

func expertChatJSON(
	ctx context.Context,
	model chat.Chat,
	system, user string,
	schema json.RawMessage,
	destination any,
	temperature float64,
	thinking bool,
	maxTokens int,
) error {
	response, err := model.Chat(ctx, []chat.Message{
		{Role: "system", Content: strings.TrimSpace(system)},
		{Role: "user", Content: strings.TrimSpace(user)},
	}, &chat.ChatOptions{
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Thinking:    &thinking,
		Format:      schema,
	})
	if err != nil {
		return err
	}
	if response == nil || strings.TrimSpace(response.Content) == "" {
		return errors.New("model returned empty JSON output")
	}
	cleaned := cleanExpertJSON(response.Content)
	if err := json.Unmarshal([]byte(cleaned), destination); err != nil {
		return fmt.Errorf("decode model JSON: %w; output=%q", err, trimMax(cleaned, 1000))
	}
	return nil
}

func cleanExpertJSON(raw string) string {
	cleaned := strings.TrimSpace(raw)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, "```"))
	start, end := strings.Index(cleaned, "{"), strings.LastIndex(cleaned, "}")
	if start >= 0 && end > start {
		return cleaned[start : end+1]
	}
	return cleaned
}

func loadExpertSkillSnapshots(definition *types.AgentDefinitionVersion) ([]expertSkillSnapshot, error) {
	if definition == nil {
		return nil, errors.New("expert definition is required")
	}
	raw := definition.CompiledConfig["skill_snapshots"]
	if raw == nil {
		if len(definition.Skills) > 0 {
			return nil, fmt.Errorf(
				"expert definition requires skills %s but has no compiled skill snapshots; import a new package version",
				strings.Join([]string(definition.Skills), ", "),
			)
		}
		return []expertSkillSnapshot{}, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encode expert skill snapshots: %w", err)
	}
	var snapshots []expertSkillSnapshot
	if err := json.Unmarshal(data, &snapshots); err != nil {
		return nil, fmt.Errorf("decode expert skill snapshots: %w", err)
	}
	byName := make(map[string]expertSkillSnapshot, len(snapshots))
	for _, snapshot := range snapshots {
		snapshot.Name = strings.TrimSpace(snapshot.Name)
		if snapshot.Name == "" || strings.TrimSpace(snapshot.Content) == "" {
			continue
		}
		if snapshot.SHA256 == "" || snapshot.SHA256 != expertStringHash(snapshot.Content) {
			return nil, fmt.Errorf("expert skill snapshot hash mismatch: %s", snapshot.Name)
		}
		byName[snapshot.Name] = snapshot
	}
	result := make([]expertSkillSnapshot, 0, len(definition.Skills))
	for _, skillName := range definition.Skills {
		snapshot, ok := byName[skillName]
		if !ok {
			return nil, fmt.Errorf("required expert skill snapshot is missing: %s", skillName)
		}
		result = append(result, snapshot)
	}
	return result, nil
}

func renderExpertSkillContext(skills []expertSkillSnapshot) string {
	if len(skills) == 0 {
		return ""
	}
	var builder strings.Builder
	builder.WriteString("\n\n# 已锁定的专家 Skills\n")
	for _, skill := range skills {
		builder.WriteString("\n## Skill: ")
		builder.WriteString(skill.Name)
		builder.WriteString("\n来源：")
		builder.WriteString(skill.Path)
		builder.WriteString("\n哈希：")
		builder.WriteString(skill.SHA256)
		builder.WriteString("\n\n")
		builder.WriteString(skill.Content)
		builder.WriteString("\n")
	}
	return builder.String()
}

func expertSkillTrace(skills []expertSkillSnapshot) []map[string]any {
	result := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		result = append(result, map[string]any{
			"name":   skill.Name,
			"path":   skill.Path,
			"sha256": skill.SHA256,
			"runes":  utf8.RuneCountInString(skill.Content),
		})
	}
	return result
}

func normalizeExpertIntakeQuestions(questions []types.ExpertIntakeQuestion) []types.ExpertIntakeQuestion {
	result := make([]types.ExpertIntakeQuestion, 0, minInt(len(questions), 5))
	seen := map[string]bool{}
	for index, question := range questions {
		if len(result) >= 5 {
			break
		}
		question.Label = strings.TrimSpace(question.Label)
		if question.Label == "" {
			continue
		}
		question.ID = normalizeExpertQuestionID(question.ID)
		if question.ID == "" {
			question.ID = fmt.Sprintf("question_%d", index+1)
		}
		if seen[question.ID] {
			question.ID = fmt.Sprintf("%s_%d", question.ID, index+1)
		}
		seen[question.ID] = true
		switch question.Type {
		case "single_choice", "multi_choice", "date", "number", "text":
		default:
			question.Type = "text"
		}
		question.Options = nonBlankStrings(question.Options...)
		if (question.Type == "single_choice" || question.Type == "multi_choice") && len(question.Options) == 0 {
			question.Type = "text"
		}
		result = append(result, question)
	}
	return result
}

func deterministicExpertIntakeQuestions(
	config types.JSONMap,
	answers types.JSONMap,
) []types.ExpertIntakeQuestion {
	raw := config["required_inputs"]
	if raw == nil {
		return []types.ExpertIntakeQuestion{}
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return []types.ExpertIntakeQuestion{}
	}
	var requiredInputs []expertRequiredInput
	if err := json.Unmarshal(data, &requiredInputs); err != nil {
		return []types.ExpertIntakeQuestion{}
	}
	questions := make([]types.ExpertIntakeQuestion, 0, minInt(len(requiredInputs), 5))
	for _, requiredInput := range requiredInputs {
		if len(questions) >= 5 {
			break
		}
		id := normalizeExpertQuestionID(requiredInput.ID)
		if id == "" || (!requiredInput.Required && !requiredInput.AskWhenMissing) || !expertAnswerBlank(answers[id]) {
			continue
		}
		questionType := strings.ToLower(strings.TrimSpace(requiredInput.Type))
		switch questionType {
		case "enum", "select", "choice":
			questionType = "single_choice"
		case "array", "multi_enum", "multi-select", "multiselect":
			questionType = "multi_choice"
		case "string", "":
			questionType = "text"
		}
		questions = append(questions, types.ExpertIntakeQuestion{
			ID:          id,
			Label:       firstNonEmpty(strings.TrimSpace(requiredInput.Label), requiredInput.ID),
			Type:        questionType,
			Required:    requiredInput.Required,
			Options:     nonBlankStrings(requiredInput.Options...),
			Description: strings.TrimSpace(requiredInput.Description),
		})
	}
	return normalizeExpertIntakeQuestions(questions)
}

func normalizeExpertQuestionID(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			builder.WriteRune(current)
			lastUnderscore = false
			continue
		}
		if builder.Len() > 0 && !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func decodeExpertIntakeInteraction(value types.JSONMap) (types.ExpertIntakeInteraction, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return types.ExpertIntakeInteraction{}, err
	}
	var interaction types.ExpertIntakeInteraction
	if err := json.Unmarshal(raw, &interaction); err != nil {
		return types.ExpertIntakeInteraction{}, err
	}
	if interaction.SchemaVersion != "intake_request_v1" || len(interaction.Questions) == 0 {
		return types.ExpertIntakeInteraction{}, errors.New("invalid intake interaction")
	}
	return interaction, nil
}

func validateExpertIntakeAnswers(
	interaction types.ExpertIntakeInteraction,
	answers types.JSONMap,
) error {
	for _, question := range interaction.Questions {
		value, exists := answers[question.ID]
		if question.Required && (!exists || expertAnswerBlank(value)) {
			return fmt.Errorf("%s is required", question.Label)
		}
		if !exists || expertAnswerBlank(value) || len(question.Options) == 0 {
			continue
		}
		allowed := make(map[string]bool, len(question.Options))
		for _, option := range question.Options {
			allowed[option] = true
		}
		switch typed := value.(type) {
		case string:
			if !allowed[strings.TrimSpace(typed)] {
				return fmt.Errorf("%s contains an unsupported option", question.Label)
			}
		case []any:
			for _, entry := range typed {
				text, ok := entry.(string)
				if !ok || !allowed[strings.TrimSpace(text)] {
					return fmt.Errorf("%s contains an unsupported option", question.Label)
				}
			}
		}
	}
	for _, question := range interaction.Questions {
		value, exists := answers[question.ID]
		if !exists || expertAnswerBlank(value) {
			continue
		}
		switch question.Type {
		case "text":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s must be text", question.Label)
			}
		case "single_choice":
			if _, ok := value.(string); !ok {
				return fmt.Errorf("%s must contain one option", question.Label)
			}
		case "multi_choice":
			switch value.(type) {
			case []any, []string:
			default:
				return fmt.Errorf("%s must contain multiple options", question.Label)
			}
		case "number":
			if !expertAnswerIsNumber(value) {
				return fmt.Errorf("%s must be a number", question.Label)
			}
		case "date":
			text, ok := value.(string)
			if !ok || !expertAnswerIsDate(text) {
				return fmt.Errorf("%s must be a date in YYYY-MM-DD format", question.Label)
			}
		}
	}
	return nil
}

func expertAnswerIsNumber(value any) bool {
	switch typed := value.(type) {
	case int, int32, int64, uint, uint32, uint64, float32, float64, json.Number:
		return true
	case string:
		var number json.Number = json.Number(strings.TrimSpace(typed))
		_, err := number.Float64()
		return err == nil
	default:
		return false
	}
}

func expertAnswerIsDate(value string) bool {
	_, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	return err == nil
}

func expertAnswerBlank(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case []string:
		return len(typed) == 0
	default:
		return false
	}
}

func normalizeExpertQuality(
	quality *expertQualityAssessment,
	config types.JSONMap,
	plan expertExecutionPlan,
	report string,
) {
	if quality.Dimensions == nil {
		quality.Dimensions = map[string]int{}
	}
	if quality.RedLines == nil {
		quality.RedLines = []string{}
	}
	if quality.Issues == nil {
		quality.Issues = []expertQualityIssue{}
	}
	if quality.Score < 0 {
		quality.Score = 0
	}
	if quality.Score > 100 {
		quality.Score = 100
	}
	applyExpertDeterministicQuality(quality, config, plan, report)
	minimum := expertQualityMinimumScore(config)
	quality.Passed = quality.Score >= minimum && len(quality.RedLines) == 0 &&
		!expertQualityHasBlockingIssue(quality.Issues)
	if !quality.Passed && strings.TrimSpace(quality.Summary) == "" {
		quality.Summary = fmt.Sprintf("质量得分 %d，未达到 %d 分门槛。", quality.Score, minimum)
	}
}

func applyExpertDeterministicQuality(
	quality *expertQualityAssessment,
	config types.JSONMap,
	plan expertExecutionPlan,
	report string,
) {
	if utf8.RuneCountInString(report) < expertWorkflowMinReportRunes {
		quality.Score = minInt(quality.Score, 60)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "report_too_short",
			Severity:    "error",
			Message:     fmt.Sprintf("报告少于 %d 个字符，无法形成完整交付。", expertWorkflowMinReportRunes),
			Instruction: "补齐专家定义要求的章节、表格和执行细节。",
		})
	}
	deliverable := expertDefinitionMap(config, "deliverable_spec")
	for _, section := range expertAnyStringList(deliverable["required_sections"]) {
		if expertReportContainsSection(report, section) {
			continue
		}
		quality.Score = minInt(quality.Score, 70)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "required_section_missing",
			Severity:    "error",
			Section:     section,
			Message:     fmt.Sprintf("报告缺少必需章节：%s。", section),
			Instruction: fmt.Sprintf("增加“%s”章节，并补齐与用户需求相关的具体内容。", section),
		})
	}
	clarification := expertDefinitionMap(config, "clarification_policy")
	if expertAnyBool(clarification["require_assumption_labels"]) &&
		len(nonBlankStrings(plan.Assumptions...)) > 0 &&
		!strings.Contains(report, "假设") {
		quality.Score = minInt(quality.Score, 70)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "assumptions_not_labeled",
			Severity:    "error",
			Section:     "假设与待确认项",
			Message:     "执行计划包含假设，但报告未显式标注假设。",
			Instruction: "增加“假设与待确认项”章节，逐条标注假设及其对方案的影响。",
		})
	}
}

func expertQualityHasBlockingIssue(issues []expertQualityIssue) bool {
	for _, issue := range issues {
		if issue.Severity == "error" || issue.Severity == "red_line" {
			return true
		}
	}
	return false
}

func appendExpertQualityIssue(quality *expertQualityAssessment, issue expertQualityIssue) {
	for _, existing := range quality.Issues {
		if existing.Code == issue.Code && existing.Section == issue.Section {
			return
		}
	}
	quality.Issues = append(quality.Issues, issue)
}

func expertAnyStringList(value any) []string {
	data, err := json.Marshal(value)
	if err != nil {
		return []string{}
	}
	var values []string
	if err := json.Unmarshal(data, &values); err != nil {
		return []string{}
	}
	return nonBlankStrings(values...)
}

func expertAnyBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}

func expertReportContainsSection(report, section string) bool {
	report = strings.ToLower(report)
	section = strings.ToLower(strings.TrimSpace(section))
	if section == "" {
		return true
	}
	candidates := []string{section}
	switch section {
	case "facts":
		candidates = append(candidates, "事实", "确认信息", "已知信息")
	case "analysis":
		candidates = append(candidates, "分析", "判断")
	case "risks":
		candidates = append(candidates, "风险")
	case "missing_information":
		candidates = append(candidates, "缺失信息", "待确认", "信息缺口")
	case "recommended_actions":
		candidates = append(candidates, "建议动作", "建议行动", "下一步", "执行建议")
	case "talk_track":
		candidates = append(candidates, "沟通参考", "沟通话术", "话术")
	case "evidence":
		candidates = append(candidates, "证据", "依据")
	}
	for _, candidate := range candidates {
		if strings.Contains(report, strings.ToLower(candidate)) {
			return true
		}
	}
	return false
}

func expertQualityMinimumScore(config types.JSONMap) int {
	rubric := expertDefinitionMap(config, "quality_rubric")
	score := expertAnyInt(rubric["minimum_score"])
	if score <= 0 || score > 100 {
		return expertWorkflowDefaultQualityScore
	}
	return score
}

func expertMaxRevisionRounds(config types.JSONMap) int {
	policy := expertDefinitionMap(config, "execution_policy")
	rounds := expertAnyInt(policy["max_revision_rounds"])
	if rounds < 0 {
		return 0
	}
	if rounds == 0 {
		return expertWorkflowDefaultRevisions
	}
	return minInt(rounds, 2)
}

func expertDefinitionMap(config types.JSONMap, key string) map[string]any {
	raw := config[key]
	data, err := json.Marshal(raw)
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	if json.Unmarshal(data, &result) != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func expertAnyInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case json.Number:
		number, _ := typed.Int64()
		return int(number)
	default:
		return 0
	}
}

func expertRevisionInstructions(quality expertQualityAssessment) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("当前得分：%d。%s\n", quality.Score, strings.TrimSpace(quality.Summary)))
	for index, issue := range quality.Issues {
		builder.WriteString(fmt.Sprintf(
			"%d. [%s] %s：%s\n修改要求：%s\n",
			index+1,
			issue.Severity,
			firstNonEmpty(issue.Section, issue.Code),
			issue.Message,
			issue.Instruction,
		))
	}
	for _, redLine := range quality.RedLines {
		builder.WriteString("- 必须消除红线：" + redLine + "\n")
	}
	return strings.TrimSpace(builder.String())
}

func fallbackExpertAgentResult(
	definition *types.AgentDefinitionVersion,
	report string,
	quality expertQualityAssessment,
) types.AgentResultV1 {
	title := strings.TrimSpace(definition.DisplayName) + "执行报告"
	result := types.AgentResultV1{
		SchemaVersion: types.AgentResultSchemaV1,
		Decision: types.AgentResultDecisionV1{
			ShouldCreateCard: quality.Passed,
			Confidence:       float64(quality.Score) / 100,
			Reason:           firstNonEmpty(strings.TrimSpace(quality.Summary), "专家工作流已完成。"),
		},
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:   types.AgentArtifactKindReport,
				Role:   types.AgentArtifactRolePrimary,
				Title:  title,
				Format: types.StructuredReportFormatV1,
				Content: types.JSONMap{
					"format":            types.StructuredReportFormatV1,
					"title":             title,
					"executive_summary": firstNonEmpty(strings.TrimSpace(quality.Summary), "专家工作流已生成详细报告。"),
					"sections": []map[string]any{
						{
							"type":    "analysis",
							"title":   "完整报告",
							"content": report,
							"items":   []string{},
						},
					},
					"evidence_refs": []string{},
				},
			},
		},
		Evidence: []types.AgentEvidenceRefV1{},
	}
	if quality.Passed {
		result.Card = &types.ServiceCardV1{
			SchemaVersion: types.ServiceCardSchemaV1,
			Title:         trimExpertRunes(title, 80),
			Summary:       trimExpertRunes(firstNonEmpty(strings.TrimSpace(quality.Summary), "专家报告已生成并通过质量检查。"), 320),
			NextAction:    "查看详细报告并确认执行参数。",
		}
	}
	return result
}

func expertConfigJSON(value any) string {
	if value == nil {
		return "{}"
	}
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func trimExpertRunes(value string, limit int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func minFloat(left, right float64) float64 {
	if left < right {
		return left
	}
	return right
}
