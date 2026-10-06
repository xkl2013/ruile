package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

type PublicKnowledgeBaseRepository interface {
	CreatePublication(ctx context.Context, publication *types.PublicKnowledgeBasePublication) error
	GetPublicationByID(ctx context.Context, id string) (*types.PublicKnowledgeBasePublication, error)
	GetPublicationByKnowledgeBaseID(ctx context.Context, kbID string) (*types.PublicKnowledgeBasePublication, error)
	ListPublications(ctx context.Context, status *types.PublicKnowledgeBasePublicationStatus, keyword string) ([]*types.PublicKnowledgeBasePublication, error)
	UpdatePublication(ctx context.Context, publication *types.PublicKnowledgeBasePublication) error
	SetPublicationStatus(ctx context.Context, id string, status types.PublicKnowledgeBasePublicationStatus, actorID string) error
	UpsertSubscription(ctx context.Context, userID, publicationID string) (*types.PublicKnowledgeBaseSubscription, error)
	CancelSubscription(ctx context.Context, userID, publicationID string) (*types.PublicKnowledgeBaseSubscription, error)
	ListActiveSubscriptionsByUserID(ctx context.Context, userID string) ([]*types.PublicKnowledgeBaseSubscription, error)
	IsActiveSubscription(ctx context.Context, userID, publicationID string) (bool, error)
}

type PublicKnowledgeBaseService interface {
	ListAdminPublications(ctx context.Context, status *types.PublicKnowledgeBasePublicationStatus, keyword string) ([]*types.PublicKnowledgeBasePublication, error)
	CreatePublication(ctx context.Context, kbID, title, description, category string) (*types.PublicKnowledgeBasePublication, error)
	GetAdminPublication(ctx context.Context, id string) (*types.PublicKnowledgeBasePublication, error)
	UpdatePublication(ctx context.Context, id, title, description, category string, featured, recommendable bool, sortOrder int) (*types.PublicKnowledgeBasePublication, error)
	PublishPublication(ctx context.Context, id string) (*types.PublicKnowledgeBasePublication, error)
	OfflinePublication(ctx context.Context, id string) (*types.PublicKnowledgeBasePublication, error)
	ListPublications(ctx context.Context, userID, keyword, category string) ([]*types.PublicKnowledgeBasePublication, error)
	GetPublicPublication(ctx context.Context, id, userID string) (*types.PublicKnowledgeBasePublication, error)
	Subscribe(ctx context.Context, publicationID string) (*types.PublicKnowledgeBaseSubscriptionResult, error)
	Unsubscribe(ctx context.Context, publicationID string) (*types.PublicKnowledgeBaseSubscriptionResult, error)
	ListMySubscriptions(ctx context.Context) ([]*types.PublicKnowledgeBasePublication, error)
}
