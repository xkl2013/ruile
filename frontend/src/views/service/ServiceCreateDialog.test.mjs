import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const dialog = readFileSync(new URL('./ServiceCreateDialog.vue', import.meta.url), 'utf8')
const hub = readFileSync(new URL('./ServiceHub.vue', import.meta.url), 'utf8')

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

test('service creation uses the built-in assistant when experts are optional and empty', () => {
  assert.match(hub, /payload\.expertIds\.length/)
  assert.match(hub, /expert_ref: BUILTIN_SMART_REASONING_ID/)
  assert.match(hub, /expert_name: '服务助理'/)
  assert.match(hub, /activate: true/)
  assert.match(hub, /const session = await ensureServiceSession\(serviceId\)/)
})
