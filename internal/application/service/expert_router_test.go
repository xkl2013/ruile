package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestExpertRouterScoresRelevantChinesePrompt(t *testing.T) {
	expert := &types.PublishedExpert{
		DisplayName:        "童创",
		Description:        "幼儿园亲子活动策划专家，负责活动方案、流程和安全预案",
		Domain:             "education",
		PackageDisplayName: "幼儿园活动专家包",
	}

	score := scorePublishedExpert(
		"请制定一份幼儿园国庆亲子运动会活动方案，包含流程、物料和安全预案",
		expertRouterTokens("请制定一份幼儿园国庆亲子运动会活动方案，包含流程、物料和安全预案"),
		expert,
	)
	require.Greater(t, score, 0.1)
}

func TestAgentRunAutoRouteRejectsUnrelatedPrompt(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "auto-route-user"

	runService, _, _, db, chatModel := newExpertAgentRunTestService(t, nil, nil)
	createPublishedExpertTestDefinition(t, db, tenantID)
	chatModel.chatResponses = []string{`{"selected_expert_id":"","confidence":0.05,"reason":"没有适合处理天气问题的专家","requires_confirmation":false}`}

	_, err := runService.EnqueuePublishedExpertAutoRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		Prompt:  "今天北京天气怎么样",
		ModelID: "chat-1",
	})
	require.ErrorIs(t, err, ErrAgentRunNoExpertMatch)
}

func TestAgentRunAutoRouteSelectsPublishedExpert(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "auto-route-user"

	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(t, nil, nil)
	pkg, definition := createPublishedExpertTestDefinition(t, db, tenantID)
	chatModel.chatResponses = []string{
		fmt.Sprintf(
			`{"selected_expert_id":%q,"confidence":0.96,"reason":"用户需要幼儿园活动策划，童创负责该领域","requires_confirmation":false}`,
			definition.ID,
		),
	}

	queued, err := runService.EnqueuePublishedExpertAutoRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		Prompt:  "请制定一份幼儿园国庆亲子运动会活动方案",
		ModelID: "chat-1",
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusQueued, queued.Status)
	require.Equal(t, types.AgentRouteModeAuto, queued.Input["route_mode"])
	require.Equal(t, definition.AgentID, queued.AgentRef)
	require.Equal(t, definition.Version, queued.AgentVersion)
	require.Len(t, enqueuer.tasks, 1)

	routing, ok := queued.Input["routing_decision"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, types.AgentRouteModeAuto, routing["route_mode"])
	require.Equal(t, definition.ID, routing["selected_expert_id"])

	events, err := runRepo.ListEvents(ctx, queued.ID, 0, 20)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, types.AgentRunEventTypeRunQueued, events[0].EventType)
	require.Equal(t, types.AgentRunEventTypeExpertRouting, events[1].EventType)
	require.Equal(t, pkg.ID, events[1].Payload["candidates"].([]any)[0].(map[string]any)["package_id"])
}

func TestAgentRunAutoRouteRequiresConfirmationForLowConfidenceSelection(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "auto-route-user"

	runService, _, enqueuer, db, chatModel := newExpertAgentRunTestService(t, nil, nil)
	_, definition := createPublishedExpertTestDefinition(t, db, tenantID)
	chatModel.chatResponses = []string{
		fmt.Sprintf(
			`{"selected_expert_id":%q,"confidence":0.62,"reason":"任务与幼儿园活动策划领域相关，但信息不完整","requires_confirmation":true}`,
			definition.ID,
		),
	}

	_, err := runService.EnqueuePublishedExpertAutoRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		Prompt:  "先做一个幼儿园节日活动方案框架",
		ModelID: "chat-1",
	})
	require.ErrorIs(t, err, ErrAgentRunRouteConfirm)
	require.Empty(t, enqueuer.tasks)
}

func TestAgentRunAutoRouteRunsConfirmedSelection(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "auto-route-user"

	runService, runRepo, enqueuer, db, chatModel := newExpertAgentRunTestService(t, nil, nil)
	_, definition := createPublishedExpertTestDefinition(t, db, tenantID)
	chatModel.chatResponses = []string{
		fmt.Sprintf(
			`{"selected_expert_id":%q,"confidence":0.62,"reason":"任务与幼儿园活动策划领域相关，但信息不完整","requires_confirmation":true}`,
			definition.ID,
		),
	}

	queued, err := runService.EnqueuePublishedExpertAutoRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		Prompt:        "先做一个幼儿园节日活动方案框架",
		ModelID:       "chat-1",
		UserConfirmed: true,
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusQueued, queued.Status)
	require.Len(t, enqueuer.tasks, 1)

	routing, ok := queued.Input["routing_decision"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, routing["requires_confirmation"])
	require.Equal(t, 0.62, routing["confidence"])
	require.Equal(t, definition.ID, routing["selected_expert_id"])
	require.Len(t, chatModel.chatResponses, 0)

	events, err := runRepo.ListEvents(ctx, queued.ID, 0, 20)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, types.AgentRunEventTypeExpertRouting, events[1].EventType)
}

func TestAgentRunAutoRouteAcceptsConfirmedRouteDecisionWithoutRerouting(t *testing.T) {
	ctx := context.Background()
	const tenantID uint64 = 9
	const userID = "auto-route-user"

	runService, _, enqueuer, db, chatModel := newExpertAgentRunTestService(t, nil, nil)
	_, definition := createPublishedExpertTestDefinition(t, db, tenantID)

	queued, err := runService.EnqueuePublishedExpertAutoRun(ctx, tenantID, userID, types.ExpertAgentTestInput{
		Prompt:        "先做一个幼儿园节日活动方案框架",
		ModelID:       "chat-1",
		UserConfirmed: true,
		RoutingDecision: types.JSONMap{
			"route_mode":            types.AgentRouteModeAuto,
			"selected_expert_id":    definition.ID,
			"selected_version":      definition.Version,
			"confidence":            0.62,
			"routing_reason":        "用户确认使用该专家",
			"requires_confirmation": true,
			"candidates": []any{
				map[string]any{
					"definition_id": definition.ID,
					"confidence":    0.62,
					"score":         0.62,
				},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.AgentRunStatusQueued, queued.Status)
	require.Len(t, enqueuer.tasks, 1)
	require.Empty(t, chatModel.chatResponses)
}
