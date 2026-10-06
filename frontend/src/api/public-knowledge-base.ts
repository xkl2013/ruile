import { del, get, post } from '@/utils/request'

export type PublicKnowledgeBaseStatus = 'draft' | 'published' | 'offline'

export interface PublicKnowledgeBasePublication {
  id: string
  knowledge_base_id: string
  title: string
  description: string
  category: string
  status: PublicKnowledgeBaseStatus
  featured: boolean
  recommendable: boolean
  sort_order?: number
  published_at?: string
  updated_at: string
  subscriber_count: number
  is_subscribed: boolean
  name?: string
  icon?: string
  icon_url?: string
  type?: string
  knowledge_count?: number
  chunk_count?: number
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
