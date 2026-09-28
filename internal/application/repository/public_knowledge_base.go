package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrPublicKnowledgeBasePublicationNotFound = errors.New("public knowledge base publication not found")

type publicKnowledgeBaseRepository struct {
	db *gorm.DB
}

func NewPublicKnowledgeBaseRepository(db *gorm.DB) interfaces.PublicKnowledgeBaseRepository {
	return &publicKnowledgeBaseRepository{db: db}
}

func (r *publicKnowledgeBaseRepository) CreatePublication(
	ctx context.Context,
	publication *types.PublicKnowledgeBasePublication,
) error {
	return r.db.WithContext(ctx).Create(publication).Error
}

func (r *publicKnowledgeBaseRepository) GetPublicationByID(
	ctx context.Context,
	id string,
) (*types.PublicKnowledgeBasePublication, error) {
	var row types.PublicKnowledgeBasePublication
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicKnowledgeBasePublicationNotFound
		}
		return nil, err
	}
	return r.loadKnowledgeBase(ctx, &row)
}

func (r *publicKnowledgeBaseRepository) GetPublicationByKnowledgeBaseID(
	ctx context.Context,
	kbID string,
) (*types.PublicKnowledgeBasePublication, error) {
	var row types.PublicKnowledgeBasePublication
	if err := r.db.WithContext(ctx).Where("knowledge_base_id = ?", kbID).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicKnowledgeBasePublicationNotFound
		}
		return nil, err
	}
	return r.loadKnowledgeBase(ctx, &row)
}

func (r *publicKnowledgeBaseRepository) ListPublications(
	ctx context.Context,
	status *types.PublicKnowledgeBasePublicationStatus,
	keyword string,
) ([]*types.PublicKnowledgeBasePublication, error) {
	query := r.db.WithContext(ctx).Model(&types.PublicKnowledgeBasePublication{})
	if status != nil {
		query = query.Where("status = ?", *status)
	}
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"LOWER(title) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?) OR LOWER(category) LIKE LOWER(?)",
			like,
			like,
			like,
		)
	}

	var rows []*types.PublicKnowledgeBasePublication
	if err := query.
		Order("featured DESC").
		Order("sort_order ASC").
		Order("updated_at DESC").
		Order("id ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		if _, err := r.loadKnowledgeBase(ctx, row); err != nil {
			return nil, err
		}
	}
	return rows, nil
}

func (r *publicKnowledgeBaseRepository) UpdatePublication(
	ctx context.Context,
	publication *types.PublicKnowledgeBasePublication,
) error {
	return r.db.WithContext(ctx).
		Model(&types.PublicKnowledgeBasePublication{}).
		Where("id = ?", publication.ID).
		Updates(map[string]interface{}{
			"title":       publication.Title,
			"description": publication.Description,
			"category":    publication.Category,
			"featured":    publication.Featured,
			"sort_order":  publication.SortOrder,
			"updated_by":  publication.UpdatedBy,
			"updated_at":  publication.UpdatedAt,
		}).Error
}

func (r *publicKnowledgeBaseRepository) SetPublicationStatus(
	ctx context.Context,
	id string,
	status types.PublicKnowledgeBasePublicationStatus,
	actorID string,
) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":     status,
		"updated_by": actorID,
		"updated_at": now,
	}
	switch status {
	case types.PublicKnowledgeBasePublicationPublished:
		updates["published_at"] = now
		updates["offline_at"] = nil
	case types.PublicKnowledgeBasePublicationOffline:
		updates["offline_at"] = now
	}
	result := r.db.WithContext(ctx).
		Model(&types.PublicKnowledgeBasePublication{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrPublicKnowledgeBasePublicationNotFound
	}
	return nil
}

func (r *publicKnowledgeBaseRepository) UpsertSubscription(
	ctx context.Context,
	userID, publicationID string,
) (*types.PublicKnowledgeBaseSubscription, error) {
	var row types.PublicKnowledgeBaseSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND publication_id = ?", userID, publicationID).
		First(&row).Error
	now := time.Now().UTC()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		row = types.PublicKnowledgeBaseSubscription{
			ID:            uuid.NewString(),
			UserID:        userID,
			PublicationID: publicationID,
			Status:        types.PublicKnowledgeBaseSubscriptionActive,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
		return &row, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Status != types.PublicKnowledgeBaseSubscriptionActive {
		row.Status = types.PublicKnowledgeBaseSubscriptionActive
		row.CancelledAt = nil
		row.UpdatedAt = now
		if err := r.db.WithContext(ctx).
			Model(&types.PublicKnowledgeBaseSubscription{}).
			Where("id = ?", row.ID).
			Updates(map[string]interface{}{
				"status":       row.Status,
				"cancelled_at": nil,
				"updated_at":   row.UpdatedAt,
			}).Error; err != nil {
			return nil, err
		}
	}
	return &row, nil
}

func (r *publicKnowledgeBaseRepository) CancelSubscription(
	ctx context.Context,
	userID, publicationID string,
) (*types.PublicKnowledgeBaseSubscription, error) {
	var row types.PublicKnowledgeBaseSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND publication_id = ?", userID, publicationID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &types.PublicKnowledgeBaseSubscription{
			UserID:        userID,
			PublicationID: publicationID,
			Status:        types.PublicKnowledgeBaseSubscriptionCancelled,
		}, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Status == types.PublicKnowledgeBaseSubscriptionCancelled {
		return &row, nil
	}
	now := time.Now().UTC()
	row.Status = types.PublicKnowledgeBaseSubscriptionCancelled
	row.CancelledAt = &now
	row.UpdatedAt = now
	if err := r.db.WithContext(ctx).
		Model(&types.PublicKnowledgeBaseSubscription{}).
		Where("id = ?", row.ID).
		Updates(map[string]interface{}{
			"status":       row.Status,
			"cancelled_at": row.CancelledAt,
			"updated_at":   row.UpdatedAt,
		}).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *publicKnowledgeBaseRepository) ListActiveSubscriptionsByUserID(
	ctx context.Context,
	userID string,
) ([]*types.PublicKnowledgeBaseSubscription, error) {
	var rows []*types.PublicKnowledgeBaseSubscription
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, types.PublicKnowledgeBaseSubscriptionActive).
		Order("created_at DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *publicKnowledgeBaseRepository) IsActiveSubscription(
	ctx context.Context,
	userID, publicationID string,
) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&types.PublicKnowledgeBaseSubscription{}).
		Where("user_id = ? AND publication_id = ? AND status = ?", userID, publicationID, types.PublicKnowledgeBaseSubscriptionActive).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *publicKnowledgeBaseRepository) loadKnowledgeBase(
	ctx context.Context,
	row *types.PublicKnowledgeBasePublication,
) (*types.PublicKnowledgeBasePublication, error) {
	if row == nil {
		return row, nil
	}
	var kb types.KnowledgeBase
	if err := r.db.WithContext(ctx).Where("id = ?", row.KnowledgeBaseID).First(&kb).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPublicKnowledgeBasePublicationNotFound
		}
		return nil, err
	}
	row.KnowledgeBase = &kb
	if err := r.db.WithContext(ctx).
		Model(&types.PublicKnowledgeBaseSubscription{}).
		Where("publication_id = ? AND status = ?", row.ID, types.PublicKnowledgeBaseSubscriptionActive).
		Count(&row.SubscriberCount).Error; err != nil {
		return nil, err
	}
	return row, nil
}
