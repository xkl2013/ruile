package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type searchScopeKnowledgeRepo struct {
	interfaces.KnowledgeRepository
	scopes []types.KnowledgeSearchScope
}

func (r *searchScopeKnowledgeRepo) SearchKnowledgeInScopes(
	_ context.Context,
	scopes []types.KnowledgeSearchScope,
	_ string,
	_ int,
	_ int,
	_ []string,
) ([]*types.Knowledge, bool, int64, error) {
	r.scopes = append([]types.KnowledgeSearchScope(nil), scopes...)
	return nil, false, 0, nil
}

type searchScopeKnowledgeBaseService struct {
	interfaces.KnowledgeBaseService
	current []*types.KnowledgeBase
	account *types.MyKnowledgeBaseList
}

func (s *searchScopeKnowledgeBaseService) ListKnowledgeBases(context.Context) ([]*types.KnowledgeBase, error) {
	return s.current, nil
}

func (s *searchScopeKnowledgeBaseService) ListMyKnowledgeBases(context.Context) (*types.MyKnowledgeBaseList, error) {
	return s.account, nil
}

func TestSearchKnowledgeIncludesTransferredAccountKnowledgeBase(t *testing.T) {
	repo := &searchScopeKnowledgeRepo{}
	kb := &types.KnowledgeBase{
		ID:        "transferred-kb",
		TenantID:  200,
		Type:      types.KnowledgeBaseTypeDocument,
		CreatorID: "new-user",
	}
	svc := &knowledgeService{
		repo: repo,
		kbService: &searchScopeKnowledgeBaseService{
			account: &types.MyKnowledgeBaseList{
				Created: []*types.MyKnowledgeBaseListItem{{
					KnowledgeBase:     kb,
					EffectiveTenantID: 200,
				}},
			},
		},
	}
	ctx := context.WithValue(context.Background(), types.TenantIDContextKey, uint64(100))
	ctx = context.WithValue(ctx, types.UserIDContextKey, "new-user")

	_, _, _, err := svc.SearchKnowledge(ctx, "常青藤", 0, 20, nil)

	require.NoError(t, err)
	require.Equal(t, []types.KnowledgeSearchScope{
		{TenantID: 200, KBID: "transferred-kb"},
	}, repo.scopes)
}
