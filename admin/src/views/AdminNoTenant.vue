<template>
  <section class="admin-state-page">
    <div class="admin-state-page__icon admin-state-page__icon--warning">
      <t-icon name="home" />
    </div>
    <h2>需要工作空间</h2>
    <p>当前管理员账号还没有可操作空间。请先创建或加入测试空间，或由系统管理员分配空间权限。</p>
    <div class="admin-state-page__actions">
      <t-button theme="primary" @click="openOnboarding">开通空间</t-button>
      <t-button variant="outline" @click="router.push('/workspaces/editions')">查看产品版本</t-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

function openOnboarding() {
  window.location.href = '/onboarding/workspace'
}
</script>
