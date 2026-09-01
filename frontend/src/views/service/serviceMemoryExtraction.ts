import type { OrganizeMemory } from '@/api/organize'

export type PriorityKey = 'high' | 'medium' | 'low'
export type ServiceTaskSource = 'memory'

export interface ServiceMemoryEvidence {
  id: string
  title: string
  summary: string
  sourceLabel: string
  occurredAtLabel: string
}

export interface ServiceTask {
  id: string
  sourceType: ServiceTaskSource
  customerName: string
  studentName: string
  title: string
  summary: string
  stage: string
  priorityKey: PriorityKey
  dueText: string
  channel: string
  decisionRole: string
  riskLabel: string
  assistReason: string
  primaryAction: string
  nextAction: string
  avoidAction: string
  contextItems: string[]
  memorySignals: string[]
  memoryEvidence: ServiceMemoryEvidence[]
  sourceMemoryIds: string[]
  sourceMemoryCount: number
  lastMemoryLabel: string
  confidenceLabel: string
  salesHighlights: string[]
  writeBackStatus: string
  writeBackDraft: string
  replyDraft: string
}

export const emptyServiceTask: ServiceTask = {
  id: '',
  sourceType: 'memory',
  customerName: '',
  studentName: '待补充',
  title: '',
  summary: '',
  stage: '客户跟进',
  priorityKey: 'low',
  dueText: '待定',
  channel: '记忆',
  decisionRole: '决策人待补充',
  riskLabel: '待判断',
  assistReason: '',
  primaryAction: '',
  nextAction: '',
  avoidAction: '',
  contextItems: [],
  memorySignals: [],
  memoryEvidence: [],
  sourceMemoryIds: [],
  sourceMemoryCount: 0,
  lastMemoryLabel: '',
  confidenceLabel: '待确认',
  salesHighlights: [],
  writeBackStatus: '',
  writeBackDraft: '',
  replyDraft: '',
}

const serviceMemoryKeywords = [
  '客户',
  '家长',
  '学员',
  '试听',
  '续费',
  '报名',
  '咨询',
  '价格',
  '顾虑',
  '异议',
  '跟进',
  '回访',
  '转介绍',
  '商机',
  '招生',
  '成交',
  '微信',
  '企微',
]

const serviceBusinessPrimaryKeywords = [
  '试听',
  '体验课',
  '公开课',
  '咨询',
  '到访',
  '邀约',
  '意向',
  '报名',
  '定金',
  '合同',
  '付款',
  '成交',
  '续费',
  '续课',
  '剩余课次',
  '到期',
  '跟进',
  '回访',
  '转介绍',
  '老带新',
  '活动邀请',
  '售后',
  '投诉',
  '反馈',
  '退费',
  '退款',
  '餐食',
  '午睡',
  '请假',
  '适应',
]

const serviceBusinessSupportingKeywords = [
  '价格',
  '费用',
  '学费',
  '优惠',
  '预算',
  '顾虑',
  '异议',
]

const serviceCustomerContextKeywords = [
  '客户',
  '家长',
  '学员',
  '学生',
  '孩子',
  '联系人',
  '试听',
  '咨询',
  '报名',
  '回访',
]

const serviceBusinessMetadataKeys = [
  'stage',
  'sales_stage',
  'salesStage',
  'service_stage',
  'serviceStage',
  'risk_label',
  'riskLabel',
  'risk',
  'concern',
  'next_action',
  'nextAction',
  'follow_up_action',
  'followUpAction',
  'next_follow_up_at',
  'nextFollowUpAt',
  'follow_up_at',
  'followUpAt',
  'due_at',
  'dueAt',
  'lead_status',
  'leadStatus',
  'deal_status',
  'dealStatus',
]

const asTrimmedString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const asStringList = (value: unknown) => {
  if (Array.isArray(value)) {
    return value.map((item) => asTrimmedString(item)).filter(Boolean)
  }
  const text = asTrimmedString(value)
  return text ? text.split(/[、,，\s]+/).map((item) => item.trim()).filter(Boolean) : []
}

const firstMetadataString = (metadata: Record<string, unknown>, keys: string[]) => {
  for (const key of keys) {
    const value = asTrimmedString(metadata[key])
    if (value) return value
  }
  return ''
}

const uniqueStrings = (items: string[]) => Array.from(new Set(items.map((item) => item.trim()).filter(Boolean)))

const stripMemoryMarkup = (value = '') => value
  .replace(/<script[\s\S]*?<\/script>/gi, ' ')
  .replace(/<style[\s\S]*?<\/style>/gi, ' ')
  .replace(/<[^>]+>/g, ' ')
  .replace(/&nbsp;/g, ' ')
  .replace(/&amp;/g, '&')
  .replace(/&lt;/g, '<')
  .replace(/&gt;/g, '>')
  .replace(/&quot;/g, '"')
  .replace(/&#39;/g, "'")
  .replace(/[#>*_`~\-\[\]()]/g, ' ')
  .replace(/\s+/g, ' ')
  .trim()

const contentExcerpt = (value: string, fallback = '暂无记忆内容') => {
  const text = stripMemoryMarkup(value)
  if (!text) return fallback
  return text.length > 86 ? `${text.slice(0, 86)}...` : text
}

const memoryTags = (memory: OrganizeMemory) => asStringList(memory.metadata?.tags)

export const memorySearchText = (memory: OrganizeMemory) => [
  memory.title,
  stripMemoryMarkup(memory.content),
  memory.source,
  memoryTags(memory).join(' '),
].join('\n')

const normalizeCustomerName = (value: string) => value.replace(/[，,。；;：:].*$/, '').trim()

export const extractCustomerName = (memory: OrganizeMemory) => {
  const metadata = memory.metadata || {}
  const fromMetadata = firstMetadataString(metadata, [
    'customer_name',
    'customerName',
    'parent_name',
    'parentName',
    'contact_name',
    'contactName',
    'lead_name',
    'leadName',
  ])
  if (fromMetadata) return normalizeCustomerName(fromMetadata)

  const text = memorySearchText(memory)
  const directMatch = text.match(/(?:客户|家长|联系人)[:：]\s*([^\s，,。；;\n]{2,18})/)
  if (directMatch?.[1]) return normalizeCustomerName(directMatch[1])

  const leadMatch = text.match(/线索[:：]\s*([^\s，,。；;\n]{2,18})/)
  if (leadMatch?.[1] && /家长|妈妈|爸爸|客户|联系人/.test(leadMatch[1])) {
    return normalizeCustomerName(leadMatch[1])
  }

  const suffixMatch = text.match(/([\u4e00-\u9fa5A-Za-z0-9]{1,12}(?:家长|妈妈|爸爸))/)
  return suffixMatch?.[1] ? normalizeCustomerName(suffixMatch[1]) : ''
}

const extractStudentName = (memory: OrganizeMemory, customerName: string) => {
  const metadata = memory.metadata || {}
  const fromMetadata = firstMetadataString(metadata, [
    'student_name',
    'studentName',
    'learner_name',
    'learnerName',
    'child_name',
    'childName',
  ])
  if (fromMetadata) return fromMetadata

  const text = memorySearchText(memory)
  const match = text.match(/(?:学员|学生|孩子)[:：]\s*([^\s，,。；;\n]{2,12})/)
  if (match?.[1]) return match[1]
  if (customerName.endsWith('家长')) return customerName.slice(0, -2)
  return '待补充'
}

const hasServiceBusinessSignal = (memory: OrganizeMemory, text = memorySearchText(memory)) => {
  const metadata = memory.metadata || {}
  const hasPrimarySignal = serviceBusinessPrimaryKeywords.some((keyword) => text.includes(keyword))
  const hasSupportingSignal = serviceBusinessSupportingKeywords.some((keyword) => text.includes(keyword))
  const hasCustomerContext = serviceCustomerContextKeywords.some((keyword) => text.includes(keyword))
  return Boolean(firstMetadataString(metadata, serviceBusinessMetadataKeys))
    || hasPrimarySignal
    || (hasSupportingSignal && hasCustomerContext)
}

export const isServiceMemory = (memory: OrganizeMemory) => {
  const text = memorySearchText(memory)
  return Boolean(extractCustomerName(memory)) && hasServiceBusinessSignal(memory, text)
}

const inferStage = (text: string) => {
  if (/续费|续课|剩余课次|到期/.test(text)) return '续费服务'
  if (/试听|到访|体验课|公开课|咨询/.test(text)) return '售前试听'
  if (/报名|定金|合同|成单|成交|付款/.test(text)) return '报名确认'
  if (/投诉|售后|餐食|午睡|请假|反馈|不满/.test(text)) return '在园服务'
  if (/转介绍|推荐|老带新|活动邀请/.test(text)) return '转介绍'
  return '客户跟进'
}

const inferChannel = (text: string) => {
  if (text.includes('企微')) return '企微'
  if (text.includes('微信')) return '微信'
  if (text.includes('电话')) return '电话'
  if (text.includes('面谈') || text.includes('到访')) return '线下'
  return '记忆'
}

const inferDecisionRole = (text: string) => {
  if (/妈妈|母亲/.test(text)) return '妈妈主沟通'
  if (/爸爸|父亲/.test(text)) return '爸爸主沟通'
  if (/父母|夫妻|双方/.test(text)) return '父母共同决策'
  return '决策人待补充'
}

const inferRiskLabel = (text: string) => {
  if (/投诉|不满|退款|退费|差评/.test(text)) return '售后风险'
  if (/价格|费用|优惠|太贵|预算/.test(text)) return '价格顾虑'
  if (/适应|焦虑|哭|不习惯/.test(text)) return '适应焦虑'
  if (/续费|续课|剩余课次|到期/.test(text)) return '续费窗口'
  if (/未回复|没回复|超过|逾期|拖延/.test(text)) return '未闭环'
  return '待判断'
}

const inferPriority = (text: string, riskLabel: string): PriorityKey => {
  if (/今天|上午|下午|今晚|投诉|不满|退款|退费|差评|未回复|逾期|48\s*小时/.test(text)) return 'high'
  if (riskLabel !== '待判断' || /本周|周五|续费|试听|报名/.test(text)) return 'medium'
  return 'low'
}

const inferNextAction = (stage: string, riskLabel: string) => {
  if (riskLabel === '售后风险') return '先确认处理结果并补齐服务闭环'
  if (riskLabel === '价格顾虑') return '准备价值证明后再回应价格问题'
  if (riskLabel === '适应焦虑') return '补充孩子观察记录并安排低压力回访'
  if (stage === '续费服务') return '生成阶段成长回顾后再进入续费沟通'
  if (stage === '售前试听') return '完成试听后回访并确认下一步安排'
  if (stage === '转介绍') return '用活动资料做一次轻触达'
  return '确认客户状态并补一条下一步记忆'
}

const inferPrimaryAction = (stage: string, riskLabel: string) => {
  if (riskLabel === '售后风险') return '先回应家长感受和处理进展，再补老师观察，不展开长解释。'
  if (riskLabel === '价格顾虑') return '先把课程价值和孩子变化讲清楚，再讨论价格或优惠边界。'
  if (riskLabel === '适应焦虑') return '先给到具体观察，再说明适应节奏，最后确认是否需要老师补充沟通。'
  if (stage === '续费服务') return '先整理阶段变化和下一阶段目标，再约一次低压力沟通。'
  return '先确认记忆里的真实事实，再生成可发送话术和下一步。'
}

const inferAvoidAction = (riskLabel: string) => {
  if (riskLabel === '价格顾虑') return '不要先抛优惠或承诺结果，避免把沟通变成纯价格谈判。'
  if (riskLabel === '售后风险') return '不要先解释原因或转移责任，先确认问题是否真正闭环。'
  if (riskLabel === '适应焦虑') return '不要承诺马上适应，也不要用泛泛安慰替代具体观察。'
  return '不要把未确认的梳理结果直接当作客户事实落地。'
}

const inferReplyDraft = (customerName: string, stage: string, riskLabel: string) => {
  if (riskLabel === '价格顾虑') {
    return `您好，${customerName}这边我先把孩子目前的学习变化和后续安排梳理一下，再跟您说明费用和可选方案，方便您一起判断是否合适。`
  }
  if (riskLabel === '售后风险') {
    return `您好，之前反馈的问题我们已经重点跟进。我想先跟您确认这两天的改善感受，再把老师观察到的情况同步给您。`
  }
  if (stage === '续费服务') {
    return `这段时间孩子有几处比较明确的变化，我先整理成阶段回顾发您，也想听听您对下一阶段最关注的目标。`
  }
  return `您好，我根据最近记录把当前情况整理了一下，想跟您确认一个下一步安排，避免遗漏您的重点关注。`
}

const deriveMemorySignals = (text: string, tags: string[]) => {
  const matchedKeywords = serviceMemoryKeywords.filter((keyword) => text.includes(keyword))
  return uniqueStrings([...tags, ...matchedKeywords]).slice(0, 6)
}

export const formatMemoryDateLabel = (value?: string) => {
  const raw = asTrimmedString(value)
  if (!raw) return '最近'
  const date = new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const formatDueText = (value: string, priorityKey: PriorityKey) => {
  const raw = asTrimmedString(value)
  if (raw) {
    const date = new Date(raw)
    if (!Number.isNaN(date.getTime())) {
      const hours = String(date.getHours()).padStart(2, '0')
      const minutes = String(date.getMinutes()).padStart(2, '0')
      return `${date.getMonth() + 1}月${date.getDate()}日 ${hours}:${minutes}`
    }
    return raw
  }
  if (priorityKey === 'high') return '今天'
  if (priorityKey === 'medium') return '本周'
  return '待定'
}

const memoryEvidenceFrom = (memory: OrganizeMemory): ServiceMemoryEvidence => ({
  id: memory.id,
  title: memory.title || '未命名记忆',
  summary: contentExcerpt(memory.content || asTrimmedString(memory.metadata?.summary)),
  sourceLabel: memory.source || '个人记忆',
  occurredAtLabel: formatMemoryDateLabel(memory.occurred_at || memory.updated_at),
})

const salesHighlightsFrom = (stage: string, riskLabel: string, text: string) => {
  const highlights: string[] = []
  if (/试听|到访|体验课|公开课|咨询/.test(text) || stage === '售前试听') {
    highlights.push('出现售前接触信号，适合尽快回访并确认下一步安排。')
  }
  if (riskLabel === '价格顾虑') {
    highlights.push('客户对价格敏感，先补价值证明和孩子变化，再谈费用。')
  }
  if (riskLabel === '适应焦虑') {
    highlights.push('客户关注适应情况，用具体观察降低不确定感。')
  }
  if (riskLabel === '续费窗口' || stage === '续费服务') {
    highlights.push('已经进入续费窗口，先做阶段成长回顾再推进判断。')
  }
  if (riskLabel === '售后风险' || stage === '在园服务') {
    highlights.push('存在售后或服务反馈，先闭环处理结果再继续后续沟通。')
  }
  if (/转介绍|推荐|老带新|活动邀请/.test(text) || stage === '转介绍') {
    highlights.push('适合用活动资料轻触达，不直接索要转介绍。')
  }
  if (/报名|定金|合同|付款|成交/.test(text) || stage === '报名确认') {
    highlights.push('出现报名确认信号，优先补齐决策人、时间和付款/合同状态。')
  }
  if (!highlights.length) {
    highlights.push('已有客户服务信息，先补齐客户状态、决策人和下一步。')
  }
  return uniqueStrings(highlights).slice(0, 4)
}

const buildMemoryServiceTask = (customerName: string, memories: OrganizeMemory[], index: number): ServiceTask => {
  const latest = memories[0]!
  const metadata = latest.metadata || {}
  const text = memories.map(memorySearchText).join('\n')
  const tags = uniqueStrings(memories.flatMap(memoryTags))
  const stage = firstMetadataString(metadata, ['stage', 'sales_stage', 'salesStage', 'service_stage', 'serviceStage']) || inferStage(text)
  const riskLabel = firstMetadataString(metadata, ['risk_label', 'riskLabel', 'risk', 'concern']) || inferRiskLabel(text)
  const priorityKey = inferPriority(text, riskLabel)
  const nextAction = firstMetadataString(metadata, ['next_action', 'nextAction', 'follow_up_action', 'followUpAction']) || inferNextAction(stage, riskLabel)
  const studentName = extractStudentName(latest, customerName)
  const channel = firstMetadataString(metadata, ['channel', 'source_channel', 'sourceChannel']) || inferChannel(text)
  const decisionRole = firstMetadataString(metadata, ['decision_role', 'decisionRole', 'decision_maker', 'decisionMaker']) || inferDecisionRole(text)
  const memoryEvidence = memories.slice(0, 3).map(memoryEvidenceFrom)
  const memorySignals = deriveMemorySignals(text, tags.length ? tags : ['个人记忆'])
  const summary = firstMetadataString(metadata, ['summary', 'source_summary', 'description', 'abstract']) || contentExcerpt(latest.content)
  const primaryAction = firstMetadataString(metadata, ['primary_action', 'primaryAction']) || inferPrimaryAction(stage, riskLabel)
  const avoidAction = firstMetadataString(metadata, ['avoid_action', 'avoidAction']) || inferAvoidAction(riskLabel)
  const replyDraft = firstMetadataString(metadata, ['reply_draft', 'replyDraft', 'draft']) || inferReplyDraft(customerName, stage, riskLabel)
  const dueText = formatDueText(
    firstMetadataString(metadata, ['next_follow_up_at', 'nextFollowUpAt', 'follow_up_at', 'followUpAt', 'due_at', 'dueAt', 'due_text', 'dueText']),
    priorityKey,
  )
  const sourceMemoryIds = memories.map((memory) => memory.id).filter(Boolean)
  const sourceMemoryCount = sourceMemoryIds.length || memories.length

  return {
    id: `memory-task-${customerName || index}`,
    sourceType: 'memory',
    customerName,
    studentName,
    title: latest.title || `${stage}线索`,
    summary,
    stage,
    priorityKey,
    dueText,
    channel,
    decisionRole,
    riskLabel,
    assistReason: `从 ${sourceMemoryCount} 条客户记忆抽到：${summary}`,
    primaryAction,
    nextAction,
    avoidAction,
    contextItems: uniqueStrings(['个人记忆', ...memorySignals]),
    memorySignals,
    memoryEvidence,
    sourceMemoryIds,
    sourceMemoryCount,
    lastMemoryLabel: formatMemoryDateLabel(latest.occurred_at || latest.updated_at),
    confidenceLabel: sourceMemoryCount > 1 ? '较高' : '待确认',
    salesHighlights: salesHighlightsFrom(stage, riskLabel, text),
    writeBackStatus: '待确认',
    writeBackDraft: `${customerName}｜${nextAction}。依据：${memoryEvidence[0]?.title || '最近记忆'}`,
    replyDraft,
  }
}

export const buildServiceTasksFromMemories = (memories: OrganizeMemory[]) => {
  const buckets = new Map<string, OrganizeMemory[]>()
  memories.forEach((memory) => {
    if (!isServiceMemory(memory)) return
    const customerName = extractCustomerName(memory)
    if (!customerName) return
    const current = buckets.get(customerName) || []
    current.push(memory)
    buckets.set(customerName, current)
  })

  return Array.from(buckets.entries())
    .map(([customerName, groupedMemories], index) => buildMemoryServiceTask(customerName, groupedMemories, index))
    .sort((a, b) => {
      const order: Record<PriorityKey, number> = { high: 0, medium: 1, low: 2 }
      return order[a.priorityKey] - order[b.priorityKey]
    })
    .slice(0, 12)
}

export const buildServiceTaskFromMemory = (memory: OrganizeMemory) => {
  if (!isServiceMemory(memory)) return null
  return buildMemoryServiceTask(extractCustomerName(memory), [memory], 0)
}
