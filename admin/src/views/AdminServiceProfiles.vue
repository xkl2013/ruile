<template>
  <section class="service-template-page">
    <div class="service-template-page__header">
      <div>
        <div class="service-template-eyebrow">服务交付 · 场景能力</div>
        <h2>服务模板管理</h2>
        <p>维护服务场景、工作指令和运行能力。Web 端创建服务时只填写名称和描述，已发布模板负责提供默认配置。</p>
      </div>
      <div class="service-template-page__actions">
        <t-button variant="outline" :loading="loading" @click="loadTemplates">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <t-button theme="primary" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          新建服务模板
        </t-button>
      </div>
    </div>

    <div class="service-template-summary">
      <article>
        <span>模板总数</span>
        <strong>{{ templates.length }}</strong>
        <em>当前工作区</em>
      </article>
      <article>
        <span>已发布</span>
        <strong>{{ publishedCount }}</strong>
        <em>Web 可使用</em>
      </article>
      <article>
        <span>待处理</span>
        <strong>{{ draftCount + disabledCount }}</strong>
        <em>草稿或已停用</em>
      </article>
    </div>

    <section class="service-template-panel">
      <div class="service-template-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索模板名称、Key 或说明">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" style="width: 150px">
          <t-option value="all" label="全部状态" />
          <t-option value="draft" label="草稿" />
          <t-option value="published" label="已发布" />
          <t-option value="disabled" label="已停用" />
        </t-select>
      </div>

      <div v-if="loading && !templates.length" class="service-template-state">
        <t-loading size="small" />
        <span>正在加载服务模板...</span>
      </div>
      <div v-else-if="!filteredTemplates.length" class="service-template-state">
        <t-icon name="service" />
        <span>{{ templates.length ? '没有符合条件的服务模板' : '暂无服务模板' }}</span>
        <t-button v-if="!templates.length" theme="primary" variant="outline" size="small" @click="openCreate">
          新建第一个模板
        </t-button>
      </div>
      <div v-else class="service-template-list">
        <article v-for="template in filteredTemplates" :key="template.id || template.key" class="service-template-row">
          <div class="service-template-row__icon">
            <t-icon name="service" />
          </div>
          <div class="service-template-row__main">
            <div class="service-template-row__title">
              <strong>{{ template.name }}</strong>
              <code>{{ template.key }}</code>
              <t-tag :theme="statusTheme(template.status)" variant="light" size="small">
                {{ statusLabel(template.status) }}
              </t-tag>
            </div>
            <p>{{ template.description || '未填写模板说明' }}</p>
            <div class="service-template-row__meta">
              <span>{{ spaceTypeLabel(template.blueprint?.proposed_space_type) }}</span>
              <span>{{ template.blueprint?.expert_suggestions?.length || 0 }} 个专家</span>
              <span>{{ template.blueprint?.knowledge_base_ids?.length || 0 }} 个知识库</span>
              <span>{{ template.blueprint?.profile_schema?.length || 0 }} 条重点记录</span>
              <span>版本 {{ template.version }}</span>
              <span>更新于 {{ formatDate(template.updated_at) }}</span>
            </div>
          </div>
          <div class="service-template-row__actions">
            <t-button variant="text" size="small" @click="openEdit(template)">
              <template #icon><t-icon name="edit" /></template>
              编辑
            </t-button>
            <t-button
              v-if="template.status !== 'published'"
              theme="primary"
              variant="outline"
              size="small"
              :loading="busyKey === template.key"
              @click="publish(template)"
            >
              发布
            </t-button>
            <t-button
              v-else
              theme="warning"
              variant="outline"
              size="small"
              :loading="busyKey === template.key"
              @click="disable(template)"
            >
              停用
            </t-button>
          </div>
        </article>
      </div>
    </section>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑服务模板' : '新建服务模板'"
      width="1040px"
      :confirm-btn="{ content: '保存草稿', loading: saving }"
      destroy-on-close
      @confirm="saveTemplate"
    >
      <t-form :data="form" label-align="top" class="service-template-form">
        <section class="service-template-form-section">
          <div class="service-template-form-section__header">
            <div>
              <strong>基本信息</strong>
              <span>定义这个模板服务什么场景，以及 Web 创建时会显示的能力说明。</span>
            </div>
          </div>
          <div class="service-template-form-grid">
            <t-form-item label="模板名称" required>
              <t-input v-model="form.name" placeholder="例如 招生跟进服务" />
            </t-form-item>
          </div>
          <t-form-item label="服务场景">
            <t-select v-model="form.space_type">
              <t-option value="customer_service" label="家长与学员" />
              <t-option value="operations" label="园所日常运营" />
              <t-option value="research" label="教研与备课" />
            </t-select>
          </t-form-item>
          <t-form-item label="模板说明">
            <t-textarea
              v-model="form.description"
              :maxlength="500"
              :autosize="{ minRows: 2, maxRows: 4 }"
              placeholder="说明这个服务模板解决什么问题，以及适合什么工作场景"
            />
          </t-form-item>
          <t-form-item label="工作指令">
            <t-textarea
              v-model="form.instruction"
              :maxlength="12000"
              :autosize="{ minRows: 6, maxRows: 14 }"
              placeholder="写清楚服务目标、关注信息、事实边界和下一步工作方式"
            />
          </t-form-item>
        </section>

        <section class="service-template-form-section">
          <div class="service-template-form-section__header">
            <div>
              <strong>运行能力</strong>
              <span>这些配置会随模板带入服务实例，Web 端不再让用户逐项选择。</span>
            </div>
          </div>
          <div class="service-template-form-subheader">
            <strong>专家建议</strong>
            <t-button size="small" variant="outline" @click="addExpert">
              <template #icon><t-icon name="add" /></template>
              添加专家
            </t-button>
          </div>
          <div v-if="!form.expert_suggestions.length" class="service-template-inline-empty">
            尚未配置专家。自动启用模板前至少添加一个专家。
          </div>
          <div v-for="(expert, index) in form.expert_suggestions" :key="`expert-${index}`" class="service-template-repeater">
            <t-input v-model="expert.expert_ref" placeholder="专家引用，例如 enrollment-advisor" />
            <t-input v-model="expert.expert_name" placeholder="专家名称" />
            <t-input v-model="expert.reason" placeholder="为什么适合这个场景" />
            <t-button
              variant="text"
              shape="square"
              aria-label="删除专家"
              @click="removeExpert(index)"
            >
              <t-icon name="delete" />
            </t-button>
          </div>

          <div class="service-template-form-grid">
            <t-form-item label="知识库">
              <t-tag-input
                v-model="form.knowledge_base_ids"
                clearable
                placeholder="输入知识库 ID 后回车"
              />
              <small>填写模板运行时允许读取的知识库 ID。</small>
            </t-form-item>
            <t-form-item label="Skill">
              <t-tag-input
                v-model="form.selected_skills"
                clearable
                placeholder="输入 Skill 标识后回车"
              />
              <small>填写需要随服务启用的 Skill 标识。</small>
            </t-form-item>
          </div>
        </section>

      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  createAdminServiceTemplate,
  disableAdminServiceTemplate,
  listAdminServiceTemplates,
  publishAdminServiceTemplate,
  updateAdminServiceTemplate,
  type AdminServiceTemplate,
  type AdminServiceTemplateInput,
  type ServiceTemplateExpertSuggestion,
  type ServiceTemplateSpaceType,
} from '@admin/api/service-template'

interface ServiceTemplateForm {
  name: string
  description: string
  space_type: ServiceTemplateSpaceType
  instruction: string
  expert_suggestions: Array<ServiceTemplateExpertSuggestion>
  knowledge_base_ids: string[]
  selected_skills: string[]
}

const templates = ref<AdminServiceTemplate[]>([])
const loading = ref(false)
const saving = ref(false)
const busyKey = ref('')
const keyword = ref('')
const statusFilter = ref<'all' | AdminServiceTemplate['status']>('all')
const editorVisible = ref(false)
const editing = ref<AdminServiceTemplate | null>(null)

const publishedCount = computed(() => templates.value.filter((template) => template.status === 'published').length)
const draftCount = computed(() => templates.value.filter((template) => template.status === 'draft').length)
const disabledCount = computed(() => templates.value.filter((template) => template.status === 'disabled').length)
const filteredTemplates = computed(() => {
  const needle = keyword.value.trim().toLowerCase()
  return templates.value.filter((template) => {
    const matchesStatus = statusFilter.value === 'all' || template.status === statusFilter.value
    const matchesKeyword = !needle
      || [template.key, template.name, template.description].some((value) => value?.toLowerCase().includes(needle))
    return matchesStatus && matchesKeyword
  })
})

function createEmptyForm(): ServiceTemplateForm {
  return {
    name: '',
    description: '',
    space_type: 'customer_service',
    instruction: '',
    expert_suggestions: [],
    knowledge_base_ids: [],
    selected_skills: [],
  }
}

const form = reactive<ServiceTemplateForm>(createEmptyForm())

async function loadTemplates() {
  if (loading.value) return
  loading.value = true
  try {
    const response = await listAdminServiceTemplates()
    templates.value = response?.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '服务模板加载失败')
    templates.value = []
  } finally {
    loading.value = false
  }
}

function applyTemplateToForm(template?: AdminServiceTemplate) {
  const next = createEmptyForm()
  if (template) {
    next.name = template.name
    next.description = template.description || ''
    next.space_type = template.blueprint?.proposed_space_type || 'customer_service'
    next.instruction = template.blueprint?.source_instruction || ''
    next.expert_suggestions = (template.blueprint?.expert_suggestions || []).map((expert) => ({ ...expert }))
    next.knowledge_base_ids = [...(template.blueprint?.knowledge_base_ids || [])]
    next.selected_skills = [...(template.blueprint?.selected_skills || [])]
  }
  Object.assign(form, next)
}

function openCreate() {
  editing.value = null
  applyTemplateToForm()
  editorVisible.value = true
}

function openEdit(template: AdminServiceTemplate) {
  editing.value = template
  applyTemplateToForm(template)
  editorVisible.value = true
}

function addExpert() {
  form.expert_suggestions.push({ expert_ref: '', expert_name: '', reason: '' })
}

function removeExpert(index: number) {
  form.expert_suggestions.splice(index, 1)
}

function toPayload(): AdminServiceTemplateInput {
  return {
    name: form.name.trim(),
    description: form.description.trim(),
    space_type: form.space_type,
    instruction: form.instruction.trim(),
    subject_policy: {
      required: true,
      allowed_types: ['member_family'],
      allow_hierarchy: false,
    },
    expert_suggestions: form.expert_suggestions
      .map((expert) => ({
        expert_ref: expert.expert_ref.trim(),
        expert_name: expert.expert_name.trim(),
        reason: expert.reason?.trim() || '',
      }))
      .filter((expert) => expert.expert_ref || expert.expert_name),
    knowledge_base_ids: form.knowledge_base_ids.map((value) => value.trim()).filter(Boolean),
    selected_skills: form.selected_skills.map((value) => value.trim()).filter(Boolean),
    match_rules: {
      min_confidence_pct: 0,
    },
  }
}

async function saveTemplate() {
  if (!form.name.trim() || !form.instruction.trim()) {
    MessagePlugin.warning('请填写模板名称和工作指令')
    return
  }
  saving.value = true
  try {
    const payload = toPayload()
    if (editing.value) {
      await updateAdminServiceTemplate(editing.value.key, payload)
      MessagePlugin.success('服务模板草稿已保存')
    } else {
      await createAdminServiceTemplate(payload)
      MessagePlugin.success('服务模板草稿已创建')
    }
    editorVisible.value = false
    await loadTemplates()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '服务模板保存失败')
  } finally {
    saving.value = false
  }
}

async function publish(template: AdminServiceTemplate) {
  busyKey.value = template.key
  try {
    await publishAdminServiceTemplate(template.key)
    MessagePlugin.success('服务模板已发布')
    await loadTemplates()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '服务模板发布失败')
  } finally {
    busyKey.value = ''
  }
}

function disable(template: AdminServiceTemplate) {
  const instance = DialogPlugin.confirm({
    header: '停用服务模板',
    body: `停用后，Web 端不能再使用“${template.name}”创建新的服务实例。`,
    confirmBtn: '停用',
    onConfirm: async () => {
      busyKey.value = template.key
      try {
        await disableAdminServiceTemplate(template.key)
        MessagePlugin.success('服务模板已停用')
        await loadTemplates()
        instance.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '服务模板停用失败')
      } finally {
        busyKey.value = ''
      }
    },
  })
}

function statusLabel(value: AdminServiceTemplate['status']) {
  return ({ draft: '草稿', published: '已发布', disabled: '已停用', archived: '已归档' } as Record<string, string>)[value] || value
}

function statusTheme(value: AdminServiceTemplate['status']) {
  if (value === 'published') return 'success'
  if (value === 'disabled') return 'warning'
  if (value === 'archived') return 'default'
  return 'primary'
}

function spaceTypeLabel(value?: ServiceTemplateSpaceType) {
  return ({
    customer_service: '家长与学员',
    operations: '园所日常运营',
    research: '教研与备课',
  } as Record<string, string>)[value || ''] || '未设置场景'
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

onMounted(() => {
  void loadTemplates()
})
</script>

<style scoped>
.service-template-page {
  display: grid;
  gap: 18px;
  width: min(100%, 1320px);
}

.service-template-page__header,
.service-template-panel,
.service-template-summary article {
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.service-template-page__header {
  display: flex;
  justify-content: space-between;
  gap: 20px;
  align-items: flex-start;
  padding: 20px;
}

.service-template-eyebrow {
  margin-bottom: 5px;
  color: var(--admin-text-secondary);
  font-size: 12px;
  letter-spacing: 0;
}

.service-template-page__header h2 {
  margin: 0 0 6px;
  color: var(--admin-text);
  font-size: 24px;
  font-weight: 650;
  line-height: 1.35;
}

.service-template-page__header p {
  max-width: 820px;
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 14px;
  line-height: 1.65;
}

.service-template-page__actions {
  display: flex;
  flex: 0 0 auto;
  gap: 8px;
}

.service-template-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.service-template-summary article {
  display: grid;
  gap: 5px;
  padding: 15px 16px;
}

.service-template-summary span,
.service-template-summary em {
  color: var(--admin-text-secondary);
  font-size: 12px;
  font-style: normal;
}

.service-template-summary strong {
  color: var(--admin-text);
  font-size: 21px;
  font-weight: 650;
}

.service-template-panel {
  overflow: hidden;
}

.service-template-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 14px 16px;
  border-bottom: 1px solid var(--admin-border);
}

.service-template-toolbar .t-input {
  width: min(100%, 420px);
}

.service-template-state {
  display: flex;
  min-height: 220px;
  gap: 10px;
  align-items: center;
  justify-content: center;
  color: var(--admin-text-secondary);
  font-size: 14px;
}

.service-template-list {
  display: grid;
}

.service-template-row {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) auto;
  gap: 14px;
  align-items: start;
  padding: 18px 20px;
  border-bottom: 1px solid var(--admin-border);
}

.service-template-row:last-child {
  border-bottom: 0;
}

.service-template-row__icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  color: var(--admin-primary);
  background: var(--admin-surface-subtle);
  font-size: 19px;
}

.service-template-row__title {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.service-template-row__title strong {
  color: var(--admin-text);
  font-size: 16px;
}

.service-template-row__title code {
  color: var(--admin-text-secondary);
  font-size: 12px;
}

.service-template-row__main p {
  margin: 7px 0 10px;
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.55;
}

.service-template-row__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 14px;
  color: var(--admin-text-secondary);
  font-size: 12px;
}

.service-template-row__actions {
  display: flex;
  gap: 4px;
  align-items: center;
}

.service-template-form {
  display: grid;
  gap: 16px;
  max-height: 70vh;
  overflow: auto;
  padding-right: 4px;
}

.service-template-form-section {
  padding: 16px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-subtle);
}

.service-template-form-section__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.service-template-form-section__header div {
  display: grid;
  gap: 4px;
}

.service-template-form-section__header strong {
  color: var(--admin-text);
  font-size: 14px;
}

.service-template-form-section__header span {
  color: var(--admin-text-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.service-template-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.service-template-form :deep(.t-form__item) {
  margin-bottom: 12px;
}

.service-template-form small {
  display: block;
  margin-top: 5px;
  color: var(--admin-text-secondary);
  font-size: 12px;
  line-height: 1.45;
}

.service-template-repeater {
  display: grid;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
}

.service-template-repeater {
  grid-template-columns: minmax(150px, 0.8fr) minmax(140px, 0.7fr) minmax(220px, 1fr) 32px;
}

@media (max-width: 860px) {
  .service-template-page__header {
    flex-direction: column;
  }

  .service-template-page__actions {
    width: 100%;
  }

  .service-template-page__actions .t-button {
    flex: 1;
  }

  .service-template-row {
    grid-template-columns: 38px minmax(0, 1fr);
  }

  .service-template-row__actions {
    grid-column: 2;
  }

  .service-template-form-grid {
    grid-template-columns: 1fr;
  }

  .service-template-repeater {
    grid-template-columns: 1fr 1fr;
  }

  .service-template-repeater .t-button {
    justify-self: start;
  }
}

@media (max-width: 560px) {
  .service-template-summary {
    grid-template-columns: 1fr;
  }

  .service-template-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .service-template-toolbar .t-input {
    width: 100%;
  }

  .service-template-repeater {
    grid-template-columns: 1fr;
  }
}
</style>
