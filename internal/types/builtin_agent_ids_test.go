package types

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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
