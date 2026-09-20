import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const component = readFileSync(new URL('./TenantInfo.vue', import.meta.url), 'utf8')
const enterpriseUsage = readFileSync(new URL('./EnterpriseUsagePolicy.vue', import.meta.url), 'utf8')
const settings = readFileSync(new URL('./Settings.vue', import.meta.url), 'utf8')

test('subscription usage reads the current workspace billing overview', () => {
  assert.ok(component.includes('getBillingOverview'))
  assert.ok(component.includes('activeTenantId'))
  assert.ok(component.includes('billing.plan.name'))
  assert.ok(component.includes('period_point_micros'))
  assert.ok(component.includes('period_used_point_micros'))
  assert.ok(component.includes('balance_point_micros'))
  assert.ok(component.includes('billing.storage'))
  assert.ok(component.includes('current_period_start'))

  assert.ok(!component.includes('memberUsageTitle'))
  assert.ok(!component.includes('getMemberCreditAllocations'))
  assert.ok(!component.includes('getTenantBillingPolicy'))
  assert.ok(!component.includes('updateTenantBillingPolicy'))
  assert.ok(!component.includes('enterprise_credits'))
  assert.ok(!component.includes('getTenantById'))
  assert.ok(!component.includes('updateTenantApi'))
  assert.ok(!component.includes('deleteTenantApi'))
  assert.ok(!component.includes('leaveTenant'))
})

test('billing overview uses the current authenticated workspace', () => {
  const billingApi = readFileSync(new URL('../../api/billing/index.ts', import.meta.url), 'utf8')
  assert.ok(billingApi.includes("'/api/v1/billing/overview'"))
})

test('enterprise usage policy lives in the enterprise settings module', () => {
  assert.ok(enterpriseUsage.includes('enterprisePolicyTitle'))
  assert.ok(enterpriseUsage.includes('memberUsageTitle'))
  assert.ok(enterpriseUsage.includes('getMemberCreditAllocations'))
  assert.ok(enterpriseUsage.includes('getTenantBillingPolicy'))
  assert.ok(enterpriseUsage.includes('updateTenantBillingPolicy'))
  assert.ok(enterpriseUsage.includes('manageableEnterpriseTenantId'))
  assert.ok(enterpriseUsage.includes("activeTenantRole.value === 'admin'"))
  assert.ok(enterpriseUsage.includes("activeTenantRole.value === 'owner'"))

  assert.ok(settings.includes("key: 'enterpriseUsage'"))
  assert.ok(settings.includes('canManageEnterpriseSettings.value'))
  assert.ok(settings.includes('<EnterpriseUsagePolicy />'))
})
