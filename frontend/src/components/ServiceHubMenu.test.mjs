import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./ServiceHubMenu.vue', import.meta.url), 'utf8')

test('service menu follows the compact organize sidebar style', () => {
  assert.ok(source.includes('class="service-hub-service-icon"'))
  assert.ok(source.includes('background: var(--td-bg-color-secondarycontainer);'))
  assert.ok(source.includes('padding: 0 12px 0 calc(var(--sidebar-inset-x) + 12px);'))

  assert.ok(!source.includes('service-hub-menu-count'))
  assert.ok(!source.includes('service-hub-menu-empty'))
  assert.ok(!source.includes('overflow-y: auto'))
})
