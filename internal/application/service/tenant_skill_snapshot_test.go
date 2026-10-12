package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTenantSkillSnapshotServiceCreatesVerifiesAndRollsBack(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillSnapshot{}))

	skillRepo := repository.NewTenantSkillRepository(db)
	files := newTenantSkillFileStub()
	readyBundle, err := files.SaveBytes(context.Background(), buildSkillArchive(t, map[string]string{
		"ready-skill/SKILL.md": "---\nname: ready-skill\ndescription: Ready\n---\n\nUse it.",
	}), 7, "ready.zip", false)
	require.NoError(t, err)
	skillService := NewTenantSkillService(skillRepo, files)
	snapshotRepo := repository.NewTenantSkillSnapshotRepository(db)
	snapshotService := NewTenantSkillSnapshotService(snapshotRepo, skillService)
	ctx := context.Background()

	ready := &types.TenantSkill{
		TenantID:   7,
		Name:       "ready-skill",
		Enabled:    true,
		Status:     types.TenantSkillStatusReady,
		Version:    "1.0.0",
		BundleSHA:  "abc",
		BundlePath: readyBundle,
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
	require.NoError(t, skillRepo.SetEnabled(ctx, 7, disabled.ID, false))

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

func TestTenantSkillSnapshotPinsOldBundleAfterUpgrade(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillSnapshot{}))

	files := newTenantSkillFileStub()
	skillRepo := repository.NewTenantSkillRepository(db)
	skillService := NewTenantSkillServiceWithTasks(
		skillRepo,
		files,
		nil,
		nil,
		repository.NewTenantSkillSnapshotRepository(db),
	).(*tenantSkillService)
	ctx := context.Background()

	firstArchive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md": "---\nname: demo\ndescription: First\n---\n\nv1",
	})
	firstMetadata, _, err := inspectTenantSkillArchive(firstArchive)
	require.NoError(t, err)
	first, err := skillService.storeTenantSkillArchive(ctx, 7, "admin", firstArchive, firstMetadata, types.TenantSkillSourceUpload, "")
	require.NoError(t, err)

	snapshotRepo := repository.NewTenantSkillSnapshotRepository(db)
	snapshotService := NewTenantSkillSnapshotService(snapshotRepo, skillService)
	firstSnapshot, err := snapshotService.Create(ctx, 7)
	require.NoError(t, err)

	secondArchive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md": "---\nname: demo\ndescription: Second\n---\n\nv2",
	})
	secondMetadata, _, err := inspectTenantSkillArchive(secondArchive)
	require.NoError(t, err)
	_, err = skillService.storeTenantSkillArchive(ctx, 7, "admin", secondArchive, secondMetadata, types.TenantSkillSourceUpload, "")
	require.NoError(t, err)

	runtimeSkills, err := skillService.ListRuntimeTenantSkills(ctx, 7)
	require.NoError(t, err)
	require.Len(t, runtimeSkills, 1)
	require.Equal(t, first.BundlePath, runtimeSkills[0].BundlePath)
	require.Equal(t, first.BundleSHA, runtimeSkills[0].BundleSHA)
	require.Equal(t, firstSnapshot.ID, mustActiveSkillSnapshot(t, snapshotRepo, ctx, 7).ID)
}

func mustActiveSkillSnapshot(
	t *testing.T,
	repo interfaces.TenantSkillSnapshotRepository,
	ctx context.Context,
	tenantID uint64,
) *types.TenantSkillSnapshot {
	t.Helper()
	active, err := repo.GetActive(ctx, tenantID)
	require.NoError(t, err)
	require.NotNil(t, active)
	return active
}

func snapshotManifestIsValid(t *testing.T, snapshot *types.TenantSkillSnapshot) bool {
	t.Helper()
	hash, err := hashTenantSkillSnapshotManifest(snapshot.Manifest)
	require.NoError(t, err)
	return hash == snapshot.ManifestSHA256
}
