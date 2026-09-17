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

            <div class="resource-row">
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

            <div class="resource-row">
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

            <div class="resource-row resource-row--credits">
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
            </div>

            <div class="resource-row resource-row--credits">
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
              <t-progress
                :percentage="card.balanceCreditMicros > 0 ? 100 : 0"
                :show-info="false"
                size="small"
                status="success"
              />
            </div>
          </div>
        </article>
      </div>

      <section v-if="isEnterprise" class="enterprise-policy">
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
  getTenantBillingPolicy,
  updateTenantBillingPolicy,
  type BillingOverview,
  type BillingUsageItem,
  type BillingUsageResponse,
  type TenantBillingPolicy,
} from '@/api/billing'
import { useAuthStore } from '@/stores/auth'

type ProgressStatus = 'success' | 'warning' | 'error'

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
  balanceCreditText: string
  balanceCreditMicros: number
  creditSource: string
}

const { t, locale } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const error = ref('')
const overview = ref<BillingOverview | null>(null)
const usageItems = ref<BillingUsageItem[]>([])
const enterprisePolicy = ref<TenantBillingPolicy | null>(null)
const policyError = ref('')
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
    progressStatus: status === 'exceeded' ? 'error' : status === 'warning' ? 'warning' : 'success',
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
  return [{
    key: billing.space_type,
    planName: billing.plan.name,
    planCode: billing.plan.code,
    periodText: formatPeriod(billing),
    storage: buildStorageDisplay(billing.storage),
    periodCreditText: formatCredits(billing.credits.period_point_micros),
    balanceCreditText: formatCredits(billing.credits.balance_point_micros),
    balanceCreditMicros: billing.credits.balance_point_micros,
    creditSource: isEnterprise
      ? t('tenant.subscriptionUsage.enterpriseCreditSource')
      : t('tenant.subscriptionUsage.personalCreditSource'),
  }]
})

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
  enterprisePolicy.value = null
  policyError.value = ''

  if (!activeTenantId.value) {
    error.value = t('tenant.subscriptionUsage.personalLoadFailed')
    loading.value = false
    return
  }

  const [response, usageResponse] = await Promise.all([
    getBillingOverview(activeTenantId.value),
    getBillingUsage(20, activeTenantId.value).catch(
      (): BillingUsageResponse => ({ success: false }),
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
  if (response.data.space_type === 'organization') {
    try {
      const policyResponse = await getTenantBillingPolicy(activeTenantId.value)
      if (sequence !== loadSequence) return
      if (!policyResponse.success || !policyResponse.data) {
        throw new Error(policyResponse.message || t('tenant.subscriptionUsage.policyLoadFailed'))
      }
      applyEnterprisePolicy(policyResponse.data)
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
  .enterprise-policy__field {
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
