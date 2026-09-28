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
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil, nil)

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
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil, nil)

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

func TestServiceSpaceReadsCurrentMarkdownArtifactsOnly(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	files := &serviceSpaceMarkdownFileStub{
		files: map[string]string{
			"resource://guide": "# 招生流程\n\n先确认学生年级。",
			"resource://notes": "这不是 Markdown 产出物。",
		},
	}
	svc := NewServiceSpaceService(repository.NewServiceSpaceRepository(db), nil, files)

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
