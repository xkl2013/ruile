import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./KnowledgeBaseList.vue', import.meta.url), 'utf8')

test('knowledge base card menu exposes delete for authorized resources', () => {
  assert.ok(source.includes('canDeleteKnowledgeBase(kb)'))
  assert.ok(source.includes('confirmDeleteKnowledgeBase(kb)'))
  assert.ok(source.includes('deleteKnowledgeBase(kb.id)'))
  assert.ok(source.includes("authStore.hasRole('admin')"))
  assert.ok(source.includes("authStore.hasRoleInTenant(targetTenantId, 'admin')"))
})

test('knowledge base cards do not render a duplicate title icon', () => {
  assert.ok(!source.includes('kb-card-icon'))
  assert.ok(!source.includes('<KnowledgeBaseIcon'))
})
