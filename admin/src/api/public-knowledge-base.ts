import { del, get, post, put } from '@/utils/request'

export type PublicKnowledgeBaseStatus = 'draft' | 'published' | 'offline'

export interface PublicKnowledgeBasePublication {
  id: string
  knowledge_base_id: string
  title: string
  description: string
  category: string
  status: PublicKnowledgeBaseStatus
  featured: boolean
  sort_order: number
  published_at?: string
  created_at: string
  updated_at: string
  subscriber_count: number
  is_subscribed?: boolean
  name?: string
  icon?: string
  icon_url?: string
  type?: string
  knowledge_count?: number
  chunk_count?: number
  knowledge_base?: Record<string, any>
}

export function listAdminPublicKnowledgeBases(params?: { status?: PublicKnowledgeBaseStatus | 'all'; keyword?: string }) {
  const query = new URLSearchParams()
  if (params?.status && params.status !== 'all') query.set('status', params.status)
  if (params?.keyword) query.set('keyword', params.keyword)
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return get<{ success: boolean; data: PublicKnowledgeBasePublication[] }>(
    `/api/v1/system/admin/public-knowledge-bases${suffix}`,
  )
}

export function createAdminPublicKnowledgeBase(data: {
  knowledge_base_id: string
  title: string
  description?: string
  category?: string
}) {
  return post<{ success: boolean; data: PublicKnowledgeBasePublication }>(
    '/api/v1/system/admin/public-knowledge-bases',
    data,
  )
}

export function updateAdminPublicKnowledgeBase(id: string, data: {
  title: string
  description?: string
  category?: string
  featured?: boolean
  sort_order?: number
}) {
  return put<{ success: boolean; data: PublicKnowledgeBasePublication }>(
    `/api/v1/system/admin/public-knowledge-bases/${id}`,
    data,
  )
}

export function publishAdminPublicKnowledgeBase(id: string) {
  return post<{ success: boolean; data: PublicKnowledgeBasePublication }>(
    `/api/v1/system/admin/public-knowledge-bases/${id}/publish`,
    {},
  )
}

export function offlineAdminPublicKnowledgeBase(id: string) {
  return post<{ success: boolean; data: PublicKnowledgeBasePublication }>(
    `/api/v1/system/admin/public-knowledge-bases/${id}/offline`,
    {},
  )
}

export function listPublicKnowledgeBases(params?: { keyword?: string; category?: string }) {
  const query = new URLSearchParams()
  if (params?.keyword) query.set('keyword', params.keyword)
  if (params?.category) query.set('category', params.category)
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return get<{ success: boolean; data: PublicKnowledgeBasePublication[] }>(
    `/api/v1/discovery/knowledge-bases${suffix}`,
  )
}

export function listMyPublicKnowledgeBaseSubscriptions() {
  return get<{ success: boolean; data: PublicKnowledgeBasePublication[] }>(
    '/api/v1/discovery/knowledge-bases/subscriptions',
  )
}

export function subscribePublicKnowledgeBase(id: string) {
  return post(`/api/v1/discovery/knowledge-bases/${id}/subscribe`, {})
}

export function unsubscribePublicKnowledgeBase(id: string) {
  return del(`/api/v1/discovery/knowledge-bases/${id}/subscribe`)
}
