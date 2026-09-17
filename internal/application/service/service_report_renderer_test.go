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
				Content: "不要执行 <img src=x onerror=alert(1)>。",
			},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rendered, "客户 &lt;script&gt;alert(1)&lt;/script&gt; 报告")
	require.Contains(t, rendered, "&lt;img src=x onerror=alert(1)&gt;")
	require.NotContains(t, rendered, "<script>alert(1)</script>")
	require.NotContains(t, rendered, "<img src=x onerror=alert(1)>")
}
