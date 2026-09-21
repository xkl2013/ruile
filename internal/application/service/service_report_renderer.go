package service

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	reportHeadingPattern   = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)\s*$`)
	reportUnorderedList    = regexp.MustCompile(`^[ \t]*[-*+][ \t]+(.+?)\s*$`)
	reportOrderedList      = regexp.MustCompile(`^[ \t]*\d+\.[ \t]+(.+?)\s*$`)
	reportBlockquote       = regexp.MustCompile(`^[ \t]*>[ \t]?(.*?)\s*$`)
	reportCodeFence        = regexp.MustCompile("^```[ \t]*([A-Za-z0-9_-]*)[ \t]*$")
	reportInlineCode       = regexp.MustCompile("`([^`\\n]+)`")
	reportMarkdownLink     = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^)\s]+|mailto:[^)\s]+)\)`)
	reportStrongAsterisk   = regexp.MustCompile(`\*\*([^*\n]+)\*\*`)
	reportStrongUnderscore = regexp.MustCompile(`__([^_\n]+)__`)
	reportEmphasis         = regexp.MustCompile(`\*([^*\n]+)\*`)
	reportStrikethrough    = regexp.MustCompile(`~~([^~\n]+)~~`)
)

// renderStructuredReportHTML only renders the platform contract through a
// fixed template. It never injects agent-provided HTML into the output.
func renderStructuredReportHTML(report types.StructuredReportV1) (string, error) {
	if strings.TrimSpace(report.Title) == "" || strings.TrimSpace(report.ExecutiveSummary) == "" {
		return "", errors.New("structured report title and executive summary are required")
	}
	renderedSections := make([]types.StructuredReportSection, 0, len(report.Sections))
	for _, section := range report.Sections {
		if strings.TrimSpace(section.Title) != "" {
			renderedSections = append(renderedSections, section)
		}
	}

	var builder strings.Builder
	builder.WriteString(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">`)
	builder.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">`)
	builder.WriteString(`<meta name="color-scheme" content="light dark">`)
	builder.WriteString(`<title>`)
	builder.WriteString(html.EscapeString(report.Title))
	builder.WriteString(`</title><style>`)
	builder.WriteString(reportHTMLStyles())
	builder.WriteString(`</style></head><body>`)
	builder.WriteString(`<div class="report-shell">`)
	builder.WriteString(`<header class="report-hero">`)
	builder.WriteString(`<div class="report-kicker"><span class="report-kicker__mark"></span><span>专家分析报告</span><span class="report-kicker__dot">·</span><span>结构化产物</span></div>`)
	builder.WriteString(`<h1>`)
	builder.WriteString(html.EscapeString(report.Title))
	builder.WriteString(`</h1>`)
	builder.WriteString(`<div class="report-hero__rule"></div>`)
	builder.WriteString(`<div class="report-summary">`)
	builder.WriteString(`<div class="report-summary__label">执行摘要</div>`)
	builder.WriteString(`<div class="report-summary__content">`)
	builder.WriteString(renderReportMarkdown(report.ExecutiveSummary))
	builder.WriteString(`</div></div>`)
	builder.WriteString(`<div class="report-stats">`)
	builder.WriteString(`<span><strong>`)
	builder.WriteString(html.EscapeString(strconv.Itoa(len(renderedSections))))
	builder.WriteString(`</strong> 个分析章节</span>`)
	if len(report.EvidenceRefs) > 0 {
		builder.WriteString(`<span><strong>`)
		builder.WriteString(html.EscapeString(strconv.Itoa(len(report.EvidenceRefs))))
		builder.WriteString(`</strong> 条依据引用</span>`)
	} else {
		builder.WriteString(`<span>暂无外部依据引用</span>`)
	}
	builder.WriteString(`</div></header>`)

	builder.WriteString(`<div class="report-layout">`)
	if len(renderedSections) > 0 {
		builder.WriteString(`<aside class="report-outline" aria-label="报告目录"><div class="report-outline__title">本报告</div><nav>`)
		for index, section := range renderedSections {
			builder.WriteString(`<a href="#section-`)
			builder.WriteString(strconv.Itoa(index + 1))
			builder.WriteString(`"><span>`)
			builder.WriteString(fmt.Sprintf("%02d", index+1))
			builder.WriteString(`</span><em>`)
			builder.WriteString(html.EscapeString(section.Title))
			builder.WriteString(`</em></a>`)
		}
		builder.WriteString(`</nav></aside>`)
	}

	builder.WriteString(`<main class="report-content">`)
	for index, section := range renderedSections {
		builder.WriteString(`<section id="section-`)
		builder.WriteString(strconv.Itoa(index + 1))
		builder.WriteString(`" class="report-section report-section--`)
		builder.WriteString(html.EscapeString(section.Type))
		builder.WriteString(`">`)
		builder.WriteString(`<div class="report-section__heading"><span class="report-section__number">`)
		builder.WriteString(fmt.Sprintf("%02d", index+1))
		builder.WriteString(`</span><div><div class="report-section__type">`)
		builder.WriteString(html.EscapeString(structuredReportSectionTypeLabel(section.Type)))
		builder.WriteString(`</div><h2>`)
		builder.WriteString(html.EscapeString(section.Title))
		builder.WriteString(`</h2></div></div>`)
		if strings.TrimSpace(section.Content) != "" {
			builder.WriteString(`<div class="report-section__content">`)
			builder.WriteString(renderReportMarkdown(section.Content))
			builder.WriteString(`</div>`)
		}
		if items := nonEmptyReportItems(section.Items); len(items) > 0 {
			builder.WriteString(`<ul class="report-list">`)
			for _, item := range items {
				builder.WriteString(`<li><span class="report-list__marker"></span><span>`)
				builder.WriteString(renderReportInlineMarkdown(item))
				builder.WriteString(`</span></li>`)
			}
			builder.WriteString(`</ul>`)
		}
		builder.WriteString(`</section>`)
	}
	if len(report.EvidenceRefs) > 0 {
		builder.WriteString(`<section class="report-references"><div class="report-section__type">依据与引用</div><h2>信息来源</h2><ol>`)
		for _, evidence := range report.EvidenceRefs {
			if strings.TrimSpace(evidence) == "" {
				continue
			}
			builder.WriteString(`<li>`)
			builder.WriteString(renderReportInlineMarkdown(evidence))
			builder.WriteString(`</li>`)
		}
		builder.WriteString(`</ol></section>`)
	}
	builder.WriteString(`</main></div>`)
	builder.WriteString(`<footer class="report-footer"><span>由 Ruile 专家运行生成</span><span>内容请结合实际情况复核后执行</span></footer>`)
	builder.WriteString(`</div></body></html>`)
	return builder.String(), nil
}

func renderStructuredReportMarkdown(report types.StructuredReportV1) (string, error) {
	if strings.TrimSpace(report.Title) == "" || strings.TrimSpace(report.ExecutiveSummary) == "" {
		return "", errors.New("structured report title and executive summary are required")
	}
	var builder strings.Builder
	builder.WriteString("# ")
	builder.WriteString(strings.TrimSpace(report.Title))
	builder.WriteString("\n\n## 执行摘要\n\n")
	builder.WriteString(strings.TrimSpace(report.ExecutiveSummary))
	for _, section := range report.Sections {
		title := strings.TrimSpace(section.Title)
		if title == "" {
			continue
		}
		builder.WriteString("\n\n## ")
		builder.WriteString(title)
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

func renderReportMarkdown(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\r\n", "\n"))
	if value == "" {
		return ""
	}

	lines := strings.Split(value, "\n")
	var builder strings.Builder
	paragraphLines := make([]string, 0, 4)
	listKind := ""
	listItems := make([]string, 0, 4)
	quoteLines := make([]string, 0, 4)
	codeLines := make([]string, 0, 4)
	inCodeFence := false

	flushParagraph := func() {
		if len(paragraphLines) == 0 {
			return
		}
		builder.WriteString("<p>")
		for index, line := range paragraphLines {
			if index > 0 {
				builder.WriteString("<br>")
			}
			builder.WriteString(renderReportInlineMarkdown(line))
		}
		builder.WriteString("</p>")
		paragraphLines = paragraphLines[:0]
	}
	flushList := func() {
		if len(listItems) == 0 {
			return
		}
		tag := "ul"
		if listKind == "ol" {
			tag = "ol"
		}
		builder.WriteString("<")
		builder.WriteString(tag)
		builder.WriteString(">")
		for _, item := range listItems {
			builder.WriteString("<li>")
			builder.WriteString(renderReportInlineMarkdown(item))
			builder.WriteString("</li>")
		}
		builder.WriteString("</")
		builder.WriteString(tag)
		builder.WriteString(">")
		listKind = ""
		listItems = listItems[:0]
	}
	flushQuote := func() {
		if len(quoteLines) == 0 {
			return
		}
		builder.WriteString("<blockquote><p>")
		for index, line := range quoteLines {
			if index > 0 {
				builder.WriteString("<br>")
			}
			builder.WriteString(renderReportInlineMarkdown(line))
		}
		builder.WriteString("</p></blockquote>")
		quoteLines = quoteLines[:0]
	}
	flushBlocks := func() {
		flushParagraph()
		flushList()
		flushQuote()
	}
	flushCode := func() {
		if !inCodeFence {
			return
		}
		builder.WriteString("<pre><code>")
		for index, line := range codeLines {
			if index > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(html.EscapeString(line))
		}
		builder.WriteString("</code></pre>")
		codeLines = codeLines[:0]
		inCodeFence = false
	}

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		trimmed := strings.TrimSpace(line)

		if inCodeFence {
			if reportCodeFence.MatchString(trimmed) {
				flushCode()
			} else {
				codeLines = append(codeLines, line)
			}
			continue
		}
		if reportCodeFence.MatchString(trimmed) {
			flushBlocks()
			inCodeFence = true
			continue
		}

		if index+1 < len(lines) && strings.Contains(line, "|") &&
			isMarkdownTableSeparator(lines[index+1]) {
			flushBlocks()
			tableEnd := index + 2
			for tableEnd < len(lines) && isMarkdownTableRow(lines[tableEnd]) {
				tableEnd++
			}
			builder.WriteString(renderReportTable(splitMarkdownTableRow(line), lines[index+2:tableEnd]))
			index = tableEnd - 1
			continue
		}

		if match := reportHeadingPattern.FindStringSubmatch(line); match != nil {
			flushBlocks()
			level := len(match[1])
			builder.WriteString("<h")
			builder.WriteString(strconv.Itoa(level))
			builder.WriteString(">")
			builder.WriteString(renderReportInlineMarkdown(match[2]))
			builder.WriteString("</h")
			builder.WriteString(strconv.Itoa(level))
			builder.WriteString(">")
			continue
		}
		if match := reportUnorderedList.FindStringSubmatch(line); match != nil {
			flushParagraph()
			flushQuote()
			if listKind != "ul" {
				flushList()
				listKind = "ul"
			}
			listItems = append(listItems, match[1])
			continue
		}
		if match := reportOrderedList.FindStringSubmatch(line); match != nil {
			flushParagraph()
			flushQuote()
			if listKind != "ol" {
				flushList()
				listKind = "ol"
			}
			listItems = append(listItems, match[1])
			continue
		}
		if match := reportBlockquote.FindStringSubmatch(line); match != nil {
			flushParagraph()
			flushList()
			quoteLines = append(quoteLines, match[1])
			continue
		}
		if trimmed == "" {
			flushBlocks()
			continue
		}
		flushList()
		flushQuote()
		paragraphLines = append(paragraphLines, line)
	}
	if inCodeFence {
		flushCode()
	} else {
		flushBlocks()
	}
	return builder.String()
}

func renderReportInlineMarkdown(value string) string {
	escaped := html.EscapeString(strings.TrimSpace(value))
	if escaped == "" {
		return ""
	}
	protected := make([]string, 0, 4)
	protect := func(value string) string {
		token := fmt.Sprintf("\x00REPORT_INLINE_%d\x00", len(protected))
		protected = append(protected, value)
		return token
	}

	escaped = reportInlineCode.ReplaceAllStringFunc(escaped, func(match string) string {
		content := reportInlineCode.FindStringSubmatch(match)[1]
		return protect("<code>" + content + "</code>")
	})
	escaped = reportMarkdownLink.ReplaceAllStringFunc(escaped, func(match string) string {
		parts := reportMarkdownLink.FindStringSubmatch(match)
		return protect(`<a href="` + parts[2] + `" target="_blank" rel="noreferrer noopener">` + parts[1] + `</a>`)
	})
	escaped = reportStrikethrough.ReplaceAllString(escaped, "<del>$1</del>")
	escaped = reportStrongAsterisk.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = reportStrongUnderscore.ReplaceAllString(escaped, "<strong>$1</strong>")
	escaped = reportEmphasis.ReplaceAllString(escaped, "<em>$1</em>")

	for index, value := range protected {
		escaped = strings.ReplaceAll(escaped, fmt.Sprintf("\x00REPORT_INLINE_%d\x00", index), value)
	}
	return escaped
}

func renderReportTable(headers []string, rawRows []string) string {
	var builder strings.Builder
	builder.WriteString(`<div class="report-table-wrap"><table><thead><tr>`)
	for _, header := range headers {
		builder.WriteString("<th>")
		builder.WriteString(renderReportInlineMarkdown(header))
		builder.WriteString("</th>")
	}
	builder.WriteString("</tr></thead><tbody>")
	for _, rawRow := range rawRows {
		cells := splitMarkdownTableRow(rawRow)
		if len(cells) == 0 {
			continue
		}
		builder.WriteString("<tr>")
		for index := range headers {
			cell := ""
			if index < len(cells) {
				cell = cells[index]
			}
			builder.WriteString("<td>")
			builder.WriteString(renderReportInlineMarkdown(cell))
			builder.WriteString("</td>")
		}
		builder.WriteString("</tr>")
	}
	builder.WriteString("</tbody></table></div>")
	return builder.String()
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

func renderReportPlainText(value string) string {
	return strings.ReplaceAll(html.EscapeString(strings.TrimSpace(value)), "\n", "<br>")
}

func nonEmptyReportItems(items types.StringArray) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			result = append(result, item)
		}
	}
	return result
}

func structuredReportSectionTypeLabel(sectionType string) string {
	switch sectionType {
	case "facts":
		return "事实层"
	case "analysis":
		return "分析判断"
	case "risks":
		return "风险提示"
	case "missing_information":
		return "待补充信息"
	case "recommended_actions":
		return "建议动作"
	case "talk_track":
		return "沟通话术"
	case "evidence":
		return "验证依据"
	default:
		return "报告章节"
	}
}

func reportHTMLStyles() string {
	return `
:root {
  color-scheme: light;
  --report-ink: #1f2937;
  --report-muted: #697386;
  --report-subtle: #9aa3b2;
  --report-line: #e5e9ef;
  --report-panel: #ffffff;
  --report-page: #f5f7fa;
  --report-blue: #2868d7;
  --report-blue-soft: #eaf2ff;
  --report-green: #197a58;
  --report-green-soft: #e7f6ef;
  --report-amber: #a35c08;
  --report-amber-soft: #fff4df;
  --report-purple: #7053b8;
  --report-purple-soft: #f2edff;
  --report-shadow: 0 18px 50px rgba(30, 45, 68, .08);
}
* { box-sizing: border-box; }
html { scroll-behavior: smooth; }
body {
  min-width: 320px;
  margin: 0;
  background: var(--report-page);
  color: var(--report-ink);
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
  font-size: 15px;
  line-height: 1.8;
  -webkit-font-smoothing: antialiased;
}
.report-shell { max-width: 1180px; margin: 0 auto; padding: 52px 32px 40px; }
.report-hero {
  position: relative;
  overflow: hidden;
  padding: 42px 48px 36px;
  border: 1px solid #dce6f2;
  border-radius: 18px;
  background: linear-gradient(135deg, #f8fbff 0%, #ffffff 58%, #f1f8f5 100%);
  box-shadow: var(--report-shadow);
}
.report-hero::after {
  position: absolute;
  right: -100px;
  bottom: -135px;
  width: 320px;
  height: 320px;
  border: 1px solid rgba(40, 104, 215, .12);
  border-radius: 50%;
  box-shadow: 0 0 0 28px rgba(40, 104, 215, .035), 0 0 0 56px rgba(40, 104, 215, .02);
  content: "";
}
.report-kicker {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 19px;
  color: var(--report-blue);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: .08em;
}
.report-kicker__mark { width: 8px; height: 8px; border-radius: 50%; background: var(--report-green); box-shadow: 0 0 0 5px rgba(25, 122, 88, .11); }
.report-kicker__dot { color: var(--report-subtle); }
.report-hero h1 { max-width: 860px; margin: 0; color: #172033; font-size: clamp(28px, 4vw, 44px); font-weight: 750; letter-spacing: -.02em; line-height: 1.25; }
.report-hero__rule { width: 64px; height: 3px; margin: 24px 0 26px; border-radius: 3px; background: linear-gradient(90deg, var(--report-blue), var(--report-green)); }
.report-summary { max-width: 900px; padding: 19px 22px; border-left: 3px solid var(--report-blue); border-radius: 0 10px 10px 0; background: rgba(255, 255, 255, .8); }
.report-summary__label { margin-bottom: 5px; color: var(--report-blue); font-size: 12px; font-weight: 700; }
.report-summary__content { color: #3d4a5e; font-size: 16px; }
.report-summary__content p, .report-section__content p { margin: 0 0 10px; }
.report-summary__content p:last-child, .report-section__content p:last-child { margin-bottom: 0; }
.report-summary__content strong, .report-section__content strong, .report-list strong { color: #202b3d; font-weight: 700; }
.report-summary__content em, .report-section__content em, .report-list em { color: #4b5563; }
.report-summary__content h1, .report-summary__content h2, .report-summary__content h3, .report-section__content h1, .report-section__content h2, .report-section__content h3 { margin: 20px 0 8px; color: #263247; line-height: 1.45; }
.report-summary__content h1, .report-section__content h1 { font-size: 22px; }
.report-summary__content h2, .report-section__content h2 { font-size: 19px; }
.report-summary__content h3, .report-section__content h3 { font-size: 17px; }
.report-summary__content ul, .report-summary__content ol, .report-section__content ul, .report-section__content ol { margin: 10px 0; padding-left: 24px; }
.report-summary__content li, .report-section__content li { margin: 4px 0; }
.report-summary__content blockquote, .report-section__content blockquote { margin: 14px 0; padding: 8px 14px; border-left: 3px solid #c9d8ee; color: var(--report-muted); background: #f8fafc; }
.report-summary__content code, .report-section__content code, .report-list code { padding: 2px 5px; border-radius: 4px; color: #9b3d08; background: #fff2e8; font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace; font-size: .88em; }
.report-summary__content pre, .report-section__content pre { overflow-x: auto; margin: 14px 0; padding: 14px 16px; border-radius: 8px; color: #e6edf7; background: #1c2737; font-size: 13px; line-height: 1.6; }
.report-summary__content pre code, .report-section__content pre code { padding: 0; color: inherit; background: transparent; }
.report-summary__content hr, .report-section__content hr { margin: 20px 0; border: 0; border-top: 1px solid var(--report-line); }
.report-summary__content table, .report-section__content table { display: block; width: 100%; overflow-x: auto; margin: 16px 0; border-collapse: collapse; border: 1px solid var(--report-line); font-size: 13px; }
.report-summary__content th, .report-section__content th { padding: 9px 11px; border: 1px solid var(--report-line); color: #344054; background: #f5f8fc; font-weight: 700; text-align: left; white-space: nowrap; }
.report-summary__content td, .report-section__content td { min-width: 90px; padding: 9px 11px; border: 1px solid var(--report-line); vertical-align: top; }
.report-summary__content tr:nth-child(even) td, .report-section__content tr:nth-child(even) td { background: #fbfcfe; }
.report-stats { display: flex; flex-wrap: wrap; gap: 18px; margin-top: 23px; color: var(--report-muted); font-size: 12px; }
.report-stats span { display: inline-flex; align-items: baseline; gap: 4px; }
.report-stats strong { color: var(--report-ink); font-size: 16px; }
.report-layout { display: grid; grid-template-columns: 190px minmax(0, 1fr); gap: 54px; align-items: start; margin-top: 44px; }
.report-outline { position: sticky; top: 24px; padding-top: 5px; }
.report-outline__title { margin-bottom: 15px; color: var(--report-subtle); font-size: 11px; font-weight: 700; letter-spacing: .12em; text-transform: uppercase; }
.report-outline nav { display: flex; flex-direction: column; gap: 5px; }
.report-outline a { display: grid; grid-template-columns: 28px minmax(0, 1fr); gap: 7px; align-items: start; padding: 7px 8px; border-radius: 6px; color: var(--report-muted); text-decoration: none; font-size: 12px; line-height: 1.5; transition: background .2s, color .2s; }
.report-outline a:hover { color: var(--report-blue); background: var(--report-blue-soft); }
.report-outline a span { color: var(--report-subtle); font-variant-numeric: tabular-nums; }
.report-outline a em { overflow-wrap: anywhere; font-style: normal; }
.report-content { min-width: 0; }
.report-section { position: relative; margin-bottom: 30px; padding: 26px 30px 29px; border: 1px solid var(--report-line); border-radius: 13px; background: var(--report-panel); box-shadow: 0 4px 18px rgba(31, 41, 55, .035); scroll-margin-top: 24px; }
.report-section__heading { display: flex; align-items: flex-start; gap: 14px; margin-bottom: 17px; }
.report-section__number { display: inline-flex; flex: 0 0 auto; width: 34px; height: 34px; align-items: center; justify-content: center; border-radius: 9px; color: var(--report-blue); background: var(--report-blue-soft); font-size: 11px; font-weight: 700; font-variant-numeric: tabular-nums; }
.report-section__type { margin-bottom: 2px; color: var(--report-blue); font-size: 11px; font-weight: 700; letter-spacing: .04em; }
.report-section h2, .report-references h2 { margin: 0; color: #202b3d; font-size: 21px; font-weight: 700; line-height: 1.4; }
.report-section__content { color: #4d596b; }
.report-list { display: grid; gap: 9px; margin: 18px 0 0; padding: 0; list-style: none; }
.report-list li { display: grid; grid-template-columns: 8px minmax(0, 1fr); gap: 10px; align-items: start; padding: 11px 13px; border: 1px solid #edf0f4; border-radius: 8px; color: #3d4a5e; background: #fbfcfd; }
.report-list li > span:last-child > p { display: inline; margin: 0; }
.report-list__marker { width: 6px; height: 6px; margin-top: 9px; border-radius: 50%; background: var(--report-blue); }
.report-section--facts .report-section__number, .report-section--evidence .report-section__number { color: var(--report-blue); background: var(--report-blue-soft); }
.report-section--analysis .report-section__number, .report-section--talk_track .report-section__number { color: var(--report-purple); background: var(--report-purple-soft); }
.report-section--risks .report-section__number { color: var(--report-amber); background: var(--report-amber-soft); }
.report-section--risks .report-section__type { color: var(--report-amber); }
.report-section--recommended_actions .report-section__number { color: var(--report-green); background: var(--report-green-soft); }
.report-section--recommended_actions .report-section__type { color: var(--report-green); }
.report-section--missing_information .report-section__number { color: #8a5b20; background: #fff6e8; }
.report-references { margin: 40px 0 0; padding: 28px 30px; border-top: 1px solid var(--report-line); }
.report-references .report-section__type { color: var(--report-muted); }
.report-references ol { margin: 15px 0 0; padding-left: 23px; color: var(--report-muted); }
.report-references li { padding-left: 5px; margin-bottom: 7px; }
.report-footer { display: flex; justify-content: space-between; gap: 20px; margin-top: 34px; padding-top: 17px; border-top: 1px solid var(--report-line); color: var(--report-subtle); font-size: 11px; }
@media (max-width: 820px) {
  .report-shell { padding: 24px 16px 28px; }
  .report-hero { padding: 29px 24px 26px; border-radius: 13px; }
  .report-layout { display: block; margin-top: 27px; }
  .report-outline { position: static; margin-bottom: 22px; padding: 0 2px; }
  .report-outline nav { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 4px; }
  .report-section { padding: 22px 21px 23px; }
}
@media (max-width: 520px) {
  .report-hero h1 { font-size: 29px; }
  .report-summary__content { font-size: 15px; }
  .report-outline nav { display: flex; }
  .report-section h2, .report-references h2 { font-size: 18px; }
  .report-footer { flex-direction: column; gap: 4px; }
}
@media print {
  body { background: #fff; }
  .report-shell { max-width: none; padding: 0; }
  .report-hero { box-shadow: none; break-inside: avoid; }
  .report-outline { display: none; }
  .report-layout { display: block; margin-top: 26px; }
  .report-section { box-shadow: none; break-inside: avoid; }
  .report-footer { margin-top: 22px; }
}
@media (prefers-color-scheme: dark) {
  :root {
    color-scheme: dark;
    --report-ink: #e5e7eb;
    --report-muted: #a8b1c1;
    --report-subtle: #788398;
    --report-line: #2e394a;
    --report-panel: #182130;
    --report-page: #0f1724;
    --report-blue-soft: #1d355b;
    --report-green-soft: #183b31;
    --report-amber-soft: #4b351a;
    --report-purple-soft: #32284f;
    --report-shadow: 0 18px 50px rgba(0, 0, 0, .18);
  }
  .report-hero { border-color: #2e425b; background: linear-gradient(135deg, #142238 0%, #182130 58%, #152b28 100%); }
  .report-hero h1, .report-section h2, .report-references h2 { color: #f1f5f9; }
  .report-summary { background: rgba(24, 33, 48, .78); }
  .report-summary__content, .report-section__content, .report-list li { color: #c2cad6; }
  .report-summary__content strong, .report-section__content strong, .report-list strong, .report-summary__content h1, .report-summary__content h2, .report-summary__content h3, .report-section__content h1, .report-section__content h2, .report-section__content h3 { color: #f1f5f9; }
  .report-summary__content em, .report-section__content em, .report-list em { color: #c2cad6; }
  .report-summary__content blockquote, .report-section__content blockquote { border-color: #40536d; background: #141d2b; }
  .report-summary__content code, .report-section__content code, .report-list code { color: #ffc294; background: #452c1d; }
  .report-summary__content th, .report-section__content th { color: #e6edf7; background: #202c3e; }
  .report-summary__content tr:nth-child(even) td, .report-section__content tr:nth-child(even) td { background: #141d2b; }
  .report-list li { border-color: #2a3545; background: #141d2b; }
}
`
}
