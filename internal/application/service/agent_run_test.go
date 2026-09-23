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
	fileservice "github.com/Tencent/WeKnora/internal/application/service/file"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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
	db := newAgentRunTestDB(t)
	runRepo := repository.NewAgentRunRepository(db)
	serviceSpaces := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil)
	expertRepo := repository.NewExpertPackageRepository(db)
	enqueuer := &agentRunTaskEnqueuer{}
	return NewAgentRunService(runRepo, serviceSpaces, expertRepo, nil, enqueuer, nil, nil, nil), runRepo, enqueuer, db
}

func newAgentRunTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(
		sqlite.Open("file:agent-run-"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"),
		&gorm.Config{},
	)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Session{},
		&types.ServiceSpace{},
		&types.ServiceSpaceMember{},
		&types.ServiceExpertBinding{},
		&types.ServiceArtifact{},
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
	return db
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
	db := newAgentRunTestDB(t)
	runRepo := repository.NewAgentRunRepository(db)
	serviceSpaces := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil)
	expertRepo := repository.NewExpertPackageRepository(db)
	enqueuer := &agentRunTaskEnqueuer{}
	chatModel := &expertTestChatModel{
		chatResponses:   append([]string(nil), chatResponses...),
		streamResponses: append([]string(nil), streamResponses...),
	}
	modelService := &expertTestModelService{chatModel: chatModel}
	runService := NewAgentRunService(
		runRepo,
		serviceSpaces,
		expertRepo,
		modelService,
		enqueuer,
		nil,
		fileservice.NewLocalFileService(t.TempDir(), ""),
		nil,
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
	require.Equal(t, "幼儿园亲子活动执行方案", result.Card.Title)
	require.NotContains(t, result.Card.Summary, "**")
	require.NotContains(t, result.Card.NextAction, "**")
	require.Len(t, result.Artifacts, 2)
	require.NotEmpty(t, result.Artifacts[0].ID)
	require.NotEmpty(t, result.Artifacts[0].VersionID)
	require.Equal(t, 1, result.Artifacts[0].Version)
	require.Equal(t, run.ID, result.Artifacts[0].RunID)
	require.Equal(t, types.AgentArtifactLifecycleTemporary, result.Artifacts[0].Lifecycle)
	require.True(t, result.Artifacts[0].Previewable)
	require.True(t, result.Artifacts[0].Downloadable)
	require.Equal(t, "text/html", result.Artifacts[0].MimeType)
	require.NotEmpty(t, result.Artifacts[0].OriginalName)
	require.True(t, strings.HasSuffix(result.Artifacts[0].OriginalName, ".html"))
	require.Equal(t, types.AgentArtifactKindReport, result.Artifacts[0].Kind)
	require.Positive(t, result.Artifacts[0].SizeBytes)
	require.NotEmpty(t, result.Artifacts[0].ResourceRef)
	require.NotEmpty(t, result.Artifacts[1].ID)
	require.NotEmpty(t, result.Artifacts[1].VersionID)
	require.Equal(t, 1, result.Artifacts[1].Version)
	require.Equal(t, run.ID, result.Artifacts[1].RunID)
	require.Equal(t, types.AgentArtifactKindText, result.Artifacts[1].Kind)
	require.Equal(t, types.AgentArtifactRoleSupporting, result.Artifacts[1].Role)
	require.Equal(t, "markdown", result.Artifacts[1].Format)
	require.Equal(t, "text/markdown; charset=utf-8", result.Artifacts[1].MimeType)
	require.True(t, strings.HasSuffix(result.Artifacts[1].OriginalName, ".md"))
	require.Positive(t, result.Artifacts[1].SizeBytes)
	require.NotEmpty(t, result.Artifacts[1].ResourceRef)
	require.True(t, decodeAgentRunValidation(t, run.Result).Valid)
	require.Equal(t, types.AgentRunPhaseCompleted, run.Phase)
	require.EqualValues(t, 92, run.Quality["score"])
	require.Len(t, chatModel.chatCalls, 3)
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
		require.Contains(t, step.Output, "duration_ms")
	}
	events, err := runService.ListAgentRunEvents(ctx, tenantID, userID, run.ID, 0, 200)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	eventTypes := make(map[string]bool, len(events))
	var previousSequence int64
	var finishedStepDuration int
	var totalDuration int
	for _, event := range events {
		eventTypes[event.EventType] = true
		require.Greater(t, event.Sequence, previousSequence)
		previousSequence = event.Sequence
		if event.EventType == types.AgentRunEventTypeStepFinished {
			finishedStepDuration = expertAnyInt(event.Payload["durationMs"])
		}
		if event.EventType == types.AgentRunEventTypeRunFinished {
			totalDuration = expertAnyInt(event.Payload["totalDurationMs"])
		}
	}
	require.True(t, eventTypes[types.AgentRunEventTypeRunQueued])
	require.True(t, eventTypes[types.AgentRunEventTypeRunStarted])
	require.True(t, eventTypes[types.AgentRunEventTypeStepStarted])
	require.True(t, eventTypes[types.AgentRunEventTypeQualityUpdated])
	require.True(t, eventTypes[types.AgentRunEventTypeArtifactVersionCreated])
	require.True(t, eventTypes[types.AgentRunEventTypeActivitySnapshot])
	require.True(t, eventTypes[types.AgentRunEventTypeRunFinished])
	require.GreaterOrEqual(t, finishedStepDuration, 0)
	require.GreaterOrEqual(t, totalDuration, 0)
}

func TestServiceAgentRunKeepsSessionBoundaryAndIndexesArtifacts(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 15
	const userID = "service-owner"
	runService, _, enqueuer, db, _ := newExpertAgentRunTestService(
		t,
		[]string{
			expertTestIntakeReadyJSON(),
			expertTestPlanJSON(),
			expertTestQualityPassedJSON(),
			`{
				"schema_version":"agent_result_v1",
				"decision":{"should_create_card":true,"confidence":0.9,"reason":"已生成服务产出物。"},
				"card":{"schema_version":"service_card_v1","title":"招生跟进清单","summary":"已整理重点跟进家长。","next_action":"逐一确认到访时间。"},
				"artifacts":[{
					"kind":"report",
					"role":"primary",
					"title":"招生跟进清单",
					"format":"structured_report_v1",
					"content":{
						"format":"structured_report_v1",
						"title":"招生跟进清单",
						"executive_summary":"本次会话形成招生跟进清单。",
						"sections":[{"type":"recommended_actions","title":"下一步","items":["确认到访时间","补齐联系方式"]}],
						"evidence_refs":[]
					}
				}],
				"evidence":[]
			}`,
		},
		[]string{expertTestLongReport()},
	)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)
	serviceSpaces := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil)
	space, err := serviceSpaces.Create(ctx, tenantID, userID, types.ServiceSpaceCreateInput{
		Name: "秋季招生咨询",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: definition.AgentID, ExpertName: definition.DisplayName},
		},
		Activate: true,
	})
	require.NoError(t, err)
	session, err := serviceSpaces.CreateSession(ctx, tenantID, userID, space.ID, types.ServiceSessionCreateInput{})
	require.NoError(t, err)

	run, err := runService.EnqueuePublishedExpertRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "整理今天开放日需要重点跟进的家长。",
		ModelID:      "chat-1",
		ServiceID:    space.ID,
		SessionID:    session.ID,
	})
	require.NoError(t, err)
	require.Equal(t, space.ID, run.ServiceID)
	require.Equal(t, session.ID, run.ThreadID)
	require.Len(t, enqueuer.tasks, 1)

	_, err = runService.GetAgentRun(ctx, tenantID, userID, run.ID)
	require.ErrorIs(t, err, ErrAgentRunNotFound)
	serviceRun, err := runService.GetAgentRunForService(ctx, tenantID, userID, space.ID, run.ID)
	require.NoError(t, err)
	require.Equal(t, run.ID, serviceRun.ID)

	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[0]))
	artifacts, total, err := serviceSpaces.ListArtifacts(ctx, tenantID, userID, space.ID, "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, artifacts, 2)
	for _, artifact := range artifacts {
		require.Equal(t, space.ID, artifact.ServiceID)
		require.Equal(t, run.ID, artifact.RunID)
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
	require.Equal(t, "幼儿园亲子活动执行方案", result.Card.Title)
	steps, err := runService.ListAgentRunSteps(ctx, tenantID, userID, completed.ID)
	require.NoError(t, err)
	require.Len(t, steps, 6)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[0].StepType)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[1].StepType)
	require.Equal(t, true, steps[1].Output["ready"])
	require.Empty(t, steps[1].Output["questions"])
	require.Equal(t, types.AgentRunStepTypePlanning, steps[2].StepType)
	events, err := runService.ListAgentRunEvents(ctx, tenantID, userID, completed.ID, 0, 200)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	var waitingSequence, resumedSequence int64
	for _, event := range events {
		switch event.EventType {
		case types.AgentRunEventTypeRunWaitingInput:
			waitingSequence = event.Sequence
		case types.AgentRunEventTypeRunResumed:
			resumedSequence = event.Sequence
		case types.AgentRunEventTypeToolCallEnd:
			require.NotEqual(t, "ask_user", event.Payload["toolCallName"])
		}
	}
	require.NotZero(t, waitingSequence)
	require.NotZero(t, resumedSequence)
	require.Less(t, waitingSequence, resumedSequence)
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
		Description:      "幼儿园亲子活动策划专家，负责活动方案、流程和安全预案",
		Domain:           "kindergarten_activity_planner",
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

func TestExpertCardPlainTextRemovesMarkdown(t *testing.T) {
	value := "### 活动概述\n- **活动名称**：秋日童行\n- 下一步：确认场地"
	cleaned := expertCardPlainText(value)
	require.NotContains(t, cleaned, "**")
	require.NotContains(t, cleaned, "###")
	require.NotContains(t, cleaned, "|")
	require.Contains(t, cleaned, "活动名称")
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

func TestExpertClarificationDefaultsToOneMacroRound(t *testing.T) {
	require.Equal(t, 1, expertClarificationMaxRounds(types.JSONMap{}))
	require.Equal(t, 4, expertClarificationMaxQuestions(types.JSONMap{}))
	require.Equal(t, 0, expertClarificationMaxRounds(types.JSONMap{
		"clarification_policy": types.JSONMap{"max_rounds": 0},
	}))
	require.Equal(t, 3, expertClarificationMaxRounds(types.JSONMap{
		"clarification_policy": types.JSONMap{"max_rounds": 9},
	}))
	require.Equal(t, 5, expertClarificationMaxQuestions(types.JSONMap{
		"clarification_policy": types.JSONMap{"max_questions": 9},
	}))
}

func TestNormalizeExpertIntakeQuestionsUsesConfiguredLimit(t *testing.T) {
	questions := []types.ExpertIntakeQuestion{
		{ID: "one", Label: "问题一", Type: "text", Required: true},
		{ID: "two", Label: "问题二", Type: "text", Required: true},
		{ID: "three", Label: "问题三", Type: "text", Required: true},
	}
	normalized := normalizeExpertIntakeQuestionsLimit(questions, 2)
	require.Len(t, normalized, 2)
	require.Equal(t, "one", normalized[0].ID)
	require.Equal(t, "two", normalized[1].ID)
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

func TestNormalizeExpertQualityRejectsInvalidDatesBudgetAndMissingEvidence(t *testing.T) {
	quality := expertQualityAssessment{
		Score:      96,
		Passed:     true,
		Summary:    "模型认为报告质量较高。",
		RedLines:   []string{},
		Dimensions: map[string]int{},
		Issues:     []expertQualityIssue{},
	}
	config := types.JSONMap{
		"quality_rubric": types.JSONMap{
			"minimum_score":     85,
			"require_citations": true,
		},
	}
	report := strings.Repeat("这是用于满足报告长度要求的执行说明。", 100) + `

## 时间安排

活动日期为 2026-02-30。

## 预算明细

| 项目 | 金额 |
| --- | ---: |
| 场地 | 100 |
| 物料 | 200 |
| 合计 | 250 |
`
	normalizeExpertQuality(&quality, config, expertExecutionPlan{}, report)

	require.False(t, quality.Passed)
	require.True(t, hasExpertQualityIssue(quality.Issues, "invalid_date"))
	require.True(t, hasExpertQualityIssue(quality.Issues, "budget_total_mismatch"))
	require.True(t, hasExpertQualityIssue(quality.Issues, "evidence_missing"))
}

func TestGetAgentRunRecoversTimedOutRunningRun(t *testing.T) {
	ctx := context.Background()
	runService, runRepo, _, _ := newAgentRunTestService(t)
	run := &types.AgentRun{
		TenantID:     9,
		UserID:       "user-a",
		RunType:      types.AgentRunTypeExpertAgentTest,
		AgentRef:     "expert-a",
		AgentVersion: "1.0.0",
		Status:       types.AgentRunStatusQueued,
	}
	require.NoError(t, runRepo.Create(ctx, run))
	require.NoError(t, runRepo.CreateInputRevision(ctx, &types.AgentRunInputRevision{
		RunID:  run.ID,
		Source: "request",
		Input:  types.JSONMap{"prompt": "测试超时恢复"},
	}))
	claimed, err := runRepo.Claim(ctx, run.ID, time.Now().UTC().Add(-agentRunTimeout-time.Minute))
	require.NoError(t, err)
	require.True(t, claimed)

	recovered, err := runService.GetAgentRun(ctx, run.TenantID, run.UserID, run.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusFailed, recovered.Status)
	require.Equal(t, types.AgentRunErrorTimedOut, recovered.ErrorCode)
	require.NotNil(t, recovered.FinishedAt)

	events, err := runRepo.ListEvents(ctx, run.ID, 0, 10)
	require.NoError(t, err)
	var failureEvent *types.AgentRunEvent
	for _, event := range events {
		if event.EventType == types.AgentRunEventTypeRunError {
			failureEvent = event
			break
		}
	}
	require.NotNil(t, failureEvent)
	require.Equal(t, types.AgentRunErrorTimedOut, failureEvent.Payload["errorCode"])
	require.Equal(t, true, failureEvent.Payload["retryable"])
	require.Equal(t, false, failureEvent.Payload["automaticRetry"])
	require.Equal(t, true, failureEvent.Payload["terminal"])
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
	require.Equal(t, parent.ThreadID, child.ThreadID)
	require.Equal(t, "regenerate", child.TriggerType)
	require.Equal(t, types.AgentRunStatusQueued, child.Status)
	require.Equal(t, "重点补充安全预案", child.Input["feedback"])
	threadRuns, err := runService.ListAgentThreadRuns(ctx, tenantID, userID, child.ID)
	require.NoError(t, err)
	require.Len(t, threadRuns, 2)
	require.Equal(t, parent.ID, threadRuns[0].ID)
	require.Equal(t, child.ID, threadRuns[1].ID)
}

func TestExpertRegenerationUsesParentReportAndSkipsClarification(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 12
	const userID = "member-user"
	finalResult := `{
		"schema_version":"agent_result_v1",
		"decision":{"should_create_card":true,"confidence":0.92,"reason":"已按纠偏意见生成行动清单。"},
		"card":{
			"schema_version":"service_card_v1",
			"title":"亲子活动行动清单",
			"summary":"在上一版活动方案基础上补充负责人、时间节点和完成标准。",
			"next_action":"确认各环节负责人并按清单推进。"
		},
		"artifacts":[{
			"kind":"report",
			"role":"primary",
			"title":"亲子活动行动清单",
			"format":"structured_report_v1",
			"content":{
				"format":"structured_report_v1",
				"title":"亲子活动行动清单",
				"executive_summary":"已将原方案整理为可执行清单。",
				"sections":[{"type":"recommended_actions","title":"行动清单","content":"完整行动清单见报告正文。"}],
				"evidence_refs":[]
			}
		}],
		"evidence":[]
	}`
	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(
		t,
		[]string{
			expertTestPlanJSON(),
			expertTestQualityPassedJSON(),
			finalResult,
		},
		[]string{expertTestLongReport()},
	)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	parent, err := runService.EnqueueExpertTest(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "制定亲子活动方案",
		ModelID:      "chat-1",
		Answers: types.JSONMap{
			"scale": "30组家庭",
		},
	})
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 1)
	claimed, err := runRepo.Claim(ctx, parent.ID, time.Now())
	require.NoError(t, err)
	require.True(t, claimed)
	parentReport := "# 上一版方案\n\n已确认30组家庭参加，方案包含流程、分工、预算和安全安排。"
	succeeded, err := runRepo.MarkSucceeded(ctx, parent.ID, types.JSONMap{
		"report_markdown": parentReport,
		"artifacts": []types.AgentArtifactResultV1{
			{
				ID:        "artifact-parent",
				VersionID: "artifact-parent-v1",
				Version:   1,
				Kind:      types.AgentArtifactKindReport,
				Role:      types.AgentArtifactRolePrimary,
				Title:     "上一版方案",
			},
		},
	}, time.Now())
	require.NoError(t, err)
	require.True(t, succeeded)

	child, err := runService.RegenerateAgentRun(ctx, tenantID, userID, parent.ID, types.AgentRunRegenerateInput{
		Feedback: "整理成行动清单，补充负责人、时间节点和完成标准。",
	})
	require.NoError(t, err)
	require.Len(t, enqueuer.tasks, 2)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[1]))

	completed, err := runRepo.GetByID(ctx, child.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusSucceeded, completed.Status, completed.ErrorMessage)
	require.Equal(t, parent.ID, completed.ParentRunID)
	persistedParent, err := runRepo.GetByID(ctx, parent.ID)
	require.NoError(t, err)
	require.Equal(t, parentReport, persistedParent.Result["report_markdown"])
	require.Len(t, chatModel.chatCalls, 2)
	require.Len(t, chatModel.streamCalls, 1)
	require.Contains(t, chatModel.chatCalls[0][1].Content, "上一版报告")
	require.Contains(t, chatModel.chatCalls[0][1].Content, parentReport)
	require.Contains(t, chatModel.chatCalls[0][1].Content, "整理成行动清单")
	require.Contains(t, chatModel.streamCalls[0][1].Content, "作为修订基线")
	require.Contains(t, chatModel.streamCalls[0][1].Content, parentReport)
	childResult := decodeAgentRunResult(t, completed.Result)
	require.NotEmpty(t, childResult.Artifacts)
	require.Equal(t, "artifact-parent", childResult.Artifacts[0].ID)
	require.NotEqual(t, "artifact-parent-v1", childResult.Artifacts[0].VersionID)
	require.Equal(t, 2, childResult.Artifacts[0].Version)
	require.Equal(t, parent.ID, childResult.Artifacts[0].Metadata["parent_run_id"])
	require.Equal(t, "artifact-parent-v1", childResult.Artifacts[0].Metadata["parent_version_id"])

	steps, err := runService.ListAgentRunSteps(ctx, tenantID, userID, child.ID)
	require.NoError(t, err)
	require.Len(t, steps, 5)
	require.Equal(t, types.AgentRunStepTypeIntake, steps[0].StepType)
	require.Equal(t, true, steps[0].Output["ready"])
	require.Empty(t, steps[0].Output["questions"])

	var snapshot types.AgentRequirementSnapshot
	require.NoError(t, db.First(&snapshot, "id = ?", completed.RequirementSnapshotID).Error)
	require.Equal(t, parent.ID, snapshot.Values["parent_run_id"])
	require.Equal(t, "整理成行动清单，补充负责人、时间节点和完成标准。", snapshot.Values["feedback"])
}

func TestPublishedExpertFollowUpReturnsChatAnswerWithoutArtifact(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 13
	const userID = "member-user"
	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(
		t,
		[]string{"上一版方案优先安排签到和分组，是因为这两个环节决定后续流程能否稳定启动。"},
		nil,
	)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	parent, err := runService.EnqueuePublishedExpertRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		PackageID:    pkg.ID,
		DefinitionID: definition.ID,
		Prompt:       "制定亲子活动方案",
		ModelID:      "chat-1",
	})
	require.NoError(t, err)
	claimed, err := runRepo.Claim(ctx, parent.ID, time.Now())
	require.NoError(t, err)
	require.True(t, claimed)
	parentReport := "# 亲子活动方案\n\n先签到，再按年龄分组，随后进入分区活动。"
	succeeded, err := runRepo.MarkSucceeded(ctx, parent.ID, types.JSONMap{
		"report_markdown": parentReport,
	}, time.Now())
	require.NoError(t, err)
	require.True(t, succeeded)

	followUp, err := runService.EnqueuePublishedExpertFollowUp(ctx, tenantID, userID, types.ExpertFollowUpInput{
		ParentRunID: parent.ID,
		Prompt:      "为什么这样安排？",
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunTypeExpertFollowUp, followUp.RunType)
	require.Equal(t, parent.ID, followUp.ParentRunID)
	require.Len(t, enqueuer.tasks, 2)
	require.NoError(t, runService.ProcessAgentRun(ctx, enqueuer.tasks[1]))

	completed, err := runRepo.GetByID(ctx, followUp.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusSucceeded, completed.Status, completed.ErrorMessage)
	require.Equal(t, parent.ID, completed.Result["parent_run_id"])
	require.Equal(t, "explanation", completed.Result["follow_up_mode"])
	require.Contains(t, completed.Result["message"], "签到和分组")
	result := decodeAgentRunResult(t, completed.Result)
	require.False(t, result.Decision.ShouldCreateCard)
	require.Empty(t, result.Artifacts)
	require.Empty(t, result.Evidence)
	require.True(t, decodeAgentRunValidation(t, completed.Result).Valid)
	require.Len(t, chatModel.chatCalls, 1)
	require.Contains(t, chatModel.chatCalls[0][1].Content, parentReport)
	require.Contains(t, chatModel.chatCalls[0][1].Content, "为什么这样安排")
}
