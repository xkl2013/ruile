package session

import (
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestComposeServiceFactProposalTextUsesConfirmedAssistantFields(t *testing.T) {
	now := time.Now()
	messages := []*types.Message{
		{
			ID:        "user-previous",
			RequestID: "request-previous",
			Role:      "user",
			CreatedAt: now.Add(-2 * time.Minute),
			Content:   "孩子年龄是10岁，班级已从4班更新为7班。",
		},
		{
			ID:          "assistant-previous",
			RequestID:   "request-previous",
			Role:        "assistant",
			IsCompleted: true,
			CreatedAt:   now.Add(-time.Minute),
			Content: `请确认以下信息：

- **孩子年龄**：10岁
- **班级**：已从4班更新为7班

还可以补充的信息，例如：
- 孩子姓名
- 主班老师`,
		},
		{
			ID:          "assistant-current",
			Role:        "assistant",
			IsCompleted: false,
			CreatedAt:   now,
		},
	}

	text := composeServiceFactProposalText("更新档案", messages, "assistant-current")

	require.Contains(t, text, "用户确认写入的上一轮候选信息")
	require.Contains(t, text, "上一轮用户提供的信息")
	require.Contains(t, text, "孩子年龄是10岁，班级已从4班更新为7班。")
	require.Contains(t, text, "孩子年龄：10岁")
	require.Contains(t, text, "班级：已从4班更新为7班")
	require.NotContains(t, text, "孩子姓名")
	require.NotContains(t, text, "主班老师")
}

func TestComposeServiceFactProposalTextUsesPreviousUserEvidenceWithoutAssistantFields(t *testing.T) {
	now := time.Now()
	messages := []*types.Message{
		{
			ID:        "user-previous",
			RequestID: "request-previous",
			Role:      "user",
			CreatedAt: now.Add(-2 * time.Minute),
			Content:   "孩子年龄是10岁，班级已从4班更新为7班。",
		},
		{
			ID:          "assistant-previous",
			RequestID:   "request-previous",
			Role:        "assistant",
			IsCompleted: true,
			CreatedAt:   now.Add(-time.Minute),
			Content:     "是否确认把这些信息更新到服务档案？",
		},
	}

	text := composeServiceFactProposalText("确认更新", messages, "")

	require.Contains(t, text, "孩子年龄是10岁，班级已从4班更新为7班。")
	require.NotContains(t, text, "上一轮助手列出的候选字段")
}

func TestComposeServiceFactProposalTextDoesNotTreatQuestionAsConfirmation(t *testing.T) {
	messages := []*types.Message{{
		ID:          "assistant-previous",
		Role:        "assistant",
		IsCompleted: true,
		Content:     "孩子年龄：10岁",
	}}

	text := composeServiceFactProposalText("为什么没有更新档案", messages, "")

	require.Equal(t, "为什么没有更新档案", text)
}

func TestExtractConfirmedServiceFactLinesRejectsPlaceholders(t *testing.T) {
	content := `档案信息：
- 年龄：10岁
- 班级：待补充
- 主班老师：未提供
- 课程偏好：周末上午`

	require.Equal(t, "年龄：10岁\n课程偏好：周末上午", extractConfirmedServiceFactLines(content))
}

func TestExtractConfirmedServiceFactLinesMatchesArchiveUpdateReply(t *testing.T) {
	content := `我已收到您的确认，现在更新服务档案。

档案更新完成：

- 孩子年龄：10岁
- 班级：已从4班更新为7班

档案已成功更新。请问还有其他需要补充的信息吗？例如：

- 孩子姓名
- 主班老师
- 课程或服务偏好`

	require.Equal(
		t,
		"孩子年龄：10岁\n班级：已从4班更新为7班",
		extractConfirmedServiceFactLines(content),
	)
}
