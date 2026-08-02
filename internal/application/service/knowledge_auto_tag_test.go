package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/assert"
)

func TestParseAutoTagResponse_JSONAndCleanup(t *testing.T) {
	t.Parallel()

	got := parseAutoTagResponse("```json\n{\"tags\":[\"合同\", \"合同\", \"document\", \"财务报表\"]}\n```")

	assert.Equal(t, []string{"合同", "财务报表"}, got)
}

func TestParseAutoTagResponse_FallbackList(t *testing.T) {
	t.Parallel()

	got := parseAutoTagResponse("法务, 销售；其他\n客户成功")

	assert.Equal(t, []string{"法务", "销售", "客户成功"}, got)
}

func TestBuildAutoTagContentSortsAndSamples(t *testing.T) {
	t.Parallel()

	longTail := strings.Repeat("尾部", 4000)
	got := buildAutoTagContent(&types.Knowledge{Title: "Q3 Report"}, []*types.Chunk{
		{ChunkType: types.ChunkTypeText, StartAt: 20, Content: longTail},
		{ChunkType: types.ChunkTypeText, StartAt: 0, Content: "开头内容"},
		{ChunkType: types.ChunkTypeFAQ, StartAt: 10, Content: "FAQ should be ignored"},
	})

	assert.Contains(t, got, "Title: Q3 Report")
	assert.Contains(t, got, "开头内容")
	assert.Contains(t, got, autoTagOmittedMarker)
	assert.LessOrEqual(t, len([]rune(got)), autoTagMaxInputRunes+len([]rune(autoTagOmittedMarker)))
	assert.NotContains(t, got, "FAQ should be ignored")
}

func TestShouldAutoTagKnowledgeOnlyUploadedFiles(t *testing.T) {
	t.Parallel()

	assert.True(t, shouldAutoTagKnowledge(&types.Knowledge{Type: "file"}))
	assert.True(t, shouldAutoTagKnowledge(&types.Knowledge{Type: "file_url"}))
	assert.False(t, shouldAutoTagKnowledge(&types.Knowledge{Type: "url"}))
	assert.False(t, shouldAutoTagKnowledge(&types.Knowledge{Type: types.KnowledgeTypeManual}))
	assert.False(t, shouldAutoTagKnowledge(&types.Knowledge{Type: types.KnowledgeTypeFAQ}))
	assert.False(t, shouldAutoTagKnowledge(nil))
}

func TestExtractAutoTagNamesSupportsStats(t *testing.T) {
	t.Parallel()

	got := extractAutoTagNames([]*types.KnowledgeTagWithStats{
		{KnowledgeTag: types.KnowledgeTag{Name: " 财务 "}},
		{KnowledgeTag: types.KnowledgeTag{Name: "财务"}},
		{KnowledgeTag: types.KnowledgeTag{Name: "document"}},
		{KnowledgeTag: types.KnowledgeTag{Name: "产品"}},
	})

	assert.Equal(t, []string{"财务", "产品"}, got)
}
