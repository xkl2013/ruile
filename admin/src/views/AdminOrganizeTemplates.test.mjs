import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./AdminOrganizeTemplates.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../api/organize.ts', import.meta.url), 'utf8')

test('organize template groups are loaded from the admin API and remain creatable', () => {
  assert.ok(source.includes('listAdminOrganizeTemplateScenes'))
  assert.ok(source.includes(':options="sceneOptions"'))
  assert.ok(source.includes('creatable'))
  assert.ok(source.includes('选择已有分组或输入新分组'))
  assert.ok(apiSource.includes('/api/v1/system/admin/organize/templates/scenes'))
  assert.ok(!source.includes('例如 教师成长、招生增长'))
})

test('template trial renders the finished markdown output instead of internal prompt placeholders', () => {
  assert.ok(source.includes("import OrganizeMarkdownRenderer from '@/views/organize/components/OrganizeMarkdownRenderer.vue'"))
  assert.ok(source.includes(':content="previewData.preview_markdown"'))
  assert.ok(source.includes('示例成品'))
  assert.ok(apiSource.includes('preview_markdown: string'))
  assert.ok(!source.includes("<pre>{{ previewData.prompt || '暂无可渲染指令' }}</pre>"))
})
