package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCustomAgentRepositoryListsGlobalBuiltinForEveryWorkspace(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(
		fmt.Sprintf("file:agent-global-%d?mode=memory&cache=shared", time.Now().UnixNano()),
	), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(customAgentsTestDDL).Error)

	repo := NewCustomAgentRepository(db)
	ctx := context.Background()
	agents := []*types.CustomAgent{
		{
			ID:        types.BuiltinQuickAnswerID,
			Name:      "Global Quick Answer",
			IsBuiltin: true,
			TenantID:  types.SystemAgentTenantID,
		},
		{
			ID:        types.BuiltinQuickAnswerID,
			Name:      "Legacy Workspace Override",
			IsBuiltin: true,
			TenantID:  10003,
		},
		{
			ID:       "workspace-agent",
			Name:     "Workspace Agent",
			TenantID: 10003,
		},
		{
			ID:       "other-agent",
			Name:     "Other Agent",
			TenantID: 10004,
		},
	}
	for _, agent := range agents {
		require.NoError(t, repo.CreateAgent(ctx, agent))
	}

	got, err := repo.ListAgentsByTenantID(ctx, 10003)
	require.NoError(t, err)
	require.Len(t, got, 2)

	byID := make(map[string]*types.CustomAgent, len(got))
	for _, agent := range got {
		byID[agent.ID] = agent
	}
	require.Equal(t, "Global Quick Answer", byID[types.BuiltinQuickAnswerID].Name)
	require.Equal(t, types.SystemAgentTenantID, byID[types.BuiltinQuickAnswerID].TenantID)
	require.Equal(t, uint64(10003), byID["workspace-agent"].TenantID)
	require.NotContains(t, byID, "other-agent")
}

func TestCustomAgentRepositoryUpdatesGlobalBuiltinWithZeroTenantID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(
		fmt.Sprintf("file:agent-global-update-%d?mode=memory&cache=shared", time.Now().UnixNano()),
	), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(customAgentsTestDDL).Error)

	repo := NewCustomAgentRepository(db)
	ctx := context.Background()
	agent := &types.CustomAgent{
		ID:        types.BuiltinQuickAnswerID,
		Name:      "Global Quick Answer",
		IsBuiltin: true,
		TenantID:  types.SystemAgentTenantID,
		Config: types.CustomAgentConfig{
			WebSearchEnabled: true,
		},
	}
	require.NoError(t, repo.CreateAgent(ctx, agent))

	agent.Config.WebSearchEnabled = false
	require.NoError(t, repo.UpdateAgent(ctx, agent))

	updated, err := repo.GetAgentByID(ctx, types.BuiltinQuickAnswerID, types.SystemAgentTenantID)
	require.NoError(t, err)
	require.False(t, updated.Config.WebSearchEnabled)

	var count int64
	require.NoError(t, db.Model(&types.CustomAgent{}).
		Where("id = ? AND tenant_id = ?", types.BuiltinQuickAnswerID, types.SystemAgentTenantID).
		Count(&count).Error)
	require.Equal(t, int64(1), count)
}
