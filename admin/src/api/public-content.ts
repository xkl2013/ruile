import { get, post, postUpload, put } from '@/utils/request'

export type PublicContentType = 'post' | 'course'
export type PublicContentStatus = 'draft' | 'pending_review' | 'published' | 'offline' | 'rejected'

export interface AdminPublicContent {
  id: string
  tenant_id?: number | string
  user_id?: string
  title: string
  content?: string
  output_type?: string
  source_summary?: string
  status?: string
  public_content_type: PublicContentType
  public_status: PublicContentStatus
  series_id?: string
  series_title?: string
  series_order?: number
  review_note?: string
  published_at?: string
  published_by?: string
  icon?: string
  metadata?: Record<string, any>
  featured?: boolean
  recommendable?: boolean
  sort_order?: number
  created_at: string
  updated_at: string
}

export interface AdminPublicContentListData {
  items: AdminPublicContent[]
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

export function listAdminPublicContents(params?: {
  status?: PublicContentStatus | 'all'
  contentType?: PublicContentType | 'all'
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return get<{ success: boolean; data: AdminPublicContentListData }>(
    withQuery('/api/v1/system/admin/public-contents', {
      status: params?.status === 'all' ? undefined : params?.status,
      content_type: params?.contentType === 'all' ? undefined : params?.contentType,
      keyword: params?.keyword,
      page: params?.page,
      page_size: params?.pageSize,
    }),
  )
}

export function uploadAdminPublicContent(file: File) {
  const formData = new FormData()
  formData.append('file', file)
  return postUpload(
    '/api/v1/system/admin/public-contents/upload',
    formData,
    undefined,
    { timeout: 300000 },
  ) as Promise<{ success: boolean; data: AdminPublicContent }>
}

export function updateAdminPublicContent(id: string, data: {
  title: string
  source_summary?: string
  public_content_type?: PublicContentType
  series_id?: string
  series_title?: string
  series_order?: number
  review_note?: string
  metadata?: Record<string, any>
  featured?: boolean
  recommendable?: boolean
  sort_order?: number
}) {
  return put<{ success: boolean; data: AdminPublicContent }>(
    `/api/v1/system/admin/public-contents/${encodeURIComponent(id)}`,
    data,
  )
}

function moderate(id: string, action: 'publish' | 'offline' | 'reject', reviewNote?: string) {
  return post<{ success: boolean; data: AdminPublicContent }>(
    `/api/v1/system/admin/public-contents/${encodeURIComponent(id)}/${action}`,
    reviewNote ? { review_note: reviewNote } : {},
  )
}

export function publishAdminPublicContent(id: string) {
  return moderate(id, 'publish')
}

export function offlineAdminPublicContent(id: string) {
  return moderate(id, 'offline')
}

export function rejectAdminPublicContent(id: string, reviewNote?: string) {
  return moderate(id, 'reject', reviewNote)
}
