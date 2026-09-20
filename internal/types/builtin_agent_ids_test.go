package types

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var removedBuiltinAgentIDs = []string{
	"builtin-wiki-researcher",
	"builtin-data-analyst",
}

func TestBuiltinAgentIDsExcludeRemovedBuiltins(t *testing.T) {
	for _, removedID := range removedBuiltinAgentIDs {
		for _, id := range GetBuiltinAgentIDs() {
			if id == removedID {
				t.Fatalf("builtin agent list still contains %q", removedID)
			}
		}
	}
}

func TestBuiltinAgentIDsExposeOnlyQuickAnswer(t *testing.T) {
	want := []string{BuiltinQuickAnswerID}
	if got := GetBuiltinAgentIDs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("user-facing builtin agents = %v, want %v", got, want)
	}
}

func TestBuiltinAgentsConfigDoesNotDefineRemovedBuiltins(t *testing.T) {
	path := filepath.Join("..", "..", "config", "builtin_agents.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read builtin agents config: %v", err)
	}
	content := string(data)
	for _, removedID := range removedBuiltinAgentIDs {
		if strings.Contains(content, removedID) {
			t.Fatalf("builtin agents config still defines %q", removedID)
		}
	}
}

func TestQuickAnswerDoesNotEnableWebSearchByDefault(t *testing.T) {
	path := filepath.Join("..", "..", "config", "builtin_agents.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read builtin agents config: %v", err)
	}

	var file builtinAgentsFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse builtin agents config: %v", err)
	}

	for _, entry := range file.BuiltinAgents {
		if entry.ID != BuiltinQuickAnswerID {
			continue
		}
		if entry.Config.WebSearchEnabled {
			t.Fatal("quick-answer web search must be disabled by default")
		}
		return
	}
	t.Fatalf("builtin agent %q not found", BuiltinQuickAnswerID)
}
