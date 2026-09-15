package types

import "errors"

// KnowledgeBaseAccessSource explains why the current caller can see or operate
// on a knowledge base. V1 populates the existing ownership/RBAC/share paths;
// subscription remains metadata-only until the V2 list and subscription APIs
// are implemented.
type KnowledgeBaseAccessSource string

const (
	KnowledgeBaseAccessSourceCreated      KnowledgeBaseAccessSource = "created"
	KnowledgeBaseAccessSourceTenantAdmin  KnowledgeBaseAccessSource = "tenant_admin"
	KnowledgeBaseAccessSourceSystemAdmin  KnowledgeBaseAccessSource = "system_admin"
	KnowledgeBaseAccessSourceAPIKey       KnowledgeBaseAccessSource = "api_key"
	KnowledgeBaseAccessSourceSharedSpace  KnowledgeBaseAccessSource = "shared_space"
	KnowledgeBaseAccessSourceSharedAgent  KnowledgeBaseAccessSource = "shared_agent"
	KnowledgeBaseAccessSourceSubscription KnowledgeBaseAccessSource = "subscription"
)

// KnowledgeBaseAccessOptions controls an access-resolution request.
type KnowledgeBaseAccessOptions struct {
	// RequiredPermission is the minimum org-level permission the caller needs.
	// Empty means viewer.
	RequiredPermission OrgMemberRole
}

// KnowledgeBaseAccess is the service-level result of resolving access to a KB.
type KnowledgeBaseAccess struct {
	KnowledgeBase     *KnowledgeBase            `json:"knowledge_base,omitempty"`
	EffectiveTenantID uint64                    `json:"effective_tenant_id"`
	Permission        OrgMemberRole             `json:"permission"`
	AccessSource      KnowledgeBaseAccessSource `json:"access_source"`
	OwnerType         *SpaceType                `json:"owner_type,omitempty"`
	OrganizationID    string                    `json:"organization_id,omitempty"`
	SharingScope      *SharingScope             `json:"sharing_scope,omitempty"`
	IsSubscribed      bool                      `json:"is_subscribed"`
}

var (
	ErrKnowledgeBaseAccessUnauthorized = errors.New("knowledge base access: unauthorized")
	ErrKnowledgeBaseAccessNotFound     = errors.New("knowledge base access: not found")
	ErrKnowledgeBaseAccessForbidden    = errors.New("knowledge base access: forbidden")
)
