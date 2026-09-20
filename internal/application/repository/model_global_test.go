package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestModelRepositorySharesSystemModelsAcrossWorkspaces(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(
		fmt.Sprintf("file:model-global-%d?mode=memory&cache=shared", time.Now().UnixNano()),
	), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.Model{}))

	repo := NewModelRepository(db)
	ctx := context.Background()
	systemModel := &types.Model{
		ID:       "system-chat",
		TenantID: types.SystemModelTenantID,
		Name:     "system-chat",
		Type:     types.ModelTypeKnowledgeQA,
		Source:   types.ModelSourceRemote,
		Status:   types.ModelStatusActive,
	}
	otherTenantModel := &types.Model{
		ID:       "other-chat",
		TenantID: 10000,
		Name:     "other-chat",
		Type:     types.ModelTypeKnowledgeQA,
		Source:   types.ModelSourceRemote,
		Status:   types.ModelStatusActive,
	}
	require.NoError(t, repo.Create(ctx, systemModel))
	require.NoError(t, repo.Create(ctx, otherTenantModel))

	models, err := repo.List(ctx, 10003, types.ModelTypeKnowledgeQA, "")
	require.NoError(t, err)
	require.Len(t, models, 1)
	require.Equal(t, "system-chat", models[0].ID)

	model, err := repo.GetByID(ctx, 10003, "system-chat")
	require.NoError(t, err)
	require.NotNil(t, model)
	require.Equal(t, types.SystemModelTenantID, model.TenantID)

	model, err = repo.GetByID(ctx, 10003, "other-chat")
	require.NoError(t, err)
	require.Nil(t, model)
}
