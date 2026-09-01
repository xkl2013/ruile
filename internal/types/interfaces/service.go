package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type ServiceRepository interface {
	CreateWorkProfile(ctx context.Context, profile *types.UserWorkProfile) error
	UpdateWorkProfile(ctx context.Context, profile *types.UserWorkProfile) error
	GetWorkProfile(ctx context.Context, tenantID uint64, id string) (*types.UserWorkProfile, error)
	GetDefaultWorkProfile(ctx context.Context, tenantID uint64, userID string) (*types.UserWorkProfile, error)
	ListWorkProfiles(ctx context.Context, tenantID uint64, userID string) ([]*types.UserWorkProfile, error)

	ReplaceAgentSettings(ctx context.Context, tenantID uint64, profileID string, settings []*types.WorkProfileAgentSetting) error
	ListAgentSettings(ctx context.Context, tenantID uint64, profileID string, onlyEnabled bool) ([]*types.WorkProfileAgentSetting, error)

	FindSubjectByKey(ctx context.Context, tenantID uint64, ownerUserID, subjectKey string) (*types.ServiceSubject, error)
	GetSubject(ctx context.Context, tenantID uint64, ownerUserID, id string) (*types.ServiceSubject, error)
	ListSubjects(ctx context.Context, query types.ServiceCustomerSpaceListQuery) ([]*types.ServiceSubject, int64, error)
	UpsertSubject(ctx context.Context, subject *types.ServiceSubject) error

	UpsertWorkDocWithLinks(ctx context.Context, doc *types.AgentWorkDoc, links []*types.AgentWorkDocMemoryLink) error
	ListWorkDocsBySubject(ctx context.Context, tenantID uint64, profileID, subjectID string) ([]*types.AgentWorkDoc, error)
	GetDailyReportDoc(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentWorkDoc, error)
	ListDailyReportDocs(ctx context.Context, query types.ServiceDailyReportListQuery) ([]*types.AgentWorkDoc, int64, error)

	UpsertReminder(ctx context.Context, reminder *types.ServiceReminder) error
	GetReminder(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceReminder, error)
	ListReminders(ctx context.Context, query types.ServiceListQuery) ([]*types.ServiceReminder, int64, error)
	UpdateReminderStatus(ctx context.Context, tenantID uint64, userID, id, status string) error
	CountRemindersByStatus(ctx context.Context, tenantID uint64, userID, profileID string) (map[string]int64, error)

	CreateActionDraft(ctx context.Context, draft *types.AgentActionDraft) error
	GetActionDraft(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentActionDraft, error)
	ListActionDrafts(ctx context.Context, tenantID uint64, userID, reminderID string) ([]*types.AgentActionDraft, error)
	UpdateActionDraftStatus(ctx context.Context, tenantID uint64, userID, id, status string) error
	CreateActionLog(ctx context.Context, log *types.AgentActionLog) error
}

type ServiceService interface {
	GetBootstrap(ctx context.Context, tenantID uint64, userID string) (*types.ServiceBootstrap, error)
	ListAgentTemplates(ctx context.Context) []types.ServiceAgentTemplate
	RefreshUserService(ctx context.Context, tenantID uint64, userID string) (*types.ServiceBootstrap, error)
	ExtractMemory(ctx context.Context, tenantID uint64, userID, memoryID string) (*types.ServiceMemoryExtraction, error)
	GenerateDailyReport(ctx context.Context, tenantID uint64, userID string, input types.ServiceDailyReportInput) (*types.ServiceDailyReport, error)
	GetDailyReport(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceDailyReport, error)
	ListDailyReports(ctx context.Context, query types.ServiceDailyReportListQuery) ([]*types.ServiceDailyReport, int64, error)
	ListCustomerSpaces(ctx context.Context, query types.ServiceCustomerSpaceListQuery) ([]*types.ServiceCustomerSpace, int64, error)
	GetCustomerSpace(ctx context.Context, tenantID uint64, userID, id, profileID string) (*types.ServiceCustomerSpaceDetail, error)

	ListReminders(ctx context.Context, query types.ServiceListQuery) ([]*types.ServiceReminder, int64, error)
	GetReminder(ctx context.Context, tenantID uint64, userID, id string) (*types.ServiceReminder, error)
	UpdateReminderStatus(ctx context.Context, tenantID uint64, userID, id, status string) (*types.ServiceReminder, error)

	ListWorkProfiles(ctx context.Context, tenantID uint64, userID string) ([]*types.UserWorkProfile, error)
	CreateWorkProfile(ctx context.Context, tenantID uint64, operatorUserID string, input types.ServiceWorkProfileInput) (*types.UserWorkProfile, error)
	UpdateWorkProfile(ctx context.Context, tenantID uint64, operatorUserID, id string, input types.ServiceWorkProfileInput) (*types.UserWorkProfile, error)
	ReplaceAgentSettings(ctx context.Context, tenantID uint64, operatorUserID, profileID string, input types.WorkProfileAgentSettingsInput) ([]*types.WorkProfileAgentSetting, error)
	ListAgentSettings(ctx context.Context, tenantID uint64, profileID string, onlyEnabled bool) ([]*types.WorkProfileAgentSetting, error)

	CreateActionDraft(ctx context.Context, tenantID uint64, userID, reminderID string, input types.AgentActionDraftInput) (*types.AgentActionDraft, error)
	ListActionDrafts(ctx context.Context, tenantID uint64, userID, reminderID string) ([]*types.AgentActionDraft, error)
	UpdateActionDraftStatus(ctx context.Context, tenantID uint64, userID, id, status string) (*types.AgentActionDraft, error)
}
