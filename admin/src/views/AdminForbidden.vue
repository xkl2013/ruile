<template>
  <section class="admin-state-page">
    <div class="admin-state-page__icon admin-state-page__icon--warning">
      <t-icon name="lock-on" />
    </div>
    <h2>无权限访问</h2>
    <p>{{ message }}</p>
    <div class="admin-state-page__actions">
      <t-button theme="primary" @click="router.push('/')">返回概览</t-button>
      <t-button variant="outline" @click="openWorkspace">回到工作台</t-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const message = computed(() => {
  const reason = route.query.reason
  if (reason === 'system_admin') return '该页面只允许系统管理员访问。'
  if (typeof reason === 'string') return `该页面需要 ${reason} 或更高空间角色。`
  return '当前账号或当前空间没有访问该后台页面的权限。'
})

function openWorkspace() {
  window.location.href = '/platform/knowledge-bases'
}
</script>

