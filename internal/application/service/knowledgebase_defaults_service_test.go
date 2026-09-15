package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type defaultsTenantRepoStub struct {
	interfaces.TenantRepository
	updated *types.Tenant
}

func (s *defaultsTenantRepoStub) UpdateTenant(_ context.Context, tenant *types.Tenant) error {
	s.updated = tenant
	return nil
}

func TestKnowledgeBaseDefaultsUpdateDoesNotRewriteExistingKnowledgeBases(t *testing.T) {
	repo := &defaultsTenantRepoStub{}
	tenant := &types.Tenant{ID: 42}
	ctx := context.WithValue(context.Background(), types.TenantInfoContextKey, tenant)
	svc := &knowledgeBaseDefaultsService{tenantRepo: repo}

	updated, applied, err := svc.Update(ctx, &types.KnowledgeBaseDefaultsConfig{
		SummaryModelID:   "llm-1",
		EmbeddingModelID: "embedding-1",
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, 0, applied)
	require.Same(t, tenant, repo.updated)
	require.NotNil(t, tenant.KnowledgeBaseDefaultsConfig)
}
