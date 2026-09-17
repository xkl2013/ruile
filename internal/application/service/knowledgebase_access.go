package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// ResolveKnowledgeBaseAccess centralizes the current KB access semantics for
// V1 of the personal/organization knowledge-space rollout. It deliberately
// mirrors the existing behavior before V2 starts adding subscriptions and
// organization-internal sharing constraints.
func (s *knowledgeBaseService) ResolveKnowledgeBaseAccess(
	ctx context.Context,
	kbID string,
	opts types.KnowledgeBaseAccessOptions,
) (*types.KnowledgeBaseAccess, error) {
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return nil, types.ErrKnowledgeBaseAccessNotFound
	}
	requiredPermission := opts.RequiredPermission
	if requiredPermission == "" {
		requiredPermission = types.OrgRoleViewer
	}
	if !requiredPermission.IsValid() {
		return nil, fmt.Errorf("invalid knowledge base permission %q", requiredPermission)
	}

	if err := types.AuthorizeTenantAPIKeyKnowledgeBases(ctx, kbID); err != nil {
		return nil, types.ErrKnowledgeBaseAccessForbidden
	}

	tenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	callerTenantRole := types.TenantRoleFromContext(ctx)

	kb, err := s.GetKnowledgeBaseByID(ctx, kbID)
	if err != nil {
		if stderrors.Is(err, apprepo.ErrKnowledgeBaseNotFound) {
			return nil, types.ErrKnowledgeBaseAccessNotFound
		}
		return nil, err
	}
	if kb == nil {
		return nil, types.ErrKnowledgeBaseAccessNotFound
	}

	ownerType := s.resolveKnowledgeBaseOwnerType(ctx, kb, tenantID)

	if types.IsSystemAdminFromContext(ctx) {
		return &types.KnowledgeBaseAccess{
			KnowledgeBase:     kb,
			EffectiveTenantID: kb.TenantID,
			Permission:        types.OrgRoleAdmin,
			AccessSource:      types.KnowledgeBaseAccessSourceSystemAdmin,
			OwnerType:         ownerType,
		}, nil
	}

	if kb.TenantID == tenantID {
		if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
			return &types.KnowledgeBaseAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceAPIKey,
				OwnerType:         ownerType,
			}, nil
		}
	}

	if kb.TenantID == tenantID {
		// An explicit team-space share is a scoped authorization boundary even
		// when the KB and receiving team belong to the same enterprise tenant.
		// Resolve it before tenant Admin/Owner or creator fallbacks so a viewer
		// share cannot be silently upgraded to write access by the enterprise
		// role. System administrators and API keys remain explicit bypasses above.
		if access, isShared, shareErr := s.resolveSharedKnowledgeBaseAccess(
			ctx,
			kb,
			kbID,
			tenantID,
			callerTenantRole,
			requiredPermission,
			ownerType,
		); shareErr != nil {
			return nil, shareErr
		} else if isShared {
			if access != nil {
				return access, nil
			}
			return nil, types.ErrKnowledgeBaseAccessForbidden
		}

		if callerTenantRole.HasPermission(types.TenantRoleAdmin) {
			return &types.KnowledgeBaseAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceTenantAdmin,
				OwnerType:         ownerType,
			}, nil
		}
		userID, _ := types.UserIDFromContext(ctx)
		if kb.CreatorID != "" && userID != "" && kb.CreatorID == userID {
			return &types.KnowledgeBaseAccess{
				KnowledgeBase:     kb,
				EffectiveTenantID: tenantID,
				Permission:        types.OrgRoleAdmin,
				AccessSource:      types.KnowledgeBaseAccessSourceCreated,
				OwnerType:         ownerType,
			}, nil
		}
	}

	// The account-centred frontend does not switch tenants when an employee
	// opens a knowledge base they created in an enterprise workspace. Keep
	// that flow account-scoped while still checking that the membership is
	// active; leaving the workspace must revoke this shortcut immediately.
	userID, _ := types.UserIDFromContext(ctx)
	if kb.TenantID != tenantID &&
		kb.CreatorID != "" &&
		userID != "" &&
		kb.CreatorID == userID &&
		s.accountCanAccessCreatedKnowledgeBase(ctx, kb, userID, tenantID) {
		return &types.KnowledgeBaseAccess{
			KnowledgeBase:     kb,
			EffectiveTenantID: kb.TenantID,
			Permission:        types.OrgRoleAdmin,
			AccessSource:      types.KnowledgeBaseAccessSourceCreated,
			OwnerType:         ownerType,
		}, nil
	}

	// Personal workspaces are isolated resources. A stale historical share
	// must not turn a personal KB into a cross-tenant resource.
	if kb.TenantID != tenantID &&
		ownerType != nil &&
		*ownerType == types.SpaceTypePersonal {
		return nil, types.ErrKnowledgeBaseAccessForbidden
	}

	if access, isShared, shareErr := s.resolveSharedKnowledgeBaseAccess(
		ctx,
		kb,
		kbID,
		tenantID,
		callerTenantRole,
		requiredPermission,
		ownerType,
	); shareErr != nil {
		return nil, shareErr
	} else if isShared {
		if access != nil {
			return access, nil
		}
		return nil, types.ErrKnowledgeBaseAccessForbidden
	}

	// A personal workspace remains the user's home context, but enterprise
	// memberships are additional account authorizations. Try each active
	// enterprise membership so a shared KB can be opened from the same
	// account without exposing a tenant switcher in the main UI.
	if kb.TenantID != tenantID && s.memberService != nil && userID != "" {
		members, memberErr := s.memberService.ListByUser(ctx, userID)
		if memberErr != nil {
			logger.Warnf(ctx, "[kb_access] failed to list memberships for user %s: %v", userID, memberErr)
		} else {
			seenTenants := map[uint64]struct{}{tenantID: {}}
			for _, member := range members {
				if member == nil ||
					member.TenantID == 0 ||
					member.Status != types.TenantMemberStatusActive ||
					member.TenantID == tenantID {
					continue
				}
				if _, seen := seenTenants[member.TenantID]; seen {
					continue
				}
				seenTenants[member.TenantID] = struct{}{}

				memberTenant := s.tenantForAccountAccess(ctx, member.TenantID)
				if memberTenant == nil {
					continue
				}
				if memberTenant.SpaceType != nil &&
					*memberTenant.SpaceType == types.SpaceTypePersonal {
					continue
				}

				memberCtx := withKnowledgeBaseTenantContext(ctx, memberTenant, member.Role)
				access, isShared, shareErr := s.resolveSharedKnowledgeBaseAccess(
					memberCtx,
					kb,
					kbID,
					member.TenantID,
					member.Role,
					requiredPermission,
					ownerType,
				)
				if shareErr != nil {
					logger.Warnf(ctx, "[kb_access] failed to resolve shared KB %s for tenant %d: %v",
						kbID, member.TenantID, shareErr)
					continue
				}
				if isShared && access != nil {
					return access, nil
				}
			}
		}
	}

	return nil, types.ErrKnowledgeBaseAccessForbidden
}

func (s *knowledgeBaseService) accountCanAccessCreatedKnowledgeBase(
	ctx context.Context,
	kb *types.KnowledgeBase,
	userID string,
	callerTenantID uint64,
) bool {
	if kb == nil || userID == "" || kb.TenantID == 0 {
		return false
	}
	if kb.TenantID == callerTenantID {
		return true
	}

	ownerType := s.resolveKnowledgeBaseOwnerType(ctx, kb, callerTenantID)
	if ownerType != nil && *ownerType == types.SpaceTypePersonal {
		return false
	}
	if s.memberService == nil {
		return false
	}
	member, err := s.memberService.GetMembership(ctx, userID, kb.TenantID)
	return err == nil &&
		member != nil &&
		member.Status == types.TenantMemberStatusActive
}

func (s *knowledgeBaseService) resolveSharedKnowledgeBaseAccess(
	ctx context.Context,
	kb *types.KnowledgeBase,
	kbID string,
	callerTenantID uint64,
	callerTenantRole types.TenantRole,
	requiredPermission types.OrgMemberRole,
	ownerType *types.SpaceType,
) (*types.KnowledgeBaseAccess, bool, error) {
	if s.kbShareService == nil {
		return nil, false, nil
	}
	permission, isShared, permErr := s.kbShareService.CheckTenantKBPermission(
		ctx,
		kbID,
		callerTenantID,
		callerTenantRole,
	)
	if permErr != nil {
		return nil, false, permErr
	}
	if !isShared {
		return nil, false, nil
	}
	if !permission.HasPermission(requiredPermission) {
		return nil, true, nil
	}

	sourceTenantID, srcErr := s.kbShareService.GetKBSourceTenant(ctx, kbID)
	if srcErr != nil || sourceTenantID == 0 {
		logger.Warnf(ctx, "[kb_access] failed to resolve source tenant for shared KB %s: %v", kbID, srcErr)
		if srcErr == nil {
			srcErr = fmt.Errorf("shared knowledge base %s has no source tenant", kbID)
		}
		return nil, true, srcErr
	}
	return &types.KnowledgeBaseAccess{
		KnowledgeBase:     kb,
		EffectiveTenantID: sourceTenantID,
		Permission:        permission,
		AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
		OwnerType:         ownerType,
	}, true, nil
}

func (s *knowledgeBaseService) tenantForAccountAccess(ctx context.Context, tenantID uint64) *types.Tenant {
	if tenantID == 0 {
		return nil
	}
	if s.tenantRepo == nil {
		return &types.Tenant{ID: tenantID}
	}
	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil || tenant == nil {
		logger.Warnf(ctx, "[kb_access] failed to load membership tenant %d: %v", tenantID, err)
		return nil
	}
	return tenant
}

func withKnowledgeBaseTenantContext(
	ctx context.Context,
	tenant *types.Tenant,
	role types.TenantRole,
) context.Context {
	if tenant == nil || tenant.ID == 0 {
		return ctx
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, tenant.ID)
	ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenant)
	ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
	return ctx
}

func (s *knowledgeBaseService) resolveKnowledgeBaseOwnerType(
	ctx context.Context,
	kb *types.KnowledgeBase,
	callerTenantID uint64,
) *types.SpaceType {
	if kb == nil {
		return nil
	}
	if tenant, ok := types.TenantInfoFromContext(ctx); ok &&
		tenant != nil &&
		tenant.ID == kb.TenantID &&
		tenant.SpaceType != nil {
		return tenant.SpaceType
	}
	if kb.TenantID == callerTenantID {
		if tenant, ok := types.TenantInfoFromContext(ctx); ok && tenant != nil && tenant.SpaceType != nil {
			return tenant.SpaceType
		}
	}
	if s.tenantRepo == nil || kb.TenantID == 0 {
		return nil
	}
	tenant, err := s.tenantRepo.GetTenantByID(ctx, kb.TenantID)
	if err != nil || tenant == nil {
		return nil
	}
	return tenant.SpaceType
}
