<template>
  <main class="admin-login-bridge">
    <section>
      <div class="admin-login-bridge__mark admin-login-bridge__mark--muted">404</div>
      <h1>页面不存在</h1>
      <p>这个 admin 路径没有对应页面。</p>
      <t-button theme="primary" size="large" @click="goHome">
        返回 Admin
      </t-button>
    </section>
  </main>
</template>

<script setup lang="ts">
function goHome() {
  window.location.href = `${import.meta.env.BASE_URL || '/admin/'}`
}
</script>

