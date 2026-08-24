import { del, get, post, postUpload, put } from '@/utils/request'

export type OrganizeMemoryKind = 'note' | 'record' | 'audio' | 'audio_card'
export type OrganizeOutputStatus = 'draft' | 'review' | 'ready' | 'archived'
export type OrganizeSproutStage = 'organizing' | 'expandable' | 'formed'

export interface OrganizeMemory {
  id: string
  kind: OrganizeMemoryKind
  title: string
  content: string
  source?: string
  occurred_at: string
  duration_seconds?: number
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
  title: string
  output_type: string
  content: string
  source_summary?: string
  status: OrganizeOutputStatus
  icon?: string
  creator_name?: string
  creator_avatar?: string
  is_subscribed?: boolean
  memory_count?: number
  memory_ids?: string[]
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface OrganizeSproutReport {
  id: string
  user_id?: string
  title: string
  summary: string
  stage: OrganizeSproutStage
  output_hint?: string
  chips?: string[]
  memory_count?: number
  memory_ids?: string[]
  memory_refs?: OrganizeMemoryReference[]
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

export interface OrganizeDiscoverData {
  tabs: OrganizeDiscoverTab[]
  featured_outputs: OrganizeOutput[]
  items: OrganizeOutput[]
  total: number
  page?: number
  page_size?: number
  featured_offset?: number
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
  output_type?: string
  content?: string
  source_summary?: string
  status?: OrganizeOutputStatus
  icon?: string
  memory_ids?: string[]
  metadata?: Record<string, unknown>
}

export interface OrganizeSproutReportInput {
  title: string
  summary?: string
  stage?: OrganizeSproutStage
  output_hint?: string
  chips?: string[]
  memory_ids?: string[]
  metadata?: Record<string, unknown>
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

export function createOrganizeMemory(input: OrganizeMemoryInput) {
  return post<OrganizeResponse<OrganizeMemory>>('/api/v1/organize/memories', input)
}

export function uploadOrganizeMemory(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return postUpload('/api/v1/organize/memories/upload', formData, undefined, { timeout: 300000 }) as Promise<OrganizeResponse<OrganizeMemory>>
}

export function updateOrganizeMemory(id: string, input: OrganizeMemoryInput) {
  return put<OrganizeResponse<OrganizeMemory>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`, input)
}

export function getOrganizeMemory(id: string) {
  return get<OrganizeResponse<OrganizeMemory>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`)
}

export function deleteOrganizeMemory(id: string) {
  return del<OrganizeResponse<null>>(`/api/v1/organize/memories/${encodeURIComponent(id)}`)
}

export function listOrganizeOutputs(params?: OrganizeListParams & { status?: OrganizeOutputStatus }) {
  return get<OrganizeResponse<OrganizeListData<OrganizeOutput>>>(withQuery('/api/v1/organize/outputs', params))
}

export function getOrganizeDiscover(params?: OrganizeListParams & { tab?: string; featured_offset?: number }) {
  return get<OrganizeResponse<OrganizeDiscoverData>>(withQuery('/api/v1/organize/discover', params))
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
