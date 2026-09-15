<template>
  <section class="membership-page">
    <section class="membership-page__panel">
      <div class="membership-page__toolbar">
        <t-input
          v-model="keyword"
          class="membership-page__search"
          clearable
          placeholder="输入企业名称或描述"
          @enter="search"
        >
          <template #prefix-icon>
            <t-icon name="search" />
          </template>
        </t-input>
        <t-button theme="primary" :loading="loading" @click="search">
          <template #icon><t-icon name="search" /></template>
          搜索
        </t-button>
        <t-button theme="primary" @click="goToProvisioning">
          <template #icon><t-icon name="add" /></template>
          添加企业
        </t-button>
        <t-button variant="outline" :loading="loading" @click="loadEnterprises">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
      </div>

      <div class="membership-page__section-heading">
        <div>
          <h3>企业列表</h3>
          <span>企业空间按开通时间倒序展示</span>
        </div>
        <span v-if="!loading && enterprises.length > 0" class="membership-page__count">
          当前显示 {{ enterprises.length }} 家
        </span>
      </div>

      <div v-if="loading" class="membership-page__state">
        <t-loading size="small" />
        <span>正在加载企业...</span>
      </div>
      <div v-else-if="errorMessage" class="membership-page__state membership-page__state--error">
        <t-icon name="error-circle" />
        <span>{{ errorMessage }}</span>
      </div>
      <div v-else-if="enterprises.length === 0" class="membership-page__state">
        <t-icon name="building-1" />
        <span>{{ keyword ? '没有找到匹配企业' : '系统中暂无企业会员' }}</span>
      </div>
      <div v-else class="membership-page__table-wrap">
        <table class="membership-page__table">
          <thead>
            <tr>
              <th>企业</th>
              <th>状态</th>
              <th>员工数</th>
              <th>容量使用</th>
              <th>企业积分</th>
              <th>开通时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="enterprise in enterprises" :key="enterprise.id">
              <td>
                <div class="membership-page__enterprise">
                  <div class="membership-page__enterprise-icon">
                    <t-icon name="building-1" />
                  </div>
                  <div>
                    <strong>{{ enterprise.name || '未命名企业' }}</strong>
                    <span>{{ enterprise.description || '暂无企业描述' }}</span>
                  </div>
                </div>
              </td>
              <td>
                <t-tag :theme="isActive(enterprise.status) ? 'success' : 'danger'" variant="light">
                  {{ isActive(enterprise.status) ? '正常' : (enterprise.status || '不可用') }}
                </t-tag>
              </td>
              <td>
                <strong class="membership-page__member-count">
                  {{ enterprise.member_count ?? 0 }}
                </strong>
                <span class="membership-page__member-unit">人</span>
              </td>
              <td>
                <div class="membership-page__usage">
                  <strong>{{ formatBytes(enterprise.storage_used) }} / {{ formatBytes(enterprise.storage_quota) }}</strong>
                  <span>{{ formatUsage(enterprise.storage_usage?.usage_percent) }}</span>
                </div>
              </td>
              <td>
                <strong class="membership-page__credits">{{ enterprise.enterprise_credits ?? 0 }}</strong>
              </td>
              <td class="membership-page__muted">{{ formatDate(enterprise.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { listSystemEnterprises, type SystemEnterpriseSummary } from '@/api/system'

const router = useRouter()
const keyword = ref('')
const enterprises = ref<SystemEnterpriseSummary[]>([])
const loading = ref(false)
const errorMessage = ref('')

function getErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string' && message.trim()) return message
  }
  return fallback
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

function formatBytes(bytes?: number) {
  if (!Number.isFinite(bytes) || !bytes || bytes <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function formatUsage(value?: number) {
  if (!Number.isFinite(value)) return '暂无用量'
  return `${Number(value).toFixed(1)}% 已使用`
}

function isActive(status?: string) {
  return !['inactive', 'disabled', 'suspended', 'deleted'].includes(
    String(status || '').toLowerCase(),
  )
}

async function loadEnterprises() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await listSystemEnterprises({ keyword: keyword.value })
    enterprises.value = response.enterprises || []
  } catch (error) {
    enterprises.value = []
    errorMessage.value = getErrorMessage(error, '企业列表加载失败')
  } finally {
    loading.value = false
  }
}

function search() {
  void loadEnterprises()
}

function goToProvisioning() {
  void router.push({ name: 'adminEnterpriseProvisioning' })
}

onMounted(() => {
  void loadEnterprises()
})
</script>

<style scoped>
.membership-page {
  display: grid;
  gap: 18px;
  padding: 28px 30px 36px;
}

.membership-page__panel {
  border: 1px solid var(--admin-border);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.membership-page h3 {
  margin: 0;
}

.membership-page h3 {
  color: var(--admin-text);
  font-size: 16px;
  line-height: 1.4;
}

.membership-page__panel {
  overflow: hidden;
}

.membership-page__toolbar,
.membership-page__section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.membership-page__toolbar {
  padding: 18px 20px;
  border-bottom: 1px solid var(--admin-border);
  background: var(--admin-surface-soft);
}

.membership-page__search {
  width: min(460px, 100%);
}

.membership-page__toolbar > :first-child {
  margin-right: auto;
}

.membership-page__section-heading {
  padding: 18px 20px 14px;
}

.membership-page__section-heading > div {
  display: grid;
  gap: 3px;
}

.membership-page__section-heading span,
.membership-page__count,
.membership-page__muted {
  color: var(--admin-text-muted);
  font-size: 13px;
}

.membership-page__count {
  color: var(--admin-text-secondary);
}

.membership-page__table-wrap {
  overflow-x: auto;
}

.membership-page__table {
  width: 100%;
  min-width: 880px;
  border-collapse: collapse;
}

.membership-page__table th,
.membership-page__table td {
  padding: 14px 20px;
  border-top: 1px solid var(--admin-border);
  text-align: left;
  vertical-align: middle;
}

.membership-page__table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 650;
  background: var(--admin-surface-soft);
}

.membership-page__table td {
  color: var(--admin-text-secondary);
  font-size: 13px;
}

.membership-page__table tbody tr:hover {
  background: var(--admin-row-hover-bg);
}

.membership-page__enterprise {
  display: flex;
  align-items: center;
  gap: 11px;
  min-width: 240px;
}

.membership-page__enterprise-icon {
  display: grid;
  flex: 0 0 auto;
  width: 34px;
  height: 34px;
  place-items: center;
  border-radius: 8px;
  background: var(--admin-brand-soft);
  color: var(--admin-brand);
}

.membership-page__enterprise > div:last-child {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.membership-page__enterprise strong {
  color: var(--admin-text);
  font-size: 14px;
  font-weight: 600;
}

.membership-page__enterprise span {
  max-width: 320px;
  overflow: hidden;
  color: var(--admin-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.membership-page__usage {
  display: grid;
  gap: 3px;
}

.membership-page__usage strong,
.membership-page__credits,
.membership-page__member-count {
  color: var(--admin-text);
  font-size: 13px;
  font-weight: 600;
}

.membership-page__member-unit {
  margin-left: 3px;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.membership-page__usage span {
  color: var(--admin-text-muted);
  font-size: 12px;
}

.membership-page__state {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  color: var(--admin-text-muted);
}

.membership-page__state--error {
  color: var(--admin-danger);
}

@media (max-width: 760px) {
  .membership-page {
    padding: 18px 14px 24px;
  }

  .membership-page__toolbar,
  .membership-page__section-heading {
    align-items: flex-start;
    flex-direction: column;
  }

  .membership-page__toolbar,
  .membership-page__section-heading {
    padding-right: 14px;
    padding-left: 14px;
  }

  .membership-page__search {
    width: 100%;
  }
}
</style>
