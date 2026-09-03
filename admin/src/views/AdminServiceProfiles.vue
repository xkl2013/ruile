<template>
  <section class="service-config-page">
    <div class="service-config-page__header">
      <div>
        <h2>服务配置</h2>
        <p>服务项继续在这里展示；员工分身在成员管理维护，AI 根据分身描述匹配可用服务能力。</p>
      </div>
    </div>

    <div class="service-config-summary">
      <article>
        <span>服务项</span>
        <strong>{{ serviceItems.length }}</strong>
        <em>保留展示</em>
      </article>
      <article>
        <span>分身来源</span>
        <strong>成员管理</strong>
        <em>必填描述</em>
      </article>
      <article>
        <span>启用方式</span>
        <strong>AI 匹配</strong>
        <em>按岗位职责判断</em>
      </article>
    </div>

    <section class="service-config-panel">
      <div class="panel-title">
        <span>
          <strong>服务项</strong>
          <em>这些服务能力保留在服务配置中查看；具体成员能力由员工分身描述驱动。</em>
        </span>
        <t-tag theme="warning" variant="light">分身不在此页维护</t-tag>
      </div>

      <t-alert
        v-if="loadFailed"
        theme="warning"
        message="服务项接口暂不可用，当前展示内置服务项。"
        class="service-config-alert"
      />

      <div class="service-item-grid">
        <article v-for="item in serviceItems" :key="item.agent_domain" class="service-item-card">
          <div class="service-item-card__head">
            <span class="service-item-card__icon">
              <t-icon :name="serviceIcon(item.agent_domain)" />
            </span>
            <span>
              <strong>{{ item.display_name }}</strong>
              <em>{{ serviceCategory(item.agent_domain) }}</em>
            </span>
            <span class="service-item-card__actions">
              <t-tag :theme="item.default_enabled ? 'success' : 'primary'" variant="light">
                {{ item.default_enabled ? '基础项' : '按分身启用' }}
              </t-tag>
              <t-button size="small" variant="outline" @click="openAgentEditor(item)">
                <template #icon><t-icon name="edit-1" /></template>
                编辑
              </t-button>
            </span>
          </div>

          <p>{{ item.description }}</p>

          <dl>
            <div>
              <dt>输入来源</dt>
              <dd>{{ serviceInputSource(item.agent_domain) }}</dd>
            </div>
            <div>
              <dt>输出结果</dt>
              <dd>{{ serviceOutput(item.agent_domain) }}</dd>
            </div>
            <div>
              <dt>成员分身</dt>
              <dd>在成员管理中配置</dd>
            </div>
          </dl>
        </article>
      </div>
    </section>

    <t-dialog
      v-model:visible="editDialogVisible"
      :header="editDialogTitle"
      width="680px"
      :confirm-btn="{ content: '保存配置', theme: 'primary', loading: editSaving, disabled: !canSaveAgentSetting }"
      :cancel-btn="{ content: '取消', disabled: editSaving }"
      :close-on-overlay-click="!editSaving"
      destroy-on-close
      @confirm="saveAgentSetting"
      @cancel="closeAgentEditor"
      @close="closeAgentEditor"
    >
      <div class="agent-edit-dialog">
        <div v-if="editingItem" class="agent-edit-summary">
          <span class="service-item-card__icon">
            <t-icon :name="serviceIcon(editingItem.agent_domain)" />
          </span>
          <div>
            <strong>{{ editingItem.display_name }}</strong>
            <p>{{ editingItem.description }}</p>
          </div>
        </div>

        <t-alert
          v-if="profileLoadFailed"
          theme="warning"
          message="成员分身读取失败，请确认当前账号有管理权限。"
        />

        <t-form :data="editForm" label-align="top" @submit.prevent>
          <t-form-item label="配置分身" name="profileId">
            <t-select
              v-model="editForm.profileId"
              :options="profileOptions"
              :loading="profileLoading || editLoading"
              :disabled="editSaving"
              placeholder="选择需要编辑的成员分身"
              @change="handleEditProfileChange"
            />
          </t-form-item>

          <div v-if="profileOptions.length === 0" class="agent-edit-empty">
            暂无可配置分身，请先在成员管理中补充分身描述。
          </div>

          <template v-else>
            <div class="agent-edit-row">
              <t-form-item label="启用状态" name="enabled">
                <t-switch v-model="editForm.enabled" :disabled="editSaving" />
              </t-form-item>
              <t-form-item label="服务名称" name="displayName">
                <t-input v-model="editForm.displayName" :maxlength="60" clearable />
              </t-form-item>
            </div>

            <t-form-item label="工作文档目录" name="workDocDirectory">
              <t-input
                v-model="editForm.workDocDirectory"
                :maxlength="120"
                clearable
                placeholder="例如：客户/、线索/、排课/"
              />
            </t-form-item>

            <div class="agent-edit-row">
              <t-form-item label="绑定知识库 ID" name="knowledgeBaseIds">
                <t-textarea
                  v-model="editForm.knowledgeBaseIds"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  placeholder="每行一个，或用逗号分隔"
                />
              </t-form-item>
              <t-form-item label="允许 Skill" name="selectedSkills">
                <t-textarea
                  v-model="editForm.selectedSkills"
                  :autosize="{ minRows: 2, maxRows: 4 }"
                  placeholder="每行一个，或用逗号分隔"
                />
              </t-form-item>
            </div>

            <t-form-item label="记忆过滤规则 JSON" name="memoryFilter">
              <t-textarea
                v-model="editForm.memoryFilter"
                :autosize="{ minRows: 3, maxRows: 6 }"
                placeholder='例如：{"source":"organize_memories"}'
              />
            </t-form-item>

            <t-form-item label="输出策略 JSON" name="outputPolicy">
              <t-textarea
                v-model="editForm.outputPolicy"
                :autosize="{ minRows: 3, maxRows: 6 }"
                placeholder='例如：{"requires_user_confirmation":true}'
              />
            </t-form-item>
          </template>
        </t-form>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listServiceAgentSettings,
  listServiceAgentTemplates,
  listServiceWorkProfiles,
  replaceServiceAgentSettings,
  type ServiceAgentTemplate,
  type ServiceWorkProfile,
  type WorkProfileAgentSetting,
} from '@/api/service'

type ServiceItem = ServiceAgentTemplate
type AgentSettingForm = {
  profileId: string
  enabled: boolean
  displayName: string
  workDocDirectory: string
  knowledgeBaseIds: string
  selectedSkills: string
  memoryFilter: string
  outputPolicy: string
}

const loading = ref(false)
const loadFailed = ref(false)
const remoteItems = ref<ServiceItem[]>([])
const profileLoading = ref(false)
const profileLoadFailed = ref(false)
const workProfiles = ref<ServiceWorkProfile[]>([])
const editDialogVisible = ref(false)
const editLoading = ref(false)
const editSaving = ref(false)
const editingItem = ref<ServiceItem | null>(null)
const agentSettings = ref<WorkProfileAgentSetting[]>([])
const agentSettingsProfileId = ref('')
const editForm = ref<AgentSettingForm>({
  profileId: '',
  enabled: false,
  displayName: '',
  workDocDirectory: '',
  knowledgeBaseIds: '',
  selectedSkills: '',
  memoryFilter: '{}',
  outputPolicy: '{}',
})

const fallbackItems: ServiceItem[] = [
  {
    agent_domain: 'memory_router',
    display_name: '记忆识别',
    description: '识别记忆属于哪个服务场景，作为后续服务能力匹配的基础。',
    default_enabled: true,
    user_visible: false,
    work_doc_directory: '路由/',
  },
  {
    agent_domain: 'lead_intake',
    display_name: '线索录入',
    description: '从咨询、试听、报名意向记忆中整理线索草稿和缺失信息。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '线索/',
  },
  {
    agent_domain: 'sales_consulting',
    display_name: '招生咨询',
    description: '生成异议处理、邀约话术、试听后跟进和下一步建议。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '线索/',
  },
  {
    agent_domain: 'customer_service',
    display_name: '客户服务',
    description: '整理客户摘要、跟进记录、续费窗口和服务闭环事项。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '客户/',
  },
  {
    agent_domain: 'schedule_coordination',
    display_name: '排课调课',
    description: '识别请假、补课、排课和调课信号，生成待确认安排。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '排课/',
  },
  {
    agent_domain: 'after_sale_risk',
    display_name: '售后风险',
    description: '识别投诉、不满、退款、退费等高风险服务信号，推动处理闭环。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '售后风险/',
  },
  {
    agent_domain: 'daily_review',
    display_name: '日报复盘',
    description: '按用户触发汇总服务提醒、风险、动作闭环和知识补齐建议。',
    default_enabled: false,
    user_visible: false,
    work_doc_directory: '日报/',
  },
]

const serviceItems = computed<ServiceItem[]>(() => {
  const merged = new Map(fallbackItems.map((item) => [item.agent_domain, item]))
  remoteItems.value.forEach((item) => {
    merged.set(item.agent_domain, {
      ...merged.get(item.agent_domain),
      ...item,
    })
  })
  return fallbackItems
    .map((item) => merged.get(item.agent_domain))
    .filter((item): item is ServiceItem => Boolean(item))
})

const profileOptions = computed(() =>
  workProfiles.value.map((profile) => ({
    label: [
      profile.name || '未命名分身',
      profile.default_profile ? '默认' : '',
      profile.enabled ? '' : '未启用',
    ].filter(Boolean).join(' · '),
    value: profile.id,
  })),
)

const editDialogTitle = computed(() => {
  const name = editingItem.value?.display_name || '服务项'
  return `编辑${name}`
})

const canSaveAgentSetting = computed(() => {
  return Boolean(editingItem.value && editForm.value.profileId && profileOptions.value.length > 0)
})

async function loadServiceItems() {
  if (loading.value) return
  loading.value = true
  loadFailed.value = false
  try {
    const response = await listServiceAgentTemplates()
    remoteItems.value = response?.data || []
  } catch (error) {
    console.warn('[AdminServiceProfiles] Failed to load service agent templates:', error)
    remoteItems.value = []
    loadFailed.value = true
  } finally {
    loading.value = false
  }
}

async function loadWorkProfiles() {
  if (profileLoading.value) return
  profileLoading.value = true
  profileLoadFailed.value = false
  try {
    const response = await listServiceWorkProfiles()
    workProfiles.value = response?.data || []
    if (!editForm.value.profileId) {
      editForm.value.profileId = preferredProfileId()
    }
  } catch (error) {
    console.warn('[AdminServiceProfiles] Failed to load work profiles:', error)
    workProfiles.value = []
    profileLoadFailed.value = true
  } finally {
    profileLoading.value = false
  }
}

async function loadAgentSettings(profileId: string, force = false) {
  if (!profileId) {
    agentSettings.value = []
    agentSettingsProfileId.value = ''
    return
  }
  if (!force && agentSettingsProfileId.value === profileId) return
  editLoading.value = true
  try {
    const response = await listServiceAgentSettings(profileId)
    agentSettings.value = response?.data || []
    agentSettingsProfileId.value = profileId
  } catch (error) {
    console.warn('[AdminServiceProfiles] Failed to load agent settings:', error)
    agentSettings.value = []
    agentSettingsProfileId.value = ''
    MessagePlugin.error('Agent 配置读取失败')
  } finally {
    editLoading.value = false
  }
}

async function openAgentEditor(item: ServiceItem) {
  editingItem.value = item
  editDialogVisible.value = true
  if (workProfiles.value.length === 0) {
    await loadWorkProfiles()
  }
  if (!editForm.value.profileId) {
    editForm.value.profileId = preferredProfileId()
  }
  if (editForm.value.profileId) {
    await loadAgentSettings(editForm.value.profileId)
  }
  applyItemToForm(item)
}

function closeAgentEditor() {
  if (editSaving.value) return
  editDialogVisible.value = false
  editingItem.value = null
}

async function handleEditProfileChange(value: unknown) {
  const profileId = String(value || '')
  editForm.value.profileId = profileId
  await loadAgentSettings(profileId, true)
  if (editingItem.value) {
    applyItemToForm(editingItem.value)
  }
}

async function saveAgentSetting() {
  const item = editingItem.value
  const profileId = editForm.value.profileId
  if (!item || !profileId) return

  const memoryFilter = parseJSONMap(editForm.value.memoryFilter, '记忆过滤规则')
  if (!memoryFilter) return
  const outputPolicy = parseJSONMap(editForm.value.outputPolicy, '输出策略')
  if (!outputPolicy) return

  editSaving.value = true
  try {
    const freshResponse = await listServiceAgentSettings(profileId)
    agentSettings.value = freshResponse?.data || []
    agentSettingsProfileId.value = profileId
    const nextSettings = agentSettings.value
      .filter((setting) => setting.agent_domain !== item.agent_domain)
      .map(settingPayload)
    nextSettings.push({
      agent_id: agentSettings.value.find((setting) => setting.agent_domain === item.agent_domain)?.agent_id || undefined,
      agent_domain: item.agent_domain,
      enabled: editForm.value.enabled,
      display_name: editForm.value.displayName.trim() || item.display_name,
      display_order: serviceItemOrder(item.agent_domain),
      memory_filter: memoryFilter,
      knowledge_base_ids: parseList(editForm.value.knowledgeBaseIds),
      work_doc_directory: editForm.value.workDocDirectory.trim() || item.work_doc_directory || fallbackDirectory(item.agent_domain),
      selected_skills: parseList(editForm.value.selectedSkills),
      output_policy: outputPolicy,
    })
    nextSettings.sort((a, b) => serviceItemOrder(a.agent_domain || '') - serviceItemOrder(b.agent_domain || ''))
    const response = await replaceServiceAgentSettings(profileId, nextSettings)
    agentSettings.value = response?.data || []
    agentSettingsProfileId.value = profileId
    MessagePlugin.success('Agent 配置已保存')
    editDialogVisible.value = false
    editingItem.value = null
  } catch (error: any) {
    console.warn('[AdminServiceProfiles] Failed to save agent setting:', error)
    MessagePlugin.error(error?.message || 'Agent 配置保存失败')
  } finally {
    editSaving.value = false
  }
}

function preferredProfileId() {
  return (
    workProfiles.value.find((profile) => profile.default_profile && profile.enabled)?.id ||
    workProfiles.value.find((profile) => profile.enabled)?.id ||
    workProfiles.value[0]?.id ||
    ''
  )
}

function applyItemToForm(item: ServiceItem) {
  const setting = agentSettings.value.find((entry) => entry.agent_domain === item.agent_domain)
  editForm.value = {
    profileId: editForm.value.profileId,
    enabled: setting?.enabled ?? item.default_enabled,
    displayName: setting?.display_name || item.display_name,
    workDocDirectory: setting?.work_doc_directory || item.work_doc_directory || fallbackDirectory(item.agent_domain),
    knowledgeBaseIds: formatList(setting?.knowledge_base_ids || []),
    selectedSkills: formatList(setting?.selected_skills || item.selected_skills || []),
    memoryFilter: formatJSON(setting?.memory_filter || item.memory_filter || {}),
    outputPolicy: formatJSON(setting?.output_policy || item.output_policy || {}),
  }
}

function settingPayload(setting: WorkProfileAgentSetting): Partial<WorkProfileAgentSetting> {
  return {
    agent_id: setting.agent_id || undefined,
    agent_domain: setting.agent_domain,
    enabled: setting.enabled,
    display_name: setting.display_name,
    display_order: setting.display_order,
    memory_filter: setting.memory_filter || {},
    knowledge_base_ids: setting.knowledge_base_ids || [],
    work_doc_directory: setting.work_doc_directory,
    selected_skills: setting.selected_skills || [],
    output_policy: setting.output_policy || {},
  }
}

function serviceItemOrder(domain: string) {
  const index = fallbackItems.findIndex((item) => item.agent_domain === domain)
  return index >= 0 ? index + 1 : fallbackItems.length + 1
}

function fallbackDirectory(domain: string) {
  return fallbackItems.find((item) => item.agent_domain === domain)?.work_doc_directory || '客户/'
}

function parseList(value: string) {
  return value
    .split(/[\n,，]/)
    .map((item) => item.trim())
    .filter(Boolean)
}

function formatList(value: string[]) {
  return value.filter(Boolean).join('\n')
}

function formatJSON(value: Record<string, unknown> | undefined) {
  return JSON.stringify(value || {}, null, 2)
}

function parseJSONMap(value: string, label: string): Record<string, unknown> | null {
  const text = value.trim()
  if (!text) return {}
  try {
    const parsed = JSON.parse(text)
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>
    }
  } catch {
    // handled below
  }
  MessagePlugin.warning(`${label} 必须是 JSON 对象`)
  return null
}

function serviceIcon(domain: string) {
  const icons: Record<string, string> = {
    memory_router: 'setting-1',
    lead_intake: 'user-add',
    sales_consulting: 'chat',
    customer_service: 'service',
    schedule_coordination: 'calendar',
    after_sale_risk: 'error-circle',
    daily_review: 'file',
  }
  return icons[domain] || 'setting-1'
}

function serviceCategory(domain: string) {
  const categories: Record<string, string> = {
    memory_router: '基础识别',
    lead_intake: '招生前置',
    sales_consulting: '招生沟通',
    customer_service: '服务跟进',
    schedule_coordination: '教务协同',
    after_sale_risk: '风险闭环',
    daily_review: '经营复盘',
  }
  return categories[domain] || '服务能力'
}

function serviceInputSource(domain: string) {
  const sources: Record<string, string> = {
    memory_router: '成员分身、记忆内容',
    daily_review: '服务提醒、处理状态',
  }
  return sources[domain] || '成员分身、客户服务记忆'
}

function serviceOutput(domain: string) {
  const outputs: Record<string, string> = {
    memory_router: '服务场景和能力匹配结果',
    lead_intake: '线索草稿和缺失信息',
    sales_consulting: '沟通话术和下一步建议',
    customer_service: '客户摘要和跟进事项',
    schedule_coordination: '待确认排课安排',
    after_sale_risk: '风险处理建议和闭环事项',
    daily_review: '服务日报和知识补齐建议',
  }
  return outputs[domain] || '服务提醒和处理建议'
}

onMounted(() => {
  void loadServiceItems()
  void loadWorkProfiles()
})
</script>

<style scoped>
.service-config-page {
  display: grid;
  gap: 16px;
}

.service-config-page__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.service-config-page__header h2 {
  margin: 0 0 6px;
  font-size: 22px;
  line-height: 1.35;
  color: var(--td-text-color-primary);
}

.service-config-page__header p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.service-config-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.service-config-summary article {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 14px 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-config-summary span,
.service-config-summary em {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.4;
  font-style: normal;
}

.service-config-summary strong {
  color: var(--td-text-color-primary);
  font-size: 18px;
  line-height: 1.35;
}

.service-config-panel {
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.panel-title {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 14px;
}

.panel-title span {
  display: grid;
  gap: 4px;
}

.panel-title strong {
  color: var(--td-text-color-primary);
  font-size: 16px;
  line-height: 1.4;
}

.panel-title em {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
  font-style: normal;
}

.service-config-alert {
  margin-bottom: 14px;
}

.service-item-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.service-item-card {
  display: grid;
  gap: 14px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-page);
}

.service-item-card__head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
}

.service-item-card__actions {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: flex-end;
}

.service-item-card__icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.service-item-card__head span:not(.service-item-card__icon) {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.service-item-card__head strong {
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 1.35;
}

.service-item-card__head em {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.35;
  font-style: normal;
}

.service-item-card p {
  min-height: 42px;
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.service-item-card dl {
  display: grid;
  gap: 8px;
  margin: 0;
}

.service-item-card dl div {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}

.service-item-card dt,
.service-item-card dd {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.service-item-card dt {
  color: var(--td-text-color-placeholder);
}

.service-item-card dd {
  color: var(--td-text-color-primary);
}

.agent-edit-dialog {
  display: grid;
  gap: 16px;
}

.agent-edit-summary {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 12px;
  border: 1px solid var(--td-border-level-1-color);
  border-radius: 8px;
  background: var(--td-bg-color-page);
}

.agent-edit-summary div {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.agent-edit-summary strong {
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 1.4;
}

.agent-edit-summary p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.agent-edit-row {
  display: grid;
  grid-template-columns: minmax(120px, 180px) minmax(0, 1fr);
  gap: 12px;
}

.agent-edit-empty {
  padding: 14px 16px;
  border: 1px dashed var(--td-border-level-2-color);
  border-radius: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.6;
  background: var(--td-bg-color-page);
}

@media (max-width: 900px) {
  .service-config-page__header {
    display: grid;
  }

  .service-config-summary,
  .service-item-grid,
  .agent-edit-row {
    grid-template-columns: 1fr;
  }
}
</style>
