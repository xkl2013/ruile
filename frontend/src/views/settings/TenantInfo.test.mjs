import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const component = readFileSync(new URL('./TenantInfo.vue', import.meta.url), 'utf8')

test('subscription usage reads the current workspace billing overview', () => {
  assert.ok(component.includes('getBillingOverview'))
  assert.ok(component.includes('activeTenantId'))
  assert.ok(component.includes('billing.plan.name'))
  assert.ok(component.includes('period_point_micros'))
  assert.ok(component.includes('balance_point_micros'))
  assert.ok(component.includes('billing.storage'))
  assert.ok(component.includes('current_period_start'))

  assert.ok(!component.includes('enterprise_credits'))
  assert.ok(!component.includes('getTenantById'))
  assert.ok(!component.includes('updateTenantApi'))
  assert.ok(!component.includes('deleteTenantApi'))
  assert.ok(!component.includes('leaveTenant'))
})

test('billing overview uses the current authenticated workspace', () => {
  const billingApi = readFileSync(new URL('../../api/billing/index.ts', import.meta.url), 'utf8')
  assert.ok(billingApi.includes("get('/api/v1/billing/overview')"))
})
