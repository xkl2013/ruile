import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./KnowledgeBase.vue', import.meta.url), 'utf8')
const actionMenuSource = readFileSync(new URL('./components/DocumentActionMenu.vue', import.meta.url), 'utf8')
const listViewSource = readFileSync(new URL('./components/DocumentListView.vue', import.meta.url), 'utf8')
const cardViewSource = readFileSync(new URL('./components/DocumentCardView.vue', import.meta.url), 'utf8')
const knowledgeApiSource = readFileSync(new URL('../../api/knowledge-base/index.ts', import.meta.url), 'utf8')

test('knowledge base directory move-up action is distinct from expand and collapse', () => {
  assert.ok(source.includes('moveDirectoryUp'))
  assert.ok(source.includes("name=\"arrow-up\" size=\"14px\""))
  assert.ok(!source.includes("name=\"chevron-up\" size=\"14px\""))
})

test('knowledge base directory tree exposes manual directory delete action', () => {
  assert.ok(source.includes('deleteDirectory(directory)'))
  assert.ok(source.includes("name=\"delete\" size=\"14px\""))
  assert.ok(source.includes('directory-tree-action--danger'))
  assert.ok(source.includes('updateKnowledgeBaseDirectoryConfig'))
})

test('directory hierarchy uses compact indentation and a leading expansion control', () => {
  assert.ok(source.includes('const DIRECTORY_TREE_INDENT_STEP = 6'))
  assert.ok(source.includes('const DIRECTORY_TREE_COMPACT_INDENT_STEP = 4'))
  assert.ok(source.includes('const level = Math.max(0, depth);'))
  assert.ok(source.includes('min-height: 28px;'))
  assert.ok(source.includes('padding: 0 4px 0 calc(4px + var(--directory-indent, 0px));'))
  assert.ok(source.includes('class="directory-tree-leading-toggle"'))
  assert.ok(source.includes('directory-tree-leading-toggle--placeholder'))
  assert.ok(source.includes('@click.stop="toggleDirectoryCollapsed(DIRECTORY_ROOT_PATH)"'))
  assert.ok(source.includes('@click.stop="toggleDirectoryCollapsed(directory.path)"'))
  assert.ok(source.includes("@click=\"selectDirectory(DIRECTORY_ROOT_PATH)\""))
  assert.ok(source.includes('@click="selectDirectory(directory.path)"'))
  assert.ok(source.includes(":name=\"directory.collapsed ? 'chevron-right' : 'chevron-down'\""))
  assert.ok(source.includes("'has-hover-actions': canEditKnowledgeBaseDirectories"))
  assert.ok(!source.includes('directory-tree-action--toggle'))
  assert.ok(!source.includes('class="directory-tree-toggle"'))
  assert.ok(!source.includes('class="directory-tree-icon"'))
})

test('directory tree defaults to showing only first-level directories', () => {
  assert.ok(source.includes('const directoryExpansionCustomized = ref(false)'))
  assert.ok(source.includes('const getDefaultCollapsedDirectoryPaths = (directories: DirectoryNode[]) =>'))
  assert.ok(source.includes('collapsedDirectoryPaths.value = getDefaultCollapsedDirectoryPaths(directories)'))
  assert.ok(source.includes('directoryExpansionCustomized.value = true'))
  assert.ok(source.includes('directoryExpansionCustomized.value = false'))
})

test('knowledge base write controls use the authoritative access projection', () => {
  assert.ok(source.includes('canWriteKnowledgeBase'))
  assert.ok(source.includes('orgStore.getSharedKnowledgeBase(kbId.value)'))
  assert.ok(source.includes('sharedPermission: currentSharedKb.value?.permission'))
  assert.ok(source.includes('<KbUploadSourceDropdown v-if="canEdit"'))
  assert.ok(source.includes(':can-edit="canEdit"'))
  assert.ok(source.includes(':canEditKB="canEdit"'))
  assert.ok(source.includes('orgStore.fetchSharedKnowledgeBases({ force: true })'))
  assert.ok(source.includes('knowledgeBasePermissionLoaded'))
  assert.ok(source.includes('const canEditKnowledgeBaseIdentity = computed(() => canEdit.value)'))
})

test('knowledge base exposes file and vector modes through URL state', () => {
  assert.ok(source.includes("type KnowledgeViewMode = 'file' | 'vector'"))
  assert.ok(source.includes("route.query.mode"))
  assert.ok(source.includes("setKnowledgeViewMode('file')"))
  assert.ok(source.includes("setKnowledgeViewMode('vector')"))
  assert.ok(source.includes("router.replace({ query: { ...route.query, mode } })"))
})

test('file mode uses the prototype-style feed and keeps the management view in vector mode', () => {
  assert.ok(source.includes("import DocumentFileFeed from './components/DocumentFileFeed.vue'"))
  assert.ok(source.includes("v-if=\"knowledgeViewMode === 'file'\" class=\"file-mode-content\""))
  assert.ok(source.includes("v-if=\"knowledgeViewMode === 'vector'\" class=\"doc-filter-bar\""))
  assert.ok(source.includes('@filter-tag="handleFileModeTagFilter"'))
  assert.ok(source.includes('openVectorIssueView'))
})

test('mode switch sits on the knowledge title row and vector actions precede it', () => {
  const headerStart = source.indexOf('<div class="document-header">')
  const headerEnd = source.indexOf('<!-- Wiki Browser / Graph', headerStart)
  const header = source.slice(headerStart, headerEnd)
  const fileToolbarStart = source.indexOf('<div class="file-mode-toolbar">')
  const fileToolbarEnd = source.indexOf('\n                </div>', fileToolbarStart)
  const fileToolbar = source.slice(fileToolbarStart, fileToolbarEnd)

  assert.ok(header.includes('class="document-header-controls"'))
  assert.ok(header.includes('v-if="canEdit && knowledgeViewMode === \'vector\'"'))
  assert.ok(header.indexOf('class="document-header-search"') < header.indexOf('<KbUploadSourceDropdown'))
  assert.ok(header.indexOf('<KbUploadSourceDropdown') < header.indexOf('class="knowledge-view-segment"'))
  assert.ok(fileToolbar.indexOf('knowledgeBase.documentCount') < fileToolbar.indexOf('<KbUploadSourceDropdown'))
  assert.ok(fileToolbar.includes('class="file-mode-toolbar__actions"'))
  assert.ok(source.includes('.file-mode-toolbar__actions {\n  margin-left: auto;'))
  assert.ok(source.includes('.document-header-controls {'))
  assert.ok(!source.includes('class="knowledge-view-modebar"'))
  assert.ok(!header.includes("knowledgeBase.createFolder"))
  assert.equal(source.match(/data-guide="kb-detail-add-doc"/g)?.length, 2)
})

test('file mode search follows the active directory while vector search stays in the header', () => {
  const fileToolbarStart = source.indexOf('<div class="file-mode-toolbar">')
  const fileToolbarEnd = source.indexOf('\n                </div>', fileToolbarStart)
  const fileToolbar = source.slice(fileToolbarStart, fileToolbarEnd)

  assert.ok(source.includes("knowledgeViewMode === 'vector'"))
  assert.ok(fileToolbar.indexOf('knowledgeBase.documentCount') < fileToolbar.indexOf('file-mode-toolbar__search'))
  assert.ok(fileToolbar.indexOf('file-mode-toolbar__search') < fileToolbar.indexOf('file-mode-toolbar__actions'))
  assert.ok(source.includes('width: min(240px, 28vw);'))
  assert.ok(source.includes('height: 30px;'))
  assert.ok(source.includes('border-radius: 8px;'))
  assert.ok(source.includes('keyword: docSearchKeyword.value ? docSearchKeyword.value.trim() : undefined'))
  assert.ok(source.includes('directory_path: activeDirectoryPath.value || undefined'))
})

test('file mode opens documents in preview-only while vector mode keeps full details', () => {
  assert.ok(source.includes('const docPreviewOnly = ref(false)'))
  assert.ok(source.includes("const previewOnly = knowledgeViewMode.value === 'file'"))
  assert.ok(source.includes('openCardDetails(item, previewOnly)'))
  assert.ok(source.includes(':preview-only="docPreviewOnly"'))
  assert.ok(source.includes('if (!previewOnly && canEdit.value && isManualDraftKnowledge(item))'))
})

test('failed or cancelled documents can be marked as unparsed from both views', () => {
  assert.ok(actionMenuSource.includes("['failed', 'cancelled'].includes"))
  assert.ok(actionMenuSource.includes("emit('ignore-parse')"))
  assert.ok(actionMenuSource.includes("knowledgeBase.ignoreParseConfirmBody"))
  assert.ok(listViewSource.includes("@ignore-parse=\"handleAction('ignore-parse', item)\""))
  assert.ok(cardViewSource.includes("@ignore-parse=\"handleAction('ignore-parse', item)\""))
  assert.ok(source.includes('ignoreKnowledgeParse(item.id)'))
  assert.ok(source.includes("current.parse_status = updated.parse_status || 'unparsed'"))
  assert.ok(knowledgeApiSource.includes('/ignore-parse'))
})
