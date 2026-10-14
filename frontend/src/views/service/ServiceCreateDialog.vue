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
          <header>
            <div>
              <h2 id="service-create-dialog-title">
                {{ isCopyMode ? '复制配置创建服务' : '新建服务' }}
              </h2>
              <p>填写服务名称即可，其他配置由系统自动完成。</p>
            </div>
            <button
              type="button"
              class="service-create-dialog-close"
              aria-label="关闭"
              :disabled="submitting"
              @click="close"
            >
              <t-icon name="close" />
            </button>
          </header>

          <div class="service-create-dialog-body">
            <label class="service-create-field" for="service-create-name">
              <span>服务名称</span>
              <t-input
                id="service-create-name"
                v-model="form.name"
                size="medium"
                placeholder="例如：秋季招生咨询"
                :status="nameError ? 'error' : undefined"
                :disabled="submitting"
                @input="nameError = ''"
                @enter="submit"
              />
              <small v-if="nameError" class="service-create-error">{{ nameError }}</small>
            </label>
          </div>

          <footer>
            <div>
              <t-button variant="outline" size="medium" :disabled="submitting" @click="close">
                取消
              </t-button>
              <t-button
                theme="primary"
                size="medium"
                :loading="submitting"
                :disabled="submitting || !form.name.trim()"
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

const props = withDefaults(defineProps<{
  visible: boolean
  source?: ServiceRecord | null
  submitting?: boolean
}>(), {
  source: null,
  submitting: false,
})

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  submit: [payload: { name: string }]
}>()

const form = reactive({ name: '' })
const nameError = ref('')
const isCopyMode = computed(() => Boolean(props.source))

const resetForm = () => {
  form.name = props.source ? `${props.source.name}（副本）` : ''
  nameError.value = ''
}

const close = () => {
  if (props.submitting) return
  emit('update:visible', false)
}

const submit = () => {
  if (props.submitting) return
  const name = form.name.trim()
  if (!name) {
    nameError.value = '请输入服务名称'
    return
  }
  emit('submit', { name })
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) resetForm()
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
  inset: 0;
  z-index: 4200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(15, 18, 25, 0.36);
}

.service-create-dialog {
  display: flex;
  width: min(560px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  box-shadow: 0 24px 72px rgba(15, 18, 25, 0.2);
  color: var(--td-text-color-primary);
}

.service-create-dialog > header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  padding: 22px 28px 14px;
}

.service-create-dialog h2,
.service-create-dialog p {
  margin: 0;
}

.service-create-dialog h2 {
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.service-create-dialog p {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.service-create-dialog-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 18px;
}

.service-create-dialog-close:hover:not(:disabled) {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-create-dialog-close:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}

.service-create-dialog-body {
  min-height: 0;
  overflow-y: auto;
  padding: 4px 28px 22px;
}

.service-create-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
}

.service-create-field :deep(.t-input) {
  min-height: 40px;
  border-radius: 8px;
  font-size: 14px;
}

.service-create-error {
  color: var(--td-error-color);
  font-size: 12px;
  font-weight: 400;
}

.service-create-dialog > footer {
  display: flex;
  justify-content: flex-end;
  padding: 14px 28px 18px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-create-dialog > footer > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-create-dialog :deep(.t-button) {
  min-width: 84px;
  border-radius: 8px;
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
    border-radius: 12px 12px 0 0;
  }

  .service-create-dialog > header {
    padding: 18px 20px 12px;
  }

  .service-create-dialog h2 {
    font-size: 19px;
    line-height: 27px;
  }

  .service-create-dialog-body {
    padding: 4px 20px 22px;
  }

  .service-create-dialog > footer {
    padding: 12px 20px 16px;
  }
}
</style>
