package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

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
	response string
	messages []chat.Message
}

func (m *expertTestChatModel) Chat(
	_ context.Context,
	messages []chat.Message,
	_ *chat.ChatOptions,
) (*types.ChatResponse, error) {
	m.messages = append([]chat.Message(nil), messages...)
	return &types.ChatResponse{Content: m.response}, nil
}

func (m *expertTestChatModel) ChatStream(
	context.Context,
	[]chat.Message,
	*chat.ChatOptions,
) (<-chan types.StreamResponse, error) {
	return nil, errors.New("streaming is not supported by the expert test model")
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
	response string,
) (interfaces.AgentRunService, interfaces.AgentRunRepository, *agentRunTaskEnqueuer, *gorm.DB, *expertTestChatModel) {
	t.Helper()
	db := newServiceDailyReportTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&types.AgentRun{},
		&types.ExpertPackage{},
		&types.ExpertPackageVersion{},
		&types.AgentDefinitionVersion{},
	))
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	runRepo := repository.NewAgentRunRepository(db)
	expertRepo := repository.NewExpertPackageRepository(db)
	enqueuer := &agentRunTaskEnqueuer{}
	chatModel := &expertTestChatModel{response: response}
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
	require.Equal(t, types.AgentRunStatusSucceeded, run.Status)
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
	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(t, response)
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
		},
	}
	require.NoError(t, db.Create(definition).Error)

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
	require.Len(t, chatModel.messages, 2)
	require.Contains(t, chatModel.messages[0].Content, "幼儿园活动策划专家")
	require.Contains(t, chatModel.messages[0].Content, "睿乐执行结果协议")
}

func TestAgentRunExpertTestRejectsUnpublishedDefinition(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	runService, _, _, db, _ := newExpertAgentRunTestService(t, `{}`)
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
