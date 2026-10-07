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

test('memory asset cards are grouped by organize status', () => {
  assert.ok(routesSource.includes("export type MemoryStatusKey = 'unorganized' | 'processing' | 'organized'"))
  assert.ok(routesSource.includes("label: '待整理'"))
  assert.ok(routesSource.includes("label: '整理中'"))
  assert.ok(routesSource.includes("label: '已整理'"))
  assert.ok(sourceIncludesStatusCards())
})

function sourceIncludesStatusCards() {
  const workspaceSource = readFileSync(new URL('./OrganizeWorkspace.vue', import.meta.url), 'utf8')
  return (
    workspaceSource.includes('memoryStatusCards') &&
    workspaceSource.includes('memoryOrganizationStatus(item.id)') &&
    workspaceSource.includes('listOrganizeJobs') &&
    workspaceSource.includes('listOrganizeOutputs')
  )
}

test('legacy sprout path redirects to my organize and templates load from the backend', () => {
  assert.ok(routerSource.includes('path: "organize/sprout"'))
  assert.ok(routerSource.includes('redirect: "/platform/organize/mine"'))
  assert.ok(routerSource.includes('path: "organize/editor/sprout/:id"'))
  assert.ok(apiSource.includes("'/api/v1/organize/templates'"))
  assert.ok(workbenchSource.includes('listOrganizeTemplates()'))
  assert.ok(workbenchSource.includes('从模板创建'))
})

test('workbench exposes persisted config cards, templates, and config dialog', () => {
  assert.ok(workbenchSource.includes('我的整理'))
  assert.ok(workbenchSource.includes('从模板创建'))
  assert.ok(!workbenchSource.includes('organize-pending-section'))
  assert.ok(!workbenchSource.includes('listOrganizePendingAssignments'))
  assert.ok(!workbenchSource.includes('openPendingAssignment(output.id)'))
  assert.ok(workbenchSource.includes('<OrganizeConfigDialog'))
  assert.ok(workbenchSource.includes('openConfig(config.id)'))
  assert.ok(workbenchSource.includes('openCreateDialog(template.key)'))
  assert.ok(!workbenchSource.includes('>从空白创建</t-button>'))
  assert.ok(!workbenchSource.includes('按要求整理'))
  assert.ok(!workbenchSource.includes('openRequirementDialog'))
  assert.ok(!workbenchSource.includes('previewOrganizeRequirement'))
  assert.ok(workbenchSource.includes('listOrganizeConfigs'))
  assert.ok(workbenchSource.includes('deleteOrganizeConfig'))
  assert.ok(workbenchSource.includes("const scene = template.scene.trim() || '其他'"))
  assert.ok(workbenchSource.includes('groups.set(scene, group)'))
})

test('organize workbench title matches the discover module heading', () => {
  assert.ok(workbenchSource.includes('.organize-page-header h1'))
  assert.ok(workbenchSource.includes('font-size: 21px;'))
  assert.ok(workbenchSource.includes('font-weight: 500;'))
  assert.ok(workbenchSource.includes('line-height: 30px;'))
  assert.ok(workbenchSource.includes('letter-spacing: 0;'))
})

test('organize config detail uses a compact timeline and result cards', () => {
  assert.ok(configDetailSource.includes('display: flex;'))
  assert.ok(configDetailSource.includes('flex: 1;'))
  assert.ok(configDetailSource.includes('width: 100%;'))
  assert.ok(configDetailSource.includes('class="organize-timeline-when"'))
  assert.ok(configDetailSource.includes('class="organize-timeline-node"'))
  assert.ok(configDetailSource.includes('width: min(100%, 920px);'))
  assert.ok(configDetailSource.includes('grid-template-columns: 76px 22px minmax(0, 1fr);'))
  assert.ok(configDetailSource.includes('grid-template-columns: minmax(0, 1fr) 28px;'))
  assert.ok(configDetailSource.includes('-webkit-line-clamp: 2;'))
  assert.ok(configDetailSource.includes('box-shadow: inset 3px 0 0 var(--td-brand-color)'))
  assert.ok(configDetailSource.includes('flex-wrap: wrap;'))
  assert.ok(!configDetailSource.includes('grid-template-columns: auto minmax(0, 1fr) auto'))
})

test('organize config detail header only shows the organize name and action', () => {
  assert.ok(configDetailSource.includes('<h2>{{ config.name }}</h2>'))
  assert.ok(!configDetailSource.includes('返回工作台'))
  assert.ok(!configDetailSource.includes('organize-detail-icon'))
  assert.ok(!configDetailSource.includes('scheduleLabel'))
})

test('organize config dialog keeps template selection outside the instruction field', () => {
  const dialogSource = readFileSync(
    new URL('./components/OrganizeConfigDialog.vue', import.meta.url),
    'utf8',
  )
  assert.ok(dialogSource.includes('<section v-if="config" class="organize-config-option">'))
  assert.ok(dialogSource.includes("const defaultTemplate = templates.value.find((template) => template.key === 'sprout_review')"))
  assert.ok(dialogSource.includes('class="organize-config-template-context"'))
  assert.ok(dialogSource.includes('template_key: form.templateKey'))
  assert.ok(!dialogSource.includes('organize-config-template-select'))
  assert.ok(!dialogSource.includes('handleTemplateChange'))
  assert.ok(!dialogSource.includes('applyTemplate'))
  assert.ok(!dialogSource.includes('<option value="">选择模板</option>'))
  assert.ok(dialogSource.includes('templateError.value = \'暂无可用整理方案，请稍后重试\''))
})

test('organize config service assignment is opt-in and required when enabled', () => {
  const dialogSource = readFileSync(
    new URL('./components/OrganizeConfigDialog.vue', import.meta.url),
    'utf8',
  )
  assert.ok(dialogSource.includes('v-model="form.assignToService"'))
  assert.ok(dialogSource.includes('assignToService: false'))
  assert.ok(dialogSource.includes('v-if="form.assignToService"'))
  assert.ok(dialogSource.includes('form.assignToService = Boolean(form.targetServiceId)'))
  assert.ok(dialogSource.includes('if (form.assignToService && !form.targetServiceId)'))
  assert.ok(dialogSource.includes("target_service_id: form.assignToService ? form.targetServiceId : ''"))
  assert.ok(!dialogSource.includes('暂不指定，生成待归属任务'))
})

test('saved organize configs and jobs are served by organize APIs', () => {
  assert.ok(organizeMenuSource.includes('listOrganizeConfigs'))
  assert.ok(organizeMenuSource.includes('v-for="config in configuredOrganizeItems"'))
  assert.ok(organizeMenuSource.includes('ORGANIZE_ROUTE_BASE_PATH}/configs/'))
  assert.ok(configDetailSource.includes('runOrganizeConfig'))
  assert.ok(configDetailSource.includes('retryOrganizeJob'))
  assert.ok(configDetailSource.includes('cancelOrganizeJob'))
})

test('completed job cards prioritize result title, preview, and created time', () => {
  assert.ok(stateSource.includes("jobMode: job.job_mode || (batchCount > 1 ? 'batch' : 'single')"))
  assert.ok(stateSource.includes('organizeOutputPreview'))
  assert.ok(stateSource.includes('preview: organizeOutputPreview(output.content'))
  assert.ok(configDetailSource.includes('listOrganizeOutputs({ config_id: configId'))
  assert.ok(configDetailSource.includes('resultCardTitle(outputForJob(job)!)'))
  assert.ok(configDetailSource.includes('outputForJob(job)!.preview'))
  assert.ok(configDetailSource.includes('outputCreatedLabel(outputForJob(job)!)'))
  assert.ok(!configDetailSource.includes('已处理 {{ job.processedCount }} / {{ job.selectedCount }}'))
  assert.ok(!configDetailSource.includes('模板版本 {{ job.templateVersion }}'))
  assert.ok(!configDetailSource.includes('{{ job.conclusionCount }} 结论'))
  assert.ok(!configDetailSource.includes('待手动分配'))
  assert.ok(apiSource.includes('config_id?: string'))
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

test('historical outputs without template metadata remain readable', () => {
  assert.ok(stateSource.includes("templateName: output.template_key"))
  assert.ok(stateSource.includes(": '历史内容'"))
  assert.ok(outputDetailSource.includes(':content="output.content"'))
})

test('output detail prioritizes the finished report over internal metadata', () => {
  assert.ok(outputDetailSource.includes('<h2>{{ output.title }}</h2>'))
  assert.ok(outputDetailSource.includes('<OrganizeMarkdownRenderer :content="output.content" profile="report" />'))
  assert.ok(!outputDetailSource.includes('output.templateName'))
  assert.ok(!outputDetailSource.includes('output.templateVersion'))
  assert.ok(!outputDetailSource.includes('output.sourceCount'))
  assert.ok(!outputDetailSource.includes('output.assignmentReason'))
  assert.ok(!outputDetailSource.includes('v-for="field in output.fields"'))
  assert.ok(!outputDetailSource.includes('organize-output-fields'))
})
