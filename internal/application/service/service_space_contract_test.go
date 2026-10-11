package service

import (
	"context"
	"testing"

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
