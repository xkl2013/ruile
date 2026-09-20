import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const agentListPath = fileURLToPath(new URL('./AgentList.vue', import.meta.url))
const agentEditorPath = fileURLToPath(new URL('./AgentEditorModal.vue', import.meta.url))
const agentListSource = fs.readFileSync(agentListPath, 'utf8')
const agentEditorSource = fs.readFileSync(agentEditorPath, 'utf8')

test('agent edits refresh the canonical chat agent cache', () => {
  const handler = agentListSource.match(
    /const handleEditorSuccess = async \(agent\?: CustomAgent\) => \{([\s\S]*?)\n\}/,
  )?.[1] || ''

  assert.match(handler, /chatResources\.invalidate\('agents'\)/)
  assert.match(handler, /await chatResources\.ensureAgents\(true\)/)
  assert.match(handler, /await fetchList\(false\)/)
})

test('agent editor emits the updated API response after edit', () => {
  const updateBranch = agentEditorSource.match(
    /\} else \{\n([\s\S]*?)\n    \}\n  \} catch/,
  )?.[1] || ''

  assert.match(updateBranch, /const result: any = await updateAgent/)
  assert.match(updateBranch, /const updated = result\?\.data as CustomAgent \| undefined/)
  assert.match(updateBranch, /emit\('success', updated\)/)
})
