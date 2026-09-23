package service

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"os"
	"path"
	"sort"
	"strings"
	"unicode"

	"github.com/Tencent/WeKnora/internal/types"
	"gopkg.in/yaml.v3"
)

const (
	ExpertPackageMaxArchiveBytes         int64 = 50 * 1024 * 1024
	expertPackageMaxUncompressedBytes          = 200 * 1024 * 1024
	expertPackageMaxArchiveFiles               = 2000
	expertPackageMaxManifestContentBytes int64 = 1024 * 1024
	expertPackageMaxAvatarBytes          int64 = 4 * 1024 * 1024
)

type expertArchiveImport struct {
	input       types.ExpertPackageImportInput
	compiled    []compiledExpertAgent
	manifest    types.JSONMap
	diagnostics map[string]any
	packageHash string
	avatarPath  string
	avatarData  []byte
}

type expertArchiveFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Kind   string `json:"kind"`
}

type expertLocalizedText struct {
	EN string `json:"en" yaml:"en"`
	ZH string `json:"zh" yaml:"zh"`
}

type workBuddyPluginManifest struct {
	Name               string                `json:"name"`
	Version            string                `json:"version"`
	Description        string                `json:"description"`
	Author             workBuddyAuthor       `json:"author"`
	Agents             []string              `json:"agents"`
	Skills             []string              `json:"skills"`
	ExpertType         string                `json:"expertType"`
	AgentName          string                `json:"agentName"`
	DisplayName        expertLocalizedText   `json:"displayName"`
	Profession         expertLocalizedText   `json:"profession"`
	DisplayDescription expertLocalizedText   `json:"displayDescription"`
	Avatar             string                `json:"avatar"`
	CategoryID         string                `json:"categoryId"`
	DefaultInitPrompt  expertLocalizedText   `json:"defaultInitPrompt"`
	Plugin             string                `json:"plugin"`
	Tags               []expertLocalizedText `json:"tags"`
	QuickPrompts       []expertLocalizedText `json:"quickPrompts"`
	License            json.RawMessage       `json:"license"`
}

type workBuddyAuthor struct {
	Name string `json:"name"`
}

type workBuddyAgentFrontMatter struct {
	Name                string                `yaml:"name"`
	Description         string                `yaml:"description"`
	DisplayName         expertLocalizedText   `yaml:"displayName"`
	Profession          expertLocalizedText   `yaml:"profession"`
	Skills              []string              `yaml:"skills"`
	MaxTurns            int                   `yaml:"maxTurns"`
	RequiredInputs      []expertRequiredInput `yaml:"required_inputs"`
	ExecutionPolicy     map[string]any        `yaml:"execution_policy"`
	ClarificationPolicy map[string]any        `yaml:"clarification_policy"`
	DeliverableSpec     map[string]any        `yaml:"deliverable_spec"`
	QualityRubric       map[string]any        `yaml:"quality_rubric"`
	LearningPolicy      map[string]any        `yaml:"learning_policy"`
}

type indexedExpertArchive struct {
	root         string
	pluginPath   string
	files        map[string]*zip.File
	inventory    []expertArchiveFile
	totalSize    int64
	ignoredFiles int
}

func (s *expertPackageService) ImportPackageArchive(
	ctx context.Context,
	tenantID uint64,
	actorID string,
	file *multipart.FileHeader,
) (*types.ExpertPackageVersion, error) {
	if tenantID == 0 || file == nil || s.fileService == nil {
		return nil, ErrExpertPackageInvalidInput
	}
	if file.Size <= 0 || file.Size > ExpertPackageMaxArchiveBytes {
		return nil, fmt.Errorf("%w: ZIP size must be between 1 byte and 50 MB", ErrExpertPackageInvalidInput)
	}
	if !strings.EqualFold(path.Ext(strings.TrimSpace(file.Filename)), ".zip") {
		return nil, fmt.Errorf("%w: only .zip expert packages are supported", ErrExpertPackageInvalidInput)
	}

	parsed, err := parseWorkBuddyExpertArchive(file)
	if err != nil {
		return nil, err
	}

	existing, err := s.resolveExistingPackageVersion(
		ctx,
		tenantID,
		parsed.input.PackageKey,
		parsed.input.Version,
		parsed.packageHash,
	)
	if err != nil || existing != nil {
		return existing, err
	}

	sourceURI, err := s.fileService.SaveFile(ctx, file, tenantID, "")
	if err != nil {
		return nil, fmt.Errorf("save expert package archive: %w", err)
	}
	parsed.input.SourceURI = sourceURI
	parsed.manifest["source_uri"] = sourceURI
	avatarRef := ""
	if len(parsed.avatarData) > 0 {
		avatarExtension := strings.ToLower(path.Ext(parsed.avatarPath))
		if avatarExtension == "" {
			avatarExtension = ".bin"
		}
		avatarRef, err = s.fileService.SaveBytes(
			ctx,
			parsed.avatarData,
			tenantID,
			"expert-avatar-"+parsed.packageHash[:12]+avatarExtension,
			false,
		)
		if err != nil {
			_ = s.fileService.DeleteFile(ctx, sourceURI)
			return nil, fmt.Errorf("save expert package avatar: %w", err)
		}
		parsed.input.Avatar = avatarRef
		parsed.manifest["avatar_ref"] = avatarRef
	}

	version, err := s.importCompiledPackage(
		ctx,
		tenantID,
		actorID,
		parsed.input,
		parsed.compiled,
		parsed.manifest,
		parsed.diagnostics,
		parsed.packageHash,
	)
	if err != nil {
		_ = s.fileService.DeleteFile(ctx, sourceURI)
		if avatarRef != "" {
			_ = s.fileService.DeleteFile(ctx, avatarRef)
		}
		return nil, err
	}
	return version, nil
}

func parseWorkBuddyExpertArchive(fileHeader *multipart.FileHeader) (*expertArchiveImport, error) {
	source, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to open ZIP", ErrExpertPackageInvalidInput)
	}
	defer source.Close()

	reader, err := zip.NewReader(source, fileHeader.Size)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ZIP archive", ErrExpertPackageInvalidInput)
	}
	indexed, err := indexExpertArchive(reader)
	if err != nil {
		return nil, err
	}

	pluginContent, err := readExpertArchiveFile(indexed.files[indexed.pluginPath], expertPackageMaxManifestContentBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot read .codebuddy-plugin/plugin.json: %v", ErrExpertPackageInvalidInput, err)
	}
	var plugin workBuddyPluginManifest
	if err := json.Unmarshal(pluginContent, &plugin); err != nil {
		return nil, fmt.Errorf("%w: invalid .codebuddy-plugin/plugin.json: %v", ErrExpertPackageInvalidInput, err)
	}
	plugin.Name = strings.TrimSpace(plugin.Name)
	plugin.Version = strings.TrimSpace(plugin.Version)
	if plugin.Name == "" || plugin.Version == "" {
		return nil, fmt.Errorf("%w: plugin.json requires name and version", ErrExpertPackageInvalidInput)
	}
	if plugin.ExpertType != "" && plugin.ExpertType != "agent" {
		return nil, fmt.Errorf("%w: WorkBuddy expert type %q is not supported yet", ErrExpertPackageInvalidInput, plugin.ExpertType)
	}

	compiled, blocking, warnings, err := compileWorkBuddyAgents(plugin, indexed.files)
	if err != nil {
		return nil, err
	}

	counts := map[string]int{}
	for _, item := range indexed.inventory {
		counts[item.Kind]++
	}
	if counts["script"] > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"package contains %d script files; script execution is disabled",
			counts["script"],
		))
	}
	if counts["vendor"] > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"package contains %d vendor files; bundled dependencies are stored but not installed",
			counts["vendor"],
		))
	}
	if counts["binary"] > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"package contains %d binary files; binary execution is disabled",
			counts["binary"],
		))
	}
	if indexed.ignoredFiles > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"ignored %d files outside the detected package root",
			indexed.ignoredFiles,
		))
	}

	avatarPath := cleanArchiveReference(plugin.Avatar)
	var avatarData []byte
	if avatarPath != "" {
		avatarEntry := indexed.files[avatarPath]
		switch {
		case avatarEntry == nil:
			warnings = append(warnings, fmt.Sprintf("avatar file %q was not found", avatarPath))
		case !strings.HasPrefix(mime.TypeByExtension(strings.ToLower(path.Ext(avatarPath))), "image/"):
			warnings = append(warnings, fmt.Sprintf("avatar file %q is not a supported image", avatarPath))
		default:
			avatarData, err = readExpertArchiveFile(avatarEntry, expertPackageMaxAvatarBytes)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("avatar file %q was ignored: %v", avatarPath, err))
				avatarData = nil
			}
		}
	}

	license := workBuddyLicense(plugin.License)
	manifest, err := expertPackageJSONMap(map[string]any{
		"source_format":       types.ExpertPackageSourceWorkBuddy,
		"package_key":         plugin.Name,
		"version":             plugin.Version,
		"archive_root":        indexed.root,
		"file_count":          len(indexed.inventory),
		"uncompressed_bytes":  indexed.totalSize,
		"files":               indexed.inventory,
		"avatar_path":         cleanArchiveReference(plugin.Avatar),
		"category_id":         plugin.CategoryID,
		"author":              plugin.Author.Name,
		"profession":          plugin.Profession,
		"default_init_prompt": plugin.DefaultInitPrompt,
		"tags":                plugin.Tags,
		"quick_prompts":       plugin.QuickPrompts,
		"compatibility": map[string]any{
			"agent_count":  len(compiled),
			"skill_count":  counts["skill"],
			"script_count": counts["script"],
			"vendor_count": counts["vendor"],
			"binary_count": counts["binary"],
		},
	})
	if err != nil {
		return nil, err
	}

	return &expertArchiveImport{
		input: types.ExpertPackageImportInput{
			SourceFormat: types.ExpertPackageSourceWorkBuddy,
			PackageKey:   plugin.Name,
			Version:      plugin.Version,
			DisplayName: firstNonEmpty(
				strings.TrimSpace(plugin.DisplayName.ZH),
				strings.TrimSpace(plugin.DisplayName.EN),
				plugin.Name,
			),
			Description: firstNonEmpty(
				strings.TrimSpace(plugin.DisplayDescription.ZH),
				strings.TrimSpace(plugin.DisplayDescription.EN),
				strings.TrimSpace(plugin.Description),
			),
			License: license,
		},
		compiled: compiled,
		manifest: manifest,
		diagnostics: map[string]any{
			"blocking": cleanExpertStrings(blocking),
			"warnings": cleanExpertStrings(warnings),
			"compatibility": map[string]any{
				"file_count":           len(indexed.inventory),
				"uncompressed_bytes":   indexed.totalSize,
				"agent_count":          len(compiled),
				"instruction_skills":   counts["skill"],
				"scripts_disabled":     counts["script"],
				"vendor_not_installed": counts["vendor"],
				"binaries_disabled":    counts["binary"],
			},
		},
		packageHash: expertArchiveHash(indexed.inventory),
		avatarPath:  avatarPath,
		avatarData:  avatarData,
	}, nil
}

func indexExpertArchive(reader *zip.Reader) (*indexedExpertArchive, error) {
	if len(reader.File) == 0 || len(reader.File) > expertPackageMaxArchiveFiles {
		return nil, fmt.Errorf(
			"%w: ZIP must contain between 1 and %d files",
			ErrExpertPackageInvalidInput,
			expertPackageMaxArchiveFiles,
		)
	}

	type archiveCandidate struct {
		path string
		file *zip.File
	}
	candidates := make([]archiveCandidate, 0, 1)
	allFiles := make([]archiveCandidate, 0, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: symbolic links are not allowed", ErrExpertPackageInvalidInput)
		}
		cleaned, err := safeExpertPackagePath(entry.Name)
		if err != nil {
			return nil, err
		}
		if isIgnoredExpertArchivePath(cleaned) {
			continue
		}
		candidate := archiveCandidate{path: cleaned, file: entry}
		allFiles = append(allFiles, candidate)
		if cleaned == ".codebuddy-plugin/plugin.json" ||
			strings.HasSuffix(cleaned, "/.codebuddy-plugin/plugin.json") {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) != 1 {
		return nil, fmt.Errorf(
			"%w: ZIP must contain exactly one .codebuddy-plugin/plugin.json",
			ErrExpertPackageInvalidInput,
		)
	}

	pluginFullPath := candidates[0].path
	root := strings.TrimSuffix(pluginFullPath, ".codebuddy-plugin/plugin.json")
	root = strings.TrimSuffix(root, "/")
	indexed := &indexedExpertArchive{
		root:      root,
		files:     make(map[string]*zip.File),
		inventory: make([]expertArchiveFile, 0, len(allFiles)),
	}

	var actualTotal int64
	for _, candidate := range allFiles {
		relative, belongs := archiveRelativePath(root, candidate.path)
		if !belongs {
			indexed.ignoredFiles++
			continue
		}
		if _, duplicate := indexed.files[relative]; duplicate {
			return nil, fmt.Errorf("%w: duplicate archive path %q", ErrExpertPackageInvalidInput, relative)
		}
		remaining := int64(expertPackageMaxUncompressedBytes) - actualTotal
		if remaining <= 0 {
			return nil, fmt.Errorf("%w: ZIP expands beyond 200 MB", ErrExpertPackageInvalidInput)
		}
		hash, size, err := hashExpertArchiveFile(candidate.file, remaining)
		if err != nil {
			return nil, err
		}
		actualTotal += size
		indexed.files[relative] = candidate.file
		indexed.inventory = append(indexed.inventory, expertArchiveFile{
			Path:   relative,
			Size:   size,
			SHA256: hash,
			Kind:   expertArchiveFileKind(relative),
		})
	}
	if actualTotal > expertPackageMaxUncompressedBytes {
		return nil, fmt.Errorf("%w: ZIP expands beyond 200 MB", ErrExpertPackageInvalidInput)
	}
	indexed.totalSize = actualTotal
	indexed.pluginPath = ".codebuddy-plugin/plugin.json"
	sort.Slice(indexed.inventory, func(i, j int) bool {
		return indexed.inventory[i].Path < indexed.inventory[j].Path
	})
	return indexed, nil
}

func isIgnoredExpertArchivePath(filePath string) bool {
	base := path.Base(filePath)
	return base == ".DS_Store" ||
		strings.HasPrefix(base, "._") ||
		strings.HasPrefix(filePath, "__MACOSX/") ||
		strings.Contains(filePath, "/__MACOSX/")
}

func compileWorkBuddyAgents(
	plugin workBuddyPluginManifest,
	files map[string]*zip.File,
) ([]compiledExpertAgent, []string, []string, error) {
	agentPaths := append([]string(nil), plugin.Agents...)
	if len(agentPaths) == 0 && strings.TrimSpace(plugin.AgentName) != "" {
		agentPaths = []string{"agents/" + strings.TrimSpace(plugin.AgentName) + ".md"}
	}
	if len(agentPaths) == 0 {
		return nil, nil, nil, fmt.Errorf("%w: plugin.json does not declare any agents", ErrExpertPackageInvalidInput)
	}

	compiled := make([]compiledExpertAgent, 0, len(agentPaths))
	blocking := make([]string, 0)
	warnings := make([]string, 0)
	for _, declaredPath := range agentPaths {
		agentPath := cleanArchiveReference(declaredPath)
		entry := files[agentPath]
		if entry == nil {
			return nil, nil, nil, fmt.Errorf(
				"%w: declared agent file %q was not found",
				ErrExpertPackageInvalidInput,
				agentPath,
			)
		}
		contentBytes, err := readExpertArchiveFile(entry, expertPackageMaxManifestContentBytes)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: cannot read %s: %v", ErrExpertPackageInvalidInput, agentPath, err)
		}
		content := string(contentBytes)
		frontMatterRaw, body, err := splitExpertMarkdownFrontMatter(content)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("%w: %s: %v", ErrExpertPackageInvalidInput, agentPath, err)
		}
		var frontMatter workBuddyAgentFrontMatter
		if err := yaml.Unmarshal([]byte(frontMatterRaw), &frontMatter); err != nil {
			return nil, nil, nil, fmt.Errorf("%w: %s: %v", ErrExpertPackageInvalidInput, agentPath, err)
		}
		frontMatter.Name = strings.TrimSpace(frontMatter.Name)
		if frontMatter.Name == "" || strings.TrimSpace(body) == "" {
			return nil, nil, nil, fmt.Errorf(
				"%w: %s requires name and body",
				ErrExpertPackageInvalidInput,
				agentPath,
			)
		}
		if utf8RuneCount(body) > expertPackageMaxPromptRunes {
			return nil, nil, nil, fmt.Errorf("%w: %s exceeds prompt size limit", ErrExpertPackageInvalidInput, agentPath)
		}

		skills := cleanExpertStrings(frontMatter.Skills)
		for _, skill := range skills {
			skillPath := path.Join("skills", skill, "SKILL.md")
			if files[skillPath] == nil {
				blocking = append(blocking, "missing instruction skill: "+skill)
			}
		}
		if frontMatter.MaxTurns > 20 {
			warnings = append(warnings, fmt.Sprintf(
				"agent %s maxTurns reduced from %d to 20",
				frontMatter.Name,
				frontMatter.MaxTurns,
			))
		}

		config, err := expertPackageJSONMap(types.CustomAgentConfig{
			AgentMode:           types.AgentModeSmartReasoning,
			SystemPrompt:        body,
			MaxIterations:       clampExpertTurns(frontMatter.MaxTurns),
			SkillsSelectionMode: "selected",
			SelectedSkills:      skills,
			AllowedTools:        []string{},
		})
		if err != nil {
			return nil, nil, nil, err
		}
		config["schema_version"] = "1.0"
		config["required_inputs"] = frontMatter.RequiredInputs
		config["execution_policy"] = cleanExpertMap(frontMatter.ExecutionPolicy)
		config["clarification_policy"] = cleanExpertMap(frontMatter.ClarificationPolicy)
		config["deliverable_spec"] = cleanExpertMap(frontMatter.DeliverableSpec)
		config["quality_rubric"] = cleanExpertMap(frontMatter.QualityRubric)
		config["learning_policy"] = cleanExpertMap(frontMatter.LearningPolicy)
		snapshots, err := expertSkillSnapshotsFromArchive(files, skills)
		if err != nil {
			return nil, nil, nil, err
		}
		config["skill_snapshots"] = snapshots
		capabilities, err := expertPackageJSONMap(map[string]any{
			"required": []string{},
			"optional": []string{},
		})
		if err != nil {
			return nil, nil, nil, err
		}
		compiled = append(compiled, compiledExpertAgent{
			AgentID: frontMatter.Name,
			Version: plugin.Version,
			DisplayName: firstNonEmpty(
				strings.TrimSpace(frontMatter.DisplayName.ZH),
				strings.TrimSpace(frontMatter.DisplayName.EN),
				strings.TrimSpace(plugin.DisplayName.ZH),
				strings.TrimSpace(plugin.DisplayName.EN),
				frontMatter.Name,
			),
			Description: firstNonEmpty(
				strings.TrimSpace(frontMatter.Description),
				strings.TrimSpace(plugin.DisplayDescription.ZH),
				strings.TrimSpace(plugin.DisplayDescription.EN),
				strings.TrimSpace(plugin.Description),
			),
			Domain:         expertDomainFromPackageKey(plugin.Name),
			SystemPrompt:   body,
			OutputContract: types.AgentResultSchemaV1,
			Skills:         types.StringArray(skills),
			CompiledConfig: config,
			Capabilities:   capabilities,
			DefinitionHash: expertStringHash(agentPath + "\n" + content),
		})
	}
	sort.Slice(compiled, func(i, j int) bool {
		return compiled[i].AgentID < compiled[j].AgentID
	})
	return compiled, blocking, warnings, nil
}

func expertSkillSnapshotsFromArchive(
	files map[string]*zip.File,
	skills []string,
) ([]map[string]any, error) {
	snapshots := make([]map[string]any, 0, len(skills))
	for _, skill := range skills {
		skillPath := path.Join("skills", skill, "SKILL.md")
		entry := files[skillPath]
		if entry == nil {
			continue
		}
		content, err := readExpertArchiveFile(entry, expertPackageMaxManifestContentBytes)
		if err != nil {
			return nil, fmt.Errorf("%w: cannot read %s: %v", ErrExpertPackageInvalidInput, skillPath, err)
		}
		runes := []rune(string(content))
		if len(runes) > expertPackageMaxSkillSnapshotRunes {
			runes = runes[:expertPackageMaxSkillSnapshotRunes]
		}
		text := string(runes)
		snapshots = append(snapshots, map[string]any{
			"name":    skill,
			"path":    skillPath,
			"content": text,
			"sha256":  expertStringHash(text),
		})
	}
	return snapshots, nil
}

func archiveRelativePath(root, fullPath string) (string, bool) {
	if root == "" {
		return fullPath, true
	}
	prefix := root + "/"
	if !strings.HasPrefix(fullPath, prefix) {
		return "", false
	}
	return strings.TrimPrefix(fullPath, prefix), true
}

func cleanArchiveReference(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "\\", "/"))
	value = strings.TrimPrefix(value, "./")
	cleaned := path.Clean(value)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func hashExpertArchiveFile(entry *zip.File, limit int64) (string, int64, error) {
	reader, err := entry.Open()
	if err != nil {
		return "", 0, fmt.Errorf("%w: cannot read %s", ErrExpertPackageInvalidInput, entry.Name)
	}
	defer reader.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(reader, limit+1))
	if err != nil {
		return "", 0, fmt.Errorf("%w: cannot read %s", ErrExpertPackageInvalidInput, entry.Name)
	}
	if size > limit {
		return "", 0, fmt.Errorf("%w: ZIP expands beyond 200 MB", ErrExpertPackageInvalidInput)
	}
	return hex.EncodeToString(hash.Sum(nil)), size, nil
}

func readExpertArchiveFile(entry *zip.File, limit int64) ([]byte, error) {
	if entry == nil {
		return nil, fmt.Errorf("file not found")
	}
	if entry.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	reader, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}

func expertArchiveFileKind(filePath string) string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasPrefix(lower, "vendor/"):
		return "vendor"
	case strings.Contains(lower, "/scripts/") || strings.HasPrefix(lower, "scripts/") ||
		strings.HasSuffix(lower, ".py") || strings.HasSuffix(lower, ".sh") ||
		strings.HasSuffix(lower, ".ps1") || strings.HasSuffix(lower, ".mjs") ||
		strings.HasSuffix(lower, ".cjs"):
		return "script"
	case strings.HasSuffix(lower, ".exe") || strings.HasSuffix(lower, ".dll") ||
		strings.HasSuffix(lower, ".so") || strings.HasSuffix(lower, ".dylib") ||
		strings.HasSuffix(lower, ".wasm") || strings.HasSuffix(lower, ".bin"):
		return "binary"
	case strings.HasPrefix(lower, "evals/"):
		return "evaluation"
	case strings.HasPrefix(lower, "avatars/"):
		return "avatar"
	case strings.HasPrefix(lower, "agents/") && strings.HasSuffix(lower, ".md"):
		return "agent"
	case strings.HasPrefix(lower, "skills/") && strings.HasSuffix(lower, "/skill.md"):
		return "skill"
	case strings.Contains(lower, "/references/") || strings.HasPrefix(lower, "references/"):
		return "reference"
	case strings.HasPrefix(lower, "license/") || strings.Contains(path.Base(lower), "license"):
		return "license"
	case path.Base(lower) == "readme.md":
		return "readme"
	case lower == ".codebuddy-plugin/plugin.json":
		return "manifest"
	default:
		return "asset"
	}
}

func expertArchiveHash(files []expertArchiveFile) string {
	ordered := append([]expertArchiveFile(nil), files...)
	sort.Slice(ordered, func(i, j int) bool {
		return ordered[i].Path < ordered[j].Path
	})
	var builder strings.Builder
	for _, file := range ordered {
		builder.WriteString(file.Path)
		builder.WriteByte(0)
		builder.WriteString(file.SHA256)
		builder.WriteByte(0)
	}
	return expertStringHash(builder.String())
}

func expertDomainFromPackageKey(value string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, current := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			builder.WriteRune(current)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore && builder.Len() > 0 {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}

func workBuddyLicense(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var text string
	if json.Unmarshal(raw, &text) == nil {
		return strings.TrimSpace(text)
	}
	var value struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(raw, &value) == nil {
		return strings.TrimSpace(value.Name)
	}
	return ""
}

func utf8RuneCount(value string) int {
	return len([]rune(value))
}
