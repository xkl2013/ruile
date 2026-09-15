package service

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// SubscribeKnowledgeBase creates or reactivates the caller's personal shortcut
// to a KB that is currently readable. Subscription is metadata only and never
// grants access on later reads.
func (s *knowledgeBaseService) SubscribeKnowledgeBase(
	ctx context.Context,
	kbID string,
) (*types.KnowledgeBaseSubscriptionResult, error) {
	userID, ok := concreteSubscriptionUserID(ctx)
	if !ok {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return nil, types.ErrKnowledgeBaseAccessNotFound
	}

	access, err := s.ResolveKnowledgeBaseAccess(ctx, kbID, types.KnowledgeBaseAccessOptions{
		RequiredPermission: types.OrgRoleViewer,
	})
	if err != nil {
		return nil, err
	}
	if !canSubscribeKnowledgeBase(userID, access) {
		return nil, types.ErrKnowledgeBaseAccessForbidden
	}

	sub, err := s.repo.UpsertKnowledgeBaseSubscription(ctx, userID, kbID)
	if err != nil {
		return nil, err
	}
	result := &types.KnowledgeBaseSubscriptionResult{
		KnowledgeBaseID: kbID,
		Subscribed:      true,
	}
	if sub != nil {
		result.SubscriptionID = sub.ID
	}
	return result, nil
}

// UnsubscribeKnowledgeBase cancels the caller's personal shortcut. This path
// intentionally does not resolve current KB access: users must be able to hide
// stale shortcuts after a share is revoked.
func (s *knowledgeBaseService) UnsubscribeKnowledgeBase(
	ctx context.Context,
	kbID string,
) (*types.KnowledgeBaseSubscriptionResult, error) {
	userID, ok := concreteSubscriptionUserID(ctx)
	if !ok {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return nil, types.ErrKnowledgeBaseAccessNotFound
	}

	sub, err := s.repo.CancelKnowledgeBaseSubscription(ctx, userID, kbID)
	if err != nil {
		return nil, err
	}
	result := &types.KnowledgeBaseSubscriptionResult{
		KnowledgeBaseID: kbID,
		Subscribed:      false,
	}
	if sub != nil {
		result.SubscriptionID = sub.ID
	}
	return result, nil
}

func concreteSubscriptionUserID(ctx context.Context) (string, bool) {
	userID, ok := types.UserIDFromContext(ctx)
	userID = strings.TrimSpace(userID)
	if !ok || userID == "" || types.IsSyntheticUserID(userID) {
		return "", false
	}
	return userID, true
}

func canSubscribeKnowledgeBase(userID string, access *types.KnowledgeBaseAccess) bool {
	if access == nil || access.KnowledgeBase == nil || access.KnowledgeBase.IsTemporary {
		return false
	}
	if access.AccessSource == types.KnowledgeBaseAccessSourceAPIKey {
		return false
	}
	if access.OwnerType != nil &&
		*access.OwnerType == types.SpaceTypePersonal &&
		access.KnowledgeBase.CreatorID != "" &&
		access.KnowledgeBase.CreatorID != userID {
		return false
	}
	return true
}
