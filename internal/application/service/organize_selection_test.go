package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestParseOrganizeMemoryQueryPlanToday(t *testing.T) {
	now := time.Date(2026, 10, 7, 15, 30, 0, 0, time.FixedZone("CST", 8*60*60))
	plan, ambiguities, warnings := parseOrganizeMemoryQueryPlan("整理今日记忆中的线索", now)

	require.Empty(t, ambiguities)
	require.Empty(t, warnings)
	require.Equal(t, "2026-10-07T00:00:00+08:00", plan.OccurredFrom)
	require.Equal(t, "2026-10-08T00:00:00+08:00", plan.OccurredTo)
	require.Empty(t, plan.Keyword)
	require.Empty(t, plan.Kinds)
}

func TestPreviewOrganizeRequirementQueriesTodayMemories(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	template := &types.OrganizeTemplate{
		Scope:              types.OrganizeTemplateScopePlatform,
		Key:                "today_leads",
		Name:               "今日线索",
		Scene:              "招生",
		OutputLabel:        "线索清单",
		DefaultInstruction: "识别记忆中的线索并保留引用。",
		Status:             types.OrganizeTemplateStatusEnabled,
		PublishedVersion:   "v1",
	}
	require.NoError(t, db.Create(template).Error)
	config, err := svc.CreateConfig(ctx, 7, "user-a", types.OrganizeConfigInput{
		Name:        "今日线索整理",
		TemplateKey: template.Key,
		Schedule:    types.OrganizeScheduleManual,
	})
	require.NoError(t, err)

	location := organizeScheduleLocation()
	today := time.Now().In(location)
	todayAtTen := time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, location)
	yesterday := todayAtTen.AddDate(0, 0, -1)
	_, err = svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "今日家长咨询",
		Content:    "王女士咨询试听课程，担心接送时间。",
		OccurredAt: &todayAtTen,
	})
	require.NoError(t, err)
	_, err = svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "昨日咨询",
		Content:    "昨日记录",
		OccurredAt: &yesterday,
	})
	require.NoError(t, err)

	preview, err := svc.PreviewOrganizeRequirement(ctx, 7, "user-a", types.OrganizeRequirementInput{
		ConfigID: config.ID,
		Text:     "整理今日记忆中的线索",
	})
	require.NoError(t, err)
	require.False(t, preview.NeedConfirmation)
	require.Equal(t, 1, preview.SelectedCount)
	require.Equal(t, 1, preview.ReadyCount)
	require.Equal(t, 1, preview.EstimatedBatchCount)
	require.Len(t, preview.MemoryIDs, 1)
	require.Equal(t, "今日家长咨询", preview.SampleMemories[0].Title)
}

func TestConfirmOrganizeRequirementCreatesBatchJob(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	template := &types.OrganizeTemplate{
		Scope:              types.OrganizeTemplateScopePlatform,
		Key:                "batch_leads",
		Name:               "批量线索",
		Scene:              "招生",
		OutputLabel:        "线索清单",
		DefaultInstruction: "提取线索并保留来源。",
		Status:             types.OrganizeTemplateStatusEnabled,
		PublishedVersion:   "v1",
	}
	require.NoError(t, db.Create(template).Error)
	config, err := svc.CreateConfig(ctx, 8, "user-b", types.OrganizeConfigInput{
		Name:        "今日批量线索",
		TemplateKey: template.Key,
		Schedule:    types.OrganizeScheduleManual,
	})
	require.NoError(t, err)

	location := organizeScheduleLocation()
	today := time.Now().In(location)
	occurredAt := time.Date(today.Year(), today.Month(), today.Day(), 9, 0, 0, 0, location)
	for index := 0; index < 8; index++ {
		_, createErr := svc.CreateMemory(ctx, 8, "user-b", types.OrganizeMemoryInput{
			Kind:       types.OrganizeMemoryKindNote,
			Title:      "线索记录",
			Content:    strings.Repeat("家长咨询课程和接送安排。", 180),
			OccurredAt: &occurredAt,
		})
		require.NoError(t, createErr)
	}

	job, err := svc.ConfirmOrganizeRequirement(ctx, 8, "user-b", types.OrganizeRequirementInput{
		ConfigID: config.ID,
		Text:     "整理今日记忆中的线索",
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeJobModeBatch, job.JobMode)
	require.Greater(t, job.BatchCount, 1)
	require.Len(t, job.Batches, job.BatchCount)
	require.Equal(t, 8, job.SelectedCount)
	require.Equal(t, 8, job.ReadyCount)
	require.NotEmpty(t, job.InputFingerprint)
	require.NotEmpty(t, job.SelectionSnapshot)
}

func TestRunConfigUsesInstructionDateScope(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	template := &types.OrganizeTemplate{
		Scope:              types.OrganizeTemplateScopePlatform,
		Key:                "manual_today_leads",
		Name:               "立即整理今日线索",
		Scene:              "招生",
		OutputLabel:        "线索清单",
		DefaultInstruction: "整理今日记忆中的线索",
		Status:             types.OrganizeTemplateStatusEnabled,
		PublishedVersion:   "v1",
	}
	require.NoError(t, db.Create(template).Error)
	config, err := svc.CreateConfig(ctx, 17, "manual-user", types.OrganizeConfigInput{
		Name:        "立即整理今日线索",
		TemplateKey: template.Key,
		Instruction: "整理今日记忆中的线索",
		Schedule:    types.OrganizeScheduleManual,
	})
	require.NoError(t, err)

	location := organizeScheduleLocation()
	today := time.Now().In(location)
	todayAtTen := time.Date(today.Year(), today.Month(), today.Day(), 10, 0, 0, 0, location)
	yesterdayAtTen := todayAtTen.AddDate(0, 0, -1)
	todayMemory, err := svc.CreateMemory(ctx, 17, "manual-user", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "今日试听咨询",
		Content:    "家长咨询周末试听安排。",
		OccurredAt: &todayAtTen,
	})
	require.NoError(t, err)
	yesterdayMemory, err := svc.CreateMemory(ctx, 17, "manual-user", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "昨日试听咨询",
		Content:    "前一天的试听咨询。",
		OccurredAt: &yesterdayAtTen,
	})
	require.NoError(t, err)

	job, err := svc.RunConfig(ctx, 17, "manual-user", config.ID, types.OrganizeJobInput{})
	require.NoError(t, err)
	require.Equal(t, 1, job.SelectedCount)
	require.Equal(t, 1, job.ReadyCount)
	require.Equal(t, 1, job.ProcessedCount)
	require.Equal(t, types.StringArray{todayMemory.ID}, job.MemoryIDs)
	require.NotContains(t, []string(job.MemoryIDs), yesterdayMemory.ID)
	require.NotEmpty(t, job.InputFingerprint)
	require.NotEmpty(t, job.SelectionSnapshot)
}

func TestScheduledOrganizeJobUsesInstructionDateScope(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	template := &types.OrganizeTemplate{
		Scope:              types.OrganizeTemplateScopePlatform,
		Key:                "scheduled_today_leads",
		Name:               "周期今日线索",
		Scene:              "招生",
		OutputLabel:        "线索清单",
		DefaultInstruction: "整理今日记忆中的线索",
		Status:             types.OrganizeTemplateStatusEnabled,
		PublishedVersion:   "v1",
	}
	require.NoError(t, db.Create(template).Error)
	config, err := svc.CreateConfig(ctx, 18, "scheduler-user", types.OrganizeConfigInput{
		Name:        "每天整理今日线索",
		TemplateKey: template.Key,
		Instruction: "整理今日记忆中的线索",
		Schedule:    types.OrganizeScheduleDaily,
	})
	require.NoError(t, err)

	location := organizeScheduleLocation()
	scheduledFor := time.Date(2026, 10, 7, 8, 0, 0, 0, location)
	todayAtSeven := time.Date(2026, 10, 7, 7, 30, 0, 0, location)
	yesterdayAtSeven := todayAtSeven.AddDate(0, 0, -1)
	todayMemory, err := svc.CreateMemory(ctx, 18, "scheduler-user", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "10 月 7 日家长咨询",
		Content:    "家长咨询课程时间和试听安排。",
		OccurredAt: &todayAtSeven,
	})
	require.NoError(t, err)
	yesterdayMemory, err := svc.CreateMemory(ctx, 18, "scheduler-user", types.OrganizeMemoryInput{
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "10 月 6 日家长咨询",
		Content:    "前一天的咨询记录。",
		OccurredAt: &yesterdayAtSeven,
	})
	require.NoError(t, err)

	created, err := svc.createScheduledOrganizeJob(ctx, config, scheduledFor)
	require.NoError(t, err)
	require.NotNil(t, created)

	job, err := svc.GetJob(ctx, 18, "scheduler-user", created.ID)
	require.NoError(t, err)
	require.Equal(t, 1, job.SelectedCount)
	require.Equal(t, 1, job.ReadyCount)
	require.Equal(t, 1, job.ProcessedCount)
	require.Equal(t, 1, job.BatchCount)
	require.Equal(t, types.StringArray{todayMemory.ID}, job.MemoryIDs)
	require.NotContains(t, []string(job.MemoryIDs), yesterdayMemory.ID)
	require.Equal(t, "2026-10-07T00:00:00+08:00", stringValue(
		organizeJSONMapValue(job.SelectionSnapshot["query_plan"]),
		"occurred_from",
	))
}
