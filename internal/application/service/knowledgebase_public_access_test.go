package service

import (
	"context"
	"testing"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type publicAccessRepoStub struct {
	interfaces.PublicKnowledgeBaseRepository
	publication *types.PublicKnowledgeBasePublication
	active      bool
}

func (r *publicAccessRepoStub) GetPublicationByKnowledgeBaseID(
	_ context.Context,
	_ string,
) (*types.PublicKnowledgeBasePublication, error) {
	if r.publication == nil {
		return nil, apprepo.ErrPublicKnowledgeBasePublicationNotFound
	}
	return r.publication, nil
}

func (r *publicAccessRepoStub) IsActiveSubscription(
	_ context.Context,
	_, _ string,
) (bool, error) {
	return r.active, nil
}

func TestResolveKnowledgeBaseAccess_PublicSubscriptionIsViewerOnly(t *testing.T) {
	repo := newFakeKBRepo()
	repo.rows["kb-public"] = &types.KnowledgeBase{
		ID:        "kb-public",
		TenantID:  100,
		CreatorID: "publisher",
	}
	svc := newPR3KBService(repo, &fakeRegistry{}, &fakeOwnership{})
	svc.publicKBRepo = &publicAccessRepoStub{
		publication: &types.PublicKnowledgeBasePublication{
			ID:              "publication-1",
			KnowledgeBaseID: "kb-public",
			Status:          types.PublicKnowledgeBasePublicationPublished,
		},
		active: true,
	}
	ctx := accessCtx(200, types.TenantRoleViewer, "subscriber-1", nil)

	access, err := svc.ResolveKnowledgeBaseAccess(
		ctx,
		"kb-public",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleViewer},
	)
	require.NoError(t, err)
	require.Equal(t, types.OrgRoleViewer, access.Permission)
	require.Equal(t, types.KnowledgeBaseAccessSourcePublicSubscription, access.AccessSource)
	require.True(t, access.IsSubscribed)
	require.Equal(t, uint64(100), access.EffectiveTenantID)

	_, err = svc.ResolveKnowledgeBaseAccess(
		ctx,
		"kb-public",
		types.KnowledgeBaseAccessOptions{RequiredPermission: types.OrgRoleEditor},
	)
	require.ErrorIs(t, err, types.ErrKnowledgeBaseAccessForbidden)
}
