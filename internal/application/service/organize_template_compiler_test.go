package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestCompileOrganizeTemplateMarkdownExtractsNumberedReportStructure(t *testing.T) {
	source := `---
name: kindergarten-activity-planner
---

# 幼儿园活动策划专家 - 童创

## 方案标准结构

每份方案按以下章节输出：

1. 活动概述（名称、主题、一句话定位）
2. 活动目标（教育目标 + 运营目标）
3. 参与对象与规模
4. 时间与地点
5. 活动流程时间轴
6. 环节详解
7. 人员分工表
8. 物料清单
9. 预算明细表
10. 场地布置与动线说明
11. 安全与应急预案
12. 活动后复盘模板

## 输出规范

- 每条任务必须带责任人和时间节点。
- 数据留空位，不编造。
`

	result, err := CompileOrganizeTemplateMarkdown(types.OrganizeTemplateCompileInput{
		SourceMarkdown: source,
	})

	require.NoError(t, err)
	require.Equal(t, "kindergarten_activity", result.Key)
	require.Equal(t, "幼儿园活动策划", result.Name)
	require.Len(t, result.Sections, 12)
	require.Contains(t, result.MarkdownTemplate, "## 5. 活动流程时间轴")
	require.Contains(t, result.MarkdownTemplate, "| 时间 | 环节 | 内容 | 负责人 | 物料 | 风险控制 |")
	require.Contains(t, result.MarkdownTemplate, "{{staff_rows}}")
	require.Contains(t, result.MarkdownTemplate, "{{open_questions}}")

	contract, ok := result.Spec["markdown_contract"].(types.JSONMap)
	require.True(t, ok)
	require.Equal(t, "___", contract["missing_value"])
	require.Contains(t, contract["repeatable_placeholders"], "timeline_rows")
	require.Contains(t, contract["constraints"], "每条任务必须带责任人和时间节点。")
}

func TestCompileOrganizeTemplateMarkdownFallsBackToGenericSections(t *testing.T) {
	result, err := CompileOrganizeTemplateMarkdown(types.OrganizeTemplateCompileInput{
		SourceMarkdown: "# 周复盘\n\n## 事实\n\n## 判断\n\n## 下一步\n",
	})

	require.NoError(t, err)
	require.Equal(t, "markdown_report", result.Key)
	require.Equal(t, "周复盘", result.Name)
	require.Equal(t, []string{"事实", "判断", "下一步"}, result.Sections)
	require.NotEmpty(t, result.Warnings)
	require.Contains(t, result.MarkdownTemplate, "- [ ] {{section_3}}")
}

func TestOrganizeMarkdownPromptAndNormalizationUseCompiledMissingValue(t *testing.T) {
	spec := types.JSONMap{
		"markdown_contract": types.JSONMap{"missing_value": "___"},
	}
	prompt := organizeMarkdownTemplatePrompt("# {{title}}", spec)
	require.Contains(t, prompt, "没有依据的栏目写“___”")

	normalized := normalizeOrganizeGeneratedMarkdown(
		"# 报告\n\n## 事实\n\n{{section_1}}",
		"# {{title}}\n\n## 事实\n\n{{section_1}}\n\n## 下一步\n\n{{section_2}}",
		"报告",
		spec,
	)
	require.Contains(t, normalized, "## 下一步\n\n___。")
	require.NotContains(t, normalized, "记录中未提供")
	require.False(t, strings.Contains(normalized, "{{"))
}
