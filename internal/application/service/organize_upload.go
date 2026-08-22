package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/models/asr"
	"github.com/Tencent/WeKnora/internal/models/chat"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/google/uuid"
)

const (
	organizeUploadPromptRuneBudget = 7000
	organizeUploadMaxTags           = 8
	organizeUploadMaxTagRunes       = 16
)

type organizeUploadAIResult struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Tags    []string `json:"tags"`
}

func (s *organizeService) CreateOutputFromUpload(
	ctx context.Context,
	tenantID uint64,
	userID, fileName, mimeType string,
	data []byte,
) (*types.OrganizeOutput, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	if s.fileService == nil {
		return nil, fmt.Errorf("file service is not configured")
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("upload file is empty")
	}

	cleanName := strings.TrimSpace(fileName)
	if cleanName == "" {
		cleanName = "upload.bin"
	}
	if !isValidFileType(cleanName) {
		return nil, fmt.Errorf("unsupported file type: %s", strings.ToLower(filepath.Ext(cleanName)))
	}

	contentKind, outputType, icon := organizeOutputKindInfo(cleanName, mimeType)
	baseName := strings.TrimSuffix(filepath.Base(cleanName), filepath.Ext(cleanName))
	if baseName == "" {
		baseName = cleanName
	}

	content, transcript, asrModelID, warnings, err := s.extractOrganizeUploadContent(ctx, cleanName, mimeType, data, contentKind)
	if err != nil {
		return nil, err
	}

	aiResult, aiModelID, aiStatus := s.generateOrganizeUploadAIResult(ctx, cleanName, outputType, content)
	if aiResult.Title == "" {
		aiResult.Title = baseName
	}
	if aiResult.Summary == "" {
		aiResult.Summary = organizeUploadFallbackSummary(cleanName, outputType, content)
	}
	aiResult.Tags = normalizeOrganizeUploadTags(append(aiResult.Tags, organizeUploadFallbackTags(outputType)...))
	if len(aiResult.Tags) == 0 {
		aiResult.Tags = normalizeOrganizeUploadTags(organizeUploadFallbackTags(outputType))
	}

	storageName := fmt.Sprintf("organize_output_%s%s", uuid.NewString()[:12], filepath.Ext(cleanName))
	storagePath, saveErr := s.fileService.SaveBytes(ctx, data, tenantID, storageName, false)
	if saveErr != nil {
		return nil, fmt.Errorf("save upload file: %w", saveErr)
	}

	metadata := types.JSONMap{
		"content_kind":        contentKind,
		"content_kind_label":  outputType,
		"file_name":           cleanName,
		"file_type":           strings.TrimPrefix(strings.ToLower(filepath.Ext(cleanName)), "."),
		"file_path":           storagePath,
		"mime_type":           strings.TrimSpace(mimeType),
		"ai_status":           aiStatus,
		"ai_model_id":         aiModelID,
		"tags":                types.StringArray(aiResult.Tags),
		"upload_source":       "file",
		"uploaded_at":         time.Now().UTC().Format(time.RFC3339),
	}
	if transcript != "" {
		metadata["transcript"] = transcript
	}
	if asrModelID != "" {
		metadata["asr_model_id"] = asrModelID
	}
	if len(warnings) > 0 {
		metadata["warnings"] = warnings
	}

	output := &types.OrganizeOutput{
		TenantID:      tenantID,
		UserID:        userID,
		Title:         trimMax(aiResult.Title, organizeMaxTitleLength),
		OutputType:    outputType,
		Content:       strings.TrimSpace(content),
		SourceSummary: trimMax(aiResult.Summary, organizeMaxShortText),
		Status:        types.OrganizeOutputStatusReview,
		Icon:          icon,
		Metadata:      normalizeJSONMap(metadata),
	}
	if err := s.repo.CreateOutput(ctx, output, nil); err != nil {
		_ = s.fileService.DeleteFile(ctx, storagePath)
		return nil, err
	}
	return s.repo.GetOutput(ctx, tenantID, userID, output.ID)
}

func (s *organizeService) extractOrganizeUploadContent(
	ctx context.Context,
	fileName, mimeType string,
	data []byte,
	contentKind string,
) (content string, transcript string, asrModelID string, warnings []string, err error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	content = strings.TrimSpace(string(data))
	if shouldUseRawTextForOrganizeUpload(ext) && content != "" {
		return content, "", "", nil, nil
	}

	if s.documentReader != nil {
		req := &types.ReadRequest{
			FileContent: data,
			FileName:    fileName,
			FileType:    ext,
		}
		if tenant, ok := ctx.Value(types.TenantInfoContextKey).(*types.Tenant); ok && tenant != nil && tenant.ParserEngineConfig != nil {
			req.ParserEngineOverrides = tenant.ParserEngineConfig.ToOverridesMap()
		}
		readResult, readErr := s.documentReader.Read(ctx, req)
		if readErr != nil {
			warnings = append(warnings, readErr.Error())
		}
		if readResult != nil {
			if strings.TrimSpace(readResult.MarkdownContent) != "" {
				content = strings.TrimSpace(readResult.MarkdownContent)
			}
			if readResult.IsAudio && len(readResult.AudioData) > 0 {
				transcript, asrModelID, err = s.transcribeOrganizeUploadAudio(ctx, fileName, readResult.AudioData)
				if err != nil {
					warnings = append(warnings, err.Error())
				}
				if strings.TrimSpace(transcript) != "" {
					content = strings.TrimSpace(transcript)
				}
			}
		}
	}

	if content == "" {
		if contentKind == organizeOutputKindAudio || contentKind == organizeOutputKindVideo {
			content = fmt.Sprintf("%s 文件 %s", organizeOutputKindLabel(contentKind), fileName)
		} else {
			content = strings.TrimSpace(string(data))
		}
	}
	if content == "" {
		content = fileName
	}
	return content, transcript, asrModelID, warnings, nil
}

func (s *organizeService) transcribeOrganizeUploadAudio(
	ctx context.Context,
	fileName string,
	audioBytes []byte,
) (string, string, error) {
	modelID := s.resolveOrganizeModelID(ctx, types.ModelTypeASR)
	if modelID == "" || s.modelService == nil {
		return "", "", nil
	}
	asrModel, err := s.modelService.GetASRModel(ctx, modelID)
	if err != nil {
		return "", modelID, err
	}
	result, err := asrModel.Transcribe(ctx, audioBytes, fileName)
	if err != nil {
		if asr.IsNonRetryable(err) {
			return "", modelID, err
		}
		return "", modelID, err
	}
	if result == nil {
		return "", modelID, nil
	}
	return strings.TrimSpace(result.Text), modelID, nil
}

func (s *organizeService) generateOrganizeUploadAIResult(
	ctx context.Context,
	fileName, outputType, content string,
) (organizeUploadAIResult, string, string) {
	modelID := s.resolveOrganizeModelID(ctx, types.ModelTypeKnowledgeQA)
	if modelID == "" || s.modelService == nil {
		return organizeUploadFallbackAIResult(fileName, outputType, content), "", "fallback"
	}

	chatModel, err := s.modelService.GetChatModel(ctx, modelID)
	if err != nil || chatModel == nil {
		return organizeUploadFallbackAIResult(fileName, outputType, content), modelID, "fallback"
	}

	prompt := fmt.Sprintf(`你在为一个成果分享模块生成卡片元数据。

请只输出 JSON，格式如下：
{"title":"标题","summary":"摘要","tags":["标签1","标签2"]}

要求：
- 标题简短准确，优先概括内容核心。
- 摘要用 1-2 句话说明这份内容能帮助观看者提升什么业务能力。
- 标签使用简洁、具体的中文词组，最多 %d 个，避免“文档”“文件”“其他”这类泛泛词。
- 标签应尽量贴近园所招生、业务能力提升、内容类型或主题。

内容类型：%s
文件名：%s

内容：
%s`, organizeUploadMaxTags, outputType, fileName, sampleRunes(strings.TrimSpace(content), organizeUploadPromptRuneBudget, "…"))

	thinking := false
	resp, chatErr := chatModel.Chat(ctx, []chat.Message{
		{Role: "system", Content: "你是一个严格的结构化信息提取助手，只能输出 JSON。"},
		{Role: "user", Content: prompt},
	}, &chat.ChatOptions{
		Temperature: 0.2,
		MaxTokens:   512,
		Thinking:    &thinking,
	})
	if chatErr != nil || resp == nil {
		return organizeUploadFallbackAIResult(fileName, outputType, content), modelID, "fallback"
	}

	parsed, ok := parseOrganizeUploadAIResponse(resp.Content)
	if !ok {
		return organizeUploadFallbackAIResult(fileName, outputType, content), modelID, "fallback"
	}

	parsed.Title = trimMax(parsed.Title, organizeMaxTitleLength)
	parsed.Summary = trimMax(parsed.Summary, organizeMaxShortText)
	parsed.Tags = normalizeOrganizeUploadTags(parsed.Tags)
	if parsed.Title == "" {
		parsed.Title = organizeUploadFallbackAIResult(fileName, outputType, content).Title
	}
	if parsed.Summary == "" {
		parsed.Summary = organizeUploadFallbackAIResult(fileName, outputType, content).Summary
	}
	if len(parsed.Tags) == 0 {
		parsed.Tags = organizeUploadFallbackTags(outputType)
	}
	return parsed, modelID, "completed"
}

func (s *organizeService) resolveOrganizeModelID(ctx context.Context, modelType types.ModelType) string {
	if s.modelService == nil {
		return ""
	}
	models, err := s.modelService.ListModels(ctx)
	if err != nil {
		return ""
	}
	var fallback string
	for _, model := range models {
		if model == nil || model.Status != types.ModelStatusActive || model.Type != modelType {
			continue
		}
		if model.IsDefault {
			return model.ID
		}
		if fallback == "" {
			fallback = model.ID
		}
	}
	return fallback
}

func parseOrganizeUploadAIResponse(content string) (organizeUploadAIResult, bool) {
	content = strings.TrimSpace(content)
	if content == "" {
		return organizeUploadAIResult{}, false
	}
	if parsed, ok := parseOrganizeUploadAIResponseJSON(content); ok {
		return parsed, true
	}
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start >= 0 && end > start {
		if parsed, ok := parseOrganizeUploadAIResponseJSON(content[start : end+1]); ok {
			return parsed, true
		}
	}
	return organizeUploadAIResult{}, false
}

func parseOrganizeUploadAIResponseJSON(content string) (organizeUploadAIResult, bool) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &obj); err != nil {
		return organizeUploadAIResult{}, false
	}
	out := organizeUploadAIResult{
		Title:   firstOrganizeUploadStringField(obj, "title", "name"),
		Summary: firstOrganizeUploadStringField(obj, "summary", "description", "desc"),
		Tags:    firstOrganizeUploadStringSliceField(obj, "tags", "keywords"),
	}
	return out, true
}

func firstOrganizeUploadStringField(obj map[string]json.RawMessage, keys ...string) string {
	for _, key := range keys {
		raw, ok := obj[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func firstOrganizeUploadStringSliceField(obj map[string]json.RawMessage, keys ...string) []string {
	for _, key := range keys {
		raw, ok := obj[key]
		if !ok || len(raw) == 0 {
			continue
		}
		var arr []string
		if err := json.Unmarshal(raw, &arr); err == nil {
			return arr
		}
	}
	return nil
}

const (
	organizeOutputKindArticle = "article"
	organizeOutputKindVideo   = "video"
	organizeOutputKindAudio   = "audio"
)

func organizeOutputKindInfo(fileName, mimeType string) (kind, label, icon string) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	switch {
	case IsAudioType(ext) || strings.HasPrefix(contentType, "audio/"):
		return organizeOutputKindAudio, "音频类", "sound"
	case IsVideoType(ext) || strings.HasPrefix(contentType, "video/"):
		return organizeOutputKindVideo, "视频类", "play-circle"
	default:
		return organizeOutputKindArticle, "图文类", "file-word"
	}
}

func organizeOutputKindLabel(kind string) string {
	switch kind {
	case organizeOutputKindVideo:
		return "视频"
	case organizeOutputKindAudio:
		return "音频"
	default:
		return "图文"
	}
}

func shouldUseRawTextForOrganizeUpload(ext string) bool {
	switch strings.ToLower(strings.TrimPrefix(ext, ".")) {
	case "md", "markdown", "txt", "csv", "json", "xml", "yaml", "yml", "log", "html", "htm":
		return true
	default:
		return false
	}
}

func organizeUploadFallbackAIResult(fileName, outputType, content string) organizeUploadAIResult {
	title := strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))
	if title == "" {
		title = fileName
	}
	summary := organizeUploadFallbackSummary(fileName, outputType, content)
	return organizeUploadAIResult{
		Title:   title,
		Summary: summary,
		Tags:    organizeUploadFallbackTags(outputType),
	}
}

func organizeUploadFallbackSummary(fileName, outputType, content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return fmt.Sprintf("%s %s，适合用于成果分享和业务学习。", outputType, strings.TrimSpace(strings.TrimSuffix(filepath.Base(fileName), filepath.Ext(fileName))))
	}
	snippet := strings.TrimSpace(sampleRunes(content, 120, "…"))
	if snippet == "" {
		return fmt.Sprintf("%s 内容摘要待补充。", outputType)
	}
	return snippet
}

func organizeUploadFallbackTags(outputType string) []string {
	tags := []string{outputType, "业务提升"}
	switch outputType {
	case "视频类":
		tags = append(tags, "视频学习")
	case "音频类":
		tags = append(tags, "音频学习")
	default:
		tags = append(tags, "图文学习")
	}
	return normalizeOrganizeUploadTags(tags)
}

func normalizeOrganizeUploadTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		tag = cleanAutoTagName(tag)
		if tag == "" {
			continue
		}
		runes := []rune(tag)
		if len(runes) > organizeUploadMaxTagRunes {
			tag = string(runes[:organizeUploadMaxTagRunes])
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		cleaned = append(cleaned, tag)
		if len(cleaned) >= organizeUploadMaxTags {
			break
		}
	}
	return cleaned
}
