import { get, post, put } from '@/utils/request'
import type {
  AgentRun as ServiceAgentRun,
  AgentRunQuality as ServiceAgentRunQuality,
  AgentRunStep as ServiceAgentRunStep,
  ExpertIntakeInteraction,
  ExpertIntakeQuestion,
  StructuredReportV1,
} from '@/api/agent-run-types'

export type {
  ServiceAgentRun,
  ServiceAgentRunQuality,
  ServiceAgentRunStep,
  ExpertIntakeInteraction,
  ExpertIntakeQuestion,
  StructuredReportV1,
}

export interface ServiceWorkProfile {
  id: string
  tenant_id: number
  user_id: string
  name: string
  work_profile_description?: string
  role_type?: string
  campus_scope?: string[]
  course_scope?: string[]
  memory_scope?: string
  tone_preference?: string
  default_profile?: boolean
  enabled?: boolean
  state?: string
  created_at?: string
  updated_at?: string
}

export interface WorkProfileAgentSetting {
  id: string
  tenant_id: number
  profile_id: string
  agent_id: string
  agent_domain: string
  enabled: boolean
  display_name: string
  display_order: number
  memory_filter?: Record<string, unknown>
  knowledge_base_ids?: string[]
  work_doc_directory?: string
  selected_skills?: string[]
  output_policy?: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export interface ServiceAgentTemplate {
  agent_domain: string
  display_name: string
  description: string
  default_enabled: boolean
  user_visible: boolean
  work_doc_directory: string
  memory_filter?: Record<string, unknown>
  output_policy?: Record<string, unknown>
  selected_skills?: string[]
}

export interface ServiceResponse<T> {
  success: boolean
  data: T
  message?: string
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

export function listServiceAgentRunSteps(id: string) {
  return get<ServiceResponse<ServiceAgentRunStep[]>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/steps`,
  )
}

export function getServiceAgentRun(id: string) {
  return get<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}`,
  )
}

export function regenerateServiceAgentRun(id: string, feedback?: string) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/regenerate`,
    feedback?.trim() ? { feedback: feedback.trim() } : {},
  )
}

export function submitServiceAgentRunAnswers(id: string, answers: Record<string, unknown>) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/answers`,
    { answers },
  )
}

export function listServiceAgentTemplates() {
  return get<ServiceResponse<ServiceAgentTemplate[]>>('/api/v1/service/agent-templates')
}

export function listServiceWorkProfiles(params?: { user_id?: string }) {
  return get<ServiceResponse<ServiceWorkProfile[]>>(
    withQuery('/api/v1/service/work-profiles', params),
  )
}

export function listServiceAgentSettings(id: string, params?: { enabled?: boolean }) {
  return get<ServiceResponse<WorkProfileAgentSetting[]>>(
    withQuery(`/api/v1/service/work-profiles/${encodeURIComponent(id)}/agent-settings`, params),
  )
}

export function replaceServiceAgentSettings(
  id: string,
  settings: Partial<WorkProfileAgentSetting>[],
) {
  return put<ServiceResponse<WorkProfileAgentSetting[]>>(
    `/api/v1/service/work-profiles/${encodeURIComponent(id)}/agent-settings`,
    { settings },
  )
}
