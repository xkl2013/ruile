package service

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
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
	agentRepo   interfaces.CustomAgentRepository
}

func NewExpertPackageService(repo interfaces.ExpertPackageRepository, fileService interfaces.FileService) interfaces.ExpertPackageService {
	return &expertPackageService{repo: repo, fileService: fileService}
}

func NewExpertPackageServiceWithAgentRepository(
	repo interfaces.ExpertPackageRepository,
	fileService interfaces.FileService,
	agentRepo interfaces.CustomAgentRepository,
) interfaces.ExpertPackageService {
	return &expertPackageService{repo: repo, fileService: fileService, agentRepo: agentRepo}
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
		"avatar":        input.Avatar,
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
		Description: input.Description, Avatar: input.Avatar, SourceFormat: input.SourceFormat, SourceURI: input.SourceURI,
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
	packages, err := s.repo.ListPackages(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, pkg := range packages {
		if pkg != nil {
			pkg.Avatar = s.resolveAvatarURL(ctx, pkg.Avatar)
		}
	}
	return packages, nil
}

func (s *expertPackageService) ListPublishedExperts(ctx context.Context, tenantID uint64) ([]*types.PublishedExpert, error) {
	if tenantID == 0 {
		return nil, ErrExpertPackageInvalidInput
	}
	experts, err := s.repo.ListPublishedExperts(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for _, expert := range experts {
		if expert != nil {
			expert.Avatar = s.resolveAvatarURL(ctx, expert.Avatar)
		}
	}
	return experts, nil
}

func (s *expertPackageService) SyncPublishedAgents(ctx context.Context, tenantID uint64, actorID string) error {
	if tenantID == 0 {
		return ErrExpertPackageInvalidInput
	}
	if s.agentRepo == nil {
		return nil
	}
	definitions, err := s.repo.ListPublishedDefinitions(ctx, tenantID)
	if err != nil {
		return err
	}
	s.syncVersionAgents(ctx, tenantID, actorID, definitions)
	return nil
}

func (s *expertPackageService) GetPackage(ctx context.Context, tenantID uint64, id string) (*types.ExpertPackage, error) {
	pkg, err := s.repo.GetPackage(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, ErrExpertPackageNotFound
	}
	pkg.Avatar = s.resolveAvatarURL(ctx, pkg.Avatar)
	return pkg, nil
}

func (s *expertPackageService) resolveAvatarURL(ctx context.Context, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	lowerValue := strings.ToLower(value)
	if strings.HasPrefix(lowerValue, "http://") || strings.HasPrefix(lowerValue, "https://") {
		return value
	}
	protected := false
	for _, prefix := range []string{
		"resource://", "storage://", "local://", "minio://", "cos://", "tos://",
		"s3://", "oss://", "ks3://", "obs://",
	} {
		if strings.HasPrefix(lowerValue, prefix) {
			protected = true
			break
		}
	}
	if !protected {
		return ""
	}
	if s.fileService == nil {
		return value
	}
	resolved, err := s.fileService.GetFileURL(ctx, value)
	if err != nil {
		return value
	}
	resolved = strings.TrimSpace(resolved)
	if strings.HasPrefix(strings.ToLower(resolved), "http://") ||
		strings.HasPrefix(strings.ToLower(resolved), "https://") {
		return resolved
	}
	return value
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
	if err := s.repo.PublishVersion(ctx, tenantID, packageID, versionID, actorID); err != nil {
		return err
	}
	// Published WorkBuddy definitions are also exposed as regular custom
	// Agents so the existing orchestration member selector and delegate_agent
	// runtime can use the same stable IDs.
	s.syncVersionAgents(ctx, tenantID, actorID, version.Definitions)
	return nil
}

func (s *expertPackageService) syncVersionAgents(
	ctx context.Context,
	tenantID uint64,
	actorID string,
	definitions []*types.AgentDefinitionVersion,
) {
	if s.agentRepo == nil {
		return
	}
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		if err := s.syncDefinitionAgent(ctx, tenantID, actorID, definition); err != nil {
			logger.Warnf(ctx, "Failed to sync published expert %q into custom Agent: %v", definition.AgentID, err)
		}
	}
}

func (s *expertPackageService) syncDefinitionAgent(
	ctx context.Context,
	tenantID uint64,
	actorID string,
	definition *types.AgentDefinitionVersion,
) error {
	if err := s.enrichDefinitionSkillSnapshots(ctx, tenantID, definition); err != nil {
		logger.Warnf(ctx, "Failed to hydrate embedded Skill files for expert %q: %v", definition.AgentID, err)
	}

	var config types.CustomAgentConfig
	compiled, err := json.Marshal(definition.CompiledConfig)
	if err != nil {
		return err
	}
	if len(compiled) > 0 && string(compiled) != "null" {
		if err := json.Unmarshal(compiled, &config); err != nil {
			return err
		}
	}
	config.AgentMode = types.AgentModeSmartReasoning
	config.AgentType = types.AgentTypeCustom
	config.SystemPrompt = definition.SystemPrompt
	config.SkillsSelectionMode = "selected"
	config.SelectedSkills = append([]string(nil), definition.Skills...)
	config.OrchestrationEnabled = false
	config.MemberAgentIDs = nil
	// The delegated runtime narrows this to the main Agent's search targets.
	config.KBSelectionMode = "all"

	name := strings.TrimSpace(definition.DisplayName)
	if name == "" {
		name = strings.TrimSpace(definition.AgentID)
	}
	if name == "" {
		return fmt.Errorf("expert definition has no display name or agent id")
	}

	platformManaged := types.IsSystemAdminFromContext(ctx)
	agentTenantID := tenantID
	idTenantID := tenantID
	if platformManaged {
		agentTenantID = types.SystemAgentTenantID
		idTenantID = types.SystemAgentTenantID
	}

	// Keep the generated Agent ID stable across package version updates while
	// avoiding a collision with ordinary UUID-based custom Agents.
	agentID := uuid.NewSHA1(
		uuid.Nil,
		[]byte(fmt.Sprintf("ruile:published-expert:%d:%s:%s", idTenantID, definition.PackageID, definition.AgentID)),
	).String()
	now := time.Now().UTC()
	agent := &types.CustomAgent{
		ID:          agentID,
		Name:        name,
		Description: strings.TrimSpace(definition.Description),
		Config:      config,
		IsBuiltin:   platformManaged,
		TenantID:    agentTenantID,
		CreatedBy:   actorID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	agent.EnsureDefaults()

	// Before platform-scoped sync was introduced, the same expert was
	// materialized under the selected workspace using a tenant-derived ID.
	// Promote that row in place when it exists so an already published expert
	// does not leave a second, hidden workspace copy behind.
	var legacy *types.CustomAgent
	if platformManaged && tenantID != types.SystemAgentTenantID {
		legacyID := uuid.NewSHA1(
			uuid.Nil,
			[]byte(fmt.Sprintf("ruile:published-expert:%d:%s:%s", tenantID, definition.PackageID, definition.AgentID)),
		).String()
		if legacyID != agentID {
			legacy, err = s.agentRepo.GetAgentByID(ctx, legacyID, tenantID)
			if err != nil && !errors.Is(err, repository.ErrCustomAgentNotFound) {
				return err
			}
		}
	}

	existing, err := s.agentRepo.GetAgentByID(ctx, agentID, agentTenantID)
	if err != nil && !errors.Is(err, repository.ErrCustomAgentNotFound) {
		return err
	}
	if existing == nil {
		if legacy != nil {
			agent.ID = legacy.ID
		}
		if err := s.agentRepo.CreateAgent(ctx, agent); err != nil {
			return err
		}
		if legacy != nil {
			return s.agentRepo.DeleteAgent(ctx, legacy.ID, legacy.TenantID)
		}
		return nil
	}

	existing.Name = agent.Name
	existing.Description = agent.Description
	existing.Avatar = agent.Avatar
	existing.Config = agent.Config
	existing.IsBuiltin = platformManaged
	existing.TenantID = agentTenantID
	existing.UpdatedAt = now
	if err := s.agentRepo.UpdateAgent(ctx, existing); err != nil {
		return err
	}
	if legacy != nil {
		return s.agentRepo.DeleteAgent(ctx, legacy.ID, legacy.TenantID)
	}
	return nil
}

func (s *expertPackageService) enrichDefinitionSkillSnapshots(
	ctx context.Context,
	tenantID uint64,
	definition *types.AgentDefinitionVersion,
) error {
	if s.fileService == nil || definition == nil || len(definition.Skills) == 0 {
		return nil
	}
	pkg, err := s.repo.GetPackage(ctx, tenantID, definition.PackageID)
	if err != nil {
		return err
	}
	if pkg == nil || strings.TrimSpace(pkg.SourceURI) == "" {
		return nil
	}
	reader, err := s.fileService.GetFile(ctx, pkg.SourceURI)
	if err != nil {
		return err
	}
	defer reader.Close()

	archiveRoot := ""
	if version, versionErr := s.repo.GetVersion(ctx, tenantID, definition.PackageID, definition.PackageVersionID); versionErr != nil {
		return versionErr
	} else if version != nil {
		if value, ok := version.Manifest["archive_root"].(string); ok {
			archiveRoot = strings.Trim(strings.TrimSpace(value), "/")
		}
	}

	data, err := io.ReadAll(io.LimitReader(reader, ExpertPackageMaxArchiveBytes+1))
	if err != nil {
		return err
	}
	if int64(len(data)) > ExpertPackageMaxArchiveBytes {
		return fmt.Errorf("expert package archive exceeds %d bytes", ExpertPackageMaxArchiveBytes)
	}
	archiveReader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	files := make(map[string]*zip.File, len(archiveReader.File))
	for _, entry := range archiveReader.File {
		cleanPath, pathErr := safeExpertPackagePath(entry.Name)
		if pathErr != nil {
			continue
		}
		relativePath, ok := expertArchiveSkillRelativePath(cleanPath, archiveRoot)
		if ok {
			files[relativePath] = entry
		}
	}
	snapshots, err := expertSkillSnapshotsFromArchive(files, []string(definition.Skills))
	if err != nil {
		return err
	}
	if len(snapshots) == 0 {
		return nil
	}
	if definition.CompiledConfig == nil {
		definition.CompiledConfig = types.JSONMap{}
	}
	definition.CompiledConfig["skill_snapshots"] = snapshots
	return nil
}

func expertArchiveSkillRelativePath(filePath, archiveRoot string) (string, bool) {
	filePath = path.Clean(strings.TrimSpace(filePath))
	if filePath == "." || filePath == "" {
		return "", false
	}
	if archiveRoot != "" {
		if relative, ok := archiveRelativePath(archiveRoot, filePath); ok {
			filePath = relative
		}
	}
	if strings.HasPrefix(filePath, "skills/") {
		return filePath, true
	}
	if index := strings.Index(filePath, "/skills/"); index >= 0 {
		return strings.TrimPrefix(filePath[index+1:], "/"), true
	}
	return "", false
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
	allowed := make(map[string]struct{}, len(skills))
	for _, skill := range skills {
		allowed[strings.TrimSpace(skill)] = struct{}{}
	}
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)

	snapshots := make([]map[string]any, 0, len(paths))
	for _, filePath := range paths {
		cleanPath := path.Clean(filePath)
		if !strings.HasPrefix(cleanPath, "skills/") {
			continue
		}
		parts := strings.Split(cleanPath, "/")
		if len(parts) < 3 {
			continue
		}
		skill := parts[1]
		if _, ok := allowed[skill]; !ok {
			continue
		}
		content := files[filePath]
		if utf8.RuneCountInString(content) > expertPackageMaxSkillSnapshotRunes {
			content = string([]rune(content)[:expertPackageMaxSkillSnapshotRunes])
		}
		snapshots = append(snapshots, map[string]any{
			"name":    skill,
			"path":    cleanPath,
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
	input.Avatar = strings.TrimSpace(input.Avatar)
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
