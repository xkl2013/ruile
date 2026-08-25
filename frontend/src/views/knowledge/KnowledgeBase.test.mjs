import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./KnowledgeBase.vue', import.meta.url), 'utf8')

test('knowledge base directory move-up action uses a supported icon', () => {
  assert.ok(source.includes('moveDirectoryUp'))
  assert.ok(source.includes("name=\"chevron-up\" size=\"14px\""))
  assert.ok(!source.includes("name=\"arrow-up\" size=\"14px\""))
})
