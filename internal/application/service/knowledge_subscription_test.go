package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestSubscribeKnowledgeBase_CreatesShortcutAndIsIdempotent(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-shared"] = &types.KnowledgeBase{
		ID:        "kb-shared",
		Name:      "shared",
		TenantID:  200,
		CreatorID: "owner-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &accessKBShareService{
		permission: types.OrgRoleViewer,
		shared:     true,
		source:     200,
	}
	ctx := accessCtx(100, types.TenantRoleContributor, "user-1", nil)

	first, err := svc.SubscribeKnowledgeBase(ctx, "kb-shared")
	require.NoError(t, err)
	require.Equal(t, "kb-shared", first.KnowledgeBaseID)
	require.True(t, first.Subscribed)
	require.Equal(t, "sub-kb-shared", first.SubscriptionID)

	second, err := svc.SubscribeKnowledgeBase(ctx, "kb-shared")
	require.NoError(t, err)
	require.Equal(t, first.SubscriptionID, second.SubscriptionID)
	require.Len(t, repo.subscriptions, 1)
	require.Equal(t, types.KnowledgeBaseSubscriptionActive, repo.subscriptions[0].Status)
}

func TestSubscribeKnowledgeBase_RejectsInaccessibleKB(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-private"] = &types.KnowledgeBase{
		ID:        "kb-private",
		Name:      "private",
		TenantID:  200,
		CreatorID: "owner-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	_, err := svc.SubscribeKnowledgeBase(
		accessCtx(100, types.TenantRoleViewer, "user-1", nil),
		"kb-private",
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
	require.Empty(t, repo.subscriptions)
}

func TestSubscribeKnowledgeBase_RejectsPersonalKBForNonCreator(t *testing.T) {
	personal := types.SpaceTypePersonal
	repo := newFakeKBRepo()
	repo.rows["kb-personal"] = &types.KnowledgeBase{
		ID:        "kb-personal",
		Name:      "personal",
		TenantID:  100,
		CreatorID: "owner-1",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	_, err := svc.SubscribeKnowledgeBase(
		accessCtx(100, types.TenantRoleAdmin, "admin-1", &personal),
		"kb-personal",
	)

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
	require.Empty(t, repo.subscriptions)
}

func TestUnsubscribeKnowledgeBase_IsIdempotentWithoutKBAccess(t *testing.T) {
	repo := newFakeKBRepo()
	repo.subscriptions = []*types.KnowledgeBaseSubscription{
		{
			ID:              "sub-stale",
			UserID:          "user-1",
			KnowledgeBaseID: "kb-stale",
			Status:          types.KnowledgeBaseSubscriptionActive,
		},
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	ctx := accessCtx(100, types.TenantRoleViewer, "user-1", nil)

	first, err := svc.UnsubscribeKnowledgeBase(ctx, "kb-stale")
	require.NoError(t, err)
	require.Equal(t, "kb-stale", first.KnowledgeBaseID)
	require.False(t, first.Subscribed)
	require.Equal(t, "sub-stale", first.SubscriptionID)
	require.Equal(t, types.KnowledgeBaseSubscriptionCancelled, repo.subscriptions[0].Status)

	second, err := svc.UnsubscribeKnowledgeBase(ctx, "kb-stale")
	require.NoError(t, err)
	require.False(t, second.Subscribed)
	require.Equal(t, "sub-stale", second.SubscriptionID)
}

func TestSubscribeKnowledgeBase_RequiresConcreteUser(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb"] = &types.KnowledgeBase{ID: "kb", TenantID: 100, CreatorID: "user-1"}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	_, err := svc.SubscribeKnowledgeBase(accessCtx(100, types.TenantRoleAdmin, "system-100", nil), "kb")

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessUnauthorized)
}
