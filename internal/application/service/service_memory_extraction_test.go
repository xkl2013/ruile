package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

type staticChatModel struct {
	response string
}

func (m *staticChatModel) Chat(context.Context, []chat.Message, *chat.ChatOptions) (*types.ChatResponse, error) {
	return &types.ChatResponse{Content: m.response}, nil
}

func (m *staticChatModel) ChatStream(context.Context, []chat.Message, *chat.ChatOptions) (<-chan types.StreamResponse, error) {
	ch := make(chan types.StreamResponse)
	close(ch)
	return ch, nil
}

func (m *staticChatModel) GetModelName() string { return "static" }
func (m *staticChatModel) GetModelID() string   { return "static" }

type autoInferenceModelService struct {
	stubModelService
	models []*types.Model
}

func (s *autoInferenceModelService) ListModels(context.Context) ([]*types.Model, error) {
	return s.models, nil
}

func TestServiceExtractMemoryRecognizesHtmlEditorNote(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceService(serviceRepo, organizeRepo)

	const tenantID uint64 = 7
	const userID = "user-a"
	profile, err := svc.CreateWorkProfile(ctx, tenantID, userID, types.ServiceWorkProfileInput{
		Name:           "招生顾问服务助理",
		RoleType:       "consultant",
		MemoryScope:    "本人记忆 · 试听咨询相关",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	})
	require.NoError(t, err)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profile.ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain:      types.ServiceAgentDomainSalesConsulting,
				Enabled:          true,
				DisplayName:      "招生咨询",
				WorkDocDirectory: "线索/",
			},
		},
	})
	require.NoError(t, err)

	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "试听课",
		Content:  `<p style="line-height: 1.5;">小明妈妈礼拜三给孩子安排了试听课,到时候直接过来就行了</p>`,
		Source:   "手动输入",
		Metadata: types.JSONMap{"tags": []string{}},
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.True(t, extracted.Generated)
	require.NotNil(t, extracted.Reminder)
	require.Equal(t, types.ServiceAgentDomainSalesConsulting, extracted.Reminder.AgentDomain)
	require.Equal(t, "小明妈妈", extracted.Reminder.Metadata["customer_name"])
	require.Equal(t, "售前试听", extracted.Reminder.Stage)
	require.Equal(t, "完成试听后回访并确认下一步安排", extracted.Reminder.NextAction)
	require.Len(t, extracted.Reminder.MemoryEvidence, 1)
	require.Contains(t, extracted.Reminder.MemoryEvidence[0].Summary, "小明妈妈礼拜三")
}

func TestServiceExtractMemoryRecognizesAudioTranscriptMetadata(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceService(serviceRepo, organizeRepo)

	const tenantID uint64 = 7
	const userID = "user-a"
	profile, err := svc.CreateWorkProfile(ctx, tenantID, userID, types.ServiceWorkProfileInput{
		Name:           "招生顾问服务助理",
		RoleType:       "consultant",
		MemoryScope:    "本人记忆 · 试听咨询相关",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	})
	require.NoError(t, err)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profile.ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain:      types.ServiceAgentDomainSalesConsulting,
				Enabled:          true,
				DisplayName:      "招生咨询",
				WorkDocDirectory: "线索/",
			},
		},
	})
	require.NoError(t, err)

	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindAudio,
		Title:    "试听电话",
		Content:  "<p>录音已保存，等待转写。</p>",
		Source:   "语音记录",
		Metadata: types.JSONMap{
			"transcription_status": "completed",
			"transcript":           "小明妈妈电话里咨询体验课安排，也问了报名政策和后续跟进节奏。",
		},
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	if !extracted.Generated {
		t.Fatalf("expected service reminder to be generated, reason=%s", extracted.Reason)
	}
	require.NotNil(t, extracted.Reminder)
	require.Equal(t, types.ServiceAgentDomainSalesConsulting, extracted.Reminder.AgentDomain)
	require.Equal(t, "小明妈妈", extracted.Reminder.Metadata["customer_name"])
	require.Equal(t, "售前试听", extracted.Reminder.Stage)
	require.Contains(t, extracted.Reminder.Summary, "小明妈妈电话里咨询体验课")
	require.Contains(t, extracted.Reminder.MemoryEvidence[0].Summary, "小明妈妈电话里咨询体验课")
}

func TestServiceListWorkProfilesMaterializesMemberDescription(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembers(serviceRepo, organizeRepo, memberRepo)

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "负责试听邀约和家长跟进，沟通风格专业温和。",
	}))

	profiles, err := svc.ListWorkProfiles(ctx, tenantID, "")
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	require.Equal(t, userID, profiles[0].UserID)
	require.Equal(t, "consultant", profiles[0].RoleType)
	require.True(t, profiles[0].DefaultProfile)
	require.True(t, profiles[0].Enabled)
	require.Equal(t, types.ServiceWorkProfileStateEnabled, profiles[0].State)

	profiles, err = svc.ListWorkProfiles(ctx, tenantID, "")
	require.NoError(t, err)
	require.Len(t, profiles, 1)
}

func TestServiceBootstrapHydratesUpdatedMemberDescription(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembers(serviceRepo, organizeRepo, memberRepo)
	memberService := NewTenantMemberService(memberRepo, nil, nil)

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "负责试听邀约和家长跟进。",
	}))

	bootstrap, err := svc.GetBootstrap(ctx, tenantID, userID)
	require.NoError(t, err)
	require.NotNil(t, bootstrap.Profile)
	require.Equal(t, "负责试听邀约和家长跟进。", bootstrap.Profile.WorkProfileDescription)

	require.NoError(t, memberService.UpdateWorkProfileDescription(
		ctx,
		userID,
		tenantID,
		"负责续费沟通和课后服务，沟通风格专业温和。",
	))

	bootstrap, err = svc.GetBootstrap(ctx, tenantID, userID)
	require.NoError(t, err)
	require.NotNil(t, bootstrap.Profile)
	require.Equal(t, "负责续费沟通和课后服务，沟通风格专业温和。", bootstrap.Profile.WorkProfileDescription)
}

func TestServiceExtractMemoryAutoEnablesDefaultAgentsAfterProfileMaterialized(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembersAndModel(serviceRepo, organizeRepo, memberRepo, &autoInferenceModelService{
		stubModelService: stubModelService{
			chatModel: &staticChatModel{
				response: `{"enabled_domains":["sales_consulting","customer_service","schedule_coordination"],"reason":"岗位描述以试听邀约、家长跟进和排课协调为主"}`,
			},
		},
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
		},
	})

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "负责试听邀约和家长跟进。",
	}))
	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "试听课",
		Content:  `<p>小明妈妈礼拜三给孩子安排了试听课</p>`,
		Source:   "手动输入",
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.True(t, extracted.Generated)
	require.NotNil(t, extracted.Reminder)
	require.Equal(t, types.ServiceAgentDomainSalesConsulting, extracted.Reminder.AgentDomain)
	require.NotEqual(t, "agent_not_enabled", extracted.Reason)

	settings, err := svc.ListAgentSettings(ctx, tenantID, extracted.Reminder.ProfileID, true)
	require.NoError(t, err)
	domains := make([]string, 0, len(settings))
	for _, setting := range settings {
		domains = append(domains, setting.AgentDomain)
	}
	require.ElementsMatch(t, []string{
		types.ServiceAgentDomainSalesConsulting,
		types.ServiceAgentDomainCustomerService,
		types.ServiceAgentDomainScheduling,
		types.ServiceAgentDomainMemoryRouter,
	}, domains)
}

func TestServiceExtractMemoryAutoRebuildsDisabledAgentSettings(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembers(serviceRepo, organizeRepo, memberRepo)

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "负责试听邀约和家长跟进。",
	}))
	profiles, err := svc.ListWorkProfiles(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profiles[0].ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain: types.ServiceAgentDomainSalesConsulting,
				Enabled:     false,
			},
		},
	})
	require.NoError(t, err)

	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "试听课",
		Content:  `<p>小明妈妈礼拜三给孩子安排了试听课</p>`,
		Source:   "手动输入",
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.True(t, extracted.Generated)
	require.NotEqual(t, "agent_not_enabled", extracted.Reason)

	settings, err := svc.ListAgentSettings(ctx, tenantID, extracted.Reminder.ProfileID, true)
	require.NoError(t, err)
	domains := make([]string, 0, len(settings))
	for _, setting := range settings {
		domains = append(domains, setting.AgentDomain)
	}
	require.ElementsMatch(t, []string{
		types.ServiceAgentDomainMemoryRouter,
		types.ServiceAgentDomainLeadIntake,
		types.ServiceAgentDomainSalesConsulting,
		types.ServiceAgentDomainCustomerService,
	}, domains)
}

func TestServiceExtractMemoryGeneratesModeWhenConfiguredAbilitiesDoNotMatch(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembersAndModel(serviceRepo, organizeRepo, memberRepo, &autoInferenceModelService{
		stubModelService: stubModelService{
			chatModel: &staticChatModel{
				response: `{"should_generate":true,"agent_domain":"customer_service","service_mode":"内部培训准备","subject_name":"就业课培训","customer_name":"就业课培训","student_name":"待补充","title":"就业课培训准备事项","summary":"需要梳理就业课培训前的资料、人员和时间安排。","stage":"培训准备","priority":"medium","due_text":"本周","risk_label":"待判断","assist_reason":"记忆包含可转服务的培训准备事项，但现有能力没有专门配置内部培训。","primary_action":"先整理培训清单，再确认负责人、讲师和交付时间。","next_action":"周五前确认讲师、签到表和设备检查清单。","avoid_action":"不要把未确认的培训安排当成已完成事项。","reply_draft":"我先把就业课培训准备事项整理成清单，并在周五前确认讲师、签到表和设备检查。","memory_signals":["培训准备","清单确认"],"sales_highlights":["该记忆适合生成内部培训服务事项。"],"reason":"未命中现有配置能力，自动生成内部培训服务模式"}`,
			},
		},
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
		},
	})

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "我是睿乐园园长，主要服务园区家长、幼儿及教职员工的协同事项。",
	}))
	profiles, err := svc.ListWorkProfiles(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profiles[0].ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain: types.ServiceAgentDomainMemoryRouter,
				Enabled:     true,
			},
		},
	})
	require.NoError(t, err)

	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "就业课培训准备事项",
		Content:  `<p>整理就业课培训大纲、签到表和设备检查，周五前确认讲师。</p>`,
		Source:   "手动输入",
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.True(t, extracted.Generated)
	require.Equal(t, "generated", extracted.Reason)
	require.NotNil(t, extracted.Reminder)
	require.Equal(t, types.ServiceAgentDomainCustomerService, extracted.Reminder.AgentDomain)
	require.Equal(t, "内部培训准备", extracted.Reminder.Metadata["service_mode"])
	require.Equal(t, "就业课培训", extracted.Reminder.Metadata["subject_name"])
	require.Empty(t, extracted.Reminder.Metadata["customer_name"])
	require.Equal(t, "培训准备", extracted.Reminder.Stage)
	require.Equal(t, "周五前确认讲师、签到表和设备检查清单。", extracted.Reminder.NextAction)
	require.Contains(t, extracted.Reminder.PrimaryAction, "整理培训清单")
	require.Contains(t, extracted.Reminder.ReplyDraft, "就业课培训准备事项")
}

func TestServiceExtractMemorySanitizesCustomerHallucinationForInvestmentDocument(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(7))
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	memberRepo := repository.NewTenantMemberRepository(db)
	svc := NewServiceServiceWithMembersAndModel(serviceRepo, organizeRepo, memberRepo, &autoInferenceModelService{
		stubModelService: stubModelService{
			chatModel: &staticChatModel{
				response: `{"should_generate":true,"agent_domain":"sales_consulting","service_mode":"售前试听","subject_name":"小明妈妈","customer_name":"小明妈妈","student_name":"小明","title":"客户摘要","summary":"小明妈妈礼拜三给孩子安排了试听课，到时候直接过来就行了。","stage":"售前试听","priority":"medium","risk_label":"价格顾虑","assist_reason":"客户有试听安排。","primary_action":"先确认客户状态，再生成可发送话术。","next_action":"完成试听后回访并确认下一步安排。","avoid_action":"不要遗漏家长关注点。","reply_draft":"您好，我根据最近记录把孩子试听情况整理了一下。","sales_highlights":["出现售前接触信号。"],"reason":"模型误判为客户服务"}`,
			},
		},
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
		},
	})

	const tenantID uint64 = 7
	const userID = "user-a"
	require.NoError(t, memberRepo.Create(ctx, &types.TenantMember{
		TenantID:               tenantID,
		UserID:                 userID,
		Role:                   types.TenantRoleContributor,
		Status:                 types.TenantMemberStatusActive,
		WorkProfileDescription: "负责整理投资研究资料和跟踪事项。",
	}))
	profiles, err := svc.ListWorkProfiles(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Len(t, profiles, 1)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profiles[0].ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain: types.ServiceAgentDomainMemoryRouter,
				Enabled:     true,
			},
		},
	})
	require.NoError(t, err)

	memory := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "英诺赛科GaN IDM龙头投资分析",
		Content:  `<p>英诺赛科（02577.HK）深度投资报告：GaN IDM龙头，关注收入增长、毛利率改善、估值消化和功率半导体需求风险。</p>`,
		Source:   "文件导入",
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	extracted, err := svc.ExtractMemory(ctx, tenantID, userID, memory.ID)
	require.NoError(t, err)
	require.True(t, extracted.Generated)
	require.Equal(t, "generated", extracted.Reason)
	require.NotNil(t, extracted.Reminder)
	require.Empty(t, extracted.Reminder.Metadata["customer_name"])
	require.Equal(t, "英诺赛科GaN IDM龙头投资分析", extracted.Reminder.Metadata["subject_name"])
	require.Equal(t, "投资分析", extracted.Reminder.Metadata["service_mode"])
	require.Equal(t, "英诺赛科GaN IDM龙头投资分析", extracted.Reminder.Title)
	require.Equal(t, "投资分析", extracted.Reminder.Stage)
	require.Equal(t, "投资风险待确认", extracted.Reminder.RiskLabel)
	require.Contains(t, extracted.Reminder.NextAction, "核心结论")
	require.Contains(t, extracted.Reminder.PrimaryAction, "投资判断")
	require.NotContains(t, extracted.Reminder.Summary, "小明妈妈")
	require.NotContains(t, extracted.Reminder.ReplyDraft, "孩子")
	require.Contains(t, extracted.Reminder.WriteBackDraft, "英诺赛科")

	docTitles := make([]string, 0, len(extracted.Reminder.WorkDocs))
	for _, doc := range extracted.Reminder.WorkDocs {
		docTitles = append(docTitles, doc.Title)
		require.NotContains(t, doc.DocPath, "待补充客户")
	}
	require.Contains(t, docTitles, "服务摘要")
	require.NotContains(t, docTitles, "客户摘要")
}
