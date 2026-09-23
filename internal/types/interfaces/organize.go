package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

type OrganizeRepository interface {
	ListTemplates(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeTemplate, error)
	GetTemplate(ctx context.Context, tenantID uint64, userID, key string) (*types.OrganizeTemplate, error)
	GetTemplateVersion(ctx context.Context, templateID, version string) (*types.OrganizeTemplateVersion, error)

	CreateConfig(ctx context.Context, config *types.OrganizeConfig) error
	GetConfig(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeConfig, error)
	UpdateConfig(ctx context.Context, config *types.OrganizeConfig) error
	DeleteConfig(ctx context.Context, tenantID uint64, userID, id string) error
	ListConfigs(ctx context.Context, query types.OrganizeConfigQuery) ([]*types.OrganizeConfig, int64, error)
	ListDueConfigs(ctx context.Context, now time.Time, limit int) ([]*types.OrganizeConfig, error)

	CreateJob(ctx context.Context, job *types.OrganizeJob) error
	GetJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	UpdateJob(ctx context.Context, job *types.OrganizeJob) error
	ListJobs(ctx context.Context, query types.OrganizeJobQuery) ([]*types.OrganizeJob, int64, error)
	ListMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) ([]*types.OrganizeMemory, error)

	CreateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	GetTenantMemory(ctx context.Context, tenantID uint64, id string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)
	CountMemoriesByKind(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)
	CountMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) (int64, error)
	CountTenantMemoriesByIDs(ctx context.Context, tenantID uint64, ids []string) (int64, error)

	CreateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error
	GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error)
	UpdateOutput(ctx context.Context, output *types.OrganizeOutput, memoryIDs []string) error
	DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error
	ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)
	CountOutputsByStatus(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)

	CreateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error
	GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error)
	UpdateSproutReport(ctx context.Context, report *types.OrganizeSproutReport, memoryIDs []string) error
	DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error
	ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error)
	CountSproutReportsByStage(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)
}

type OrganizeService interface {
	ListTemplates(ctx context.Context, tenantID uint64, userID string) ([]*types.OrganizeTemplate, error)
	GetTemplate(ctx context.Context, tenantID uint64, userID, key string) (*types.OrganizeTemplate, error)
	ListExperts(ctx context.Context, tenantID uint64, userID string) ([]types.OrganizeExpert, error)

	CreateConfig(ctx context.Context, tenantID uint64, userID string, input types.OrganizeConfigInput) (*types.OrganizeConfig, error)
	GetConfig(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeConfig, error)
	UpdateConfig(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeConfigInput) (*types.OrganizeConfig, error)
	DeleteConfig(ctx context.Context, tenantID uint64, userID, id string) error
	ListConfigs(ctx context.Context, query types.OrganizeConfigQuery) ([]*types.OrganizeConfig, int64, error)
	RunConfig(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeJobInput) (*types.OrganizeJob, error)

	CreateJob(ctx context.Context, tenantID uint64, userID string, input types.OrganizeJobInput) (*types.OrganizeJob, error)
	GetJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	ListJobs(ctx context.Context, query types.OrganizeJobQuery) ([]*types.OrganizeJob, int64, error)
	RetryJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	CancelJob(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeJob, error)
	ProcessOrganizeJob(ctx context.Context, task *asynq.Task) error
	RunDueConfigs(ctx context.Context, now time.Time) error

	CreateMemory(ctx context.Context, tenantID uint64, userID string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	CreateMemoryFromUpload(ctx context.Context, tenantID uint64, userID, fileName, mimeType string, data []byte, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)
	ProcessMemoryTranscribe(ctx context.Context, task *asynq.Task) error

	CreateOutput(ctx context.Context, tenantID uint64, userID string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	CreateOutputFromUpload(ctx context.Context, tenantID uint64, userID, fileName, mimeType string, data []byte) (*types.OrganizeOutput, error)
	GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error)
	UpdateOutput(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error
	ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)

	CreateSproutReport(ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	CreateSproutReportFromMemory(ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutFromMemoryInput) (*types.OrganizeSproutReport, error)
	GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error)
	UpdateSproutReport(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error
	ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error)

	GetDiscover(ctx context.Context, tenantID uint64, userID string, query types.OrganizeDiscoverQuery) (*types.OrganizeDiscover, error)
	GetOverview(ctx context.Context, tenantID uint64, userID string) (*types.OrganizeOverview, error)
}
