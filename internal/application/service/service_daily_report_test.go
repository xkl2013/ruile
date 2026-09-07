package service

import (
	"context"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newServiceDailyReportTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared&_foreign_keys=on"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.UserWorkProfile{},
		&types.WorkProfileAgentSetting{},
		&types.ServiceSubject{},
		&types.AgentWorkDoc{},
		&types.AgentWorkDocMemoryLink{},
		&types.ServiceReminder{},
		&types.AgentActionDraft{},
		&types.AgentActionLog{},
		&types.OrganizeMemory{},
		&types.TenantMember{},
	))
	return db
}

func TestServiceGenerateDailyReportFromUserTrigger(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceService(serviceRepo, organizeRepo)

	const tenantID uint64 = 7
	const userID = "user-a"
	profile, err := svc.CreateWorkProfile(ctx, tenantID, userID, types.ServiceWorkProfileInput{
		Name:           "班主任服务助理",
		RoleType:       "teacher",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	})
	require.NoError(t, err)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profile.ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain:      types.ServiceAgentDomainCustomerService,
				Enabled:          true,
				DisplayName:      "客户服务",
				WorkDocDirectory: "客户/",
			},
		},
	})
	require.NoError(t, err)

	occurredAt := time.Date(2026, 9, 1, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60)).UTC()
	memory := &types.OrganizeMemory{
		TenantID:   tenantID,
		UserID:     userID,
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "陈屿续费回访",
		Content:    "客户：陈屿妈妈\n学员：陈屿\n家长认可最近课堂反馈，本周进入续费窗口，需要先整理阶段成长记录。",
		Source:     "企微",
		OccurredAt: occurredAt,
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, memory))

	report, err := svc.GenerateDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeDay,
		Date:     "2026-09-01",
		Timezone: "Asia/Shanghai",
	})
	require.NoError(t, err)
	require.NotNil(t, report)
	require.Equal(t, types.ServiceDailyReportRangeDay, report.Range)
	require.Equal(t, "已生成", report.Stage)
	require.Equal(t, "formed", report.StageKey)
	require.Equal(t, 1, report.ActionCount)
	require.Equal(t, 1, report.CustomerCount)
	require.Contains(t, report.Title, "2026年9月1日服务日报")
	require.Contains(t, report.Content, "陈屿妈妈")
	require.Contains(t, report.Content, "生成阶段成长回顾")
	require.Contains(t, report.Content, "分身：班主任服务助理")
	require.Contains(t, report.Content, "回写建议")
	require.Equal(t, 1, report.SubjectCount)
	require.Equal(t, 1, report.OpenActionCount)
	require.Equal(t, 0, report.ClosedActionCount)
	require.Equal(t, 1, report.SourceMemoryCount)
	require.Equal(t, 100, report.EvidenceCompleteRate)
	require.Equal(t, profile.ID, report.ProfileID)
	require.Equal(t, profile.Name, report.ProfileName)
	require.Equal(t, profile.MemoryScope, report.MemoryScope)
	require.True(t, report.CanSupplement)
	require.Equal(t, "user_requested", report.Metadata["trigger"])
	require.ElementsMatch(t, []string{memory.ID}, []string(report.SourceMemoryIDs))

	customerSpaces, customerTotal, err := svc.ListCustomerSpaces(ctx, types.ServiceCustomerSpaceListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), customerTotal)
	require.Len(t, customerSpaces, 1)
	require.Equal(t, "陈屿妈妈", customerSpaces[0].DisplayName)
	require.Equal(t, 4, customerSpaces[0].WorkDocCount)
	require.Equal(t, 1, customerSpaces[0].ReminderCount)
	require.Equal(t, 1, customerSpaces[0].SourceMemoryCount)
	require.Contains(t, customerSpaces[0].Summary, "续费")

	customerSpace, err := svc.GetCustomerSpace(ctx, tenantID, userID, customerSpaces[0].ID, "")
	require.NoError(t, err)
	require.NotNil(t, customerSpace)
	require.Equal(t, customerSpaces[0].ID, customerSpace.Subject.ID)
	require.Len(t, customerSpace.WorkDocs, 4)
	require.Len(t, customerSpace.Reminders, 1)
	require.Len(t, customerSpace.MemoryEvidence, 1)
	require.Equal(t, memory.ID, customerSpace.MemoryEvidence[0].ID)

	regenerated, err := svc.GenerateDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeDay,
		Date:     "2026-09-01",
		Timezone: "Asia/Shanghai",
		Trigger:  "supplement",
	})
	require.NoError(t, err)
	require.Equal(t, report.ID, regenerated.ID)
	require.Equal(t, "supplement", regenerated.Metadata["trigger"])

	reports, total, err := svc.ListDailyReports(ctx, types.ServiceDailyReportListQuery{
		TenantID: tenantID,
		UserID:   userID,
		Range:    types.ServiceDailyReportRangeDay,
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, reports, 1)
	require.Equal(t, report.ID, reports[0].ID)

	got, err := svc.GetDailyReport(ctx, tenantID, userID, report.ID)
	require.NoError(t, err)
	require.Equal(t, report.ID, got.ID)
	require.Contains(t, got.Content, "证据来源")
}

func TestServiceGenerateWeeklyReportUsesDailySources(t *testing.T) {
	ctx := context.Background()
	db := newServiceDailyReportTestDB(t)
	serviceRepo := repository.NewServiceRepository(db)
	organizeRepo := repository.NewOrganizeRepository(db)
	svc := NewServiceService(serviceRepo, organizeRepo)

	const tenantID uint64 = 8
	const userID = "user-week"
	profile, err := svc.CreateWorkProfile(ctx, tenantID, userID, types.ServiceWorkProfileInput{
		Name:           "园区运营分身",
		RoleType:       "operator",
		DefaultProfile: true,
		Enabled:        true,
		State:          types.ServiceWorkProfileStateEnabled,
	})
	require.NoError(t, err)
	_, err = svc.ReplaceAgentSettings(ctx, tenantID, userID, profile.ID, types.WorkProfileAgentSettingsInput{
		Settings: []types.WorkProfileAgentSettingInput{
			{
				AgentDomain:      types.ServiceAgentDomainCustomerService,
				Enabled:          true,
				DisplayName:      "客户服务",
				WorkDocDirectory: "客户/",
			},
		},
	})
	require.NoError(t, err)

	day1Memory := &types.OrganizeMemory{
		TenantID:   tenantID,
		UserID:     userID,
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "林一妈妈续费回访",
		Content:    "客户：林一妈妈\n学员：林一\n家长认可最近课堂反馈，本周进入续费窗口，需要先整理阶段成长记录。",
		Source:     "企微",
		OccurredAt: time.Date(2026, 9, 1, 10, 0, 0, 0, time.FixedZone("CST", 8*60*60)).UTC(),
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, day1Memory))
	day1Report, err := svc.GenerateDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeDay,
		Date:     "2026-09-01",
		Timezone: "Asia/Shanghai",
	})
	require.NoError(t, err)

	day2Memory := &types.OrganizeMemory{
		TenantID:   tenantID,
		UserID:     userID,
		Kind:       types.OrganizeMemoryKindNote,
		Title:      "周予爸爸售后沟通",
		Content:    "客户：周予爸爸\n学员：周予\n今天反馈午睡不适应，家长担心老师没有跟进，需要今晚确认处理结果并记录反馈。",
		Source:     "企微",
		OccurredAt: time.Date(2026, 9, 2, 11, 0, 0, 0, time.FixedZone("CST", 8*60*60)).UTC(),
	}
	require.NoError(t, organizeRepo.CreateMemory(ctx, day2Memory))
	day2Report, err := svc.GenerateDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeDay,
		Date:     "2026-09-02",
		Timezone: "Asia/Shanghai",
	})
	require.NoError(t, err)

	weekReport, err := svc.GenerateDailyReport(ctx, tenantID, userID, types.ServiceDailyReportInput{
		Range:    types.ServiceDailyReportRangeWeek,
		Date:     "2026-09-04",
		Timezone: "Asia/Shanghai",
	})
	require.NoError(t, err)
	require.Equal(t, types.ServiceDailyReportRangeWeek, weekReport.Range)
	require.Contains(t, weekReport.Title, "服务周报")
	require.Equal(t, 2, weekReport.DailySourceCount)
	require.False(t, weekReport.CanSupplement)
	require.Equal(t, profile.ID, weekReport.ProfileID)
	require.Equal(t, profile.Name, weekReport.ProfileName)
	require.Contains(t, weekReport.Content, "日粒度来源")
	require.Contains(t, weekReport.Content, day1Report.Title)
	require.Contains(t, weekReport.Content, day2Report.Title)
	require.Contains(t, weekReport.Content, "知识库审核")
	require.ElementsMatch(t, []string{day1Memory.ID, day2Memory.ID}, []string(weekReport.SourceMemoryIDs))
}
