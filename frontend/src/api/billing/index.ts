import { get } from '@/utils/request'

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

export async function getBillingOverview(): Promise<BillingOverviewResponse> {
  return get('/api/v1/billing/overview') as unknown as BillingOverviewResponse
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

export async function getBillingUsage(limit = 20): Promise<BillingUsageResponse> {
  return get(`/api/v1/billing/usage?limit=${limit}`) as unknown as BillingUsageResponse
}
