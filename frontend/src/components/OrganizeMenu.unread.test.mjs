import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const menuSource = readFileSync(new URL('./OrganizeMenu.vue', import.meta.url), 'utf8')
const outputDetailSource = readFileSync(
  new URL('../views/organize/OrganizeOutputDetail.vue', import.meta.url),
  'utf8',
)
const configDetailSource = readFileSync(
  new URL('../views/organize/OrganizeConfigDetail.vue', import.meta.url),
  'utf8',
)
const stateSource = readFileSync(
  new URL('../views/organize/organizeWorkbenchState.ts', import.meta.url),
  'utf8',
)
const apiSource = readFileSync(new URL('../api/organize/index.ts', import.meta.url), 'utf8')

test('organize menu shows and immediately clears unread output markers', () => {
  assert.ok(menuSource.includes('v-if="config.hasUnreadOutput"'))
  assert.ok(menuSource.includes('class="organize-menu-item-unread"'))
  assert.ok(menuSource.includes('aria-label="有未读整理报告"'))
  assert.ok(menuSource.includes('window.addEventListener(ORGANIZE_CONFIG_UNREAD_EVENT'))
  assert.ok(menuSource.includes("latestOutputId !== detail?.viewedOutputId"))
  assert.ok(configDetailSource.includes('new CustomEvent(ORGANIZE_CONFIG_UNREAD_EVENT'))
  assert.ok(configDetailSource.includes('hasUnreadOutput: mapped.hasUnreadOutput'))
})

test('opening an organize output persists the read watermark', () => {
  assert.ok(apiSource.includes('has_unread_output?: boolean'))
  assert.ok(apiSource.includes('/read-output'))
  assert.ok(stateSource.includes("typeof config.has_unread_output === 'boolean'"))
  assert.ok(stateSource.includes('fallbackConfigHasUnreadOutput(config)'))
  assert.ok(stateSource.includes('markOrganizeConfigOutputLocallyRead'))
  assert.ok(outputDetailSource.includes('markOrganizeConfigOutputRead(item.configId, item.id)'))
  assert.ok(outputDetailSource.includes('detail.hasUnreadOutput = Boolean(response.data?.has_unread_output)'))
  assert.ok(outputDetailSource.includes('viewedOutputId: item.id'))
  assert.ok(outputDetailSource.includes('new CustomEvent(ORGANIZE_CONFIG_UNREAD_EVENT'))
})
