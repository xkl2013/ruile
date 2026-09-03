import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./KnowledgeBaseMenu.vue', import.meta.url), 'utf8')

test('knowledge base menu keeps admin-only configuration out of the main app', () => {
  assert.ok(source.includes('goToKnowledgeBaseList'))
  assert.ok(source.includes('openKnowledgeBase'))
  assert.ok(source.includes('kb-menu-item-dot'))

  assert.ok(!source.includes('reorderKnowledgeBases'))
  assert.ok(!source.includes('moveKnowledgeBaseUp'))
  assert.ok(!source.includes('kb-menu-item-action'))
  assert.ok(!source.includes('kb-menu-action-btn--create'))
  assert.ok(!source.includes('navigateToAdmin'))
})
