import type { OrganizeMemory } from '@/api/organize'

export type PriorityKey = 'high' | 'medium' | 'low'
export type ServiceTaskSource = 'memory' | 'demo'

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

export const serviceDemoTasks: ServiceTask[] = [
  {
    id: 'demo-lead-trial-001',
    sourceType: 'demo',
    customerName: '林女士',
    studentName: '林一一',
    title: '新线索咨询自然拼读试听，需要今天约课',
    summary: '客户: 林女士\n学员: 林一一\n渠道: 小红书私信\n阶段: 售前咨询\n家长关注自然拼读启蒙，希望先看一节试听课，能接受周三或周六下午。',
    stage: '售前咨询',
    priorityKey: 'high',
    dueText: '今天 15:00',
    channel: '小红书私信',
    decisionRole: '妈妈主沟通',
    riskLabel: '试听未约定',
    assistReason: '演示数据：客户已经表达试听意向，但还没有确认具体时间，适合今天完成约课。',
    primaryAction: '先确认孩子年龄和可上课时间，再给出两个试听时段。',
    nextAction: '发送试听时段并确认到店时间',
    avoidAction: '不要直接催报名，也不要一次性发送过多课程介绍。',
    contextItems: ['4 岁半', '自然拼读', '周三/周六下午可约'],
    memorySignals: ['新线索', '试听意向', '时间待确认'],
    memoryEvidence: [
      {
        id: 'demo-memory-lead-trial-001',
        title: '林女士咨询自然拼读试听',
        summary: '林女士从小红书私信咨询自然拼读，孩子 4 岁半，想先体验试听课。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 10:20',
      },
      {
        id: 'demo-memory-lead-trial-002',
        title: '林女士补充可约时间',
        summary: '家长表示周三和周六下午比较方便，希望课程顾问给两个可选时段。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 11:05',
      },
    ],
    sourceMemoryIds: ['demo-memory-lead-trial-001', 'demo-memory-lead-trial-002'],
    sourceMemoryCount: 2,
    lastMemoryLabel: '今天 11:05',
    confidenceLabel: '演示',
    salesHighlights: [
      '家长有明确试听意向，先完成时段确认。',
      '需要补充孩子年龄、校区和上课偏好。',
    ],
    writeBackStatus: '待确认',
    writeBackDraft: '林女士｜发送试听时段并确认到店时间。依据：家长表达自然拼读试听意向，周三/周六下午可约。',
    replyDraft: '您好，我看您这边主要想先了解自然拼读试听。我先帮您预留两个可选时间：周三下午和周六下午，您看哪个更方便？确认后我再把试听准备事项发您。',
  },
  {
    id: 'demo-customer-growth-001',
    sourceType: 'demo',
    customerName: '陈屿妈妈',
    studentName: '陈屿',
    title: '在读客户进入续费窗口，先整理成长回顾',
    summary: '客户: 陈屿妈妈\n学员: 陈屿\n渠道: 企微\n阶段: 续费服务\n近 4 次课堂反馈稳定，表达主动性提升，剩余课次进入续费提醒窗口。',
    stage: '续费服务',
    priorityKey: 'medium',
    dueText: '明天 19:30',
    channel: '企微',
    decisionRole: '妈妈主沟通',
    riskLabel: '续费窗口',
    assistReason: '演示数据：客户已经进入续费窗口，但适合先用孩子变化打开沟通。',
    primaryAction: '先生成阶段成长回顾，再预约一次低压力沟通。',
    nextAction: '生成阶段成长回顾后再进入续费沟通',
    avoidAction: '不要第一句话提醒课次不足，避免让家长感到被推进。',
    contextItems: ['剩余课次 6 次', '表达主动性提升', '家长关注课堂参与'],
    memorySignals: ['续费窗口', '成长记录', '在读服务'],
    memoryEvidence: [
      {
        id: 'demo-memory-growth-001',
        title: '陈屿近四次课堂反馈',
        summary: '陈屿课堂参与更主动，能够主动回答老师问题，但复述完整句仍需要提示。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '昨天 18:10',
      },
      {
        id: 'demo-memory-growth-002',
        title: '陈屿妈妈关注表达变化',
        summary: '家长提到孩子在家愿意模仿英文句子，希望了解下一阶段学习目标。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 09:15',
      },
    ],
    sourceMemoryIds: ['demo-memory-growth-001', 'demo-memory-growth-002'],
    sourceMemoryCount: 2,
    lastMemoryLabel: '今天 09:15',
    confidenceLabel: '演示',
    salesHighlights: [
      '先呈现阶段变化，再讨论下一阶段安排。',
      '把续费动作包装成学习规划沟通。',
    ],
    writeBackStatus: '待确认',
    writeBackDraft: '陈屿妈妈｜生成阶段成长回顾后再进入续费沟通。依据：剩余课次进入提醒窗口，家长关注表达变化。',
    replyDraft: '陈屿妈妈您好，我整理了一下陈屿最近几次课堂变化，他现在主动表达明显比之前多了。想和您约 10 分钟，把当前进展和下一阶段目标一起过一下，您明晚方便吗？',
  },
  {
    id: 'demo-schedule-makeup-001',
    sourceType: 'demo',
    customerName: '刘爸爸',
    studentName: '刘念',
    title: '请假后补课时间未确认，需要排课提醒',
    summary: '客户: 刘爸爸\n学员: 刘念\n渠道: 电话\n阶段: 排课调课\n上周因感冒请假，家长希望本周补课，但还没有确认老师和教室资源。',
    stage: '排课调课',
    priorityKey: 'medium',
    dueText: '今天 17:30',
    channel: '电话',
    decisionRole: '爸爸主沟通',
    riskLabel: '排课未闭环',
    assistReason: '演示数据：请假补课已经产生服务承诺，需要在今天确认可选时段。',
    primaryAction: '先查老师空闲，再给家长两个补课时段。',
    nextAction: '确认补课老师和时段后回复家长',
    avoidAction: '不要先答应固定时间，避免和实际排课资源冲突。',
    contextItems: ['上周请假', '本周补课', '老师资源待查'],
    memorySignals: ['请假', '补课', '排课'],
    memoryEvidence: [
      {
        id: 'demo-memory-schedule-001',
        title: '刘念上周请假',
        summary: '刘爸爸电话请假，说明孩子感冒，本周恢复后想尽快补课。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '8月30日',
      },
      {
        id: 'demo-memory-schedule-002',
        title: '刘爸爸追问补课安排',
        summary: '家长希望本周内完成补课，方便的话优先安排原老师。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 14:10',
      },
    ],
    sourceMemoryIds: ['demo-memory-schedule-001', 'demo-memory-schedule-002'],
    sourceMemoryCount: 2,
    lastMemoryLabel: '今天 14:10',
    confidenceLabel: '演示',
    salesHighlights: [
      '排课动作需要确认老师和教室，不应直接承诺固定时间。',
      '家长已有追问，今天需要给出明确反馈。',
    ],
    writeBackStatus: '待确认',
    writeBackDraft: '刘爸爸｜确认补课老师和时段后回复家长。依据：请假后补课需求未闭环，家长今天再次追问。',
    replyDraft: '刘爸爸您好，我先帮刘念看一下本周原老师的可补课时段，确认好教室和老师后给您两个选择。今天 17:30 前我回复您，避免时间来回调整。',
  },
  {
    id: 'demo-risk-after-sale-001',
    sourceType: 'demo',
    customerName: '赵女士',
    studentName: '赵小安',
    title: '家长对请假扣课不满，需要先做服务闭环',
    summary: '客户: 赵女士\n学员: 赵小安\n渠道: 企微\n阶段: 在园服务\n家长反馈上次请假扣课规则没有提前说明，情绪偏强，需要先确认规则和处理方案。',
    stage: '在园服务',
    priorityKey: 'high',
    dueText: '今天 18:00',
    channel: '企微',
    decisionRole: '妈妈主沟通',
    riskLabel: '售后风险',
    assistReason: '演示数据：家长已经表达不满，直接解释规则容易升级，需要先回应感受并确认处理边界。',
    primaryAction: '先核对请假记录和规则说明，再给出可执行处理方案。',
    nextAction: '确认请假记录后给出处理方案',
    avoidAction: '不要直接说系统规则就是这样，也不要在事实未核对前承诺补偿。',
    contextItems: ['请假扣课', '规则说明争议', '情绪偏强'],
    memorySignals: ['售后风险', '未闭环', '请假'],
    memoryEvidence: [
      {
        id: 'demo-memory-risk-001',
        title: '赵女士反馈请假扣课不满',
        summary: '家长认为请假扣课规则没有提前说明，希望机构给一个说法。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 12:40',
      },
      {
        id: 'demo-memory-risk-002',
        title: '班主任记录待核对',
        summary: '班主任记得曾口头说明过请假规则，但没有找到完整文字记录。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '今天 13:05',
      },
    ],
    sourceMemoryIds: ['demo-memory-risk-001', 'demo-memory-risk-002'],
    sourceMemoryCount: 2,
    lastMemoryLabel: '今天 13:05',
    confidenceLabel: '演示',
    salesHighlights: [
      '先回应情绪，再核对事实和规则。',
      '需要记录处理结果，避免二次沟通无依据。',
    ],
    writeBackStatus: '待确认',
    writeBackDraft: '赵女士｜确认请假记录后给出处理方案。依据：家长对请假扣课不满，规则说明存在争议。',
    replyDraft: '赵女士您好，这件事让您感觉不清楚，我先跟您说声抱歉。我现在去核对上次请假记录和当时的规则说明，今天 18:00 前给您一个明确处理方案。',
  },
  {
    id: 'demo-lead-price-001',
    sourceType: 'demo',
    customerName: '周先生',
    studentName: '周可',
    title: '试听后有价格顾虑，需要补价值证明',
    summary: '客户: 周先生\n学员: 周可\n渠道: 到店咨询\n阶段: 售前试听\n孩子试听参与度高，家长认可老师，但认为费用比另一家机构高。',
    stage: '售前试听',
    priorityKey: 'medium',
    dueText: '明天 11:00',
    channel: '到店咨询',
    decisionRole: '爸爸主沟通',
    riskLabel: '价格顾虑',
    assistReason: '演示数据：家长认可试听体验但卡在价格，需要先补孩子变化和课程价值。',
    primaryAction: '准备试听观察记录和阶段目标，再回应价格差异。',
    nextAction: '发送试听观察记录和课程价值说明',
    avoidAction: '不要马上给优惠，先确认家长具体比较点。',
    contextItems: ['试听参与度高', '认可老师', '对比竞品价格'],
    memorySignals: ['试听', '价格顾虑', '价值证明'],
    memoryEvidence: [
      {
        id: 'demo-memory-price-001',
        title: '周可试听反馈',
        summary: '试听课参与度高，能够跟老师完成互动，家长对老师认可。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '昨天 16:30',
      },
      {
        id: 'demo-memory-price-002',
        title: '周先生比较价格',
        summary: '家长提到另一家机构价格更低，希望了解课程差异。',
        sourceLabel: '演示记忆笔记',
        occurredAtLabel: '昨天 18:00',
      },
    ],
    sourceMemoryIds: ['demo-memory-price-001', 'demo-memory-price-002'],
    sourceMemoryCount: 2,
    lastMemoryLabel: '昨天 18:00',
    confidenceLabel: '演示',
    salesHighlights: [
      '先讲孩子试听表现和阶段目标，再处理价格比较。',
      '可调用公共知识库里的课程体系和成果案例。',
    ],
    writeBackStatus: '待确认',
    writeBackDraft: '周先生｜发送试听观察记录和课程价值说明。依据：试听认可但有价格顾虑。',
    replyDraft: '周先生您好，我把周可昨天试听时的表现和接下来适合他的学习目标整理了一下。价格这块我也会结合课程安排一起说明，方便您判断差异不只是单节课费用。',
  },
]

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
