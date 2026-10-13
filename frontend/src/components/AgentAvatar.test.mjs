import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const avatarSource = readFileSync(new URL('./AgentAvatar.vue', import.meta.url), 'utf8')
const listSource = readFileSync(new URL('../views/agent/AgentList.vue', import.meta.url), 'utf8')
const selectorSource = readFileSync(new URL('./AgentSelector.vue', import.meta.url), 'utf8')

test('agent avatars support protected images and emoji fallbacks', () => {
  assert.ok(avatarSource.includes('avatarImageSource'))
  assert.ok(avatarSource.includes('avatarEmoji'))
  assert.ok(avatarSource.includes('hydrateProtectedFileImages'))
})

test('published platform experts render their imported avatar instead of the generic built-in icon', () => {
  assert.ok(listSource.includes('<AgentAvatar v-else :name="agent.name" :avatar="agent.avatar" size="small" />'))
  assert.ok(selectorSource.includes('<AgentAvatar v-else :name="agent.name" :avatar="agent.avatar" size="small" />'))
  assert.ok(selectorSource.includes('<AgentAvatar :name="agent.name" :avatar="agent.avatar" size="small" />'))
})

test('the admin Agent list repairs previously published expert metadata before loading', () => {
  assert.ok(listSource.includes("import { syncPublishedExpertAgents } from '@/api/expert-package'"))
  assert.ok(listSource.includes('await syncPublishedExpertAgents()'))
  assert.ok(listSource.includes("chatResources.invalidate('agents')"))
})

test('published experts are regular agents with an expert tag', () => {
  assert.ok(listSource.includes('isExpertAgent(agent)'))
  assert.ok(listSource.includes('>专家</t-tag>'))
  assert.ok(selectorSource.includes("agent.config?.tags?.includes('expert')"))
  assert.ok(selectorSource.includes('const customAgents = computed(() => agentsList.value.filter(a => !isBuiltinAgent(a.id)))'))
  assert.ok(!selectorSource.includes('已发布专家'))
})
