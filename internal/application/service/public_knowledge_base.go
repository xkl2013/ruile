package service

import (
	"context"
	"errors"
	"strings"
	"time"

	apprepo "github.com/Tencent/WeKnora/internal/application/repository"
	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
)

type publicKnowledgeBaseService struct {
	repo  interfaces.PublicKnowledgeBaseRepository
	kbSvc interfaces.KnowledgeBaseService
}

func NewPublicKnowledgeBaseService(
	repo interfaces.PublicKnowledgeBaseRepository,
	kbSvc interfaces.KnowledgeBaseService,
) interfaces.PublicKnowledgeBaseService {
	return &publicKnowledgeBaseService{repo: repo, kbSvc: kbSvc}
}

func (s *publicKnowledgeBaseService) ListAdminPublications(
	ctx context.Context,
	status *types.PublicKnowledgeBasePublicationStatus,
	keyword string,
) ([]*types.PublicKnowledgeBasePublication, error) {
	return s.repo.ListPublications(ctx, status, keyword)
}

func (s *publicKnowledgeBaseService) CreatePublication(
	ctx context.Context,
	kbID, title, description, category string,
) (*types.PublicKnowledgeBasePublication, error) {
	kbID = strings.TrimSpace(kbID)
	if kbID == "" {
		return nil, errors.New("knowledge_base_id is required")
	}
	kb, err := s.kbSvc.GetKnowledgeBaseByIDOnly(ctx, kbID)
	if err != nil {
		return nil, err
	}
	if kb == nil || kb.IsTemporary {
		return nil, apprepo.ErrKnowledgeBaseNotFound
	}
	if existing, err := s.repo.GetPublicationByKnowledgeBaseID(ctx, kbID); err == nil && existing != nil {
		return existing, errors.New("knowledge base already has a publication")
	} else if err != nil && !errors.Is(err, apprepo.ErrPublicKnowledgeBasePublicationNotFound) {
		return nil, err
	}

	actorID, _ := types.UserIDFromContext(ctx)
	now := time.Now().UTC()
	publication := &types.PublicKnowledgeBasePublication{
		ID:              uuid.NewString(),
		KnowledgeBaseID: kbID,
		Title:           firstNonEmptyPublicKB(strings.TrimSpace(title), kb.Name),
		Description:     strings.TrimSpace(description),
		Category:        strings.TrimSpace(category),
		Status:          types.PublicKnowledgeBasePublicationDraft,
		CreatedBy:       actorID,
		UpdatedBy:       actorID,
		CreatedAt:       now,
		UpdatedAt:       now,
		KnowledgeBase:   kb,
	}
	if publication.Title == "" {
		return nil, errors.New("title is required")
	}
	if err := s.repo.CreatePublication(ctx, publication); err != nil {
		return nil, err
	}
	return publication, nil
}

func (s *publicKnowledgeBaseService) GetAdminPublication(
	ctx context.Context,
	id string,
) (*types.PublicKnowledgeBasePublication, error) {
	return s.repo.GetPublicationByID(ctx, strings.TrimSpace(id))
}

func (s *publicKnowledgeBaseService) UpdatePublication(
	ctx context.Context,
	id, title, description, category string,
	featured bool,
	sortOrder int,
) (*types.PublicKnowledgeBasePublication, error) {
	publication, err := s.repo.GetPublicationByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("title is required")
	}
	actorID, _ := types.UserIDFromContext(ctx)
	publication.Title = title
	publication.Description = strings.TrimSpace(description)
	publication.Category = strings.TrimSpace(category)
	publication.Featured = featured
	publication.SortOrder = sortOrder
	publication.UpdatedBy = actorID
	publication.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdatePublication(ctx, publication); err != nil {
		return nil, err
	}
	return publication, nil
}

func (s *publicKnowledgeBaseService) PublishPublication(
	ctx context.Context,
	id string,
) (*types.PublicKnowledgeBasePublication, error) {
	publication, err := s.repo.GetPublicationByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if publication.KnowledgeBase == nil {
		return nil, apprepo.ErrPublicKnowledgeBasePublicationNotFound
	}
	if err := s.kbSvc.FillKnowledgeBaseCounts(ctx, publication.KnowledgeBase); err != nil {
		return nil, err
	}
	contentCount := publication.KnowledgeBase.KnowledgeCount
	if publication.KnowledgeBase.Type == types.KnowledgeBaseTypeFAQ {
		contentCount = publication.KnowledgeBase.ChunkCount
	}
	if publication.Title == "" || contentCount == 0 {
		return nil, errors.New("published knowledge base requires a title and at least one content item")
	}
	if publication.KnowledgeBase.IsProcessing || publication.KnowledgeBase.ProcessingCount > 0 {
		return nil, errors.New("knowledge base is still processing")
	}
	actorID, _ := types.UserIDFromContext(ctx)
	if err := s.repo.SetPublicationStatus(
		ctx,
		publication.ID,
		types.PublicKnowledgeBasePublicationPublished,
		actorID,
	); err != nil {
		return nil, err
	}
	return s.repo.GetPublicationByID(ctx, publication.ID)
}

func (s *publicKnowledgeBaseService) OfflinePublication(
	ctx context.Context,
	id string,
) (*types.PublicKnowledgeBasePublication, error) {
	id = strings.TrimSpace(id)
	if _, err := s.repo.GetPublicationByID(ctx, id); err != nil {
		return nil, err
	}
	actorID, _ := types.UserIDFromContext(ctx)
	if err := s.repo.SetPublicationStatus(
		ctx,
		id,
		types.PublicKnowledgeBasePublicationOffline,
		actorID,
	); err != nil {
		return nil, err
	}
	return s.repo.GetPublicationByID(ctx, id)
}

func (s *publicKnowledgeBaseService) ListPublications(
	ctx context.Context,
	userID, keyword, category string,
) ([]*types.PublicKnowledgeBasePublication, error) {
	status := types.PublicKnowledgeBasePublicationPublished
	rows, err := s.repo.ListPublications(ctx, &status, keyword)
	if err != nil {
		return nil, err
	}
	category = strings.TrimSpace(category)
	if category == "" && userID == "" {
		return rows, nil
	}

	activeByPublication := map[string]bool{}
	if userID != "" {
		subscriptions, err := s.repo.ListActiveSubscriptionsByUserID(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, subscription := range subscriptions {
			if subscription != nil {
				activeByPublication[subscription.PublicationID] = true
			}
		}
	}
	out := make([]*types.PublicKnowledgeBasePublication, 0, len(rows))
	for _, row := range rows {
		if row == nil || (category != "" && row.Category != category) {
			continue
		}
		row.IsSubscribed = activeByPublication[row.ID]
		out = append(out, row)
	}
	return out, nil
}

func (s *publicKnowledgeBaseService) GetPublicPublication(
	ctx context.Context,
	id, userID string,
) (*types.PublicKnowledgeBasePublication, error) {
	row, err := s.repo.GetPublicationByID(ctx, strings.TrimSpace(id))
	if err != nil {
		return nil, err
	}
	if row.Status != types.PublicKnowledgeBasePublicationPublished {
		return nil, apprepo.ErrPublicKnowledgeBasePublicationNotFound
	}
	if strings.TrimSpace(userID) != "" {
		row.IsSubscribed, err = s.repo.IsActiveSubscription(ctx, userID, row.ID)
		if err != nil {
			return nil, err
		}
	}
	return row, nil
}

func (s *publicKnowledgeBaseService) Subscribe(
	ctx context.Context,
	publicationID string,
) (*types.PublicKnowledgeBaseSubscriptionResult, error) {
	userID, ok := concreteSubscriptionUserID(ctx)
	if !ok {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	publication, err := s.GetPublicPublication(ctx, publicationID, userID)
	if err != nil {
		return nil, err
	}
	subscription, err := s.repo.UpsertSubscription(ctx, userID, publication.ID)
	if err != nil {
		return nil, err
	}
	return &types.PublicKnowledgeBaseSubscriptionResult{
		PublicationID:   publication.ID,
		KnowledgeBaseID: publication.KnowledgeBaseID,
		Subscribed:      true,
		SubscriptionID:  subscription.ID,
	}, nil
}

func (s *publicKnowledgeBaseService) Unsubscribe(
	ctx context.Context,
	publicationID string,
) (*types.PublicKnowledgeBaseSubscriptionResult, error) {
	userID, ok := concreteSubscriptionUserID(ctx)
	if !ok {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	publication, err := s.repo.GetPublicationByID(ctx, strings.TrimSpace(publicationID))
	if err != nil {
		return nil, err
	}
	subscription, err := s.repo.CancelSubscription(ctx, userID, publication.ID)
	if err != nil {
		return nil, err
	}
	result := &types.PublicKnowledgeBaseSubscriptionResult{
		PublicationID:   publication.ID,
		KnowledgeBaseID: publication.KnowledgeBaseID,
		Subscribed:      false,
	}
	if subscription != nil {
		result.SubscriptionID = subscription.ID
	}
	return result, nil
}

func (s *publicKnowledgeBaseService) ListMySubscriptions(
	ctx context.Context,
) ([]*types.PublicKnowledgeBasePublication, error) {
	userID, ok := concreteSubscriptionUserID(ctx)
	if !ok {
		return nil, types.ErrKnowledgeBaseAccessUnauthorized
	}
	subscriptions, err := s.repo.ListActiveSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]*types.PublicKnowledgeBasePublication, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription == nil {
			continue
		}
		row, err := s.GetPublicPublication(ctx, subscription.PublicationID, userID)
		if err != nil {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func firstNonEmptyPublicKB(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
