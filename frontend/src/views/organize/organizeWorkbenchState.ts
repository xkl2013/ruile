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
  markdownTemplate: string
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
  jobMode: 'single' | 'batch'
  selectedCount: number
  readyCount: number
  processedCount: number
  failedCount: number
  overlapCount: number
  batchCount: number
  coverageRatio: number
  conclusionCount?: number
  todoCount?: number
  outputId?: string
  assignmentStatus?: 'pending' | 'assigned'
  assignmentReason?: string
  assignedServiceId?: string
  errorMessage?: string
  fresh?: boolean
}

export interface OrganizeConfig {
  id: string
  name: string
  templateKey: string
  targetServiceId: string
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
  preview: string
  createdAt: string
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
  fieldValues: Record<string, string>
  citations: OrganizeCitation[]
  content: string
  assignmentStatus: 'pending' | 'assigned'
  assignmentReason: string
  assignedServiceId: string
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

const cleanOrganizeMarkdownPreview = (value: string) =>
  value
    .replace(/!\[[^\]]*]\([^)]*\)/g, ' ')
    .replace(/\[([^\]]+)]\([^)]*\)/g, '$1')
    .replace(/\[(?:M|S)\d+]/gi, '')
    .replace(/[*_`~]/g, '')
    .replace(/\s+/g, ' ')
    .trim()

export const organizeOutputPreview = (content: string, fallback = '') => {
  const lines = String(content || '').split(/\r?\n/)
  const quoted = lines
    .map((line) => line.trim())
    .filter((line) => line.startsWith('>'))
    .map((line) => cleanOrganizeMarkdownPreview(line.replace(/^>\s*/, '')))
    .filter(Boolean)
  const bodyLine = lines
    .map((line) => line.trim())
    .find((line) =>
      Boolean(line) &&
      !/^#{1,6}\s/.test(line) &&
      !/^\|/.test(line) &&
      !/^[-:|\s]+$/.test(line),
    )
  const preview = cleanOrganizeMarkdownPreview(quoted.join(' ') || bodyLine || fallback)
  if (preview.length <= 260) return preview
  return `${preview.slice(0, 260).trim()}...`
}

export const toOrganizeTemplate = (template: ApiOrganizeTemplate): OrganizeTemplate => ({
  key: template.key,
  name: template.name,
  scene: template.scene,
  description: template.description,
  outputLabel: template.output_label,
  icon: template.icon || 'dashboard',
  defaultInstruction: template.default_instruction,
  markdownTemplate: template.markdown_template || '',
  expertIds: [...(template.expert_ids || [])],
  publishedVersion: template.published_version,
})

export const toOrganizeJob = (job: ApiOrganizeJob): OrganizeJob => {
  const timestamp = job.started_at || job.created_at
  const dateTime = formatDateTime(timestamp)
  const selectedCount = job.selected_count ?? job.memory_ids?.length ?? 0
  const processedCount = job.processed_count ?? toNumber(job.result?.processed_count)
  const batchCount = job.batch_count ?? toNumber(job.result?.batch_count)
  const resultCoverage = job.result?.coverage as Record<string, unknown> | undefined
  const coverageRatio = Number(job.coverage?.ratio ?? resultCoverage?.ratio ?? 0)
  return {
    id: job.id,
    ...dateTime,
    rangeLabel: `${selectedCount} 条记忆${batchCount > 1 ? ` · ${batchCount} 批` : ''}`,
    templateVersion: job.template_version || '-',
    state: job.status,
    summary: job.summary || job.error_message || '等待执行',
    stage: job.stage,
    progress: job.progress,
    jobMode: job.job_mode || (batchCount > 1 ? 'batch' : 'single'),
    selectedCount,
    readyCount: job.ready_count ?? selectedCount,
    processedCount,
    failedCount: job.failed_count ?? toNumber(job.result?.failed_count),
    overlapCount: job.overlap_count ?? toNumber(job.result?.overlap_count),
    batchCount,
    coverageRatio: Number.isFinite(coverageRatio) ? coverageRatio : 0,
    conclusionCount: toNumber(job.result?.conclusion_count),
    todoCount: toNumber(job.result?.todo_count),
    outputId: job.output_id,
    assignmentStatus: job.result?.assignment_status === 'assigned' ? 'assigned' : 'pending',
    assignmentReason: String(job.result?.assignment_reason || ''),
    assignedServiceId: String(job.result?.assigned_service_id || ''),
    errorMessage: job.error_message,
    fresh: Boolean(job.finished_at && Date.now() - new Date(job.finished_at).getTime() < 5 * 60 * 1000),
  }
}

export const toOrganizeConfig = (config: ApiOrganizeConfig): OrganizeConfig => ({
  id: config.id,
  name: config.name,
  templateKey: config.template_key,
  targetServiceId: config.target_service_id || '',
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
  const fieldValues = Object.fromEntries(fields.map((field) => [field.label, field.value]))
  return {
    id: output.id,
    configId: output.config_id || '',
    jobId: output.job_id || '',
    title: output.title,
    preview: organizeOutputPreview(output.content, output.source_summary || output.output_type),
    createdAt: output.created_at || output.updated_at || '',
    subject: output.source_summary || output.output_type || '整理结果',
    templateKey: output.template_key || '',
    templateName: output.template_key
      ? template?.name || output.output_type || '整理结果'
      : '历史内容',
    templateVersion: output.template_version || '-',
    date: (output.updated_at || output.created_at || '').slice(0, 10),
    tags: toStringList(metadata.tags),
    conclusionCount: toNumber(metadata.conclusion_count),
    todoCount: toNumber(metadata.todo_count),
    sourceCount: output.memory_count || output.memory_ids?.length || 0,
    fields,
    fieldValues,
    citations,
    content: output.content || '',
    assignmentStatus: output.assignment_status || 'pending',
    assignmentReason: output.assignment_reason || String(metadata.assignment_reason || ''),
    assignedServiceId: output.assigned_service_id || String(metadata.assigned_service_id || ''),
  }
}
