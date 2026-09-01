<template>
  <section class="service-menu">
    <div
      class="service-menu-header"
      role="button"
      tabindex="0"
      @click="openDefaultRoute"
      @keydown.enter.prevent="openDefaultRoute"
      @keydown.space.prevent="openDefaultRoute"
    >
      <span class="service-menu-header-title">服务</span>

      <div class="service-menu-actions" @click.stop>
        <t-tooltip :content="isExpanded ? t('common.collapse') : t('common.expand')" placement="bottom">
          <button
            type="button"
            class="service-menu-action-btn"
            :aria-expanded="isExpanded"
            :aria-controls="serviceMenuListId"
            :aria-label="isExpanded ? t('common.collapse') : t('common.expand')"
            @click.stop="toggleExpanded"
          >
            <t-icon :name="isExpanded ? 'chevron-down' : 'chevron-right'" size="16px" />
          </button>
        </t-tooltip>
      </div>
    </div>

    <div v-show="isExpanded" :id="serviceMenuListId" class="service-menu-list">
      <button
        v-for="item in serviceItems"
        :key="item.key"
        type="button"
        class="service-menu-item"
        :class="{ active: isServiceRoute && activeTab === item.key }"
        :title="item.label"
        :aria-current="isServiceRoute && activeTab === item.key ? 'page' : undefined"
        @click="openRoute(item.path)"
      >
        <t-icon :name="item.icon" class="service-menu-item-icon" />
        <span class="service-menu-item-name">{{ item.label }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  SERVICE_MENU_ROUTES,
  isServiceTab,
  type ServiceTab,
} from '@/views/service/serviceRoutes'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const serviceMenuListId = 'service-menu-list'
const SERVICE_MENU_EXPANDED_STORAGE_KEY = 'sidebar-service-menu-expanded'
const serviceItems = SERVICE_MENU_ROUTES

const loadExpandedState = () => {
  if (typeof window === 'undefined') return true
  try {
    return window.localStorage.getItem(SERVICE_MENU_EXPANDED_STORAGE_KEY) !== 'false'
  } catch {
    return true
  }
}

const isExpanded = ref(loadExpandedState())
const isServiceRoute = computed(() => typeof route.name === 'string' && route.name.startsWith('service'))

const activeTab = computed<ServiceTab>(() => {
  if (!isServiceRoute.value) return 'messages'
  const tab = route.meta.serviceTab
  return isServiceTab(tab) ? tab : 'messages'
})

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(SERVICE_MENU_EXPANDED_STORAGE_KEY, String(isExpanded.value))
  } catch {
    // Ignore storage failures; the toggle should still work for this session.
  }
}

const openRoute = async (path: string) => {
  if (route.path === path) return
  await router.push(path)
}

const openDefaultRoute = async () => {
  await openRoute(serviceItems[0].path)
}
</script>

<style scoped lang="less">
.service-menu {
  display: flex;
  flex-direction: column;
  padding: 1px 0 5px;
}

.service-menu-header {
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

.service-menu-header-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-menu-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
}

.service-menu-header:hover .service-menu-actions,
.service-menu-header:focus-within .service-menu-actions {
  opacity: 1;
  pointer-events: auto;
}

.service-menu-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background-color 0.18s ease, color 0.18s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }
}

.service-menu-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1px 0 4px;
}

.service-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 30px;
  padding: 0 12px 0 calc(var(--sidebar-inset-x) + 12px);
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background-color 0.18s ease, color 0.18s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.service-menu-item-icon {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.service-menu-item.active .service-menu-item-icon {
  color: var(--td-brand-color);
}

.service-menu-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
