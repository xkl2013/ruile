<template>
  <div class="subscription-usage">
    <header class="section-header">
      <h2>{{ t('tenant.subscriptionUsage.title') }}</h2>
      <p>{{ t('tenant.subscriptionUsage.description') }}</p>
    </header>

    <div v-if="loading" class="loading-state">
      <t-loading size="small" />
      <span>{{ t('tenant.subscriptionUsage.loading') }}</span>
    </div>

    <div v-else-if="error" class="error-state">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadUsage">{{ t('tenant.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="usage-content">
      <div class="usage-card-list">
        <article v-for="card in usageCards" :key="card.key" class="usage-card">
          <div class="resource-usage">
            <div class="resource-heading">
              <span>{{ t('tenant.subscriptionUsage.resourceUsage') }}</span>
            </div>

            <div v-if="card.showPoolDetails" class="resource-row">
              <div class="resource-label-row">
                <div class="resource-label">
                  <t-icon name="layers" />
                  <span>{{ t('tenant.subscriptionUsage.currentPlan') }}</span>
                </div>
                <span class="resource-primary">{{ card.planName }}</span>
              </div>
              <div class="resource-meta">
                <span>{{ card.planCode }}</span>
                <span>{{ card.periodText }}</span>
              </div>
            </div>

            <div v-if="card.showPoolDetails" class="resource-row">
              <div class="resource-label-row">
                <div class="resource-label">
                  <t-icon name="save" />
                  <span>{{ t('tenant.subscriptionUsage.storageUsage') }}</span>
                </div>
                <span class="resource-primary">
                  {{ t('tenant.subscriptionUsage.usedOfTotal', {
                    used: card.storage.usedText,
                    total: card.storage.quotaText,
                  }) }}
                </span>
              </div>
              <div class="resource-meta">
                <span>{{ card.storage.percentText }}</span>
                <span>
                  {{ t('tenant.subscriptionUsage.remaining', {
                    amount: card.storage.remainingText,
                  }) }}
                </span>
              </div>
              <t-progress
                v-if="!card.storage.unlimited"
                :percentage="card.storage.progress"
                :show-info="false"
                size="small"
                :status="card.storage.progressStatus"
              />
              <div v-else class="unlimited-track">
                <span />
              </div>
            </div>

            <div v-if="card.showPoolDetails" class="resource-row resource-row--credits">
              <div class="resource-label-row">
                <div class="resource-label">
                  <t-icon name="wealth-1" />
                  <span>{{ t('tenant.subscriptionUsage.periodCredits') }}</span>
                </div>
                <span class="resource-primary resource-primary--credits">
                  {{ t('tenant.subscriptionUsage.points', { count: card.periodCreditText }) }}
                </span>
              </div>
              <div class="resource-meta">
                <span>{{ t('tenant.subscriptionUsage.periodCreditsDescription') }}</span>
                <span>{{ card.periodText }}</span>
              </div>
              <div class="resource-meta resource-meta--credits-detail">
                <span>{{ t('tenant.subscriptionUsage.periodCreditsUsed', { count: card.periodUsedCreditText }) }}</span>
                <span>{{ t('tenant.subscriptionUsage.periodCreditsRemaining', { count: card.periodRemainingCreditText }) }}</span>
              </div>
              <t-progress
                :percentage="card.periodCreditProgress"
                :show-info="false"
                size="small"
                :status="card.periodCreditProgress >= 100 ? 'success' : 'active'"
              />
            </div>

            <div v-if="card.showPoolDetails" class="resource-row resource-row--credits">
              <div class="resource-label-row">
                <div class="resource-label">
                  <t-icon name="wallet" />
                  <span>{{ t('tenant.subscriptionUsage.creditBalance') }}</span>
                </div>
                <span class="resource-primary resource-primary--credits">
                  {{ t('tenant.subscriptionUsage.points', { count: card.balanceCreditText }) }}
                </span>
              </div>
              <div class="resource-meta">
                <span>{{ t('tenant.subscriptionUsage.creditAvailable') }}</span>
                <span>{{ card.creditSource }}</span>
              </div>
            </div>

            <div v-else-if="card.memberUsage" class="resource-row resource-row--credits">
              <div class="resource-label-row">
                <div class="resource-label">
                  <t-icon name="user" />
                  <span>{{ t('tenant.subscriptionUsage.myMonthlyUsage') }}</span>
                </div>
                <span class="resource-primary resource-primary--credits">
                  {{ t('tenant.subscriptionUsage.points', { count: card.memberUsage.usedText }) }}
                </span>
              </div>
              <div class="resource-meta">
                <span>{{ card.memberUsage.limitText }}</span>
                <span>{{ card.memberUsage.overageText }}</span>
              </div>
              <t-progress
                v-if="!card.memberUsage.unlimited"
                :percentage="card.memberUsage.progress"
                :show-info="false"
                size="small"
                :status="card.memberUsage.progress >= 100 ? 'success' : 'active'"
              />
              <div v-else class="unlimited-track">
                <span />
              </div>
            </div>
          </div>
        </article>
      </div>

      <section v-if="catalog" class="billing-catalog">
        <div class="billing-catalog__heading">
          <div>
            <h3>套餐与增值服务</h3>
            <p>查看当前空间可用的套餐、积分包和存储包。</p>
          </div>
          <t-tag :theme="catalog.payment.enabled ? 'success' : 'warning'" variant="light">
            {{ catalog.payment.enabled ? '在线支付已启用' : '在线支付未配置' }}
          </t-tag>
        </div>

        <t-alert
          v-if="!catalog.payment.enabled"
          theme="warning"
          :message="catalog.payment.reason || '当前暂不支持在线购买，请联系管理员处理。'"
        />

        <div class="billing-catalog__group">
          <h4>可用套餐</h4>
          <div v-if="catalog.plans.length" class="billing-catalog__grid">
            <article v-for="plan in catalog.plans" :key="plan.id" class="billing-catalog__item">
              <div class="billing-catalog__item-heading">
                <strong>{{ plan.name }}</strong>
                <span>{{ plan.code }}</span>
              </div>
              <p>{{ plan.description || '暂无套餐说明' }}</p>
              <div class="billing-catalog__facts">
                <span>{{ formatCatalogBytes(plan.included_storage_bytes) }} 存储</span>
                <span>{{ formatCatalogPoints(plan.included_point_micros) }} 周期积分</span>
              </div>
              <div class="billing-catalog__price">
                {{ planPriceLabel(plan.id) }}
                <t-button size="small" variant="outline" disabled>
                  {{ catalog.payment.enabled ? '立即购买' : '暂不可购买' }}
                </t-button>
              </div>
            </article>
          </div>
          <div v-else class="billing-catalog__empty">暂无可用套餐。</div>
        </div>

        <div class="billing-catalog__group">
          <h4>积分与存储包</h4>
          <div v-if="catalog.purchase_items.length" class="billing-catalog__grid">
            <article v-for="item in catalog.purchase_items" :key="item.id" class="billing-catalog__item">
              <div class="billing-catalog__item-heading">
                <strong>{{ item.name }}</strong>
                <span>{{ item.item_type === 'topup' ? '积分包' : '存储包' }}</span>
              </div>
              <p>{{ item.description || '暂无商品说明' }}</p>
              <div class="billing-catalog__facts">
                <span v-if="item.item_type === 'topup'">{{ formatCatalogPoints(item.credit_point_micros) }} 积分</span>
                <span v-else>{{ formatCatalogBytes(item.storage_quota_bytes) }} 存储</span>
                <span>{{ formatCatalogMoney(item.amount_cents, item.currency) }}</span>
              </div>
              <div class="billing-catalog__price">
                <span>{{ item.code }}</span>
                <t-button size="small" variant="outline" disabled>
                  {{ catalog.payment.enabled ? '立即购买' : '暂不可购买' }}
                </t-button>
              </div>
            </article>
          </div>
          <div v-else class="billing-catalog__empty">暂无可用增值服务。</div>
        </div>
      </section>

      <section class="billing-orders">
        <div class="billing-orders__heading">
          <div>
            <h3>订单记录</h3>
            <p>仅展示当前工作空间的订阅、积分和存储订单。</p>
          </div>
          <t-tag variant="light">{{ orders.length }}</t-tag>
        </div>
        <div v-if="orders.length" class="billing-orders__table-wrap">
          <table class="billing-orders__table">
            <thead>
              <tr>
                <th>订单号</th>
                <th>类型</th>
                <th>内容</th>
                <th>金额</th>
                <th>状态</th>
                <th>时间</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="order in orders" :key="order.id">
                <td><strong>{{ order.order_no }}</strong><span>{{ order.provider || '-' }}</span></td>
                <td>{{ orderTypeLabel(order.order_type) }}</td>
                <td>{{ order.plan_name || order.item_name || '-' }}</td>
                <td>{{ formatCatalogMoney(order.amount_cents, order.currency) }}</td>
                <td><t-tag :theme="orderStatusTheme(order.status)" variant="light">{{ orderStatusLabel(order.status) }}</t-tag></td>
                <td>{{ formatUsageTime(order.paid_at || order.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div v-else class="billing-orders__empty">暂无订单记录。</div>
      </section>

      <section v-if="isEnterprise && canManageEnterprisePolicy" class="enterprise-policy">
        <div class="enterprise-policy__heading">
          <div>
            <h3>{{ t('tenant.subscriptionUsage.enterprisePolicyTitle') }}</h3>
            <p>{{ t('tenant.subscriptionUsage.enterprisePolicyDescription') }}</p>
          </div>
          <t-button
            v-if="canManageEnterprisePolicy"
            theme="primary"
            :loading="savingPolicy"
            @click="saveEnterprisePolicy"
          >
            {{ t('tenant.subscriptionUsage.savePolicy') }}
          </t-button>
        </div>

        <t-alert
          v-if="policyError"
          theme="error"
          :message="policyError"
          class="enterprise-policy__error"
        />

        <div v-else class="enterprise-policy__fields">
          <div class="enterprise-policy__field">
            <div>
              <strong>{{ t('tenant.subscriptionUsage.defaultMemberMonthlyLimit') }}</strong>
              <span>{{ t('tenant.subscriptionUsage.defaultMemberMonthlyLimitDescription') }}</span>
            </div>
            <div class="enterprise-policy__control enterprise-policy__limit">
              <t-input-number
                v-model="policyForm.defaultMemberMonthlyLimitPoints"
                :min="0"
                :max="1000000000"
                :decimal-places="0"
                :disabled="!canManageEnterprisePolicy"
                theme="column"
              />
              <span>{{ t('tenant.subscriptionUsage.creditUnit') }}</span>
            </div>
          </div>

          <div class="enterprise-policy__field">
            <div>
              <strong>{{ t('tenant.subscriptionUsage.overagePolicy') }}</strong>
              <span>{{ t('tenant.subscriptionUsage.overagePolicyDescription') }}</span>
            </div>
            <t-radio-group
              v-model="policyForm.memberOveragePolicy"
              :disabled="!canManageEnterprisePolicy"
            >
              <t-radio-button value="block">
                {{ t('tenant.subscriptionUsage.overageBlock') }}
              </t-radio-button>
              <t-radio-button value="use_enterprise_balance">
                {{ t('tenant.subscriptionUsage.overageUseBalance') }}
              </t-radio-button>
            </t-radio-group>
          </div>
        </div>
      </section>

      <section v-if="isEnterprise && canManageEnterprisePolicy" class="enterprise-member-usage">
        <div class="enterprise-member-usage__heading">
          <div>
            <h3>{{ t('tenant.subscriptionUsage.memberUsageTitle') }}</h3>
            <p>{{ t('tenant.subscriptionUsage.memberUsageDescription') }}</p>
          </div>
          <t-tag variant="light">{{ enterpriseMemberRows.length }}</t-tag>
        </div>
        <t-alert
          v-if="memberUsageError"
          theme="error"
          :message="memberUsageError"
          class="enterprise-member-usage__error"
        />
        <div v-else class="enterprise-member-usage__table-wrap">
          <table class="enterprise-member-usage__table">
            <thead>
              <tr>
                <th>{{ t('tenant.subscriptionUsage.member') }}</th>
                <th>{{ t('tenant.subscriptionUsage.memberUsage') }}</th>
                <th>{{ t('tenant.subscriptionUsage.memberStrategy') }}</th>
                <th>{{ t('tenant.subscriptionUsage.memberOverage') }}</th>
                <th>{{ t('tenant.subscriptionUsage.memberLastBilling') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="member in enterpriseMemberRows" :key="member.userId">
                <td>
                  <strong>{{ member.name }}</strong>
                  <span>{{ member.role }}</span>
                </td>
                <td>
                  <strong>{{ member.usedText }}</strong>
                  <span>{{ member.limitText }}</span>
                </td>
                <td>{{ member.strategy }}</td>
                <td>{{ member.overage }}</td>
                <td>{{ member.lastBilling }}</td>
              </tr>
              <tr v-if="enterpriseMemberRows.length === 0">
                <td colspan="5" class="enterprise-member-usage__empty">
                  {{ t('tenant.subscriptionUsage.noMemberUsage') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <section class="recent-usage">
        <div class="recent-usage__heading">
          <div>
            <h3>{{ t('tenant.subscriptionUsage.recentUsage') }}</h3>
            <p>{{ t('tenant.subscriptionUsage.recentUsageDescription') }}</p>
          </div>
          <t-tag variant="light">{{ usageItems.length }}</t-tag>
        </div>
        <div class="recent-usage__table-wrap">
          <table class="recent-usage__table">
            <thead>
              <tr>
                <th>{{ t('tenant.subscriptionUsage.model') }}</th>
                <th>{{ t('tenant.subscriptionUsage.tokens') }}</th>
                <th>{{ t('tenant.subscriptionUsage.billedCredits') }}</th>
                <th>{{ t('tenant.subscriptionUsage.usageStatus') }}</th>
                <th>{{ t('tenant.subscriptionUsage.billingTime') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="item in usageItems" :key="item.id">
                <td><strong>{{ item.model_key || '-' }}</strong><span>{{ item.provider || '-' }}</span></td>
                <td>
                  <strong>{{ item.input_tokens }} / {{ item.output_tokens }}</strong>
                  <span>{{ t('tenant.subscriptionUsage.inputOutputTokens') }}</span>
                </td>
                <td>{{ formatCredits(item.billed_point_micros) }}</td>
                <td>
                  <t-tag :theme="usageStatusTheme(item.status)" variant="light" size="small">
                    {{ item.status }}
                  </t-tag>
                  <span v-if="item.failure_code">{{ item.failure_code }}</span>
                </td>
                <td>{{ formatUsageTime(item.billing_at) }}</td>
              </tr>
              <tr v-if="usageItems.length === 0">
                <td colspan="5" class="recent-usage__empty">
                  {{ t('tenant.subscriptionUsage.noRecentUsage') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  getBillingOverview,
  getBillingUsage,
  getBillingCatalog,
  getBillingOrders,
  getMemberCreditAllocations,
  getTenantBillingPolicy,
  updateTenantBillingPolicy,
  type BillingOverview,
  type BillingCatalog,
  type BillingCatalogResponse,
  type BillingOrderItem,
  type BillingOrdersResponse,
  type BillingUsageItem,
  type BillingUsageResponse,
  type TenantBillingPolicy,
} from '@/api/billing'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'

type ProgressStatus = 'active' | 'success' | 'warning' | 'error'

interface StorageDisplay {
  usedText: string
  quotaText: string
  remainingText: string
  percentText: string
  progress: number
  progressStatus: ProgressStatus
  unlimited: boolean
}

interface UsageCard {
  key: string
  planName: string
  planCode: string
  periodText: string
  storage: StorageDisplay
  periodCreditText: string
  periodUsedCreditText: string
  periodRemainingCreditText: string
  periodCreditProgress: number
  balanceCreditText: string
  creditSource: string
  showPoolDetails: boolean
  memberUsage?: {
    usedText: string
    limitText: string
    overageText: string
    progress: number
    unlimited: boolean
  }
}

interface EnterpriseMemberUsageRow {
  userId: string
  name: string
  role: string
  usedText: string
  limitText: string
  strategy: string
  overage: string
  lastBilling: string
  usedMicros: number
}

const { t, locale } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const error = ref('')
const overview = ref<BillingOverview | null>(null)
const usageItems = ref<BillingUsageItem[]>([])
const catalog = ref<BillingCatalog | null>(null)
const orders = ref<BillingOrderItem[]>([])
const enterprisePolicy = ref<TenantBillingPolicy | null>(null)
const policyError = ref('')
const enterpriseMemberRows = ref<EnterpriseMemberUsageRow[]>([])
const memberUsageError = ref('')
const savingPolicy = ref(false)
const policyForm = reactive<{
  defaultMemberMonthlyLimitPoints: number
  memberOveragePolicy: 'block' | 'use_enterprise_balance'
}>({
  defaultMemberMonthlyLimitPoints: 100,
  memberOveragePolicy: 'block',
})
let loadSequence = 0

const activeTenantId = computed(() =>
  Number(
    authStore.enterpriseSettingsTenantId
      || authStore.effectiveTenantId
      || authStore.currentTenantId
      || authStore.user?.tenant_id
      || 0,
  ),
)
const isEnterprise = computed(() => overview.value?.space_type === 'organization')
const canManageEnterprisePolicy = computed(() =>
  authStore.hasRoleInTenant(activeTenantId.value, 'admin'),
)

const toFiniteNumber = (value: unknown, fallback = 0) => {
  const normalized = Number(value)
  return Number.isFinite(normalized) ? normalized : fallback
}

const formatBytes = (bytes: number) => {
  const normalized = Math.max(0, toFiniteNumber(bytes, 0))
  if (normalized === 0) return '0 B'

  const unit = 1024
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(Math.floor(Math.log(normalized) / Math.log(unit)), units.length - 1)
  return `${parseFloat((normalized / Math.pow(unit, index)).toFixed(2))} ${units[index]}`
}

const formatPercent = (value: number) => {
  const normalized = Math.max(0, toFiniteNumber(value, 0))
  const rounded = Math.round(normalized * 100) / 100
  return `${String(rounded).replace(/\.0+$/, '').replace(/(\.\d*[1-9])0+$/, '$1')}%`
}

const formatCredits = (pointMicros: number) =>
  new Intl.NumberFormat(locale.value || 'zh-CN', {
    maximumFractionDigits: 2,
  }).format(Math.max(0, toFiniteNumber(pointMicros, 0)) / 1_000_000)

const formatUsageTime = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '-'
  return new Intl.DateTimeFormat(locale.value || 'zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

const formatCatalogMoney = (amountMinor: number, currency = 'CNY') =>
  new Intl.NumberFormat(locale.value || 'zh-CN', {
    style: 'currency',
    currency,
    minimumFractionDigits: 2,
  }).format(Math.max(0, toFiniteNumber(amountMinor, 0)) / 100)

const formatCatalogPoints = (pointMicros: number) =>
  formatCredits(pointMicros)

const formatCatalogBytes = (bytes: number) =>
  formatBytes(bytes)

const planPriceLabel = (planID: string) => {
  const price = catalog.value?.prices.find((item) => item.plan_id === planID && item.is_default)
    || catalog.value?.prices.find((item) => item.plan_id === planID)
  if (!price) return '价格待配置'
  if (price.amount_minor === 0) return '免费'
  const interval = price.billing_interval === 'year'
    ? '/年'
    : price.billing_interval === 'month'
      ? '/月'
      : ''
  return `${formatCatalogMoney(price.amount_minor, price.currency)}${interval}`
}

const orderTypeLabel = (type: BillingOrderItem['order_type']) => {
  if (type === 'subscription' || type === 'manual_contract') return '订阅'
  if (type === 'topup') return '积分包'
  return '存储包'
}

const orderStatusLabel = (status: string) => {
  const labels: Record<string, string> = {
    paid: '已完成',
    pending: '待支付',
    failed: '失败',
    expired: '已过期',
    closed: '已关闭',
    reconciliation: '对账中',
  }
  return labels[status] || status || '未知'
}

const orderStatusTheme = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'paid') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'failed' || status === 'expired' || status === 'reconciliation') return 'danger'
  return 'default'
}

const usageStatusTheme = (status: string): 'success' | 'warning' | 'danger' | 'default' => {
  if (status === 'settled') return 'success'
  if (status === 'unpriced' || status === 'reconciliation') return 'danger'
  if (status === 'observed') return 'warning'
  return 'default'
}

const buildStorageDisplay = (raw: BillingOverview['storage']): StorageDisplay => {
  const quotaBytes = Math.max(0, toFiniteNumber(raw.quota_bytes, 0))
  const usedBytes = Math.max(0, toFiniteNumber(raw.used_bytes, 0))
  const unlimited = raw.unlimited === true || quotaBytes <= 0
  const remainingBytes = unlimited
    ? 0
    : Math.max(0, toFiniteNumber(raw.remaining_bytes, quotaBytes - usedBytes))
  const usagePercent = unlimited
    ? 0
    : Math.max(0, toFiniteNumber(raw.usage_percent, quotaBytes > 0 ? (usedBytes / quotaBytes) * 100 : 0))
  const status = raw.status
    || (usedBytes >= quotaBytes ? 'exceeded' : usagePercent >= 80 ? 'warning' : 'ok')

  return {
    usedText: formatBytes(usedBytes),
    quotaText: unlimited ? t('tenant.storage.unlimited') : formatBytes(quotaBytes),
    remainingText: unlimited ? t('tenant.storage.unlimited') : formatBytes(remainingBytes),
    percentText: unlimited
      ? t('tenant.storage.usedOfUnlimited', { used: formatBytes(usedBytes) })
      : formatPercent(usagePercent),
    progress: Math.min(100, Math.round(usagePercent * 100) / 100),
    progressStatus: status === 'exceeded'
      ? 'error'
      : status === 'warning'
        ? 'warning'
        : usagePercent >= 100
          ? 'success'
          : 'active',
    unlimited,
  }
}

const formatPeriod = (billing: BillingOverview) => {
  const start = billing.subscription.current_period_start
  const end = billing.subscription.current_period_end
  if (!start || !end) return t('tenant.subscriptionUsage.noFixedPeriod')
  const formatter = new Intl.DateTimeFormat(locale.value || 'zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  return `${formatter.format(new Date(start))} - ${formatter.format(new Date(end))}`
}

const usageCards = computed<UsageCard[]>(() => {
  if (!overview.value) return []
  const billing = overview.value
  const isEnterprise = billing.space_type === 'organization'
  const showPoolDetails = billing.storage.visible !== false && billing.credits.visible !== false
  const periodTotalMicros = Math.max(0, toFiniteNumber(billing.credits.period_point_micros, 0))
  const periodUsedMicros = Math.max(0, toFiniteNumber(billing.credits.period_used_point_micros, 0))
  const periodRemainingMicros = Math.max(
	    0,
	    toFiniteNumber(
	      billing.credits.period_remaining_point_micros,
	      periodTotalMicros - periodUsedMicros,
	    ),
	  )
  const memberUsage = billing.member_usage
  const memberLimitMicros = Math.max(
    0,
    toFiniteNumber(memberUsage?.monthly_limit_point_micros, 0),
  )
  const memberUsedMicros = Math.max(
    0,
    toFiniteNumber(memberUsage?.monthly_used_point_micros, 0),
  )
  const memberUnlimited = memberUsage?.limit_mode === 'unlimited'
  return [{
    key: billing.space_type,
    planName: billing.plan.name,
    planCode: billing.plan.code,
    periodText: formatPeriod(billing),
    storage: buildStorageDisplay(billing.storage),
    periodCreditText: formatCredits(billing.credits.period_point_micros),
    periodUsedCreditText: formatCredits(periodUsedMicros),
    periodRemainingCreditText: formatCredits(periodRemainingMicros),
    periodCreditProgress: periodTotalMicros > 0
      ? Math.min(100, Math.round((periodUsedMicros / periodTotalMicros) * 10000) / 100)
      : 0,
    balanceCreditText: formatCredits(billing.credits.balance_point_micros),
    creditSource: isEnterprise
      ? t('tenant.subscriptionUsage.enterpriseCreditSource')
      : t('tenant.subscriptionUsage.personalCreditSource'),
    showPoolDetails,
    memberUsage: !showPoolDetails && memberUsage
      ? {
        usedText: formatCredits(memberUsedMicros),
        limitText: memberUnlimited
          ? t('tenant.subscriptionUsage.unlimited')
          : t('tenant.subscriptionUsage.memberLimit', {
            limit: formatCredits(memberLimitMicros),
          }),
        overageText: memberUsage.overage_policy === 'use_enterprise_balance'
          ? t('tenant.subscriptionUsage.overageUseBalance')
          : t('tenant.subscriptionUsage.overageBlock'),
        progress: memberUnlimited || memberLimitMicros <= 0
          ? 0
          : Math.min(100, Math.round((memberUsedMicros / memberLimitMicros) * 10000) / 100),
        unlimited: memberUnlimited,
      }
      : undefined,
  }]
})

const memberName = (member: TenantMember) =>
  member.username?.trim() || member.email?.trim() || member.user_id

const memberStrategyLabel = (mode: string) => {
  const key = mode === 'custom' || mode === 'unlimited' ? mode : 'inherit'
  return t(`tenant.subscriptionUsage.strategy.${key}`)
}

const loadEnterpriseMemberUsage = async () => {
  enterpriseMemberRows.value = []
  memberUsageError.value = ''
  if (!activeTenantId.value || !canManageEnterprisePolicy.value) return
  try {
    const [allocationResponse, members] = await Promise.all([
      getMemberCreditAllocations(activeTenantId.value),
      fetchAllTenantMembers(activeTenantId.value),
    ])
    if (!allocationResponse.success) {
      throw new Error(
        allocationResponse.message || t('tenant.subscriptionUsage.memberUsageLoadFailed'),
      )
    }
    const allocationByUserID = new Map(
      (allocationResponse.data || []).map((allocation) => [allocation.user_id, allocation]),
    )
    const defaultLimitMicros = Math.max(
      0,
      enterprisePolicy.value?.default_member_monthly_limit_point_micros || 0,
    )
    enterpriseMemberRows.value = members
      .filter((member) => member.status === 'active')
      .map((member) => {
        const allocation = allocationByUserID.get(member.user_id)
        const mode = allocation?.limit_mode || 'inherit'
        const effectiveLimitMicros = mode === 'custom'
          ? Math.max(0, allocation?.effective_monthly_limit_point_micros || allocation?.monthly_limit_point_micros || 0)
          : mode === 'unlimited'
            ? 0
            : defaultLimitMicros
        const usedMicros = Math.max(0, allocation?.used_point_micros || 0)
        return {
          userId: member.user_id,
          name: memberName(member),
          role: t(`tenantMember.role.${member.role}`),
          usedText: t('tenant.subscriptionUsage.points', { count: formatCredits(usedMicros) }),
          limitText: mode === 'unlimited'
            ? t('tenant.subscriptionUsage.unlimited')
            : t('tenant.subscriptionUsage.memberLimit', {
              limit: formatCredits(effectiveLimitMicros),
            }),
          strategy: memberStrategyLabel(mode),
          overage: allocation?.effective_overage_policy === 'use_enterprise_balance'
            ? t('tenant.subscriptionUsage.overageUseBalance')
            : t('tenant.subscriptionUsage.overageBlock'),
          lastBilling: allocation?.last_billing_at
            ? formatUsageTime(allocation.last_billing_at)
            : '-',
          usedMicros,
        }
      })
      .sort((left, right) => right.usedMicros - left.usedMicros || left.name.localeCompare(right.name))
  } catch (err: any) {
    memberUsageError.value = err?.message || t('tenant.subscriptionUsage.memberUsageLoadFailed')
  }
}

const applyEnterprisePolicy = (policy: TenantBillingPolicy) => {
  enterprisePolicy.value = policy
  policyForm.defaultMemberMonthlyLimitPoints = Math.max(
    0,
    Math.round(toFiniteNumber(policy.default_member_monthly_limit_point_micros, 0) / 1_000_000),
  )
  policyForm.memberOveragePolicy = policy.member_overage_policy === 'use_enterprise_balance'
    ? 'use_enterprise_balance'
    : 'block'
}

const saveEnterprisePolicy = async () => {
  if (!activeTenantId.value || savingPolicy.value || !canManageEnterprisePolicy.value) return
  savingPolicy.value = true
  policyError.value = ''
  try {
    const response = await updateTenantBillingPolicy(activeTenantId.value, {
      default_member_monthly_limit_points: Math.max(
        0,
        Math.trunc(toFiniteNumber(policyForm.defaultMemberMonthlyLimitPoints, 0)),
      ),
      member_overage_policy: policyForm.memberOveragePolicy,
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || t('tenant.subscriptionUsage.policySaveFailed'))
    }
    applyEnterprisePolicy(response.data)
    await loadEnterpriseMemberUsage()
    MessagePlugin.success(t('tenant.subscriptionUsage.policySaved'))
  } catch (err: any) {
    policyError.value = err?.message || t('tenant.subscriptionUsage.policySaveFailed')
  } finally {
    savingPolicy.value = false
  }
}

const loadUsage = async () => {
  const sequence = ++loadSequence

  loading.value = true
  error.value = ''
  overview.value = null
  usageItems.value = []
  catalog.value = null
  orders.value = []
  enterprisePolicy.value = null
  policyError.value = ''
  enterpriseMemberRows.value = []
  memberUsageError.value = ''

  if (!activeTenantId.value) {
    error.value = t('tenant.subscriptionUsage.personalLoadFailed')
    loading.value = false
    return
  }

  const [response, usageResponse, catalogResponse, ordersResponse] = await Promise.all([
    getBillingOverview(activeTenantId.value),
    getBillingUsage(20, activeTenantId.value).catch(
      (): BillingUsageResponse => ({ success: false }),
    ),
    getBillingCatalog(activeTenantId.value).catch(
      (): BillingCatalogResponse => ({ success: false }),
    ),
    getBillingOrders(20, activeTenantId.value).catch(
      (): BillingOrdersResponse => ({ success: false }),
    ),
  ])

  if (sequence !== loadSequence) return

  if (!response.success || !response.data) {
    error.value = response.message || t('tenant.subscriptionUsage.personalLoadFailed')
    loading.value = false
    return
  }

  overview.value = response.data
  usageItems.value = usageResponse.success && usageResponse.data ? usageResponse.data : []
  catalog.value = catalogResponse.success && catalogResponse.data ? catalogResponse.data : null
  orders.value = ordersResponse.success && ordersResponse.data ? ordersResponse.data : []
  if (response.data.space_type === 'organization' && canManageEnterprisePolicy.value) {
    try {
      const policyResponse = await getTenantBillingPolicy(activeTenantId.value)
      if (sequence !== loadSequence) return
      if (!policyResponse.success || !policyResponse.data) {
        throw new Error(policyResponse.message || t('tenant.subscriptionUsage.policyLoadFailed'))
	      }
      applyEnterprisePolicy(policyResponse.data)
      await loadEnterpriseMemberUsage()
    } catch (err: any) {
      policyError.value = err?.message || t('tenant.subscriptionUsage.policyLoadFailed')
    }
  }
  loading.value = false
}

watch(
  activeTenantId,
  () => {
    void loadUsage()
  },
  { immediate: true },
)
</script>

<style lang="less" scoped>
.subscription-usage {
  width: 100%;
}

.section-header {
  margin-bottom: 28px;

  h2 {
    margin: 0 0 8px;
    color: var(--td-text-color-primary);
    font-size: 22px;
    font-weight: 600;
    line-height: 1.35;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 1.6;
  }
}

.loading-state {
  min-height: 260px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.error-state {
  padding-top: 8px;
}

.usage-content,
.usage-card-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.recent-usage {
  margin-top: 8px;
  border-top: 1px solid var(--td-component-stroke);
  padding-top: 24px;
}

.enterprise-policy {
  padding: 24px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.enterprise-policy__heading,
.enterprise-policy__field {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.enterprise-policy__heading {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--td-component-stroke);

  h3,
  p {
    margin: 0;
  }

  h3 {
    color: var(--td-text-color-primary);
    font-size: 16px;
  }

  p {
    margin-top: 5px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.enterprise-policy__error {
  margin-top: 18px;
}

.enterprise-policy__fields {
  display: flex;
  flex-direction: column;
}

.enterprise-policy__field {
  padding: 20px 0;

  & + & {
    border-top: 1px solid var(--td-component-stroke);
  }

  &:last-child {
    padding-bottom: 0;
  }

  > div:first-child {
    display: flex;
    flex-direction: column;
    gap: 5px;
    min-width: 0;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 14px;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
}

.enterprise-policy__control {
  flex-shrink: 0;
}

.enterprise-policy__limit {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input-number) {
    width: 180px;
  }
}

.enterprise-member-usage {
  padding: 24px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.enterprise-member-usage__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h3,
  p {
    margin: 0;
  }

  h3 {
    color: var(--td-text-color-primary);
    font-size: 16px;
  }

  p {
    margin-top: 5px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.enterprise-member-usage__error {
  margin-top: 16px;
}

.enterprise-member-usage__table-wrap {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.enterprise-member-usage__table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;

  th,
  td {
    padding: 12px 14px;
    border-bottom: 1px solid var(--td-component-stroke);
    color: var(--td-text-color-secondary);
    font-size: 13px;
    text-align: left;
  }

  th {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-weight: 500;
  }

  tbody tr:last-child td {
    border-bottom: 0;
  }

  td strong,
  td span {
    display: block;
  }

  td strong {
    color: var(--td-text-color-primary);
  }

  td span {
    margin-top: 3px;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.enterprise-member-usage__empty {
  height: 88px;
  text-align: center !important;
}

.recent-usage__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;

  h3,
  p {
    margin: 0;
  }

  h3 {
    color: var(--td-text-color-primary);
    font-size: 16px;
  }

  p {
    margin-top: 5px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.recent-usage__table-wrap {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.recent-usage__table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;

  th,
  td {
    padding: 12px 14px;
    border-bottom: 1px solid var(--td-component-stroke);
    color: var(--td-text-color-secondary);
    font-size: 13px;
    text-align: left;
  }

  th {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-weight: 500;
  }

  tbody tr:last-child td {
    border-bottom: 0;
  }

  td strong,
  td span {
    display: block;
  }

  td strong {
    color: var(--td-text-color-primary);
  }

  td span {
    margin-top: 3px;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.recent-usage__empty {
  height: 96px;
  text-align: center !important;
}

.billing-catalog,
.billing-orders {
  display: grid;
  gap: 18px;
  padding: 24px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.billing-catalog__heading,
.billing-orders__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;

  h3,
  p {
    margin: 0;
  }

  h3 {
    color: var(--td-text-color-primary);
    font-size: 16px;
  }

  p {
    margin-top: 5px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.billing-catalog__group {
  display: grid;
  gap: 12px;

  h4 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 14px;
  }
}

.billing-catalog__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 12px;
}

.billing-catalog__item {
  display: grid;
  gap: 10px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);

  p {
    min-height: 36px;
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
}

.billing-catalog__item-heading {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;

  strong {
    min-width: 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.billing-catalog__facts {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.billing-catalog__price {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--td-brand-color);
  font-size: 14px;
  font-weight: 600;
}

.billing-catalog__empty,
.billing-orders__empty {
  padding: 24px;
  color: var(--td-text-color-placeholder);
  text-align: center;
}

.billing-orders__table-wrap {
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.billing-orders__table {
  width: 100%;
  min-width: 760px;
  border-collapse: collapse;

  th,
  td {
    padding: 12px 14px;
    border-bottom: 1px solid var(--td-component-stroke);
    color: var(--td-text-color-secondary);
    font-size: 13px;
    text-align: left;
  }

  th {
    background: var(--td-bg-color-secondarycontainer);
    font-weight: 500;
  }

  tbody tr:last-child td {
    border-bottom: 0;
  }

  td strong,
  td span {
    display: block;
  }

  td strong {
    color: var(--td-text-color-primary);
  }

  td span {
    margin-top: 3px;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
  }
}

.enterprise-warning {
  margin-bottom: 2px;
}

.usage-card {
  padding: 24px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
}

.resource-usage {
  min-width: 0;
}

.resource-heading {
  margin-bottom: 18px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.resource-row {
  padding-bottom: 20px;

  & + & {
    padding-top: 20px;
    border-top: 1px solid var(--td-component-stroke);
  }

  &:last-child {
    padding-bottom: 0;
  }
}

.resource-label-row,
.resource-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.resource-label {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;

  .t-icon {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);
  }
}

.resource-primary {
  flex-shrink: 0;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;

  &--credits {
    color: var(--td-brand-color);
    font-size: 18px;
    font-weight: 600;
  }
}

.resource-meta {
  margin: 8px 0 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.unlimited-track {
  width: 100%;
  height: 4px;
  overflow: hidden;
  border-radius: 2px;
  background: var(--td-bg-color-secondarycontainer);

  span {
    display: block;
    width: 100%;
    height: 100%;
    background: var(--td-brand-color-light-active);
  }
}

@media (max-width: 720px) {
  .usage-card {
    padding: 20px;
  }

  .enterprise-policy__heading,
  .enterprise-policy__field,
  .enterprise-member-usage__heading,
  .billing-catalog__heading,
  .billing-orders__heading {
    flex-direction: column;
  }
}

@media (max-width: 480px) {
  .resource-label-row,
  .resource-meta {
    align-items: flex-start;
  }

  .resource-primary {
    white-space: normal;
    text-align: right;
  }
}
</style>
