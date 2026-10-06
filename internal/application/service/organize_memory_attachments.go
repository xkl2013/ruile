package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/google/uuid"
)

// CreateMemoryFromUploads creates one memory container and one attachment row
// per uploaded file. The legacy single-file method delegates here for new
// document imports while its historical audio metadata remains supported.
func (s *organizeService) CreateMemoryFromUploads(
	ctx context.Context,
	tenantID uint64,
	userID string,
	uploads []types.OrganizeMemoryUpload,
	input types.OrganizeMemoryInput,
) (*types.OrganizeMemory, error) {
	if err := validateOrganizeScope(tenantID, userID); err != nil {
		return nil, err
	}
	if s.fileService == nil && s.storageResolver == nil {
		return nil, fmt.Errorf("file service is not configured")
	}
	if len(uploads) == 0 {
		return nil, fmt.Errorf("upload file is empty")
	}

	kind := normalizeMemoryKind(input.Kind)
	firstName := strings.TrimSpace(uploads[0].FileName)
	if firstName == "" {
		firstName = "导入文件"
	}
	if kind == "" {
		if organizeUploadIsAudioVisual(firstName, uploads[0].MimeType) {
			kind = types.OrganizeMemoryKindAudio
		} else {
			kind = types.OrganizeMemoryKindNote
		}
	}
	if !types.IsValidOrganizeMemoryKind(kind) {
		return nil, ErrOrganizeInvalidMemoryKind
	}

	prepared := make([]organizeMemoryUploadPrepared, 0, len(uploads))
	memoryID := uuid.NewString()
	for index, upload := range uploads {
		item, err := s.prepareOrganizeMemoryUpload(ctx, tenantID, userID, memoryID, index, upload)
		if err != nil {
			for _, previous := range prepared {
				_ = previous.fileService.DeleteFile(ctx, previous.storagePath)
				s.releaseOrganizeMemoryStorage(ctx, tenantID, previous.storageRef, "memory_upload_failed")
			}
			return nil, err
		}
		prepared = append(prepared, item)
	}

	baseTitle := strings.TrimSuffix(filepath.Base(prepared[0].storedName), filepath.Ext(prepared[0].storedName))
	if baseTitle == "" {
		baseTitle = "导入记忆"
	}
	title := trimMax(input.Title, organizeMaxTitleLength)
	if title == "" {
		title = trimMax(baseTitle, organizeMaxTitleLength)
	}
	if title == "" {
		title = "导入记忆"
	}

	occurredAt := time.Now().UTC()
	if input.OccurredAt != nil && !input.OccurredAt.IsZero() {
		occurredAt = input.OccurredAt.UTC()
	}
	source := trimMax(input.Source, organizeMaxShortText)
	if source == "" {
		source = "文件导入"
	}
	metadata := normalizeJSONMap(input.Metadata)
	if metadata == nil {
		metadata = types.JSONMap{}
	}
	metadata["attachment_count"] = len(prepared)
	metadata["attachment_status"] = types.OrganizeMemoryAttachmentAggregatePending
	metadata["attachment_completed_count"] = 0
	metadata["attachment_failed_count"] = 0
	metadata["transcription_status"] = "pending"
	if len(prepared) == 1 {
		first := prepared[0]
		metadata["storage_size_bytes"] = first.sizeBytes
		metadata["file_name"] = first.storedName
		metadata["file_path"] = first.storagePath
		metadata["file_url"] = first.storageURL
		metadata["audio_file_name"] = first.storedName
		metadata["audio_file_path"] = first.storagePath
		metadata["audio_url"] = first.storageURL
		metadata["audio_mime_type"] = first.mimeType
		metadata["audio_codec"] = memoryAudioCodec(first.storedName, nil)
		metadata["audio_size_bytes"] = first.sizeBytes
	}

	content := trimMax(input.Content, 0)
	if content == "" {
		content = "<p>文件已保存，等待解析。</p>"
	}
	memory := &types.OrganizeMemory{
		ID:              memoryID,
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
		for _, item := range prepared {
			_ = item.fileService.DeleteFile(ctx, item.storagePath)
			s.releaseOrganizeMemoryStorage(ctx, tenantID, item.storageRef, "memory_upload_failed")
		}
		return nil, err
	}

	created := make([]*types.OrganizeMemoryAttachment, 0, len(prepared))
	for _, item := range prepared {
		attachment := &types.OrganizeMemoryAttachment{
			ID:          item.attachmentID,
			TenantID:    tenantID,
			UserID:      userID,
			MemoryID:    memory.ID,
			FileName:    item.storedName,
			MimeType:    item.mimeType,
			StoragePath: item.storagePath,
			StorageURL:  item.storageURL,
			SizeBytes:   item.sizeBytes,
			SortOrder:   item.sortOrder,
			Status:      types.OrganizeMemoryAttachmentStatusPending,
			Metadata: types.JSONMap{
				"extension":   strings.TrimPrefix(strings.ToLower(filepath.Ext(item.storedName)), "."),
				"source_kind": organizeUploadSourceKind(item.storedName, item.mimeType),
			},
		}
		if err := s.repo.CreateMemoryAttachment(ctx, attachment); err != nil {
			_ = s.repo.DeleteMemory(ctx, tenantID, userID, memory.ID)
			for _, previous := range prepared {
				_ = previous.fileService.DeleteFile(ctx, previous.storagePath)
				s.releaseOrganizeMemoryStorage(ctx, tenantID, previous.storageRef, "memory_upload_failed")
			}
			return nil, err
		}
		created = append(created, attachment)
		if s.resourceCatalog != nil {
			if _, ok := types.ParseResourcePath(item.storagePath); ok {
				if err := s.resourceCatalog.Bind(ctx, item.storagePath, "memory", memory.ID, "source_file"); err != nil {
					_ = s.repo.DeleteMemory(ctx, tenantID, userID, memory.ID)
					for _, previous := range prepared {
						_ = previous.fileService.DeleteFile(ctx, previous.storagePath)
						s.releaseOrganizeMemoryStorage(ctx, tenantID, previous.storageRef, "memory_upload_failed")
					}
					return nil, fmt.Errorf("bind memory resource: %w", err)
				}
			}
		}
		if err := s.commitOrganizeMemoryStorage(
			ctx,
			tenantID,
			item.storageRef,
			item.sizeBytes,
			memory.ID,
			item.storedName,
		); err != nil {
			return nil, err
		}
	}

	for _, attachment := range created {
		if err := s.scheduleMemoryTranscription(ctx, tenantID, memory.ID, attachment.ID); err != nil {
			attachment.Status = types.OrganizeMemoryAttachmentStatusFailed
			attachment.ErrorStage = "queue"
			attachment.ErrorMessage = trimMax(err.Error(), organizeMaxShortText)
			_ = s.repo.UpdateMemoryAttachment(ctx, attachment)
		}
	}
	return s.GetMemory(ctx, tenantID, userID, memory.ID)
}

type organizeMemoryUploadPrepared struct {
	attachmentID string
	storedName   string
	mimeType     string
	storagePath  string
	storageURL   string
	storageRef   string
	sizeBytes    int64
	sortOrder    int
	fileService  interface {
		SaveBytes(context.Context, []byte, uint64, string, bool) (string, error)
		DeleteFile(context.Context, string) error
	}
}

func (s *organizeService) prepareOrganizeMemoryUpload(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID string,
	sortOrder int,
	upload types.OrganizeMemoryUpload,
) (organizeMemoryUploadPrepared, error) {
	_ = userID
	cleanName := strings.TrimSpace(upload.FileName)
	if cleanName == "" {
		return organizeMemoryUploadPrepared{}, fmt.Errorf("file name is required")
	}
	if !isValidFileType(cleanName) {
		return organizeMemoryUploadPrepared{}, fmt.Errorf("unsupported file type: %s", strings.ToLower(filepath.Ext(cleanName)))
	}
	if len(upload.Data) == 0 {
		return organizeMemoryUploadPrepared{}, fmt.Errorf("upload file is empty")
	}
	safeName, valid := secutils.ValidateInput(cleanName)
	if !valid {
		return organizeMemoryUploadPrepared{}, fmt.Errorf("invalid characters in file name")
	}
	baseName, err := secutils.SafeFileName(safeName)
	if err != nil {
		return organizeMemoryUploadPrepared{}, fmt.Errorf("unsafe file name: %w", err)
	}

	storedBytes := upload.Data
	storedName := baseName
	if organizeUploadIsAudioVisual(cleanName, upload.MimeType) && !organizeUploadIsVideo(cleanName, upload.MimeType) {
		storedBytes, storedName, err = s.normalizeOrganizeAudioForStorage(ctx, upload.Data, baseName)
		if err != nil {
			return organizeMemoryUploadPrepared{}, fmt.Errorf("normalize audio for storage: %w", err)
		}
	}
	if storedName == "" {
		storedName = baseName
	}
	fileService, err := s.resolveOrganizeFileService(ctx, tenantID, "")
	if err != nil {
		return organizeMemoryUploadPrepared{}, err
	}
	attachmentID := uuid.NewString()
	storageRef := fmt.Sprintf("organize:memory_attachment_upload:%s", attachmentID)
	if err := s.reserveOrganizeMemoryStorage(
		ctx,
		tenantID,
		storageRef,
		memoryID,
		storedName,
		int64(len(storedBytes)),
	); err != nil {
		return organizeMemoryUploadPrepared{}, err
	}
	storageName := fmt.Sprintf("organize_memory_%s%s", uuid.NewString()[:12], filepath.Ext(storedName))
	storagePath, err := fileService.SaveBytes(ctx, storedBytes, tenantID, storageName, false)
	if err != nil {
		s.releaseOrganizeMemoryStorage(ctx, tenantID, storageRef, "memory_upload_failed")
		return organizeMemoryUploadPrepared{}, fmt.Errorf("save memory file: %w", err)
	}
	storageURL := resolveOrganizeMemoryAudioURL(ctx, fileService, storagePath)
	return organizeMemoryUploadPrepared{
		attachmentID: attachmentID,
		storedName:   storedName,
		mimeType:     organizeUploadMimeType(storedName, upload.MimeType),
		storagePath:  storagePath,
		storageURL:   storageURL,
		storageRef:   storageRef,
		sizeBytes:    int64(len(storedBytes)),
		sortOrder:    sortOrder,
		fileService:  fileService,
	}, nil
}

func (s *organizeService) processMemoryAttachment(
	ctx context.Context,
	payload types.OrganizeMemoryTranscribeTaskPayload,
) error {
	attachment, err := s.repo.GetTenantMemoryAttachment(ctx, payload.TenantID, strings.TrimSpace(payload.AttachmentID))
	if err != nil {
		return err
	}
	if attachment == nil {
		return nil
	}
	if attachment.Status == types.OrganizeMemoryAttachmentStatusCompleted {
		return nil
	}
	memory, err := s.repo.GetTenantMemory(ctx, payload.TenantID, strings.TrimSpace(payload.MemoryID))
	if err != nil {
		return err
	}
	if memory == nil || memory.ID != attachment.MemoryID {
		return nil
	}

	attachment.Status = types.OrganizeMemoryAttachmentStatusProcessing
	attachment.ErrorStage = ""
	attachment.ErrorMessage = ""
	attachment.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMemoryAttachment(ctx, attachment); err != nil {
		return err
	}
	_ = s.refreshOrganizeMemoryAttachmentSummary(ctx, memory)

	filePath := strings.TrimSpace(attachment.StoragePath)
	if filePath == "" {
		return s.failMemoryAttachment(ctx, memory, attachment, "storage", "source file path missing", nil)
	}
	fileService, err := s.resolveOrganizeFileService(ctx, payload.TenantID, filePath)
	if err != nil {
		return s.failMemoryAttachment(ctx, memory, attachment, "storage", "failed to resolve source storage", err)
	}
	file, err := fileService.GetFile(ctx, filePath)
	if err != nil {
		return s.failMemoryAttachment(ctx, memory, attachment, "storage", "failed to open source file", err)
	}
	data, readErr := io.ReadAll(file)
	_ = file.Close()
	if readErr != nil {
		return s.failMemoryAttachment(ctx, memory, attachment, "storage", "failed to read source file", readErr)
	}
	if len(data) == 0 {
		return s.failMemoryAttachment(ctx, memory, attachment, "storage", "source file is empty", nil)
	}

	content, transcript, asrModelID, parseErr := s.parseOrganizeMemoryAttachment(
		ctx,
		memory.Title,
		attachment.FileName,
		attachment.MimeType,
		data,
	)
	if parseErr != nil {
		return s.failMemoryAttachment(ctx, memory, attachment, "parse", "failed to parse source file", parseErr)
	}
	if strings.TrimSpace(content) == "" {
		return s.failMemoryAttachment(ctx, memory, attachment, "parse", "source file produced no readable content", nil)
	}

	attachment.Content = trimMax(content, 0)
	attachment.Transcript = trimMax(transcript, 0)
	attachment.Status = types.OrganizeMemoryAttachmentStatusCompleted
	attachment.Metadata = normalizeJSONMap(attachment.Metadata)
	attachment.Metadata["parser_status"] = "completed"
	if asrModelID != "" {
		attachment.Metadata["asr_model_id"] = asrModelID
	}
	attachment.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMemoryAttachment(ctx, attachment); err != nil {
		return err
	}
	return s.refreshOrganizeMemoryAttachmentSummary(ctx, memory)
}

func (s *organizeService) RetryMemoryAttachment(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID, attachmentID string,
) (*types.OrganizeMemory, error) {
	memory, err := s.GetMemory(ctx, tenantID, userID, memoryID)
	if err != nil {
		return nil, err
	}
	attachment, err := s.repo.GetMemoryAttachment(ctx, tenantID, userID, strings.TrimSpace(attachmentID))
	if err != nil {
		return nil, err
	}
	if attachment == nil || attachment.MemoryID != memory.ID {
		return nil, ErrOrganizeNotFound
	}
	if attachment.Status == types.OrganizeMemoryAttachmentStatusProcessing {
		return nil, ErrOrganizeMemoryNotReady
	}
	if attachment.Status == types.OrganizeMemoryAttachmentStatusCompleted {
		return memory, nil
	}
	attachment.Status = types.OrganizeMemoryAttachmentStatusPending
	attachment.ErrorStage = ""
	attachment.ErrorMessage = ""
	attachment.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateMemoryAttachment(ctx, attachment); err != nil {
		return nil, err
	}
	if err := s.scheduleMemoryTranscription(ctx, tenantID, memory.ID, attachment.ID); err != nil {
		attachment.Status = types.OrganizeMemoryAttachmentStatusFailed
		attachment.ErrorStage = "queue"
		attachment.ErrorMessage = trimMax(err.Error(), organizeMaxShortText)
		_ = s.repo.UpdateMemoryAttachment(ctx, attachment)
		return nil, err
	}
	_ = s.refreshOrganizeMemoryAttachmentSummary(ctx, memory)
	return s.GetMemory(ctx, tenantID, userID, memory.ID)
}

func (s *organizeService) parseOrganizeMemoryAttachment(
	ctx context.Context,
	title, fileName, mimeType string,
	data []byte,
) (content, transcript, asrModelID string, err error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	if shouldUseRawTextForOrganizeUpload(ext) {
		return trimMax(string(data), 0), "", "", nil
	}

	if s.documentReader != nil {
		result, readErr := s.documentReader.Read(ctx, &types.ReadRequest{
			FileContent: data,
			FileName:    fileName,
			FileType:    ext,
			MimeType:    mimeType,
			Title:       title,
		})
		if readErr != nil {
			return "", "", "", readErr
		}
		if result != nil {
			content = trimMax(result.MarkdownContent, 0)
			if result.IsAudio && len(result.AudioData) > 0 {
				transcript, asrModelID, err = s.transcribeOrganizeUploadAudio(ctx, fileName, result.AudioData)
				if err != nil {
					return "", "", asrModelID, err
				}
				content = transcript
			}
			if content != "" {
				return content, transcript, asrModelID, nil
			}
		}
	}

	if organizeUploadIsAudioVisual(fileName, mimeType) {
		transcript, asrModelID, err = s.transcribeOrganizeUploadAudio(ctx, fileName, data)
		if err != nil {
			return "", "", asrModelID, err
		}
		if transcript != "" {
			return transcript, transcript, asrModelID, nil
		}
	}
	if shouldUseRawTextForOrganizeUpload(ext) {
		return trimMax(string(data), 0), "", "", nil
	}
	return trimMax(fileName, 0), "", "", nil
}

func (s *organizeService) failMemoryAttachment(
	ctx context.Context,
	memory *types.OrganizeMemory,
	attachment *types.OrganizeMemoryAttachment,
	stage, message string,
	cause error,
) error {
	attachment.Status = types.OrganizeMemoryAttachmentStatusFailed
	attachment.ErrorStage = trimMax(stage, 64)
	attachment.ErrorMessage = trimMax(message, organizeMaxShortText)
	attachment.UpdatedAt = time.Now().UTC()
	_ = s.repo.UpdateMemoryAttachment(ctx, attachment)
	_ = s.refreshOrganizeMemoryAttachmentSummary(ctx, memory)
	if cause != nil {
		return cause
	}
	return fmt.Errorf("%s", message)
}

func (s *organizeService) refreshOrganizeMemoryAttachmentSummary(
	ctx context.Context,
	memory *types.OrganizeMemory,
) error {
	if memory == nil {
		return nil
	}
	attachments, err := s.repo.ListMemoryAttachments(ctx, memory.TenantID, memory.UserID, memory.ID)
	if err != nil {
		return err
	}
	if len(attachments) == 0 {
		return nil
	}
	pending, processing, completed, failed, skipped := 0, 0, 0, 0, 0
	transcripts := make([]string, 0, len(attachments))
	contents := make([]string, 0, len(attachments))
	for _, attachment := range attachments {
		switch attachment.Status {
		case types.OrganizeMemoryAttachmentStatusPending:
			pending++
		case types.OrganizeMemoryAttachmentStatusProcessing:
			processing++
		case types.OrganizeMemoryAttachmentStatusCompleted:
			completed++
		case types.OrganizeMemoryAttachmentStatusFailed:
			failed++
		case types.OrganizeMemoryAttachmentStatusSkipped:
			skipped++
		}
		if strings.TrimSpace(attachment.Transcript) != "" {
			transcripts = append(transcripts, formatOrganizeAttachmentSection(attachment.FileName, attachment.Transcript, len(attachments) > 1))
		}
		if strings.TrimSpace(attachment.Content) != "" && strings.TrimSpace(attachment.Transcript) == "" {
			contents = append(contents, formatOrganizeAttachmentSection(attachment.FileName, attachment.Content, len(attachments) > 1))
		}
	}
	status := types.OrganizeMemoryAttachmentAggregatePending
	switch {
	case processing > 0:
		status = types.OrganizeMemoryAttachmentAggregateProcessing
	case completed > 0 && (failed > 0 || skipped > 0):
		status = types.OrganizeMemoryAttachmentAggregatePartial
	case completed == len(attachments):
		status = types.OrganizeMemoryAttachmentAggregateCompleted
	case failed+skipped == len(attachments):
		status = types.OrganizeMemoryAttachmentAggregateFailed
	}

	metadata := normalizeJSONMap(memory.Metadata)
	metadata["attachment_count"] = len(attachments)
	metadata["attachment_status"] = status
	metadata["attachment_pending_count"] = pending
	metadata["attachment_processing_count"] = processing
	metadata["attachment_completed_count"] = completed
	metadata["attachment_failed_count"] = failed
	metadata["attachment_skipped_count"] = skipped
	if len(transcripts) > 0 {
		combinedTranscript := strings.TrimSpace(strings.Join(transcripts, "\n\n"))
		metadata["transcript"] = combinedTranscript
		metadata["raw_transcript"] = combinedTranscript
	}
	switch status {
	case types.OrganizeMemoryAttachmentAggregateCompleted:
		metadata["transcription_status"] = "completed"
	case types.OrganizeMemoryAttachmentAggregatePartial:
		metadata["transcription_status"] = "partial"
	case types.OrganizeMemoryAttachmentAggregateFailed:
		metadata["transcription_status"] = "failed"
	default:
		metadata["transcription_status"] = "transcribing"
	}
	memory.Metadata = metadata
	userEdited := organizeMetadataBool(memory.Metadata, "note_user_edited")
	if len(transcripts) > 0 {
		allText := strings.TrimSpace(strings.Join(transcripts, "\n\n"))
		if !userEdited && (status == types.OrganizeMemoryAttachmentAggregateCompleted ||
			status == types.OrganizeMemoryAttachmentAggregatePartial) {
			aiResult, aiModelID, aiStatus := s.generateMemoryRecordingNoteAIResult(
				ctx,
				memory.Title,
				attachmentDisplayName(attachments),
				memory.Source,
				allText,
			)
			noteMarkdown := organizeRecordingNoteMarkdown(aiResult.Summary, allText, aiResult.NoteMarkdown)
			if title := trimMax(aiResult.Title, organizeMaxTitleLength); title != "" {
				memory.Title = title
			}
			memory.Content = organizeUploadContentToNoteHTML(memory.Title, noteMarkdown)
			memory.Metadata["summary"] = trimMax(aiResult.Summary, organizeMaxShortText)
			memory.Metadata["tags"] = types.StringArray(aiResult.Tags)
			memory.Metadata["note_generation_status"] = aiStatus
			memory.Metadata["note_source"] = "generated"
			if aiModelID != "" {
				memory.Metadata["ai_model_id"] = aiModelID
			}
		} else if userEdited {
			memory.Metadata["note_source"] = "user"
		}
	} else if len(contents) > 0 && !userEdited {
		memory.Content = organizeUploadContentToNoteHTML(memory.Title, strings.Join(contents, "\n\n"))
		memory.Metadata["note_source"] = "parsed"
	}
	memory.UpdatedAt = time.Now().UTC()
	return s.repo.UpdateMemory(ctx, memory)
}

func formatOrganizeAttachmentSection(fileName, content string, includeFileName bool) string {
	content = strings.TrimSpace(content)
	if !includeFileName {
		return content
	}
	return fmt.Sprintf("## %s\n\n%s", strings.TrimSpace(fileName), content)
}

func attachmentDisplayName(attachments []*types.OrganizeMemoryAttachment) string {
	if len(attachments) == 0 {
		return "导入文件"
	}
	if len(attachments) == 1 {
		return attachments[0].FileName
	}
	return fmt.Sprintf("%s 等 %d 个文件", attachments[0].FileName, len(attachments))
}

func organizeUploadIsAudioVisual(fileName, mimeType string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	return IsAudioType(ext) || IsVideoType(ext) ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])), "audio/") ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])), "video/")
}

func organizeUploadIsVideo(fileName, mimeType string) bool {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	return IsVideoType(ext) ||
		strings.HasPrefix(strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0])), "video/")
}

func organizeUploadSourceKind(fileName, mimeType string) string {
	switch {
	case organizeUploadIsVideo(fileName, mimeType):
		return "video"
	case organizeUploadIsAudioVisual(fileName, mimeType):
		return "audio"
	default:
		return "document"
	}
}

func organizeUploadMimeType(fileName, fallback string) string {
	if organizeUploadIsAudioVisual(fileName, fallback) {
		return organizeAudioMimeType(fileName, fallback)
	}
	fallback = strings.TrimSpace(strings.Split(fallback, ";")[0])
	if fallback != "" {
		return fallback
	}
	return "application/octet-stream"
}

func organizeMetadataBool(metadata types.JSONMap, key string) bool {
	switch value := metadata[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}
