import { del, get, patch, post, postUpload, put } from '@/utils/request'

export interface GlobalSkill {
  id: string
  name: string
  description: string
  version?: string
  source_type?: string
  source_ref?: string
  env_var_names?: string[]
  status: string
  enabled: boolean
  created_at?: string
  updated_at?: string
}

export interface SkillInstallJob {
  id: string
  task_id?: string
  source_type: string
  source_ref: string
  skill_id?: string
  status: 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelling' | 'cancelled' | string
  error?: string
  attempt: number
  cancel_requested: boolean
  started_at?: string
  finished_at?: string
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

export function uploadGlobalSkillFromSource(source: string) {
  return post<{ data: GlobalSkill }>(`${basePath}/source`, { source }, { timeout: 300000 })
}

export function startGlobalSkillSourceInstall(source: string) {
  return post<{ data: SkillInstallJob }>(`${basePath}/install-jobs`, { source })
}

export function getGlobalSkillInstallJob(id: string) {
  return get<{ data: SkillInstallJob }>(`${basePath}/install-jobs/${encodeURIComponent(id)}`)
}

export function cancelGlobalSkillInstall(id: string) {
  return post<{ data: SkillInstallJob }>(
    `${basePath}/install-jobs/${encodeURIComponent(id)}/cancel`,
    {},
  )
}

export function retryGlobalSkillInstall(id: string) {
  return post<{ data: SkillInstallJob }>(
    `${basePath}/install-jobs/${encodeURIComponent(id)}/retry`,
    {},
  )
}

export function setGlobalSkillEnabled(id: string, enabled: boolean) {
  return patch<void>(`${basePath}/${encodeURIComponent(id)}`, { enabled })
}

export function setGlobalSkillEnvVars(id: string, envVars: Record<string, string>) {
  return put<void>(`${basePath}/${encodeURIComponent(id)}/env`, { env_vars: envVars })
}

export function getGlobalSkillEnvVars(id: string) {
  return get<{ data: Record<string, string> }>(`${basePath}/${encodeURIComponent(id)}/env`)
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
