<template>
  <section class="organize-menu">
    <div class="organize-menu-header" role="button" tabindex="0" @click="openDefaultRoute"
      @keydown.enter.prevent="openDefaultRoute" @keydown.space.prevent="openDefaultRoute">
      <span class="organize-menu-header-title">{{ t('menu.organize') }}</span>

      <div class="organize-menu-actions" @click.stop>
        <t-tooltip :content="isExpanded ? t('common.collapse') : t('common.expand')" placement="bottom">
          <button type="button" class="organize-menu-action-btn" :aria-expanded="isExpanded"
            :aria-controls="organizeMenuListId"
            :aria-label="isExpanded ? t('common.collapse') : t('common.expand')" @click.stop="toggleExpanded">
            <t-icon :name="isExpanded ? 'chevron-down' : 'chevron-right'" size="16px" />
          </button>
        </t-tooltip>
      </div>
    </div>

    <div v-show="isExpanded" :id="organizeMenuListId" class="organize-menu-list">
      <button v-for="item in organizeItems" :key="item.key" type="button" class="organize-menu-item"
        :class="{ active: isOrganizeRoute && activeTab === item.key }" :title="item.label"
        :aria-current="isOrganizeRoute && activeTab === item.key ? 'page' : undefined" @click="openRoute(item.path)">
        <OrganizeSproutIcon v-if="item.key === 'sprout'" class="organize-menu-item-icon" />
        <t-icon v-else :name="item.icon" class="organize-menu-item-icon" />
        <span class="organize-menu-item-name">{{ item.label }}</span>
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  ORGANIZE_MENU_ROUTES,
  isOrganizeTab,
  type OrganizeTab,
} from '@/views/organize/organizeRoutes'
import OrganizeSproutIcon from '@/views/organize/components/OrganizeSproutIcon.vue'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const organizeMenuListId = 'organize-menu-list'
const ORGANIZE_MENU_EXPANDED_STORAGE_KEY = 'sidebar-organize-menu-expanded'
const organizeItems = ORGANIZE_MENU_ROUTES

const loadExpandedState = () => {
  if (typeof window === 'undefined') return true
  try {
    return window.localStorage.getItem(ORGANIZE_MENU_EXPANDED_STORAGE_KEY) !== 'false'
  } catch {
    return true
  }
}

const isExpanded = ref(loadExpandedState())
const isOrganizeRoute = computed(() => typeof route.name === 'string' && route.name.startsWith('organize'))

const activeTab = computed<OrganizeTab>(() => {
  if (!isOrganizeRoute.value) return 'memory'
  const tab = route.meta.organizeTab
  return isOrganizeTab(tab) ? tab : 'memory'
})

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(ORGANIZE_MENU_EXPANDED_STORAGE_KEY, String(isExpanded.value))
  } catch {
    // Ignore storage failures; the toggle should still work for this session.
  }
}

const openRoute = async (path: string) => {
  if (route.path === path) return
  await router.push(path)
}

const openDefaultRoute = async () => {
  await openRoute(organizeItems[0].path)
}
</script>

<style scoped lang="less">
.organize-menu {
  display: flex;
  flex-direction: column;
  padding: 1px 0 5px;
}

.organize-menu-header {
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

.organize-menu-header-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.organize-menu-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
}

.organize-menu-header:hover .organize-menu-actions,
.organize-menu-header:focus-within .organize-menu-actions {
  opacity: 1;
  pointer-events: auto;
}

.organize-menu-action-btn {
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 6px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background-color 0.18s ease, color 0.18s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }
}

.organize-menu-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1px 0 4px;
}

.organize-menu-item {
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

.organize-menu-item-icon {
  flex-shrink: 0;
  color: var(--td-text-color-secondary);
}

.organize-menu-item.active .organize-menu-item-icon {
  color: var(--td-brand-color);
}

.organize-menu-item-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

</style>
