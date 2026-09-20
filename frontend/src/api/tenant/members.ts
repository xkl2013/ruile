import { get, post, put, del } from '@/utils/request'

// TenantRole mirrors internal/types/tenant_member.go's four-role enum.
// Keep the string values aligned with the Go constants.
export type TenantRole = 'owner' | 'admin' | 'contributor' | 'viewer'

export type TenantMemberStatus = 'active' | 'invited' | 'suspended'
export type TenantMemberSource = 'manual' | 'invite' | 'sso' | 'scim' | 'ldap' | 'hris'

export const DEFAULT_WORK_PROFILE_DESCRIPTION = `1.【岗位与执教履历】
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

// TenantMember is the API projection of a (user, tenant) membership row,
// already joined with the user's email/username/avatar by the backend.
export interface TenantMember {
  user_id: string
  email: string
  username: string
  avatar?: string
  role: TenantRole
  status: TenantMemberStatus
  source?: TenantMemberSource
  external_user_id?: string
  department?: string
  work_profile_description?: string
  invited_by?: string | null
  joined_at: string
  expires_at?: string | null
  suspended_at?: string | null
}

export type MemberAssetType = 'knowledge_base' | 'agent'
export type MemberAssetTransferTargetType = 'member' | 'enterprise'
export type MemberAssetTransferScope = 'all' | 'selected'

export interface MemberTransferableAsset {
  id: string
  name: string
  type: MemberAssetType
}

export interface MemberTransferableAssets {
  tenant_id: number
  source_user_id: string
  knowledge_bases: MemberTransferableAsset[]
  agents: MemberTransferableAsset[]
  total: number
}

export interface MemberAssetTransferRequest {
  target_type: MemberAssetTransferTargetType
  target_user_id?: string
  scope: MemberAssetTransferScope
  asset_types?: MemberAssetType[]
  knowledge_base_ids?: string[]
  agent_ids?: string[]
  reason: string
}

export interface MemberAssetTransferResult {
  tenant_id: number
  source_user_id: string
  target_type: MemberAssetTransferTargetType
  target_user_id?: string
  knowledge_bases_transferred: number
  agents_transferred: number
  total_transferred: number
}

export interface MemberTransferableAssetsResponse {
  success: boolean
  data?: MemberTransferableAssets
  message?: string
}

export interface MemberAssetTransferResponse {
  success: boolean
  data?: MemberAssetTransferResult
  message?: string
}

export interface ListMembersResponse {
  success: boolean
  data?: {
    members: TenantMember[]
    total: number
    page?: number
    page_size?: number
  }
  message?: string
}

export interface ListMembersParams {
  page?: number
  page_size?: number
  /** 按邮箱/用户名筛选（服务端模糊匹配） */
  q?: string
  role?: TenantRole | ''
  status?: TenantMemberStatus | ''
  source?: TenantMemberSource | ''
  department?: string
}

function buildMembersQuery(params: ListMembersParams | undefined): string {
  if (!params) return ''
  const u = new URLSearchParams()
  if (params.page != null && params.page > 0) u.set('page', String(params.page))
  if (params.page_size != null && params.page_size > 0) u.set('page_size', String(params.page_size))
  const q = params.q?.trim()
  if (q) u.set('q', q)
  if (params.role) u.set('role', params.role)
  if (params.status) u.set('status', params.status)
  if (params.source) u.set('source', params.source)
  const department = params.department?.trim()
  if (department) u.set('department', department)
  const qs = u.toString()
  return qs ? `?${qs}` : ''
}

function tenantScopedConfig(tenantId: number) {
  return {
    headers: {
      'X-Tenant-ID': String(tenantId),
    },
  }
}

export interface AddMemberRequest {
  email: string
  role: TenantRole
  work_profile_description: string
}

export interface AdminCreateMemberRequest {
  phone: string
  name: string
  role?: TenantRole
  work_profile_description: string
}

export interface AddMemberResponse {
  success: boolean
  data?: TenantMember
  /** True only when admin-create had to create a new global account. */
  account_created?: boolean
  message?: string
}

export interface SimpleResponse {
  success: boolean
  message?: string
}

export interface MyMemberProfile {
  user_id: string
  tenant_id: number
  work_profile_description: string
}

export interface MyMemberProfileResponse {
  success: boolean
  data?: MyMemberProfile
  message?: string
}

export interface GenerateMyMemberProfileRequest {
  prompt: string
}

export interface GenerateMemberWorkProfileRequest {
  job_title: string
  member_name?: string
  existing_description?: string
}

export interface GenerateMemberWorkProfileResponse {
  success: boolean
  data?: {
    description: string
    source: 'ai' | 'fallback'
    model_id?: string
  }
  message?: string
}

/**
 * 分页列出空间成员。
 * Backend: GET /api/v1/tenants/:id/members (Viewer+)。
 * 查询参数：`q`、`role`、`status`、`source`、`department`、`page`、`page_size`。
 */
export async function listMembers(
  tenantId: number,
  params: ListMembersParams = {},
): Promise<ListMembersResponse> {
  const qs = buildMembersQuery(params)
  return (await get(
    `/api/v1/tenants/${tenantId}/members${qs}`,
    tenantScopedConfig(tenantId),
  )) as unknown as ListMembersResponse
}

/**
 * 遍历分页拉取空间的全部成员（每页最大 100，最多 500 页兜底）。
 * 用于「退出空间」等对全量成员的轻量校验；普通表格请直接使用 {@link listMembers} 分页接口。
 */
export async function fetchAllTenantMembers(tenantId: number): Promise<TenantMember[]> {
  const pageSize = 100
  let page = 1
  const out: TenantMember[] = []
  let total = Number.POSITIVE_INFINITY
  for (let guard = 0; guard < 500 && out.length < total; guard++) {
    const resp = await listMembers(tenantId, { page, page_size: pageSize })
    if (!resp.success || !resp.data) break
    total = resp.data.total
    const batch = resp.data.members || []
    if (batch.length === 0 && page >= 2) break
    out.push(...batch)
    if (batch.length < pageSize) break
    page++
  }
  return out
}

/**
 * Invite an existing user (by email) to the tenant with the given role.
 * Backend: POST /api/v1/tenants/:id/members (Admin+; Owner role still needs Owner/system admin).
 *
 * Returns 404 when the email does not match any registered user — the
 * caller should ask the invitee to register first. PR 3 does not yet
 * support email-based invites for users who don't have an account.
 */
export async function addMember(
  tenantId: number,
  body: AddMemberRequest,
): Promise<AddMemberResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as AddMemberResponse
}

/**
 * Create an account from the admin member-management screen and immediately
 * add it to the current tenant. Backend sets the initial password to
 * `rl` + the last six digits of the phone number.
 * Backend: POST /api/v1/tenants/:id/members/admin-create (Admin+).
 */
export async function adminCreateMember(
  tenantId: number,
  body: AdminCreateMemberRequest,
): Promise<AddMemberResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/admin-create`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as AddMemberResponse
}

/**
 * Change an existing member's role.
 * Backend: PUT /api/v1/tenants/:id/members/:user_id (Admin+; Owner role still needs Owner/system admin).
 *
 * Returns 409 when this would demote the last active Owner of the tenant.
 */
export async function updateMemberRole(
  tenantId: number,
  userId: string,
  role: TenantRole,
): Promise<SimpleResponse> {
  return (await put(
    `/api/v1/tenants/${tenantId}/members/${userId}`,
    { role },
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Update tenant-scoped member metadata used by service routing.
 * Backend: PUT /api/v1/tenants/:id/members/:user_id/profile (Admin+).
 */
export async function updateMemberProfile(
  tenantId: number,
  userId: string,
  body: { work_profile_description: string },
): Promise<SimpleResponse> {
  return (await put(
    `/api/v1/tenants/${tenantId}/members/${userId}/profile`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Read the current user's tenant-scoped work profile.
 * Backend: GET /api/v1/tenants/:id/members/me/profile (Viewer+).
 */
export async function getMyMemberProfile(
  tenantId: number,
): Promise<MyMemberProfileResponse> {
  return (await get(
    `/api/v1/tenants/${tenantId}/members/me/profile`,
    tenantScopedConfig(tenantId),
  )) as unknown as MyMemberProfileResponse
}

/**
 * Update the current user's tenant-scoped work profile.
 * Backend: PUT /api/v1/tenants/:id/members/me/profile (Viewer+).
 */
export async function updateMyMemberProfile(
  tenantId: number,
  body: { work_profile_description: string },
): Promise<SimpleResponse> {
  return (await put(
    `/api/v1/tenants/${tenantId}/members/me/profile`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Generate the current user's tenant-scoped work profile from an explicit prompt.
 * Backend: POST /api/v1/tenants/:id/members/me/profile/generate (Viewer+).
 * The response is a draft; callers must explicitly save it.
 */
export async function generateMyMemberProfile(
  tenantId: number,
  body: GenerateMyMemberProfileRequest,
): Promise<GenerateMemberWorkProfileResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/me/profile/generate`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as GenerateMemberWorkProfileResponse
}

/**
 * Generate a member avatar/work-profile description from a job title.
 * Backend: POST /api/v1/tenants/:id/members/work-profile/suggest (Admin+).
 */
export async function generateMemberWorkProfile(
  tenantId: number,
  body: GenerateMemberWorkProfileRequest,
): Promise<GenerateMemberWorkProfileResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/work-profile/suggest`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as GenerateMemberWorkProfileResponse
}

/**
 * List enterprise assets assigned to one member.
 * Backend: GET /api/v1/tenants/:id/members/:user_id/transferable-assets (Admin+).
 */
export async function listMemberTransferableAssets(
  tenantId: number,
  userId: string,
): Promise<MemberTransferableAssetsResponse> {
  return (await get(
    `/api/v1/tenants/${tenantId}/members/${userId}/transferable-assets`,
    tenantScopedConfig(tenantId),
  )) as unknown as MemberTransferableAssetsResponse
}

/**
 * Transfer enterprise knowledge bases and custom agents to another active
 * member or to enterprise-level ownership.
 */
export async function transferMemberAssets(
  tenantId: number,
  userId: string,
  body: MemberAssetTransferRequest,
): Promise<MemberAssetTransferResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/${userId}/asset-transfer`,
    body,
    tenantScopedConfig(tenantId),
  )) as unknown as MemberAssetTransferResponse
}

/**
 * Remove a member from the tenant.
 * Backend: DELETE /api/v1/tenants/:id/members/:user_id (Admin+; removing Owner still needs Owner/system admin).
 *
 * Returns 409 when this would remove the last active Owner.
 */
export async function removeMember(
  tenantId: number,
  userId: string,
): Promise<SimpleResponse> {
  return (await del(
    `/api/v1/tenants/${tenantId}/members/${userId}`,
    undefined,
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Suspend a member's workspace access and revoke active sessions.
 * Backend: POST /api/v1/tenants/:id/members/:user_id/suspend (Admin+).
 */
export async function suspendMember(
  tenantId: number,
  userId: string,
): Promise<SimpleResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/${userId}/suspend`,
    {},
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Restore a suspended member's workspace access.
 * Backend: POST /api/v1/tenants/:id/members/:user_id/reactivate (Admin+).
 */
export async function reactivateMember(
  tenantId: number,
  userId: string,
): Promise<SimpleResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/members/${userId}/reactivate`,
    {},
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}

/**
 * Quit the tenant on your own. Same last-Owner invariant as
 * removeMember, but does NOT require Owner+ — any active member can
 * call it.
 * Backend: POST /api/v1/tenants/:id/leave (Viewer+).
 */
export async function leaveTenant(tenantId: number): Promise<SimpleResponse> {
  return (await post(
    `/api/v1/tenants/${tenantId}/leave`,
    {},
    tenantScopedConfig(tenantId),
  )) as unknown as SimpleResponse
}
