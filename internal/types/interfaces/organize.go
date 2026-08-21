package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type OrganizeRepository interface {
	CreateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, memory *types.OrganizeMemory) error
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)
	CountMemoriesByKind(ctx context.Context, tenantID uint64, userID string) (map[string]int64, error)
	CountMemoriesByIDs(ctx context.Context, tenantID uint64, userID string, ids []string) (int64, error)

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
	CreateMemory(ctx context.Context, tenantID uint64, userID string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	GetMemory(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeMemory, error)
	UpdateMemory(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeMemoryInput) (*types.OrganizeMemory, error)
	DeleteMemory(ctx context.Context, tenantID uint64, userID, id string) error
	ListMemories(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeMemory, int64, error)

	CreateOutput(ctx context.Context, tenantID uint64, userID string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	CreateOutputFromUpload(ctx context.Context, tenantID uint64, userID, fileName, mimeType string, data []byte) (*types.OrganizeOutput, error)
	GetOutput(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeOutput, error)
	UpdateOutput(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeOutputInput) (*types.OrganizeOutput, error)
	DeleteOutput(ctx context.Context, tenantID uint64, userID, id string) error
	ListOutputs(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeOutput, int64, error)

	CreateSproutReport(ctx context.Context, tenantID uint64, userID string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	GetSproutReport(ctx context.Context, tenantID uint64, userID, id string) (*types.OrganizeSproutReport, error)
	UpdateSproutReport(ctx context.Context, tenantID uint64, userID, id string, input types.OrganizeSproutReportInput) (*types.OrganizeSproutReport, error)
	DeleteSproutReport(ctx context.Context, tenantID uint64, userID, id string) error
	ListSproutReports(ctx context.Context, query types.OrganizeListQuery) ([]*types.OrganizeSproutReport, int64, error)

	GetOverview(ctx context.Context, tenantID uint64, userID string) (*types.OrganizeOverview, error)
}
