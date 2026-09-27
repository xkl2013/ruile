package types

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestServiceSpaceBlueprintValidatesStructuredDraft(t *testing.T) {
	blueprint := ServiceSpaceBlueprint{
		SourceInstruction: "围绕项目参与者跟进进展、风险和下一步动作",
		ProposedSpaceType: ServiceSpaceTypeCustomerService,
		SubjectPolicy:     ServiceSubjectPolicy{Required: true, AllowedTypes: []string{"project_participant"}, AllowHierarchy: true},
		ProfileSchema:     []ServiceSpaceProfileField{{Key: "progress", Label: "进展", ValueType: "text", Source: "facts"}},
		SummarySchema:     []ServiceSpaceSummarySection{{Key: "next_actions", Label: "下一步动作", RefreshPolicy: "on_fact_change"}},
		Status:            ServiceSpaceBlueprintStatusDraft,
		Version:           1,
	}

	require.NoError(t, blueprint.Validate())
}

func TestServiceSpaceBlueprintRejectsUnsafeOrMalformedDraft(t *testing.T) {
	tests := []ServiceSpaceBlueprint{
		{
			SourceInstruction: "指令",
			ProposedSpaceType: "unsupported",
			Status:            ServiceSpaceBlueprintStatusDraft,
			Version:           1,
		},
		{
			SourceInstruction: "指令",
			ProposedSpaceType: ServiceSpaceTypeOperations,
			Status:            ServiceSpaceBlueprintStatusDraft,
			Version:           1,
			ProfileSchema:     []ServiceSpaceProfileField{{Key: "status", Label: "状态", ValueType: "text"}, {Key: "status", Label: "重复", ValueType: "text"}},
		},
		{
			SourceInstruction: "指令",
			ProposedSpaceType: ServiceSpaceTypeResearch,
			Status:            ServiceSpaceBlueprintStatusDraft,
			Version:           1,
			SummarySchema:     []ServiceSpaceSummarySection{{Key: "risk", Label: "风险", RefreshPolicy: ""}},
		},
		{
			SourceInstruction: strings.Repeat("x", MaxCustomPromptInstructionsLength+1),
			ProposedSpaceType: ServiceSpaceTypeResearch,
			Status:            ServiceSpaceBlueprintStatusDraft,
			Version:           1,
		},
	}

	for _, blueprint := range tests {
		require.Error(t, blueprint.Validate())
	}
}

func TestServiceSpaceBlueprintInputValidation(t *testing.T) {
	require.NoError(t, (ServiceSpaceBlueprintPreviewInput{
		Instruction: "跟进项目参与者的风险和下一步动作",
	}).Validate())
	require.Error(t, (ServiceSpaceBlueprintPreviewInput{}).Validate())

	require.NoError(t, (ServiceSpaceBlueprintConfirmInput{
		BlueprintID:     "bp_001",
		ExpectedVersion: 1,
		IdempotencyKey:  "confirm-bp-001",
	}).Validate())
	require.Error(t, (ServiceSpaceBlueprintConfirmInput{
		BlueprintID:     "bp_001",
		ExpectedVersion: 0,
		IdempotencyKey:  "confirm-bp-001",
	}).Validate())
	require.Error(t, (ServiceSpaceBlueprintConfirmInput{
		BlueprintID:     "bp_001",
		ExpectedVersion: 1,
	}).Validate())
}

func TestServiceSpaceTemplateValidatesPublishedAutoApplyBlueprint(t *testing.T) {
	template := ServiceSpaceTemplate{
		Key:        "project-follow-up",
		Name:       "项目参与者跟进",
		Version:    2,
		Status:     ServiceSpaceTemplateStatusPublished,
		AutoApply:  true,
		RiskLevel:  ServiceSpaceTemplateRiskLow,
		MatchRules: ServiceSpaceTemplateMatchRules{MinConfidencePct: 90},
		Blueprint: ServiceSpaceBlueprint{
			SourceType:          ServiceSpaceBlueprintSourceTemplate,
			SourceInstruction:   "围绕项目参与者跟进进展、风险和下一步动作",
			ProposedSpaceType:   ServiceSpaceTypeCustomerService,
			ProposedTemplateKey: "project-follow-up",
			TemplateVersion:     2,
			Status:              ServiceSpaceBlueprintStatusConfirmed,
			ConfirmationMode:    ServiceSpaceBlueprintConfirmationAutoApply,
			Version:             1,
			ProfileSchema:       []ServiceSpaceProfileField{{Key: "progress", Label: "进展", ValueType: "text"}},
			SummarySchema:       []ServiceSpaceSummarySection{{Key: "next_actions", Label: "下一步动作", RefreshPolicy: "on_fact_change"}},
		},
	}

	require.NoError(t, template.Validate())
}

func TestServiceSpaceTemplateRejectsUnsafeAutoApply(t *testing.T) {
	template := ServiceSpaceTemplate{
		Key:       "high-risk",
		Name:      "高风险模板",
		Version:   1,
		Status:    ServiceSpaceTemplateStatusPublished,
		AutoApply: true,
		RiskLevel: ServiceSpaceTemplateRiskHigh,
	}

	require.Error(t, template.Validate())

	require.Error(t, (ServiceSpaceTemplateApplyInput{
		TemplateKey:    "project-follow-up",
		Name:           "项目跟进",
		IdempotencyKey: "",
	}).Validate())
}

func TestServiceSpaceTemplateAllowsPublishedManualRecommendation(t *testing.T) {
	template := ServiceSpaceTemplate{
		Key:       "manual-review",
		Name:      "需要确认的推荐模板",
		Version:   1,
		Status:    ServiceSpaceTemplateStatusPublished,
		RiskLevel: ServiceSpaceTemplateRiskMedium,
		Blueprint: ServiceSpaceBlueprint{
			SourceType:          ServiceSpaceBlueprintSourceTemplate,
			SourceInstruction:   "先生成空间结构，再由用户确认",
			ProposedSpaceType:   ServiceSpaceTypeResearch,
			ProposedTemplateKey: "manual-review",
			TemplateVersion:     1,
			Status:              ServiceSpaceBlueprintStatusConfirmed,
			ConfirmationMode:    ServiceSpaceBlueprintConfirmationManual,
			Version:             1,
		},
	}

	require.NoError(t, template.Validate())
}
