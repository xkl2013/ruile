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

test('service creation uses the built-in assistant when experts are optional and empty', () => {
  assert.match(hub, /payload\.expertIds\.length/)
  assert.match(hub, /expert_ref: BUILTIN_SMART_REASONING_ID/)
  assert.match(hub, /expert_name: '服务助理'/)
  assert.match(hub, /activate: true/)
  assert.match(hub, /const session = await ensureServiceSession\(serviceId\)/)
})

test('templates live under my services and populate the creation dialog', () => {
  assert.match(hub, /我的服务/)
  assert.match(hub, /从模板创建/)
  assert.match(hub, /v-for="group in filteredTemplateGroups"/)
  assert.match(hub, /@click="openCreate\(template\)"/)
  assert.match(hub, /:source="createSource"/)
  assert.match(dialog, /template\?\.description/)
  assert.match(dialog, /template\?\.instruction/)
  assert.doesNotMatch(dialog, /推荐配置/)
  assert.doesNotMatch(dialog, /当前工作画像/)
  assert.doesNotMatch(dialog, /切换模板/)
  assert.match(dialog, /已带入模板/)
  assert.match(dialog, /命中后免确认/)
  assert.match(dialog, /直接创建空间/)
})

test('planned service flow confirms instruction-generated blueprints before activation', () => {
  assert.match(hub, /applyServiceTemplate/)
  assert.match(hub, /previewServiceBlueprint/)
  assert.match(hub, /confirmServiceBlueprint/)
  assert.match(hub, /确认服务空间蓝图/)
  assert.match(hub, /activate: true/)
  assert.match(hub, /空间档案/)
  assert.match(hub, /首页摘要/)
})

test('service workspace exposes generated profile and summary panels on web only', () => {
  assert.match(hub, /getServiceProfile/)
  assert.match(hub, /getServiceSummary/)
  assert.match(hub, /refreshServiceSummary/)
  assert.match(hub, /route\.meta\.mobileEntry/)
  assert.match(hub, /serviceProfile\?\.schema/)
  assert.match(hub, /serviceSummary\?\.schema/)
})

test('service workspace refreshes and focuses imported organize sources from the route', () => {
  assert.match(hub, /route\.query\.context_source/)
  assert.match(hub, /await loadContextSources\(queryService\)/)
  assert.match(hub, /panel\.value = 'context'/)
  assert.match(hub, /sourceMatchesRoute\(source\)/)
  assert.match(hub, /刚带入/)
})

test('service workspace exposes service-scoped reminders and status actions', () => {
  assert.match(hub, /panel === 'reminders'/)
  assert.match(hub, /listServiceReminders/)
  assert.match(hub, /createServiceReminder/)
  assert.match(hub, /updateServiceReminder/)
  assert.match(hub, /deleteServiceReminder/)
  assert.match(hub, /nextReminderStatuses/)
  assert.match(hub, /服务待办/)
  assert.match(hub, /待办已创建/)
})

test('market research template covers the full research-to-review workflow', () => {
  assert.match(hubState, /市场调研与竞品分析协同助手/)
  assert.match(hubState, /阶段一：调研课题定义/)
  assert.match(hubState, /阶段七：持续跟踪/)
  assert.match(hubState, /事实、分析和建议/)
  assert.match(hubState, /调研结论评审/)
  assert.match(hubState, /调研主体/)
})

test('early childhood membership template covers member lifecycle service', () => {
  assert.match(hubState, /早教机构会员服务/)
  assert.match(hubState, /会员家庭/)
  assert.match(hubState, /会员档案/)
  assert.match(hubState, /课程预约与调整/)
  assert.match(hubState, /权益与使用/)
  assert.match(hubState, /投诉与负面反馈/)
  assert.match(hubState, /儿童健康或安全相关情形/)
  assert.match(hubState, /续费风险/)
})
