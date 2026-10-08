<template>
  <Teleport to="body">
    <Transition name="service-kb-picker">
      <div
        v-if="visible"
        class="service-kb-picker-overlay"
        role="presentation"
        @click.self="close"
      >
        <section
          class="service-kb-picker-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="service-kb-picker-title"
          @keydown.esc="close"
        >
          <header class="service-kb-picker-header">
            <div>
              <h2 id="service-kb-picker-title">选择知识库</h2>
              <p>当前账号有权访问的全部知识库，可多选</p>
            </div>
            <button type="button" class="service-kb-picker-close" aria-label="关闭" @click="close">
              <t-icon name="close" />
            </button>
          </header>

          <div class="service-kb-picker-toolbar">
            <t-input
              v-model="query"
              clearable
              placeholder="搜索知识库名称"
            >
              <template #prefix-icon><t-icon name="search" /></template>
            </t-input>
            <span>{{ filteredKnowledgeBases.length }} 个知识库</span>
          </div>

          <div class="service-kb-picker-body">
            <div v-if="loading" class="service-kb-picker-state">
              <t-icon name="loading" class="service-kb-picker-loading" />
              正在加载权限内的知识库
            </div>
            <div v-else-if="error" class="service-kb-picker-state service-kb-picker-state--error">
              <span>{{ error }}</span>
              <button type="button" @click="$emit('retry')">重试</button>
            </div>
            <div v-else-if="filteredKnowledgeBases.length" class="service-kb-picker-list">
              <button
                v-for="knowledgeBase in filteredKnowledgeBases"
                :key="knowledgeBase.id"
                type="button"
                class="service-kb-picker-option"
                :class="{ selected: draftSelectedIds.includes(String(knowledgeBase.id)) }"
                :aria-pressed="draftSelectedIds.includes(String(knowledgeBase.id))"
                @click="toggleKnowledgeBase(String(knowledgeBase.id))"
              >
                <KnowledgeBaseIcon
                  :icon="knowledgeBase.icon"
                  :icon-url="knowledgeBase.icon_url"
                  :type="knowledgeBase.type"
                  size="medium"
                />
                <span class="service-kb-picker-copy">
                  <span class="service-kb-picker-name">
                    <strong>{{ knowledgeBase.name }}</strong>
                    <em>{{ accessSourceLabel(knowledgeBase) }}</em>
                    <em>{{ permissionLabel(knowledgeBase) }}</em>
                  </span>
                  <small>{{ knowledgeBaseMeta(knowledgeBase) }}</small>
                </span>
                <span class="service-kb-picker-check">
                  <t-icon :name="draftSelectedIds.includes(String(knowledgeBase.id)) ? 'check' : 'add'" />
                </span>
              </button>
            </div>
            <div v-else class="service-kb-picker-state">
              {{ query ? '没有找到匹配的知识库' : '当前账号没有可访问的知识库' }}
            </div>
          </div>

          <footer class="service-kb-picker-footer">
            <span>{{ draftSelectedIds.length ? `已选择 ${draftSelectedIds.length} 个知识库` : '暂未选择知识库' }}</span>
            <div class="service-kb-picker-actions">
              <t-button variant="outline" size="medium" @click="close">取消</t-button>
              <t-button theme="primary" size="medium" @click="confirm">确定</t-button>
            </div>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'

type KnowledgeBaseOption = {
  id: string
  name: string
  icon?: string
  icon_url?: string
  type?: string
  description?: string
  knowledge_count?: number
  chunk_count?: number
  updated_at?: string
  access_source?: string
  my_permission?: string
  permission?: string
  org_name?: string
}

const props = withDefaults(defineProps<{
  visible: boolean
  knowledgeBases: KnowledgeBaseOption[]
  selectedIds: string[]
  loading?: boolean
  error?: string
}>(), {
  loading: false,
  error: '',
})

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  confirm: [ids: string[]]
  retry: []
}>()

const query = ref('')
const draftSelectedIds = ref<string[]>([])

const filteredKnowledgeBases = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase()
  if (!normalizedQuery) return props.knowledgeBases
  return props.knowledgeBases.filter((knowledgeBase) =>
    [
      knowledgeBase.name,
      knowledgeBase.description,
      knowledgeBase.org_name,
      accessSourceLabel(knowledgeBase),
      permissionLabel(knowledgeBase),
    ].filter(Boolean).join(' ').toLowerCase().includes(normalizedQuery))
})

const accessSourceLabel = (knowledgeBase: KnowledgeBaseOption) => {
  switch (knowledgeBase.access_source) {
    case 'created':
      return '我创建的'
    case 'shared_space':
    case 'shared_agent':
      return knowledgeBase.org_name || '共享给我'
    case 'subscription':
    case 'public_subscription':
      return '已订阅'
    case 'tenant_admin':
      return '工作空间'
    case 'system_admin':
      return '系统可见'
    default:
      return knowledgeBase.org_name || '有权访问'
  }
}

const permissionLabel = (knowledgeBase: KnowledgeBaseOption) => {
  switch (knowledgeBase.my_permission || knowledgeBase.permission) {
    case 'admin':
      return '可管理'
    case 'editor':
      return '可编辑'
    default:
      return '只读'
  }
}

const knowledgeBaseMeta = (knowledgeBase: KnowledgeBaseOption) => {
  const count = knowledgeBase.type === 'faq'
    ? knowledgeBase.chunk_count || 0
    : knowledgeBase.knowledge_count || 0
  const updatedAt = formatDate(knowledgeBase.updated_at)
  return `${count} 条内容${updatedAt ? ` · 更新于 ${updatedAt}` : ''}`
}

const formatDate = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}-${String(date.getDate()).padStart(2, '0')}`
}

const reset = () => {
  draftSelectedIds.value = [...props.selectedIds]
  query.value = ''
}

const close = () => {
  emit('update:visible', false)
}

const toggleKnowledgeBase = (knowledgeBaseId: string) => {
  draftSelectedIds.value = draftSelectedIds.value.includes(knowledgeBaseId)
    ? draftSelectedIds.value.filter((id) => id !== knowledgeBaseId)
    : [...draftSelectedIds.value, knowledgeBaseId]
}

const confirm = () => {
  emit('confirm', [...draftSelectedIds.value])
  close()
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) reset()
  },
  { immediate: true },
)
</script>

<style scoped lang="less">
.service-kb-picker-overlay {
  position: fixed;
  z-index: 3300;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.42);
}

.service-kb-picker-dialog {
  display: flex;
  width: min(720px, calc(100vw - 32px));
  max-height: min(680px, calc(100vh - 32px));
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  box-shadow: 0 20px 56px rgba(0, 0, 0, 0.18);
}

.service-kb-picker-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 22px 14px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-kb-picker-header h2,
.service-kb-picker-header p {
  margin: 0;
}

.service-kb-picker-header h2 {
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
}

.service-kb-picker-header p {
  margin-top: 2px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.service-kb-picker-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 18px;
}

.service-kb-picker-close:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-kb-picker-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 22px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-kb-picker-toolbar :deep(.t-input) {
  flex: 1;
  border-radius: 8px;
}

.service-kb-picker-toolbar > span {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-kb-picker-body {
  min-height: 240px;
  flex: 1;
  overflow-y: auto;
  padding: 8px 14px;
}

.service-kb-picker-list {
  display: grid;
  gap: 2px;
}

.service-kb-picker-option {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) 28px;
  align-items: center;
  width: 100%;
  min-height: 64px;
  gap: 12px;
  padding: 9px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.service-kb-picker-option:hover {
  background: var(--td-bg-color-secondarycontainer);
}

.service-kb-picker-option.selected {
  border-color: var(--td-brand-color-4);
  background: var(--td-brand-color-1);
}

.service-kb-picker-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-kb-picker-name {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 6px;
}

.service-kb-picker-name strong {
  min-width: 0;
  overflow: hidden;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-kb-picker-name em {
  flex: none;
  padding: 1px 5px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 10px;
  font-style: normal;
  line-height: 16px;
}

.service-kb-picker-option.selected .service-kb-picker-name em {
  background: var(--td-bg-color-container);
}

.service-kb-picker-copy small {
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-kb-picker-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 50%;
  color: var(--td-text-color-placeholder);
}

.service-kb-picker-option.selected .service-kb-picker-check {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.service-kb-picker-state {
  display: flex;
  min-height: 240px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.service-kb-picker-state--error {
  flex-direction: column;
}

.service-kb-picker-state--error button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color-7);
  cursor: pointer;
  font: inherit;
}

.service-kb-picker-loading {
  animation: service-kb-picker-spin 0.9s linear infinite;
}

@keyframes service-kb-picker-spin {
  to {
    transform: rotate(360deg);
  }
}

.service-kb-picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 22px 14px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-kb-picker-footer > span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-kb-picker-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-kb-picker-actions :deep(.t-button) {
  min-width: 76px;
  border-radius: 8px;
}

.service-kb-picker-enter-active,
.service-kb-picker-leave-active {
  transition: opacity 0.18s ease;
}

.service-kb-picker-enter-active .service-kb-picker-dialog,
.service-kb-picker-leave-active .service-kb-picker-dialog {
  transition: transform 0.18s ease;
}

.service-kb-picker-enter-from,
.service-kb-picker-leave-to {
  opacity: 0;
}

.service-kb-picker-enter-from .service-kb-picker-dialog,
.service-kb-picker-leave-to .service-kb-picker-dialog {
  transform: translateY(10px) scale(0.985);
}

@media (max-width: 620px) {
  .service-kb-picker-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .service-kb-picker-dialog {
    width: 100%;
    max-height: 90vh;
    border-radius: 12px 12px 0 0;
  }

  .service-kb-picker-header,
  .service-kb-picker-toolbar,
  .service-kb-picker-footer {
    padding-right: 16px;
    padding-left: 16px;
  }

  .service-kb-picker-toolbar {
    align-items: stretch;
    flex-direction: column;
    gap: 6px;
  }

  .service-kb-picker-name {
    flex-wrap: wrap;
  }

  .service-kb-picker-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .service-kb-picker-actions {
    justify-content: flex-end;
  }
}
</style>
