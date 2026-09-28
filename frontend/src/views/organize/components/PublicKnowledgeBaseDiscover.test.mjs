import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./PublicKnowledgeBaseDiscover.vue', import.meta.url), 'utf8')

test('public knowledge-base cards follow the reference hierarchy and spacing', () => {
  assert.ok(source.includes('grid-template-columns: minmax(0, 1fr);'))
  assert.ok(source.includes('min-height: 145px;'))
  assert.ok(source.includes('font-size: 16px;'))
  assert.ok(source.includes('font-weight: 600;'))
  assert.ok(source.includes('font-size: 13px;'))
  assert.ok(source.includes('font-size: 12px;'))
})
