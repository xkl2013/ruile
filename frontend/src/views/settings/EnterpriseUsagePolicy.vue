<template>
  <div class="enterprise-usage">
    <div v-if="loading" class="loading-state">
      <t-loading size="small" />
      <span>{{ t('tenant.subscriptionUsage.loading') }}</span>
    </div>

    <div v-else-if="error" class="error-state">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadEnterpriseUsage">
            {{ t('tenant.retry') }}
          </t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="enterprise-usage__content">
      <section class="enterprise-credit-pool">
        <div class="enterprise-credit-pool__heading">
          <div>
            <h2>{{ t('tenant.subscriptionUsage.enterpriseCreditPoolTitle') }}</h2>
            <p>{{ t('tenant.subscriptionUsage.enterpriseCreditPoolDescription') }}</p>
          </div>
          <t-tag variant="light">{{ enterpriseCreditPool.periodText }}</t-tag>
        </div>

        <div class="enterprise-credit-pool__grid">
          <div class="enterprise-credit-pool__metric">
            <span>{{ t('tenant.subscriptionUsage.creditAvailable') }}</span>
            <strong>{{ t('tenant.subscriptionUsage.points', { count: enterpriseCreditPool.balanceAvailableText }) }}</strong>
            <small>{{ t('tenant.subscriptionUsage.enterpriseCreditSource') }}</small>
          </div>
          <div class="enterprise-credit-pool__metric">
            <span>{{ t('tenant.subscriptionUsage.enterpriseBalanceUsage') }}</span>
            <strong>
              {{ t('tenant.subscriptionUsage.usedOfTotal', {
                used: enterpriseCreditPool.balanceUsedText,
                total: enterpriseCreditPool.balanceTotalText,
              }) }}
            </strong>
            <small>{{ t('tenant.subscriptionUsage.enterpriseBalanceRemaining', { count: enterpriseCreditPool.balanceAvailableText }) }}</small>
          </div>
          <div class="enterprise-credit-pool__metric">
            <span>{{ t('tenant.subscriptionUsage.periodCredits') }}</span>
            <strong>
              {{ t('tenant.subscriptionUsage.usedOfTotal', {
                used: enterpriseCreditPool.periodUsedText,
                total: enterpriseCreditPool.periodTotalText,
              }) }}
            </strong>
            <small>{{ t('tenant.subscriptionUsage.periodCreditsRemaining', { count: enterpriseCreditPool.periodRemainingText }) }}</small>
            <t-progress
              :percentage="enterpriseCreditPool.periodProgress"
              :show-info="false"
              size="small"
              :status="enterpriseCreditPool.periodProgress >= 100 ? 'success' : 'active'"
            />
          </div>
        </div>
      </section>

      <section class="enterprise-policy">
        <div class="enterprise-policy__heading">
          <div>
            <h2>{{ t('tenant.subscriptionUsage.enterprisePolicyTitle') }}</h2>
            <p>{{ t('tenant.subscriptionUsage.enterprisePolicyDescription') }}</p>
          </div>
          <t-button
            theme="primary"
            :loading="savingPolicy"
            :disabled="!canManageEnterprisePolicy"
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

      <section class="enterprise-member-usage">
        <div class="enterprise-member-usage__heading">
          <div>
            <h2>{{ t('tenant.subscriptionUsage.memberUsageTitle') }}</h2>
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
                <th>{{ t('tenant.subscriptionUsage.memberStorage') }}</th>
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
                <td>
                  <strong>{{ member.storageUsedText }}</strong>
                  <span>{{ member.storageTransactionText }}</span>
                </td>
                <td>{{ member.strategy }}</td>
                <td>{{ member.overage }}</td>
                <td>{{ member.lastBilling }}</td>
              </tr>
              <tr v-if="enterpriseMemberRows.length === 0">
                <td colspan="6" class="enterprise-member-usage__empty">
                  {{ t('tenant.subscriptionUsage.noMemberUsage') }}
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
  getMemberCreditAllocations,
  getMemberStorageUsage,
  getTenantBillingPolicy,
  updateTenantBillingPolicy,
  type BillingOverview,
  type TenantBillingPolicy,
} from '@/api/billing'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'

interface EnterpriseMemberUsageRow {
  userId: string
  name: string
  role: string
  usedText: string
  limitText: string
  storageUsedText: string
  storageTransactionText: string
  strategy: string
  overage: string
  lastBilling: string
  usedMicros: number
}

const { t, locale } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const error = ref('')
const enterpriseOverview = ref<BillingOverview | null>(null)
const enterprisePolicy = ref<TenantBillingPolicy | null>(null)
const policyError = ref('')
const enterpriseMemberRows = ref<EnterpriseMemberUsageRow[]>([])
const memberUsageError = ref('')
const enterpriseCustomAssignedPointMicros = ref(0)
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
  Number(authStore.manageableEnterpriseTenantId || 0),
)
const activeTenantRole = computed(() =>
  String(
    authStore.memberships.find((membership) =>
      Number(membership.tenant_id) === activeTenantId.value
      && membership.space_type === 'organization')?.role || '',
  ).toLowerCase(),
)
const canManageEnterprisePolicy = computed(() =>
  activeTenantRole.value === 'admin' || activeTenantRole.value === 'owner',
)

const toFiniteNumber = (value: unknown, fallback = 0) => {
  const normalized = Number(value)
  return Number.isFinite(normalized) ? normalized : fallback
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

const formatPeriod = (billing: BillingOverview | null) => {
  if (!billing?.subscription?.current_period_start || !billing.subscription.current_period_end) {
    return t('tenant.subscriptionUsage.noFixedPeriod')
  }
  const start = new Date(billing.subscription.current_period_start)
  const end = new Date(billing.subscription.current_period_end)
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return t('tenant.subscriptionUsage.noFixedPeriod')
  }
  const formatter = new Intl.DateTimeFormat(locale.value || 'zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  return `${formatter.format(start)} - ${formatter.format(end)}`
}

const enterpriseCreditPool = computed(() => {
  const credits = enterpriseOverview.value?.credits
  const periodTotalMicros = Math.max(0, toFiniteNumber(credits?.period_point_micros, 0))
  const periodUsedMicros = Math.max(0, toFiniteNumber(credits?.period_used_point_micros, 0))
  const periodRemainingMicros = Math.max(
    0,
    toFiniteNumber(credits?.period_remaining_point_micros, periodTotalMicros - periodUsedMicros),
  )
  const balanceAvailableMicros = Math.max(0, toFiniteNumber(credits?.balance_point_micros, 0))
  const balanceTotalMicros = Math.max(
    balanceAvailableMicros,
    toFiniteNumber(credits?.balance_allocated_point_micros, balanceAvailableMicros),
  )
  const balanceUsedMicros = Math.max(0, balanceTotalMicros - balanceAvailableMicros)

  return {
    periodText: formatPeriod(enterpriseOverview.value),
    balanceAvailableText: formatCredits(balanceAvailableMicros),
    balanceTotalText: formatCredits(balanceTotalMicros),
    balanceUsedText: formatCredits(balanceUsedMicros),
    periodTotalText: formatCredits(periodTotalMicros),
    periodUsedText: formatCredits(periodUsedMicros),
    periodRemainingText: formatCredits(periodRemainingMicros),
    periodProgress: periodTotalMicros > 0
      ? Math.min(100, Math.round((periodUsedMicros / periodTotalMicros) * 10000) / 100)
      : 0,
  }
})

const memberCustomAssignedLimitMicros = (allocation: { monthly_limit_point_micros?: number; allocated_period_point_micros?: number; allocated_balance_point_micros?: number } | undefined) => {
  if (!allocation) return 0
  const monthlyLimit = toFiniteNumber(allocation.monthly_limit_point_micros, 0)
  if (monthlyLimit > 0) return Math.max(0, monthlyLimit)
  return Math.max(
    0,
    toFiniteNumber(allocation.allocated_period_point_micros, 0)
      + toFiniteNumber(allocation.allocated_balance_point_micros, 0),
  )
}

const enterpriseAssignableCreditMicros = computed(() => {
  const credits = enterpriseOverview.value?.credits
  const periodTotalMicros = Math.max(0, toFiniteNumber(credits?.period_point_micros, 0))
  const balanceAvailableMicros = Math.max(0, toFiniteNumber(credits?.balance_point_micros, 0))
  const balanceTotalMicros = Math.max(
    balanceAvailableMicros,
    toFiniteNumber(credits?.balance_allocated_point_micros, balanceAvailableMicros),
  )
  return periodTotalMicros + balanceTotalMicros
})

const inheritedMemberLimitMicros = computed(() => {
  const defaultLimitMicros = Math.max(
    0,
    enterprisePolicy.value?.default_member_monthly_limit_point_micros || 0,
  )
  const assignedMicros = Math.max(0, enterpriseCustomAssignedPointMicros.value)
  if (assignedMicros <= 0) return defaultLimitMicros
  const remainingMicros = Math.max(0, enterpriseAssignableCreditMicros.value - assignedMicros)
  return Math.min(defaultLimitMicros, remainingMicros)
})

const formatBytes = (bytes: number) => {
  const value = Math.max(0, toFiniteNumber(bytes, 0))
  if (value < 1024) return `${Math.round(value)} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let scaled = value
  let unitIndex = -1
  while (scaled >= 1024 && unitIndex < units.length - 1) {
    scaled /= 1024
    unitIndex += 1
  }
  const maximumFractionDigits = scaled >= 100 ? 0 : scaled >= 10 ? 1 : 2
  return `${new Intl.NumberFormat(locale.value || 'zh-CN', {
    maximumFractionDigits,
  }).format(scaled)} ${units[unitIndex]}`
}

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
    const [allocationResponse, storageResponse, members] = await Promise.all([
      getMemberCreditAllocations(activeTenantId.value),
      getMemberStorageUsage(activeTenantId.value),
      fetchAllTenantMembers(activeTenantId.value),
    ])
    if (!allocationResponse.success) {
      throw new Error(
        allocationResponse.message || t('tenant.subscriptionUsage.memberUsageLoadFailed'),
      )
    }
    if (!storageResponse.success) {
      throw new Error(
        storageResponse.message || t('tenant.subscriptionUsage.memberUsageLoadFailed'),
      )
    }

    const allocationByUserID = new Map(
      (allocationResponse.data || []).map((allocation) => [allocation.user_id, allocation]),
    )
    enterpriseCustomAssignedPointMicros.value = (allocationResponse.data || [])
      .filter((allocation) => allocation.limit_mode === 'custom')
      .reduce((total, allocation) => total + memberCustomAssignedLimitMicros(allocation), 0)
    const storageByUserID = new Map(
      (storageResponse.data || []).map((usage) => [usage.actor_user_id, usage]),
    )

    enterpriseMemberRows.value = members
      .filter((member) => member.status === 'active')
      .map((member) => {
        const allocation = allocationByUserID.get(member.user_id)
        const mode = allocation?.limit_mode || 'inherit'
        const defaultOveragePolicy = enterprisePolicy.value?.member_overage_policy === 'use_enterprise_balance'
          ? 'use_enterprise_balance'
          : 'block'
        const effectiveOveragePolicy = allocation?.effective_overage_policy || defaultOveragePolicy
        const effectiveLimitMicros = mode === 'custom'
          ? Math.max(
            0,
            allocation?.effective_monthly_limit_point_micros
              || allocation?.monthly_limit_point_micros
              || 0,
          )
          : mode === 'unlimited'
            ? 0
            : Math.max(
              0,
              allocation?.effective_monthly_limit_point_micros || inheritedMemberLimitMicros.value,
            )
        const usedMicros = Math.max(0, allocation?.used_point_micros || 0)
        const storageUsage = storageByUserID.get(member.user_id)

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
          storageUsedText: formatBytes(storageUsage?.used_bytes || 0),
          storageTransactionText: storageUsage?.transaction_count
            ? t('tenant.subscriptionUsage.memberStorageTransactions', {
              count: storageUsage.transaction_count,
            })
            : t('tenant.subscriptionUsage.memberStorageNoTransactions'),
          strategy: memberStrategyLabel(mode),
          overage: effectiveOveragePolicy === 'use_enterprise_balance'
            ? t('tenant.subscriptionUsage.overageUseBalance')
            : t('tenant.subscriptionUsage.overageBlock'),
          lastBilling: allocation?.last_billing_at
            ? formatUsageTime(allocation.last_billing_at)
            : '-',
          usedMicros,
        }
      })
      .sort((left, right) =>
        right.usedMicros - left.usedMicros || left.name.localeCompare(right.name))
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

const loadEnterpriseUsage = async () => {
  const sequence = ++loadSequence
  loading.value = true
  error.value = ''
  policyError.value = ''
  enterpriseOverview.value = null
  enterprisePolicy.value = null
  enterpriseMemberRows.value = []
  memberUsageError.value = ''
  enterpriseCustomAssignedPointMicros.value = 0

  if (!activeTenantId.value || !canManageEnterprisePolicy.value) {
    error.value = t('tenant.subscriptionUsage.enterprisePermissionRequired')
    loading.value = false
    return
  }

  try {
    const [overviewResponse, policyResponse] = await Promise.all([
      getBillingOverview(activeTenantId.value),
      getTenantBillingPolicy(activeTenantId.value),
    ])
    if (sequence !== loadSequence) return
    if (!overviewResponse.success || !overviewResponse.data) {
      throw new Error(overviewResponse.message || t('tenant.subscriptionUsage.enterpriseCreditPoolLoadFailed'))
    }
    if (!policyResponse.success || !policyResponse.data) {
      throw new Error(policyResponse.message || t('tenant.subscriptionUsage.policyLoadFailed'))
    }
    enterpriseOverview.value = overviewResponse.data
    applyEnterprisePolicy(policyResponse.data)
    await loadEnterpriseMemberUsage()
  } catch (err: any) {
    if (sequence !== loadSequence) return
    error.value = err?.message || t('tenant.subscriptionUsage.policyLoadFailed')
  } finally {
    if (sequence === loadSequence) loading.value = false
  }
}

watch(
  [activeTenantId, canManageEnterprisePolicy],
  () => {
    void loadEnterpriseUsage()
  },
  { immediate: true },
)
</script>

<style lang="less" scoped>
.enterprise-usage {
  width: 100%;
}

.loading-state {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 24px 0;
  color: var(--td-text-color-secondary);
}

.error-state {
  padding-top: 8px;
}

.enterprise-usage__content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.enterprise-credit-pool,
.enterprise-policy,
.enterprise-member-usage {
  padding: 24px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.enterprise-credit-pool__heading,
.enterprise-policy__heading,
.enterprise-policy__field {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.enterprise-credit-pool__heading,
.enterprise-policy__heading {
  padding-bottom: 18px;
  border-bottom: 1px solid var(--td-component-stroke);

  h2,
  p {
    margin: 0;
  }

  h2 {
    color: var(--td-text-color-primary);
    font-size: 18px;
  }

  p {
    margin-top: 5px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
  }
}

.enterprise-credit-pool__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  padding-top: 20px;
}

.enterprise-credit-pool__metric {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 0 20px;

  &:first-child {
    padding-left: 0;
  }

  & + & {
    border-left: 1px solid var(--td-component-stroke);
  }

  span,
  small {
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 20px;
    line-height: 1.35;
  }

  :deep(.t-progress) {
    margin-top: 4px;
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

.enterprise-member-usage__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;

  h2,
  p {
    margin: 0;
  }

  h2 {
    color: var(--td-text-color-primary);
    font-size: 18px;
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

@media (max-width: 720px) {
  .enterprise-credit-pool__heading,
  .enterprise-policy__heading,
  .enterprise-policy__field,
  .enterprise-member-usage__heading {
    flex-direction: column;
  }

  .enterprise-credit-pool__grid {
    grid-template-columns: 1fr;
    gap: 16px;
  }

  .enterprise-credit-pool__metric {
    padding: 0;

    & + & {
      padding-top: 16px;
      border-top: 1px solid var(--td-component-stroke);
      border-left: 0;
    }
  }
}
</style>
