<template>
  <Teleport to="body">
    <Transition name="service-expert-picker">
      <div
        v-if="visible"
        class="service-expert-picker-overlay"
        role="presentation"
        @click.self="close"
      >
        <section
          class="service-expert-picker-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="service-expert-picker-title"
          @keydown.esc="close"
        >
          <header class="service-expert-picker-header">
            <h2 id="service-expert-picker-title">选择专家</h2>
            <div class="service-expert-picker-header-tools">
              <t-input
                v-model="query"
                class="service-expert-picker-search"
                size="large"
                clearable
                placeholder="搜索专家职称或描述"
              >
                <template #prefix-icon><t-icon name="search" /></template>
              </t-input>
              <button type="button" class="service-expert-picker-close" aria-label="关闭" @click="close">
                <t-icon name="close" />
              </button>
            </div>
          </header>

          <div class="service-expert-picker-tabs" role="tablist" aria-label="专家来源">
            <button
              type="button"
              role="tab"
              :aria-selected="activeTab === 'system'"
              class="service-expert-picker-tab"
              :class="{ active: activeTab === 'system' }"
              @click="activeTab = 'system'"
            >
              专家
            </button>
            <button
              type="button"
              role="tab"
              :aria-selected="activeTab === 'mine'"
              class="service-expert-picker-tab"
              :class="{ active: activeTab === 'mine' }"
              @click="activeTab = 'mine'"
            >
              我的专家
            </button>
          </div>

          <div v-if="activeTab === 'system'" class="service-expert-picker-categories">
            <button
              v-for="category in categories"
              :key="category.value"
              type="button"
              class="service-expert-picker-category"
              :class="{ active: activeCategory === category.value }"
              @click="activeCategory = category.value"
            >
              {{ category.label }}
            </button>
          </div>

          <div class="service-expert-picker-body">
            <div v-if="loading" class="service-expert-picker-state">
              <t-icon name="loading" class="service-expert-picker-loading" />
              正在读取系统专家
            </div>
            <div v-else-if="error" class="service-expert-picker-state service-expert-picker-state--error">
              <span>{{ error }}</span>
              <button type="button" @click="$emit('retry')">重试</button>
            </div>
            <div v-else-if="activeTab === 'mine'" class="service-expert-picker-state">
              暂无我的专家
            </div>
            <div v-else-if="filteredExperts.length" class="service-expert-picker-grid">
              <button
                v-for="expert in filteredExperts"
                :key="expert.id"
                type="button"
                class="service-expert-card"
                :class="{ selected: draftSelectedIds.includes(expert.id) }"
                :aria-pressed="draftSelectedIds.includes(expert.id)"
                @click="toggleExpert(expert.id)"
              >
                <div class="service-expert-card-head">
                  <AgentAvatar :name="expert.name" :avatar="expert.avatar" size="large" />
                  <span class="service-expert-card-check">
                    <t-icon :name="draftSelectedIds.includes(expert.id) ? 'check' : 'add'" />
                  </span>
                </div>
                <strong>{{ expert.name }}</strong>
                <small>{{ expert.packageName || '系统专家' }}</small>
                <p>{{ expert.description }}</p>
                <span class="service-expert-card-tags">
                  <em v-if="expert.domain">{{ domainLabel(expert.domain) }}</em>
                  <em
                    v-for="skill in expert.skills.slice(0, 2)"
                    :key="`${expert.id}-${skill}`"
                  >
                    {{ skillLabel(skill) }}
                  </em>
                </span>
              </button>
            </div>
            <div v-else class="service-expert-picker-state">
              {{ query || activeCategory !== 'all' ? '没有找到匹配的专家' : '暂无可用的系统专家' }}
            </div>
          </div>

          <footer class="service-expert-picker-footer">
            <span>{{ draftSelectedIds.length ? `已选择 ${draftSelectedIds.length} 位专家` : '请选择要加入服务空间的专家' }}</span>
            <div class="service-expert-picker-actions">
              <t-button variant="outline" size="large" @click="close">取消</t-button>
              <t-button
                theme="primary"
                size="large"
                :disabled="draftSelectedIds.length === 0"
                @click="confirm"
              >
                确定
              </t-button>
            </div>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import AgentAvatar from '@/components/AgentAvatar.vue'
import type { ServiceExpertOption } from './serviceHubState'

const props = withDefaults(defineProps<{
  visible: boolean
  experts: ServiceExpertOption[]
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

type ExpertTab = 'system' | 'mine'

const activeTab = ref<ExpertTab>('system')
const activeCategory = ref('all')
const query = ref('')
const draftSelectedIds = ref<string[]>([])

const categoryNameMap: Record<string, string> = {
  education: '教育学习',
  education_learning: '教育学习',
  customer_service: '客户服务',
  sales: '营销增长',
  marketing: '营销增长',
  finance: '金融投资',
  technology: '技术工程',
  engineering: '技术工程',
  data: '数据智能',
  research: '研究分析',
  operations: '运营管理',
}

const categories = computed(() => {
  const values = Array.from(new Set(
    props.experts
      .map((expert) => expert.domain?.trim())
      .filter((domain): domain is string => Boolean(domain)),
  ))
  return [
    { value: 'all', label: '全部' },
    ...values.map((value) => ({ value, label: domainLabel(value) })),
  ]
})

const filteredExperts = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase()
  return props.experts.filter((expert) => {
    const matchesCategory = activeCategory.value === 'all' || expert.domain === activeCategory.value
    if (!matchesCategory) return false
    if (!normalizedQuery) return true
    return [
      expert.name,
      expert.description,
      expert.packageName,
      expert.domain,
      ...expert.skills,
    ].join(' ').toLowerCase().includes(normalizedQuery)
  })
})

const domainLabel = (domain?: string) => {
  const normalized = domain?.trim() || ''
  if (!normalized) return '通用能力'
  return categoryNameMap[normalized.toLowerCase()] || normalized.replace(/[_-]+/g, ' ')
}

const skillLabel = (skill: string) => skill.replace(/[_-]+/g, ' ')

const reset = () => {
  draftSelectedIds.value = [...props.selectedIds]
  activeTab.value = 'system'
  activeCategory.value = 'all'
  query.value = ''
}

const close = () => {
  emit('update:visible', false)
}

const toggleExpert = (expertId: string) => {
  draftSelectedIds.value = draftSelectedIds.value.includes(expertId)
    ? draftSelectedIds.value.filter((id) => id !== expertId)
    : [...draftSelectedIds.value, expertId]
}

const confirm = () => {
  if (!draftSelectedIds.value.length) return
  emit('confirm', [...draftSelectedIds.value])
  close()
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) reset()
  },
)
</script>

<style scoped lang="less">
.service-expert-picker-overlay {
  position: fixed;
  z-index: 3200;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: rgba(0, 0, 0, 0.42);
}

.service-expert-picker-dialog {
  display: flex;
  width: min(1040px, calc(100vw - 40px));
  max-height: min(820px, calc(100vh - 40px));
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 18px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  box-shadow: 0 24px 70px rgba(0, 0, 0, 0.2);
}

.service-expert-picker-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 24px 28px 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-expert-picker-header h2 {
  margin: 0;
  font-size: 22px;
  font-weight: 650;
  line-height: 30px;
}

.service-expert-picker-header-tools {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.service-expert-picker-search {
  width: min(320px, 38vw);
}

.service-expert-picker-search :deep(.t-input) {
  border-radius: 9px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-expert-picker-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 20px;
}

.service-expert-picker-close:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-expert-picker-tabs,
.service-expert-picker-categories {
  display: flex;
  align-items: center;
  gap: 8px;
  overflow-x: auto;
  padding: 12px 28px 0;
}

.service-expert-picker-categories {
  padding-top: 8px;
  padding-bottom: 4px;
}

.service-expert-picker-tab,
.service-expert-picker-category {
  flex: none;
  padding: 7px 14px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
}

.service-expert-picker-tab:hover,
.service-expert-picker-category:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-expert-picker-tab.active,
.service-expert-picker-category.active {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.service-expert-picker-body {
  min-height: 0;
  flex: 1;
  overflow-y: auto;
  padding: 16px 28px 24px;
}

.service-expert-picker-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.service-expert-card {
  display: flex;
  min-width: 0;
  min-height: 214px;
  flex-direction: column;
  align-items: flex-start;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, background 0.16s ease;
}

.service-expert-card:hover {
  border-color: var(--td-brand-color-4);
  background: var(--td-bg-color-secondarycontainer);
}

.service-expert-card.selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  box-shadow: 0 0 0 1px var(--td-brand-color);
}

.service-expert-card-head {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.service-expert-card-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex: none;
  border: 1px solid var(--td-component-stroke);
  border-radius: 50%;
  color: var(--td-text-color-placeholder);
  font-size: 16px;
}

.service-expert-card.selected .service-expert-card-check {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.service-expert-card > strong {
  max-width: 100%;
  overflow: hidden;
  font-size: 15px;
  font-weight: 650;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-expert-card > small {
  max-width: 100%;
  margin-top: 1px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-expert-card > p {
  display: -webkit-box;
  min-height: 40px;
  margin: 10px 0 12px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.service-expert-card-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: auto;
}

.service-expert-card-tags em {
  max-width: 100%;
  overflow: hidden;
  padding: 3px 7px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  font-style: normal;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-expert-picker-state {
  display: flex;
  min-height: 240px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  text-align: center;
}

.service-expert-picker-state--error {
  flex-direction: column;
}

.service-expert-picker-state--error button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color-7);
  cursor: pointer;
  font: inherit;
}

.service-expert-picker-loading {
  animation: service-expert-picker-spin 0.9s linear infinite;
}

@keyframes service-expert-picker-spin {
  to {
    transform: rotate(360deg);
  }
}

.service-expert-picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 28px 20px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-expert-picker-footer > span {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.service-expert-picker-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}

.service-expert-picker-actions :deep(.t-button) {
  min-width: 86px;
  border-radius: 8px;
}

.service-expert-picker-enter-active,
.service-expert-picker-leave-active {
  transition: opacity 0.18s ease;
}

.service-expert-picker-enter-active .service-expert-picker-dialog,
.service-expert-picker-leave-active .service-expert-picker-dialog {
  transition: transform 0.18s ease;
}

.service-expert-picker-enter-from,
.service-expert-picker-leave-to {
  opacity: 0;
}

.service-expert-picker-enter-from .service-expert-picker-dialog,
.service-expert-picker-leave-to .service-expert-picker-dialog {
  transform: translateY(10px) scale(0.985);
}

@media (max-width: 860px) {
  .service-expert-picker-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 620px) {
  .service-expert-picker-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .service-expert-picker-dialog {
    width: 100%;
    max-height: 92vh;
    border-radius: 18px 18px 0 0;
  }

  .service-expert-picker-header {
    align-items: flex-start;
    flex-direction: column;
    padding: 20px 18px 12px;
  }

  .service-expert-picker-header-tools {
    width: 100%;
  }

  .service-expert-picker-search {
    width: auto;
    flex: 1;
  }

  .service-expert-picker-tabs,
  .service-expert-picker-categories,
  .service-expert-picker-body {
    padding-right: 18px;
    padding-left: 18px;
  }

  .service-expert-picker-grid {
    grid-template-columns: 1fr;
  }

  .service-expert-picker-footer {
    align-items: stretch;
    flex-direction: column;
    padding: 14px 18px 18px;
  }

  .service-expert-picker-actions {
    justify-content: flex-end;
  }
}
</style>
