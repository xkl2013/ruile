<template>
  <section class="admin-overview">
    <div class="admin-overview__grid">
      <article class="admin-overview__panel admin-overview__panel--identity">
        <div class="panel-heading">
          <t-icon name="dashboard" />
          <h2>系统运行</h2>
        </div>
        <div v-if="loading" class="admin-overview__loading">
          <t-loading size="small" />
          <span>正在读取系统信息...</span>
        </div>
        <t-alert v-else-if="systemError" theme="error" :message="systemError">
          <template #operation>
            <t-button size="small" variant="outline" @click="loadOverview">重试</t-button>
          </template>
        </t-alert>
        <template v-else>
          <div class="system-runtime__headline">
            <div>
              <span class="system-runtime__eyebrow">当前服务版本</span>
              <strong>{{ systemInfo?.version || '未知版本' }}</strong>
            </div>
            <t-tag :theme="systemHealthTheme" variant="light">{{ systemHealthLabel }}</t-tag>
          </div>
          <dl class="identity-list">
            <div>
              <dt>产品形态</dt>
              <dd>{{ formatEdition(systemInfo?.edition) }}</dd>
            </div>
            <div>
              <dt>运行时长</dt>
              <dd>{{ formatUptime(displayUptimeSeconds) }}</dd>
            </div>
            <div>
              <dt>数据库</dt>
              <dd>{{ systemInfo?.db_version || '未知' }}</dd>
            </div>
          </dl>
          <div class="system-runtime__engines">
            <div v-for="engine in systemEngines" :key="engine.label">
              <small>{{ engine.label }}</small>
              <strong>{{ engine.value }}</strong>
            </div>
          </div>
        </template>
      </article>

      <article class="admin-overview__panel">
        <div class="panel-heading">
          <t-icon name="layers" />
          <h2>系统规模</h2>
        </div>
        <p class="admin-overview__hint">
          {{ statsScopeText }}
        </p>
        <div v-if="statsLoading" class="admin-overview__loading">
          <t-loading size="small" />
          <span>正在汇总工作空间...</span>
        </div>
        <div v-else-if="statsError" class="system-stats__empty">
          <t-icon name="info-circle" />
          <span>{{ statsError }}</span>
        </div>
        <div v-else class="system-stats-grid">
          <div class="system-stat">
            <span>工作空间</span>
            <strong>{{ tenantStats.total }}</strong>
            <small>系统内全部空间</small>
          </div>
          <div class="system-stat">
            <span>活跃空间</span>
            <strong>{{ tenantStats.active }}</strong>
            <small>当前可用空间</small>
          </div>
          <div class="system-stat">
            <span>个人版</span>
            <strong>{{ tenantStats.personal }}</strong>
            <small>个人工作区</small>
          </div>
          <div class="system-stat">
            <span>企业版</span>
            <strong>{{ tenantStats.enterprise }}</strong>
            <small>企业工作区</small>
          </div>
          <div class="system-stat system-stat--wide">
            <span>存储使用量</span>
            <strong>{{ formatBytes(tenantStats.storageUsed) }}</strong>
            <small>{{ storageQuotaText }}</small>
          </div>
        </div>
        <div class="system-scale__footer">
          <span>版本能力</span>
          <div class="edition-pills">
            <RouterLink v-for="edition in editions" :key="edition.key" to="/workspaces/editions">
              <strong>{{ edition.name }}</strong>
              <span>{{ edition.target }}</span>
            </RouterLink>
          </div>
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
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { getSystemInfo, type SystemInfo } from '@/api/system'
import { listAllTenants, type TenantInfo } from '@/api/tenant'
import { useAuthStore } from '@/stores/auth'
import { ADMIN_NAV_ITEMS, type AdminNavItem } from '@admin/config/navigation'

const authStore = useAuthStore()

interface TenantStats {
  total: number
  active: number
  personal: number
  enterprise: number
  storageUsed: number
  storageQuota: number
  hasUnlimitedQuota: boolean
}

const emptyTenantStats: TenantStats = {
  total: 0,
  active: 0,
  personal: 0,
  enterprise: 0,
  storageUsed: 0,
  storageQuota: 0,
  hasUnlimitedQuota: false,
}

const editions = [
  { key: 'personal', name: '个人版', target: '个人工作区' },
  { key: 'enterprise', name: '企业版', target: '企业工作区' },
]

const systemInfo = ref<SystemInfo | null>(null)
const tenantStats = ref<TenantStats>({ ...emptyTenantStats })
const loading = ref(true)
const systemError = ref('')
const statsLoading = ref(false)
const statsError = ref('')
const uptimeTick = ref(0)
let uptimeTicker: ReturnType<typeof setInterval> | null = null

const canLoadWorkspaceStats = computed(() => Boolean(authStore.canAccessAllTenants))

const displayUptimeSeconds = computed(() => {
  void uptimeTick.value
  const info = systemInfo.value
  if (info?.started_at) {
    const boot = new Date(info.started_at).getTime()
    if (!Number.isNaN(boot)) {
      return Math.max(0, Math.floor((Date.now() - boot) / 1000))
    }
  }
  return info?.uptime_seconds ?? null
})

const systemEngines = computed(() => [
  { label: '关键词索引', value: systemInfo.value?.keyword_index_engine || '未知' },
  { label: '向量存储', value: systemInfo.value?.vector_store_engine || '未知' },
  { label: '图谱数据库', value: systemInfo.value?.graph_database_engine || '未启用' },
])

const systemHealthLabel = computed(() => (
  systemInfo.value?.db_migration_error ? '需处理' : '运行中'
))

const systemHealthTheme = computed(() => (
  systemInfo.value?.db_migration_error ? 'danger' : 'success'
))

const statsScopeText = computed(() => {
  if (canLoadWorkspaceStats.value) return '统计口径：当前系统内全部工作空间，不受顶部空间选择器影响。'
  return '当前账号没有跨空间统计权限，仅展示系统运行信息。'
})

const storageQuotaText = computed(() => {
  if (tenantStats.value.hasUnlimitedQuota) return '系统配额包含不限额空间'
  return `空间总配额 ${formatBytes(tenantStats.value.storageQuota)}`
})

function formatEdition(edition?: string): string {
  if (edition === 'lite') return 'Lite'
  if (edition === 'standard') return 'Standard'
  return edition || '未知'
}

function formatUptime(totalSeconds: number | null): string {
  if (totalSeconds == null) return '未知'
  const seconds = Math.max(0, Math.floor(totalSeconds))
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}天 ${hours}小时`
  if (hours > 0) return `${hours}小时 ${minutes}分钟`
  if (minutes > 0) return `${minutes}分钟`
  return `${seconds}秒`
}

function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const unitIndex = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** unitIndex).toFixed(unitIndex === 0 ? 0 : 1)} ${units[unitIndex]}`
}

function isActiveTenant(tenant: TenantInfo): boolean {
  const status = String(tenant.status || '').toLowerCase()
  return !['inactive', 'disabled', 'suspended', 'deleted'].includes(status)
}

function tenantEdition(tenant: TenantInfo): 'personal' | 'enterprise' {
  if (tenant.edition === 'personal' || tenant.space_type === 'personal') return 'personal'
  return 'enterprise'
}

function toFiniteNumber(value: unknown): number {
  const parsed = Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : 0
}

function buildTenantStats(tenants: TenantInfo[]): TenantStats {
  return tenants.reduce<TenantStats>((stats, tenant) => {
    const usedBytes = toFiniteNumber(tenant.storage_used ?? tenant.storage_usage?.used_bytes)
    const quotaBytes = Number(tenant.storage_quota ?? tenant.storage_usage?.quota_bytes ?? 0)
    const edition = tenantEdition(tenant)
    stats.total += 1
    if (isActiveTenant(tenant)) stats.active += 1
    stats[edition] += 1
    stats.storageUsed += usedBytes
    if (quotaBytes > 0) stats.storageQuota += quotaBytes
    else stats.hasUnlimitedQuota = true
    return stats
  }, { ...emptyTenantStats })
}

function errorMessage(error: unknown, fallback: string): string {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string' && message.trim()) return message
  }
  return fallback
}

async function loadOverview() {
  loading.value = true
  systemError.value = ''
  try {
    const response = await getSystemInfo()
    if (!response?.data) {
      throw new Error('系统信息为空')
    }
    systemInfo.value = response.data
  } catch (error) {
    systemError.value = errorMessage(error, '系统信息加载失败')
  } finally {
    loading.value = false
  }

  statsError.value = ''
  if (!canLoadWorkspaceStats.value) {
    tenantStats.value = { ...emptyTenantStats }
    statsError.value = '当前账号无法查看跨空间统计。'
    return
  }

  statsLoading.value = true
  try {
    const response = await listAllTenants()
    if (!response.success || !response.data) {
      throw new Error(response.message || '工作空间统计加载失败')
    }
    tenantStats.value = buildTenantStats(response.data.items || [])
  } catch (error) {
    tenantStats.value = { ...emptyTenantStats }
    statsError.value = errorMessage(error, '工作空间统计加载失败')
  } finally {
    statsLoading.value = false
  }
}

function canSeeItem(item: AdminNavItem): boolean {
  if (item.key === 'overview') return false
  if (item.requiresSystemAdmin) return authStore.isSystemAdmin
  if (item.minRole) return authStore.hasRole(item.minRole)
  return true
}

const visibleItems = computed(() => ADMIN_NAV_ITEMS.filter(canSeeItem))

onMounted(() => {
  void loadOverview()
  uptimeTicker = setInterval(() => {
    uptimeTick.value += 1
  }, 30_000)
})

onUnmounted(() => {
  if (uptimeTicker) clearInterval(uptimeTicker)
})
</script>
