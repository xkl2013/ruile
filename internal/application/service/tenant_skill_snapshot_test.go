package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantSkillSnapshotServiceCreatesVerifiesAndRollsBack(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillSnapshot{}))

	skillRepo := repository.NewTenantSkillRepository(db)
	skillService := NewTenantSkillService(skillRepo, newTenantSkillFileStub())
	snapshotRepo := repository.NewTenantSkillSnapshotRepository(db)
	snapshotService := NewTenantSkillSnapshotService(snapshotRepo, skillService)
	ctx := context.Background()

	ready := &types.TenantSkill{
		TenantID:  7,
		Name:      "ready-skill",
		Enabled:   true,
		Status:    types.TenantSkillStatusReady,
		Version:   "1.0.0",
		BundleSHA: "abc",
	}
	disabled := &types.TenantSkill{
		TenantID:  7,
		Name:      "disabled-skill",
		Enabled:   false,
		Status:    types.TenantSkillStatusReady,
		BundleSHA: "def",
	}
	require.NoError(t, skillRepo.Create(ctx, ready))
	require.NoError(t, skillRepo.Create(ctx, disabled))

	first, err := snapshotService.Create(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 1, first.Generation)
	require.Equal(t, types.TenantSkillSnapshotStateActive, first.State)
	require.True(t, snapshotManifestIsValid(t, first))

	require.NoError(t, skillRepo.SetEnabled(ctx, 7, ready.ID, false))
	second, err := snapshotService.Create(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, 2, second.Generation)
	require.Equal(t, first.ID, second.ParentSnapshotID)

	valid, err := snapshotService.Verify(ctx, 7, first.ID)
	require.NoError(t, err)
	require.True(t, valid)

	rolledBack, err := snapshotService.Rollback(ctx, 7, first.ID)
	require.NoError(t, err)
	require.Equal(t, first.ID, rolledBack.ID)
	require.Equal(t, types.TenantSkillSnapshotStateActive, rolledBack.State)
	active, err := snapshotRepo.GetActive(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, first.ID, active.ID)
}

func snapshotManifestIsValid(t *testing.T, snapshot *types.TenantSkillSnapshot) bool {
	t.Helper()
	hash, err := hashTenantSkillSnapshotManifest(snapshot.Manifest)
	require.NoError(t, err)
	return hash == snapshot.ManifestSHA256
}
