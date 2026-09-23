package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestServiceSpaceLifecycleMembershipAndOwnership(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil)

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

func TestServiceSpaceIndexesArtifactsUnderRunService(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil)

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
