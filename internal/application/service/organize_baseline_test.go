package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

type organizeBaselineMemory struct {
	ID          string                       `json:"id"`
	Kind        string                       `json:"kind"`
	Title       string                       `json:"title"`
	Content     string                       `json:"content"`
	Source      string                       `json:"source"`
	OccurredAt  string                       `json:"occurred_at"`
	Attachments []organizeBaselineAttachment `json:"attachments"`
}

type organizeBaselineAttachment struct {
	FileName   string `json:"file_name"`
	MIMEType   string `json:"mime_type"`
	Status     string `json:"status"`
	Transcript string `json:"transcript"`
}

type organizeBaselineOutputSnapshot struct {
	Input        string   `json:"input"`
	ExpectedKeys []string `json:"expected_keys"`
	Format       string   `json:"format"`
}

func loadOrganizeBaselineMemories(t *testing.T) []organizeBaselineMemory {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "organize_baseline", "memories.json"))
	require.NoError(t, err)
	var memories []organizeBaselineMemory
	require.NoError(t, json.Unmarshal(raw, &memories))
	return memories
}

func loadOrganizeBaselineSnapshots(t *testing.T) map[string]organizeBaselineOutputSnapshot {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "organize_baseline", "output_snapshots.json"))
	require.NoError(t, err)
	var snapshots map[string]organizeBaselineOutputSnapshot
	require.NoError(t, json.Unmarshal(raw, &snapshots))
	return snapshots
}

func canonicalOrganizeSnapshot(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	require.NoError(t, err)
	return string(raw)
}

func organizeSnapshotHash(t *testing.T, value any) string {
	t.Helper()
	sum := sha256.Sum256([]byte(canonicalOrganizeSnapshot(t, value)))
	return hex.EncodeToString(sum[:])
}

func templateSnapshotForTest(template *types.OrganizeTemplate) types.JSONMap {
	return types.JSONMap{
		"key":                 template.Key,
		"name":                template.Name,
		"scene":               template.Scene,
		"description":         template.Description,
		"output_label":        template.OutputLabel,
		"default_instruction": template.DefaultInstruction,
		"expert_ids":          template.ExpertIDs,
		"spec":                template.Spec,
		"version":             template.PublishedVersion,
	}
}

func jobSnapshotForTest(job *types.OrganizeJob) types.JSONMap {
	return types.JSONMap{
		"id":               job.ID,
		"config_id":        job.ConfigID,
		"template_key":     job.TemplateKey,
		"template_version": job.TemplateVersion,
		"memory_ids":       job.MemoryIDs,
		"requirement":      job.Requirement,
	}
}

func TestOrganizeBaselineFixtureIsCompleteAndStable(t *testing.T) {
	memories := loadOrganizeBaselineMemories(t)
	require.Len(t, memories, 4)
	require.ElementsMatch(t, []string{"note", "audio"}, []string{
		memories[0].Kind,
		memories[1].Kind,
	})
	require.NotEmpty(t, memories[1].Attachments)
	require.Len(t, memories[2].Attachments, 2)
	require.Empty(t, memories[3].Content)

	first := organizeSnapshotHash(t, memories)
	second := organizeSnapshotHash(t, memories)
	require.Equal(t, first, second)
	require.Equal(t, "882ca8a756ac8113fd9fd6ef0f17b8463438aa77e55be1d013ea7ffdf2ac50b8", first)
}

func TestOrganizeBaselineOutputSnapshotsCoverLegacyChains(t *testing.T) {
	snapshots := loadOrganizeBaselineSnapshots(t)
	require.Equal(t, []string{
		"note_audio_transcribe",
		"note_import_meta",
		"output_card_meta",
		"sprout_review",
	}, sortedKeys(snapshots))

	for chain, snapshot := range snapshots {
		require.NotEmpty(t, snapshot.Input, chain)
		require.NotEmpty(t, snapshot.ExpectedKeys, chain)
		require.Contains(t, []string{"json", "markdown"}, snapshot.Format, chain)
	}
}

func TestOrganizeFeatureFlagsDefaultToCompatibilityBaseline(t *testing.T) {
	flags := ResolveOrganizeFeatureFlags(context.Background(), nil)
	require.False(t, flags.TemplateEngineEnabled)
	require.False(t, flags.TemplateEngineObserveOnly)
	require.False(t, flags.SSEEnabled)
	require.False(t, flags.CustomRequirementEnabled)
	require.False(t, flags.DiscoverCategoriesEnabled)
}

func TestOrganizeObservationOnlyComparesHashesWithoutChangingValues(t *testing.T) {
	legacy := map[string]any{"title": "原始结果", "tags": []string{"教研"}}
	candidate := map[string]any{"title": "新结果", "tags": []string{"教研"}}

	observation := BuildOrganizeObservation("note_import_meta", legacy, candidate)
	require.Equal(t, "note_import_meta", observation.Chain)
	require.NotEmpty(t, observation.LegacyHash)
	require.NotEmpty(t, observation.CandidateHash)
	require.True(t, observation.Changed)
	require.Equal(t, "原始结果", legacy["title"])
}

func sortedKeys[T any](values map[string]T) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func TestOrganizeSnapshotHelpersKeepTemplateAndJobContracts(t *testing.T) {
	template := &types.OrganizeTemplate{
		Key:                "baseline",
		Name:               "基线模板",
		Scene:              "测试",
		Description:        "固定测试模板",
		OutputLabel:        "测试结果",
		DefaultInstruction: "只依据输入内容整理",
		PublishedVersion:   "v1",
		Spec:               types.JSONMap{"sections": []string{"结论", "待办"}},
	}
	templateSnapshot := templateSnapshotForTest(template)
	require.Equal(t, "baseline", templateSnapshot["key"])
	require.Equal(t, "v1", templateSnapshot["version"])

	job := &types.OrganizeJob{
		ID:              "job-baseline",
		ConfigID:        "config-baseline",
		TemplateKey:     "baseline",
		TemplateVersion: "v1",
		MemoryIDs:       types.StringArray{"baseline-note"},
		Requirement:     types.JSONMap{"template_spec": template.Spec},
	}
	jobSnapshot := jobSnapshotForTest(job)
	require.Equal(t, "baseline", jobSnapshot["template_key"])
	require.Equal(t, []string{"baseline-note"}, []string(jobSnapshot["memory_ids"].(types.StringArray)))
}
