package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func (s *knowledgeService) storageAccountingRepository() interfaces.StorageAccountingRepository {
	if s == nil || s.tenantRepo == nil {
		return nil
	}
	repo, _ := s.tenantRepo.(interfaces.StorageAccountingRepository)
	return repo
}

func storageActorUserID(ctx context.Context) string {
	userID, _ := types.UserIDFromContext(ctx)
	return strings.TrimSpace(userID)
}

func storageMetadata(value map[string]any) types.JSON {
	if len(value) == 0 {
		return types.JSON([]byte("{}"))
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return types.JSON([]byte("{}"))
	}
	return types.JSON(raw)
}

func knowledgeStorageRef(operation, knowledgeID string, attempt int) string {
	if attempt < 0 {
		attempt = 0
	}
	return fmt.Sprintf("knowledge:%s:%s:%d", operation, knowledgeID, attempt)
}

func (s *knowledgeService) reserveKnowledgeStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	operation string,
	requestedBytes int64,
	metadata map[string]any,
) error {
	if requestedBytes <= 0 {
		return nil
	}
	repo := s.storageAccountingRepository()
	if repo == nil {
		tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
		if err != nil {
			return err
		}
		if tenant.StorageQuota > 0 && tenant.StorageUsed+requestedBytes > tenant.StorageQuota {
			return types.NewStorageQuotaExceededError()
		}
		return nil
	}
	_, err := repo.ReserveStorage(ctx, &types.TenantStorageReservation{
		TenantID:       tenantID,
		ActorUserID:    storageActorUserID(ctx),
		RefNo:          refNo,
		Operation:      operation,
		RequestedBytes: requestedBytes,
		ExpiresAt:      time.Now().UTC().Add(2 * time.Hour),
		MetadataJSON:   storageMetadata(metadata),
	})
	return err
}

func (s *knowledgeService) commitKnowledgeStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	actualBytes int64,
	metadata map[string]any,
) error {
	if actualBytes <= 0 {
		return nil
	}
	repo := s.storageAccountingRepository()
	if repo == nil {
		return s.tenantRepo.AdjustStorageUsed(ctx, tenantID, actualBytes)
	}
	_, err := repo.CommitStorageReservation(
		ctx,
		tenantID,
		refNo,
		actualBytes,
		storageMetadata(metadata),
	)
	return err
}

func (s *knowledgeService) releaseKnowledgeStorage(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	failureCode string,
) {
	repo := s.storageAccountingRepository()
	if repo == nil {
		return
	}
	_ = repo.ReleaseStorageReservation(ctx, tenantID, refNo, failureCode)
}

func (s *knowledgeService) recordKnowledgeStorageDelta(
	ctx context.Context,
	tenantID uint64,
	refNo string,
	operation string,
	deltaBytes int64,
	metadata map[string]any,
) error {
	return recordStorageDeltaWithRepository(
		ctx,
		s.tenantRepo,
		tenantID,
		refNo,
		operation,
		deltaBytes,
		metadata,
	)
}

func recordStorageDeltaWithRepository(
	ctx context.Context,
	tenantRepo interfaces.TenantRepository,
	tenantID uint64,
	refNo string,
	operation string,
	deltaBytes int64,
	metadata map[string]any,
) error {
	if deltaBytes == 0 {
		return nil
	}
	if tenantRepo == nil {
		return fmt.Errorf("storage: tenant repository is unavailable")
	}
	repo, _ := tenantRepo.(interfaces.StorageAccountingRepository)
	if repo == nil {
		return tenantRepo.AdjustStorageUsed(ctx, tenantID, deltaBytes)
	}
	_, err := repo.RecordStorageTransaction(ctx, &types.TenantStorageTransaction{
		TenantID:     tenantID,
		ActorUserID:  storageActorUserID(ctx),
		RefNo:        refNo,
		Operation:    operation,
		AmountBytes:  deltaBytes,
		MetadataJSON: storageMetadata(metadata),
	})
	return err
}
