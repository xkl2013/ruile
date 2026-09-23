<template>
  <Teleport to="body">
    <Transition name="organize-config-dialog">
      <div
        v-if="visible"
        class="organize-config-dialog-overlay"
        role="presentation"
        @click.self="close"
      >
        <section
          class="organize-config-dialog"
          role="dialog"
          aria-modal="true"
          aria-labelledby="organize-config-dialog-title"
          @keydown.esc="close"
        >
          <header>
            <div>
              <h2 id="organize-config-dialog-title">{{ config ? '编辑整理' : templateKey ? '从模板新建整理' : '新建整理' }}</h2>
              <p>整理会按这里的配置执行</p>
            </div>
            <button type="button" class="dialog-icon-button" aria-label="关闭" @click="close">
              <t-icon name="close" />
            </button>
          </header>

          <div class="organize-config-dialog-body">
            <label class="organize-config-field">
              <span>名称</span>
              <t-input
                v-model="form.name"
                size="medium"
                placeholder="请输入整理名称，如：教研周整理"
                :status="error ? 'error' : undefined"
                @input="error = ''"
              />
              <small v-if="error" class="organize-config-error">{{ error }}</small>
            </label>

            <div class="organize-config-field">
              <div class="organize-config-field-head">
                <span>指令</span>
                <label>
                  <span class="sr-only">选择整理模板</span>
                  <select v-model="form.templateKey" @change="applyTemplate">
                    <option value="">选择模板</option>
                    <option v-for="template in templates" :key="template.key" :value="template.key">
                      {{ template.name }}（{{ template.scene }}）
                    </option>
                  </select>
                </label>
              </div>
              <t-textarea
                v-model="form.instruction"
                placeholder="告诉大模型把记忆整理成什么：先怎么归纳、再给什么、最后列什么。"
                :maxlength="4000"
                :autosize="{ minRows: 5, maxRows: 9 }"
              />
              <small>产出结构由模板决定，指令负责按什么口径整理</small>
            </div>

            <section class="organize-config-option">
              <button
                type="button"
                class="organize-config-option-row"
                :aria-expanded="expertPickerOpen"
                @click="expertPickerOpen = !expertPickerOpen"
              >
                <span>
                  <strong>专家</strong>
                  <em>（可选）</em>
                  <small v-if="form.expertIds.length">{{ form.expertIds.length }} 个</small>
                </span>
                <span>{{ expertPickerOpen ? '收起' : '+ 添加' }}</span>
              </button>

              <div v-if="expertPickerOpen" class="organize-config-expert-list">
                <button
                  v-for="expert in experts"
                  :key="expert.id"
                  type="button"
                  :class="{ selected: form.expertIds.includes(expert.id) }"
                  @click="toggleExpert(expert.id)"
                >
                  <span>
                    <strong>{{ expert.name }}</strong>
                    <small>{{ expert.description }}</small>
                  </span>
                  <t-icon :name="form.expertIds.includes(expert.id) ? 'check' : 'add'" />
                </button>
              </div>

              <div v-if="selectedExperts.length" class="organize-config-selected">
                <span v-for="expert in selectedExperts" :key="expert.id">
                  {{ expert.name }}
                  <button type="button" :aria-label="`移除${expert.name}`" @click="toggleExpert(expert.id)">
                    <t-icon name="close" />
                  </button>
                </span>
              </div>
            </section>

            <label class="organize-config-field">
              <span>执行周期</span>
              <select v-model="form.schedule" class="organize-config-schedule">
                <option v-for="option in scheduleOptions" :key="option.value" :value="option.value">
                  {{ option.label }}
                </option>
              </select>
            </label>
          </div>

          <footer>
            <div>
              <t-button variant="outline" size="medium" @click="close">取消</t-button>
              <t-button theme="primary" size="medium" :loading="saving" @click="submit">保存</t-button>
            </div>
          </footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createOrganizeConfig,
  listOrganizeExperts,
  listOrganizeTemplates,
  updateOrganizeConfig,
} from '@/api/organize'
import {
  organizeScheduleLabels,
  toOrganizeConfig,
  toOrganizeTemplate,
  type OrganizeConfig,
  type OrganizeExpert,
  type OrganizeScheduleKey,
  type OrganizeTemplate,
} from '../organizeWorkbenchState'

const props = withDefaults(defineProps<{
  visible: boolean
  templateKey?: string
  config?: OrganizeConfig | null
}>(), {
  templateKey: '',
  config: null,
})

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  saved: [config: OrganizeConfig]
}>()

const form = reactive({
  name: '',
  templateKey: '',
  instruction: '',
  expertIds: [] as string[],
  schedule: 'manual' as OrganizeScheduleKey,
})
const error = ref('')
const expertPickerOpen = ref(false)
const saving = ref(false)
const templates = ref<OrganizeTemplate[]>([])
const experts = ref<OrganizeExpert[]>([])

const scheduleOptions = Object.entries(organizeScheduleLabels).map(([value, label]) => ({
  value: value as OrganizeScheduleKey,
  label,
}))

const selectedExperts = computed(() =>
  experts.value.filter((expert) => form.expertIds.includes(expert.id)),
)

const resetForm = () => {
  const source = props.config
  const initialTemplateKey = source?.templateKey || props.templateKey || ''
  const template = templates.value.find((item) => item.key === initialTemplateKey)
  form.name = source?.name || (template ? `${template.name}整理` : '')
  form.templateKey = initialTemplateKey
  form.instruction = source?.instruction || template?.defaultInstruction || ''
  form.expertIds = source ? [...source.expertIds] : [...(template?.expertIds || [])]
  form.schedule = source?.schedule || 'manual'
  error.value = ''
  expertPickerOpen.value = false
}

watch(
  () => [props.visible, props.templateKey, props.config?.id],
  async ([visible]) => {
    if (!visible) return
    await loadOptions()
    resetForm()
  },
  { immediate: true },
)

const applyTemplate = () => {
  const template = templates.value.find((item) => item.key === form.templateKey)
  if (!template) return
  form.instruction = template.defaultInstruction
  form.expertIds = [...template.expertIds]
  if (!form.name.trim()) form.name = `${template.name}整理`
}

const toggleExpert = (expertId: string) => {
  form.expertIds = form.expertIds.includes(expertId)
    ? form.expertIds.filter((id) => id !== expertId)
    : [...form.expertIds, expertId]
}

const close = () => emit('update:visible', false)

const loadOptions = async () => {
  if (templates.value.length && experts.value.length) return
  try {
    const [templateResponse, expertResponse] = await Promise.all([
      listOrganizeTemplates(),
      listOrganizeExperts(),
    ])
    templates.value = (templateResponse.data || []).map(toOrganizeTemplate)
    experts.value = expertResponse.data || []
  } catch (loadError: any) {
    MessagePlugin.error(loadError?.message || '整理配置选项加载失败')
  }
}

const submit = async () => {
  const name = form.name.trim()
  if (!name) {
    error.value = '请先填写整理名称'
    return
  }
  if (!form.templateKey) {
    error.value = '请选择整理模板'
    return
  }

  saving.value = true
  try {
    const input = {
      name,
      template_key: form.templateKey,
      instruction: form.instruction.trim(),
      expert_ids: [...form.expertIds],
      schedule: form.schedule,
    }
    const response = props.config?.id
      ? await updateOrganizeConfig(props.config.id, input)
      : await createOrganizeConfig(input)
    emit('saved', toOrganizeConfig(response.data))
    close()
  } catch (saveError: any) {
    MessagePlugin.error(saveError?.message || '整理配置保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped lang="less">
.organize-config-dialog-overlay {
  position: fixed;
  inset: 0;
  z-index: 4200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(15, 18, 25, 0.36);
}

.organize-config-dialog {
  display: flex;
  width: min(760px, calc(100vw - 32px));
  max-height: min(780px, calc(100vh - 48px));
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  box-shadow: 0 24px 72px rgba(15, 18, 25, 0.2);
  color: var(--td-text-color-primary);
}

.organize-config-dialog > header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 28px 14px;
}

.organize-config-dialog h2,
.organize-config-dialog p {
  margin: 0;
}

.organize-config-dialog h2 {
  font-size: 22px;
  line-height: 30px;
}

.organize-config-dialog p {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.dialog-icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.dialog-icon-button:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.organize-config-dialog-body {
  min-height: 0;
  overflow-y: auto;
  padding: 4px 28px 22px;
}

.organize-config-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 18px;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
}

.organize-config-field > small,
.organize-config-field-head + :deep(.t-textarea) + small {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
}

.organize-config-field :deep(.t-input),
.organize-config-field :deep(.t-textarea__inner) {
  border-radius: 8px;
  font-size: 14px;
}

.organize-config-field-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 15px;
  font-weight: 600;
}

.organize-config-field-head select,
.organize-config-schedule {
  min-height: 36px;
  padding: 0 34px 0 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font: inherit;
  font-size: 13px;
}

.organize-config-schedule {
  width: 100%;
}

.organize-config-error {
  color: var(--td-error-color);
  font-size: 12px;
}

.organize-config-option {
  margin-bottom: 18px;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
}

.organize-config-option-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  min-height: 52px;
  padding: 0 14px;
  border: 0;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
}

.organize-config-option-row strong {
  font-size: 15px;
}

.organize-config-option-row em,
.organize-config-option-row small {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-style: normal;
  font-weight: 400;
}

.organize-config-expert-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 0 14px 14px;
}

.organize-config-expert-list > button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 66px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.organize-config-expert-list > button.selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.organize-config-expert-list strong,
.organize-config-expert-list small {
  display: block;
}

.organize-config-expert-list strong {
  font-size: 13px;
}

.organize-config-expert-list small {
  margin-top: 3px;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
}

.organize-config-selected {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 14px;
}

.organize-config-selected > span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 6px 0 9px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  font-size: 12px;
}

.organize-config-selected button {
  display: inline-flex;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.organize-config-dialog > footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 16px;
  padding: 14px 28px 18px;
  border-top: 1px solid var(--td-component-stroke);
}

.organize-config-dialog > footer > div {
  display: flex;
  gap: 8px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
}

.organize-config-dialog-enter-active,
.organize-config-dialog-leave-active {
  transition: opacity 0.18s ease;
}

.organize-config-dialog-enter-from,
.organize-config-dialog-leave-to {
  opacity: 0;
}

@media (max-width: 640px) {
  .organize-config-dialog-overlay {
    align-items: flex-end;
    padding: 0;
  }

  .organize-config-dialog {
    width: 100%;
    max-height: 92vh;
    border-radius: 12px 12px 0 0;
  }

  .organize-config-expert-list {
    grid-template-columns: 1fr;
  }

  .organize-config-dialog > footer {
    align-items: stretch;
    flex-direction: column;
  }

  .organize-config-dialog > footer > div {
    justify-content: flex-end;
  }
}
</style>
