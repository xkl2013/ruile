package types

import (
	"time"

	"gorm.io/gorm"
)

// OrgMemberRole represents the role of an organization member
type OrgMemberRole string

const (
	// OrgRoleAdmin has full control over the organization and shared knowledge bases
	OrgRoleAdmin OrgMemberRole = "admin"
	// OrgRoleEditor can edit shared knowledge base content but cannot manage settings
	OrgRoleEditor OrgMemberRole = "editor"
	// OrgRoleViewer can only view and search shared knowledge bases
	OrgRoleViewer OrgMemberRole = "viewer"
)

// IsValid checks if the role is valid
func (r OrgMemberRole) IsValid() bool {
	switch r {
	case OrgRoleAdmin, OrgRoleEditor, OrgRoleViewer:
		return true
	default:
		return false
	}
}

// HasPermission checks if this role has at least the required permission level
func (r OrgMemberRole) HasPermission(required OrgMemberRole) bool {
	roleLevel := map[OrgMemberRole]int{
		OrgRoleAdmin:  3,
		OrgRoleEditor: 2,
		OrgRoleViewer: 1,
	}
	return roleLevel[r] >= roleLevel[required]
}

// MinOrgRole returns whichever of a / b is the lower role on the
// admin > editor > viewer ladder. Used to apply caps when combining
// (a) the share's grant, (b) the tenant's role inside the org, and
// (c) the caller's tenant-RBAC ceiling. A zero/empty role is treated
// as "less than viewer" so it short-circuits to whatever the other
// argument is.
func MinOrgRole(a, b OrgMemberRole) OrgMemberRole {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a.HasPermission(b) {
		return b
	}
	return a
}

// Organization represents a collaboration organization for cross-tenant sharing
type Organization struct {
	// Unique identifier of the organization
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`
	// Name of the organization
	Name string `json:"name" gorm:"type:varchar(255);not null"`
	// Description of the organization
	Description string `json:"description" gorm:"type:text"`
	// Avatar URL for display in list and settings
	Avatar string `json:"avatar" gorm:"type:varchar(512)"`
	// User ID of the organization owner
	OwnerID string `json:"owner_id" gorm:"type:varchar(36);not null;index"`
	// OwnerTenantID is the tenant the owner belonged to when the
	// organization was created. It is retained for workspace context and
	// legacy data; owner protections use OwnerID.
	OwnerTenantID uint64 `json:"owner_tenant_id" gorm:"not null;index"`
	// Compatibility-only invite code column. Self-service joining is disabled;
	// organization membership is managed through admin-added tenant members.
	InviteCode string `json:"invite_code" gorm:"type:varchar(32);uniqueIndex"`
	// Compatibility-only expiry for the invite code column.
	InviteCodeExpiresAt *time.Time `json:"invite_code_expires_at" gorm:"type:timestamp with time zone"`
	// Compatibility-only invite validity. Not exposed in organization responses.
	InviteCodeValidityDays int `json:"invite_code_validity_days" gorm:"default:7"`
	// Deprecated join-setting column retained for existing schemas.
	RequireApproval bool `json:"require_approval" gorm:"default:false"`
	// Deprecated discovery column retained for existing schemas.
	Searchable bool `json:"searchable" gorm:"default:false"`
	// Max members allowed; 0 means no limit
	MemberLimit int `json:"member_limit" gorm:"default:50"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last updated time
	UpdatedAt time.Time `json:"updated_at"`
	// Deletion time (soft delete)
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Associations (not stored in database)
	Owner   *User                      `json:"owner,omitempty" gorm:"foreignKey:OwnerID"`
	Members []OrganizationTenantMember `json:"members,omitempty" gorm:"foreignKey:OrganizationID"`
	Shares  []KnowledgeBaseShare       `json:"shares,omitempty" gorm:"foreignKey:OrganizationID"`
}

// TableName returns the table name for GORM
func (Organization) TableName() string {
	return "organizations"
}

// OrganizationTenantMember represents one account participating in an
// organization. The historical table name is kept, but representative_user_id
// is the user who receives the shared-space grant. tenant_id is retained as
// the member-management source context. Permission checks use
// (org, representative_user_id, role).
type OrganizationTenantMember struct {
	ID                   string        `json:"id" gorm:"type:varchar(36);primaryKey"`
	OrganizationID       string        `json:"organization_id" gorm:"type:varchar(36);not null;index"`
	TenantID             uint64        `json:"tenant_id" gorm:"not null;index"`
	Role                 OrgMemberRole `json:"role" gorm:"type:varchar(32);not null;default:'viewer'"`
	RepresentativeUserID string        `json:"representative_user_id" gorm:"type:varchar(36);default:''"`
	JoinedAt             *time.Time    `json:"joined_at"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`

	Organization       *Organization `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
	RepresentativeUser *User         `json:"representative_user,omitempty" gorm:"foreignKey:RepresentativeUserID"`
}

// TableName returns the table name for GORM
func (OrganizationTenantMember) TableName() string {
	return "organization_tenant_members"
}

// KnowledgeBaseShare represents a sharing record of a knowledge base to an organization
type KnowledgeBaseShare struct {
	// Unique identifier
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`
	// Knowledge base ID being shared
	KnowledgeBaseID string `json:"knowledge_base_id" gorm:"type:varchar(36);not null;index"`
	// Organization ID receiving the share
	OrganizationID string `json:"organization_id" gorm:"type:varchar(36);not null;index"`
	// User ID who shared the knowledge base
	SharedByUserID string `json:"shared_by_user_id" gorm:"type:varchar(36);not null"`
	// Original tenant ID of the knowledge base (for cross-tenant embedding model access)
	SourceTenantID uint64 `json:"source_tenant_id" gorm:"not null;index"`
	// Permission level (admin/editor/viewer)
	Permission OrgMemberRole `json:"permission" gorm:"type:varchar(32);not null;default:'viewer'"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last updated time
	UpdatedAt time.Time `json:"updated_at"`
	// Deletion time (soft delete)
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`

	// Associations (not stored in database)
	KnowledgeBase *KnowledgeBase `json:"knowledge_base,omitempty" gorm:"foreignKey:KnowledgeBaseID"`
	Organization  *Organization  `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

// TableName returns the table name for GORM
func (KnowledgeBaseShare) TableName() string {
	return "kb_shares"
}

// SharedKnowledgeBaseInfo represents a shared knowledge base with additional sharing info
type SharedKnowledgeBaseInfo struct {
	KnowledgeBase  *KnowledgeBase `json:"knowledge_base"`
	ShareID        string         `json:"share_id"`
	OrganizationID string         `json:"organization_id"`
	OrgName        string         `json:"org_name"`
	Permission     OrgMemberRole  `json:"permission"`
	SourceTenantID uint64         `json:"source_tenant_id"`
	SharedAt       time.Time      `json:"shared_at"`
}

// AgentShare represents a sharing record of an agent to an organization
type AgentShare struct {
	ID             string         `json:"id" gorm:"type:varchar(36);primaryKey"`
	AgentID        string         `json:"agent_id" gorm:"type:varchar(36);not null;index"`
	OrganizationID string         `json:"organization_id" gorm:"type:varchar(36);not null;index"`
	SharedByUserID string         `json:"shared_by_user_id" gorm:"type:varchar(36);not null"`
	SourceTenantID uint64         `json:"source_tenant_id" gorm:"not null;index"`
	Permission     OrgMemberRole  `json:"permission" gorm:"type:varchar(32);not null;default:'viewer'"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"deleted_at" gorm:"index"`
	Agent          *CustomAgent   `json:"agent,omitempty" gorm:"foreignKey:AgentID,SourceTenantID;references:ID,TenantID"`
	Organization   *Organization  `json:"organization,omitempty" gorm:"foreignKey:OrganizationID"`
}

// TableName returns the table name for GORM
func (AgentShare) TableName() string {
	return "agent_shares"
}

// SharedAgentInfo represents a shared agent with additional sharing info
type SharedAgentInfo struct {
	Agent            *CustomAgent  `json:"agent"`
	ShareID          string        `json:"share_id"`
	OrganizationID   string        `json:"organization_id"`
	OrgName          string        `json:"org_name"`
	Permission       OrgMemberRole `json:"permission"`
	SourceTenantID   uint64        `json:"source_tenant_id"`
	SharedAt         time.Time     `json:"shared_at"`
	SharedByUserID   string        `json:"shared_by_user_id,omitempty"`
	SharedByUsername string        `json:"shared_by_username,omitempty"`
	// DisabledByMe: current tenant has hidden this shared agent from their conversation dropdown (per-user preference)
	DisabledByMe bool `json:"disabled_by_me"`
}

// SourceFromAgentInfo indicates the KB is visible in the space via a shared agent (read-only, no KB share record).
type SourceFromAgentInfo struct {
	AgentID         string `json:"agent_id"`
	AgentName       string `json:"agent_name"`
	KBSelectionMode string `json:"kb_selection_mode"` // "all" | "selected" | "none"; for drawer copy "该智能体对知识库的策略"
}

// OrganizationSharedKnowledgeBaseItem is used by GET /organizations/:id/shared-knowledge-bases (space-scoped list including mine).
// When SourceFromAgent is set, the KB is from a shared agent's config (no direct KB share); show as read-only and "来自智能体 XXX".
type OrganizationSharedKnowledgeBaseItem struct {
	SharedKnowledgeBaseInfo
	IsMine          bool                 `json:"is_mine"`
	SourceFromAgent *SourceFromAgentInfo `json:"source_from_agent,omitempty"`
}

// OrganizationSharedAgentItem is used by GET /organizations/:id/shared-agents (space-scoped list including mine).
type OrganizationSharedAgentItem struct {
	SharedAgentInfo
	IsMine bool `json:"is_mine"`
}

// TenantDisabledSharedAgent records that a tenant has "disabled" a shared agent for their own dropdown
type TenantDisabledSharedAgent struct {
	TenantID       uint64    `json:"tenant_id" gorm:"primaryKey"`
	AgentID        string    `json:"agent_id" gorm:"type:varchar(36);primaryKey"`
	SourceTenantID uint64    `json:"source_tenant_id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at"`
}

// TableName returns the table name for GORM
func (TenantDisabledSharedAgent) TableName() string {
	return "tenant_disabled_shared_agents"
}

// ----------------------
// Request/Response Types
// ----------------------

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name        string `json:"name" binding:"required,min=1,max=255"`
	Description string `json:"description" binding:"max=1000"`
	Avatar      string `json:"avatar" binding:"omitempty,max=512"` // optional avatar URL
	MemberLimit *int   `json:"member_limit"`                       // optional: max members; 0=unlimited; default 50
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=1,max=255"`
	Description *string `json:"description" binding:"omitempty,max=1000"`
	Avatar      *string `json:"avatar" binding:"omitempty,max=512"` // optional avatar URL
	MemberLimit *int    `json:"member_limit"`                       // max members; 0=unlimited
}

// AddMemberRequest represents a request to add a member to an organization
type AddMemberRequest struct {
	Email string        `json:"email" binding:"required,email"`
	Role  OrgMemberRole `json:"role" binding:"required"`
}

// UpdateMemberRoleRequest represents a request to update a member's role
type UpdateMemberRoleRequest struct {
	Role OrgMemberRole `json:"role" binding:"required"`
}

// InviteMemberRequest represents a request to directly add a member to an organization.
//
// Callers should set RepresentativeUserID/UserID. TenantID is a hidden source
// context used to validate the account against member management. For backward
// compatibility with older SDK callers that still send UserID alone, the
// handler resolves one active TenantID for that user.
type InviteMemberRequest struct {
	// TenantID is the hidden member-management source context.
	TenantID uint64 `json:"tenant_id"`
	// RepresentativeUserID identifies the account receiving access.
	RepresentativeUserID string `json:"representative_user_id"`
	// UserID is retained for backward compatibility. When set without
	// TenantID, the handler resolves the user's TenantID and uses this
	// user as the representative.
	UserID string        `json:"user_id"`
	Role   OrgMemberRole `json:"role" binding:"required"` // Role to assign: admin/editor/viewer
}

// ShareKnowledgeBaseRequest represents a request to share a knowledge base
type ShareKnowledgeBaseRequest struct {
	OrganizationID string        `json:"organization_id" binding:"required"`
	Permission     OrgMemberRole `json:"permission" binding:"required"`
}

// UpdateSharePermissionRequest represents a request to update share permission
type UpdateSharePermissionRequest struct {
	Permission OrgMemberRole `json:"permission" binding:"required"`
}

// OrganizationResponse represents an organization in API responses
type OrganizationResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Avatar      string `json:"avatar,omitempty"`
	OwnerID     string `json:"owner_id"`
	// OwnerTenantID is the persisted owner workspace of the organization.
	OwnerTenantID   uint64    `json:"owner_tenant_id"`
	MemberLimit     int       `json:"member_limit"` // 0 = unlimited
	MemberCount     int       `json:"member_count"`
	ShareCount      int       `json:"share_count"`       // 共享到该组织的知识库数量
	AgentShareCount int       `json:"agent_share_count"` // 共享到该组织的智能体数量
	IsOwner         bool      `json:"is_owner"`
	MyRole          string    `json:"my_role,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// OrganizationMemberResponse represents a member in API responses.
//
// Every row is one concrete account grant. TenantID/TenantName are retained as
// legacy/source context; UserID/RepresentativeUserID are the account identity.
type OrganizationMemberResponse struct {
	ID                   string                        `json:"id"`
	UserID               string                        `json:"user_id"`
	RepresentativeUserID string                        `json:"representative_user_id"`
	Username             string                        `json:"username"`
	Phone                string                        `json:"phone,omitempty"`
	Email                string                        `json:"email"`
	Avatar               string                        `json:"avatar"`
	Role                 string                        `json:"role"`
	TenantID             uint64                        `json:"tenant_id"`
	TenantName           string                        `json:"tenant_name,omitempty"`
	JoinedAt             time.Time                     `json:"joined_at"`
	TenantMembers        []TenantMemberSummaryResponse `json:"tenant_members,omitempty"`
}

// TenantMemberSummaryResponse is retained for old clients. The current member
// roster no longer nests every tenant user under a shared-space row.
type TenantMemberSummaryResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar,omitempty"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// TenantInviteCandidate is a legacy row in the search-tenants-for-invite picker.
type TenantInviteCandidate struct {
	TenantID               uint64 `json:"tenant_id"`
	TenantName             string `json:"tenant_name"`
	RepresentativeUserID   string `json:"representative_user_id"`
	RepresentativeUsername string `json:"representative_username"`
	RepresentativePhone    string `json:"representative_phone,omitempty"`
	RepresentativeEmail    string `json:"representative_email"`
	RepresentativeAvatar   string `json:"representative_avatar,omitempty"`
}

// UserInviteCandidate is one row in the search-users-for-invite picker.
// The UI lets admins choose an active account from tenant_members, the same
// source used by member management. IsAlreadyMember is true when that account
// is already present in the shared space.
type UserInviteCandidate struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	Phone           string `json:"phone,omitempty"`
	Email           string `json:"email"`
	Avatar          string `json:"avatar,omitempty"`
	TenantID        uint64 `json:"tenant_id"`
	TenantName      string `json:"tenant_name,omitempty"`
	IsAlreadyMember bool   `json:"is_already_member"`
}

// KnowledgeBaseShareResponse represents a share record in API responses
type KnowledgeBaseShareResponse struct {
	ID                string    `json:"id"`
	KnowledgeBaseID   string    `json:"knowledge_base_id"`
	KnowledgeBaseName string    `json:"knowledge_base_name"`
	KnowledgeBaseType string    `json:"knowledge_base_type"`
	KnowledgeCount    int64     `json:"knowledge_count"`
	ChunkCount        int64     `json:"chunk_count"`
	OrganizationID    string    `json:"organization_id"`
	OrganizationName  string    `json:"organization_name"`
	SharedByUserID    string    `json:"shared_by_user_id"`
	SharedByUsername  string    `json:"shared_by_username"`
	SourceTenantID    uint64    `json:"source_tenant_id"`
	Permission        string    `json:"permission"`     // Share permission (what the space was granted: viewer/editor)
	MyRoleInOrg       string    `json:"my_role_in_org"` // Current user's role in this organization (admin/editor/viewer)
	MyPermission      string    `json:"my_permission"`  // Effective permission for current user = min(Permission, MyRoleInOrg)
	CreatedAt         time.Time `json:"created_at"`
	RequireApproval   bool      `json:"require_approval"`
}

// AgentShareResponse represents an agent share record in API responses
type AgentShareResponse struct {
	ID               string    `json:"id"`
	AgentID          string    `json:"agent_id"`
	AgentName        string    `json:"agent_name"`
	OrganizationID   string    `json:"organization_id"`
	OrganizationName string    `json:"organization_name"`
	SharedByUserID   string    `json:"shared_by_user_id"`
	SharedByUsername string    `json:"shared_by_username"`
	SourceTenantID   uint64    `json:"source_tenant_id"`
	Permission       string    `json:"permission"`
	MyRoleInOrg      string    `json:"my_role_in_org,omitempty"`
	MyPermission     string    `json:"my_permission,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	// Agent scope summary for list display (from agent config when available)
	ScopeKB        string `json:"scope_kb,omitempty"`       // "all" | "selected" | "none"
	ScopeKBCount   int    `json:"scope_kb_count,omitempty"` // when selected
	ScopeWebSearch bool   `json:"scope_web_search,omitempty"`
	ScopeMCP       string `json:"scope_mcp,omitempty"`       // "all" | "selected" | "none"
	ScopeMCPCount  int    `json:"scope_mcp_count,omitempty"` // when selected
	// Agent avatar (emoji or icon name) for list display
	AgentAvatar string `json:"agent_avatar,omitempty"`
}

// ListOrganizationsResponse represents the response for listing organizations
type ListOrganizationsResponse struct {
	Organizations  []OrganizationResponse       `json:"organizations"`
	Total          int64                        `json:"total"`
	ResourceCounts *ResourceCountsByOrgResponse `json:"resource_counts,omitempty"` // 各空间内知识库/智能体数量，供列表侧栏展示
}

// ResourceCountsByOrgResponse is the response for GET /me/resource-counts (sidebar counts per space)
type ResourceCountsByOrgResponse struct {
	KnowledgeBases struct {
		ByOrganization map[string]int `json:"by_organization"`
	} `json:"knowledge_bases"`
	Agents struct {
		ByOrganization map[string]int `json:"by_organization"`
	} `json:"agents"`
}

// ListMembersResponse represents the response for listing members
type ListMembersResponse struct {
	Members []OrganizationMemberResponse `json:"members"`
	Total   int64                        `json:"total"`
}

// ListSharesResponse represents the response for listing shares
type ListSharesResponse struct {
	Shares []KnowledgeBaseShareResponse `json:"shares"`
	Total  int64                        `json:"total"`
}
