package service

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// resolveReadableKnowledgeBaseTenant returns the tenant whose chunks should be
// queried for kbID, but only when the current caller has read access to that KB.
func (s *sessionService) resolveReadableKnowledgeBaseTenant(
	ctx context.Context,
	retrievalTenantID uint64,
	kbID string,
	kb *types.KnowledgeBase,
) (uint64, bool) {
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return 0, false
	}

	if scope, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		if !scope.AllowsKnowledgeBase(kbID) {
			logger.Warnf(ctx, "Dropping KB %s from search targets: outside API key scope", kbID)
			return 0, false
		}
		if kb == nil {
			return retrievalTenantID, true
		}
		if kb.TenantID == retrievalTenantID {
			return kb.TenantID, true
		}
		logger.Warnf(ctx, "Dropping KB %s from search targets: API key cannot search cross-tenant KB", kbID)
		return 0, false
	}

	if kb == nil {
		if callerIsTenantAdmin(ctx) {
			return retrievalTenantID, true
		}
		logger.Warnf(ctx, "Dropping KB %s from search targets: knowledge base metadata not found", kbID)
		return 0, false
	}

	if isSharedAgentRuntime(ctx, retrievalTenantID) && kb.TenantID == retrievalTenantID {
		return kb.TenantID, true
	}

	if kb.TenantID == retrievalTenantID && callerIsTenantAdmin(ctx) {
		return kb.TenantID, true
	}

	callerTenantID := callerTenantIDForSearchVisibility(ctx, retrievalTenantID)
	callerTenantRole := types.TenantRoleFromContext(ctx)
	if s.kbShareService != nil && callerTenantID != 0 {
		hasAccess, err := s.kbShareService.HasTenantKBPermission(
			ctx, kbID, callerTenantID, callerTenantRole, types.OrgRoleViewer,
		)
		if err != nil {
			logger.Warnf(ctx, "Failed to resolve shared KB permission for %s: %v", kbID, err)
		} else if hasAccess {
			return kb.TenantID, true
		}
	}

	if kb.TenantID != retrievalTenantID {
		logger.Warnf(ctx, "Dropping KB %s from search targets: no shared read permission", kbID)
		return 0, false
	}

	userID, ok := types.UserIDFromContext(ctx)
	if ok && userID != "" && kb.CreatorID != "" && kb.CreatorID == userID {
		return kb.TenantID, true
	}

	logger.Warnf(ctx, "Dropping KB %s from search targets: caller is not creator/admin/shared reader", kbID)
	return 0, false
}

func callerIsTenantAdmin(ctx context.Context) bool {
	return types.IsSystemAdminFromContext(ctx) ||
		types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin)
}

func isSharedAgentRuntime(ctx context.Context, retrievalTenantID uint64) bool {
	sessionTenantID, ok := types.SessionTenantIDFromContext(ctx)
	return ok && sessionTenantID != 0 && retrievalTenantID != 0 && sessionTenantID != retrievalTenantID
}

func callerTenantIDForSearchVisibility(ctx context.Context, retrievalTenantID uint64) uint64 {
	if isSharedAgentRuntime(ctx, retrievalTenantID) {
		if sessionTenantID, ok := types.SessionTenantIDFromContext(ctx); ok {
			return sessionTenantID
		}
	}
	if tenantID, ok := types.TenantIDFromContext(ctx); ok {
		return tenantID
	}
	return retrievalTenantID
}

func filterKnowledgeBaseIDsBySearchTargets(kbIDs []string, searchTargets types.SearchTargets) []string {
	if len(kbIDs) == 0 || len(searchTargets) == 0 {
		return nil
	}
	allowed := make(map[string]bool, len(searchTargets))
	for _, target := range searchTargets {
		if target != nil && target.KnowledgeBaseID != "" {
			allowed[target.KnowledgeBaseID] = true
		}
	}
	filtered := make([]string, 0, len(kbIDs))
	for _, kbID := range kbIDs {
		if kbID != "" && allowed[kbID] {
			filtered = append(filtered, kbID)
		}
	}
	return uniqueNonEmptyStrings(filtered)
}

func filterKnowledgeIDsBySearchTargets(knowledgeIDs []string, searchTargets types.SearchTargets) []string {
	if len(knowledgeIDs) == 0 || len(searchTargets) == 0 {
		return nil
	}
	allowed := make(map[string]bool)
	for _, knowledgeID := range searchTargets.GetAllKnowledgeIDs() {
		allowed[knowledgeID] = true
	}
	filtered := make([]string, 0, len(knowledgeIDs))
	for _, knowledgeID := range knowledgeIDs {
		if knowledgeID != "" && allowed[knowledgeID] {
			filtered = append(filtered, knowledgeID)
		}
	}
	return uniqueNonEmptyStrings(filtered)
}
