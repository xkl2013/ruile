import { computed, reactive } from 'vue'

export type ServiceHubMode = 'data' | 'empty' | 'archived'
export type ServiceRole = '拥有者' | '管理员' | '编辑者' | '只读'

export interface ServiceTemplate {
  id: string
  name: string
  description: string
  instruction: string
  experts: string[]
  icon: string
}

export interface ServiceRecord {
  id: string
  name: string
  description: string
  templateId: string
  expertIds?: string[]
  knowledgeBaseIds?: string[]
  skillIds?: string[]
  role: ServiceRole
  members: number
  updatedLabel: string
  updatedAt: number
  state: 'active' | 'archived' | 'draft'
}

export interface ServiceSessionMessage {
  id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  state?: '已完成' | '进行中'
  steps?: string[]
  artifactId?: string
}

export interface ServiceSession {
  id: string
  chatSessionId?: string
  serviceId: string
  title: string
  expert: string
  pinned: boolean
  updatedLabel: string
  messages: ServiceSessionMessage[]
}

export interface ServiceArtifact {
  id: string
  title: string
  format: string
  version: string
  meta: string
  preview: string
}

export const serviceTemplates: ServiceTemplate[] = [
  {
    id: 't1',
    name: '招生咨询全流程',
    description: '从线索进入、跟进到转化，含话术与结果归档',
    instruction: '跟进 2026 秋季招生线索，语气亲切不催促，重点记录家长顾虑与到访意向；涉及课程与价格时先查知识库再回答。',
    experts: ['招生顾问', '教务主管'],
    icon: 'usergroup',
  },
  {
    id: 't2',
    name: '家长续费跟进',
    description: '续费窗口跟进、沟通话术与结果归档',
    instruction: '在续费窗口期跟进家长，说明续费政策与优惠，记录家长反馈与决策结果。',
    experts: ['教务主管'],
    icon: 'refresh',
  },
  {
    id: 't3',
    name: '一日流程巡查',
    description: '巡班记录、异常上报与每日汇总',
    instruction: '按一日流程巡查各班，记录异常并当日汇总上报。',
    experts: ['带班老师', '保育员'],
    icon: 'browse',
  },
  {
    id: 't4',
    name: '新教师带教',
    description: '带教计划、阶段评估与反馈沉淀',
    instruction: '为新教师制定带教计划，按阶段评估并沉淀反馈。',
    experts: ['教务主管', '带班老师'],
    icon: 'user',
  },
  {
    id: 't5',
    name: '活动筹备',
    description: '排期、通知与活动后复盘',
    instruction: '筹备活动：排期、通知、物资准备与活动后复盘。',
    experts: ['后勤主任'],
    icon: 'calendar',
  },
  {
    id: 't6',
    name: '教学教研沉淀',
    description: '观察记录、教研资料与经验沉淀',
    instruction: '整理课堂观察记录与教研资料，形成可复用的经验条目。',
    experts: ['带班老师'],
    icon: 'book-open',
  },
]

export const serviceExperts = [
  { id: 'e1', name: '招生顾问', description: '线索跟进、家长沟通与转化建议' },
  { id: 'e2', name: '教务主管', description: '排课协调、续费窗口与教务合规' },
  { id: 'e3', name: '带班老师', description: '一日流程巡查、记录整理与异常上报' },
  { id: 'e4', name: '保育员', description: '生活照料记录与卫生保健' },
  { id: 'e5', name: '后勤主任', description: '安全巡查、物资与场地安排' },
]

export const serviceKnowledgeBases = [
  { id: 'kb1', name: '招生话术与政策库', meta: '32 份 · 更新于 9-18' },
  { id: 'kb2', name: '家长沟通记录库', meta: '128 条 · 更新于今天' },
  { id: 'kb3', name: '课程与价格资料', meta: '18 份 · 更新于 9-12' },
  { id: 'kb4', name: '一日流程与安全制度', meta: '24 份 · 更新于 8-30' },
  { id: 'kb5', name: '教研资料库', meta: '56 份 · 更新于 9-20' },
]

export const serviceSkills = [
  { id: 'sk1', name: '引用生成器', description: '给知识库检索结果标注出处' },
  { id: 'sk2', name: '数据处理器', description: '统计分析、格式转换、生成报告' },
  { id: 'sk3', name: '文档协作', description: '按结构化流程一起写文档' },
  { id: 'sk4', name: '文档分析器', description: '拆解文档结构、提取关键信息' },
]

const now = Date.now()

const initialServices: ServiceRecord[] = [
  {
    id: 's1',
    name: '秋季招生咨询',
    description: '从线索进入、跟进到转化，覆盖 2026 秋季招生全流程。',
    templateId: 't1',
    expertIds: ['e1', 'e2'],
    knowledgeBaseIds: ['kb1', 'kb2', 'kb3'],
    skillIds: ['sk1', 'sk2'],
    role: '管理员',
    members: 5,
    updatedLabel: '2 小时前',
    updatedAt: now - 2 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's2',
    name: '家长续费跟进',
    description: '到期前后与家长沟通续费，记录顾虑与结果。',
    templateId: 't2',
    expertIds: ['e2'],
    knowledgeBaseIds: ['kb2', 'kb3'],
    skillIds: ['sk1'],
    role: '编辑者',
    members: 3,
    updatedLabel: '昨天',
    updatedAt: now - 26 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's3',
    name: '晨检异常分析',
    description: '汇总每日晨检异常并跟踪随访，形成周报。',
    templateId: 't3',
    expertIds: ['e3', 'e4'],
    knowledgeBaseIds: ['kb4'],
    skillIds: ['sk2'],
    role: '只读',
    members: 2,
    updatedLabel: '3 天前',
    updatedAt: now - 3 * 24 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's4',
    name: '新教师带教',
    description: '新入职教师的第一周带教计划、观察与反馈。',
    templateId: 't4',
    expertIds: ['e2', 'e3'],
    knowledgeBaseIds: ['kb4', 'kb5'],
    skillIds: ['sk3'],
    role: '拥有者',
    members: 4,
    updatedLabel: '上周',
    updatedAt: now - 7 * 24 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's5',
    name: '期末活动筹备',
    description: '期末汇演的场地、物资、节目与家长通知。',
    templateId: 't5',
    expertIds: ['e5'],
    knowledgeBaseIds: ['kb4'],
    skillIds: ['sk2', 'sk3'],
    role: '编辑者',
    members: 6,
    updatedLabel: '1 个月前',
    updatedAt: now - 30 * 24 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's6',
    name: '课堂观察记录',
    description: '日常听课观察记录与教研素材沉淀。',
    templateId: 't6',
    expertIds: ['e3'],
    knowledgeBaseIds: ['kb5'],
    skillIds: ['sk3', 'sk4'],
    role: '管理员',
    members: 3,
    updatedLabel: '上周',
    updatedAt: now - 8 * 24 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's7',
    name: '家长满意度回访',
    description: '',
    templateId: '',
    expertIds: [],
    knowledgeBaseIds: [],
    skillIds: [],
    role: '拥有者',
    members: 2,
    updatedLabel: '2 周前',
    updatedAt: now - 14 * 24 * 60 * 60 * 1000,
    state: 'active',
  },
  {
    id: 's8',
    name: '教研资料沉淀',
    description: '把教研会上的结论与案例整理成可复用的资料。',
    templateId: 't6',
    expertIds: ['e3'],
    knowledgeBaseIds: ['kb5'],
    skillIds: ['sk3', 'sk4'],
    role: '管理员',
    members: 4,
    updatedLabel: '昨天',
    updatedAt: now - 20 * 60 * 60 * 1000,
    state: 'active',
  },
]

const initialSessions: ServiceSession[] = [
  {
    id: 'c1',
    serviceId: 's1',
    title: '秋季招生线索跟进',
    expert: '招生顾问',
    pinned: true,
    updatedLabel: '3 小时前',
    messages: [
      { id: 'm1', role: 'user', content: '把 9 月秋季招生线索按到访意向排个序，列成表。' },
      {
        id: 'm2',
        role: 'assistant',
        content: '共 12 条线索：到访意向明确 5 条、待跟进 7 条。东苑分园的线索尚未并入。',
        state: '已完成',
        steps: ['正在读取相关资料', '正在合并线索台账与到访记录', '正在按意向等级排序', '正在生成表格'],
        artifactId: 'a1',
      },
      { id: 'm3', role: 'user', content: '把东苑分园那 4 条也并进来，按意向重新排。' },
      {
        id: 'm4',
        role: 'assistant',
        content: '已并入东苑分园 4 条，去重后共 16 条：到访意向明确 7 条、待跟进 9 条。',
        state: '已完成',
        steps: ['正在读取东苑分园线索', '正在合并并去重', '正在重新排序', '正在更新表格产物'],
        artifactId: 'a2',
      },
    ],
  },
  {
    id: 'c2',
    serviceId: 's1',
    title: '家长顾虑类型归纳',
    expert: '招生顾问',
    pinned: false,
    updatedLabel: '昨天',
    messages: [
      { id: 'm6', role: 'user', content: '这周家长问得最多的顾虑是什么？' },
      {
        id: 'm7',
        role: 'assistant',
        content: '23 条沟通记录中，顾虑集中在「上课时间冲突」9 条、「价格与优惠」7 条、「接送距离」4 条。',
        state: '已完成',
        steps: ['正在读取本周沟通记录', '正在归类家长顾虑', '正在归纳结论'],
        artifactId: 'a3',
      },
    ],
  },
  {
    id: 'c3',
    serviceId: 's1',
    title: '开放日到访名单整理',
    expert: '教务主管',
    pinned: false,
    updatedLabel: '3 天前',
    messages: [
      { id: 'm8', role: 'user', content: '把开放日预约的家长名单整理成一张表，标出已确认到访的。' },
      {
        id: 'm9',
        role: 'assistant',
        content: '预约 28 组，已确认到访 19 组、待确认 9 组；小班时段已满。',
        state: '已完成',
        steps: ['正在读取预约记录', '正在核对确认状态', '正在生成表格'],
      },
    ],
  },
  {
    id: 'c4',
    serviceId: 's2',
    title: '本月到期未续费清单',
    expert: '教务主管',
    pinned: false,
    updatedLabel: '5 小时前',
    messages: [
      { id: 'm10', role: 'user', content: '本月到期的孩子里，哪些家长还没续费？' },
      {
        id: 'm11',
        role: 'assistant',
        content: '到期 17 人，未续费 9 人，其中已联系 3 人、未联系 6 人。',
        state: '已完成',
        steps: ['正在核对到期名单', '正在比对缴费记录', '正在生成清单'],
        artifactId: 'a4',
      },
    ],
  },
  {
    id: 'c5',
    serviceId: 's3',
    title: '本周晨检异常汇总',
    expert: '带班老师',
    pinned: false,
    updatedLabel: '2 天前',
    messages: [
      { id: 'm12', role: 'user', content: '把本周的晨检异常汇总一下，要能直接给保健室看。' },
      {
        id: 'm13',
        role: 'assistant',
        content: '共 6 例异常：发热 3 例、皮疹 2 例、口腔异常 1 例，均已通知家长并记录随访。',
        state: '已完成',
        steps: ['正在读取晨检记录', '正在归并同类异常', '正在生成汇总'],
        artifactId: 'a5',
      },
    ],
  },
  {
    id: 'c6',
    serviceId: 's4',
    title: '新教师第一周带教计划',
    expert: '带班老师',
    pinned: false,
    updatedLabel: '6 天前',
    messages: [
      { id: 'm14', role: 'user', content: '给这学期新来的 3 位老师排一份第一周带教计划。' },
      {
        id: 'm15',
        role: 'assistant',
        content: '3 位新教师按班级分组，第一周覆盖一日流程观摩、家长沟通跟岗、班级常规建立三块。',
        state: '已完成',
        steps: ['正在读取带教模板', '正在按岗位拆分任务', '正在排期'],
      },
    ],
  },
  {
    id: 'c7',
    serviceId: 's5',
    title: '元旦汇演节目排期',
    expert: '后勤主任',
    pinned: false,
    updatedLabel: '9 天前',
    messages: [
      { id: 'm16', role: 'user', content: '元旦汇演各班节目怎么排？场地只有一天。' },
      {
        id: 'm17',
        role: 'assistant',
        content: '共 11 个节目，按大班→中班→小班三段排；上午彩排、下午正式演出。',
        state: '已完成',
        steps: ['正在读取各班报名', '正在核对场地时段', '正在排期'],
      },
    ],
  },
  {
    id: 'c8',
    serviceId: 's8',
    title: '11 月课堂观察记录归档',
    expert: '带班老师',
    pinned: false,
    updatedLabel: '昨天',
    messages: [
      { id: 'm18', role: 'user', content: '把 11 月的课堂观察记录整理归档，顺带看看有没有值得沉淀的经验。' },
      {
        id: 'm19',
        role: 'assistant',
        content: '共 24 份记录，归为 4 类：师幼互动 9、区域活动 7、户外运动 5、过渡环节 3。',
        state: '已完成',
        steps: ['正在读取观察记录', '正在按主题归类', '正在抽取可复用经验'],
      },
    ],
  },
]

export const serviceArtifacts: Record<string, ServiceArtifact> = {
  a1: {
    id: 'a1',
    title: '秋季招生线索跟进表.xlsx',
    format: 'XLSX',
    version: 'v1',
    meta: '表格 · v1 · 12 条 · 18 KB',
    preview: '秋季招生线索跟进表（v1）\n\n乐乐　明确　待跟进\n朵朵　待观察　已沟通\n小宝　明确　待跟进\n\n共 12 条线索',
  },
  a2: {
    id: 'a2',
    title: '秋季招生线索跟进表.xlsx',
    format: 'XLSX',
    version: 'v2',
    meta: '表格 · v2（已更新）· 16 条 · 22 KB',
    preview: '秋季招生线索跟进表（v2）\n\n乐乐　明确　待跟进\n朵朵　待观察　已沟通\n小宝　明确　待跟进\n东苑 · 果果　明确　待跟进\n东苑 · 米米　待观察　待跟进\n\n共 16 条线索',
  },
  a3: {
    id: 'a3',
    title: '家长顾虑归纳.md',
    format: 'MD',
    version: 'v1',
    meta: '文档 · v1 · 3 KB',
    preview: '本周家长顾虑归纳\n\n1. 上课时间冲突 —— 9 条\n2. 价格与优惠 —— 7 条\n3. 接送距离 —— 4 条\n4. 其他 —— 3 条',
  },
  a4: {
    id: 'a4',
    title: '到期未续费家长清单.xlsx',
    format: 'XLSX',
    version: 'v1',
    meta: '表格 · v1 · 9 条 · 14 KB',
    preview: '本月到期未续费家长清单\n\n阳阳　9/18　已联系\n糖糖　9/20　未联系\n豆豆　9/25　未联系\n\n共 9 条，其中 3 条已联系',
  },
  a5: {
    id: 'a5',
    title: '本周晨检异常汇总.md',
    format: 'MD',
    version: 'v1',
    meta: '文档 · v1 · 4 KB',
    preview: '本周晨检异常汇总\n\n共 6 例异常：发热 3 例、皮疹 2 例、口腔异常 1 例。\n均已通知家长并记录随访结果。',
  },
}

export const serviceHubState = reactive({
  mode: 'data' as ServiceHubMode,
  services: initialServices,
  sessions: initialSessions,
  activeServiceId: '',
  activeSessionId: '',
  expandedServices: {
    s1: true,
    s2: false,
    s3: false,
    s4: false,
    s5: false,
    s6: false,
    s7: false,
    s8: false,
  } as Record<string, boolean>,
})

export const serviceCount = computed(() => serviceHubState.services.length)

export const getServiceTemplate = (service: ServiceRecord | undefined) =>
  serviceTemplates.find((template) => template.id === service?.templateId)

export const getService = (id: string) => serviceHubState.services.find((service) => service.id === id)

export const getSession = (id: string) => serviceHubState.sessions.find((session) => session.id === id)

export const getServiceSessions = (serviceId: string) =>
  serviceHubState.sessions.filter((session) => session.serviceId === serviceId)

export const getFirstServiceSession = (serviceId: string) => getServiceSessions(serviceId)[0]

export const createServiceRecord = (payload: {
  name: string
  description: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
  skillIds: string[]
}) => {
  const id = `s${Date.now()}`
  const template = serviceTemplates.find((item) => item.id === payload.templateId)
  const service: ServiceRecord = {
    id,
    name: payload.name.trim(),
    description: payload.description.trim(),
    templateId: payload.templateId,
    expertIds: [...payload.expertIds],
    knowledgeBaseIds: [...payload.knowledgeBaseIds],
    skillIds: [...payload.skillIds],
    role: '拥有者',
    members: 1,
    updatedLabel: '刚刚',
    updatedAt: Date.now(),
    state: 'active',
  }
  serviceHubState.services.unshift(service)
  serviceHubState.expandedServices[id] = true

  const sessionId = `c${Date.now()}`
  serviceHubState.sessions.unshift({
    id: sessionId,
    serviceId: id,
    title: '开始一段新的工作',
    expert: template?.experts[0] || serviceExperts.find((expert) => payload.expertIds.includes(expert.id))?.name || '服务助理',
    pinned: false,
    updatedLabel: '刚刚',
    messages: [],
  })
  return { service, sessionId }
}

export const createSession = (serviceId: string) => {
  const id = `c${Date.now()}`
  const session: ServiceSession = {
    id,
    serviceId,
    title: '开始一段新的工作',
    expert: getServiceTemplate(getService(serviceId))?.experts[0] || '服务助理',
    pinned: false,
    updatedLabel: '刚刚',
    messages: [],
  }
  serviceHubState.sessions.unshift(session)
  serviceHubState.expandedServices[serviceId] = true
  return session
}

export const addSessionMessage = (
  session: ServiceSession,
  message: Omit<ServiceSessionMessage, 'id'>,
) => {
  session.messages.push({ ...message, id: `m${Date.now()}-${session.messages.length}` })
  session.updatedLabel = '刚刚'
  const service = getService(session.serviceId)
  if (service) {
    service.updatedLabel = '刚刚'
    service.updatedAt = Date.now()
  }
}
