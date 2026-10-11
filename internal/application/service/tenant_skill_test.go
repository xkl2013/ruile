package service

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/agent/skills"
	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/Tencent/WeKnora/internal/utils"
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

func TestTenantSkillEnvVarsValidateAndPersist(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}))

	repo := repository.NewTenantSkillRepository(db)
	skill := &types.TenantSkill{
		TenantID:   42,
		CreatedBy:  "admin-1",
		Name:       "env-demo",
		BundlePath: "resource://bundle",
	}
	require.NoError(t, repo.Create(context.Background(), skill))
	svc := NewTenantSkillService(repo, newTenantSkillFileStub()).(*tenantSkillService)

	require.NoError(t, svc.SetTenantSkillEnvVars(
		context.Background(),
		42,
		skill.ID,
		map[string]string{"API_TOKEN": "secret", "REGION": "cn"},
	))
	stored, err := repo.GetByID(context.Background(), 42, skill.ID)
	require.NoError(t, err)
	require.Equal(t, types.TenantSkillEnvVars{"API_TOKEN": "secret", "REGION": "cn"}, stored.EnvVars)

	for _, invalid := range []map[string]string{
		{"path": "blocked"},
		{"WEKNORA_SKILL_OUTPUT_DIR": "blocked"},
		{"API-TOKEN": "blocked"},
		{"API_TOKEN": string(make([]byte, maxTenantSkillEnvValueBytes+1))},
	} {
		err := svc.SetTenantSkillEnvVars(context.Background(), 42, skill.ID, invalid)
		require.Error(t, err)
	}
}

func TestParseTenantSkillSource(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		kind string
	}{
		{
			name: "github archive",
			raw:  "https://github.com/acme/demo/archive/refs/heads/main.zip",
			kind: types.TenantSkillSourceGitHub,
		},
		{
			name: "gitlab archive",
			raw:  "https://gitlab.com/acme/demo/-/archive/main/demo-main.zip",
			kind: types.TenantSkillSourceGitLab,
		},
		{
			name: "registry download",
			raw:  "https://clawhub.ai/api/v1/download?slug=demo",
			kind: types.TenantSkillSourceRegistry,
		},
		{
			name: "direct archive",
			raw:  "https://example.com/demo.zip",
			kind: types.TenantSkillSourceDirect,
		},
		{
			name: "clawhub shorthand",
			raw:  "@acme/demo@1.2.3",
			kind: types.TenantSkillSourceRegistry,
		},
		{
			name: "skills.sh shorthand",
			raw:  "skills-sh:acme/tools/demo",
			kind: types.TenantSkillSourceSkillsSh,
		},
		{
			name: "github tree",
			raw:  "https://github.com/acme/tools/tree/main/skills/demo",
			kind: types.TenantSkillSourceGitHub,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := parseTenantSkillSource(tt.raw)
			require.NoError(t, err)
			require.Equal(t, tt.kind, source.Kind)
			if tt.raw[0] == '@' || tt.raw[:len("skills-sh:")] == "skills-sh:" {
				require.NotEmpty(t, source.Ref)
			} else {
				require.Equal(t, tt.raw, source.Ref)
			}
		})
	}
}

func TestParseTenantSkillSourceRejectsUnsafeOrUnsupportedURLs(t *testing.T) {
	source, err := parseTenantSkillSource("https://github.com/acme/demo")
	require.NoError(t, err)
	require.Equal(t, types.TenantSkillSourceGitHub, source.Kind)
	require.Equal(t, "acme", source.Owner)
	require.Equal(t, "demo", source.Repo)

	_, err = parseTenantSkillSource("https://example.com/download.zip?access_token=secret")
	require.ErrorContains(t, err, "access credentials")

	_, err = parseTenantSkillSource("file:///tmp/demo.zip")
	require.ErrorContains(t, err, "http(s)")
}

func TestFetchTenantSkillSourceNormalizesRawSkillFile(t *testing.T) {
	t.Setenv("SSRF_WHITELIST", "127.0.0.1,::1,localhost")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/markdown")
		_, _ = io.WriteString(w, "---\nname: remote-demo\ndescription: Remote demo\n---\n\nRun it.")
	}))
	defer server.Close()

	sourceURL, err := url.Parse(server.URL + "/SKILL.md")
	require.NoError(t, err)
	data, err := fetchTenantSkillSource(context.Background(), tenantSkillSource{
		Kind: types.TenantSkillSourceDirect,
		Ref:  sourceURL.String(),
		URL:  sourceURL,
	}, server.Client())
	require.NoError(t, err)

	metadata, root, err := inspectTenantSkillArchive(data)
	require.NoError(t, err)
	require.Equal(t, "remote-demo", metadata.Name)
	require.Equal(t, ".", root)
}

func TestFetchTenantSkillSourceFollowsRegistryHandoff(t *testing.T) {
	t.Setenv("SSRF_WHITELIST", "127.0.0.1,::1,localhost")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)

	archive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md":  "---\nname: demo\ndescription: Handoff demo\n---\n\nRun it.",
		"demo/README.md": "handoff",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/download":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"ok":true,"archiveUrl":"/archive.zip"}`)
		case "/archive.zip":
			w.Header().Set("Content-Type", "application/zip")
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	data, err := fetchTenantSkillSource(context.Background(), tenantSkillSource{
		Kind:     tenantSkillSourceRegistry,
		Registry: server.URL,
		Slug:     "demo",
	}, server.Client())
	require.NoError(t, err)

	metadata, root, err := inspectTenantSkillArchive(data)
	require.NoError(t, err)
	require.Equal(t, "demo", metadata.Name)
	require.Equal(t, "demo", root)
}

func TestFetchTenantSkillSourceFiltersRepositorySubdirectory(t *testing.T) {
	t.Setenv("SSRF_WHITELIST", "127.0.0.1,::1,localhost")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)

	archive := buildSkillArchive(t, map[string]string{
		"repo/skills/demo/SKILL.md":    "---\nname: demo\ndescription: Nested demo\n---\n\nRun it.",
		"repo/skills/demo/config.json": `{"enabled":true}`,
		"repo/README.md":               "outside",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	sourceURL, err := url.Parse(server.URL + "/repo.zip")
	require.NoError(t, err)
	data, err := fetchTenantSkillSource(context.Background(), tenantSkillSource{
		Kind:      tenantSkillSourceDirect,
		Ref:       sourceURL.String(),
		URL:       sourceURL,
		DirectURL: sourceURL.String(),
		Subdir:    "repo/skills/demo",
	}, server.Client())
	require.NoError(t, err)

	metadata, root, err := inspectTenantSkillArchive(data)
	require.NoError(t, err)
	require.Equal(t, "demo", metadata.Name)
	require.Equal(t, ".", root)
}

func TestTenantSkillSourceInstallJobCompletesAndCreatesCatalogSkill(t *testing.T) {
	withTenantSkillSourceTestSSRFWhitelist(t)
	archive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md": "---\nname: demo\ndescription: Async demo\n---\n\nRun it.",
	})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillInstallJob{}))
	repo := repository.NewTenantSkillRepository(db)
	svc := NewTenantSkillService(repo, newTenantSkillFileStub()).(*tenantSkillService)

	job, err := svc.StartTenantSkillSourceInstall(
		context.Background(),
		42,
		"admin-1",
		server.URL+"/demo.zip",
	)
	require.NoError(t, err)
	require.Equal(t, types.TenantSkillInstallStatusPending, job.Status)

	final := waitForTenantSkillInstallJob(t, repo, 42, job.ID, types.TenantSkillInstallStatusSucceeded)
	require.NotEmpty(t, final.SkillID)
	require.Equal(t, 1, final.Attempt)
	skill, err := repo.GetByID(context.Background(), 42, final.SkillID)
	require.NoError(t, err)
	require.NotNil(t, skill)
	require.Equal(t, "demo", skill.Name)
}

func TestTenantSkillSourceInstallJobCanRetryAfterFailure(t *testing.T) {
	withTenantSkillSourceTestSSRFWhitelist(t)
	archive := buildSkillArchive(t, map[string]string{
		"retry-demo/SKILL.md": "---\nname: retry-demo\ndescription: Retry demo\n---\n\nRun it.",
	})
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			http.Error(w, "temporary source failure", http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(archive)
	}))
	defer server.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillInstallJob{}))
	repo := repository.NewTenantSkillRepository(db)
	svc := NewTenantSkillService(repo, newTenantSkillFileStub()).(*tenantSkillService)

	job, err := svc.StartTenantSkillSourceInstall(context.Background(), 42, "admin-1", server.URL+"/retry.zip")
	require.NoError(t, err)
	failed := waitForTenantSkillInstallJob(t, repo, 42, job.ID, types.TenantSkillInstallStatusFailed)
	require.Contains(t, failed.Error, "HTTP 502")

	retried, err := svc.RetryTenantSkillInstall(context.Background(), 42, job.ID)
	require.NoError(t, err)
	require.Equal(t, 2, retried.Attempt)
	succeeded := waitForTenantSkillInstallJob(t, repo, 42, job.ID, types.TenantSkillInstallStatusSucceeded)
	require.NotEmpty(t, succeeded.SkillID)
	require.Equal(t, 2, succeeded.Attempt)
}

func TestTenantSkillSourceInstallJobCanBeCancelled(t *testing.T) {
	withTenantSkillSourceTestSSRFWhitelist(t)
	requestStarted := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}, &types.TenantSkillInstallJob{}))
	repo := repository.NewTenantSkillRepository(db)
	svc := NewTenantSkillService(repo, newTenantSkillFileStub()).(*tenantSkillService)

	job, err := svc.StartTenantSkillSourceInstall(context.Background(), 42, "admin-1", server.URL+"/slow.zip")
	require.NoError(t, err)
	<-requestStarted
	cancelled, err := svc.CancelTenantSkillInstall(context.Background(), 42, job.ID)
	require.NoError(t, err)
	require.Equal(t, types.TenantSkillInstallStatusCancelling, cancelled.Status)
	final := waitForTenantSkillInstallJob(t, repo, 42, job.ID, types.TenantSkillInstallStatusCancelled)
	require.Equal(t, types.TenantSkillInstallStatusCancelled, final.Status)
}

func TestStoreTenantSkillArchiveUpgradesSameNameWithoutLosingState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&types.TenantSkill{}))

	repo := repository.NewTenantSkillRepository(db)
	files := newTenantSkillFileStub()
	svc := NewTenantSkillService(repo, files).(*tenantSkillService)

	firstArchive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md": "---\nname: demo\ndescription: First version\n---\n\nUse v1.",
	})
	firstMetadata, _, err := inspectTenantSkillArchive(firstArchive)
	require.NoError(t, err)
	first, err := svc.storeTenantSkillArchive(
		context.Background(),
		42,
		"admin-1",
		firstArchive,
		firstMetadata,
		types.TenantSkillSourceUpload,
		"",
	)
	require.NoError(t, err)
	require.NoError(t, repo.SetEnabled(context.Background(), 42, first.ID, false))

	secondArchive := buildSkillArchive(t, map[string]string{
		"demo/SKILL.md": "---\nname: demo\ndescription: Second version\n---\n\nUse v2.",
	})
	secondMetadata, _, err := inspectTenantSkillArchive(secondArchive)
	require.NoError(t, err)
	second, err := svc.storeTenantSkillArchive(
		context.Background(),
		42,
		"admin-2",
		secondArchive,
		secondMetadata,
		types.TenantSkillSourceGitHub,
		"https://github.com/acme/demo/archive/refs/heads/main.zip",
	)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, "Second version", second.Description)
	require.Equal(t, types.TenantSkillSourceGitHub, second.SourceType)
	require.Equal(t, "https://github.com/acme/demo/archive/refs/heads/main.zip", second.SourceRef)
	require.False(t, second.Enabled)
	require.NotEqual(t, first.BundlePath, second.BundlePath)
	require.Contains(t, files.deleted, first.BundlePath)

	stored, err := repo.GetByID(context.Background(), 42, first.ID)
	require.NoError(t, err)
	require.Equal(t, second.BundleSHA, stored.BundleSHA)
	require.Equal(t, second.BundlePath, stored.BundlePath)
}

type tenantSkillFileStub struct {
	files   map[string][]byte
	counter int
	deleted []string
}

func withTenantSkillSourceTestSSRFWhitelist(t *testing.T) {
	t.Helper()
	t.Setenv("SSRF_WHITELIST", "127.0.0.1,::1,localhost")
	utils.ResetSSRFWhitelistForTest()
	t.Cleanup(utils.ResetSSRFWhitelistForTest)
}

func waitForTenantSkillInstallJob(
	t *testing.T,
	repo interfaces.TenantSkillRepository,
	tenantID uint64,
	id, status string,
) *types.TenantSkillInstallJob {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		job, err := repo.GetInstallJob(context.Background(), tenantID, id)
		require.NoError(t, err)
		if job != nil && job.Status == status {
			return job
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, err := repo.GetInstallJob(context.Background(), tenantID, id)
	require.NoError(t, err)
	require.Failf(t, "skill install job did not reach expected status", "expected %s, got %#v", status, job)
	return job
}

func newTenantSkillFileStub() *tenantSkillFileStub {
	return &tenantSkillFileStub{files: make(map[string][]byte)}
}

func (s *tenantSkillFileStub) CheckConnectivity(context.Context) error { return nil }

func (s *tenantSkillFileStub) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	return "", nil
}

func (s *tenantSkillFileStub) SaveBytes(_ context.Context, data []byte, _ uint64, _ string, _ bool) (string, error) {
	s.counter++
	path := fmt.Sprintf("resource://skill/%d", s.counter)
	s.files[path] = append([]byte(nil), data...)
	return path, nil
}

func (s *tenantSkillFileStub) GetFile(_ context.Context, path string) (io.ReadCloser, error) {
	data, ok := s.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (s *tenantSkillFileStub) GetFileURL(context.Context, string) (string, error) {
	return "", nil
}

func (s *tenantSkillFileStub) DeleteFile(_ context.Context, path string) error {
	s.deleted = append(s.deleted, path)
	delete(s.files, path)
	return nil
}

func (s *tenantSkillFileStub) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", nil
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
