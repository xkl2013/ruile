import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const uploadSources = readFileSync(new URL('./utils/uploadSources.ts', import.meta.url), 'utf8')
const utils = readFileSync(new URL('../../utils/index.ts', import.meta.url), 'utf8')

test('knowledge upload accepts video extensions in the default whitelist', () => {
  for (const ext of ['mp4', 'mov', 'webm', 'mkv']) {
    assert.match(utils, new RegExp(`"${ext}"`))
  }
})

test('knowledge upload no longer filters videos before validation', () => {
  assert.doesNotMatch(uploadSources, new RegExp(['video', 'Filtered', 'Count'].join('')))
  assert.doesNotMatch(uploadSources, new RegExp(['UPLOAD', 'VIDEO', 'EXTENSIONS'].join('_')))
})
