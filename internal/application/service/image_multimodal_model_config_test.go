package service

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/vlm"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

type captureMultimodalModelService struct {
	interfaces.ModelService
	requestedVLMModelID string
}

func (s *captureMultimodalModelService) GetVLMModel(
	_ context.Context,
	modelID string,
) (vlm.VLM, error) {
	s.requestedVLMModelID = modelID
	return nil, nil
}

func TestResolveVLMUsesKnowledgeProcessingConfig(t *testing.T) {
	kb := &types.KnowledgeBase{
		ID:             "kb-1",
		SummaryModelID: "qwen3.8",
		VLMConfig: types.VLMConfig{
			Enabled: true,
			ModelID: "qwen3-vl-plus",
		},
	}
	knowledge := &types.Knowledge{ID: "knowledge-1"}
	models := &captureMultimodalModelService{}
	svc := &ImageMultimodalService{
		kbService:     &orphanKBService{kb: kb},
		knowledgeRepo: &orphanKnowledgeRepo{knowledge: knowledge},
		modelService:  models,
	}

	_, cfg, err := svc.resolveVLM(context.Background(), kb.ID, knowledge.ID)

	require.NoError(t, err)
	require.Equal(t, "qwen3-vl-plus", cfg.ModelID)
	require.Equal(t, "qwen3-vl-plus", models.requestedVLMModelID)
}

func TestResolveVLMUsesPersistedUploadOverride(t *testing.T) {
	kb := &types.KnowledgeBase{
		ID: "kb-1",
		VLMConfig: types.VLMConfig{
			Enabled: true,
			ModelID: "qwen3-vl-plus",
		},
	}
	knowledge := &types.Knowledge{ID: "knowledge-1"}
	require.NoError(t, knowledge.SetProcessOverrides(&types.KnowledgeProcessOverrides{
		VLMConfig: &types.VLMConfig{
			Enabled: true,
			ModelID: "upload-vlm",
		},
	}))
	models := &captureMultimodalModelService{}
	svc := &ImageMultimodalService{
		kbService:     &orphanKBService{kb: kb},
		knowledgeRepo: &orphanKnowledgeRepo{knowledge: knowledge},
		modelService:  models,
	}

	_, cfg, err := svc.resolveVLM(context.Background(), kb.ID, knowledge.ID)

	require.NoError(t, err)
	require.Equal(t, "upload-vlm", cfg.ModelID)
	require.Equal(t, "upload-vlm", models.requestedVLMModelID)
}
