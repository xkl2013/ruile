import assert from 'node:assert/strict'
import test from 'node:test'

import {
  applyBucketCountProbe,
  bucketVisible,
  createEmptyBucket,
  mergeBucketPage,
  prependSessionToWebBucket,
  type BucketDefinition,
} from './sessionSidebarBuckets.ts'

const webDef: BucketDefinition = {
  key: 'web',
  apiSource: 'web',
  label: 'Chats',
  kind: 'web',
}

const imDef: BucketDefinition = {
  key: 'im:feishu',
  apiSource: 'feishu',
  label: 'Feishu',
  kind: 'im',
  platform: 'feishu',
}

test('bucketVisible hides channel buckets until count is known', () => {
  const bucket = createEmptyBucket(imDef)
  assert.equal(bucketVisible(bucket), false)
})

test('bucketVisible hides channel buckets with zero sessions after probe', () => {
  const bucket = applyBucketCountProbe(createEmptyBucket(imDef), 0)
  assert.equal(bucketVisible(bucket), false)
})

test('bucketVisible shows channel buckets with sessions after probe', () => {
  const bucket = applyBucketCountProbe(createEmptyBucket(imDef), 3)
  assert.equal(bucketVisible(bucket), true)
})

test('mergeBucketPage preserves a generated title when a stale API row has no title', () => {
  const bucket = {
    ...createEmptyBucket(webDef),
    items: [{
      id: 'session-1',
      title: '生成后的标题',
      isNoTitle: false,
      updated_at: '2026-10-12T10:00:00.000Z',
    }],
  }

  const next = mergeBucketPage(bucket, [{
    id: 'session-1',
    title: '',
    isNoTitle: true,
    updated_at: '2026-10-12T10:01:00.000Z',
  }], 1, 1)

  assert.equal(next.items[0].title, '生成后的标题')
  assert.equal((next.items[0] as typeof bucket.items[number]).isNoTitle, false)
  assert.equal(next.items[0].updated_at, '2026-10-12T10:01:00.000Z')
})

test('prependSessionToWebBucket inserts new session at front and bumps total', () => {
  const bucket = { ...createEmptyBucket(webDef), total: 2, items: [{ id: 'a' }] }
  const next = prependSessionToWebBucket(bucket, { id: 'b', title: 'New' })
  assert.deepEqual(next.items.map((s) => s.id), ['b', 'a'])
  assert.equal(next.total, 3)
})

test('prependSessionToWebBucket is idempotent for existing session', () => {
  const bucket = { ...createEmptyBucket(webDef), total: 1, items: [{ id: 'a' }] }
  const next = prependSessionToWebBucket(bucket, { id: 'a', title: 'Same' })
  assert.equal(next, bucket)
})
