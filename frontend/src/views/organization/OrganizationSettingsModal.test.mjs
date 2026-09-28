import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'

const sourcePath = path.resolve(new URL('.', import.meta.url).pathname, 'OrganizationSettingsModal.vue')
const source = fs.readFileSync(sourcePath, 'utf8')

test('shows shared-space deletion only for the owner and uses the organization tenant context', () => {
  assert.ok(source.includes('v-if="canDeleteOrganization"'))
  assert.ok(source.includes('const canDeleteOrganization = computed'))
  assert.ok(source.includes('authStore.isSystemAdmin || hasTenantAdmin.value'))
  assert.ok(source.includes('orgStore.remove(props.orgId, requestOptions.value)'))
})

test('allows organization admins to update a knowledge-base share permission', () => {
  assert.ok(source.includes('v-if="isAdmin"'))
  assert.ok(source.includes('class="share-permission-select"'))
  assert.ok(source.includes('handleSharePermissionChange(share, value)'))
  assert.ok(source.includes('updateSharePermission('))
  assert.ok(source.includes('requestOptions.value'))
})
