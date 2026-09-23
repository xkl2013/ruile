import { del, get, post, put } from '@/utils/request'
import type {
  AgentRun as ServiceAgentRun,
  AgentRunEvent as ServiceAgentRunEvent,
  AgentRunQuality as ServiceAgentRunQuality,
  AgentRunStep as ServiceAgentRunStep,
  ExpertIntakeInteraction,
  ExpertIntakeQuestion,
  StructuredReportV1,
} from '@/api/agent-run-types'

export type {
  ServiceAgentRun,
  ServiceAgentRunEvent,
  ServiceAgentRunQuality,
  ServiceAgentRunStep,
  ExpertIntakeInteraction,
  ExpertIntakeQuestion,
  StructuredReportV1,
}

export interface ServiceResponse<T> {
  success: boolean
  data: T
  message?: string
}

export type ServiceSpaceState = 'draft' | 'active' | 'paused' | 'archived'

export interface ServiceSpace {
  id: string
  tenant_id: number
  owner_user_id: string
  name: string
  description: string
  instruction: string
  knowledge_base_ids?: string[]
  selected_skills?: string[]
  template_key?: string
  state: ServiceSpaceState
  is_default?: boolean
  visibility?: 'private' | 'tenant'
  member_limit?: number
  role?: 'owner' | 'admin' | 'editor' | 'viewer'
  member_count?: number
  created_at?: string
  updated_at?: string
}

export interface ServiceSession {
  id: string
  tenant_id: number
  user_id: string
  service_id: string
  title: string
  description?: string
  expert_ref?: string
  expert_name?: string
  is_pinned?: boolean
  created_at?: string
  updated_at?: string
}

export interface ServiceExpertBinding {
  id: string
  service_id: string
  expert_ref: string
  expert_name: string
  expert_domain?: string
  source?: string
  enabled: boolean
  display_order?: number
}

export interface ServiceArtifact {
  id: string
  service_id: string
  artifact_id: string
  version_id: string
  kind: string
  format?: string
  title: string
  summary?: string
  resource_id?: string
  resource_ref?: string
  mime_type?: string
  original_name?: string
  lifecycle: 'temporary' | 'saved' | 'shared' | 'archived'
  version: number
  is_current: boolean
  previewable: boolean
  downloadable: boolean
  created_at?: string
  updated_at?: string
}

export interface ServicePage<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

function withQuery<T extends object>(path: string, params?: T) {
  const query = new URLSearchParams()
  Object.entries(params || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      query.set(key, String(value))
    }
  })
  const suffix = query.toString()
  return suffix ? `${path}?${suffix}` : path
}

export function listServiceSpaces(params?: { include_archived?: boolean }) {
  return get<ServiceResponse<ServiceSpace[]>>(
    withQuery('/api/v1/services', params),
  )
}

export function createServiceSpace(input: {
  name: string
  description?: string
  instruction?: string
  knowledge_base_ids?: string[]
  selected_skills?: string[]
  template_key?: string
  activate?: boolean
  experts?: Array<{
    expert_ref: string
    expert_name: string
    expert_domain?: string
    enabled?: boolean
    display_order?: number
  }>
}) {
  return post<ServiceResponse<ServiceSpace>>('/api/v1/services', input)
}

export function updateServiceSpace(
  serviceId: string,
  input: Partial<{
    name: string
    description: string
    instruction: string
    knowledge_base_ids: string[]
    selected_skills: string[]
    visibility: 'private' | 'tenant'
    member_limit: number
  }>,
) {
  return put<ServiceResponse<ServiceSpace>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}`,
    input,
  )
}

export function setServiceSpaceState(serviceId: string, state: ServiceSpaceState) {
  return post<ServiceResponse<ServiceSpace>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/state/${encodeURIComponent(state)}`,
    {},
  )
}

export function deleteServiceSpace(serviceId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}`,
  )
}

export function listServiceSessions(
  serviceId: string,
  params?: { page?: number; page_size?: number; keyword?: string },
) {
  return get<ServiceResponse<ServicePage<ServiceSession>>>(
    withQuery(`/api/v1/services/${encodeURIComponent(serviceId)}/sessions`, params),
  )
}

export function createServiceSession(
  serviceId: string,
  input?: { title?: string; description?: string; expert_ref?: string; expert_name?: string },
) {
  return post<ServiceResponse<ServiceSession>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/sessions`,
    input || {},
  )
}

export function updateServiceSession(
  serviceId: string,
  sessionId: string,
  input: Partial<{ title: string; description: string; expert_ref: string; expert_name: string }>,
) {
  return put<ServiceResponse<ServiceSession>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/sessions/${encodeURIComponent(sessionId)}`,
    input,
  )
}

export function setServiceSessionPinned(serviceId: string, sessionId: string, pinned: boolean) {
  return put<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/sessions/${encodeURIComponent(sessionId)}/pinned`,
    { pinned },
  )
}

export function deleteServiceSession(serviceId: string, sessionId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/sessions/${encodeURIComponent(sessionId)}`,
  )
}

export function listServiceExperts(serviceId: string) {
  return get<ServiceResponse<ServiceExpertBinding[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/experts`,
  )
}

export function replaceServiceExperts(
  serviceId: string,
  experts: Array<{
    expert_ref: string
    expert_name: string
    expert_domain?: string
    enabled?: boolean
    display_order?: number
  }>,
) {
  return put<ServiceResponse<ServiceExpertBinding[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/experts`,
    { experts },
  )
}

export function listServiceArtifacts(
  serviceId: string,
  params?: { page?: number; page_size?: number; lifecycle?: string },
) {
  return get<ServiceResponse<ServicePage<ServiceArtifact>>>(
    withQuery(`/api/v1/services/${encodeURIComponent(serviceId)}/artifacts`, params),
  )
}

export function createServiceAgentRun(
  serviceId: string,
  input: {
    package_id: string
    definition_id: string
    prompt: string
    model_id?: string
    session_id: string
    route_mode?: 'manual' | 'auto'
    answers?: Record<string, unknown>
  },
) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/agent-runs`,
    input,
  )
}

export function getServiceAgentRun(serviceId: string, runId: string) {
  return get<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/agent-runs/${encodeURIComponent(runId)}`,
  )
}

export function listServiceAgentRunSteps(serviceId: string, runId: string) {
  return get<ServiceResponse<ServiceAgentRunStep[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/agent-runs/${encodeURIComponent(runId)}/steps`,
  )
}

export function listServiceAgentRunEvents(
  serviceId: string,
  runId: string,
  after = 0,
) {
  return get<ServiceResponse<ServiceAgentRunEvent[]>>(
    withQuery(
      `/api/v1/services/${encodeURIComponent(serviceId)}/agent-runs/${encodeURIComponent(runId)}/events`,
      { after },
    ),
  )
}
