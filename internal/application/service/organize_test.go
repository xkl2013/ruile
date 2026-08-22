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

func TestOrganizeServiceDiscover(t *testing.T) {
	ctx := context.Background()
	svc := newOrganizeServiceForTest(t)

	_, err := svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:      "电力行业相关企业分析及功率半导体产业链解读",
		OutputType: "图文类",
		Status:     types.OrganizeOutputStatusReady,
		Metadata: types.JSONMap{
			"tags": []string{"产业链", "功率半导体"},
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:      "能源行业公司分析及中国电力结构探讨",
		OutputType: "视频类",
		Status:     types.OrganizeOutputStatusReview,
		Metadata: types.JSONMap{
			"tags": []string{"能源结构", "产业链"},
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:      "燃气轮机核心配件供应链梳理",
		OutputType: "音频类",
		Status:     types.OrganizeOutputStatusDraft,
		Metadata: types.JSONMap{
			"tags": []string{"供应链"},
		},
	})
	require.NoError(t, err)

	discover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{})
	require.NoError(t, err)
	require.Equal(t, 1, discover.Page)
	require.Equal(t, 30, discover.PageSize)
	require.Len(t, discover.Tabs, 8)
	require.Equal(t, "recommended", discover.Tabs[0].Value)
	require.Equal(t, int64(3), discover.Tabs[0].Count)
	require.Equal(t, "article", discover.Tabs[1].Value)
	require.Equal(t, "video", discover.Tabs[2].Value)
	require.Equal(t, "audio", discover.Tabs[3].Value)

	tabValues := make([]string, 0, len(discover.Tabs))
	for _, tab := range discover.Tabs {
		tabValues = append(tabValues, tab.Value)
	}
	require.Contains(t, tabValues, "tag:产业链")
	require.Contains(t, tabValues, "tag:供应链")

	require.Len(t, discover.FeaturedOutputs, 3)
	require.Equal(t, "电力行业相关企业分析及功率半导体产业链解读", discover.FeaturedOutputs[0].Title)

	videoDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{Tab: "video"})
	require.NoError(t, err)
	require.Len(t, videoDiscover.Items, 1)
	require.Equal(t, "能源行业公司分析及中国电力结构探讨", videoDiscover.Items[0].Title)

	tagDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{Tab: "tag:产业链"})
	require.NoError(t, err)
	require.Len(t, tagDiscover.Items, 2)
	require.Equal(t, "电力行业相关企业分析及功率半导体产业链解读", tagDiscover.Items[0].Title)

	pagedDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Equal(t, 2, pagedDiscover.Page)
	require.Equal(t, 2, pagedDiscover.PageSize)
	require.Len(t, pagedDiscover.Items, 1)
	require.Equal(t, discover.Items[2].ID, pagedDiscover.Items[0].ID)

	rotated, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{FeaturedOffset: 1})
	require.NoError(t, err)
	require.Len(t, rotated.FeaturedOutputs, 3)
	require.NotEqual(t, discover.FeaturedOutputs[0].ID, rotated.FeaturedOutputs[0].ID)
}
