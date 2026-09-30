import { del, get, post, postUpload } from '@/utils/request'

export type CourseStatus = 'draft' | 'pending_review' | 'published' | 'offline' | 'rejected'
export type CourseSource = 'official' | 'creator'
export type CourseLessonType = 'video' | 'audio' | 'article'

export interface AdminCourseLesson {
  id: string
  course_id: string
  output_id: string
  title: string
  lesson_type: CourseLessonType
  duration_seconds: number
  sort_order: number
}

export interface AdminCourse {
  id: string
  tenant_id: number
  user_id: string
  source: CourseSource
  title: string
  summary: string
  category: string
  cover_url: string
  teacher_name: string
  teacher_title: string
  public_status: CourseStatus
  lesson_count: number
  learner_count: number
  created_at: string
  updated_at: string
  lessons?: AdminCourseLesson[]
}

export interface AdminCourseSkippedFile {
  file_name: string
  reason: string
}

export interface AdminCourseUploadResult {
  course: AdminCourse
  skipped?: AdminCourseSkippedFile[]
  warnings?: string[]
}

export interface AdminCourseListData {
  items: AdminCourse[]
  total: number
  page: number
  page_size: number
  stats?: AdminCourseStats
}

export interface AdminCourseStats {
  total: number
  published: number
  offline: number
  lesson_total: number
}

export interface CourseUploadMetadata {
  title: string
  summary?: string
  category?: string
  coverUrl?: string
  teacherName?: string
  teacherTitle?: string
  source?: CourseSource
  publicStatus?: CourseStatus
  /** The picked folder's name, used as the course title fallback. */
  folder?: string
}

function withQuery(path: string, params: Record<string, string | number | undefined>) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== '') query.set(key, String(value))
  })
  const suffix = query.toString() ? `?${query.toString()}` : ''
  return `${path}${suffix}`
}

export function listAdminCourses(params?: {
  status?: CourseStatus | 'all'
  source?: CourseSource | 'all'
  category?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  return get<{ success: boolean; data: AdminCourseListData }>(
    withQuery('/api/v1/system/admin/courses', {
      status: params?.status === 'all' ? undefined : params?.status,
      source: params?.source === 'all' ? undefined : params?.source,
      category: params?.category,
      keyword: params?.keyword,
      page: params?.page,
      page_size: params?.pageSize,
    }),
  )
}

export function getAdminCourse(id: string) {
  return get<{ success: boolean; data: AdminCourse }>(
    `/api/v1/system/admin/courses/${encodeURIComponent(id)}`,
  )
}

/**
 * Uploads a whole folder as one course.
 *
 * The backend pairs `files` with `file_paths` by position, which is how
 * __MACOSX / dotfile noise is filtered out. So both lists must be built from
 * the same iteration order — do not resort one of them independently.
 */
export function uploadAdminCourse(
  items: Array<{ file: File; relativePath: string }>,
  meta: CourseUploadMetadata,
  onUploadProgress?: (progressEvent: any) => void,
) {
  const formData = new FormData()
  items.forEach(({ file, relativePath }) => {
    formData.append('files', file)
    formData.append('file_paths', relativePath)
  })
  formData.append('title', meta.title)
  formData.append('summary', meta.summary || '')
  formData.append('category', meta.category || '')
  formData.append('cover_url', meta.coverUrl || '')
  formData.append('teacher_name', meta.teacherName || '')
  formData.append('teacher_title', meta.teacherTitle || '')
  formData.append('source', meta.source || 'official')
  formData.append('public_status', meta.publicStatus || 'published')
  formData.append('folder', meta.folder || '')

  return postUpload(
    '/api/v1/system/admin/courses/upload',
    formData,
    onUploadProgress,
    { timeout: 0 },
  ) as Promise<{ success: boolean; data: AdminCourseUploadResult }>
}

function moderate(id: string, action: 'publish' | 'offline') {
  return post<{ success: boolean; data: AdminCourse }>(
    `/api/v1/system/admin/courses/${encodeURIComponent(id)}/${action}`,
    {},
  )
}

export function publishAdminCourse(id: string) {
  return moderate(id, 'publish')
}

export function offlineAdminCourse(id: string) {
  return moderate(id, 'offline')
}

export function deleteAdminCourse(id: string) {
  return del<{ success: boolean; data: { deleted: boolean } }>(
    `/api/v1/system/admin/courses/${encodeURIComponent(id)}`,
  )
}
