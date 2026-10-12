import { del, get, patch, post, postUpload, put } from "../../utils/request";

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
  source_type?: string;
  source_ref?: string;
  env_var_names?: string[];
  status: string;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface SkillInstallJob {
  id: string;
  task_id?: string;
  source_type: string;
  source_ref: string;
  skill_id?: string;
  status: 'pending' | 'running' | 'succeeded' | 'failed' | 'cancelling' | 'cancelled' | string;
  error?: string;
  attempt: number;
  cancel_requested: boolean;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface TenantSandboxConfig {
  id: string;
  name: string;
  mode: 'docker' | 'local' | 'disabled' | string;
  enabled: boolean;
  active: boolean;
  fallback_enabled: boolean;
  timeout_seconds: number;
  docker_image?: string;
  created_at?: string;
  updated_at?: string;
}

export interface TenantSandboxCheck {
  config_id: string;
  mode: string;
  available: boolean;
  effective_mode: string;
  message?: string;
}

export interface TenantSkillSnapshot {
  id: string;
  generation: number;
  parent_snapshot_id?: string;
  state: string;
  manifest_sha256: string;
  manifest?: Record<string, unknown>;
  created_at?: string;
  activated_at?: string;
  superseded_at?: string;
}

export function listTenantSandboxConfigs() {
  return get<{ data: TenantSandboxConfig[] }>('/api/v1/sandbox/configs');
}

export function createTenantSandboxConfig(input: Partial<TenantSandboxConfig>) {
  return post<{ data: TenantSandboxConfig }>('/api/v1/sandbox/configs', input);
}

export function updateTenantSandboxConfig(id: string, input: Partial<TenantSandboxConfig>) {
  return put<{ data: TenantSandboxConfig }>(`/api/v1/sandbox/configs/${encodeURIComponent(id)}`, input);
}

export function activateTenantSandboxConfig(id: string) {
  return post<{ data: TenantSandboxConfig }>(`/api/v1/sandbox/configs/${encodeURIComponent(id)}/activate`, {});
}

export function deactivateTenantSandboxConfig(id: string) {
  return post<{ data: TenantSandboxConfig }>(`/api/v1/sandbox/configs/${encodeURIComponent(id)}/deactivate`, {});
}

export function checkTenantSandboxConfig(id: string) {
  return post<{ data: TenantSandboxCheck }>(`/api/v1/sandbox/configs/${encodeURIComponent(id)}/check`, {});
}

export function deleteTenantSandboxConfig(id: string) {
  return del<void>(`/api/v1/sandbox/configs/${encodeURIComponent(id)}`);
}

export function listTenantSkillSnapshots() {
  return get<{ data: TenantSkillSnapshot[] }>('/api/v1/skills/snapshots');
}

export function createTenantSkillSnapshot() {
  return post<{ data: TenantSkillSnapshot }>('/api/v1/skills/snapshots', {});
}

export function activateTenantSkillSnapshot(id: string) {
  return post<{ data: TenantSkillSnapshot }>(`/api/v1/skills/snapshots/${encodeURIComponent(id)}/activate`, {});
}

export function rollbackTenantSkillSnapshot(id: string) {
  return post<{ data: TenantSkillSnapshot }>(`/api/v1/skills/snapshots/${encodeURIComponent(id)}/rollback`, {});
}

export function verifyTenantSkillSnapshot(id: string) {
  return get<{ data: { valid: boolean } }>(`/api/v1/skills/snapshots/${encodeURIComponent(id)}/verify`);
}

export function listTenantSkills() {
  return get<{ data: TenantSkill[] }>('/api/v1/skills/catalog');
}

export function uploadTenantSkill(file: File) {
  const form = new FormData()
  form.append('file', file)
  return postUpload('/api/v1/skills/catalog', form, undefined, { timeout: 300000 }) as Promise<{ data: TenantSkill }>
}

export function uploadTenantSkillFromSource(source: string) {
  return post<{ data: TenantSkill }>('/api/v1/skills/catalog/source', { source }, { timeout: 300000 })
}

export function startTenantSkillSourceInstall(source: string) {
  return post<{ data: SkillInstallJob }>('/api/v1/skills/catalog/install-jobs', { source })
}

export function getTenantSkillInstallJob(id: string) {
  return get<{ data: SkillInstallJob }>(`/api/v1/skills/catalog/install-jobs/${encodeURIComponent(id)}`)
}

export function cancelTenantSkillInstall(id: string) {
  return post<{ data: SkillInstallJob }>(
    `/api/v1/skills/catalog/install-jobs/${encodeURIComponent(id)}/cancel`,
    {},
  )
}

export function retryTenantSkillInstall(id: string) {
  return post<{ data: SkillInstallJob }>(
    `/api/v1/skills/catalog/install-jobs/${encodeURIComponent(id)}/retry`,
    {},
  )
}

export function setTenantSkillEnabled(id: string, enabled: boolean) {
  return patch<void>(`/api/v1/skills/catalog/${encodeURIComponent(id)}`, { enabled })
}

export function setTenantSkillEnvVars(id: string, envVars: Record<string, string>) {
  return put<void>(`/api/v1/skills/catalog/${encodeURIComponent(id)}/env`, { env_vars: envVars })
}

export function getTenantSkillEnvVars(id: string) {
  return get<{ data: Record<string, string> }>(`/api/v1/skills/catalog/${encodeURIComponent(id)}/env`)
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
