package types

import (
	"time"

	"gorm.io/gorm"
)

// TenantRole represents a user's role inside a single tenant.
//
// Tenant roles govern intra-tenant authority (who can create/edit/delete
// resources, manage tenant settings, etc.) and are orthogonal to the
// OrgMemberRole defined in organization.go, which governs cross-tenant
// sharing. A user may carry different TenantRole values in their personal
// workspace and single enterprise workspace (one TenantMember row per
// (user, tenant) pair).
type TenantRole string

const (
	// TenantRoleOwner has full control over the tenant, including tenant
	// deletion, ownership transfer, and managing tenant API keys.
	TenantRoleOwner TenantRole = "owner"
	// TenantRoleAdmin manages users, integrations, and tenant-scoped
	// configuration such as model providers, vector stores, MCP services
	// and IM channels, but cannot delete the tenant or change Owners.
	TenantRoleAdmin TenantRole = "admin"
	// TenantRoleContributor can edit resources they created, plus resources
	// explicitly shared with them.
	TenantRoleContributor TenantRole = "contributor"
	// TenantRoleViewer has read-only access to resources explicitly visible to
	// them and can run agents that are marked as runnable by viewers.
	TenantRoleViewer TenantRole = "viewer"
)

// tenantRoleLevel maps each role to a numeric level used for hierarchy
// comparisons. Higher means more privileged. Levels are spaced by 10 so
// new roles can be inserted between existing ones if needed.
var tenantRoleLevel = map[TenantRole]int{
	TenantRoleOwner:       40,
	TenantRoleAdmin:       30,
	TenantRoleContributor: 20,
	TenantRoleViewer:      10,
}

// IsValid reports whether r is one of the four defined tenant roles.
func (r TenantRole) IsValid() bool {
	_, ok := tenantRoleLevel[r]
	return ok
}

// Level returns the numeric privilege level of the role. Unknown roles
// return 0, which is strictly less than any defined role.
func (r TenantRole) Level() int {
	return tenantRoleLevel[r]
}

// HasPermission reports whether r is at least as privileged as required.
// Used by RequireRole-style middleware to gate endpoints.
func (r TenantRole) HasPermission(required TenantRole) bool {
	return r.Level() >= required.Level()
}

// TenantMemberStatus enumerates the lifecycle states of a membership row.
type TenantMemberStatus string

const (
	// TenantMemberStatusActive is the normal membership state; the user
	// can authenticate into the tenant and is subject to their role.
	TenantMemberStatusActive TenantMemberStatus = "active"
	// TenantMemberStatusInvited represents a pending invitation that has
	// not yet been accepted. The auth middleware treats this as "not a
	// member" until the status flips to active.
	TenantMemberStatusInvited TenantMemberStatus = "invited"
	// TenantMemberStatusSuspended is an admin-revoked membership. The
	// row is preserved for audit trail but the user cannot authenticate
	// into the tenant.
	TenantMemberStatusSuspended TenantMemberStatus = "suspended"
)

// TenantMemberSource records how a membership is managed. Manual rows are
// created inside WeKnora; SSO/SCIM/LDAP/HRIS rows are owned by an enterprise
// directory sync and should generally be edited at the source system.
type TenantMemberSource string

const (
	TenantMemberSourceManual TenantMemberSource = "manual"
	TenantMemberSourceInvite TenantMemberSource = "invite"
	TenantMemberSourceSSO    TenantMemberSource = "sso"
	TenantMemberSourceSCIM   TenantMemberSource = "scim"
	TenantMemberSourceLDAP   TenantMemberSource = "ldap"
	TenantMemberSourceHRIS   TenantMemberSource = "hris"
)

// DefaultWorkProfileDescription is assigned to the owner membership created
// during account provisioning. It gives newly created accounts a usable
// service-facing profile without overwriting later user edits.
const DefaultWorkProfileDescription = `1.【岗位与执教履历】
- 岗位角色：教培工作者，面向学生、家长、教师和教培团队提供教学与服务支持。
- 执教/服务领域：课程教学、学员辅导、家校沟通、课程跟进和教育内容整理。
- 机构环境：根据当前空间提供的课程资料、服务规则和业务信息开展工作，不臆造个人资历或机构信息。

2.【工作与协作偏好】
- 输出格式偏好：优先使用清晰的分点、步骤、表格或可直接复用的文案。
- 沟通与决策风格：先确认事实和目标，再给出具体、稳妥、可执行的建议；涉及不确定信息时明确说明。
- AI 协作期望：协助整理课程资料、分析学习与服务问题、准备家校沟通内容和跟进清单。

3.【近期业务重心】
- 近期关注：提升教学质量、学习效果、家长满意度和续费、转化跟进效率。
- 处理边界：不负责超出本人岗位和授权范围的财务审批、人事决策、跨空间数据访问、医疗或心理诊断以及平台运维。
- 记忆与系统范围：仅读取当前空间授权的知识库、服务规则和与当前工作相关的记忆；涉及外部系统时只生成建议或草稿，不直接执行敏感操作。`

func (s TenantMemberSource) IsValid() bool {
	switch s {
	case TenantMemberSourceManual,
		TenantMemberSourceInvite,
		TenantMemberSourceSSO,
		TenantMemberSourceSCIM,
		TenantMemberSourceLDAP,
		TenantMemberSourceHRIS:
		return true
	default:
		return false
	}
}

// TenantMember represents the (user, tenant) membership record that
// carries the user's TenantRole for that specific tenant.
//
// An account may have its personal workspace plus at most one enterprise
// membership. The home tenant recorded on User.TenantID is always one of
// these rows.
type TenantMember struct {
	// Surrogate primary key.
	ID uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	// UserID references users.id. Together with TenantID forms the logical
	// key enforced by the partial unique index uniq_user_tenant.
	UserID string `json:"user_id" gorm:"type:varchar(36);not null;index"`
	// TenantID references tenants.id.
	TenantID uint64 `json:"tenant_id" gorm:"not null;index"`
	// Role held by the user inside this tenant.
	Role TenantRole `json:"role" gorm:"type:varchar(20);not null;default:'contributor'"`
	// Status controls whether this membership is honoured by the auth
	// middleware; see TenantMemberStatus constants.
	Status TenantMemberStatus `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	// InvitedBy records the user ID of the admin who created this row via
	// an invitation flow. Nil for rows created by self-service registration.
	InvitedBy *string `json:"invited_by,omitempty" gorm:"type:varchar(36)"`
	// Source records the system that owns or created the membership.
	Source TenantMemberSource `json:"source" gorm:"type:varchar(32);not null;default:'manual';index"`
	// ExternalUserID links the membership to an IdP/SCIM/LDAP/HRIS identity.
	ExternalUserID string `json:"external_user_id,omitempty" gorm:"type:varchar(128);not null;default:''"`
	// Department is a lightweight enterprise directory attribute for member
	// filtering and future group/ACL mapping.
	Department string `json:"department,omitempty" gorm:"type:varchar(128);not null;default:'';index"`
	// WorkProfileDescription describes the member's service-facing work avatar.
	// AI service routing reads this text to infer scope and capability hints.
	WorkProfileDescription string `json:"work_profile_description,omitempty" gorm:"type:text;not null;default:''"`
	// ExpiresAt supports guest / contractor access windows.
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	// SuspendedAt records when an operator or directory sync disabled access.
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	// JoinedAt is when the membership became active.
	JoinedAt  time.Time      `json:"joined_at"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

// TableName binds TenantMember to the tenant_members table.
func (TenantMember) TableName() string {
	return "tenant_members"
}

// Membership is the login-response-friendly projection of a TenantMember
// joined with tenant name. Returned as part of LoginResponse so the
// frontend can render a tenant switcher and gate UI by role.
type Membership struct {
	TenantID   uint64     `json:"tenant_id"`
	TenantName string     `json:"tenant_name"`
	Role       TenantRole `json:"role"`
	SpaceType  *SpaceType `json:"space_type,omitempty"`
}

// TenantMemberResponse is the API projection of a TenantMember row joined
// with the human-facing user fields the management UI needs (email,
// username, avatar). It is intentionally NOT the GORM model: returning
// the model directly would leak DeletedAt/UpdatedAt and lock the DB
// schema into the public API. Use this for `/tenants/:id/members` only.
type TenantMemberResponse struct {
	UserID                 string             `json:"user_id"`
	Email                  string             `json:"email"`
	Username               string             `json:"username"`
	Avatar                 string             `json:"avatar,omitempty"`
	Role                   TenantRole         `json:"role"`
	Status                 TenantMemberStatus `json:"status"`
	Source                 TenantMemberSource `json:"source"`
	ExternalUserID         string             `json:"external_user_id,omitempty"`
	Department             string             `json:"department,omitempty"`
	WorkProfileDescription string             `json:"work_profile_description,omitempty"`
	InvitedBy              *string            `json:"invited_by,omitempty"`
	JoinedAt               time.Time          `json:"joined_at"`
	ExpiresAt              *time.Time         `json:"expires_at,omitempty"`
	SuspendedAt            *time.Time         `json:"suspended_at,omitempty"`
}

// TenantMemberListFilter is the server-side filter set for enterprise member
// lists. Empty fields mean "all".
type TenantMemberListFilter struct {
	Query      string
	Role       TenantRole
	Status     TenantMemberStatus
	Source     TenantMemberSource
	Department string
}
