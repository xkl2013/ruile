import assert from 'node:assert/strict'
import test from 'node:test'

import { highestSharedKnowledgeBase } from './sharedKnowledgeBasePermission.ts'

test('selects editor when the same knowledge base is viewer in one space and editor in another', () => {
  const records = [
    {
      organization_id: 'space-viewer',
      permission: 'viewer',
      knowledge_base: { id: 'kb-1' },
    },
    {
      organization_id: 'space-editor',
      permission: 'editor',
      knowledge_base: { id: 'kb-1' },
    },
  ]

  const selected = highestSharedKnowledgeBase(records, 'kb-1')

  assert.equal(selected?.organization_id, 'space-editor')
  assert.equal(selected?.permission, 'editor')
})

test('selection is independent of shared-space response order', () => {
  const records = [
    { permission: 'editor', knowledge_base: { id: 'kb-1' } },
    { permission: 'viewer', knowledge_base: { id: 'kb-1' } },
  ]

  assert.equal(highestSharedKnowledgeBase(records, 'kb-1')?.permission, 'editor')
})
