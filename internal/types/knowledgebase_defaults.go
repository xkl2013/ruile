package types

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
)

// KnowledgeBaseDefaultsConfig is the advanced configuration owned by the
// current workspace. It is applied to every non-temporary knowledge base in
// that workspace and inherited by newly-created knowledge bases.
type KnowledgeBaseDefaultsConfig struct {
	SummaryModelID           string                    `json:"summary_model_id"`
	EmbeddingModelID         string                    `json:"embedding_model_id"`
	VLMConfig                VLMConfig                 `json:"vlm_config"`
	OCRConfig                OCRConfig                 `json:"ocr_config"`
	ASRConfig                ASRConfig                 `json:"asr_config"`
	ChunkingConfig           ChunkingConfig            `json:"chunking_config"`
	StorageProvider          string                    `json:"storage_provider"`
	StorageBackendID         string                    `json:"storage_backend_id"`
	ExtractConfig            *ExtractConfig            `json:"extract_config,omitempty"`
	QuestionGenerationConfig *QuestionGenerationConfig `json:"question_generation_config,omitempty"`
	WikiConfig               *WikiConfig               `json:"wiki_config,omitempty"`
	FAQConfig                *FAQConfig                `json:"faq_config,omitempty"`
	IndexingStrategy         IndexingStrategy          `json:"indexing_strategy"`
}

// Value implements driver.Valuer for the tenant JSON column.
func (c KnowledgeBaseDefaultsConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements sql.Scanner for the tenant JSON column.
func (c *KnowledgeBaseDefaultsConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		if len(v) == 0 {
			return nil
		}
		return json.Unmarshal(v, c)
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return json.Unmarshal([]byte(v), c)
	default:
		return nil
	}
}

// KnowledgeBaseDefaultsConfigFromKnowledgeBase creates a workspace-default
// snapshot from an existing knowledge base. It is used to bootstrap the
// unified workspace configuration UI on deployments that predate this field.
func KnowledgeBaseDefaultsConfigFromKnowledgeBase(kb *KnowledgeBase) *KnowledgeBaseDefaultsConfig {
	if kb == nil {
		return &KnowledgeBaseDefaultsConfig{}
	}
	cfg := &KnowledgeBaseDefaultsConfig{
		SummaryModelID:           kb.SummaryModelID,
		EmbeddingModelID:         kb.EmbeddingModelID,
		VLMConfig:                kb.VLMConfig,
		OCRConfig:                kb.OCRConfig,
		ASRConfig:                kb.ASRConfig,
		ChunkingConfig:           kb.ChunkingConfig,
		StorageProvider:          kb.GetStorageProvider(),
		ExtractConfig:            kb.ExtractConfig,
		QuestionGenerationConfig: kb.QuestionGenerationConfig,
		WikiConfig:               kb.WikiConfig,
		FAQConfig:                kb.FAQConfig,
		IndexingStrategy:         kb.IndexingStrategy,
	}
	if kb.StorageBackendID != nil {
		cfg.StorageBackendID = strings.TrimSpace(*kb.StorageBackendID)
	}
	return cfg
}

// ApplyToKnowledgeBase copies the workspace-owned advanced fields onto a
// concrete knowledge base while preserving its identity, ownership, content
// and directory metadata.
func (c *KnowledgeBaseDefaultsConfig) ApplyToKnowledgeBase(kb *KnowledgeBase) {
	if c == nil || kb == nil {
		return
	}
	kb.SummaryModelID = strings.TrimSpace(c.SummaryModelID)
	kb.EmbeddingModelID = strings.TrimSpace(c.EmbeddingModelID)
	kb.VLMConfig = c.VLMConfig
	kb.OCRConfig = c.OCRConfig
	kb.ASRConfig = c.ASRConfig
	kb.ChunkingConfig = c.ChunkingConfig
	kb.ExtractConfig = c.ExtractConfig
	kb.QuestionGenerationConfig = c.QuestionGenerationConfig
	kb.WikiConfig = c.WikiConfig
	kb.IndexingStrategy = c.IndexingStrategy
	if kb.Type == KnowledgeBaseTypeFAQ {
		// FAQ settings are not edited on the unified configuration page.
		// Keep the existing per-FAQ value when the global payload omits them.
		if c.FAQConfig != nil {
			kb.FAQConfig = c.FAQConfig
		}
	} else {
		kb.FAQConfig = nil
	}
	if strings.TrimSpace(c.StorageBackendID) != "" {
		id := strings.TrimSpace(c.StorageBackendID)
		kb.StorageBackendID = &id
	} else {
		kb.StorageBackendID = nil
	}
	provider := strings.ToLower(strings.TrimSpace(c.StorageProvider))
	if provider != "" {
		kb.SetStorageProvider(provider)
	}
}
