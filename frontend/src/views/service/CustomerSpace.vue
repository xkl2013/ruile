<template>
  <div class="customer-space-page" :class="{ 'customer-space-page--mobile': isMobileEntry }">
    <main class="customer-space-shell">
      <header class="customer-space-header">
        <div class="customer-space-heading">
          <h2>客户空间</h2>
          <p>{{ customerSpaceSubtitle }}</p>
        </div>

        <div class="customer-space-header-tools">
          <div class="customer-space-count">
            <span>待处理</span>
            <strong>{{ totalOpenReminders }} 条</strong>
          </div>
          <button
            type="button"
            class="customer-space-icon-btn"
            :disabled="loading"
            title="刷新客户空间"
            aria-label="刷新客户空间"
            @click="refreshCustomerSpaces"
          >
            <t-icon :name="loading ? 'loading' : 'refresh'" />
          </button>
        </div>
      </header>

      <section class="customer-space-main">
        <aside class="customer-space-list-panel" aria-label="客户列表">
          <t-input
            v-model="keyword"
            class="customer-space-search"
            clearable
            placeholder="搜索客户或学员"
            size="small"
          >
            <template #prefix-icon>
              <t-icon name="search" />
            </template>
          </t-input>

          <div class="customer-space-list-meta">
            <span>{{ total }} 位客户</span>
            <em>{{ listStateLabel }}</em>
          </div>

          <div v-if="loading && spaces.length === 0" class="customer-space-skeleton-list">
            <div v-for="n in 5" :key="`space-skeleton-${n}`" class="customer-space-skeleton-row">
              <t-skeleton
                animation="gradient"
                :row-col="[
                  { width: '34px', height: '34px', type: 'rect' },
                  { width: '70%', height: '14px' },
                  { width: '52%', height: '12px' },
                ]"
              />
            </div>
          </div>

          <div v-else-if="spaces.length > 0" class="customer-space-list" role="list">
            <button
              v-for="space in spaces"
              :key="space.id"
              type="button"
              class="customer-space-list-item"
              :class="{ active: activeSubjectId === space.id }"
              @click="selectSpace(space.id)"
            >
              <span class="customer-space-avatar" aria-hidden="true">{{ customerInitial(space) }}</span>
              <span class="customer-space-list-copy">
                <span class="customer-space-title-row">
                  <strong>{{ space.name || space.display_name }}</strong>
                  <em>{{ formatDateLabel(space.latest_memory_at || space.updated_at) }}</em>
                </span>
                <span>{{ customerSpaceListDescription(space) }}</span>
                <span class="customer-space-list-tags">
                  <small v-if="space.open_reminder_count > 0" class="tag-open">
                    {{ space.open_reminder_count }} 条待处理
                  </small>
                  <small v-if="space.stage">{{ space.stage }}</small>
                  <small v-if="space.risk_label">{{ space.risk_label }}</small>
                </span>
              </span>
            </button>
          </div>

          <div v-else class="customer-space-empty-list">
            <t-icon name="folder-open" />
            <strong>暂无客户空间</strong>
            <p>从记忆里提取服务提醒后，这里会按客户沉淀文档。</p>
          </div>
        </aside>

        <section class="customer-space-detail-panel" aria-label="客户空间详情">
          <div v-if="activeSummary" class="customer-space-detail">
            <div class="customer-space-detail-header">
              <div class="customer-space-detail-title">
                <span class="customer-space-avatar customer-space-avatar--large" aria-hidden="true">
                  {{ customerInitial(activeSummary) }}
                </span>
                <div>
                  <h3>{{ activeSummary.name || activeSummary.display_name }}</h3>
                  <p>{{ activeSubjectLine }}</p>
                </div>
              </div>

              <div class="customer-space-detail-actions">
                <span class="customer-space-status" :class="statusClass(activeSummary.status)">
                  {{ statusLabel(activeSummary.status) }}
                </span>
              </div>
            </div>

            <div class="customer-space-stats">
              <div>
                <span>工作文档</span>
                <strong>{{ activeSummary.work_doc_count }}</strong>
              </div>
              <div>
                <span>服务提醒</span>
                <strong>{{ activeSummary.reminder_count }}</strong>
              </div>
              <div>
                <span>记忆证据</span>
                <strong>{{ activeSummary.source_memory_count }}</strong>
              </div>
            </div>

            <div class="customer-space-content-grid">
              <aside class="customer-space-doc-list" aria-label="客户空间文档">
                <div class="customer-space-section-head">
                  <span>客户空间文档</span>
                  <em>{{ orderedDocs.length }} 份</em>
                </div>
                <button
                  v-for="doc in orderedDocs"
                  :key="doc.id"
                  type="button"
                  class="customer-space-doc-item"
                  :class="{ active: activeDocId === doc.id }"
                  @click="activeDocId = doc.id"
                >
                  <t-icon :name="docIconName(doc)" />
                  <span>
                    <strong>{{ doc.title || docFileName(doc.doc_path) }}</strong>
                    <em>{{ doc.doc_path }}</em>
                  </span>
                </button>
                <div v-if="orderedDocs.length === 0" class="customer-space-inline-empty">暂无文档</div>
              </aside>

              <article class="customer-space-doc-preview">
                <template v-if="activeDoc">
                  <div class="customer-space-doc-preview-head">
                    <div>
                      <span>{{ docFileName(activeDoc.doc_path) }}</span>
                      <h4>{{ activeDoc.title || docFileName(activeDoc.doc_path) }}</h4>
                    </div>
                    <em>{{ formatDateLabel(activeDoc.updated_at) }}</em>
                  </div>
                  <div class="customer-space-markdown" v-html="renderedActiveDocContent" />
                </template>
                <div v-else class="customer-space-inline-empty customer-space-inline-empty--center">
                  选择左侧客户空间文档查看内容
                </div>
              </article>

              <aside class="customer-space-side-panel" aria-label="服务提醒和证据">
                <section class="customer-space-side-section">
                  <div class="customer-space-section-head">
                    <span>服务提醒</span>
                    <em>{{ activeReminders.length }} 条</em>
                  </div>
                  <div v-if="activeReminders.length > 0" class="customer-space-reminder-list">
                    <article
                      v-for="reminder in activeReminders"
                      :key="reminder.id"
                      class="customer-space-reminder"
                    >
                      <div>
                        <strong>{{ reminder.title }}</strong>
                        <span :class="priorityClass(reminder.priority)">{{ priorityLabel(reminder.priority) }}</span>
                      </div>
                      <p>{{ reminder.next_action || reminder.summary }}</p>
                      <em>{{ reminder.due_text || formatDateLabel(reminder.due_at || reminder.updated_at) }}</em>
                    </article>
                  </div>
                  <div v-else class="customer-space-inline-empty">暂无提醒</div>
                </section>

                <section class="customer-space-side-section">
                  <div class="customer-space-section-head">
                    <span>记忆证据</span>
                    <em>{{ activeEvidence.length }} 条</em>
                  </div>
                  <div v-if="activeEvidence.length > 0" class="customer-space-evidence-list">
                    <article
                      v-for="memory in activeEvidence"
                      :key="memory.id"
                      class="customer-space-evidence"
                    >
                      <strong>{{ memory.title }}</strong>
                      <p>{{ memory.summary }}</p>
                      <span>{{ memory.sourceLabel }} · {{ memory.occurredAtLabel }}</span>
                    </article>
                  </div>
                  <div v-else class="customer-space-inline-empty">暂无证据</div>
                </section>
              </aside>
            </div>
          </div>

          <div v-else-if="detailLoading" class="customer-space-detail-loading">
            <t-skeleton
              animation="gradient"
              :row-col="[
                { width: '38%', height: '24px' },
                { width: '62%', height: '14px' },
                { width: '100%', height: '320px', type: 'rect' },
              ]"
            />
          </div>

          <div v-else class="customer-space-empty-detail">
            <t-icon name="folder-open" />
            <strong>暂无可查看的客户空间</strong>
            <p>有服务相关记忆沉淀后，可在这里查看客户摘要、跟进记录和未闭环事项。</p>
          </div>
        </section>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  getServiceCustomerSpace,
  listServiceCustomerSpaces,
  refreshServiceModule,
  type AgentWorkDoc,
  type ServiceCustomerSpaceDTO,
  type ServiceCustomerSpaceDetailDTO,
  type ServiceMemoryEvidenceDTO,
  type ServiceReminderDTO,
} from '@/api/service'
import { sproutReportContentForEditor } from '../organize/sproutReport'

const route = useRoute()
const router = useRouter()

const keyword = ref('')
const spaces = ref<ServiceCustomerSpaceDTO[]>([])
const total = ref(0)
const loading = ref(false)
const loaded = ref(false)
const detailLoading = ref(false)
const activeSubjectId = ref('')
const activeDetail = ref<ServiceCustomerSpaceDetailDTO | null>(null)
const activeDocId = ref('')
let keywordTimer: number | null = null

const isMobileEntry = computed(() => route.meta.mobileEntry === true || route.path.startsWith('/mobile/'))
const activeSummary = computed(() =>
  activeDetail.value?.summary || spaces.value.find((item) => item.id === activeSubjectId.value) || null,
)
const activeReminders = computed(() => activeDetail.value?.reminders || [])
const activeEvidence = computed<ServiceMemoryEvidenceDTO[]>(() => activeDetail.value?.memory_evidence || [])
const totalOpenReminders = computed(() =>
  spaces.value.reduce((sum, item) => sum + (item.open_reminder_count || 0), 0),
)
const customerSpaceSubtitle = computed(() => {
  if (!loaded.value && loading.value) return '正在读取客户空间'
  if (total.value > 0) return `${total.value} 位客户 · ${totalOpenReminders.value} 条待处理`
  return '按客户查看摘要、跟进和未闭环事项'
})
const listStateLabel = computed(() => {
  if (loading.value) return '同步中'
  if (!loaded.value) return '待同步'
  if (total.value === 0) return '暂无数据'
  return '已同步'
})
const orderedDocs = computed(() => {
  const docs = activeDetail.value?.work_docs || []
  const order = ['客户摘要', '跟进记录', '未闭环事项', '证据索引']
  return [...docs].sort((a, b) => {
    const ai = order.findIndex((name) => (a.title || a.doc_path).includes(name))
    const bi = order.findIndex((name) => (b.title || b.doc_path).includes(name))
    const av = ai < 0 ? order.length : ai
    const bv = bi < 0 ? order.length : bi
    if (av !== bv) return av - bv
    return (a.doc_path || '').localeCompare(b.doc_path || '')
  })
})
const activeDoc = computed(() => orderedDocs.value.find((doc) => doc.id === activeDocId.value) || orderedDocs.value[0] || null)
const renderedActiveDocContent = computed(() => {
  const content = activeDoc.value?.content?.trim()
  if (!content) return '<p>暂无文档内容。</p>'
  return sproutReportContentForEditor(content)
})
const activeSubjectLine = computed(() => {
  const summary = activeSummary.value
  if (!summary) return ''
  return [
    summary.student_name ? `学员：${summary.student_name}` : '',
    summary.stage || '',
    summary.risk_label || '',
  ].filter(Boolean).join(' · ') || '客户服务记录'
})

const customerInitial = (space: ServiceCustomerSpaceDTO) => {
  const name = (space.name || space.display_name || space.student_name || '客').trim()
  return Array.from(name)[0] || '客'
}

const customerSpaceListDescription = (space: ServiceCustomerSpaceDTO) => {
  return space.summary || space.description || space.latest_action || '暂无客户摘要'
}

const formatDateLabel = (value?: string) => {
  if (!value) return '最近'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const docFileName = (path = '') => {
  const parts = path.split('/').filter(Boolean)
  return parts[parts.length - 1] || path || '客户空间文档'
}

const docIconName = (doc: AgentWorkDoc) => {
  const name = `${doc.title || ''}${doc.doc_path || ''}`
  if (name.includes('未闭环')) return 'flag'
  if (name.includes('证据')) return 'link'
  if (name.includes('跟进')) return 'time'
  return 'file'
}

const statusLabel = (status?: string) => {
  switch (status) {
    case 'completed':
    case 'confirmed':
      return '已闭环'
    case 'ignored':
      return '已忽略'
    case 'snoozed':
      return '稍后'
    case 'pending':
    case 'generated':
    case 'candidate':
      return '待处理'
    default:
      return '当前'
  }
}

const statusClass = (status?: string) => {
  if (status === 'completed' || status === 'confirmed') return 'is-done'
  if (status === 'ignored') return 'is-muted'
  return 'is-open'
}

const priorityLabel = (priority?: string) => {
  if (priority === 'high') return '高'
  if (priority === 'medium') return '中'
  return '低'
}

const priorityClass = (priority?: string) => `priority-${priority || 'low'}`

const loadCustomerSpaces = async (preferSubjectId = '') => {
  loading.value = true
  try {
    const res = await listServiceCustomerSpaces({
      keyword: keyword.value.trim() || undefined,
      page: 1,
      page_size: 50,
    })
    const data = res.data
    spaces.value = data?.items || []
    total.value = data?.total || spaces.value.length
    loaded.value = true
    const routeSubjectId = typeof route.params.subjectId === 'string' ? route.params.subjectId : ''
    const nextSubjectId = preferSubjectId || routeSubjectId || activeSubjectId.value || spaces.value[0]?.id || ''
    if (nextSubjectId && spaces.value.some((item) => item.id === nextSubjectId)) {
      await selectSpace(nextSubjectId, false)
    } else {
      activeSubjectId.value = ''
      activeDetail.value = null
      activeDocId.value = ''
    }
  } catch (error) {
    console.error('Load customer spaces failed:', error)
    MessagePlugin.error('客户空间读取失败')
  } finally {
    loading.value = false
  }
}

const loadCustomerSpaceDetail = async (id: string) => {
  if (!id) return
  detailLoading.value = true
  try {
    const res = await getServiceCustomerSpace(id)
    activeDetail.value = res.data
    activeDocId.value = orderedDocs.value[0]?.id || ''
  } catch (error) {
    console.error('Load customer space detail failed:', error)
    MessagePlugin.error('客户空间详情读取失败')
  } finally {
    detailLoading.value = false
  }
}

const selectSpace = async (id: string, syncRoute = true) => {
  if (!id) return
  activeSubjectId.value = id
  if (syncRoute) {
    const base = isMobileEntry.value ? '/mobile/service/customers' : '/platform/service/customers'
    if (route.path !== `${base}/${id}`) {
      await router.push(`${base}/${id}`)
    }
  }
  await loadCustomerSpaceDetail(id)
}

const refreshCustomerSpaces = async () => {
  try {
    await refreshServiceModule()
  } catch (error) {
    console.warn('Refresh service module failed, fallback to customer-space list:', error)
  }
  await loadCustomerSpaces(activeSubjectId.value)
}

watch(
  () => keyword.value,
  () => {
    if (keywordTimer) window.clearTimeout(keywordTimer)
    keywordTimer = window.setTimeout(() => {
      void loadCustomerSpaces()
    }, 260)
  },
)

watch(
  () => route.params.subjectId,
  (value) => {
    const id = typeof value === 'string' ? value : ''
    if (id && id !== activeSubjectId.value) {
      void selectSpace(id, false)
    }
  },
)

onMounted(() => {
  void loadCustomerSpaces()
})
</script>

<style scoped lang="less">
.customer-space-page {
  min-height: 100%;
  background: var(--td-bg-color-page);
}

.customer-space-shell {
  display: flex;
  min-height: 100vh;
  flex-direction: column;
  overflow: hidden;
}

.customer-space-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 88px;
  padding: 22px 28px 18px;
  border-bottom: 1px solid var(--td-component-border);
  background: var(--td-bg-color-container);
}

.customer-space-heading {
  min-width: 0;

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 22px;
    font-weight: 700;
    line-height: 30px;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.customer-space-header-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.customer-space-count {
  display: flex;
  flex-direction: column;
  gap: 2px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;

  strong {
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 600;
  }
}

.customer-space-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  transition: background 0.18s ease, border-color 0.18s ease, color 0.18s ease;

  &:hover:not(:disabled) {
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }
}

.customer-space-main {
  display: grid;
  min-height: 0;
  flex: 1;
  grid-template-columns: minmax(280px, 340px) minmax(0, 1fr);
  background: var(--td-bg-color-page);
}

.customer-space-list-panel {
  display: flex;
  min-height: 0;
  flex-direction: column;
  border-right: 1px solid var(--td-component-border);
  background: var(--td-bg-color-container);
}

.customer-space-search {
  margin: 16px 16px 10px;
}

.customer-space-list-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;

  em {
    font-style: normal;
  }
}

.customer-space-list,
.customer-space-skeleton-list {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 4px 8px 16px;
}

.customer-space-skeleton-row {
  padding: 12px 8px;
}

.customer-space-list-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  min-height: 92px;
  padding: 13px 12px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background 0.18s ease;

  &:hover,
  &.active {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.customer-space-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--td-brand-color) 14%, var(--td-bg-color-container));
  color: var(--td-brand-color);
  font-size: 16px;
  font-weight: 700;
}

.customer-space-avatar--large {
  width: 48px;
  height: 48px;
  flex-basis: 48px;
  border-radius: 10px;
  font-size: 20px;
}

.customer-space-list-copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 19px;

  > span:not(.customer-space-title-row):not(.customer-space-list-tags) {
    display: -webkit-box;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.customer-space-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;

  strong {
    min-width: 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 600;
    line-height: 21px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  em {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-style: normal;
  }
}

.customer-space-list-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;

  small {
    max-width: 100%;
    overflow: hidden;
    padding: 2px 6px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 11px;
    line-height: 16px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .tag-open {
    background: color-mix(in srgb, var(--td-warning-color) 14%, var(--td-bg-color-container));
    color: var(--td-warning-color);
  }
}

.customer-space-detail-panel {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 22px 24px 28px;
}

.customer-space-detail {
  display: flex;
  min-height: 100%;
  flex-direction: column;
  gap: 16px;
}

.customer-space-detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 4px 0 2px;
}

.customer-space-detail-title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 14px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 24px;
    font-weight: 700;
    line-height: 32px;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.customer-space-detail-actions {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.customer-space-status {
  display: inline-flex;
  align-items: center;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 600;

  &.is-open {
    background: color-mix(in srgb, var(--td-warning-color) 16%, var(--td-bg-color-container));
    color: var(--td-warning-color);
  }

  &.is-done {
    background: color-mix(in srgb, var(--td-success-color) 14%, var(--td-bg-color-container));
    color: var(--td-success-color);
  }
}

.customer-space-stats {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;

  div {
    display: flex;
    min-height: 68px;
    flex-direction: column;
    justify-content: center;
    gap: 4px;
    padding: 12px 14px;
    border: 1px solid var(--td-component-border);
    border-radius: 8px;
    background: var(--td-bg-color-container);
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 22px;
    line-height: 28px;
  }
}

.customer-space-content-grid {
  display: grid;
  min-height: 520px;
  flex: 1;
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr) minmax(260px, 320px);
  gap: 12px;
}

.customer-space-doc-list,
.customer-space-doc-preview,
.customer-space-side-panel {
  min-height: 0;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.customer-space-doc-list {
  display: flex;
  flex-direction: column;
  overflow: hidden;
  padding: 12px;
}

.customer-space-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 30px;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;

  em {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-style: normal;
    font-weight: 400;
  }
}

.customer-space-doc-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
  min-height: 58px;
  padding: 10px 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  text-align: left;
  transition: background 0.18s ease, color 0.18s ease;

  &:hover,
  &.active {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
  }

  > span {
    display: flex;
    min-width: 0;
    flex: 1;
    flex-direction: column;
    gap: 3px;
  }

  strong,
  em {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 18px;
  }

  em {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-style: normal;
    line-height: 17px;
  }
}

.customer-space-doc-preview {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
}

.customer-space-doc-preview-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 70px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--td-component-border);

  span,
  em {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-style: normal;
  }

  h4 {
    margin: 3px 0 0;
    color: var(--td-text-color-primary);
    font-size: 17px;
    font-weight: 700;
    line-height: 24px;
  }
}

.customer-space-markdown {
  min-height: 0;
  flex: 1;
  overflow: auto;
  padding: 20px 22px 28px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 1.72;

  :deep(h1),
  :deep(h2),
  :deep(h3) {
    margin: 18px 0 8px;
    color: var(--td-text-color-primary);
    font-weight: 700;
  }

  :deep(h1) {
    font-size: 22px;
  }

  :deep(h2) {
    font-size: 17px;
  }

  :deep(h3) {
    font-size: 15px;
  }

  :deep(p),
  :deep(ul),
  :deep(ol),
  :deep(table) {
    margin: 8px 0;
  }

  :deep(table) {
    width: 100%;
    border-collapse: collapse;
    font-size: 13px;
  }

  :deep(th),
  :deep(td) {
    padding: 8px 10px;
    border: 1px solid var(--td-component-border);
    text-align: left;
  }
}

.customer-space-side-panel {
  display: flex;
  flex-direction: column;
  gap: 0;
  overflow: hidden;
}

.customer-space-side-section {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  padding: 12px;

  & + & {
    border-top: 1px solid var(--td-component-border);
  }
}

.customer-space-reminder-list,
.customer-space-evidence-list {
  display: flex;
  min-height: 0;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  overflow: auto;
  padding-top: 6px;
}

.customer-space-reminder,
.customer-space-evidence {
  padding: 10px 11px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  background: var(--td-bg-color-page);

  strong {
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 18px;
  }

  p {
    display: -webkit-box;
    margin: 6px 0;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
  }

  em,
  span {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-style: normal;
  }
}

.customer-space-reminder {
  > div {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  span {
    display: inline-flex;
    align-items: center;
    min-height: 20px;
    padding: 0 6px;
    border-radius: 999px;
    font-weight: 600;
  }

  .priority-high {
    background: color-mix(in srgb, var(--td-error-color) 12%, var(--td-bg-color-container));
    color: var(--td-error-color);
  }

  .priority-medium {
    background: color-mix(in srgb, var(--td-warning-color) 12%, var(--td-bg-color-container));
    color: var(--td-warning-color);
  }

  .priority-low {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
  }
}

.customer-space-empty-list,
.customer-space-empty-detail,
.customer-space-detail-loading,
.customer-space-inline-empty {
  color: var(--td-text-color-secondary);
}

.customer-space-empty-list,
.customer-space-empty-detail {
  display: flex;
  flex: 1;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 28px;
  text-align: center;

  .t-icon {
    color: var(--td-text-color-placeholder);
    font-size: 30px;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 15px;
  }

  p {
    max-width: 280px;
    margin: 0;
    font-size: 13px;
    line-height: 20px;
  }
}

.customer-space-detail-loading {
  padding: 16px;
}

.customer-space-inline-empty {
  padding: 18px 8px;
  font-size: 13px;
  text-align: center;
}

.customer-space-inline-empty--center {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
}

@media (max-width: 1180px) {
  .customer-space-content-grid {
    grid-template-columns: minmax(210px, 240px) minmax(0, 1fr);
  }

  .customer-space-side-panel {
    grid-column: 1 / -1;
    min-height: 280px;
  }
}

@media (max-width: 840px) {
  .customer-space-shell {
    min-height: 100dvh;
  }

  .customer-space-header {
    min-height: 76px;
    padding: 16px 16px 12px;
  }

  .customer-space-heading h2 {
    font-size: 20px;
    line-height: 28px;
  }

  .customer-space-main {
    grid-template-columns: 1fr;
  }

  .customer-space-list-panel {
    max-height: 42dvh;
    border-right: 0;
    border-bottom: 1px solid var(--td-component-border);
  }

  .customer-space-detail-panel {
    padding: 16px;
  }

  .customer-space-content-grid {
    grid-template-columns: 1fr;
  }

  .customer-space-detail-header,
  .customer-space-stats {
    grid-template-columns: 1fr;
  }

  .customer-space-detail-header {
    align-items: flex-start;
  }
}

.customer-space-page--mobile {
  .customer-space-header {
    position: sticky;
    top: 0;
    z-index: 10;
  }
}
</style>
