import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const pageSource = readFileSync(new URL('./AdminAgentManagement.vue', import.meta.url), 'utf8')
const expertSource = readFileSync(new URL('./AdminExpertPackages.vue', import.meta.url), 'utf8')
const navigationSource = readFileSync(new URL('../config/navigation.ts', import.meta.url), 'utf8')
const routerSource = readFileSync(new URL('../router/index.ts', import.meta.url), 'utf8')

test('expert package management is embedded in the agent page', () => {
  assert.ok(pageSource.includes('<span>智能体</span>'))
  assert.ok(pageSource.includes('<span>导入专家</span>'))
  assert.ok(pageSource.includes('<AdminExpertPackages v-else embedded />'))
  assert.ok(expertSource.includes('embedded?: boolean'))
  assert.ok(expertSource.includes('导入 WorkBuddy 专家包，诊断通过后发布为平台智能体。'))
  assert.ok(expertSource.includes('<strong>Agent 智能体</strong>'))
  assert.ok(expertSource.includes('发布后作为 Agent 智能体使用'))
  assert.ok(!expertSource.includes('服务空间'))
})

test('standalone expert navigation is removed and the legacy route stays compatible', () => {
  assert.ok(!navigationSource.includes("key: 'expert-packages'"))
  assert.ok(!navigationSource.includes("path: '/experts'"))
  assert.ok(routerSource.includes("import('@admin/views/AdminAgentManagement.vue')"))
  assert.match(
    routerSource,
    /path: 'experts'[\s\S]*name: 'adminExpertPackages'[\s\S]*query: \{ \.\.\.to\.query, tab: 'import' \}/,
  )
})
