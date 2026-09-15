package handler

import (
	"context"
	"testing"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type knowledgeBaseCreationTenantServiceStub struct {
	interfaces.TenantService
	tenants map[uint64]*types.Tenant
}

func (s *knowledgeBaseCreationTenantServiceStub) GetTenantByID(
	_ context.Context,
	id uint64,
) (*types.Tenant, error) {
	tenant, ok := s.tenants[id]
	if !ok {
		return nil, errKnowledgeBaseCreationTenantNotFound{id: id}
	}
	return tenant, nil
}

type errKnowledgeBaseCreationTenantNotFound struct {
	id uint64
}

func (e errKnowledgeBaseCreationTenantNotFound) Error() string {
	return "tenant not found"
}

type knowledgeBaseCreationMemberServiceStub struct {
	interfaces.TenantMemberService
	members map[string][]*types.TenantMember
}

func (s *knowledgeBaseCreationMemberServiceStub) GetMembership(
	_ context.Context,
	userID string,
	tenantID uint64,
) (*types.TenantMember, error) {
	for _, member := range s.members[userID] {
		if member != nil && member.TenantID == tenantID {
			return member, nil
		}
	}
	return nil, nil
}

func (s *knowledgeBaseCreationMemberServiceStub) ListByUser(
	_ context.Context,
	userID string,
) ([]*types.TenantMember, error) {
	return s.members[userID], nil
}

func testKnowledgeBaseCreationTenant(id uint64, spaceType types.SpaceType) *types.Tenant {
	return &types.Tenant{
		ID:        id,
		Name:      "tenant",
		SpaceType: &spaceType,
	}
}

func testKnowledgeBaseCreationMember(
	userID string,
	tenantID uint64,
	role types.TenantRole,
	status types.TenantMemberStatus,
) *types.TenantMember {
	return &types.TenantMember{
		UserID:   userID,
		TenantID: tenantID,
		Role:     role,
		Status:   status,
	}
}

func testKnowledgeBaseCreationContext(
	userID string,
	currentTenantID uint64,
	currentTenant *types.Tenant,
	userTenantID uint64,
	role types.TenantRole,
) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
	ctx = context.WithValue(ctx, types.TenantIDContextKey, currentTenantID)
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, currentTenant)
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	ctx = context.WithValue(ctx, types.UserContextKey, &types.User{
		ID:       userID,
		TenantID: userTenantID,
	})
	return ctx
}

func newKnowledgeBaseCreationHandler(
	tenants map[uint64]*types.Tenant,
	members map[string][]*types.TenantMember,
) *KnowledgeBaseHandler {
	return &KnowledgeBaseHandler{
		tenantService:       &knowledgeBaseCreationTenantServiceStub{tenants: tenants},
		tenantMemberService: &knowledgeBaseCreationMemberServiceStub{members: members},
	}
}

func TestResolveKnowledgeBaseCreationContextDefaultsToPersonalHome(t *testing.T) {
	const userID = "user-1"
	personal := testKnowledgeBaseCreationTenant(10, types.SpaceTypePersonal)
	enterprise := testKnowledgeBaseCreationTenant(20, types.SpaceTypeOrganization)
	h := newKnowledgeBaseCreationHandler(
		map[uint64]*types.Tenant{
			personal.ID:   personal,
			enterprise.ID: enterprise,
		},
		map[string][]*types.TenantMember{
			userID: {
				testKnowledgeBaseCreationMember(userID, personal.ID, types.TenantRoleOwner, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, enterprise.ID, types.TenantRoleContributor, types.TenantMemberStatusActive),
			},
		},
	)
	ctx := testKnowledgeBaseCreationContext(userID, enterprise.ID, enterprise, personal.ID, types.TenantRoleContributor)

	resolved, tenantID, err := h.resolveKnowledgeBaseCreationContext(ctx, CreateKnowledgeBaseRequest{
		Name: "personal kb",
	})
	if err != nil {
		t.Fatalf("resolve creation context: %v", err)
	}
	if tenantID != personal.ID {
		t.Fatalf("expected personal home tenant %d, got %d", personal.ID, tenantID)
	}
	resolvedTenantID, ok := types.TenantIDFromContext(resolved)
	if !ok || resolvedTenantID != personal.ID {
		t.Fatalf("resolved context tenant = %d, ok=%v", resolvedTenantID, ok)
	}
	if got := types.TenantRoleFromContext(resolved); got != types.TenantRoleOwner {
		t.Fatalf("resolved personal role = %q, want %q", got, types.TenantRoleOwner)
	}
}

func TestResolveKnowledgeBaseCreationContextSelectsEnterpriseMembership(t *testing.T) {
	const userID = "user-1"
	personal := testKnowledgeBaseCreationTenant(10, types.SpaceTypePersonal)
	enterprise := testKnowledgeBaseCreationTenant(20, types.SpaceTypeOrganization)
	h := newKnowledgeBaseCreationHandler(
		map[uint64]*types.Tenant{
			personal.ID:   personal,
			enterprise.ID: enterprise,
		},
		map[string][]*types.TenantMember{
			userID: {
				testKnowledgeBaseCreationMember(userID, personal.ID, types.TenantRoleOwner, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, enterprise.ID, types.TenantRoleContributor, types.TenantMemberStatusActive),
			},
		},
	)
	ctx := testKnowledgeBaseCreationContext(userID, personal.ID, personal, personal.ID, types.TenantRoleOwner)

	resolved, tenantID, err := h.resolveKnowledgeBaseCreationContext(ctx, CreateKnowledgeBaseRequest{
		Name:               "enterprise kb",
		Scope:              knowledgeBaseCreationScopeEnterprise,
		EnterpriseTenantID: enterprise.ID,
	})
	if err != nil {
		t.Fatalf("resolve creation context: %v", err)
	}
	if tenantID != enterprise.ID {
		t.Fatalf("expected enterprise tenant %d, got %d", enterprise.ID, tenantID)
	}
	if resolvedTenantID, _ := types.TenantIDFromContext(resolved); resolvedTenantID != enterprise.ID {
		t.Fatalf("resolved context tenant = %d, want %d", resolvedTenantID, enterprise.ID)
	}
	if got := types.TenantRoleFromContext(resolved); got != types.TenantRoleContributor {
		t.Fatalf("resolved enterprise role = %q, want %q", got, types.TenantRoleContributor)
	}
}

func TestResolveKnowledgeBaseCreationContextRequiresEnterpriseTargetWhenAmbiguous(t *testing.T) {
	const userID = "user-1"
	personal := testKnowledgeBaseCreationTenant(10, types.SpaceTypePersonal)
	firstEnterprise := testKnowledgeBaseCreationTenant(20, types.SpaceTypeOrganization)
	secondEnterprise := testKnowledgeBaseCreationTenant(30, types.SpaceTypeOrganization)
	h := newKnowledgeBaseCreationHandler(
		map[uint64]*types.Tenant{
			personal.ID:         personal,
			firstEnterprise.ID:  firstEnterprise,
			secondEnterprise.ID: secondEnterprise,
		},
		map[string][]*types.TenantMember{
			userID: {
				testKnowledgeBaseCreationMember(userID, personal.ID, types.TenantRoleOwner, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, firstEnterprise.ID, types.TenantRoleContributor, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, secondEnterprise.ID, types.TenantRoleAdmin, types.TenantMemberStatusActive),
			},
		},
	)
	ctx := testKnowledgeBaseCreationContext(userID, personal.ID, personal, personal.ID, types.TenantRoleOwner)

	_, _, err := h.resolveKnowledgeBaseCreationContext(ctx, CreateKnowledgeBaseRequest{
		Name:  "ambiguous enterprise kb",
		Scope: knowledgeBaseCreationScopeEnterprise,
	})
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.HTTPCode != 409 {
		t.Fatalf("expected HTTP 409, got %d (%v)", appErr.HTTPCode, err)
	}
}

func TestResolveKnowledgeBaseCreationContextRejectsUnauthorizedEnterpriseTarget(t *testing.T) {
	const userID = "user-1"
	personal := testKnowledgeBaseCreationTenant(10, types.SpaceTypePersonal)
	allowedEnterprise := testKnowledgeBaseCreationTenant(20, types.SpaceTypeOrganization)
	otherEnterprise := testKnowledgeBaseCreationTenant(30, types.SpaceTypeOrganization)
	h := newKnowledgeBaseCreationHandler(
		map[uint64]*types.Tenant{
			personal.ID:          personal,
			allowedEnterprise.ID: allowedEnterprise,
			otherEnterprise.ID:   otherEnterprise,
		},
		map[string][]*types.TenantMember{
			userID: {
				testKnowledgeBaseCreationMember(userID, personal.ID, types.TenantRoleOwner, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, allowedEnterprise.ID, types.TenantRoleContributor, types.TenantMemberStatusActive),
			},
		},
	)
	ctx := testKnowledgeBaseCreationContext(userID, personal.ID, personal, personal.ID, types.TenantRoleOwner)

	_, _, err := h.resolveKnowledgeBaseCreationContext(ctx, CreateKnowledgeBaseRequest{
		Name:               "unauthorized enterprise kb",
		Scope:              knowledgeBaseCreationScopeEnterprise,
		EnterpriseTenantID: otherEnterprise.ID,
	})
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.HTTPCode != 403 {
		t.Fatalf("expected HTTP 403, got %d (%v)", appErr.HTTPCode, err)
	}
}

func TestResolveKnowledgeBaseCreationContextIgnoresInactiveEnterpriseMembership(t *testing.T) {
	const userID = "user-1"
	personal := testKnowledgeBaseCreationTenant(10, types.SpaceTypePersonal)
	enterprise := testKnowledgeBaseCreationTenant(20, types.SpaceTypeOrganization)
	h := newKnowledgeBaseCreationHandler(
		map[uint64]*types.Tenant{
			personal.ID:   personal,
			enterprise.ID: enterprise,
		},
		map[string][]*types.TenantMember{
			userID: {
				testKnowledgeBaseCreationMember(userID, personal.ID, types.TenantRoleOwner, types.TenantMemberStatusActive),
				testKnowledgeBaseCreationMember(userID, enterprise.ID, types.TenantRoleAdmin, types.TenantMemberStatusSuspended),
			},
		},
	)
	ctx := testKnowledgeBaseCreationContext(userID, personal.ID, personal, personal.ID, types.TenantRoleOwner)

	_, _, err := h.resolveKnowledgeBaseCreationContext(ctx, CreateKnowledgeBaseRequest{
		Name:  "inactive enterprise kb",
		Scope: knowledgeBaseCreationScopeEnterprise,
	})
	appErr, ok := err.(*apperrors.AppError)
	if !ok {
		t.Fatalf("expected AppError, got %T (%v)", err, err)
	}
	if appErr.HTTPCode != 403 {
		t.Fatalf("expected HTTP 403, got %d (%v)", appErr.HTTPCode, err)
	}
}
