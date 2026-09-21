export interface AgentRunTimelineEvent {
  type?: string
  id?: string
  sequence?: number
  stepId?: string
  stepType?: string
  phase?: string
  status?: string
  label?: string
  message?: string
  createdAt?: string
  startedAt?: string
  finishedAt?: string
  durationMs?: number
  totalDurationMs?: number
  queueDurationMs?: number
  delta?: string
  title?: string
  messageId?: string
  toolCallId?: string
  toolCallName?: string
  activityType?: string
  interaction?: unknown
  quality?: unknown
  answers?: unknown
  content?: unknown
  output?: unknown
  result?: unknown
  error?: unknown
  errorCode?: string
  [key: string]: unknown
}

export interface AgentRunTimelineStep {
  id: string
  stepType: string
  phase: string
  label: string
  message: string
  status: 'running' | 'succeeded' | 'failed'
  startedAt?: string
  finishedAt?: string
  durationMs: number
  sequence: number
}

export interface AgentRunStepDetail {
  id?: string
  step_type?: string
  input?: unknown
  output?: unknown
  error?: string
}

export interface AgentRunEventDetail {
  label: string
  value: unknown
  tone?: 'default' | 'error'
}

export interface AgentRunEventRow {
  id: string
  type: string
  label: string
  summary: string
  status: 'running' | 'succeeded' | 'failed'
  sequence: number
  durationMs: number
  details: AgentRunEventDetail[]
}

const phaseLabels: Record<string, string> = {
  intake: '需求澄清',
  planning: '执行规划',
  drafting: '报告撰写',
  reviewing: '质量审查',
  revising: '报告修订',
  packaging: '结果整理',
  completed: '任务完成',
}

const terminalRunStatuses = new Set(['succeeded', 'failed', 'cancelled'])
const eventLabels: Record<string, string> = {
  RUN_QUEUED: '任务进入队列',
  RUN_STARTED: '开始执行任务',
  RUN_WAITING_INPUT: '等待补充信息',
  RUN_RESUMED: '收到补充信息，继续执行',
  RUN_FINISHED: '任务执行完成',
  RUN_ERROR: '任务执行失败',
  RUN_CANCELLED: '任务已取消',
  REASONING_START: '开始执行过程',
  REASONING_MESSAGE_START: '开始记录执行进度',
  REASONING_MESSAGE_CONTENT: '执行进度更新',
  REASONING_MESSAGE_END: '执行进度记录完成',
  REASONING_END: '执行过程结束',
  AGENT_STEP_STARTED: '开始执行步骤',
  AGENT_STEP_FINISHED: '步骤执行完成',
  AGENT_STEP_ERROR: '步骤执行失败',
  TOOL_CALL_START: '开始调用工具',
  TOOL_CALL_ARGS: '工具调用入参',
  TOOL_CALL_END: '工具调用结束',
  TOOL_CALL_RESULT: '工具返回结果',
  TEXT_MESSAGE_START: '开始生成回复',
  TEXT_MESSAGE_CONTENT: '回复内容更新',
  TEXT_MESSAGE_END: '回复生成完成',
  ACTIVITY_SNAPSHOT: '产物列表更新',
  ACTIVITY_DELTA: '产物增量更新',
  QUALITY_UPDATED: '质量审查结果',
  EXPERT_ROUTING: '已匹配专家',
}

const eventTechnicalKeys = new Set([
  'type',
  'id',
  'runId',
  'threadId',
  'sequence',
  'createdAt',
  'startedAt',
  'finishedAt',
  'durationMs',
  'totalDurationMs',
])

const dateMilliseconds = (value?: string) => {
  if (!value) return 0
  const timestamp = Date.parse(value)
  return Number.isFinite(timestamp) ? timestamp : 0
}

const eventSequence = (event: AgentRunTimelineEvent, index: number) => {
  const sequence = Number(event.sequence)
  return Number.isFinite(sequence) && sequence > 0 ? sequence : index + 1
}

const normalizedDuration = (
  event: AgentRunTimelineEvent,
  start?: string,
  finish?: string,
  nowMs = Date.now(),
) => {
  const supplied = Number(event.durationMs)
  if (Number.isFinite(supplied) && supplied >= 0) return supplied
  const startedAt = dateMilliseconds(start)
  const finishedAt = dateMilliseconds(finish) || nowMs
  return startedAt > 0 && finishedAt >= startedAt ? finishedAt - startedAt : 0
}

const hasValue = (value: unknown) => {
  if (value === undefined || value === null || value === '') return false
  if (Array.isArray(value)) return value.length > 0
  if (typeof value === 'object') return Object.keys(value as Record<string, unknown>).length > 0
  return true
}

const parsePossibleJSON = (value: unknown) => {
  if (typeof value !== 'string') return value
  const trimmed = value.trim()
  if (!trimmed || !['{', '['].includes(trimmed[0])) return value
  try {
    return JSON.parse(trimmed)
  } catch {
    return value
  }
}

const eventPayload = (event: AgentRunTimelineEvent) => Object.fromEntries(
  Object.entries(event).filter(([key, value]) => !eventTechnicalKeys.has(key) && hasValue(value)),
)

const eventPhaseLabel = (event: AgentRunTimelineEvent) => {
  const phase = String(event.phase || event.stepType || '')
  return String(event.label || phaseLabels[phase] || '')
}

const eventLabel = (event: AgentRunTimelineEvent) => {
  const type = String(event.type || '')
  const phaseLabel = eventPhaseLabel(event)
  const toolName = String(event.toolCallName || '')
  if (type === 'AGENT_STEP_STARTED' && phaseLabel) return `开始${phaseLabel}`
  if (type === 'AGENT_STEP_FINISHED' && phaseLabel) return `完成${phaseLabel}`
  if (type === 'AGENT_STEP_ERROR' && phaseLabel) return `${phaseLabel}失败`
  if (type.startsWith('TOOL_CALL_') && toolName) return `${eventLabels[type] || '工具事件'} · ${toolName}`
  return eventLabels[type] || type || 'Agent 事件'
}

const compactEventText = (value: string, maxLength = 120) => {
  const compact = value.trim().replace(/\s+/g, ' ')
  return compact.length > maxLength ? `${compact.slice(0, maxLength)}…` : compact
}

const eventSummary = (event: AgentRunTimelineEvent) => {
  if (typeof event.message === 'string' && event.message.trim()) return event.message.trim()
  if (event.type !== 'TOOL_CALL_ARGS' && typeof event.delta === 'string' && event.delta.trim()) {
    return compactEventText(event.delta)
  }
  const quality = event.quality as Record<string, unknown> | undefined
  if (quality && hasValue(quality.score)) {
    return `评分 ${quality.score}${quality.passed === true ? '，已通过' : quality.passed === false ? '，需修订' : ''}`
  }
  const interaction = event.interaction as Record<string, unknown> | undefined
  if (Array.isArray(interaction?.questions)) return `${interaction.questions.length} 个问题等待确认`
  const content = event.content as Record<string, unknown> | undefined
  if (typeof content?.title === 'string') {
    const files = Array.isArray(content.files) ? ` · ${content.files.length} 个文件` : ''
    return `${content.title}${files}`
  }
  if (event.queueDurationMs) return `排队 ${formatAgentRunDuration(event.queueDurationMs)}`
  if (event.totalDurationMs) return `总耗时 ${formatAgentRunDuration(event.totalDurationMs)}`
  return ''
}

const eventStatus = (
  event: AgentRunTimelineEvent,
  completedStepIds: Set<string>,
  completedToolIds: Set<string>,
  reasoningEnded: boolean,
) => {
  const type = String(event.type || '')
  if (type.includes('ERROR') || event.status === 'failed') return 'failed'
  if (type === 'AGENT_STEP_STARTED') {
    return completedStepIds.has(String(event.stepId || '')) ? 'succeeded' : 'running'
  }
  if (type === 'TOOL_CALL_START') {
    return completedToolIds.has(String(event.toolCallId || '')) ? 'succeeded' : 'running'
  }
  if (type === 'REASONING_START' || type === 'REASONING_MESSAGE_START') {
    return reasoningEnded ? 'succeeded' : 'running'
  }
  if (type === 'RUN_QUEUED' || type === 'RUN_STARTED') return 'succeeded'
  return 'succeeded'
}

const eventDetails = (
  event: AgentRunTimelineEvent,
  step?: AgentRunStepDetail,
): AgentRunEventDetail[] => {
  const type = String(event.type || '')
  const details: AgentRunEventDetail[] = []
  const add = (label: string, value: unknown, tone: AgentRunEventDetail['tone'] = 'default') => {
    if (hasValue(value)) details.push({ label, value: parsePossibleJSON(value), tone })
  }

  if (type === 'AGENT_STEP_STARTED') add('入参', step?.input)
  if (type === 'AGENT_STEP_FINISHED') add('返回', step?.output)
  if (type === 'AGENT_STEP_ERROR') {
    add('错误', step?.error || event.error || event.message, 'error')
  }
  if (type === 'TOOL_CALL_ARGS') add('入参', event.delta)
  if (type === 'TOOL_CALL_RESULT') add('返回', event.result ?? event.output ?? event.content)
  if (type === 'RUN_RESUMED') add('补充信息', event.answers)
  if (type === 'RUN_WAITING_INPUT') add('返回', event.interaction)
  if (type === 'QUALITY_UPDATED') add('返回', event.quality)
  if (type === 'ACTIVITY_SNAPSHOT' || type === 'ACTIVITY_DELTA') add('返回', event.content)
  if (type === 'TEXT_MESSAGE_CONTENT' || type === 'REASONING_MESSAGE_CONTENT') add('返回', event.delta)
  if (type === 'RUN_ERROR') add('错误', event.message || event.error, 'error')

  if (!details.length) add('事件数据', eventPayload(event))
  return details
}

export function buildAgentRunEventRows(
  events: AgentRunTimelineEvent[] = [],
  steps: AgentRunStepDetail[] = [],
): AgentRunEventRow[] {
  const orderedEvents = [...events]
    .map((event, index) => ({ event, order: eventSequence(event, index) }))
    .sort((left, right) => left.order - right.order)
  const stepById = new Map(steps.map((step) => [String(step.id || ''), step]))
  const completedStepIds = new Set(
    orderedEvents
      .filter(({ event }) => ['AGENT_STEP_FINISHED', 'AGENT_STEP_ERROR'].includes(String(event.type)))
      .map(({ event }) => String(event.stepId || '')),
  )
  const completedToolIds = new Set(
    orderedEvents
      .filter(({ event }) => ['TOOL_CALL_END', 'TOOL_CALL_RESULT'].includes(String(event.type)))
      .map(({ event }) => String(event.toolCallId || '')),
  )
  const reasoningEnded = orderedEvents.some(({ event }) => event.type === 'REASONING_END')
  const toolNames = new Map(
    orderedEvents
      .filter(({ event }) => event.type === 'TOOL_CALL_START' && event.toolCallId)
      .map(({ event }) => [String(event.toolCallId), String(event.toolCallName || '')]),
  )

  return orderedEvents.map(({ event, order }, index) => {
    const normalizedEvent = !event.toolCallName && event.toolCallId && toolNames.get(String(event.toolCallId))
      ? { ...event, toolCallName: toolNames.get(String(event.toolCallId)) }
      : event
    const step = stepById.get(String(normalizedEvent.stepId || ''))
    return {
      id: String(normalizedEvent.id || `${normalizedEvent.type || 'event'}-${order}-${index}`),
      type: String(normalizedEvent.type || 'EVENT'),
      label: eventLabel(normalizedEvent),
      summary: eventSummary(normalizedEvent),
      status: eventStatus(normalizedEvent, completedStepIds, completedToolIds, reasoningEnded),
      sequence: order,
      durationMs: Number(normalizedEvent.durationMs || normalizedEvent.totalDurationMs || 0),
      details: eventDetails(normalizedEvent, step),
    }
  })
}

export function formatAgentRunEventValue(value: unknown) {
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value, null, 2)
  } catch {
    return String(value)
  }
}

export function buildAgentRunTimeline(
  events: AgentRunTimelineEvent[] = [],
  run: Record<string, unknown> = {},
  nowMs = Date.now(),
) {
  const orderedEvents = [...events]
    .map((event, index) => ({ event, order: eventSequence(event, index) }))
    .sort((left, right) => left.order - right.order)
  const steps = new Map<string, AgentRunTimelineStep>()

  for (const { event, order } of orderedEvents) {
    if (!['AGENT_STEP_STARTED', 'AGENT_STEP_FINISHED', 'AGENT_STEP_ERROR'].includes(String(event.type))) {
      continue
    }
    const id = String(event.stepId || event.id || `${event.stepType || event.phase || 'step'}-${order}`)
    const existing = steps.get(id)
    const status = event.type === 'AGENT_STEP_ERROR'
      ? 'failed'
      : event.type === 'AGENT_STEP_FINISHED'
        ? 'succeeded'
        : 'running'
    const startedAt = event.startedAt || existing?.startedAt || event.createdAt
    const finishedAt = event.finishedAt || (status === 'running' ? undefined : event.createdAt)
    const phase = String(event.phase || existing?.phase || event.stepType || '')
    const stepType = String(event.stepType || existing?.stepType || phase)

    steps.set(id, {
      id,
      stepType,
      phase,
      label: String(event.label || existing?.label || phaseLabels[phase] || phaseLabels[stepType] || '执行步骤'),
      message: String(event.message || existing?.message || ''),
      status,
      startedAt,
      finishedAt,
      durationMs: normalizedDuration(event, startedAt, finishedAt, nowMs),
      sequence: existing?.sequence || order,
    })
  }

  const resultSteps = Array.from(steps.values())
    .sort((left, right) => left.sequence - right.sequence)
    .map((step) => ({
      ...step,
      durationMs: step.status === 'running'
        ? normalizedDuration({}, step.startedAt, undefined, nowMs)
        : step.durationMs,
    }))

  const finishEvent = [...orderedEvents]
    .reverse()
    .find(({ event }) => event.type === 'RUN_FINISHED' || event.type === 'RUN_ERROR' || event.type === 'RUN_CANCELLED')
    ?.event
  const suppliedTotal = Number(finishEvent?.totalDurationMs)
  const startedAt = String(run.started_at || run.startedAt || '')
  const finishedAt = String(run.finished_at || run.finishedAt || finishEvent?.createdAt || '')
  const totalDurationMs = Number.isFinite(suppliedTotal) && suppliedTotal >= 0
    ? suppliedTotal
    : normalizedDuration({}, startedAt, finishedAt, nowMs)
  const status = String(run.status || '')

  return {
    steps: resultSteps,
    totalDurationMs,
    active: !terminalRunStatuses.has(status),
    failedCount: resultSteps.filter((step) => step.status === 'failed').length,
    completedCount: resultSteps.filter((step) => step.status === 'succeeded').length,
  }
}

export function formatAgentRunDuration(durationMs: number) {
  const milliseconds = Math.max(0, Number(durationMs) || 0)
  if (milliseconds < 1000) return milliseconds > 0 ? '<1 秒' : ''
  const seconds = Math.round(milliseconds / 1000)
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  const rest = seconds % 60
  return rest > 0 ? `${minutes} 分 ${rest} 秒` : `${minutes} 分`
}

export function agentRunPhaseLabel(phase?: string) {
  return phaseLabels[String(phase || '')] || 'Agent 执行'
}
