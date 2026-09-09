package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// resolveSuggestionReadableKnowledgeBaseTenant returns the tenant whose rows
// should be queried for kb, but only when the caller is allowed to see that KB
// in suggestion surfaces.
func (s *customAgentService) resolveSuggestionReadableKnowledgeBaseTenant(
	ctx context.Context,
	callerTenantID uint64,
	kb *types.KnowledgeBase,
) (uint64, bool) {
	if kb == nil || kb.ID == "" {
		return 0, false
	}

	if scope, ok := types.TenantAPIKeyScopeFromContext(ctx); ok {
		if !scope.AllowsKnowledgeBase(kb.ID) {
			logger.Warnf(ctx, "Dropping KB %s from suggestions: outside API key scope", kb.ID)
			return 0, false
		}
		if scope.IsKnowledgeBaseRestricted() {
			if kb.TenantID == callerTenantID {
				return kb.TenantID, true
			}
			return 0, false
		}
		if kb.TenantID == callerTenantID {
			return kb.TenantID, true
		}
	}

	if types.IsSystemAdminFromContext(ctx) {
		return kb.TenantID, true
	}
	if types.TenantRoleFromContext(ctx).HasPermission(types.TenantRoleAdmin) && kb.TenantID == callerTenantID {
		return kb.TenantID, true
	}

	if userID, ok := types.UserIDFromContext(ctx); ok && userID != "" && kb.CreatorID != "" && kb.CreatorID == userID && kb.TenantID == callerTenantID {
		return kb.TenantID, true
	}

	if s.kbShareService != nil {
		hasAccess, err := s.kbShareService.HasTenantKBPermission(
			ctx, kb.ID, callerTenantID, types.TenantRoleFromContext(ctx), types.OrgRoleViewer,
		)
		if err != nil {
			logger.Warnf(ctx, "Failed to resolve shared KB permission for suggestions, kb_id=%s: %v", kb.ID, err)
		} else if hasAccess {
			return kb.TenantID, true
		}
	}

	logger.Warnf(ctx, "Dropping KB %s from suggestions: caller cannot view it", kb.ID)
	return 0, false
}

func (s *customAgentService) filterReadableSuggestionKnowledgeBaseIDs(
	ctx context.Context,
	callerTenantID uint64,
	kbIDs []string,
) []string {
	if len(kbIDs) == 0 || s.kbService == nil {
		return nil
	}
	kbIDs = uniqueNonEmptyStrings(kbIDs)
	kbs, err := s.kbService.GetKnowledgeBasesByIDsOnly(ctx, kbIDs)
	if err != nil {
		logger.Warnf(ctx, "Failed to fetch knowledge bases for suggestion visibility: %v", err)
		return nil
	}
	kbByID := make(map[string]*types.KnowledgeBase, len(kbs))
	for _, kb := range kbs {
		if kb != nil {
			kbByID[kb.ID] = kb
		}
	}
	filtered := make([]string, 0, len(kbIDs))
	for _, kbID := range kbIDs {
		tenantID, ok := s.resolveSuggestionReadableKnowledgeBaseTenant(ctx, callerTenantID, kbByID[kbID])
		if !ok {
			continue
		}
		if tenantID == 0 {
			continue
		}
		filtered = append(filtered, kbID)
	}
	return filtered
}

func (s *customAgentService) filterReadableSuggestionKnowledgeScope(
	ctx context.Context,
	callerTenantID uint64,
	knowledgeIDs []string,
) ([]string, []string) {
	if len(knowledgeIDs) == 0 || s.knowledgeRepo == nil || s.kbService == nil {
		return nil, nil
	}
	knowledgeIDs = uniqueNonEmptyStrings(knowledgeIDs)
	filtered := make([]string, 0, len(knowledgeIDs))
	kbIDs := make([]string, 0, len(knowledgeIDs))
	seenKBIDs := make(map[string]bool)
	for _, knowledgeID := range knowledgeIDs {
		knowledge, err := s.knowledgeRepo.GetKnowledgeByIDOnly(ctx, knowledgeID)
		if err != nil || knowledge == nil || knowledge.KnowledgeBaseID == "" {
			continue
		}
		kb, err := s.kbService.GetKnowledgeBaseByIDOnly(ctx, knowledge.KnowledgeBaseID)
		if err != nil || kb == nil {
			continue
		}
		if _, ok := s.resolveSuggestionReadableKnowledgeBaseTenant(ctx, callerTenantID, kb); !ok {
			continue
		}
		filtered = append(filtered, knowledgeID)
		if !seenKBIDs[kb.ID] {
			seenKBIDs[kb.ID] = true
			kbIDs = append(kbIDs, kb.ID)
		}
	}
	return uniqueNonEmptyStrings(filtered), uniqueNonEmptyStrings(kbIDs)
}

func (s *customAgentService) filterReadableSuggestionKnowledgeIDs(
	ctx context.Context,
	callerTenantID uint64,
	knowledgeIDs []string,
) []string {
	filtered, _ := s.filterReadableSuggestionKnowledgeScope(ctx, callerTenantID, knowledgeIDs)
	return filtered
}
