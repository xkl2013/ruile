import type {
  OrganizeConfig as ApiOrganizeConfig,
  OrganizeExpert,
  OrganizeJob as ApiOrganizeJob,
  OrganizeJobStatus,
  OrganizeOutput as ApiOrganizeOutput,
  OrganizeScheduleKey,
  OrganizeTemplate as ApiOrganizeTemplate,
} from '@/api/organize'

export type { OrganizeExpert, OrganizeScheduleKey }

export interface OrganizeTemplate {
  key: string
  name: string
  scene: string
  description: string
  outputLabel: string
  icon: string
  defaultInstruction: string
  expertIds: string[]
  publishedVersion: string
}

export interface OrganizeJob {
  id: string
  dateLabel: string
  timeLabel: string
  rangeLabel: string
  templateVersion: string
  state: OrganizeJobStatus
  summary: string
  stage?: string
  progress?: number
  conclusionCount?: number
  todoCount?: number
  outputId?: string
  errorMessage?: string
  fresh?: boolean
}

export interface OrganizeConfig {
  id: string
  name: string
  templateKey: string
  instruction: string
  expertIds: string[]
  schedule: OrganizeScheduleKey
  status: 'active' | 'disabled'
  jobs: OrganizeJob[]
  jobCount: number
  updatedOrder: number
  template?: OrganizeTemplate
}

export interface OrganizeOutputField {
  label: string
  value: string
}

export interface OrganizeCitation {
  label: string
  id: string
  kind?: string
  title: string
  source?: string
}

export interface OrganizeOutput {
  id: string
  configId: string
  jobId: string
  title: string
  subject: string
  templateKey: string
  templateName: string
  templateVersion: string
  date: string
  tags: string[]
  conclusionCount: number
  todoCount: number
  sourceCount: number
  fields: OrganizeOutputField[]
  citations: OrganizeCitation[]
  content: string
}

export const organizeScheduleLabels: Record<OrganizeScheduleKey, string> = {
  manual: '手动触发',
  daily: '每天 08:00',
  weekly: '每周一 09:00',
  monthly: '每月 1 日 09:00',
}

const toNumber = (value: unknown) => {
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : 0
}

const toStringList = (value: unknown) =>
  Array.isArray(value)
    ? value.map((item) => String(item).trim()).filter(Boolean)
    : []

const formatDateTime = (value?: string) => {
  const date = value ? new Date(value) : new Date()
  if (Number.isNaN(date.getTime())) {
    return { dateLabel: value || '', timeLabel: '' }
  }
  const today = new Date()
  const dateText = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
  const todayText = new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(today)
  return {
    dateLabel: dateText === todayText ? '今天' : dateText.replaceAll('/', '-'),
    timeLabel: new Intl.DateTimeFormat('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    }).format(date),
  }
}

const formatFieldValue = (value: unknown): string => {
  if (Array.isArray(value)) return value.map((item) => String(item)).join(' / ')
  if (value == null) return ''
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

export const toOrganizeTemplate = (template: ApiOrganizeTemplate): OrganizeTemplate => ({
  key: template.key,
  name: template.name,
  scene: template.scene,
  description: template.description,
  outputLabel: template.output_label,
  icon: template.icon || 'dashboard',
  defaultInstruction: template.default_instruction,
  expertIds: [...(template.expert_ids || [])],
  publishedVersion: template.published_version,
})

export const toOrganizeJob = (job: ApiOrganizeJob): OrganizeJob => {
  const timestamp = job.started_at || job.created_at
  const dateTime = formatDateTime(timestamp)
  return {
    id: job.id,
    ...dateTime,
    rangeLabel: `${job.memory_ids?.length || 0} 条记忆`,
    templateVersion: job.template_version || '-',
    state: job.status,
    summary: job.summary || job.error_message || '等待执行',
    stage: job.stage,
    progress: job.progress,
    conclusionCount: toNumber(job.result?.conclusion_count),
    todoCount: toNumber(job.result?.todo_count),
    outputId: job.output_id,
    errorMessage: job.error_message,
    fresh: Boolean(job.finished_at && Date.now() - new Date(job.finished_at).getTime() < 5 * 60 * 1000),
  }
}

export const toOrganizeConfig = (config: ApiOrganizeConfig): OrganizeConfig => ({
  id: config.id,
  name: config.name,
  templateKey: config.template_key,
  instruction: config.instruction,
  expertIds: [...(config.expert_ids || [])],
  schedule: config.schedule,
  status: config.status,
  jobs: config.latest_job ? [toOrganizeJob(config.latest_job)] : [],
  jobCount: config.job_count || 0,
  updatedOrder: new Date(config.updated_at).getTime() || 0,
  template: config.template ? toOrganizeTemplate(config.template) : undefined,
})

export const toOrganizeOutput = (
  output: ApiOrganizeOutput,
  templates: OrganizeTemplate[] = [],
): OrganizeOutput => {
  const template = templates.find((item) => item.key === output.template_key)
  const metadata = output.metadata || {}
  const rawRefs = output.citations?.memory_refs
  const citations: OrganizeCitation[] = Array.isArray(rawRefs)
    ? rawRefs
        .filter((item): item is Record<string, unknown> => Boolean(item) && typeof item === 'object')
        .map((item) => ({
          label: String(item.label || ''),
          id: String(item.id || ''),
          kind: item.kind ? String(item.kind) : undefined,
          title: String(item.title || ''),
          source: item.source ? String(item.source) : undefined,
        }))
        .filter((item) => item.id && item.label)
    : []
  const fields = Object.entries(output.fields || {}).map(([label, value]) => ({
    label,
    value: formatFieldValue(value),
  }))
  return {
    id: output.id,
    configId: output.config_id || '',
    jobId: output.job_id || '',
    title: output.title,
    subject: output.source_summary || output.output_type || '整理结果',
    templateKey: output.template_key || '',
    templateName: template?.name || output.output_type || '整理结果',
    templateVersion: output.template_version || '-',
    date: (output.updated_at || output.created_at || '').slice(0, 10),
    tags: toStringList(metadata.tags),
    conclusionCount: toNumber(metadata.conclusion_count),
    todoCount: toNumber(metadata.todo_count),
    sourceCount: output.memory_count || output.memory_ids?.length || 0,
    fields,
    citations,
    content: output.content || '',
  }
}
