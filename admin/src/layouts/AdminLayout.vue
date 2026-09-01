<template>
  <div class="admin-layout">
    <aside class="admin-sidebar">
      <RouterLink class="admin-brand" to="/">
        <span class="admin-brand__mark">睿</span>
        <span class="admin-brand__text">
          <strong>睿乐 Admin</strong>
          <small>开发运营后台</small>
        </span>
      </RouterLink>

      <nav class="admin-nav" aria-label="Admin">
        <section v-for="group in visibleGroups" :key="group.key" class="admin-nav__group">
          <h2>{{ group.label }}</h2>
          <RouterLink
            v-for="item in group.items"
            :key="item.key"
            class="admin-nav__item"
            :class="{ 'is-active': activeNavKey === item.key }"
            :to="item.path"
          >
            <t-icon :name="item.icon" />
            <span>
              <strong>{{ item.label }}</strong>
              <small>{{ item.description }}</small>
            </span>
          </RouterLink>
        </section>
      </nav>
    </aside>

    <div class="admin-main">
      <header class="admin-topbar">
        <div class="admin-topbar__title">
          <h1>{{ routeTitle }}</h1>
          <p v-if="routeDescription">{{ routeDescription }}</p>
        </div>
        <div class="admin-topbar__actions">
          <AdminTenantSelect />
          <t-button variant="outline" size="small" @click="openWorkspace">
            <template #icon><t-icon name="jump" /></template>
            工作台
          </t-button>
          <t-dropdown :options="userOptions" trigger="click" placement="bottom-right" @click="handleUserMenu">
            <t-button variant="text" shape="square" size="small" class="admin-user-button">
              <template #icon><t-icon name="user-circle" /></template>
            </t-button>
          </t-dropdown>
        </div>
      </header>

      <main class="admin-content">
        <RouterView />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AdminTenantSelect from '@admin/components/AdminTenantSelect.vue'
import { ADMIN_NAV_GROUPS, type AdminNavItem } from '@admin/config/navigation'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const routeTitle = computed(() => route.meta.title || '睿乐 Admin')
const routeDescription = computed(() => route.meta.description || '')
const activeNavKey = computed(() => route.meta.navKey || 'overview')

const userOptions = computed(() => [
  {
    content: authStore.user?.email || authStore.user?.username || '当前账号',
    value: 'account',
    disabled: true,
  },
  {
    content: '退出登录',
    value: 'logout',
    theme: 'error',
  },
])

function canSeeItem(item: AdminNavItem): boolean {
  if (item.requiresSystemAdmin) return authStore.isSystemAdmin
  if (item.minRole) return authStore.hasRole(item.minRole)
  return true
}

const visibleGroups = computed(() => (
  ADMIN_NAV_GROUPS
    .map((group) => ({
      ...group,
      items: group.items.filter(canSeeItem),
    }))
    .filter((group) => group.items.length > 0)
))

function openWorkspace() {
  window.location.href = '/platform/knowledge-bases'
}

function handleUserMenu(data: { value: unknown }) {
  if (data.value !== 'logout') return
  authStore.logout()
  void router.replace({ name: 'adminLogin' })
}
</script>
