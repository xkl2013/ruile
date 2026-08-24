package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrganizeTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.OrganizeMemory{},
		&types.OrganizeOutput{},
		&types.OrganizeOutputMemory{},
		&types.OrganizeSproutReport{},
		&types.OrganizeSproutMemory{},
	))
	return db
}

func TestOrganizeRepositoryRoundTrip(t *testing.T) {
	ctx := context.Background()
	db := newOrganizeTestDB(t)
	repo := NewOrganizeRepository(db)

	const tenantID uint64 = 42
	const userID = "user-a"

	note := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "Alpha note",
	}
	audio := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   userID,
		Kind:     types.OrganizeMemoryKindAudio,
		Title:    "Beta audio",
	}
	otherUser := &types.OrganizeMemory{
		TenantID: tenantID,
		UserID:   "user-b",
		Kind:     types.OrganizeMemoryKindNote,
		Title:    "Hidden note",
	}
	require.NoError(t, repo.CreateMemory(ctx, note))
	require.NoError(t, repo.CreateMemory(ctx, audio))
	require.NoError(t, repo.CreateMemory(ctx, otherUser))

	tenantMemory, err := repo.GetTenantMemory(ctx, tenantID, otherUser.ID)
	require.NoError(t, err)
	require.NotNil(t, tenantMemory)
	require.Equal(t, otherUser.ID, tenantMemory.ID)

	memories, total, err := repo.ListMemories(ctx, types.OrganizeListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, memories, 2)

	memoryCounts, err := repo.CountMemoriesByKind(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Equal(t, int64(1), memoryCounts[types.OrganizeMemoryKindNote])
	require.Equal(t, int64(1), memoryCounts[types.OrganizeMemoryKindAudio])

	output := &types.OrganizeOutput{
		TenantID: tenantID,
		UserID:   userID,
		Title:    "Deliverable",
		Status:   types.OrganizeOutputStatusReview,
	}
	require.NoError(t, repo.CreateOutput(ctx, output, []string{note.ID, audio.ID}))

	gotOutput, err := repo.GetOutput(ctx, tenantID, userID, output.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), gotOutput.MemoryCount)
	require.ElementsMatch(t, []string{note.ID, audio.ID}, gotOutput.MemoryIDs)

	outputStats, err := repo.CountOutputsByStatus(ctx, tenantID, userID)
	require.NoError(t, err)
	require.Equal(t, int64(1), outputStats[types.OrganizeOutputStatusReview])

	report := &types.OrganizeSproutReport{
		TenantID: tenantID,
		UserID:   userID,
		Title:    "Research sprout",
		Stage:    types.OrganizeSproutStageOrganizing,
	}
	require.NoError(t, repo.CreateSproutReport(ctx, report, []string{note.ID}))

	gotReport, err := repo.GetSproutReport(ctx, tenantID, userID, report.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotReport.MemoryCount)
	require.ElementsMatch(t, []string{note.ID}, gotReport.MemoryIDs)

	require.NoError(t, repo.DeleteMemory(ctx, tenantID, userID, note.ID))
	gotOutput, err = repo.GetOutput(ctx, tenantID, userID, output.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), gotOutput.MemoryCount)
	require.ElementsMatch(t, []string{audio.ID}, gotOutput.MemoryIDs)
}
