import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./AgentEditorModal.vue', import.meta.url), 'utf8')

test('agent editor keeps the Skills entry visible in Agent mode', () => {
  assert.ok(source.includes("if (isAgentMode.value) {"))
  assert.ok(source.includes("items.push({ key: 'skills'"))
  assert.ok(!source.includes("if (isAgentMode.value && skillsAvailable.value)"))
})

test('agent editor still publishes skill data when optional dependencies fail', () => {
  assert.ok(source.includes('Promise.allSettled(['))
  assert.ok(source.includes('editorResources.prefetchAgentEditorDeps()'))
  assert.ok(source.includes('skillsAvailable.value = editorResources.skillsAvailable'))
  assert.ok(source.includes('skillOptions.value = editorResources.skills'))
})
