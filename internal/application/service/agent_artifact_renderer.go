package service

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	reportHeadingPattern   = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)\s*$`)
	reportUnorderedList    = regexp.MustCompile(`^[ \t]*[-*+][ \t]+(.+?)\s*$`)
	reportOrderedList      = regexp.MustCompile(`^[ \t]*\d+\.[ \t]+(.+?)\s*$`)
	reportCodeFence        = regexp.MustCompile("^```[ \t]*([A-Za-z0-9_-]*)[ \t]*$")
	reportInlineCode       = regexp.MustCompile("`([^`\\n]+)`")
	reportMarkdownLink     = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+|mailto:[^)\s]+)\)`)
	reportStrongAsterisk   = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reportStrongUnderscore = regexp.MustCompile(`__([^_\n]+)__`)
	reportStrikethrough    = regexp.MustCompile(`~~([^~\n]+)~~`)
)

func renderStructuredReportHTML(report types.StructuredReportV1) (string, error) {
	title := strings.TrimSpace(report.Title)
	summary := strings.TrimSpace(report.ExecutiveSummary)
	if title == "" || summary == "" {
		return "", errors.New("structured report title and executive summary are required")
	}

	var builder strings.Builder
	builder.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	builder.WriteString(`<meta name="viewport" content="width=device-width,initial-scale=1">`)
	builder.WriteString(`<title>`)
	builder.WriteString(html.EscapeString(title))
	builder.WriteString(`</title><style>`)
	builder.WriteString(`:root{color-scheme:light;font-family:Inter,"PingFang SC","Microsoft YaHei",sans-serif;color:#18201c;background:#f5f7f6}`)
	builder.WriteString(`*{box-sizing:border-box}body{margin:0}.report{max-width:960px;margin:0 auto;padding:48px 32px 72px;background:#fff;min-height:100vh}`)
	builder.WriteString(`header{border-bottom:1px solid #dfe5e1;padding-bottom:28px}h1{font-size:34px;line-height:1.25;margin:0 0 20px}`)
	builder.WriteString(`.summary{font-size:17px;line-height:1.8;color:#3d4a43;white-space:pre-wrap}.section{padding:30px 0;border-bottom:1px solid #edf0ee}`)
	builder.WriteString(`.section-label{font-size:12px;color:#07894d;font-weight:700;text-transform:uppercase}.section h2{font-size:23px;margin:8px 0 14px}`)
	builder.WriteString(`.content{font-size:16px;line-height:1.85;white-space:pre-wrap}ul,ol{padding-left:24px;line-height:1.8}.references{margin-top:32px}`)
	builder.WriteString(`footer{margin-top:36px;color:#738079;font-size:13px}@media(max-width:640px){.report{padding:28px 20px}h1{font-size:28px}}`)
	builder.WriteString(`</style></head><body><article class="report"><header><h1>`)
	builder.WriteString(html.EscapeString(title))
	builder.WriteString(`</h1><div class="summary">`)
	builder.WriteString(html.EscapeString(summary))
	builder.WriteString(`</div></header>`)

	for index, section := range report.Sections {
		sectionTitle := strings.TrimSpace(section.Title)
		if sectionTitle == "" {
			continue
		}
		builder.WriteString(`<section class="section"><div class="section-label">`)
		builder.WriteString(html.EscapeString(fmt.Sprintf("%02d · %s", index+1, structuredReportSectionTypeLabel(section.Type))))
		builder.WriteString(`</div><h2>`)
		builder.WriteString(html.EscapeString(sectionTitle))
		builder.WriteString(`</h2>`)
		if content := strings.TrimSpace(section.Content); content != "" {
			builder.WriteString(`<div class="content">`)
			builder.WriteString(html.EscapeString(content))
			builder.WriteString(`</div>`)
		}
		items := nonEmptyReportItems(section.Items)
		if len(items) > 0 {
			builder.WriteString(`<ul>`)
			for _, item := range items {
				builder.WriteString(`<li>`)
				builder.WriteString(html.EscapeString(item))
				builder.WriteString(`</li>`)
			}
			builder.WriteString(`</ul>`)
		}
		builder.WriteString(`</section>`)
	}

	references := nonEmptyReportItems(report.EvidenceRefs)
	if len(references) > 0 {
		builder.WriteString(`<section class="references"><h2>依据与引用</h2><ol>`)
		for _, reference := range references {
			builder.WriteString(`<li>`)
			builder.WriteString(html.EscapeString(reference))
			builder.WriteString(`</li>`)
		}
		builder.WriteString(`</ol></section>`)
	}
	builder.WriteString(`<footer>由 Ruile Agent 生成，请结合实际情况复核后执行。</footer></article></body></html>`)
	return builder.String(), nil
}

func renderStructuredReportMarkdown(report types.StructuredReportV1) (string, error) {
	title := strings.TrimSpace(report.Title)
	summary := strings.TrimSpace(report.ExecutiveSummary)
	if title == "" || summary == "" {
		return "", errors.New("structured report title and executive summary are required")
	}

	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n## 执行摘要\n\n")
	builder.WriteString(summary)
	for _, section := range report.Sections {
		sectionTitle := strings.TrimSpace(section.Title)
		if sectionTitle == "" {
			continue
		}
		builder.WriteString("\n\n## ")
		builder.WriteString(sectionTitle)
		if content := strings.TrimSpace(section.Content); content != "" {
			builder.WriteString("\n\n")
			builder.WriteString(content)
		}
		for _, item := range nonEmptyReportItems(section.Items) {
			builder.WriteString("\n- ")
			builder.WriteString(item)
		}
	}
	references := nonBlankStrings(report.EvidenceRefs...)
	if len(references) > 0 {
		builder.WriteString("\n\n## 依据与引用\n")
		for _, reference := range references {
			builder.WriteString("\n- ")
			builder.WriteString(reference)
		}
	}
	builder.WriteString("\n")
	return builder.String(), nil
}

func structuredReportSectionTypeLabel(sectionType string) string {
	switch strings.TrimSpace(sectionType) {
	case "facts":
		return "事实"
	case "analysis":
		return "分析"
	case "risks":
		return "风险"
	case "missing_information":
		return "待补充信息"
	case "recommended_actions":
		return "建议行动"
	case "talk_track":
		return "沟通话术"
	case "evidence":
		return "依据"
	default:
		return "分析章节"
	}
}

func nonEmptyReportItems(items types.StringArray) []string {
	return nonBlankStrings(items...)
}

func nonBlankStrings(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func isMarkdownTableSeparator(line string) bool {
	cells := splitMarkdownTableRow(line)
	if len(cells) < 2 {
		return false
	}
	for _, cell := range cells {
		cell = strings.TrimSpace(cell)
		if len(cell) < 3 || strings.Trim(cell, ":-") != "" || !strings.Contains(cell, "-") {
			return false
		}
	}
	return true
}

func isMarkdownTableRow(line string) bool {
	return strings.Contains(line, "|") && len(splitMarkdownTableRow(line)) >= 2
}

func splitMarkdownTableRow(line string) []string {
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "|") {
		line = strings.TrimSpace(line[1:])
	}
	if strings.HasSuffix(line, "|") && !strings.HasSuffix(line, `\|`) {
		line = strings.TrimSpace(line[:len(line)-1])
	}
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(strings.ReplaceAll(parts[index], `\|`, "|"))
	}
	return parts
}
