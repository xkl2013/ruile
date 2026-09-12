package types

import "time"

// KnowledgeBaseSubscriptionStatus is the lifecycle state of a user's
// shortcut to a knowledge base. Subscription state never grants access.
type KnowledgeBaseSubscriptionStatus string

const (
	KnowledgeBaseSubscriptionActive    KnowledgeBaseSubscriptionStatus = "active"
	KnowledgeBaseSubscriptionCancelled KnowledgeBaseSubscriptionStatus = "cancelled"
)

// IsValid reports whether the subscription status is supported.
func (s KnowledgeBaseSubscriptionStatus) IsValid() bool {
	switch s {
	case KnowledgeBaseSubscriptionActive, KnowledgeBaseSubscriptionCancelled:
		return true
	default:
		return false
	}
}

// KnowledgeBaseSubscription stores a user's shortcut relationship to a
// knowledge base without copying any knowledge-base content.
type KnowledgeBaseSubscription struct {
	ID              string                          `json:"id" gorm:"type:varchar(36);primaryKey"`
	UserID          string                          `json:"user_id" gorm:"type:varchar(36);not null;index"`
	KnowledgeBaseID string                          `json:"knowledge_base_id" gorm:"type:varchar(36);not null;index"`
	Status          KnowledgeBaseSubscriptionStatus `json:"status" gorm:"type:varchar(32);not null;default:'active'"`
	CreatedAt       time.Time                       `json:"created_at"`
	UpdatedAt       time.Time                       `json:"updated_at"`
}

// TableName returns the subscription table name.
func (KnowledgeBaseSubscription) TableName() string {
	return "knowledge_base_subscriptions"
}
