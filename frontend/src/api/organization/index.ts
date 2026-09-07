import { get, post, put, del } from '@/utils/request'

// Organization types
export interface Organization {
  id: string
  name: string
  description: string
  avatar?: string
  owner_id: string
  /** Persisted owner tenant of the organization, retained as workspace context. */
  owner_tenant_id: number
  /** Max members; 0 = unlimited */
  member_limit?: number
  member_count?: number
  share_count?: number
  agent_share_count?: number
  is_owner?: boolean
  my_role?: string
  created_at: string
  updated_at: string
}

/**
 * OrganizationMember represents one concrete member row in the shared space.
 * `user_id` and `representative_user_id` identify the account who receives
 * access. `tenant_id` is hidden user-management source context.
 */
export interface OrganizationMember {
  id: string
  user_id: string
  representative_user_id?: string
  username: string
  phone?: string
  email: string
  avatar?: string
  role: 'admin' | 'editor' | 'viewer'
  tenant_id: number
  tenant_name?: string
  joined_at: string
  /** Legacy-only, no longer rendered by the shared-space member UI. */
  tenant_members?: TenantMemberSummary[]
}

export interface TenantMemberSummary {
  user_id: string
  username?: string
  phone?: string
  email?: string
  avatar?: string
  role?: string
  status?: string
}

export interface KnowledgeBaseShare {
  id: string
  knowledge_base_id: string
  knowledge_base_name?: string
  knowledge_base_type?: string
  knowledge_count?: number
  chunk_count?: number
  organization_id: string
  organization_name?: string
  shared_by_user_id: string
  shared_by_username?: string
  source_tenant_id: number
  /** Share permission: what the space was granted (viewer/editor) */
  permission: 'admin' | 'editor' | 'viewer'
  /** Current user's role in this organization (admin/editor/viewer) */
  my_role_in_org?: 'admin' | 'editor' | 'viewer'
  /** Effective permission for current user = min(permission, my_role_in_org) */
  my_permission?: 'admin' | 'editor' | 'viewer'
  created_at: string
}

export interface SharedKnowledgeBase {
  knowledge_base: {
    id: string
    name: string
    description: string
    icon?: string
    icon_url?: string
    type: string
    knowledge_count?: number
    chunk_count?: number
  }
  share_id: string
  organization_id: string
  org_name: string
  permission: 'admin' | 'editor' | 'viewer'
  source_tenant_id: number
  shared_at: string
}

/** When set, this KB is visible in the space via a shared agent (read-only, no direct KB share) */
export interface SourceFromAgentInfo {
  agent_id: string
  agent_name: string
  /** "all" | "selected" | "none" — for showing agent's KB strategy in the drawer */
  kb_selection_mode?: string
}

/** Item from GET /organizations/:id/shared-knowledge-bases (space-scoped list including mine and agent-carried) */
export type OrganizationSharedKnowledgeBaseItem = SharedKnowledgeBase & {
  is_mine: boolean
  /** Present when the KB is from a shared agent's config (not directly shared to the space) */
  source_from_agent?: SourceFromAgentInfo
}

// Request types
export interface CreateOrganizationRequest {
  name: string
  description?: string
  avatar?: string
  member_limit?: number // 0=unlimited; default 50
}

export interface UpdateOrganizationRequest {
  name?: string
  description?: string
  avatar?: string
  member_limit?: number // 0=unlimited
}

export interface UpdateMemberRoleRequest {
  role: 'admin' | 'editor' | 'viewer'
}

export interface ShareKnowledgeBaseRequest {
  organization_id: string
  permission: 'admin' | 'editor' | 'viewer'
}

export interface UpdateSharePermissionRequest {
  permission: 'admin' | 'editor' | 'viewer'
}

// Response types
export interface ApiResponse<T> {
  success: boolean
  data?: T
  message?: string
}

/** Per-org resource counts (included in list my organizations to avoid extra GET /me/resource-counts) */
export interface ResourceCountsByOrg {
  knowledge_bases: { by_organization: Record<string, number> }
  agents: { by_organization: Record<string, number> }
}

export interface ListOrganizationsResponse {
  organizations: Organization[]
  total: number
  resource_counts?: ResourceCountsByOrg
}

export interface ListMembersResponse {
  members: OrganizationMember[]
  total: number
}

/** InviteMemberRequest adds a concrete member to the organization. */
export interface InviteMemberRequest {
  tenant_id?: number
  representative_user_id?: string
  user_id?: string
  role: 'admin' | 'editor' | 'viewer'
}

/**
 * UserSearchResult is backed by active tenant_members rows in the current
 * workspace user-management data. Results are presented account-first;
 * tenant_id is only the hidden source context needed when submitting the
 * add-member request.
 */
export interface UserSearchResult {
  id?: string
  user_id?: string
  username?: string
  phone?: string
  email?: string
  avatar?: string
  tenant_id?: number
  tenant_name?: string
  is_already_member?: boolean
  representative_user_id?: string
  representative_username?: string
  representative_phone?: string
  representative_email?: string
  representative_avatar?: string
}

/**
 * TenantInviteCandidate is one row in the search-tenants-for-invite picker.
 * The picker is tenant-centric; the representative_* fields describe the
 * user that caused the tenant to appear in the search (for display only).
 */
export interface TenantInviteCandidate {
  tenant_id: number
  tenant_name: string
  representative_user_id: string
  representative_username: string
  representative_phone?: string
  representative_email: string
  representative_avatar?: string
}

export interface ListSharesResponse {
  shares: KnowledgeBaseShare[]
  total: number
}

// Agent share types
export interface AgentShareResponse {
  id: string
  agent_id: string
  agent_name?: string
  organization_id: string
  organization_name?: string
  shared_by_user_id: string
  shared_by_username?: string
  source_tenant_id: number
  permission: string
  my_role_in_org?: string
  my_permission?: string
  created_at: string
  /** Agent scope summary for list display */
  scope_kb?: string
  scope_kb_count?: number
  scope_web_search?: boolean
  scope_mcp?: string
  scope_mcp_count?: number
  /** Agent avatar (emoji) for list display */
  agent_avatar?: string
}

export interface SharedAgentInfo {
  agent: { id: string; name: string; description?: string; [key: string]: any }
  share_id: string
  organization_id: string
  org_name: string
  permission: string
  source_tenant_id: number
  shared_at: string
  shared_by_user_id?: string
  shared_by_username?: string
  /** 当前用户是否已停用该共享智能体（仅影响本人对话下拉显示） */
  disabled_by_me?: boolean
}

/** Item from GET /organizations/:id/shared-agents (space-scoped list including mine) */
export type OrganizationSharedAgentItem = SharedAgentInfo & { is_mine: boolean }

export interface ListAgentSharesResponse {
  shares: AgentShareResponse[]
  total: number
}

// Organization API functions

/**
 * Create a new organization
 */
export async function createOrganization(req: CreateOrganizationRequest): Promise<ApiResponse<Organization>> {
  try {
    const response = await post('/api/v1/organizations', req)
    return response as unknown as ApiResponse<Organization>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to create organization' }
  }
}

/**
 * Get organization by ID
 */
export async function getOrganization(id: string): Promise<ApiResponse<Organization>> {
  try {
    const response = await get(`/api/v1/organizations/${id}`)
    return response as unknown as ApiResponse<Organization>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to get organization' }
  }
}

/**
 * List my organizations
 */
export async function listMyOrganizations(): Promise<ApiResponse<ListOrganizationsResponse>> {
  try {
    const response = await get('/api/v1/organizations')
    return response as unknown as ApiResponse<ListOrganizationsResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list organizations' }
  }
}

/**
 * Update organization
 */
export async function updateOrganization(id: string, req: UpdateOrganizationRequest): Promise<ApiResponse<Organization>> {
  try {
    const response = await put(`/api/v1/organizations/${id}`, req)
    return response as unknown as ApiResponse<Organization>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to update organization' }
  }
}

/**
 * Delete organization
 */
export async function deleteOrganization(id: string): Promise<ApiResponse<void>> {
  try {
    const response = await del(`/api/v1/organizations/${id}`)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to delete organization' }
  }
}

/**
 * Leave organization
 */
export async function leaveOrganization(id: string): Promise<ApiResponse<void>> {
  try {
    const response = await post(`/api/v1/organizations/${id}/leave`, {})
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to leave organization' }
  }
}

// Participating user management

/**
 * List organization members
 */
export async function listMembers(orgId: string): Promise<ApiResponse<ListMembersResponse>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/members`)
    return response as unknown as ApiResponse<ListMembersResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list members' }
  }
}

/**
 * Update member role (member is identified by organization member row ID)
 */
export async function updateMemberRole(orgId: string, memberId: string, req: UpdateMemberRoleRequest): Promise<ApiResponse<void>> {
  try {
    const response = await put(`/api/v1/organizations/${orgId}/members/${memberId}`, req)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to update member role' }
  }
}

/**
 * Remove member (member is identified by organization member row ID)
 */
export async function removeMember(orgId: string, memberId: string): Promise<ApiResponse<void>> {
  try {
    const response = await del(`/api/v1/organizations/${orgId}/members/${memberId}`)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to remove member' }
  }
}

// Knowledge base sharing

/**
 * Share knowledge base to organization
 */
export async function shareKnowledgeBase(kbId: string, req: ShareKnowledgeBaseRequest): Promise<ApiResponse<KnowledgeBaseShare>> {
  try {
    const response = await post(`/api/v1/knowledge-bases/${kbId}/shares`, req)
    return response as unknown as ApiResponse<KnowledgeBaseShare>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to share knowledge base' }
  }
}

/**
 * List shares for a knowledge base
 */
export async function listKBShares(kbId: string): Promise<ApiResponse<ListSharesResponse>> {
  try {
    const response = await get(`/api/v1/knowledge-bases/${kbId}/shares`)
    return response as unknown as ApiResponse<ListSharesResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list shares' }
  }
}

/**
 * Update share permission
 */
export async function updateSharePermission(kbId: string, shareId: string, req: UpdateSharePermissionRequest): Promise<ApiResponse<void>> {
  try {
    const response = await put(`/api/v1/knowledge-bases/${kbId}/shares/${shareId}`, req)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to update share permission' }
  }
}

/**
 * Remove share
 */
export async function removeShare(kbId: string, shareId: string): Promise<ApiResponse<void>> {
  try {
    const response = await del(`/api/v1/knowledge-bases/${kbId}/shares/${shareId}`)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to remove share' }
  }
}

/**
 * List shared knowledge bases (shared to me through organizations)
 */
export async function listSharedKnowledgeBases(): Promise<ApiResponse<SharedKnowledgeBase[]>> {
  try {
    const response = await get('/api/v1/shared-knowledge-bases')
    return response as unknown as ApiResponse<SharedKnowledgeBase[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list shared knowledge bases' }
  }
}

/**
 * List all knowledge bases in an organization (including those shared by current tenant), for list page when a space is selected.
 */
export async function listOrganizationSharedKnowledgeBases(orgId: string): Promise<ApiResponse<OrganizationSharedKnowledgeBaseItem[]>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/shared-knowledge-bases`)
    return response as unknown as ApiResponse<OrganizationSharedKnowledgeBaseItem[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list organization shared knowledge bases' }
  }
}

/**
 * List knowledge bases shared to a specific organization
 */
export async function listOrgShares(orgId: string): Promise<ApiResponse<ListSharesResponse>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/shares`)
    return response as unknown as ApiResponse<ListSharesResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list organization shares' }
  }
}

// Agent sharing
export async function shareAgent(agentId: string, req: ShareKnowledgeBaseRequest): Promise<ApiResponse<AgentShareResponse>> {
  try {
    const response = await post(`/api/v1/agents/${agentId}/shares`, req)
    return response as unknown as ApiResponse<AgentShareResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to share agent' }
  }
}

export async function listAgentShares(agentId: string): Promise<ApiResponse<ListAgentSharesResponse>> {
  try {
    const response = await get(`/api/v1/agents/${agentId}/shares`)
    return response as unknown as ApiResponse<ListAgentSharesResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list agent shares' }
  }
}

export async function updateAgentSharePermission(agentId: string, shareId: string, req: UpdateSharePermissionRequest): Promise<ApiResponse<void>> {
  try {
    const response = await put(`/api/v1/agents/${agentId}/shares/${shareId}`, req)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to update share permission' }
  }
}

export async function removeAgentShare(agentId: string, shareId: string): Promise<ApiResponse<void>> {
  try {
    const response = await del(`/api/v1/agents/${agentId}/shares/${shareId}`)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to remove share' }
  }
}

export async function listSharedAgents(): Promise<ApiResponse<SharedAgentInfo[]>> {
  try {
    const response = await get('/api/v1/shared-agents')
    return response as unknown as ApiResponse<SharedAgentInfo[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list shared agents' }
  }
}

/**
 * List all agents in an organization (including those shared by current tenant), for list page when a space is selected.
 */
export async function listOrganizationSharedAgents(orgId: string): Promise<ApiResponse<OrganizationSharedAgentItem[]>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/shared-agents`)
    return response as unknown as ApiResponse<OrganizationSharedAgentItem[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list organization shared agents' }
  }
}

/** 设置当前用户对某共享智能体的停用状态（仅影响本人对话下拉显示） */
export async function setSharedAgentDisabledByMe(
  agentId: string,
  disabled: boolean
): Promise<ApiResponse<void>> {
  try {
    const response = await post('/api/v1/shared-agents/disabled', {
      agent_id: agentId,
      disabled
    })
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to update preference' }
  }
}

export async function listOrgAgentShares(orgId: string): Promise<ApiResponse<ListAgentSharesResponse>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/agent-shares`)
    return response as unknown as ApiResponse<ListAgentSharesResponse>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to list organization agent shares' }
  }
}

/**
 * Search candidate workspaces for legacy inviting flows.
 */
export async function searchTenantsForInvite(
  orgId: string,
  query: string,
  limit: number = 10
): Promise<ApiResponse<TenantInviteCandidate[]>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/search-tenants?q=${encodeURIComponent(query)}&limit=${limit}`)
    return response as unknown as ApiResponse<TenantInviteCandidate[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to search tenants' }
  }
}

export async function searchUsersForInvite(
  orgId: string,
  query: string = '',
  limit: number = 10
): Promise<ApiResponse<UserSearchResult[]>> {
  try {
    const response = await get(`/api/v1/organizations/${orgId}/search-users?q=${encodeURIComponent(query)}&limit=${limit}`)
    return response as unknown as ApiResponse<UserSearchResult[]>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to search member candidates' }
  }
}

/**
 * Add a member to the organization directly (admin only). Prefer sending
 * representative_user_id/user_id; tenant_id is retained as hidden source
 * context for validation against user management.
 */
export async function inviteMember(
  orgId: string,
  req: InviteMemberRequest
): Promise<ApiResponse<void>> {
  try {
    const response = await post(`/api/v1/organizations/${orgId}/invite`, req)
    return response as unknown as ApiResponse<void>
  } catch (error: any) {
    return { success: false, message: error.message || 'Failed to invite member' }
  }
}
