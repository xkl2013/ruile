package config

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestPromptTemplatesUseRuileDeveloperIdentity(t *testing.T) {
	promptTemplates, err := loadPromptTemplates(filepath.Join("..", "..", "config"))
	if err != nil {
		t.Fatalf("load prompt templates: %v", err)
	}
	if promptTemplates == nil {
		t.Fatal("prompt templates were not loaded")
	}

	developerAttribution := regexp.MustCompile(`(?i)(?:developed|created|built)\s+by\s+([^\s,.;]+)`)
	templateGroups := [][]PromptTemplate{
		promptTemplates.SystemPrompt,
		promptTemplates.ContextTemplate,
		promptTemplates.Rewrite,
		promptTemplates.Fallback,
		promptTemplates.GenerateSessionTitle,
		promptTemplates.GenerateSummary,
		promptTemplates.KeywordsExtraction,
		promptTemplates.AgentSystemPrompt,
		promptTemplates.GraphExtraction,
		promptTemplates.GenerateQuestions,
		promptTemplates.IntentPrompts,
	}

	for _, templates := range templateGroups {
		for _, template := range templates {
			for fieldName, content := range map[string]string{
				"content": template.Content,
				"user":    template.User,
			} {
				for _, match := range developerAttribution.FindAllStringSubmatch(content, -1) {
					if len(match) < 2 || strings.EqualFold(match[1], "睿乐") {
						continue
					}
					t.Errorf(
						"prompt template %q %s attributes the product to %q; use 睿乐",
						template.ID,
						fieldName,
						match[1],
					)
				}
			}
		}
	}
}
