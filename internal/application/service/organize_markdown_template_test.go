package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultOrganizeMarkdownTemplateUsesSpecSections(t *testing.T) {
	template := defaultOrganizeMarkdownTemplate("sprout_review", types.JSONMap{
		"sections": []string{"原始种子", "可发展方向", "下一步尝试"},
	})

	require.Contains(t, template, "# {{title}}")
	require.Contains(t, template, "## 01. 原始种子")
	require.Contains(t, template, "## 03. 下一步尝试")
	require.Contains(t, template, "- [ ] {{section_3}}")
	require.Contains(t, template, "### 依据")
}

func TestNormalizeOrganizeMarkdownTemplateRejectsUnsafeOrUnstructuredInput(t *testing.T) {
	_, errorsList := normalizeOrganizeMarkdownTemplate("<script>alert(1)</script>", "custom", types.JSONMap{})

	require.Contains(t, errorsList, "markdown template must not contain script tags")
	require.Contains(t, errorsList, "markdown template must contain at least one Markdown heading")
}

func TestNormalizeOrganizeGeneratedMarkdownRemovesFenceAndAddsMissingSections(t *testing.T) {
	preset := "# {{title}}\n\n## 核心发现\n\n{{section_1}}\n\n## 下一步\n\n{{section_2}}"
	content := "```markdown\n# 周复盘\n\n## 核心发现\n\n已确认课堂存在重复等待 [M1]\n```"

	normalized := normalizeOrganizeGeneratedMarkdown(content, preset, "周复盘")

	require.True(t, strings.HasPrefix(normalized, "# 周复盘"))
	require.NotContains(t, normalized, "```")
	require.Contains(t, normalized, "## 核心发现")
	require.Contains(t, normalized, "## 下一步")
	require.Contains(t, normalized, "记录中未提供")
}

func TestBuildOrganizeJobPromptUsesMarkdownPresetInsteadOfJSONSpec(t *testing.T) {
	job := &types.OrganizeJob{
		TemplateVersion: "v1",
		Requirement: types.JSONMap{
			"config_name":          "周复盘",
			"template_name":        "教研提炼",
			"template_instruction": "只依据输入记忆整理",
			"template_markdown":    "# {{title}}\n\n## 核心发现\n\n{{section_1}}",
			"template_spec":        types.JSONMap{"sections": []string{"核心发现"}},
		},
	}

	prompt := buildOrganizeJobPrompt(job, nil)

	require.Contains(t, prompt, "Markdown 报告预设")
	require.Contains(t, prompt, "## 核心发现")
	require.NotContains(t, prompt, `"sections"`)
}
