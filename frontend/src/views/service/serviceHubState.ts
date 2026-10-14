import { computed, reactive, ref } from 'vue'
import {
  createServiceSession as createServiceSessionRequest,
  getServiceProfile,
  getServiceSubjectProfile,
  listServiceFacts,
  listServiceReminders,
  listServiceSessions,
  listServiceSpaces,
  listServiceSubjects,
  type ServiceExpertBinding,
  type ServiceFact,
  type ServiceReminder,
  type ServiceSpace,
  type ServiceSession as ApiServiceSession,
  type ServiceSpaceProfile,
  type ServiceSubject,
} from '@/api/service'
import { listPublishedExperts, type PublishedExpert } from '@/api/expert-package'
import {
  buildCustomerInfo,
  factToActivity,
  reminderToActivity,
  sortActivities,
  type ServiceActivityItem,
  type ServiceCustomerInfo,
} from './servicePresentation'

export type ServiceHubMode = 'data' | 'empty' | 'archived'
export type ServiceRole = '拥有者' | '管理员' | '编辑者' | '只读'

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
  customer?: ServiceCustomerInfo
  recentActivity?: ServiceActivityItem[]
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
  title: session.title || '新的会话',
  expert: '服务主理人',
  pinned: Boolean(session.is_pinned),
  updatedLabel: formatUpdatedLabel(session.updated_at),
  updatedAt: session.updated_at ? Date.parse(session.updated_at) || Date.now() : Date.now(),
})

const loadServicePresentation = async (
  service: ServiceRecord,
) => {
  try {
    const [subjectResponse, factsResponse, remindersResponse] = await Promise.all([
      listServiceSubjects(service.id, { page: 1, page_size: 1 }),
      listServiceFacts(service.id, { page: 1, page_size: 12 }),
      listServiceReminders(service.id, { page: 1, page_size: 12 }),
    ])
    const subject = subjectResponse?.data?.items?.[0] || null
    let profile: ServiceSpaceProfile | null = null
    if (subject) {
      const profileResponse = await getServiceSubjectProfile(service.id, subject.id)
      profile = profileResponse?.data || null
    } else {
      const profileResponse = await getServiceProfile(service.id)
      profile = profileResponse?.data || null
    }
    const facts = factsResponse?.data?.items || []
    const reminders = remindersResponse?.data?.items || []
    service.customer = buildCustomerInfo(subject, profile, service.name)
    service.recentActivity = sortActivities([
      ...facts.map((item: ServiceFact) => factToActivity(item, profile)),
      ...reminders.map((item: ServiceReminder) => reminderToActivity(item)),
    ])
  } catch (error) {
    console.warn(`[ServiceHub] failed to load presentation data for ${service.id}`, error)
    service.customer = buildCustomerInfo(null, null, service.name)
    service.recentActivity = []
  }
  return service
}

export const loadServiceHub = async (force = false) => {
  if (loaded && !force) return
  if (loadPromise && !force) return loadPromise
  loadPromise = (async () => {
    try {
      const response = await listServiceSpaces({ include_archived: true })
      const services = Array.isArray(response?.data) ? response.data : []
      const mappedServices = await Promise.all(
        services.map((service) => loadServicePresentation(mapService(service, []))),
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
  return session
}

export const serviceCount = computed(() => serviceHubState.services.length)

export const getService = (id: string) => serviceHubState.services.find((service) => service.id === id)

export const getSession = (id: string) => serviceHubState.sessions.find((session) => session.id === id)

export const getServiceSessions = (serviceId: string) =>
  serviceHubState.sessions.filter((session) => session.serviceId === serviceId)

export const getFirstServiceSession = (serviceId: string) =>
  [...getServiceSessions(serviceId)].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return (b.updatedAt || 0) - (a.updatedAt || 0)
  })[0]
