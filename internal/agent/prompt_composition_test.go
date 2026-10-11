package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/tools"
	"github.com/Tencent/WeKnora/internal/event"
	"github.com/Tencent/WeKnora/internal/llmreference"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestBuildSystemPromptSectionsPreservesRuntimeGuidanceForCustomTemplate(t *testing.T) {
	const custom = "只输出简洁的业务结论。"
	sections := BuildSystemPromptSections(
		[]*KnowledgeBaseInfo{{ID: "kb-1"}},
		false,
		&BuildSystemPromptOptions{
			SelectedTools: []string{tools.ToolKnowledgeSearch, tools.ToolListKnowledgeChunks},
			Instructions: []SystemPromptSection{
				{Name: "service_space", Content: "遵守当前服务空间的业务边界。"},
			},
		},
		custom,
	)

	require.Len(t, sections, 3)
	require.Equal(t, "base", sections[0].Name)
	require.Equal(t, custom, sections[0].Content)
	require.Equal(t, "grounding", sections[1].Name)
	require.Equal(t, "service_space", sections[2].Name)

	prompt := renderSystemPromptSections(sections)
	require.True(t, strings.HasPrefix(prompt, custom))
	require.Contains(t, prompt, "Content grounding")
	require.Contains(t, prompt, "knowledge_search, list_knowledge_chunks")
	require.Contains(t, prompt, "<service_space_business_instructions>")
	require.Contains(t, prompt, "遵守当前服务空间的业务边界。")
}

func TestAgentEngineKeepsCitationsAndBusinessInstructionsWithCustomPrompt(t *testing.T) {
	citations := true
	engine := NewAgentEngine(
		&types.AgentConfig{
			CitationEnabled: &citations,
			BusinessInstructions: []types.PromptInstruction{{
				Label:   "service_space",
				Content: "按服务空间规则回答。",
			}},
		},
		&mockChat{},
		nil,
		event.NewEventBus(),
		nil,
		nil,
		"test-session",
		"Custom base prompt",
	)
	require.NotNil(t, engine)

	prompt := engine.buildSystemPrompt(context.Background())
	require.True(t, strings.HasPrefix(prompt, "Custom base prompt"))
	require.Contains(t, prompt, "Content grounding")
	require.Contains(t, prompt, "<service_space_business_instructions>")
	require.Contains(t, prompt, "Source citations are enabled")
	require.True(t, strings.HasSuffix(prompt, llmreference.ProtocolPrompt(true)))
}
