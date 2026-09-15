package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type myListKBShareService struct {
	interfaces.KBShareService
	list        []*types.SharedKnowledgeBaseInfo
	permissions map[string]types.OrgMemberRole
	shared      map[string]bool
	sources     map[string]uint64
}

func (s *myListKBShareService) ListSharedKnowledgeBases(
	context.Context,
	uint64,
	types.TenantRole,
) ([]*types.SharedKnowledgeBaseInfo, error) {
	return s.list, nil
}

func (s *myListKBShareService) CheckTenantKBPermission(
	_ context.Context,
	kbID string,
	_ uint64,
	_ types.TenantRole,
) (types.OrgMemberRole, bool, error) {
	return s.permissions[kbID], s.shared[kbID], nil
}

func (s *myListKBShareService) GetKBSourceTenant(_ context.Context, kbID string) (uint64, error) {
	return s.sources[kbID], nil
}

func TestListMyKnowledgeBases_CategorizesAndFiltersSubscriptions(t *testing.T) {
	now := time.Date(2026, 9, 13, 10, 0, 0, 0, time.UTC)
	repo := newFakeKBRepo()
	repo.rows["created"] = &types.KnowledgeBase{
		ID:        "created",
		Name:      "created",
		TenantID:  100,
		CreatorID: "user-1",
		CreatedAt: now,
	}
	repo.rows["shared"] = &types.KnowledgeBase{
		ID:        "shared",
		Name:      "shared",
		TenantID:  200,
		CreatorID: "owner-1",
		CreatedAt: now.Add(-time.Hour),
	}
	repo.rows["revoked"] = &types.KnowledgeBase{
		ID:        "revoked",
		Name:      "revoked",
		TenantID:  300,
		CreatorID: "owner-2",
		CreatedAt: now.Add(-2 * time.Hour),
	}
	repo.rows["cancelled"] = &types.KnowledgeBase{
		ID:        "cancelled",
		Name:      "cancelled",
		TenantID:  100,
		CreatorID: "user-1",
		CreatedAt: now.Add(-3 * time.Hour),
	}
	repo.subscriptions = []*types.KnowledgeBaseSubscription{
		{
			ID:              "sub-created",
			UserID:          "user-1",
			KnowledgeBaseID: "created",
			Status:          types.KnowledgeBaseSubscriptionActive,
			CreatedAt:       now,
		},
		{
			ID:              "sub-shared",
			UserID:          "user-1",
			KnowledgeBaseID: "shared",
			Status:          types.KnowledgeBaseSubscriptionActive,
			CreatedAt:       now.Add(-time.Minute),
		},
		{
			ID:              "sub-revoked",
			UserID:          "user-1",
			KnowledgeBaseID: "revoked",
			Status:          types.KnowledgeBaseSubscriptionActive,
			CreatedAt:       now.Add(-2 * time.Minute),
		},
		{
			ID:              "sub-cancelled",
			UserID:          "user-1",
			KnowledgeBaseID: "cancelled",
			Status:          types.KnowledgeBaseSubscriptionCancelled,
			CreatedAt:       now.Add(-3 * time.Minute),
		},
	}

	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.kbShareService = &myListKBShareService{
		list: []*types.SharedKnowledgeBaseInfo{
			{
				KnowledgeBase:  repo.rows["shared"],
				ShareID:        "share-1",
				OrganizationID: "org-1",
				OrgName:        "招生共享空间",
				Permission:     types.OrgRoleEditor,
				SourceTenantID: 200,
				SharedAt:       now,
			},
		},
		permissions: map[string]types.OrgMemberRole{
			"shared": types.OrgRoleEditor,
		},
		shared: map[string]bool{
			"shared": true,
		},
		sources: map[string]uint64{
			"shared": 200,
		},
	}

	result, err := svc.ListMyKnowledgeBases(
		accessCtx(100, types.TenantRoleContributor, "user-1", nil),
	)

	require.NoError(t, err)
	require.Len(t, result.Created, 2)
	require.Equal(t, "created", result.Created[0].KnowledgeBase.ID)
	require.True(t, result.Created[0].IsSubscribed)
	require.Equal(t, "sub-created", result.Created[0].SubscriptionID)
	require.Equal(t, types.KnowledgeBaseAccessSourceCreated, result.Created[0].AccessSource)

	require.Len(t, result.Shared, 1)
	require.Equal(t, "shared", result.Shared[0].KnowledgeBase.ID)
	require.True(t, result.Shared[0].IsSubscribed)
	require.Equal(t, "sub-shared", result.Shared[0].SubscriptionID)
	require.Equal(t, types.OrgRoleEditor, result.Shared[0].Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourceSharedSpace, result.Shared[0].AccessSource)
	require.Equal(t, uint64(200), result.Shared[0].EffectiveTenantID)

	require.Len(t, result.Subscribed, 2)
	subscribedByID := map[string]*types.MyKnowledgeBaseListItem{}
	for _, item := range result.Subscribed {
		subscribedByID[item.KnowledgeBase.ID] = item
	}
	require.Contains(t, subscribedByID, "created")
	require.Contains(t, subscribedByID, "shared")
	require.NotContains(t, subscribedByID, "revoked")
	require.NotContains(t, subscribedByID, "cancelled")
	require.Equal(t, types.KnowledgeBaseAccessSourceCreated, subscribedByID["created"].AccessSource)
	require.Equal(t, types.KnowledgeBaseAccessSourceSharedSpace, subscribedByID["shared"].AccessSource)
}

func TestListMyKnowledgeBases_RequiresConcreteUser(t *testing.T) {
	repo := newFakeKBRepo()
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})

	_, err := svc.ListMyKnowledgeBases(accessCtx(100, types.TenantRoleViewer, "", nil))

	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessUnauthorized)
}
