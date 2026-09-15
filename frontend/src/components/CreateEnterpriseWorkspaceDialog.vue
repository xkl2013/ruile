<template>
  <t-dialog :visible="visible" width="500px" :on-confirm="handleSubmit" :on-close="handleClose"
    :confirm-btn="{ content: $t('tenant.enterprise.submit'), loading: submitting, theme: 'primary' }"
    :cancel-btn="{ content: $t('tenant.enterprise.cancel') }" :close-on-overlay-click="!submitting"
    :close-on-esc-keydown="!submitting" @update:visible="onVisibleUpdate">
    <template #header>
      <span class="enterprise-dialog-header">
        <t-icon name="usergroup-add" size="20px" class="enterprise-dialog-header-icon" aria-hidden="true" />
        <span>{{ $t('tenant.enterprise.dialogTitle') }}</span>
      </span>
    </template>

    <p class="enterprise-dialog-tip">{{ $t('tenant.enterprise.dialogSubtitle') }}</p>

    <t-form ref="formRef" :data="form" :rules="formRules" label-align="top" class="enterprise-dialog-form"
      @submit.prevent>
      <t-form-item :label="$t('tenant.enterprise.nameLabel')" name="name">
        <t-input v-model="form.name" :placeholder="$t('tenant.enterprise.namePlaceholder')" :maxlength="128"
          autofocus :disabled="submitting" @enter="handleSubmit" />
      </t-form-item>
      <t-form-item :label="$t('tenant.enterprise.descriptionLabel')" name="description">
        <t-textarea v-model="form.description" :placeholder="$t('tenant.enterprise.descriptionPlaceholder')"
          :maxlength="512" :autosize="{ minRows: 3, maxRows: 5 }" :disabled="submitting" />
      </t-form-item>
    </t-form>
  </t-dialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin, type FormInstanceFunctions, type FormRule } from 'tdesign-vue-next'
import { createEnterpriseWorkspace, type TenantInfo } from '@/api/tenant'
import { switchTenant } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'

const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'created', tenant: TenantInfo): void
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const formRef = ref<FormInstanceFunctions | null>(null)
const submitting = ref(false)

const form = reactive({
  name: '',
  description: '',
})

const formRules: Record<string, FormRule[]> = {
  name: [
    {
      validator: (value: string) => (value ?? '').trim().length > 0,
      message: t('tenant.enterprise.nameRequired'),
      trigger: 'blur',
    },
  ],
}

watch(
  () => props.visible,
  (open) => {
    if (open) {
      form.name = ''
      form.description = ''
      requestAnimationFrame(() => formRef.value?.clearValidate?.())
    }
  },
)

const onVisibleUpdate = (next: boolean) => {
  if (!next && submitting.value) return
  emit('update:visible', next)
}

const handleClose = () => {
  if (submitting.value) return
  emit('update:visible', false)
}

const handleSubmit = async () => {
  if (submitting.value) return
  const validateResult = await formRef.value?.validate?.()
  if (validateResult !== true) return

  submitting.value = true
  try {
    const response = await createEnterpriseWorkspace({
      name: form.name.trim(),
      description: form.description.trim() || undefined,
    })
    if (!response.success || !response.data) {
      MessagePlugin.error(response.message || t('tenant.enterprise.failed'))
      return
    }

    const tenant = response.data
    const targetTenantID = Number(tenant.id)
    const switchResponse = await switchTenant({
      tenant_id: targetTenantID,
      refresh_token: authStore.refreshToken || undefined,
    })

    if (switchResponse.success && switchResponse.token) {
      authStore.setToken(switchResponse.token)
      if (switchResponse.refresh_token) {
        authStore.setRefreshToken(switchResponse.refresh_token)
      }
      authStore.setSelectedTenant(targetTenantID, tenant.name)
      await authStore.refreshFromAuthMe()
      stashTenantSwitchToast({
        name: tenant.name,
        role: t('tenantMember.role.owner'),
        roleEnum: 'owner',
      })
      await Promise.race([
        persistLastActiveTenantPreference(targetTenantID),
        new Promise((resolve) => setTimeout(resolve, 500)),
      ])
      MessagePlugin.success(t('tenant.enterprise.success'))
      emit('created', tenant)
      emit('update:visible', false)
      navigateAfterTenantSwitch()
      return
    }

    await authStore.refreshFromAuthMe()
    MessagePlugin.warning(t('tenant.enterprise.createdButSwitchFailed'))
    emit('created', tenant)
    emit('update:visible', false)
  } catch (error: any) {
    console.error('Failed to create enterprise workspace:', error)
    MessagePlugin.error(error?.message || t('tenant.enterprise.failed'))
  } finally {
    submitting.value = false
  }
}
</script>

<style lang="less" scoped>
.enterprise-dialog-header {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.enterprise-dialog-header-icon {
  flex-shrink: 0;
  color: var(--td-brand-color);
}

.enterprise-dialog-tip {
  margin: 0 0 16px;
  font-size: 13px;
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.enterprise-dialog-form {
  :deep(.t-form__item):last-child {
    margin-bottom: 0;
  }
}
</style>
