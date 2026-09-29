package database

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"testing"
)

func TestMigrationFilesHaveUniqueVersions(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve migration test location")
	}

	for _, name := range []string{"versioned", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			migrationsDir := filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", name)
			entries, err := os.ReadDir(migrationsDir)
			if err != nil {
				t.Fatalf("read migration directory: %v", err)
			}

			pattern := regexp.MustCompile(`^([0-9]+)_.+\.(up|down)\.sql$`)
			seen := make(map[string]string)
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}

				matches := pattern.FindStringSubmatch(entry.Name())
				if matches == nil {
					continue
				}

				key := fmt.Sprintf("%s.%s", matches[1], matches[2])
				if previous, exists := seen[key]; exists {
					t.Fatalf(
						"duplicate migration version %s: %s and %s",
						key,
						previous,
						entry.Name(),
					)
				}
				seen[key] = entry.Name()
			}
		})
	}
}
