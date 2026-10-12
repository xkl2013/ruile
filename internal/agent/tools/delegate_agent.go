package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/utils"
)

const (
	defaultMaxParallelMembers = 3
	maxDelegationItems        = 8
	maxDelegationTaskRunes    = 12000
)

type DelegateAgentItem struct {
	AgentID string `json:"agent_id" jsonschema:"Configured member agent ID"`
	Task    string `json:"task" jsonschema:"Self-contained subtask for the member agent"`
}

type DelegateAgentInput struct {
	Items []DelegateAgentItem `json:"items" jsonschema:"Independent subtasks to delegate"`
}

type DelegateAgentRunner func(context.Context, string, string) (string, error)

type DelegateAgentTool struct {
	BaseTool
	allowedIDs  map[string]struct{}
	maxParallel int
	taskTimeout time.Duration
	run         DelegateAgentRunner
}

func NewDelegateAgentTool(
	allowedIDs []string,
	maxParallel int,
	run DelegateAgentRunner,
) *DelegateAgentTool {
	return NewDelegateAgentToolWithTimeout(allowedIDs, maxParallel, 0, run)
}

func NewDelegateAgentToolWithTimeout(
	allowedIDs []string,
	maxParallel int,
	taskTimeout time.Duration,
	run DelegateAgentRunner,
) *DelegateAgentTool {
	if maxParallel <= 0 {
		maxParallel = defaultMaxParallelMembers
	}
	if maxParallel > maxDelegationItems {
		maxParallel = maxDelegationItems
	}
	allowed := make(map[string]struct{}, len(allowedIDs))
	for _, id := range allowedIDs {
		if id = strings.TrimSpace(id); id != "" {
			allowed[id] = struct{}{}
		}
	}
	return &DelegateAgentTool{
		BaseTool: NewBaseTool(
			ToolDelegateAgent,
			"Delegate independent subtasks to configured member agents. Use only for bounded, parallelizable work; the main agent remains responsible for synthesis.",
			utils.GenerateSchema[DelegateAgentInput](),
		),
		allowedIDs:  allowed,
		maxParallel: maxParallel,
		taskTimeout: taskTimeout,
		run:         run,
	}
}

func (t *DelegateAgentTool) Execute(ctx context.Context, args json.RawMessage) (*types.ToolResult, error) {
	if t.run == nil {
		return &types.ToolResult{Success: false, Error: "member agent runner is not configured"}, nil
	}
	var input DelegateAgentInput
	if err := json.Unmarshal(args, &input); err != nil {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("invalid delegation input: %v", err)}, nil
	}
	if len(input.Items) == 0 {
		return &types.ToolResult{Success: false, Error: "items must not be empty"}, nil
	}
	if len(input.Items) > maxDelegationItems {
		return &types.ToolResult{Success: false, Error: fmt.Sprintf("at most %d member tasks are allowed", maxDelegationItems)}, nil
	}

	results := make([]map[string]any, len(input.Items))
	sem := make(chan struct{}, t.maxParallel)
	var wg sync.WaitGroup
	for index, item := range input.Items {
		index, item := index, item
		item.AgentID = strings.TrimSpace(item.AgentID)
		item.Task = strings.TrimSpace(item.Task)
		if _, ok := t.allowedIDs[item.AgentID]; !ok {
			results[index] = map[string]any{"agent_id": item.AgentID, "success": false, "error": "agent is not in the member allowlist"}
			continue
		}
		if item.Task == "" {
			results[index] = map[string]any{"agent_id": item.AgentID, "success": false, "error": "task is required"}
			continue
		}
		if len([]rune(item.Task)) > maxDelegationTaskRunes {
			results[index] = map[string]any{"agent_id": item.AgentID, "success": false, "error": "task is too long"}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			runCtx := ctx
			cancel := func() {}
			if t.taskTimeout > 0 {
				runCtx, cancel = context.WithTimeout(ctx, t.taskTimeout)
			}
			defer cancel()
			output, err := t.run(runCtx, item.AgentID, item.Task)
			result := map[string]any{"agent_id": item.AgentID, "success": err == nil, "output": output}
			if err != nil {
				result["error"] = err.Error()
			}
			results[index] = result
		}()
	}
	wg.Wait()

	successCount := 0
	var builder strings.Builder
	for index, result := range results {
		if result == nil {
			continue
		}
		success, _ := result["success"].(bool)
		if success {
			successCount++
		}
		builder.WriteString(fmt.Sprintf("[%d] agent=%s success=%t\n", index+1, result["agent_id"], success))
		if output, _ := result["output"].(string); output != "" {
			builder.WriteString(output)
			builder.WriteString("\n")
		}
		if errText, _ := result["error"].(string); errText != "" {
			builder.WriteString("error: ")
			builder.WriteString(errText)
			builder.WriteString("\n")
		}
	}
	return &types.ToolResult{
		Success: successCount > 0,
		Output:  builder.String(),
		Data: map[string]any{
			"results":       results,
			"success_count": successCount,
			"total_count":   len(results),
		},
	}, nil
}
