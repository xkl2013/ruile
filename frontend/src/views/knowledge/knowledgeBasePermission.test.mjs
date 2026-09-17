import assert from 'node:assert/strict'
import test from 'node:test'

import {
  canWriteKnowledgeBase,
  effectiveKnowledgeBasePermission,
  isSharedKnowledgeBaseAccess,
} from './knowledgeBasePermission.ts'

test('viewer-only shared knowledge base stays read-only for tenant admins', () => {
  const kbInfo = {
    access_source: 'shared_space',
    my_permission: 'viewer',
  }

  assert.equal(isSharedKnowledgeBaseAccess(kbInfo), true)
  assert.equal(effectiveKnowledgeBasePermission(kbInfo), 'viewer')
  assert.equal(canWriteKnowledgeBase({
    kbInfo,
    isTenantAdmin: true,
    isSystemAdmin: false,
  }), false)
})

test('shared editor and admin permissions allow content editing', () => {
  for (const permission of ['editor', 'admin']) {
    assert.equal(canWriteKnowledgeBase({
      kbInfo: {
        access_source: 'shared_space',
        my_permission: permission,
      },
    }), true)
  }
})

test('aggregated detail permission wins over a lower shared-list role', () => {
  assert.equal(effectiveKnowledgeBasePermission({
    access_source: 'shared_space',
    my_permission: 'editor',
  }, 'viewer'), 'editor')
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'shared_space',
      my_permission: 'editor',
    },
    sharedPermission: 'viewer',
    hasSharedRecord: true,
  }), true)
})

test('knowledge base visible only through a shared agent is always read-only', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'shared_agent',
      my_permission: 'admin',
    },
    isOwner: true,
    isTenantAdmin: true,
    isSystemAdmin: true,
  }), false)
})

test('owned and home-tenant admin knowledge bases remain editable', () => {
  const loadedKb = { access_source: 'created' }
  assert.equal(canWriteKnowledgeBase({ kbInfo: loadedKb, isOwner: true }), true)
  assert.equal(canWriteKnowledgeBase({ kbInfo: loadedKb, isTenantAdmin: true }), true)
  assert.equal(canWriteKnowledgeBase({ kbInfo: loadedKb, isSystemAdmin: true }), true)
})

test('permission is read-only until the knowledge base access projection is loaded', () => {
  assert.equal(canWriteKnowledgeBase({
    permissionLoaded: false,
    isTenantAdmin: true,
  }), false)
})
