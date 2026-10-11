package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantSandboxConfigRepositoryActivatesOnlyEnabledConfig(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSandboxConfig{}))

	repo := NewTenantSandboxConfigRepository(db)
	ctx := context.Background()
	first := &types.TenantSandboxConfig{
		TenantID: 42,
		Name:     "local",
		Mode:     types.TenantSandboxModeLocal,
		Enabled:  true,
	}
	second := &types.TenantSandboxConfig{
		TenantID: 42,
		Name:     "docker",
		Mode:     types.TenantSandboxModeDocker,
		Enabled:  true,
	}
	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Create(ctx, second))

	require.NoError(t, repo.SetActive(ctx, first.TenantID, first.ID))
	active, err := repo.GetActive(ctx, first.TenantID)
	require.NoError(t, err)
	require.Equal(t, first.ID, active.ID)

	require.NoError(t, repo.SetActive(ctx, second.TenantID, second.ID))
	active, err = repo.GetActive(ctx, first.TenantID)
	require.NoError(t, err)
	require.Equal(t, second.ID, active.ID)

	second.Enabled = false
	require.NoError(t, repo.Update(ctx, second))
	require.ErrorIs(t, repo.SetActive(ctx, second.TenantID, second.ID), gorm.ErrRecordNotFound)
	active, err = repo.GetActive(ctx, first.TenantID)
	require.NoError(t, err)
	require.Nil(t, active)
}
