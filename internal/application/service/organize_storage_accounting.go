package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func (s *organizeService) storageAccountingRepository() interfaces.StorageAccountingRepository {
	if s == nil || s.tenantRepo == nil {
		return nil
	}
	repo, _ := s.tenantRepo.(interfaces.StorageAccountingRepository)
	return repo
}

func (s *organizeService) reserveOrganizeMemoryStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	memoryID string,
	fileName string,
	requestedBytes int64,
) error {
	if requestedBytes <= 0 || s == nil || s.tenantRepo == nil {
		return nil
	}

	metadata := storageMetadata(map[string]any{
		"memory_id": memoryID,
		"file_name": fileName,
	})
	if repo := s.storageAccountingRepository(); repo != nil {
		_, err := repo.ReserveStorage(ctx, &types.TenantStorageReservation{
			TenantID:       tenantID,
			ActorUserID:    storageActorUserID(ctx),
			RefNo:          refNo,
			Operation:      "organize_memory_upload",
			RequestedBytes: requestedBytes,
			ExpiresAt:      time.Now().UTC().Add(2 * time.Hour),
			MetadataJSON:   metadata,
		})
		return err
	}

	tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return err
	}
	if tenant != nil && tenant.StorageQuota > 0 && tenant.StorageUsed+requestedBytes > tenant.StorageQuota {
		return types.NewStorageQuotaExceededError()
	}
	return nil
}

func (s *organizeService) commitOrganizeMemoryStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	actualBytes int64,
	memoryID string,
	fileName string,
) error {
	if actualBytes <= 0 || s == nil || s.tenantRepo == nil {
		return nil
	}

	metadata := storageMetadata(map[string]any{
		"memory_id":      memoryID,
		"file_name":      fileName,
		"actual_bytes":   actualBytes,
		"storage_source": "organize_memory_upload",
	})
	if repo := s.storageAccountingRepository(); repo != nil {
		_, err := repo.CommitStorageReservation(ctx, tenantID, refNo, actualBytes, metadata)
		return err
	}
	return s.tenantRepo.AdjustStorageUsed(ctx, tenantID, actualBytes)
}

func (s *organizeService) releaseOrganizeMemoryStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	failureCode string,
) {
	if repo := s.storageAccountingRepository(); repo != nil {
		_ = repo.ReleaseStorageReservation(ctx, tenantID, refNo, failureCode)
	}
}

func (s *organizeService) recordOrganizeMemoryStorageRelease(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	memoryID string,
	fileName string,
	amountBytes int64,
) error {
	if amountBytes <= 0 || s == nil || s.tenantRepo == nil {
		return nil
	}
	return recordStorageDeltaWithRepository(
		ctx,
		s.tenantRepo,
		tenantID,
		refNo,
		"organize_memory_delete",
		-amountBytes,
		map[string]any{
			"memory_id":    memoryID,
			"file_name":    fileName,
			"actual_bytes": amountBytes,
		},
	)
}

func organizeMemoryStorageSize(
	ctx context.Context,
	catalog interfaces.ResourceCatalog,
	metadata types.JSONMap,
	filePath string,
) int64 {
	if catalog != nil {
		if reference, ok := types.ParseResourcePath(strings.TrimSpace(filePath)); ok {
			resource, err := catalog.Resolve(ctx, types.BuildResourcePath(reference))
			if err == nil && resource != nil && resource.Size > 0 {
				return resource.Size
			}
		}
	}

	for _, key := range []string{"storage_size_bytes", "audio_size_bytes", "file_size_bytes"} {
		if value := metadataInt64(metadata, key); value > 0 {
			return value
		}
	}
	return 0
}

func organizeMemoryStorageReleaseRef(memoryID string, updatedAt time.Time, size int64) string {
	return fmt.Sprintf(
		"organize:memory_delete:%s:%d:%d",
		strings.TrimSpace(memoryID),
		updatedAt.UnixNano(),
		size,
	)
}

func metadataInt64(metadata types.JSONMap, key string) int64 {
	switch value := metadata[key].(type) {
	case int:
		return int64(value)
	case int8:
		return int64(value)
	case int16:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case uint:
		return int64(value)
	case uint8:
		return int64(value)
	case uint16:
		return int64(value)
	case uint32:
		return int64(value)
	case uint64:
		if value > uint64(^uint64(0)>>1) {
			return 0
		}
		return int64(value)
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}
