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

    <div v-show="isExpanded" :id="organizeMenuListId" class="organize-menu-list organize-menu-list--nested">
      <div v-for="config in configuredOrganizeItems" :key="config.id"
        class="organize-menu-item" :class="{ active: isConfigActive(config) }"
        role="button" tabindex="0"
        :aria-current="isConfigActive(config) ? 'page' : undefined"
        @click="openConfig(config.id)"
        @keydown.enter.self.prevent="openConfig(config.id)"
        @keydown.space.self.prevent="openConfig(config.id)">
        <t-icon :name="configIcon(config)" class="organize-menu-item-icon" />
        <span class="organize-menu-item-name" :title="config.name">{{ config.name }}</span>
        <span class="organize-menu-item-actions" @click.stop>
          <t-tooltip content="设置" placement="top">
            <button
              type="button"
              class="organize-menu-item-action"
              :aria-label="`设置${config.name}`"
              @click.stop="openConfigSettings(config)"
            >
              <t-icon name="setting" size="14px" />
            </button>
          </t-tooltip>
          <t-tooltip content="删除" placement="top">
            <button
              type="button"
              class="organize-menu-item-action organize-menu-item-action--danger"
              :aria-label="`删除${config.name}`"
              @click.stop="removeConfig(config)"
            >
              <t-icon name="delete" size="14px" />
            </button>
          </t-tooltip>
        </span>
      </div>
    </div>

    <OrganizeConfigDialog
      v-model:visible="configDialogVisible"
      :config="editingConfig"
      @saved="handleConfigSaved"
    />
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { deleteOrganizeConfig, listOrganizeConfigs } from '@/api/organize'
import OrganizeConfigDialog from '@/views/organize/components/OrganizeConfigDialog.vue'
import {
  ORGANIZE_ROUTE_BASE_PATH,
  ORGANIZE_ROUTE_NAMES,
} from '@/views/organize/organizeRoutes'
import {
  toOrganizeConfig,
  type OrganizeConfig,
} from '@/views/organize/organizeWorkbenchState'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()

const organizeMenuListId = 'organize-menu-list'
const ORGANIZE_MENU_EXPANDED_STORAGE_KEY = 'sidebar-organize-menu-expanded'
const organizeWorkbenchPath = `${ORGANIZE_ROUTE_BASE_PATH}/hub`
const configuredOrganizeItems = ref<OrganizeConfig[]>([])
const configDialogVisible = ref(false)
const editingConfig = ref<OrganizeConfig | null>(null)
let refreshTimer: number | undefined

const loadExpandedState = () => {
  if (typeof window === 'undefined') return true
  try {
    return window.localStorage.getItem(ORGANIZE_MENU_EXPANDED_STORAGE_KEY) !== 'false'
  } catch {
    return true
  }
}

const isExpanded = ref(loadExpandedState())

const isConfigActive = (config: OrganizeConfig) => {
  if (route.name === ORGANIZE_ROUTE_NAMES.configDetail) {
    return String(route.params.configId || '') === config.id
  }
  return route.name === ORGANIZE_ROUTE_NAMES.outputDetail &&
    String(route.query.configId || '') === config.id
}

const configIcon = (config: OrganizeConfig) =>
  config.template?.icon || 'dashboard'

const loadConfiguredItems = async () => {
  try {
    const response = await listOrganizeConfigs({ page: 1, page_size: 100, status: 'active' })
    configuredOrganizeItems.value = (response.data?.items || []).map(toOrganizeConfig)
  } catch {
    configuredOrganizeItems.value = []
  }
}

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
  await openRoute(organizeWorkbenchPath)
}

const openConfig = async (configId: string) => {
  await openRoute(`${ORGANIZE_ROUTE_BASE_PATH}/configs/${encodeURIComponent(configId)}`)
}

const openConfigSettings = (config: OrganizeConfig) => {
  editingConfig.value = config
  configDialogVisible.value = true
}

const handleConfigSaved = async (config: OrganizeConfig) => {
  MessagePlugin.success(`整理「${config.name}」已保存`)
  await loadConfiguredItems()
}

const removeConfig = (config: OrganizeConfig) => {
  const dialog = DialogPlugin.confirm({
    header: '删除整理配置',
    body: `确认删除「${config.name}」？已生成的整理结果会继续保留。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: '取消',
    onConfirm: async () => {
      try {
        await deleteOrganizeConfig(config.id)
        configuredOrganizeItems.value = configuredOrganizeItems.value.filter((item) => item.id !== config.id)
        if (isConfigActive(config)) {
          await router.push(organizeWorkbenchPath)
        }
        MessagePlugin.success('整理配置已删除')
        dialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '整理配置删除失败')
      }
    },
  })
}

onMounted(() => {
  void loadConfiguredItems()
  refreshTimer = window.setInterval(loadConfiguredItems, 30000)
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})

watch(
  () => route.fullPath,
  () => void loadConfiguredItems(),
)
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

.organize-menu-item-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
  flex: 0 0 42px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.18s ease;
}

.organize-menu-item:hover .organize-menu-item-actions,
.organize-menu-item:focus-within .organize-menu-item-actions {
  opacity: 1;
  pointer-events: auto;
}

.organize-menu-item-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background-color 0.18s ease, color 0.18s ease;
}

.organize-menu-item-action:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.organize-menu-item-action--danger:hover {
  background: var(--td-error-color-light);
  color: var(--td-error-color);
}

</style>
