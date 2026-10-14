package session

import "testing"

func TestIsValidFileTypeAcceptsVideoAttachments(t *testing.T) {
	for _, fileName := range []string{
		"clip.mp4",
		"clip.MOV",
		"clip.m4v",
		"clip.avi",
		"clip.mkv",
		"clip.webm",
		"clip.wmv",
		"clip.flv",
	} {
		if !isValidFileType(fileName) {
			t.Errorf("isValidFileType(%q) = false, want true", fileName)
		}
	}
}
