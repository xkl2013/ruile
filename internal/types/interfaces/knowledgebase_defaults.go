package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// KnowledgeBaseDefaultsService manages the advanced configuration inherited by
// knowledge bases created after the workspace setting is saved. Existing
// knowledge bases require an explicit migration before their configuration is
// rewritten.
type KnowledgeBaseDefaultsService interface {
	Get(ctx context.Context) (*types.KnowledgeBaseDefaultsConfig, error)
	Update(ctx context.Context, config *types.KnowledgeBaseDefaultsConfig) (*types.KnowledgeBaseDefaultsConfig, int, error)
}
