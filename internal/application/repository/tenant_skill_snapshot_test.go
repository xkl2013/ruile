package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantSkillSnapshotRepositoryActivationSupersedesPreviousSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkillSnapshot{}))

	repo := NewTenantSkillSnapshotRepository(db)
	ctx := context.Background()
	first := &types.TenantSkillSnapshot{
		TenantID:       42,
		Generation:     1,
		State:          types.TenantSkillSnapshotStateBuilding,
		Manifest:       types.JSONMap{"skills": []any{}},
		ManifestSHA256: "first",
	}
	second := &types.TenantSkillSnapshot{
		TenantID:         42,
		Generation:       2,
		ParentSnapshotID: "first",
		State:            types.TenantSkillSnapshotStateBuilding,
		Manifest:         types.JSONMap{"skills": []any{"demo"}},
		ManifestSHA256:   "second",
	}
	require.NoError(t, repo.Create(ctx, first))
	require.NoError(t, repo.Create(ctx, second))
	now := time.Now().UTC()
	require.NoError(t, repo.Activate(ctx, 42, first.ID, now, now))
	require.NoError(t, repo.Activate(ctx, 42, second.ID, now.Add(time.Second), now.Add(time.Second)))

	active, err := repo.GetActive(ctx, 42)
	require.NoError(t, err)
	require.Equal(t, second.ID, active.ID)

	var states []struct {
		ID    string
		State string
	}
	require.NoError(t, db.Model(&types.TenantSkillSnapshot{}).
		Where("tenant_id = ?", 42).
		Order("generation ASC").
		Find(&states).Error)
	require.Equal(t, types.TenantSkillSnapshotStateSuperseded, states[0].State)
	require.Equal(t, types.TenantSkillSnapshotStateActive, states[1].State)
}
