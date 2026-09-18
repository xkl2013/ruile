package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"gopkg.in/yaml.v3"
)

var (
	ErrExpertPackageNotFound        = errors.New("expert package not found")
	ErrExpertPackageVersionNotFound = errors.New("expert package version not found")
	ErrExpertPackageVersionExists   = errors.New("expert package version already exists")
	ErrExpertPackageInvalidInput    = errors.New("invalid expert package input")
	ErrExpertPackageNotPublished    = errors.New("agent definition version is not published")
)

const (
	expertPackageMaxPromptRunes        = 24000
	expertPackageMaxSkillSnapshotRunes = 120000
)

type expertPackageService struct {
	repo        interfaces.ExpertPackageRepository
	fileService interfaces.FileService
}

func NewExpertPackageService(repo interfaces.ExpertPackageRepository, fileService interfaces.FileService) interfaces.ExpertPackageService {
	return &expertPackageService{repo: repo, fileService: fileService}
}

func (s *expertPackageService) ImportPackage(
	ctx context.Context,
	tenantID uint64,
	actorID string,
	input types.ExpertPackageImportInput,
) (*types.ExpertPackageVersion, error) {
	if tenantID == 0 {
		return nil, ErrExpertPackageInvalidInput
	}
	input = normalizeExpertPackageInput(input)
	if !types.IsValidExpertPackageSource(input.SourceFormat) || input.PackageKey == "" || input.Version == "" ||
		input.DisplayName == "" || len(input.Files) == 0 {
		return nil, ErrExpertPackageInvalidInput
	}
	compiled, diagnostics, err := compileExpertPackage(input)
	if err != nil {
		return nil, err
	}
	manifest, err := expertPackageJSONMap(map[string]any{
		"source_format": input.SourceFormat,
		"package_key":   input.PackageKey,
		"version":       input.Version,
		"files":         input.Files,
	})
	if err != nil {
		return nil, err
	}
	return s.importCompiledPackage(
		ctx,
		tenantID,
		actorID,
		input,
		compiled,
		manifest,
		diagnostics,
		expertPackageHash(input.Files),
	)
}

func (s *expertPackageService) importCompiledPackage(
	ctx context.Context,
	tenantID uint64,
	actorID string,
	input types.ExpertPackageImportInput,
	compiled []compiledExpertAgent,
	manifest types.JSONMap,
	diagnostics map[string]any,
	packageHash string,
) (*types.ExpertPackageVersion, error) {
	existing, err := s.resolveExistingPackageVersion(ctx, tenantID, input.PackageKey, input.Version, packageHash)
	if err != nil || existing != nil {
		return existing, err
	}
	diagnosticMap, err := expertPackageJSONMap(diagnostics)
	if err != nil {
		return nil, err
	}
	pkg := &types.ExpertPackage{
		TenantID: tenantID, PackageKey: input.PackageKey, DisplayName: input.DisplayName,
		Description: input.Description, SourceFormat: input.SourceFormat, SourceURI: input.SourceURI,
		License: input.License, CreatedBy: actorID,
	}
	version := &types.ExpertPackageVersion{
		Version: input.Version, Manifest: manifest, PackageHash: packageHash,
		Diagnostics: diagnosticMap, CreatedBy: actorID,
	}
	definitions := make([]*types.AgentDefinitionVersion, 0, len(compiled))
	for _, item := range compiled {
		definitions = append(definitions, &types.AgentDefinitionVersion{
			TenantID: tenantID, AgentID: item.AgentID, Version: item.Version, DisplayName: item.DisplayName,
			Description: item.Description, Domain: item.Domain, SystemPrompt: item.SystemPrompt,
			CompiledConfig: item.CompiledConfig, Skills: item.Skills, Capabilities: item.Capabilities,
			OutputContract: item.OutputContract, DefinitionHash: item.DefinitionHash,
		})
	}
	if err := s.repo.Import(ctx, pkg, version, definitions); err != nil {
		existing, lookupErr := s.resolveExistingPackageVersion(ctx, tenantID, input.PackageKey, input.Version, packageHash)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existing != nil {
			return existing, nil
		}
		return nil, err
	}
	version.Definitions = definitions
	return version, nil
}

func (s *expertPackageService) resolveExistingPackageVersion(
	ctx context.Context,
	tenantID uint64,
	packageKey, version, packageHash string,
) (*types.ExpertPackageVersion, error) {
	existing, err := s.repo.GetVersionByPackageKey(ctx, tenantID, packageKey, version)
	if err != nil || existing == nil {
		return existing, err
	}
	if existing.PackageHash == packageHash {
		return existing, nil
	}
	return nil, fmt.Errorf(
		"%w: package %q version %q already exists with different content; update plugin.json version before re-importing",
		ErrExpertPackageVersionExists,
		packageKey,
		version,
	)
}

func (s *expertPackageService) ListPackages(ctx context.Context, tenantID uint64) ([]*types.ExpertPackage, error) {
	if tenantID == 0 {
		return nil, ErrExpertPackageInvalidInput
	}
	return s.repo.ListPackages(ctx, tenantID)
}

func (s *expertPackageService) ListPublishedExperts(ctx context.Context, tenantID uint64) ([]*types.PublishedExpert, error) {
	if tenantID == 0 {
		return nil, ErrExpertPackageInvalidInput
	}
	return s.repo.ListPublishedExperts(ctx, tenantID)
}

func (s *expertPackageService) GetPackage(ctx context.Context, tenantID uint64, id string) (*types.ExpertPackage, error) {
	pkg, err := s.repo.GetPackage(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, ErrExpertPackageNotFound
	}
	return pkg, nil
}

func (s *expertPackageService) PublishVersion(
	ctx context.Context,
	tenantID uint64,
	actorID, packageID, versionID string,
) error {
	version, err := s.repo.GetVersion(ctx, tenantID, packageID, versionID)
	if err != nil {
		return err
	}
	if version == nil {
		return ErrExpertPackageVersionNotFound
	}
	if len(version.Definitions) == 0 {
		return ErrExpertPackageInvalidInput
	}
	if expertPackageHasBlockingDiagnostics(version.Diagnostics) {
		return ErrExpertPackageInvalidInput
	}
	return s.repo.PublishVersion(ctx, tenantID, packageID, versionID, actorID)
}

func (s *expertPackageService) BindAgent(
	ctx context.Context,
	tenantID uint64,
	actorID, packageID string,
	input types.AgentBindingInput,
) (*types.AgentBinding, error) {
	if strings.TrimSpace(input.ProfileID) == "" || strings.TrimSpace(input.AgentDefinitionVersionID) == "" {
		return nil, ErrExpertPackageInvalidInput
	}
	pkg, err := s.GetPackage(ctx, tenantID, packageID)
	if err != nil {
		return nil, err
	}
	var definition *types.AgentDefinitionVersion
	var state string
	for _, version := range pkg.Versions {
		for _, item := range version.Definitions {
			if item.ID == input.AgentDefinitionVersionID {
				definition, state = item, version.State
				break
			}
		}
	}
	if definition == nil || state != types.ExpertPackageVersionPublished {
		return nil, ErrExpertPackageNotPublished
	}
	binding := &types.AgentBinding{
		TenantID: tenantID, ProfileID: strings.TrimSpace(input.ProfileID),
		AgentDefinitionVersionID: definition.ID, AgentDomain: firstNonEmpty(strings.TrimSpace(input.AgentDomain), definition.Domain),
		Enabled: input.Enabled, CreatedBy: actorID,
	}
	if err := s.repo.UpsertBinding(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *expertPackageService) ListBindings(ctx context.Context, tenantID uint64, profileID string) ([]*types.AgentBinding, error) {
	return s.repo.ListBindings(ctx, tenantID, profileID)
}

type expertAgentFrontMatter struct {
	Kind                string                `yaml:"kind"`
	SchemaVersion       string                `yaml:"schema_version"`
	ID                  string                `yaml:"id"`
	Version             string                `yaml:"version"`
	DisplayName         string                `yaml:"display_name"`
	Description         string                `yaml:"description"`
	Domain              string                `yaml:"domain"`
	MaxTurns            int                   `yaml:"max_turns"`
	Skills              []string              `yaml:"skills"`
	OutputContract      string                `yaml:"output_contract"`
	RequiredInputs      []expertRequiredInput `yaml:"required_inputs"`
	ExecutionPolicy     map[string]any        `yaml:"execution_policy"`
	ClarificationPolicy map[string]any        `yaml:"clarification_policy"`
	DeliverableSpec     map[string]any        `yaml:"deliverable_spec"`
	QualityRubric       map[string]any        `yaml:"quality_rubric"`
	LearningPolicy      map[string]any        `yaml:"learning_policy"`
	Capabilities        struct {
		Required []string `yaml:"required"`
		Optional []string `yaml:"optional"`
	} `yaml:"capabilities"`
}

type expertRequiredInput struct {
	ID             string   `json:"id" yaml:"id"`
	Label          string   `json:"label" yaml:"label"`
	Type           string   `json:"type" yaml:"type"`
	Required       bool     `json:"required" yaml:"required"`
	AskWhenMissing bool     `json:"ask_when_missing" yaml:"ask_when_missing"`
	Options        []string `json:"options,omitempty" yaml:"options"`
	Description    string   `json:"description,omitempty" yaml:"description"`
}

type compiledExpertAgent struct {
	AgentID, Version, DisplayName, Description, Domain, SystemPrompt, OutputContract, DefinitionHash string
	Skills                                                                                           types.StringArray
	CompiledConfig, Capabilities                                                                     types.JSONMap
}

func compileExpertPackage(input types.ExpertPackageImportInput) ([]compiledExpertAgent, map[string]any, error) {
	files := map[string]string{}
	for _, file := range input.Files {
		cleanPath, err := safeExpertPackagePath(file.Path)
		if err != nil {
			return nil, nil, err
		}
		if _, duplicate := files[cleanPath]; duplicate {
			return nil, nil, fmt.Errorf("%w: duplicate file path %q", ErrExpertPackageInvalidInput, cleanPath)
		}
		files[cleanPath] = file.Content
	}
	agents := make([]compiledExpertAgent, 0)
	blocking := make([]string, 0)
	warnings := make([]string, 0)
	for filePath, content := range files {
		if !strings.HasPrefix(filePath, "agents/") || !strings.HasSuffix(strings.ToLower(filePath), ".md") {
			continue
		}
		frontMatter, body, err := parseExpertAgentMarkdown(content)
		if err != nil {
			return nil, nil, fmt.Errorf("%w: %s: %v", ErrExpertPackageInvalidInput, filePath, err)
		}
		if frontMatter.ID == "" || frontMatter.Version == "" || frontMatter.DisplayName == "" || strings.TrimSpace(body) == "" {
			return nil, nil, fmt.Errorf("%w: %s requires id, version, display_name and body", ErrExpertPackageInvalidInput, filePath)
		}
		if utf8.RuneCountInString(body) > expertPackageMaxPromptRunes {
			return nil, nil, fmt.Errorf("%w: %s exceeds prompt size limit", ErrExpertPackageInvalidInput, filePath)
		}
		skills := cleanExpertStrings(frontMatter.Skills)
		for _, skill := range skills {
			if _, ok := files[path.Join("skills", skill, "SKILL.md")]; !ok {
				blocking = append(blocking, "missing instruction skill: "+skill)
			}
		}
		required, unsupportedRequired := resolveExpertCapabilities(frontMatter.Capabilities.Required)
		optional, unsupportedOptional := resolveExpertCapabilities(frontMatter.Capabilities.Optional)
		blocking = append(blocking, unsupportedRequired...)
		warnings = append(warnings, unsupportedOptional...)
		config, err := expertPackageJSONMap(types.CustomAgentConfig{
			AgentMode: types.AgentModeSmartReasoning, SystemPrompt: body,
			MaxIterations: clampExpertTurns(frontMatter.MaxTurns), SkillsSelectionMode: "selected",
			SelectedSkills: skills, AllowedTools: expertAllowedTools(append(required, optional...)),
		})
		if err != nil {
			return nil, nil, err
		}
		config["schema_version"] = firstNonEmpty(frontMatter.SchemaVersion, "1.0")
		config["required_inputs"] = frontMatter.RequiredInputs
		config["execution_policy"] = cleanExpertMap(frontMatter.ExecutionPolicy)
		config["clarification_policy"] = cleanExpertMap(frontMatter.ClarificationPolicy)
		config["deliverable_spec"] = cleanExpertMap(frontMatter.DeliverableSpec)
		config["quality_rubric"] = cleanExpertMap(frontMatter.QualityRubric)
		config["learning_policy"] = cleanExpertMap(frontMatter.LearningPolicy)
		config["skill_snapshots"] = expertSkillSnapshotsFromText(files, skills)
		capabilities, err := expertPackageJSONMap(map[string]any{
			"required": required, "optional": optional,
		})
		if err != nil {
			return nil, nil, err
		}
		definitionHash := expertStringHash(filePath + "\n" + content)
		agents = append(agents, compiledExpertAgent{
			AgentID: frontMatter.ID, Version: frontMatter.Version, DisplayName: frontMatter.DisplayName,
			Description: frontMatter.Description, Domain: frontMatter.Domain, SystemPrompt: body,
			OutputContract: firstNonEmpty(frontMatter.OutputContract, types.AgentResultSchemaV1),
			Skills:         types.StringArray(skills), CompiledConfig: config, Capabilities: capabilities, DefinitionHash: definitionHash,
		})
	}
	if len(agents) == 0 {
		return nil, nil, fmt.Errorf("%w: no agents/*.md files found", ErrExpertPackageInvalidInput)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].AgentID < agents[j].AgentID })
	return agents, map[string]any{
		"blocking": cleanExpertStrings(blocking),
		"warnings": cleanExpertStrings(warnings),
	}, nil
}

func cleanExpertMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func expertSkillSnapshotsFromText(files map[string]string, skills []string) []map[string]any {
	snapshots := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		skillPath := path.Join("skills", skill, "SKILL.md")
		content, ok := files[skillPath]
		if !ok {
			continue
		}
		if utf8.RuneCountInString(content) > expertPackageMaxSkillSnapshotRunes {
			content = string([]rune(content)[:expertPackageMaxSkillSnapshotRunes])
		}
		snapshots = append(snapshots, map[string]any{
			"name":    skill,
			"path":    skillPath,
			"content": content,
			"sha256":  expertStringHash(content),
		})
	}
	return snapshots
}

func parseExpertAgentMarkdown(content string) (expertAgentFrontMatter, string, error) {
	frontMatterRaw, body, err := splitExpertMarkdownFrontMatter(content)
	if err != nil {
		return expertAgentFrontMatter{}, "", err
	}
	var frontMatter expertAgentFrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterRaw), &frontMatter); err != nil {
		return expertAgentFrontMatter{}, "", err
	}
	return frontMatter, body, nil
}

func splitExpertMarkdownFrontMatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", errors.New("front matter is required")
	}
	end := strings.Index(content[4:], "\n---\n")
	if end < 0 {
		return "", "", errors.New("front matter is not closed")
	}
	frontMatterRaw := content[4 : end+4]
	body := strings.TrimSpace(content[end+9:])
	return frontMatterRaw, body, nil
}

func normalizeExpertPackageInput(input types.ExpertPackageImportInput) types.ExpertPackageImportInput {
	input.SourceFormat = strings.TrimSpace(strings.ToLower(input.SourceFormat))
	input.PackageKey = strings.TrimSpace(input.PackageKey)
	input.Version = strings.TrimSpace(input.Version)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Description = strings.TrimSpace(input.Description)
	input.SourceURI = strings.TrimSpace(input.SourceURI)
	input.License = strings.TrimSpace(input.License)
	return input
}

func safeExpertPackagePath(value string) (string, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	cleaned := path.Clean(value)
	if value == "" || strings.HasPrefix(cleaned, "../") || cleaned == "." || strings.HasPrefix(cleaned, "/") {
		return "", fmt.Errorf("%w: unsafe package path", ErrExpertPackageInvalidInput)
	}
	return cleaned, nil
}

func resolveExpertCapabilities(values []string) ([]string, []string) {
	resolved, unsupported := make([]string, 0), make([]string, 0)
	for _, value := range cleanExpertStrings(values) {
		switch value {
		case "knowledge.read", "web.search", "web.fetch", "artifact.publish", "report.render_html":
			resolved = append(resolved, value)
		default:
			unsupported = append(unsupported, "unsupported capability: "+value)
		}
	}
	return resolved, unsupported
}

func expertAllowedTools(capabilities []string) []string {
	tools := make([]string, 0)
	for _, capability := range capabilities {
		switch capability {
		case "knowledge.read":
			tools = append(tools, "knowledge_search")
		case "web.search":
			tools = append(tools, "web_search")
		case "web.fetch":
			tools = append(tools, "web_fetch")
		}
	}
	return cleanExpertStrings(tools)
}

func clampExpertTurns(value int) int {
	if value <= 0 {
		return 12
	}
	if value > 20 {
		return 20
	}
	return value
}

func cleanExpertStrings(values []string) []string {
	result, seen := make([]string, 0, len(values)), map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func expertPackageHash(files []types.ExpertPackageFileInput) string {
	ordered := append([]types.ExpertPackageFileInput(nil), files...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].Path < ordered[j].Path })
	var builder strings.Builder
	for _, file := range ordered {
		builder.WriteString(file.Path)
		builder.WriteByte(0)
		builder.WriteString(file.Content)
		builder.WriteByte(0)
	}
	return expertStringHash(builder.String())
}

func expertStringHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func expertPackageJSONMap(value any) (types.JSONMap, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result types.JSONMap
	return result, json.Unmarshal(raw, &result)
}

func expertPackageHasBlockingDiagnostics(diagnostics types.JSONMap) bool {
	raw, ok := diagnostics["blocking"]
	if !ok {
		return false
	}
	switch values := raw.(type) {
	case []string:
		return len(cleanExpertStrings(values)) > 0
	case types.StringArray:
		return len(cleanExpertStrings(values)) > 0
	case []any:
		items := make([]string, 0, len(values))
		for _, value := range values {
			if item, ok := value.(string); ok {
				items = append(items, item)
			}
		}
		return len(cleanExpertStrings(items)) > 0
	default:
		return false
	}
}
