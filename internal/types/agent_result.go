package types

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	AgentResultSchemaV1             = "agent_result_v1"
	ServiceCardSchemaV1             = "service_card_v1"
	StructuredReportFormatV1        = "structured_report_v1"
	AgentArtifactLifecycleTemporary = "temporary"
	AgentArtifactLifecycleSaved     = "saved"
	AgentArtifactLifecycleShared    = "shared"
	AgentArtifactLifecycleArchived  = "archived"
	AgentArtifactRolePrimary        = "primary"
	AgentArtifactRoleSupporting     = "supporting"
	AgentArtifactKindText           = "text"
	AgentArtifactKindReport         = "report"
	AgentArtifactKindHTML           = "html"
	AgentArtifactKindImage          = "image"
	AgentArtifactKindPDF            = "pdf"
	AgentArtifactKindDocument       = "document"
	AgentArtifactKindSpreadsheet    = "spreadsheet"
	AgentArtifactKindPresentation   = "presentation"
	AgentArtifactKindAudio          = "audio"
	AgentArtifactKindVideo          = "video"
	AgentArtifactKindData           = "data"
)

// AgentResultV1 is the platform-owned final output contract for all service
// agents. Business services decide how to persist the validated result.
type AgentResultV1 struct {
	SchemaVersion string                  `json:"schema_version"`
	Decision      AgentResultDecisionV1   `json:"decision"`
	Card          *ServiceCardV1          `json:"card,omitempty"`
	Artifacts     []AgentArtifactResultV1 `json:"artifacts"`
	Evidence      []AgentEvidenceRefV1    `json:"evidence"`
}

type AgentResultDecisionV1 struct {
	ShouldCreateCard bool    `json:"should_create_card"`
	Confidence       float64 `json:"confidence"`
	Reason           string  `json:"reason"`
}

// ServiceCardV1 intentionally contains only the three fields visible in the
// service-card body. Status, owner, evidence and agent metadata stay outside
// this object.
type ServiceCardV1 struct {
	SchemaVersion string `json:"schema_version"`
	Title         string `json:"title"`
	Summary       string `json:"summary"`
	NextAction    string `json:"next_action"`
}

type AgentArtifactResultV1 struct {
	// ID identifies the logical artifact. VersionID identifies the immutable
	// content version represented by this result.
	ID           string     `json:"id,omitempty"`
	VersionID    string     `json:"version_id,omitempty"`
	Version      int        `json:"version,omitempty"`
	RunID        string     `json:"run_id,omitempty"`
	Kind         string     `json:"kind"`
	Role         string     `json:"role"`
	Title        string     `json:"title"`
	Format       string     `json:"format,omitempty"`
	MimeType     string     `json:"mime_type,omitempty"`
	OriginalName string     `json:"original_name,omitempty"`
	SizeBytes    int64      `json:"size_bytes,omitempty"`
	ResourceRef  string     `json:"resource_ref,omitempty"`
	Lifecycle    string     `json:"lifecycle,omitempty"`
	Previewable  bool       `json:"previewable,omitempty"`
	Downloadable bool       `json:"downloadable,omitempty"`
	Shareable    bool       `json:"shareable,omitempty"`
	CreatedAt    *time.Time `json:"created_at,omitempty"`
	Metadata     JSONMap    `json:"metadata,omitempty"`
	Content      JSONMap    `json:"content,omitempty"`
}

type AgentEvidenceRefV1 struct {
	SourceType string `json:"source_type"`
	SourceID   string `json:"source_id"`
	Relation   string `json:"relation"`
	Excerpt    string `json:"excerpt,omitempty"`
}

type StructuredReportV1 struct {
	Format           string                    `json:"format"`
	Title            string                    `json:"title"`
	ExecutiveSummary string                    `json:"executive_summary"`
	Sections         []StructuredReportSection `json:"sections"`
	EvidenceRefs     StringArray               `json:"evidence_refs"`
}

type StructuredReportSection struct {
	Type    string      `json:"type"`
	Title   string      `json:"title"`
	Content string      `json:"content,omitempty"`
	Items   StringArray `json:"items,omitempty"`
}

// AgentResultValidation is persisted with an AgentRun result so that callers
// can distinguish a successful execution from a successfully validated output.
type AgentResultValidation struct {
	Contract string      `json:"contract"`
	Valid    bool        `json:"valid"`
	Errors   StringArray `json:"errors"`
}

func (r AgentResultV1) ToJSONMap() (JSONMap, error) {
	return toJSONMap(r)
}

// NormalizeAgentResultV1 enriches an output produced by an Agent with the
// platform-owned identity and delivery metadata. It deliberately keeps
// business content unchanged so older agents can continue to emit the same
// agent_result_v1 payload.
func NormalizeAgentResultV1(result AgentResultV1, runID string, now time.Time) AgentResultV1 {
	if result.Artifacts == nil {
		result.Artifacts = []AgentArtifactResultV1{}
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for index := range result.Artifacts {
		artifact := &result.Artifacts[index]
		if artifact.ID == "" {
			artifact.ID = uuid.NewString()
		}
		if artifact.VersionID == "" {
			artifact.VersionID = uuid.NewString()
		}
		if artifact.Version <= 0 {
			artifact.Version = 1
		}
		if artifact.RunID == "" {
			artifact.RunID = strings.TrimSpace(runID)
		}
		if artifact.Lifecycle == "" {
			artifact.Lifecycle = AgentArtifactLifecycleTemporary
		}
		if artifact.CreatedAt == nil {
			createdAt := now
			artifact.CreatedAt = &createdAt
		}
		if artifact.Metadata == nil {
			artifact.Metadata = JSONMap{}
		}
		if !artifact.Previewable {
			artifact.Previewable = isAgentArtifactPreviewable(*artifact)
		}
		if !artifact.Downloadable {
			artifact.Downloadable = strings.TrimSpace(artifact.ResourceRef) != ""
		}
	}
	return result
}

func (r StructuredReportV1) ToJSONMap() (JSONMap, error) {
	return toJSONMap(r)
}

func DecodeStructuredReportV1(value any) (*StructuredReportV1, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var report StructuredReportV1
	if err := json.Unmarshal(raw, &report); err != nil {
		return nil, err
	}
	return &report, nil
}

func ValidateAgentResultV1(result AgentResultV1) AgentResultValidation {
	errors := make([]string, 0)
	if result.SchemaVersion != AgentResultSchemaV1 {
		errors = append(errors, fmt.Sprintf("schema_version must be %q", AgentResultSchemaV1))
	}
	if result.Decision.Confidence < 0 || result.Decision.Confidence > 1 {
		errors = append(errors, "decision.confidence must be between 0 and 1")
	}
	if isBlank(result.Decision.Reason) {
		errors = append(errors, "decision.reason is required")
	}
	if result.Decision.ShouldCreateCard && result.Card == nil {
		errors = append(errors, "card is required when decision.should_create_card is true")
	}
	if !result.Decision.ShouldCreateCard && result.Card != nil {
		errors = append(errors, "card must be omitted when decision.should_create_card is false")
	}
	if result.Card != nil {
		errors = append(errors, validateServiceCardV1(*result.Card)...)
	}
	primaryCount := 0
	for index, artifact := range result.Artifacts {
		if artifact.Role == AgentArtifactRolePrimary {
			primaryCount++
		}
		for _, validationError := range validateAgentArtifactResultV1(artifact) {
			errors = append(errors, fmt.Sprintf("artifacts[%d].%s", index, validationError))
		}
	}
	if primaryCount > 1 {
		errors = append(errors, "only one primary artifact is allowed")
	}
	for index, evidence := range result.Evidence {
		if isBlank(evidence.SourceType) {
			errors = append(errors, fmt.Sprintf("evidence[%d].source_type is required", index))
		}
		if isBlank(evidence.SourceID) {
			errors = append(errors, fmt.Sprintf("evidence[%d].source_id is required", index))
		}
		if isBlank(evidence.Relation) {
			errors = append(errors, fmt.Sprintf("evidence[%d].relation is required", index))
		}
		if runeCount(evidence.Excerpt) > 1000 {
			errors = append(errors, fmt.Sprintf("evidence[%d].excerpt exceeds 1000 characters", index))
		}
	}
	return AgentResultValidation{
		Contract: AgentResultSchemaV1,
		Valid:    len(errors) == 0,
		Errors:   StringArray(errors),
	}
}

func validateServiceCardV1(card ServiceCardV1) []string {
	errors := make([]string, 0)
	if card.SchemaVersion != ServiceCardSchemaV1 {
		errors = append(errors, fmt.Sprintf("card.schema_version must be %q", ServiceCardSchemaV1))
	}
	errors = append(errors, validateRequiredText("card.title", card.Title, 80)...)
	errors = append(errors, validateRequiredText("card.summary", card.Summary, 320)...)
	errors = append(errors, validateRequiredText("card.next_action", card.NextAction, 320)...)
	return errors
}

func validateAgentArtifactResultV1(artifact AgentArtifactResultV1) []string {
	errors := make([]string, 0)
	if !isSupportedAgentArtifactKind(artifact.Kind) {
		errors = append(errors, "kind is not supported")
	}
	if artifact.Role != AgentArtifactRolePrimary && artifact.Role != AgentArtifactRoleSupporting {
		errors = append(errors, "role must be primary or supporting")
	}
	errors = append(errors, validateRequiredText("title", artifact.Title, 160)...)
	if artifact.Version < 0 {
		errors = append(errors, "version must be greater than or equal to 0")
	}
	if artifact.VersionID != "" && artifact.ID == "" {
		errors = append(errors, "id is required when version_id is provided")
	}
	if artifact.Lifecycle != "" && !isSupportedAgentArtifactLifecycle(artifact.Lifecycle) {
		errors = append(errors, "lifecycle is not supported")
	}
	if artifact.Kind != AgentArtifactKindReport {
		return errors
	}
	if artifact.Format != StructuredReportFormatV1 {
		errors = append(errors, fmt.Sprintf("format must be %q for report artifacts", StructuredReportFormatV1))
		return errors
	}
	if len(artifact.Content) == 0 {
		errors = append(errors, "content is required for report artifacts")
		return errors
	}
	report, err := DecodeStructuredReportV1(artifact.Content)
	if err != nil {
		errors = append(errors, "content is not a valid structured report")
		return errors
	}
	for _, validationError := range validateStructuredReportV1(*report) {
		errors = append(errors, "content."+validationError)
	}
	return errors
}

func isSupportedAgentArtifactLifecycle(lifecycle string) bool {
	switch lifecycle {
	case AgentArtifactLifecycleTemporary, AgentArtifactLifecycleSaved,
		AgentArtifactLifecycleShared, AgentArtifactLifecycleArchived:
		return true
	default:
		return false
	}
}

func isAgentArtifactPreviewable(artifact AgentArtifactResultV1) bool {
	if artifact.Kind == AgentArtifactKindReport || artifact.Kind == AgentArtifactKindText {
		return true
	}
	mimeType := strings.ToLower(strings.TrimSpace(artifact.MimeType))
	return strings.HasPrefix(mimeType, "text/") ||
		strings.HasPrefix(mimeType, "image/") ||
		strings.HasPrefix(mimeType, "audio/") ||
		strings.HasPrefix(mimeType, "video/") ||
		mimeType == "application/pdf"
}

func validateStructuredReportV1(report StructuredReportV1) []string {
	errors := make([]string, 0)
	if report.Format != StructuredReportFormatV1 {
		errors = append(errors, fmt.Sprintf("format must be %q", StructuredReportFormatV1))
	}
	errors = append(errors, validateRequiredText("title", report.Title, 160)...)
	errors = append(errors, validateRequiredText("executive_summary", report.ExecutiveSummary, 2000)...)
	if len(report.Sections) == 0 {
		errors = append(errors, "sections is required")
	}
	for index, section := range report.Sections {
		if !isStructuredReportSectionType(section.Type) {
			errors = append(errors, fmt.Sprintf("sections[%d].type is not supported", index))
		}
		errors = append(errors, prefixValidationErrors(
			fmt.Sprintf("sections[%d]", index),
			validateRequiredText("title", section.Title, 80),
		)...)
		if isBlank(section.Content) && len(section.Items) == 0 {
			errors = append(errors, fmt.Sprintf("sections[%d] must contain content or items", index))
		}
		if runeCount(section.Content) > 8000 {
			errors = append(errors, fmt.Sprintf("sections[%d].content exceeds 8000 characters", index))
		}
		for itemIndex, item := range section.Items {
			if isBlank(item) {
				errors = append(errors, fmt.Sprintf("sections[%d].items[%d] is required", index, itemIndex))
			}
			if runeCount(item) > 1000 {
				errors = append(errors, fmt.Sprintf("sections[%d].items[%d] exceeds 1000 characters", index, itemIndex))
			}
		}
	}
	for index, evidenceRef := range report.EvidenceRefs {
		if isBlank(evidenceRef) {
			errors = append(errors, fmt.Sprintf("evidence_refs[%d] is required", index))
		}
	}
	return errors
}

func isSupportedAgentArtifactKind(kind string) bool {
	switch kind {
	case AgentArtifactKindText, AgentArtifactKindReport, AgentArtifactKindHTML, AgentArtifactKindImage,
		AgentArtifactKindPDF, AgentArtifactKindDocument, AgentArtifactKindSpreadsheet,
		AgentArtifactKindPresentation, AgentArtifactKindAudio, AgentArtifactKindVideo, AgentArtifactKindData:
		return true
	default:
		return false
	}
}

func isStructuredReportSectionType(sectionType string) bool {
	switch sectionType {
	case "facts", "analysis", "risks", "missing_information", "recommended_actions", "talk_track", "evidence":
		return true
	default:
		return false
	}
}

func validateRequiredText(name, value string, maxLength int) []string {
	if isBlank(value) {
		return []string{name + " is required"}
	}
	if runeCount(value) > maxLength {
		return []string{fmt.Sprintf("%s exceeds %d characters", name, maxLength)}
	}
	return nil
}

func prefixValidationErrors(prefix string, errors []string) []string {
	for index, validationError := range errors {
		errors[index] = prefix + "." + validationError
	}
	return errors
}

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func runeCount(value string) int {
	return utf8.RuneCountInString(value)
}

func toJSONMap(value any) (JSONMap, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result JSONMap
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, err
	}
	return result, nil
}
