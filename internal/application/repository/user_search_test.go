package repository

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserRepositorySearchUsersMatchesUsernameAndPhoneOnSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:user_search?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&types.Tenant{}, &types.User{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewUserRepository(db)
	user := &types.User{
		ID:           "user-phone",
		Username:     "2333",
		Email:        "13258978277",
		PasswordHash: "hashed",
		IsActive:     true,
	}
	if err := repo.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	inactive := &types.User{
		ID:           "inactive-phone",
		Username:     "inactive",
		Email:        "15500001111",
		PasswordHash: "hashed",
		IsActive:     false,
	}
	if err := repo.CreateUser(context.Background(), inactive); err != nil {
		t.Fatalf("CreateUser inactive: %v", err)
	}

	for _, query := range []string{"2333", "13258978277", "258978"} {
		users, err := repo.SearchUsers(context.Background(), query, 10)
		if err != nil {
			t.Fatalf("SearchUsers(%q): %v", query, err)
		}
		if len(users) != 1 || users[0].ID != user.ID {
			t.Fatalf("SearchUsers(%q) = %+v, want user-phone", query, users)
		}
	}

	users, err := repo.SearchUsers(context.Background(), "15500001111", 10)
	if err != nil {
		t.Fatalf("SearchUsers inactive phone: %v", err)
	}
	if len(users) != 1 || users[0].ID != inactive.ID {
		t.Fatalf("SearchUsers inactive phone = %+v, want inactive-phone", users)
	}
}
