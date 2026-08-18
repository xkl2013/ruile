package types

import "testing"

func TestParserEngineConfigResolveChatParserEngine(t *testing.T) {
	config := &ParserEngineConfig{ChatParserEngineRules: []ParserEngineRule{
		{FileTypes: []string{"pdf", ".docx"}, Engine: "mineru"},
		{FileTypes: []string{"xlsx"}, Engine: "markitdown"},
	}}
	for input, expected := range map[string]string{
		"PDF": "mineru", ".docx": "mineru", "xlsx": "markitdown", "txt": "",
	} {
		if actual := config.ResolveChatParserEngine(input); actual != expected {
			t.Fatalf("ResolveChatParserEngine(%q) = %q, want %q", input, actual, expected)
		}
	}
	var nilConfig *ParserEngineConfig
	if actual := nilConfig.ResolveChatParserEngine("pdf"); actual != "" {
		t.Fatalf("nil config resolved %q", actual)
	}
}

func TestChunkingConfigResolveParserEngineDefaultsPowerPointToMarkitdown(t *testing.T) {
	for _, tt := range []struct {
		name     string
		config   ChunkingConfig
		fileType string
		expected string
	}{
		{
			name:     "pptx defaults to markitdown",
			fileType: "pptx",
			expected: "markitdown",
		},
		{
			name:     "legacy ppt normalizes dotted uppercase extension",
			fileType: ".PPT",
			expected: "markitdown",
		},
		{
			name: "unrelated rules keep pptx default",
			config: ChunkingConfig{ParserEngineRules: []ParserEngineRule{
				{FileTypes: []string{"pdf"}, Engine: "builtin"},
			}},
			fileType: "pptx",
			expected: "markitdown",
		},
		{
			name: "explicit rule wins",
			config: ChunkingConfig{ParserEngineRules: []ParserEngineRule{
				{FileTypes: []string{".PPTX"}, Engine: "mineru"},
			}},
			fileType: "pptx",
			expected: "mineru",
		},
		{
			name:     "other types keep builtin sentinel",
			fileType: "pdf",
			expected: "",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if actual := tt.config.ResolveParserEngine(tt.fileType); actual != tt.expected {
				t.Fatalf("ResolveParserEngine(%q) = %q, want %q", tt.fileType, actual, tt.expected)
			}
		})
	}
}
