package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	ErrOrganizeTemplateSourceRequired = errors.New("markdown source is required")
	organizeCompilerHeadingPattern    = regexp.MustCompile(`(?m)^\s{0,3}(#{1,6})\s+(.+?)\s*$`)
	organizeCompilerNumberedItem      = regexp.MustCompile(`^\s*\d+[.、)]\s*(.+?)\s*$`)
)

type compiledOrganizeSection struct {
	Title       string
	Placeholder string
	Kind        string
	Columns     []string
}

// CompileOrganizeTemplateMarkdown turns a skill-like Markdown instruction into
// a stable report preset. It intentionally uses deterministic extraction so an
// admin can preview and edit the result without paying for an LLM call.
func CompileOrganizeTemplateMarkdown(input types.OrganizeTemplateCompileInput) (*types.OrganizeTemplateCompileResult, error) {
	source := normalizeMarkdownSource(input.SourceMarkdown)
	if source == "" {
		return nil, ErrOrganizeTemplateSourceRequired
	}

	title := extractCompilerTitle(source)
	name := compilerTemplateName(title, source)
	sections, warnings := extractCompilerSections(source)
	if len(sections) == 0 {
		sections = []compiledOrganizeSection{
			{Title: "核心发现", Placeholder: "findings", Kind: "prose"},
			{Title: "下一步", Placeholder: "next_steps", Kind: "checklist"},
		}
		warnings = append(warnings, "未识别到标准章节，已使用通用报告结构")
	}

	markdownTemplate := buildCompiledMarkdownTemplate(sections, source)
	spec := buildCompiledTemplateSpec(sections, source)
	description := compilerDescription(name, sections)

	return &types.OrganizeTemplateCompileResult{
		Key:                compilerTemplateKey(source, title),
		Name:               name,
		Scene:              compilerTemplateScene(source, name),
		Description:        description,
		OutputLabel:        compilerOutputLabel(source, name),
		DefaultInstruction: source,
		MarkdownTemplate:   markdownTemplate,
		Spec:               spec,
		Sections:           sectionTitles(sections),
		Warnings:           warnings,
	}, nil
}

func normalizeMarkdownSource(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "\uFEFF"))
	value = strings.ReplaceAll(value, "\r\n", "\n")
	return strings.ReplaceAll(value, "\r", "\n")
}

func extractCompilerTitle(source string) string {
	for _, match := range organizeCompilerHeadingPattern.FindAllStringSubmatch(source, -1) {
		if len(match) >= 3 && len(match[1]) == 1 {
			return cleanCompilerHeading(match[2])
		}
	}
	return ""
}

func extractCompilerSections(source string) ([]compiledOrganizeSection, []string) {
	lines := strings.Split(source, "\n")
	standardStart := -1
	for index, line := range lines {
		if !organizeCompilerHeadingPattern.MatchString(line) {
			continue
		}
		heading := cleanCompilerHeading(strings.TrimSpace(strings.TrimLeft(line, "#")))
		if containsAnyCompilerKeyword(heading, "标准结构", "报告结构", "输出结构", "章节结构", "方案结构") {
			standardStart = index
			break
		}
	}

	var sections []compiledOrganizeSection
	var warnings []string
	if standardStart >= 0 {
		for _, line := range lines[standardStart+1:] {
			match := organizeCompilerNumberedItem.FindStringSubmatch(line)
			if len(match) != 2 {
				if len(sections) > 0 && strings.HasPrefix(strings.TrimSpace(line), "#") {
					break
				}
				continue
			}
			title := cleanCompilerSectionTitle(match[1])
			if title == "" {
				continue
			}
			sections = appendUniqueCompilerSection(sections, title)
		}
	}

	if len(sections) == 0 {
		for _, match := range organizeCompilerHeadingPattern.FindAllStringSubmatch(source, -1) {
			if len(match) != 3 || len(match[1]) != 2 {
				continue
			}
			title := cleanCompilerHeading(match[2])
			if title == "" || containsAnyCompilerKeyword(title, "核心能力", "工作流程", "注意事项", "输出规范", "沟通方式", "活动策划四类打法") {
				continue
			}
			sections = appendUniqueCompilerSection(sections, cleanCompilerSectionTitle(title))
		}
		if len(sections) > 0 {
			warnings = append(warnings, "未找到“标准结构”清单，已从二级标题推断报告章节")
		}
	}

	if len(sections) > 12 {
		warnings = append(warnings, fmt.Sprintf("识别到 %d 个章节，已保留前 12 个作为报告主结构", len(sections)))
		sections = sections[:12]
	}
	for index := range sections {
		sections[index].Placeholder, sections[index].Kind, sections[index].Columns = compilerSectionPresentation(sections[index].Title, index+1)
	}
	return sections, warnings
}

func appendUniqueCompilerSection(sections []compiledOrganizeSection, title string) []compiledOrganizeSection {
	for _, section := range sections {
		if section.Title == title {
			return sections
		}
	}
	return append(sections, compiledOrganizeSection{Title: title})
}

func cleanCompilerHeading(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "*_`")
	return strings.TrimSpace(value)
}

func cleanCompilerSectionTitle(value string) string {
	value = cleanCompilerHeading(value)
	if index := strings.IndexAny(value, "（("); index > 0 {
		value = strings.TrimSpace(value[:index])
	}
	return strings.TrimSpace(value)
}

func compilerSectionPresentation(title string, index int) (string, string, []string) {
	switch {
	case containsAnyCompilerKeyword(title, "流程", "时间轴", "进度"):
		return "timeline_rows", "table", []string{"时间", "环节", "内容", "负责人", "物料", "风险控制"}
	case containsAnyCompilerKeyword(title, "人员", "分工", "岗位", "责任"):
		return "staff_rows", "table", []string{"岗位", "角色", "职责", "关键时间点"}
	case containsAnyCompilerKeyword(title, "物料", "材料", "采购"):
		return "material_rows", "table", []string{"物料", "规格", "数量", "采购渠道", "责任人", "到位时间"}
	case containsAnyCompilerKeyword(title, "预算", "费用", "成本"):
		return "budget_rows", "table", []string{"项目", "单价", "数量", "小计", "备注"}
	case containsAnyCompilerKeyword(title, "下一步", "待办", "行动", "跟进", "尝试"):
		return fmt.Sprintf("section_%d", index), "checklist", nil
	default:
		return compilerPlaceholderForTitle(title, index), "prose", nil
	}
}

func compilerPlaceholderForTitle(title string, index int) string {
	switch {
	case containsAnyCompilerKeyword(title, "概述", "定位"):
		return "overview"
	case containsAnyCompilerKeyword(title, "目标"):
		return "goals"
	case containsAnyCompilerKeyword(title, "参与对象", "规模", "对象"):
		return "participants"
	case containsAnyCompilerKeyword(title, "时间", "地点"):
		return "time_and_location"
	case containsAnyCompilerKeyword(title, "环节", "活动详解", "内容详解"):
		return "segment_details"
	case containsAnyCompilerKeyword(title, "场地", "动线"):
		return "venue_layout"
	case containsAnyCompilerKeyword(title, "安全", "应急", "风险"):
		return "safety_plan"
	case containsAnyCompilerKeyword(title, "复盘", "总结"):
		return "review_template"
	default:
		return fmt.Sprintf("section_%d", index)
	}
}

func buildCompiledMarkdownTemplate(sections []compiledOrganizeSection, source string) string {
	var builder strings.Builder
	builder.WriteString("# {{title}}\n\n")
	builder.WriteString("> {{summary}}\n\n")
	for index, section := range sections {
		fmt.Fprintf(&builder, "## %d. %s\n\n", index+1, section.Title)
		switch section.Kind {
		case "table":
			fmt.Fprintf(&builder, "| %s |\n", strings.Join(section.Columns, " | "))
			separator := make([]string, len(section.Columns))
			for columnIndex := range separator {
				separator[columnIndex] = "---"
			}
			fmt.Fprintf(&builder, "| %s |\n", strings.Join(separator, " | "))
			fmt.Fprintf(&builder, "{{%s}}\n\n", section.Placeholder)
		case "checklist":
			fmt.Fprintf(&builder, "- [ ] {{%s}}\n\n", section.Placeholder)
		default:
			fmt.Fprintf(&builder, "{{%s}}\n\n", section.Placeholder)
		}
	}
	if containsAnyCompilerKeyword(source, "依据", "引用", "来源") {
		builder.WriteString("### 依据\n\n{{citations}}\n\n")
	}
	builder.WriteString("### 信息缺口与待确认项\n\n- [ ] {{open_questions}}\n\n")
	builder.WriteString("**标签：** {{tags}}\n")
	return strings.TrimSpace(builder.String())
}

func buildCompiledTemplateSpec(sections []compiledOrganizeSection, source string) types.JSONMap {
	required := make([]any, 0, len(sections))
	repeatable := make([]string, 0)
	for index, section := range sections {
		item := types.JSONMap{
			"key":      section.Placeholder,
			"title":    section.Title,
			"kind":     section.Kind,
			"required": true,
			"order":    index + 1,
		}
		if len(section.Columns) > 0 {
			item["columns"] = section.Columns
			repeatable = append(repeatable, section.Placeholder)
		}
		required = append(required, item)
	}
	return types.JSONMap{
		"sections": sectionTitles(sections),
		"markdown_contract": types.JSONMap{
			"version":                 "v1",
			"missing_value":           "___",
			"citation_mode":           "[M1]",
			"required_sections":       required,
			"repeatable_placeholders": repeatable,
			"output_rules": []string{
				"保留章节顺序和标题层级",
				"表格类占位符必须展开为完整的 Markdown 表格行",
				"每项任务必须包含责任人和时间节点",
				"未知数据使用 ___，不得编造",
				"关键事实和结论标注 [M1]、[M2] 来源",
			},
			"constraints": extractCompilerConstraints(source),
		},
	}
}

func extractCompilerConstraints(source string) []string {
	var constraints []string
	lines := strings.Split(source, "\n")
	for _, line := range lines {
		value := strings.TrimSpace(line)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		if strings.HasPrefix(value, "- ") && (strings.Contains(value, "不得") || strings.Contains(value, "必须") || strings.Contains(value, "需要") || strings.Contains(value, "不") && len([]rune(value)) < 120) {
			constraints = append(constraints, strings.TrimSpace(strings.TrimPrefix(value, "- ")))
		}
	}
	if len(constraints) > 20 {
		constraints = constraints[:20]
	}
	return constraints
}

func sectionTitles(sections []compiledOrganizeSection) []string {
	titles := make([]string, 0, len(sections))
	for _, section := range sections {
		titles = append(titles, section.Title)
	}
	return titles
}

func compilerTemplateName(title, source string) string {
	name := cleanCompilerHeading(title)
	if index := strings.IndexAny(name, "-—"); index > 0 {
		name = strings.TrimSpace(name[:index])
	}
	name = strings.TrimSpace(strings.TrimSuffix(name, "专家"))
	if name == "" {
		if containsAnyCompilerKeyword(source, "活动策划", "幼儿园") {
			return "幼儿园活动策划"
		}
		return "Markdown 整理报告"
	}
	return name
}

func compilerTemplateScene(source, name string) string {
	switch {
	case containsAnyCompilerKeyword(source, "幼儿园", "活动策划"):
		return "幼儿园活动策划"
	case containsAnyCompilerKeyword(source, "教研", "课程"):
		return "教研与课程"
	case containsAnyCompilerKeyword(source, "招生"):
		return "招生与增长"
	default:
		return name
	}
}

func compilerOutputLabel(source, name string) string {
	if containsAnyCompilerKeyword(source, "方案") {
		return "活动策划方案"
	}
	if containsAnyCompilerKeyword(source, "清单") {
		return "执行清单"
	}
	return name + "报告"
}

func compilerDescription(name string, sections []compiledOrganizeSection) string {
	if len(sections) == 0 {
		return "由 Markdown 指令自动生成的结构化整理报告"
	}
	return fmt.Sprintf("由 Markdown 指令自动生成，包含 %d 个固定章节和结构化输出规则的%s", len(sections), name)
}

func compilerTemplateKey(source, title string) string {
	switch {
	case containsAnyCompilerKeyword(source, "幼儿园", "活动策划"):
		return "kindergarten_activity"
	case containsAnyCompilerKeyword(source, "教研", "课程"):
		return "teaching_research"
	case containsAnyCompilerKeyword(source, "招生"):
		return "enrollment_growth"
	default:
		base := strings.ToLower(strings.TrimSpace(title))
		var builder strings.Builder
		for _, char := range base {
			if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
				builder.WriteRune(char)
			}
		}
		key := strings.Trim(builder.String(), "-_")
		if key == "" {
			return "markdown_report"
		}
		return key
	}
}

func containsAnyCompilerKeyword(value string, keywords ...string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}
