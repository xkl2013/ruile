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
                填写服务名称和一句话说明，客户与工作方式由后台配置。
              </p>
            </div>
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
                size="medium"
                placeholder="例如：秋季招生咨询"
                :status="nameError ? 'error' : undefined"
                @input="nameError = ''"
              />
              <small v-if="nameError" class="service-create-error">{{ nameError }}</small>
            </div>

            <div class="service-create-field service-create-description-field">
              <label for="service-create-description">一句话说明 <em>（选填）</em></label>
              <t-textarea
                id="service-create-description"
                v-model="form.description"
                placeholder="例如：跟进秋季招生咨询，记录家长顾虑和下一步安排"
                :maxlength="500"
                :autosize="{ minRows: 3, maxRows: 6 }"
              />
              <small class="service-create-field-hint">
                客户信息和工作方式已经在后台配置好，创建后可以直接进入服务。
              </small>
            </div>
          </div>

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
import type { ServiceRecord } from './serviceHubState'

type ServiceCreateSource = ServiceRecord | null

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
  }]
}>()

const nameError = ref('')
const form = reactive({
  name: '',
  description: '',
})

const isCopyMode = computed(() => Boolean(props.source))

const resetForm = () => {
  const source = props.source
  form.name = source ? `${source.name}（副本）` : ''
  form.description = source?.description || ''
  nameError.value = ''
}

const close = () => {
  emit('update:visible', false)
}

const submit = () => {
  const name = form.name.trim()
  if (!name) {
    nameError.value = '请输入服务名称'
    return
  }
  emit('submit', {
    name,
    description: form.description.trim(),
  })
}

watch(
  () => props.visible,
  (visible) => {
    if (!visible) return
    resetForm()
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

.service-create-error {
  display: block;
  margin-top: 7px;
  color: var(--td-error-color);
  font-size: 13px;
}

.service-create-field-hint {
  display: block;
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
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
