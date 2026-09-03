<template>
  <section class="admin-kb-page">
    <header class="admin-kb-page__header">
      <div class="admin-kb-summary">
        <article>
          <span>知识库</span>
          <strong>{{ knowledgeBases.length }}</strong>
          <em>当前空间</em>
        </article>
        <article>
          <span>文档库</span>
          <strong>{{ documentCount }}</strong>
          <em>资料和 Wiki</em>
        </article>
        <article>
          <span>FAQ 库</span>
          <strong>{{ faqCount }}</strong>
          <em>结构化问答</em>
        </article>
        <article :class="{ 'is-warning': uninitializedCount > 0 }">
          <span>待配置</span>
          <strong>{{ uninitializedCount }}</strong>
          <em>模型或索引未完整</em>
        </article>
      </div>
    </header>

    <section class="admin-kb-panel">
      <div class="admin-kb-panel__toolbar">
        <div>
          <h2>知识库资源</h2>
          <p>只管理后台配置和资源治理；阅读、问答、上传和 wiki 浏览仍在主工作台。</p>
        </div>
        <div class="admin-kb-toolbar-actions">
          <t-input
            v-model.trim="keyword"
            class="admin-kb-search"
            clearable
            placeholder="按名称、描述或创建者搜索"
          >
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
          <t-select v-model="typeFilter" class="admin-kb-type-filter">
            <t-option value="all" label="全部类型" />
            <t-option value="document" label="文档库" />
            <t-option value="faq" label="FAQ 库" />
          </t-select>
          <t-button
            v-if="canSortKnowledgeBases"
            :theme="sortMode ? 'primary' : 'default'"
            variant="outline"
            :disabled="knowledgeBases.length < 2 || isReorderFiltered"
            @click="toggleSortMode"
          >
            <template #icon><t-icon name="order-adjustment-column" /></template>
            {{ sortMode ? '退出排序' : '排序' }}
          </t-button>
          <t-button
            theme="primary"
            :disabled="!canCreateKnowledgeBase"
            @click="openCreate()"
          >
            <template #icon><t-icon name="folder-add" /></template>
            新建知识库
          </t-button>
        </div>
      </div>

      <t-alert
        v-if="!canCreateKnowledgeBase"
        theme="warning"
        variant="light"
        message="当前角色只能查看知识库后台状态，创建、删除和排序需要空间 Admin 或系统管理员。"
        class="admin-kb-alert"
      />

      <t-alert
        v-if="isReorderFiltered"
        theme="info"
        variant="light"
        message="当前处于筛选状态，排序按钮已隐藏；清空搜索和类型筛选后可调整完整列表顺序。"
        class="admin-kb-alert"
      />

      <t-alert
        v-if="sortMode && !isReorderFiltered"
        theme="success"
        variant="light"
        message="排序模式已开启。使用操作列中的置顶、上移、下移、置底调整知识库顺序，调整后会立即保存。"
        class="admin-kb-alert"
      />

      <div class="admin-kb-table-shell">
        <t-table
          row-key="id"
          :data="filteredKnowledgeBases"
          :columns="displayColumns"
          :loading="loading"
          :hover="true"
          table-layout="fixed"
        >
          <template #manual_order="{ rowIndex }">
            <div class="admin-kb-order-cell">
              <span>{{ rowIndex + 1 }}</span>
            </div>
          </template>

          <template #name="{ row }">
            <div class="admin-kb-name-cell">
              <KnowledgeBaseIcon
                :icon="row.icon"
                :icon-url="row.icon_url"
                :type="row.type"
                size="large"
              />
              <span>
                <strong :title="row.name">{{ row.name || '-' }}</strong>
                <small :title="row.description || ''">{{ row.description || '暂无描述' }}</small>
              </span>
            </div>
          </template>

          <template #type="{ row }">
            <t-tag :theme="row.type === 'faq' ? 'primary' : 'success'" variant="light">
              {{ row.type === 'faq' ? 'FAQ' : '文档' }}
            </t-tag>
          </template>

          <template #counts="{ row }">
            <span class="admin-kb-count">
              {{ row.type === 'faq' ? row.chunk_count || 0 : row.knowledge_count || 0 }}
            </span>
          </template>

          <template #health="{ row }">
            <t-tag :theme="isInitialized(row) ? 'success' : 'warning'" variant="light">
              {{ isInitialized(row) ? '已配置' : '待配置' }}
            </t-tag>
            <div class="admin-kb-indexing">
              {{ indexingSummary(row) }}
            </div>
          </template>

          <template #vector_store="{ row }">
            <VectorStoreBadge
              :source="row.vector_store_source"
              :name="row.vector_store_name"
              :engine-type="row.vector_store_engine_type"
              :status="row.vector_store_status"
            />
          </template>

          <template #updated_at="{ row }">
            <span class="admin-kb-date">{{ formatDate(row.updated_at || row.created_at) }}</span>
          </template>

          <template #operation="{ row, rowIndex }">
            <div class="admin-kb-actions">
              <template v-if="!sortMode">
                <t-tooltip content="后台设置" placement="top">
                  <t-button
                    size="small"
                    theme="primary"
                    variant="outline"
                    class="admin-kb-config-btn"
                    :disabled="!canConfigure(row)"
                    @click="openSettings(row)"
                  >
                    <template #icon><t-icon name="setting" /></template>
                    配置
                  </t-button>
                </t-tooltip>
                <t-tooltip content="打开工作台" placement="top">
                  <t-button
                    size="small"
                    variant="text"
                    shape="square"
                    @click="openWorkspace(row)"
                  >
                    <template #icon><t-icon name="jump" /></template>
                  </t-button>
                </t-tooltip>
              </template>
              <div v-if="sortMode && canSortKnowledgeBases && !isReorderFiltered" class="admin-kb-order-actions">
                <t-tooltip content="置顶" placement="top">
                  <t-button
                    size="small"
                    variant="text"
                    shape="square"
                    :disabled="rowIndex === 0 || reorderingId === row.id"
                    @click="moveKnowledgeBaseTo(row.id, 0)"
                  >
                    <template #icon><t-icon :name="reorderingId === row.id ? 'loading' : 'chevron-up-double'" /></template>
                  </t-button>
                </t-tooltip>
                <t-tooltip content="上移" placement="top">
                  <t-button
                    size="small"
                    variant="text"
                    shape="square"
                    :disabled="rowIndex === 0 || reorderingId === row.id"
                    @click="moveKnowledgeBase(row.id, -1)"
                  >
                    <template #icon><t-icon :name="reorderingId === row.id ? 'loading' : 'chevron-up'" /></template>
                  </t-button>
                </t-tooltip>
                <t-tooltip content="下移" placement="top">
                  <t-button
                    size="small"
                    variant="text"
                    shape="square"
                    :disabled="rowIndex === filteredKnowledgeBases.length - 1 || reorderingId === row.id"
                    @click="moveKnowledgeBase(row.id, 1)"
                  >
                    <template #icon><t-icon :name="reorderingId === row.id ? 'loading' : 'chevron-down'" /></template>
                  </t-button>
                </t-tooltip>
                <t-tooltip content="置底" placement="top">
                  <t-button
                    size="small"
                    variant="text"
                    shape="square"
                    :disabled="rowIndex === filteredKnowledgeBases.length - 1 || reorderingId === row.id"
                    @click="moveKnowledgeBaseTo(row.id, knowledgeBases.length - 1)"
                  >
                    <template #icon><t-icon :name="reorderingId === row.id ? 'loading' : 'chevron-down-double'" /></template>
                  </t-button>
                </t-tooltip>
              </div>
              <t-dropdown
                v-if="!sortMode && canConfigure(row)"
                trigger="click"
                :options="actionOptions(row)"
                @click="(data: { value: unknown }) => handleRowAction(row, data.value)"
              >
                <t-button size="small" variant="text" shape="square">
                  <template #icon><t-icon name="ellipsis" /></template>
                </t-button>
              </t-dropdown>
            </div>
          </template>

          <template #empty>
            <div class="admin-kb-empty">
              <t-icon name="folder-open" />
              <span>{{ loading ? '加载中' : '暂无知识库' }}</span>
            </div>
          </template>
        </t-table>
      </div>
    </section>

    <KnowledgeBaseEditorModal
      :visible="createVisible"
      mode="create"
      :initial-type="createInitialType"
      @update:visible="createVisible = $event"
      @success="handleCreateSuccess"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  deleteKnowledgeBase,
  listKnowledgeBases,
  reorderKnowledgeBases,
} from '@/api/knowledge-base'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import VectorStoreBadge from '@/components/VectorStoreBadge.vue'
import KnowledgeBaseEditorModal from '@/views/knowledge/KnowledgeBaseEditorModal.vue'
import { useAuthStore } from '@/stores/auth'
import { useChatResourcesStore } from '@/stores/chatResources'
import { openMainAppPath } from '@admin/utils/navigation'

type KnowledgeBaseRow = {
  id: string
  name: string
  description?: string
  icon?: string
  icon_url?: string
  type?: 'document' | 'faq' | string
  creator_id?: string
  creator_name?: string
  created_at?: string
  updated_at?: string
  knowledge_count?: number
  chunk_count?: number
  embedding_model_id?: string
  summary_model_id?: string
  indexing_strategy?: {
    vector_enabled?: boolean
    keyword_enabled?: boolean
    wiki_enabled?: boolean
    graph_enabled?: boolean
  }
  vector_store_source?: 'env' | 'user' | 'shared' | 'unavailable'
  vector_store_name?: string
  vector_store_engine_type?: string
  vector_store_status?: 'available' | 'unavailable'
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const chatResources = useChatResourcesStore()

const knowledgeBases = ref<KnowledgeBaseRow[]>([])
const loading = ref(false)
const createVisible = ref(false)
const createInitialType = ref<'document' | 'faq'>('document')
const keyword = ref('')
const typeFilter = ref<'all' | 'document' | 'faq'>('all')
const reorderingId = ref('')
const sortMode = ref(false)

const canCreateKnowledgeBase = computed(() => authStore.hasRole('admin') || authStore.isSystemAdmin)
const canSortKnowledgeBases = computed(() => authStore.hasRole('admin') || authStore.isSystemAdmin)
const isReorderFiltered = computed(() => Boolean(keyword.value.trim()) || typeFilter.value !== 'all')

const columns = computed(() => [
  { colKey: 'name', title: '知识库', minWidth: 280 },
  { colKey: 'type', title: '类型', width: 92 },
  { colKey: 'counts', title: '内容量', width: 92, align: 'right' },
  { colKey: 'health', title: '配置状态', minWidth: 180 },
  { colKey: 'vector_store', title: '向量库', minWidth: 180 },
  { colKey: 'updated_at', title: '更新时间', width: 156 },
  { colKey: 'operation', title: sortMode.value ? '排序操作' : '操作', width: 176, fixed: 'right', cell: 'operation' },
])

const displayColumns = computed(() => {
  if (!sortMode.value || !canSortKnowledgeBases.value || isReorderFiltered.value) return columns.value
  return [
    { colKey: 'manual_order', title: '顺序', width: 72, cell: 'manual_order' },
    ...columns.value,
  ]
})

watch([keyword, typeFilter], () => {
  if (isReorderFiltered.value) sortMode.value = false
})

const filteredKnowledgeBases = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return knowledgeBases.value.filter((kb) => {
    if (typeFilter.value !== 'all' && (kb.type || 'document') !== typeFilter.value) return false
    if (!kw) return true
    return [kb.name, kb.description, kb.creator_name, kb.id]
      .some((value) => String(value || '').toLowerCase().includes(kw))
  })
})

const documentCount = computed(() => knowledgeBases.value.filter((kb) => (kb.type || 'document') !== 'faq').length)
const faqCount = computed(() => knowledgeBases.value.filter((kb) => kb.type === 'faq').length)
const uninitializedCount = computed(() => knowledgeBases.value.filter((kb) => !isInitialized(kb)).length)

function isInitialized(kb: KnowledgeBaseRow): boolean {
  if (!kb.summary_model_id) return false
  const strategy = kb.indexing_strategy
  const needsEmbedding = !strategy || strategy.vector_enabled !== false || strategy.keyword_enabled !== false
  return !needsEmbedding || Boolean(kb.embedding_model_id)
}

function indexingSummary(kb: KnowledgeBaseRow): string {
  if (kb.type === 'faq') return 'FAQ 问答索引'
  const strategy = kb.indexing_strategy
  if (!strategy) return '向量检索、关键词检索'
  const items: string[] = []
  if (strategy.vector_enabled) items.push('向量')
  if (strategy.keyword_enabled) items.push('关键词')
  if (strategy.wiki_enabled) items.push('Wiki')
  if (strategy.graph_enabled) items.push('图谱')
  return items.length ? items.join('、') : '未启用索引'
}

function canConfigure(kb: KnowledgeBaseRow): boolean {
  if (authStore.isSystemAdmin || authStore.hasRole('admin')) return true
  const userId = authStore.user?.id || ''
  return Boolean(kb.creator_id && userId && kb.creator_id === userId)
}

function actionOptions(kb: KnowledgeBaseRow) {
  const options: Array<{ content: string; value: string; theme?: 'default' | 'error'; disabled?: boolean }> = [
    { content: '删除知识库', value: 'delete', theme: 'error', disabled: !canConfigure(kb) },
  ]
  return options
}

async function fetchList() {
  loading.value = true
  try {
    const res: any = await listKnowledgeBases({ creator: 'all' })
    knowledgeBases.value = Array.isArray(res?.data) ? res.data : []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库列表加载失败')
    knowledgeBases.value = []
  } finally {
    loading.value = false
  }
}

function normalizeCreateType(value: unknown): 'document' | 'faq' {
  const raw = Array.isArray(value) ? value[0] : value
  return raw === 'faq' ? 'faq' : 'document'
}

function openCreate(type: 'document' | 'faq' = 'document') {
  if (!canCreateKnowledgeBase.value) return
  createInitialType.value = type
  createVisible.value = true
}

function toggleSortMode() {
  if (!canSortKnowledgeBases.value || knowledgeBases.value.length < 2) return
  if (isReorderFiltered.value) {
    MessagePlugin.info('请先清空搜索和类型筛选，再调整知识库顺序')
    return
  }
  sortMode.value = !sortMode.value
}

function openSettings(row: KnowledgeBaseRow) {
  if (!canConfigure(row)) return
  void router.push({
    name: 'adminKnowledgeBaseSettings',
    params: { kbId: row.id },
  })
}

function openWorkspace(row: KnowledgeBaseRow) {
  openMainAppPath(`/platform/knowledge-bases/${row.id}`)
}

async function handleCreateSuccess(kbId: string) {
  createVisible.value = false
  chatResources.invalidate('knowledgeBases')
  await fetchList()
  if (kbId) {
    void router.push({
      name: 'adminKnowledgeBaseSettings',
      params: { kbId },
    })
  }
}

function deleteKB(row: KnowledgeBaseRow) {
  if (!canConfigure(row)) return
  const dialog = DialogPlugin.confirm({
    header: '删除知识库',
    body: `确认删除「${row.name || row.id}」？该操作会影响知识库内容和相关配置。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: { content: '取消' },
    onConfirm: async () => {
      dialog.destroy()
      try {
        const res: any = await deleteKnowledgeBase(row.id)
        if (!res?.success) {
          throw new Error(res?.message || '删除知识库失败')
        }
        MessagePlugin.success('知识库已删除')
        chatResources.invalidate('knowledgeBases')
        await fetchList()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '删除知识库失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

function handleRowAction(row: KnowledgeBaseRow, value: unknown) {
  if (value === 'delete') {
    deleteKB(row)
  }
}

async function moveKnowledgeBase(kbId: string, offset: -1 | 1) {
  if (!canSortKnowledgeBases.value || isReorderFiltered.value || reorderingId.value) return
  const currentIndex = knowledgeBases.value.findIndex((kb) => kb.id === kbId)
  const nextIndex = currentIndex + offset
  if (currentIndex < 0 || nextIndex < 0 || nextIndex >= knowledgeBases.value.length) return

  await moveKnowledgeBaseTo(kbId, nextIndex)
}

async function moveKnowledgeBaseTo(kbId: string, targetIndex: number) {
  if (!canSortKnowledgeBases.value || isReorderFiltered.value || reorderingId.value) return
  const currentIndex = knowledgeBases.value.findIndex((kb) => kb.id === kbId)
  const normalizedTargetIndex = Math.max(0, Math.min(targetIndex, knowledgeBases.value.length - 1))
  if (currentIndex < 0 || currentIndex === normalizedTargetIndex) return

  const next = [...knowledgeBases.value]
  const [item] = next.splice(currentIndex, 1)
  next.splice(normalizedTargetIndex, 0, item)

  reorderingId.value = kbId
  const previous = knowledgeBases.value
  knowledgeBases.value = next
  try {
    const res: any = await reorderKnowledgeBases(next.map((kb) => kb.id))
    if (Array.isArray(res?.data)) {
      knowledgeBases.value = res.data
    }
    MessagePlugin.success('排序已保存')
    chatResources.invalidate('knowledgeBases')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '排序保存失败')
    knowledgeBases.value = previous
    await fetchList()
  } finally {
    reorderingId.value = ''
  }
}

function formatDate(value?: string): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function consumeCreateQuery() {
  if (route.query.create !== '1') return
  openCreate(normalizeCreateType(route.query.type))
  const { create: _create, type: _type, ...rest } = route.query
  void router.replace({ query: rest })
}

onMounted(async () => {
  await fetchList()
  consumeCreateQuery()
})

watch(() => route.query.create, consumeCreateQuery)
</script>

<style scoped lang="less">
.admin-kb-page {
  display: grid;
  gap: 18px;
  max-width: 1280px;
}

.admin-kb-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;

  article {
    display: grid;
    gap: 4px;
    min-width: 0;
    padding: 16px;
    border: 1px solid #dde6ed;
    border-radius: 8px;
    background: #fff;

    span,
    em {
      color: var(--td-text-color-secondary);
      font-size: 12px;
      font-style: normal;
      line-height: 1.4;
    }

    strong {
      color: var(--td-text-color-primary);
      font-size: 24px;
      font-weight: 650;
      line-height: 1.2;
    }

    &.is-warning strong {
      color: var(--td-warning-color);
    }
  }
}

.admin-kb-panel {
  min-width: 0;
  padding: 18px;
  border: 1px solid #dde6ed;
  border-radius: 8px;
  background: #fff;
}

.admin-kb-panel__toolbar {
  display: flex;
  gap: 16px;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 14px;

  h2 {
    margin: 0 0 4px;
    font-size: 17px;
    font-weight: 650;
    line-height: 1.35;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
}

.admin-kb-toolbar-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 10px;
  align-items: center;
}

.admin-kb-search {
  width: 260px;
}

.admin-kb-type-filter {
  width: 120px;
}

.admin-kb-alert {
  margin-bottom: 12px;
}

.admin-kb-table-shell {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.admin-kb-name-cell {
  display: flex;
  gap: 10px;
  align-items: center;
  min-width: 0;

  span {
    display: grid;
    gap: 2px;
    min-width: 0;
  }

  strong,
  small {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    font-size: 14px;
    font-weight: 650;
    line-height: 1.4;
  }

  small {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 1.35;
  }
}

.admin-kb-count {
  font-variant-numeric: tabular-nums;
}

.admin-kb-indexing {
  margin-top: 6px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}

.admin-kb-date {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.admin-kb-order-cell {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 24px;
  border-radius: 6px;
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}

.admin-kb-actions {
  display: inline-flex;
  gap: 6px;
  align-items: center;
  min-width: 0;
  white-space: nowrap;
}

.admin-kb-order-actions {
  display: inline-flex;
  gap: 2px;
  align-items: center;
}

.admin-kb-config-btn {
  min-width: 68px;
}

.admin-kb-empty {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 128px;
  color: var(--td-text-color-placeholder);
}

@media (max-width: 960px) {
  .admin-kb-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .admin-kb-panel__toolbar,
  .admin-kb-toolbar-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .admin-kb-search,
  .admin-kb-type-filter {
    width: 100%;
  }
}
</style>
