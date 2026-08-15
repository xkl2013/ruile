<template>
  <section class="kb-menu">
    <div class="kb-menu-header" role="button" tabindex="0" @click="goToKnowledgeBaseList"
      @keydown.enter.prevent="goToKnowledgeBaseList" @keydown.space.prevent="goToKnowledgeBaseList">
      <span class="kb-menu-header-title">{{ title }}</span>

      <div class="kb-menu-actions" @click.stop>
        <t-tooltip :content="isExpanded ? t('common.collapse') : t('common.expand')" placement="bottom">
          <button type="button" class="kb-menu-action-btn kb-menu-action-btn--toggle"
            :aria-expanded="isExpanded" :aria-controls="kbMenuListId"
            :aria-label="isExpanded ? t('common.collapse') : t('common.expand')" @click.stop="toggleExpanded">
            <t-icon :name="isExpanded ? 'chevron-down' : 'chevron-right'" size="16px" />
          </button>
        </t-tooltip>

        <t-tooltip v-if="canCreateKnowledgeBase" :content="t('knowledgeList.create')" placement="bottom">
          <button type="button" class="kb-menu-action-btn kb-menu-action-btn--create"
            :aria-label="t('knowledgeList.create')" @click.stop="openCreateKnowledgeBase('document')">
            <t-icon name="add" size="16px" />
          </button>
        </t-tooltip>
      </div>
    </div>

    <div v-show="isExpanded" :id="kbMenuListId" class="kb-menu-list">
      <button v-for="kb in knowledgeBases" :key="kb.id" type="button" class="kb-menu-item"
        :class="{ active: kb.id === activeKbId }" :title="kb.name" :aria-current="kb.id === activeKbId ? 'page' : undefined"
        @click="openKnowledgeBase(kb.id)">
        <KnowledgeBaseIcon :icon="kb.icon" :type="kb.type" size="small" class="kb-menu-item-icon" />
        <span class="kb-menu-item-name">{{ kb.name }}</span>
        <span v-if="kb.id === activeKbId" class="kb-menu-item-dot" />
      </button>

      <div v-if="refreshing && knowledgeBases.length === 0" class="kb-menu-loading">
        <t-loading size="small" />
      </div>
      <div v-else-if="!refreshing && knowledgeBases.length === 0" class="kb-menu-empty">
        {{ t('knowledgeBase.noKnowledge') }}
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import { useUIStore } from '@/stores/ui'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useOrganizationStore } from '@/stores/organization'
import { mergeAllScopeKnowledgeBases } from '@/views/knowledge/kbListMerge'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'

type SidebarKnowledgeBase = {
  id: string
  name: string
  icon?: string
  type?: 'document' | 'faq'
  isMine?: boolean
  is_pinned?: boolean
  pinned_at?: string
  creator_id?: string
  description?: string
  permission?: string
  shared_at?: string
  share_id?: string
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const uiStore = useUIStore()
const chatResources = useChatResourcesStore()
const orgStore = useOrganizationStore()
const { rawKnowledgeBases } = storeToRefs(chatResources)
const { sharedKnowledgeBases } = storeToRefs(orgStore)

const title = computed(() => t('menu.knowledgeBase'))
const refreshing = ref(false)
const kbMenuListId = 'knowledge-base-menu-list'
const KB_MENU_EXPANDED_STORAGE_KEY = 'sidebar-knowledge-base-menu-expanded'
const isExpanded = ref(true)
const currentRouteName = computed(() => {
  if (typeof route.name === 'string') return route.name
  return route.name ? String(route.name) : ''
})

const loadExpandedState = () => {
  if (typeof window === 'undefined') return true
  try {
    return window.localStorage.getItem(KB_MENU_EXPANDED_STORAGE_KEY) !== 'false'
  } catch {
    return true
  }
}

isExpanded.value = loadExpandedState()

const activeKbId = computed(() => {
  const kbId = route.params.kbId
  if (typeof kbId === 'string') return kbId
  if (Array.isArray(kbId)) return kbId[0] || ''
  return ''
})

const canCreateKnowledgeBase = computed(() => authStore.hasRole('admin'))

const canReadTenantKnowledgeBase = (kb: SidebarKnowledgeBase): boolean => {
  if (authStore.hasRole('admin')) return true
  const userId = authStore.user?.id || ''
  return !!(kb.creator_id && userId && kb.creator_id === userId)
}

const knowledgeBases = computed<SidebarKnowledgeBase[]>(() => {
  const merged = mergeAllScopeKnowledgeBases(
    (rawKnowledgeBases.value as unknown as SidebarKnowledgeBase[]).filter(canReadTenantKnowledgeBase),
    sharedKnowledgeBases.value as any[],
    authStore.user?.id,
  )

  return merged
    .filter((kb: any) => !!kb?.id)
    .map((kb: any) => ({
      id: String(kb.id),
      name: String(kb.name || kb.id),
      icon: kb.icon,
      type: kb.type,
      isMine: kb.isMine === true,
      is_pinned: !!kb.is_pinned,
      pinned_at: kb.pinned_at,
      creator_id: kb.creator_id,
      description: kb.description,
      permission: kb.permission,
      shared_at: kb.shared_at,
      share_id: kb.share_id,
    }))
})

const createHostRoutes = new Set(['knowledgeBaseList', 'knowledgeBaseDetail', 'kbCreatChat', 'globalCreatChat', 'chat', 'home'])

const goToKnowledgeBaseList = async () => {
  if (route.name === 'knowledgeBaseList') return
  await router.push('/platform/knowledge-bases')
}

const toggleExpanded = () => {
  isExpanded.value = !isExpanded.value
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(KB_MENU_EXPANDED_STORAGE_KEY, String(isExpanded.value))
  } catch {
    // Ignore storage failures; the toggle should still work for this session.
  }
}

const openKnowledgeBase = async (kbId: string) => {
  if (!kbId) return
  if (activeKbId.value === kbId && currentRouteName.value === 'knowledgeBaseDetail') return
  await router.push(`/platform/knowledge-bases/${kbId}`)
}

const openCreateKnowledgeBase = async (type: 'document' | 'faq' = 'document') => {
  if (!canCreateKnowledgeBase.value) return
  if (!createHostRoutes.has(currentRouteName.value)) {
    await router.push('/platform/knowledge-bases')
  }
  uiStore.openCreateKB(type)
}

const refreshKnowledgeBases = async () => {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await chatResources.ensureKnowledgeBases(true)
  } catch (error) {
    console.error('[KnowledgeBaseMenu] refresh failed:', error)
  } finally {
    refreshing.value = false
  }
}

onMounted(() => {
  void refreshKnowledgeBases()
})
</script>

<style scoped lang="less">
.kb-menu {
  display: flex;
  flex-direction: column;
  padding: 1px 0 5px;
}

.kb-menu-header {
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

.kb-menu-header-title {
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

.kb-menu-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
}

.kb-menu-header:hover .kb-menu-actions,
.kb-menu-header:focus-within .kb-menu-actions {
  opacity: 1;
  pointer-events: auto;
}

.kb-menu-action-btn {
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

  &:disabled {
    cursor: not-allowed;
    opacity: 0.75;
  }
}

.kb-menu-action-btn--toggle {
  color: var(--td-text-color-secondary);
}

.kb-menu-action-btn--create {
  color: var(--td-text-color-primary);
}

.kb-menu-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 1px 0 4px;
}

.kb-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 28px;
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

.kb-menu-item-icon {
  flex-shrink: 0;
}

.kb-menu-item-name {
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

.kb-menu-item-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--td-brand-color);
  flex-shrink: 0;
}

.kb-menu-loading,
.kb-menu-empty {
  padding: 8px 14px 6px var(--sidebar-inset-x);
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.kb-menu-loading {
  display: flex;
  align-items: center;
}
</style>
