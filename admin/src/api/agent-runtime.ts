import { del, get, post, put } from '@/utils/request'

export interface SandboxConfig {
  id: string
  name: string
  mode: 'docker' | 'local' | 'disabled' | string
  enabled: boolean
  active: boolean
  fallback_enabled: boolean
  timeout_seconds: number
  docker_image?: string
  created_at?: string
  updated_at?: string
}

export interface SandboxCheck {
  config_id: string
  mode: string
  available: boolean
  effective_mode: string
  message?: string
}

export interface SkillSnapshot {
  id: string
  generation: number
  parent_snapshot_id?: string
  state: 'building' | 'active' | 'superseded' | string
  manifest_sha256: string
  manifest?: Record<string, unknown>
  created_at?: string
  activated_at?: string
  superseded_at?: string
}

export interface DelegationRun {
  id: string
  tenant_id: number
  parent_session_id: string
  parent_message_id: string
  member_agent_id: string
  task: string
  status: 'running' | 'succeeded' | 'failed' | string
  started_at?: string
  finished_at?: string
  duration_millis: number
  input_tokens: number
  cached_tokens: number
  output_tokens: number
  reasoning_tokens: number
  call_count: number
  base_cost_nanousd: number
  rated_cost_nanousd: number
  billed_point_micros: number
  usage_ledger_count: number
  output_preview?: string
  error_code?: string
  error_message?: string
  created_at?: string
  updated_at?: string
}

export interface DelegationRunListParams {
  limit?: number
  parent_session_id?: string
  member_agent_id?: string
}

const sandboxPath = '/api/v1/sandbox/configs'
const snapshotPath = '/api/v1/skills/snapshots'
const delegationRunPath = '/api/v1/agent/delegation-runs'

export function listSandboxConfigs() {
  return get<{ data: SandboxConfig[] }>(sandboxPath)
}

export function createSandboxConfig(input: Partial<SandboxConfig>) {
  return post<{ data: SandboxConfig }>(sandboxPath, input)
}

export function updateSandboxConfig(id: string, input: Partial<SandboxConfig>) {
  return put<{ data: SandboxConfig }>(`${sandboxPath}/${encodeURIComponent(id)}`, input)
}

export function activateSandboxConfig(id: string) {
  return post<{ data: SandboxConfig }>(`${sandboxPath}/${encodeURIComponent(id)}/activate`, {})
}

export function deactivateSandboxConfig(id: string) {
  return post<{ data: SandboxConfig }>(`${sandboxPath}/${encodeURIComponent(id)}/deactivate`, {})
}

export function checkSandboxConfig(id: string) {
  return post<{ data: SandboxCheck }>(`${sandboxPath}/${encodeURIComponent(id)}/check`, {})
}

export function deleteSandboxConfig(id: string) {
  return del<void>(`${sandboxPath}/${encodeURIComponent(id)}`)
}

export function listSkillSnapshots() {
  return get<{ data: SkillSnapshot[] }>(snapshotPath)
}

export function createSkillSnapshot() {
  return post<{ data: SkillSnapshot }>(snapshotPath, {})
}

export function activateSkillSnapshot(id: string) {
  return post<{ data: SkillSnapshot }>(`${snapshotPath}/${encodeURIComponent(id)}/activate`, {})
}

export function rollbackSkillSnapshot(id: string) {
  return post<{ data: SkillSnapshot }>(`${snapshotPath}/${encodeURIComponent(id)}/rollback`, {})
}

export function verifySkillSnapshot(id: string) {
  return get<{ data: { valid: boolean } }>(`${snapshotPath}/${encodeURIComponent(id)}/verify`)
}

export function listDelegationRuns(params: DelegationRunListParams = {}) {
  const query = new URLSearchParams()
  if (params.limit) query.set('limit', String(params.limit))
  if (params.parent_session_id) query.set('parent_session_id', params.parent_session_id)
  if (params.member_agent_id) query.set('member_agent_id', params.member_agent_id)
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return get<{ data: DelegationRun[] }>(`${delegationRunPath}${suffix}`)
}

export function getDelegationRun(id: string) {
  return get<{ data: DelegationRun }>(`${delegationRunPath}/${encodeURIComponent(id)}`)
}
