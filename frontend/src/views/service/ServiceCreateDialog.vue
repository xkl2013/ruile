<template>
  <Teleport to="body" :disabled="displayMode === 'page'">
    <Transition name="service-create-dialog">
      <div
        v-if="visible"
        class="service-create-dialog-overlay"
        :class="{ 'is-page': displayMode === 'page' }"
        role="presentation"
        @click.self="displayMode === 'dialog' && close()"
      >
        <section
          class="service-create-dialog"
          :class="{ 'is-page': displayMode === 'page' }"
          :role="displayMode === 'dialog' ? 'dialog' : undefined"
          :aria-modal="displayMode === 'dialog' ? 'true' : undefined"
          aria-labelledby="service-create-dialog-title"
          @keydown.esc="close"
        >
          <header class="service-create-dialog-header">
            <div>
              <h2 id="service-create-dialog-title">{{ isCopyMode ? '复制配置创建服务' : '新建服务' }}</h2>
              <p class="service-create-dialog-subtitle">
                {{ selectedTemplate ? '模板内容已带入，可按当前工作调整' : '写清要做什么，再选择一起工作的专家与资料' }}
              </p>
            </div>
            <button type="button" class="service-create-dialog-close" aria-label="关闭" @click="close">
              <t-icon name="close" />
            </button>
          </header>

          <div class="service-create-dialog-body">
            <section v-if="selectedTemplate" class="service-create-template-preview">
              <div class="service-create-preview-head">
                <div>
                  <span class="service-create-section-kicker">已带入模板</span>
                  <h3>{{ selectedTemplate.name }}</h3>
                </div>
                <span class="service-create-auto-apply">
                  <t-icon name="check-circle" />
                  创建后仍可调整
                </span>
              </div>
              <p class="service-create-preview-description">{{ selectedTemplate.description }}</p>
            </section>

            <div class="service-create-field">
              <label for="service-create-name">服务名称</label>
              <t-input
                id="service-create-name"
                v-model="form.name"
                size="medium"
                placeholder="例如：秋季招生咨询"
                :status="formError ? 'error' : undefined"
                @input="formError = ''"
              />
              <small v-if="formError" class="service-create-error">{{ formError }}</small>
            </div>

            <div class="service-create-field service-create-instruction-field">
              <label for="service-create-instruction">工作指令</label>
              <t-textarea
                id="service-create-instruction"
                v-model="form.instruction"
                class="service-create-instruction"
                :placeholder="form.templateId ? '模板指令已填充，可按需要补充' : '例如：跟进秋季招生线索，重点记录家长顾虑与到访意向；涉及课程与价格时先查知识库'"
                :maxlength="4000"
                :autosize="{ minRows: 4, maxRows: 8 }"
              />
            </div>

            <section class="service-create-type-section">
              <div class="service-create-type-heading">
                <strong>主要围绕谁工作</strong>
                <span>用于预设服务的记录方式，创建后仍可调整</span>
              </div>
              <div class="service-create-type-options">
                <button
                  v-for="option in serviceTypeOptions"
                  :key="option.value"
                  type="button"
                  class="service-create-type-option"
                  :class="{ selected: form.spaceType === option.value }"
                  @click="form.spaceType = option.value"
                >
                  <t-icon :name="option.icon" />
                  <span>
                    <strong>{{ option.label }}</strong>
                    <small>{{ option.description }}</small>
                  </span>
                  <t-icon v-if="form.spaceType === option.value" name="check-circle-filled" class="service-create-type-check" />
                </button>
              </div>
            </section>

            <section class="service-create-option-section">
              <button
                type="button"
                class="service-create-option-row"
                :class="{ expanded: selectedExperts.length > 0 }"
                :aria-expanded="selectedExperts.length > 0"
                @click="expertPickerVisible = true"
              >
                <span class="service-create-option-title">
                  <strong>专家</strong>
                  <em>（可选）</em>
                  <span v-if="form.expertIds.length" class="service-create-selected-count">
                    {{ form.expertIds.length }} 个
                  </span>
                </span>
                <span class="service-create-option-action">
                  {{ form.expertIds.length ? '调整' : '+ 添加' }}
                </span>
              </button>

              <div v-if="selectedExperts.length" class="service-create-expert-list">
                <button
                  v-for="expert in selectedExperts"
                  :key="expert.id"
                  type="button"
                  class="service-create-expert-row"
                  @click="toggleExpert(expert.id)"
                >
                  <AgentAvatar :name="expert.name" :avatar="expert.avatar" size="small" />
                  <span class="service-create-picker-copy">
                    <strong>{{ expert.name }}</strong>
                    <small>{{ expert.description }}</small>
                  </span>
                  <span class="service-create-expert-remove" :aria-label="`移除${expert.name}`">
                    <t-icon name="close" />
                  </span>
                </button>
              </div>
              <div v-else class="service-create-option-empty">
                从系统专家列表中选择要加入这个服务空间的专家
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

          <ServiceExpertPickerDialog
            v-model:visible="expertPickerVisible"
            :experts="serviceExperts"
            :selected-ids="form.expertIds"
            :loading="serviceExpertsLoading"
            :error="serviceExpertsError"
            @confirm="setSelectedExperts"
            @retry="loadExperts"
          />

          <footer class="service-create-dialog-footer">
            <span>创建后直接进入服务，可以马上开始一段工作</span>
            <div class="service-create-dialog-actions">
              <t-button variant="outline" size="medium" @click="close">取消</t-button>
              <t-button
                theme="primary"
                size="medium"
                class="service-create-confirm"
                :disabled="!form.name.trim()"
                @click="submit"
              >
                创建服务
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
import AgentAvatar from '@/components/AgentAvatar.vue'
import { useChatResourcesStore } from '@/stores/chatResources'
import ServiceExpertPickerDialog from './ServiceExpertPickerDialog.vue'
import {
  loadServiceExperts,
  serviceExperts,
  serviceExpertsError,
  serviceExpertsLoading,
  serviceTemplates,
  type ServiceRecord,
  type ServiceTemplate,
} from './serviceHubState'

type ServiceCreateSource = ServiceTemplate | ServiceRecord | null
type ServiceSpaceType = 'customer_service' | 'operations' | 'research'
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
  displayMode?: 'dialog' | 'page'
}>(), {
  source: null,
  displayMode: 'dialog',
})
const visible = computed(() => props.visible)
const displayMode = computed(() => props.displayMode)

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  submit: [payload: {
    name: string
    description: string
    instruction: string
    templateId: string
    expertIds: string[]
    knowledgeBaseIds: string[]
    spaceType: ServiceSpaceType
  }]
}>()

const chatResources = useChatResourcesStore()
const knowledgeBases = computed<KnowledgeBaseOption[]>(() => chatResources.validAccountKnowledgeBases as KnowledgeBaseOption[])
const knowledgeBasesLoading = ref(false)
const knowledgeBasesError = ref('')
const knowledgeBaseQuery = ref('')
const expertPickerVisible = ref(false)
const knowledgeBasePickerOpen = ref(false)
const formError = ref('')
const form = reactive({
  name: '',
  description: '',
  instruction: '',
  templateId: '',
  expertIds: [] as string[],
  knowledgeBaseIds: [] as string[],
  spaceType: 'customer_service' as ServiceSpaceType,
})

const serviceTypeOptions: Array<{
  value: ServiceSpaceType
  label: string
  description: string
  icon: string
}> = [
  {
    value: 'customer_service',
    label: '家长与学员',
    description: '持续跟进家庭、学员、会员与招生线索',
    icon: 'usergroup',
  },
  {
    value: 'operations',
    label: '园所日常运营',
    description: '围绕活动、巡查、团队与跨岗位协作',
    icon: 'task',
  },
  {
    value: 'research',
    label: '教研与备课',
    description: '沉淀观察、课题、课程与可复用经验',
    icon: 'book-open',
  },
]

const isCopyMode = computed(() => Boolean(props.source && 'templateId' in props.source))
const selectedTemplate = computed(() => serviceTemplates.find((template) => template.id === form.templateId))
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

const expertIdsForTemplate = (template?: ServiceTemplate) => template?.experts
  .map((name) => serviceExperts.find((expert) => expert.name === name)?.id)
  .filter((id): id is string => Boolean(id)) || []

const resetForm = () => {
  const source = props.source
  const template = sourceTemplate(source)
  const service = source && 'templateId' in source ? source : null
  form.name = service ? `${service.name}（副本）` : template?.name || ''
  form.description = service?.description || template?.description || ''
  form.instruction = service?.instruction || template?.instruction || ''
  form.templateId = template?.id || ''
  form.expertIds = service?.expertIds?.length
    ? [...service.expertIds]
    : expertIdsForTemplate(template)
  form.knowledgeBaseIds = service?.knowledgeBaseIds ? [...service.knowledgeBaseIds] : []
  form.spaceType = service?.spaceType || template?.spaceType || 'customer_service'
  formError.value = ''
  knowledgeBaseQuery.value = ''
  expertPickerVisible.value = false
  knowledgeBasePickerOpen.value = false
}

const loadExperts = async () => {
  try {
    await loadServiceExperts()
  } catch (error) {
    console.error('[ServiceCreateDialog] Failed to load published experts:', error)
  }
}

const setSelectedExperts = (expertIds: string[]) => {
  form.expertIds = [...expertIds]
}

const syncTemplateExperts = () => {
  const source = props.source
  const service = source && 'templateId' in source ? source : null
  if (service?.expertIds?.length || form.expertIds.length) return
  const ids = expertIdsForTemplate(sourceTemplate(source))
  if (ids.length) form.expertIds = ids
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
    spaceType: form.spaceType,
  })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    resetForm()
    void loadKnowledgeBases()
    void loadExperts().then(() => {
      if (props.visible) syncTemplateExperts()
    })
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
  padding: 20px;
  background: rgba(0, 0, 0, 0.42);
}

.service-create-dialog-overlay.is-page {
  position: static;
  z-index: auto;
  display: block;
  min-height: 100%;
  padding: 20px 28px 32px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
}

.service-create-dialog {
  display: flex;
  width: min(760px, calc(100vw - 40px));
  max-height: min(680px, calc(100vh - 40px));
  flex-direction: column;
  overflow: hidden;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 14px;
  background: var(--td-bg-color-container);
  box-shadow: 0 18px 48px rgba(0, 0, 0, 0.16);
  color: var(--td-text-color-primary);
}

.service-create-dialog.is-page {
  width: min(920px, 100%);
  max-height: none;
  margin: 0 auto;
  overflow: visible;
  border: 0;
  border-radius: 0;
  box-shadow: none;
}

.service-create-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex: none;
  padding: 16px 24px 10px;
}

.service-create-dialog.is-page .service-create-dialog-header {
  padding: 0 0 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-create-dialog-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
}

.service-create-dialog.is-page .service-create-dialog-header h2 {
  font-size: 21px;
  font-weight: 500;
  line-height: 30px;
}

.service-create-dialog-subtitle {
  margin: 2px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.service-create-dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-size: 18px;
}

.service-create-dialog-close:hover {
  background: var(--td-bg-color-container-hover);
}

.service-create-dialog-body {
  min-height: 0;
  overflow-y: auto;
  padding: 2px 24px 16px;
}

.service-create-dialog.is-page .service-create-dialog-body {
  overflow: visible;
  padding: 20px 0 24px;
}

.service-create-section-kicker {
  display: block;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0;
  line-height: 16px;
  text-transform: uppercase;
}

.service-create-preview-head h3 {
  margin: 1px 0 0;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 21px;
}

.service-create-template-preview {
  margin: 0 0 18px;
  padding: 13px 14px;
  border: 1px solid var(--td-brand-color-3);
  border-radius: 8px;
  background: var(--td-brand-color-1);
}

.service-create-preview-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.service-create-auto-apply {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: none;
  color: var(--td-brand-color-7);
  font-size: 11px;
  line-height: 16px;
}

.service-create-preview-description {
  margin: 5px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.service-create-preview-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(0, 82, 217, 0.12);
}

.service-create-preview-meta span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.service-create-preview-meta b {
  margin-right: 5px;
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.service-create-preview-columns {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
  padding-top: 10px;
}

.service-create-preview-columns > div {
  min-width: 0;
}

.service-create-preview-columns > div > span {
  display: block;
  margin-bottom: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-create-preview-columns > div > div {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.service-create-preview-columns em {
  padding: 3px 6px;
  border-radius: 4px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  font-style: normal;
  line-height: 16px;
}

.service-create-field {
  margin-bottom: 16px;
}

.service-create-field > label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  line-height: 18px;
}

.service-create-field :deep(.t-input),
.service-create-field :deep(.t-textarea__inner) {
  border-radius: 8px;
  font-size: 14px;
}

.service-create-field :deep(.t-input) {
  min-height: 40px;
}

.service-create-field :deep(.t-textarea__inner) {
  min-height: 112px;
  padding: 10px 12px;
  line-height: 1.55;
}

.service-create-type-section {
  margin: 0 0 18px;
}

.service-create-type-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.service-create-type-heading strong {
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
}

.service-create-type-heading span {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-create-type-options {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.service-create-type-option {
  display: grid;
  position: relative;
  grid-template-columns: 28px minmax(0, 1fr);
  align-items: center;
  min-height: 72px;
  gap: 10px;
  padding: 10px 32px 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.service-create-type-option:hover,
.service-create-type-option.selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
}

.service-create-type-option > .t-icon:first-child {
  color: var(--td-brand-color-7);
  font-size: 20px;
}

.service-create-type-option > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-create-type-option strong {
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
}

.service-create-type-option small {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
}

.service-create-type-check {
  position: absolute;
  top: 9px;
  right: 9px;
  color: var(--td-brand-color);
}

.service-create-error {
  display: block;
  margin-top: 7px;
  color: var(--td-error-color);
  font-size: 13px;
}

.service-create-option-section {
  margin-top: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
  background: var(--td-bg-color-container);
}

.service-create-option-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 50px;
  padding: 0 14px;
  border: 0;
  border-radius: 9px;
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
  gap: 6px;
}

.service-create-option-title strong {
  font-size: 15px;
  font-weight: 600;
  line-height: 20px;
}

.service-create-option-title em {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-style: normal;
  font-weight: 400;
}

.service-create-selected-count {
  color: var(--td-brand-color-7);
  font-size: 12px;
}

.service-create-option-action {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.service-create-picker {
  margin: 0 14px 12px;
  padding: 6px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-create-expert-list {
  display: grid;
  gap: 1px;
  margin: 0 14px 12px;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-component-stroke);
}

.service-create-expert-row {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 8px;
  border: 0;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.service-create-expert-row:hover {
  background: var(--td-bg-color-secondarycontainer);
}

.service-create-expert-row .service-create-picker-copy {
  min-width: 0;
  flex: 1;
}

.service-create-expert-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  flex: none;
  border-radius: 50%;
  color: var(--td-text-color-placeholder);
}

.service-create-expert-row:hover .service-create-expert-remove {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-create-option-empty {
  padding: 0 14px 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-create-picker-option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
  padding: 10px;
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
  gap: 3px;
}

.service-create-picker-copy strong {
  font-size: 14px;
  font-weight: 600;
}

.service-create-picker-copy small {
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 17px;
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
  gap: 8px;
  margin-bottom: 4px;
  padding: 0 2px 2px;
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
  max-height: 150px;
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
  min-height: 56px;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
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
  gap: 6px;
  padding: 0 14px 12px;
}

.service-create-selected-item {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 100%;
  padding: 6px 7px 6px 10px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
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
  gap: 16px;
  flex: none;
  padding: 12px 24px 14px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-create-dialog.is-page .service-create-dialog-footer {
  position: sticky;
  bottom: 0;
  padding: 12px 0;
  background: var(--td-bg-color-container);
}

.service-create-dialog-footer > span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.service-create-dialog-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-create-dialog-actions :deep(.t-button) {
  min-width: 76px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
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
    max-height: 92vh;
    border-radius: 16px 16px 0 0;
  }

  .service-create-dialog-overlay.is-page {
    padding: 14px 16px 24px;
  }

  .service-create-dialog.is-page {
    width: 100%;
    max-height: none;
    border-radius: 0;
  }

  .service-create-dialog-header {
    padding: 16px 16px 8px;
  }

  .service-create-dialog-header h2 {
    font-size: 17px;
    line-height: 24px;
  }

  .service-create-dialog-subtitle {
    max-width: 260px;
  }

  .service-create-dialog-body {
    padding: 2px 16px 16px;
  }

  .service-create-preview-columns {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .service-create-type-heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;
  }

  .service-create-type-options {
    grid-template-columns: 1fr;
  }

  .service-create-field > label {
    font-size: 14px;
    line-height: 20px;
  }

  .service-create-field :deep(.t-input),
  .service-create-field :deep(.t-textarea__inner) {
    font-size: 14px;
  }

  .service-create-field :deep(.t-textarea__inner) {
    min-height: 128px;
  }

  .service-create-option-row {
    min-height: 50px;
    padding: 0 14px;
  }

  .service-create-option-title strong {
    font-size: 14px;
  }

  .service-create-option-title em,
  .service-create-option-action {
    font-size: 13px;
  }

  .service-create-picker {
    margin-right: 16px;
    margin-left: 16px;
  }

  .service-create-expert-list {
    margin-right: 16px;
    margin-left: 16px;
  }

  .service-create-option-empty {
    padding-right: 16px;
    padding-left: 16px;
  }

  .service-create-selected-list {
    padding-right: 16px;
    padding-left: 16px;
  }

  .service-create-dialog-footer {
    align-items: stretch;
    padding: 10px 16px 14px;
    flex-direction: column;
  }

  .service-create-dialog.is-page .service-create-dialog-footer {
    padding: 10px 0 0;
  }

  .service-create-dialog-footer > span {
    font-size: 12px;
  }

  .service-create-dialog-actions {
    justify-content: flex-end;
  }

  .service-create-dialog-actions :deep(.t-button) {
    min-width: 88px;
  }
}
</style>
