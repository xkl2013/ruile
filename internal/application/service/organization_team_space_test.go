package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type teamSpaceOrganizationRepoStub struct {
	interfaces.OrganizationRepository
	org       *types.Organization
	orgs      []*types.Organization
	orgsByID  map[string]*types.Organization
	members   map[string]*types.OrganizationTenantMember
	added     *types.OrganizationTenantMember
	deleteID  string
	deleteErr error
}

func (s *teamSpaceOrganizationRepoStub) GetByID(_ context.Context, id string) (*types.Organization, error) {
	if s.orgsByID != nil {
		if org := s.orgsByID[id]; org != nil {
			return org, nil
		}
		return nil, repository.ErrOrganizationNotFound
	}
	if s.org != nil {
		return s.org, nil
	}
	return nil, repository.ErrOrganizationNotFound
}

func (s *teamSpaceOrganizationRepoStub) ListByTenantID(context.Context, uint64) ([]*types.Organization, error) {
	return s.orgs, nil
}

func (s *teamSpaceOrganizationRepoStub) CountTenantMembers(context.Context, string) (int64, error) {
	return 1, nil
}

func (s *teamSpaceOrganizationRepoStub) AddTenantMember(_ context.Context, member *types.OrganizationTenantMember) error {
	s.added = member
	return nil
}

func (s *teamSpaceOrganizationRepoStub) GetTenantMember(_ context.Context, orgID string, _ uint64) (*types.OrganizationTenantMember, error) {
	if s.members != nil {
		if member := s.members[orgID]; member != nil {
			return member, nil
		}
	}
	return nil, repository.ErrOrgMemberNotFound
}

func (s *teamSpaceOrganizationRepoStub) ListTenantMembersByTenantForOrgs(_ context.Context, _ uint64, orgIDs []string) (map[string]*types.OrganizationTenantMember, error) {
	out := make(map[string]*types.OrganizationTenantMember)
	for _, orgID := range orgIDs {
		if s.members != nil && s.members[orgID] != nil {
			out[orgID] = s.members[orgID]
		}
	}
	return out, nil
}

func (s *teamSpaceOrganizationRepoStub) Delete(_ context.Context, id string) error {
	s.deleteID = id
	return s.deleteErr
}

type teamSpaceTenantRepoStub struct {
	interfaces.TenantRepository
	tenant *types.Tenant
}

func (s *teamSpaceTenantRepoStub) GetTenantByID(context.Context, uint64) (*types.Tenant, error) {
	return s.tenant, nil
}

func TestAddTenantMemberEnforcesOwnerEnterpriseBoundary(t *testing.T) {
	scope := types.SharingScopeTenantInternal
	spaceType := types.SpaceTypeOrganization
	orgRepo := &teamSpaceOrganizationRepoStub{
		org: &types.Organization{
			ID:            "team-space-1",
			OwnerTenantID: 100,
			SharingScope:  &scope,
			MemberLimit:   10,
		},
	}
	svc := &organizationService{
		orgRepo:    orgRepo,
		tenantRepo: &teamSpaceTenantRepoStub{tenant: &types.Tenant{ID: 200, SpaceType: &spaceType}},
	}

	err := svc.AddTenantMember(
		context.Background(),
		"team-space-1",
		200,
		"user-200",
		types.OrgRoleViewer,
	)

	require.ErrorIs(t, err, ErrOrganizationMemberOutsideOwnerWorkspace)
	require.Nil(t, orgRepo.added)
}

func TestAddTenantMemberAllowsOwnerEnterpriseAccounts(t *testing.T) {
	scope := types.SharingScopeTenantInternal
	spaceType := types.SpaceTypeOrganization
	orgRepo := &teamSpaceOrganizationRepoStub{
		org: &types.Organization{
			ID:            "team-space-1",
			OwnerTenantID: 100,
			SharingScope:  &scope,
			MemberLimit:   10,
		},
	}
	svc := &organizationService{
		orgRepo:    orgRepo,
		tenantRepo: &teamSpaceTenantRepoStub{tenant: &types.Tenant{ID: 100, SpaceType: &spaceType}},
	}

	err := svc.AddTenantMember(
		context.Background(),
		"team-space-1",
		100,
		"user-100",
		types.OrgRoleViewer,
	)

	require.NoError(t, err)
	require.NotNil(t, orgRepo.added)
	require.Equal(t, uint64(100), orgRepo.added.TenantID)
	require.Equal(t, "user-100", orgRepo.added.RepresentativeUserID)
}

func TestListTenantOrganizationsFiltersTenantInternalByActiveTenant(t *testing.T) {
	teamScope := types.SharingScopeTenantInternal
	legacyScope := types.SharingScopeLegacyCrossSpace
	orgRepo := &teamSpaceOrganizationRepoStub{
		orgs: []*types.Organization{
			{ID: "owner-team", OwnerTenantID: 100, SharingScope: &teamScope},
			{ID: "other-team", OwnerTenantID: 300, SharingScope: &teamScope},
			{ID: "legacy-space", OwnerTenantID: 300, SharingScope: &legacyScope},
		},
	}
	svc := &organizationService{orgRepo: orgRepo}

	orgs, err := svc.ListTenantOrganizations(context.Background(), 100)

	require.NoError(t, err)
	require.ElementsMatch(t, []string{"owner-team", "legacy-space"}, collectOrganizationIDs(orgs))
}

func TestGetTenantRoleInOrgRejectsTenantInternalOutsideActiveTenant(t *testing.T) {
	teamScope := types.SharingScopeTenantInternal
	orgRepo := &teamSpaceOrganizationRepoStub{
		orgsByID: map[string]*types.Organization{
			"owner-team": {ID: "owner-team", OwnerTenantID: 100, SharingScope: &teamScope},
		},
		members: map[string]*types.OrganizationTenantMember{
			"owner-team": {
				OrganizationID:       "owner-team",
				TenantID:             100,
				RepresentativeUserID: "user-1",
				Role:                 types.OrgRoleAdmin,
			},
		},
	}
	svc := &organizationService{orgRepo: orgRepo}

	role, err := svc.GetTenantRoleInOrg(context.Background(), "owner-team", 200)

	require.ErrorIs(t, err, ErrTenantNotInOrg)
	require.Empty(t, role)
}

func TestGetTenantRoleInOrgAllowsTenantInternalOwnerTenant(t *testing.T) {
	teamScope := types.SharingScopeTenantInternal
	orgRepo := &teamSpaceOrganizationRepoStub{
		orgsByID: map[string]*types.Organization{
			"owner-team": {ID: "owner-team", OwnerTenantID: 100, SharingScope: &teamScope},
		},
		members: map[string]*types.OrganizationTenantMember{
			"owner-team": {
				OrganizationID:       "owner-team",
				TenantID:             100,
				RepresentativeUserID: "user-1",
				Role:                 types.OrgRoleAdmin,
			},
		},
	}
	svc := &organizationService{orgRepo: orgRepo}

	role, err := svc.GetTenantRoleInOrg(context.Background(), "owner-team", 100)

	require.NoError(t, err)
	require.Equal(t, types.OrgRoleAdmin, role)
}

func collectOrganizationIDs(orgs []*types.Organization) []string {
	out := make([]string, 0, len(orgs))
	for _, org := range orgs {
		out = append(out, org.ID)
	}
	return out
}
