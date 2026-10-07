<template>
  <div class="organize-product-page">
    <main v-if="output" class="organize-output-detail-scroll">
      <header class="organize-output-detail-head">
        <button type="button" class="organize-back-button" @click="close">
          <t-icon name="chevron-left" />
          返回{{ fromConfig ? '整理任务' : '我的整理' }}
        </button>
        <div class="organize-output-detail-actions">
          <t-button theme="primary" size="small" :loading="serviceImporting" @click="openServiceDialog">
            <template #icon><t-icon name="share" /></template>
            分配到服务
          </t-button>
          <t-button variant="outline" size="small" @click="close">关闭</t-button>
        </div>
      </header>

      <article class="organize-output-document">
        <header class="organize-output-document-head">
          <h2>{{ output.title }}</h2>
        </header>
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

        <OrganizeMarkdownRenderer :content="output.content" profile="report" />
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

    <t-dialog v-model:visible="serviceDialogVisible" header="分配到服务" width="620px" :footer="false">
      <div class="organize-service-dialog">
        <p class="organize-service-dialog-copy">
          将这份整理结果归属到服务空间。归属后，服务助理会把它作为持续工作的背景资料。
        </p>
        <div v-if="serviceLoading" class="organize-service-dialog-state">
          <t-loading size="small" />
          <span>正在加载我的服务</span>
        </div>
        <div v-else-if="serviceError" class="organize-service-dialog-state organize-service-dialog-state--error">
          <span>{{ serviceError }}</span>
          <t-button variant="text" theme="primary" size="small" @click="loadServices">重试</t-button>
        </div>
        <div v-else class="organize-service-dialog-list">
          <button
            v-for="service in services"
            :key="service.id"
            type="button"
            class="organize-service-option"
            :class="{ selected: selectedServiceId === service.id }"
            @click="selectedServiceId = service.id"
          >
            <span class="organize-service-option-icon"><t-icon name="folder" /></span>
            <span class="organize-service-option-main">
              <strong>{{ service.name }}</strong>
              <small>{{ service.description || '未填写服务描述' }}</small>
            </span>
            <span class="organize-service-option-state">{{ service.state === 'active' ? '进行中' : '草稿' }}</span>
          </button>
          <div v-if="!services.length" class="organize-service-dialog-empty">
            还没有可用的服务空间，请先创建一个服务。
          </div>
        </div>
        <div class="organize-service-dialog-actions">
          <t-button variant="outline" @click="createNewService">新建服务并使用</t-button>
          <t-button
            theme="primary"
            :disabled="!selectedServiceId || serviceImporting"
            :loading="serviceImporting"
            @click="importToSelectedService"
          >
            确认分配
          </t-button>
        </div>
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  assignOrganizeOutputToService,
  getOrganizeOutputCitation,
  getOrganizeOutput,
  listOrganizeTemplates,
  markOrganizeConfigOutputRead,
} from '@/api/organize'
import { listServiceSpaces, type ServiceSpace } from '@/api/service'
import {
  ORGANIZE_CONFIG_UNREAD_EVENT,
  markOrganizeConfigOutputLocallyRead,
  toOrganizeOutput,
  toOrganizeTemplate,
  type OrganizeOutput,
} from './organizeWorkbenchState'
import OrganizeMarkdownRenderer from './components/OrganizeMarkdownRenderer.vue'

const route = useRoute()
const router = useRouter()
const output = ref<OrganizeOutput | null>(null)
const loading = ref(true)
const fromConfig = computed(() => route.query.from === 'config')
const serviceDialogVisible = ref(false)
const serviceLoading = ref(false)
const serviceImporting = ref(false)
const serviceError = ref('')
const services = ref<ServiceSpace[]>([])
const selectedServiceId = ref('')

const markOutputRead = async (item: OrganizeOutput) => {
  if (!item.configId) return
  markOrganizeConfigOutputLocallyRead(item.configId, item.id)
  const detail: {
    configId: string
    viewedOutputId: string
    hasUnreadOutput?: boolean
  } = {
    configId: item.configId,
    viewedOutputId: item.id,
  }
  try {
    const response = await markOrganizeConfigOutputRead(item.configId, item.id)
    detail.hasUnreadOutput = Boolean(response.data?.has_unread_output)
  } catch {
    // The local watermark keeps rolling frontend/backend deployments usable.
  }
  window.dispatchEvent(new CustomEvent(ORGANIZE_CONFIG_UNREAD_EVENT, { detail }))
}

const loadOutput = async () => {
  loading.value = true
  try {
    const [outputResponse, templateResponse] = await Promise.all([
      getOrganizeOutput(String(route.params.outputId || '')),
      listOrganizeTemplates(),
    ])
    const templates = (templateResponse.data || []).map(toOrganizeTemplate)
    const nextOutput = toOrganizeOutput(outputResponse.data, templates)
    output.value = nextOutput
    void markOutputRead(nextOutput)
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
  if (route.query.from === 'service') {
    await router.push('/platform/service')
    return
  }
  await router.push('/platform/organize/mine')
}

const showCitation = async (citationId: string) => {
  const citation = output.value?.citations.find((item) => item.id === citationId)
  if (!citation || !output.value) return
  try {
    const response = await getOrganizeOutputCitation(output.value.id, citationId)
    if (response.data?.missing || !response.data?.memory) {
      MessagePlugin.warning('引用来源已不存在，无法打开原始记忆')
      return
    }
    const memory = response.data.memory
    const documentType = memory.kind === 'audio_card' ? 'audio-card' : memory.kind || 'note'
    await router.push({
      path: `/platform/organize/editor/${encodeURIComponent(documentType)}/${encodeURIComponent(memory.id)}`,
      query: { from: 'output', output: output.value.id },
    })
  } catch (error: any) {
    MessagePlugin.error(error?.message || '打开引用来源失败')
  }
}

const loadServices = async () => {
  serviceLoading.value = true
  serviceError.value = ''
  try {
    const response = await listServiceSpaces()
    services.value = (response.data || []).filter((service) => service.state !== 'archived')
    if (!services.value.some((service) => service.id === selectedServiceId.value)) {
      selectedServiceId.value = services.value[0]?.id || ''
    }
  } catch (error: any) {
    services.value = []
    serviceError.value = error?.message || '服务列表加载失败'
  } finally {
    serviceLoading.value = false
  }
}

const openServiceDialog = async () => {
  if (!output.value?.id) return
  serviceDialogVisible.value = true
  await loadServices()
}

const importToSelectedService = async () => {
  if (!output.value?.id || !selectedServiceId.value || serviceImporting.value) return
  serviceImporting.value = true
  try {
    await assignOrganizeOutputToService(output.value.id, selectedServiceId.value)
    serviceDialogVisible.value = false
    await router.push({
      path: '/platform/service',
      query: {
        service: selectedServiceId.value,
        context_source: output.value.id,
      },
    })
    MessagePlugin.success('整理结果已带入服务')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '带入服务失败')
  } finally {
    serviceImporting.value = false
  }
}

const createNewService = async () => {
  if (!output.value?.id) return
  serviceDialogVisible.value = false
  await router.push({
    path: '/platform/service',
    query: {
      create: '1',
      source_type: 'organize_output',
      source_id: output.value.id,
    },
  })
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

.organize-output-detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.organize-service-dialog-copy {
  margin: 0 0 14px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.organize-service-dialog-list {
  display: flex;
  max-height: 300px;
  overflow-y: auto;
  flex-direction: column;
  gap: 8px;
}

.organize-service-option {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 10px;
  padding: 11px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.organize-service-option:hover,
.organize-service-option.selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
}

.organize-service-option-icon {
  display: inline-flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 6px;
  background: var(--td-brand-color-2);
  color: var(--td-brand-color);
}

.organize-service-option-main {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.organize-service-option-main strong,
.organize-service-option-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-service-option-main strong {
  font-size: 13px;
  font-weight: 600;
}

.organize-service-option-main small,
.organize-service-option-state {
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.organize-service-option-state {
  flex: none;
}

.organize-service-dialog-state,
.organize-service-dialog-empty {
  display: flex;
  min-height: 100px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.organize-service-dialog-state--error {
  flex-direction: column;
}

.organize-service-dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 18px;
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
  padding-bottom: 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.organize-output-document-head h2 {
  margin: 0;
  font-size: 24px;
  line-height: 32px;
}

.organize-output-citations {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 16px 0 6px;
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
}
</style>
