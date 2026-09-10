package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

func (s *organizeService) resolveOrganizeFileService(
	ctx context.Context,
	tenantID uint64,
	filePath string,
) (interfaces.FileService, error) {
	if s == nil {
		return nil, fmt.Errorf("organize service is not configured")
	}
	if s.storageResolver == nil {
		if s.fileService == nil {
			return nil, fmt.Errorf("file service is not configured")
		}
		return s.fileService, nil
	}

	tenant, err := s.resolveOrganizeTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	backendID, provider, err := s.organizeStorageSelection(ctx, filePath)
	if err != nil {
		return nil, err
	}
	fileService, _, err := s.storageResolver.ResolveFileService(
		ctx,
		tenant,
		backendID,
		provider,
		strings.TrimSpace(os.Getenv("LOCAL_STORAGE_BASE_DIR")),
	)
	if err != nil {
		return nil, fmt.Errorf("resolve organize storage: %w", err)
	}
	if fileService == nil {
		return nil, fmt.Errorf("resolve organize storage returned nil file service")
	}
	return fileService, nil
}

func (s *organizeService) resolveOrganizeTenant(ctx context.Context, tenantID uint64) (*types.Tenant, error) {
	if tenant, ok := ctx.Value(types.TenantInfoContextKey).(*types.Tenant); ok && tenant != nil {
		if tenant.ID == 0 {
			copy := *tenant
			copy.ID = tenantID
			return &copy, nil
		}
		if tenant.ID == tenantID {
			return tenant, nil
		}
	}
	if s.tenantRepo != nil {
		tenant, err := s.tenantRepo.GetTenantByID(ctx, tenantID)
		if err != nil {
			return nil, fmt.Errorf("load organize tenant %d: %w", tenantID, err)
		}
		if tenant != nil {
			return tenant, nil
		}
	}
	return nil, fmt.Errorf("workspace context missing for tenant %d", tenantID)
}

func (s *organizeService) organizeStorageSelection(
	ctx context.Context,
	filePath string,
) (backendID, provider string, err error) {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return "", "", nil
	}

	if _, ok := types.ParseResourcePath(filePath); ok {
		if s.resourceCatalog == nil {
			return "", "", fmt.Errorf("resource catalog is not configured for %q", filePath)
		}
		resource, resolveErr := s.resourceCatalog.Resolve(ctx, filePath)
		if resolveErr != nil {
			return "", "", fmt.Errorf("resolve stored resource %q: %w", filePath, resolveErr)
		}
		if resource == nil {
			return "", "", fmt.Errorf("stored resource %q not found", filePath)
		}
		return strings.TrimSpace(resource.StorageBackendID), strings.ToLower(strings.TrimSpace(resource.Provider)), nil
	}

	if backendID, providerPath, ok := types.ParseStorageBackendPath(filePath); ok {
		return backendID, types.ParseProviderScheme(providerPath), nil
	}
	return "", types.ParseProviderScheme(filePath), nil
}

// deleteOrganizeStoredFile removes the persistent source owned by an organize
// item. Mobile-only paths such as audio_local_path/local_audio_path are
// intentionally ignored because they point to the client device, not the
// server storage backend.
func (s *organizeService) deleteOrganizeStoredFile(
	ctx context.Context,
	tenantID uint64,
	ownerType, ownerID string,
	metadata types.JSONMap,
	keys ...string,
) error {
	filePath := organizeStoredFilePath(metadata, keys...)
	if filePath == "" {
		return nil
	}
	fileService, err := s.resolveOrganizeFileService(ctx, tenantID, filePath)
	if err != nil {
		return fmt.Errorf("resolve %s file service for %q: %w", ownerType, filePath, err)
	}
	if err := fileService.DeleteFile(ctx, filePath); err != nil {
		return fmt.Errorf("delete %s file for %s %q: %w", ownerType, ownerID, filePath, err)
	}
	return nil
}

func organizeStoredFilePath(metadata types.JSONMap, keys ...string) string {
	for _, key := range keys {
		value, ok := metadata[key].(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		// Accept stable resource refs and provider paths, including scoped
		// storage://backend-id/provider://... paths. Reject arbitrary local
		// client paths from mobile metadata.
		if _, ok := types.ParseResourcePath(value); ok ||
			types.ParseProviderScheme(value) != "" {
			return value
		}
	}
	return ""
}
