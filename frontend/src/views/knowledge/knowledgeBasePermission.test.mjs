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

test('public subscription knowledge bases are always read-only', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'public_subscription',
      my_permission: 'viewer',
    },
    isOwner: true,
    isTenantAdmin: true,
    isSystemAdmin: true,
  }), false)
})

test('legacy subscription knowledge bases are also read-only', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'subscription',
      my_permission: 'viewer',
    },
    isOwner: true,
    isTenantAdmin: true,
    isSystemAdmin: true,
  }), false)
})

test('an explicit viewer projection remains read-only', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'created',
      my_permission: 'viewer',
    },
    isOwner: true,
    isTenantAdmin: true,
  }), false)
})

test('explicit access permission takes precedence over the active tenant role', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'tenant_admin',
      my_permission: 'viewer',
    },
    isTenantAdmin: true,
  }), false)
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {
      access_source: 'shared_space',
      my_permission: 'editor',
    },
    isTenantAdmin: false,
  }), true)
})

test('missing access metadata does not expose edit controls to tenant admins', () => {
  assert.equal(canWriteKnowledgeBase({
    kbInfo: {},
    isTenantAdmin: true,
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
