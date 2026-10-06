package types

const (
	OrganizeTemplateEngineEnabledSetting     = "organize.template_engine.enabled"
	OrganizeTemplateEngineObserveOnlySetting = "organize.template_engine.observe_only"
	OrganizeSSEEnabledSetting                = "organize.sse.enabled"
	OrganizeCustomRequirementEnabledSetting  = "organize.custom_requirement.enabled"
	OrganizeDiscoverCategoriesEnabledSetting = "organize.discover_categories.enabled"
)

// OrganizeFeatureFlags is the compatibility boundary for the organize
// refactor. New execution paths must be opt-in until their baseline has
// passed regression and observation checks.
type OrganizeFeatureFlags struct {
	TemplateEngineEnabled     bool `json:"template_engine_enabled"`
	TemplateEngineObserveOnly bool `json:"template_engine_observe_only"`
	SSEEnabled                bool `json:"sse_enabled"`
	CustomRequirementEnabled  bool `json:"custom_requirement_enabled"`
	DiscoverCategoriesEnabled bool `json:"discover_categories_enabled"`
}
