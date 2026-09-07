package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Organization represents a collaboration organization
type Organization struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Avatar        string    `json:"avatar,omitempty"`
	OwnerID       string    `json:"owner_id"`
	OwnerTenantID uint64    `json:"owner_tenant_id"`
	MemberLimit   int       `json:"member_limit"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// OrganizationResponse represents an organization in API responses (with counts)
type OrganizationResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	Avatar          string    `json:"avatar,omitempty"`
	OwnerID         string    `json:"owner_id"`
	OwnerTenantID   uint64    `json:"owner_tenant_id"`
	MemberLimit     int       `json:"member_limit"`
	MemberCount     int       `json:"member_count"`
	ShareCount      int       `json:"share_count"`
	AgentShareCount int       `json:"agent_share_count"`
	IsOwner         bool      `json:"is_owner"`
	MyRole          string    `json:"my_role,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
	MemberLimit *int   `json:"member_limit,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
	MemberLimit *int    `json:"member_limit,omitempty"`
}

// OrganizationMemberResponse represents an account grant in API responses.
// TenantID/TenantName are retained as legacy/source context.
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

// TenantMemberSummaryResponse is retained for old clients; current shared-space
// member lists no longer nest workspace members under one row.
type TenantMemberSummaryResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Phone    string `json:"phone,omitempty"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar,omitempty"`
	Role     string `json:"role"`
	Status   string `json:"status"`
}

// KnowledgeBaseShareResponse represents a KB share record in API responses
type KnowledgeBaseShareResponse struct {
	ID                string    `json:"id"`
	KnowledgeBaseID   string    `json:"knowledge_base_id"`
	KnowledgeBaseName string    `json:"knowledge_base_name"`
	OrganizationID    string    `json:"organization_id"`
	OrganizationName  string    `json:"organization_name"`
	SharedByUserID    string    `json:"shared_by_user_id"`
	SharedByUsername  string    `json:"shared_by_username"`
	SourceTenantID    uint64    `json:"source_tenant_id"`
	Permission        string    `json:"permission"`
	MyRoleInOrg       string    `json:"my_role_in_org"`
	MyPermission      string    `json:"my_permission"`
	CreatedAt         time.Time `json:"created_at"`
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
	CreatedAt        time.Time `json:"created_at"`
}

// SharedKnowledgeBaseInfo represents a shared knowledge base
type SharedKnowledgeBaseInfo struct {
	ShareID        string    `json:"share_id"`
	OrganizationID string    `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Permission     string    `json:"permission"`
	SourceTenantID uint64    `json:"source_tenant_id"`
	SharedAt       time.Time `json:"shared_at"`
}

// SharedAgentInfo represents a shared agent
type SharedAgentInfo struct {
	ShareID        string    `json:"share_id"`
	OrganizationID string    `json:"organization_id"`
	OrgName        string    `json:"org_name"`
	Permission     string    `json:"permission"`
	SourceTenantID uint64    `json:"source_tenant_id"`
	SharedAt       time.Time `json:"shared_at"`
}

// UserInfo represents user information for API responses
type UserInfo struct {
	ID                  string    `json:"id"`
	Username            string    `json:"username"`
	Email               string    `json:"email"`
	Avatar              string    `json:"avatar"`
	TenantID            uint64    `json:"tenant_id"`
	IsActive            bool      `json:"is_active"`
	CanAccessAllTenants bool      `json:"can_access_all_tenants"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// --- Organization CRUD ---

// CreateOrganization creates a new organization
func (c *Client) CreateOrganization(ctx context.Context, req *CreateOrganizationRequest) (*OrganizationResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/api/v1/organizations", req, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                  `json:"success"`
		Data    *OrganizationResponse `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListMyOrganizations lists organizations the current user belongs to
func (c *Client) ListMyOrganizations(ctx context.Context) ([]OrganizationResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/organizations", nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Organizations []OrganizationResponse `json:"organizations"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Organizations, nil
}

// GetOrganization gets an organization by ID
func (c *Client) GetOrganization(ctx context.Context, orgID string) (*OrganizationResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s", orgID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                  `json:"success"`
		Data    *OrganizationResponse `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// UpdateOrganization updates an organization
func (c *Client) UpdateOrganization(ctx context.Context, orgID string, req *UpdateOrganizationRequest) (*OrganizationResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/organizations/%s", orgID), req, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                  `json:"success"`
		Data    *OrganizationResponse `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// DeleteOrganization deletes an organization
func (c *Client) DeleteOrganization(ctx context.Context, orgID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/organizations/%s", orgID), nil, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// --- Organization membership ---

// LeaveOrganization leaves an organization
func (c *Client) LeaveOrganization(ctx context.Context, orgID string) error {
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/organizations/%s/leave", orgID), nil, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// SearchUsersForInvite searches users to invite into an organization (admin only)
func (c *Client) SearchUsersForInvite(ctx context.Context, orgID, keyword string) ([]UserInfo, error) {
	q := url.Values{}
	if keyword != "" {
		q.Set("keyword", keyword)
	}
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/search-users", orgID), nil, q)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool       `json:"success"`
		Data    []UserInfo `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// InviteMember directly invites a user to an organization (admin only)
func (c *Client) InviteMember(ctx context.Context, orgID, userID, role string) error {
	req := map[string]string{
		"user_id": userID,
		"role":    role,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/organizations/%s/invite", orgID), req, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// ListMembers lists members of an organization
func (c *Client) ListOrgMembers(ctx context.Context, orgID string) ([]OrganizationMemberResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/members", orgID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Members []OrganizationMemberResponse `json:"members"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Members, nil
}

// UpdateMemberRole updates a shared-space member's role. memberID is the
// organization member row ID.
func (c *Client) UpdateMemberRole(ctx context.Context, orgID, memberID, role string) error {
	req := map[string]string{"role": role}
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, memberID), req, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// RemoveMember removes a shared-space member. memberID is the organization
// member row ID.
func (c *Client) RemoveMember(ctx context.Context, orgID, memberID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/organizations/%s/members/%s", orgID, memberID), nil, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// --- Knowledge base sharing ---

// ShareKnowledgeBase shares a knowledge base with an organization
func (c *Client) ShareKnowledgeBase(ctx context.Context, kbID, orgID, permission string) (*KnowledgeBaseShareResponse, error) {
	req := map[string]string{
		"organization_id": orgID,
		"permission":      permission,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/knowledge-bases/%s/shares", kbID), req, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                        `json:"success"`
		Data    *KnowledgeBaseShareResponse `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListKBShares lists shares of a knowledge base
func (c *Client) ListKBShares(ctx context.Context, kbID string) ([]KnowledgeBaseShareResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/knowledge-bases/%s/shares", kbID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Shares []KnowledgeBaseShareResponse `json:"shares"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Shares, nil
}

// UpdateSharePermission updates a KB share's permission
func (c *Client) UpdateSharePermission(ctx context.Context, kbID, shareID, permission string) error {
	req := map[string]string{"permission": permission}
	resp, err := c.doRequest(ctx, http.MethodPut, fmt.Sprintf("/api/v1/knowledge-bases/%s/shares/%s", kbID, shareID), req, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// RemoveKBShare removes a KB share
func (c *Client) RemoveKBShare(ctx context.Context, kbID, shareID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/knowledge-bases/%s/shares/%s", kbID, shareID), nil, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// --- Agent sharing ---

// ShareAgent shares an agent with an organization
func (c *Client) ShareAgent(ctx context.Context, agentID, orgID, permission string) (*AgentShareResponse, error) {
	req := map[string]string{
		"organization_id": orgID,
		"permission":      permission,
	}
	resp, err := c.doRequest(ctx, http.MethodPost, fmt.Sprintf("/api/v1/agents/%s/shares", agentID), req, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                `json:"success"`
		Data    *AgentShareResponse `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListAgentShares lists shares of an agent
func (c *Client) ListAgentShares(ctx context.Context, agentID string) ([]AgentShareResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/agents/%s/shares", agentID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Shares []AgentShareResponse `json:"shares"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Shares, nil
}

// RemoveAgentShare removes an agent share
func (c *Client) RemoveAgentShare(ctx context.Context, agentID, shareID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/agents/%s/shares/%s", agentID, shareID), nil, nil)
	if err != nil {
		return err
	}
	return parseResponse(resp, nil)
}

// --- Organization shared resources ---

// ListOrgShares lists knowledge bases shared to an organization
func (c *Client) ListOrgShares(ctx context.Context, orgID string) ([]KnowledgeBaseShareResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/shares", orgID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Shares []KnowledgeBaseShareResponse `json:"shares"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Shares, nil
}

// ListOrgAgentShares lists agents shared to an organization
func (c *Client) ListOrgAgentShares(ctx context.Context, orgID string) ([]AgentShareResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, fmt.Sprintf("/api/v1/organizations/%s/agent-shares", orgID), nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool `json:"success"`
		Data    struct {
			Shares []AgentShareResponse `json:"shares"`
		} `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data.Shares, nil
}

// ListSharedKnowledgeBases lists all knowledge bases shared to the current user
func (c *Client) ListSharedKnowledgeBases(ctx context.Context) ([]SharedKnowledgeBaseInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/shared-knowledge-bases", nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool                      `json:"success"`
		Data    []SharedKnowledgeBaseInfo `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}

// ListSharedAgents lists all agents shared to the current user
func (c *Client) ListSharedAgents(ctx context.Context) ([]SharedAgentInfo, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/api/v1/shared-agents", nil, nil)
	if err != nil {
		return nil, err
	}
	var result struct {
		Success bool              `json:"success"`
		Data    []SharedAgentInfo `json:"data"`
	}
	if err := parseResponse(resp, &result); err != nil {
		return nil, err
	}
	return result.Data, nil
}
