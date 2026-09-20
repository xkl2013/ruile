package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositorySystemUserLifecycleGuardsAndHardDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_lifecycle?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.Tenant{},
		&types.User{},
		&types.AuthToken{},
		&types.TenantMember{},
		&types.KnowledgeBase{},
		&types.CustomAgent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewUserRepository(db)
	users := []*types.User{
		{ID: "admin-1", Username: "admin1", Email: "admin1@example.com", PasswordHash: "hashed", IsActive: true, IsSystemAdmin: true},
		{ID: "admin-2", Username: "admin2", Email: "admin2@example.com", PasswordHash: "hashed", IsActive: true, IsSystemAdmin: true},
		{ID: "member-1", Username: "member1", Email: "member1@example.com", PasswordHash: "hashed", IsActive: true},
	}
	for _, user := range users {
		if err := repo.CreateUser(context.Background(), user); err != nil {
			t.Fatalf("create %s: %v", user.ID, err)
		}
	}

	updated, err := repo.SetSystemUserActive(context.Background(), "admin-2", "admin-1", false)
	if err != nil {
		t.Fatalf("disable second admin: %v", err)
	}
	if updated.IsActive {
		t.Fatal("second admin should be disabled")
	}
	_, err = repo.SetSystemUserActive(context.Background(), "admin-1", "admin-2", false)
	if !errors.Is(err, ErrLastActiveSystemAdmin) {
		t.Fatalf("disable last active admin error=%v", err)
	}

	deleted, err := repo.DeleteSystemUser(context.Background(), "member-1", "admin-1")
	if err != nil {
		t.Fatalf("delete member: %v", err)
	}
	if deleted.ID != "member-1" {
		t.Fatalf("deleted user=%+v", deleted)
	}
	if _, err := repo.GetUserByID(context.Background(), "member-1"); !errors.Is(err, ErrUserNotFound) {
		t.Fatalf("hard-deleted user lookup error=%v", err)
	}
	var deletedRows int64
	if err := db.Unscoped().Model(&types.User{}).Where("id = ?", "member-1").Count(&deletedRows).Error; err != nil {
		t.Fatalf("count deleted user: %v", err)
	}
	if deletedRows != 0 {
		t.Fatalf("deleted user row still exists: %d", deletedRows)
	}
	if err := repo.CreateUser(context.Background(), &types.User{
		ID:           "member-1-recreated",
		Username:     "member1",
		Email:        "member1@example.com",
		PasswordHash: "hashed",
		IsActive:     true,
	}); err != nil {
		t.Fatalf("recreate deleted user identity: %v", err)
	}

	if _, err := repo.SetSystemUserActive(context.Background(), "admin-1", "admin-1", false); !errors.Is(err, ErrCannotDisableSelf) {
		t.Fatalf("self-disable error=%v", err)
	}
	if _, err := repo.DeleteSystemUser(context.Background(), "admin-1", "admin-1"); !errors.Is(err, ErrCannotDeleteSelf) {
		t.Fatalf("self-delete error=%v", err)
	}

	if _, err = repo.DeleteSystemUser(context.Background(), "admin-1", "admin-2"); err != nil {
		t.Fatalf("delete one of two system admins: %v", err)
	}
	if _, err = repo.DeleteSystemUser(context.Background(), "admin-2", "admin-1"); !errors.Is(err, ErrLastSystemAdmin) {
		t.Fatalf("delete last system admin error=%v", err)
	}
}

func TestUserRepositoryPurgesLegacySoftDeletedIdentity(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_legacy_tombstone?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.Tenant{},
		&types.User{},
		&types.AuthToken{},
		&types.TenantMember{},
		&types.KnowledgeBase{},
		&types.CustomAgent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewUserRepository(db)
	legacy := &types.User{
		ID:           "legacy-user",
		Username:     "legacy",
		Email:        "13901156168",
		PasswordHash: "hashed",
		IsActive:     true,
	}
	if err := repo.CreateUser(context.Background(), legacy); err != nil {
		t.Fatalf("create legacy user: %v", err)
	}
	if err := db.Create(&types.AuthToken{
		ID:        "legacy-token",
		UserID:    legacy.ID,
		Token:     "token",
		TokenType: "refresh_token",
	}).Error; err != nil {
		t.Fatalf("create legacy token: %v", err)
	}
	if err := db.Delete(legacy).Error; err != nil {
		t.Fatalf("soft-delete legacy user: %v", err)
	}

	if err := repo.PurgeDeletedUserByIdentity(context.Background(), legacy.Email, legacy.Username); err != nil {
		t.Fatalf("purge legacy identity: %v", err)
	}
	if err := repo.CreateUser(context.Background(), &types.User{
		ID:           "new-user",
		Username:     "legacy",
		Email:        "13901156168",
		PasswordHash: "hashed",
		IsActive:     true,
	}); err != nil {
		t.Fatalf("recreate legacy identity: %v", err)
	}
	var tokenCount int64
	if err := db.Unscoped().Model(&types.AuthToken{}).Where("user_id = ?", legacy.ID).Count(&tokenCount).Error; err != nil {
		t.Fatalf("count legacy tokens: %v", err)
	}
	if tokenCount != 0 {
		t.Fatalf("legacy tokens remain: %d", tokenCount)
	}
}

func TestUserRepositoryDeleteSystemUserRequiresEnterpriseAssetTransfer(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_asset_transfer_guard?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&types.Tenant{},
		&types.User{},
		&types.AuthToken{},
		&types.TenantMember{},
		&types.KnowledgeBase{},
		&types.CustomAgent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	organization := types.SpaceTypeOrganization
	personal := types.SpaceTypePersonal
	if err := db.Create(&[]types.Tenant{
		{ID: 1, Name: "企业", SpaceType: &organization},
		{ID: 2, Name: "个人空间", SpaceType: &personal},
	}).Error; err != nil {
		t.Fatalf("seed tenants: %v", err)
	}

	repo := NewUserRepository(db)
	for _, user := range []*types.User{
		{ID: "admin", Username: "admin", Email: "admin@example.com", PasswordHash: "hashed", IsActive: true, IsSystemAdmin: true},
		{ID: "source", Username: "source", Email: "source@example.com", PasswordHash: "hashed", IsActive: true},
		{ID: "target", Username: "target", Email: "target@example.com", PasswordHash: "hashed", IsActive: true},
	} {
		if err := repo.CreateUser(context.Background(), user); err != nil {
			t.Fatalf("create %s: %v", user.ID, err)
		}
	}
	if err := db.Create(&[]types.KnowledgeBase{
		{ID: "enterprise-kb", Name: "企业知识库", TenantID: 1, CreatorID: "source"},
		{ID: "personal-kb", Name: "个人知识库", TenantID: 2, CreatorID: "source"},
	}).Error; err != nil {
		t.Fatalf("seed knowledge bases: %v", err)
	}
	if err := db.Create(&types.CustomAgent{
		ID:        "enterprise-agent",
		Name:      "企业智能体",
		TenantID:  1,
		CreatedBy: "source",
	}).Error; err != nil {
		t.Fatalf("seed custom agent: %v", err)
	}

	if _, err := repo.DeleteSystemUser(context.Background(), "source", "admin"); !errors.Is(err, ErrUserHasEnterpriseAssets) {
		t.Fatalf("delete with enterprise assets error=%v", err)
	}

	if err := db.Model(&types.KnowledgeBase{}).
		Where("id = ?", "enterprise-kb").
		Update("creator_id", "target").Error; err != nil {
		t.Fatalf("transfer enterprise knowledge base: %v", err)
	}
	if err := db.Model(&types.CustomAgent{}).
		Where("id = ? AND tenant_id = ?", "enterprise-agent", 1).
		Update("created_by", "target").Error; err != nil {
		t.Fatalf("transfer enterprise agent: %v", err)
	}

	deleted, err := repo.DeleteSystemUser(context.Background(), "source", "admin")
	if err != nil {
		t.Fatalf("delete after enterprise transfer: %v", err)
	}
	if deleted.ID != "source" {
		t.Fatalf("deleted user = %+v", deleted)
	}
	var personalKB types.KnowledgeBase
	if err := db.First(&personalKB, "id = ?", "personal-kb").Error; err != nil {
		t.Fatalf("load retained personal knowledge base: %v", err)
	}
	if personalKB.CreatorID != "source" {
		t.Fatalf("personal asset should not be rewritten, creator=%q", personalKB.CreatorID)
	}
}
