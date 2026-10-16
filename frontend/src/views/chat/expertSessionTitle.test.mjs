import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const createChatSource = readFileSync(new URL('../creatChat/creatChat.vue', import.meta.url), 'utf8')
const agentListSource = readFileSync(new URL('../agent/AgentList.vue', import.meta.url), 'utf8')

function sliceFlow(source, startMarker, endMarker) {
  const start = source.indexOf(startMarker)
  const end = source.indexOf(endMarker, start)
  assert.notEqual(start, -1, `missing flow start: ${startMarker}`)
  assert.notEqual(end, -1, `missing flow end: ${endMarker}`)
  return source.slice(start, end)
}

test('expert entry sessions stay untitled until AI title generation completes', () => {
  const homeExpertFlow = sliceFlow(
    createChatSource,
    'const selectExpert = async',
    'const showChatContextualGuide',
  )
  const sharedExpertFlow = sliceFlow(
    agentListSource,
    'async function handleUseSharedAgentInChat',
    'const handleEdit',
  )

  for (const flow of [homeExpertFlow, sharedExpertFlow]) {
    assert.match(flow, /createSessions\(\{\}\)/)
    assert.doesNotMatch(flow, /createSessions\(\{\s*title:/)
    assert.match(flow, /title:\s*t\('createChat\.newSessionTitle'\)/)
    assert.match(flow, /isNoTitle:\s*true/)
  }
})
