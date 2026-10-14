package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestServiceSpaceCreateRejectsInvalidSpaceType(t *testing.T) {
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	_, err := svc.Create(context.Background(), 101, "owner", types.ServiceSpaceCreateInput{
		Name:      "非法空间类型测试",
		SpaceType: "legacy",
	})
	require.ErrorIs(t, err, ErrServiceSpaceInvalidType)
}

func TestServiceSpaceCreateDefaultsCustomerServiceType(t *testing.T) {
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(context.Background(), 102, "owner", types.ServiceSpaceCreateInput{
		Name: "默认客户服务空间",
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceTypeCustomerService, space.SpaceType)
	require.Equal(t, types.ServiceSpaceStateDraft, space.State)
	require.NotNil(t, space.StateChangedAt)
}

func TestServiceSpaceCreateActiveDefaultsBuiltinExpert(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 105, "owner", types.ServiceSpaceCreateInput{
		Name:     "名称直建服务",
		Activate: true,
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceStateActive, space.State)

	experts, err := svc.ListExperts(ctx, 105, "owner", space.ID)
	require.NoError(t, err)
	require.Len(t, experts, 1)
	require.Equal(t, types.BuiltinSmartReasoningID, experts[0].ExpertRef)
	require.Equal(t, "服务主理人", experts[0].ExpertName)
	require.True(t, experts[0].Enabled)

	session, err := svc.CreateSession(ctx, 105, "owner", space.ID, types.ServiceSessionCreateInput{})
	require.NoError(t, err)
	require.Equal(t, types.BuiltinSmartReasoningID, session.ExpertRef)
}

func TestServiceSpaceNormalizesLegacySessionPrincipalOnRead(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	repo := repository.NewServiceSpaceRepository(db)
	svc := NewServiceSpaceService(repo, repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 106, "owner", types.ServiceSpaceCreateInput{
		Name:     "历史会话主理人归一化",
		Activate: true,
	})
	require.NoError(t, err)
	session, err := svc.CreateSession(ctx, 106, "owner", space.ID, types.ServiceSessionCreateInput{
		Title: "旧会话",
	})
	require.NoError(t, err)
	require.NoError(t, db.Model(&types.Session{}).
		Where("id = ?", session.ID).
		Updates(map[string]any{
			"expert_ref":  "legacy-member-agent",
			"expert_name": "旧专家",
		}).Error)

	got, err := svc.GetSession(ctx, 106, "owner", space.ID, session.ID)
	require.NoError(t, err)
	require.Equal(t, types.BuiltinSmartReasoningID, got.ExpertRef)
	require.Equal(t, "服务主理人", got.ExpertName)

	scoped, err := svc.GetSessionScope(ctx, 106, session.ID)
	require.NoError(t, err)
	require.Equal(t, types.BuiltinSmartReasoningID, scoped.ExpertRef)

	items, total, err := svc.ListSessions(ctx, 106, "owner", space.ID, "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Equal(t, types.BuiltinSmartReasoningID, items[0].ExpertRef)

	overview, err := svc.GetOverview(ctx, 106, "owner", space.ID)
	require.NoError(t, err)
	require.NotEmpty(t, overview.RecentSessions)
	require.Equal(t, types.BuiltinSmartReasoningID, overview.RecentSessions[0].ExpertRef)
}

func TestServiceSpaceCreateActiveAppliesLatestPublishedTenantTemplate(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	serviceRepo := repository.NewServiceSpaceRepository(db)
	svc := NewServiceSpaceService(serviceRepo, repository.NewOrganizeRepository(db), nil, nil)

	template, err := svc.CreateAdminTemplate(ctx, 107, "owner", types.ServiceSpaceTemplateAdminInput{
		Name:        "学员档案服务",
		SpaceType:   types.ServiceSpaceTypeCustomerService,
		Instruction: "服务家长与学员，重点记录孩子姓名、年龄、班级和主班老师。",
		SubjectPolicy: types.ServiceSubjectPolicy{
			Required: false,
		},
	})
	require.NoError(t, err)
	_, err = svc.PublishAdminTemplate(ctx, 107, "owner", template.Key)
	require.NoError(t, err)

	space, err := svc.Create(ctx, 107, "owner", types.ServiceSpaceCreateInput{
		Name:     "乐乐",
		Activate: true,
	})
	require.NoError(t, err)
	require.Equal(t, template.Key, space.TemplateKey)
	require.Equal(t, "服务家长与学员，重点记录孩子姓名、年龄、班级和主班老师。", space.Instruction)
	require.Equal(t, types.ServiceSpaceStateActive, space.State)

	record, err := serviceRepo.GetLatestBlueprint(ctx, 107, space.ID)
	require.NoError(t, err)
	require.NotNil(t, record)
	blueprint, err := record.ToBlueprint()
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceBlueprintSourceTemplate, blueprint.SourceType)
	require.Equal(t, types.ServiceSpaceBlueprintConfirmationAutoApply, blueprint.ConfirmationMode)
	require.False(t, blueprint.SubjectPolicy.Required)
	require.Equal(t, []string{"孩子姓名", "年龄", "班级", "主班老师"}, []string{
		blueprint.ProfileSchema[0].Label,
		blueprint.ProfileSchema[1].Label,
		blueprint.ProfileSchema[2].Label,
		blueprint.ProfileSchema[3].Label,
	})

	profile, err := svc.GetProfile(ctx, 107, "owner", space.ID)
	require.NoError(t, err)
	require.Equal(t, blueprint.ProfileSchema, profile.Schema)
}

func TestDefaultPublishedServiceTemplatePrefersMostRecentlyPublished(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	serviceRepo := repository.NewServiceSpaceRepository(db)
	svc := NewServiceSpaceService(serviceRepo, repository.NewOrganizeRepository(db), nil, nil)
	application := svc.(*serviceSpaceService)
	now := time.Now().UTC()

	for index, publishedAt := range []time.Time{now.Add(-time.Hour), now} {
		template, record, err := buildAdminServiceTemplate(108, types.ServiceSpaceTemplateAdminInput{
			Key:         []string{"older", "newer"}[index],
			Name:        []string{"较早模板", "最新模板"}[index],
			SpaceType:   types.ServiceSpaceTypeCustomerService,
			Instruction: "重点记录年龄和班级。",
		}, 1)
		require.NoError(t, err)
		record.Status = string(types.ServiceSpaceTemplateStatusPublished)
		record.PublishedAt = &publishedAt
		record.UpdatedAt = publishedAt
		require.NoError(t, serviceRepo.CreateTemplate(ctx, record))
		require.NotNil(t, template)
	}

	selected, err := application.defaultPublishedServiceTemplate(ctx, 108)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, "newer", selected.Key)
}

func TestServiceSpaceCreateActiveKeepsPrincipalWhenMemberAgentIsDisabled(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)
	disabled := false

	space, err := svc.Create(ctx, 106, "owner", types.ServiceSpaceCreateInput{
		Name: "禁用专家服务",
		Experts: []types.ServiceExpertBindingInput{{
			ExpertRef: "disabled-expert",
			Enabled:   &disabled,
		}},
		Activate: true,
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceStateActive, space.State)

	experts, err := svc.ListExperts(ctx, 106, "owner", space.ID)
	require.NoError(t, err)
	require.Len(t, experts, 2)
	require.Equal(t, types.BuiltinSmartReasoningID, experts[0].ExpertRef)
	require.True(t, experts[0].Enabled)
	require.Equal(t, "disabled-expert", experts[1].ExpertRef)
	require.False(t, experts[1].Enabled)
}

func TestServiceSpaceCreateInfersAndPersistsInstructionConfiguration(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	serviceRepo := repository.NewServiceSpaceRepository(db)
	svc := NewServiceSpaceService(serviceRepo, repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 103, "owner", types.ServiceSpaceCreateInput{
		Name:        "竞品调研",
		Instruction: "围绕竞品课程开展调研，记录研究问题、关键发现、事实依据和下一步验证",
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceTypeResearch, space.SpaceType)

	record, err := serviceRepo.GetLatestBlueprint(ctx, 103, space.ID)
	require.NoError(t, err)
	require.NotNil(t, record)
	blueprint, err := record.ToBlueprint()
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceBlueprintSourceInstruction, blueprint.SourceType)
	require.Equal(t, types.ServiceSpaceBlueprintStatusConfirmed, blueprint.Status)
	require.Equal(t, types.ServiceSpaceBlueprintConfirmationInstructionAutoApply, blueprint.ConfirmationMode)
	require.Equal(t, "research_question", blueprint.ProfileSchema[0].Key)

	profile, err := svc.GetProfile(ctx, 103, "owner", space.ID)
	require.NoError(t, err)
	require.Equal(t, blueprint.ProfileSchema, profile.Schema)
	summary, err := svc.GetSummary(ctx, 103, "owner", space.ID)
	require.NoError(t, err)
	require.Equal(t, blueprint.SummarySchema, summary.Schema)
}

func TestServiceSpaceCreateKeepsExplicitTypeForLegacyClients(t *testing.T) {
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(context.Background(), 104, "owner", types.ServiceSpaceCreateInput{
		Name:        "显式类型兼容",
		SpaceType:   string(types.ServiceSpaceTypeOperations),
		Instruction: "持续跟进家长需求和下一步动作",
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceTypeOperations, space.SpaceType)
}

func TestBuildAdminServiceTemplateDerivesProfileSchemaFromInstruction(t *testing.T) {
	template, _, err := buildAdminServiceTemplate(1, types.ServiceSpaceTemplateAdminInput{
		Name:        "招生跟进服务",
		SpaceType:   types.ServiceSpaceTypeCustomerService,
		Instruction: "跟进家长到访与报名，重点记录家长顾虑与到访意向。",
		SubjectPolicy: types.ServiceSubjectPolicy{
			Required:     true,
			AllowedTypes: []string{"member_family"},
		},
	}, 1)
	require.NoError(t, err)
	require.Len(t, template.Blueprint.ProfileSchema, 2)
	require.Equal(t, "家长顾虑", template.Blueprint.ProfileSchema[0].Label)
	require.Equal(t, "到访意向", template.Blueprint.ProfileSchema[1].Label)
	require.Equal(t, "facts", template.Blueprint.ProfileSchema[0].Source)
	require.NotEmpty(t, template.Blueprint.ProfileSchema[0].Key)
	require.NotEmpty(t, template.Blueprint.SummarySchema)
}

func TestBuildAdminServiceTemplateKeepsSubjectPolicyOptional(t *testing.T) {
	template, _, err := buildAdminServiceTemplate(1, types.ServiceSpaceTemplateAdminInput{
		Name:        "单客户服务",
		SpaceType:   types.ServiceSpaceTypeCustomerService,
		Instruction: "重点记录孩子姓名、年龄和班级。",
		SubjectPolicy: types.ServiceSubjectPolicy{
			Required:       false,
			AllowedTypes:   []string{},
			AllowHierarchy: false,
		},
	}, 1)
	require.NoError(t, err)
	require.False(t, template.Blueprint.SubjectPolicy.Required)
	require.Empty(t, template.Blueprint.SubjectPolicy.AllowedTypes)
	require.False(t, template.Blueprint.SubjectPolicy.AllowHierarchy)
}
