import { get, put } from '@/utils/request'

export interface BillingOverview {
  tenant_id: number
  tenant_name: string
  space_type: 'personal' | 'organization' | 'legacy'
  policy: {
    enabled: boolean
    enforcement_mode: 'off' | 'observe' | 'enforce'
    point_micros_per_usd: number
  }
  plan: {
    code: string
    name: string
    edition: string
    status: string
    included_storage_bytes: number
    included_point_micros: number
  }
  subscription: {
    status: string
    billing_interval: string
    current_period_start?: string
    current_period_end?: string
    source: string
  }
  storage: {
    used_bytes: number
    quota_bytes: number
    remaining_bytes: number
    usage_percent: number
    unlimited: boolean
    status: string
  }
  credits: {
    balance_point_micros: number
    period_point_micros: number
  }
  compatibility_mode: boolean
}

export interface BillingOverviewResponse {
  success: boolean
  data?: BillingOverview
  message?: string
}

export async function getBillingOverview(tenantId?: number): Promise<BillingOverviewResponse> {
  return get(
    '/api/v1/billing/overview',
    tenantId ? tenantScopedConfig(tenantId) : undefined,
  ) as unknown as BillingOverviewResponse
}

export interface BillingUsageItem {
  id: string
  model_key: string
  provider: string
  input_tokens: number
  cached_tokens: number
  output_tokens: number
  billed_point_micros: number
  status: string
  failure_code: string
  billing_at: string
}

export interface BillingUsageResponse {
  success: boolean
  data?: BillingUsageItem[]
  message?: string
}

export async function getBillingUsage(
  limit = 20,
  tenantId?: number,
): Promise<BillingUsageResponse> {
  return get(
    `/api/v1/billing/usage?limit=${limit}`,
    tenantId ? tenantScopedConfig(tenantId) : undefined,
  ) as unknown as BillingUsageResponse
}

export interface MemberCreditAllocation {
  id: string
  tenant_id: number
  user_id: string
  period_start_at: string
  period_end_at: string
  allocated_period_point_micros: number
  allocated_balance_point_micros: number
  limit_mode: 'inherit' | 'custom' | 'unlimited'
  monthly_limit_point_micros: number
  overage_policy: 'inherit' | 'block' | 'use_enterprise_balance'
  effective_monthly_limit_point_micros: number
  effective_overage_policy: 'block' | 'use_enterprise_balance'
  status: string
  used_point_micros: number
  input_tokens: number
  output_tokens: number
  reasoning_tokens: number
  ledger_count: number
  last_billing_at?: string
}

export interface MemberCreditAllocationsResponse {
  success: boolean
  data?: MemberCreditAllocation[]
  message?: string
}

function tenantScopedConfig(tenantId: number) {
  return {
    headers: {
      'X-Tenant-ID': String(tenantId),
    },
  }
}

export async function getMemberCreditAllocations(
  tenantId: number,
): Promise<MemberCreditAllocationsResponse> {
  return get(
    '/api/v1/billing/member-policies',
    tenantScopedConfig(tenantId),
  ) as unknown as MemberCreditAllocationsResponse
}

export async function updateMemberCreditPolicy(
  tenantId: number,
  userId: string,
  input: {
    limit_mode: 'inherit' | 'custom' | 'unlimited'
    monthly_limit_points: number
  },
): Promise<{ success: boolean; data?: MemberCreditAllocation; message?: string }> {
  return put(
    `/api/v1/billing/member-policies/${encodeURIComponent(userId)}`,
    input,
    tenantScopedConfig(tenantId),
  ) as unknown as { success: boolean; data?: MemberCreditAllocation; message?: string }
}

export interface TenantBillingPolicy {
  tenant_id: number
  default_member_monthly_limit_point_micros: number
  member_overage_policy: 'block' | 'use_enterprise_balance'
  updated_by_user_id: string
  created_at: string
  updated_at: string
}

export interface TenantBillingPolicyResponse {
  success: boolean
  data?: TenantBillingPolicy
  message?: string
}

export async function getTenantBillingPolicy(
  tenantId: number,
): Promise<TenantBillingPolicyResponse> {
  return get(
    '/api/v1/billing/policy',
    tenantScopedConfig(tenantId),
  ) as unknown as TenantBillingPolicyResponse
}

export async function updateTenantBillingPolicy(
  tenantId: number,
  input: {
    default_member_monthly_limit_points: number
    member_overage_policy: 'block' | 'use_enterprise_balance'
  },
): Promise<TenantBillingPolicyResponse> {
  return put(
    '/api/v1/billing/policy',
    input,
    tenantScopedConfig(tenantId),
  ) as unknown as TenantBillingPolicyResponse
}
