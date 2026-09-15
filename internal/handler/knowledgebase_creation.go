package handler

import (
	"context"
	"strings"

	apperrors "github.com/Tencent/WeKnora/internal/errors"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

const (
	knowledgeBaseCreationScopePersonal   = "personal"
	knowledgeBaseCreationScopeEnterprise = "enterprise"
)

type knowledgeBaseCreationTarget struct {
	tenant *types.Tenant
	role   types.TenantRole
}

// knowledgeBaseRequestContext keeps the handler testable with Gin contexts
// that use c.Set while production requests use request.Context(). The auth
// middleware populates both surfaces, so this is only a compatibility bridge.
func knowledgeBaseRequestContext(c *gin.Context) context.Context {
	ctx := c.Request.Context()

	if _, ok := types.TenantIDFromContext(ctx); !ok {
		if value, exists := c.Get(types.TenantIDContextKey.String()); exists {
			if tenantID, ok := value.(uint64); ok {
				ctx = context.WithValue(ctx, types.TenantIDContextKey, tenantID)
			}
		}
	}
	if _, ok := types.UserIDFromContext(ctx); !ok {
		if value, exists := c.Get(types.UserIDContextKey.String()); exists {
			if userID, ok := value.(string); ok && strings.TrimSpace(userID) != "" {
				ctx = context.WithValue(ctx, types.UserIDContextKey, userID)
			}
		}
	}
	if _, ok := ctx.Value(types.UserContextKey).(*types.User); !ok {
		if value, exists := c.Get(types.UserContextKey.String()); exists {
			if user, ok := value.(*types.User); ok && user != nil {
				ctx = context.WithValue(ctx, types.UserContextKey, user)
			}
		}
	}
	if _, ok := types.TenantInfoFromContext(ctx); !ok {
		if value, exists := c.Get(types.TenantInfoContextKey.String()); exists {
			if tenant, ok := value.(*types.Tenant); ok && tenant != nil {
				ctx = context.WithValue(ctx, types.TenantInfoContextKey, tenant)
			}
		}
	}
	if _, ok := ctx.Value(types.TenantRoleContextKey).(types.TenantRole); !ok {
		if value, exists := c.Get(types.TenantRoleContextKey.String()); exists {
			if role, ok := value.(types.TenantRole); ok {
				ctx = context.WithValue(ctx, types.TenantRoleContextKey, role)
			}
		}
	}
	if _, ok := ctx.Value(types.SystemAdminContextKey).(bool); !ok {
		if value, exists := c.Get(types.SystemAdminContextKey.String()); exists {
			if systemAdmin, ok := value.(bool); ok {
				ctx = context.WithValue(ctx, types.SystemAdminContextKey, systemAdmin)
			}
		}
	}
	return ctx
}

func (h *KnowledgeBaseHandler) resolveKnowledgeBaseCreationContext(
	ctx context.Context,
	req CreateKnowledgeBaseRequest,
) (context.Context, uint64, error) {
	currentTenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || currentTenantID == 0 {
		return nil, 0, apperrors.NewUnauthorizedError("Unauthorized")
	}

	// API keys are tenant principals, not account principals. Preserve their
	// existing behavior and reject the human-only destination fields.
	if _, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		if strings.TrimSpace(req.Scope) != "" || req.EnterpriseTenantID != 0 {
			return nil, 0, apperrors.NewBadRequestError(
				"workspace selection is not supported for API key requests",
			)
		}
		return ctx, currentTenantID, nil
	}

	userID, ok := types.UserIDFromContext(ctx)
	userID = strings.TrimSpace(userID)
	if !ok || userID == "" || types.IsSyntheticUserID(userID) {
		return nil, 0, apperrors.NewUnauthorizedError("Unauthorized")
	}

	user, _ := ctx.Value(types.UserContextKey).(*types.User)
	if user == nil && h.userService != nil {
		loaded, err := h.userService.GetCurrentUser(ctx)
		if err == nil {
			user = loaded
		}
	}
	if user == nil {
		// This fallback exists for legacy handler integrations that provide
		// only UserID. The authenticated production path always carries the
		// full user object, including the immutable home tenant ID.
		user = &types.User{ID: userID, TenantID: currentTenantID}
	}
	if user.ID == "" {
		user.ID = userID
	}

	scope := strings.ToLower(strings.TrimSpace(req.Scope))
	if scope == "" {
		scope = knowledgeBaseCreationScopePersonal
	}

	var target knowledgeBaseCreationTarget
	var err error
	switch scope {
	case knowledgeBaseCreationScopePersonal:
		if req.EnterpriseTenantID != 0 {
			return nil, 0, apperrors.NewBadRequestError(
				"enterprise_tenant_id requires enterprise scope",
			)
		}
		target, err = h.resolvePersonalKnowledgeBaseTarget(
			ctx,
			user,
			userID,
			currentTenantID,
		)
		if err != nil {
			return nil, 0, err
		}
	case knowledgeBaseCreationScopeEnterprise:
		target, err = h.resolveEnterpriseKnowledgeBaseTarget(
			ctx,
			userID,
			currentTenantID,
			req.EnterpriseTenantID,
		)
		if err != nil {
			return nil, 0, err
		}
	default:
		return nil, 0, apperrors.NewBadRequestError(
			"scope must be personal or enterprise",
		)
	}

	if target.tenant == nil || target.tenant.ID == 0 {
		return nil, 0, apperrors.NewInternalServerError(
			"knowledge base workspace could not be resolved",
		)
	}
	return withKnowledgeBaseCreationTenantContext(ctx, target.tenant, target.role), target.tenant.ID, nil
}

func (h *KnowledgeBaseHandler) resolvePersonalKnowledgeBaseTarget(
	ctx context.Context,
	user *types.User,
	userID string,
	currentTenantID uint64,
) (knowledgeBaseCreationTarget, error) {
	homeTenantID := user.TenantID
	if homeTenantID == 0 {
		homeTenantID = currentTenantID
	}
	tenant, err := h.loadKnowledgeBaseCreationTenant(ctx, homeTenantID)
	if err != nil {
		return knowledgeBaseCreationTarget{}, apperrors.NewInternalServerError(
			"failed to load personal workspace",
		).WithDetails(err.Error())
	}

	role := types.TenantRoleFromContext(ctx)
	if h.tenantMemberService != nil {
		member, membershipErr := h.tenantMemberService.GetMembership(ctx, userID, homeTenantID)
		if membershipErr != nil {
			return knowledgeBaseCreationTarget{}, apperrors.NewInternalServerError(
				"failed to verify personal workspace membership",
			).WithDetails(membershipErr.Error())
		}
		if member == nil || member.Status != types.TenantMemberStatusActive {
			return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
				"personal workspace membership is not active",
			)
		}
		role = member.Role
	}
	if types.IsSystemAdminFromContext(ctx) {
		role = types.TenantRoleAdmin
	}
	if !types.IsSystemAdminFromContext(ctx) &&
		!role.HasPermission(types.TenantRoleContributor) {
		return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
			"no permission to create a personal knowledge base",
		)
	}
	return knowledgeBaseCreationTarget{tenant: tenant, role: role}, nil
}

func (h *KnowledgeBaseHandler) resolveEnterpriseKnowledgeBaseTarget(
	ctx context.Context,
	userID string,
	currentTenantID uint64,
	requestedTenantID uint64,
) (knowledgeBaseCreationTarget, error) {
	if types.IsSystemAdminFromContext(ctx) {
		if requestedTenantID != 0 {
			tenant, err := h.loadKnowledgeBaseCreationTenant(ctx, requestedTenantID)
			if err != nil {
				return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
					"enterprise workspace is not available",
				)
			}
			if !isEnterpriseKnowledgeBaseTenant(tenant) {
				return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
					"enterprise workspace is not available",
				)
			}
			return knowledgeBaseCreationTarget{tenant: tenant, role: types.TenantRoleAdmin}, nil
		}

		currentTenant, err := h.loadKnowledgeBaseCreationTenant(ctx, currentTenantID)
		if err == nil && isEnterpriseKnowledgeBaseTenant(currentTenant) {
			return knowledgeBaseCreationTarget{tenant: currentTenant, role: types.TenantRoleAdmin}, nil
		}
	}

	if h.tenantMemberService == nil {
		tenant, err := h.loadKnowledgeBaseCreationTenant(ctx, currentTenantID)
		if err != nil || !isEnterpriseKnowledgeBaseTenant(tenant) {
			return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
				"no enterprise workspace creation permission",
			)
		}
		role := types.TenantRoleFromContext(ctx)
		if requestedTenantID != 0 && requestedTenantID != currentTenantID {
			return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
				"enterprise workspace is not available",
			)
		}
		if !role.HasPermission(types.TenantRoleContributor) &&
			!types.IsSystemAdminFromContext(ctx) {
			return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
				"no enterprise workspace creation permission",
			)
		}
		return knowledgeBaseCreationTarget{tenant: tenant, role: role}, nil
	}

	members, err := h.tenantMemberService.ListByUser(ctx, userID)
	if err != nil {
		return knowledgeBaseCreationTarget{}, apperrors.NewInternalServerError(
			"failed to load enterprise workspace memberships",
		).WithDetails(err.Error())
	}

	candidates := make(map[uint64]knowledgeBaseCreationTarget)
	for _, member := range members {
		if member == nil ||
			member.TenantID == 0 ||
			member.Status != types.TenantMemberStatusActive ||
			!member.Role.HasPermission(types.TenantRoleContributor) {
			continue
		}
		tenant, tenantErr := h.loadKnowledgeBaseCreationTenant(ctx, member.TenantID)
		if tenantErr != nil {
			logger.Warnf(ctx,
				"[kb.create] skip membership tenant %d while resolving enterprise targets: %v",
				member.TenantID, tenantErr)
			continue
		}
		if !isEnterpriseKnowledgeBaseTenant(tenant) {
			continue
		}
		candidates[member.TenantID] = knowledgeBaseCreationTarget{
			tenant: tenant,
			role:   member.Role,
		}
	}

	if requestedTenantID != 0 {
		target, ok := candidates[requestedTenantID]
		if !ok {
			return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
				"no permission to create a knowledge base in this enterprise workspace",
			)
		}
		return target, nil
	}

	switch len(candidates) {
	case 0:
		return knowledgeBaseCreationTarget{}, apperrors.NewForbiddenError(
			"no enterprise workspace creation permission",
		)
	case 1:
		for _, target := range candidates {
			return target, nil
		}
	default:
		return knowledgeBaseCreationTarget{}, apperrors.NewConflictError(
			"select an enterprise workspace before creating the knowledge base",
		)
	}
	return knowledgeBaseCreationTarget{}, apperrors.NewInternalServerError(
		"enterprise workspace target could not be resolved",
	)
}

func (h *KnowledgeBaseHandler) loadKnowledgeBaseCreationTenant(
	ctx context.Context,
	tenantID uint64,
) (*types.Tenant, error) {
	if tenantID == 0 {
		return nil, apperrors.NewBadRequestError("workspace ID is required")
	}
	if h.tenantService != nil {
		return h.tenantService.GetTenantByID(ctx, tenantID)
	}
	if tenant, ok := types.TenantInfoFromContext(ctx); ok && tenant != nil && tenant.ID == tenantID {
		return tenant, nil
	}
	return &types.Tenant{ID: tenantID}, nil
}

func isEnterpriseKnowledgeBaseTenant(tenant *types.Tenant) bool {
	return tenant != nil &&
		tenant.SpaceType != nil &&
		*tenant.SpaceType == types.SpaceTypeOrganization
}

func withKnowledgeBaseCreationTenantContext(
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
