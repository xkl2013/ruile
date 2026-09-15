package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/storageallowlist"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

const knowledgeBaseWorkspaceConfigVersion = "workspace-default-v1"

type knowledgeBaseDefaultsService struct {
	tenantRepo      interfaces.TenantRepository
	knowledgeRepo   interfaces.KnowledgeRepository
	kbRepo          interfaces.KnowledgeBaseRepository
	modelService    interfaces.ModelService
	storageResolver interfaces.StorageBackendResolver
}

func NewKnowledgeBaseDefaultsService(
	tenantRepo interfaces.TenantRepository,
	knowledgeRepo interfaces.KnowledgeRepository,
	kbRepo interfaces.KnowledgeBaseRepository,
	modelService interfaces.ModelService,
	storageResolver interfaces.StorageBackendResolver,
) interfaces.KnowledgeBaseDefaultsService {
	return &knowledgeBaseDefaultsService{
		tenantRepo:      tenantRepo,
		knowledgeRepo:   knowledgeRepo,
		kbRepo:          kbRepo,
		modelService:    modelService,
		storageResolver: storageResolver,
	}
}

func (s *knowledgeBaseDefaultsService) Get(ctx context.Context) (*types.KnowledgeBaseDefaultsConfig, error) {
	tenant, ok := types.TenantInfoFromContext(ctx)
	if !ok || tenant == nil {
		return nil, errors.New("workspace context missing")
	}
	if tenant.KnowledgeBaseDefaultsConfig != nil {
		return tenant.KnowledgeBaseDefaultsConfig, nil
	}

	kbs, err := s.kbRepo.ListKnowledgeBasesByTenantID(ctx, tenant.ID)
	if err != nil {
		return nil, err
	}
	for _, kb := range kbs {
		if kb == nil || kb.IsTemporary {
			continue
		}
		kb.EnsureDefaults()
		return types.KnowledgeBaseDefaultsConfigFromKnowledgeBase(kb), nil
	}
	return &types.KnowledgeBaseDefaultsConfig{
		ChunkingConfig:   types.ChunkingConfig{ChunkSize: 512, ChunkOverlap: 80},
		IndexingStrategy: types.DefaultIndexingStrategy(),
		StorageProvider:  storageallowlist.FirstAllowed(),
	}, nil
}

func (s *knowledgeBaseDefaultsService) Update(
	ctx context.Context,
	config *types.KnowledgeBaseDefaultsConfig,
) (*types.KnowledgeBaseDefaultsConfig, int, error) {
	if config == nil {
		return nil, 0, errors.New("knowledge base defaults are required")
	}
	tenant, ok := types.TenantInfoFromContext(ctx)
	if !ok || tenant == nil {
		return nil, 0, errors.New("workspace context missing")
	}

	normalized := *config
	normalizeKnowledgeBaseDefaults(&normalized)
	if err := s.validate(ctx, tenant, &normalized); err != nil {
		return nil, 0, err
	}

	tenant.KnowledgeBaseDefaultsConfig = &normalized
	tenant.UpdatedAt = time.Now()
	if err := s.tenantRepo.UpdateTenant(ctx, tenant); err != nil {
		return nil, 0, err
	}

	// Existing knowledge bases keep their own effective configuration. The
	// workspace record is a default for future knowledge bases only; an
	// explicit migration must be used for any existing data rewrite.
	return &normalized, 0, nil
}

func normalizeKnowledgeBaseDefaults(config *types.KnowledgeBaseDefaultsConfig) {
	if config == nil {
		return
	}
	config.SummaryModelID = strings.TrimSpace(config.SummaryModelID)
	config.EmbeddingModelID = strings.TrimSpace(config.EmbeddingModelID)
	config.StorageProvider = strings.ToLower(strings.TrimSpace(config.StorageProvider))
	config.StorageBackendID = strings.TrimSpace(config.StorageBackendID)
	if config.StorageProvider == "" {
		config.StorageProvider = storageallowlist.FirstAllowed()
	}
	if config.ChunkingConfig.ChunkSize <= 0 {
		config.ChunkingConfig.ChunkSize = 512
	}
	if config.ChunkingConfig.ChunkOverlap < 0 {
		config.ChunkingConfig.ChunkOverlap = 0
	}
	if len(config.ChunkingConfig.Separators) == 0 {
		config.ChunkingConfig.Separators = append([]string(nil), defaultKnowledgeBaseSeparators...)
	}
	if config.ChunkingConfig.ParentChunkSize <= 0 {
		config.ChunkingConfig.ParentChunkSize = 4096
	}
	if config.ChunkingConfig.ChildChunkSize <= 0 {
		config.ChunkingConfig.ChildChunkSize = 384
	}
	if config.IndexingStrategy.IsZero() {
		config.IndexingStrategy = types.DefaultIndexingStrategy()
	}
}

func (s *knowledgeBaseDefaultsService) validate(
	ctx context.Context,
	tenant *types.Tenant,
	config *types.KnowledgeBaseDefaultsConfig,
) error {
	if config.SummaryModelID == "" {
		return errors.New("LLM model is required")
	}
	if config.IndexingStrategy.NeedsEmbedding() && config.EmbeddingModelID == "" {
		return errors.New("embedding model is required when retrieval indexing is enabled")
	}
	if s.modelService != nil {
		for label, id := range map[string]string{
			"LLM":       config.SummaryModelID,
			"Embedding": config.EmbeddingModelID,
			"VLM":       config.VLMConfig.ModelID,
			"OCR":       config.OCRConfig.ModelID,
			"ASR":       config.ASRConfig.ModelID,
		} {
			if id == "" {
				continue
			}
			model, err := s.modelService.GetModelByID(ctx, id)
			if err != nil || model == nil {
				return errors.New(label + " model does not exist")
			}
		}
	}
	if config.StorageProvider != "" && !storageallowlist.IsAllowed(config.StorageProvider) {
		return errors.New("storage provider is not allowed by STORAGE_ALLOW_LIST")
	}
	if config.StorageBackendID != "" && s.storageResolver != nil {
		backend, err := s.storageResolver.ResolveBackend(ctx, tenant, config.StorageBackendID, "")
		if err != nil || backend == nil {
			return errors.New("storage backend is unavailable")
		}
	}
	return nil
}
