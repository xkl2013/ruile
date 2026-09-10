package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestResolveLocalSourcePathUsesLegacyBackendPrefix(t *testing.T) {
	root := t.TempDir()
	legacyRoot := filepath.Join(root, "workspace-a")
	source := filepath.Join(legacyRoot, "10000", "kb-1", "123.pdf")
	require.NoError(t, os.MkdirAll(filepath.Dir(source), 0o755))
	require.NoError(t, os.WriteFile(source, []byte("pdf"), 0o644))

	backends := map[string]*types.StorageBackend{
		"local-backend": {
			ID:       "local-backend",
			Provider: "local",
			Config:   types.StorageBackendConfig{PathPrefix: "workspace-a"},
		},
	}
	resource := types.StoredResource{
		TenantID:         10000,
		StorageBackendID: "local-backend",
		Provider:         "local",
		PhysicalPath:     "storage://local-backend/local://10000/kb-1/123.pdf",
	}

	gotPath, gotRelative, err := resolveLocalSourcePath(root, backends, resource)
	require.NoError(t, err)
	require.Equal(t, source, gotPath)
	require.Equal(t, "10000/kb-1/123.pdf", gotRelative)
}

func TestBuildObjectKeyPreservesLocalLayout(t *testing.T) {
	got, err := buildObjectKey("weknora/", "10000/kb-1/123.pdf")
	require.NoError(t, err)
	require.Equal(t, "weknora/10000/kb-1/123.pdf", got)
}

func TestBuildObjectKeyRejectsTraversal(t *testing.T) {
	_, err := buildObjectKey("weknora/", "../secret.pdf")
	require.Error(t, err)
}
