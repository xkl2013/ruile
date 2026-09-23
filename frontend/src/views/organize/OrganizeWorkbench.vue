<template>
  <div class="organize-product-page">
    <main class="organize-product-scroll">
      <header class="organize-page-header">
        <div>
          <h1>整理</h1>
          <p>把记忆整理成便于查看的数据</p>
        </div>
        <div class="organize-header-art" aria-hidden="true">
          <div class="organize-art-window"><span /><span /><span /></div>
          <div class="organize-art-card"><span /><span /></div>
          <div class="organize-art-dot" />
        </div>
      </header>

      <div class="organize-create-row">
        <t-button theme="primary" class="organize-primary-button" @click="openCreateDialog()">
          <template #icon><t-icon name="add" /></template>
          新建整理
        </t-button>
      </div>

      <div v-if="runningCount" class="organize-running-banner">
        <t-icon name="loading" />
        <span>有 {{ runningCount }} 条整理正在生成，完成后会出现在下面。</span>
      </div>

      <section class="organize-board-section organize-list-main">
        <header class="organize-section-head">
          <span class="organize-section-title">我的整理</span>
          <div class="organize-section-tools">
            <t-select v-model="sortMode" class="organize-sort" size="small" :options="sortOptions" />
            <t-input v-model="configQuery" class="organize-search" size="small" placeholder="搜索整理">
              <template #prefix-icon><t-icon name="search" /></template>
            </t-input>
          </div>
        </header>

        <div v-if="filteredConfigs.length" class="organize-config-grid">
          <article
            v-for="config in filteredConfigs"
            :key="config.id"
            class="organize-config-card"
            :class="{ 'is-running': isActiveJob(latestJob(config)) }"
            role="button"
            tabindex="0"
            @click="openConfig(config.id)"
            @keydown.enter.self.prevent="openConfig(config.id)"
          >
            <div class="organize-card-title-row">
              <span class="organize-card-icon">
                <t-icon :name="templateFor(config)?.icon || 'dashboard'" />
              </span>
              <strong>{{ config.name }}</strong>
              <t-dropdown trigger="click" placement="bottom-right" @click.stop>
                <button type="button" class="organize-card-more" aria-label="更多操作" @click.stop>
                  <t-icon name="more" />
                </button>
                <template #dropdown>
                  <t-dropdown-menu>
                    <t-dropdown-item @click="openEditDialog(config)">设置</t-dropdown-item>
                    <t-dropdown-item @click="openConfig(config.id)">查看任务</t-dropdown-item>
                    <t-dropdown-item theme="error" @click="removeConfig(config)">删除</t-dropdown-item>
                  </t-dropdown-menu>
                </template>
              </t-dropdown>
            </div>

            <div class="organize-card-tags">
              <span class="organize-tag organize-tag--accent">{{ templateFor(config)?.name || '历史内容' }}</span>
              <span class="organize-tag">{{ scheduleLabel(config.schedule) }}</span>
              <span v-if="isActiveJob(latestJob(config))" class="organize-tag organize-tag--progress">
                {{ latestJob(config)?.stage }}
              </span>
              <span v-else-if="isFailedJob(latestJob(config))" class="organize-tag">执行失败</span>
              <span v-else-if="latestJob(config)" class="organize-tag organize-tag--success">
                {{ latestJob(config)?.state === 'fallback' ? '基础结果' : '已完成' }}
              </span>
              <span v-else class="organize-tag">待首次执行</span>
            </div>

            <p class="organize-card-description">
              {{ configDescription(config) }}
            </p>
          </article>
        </div>
        <div v-else class="organize-empty-state">
          <div>
            <h2>还没有整理</h2>
            <p>选一个模板开始，配置完成后可以随时调整。<br />也可以从空白配置创建整理。</p>
          </div>
          <t-button variant="outline" @click="openCreateDialog()">从空白创建</t-button>
        </div>
      </section>

      <section class="organize-board-section organize-template-section">
        <header class="organize-section-head organize-template-head">
          <span class="organize-section-title">从模板创建</span>
          <t-input v-model="templateQuery" class="organize-search" size="small" placeholder="搜索模板">
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
        </header>

        <div class="organize-template-groups">
          <section v-for="group in filteredTemplateGroups" :key="group.scene" class="organize-template-group">
            <div class="organize-template-group-label">{{ group.scene }}</div>
            <div class="organize-template-grid">
              <button
                v-for="template in group.items"
                :key="template.key"
                type="button"
                class="organize-template-card"
                @click="openCreateDialog(template.key)"
              >
                <div class="organize-template-top">
                  <span class="organize-template-icon">
                    <t-icon :name="template.icon" />
                  </span>
                  <strong>{{ template.name }}</strong>
                </div>
                <p>{{ template.description }}</p>
                <span class="organize-template-card-meta">产出 {{ template.outputLabel }} · 创建后可调整</span>
              </button>
            </div>
          </section>
        </div>
      </section>
    </main>

    <OrganizeConfigDialog
      v-model:visible="dialogVisible"
      :template-key="dialogTemplateKey"
      :config="editingConfig"
      @saved="handleConfigSaved"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  deleteOrganizeConfig,
  listOrganizeConfigs,
  listOrganizeTemplates,
} from '@/api/organize'
import OrganizeConfigDialog from './components/OrganizeConfigDialog.vue'
import {
  organizeScheduleLabels,
  toOrganizeConfig,
  toOrganizeTemplate,
  type OrganizeConfig,
  type OrganizeJob,
  type OrganizeTemplate,
} from './organizeWorkbenchState'

type SortMode = 'time' | 'todo'

const sortOptions = [
  { label: '按时间', value: 'time' },
  { label: '按待办数', value: 'todo' },
]

const route = useRoute()
const router = useRouter()
const configQuery = ref('')
const templateQuery = ref('')
const sortMode = ref<SortMode>('time')
const dialogVisible = ref(false)
const dialogTemplateKey = ref('')
const editingConfig = ref<OrganizeConfig | null>(null)
const handledQuery = ref('')
const configs = ref<OrganizeConfig[]>([])
const organizeTemplates = ref<OrganizeTemplate[]>([])
let refreshTimer: number | undefined

const runningCount = computed(() =>
  configs.value.reduce(
    (count, config) => count + config.jobs.filter((job) => isActiveJob(job)).length,
    0,
  ),
)

const templateFor = (config: OrganizeConfig) =>
  config.template || organizeTemplates.value.find((item) => item.key === config.templateKey)

const latestJob = (config: OrganizeConfig) => config.jobs[0]
const isActiveJob = (job?: OrganizeJob) =>
  Boolean(job && ['queued', 'running', 'repairing'].includes(job.state))
const isFailedJob = (job?: OrganizeJob) =>
  Boolean(job && ['failed', 'canceled'].includes(job.state))

const configDescription = (config: OrganizeConfig) => {
  const job = latestJob(config)
  if (isActiveJob(job)) return '正在按配置生成，完成后自动出现在任务列表'
  if (job?.state === 'failed') return job.errorMessage || '上次执行失败，可进入任务详情重试'
  return config.instruction || job?.summary || '尚未执行过，可手动发起或等待周期触发'
}

const scheduleLabel = (schedule: OrganizeConfig['schedule']) => organizeScheduleLabels[schedule]

const filteredConfigs = computed(() => {
  const query = configQuery.value.trim().toLowerCase()
  const list = configs.value.filter((config) => {
    if (!query) return true
    const job = latestJob(config)
    return [config.name, config.instruction, job?.summary, templateFor(config)?.name]
      .filter(Boolean)
      .join(' ')
      .toLowerCase()
      .includes(query)
  })

  if (sortMode.value === 'todo') {
    return [...list].sort((a, b) => (b.jobs[0]?.todoCount || 0) - (a.jobs[0]?.todoCount || 0))
  }
  return [...list].sort((a, b) => b.updatedOrder - a.updatedOrder)
})

const filteredTemplateGroups = computed(() => {
  const query = templateQuery.value.trim().toLowerCase()
  const groups = new Map<string, OrganizeTemplate[]>()
  organizeTemplates.value
    .filter((template) => {
      if (!query) return true
      return [template.name, template.description, template.scene, template.outputLabel]
        .join(' ')
        .toLowerCase()
        .includes(query)
    })
    .forEach((template) => {
      const group = groups.get(template.scene) || []
      group.push(template)
      groups.set(template.scene, group)
    })
  return Array.from(groups, ([scene, items]) => ({ scene, items }))
})

const openCreateDialog = (templateKey = '') => {
  editingConfig.value = null
  dialogTemplateKey.value = templateKey
  dialogVisible.value = true
}

const openEditDialog = (config: OrganizeConfig) => {
  editingConfig.value = config
  dialogTemplateKey.value = ''
  dialogVisible.value = true
}

const openConfig = async (configId: string) => {
  await router.push(`/platform/organize/configs/${encodeURIComponent(configId)}`)
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
        configs.value = configs.value.filter((item) => item.id !== config.id)
        MessagePlugin.success('整理配置已删除')
        dialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '整理配置删除失败')
      }
    },
  })
}

const handleConfigSaved = async (config: OrganizeConfig) => {
  MessagePlugin.success(`整理「${config.name}」已保存`)
  await loadWorkbench()
}

const loadWorkbench = async () => {
  try {
    const [templateResponse, configResponse] = await Promise.all([
      listOrganizeTemplates(),
      listOrganizeConfigs({ page: 1, page_size: 100 }),
    ])
    organizeTemplates.value = (templateResponse.data || []).map(toOrganizeTemplate)
    configs.value = (configResponse.data?.items || []).map(toOrganizeConfig)
    if (route.query.config === 'edit' && typeof route.query.configId === 'string') {
      const config = configs.value.find((item) => item.id === route.query.configId)
      if (config && !dialogVisible.value) openEditDialog(config)
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '整理工作台加载失败')
  }
}

onMounted(() => {
  void loadWorkbench()
  refreshTimer = window.setInterval(() => {
    if (runningCount.value > 0) void loadWorkbench()
  }, 5000)
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})

watch(
  () => [route.query.config, route.query.configId, route.query.template, route.fullPath].join('|'),
  () => {
    const queryKey = [
      String(route.query.config || ''),
      String(route.query.configId || ''),
      String(route.query.template || ''),
      route.fullPath,
    ].join('|')
    if (handledQuery.value === queryKey) return
    handledQuery.value = queryKey
    if (route.query.config === 'new') {
      openCreateDialog(typeof route.query.template === 'string' ? route.query.template : '')
      return
    }
    if (route.query.config === 'edit' && typeof route.query.configId === 'string') {
      const config = configs.value.find((item) => item.id === route.query.configId)
      if (config) openEditDialog(config)
    }
  },
  { immediate: true },
)

watch(dialogVisible, (visible) => {
  if (visible || (route.query.config !== 'new' && route.query.config !== 'edit')) return
  const query = { ...route.query }
  delete query.config
  delete query.configId
  delete query.template
  void router.replace({ path: '/platform/organize/hub', query })
})
</script>

<style scoped lang="less">
.organize-product-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  --organize-text: var(--td-text-color-primary);
  --organize-secondary: var(--td-text-color-secondary);
  --organize-muted: var(--td-text-color-placeholder);
  --organize-border: var(--td-component-stroke);
}

.organize-product-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 20px 28px 32px;
}

.organize-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  min-height: 60px;
  margin-bottom: 16px;
}

.organize-page-header h1 {
  margin: 0;
  color: var(--organize-text);
  font-size: 24px;
  font-weight: 600;
  line-height: 32px;
}

.organize-page-header p {
  margin: 4px 0 0;
  color: var(--organize-secondary);
  font-size: 13px;
  line-height: 20px;
}

.organize-header-art {
  position: relative;
  width: 124px;
  height: 54px;
  flex: none;
  opacity: 0.72;
}

.organize-art-window,
.organize-art-card {
  position: absolute;
  display: flex;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.organize-art-window {
  top: 10px;
  left: 2px;
  width: 44px;
  height: 32px;
  padding: 0 8px;
  flex-wrap: wrap;
}

.organize-art-window span,
.organize-art-card span {
  width: 4px;
  height: 4px;
  margin: 0;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
  background: transparent;
}

.organize-art-window span:nth-child(3) {
  width: 21px;
  height: 1px;
  border: 0;
  border-radius: 0;
  background: var(--td-component-border);
}

.organize-art-card {
  top: 13px;
  left: 58px;
  width: 42px;
  height: 26px;
  padding: 0 8px;
  border-color: var(--td-brand-color);
}

.organize-art-card span {
  display: block;
  background: var(--td-brand-color);
}

.organize-art-card span:last-child {
  width: 12px;
  height: 1px;
  border-radius: 0;
}

.organize-art-dot {
  position: absolute;
  top: 18px;
  right: 3px;
  width: 17px;
  height: 17px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
  background: var(--td-bg-color-container);
}

.organize-create-row {
  margin-bottom: 24px;
}

.organize-primary-button {
  background: var(--td-brand-color);
  border: 0;
  color: var(--td-text-color-anti);
}

.organize-primary-button:hover {
  background: var(--td-brand-color-hover);
}

.organize-running-banner {
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

.organize-running-banner :deep(.t-icon) {
  animation: organize-spin 1.1s linear infinite;
}

.organize-board-section {
  min-width: 0;
}

.organize-template-section {
  margin-top: 26px;
}

.organize-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.organize-section-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.organize-template-head {
  margin-top: 26px;
}

.organize-section-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.organize-sort {
  width: 132px;
}

.organize-search {
  width: 180px;
}

.organize-config-grid,
.organize-template-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.organize-config-card,
.organize-template-card {
  position: relative;
  min-width: 0;
  border: 1px solid var(--organize-border);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  text-align: left;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.organize-config-card {
  min-height: 137px;
  padding: 10px 11px;
  cursor: pointer;
}

.organize-config-card:hover,
.organize-template-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.04);
}

.organize-config-card.is-running {
  border-color: var(--td-brand-color);
}

.organize-card-title-row {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}

.organize-card-title-row strong {
  min-width: 0;
  overflow: hidden;
  color: var(--organize-text);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-card-icon,
.organize-template-icon {
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

.organize-template-icon {
  width: 22px;
  height: 22px;
}

.organize-card-icon.is-legacy {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--organize-muted);
}

.organize-card-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--organize-muted);
  cursor: pointer;
}

.organize-card-more:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.organize-card-tags,
.organize-template-card-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 8px;
}

.organize-tag {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 1px 5px;
  border: 0;
  border-radius: 3px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--organize-secondary);
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.organize-tag--accent {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.organize-tag--success {
  background: #eef9f3;
  color: #23805a;
}

.organize-tag--progress {
  background: #fff7e8;
  color: #a46715;
}

.organize-card-description {
  display: -webkit-box;
  min-height: 34px;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--organize-secondary);
  font-size: 12px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.organize-template-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.organize-template-group-label {
  margin-bottom: 6px;
  color: var(--organize-secondary);
  font-size: 12px;
  font-weight: 500;
}

.organize-template-card {
  display: flex;
  min-height: 108px;
  padding: 11px 12px;
  flex-direction: column;
  cursor: pointer;
  font: inherit;
}

.organize-template-top {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.organize-template-top strong {
  min-width: 0;
  overflow: hidden;
  color: var(--organize-text);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-template-card p {
  display: -webkit-box;
  min-height: 34px;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--organize-muted);
  font-size: 11px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.organize-template-card-meta {
  display: block;
  margin-top: 5px;
  color: var(--organize-muted);
  font-size: 11px;
  line-height: 16px;
}

.organize-empty-state {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-top: 6px;
  padding: 20px 22px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.organize-empty-state h2 {
  margin: 0;
  color: var(--organize-text);
  font-size: 14px;
  font-weight: 500;
}

.organize-empty-state p {
  margin: 6px 0 0;
  color: var(--organize-secondary);
  font-size: 13px;
  line-height: 22px;
}

@keyframes organize-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1100px) {
  .organize-config-grid,
  .organize-template-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 820px) {
  .organize-product-scroll {
    padding: 20px 20px 32px;
  }

  .organize-header-art {
    display: none;
  }

  .organize-config-grid,
  .organize-template-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 560px) {
  .organize-section-head {
    align-items: stretch;
    flex-direction: column;
  }

  .organize-section-tools {
    width: 100%;
  }

  .organize-sort,
  .organize-section-tools :deep(.t-input),
  .organize-template-head > :deep(.t-input) {
    width: 100%;
    flex: 1;
  }

  .organize-config-grid,
  .organize-template-grid {
    grid-template-columns: 1fr;
  }
}
</style>
