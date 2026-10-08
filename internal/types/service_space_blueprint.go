package types

import (
	"fmt"
	"strings"
	"time"
)

type ServiceSpaceBlueprintStatus string

const (
	ServiceSpaceBlueprintStatusDraft     ServiceSpaceBlueprintStatus = "draft"
	ServiceSpaceBlueprintStatusConfirmed ServiceSpaceBlueprintStatus = "confirmed"
	ServiceSpaceBlueprintStatusRejected  ServiceSpaceBlueprintStatus = "rejected"
	ServiceSpaceBlueprintStatusExpired   ServiceSpaceBlueprintStatus = "expired"
)

type ServiceSpaceBlueprintSourceType string

const (
	ServiceSpaceBlueprintSourceInstruction ServiceSpaceBlueprintSourceType = "instruction"
	ServiceSpaceBlueprintSourceTemplate    ServiceSpaceBlueprintSourceType = "template"
)

type ServiceSpaceBlueprintConfirmationMode string

const (
	ServiceSpaceBlueprintConfirmationPending              ServiceSpaceBlueprintConfirmationMode = "pending"
	ServiceSpaceBlueprintConfirmationManual               ServiceSpaceBlueprintConfirmationMode = "manual"
	ServiceSpaceBlueprintConfirmationAutoApply            ServiceSpaceBlueprintConfirmationMode = "template_auto_apply"
	ServiceSpaceBlueprintConfirmationInstructionAutoApply ServiceSpaceBlueprintConfirmationMode = "instruction_auto_apply"
)

func (s ServiceSpaceBlueprintStatus) IsValid() bool {
	switch s {
	case ServiceSpaceBlueprintStatusDraft,
		ServiceSpaceBlueprintStatusConfirmed,
		ServiceSpaceBlueprintStatusRejected,
		ServiceSpaceBlueprintStatusExpired:
		return true
	default:
		return false
	}
}

func (s ServiceSpaceBlueprintSourceType) IsValid() bool {
	switch s {
	case ServiceSpaceBlueprintSourceInstruction, ServiceSpaceBlueprintSourceTemplate:
		return true
	default:
		return false
	}
}

func (m ServiceSpaceBlueprintConfirmationMode) IsValid() bool {
	switch m {
	case ServiceSpaceBlueprintConfirmationPending,
		ServiceSpaceBlueprintConfirmationManual,
		ServiceSpaceBlueprintConfirmationAutoApply,
		ServiceSpaceBlueprintConfirmationInstructionAutoApply:
		return true
	default:
		return false
	}
}

// ServiceSubjectPolicy describes the subject shape proposed by a blueprint.
// AllowedTypes are tenant/template-configured keys, not a platform enum.
type ServiceSubjectPolicy struct {
	Required       bool     `json:"required"`
	AllowedTypes   []string `json:"allowed_types,omitempty"`
	AllowHierarchy bool     `json:"allow_hierarchy"`
}

type ServiceSpaceProfileField struct {
	Key                 string   `json:"key"`
	Label               string   `json:"label"`
	ValueType           string   `json:"value_type"`
	Source              string   `json:"source"`
	Required            bool     `json:"required"`
	Sensitive           bool     `json:"sensitive"`
	DisplayOrder        int      `json:"display_order"`
	Aliases             []string `json:"aliases,omitempty"`
	ExtractionHint      string   `json:"extraction_hint,omitempty"`
	ConfidenceThreshold float64  `json:"confidence_threshold,omitempty"`
	OverwritePolicy     string   `json:"overwrite_policy,omitempty"`
	AskWhenMissing      bool     `json:"ask_when_missing,omitempty"`
}

type ServiceSpaceSummarySection struct {
	Key           string   `json:"key"`
	Label         string   `json:"label"`
	SourceScopes  []string `json:"source_scopes,omitempty"`
	RefreshPolicy string   `json:"refresh_policy"`
	DisplayOrder  int      `json:"display_order"`
}

type ServiceSpaceBlueprintExpertSuggestion struct {
	ExpertRef  string `json:"expert_ref"`
	ExpertName string `json:"expert_name"`
	Reason     string `json:"reason,omitempty"`
}

type ServiceSpaceBlueprintPreviewInput struct {
	Instruction   string `json:"instruction"`
	BaseServiceID string `json:"base_service_id,omitempty"`
}

func (i ServiceSpaceBlueprintPreviewInput) Validate() error {
	if strings.TrimSpace(i.Instruction) == "" {
		return fmt.Errorf("blueprint instruction is required")
	}
	if len([]rune(i.Instruction)) > MaxCustomPromptInstructionsLength {
		return fmt.Errorf("blueprint instruction exceeds %d characters", MaxCustomPromptInstructionsLength)
	}
	return nil
}

type ServiceSpaceBlueprintConfirmInput struct {
	BlueprintID     string `json:"blueprint_id"`
	ExpectedVersion int    `json:"expected_version"`
	Activate        bool   `json:"activate,omitempty"`
	IdempotencyKey  string `json:"idempotency_key"`
}

func (i ServiceSpaceBlueprintConfirmInput) Validate() error {
	if strings.TrimSpace(i.BlueprintID) == "" {
		return fmt.Errorf("blueprint id is required")
	}
	if i.ExpectedVersion < 1 {
		return fmt.Errorf("expected blueprint version must be positive")
	}
	if strings.TrimSpace(i.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key is required")
	}
	return nil
}

type ServiceSpaceTemplateStatus string

const (
	ServiceSpaceTemplateStatusDraft     ServiceSpaceTemplateStatus = "draft"
	ServiceSpaceTemplateStatusPublished ServiceSpaceTemplateStatus = "published"
	ServiceSpaceTemplateStatusDisabled  ServiceSpaceTemplateStatus = "disabled"
	ServiceSpaceTemplateStatusArchived  ServiceSpaceTemplateStatus = "archived"
)

func (s ServiceSpaceTemplateStatus) IsValid() bool {
	switch s {
	case ServiceSpaceTemplateStatusDraft,
		ServiceSpaceTemplateStatusPublished,
		ServiceSpaceTemplateStatusDisabled,
		ServiceSpaceTemplateStatusArchived:
		return true
	default:
		return false
	}
}

type ServiceSpaceTemplateRiskLevel string

const (
	ServiceSpaceTemplateRiskLow    ServiceSpaceTemplateRiskLevel = "low"
	ServiceSpaceTemplateRiskMedium ServiceSpaceTemplateRiskLevel = "medium"
	ServiceSpaceTemplateRiskHigh   ServiceSpaceTemplateRiskLevel = "high"
)

func (r ServiceSpaceTemplateRiskLevel) IsValid() bool {
	switch r {
	case ServiceSpaceTemplateRiskLow, ServiceSpaceTemplateRiskMedium, ServiceSpaceTemplateRiskHigh:
		return true
	default:
		return false
	}
}

type ServiceSpaceTemplateMatchRules struct {
	Roles            []TenantRole `json:"roles,omitempty"`
	Departments      []string     `json:"departments,omitempty"`
	WorkProfileTerms []string     `json:"work_profile_terms,omitempty"`
	ServiceScopeKeys []string     `json:"service_scope_keys,omitempty"`
	RequiredClaims   []string     `json:"required_claims,omitempty"`
	MinConfidencePct int          `json:"min_confidence_pct"`
}

type ServiceUserProfileSnapshot struct {
	UserID                 string     `json:"user_id"`
	TenantID               uint64     `json:"tenant_id"`
	Version                int        `json:"version"`
	Role                   TenantRole `json:"role"`
	Department             string     `json:"department,omitempty"`
	WorkProfileDescription string     `json:"work_profile_description,omitempty"`
	ServiceScopeKeys       []string   `json:"service_scope_keys,omitempty"`
	ReadableKnowledgeBases []string   `json:"readable_knowledge_bases,omitempty"`
	Attributes             JSONMap    `json:"attributes,omitempty"`
	Hash                   string     `json:"hash"`
}

type ServiceSpaceTemplate struct {
	ID           string                         `json:"id"`
	TenantID     uint64                         `json:"tenant_id"`
	Key          string                         `json:"key"`
	Name         string                         `json:"name"`
	Version      int                            `json:"version"`
	Status       ServiceSpaceTemplateStatus     `json:"status"`
	MatchRules   ServiceSpaceTemplateMatchRules `json:"match_rules"`
	Blueprint    ServiceSpaceBlueprint          `json:"blueprint"`
	AutoApply    bool                           `json:"auto_apply"`
	AutoActivate bool                           `json:"auto_activate"`
	RiskLevel    ServiceSpaceTemplateRiskLevel  `json:"risk_level"`
	PublishedBy  string                         `json:"published_by,omitempty"`
	PublishedAt  *time.Time                     `json:"published_at,omitempty"`
	CreatedAt    time.Time                      `json:"created_at"`
	UpdatedAt    time.Time                      `json:"updated_at"`
}

func (t *ServiceSpaceTemplate) Validate() error {
	if strings.TrimSpace(t.Key) == "" {
		return fmt.Errorf("template key is required")
	}
	if strings.TrimSpace(t.Name) == "" {
		return fmt.Errorf("template name is required")
	}
	if t.Version < 1 {
		return fmt.Errorf("template version must be positive")
	}
	if !t.Status.IsValid() {
		return fmt.Errorf("invalid service space template status")
	}
	if !t.RiskLevel.IsValid() {
		return fmt.Errorf("invalid service space template risk level")
	}
	if t.MatchRules.MinConfidencePct < 0 || t.MatchRules.MinConfidencePct > 100 {
		return fmt.Errorf("template minimum confidence must be between 0 and 100")
	}
	if t.AutoApply && t.RiskLevel != ServiceSpaceTemplateRiskLow {
		return fmt.Errorf("only low-risk templates can be auto-applied")
	}
	if t.AutoActivate && !t.AutoApply {
		return fmt.Errorf("auto-activate requires auto-apply")
	}
	if t.Blueprint.SourceType != ServiceSpaceBlueprintSourceTemplate {
		return fmt.Errorf("template blueprint source type must be template")
	}
	if t.Blueprint.ProposedTemplateKey != t.Key {
		return fmt.Errorf("template blueprint key must match template key")
	}
	if t.Blueprint.TemplateVersion != t.Version {
		return fmt.Errorf("template blueprint version must match template version")
	}
	if t.Blueprint.Status != ServiceSpaceBlueprintStatusConfirmed {
		return fmt.Errorf("template blueprint must be confirmed")
	}
	if t.AutoApply && t.Blueprint.ConfirmationMode != ServiceSpaceBlueprintConfirmationAutoApply {
		return fmt.Errorf("auto-applied template blueprint must use auto-apply confirmation mode")
	}
	if !t.AutoApply && t.Blueprint.ConfirmationMode != ServiceSpaceBlueprintConfirmationManual &&
		t.Blueprint.ConfirmationMode != ServiceSpaceBlueprintConfirmationAutoApply {
		return fmt.Errorf("published template blueprint must use a supported confirmation mode")
	}
	return t.Blueprint.Validate()
}

type ServiceSpaceTemplateApplyInput struct {
	TemplateKey      string                      `json:"template_key"`
	TemplateVersion  int                         `json:"template_version,omitempty"`
	Name             string                      `json:"name"`
	SpaceType        ServiceSpaceType            `json:"space_type,omitempty"`
	Description      string                      `json:"description,omitempty"`
	Instruction      string                      `json:"instruction,omitempty"`
	IdempotencyKey   string                      `json:"idempotency_key"`
	KnowledgeBaseIDs StringArray                 `json:"knowledge_base_ids,omitempty"`
	Experts          []ServiceExpertBindingInput `json:"experts,omitempty"`
}

func (i ServiceSpaceTemplateApplyInput) Validate() error {
	if strings.TrimSpace(i.TemplateKey) == "" {
		return fmt.Errorf("template key is required")
	}
	if i.TemplateVersion < 0 {
		return fmt.Errorf("template version cannot be negative")
	}
	if strings.TrimSpace(i.Name) == "" {
		return fmt.Errorf("service space name is required")
	}
	if i.SpaceType != "" && !i.SpaceType.IsValid() {
		return fmt.Errorf("invalid service space type")
	}
	if strings.TrimSpace(i.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key is required")
	}
	return nil
}

type ServiceSpaceTemplateApplicationResult string

const (
	ServiceSpaceTemplateApplicationApplied              ServiceSpaceTemplateApplicationResult = "applied"
	ServiceSpaceTemplateApplicationRejected             ServiceSpaceTemplateApplicationResult = "rejected"
	ServiceSpaceTemplateApplicationFallbackConfirmation ServiceSpaceTemplateApplicationResult = "fallback_confirmation"
)

type ServiceSpaceTemplateApplication struct {
	ID              string                                `json:"id"`
	TenantID        uint64                                `json:"tenant_id"`
	ServiceID       string                                `json:"service_id"`
	TemplateKey     string                                `json:"template_key"`
	TemplateVersion int                                   `json:"template_version"`
	ProfileVersion  int                                   `json:"profile_version"`
	ProfileHash     string                                `json:"profile_hash"`
	MatchReason     JSONMap                               `json:"match_reason"`
	ApplyMode       ServiceSpaceBlueprintConfirmationMode `json:"apply_mode"`
	IdempotencyKey  string                                `json:"idempotency_key"`
	Result          ServiceSpaceTemplateApplicationResult `json:"result"`
	CreatedAt       time.Time                             `json:"created_at"`
}

// ServiceSpaceBlueprint is the structured, reviewable result of interpreting
// a natural-language service instruction. It is a transport/validation
// contract; V2 persistence and model execution are intentionally separate.
type ServiceSpaceBlueprint struct {
	ID                        string                                  `json:"id"`
	TenantID                  uint64                                  `json:"tenant_id"`
	ServiceID                 string                                  `json:"service_id,omitempty"`
	SourceType                ServiceSpaceBlueprintSourceType         `json:"source_type,omitempty"`
	SourceInstruction         string                                  `json:"source_instruction"`
	ProposedSpaceType         ServiceSpaceType                        `json:"proposed_space_type"`
	ProposedTemplateKey       string                                  `json:"proposed_template_key,omitempty"`
	TemplateVersion           int                                     `json:"template_version,omitempty"`
	SubjectPolicy             ServiceSubjectPolicy                    `json:"subject_policy"`
	ProfileSchema             []ServiceSpaceProfileField              `json:"profile_schema"`
	SummarySchema             []ServiceSpaceSummarySection            `json:"summary_schema"`
	ExpertSuggestions         []ServiceSpaceBlueprintExpertSuggestion `json:"expert_suggestions,omitempty"`
	KnowledgeScopeSuggestions JSONMap                                 `json:"knowledge_scope_suggestions,omitempty"`
	GenerationMeta            JSONMap                                 `json:"generation_meta,omitempty"`
	Status                    ServiceSpaceBlueprintStatus             `json:"status"`
	ConfirmationMode          ServiceSpaceBlueprintConfirmationMode   `json:"confirmation_mode,omitempty"`
	ProfileVersion            int                                     `json:"profile_version,omitempty"`
	ProfileHash               string                                  `json:"profile_hash,omitempty"`
	Version                   int                                     `json:"version"`
	ConfirmedBy               string                                  `json:"confirmed_by,omitempty"`
	ConfirmedAt               *time.Time                              `json:"confirmed_at,omitempty"`
	CreatedAt                 time.Time                               `json:"created_at"`
	UpdatedAt                 time.Time                               `json:"updated_at"`
}

func (b *ServiceSpaceBlueprint) Validate() error {
	if strings.TrimSpace(b.SourceInstruction) == "" {
		return fmt.Errorf("blueprint source instruction is required")
	}
	if len([]rune(b.SourceInstruction)) > MaxCustomPromptInstructionsLength {
		return fmt.Errorf("blueprint source instruction exceeds %d characters", MaxCustomPromptInstructionsLength)
	}
	if !b.ProposedSpaceType.IsValid() {
		return fmt.Errorf("invalid proposed service space type")
	}
	if b.SourceType != "" && !b.SourceType.IsValid() {
		return fmt.Errorf("invalid service space blueprint source type")
	}
	if b.ConfirmationMode != "" && !b.ConfirmationMode.IsValid() {
		return fmt.Errorf("invalid service space blueprint confirmation mode")
	}
	if !b.Status.IsValid() {
		return fmt.Errorf("invalid service space blueprint status")
	}
	if b.Version < 1 {
		return fmt.Errorf("blueprint version must be positive")
	}
	if b.SourceType == ServiceSpaceBlueprintSourceTemplate {
		if strings.TrimSpace(b.ProposedTemplateKey) == "" {
			return fmt.Errorf("template blueprint key is required")
		}
		if b.TemplateVersion < 1 {
			return fmt.Errorf("template blueprint version must be positive")
		}
		if b.Status != ServiceSpaceBlueprintStatusConfirmed {
			return fmt.Errorf("template blueprint must be confirmed")
		}
		if b.ConfirmationMode != ServiceSpaceBlueprintConfirmationManual &&
			b.ConfirmationMode != ServiceSpaceBlueprintConfirmationAutoApply {
			return fmt.Errorf("template blueprint must use a supported confirmation mode")
		}
	}
	if err := validateBlueprintFields(b.ProfileSchema); err != nil {
		return fmt.Errorf("profile schema: %w", err)
	}
	if err := validateBlueprintSummarySections(b.SummarySchema); err != nil {
		return fmt.Errorf("summary schema: %w", err)
	}
	return nil
}

func validateBlueprintFields(fields []ServiceSpaceProfileField) error {
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return fmt.Errorf("field key is required")
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate field key %q", key)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("field %q label is required", key)
		}
		if strings.TrimSpace(field.ValueType) == "" {
			return fmt.Errorf("field %q value_type is required", key)
		}
	}
	return nil
}

func validateBlueprintSummarySections(sections []ServiceSpaceSummarySection) error {
	seen := make(map[string]struct{}, len(sections))
	for _, section := range sections {
		key := strings.TrimSpace(section.Key)
		if key == "" {
			return fmt.Errorf("summary section key is required")
		}
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate summary section key %q", key)
		}
		seen[key] = struct{}{}
		if strings.TrimSpace(section.Label) == "" {
			return fmt.Errorf("summary section %q label is required", key)
		}
		if strings.TrimSpace(section.RefreshPolicy) == "" {
			return fmt.Errorf("summary section %q refresh_policy is required", key)
		}
	}
	return nil
}
