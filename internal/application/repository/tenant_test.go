package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database with tenant table.
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&types.Tenant{},
		&types.TenantMember{},
		&types.TenantStorageReservation{},
		&types.TenantStorageTransaction{},
	))
	return db
}

func TestDeleteTenant_SoftDeletesMemberships(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := NewTenantRepository(db)

	tenant := &types.Tenant{Name: "gone", Status: "active"}
	require.NoError(t, db.Create(tenant).Error)

	member := &types.TenantMember{
		UserID:   "user-1",
		TenantID: tenant.ID,
		Role:     types.TenantRoleOwner,
		Status:   types.TenantMemberStatusActive,
	}
	require.NoError(t, db.Create(member).Error)

	require.NoError(t, repo.DeleteTenant(ctx, tenant.ID))

	var tenantCount int64
	require.NoError(t, db.Model(&types.Tenant{}).Count(&tenantCount).Error)
	assert.Equal(t, int64(0), tenantCount)

	var memberCount int64
	require.NoError(t, db.Model(&types.TenantMember{}).Count(&memberCount).Error)
	assert.Equal(t, int64(0), memberCount)

	// Unscoped: rows still exist but are soft-deleted.
	var rawTenantCount int64
	require.NoError(t, db.Unscoped().Model(&types.Tenant{}).Count(&rawTenantCount).Error)
	assert.Equal(t, int64(1), rawTenantCount)

	var rawMemberCount int64
	require.NoError(t, db.Unscoped().Model(&types.TenantMember{}).Count(&rawMemberCount).Error)
	assert.Equal(t, int64(1), rawMemberCount)
}

func TestStorageAccountingReservationCommitAndIdempotency(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	baseRepo := NewTenantRepository(db)
	repo, ok := baseRepo.(interfaces.StorageAccountingRepository)
	require.True(t, ok)

	tenant := &types.Tenant{
		Name:         "storage-test",
		Status:       "active",
		StorageQuota: 100,
		StorageUsed:  20,
	}
	require.NoError(t, db.Create(tenant).Error)

	first, err := repo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenant.ID,
		ActorUserID:    "user-1",
		RefNo:          "storage:first",
		Operation:      "knowledge_index",
		RequestedBytes: 60,
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	assert.Equal(t, types.StorageReservationStatusReserved, first.Status)

	_, err = repo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenant.ID,
		ActorUserID:    "user-2",
		RefNo:          "storage:overflow",
		Operation:      "knowledge_index",
		RequestedBytes: 21,
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	var quotaErr *types.StorageQuotaExceededError
	require.True(t, errors.As(err, &quotaErr))

	transaction, err := repo.CommitStorageReservation(
		ctx,
		tenant.ID,
		"storage:first",
		55,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, int64(55), transaction.AmountBytes)
	assert.Equal(t, int64(75), transaction.StorageUsedAfterBytes)

	retried, err := repo.CommitStorageReservation(
		ctx,
		tenant.ID,
		"storage:first",
		55,
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, transaction.ID, retried.ID)

	var refreshed types.Tenant
	require.NoError(t, db.First(&refreshed, tenant.ID).Error)
	assert.Equal(t, int64(75), refreshed.StorageUsed)

	release, err := repo.RecordStorageTransaction(ctx, &types.TenantStorageTransaction{
		TenantID:     tenant.ID,
		ActorUserID:  "user-1",
		RefNo:        "storage:delete",
		Operation:    "knowledge_delete",
		AmountBytes:  -30,
		MetadataJSON: types.JSON([]byte("{}")),
	})
	require.NoError(t, err)
	assert.Equal(t, int64(45), release.StorageUsedAfterBytes)

	retriedRelease, err := repo.RecordStorageTransaction(ctx, &types.TenantStorageTransaction{
		TenantID:     tenant.ID,
		RefNo:        "storage:delete",
		Operation:    "knowledge_delete",
		AmountBytes:  -30,
		MetadataJSON: types.JSON([]byte("{}")),
	})
	require.NoError(t, err)
	assert.Equal(t, release.ID, retriedRelease.ID)
	require.NoError(t, db.First(&refreshed, tenant.ID).Error)
	assert.Equal(t, int64(45), refreshed.StorageUsed)
}

func TestReleasedStorageReservationStopsBlockingQuota(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	baseRepo := NewTenantRepository(db)
	repo := baseRepo.(interfaces.StorageAccountingRepository)

	tenant := &types.Tenant{
		Name:         "storage-release-test",
		Status:       "active",
		StorageQuota: 100,
	}
	require.NoError(t, db.Create(tenant).Error)

	_, err := repo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenant.ID,
		RefNo:          "storage:released",
		Operation:      "knowledge_index",
		RequestedBytes: 100,
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	require.NoError(t, repo.ReleaseStorageReservation(
		ctx,
		tenant.ID,
		"storage:released",
		"upstream_failed",
	))

	_, err = repo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenant.ID,
		RefNo:          "storage:replacement",
		Operation:      "knowledge_index",
		RequestedBytes: 100,
		ExpiresAt:      time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
}
