import { get, post, put, del } from '@/utils/request'

// TenantRole mirrors internal/types/tenant_member.go's four-role enum.
// Keep the string values aligned with the Go constants.
export type TenantRole = 'owner' | 'admin' | 'contributor' | 'viewer'

export type TenantMemberStatus = 'active' | 'invited' | 'suspended'
export type TenantMemberSource = 'manual' | 'invite' | 'sso' | 'scim' | 'ldap' | 'hris'

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
  invited_by?: string | null
  joined_at: string
  expires_at?: string | null
  suspended_at?: string | null
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

export interface AddMemberRequest {
  email: string
  role: TenantRole
}

export interface AdminCreateMemberRequest {
  phone: string
  name: string
  role?: TenantRole
}

export interface AddMemberResponse {
  success: boolean
  data?: TenantMember
  message?: string
}

export interface SimpleResponse {
  success: boolean
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
  return (await post(`/api/v1/tenants/${tenantId}/members`, body)) as unknown as AddMemberResponse
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
  return (await put(`/api/v1/tenants/${tenantId}/members/${userId}`, { role })) as unknown as SimpleResponse
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
  return (await del(`/api/v1/tenants/${tenantId}/members/${userId}`)) as unknown as SimpleResponse
}

/**
 * Suspend a member's workspace access and revoke active sessions.
 * Backend: POST /api/v1/tenants/:id/members/:user_id/suspend (Admin+).
 */
export async function suspendMember(
  tenantId: number,
  userId: string,
): Promise<SimpleResponse> {
  return (await post(`/api/v1/tenants/${tenantId}/members/${userId}/suspend`)) as unknown as SimpleResponse
}

/**
 * Restore a suspended member's workspace access.
 * Backend: POST /api/v1/tenants/:id/members/:user_id/reactivate (Admin+).
 */
export async function reactivateMember(
  tenantId: number,
  userId: string,
): Promise<SimpleResponse> {
  return (await post(`/api/v1/tenants/${tenantId}/members/${userId}/reactivate`)) as unknown as SimpleResponse
}

/**
 * Quit the tenant on your own. Same last-Owner invariant as
 * removeMember, but does NOT require Owner+ — any active member can
 * call it.
 * Backend: POST /api/v1/tenants/:id/leave (Viewer+).
 */
export async function leaveTenant(tenantId: number): Promise<SimpleResponse> {
  return (await post(`/api/v1/tenants/${tenantId}/leave`)) as unknown as SimpleResponse
}
