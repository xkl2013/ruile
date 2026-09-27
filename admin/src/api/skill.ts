import { del, get, patch, postUpload } from '@/utils/request'

export interface GlobalSkill {
  id: string
  name: string
  description: string
  version?: string
  status: string
  enabled: boolean
  created_at?: string
  updated_at?: string
}

const basePath = '/api/v1/system/admin/skills/catalog'

export function listGlobalSkills() {
  return get<{ data: GlobalSkill[] }>(basePath)
}

export function uploadGlobalSkill(file: File) {
  const form = new FormData()
  form.append('file', file)
  return postUpload(basePath, form, undefined, { timeout: 300000 }) as Promise<{ data: GlobalSkill }>
}

export function setGlobalSkillEnabled(id: string, enabled: boolean) {
  return patch<void>(`${basePath}/${encodeURIComponent(id)}`, { enabled })
}

export function deleteGlobalSkill(id: string) {
  return del<void>(`${basePath}/${encodeURIComponent(id)}`)
}

export function listGlobalSkillFiles(id: string) {
  return get<{ data: string[] }>(`${basePath}/${encodeURIComponent(id)}/files`)
}

export function readGlobalSkillFile(id: string, filePath: string) {
  return get<string>(`${basePath}/${encodeURIComponent(id)}/file`, {
    params: { path: filePath },
    responseType: 'text',
  })
}
