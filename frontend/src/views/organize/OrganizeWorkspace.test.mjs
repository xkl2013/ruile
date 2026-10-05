import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./OrganizeWorkspace.vue', import.meta.url), 'utf8')
const courseSource = readFileSync(new URL('./OrganizeCourseDiscover.vue', import.meta.url), 'utf8')
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

test('discover output menu exposes deletion and uses the output delete API', () => {
  assert.ok(apiSource.includes('deleteOrganizeOutput'))
  assert.ok(apiSource.includes('/api/v1/organize/outputs/${encodeURIComponent(id)}'))
  assert.ok(source.includes("value: 'delete'"))
  assert.ok(source.includes('deleteOutputItem'))
  assert.ok(source.includes('源文件也会一并删除'))
})

test('memory actions create organize jobs instead of legacy sprout reports', () => {
  assert.ok(source.includes("type MemoryMenuAction = 'edit' | 'organize' | 'delete'"))
  assert.ok(source.includes("handleMemoryMenuAction(item, 'organize')"))
  assert.ok(source.includes('createOrganizeFromMemory'))
  assert.ok(source.includes('listOrganizeConfigs'))
  assert.ok(source.includes('createOrganizeJob'))
  assert.ok(!source.includes('createOrganizeSproutReportFromMemory'))
  assert.ok(!source.includes('memorySproutActionLabel'))
  assert.ok(!source.includes('发芽'))
})

test('discover uses the fixed first-version kindergarten columns', () => {
  assert.ok(categorySource.includes("{ key: 'admissions_growth', label: '招生增长' }"))
  assert.ok(categorySource.includes("{ key: 'nutrition_food_education', label: '儿童营养与食育' }"))
  assert.ok(source.includes('DISCOVER_CATEGORIES'))
  assert.ok(source.includes('categoryLabel: discoverCategoryLabel'))
  assert.ok(source.includes("discover_category: normalizeDiscoverCategory(item.categoryLabel)"))
  assert.ok(!source.includes('tag:'))
})

test('discover fills an empty featured area with published courses', () => {
  assert.ok(source.includes('showFeaturedCourseFallback'))
  assert.ok(source.includes('featuredCourseDiscoverRef'))
  assert.ok(source.includes('variant="featured"'))
  assert.ok(source.includes(':limit="FEATURED_OUTPUT_SIZE"'))
  assert.ok(courseSource.includes('暂无可推荐内容或课程'))
  assert.ok(!source.includes('暂无精选'))
})
