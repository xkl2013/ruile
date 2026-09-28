package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestApplyServiceRuntimeContextLoadsImportedOrganizeSources(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	organizeRepo := repository.NewOrganizeRepository(db)
	serviceSpaces := NewServiceSpaceService(
		repository.NewServiceSpaceRepository(db),
		organizeRepo,
		nil,
		nil,
	)

	space, err := serviceSpaces.Create(ctx, 93, "owner", types.ServiceSpaceCreateInput{
		Name: "问答上下文服务",
	})
	require.NoError(t, err)

	output := &types.OrganizeOutput{
		TenantID:      93,
		UserID:        "owner",
		Title:         "会员服务事实",
		SourceSummary: "本周会员续费事实",
		Content:       "# 续费事实\n\n会员已确认下月续费。",
		Status:        types.OrganizeOutputStatusReady,
	}
	require.NoError(t, organizeRepo.CreateOutput(ctx, output, nil))
	_, err = serviceSpaces.ImportOrganizeOutput(ctx, 93, "owner", space.ID, output.ID)
	require.NoError(t, err)

	svc := &sessionService{serviceSpace: serviceSpaces}
	req := &types.QARequest{
		Session: &types.Session{
			TenantID:  93,
			UserID:    "owner",
			ServiceID: space.ID,
		},
	}
	require.NoError(t, svc.applyServiceRuntimeContext(ctx, req))
	require.Contains(t, req.ServiceRuntimeContext, "会员已确认下月续费")
	require.NotEmpty(t, req.ServiceRuntimeContextHash)
}
