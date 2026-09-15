package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

var defaultKnowledgeBaseSeparators = []string{"\n\n", "\n", "。", "！", "？", ";", "；"}

// applyKnowledgeBaseCreationDefaults resolves the backend-owned defaults used
// by the minimal knowledge-base creation API. Existing explicit values remain
// untouched so internal workflows such as evaluation and temporary web-search
// KBs keep their current behavior.
func (s *knowledgeBaseService) applyKnowledgeBaseCreationDefaults(
	ctx context.Context,
	kb *types.KnowledgeBase,
) error {
	if kb == nil {
		return fmt.Errorf("knowledge base is required")
	}

	bareUserCreate := !kb.IsTemporary &&
		strings.TrimSpace(kb.EmbeddingModelID) == "" &&
		strings.TrimSpace(kb.SummaryModelID) == ""

	kb.EnsureDefaults()
	if bareUserCreate {
		if tenant, ok := types.TenantInfoFromContext(ctx); ok &&
			tenant != nil &&
			tenant.KnowledgeBaseDefaultsConfig != nil {
			tenant.KnowledgeBaseDefaultsConfig.ApplyToKnowledgeBase(kb)
		}
		applyKnowledgeBaseChunkingDefaults(kb)
		if kb.QuestionGenerationConfig == nil {
			kb.QuestionGenerationConfig = &types.QuestionGenerationConfig{
				Enabled:       true,
				QuestionCount: 3,
			}
		}

		if s.modelService != nil {
			models, err := s.modelService.ListModels(ctx)
			if err != nil {
				return fmt.Errorf("resolve knowledge base default models: %w", err)
			}
			if err := applyDefaultKnowledgeBaseModels(kb, models); err != nil {
				return err
			}
		}
	}

	if kb.ConfigSource == nil || strings.TrimSpace(string(*kb.ConfigSource)) == "" {
		source := types.KnowledgeBaseConfigSourceDefault
		if tenant, ok := types.TenantInfoFromContext(ctx); ok &&
			tenant != nil &&
			tenant.KnowledgeBaseDefaultsConfig != nil {
			source = types.KnowledgeBaseConfigSourceWorkspaceDefault
		}
		kb.ConfigSource = &source
	}
	if kb.ConfigVersion == nil || strings.TrimSpace(*kb.ConfigVersion) == "" {
		version := knowledgeBaseDefaultConfigVersion
		if kb.ConfigSource != nil &&
			*kb.ConfigSource == types.KnowledgeBaseConfigSourceWorkspaceDefault {
			version = knowledgeBaseWorkspaceConfigVersion
		}
		kb.ConfigVersion = &version
	}
	return nil
}

func applyKnowledgeBaseChunkingDefaults(kb *types.KnowledgeBase) {
	cfg := &kb.ChunkingConfig
	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = 512
	}
	if cfg.ChunkOverlap <= 0 {
		cfg.ChunkOverlap = 80
	}
	if len(cfg.Separators) == 0 {
		cfg.Separators = append([]string(nil), defaultKnowledgeBaseSeparators...)
	}
	if cfg.ParentChunkSize <= 0 {
		cfg.ParentChunkSize = 4096
	}
	if cfg.ChildChunkSize <= 0 {
		cfg.ChildChunkSize = 384
	}
	if strings.TrimSpace(cfg.Strategy) == "" {
		cfg.Strategy = "auto"
	}
}

func applyDefaultKnowledgeBaseModels(kb *types.KnowledgeBase, models []*types.Model) error {
	if kb.IndexingStrategy.NeedsEmbedding() && strings.TrimSpace(kb.EmbeddingModelID) == "" {
		model := selectDefaultKnowledgeBaseModel(models, types.ModelTypeEmbedding)
		if model == nil {
			return fmt.Errorf("no active embedding model is configured")
		}
		kb.EmbeddingModelID = model.ID
	}
	if strings.TrimSpace(kb.SummaryModelID) == "" {
		model := selectDefaultKnowledgeBaseModel(models, types.ModelTypeKnowledgeQA)
		if model == nil {
			return fmt.Errorf("no active knowledge QA model is configured")
		}
		kb.SummaryModelID = model.ID
	}
	return nil
}

func selectDefaultKnowledgeBaseModel(models []*types.Model, modelType types.ModelType) *types.Model {
	var fallback *types.Model
	for _, model := range models {
		if model == nil || model.Type != modelType || model.Status != types.ModelStatusActive {
			continue
		}
		if model.IsDefault {
			return model
		}
		if fallback == nil {
			fallback = model
		}
	}
	return fallback
}
