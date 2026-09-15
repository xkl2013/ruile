package types

import "time"

// MyKnowledgeBaseList is the account-centred projection for the knowledge-base
// home page. A knowledge base may appear in more than one group; subscription
// rows are shortcuts only and never grant access.
type MyKnowledgeBaseList struct {
	Created    []*MyKnowledgeBaseListItem `json:"created"`
	Shared     []*MyKnowledgeBaseListItem `json:"shared"`
	Subscribed []*MyKnowledgeBaseListItem `json:"subscribed"`
}

// MyKnowledgeBaseListItem wraps a knowledge base with the caller's effective
// access projection for one list category.
type MyKnowledgeBaseListItem struct {
	KnowledgeBase     *KnowledgeBase            `json:"knowledge_base"`
	EffectiveTenantID uint64                    `json:"effective_tenant_id"`
	Permission        OrgMemberRole             `json:"permission"`
	AccessSource      KnowledgeBaseAccessSource `json:"access_source"`
	OwnerType         *SpaceType                `json:"owner_type,omitempty"`
	OrganizationID    string                    `json:"organization_id,omitempty"`
	OrgName           string                    `json:"org_name,omitempty"`
	SharingScope      *SharingScope             `json:"sharing_scope,omitempty"`
	ShareID           string                    `json:"share_id,omitempty"`
	SharedAt          *time.Time                `json:"shared_at,omitempty"`
	IsSubscribed      bool                      `json:"is_subscribed"`
	SubscriptionID    string                    `json:"subscription_id,omitempty"`
	SubscribedAt      *time.Time                `json:"subscribed_at,omitempty"`
}
