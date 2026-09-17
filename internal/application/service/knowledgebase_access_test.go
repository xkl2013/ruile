package service

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type accessKBShareService struct {
	interfaces.KBShareService
	permission types.OrgMemberRole
	shared     bool
	source     uint64
	checkErr   error
	sourceErr  error
}

type accessTenantRepo struct {
	interfaces.TenantRepository
	tenants map[uint64]*types.Tenant
}

func (r *accessTenantRepo) GetTenantByID(_ context.Context, id uint64) (*types.Tenant, error) {
	return r.tenants[id], nil
}

type accessTenantMemberService struct {
	interfaces.TenantMemberService
	members map[string]map[uint64]*types.TenantMember
}

func (s *accessTenantMemberService) GetMembership(
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	return s.members[userID][tenantID], nil
}

func (s *accessTenantMemberService) ListByUser(
	_ context.Context,
	userID string,
) ([]*types.TenantMember, error) {
	byTenant := s.members[userID]
	out := make([]*types.TenantMember, 0, len(byTenant))
	for _, member := range byTenant {
		out = append(out, member)
	}
	return out, nil
}

func (s *accessKBShareService) CheckTenantKBPermission(
	context.Context,
	string,
	uint64,
	types.TenantRole,
) (types.OrgMemberRole, bool, error) {
	if s.checkErr != nil {
		return "", false, s.checkErr
	}
	return s.permission, s.shared, nil
}

func (s *accessKBShareService) GetKBSourceTenant(context.Context, string) (uint64, error) {
	if s.sourceErr != nil {
		return 0, s.sourceErr
	}
	return s.source, nil
}

func accessCtx(tenantID uint64, role types.TenantRole, userID string, spaceType *types.SpaceType) context.Context {
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, tenantID)
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	if userID != "" {
		ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	}
	if spaceType != nil {
		ctx = context.WithValue(ctx, types.TenantInfoContextKey, &types.Tenant{
			ID:        tenantID,
			SpaceType: spaceType,
		})
	}
	return ctx
}

func TestResolveKnowledgeBaseAccess_CreatorGetsAdminFromPersonalSpace(t *testing.T) {
	spaceType := types.SpaceTypePersonal
	repo := newFakeKBRepo()
	repo.rows["kb-owned"] = &types.KnowledgeBase{ID: "kb-owned", TenantID: 100, CreatorID: "user-1"}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	access, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleViewer, "user-1", &spaceType),
		"kb-owned",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(100), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleAdmin, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceCreated, access.AccessSource)
	require.NotNil(t, access.OwnerType)
	require.Equal(t, types.SpaceTypePersonal, *access.OwnerType)
	require.False(t, access.IsSubscribed)
}

func TestResolveKnowledgeBaseAccess_SameTenantAdminGetsTenantAdminSource(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-team"] = &types.KnowledgeBase{ID: "kb-team", TenantID: 100, CreatorID: "user-1"}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	access, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleAdmin, "admin-1", nil),
		"kb-team",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(100), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleAdmin, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceTenantAdmin, access.AccessSource)
}

func TestResolveKnowledgeBaseAccess_SameTenantAdminIsCappedByViewerShare(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-team-viewer"] = &types.KnowledgeBase{
		ID:        "kb-team-viewer",
		TenantID:  100,
		CreatorID: "user-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleViewer,
		shared:     true,
		source:     100,
	}

	readAccess, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleOwner, "owner-2", nil),
		"kb-team-viewer",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)
	require.NoError(t, err)
	require.Equal(t, types.OrgRoleViewer, readAccess.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceSharedSpace, readAccess.AccessSource)

	_, err = svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleOwner, "owner-2", nil),
		"kb-team-viewer",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)
	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}

func TestResolveKnowledgeBaseAccess_CreatorCanOpenEnterpriseKBFromPersonalContext(t *testing.T) {
	personal := types.SpaceTypePersonal
	enterprise := types.SpaceTypeOrganization
	repo := newFakeKBRepo()
	repo.rows["kb-enterprise-created"] = &types.KnowledgeBase{
		ID:        "kb-enterprise-created",
		TenantID:  200,
		CreatorID: "user-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.tenantRepo = &accessTenantRepo{
		tenants: map[uint64]*types.Tenant{
			200: {ID: 200, SpaceType: &enterprise},
		},
	}
	svc.memberService = &accessTenantMemberService{
		members: map[string]map[uint64]*types.TenantMember{
			"user-1": {
				200: {
					UserID:   "user-1",
					TenantID: 200,
					Role:     types.TenantRoleContributor,
					Status:   types.TenantMemberStatusActive,
				},
			},
		},
	}

	access, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleOwner, "user-1", &personal),
		"kb-enterprise-created",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(200), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleAdmin, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceCreated, access.AccessSource)
	require.NotNil(t, access.OwnerType)
	require.Equal(t, types.SpaceTypeOrganization, *access.OwnerType)
}

func TestResolveKnowledgeBaseAccess_CreatorLosesEnterpriseKBWhenMembershipInactive(t *testing.T) {
	personal := types.SpaceTypePersonal
	enterprise := types.SpaceTypeOrganization
	repo := newFakeKBRepo()
	repo.rows["kb-enterprise-created"] = &types.KnowledgeBase{
		ID:        "kb-enterprise-created",
		TenantID:  200,
		CreatorID: "user-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.tenantRepo = &accessTenantRepo{
		tenants: map[uint64]*types.Tenant{
			200: {ID: 200, SpaceType: &enterprise},
		},
	}
	svc.memberService = &accessTenantMemberService{
		members: map[string]map[uint64]*types.TenantMember{
			"user-1": {
				200: {
					UserID:   "user-1",
					TenantID: 200,
					Role:     types.TenantRoleContributor,
					Status:   types.TenantMemberStatusSuspended,
				},
			},
		},
	}

	_, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleOwner, "user-1", &personal),
		"kb-enterprise-created",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}

func TestResolveKnowledgeBaseAccess_SharedKBUsesSourceTenant(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-shared"] = &types.KnowledgeBase{ID: "kb-shared", TenantID: 200, CreatorID: "owner-1"}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleEditor,
		shared:     true,
		source:     200,
	}

	access, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleContributor, "member-1", nil),
		"kb-shared",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(200), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleEditor, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceSharedSpace, access.AccessSource)
}

func TestResolveKnowledgeBaseAccess_SameTenantShareCanGrantNonCreatorRead(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-same-tenant"] = &types.KnowledgeBase{ID: "kb-same-tenant", TenantID: 100, CreatorID: "owner-1"}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleViewer,
		shared:     true,
		source:     100,
	}

	access, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleViewer, "member-1", nil),
		"kb-same-tenant",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(100), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleViewer, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceSharedSpace, access.AccessSource)
}

func TestResolveKnowledgeBaseAccess_SharedPermissionBelowRequirementIsForbidden(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-shared"] = &types.KnowledgeBase{ID: "kb-shared", TenantID: 200}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleViewer,
		shared:     true,
		source:     200,
	}

	_, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleContributor, "member-1", nil),
		"kb-shared",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}

func TestResolveKnowledgeBaseAccess_PersonalKBCannotBeReachedAcrossTenants(t *testing.T) {
	personal := types.SpaceTypePersonal
	repo := newFakeKBRepo()
	repo.rows["kb-personal"] = &types.KnowledgeBase{ID: "kb-personal", TenantID: 200}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.tenantRepo = &accessTenantRepo{
		tenants: map[uint64]*types.Tenant{
			200: {ID: 200, SpaceType: &personal},
		},
	}
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleEditor,
		shared:     true,
		source:     200,
	}

	_, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleContributor, "member-1", nil),
		"kb-personal",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}

func TestResolveKnowledgeBaseAccess_SystemAdminGetsSourceTenant(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-system"] = &types.KnowledgeBase{ID: "kb-system", TenantID: 200}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	ctx := accessCtx(100, types.TenantRoleViewer, "system-1", nil)
	ctx = context.WithValue(ctx, types.SystemAdminContextKey, true)

	access, err := svc.ResolveKnowledgeBaseAccess(
		ctx,
		"kb-system",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleAdmin},
	)

	require.NoError(t, err)
	require.Equal(t, uint64(200), access.EffectiveTenantID)
	require.Equal(t, types.OrgRoleAdmin, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceSystemAdmin, access.AccessSource)
}

func TestResolveKnowledgeBaseAccess_UnauthorizedAndNotFound(t *testing.T) {
	repo := newFakeKBRepo()
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	_, err := svc.ResolveKnowledgeBaseAccess(
		context.Background(),
		"kb-missing",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)
	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessUnauthorized)

	_, err = svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleViewer, "user-1", nil),
		"kb-missing",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)
	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessNotFound)
}

func TestResolveKnowledgeBaseAccess_SourceTenantLookupFailureDeniesLikeLegacyGuard(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-shared"] = &types.KnowledgeBase{ID: "kb-shared", TenantID: 200}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleEditor,
		shared:     true,
		sourceErr:  stderrors.New("source lookup failed"),
	}

	_, err := svc.ResolveKnowledgeBaseAccess(
		accessCtx(100, types.TenantRoleContributor, "member-1", nil),
		"kb-shared",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}
