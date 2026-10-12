package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type AgentDelegationRunRepository interface {
	Create(ctx context.Context, run *types.AgentDelegationRun) error
	GetByID(ctx context.Context, tenantID uint64, id string) (*types.AgentDelegationRun, error)
	Update(ctx context.Context, run *types.AgentDelegationRun) error
	List(
		ctx context.Context,
		tenantID uint64,
		parentSessionID string,
		memberAgentID string,
		limit int,
	) ([]*types.AgentDelegationRun, error)
}

type AgentDelegationRunService interface {
	Start(ctx context.Context, input types.AgentDelegationRunStartInput) (*types.AgentDelegationRun, error)
	Finish(
		ctx context.Context,
		tenantID uint64,
		id string,
		state *types.AgentState,
		output string,
		runErr error,
	) (*types.AgentDelegationRun, error)
	Get(ctx context.Context, tenantID uint64, id string) (*types.AgentDelegationRun, error)
	List(
		ctx context.Context,
		tenantID uint64,
		parentSessionID string,
		memberAgentID string,
		limit int,
	) ([]*types.AgentDelegationRun, error)
}
