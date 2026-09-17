import assert from 'node:assert/strict'
import test from 'node:test'
import { resolveKnowledgeBaseScope } from './knowledgeBaseScope.ts'

test('resolves personal, enterprise, and subscribed knowledge-base scopes', () => {
  assert.equal(resolveKnowledgeBaseScope({ access_source: 'created', owner_type: 'personal' }), 'personal')
  assert.equal(resolveKnowledgeBaseScope({ access_source: 'created', owner_type: 'organization' }), 'enterprise')
  assert.equal(resolveKnowledgeBaseScope({ access_source: 'shared_space' }), 'enterprise')
  assert.equal(resolveKnowledgeBaseScope({ isMine: false }), 'enterprise')
  assert.equal(resolveKnowledgeBaseScope({ list_category: 'subscribed', owner_type: 'organization' }), 'subscribed')
  assert.equal(resolveKnowledgeBaseScope({}, 'subscribed'), 'subscribed')
})
