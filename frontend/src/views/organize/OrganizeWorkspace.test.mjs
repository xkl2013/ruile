import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./OrganizeWorkspace.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../api/organize/index.ts', import.meta.url), 'utf8')
const categorySource = readFileSync(new URL('./discoverCategories.ts', import.meta.url), 'utf8')

test('memory add button opens create/import menu', () => {
  assert.ok(source.includes('memoryCreateOptions'))
  assert.ok(source.includes('新建笔记'))
  assert.ok(source.includes('导入文件'))
  assert.ok(source.includes('memoryImportInputRef'))
  assert.ok(source.includes('uploadOrganizeMemory'))
  assert.ok(apiSource.includes('timeout: 300000'))
  assert.ok(source.includes('handleMemoryCreateAction'))
  assert.ok(source.includes('handleMemoryImportFileChange'))
  assert.ok(source.includes("openDocumentEditor('memory', imported.id, imported)"))
})

test('sprout report cards show linked memory references', () => {
  assert.ok(apiSource.includes('memory_refs?: OrganizeMemoryReference[]'))
  assert.ok(source.includes('sproutReportReferenceLabels(report)'))
  assert.ok(source.includes('sproutReportSourceLabels(report)'))
  assert.ok(source.includes('memoryRefs: item.memory_refs || []'))
  assert.ok(source.includes('@创建了'))
  assert.ok(!source.includes('<span>@记忆</span>'))
})

test('discover uses the fixed first-version kindergarten columns', () => {
  assert.ok(categorySource.includes("{ key: 'admissions_growth', label: '招生增长' }"))
  assert.ok(categorySource.includes("{ key: 'nutrition_food_education', label: '儿童营养与食育' }"))
  assert.ok(source.includes('DISCOVER_CATEGORIES'))
  assert.ok(source.includes('categoryLabel: discoverCategoryLabel'))
  assert.ok(source.includes("discover_category: normalizeDiscoverCategory(item.categoryLabel)"))
  assert.ok(!source.includes('tag:'))
})
