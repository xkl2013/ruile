package tools

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDelegateAgentToolRunsAllowedMembersWithConcurrencyLimit(t *testing.T) {
	var running atomic.Int32
	var maxRunning atomic.Int32
	tool := NewDelegateAgentTool(
		[]string{"member-a", "member-b", "member-c"},
		2,
		func(_ context.Context, agentID, task string) (string, error) {
			current := running.Add(1)
			for {
				previous := maxRunning.Load()
				if current <= previous || maxRunning.CompareAndSwap(previous, current) {
					break
				}
			}
			time.Sleep(20 * time.Millisecond)
			running.Add(-1)
			return agentID + ":" + task, nil
		},
	)

	result, err := tool.Execute(context.Background(), json.RawMessage(`{
		"items": [
			{"agent_id":"member-a","task":"a"},
			{"agent_id":"member-b","task":"b"},
			{"agent_id":"member-c","task":"c"},
			{"agent_id":"unknown","task":"d"}
		]
	}`))
	require.NoError(t, err)
	require.True(t, result.Success)
	require.Equal(t, int32(2), maxRunning.Load())
	require.Contains(t, result.Output, "member-a:a")
	require.Contains(t, result.Output, "member-b:b")
	require.Contains(t, result.Output, "member-c:c")
	require.Contains(t, result.Output, "not in the member allowlist")
}

func TestDelegateAgentToolRejectsOversizedBatch(t *testing.T) {
	tool := NewDelegateAgentTool([]string{"member-a"}, 1, func(context.Context, string, string) (string, error) {
		return "unused", nil
	})
	input := `{"items":[
		{"agent_id":"member-a","task":"1"},
		{"agent_id":"member-a","task":"2"},
		{"agent_id":"member-a","task":"3"},
		{"agent_id":"member-a","task":"4"},
		{"agent_id":"member-a","task":"5"},
		{"agent_id":"member-a","task":"6"},
		{"agent_id":"member-a","task":"7"},
		{"agent_id":"member-a","task":"8"},
		{"agent_id":"member-a","task":"9"}
	]}`
	result, err := tool.Execute(context.Background(), json.RawMessage(input))
	require.NoError(t, err)
	require.False(t, result.Success)
	require.Contains(t, result.Error, "at most")
}
