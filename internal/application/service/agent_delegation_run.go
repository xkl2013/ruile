package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const agentDelegationOutputPreviewRunes = 4000

var ErrAgentDelegationRunNotFound = errors.New("agent delegation run not found")

type agentDelegationRunService struct {
	repo    interfaces.AgentDelegationRunRepository
	billing interfaces.UsageBillingService
}

type delegationUsageLedgerReader interface {
	ListUsageLedgersByDelegationRun(
		ctx context.Context,
		tenantID uint64,
		delegationRunID string,
		limit int,
	) ([]*types.BillingUsageLedgerSummary, error)
}

func NewAgentDelegationRunService(
	repo interfaces.AgentDelegationRunRepository,
	billing interfaces.UsageBillingService,
) interfaces.AgentDelegationRunService {
	return &agentDelegationRunService{repo: repo, billing: billing}
}

func (s *agentDelegationRunService) Start(
	ctx context.Context,
	input types.AgentDelegationRunStartInput,
) (*types.AgentDelegationRun, error) {
	if s.repo == nil {
		return nil, errors.New("agent delegation run repository is not configured")
	}
	if input.TenantID == 0 {
		return nil, errors.New("tenant id is required")
	}
	if strings.TrimSpace(input.ParentSessionID) == "" {
		return nil, errors.New("parent session id is required")
	}
	if strings.TrimSpace(input.MemberAgentID) == "" {
		return nil, errors.New("member agent id is required")
	}
	now := time.Now().UTC()
	run := &types.AgentDelegationRun{
		ID:              uuid.NewString(),
		TenantID:        input.TenantID,
		ParentSessionID: strings.TrimSpace(input.ParentSessionID),
		ParentMessageID: strings.TrimSpace(input.ParentMessageID),
		MemberAgentID:   strings.TrimSpace(input.MemberAgentID),
		Task:            strings.TrimSpace(input.Task),
		Status:          types.AgentDelegationRunStatusRunning,
		StartedAt:       &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repo.Create(ctx, run); err != nil {
		return nil, fmt.Errorf("create agent delegation run: %w", err)
	}
	return run, nil
}

func (s *agentDelegationRunService) Finish(
	ctx context.Context,
	tenantID uint64,
	id string,
	state *types.AgentState,
	output string,
	runErr error,
) (*types.AgentDelegationRun, error) {
	if s.repo == nil {
		return nil, errors.New("agent delegation run repository is not configured")
	}
	run, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, fmt.Errorf("load agent delegation run: %w", err)
	}
	if run == nil {
		return nil, ErrAgentDelegationRunNotFound
	}

	finishedAt := time.Now().UTC()
	run.FinishedAt = &finishedAt
	if run.StartedAt != nil {
		run.DurationMillis = finishedAt.Sub(*run.StartedAt).Milliseconds()
	}
	if state != nil {
		run.InputTokens = int64(state.TurnUsage.PromptTokens)
		run.CachedTokens = int64(state.TurnUsage.CachedTokens)
		run.OutputTokens = int64(state.TurnUsage.CompletionTokens)
		run.CallCount = 1
	}
	if runErr != nil {
		run.Status = types.AgentDelegationRunStatusFailed
		run.ErrorCode = "execution_failed"
		run.ErrorMessage = truncateDelegationText(runErr.Error(), agentDelegationOutputPreviewRunes)
	} else {
		run.Status = types.AgentDelegationRunStatusSucceeded
		run.OutputPreview = truncateDelegationText(output, agentDelegationOutputPreviewRunes)
	}

	if reader, ok := s.billing.(delegationUsageLedgerReader); ok {
		ledgers, ledgerErr := reader.ListUsageLedgersByDelegationRun(
			context.WithoutCancel(ctx),
			tenantID,
			run.ID,
			500,
		)
		if ledgerErr != nil {
			logger.Warnf(ctx, "failed to aggregate delegated Agent usage: %v", ledgerErr)
		} else if len(ledgers) > 0 {
			run.UsageLedgerCount = len(ledgers)
			run.InputTokens = 0
			run.CachedTokens = 0
			run.OutputTokens = 0
			run.ReasoningTokens = 0
			run.CallCount = 0
			run.BaseCostNanoUSD = 0
			run.RatedCostNanoUSD = 0
			run.BilledPointMicros = 0
			for _, ledger := range ledgers {
				if ledger == nil {
					continue
				}
				run.InputTokens += ledger.InputTokens
				run.CachedTokens += ledger.CachedTokens
				run.OutputTokens += ledger.OutputTokens
				run.ReasoningTokens += ledger.ReasoningTokens
				run.CallCount += ledger.CallCount
				run.BaseCostNanoUSD += ledger.BaseCostNanoUSD
				run.RatedCostNanoUSD += ledger.RatedCostNanoUSD
				run.BilledPointMicros += ledger.BilledPointMicros
			}
		}
	}
	run.UpdatedAt = finishedAt
	if err := s.repo.Update(context.WithoutCancel(ctx), run); err != nil {
		return nil, fmt.Errorf("update agent delegation run: %w", err)
	}
	return run, nil
}

func (s *agentDelegationRunService) Get(
	ctx context.Context,
	tenantID uint64,
	id string,
) (*types.AgentDelegationRun, error) {
	run, err := s.repo.GetByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return nil, ErrAgentDelegationRunNotFound
	}
	return run, nil
}

func (s *agentDelegationRunService) List(
	ctx context.Context,
	tenantID uint64,
	parentSessionID string,
	memberAgentID string,
	limit int,
) ([]*types.AgentDelegationRun, error) {
	return s.repo.List(ctx, tenantID, parentSessionID, memberAgentID, limit)
}

func truncateDelegationText(value string, maxRunes int) string {
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= maxRunes {
		return string(runes)
	}
	return string(runes[:maxRunes]) + "..."
}
