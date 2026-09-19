package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type agentRunTaskEnqueuer struct {
	tasks []*asynq.Task
	err   error
}

func (e *agentRunTaskEnqueuer) Enqueue(task *asynq.Task, _ ...asynq.Option) (*asynq.TaskInfo, error) {
	if e.err != nil {
		return nil, e.err
	}
	e.tasks = append(e.tasks, task)
	return &asynq.TaskInfo{ID: "agent-run-task", Type: task.Type(), Queue: types.QueueAgent}, nil
}

func newAgentRunTestService(
	t *testing.T,
) (interfaces.AgentRunService, interfaces.AgentRunRepository, *agentRunTaskEnqueuer, *gorm.DB) {
	t.Helper()
	db := newServiceDailyReportTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&types.AgentRun{},
		&types.AgentRunEvent{},
		&types.AgentRunEventSequence{},
		&types.AgentRunStep{},
		&types.AgentRunInputRevision{},
		&types.AgentRequirementSnapshot{},
		&types.ExpertPackage{},
		&types.ExpertPackageVersion{},
		&types.AgentDefinitionVersion{},
	))

	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	baseService := NewServiceService(serviceRepo, organizeRepo)
	runRepo := repository.NewAgentRunRepository(db)
	expertRepo := repository.NewExpertPackageRepository(db)
	enqueuer := &agentRunTaskEnqueuer{}
	return NewAgentRunService(runRepo, baseService, expertRepo, nil, enqueuer), runRepo, enqueuer, db
}

type expertTestChatModel struct {
	mu              sync.Mutex
	chatResponses   []string
	streamResponses []string
	chatCalls       [][]chat.Message
	streamCalls     [][]chat.Message
}

func (m *expertTestChatModel) Chat(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (*types.ChatResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chatCalls = append(m.chatCalls, append([]chat.Message(nil), messages...))
	if len(m.chatResponses) == 0 {
		return nil, fmt.Errorf("unexpected Chat call #%d", len(m.chatCalls))
	}
	response := m.chatResponses[0]
	m.chatResponses = m.chatResponses[1:]
	return &types.ChatResponse{Content: response, FinishReason: "stop"}, nil
}

func (m *expertTestChatModel) ChatStream(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamCalls = append(m.streamCalls, append([]chat.Message(nil), messages...))
	if len(m.streamResponses) == 0 {
		return nil, fmt.Errorf("unexpected ChatStream call #%d", len(m.streamCalls))
	}
	response := m.streamResponses[0]
	m.streamResponses = m.streamResponses[1:]
	ch := make(chan types.StreamResponse, 1)
	ch <- types.StreamResponse{
		ResponseType: types.ResponseTypeAnswer,
		Content:      response,
		Done:         true,
		FinishReason: "stop",
	}
	close(ch)
	return ch, nil
}

func (m *expertTestChatModel) GetModelName() string { return "expert-test" }
func (m *expertTestChatModel) GetModelID() string   { return "chat-1" }

type expertTestModelService struct {
	interfaces.ModelService
	chatModel chat.Chat
}

func (s *expertTestModelService) ListModels(context.Context) ([]*types.Model, error) {
	return []*types.Model{
		{
			ID:        "chat-1",
			Type:      types.ModelTypeKnowledgeQA,
			Status:    types.ModelStatusActive,
			IsDefault: true,
		},
	}, nil
}

func (s *expertTestModelService) GetChatModel(_ context.Context, modelID string) (chat.Chat, error) {
	if modelID != "chat-1" {
		return nil, errors.New("unexpected model")
	}
	return s.chatModel, nil
}

func newExpertAgentRunTestService(
	t *testing.T,
	chatResponses []string,
	streamResponses []string,
) (interfaces.AgentRunService, interfaces.AgentRunRepository, *agentRunTaskEnqueuer, *gorm.DB, *expertTestChatModel) {
	t.Helper()
	db := newServiceDailyReportTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&types.AgentRun{},
		&types.AgentRunEvent{},
		&types.AgentRunEventSequence{},
		&types.AgentRunStep{},
		&types.AgentRunInputRevision{},
		&types.AgentRequirementSnapshot{},
		&types.ExpertPackage{},
		&types.ExpertPackageVersion{},
		&types.AgentDefinitionVersion{},
	))
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	runRepo := repository.NewAgentRunRepository(db)
	expertRepo := repository.NewExpertPackageRepository(db)
	enqueuer := &agentRunTaskEnqueuer{}
	chatModel := &expertTestChatModel{
		chatResponses:   append([]string(nil), chatResponses...),
		streamResponses: append([]string(nil), streamResponses...),
	}
	modelService := &expertTestModelService{chatModel: chatModel}
	runService := NewAgentRunService(
		runRepo,
		NewServiceService(serviceRepo, organizeRepo),
		expertRepo,
		modelService,
		enqueuer,
	)
	return runService, runRepo, enqueuer, db, chatModel
}

func TestAgentRunEventsReplayFromSequence(t *testing.T) {
	ctx := context.Background()
	_, runRepo, _, _ := newAgentRunTestService(t)
	run := &types.AgentRun{
		TenantID: 9,
		UserID:   "event-user",
		RunType:  types.AgentRunTypeExpertAgentTest,
		Input:    types.JSONMap{"prompt": "event replay"},
	}
	require.NoError(t, runRepo.Create(ctx, run))

	for _, eventType := range []string{
		types.AgentRunEventTypeRunQueued,
		types.AgentRunEventTypeRunStarted,
		types.AgentRunEventTypeRunFinished,
	} {
		require.NoError(t, runRepo.CreateEvent(ctx, &types.AgentRunEvent{
			RunID:     run.ID,
			EventType: eventType,
			Payload:   types.JSONMap{"runId": run.ID},
		}))
	}

	all, err := runRepo.ListEvents(ctx, run.ID, 0, 10)
	require.NoError(t, err)
	require.Len(t, all, 3)
	require.Equal(t, int64(1), all[0].Sequence)
	require.Equal(t, int64(2), all[1].Sequence)
	require.Equal(t, int64(3), all[2].Sequence)

	resumed, err := runRepo.ListEvents(ctx, run.ID, 1, 10)
	require.NoError(t, err)
	require.Len(t, resumed, 2)
	require.Equal(t, types.AgentRunEventTypeRunStarted, resumed[0].EventType)
	require.Equal(t, types.AgentRunEventTypeRunFinished, resumed[1].EventType)
}

func createAgentRunProfile(t *testing.T, db *gorm.DB, tenantID uint64, userID string) *types.UserWorkProfile {
	t.Helper()
	profile := &types.UserWorkProfile{
		TenantID:       tenantID,
		UserID:         userID,
		Name:           "服务执行测试",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	}
	require.NoError(t, db.Create(profile).Error)
	return profile
}

func TestAgentRunEnqueuesDailyReportOnceAndPersistsArtifactReference(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "user-a"

	runService, runRepo, enqueuer, db := newAgentRunTestService(t)
	profile := createAgentRunProfile(t, db, tenantID, userID)

	input := types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeDay,
		Date:     "2026-09-17",
		Timezone: "Asia/Shanghai",
		Trigger:  "user_requested",
	}
	first, err := runService.EnqueueDailyReport(ctx, tenantID, userID, input)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusQueued, first.Status)
	require.Len(t, enqueuer.tasks, 1)
	require.Equal(t, types.TypeAgentRunExecute, enqueuer.tasks[0].Type())

	duplicate, err := runService.EnqueueDailyReport(ctx, tenantID, userID, input)
	require.NoError(t, err)
	require.Equal(t, first.ID, duplicate.ID)
	require.Len(t, enqueuer.tasks, 1)

	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))
	run, err := runRepo.GetByID(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusSucceeded, run.Status, run.ErrorMessage)
	require.Equal(t, types.BuiltinServiceAssistantVersion, run.AgentVersion)
	require.Equal(t, profile.ID, run.ProfileID)
	require.Equal(t, types.AgentResultSchemaV1, run.Result["schema_version"])
	require.Equal(t, "service_daily_report", run.Result["artifact_type"])
	require.NotEmpty(t, run.Result["daily_report_id"])
	require.Equal(t, 1, run.Attempt)

	result := decodeAgentRunResult(t, run.Result)
	require.False(t, result.Decision.ShouldCreateCard)
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, types.AgentArtifactKindReport, result.Artifacts[0].Kind)
	require.Equal(t, types.StructuredReportFormatV1, result.Artifacts[0].Format)
	require.Empty(t, result.Evidence)
	validation := decodeAgentRunValidation(t, run.Result)
	require.True(t, validation.Valid)
}

func TestAgentRunCancellationStopsQueuedTask(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "user-a"

	runService, runRepo, enqueuer, _ := newAgentRunTestService(t)
	run, err := runService.EnqueueMemoryExtraction(ctx, tenantID, userID, "memory-a")
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 1)

	cancelled, err := runService.CancelAgentRun(ctx, tenantID, userID, run.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusCancelled, cancelled.Status)

	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))
	persisted, err := runRepo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusCancelled, persisted.Status)
}

func TestAgentRunRecordsTerminalFailureWithoutWorkerRetryContext(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "user-a"

	runService, runRepo, enqueuer, _ := newAgentRunTestService(t)
	run, err := runService.EnqueueDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range: types.ServiceDailyReportRangeDay,
		Date:  "2026-09-17",
	})
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 1)

	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))
	persisted, err := runRepo.GetByID(ctx, run.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusFailed, persisted.Status)
	require.Equal(t, types.AgentRunErrorExecutionFailed, persisted.ErrorCode)
	require.Contains(t, persisted.ErrorMessage, ErrServiceProfileNotConfigured.Error())
}

func TestAgentRunMarksEnqueueFailure(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "user-a"

	runService, _, enqueuer, db := newAgentRunTestService(t)
	enqueuer.err = errors.New("queue unavailable")

	run, err := runService.EnqueueMemoryExtraction(ctx, tenantID, userID, "memory-a")
	require.Error(t, err)
	require.Nil(t, run)

	var persisted types.AgentRun
	require.NoError(t, db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Order("created_at DESC").
		First(&persisted).Error)
	require.Equal(t, types.AgentRunStatusFailed, persisted.Status)
	require.Equal(t, types.AgentRunErrorEnqueueFailed, persisted.ErrorCode)
}

func TestAgentRunMemoryExtractionPersistsThreeFieldCardAndEvidence(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "user-a"

	runService, runRepo, enqueuer, db := newAgentRunTestService(t)
	profile := createAgentRunProfile(t, db, tenantID, userID)
	require.NoError(t, db.Create(&types.WorkProfileAgentSetting{
		TenantID:         tenantID,
		ProfileID:        profile.ID,
		AgentID:          types.BuiltinServiceAssistantID,
		AgentDomain:      types.ServiceAgentDomainSalesConsulting,
		Enabled:          true,
		DisplayName:      "招生咨询",
		DisplayOrder:     1,
		WorkDocDirectory: "线索/",
	}).Error)
	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "试听课安排",
		Content:  "小明妈妈已经确认本周三参加试听课，需要在试听后主动回访并确认下一步安排。",
		Source:   "手动输入",
		Metadata: types.JSONMap{"tags": []string{}},
	}
	require.NoError(t, db.Create(memory).Error)

	queued, err := runService.EnqueueMemoryExtraction(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 1)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))

	run, err := runRepo.GetByID(ctx, queued.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusSucceeded, run.Status)
	require.Equal(t, profile.ID, run.ProfileID)
	require.Equal(t, types.BuiltinServiceAssistantVersion, run.AgentVersion)
	require.Equal(t, memory.ID, run.Result["memory_id"])
	require.Equal(t, true, run.Result["generated"])
	require.NotEmpty(t, run.Result["service_reminder_id"])

	result := decodeAgentRunResult(t, run.Result)
	require.True(t, result.Decision.ShouldCreateCard)
	require.NotNil(t, result.Card)
	require.Equal(t, types.ServiceCardSchemaV1, result.Card.SchemaVersion)
	require.NotEmpty(t, result.Card.Title)
	require.NotEmpty(t, result.Card.Summary)
	require.NotEmpty(t, result.Card.NextAction)
	require.Len(t, result.Artifacts, 1)
	require.Equal(t, types.AgentArtifactKindReport, result.Artifacts[0].Kind)
	require.Equal(t, types.StructuredReportFormatV1, result.Artifacts[0].Format)
	require.NotEmpty(t, result.Evidence)
	require.Equal(t, memory.ID, result.Evidence[0].SourceID)
	require.Equal(t, "trigger", result.Evidence[0].Relation)
	require.True(t, decodeAgentRunValidation(t, run.Result).Valid)
}

func TestAgentRunExpertTestExecutesPublishedDefinitionAndKeepsResultOnRun(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "admin-user"
	response := `{
		"schema_version":"agent_result_v1",
		"decision":{"should_create_card":true,"confidence":0.92,"reason":"活动需求足以形成执行方案。"},
		"card":{
			"schema_version":"service_card_v1",
			"title":"中班中秋亲子活动筹备",
			"summary":"为30组家庭设计90分钟中秋亲子活动，预算3000元。",
			"next_action":"今天确认场地尺寸、教师名单和食品过敏信息。"
		},
		"artifacts":[{
			"kind":"report",
			"role":"primary",
			"title":"中秋亲子活动执行方案",
			"format":"structured_report_v1",
			"content":{
				"format":"structured_report_v1",
				"title":"中秋亲子活动执行方案",
				"executive_summary":"按签到、亲子任务、月饼体验和合影四个阶段推进。",
				"sections":[
					{"type":"facts","title":"已知事实","items":["中班","30组家庭","预算3000元"]},
					{"type":"recommended_actions","title":"建议动作","items":["确认场地","完成分工","采购物料"]}
				],
				"evidence_refs":[]
			}
		}],
		"evidence":[]
	}`
	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(
		t,
		[]string{
			expertTestIntakeReadyJSON(),
			expertTestPlanJSON(),
			expertTestQualityPassedJSON(),
			response,
		},
		[]string{expertTestLongReport()},
	)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	queued, err := runService.EnqueueExpertTest(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "策划一场中班中秋亲子活动，30组家庭，预算3000元，时长90分钟。",
		ModelID:      "chat-1",
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunTypeExpertAgentTest, queued.RunType)
	require.Len(t, enqueuer.tasks, 1)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))

	run, err := runRepo.GetByID(ctx, queued.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusSucceeded, run.Status)
	require.Equal(t, "kindergarten-activity-planner", run.AgentRef)
	require.Equal(t, definition.ID, run.Result["definition_id"])
	require.Equal(t, "chat-1", run.Result["model_id"])
	require.Equal(t, "expert_agent_test", run.Result["artifact_type"])
	result := decodeAgentRunResult(t, run.Result)
	require.True(t, result.Decision.ShouldCreateCard)
	require.NotNil(t, result.Card)
	require.Equal(t, "中班中秋亲子活动筹备", result.Card.Title)
	require.Len(t, result.Artifacts, 1)
	require.True(t, decodeAgentRunValidation(t, run.Result).Valid)
	require.Equal(t, types.AgentRunPhaseCompleted, run.Phase)
	require.EqualValues(t, 92, run.Quality["score"])
	require.Len(t, chatModel.chatCalls, 4)
	require.Len(t, chatModel.streamCalls, 1)
	require.Contains(t, chatModel.chatCalls[0][0].Content, "幼儿园活动策划专家")
	require.Contains(t, chatModel.chatCalls[0][0].Content, "睿乐需求澄清")
	steps, err := runService.ListAgentRunSteps(ctx, tenantID, userID, run.ID)
	require.NoError(t, err)
	require.Len(t, steps, 5)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[0].StepType)
	require.Equal(t, types.AgentRunStepTypePackaging, steps[4].StepType)
	for _, step := range steps {
		require.Equal(t, types.AgentRunStepStatusSucceeded, step.Status)
	}
}

func TestAgentRunExpertTestWaitsForAnswersAndResumesSameRun(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "admin-user"
	finalResult := `{
		"schema_version":"agent_result_v1",
		"decision":{"should_create_card":true,"confidence":0.9,"reason":"需求已补齐并通过质量检查。"},
		"card":{
			"schema_version":"service_card_v1",
			"title":"国庆亲子运动会筹备",
			"summary":"为120组家庭设计上午半日亲子运动会。",
			"next_action":"确认操场分区和各班带队教师。"
		},
		"artifacts":[{
			"kind":"report",
			"role":"primary",
			"title":"国庆亲子运动会活动方案",
			"format":"structured_report_v1",
			"content":{
				"format":"structured_report_v1",
				"title":"国庆亲子运动会活动方案",
				"executive_summary":"按入场、热身、分区项目、颁奖和离场组织。",
				"sections":[{"type":"analysis","title":"完整方案","content":"完整执行内容见工作流报告。"}],
				"evidence_refs":[]
			}
		}],
		"evidence":[]
	}`
	runService, runRepo, enqueuer, db, _ := newExpertAgentRunTestService(
		t,
		[]string{
			`{"ready":false,"questions":[{"id":"family_count","label":"参与家庭数量","type":"number","required":true},{"id":"event_date","label":"活动日期","type":"date","required":true}],"assumptions":[]}`,
			expertTestIntakeReadyJSON(),
			expertTestPlanJSON(),
			expertTestQualityPassedJSON(),
			finalResult,
		},
		[]string{expertTestLongReport()},
	)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	queued, err := runService.EnqueueExpertTest(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "策划国庆亲子运动会。",
		ModelID:      "chat-1",
	})
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 1)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))

	waiting, err := runRepo.GetByID(ctx, queued.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusWaitingInput, waiting.Status)
	require.Equal(t, types.AgentRunPhaseIntake, waiting.Phase)
	interaction, err := decodeExpertIntakeInteraction(waiting.Interaction)
	require.NoError(t, err)
	require.Len(t, interaction.Questions, 2)

	resumed, err := runService.SubmitAgentRunAnswers(ctx, tenantID, userID, waiting.ID, types.AgentRunAnswersInput{
		Answers: types.JSONMap{
			"family_count": 120,
			"event_date":   "2026-09-28",
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusQueued, resumed.Status)
	require.NotEmpty(t, resumed.RequirementSnapshotID)
	require.Len(t, enqueuer.tasks, 2)
	require.NotEqual(t, waiting.TaskID, resumed.TaskID)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[1]))

	completed, err := runRepo.GetByID(ctx, queued.ID)
	require.NoError(t, err)
	require.Equal(t, queued.ID, completed.ID)
	require.Equal(t, types.AgentRunStatusSucceeded, completed.Status, completed.ErrorMessage)
	require.Equal(t, 2, completed.Attempt)
	require.NotNil(t, completed.ResumedAt)
	result := decodeAgentRunResult(t, completed.Result)
	require.Equal(t, "国庆亲子运动会筹备", result.Card.Title)
	steps, err := runService.ListAgentRunSteps(ctx, tenantID, userID, completed.ID)
	require.NoError(t, err)
	require.Len(t, steps, 6)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[0].StepType)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[1].StepType)
	require.Equal(t, types.AgentRunStepTypePlanning, steps[2].StepType)
}

func TestAgentRunExpertTestRejectsUnpublishedDefinition(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	runService, _, _, db, _ := newExpertAgentRunTestService(t, nil, nil)
	pkg := &types.ExpertPackage{
		TenantID:     tenantID,
		PackageKey:   "testing-expert",
		DisplayName:  "测试专家",
		SourceFormat: types.ExpertPackageSourceWorkBuddy,
	}
	require.NoError(t, db.Create(pkg).Error)
	version := &types.ExpertPackageVersion{
		PackageID:   pkg.ID,
		Version:     "1.0.0",
		State:       types.ExpertPackageVersionTesting,
		PackageHash: "hash",
	}
	require.NoError(t, db.Create(version).Error)
	definition := &types.AgentDefinitionVersion{
		TenantID:         tenantID,
		PackageID:        pkg.ID,
		PackageVersionID: version.ID,
		AgentID:          "testing-expert",
		Version:          "1.0.0",
		DisplayName:      "测试专家",
		SystemPrompt:     "测试",
		OutputContract:   types.AgentResultSchemaV1,
		DefinitionHash:   "hash",
	}
	require.NoError(t, db.Create(definition).Error)

	run, err := runService.EnqueueExpertTest(ctx, tenantID, "admin-user", types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "执行测试",
		ModelID:      "chat-1",
	})
	require.ErrorIs(t, err, ErrExpertPackageNotPublished)
	require.Nil(t, run)
}

func TestValidateExpertTestResultShapeRequiresPrimaryReport(t *testing.T) {
	require.Error(t, validateExpertTestResultShape(types.AgentResultV1{}))
	require.Error(t, validateExpertTestResultShape(types.AgentResultV1{
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:   types.AgentArtifactKindText,
				Role:   types.AgentArtifactRolePrimary,
				Title:  "plain text",
				Format: "text",
			},
		},
	}))
	require.NoError(t, validateExpertTestResultShape(types.AgentResultV1{
		Artifacts: []types.AgentArtifactResultV1{
			{
				Kind:   types.AgentArtifactKindReport,
				Role:   types.AgentArtifactRolePrimary,
				Title:  "report",
				Format: types.StructuredReportFormatV1,
			},
		},
	}))
}

func decodeAgentRunResult(t *testing.T, value types.JSONMap) types.AgentResultV1 {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	var result types.AgentResultV1
	require.NoError(t, json.Unmarshal(raw, &result))
	return result
}

func decodeAgentRunValidation(t *testing.T, value types.JSONMap) types.AgentResultValidation {
	t.Helper()
	raw, err := json.Marshal(value["validation"])
	require.NoError(t, err)
	var validation types.AgentResultValidation
	require.NoError(t, json.Unmarshal(raw, &validation))
	return validation
}

func createPublishedExpertTestDefinition(
	t *testing.T,
	db *gorm.DB,
	tenantID uint64,
) (*types.ExpertPackage, *types.AgentDefinitionVersion) {
	t.Helper()
	pkg := &types.ExpertPackage{
		TenantID:     tenantID,
		PackageKey:   "kindergarten-activity-planner",
		DisplayName:  "童创",
		SourceFormat: types.ExpertPackageSourceWorkBuddy,
	}
	require.NoError(t, db.Create(pkg).Error)
	version := &types.ExpertPackageVersion{
		PackageID:   pkg.ID,
		Version:     "1.0.0",
		State:       types.ExpertPackageVersionPublished,
		PackageHash: "hash",
	}
	require.NoError(t, db.Create(version).Error)
	definition := &types.AgentDefinitionVersion{
		TenantID:         tenantID,
		PackageID:        pkg.ID,
		PackageVersionID: version.ID,
		AgentID:          "kindergarten-activity-planner",
		Version:          "1.0.0",
		DisplayName:      "童创",
		SystemPrompt:     "你是幼儿园活动策划专家。",
		OutputContract:   types.AgentResultSchemaV1,
		DefinitionHash:   "definition-hash",
		CompiledConfig: types.JSONMap{
			"allowed_tools":         []string{},
			"temperature":           0.1,
			"max_completion_tokens": 3000,
			"quality_rubric": types.JSONMap{
				"minimum_score": 85,
			},
			"execution_policy": types.JSONMap{
				"max_revision_rounds": 2,
			},
		},
	}
	require.NoError(t, db.Create(definition).Error)
	return pkg, definition
}

func expertTestIntakeReadyJSON() string {
	return `{"ready":true,"questions":[],"assumptions":[]}`
}

func expertTestPlanJSON() string {
	return `{
		"objective":"形成可直接执行的幼儿园亲子活动方案",
		"requirements":["覆盖时间、人员、物料、预算和安全安排"],
		"assumptions":[],
		"sections":["活动概览","流程安排","人员分工","物料预算","安全预案"],
		"checklist":["人数一致","预算一致","安全责任明确"]
	}`
}

func expertTestQualityPassedJSON() string {
	return `{
		"score":92,
		"passed":true,
		"summary":"报告结构完整，关键执行参数清晰。",
		"red_lines":[],
		"dimensions":{"需求覆盖":94,"可执行性":92,"安全合规":90},
		"issues":[]
	}`
}

func expertTestLongReport() string {
	section := "活动按签到入场、集体热身、分区项目、补水休整、成果展示和分批离场推进。每个环节明确负责人、协作教师、起止时间、场地边界、所需物料、风险检查和异常处置方式。"
	return "# 幼儿园亲子活动执行方案\n\n" + strings.Repeat(section+"\n\n", 24)
}

func TestDeterministicExpertIntakeQuestions(t *testing.T) {
	config := types.JSONMap{
		"required_inputs": []map[string]any{
			{
				"id":          "event_date",
				"label":       "活动日期",
				"type":        "date",
				"required":    true,
				"options":     []string{},
				"description": "用于安排筹备时间。",
			},
			{
				"id":               "audience",
				"label":            "使用对象",
				"type":             "enum",
				"required":         false,
				"ask_when_missing": true,
				"options":          []string{"园长", "教师"},
			},
			{
				"id":       "internal_note",
				"label":    "内部备注",
				"type":     "text",
				"required": false,
			},
		},
	}

	questions := deterministicExpertIntakeQuestions(config, types.JSONMap{})
	require.Len(t, questions, 2)
	require.Equal(t, "event_date", questions[0].ID)
	require.Equal(t, "date", questions[0].Type)
	require.True(t, questions[0].Required)
	require.Equal(t, "audience", questions[1].ID)
	require.Equal(t, "single_choice", questions[1].Type)
	require.False(t, questions[1].Required)

	questions = deterministicExpertIntakeQuestions(config, types.JSONMap{
		"event_date": "2026-10-01",
		"audience":   "园长",
	})
	require.Empty(t, questions)
}

func TestValidateExpertIntakeAnswersTypes(t *testing.T) {
	interaction := types.ExpertIntakeInteraction{
		SchemaVersion: "intake_request_v1",
		Questions: []types.ExpertIntakeQuestion{
			{ID: "event_date", Label: "活动日期", Type: "date", Required: true},
			{ID: "count", Label: "人数", Type: "number", Required: true},
			{ID: "audience", Label: "使用对象", Type: "single_choice", Required: true, Options: []string{"园长", "教师"}},
		},
	}
	require.NoError(t, validateExpertIntakeAnswers(interaction, types.JSONMap{
		"event_date": "2026-10-01",
		"count":      120,
		"audience":   "园长",
	}))
	require.Error(t, validateExpertIntakeAnswers(interaction, types.JSONMap{
		"event_date": "2026/10/01",
		"count":      "很多",
		"audience":   "家长",
	}))
}

func TestNormalizeExpertQualityAppliesDeterministicRules(t *testing.T) {
	quality := expertQualityAssessment{
		Score:      96,
		Passed:     true,
		Summary:    "模型认为报告质量较高。",
		RedLines:   []string{},
		Dimensions: map[string]int{},
		Issues:     []expertQualityIssue{},
	}
	config := types.JSONMap{
		"quality_rubric": types.JSONMap{"minimum_score": 85},
		"deliverable_spec": types.JSONMap{
			"required_sections": []string{"facts", "recommended_actions"},
		},
		"clarification_policy": types.JSONMap{
			"require_assumption_labels": true,
		},
	}
	plan := expertExecutionPlan{
		Assumptions: []string{"天气以活动前一周预报为准"},
	}
	normalizeExpertQuality(&quality, config, plan, strings.Repeat("只有一些泛泛而谈的内容。", 20))
	require.False(t, quality.Passed)
	require.Contains(t, quality.Issues, expertQualityIssue{
		Code:        "required_section_missing",
		Section:     "facts",
		Severity:    "error",
		Message:     "报告缺少必需章节：facts。",
		Instruction: "增加“facts”章节，并补齐与用户需求相关的具体内容。",
	})
	require.True(t, hasExpertQualityIssue(quality.Issues, "report_too_short"))
	require.True(t, hasExpertQualityIssue(quality.Issues, "assumptions_not_labeled"))
}

func hasExpertQualityIssue(issues []expertQualityIssue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestRegenerateAgentRunCreatesChildRunWithFeedback(t *testing.T) {
	ctx := context.Background()
	tenantID := uint64(11)
	userID := "admin-user"
	runService, runRepo, _, db := newAgentRunTestService(t)
	packageRecord, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	parent, err := runService.EnqueueExpertTest(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    packageRecord.ID,
		DefinitionID: definition.ID,
		Prompt:       "制定亲子活动方案",
	})
	require.NoError(t, err)
	_, err = runRepo.MarkFailed(ctx, parent.ID, types.AgentRunErrorExecutionFailed, "test failure", time.Now())
	require.NoError(t, err)

	child, err := runService.RegenerateAgentRun(ctx, tenantID, userID, parent.ID, types.AgentRunRegenerateInput{
		Feedback: "重点补充安全预案",
	})
	require.NoError(t, err)
	require.Equal(t, parent.ID, child.ParentRunID)
	require.Equal(t, "regenerate", child.TriggerType)
	require.Equal(t, types.AgentRunStatusQueued, child.Status)
	require.Equal(t, "重点补充安全预案", child.Input["feedback"])
}
