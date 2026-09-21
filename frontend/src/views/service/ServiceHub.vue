<template>
  <div class="service-hub-page">
    <section v-if="view === 'list'" class="service-hub-list-view">
      <header class="service-hub-page-header">
        <div>
          <h1>服务</h1>
          <p>每个服务一个空间，专家在里面干活</p>
        </div>
        <div class="service-hub-header-art" aria-hidden="true">
          <div class="service-hub-art-window">
            <span></span><span></span><span></span>
          </div>
          <div class="service-hub-art-card">
            <span></span><span></span>
          </div>
          <div class="service-hub-art-dot"></div>
        </div>
      </header>

      <div class="service-hub-create-row">
        <t-button theme="primary" class="service-hub-primary-button" @click="openCreate()">
          <template #icon><t-icon name="add" /></template>
          新建服务
        </t-button>
      </div>

      <div v-if="showNoExpertBanner" class="service-hub-banner">
        <t-icon name="info-circle" />
        <span>有 1 个服务还没有添加专家，进入服务后可以继续配置。</span>
        <button type="button" aria-label="关闭提示" @click="showNoExpertBanner = false">
          <t-icon name="close" />
        </button>
      </div>

      <div v-if="services.length > 0" class="service-hub-list-main">
        <div class="service-hub-section-head">
          <span class="service-hub-section-title">我的服务</span>
          <div class="service-hub-section-tools">
            <t-select v-model="sortMode" class="service-hub-sort" size="small" :options="sortOptions" />
            <t-input v-model="serviceQuery" class="service-hub-search" size="small" placeholder="搜索服务">
              <template #prefix-icon><t-icon name="search" /></template>
            </t-input>
          </div>
        </div>

        <div v-if="filteredServices.length" class="service-hub-grid">
          <article
            v-for="service in filteredServices"
            :key="service.id"
            class="service-hub-card"
            @click="openWorkspace(service.id)"
          >
            <div class="service-hub-card-top">
              <div class="service-hub-card-icon">
                <t-icon :name="templateFor(service)?.icon || 'folder'" />
              </div>
              <strong>{{ service.name }}</strong>
              <t-dropdown trigger="click" placement="bottom-right" @click.stop>
                <button type="button" class="service-hub-more" aria-label="更多操作" @click.stop>
                  <t-icon name="more" />
                </button>
                <template #dropdown>
                  <t-dropdown-menu>
                    <t-dropdown-item @click="openCreate(service)">复制配置创建</t-dropdown-item>
                    <t-dropdown-item @click="archiveService(service)">
                      {{ service.state === 'archived' ? '恢复服务' : '归档服务' }}
                    </t-dropdown-item>
                  </t-dropdown-menu>
                </template>
              </t-dropdown>
            </div>
            <div class="service-hub-card-tags">
              <span class="service-hub-tag">{{ templateFor(service)?.name || '自定义服务' }}</span>
              <span class="service-hub-role">{{ service.role }}</span>
            </div>
            <p class="service-hub-card-description">{{ service.description || '还没有填写服务描述' }}</p>
            <div class="service-hub-card-meta">
              <span>{{ service.updatedLabel }}</span>
              <span>{{ service.members }} 位成员</span>
            </div>
          </article>
        </div>
        <div v-else class="service-hub-no-result">没有找到匹配的服务</div>

        <div class="service-hub-section-head service-hub-template-head">
          <span class="service-hub-section-title">从模板创建</span>
          <t-input v-model="templateQuery" class="service-hub-search" size="small" placeholder="搜索模板">
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
        </div>
        <div class="service-hub-grid">
          <article
            v-for="template in filteredTemplates"
            :key="template.id"
            class="service-hub-template-card"
            @click="openCreate(template)"
          >
            <div class="service-hub-template-top">
              <div class="service-hub-template-icon"><t-icon :name="template.icon" /></div>
              <strong>{{ template.name }}</strong>
            </div>
            <p>{{ template.description }}</p>
            <span>{{ template.experts.length }} 位专家 · 创建后可调整</span>
          </article>
        </div>
      </div>

      <div v-else class="service-hub-empty-state">
        <div>
          <h2>还没有服务</h2>
          <p>选一个模板开始，配置都预置好了，创建后随时能改。<br />也可以从空白服务自己配全套。</p>
        </div>
        <t-button variant="outline" @click="openCreate()">从空白创建</t-button>
      </div>
    </section>

    <section v-else-if="view === 'create'" class="service-hub-create-view">
      <header class="service-hub-page-header service-hub-create-header">
        <div>
          <h1>{{ editingService ? '复制服务配置' : '新建服务' }}</h1>
          <p>写清指令，选好专家、知识库与技能，创建后即可开始在服务空间里工作</p>
        </div>
      </header>

      <div class="service-hub-form">
        <section class="service-hub-form-panel">
          <div class="service-hub-field">
            <label for="service-name">服务名称 <em>*</em></label>
            <t-input id="service-name" v-model="form.name" placeholder="例如：秋季招生咨询" :status="formError ? 'error' : undefined" />
            <small v-if="formError" class="service-hub-error">{{ formError }}</small>
          </div>
          <div class="service-hub-field">
            <label for="service-description">服务描述</label>
            <t-textarea id="service-description" v-model="form.description" placeholder="说明这个服务负责什么，帮助成员和专家快速判断边界" :autosize="{ minRows: 3, maxRows: 5 }" />
          </div>
          <div class="service-hub-field">
            <label for="service-instruction">工作指令</label>
            <t-textarea id="service-instruction" v-model="form.instruction" placeholder="告诉专家要完成什么、优先参考什么资料、输出什么结果" :autosize="{ minRows: 5, maxRows: 8 }" />
            <div class="service-hub-count">{{ form.instruction.length }} / 4000</div>
          </div>
        </section>

        <section class="service-hub-form-panel">
          <div class="service-hub-field-head">
            <div>
              <h2>专家</h2>
              <p>专家在服务空间里协作完成工作，可以配置多个。</p>
            </div>
            <t-button variant="text" theme="primary" size="small" @click="expertPickerOpen = !expertPickerOpen">
              <template #icon><t-icon name="add" /></template>
              添加专家
            </t-button>
          </div>
          <div v-if="expertPickerOpen" class="service-hub-picker">
            <button
              v-for="expert in serviceExperts"
              :key="expert.id"
              type="button"
              class="service-hub-picker-option"
              :class="{ selected: form.expertIds.includes(expert.id) }"
              @click="toggleExpert(expert.id)"
            >
              <span>
                <strong>{{ expert.name }}</strong>
                <small>{{ expert.description }}</small>
              </span>
              <t-icon :name="form.expertIds.includes(expert.id) ? 'check' : 'add'" />
            </button>
          </div>
          <div class="service-hub-expert-list">
            <div v-for="expertId in form.expertIds" :key="expertId" class="service-hub-expert-row">
              <div class="service-hub-expert-avatar">{{ expertFor(expertId)?.name.slice(0, 1) }}</div>
              <div>
                <strong>{{ expertFor(expertId)?.name }}</strong>
                <small>{{ expertFor(expertId)?.description }}</small>
              </div>
              <button type="button" aria-label="移除专家" @click="toggleExpert(expertId)">
                <t-icon name="close" />
              </button>
            </div>
            <div v-if="form.expertIds.length === 0" class="service-hub-form-empty">还没有添加专家，服务创建后仍可以继续配置。</div>
          </div>
        </section>

        <section class="service-hub-form-panel">
          <div class="service-hub-field-head">
            <div>
              <h2>知识库</h2>
              <p>服务内的专家共享这些知识库，配置一次即可。</p>
            </div>
          </div>
          <div class="service-hub-choice-list">
            <button
              v-for="knowledgeBase in serviceKnowledgeBases"
              :key="knowledgeBase.id"
              type="button"
              class="service-hub-choice-row"
              :class="{ selected: form.knowledgeBaseIds.includes(knowledgeBase.id) }"
              @click="toggleChoice('knowledgeBaseIds', knowledgeBase.id)"
            >
              <span class="service-hub-choice-check"><t-icon name="check" /></span>
              <span><strong>{{ knowledgeBase.name }}</strong><small>{{ knowledgeBase.meta }}</small></span>
            </button>
          </div>
        </section>

        <section class="service-hub-form-panel">
          <div class="service-hub-field-head">
            <div>
              <h2>技能</h2>
              <p>为服务补充数据处理、文档协作和引用生成能力。</p>
            </div>
          </div>
          <div class="service-hub-choice-list">
            <button
              v-for="skill in serviceSkills"
              :key="skill.id"
              type="button"
              class="service-hub-choice-row"
              :class="{ selected: form.skillIds.includes(skill.id) }"
              @click="toggleChoice('skillIds', skill.id)"
            >
              <span class="service-hub-choice-check"><t-icon name="check" /></span>
              <span><strong>{{ skill.name }}</strong><small>{{ skill.description }}</small></span>
            </button>
          </div>
        </section>
      </div>

      <footer class="service-hub-form-footer">
        <div>
          <t-button variant="text" theme="primary" @click="saveDraft">保存为草稿</t-button>
          <span>可稍后继续</span>
        </div>
        <div>
          <t-button variant="outline" @click="backToList">取消</t-button>
          <t-button theme="primary" class="service-hub-primary-button" @click="submitForm">创建服务</t-button>
        </div>
      </footer>
    </section>

    <section v-else class="service-hub-workspace-view">
      <header class="service-hub-space-head">
        <button type="button" class="service-hub-back-button" @click="backToList">
          <t-icon name="chevron-left" />
          返回服务列表
        </button>
        <span class="service-hub-space-identity">
          <span class="service-hub-space-icon" aria-hidden="true">
            <t-icon :name="templateFor(activeService)?.icon || 'folder'" />
          </span>
          <span class="service-hub-space-name">{{ activeService?.name }}</span>
        </span>
        <span class="service-hub-space-template">{{ templateFor(activeService)?.name || '自定义服务' }}</span>
        <div class="service-hub-space-actions">
          <button
            v-for="tool in headTools"
            :key="tool.key"
            type="button"
            class="service-hub-head-tool"
            :class="{ active: panel === tool.key }"
            @click="panel = panel === tool.key ? '' : tool.key"
          >
            <t-icon :name="tool.icon" />
            <span>{{ tool.label }}</span>
            <small>{{ tool.count }}</small>
          </button>
          <span class="service-hub-active-state">进行中</span>
          <button type="button" class="service-hub-circle-button" title="新建会话" aria-label="新建会话" @click="startNewSession">＋</button>
          <t-dropdown trigger="click" placement="bottom-right">
            <button type="button" class="service-hub-circle-button" title="更多操作" aria-label="更多操作">
              <t-icon name="more" />
            </button>
            <template #dropdown>
              <t-dropdown-menu>
                <t-dropdown-item @click="openCreate(activeService)">复制配置创建</t-dropdown-item>
                <t-dropdown-item @click="archiveService(activeService)">归档服务</t-dropdown-item>
              </t-dropdown-menu>
            </template>
          </t-dropdown>
        </div>
      </header>

      <div class="service-hub-workbench">
        <article class="service-hub-chat">
          <header class="service-hub-chat-head">
            <strong>{{ activeSession?.title || '开始一段新的工作' }}</strong>
            <span>{{ activeSession?.expert || '服务助理' }}</span>
          </header>
          <div class="service-hub-chat-body">
            <ChatView
              v-if="activeChatSessionId"
              :key="`${activeSession?.id}:${activeChatSessionId}`"
              ref="serviceChatViewRef"
              :session_id="activeChatSessionId"
              :agent-id="serviceChatAgentId"
              :kb-ids="activeServiceKnowledgeBaseIds"
              :quoted-context="activeServiceChatContext"
              embedded-input-placeholder="围绕当前服务整理摘要、话术和下一步"
              embedded-mode
            >
              <template #empty-suggestions>
                <div class="service-hub-chat-prompts">
                  <strong>从一段工作开始</strong>
                  <p>把要整理、分析或推进的事情告诉专家，工作过程与产物会持续沉淀在服务空间。</p>
                  <button
                    v-for="prompt in serviceChatPrompts"
                    :key="prompt"
                    type="button"
                    @click="sendServiceChatPrompt(prompt)"
                  >
                    {{ prompt }}
                  </button>
                </div>
              </template>
            </ChatView>
            <div v-else class="service-hub-chat-state">
              <t-icon :name="activeChatSessionLoading ? 'loading' : 'chat'" :class="{ 'is-loading': activeChatSessionLoading }" />
              <span>{{ activeChatSessionLoading ? '正在准备会话' : activeChatSessionError || '会话暂不可用' }}</span>
              <t-button v-if="activeChatSessionError" variant="text" theme="primary" size="small" @click="retryActiveChatSession">
                重试
              </t-button>
            </div>

            <aside v-if="panel" class="service-hub-side-panel">
              <header>
                <strong>产物</strong>
                <button type="button" aria-label="关闭面板" @click="panel = ''"><t-icon name="close" /></button>
              </header>
              <div class="service-hub-side-panel-body">
                <button
                  v-for="artifact in activeArtifacts"
                  :key="artifact.id"
                  type="button"
                  class="service-hub-side-artifact"
                  :class="{ active: selectedArtifact?.id === artifact.id }"
                  @click="selectedArtifact = artifact"
                >
                  <span class="service-hub-artifact-icon">{{ artifact.format }}</span>
                  <span><strong>{{ artifact.title }}</strong><small>{{ artifact.meta }}</small></span>
                </button>
                <div v-if="!activeArtifacts.length" class="service-hub-panel-empty">当前会话还没有产物。</div>
                <pre v-if="selectedArtifact" class="service-hub-artifact-preview">{{ selectedArtifact.preview }}</pre>
              </div>
            </aside>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import ChatView from '@/views/chat/index.vue'
import { BUILTIN_QUICK_ANSWER_ID } from '@/api/agent'
import { createSessions } from '@/api/chat'
import {
  createServiceRecord,
  createSession,
  getFirstServiceSession,
  getService,
  getServiceTemplate,
  getSession,
  getServiceSessions,
  serviceArtifacts,
  serviceExperts,
  serviceHubState,
  serviceKnowledgeBases,
  serviceSkills,
  serviceTemplates,
  type ServiceRecord,
  type ServiceTemplate,
} from './serviceHubState'

type HubView = 'list' | 'create' | 'workspace'
type HubPanel = '' | 'artifacts'
type SortMode = 'recent' | 'created' | 'name'

const route = useRoute()
const router = useRouter()
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const serviceChatAgentId = BUILTIN_QUICK_ANSWER_ID
type ServiceChatViewExpose = {
  triggerSend?: (question: string) => void
}

const view = ref<HubView>('list')
const serviceQuery = ref('')
const templateQuery = ref('')
const sortMode = ref<SortMode>('recent')
const showNoExpertBanner = ref(true)
const editingService = ref<ServiceRecord | null>(null)
const selectedArtifact = ref<(typeof serviceArtifacts)[string] | null>(null)
const panel = ref<HubPanel>('')
const serviceChatViewRef = ref<ServiceChatViewExpose | null>(null)
const activeChatSessionLoadingId = ref('')
const activeChatSessionError = ref('')
const expertPickerOpen = ref(false)
const formError = ref('')
const form = reactive({
  name: '',
  description: '',
  instruction: '',
  templateId: '',
  expertIds: [] as string[],
  knowledgeBaseIds: [] as string[],
  skillIds: [] as string[],
})

const sortOptions = [
  { label: '按最近活动', value: 'recent' },
  { label: '按创建时间', value: 'created' },
  { label: '按名称', value: 'name' },
]

const activeService = computed(() => getService(serviceHubState.activeServiceId))
const activeSession = computed(() => getSession(serviceHubState.activeSessionId))
const activeChatSessionId = computed(() => activeSession.value?.chatSessionId || '')
const activeChatSessionLoading = computed(() => activeChatSessionLoadingId.value === activeSession.value?.id)
const activeServiceKnowledgeBaseIds = computed(() => activeService.value?.knowledgeBaseIds || [])
const activeServiceChatContext = computed(() => {
  if (!activeService.value) return ''
  const template = templateFor(activeService.value)
  return [
    `服务：${activeService.value.name}`,
    activeService.value.description ? `服务描述：${activeService.value.description}` : '',
    template?.instruction ? `工作指令：${template.instruction}` : '',
    activeSession.value?.expert ? `当前专家：${activeSession.value.expert}` : '',
  ].filter(Boolean).join('\n')
})
const serviceChatPrompts = [
  '帮我整理本周需要优先推进的重点工作',
  '把相关资料归纳成一份可执行的清单',
]
const services = computed(() => {
  if (serviceHubState.mode === 'archived') return serviceHubState.services.filter((service) => service.state === 'archived')
  return serviceHubState.services.filter((service) => service.state !== 'archived')
})
const filteredServices = computed(() => {
  const query = serviceQuery.value.trim().toLowerCase()
  const rows = services.value.filter((service) => `${service.name} ${service.description}`.toLowerCase().includes(query))
  return [...rows].sort((a, b) => {
    if (sortMode.value === 'name') return a.name.localeCompare(b.name, 'zh-CN')
    return b.updatedAt - a.updatedAt
  })
})
const filteredTemplates = computed(() => {
  const query = templateQuery.value.trim().toLowerCase()
  return serviceTemplates.filter((template) => `${template.name} ${template.description}`.toLowerCase().includes(query))
})
const activeArtifacts = computed(() => {
  const ids = activeSession.value?.messages.map((message) => message.artifactId).filter(Boolean) || []
  return [...new Set(ids)].map((id) => serviceArtifacts[id as string]).filter(Boolean)
})
const templateFor = (service: ServiceRecord | undefined) => getServiceTemplate(service)
const expertFor = (expertId: string) => serviceExperts.find((expert) => expert.id === expertId)
const getTemplateById = (templateId: string) => serviceTemplates.find((template) => template.id === templateId)
const headTools = computed(() => [
  { key: 'artifacts' as const, label: '产物', icon: 'file', count: activeArtifacts.value.length },
])

const isServiceRecord = (
  source: ServiceTemplate | ServiceRecord | null | undefined,
): source is ServiceRecord => Boolean(source && 'templateId' in source)

const isServiceTemplate = (
  source: ServiceTemplate | ServiceRecord | null | undefined,
): source is ServiceTemplate => Boolean(source && !('templateId' in source))

const resetForm = (source?: ServiceTemplate | ServiceRecord | null) => {
  const service = isServiceRecord(source) ? source : null
  const template = service
    ? getTemplateById(service.templateId)
    : isServiceTemplate(source)
      ? source
      : undefined
  form.name = service ? `${service.name}（副本）` : template?.name || ''
  form.description = service?.description || ''
  form.instruction = template?.instruction || ''
  form.expertIds = template?.experts
    .map((name) => serviceExperts.find((expert) => expert.name === name)?.id)
    .filter((id): id is string => Boolean(id)) || []
  form.knowledgeBaseIds = []
  form.skillIds = []
  form.templateId = template?.id || service?.templateId || ''
  formError.value = ''
  expertPickerOpen.value = false
}

const openCreate = (source?: ServiceTemplate | ServiceRecord | null) => {
  editingService.value = isServiceRecord(source) ? source : null
  resetForm(source)
  view.value = 'create'
}

const openWorkspace = async (serviceId: string, sessionId?: string) => {
  const service = getService(serviceId)
  if (!service) return
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = sessionId || getFirstServiceSession(serviceId)?.id || createSession(serviceId).id
  view.value = 'workspace'
  await router.replace({
    path: serviceBasePath.value,
    query: { service: serviceHubState.activeServiceId, session: serviceHubState.activeSessionId },
  })
}

const backToList = async () => {
  view.value = 'list'
  panel.value = ''
  await router.replace(serviceBasePath.value)
}

const startNewSession = async () => {
  if (!activeService.value) return
  const session = createSession(activeService.value.id)
  serviceHubState.activeSessionId = session.id
  panel.value = ''
  await router.replace({
    path: serviceBasePath.value,
    query: { service: activeService.value.id, session: session.id },
  })
}

const syncFromRoute = () => {
  const queryService = typeof route.query.service === 'string' ? route.query.service : ''
  const querySession = typeof route.query.session === 'string' ? route.query.session : ''
  if (queryService && getService(queryService)) {
    serviceHubState.activeServiceId = queryService
    serviceHubState.activeSessionId = querySession && getSession(querySession)?.serviceId === queryService
      ? querySession
      : getFirstServiceSession(queryService)?.id || createSession(queryService).id
    view.value = 'workspace'
    return
  }
  view.value = 'list'
}

const submitForm = () => {
  const name = form.name.trim()
  if (!name) {
    formError.value = '请输入服务名称'
    return
  }
  const result = createServiceRecord({
    name,
    description: form.description,
    templateId: form.templateId,
    expertIds: form.expertIds,
    knowledgeBaseIds: form.knowledgeBaseIds,
    skillIds: form.skillIds,
  })
  serviceHubState.activeServiceId = result.service.id
  serviceHubState.activeSessionId = result.sessionId
  view.value = 'workspace'
  void router.replace({
    path: serviceBasePath.value,
    query: { service: result.service.id, session: result.sessionId },
  })
  MessagePlugin.success('服务已创建')
}

const saveDraft = () => {
  MessagePlugin.success('已保存为草稿')
}

const toggleExpert = (expertId: string) => {
  form.expertIds = form.expertIds.includes(expertId)
    ? form.expertIds.filter((id) => id !== expertId)
    : [...form.expertIds, expertId]
}

const toggleChoice = (key: 'knowledgeBaseIds' | 'skillIds', id: string) => {
  form[key] = form[key].includes(id) ? form[key].filter((item) => item !== id) : [...form[key], id]
}

const archiveService = (service: ServiceRecord | undefined) => {
  if (!service) return
  service.state = service.state === 'archived' ? 'active' : 'archived'
  MessagePlugin.success(service.state === 'archived' ? '服务已归档' : '服务已恢复')
  if (view.value === 'workspace' && service.state === 'archived') {
    void backToList()
  }
}

const openArtifact = (artifactId: string) => {
  selectedArtifact.value = serviceArtifacts[artifactId] || null
  panel.value = 'artifacts'
}

const chatSessionRequests = new Map<string, Promise<string>>()

const ensureChatSession = async (session = activeSession.value) => {
  if (!session) return ''
  if (session.chatSessionId) return session.chatSessionId
  if (/^c[1-8]$/.test(session.id)) {
    session.chatSessionId = session.id
    return session.chatSessionId
  }
  const pending = chatSessionRequests.get(session.id)
  if (pending) return pending

  activeChatSessionLoadingId.value = session.id
  activeChatSessionError.value = ''
  const request = (async () => {
    try {
      const response = await createSessions({
        title: session.title,
        description: `service:${session.serviceId};service-session:${session.id};expert:${session.expert}`,
      })
      const chatSessionId = response?.data?.id
      if (!chatSessionId) throw new Error('missing session id')
      session.chatSessionId = String(chatSessionId)
      return session.chatSessionId
    } catch (error) {
      console.error('[ServiceHub] Failed to create chat session:', error)
      activeChatSessionError.value = '会话创建失败，请稍后重试'
      return ''
    } finally {
      chatSessionRequests.delete(session.id)
      if (activeChatSessionLoadingId.value === session.id) {
        activeChatSessionLoadingId.value = ''
      }
    }
  })()
  chatSessionRequests.set(session.id, request)
  return request
}

const retryActiveChatSession = () => {
  activeChatSessionError.value = ''
  void ensureChatSession()
}

const sendServiceChatPrompt = (prompt: string) => {
  serviceChatViewRef.value?.triggerSend?.(prompt)
}

watch(
  () => [route.query.service, route.query.session],
  () => syncFromRoute(),
  { immediate: true },
)

watch(
  activeSession,
  (session) => {
    if (session) void ensureChatSession(session)
  },
  { immediate: true },
)
</script>

<style scoped lang="less">
.service-hub-page {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-hub-list-view,
.service-hub-create-view,
.service-hub-workspace-view {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  box-sizing: border-box;
}

.service-hub-list-view,
.service-hub-create-view {
  overflow-y: auto;
  padding: 20px 28px 32px;
}

.service-hub-list-view {
  padding-right: 28px;
}

.service-hub-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  min-height: 60px;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    font-size: 24px;
    font-weight: 600;
    line-height: 32px;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.service-hub-header-art {
  position: relative;
  flex: none;
  width: 124px;
  height: 54px;
  opacity: 0.72;
}

.service-hub-art-window,
.service-hub-art-card {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.service-hub-art-window {
  top: 10px;
  left: 2px;
  width: 44px;
  height: 32px;
  padding: 0 8px;
  flex-wrap: wrap;
}

.service-hub-art-window span {
  width: 4px;
  height: 4px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
}

.service-hub-art-window span:nth-child(3) {
  width: 21px;
  height: 1px;
  border: 0;
  border-radius: 0;
  background: var(--td-component-border);
}

.service-hub-art-card {
  top: 13px;
  left: 58px;
  width: 42px;
  height: 26px;
  padding: 0 8px;
  border-color: var(--td-brand-color);
}

.service-hub-art-card span {
  display: block;
  width: 4px;
  height: 4px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-hub-art-card span:last-child {
  width: 12px;
  height: 1px;
  border-radius: 0;
}

.service-hub-art-dot {
  position: absolute;
  top: 18px;
  right: 3px;
  width: 17px;
  height: 17px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
}

.service-hub-create-row {
  margin-bottom: 24px;
}

.service-hub-primary-button {
  background: var(--td-brand-color);
  border: 0;
  color: var(--td-text-color-anti);
}

.service-hub-primary-button:hover {
  background: var(--td-brand-color-hover);
}

.service-hub-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 18px;
  padding: 10px 12px;
  border: 1px solid #c9edd6;
  border-radius: 6px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-banner button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-left: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.service-hub-list-main {
  min-width: 0;
}

.service-hub-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.service-hub-section-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.service-hub-section-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-sort {
  width: 132px;
}

.service-hub-search {
  width: 180px;
}

.service-hub-template-head {
  margin-top: 26px;
}

.service-hub-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.service-hub-card,
.service-hub-template-card {
  min-width: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.service-hub-card {
  min-height: 137px;
  padding: 10px 11px;
}

.service-hub-card:hover,
.service-hub-template-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.04);
}

.service-hub-card-top,
.service-hub-template-top {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.service-hub-card-top strong,
.service-hub-template-top strong {
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-card-icon,
.service-hub-template-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-card-icon {
  width: 24px;
  height: 24px;
}

.service-hub-template-icon {
  width: 22px;
  height: 22px;
}

.service-hub-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 24px;
  height: 24px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-more:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-card-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 8px;
}

.service-hub-tag,
.service-hub-role {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.service-hub-tag {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-role {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
}

.service-hub-card-description {
  display: -webkit-box;
  min-height: 34px;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.service-hub-card-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-template-card {
  min-height: 108px;
  padding: 11px 12px;
}

.service-hub-template-card p {
  margin: 7px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
}

.service-hub-template-card > span {
  display: block;
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-no-result,
.service-hub-panel-empty {
  padding: 32px 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.service-hub-empty-state {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-top: 6px;
  padding: 20px 22px;
  border: 1px dashed var(--td-component-border);
  border-radius: 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-empty-state h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-empty-state p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;
}

.service-hub-create-view {
  max-width: 1080px;
}

.service-hub-create-header {
  margin-bottom: 22px;
}

.service-hub-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-hub-form-panel {
  padding: 20px 22px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
}

.service-hub-field {
  margin-bottom: 16px;
}

.service-hub-field:last-child {
  margin-bottom: 0;
}

.service-hub-field > label {
  display: block;
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.service-hub-field > label em {
  color: var(--td-error-color);
  font-style: normal;
}

.service-hub-count {
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-align: right;
}

.service-hub-error {
  display: block;
  margin-top: 5px;
  color: var(--td-error-color);
  font-size: 12px;
}

.service-hub-field-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.service-hub-field-head h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-field-head p {
  margin: 4px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-picker {
  margin-bottom: 12px;
  padding: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.service-hub-picker-option,
.service-hub-choice-row {
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: var(--app-font-family);
  text-align: left;
}

.service-hub-picker-option {
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
}

.service-hub-picker-option:hover,
.service-hub-picker-option.selected,
.service-hub-choice-row:hover,
.service-hub-choice-row.selected {
  background: var(--td-brand-color-1);
}

.service-hub-picker-option > span,
.service-hub-choice-row > span:last-child {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.service-hub-picker-option strong,
.service-hub-choice-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-picker-option small,
.service-hub-choice-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-list,
.service-hub-choice-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-hub-expert-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-expert-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 12px;
  font-weight: 600;
}

.service-hub-expert-row > div:nth-child(2) {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-expert-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-expert-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-row button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-form-empty {
  padding: 10px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-choice-row {
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-choice-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex: none;
  border: 1px solid var(--td-component-border);
  border-radius: 3px;
  color: transparent;
}

.service-hub-choice-row.selected .service-hub-choice-check {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.service-hub-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 20px;
}

.service-hub-form-footer > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-form-footer span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-workspace-view {
  padding: 18px 24px 18px 28px;
  overflow: hidden;
}

.service-hub-space-head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 38px;
  padding-bottom: 12px;
}

.service-hub-back-button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 4px 6px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 13px;
}

.service-hub-back-button:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-space-name {
  color: var(--td-text-color-primary);
  font-size: 16px;
  font-weight: 600;
}

.service-hub-space-identity {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.service-hub-space-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 24px;
  height: 24px;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-space-template,
.service-hub-active-state {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-space-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

.service-hub-head-tool {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 999px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
}

.service-hub-head-tool:hover,
.service-hub-head-tool.active {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-head-tool small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-hub-circle-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 50%;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-hub-circle-button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-workbench {
  display: flex;
  flex: 1;
  min-height: 0;
}

.service-hub-chat {
  display: flex;
  position: relative;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
}

.service-hub-chat-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 11px 14px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-chat-head strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-chat-head span {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-hub-chat-body {
  display: flex;
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.service-hub-chat-body :deep(.chat) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  padding: 0;
}

.service-hub-chat-body :deep(.chat_scroll_box) {
  padding: 12px 18px 0;
}

.service-hub-chat-body :deep(.msg_list) {
  max-width: 960px;
}

.service-hub-chat-body :deep(.input-container.is-embedded) {
  padding: 12px 18px 16px;
}

.service-hub-chat-state {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.service-hub-chat-state .t-icon {
  font-size: 24px;
}

.service-hub-chat-state .is-loading {
  animation: service-hub-spin 0.9s linear infinite;
}

@keyframes service-hub-spin {
  to {
    transform: rotate(360deg);
  }
}

.service-hub-chat-prompts {
  display: flex;
  width: min(520px, 100%);
  margin: auto;
  padding: 24px 0;
  flex-direction: column;
  gap: 8px;
  text-align: center;
}

.service-hub-chat-prompts strong {
  font-size: 15px;
  font-weight: 500;
}

.service-hub-chat-prompts p {
  margin: 0 0 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 20px;
}

.service-hub-chat-prompts button {
  width: 100%;
  padding: 9px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 18px;
  text-align: left;
}

.service-hub-chat-prompts button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-side-artifact {
  display: flex;
  align-items: center;
  gap: 9px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.service-hub-side-artifact:hover,
.service-hub-side-artifact.active {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
}

.service-hub-artifact-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 6px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 10px;
  font-weight: 600;
}

.service-hub-side-artifact > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-side-artifact strong {
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-side-artifact small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-hub-side-panel {
  display: flex;
  position: absolute;
  z-index: 2;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(320px, 42%);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  box-shadow: -10px 0 26px rgba(0, 0, 0, 0.1);
}

.service-hub-side-panel > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 11px 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-side-panel > header strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-side-panel > header button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-side-panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.service-hub-side-artifact {
  width: 100%;
  margin-bottom: 8px;
  padding: 8px 10px;
}

.service-hub-artifact-preview {
  margin: 4px 0 0;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 21px;
  white-space: pre-wrap;
}

@media (max-width: 1180px) {
  .service-hub-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 920px) {
  .service-hub-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .service-hub-header-art {
    display: none;
  }

  .service-hub-space-head {
    flex-wrap: wrap;
  }

  .service-hub-space-actions {
    width: 100%;
    margin-left: 0;
  }
}

@media (max-width: 680px) {
  .service-hub-list-view,
  .service-hub-create-view,
  .service-hub-workspace-view {
    padding: 16px;
  }

  .service-hub-grid {
    grid-template-columns: 1fr;
  }

  .service-hub-section-head,
  .service-hub-section-tools,
  .service-hub-form-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .service-hub-search,
  .service-hub-sort {
    width: 100%;
  }

  .service-hub-form-footer > div {
    justify-content: flex-end;
  }

  .service-hub-space-actions {
    overflow-x: auto;
  }

  .service-hub-active-state {
    display: none;
  }

  .service-hub-side-panel {
    width: 78%;
  }
}
</style>
