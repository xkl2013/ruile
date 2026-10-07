import { fetchEventSource } from '@microsoft/fetch-event-source'
import { del, get, getDown, post, postUpload, put } from '@/utils/request'
import { generateRandomString } from '@/utils/index'
import { getApiBaseUrl } from '@/utils/api-base'

export type OrganizeMemoryKind = 'note' | 'record' | 'audio' | 'audio_card'
export type OrganizeOutputStatus = 'draft' | 'review' | 'ready' | 'archived'
export type OrganizeAssignmentStatus = 'pending' | 'assigned'
export type OrganizePublicContentType = 'post' | 'course'
export type OrganizePublicContentStatus = 'draft' | 'pending_review' | 'published' | 'offline' | 'rejected'
export type OrganizeSproutStage = 'organizing' | 'expandable' | 'formed'
export type OrganizeScheduleKey = 'manual' | 'daily' | 'weekly' | 'monthly'
export type OrganizeJobStatus =
  | 'queued'
  | 'running'
  | 'repairing'
  | 'completed'
  | 'fallback'
  | 'failed'
  | 'canceled'
export type OrganizeJobMode = 'single' | 'batch'

export type OrganizeMemoryAttachmentStatus = 'pending' | 'processing' | 'completed' | 'failed' | 'skipped'
export type OrganizeMemoryAttachmentAggregateStatus = 'pending' | 'processing' | 'partial' | 'completed' | 'failed'

export interface OrganizeTemplate {
  id: string
  key: string
  name: string
  scene: string
  description: string
  output_label: string
  icon: string
  default_instruction: string
  markdown_template: string
  expert_ids: string[]
  spec?: Record<string, unknown>
  status: 'draft' | 'enabled' | 'disabled'
  published_version: string
  sort_order: number
  created_at: string
  updated_at: string
}

export interface OrganizeExpert {
  id: string
  name: string
  description: string
}

export interface OrganizeJob {
  id: string
  config_id: string
  template_key: string
  template_version: string
  target_service_id?: string
  status: OrganizeJobStatus
  stage: string
  progress: number
  requirement?: Record<string, unknown>
  memory_ids?: string[]
  model_id?: string
  output_id?: string
  summary?: string
  result?: Record<string, unknown>
  error_message?: string
  job_mode?: OrganizeJobMode
  selection_snapshot?: Record<string, unknown>
  input_fingerprint?: string
  selected_count?: number
  ready_count?: number
  processed_count?: number
  failed_count?: number
  overlap_count?: number
  batch_count?: number
  coverage?: Record<string, unknown>
  batches?: OrganizeJobBatch[]
  scheduled_for?: string
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export interface OrganizeJobBatch {
  id: string
  parent_job_id: string
  batch_index: number
  batch_count: number
  status: 'queued' | 'running' | 'completed' | 'fallback' | 'failed' | 'canceled'
  stage: string
  progress: number
  memory_ids?: string[]
  input_chars: number
  summary?: string
  error_message?: string
  retry_count: number
  started_at?: string
  finished_at?: string
  created_at: string
  updated_at: string
}

export interface OrganizeConfig {
  id: string
  name: string
  template_key: string
  target_service_id?: string
  instruction: string
  expert_ids: string[]
  schedule: OrganizeScheduleKey
  status: 'active' | 'disabled'
  next_run_at?: string
  last_run_at?: string
  metadata?: Record<string, unknown>
  template?: OrganizeTemplate
  latest_job?: OrganizeJob
  job_count?: number
  has_unread_output?: boolean
  created_at: string
  updated_at: string
}

export interface OrganizeMemory {
  id: string
  kind: OrganizeMemoryKind
  title: string
  content: string
  source?: string
  occurred_at: string
  duration_seconds?: number
  metadata?: Record<string, unknown>
  attachments?: OrganizeMemoryAttachment[]
  created_at: string
  updated_at: string
}

export interface OrganizeMemoryAttachment {
  id: string
  memory_id: string
  file_name: string
  mime_type?: string
  storage_path?: string
  storage_url?: string
  size_bytes?: number
  sort_order: number
  status: OrganizeMemoryAttachmentStatus
  error_stage?: string
  error_message?: string
  content?: string
  transcript?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface OrganizeMemoryReference {
  id: string
  kind?: OrganizeMemoryKind | string
  title: string
  source?: string
}

export interface OrganizeOutput {
  id: string
  tenant_id?: number | string
  user_id?: string
  config_id?: string
  job_id?: string
  assigned_service_id?: string
  assignment_status?: OrganizeAssignmentStatus
  assignment_reason?: string
  template_key?: string
  template_version?: string
  title: string
  output_type: string
  content: string
  source_summary?: string
  status: OrganizeOutputStatus
  public_content_type?: OrganizePublicContentType
  public_status?: OrganizePublicContentStatus
  series_id?: string
  series_title?: string
  series_order?: number
  review_note?: string
  published_at?: string
  published_by?: string
  icon?: string
  creator_name?: string
  creator_avatar?: string
  is_subscribed?: boolean
  memory_count?: number
  memory_ids?: string[]
  featured?: boolean
  recommendable?: boolean
  sort_order?: number
  fields?: Record<string, unknown>
  citations?: Record<string, unknown>
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface OrganizeOutputFacetValue {
  value: string
  count: number
}

export interface OrganizeOutputFacet {
  key: string
  label: string
  values: OrganizeOutputFacetValue[]
}

export interface OrganizeOutputFacets {
  fields: OrganizeOutputFacet[]
}

export interface OrganizeSproutReport {
  id: string
  user_id?: string
  title: string
  template_key?: string
  template_version?: string
  summary: string
  stage: OrganizeSproutStage
  output_hint?: string
  chips?: string[]
  memory_count?: number
  memory_ids?: string[]
  memory_refs?: OrganizeMemoryReference[]
  fields?: Record<string, unknown>
  creator_name?: string
  creator_avatar?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface OrganizeListData<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface OrganizeDiscoverTab {
  label: string
  value: string
  count?: number
}

export interface OrganizeDiscoverCategory {
  id: string
  key: string
  label: string
  description?: string
  sort_order: number
  status: 'enabled' | 'disabled'
  created_at: string
  updated_at: string
}

export interface OrganizeDiscoverData {
  tabs: OrganizeDiscoverTab[]
  featured_outputs: OrganizeOutput[]
  items: OrganizeOutput[]
  total: number
  page?: number
  page_size?: number
  featured_offset?: number
}

export interface OrganizeRequirementInput {
  config_id: string
  template_key?: string
  text: string
  memory_ids?: string[]
  model_id?: string
  allow_partial?: boolean
  confirmed?: boolean
}

export interface OrganizeMemoryQueryPlan {
  time_field?: string
  occurred_from?: string
  occurred_to?: string
  kinds?: string[]
  keyword?: string
  sources?: string[]
  ready_only?: boolean
  timezone?: string
}

export interface OrganizeRequirementPreview {
  config_id: string
  template_key: string
  template_name: string
  scene: string
  normalized_text: string
  query_plan: OrganizeMemoryQueryPlan
  memory_ids?: string[]
  selected_count: number
  ready_count: number
  unready_count: number
  overlap_count: number
  estimated_batch_count: number
  sample_memories?: OrganizeMemoryReference[]
  ambiguities?: string[]
  suggestions?: string[]
  warnings?: string[]
  need_confirmation: boolean
}

export interface OrganizeResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface OrganizeListParams {
  keyword?: string
  page?: number
  page_size?: number
}

export type OrganizeCourseSource = 'official' | 'creator'
export type OrganizeCourseLessonType = 'video' | 'audio' | 'article'

export interface OrganizeCourseLesson {
  id: string
  course_id: string
  title: string
  lesson_type: OrganizeCourseLessonType
  duration_seconds: number
  sort_order: number
  created_at: string
  updated_at: string
  available: boolean
  content?: string
  media_url?: string
  source_file_name?: string
}

export interface OrganizeCourseLessonMediaURL {
  url: string
  file_name?: string
  mime_type?: string
}

export interface OrganizeCourse {
  id: string
  source: OrganizeCourseSource
  title: string
  summary: string
  category: string
  cover_url: string
  teacher_name: string
  teacher_title: string
  lesson_count: number
  learner_count: number
  featured: boolean
  recommendable: boolean
  sort_order: number
  created_at: string
  updated_at: string
  lessons?: OrganizeCourseLesson[]
}

export interface OrganizeMemoryInput {
  kind?: OrganizeMemoryKind
  title: string
  content?: string
  source?: string
  occurred_at?: string
  duration_seconds?: number
  metadata?: Record<string, unknown>
}

export interface OrganizeOutputInput {
  title: string
  config_id?: string
  job_id?: string
  template_key?: string
  template_version?: string
  output_type?: string
  content?: string
  source_summary?: string
  status?: OrganizeOutputStatus
  public_content_type?: OrganizePublicContentType
  public_status?: OrganizePublicContentStatus
  series_id?: string
  series_title?: string
  series_order?: number
  review_note?: string
  icon?: string
  memory_ids?: string[]
  fields?: Record<string, unknown>
  citations?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface OrganizeSproutReportInput {
  title: string
  template_key?: string
  template_version?: string
  summary?: string
  stage?: OrganizeSproutStage
  output_hint?: string
  chips?: string[]
  memory_ids?: string[]
  fields?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface OrganizeConfigInput {
  name: string
  template_key: string
  target_service_id?: string
  instruction?: string
  expert_ids?: string[]
  schedule?: OrganizeScheduleKey
  metadata?: Record<string, unknown>
}

export interface OrganizeJobInput {
  config_id?: string
  memory_ids?: string[]
  model_id?: string
  requirement?: string
  allow_partial?: boolean
  batch_policy?: string
  force_rerun?: boolean
  selection_snapshot?: Record<string, unknown>
}

export interface OrganizeSproutFromMemoryInput {
  memory_id: string
  model_id?: string
  role_config?: Record<string, unknown>
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

export function listOrganizeMemories(params?: OrganizeListParams & { kind?: OrganizeMemoryKind }) {
  return get<OrganizeResponse<OrganizeListData<OrganizeMemory>>>(withQuery('/api/v1/organize/memories', params))
}

export function listOrganizeTemplates() {
  return get<OrganizeResponse<OrganizeTemplate[]>>('/api/v1/organize/templates')
}

export function listOrganizeTemplateScenes() {
  return get<OrganizeResponse<string[]>>('/api/v1/organize/templates/scenes')
}

export function getOrganizeTemplate(key: string) {
  return get<OrganizeResponse<OrganizeTemplate>>(`/api/v1/organize/templates/${encodeURIComponent(key)}`)
}

export function listOrganizeExperts() {
  return get<OrganizeResponse<OrganizeExpert[]>>('/api/v1/organize/experts')
}

export function listOrganizeConfigs(params?: OrganizeListParams & { status?: 'active' | 'disabled' }) {
  return get<OrganizeResponse<OrganizeListData<OrganizeConfig>>>(withQuery('/api/v1/organize/configs', params))
}

export function createOrganizeConfig(input: OrganizeConfigInput) {
  return post<OrganizeResponse<OrganizeConfig>>('/api/v1/organize/configs', input)
}

export function getOrganizeConfig(id: string) {
  return get<OrganizeResponse<OrganizeConfig>>(`/api/v1/organize/configs/${encodeURIComponent(id)}`)
}

export function updateOrganizeConfig(id: string, input: OrganizeConfigInput) {
  return put<OrganizeResponse<OrganizeConfig>>(`/api/v1/organize/configs/${encodeURIComponent(id)}`, input)
}

export function markOrganizeConfigOutputRead(id: string, outputId: string) {
  return post<OrganizeResponse<OrganizeConfig>>(
    `/api/v1/organize/configs/${encodeURIComponent(id)}/read-output`,
    { output_id: outputId },
  )
}

export function deleteOrganizeConfig(id: string) {
  return del<OrganizeResponse<null>>(`/api/v1/organize/configs/${encodeURIComponent(id)}`)
}

export function runOrganizeConfig(id: string, input: OrganizeJobInput = {}) {
  return post<OrganizeResponse<OrganizeJob>>(`/api/v1/organize/configs/${encodeURIComponent(id)}/run`, input)
}

export function listOrganizeConfigJobs(
  id: string,
  params?: OrganizeListParams & { status?: OrganizeJobStatus },
) {
  return get<OrganizeResponse<OrganizeListData<OrganizeJob>>>(
    withQuery(`/api/v1/organize/configs/${encodeURIComponent(id)}/jobs`, params),
  )
}

export function listOrganizeJobs(
  params?: OrganizeListParams & { config_id?: string; status?: OrganizeJobStatus },
) {
  return get<OrganizeResponse<OrganizeListData<OrganizeJob>>>(withQuery('/api/v1/organize/jobs', params))
}

export function createOrganizeJob(input: OrganizeJobInput) {
  return post<OrganizeResponse<OrganizeJob>>('/api/v1/organize/jobs', input)
}

export function getOrganizeJob(id: string) {
  return get<OrganizeResponse<OrganizeJob>>(`/api/v1/organize/jobs/${encodeURIComponent(id)}`)
}

export function retryOrganizeJob(id: string) {
  return post<OrganizeResponse<OrganizeJob>>(`/api/v1/organize/jobs/${encodeURIComponent(id)}/retry`)
}

export function cancelOrganizeJob(id: string) {
  return post<OrganizeResponse<OrganizeJob>>(`/api/v1/organize/jobs/${encodeURIComponent(id)}/cancel`)
}

export async function streamOrganizeJobEvents(
  id: string,
  options: {
    signal?: AbortSignal
    onJob: (job: OrganizeJob) => void
  },
) {
  const token = localStorage.getItem('weknora_token')
  if (!token) throw new Error('登录状态已失效，请重新登录')
  const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id')
  const url = `${getApiBaseUrl()}/api/v1/organize/jobs/${encodeURIComponent(id)}/events`

  await fetchEventSource(url, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'text/event-stream',
      'Accept-Language': localStorage.getItem('locale') || 'zh-CN',
      'X-Request-ID': generateRandomString(12),
      ...(selectedTenantId ? { 'X-Tenant-ID': selectedTenantId } : {}),
    },
    signal: options.signal,
    openWhenHidden: true,
    onopen: async (response) => {
      if (!response.ok) {
        throw new Error(`任务事件流连接失败（HTTP ${response.status}）`)
      }
    },
    onmessage: (message) => {
      if (!message.data) return
      options.onJob(JSON.parse(message.data) as OrganizeJob)
    },
    onclose: () => undefined,
    onerror: (error) => {
      throw error instanceof Error ? error : new Error('任务事件流连接失败')
    },
  })
}

export function createOrganizeMemory(input: OrganizeMemoryInput) {
  return post<OrganizeResponse<OrganizeMemory>>('/api/v1/organize/memories', input)
}

export function uploadOrganizeMemory(file: File | File[]) {
  const formData = new FormData()
  const files = Array.isArray(file) ? file : [file]
  files.forEach((item) => formData.append(files.length > 1 ? 'files' : 'file', item))
  return postUpload('/api/v1/organize/memories/upload', formData, undefined, { timeout: 300000 }) as Promise<OrganizeResponse<OrganizeMemory>>
}

export function updateOrganizeMemory(id: string, input: OrganizeMemoryInput) {
  return put<OrganizeResponse<OrganizeMemory>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`, input)
}

export function getOrganizeMemory(id: string) {
  return get<OrganizeResponse<OrganizeMemory>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`)
}

export function retryOrganizeMemoryAttachment(memoryID: string, attachmentID: string) {
  return post<OrganizeResponse<OrganizeMemory>>(
    `/api/v1/organize/memories/${encodeURIComponent(memoryID)}/attachments/${encodeURIComponent(attachmentID)}/retry`,
  )
}

export function deleteOrganizeMemory(id: string) {
  return del<OrganizeResponse<null>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`)
}

export function listOrganizeOutputs(params?: OrganizeListParams & {
  status?: OrganizeOutputStatus
  config_id?: string
  template_key?: string
  scene?: string
  field_filters?: Record<string, string>
  sort_by?: string
  sort_order?: string
}) {
  const query = new URLSearchParams()
  Object.entries(params || {}).forEach(([key, value]) => {
    if (key === 'field_filters' || value === undefined || value === null || value === '') return
    query.set(key, String(value))
  })
  Object.entries(params?.field_filters || {}).forEach(([key, value]) => {
    if (key && value) query.set(`field.${key}`, value)
  })
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return get<OrganizeResponse<OrganizeListData<OrganizeOutput>>>(`/api/v1/organize/outputs${suffix}`)
}

export function listOrganizeOutputFacets(params?: {
  keyword?: string
  status?: OrganizeOutputStatus
  template_key?: string
  scene?: string
}) {
  return get<OrganizeResponse<OrganizeOutputFacets>>(withQuery('/api/v1/organize/outputs/facets', params))
}

export function listOrganizePendingAssignments(params?: OrganizeListParams) {
  return get<OrganizeResponse<OrganizeListData<OrganizeOutput>>>(
    withQuery('/api/v1/organize/assignments/pending', params),
  )
}

export function assignOrganizeOutputToService(outputId: string, serviceId: string) {
  return post<OrganizeResponse<OrganizeOutput>>(
    `/api/v1/organize/outputs/${encodeURIComponent(outputId)}/assign`,
    { service_id: serviceId },
  )
}

export function getOrganizeDiscover(params?: OrganizeListParams & { tab?: string; featured_offset?: number }) {
  return get<OrganizeResponse<OrganizeDiscoverData>>(withQuery('/api/v1/organize/discover', params))
}

export function listOrganizeDiscoverCategories() {
  return get<OrganizeResponse<OrganizeDiscoverCategory[]>>('/api/v1/organize/discover/categories')
}

export function previewOrganizeRequirement(input: OrganizeRequirementInput) {
  return post<OrganizeResponse<OrganizeRequirementPreview>>('/api/v1/organize/requirements/preview', input)
}

export function confirmOrganizeRequirement(input: OrganizeRequirementInput) {
  return post<OrganizeResponse<OrganizeJob>>('/api/v1/organize/requirements/confirm', input)
}

/**
 * Loads the course cards shown inside 推荐. The pool is cross-tenant and
 * published-only, so no workspace scope is sent from the client.
 */
export function getOrganizeCourses(params?: OrganizeListParams & {
  category?: string
  source?: OrganizeCourseSource
  featured?: boolean
  recommendable?: boolean
}) {
  return get<OrganizeResponse<OrganizeListData<OrganizeCourse>>>(withQuery('/api/v1/organize/courses', params))
}

export function getOrganizeCourse(id: string, config?: any) {
  return get<OrganizeResponse<OrganizeCourse>>(`/api/v1/organize/courses/${encodeURIComponent(id)}`, config)
}

export function getOrganizeCourseCover(url: string) {
  return getDown(url)
}

// The outline deliberately carries no lesson bodies, so each chapter is fetched
// on demand; see getOrganizeCourse.
export function getOrganizeCourseLessonContent(courseId: string, lessonId: string, config?: any) {
  return get<OrganizeResponse<{ content: string }>>(
    `/api/v1/organize/courses/${encodeURIComponent(courseId)}/lessons/${encodeURIComponent(lessonId)}/content`,
    config,
  )
}

export function getOrganizeCourseLessonMediaURL(courseId: string, lessonId: string, config?: any) {
  return get<OrganizeResponse<OrganizeCourseLessonMediaURL>>(
    `/api/v1/organize/courses/${encodeURIComponent(courseId)}/lessons/${encodeURIComponent(lessonId)}/media-url`,
    config,
  )
}

export function createOrganizeOutput(input: OrganizeOutputInput) {
  return post<OrganizeResponse<OrganizeOutput>>('/api/v1/organize/outputs', input)
}

export function uploadOrganizeOutput(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return postUpload('/api/v1/organize/outputs/upload', formData)
}

export function updateOrganizeOutput(id: string, input: OrganizeOutputInput) {
  return put<OrganizeResponse<OrganizeOutput>>(`/api/v1/organize/outputs/${encodeURIComponent(id)}`, input)
}

export function getOrganizeOutput(id: string) {
  return get<OrganizeResponse<OrganizeOutput>>(`/api/v1/organize/outputs/${encodeURIComponent(id)}`)
}

export function getOrganizeOutputCitation(id: string, ref: string) {
  return get<OrganizeResponse<{ memory: OrganizeMemory | null; missing: boolean }>>(
    withQuery(`/api/v1/organize/outputs/${encodeURIComponent(id)}/citation`, { ref }),
  )
}

export function deleteOrganizeOutput(id: string) {
  return del<OrganizeResponse<null>>(`/api/v1/organize/outputs/${encodeURIComponent(id)}`)
}

export function listOrganizeSproutReports(params?: OrganizeListParams & { stage?: OrganizeSproutStage; memory_id?: string }) {
  return get<OrganizeResponse<OrganizeListData<OrganizeSproutReport>>>(withQuery('/api/v1/organize/sprout-reports', params))
}

export function createOrganizeSproutReport(input: OrganizeSproutReportInput) {
  return post<OrganizeResponse<OrganizeSproutReport>>('/api/v1/organize/sprout-reports', input)
}

export function createOrganizeSproutReportFromMemory(input: OrganizeSproutFromMemoryInput) {
  return post<OrganizeResponse<OrganizeSproutReport>>('/api/v1/organize/sprout-reports/from-memory', input)
}

export function updateOrganizeSproutReport(id: string, input: OrganizeSproutReportInput) {
  return put<OrganizeResponse<OrganizeSproutReport>>(`/api/v1/organize/sprout-reports/${encodeURIComponent(id)}`, input)
}

export function getOrganizeSproutReport(id: string) {
  return get<OrganizeResponse<OrganizeSproutReport>>(`/api/v1/organize/sprout-reports/${encodeURIComponent(id)}`)
}
