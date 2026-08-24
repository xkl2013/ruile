import assert from 'node:assert/strict'
import test from 'node:test'

import {
  buildSmartNoteTags,
  mergeNoteMetadata,
  normalizeNoteTags,
} from './noteEditor'

test('normalizeNoteTags trims, splits, and deduplicates', () => {
  assert.deepEqual(
    normalizeNoteTags(' #招生 , 家长; 招生\n@复盘 '),
    ['招生', '家长', '复盘'],
  )
})

test('buildSmartNoteTags keeps the strongest repeated phrase first', () => {
  const tags = buildSmartNoteTags('招生沟通', '招生沟通 招生沟通', [])

  assert.ok(tags.length > 0)
  assert.equal(tags[0], '招生沟通')
  assert.equal(new Set(tags).size, tags.length)
})

test('mergeNoteMetadata preserves other fields and normalizes tags', () => {
  const metadata = mergeNoteMetadata({ source: '手动输入', extra: true }, [' 标签一 ', '标签二', '标签一'])

  assert.deepEqual(metadata, {
    source: '手动输入',
    extra: true,
    tags: ['标签一', '标签二'],
  })
})
