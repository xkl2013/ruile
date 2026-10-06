package types

import "time"

// PublicKnowledgeBasePublicationStatus is the lifecycle state of a platform
// knowledge-base publication.
type PublicKnowledgeBasePublicationStatus string

const (
	PublicKnowledgeBasePublicationDraft     PublicKnowledgeBasePublicationStatus = "draft"
	PublicKnowledgeBasePublicationPublished PublicKnowledgeBasePublicationStatus = "published"
	PublicKnowledgeBasePublicationOffline   PublicKnowledgeBasePublicationStatus = "offline"
)

// IsValid reports whether the publication status is supported.
func (s PublicKnowledgeBasePublicationStatus) IsValid() bool {
	switch s {
	case PublicKnowledgeBasePublicationDraft,
		PublicKnowledgeBasePublicationPublished,
		PublicKnowledgeBasePublicationOffline:
		return true
	default:
		return false
	}
}

// PublicKnowledgeBasePublication is the platform-facing metadata for a
// knowledge base that can be discovered and subscribed to by users.
type PublicKnowledgeBasePublication struct {
	ID              string                               `json:"id" gorm:"type:varchar(36);primaryKey"`
	KnowledgeBaseID string                               `json:"knowledge_base_id" gorm:"type:varchar(36);not null;uniqueIndex"`
	Title           string                               `json:"title" gorm:"type:varchar(120);not null"`
	Description     string                               `json:"description" gorm:"type:text;not null;default:''"`
	Category        string                               `json:"category" gorm:"type:varchar(64);not null;default:'';index"`
	Status          PublicKnowledgeBasePublicationStatus `json:"status" gorm:"type:varchar(32);not null;default:'draft';index"`
	Featured        bool                                 `json:"featured" gorm:"not null;default:false"`
	Recommendable   bool                                 `json:"recommendable" gorm:"not null;default:true;index"`
	SortOrder       int                                  `json:"sort_order" gorm:"not null;default:0"`
	PublishedAt     *time.Time                           `json:"published_at,omitempty"`
	OfflineAt       *time.Time                           `json:"offline_at,omitempty"`
	CreatedBy       string                               `json:"created_by" gorm:"type:varchar(36);not null;index"`
	UpdatedBy       string                               `json:"updated_by" gorm:"type:varchar(36);not null;index"`
	CreatedAt       time.Time                            `json:"created_at"`
	UpdatedAt       time.Time                            `json:"updated_at"`

	KnowledgeBase   *KnowledgeBase `json:"knowledge_base,omitempty" gorm:"-"`
	SubscriberCount int64          `json:"subscriber_count" gorm:"-"`
	IsSubscribed    bool           `json:"is_subscribed" gorm:"-"`
}

func (PublicKnowledgeBasePublication) TableName() string {
	return "knowledge_base_publications"
}

// PublicKnowledgeBaseSubscription is the access-granting subscription
// relation for a published platform knowledge base. It is intentionally
// separate from knowledge_base_subscriptions, which remains a user shortcut.
type PublicKnowledgeBaseSubscription struct {
	ID            string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID        string     `json:"user_id" gorm:"type:varchar(36);not null;index"`
	PublicationID string     `json:"publication_id" gorm:"type:varchar(36);not null;index"`
	Status        string     `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
}

func (PublicKnowledgeBaseSubscription) TableName() string {
	return "public_knowledge_base_subscriptions"
}

const (
	PublicKnowledgeBaseSubscriptionActive    = "active"
	PublicKnowledgeBaseSubscriptionCancelled = "cancelled"
)

// PublicKnowledgeBaseSubscriptionResult is returned by public subscribe APIs.
type PublicKnowledgeBaseSubscriptionResult struct {
	PublicationID   string `json:"publication_id"`
	KnowledgeBaseID string `json:"knowledge_base_id"`
	Subscribed      bool   `json:"subscribed"`
	SubscriptionID  string `json:"subscription_id,omitempty"`
}
