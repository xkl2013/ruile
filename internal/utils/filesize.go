package utils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const audioMaxFileSizeMB int64 = 100

var audioFileTypes = map[string]struct{}{
	"mp3":  {},
	"wav":  {},
	"m4a":  {},
	"flac": {},
	"ogg":  {},
	"aac":  {},
	"sbc":  {},
}

// GetMaxFileSize returns the default non-audio file upload size in bytes.
// Default is 50MB, can be configured via MAX_FILE_SIZE_MB environment variable.
//
// MAX_FILE_SIZE_MB is intentionally a deploy-time-only knob (NOT a
// runtime system_setting). The effective upload limit is gated by
// three other layers that all read this env at startup and cache the
// value or a larger request cap:
//   - frontend nginx client_max_body_size (envsubst from UPLOAD_REQUEST_MAX_FILE_SIZE_MB)
//   - docreader gRPC max_send/recv_message_length
//   - frontend client-side check via window.__RUNTIME_CONFIG__
//
// Surfacing a SystemAdmin UI knob whose effect is silently capped by
// any of the above would mislead operators ("I raised it to 200MB but
// nginx still returns 413"). Until all four layers can be reconfigured
// in lockstep without container restarts, every call site must read
// the env directly via this helper.
func GetMaxFileSize() int64 {
	if sizeStr := os.Getenv("MAX_FILE_SIZE_MB"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			return size * 1024 * 1024
		}
	}
	return 50 * 1024 * 1024 // default 50MB
}

// GetMaxFileSizeMB returns the default non-audio file upload size in MB. Same
// caveat as GetMaxFileSize — handlers should prefer SystemSettingService.GetInt.
func GetMaxFileSizeMB() int64 {
	if sizeStr := os.Getenv("MAX_FILE_SIZE_MB"); sizeStr != "" {
		if size, err := strconv.ParseInt(sizeStr, 10, 64); err == nil && size > 0 {
			return size
		}
	}
	return 50 // default 50MB
}

func GetAudioMaxFileSizeMB() int64 {
	return audioMaxFileSizeMB
}

func IsAudioFileType(fileType string) bool {
	normalized := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(fileType)), ".")
	_, ok := audioFileTypes[normalized]
	return ok
}

func IsAudioContentType(contentType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return strings.HasPrefix(normalized, "audio/")
}

func IsAudioUpload(fileName, contentType string) bool {
	return IsAudioFileType(filepath.Ext(fileName)) || IsAudioContentType(contentType)
}

func GetMaxFileSizeMBForFileType(fileType string) int64 {
	if IsAudioFileType(fileType) {
		return audioMaxFileSizeMB
	}
	return GetMaxFileSizeMB()
}

func GetMaxFileSizeMBForFileName(fileName string) int64 {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileName)), ".")
	return GetMaxFileSizeMBForFileType(ext)
}

func GetMaxFileSizeMBForUpload(fileName, contentType string) int64 {
	if IsAudioUpload(fileName, contentType) {
		return audioMaxFileSizeMB
	}
	return GetMaxFileSizeMB()
}
