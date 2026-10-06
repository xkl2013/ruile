package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const (
	organizeTemplateEngineEnv     = "WEKNORA_ORGANIZE_TEMPLATE_ENGINE_ENABLED"
	organizeTemplateObserveEnv    = "WEKNORA_ORGANIZE_TEMPLATE_ENGINE_OBSERVE_ONLY"
	organizeSSEEnv                = "WEKNORA_ORGANIZE_SSE_ENABLED"
	organizeCustomRequirementEnv  = "WEKNORA_ORGANIZE_CUSTOM_REQUIREMENT_ENABLED"
	organizeDiscoverCategoriesEnv = "WEKNORA_ORGANIZE_DISCOVER_CATEGORIES_ENABLED"
)

// ResolveOrganizeFeatureFlags resolves all organize rollout switches through
// the existing system-settings precedence: DB override, ENV, then false.
// A nil service deliberately returns the compatibility baseline.
func ResolveOrganizeFeatureFlags(
	ctx context.Context,
	settings interfaces.SystemSettingService,
) types.OrganizeFeatureFlags {
	if settings == nil {
		return types.OrganizeFeatureFlags{}
	}
	return types.OrganizeFeatureFlags{
		TemplateEngineEnabled: settings.GetBool(
			ctx,
			types.OrganizeTemplateEngineEnabledSetting,
			organizeTemplateEngineEnv,
			false,
		),
		TemplateEngineObserveOnly: settings.GetBool(
			ctx,
			types.OrganizeTemplateEngineObserveOnlySetting,
			organizeTemplateObserveEnv,
			false,
		),
		SSEEnabled: settings.GetBool(
			ctx,
			types.OrganizeSSEEnabledSetting,
			organizeSSEEnv,
			false,
		),
		CustomRequirementEnabled: settings.GetBool(
			ctx,
			types.OrganizeCustomRequirementEnabledSetting,
			organizeCustomRequirementEnv,
			false,
		),
		DiscoverCategoriesEnabled: settings.GetBool(
			ctx,
			types.OrganizeDiscoverCategoriesEnabledSetting,
			organizeDiscoverCategoriesEnv,
			false,
		),
	}
}

// OrganizeObservation captures a comparable old/new result pair without
// retaining prompt or content payloads in logs. Callers can record Changed
// while continuing to return the legacy result to the user.
type OrganizeObservation struct {
	Chain         string `json:"chain"`
	LegacyHash    string `json:"legacy_hash"`
	CandidateHash string `json:"candidate_hash"`
	Changed       bool   `json:"changed"`
}

func BuildOrganizeObservation(chain string, legacy, candidate any) OrganizeObservation {
	legacyHash := hashOrganizeObservationValue(legacy)
	candidateHash := hashOrganizeObservationValue(candidate)
	return OrganizeObservation{
		Chain:         chain,
		LegacyHash:    legacyHash,
		CandidateHash: candidateHash,
		Changed:       legacyHash != candidateHash,
	}
}

func hashOrganizeObservationValue(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("<unserializable>")
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
