package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

var (
	ErrOrganizeAdminTemplateKeyRequired     = errors.New("template key is required")
	ErrOrganizeAdminTemplateKeyInvalid      = errors.New("template key must contain only lowercase letters, numbers, hyphens, or underscores")
	ErrOrganizeAdminTemplateKeyReserved     = errors.New("template key is reserved for non-organize processing")
	ErrOrganizeAdminTemplateNameRequired    = errors.New("template name is required")
	ErrOrganizeAdminTemplateVersionNeeded   = errors.New("template version is required")
	ErrOrganizeAdminTemplateInvalidSpec     = errors.New("template spec is invalid")
	ErrOrganizeAdminTemplateInvalidMarkdown = errors.New("template markdown preset is invalid")
	ErrOrganizeAdminTemplateNotFound        = errors.New("organize template not found")
	ErrOrganizeAdminTemplateKeyImmutable    = errors.New("template key cannot be changed")
	ErrOrganizeAdminTemplateNotPublishable  = errors.New("organize template is not publishable")
)

var (
	organizeAdminTemplateKeyPattern        = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	organizeAdminPreviewPlaceholderPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)
)

func (s *organizeService) ListAdminTemplates(
	ctx context.Context,
	query types.OrganizeTemplateAdminQuery,
) ([]*types.OrganizeTemplate, int64, error) {
	query.Keyword = strings.TrimSpace(query.Keyword)
	query.Scene = strings.TrimSpace(query.Scene)
	query.Status = strings.TrimSpace(query.Status)
	query.Page, query.PageSize = normalizeOrganizePage(query.Page, query.PageSize)
	templates, total, err := s.repo.ListAdminTemplates(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	for _, template := range templates {
		hydrateOrganizeTemplateMarkdown(template)
	}
	return templates, total, nil
}

func (s *organizeService) ListAdminTemplateScenes(ctx context.Context) ([]string, error) {
	return s.repo.ListAdminTemplateScenes(ctx)
}

func (s *organizeService) GetAdminTemplate(
	ctx context.Context,
	key string,
) (*types.OrganizeTemplate, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, ErrOrganizeAdminTemplateKeyRequired
	}
	if types.IsOrganizeInternalTemplateKey(key) {
		return nil, ErrOrganizeAdminTemplateNotFound
	}
	template, err := s.repo.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, ErrOrganizeAdminTemplateNotFound
	}
	return hydrateOrganizeTemplateMarkdown(template), nil
}

func (s *organizeService) CreateAdminTemplate(
	ctx context.Context,
	actorID string,
	input types.OrganizeTemplateAdminInput,
) (*types.OrganizeTemplate, error) {
	template, err := normalizeAdminTemplateInput(input)
	if err != nil {
		return nil, err
	}
	template.TenantID = 0
	template.OwnerUserID = ""
	template.Scope = types.OrganizeTemplateScopePlatform
	template.Status = types.OrganizeTemplateStatusDraft
	template.ValidationResult = types.JSONMap{}
	if err := s.repo.CreateTemplate(ctx, template); err != nil {
		return nil, err
	}
	if strings.TrimSpace(input.ChangeNote) != "" {
		template.ValidationResult = types.JSONMap{
			"valid":       true,
			"change_note": strings.TrimSpace(input.ChangeNote),
			"created_by":  strings.TrimSpace(actorID),
		}
		_ = s.repo.UpdateTemplate(ctx, template)
	}
	return s.GetAdminTemplate(ctx, template.Key)
}

func (s *organizeService) UpdateAdminTemplate(
	ctx context.Context,
	key, actorID string,
	input types.OrganizeTemplateAdminInput,
) (*types.OrganizeTemplate, error) {
	current, err := s.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	if normalizedKey := strings.TrimSpace(input.Key); normalizedKey != "" && normalizedKey != current.Key {
		return nil, ErrOrganizeAdminTemplateKeyImmutable
	}
	if strings.TrimSpace(input.Key) == "" {
		input.Key = current.Key
	}
	updated, err := normalizeAdminTemplateInput(input)
	if err != nil {
		return nil, err
	}
	updated.ID = current.ID
	updated.Key = current.Key
	updated.TenantID = current.TenantID
	updated.OwnerUserID = current.OwnerUserID
	updated.Scope = current.Scope
	updated.Status = current.Status
	updated.PublishedVersion = current.PublishedVersion
	updated.PublishedAt = current.PublishedAt
	updated.PublishedBy = current.PublishedBy
	updated.ValidationResult = current.ValidationResult
	updated.CreatedAt = current.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateTemplate(ctx, updated); err != nil {
		return nil, err
	}
	return s.GetAdminTemplate(ctx, current.Key)
}

func (s *organizeService) PublishAdminTemplate(
	ctx context.Context,
	key, actorID, changeNote string,
) (*types.OrganizeTemplate, error) {
	template, err := s.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	if errorsList := validateOrganizeTemplateSpec(template.Spec); len(errorsList) > 0 {
		template.ValidationResult = types.JSONMap{
			"valid":  false,
			"errors": errorsList,
		}
		_ = s.repo.UpdateTemplate(ctx, template)
		return nil, fmt.Errorf("%w: %s", ErrOrganizeAdminTemplateNotPublishable, strings.Join(errorsList, "; "))
	}
	if _, errorsList := normalizeOrganizeMarkdownTemplate(template.MarkdownTemplate, template.Key, template.Spec); len(errorsList) > 0 {
		template.ValidationResult = types.JSONMap{
			"valid":  false,
			"errors": errorsList,
		}
		_ = s.repo.UpdateTemplate(ctx, template)
		return nil, fmt.Errorf("%w: %s", ErrOrganizeAdminTemplateNotPublishable, strings.Join(errorsList, "; "))
	}

	versions, _, err := s.repo.ListTemplateVersions(ctx, types.OrganizeTemplateVersionQuery{
		TemplateKey: template.Key,
		Page:        1,
		PageSize:    1000,
	})
	if err != nil {
		return nil, err
	}
	version := nextOrganizeTemplateVersion(versions)
	now := time.Now().UTC()
	snapshot := organizeTemplateSnapshot(template)
	snapshot["version"] = version
	snapshot["change_note"] = strings.TrimSpace(changeNote)
	snapshot["published_by"] = strings.TrimSpace(actorID)
	templateVersion := &types.OrganizeTemplateVersion{
		TemplateID:  template.ID,
		TemplateKey: template.Key,
		Version:     version,
		Snapshot:    snapshot,
		CreatedBy:   strings.TrimSpace(actorID),
		ChangeNote:  strings.TrimSpace(changeNote),
	}
	if err := s.repo.CreateTemplateVersion(ctx, templateVersion); err != nil {
		return nil, err
	}
	template.Status = types.OrganizeTemplateStatusEnabled
	template.PublishedVersion = version
	template.PublishedAt = &now
	template.PublishedBy = strings.TrimSpace(actorID)
	template.ValidationResult = types.JSONMap{
		"valid":        true,
		"errors":       []string{},
		"version":      version,
		"validated_at": now.Format(time.RFC3339),
	}
	template.UpdatedAt = now
	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, err
	}
	return s.GetAdminTemplate(ctx, template.Key)
}

func (s *organizeService) DisableAdminTemplate(
	ctx context.Context,
	key, actorID string,
) (*types.OrganizeTemplate, error) {
	template, err := s.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	template.Status = types.OrganizeTemplateStatusDisabled
	template.UpdatedAt = time.Now().UTC()
	template.ValidationResult = types.JSONMap{
		"valid":       true,
		"disabled_by": strings.TrimSpace(actorID),
		"disabled_at": template.UpdatedAt.Format(time.RFC3339),
	}
	if err := s.repo.UpdateTemplate(ctx, template); err != nil {
		return nil, err
	}
	return s.GetAdminTemplate(ctx, template.Key)
}

func (s *organizeService) ListAdminTemplateVersions(
	ctx context.Context,
	query types.OrganizeTemplateVersionQuery,
) ([]*types.OrganizeTemplateVersion, int64, error) {
	query.TemplateKey = strings.TrimSpace(query.TemplateKey)
	if query.TemplateKey == "" {
		return nil, 0, ErrOrganizeAdminTemplateKeyRequired
	}
	if _, err := s.GetAdminTemplate(ctx, query.TemplateKey); err != nil {
		return nil, 0, err
	}
	query.Page, query.PageSize = normalizeOrganizePage(query.Page, query.PageSize)
	return s.repo.ListTemplateVersions(ctx, query)
}

func (s *organizeService) RollbackAdminTemplate(
	ctx context.Context,
	key, version, actorID, changeNote string,
) (*types.OrganizeTemplate, error) {
	template, err := s.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	version = strings.TrimSpace(version)
	if version == "" {
		return nil, ErrOrganizeAdminTemplateVersionNeeded
	}
	previous, err := s.repo.GetTemplateVersion(ctx, template.ID, version)
	if err != nil {
		return nil, err
	}
	if previous == nil {
		return nil, ErrOrganizeNotFound
	}
	restored, err := adminTemplateFromSnapshot(previous.Snapshot)
	if err != nil {
		return nil, err
	}
	restored.ID = template.ID
	restored.Key = template.Key
	restored.TenantID = template.TenantID
	restored.OwnerUserID = template.OwnerUserID
	restored.Scope = template.Scope
	restored.Status = types.OrganizeTemplateStatusEnabled

	versions, _, err := s.repo.ListTemplateVersions(ctx, types.OrganizeTemplateVersionQuery{
		TemplateKey: template.Key,
		Page:        1,
		PageSize:    1000,
	})
	if err != nil {
		return nil, err
	}
	nextVersion := nextOrganizeTemplateVersion(versions)
	now := time.Now().UTC()
	snapshot := organizeTemplateSnapshot(restored)
	snapshot["version"] = nextVersion
	snapshot["rollback_from"] = version
	snapshot["change_note"] = strings.TrimSpace(changeNote)
	snapshot["published_by"] = strings.TrimSpace(actorID)
	if err := s.repo.CreateTemplateVersion(ctx, &types.OrganizeTemplateVersion{
		TemplateID:  template.ID,
		TemplateKey: template.Key,
		Version:     nextVersion,
		Snapshot:    snapshot,
		CreatedBy:   strings.TrimSpace(actorID),
		ChangeNote:  strings.TrimSpace(changeNote),
	}); err != nil {
		return nil, err
	}
	restored.PublishedVersion = nextVersion
	restored.PublishedAt = &now
	restored.PublishedBy = strings.TrimSpace(actorID)
	restored.ValidationResult = types.JSONMap{
		"valid":         true,
		"rollback_from": version,
		"validated_at":  now.Format(time.RFC3339),
		"change_note":   strings.TrimSpace(changeNote),
		"published_by":  strings.TrimSpace(actorID),
	}
	restored.UpdatedAt = now
	if err := s.repo.UpdateTemplate(ctx, restored); err != nil {
		return nil, err
	}
	return s.GetAdminTemplate(ctx, template.Key)
}

func (s *organizeService) PreviewAdminTemplate(
	ctx context.Context,
	key string,
	input types.OrganizeTemplatePreviewInput,
) (*types.OrganizeTemplatePreview, error) {
	template, err := s.GetAdminTemplate(ctx, key)
	if err != nil {
		return nil, err
	}
	errorsList := validateOrganizeTemplateSpec(template.Spec)
	vars := make(map[string]string, len(input.Variables)+2)
	for name, value := range input.Variables {
		vars[name] = strings.TrimSpace(fmt.Sprint(value))
	}
	if len(input.MemoryIDs) > 0 || strings.TrimSpace(vars["memory_count"]) == "" {
		vars["memory_count"] = strconv.Itoa(len(input.MemoryIDs))
	}
	vars["template_name"] = template.Name
	prompt := renderOrganizeTemplateInstruction(template.DefaultInstruction, vars)
	markdownTemplate := hydrateOrganizeTemplateMarkdown(template).MarkdownTemplate
	if prompt == "" {
		prompt = organizeMarkdownTemplatePrompt(markdownTemplate, template.Spec)
	} else {
		prompt = prompt + "\n\n" + organizeMarkdownTemplatePrompt(markdownTemplate, template.Spec)
	}
	return &types.OrganizeTemplatePreview{
		TemplateKey:      template.Key,
		Version:          template.PublishedVersion,
		Prompt:           prompt,
		MarkdownTemplate: markdownTemplate,
		PreviewMarkdown:  renderOrganizeTemplatePreviewMarkdown(template, markdownTemplate, vars),
		Spec:             template.Spec,
		Errors:           errorsList,
	}, nil
}

func renderOrganizeTemplatePreviewMarkdown(
	template *types.OrganizeTemplate,
	markdownTemplate string,
	variables map[string]string,
) string {
	values := make(map[string]string, len(variables)+8)
	for key, value := range variables {
		values[key] = strings.TrimSpace(value)
	}

	title := emptyFallback(values["title"], strings.TrimSpace(template.Name))
	if title == "" {
		title = "整理结果示例"
	}
	values["title"] = title

	summary := emptyFallback(values["summary"], strings.TrimSpace(template.Description))
	if summary == "" {
		summary = fmt.Sprintf(
			"基于 %s 条示例记忆，已归纳关键事实、判断依据和后续行动。",
			emptyFallback(values["memory_count"], "3"),
		)
	}
	values["summary"] = strings.TrimSuffix(summary, "。") + "。"

	sections := organizeTemplateStringList(
		types.JSONMap{"template_spec": template.Spec},
		"template_spec",
		"sections",
	)
	for index, section := range sections {
		key := fmt.Sprintf("section_%d", index+1)
		if strings.TrimSpace(values[key]) == "" {
			values[key] = organizeTemplatePreviewSectionContent(section)
		}
	}

	if strings.TrimSpace(values["citations"]) == "" {
		values["citations"] = "- [M1] 示例记忆 1：原始沟通与事实记录\n- [M2] 示例记忆 2：补充材料与行动记录"
	}
	if strings.TrimSpace(values["tags"]) == "" {
		tags := organizeTemplateStringList(
			types.JSONMap{"template_spec": template.Spec},
			"template_spec",
			"tags",
		)
		if len(tags) == 0 && strings.TrimSpace(template.Scene) != "" {
			tags = []string{template.Scene}
		}
		if len(tags) == 0 {
			tags = []string{"整理示例"}
		}
		for index, tag := range tags {
			tag = strings.TrimSpace(strings.TrimPrefix(tag, "#"))
			tags[index] = "#" + tag
		}
		values["tags"] = strings.Join(tags, " ")
	}

	preview := organizeAdminPreviewPlaceholderPattern.ReplaceAllStringFunc(
		markdownTemplate,
		func(placeholder string) string {
			match := organizeAdminPreviewPlaceholderPattern.FindStringSubmatch(placeholder)
			if len(match) != 2 {
				return "示例内容"
			}
			key := strings.TrimSpace(match[1])
			if value := strings.TrimSpace(values[key]); value != "" {
				return value
			}
			return organizeTemplatePreviewPlaceholderContent(key)
		},
	)
	return normalizeOrganizeGeneratedMarkdown(preview, markdownTemplate, title, template.Spec)
}

func organizeTemplatePreviewSectionContent(section string) string {
	section = strings.TrimSpace(section)
	switch {
	case strings.Contains(section, "材料") || strings.Contains(section, "成本") || strings.Contains(section, "预算"):
		return "优先复用现有材料，补充必要采购项，并同步记录数量、预算和替代方案。"
	case strings.Contains(section, "实施") || strings.Contains(section, "行动") ||
		strings.Contains(section, "下一步") || strings.Contains(section, "跟进") ||
		strings.Contains(section, "待办"):
		return "确认实施顺序、负责人和完成时间，并在执行后回写结果"
	case strings.Contains(section, "动线") || strings.Contains(section, "安全"):
		return "保留主要通行路径和观察视线，重点区域增加缓冲空间与安全检查。"
	case strings.Contains(section, "现状") || strings.Contains(section, "背景"):
		return "已汇总当前条件、核心需求和主要限制，关键问题及影响范围已形成统一记录。"
	case strings.Contains(section, "依据") || strings.Contains(section, "证据"):
		return "关键判断均保留对应记忆来源，方便后续复核和补充。"
	default:
		return fmt.Sprintf("围绕“%s”归纳了示例记忆中的关键事实、已有判断和需要继续确认的信息。", emptyFallback(section, "本章节"))
	}
}

func organizeTemplatePreviewPlaceholderContent(key string) string {
	label := strings.NewReplacer("_", " ", "-", " ", ".", " ").Replace(strings.TrimSpace(key))
	if strings.HasSuffix(key, "_rows") {
		return "示例项目 | 已整理 | 待确认"
	}
	if strings.Contains(key, "question") {
		return "- 需要结合后续记录继续确认的信息"
	}
	return fmt.Sprintf("已根据示例记忆整理%s相关内容。", emptyFallback(label, "本栏目"))
}

func normalizeAdminTemplateInput(input types.OrganizeTemplateAdminInput) (*types.OrganizeTemplate, error) {
	key := strings.TrimSpace(input.Key)
	if key == "" {
		return nil, ErrOrganizeAdminTemplateKeyRequired
	}
	if !organizeAdminTemplateKeyPattern.MatchString(key) {
		return nil, ErrOrganizeAdminTemplateKeyInvalid
	}
	if types.IsOrganizeInternalTemplateKey(key) {
		return nil, ErrOrganizeAdminTemplateKeyReserved
	}
	name := trimMax(input.Name, organizeMaxTitleLength)
	if name == "" {
		return nil, ErrOrganizeAdminTemplateNameRequired
	}
	spec := normalizeJSONMap(input.Spec)
	if errorsList := validateOrganizeTemplateSpec(spec); len(errorsList) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrOrganizeAdminTemplateInvalidSpec, strings.Join(errorsList, "; "))
	}
	markdownTemplate, markdownErrors := normalizeOrganizeMarkdownTemplate(input.MarkdownTemplate, key, spec)
	if len(markdownErrors) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrOrganizeAdminTemplateInvalidMarkdown, strings.Join(markdownErrors, "; "))
	}
	expertIDs := append(types.StringArray(nil), input.ExpertIDs...)
	if expertIDs == nil {
		expertIDs = types.StringArray{}
	}
	return &types.OrganizeTemplate{
		Key:                key,
		Name:               name,
		Scene:              trimMax(input.Scene, 128),
		Description:        trimMax(input.Description, 0),
		OutputLabel:        trimMax(input.OutputLabel, 128),
		Icon:               trimMax(input.Icon, 64),
		DefaultInstruction: trimMax(input.DefaultInstruction, 0),
		MarkdownTemplate:   markdownTemplate,
		ExpertIDs:          expertIDs,
		Spec:               spec,
		SortOrder:          input.SortOrder,
	}, nil
}

func validateOrganizeTemplateSpec(spec types.JSONMap) []string {
	if spec == nil {
		return []string{"spec must be an object"}
	}
	errorsList := make([]string, 0)
	if raw, ok := spec["fields"]; ok {
		switch fields := raw.(type) {
		case []any:
			for index, rawField := range fields {
				field, ok := rawField.(map[string]any)
				if !ok {
					errorsList = append(errorsList, fmt.Sprintf("fields[%d] must be an object", index))
					continue
				}
				if strings.TrimSpace(fmt.Sprint(field["key"])) == "" {
					errorsList = append(errorsList, fmt.Sprintf("fields[%d].key is required", index))
				}
				if strings.TrimSpace(fmt.Sprint(field["label"])) == "" {
					errorsList = append(errorsList, fmt.Sprintf("fields[%d].label is required", index))
				}
			}
		case map[string]any:
			for key, rawField := range fields {
				if strings.TrimSpace(key) == "" {
					errorsList = append(errorsList, "fields contains an empty key")
					continue
				}
				field, ok := rawField.(map[string]any)
				if !ok {
					continue
				}
				if label := strings.TrimSpace(fmt.Sprint(field["label"])); label == "" {
					errorsList = append(errorsList, fmt.Sprintf("fields.%s.label is required", key))
				}
			}
		default:
			errorsList = append(errorsList, "fields must be an array or object")
		}
	}
	return errorsList
}

func organizeTemplateSnapshot(template *types.OrganizeTemplate) types.JSONMap {
	return types.JSONMap{
		"name":                template.Name,
		"scene":               template.Scene,
		"description":         template.Description,
		"output_label":        template.OutputLabel,
		"icon":                template.Icon,
		"default_instruction": template.DefaultInstruction,
		"markdown_template":   template.MarkdownTemplate,
		"expert_ids":          append([]string(nil), template.ExpertIDs...),
		"spec":                normalizeJSONMap(template.Spec),
		"sort_order":          template.SortOrder,
	}
}

func adminTemplateFromSnapshot(snapshot types.JSONMap) (*types.OrganizeTemplate, error) {
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	var data struct {
		Name               string            `json:"name"`
		Scene              string            `json:"scene"`
		Description        string            `json:"description"`
		OutputLabel        string            `json:"output_label"`
		Icon               string            `json:"icon"`
		DefaultInstruction string            `json:"default_instruction"`
		MarkdownTemplate   string            `json:"markdown_template"`
		ExpertIDs          types.StringArray `json:"expert_ids"`
		Spec               types.JSONMap     `json:"spec"`
		SortOrder          int               `json:"sort_order"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	template := &types.OrganizeTemplate{
		Name:               data.Name,
		Scene:              data.Scene,
		Description:        data.Description,
		OutputLabel:        data.OutputLabel,
		Icon:               data.Icon,
		DefaultInstruction: data.DefaultInstruction,
		MarkdownTemplate:   data.MarkdownTemplate,
		ExpertIDs:          data.ExpertIDs,
		Spec:               data.Spec,
		SortOrder:          data.SortOrder,
	}
	normalized, err := normalizeAdminTemplateInput(types.OrganizeTemplateAdminInput{
		Key:                "restored",
		Name:               template.Name,
		Scene:              template.Scene,
		Description:        template.Description,
		OutputLabel:        template.OutputLabel,
		Icon:               template.Icon,
		DefaultInstruction: template.DefaultInstruction,
		MarkdownTemplate:   template.MarkdownTemplate,
		ExpertIDs:          template.ExpertIDs,
		Spec:               template.Spec,
		SortOrder:          template.SortOrder,
	})
	if err != nil {
		return nil, err
	}
	template.MarkdownTemplate = normalized.MarkdownTemplate
	return template, nil
}

func nextOrganizeTemplateVersion(versions []*types.OrganizeTemplateVersion) string {
	next := 1
	for _, version := range versions {
		if version == nil {
			continue
		}
		value := strings.TrimPrefix(strings.TrimSpace(version.Version), "v")
		number, err := strconv.Atoi(value)
		if err == nil && number >= next {
			next = number + 1
		}
	}
	return fmt.Sprintf("v%d", next)
}

func renderOrganizeTemplateInstruction(instruction string, variables map[string]string) string {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		return ""
	}
	keys := make([]string, 0, len(variables))
	for key := range variables {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		instruction = strings.ReplaceAll(instruction, "{{"+key+"}}", variables[key])
		instruction = strings.ReplaceAll(instruction, "${"+key+"}", variables[key])
	}
	return instruction
}

func (s *organizeService) resolvePlatformTemplateInstruction(
	ctx context.Context,
	key string,
	fallback string,
	variables map[string]string,
) string {
	instruction := strings.TrimSpace(fallback)
	template, err := s.repo.GetAdminTemplate(ctx, strings.TrimSpace(key))
	if err == nil && template != nil && template.Status == types.OrganizeTemplateStatusEnabled {
		if value := strings.TrimSpace(template.DefaultInstruction); value != "" {
			instruction = value
		}
	}
	return renderOrganizeTemplateInstruction(instruction, variables)
}
