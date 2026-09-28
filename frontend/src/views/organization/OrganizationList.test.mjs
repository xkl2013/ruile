import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./OrganizationList.vue', import.meta.url), 'utf8')

test('team-space owner delete uses the organization owner tenant', () => {
  assert.ok(source.includes('v-if="canDeleteOrganization(org)"'))
  assert.ok(source.includes('org.owner_tenant_id'))
  assert.ok(source.includes('organizationTenantId(deletingOrg.value)'))
  assert.ok(source.includes("authStore.hasRoleInTenant(tenantId, 'admin')"))
})

test('team-space cards use content-width responsive layout and stable spacing', () => {
  assert.ok(source.includes('grid-template-columns: repeat(auto-fill, minmax(min(100%, 260px), 1fr));'))
  assert.ok(source.includes('padding: 16px;'))
  assert.ok(source.includes('height: 160px;'))
  assert.ok(source.includes('padding-right: 40px;'))
})
