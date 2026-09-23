package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"testing"

	"github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newExpertPackageTestService(t *testing.T) interfaces.ExpertPackageService {
	return newExpertPackageTestServiceWithFileService(t, nil)
}

func newExpertPackageTestServiceWithFileService(t *testing.T, fileService interfaces.FileService) interfaces.ExpertPackageService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:expert-package-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.ExpertPackage{},
		&types.ExpertPackageVersion{},
		&types.AgentDefinitionVersion{},
		&types.AgentBinding{},
	))
	return NewExpertPackageService(repository.NewExpertPackageRepository(db), fileService)
}

type expertPackageFileServiceStub struct {
	saved     bool
	saveCount int
}

func (s *expertPackageFileServiceStub) CheckConnectivity(context.Context) error { return nil }

func (s *expertPackageFileServiceStub) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	s.saved = true
	s.saveCount++
	return "resource://expert-packages/kindergarten-activity-planner.zip", nil
}

func (s *expertPackageFileServiceStub) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	return "", nil
}

func (s *expertPackageFileServiceStub) GetFile(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (s *expertPackageFileServiceStub) GetFileURL(context.Context, string) (string, error) {
	return "", nil
}

func (s *expertPackageFileServiceStub) DeleteFile(context.Context, string) error { return nil }

func (s *expertPackageFileServiceStub) CopyFile(context.Context, string, uint64, string) (string, error) {
	return "", nil
}

func TestExpertPackageImportPublishAndBind(t *testing.T) {
	ctx := context.Background()
	svc := newExpertPackageTestService(t)
	const tenantID uint64 = 11

	version, err := svc.ImportPackage(ctx, tenantID, "admin-1", types.ExpertPackageImportInput{
		SourceFormat: types.ExpertPackageSourceWorkBuddy,
		PackageKey:   "consulting-partners",
		Version:      "1.0.0",
		DisplayName:  "咨询专家包",
		Files: []types.ExpertPackageFileInput{
			{
				Path: "agents/consulting-partner.md",
				Content: `---
kind: service-agent
schema_version: "1.0"
id: consulting-partner
version: "1.0.0"
display_name: 咨询顾问
description: 帮助梳理客户问题
domain: customer_service
max_turns: 30
skills:
  - evidence-analysis
capabilities:
  required:
    - knowledge.read
  optional:
    - web.search
    - script.python
output_contract: agent_result_v1
---

# 角色

基于已知事实提出可执行建议。`,
			},
			{Path: "skills/evidence-analysis/SKILL.md", Content: "# 证据分析\n"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ExpertPackageVersionTesting, version.State)
	require.Len(t, version.Definitions, 1)
	require.Equal(t, 20, int(version.Definitions[0].CompiledConfig["max_iterations"].(float64)))
	require.Contains(t, version.Diagnostics["warnings"], "unsupported capability: script.python")
	snapshots, err := loadExpertSkillSnapshots(version.Definitions[0])
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, "evidence-analysis", snapshots[0].Name)
	require.Equal(t, "# 证据分析\n", snapshots[0].Content)

	packages, err := svc.ListPackages(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Len(t, packages[0].Versions, 1)
	require.Len(t, packages[0].Versions[0].Definitions, 1)

	_, err = svc.BindAgent(ctx, tenantID, "admin-1", packages[0].ID, types.AgentBindingInput{
		ProfileID:                "profile-1",
		AgentDefinitionVersionID: version.Definitions[0].ID,
		Enabled:                  true,
	})
	require.ErrorIs(t, err, ErrExpertPackageNotPublished)

	require.NoError(t, svc.PublishVersion(ctx, tenantID, "admin-1", packages[0].ID, version.ID))
	published, err := svc.ListPublishedExperts(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, published, 1)
	require.Equal(t, version.Definitions[0].ID, published[0].DefinitionID)
	require.Equal(t, "咨询顾问", published[0].DisplayName)
	require.Empty(t, published[0].Capabilities["system_prompt"])

	binding, err := svc.BindAgent(ctx, tenantID, "admin-1", packages[0].ID, types.AgentBindingInput{
		ProfileID:                "profile-1",
		AgentDefinitionVersionID: version.Definitions[0].ID,
		Enabled:                  true,
	})
	require.NoError(t, err)
	require.Equal(t, "customer_service", binding.AgentDomain)

	bindings, err := svc.ListBindings(ctx, tenantID, "profile-1")
	require.NoError(t, err)
	require.Len(t, bindings, 1)
}

func TestExpertPackageImportRejectsUnsafePath(t *testing.T) {
	svc := newExpertPackageTestService(t)
	_, err := svc.ImportPackage(context.Background(), 11, "admin-1", types.ExpertPackageImportInput{
		SourceFormat: types.ExpertPackageSourceRuileNative,
		PackageKey:   "unsafe",
		Version:      "1.0.0",
		DisplayName:  "Unsafe",
		Files: []types.ExpertPackageFileInput{
			{Path: "../agents/unsafe.md", Content: "---\nid: unsafe\n---\nbody"},
		},
	})
	require.True(t, errors.Is(err, ErrExpertPackageInvalidInput))
}

func TestExpertPackagePublishBlocksMissingSkill(t *testing.T) {
	ctx := context.Background()
	svc := newExpertPackageTestService(t)
	version, err := svc.ImportPackage(ctx, 11, "admin-1", types.ExpertPackageImportInput{
		SourceFormat: types.ExpertPackageSourceRuileNative,
		PackageKey:   "blocked",
		Version:      "1.0.0",
		DisplayName:  "Blocked",
		Files: []types.ExpertPackageFileInput{{
			Path:    "agents/blocked.md",
			Content: "---\nid: blocked\nversion: 1.0.0\ndisplay_name: Blocked\nskills:\n  - absent\n---\n\n# Role\n",
		}},
	})
	require.NoError(t, err)
	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.ErrorIs(t, svc.PublishVersion(ctx, 11, "admin-1", packages[0].ID, version.ID), ErrExpertPackageInvalidInput)
}

func TestExpertPackageImportWorkBuddyArchive(t *testing.T) {
	ctx := context.Background()
	fileService := &expertPackageFileServiceStub{}
	svc := newExpertPackageTestServiceWithFileService(t, fileService)

	zipBytes := newWorkBuddyExpertZip(t)
	version, err := svc.ImportPackageArchive(
		ctx,
		11,
		"admin-1",
		newMultipartFileHeader(t, "kindergarten-activity-planner.zip", string(zipBytes)),
	)
	require.NoError(t, err)
	require.True(t, fileService.saved)
	require.Len(t, version.Definitions, 1)
	require.Equal(t, "1.0.0", version.Version)
	require.Equal(t, "kindergarten-activity-planner", version.Definitions[0].AgentID)
	require.Equal(t, "童创", version.Definitions[0].DisplayName)
	require.Equal(t, "kindergarten_activity_planner", version.Definitions[0].Domain)
	require.Equal(t, "resource://expert-packages/kindergarten-activity-planner.zip", version.Manifest["source_uri"])
	require.Contains(t, version.Diagnostics["warnings"], "agent kindergarten-activity-planner maxTurns reduced from 50 to 20")

	config := version.Definitions[0].CompiledConfig
	require.Equal(t, float64(20), config["max_iterations"])

	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Equal(t, "kindergarten-activity-planner", packages[0].PackageKey)
	require.Equal(t, "童创", packages[0].DisplayName)
}

func TestExpertPackageImportWorkBuddyArchiveIsIdempotentForSameContent(t *testing.T) {
	ctx := context.Background()
	fileService := &expertPackageFileServiceStub{}
	svc := newExpertPackageTestServiceWithFileService(t, fileService)

	zipBytes := newWorkBuddyExpertZip(t)
	finderZipBytes := newWorkBuddyExpertZipWithOptions(
		t,
		"kindergarten-activity-planner 2",
		defaultWorkBuddyExpertBody,
		true,
	)
	first, err := svc.ImportPackageArchive(
		ctx,
		11,
		"admin-1",
		newMultipartFileHeader(t, "kindergarten-activity-planner.zip", string(zipBytes)),
	)
	require.NoError(t, err)

	second, err := svc.ImportPackageArchive(
		ctx,
		11,
		"admin-1",
		newMultipartFileHeader(t, "kindergarten-activity-planner 2.zip", string(finderZipBytes)),
	)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.PackageHash, second.PackageHash)
	require.Len(t, second.Definitions, 1)
	require.Equal(t, 1, fileService.saveCount)

	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Len(t, packages[0].Versions, 1)
}

func TestExpertPackageImportWorkBuddyArchiveRejectsChangedDuplicateVersion(t *testing.T) {
	ctx := context.Background()
	fileService := &expertPackageFileServiceStub{}
	svc := newExpertPackageTestServiceWithFileService(t, fileService)

	_, err := svc.ImportPackageArchive(
		ctx,
		11,
		"admin-1",
		newMultipartFileHeader(t, "kindergarten-activity-planner.zip", string(newWorkBuddyExpertZip(t))),
	)
	require.NoError(t, err)

	_, err = svc.ImportPackageArchive(
		ctx,
		11,
		"admin-1",
		newMultipartFileHeader(t, "kindergarten-activity-planner.zip", string(newWorkBuddyExpertZipWithAgentBody(t, "# 幼儿园活动策划专家\n\n已修改。"))),
	)
	require.ErrorIs(t, err, ErrExpertPackageVersionExists)
	require.Contains(t, err.Error(), "update plugin.json version")
	require.Equal(t, 1, fileService.saveCount)
}

func newWorkBuddyExpertZip(t *testing.T) []byte {
	return newWorkBuddyExpertZipWithAgentBody(t, defaultWorkBuddyExpertBody)
}

func newWorkBuddyExpertZipWithAgentBody(t *testing.T, agentBody string) []byte {
	return newWorkBuddyExpertZipWithOptions(t, "kindergarten-activity-planner", agentBody, false)
}

const defaultWorkBuddyExpertBody = "# 幼儿园活动策划专家\n\n输出活动流程时间轴、人员分工、物料清单和安全预案。"

func newWorkBuddyExpertZipWithOptions(t *testing.T, root string, agentBody string, includeMacMetadata bool) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	writeZipFile := func(name, content string) {
		t.Helper()
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write([]byte(content))
		require.NoError(t, err)
	}
	prefix := root + "/"
	if includeMacMetadata {
		writeZipFile(prefix+".DS_Store", "finder metadata")
		writeZipFile("__MACOSX/._"+root, "appledouble metadata")
	}
	writeZipFile(prefix+".codebuddy-plugin/plugin.json", `{
  "name": "kindergarten-activity-planner",
  "version": "1.0.0",
  "description": "Kindergarten activity planning expert.",
  "agents": ["./agents/kindergarten-activity-planner.md"],
  "expertType": "agent",
  "agentName": "kindergarten-activity-planner",
  "displayName": {"en": "Tongchuang", "zh": "童创"},
  "profession": {"en": "Kindergarten Activity Planning Expert", "zh": "幼儿园活动策划专家"},
  "displayDescription": {"en": "Activity planner", "zh": "10年幼儿园活动策划经验，输出可直接落地的执行方案。"},
  "avatar": "avatars/expert.png",
  "categoryId": "06-ContentCreative",
  "plugin": "kindergarten-activity-planner"
}`)
	writeZipFile(prefix+"agents/kindergarten-activity-planner.md", `---
name: kindergarten-activity-planner
description: "Kindergarten activity planning expert."
displayName:
  en: "Tongchuang"
  zh: "童创"
maxTurns: 50
---

`+agentBody)
	writeZipFile(prefix+"avatars/expert.png", "fake-png")
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
