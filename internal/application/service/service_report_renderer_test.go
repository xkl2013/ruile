package service

import (
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestRenderStructuredReportHTMLEscapesReportText(t *testing.T) {
	rendered, err := renderStructuredReportHTML(types.StructuredReportV1{
		Format:           types.StructuredReportFormatV1,
		Title:            "客户 <script>alert(1)</script> 报告",
		ExecutiveSummary: "需要确认 <b>实际</b> 进展。",
		Sections: []types.StructuredReportSection{
			{
				Type:    "analysis",
				Title:   "判断",
				Content: "不要执行 <img src=x onerror=alert(1)>。\n\n### 物料清单\n\n| 类别 | 数量 |\n| --- | --- |\n| **道具** | 5个 |",
			},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rendered, "客户 &lt;script&gt;alert(1)&lt;/script&gt; 报告")
	require.Contains(t, rendered, "&lt;img src=x onerror=alert(1)&gt;")
	require.NotContains(t, rendered, "<script>alert(1)</script>")
	require.NotContains(t, rendered, "<img src=x onerror=alert(1)>")
	require.Contains(t, rendered, "<h3>物料清单</h3>")
	require.Contains(t, rendered, "<table>")
	require.Contains(t, rendered, "<strong>道具</strong>")
	require.NotContains(t, rendered, "| --- | --- |")
	require.Contains(t, rendered, `class="report-hero"`)
	require.Contains(t, rendered, `class="report-summary"`)
	require.Contains(t, rendered, `class="report-outline"`)
	require.Contains(t, rendered, `class="report-section report-section--analysis"`)
	require.Contains(t, rendered, `@media print`)
	require.Contains(t, rendered, `@media (prefers-color-scheme: dark)`)
}

func TestRenderStructuredReportMarkdown(t *testing.T) {
	rendered, err := renderStructuredReportMarkdown(types.StructuredReportV1{
		Format:           types.StructuredReportFormatV1,
		Title:            "亲子活动方案",
		ExecutiveSummary: "按三个阶段推进。",
		Sections: []types.StructuredReportSection{
			{
				Type:    "recommended_actions",
				Title:   "执行安排",
				Content: "先确认场地。",
				Items:   types.StringArray{"完成分工", "采购物料"},
			},
		},
		EvidenceRefs: types.StringArray{"用户确认：30组家庭"},
	})

	require.NoError(t, err)
	require.Contains(t, rendered, "# 亲子活动方案")
	require.Contains(t, rendered, "## 执行摘要")
	require.Contains(t, rendered, "## 执行安排")
	require.Contains(t, rendered, "- 完成分工")
	require.Contains(t, rendered, "## 依据与引用")
}
