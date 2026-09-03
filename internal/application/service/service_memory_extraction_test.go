package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

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

func TestServiceExtractMemoryReportsAgentNotEnabledAfterProfileMaterialized(t *testing.T) {
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
	require.False(t, extracted.Generated)
	require.Equal(t, "agent_not_enabled", extracted.Reason)
}
