package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNormalizeAgentResultV1AddsArtifactIdentityAndDeliveryMetadata(t *testing.T) {
	now := time.Date(2026, 9, 22, 3, 4, 5, 0, time.UTC)
	result := NormalizeAgentResultV1(AgentResultV1{
		SchemaVersion: AgentResultSchemaV1,
		Decision: AgentResultDecisionV1{
			Confidence: 0.8,
			Reason:     "需要输出一份结构化报告。",
		},
		Artifacts: []AgentArtifactResultV1{
			{
				Kind:   AgentArtifactKindReport,
				Role:   AgentArtifactRolePrimary,
				Title:  "执行报告",
				Format: StructuredReportFormatV1,
			},
		},
	}, "run-1", now)

	require.Len(t, result.Artifacts, 1)
	artifact := result.Artifacts[0]
	require.NotEmpty(t, artifact.ID)
	require.NotEmpty(t, artifact.VersionID)
	require.Equal(t, 1, artifact.Version)
	require.Equal(t, "run-1", artifact.RunID)
	require.Equal(t, AgentArtifactLifecycleTemporary, artifact.Lifecycle)
	require.True(t, artifact.Previewable)
	require.False(t, artifact.Downloadable)
	require.Equal(t, now, *artifact.CreatedAt)
	require.NotNil(t, artifact.Metadata)
}

func TestValidateAgentResultV1RejectsInvalidArtifactVersionMetadata(t *testing.T) {
	validation := ValidateAgentResultV1(AgentResultV1{
		SchemaVersion: AgentResultSchemaV1,
		Decision: AgentResultDecisionV1{
			Confidence: 0.8,
			Reason:     "输出有效。",
		},
		Artifacts: []AgentArtifactResultV1{
			{
				ID:        "",
				VersionID: "version-1",
				Version:   -1,
				Kind:      AgentArtifactKindText,
				Role:      AgentArtifactRolePrimary,
				Title:     "文本产物",
				Lifecycle: "invalid",
			},
		},
	})

	require.False(t, validation.Valid)
	require.Contains(t, validation.Errors, "artifacts[0].version must be greater than or equal to 0")
	require.Contains(t, validation.Errors, "artifacts[0].id is required when version_id is provided")
	require.Contains(t, validation.Errors, "artifacts[0].lifecycle is not supported")
}

func TestValidateAgentResultV1AcceptsServiceCardAndStructuredReport(t *testing.T) {
	report := StructuredReportV1{
		Format:           StructuredReportFormatV1,
		Title:            "客户跟进分析",
		ExecutiveSummary: "客户等待明确答复，需要先确认内部安排。",
		Sections: []StructuredReportSection{
			{
				Type:  "facts",
				Title: "已知事实",
				Items: StringArray{"客户已询问处理时间。"},
			},
		},
		EvidenceRefs: StringArray{"memory-1"},
	}
	content, err := report.ToJSONMap()
	require.NoError(t, err)

	validation := ValidateAgentResultV1(AgentResultV1{
		SchemaVersion: AgentResultSchemaV1,
		Decision: AgentResultDecisionV1{
			ShouldCreateCard: true,
			Confidence:       0.88,
			Reason:           "客户问题尚未闭环。",
		},
		Card: &ServiceCardV1{
			SchemaVersion: ServiceCardSchemaV1,
			Title:         "跟进客户处理时间",
			Summary:       "客户正在等待明确处理进度。",
			NextAction:    "确认内部安排后回复客户。",
		},
		Artifacts: []AgentArtifactResultV1{
			{
				Kind:    AgentArtifactKindReport,
				Role:    AgentArtifactRolePrimary,
				Title:   report.Title,
				Format:  StructuredReportFormatV1,
				Content: content,
			},
		},
		Evidence: []AgentEvidenceRefV1{
			{SourceType: "memory", SourceID: "memory-1", Relation: "trigger"},
		},
	})

	require.True(t, validation.Valid)
	require.Empty(t, validation.Errors)
}

func TestValidateAgentResultV1RejectsIncompleteCardAndReport(t *testing.T) {
	validation := ValidateAgentResultV1(AgentResultV1{
		SchemaVersion: AgentResultSchemaV1,
		Decision: AgentResultDecisionV1{
			ShouldCreateCard: true,
			Confidence:       1.2,
		},
		Card: &ServiceCardV1{
			SchemaVersion: ServiceCardSchemaV1,
			Title:         "未完成卡片",
		},
		Artifacts: []AgentArtifactResultV1{
			{
				Kind:   AgentArtifactKindReport,
				Role:   AgentArtifactRolePrimary,
				Title:  "不完整报告",
				Format: StructuredReportFormatV1,
			},
		},
	})

	require.False(t, validation.Valid)
	require.Contains(t, validation.Errors, "decision.confidence must be between 0 and 1")
	require.Contains(t, validation.Errors, "decision.reason is required")
	require.Contains(t, validation.Errors, "card.summary is required")
	require.Contains(t, validation.Errors, "card.next_action is required")
	require.Contains(t, validation.Errors, "artifacts[0].content is required for report artifacts")
}
