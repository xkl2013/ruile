package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type teamScopeAgentShareRepoStub struct {
	interfaces.AgentShareRepository
	shares          []*types.AgentShare
	excludeTenantID uint64
}

func (s *teamScopeAgentShareRepoStub) ListSharedAgentsForTenant(context.Context, uint64) ([]*types.AgentShare, error) {
	return s.shares, nil
}

func (s *teamScopeAgentShareRepoStub) GetShareByAgentIDForTenant(_ context.Context, _ uint64, agentID string, excludeTenantID uint64) (*types.AgentShare, error) {
	s.excludeTenantID = excludeTenantID
	for _, share := range s.shares {
		if share.AgentID == agentID {
			return share, nil
		}
	}
	return nil, repository.ErrAgentShareNotFound
}

type teamScopeDisabledAgentRepoStub struct {
	interfaces.TenantDisabledSharedAgentRepository
}

func (s *teamScopeDisabledAgentRepoStub) ListByTenantID(context.Context, uint64) ([]*types.TenantDisabledSharedAgent, error) {
	return nil, nil
}

type teamScopeAgentRepoStub struct {
	interfaces.CustomAgentRepository
}

func (s *teamScopeAgentRepoStub) GetAgentByID(_ context.Context, agentID string, tenantID uint64) (*types.CustomAgent, error) {
	return &types.CustomAgent{ID: agentID, TenantID: tenantID, Name: "Team agent"}, nil
}

func TestListSharedAgentsIncludesTenantInternalSameTenantShare(t *testing.T) {
	svc, _ := newTenantInternalAgentShareService()

	agents, err := svc.ListSharedAgents(context.Background(), 100, types.TenantRoleAdmin)

	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, "agent-1", agents[0].Agent.ID)
	require.Equal(t, uint64(100), agents[0].SourceTenantID)
}

func TestListSharedAgentsRejectsTenantInternalOutsideActiveTenant(t *testing.T) {
	svc, _ := newTenantInternalAgentShareService()

	agents, err := svc.ListSharedAgents(context.Background(), 200, types.TenantRoleAdmin)

	require.NoError(t, err)
	require.Empty(t, agents)
}

func TestGetSharedAgentForTenantAllowsTenantInternalSameTenantShare(t *testing.T) {
	svc, shareRepo := newTenantInternalAgentShareService()

	agent, err := svc.GetSharedAgentForTenant(context.Background(), 100, types.TenantRoleAdmin, "agent-1")

	require.NoError(t, err)
	require.Equal(t, "agent-1", agent.ID)
	require.Equal(t, uint64(100), agent.TenantID)
	require.Zero(t, shareRepo.excludeTenantID)
}

func TestGetSharedAgentForTenantRejectsTenantInternalOutsideActiveTenant(t *testing.T) {
	svc, _ := newTenantInternalAgentShareService()

	agent, err := svc.GetSharedAgentForTenant(context.Background(), 200, types.TenantRoleAdmin, "agent-1")

	require.ErrorIs(t, err, ErrAgentSharePermission)
	require.Nil(t, agent)
}

func newTenantInternalAgentShareService() (*agentShareService, *teamScopeAgentShareRepoStub) {
	scope := types.SharingScopeTenantInternal
	org := &types.Organization{ID: "team-space-1", OwnerTenantID: 100, SharingScope: &scope}
	agent := &types.CustomAgent{ID: "agent-1", TenantID: 100, Name: "Team agent"}
	share := &types.AgentShare{
		ID:             "share-1",
		AgentID:        agent.ID,
		OrganizationID: org.ID,
		SourceTenantID: 100,
		Permission:     types.OrgRoleViewer,
		Organization:   org,
		Agent:          agent,
	}
	shareRepo := &teamScopeAgentShareRepoStub{shares: []*types.AgentShare{share}}
	orgRepo := &teamSpaceOrganizationRepoStub{
		orgsByID: map[string]*types.Organization{org.ID: org},
		members: map[string]*types.OrganizationTenantMember{
			org.ID: {
				OrganizationID:       org.ID,
				TenantID:             100,
				RepresentativeUserID: "user-1",
				Role:                 types.OrgRoleViewer,
			},
		},
	}
	return &agentShareService{
		shareRepo:    shareRepo,
		disabledRepo: &teamScopeDisabledAgentRepoStub{},
		orgRepo:      orgRepo,
		agentRepo:    &teamScopeAgentRepoStub{},
	}, shareRepo
}
