import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const source = readFileSync(new URL('./PublicKnowledgeBaseDiscover.vue', import.meta.url), 'utf8')

test('public knowledge-base discover follows the reference layout', () => {
  assert.ok(!source.includes('public-kb-header'))
  assert.ok(!source.includes('placeholder="搜索知识库"'))
  assert.ok(source.includes('<h3>精选</h3>'))
  assert.ok(source.includes('class="public-kb-section-title--recommended">推荐</h3>'))
  assert.ok(source.includes('换一换'))
  assert.ok(source.includes('grid-template-columns: repeat(2, minmax(0, 1fr));'))
  assert.ok(source.includes('class="output-card output-card--editable discover-card public-kb-card"'))
  assert.ok(source.includes('class="public-kb-card__image"'))
  assert.ok(source.includes('width: 56px;'))
  assert.ok(source.includes('height: 56px;'))
  assert.ok(source.includes('class="output-summary"'))
  assert.ok(source.includes('class="discover-card-meta"'))
  assert.ok(source.includes('font-size: 12px;'))
  assert.ok(source.includes('font-weight: 400;'))
  assert.ok(source.includes('line-height: 18px;'))
  assert.ok(source.includes('FEATURED_PAGE_SIZE = 4'))
  assert.ok(source.includes('rotateFeaturedItems'))
})

test('public knowledge-base discover keeps subscription and navigation actions', () => {
  assert.ok(source.includes('订阅成功'))
  assert.ok(source.includes('查看知识库'))
  assert.ok(source.includes('openKnowledgeBase'))
  assert.ok(source.includes('toggleSubscription'))
  assert.ok(source.includes('knowledgeBaseActionOptions'))
  assert.ok(source.includes('handleKnowledgeBaseAction'))
})
