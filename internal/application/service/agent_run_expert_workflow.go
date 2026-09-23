package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
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
	expertWorkflowDefaultQualityScore  = 85
	expertWorkflowDefaultRevisions     = 2
	expertWorkflowMinReportRunes       = 1200
	expertWorkflowClarificationRounds  = 1
	expertWorkflowClarificationItems   = 4
	expertWorkflowMaxParentReportRunes = 40000
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

var (
	expertDatePattern   = regexp.MustCompile(`(20\d{2})\s*(?:年|[-/.])\s*(\d{1,2})\s*(?:月|[-/.])\s*(\d{1,2})\s*日?`)
	expertAmountPattern = regexp.MustCompile(`-?\d[\d,]*(?:\.\d+)?`)
)

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
	parentReport, err := s.loadExpertParentReport(ctx, run)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}
	threadContext, err := s.renderExpertThreadContext(ctx, run)
	if err != nil {
		return types.AgentResultV1{}, nil, "", permanentAgentRunError{err: err}
	}

	intake, err := s.runExpertIntake(
		ctx,
		run,
		input,
		definition,
		modelID,
		chatModel,
		skillContext,
		parentReport,
	)
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
	plan, err := s.runExpertPlanning(
		ctx,
		run,
		input,
		definition,
		modelID,
		chatModel,
		skillContext,
		parentReport,
		threadContext,
	)
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
		parentReport,
		threadContext,
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
			parentReport,
			threadContext,
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
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeQualityUpdated, types.JSONMap{
		"runId":   run.ID,
		"phase":   types.AgentRunPhaseReviewing,
		"status":  types.AgentRunStatusRunning,
		"quality": cloneAgentRunJSONMap(qualityMap),
	})

	result, err := s.runExpertPackaging(
		ctx,
		run,
		input,
		definition,
		modelID,
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
	}, "", nil
}

func (s *agentRunService) runExpertIntake(
	ctx context.Context,
	run *types.AgentRun,
	input types.ExpertAgentTestInput,
	definition *types.AgentDefinitionVersion,
	modelID string,
	chatModel chat.Chat,
	skillContext string,
	parentReport string,
) (expertIntakeResult, error) {
	stepInput := types.JSONMap{"prompt": input.Prompt}
	if run.ParentRunID != "" {
		stepInput["parent_run_id"] = run.ParentRunID
	}
	step, err := s.startExpertRunStep(
		ctx,
		run,
		types.AgentRunStepTypeIntake,
		types.AgentRunPhaseIntake,
		modelID,
		stepInput,
	)
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
	if strings.TrimSpace(parentReport) != "" {
		intake := expertIntakeResult{
			Ready: true,
			Assumptions: []string{
				"本次沿用上一版已确认的需求与合理假设，仅按用户纠偏生成新版本，不重复发起需求澄清。",
			},
		}
		output, _ := agentRunJSONMap(intake)
		if err := s.completeExpertRunStep(ctx, step, output); err != nil {
			return expertIntakeResult{}, err
		}
		return intake, nil
	}

	previousSteps, err := s.repo.ListSteps(ctx, run.ID)
	if err != nil {
		_ = s.failExpertRunStep(ctx, step, err)
		return expertIntakeResult{}, fmt.Errorf("list prior expert intake steps: %w", err)
	}
	completedIntakeRounds := 0
	for _, previousStep := range previousSteps {
		if previousStep.ID == step.ID {
			continue
		}
		if previousStep.StepType == types.AgentRunStepTypeIntake &&
			previousStep.Status == types.AgentRunStepStatusSucceeded {
			completedIntakeRounds++
		}
	}
	maxClarificationRounds := expertClarificationMaxRounds(definition.CompiledConfig)
	if completedIntakeRounds >= maxClarificationRounds {
		intake := expertIntakeResult{
			Ready: true,
			Assumptions: []string{
				"未提供的微观执行参数按行业常规给出建议值，并在报告中标记为待确认项，不阻断首版方案生成。",
			},
		}
		output, _ := agentRunJSONMap(intake)
		if err := s.completeExpertRunStep(ctx, step, output); err != nil {
			return expertIntakeResult{}, err
		}
		return intake, nil
	}

	maxQuestions := expertClarificationMaxQuestions(definition.CompiledConfig)
	system := strings.TrimSpace(definition.SystemPrompt) + skillContext + fmt.Sprintf(`

# 睿乐需求澄清
你现在只判断信息是否足以开始正式交付，不生成方案。
以下平台交互策略优先于专家正文中的询问习惯。
目标是尽快形成一版宏观可用交付，而不是在生成前收集全部执行细节。
只识别会改变交付类型、核心目标、目标对象、总体范围或硬性边界的关键变量。
用户已经明确提供的信息不要重复询问，优先提供单选项。
本次最多提出 %d 个问题，且平台默认只允许一轮需求澄清。
人员姓名、品牌型号、精确时间和数量、可选偏好、实现参数、辅助材料和其他微观细节不得阻断首版交付。
这些细节缺失时，应使用行业常规建议、留空位或列入“假设与待确认项”。
只有缺失信息导致交付类型、核心目标或基本对象都无法判断，或存在无法用保守假设规避的安全与合规红线时，才返回 ready=false。
如果信息足够，返回 ready=true 和空 questions。`, maxQuestions)
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
	intake.Questions = normalizeExpertIntakeQuestionsLimit(intake.Questions, maxQuestions)
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
	values := types.JSONMap{
		"prompt":  input.Prompt,
		"answers": cloneAgentRunJSONMap(input.Answers),
	}
	if run.ParentRunID != "" {
		values["parent_run_id"] = run.ParentRunID
	}
	if feedback := strings.TrimSpace(input.Feedback); feedback != "" {
		values["feedback"] = feedback
	}
	snapshot := &types.AgentRequirementSnapshot{
		RunID:       run.ID,
		Values:      values,
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
	parentReport string,
	threadContext string,
) (expertExecutionPlan, error) {
	stepInput := types.JSONMap{
		"prompt":  input.Prompt,
		"answers": cloneAgentRunJSONMap(input.Answers),
	}
	if feedback := strings.TrimSpace(input.Feedback); feedback != "" {
		stepInput["feedback"] = feedback
	}
	if run.ParentRunID != "" {
		stepInput["parent_run_id"] = run.ParentRunID
	}
	step, err := s.startExpertRunStep(
		ctx,
		run,
		types.AgentRunStepTypePlanning,
		types.AgentRunPhasePlanning,
		modelID,
		stepInput,
	)
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
	if strings.TrimSpace(parentReport) != "" {
		user += "\n\n上一版报告：\n" + parentReport
	}
	if strings.TrimSpace(threadContext) != "" {
		user += "\n\n同一专家线程的近期交互：\n" + threadContext
	}
	if feedback := strings.TrimSpace(input.Feedback); feedback != "" {
		user += "\n\n本次用户纠偏意见：\n" + feedback
	}
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
	parentReport string,
	threadContext string,
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
	if strings.TrimSpace(parentReport) != "" {
		query += "\n\n上一版报告（作为修订基线）：\n" + parentReport
	}
	if strings.TrimSpace(threadContext) != "" {
		query += "\n\n同一专家线程的近期交互：\n" + threadContext
	}
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

func (s *agentRunService) loadExpertParentReport(
	ctx context.Context,
	run *types.AgentRun,
) (string, error) {
	if run == nil {
		return "", nil
	}
	parentID := strings.TrimSpace(run.ParentRunID)
	for depth := 0; parentID != "" && depth < 12; depth++ {
		parent, err := s.repo.GetByID(ctx, parentID)
		if err != nil {
			return "", fmt.Errorf("load parent expert run: %w", err)
		}
		if parent == nil {
			return "", errors.New("parent expert run does not exist")
		}
		if parent.TenantID != run.TenantID ||
			parent.UserID != run.UserID ||
			(parent.RunType != types.AgentRunTypeExpertAgentTest && parent.RunType != types.AgentRunTypeExpertFollowUp) {
			return "", errors.New("parent expert run scope mismatch")
		}
		report, _ := parent.Result["report_markdown"].(string)
		if report == "" {
			if parent.Result != nil {
				if artifactMessage, ok := parent.Result["message"].(string); ok {
					report = artifactMessage
				}
			}
		}
		if report = strings.TrimSpace(report); report != "" {
			return trimExpertRunes(report, expertWorkflowMaxParentReportRunes), nil
		}
		parentID = strings.TrimSpace(parent.ParentRunID)
	}
	return "", nil
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
	result := deterministicExpertAgentResult(definition, input, report, quality)
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
	label := agentRunPhaseLabel(phase)
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeStepStarted, types.JSONMap{
		"runId":     run.ID,
		"stepId":    step.ID,
		"stepType":  stepType,
		"phase":     phase,
		"status":    types.AgentRunStatusRunning,
		"message":   "开始" + label,
		"label":     label,
		"modelId":   modelID,
		"startedAt": step.StartedAt,
	})
	s.emitRunEvent(ctx, run.ID, types.AgentRunEventTypeReasoningMessageContent, types.JSONMap{
		"runId":     run.ID,
		"messageId": "reasoning-" + run.ID,
		"phase":     phase,
		"status":    types.AgentRunStatusRunning,
		"delta":     "开始" + label + "。\n",
	})
	return step, nil
}

func (s *agentRunService) completeExpertRunStep(
	ctx context.Context,
	step *types.AgentRunStep,
	output types.JSONMap,
) error {
	finishedAt := time.Now().UTC()
	durationMs := elapsedMilliseconds(step.StartedAt, finishedAt)
	output = cloneAgentRunJSONMap(output)
	output["duration_ms"] = durationMs
	if err := s.repo.CompleteStep(ctx, step.ID, output, finishedAt); err != nil {
		return err
	}
	s.emitRunEvent(ctx, step.RunID, types.AgentRunEventTypeStepFinished, types.JSONMap{
		"runId":      step.RunID,
		"stepId":     step.ID,
		"stepType":   step.StepType,
		"phase":      agentRunPhaseForStep(step.StepType),
		"status":     types.AgentRunStepStatusSucceeded,
		"message":    "完成" + agentRunStepLabel(step.StepType),
		"startedAt":  step.StartedAt,
		"finishedAt": finishedAt,
		"durationMs": durationMs,
	})
	s.emitRunEvent(ctx, step.RunID, types.AgentRunEventTypeReasoningMessageContent, types.JSONMap{
		"runId":     step.RunID,
		"messageId": "reasoning-" + step.RunID,
		"phase":     agentRunPhaseForStep(step.StepType),
		"status":    types.AgentRunStatusRunning,
		"delta":     "完成" + agentRunStepLabel(step.StepType) + "。\n",
	})
	return nil
}

func (s *agentRunService) failExpertRunStep(
	ctx context.Context,
	step *types.AgentRunStep,
	err error,
) error {
	message := truncateAgentRunError(err)
	finishedAt := time.Now().UTC()
	durationMs := elapsedMilliseconds(step.StartedAt, finishedAt)
	if updateErr := s.repo.FailStep(ctx, step.ID, message, finishedAt); updateErr != nil {
		return updateErr
	}
	s.emitRunEvent(ctx, step.RunID, types.AgentRunEventTypeStepError, types.JSONMap{
		"runId":      step.RunID,
		"stepId":     step.ID,
		"stepType":   step.StepType,
		"phase":      agentRunPhaseForStep(step.StepType),
		"status":     types.AgentRunStepStatusFailed,
		"message":    message,
		"startedAt":  step.StartedAt,
		"finishedAt": finishedAt,
		"durationMs": durationMs,
	})
	return nil
}

func agentRunPhaseForStep(stepType string) string {
	switch stepType {
	case types.AgentRunStepTypeIntake:
		return types.AgentRunPhaseIntake
	case types.AgentRunStepTypePlanning:
		return types.AgentRunPhasePlanning
	case types.AgentRunStepTypeDrafting:
		return types.AgentRunPhaseDrafting
	case types.AgentRunStepTypeReviewing:
		return types.AgentRunPhaseReviewing
	case types.AgentRunStepTypeRevising:
		return types.AgentRunPhaseRevising
	case types.AgentRunStepTypePackaging:
		return types.AgentRunPhasePackaging
	default:
		return ""
	}
}

func agentRunPhaseLabel(phase string) string {
	switch phase {
	case types.AgentRunPhaseIntake:
		return "需求澄清"
	case types.AgentRunPhasePlanning:
		return "执行规划"
	case types.AgentRunPhaseDrafting:
		return "报告撰写"
	case types.AgentRunPhaseReviewing:
		return "质量审查"
	case types.AgentRunPhaseRevising:
		return "报告修订"
	case types.AgentRunPhasePackaging:
		return "结果整理"
	case types.AgentRunPhaseCompleted:
		return "任务完成"
	default:
		return firstNonEmpty(strings.TrimSpace(phase), "Agent 执行")
	}
}

func agentRunStepLabel(stepType string) string {
	return agentRunPhaseLabel(stepType)
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
	return normalizeExpertIntakeQuestionsLimit(questions, 5)
}

func normalizeExpertIntakeQuestionsLimit(
	questions []types.ExpertIntakeQuestion,
	limit int,
) []types.ExpertIntakeQuestion {
	if limit <= 0 {
		return []types.ExpertIntakeQuestion{}
	}
	limit = minInt(limit, 5)
	result := make([]types.ExpertIntakeQuestion, 0, minInt(len(questions), limit))
	seen := map[string]bool{}
	for index, question := range questions {
		if len(result) >= limit {
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
	if invalidDates := expertInvalidReportDates(report); len(invalidDates) > 0 {
		quality.Score = minInt(quality.Score, 70)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "invalid_date",
			Severity:    "error",
			Section:     "时间安排",
			Message:     "报告包含无效日期：" + strings.Join(invalidDates, "、") + "。",
			Instruction: "修正日期并重新检查时间轴、提前量和执行节点。",
		})
	}
	if mismatch, detail := expertBudgetTotalMismatch(report); mismatch {
		quality.Score = minInt(quality.Score, 70)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "budget_total_mismatch",
			Severity:    "error",
			Section:     "预算",
			Message:     detail,
			Instruction: "重新计算预算表各项金额与合计，确保数量、单价、小计和总计一致。",
		})
	}
	rubric := expertDefinitionMap(config, "quality_rubric")
	if (expertAnyBool(rubric["require_citations"]) || expertAnyBool(deliverable["require_evidence"])) &&
		!expertReportContainsSection(report, "evidence") {
		quality.Score = minInt(quality.Score, 70)
		appendExpertQualityIssue(quality, expertQualityIssue{
			Code:        "evidence_missing",
			Severity:    "error",
			Section:     "依据与引用",
			Message:     "交付规格要求提供依据或引用，但报告未包含对应章节。",
			Instruction: "增加“依据与引用”章节，区分用户输入、知识库内容、外部来源和合理假设。",
		})
	}
}

func expertInvalidReportDates(report string) []string {
	matches := expertDatePattern.FindAllStringSubmatch(report, -1)
	invalid := make([]string, 0)
	seen := map[string]struct{}{}
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		year, _ := strconv.Atoi(match[1])
		month, _ := strconv.Atoi(match[2])
		day, _ := strconv.Atoi(match[3])
		value := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
		if value.Year() == year && int(value.Month()) == month && value.Day() == day {
			continue
		}
		label := strings.TrimSpace(match[0])
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		invalid = append(invalid, label)
	}
	return invalid
}

func expertBudgetTotalMismatch(report string) (bool, string) {
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	inBudgetSection := false
	itemTotal := 0.0
	declaredTotal := 0.0
	itemCount := 0
	hasDeclaredTotal := false
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if strings.HasPrefix(line, "#") {
			heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
			inBudgetSection = strings.Contains(heading, "预算") ||
				strings.Contains(heading, "费用") ||
				strings.Contains(heading, "报价")
			continue
		}
		if !inBudgetSection || !strings.Contains(line, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "|"), "|")
		for index := range cells {
			cells[index] = strings.TrimSpace(cells[index])
		}
		if len(cells) < 2 || expertMarkdownSeparatorRow(cells) {
			continue
		}
		label := strings.Join(cells[:len(cells)-1], "")
		amount, ok := expertLastAmount(cells)
		if !ok {
			continue
		}
		if strings.Contains(label, "合计") || strings.Contains(label, "总计") ||
			strings.Contains(label, "预算总额") {
			declaredTotal = amount
			hasDeclaredTotal = true
			continue
		}
		if strings.Contains(label, "金额") || strings.Contains(label, "小计") ||
			strings.Contains(label, "单价") || strings.Contains(label, "数量") {
			continue
		}
		itemTotal += amount
		itemCount++
	}
	if !hasDeclaredTotal || itemCount == 0 {
		return false, ""
	}
	tolerance := math.Max(1, math.Abs(declaredTotal)*0.001)
	if math.Abs(itemTotal-declaredTotal) <= tolerance {
		return false, ""
	}
	return true, fmt.Sprintf("预算分项合计为 %.2f，但报告总计为 %.2f。", itemTotal, declaredTotal)
}

func expertMarkdownSeparatorRow(cells []string) bool {
	for _, cell := range cells {
		value := strings.Trim(cell, " :-")
		if value != "" {
			return false
		}
	}
	return true
}

func expertLastAmount(cells []string) (float64, bool) {
	for index := len(cells) - 1; index >= 0; index-- {
		match := expertAmountPattern.FindString(cells[index])
		if match == "" {
			continue
		}
		value, err := strconv.ParseFloat(strings.ReplaceAll(match, ",", ""), 64)
		if err == nil {
			return value, true
		}
	}
	return 0, false
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

func expertClarificationMaxRounds(config types.JSONMap) int {
	policy := expertDefinitionMap(config, "clarification_policy")
	value, configured := policy["max_rounds"]
	if !configured {
		return expertWorkflowClarificationRounds
	}
	rounds := expertAnyInt(value)
	if rounds <= 0 {
		return 0
	}
	return minInt(rounds, 3)
}

func expertClarificationMaxQuestions(config types.JSONMap) int {
	policy := expertDefinitionMap(config, "clarification_policy")
	questions := expertAnyInt(policy["max_questions"])
	if questions <= 0 {
		return expertWorkflowClarificationItems
	}
	return minInt(questions, 5)
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

func deterministicExpertAgentResult(
	definition *types.AgentDefinitionVersion,
	input types.ExpertAgentTestInput,
	report string,
	quality expertQualityAssessment,
) types.AgentResultV1 {
	structured := expertStructuredReportFromMarkdown(definition, report, quality)
	content, _ := structured.ToJSONMap()
	title := structured.Title
	summary := trimExpertRunes(expertCardPlainText(firstNonEmpty(
		expertReportLead(report),
		strings.TrimSpace(quality.Summary),
		"专家工作流已生成详细报告。",
	)), 320)
	result := types.AgentResultV1{
		SchemaVersion: types.AgentResultSchemaV1,
		Decision: types.AgentResultDecisionV1{
			ShouldCreateCard: quality.Passed,
			Confidence:       float64(quality.Score) / 100,
			Reason:           firstNonEmpty(strings.TrimSpace(quality.Summary), "专家工作流已完成。"),
		},
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:    types.AgentArtifactKindReport,
				Role:    types.AgentArtifactRolePrimary,
				Title:   title,
				Format:  types.StructuredReportFormatV1,
				Content: content,
			},
		},
		Evidence: []types.AgentEvidenceRefV1{},
	}
	if quality.Passed {
		result.Card = &types.ServiceCardV1{
			SchemaVersion: types.ServiceCardSchemaV1,
			Title:         trimExpertRunes(title, 80),
			Summary:       summary,
			NextAction:    trimExpertRunes(expertCardPlainText(expertReportNextAction(report, input)), 320),
		}
	}
	return result
}

func expertStructuredReportFromMarkdown(
	definition *types.AgentDefinitionVersion,
	report string,
	quality expertQualityAssessment,
) types.StructuredReportV1 {
	title := firstNonEmpty(expertReportTitle(report), strings.TrimSpace(definition.DisplayName)+"执行报告")
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	sections := make([]types.StructuredReportSection, 0, 8)
	currentTitle := "完整报告"
	currentType := "analysis"
	currentLines := make([]string, 0, 24)
	flush := func() {
		content := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if content == "" {
			currentLines = currentLines[:0]
			return
		}
		sections = append(sections, types.StructuredReportSection{
			Type:    currentType,
			Title:   trimExpertRunes(currentTitle, 80),
			Content: trimExpertRunes(content, 8000),
			Items:   types.StringArray{},
		})
		currentLines = currentLines[:0]
	}
	for _, line := range lines {
		match := reportHeadingPattern.FindStringSubmatch(line)
		if match == nil {
			currentLines = append(currentLines, line)
			continue
		}
		heading := strings.TrimSpace(match[2])
		if heading == "" || heading == title {
			continue
		}
		flush()
		currentTitle = heading
		currentType = expertReportSectionType(heading)
	}
	flush()
	if len(sections) == 0 {
		sections = append(sections, types.StructuredReportSection{
			Type:    "analysis",
			Title:   "完整报告",
			Content: trimExpertRunes(report, 8000),
			Items:   types.StringArray{},
		})
	}
	return types.StructuredReportV1{
		Format:           types.StructuredReportFormatV1,
		Title:            trimExpertRunes(title, 160),
		ExecutiveSummary: trimExpertRunes(firstNonEmpty(expertReportLead(report), strings.TrimSpace(quality.Summary), "专家报告已生成。"), 2000),
		Sections:         sections,
		EvidenceRefs:     types.StringArray{},
	}
}

func expertReportTitle(report string) string {
	for _, line := range strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n") {
		match := reportHeadingPattern.FindStringSubmatch(line)
		if match != nil && strings.TrimSpace(match[2]) != "" {
			return strings.TrimSpace(match[2])
		}
	}
	return ""
}

func expertReportLead(report string) string {
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	paragraph := make([]string, 0, 4)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if len(paragraph) > 0 {
				break
			}
			continue
		}
		if reportHeadingPattern.MatchString(trimmed) ||
			reportCodeFence.MatchString(trimmed) ||
			isMarkdownTableSeparator(trimmed) {
			continue
		}
		if match := reportUnorderedList.FindStringSubmatch(trimmed); match != nil {
			if len(paragraph) == 0 {
				return strings.TrimSpace(match[1])
			}
			break
		}
		if match := reportOrderedList.FindStringSubmatch(trimmed); match != nil {
			if len(paragraph) == 0 {
				return strings.TrimSpace(match[1])
			}
			break
		}
		if isMarkdownTableRow(trimmed) {
			continue
		}
		paragraph = append(paragraph, trimmed)
	}
	if len(paragraph) > 0 {
		return strings.Join(paragraph, " ")
	}
	return ""
}

func expertCardPlainText(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if value == "" {
		return ""
	}
	value = reportMarkdownLink.ReplaceAllString(value, "$1")
	value = reportInlineCode.ReplaceAllString(value, "$1")
	value = strings.NewReplacer(
		"**", "",
		"__", "",
		"~~", "",
		"`", "",
		"|", " ",
	).Replace(value)
	lines := strings.Split(value, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimLeft(line, "#>-*+ ")
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	return strings.Join(strings.Fields(strings.Join(cleaned, " ")), " ")
}

func expertReportSectionType(title string) string {
	switch {
	case strings.Contains(title, "事实"), strings.Contains(title, "背景"), strings.Contains(title, "概览"):
		return "facts"
	case strings.Contains(title, "风险"), strings.Contains(title, "安全"), strings.Contains(title, "合规"):
		return "risks"
	case strings.Contains(title, "待确认"), strings.Contains(title, "待补充"), strings.Contains(title, "假设"):
		return "missing_information"
	case strings.Contains(title, "行动"), strings.Contains(title, "执行"), strings.Contains(title, "建议"), strings.Contains(title, "下一步"):
		return "recommended_actions"
	case strings.Contains(title, "话术"), strings.Contains(title, "沟通"):
		return "talk_track"
	case strings.Contains(title, "依据"), strings.Contains(title, "证据"), strings.Contains(title, "来源"):
		return "evidence"
	default:
		return "analysis"
	}
}

func expertReportNextAction(report string, input types.ExpertAgentTestInput) string {
	lines := strings.Split(strings.ReplaceAll(report, "\r\n", "\n"), "\n")
	inActionSection := false
	for _, line := range lines {
		if match := reportHeadingPattern.FindStringSubmatch(line); match != nil {
			inActionSection = expertReportSectionType(strings.TrimSpace(match[2])) == "recommended_actions"
			continue
		}
		if !inActionSection {
			continue
		}
		if match := reportUnorderedList.FindStringSubmatch(line); match != nil {
			return strings.TrimSpace(match[1])
		}
		if match := reportOrderedList.FindStringSubmatch(line); match != nil {
			return strings.TrimSpace(match[1])
		}
		if text := strings.TrimSpace(line); text != "" {
			return text
		}
	}
	if strings.TrimSpace(input.Feedback) != "" {
		return "查看新版本报告，确认本次纠偏是否符合预期。"
	}
	return "查看详细报告，确认关键时间、人员、预算和执行边界。"
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
