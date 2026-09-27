import { del, get, patch, postUpload } from "../../utils/request";

// Skill信息
export interface SkillInfo {
  name: string;
  description: string;
}

// 获取预装Skills列表；skills_available 为 false 表示沙箱未启用，前端应隐藏/禁用 Skills 配置
export function listSkills() {
  return get<{ data: SkillInfo[]; skills_available?: boolean }>('/api/v1/skills');
}

export interface TenantSkill {
  id: string;
  name: string;
  description: string;
  version?: string;
  status: string;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export function listTenantSkills() {
  return get<{ data: TenantSkill[] }>('/api/v1/skills/catalog');
}

export function uploadTenantSkill(file: File) {
  const form = new FormData()
  form.append('file', file)
  return postUpload('/api/v1/skills/catalog', form, undefined, { timeout: 300000 }) as Promise<{ data: TenantSkill }>
}

export function setTenantSkillEnabled(id: string, enabled: boolean) {
  return patch<void>(`/api/v1/skills/catalog/${encodeURIComponent(id)}`, { enabled })
}

export function deleteTenantSkill(id: string) {
  return del<void>(`/api/v1/skills/catalog/${encodeURIComponent(id)}`)
}

export function listTenantSkillFiles(id: string) {
  return get<{ data: string[] }>(`/api/v1/skills/catalog/${encodeURIComponent(id)}/files`)
}

export function readTenantSkillFile(id: string, filePath: string) {
  return get<string>(`/api/v1/skills/catalog/${encodeURIComponent(id)}/file`, {
    params: { path: filePath },
    responseType: 'text',
  })
}
