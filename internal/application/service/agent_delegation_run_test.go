package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

type delegationRunBillingReader struct {
	interfaces.UsageBillingService
	ledgers []*types.BillingUsageLedgerSummary
}

func (r *delegationRunBillingReader) ListUsageLedgersByDelegationRun(
	_ context.Context,
	_ uint64,
	_ string,
	_ int,
) ([]*types.BillingUsageLedgerSummary, error) {
	return r.ledgers, nil
}

func TestAgentDelegationRunServiceAggregatesLinkedUsage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.AgentDelegationRun{}))

	billing := &delegationRunBillingReader{
		ledgers: []*types.BillingUsageLedgerSummary{
			{TenantUsageLedger: types.TenantUsageLedger{
				InputTokens:       100,
				CachedTokens:      20,
				OutputTokens:      30,
				ReasoningTokens:   4,
				CallCount:         1,
				BaseCostNanoUSD:   11,
				RatedCostNanoUSD:  13,
				BilledPointMicros: 7,
			}},
			{TenantUsageLedger: types.TenantUsageLedger{
				InputTokens:       200,
				CachedTokens:      50,
				OutputTokens:      60,
				ReasoningTokens:   8,
				CallCount:         2,
				BaseCostNanoUSD:   17,
				RatedCostNanoUSD:  19,
				BilledPointMicros: 23,
			}},
		},
	}
	service := NewAgentDelegationRunService(
		repository.NewAgentDelegationRunRepository(db),
		billing,
	)

	started, err := service.Start(context.Background(), types.AgentDelegationRunStartInput{
		TenantID:        7,
		ParentSessionID: "session-7",
		ParentMessageID: "message-9",
		MemberAgentID:   "member-agent-1",
		Task:            "整理客户反馈",
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentDelegationRunStatusRunning, started.Status)
	require.NotEmpty(t, started.ID)

	state := &types.AgentState{
		TurnUsage: types.TokenUsage{
			PromptTokens:     999,
			CachedTokens:     888,
			CompletionTokens: 777,
		},
	}
	finished, err := service.Finish(
		context.Background(),
		7,
		started.ID,
		state,
		"已完成",
		nil,
	)
	require.NoError(t, err)
	require.Equal(t, types.AgentDelegationRunStatusSucceeded, finished.Status)
	require.Equal(t, "session-7", finished.ParentSessionID)
	require.Equal(t, "member-agent-1", finished.MemberAgentID)
	require.Equal(t, int64(2), int64(finished.UsageLedgerCount))
	require.Equal(t, int64(300), finished.InputTokens)
	require.Equal(t, int64(70), finished.CachedTokens)
	require.Equal(t, int64(90), finished.OutputTokens)
	require.Equal(t, int64(12), finished.ReasoningTokens)
	require.Equal(t, int64(3), finished.CallCount)
	require.Equal(t, int64(28), finished.BaseCostNanoUSD)
	require.Equal(t, int64(32), finished.RatedCostNanoUSD)
	require.Equal(t, int64(30), finished.BilledPointMicros)
	require.Equal(t, "已完成", finished.OutputPreview)
	require.NotNil(t, finished.FinishedAt)
	require.GreaterOrEqual(t, finished.DurationMillis, int64(0))

	runs, err := service.List(context.Background(), 7, "session-7", "member-agent-1", 10)
	require.NoError(t, err)
	require.Len(t, runs, 1)
	require.Equal(t, finished.ID, runs[0].ID)

	otherTenantRuns, err := service.List(context.Background(), 8, "", "", 10)
	require.NoError(t, err)
	require.Empty(t, otherTenantRuns)
}

func TestAgentDelegationRunServiceRecordsFailure(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.AgentDelegationRun{}))

	service := NewAgentDelegationRunService(
		repository.NewAgentDelegationRunRepository(db),
		nil,
	)
	started, err := service.Start(context.Background(), types.AgentDelegationRunStartInput{
		TenantID:        7,
		ParentSessionID: "session-7",
		MemberAgentID:   "member-agent-2",
		Task:            "生成摘要",
	})
	require.NoError(t, err)

	finished, err := service.Finish(
		context.Background(),
		7,
		started.ID,
		nil,
		"",
		context.DeadlineExceeded,
	)
	require.NoError(t, err)
	require.Equal(t, types.AgentDelegationRunStatusFailed, finished.Status)
	require.Equal(t, "execution_failed", finished.ErrorCode)
	require.Equal(t, context.DeadlineExceeded.Error(), finished.ErrorMessage)
	require.Empty(t, finished.OutputPreview)
}
