<template>
  <t-dialog
    v-model:visible="dialogVisible"
    :header="$t('knowledgeBase.edit')"
    width="460px"
    :confirm-btn="{
      content: $t('common.confirm'),
      theme: 'primary',
      loading: saving,
    }"
    :cancel-btn="{ content: $t('common.cancel') }"
    @confirm="handleConfirm"
  >
    <t-form :data="form" label-align="top" @submit.prevent>
      <t-form-item :label="$t('knowledgeBase.name')" required>
        <t-input
          v-model="form.name"
          :maxlength="50"
          :placeholder="$t('initialization.knowledgeBaseNamePlaceholder')"
          autofocus
          @enter="handleConfirm"
        />
      </t-form-item>
      <t-form-item :label="$t('knowledgeBase.description')">
        <t-textarea
          v-model="form.description"
          :maxlength="200"
          :autosize="{ minRows: 3, maxRows: 6 }"
          :placeholder="$t('knowledgeEditor.basic.descriptionPlaceholder')"
        />
      </t-form-item>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { updateKnowledgeBase } from '@/api/knowledge-base'

const props = defineProps<{
  visible: boolean
  kbInfo: any
  canEdit: boolean
}>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  success: [data: any]
}>()

const { t } = useI18n()
const saving = ref(false)
const form = reactive({
  name: '',
  description: '',
})

const dialogVisible = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

watch(
  () => [props.visible, props.kbInfo?.id] as const,
  ([visible]) => {
    if (!visible || !props.kbInfo) return
    form.name = String(props.kbInfo.name || '')
    form.description = String(props.kbInfo.description || '')
  },
  { immediate: true },
)

async function handleConfirm() {
  if (saving.value || !props.kbInfo?.id) return
  if (!props.canEdit) {
    MessagePlugin.warning(t('knowledgeBase.updateFailed'))
    return
  }

  const name = form.name.trim()
  if (!name) {
    MessagePlugin.warning(t('initialization.pleaseEnterKnowledgeBaseName'))
    return
  }

  saving.value = true
  try {
    const response: any = await updateKnowledgeBase(props.kbInfo.id, {
      name,
      description: form.description.trim(),
    })
    const data = response?.data || {
      ...props.kbInfo,
      name,
      description: form.description.trim(),
    }
    emit('success', data)
    dialogVisible.value = false
    MessagePlugin.success(t('knowledgeBase.updateSuccess'))
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('knowledgeBase.updateFailed'))
  } finally {
    saving.value = false
  }
}
</script>
