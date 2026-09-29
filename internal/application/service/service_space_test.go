package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceSpaceLifecycleMembershipAndOwnership(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 7, "owner", types.ServiceSpaceCreateInput{
		Name:        "秋季招生咨询",
		Description: "招生咨询、家长跟进与报名材料整理",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "admission-expert", ExpertName: "招生咨询专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceStateActive, space.State)
	require.Equal(t, types.ServiceMemberRoleOwner, space.Role)
	require.True(t, space.IsDefault)

	_, err = svc.Get(ctx, 7, "outsider", space.ID)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)

	member, err := svc.AddMember(ctx, 7, "owner", space.ID, types.ServiceMemberInput{
		UserID: "editor",
		Role:   types.ServiceMemberRoleEditor,
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceMemberRoleEditor, member.Role)

	session, err := svc.CreateSession(ctx, 7, "editor", space.ID, types.ServiceSessionCreateInput{
		Title: "开放日到访名单跟进",
	})
	require.NoError(t, err)
	require.Equal(t, space.ID, session.ServiceID)
	require.Equal(t, "admission-expert", session.ExpertRef)

	_, err = svc.GetSession(ctx, 7, "outsider", space.ID, session.ID)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)

	paused, err := svc.SetState(ctx, 7, "owner", space.ID, types.ServiceSpaceStatePaused)
	require.NoError(t, err)
	require.Equal(t, types.ServiceSpaceStatePaused, paused.State)
	_, err = svc.CreateSession(ctx, 7, "editor", space.ID, types.ServiceSessionCreateInput{})
	require.ErrorIs(t, err, ErrServiceSpaceNotActive)
}

func TestServiceSpaceSubjectsSupportGenericHierarchyAndIsolation(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 70, "owner", types.ServiceSpaceCreateInput{
		Name: "会员服务主体测试",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "membership-expert", ExpertName: "会员服务专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)

	parent, err := svc.CreateSubject(ctx, 70, "owner", space.ID, types.ServiceSubjectCreateInput{
		SubjectType: "Member_Family",
		SubjectKey:  "family-001",
		DisplayName: "一号会员家庭",
		Metadata:    types.JSONMap{"level": "gold"},
	})
	require.NoError(t, err)
	require.Equal(t, "member_family", parent.SubjectType)
	require.Equal(t, "family-001", parent.SubjectKey)

	child, err := svc.CreateSubject(ctx, 70, "owner", space.ID, types.ServiceSubjectCreateInput{
		SubjectType:     "Child",
		SubjectKey:      "child-001",
		DisplayName:     "一号孩子",
		ParentSubjectID: &parent.ID,
	})
	require.NoError(t, err)
	require.Equal(t, "child", child.SubjectType)
	require.Equal(t, parent.ID, *child.ParentSubjectID)

	subjects, total, err := svc.ListSubjects(ctx, 70, "owner", space.ID, "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 2, total)
	require.Len(t, subjects, 2)

	updatedMetadata := types.JSONMap{"level": "platinum", "renewal_risk": "low"}
	updated, err := svc.UpdateSubject(ctx, 70, "owner", space.ID, parent.ID, types.ServiceSubjectUpdateInput{
		DisplayName: &[]string{"一号会员家庭（已更新）"}[0],
		Metadata:    &updatedMetadata,
	})
	require.NoError(t, err)
	require.Equal(t, "一号会员家庭（已更新）", updated.DisplayName)
	require.Equal(t, "platinum", updated.Metadata["level"])

	otherSpace, err := svc.Create(ctx, 70, "owner", types.ServiceSpaceCreateInput{
		Name: "另一服务空间",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "other-expert", ExpertName: "另一服务专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)
	_, err = svc.CreateSubject(ctx, 70, "owner", otherSpace.ID, types.ServiceSubjectCreateInput{
		SubjectType:     "project",
		SubjectKey:      "project-001",
		ParentSubjectID: &parent.ID,
	})
	require.ErrorIs(t, err, ErrServiceSpaceSubjectParent)

	_, err = svc.GetSubject(ctx, 70, "outsider", space.ID, parent.ID)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)

	require.NoError(t, svc.DeleteSubject(ctx, 70, "owner", space.ID, child.ID))
	_, err = svc.GetSubject(ctx, 70, "owner", space.ID, child.ID)
	require.ErrorIs(t, err, ErrServiceSpaceSubjectNotFound)
}

func TestServiceSpaceReminderStatusMachineSeedsAndGuardsChanges(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 71, "owner", types.ServiceSpaceCreateInput{
		Name: "Status machine",
	})
	require.NoError(t, err)

	statuses, err := svc.ListReminderStatuses(ctx, 71, "owner", space.ID, false)
	require.NoError(t, err)
	require.Len(t, statuses, 9)
	require.Equal(t, "candidate", statuses[0].StatusKey)
	require.True(t, statuses[0].IsInitial)

	transitions, err := svc.ListReminderStatusTransitions(ctx, 71, "owner", space.ID)
	require.NoError(t, err)
	require.Len(t, transitions, 12)

	custom, err := svc.CreateReminderStatus(ctx, 71, "owner", space.ID, types.ServiceReminderStatusCreateInput{
		StatusKey:    "awaiting_customer",
		Label:        "等待用户",
		Category:     types.ServiceReminderStatusCategoryInProgress,
		DisplayOrder: 25,
	})
	require.NoError(t, err)
	require.Equal(t, "awaiting_customer", custom.StatusKey)

	_, err = svc.CreateReminderStatus(ctx, 71, "owner", space.ID, types.ServiceReminderStatusCreateInput{
		StatusKey: "awaiting_customer",
		Label:     "重复状态",
		Category:  types.ServiceReminderStatusCategoryOpen,
	})
	require.ErrorIs(t, err, ErrServiceSpaceDuplicateStatusKey)

	_, err = svc.ReplaceReminderStatusTransitions(ctx, 71, "owner", space.ID, types.ServiceReminderStatusTransitionReplaceInput{
		Transitions: []types.ServiceReminderStatusTransitionInput{{
			FromStatusID: statuses[0].ID,
			ToStatusID:   custom.ID,
			Enabled:      true,
		}},
	})
	require.NoError(t, err)

	err = svc.DeleteReminderStatus(ctx, 71, "owner", space.ID, custom.ID)
	require.ErrorIs(t, err, ErrServiceSpaceStatusInUse)

	label := "等待客户反馈"
	updated, err := svc.UpdateReminderStatus(ctx, 71, "owner", space.ID, custom.ID, types.ServiceReminderStatusUpdateInput{
		Label: &label,
	})
	require.NoError(t, err)
	require.Equal(t, "等待客户反馈", updated.Label)
}

func TestServiceSpaceRemindersUseServiceScopeAndStatusTransitions(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 72, "owner", types.ServiceSpaceCreateInput{
		Name: "会员事项测试",
	})
	require.NoError(t, err)

	created, err := svc.CreateReminder(ctx, 72, "owner", space.ID, types.ServiceReminderCreateInput{
		Title:      "回访会员家庭",
		Summary:    "确认体验课后的反馈与下一步安排",
		Priority:   types.ServiceReminderPriorityHigh,
		DueText:    "本周五前",
		NextAction: "联系会员家庭并记录结果",
	})
	require.NoError(t, err)
	require.Equal(t, space.ID, created.ServiceID)
	require.Equal(t, space.ID, created.ProfileID)
	require.Equal(t, types.ServiceReminderStatusCandidate, created.Status)

	reminders, total, err := svc.ListReminders(ctx, 72, "owner", space.ID, "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, reminders, 1)

	targetStatus := "pending"
	updated, err := svc.UpdateReminder(ctx, 72, "owner", space.ID, created.ID, types.ServiceReminderUpdateInput{
		Status: &targetStatus,
	})
	require.NoError(t, err)
	require.Equal(t, targetStatus, updated.Status)

	invalidTarget := "completed"
	_, err = svc.UpdateReminder(ctx, 72, "owner", space.ID, created.ID, types.ServiceReminderUpdateInput{
		Status: &invalidTarget,
	})
	require.ErrorIs(t, err, ErrServiceSpaceReminderTransition)

	_, err = svc.GetReminder(ctx, 72, "outsider", space.ID, created.ID)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)

	require.NoError(t, svc.DeleteReminder(ctx, 72, "owner", space.ID, created.ID))
	_, err = svc.GetReminder(ctx, 72, "owner", space.ID, created.ID)
	require.ErrorIs(t, err, ErrServiceSpaceReminderNotFound)
}

func TestServiceSpaceReminderCollaborationSupportsHierarchyAssignmentsCommentsAndHistory(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 74, "owner", types.ServiceSpaceCreateInput{
		Name: "会员服务协作事项",
	})
	require.NoError(t, err)
	_, err = svc.AddMember(ctx, 74, "owner", space.ID, types.ServiceMemberInput{
		UserID: "coach",
		Role:   types.ServiceMemberRoleEditor,
	})
	require.NoError(t, err)

	parent, err := svc.CreateReminder(ctx, 74, "owner", space.ID, types.ServiceReminderCreateInput{
		Title:           "完成首次回访",
		AssigneeUserIDs: []string{"owner", "coach"},
	})
	require.NoError(t, err)
	require.Equal(t, 0, parent.Depth)

	child, err := svc.CreateReminder(ctx, 74, "owner", space.ID, types.ServiceReminderCreateInput{
		ParentReminderID: parent.ID,
		Title:            "整理回访结果",
		AssigneeUserIDs:  []string{"coach"},
	})
	require.NoError(t, err)
	require.Equal(t, parent.ID, child.ParentReminderID)
	require.Equal(t, 1, child.Depth)

	assignees, err := svc.ListReminderAssignees(ctx, 74, "owner", space.ID, parent.ID)
	require.NoError(t, err)
	require.Len(t, assignees, 2)
	require.Equal(t, "owner", assignees[0].UserID)
	require.Equal(t, types.ServiceReminderAssigneeRolePrimary, assignees[0].Role)

	comment, err := svc.AddReminderComment(ctx, 74, "coach", space.ID, parent.ID, types.ServiceReminderCommentCreateInput{
		Content: "已完成首次沟通，等待家长确认下次到访时间。",
	})
	require.NoError(t, err)
	require.Equal(t, "coach", comment.UserID)

	comments, err := svc.ListReminderComments(ctx, 74, "owner", space.ID, parent.ID)
	require.NoError(t, err)
	require.Len(t, comments, 1)
	require.Equal(t, comment.ID, comments[0].ID)

	history, err := svc.ListReminderHistory(ctx, 74, "owner", space.ID, parent.ID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(history), 2)

	_, err = svc.UpdateReminder(ctx, 74, "owner", space.ID, parent.ID, types.ServiceReminderUpdateInput{
		ParentReminderID: &parent.ID,
	})
	require.ErrorIs(t, err, ErrServiceSpaceReminderParent)
}

func TestServiceSpaceIndexesArtifactsUnderRunService(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, nil)

	space, err := svc.Create(ctx, 8, "owner", types.ServiceSpaceCreateInput{
		Name: "续费跟进",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "renewal-expert", ExpertName: "续费跟进专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)
	session, err := svc.CreateSession(ctx, 8, "owner", space.ID, types.ServiceSessionCreateInput{})
	require.NoError(t, err)

	run := &types.AgentRun{
		ID:        "run-service-artifact",
		TenantID:  8,
		UserID:    "owner",
		ServiceID: space.ID,
		ThreadID:  session.ID,
	}
	err = svc.IndexRunArtifacts(ctx, run, []types.AgentArtifactResultV1{
		{
			ID:           "artifact-renewal-list",
			VersionID:    "artifact-renewal-list-v1",
			Version:      1,
			Kind:         types.AgentArtifactKindReport,
			Role:         types.AgentArtifactRolePrimary,
			Title:        "本月到期未续费家长清单",
			Format:       types.StructuredReportFormatV1,
			Lifecycle:    types.AgentArtifactLifecycleSaved,
			Previewable:  true,
			Downloadable: true,
		},
	})
	require.NoError(t, err)

	artifacts, total, err := svc.ListArtifacts(ctx, 8, "owner", space.ID, "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, artifacts, 1)
	require.Equal(t, space.ID, artifacts[0].ServiceID)
	require.Equal(t, session.ID, run.ThreadID)
	require.Equal(t, "artifact-renewal-list", artifacts[0].ArtifactID)
	require.Equal(t, types.ServiceArtifactLifecycleSaved, artifacts[0].Lifecycle)
	archived, err := svc.UpdateArtifactLifecycle(ctx, 8, "owner", space.ID, "artifact-renewal-list", types.ServiceArtifactLifecycleArchived, "artifact-op-1")
	require.NoError(t, err)
	require.Equal(t, types.ServiceArtifactLifecycleArchived, archived.Lifecycle)
	replayed, err := svc.UpdateArtifactLifecycle(ctx, 8, "owner", space.ID, "artifact-renewal-list", types.ServiceArtifactLifecycleSaved, "artifact-op-1")
	require.NoError(t, err)
	require.Equal(t, types.ServiceArtifactLifecycleArchived, replayed.Lifecycle)

	other, err := svc.Create(ctx, 8, "owner", types.ServiceSpaceCreateInput{
		Name: "晨检异常",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "health-expert", ExpertName: "健康专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)
	otherArtifacts, otherTotal, err := svc.ListArtifacts(ctx, 8, "owner", other.ID, "", 1, 20)
	require.NoError(t, err)
	require.Zero(t, otherTotal)
	require.Empty(t, otherArtifacts)
}

func TestServiceSpaceReadsCurrentMarkdownArtifactsOnly(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	files := &serviceSpaceMarkdownFileStub{
		files: map[string]string{
			"resource://guide": "# 招生流程\n\n先确认学生年级。",
			"resource://notes": "这不是 Markdown 产出物。",
		},
	}
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), repository.NewOrganizeRepository(db), nil, files)

	space, err := svc.Create(ctx, 9, "owner", types.ServiceSpaceCreateInput{
		Name: "秋季招生咨询",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "admission-expert", ExpertName: "招生咨询专家"},
		},
		Activate: true,
	})
	require.NoError(t, err)
	session, err := svc.CreateSession(ctx, 9, "owner", space.ID, types.ServiceSessionCreateInput{})
	require.NoError(t, err)

	err = svc.IndexRunArtifacts(ctx, &types.AgentRun{
		ID:        "run-markdown-context",
		TenantID:  9,
		UserID:    "owner",
		ServiceID: space.ID,
		ThreadID:  session.ID,
	}, []types.AgentArtifactResultV1{
		{
			ID:           "markdown-guide",
			VersionID:    "markdown-guide-v1",
			Version:      1,
			Title:        "招生流程",
			Format:       "markdown",
			OriginalName: "招生流程.md",
			MimeType:     "text/markdown",
			ResourceRef:  "resource://guide",
			Lifecycle:    types.AgentArtifactLifecycleSaved,
		},
		{
			ID:           "text-notes",
			VersionID:    "text-notes-v1",
			Version:      1,
			Title:        "普通文本",
			Format:       "text",
			OriginalName: "普通文本.txt",
			MimeType:     "text/plain",
			ResourceRef:  "resource://notes",
			Lifecycle:    types.AgentArtifactLifecycleSaved,
		},
	})
	require.NoError(t, err)

	contextText, err := svc.ReadMarkdownContext(ctx, 9, "owner", space.ID)
	require.NoError(t, err)
	assert.Contains(t, contextText, "招生流程.md")
	assert.Contains(t, contextText, "先确认学生年级")
	assert.NotContains(t, contextText, "这不是 Markdown 产出物")
}

func TestServiceSpaceContextSourcesImportReadyOutputIdempotently(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), organizeRepo, nil, nil)

	space, err := svc.Create(ctx, 73, "owner", types.ServiceSpaceCreateInput{
		Name: "整理结果服务",
	})
	require.NoError(t, err)

	output := &types.OrganizeOutput{
		TenantID:        73,
		UserID:          "owner",
		Title:           "本周会员服务摘要",
		TemplateKey:     "weekly-service",
		TemplateVersion: "1.0",
		OutputType:      "summary",
		Content:         "# 本周摘要\n\n优先跟进待续费会员。",
		SourceSummary:   "来自本周会员服务记忆",
		Status:          types.OrganizeOutputStatusReady,
	}
	require.NoError(t, organizeRepo.CreateOutput(ctx, output, []string{"memory-001", "memory-002"}))

	source, err := svc.ImportOrganizeOutput(ctx, 73, "owner", space.ID, output.ID)
	require.NoError(t, err)
	assignedOutput, err := organizeRepo.GetOutput(ctx, 73, "owner", output.ID)
	require.NoError(t, err)
	require.Equal(t, types.OrganizeAssignmentStatusAssigned, assignedOutput.AssignmentStatus)
	require.Equal(t, space.ID, assignedOutput.AssignedServiceID)
	require.Equal(t, types.ServiceContextSourceTypeOrganizeOutput, source.SourceType)
	require.Equal(t, []string{"memory-001", "memory-002"}, []string(source.MemoryIDs))
	require.Equal(t, output.ID, source.SourceID)

	duplicate, err := svc.ImportOrganizeOutput(ctx, 73, "owner", space.ID, output.ID)
	require.NoError(t, err)
	require.Equal(t, source.ID, duplicate.ID)

	anotherSpace, err := svc.Create(ctx, 73, "owner", types.ServiceSpaceCreateInput{
		Name: "另一个整理结果服务",
	})
	require.NoError(t, err)
	_, err = svc.ImportOrganizeOutput(ctx, 73, "owner", anotherSpace.ID, output.ID)
	require.ErrorIs(t, err, ErrServiceSpaceContextSourceAssigned)

	sources, err := svc.ListContextSources(ctx, 73, "owner", space.ID)
	require.NoError(t, err)
	require.Len(t, sources, 1)

	runtimeContext, err := svc.ResolveRuntimeContext(ctx, 73, "owner", space.ID)
	require.NoError(t, err)
	require.Len(t, runtimeContext.ContextSources, 1)
	require.Len(t, runtimeContext.Facts, 1)
	require.Equal(t, types.ServiceFactTypeOrganizeOutput, runtimeContext.Facts[0].FactType)
	require.NotEmpty(t, runtimeContext.ContextHash)

	contextText, err := svc.ReadMarkdownContext(ctx, 73, "owner", space.ID)
	require.NoError(t, err)
	require.Contains(t, contextText, "优先跟进待续费会员")
	require.Contains(t, contextText, "服务空间事实")
	require.Contains(t, contextText, output.ID)

	require.NoError(t, svc.DeleteContextSource(ctx, 73, "owner", space.ID, source.ID))
	sources, err = svc.ListContextSources(ctx, 73, "owner", space.ID)
	require.NoError(t, err)
	require.Empty(t, sources)
}

func TestServiceSpaceFactsAreScopedAppendOnlyAndAudited(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	audit := &captureAuditLogService{}
	svc := NewServiceSpaceServiceWithDependencies(
		repository.NewServiceSpaceRepository(db),
		repository.NewOrganizeRepository(db),
		nil,
		nil,
		nil,
		audit,
	)

	first, err := svc.Create(ctx, 75, "owner", types.ServiceSpaceCreateInput{Name: "事实服务一"})
	require.NoError(t, err)
	second, err := svc.Create(ctx, 75, "owner", types.ServiceSpaceCreateInput{Name: "事实服务二"})
	require.NoError(t, err)

	input := types.ServiceFactAppendInput{
		FactType:      "member_status",
		FactKey:       "member-001",
		Value:         types.JSONMap{"status": "active"},
		SourceType:    "manual_note",
		SourceID:      "note-001",
		SourceVersion: "v1",
	}
	fact, err := svc.AppendFact(ctx, 75, "owner", first.ID, input)
	require.NoError(t, err)
	require.NotEmpty(t, fact.ID)

	duplicate, err := svc.AppendFact(ctx, 75, "owner", first.ID, input)
	require.NoError(t, err)
	require.Equal(t, fact.ID, duplicate.ID)

	facts, total, err := svc.ListFacts(ctx, 75, "owner", first.ID, "", "", "", "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 1, total)
	require.Len(t, facts, 1)
	summaryBefore, err := svc.GetSummary(ctx, 75, "owner", first.ID)
	require.NoError(t, err)
	require.NotEmpty(t, summaryBefore.SourceWatermark)

	otherFacts, total, err := svc.ListFacts(ctx, 75, "owner", second.ID, "", "", "", "", 1, 20)
	require.NoError(t, err)
	require.EqualValues(t, 0, total)
	require.Empty(t, otherFacts)

	_, err = svc.AppendFact(ctx, 75, "owner", first.ID, types.ServiceFactAppendInput{
		FactType: "member_status", FactKey: "member-002",
		Value:      types.JSONMap{"status": "pending"},
		SourceType: "manual_note", SourceID: "note-002",
	})
	require.NoError(t, err)
	summaryAfter, err := svc.GetSummary(ctx, 75, "owner", first.ID)
	require.NoError(t, err)
	require.Greater(t, summaryAfter.Version, summaryBefore.Version)
	require.NotEqual(t, summaryBefore.SourceWatermark, summaryAfter.SourceWatermark)

	_, _, err = svc.ListFacts(ctx, 75, "outsider", first.ID, "", "", "", "", 1, 20)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)

	_, err = svc.AppendFact(ctx, 75, "owner", first.ID, types.ServiceFactAppendInput{
		FactType: "member_status",
		Value:    types.JSONMap{"status": "active"},
	})
	require.ErrorIs(t, err, ErrServiceSpaceFactSourceRequired)

	require.NotEmpty(t, audit.entries)
	var sawFactAppend, sawSummaryRefresh bool
	for _, entry := range audit.entries {
		if entry.Action == types.AuditActionServiceFactAppend {
			sawFactAppend = true
		}
		if entry.Action == types.AuditActionServiceSummaryRefresh {
			sawSummaryRefresh = true
		}
	}
	require.True(t, sawFactAppend)
	require.True(t, sawSummaryRefresh)
}

func TestServiceSpaceContextSourcesRejectNonReadyOutputAndUnauthorizedUser(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), organizeRepo, nil, nil)

	space, err := svc.Create(ctx, 74, "owner", types.ServiceSpaceCreateInput{Name: "来源权限测试"})
	require.NoError(t, err)
	output := &types.OrganizeOutput{
		TenantID: 74,
		UserID:   "owner",
		Title:    "待审核结果",
		Status:   types.OrganizeOutputStatusReview,
	}
	require.NoError(t, organizeRepo.CreateOutput(ctx, output, nil))

	_, err = svc.ImportOrganizeOutput(ctx, 74, "owner", space.ID, output.ID)
	require.ErrorIs(t, err, ErrServiceSpaceContextSourceNotReady)

	_, err = svc.ListContextSources(ctx, 74, "outsider", space.ID)
	require.ErrorIs(t, err, ErrServiceSpaceNotFound)
}

type serviceSpaceMarkdownFileStub struct {
	files map[string]string
}

func (s *serviceSpaceMarkdownFileStub) CheckConnectivity(context.Context) error {
	return nil
}

func (s *serviceSpaceMarkdownFileStub) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *serviceSpaceMarkdownFileStub) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	return "", errors.New("not implemented")
}

func (s *serviceSpaceMarkdownFileStub) GetFile(_ context.Context, filePath string) (io.ReadCloser, error) {
	content, ok := s.files[filePath]
	if !ok {
		return nil, errors.New("file not found")
	}
	return io.NopCloser(strings.NewReader(content)), nil
}

func (s *serviceSpaceMarkdownFileStub) GetFileURL(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (s *serviceSpaceMarkdownFileStub) DeleteFile(context.Context, string) error {
	return nil
}

func (s *serviceSpaceMarkdownFileStub) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", errors.New("not implemented")
}
