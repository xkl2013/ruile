import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const selectorPath = fileURLToPath(new URL('./AgentSelector.vue', import.meta.url))
const inputPath = fileURLToPath(new URL('./Input-field.vue', import.meta.url))
const selectorSource = fs.readFileSync(selectorPath, 'utf8')
const inputSource = fs.readFileSync(inputPath, 'utf8')

test('agent selector respects workspace response-tier readiness', () => {
  assert.match(selectorSource, /responseTierEnabled\?: boolean/)
  assert.match(selectorSource, /responseTierEnabled: props\.responseTierEnabled/)
  assert.match(
    inputSource,
    /:response-tier-enabled="responseTierConfig\.enabled"/,
  )
})

test('current agent still displays missing readiness reasons', () => {
  assert.match(
    selectorSource,
    /<span v-if="isDetailCurrent" class="detail-current">/,
  )
  assert.match(
    selectorSource,
    /<div v-if="activeDetailNotReadyLabels\.length" class="detail-not-ready">/,
  )
  assert.doesNotMatch(
    selectorSource,
    /<div v-else-if="activeDetailNotReadyLabels\.length" class="detail-not-ready">/,
  )
})
