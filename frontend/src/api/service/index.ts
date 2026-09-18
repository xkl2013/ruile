import { get, post, put } from '@/utils/request'
import type { ServiceTask, ServiceTaskSource } from '@/views/service/serviceMemoryExtraction'

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

export interface AgentWorkDoc {
  id: string
  tenant_id?: number
  profile_id?: string
  subject_id?: string
  owner_user_id?: string
  agent_domain?: string
  doc_type: string
  doc_path: string
  title: string
  content?: string
  status: string
  source_memory_ids?: string[]
  metadata?: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export interface ServiceMemoryEvidenceDTO {
  id: string
  title: string
  summary: string
  sourceLabel?: string
  occurredAtLabel?: string
}

export interface ServiceReminderDTO {
  id: string
  subject_id?: string
  profile_id?: string
  agent_domain?: string
  title: string
  summary: string
  status: string
  priority: 'high' | 'medium' | 'low'
  due_at?: string
  due_text?: string
  stage?: string
  service_mode?: string
  subject_name?: string
  channel?: string
  decision_role?: string
  risk_label?: string
  assist_reason?: string
  primary_action?: string
  next_action?: string
  avoid_action?: string
  context_items?: string[]
  memory_signals?: string[]
  source_memory_ids?: string[]
  source_memory_count?: number
  last_memory_at?: string
  confidence?: number
  sales_highlights?: string[]
  write_back_status?: string
  write_back_draft?: string
  reply_draft?: string
  metadata?: Record<string, unknown>
  memory_evidence?: ServiceMemoryEvidenceDTO[]
  work_docs?: AgentWorkDoc[]
  created_at?: string
  updated_at?: string
}

export interface AgentActionDraft {
  id: string
  reminder_id: string
  agent_id: string
  agent_domain: string
  action_type: string
  status: string
  title: string
  summary: string
  payload?: Record<string, unknown>
  source_memory_ids?: string[]
  external_system?: string
  external_object_id?: string
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

export interface ServiceBootstrapData {
  profile?: ServiceWorkProfile
  agent_settings?: WorkProfileAgentSetting[]
  reminders: ServiceReminderDTO[]
  total: number
  stats: Record<string, number>
  templates?: ServiceAgentTemplate[]
}

export interface ServiceMemoryExtraction {
  memory_id: string
  generated: boolean
  reason?: string
  reminder?: ServiceReminderDTO
}

export type ServiceAgentRunStatus = 'queued' | 'running' | 'waiting_input' | 'succeeded' | 'failed' | 'cancelled'

export type ServiceAgentRunPhase = 'intake' | 'planning' | 'drafting' | 'reviewing' | 'revising' | 'packaging' | 'completed'

export interface ExpertIntakeQuestion {
  id: string
  label: string
  type: 'text' | 'single_choice' | 'multi_choice' | 'date' | 'number'
  required: boolean
  options?: string[]
  description?: string
}

export interface ExpertIntakeInteraction {
  schema_version: 'intake_request_v1'
  questions: ExpertIntakeQuestion[]
}

export interface ServiceAgentRunQualityIssue {
  code: string
  severity: 'warning' | 'error' | 'red_line'
  section?: string
  message: string
  instruction: string
}

export interface ServiceAgentRunQuality {
  score?: number
  passed?: boolean
  summary?: string
  red_lines?: string[]
  dimensions?: Record<string, number>
  issues?: ServiceAgentRunQualityIssue[]
}

export interface ServiceAgentRunStep {
  id: string
  run_id: string
  sequence: number
  step_type: ServiceAgentRunPhase
  status: 'running' | 'succeeded' | 'failed'
  model_id?: string
  input?: Record<string, unknown>
  output?: Record<string, unknown>
  error?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
}

export interface ServiceCardV1 {
  schema_version: 'service_card_v1'
  title: string
  summary: string
  next_action: string
}

export interface StructuredReportSectionV1 {
  type: 'facts' | 'analysis' | 'risks' | 'missing_information' | 'recommended_actions' | 'talk_track' | 'evidence'
  title: string
  content?: string
  items?: string[]
}

export interface StructuredReportV1 {
  format: 'structured_report_v1'
  title: string
  executive_summary: string
  sections: StructuredReportSectionV1[]
  evidence_refs: string[]
}

export interface ServiceAgentArtifactResultV1 {
  kind: 'text' | 'report' | 'html' | 'image' | 'pdf' | 'document' | 'spreadsheet' | 'presentation' | 'audio' | 'video' | 'data'
  role: 'primary' | 'supporting'
  title: string
  format?: string
  content?: StructuredReportV1 | Record<string, unknown>
}

export interface ServiceAgentEvidenceRefV1 {
  source_type: string
  source_id: string
  relation: string
  excerpt?: string
}

export interface ServiceAgentResultValidation {
  contract: 'agent_result_v1'
  valid: boolean
  errors: string[]
}

export interface ServiceAgentRunResult extends Record<string, unknown> {
  schema_version?: 'agent_result_v1'
  decision?: {
    should_create_card: boolean
    confidence: number
    reason: string
  }
  card?: ServiceCardV1
  artifacts?: ServiceAgentArtifactResultV1[]
  evidence?: ServiceAgentEvidenceRefV1[]
  validation?: ServiceAgentResultValidation
  artifact_type?: string
  daily_report_id?: string
  memory_id?: string
  generated?: boolean
  reason?: string
  service_reminder_id?: string
}

export interface ServiceAgentRun {
  id: string
  tenant_id: number
  user_id: string
  profile_id?: string
  parent_run_id?: string
  requirement_snapshot_id?: string
  run_type: 'service_daily_report' | 'service_memory_extract' | 'expert_agent_test'
  agent_ref?: string
  agent_version?: string
  trigger_type?: string
  trigger_id?: string
  status: ServiceAgentRunStatus
  phase?: ServiceAgentRunPhase
  input?: Record<string, unknown>
  interaction?: ExpertIntakeInteraction | Record<string, unknown>
  quality?: ServiceAgentRunQuality
  result?: ServiceAgentRunResult
  error_code?: string
  error_message?: string
  task_id?: string
  attempt?: number
  queued_at?: string
  resumed_at?: string
  started_at?: string
  finished_at?: string
  created_at?: string
  updated_at?: string
}

export type ServiceDailyReportRange = 'day' | 'week' | 'month'

export interface ServiceDailyReportDTO {
  id: string
  title: string
  summary?: string
  content: string
  structured_report?: StructuredReportV1
  range: ServiceDailyReportRange
  stage: string
  stage_key?: string
  updated?: string
  action_count?: number
  customer_count?: number
  subject_count?: number
  open_action_count?: number
  closed_action_count?: number
  high_risk_count?: number
  knowledge_gap_count?: number
  source_memory_count?: number
  daily_source_count?: number
  evidence_complete_rate?: number
  profile_id?: string
  profile_name?: string
  memory_scope?: string
  can_supplement?: boolean
  chips?: string[]
  source_memory_ids?: string[]
  metadata?: Record<string, unknown>
  created_at?: string
  updated_at?: string
}

export interface ServiceSubjectDTO {
  id: string
  tenant_id: number
  owner_user_id: string
  subject_key: string
  display_name: string
  student_name?: string
  relation?: string
  aliases?: string[]
  external_refs?: Record<string, unknown>
  visibility_scope?: string
  confidence?: number
  created_at?: string
  updated_at?: string
}

export interface ServiceCustomerSpaceDTO {
  id: string
  tenant_id: number
  owner_user_id: string
  profile_id?: string
  subject_key: string
  display_name: string
  name: string
  student_name?: string
  relation?: string
  description?: string
  summary?: string
  status: string
  priority?: 'high' | 'medium' | 'low' | ''
  stage?: string
  risk_label?: string
  latest_action?: string
  visibility_scope?: string
  confidence?: number
  work_doc_count: number
  reminder_count: number
  open_reminder_count: number
  source_memory_count: number
  directories?: string[]
  chips?: string[]
  latest_memory_at?: string
  latest_reminder_at?: string
  created_at?: string
  updated_at?: string
}

export interface ServiceCustomerSpaceDetailDTO {
  summary: ServiceCustomerSpaceDTO
  subject: ServiceSubjectDTO
  work_docs: AgentWorkDoc[]
  reminders: ServiceReminderDTO[]
  memory_evidence: ServiceMemoryEvidenceDTO[]
  directories?: string[]
  stats: Record<string, number>
}

export interface ServiceResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface ServiceListData<T> {
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

const asString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const asStringList = (value: unknown) => Array.isArray(value)
  ? value.map((item) => asString(item)).filter(Boolean)
  : []

const formatConfidence = (confidence?: number) => {
  if (typeof confidence !== 'number' || Number.isNaN(confidence)) return '待确认'
  if (confidence >= 0.8) return '较高'
  if (confidence >= 0.6) return '待确认'
  return '低'
}

const formatMemoryDateLabel = (value?: string) => {
  const raw = asString(value)
  if (!raw) return '最近'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const taskSource: ServiceTaskSource = 'memory'

export function mapServiceReminderToTask(item: ServiceReminderDTO): ServiceTask {
  const metadata = item.metadata || {}
  const subjectName = asString(metadata.subject_name) || asString(item.subject_name) || item.title || ''
  const serviceMode = asString(metadata.service_mode) || asString(item.service_mode) || item.stage || ''
  const rawCustomerName = asString(metadata.customer_name)
  const hasCustomerIdentity = Boolean(rawCustomerName)
  const customerName = rawCustomerName || subjectName || item.title || '服务事项'
  const studentName = asString(metadata.student_name) || (hasCustomerIdentity ? '待补充' : '')
  return {
    id: item.id,
    subjectId: item.subject_id,
    sourceType: taskSource,
    hasCustomerIdentity,
    subjectName,
    serviceMode,
    customerName,
    studentName,
    title: item.title || customerName,
    summary: item.summary || '',
    stage: item.stage || serviceMode || (hasCustomerIdentity ? '客户跟进' : '服务事项'),
    priorityKey: item.priority || 'low',
    dueText: item.due_text || formatMemoryDateLabel(item.due_at),
    channel: item.channel || '记忆',
    decisionRole: item.decision_role || '决策人待补充',
    riskLabel: item.risk_label || '待判断',
    assistReason: item.assist_reason || item.summary || '',
    primaryAction: item.primary_action || '',
    nextAction: item.next_action || (hasCustomerIdentity ? '确认客户状态并补一条下一步记忆' : '确认服务事项并补充下一步'),
    avoidAction: item.avoid_action || '',
    contextItems: asStringList(item.context_items),
    memorySignals: asStringList(item.memory_signals),
    memoryEvidence: (item.memory_evidence || []).map((memory) => ({
      id: memory.id,
      title: memory.title,
      summary: memory.summary,
      sourceLabel: memory.sourceLabel || '个人记忆',
      occurredAtLabel: memory.occurredAtLabel || '最近',
    })),
    sourceMemoryIds: asStringList(item.source_memory_ids),
    sourceMemoryCount: item.source_memory_count || item.source_memory_ids?.length || 0,
    lastMemoryLabel: formatMemoryDateLabel(item.last_memory_at || item.updated_at),
    confidenceLabel: formatConfidence(item.confidence),
    salesHighlights: asStringList(item.sales_highlights),
    writeBackStatus: item.write_back_status || '待确认',
    writeBackDraft: item.write_back_draft || '',
    replyDraft: item.reply_draft || '',
  }
}

export function getServiceBootstrap() {
  return get<ServiceResponse<ServiceBootstrapData>>('/api/v1/service/bootstrap')
}

export function refreshServiceModule() {
  return post<ServiceResponse<ServiceBootstrapData>>('/api/v1/service/refresh')
}

export function extractServiceMemory(memoryId: string) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/memories/${encodeURIComponent(memoryId)}/extract`,
  )
}

export function getServiceAgentRun(id: string) {
  return get<ServiceResponse<ServiceAgentRun>>(`/api/v1/service/agent-runs/${encodeURIComponent(id)}`)
}

export function listServiceAgentRunSteps(id: string) {
  return get<ServiceResponse<ServiceAgentRunStep[]>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/steps`,
  )
}

export function getServiceAgentRunQuality(id: string) {
  return get<ServiceResponse<ServiceAgentRunQuality>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/quality`,
  )
}

export function submitServiceAgentRunAnswers(id: string, answers: Record<string, unknown>) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/answers`,
    { answers },
  )
}

export function regenerateServiceAgentRun(id: string, feedback?: string) {
  return post<ServiceResponse<ServiceAgentRun>>(
    `/api/v1/service/agent-runs/${encodeURIComponent(id)}/regenerate`,
    feedback?.trim() ? { feedback: feedback.trim() } : {},
  )
}

export async function waitForServiceAgentRun(
  id: string,
  options?: { intervalMs?: number; timeoutMs?: number },
) {
  const intervalMs = options?.intervalMs ?? 1200
  const timeoutMs = options?.timeoutMs ?? 5 * 60 * 1000
  const deadline = Date.now() + timeoutMs

  while (true) {
    const response = await getServiceAgentRun(id)
    const run = response?.data
    if (!response?.success || !run) {
      throw new Error(response?.message || '任务状态读取失败')
    }
    if (run.status === 'succeeded' || run.status === 'waiting_input') return run
    if (run.status === 'failed' || run.status === 'cancelled') {
      throw new Error(run.error_message || '任务未完成')
    }
    if (Date.now() >= deadline) {
      throw new Error('任务仍在执行，请稍后在服务空间查看结果')
    }
    await new Promise<void>((resolve) => window.setTimeout(resolve, intervalMs))
  }
}

export function listServiceDailyReports(params?: {
  range?: ServiceDailyReportRange
  keyword?: string
  profile_id?: string
  page?: number
  page_size?: number
}) {
  return get<ServiceResponse<ServiceListData<ServiceDailyReportDTO>>>(withQuery('/api/v1/service/daily-reports', params))
}

export function getServiceDailyReport(id: string) {
  return get<ServiceResponse<ServiceDailyReportDTO>>(`/api/v1/service/daily-reports/${encodeURIComponent(id)}`)
}

export function generateServiceDailyReport(data?: {
  range?: ServiceDailyReportRange
  date?: string
  timezone?: string
  trigger?: string
}) {
  return post<ServiceResponse<ServiceAgentRun>>('/api/v1/service/daily-reports', data || {})
}

export function listServiceCustomerSpaces(params?: {
  keyword?: string
  profile_id?: string
  page?: number
  page_size?: number
}) {
  return get<ServiceResponse<ServiceListData<ServiceCustomerSpaceDTO>>>(withQuery('/api/v1/service/customer-spaces', params))
}

export function getServiceCustomerSpace(id: string, params?: { profile_id?: string }) {
  return get<ServiceResponse<ServiceCustomerSpaceDetailDTO>>(
    withQuery(`/api/v1/service/customer-spaces/${encodeURIComponent(id)}`, params),
  )
}

export function listServiceReminders(params?: {
  keyword?: string
  status?: string
  agent_domain?: string
  profile_id?: string
  memory_id?: string
  page?: number
  page_size?: number
}) {
  return get<ServiceResponse<ServiceListData<ServiceReminderDTO>>>(withQuery('/api/v1/service/reminders', params))
}

export function updateServiceReminderStatus(id: string, status: string) {
  return put<ServiceResponse<ServiceReminderDTO>>(`/api/v1/service/reminders/${encodeURIComponent(id)}/status`, { status })
}

export function createServiceActionDraft(id: string, data?: Partial<AgentActionDraft>) {
  return post<ServiceResponse<AgentActionDraft>>(`/api/v1/service/reminders/${encodeURIComponent(id)}/action-drafts`, data || {})
}

export function listServiceAgentTemplates() {
  return get<ServiceResponse<ServiceAgentTemplate[]>>('/api/v1/service/agent-templates')
}

export function listServiceWorkProfiles(params?: { user_id?: string }) {
  return get<ServiceResponse<ServiceWorkProfile[]>>(withQuery('/api/v1/service/work-profiles', params))
}

export function listServiceAgentSettings(id: string, params?: { enabled?: boolean }) {
  return get<ServiceResponse<WorkProfileAgentSetting[]>>(
    withQuery(`/api/v1/service/work-profiles/${encodeURIComponent(id)}/agent-settings`, params),
  )
}

export function createServiceWorkProfile(data: Partial<ServiceWorkProfile>) {
  return post<ServiceResponse<ServiceWorkProfile>>('/api/v1/service/work-profiles', data)
}

export function updateServiceWorkProfile(id: string, data: Partial<ServiceWorkProfile>) {
  return put<ServiceResponse<ServiceWorkProfile>>(`/api/v1/service/work-profiles/${encodeURIComponent(id)}`, data)
}

export function replaceServiceAgentSettings(id: string, settings: Partial<WorkProfileAgentSetting>[]) {
  return put<ServiceResponse<WorkProfileAgentSetting[]>>(`/api/v1/service/work-profiles/${encodeURIComponent(id)}/agent-settings`, { settings })
}
