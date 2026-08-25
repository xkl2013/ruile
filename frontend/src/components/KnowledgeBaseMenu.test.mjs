import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./KnowledgeBaseMenu.vue', import.meta.url), 'utf8')

test('knowledge base menu exposes a visible move-up action', () => {
  assert.ok(source.includes('moveKnowledgeBaseUp'))
  assert.ok(source.includes("t-icon :name=\"reorderingKnowledgeBaseId === kb.id ? 'loading' : 'chevron-up'\""))
  assert.ok(source.includes('reorderKnowledgeBases(orderedIds)'))
  assert.ok(source.includes('kb-menu-item-action'))
  assert.ok(source.includes('sortableKnowledgeBaseIds.value.includes(kbId)'))

  const actionBlock = source.match(/\.kb-menu-item-action \{[\s\S]*?\n\}/)?.[0] || ''
  assert.ok(actionBlock)
  assert.ok(!actionBlock.includes('opacity: 0;'))
  assert.ok(!actionBlock.includes('pointer-events: none;'))
})
