package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type PublicCreatorService interface {
	ListCreators(ctx context.Context, query types.PublicCreatorQuery) ([]*types.PublicCreatorSummary, int64, error)
	GetCreator(ctx context.Context, creatorID string) (*types.PublicCreatorDetail, error)
	PublishKnowledgeBase(ctx context.Context, creatorID, knowledgeBaseID string) (*types.PublicCreatorPublishResult, error)
	PublishCreator(ctx context.Context, creatorID string) (*types.PublicCreatorPublishResult, error)
	OfflineCreator(ctx context.Context, creatorID string) (*types.PublicCreatorPublishResult, error)
}
