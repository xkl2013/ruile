package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newAssetTransferTestRepository(t *testing.T) (*tenantMemberRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf(
		"file:%s?mode=memory&cache=shared",
		strings.NewReplacer("/", "_", " ", "_").Replace(t.Name()),
	)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.TenantMember{},
		&types.KnowledgeBase{},
		&types.CustomAgent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	repo, ok := NewTenantMemberRepository(db).(*tenantMemberRepository)
	if !ok {
		t.Fatal("unexpected tenant member repository implementation")
	}
	return repo, db
}

func seedAssetTransferMembers(t *testing.T, db *gorm.DB) {
	t.Helper()
	members := []types.TenantMember{
		{
			UserID:   "source",
			TenantID: 1,
			Role:     types.TenantRoleContributor,
			Status:   types.TenantMemberStatusActive,
		},
		{
			UserID:   "target",
			TenantID: 1,
			Role:     types.TenantRoleAdmin,
			Status:   types.TenantMemberStatusActive,
		},
		{
			UserID:   "suspended",
			TenantID: 1,
			Role:     types.TenantRoleContributor,
			Status:   types.TenantMemberStatusSuspended,
		},
	}
	if err := db.Create(&members).Error; err != nil {
		t.Fatalf("seed members: %v", err)
	}
}

func seedTransferableAssets(t *testing.T, db *gorm.DB) {
	t.Helper()
	knowledgeBases := []types.KnowledgeBase{
		{ID: "kb-1", Name: "招生知识库", TenantID: 1, CreatorID: "source"},
		{ID: "kb-2", Name: "教研知识库", TenantID: 1, CreatorID: "source"},
		{ID: "kb-other", Name: "其他成员知识库", TenantID: 1, CreatorID: "other"},
		{ID: "kb-temp", Name: "临时知识库", TenantID: 1, CreatorID: "source", IsTemporary: true},
	}
	if err := db.Create(&knowledgeBases).Error; err != nil {
		t.Fatalf("seed knowledge bases: %v", err)
	}
	agents := []types.CustomAgent{
		{ID: "agent-1", Name: "招生助手", TenantID: 1, CreatedBy: "source"},
		{ID: "agent-other", Name: "其他成员助手", TenantID: 1, CreatedBy: "other"},
		{ID: "agent-builtin", Name: "内置助手", TenantID: 1, CreatedBy: "source", IsBuiltin: true},
	}
	if err := db.Create(&agents).Error; err != nil {
		t.Fatalf("seed agents: %v", err)
	}
}

func TestTenantMemberRepository_ListAndTransferMemberAssets(t *testing.T) {
	repo, db := newAssetTransferTestRepository(t)
	seedAssetTransferMembers(t, db)
	seedTransferableAssets(t, db)
	ctx := context.Background()

	assets, err := repo.ListTransferableAssets(ctx, 1, "source")
	if err != nil {
		t.Fatalf("ListTransferableAssets: %v", err)
	}
	if assets.Total != 3 || len(assets.KnowledgeBases) != 2 || len(assets.Agents) != 1 {
		t.Fatalf("unexpected asset inventory: %+v", assets)
	}
	knowledgeBaseCount, agentCount, err := repo.CountTransferableAssets(ctx, 1, "source")
	if err != nil {
		t.Fatalf("CountTransferableAssets: %v", err)
	}
	if knowledgeBaseCount != 2 || agentCount != 1 {
		t.Fatalf("counts = (%d, %d), want (2, 1)", knowledgeBaseCount, agentCount)
	}

	result, err := repo.TransferMemberAssets(ctx, types.MemberAssetTransferCommand{
		TenantID:         1,
		SourceUserID:     "source",
		TargetType:       types.MemberAssetTransferTargetMember,
		TargetUserID:     "target",
		Scope:            types.MemberAssetTransferScopeSelected,
		KnowledgeBaseIDs: []string{"kb-1"},
		AgentIDs:         []string{"agent-1"},
	})
	if err != nil {
		t.Fatalf("TransferMemberAssets selected: %v", err)
	}
	if result.KnowledgeBasesTransferred != 1 || result.AgentsTransferred != 1 || result.TotalTransferred != 2 {
		t.Fatalf("unexpected transfer result: %+v", result)
	}

	var transferredKB types.KnowledgeBase
	if err := db.First(&transferredKB, "id = ?", "kb-1").Error; err != nil {
		t.Fatalf("load transferred knowledge base: %v", err)
	}
	if transferredKB.CreatorID != "target" {
		t.Fatalf("kb-1 creator = %q, want target", transferredKB.CreatorID)
	}
	var transferredAgent types.CustomAgent
	if err := db.Where("id = ? AND tenant_id = ?", "agent-1", 1).First(&transferredAgent).Error; err != nil {
		t.Fatalf("load transferred agent: %v", err)
	}
	if transferredAgent.CreatedBy != "target" {
		t.Fatalf("agent-1 creator = %q, want target", transferredAgent.CreatedBy)
	}

	result, err = repo.TransferMemberAssets(ctx, types.MemberAssetTransferCommand{
		TenantID:     1,
		SourceUserID: "source",
		TargetType:   types.MemberAssetTransferTargetEnterprise,
		Scope:        types.MemberAssetTransferScopeAll,
	})
	if err != nil {
		t.Fatalf("TransferMemberAssets enterprise: %v", err)
	}
	if result.TotalTransferred != 1 || result.KnowledgeBasesTransferred != 1 {
		t.Fatalf("unexpected enterprise transfer result: %+v", result)
	}
	var enterpriseKB types.KnowledgeBase
	if err := db.First(&enterpriseKB, "id = ?", "kb-2").Error; err != nil {
		t.Fatalf("load enterprise knowledge base: %v", err)
	}
	if enterpriseKB.CreatorID != "" {
		t.Fatalf("kb-2 creator = %q, want enterprise ownership", enterpriseKB.CreatorID)
	}

	var untouchedTemporary types.KnowledgeBase
	if err := db.First(&untouchedTemporary, "id = ?", "kb-temp").Error; err != nil {
		t.Fatalf("load temporary knowledge base: %v", err)
	}
	if untouchedTemporary.CreatorID != "source" {
		t.Fatalf("temporary knowledge base creator changed to %q", untouchedTemporary.CreatorID)
	}
	var untouchedBuiltin types.CustomAgent
	if err := db.Where("id = ? AND tenant_id = ?", "agent-builtin", 1).First(&untouchedBuiltin).Error; err != nil {
		t.Fatalf("load built-in agent: %v", err)
	}
	if untouchedBuiltin.CreatedBy != "source" {
		t.Fatalf("built-in agent creator changed to %q", untouchedBuiltin.CreatedBy)
	}
}

func TestTenantMemberRepository_TransferMemberAssetsRollsBackInvalidSelection(t *testing.T) {
	repo, db := newAssetTransferTestRepository(t)
	seedAssetTransferMembers(t, db)
	seedTransferableAssets(t, db)

	_, err := repo.TransferMemberAssets(context.Background(), types.MemberAssetTransferCommand{
		TenantID:         1,
		SourceUserID:     "source",
		TargetType:       types.MemberAssetTransferTargetMember,
		TargetUserID:     "target",
		Scope:            types.MemberAssetTransferScopeSelected,
		KnowledgeBaseIDs: []string{"kb-1"},
		AgentIDs:         []string{"missing-agent"},
	})
	if !errors.Is(err, ErrAssetTransferSelection) {
		t.Fatalf("error = %v, want ErrAssetTransferSelection", err)
	}

	var knowledgeBase types.KnowledgeBase
	if err := db.First(&knowledgeBase, "id = ?", "kb-1").Error; err != nil {
		t.Fatalf("load knowledge base after rollback: %v", err)
	}
	if knowledgeBase.CreatorID != "source" {
		t.Fatalf("partial update was not rolled back, creator=%q", knowledgeBase.CreatorID)
	}
}

func TestTenantMemberRepository_TransferMemberAssetsRequiresActiveTarget(t *testing.T) {
	repo, db := newAssetTransferTestRepository(t)
	seedAssetTransferMembers(t, db)
	seedTransferableAssets(t, db)

	_, err := repo.TransferMemberAssets(context.Background(), types.MemberAssetTransferCommand{
		TenantID:     1,
		SourceUserID: "source",
		TargetType:   types.MemberAssetTransferTargetMember,
		TargetUserID: "suspended",
		Scope:        types.MemberAssetTransferScopeAll,
	})
	if !errors.Is(err, ErrAssetTransferTargetNotActive) {
		t.Fatalf("error = %v, want ErrAssetTransferTargetNotActive", err)
	}
}
