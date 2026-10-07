package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

func TestOrganizeAdminTemplateLifecycle(t *testing.T) {
	svc, _ := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()

	created, err := svc.CreateAdminTemplate(ctx, "admin-1", types.OrganizeTemplateAdminInput{
		Key:                "weekly_review",
		Name:               "周复盘",
		Scene:              "教师成长",
		DefaultInstruction: "整理 {{memory_count}} 条记忆。",
		Spec: types.JSONMap{
			"fields": []any{
				map[string]any{"key": "status", "label": "状态"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeTemplateStatusDraft, created.Status)
	require.Equal(t, uint64(0), created.TenantID)
	require.Equal(t, types.OrganizeTemplateScopePlatform, created.Scope)
	require.Contains(t, created.MarkdownTemplate, "# {{title}}")

	updated, err := svc.UpdateAdminTemplate(ctx, created.Key, "admin-1", types.OrganizeTemplateAdminInput{
		Name:               "周复盘 V2",
		Scene:              "教师成长",
		DefaultInstruction: "整理 {{memory_count}} 条记忆，并标注状态。",
		Spec: types.JSONMap{
			"fields": []any{
				map[string]any{"key": "status", "label": "状态"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "周复盘 V2", updated.Name)
	require.Equal(t, created.Key, updated.Key)

	preview, err := svc.PreviewAdminTemplate(ctx, created.Key, types.OrganizeTemplatePreviewInput{
		MemoryIDs: []string{"m1", "m2"},
	})
	require.NoError(t, err)
	require.Contains(t, preview.Prompt, "2")
	require.Contains(t, preview.Prompt, "Markdown 报告预设")
	require.Contains(t, preview.MarkdownTemplate, "##")
	require.Contains(t, preview.PreviewMarkdown, "# 周复盘 V2")
	require.Contains(t, preview.PreviewMarkdown, "示例记忆 1")
	require.NotContains(t, preview.PreviewMarkdown, "{{")

	published, err := svc.PublishAdminTemplate(ctx, created.Key, "admin-1", "首版发布")
	require.NoError(t, err)
	require.Equal(t, types.OrganizeTemplateStatusEnabled, published.Status)
	require.Equal(t, "v1", published.PublishedVersion)

	updated, err = svc.UpdateAdminTemplate(ctx, created.Key, "admin-1", types.OrganizeTemplateAdminInput{
		Name: "周复盘 V3",
		Spec: types.JSONMap{
			"fields": []any{
				map[string]any{"key": "status", "label": "状态"},
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "周复盘 V3", updated.Name)

	published, err = svc.PublishAdminTemplate(ctx, created.Key, "admin-1", "第二版发布")
	require.NoError(t, err)
	require.Equal(t, "v2", published.PublishedVersion)

	rolledBack, err := svc.RollbackAdminTemplate(ctx, created.Key, "v1", "admin-1", "恢复首版")
	require.NoError(t, err)
	require.Equal(t, "v3", rolledBack.PublishedVersion)
	require.Equal(t, "周复盘 V2", rolledBack.Name)

	versions, total, err := svc.ListAdminTemplateVersions(ctx, types.OrganizeTemplateVersionQuery{
		TemplateKey: created.Key,
		Page:        1,
		PageSize:    20,
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)
	require.Len(t, versions, 3)

	disabled, err := svc.DisableAdminTemplate(ctx, created.Key, "admin-1")
	require.NoError(t, err)
	require.Equal(t, types.OrganizeTemplateStatusDisabled, disabled.Status)

	_, err = svc.UpdateAdminTemplate(ctx, created.Key, "admin-1", types.OrganizeTemplateAdminInput{
		Key:  "other_key",
		Name: "invalid",
		Spec: types.JSONMap{},
	})
	require.ErrorIs(t, err, ErrOrganizeAdminTemplateKeyImmutable)
}

func TestOrganizeAdminTemplateScenesComeFromStoredTemplates(t *testing.T) {
	svc, _ := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()

	for _, input := range []types.OrganizeTemplateAdminInput{
		{
			Key:                "parent_followup",
			Name:               "家长跟进",
			Scene:              "家长服务",
			DefaultInstruction: "整理家长跟进记录。",
			SortOrder:          30,
		},
		{
			Key:                "lead_review",
			Name:               "线索复盘",
			Scene:              "招生增长",
			DefaultInstruction: "整理线索复盘记录。",
			SortOrder:          10,
		},
		{
			Key:                "channel_review",
			Name:               "渠道复盘",
			Scene:              "招生增长",
			DefaultInstruction: "整理渠道复盘记录。",
			SortOrder:          20,
		},
	} {
		_, err := svc.CreateAdminTemplate(ctx, "admin-1", input)
		require.NoError(t, err)
	}

	scenes, err := svc.ListAdminTemplateScenes(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{"招生增长", "家长服务"}, scenes)
}

func TestOrganizeUserTemplateListHidesInternalRecipes(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	for _, template := range []*types.OrganizeTemplate{
		{
			Scope:              types.OrganizeTemplateScopePlatform,
			Key:                "note_import_meta",
			Name:               "文件导入元数据",
			DefaultInstruction: "内部导入处理模板",
			Status:             types.OrganizeTemplateStatusEnabled,
			PublishedVersion:   "v1",
		},
		{
			Scope:              types.OrganizeTemplateScopePlatform,
			Key:                "teacher_research",
			Name:               "教研提炼",
			DefaultInstruction: "用户可选整理模板",
			Status:             types.OrganizeTemplateStatusEnabled,
			PublishedVersion:   "v1",
		},
	} {
		require.NoError(t, db.Create(template).Error)
	}

	templates, err := svc.ListTemplates(ctx, 7, "user-a")
	require.NoError(t, err)
	require.Len(t, templates, 1)
	require.Equal(t, "teacher_research", templates[0].Key)

	internalTemplate, err := svc.GetTemplate(ctx, 7, "user-a", "note_import_meta")
	require.ErrorIs(t, err, ErrOrganizeTemplateDisabled)
	require.Nil(t, internalTemplate)
}

func TestOrganizeAdminRejectsNonOrganizeTemplateKeys(t *testing.T) {
	svc, _ := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()

	_, err := svc.CreateAdminTemplate(ctx, "admin-1", types.OrganizeTemplateAdminInput{
		Key:                "note_audio_transcribe",
		Name:               "录音笔记整理",
		DefaultInstruction: "内部处理规则",
		Spec:               types.JSONMap{},
	})
	require.ErrorIs(t, err, ErrOrganizeAdminTemplateKeyReserved)

	_, err = svc.GetAdminTemplate(ctx, "note_audio_transcribe")
	require.ErrorIs(t, err, ErrOrganizeAdminTemplateNotFound)
}

func TestOrganizeOutputFacetsAndCitation(t *testing.T) {
	svc, _ := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()

	memory, err := svc.CreateMemory(ctx, 7, "user-a", types.OrganizeMemoryInput{
		Kind:    types.OrganizeMemoryKindNote,
		Title:   "课堂记录",
		Content: "儿童在建构区持续合作。",
	})
	require.NoError(t, err)
	output, err := svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title:       "教研提炼",
		TemplateKey: "teacher_research",
		Fields: types.JSONMap{
			"status": "待跟进",
			"owner":  "小王",
		},
		Citations: types.JSONMap{
			"memory_refs": []any{
				map[string]any{"label": "M1", "id": memory.ID},
			},
		},
		MemoryIDs: []string{memory.ID},
	})
	require.NoError(t, err)

	facets, err := svc.ListOutputFacets(ctx, types.OrganizeListQuery{
		TenantID: 7,
		UserID:   "user-a",
	})
	require.NoError(t, err)
	require.Len(t, facets.Fields, 2)

	resolved, missing, err := svc.GetOutputCitation(ctx, 7, "user-a", output.ID, "M1")
	require.NoError(t, err)
	require.False(t, missing)
	require.Equal(t, memory.ID, resolved.ID)

	_, missing, err = svc.GetOutputCitation(ctx, 7, "user-a", output.ID, "M9")
	require.NoError(t, err)
	require.True(t, missing)
}

func TestOrganizeDiscoverCategoryLifecycle(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	require.NoError(t, db.AutoMigrate(&types.OrganizeDiscoverCategoryRecord{}))
	ctx := context.Background()

	created, err := svc.CreateAdminDiscoverCategory(ctx, types.OrganizeDiscoverCategoryInput{
		Key:         "custom_research",
		Label:       "自定义教研",
		Description: "用于验证动态发现栏目",
		SortOrder:   15,
	})
	require.NoError(t, err)
	require.Equal(t, types.OrganizeDiscoverCategoryStatusEnabled, created.Status)

	public, err := svc.ListDiscoverCategories(ctx)
	require.NoError(t, err)
	require.Len(t, public, 1)
	require.Equal(t, "custom_research", public[0].Key)

	updated, err := svc.UpdateAdminDiscoverCategory(ctx, created.Key, types.OrganizeDiscoverCategoryInput{
		Key:         created.Key,
		Label:       "自定义教研栏目",
		Description: "已更新",
		SortOrder:   20,
	})
	require.NoError(t, err)
	require.Equal(t, "自定义教研栏目", updated.Label)

	disabled, err := svc.DisableAdminDiscoverCategory(ctx, created.Key)
	require.NoError(t, err)
	require.Equal(t, types.OrganizeDiscoverCategoryStatusDisabled, disabled.Status)

	public, err = svc.ListDiscoverCategories(ctx)
	require.NoError(t, err)
	require.Empty(t, public)

	_, err = svc.CreateOutput(ctx, 7, "user-a", types.OrganizeOutputInput{
		Title: "停用栏目不应继续接收新内容",
		Metadata: types.JSONMap{
			"discover_category": created.Key,
		},
	})
	require.ErrorIs(t, err, ErrOrganizeInvalidCategory)
}

func TestOrganizeRequirementPreviewRequiresConfirmationForImplicitScope(t *testing.T) {
	svc, db := newOrganizeServiceWithDBForTest(t)
	ctx := context.Background()
	template := &types.OrganizeTemplate{
		Scope:              types.OrganizeTemplateScopePlatform,
		Key:                "requirement_review",
		Name:               "要求复盘",
		Scene:              "测试",
		OutputLabel:        "复盘结果",
		DefaultInstruction: "按要求整理并保留事实引用。",
		Status:             types.OrganizeTemplateStatusEnabled,
		PublishedVersion:   "v1",
	}
	require.NoError(t, db.Create(template).Error)

	config, err := svc.CreateConfig(ctx, 7, "user-a", types.OrganizeConfigInput{
		Name:        "临时要求复盘",
		TemplateKey: template.Key,
		Schedule:    types.OrganizeScheduleManual,
	})
	require.NoError(t, err)

	preview, err := svc.PreviewOrganizeRequirement(ctx, 7, "user-a", types.OrganizeRequirementInput{
		ConfigID: config.ID,
		Text:     "整理重点事项",
	})
	require.NoError(t, err)
	require.Equal(t, template.Key, preview.TemplateKey)
	require.True(t, preview.NeedConfirmation)
	require.Contains(t, preview.Ambiguities, "未识别到明确时间范围，将使用当前配置的默认记忆范围")

	_, err = svc.ConfirmOrganizeRequirement(ctx, 7, "user-a", types.OrganizeRequirementInput{
		ConfigID: config.ID,
		Text:     "整理重点事项",
	})
	require.ErrorIs(t, err, ErrOrganizeRequirementConfirmationNeeded)
}
