import { get, post, postUpload } from '@/utils/request'
import type { AgentRun } from '@/api/agent-run-types'

export interface ExpertPackageResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface ExpertPackageFileInput {
  path: string
  content: string
}

export interface ExpertPackageImportInput {
  source_format: string
  package_key: string
  version: string
  display_name: string
  description?: string
  avatar?: string
  source_uri?: string
  license?: string
  files: ExpertPackageFileInput[]
}

export interface ExpertPackageDiagnostics {
  blocking?: string[]
  warnings?: string[]
  [key: string]: unknown
}

export interface AgentDefinitionVersion {
  id: string
  tenant_id: number
  package_id: string
  package_version_id: string
  agent_id: string
  version: string
  display_name: string
  description?: string
  domain?: string
  system_prompt?: string
  compiled_config?: Record<string, unknown>
  skills?: string[]
  capabilities?: Record<string, unknown>
  output_contract?: string
  definition_hash?: string
  created_at?: string
  updated_at?: string
}

export interface PublishedExpert {
  package_id: string
  package_version_id: string
  package_key: string
  package_display_name: string
  package_description?: string
  avatar?: string
  package_version: string
  definition_id: string
  agent_id: string
  version: string
  display_name: string
  description?: string
  domain?: string
  output_contract?: string
  skills?: string[]
  capabilities?: Record<string, unknown>
}

export interface PublishedExpertRouteCandidate {
  package_id: string
  package_version_id: string
  package_display_name: string
  avatar?: string
  definition_id: string
  agent_id: string
  version: string
  display_name: string
  description?: string
  domain?: string
  score: number
  confidence: number
}

export interface PublishedExpertRouteDecision {
  route_mode: 'auto' | 'manual'
  selected_expert_id: string
  selected_version: string
  confidence: number
  candidates: PublishedExpertRouteCandidate[]
  routing_reason: string
  requires_confirmation: boolean
}

export interface ExpertPackageVersion {
  id: string
  package_id: string
  version: string
  state: string
  manifest?: Record<string, unknown>
  package_hash: string
  diagnostics?: ExpertPackageDiagnostics
  created_by?: string
  published_by?: string
  published_at?: string
  created_at?: string
  updated_at?: string
  definitions?: AgentDefinitionVersion[]
}

export interface ExpertPackage {
  id: string
  tenant_id: number
  package_key: string
  display_name: string
  description?: string
  avatar?: string
  source_format: string
  source_uri?: string
  license?: string
  created_by?: string
  created_at?: string
  updated_at?: string
  versions?: ExpertPackageVersion[]
}

export interface ExpertAgentTestInput {
  prompt: string
  model_id?: string
  profile_id?: string
  feedback?: string
  route_mode?: 'auto' | 'manual'
  routing_decision?: Record<string, unknown>
  user_confirmed?: boolean
}

export interface ExpertFollowUpInput {
  prompt: string
  mode?: 'explanation'
}

export function listExpertPackages() {
  return get<ExpertPackageResponse<ExpertPackage[]>>('/api/v1/admin/expert-packages')
}

export function getExpertPackage(id: string) {
  return get<ExpertPackageResponse<ExpertPackage>>(`/api/v1/admin/expert-packages/${encodeURIComponent(id)}`)
}

export function importExpertPackageJSON(data: ExpertPackageImportInput) {
  return post<ExpertPackageResponse<ExpertPackageVersion>>('/api/v1/admin/expert-packages/import-json', data)
}

export function importExpertPackageArchive(
  file: File,
  onUploadProgress?: (progressEvent: { loaded?: number; total?: number }) => void,
) {
  const form = new FormData()
  form.append('file', file)
  return postUpload(
    '/api/v1/admin/expert-packages/import',
    form,
    onUploadProgress,
    { timeout: 300000 },
  ) as Promise<ExpertPackageResponse<ExpertPackageVersion>>
}

export function publishExpertPackageVersion(packageId: string, versionId: string) {
  return post<void>(
    `/api/v1/admin/expert-packages/${encodeURIComponent(packageId)}/versions/${encodeURIComponent(versionId)}/publish`,
  )
}

export function listPublishedExperts() {
  return get<ExpertPackageResponse<PublishedExpert[]>>('/api/v1/expert-packages/published')
}

export function routePublishedExpert(prompt: string, modelId?: string) {
  return post<ExpertPackageResponse<PublishedExpertRouteDecision>>(
    '/api/v1/expert-packages/published/route',
    { prompt, model_id: modelId || undefined },
  )
}

export function runPublishedExpert(
  packageId: string,
  definitionId: string,
  data: ExpertAgentTestInput,
) {
  return post<ExpertPackageResponse<AgentRun>>(
    `/api/v1/expert-packages/published/${encodeURIComponent(packageId)}/definitions/${encodeURIComponent(definitionId)}/runs`,
    data,
  )
}

export function runPublishedExpertAuto(data: ExpertAgentTestInput) {
  return post<ExpertPackageResponse<AgentRun>>(
    '/api/v1/expert-packages/published/auto-runs',
    { ...data, route_mode: 'auto' },
  )
}

export function runPublishedExpertFollowUp(runId: string, data: ExpertFollowUpInput) {
  return post<ExpertPackageResponse<AgentRun>>(
    `/api/v1/expert-packages/published/runs/${encodeURIComponent(runId)}/follow-ups`,
    data,
  )
}

export function testExpertPackageAgent(
  packageId: string,
  definitionId: string,
  data: ExpertAgentTestInput,
) {
  return post<ExpertPackageResponse<AgentRun>>(
    `/api/v1/admin/expert-packages/${encodeURIComponent(packageId)}/definitions/${encodeURIComponent(definitionId)}/test-runs`,
    data,
  )
}
