package service

import (
	"context"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// fetchKnowledgeData gets knowledge data in batch.
func (s *knowledgeBaseService) fetchKnowledgeData(ctx context.Context,
	tenantID uint64,
	knowledgeIDs []string,
) (map[string]*types.Knowledge, error) {
	knowledges, err := s.kgRepo.GetKnowledgeBatch(ctx, tenantID, knowledgeIDs)
	if err != nil {
		logger.ErrorWithFields(ctx, err, map[string]interface{}{
			"tenant_id":     tenantID,
			"knowledge_ids": knowledgeIDs,
		})
		return nil, err
	}

	knowledgeMap := make(map[string]*types.Knowledge, len(knowledges))
	for _, knowledge := range knowledges {
		knowledgeMap[knowledge.ID] = knowledge
	}

	return knowledgeMap, nil
}

// fetchKnowledgeDataWithShared gets knowledge data in batch, including knowledge
// from shared KBs the user has access to.
func (s *knowledgeBaseService) fetchKnowledgeDataWithShared(ctx context.Context,
	tenantID uint64,
	knowledgeIDs []string,
) (map[string]*types.Knowledge, error) {
	knowledgeMap, err := s.fetchKnowledgeData(ctx, tenantID, knowledgeIDs)
	if err != nil {
		return nil, err
	}

	missingIDs := s.findMissingIDs(knowledgeIDs, func(id string) bool {
		return knowledgeMap[id] != nil
	})
	if len(missingIDs) == 0 {
		return knowledgeMap, nil
	}
	logger.Infof(ctx, "[fetchKnowledgeDataWithShared] %d knowledge IDs not found in current tenant, attempting shared KB lookup", len(missingIDs))

	logger.Infof(ctx, "[fetchKnowledgeDataWithShared] Looking up %d missing knowledge IDs with account-level access checks", len(missingIDs))
	callerTenantRole := types.TenantRoleFromContext(ctx)
	for _, id := range missingIDs {
		k, err := s.kgRepo.GetKnowledgeByIDOnly(ctx, id)
		if err != nil || k == nil || k.KnowledgeBaseID == "" {
			logger.Debugf(ctx, "[fetchKnowledgeDataWithShared] Knowledge %s not found or has no KB", id)
			continue
		}
		hasPermission, err := s.hasSearchReadAccess(
			ctx, k.KnowledgeBaseID, tenantID, callerTenantRole,
		)
		if err != nil {
			logger.Debugf(ctx, "[fetchKnowledgeDataWithShared] Permission check error for KB %s: %v", k.KnowledgeBaseID, err)
			continue
		}
		if !hasPermission {
			logger.Debugf(ctx, "[fetchKnowledgeDataWithShared] No permission for KB %s", k.KnowledgeBaseID)
			continue
		}
		logger.Debugf(ctx, "[fetchKnowledgeDataWithShared] Found shared knowledge %s in KB %s", id, k.KnowledgeBaseID)
		knowledgeMap[k.ID] = k
	}

	logger.Infof(ctx, "[fetchKnowledgeDataWithShared] After shared lookup, total knowledge found: %d", len(knowledgeMap))
	return knowledgeMap, nil
}

// listChunksByIDWithShared fetches chunks by IDs, including chunks from shared KBs the user has access to.
func (s *knowledgeBaseService) listChunksByIDWithShared(ctx context.Context,
	tenantID uint64,
	chunkIDs []string,
) ([]*types.Chunk, error) {
	chunks, err := s.chunkRepo.ListChunksByID(ctx, tenantID, chunkIDs)
	if err != nil {
		return nil, err
	}

	foundSet := make(map[string]bool, len(chunks))
	for _, c := range chunks {
		if c != nil {
			foundSet[c.ID] = true
		}
	}

	missing := s.findMissingIDs(chunkIDs, func(id string) bool {
		return foundSet[id]
	})
	if len(missing) == 0 {
		return chunks, nil
	}
	logger.Infof(ctx, "[listChunksByIDWithShared] %d chunks not found in current tenant, attempting shared KB lookup", len(missing))

	logger.Infof(ctx, "[listChunksByIDWithShared] Looking up %d missing chunks with account-level access checks", len(missing))
	callerTenantRole := types.TenantRoleFromContext(ctx)
	crossChunks, err := s.chunkRepo.ListChunksByIDOnly(ctx, missing)
	if err != nil {
		logger.Warnf(ctx, "[listChunksByIDWithShared] Failed to fetch chunks by ID only: %v", err)
		return chunks, nil
	}
	logger.Infof(ctx, "[listChunksByIDWithShared] Found %d chunks without tenant filter", len(crossChunks))

	for _, c := range crossChunks {
		if c == nil || c.KnowledgeBaseID == "" {
			continue
		}
		hasPermission, err := s.hasSearchReadAccess(
			ctx, c.KnowledgeBaseID, tenantID, callerTenantRole,
		)
		if err != nil {
			logger.Debugf(ctx, "[listChunksByIDWithShared] Permission check error for KB %s: %v", c.KnowledgeBaseID, err)
			continue
		}
		if !hasPermission {
			logger.Debugf(ctx, "[listChunksByIDWithShared] No permission for KB %s", c.KnowledgeBaseID)
			continue
		}
		chunks = append(chunks, c)
	}

	logger.Infof(ctx, "[listChunksByIDWithShared] After shared lookup, total chunks: %d", len(chunks))
	return chunks, nil
}

func (s *knowledgeBaseService) hasSearchReadAccess(
	ctx context.Context,
	kbID string,
	tenantID uint64,
	callerTenantRole types.TenantRole,
) (bool, error) {
	if s.repo != nil {
		access, err := s.ResolveKnowledgeBaseAccess(
			ctx,
			kbID,
			types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
		)
		if err == nil && access != nil {
			return true, nil
		}
		if err != nil {
			logger.Debugf(ctx, "[search_read_access] account-level access denied for KB %s: %v", kbID, err)
		}
	}

	if s.kbShareService == nil {
		return false, nil
	}
	return s.kbShareService.HasTenantKBPermission(
		ctx,
		kbID,
		tenantID,
		callerTenantRole,
		types.OrgRoleViewer,
	)
}

// findMissingIDs returns IDs from the input slice that are not found by the exists predicate.
func (s *knowledgeBaseService) findMissingIDs(ids []string, exists func(string) bool) []string {
	var missing []string
	for _, id := range ids {
		if !exists(id) {
			missing = append(missing, id)
		}
	}
	return missing
}

// extractUserID extracts the user ID from context, returning ("", false) if not found.
func (s *knowledgeBaseService) extractUserID(ctx context.Context) (string, bool) {
	userIDVal := ctx.Value(types.UserIDContextKey)
	if userIDVal == nil {
		return "", false
	}
	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		return "", false
	}
	return userID, true
}
