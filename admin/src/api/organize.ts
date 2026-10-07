import { get, post, put } from '@/utils/request'

export type OrganizeTemplateStatus = 'draft' | 'enabled' | 'disabled'

export interface AdminOrganizeTemplate {
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
  spec: Record<string, any>
  status: OrganizeTemplateStatus
  published_version: string
  published_at?: string
  published_by?: string
  validation_result?: Record<string, any>
  sort_order: number
  created_at: string
  updated_at: string
}

export interface AdminOrganizeTemplateVersion {
  id: string
  template_id: string
  template_key: string
  version: string
  snapshot: Record<string, any>
  created_by?: string
  change_note?: string
  created_at: string
}

export interface AdminOrganizeTemplateListData {
  items: AdminOrganizeTemplate[]
  total: number
  page: number
  page_size: number
}

export interface AdminOrganizeTemplateVersionListData {
  items: AdminOrganizeTemplateVersion[]
  total: number
  page: number
  page_size: number
}

export interface AdminOrganizeTemplateInput {
  key?: string
  name: string
  scene?: string
  description?: string
  output_label?: string
  icon?: string
  default_instruction?: string
  markdown_template?: string
  expert_ids?: string[]
  spec?: Record<string, any>
  sort_order?: number
  change_note?: string
}

export interface AdminOrganizeTemplateCompileResult {
  key: string
  name: string
  scene: string
  description: string
  output_label: string
  default_instruction: string
  markdown_template: string
  spec: Record<string, any>
  sections: string[]
  warnings: string[]
}

export function compileAdminOrganizeTemplate(sourceMarkdown: string) {
  return post<{ success: boolean; data: AdminOrganizeTemplateCompileResult }>(
    '/api/v1/system/admin/organize/templates/compile',
    { source_markdown: sourceMarkdown },
  )
}

export type AdminOrganizeDiscoverCategoryStatus = 'enabled' | 'disabled'

export interface AdminOrganizeDiscoverCategory {
  id: string
  key: string
  label: string
  description: string
  sort_order: number
  status: AdminOrganizeDiscoverCategoryStatus
  created_at: string
  updated_at: string
}

export interface AdminOrganizeDiscoverCategoryListData {
  items: AdminOrganizeDiscoverCategory[]
  total: number
  page: number
  page_size: number
}

export interface AdminOrganizeDiscoverCategoryInput {
  key: string
  label: string
  description?: string
  sort_order?: number
}

function withQuery(path: string, params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return `${path}${suffix}`
}

export function listAdminOrganizeTemplates(params?: {
  keyword?: string
  scene?: string
  status?: OrganizeTemplateStatus | 'all'
  page?: number
  pageSize?: number
}) {
  return get<{ success: boolean; data: AdminOrganizeTemplateListData }>(
    withQuery('/api/v1/system/admin/organize/templates', {
      q: params?.keyword,
      scene: params?.scene,
      status: params?.status === 'all' ? undefined : params?.status,
      page: params?.page,
      page_size: params?.pageSize,
    }),
  )
}

export function createAdminOrganizeTemplate(input: AdminOrganizeTemplateInput) {
  return post<{ success: boolean; data: AdminOrganizeTemplate }>(
    '/api/v1/system/admin/organize/templates',
    input,
  )
}

export function updateAdminOrganizeTemplate(key: string, input: AdminOrganizeTemplateInput) {
  return put<{ success: boolean; data: AdminOrganizeTemplate }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}`,
    input,
  )
}

export function previewAdminOrganizeTemplate(key: string, variables?: Record<string, any>) {
  return post<{
    success: boolean
    data: {
      template_key: string
      version: string
      prompt: string
      markdown_template: string
      spec: Record<string, any>
      errors: string[]
    }
  }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}/preview`,
    { variables: variables || {} },
  )
}

export function publishAdminOrganizeTemplate(key: string, changeNote?: string) {
  return post<{ success: boolean; data: AdminOrganizeTemplate }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}/publish`,
    { change_note: changeNote || '' },
  )
}

export function disableAdminOrganizeTemplate(key: string) {
  return post<{ success: boolean; data: AdminOrganizeTemplate }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}/disable`,
    {},
  )
}

export function listAdminOrganizeTemplateVersions(key: string) {
  return get<{ success: boolean; data: AdminOrganizeTemplateVersionListData }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}/versions?page=1&page_size=100`,
  )
}

export function rollbackAdminOrganizeTemplate(key: string, version: string, changeNote?: string) {
  return post<{ success: boolean; data: AdminOrganizeTemplate }>(
    `/api/v1/system/admin/organize/templates/${encodeURIComponent(key)}/rollback`,
    { version, change_note: changeNote || '' },
  )
}

export function listAdminOrganizeDiscoverCategories(params?: {
  keyword?: string
  status?: AdminOrganizeDiscoverCategoryStatus | 'all'
  page?: number
  pageSize?: number
}) {
  return get<{ success: boolean; data: AdminOrganizeDiscoverCategoryListData }>(
    withQuery('/api/v1/system/admin/organize/discover-categories', {
      q: params?.keyword,
      status: params?.status === 'all' ? undefined : params?.status,
      page: params?.page,
      page_size: params?.pageSize,
    }),
  )
}

export function createAdminOrganizeDiscoverCategory(input: AdminOrganizeDiscoverCategoryInput) {
  return post<{ success: boolean; data: AdminOrganizeDiscoverCategory }>(
    '/api/v1/system/admin/organize/discover-categories',
    input,
  )
}

export function updateAdminOrganizeDiscoverCategory(
  key: string,
  input: AdminOrganizeDiscoverCategoryInput,
) {
  return put<{ success: boolean; data: AdminOrganizeDiscoverCategory }>(
    `/api/v1/system/admin/organize/discover-categories/${encodeURIComponent(key)}`,
    input,
  )
}

export function disableAdminOrganizeDiscoverCategory(key: string) {
  return post<{ success: boolean; data: AdminOrganizeDiscoverCategory }>(
    `/api/v1/system/admin/organize/discover-categories/${encodeURIComponent(key)}/disable`,
    {},
  )
}
