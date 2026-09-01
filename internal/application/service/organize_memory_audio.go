package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/models/asr"
	"github.com/Tencent/WeKnora/internal/tracing/langfuse"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
)

const (
	organizeMemoryTranscribeRetryCount = 2
	organizeMemoryTranscribeTimeout    = 15 * time.Minute
)

func (s *organizeService) CreateMemoryFromUpload(
	ctx context.Context,
	tenantID uint64,
	userID, fileName, mimeType string,
	data []byte,
	input types.OrganizeMemoryInput,
) (*types.OrganizeMemory, error) {
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
		cleanName = "audio.mp3"
	}
	if !isValidFileType(cleanName) {
		return nil, fmt.Errorf("unsupported file type: %s", strings.ToLower(filepath.Ext(cleanName)))
	}

	safeName, valid := secutils.ValidateInput(cleanName)
	if !valid {
		return nil, fmt.Errorf("invalid characters in file name")
	}
	baseName, err := secutils.SafeFileName(safeName)
	if err != nil {
		return nil, fmt.Errorf("unsafe file name: %w", err)
	}
	baseTitle := strings.TrimSuffix(filepath.Base(baseName), filepath.Ext(baseName))
	if baseTitle == "" {
		baseTitle = strings.TrimSuffix(cleanName, filepath.Ext(cleanName))
	}
	if baseTitle == "" {
		baseTitle = "音频记忆"
	}

	storedBytes, storedName, err := s.normalizeOrganizeAudioForStorage(ctx, data, baseName)
	if err != nil {
		return nil, fmt.Errorf("normalize audio for storage: %w", err)
	}
	if storedName == "" {
		storedName = replaceOrganizeAudioExtension(baseName, ".mp3")
	}

	kind := normalizeMemoryKind(input.Kind)
	if kind == "" {
		kind = types.OrganizeMemoryKindAudio
	}
	if !types.IsValidOrganizeMemoryKind(kind) {
		return nil, ErrOrganizeInvalidMemoryKind
	}

	title := trimMax(input.Title, organizeMaxTitleLength)
	if title == "" {
		title = trimMax(baseTitle, organizeMaxTitleLength)
	}
	if title == "" {
		title = "音频记忆"
	}

	occurredAt := time.Now().UTC()
	if input.OccurredAt != nil && !input.OccurredAt.IsZero() {
		occurredAt = input.OccurredAt.UTC()
	}
	source := trimMax(input.Source, organizeMaxShortText)
	if source == "" {
		if kind == types.OrganizeMemoryKindAudioCard {
			source = "录音卡"
		} else {
			source = "语音记录"
		}
	}

	storageName := fmt.Sprintf("organize_memory_%s%s", uuid.NewString()[:12], filepath.Ext(storedName))
	storagePath, saveErr := s.fileService.SaveBytes(ctx, storedBytes, tenantID, storageName, false)
	if saveErr != nil {
		return nil, fmt.Errorf("save audio file: %w", saveErr)
	}

	audioURL := resolveOrganizeMemoryAudioURL(ctx, s.fileService, storagePath)

	content := trimMax(input.Content, 0)
	if content == "" {
		content = "<p>录音已保存，等待转写。</p>"
	}

	metadata := normalizeJSONMap(input.Metadata)
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	metadata["file_name"] = trimMax(storedName, 0)
	metadata["file_path"] = trimMax(storagePath, 0)
	metadata["file_url"] = audioURL
	metadata["audio_file_name"] = trimMax(storedName, 0)
	metadata["audio_file_path"] = trimMax(storagePath, 0)
	metadata["audio_url"] = audioURL
	metadata["audio_mime_type"] = "audio/mpeg"
	metadata["audio_codec"] = "mp3"
	metadata["audio_size_bytes"] = len(storedBytes)
	metadata["transcription_status"] = "pending"
	if _, ok := metadata["recording_file_name"]; !ok {
		metadata["recording_file_name"] = trimMax(baseTitle, organizeMaxTitleLength)
	}
	if _, ok := metadata["sync_source"]; !ok {
		metadata["sync_source"] = trimMax(memoryUploadSource(kind), organizeMaxShortText)
	}

	memory := &types.OrganizeMemory{
		TenantID:        tenantID,
		UserID:          userID,
		Kind:            kind,
		Title:           title,
		Content:         content,
		Source:          source,
		OccurredAt:      occurredAt,
		DurationSeconds: nonNegative(input.DurationSeconds),
		Metadata:        metadata,
	}
	if err := s.repo.CreateMemory(ctx, memory); err != nil {
		_ = s.fileService.DeleteFile(ctx, storagePath)
		return nil, err
	}

	if err := s.scheduleMemoryTranscription(ctx, tenantID, memory.ID); err != nil {
		logger.Warnf(ctx, "[Organize] schedule memory transcription failed: %v", err)
		memory.Metadata["transcription_status"] = "queued_failed"
		memory.Metadata["transcription_error"] = trimMax(err.Error(), organizeMaxShortText)
		memory.UpdatedAt = time.Now().UTC()
		_ = s.repo.UpdateMemory(ctx, memory)
	}

	return s.repo.GetMemory(ctx, tenantID, userID, memory.ID)
}

func resolveOrganizeMemoryAudioURL(
	ctx context.Context,
	fileService interfaces.FileService,
	storagePath string,
) string {
	storagePath = trimMax(storagePath, 0)
	if storagePath == "" {
		return ""
	}
	if fileService != nil {
		if resolved, err := fileService.GetFileURL(ctx, storagePath); err == nil {
			if normalized := strings.TrimSpace(resolved); isOrganizeAccessibleAudioURL(normalized) {
				return normalized
			}
		}
	}
	return organizeAuthenticatedFileURL(storagePath)
}

func isOrganizeAccessibleAudioURL(value string) bool {
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") {
		return true
	}
	parsed, err := neturl.Parse(value)
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return true
	default:
		return false
	}
}

func organizeAuthenticatedFileURL(filePath string) string {
	filePath = trimMax(filePath, 0)
	if filePath == "" {
		return ""
	}
	query := neturl.Values{}
	query.Set("file_path", filePath)
	return "/files?" + query.Encode()
}

func (s *organizeService) normalizeOrganizeAudioForStorage(
	ctx context.Context,
	audioBytes []byte,
	fileName string,
) ([]byte, string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if ext == ".mp3" {
		return audioBytes, replaceOrganizeAudioExtension(fileName, ".mp3"), nil
	}
	transcoder := transcodeOrganizeAudioToMP3
	if s != nil && s.audioTranscoder != nil {
		transcoder = s.audioTranscoder
	}
	return transcoder(ctx, audioBytes, fileName)
}

func (s *organizeService) ProcessMemoryTranscribe(ctx context.Context, task *asynq.Task) error {
	var payload types.OrganizeMemoryTranscribeTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode organize memory transcribe task: %w", err)
	}
	if payload.TenantID == 0 || strings.TrimSpace(payload.MemoryID) == "" {
		return fmt.Errorf("invalid organize memory transcribe payload")
	}
	ctx = context.WithValue(ctx, types.TenantIDContextKey, payload.TenantID)

	memory, err := s.repo.GetTenantMemory(ctx, payload.TenantID, strings.TrimSpace(payload.MemoryID))
	if err != nil {
		return err
	}
	if memory == nil {
		return nil
	}

	metadata := normalizeJSONMap(memory.Metadata)
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	metadata["transcription_status"] = "transcribing"
	metadata["transcription_error"] = ""
	memory.Metadata = metadata
	memory.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMemory(ctx, memory); err != nil {
		return err
	}

	filePath := organizeMemoryAudioPath(metadata)
	if filePath == "" {
		return s.failOrganizeMemoryTranscription(ctx, memory, "audio file path missing", nil)
	}

	audioFileName := organizeMemoryAudioFileName(metadata, filePath)
	file, err := s.fileService.GetFile(ctx, filePath)
	if err != nil {
		return s.failOrganizeMemoryTranscription(ctx, memory, "failed to open audio file", err)
	}
	audioBytes, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil {
		return s.failOrganizeMemoryTranscription(ctx, memory, "failed to read audio file", readErr)
	}
	if len(audioBytes) == 0 {
		return s.failOrganizeMemoryTranscription(ctx, memory, "audio file is empty", nil)
	}

	normalizedBytes, normalizedName, normalizeErr := normalizeOrganizeAudioForASR(ctx, audioBytes, audioFileName)
	if normalizeErr != nil {
		return s.failOrganizeMemoryTranscription(ctx, memory, "failed to normalize audio", normalizeErr)
	}

	modelID := s.resolveOrganizeModelID(ctx, types.ModelTypeASR)
	if modelID == "" || s.modelService == nil {
		memory.Metadata["transcription_status"] = "skipped"
		memory.Metadata["transcription_error"] = ""
		memory.Metadata["transcription_reason"] = "asr model not configured"
		memory.UpdatedAt = time.Now().UTC()
		return s.repo.UpdateMemory(ctx, memory)
	}

	asrModel, err := s.modelService.GetASRModel(ctx, modelID)
	if err != nil {
		return s.failOrganizeMemoryTranscription(ctx, memory, "failed to load asr model", err)
	}
	result, err := asrModel.Transcribe(ctx, normalizedBytes, normalizedName)
	if err != nil {
		return s.failOrganizeMemoryTranscription(ctx, memory, "transcription failed", err)
	}

	transcript := ""
	if result != nil {
		transcript = trimMax(result.Text, 0)
	}
	if transcript == "" {
		transcript = "未识别到语音内容。"
	}

	now := time.Now().UTC()
	aiResult, aiModelID, aiStatus := s.generateOrganizeRecordingNoteAIResult(
		ctx,
		memory.Title,
		audioFileName,
		memory.Source,
		transcript,
	)
	noteMarkdown := strings.TrimSpace(aiResult.NoteMarkdown)
	if noteMarkdown == "" {
		noteMarkdown = transcript
	}
	if title := trimMax(aiResult.Title, organizeMaxTitleLength); title != "" {
		memory.Title = title
	}
	memory.Content = organizeUploadContentToNoteHTML(memory.Title, noteMarkdown)
	memory.Metadata["transcription_status"] = "completed"
	memory.Metadata["transcription_error"] = ""
	memory.Metadata["transcript"] = transcript
	memory.Metadata["transcribed_at"] = now.Format(time.RFC3339)
	memory.Metadata["asr_model_id"] = modelID
	memory.Metadata["summary"] = trimMax(aiResult.Summary, organizeMaxShortText)
	memory.Metadata["tags"] = types.StringArray(aiResult.Tags)
	memory.Metadata["note_generation_status"] = aiStatus
	memory.Metadata["note_generated_at"] = now.Format(time.RFC3339)
	if aiModelID != "" {
		memory.Metadata["ai_model_id"] = aiModelID
	}
	if normalizedName != audioFileName {
		memory.Metadata["transcription_audio_file_name"] = normalizedName
	}
	memory.UpdatedAt = now
	return s.repo.UpdateMemory(ctx, memory)
}

func (s *organizeService) scheduleMemoryTranscription(ctx context.Context, tenantID uint64, memoryID string) error {
	if s.taskEnqueuer == nil {
		return nil
	}
	if tenantID == 0 {
		return fmt.Errorf("tenant_id is required")
	}
	memoryID = strings.TrimSpace(memoryID)
	if memoryID == "" {
		return fmt.Errorf("memory_id is required")
	}
	payload := types.OrganizeMemoryTranscribeTaskPayload{
		TenantID: tenantID,
		MemoryID: memoryID,
	}
	langfuse.InjectTracing(ctx, &payload)
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	queue, ok := types.QueueForTaskType(types.TypeOrganizeMemoryTranscribe)
	if !ok {
		return fmt.Errorf("queue not declared for task type %s", types.TypeOrganizeMemoryTranscribe)
	}
	_, err = s.taskEnqueuer.Enqueue(
		asynq.NewTask(types.TypeOrganizeMemoryTranscribe, raw),
		asynq.Queue(queue), asynq.MaxRetry(organizeMemoryTranscribeRetryCount), asynq.Timeout(organizeMemoryTranscribeTimeout),
	)
	return err
}

func (s *organizeService) failOrganizeMemoryTranscription(
	ctx context.Context,
	memory *types.OrganizeMemory,
	message string,
	err error,
) error {
	if memory == nil {
		if err != nil {
			return err
		}
		return errors.New(message)
	}

	metadata := normalizeJSONMap(memory.Metadata)
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	metadata["transcription_status"] = "failed"
	metadata["transcription_error"] = trimMax(message, organizeMaxShortText)
	memory.Metadata = metadata
	memory.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateMemory(ctx, memory)
	if err != nil {
		return err
	}
	return nil
}

func memoryUploadSource(kind string) string {
	switch normalizeMemoryKind(kind) {
	case types.OrganizeMemoryKindAudioCard:
		return "录音卡"
	case types.OrganizeMemoryKindAudio:
		return "语音记录"
	default:
		return "录音"
	}
}

func memoryAudioCodec(fileName string, metadata types.JSONMap) string {
	if raw, ok := metadata["audio_codec"]; ok {
		if value := trimMax(fmt.Sprint(raw), 0); value != "" {
			return value
		}
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if ext == "" {
		return "audio"
	}
	return ext
}

func organizeMemoryAudioPath(metadata types.JSONMap) string {
	for _, key := range []string{"audio_file_path", "file_path", "storage_path", "local_audio_path"} {
		if value, ok := metadata[key]; ok {
			if text := trimMax(fmt.Sprint(value), 0); text != "" {
				return text
			}
		}
	}
	return ""
}

func organizeMemoryAudioFileName(metadata types.JSONMap, fallbackPath string) string {
	for _, key := range []string{"audio_file_name", "file_name", "recording_file_name"} {
		if value, ok := metadata[key]; ok {
			if text := trimMax(fmt.Sprint(value), 0); text != "" {
				return text
			}
		}
	}
	base := filepath.Base(fallbackPath)
	if base != "." && base != string(filepath.Separator) {
		return base
	}
	return "audio.m4a"
}

func normalizeOrganizeAudioForASR(ctx context.Context, audioBytes []byte, fileName string) ([]byte, string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName)))
	if isOrganizeASRCompatibleAudio(ext) {
		return audioBytes, fileName, nil
	}
	return transcodeOrganizeAudioToMP3(ctx, audioBytes, fileName)
}

func isOrganizeASRCompatibleAudio(ext string) bool {
	switch strings.TrimPrefix(strings.ToLower(strings.TrimSpace(ext)), ".") {
	case "mp3", "wav", "m4a", "flac", "ogg", "aac", "amr", "opus", "mp4":
		return true
	default:
		return false
	}
}

func transcodeOrganizeAudioToMP3(ctx context.Context, audioBytes []byte, fileName string) ([]byte, string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return nil, "", fmt.Errorf("ffmpeg is required to transcode audio: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "weknora-organize-asr-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	inputExt := asr.DetectAudioFormat(audioBytes, fileName)
	inputPath := filepath.Join(tmpDir, "input"+inputExt)
	outputPath := filepath.Join(tmpDir, "normalized.mp3")
	if err := os.WriteFile(inputPath, audioBytes, 0o600); err != nil {
		return nil, "", fmt.Errorf("write temp input audio: %w", err)
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-y",
		"-i", inputPath,
		"-vn",
		"-ac", "1",
		"-ar", "16000",
		"-b:a", "32k",
		outputPath,
	}
	if err := runOrganizeFFmpeg(ctx, args...); err != nil {
		return nil, "", fmt.Errorf("ffmpeg transcode audio: %w", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, "", fmt.Errorf("read transcoded audio: %w", err)
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("ffmpeg produced empty audio")
	}

	return data, replaceOrganizeAudioExtension(fileName, ".mp3"), nil
}

func runOrganizeFFmpeg(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func replaceOrganizeAudioExtension(fileName, ext string) string {
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if strings.TrimSpace(base) == "" {
		return "audio" + ext
	}
	return base + ext
}
