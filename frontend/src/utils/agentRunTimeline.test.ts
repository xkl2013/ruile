import assert from 'node:assert/strict'
import test from 'node:test'
import {
  buildAgentRunEventRows,
  buildAgentRunTimeline,
  formatAgentRunDuration,
} from './agentRunTimeline'

test('buildAgentRunTimeline merges start and finish events', () => {
  const timeline = buildAgentRunTimeline([
    {
      type: 'AGENT_STEP_STARTED',
      sequence: 1,
      stepId: 'step-1',
      stepType: 'planning',
      phase: 'planning',
      startedAt: '2026-09-20T06:00:00Z',
    },
    {
      type: 'AGENT_STEP_FINISHED',
      sequence: 2,
      stepId: 'step-1',
      stepType: 'planning',
      phase: 'planning',
      durationMs: 3250,
    },
    {
      type: 'RUN_FINISHED',
      sequence: 3,
      totalDurationMs: 5000,
    },
  ], { status: 'succeeded' })

  assert.equal(timeline.steps.length, 1)
  assert.equal(timeline.steps[0].label, '执行规划')
  assert.equal(timeline.steps[0].status, 'succeeded')
  assert.equal(timeline.steps[0].durationMs, 3250)
  assert.equal(timeline.totalDurationMs, 5000)
  assert.equal(timeline.active, false)
})

test('formatAgentRunDuration renders readable durations', () => {
  assert.equal(formatAgentRunDuration(900), '<1 秒')
  assert.equal(formatAgentRunDuration(12_000), '12 秒')
  assert.equal(formatAgentRunDuration(75_000), '1 分 15 秒')
})

test('buildAgentRunEventRows keeps every event and attaches step input and output', () => {
  const rows = buildAgentRunEventRows([
    {
      type: 'AGENT_STEP_STARTED',
      sequence: 1,
      stepId: 'step-1',
      phase: 'planning',
    },
    {
      type: 'AGENT_STEP_FINISHED',
      sequence: 2,
      stepId: 'step-1',
      phase: 'planning',
      durationMs: 1200,
    },
    {
      type: 'QUALITY_UPDATED',
      sequence: 3,
      quality: { score: 92, passed: true },
    },
  ], [{
    id: 'step-1',
    input: { prompt: '制定计划' },
    output: { objective: '完成计划' },
  }])

  assert.equal(rows.length, 3)
  assert.equal(rows[0].label, '开始执行规划')
  assert.deepEqual(rows[0].details[0], { label: '入参', value: { prompt: '制定计划' }, tone: 'default' })
  assert.deepEqual(rows[1].details[0], { label: '返回', value: { objective: '完成计划' }, tone: 'default' })
  assert.equal(rows[2].summary, '评分 92，已通过')
})
