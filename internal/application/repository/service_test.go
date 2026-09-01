package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newServiceRepositoryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.UserWorkProfile{},
		&types.WorkProfileAgentSetting{},
		&types.ServiceSubject{},
		&types.AgentWorkDoc{},
		&types.AgentWorkDocMemoryLink{},
		&types.ServiceReminder{},
		&types.AgentActionDraft{},
		&types.AgentActionLog{},
		&types.OrganizeMemory{},
	))
	return db
}

func TestServiceRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := newServiceRepositoryTestDB(t)
	repo := NewServiceRepository(db)

	const tenantID uint64 = 7
	const userID = "user-a"

	firstProfile := &types.UserWorkProfile{
		TenantID:       tenantID,
		UserID:         userID,
		Name:           "旧服务配置",
		MemoryScope:    "本人记忆",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	}
	require.NoError(t, repo.CreateWorkProfile(ctx, firstProfile))

	defaultProfile := &types.UserWorkProfile{
		TenantID:       tenantID,
		UserID:         userID,
		Name:           "招生顾问服务配置",
		RoleType:       "consultant",
		MemoryScope:    "本人记忆 · 服务相关",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	}
	require.NoError(t, repo.CreateWorkProfile(ctx, defaultProfile))

	gotDefault, err := repo.GetDefaultWorkProfile(ctx, tenantID, userID)
	require.NoError(t, err)
	require.NotNil(t, gotDefault)
	require.Equal(t, defaultProfile.ID, gotDefault.ID)

	profiles, err := repo.ListWorkProfiles(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Len(t, profiles, 2)
	require.False(t, profiles[1].DefaultProfile)

	settings := []*types.WorkProfileAgentSetting{
		{
			TenantID:         tenantID,
			ProfileID:        defaultProfile.ID,
			AgentID:          types.BuiltinServiceAssistantID,
			AgentDomain:      types.ServiceAgentDomainCustomerService,
			Enabled:          true,
			DisplayName:      "客户服务",
			DisplayOrder:     1,
			WorkDocDirectory: "客户/",
		},
		{
			TenantID:         tenantID,
			ProfileID:        defaultProfile.ID,
			AgentID:          types.BuiltinServiceAssistantID,
			AgentDomain:      types.ServiceAgentDomainLeadIntake,
			Enabled:          false,
			DisplayName:      "线索录入",
			DisplayOrder:     2,
			WorkDocDirectory: "线索/",
		},
	}
	require.NoError(t, repo.ReplaceAgentSettings(ctx, tenantID, defaultProfile.ID, settings))
	enabledSettings, err := repo.ListAgentSettings(ctx, tenantID, defaultProfile.ID, true)
	require.NoError(t, err)
	require.Len(t, enabledSettings, 1)
	require.Equal(t, types.ServiceAgentDomainCustomerService, enabledSettings[0].AgentDomain)

	subject := &types.ServiceSubject{
		TenantID:        tenantID,
		OwnerUserID:     userID,
		SubjectKey:      "chenyu|chenyu",
		DisplayName:     "陈屿妈妈",
		StudentName:     "陈屿",
		Aliases:         types.StringArray{"陈屿妈妈"},
		VisibilityScope: "private",
		Confidence:      0.86,
	}
	require.NoError(t, repo.UpsertSubject(ctx, subject))
	require.NotEmpty(t, subject.ID)

	memory := &types.OrganizeMemory{
		TenantID:   tenantID,
		UserID:     userID,
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "陈屿续费回访",
		Content:    "家长认可最近课堂反馈，需要先整理阶段成长记录。",
		Source:     "企微",
		OccurredAt: time.Now().UTC(),
		Metadata: types.JSONMap{
			"summary": "家长认可课堂反馈，进入续费服务窗口。",
		},
	}
	require.NoError(t, db.WithContext(ctx).Create(memory).Error)

	reminder := &types.ServiceReminder{
		ID:                "service-reminder-test",
		TenantID:          tenantID,
		UserID:            userID,
		ProfileID:         defaultProfile.ID,
		SubjectID:         subject.ID,
		AgentDomain:       types.ServiceAgentDomainCustomerService,
		Title:             "陈屿续费服务提醒",
		Summary:           "进入续费服务窗口，需要先补成长回顾。",
		Status:            types.ServiceReminderStatusPending,
		Priority:          types.ServicePriorityMedium,
		Stage:             "续费服务",
		NextAction:        "整理阶段成长回顾",
		SourceMemoryIDs:   types.StringArray{memory.ID},
		SourceMemoryCount: 1,
		Metadata:          types.JSONMap{"customer_name": "陈屿妈妈", "student_name": "陈屿"},
	}
	require.NoError(t, repo.UpsertReminder(ctx, reminder))

	doc := &types.AgentWorkDoc{
		TenantID:        tenantID,
		ProfileID:       defaultProfile.ID,
		SubjectID:       subject.ID,
		OwnerUserID:     userID,
		AgentDomain:     types.ServiceAgentDomainCustomerService,
		DocPath:         "客户/陈屿/客户摘要.md",
		Title:           "客户摘要",
		Content:         "# 客户摘要\n\n- 进入续费服务窗口",
		SourceMemoryIDs: types.StringArray{memory.ID},
	}
	require.NoError(t, repo.UpsertWorkDocWithLinks(ctx, doc, []*types.AgentWorkDocMemoryLink{
		{
			MemoryID:        memory.ID,
			SubjectID:       subject.ID,
			AgentDomain:     types.ServiceAgentDomainCustomerService,
			LinkType:        types.AgentWorkDocLinkTypeEvidence,
			EvidenceExcerpt: "进入续费服务窗口",
		},
	}))

	draft := &types.AgentActionDraft{
		TenantID:    tenantID,
		UserID:      userID,
		ReminderID:  reminder.ID,
		AgentID:     types.BuiltinServiceAssistantID,
		AgentDomain: types.ServiceAgentDomainCustomerService,
		ActionType:  "follow_up",
		Title:       "整理阶段成长回顾",
		Summary:     "先生成成长回顾，再进入续费沟通。",
	}
	require.NoError(t, repo.CreateActionDraft(ctx, draft))

	reminders, total, err := repo.ListReminders(ctx, types.ServiceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: defaultProfile.ID,
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, reminders, 1)
	require.Len(t, reminders[0].MemoryEvidence, 1)
	require.Len(t, reminders[0].WorkDocs, 1)
	require.Len(t, reminders[0].ActionDrafts, 1)
	require.Equal(t, "陈屿续费回访", reminders[0].MemoryEvidence[0].Title)
	require.Equal(t, "客户/陈屿/客户摘要.md", reminders[0].WorkDocs[0].DocPath)

	subjects, subjectTotal, err := repo.ListSubjects(ctx, types.ServiceCustomerSpaceListQuery{
		TenantID:  tenantID,
		UserID:    userID,
		ProfileID: defaultProfile.ID,
		Page:      1,
		PageSize:  10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), subjectTotal)
	require.Len(t, subjects, 1)
	require.Equal(t, subject.ID, subjects[0].ID)

	gotSubject, err := repo.GetSubject(ctx, tenantID, userID, subject.ID)
	require.NoError(t, err)
	require.NotNil(t, gotSubject)
	require.Equal(t, "陈屿妈妈", gotSubject.DisplayName)

	require.NoError(t, repo.UpdateReminderStatus(ctx, tenantID, userID, reminder.ID, types.ServiceReminderStatusCompleted))
	updatedReminder, err := repo.GetReminder(ctx, tenantID, userID, reminder.ID)
	require.NoError(t, err)
	require.Equal(t, types.ServiceReminderStatusCompleted, updatedReminder.Status)
	require.Equal(t, "已完成", updatedReminder.WriteBackStatus)

	require.NoError(t, repo.UpdateActionDraftStatus(ctx, tenantID, userID, draft.ID, types.AgentActionDraftStatusConfirmed))
	updatedDraft, err := repo.GetActionDraft(ctx, tenantID, userID, draft.ID)
	require.NoError(t, err)
	require.Equal(t, types.AgentActionDraftStatusConfirmed, updatedDraft.Status)

	stats, err := repo.CountRemindersByStatus(ctx, tenantID, userID, defaultProfile.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), stats[types.ServiceReminderStatusCompleted])
}
