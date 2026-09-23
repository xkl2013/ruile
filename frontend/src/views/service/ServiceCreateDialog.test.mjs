import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const dialog = readFileSync(new URL('./ServiceCreateDialog.vue', import.meta.url), 'utf8')
const hub = readFileSync(new URL('./ServiceHub.vue', import.meta.url), 'utf8')
const hubState = readFileSync(new URL('./serviceHubState.ts', import.meta.url), 'utf8')

test('service creation uses a modal and loads account-visible knowledge bases', () => {
  assert.match(dialog, /<Teleport to="body">/)
  assert.match(dialog, /role="dialog"/)
  assert.match(dialog, /fetchMyKnowledgeBases\(true\)/)
  assert.match(dialog, /validAccountKnowledgeBases/)
  assert.match(dialog, /知识库/)
  assert.match(hub, /instruction: payload\.instruction/)
})

test('service creation modal does not expose skills', () => {
  assert.doesNotMatch(dialog, /技能/)
  assert.doesNotMatch(hub, /serviceSkills/)
  assert.doesNotMatch(hub, /view === 'create'/)
})

test('service experts come from published system experts and use a searchable picker', () => {
  assert.match(dialog, /ServiceExpertPickerDialog/)
  assert.match(dialog, /serviceExpertsLoading/)
  assert.match(dialog, /serviceExpertsError/)
  assert.match(hubState, /listPublishedExperts/)
  assert.match(hubState, /loadServiceExperts/)
})
