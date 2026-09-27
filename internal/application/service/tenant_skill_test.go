package service

import (
	"archive/zip"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInspectTenantSkillArchive(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md":       "---\nname: demo\ndescription: A demo skill\n---\n\nUse the demo skill.",
		"demo/scripts/run.py": "print('ok')",
	})

	metadata, root, err := inspectTenantSkillArchive(archive)
	require.NoError(t, err)
	require.Equal(t, "demo", metadata.Name)
	require.Equal(t, "A demo skill", metadata.Description)
	require.Equal(t, "demo", root)
}

func TestInspectTenantSkillArchiveRejectsTraversal(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"../SKILL.md": "---\nname: demo\ndescription: invalid\n---\n",
	})

	_, _, err := inspectTenantSkillArchive(archive)
	require.Error(t, err)
}

func TestExtractTenantSkillArchiveProducesLoaderCompatibleDirectory(t *testing.T) {
	archive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md":  "---\nname: demo\ndescription: A demo skill\n---\n\nUse the demo skill.",
		"demo/README.md": "read me",
	})
	_, root, err := inspectTenantSkillArchive(archive)
	require.NoError(t, err)

	baseDir := t.TempDir()
	require.NoError(t, extractTenantSkillArchive(archive, baseDir, "demo", root))

	skillRoot := filepath.Join(baseDir, "demo")
	loader := skills.NewLoader([]string{baseDir})
	metadata, err := loader.DiscoverSkills()
	require.NoError(t, err)
	require.Len(t, metadata, 1)
	require.Equal(t, "demo", metadata[0].Name)

	content, err := os.ReadFile(filepath.Join(skillRoot, "README.md"))
	require.NoError(t, err)
	require.Equal(t, "read me", string(content))
}

func TestTenantSkillRepositoryPersistsBundleSHA256Column(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}))

	repo := repository.NewTenantSkillRepository(db)
	skill := &types.TenantSkill{
		TenantID:   0,
		CreatedBy:  "system-admin",
		Name:       "pdf-processing",
		BundlePath: "resource://bundle",
		BundleSHA:  "0123456789abcdef",
	}
	require.NoError(t, repo.Create(context.Background(), skill))

	var stored struct {
		BundleSHA256 string `gorm:"column:bundle_sha256"`
	}
	require.NoError(t, db.Table("tenant_skills").Select("bundle_sha256").Where("id = ?", skill.ID).Scan(&stored).Error)
	require.Equal(t, skill.BundleSHA, stored.BundleSHA256)
}

func buildSkillArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for name, content := range files {
		entry, err := writer.Create(name)
		require.NoError(t, err)
		_, err = entry.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return output.Bytes()
}
