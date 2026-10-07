<template>
  <section class="service-hub-menu">
    <div
      class="service-hub-menu-header"
      role="button"
      tabindex="0"
      @click="openServiceList"
      @keydown.enter.prevent="openServiceList"
      @keydown.space.prevent="openServiceList"
    >
      <span class="service-hub-menu-title">服务</span>
      <div class="service-hub-menu-actions" @click.stop>
        <span class="service-hub-menu-count">{{ visibleServices.length }}</span>
        <t-tooltip :content="isExpanded ? t('common.collapse') : t('common.expand')" placement="bottom">
          <button
            type="button"
            class="service-hub-menu-toggle"
            :aria-expanded="isExpanded"
            aria-label="展开或收起服务列表"
            @click.stop="toggleExpanded"
          >
            <t-icon :name="isExpanded ? 'chevron-down' : 'chevron-right'" size="16px" />
          </button>
        </t-tooltip>
      </div>
    </div>

    <div v-show="isExpanded" class="service-hub-menu-list">
      <div v-for="service in visibleServices" :key="service.id" class="service-hub-group">
        <div
          class="service-hub-service-item"
          :class="{ active: activeServiceId === service.id }"
          role="button"
          tabindex="0"
          :title="service.name"
          @click="openService(service.id)"
          @keydown.enter.prevent="openService(service.id)"
          @keydown.space.prevent="openService(service.id)"
        >
          <span class="service-hub-service-name">{{ service.name }}</span>
          <span class="service-hub-service-trailing">
            <button
              type="button"
              class="service-hub-session-count-action"
              :aria-label="`在${service.name}中新建会话`"
              @click.stop="startServiceSession(service.id)"
            >
              <span class="service-hub-session-count">{{ sortedSessions(service.id).length }}</span>
              <t-icon name="add" class="service-hub-session-add" size="14px" />
            </button>
            <button
              v-if="sortedSessions(service.id).length"
              type="button"
              class="service-hub-service-caret"
              :class="{ collapsed: expandedServiceId !== service.id }"
              aria-label="展开或收起会话"
              @click.stop="toggleService(service.id)"
            >
              <t-icon name="chevron-down" size="14px" />
            </button>
            <span v-else class="service-hub-service-caret-placeholder" />
            <span v-if="activeServiceId === service.id" class="service-hub-active-dot" />
          </span>
        </div>

        <div v-if="expandedServiceId === service.id" class="service-hub-session-list">
          <div
            v-for="session in visibleSessions(service.id)"
            :key="session.id"
            class="service-hub-session-item"
            :class="{ active: activeSessionId === session.id }"
          >
            <input
              v-if="renamingSessionId === session.id"
              ref="renameInputRef"
              v-model="renameDraft"
              class="service-hub-session-rename"
              maxlength="255"
              @click.stop
              @keydown.enter.prevent="saveSessionRename(service.id, session.id)"
              @keydown.esc.prevent="cancelSessionRename"
              @blur="saveSessionRename(service.id, session.id)"
            />
            <button
              v-else
              type="button"
              class="service-hub-session-open"
              :title="session.title"
              @click="openSession(service.id, session.id)"
            >
              <span class="service-hub-session-name">{{ session.title }}</span>
              <span class="service-hub-session-pin">{{ session.pinned ? '●' : '' }}</span>
            </button>
            <t-dropdown
              v-if="activeServiceId === service.id && renamingSessionId !== session.id"
              trigger="click"
              placement="bottom-right"
              attach="body"
            >
              <button
                type="button"
                class="service-hub-session-more"
                aria-label="会话操作"
                @click.stop
              >
                <t-icon name="more" size="14px" />
              </button>
              <template #dropdown>
                <t-dropdown-menu>
                  <t-dropdown-item @click="beginSessionRename(session)">重命名</t-dropdown-item>
                  <t-dropdown-item @click="toggleSessionPinned(service.id, session.id, !session.pinned)">
                    {{ session.pinned ? '取消置顶' : '置顶' }}
                  </t-dropdown-item>
                  <t-dropdown-item theme="error" @click="confirmRemoveSession(service.id, session.id, session.title)">
                    删除
                  </t-dropdown-item>
                </t-dropdown-menu>
              </template>
            </t-dropdown>
            <span v-else class="service-hub-session-more-placeholder" />
          </div>

          <button
            v-if="hiddenSessionCount(service.id) > 0"
            type="button"
            class="service-hub-session-all"
            @click="expandedAll[service.id] = true"
          >
            还有 {{ hiddenSessionCount(service.id) }} 个 ›
          </button>
          <button
            v-else-if="sortedSessions(service.id).length === 0"
            type="button"
            class="service-hub-session-empty"
            @click="startServiceSession(service.id)"
          >
            还没有进行中的工作，去开一段 ›
          </button>
        </div>
      </div>

      <div v-if="visibleServices.length === 0" class="service-hub-menu-empty">
        {{ archivedServiceCount ? '服务都已归档' : '还没有服务' }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  removeSession as removeSessionMutation,
  renameSession as renameSessionMutation,
  setSessionPinned as setSessionPinnedMutation,
} from '@/components/sessionMutations'
import {
  createServiceSession,
  getServiceSessions,
  loadServiceHub,
  serviceHubState,
  type ServiceSession,
} from '@/views/service/serviceHubState'

const SESSION_LIMIT = 8

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const isExpanded = ref(true)
const expandedServiceId = ref('')
const expandedAll = ref<Record<string, boolean>>({})
const renamingSessionId = ref('')
const renameDraft = ref('')
const renameInputRef = ref<HTMLInputElement | null>(null)
const savingRename = ref(false)

const visibleServices = computed(() =>
  serviceHubState.services
    .filter((service) => service.state === 'active' || service.state === 'paused')
    .sort((a, b) => b.updatedAt - a.updatedAt),
)
const archivedServiceCount = computed(() =>
  serviceHubState.services.filter((service) => service.state === 'archived').length,
)
const isServiceWorkspace = computed(() =>
  route.meta.mobileEntry
    ? Boolean(route.query.service)
    : Boolean(route.params.serviceId),
)
const activeServiceId = computed(() => isServiceWorkspace.value
  ? String(route.params.serviceId || route.query.service || serviceHubState.activeServiceId || '')
  : '',
)
const activeSessionId = computed(() => isServiceWorkspace.value
  ? String(route.params.sessionId || route.query.session || serviceHubState.activeSessionId || '')
  : '',
)
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')

const sortedSessions = (serviceId: string) =>
  [...getServiceSessions(serviceId)].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return (b.updatedAt || 0) - (a.updatedAt || 0)
  })

const visibleSessions = (serviceId: string) => {
  const sessions = sortedSessions(serviceId)
  return expandedAll.value[serviceId] ? sessions : sessions.slice(0, SESSION_LIMIT)
}

const hiddenSessionCount = (serviceId: string) =>
  expandedAll.value[serviceId] ? 0 : Math.max(0, sortedSessions(serviceId).length - SESSION_LIMIT)

const serviceRoute = (serviceId: string, sessionId?: string) => {
  if (route.meta.mobileEntry) {
    return {
      path: serviceBasePath.value,
      query: { service: serviceId, ...(sessionId ? { session: sessionId } : {}) },
    }
  }
  return {
    path: sessionId
      ? `${serviceBasePath.value}/${encodeURIComponent(serviceId)}/sessions/${encodeURIComponent(sessionId)}`
      : `${serviceBasePath.value}/${encodeURIComponent(serviceId)}`,
  }
}

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
}

const toggleService = (serviceId: string) => {
  expandedServiceId.value = expandedServiceId.value === serviceId ? '' : serviceId
  expandedAll.value[serviceId] = false
}

const openServiceList = async () => {
  await router.push(serviceBasePath.value)
}

const openService = async (serviceId: string) => {
  if (activeServiceId.value === serviceId) {
    expandedServiceId.value = serviceId
    return
  }
  expandedServiceId.value = serviceId
  expandedAll.value[serviceId] = false
  const session = sortedSessions(serviceId)[0]
  await router.push(serviceRoute(serviceId, session?.id))
}

const startServiceSession = async (serviceId: string) => {
  try {
    const session = await createServiceSession(serviceId, { title: '开始一段新的工作' })
    serviceHubState.activeServiceId = serviceId
    serviceHubState.activeSessionId = session.id
    expandedServiceId.value = serviceId
    expandedAll.value[serviceId] = false
    await router.push(serviceRoute(serviceId, session.id))
  } catch (error) {
    console.error('[ServiceHubMenu] Failed to create service session:', error)
    MessagePlugin.error('新会话创建失败，请稍后重试')
  }
}

const openSession = async (serviceId: string, sessionId: string) => {
  if (activeServiceId.value === serviceId && activeSessionId.value === sessionId) return
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = sessionId
  expandedServiceId.value = serviceId
  await router.push(serviceRoute(serviceId, sessionId))
}

const beginSessionRename = (session: ServiceSession) => {
  renamingSessionId.value = session.id
  renameDraft.value = session.title
  void nextTick(() => renameInputRef.value?.focus())
}

const cancelSessionRename = () => {
  renamingSessionId.value = ''
  renameDraft.value = ''
}

const saveSessionRename = async (serviceId: string, sessionId: string) => {
  if (savingRename.value || renamingSessionId.value !== sessionId) return
  const title = renameDraft.value.trim()
  const session = serviceHubState.sessions.find((item) => item.id === sessionId)
  if (!session) return cancelSessionRename()
  if (!title || title === session.title) return cancelSessionRename()
  savingRename.value = true
  try {
    await renameSessionMutation(sessionId, title, '', serviceId)
    session.title = title
    MessagePlugin.success('会话名称已更新')
  } catch (error) {
    console.error('[ServiceHubMenu] Failed to rename session:', error)
    MessagePlugin.error('会话重命名失败')
  } finally {
    savingRename.value = false
    cancelSessionRename()
  }
}

const toggleSessionPinned = async (serviceId: string, sessionId: string, pinned: boolean) => {
  try {
    await setSessionPinnedMutation(sessionId, pinned, serviceId)
    const session = serviceHubState.sessions.find((item) => item.id === sessionId)
    if (session) session.pinned = pinned
    MessagePlugin.success(pinned ? '会话已置顶' : '已取消置顶')
  } catch (error) {
    console.error('[ServiceHubMenu] Failed to pin session:', error)
    MessagePlugin.error('会话置顶状态更新失败')
  }
}

const confirmRemoveSession = (serviceId: string, sessionId: string, title: string) => {
  const dialog = DialogPlugin.confirm({
    header: '删除会话',
    body: `确定删除“${title}”吗？此操作不会删除已经保存的服务产物。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: '取消',
    onConfirm: async () => {
      dialog.setConfirmLoading(true)
      try {
        await removeSessionMutation(sessionId, serviceId)
        serviceHubState.sessions = serviceHubState.sessions.filter((item) => item.id !== sessionId)
        dialog.destroy()
        MessagePlugin.success('会话已删除')
        if (activeSessionId.value === sessionId) {
          const nextSession = sortedSessions(serviceId)[0]
          await router.push(serviceRoute(serviceId, nextSession?.id))
        }
      } catch (error) {
        console.error('[ServiceHubMenu] Failed to remove session:', error)
        dialog.setConfirmLoading(false)
        MessagePlugin.error('会话删除失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

watch(
  activeServiceId,
  (serviceId) => {
    if (!serviceId) return
    expandedServiceId.value = serviceId
    expandedAll.value[serviceId] = false
  },
  { immediate: true },
)

onMounted(() => {
  void loadServiceHub()
})
</script>

<style scoped lang="less">
.service-hub-menu {
  display: flex;
  min-height: 0;
  flex-direction: column;
  padding: 1px 0 5px;
}

.service-hub-menu-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 28px;
  padding: 0 8px 0 var(--sidebar-inset-x);
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.18s ease;
}

.service-hub-menu-header:hover {
  background: var(--td-bg-color-container-hover);
}

.service-hub-menu-title,
.service-hub-service-name,
.service-hub-session-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-menu-title {
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.service-hub-menu-actions {
  display: flex;
  align-items: center;
  min-width: 38px;
  justify-content: flex-end;
}

.service-hub-menu-count {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-menu-toggle,
.service-hub-service-caret,
.service-hub-session-count-action,
.service-hub-session-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-hub-menu-toggle {
  width: 24px;
  height: 24px;
  padding: 0;
  border-radius: 6px;
  opacity: 0;
  pointer-events: none;
}

.service-hub-menu-header:hover .service-hub-menu-toggle,
.service-hub-menu-header:focus-within .service-hub-menu-toggle {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-menu-header:hover .service-hub-menu-count,
.service-hub-menu-header:focus-within .service-hub-menu-count {
  display: none;
}

.service-hub-menu-list {
  display: flex;
  min-height: 0;
  max-height: min(56vh, 520px);
  padding: 1px 0 4px;
  flex-direction: column;
  gap: 2px;
  overflow-y: auto;
}

.service-hub-service-item {
  display: flex;
  align-items: center;
  box-sizing: border-box;
  width: 100%;
  min-height: 28px;
  padding: 0 8px 0 calc(var(--sidebar-inset-x) + 20px);
  border-radius: 8px;
  cursor: pointer;
}

.service-hub-service-item:hover,
.service-hub-service-item.active {
  background: var(--td-bg-color-container-hover);
}

.service-hub-service-name {
  min-width: 0;
  flex: 1;
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-service-item.active .service-hub-service-name {
  font-weight: 500;
}

.service-hub-service-trailing {
  display: inline-flex;
  align-items: center;
  min-width: 50px;
  justify-content: flex-end;
}

.service-hub-session-count-action {
  width: 22px;
  height: 22px;
  padding: 0;
  border-radius: 5px;
}

.service-hub-session-count {
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-session-add {
  display: none;
}

.service-hub-service-item:hover .service-hub-session-count,
.service-hub-service-item:focus-within .service-hub-session-count {
  display: none;
}

.service-hub-service-item:hover .service-hub-session-add,
.service-hub-service-item:focus-within .service-hub-session-add {
  display: inline-flex;
}

.service-hub-session-count-action:hover,
.service-hub-service-caret:hover,
.service-hub-session-more:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-service-caret,
.service-hub-service-caret-placeholder {
  width: 20px;
  height: 20px;
  flex: none;
}

.service-hub-service-caret {
  padding: 0;
  border-radius: 5px;
  opacity: 0;
  pointer-events: none;
}

.service-hub-service-caret.collapsed :deep(.t-icon) {
  transform: rotate(-90deg);
}

.service-hub-service-item:hover .service-hub-service-caret,
.service-hub-service-item:focus-within .service-hub-service-caret {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-active-dot {
  width: 8px;
  height: 8px;
  margin-left: 4px;
  flex: none;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-hub-session-list {
  display: flex;
  padding: 1px 0 3px;
  flex-direction: column;
  gap: 2px;
}

.service-hub-session-item {
  display: flex;
  align-items: center;
  min-height: 26px;
  padding: 0 8px 0 54px;
  border-radius: 8px;
}

.service-hub-session-item:hover,
.service-hub-session-item.active {
  background: var(--td-bg-color-container-hover);
}

.service-hub-session-open {
  display: flex;
  align-items: center;
  min-width: 0;
  min-height: 26px;
  padding: 0;
  flex: 1;
  gap: 6px;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.service-hub-session-name {
  min-width: 0;
  flex: 1;
  font-size: 12px;
  line-height: 18px;
}

.service-hub-session-item.active .service-hub-session-name {
  color: var(--td-text-color-primary);
  font-weight: 500;
}

.service-hub-session-pin {
  width: 10px;
  flex: none;
  color: var(--td-brand-color);
  font-size: 8px;
  line-height: 1;
  text-align: center;
}

.service-hub-session-more,
.service-hub-session-more-placeholder {
  width: 22px;
  height: 22px;
  flex: none;
}

.service-hub-session-more {
  padding: 0;
  border-radius: 5px;
  opacity: 0;
  pointer-events: none;
}

.service-hub-session-item:hover .service-hub-session-more,
.service-hub-session-item:focus-within .service-hub-session-more {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-session-rename {
  min-width: 0;
  height: 24px;
  padding: 0 6px;
  flex: 1;
  border: 1px solid var(--td-brand-color);
  border-radius: 4px;
  outline: 0;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font: inherit;
  font-size: 12px;
}

.service-hub-session-all,
.service-hub-session-empty,
.service-hub-menu-empty {
  width: 100%;
  padding: 4px 8px 5px 54px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 18px;
  text-align: left;
}

.service-hub-session-all:hover,
.service-hub-session-empty:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-menu-empty {
  padding-left: var(--sidebar-inset-x);
  cursor: default;
}
</style>
