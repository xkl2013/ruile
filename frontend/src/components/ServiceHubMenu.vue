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
          <t-icon name="chat" class="service-hub-service-icon" />
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
            <span v-if="activeServiceId === service.id" class="service-hub-active-dot" />
          </span>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createServiceSession,
  getServiceSessions,
  loadServiceHub,
  serviceHubState,
} from '@/views/service/serviceHubState'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const isExpanded = ref(true)

const visibleServices = computed(() =>
  serviceHubState.services
    .filter((service) => service.state === 'active' || service.state === 'paused')
    .sort((a, b) => b.updatedAt - a.updatedAt),
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
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')

const sortedSessions = (serviceId: string) =>
  [...getServiceSessions(serviceId)].sort((a, b) => {
    if (a.pinned !== b.pinned) return a.pinned ? -1 : 1
    return (b.updatedAt || 0) - (a.updatedAt || 0)
  })

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

const openServiceList = async () => {
  await router.push(serviceBasePath.value)
}

const openService = async (serviceId: string) => {
  try {
    serviceHubState.activeServiceId = serviceId
    const session = sortedSessions(serviceId)[0]
      || await createServiceSession(serviceId, { title: '新的会话' })
    serviceHubState.activeSessionId = session.id
    await router.push(serviceRoute(serviceId, session.id))
  } catch (error) {
    console.error('[ServiceHubMenu] Failed to open service:', error)
    MessagePlugin.error('服务打开失败，请稍后重试')
  }
}

const startServiceSession = async (serviceId: string) => {
  try {
    const session = await createServiceSession(serviceId, { title: '新的会话' })
    serviceHubState.activeServiceId = serviceId
    serviceHubState.activeSessionId = session.id
    await router.push(serviceRoute(serviceId, session.id))
  } catch (error) {
    console.error('[ServiceHubMenu] Failed to create service session:', error)
    MessagePlugin.error('新会话创建失败，请稍后重试')
  }
}

onMounted(() => {
  void loadServiceHub()
})
</script>

<style scoped lang="less">
.service-hub-menu {
  display: flex;
  flex-direction: column;
  padding: 1px 0 5px;
}

.service-hub-menu-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
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
.service-hub-service-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-menu-title {
  flex: 1;
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.service-hub-menu-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
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
  transition: background-color 0.18s ease, color 0.18s ease;
}

.service-hub-menu-header:hover .service-hub-menu-actions,
.service-hub-menu-header:focus-within .service-hub-menu-actions {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-menu-toggle:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-menu-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1px 0 4px;
}

.service-hub-service-item {
  display: flex;
  align-items: center;
  gap: 8px;
  box-sizing: border-box;
  width: 100%;
  min-height: 30px;
  padding: 0 12px 0 calc(var(--sidebar-inset-x) + 12px);
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background-color 0.18s ease, color 0.18s ease;
}

.service-hub-service-item:hover {
  background: var(--td-bg-color-container-hover);
}

.service-hub-service-item.active {
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-service-icon {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.service-hub-service-item.active .service-hub-service-icon {
  color: var(--td-brand-color);
}

.service-hub-service-name {
  min-width: 0;
  flex: 1;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.service-hub-service-trailing {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  flex: 0 0 42px;
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
.service-hub-session-count-action:focus-visible {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-active-dot {
  width: 8px;
  height: 8px;
  margin-left: 4px;
  flex: none;
  border-radius: 50%;
  background: var(--td-brand-color);
}
</style>
