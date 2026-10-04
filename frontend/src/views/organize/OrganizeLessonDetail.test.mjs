import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./OrganizeLessonDetail.vue', import.meta.url), 'utf8')

test('course lesson media is loaded through the authenticated request helper', () => {
  assert.ok(source.includes("import { getDown } from '@/utils/request'"))
  assert.ok(source.includes('const rawBlob = await getDown(sourceUrl)'))
  assert.ok(source.includes('URL.createObjectURL(blob)'))
  assert.ok(source.includes('URL.revokeObjectURL(mediaObjectUrl)'))
  assert.ok(source.includes('onBeforeUnmount(() => {'))
})

test('course lesson media does not bind the protected API URL directly to the player', () => {
  assert.ok(source.includes(':src="mediaPlayerUrl"'))
  assert.ok(!source.includes(':src="mediaUrl"'))
})
