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
          <span class="service-hub-service-icon" aria-hidden="true">
            <t-icon :name="getServiceTemplate(service)?.icon || 'folder'" />
          </span>
          <span class="service-hub-service-name">{{ service.name }}</span>
          <span class="service-hub-service-trailing">
            <button
              type="button"
              class="service-hub-service-caret"
              :class="{ collapsed: !isServiceExpanded(service.id) }"
              aria-label="展开或收起会话"
              @click.stop="toggleService(service.id)"
            >
              <t-icon name="chevron-down" size="14px" />
            </button>
            <span v-if="activeServiceId === service.id" class="service-hub-active-dot" />
          </span>
        </div>

        <div v-if="isServiceExpanded(service.id)" class="service-hub-session-list">
          <button
            v-for="session in getServiceSessions(service.id).slice(0, 5)"
            :key="session.id"
            type="button"
            class="service-hub-session-item"
            :class="{ active: activeSessionId === session.id }"
            :title="session.title"
            @click="openSession(service.id, session.id)"
          >
            <span class="service-hub-session-name">{{ session.title }}</span>
            <span class="service-hub-session-pin">{{ session.pinned ? '●' : '' }}</span>
            <span class="service-hub-session-more">···</span>
          </button>
          <button
            v-if="getServiceSessions(service.id).length > 5"
            type="button"
            class="service-hub-session-all"
            @click="openService(service.id)"
          >
            还有 {{ getServiceSessions(service.id).length - 5 }} 个
          </button>
          <button
            v-else-if="getServiceSessions(service.id).length === 0"
            type="button"
            class="service-hub-session-empty"
            @click="openService(service.id)"
          >
            开始一段工作
          </button>
        </div>
      </div>

      <div v-if="visibleServices.length === 0" class="service-hub-menu-empty">
        暂无服务
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  getServiceSessions,
  getServiceTemplate,
  serviceHubState,
} from '@/views/service/serviceHubState'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const isExpanded = ref(true)
const visibleServices = computed(() => {
  if (serviceHubState.mode === 'archived') {
    return serviceHubState.services.filter((service) => service.state === 'archived')
  }
  return serviceHubState.services.filter((service) => service.state !== 'archived')
})
const activeServiceId = computed(() => String(route.query.service || serviceHubState.activeServiceId || ''))
const activeSessionId = computed(() => String(route.query.session || serviceHubState.activeSessionId || ''))

const isServiceExpanded = (serviceId: string) => serviceHubState.expandedServices[serviceId] !== false

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
}

const toggleService = (serviceId: string) => {
  serviceHubState.expandedServices[serviceId] = !isServiceExpanded(serviceId)
}

const openServiceList = async () => {
  await router.push('/platform/service')
}

const openService = async (serviceId: string) => {
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = getServiceSessions(serviceId)[0]?.id || ''
  await router.push({
    path: '/platform/service',
    query: {
      service: serviceId,
      ...(serviceHubState.activeSessionId ? { session: serviceHubState.activeSessionId } : {}),
    },
  })
}

const openSession = async (serviceId: string, sessionId: string) => {
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = sessionId
  await router.push({
    path: '/platform/service',
    query: { service: serviceId, session: sessionId },
  })
}
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

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.service-hub-menu-title,
.service-hub-service-name,
.service-hub-session-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-service-icon {
  flex: none;
  color: var(--td-text-color-secondary);
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
  gap: 4px;
  flex-shrink: 0;
}

.service-hub-menu-count {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-menu-toggle,
.service-hub-service-caret {
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
  transition: opacity 0.18s ease, background-color 0.18s ease;
}

.service-hub-menu-header:hover .service-hub-menu-toggle,
.service-hub-menu-header:focus-within .service-hub-menu-toggle {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-menu-toggle:hover,
.service-hub-service-caret:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-menu-list,
.service-hub-session-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.service-hub-menu-list {
  padding: 1px 0 4px;
}

.service-hub-service-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 28px;
  padding: 0 8px 0 calc(var(--sidebar-inset-x) + 20px);
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.18s ease;

  &:hover,
  &.active {
    background: var(--td-bg-color-container-hover);
  }
}

.service-hub-service-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 18px;
}

.service-hub-service-item:hover .service-hub-service-icon,
.service-hub-service-item.active .service-hub-service-icon {
  color: var(--td-brand-color);
}

.service-hub-service-name {
  flex: 1;
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.service-hub-service-trailing {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  flex-shrink: 0;
  min-width: 36px;
}

.service-hub-service-caret {
  width: 20px;
  height: 20px;
  padding: 0;
  border-radius: 6px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease, background-color 0.18s ease;

  &.collapsed :deep(.t-icon) {
    transform: rotate(-90deg);
  }
}

.service-hub-service-item:hover .service-hub-service-caret,
.service-hub-service-item:focus-within .service-hub-service-caret {
  opacity: 1;
  pointer-events: auto;
}

.service-hub-active-dot {
  width: 12px;
  height: 12px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-hub-session-list {
  padding: 1px 0 3px;
}

.service-hub-session-item,
.service-hub-session-all,
.service-hub-session-empty {
  width: 100%;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  text-align: left;
}

.service-hub-session-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 26px;
  padding: 0 8px 0 54px;

  &:hover,
  &.active {
    background: var(--td-bg-color-container-hover);
  }
}

.service-hub-session-name {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  line-height: 18px;
}

.service-hub-session-item.active .service-hub-session-name {
  color: var(--td-text-color-primary);
  font-weight: 500;
}

.service-hub-session-pin {
  flex: none;
  width: 10px;
  color: var(--td-brand-color);
  font-size: 8px;
  line-height: 1;
  text-align: center;
}

.service-hub-session-more {
  flex: none;
  width: 18px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 18px;
  opacity: 0;
}

.service-hub-session-item:hover .service-hub-session-more {
  opacity: 1;
}

.service-hub-session-all,
.service-hub-session-empty,
.service-hub-menu-empty {
  padding: 4px 8px 5px 54px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-session-all:hover,
.service-hub-session-empty:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-menu-empty {
  padding-left: var(--sidebar-inset-x);
}
</style>
