import { get, post } from '@/utils/request'

export type PublicCreatorStatus = 'all' | 'pending' | 'published'
export type PublicCreatorPublicationStatus = 'unpublished' | 'draft' | 'published' | 'offline'

export interface AdminPublicCreatorSummary {
  id: string
  display_name: string
  username: string
  email: string
  avatar?: string
  knowledge_base_count: number
  content_count: number
  published_knowledge_base_count: number
  published_content_count: number
  pending_knowledge_base_count: number
  pending_content_count: number
}

export interface AdminPublicCreatorKnowledgeBase {
  id: string
  name: string
  description?: string
  type?: string
  icon?: string
  creator_id: string
  tenant_id: number
  publication_id?: string
  publication_status: PublicCreatorPublicationStatus
  knowledge_count: number
  chunk_count: number
  is_processing: boolean
  processing_count: number
  can_publish: boolean
  publish_block_reason?: string
  created_at: string
  updated_at: string
}

export interface AdminPublicCreatorContent {
  id: string
  title: string
  content?: string
  output_type?: string
  source_summary?: string
  public_content_type: 'post' | 'course'
  public_status: string
  series_id?: string
  series_title?: string
  series_order?: number
  review_note?: string
  published_at?: string
  metadata?: Record<string, any>
  created_at: string
  updated_at: string
}

export interface AdminPublicCreatorDetail extends AdminPublicCreatorSummary {
  knowledge_bases: AdminPublicCreatorKnowledgeBase[]
  contents: AdminPublicCreatorContent[]
}

export interface AdminPublicCreatorPublishFailure {
  asset_type: 'knowledge_base' | 'content' | string
  asset_id: string
  title: string
  message: string
}

export interface AdminPublicCreatorPublishResult {
  creator_id: string
  knowledge_bases_published: number
  contents_published: number
  knowledge_bases_offlined: number
  contents_offlined: number
  knowledge_bases_skipped: number
  contents_skipped: number
  failures: AdminPublicCreatorPublishFailure[]
}

export interface AdminPublicCreatorListData {
  items: AdminPublicCreatorSummary[]
  total: number
  page: number
  page_size: number
}

function withQuery(path: string, params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return `${path}${suffix}`
}

export function listAdminPublicCreators(params?: {
  status?: PublicCreatorStatus
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return get<{ success: boolean; data: AdminPublicCreatorListData }>(
    withQuery('/api/v1/system/admin/creators', {
      status: params?.status === 'all' ? undefined : params?.status,
      keyword: params?.keyword,
      page: params?.page,
      page_size: params?.pageSize,
    }),
  )
}

export function getAdminPublicCreator(id: string) {
  return get<{ success: boolean; data: AdminPublicCreatorDetail }>(
    `/api/v1/system/admin/creators/${encodeURIComponent(id)}`,
  )
}

export function publishAdminPublicCreatorKnowledgeBase(creatorId: string, knowledgeBaseId: string) {
  return post<{ success: boolean; data: AdminPublicCreatorPublishResult }>(
    `/api/v1/system/admin/creators/${encodeURIComponent(creatorId)}/knowledge-bases/${encodeURIComponent(knowledgeBaseId)}/publish`,
    {},
  )
}

function moderateCreator(id: string, action: 'publish' | 'offline') {
  return post<{ success: boolean; data: AdminPublicCreatorPublishResult }>(
    `/api/v1/system/admin/creators/${encodeURIComponent(id)}/${action}`,
    {},
  )
}

export function publishAdminPublicCreator(id: string) {
  return moderateCreator(id, 'publish')
}

export function offlineAdminPublicCreator(id: string) {
  return moderateCreator(id, 'offline')
}
