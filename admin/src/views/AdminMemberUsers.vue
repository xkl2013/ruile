<template>
  <section class="membership-page">
    <section class="membership-page__panel">
      <div class="membership-page__toolbar">
        <t-input
          v-model="keyword"
          class="membership-page__search"
          clearable
          placeholder="输入用户名、邮箱或手机号"
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
        <t-button variant="outline" :loading="loading" @click="refresh">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
      </div>

      <div class="membership-page__section-heading">
        <div>
          <h3>用户列表</h3>
          <span>第 {{ page }} 页，每页 {{ pageSize }} 条</span>
        </div>
        <span v-if="!loading && users.length > 0" class="membership-page__count">
          当前显示 {{ users.length }} 个用户
        </span>
      </div>

      <div v-if="loading" class="membership-page__state">
        <t-loading size="small" />
        <span>正在加载用户...</span>
      </div>
      <div v-else-if="errorMessage" class="membership-page__state membership-page__state--error">
        <t-icon name="error-circle" />
        <span>{{ errorMessage }}</span>
      </div>
      <div v-else-if="users.length === 0" class="membership-page__state">
        <t-icon name="user-search" />
        <span>{{ keyword ? '没有找到匹配用户' : '系统中暂无用户' }}</span>
      </div>
      <div v-else class="membership-page__table-wrap">
        <table class="membership-page__table">
          <thead>
            <tr>
	              <th>用户</th>
	              <th>个人订阅</th>
	              <th>企业订阅</th>
	              <th>账户用量</th>
	              <th>账号状态</th>
	              <th>注册时间</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td>
                <div class="membership-page__user">
                  <t-avatar size="34px" :image="user.avatar">
                    {{ (user.username || user.email || '?').slice(0, 1).toUpperCase() }}
                  </t-avatar>
                  <div>
                    <strong>{{ user.username || '未设置用户名' }}</strong>
                    <span>{{ user.email || '未设置邮箱' }}</span>
                  </div>
                </div>
              </td>
              <td>
                <div v-if="user.personal_subscription" class="membership-page__subscription">
                  <strong>{{ user.personal_subscription.plan_name }}</strong>
                  <span>
                    {{ user.personal_subscription.tenant_name || `#${user.personal_subscription.tenant_id}` }}
                    · {{ subscriptionStatusLabel(user.personal_subscription.status) }}
                  </span>
                </div>
                <div v-else-if="user.tenant_id" class="membership-page__subscription">
                  <strong>#{{ user.tenant_id }}</strong>
                  <span>未配置订阅</span>
                </div>
                <span v-else class="membership-page__muted">未创建个人空间</span>
              </td>
              <td>
                <div
                  v-if="user.enterprise_subscriptions?.length"
                  class="membership-page__subscription-list"
                >
                  <div
                    v-for="subscription in user.enterprise_subscriptions"
                    :key="subscription.tenant_id"
                    class="membership-page__subscription"
                  >
                    <strong>{{ subscription.tenant_name }}</strong>
                    <span>{{ subscription.plan_name }} · {{ subscriptionStatusLabel(subscription.status) }}</span>
                  </div>
	                </div>
	                <span v-else class="membership-page__muted">未加入企业订阅</span>
	              </td>
	              <td>
	                <div v-if="user.usage_summary?.ledger_count" class="membership-page__usage">
	                  <strong>{{ formatTokenCount(totalTokens(user)) }} Token</strong>
	                  <span>
	                    {{ formatPoints(user.usage_summary.billed_point_micros) }} 积分
	                    · 企业 {{ user.usage_summary.enterprise_ledger_count || 0 }} 次
	                  </span>
	                  <span v-if="user.usage_summary.last_billing_at">
	                    最近 {{ formatDateTime(user.usage_summary.last_billing_at) }}
	                  </span>
	                </div>
	                <span v-else class="membership-page__muted">暂无调用</span>
	              </td>
	              <td>
                <div class="membership-page__tags">
                  <t-tag :theme="user.is_active ? 'success' : 'danger'" variant="light">
                    {{ user.is_active ? '正常' : '已停用' }}
                  </t-tag>
                  <t-tag v-if="user.is_system_admin" theme="primary" variant="light">
                    系统管理员
                  </t-tag>
                </div>
              </td>
              <td class="membership-page__muted">{{ formatDate(user.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <footer class="membership-page__pagination">
        <span v-if="errorMessage" class="membership-page__muted">数据加载失败</span>
        <span v-else class="membership-page__muted">系统用户按注册时间倒序展示</span>
        <div>
          <t-button
            variant="outline"
            size="small"
            :disabled="loading || page <= 1"
            @click="goToPage(page - 1)"
          >
            <template #icon><t-icon name="chevron-left" /></template>
            上一页
          </t-button>
          <t-button
            variant="outline"
            size="small"
            :disabled="loading || !hasMore"
            @click="goToPage(page + 1)"
          >
            下一页
            <template #suffixIcon><t-icon name="chevron-right" /></template>
          </t-button>
        </div>
      </footer>
    </section>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { listSystemUsers, type SystemUserSummary } from '@/api/system'

const pageSize = 20
const keyword = ref('')
const users = ref<SystemUserSummary[]>([])
const page = ref(1)
const hasMore = ref(false)
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

function formatDateTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function formatTokenCount(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0'
  return new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 0 }).format(value)
}

function formatPoints(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '0'
  return new Intl.NumberFormat('zh-CN', {
    maximumFractionDigits: 4,
  }).format(value / 1_000_000)
}

function totalTokens(user: SystemUserSummary) {
  const usage = user.usage_summary
  if (!usage) return 0
  return (usage.input_tokens || 0) + (usage.cached_tokens || 0) + (usage.output_tokens || 0)
}

function subscriptionStatusLabel(status: string) {
  const labels: Record<string, string> = {
    active: '生效',
    trialing: '试用',
    legacy: '兼容',
    canceled: '已取消',
    expired: '已过期',
  }
  return labels[status] || status || '未知'
}

async function loadUsers(targetPage = page.value) {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await listSystemUsers({
      keyword: keyword.value,
      page: targetPage,
      page_size: pageSize,
    })
    users.value = response.users || []
    page.value = response.page || targetPage
    hasMore.value = Boolean(response.has_more)
  } catch (error) {
    users.value = []
    hasMore.value = false
    errorMessage.value = getErrorMessage(error, '用户列表加载失败')
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  void loadUsers(1)
}

function refresh() {
  void loadUsers(page.value)
}

function goToPage(targetPage: number) {
  if (targetPage < 1 || loading.value) return
  void loadUsers(targetPage)
}

onMounted(() => {
  void loadUsers(1)
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
.membership-page__section-heading,
.membership-page__pagination {
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
  min-width: 1120px;
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

.membership-page__user {
  display: flex;
  align-items: center;
  gap: 11px;
}

.membership-page__user > div {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.membership-page__user strong {
  color: var(--admin-text);
  font-size: 14px;
  font-weight: 600;
}

.membership-page__user span {
  color: var(--admin-text-muted);
  overflow-wrap: anywhere;
}

.membership-page__mono {
  color: var(--admin-text-secondary);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 12px;
}

.membership-page__subscription,
.membership-page__subscription-list {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.membership-page__subscription strong {
  color: var(--admin-text);
  font-size: 13px;
  font-weight: 600;
}

.membership-page__subscription span {
  color: var(--admin-text-muted);
  font-size: 12px;
  overflow-wrap: anywhere;
}

.membership-page__subscription-list {
  max-width: 320px;
}

.membership-page__usage {
  display: grid;
  gap: 4px;
  min-width: 150px;
}

.membership-page__usage strong {
  color: var(--admin-text);
  font-size: 14px;
  line-height: 1.35;
}

.membership-page__usage span {
  color: var(--admin-text-muted);
  font-size: 12px;
  line-height: 1.35;
}

.membership-page__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
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

.membership-page__pagination {
  padding: 14px 20px 18px;
  border-top: 1px solid var(--admin-border);
}

.membership-page__pagination > div {
  display: flex;
  gap: 8px;
}

@media (max-width: 760px) {
  .membership-page {
    padding: 18px 14px 24px;
  }

  .membership-page__toolbar,
  .membership-page__section-heading,
  .membership-page__pagination {
    align-items: flex-start;
    flex-direction: column;
  }

  .membership-page__toolbar,
  .membership-page__section-heading,
  .membership-page__pagination {
    padding-right: 14px;
    padding-left: 14px;
  }

  .membership-page__search {
    width: 100%;
  }
}
</style>
