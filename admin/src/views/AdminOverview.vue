<template>
  <section class="admin-overview">
    <div class="admin-overview__grid">
      <article class="admin-overview__panel admin-overview__panel--identity">
        <div class="panel-heading">
          <t-icon name="home" />
          <h2>当前空间</h2>
        </div>
        <dl class="identity-list">
          <div>
            <dt>空间</dt>
            <dd>{{ authStore.currentTenantName || '未选择空间' }}</dd>
          </div>
          <div>
            <dt>角色</dt>
            <dd>{{ currentRoleText }}</dd>
          </div>
          <div>
            <dt>账号</dt>
            <dd>{{ authStore.user?.email || authStore.user?.username || '-' }}</dd>
          </div>
        </dl>
      </article>

      <article class="admin-overview__panel">
        <div class="panel-heading">
          <t-icon name="layers" />
          <h2>产品版本与空间形态</h2>
        </div>
        <p class="admin-overview__hint">
          内部运营视角，用于识别和配置客户空间，不是园长或园所成员的日常入口。
        </p>
        <div class="edition-pills">
          <RouterLink v-for="edition in editions" :key="edition.key" to="/workspaces/editions">
            <strong>{{ edition.name }}</strong>
            <span>{{ edition.target }}</span>
          </RouterLink>
        </div>
      </article>
    </div>

    <section class="admin-overview__modules">
      <div class="panel-heading">
        <t-icon name="app" />
        <h2>后台模块</h2>
      </div>
      <div class="module-grid">
        <RouterLink
          v-for="item in visibleItems"
          :key="item.key"
          :to="item.path"
          class="module-tile"
        >
          <t-icon :name="item.icon" />
          <span>
            <strong>{{ item.label }}</strong>
            <small>{{ item.description }}</small>
          </span>
        </RouterLink>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { ADMIN_NAV_ITEMS, type AdminNavItem } from '@admin/config/navigation'

const authStore = useAuthStore()

const editions = [
  { key: 'principal', name: '园长版', target: '园长个人空间' },
  { key: 'school', name: '园所版', target: '园所团队空间' },
  { key: 'group', name: '集团版', target: '多园所组织' },
]

const currentRoleText = computed(() => {
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

function canSeeItem(item: AdminNavItem): boolean {
  if (item.key === 'overview') return false
  if (item.requiresSystemAdmin) return authStore.isSystemAdmin
  if (item.minRole) return authStore.hasRole(item.minRole)
  return true
}

const visibleItems = computed(() => ADMIN_NAV_ITEMS.filter(canSeeItem))
</script>
