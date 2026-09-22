<template>
  <Teleport to="body">
    <Transition name="service-create-dialog">
      <div
        v-if="visible"
        class="service-create-dialog-overlay"
        role="presentation"
        @click.self="close"
      >
        <section
          class="service-create-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="service-create-dialog-title"
          @keydown.esc="close"
        >
          <header class="service-create-dialog-header">
            <h2 id="service-create-dialog-title">{{ isCopyMode ? '复制配置创建服务' : '新建服务' }}</h2>
            <button type="button" class="service-create-dialog-close" aria-label="关闭" @click="close">
              <t-icon name="close" />
            </button>
          </header>

          <div class="service-create-dialog-body">
            <div class="service-create-field">
              <label for="service-create-name">服务名称</label>
              <t-input
                id="service-create-name"
                v-model="form.name"
                size="large"
                placeholder="请输入服务名称"
                :status="formError ? 'error' : undefined"
                @input="formError = ''"
              />
              <small v-if="formError" class="service-create-error">{{ formError }}</small>
            </div>

            <div class="service-create-field service-create-instruction-field">
              <div class="service-create-field-head">
                <label for="service-create-instruction">指令</label>
                <t-select
                  v-model="form.templateId"
                  class="service-create-template-select"
                  size="large"
                  clearable
                  placeholder="选择模板"
                  :options="templateOptions"
                  @change="applyTemplate"
                />
              </div>
              <t-textarea
                id="service-create-instruction"
                v-model="form.instruction"
                class="service-create-instruction"
                placeholder="提供当前服务的背景信息和规范，让服务内的专家回复更精准、更符合要求。例如：服务目标、团队习惯、风格偏好、输出约束等"
                :maxlength="4000"
                :autosize="{ minRows: 7, maxRows: 12 }"
              />
            </div>

            <section class="service-create-option-section">
              <button
                type="button"
                class="service-create-option-row"
                :class="{ expanded: expertPickerOpen }"
                :aria-expanded="expertPickerOpen"
                @click="expertPickerOpen = !expertPickerOpen"
              >
                <span class="service-create-option-title">
                  <strong>专家</strong>
                  <em>（可选）</em>
                  <span v-if="form.expertIds.length" class="service-create-selected-count">
                    {{ form.expertIds.length }} 个
                  </span>
                </span>
                <span class="service-create-option-action">
                  {{ expertPickerOpen ? '收起' : '+ 添加' }}
                </span>
              </button>

              <div v-if="expertPickerOpen" class="service-create-picker">
                <button
                  v-for="expert in serviceExperts"
                  :key="expert.id"
                  type="button"
                  class="service-create-picker-option"
                  :class="{ selected: form.expertIds.includes(expert.id) }"
                  @click="toggleExpert(expert.id)"
                >
                  <span class="service-create-picker-copy">
                    <strong>{{ expert.name }}</strong>
                    <small>{{ expert.description }}</small>
                  </span>
                  <span class="service-create-picker-check">
                    <t-icon :name="form.expertIds.includes(expert.id) ? 'check' : 'add'" />
                  </span>
                </button>
              </div>

              <div v-if="selectedExperts.length" class="service-create-selected-list">
                <span v-for="expert in selectedExperts" :key="expert.id" class="service-create-selected-item">
                  {{ expert.name }}
                  <button type="button" :aria-label="`移除${expert.name}`" @click="toggleExpert(expert.id)">
                    <t-icon name="close" />
                  </button>
                </span>
              </div>
            </section>

            <section class="service-create-option-section">
              <button
                type="button"
                class="service-create-option-row"
                :class="{ expanded: knowledgeBasePickerOpen }"
                :aria-expanded="knowledgeBasePickerOpen"
                @click="knowledgeBasePickerOpen = !knowledgeBasePickerOpen"
              >
                <span class="service-create-option-title">
                  <strong>知识库</strong>
                  <em>（可选）</em>
                  <span v-if="form.knowledgeBaseIds.length" class="service-create-selected-count">
                    {{ form.knowledgeBaseIds.length }} 个
                  </span>
                </span>
                <span class="service-create-option-action">
                  {{ knowledgeBasePickerOpen ? '收起' : '+ 添加' }}
                </span>
              </button>

              <div v-if="knowledgeBasePickerOpen" class="service-create-picker service-create-kb-picker">
                <div class="service-create-kb-toolbar">
                  <t-input v-model="knowledgeBaseQuery" size="small" placeholder="搜索知识库">
                    <template #prefix-icon><t-icon name="search" /></template>
                  </t-input>
                  <span v-if="knowledgeBasesLoading" class="service-create-kb-status">正在加载</span>
                </div>

                <div v-if="knowledgeBasesLoading" class="service-create-picker-empty">
                  <t-icon name="loading" class="service-create-loading-icon" />
                  正在加载权限内的知识库
                </div>
                <div v-else-if="knowledgeBasesError" class="service-create-picker-empty service-create-picker-error">
                  {{ knowledgeBasesError }}
                  <button type="button" @click="loadKnowledgeBases">重试</button>
                </div>
                <div v-else-if="filteredKnowledgeBases.length" class="service-create-kb-list">
                  <button
                    v-for="knowledgeBase in filteredKnowledgeBases"
                    :key="knowledgeBase.id"
                    type="button"
                    class="service-create-picker-option"
                    :class="{ selected: form.knowledgeBaseIds.includes(String(knowledgeBase.id)) }"
                    @click="toggleKnowledgeBase(String(knowledgeBase.id))"
                  >
                    <span class="service-create-picker-copy">
                      <span class="service-create-kb-name-line">
                        <KnowledgeBaseIcon
                          :icon="knowledgeBase.icon"
                          :icon-url="knowledgeBase.icon_url"
                          :type="knowledgeBase.type"
                          size="small"
                        />
                        <strong>{{ knowledgeBase.name }}</strong>
                      </span>
                      <small>{{ knowledgeBaseMeta(knowledgeBase) }}</small>
                    </span>
                    <span class="service-create-picker-check">
                      <t-icon :name="form.knowledgeBaseIds.includes(String(knowledgeBase.id)) ? 'check' : 'add'" />
                    </span>
                  </button>
                </div>
                <div v-else class="service-create-picker-empty">
                  {{ knowledgeBaseQuery ? '没有找到匹配的知识库' : '当前没有可用的知识库' }}
                </div>
              </div>

              <div v-if="selectedKnowledgeBases.length" class="service-create-selected-list">
                <span
                  v-for="knowledgeBase in selectedKnowledgeBases"
                  :key="knowledgeBase.id"
                  class="service-create-selected-item service-create-selected-kb"
                >
                  <KnowledgeBaseIcon
                    :icon="knowledgeBase.icon"
                    :icon-url="knowledgeBase.icon_url"
                    :type="knowledgeBase.type"
                    size="small"
                  />
                  <span>{{ knowledgeBase.name }}</span>
                  <button type="button" :aria-label="`移除${knowledgeBase.name}`" @click="toggleKnowledgeBase(String(knowledgeBase.id))">
                    <t-icon name="close" />
                  </button>
                </span>
              </div>
            </section>
          </div>

          <footer class="service-create-dialog-footer">
            <span>创建后可继续调整服务配置</span>
            <div class="service-create-dialog-actions">
              <t-button variant="outline" size="large" @click="close">取消</t-button>
              <t-button
                theme="primary"
                size="large"
                class="service-create-confirm"
                :disabled="!form.name.trim()"
                @click="submit"
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
import { computed, reactive, ref, watch } from 'vue'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import { useChatResourcesStore } from '@/stores/chatResources'
import {
  serviceExperts,
  serviceTemplates,
  type ServiceRecord,
  type ServiceTemplate,
} from './serviceHubState'

type ServiceCreateSource = ServiceTemplate | ServiceRecord | null
type KnowledgeBaseOption = {
  id: string
  name: string
  icon?: string
  icon_url?: string
  type?: string
  knowledge_count?: number
  chunk_count?: number
  updated_at?: string
}

const props = withDefaults(defineProps<{
  visible: boolean
  source?: ServiceCreateSource
}>(), {
  source: null,
})

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  submit: [payload: {
    name: string
    description: string
    instruction: string
    templateId: string
    expertIds: string[]
    knowledgeBaseIds: string[]
  }]
}>()

const chatResources = useChatResourcesStore()
const knowledgeBases = computed<KnowledgeBaseOption[]>(() => chatResources.validAccountKnowledgeBases as KnowledgeBaseOption[])
const knowledgeBasesLoading = ref(false)
const knowledgeBasesError = ref('')
const knowledgeBaseQuery = ref('')
const expertPickerOpen = ref(false)
const knowledgeBasePickerOpen = ref(false)
const formError = ref('')
const form = reactive({
  name: '',
  description: '',
  instruction: '',
  templateId: '',
  expertIds: [] as string[],
  knowledgeBaseIds: [] as string[],
})

const isCopyMode = computed(() => Boolean(props.source && 'templateId' in props.source))
const templateOptions = computed(() => serviceTemplates.map((template) => ({
  label: template.name,
  value: template.id,
})))
const selectedExperts = computed(() => form.expertIds
  .map((id) => serviceExperts.find((expert) => expert.id === id))
  .filter((expert): expert is (typeof serviceExperts)[number] => Boolean(expert)))
const selectedKnowledgeBases = computed(() => form.knowledgeBaseIds
  .map((id) => knowledgeBases.value.find((knowledgeBase) => String(knowledgeBase.id) === id))
  .filter((knowledgeBase): knowledgeBase is KnowledgeBaseOption => Boolean(knowledgeBase)))
const filteredKnowledgeBases = computed(() => {
  const query = knowledgeBaseQuery.value.trim().toLowerCase()
  if (!query) return knowledgeBases.value
  return knowledgeBases.value.filter((knowledgeBase) => knowledgeBase.name.toLowerCase().includes(query))
})

const sourceTemplate = (source: ServiceCreateSource) => {
  if (!source) return undefined
  if ('templateId' in source) return serviceTemplates.find((template) => template.id === source.templateId)
  return source
}

const resetForm = () => {
  const source = props.source
  const template = sourceTemplate(source)
  const service = source && 'templateId' in source ? source : null
  form.name = service ? `${service.name}（副本）` : template?.name || ''
  form.description = service?.description || ''
  form.instruction = service?.instruction || template?.instruction || ''
  form.templateId = template?.id || ''
  form.expertIds = service?.expertIds?.length
    ? [...service.expertIds]
    : template?.experts
      .map((name) => serviceExperts.find((expert) => expert.name === name)?.id)
      .filter((id): id is string => Boolean(id)) || []
  form.knowledgeBaseIds = service?.knowledgeBaseIds ? [...service.knowledgeBaseIds] : []
  formError.value = ''
  knowledgeBaseQuery.value = ''
  expertPickerOpen.value = false
  knowledgeBasePickerOpen.value = false
}

const loadKnowledgeBases = async () => {
  knowledgeBasesLoading.value = true
  knowledgeBasesError.value = ''
  try {
    await chatResources.fetchMyKnowledgeBases(true)
  } catch (error) {
    console.error('[ServiceCreateDialog] Failed to load account-visible knowledge bases:', error)
    knowledgeBasesError.value = '知识库加载失败，请重试'
  } finally {
    knowledgeBasesLoading.value = false
  }
}

const applyTemplate = (templateId: string) => {
  const template = serviceTemplates.find((item) => item.id === templateId)
  if (!template) return
  form.instruction = template.instruction
  form.expertIds = template.experts
    .map((name) => serviceExperts.find((expert) => expert.name === name)?.id)
    .filter((id): id is string => Boolean(id))
}

const toggleExpert = (expertId: string) => {
  form.expertIds = form.expertIds.includes(expertId)
    ? form.expertIds.filter((id) => id !== expertId)
    : [...form.expertIds, expertId]
}

const toggleKnowledgeBase = (knowledgeBaseId: string) => {
  form.knowledgeBaseIds = form.knowledgeBaseIds.includes(knowledgeBaseId)
    ? form.knowledgeBaseIds.filter((id) => id !== knowledgeBaseId)
    : [...form.knowledgeBaseIds, knowledgeBaseId]
}

const knowledgeBaseMeta = (knowledgeBase: KnowledgeBaseOption) => {
  const count = knowledgeBase.type === 'faq'
    ? knowledgeBase.chunk_count || 0
    : knowledgeBase.knowledge_count || 0
  return `${count} 条内容${knowledgeBase.updated_at ? ` · 更新于 ${formatDate(knowledgeBase.updated_at)}` : ''}`
}

const formatDate = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${date.getMonth() + 1}-${String(date.getDate()).padStart(2, '0')}`
}

const close = () => {
  emit('update:visible', false)
}

const submit = () => {
  const name = form.name.trim()
  if (!name) {
    formError.value = '请输入服务名称'
    return
  }
  emit('submit', {
    name,
    description: form.description.trim(),
    instruction: form.instruction.trim(),
    templateId: form.templateId,
    expertIds: [...form.expertIds],
    knowledgeBaseIds: [...form.knowledgeBaseIds],
  })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    resetForm()
    void loadKnowledgeBases()
  },
)

watch(
  () => props.source,
  () => {
    if (props.visible) resetForm()
  },
)
</script>

<style scoped lang="less">
.service-create-dialog-overlay {
  position: fixed;
  z-index: 3000;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
  background: rgba(0, 0, 0, 0.56);
}

.service-create-dialog {
  display: flex;
  width: min(1280px, calc(100vw - 32px));
  max-height: calc(100vh - 32px);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 28px;
  background: var(--td-bg-color-container);
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.18);
  color: var(--td-text-color-primary);
}

.service-create-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex: none;
  padding: 32px 48px 20px;
}

.service-create-dialog-header h2 {
  margin: 0;
  font-size: 28px;
  font-weight: 650;
  line-height: 38px;
}

.service-create-dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-size: 28px;
}

.service-create-dialog-close:hover {
  background: var(--td-bg-color-container-hover);
}

.service-create-dialog-body {
  min-height: 0;
  overflow-y: auto;
  padding: 4px 48px 32px;
}

.service-create-field {
  margin-bottom: 30px;
}

.service-create-field > label,
.service-create-field-head > label {
  display: block;
  margin-bottom: 12px;
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.service-create-field :deep(.t-input),
.service-create-field :deep(.t-textarea__inner) {
  border-radius: 16px;
  font-size: 18px;
}

.service-create-field :deep(.t-input) {
  min-height: 58px;
}

.service-create-field :deep(.t-textarea__inner) {
  min-height: 220px;
  padding: 18px 22px;
  line-height: 1.65;
}

.service-create-field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.service-create-field-head > label {
  margin-bottom: 0;
}

.service-create-template-select {
  width: 184px;
  flex: none;
}

.service-create-template-select :deep(.t-input) {
  min-height: 48px;
  border: 0;
  border-radius: 14px;
  background: var(--td-bg-color-secondarycontainer);
  font-size: 16px;
}

.service-create-error {
  display: block;
  margin-top: 7px;
  color: var(--td-error-color);
  font-size: 13px;
}

.service-create-option-section {
  margin-top: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 16px;
  background: var(--td-bg-color-container);
}

.service-create-option-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 76px;
  padding: 0 24px;
  border: 0;
  border-radius: 16px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}

.service-create-option-row:hover,
.service-create-option-row.expanded {
  background: var(--td-bg-color-secondarycontainer);
}

.service-create-option-title {
  display: inline-flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 8px;
}

.service-create-option-title strong {
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.service-create-option-title em {
  color: var(--td-text-color-secondary);
  font-size: 18px;
  font-style: normal;
  font-weight: 400;
}

.service-create-selected-count {
  color: var(--td-brand-color-7);
  font-size: 14px;
}

.service-create-option-action {
  color: var(--td-text-color-secondary);
  font-size: 18px;
  font-weight: 500;
}

.service-create-picker {
  margin: 0 24px 20px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-create-picker-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 16px;
  padding: 12px;
  border: 0;
  border-radius: 9px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: inherit;
  text-align: left;
}

.service-create-picker-option:hover,
.service-create-picker-option.selected {
  background: var(--td-bg-color-container);
}

.service-create-picker-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-create-picker-copy strong {
  font-size: 15px;
  font-weight: 600;
}

.service-create-picker-copy small {
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-create-picker-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: none;
  color: var(--td-text-color-placeholder);
}

.service-create-picker-option.selected .service-create-picker-check {
  color: var(--td-brand-color);
}

.service-create-kb-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
  padding: 0 2px 4px;
}

.service-create-kb-toolbar :deep(.t-input) {
  flex: 1;
}

.service-create-kb-status {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-create-kb-list {
  max-height: 230px;
  overflow-y: auto;
}

.service-create-kb-name-line {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.service-create-kb-name-line strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-create-picker-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 68px;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.service-create-picker-empty button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color-7);
  cursor: pointer;
  font: inherit;
}

.service-create-picker-error {
  flex-direction: column;
}

.service-create-loading-icon {
  animation: service-create-spin 0.9s linear infinite;
}

@keyframes service-create-spin {
  to {
    transform: rotate(360deg);
  }
}

.service-create-selected-list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 24px 20px;
}

.service-create-selected-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 7px 8px 7px 12px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.service-create-selected-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-create-selected-item button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-create-selected-item button:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-create-selected-kb {
  min-width: 0;
}

.service-create-dialog-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex: none;
  padding: 20px 48px 26px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-create-dialog-footer > span {
  color: var(--td-text-color-secondary);
  font-size: 16px;
}

.service-create-dialog-actions {
  display: flex;
  align-items: center;
  gap: 14px;
}

.service-create-dialog-actions :deep(.t-button) {
  min-width: 128px;
  border-radius: 14px;
  font-size: 17px;
  font-weight: 600;
}

.service-create-confirm {
  border: 0;
}

.service-create-dialog-enter-active,
.service-create-dialog-leave-active {
  transition: opacity 0.18s ease;
}

.service-create-dialog-enter-active .service-create-dialog,
.service-create-dialog-leave-active .service-create-dialog {
  transition: transform 0.18s ease;
}

.service-create-dialog-enter-from,
.service-create-dialog-leave-to {
  opacity: 0;
}

.service-create-dialog-enter-from .service-create-dialog,
.service-create-dialog-leave-to .service-create-dialog {
  transform: translateY(12px) scale(0.985);
}

@media (max-width: 720px) {
  .service-create-dialog-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .service-create-dialog {
    width: 100%;
    max-height: 94vh;
    border-radius: 22px 22px 0 0;
  }

  .service-create-dialog-header {
    padding: 22px 20px 12px;
  }

  .service-create-dialog-header h2 {
    font-size: 24px;
    line-height: 32px;
  }

  .service-create-dialog-body {
    padding: 4px 20px 24px;
  }

  .service-create-field > label,
  .service-create-field-head > label,
  .service-create-option-title strong {
    font-size: 19px;
  }

  .service-create-field :deep(.t-input),
  .service-create-field :deep(.t-textarea__inner) {
    font-size: 16px;
  }

  .service-create-field :deep(.t-textarea__inner) {
    min-height: 170px;
  }

  .service-create-field-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .service-create-template-select {
    width: 100%;
  }

  .service-create-option-row {
    min-height: 64px;
    padding: 0 16px;
  }

  .service-create-option-title em,
  .service-create-option-action {
    font-size: 15px;
  }

  .service-create-picker {
    margin-right: 16px;
    margin-left: 16px;
  }

  .service-create-selected-list {
    padding-right: 16px;
    padding-left: 16px;
  }

  .service-create-dialog-footer {
    align-items: stretch;
    padding: 16px 20px 20px;
    flex-direction: column;
  }

  .service-create-dialog-footer > span {
    font-size: 13px;
  }

  .service-create-dialog-actions {
    justify-content: flex-end;
  }

  .service-create-dialog-actions :deep(.t-button) {
    min-width: 104px;
  }
}
</style>
