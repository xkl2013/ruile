package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeAgentKnowledgeBaseService struct {
	interfaces.KnowledgeBaseService
	kb *types.KnowledgeBase
}

func (s *fakeAgentKnowledgeBaseService) GetKnowledgeBaseByID(context.Context, string) (*types.KnowledgeBase, error) {
	if s.kb == nil {
		return nil, errors.New("knowledge base not found")
	}
	return s.kb, nil
}

type fakeAgentKnowledgeService struct {
	interfaces.KnowledgeService
	knowledges []*types.Knowledge
	lastFilter types.KnowledgeListFilter
}

func (s *fakeAgentKnowledgeService) ListPagedKnowledgeByKnowledgeBaseID(
	_ context.Context,
	_ string,
	page *types.Pagination,
	filter types.KnowledgeListFilter,
) (*types.PageResult, error) {
	s.lastFilter = filter

	filtered := make([]*types.Knowledge, 0, len(s.knowledges))
	for _, knowledge := range s.knowledges {
		if filter.ParseStatus != "" && knowledge.ParseStatus != filter.ParseStatus {
			continue
		}
		filtered = append(filtered, knowledge)
	}
	return types.NewPageResult(int64(len(filtered)), page, filtered), nil
}

func TestGetKnowledgeBaseInfos_ExcludesUnprocessedDocuments(t *testing.T) {
	now := time.Now()
	knowledgeService := &fakeAgentKnowledgeService{
		knowledges: []*types.Knowledge{
			{
				ID:              "doc-processing",
				KnowledgeBaseID: "kb-1",
				Title:           "still parsing",
				FileName:        "processing.pdf",
				FileType:        "pdf",
				ParseStatus:     types.ParseStatusProcessing,
				CreatedAt:       now,
			},
			{
				ID:              "doc-completed",
				KnowledgeBaseID: "kb-1",
				Title:           "ready document",
				FileName:        "ready.pdf",
				FileType:        "pdf",
				ParseStatus:     types.ParseStatusCompleted,
				CreatedAt:       now.Add(-time.Minute),
			},
		},
	}
	service := &agentService{
		knowledgeBaseService: &fakeAgentKnowledgeBaseService{
			kb: &types.KnowledgeBase{
				ID:       "kb-1",
				Name:     "KB",
				Type:     types.KnowledgeBaseTypeDocument,
				TenantID: 1,
			},
		},
		knowledgeService: knowledgeService,
	}

	infos, err := service.getKnowledgeBaseInfos(context.Background(), []string{"kb-1"})

	require.NoError(t, err)
	require.Len(t, infos, 1)
	assert.Equal(t, types.ParseStatusCompleted, knowledgeService.lastFilter.ParseStatus)
	assert.Equal(t, 1, infos[0].DocCount)
	require.Len(t, infos[0].RecentDocs, 1)
	assert.Equal(t, "doc-completed", infos[0].RecentDocs[0].KnowledgeID)
}

func TestDeriveDelegatedAgentConfigUsesMemberPolicyWithoutNestedDelegation(t *testing.T) {
	thinking := true
	citations := false
	parent := &types.AgentConfig{
		MaxIterations:        12,
		Temperature:          0.8,
		WebSearchEnabled:     true,
		SkillsEnabled:        true,
		OrchestrationEnabled: true,
		MemberAgentIDs:       []string{"member-a"},
		SearchTargets:        types.SearchTargets{&types.SearchTarget{KnowledgeBaseID: "parent-kb"}},
	}
	member := &types.CustomAgent{
		Config: types.CustomAgentConfig{
			SystemPrompt:        "member prompt",
			AllowedTools:        []string{"thinking"},
			MaxIterations:       4,
			MaxCompletionTokens: 1024,
			Temperature:         0.3,
			WebSearchEnabled:    false,
			WebSearchProviderID: "provider-member",
			MCPSelectionMode:    "selected",
			MCPServices:         []string{"mcp-1"},
			Thinking:            &thinking,
			CitationEnabled:     &citations,
			LLMCallTimeout:      45,
			SkillsSelectionMode: "selected",
			SelectedSkills:      []string{"skill-a"},
		},
	}

	child := deriveDelegatedAgentConfig(parent, member)

	require.NotNil(t, child)
	assert.False(t, child.OrchestrationEnabled)
	assert.Empty(t, child.MemberAgentIDs)
	assert.Equal(t, "member prompt", child.SystemPrompt)
	assert.Equal(t, []string{"thinking"}, child.AllowedTools)
	assert.Equal(t, 4, child.MaxIterations)
	assert.Equal(t, 1024, child.MaxCompletionTokens)
	assert.Equal(t, 0.3, child.Temperature)
	assert.False(t, child.WebSearchEnabled)
	assert.Equal(t, "provider-member", child.WebSearchProviderID)
	assert.Equal(t, "selected", child.MCPSelectionMode)
	assert.Equal(t, []string{"mcp-1"}, child.MCPServices)
	assert.Same(t, &thinking, child.Thinking)
	assert.Same(t, &citations, child.CitationEnabled)
	assert.Equal(t, 45, child.LLMCallTimeout)
	assert.True(t, child.SkillsEnabled)
	assert.Equal(t, []string{"skill-a"}, child.AllowedSkills)
	assert.Equal(t, "parent-kb", child.SearchTargets[0].KnowledgeBaseID)
}

func TestDeriveDelegatedAgentConfigFiltersMemberKnowledgeBases(t *testing.T) {
	parent := &types.AgentConfig{
		KnowledgeBases: []string{"kb-parent", "kb-member"},
		SearchTargets: types.SearchTargets{
			&types.SearchTarget{KnowledgeBaseID: "kb-parent"},
			&types.SearchTarget{KnowledgeBaseID: "kb-member"},
		},
	}
	member := &types.CustomAgent{
		Config: types.CustomAgentConfig{
			KBSelectionMode: "selected",
			KnowledgeBases:  []string{"kb-member"},
		},
	}

	child := deriveDelegatedAgentConfig(parent, member)

	require.Equal(t, []string{"kb-member"}, child.KnowledgeBases)
	require.Len(t, child.SearchTargets, 1)
	require.Equal(t, "kb-member", child.SearchTargets[0].KnowledgeBaseID)
}

func TestMaterializeAgentSkillSnapshotsLoadsInstructionsAndReferences(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())

	dir, err := materializeAgentSkillSnapshots([]types.CustomAgentSkillSnapshot{
		{
			Name:    "demo-skill",
			Path:    "skills/demo-skill/SKILL.md",
			Content: "---\nname: demo-skill\ndescription: demo\n---\n\n# Demo\n",
		},
		{
			Name:    "demo-skill",
			Path:    "skills/demo-skill/references/ref.md",
			Content: "# Ref\n",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, dir)

	loader := skills.NewLoader([]string{dir})
	metadata, err := loader.DiscoverSkills()
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.Equal(t, "demo-skill", metadata[0].Name)

	skill, err := loader.LoadSkillInstructions("demo-skill")
	require.NoError(t, err)
	require.Contains(t, skill.Instructions, "# Demo")

	file, err := loader.LoadSkillFile("demo-skill", "references/ref.md")
	require.NoError(t, err)
	require.Equal(t, "# Ref\n", file.Content)
}

func TestBridgeDelegatedAgentEventsDoesNotForwardCompletion(t *testing.T) {
	childBus := event.NewEventBus()
	parentBus := event.NewEventBus()
	var received []event.Event
	for _, eventType := range []event.EventType{
		event.EventAgentThought,
		event.EventAgentToolCall,
		event.EventAgentComplete,
		event.EventAgentFinalAnswer,
	} {
		eventType := eventType
		parentBus.On(eventType, func(_ context.Context, evt event.Event) error {
			received = append(received, evt)
			return nil
		})
	}
	bridgeDelegatedAgentEvents(childBus, parentBus, "parent-session", "parent-session::member:a", "member-a")

	require.NoError(t, childBus.Emit(context.Background(), event.Event{
		ID:        "thought-1",
		Type:      event.EventAgentThought,
		SessionID: "parent-session::member:a",
		Data:      event.AgentThoughtData{Content: "working"},
	}))
	require.NoError(t, childBus.Emit(context.Background(), event.Event{
		ID:        "complete-1",
		Type:      event.EventAgentComplete,
		SessionID: "parent-session::member:a",
		Data:      event.AgentCompleteData{FinalAnswer: "member answer"},
	}))
	require.NoError(t, childBus.Emit(context.Background(), event.Event{
		ID:        "answer-1",
		Type:      event.EventAgentFinalAnswer,
		SessionID: "parent-session::member:a",
		Data:      event.AgentFinalAnswerData{Content: "member answer"},
	}))

	require.Len(t, received, 1)
	require.Equal(t, event.EventAgentThought, received[0].Type)
	require.Equal(t, "parent-session", received[0].SessionID)
	require.Equal(t, true, received[0].Metadata["delegated"])
	require.Equal(t, "member-a", received[0].Metadata["delegated_agent_id"])
}
