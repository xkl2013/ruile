import { del, get, patch, post, put } from '@/utils/request'
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
export type ServiceSpaceType = 'customer_service' | 'operations' | 'research'

export interface ServiceSpace {
  id: string
  tenant_id: number
  owner_user_id: string
  space_type: ServiceSpaceType
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

export interface ServiceSubject {
  id: string
  tenant_id: number
  service_id: string
  owner_user_id?: string
  subject_type: string
  subject_key: string
  display_name: string
  parent_subject_id?: string | null
  metadata?: Record<string, unknown>
  visibility_scope?: 'private' | 'tenant' | string
  created_at?: string
  updated_at?: string
}

export type ServiceReminderStatusCategory = 'open' | 'in_progress' | 'done' | 'dismissed'

export interface ServiceReminderStatus {
  id: string
  tenant_id: number
  service_id: string
  status_key: string
  label: string
  category: ServiceReminderStatusCategory
  is_initial: boolean
  is_terminal: boolean
  display_order: number
  color?: string
  description?: string
  is_system: boolean
  enabled: boolean
  created_by?: string
  created_at?: string
  updated_at?: string
}

export interface ServiceReminderStatusTransition {
  id: string
  tenant_id: number
  service_id: string
  from_status_id: string
  to_status_id: string
  allowed_roles?: string[]
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface ServiceReminder {
  id: string
  tenant_id: number
  user_id: string
  service_id: string
  subject_id?: string
  parent_reminder_id?: string
  depth?: number
  title: string
  summary?: string
  status: string
  priority: 'high' | 'medium' | 'low'
  due_at?: string
  due_text?: string
  next_action?: string
  agent_domain?: string
  metadata?: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export interface ServiceReminderAssignee {
  id: string
  tenant_id: number
  service_id: string
  reminder_id: string
  user_id: string
  role: 'primary' | 'member' | string
  assigned_by?: string
  created_at?: string
}

export interface ServiceReminderComment {
  id: string
  tenant_id: number
  service_id: string
  reminder_id: string
  user_id: string
  content: string
  created_at?: string
  updated_at?: string
}

export interface ServiceReminderHistory {
  id: string
  tenant_id: number
  service_id: string
  reminder_id: string
  user_id: string
  action: string
  from_status?: string
  to_status?: string
  change_detail?: Record<string, unknown>
  created_at?: string
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

export interface ServiceSpaceTemplate {
  id: string
  tenant_id: number
  key: string
  name: string
  version: number
  status: string
  match_rules?: Record<string, unknown>
  blueprint: ServiceSpaceBlueprint
  auto_apply: boolean
  auto_activate: boolean
  risk_level: string
  published_at?: string
}

export interface ServiceSpaceBlueprint {
  id: string
  tenant_id: number
  service_id?: string
  source_type?: 'instruction' | 'template'
  source_instruction: string
  proposed_space_type: ServiceSpaceType
  proposed_template_key?: string
  template_version?: number
  subject_policy: {
    required: boolean
    allowed_types?: string[]
    allow_hierarchy?: boolean
  }
  profile_schema: Array<{
    key: string
    label: string
    value_type: string
    source: string
    required: boolean
    sensitive: boolean
    display_order: number
  }>
  summary_schema: Array<{
    key: string
    label: string
    source_scopes?: string[]
    refresh_policy: string
    display_order: number
  }>
  expert_suggestions?: Array<{
    expert_ref: string
    expert_name: string
    reason?: string
  }>
  generation_meta?: Record<string, unknown>
  status: 'draft' | 'confirmed' | 'rejected' | 'expired'
  confirmation_mode?: 'pending' | 'manual' | 'template_auto_apply'
  profile_version?: number
  version: number
}

export interface ServiceSpaceProfile {
  id: string
  service_id: string
  blueprint_version: number
  version: number
  schema: ServiceSpaceBlueprint['profile_schema']
  values: Record<string, unknown>
  source_watermark?: string
  frozen?: boolean
  updated_at?: string
}

export type ServiceSpaceProfileField = ServiceSpaceProfile['schema'][number]

export interface ServiceSpaceSummary {
  id: string
  service_id: string
  blueprint_version: number
  version: number
  schema: ServiceSpaceBlueprint['summary_schema']
  sections: Record<string, unknown>
  source_watermark?: string
  frozen?: boolean
  refresh_status?: string
  error_message?: string
  generated_at?: string
  updated_at?: string
}

export interface ServiceRuntimeContext {
	service_id: string
	space_type: ServiceSpaceType
  instruction: string
  knowledge_base_ids: string[]
  template_key?: string
  template_version?: number
  blueprint_version?: number
  experts?: ServiceExpertBinding[]
  profile?: ServiceSpaceProfile
	summary?: ServiceSpaceSummary
	artifacts?: ServiceArtifact[]
	context_sources?: ServiceContextSource[]
	context_hash: string
}

export interface ServiceContextSource {
  id: string
  tenant_id: number
  service_id: string
  source_type: 'organize_output' | string
  source_id: string
  source_title: string
  source_version?: string
  source_summary?: string
  source_content?: string
  memory_ids?: string[]
  metadata?: Record<string, unknown>
  imported_by?: string
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
  space_type?: ServiceSpaceType
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

export function listServiceTemplates() {
  return get<ServiceResponse<ServiceSpaceTemplate[]>>('/api/v1/services/templates')
}

export function applyServiceTemplate(
  templateKey: string,
  input: {
    name: string
    description?: string
    instruction?: string
    template_key?: string
    template_version?: number
    knowledge_base_ids?: string[]
    experts?: Array<{
      expert_ref: string
      expert_name: string
      expert_domain?: string
      enabled?: boolean
      display_order?: number
    }>
    idempotency_key: string
  },
) {
  return post<ServiceResponse<ServiceSpace>>(
    `/api/v1/services/templates/${encodeURIComponent(templateKey)}/apply`,
    { ...input, template_key: input.template_key || templateKey },
  )
}

export function previewServiceBlueprint(input: { instruction: string }) {
  return post<ServiceResponse<ServiceSpaceBlueprint>>(
    '/api/v1/services/blueprint/preview',
    input,
  )
}

export function previewServiceBlueprintForService(
  serviceId: string,
  input: { instruction: string },
) {
  return post<ServiceResponse<ServiceSpaceBlueprint>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/blueprint/preview`,
    input,
  )
}

export function getServiceBlueprint(serviceId: string) {
  return get<ServiceResponse<ServiceSpaceBlueprint>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/blueprint`,
  )
}

export function confirmServiceBlueprint(
  serviceId: string,
  input: {
    blueprint_id: string
    expected_version: number
    activate?: boolean
    idempotency_key: string
  },
) {
  return post<ServiceResponse<ServiceSpace>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/blueprint/confirm`,
    input,
  )
}

export function getServiceProfile(serviceId: string) {
  return get<ServiceResponse<ServiceSpaceProfile>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/profile`,
  )
}

export function updateServiceProfile(
  serviceId: string,
  values: Record<string, unknown>,
  schema?: ServiceSpaceProfileField[],
) {
  return put<ServiceResponse<ServiceSpaceProfile>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/profile`,
    schema ? { values, schema } : { values },
  )
}

export function getServiceSummary(serviceId: string) {
  return get<ServiceResponse<ServiceSpaceSummary>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/summary`,
  )
}

export function refreshServiceSummary(serviceId: string) {
  return post<ServiceResponse<ServiceSpaceSummary>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/summary/refresh`,
    {},
  )
}

export function getServiceRuntimeContext(serviceId: string) {
	return get<ServiceResponse<ServiceRuntimeContext>>(
		`/api/v1/services/${encodeURIComponent(serviceId)}/context`,
	)
}

export function listServiceContextSources(serviceId: string) {
  return get<ServiceResponse<ServiceContextSource[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/context-sources`,
  )
}

export function importServiceOrganizeOutput(serviceId: string, outputId: string) {
  return post<ServiceResponse<ServiceContextSource>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/context-sources`,
    { source_type: 'organize_output', source_id: outputId },
  )
}

export function deleteServiceContextSource(serviceId: string, sourceId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/context-sources/${encodeURIComponent(sourceId)}`,
  )
}

export function updateServiceArtifactLifecycle(
  serviceId: string,
  artifactId: string,
  input: { lifecycle: ServiceArtifact['lifecycle']; idempotency_key: string },
) {
  return post<ServiceResponse<ServiceArtifact>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/artifacts/${encodeURIComponent(artifactId)}/lifecycle`,
    input,
  )
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

export function listServiceSubjects(
  serviceId: string,
  params?: { page?: number; page_size?: number; subject_type?: string },
) {
  return get<ServiceResponse<ServicePage<ServiceSubject>>>(
    withQuery(`/api/v1/services/${encodeURIComponent(serviceId)}/subjects`, params),
  )
}

export function getServiceSubject(serviceId: string, subjectId: string) {
  return get<ServiceResponse<ServiceSubject>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/subjects/${encodeURIComponent(subjectId)}`,
  )
}

export function createServiceSubject(
  serviceId: string,
  input: {
    subject_type: string
    subject_key: string
    display_name: string
    parent_subject_id?: string
    metadata?: Record<string, unknown>
    visibility_scope?: 'private' | 'tenant'
  },
) {
  return post<ServiceResponse<ServiceSubject>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/subjects`,
    input,
  )
}

export function updateServiceSubject(
  serviceId: string,
  subjectId: string,
  input: Partial<{
    subject_type: string
    subject_key: string
    display_name: string
    parent_subject_id: string | null
    metadata: Record<string, unknown>
    visibility_scope: 'private' | 'tenant'
  }>,
) {
  return put<ServiceResponse<ServiceSubject>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/subjects/${encodeURIComponent(subjectId)}`,
    input,
  )
}

export function deleteServiceSubject(serviceId: string, subjectId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/subjects/${encodeURIComponent(subjectId)}`,
  )
}

export function listServiceReminderStatuses(serviceId: string, includeDisabled = false) {
  return get<ServiceResponse<ServiceReminderStatus[]>>(
    withQuery(`/api/v1/services/${encodeURIComponent(serviceId)}/statuses`, {
      include_disabled: includeDisabled ? 'true' : undefined,
    }),
  )
}

export function createServiceReminderStatus(
  serviceId: string,
  input: {
    status_key: string
    label: string
    category: ServiceReminderStatusCategory
    is_initial?: boolean
    is_terminal?: boolean
    display_order?: number
    color?: string
    description?: string
    enabled?: boolean
  },
) {
  return post<ServiceResponse<ServiceReminderStatus>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/statuses`,
    input,
  )
}

export function updateServiceReminderStatus(
  serviceId: string,
  statusId: string,
  input: Partial<{
    label: string
    is_initial: boolean
    is_terminal: boolean
    display_order: number
    color: string
    description: string
    enabled: boolean
  }>,
) {
  return patch<ServiceResponse<ServiceReminderStatus>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/statuses/${encodeURIComponent(statusId)}`,
    input,
  )
}

export function deleteServiceReminderStatus(serviceId: string, statusId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/statuses/${encodeURIComponent(statusId)}`,
  )
}

export function listServiceReminderStatusTransitions(serviceId: string) {
  return get<ServiceResponse<ServiceReminderStatusTransition[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/statuses/transitions`,
  )
}

export function replaceServiceReminderStatusTransitions(
  serviceId: string,
  transitions: Array<{
    from_status_id: string
    to_status_id: string
    allowed_roles?: string[]
    enabled: boolean
  }>,
) {
  return put<ServiceResponse<ServiceReminderStatusTransition[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/statuses/transitions`,
    { transitions },
  )
}

export function listServiceReminders(
  serviceId: string,
  params?: { page?: number; page_size?: number; status?: string },
) {
  return get<ServiceResponse<ServicePage<ServiceReminder>>>(
    withQuery(`/api/v1/services/${encodeURIComponent(serviceId)}/reminders`, params),
  )
}

export function getServiceReminder(serviceId: string, reminderId: string) {
  return get<ServiceResponse<ServiceReminder>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}`,
  )
}

export function createServiceReminder(
  serviceId: string,
  input: {
    subject_id?: string
    parent_reminder_id?: string
    title: string
    summary?: string
    status?: string
    priority?: 'high' | 'medium' | 'low'
    due_at?: string
    due_text?: string
    next_action?: string
    agent_domain?: string
    assignee_user_ids?: string[]
    metadata?: Record<string, unknown>
  },
) {
  return post<ServiceResponse<ServiceReminder>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders`,
    input,
  )
}

export function updateServiceReminder(
  serviceId: string,
  reminderId: string,
  input: Partial<{
    title: string
    parent_reminder_id: string | null
    summary: string
    status: string
    priority: 'high' | 'medium' | 'low'
    due_at: string | null
    due_text: string
    next_action: string
    assignee_user_ids: string[]
    metadata: Record<string, unknown>
  }>,
) {
  return patch<ServiceResponse<ServiceReminder>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}`,
    input,
  )
}

export function deleteServiceReminder(serviceId: string, reminderId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}`,
  )
}

export function listServiceReminderAssignees(serviceId: string, reminderId: string) {
  return get<ServiceResponse<ServiceReminderAssignee[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/assignees`,
  )
}

export function replaceServiceReminderAssignees(
  serviceId: string,
  reminderId: string,
  userIds: string[],
) {
  return put<ServiceResponse<ServiceReminderAssignee[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/assignees`,
    { user_ids: userIds },
  )
}

export function listServiceReminderComments(serviceId: string, reminderId: string) {
  return get<ServiceResponse<ServiceReminderComment[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/comments`,
  )
}

export function addServiceReminderComment(serviceId: string, reminderId: string, content: string) {
  return post<ServiceResponse<ServiceReminderComment>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/comments`,
    { content },
  )
}

export function deleteServiceReminderComment(serviceId: string, reminderId: string, commentId: string) {
  return del<ServiceResponse<unknown>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/comments/${encodeURIComponent(commentId)}`,
  )
}

export function listServiceReminderHistory(serviceId: string, reminderId: string) {
  return get<ServiceResponse<ServiceReminderHistory[]>>(
    `/api/v1/services/${encodeURIComponent(serviceId)}/reminders/${encodeURIComponent(reminderId)}/history`,
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
