package service

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func (s *organizeService) GetMemoryAudioURL(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID string,
) (string, string, string, error) {
	memory, err := s.GetMemory(ctx, tenantID, userID, memoryID)
	if err != nil {
		return "", "", "", err
	}

	var attachmentErr error
	for _, attachment := range memory.Attachments {
		if attachment == nil || !isOrganizeAudioSource(attachment.FileName, attachment.MimeType) {
			continue
		}
		fileURL, err := s.organizePreviewURL(
			ctx,
			tenantID,
			attachment.StoragePath,
		)
		if err != nil {
			attachmentErr = err
			continue
		}
		return fileURL, attachment.FileName, attachment.MimeType, nil
	}

	filePath := organizeStoredFilePath(
		memory.Metadata,
		"audio_file_path",
		"file_path",
		"storage_path",
	)
	fileURL, err := s.organizePreviewURL(
		ctx,
		tenantID,
		filePath,
	)
	if err != nil {
		if attachmentErr != nil {
			return "", "", "", fmt.Errorf(
				"generate memory audio preview URL: attachment: %v; metadata: %w",
				attachmentErr,
				err,
			)
		}
		return "", "", "", err
	}
	fileName := stringValue(memory.Metadata, "audio_file_name")
	if fileName == "" {
		fileName = stringValue(memory.Metadata, "file_name")
	}
	mimeType := stringValue(memory.Metadata, "audio_mime_type")
	if mimeType == "" {
		mimeType = stringValue(memory.Metadata, "mime_type")
	}
	return fileURL, fileName, mimeType, nil
}

func (s *organizeService) GetMemoryAttachmentPreviewURL(
	ctx context.Context,
	tenantID uint64,
	userID, memoryID, attachmentID string,
) (string, string, string, error) {
	memory, err := s.GetMemory(ctx, tenantID, userID, memoryID)
	if err != nil {
		return "", "", "", err
	}
	attachment, err := s.repo.GetMemoryAttachment(
		ctx,
		tenantID,
		userID,
		strings.TrimSpace(attachmentID),
	)
	if err != nil {
		return "", "", "", err
	}
	if attachment == nil || attachment.MemoryID != memory.ID {
		return "", "", "", ErrOrganizeNotFound
	}

	fileURL, err := s.organizePreviewURL(
		ctx,
		tenantID,
		attachment.StoragePath,
	)
	if err != nil {
		return "", "", "", err
	}
	return fileURL, attachment.FileName, attachment.MimeType, nil
}

func (s *organizeService) organizePreviewURL(
	ctx context.Context,
	tenantID uint64,
	storagePath string,
) (string, error) {
	storagePath = strings.TrimSpace(storagePath)
	if storagePath == "" {
		return "", fmt.Errorf("stable storage path is unavailable")
	}

	storageCtx := context.WithValue(ctx, types.TenantIDContextKey, tenantID)
	fileService, err := s.resolveOrganizeFileService(storageCtx, tenantID, storagePath)
	if err != nil {
		return "", err
	}
	var fileURL string
	if direct, ok := fileService.(interfaces.DirectFileURLService); ok {
		fileURL, err = direct.GetDirectFileURL(storageCtx, storagePath)
	} else {
		fileURL, err = fileService.GetFileURL(storageCtx, storagePath)
	}
	if err != nil {
		return "", err
	}

	fileURL = strings.TrimSpace(fileURL)
	if !isDirectOrganizePreviewURL(fileURL) {
		return "", fmt.Errorf("storage provider did not return a direct preview URL")
	}
	return fileURL, nil
}

func isDirectOrganizePreviewURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return false
	}
	normalizedPath := "/" + strings.TrimPrefix(path.Clean(parsed.Path), "/")
	if normalizedPath == "/files" ||
		normalizedPath == "/r" ||
		strings.HasPrefix(normalizedPath, "/r/") ||
		normalizedPath == "/api/v1/files" ||
		normalizedPath == "/api/v1/files/presigned" ||
		(strings.HasPrefix(normalizedPath, "/api/v1/knowledge/") &&
			strings.HasSuffix(normalizedPath, "/preview")) ||
		(strings.HasPrefix(normalizedPath, "/api/v1/organize/") &&
			strings.HasSuffix(normalizedPath, "/media")) {
		return false
	}
	return true
}

func isOrganizeAudioSource(fileName, mimeType string) bool {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(mimeType)), "audio/") {
		return true
	}
	switch strings.ToLower(path.Ext(strings.TrimSpace(fileName))) {
	case ".mp3", ".wav", ".m4a", ".aac", ".flac", ".ogg", ".opus":
		return true
	default:
		return false
	}
}
