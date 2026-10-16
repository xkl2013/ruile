package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/tools"
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
	require.Contains(t, req.ServiceRuntimeInstruction, "待确认提案")
	require.Contains(t, req.ServiceRuntimeInstruction, "不能直接修改服务档案")
}

func TestApplyServiceOrchestrationUsesBoundMemberAgents(t *testing.T) {
	ctx := context.Background()
	db := newAgentRunTestDB(t)
	serviceSpaces := NewServiceSpaceService(
		repository.NewServiceSpaceRepository(db),
		repository.NewOrganizeRepository(db),
		nil,
		nil,
	)
	space, err := serviceSpaces.Create(ctx, 94, "owner", types.ServiceSpaceCreateInput{
		Name: "多 Agent 服务",
		Experts: []types.ServiceExpertBindingInput{
			{ExpertRef: "agent-admission", ExpertName: "招生 Agent"},
			{ExpertRef: "agent-course", ExpertName: "课程 Agent"},
		},
		Activate: true,
	})
	require.NoError(t, err)

	svc := &sessionService{serviceSpace: serviceSpaces}
	req := &types.QARequest{
		Session: &types.Session{
			TenantID:  94,
			UserID:    "owner",
			ServiceID: space.ID,
		},
	}
	config := &types.AgentConfig{
		AllowedTools: []string{"knowledge_search", tools.ToolDelegateAgent},
	}
	require.NoError(t, svc.applyServiceOrchestration(ctx, req, config))
	require.True(t, config.OrchestrationEnabled)
	require.Equal(t, []string{"agent-admission", "agent-course"}, config.MemberAgentIDs)
	require.Equal(t, []string{"knowledge_search", tools.ToolDelegateAgent}, config.AllowedTools)
	require.NotEmpty(t, config.BusinessInstructions)
	require.Equal(t, "service_orchestrator", config.BusinessInstructions[len(config.BusinessInstructions)-1].Label)
}

func TestApplyServiceRuntimeContextRejectsDirectMemberAgent(t *testing.T) {
	svc := &sessionService{
		serviceSpace: &serviceSpaceService{},
	}
	req := &types.QARequest{
		Session: &types.Session{
			ServiceID: "service-1",
		},
		CustomAgent: &types.CustomAgent{
			ID: "member-agent",
		},
	}

	err := svc.applyServiceRuntimeContext(context.Background(), req)
	require.ErrorIs(t, err, ErrServiceSpacePrincipalRequired)
}
