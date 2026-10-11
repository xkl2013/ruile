import { get, post, put } from '@/utils/request'

export type ServiceTemplateStatus = 'draft' | 'published' | 'disabled' | 'archived'
export type ServiceTemplateRiskLevel = 'low' | 'medium' | 'high'
export type ServiceTemplateSpaceType = 'customer_service' | 'operations' | 'research'

export interface ServiceTemplateProfileField {
  key: string
  label: string
  value_type: string
  source: string
  required: boolean
  sensitive: boolean
  display_order: number
  aliases?: string[]
  extraction_hint?: string
  confidence_threshold?: number
  overwrite_policy?: string
  ask_when_missing?: boolean
}

export interface ServiceTemplateSummarySection {
  key: string
  label: string
  source_scopes?: string[]
  refresh_policy: string
  display_order: number
}

export interface ServiceTemplateExpertSuggestion {
  expert_ref: string
  expert_name: string
  reason?: string
}

export interface ServiceTemplateBlueprint {
  source_instruction: string
  proposed_space_type: ServiceTemplateSpaceType
  subject_policy: {
    required: boolean
    allowed_types?: string[]
    allow_hierarchy?: boolean
  }
  profile_schema: ServiceTemplateProfileField[]
  summary_schema: ServiceTemplateSummarySection[]
  expert_suggestions?: ServiceTemplateExpertSuggestion[]
  knowledge_base_ids?: string[]
  selected_skills?: string[]
}

export interface AdminServiceTemplate {
  id: string
  tenant_id: number
  key: string
  name: string
  description: string
  version: number
  status: ServiceTemplateStatus
  match_rules: {
    roles?: string[]
    departments?: string[]
    work_profile_terms?: string[]
    service_scope_keys?: string[]
    required_claims?: string[]
    min_confidence_pct?: number
  }
  blueprint: ServiceTemplateBlueprint
  auto_apply: boolean
  auto_activate: boolean
  risk_level: ServiceTemplateRiskLevel
  published_by?: string
  published_at?: string
  created_at: string
  updated_at: string
}

export interface AdminServiceTemplateInput {
  key?: string
  name: string
  description?: string
  space_type: ServiceTemplateSpaceType
  instruction: string
  subject_policy: {
    required: boolean
    allowed_types?: string[]
    allow_hierarchy?: boolean
  }
  expert_suggestions: ServiceTemplateExpertSuggestion[]
  knowledge_base_ids: string[]
  selected_skills: string[]
  match_rules: AdminServiceTemplate['match_rules']
  auto_apply: boolean
  auto_activate: boolean
}

const basePath = '/api/v1/services/admin/templates'

export function listAdminServiceTemplates() {
  return get<{ success: boolean; data: AdminServiceTemplate[] }>(basePath)
}

export function getAdminServiceTemplate(key: string) {
  return get<{ success: boolean; data: AdminServiceTemplate }>(
    `${basePath}/${encodeURIComponent(key)}`,
  )
}

export function createAdminServiceTemplate(input: AdminServiceTemplateInput) {
  return post<{ success: boolean; data: AdminServiceTemplate }>(basePath, input)
}

export function updateAdminServiceTemplate(key: string, input: AdminServiceTemplateInput) {
  return put<{ success: boolean; data: AdminServiceTemplate }>(
    `${basePath}/${encodeURIComponent(key)}`,
    input,
  )
}

export function publishAdminServiceTemplate(key: string) {
  return post<{ success: boolean; data: AdminServiceTemplate }>(
    `${basePath}/${encodeURIComponent(key)}/publish`,
    {},
  )
}

export function disableAdminServiceTemplate(key: string) {
  return post<{ success: boolean; data: AdminServiceTemplate }>(
    `${basePath}/${encodeURIComponent(key)}/disable`,
    {},
  )
}
