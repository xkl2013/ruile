package utils

import "testing"

func TestSafeContentTypeByFilenameAudioFormats(t *testing.T) {
	tests := []struct {
		name       string
		wantType   string
		wantInline bool
	}{
		{name: "recording.mp3", wantType: "audio/mpeg", wantInline: true},
		{name: "recording.m4a", wantType: "audio/mp4", wantInline: true},
		{name: "recording.wav", wantType: "audio/wav", wantInline: true},
		{name: "recording.flac", wantType: "audio/flac", wantInline: true},
		{name: "recording.ogg", wantType: "audio/ogg", wantInline: true},
		{name: "recording.aac", wantType: "audio/aac", wantInline: true},
		{name: "recording.amr", wantType: "audio/amr", wantInline: true},
		{name: "recording.opus", wantType: "audio/opus", wantInline: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotInline := SafeContentTypeByFilename(tt.name)
			if gotType != tt.wantType {
				t.Fatalf("content type = %q, want %q", gotType, tt.wantType)
			}
			if gotInline != tt.wantInline {
				t.Fatalf("inline = %v, want %v", gotInline, tt.wantInline)
			}
		})
	}
}
