<template>
  <div class="service-hub-page">
    <section class="service-hub-workspace-view">
      <header class="service-hub-space-head">
        <button type="button" class="service-hub-back-button" @click="backToList">
          <t-icon name="chevron-left" />
          返回
        </button>
        <span class="service-hub-space-identity">
          <span class="service-hub-space-icon" aria-hidden="true"><t-icon name="folder" /></span>
          <span class="service-hub-space-name">{{ activeSession?.title || activeService?.name || '服务' }}</span>
        </span>
        <span class="service-hub-space-template">
          {{ activeService?.customer?.familyName || activeService?.name }}
        </span>
        <span class="service-hub-principal-label" title="由主理人统一调度成员 Agent">
          <t-icon name="user-circle" />
          主理人
        </span>
        <div class="service-hub-space-actions">
          <t-popup v-model:visible="historyVisible" trigger="click" placement="bottom-left">
            <button
              type="button"
              class="service-hub-history-button"
              :class="{ active: historyVisible }"
              title="历史会话"
              aria-label="历史会话"
              @click="closeWorkspaceMenus('history')"
            >
              <span aria-hidden="true">◷</span>
            </button>
            <template #content>
              <div class="service-hub-history-menu">
                <div class="service-hub-history-title">历史会话</div>
                <div v-if="serviceSessions.length" class="service-hub-history-list">
                  <button
                    v-for="session in serviceSessions"
                    :key="session.id"
                    type="button"
                    :class="{ active: session.id === activeSession?.id }"
                    @click="openHistorySession(session)"
                  >
                    <span class="service-hub-history-pin">{{ session.pinned ? '●' : '' }}</span>
                    <span class="service-hub-history-copy">
                      <strong>{{ session.title }}</strong>
                      <small>{{ session.updatedLabel }}</small>
                    </span>
                    <t-icon v-if="session.id === activeSession?.id" name="check" />
                  </button>
                </div>
                <div v-else class="service-hub-history-empty">还没有历史会话</div>
              </div>
            </template>
          </t-popup>

          <button
            v-for="tool in headTools"
            :key="tool.key"
            type="button"
            class="service-hub-head-tool"
            :class="{ active: panel === tool.key }"
            @click="togglePanel(tool.key)"
          >
            <t-icon :name="tool.icon" />
            <span>{{ tool.label }}</span>
            <small>{{ tool.count }}</small>
          </button>
          <span v-if="activeService?.state !== 'active'" class="service-hub-active-state">
            {{ serviceStateLabel(activeService?.state) }}
          </span>
          <ServiceMoreMenu
            v-if="activeService"
            v-model:visible="serviceMenuVisible"
            :service="activeService"
            @navigate="openServicePage"
            @updated="updateActiveService"
            @archived="backToList"
            @removed="handleServiceRemoved"
          />
        </div>
      </header>

      <div class="service-hub-workbench">
        <article class="service-hub-chat">
          <div class="service-hub-chat-body">
            <ChatView
              v-if="activeChatSessionId"
              :key="`${activeSession?.id}:${activeChatSessionId}`"
              :session_id="activeChatSessionId"
              :service-id="activeService?.id || ''"
              :agent-id="serviceChatAgentId"
              :kb-ids="activeServiceKnowledgeBaseIds"
              embedded-input-placeholder="围绕当前服务整理摘要、话术和下一步"
              embedded-mode
              hosted-mode
            >
              <template #empty-suggestions>
                <div class="service-hub-progress-card">
                  <span>当前进展</span>
                  <strong>{{ activeService?.recentActivity?.[0]?.title || '这个服务还没有新的进展' }}</strong>
                  <p>
                    {{ activeService?.recentActivity?.[0]?.detail
                      || '从输入框开始一段工作，重点记录和下一步动作会自动沉淀在这里。' }}
                  </p>
                </div>
              </template>
            </ChatView>
            <div v-else class="service-hub-chat-state">
              <t-icon
                :name="chatSessionLoading ? 'loading' : 'chat'"
                :class="{ 'is-loading': chatSessionLoading }"
              />
              <span>{{ chatSessionLoading ? '正在准备会话' : chatSessionError || '会话暂不可用' }}</span>
              <t-button
                v-if="chatSessionError"
                variant="text"
                theme="primary"
                size="small"
                @click="retryChatSession"
              >
                重试
              </t-button>
            </div>

            <aside v-show="panel" class="service-hub-side-panel">
              <header>
                <strong>{{ panel === 'reminders' ? '服务待办' : '文件' }}</strong>
                <button
                  type="button"
                  class="service-hub-side-panel-close"
                  aria-label="关闭面板"
                  @click="panel = ''"
                >
                  <t-icon name="close" />
                </button>
              </header>
              <ServiceReminderPanel
                v-show="panel === 'reminders'"
                :service-id="activeService?.id || ''"
                @count-change="reminderCount = $event"
              />
              <ServiceArtifactPanel
                v-show="panel === 'artifacts'"
                :service-id="activeService?.id || ''"
                @count-change="artifactCount = $event"
              />
            </aside>
          </div>
        </article>
        <ServiceInfoRail
          :service-id="activeService?.id || ''"
          :fallback-name="activeService?.name"
          @open-activity="openServicePage('activity')"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ChatView from '@/views/chat/index.vue'
import { BUILTIN_SMART_REASONING_ID } from '@/api/agent'
import {
  SESSION_MUTATION_EVENT,
  type SessionMutationDetail,
} from '@/components/sessionMutations'
import ServiceArtifactPanel from './ServiceArtifactPanel.vue'
import ServiceInfoRail from './ServiceInfoRail.vue'
import ServiceMoreMenu from './ServiceMoreMenu.vue'
import ServiceReminderPanel from './ServiceReminderPanel.vue'
import {
  getService,
  getServiceSessions,
  getSession,
  loadServiceHub,
  serviceHubState,
  type ServiceRecord,
  type ServiceSession,
} from './serviceHubState'
import {
  ensureServiceSession,
  serviceBasePath,
  serviceDetailPath,
  serviceStateLabel,
  serviceWorkspaceLocation,
} from './serviceNavigation'

type HubPanel = '' | 'artifacts' | 'reminders'
type DetailSuffix = 'overview' | 'activity' | 'settings/basic'

const route = useRoute()
const router = useRouter()
const serviceChatAgentId = BUILTIN_SMART_REASONING_ID
const panel = ref<HubPanel>('')
const historyVisible = ref(false)
const serviceMenuVisible = ref(false)
const artifactCount = ref(0)
const reminderCount = ref(0)
const chatSessionLoadingId = ref('')
const chatSessionError = ref('')
const chatSessionRequests = new Map<string, Promise<string>>()

const activeService = computed(() => getService(serviceHubState.activeServiceId))
const activeSession = computed(() => getSession(serviceHubState.activeSessionId))
const activeChatSessionId = computed(() => activeSession.value?.chatSessionId || '')
const chatSessionLoading = computed(() => chatSessionLoadingId.value === activeSession.value?.id)
const activeServiceKnowledgeBaseIds = computed(() => activeService.value?.knowledgeBaseIds || [])
const serviceSessions = computed(() =>
  getServiceSessions(activeService.value?.id || '').sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return (b.updatedAt || 0) - (a.updatedAt || 0)
  }))
const headTools = computed(() => [
  { key: 'artifacts' as const, label: '文件', icon: 'file', count: artifactCount.value },
  { key: 'reminders' as const, label: '待办', icon: 'check-circle', count: reminderCount.value },
])

const routeServiceId = () => {
  const param = typeof route.params.serviceId === 'string' ? route.params.serviceId : ''
  return param || (typeof route.query.service === 'string' ? route.query.service : '')
}
const routeSessionId = () => {
  const param = typeof route.params.sessionId === 'string' ? route.params.sessionId : ''
  return param || (typeof route.query.session === 'string' ? route.query.session : '')
}

const syncWorkspace = async () => {
  await loadServiceHub()
  const serviceId = routeServiceId()
  const service = getService(serviceId)
  if (!service) {
    await router.replace(serviceBasePath(route))
    return
  }
  const requestedSession = routeSessionId() ? getSession(routeSessionId()) : undefined
  const session = requestedSession?.serviceId === serviceId
    ? requestedSession
    : await ensureServiceSession(serviceId)
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = session.id
  panel.value = route.query.panel === 'reminders'
    ? 'reminders'
    : route.query.panel === 'artifacts'
      ? 'artifacts'
      : ''

  if (!route.meta.mobileEntry && !route.params.sessionId) {
    await router.replace(serviceWorkspaceLocation(route, serviceId, session.id, {
      ...(panel.value ? { panel: panel.value } : {}),
    }))
  }
}

const ensureChatSession = async (session = activeSession.value) => {
  if (!session) return ''
  if (session.chatSessionId) return session.chatSessionId
  const pending = chatSessionRequests.get(session.id)
  if (pending) return pending
  chatSessionLoadingId.value = session.id
  chatSessionError.value = ''
  const request = Promise.resolve(session.id)
    .then((sessionId) => {
      session.chatSessionId = sessionId
      return sessionId
    })
    .catch((error) => {
      console.error('[ServiceWorkspace] Failed to prepare chat session:', error)
      chatSessionError.value = '会话创建失败，请稍后重试'
      return ''
    })
    .finally(() => {
      chatSessionRequests.delete(session.id)
      if (chatSessionLoadingId.value === session.id) chatSessionLoadingId.value = ''
    })
  chatSessionRequests.set(session.id, request)
  return request
}

const closeWorkspaceMenus = (source: 'history' | 'service') => {
  panel.value = ''
  if (source === 'history') {
    serviceMenuVisible.value = false
  } else {
    historyVisible.value = false
  }
}
const togglePanel = (nextPanel: Exclude<HubPanel, ''>) => {
  historyVisible.value = false
  serviceMenuVisible.value = false
  panel.value = panel.value === nextPanel ? '' : nextPanel
}
const openHistorySession = async (session: ServiceSession) => {
  if (!activeService.value) return
  closeWorkspaceMenus('history')
  historyVisible.value = false
  serviceHubState.activeSessionId = session.id
  await router.push(serviceWorkspaceLocation(route, activeService.value.id, session.id))
}
const openServicePage = (suffix: DetailSuffix) => {
  if (!activeService.value) return
  serviceMenuVisible.value = false
  panel.value = ''
  return router.push(serviceDetailPath(route, activeService.value.id, suffix))
}
const updateActiveService = (patch: Partial<ServiceRecord>) => {
  if (activeService.value) Object.assign(activeService.value, patch)
}
const backToList = async () => {
  serviceMenuVisible.value = false
  panel.value = ''
  serviceHubState.activeServiceId = ''
  serviceHubState.activeSessionId = ''
  await router.push(serviceBasePath(route))
}
const handleServiceRemoved = async () => {
  await loadServiceHub(true)
  await backToList()
}
const retryChatSession = () => {
  chatSessionError.value = ''
  void ensureChatSession()
}
const handleSessionMutation = (event: Event) => {
  const detail = (event as CustomEvent<SessionMutationDetail>).detail
  if (!detail?.sessionId) return
  const session = serviceHubState.sessions.find((item) => item.chatSessionId === detail.sessionId)
  if (!session) return
  if (detail.patch?.title) session.title = detail.patch.title
  if (typeof detail.patch?.is_pinned === 'boolean') session.pinned = detail.patch.is_pinned
  if (detail.removed) {
    session.chatSessionId = undefined
    if (session.id === serviceHubState.activeSessionId) void ensureChatSession(session)
  }
}

watch(
  () => [
    route.params.serviceId,
    route.params.sessionId,
    route.query.service,
    route.query.session,
    route.query.panel,
  ],
  syncWorkspace,
  { immediate: true },
)
watch(activeSession, (session) => {
  if (session) void ensureChatSession(session)
}, { immediate: true })
watch(serviceMenuVisible, (visible) => {
  if (visible) {
    historyVisible.value = false
    panel.value = ''
  }
})

onMounted(() => window.addEventListener(SESSION_MUTATION_EVENT, handleSessionMutation))
onUnmounted(() => window.removeEventListener(SESSION_MUTATION_EVENT, handleSessionMutation))
</script>
