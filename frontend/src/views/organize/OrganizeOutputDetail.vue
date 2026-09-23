<template>
  <div class="organize-product-page">
    <main v-if="output" class="organize-output-detail-scroll">
      <header class="organize-output-detail-head">
        <button type="button" class="organize-back-button" @click="close">
          <t-icon name="chevron-left" />
          返回{{ fromConfig ? '整理任务' : '我的整理' }}
        </button>
        <div class="organize-output-detail-actions">
          <t-button variant="outline" size="small" @click="close">关闭</t-button>
        </div>
      </header>

      <article class="organize-output-document">
        <header class="organize-output-document-head">
          <div>
            <span class="organize-output-eyebrow">{{ output.templateName }} · {{ output.templateVersion }}</span>
            <h2>{{ output.title }}</h2>
            <p>{{ output.date }} · 来源 {{ output.sourceCount }} 条记忆</p>
          </div>
          <span class="organize-tag organize-tag--success">已完成</span>
        </header>

        <div v-if="output.fields.length" class="organize-output-fields">
          <div v-for="field in output.fields" :key="field.label">
            <span>{{ field.label }}</span>
            <strong>{{ field.value }}</strong>
          </div>
        </div>
        <div v-if="output.citations.length" class="organize-output-citations">
          <button
            v-for="citation in output.citations"
            :key="citation.id"
            type="button"
            class="organize-citation"
            @click="showCitation(citation.id)"
          >
            {{ citation.label }} · {{ citation.title }}
          </button>
        </div>

        <div class="organize-output-document-body markdown-content" v-html="outputHtml" />
      </article>
    </main>

    <main v-else-if="loading" class="organize-output-detail-scroll">
      <div class="organize-detail-empty">
        <t-loading size="small" />
        <strong>正在加载整理结果</strong>
      </div>
    </main>

    <main v-else class="organize-output-detail-scroll">
      <div class="organize-detail-empty">
        <t-icon name="error-circle" />
        <strong>产物不存在</strong>
        <t-button variant="outline" size="small" @click="close">返回我的整理</t-button>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  getOrganizeOutput,
  listOrganizeTemplates,
} from '@/api/organize'
import {
  toOrganizeOutput,
  toOrganizeTemplate,
  type OrganizeOutput,
} from './organizeWorkbenchState'
import { renderSproutReportHtml } from './sproutReport'

const route = useRoute()
const router = useRouter()
const output = ref<OrganizeOutput | null>(null)
const loading = ref(true)
const fromConfig = computed(() => route.query.from === 'config')
const outputHtml = computed(() => renderSproutReportHtml(output.value?.content || ''))

const loadOutput = async () => {
  loading.value = true
  try {
    const [outputResponse, templateResponse] = await Promise.all([
      getOrganizeOutput(String(route.params.outputId || '')),
      listOrganizeTemplates(),
    ])
    const templates = (templateResponse.data || []).map(toOrganizeTemplate)
    output.value = toOrganizeOutput(outputResponse.data, templates)
  } catch (error: any) {
    output.value = null
    MessagePlugin.error(error?.message || '整理结果加载失败')
  } finally {
    loading.value = false
  }
}

const close = async () => {
  if (fromConfig.value && route.query.configId) {
    await router.push({
      path: `/platform/organize/configs/${encodeURIComponent(String(route.query.configId))}`,
      query: route.query.job ? { job: String(route.query.job) } : undefined,
    })
    return
  }
  await router.push('/platform/organize/mine')
}

const showCitation = (citationId: string) => {
  const citation = output.value?.citations.find((item) => item.id === citationId)
  if (!citation) return
  const source = citation.source ? ` · ${citation.source}` : ''
  MessagePlugin.info(`${citation.label} · ${citation.title}${source}`)
}

onMounted(() => {
  void loadOutput()
})

watch(
  () => route.params.outputId,
  () => void loadOutput(),
)
</script>

<style scoped lang="less">
@import '../../components/css/chat-markdown.less';

.organize-product-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
}

.organize-output-detail-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 42px 48px;
}

.organize-output-detail-head,
.organize-output-document {
  max-width: 860px;
  margin-right: auto;
  margin-left: auto;
}

.organize-output-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
}

.organize-back-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
}

.organize-output-document {
  padding: 26px 30px 38px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
}

.organize-output-document-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.organize-output-eyebrow {
  color: var(--td-brand-color);
  font-size: 12px;
}

.organize-output-document-head h2 {
  margin: 8px 0 5px;
  font-size: 24px;
  line-height: 32px;
}

.organize-output-document-head p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.organize-output-fields {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
  padding: 18px 0 22px;
}

.organize-output-fields > div {
  min-height: 58px;
  padding: 10px 12px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.organize-output-fields span,
.organize-output-fields strong {
  display: block;
}

.organize-output-fields span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.organize-output-fields strong {
  margin-top: 5px;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
}

.organize-output-citations {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 4px 0 14px;
}

.organize-output-document-body {
  padding-top: 8px;
  .chat-markdown-typography();
}

.organize-citation {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  margin-left: 4px;
  padding: 0 5px;
  border: 1px solid #d9d6f4;
  border-radius: 4px;
  background: #f4f2ff;
  color: #5d56ad;
  cursor: pointer;
  font: inherit;
  font-size: 11px;
  line-height: 18px;
  vertical-align: 1px;
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

.organize-tag--success {
  border-color: #b7e1cf;
  background: #eef9f3;
  color: #23805a;
}

.organize-detail-empty {
  display: flex;
  min-height: 200px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-secondary);
}

.organize-detail-empty :deep(.t-icon) {
  color: var(--td-text-color-placeholder);
  font-size: 26px;
}

.organize-detail-empty strong {
  color: var(--td-text-color-primary);
  font-size: 14px;
}

@media (max-width: 760px) {
  .organize-output-detail-scroll {
    padding: 20px 18px 36px;
  }

  .organize-output-document {
    padding: 22px 18px 30px;
  }

  .organize-output-fields {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 460px) {
  .organize-output-fields {
    grid-template-columns: 1fr;
  }
}
</style>
