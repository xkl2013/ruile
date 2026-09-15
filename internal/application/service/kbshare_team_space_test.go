package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type teamScopeKBShareRepoStub struct {
	interfaces.KBShareRepository
	shares []*types.KnowledgeBaseShare
}

func (s *teamScopeKBShareRepoStub) ListByKnowledgeBase(context.Context, string) ([]*types.KnowledgeBaseShare, error) {
	return s.shares, nil
}

func (s *teamScopeKBShareRepoStub) ListByOrganizations(context.Context, []string) ([]*types.KnowledgeBaseShare, error) {
	return s.shares, nil
}

func (s *teamScopeKBShareRepoStub) ListSharedKBsForTenant(context.Context, uint64) ([]*types.KnowledgeBaseShare, error) {
	return s.shares, nil
}

func TestCheckTenantKBPermissionRejectsTenantInternalOutsideActiveTenant(t *testing.T) {
	svc := newTenantInternalKBShareService()

	role, isShared, err := svc.CheckTenantKBPermission(context.Background(), "kb-1", 200, types.TenantRoleAdmin)

	require.NoError(t, err)
	require.False(t, isShared)
	require.Empty(t, role)
}

func TestCheckTenantKBPermissionAllowsTenantInternalOwnerTenant(t *testing.T) {
	svc := newTenantInternalKBShareService()

	role, isShared, err := svc.CheckTenantKBPermission(context.Background(), "kb-1", 100, types.TenantRoleAdmin)

	require.NoError(t, err)
	require.True(t, isShared)
	require.Equal(t, types.OrgRoleEditor, role)
}

func TestListSharedKnowledgeBaseIDsByOrganizationsRejectsTenantInternalOutsideActiveTenant(t *testing.T) {
	svc := newTenantInternalKBShareService()

	ids, err := svc.ListSharedKnowledgeBaseIDsByOrganizations(context.Background(), []string{"team-space-1"}, 200)

	require.NoError(t, err)
	require.Empty(t, ids["team-space-1"])
}

func newTenantInternalKBShareService() *kbShareService {
	scope := types.SharingScopeTenantInternal
	org := &types.Organization{ID: "team-space-1", OwnerTenantID: 100, SharingScope: &scope}
	orgRepo := &teamSpaceOrganizationRepoStub{
		orgsByID: map[string]*types.Organization{org.ID: org},
		members: map[string]*types.OrganizationTenantMember{
			org.ID: {
				OrganizationID:       org.ID,
				TenantID:             100,
				RepresentativeUserID: "user-1",
				Role:                 types.OrgRoleEditor,
			},
		},
	}
	share := &types.KnowledgeBaseShare{
		ID:              "share-1",
		KnowledgeBaseID: "kb-1",
		OrganizationID:  org.ID,
		SourceTenantID:  100,
		Permission:      types.OrgRoleEditor,
		Organization:    org,
		KnowledgeBase:   &types.KnowledgeBase{ID: "kb-1", TenantID: 100},
	}
	return &kbShareService{
		shareRepo: &teamScopeKBShareRepoStub{shares: []*types.KnowledgeBaseShare{share}},
		orgRepo:   orgRepo,
	}
}
