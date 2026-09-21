package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type stubModelRepoForModelCreate struct {
	interfaces.ModelRepository
	existing []*types.Model
	created  bool
}

func (s *stubModelRepoForModelCreate) List(
	context.Context,
	uint64,
	types.ModelType,
	types.ModelSource,
) ([]*types.Model, error) {
	return s.existing, nil
}

func (s *stubModelRepoForModelCreate) Create(context.Context, *types.Model) error {
	s.created = true
	return nil
}

func TestCreateModelRejectsDuplicatePlatformIdentity(t *testing.T) {
	repo := &stubModelRepoForModelCreate{
		existing: []*types.Model{{
			ID:       "existing-model",
			TenantID: types.SystemModelTenantID,
			Name:     " QWEN3.8-MAX ",
			Type:     types.ModelTypeKnowledgeQA,
			Source:   types.ModelSourceRemote,
		}},
	}
	svc := NewModelService(repo, nil, nil, nil, nil, nil)

	err := svc.CreateModel(context.Background(), &types.Model{
		ID:       "new-model",
		TenantID: types.SystemModelTenantID,
		Name:     "qwen3.8-max",
		Type:     types.ModelTypeKnowledgeQA,
		Source:   types.ModelSourceRemote,
	})

	require.ErrorIs(t, err, ErrModelAlreadyExists)
	require.False(t, repo.created)
}
