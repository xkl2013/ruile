package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

const organizeMarkdownTemplateMaxLength = 12000

var (
	organizeMarkdownHeadingPattern       = regexp.MustCompile(`(?m)^\s{0,3}(#{1,6})\s+(.+?)\s*$`)
	organizeMarkdownPlaceholderPattern   = regexp.MustCompile(`\{\{[^{}\n]+\}\}`)
	organizeMarkdownCodeFencePattern     = regexp.MustCompile("(?s)^\\s*```(?:markdown|md)?\\s*\\n?(.*?)\\n?```\\s*$")
	organizeMarkdownHeadingNumberPattern = regexp.MustCompile(`^\d+[.、]\s*`)
)

func defaultOrganizeMarkdownTemplate(key string, spec types.JSONMap) string {
	sections := organizeTemplateStringList(types.JSONMap{"template_spec": spec}, "template_spec", "sections")
	if len(sections) == 0 {
		sections = []string{"核心发现", "证据与依据", "下一步"}
	}

	var builder strings.Builder
	builder.WriteString("# {{title}}\n\n")
	builder.WriteString("> {{summary}}\n\n")
	for index, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}
		fmt.Fprintf(&builder, "## %02d. %s\n\n", index+1, section)
		if isOrganizeActionSection(section) {
			fmt.Fprintf(&builder, "- [ ] {{section_%d}}\n\n", index+1)
		} else {
			fmt.Fprintf(&builder, "{{section_%d}}\n\n", index+1)
		}
	}
	builder.WriteString("### 依据\n\n{{citations}}\n\n")
	builder.WriteString("**标签：** {{tags}}\n")
	return strings.TrimSpace(builder.String())
}

func isOrganizeActionSection(section string) bool {
	return strings.Contains(section, "待办") ||
		strings.Contains(section, "下一步") ||
		strings.Contains(section, "行动") ||
		strings.Contains(section, "跟进") ||
		strings.Contains(section, "尝试") ||
		strings.Contains(section, "实施")
}

func normalizeOrganizeMarkdownTemplate(value, key string, spec types.JSONMap) (string, []string) {
	value = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"))
	if value == "" {
		value = defaultOrganizeMarkdownTemplate(key, spec)
	}
	errorsList := make([]string, 0)
	if len(value) > organizeMarkdownTemplateMaxLength {
		errorsList = append(errorsList, fmt.Sprintf("markdown template exceeds %d characters", organizeMarkdownTemplateMaxLength))
	}
	if strings.Contains(strings.ToLower(value), "<script") {
		errorsList = append(errorsList, "markdown template must not contain script tags")
	}
	if !organizeMarkdownHeadingPattern.MatchString(value) {
		errorsList = append(errorsList, "markdown template must contain at least one Markdown heading")
	}
	return value, errorsList
}

func hydrateOrganizeTemplateMarkdown(template *types.OrganizeTemplate) *types.OrganizeTemplate {
	if template == nil {
		return nil
	}
	if strings.TrimSpace(template.MarkdownTemplate) == "" {
		template.MarkdownTemplate = defaultOrganizeMarkdownTemplate(template.Key, template.Spec)
	}
	return template
}

func organizeMarkdownTemplatePrompt(template string, specs ...types.JSONMap) string {
	template = strings.TrimSpace(template)
	missingValue := "记录中未提供"
	if len(specs) > 0 {
		if value := organizeMarkdownMissingValue(specs[0]); value != "" {
			missingValue = value
		}
	}
	return fmt.Sprintf(`Markdown 报告预设：
%s

隐藏格式约束：
- 必须输出完整的中文 Markdown，不要输出 JSON，不要使用 Markdown 代码围栏包裹整篇结果。
- 保留预设中的标题层级、章节名称和章节顺序；不要新增与输入无关的章节。
- 将预设中的 {{...}} 占位符替换为实际内容，不得在最终结果中保留未替换占位符。
- 只依据原始记忆生成内容；没有依据的栏目写“%s”，不得猜测、补造数据。
- 关键事实和结论使用 [M1]、[M2] 等来源标记；待办内容使用 - [ ] 清单。
- 表格占位符必须展开为完整的 Markdown 行，不能把整张表格压缩成一句话。
	- 如果一个章节没有足够内容，也必须保留章节并明确标注信息缺失。`, template, missingValue)
}

func normalizeOrganizeGeneratedMarkdown(content, preset, title string, specs ...types.JSONMap) string {
	content = strings.TrimSpace(content)
	if match := organizeMarkdownCodeFencePattern.FindStringSubmatch(content); len(match) == 2 {
		content = strings.TrimSpace(match[1])
	}
	content = strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
	missingValue := "记录中未提供"
	if len(specs) > 0 {
		if value := organizeMarkdownMissingValue(specs[0]); value != "" {
			missingValue = value
		}
	}
	content = organizeMarkdownPlaceholderPattern.ReplaceAllString(content, missingValue)
	if content == "" {
		content = fmt.Sprintf("# %s\n\n%s。", emptyFallback(strings.TrimSpace(title), "整理结果"), missingValue)
	}

	headings := markdownTemplateSectionHeadings(preset)
	actual := map[string]bool{}
	for _, match := range organizeMarkdownHeadingPattern.FindAllStringSubmatch(content, -1) {
		if len(match) < 3 {
			continue
		}
		level := len(match[1])
		if level < 2 {
			continue
		}
		actual[normalizeMarkdownHeading(match[2])] = true
	}
	var missing []string
	for _, heading := range headings {
		if heading == "" || actual[heading] {
			continue
		}
		missing = append(missing, heading)
	}
	if len(missing) > 0 {
		var builder strings.Builder
		builder.WriteString(strings.TrimSpace(content))
		for _, heading := range missing {
			fmt.Fprintf(&builder, "\n\n## %s\n\n%s。", heading, missingValue)
		}
		content = builder.String()
	}
	return strings.TrimSpace(content)
}

func organizeMarkdownMissingValue(spec types.JSONMap) string {
	contract := organizeJSONMapValue(spec["markdown_contract"])
	if contract == nil {
		return ""
	}
	value := strings.TrimSpace(fmt.Sprint(contract["missing_value"]))
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func organizeJSONMapValue(value any) types.JSONMap {
	switch typed := value.(type) {
	case types.JSONMap:
		return typed
	case map[string]any:
		return types.JSONMap(typed)
	default:
		return nil
	}
}

func markdownTemplateSectionHeadings(preset string) []string {
	seen := map[string]bool{}
	headings := make([]string, 0)
	for _, match := range organizeMarkdownHeadingPattern.FindAllStringSubmatch(preset, -1) {
		if len(match) < 3 || len(match[1]) < 2 {
			continue
		}
		heading := normalizeMarkdownHeading(match[2])
		if heading == "" || seen[heading] {
			continue
		}
		seen[heading] = true
		headings = append(headings, heading)
	}
	return headings
}

func normalizeMarkdownHeading(value string) string {
	value = organizeMarkdownPlaceholderPattern.ReplaceAllString(value, "")
	value = organizeMarkdownHeadingNumberPattern.ReplaceAllString(strings.TrimSpace(value), "")
	value = strings.Trim(strings.TrimSpace(value), "*_`")
	return strings.TrimSpace(value)
}
