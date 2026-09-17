package service

import (
	"errors"
	"html"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// renderStructuredReportHTML only renders the platform contract through a
// fixed template. It never injects agent-provided HTML into the output.
func renderStructuredReportHTML(report types.StructuredReportV1) (string, error) {
	if strings.TrimSpace(report.Title) == "" || strings.TrimSpace(report.ExecutiveSummary) == "" {
		return "", errors.New("structured report title and executive summary are required")
	}
	var builder strings.Builder
	builder.WriteString("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\">")
	builder.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	builder.WriteString("<title>")
	builder.WriteString(html.EscapeString(report.Title))
	builder.WriteString("</title><style>")
	builder.WriteString("body{max-width:860px;margin:0 auto;padding:40px 24px;font-family:-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif;color:#1f2937;line-height:1.7}")
	builder.WriteString("h1{font-size:28px;line-height:1.3;margin:0 0 12px}h2{font-size:18px;margin:32px 0 8px;border-bottom:1px solid #e5e7eb;padding-bottom:8px}")
	builder.WriteString("p{margin:8px 0}ul{margin:8px 0;padding-left:24px}.summary{color:#4b5563;font-size:16px}</style></head><body>")
	builder.WriteString("<h1>")
	builder.WriteString(html.EscapeString(report.Title))
	builder.WriteString("</h1><p class=\"summary\">")
	builder.WriteString(renderReportText(report.ExecutiveSummary))
	builder.WriteString("</p>")
	for _, section := range report.Sections {
		if strings.TrimSpace(section.Title) == "" {
			continue
		}
		builder.WriteString("<section><h2>")
		builder.WriteString(html.EscapeString(section.Title))
		builder.WriteString("</h2>")
		if strings.TrimSpace(section.Content) != "" {
			builder.WriteString("<p>")
			builder.WriteString(renderReportText(section.Content))
			builder.WriteString("</p>")
		}
		if len(section.Items) > 0 {
			builder.WriteString("<ul>")
			for _, item := range section.Items {
				if strings.TrimSpace(item) == "" {
					continue
				}
				builder.WriteString("<li>")
				builder.WriteString(renderReportText(item))
				builder.WriteString("</li>")
			}
			builder.WriteString("</ul>")
		}
		builder.WriteString("</section>")
	}
	builder.WriteString("</body></html>")
	return builder.String(), nil
}

func renderReportText(value string) string {
	return strings.ReplaceAll(html.EscapeString(strings.TrimSpace(value)), "\n", "<br>")
}
