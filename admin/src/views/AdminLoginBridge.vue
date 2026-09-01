<template>
  <main class="admin-login-bridge">
    <section class="admin-login-card">
      <div class="admin-login-bridge__mark">睿</div>
      <h1>登录睿乐 Admin</h1>
      <p>仅面向软件开发者、实施/运维和平台运营人员。</p>

      <form class="admin-login-form" @submit.prevent="handleLogin">
        <label>
          <span>手机号</span>
          <t-input
            v-model="phone"
            type="tel"
            autocomplete="tel"
            inputmode="tel"
            clearable
            placeholder="输入手机号"
            :disabled="loading"
          />
        </label>

        <label>
          <span>密码</span>
          <t-input
            v-model="password"
            type="password"
            autocomplete="current-password"
            clearable
            placeholder="输入密码（8-32个字符，包含字母和数字）"
            :disabled="loading"
            @enter="handleLogin"
          />
        </label>

        <t-alert v-if="error" theme="error" variant="light" :message="error" />

        <t-button
          theme="primary"
          size="large"
          type="submit"
          block
          :loading="loading"
          :disabled="!canSubmit"
        >
          登录
        </t-button>
      </form>

      <t-button variant="text" size="small" @click="openMainLogin">
        使用主应用登录
      </t-button>
    </section>
  </main>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { login, userInfoFromApi, type LoginResponse } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const phone = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const phonePattern = /^1[3-9]\d{9}$/

const redirectTarget = computed(() => {
  const raw = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
  if (!raw || raw === '/login') return '/'
  return raw.startsWith('/') ? raw : `/${raw}`
})

const mainLoginUrl = computed(() => {
  const base = import.meta.env.DEV
    ? (import.meta.env.VITE_ADMIN_LOGIN_URL || 'http://localhost:8081/login')
    : (import.meta.env.VITE_ADMIN_LOGIN_URL || '/login')
  const url = new URL(base, window.location.origin)
  url.searchParams.set('redirect', `${window.location.origin}${import.meta.env.BASE_URL || '/admin/'}${redirectTarget.value.replace(/^\//, '')}`)
  return url.toString()
})

const canSubmit = computed(() => {
  return phone.value.trim().length > 0 && password.value.length > 0
})

function applyLoginResponse(response: LoginResponse) {
  if (!response.user || !response.token) {
    throw new Error(response.message || '登录响应缺少账号信息')
  }

  const activeTenant = response.active_tenant || response.tenant || null
  const homeTenantIdRaw = response.user.tenant_id ?? activeTenant?.id ?? ''

  authStore.setUser(userInfoFromApi(response.user, homeTenantIdRaw))
  authStore.setToken(response.token)
  if (response.refresh_token) {
    authStore.setRefreshToken(response.refresh_token)
  }

  if (activeTenant) {
    authStore.setTenant({
      id: String(activeTenant.id) || '',
      name: activeTenant.name || '',
      owner_id: response.user.id || '',
      description: activeTenant.description,
      status: activeTenant.status,
      business: activeTenant.business,
      storage_quota: activeTenant.storage_quota,
      storage_used: activeTenant.storage_used,
      storage_usage: activeTenant.storage_usage,
      created_at: activeTenant.created_at || new Date().toISOString(),
      updated_at: activeTenant.updated_at || new Date().toISOString(),
    })
  } else {
    authStore.setTenant(null)
  }

  if (Array.isArray(response.memberships)) {
    authStore.setMemberships(response.memberships)
  }

  const activeIdNum = Number(activeTenant?.id)
  const homeIdNum = Number(homeTenantIdRaw)
  if (Number.isFinite(activeIdNum) && Number.isFinite(homeIdNum) && activeIdNum !== homeIdNum) {
    authStore.setSelectedTenant(activeIdNum, activeTenant?.name || null)
  } else {
    authStore.setSelectedTenant(null, null)
  }
}

async function handleLogin() {
  if (!canSubmit.value || loading.value) return
  loading.value = true
  error.value = ''

  try {
    const normalizedPhone = phone.value.trim()
    if (!phonePattern.test(normalizedPhone)) {
      throw new Error('请输入正确的手机号')
    }

    const response = await login({
      phone: normalizedPhone,
      password: password.value,
    })

    if (!response.success) {
      throw new Error(response.message || '登录失败')
    }

    applyLoginResponse(response)
    await authStore.refreshFromAuthMe()
    MessagePlugin.success('登录成功')
    await router.replace(authStore.hasValidTenant ? redirectTarget.value : '/no-tenant')
  } catch (err: any) {
    error.value = err?.message || '登录失败，请检查手机号和密码'
  } finally {
    loading.value = false
  }
}

function openMainLogin() {
  window.location.href = mainLoginUrl.value
}
</script>
