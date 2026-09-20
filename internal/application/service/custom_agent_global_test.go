package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type globalBuiltinAgentRepoStub struct {
	interfaces.CustomAgentRepository
	agent             *types.CustomAgent
	requestedTenantID uint64
	saved             *types.CustomAgent
}

func (r *globalBuiltinAgentRepoStub) GetAgentByID(
	_ context.Context,
	id string,
	tenantID uint64,
) (*types.CustomAgent, error) {
	r.requestedTenantID = tenantID
	if r.agent == nil || r.agent.ID != id || r.agent.TenantID != tenantID {
		return nil, repository.ErrCustomAgentNotFound
	}
	copy := *r.agent
	return &copy, nil
}

func (r *globalBuiltinAgentRepoStub) UpdateAgent(_ context.Context, agent *types.CustomAgent) error {
	copy := *agent
	r.saved = &copy
	return nil
}

func TestGetBuiltinAgentUsesGlobalConfigAndWorkspaceRuntimeScope(t *testing.T) {
	withQuickAnswerBuiltin(t)
	repo := &globalBuiltinAgentRepoStub{
		agent: &types.CustomAgent{
			ID:        types.BuiltinQuickAnswerID,
			Name:      "Quick Answer",
			IsBuiltin: true,
			TenantID:  types.SystemAgentTenantID,
			Config: types.CustomAgentConfig{
				ModelID: "system-chat",
			},
		},
	}
	svc := NewCustomAgentService(repo, nil, nil, nil, nil, nil, nil)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10003))

	agent, err := svc.GetAgentByID(ctx, types.BuiltinQuickAnswerID)

	require.NoError(t, err)
	require.Equal(t, types.SystemAgentTenantID, repo.requestedTenantID)
	require.Equal(t, uint64(10003), agent.TenantID)
	require.Equal(t, "system-chat", agent.Config.ModelID)
}

func TestUpdateBuiltinAgentRequiresSystemAdminAndSavesGlobally(t *testing.T) {
	withQuickAnswerBuiltin(t)
	repo := &globalBuiltinAgentRepoStub{
		agent: &types.CustomAgent{
			ID:        types.BuiltinQuickAnswerID,
			Name:      "Quick Answer",
			IsBuiltin: true,
			TenantID:  types.SystemAgentTenantID,
		},
	}
	svc := NewCustomAgentService(repo, nil, nil, nil, nil, nil, nil)
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(10003))
	update := &types.CustomAgent{
		ID: types.BuiltinQuickAnswerID,
		Config: types.CustomAgentConfig{
			ModelID: "system-chat",
		},
	}

	_, err := svc.UpdateAgent(ctx, update)
	require.ErrorIs(t, err, ErrBuiltinAgentRequiresSystemAdmin)
	require.Nil(t, repo.saved)

	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)
	agent, err := svc.UpdateAgent(ctx, update)
	require.NoError(t, err)
	require.NotNil(t, repo.saved)
	require.Equal(t, types.SystemAgentTenantID, repo.saved.TenantID)
	require.Equal(t, "system-chat", repo.saved.Config.ModelID)
	require.Equal(t, uint64(10003), agent.TenantID)
}

func withQuickAnswerBuiltin(t *testing.T) {
	t.Helper()
	oldFactory, existed := types.BuiltinAgentRegistry[types.BuiltinQuickAnswerID]
	types.BuiltinAgentRegistry[types.BuiltinQuickAnswerID] = func(tenantID uint64) *types.CustomAgent {
		return &types.CustomAgent{
			ID:        types.BuiltinQuickAnswerID,
			Name:      "Quick Answer",
			IsBuiltin: true,
			TenantID:  tenantID,
		}
	}
	t.Cleanup(func() {
		if existed {
			types.BuiltinAgentRegistry[types.BuiltinQuickAnswerID] = oldFactory
			return
		}
		delete(types.BuiltinAgentRegistry, types.BuiltinQuickAnswerID)
	})
}
