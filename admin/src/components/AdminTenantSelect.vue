<template>
  <div class="admin-tenant-select">
    <t-select
      :model-value="selectedTenantValue"
      :options="tenantOptions"
      :disabled="tenantOptions.length === 0"
      size="small"
      class="admin-tenant-select__control"
      @change="handleTenantChange"
    />
    <t-tag size="small" variant="light" :theme="roleTagTheme">
      {{ roleLabel }}
    </t-tag>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

const tenantOptions = computed(() => {
  const map = new Map<number, string>()

  for (const membership of authStore.memberships ?? []) {
    map.set(Number(membership.tenant_id), membership.tenant_name || `空间 ${membership.tenant_id}`)
  }

  if (authStore.tenant?.id) {
    map.set(Number(authStore.tenant.id), authStore.tenant.name || `空间 ${authStore.tenant.id}`)
  }

  return Array.from(map.entries()).map(([value, label]) => ({
    label,
    value,
  }))
})

const selectedTenantValue = computed(() => {
  const selected = authStore.selectedTenantId
  if (selected) return Number(selected)
  if (authStore.tenant?.id) return Number(authStore.tenant.id)
  return undefined
})

const roleLabel = computed(() => {
  if (authStore.isSystemAdmin) return '系统管理员'
  const role = authStore.currentTenantRole
  const labels: Record<string, string> = {
    owner: 'Owner',
    admin: 'Admin',
    contributor: 'Contributor',
    viewer: 'Viewer',
  }
  return labels[role] || '未加入空间'
})

const roleTagTheme = computed(() => {
  if (authStore.isSystemAdmin) return 'danger'
  if (authStore.currentTenantRole === 'owner') return 'warning'
  if (authStore.currentTenantRole === 'admin') return 'success'
  return 'default'
})

async function handleTenantChange(value: unknown) {
  const id = Number(value)
  if (!Number.isFinite(id)) return
  const matched = tenantOptions.value.find((option) => Number(option.value) === id)
  authStore.setSelectedTenant(id, matched?.label ?? null)
  const ok = await authStore.refreshFromAuthMe()
  if (ok) {
    MessagePlugin.success('已切换空间')
    return
  }
  MessagePlugin.error('切换空间失败，请重新登录')
}
</script>

