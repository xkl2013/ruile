package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	))
	return NewExpertPackageService(repository.NewExpertPackageRepository(db), fileService)
}

type expertPackageFileServiceStub struct {
	saved          bool
	savedBytes     bool
	saveCount      int
	saveBytesCount int
}

func (s *expertPackageFileServiceStub) CheckConnectivity(context.Context) error { return nil }

func (s *expertPackageFileServiceStub) SaveFile(context.Context, *multipart.FileHeader, uint64, string) (string, error) {
	s.saved = true
	s.saveCount++
	return "resource://expert-packages/kindergarten-activity-planner.zip", nil
}

func (s *expertPackageFileServiceStub) SaveBytes(context.Context, []byte, uint64, string, bool) (string, error) {
	s.savedBytes = true
	s.saveBytesCount++
	return "resource://expert-packages/kindergarten-activity-planner-avatar", nil
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

type expertPackageModelServiceStub struct {
	interfaces.ModelService
	models []*types.Model
}

func (s *expertPackageModelServiceStub) ListModels(context.Context) ([]*types.Model, error) {
	return s.models, nil
}

func TestExpertPackageImportAndPublish(t *testing.T) {
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
			{Path: "skills/evidence-analysis/references/checklist.md", Content: "# 检查清单\n"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.ExpertPackageVersionTesting, version.State)
	require.Len(t, version.Definitions, 1)
	require.Equal(t, 20, int(version.Definitions[0].CompiledConfig["max_iterations"].(float64)))
	var importedConfig types.CustomAgentConfig
	importedConfigData, err := json.Marshal(version.Definitions[0].CompiledConfig)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(importedConfigData, &importedConfig))
	require.NotNil(t, importedConfig.QuestionSuggestions)
	require.False(t, importedConfig.QuestionSuggestions.Starters.Enabled)
	require.Equal(t, types.SuggestionModeCurated, importedConfig.QuestionSuggestions.Starters.Mode)
	require.Contains(t, version.Diagnostics["warnings"], "unsupported capability: script.python")
	snapshots, err := loadExpertSkillSnapshots(version.Definitions[0])
	require.NoError(t, err)
	require.Len(t, snapshots, 1)
	require.Equal(t, "evidence-analysis", snapshots[0].Name)
	require.Equal(t, "skills/evidence-analysis/SKILL.md", snapshots[0].Path)
	require.Equal(t, "# 证据分析\n", snapshots[0].Content)
	var snapshotFiles []map[string]any
	rawSnapshots, err := json.Marshal(version.Definitions[0].CompiledConfig["skill_snapshots"])
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(rawSnapshots, &snapshotFiles))
	require.Len(t, snapshotFiles, 2)
	require.Equal(t, "skills/evidence-analysis/references/checklist.md", snapshotFiles[1]["path"])

	packages, err := svc.ListPackages(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Len(t, packages[0].Versions, 1)
	require.Len(t, packages[0].Versions[0].Definitions, 1)

	require.NoError(t, svc.PublishVersion(ctx, tenantID, "admin-1", packages[0].ID, version.ID))
	published, err := svc.ListPublishedExperts(ctx, tenantID)
	require.NoError(t, err)
	require.Len(t, published, 1)
	require.Equal(t, version.Definitions[0].ID, published[0].DefinitionID)
	require.Equal(t, "咨询顾问", published[0].DisplayName)
	require.Empty(t, published[0].Capabilities["system_prompt"])

}

func TestPublishedExpertIsMaterializedAsCustomAgent(t *testing.T) {
	ctx := context.WithValue(context.Background(), types.SystemAdminContextKey, true)
	db, err := gorm.Open(sqlite.Open("file:expert-agent-sync-"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.CustomAgent{},
		&types.ExpertPackage{},
		&types.ExpertPackageVersion{},
		&types.AgentDefinitionVersion{},
	))
	svc := NewExpertPackageServiceWithAgentRepository(
		repository.NewExpertPackageRepository(db),
		nil,
		repository.NewCustomAgentRepository(db),
		&expertPackageModelServiceStub{models: []*types.Model{
			{
				ID:     "rerank-fallback",
				Type:   types.ModelTypeRerank,
				Status: types.ModelStatusActive,
			},
			{
				ID:        "rerank-default",
				Type:      types.ModelTypeRerank,
				Status:    types.ModelStatusActive,
				IsDefault: true,
			},
		}},
	)

	version, err := svc.ImportPackage(ctx, 11, "admin-1", types.ExpertPackageImportInput{
		SourceFormat: types.ExpertPackageSourceWorkBuddy,
		PackageKey:   "sync-test",
		Version:      "1.0.0",
		DisplayName:  "同步测试专家包",
		Description:  "中文专家展示描述",
		Avatar:       "resource://expert-avatar",
		Files: []types.ExpertPackageFileInput{{
			Path: "agents/researcher.md",
			Content: `---
id: researcher
version: 1.0.0
display_name: 研究专家
description: English agent description.
skills:
  - researcher
quick_prompts:
  - 先帮我梳理研究目标
  - 请列出需要核验的证据
capabilities:
  required:
    - knowledge.read
---

请基于证据完成研究。`,
		}, {
			Path:    "skills/researcher/SKILL.md",
			Content: "---\nname: researcher\ndescription: 研究技能\n---\n\n# 研究\n",
		}},
	})
	require.NoError(t, err)
	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.Len(t, packages, 1)

	// Older imports persisted quick_prompts only in the package manifest.
	// Publishing or re-syncing them must hydrate the materialized Agent
	// without requiring the package to be imported again.
	delete(version.Definitions[0].CompiledConfig, "question_suggestions")
	require.NoError(t, db.Model(&types.AgentDefinitionVersion{}).
		Where("id = ?", version.Definitions[0].ID).
		Update("compiled_config", version.Definitions[0].CompiledConfig).Error)
	version.Manifest["quick_prompts"] = []expertLocalizedText{
		{ZH: "先帮我梳理研究目标"},
		{ZH: "请列出需要核验的证据"},
	}
	require.NoError(t, db.Model(&types.ExpertPackageVersion{}).
		Where("id = ?", version.ID).
		Update("manifest", version.Manifest).Error)

	require.NoError(t, svc.PublishVersion(ctx, 11, "admin-1", packages[0].ID, version.ID))
	agents, err := repository.NewCustomAgentRepository(db).ListAgentsByTenantID(ctx, 11)
	require.NoError(t, err)
	require.Len(t, agents, 1)
	require.Equal(t, "研究专家", agents[0].Name)
	require.Equal(t, "中文专家展示描述", agents[0].Description)
	require.Equal(t, "resource://expert-avatar", agents[0].Avatar)
	require.True(t, agents[0].IsBuiltin)
	require.Equal(t, types.SystemAgentTenantID, agents[0].TenantID)
	require.Equal(t, types.AgentModeSmartReasoning, agents[0].Config.AgentMode)
	require.Contains(t, agents[0].Config.Tags, types.AgentTagExpert)
	require.Equal(t, "请基于证据完成研究。", agents[0].Config.SystemPrompt)
	require.Contains(t, agents[0].Config.AllowedTools, "knowledge_search")
	require.Len(t, agents[0].Config.SkillSnapshots, 1)
	require.Equal(t, "researcher", agents[0].Config.SkillSnapshots[0].Name)
	require.Equal(t, "rerank-default", agents[0].Config.RerankModelID)
	require.NotNil(t, agents[0].Config.QuestionSuggestions)
	require.True(t, agents[0].Config.QuestionSuggestions.Starters.Enabled)
	require.Equal(t, types.SuggestionModeCurated, agents[0].Config.QuestionSuggestions.Starters.Mode)
	require.Equal(t, []string{
		"先帮我梳理研究目标",
		"请列出需要核验的证据",
	}, agents[0].Config.QuestionSuggestions.Starters.Items)

	require.NoError(t, db.Model(&types.CustomAgent{}).
		Where("id = ? AND tenant_id = ?", agents[0].ID, agents[0].TenantID).
		Updates(map[string]any{
			"description": "stale English description",
			"avatar":      "",
		}).Error)
	require.NoError(t, svc.SyncPublishedAgents(ctx, 11, "admin-1"))

	resynced, err := repository.NewCustomAgentRepository(db).GetAgentByID(
		ctx,
		agents[0].ID,
		types.SystemAgentTenantID,
	)
	require.NoError(t, err)
	require.Equal(t, "中文专家展示描述", resynced.Description)
	require.Equal(t, "resource://expert-avatar", resynced.Avatar)
	require.Equal(t, "rerank-default", resynced.Config.RerankModelID)
	require.Equal(t, types.SuggestionModeCurated, resynced.Config.QuestionSuggestions.Starters.Mode)

	// Older releases promoted the workspace-derived ID into tenant 0. Keep
	// updating that active row when the newer platform-derived ID was deleted.
	require.NoError(t, repository.NewCustomAgentRepository(db).DeleteAgent(
		ctx,
		resynced.ID,
		types.SystemAgentTenantID,
	))
	legacyID := uuid.NewSHA1(
		uuid.Nil,
		[]byte(fmt.Sprintf(
			"ruile:published-expert:%d:%s:%s",
			uint64(11),
			packages[0].ID,
			version.Definitions[0].AgentID,
		)),
	).String()
	promotedLegacy := *resynced
	promotedLegacy.ID = legacyID
	promotedLegacy.Description = "stale promoted description"
	promotedLegacy.Avatar = ""
	promotedLegacy.DeletedAt = gorm.DeletedAt{}
	require.NoError(t, repository.NewCustomAgentRepository(db).CreateAgent(ctx, &promotedLegacy))

	require.NoError(t, svc.SyncPublishedAgents(ctx, 11, "admin-1"))
	resyncedLegacy, err := repository.NewCustomAgentRepository(db).GetAgentByID(
		ctx,
		legacyID,
		types.SystemAgentTenantID,
	)
	require.NoError(t, err)
	require.Equal(t, "中文专家展示描述", resyncedLegacy.Description)
	require.Equal(t, "resource://expert-avatar", resyncedLegacy.Avatar)
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
	require.True(t, fileService.savedBytes)
	require.Len(t, version.Definitions, 1)
	require.Equal(t, "1.0.0", version.Version)
	require.Equal(t, "kindergarten-activity-planner", version.Definitions[0].AgentID)
	require.Equal(t, "童创", version.Definitions[0].DisplayName)
	require.Equal(t, "10年幼儿园活动策划经验，输出可直接落地的执行方案。", version.Definitions[0].Description)
	require.Equal(t, "kindergarten_activity_planner", version.Definitions[0].Domain)
	require.Equal(t, "resource://expert-packages/kindergarten-activity-planner.zip", version.Manifest["source_uri"])
	require.Equal(t, "resource://expert-packages/kindergarten-activity-planner-avatar", version.Manifest["avatar_ref"])
	require.Contains(t, version.Diagnostics["warnings"], "agent kindergarten-activity-planner maxTurns reduced from 50 to 20")

	config := version.Definitions[0].CompiledConfig
	require.Equal(t, float64(20), config["max_iterations"])
	var compiledConfig types.CustomAgentConfig
	configData, err := json.Marshal(config)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(configData, &compiledConfig))
	require.NotNil(t, compiledConfig.QuestionSuggestions)
	require.Equal(t, types.SuggestionModeCurated, compiledConfig.QuestionSuggestions.Starters.Mode)
	require.Equal(t, []string{
		"帮我策划一场秋季亲子运动会",
		"生成活动所需的物料清单",
		"检查活动方案中的安全风险",
	}, compiledConfig.QuestionSuggestions.Starters.Items)

	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Equal(t, "kindergarten-activity-planner", packages[0].PackageKey)
	require.Equal(t, "童创", packages[0].DisplayName)
	require.Equal(t, "10年幼儿园活动策划经验，输出可直接落地的执行方案。", packages[0].Description)
	require.Equal(t, "resource://expert-packages/kindergarten-activity-planner-avatar", packages[0].Avatar)

	require.NoError(t, svc.PublishVersion(ctx, 11, "admin-1", packages[0].ID, version.ID))
	published, err := svc.ListPublishedExperts(ctx, 11)
	require.NoError(t, err)
	require.Len(t, published, 1)
	require.Equal(t, packages[0].Avatar, published[0].Avatar)
	require.Equal(t, "10年幼儿园活动策划经验，输出可直接落地的执行方案。", published[0].Description)
	require.Equal(t, types.StringArray{"活动策划", "家长沟通", "安全预案"}, published[0].Tags)
}

func TestExpertArchiveSkillRelativePathStripsPackageRoot(t *testing.T) {
	path, ok := expertArchiveSkillRelativePath(
		"life-code-analyst/skills/life-code-analysis/references/01-rules.md",
		"life-code-analyst",
	)
	require.True(t, ok)
	require.Equal(t, "skills/life-code-analysis/references/01-rules.md", path)

	path, ok = expertArchiveSkillRelativePath(
		"skills/life-code-analysis/SKILL.md",
		"",
	)
	require.True(t, ok)
	require.Equal(t, "skills/life-code-analysis/SKILL.md", path)
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
	require.Contains(t, first.Definitions[0].CompiledConfig["tags"], "活动策划")
	require.Equal(t, 1, fileService.saveCount)
	require.Equal(t, 1, fileService.saveBytesCount)

	packages, err := svc.ListPackages(ctx, 11)
	require.NoError(t, err)
	require.Len(t, packages, 1)
	require.Len(t, packages[0].Versions, 1)
}

func TestWorkBuddyTagValuesUsesChineseLabelsAndLimitsThree(t *testing.T) {
	got := workBuddyTagValues([]expertLocalizedText{
		{EN: "English only", ZH: "中文标签"},
		{EN: "Duplicate", ZH: "中文标签"},
		{EN: "Second", ZH: "第二个"},
		{EN: "Third", ZH: "第三个"},
		{EN: "Fourth", ZH: "第四个"},
	})

	require.Equal(t, []string{"中文标签", "第二个", "第三个"}, got)
}

func TestExpertQuickPromptsFromManifestUsesChineseLabelsAndLimitsEight(t *testing.T) {
	got := expertQuickPromptsFromManifest([]expertLocalizedText{
		{EN: "English only", ZH: "中文提问"},
		{EN: "Duplicate", ZH: "中文提问"},
		{ZH: "第二个"},
		{ZH: "第三个"},
		{ZH: "第四个"},
		{ZH: "第五个"},
		{ZH: "第六个"},
		{ZH: "第七个"},
		{ZH: "第八个"},
		{ZH: "第九个"},
	})

	require.Equal(t, []string{
		"中文提问",
		"第二个",
		"第三个",
		"第四个",
		"第五个",
		"第六个",
		"第七个",
		"第八个",
	}, got)
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
  "tags": [
    {"en": "Activity Planning", "zh": "活动策划"},
    {"en": "Parent Communication", "zh": "家长沟通"},
    {"en": "Safety Planning", "zh": "安全预案"}
  ],
  "quickPrompts": [
    {"en": "Plan an autumn parent-child sports day", "zh": "帮我策划一场秋季亲子运动会"},
    {"en": "Generate a materials checklist", "zh": "生成活动所需的物料清单"},
    {"en": "Review safety risks", "zh": "检查活动方案中的安全风险"}
  ],
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
