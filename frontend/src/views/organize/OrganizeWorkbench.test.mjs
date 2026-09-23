import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const routesSource = readFileSync(new URL('./organizeRoutes.ts', import.meta.url), 'utf8')
const routerSource = readFileSync(new URL('../../router/index.ts', import.meta.url), 'utf8')
const stateSource = readFileSync(new URL('./organizeWorkbenchState.ts', import.meta.url), 'utf8')
const workbenchSource = readFileSync(new URL('./OrganizeWorkbench.vue', import.meta.url), 'utf8')
const organizeMenuSource = readFileSync(new URL('../../components/OrganizeMenu.vue', import.meta.url), 'utf8')
const configDetailSource = readFileSync(new URL('./OrganizeConfigDetail.vue', import.meta.url), 'utf8')
const outputListSource = readFileSync(new URL('./OrganizeOutputList.vue', import.meta.url), 'utf8')
const outputDetailSource = readFileSync(new URL('./OrganizeOutputDetail.vue', import.meta.url), 'utf8')
const apiSource = readFileSync(new URL('../../api/organize/index.ts', import.meta.url), 'utf8')

test('organize navigation exposes workbench, my organize, memory, and discover', () => {
  assert.ok(routesSource.includes("export type OrganizeTab = 'hub' | 'mine' | 'memory' | 'discover'"))
  assert.ok(routesSource.includes("label: '工作台'"))
  assert.ok(routesSource.includes("label: '我的整理'"))
  assert.ok(routesSource.includes("label: '记忆'"))
  assert.ok(routesSource.includes("label: '发现'"))

  const menuRoutes = routesSource.slice(
    routesSource.indexOf('export const ORGANIZE_MENU_ROUTES'),
    routesSource.indexOf('export const ORGANIZE_MEMORY_ASSET_ROUTES'),
  )
  assert.ok(!menuRoutes.includes("label: '发芽'"))
  assert.ok(routerSource.includes('component: organizeWorkspaceComponent'))
})

test('legacy sprout path redirects to my organize and templates load from the backend', () => {
  assert.ok(routerSource.includes('path: "organize/sprout"'))
  assert.ok(routerSource.includes('redirect: "/platform/organize/mine"'))
  assert.ok(apiSource.includes("'/api/v1/organize/templates'"))
  assert.ok(workbenchSource.includes('listOrganizeTemplates()'))
  assert.ok(workbenchSource.includes('从模板创建'))
})

test('workbench exposes persisted config cards, templates, and config dialog', () => {
  assert.ok(workbenchSource.includes('我的整理'))
  assert.ok(workbenchSource.includes('从模板创建'))
  assert.ok(workbenchSource.includes('<OrganizeConfigDialog'))
  assert.ok(workbenchSource.includes('openConfig(config.id)'))
  assert.ok(workbenchSource.includes('openCreateDialog(template.key)'))
  assert.ok(workbenchSource.includes('listOrganizeConfigs'))
  assert.ok(workbenchSource.includes('deleteOrganizeConfig'))
})

test('organize config detail header keeps identity and action aligned responsively', () => {
  assert.ok(configDetailSource.includes('display: flex;'))
  assert.ok(configDetailSource.includes('flex: 1;'))
  assert.ok(configDetailSource.includes('width: 100%;'))
  assert.ok(configDetailSource.includes('max-width: none;'))
  assert.ok(configDetailSource.includes('width: min(100%, 1040px);'))
  assert.ok(configDetailSource.includes('flex-wrap: wrap;'))
  assert.ok(!configDetailSource.includes('grid-template-columns: auto minmax(0, 1fr) auto'))
})

test('organize config validation points missing template errors at the template selector', () => {
  const dialogSource = readFileSync(
    new URL('./components/OrganizeConfigDialog.vue', import.meta.url),
    'utf8',
  )
  assert.ok(dialogSource.includes('const templateError = ref(\'\')'))
  assert.ok(dialogSource.includes(':class="{ \'is-error\': templateError }"'))
  assert.ok(dialogSource.includes('templateError.value = \'请选择整理模板\''))
  assert.ok(dialogSource.includes('organize-config-template-select.is-error'))
  assert.ok(!dialogSource.includes(':status="error ? \'error\' : undefined"'))
})

test('saved organize configs and jobs are served by organize APIs', () => {
  assert.ok(organizeMenuSource.includes('listOrganizeConfigs'))
  assert.ok(organizeMenuSource.includes('v-for="config in configuredOrganizeItems"'))
  assert.ok(organizeMenuSource.includes('ORGANIZE_ROUTE_BASE_PATH}/configs/'))
  assert.ok(configDetailSource.includes('runOrganizeConfig'))
  assert.ok(configDetailSource.includes('retryOrganizeJob'))
  assert.ok(configDetailSource.includes('cancelOrganizeJob'))
})

test('organize workbench no longer contains session mock data', () => {
  for (const marker of [
    'initialOutputs',
    'initialConfigs',
    'startMockOrganizeJob',
    'saveOrganizeConfig',
    'organizeWorkbenchState = reactive',
    "id: 'G1'",
    "id: 'O1'",
  ]) {
    assert.ok(!stateSource.includes(marker), `unexpected mock marker: ${marker}`)
  }
  assert.ok(outputListSource.includes('listOrganizeOutputs'))
  assert.ok(outputDetailSource.includes('getOrganizeOutput'))
})
