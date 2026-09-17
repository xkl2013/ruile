import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./KnowledgeBase.vue', import.meta.url), 'utf8')

test('knowledge base directory move-up action uses a supported icon', () => {
  assert.ok(source.includes('moveDirectoryUp'))
  assert.ok(source.includes("name=\"chevron-up\" size=\"14px\""))
  assert.ok(!source.includes("name=\"arrow-up\" size=\"14px\""))
})

test('knowledge base directory tree exposes manual directory delete action', () => {
  assert.ok(source.includes('deleteDirectory(directory)'))
  assert.ok(source.includes("name=\"delete\" size=\"14px\""))
  assert.ok(source.includes('directory-tree-action--danger'))
  assert.ok(source.includes('updateKnowledgeBaseDirectoryConfig'))
})

test('knowledge base write controls use the authoritative access projection', () => {
  assert.ok(source.includes('canWriteKnowledgeBase'))
  assert.ok(source.includes('orgStore.getSharedKnowledgeBase(kbId.value)'))
  assert.ok(source.includes('sharedPermission: currentSharedKb.value?.permission'))
  assert.ok(source.includes('v-if="canEdit" class="doc-filter-actions"'))
  assert.ok(source.includes(':can-edit="canEdit"'))
  assert.ok(source.includes(':canEditKB="canEdit"'))
  assert.ok(source.includes('orgStore.fetchSharedKnowledgeBases({ force: true })'))
  assert.ok(source.includes('knowledgeBasePermissionLoaded'))
  assert.ok(source.includes('const canEditKnowledgeBaseIdentity = computed(() => canEdit.value)'))
})
