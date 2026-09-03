package service

import (
	"context"
	"strings"
	"testing"
	"time"

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

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title: "Invalid category",
		Metadata: types.JSONMap{
			"discover_category": "unknown_category",
		},
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidCategory)

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
	require.Equal(t, []types.OrganizeMemoryReference{
		{ID: memory.ID, Kind: memory.Kind, Title: memory.Title, Source: memory.Source},
	}, report.MemoryRefs)

	linkedReports, linkedTotal, err := svc.ListSproutReports(ctx, types.OrganizeListQuery{
		TenantID: 7,
		UserID:   "user-a",
		MemoryID: memory.ID,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), linkedTotal)
	require.Len(t, linkedReports, 1)
	require.Equal(t, report.ID, linkedReports[0].ID)

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
			"tags":              []string{"产业链", "功率半导体"},
			"discover_category": types.OrganizeDiscoverCategoryAdmissionsGrowth,
		},
	})
	require.NoError(t, err)

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:      "能源行业公司分析及中国电力结构探讨",
		OutputType: "视频类",
		Status:     types.OrganizeOutputStatusReview,
		Metadata: types.JSONMap{
			"tags":              []string{"能源结构", "产业链"},
			"discover_category": types.OrganizeDiscoverCategoryParentService,
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
	require.Len(t, discover.Tabs, 9)
	require.Equal(t, "recommended", discover.Tabs[0].Value)
	require.Equal(t, int64(3), discover.Tabs[0].Count)
	require.Equal(t, types.OrganizeDiscoverCategoryAdmissionsGrowth, discover.Tabs[1].Value)
	require.Equal(t, int64(1), discover.Tabs[1].Count)

	require.Len(t, discover.FeaturedOutputs, 3)
	require.Equal(t, "电力行业相关企业分析及功率半导体产业链解读", discover.FeaturedOutputs[0].Title)

	recommendedDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{Tab: "recommended"})
	require.NoError(t, err)
	require.Len(t, recommendedDiscover.Items, 3)

	categoryDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{
		Tab: types.OrganizeDiscoverCategoryAdmissionsGrowth,
	})
	require.NoError(t, err)
	require.Len(t, categoryDiscover.Items, 1)
	require.Equal(t, "电力行业相关企业分析及功率半导体产业链解读", categoryDiscover.Items[0].Title)

	emptyCategoryDiscover, err := svc.GetDiscover(ctx, 7, "user-a", types.OrganizeDiscoverQuery{
		Tab: types.OrganizeDiscoverCategoryEventPlanning,
	})
	require.NoError(t, err)
	require.NotNil(t, emptyCategoryDiscover.Items)
	require.NotNil(t, emptyCategoryDiscover.FeaturedOutputs)
	require.Empty(t, emptyCategoryDiscover.Items)
	require.Empty(t, emptyCategoryDiscover.FeaturedOutputs)

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

func TestOrganizeServiceCreateSproutReportFromMemoryGeneratesWithRoleConfig(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.TenantRoleContextKey, types.TenantRoleAdmin)
	svc := newOrganizeServiceForTest(t)
	svc.modelService = &stubOrganizeModelService{
		models: []*types.Model{
			{ID: "chat-1", Type: types.ModelTypeKnowledgeQA, Status: types.ModelStatusActive, IsDefault: true},
		},
		chatModel: &stubOrganizeChatModel{
			content: "AI发芽报告\n\n## 01. 经营信号\n\n> **🌱 种子**\n> 家长关注开放日体验。\n\n> **✨ Aha 瞬间**\n> 需要把体验拆成可跟进动作。",
		},
	}

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:    types.OrganizeMemoryKindNote,
		Title:   "开放日家长反馈",
		Content: "家长集中询问开放日动线、试听安排和报名政策。",
		Source:  "随笔",
		Metadata: types.JSONMap{
			"tags": []string{"开放日", "家长沟通"},
		},
	})
	require.NoError(t, err)

	report, err := svc.CreateSproutReportFromMemory(ctx, 7, "user-b", types.OrganizeSproutFromMemoryInput{
		MemoryID: memory.ID,
		RoleConfig: types.JSONMap{
			"position": "园长",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "user-b", report.UserID)
	require.Equal(t, types.OrganizeSproutStageOrganizing, report.Stage)
	require.Equal(t, int64(1), report.MemoryCount)
	require.ElementsMatch(t, []string{memory.ID}, report.MemoryIDs)
	require.Equal(t, []types.OrganizeMemoryReference{
		{ID: memory.ID, Kind: memory.Kind, Title: memory.Title, Source: memory.Source},
	}, report.MemoryRefs)
	require.Equal(t, "pending", report.Metadata["ai_status"])
	require.Equal(t, string(types.TenantRoleAdmin), report.Metadata["role"])

	require.Eventually(t, func() bool {
		generated, err := svc.GetSproutReport(ctx, 7, "user-b", report.ID)
		if err != nil {
			return false
		}
		return generated.Stage == types.OrganizeSproutStageFormed &&
			strings.Contains(generated.Summary, "AI发芽报告") &&
			generated.Metadata["ai_status"] == "completed" &&
			generated.Metadata["ai_model_id"] == "chat-1"
	}, 2*time.Second, 20*time.Millisecond)
}
