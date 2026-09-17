package service

import (
	"context"
	stderrors "errors"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// ListMyKnowledgeBases returns the account-centred created/shared/subscribed
// projection for the knowledge-base home page. Subscription rows are only
// shortcuts: every subscribed KB is re-checked through ResolveKnowledgeBaseAccess.
func (s *knowledgeBaseService) ListMyKnowledgeBases(ctx context.Context) (*types.MyKnowledgeBaseList, error) {
	userID, ok := types.UserIDFromContext(ctx)
	userID = strings.TrimSpace(userID)
	if !ok || userID == "" || types.IsSyntheticUserID(userID) {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	callerTenantID, ok := types.TenantIDFromContext(ctx)
	if !ok || callerTenantID == 0 {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	callerTenantRole := types.TenantRoleFromContext(ctx)

	result := &types.MyKnowledgeBaseList{
		Created:    []*types.MyKnowledgeBaseListItem{},
		Shared:     []*types.MyKnowledgeBaseListItem{},
		Subscribed: []*types.MyKnowledgeBaseListItem{},
	}

	subscriptions, err := s.repo.ListActiveKnowledgeBaseSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	subscriptionsByKBID := activeSubscriptionsByKBID(subscriptions)

	created, err := s.repo.ListKnowledgeBasesByCreatorID(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, kb := range created {
		if kb == nil || kb.IsTemporary {
			continue
		}
		if !types.IsSystemAdminFromContext(ctx) &&
			!s.accountCanAccessCreatedKnowledgeBase(ctx, kb, userID, callerTenantID) {
			continue
		}
		result.Created = append(result.Created, s.myKnowledgeBaseItemFromCreated(
			ctx,
			kb,
			callerTenantID,
			subscriptionsByKBID,
		))
	}

	if s.kbShareService != nil {
		shared, err := s.listAccountSharedKnowledgeBases(
			ctx,
			userID,
			callerTenantID,
			callerTenantRole,
		)
		if err != nil {
			return nil, err
		}
		for _, info := range shared {
			if info == nil || info.KnowledgeBase == nil || info.KnowledgeBase.IsTemporary {
				continue
			}
			result.Shared = append(result.Shared, s.myKnowledgeBaseItemFromShared(
				ctx,
				info,
				callerTenantID,
				subscriptionsByKBID,
			))
		}
	}

	result.Subscribed = s.listAccessibleSubscribedKnowledgeBases(ctx, subscriptions)

	s.finalizeMyKnowledgeBaseItems(ctx, userID, result.Created)
	s.finalizeMyKnowledgeBaseItems(ctx, userID, result.Shared)
	s.finalizeMyKnowledgeBaseItems(ctx, userID, result.Subscribed)
	return result, nil
}

type accountKnowledgeBaseTenant struct {
	tenant *types.Tenant
	role   types.TenantRole
}

func (s *knowledgeBaseService) listAccountKnowledgeBaseTenants(
	ctx context.Context,
	userID string,
	callerTenantID uint64,
	callerTenantRole types.TenantRole,
) ([]accountKnowledgeBaseTenant, error) {
	if callerTenantID == 0 {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}

	tenants := make([]accountKnowledgeBaseTenant, 0)
	seen := make(map[uint64]struct{})
	currentTenant, _ := types.TenantInfoFromContext(ctx)
	if currentTenant == nil || currentTenant.ID != callerTenantID {
		currentTenant = &types.Tenant{ID: callerTenantID}
	}
	// A personal home is not a collaboration target. Keep legacy /
	// unclassified current contexts for backward compatibility while the
	// tenant metadata rollout is still incomplete.
	if currentTenant.SpaceType == nil || *currentTenant.SpaceType != types.SpaceTypePersonal {
		tenants = append(tenants, accountKnowledgeBaseTenant{
			tenant: currentTenant,
			role:   callerTenantRole,
		})
		seen[callerTenantID] = struct{}{}
	}

	if s.memberService == nil || userID == "" {
		if len(tenants) == 0 {
			return []accountKnowledgeBaseTenant{{
				tenant: &types.Tenant{ID: callerTenantID},
				role:   callerTenantRole,
			}}, nil
		}
		return tenants, nil
	}

	members, err := s.memberService.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, member := range members {
		if member == nil ||
			member.TenantID == 0 ||
			member.Status != types.TenantMemberStatusActive {
			continue
		}
		if _, exists := seen[member.TenantID]; exists {
			continue
		}
		tenant := s.tenantForAccountAccess(ctx, member.TenantID)
		if tenant == nil {
			continue
		}
		if tenant.SpaceType != nil && *tenant.SpaceType == types.SpaceTypePersonal {
			continue
		}
		tenants = append(tenants, accountKnowledgeBaseTenant{
			tenant: tenant,
			role:   member.Role,
		})
		seen[member.TenantID] = struct{}{}
	}
	return tenants, nil
}

func (s *knowledgeBaseService) listAccountSharedKnowledgeBases(
	ctx context.Context,
	userID string,
	callerTenantID uint64,
	callerTenantRole types.TenantRole,
) ([]*types.SharedKnowledgeBaseInfo, error) {
	accountTenants, err := s.listAccountKnowledgeBaseTenants(
		ctx,
		userID,
		callerTenantID,
		callerTenantRole,
	)
	if err != nil {
		return nil, err
	}

	byKBID := make(map[string]*types.SharedKnowledgeBaseInfo)
	for _, accountTenant := range accountTenants {
		if accountTenant.tenant == nil || accountTenant.tenant.ID == 0 {
			continue
		}
		scopedCtx := withKnowledgeBaseTenantContext(
			ctx,
			accountTenant.tenant,
			accountTenant.role,
		)
		shared, err := s.kbShareService.ListSharedKnowledgeBases(
			scopedCtx,
			accountTenant.tenant.ID,
			accountTenant.role,
		)
		if err != nil {
			return nil, err
		}
		for _, info := range shared {
			if info == nil || info.KnowledgeBase == nil || info.KnowledgeBase.ID == "" {
				continue
			}
			existing, exists := byKBID[info.KnowledgeBase.ID]
			if !exists ||
				(types.MaxOrgRole(existing.Permission, info.Permission) == info.Permission &&
					info.Permission != existing.Permission) {
				byKBID[info.KnowledgeBase.ID] = info
			}
		}
	}

	out := make([]*types.SharedKnowledgeBaseInfo, 0, len(byKBID))
	for _, info := range byKBID {
		out = append(out, info)
	}
	return out, nil
}

func activeSubscriptionsByKBID(
	subscriptions []*types.KnowledgeBaseSubscription,
) map[string]*types.KnowledgeBaseSubscription {
	out := make(map[string]*types.KnowledgeBaseSubscription, len(subscriptions))
	for _, sub := range subscriptions {
		if sub == nil || sub.KnowledgeBaseID == "" {
			continue
		}
		if _, exists := out[sub.KnowledgeBaseID]; !exists {
			out[sub.KnowledgeBaseID] = sub
		}
	}
	return out
}

func (s *knowledgeBaseService) myKnowledgeBaseItemFromCreated(
	ctx context.Context,
	kb *types.KnowledgeBase,
	callerTenantID uint64,
	subscriptions map[string]*types.KnowledgeBaseSubscription,
) *types.MyKnowledgeBaseListItem {
	s.enrichMyKnowledgeBaseCounts(ctx, kb)
	item := &types.MyKnowledgeBaseListItem{
		KnowledgeBase:     kb,
		EffectiveTenantID: kb.TenantID,
		Permission:        types.OrgRoleAdmin,
		AccessSource:      types.KnowledgeBaseAccessSourceCreated,
		OwnerType:         s.resolveKnowledgeBaseOwnerType(ctx, kb, callerTenantID),
	}
	applySubscriptionToMyKnowledgeBaseItem(item, subscriptions[kb.ID])
	return item
}

func (s *knowledgeBaseService) myKnowledgeBaseItemFromShared(
	ctx context.Context,
	info *types.SharedKnowledgeBaseInfo,
	callerTenantID uint64,
	subscriptions map[string]*types.KnowledgeBaseSubscription,
) *types.MyKnowledgeBaseListItem {
	kb := info.KnowledgeBase
	s.enrichMyKnowledgeBaseCounts(ctx, kb)
	sourceTenantID := info.SourceTenantID
	if sourceTenantID == 0 && kb != nil {
		sourceTenantID = kb.TenantID
	}
	sharedAt := info.SharedAt
	item := &types.MyKnowledgeBaseListItem{
		KnowledgeBase:     kb,
		EffectiveTenantID: sourceTenantID,
		Permission:        info.Permission,
		AccessSource:      types.KnowledgeBaseAccessSourceSharedSpace,
		OwnerType:         s.resolveKnowledgeBaseOwnerType(ctx, kb, callerTenantID),
		OrganizationID:    info.OrganizationID,
		OrgName:           info.OrgName,
		ShareID:           info.ShareID,
		SharedAt:          &sharedAt,
	}
	applySubscriptionToMyKnowledgeBaseItem(item, subscriptions[kb.ID])
	return item
}

func (s *knowledgeBaseService) listAccessibleSubscribedKnowledgeBases(
	ctx context.Context,
	subscriptions []*types.KnowledgeBaseSubscription,
) []*types.MyKnowledgeBaseListItem {
	if len(subscriptions) == 0 {
		return []*types.MyKnowledgeBaseListItem{}
	}
	items := make([]*types.MyKnowledgeBaseListItem, 0, len(subscriptions))
	seen := make(map[string]struct{}, len(subscriptions))
	for _, sub := range subscriptions {
		if sub == nil || sub.KnowledgeBaseID == "" {
			continue
		}
		if _, ok := seen[sub.KnowledgeBaseID]; ok {
			continue
		}
		seen[sub.KnowledgeBaseID] = struct{}{}

		access, err := s.ResolveKnowledgeBaseAccess(ctx, sub.KnowledgeBaseID, types.KnowledgeBaseAccessOptions{
			RequiredPermission: types.OrgRoleViewer,
		})
		if err != nil {
			if stderrors.Is(err, types.ErrKnowledgeBaseAccessForbidden) ||
				stderrors.Is(err, types.ErrKnowledgeBaseAccessNotFound) {
				continue
			}
			logger.Warnf(ctx, "[kb.my] skip subscribed KB %s: %v", sub.KnowledgeBaseID, err)
			continue
		}
		if access == nil || access.KnowledgeBase == nil || access.KnowledgeBase.IsTemporary {
			continue
		}

		s.enrichMyKnowledgeBaseCounts(ctx, access.KnowledgeBase)
		subscribedAt := sub.CreatedAt
		item := &types.MyKnowledgeBaseListItem{
			KnowledgeBase:     access.KnowledgeBase,
			EffectiveTenantID: access.EffectiveTenantID,
			Permission:        access.Permission,
			AccessSource:      access.AccessSource,
			OwnerType:         access.OwnerType,
			OrganizationID:    access.OrganizationID,
			SharingScope:      access.SharingScope,
			IsSubscribed:      true,
			SubscriptionID:    sub.ID,
			SubscribedAt:      &subscribedAt,
		}
		items = append(items, item)
	}
	return items
}

func applySubscriptionToMyKnowledgeBaseItem(
	item *types.MyKnowledgeBaseListItem,
	sub *types.KnowledgeBaseSubscription,
) {
	if item == nil || sub == nil {
		return
	}
	item.IsSubscribed = true
	item.SubscriptionID = sub.ID
	subscribedAt := sub.CreatedAt
	item.SubscribedAt = &subscribedAt
}

func (s *knowledgeBaseService) enrichMyKnowledgeBaseCounts(ctx context.Context, kb *types.KnowledgeBase) {
	if kb == nil {
		return
	}
	kb.EnsureDefaults()
	switch kb.Type {
	case types.KnowledgeBaseTypeDocument:
		if s.kgRepo != nil {
			if count, err := s.kgRepo.CountKnowledgeByKnowledgeBaseID(ctx, kb.TenantID, kb.ID); err == nil {
				kb.KnowledgeCount = count
			}
		}
	case types.KnowledgeBaseTypeFAQ:
		if s.chunkRepo != nil {
			if count, err := s.chunkRepo.CountChunksByKnowledgeBaseID(ctx, kb.TenantID, kb.ID); err == nil {
				kb.ChunkCount = count
			}
		}
	}
	if s.kgRepo != nil {
		if processingCount, err := s.kgRepo.CountKnowledgeByStatus(ctx, kb.TenantID, kb.ID, []string{"pending", "processing"}); err == nil {
			kb.IsProcessing = processingCount > 0
			kb.ProcessingCount = processingCount
		}
	}
}

func (s *knowledgeBaseService) finalizeMyKnowledgeBaseItems(
	ctx context.Context,
	userID string,
	items []*types.MyKnowledgeBaseListItem,
) {
	if len(items) == 0 || userID == "" {
		return
	}
	s.stampUserKBPinsByKnowledgeBaseTenant(ctx, userID, items)
	sort.SliceStable(items, func(i, j int) bool {
		return myKnowledgeBaseListItemLess(items[i], items[j])
	})
}

func (s *knowledgeBaseService) stampUserKBPinsByKnowledgeBaseTenant(
	ctx context.Context,
	userID string,
	items []*types.MyKnowledgeBaseListItem,
) {
	if s.repo == nil {
		return
	}
	tenantIDs := make(map[uint64]struct{})
	for _, item := range items {
		if item == nil || item.KnowledgeBase == nil || item.KnowledgeBase.TenantID == 0 {
			continue
		}
		tenantIDs[item.KnowledgeBase.TenantID] = struct{}{}
	}
	for tenantID := range tenantIDs {
		pins, err := s.repo.ListUserKBPinIDs(ctx, tenantID, userID)
		if err != nil {
			logger.Warnf(ctx, "stampUserKBPinsByKnowledgeBaseTenant: tenant=%d user=%s: %v", tenantID, userID, err)
			continue
		}
		if len(pins) == 0 {
			continue
		}
		for _, item := range items {
			if item == nil || item.KnowledgeBase == nil || item.KnowledgeBase.TenantID != tenantID {
				continue
			}
			if pinnedAt, ok := pins[item.KnowledgeBase.ID]; ok {
				t := pinnedAt
				item.KnowledgeBase.IsPinned = true
				item.KnowledgeBase.PinnedAt = &t
			}
		}
	}
}

func myKnowledgeBaseListItemLess(a, b *types.MyKnowledgeBaseListItem) bool {
	if a == nil || a.KnowledgeBase == nil {
		return b != nil && b.KnowledgeBase != nil
	}
	if b == nil || b.KnowledgeBase == nil {
		return true
	}
	akb, bkb := a.KnowledgeBase, b.KnowledgeBase
	if akb.IsPinned != bkb.IsPinned {
		return akb.IsPinned
	}
	if akb.IsPinned && bkb.IsPinned {
		at, bt := akb.PinnedAt, bkb.PinnedAt
		if at != nil && bt != nil && !at.Equal(*bt) {
			return at.After(*bt)
		}
		if at != nil && bt == nil {
			return true
		}
		if at == nil && bt != nil {
			return false
		}
	}
	return knowledgeBaseManualOrderLess(akb, bkb)
}
