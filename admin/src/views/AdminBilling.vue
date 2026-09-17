<template>
  <section class="billing-admin">
    <div class="billing-admin__toolbar">
      <div>
        <h2>订阅与计费</h2>
        <p>管理套餐及其价格，并核对工作空间积分与模型用量账本。模型配置和模型价格统一在「模型」页面维护。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadAll">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-alert
      v-if="errorMessage"
      theme="error"
      :message="errorMessage"
      class="billing-admin__alert"
    />

    <t-tabs v-model="activeTab" class="billing-admin__tabs">
      <t-tab-panel value="plans" label="套餐">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table">
            <thead>
              <tr>
                <th>套餐</th>
                <th>版本</th>
                <th>空间类型</th>
                <th>包含存储</th>
                <th>周期积分</th>
                <th>价格</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="plan in plans" :key="plan.id">
                <td><strong>{{ plan.name }}</strong><span>{{ plan.code }}</span></td>
                <td>{{ plan.edition }}</td>
                <td>{{ plan.space_type }}</td>
                <td>{{ formatBytes(plan.included_storage_bytes) }}</td>
                <td>{{ formatPoints(plan.included_point_micros) }}</td>
                <td>
                  <div v-if="pricesByPlanID.get(plan.id)?.length" class="billing-admin__plan-prices">
                    <div v-for="price in pricesByPlanID.get(plan.id)" :key="price.id" class="billing-admin__plan-price">
                      <strong>{{ formatMoney(price.amount_minor, price.currency) }}</strong>
                      <span>
                        {{ price.billing_interval }} · {{ price.code }}
                        <t-tag v-if="price.is_default" theme="success" variant="light" size="small">默认</t-tag>
                      </span>
                    </div>
                  </div>
                  <span v-else>暂未设置</span>
                </td>
                <td>
                  <t-tag :theme="plan.status === 'active' ? 'success' : 'default'" variant="light">
                    {{ plan.status }}
                  </t-tag>
                </td>
              </tr>
              <tr v-if="!loading && plans.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无套餐</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="accounts" label="积分账户">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table">
            <thead>
              <tr>
                <th>工作空间</th>
                <th>空间类型</th>
                <th>积分余额</th>
                <th>账户版本</th>
                <th>创建时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="account in accounts" :key="account.id">
                <td><strong>{{ account.tenant_name }}</strong><span>#{{ account.tenant_id }}</span></td>
                <td>{{ account.space_type }}</td>
                <td><strong class="billing-admin__points">{{ formatPoints(account.balance_point_micros) }}</strong></td>
                <td>{{ account.version }}</td>
                <td>{{ formatDate(account.created_at) }}</td>
              </tr>
              <tr v-if="!loading && accounts.length === 0">
                <td colspan="5" class="billing-admin__empty">暂无积分账户</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>

      <t-tab-panel value="usage-ledgers" label="用量账本">
        <div class="billing-admin__table-wrap">
          <table class="billing-admin__table billing-admin__table--wide">
            <thead>
              <tr>
                <th>工作空间</th>
                <th>模型</th>
                <th>Token</th>
                <th>计费积分</th>
                <th>价格版本</th>
                <th>状态</th>
                <th>计费时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="ledger in usageLedgers" :key="ledger.id">
                <td><strong>{{ ledger.tenant_name }}</strong><span>#{{ ledger.tenant_id }}</span></td>
                <td><strong>{{ ledger.model_key || '-' }}</strong><span>{{ ledger.provider || '-' }}</span></td>
                <td>
                  <strong>{{ ledger.input_tokens }} / {{ ledger.output_tokens }}</strong>
                  <span>输入 / 输出，缓存 {{ ledger.cached_tokens }}</span>
                </td>
                <td>{{ formatPoints(ledger.billed_point_micros) }}</td>
                <td>{{ ledger.pricing_version ? `v${ledger.pricing_version}` : '-' }}</td>
                <td>
                  <t-tag :theme="ledgerStatusTheme(ledger.status)" variant="light">
                    {{ ledger.status }}
                  </t-tag>
                  <span v-if="ledger.failure_code">{{ ledger.failure_code }}</span>
                </td>
                <td>{{ formatDate(ledger.billing_at) }}</td>
              </tr>
              <tr v-if="!loading && usageLedgers.length === 0">
                <td colspan="7" class="billing-admin__empty">暂无模型用量账本</td>
              </tr>
            </tbody>
          </table>
        </div>
      </t-tab-panel>
    </t-tabs>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  listBillingCreditAccounts,
  listBillingPlans,
  listBillingPrices,
  listBillingUsageLedgers,
  type BillingCreditAccountItem,
  type BillingPlanItem,
  type BillingPriceItem,
  type BillingUsageLedgerItem,
} from '@/api/system'

const activeTab = ref('plans')
const loading = ref(false)
const errorMessage = ref('')
const plans = ref<BillingPlanItem[]>([])
const prices = ref<BillingPriceItem[]>([])
const accounts = ref<BillingCreditAccountItem[]>([])
const usageLedgers = ref<BillingUsageLedgerItem[]>([])

const pricesByPlanID = computed(() => {
  const rows = new Map<string, BillingPriceItem[]>()
  for (const price of prices.value) {
    const planPrices = rows.get(price.plan_id) || []
    planPrices.push(price)
    rows.set(price.plan_id, planPrices)
  }
  return rows
})

function formatBytes(value: number) {
  const bytes = Number(value) || 0
  if (bytes <= 0) return '按空间配置'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** index).toFixed(index === 0 ? 0 : 1)} ${units[index]}`
}

function formatPoints(pointMicros: number) {
  return `${new Intl.NumberFormat('zh-CN', { maximumFractionDigits: 2 }).format((Number(pointMicros) || 0) / 1_000_000)} 积分`
}

function formatMoney(amountMinor: number, currency: string) {
  return new Intl.NumberFormat('zh-CN', {
    style: 'currency',
    currency: currency || 'CNY',
  }).format((Number(amountMinor) || 0) / 100)
}

function formatDate(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

function ledgerStatusTheme(status: string): 'success' | 'warning' | 'danger' | 'default' {
  if (status === 'settled') return 'success'
  if (status === 'reconciliation' || status === 'unpriced') return 'danger'
  if (status === 'observed') return 'warning'
  return 'default'
}

async function loadAll() {
  loading.value = true
  errorMessage.value = ''
  try {
    const [planRows, priceRows, accountRows, ledgerRows] = await Promise.all([
      listBillingPlans(),
      listBillingPrices(),
      listBillingCreditAccounts(),
      listBillingUsageLedgers(),
    ])
    plans.value = planRows || []
    prices.value = priceRows || []
    accounts.value = accountRows || []
    usageLedgers.value = ledgerRows || []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '计费数据加载失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadAll()
})
</script>

<style scoped>
.billing-admin {
  margin: 28px 30px 36px;
  border: 1px solid var(--admin-border);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.billing-admin__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 20px;
  border-bottom: 1px solid var(--admin-border);
}

.billing-admin__toolbar h2,
.billing-admin__toolbar p {
  margin: 0;
}

.billing-admin__toolbar h2 {
  color: var(--admin-text);
  font-size: 18px;
}

.billing-admin__toolbar p {
  margin-top: 5px;
  color: var(--admin-text-muted);
  font-size: 13px;
}

.billing-admin__alert {
  margin: 16px 20px 0;
}

.billing-admin__tabs {
  padding: 0 20px 20px;
}

.billing-admin__table-wrap {
  overflow-x: auto;
  padding-top: 12px;
}

.billing-admin__table {
  width: 100%;
  min-width: 820px;
  border-collapse: collapse;
}

.billing-admin__table--wide {
  min-width: 1080px;
}

.billing-admin__table th,
.billing-admin__table td {
  padding: 13px 12px;
  border-bottom: 1px solid var(--admin-border);
  color: var(--admin-text-secondary);
  font-size: 13px;
  text-align: left;
  vertical-align: middle;
}

.billing-admin__table th {
  color: var(--admin-text-muted);
  font-weight: 500;
  background: var(--admin-surface-soft);
}

.billing-admin__table td strong,
.billing-admin__table td span {
  display: block;
}

.billing-admin__table td span {
  margin-top: 3px;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.billing-admin__points {
  color: var(--td-success-color);
}

.billing-admin__plan-prices {
  display: grid;
  gap: 6px;
}

.billing-admin__plan-price {
  display: grid;
  gap: 2px;
}

.billing-admin__plan-price span {
  display: flex !important;
  align-items: center;
  gap: 4px;
}

.billing-admin__empty {
  height: 120px;
  color: var(--admin-text-muted);
  text-align: center !important;
}

@media (max-width: 720px) {
  .billing-admin {
    margin: 16px;
  }

  .billing-admin__toolbar {
    align-items: flex-start;
  }

}
</style>
