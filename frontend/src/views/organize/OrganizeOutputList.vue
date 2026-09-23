<template>
  <div class="organize-product-page">
    <main class="organize-list-scroll">
      <header class="organize-list-hero">
        <div>
          <span class="organize-eyebrow">产物列表</span>
          <h2>我的整理</h2>
          <p>按标签、场景和时间找到已经整理好的内容。</p>
        </div>
        <div class="organize-list-count">{{ filteredOutputs.length }} 条结果</div>
      </header>

      <div class="organize-output-toolbar">
        <t-input v-model="query" placeholder="搜索整理结果">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <div class="organize-output-tags" aria-label="按场景筛选">
          <button
            v-for="option in sceneOptions"
            :key="option.value"
            type="button"
            :class="{ active: sceneFilter === option.value }"
            @click="sceneFilter = option.value"
          >
            {{ option.label }}
          </button>
        </div>
        <select v-model="sortMode" aria-label="产物排序">
          <option value="time">按时间</option>
          <option value="todo">按待办数</option>
        </select>
      </div>

      <section class="organize-output-list">
        <article
          v-for="output in filteredOutputs"
          :key="output.id"
          class="organize-output-card"
          role="button"
          tabindex="0"
          @click="openOutput(output.id)"
          @keydown.enter.self.prevent="openOutput(output.id)"
        >
          <div class="organize-output-card-main">
            <div class="organize-output-title-row">
              <h3>{{ output.title }}</h3>
              <span class="organize-tag organize-tag--success">已完成</span>
            </div>
            <p class="organize-output-subject">
              主题：{{ output.subject }}
            </p>
            <div class="organize-output-meta">
              <span>{{ output.date }}</span>
              <span class="organize-output-separator" />
              <span>结论 {{ output.conclusionCount }} 条</span>
              <span class="organize-output-separator" />
              <span>待办 {{ output.todoCount }} 项</span>
              <span class="organize-output-separator" />
              <span>来源 {{ output.sourceCount }} 条记忆</span>
              <span v-for="tag in output.tags" :key="`${output.id}-${tag}`" class="organize-tag organize-tag--accent">
                {{ tag }}
              </span>
            </div>
          </div>
          <div class="organize-output-card-side">
            <span>{{ output.templateName }}</span>
            <t-icon name="chevron-right" />
          </div>
        </article>

        <div v-if="!filteredOutputs.length" class="organize-list-empty">
          <t-icon name="search" />
          <strong>没有匹配的整理结果</strong>
          <span>换一个关键词或筛选条件试试。</span>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import {
  listOrganizeOutputs,
  listOrganizeTemplates,
} from '@/api/organize'
import {
  toOrganizeOutput,
  toOrganizeTemplate,
  type OrganizeOutput,
  type OrganizeTemplate,
} from './organizeWorkbenchState'

type SortMode = 'time' | 'todo'

const router = useRouter()
const query = ref('')
const sceneFilter = ref('all')
const sortMode = ref<SortMode>('time')
const outputs = ref<OrganizeOutput[]>([])
const templates = ref<OrganizeTemplate[]>([])

const sceneOptions = computed(() => [
  { value: 'all', label: '全部' },
  ...Array.from(new Set(templates.value.map((template) => template.scene))).map((scene) => ({
    value: scene,
    label: scene,
  })),
])

const filteredOutputs = computed(() => {
  const normalizedQuery = query.value.trim().toLowerCase()
  const filtered = outputs.value.filter((output) => {
    const template = templates.value.find((item) => item.key === output.templateKey)
    const sceneMatched = sceneFilter.value === 'all' || template?.scene === sceneFilter.value
    const textMatched =
      !normalizedQuery ||
      [output.title, output.subject, output.templateName, ...output.tags]
        .join(' ')
        .toLowerCase()
        .includes(normalizedQuery)
    return sceneMatched && textMatched
  })

  if (sortMode.value === 'todo') {
    return [...filtered].sort((a, b) => (b.todoCount || 0) - (a.todoCount || 0))
  }
  return [...filtered].sort((a, b) => b.date.localeCompare(a.date))
})

const loadOutputs = async () => {
  try {
    const [templateResponse, outputResponse] = await Promise.all([
      listOrganizeTemplates(),
      listOrganizeOutputs({ page: 1, page_size: 100 }),
    ])
    templates.value = (templateResponse.data || []).map(toOrganizeTemplate)
    outputs.value = (outputResponse.data?.items || []).map((item) =>
      toOrganizeOutput(item, templates.value),
    )
  } catch (error: any) {
    MessagePlugin.error(error?.message || '整理结果加载失败')
  }
}

const openOutput = async (outputId: string) => {
  await router.push({
    path: `/platform/organize/outputs/${encodeURIComponent(outputId)}`,
    query: { from: 'mine' },
  })
}

onMounted(() => {
  void loadOutputs()
})
</script>

<style scoped lang="less">
.organize-product-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
}

.organize-list-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 32px 42px 48px;
}

.organize-list-hero {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  max-width: 1040px;
  margin: 0 auto 24px;
}

.organize-eyebrow {
  color: var(--td-brand-color);
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.08em;
}

.organize-list-hero h2 {
  margin: 7px 0 5px;
  font-size: 28px;
  line-height: 36px;
}

.organize-list-hero p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.organize-list-count {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.organize-output-toolbar,
.organize-output-list {
  max-width: 1040px;
  margin-right: auto;
  margin-left: auto;
}

.organize-output-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}

.organize-output-toolbar :deep(.t-input) {
  width: 210px;
}

.organize-output-tags {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow-x: auto;
}

.organize-output-tags button {
  min-height: 30px;
  padding: 0 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  white-space: nowrap;
}

.organize-output-tags button:hover,
.organize-output-tags button.active {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.organize-output-toolbar > select {
  min-height: 32px;
  margin-left: auto;
  padding: 0 28px 0 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font: inherit;
  font-size: 12px;
}

.organize-output-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.organize-output-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  min-height: 94px;
  padding: 16px 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
}

.organize-output-card:hover,
.organize-output-card:focus-visible {
  border-color: var(--td-brand-color-3);
  box-shadow: 0 6px 18px rgba(43, 48, 64, 0.07);
  outline: none;
}

.organize-output-card-main {
  min-width: 0;
}

.organize-output-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}

.organize-output-title-row h3 {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 650;
}

.organize-output-subject {
  margin: 8px 0 11px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-output-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.organize-output-separator {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--td-text-color-placeholder);
}

.organize-tag {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 7px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  white-space: nowrap;
}

.organize-tag--accent {
  border-color: #d9d6f4;
  background: #f4f2ff;
  color: #5d56ad;
}

.organize-tag--success {
  border-color: #b7e1cf;
  background: #eef9f3;
  color: #23805a;
}

.organize-output-card-side {
  display: flex;
  align-items: center;
  flex: none;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.organize-output-card-side :deep(.t-icon) {
  color: var(--td-text-color-placeholder);
}

.organize-list-empty {
  display: flex;
  min-height: 190px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 9px;
  color: var(--td-text-color-secondary);
}

.organize-list-empty :deep(.t-icon) {
  color: var(--td-text-color-placeholder);
  font-size: 24px;
}

.organize-list-empty strong {
  color: var(--td-text-color-primary);
  font-size: 14px;
}

.organize-list-empty span {
  font-size: 12px;
}

@media (max-width: 760px) {
  .organize-list-scroll {
    padding: 24px 20px 36px;
  }

  .organize-output-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .organize-output-toolbar :deep(.t-input),
  .organize-output-toolbar > select {
    width: 100%;
    margin-left: 0;
  }

  .organize-output-card {
    align-items: flex-start;
    flex-direction: column;
    gap: 12px;
  }
}
</style>
