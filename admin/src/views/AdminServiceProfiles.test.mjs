import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./AdminServiceProfiles.vue', import.meta.url), 'utf8')

test('service templates keep the one-service-one-customer subject policy internal', () => {
  assert.match(source, /subject_policy:\s*\{\s*required: false,/)
  assert.match(source, /allowed_types: \[\]/)
  assert.doesNotMatch(source, /required: true,\s*allowed_types: \['member_family'\]/)
})

test('service templates expose the published Web default used by simple creation', () => {
  assert.match(source, /最近发布的低风险模板/)
  assert.match(source, /defaultPublishedTemplateKey/)
  assert.match(source, /Web 默认/)
})

test('service templates configure member Agents while the principal stays system-owned', () => {
  assert.match(source, /成员 Agent/)
  assert.match(source, /主理人仍可独立工作/)
  assert.match(source, /servicePrincipalAgentId = 'builtin-smart-reasoning'/)
  assert.match(source, /expert\.expert_ref !== servicePrincipalAgentId/)
  assert.doesNotMatch(source, /自动启用模板前至少添加一个专家/)
})
