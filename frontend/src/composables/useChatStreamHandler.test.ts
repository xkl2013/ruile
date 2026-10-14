import assert from 'node:assert/strict'
import test from 'node:test'

import { mergeServiceFactProposal } from './serviceProposalState.ts'

test('keeps a resolved service proposal when an older pending event is replayed', () => {
  const current = {
    id: 'proposal-1',
    status: 'confirmed',
    resolved_at: '2026-10-14T09:26:41+08:00',
  }
  const incoming = {
    id: 'proposal-1',
    status: 'pending',
  }

  assert.deepEqual(mergeServiceFactProposal(current, incoming), current)
})

test('allows a resolved service proposal to replace its pending stream state', () => {
  const current = {
    id: 'proposal-1',
    status: 'pending',
  }
  const incoming = {
    id: 'proposal-1',
    status: 'confirmed',
    resolved_at: '2026-10-14T09:26:41+08:00',
  }

  assert.deepEqual(mergeServiceFactProposal(current, incoming), {
    ...current,
    ...incoming,
  })
})
