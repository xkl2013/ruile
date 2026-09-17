package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/hibiken/asynq"
)

// AgentRunRepository owns only durable execution state. Domain artifacts are
// persisted by the service that produced them.
type AgentRunRepository interface {
	Create(ctx context.Context, run *types.AgentRun) error
	GetByID(ctx context.Context, id string) (*types.AgentRun, error)
	GetByIDForUser(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentRun, error)
	FindActiveByIdempotency(ctx context.Context, tenantID uint64, userID, runType, key string) (*types.AgentRun, error)
	SetTaskID(ctx context.Context, id, taskID string) error
	Claim(ctx context.Context, id string, startedAt time.Time) (bool, error)
	MarkRetry(ctx context.Context, id, code, message string) error
	MarkSucceeded(ctx context.Context, id, profileID string, result types.JSONMap, finishedAt time.Time) (bool, error)
	MarkFailed(ctx context.Context, id, code, message string, finishedAt time.Time) (bool, error)
	MarkFailedWithResult(ctx context.Context, id string, result types.JSONMap, code, message string, finishedAt time.Time) (bool, error)
	CancelQueued(ctx context.Context, tenantID uint64, userID, id string, finishedAt time.Time) (bool, error)
}

type AgentRunService interface {
	EnqueueMemoryExtraction(ctx context.Context, tenantID uint64, userID, memoryID string) (*types.AgentRun, error)
	EnqueueDailyReport(ctx context.Context, tenantID uint64, userID string, input types.ServiceDailyReportInput) (*types.AgentRun, error)
	EnqueueExpertTest(ctx context.Context, tenantID uint64, userID string, input types.ExpertAgentTestInput) (*types.AgentRun, error)
	GetAgentRun(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentRun, error)
	CancelAgentRun(ctx context.Context, tenantID uint64, userID, id string) (*types.AgentRun, error)
	ProcessAgentRun(ctx context.Context, task *asynq.Task) error
}
