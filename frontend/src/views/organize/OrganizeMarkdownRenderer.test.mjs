import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const utilitySource = readFileSync(new URL('./organizeMarkdown.ts', import.meta.url), 'utf8')
const componentSource = readFileSync(
  new URL('./components/OrganizeMarkdownRenderer.vue', import.meta.url),
  'utf8',
)
const outputDetailSource = readFileSync(new URL('./OrganizeOutputDetail.vue', import.meta.url), 'utf8')
const sproutReportSource = readFileSync(new URL('./sproutReport.ts', import.meta.url), 'utf8')

test('organize Markdown rendering uses one sanitized generic entry point', () => {
  assert.ok(utilitySource.includes('export const renderOrganizeMarkdown'))
  assert.ok(utilitySource.includes('safeMarkdownToHTML'))
  assert.ok(utilitySource.includes('sanitizeMarkdownHTML'))
  assert.ok(utilitySource.includes('sanitizeHTML(source)'))
  assert.ok(componentSource.includes('renderOrganizeMarkdown'))
  assert.ok(componentSource.includes("type OrganizeMarkdownProfile = 'report' | 'checklist'"))
  assert.ok(componentSource.includes('v-html="renderedHtml"'))
})

test('output detail uses the generic renderer and preserves sprout compatibility', () => {
  assert.ok(outputDetailSource.includes('<OrganizeMarkdownRenderer :content="output.content" profile="report" />'))
  assert.ok(!outputDetailSource.includes('v-html="outputHtml"'))
  assert.ok(!outputDetailSource.includes('renderSproutReportHtml'))
  assert.ok(sproutReportSource.includes('renderOrganizeMarkdown(value, { normalize: normalizeSproutMarkdownSource })'))
})
