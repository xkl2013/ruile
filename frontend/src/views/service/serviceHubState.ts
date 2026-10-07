import { computed, reactive, ref } from 'vue'
import {
  createServiceSession as createServiceSessionRequest,
  listServiceExperts,
  listServiceSessions,
  listServiceSpaces,
  listServiceTemplates,
  type ServiceExpertBinding,
  type ServiceSpace,
  type ServiceSpaceTemplate as ApiServiceSpaceTemplate,
  type ServiceSession as ApiServiceSession,
} from '@/api/service'
import { listPublishedExperts, type PublishedExpert } from '@/api/expert-package'

export type ServiceHubMode = 'data' | 'empty' | 'archived'
export type ServiceRole = '拥有者' | '管理员' | '编辑者' | '只读'

export interface ServiceTemplate {
  id: string
  name: string
  description: string
  instruction: string
  experts: string[]
  icon: string
  spaceType?: 'customer_service' | 'operations' | 'research'
  spaceTypeLabel?: string
  subjectLabel?: string
  matchScore?: number
  matchReason?: string
  profileFields?: string[]
  summarySections?: string[]
  autoApply?: boolean
}

export interface ServiceExpertOption {
  id: string
  name: string
  description: string
  avatar?: string
  domain?: string
  packageName?: string
  packageDescription?: string
  skills: string[]
}

export interface ServiceRecord {
  id: string
  name: string
  description: string
  instruction?: string
  spaceType?: 'customer_service' | 'operations' | 'research'
  templateId: string
  expertIds?: string[]
  knowledgeBaseIds?: string[]
  skillIds?: string[]
  visibility?: 'private' | 'tenant'
  role: ServiceRole
  members: number
  memberLimit: number
  updatedLabel: string
  updatedAt: number
  createdAt: number
  state: 'active' | 'archived' | 'draft' | 'paused'
}

export interface ServiceSession {
  id: string
  chatSessionId?: string
  serviceId: string
  title: string
  expert: string
  pinned: boolean
  updatedLabel: string
  updatedAt?: number
}

const marketResearchInstruction = `你是“市场调研与竞品分析协同助手”，服务产品、市场、运营、战略和业务决策团队，负责从调研课题定义、资料收集、竞品拆解、洞察分析、报告产出、结论评审到持续跟踪的完整周期。

【服务主体】
本空间的主体不限定为园所、家长或某一种固定角色。请根据调研课题识别并建立可持续跟踪的调研主体，包括但不限于行业、细分市场、竞品、替代方案、标杆产品、用户群体、客户案例、渠道、政策、内部业务团队和关键决策人。空间档案的主体字段应随课题动态生成，至少包含：主体名称、主体类型、所属范围、关键属性、证据来源、当前状态和最近更新时间。

【核心工作原则】
1. 严格区分事实、分析和建议。事实必须有来源、时间和适用范围；分析要说明推导依据；建议要说明适用条件、预期价值和风险。
2. 对关键数据、竞品信息、行业观点和用户反馈尽量标注来源、发布时间、来源可信度、样本范围和可能偏差。
3. 不把未经验证的推测写成确定结论，不编造数据、案例、引用或竞品能力。信息不足时明确标记“待验证”，并给出补充验证方式。
4. 每次处理资料后，更新相关主体的空间档案和首页摘要；重要变化要保留版本、时间和证据链。
5. 低风险的目录、字段、摘要和报告结构可按本模板直接生成；涉及权限、外部发送、数据范围或不可逆变更时，先提示用户确认。

【阶段一：调研课题定义】
当用户正在定义调研课题时：
- 澄清业务背景、核心问题、目标受众、决策用途和时间范围。
- 判断调研类型：行业趋势、竞品分析、用户需求、市场规模、商业模式、渠道策略、增长机会或其他类型。
- 拆解一级问题、二级问题、待验证假设和关键信息。
- 输出《调研 Brief》、问题清单、调研范围、信息来源建议和预期交付物。
- 将课题、目标、范围、假设和时间范围写入空间档案，并生成首页摘要。
- 提醒用户可以创建调研待办，分配调研负责人，并添加业务方或决策者为关注人。

【阶段二：资料收集】
当用户正在收集资料时：
- 整理公开资料、行业报告、新闻资讯、问卷结果、用户访谈、竞品官网、产品体验和内部文档。
- 识别资料来源、发布时间、适用范围、可信度、样本限制和可能偏差。
- 按市场数据、用户洞察、产品功能、价格策略、渠道策略、客户案例、商业模式等类别归档。
- 对相互矛盾的信息并列展示，不强行合并；说明差异可能来自时间、样本或口径不同。
- 提醒用户将原始资料、链接、问卷结果和访谈记录上传到项目资料库，便于复核和追问。

【阶段三：竞品拆解】
当用户正在分析竞品时：
- 先确定竞品范围，区分直接竞品、间接竞品、替代方案和标杆产品。
- 从目标用户、核心场景、功能结构、产品流程、商业模式、定价策略、增长渠道、客户案例和优劣势等维度拆解。
- 输出竞品对比表、功能矩阵、能力差异、体验亮点和风险点。
- 避免只罗列功能，进一步提炼竞品背后的产品策略、用户价值和业务意图。
- 对缺少一手验证的内容标记来源等级和待验证项，提醒用户上传竞品分析表或体验截图到项目资料库。

【阶段四：洞察分析】
当用户正在提炼调研结论时：
- 从已收集的信息中识别趋势、机会点、风险点、用户痛点和潜在策略方向。
- 将事实、判断和建议分栏表达，并为关键结论补充证据来源、适用条件和不确定性说明。
- 根据课题需要形成 SWOT、机会优先级、竞品差距、用户需求洞察或市场进入建议。
- 输出“核心发现、证据依据、判断与假设、机会与风险、下一步验证”结构的洞察摘要。
- 重要洞察可以创建为“调研结论评审”待办，邀请产品、市场、运营或业务负责人讨论确认。

【阶段五：报告产出】
当用户需要输出调研报告时：
- 按调研背景、调研目标、方法说明、核心发现、竞品分析、机会判断、风险提示和建议动作组织报告。
- 根据使用对象调整口径：产品团队重点突出需求机会和功能建议；管理层重点突出市场判断、投入产出和决策建议；运营团队重点突出渠道、流程和行动清单。
- 生成汇报大纲、PPT 结构、结论页、竞品对比表和行动建议清单。
- 对引用的数据、案例和观点标注来源、时间和依据；无法确认时明确标注待核实。
- 报告完成后提醒用户上传到项目资料库，并创建“调研结论评审”待办，添加业务方或决策者为关注人。

【阶段六：结论评审】
当团队正在评审调研结论时：
- 整理评审议题、关键结论、争议点、待确认问题和决策项。
- 支持基于已有报告追问“这个竞品为什么这么做”“这个机会是否适合我们”“下一步应该验证什么”。
- 将评审反馈转化为行动项：继续补充调研、进入需求设计、开展用户访谈、拉取数据验证或暂不推进。
- 将已确认的结论、争议处理结果和行动项沉淀到空间档案、首页摘要和项目资料库。

【阶段七：持续跟踪】
当调研结论需要持续更新时：
- 跟踪竞品动态、行业变化、政策变化、用户反馈和市场数据变化。
- 定期输出变化摘要，说明变化事实、影响范围、对原结论的影响和是否需要重新验证。
- 维护竞品分析表、市场观察清单、主体档案和机会池。
- 重要变化更新到原报告中，并提醒关注人重新评估。

【默认输出格式】
除非用户另有要求，先给出结论摘要，再给出证据、事实与判断区分、不确定性、影响范围和下一步动作。对于资料不足的任务，固定输出“已知事实、未知信息、当前假设、建议验证方式”。每个阶段完成后，主动提醒用户将资料、分析表、报告或评审结论上传到项目资料库，并根据需要创建待办、分配负责人和添加关注人。`

const earlyChildhoodMembershipInstruction = `你是“早教机构会员服务助手”，服务早教机构的会员服务、教务、课程顾问、班主任、运营和管理团队，负责围绕会员家庭提供连续、准确、可追踪的服务。

【服务主体】
以“会员家庭”作为主要服务主体，以“孩子”作为关联服务对象；不要把家长、孩子、会员账户和一次服务请求混为同一主体。根据实际情况建立或更新会员档案，主体粒度可扩展为会员家庭、孩子、课程包、权益、服务待办、活动或订单。

【会员档案】
优先维护以下信息，并只记录完成服务所必需的内容：
- 会员家庭：会员编号、联系人、联系方式、会员等级、会员状态、入会时间、来源渠道。
- 关联孩子：称呼或编号、年龄阶段、已购课程、上课班级、出勤状态、学习偏好和服务关注点。
- 会员权益：课程包或会员方案、剩余次数、有效期、可用权益、已使用权益、预约限制和到期提醒。
- 服务偏好：常用沟通方式、到店时段、课程偏好、活动偏好和特殊沟通要求。
- 服务记录：问题或需求、发生时间、沟通渠道、处理过程、处理结果、负责人、承诺时间和下一步动作。
- 对健康、安全、支付和身份等敏感信息，只在获得授权且确有服务必要时记录；不在普通回复中暴露无关隐私。

【核心工作原则】
1. 先识别会员主体、关联孩子、当前会员状态和本次服务待办，再给出回复或建议。
2. 涉及课程安排、请假转课、退费、续费、会员权益、价格、活动规则和机构政策时，优先检索已授权知识库；资料不足时明确说明，不擅自承诺。
3. 严格区分“已确认事实、待确认信息、服务建议和需要人工决策的待办”，不把推测写成机构承诺。
4. 每次服务完成后更新会员档案、服务记录和首页摘要；低风险的字段补全、摘要更新、待办整理和提醒建议可直接生成，无需用户二次确认。
5. 涉及退款、权益变更、课程调换、对外发送、投诉定责、健康安全风险、隐私披露或其他不可逆操作时，先整理方案并提示负责人确认。
6. 回复要清晰、友好、克制，不使用过度营销话术；对儿童安全、身体不适、疑似意外或家长重大投诉，优先升级给对应负责人并记录升级原因。

【会员服务工作流】
当用户提出会员服务需求时：
1. 识别服务待办类型：会员咨询、入会/入班、课程预约、请假转课、活动报名、权益使用、课程进度、出勤异常、账单支付、续费提醒、投诉建议或其他待办。
2. 建立服务上下文：确认会员家庭、关联孩子、课程或权益、事件时间、当前状态、用户诉求和期望完成时间。
3. 查询可用资料和历史记录，检查是否存在重复工单、未完成承诺、到期权益或需要跟进的风险。
4. 输出当前可执行的回复、处理步骤、所需资料、负责人和下一步动作。
5. 服务结束后沉淀一条结构化服务记录，并更新会员首页摘要。

【不同场景处理】
- 会员咨询：先回答已确认的信息，再列出需要补充的信息和可选方案。
- 课程预约与调整：核对会员权益、课程规则、班级容量和时间冲突；无法确认时不要直接承诺名额。
- 请假、转课和补课：记录原课程、申请原因、目标课程、规则依据、处理状态和到期时间。
- 会员权益：展示权益总量、已用数量、剩余数量、有效期和使用限制；数字来源不明确时标记待核对。
- 续费服务：根据到期时间、使用频率、剩余权益、近期反馈和历史沟通生成分层跟进建议；不把建议当成强制销售话术。
- 投诉与负面反馈：先复述事实和诉求，区分情绪表达、事实问题和责任判断；形成处理方案、负责人、时限和回访节点。
- 儿童健康或安全相关情形：不进行医疗诊断；记录已知事实，建议联系家长和机构指定负责人，必要时立即升级处理。

【空间档案与首页摘要】
本空间自动维护：
- 会员概况：会员状态、会员等级、入会时间、关联孩子和当前课程。
- 权益与使用：课程包、剩余权益、有效期、预约/请假限制和最近使用情况。
- 近期服务：最近咨询、未完成待办、投诉建议和承诺节点。
- 会员关系：满意度信号、服务偏好、续费状态和可能流失风险。
- 下一步动作：负责人、待办待办、计划完成时间和需要会员确认的信息。
每次更新标注更新时间和信息来源；发生冲突时保留原记录并提示核对，不静默覆盖。

【协同与待办】
当出现未完成服务、跨部门处理、到期提醒、会员回访、投诉跟进或续费跟进时，提醒用户创建待办，分配会员服务负责人，并根据需要添加教务、课程顾问、班主任、运营负责人或管理者为关注人。重要处理结果应同步沉淀到项目资料库或会员服务记录。

【默认输出格式】
除非用户另有要求，按以下顺序输出：
1. 当前结论或可执行回复；
2. 会员主体、关联孩子和服务待办；
3. 已确认事实与依据；
4. 待确认信息、风险和权限边界；
5. 处理步骤、负责人和下一步动作。
当资料不足时固定输出“已知信息、缺失信息、当前建议、需要谁确认”。`

export const serviceTemplates = reactive<ServiceTemplate[]>([
  {
    id: 't1',
    name: '招生咨询全流程',
    description: '从线索进入、跟进到转化，含话术与结果归档',
    instruction: '跟进 2026 秋季招生线索，语气亲切不催促，重点记录家长顾虑与到访意向；涉及课程与价格时先查知识库再回答。',
    experts: ['招生顾问', '教务主管'],
    icon: 'usergroup',
    spaceType: 'customer_service',
    spaceTypeLabel: '客户服务',
    subjectLabel: '服务对象',
    matchScore: 96,
    matchReason: '匹配你的招生与家校沟通工作画像',
    profileFields: ['当前阶段', '核心顾虑', '到访意向'],
    summarySections: ['进展', '风险与卡点', '下一步动作'],
    autoApply: true,
  },
  {
    id: 't2',
    name: '家长续费跟进',
    description: '续费窗口跟进、沟通话术与结果归档',
    instruction: '在续费窗口期跟进家长，说明续费政策与优惠，记录家长反馈与决策结果。',
    experts: ['教务主管'],
    icon: 'refresh',
    spaceType: 'customer_service',
    spaceTypeLabel: '客户服务',
    subjectLabel: '服务对象',
    matchScore: 88,
    matchReason: '匹配你的续费与家长沟通工作画像',
    profileFields: ['续费阶段', '决策顾虑', '沟通记录'],
    summarySections: ['续费进展', '待确认信息', '下一步动作'],
    autoApply: true,
  },
  {
    id: 't3',
    name: '一日流程巡查',
    description: '巡班记录、异常上报与每日汇总',
    instruction: '按一日流程巡查各班，记录异常并当日汇总上报。',
    experts: ['带班老师', '保育员'],
    icon: 'browse',
    spaceType: 'operations',
    spaceTypeLabel: '运营管理',
    subjectLabel: '业务对象',
    matchScore: 74,
    matchReason: '匹配你的日常运营与异常跟进工作画像',
    profileFields: ['巡查日期', '异常类型', '处理状态'],
    summarySections: ['今日概况', '异常与卡点', '待办'],
    autoApply: true,
  },
  {
    id: 't4',
    name: '新教师带教',
    description: '带教计划、阶段评估与反馈沉淀',
    instruction: '为新教师制定带教计划，按阶段评估并沉淀反馈。',
    experts: ['教务主管', '带班老师'],
    icon: 'user',
    spaceType: 'operations',
    spaceTypeLabel: '运营管理',
    subjectLabel: '服务对象',
    matchScore: 68,
    matchReason: '匹配你的团队培养与教学管理工作画像',
    profileFields: ['带教阶段', '观察重点', '反馈记录'],
    summarySections: ['带教进展', '待改进项', '下一步计划'],
    autoApply: true,
  },
  {
    id: 't5',
    name: '活动筹备',
    description: '排期、通知与活动后复盘',
    instruction: '筹备活动：排期、通知、物资准备与活动后复盘。',
    experts: ['后勤主任'],
    icon: 'calendar',
    spaceType: 'operations',
    spaceTypeLabel: '运营管理',
    subjectLabel: '项目对象',
    matchScore: 61,
    matchReason: '匹配你的活动统筹与跨角色协作工作画像',
    profileFields: ['活动阶段', '关键节点', '物资状态'],
    summarySections: ['整体进展', '待办与风险', '复盘要点'],
    autoApply: true,
  },
  {
    id: 't6',
    name: '教学教研沉淀',
    description: '观察记录、教研资料与经验沉淀',
    instruction: '整理课堂观察记录与教研资料，形成可复用的经验条目。',
    experts: ['带班老师'],
    icon: 'book-open',
    spaceType: 'research',
    spaceTypeLabel: '研究沉淀',
    subjectLabel: '研究对象',
    matchScore: 52,
    matchReason: '匹配你的教学观察与经验沉淀工作画像',
    profileFields: ['观察主题', '关键发现', '可复用做法'],
    summarySections: ['研究进展', '关键发现', '待验证问题'],
    autoApply: true,
  },
  {
    id: 't7',
    name: '市场调研与竞品分析协同助手',
    description: '从课题定义、资料收集到报告评审与持续跟踪，沉淀可复用的市场与竞品洞察',
    instruction: marketResearchInstruction,
    experts: ['市场研究员', '竞品分析师'],
    icon: 'search',
    spaceType: 'research',
    spaceTypeLabel: '研究协同',
    subjectLabel: '调研主体',
    matchScore: 80,
    matchReason: '适合行业、竞品与用户需求类调研课题',
    profileFields: ['课题类型', '研究范围', '核心假设', '关键结论'],
    summarySections: ['核心发现', '证据与不确定性', '机会与风险', '下一步验证'],
    autoApply: true,
  },
  {
    id: 't8',
    name: '早教机构会员服务',
    description: '围绕会员家庭管理权益、课程服务、问题跟进与续费生命周期',
    instruction: earlyChildhoodMembershipInstruction,
    experts: ['会员服务顾问', '教务主管', '班主任'],
    icon: 'usergroup',
    spaceType: 'customer_service',
    spaceTypeLabel: '会员服务',
    subjectLabel: '会员家庭',
    matchScore: 86,
    matchReason: '适合早教机构的会员档案、课程权益与服务跟进',
    profileFields: ['会员状态', '孩子阶段', '权益使用', '服务偏好', '续费风险'],
    summarySections: ['会员概况', '近期服务', '权益与到期', '待跟进'],
    autoApply: true,
  },
])

export const serviceExperts = reactive<ServiceExpertOption[]>([])
export const serviceExpertsLoading = ref(false)
export const serviceExpertsError = ref('')

let serviceExpertsLoadPromise: Promise<ServiceExpertOption[]> | null = null

const mapPublishedExpert = (expert: PublishedExpert): ServiceExpertOption => ({
  id: expert.definition_id,
  name: expert.display_name,
  description: expert.description || expert.package_description || '暂无专家描述',
  avatar: expert.avatar,
  domain: expert.domain,
  packageName: expert.package_display_name,
  packageDescription: expert.package_description,
  skills: Array.isArray(expert.skills) ? expert.skills.filter(Boolean) : [],
})

export const loadServiceExperts = async (force = false) => {
  if (serviceExperts.length && !force) return serviceExperts
  if (serviceExpertsLoadPromise && !force) return serviceExpertsLoadPromise
  serviceExpertsLoading.value = true
  serviceExpertsError.value = ''
  serviceExpertsLoadPromise = listPublishedExperts()
    .then((response) => {
      const experts = Array.isArray(response?.data)
        ? response.data.map(mapPublishedExpert)
        : []
      serviceExperts.splice(0, serviceExperts.length, ...experts)
      return serviceExperts
    })
    .catch((error) => {
      serviceExpertsError.value = error?.message || '系统专家列表加载失败'
      throw error
    })
    .finally(() => {
      serviceExpertsLoading.value = false
      serviceExpertsLoadPromise = null
    })
  return serviceExpertsLoadPromise
}

export const getServiceExpert = (id: string) =>
  serviceExperts.find((expert) => expert.id === id)

export const serviceHubState = reactive({
  mode: 'data' as ServiceHubMode,
  services: [] as ServiceRecord[],
  sessions: [] as ServiceSession[],
  activeServiceId: '',
  activeSessionId: '',
  expandedServices: {} as Record<string, boolean>,
})

let loadPromise: Promise<void> | null = null
let loaded = false

const serviceRoleLabel = (role?: string): ServiceRole => {
  switch (role) {
    case 'owner':
      return '拥有者'
    case 'admin':
      return '管理员'
    case 'editor':
      return '编辑者'
    default:
      return '只读'
  }
}

const formatUpdatedLabel = (value?: string) => {
  if (!value) return '刚刚'
  const timestamp = Date.parse(value)
  if (!Number.isFinite(timestamp)) return '刚刚'
  const diffMinutes = Math.max(0, Math.floor((Date.now() - timestamp) / 60000))
  if (diffMinutes < 1) return '刚刚'
  if (diffMinutes < 60) return `${diffMinutes} 分钟前`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours} 小时前`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 7) return `${diffDays} 天前`
  return new Date(timestamp).toLocaleDateString('zh-CN', { month: 'numeric', day: 'numeric' })
}

const mapRemoteServiceTemplate = (
  remoteTemplate: ApiServiceSpaceTemplate,
  existing?: ServiceTemplate,
): ServiceTemplate => {
  const blueprint = remoteTemplate.blueprint
  return {
    id: remoteTemplate.key,
    name: remoteTemplate.name || existing?.name || remoteTemplate.key,
    description: existing?.description || blueprint?.source_instruction || '已发布的服务空间模板',
    instruction: existing?.instruction || blueprint?.source_instruction || '',
    experts: existing?.experts
      || blueprint?.expert_suggestions?.map((expert) => expert.expert_ref).filter(Boolean)
      || [],
    icon: existing?.icon || 'folder',
    spaceType: blueprint?.proposed_space_type || existing?.spaceType,
    spaceTypeLabel: existing?.spaceTypeLabel,
    subjectLabel: existing?.subjectLabel,
    matchScore: existing?.matchScore,
    matchReason: existing?.matchReason,
    profileFields: blueprint?.profile_schema?.map((field) => field.label) || existing?.profileFields || [],
    summarySections: blueprint?.summary_schema?.map((section) => section.label) || existing?.summarySections || [],
    autoApply: remoteTemplate.auto_apply,
  }
}

const syncServiceTemplates = async () => {
  try {
    const response = await listServiceTemplates()
    const remoteTemplates = Array.isArray(response?.data) ? response.data : []
    remoteTemplates.forEach((remoteTemplate) => {
      const existing = serviceTemplates.find((template) => template.id === remoteTemplate.key)
      const mapped = mapRemoteServiceTemplate(remoteTemplate, existing)
      if (existing) {
        Object.assign(existing, mapped)
      } else {
        serviceTemplates.push(mapped)
      }
    })
  } catch (error) {
    console.warn('[ServiceHub] failed to load published service templates', error)
  }
}

const mapService = (service: ServiceSpace, experts: ServiceExpertBinding[]): ServiceRecord => ({
  id: service.id,
  name: service.name,
  description: service.description || '',
  instruction: service.instruction || '',
  spaceType: service.space_type,
  templateId: service.template_key || '',
  expertIds: experts.map((expert) => expert.expert_ref),
  knowledgeBaseIds: service.knowledge_base_ids || [],
  skillIds: service.selected_skills || [],
  visibility: service.visibility || 'private',
  role: serviceRoleLabel(service.role),
  members: service.member_count || 0,
  memberLimit: service.member_limit || 20,
  updatedLabel: formatUpdatedLabel(service.updated_at),
  updatedAt: service.updated_at ? Date.parse(service.updated_at) || Date.now() : Date.now(),
  createdAt: service.created_at ? Date.parse(service.created_at) || Date.now() : Date.now(),
  state: service.state,
})

const mapSession = (session: ApiServiceSession): ServiceSession => ({
  id: session.id,
  chatSessionId: session.id,
  serviceId: session.service_id,
  title: session.title || '开始一段新的工作',
  expert: session.expert_name || session.expert_ref || '服务助理',
  pinned: Boolean(session.is_pinned),
  updatedLabel: formatUpdatedLabel(session.updated_at),
  updatedAt: session.updated_at ? Date.parse(session.updated_at) || Date.now() : Date.now(),
})

export const loadServiceHub = async (force = false) => {
  if (loaded && !force) return
  if (loadPromise && !force) return loadPromise
  loadPromise = (async () => {
    try {
      await syncServiceTemplates()
      await loadServiceExperts().catch((error) => {
        console.warn('[ServiceHub] failed to load published experts', error)
      })
      const response = await listServiceSpaces({ include_archived: true })
      const services = Array.isArray(response?.data) ? response.data : []
      const mappedServices = await Promise.all(
        services.map(async (service) => {
          try {
            const expertsResponse = await listServiceExperts(service.id)
            return mapService(service, Array.isArray(expertsResponse?.data) ? expertsResponse.data : [])
          } catch (error) {
            console.warn(`[ServiceHub] failed to load experts for ${service.id}`, error)
            return mapService(service, [])
          }
        }),
      )
      const sessionGroups = await Promise.all(
        services.map(async (service) => {
          try {
            const sessionsResponse = await listServiceSessions(service.id, { page: 1, page_size: 100 })
            return sessionsResponse?.data?.items || []
          } catch (error) {
            console.warn(`[ServiceHub] failed to load sessions for ${service.id}`, error)
            return []
          }
        }),
      )
      serviceHubState.services = mappedServices
      serviceHubState.sessions = sessionGroups.flat().map(mapSession)
      mappedServices.forEach((service) => {
        if (!(service.id in serviceHubState.expandedServices)) {
          serviceHubState.expandedServices[service.id] = false
        }
      })
      loaded = true
    } finally {
      loadPromise = null
    }
  })()
  return loadPromise
}

export const createServiceSession = async (
  serviceId: string,
  input?: { title?: string; description?: string; expert_ref?: string; expert_name?: string },
) => {
  const response = await createServiceSessionRequest(serviceId, input)
  if (!response?.data) throw new Error('missing service session')
  const session = mapSession(response.data)
  serviceHubState.sessions = [
    session,
    ...serviceHubState.sessions.filter((item) => item.id !== session.id),
  ]
  serviceHubState.expandedServices[serviceId] = true
  return session
}

export const serviceCount = computed(() => serviceHubState.services.length)

export const getServiceTemplate = (service: ServiceRecord | undefined) =>
  serviceTemplates.find((template) => template.id === service?.templateId)

export const getService = (id: string) => serviceHubState.services.find((service) => service.id === id)

export const getSession = (id: string) => serviceHubState.sessions.find((session) => session.id === id)

export const getServiceSessions = (serviceId: string) =>
  serviceHubState.sessions.filter((session) => session.serviceId === serviceId)

export const getFirstServiceSession = (serviceId: string) =>
  [...getServiceSessions(serviceId)].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return (b.updatedAt || 0) - (a.updatedAt || 0)
  })[0]
