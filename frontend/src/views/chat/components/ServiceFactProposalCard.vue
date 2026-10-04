<template>
  <div class="service-fact-proposal-card" :class="`is-${status}`">
    <div class="proposal-header">
      <div class="proposal-title">
        <t-icon class="proposal-title-icon" :name="statusIcon" />
        <span>{{ title }}</span>
      </div>
      <span class="proposal-status">{{ statusLabel }}</span>
    </div>

    <p v-if="status === 'pending' && proposal.question" class="proposal-question">
      {{ proposal.question }}
    </p>

    <div v-if="subjectRequired" class="proposal-subject-picker">
      <label for="service-fact-proposal-subject">更新到服务主体</label>
      <t-select
        id="service-fact-proposal-subject"
        v-model="selectedSubjectId"
        size="small"
        :options="subjectOptions"
        :loading="subjectsLoading"
        :disabled="submitting"
        placeholder="选择主体"
        clearable
      />
      <span v-if="subjectsError" class="proposal-error">{{ subjectsError }}</span>
    </div>

    <div class="proposal-items">
      <div v-for="item in proposal.items || []" :key="item.field_key" class="proposal-item">
        <div class="proposal-item-main">
          <span class="proposal-item-label">{{ item.field_label || item.field_key }}</span>
          <strong>{{ formatValue(item.value) }}</strong>
        </div>
        <div v-if="item.evidence || item.confidence !== undefined" class="proposal-item-meta">
          <span v-if="item.evidence" class="proposal-evidence">依据：{{ item.evidence }}</span>
          <span v-if="item.confidence !== undefined" class="proposal-confidence">
            置信度 {{ formatConfidence(item.confidence) }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="status === 'pending'" class="proposal-actions">
      <button type="button" class="proposal-action" :disabled="submitting" @click="resolve('rejected')">
        <t-icon name="close-circle" />
        忽略
      </button>
      <button
        type="button"
        class="proposal-action is-primary"
        :disabled="submitting || !serviceId || (subjectRequired && !selectedSubjectId)"
        @click="resolve('confirmed')"
      >
        <t-icon name="check-circle" />
        {{ submitting ? '处理中' : '确认更新' }}
      </button>
    </div>

    <div v-else class="proposal-resolved">
      <t-icon :name="statusIcon" />
      {{ resolvedMessage }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listServiceSubjects,
  resolveServiceFactProposal,
  type ServiceFactProposal,
  type ServiceSubject,
} from '@/api/service'

const props = defineProps<{
  serviceId: string
  proposal: ServiceFactProposal
}>()

const submitting = ref(false)
const subjectsLoading = ref(false)
const subjectsError = ref('')
const subjects = ref<ServiceSubject[]>([])
const selectedSubjectId = ref(props.proposal.subject_id || '')
const status = ref<ServiceFactProposal['status']>(props.proposal.status || 'pending')

const subjectRequired = computed(() => status.value === 'pending' && (
  Boolean(props.proposal.needs_subject)
))

const subjectOptions = computed(() => subjects.value.map((subject) => ({
  label: `${subject.display_name} (${subject.subject_key})`,
  value: subject.id,
})))

const title = computed(() => {
  if (status.value === 'confirmed') return '服务档案已更新'
  if (status.value === 'rejected') return '档案更新提案已忽略'
  return '识别到档案信息'
})

const statusIcon = computed(() => {
  if (status.value === 'confirmed') return 'check-circle'
  if (status.value === 'rejected') return 'close-circle'
  return 'edit-1'
})

const statusLabel = computed(() => {
  if (status.value === 'confirmed') return '已确认'
  if (status.value === 'rejected') return '已忽略'
  return '待确认'
})

const resolvedMessage = computed(() => {
  if (status.value === 'confirmed') {
    return selectedSubjectId.value || props.proposal.subject_id
      ? '事实已写入主体档案，并刷新空间摘要。'
      : '事实已写入服务空间档案，并刷新空间摘要。'
  }
  return '本次识别结果未写入服务空间。'
})

const formatValue = (value: unknown) => {
  if (value === undefined || value === null || value === '') return '待补充'
  if (Array.isArray(value)) return value.join('、')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const formatConfidence = (value: number) => `${Math.round(value * 100)}%`

const loadSubjects = async () => {
  if (!props.serviceId || !subjectRequired.value || subjects.value.length) return
  subjectsLoading.value = true
  subjectsError.value = ''
  try {
    const response = await listServiceSubjects(props.serviceId, { page: 1, page_size: 100 })
    subjects.value = response?.data?.items || []
  } catch (error: any) {
    subjectsError.value = error?.response?.data?.message || '主体列表加载失败'
  } finally {
    subjectsLoading.value = false
  }
}

const notifyServiceSpaceUpdated = (resolvedProposal: ServiceFactProposal) => {
  window.dispatchEvent(new CustomEvent('service-space-data-updated', {
    detail: {
      serviceId: props.serviceId,
      proposalId: resolvedProposal.id || props.proposal.id,
      proposal: resolvedProposal,
    },
  }))
}

const resolve = async (decision: 'confirmed' | 'rejected') => {
  if (submitting.value || !props.serviceId || status.value !== 'pending') return
  if (decision === 'confirmed' && subjectRequired.value && !selectedSubjectId.value) {
    await loadSubjects()
    MessagePlugin.warning('请选择要更新的服务主体')
    return
  }

  submitting.value = true
  try {
    const response = await resolveServiceFactProposal(props.serviceId, props.proposal.id, {
      decision,
      subject_id: selectedSubjectId.value || undefined,
    })
    const resolvedProposal = response?.data || {
      ...props.proposal,
      status: decision,
      subject_id: selectedSubjectId.value || props.proposal.subject_id,
    }
    status.value = resolvedProposal.status || decision
    notifyServiceSpaceUpdated(resolvedProposal)
    MessagePlugin.success(decision === 'confirmed' ? '服务档案已更新' : '已忽略档案更新')
  } catch (error: any) {
    const message = error?.response?.data?.message || error?.message || '提案处理失败'
    MessagePlugin.error(message)
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  void loadSubjects()
})
</script>

<style scoped lang="less">
.service-fact-proposal-card {
  width: min(100%, 560px);
  margin: 4px 0 8px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.proposal-header,
.proposal-title,
.proposal-actions,
.proposal-resolved,
.proposal-item-main,
.proposal-item-meta {
  display: flex;
  align-items: center;
}

.proposal-header {
  justify-content: space-between;
  gap: 12px;
}

.proposal-title {
  gap: 7px;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
}

.proposal-title-icon {
  flex: 0 0 auto;
  color: var(--td-brand-color);
}

.is-confirmed .proposal-title-icon,
.is-confirmed .proposal-resolved {
  color: var(--td-success-color);
}

.is-rejected .proposal-title-icon,
.is-rejected .proposal-resolved {
  color: var(--td-text-color-placeholder);
}

.proposal-status {
  flex: 0 0 auto;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.proposal-question {
  margin: 8px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.proposal-subject-picker {
  display: grid;
  grid-template-columns: auto minmax(180px, 1fr);
  gap: 6px 10px;
  align-items: center;
  margin-top: 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.proposal-subject-picker :deep(.t-select) {
  min-width: 0;
}

.proposal-error {
  grid-column: 2;
  color: var(--td-error-color);
}

.proposal-items {
  display: grid;
  gap: 7px;
  margin-top: 10px;
}

.proposal-item {
  padding: 8px 10px;
  border-left: 2px solid var(--td-brand-color);
  background: var(--td-bg-color-container-hover);
}

.proposal-item-main {
  justify-content: space-between;
  gap: 12px;
  line-height: 1.45;
}

.proposal-item-label {
  color: var(--td-text-color-secondary);
  flex: 0 0 auto;
}

.proposal-item-main strong {
  min-width: 0;
  text-align: right;
  overflow-wrap: anywhere;
}

.proposal-item-meta {
  justify-content: space-between;
  gap: 10px;
  margin-top: 4px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 1.45;
}

.proposal-evidence {
  min-width: 0;
  overflow-wrap: anywhere;
}

.proposal-confidence {
  flex: 0 0 auto;
}

.proposal-actions {
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.proposal-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 28px;
  padding: 3px 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 12px;
}

.proposal-action:hover:not(:disabled) {
  border-color: var(--td-brand-color);
  color: var(--td-brand-color);
}

.proposal-action.is-primary {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.proposal-action.is-primary:hover:not(:disabled) {
  border-color: var(--td-brand-color-hover);
  background: var(--td-brand-color-hover);
  color: var(--td-text-color-anti);
}

.proposal-action:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.proposal-resolved {
  gap: 5px;
  margin-top: 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

@media (max-width: 640px) {
  .proposal-subject-picker {
    grid-template-columns: 1fr;
  }

  .proposal-error {
    grid-column: auto;
  }

  .proposal-item-main,
  .proposal-item-meta {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;
  }

  .proposal-item-main strong {
    text-align: left;
  }
}
</style>
