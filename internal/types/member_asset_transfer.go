package types

// MemberAssetType identifies an enterprise asset whose responsible member can
// be reassigned without changing the owning tenant.
type MemberAssetType string

const (
	MemberAssetTypeKnowledgeBase MemberAssetType = "knowledge_base"
	MemberAssetTypeAgent         MemberAssetType = "agent"
)

// MemberAssetTransferTargetType distinguishes assigning assets to another
// active member from returning them to enterprise-level ownership.
type MemberAssetTransferTargetType string

const (
	MemberAssetTransferTargetMember     MemberAssetTransferTargetType = "member"
	MemberAssetTransferTargetEnterprise MemberAssetTransferTargetType = "enterprise"
)

// MemberAssetTransferScope controls whether all assets of the requested types
// or only explicitly selected asset IDs are transferred.
type MemberAssetTransferScope string

const (
	MemberAssetTransferScopeAll      MemberAssetTransferScope = "all"
	MemberAssetTransferScopeSelected MemberAssetTransferScope = "selected"
)

// MemberTransferableAsset is the lightweight projection shown in the member
// offboarding dialog.
type MemberTransferableAsset struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Type MemberAssetType `json:"type"`
}

// MemberTransferableAssets groups every supported enterprise asset currently
// assigned to a member.
type MemberTransferableAssets struct {
	TenantID       uint64                    `json:"tenant_id"`
	SourceUserID   string                    `json:"source_user_id"`
	KnowledgeBases []MemberTransferableAsset `json:"knowledge_bases"`
	Agents         []MemberTransferableAsset `json:"agents"`
	Total          int                       `json:"total"`
}

// MemberAssetTransferCommand is the validated service/repository command for
// an enterprise-internal ownership handoff.
type MemberAssetTransferCommand struct {
	TenantID         uint64
	SourceUserID     string
	TargetType       MemberAssetTransferTargetType
	TargetUserID     string
	Scope            MemberAssetTransferScope
	AssetTypes       []MemberAssetType
	KnowledgeBaseIDs []string
	AgentIDs         []string
	Reason           string
}

// MemberAssetTransferResult reports the committed transfer counts.
type MemberAssetTransferResult struct {
	TenantID                  uint64                        `json:"tenant_id"`
	SourceUserID              string                        `json:"source_user_id"`
	TargetType                MemberAssetTransferTargetType `json:"target_type"`
	TargetUserID              string                        `json:"target_user_id,omitempty"`
	KnowledgeBasesTransferred int64                         `json:"knowledge_bases_transferred"`
	AgentsTransferred         int64                         `json:"agents_transferred"`
	TotalTransferred          int64                         `json:"total_transferred"`
}
