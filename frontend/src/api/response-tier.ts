import { get, put } from '@/utils/request'

export type ResponseTier = 'fast' | 'balanced' | 'ultimate'

export interface ResponseTierProfile {
  model_id: string
  thinking?: boolean | null
}

export interface ResponseTierConfig {
  enabled: boolean
  default_tier: ResponseTier
  fast: ResponseTierProfile
  balanced: ResponseTierProfile
  ultimate: ResponseTierProfile
}

export const defaultResponseTierConfig: ResponseTierConfig = {
  enabled: false,
  default_tier: 'balanced',
  fast: { model_id: '', thinking: null },
  balanced: { model_id: '', thinking: null },
  ultimate: { model_id: '', thinking: null },
}

export async function getResponseTierConfig(): Promise<ResponseTierConfig> {
  const response: any = await get('/api/v1/tenants/kv/response-tier-config')
  return (response?.data || response || defaultResponseTierConfig) as ResponseTierConfig
}

export async function updateResponseTierConfig(
  config: ResponseTierConfig,
): Promise<ResponseTierConfig> {
  const response: any = await put('/api/v1/tenants/kv/response-tier-config', config)
  return (response?.data || response) as ResponseTierConfig
}

export async function getSystemResponseTierConfig(): Promise<ResponseTierConfig> {
  const response: any = await get('/api/v1/system/admin/response-tier-config')
  return (response?.data || response || defaultResponseTierConfig) as ResponseTierConfig
}

export async function updateSystemResponseTierConfig(
  config: ResponseTierConfig,
): Promise<ResponseTierConfig> {
  const response: any = await put('/api/v1/system/admin/response-tier-config', config)
  return (response?.data || response) as ResponseTierConfig
}
