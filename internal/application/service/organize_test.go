package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newOrganizeServiceForTest(t *testing.T) *organizeService {
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
	return &organizeService{repo: repository.NewOrganizeRepository(db)}
}

func TestOrganizeServiceValidationAndOverview(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)

	_, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{Kind: "note", Title: "   "})
	require.ErrorIs(t, err, ErrOrganizeTitleRequired)

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:            "audio-card",
		Title:           "  Voice card  ",
		DurationSeconds: -10,
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeMemoryKindAudioCard, memory.Kind)
	require.Equal(t, "Voice card", memory.Title)
	require.Equal(t, 0, memory.DurationSeconds)

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:     "Bad output",
		MemoryIDs: []string{"missing-memory"},
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidMemoryRefs)

	output, err := svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:     "Draft output",
		MemoryIDs: []string{memory.ID, memory.ID},
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeOutputStatusDraft, output.Status)
	require.Equal(t, int64(1), output.MemoryCount)
	require.ElementsMatch(t, []string{memory.ID}, output.MemoryIDs)

	_, _, err = svc.ListOutputs(ctx, types.OrganizeListQuery{
		TenantID: 7,
		UserID:   "user-a",
		Status:   "unknown",
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidStatus)

	report, err := svc.CreateSproutReport(ctx, 7, "user-a", types.OrganizeSproutReportInput{
		Title:     "Sprout",
		Stage:     types.OrganizeSproutStageExpandable,
		Chips:     types.StringArray{"产业链", "产业链", "国产替代"},
		MemoryIDs: []string{memory.ID},
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeSproutStageExpandable, report.Stage)
	require.ElementsMatch(t, []string{"产业链", "国产替代"}, []string(report.Chips))

	_, err = svc.CreateSproutReport(ctx, 7, "user-a", types.OrganizeSproutReportInput{
		Title: "Invalid stage",
		Stage: "梳理中",
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidStage)

	_, _, err = svc.ListSproutReports(ctx, types.OrganizeListQuery{
		TenantID: 7,
		UserID:   "user-a",
		Stage:    "梳理中",
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidStage)

	overview, err := svc.GetOverview(ctx, 7, "user-a")
	require.NoError(t, err)
	require.Len(t, overview.Tabs, 3)
	require.Equal(t, int64(1), overview.Tabs[0].Count)
	require.Equal(t, int64(1), overview.Tabs[1].Count)
	require.Equal(t, int64(1), overview.Tabs[2].Count)
}
