<template>
  <section class="admin-state-page">
    <div class="admin-state-page__icon">
      <t-icon name="tools" />
    </div>
    <h2>{{ route.meta.title || '模块建设中' }}</h2>
    <p>{{ route.meta.description || '该模块已在 admin 信息架构中占位，后续按 PRD 迁移。' }}</p>
    <div class="admin-state-page__actions">
      <t-button theme="primary" @click="goHome">返回概览</t-button>
      <t-button variant="outline" @click="openWorkspace">回到工作台</t-button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

function goHome() {
  void router.push('/')
}

function openWorkspace() {
  window.location.href = '/platform/knowledge-bases'
}
</script>
