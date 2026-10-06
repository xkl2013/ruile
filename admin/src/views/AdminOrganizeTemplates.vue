<template>
  <section class="organize-template-page">
    <div class="organize-template-page__header">
      <div>
        <div class="organize-template-eyebrow">整理模块 · 平台治理</div>
        <h2>整理模板</h2>
        <p>维护整理场景的指令、Markdown 报告预设和发布版本，所有用户使用已发布模板。</p>
      </div>
      <div class="organize-template-page__actions">
        <t-button variant="outline" :loading="loading" @click="loadTemplates">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <t-button theme="primary" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          新建模板
        </t-button>
      </div>
    </div>

    <section class="organize-template-panel">
      <div class="organize-template-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索模板名称、Key 或说明" @enter="loadTemplates">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-input v-model="scene" clearable placeholder="按场景筛选" @enter="loadTemplates" />
        <t-select v-model="status" style="width: 140px" @change="loadTemplates">
          <t-option value="all" label="全部状态" />
          <t-option value="draft" label="草稿" />
          <t-option value="enabled" label="已发布" />
          <t-option value="disabled" label="已停用" />
        </t-select>
        <t-button variant="outline" @click="loadTemplates">查询</t-button>
      </div>

      <div v-if="loading && !templates.length" class="organize-template-state">
        <t-loading size="small" /> 加载中
      </div>
      <div v-else-if="!templates.length" class="organize-template-state">
        <t-icon name="file-paste" />
        <span>暂无整理模板</span>
      </div>
      <div v-else class="organize-template-list">
        <article v-for="template in templates" :key="template.id" class="organize-template-row">
          <div class="organize-template-row__icon">
            <t-icon :name="template.icon || 'file-paste'" />
          </div>
          <div class="organize-template-row__main">
            <div class="organize-template-row__title">
              <strong>{{ template.name }}</strong>
              <code>{{ template.key }}</code>
              <t-tag :theme="statusTheme(template.status)" variant="light" size="small">
                {{ statusLabel(template.status) }}
              </t-tag>
            </div>
            <p>{{ template.description || '未填写模板说明' }}</p>
            <div class="organize-template-row__meta">
              <span>{{ template.scene || '未分类场景' }}</span>
              <span>{{ template.markdown_template ? '已配置 Markdown 预设' : '使用系统默认预设' }}</span>
              <span>当前版本 {{ template.published_version || '未发布' }}</span>
              <span>更新于 {{ formatDate(template.updated_at) }}</span>
            </div>
          </div>
          <div class="organize-template-row__actions">
            <t-button variant="text" size="small" @click="openEdit(template)">
              <template #icon><t-icon name="edit" /></template>
              编辑
            </t-button>
            <t-button variant="text" size="small" @click="preview(template)">
              <template #icon><t-icon name="view-module" /></template>
              试跑
            </t-button>
            <t-button variant="text" size="small" @click="openVersions(template)">
              <template #icon><t-icon name="history" /></template>
              版本
            </t-button>
            <t-button
              v-if="template.status !== 'enabled'"
              theme="primary"
              variant="outline"
              size="small"
              :loading="busyKey === template.key"
              @click="publish(template)"
            >发布</t-button>
            <t-button
              v-else
              theme="warning"
              variant="outline"
              size="small"
              :loading="busyKey === template.key"
              @click="disable(template)"
            >停用</t-button>
          </div>
        </article>
      </div>
      <div v-if="total > pageSize" class="organize-template-pagination">
        <t-pagination
          v-model="page"
          :page-size="pageSize"
          :total="total"
          size="small"
          show-page-number
          @change="loadTemplates"
        />
      </div>
    </section>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑整理模板' : '新建整理模板'"
      width="820px"
      :confirm-btn="{ content: '保存', loading: saving }"
      destroy-on-close
      @confirm="saveTemplate"
    >
      <t-form :data="form" label-align="top">
        <div class="organize-template-form-grid">
          <t-form-item label="模板 Key">
            <t-input v-model="form.key" :disabled="Boolean(editing)" placeholder="例如 teacher_research" />
          </t-form-item>
          <t-form-item label="模板名称">
            <t-input v-model="form.name" placeholder="用户看到的模板名称" />
          </t-form-item>
        </div>
        <div class="organize-template-form-grid">
          <t-form-item label="整理场景">
            <t-input v-model="form.scene" placeholder="例如 教师成长、招生增长" />
          </t-form-item>
          <t-form-item label="输出名称">
            <t-input v-model="form.output_label" placeholder="例如 提炼清单" />
          </t-form-item>
        </div>
        <t-form-item label="模板说明">
          <t-textarea v-model="form.description" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="说明该模板解决什么整理问题" />
        </t-form-item>
        <t-form-item label="整理指令">
          <t-textarea
            v-model="form.default_instruction"
            :autosize="{ minRows: 5, maxRows: 10 }"
            placeholder="写清楚整理目标、事实边界、输出结构和引用要求"
          />
        </t-form-item>
        <t-form-item label="Markdown 报告预设">
          <t-textarea
            v-model="form.markdown_template"
            :autosize="{ minRows: 12, maxRows: 24 }"
            placeholder="# {{title}}\n\n> {{summary}}\n\n## 核心发现\n\n{{section_1}}\n\n## 下一步\n\n- [ ] {{section_2}}"
          />
          <p class="organize-template-form-help">使用标题、引用、列表和待办定义报告骨架，系统会自动约束模型保留章节结构。</p>
        </t-form-item>
        <div class="organize-template-form-grid">
          <t-form-item label="图标">
            <t-input v-model="form.icon" placeholder="file-paste" />
          </t-form-item>
          <t-form-item label="排序">
            <t-input-number v-model="form.sort_order" :min="0" :max="9999" theme="normal" />
          </t-form-item>
        </div>
        <t-form-item label="变更说明">
          <t-input v-model="form.change_note" placeholder="选填，说明本次编辑或发布目的" />
        </t-form-item>
      </t-form>
    </t-dialog>

    <t-dialog v-model:visible="previewVisible" header="模板试跑" width="760px" :footer="false" destroy-on-close>
      <div v-if="previewData" class="organize-template-preview">
        <div class="organize-template-preview__meta">
          <span>模板：{{ previewData.template_key }}</span>
          <span>版本：{{ previewData.version || '草稿' }}</span>
        </div>
        <t-alert v-if="previewData.errors?.length" theme="warning" :message="previewData.errors.join('；')" />
        <pre>{{ previewData.prompt || '暂无可渲染指令' }}</pre>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="versionsVisible" :header="`${selectedTemplate?.name || ''} · 版本记录`" width="820px" :footer="false" destroy-on-close>
      <div v-if="versionsLoading" class="organize-template-state"><t-loading size="small" /> 加载中</div>
      <div v-else-if="!versions.length" class="organize-template-state">暂无发布版本</div>
      <div v-else class="organize-template-version-list">
        <div v-for="version in versions" :key="version.id" class="organize-template-version-row">
          <div>
            <strong>{{ version.version }}</strong>
            <span>{{ version.change_note || '未填写变更说明' }}</span>
            <small>{{ formatDate(version.created_at) }} · {{ version.created_by || '系统' }}</small>
          </div>
          <t-button
            v-if="version.version !== selectedTemplate?.published_version"
            variant="outline"
            size="small"
            :loading="busyKey === `${selectedTemplate?.key}:${version.version}`"
            @click="rollback(version)"
          >回滚并发布</t-button>
          <t-tag v-else theme="success" variant="light" size="small">当前版本</t-tag>
        </div>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  createAdminOrganizeTemplate,
  disableAdminOrganizeTemplate,
  listAdminOrganizeTemplateVersions,
  listAdminOrganizeTemplates,
  previewAdminOrganizeTemplate,
  publishAdminOrganizeTemplate,
  rollbackAdminOrganizeTemplate,
  updateAdminOrganizeTemplate,
  type AdminOrganizeTemplate,
  type AdminOrganizeTemplateVersion,
  type OrganizeTemplateStatus,
} from '@admin/api/organize'

const templates = ref<AdminOrganizeTemplate[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const scene = ref('')
const status = ref<OrganizeTemplateStatus | 'all'>('all')
const loading = ref(false)
const saving = ref(false)
const busyKey = ref('')
const editorVisible = ref(false)
const editing = ref<AdminOrganizeTemplate | null>(null)
const previewVisible = ref(false)
const previewData = ref<{ template_key: string; version: string; prompt: string; markdown_template?: string; spec: Record<string, any>; errors: string[] } | null>(null)
const versionsVisible = ref(false)
const versionsLoading = ref(false)
const selectedTemplate = ref<AdminOrganizeTemplate | null>(null)
const versions = ref<AdminOrganizeTemplateVersion[]>([])

const form = reactive({
  key: '',
  name: '',
  scene: '',
  description: '',
  output_label: '',
  icon: 'file-paste',
  default_instruction: '',
  markdown_template: '',
  sort_order: 0,
  change_note: '',
})

function statusLabel(value: OrganizeTemplateStatus) {
  return ({ draft: '草稿', enabled: '已发布', disabled: '已停用' } as Record<OrganizeTemplateStatus, string>)[value] || value
}

function statusTheme(value: OrganizeTemplateStatus) {
  if (value === 'enabled') return 'success'
  if (value === 'disabled') return 'warning'
  return 'default'
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

async function loadTemplates() {
  loading.value = true
  try {
    const response = await listAdminOrganizeTemplates({
      keyword: keyword.value.trim(),
      scene: scene.value.trim(),
      status: status.value,
      page: page.value,
      pageSize,
    })
    templates.value = response.data?.items || []
    total.value = response.data?.total || 0
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载整理模板失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    key: '',
    name: '',
    scene: '',
    description: '',
    output_label: '',
    icon: 'file-paste',
    default_instruction: '',
    markdown_template: defaultMarkdownTemplate(),
    sort_order: 0,
    change_note: '',
  })
}

function openCreate() {
  editing.value = null
  resetForm()
  editorVisible.value = true
}

function openEdit(template: AdminOrganizeTemplate) {
  editing.value = template
  Object.assign(form, {
    key: template.key,
    name: template.name,
    scene: template.scene || '',
    description: template.description || '',
    output_label: template.output_label || '',
    icon: template.icon || 'file-paste',
    default_instruction: template.default_instruction || '',
    markdown_template: template.markdown_template || defaultMarkdownTemplate(),
    sort_order: template.sort_order || 0,
    change_note: '',
  })
  editorVisible.value = true
}

async function saveTemplate() {
  if (!form.key.trim() || !form.name.trim()) {
    MessagePlugin.warning('请填写模板 Key 和模板名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      key: form.key.trim(),
      name: form.name.trim(),
      scene: form.scene.trim(),
      description: form.description.trim(),
      output_label: form.output_label.trim(),
      icon: form.icon.trim(),
      default_instruction: form.default_instruction.trim(),
      markdown_template: form.markdown_template.trim(),
      spec: editing.value?.spec || {},
      sort_order: form.sort_order || 0,
      change_note: form.change_note.trim(),
    }
    if (editing.value) {
      await updateAdminOrganizeTemplate(editing.value.key, payload)
    } else {
      await createAdminOrganizeTemplate(payload)
    }
    MessagePlugin.success(editing.value ? '整理模板已保存' : '整理模板已创建')
    editorVisible.value = false
    await loadTemplates()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存整理模板失败')
  } finally {
    saving.value = false
  }
}

async function preview(template: AdminOrganizeTemplate) {
  try {
    const response = await previewAdminOrganizeTemplate(template.key, { memory_count: 3 })
    previewData.value = response.data
    previewVisible.value = true
  } catch (error: any) {
    MessagePlugin.error(error?.message || '模板试跑失败')
  }
}

async function publish(template: AdminOrganizeTemplate) {
  busyKey.value = template.key
  try {
    await publishAdminOrganizeTemplate(template.key, 'Admin 发布模板')
    MessagePlugin.success('整理模板已发布')
    await loadTemplates()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发布整理模板失败')
  } finally {
    busyKey.value = ''
  }
}

async function disable(template: AdminOrganizeTemplate) {
  const instance = DialogPlugin.confirm({
    header: '停用整理模板',
    body: `停用后，用户不能再用“${template.name}”创建新的整理任务。`,
    confirmBtn: '停用',
    onConfirm: async () => {
      busyKey.value = template.key
      try {
        await disableAdminOrganizeTemplate(template.key)
        MessagePlugin.success('整理模板已停用')
        await loadTemplates()
        instance.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '停用整理模板失败')
      } finally {
        busyKey.value = ''
      }
    },
  })
}

async function openVersions(template: AdminOrganizeTemplate) {
  selectedTemplate.value = template
  versionsVisible.value = true
  versionsLoading.value = true
  try {
    const response = await listAdminOrganizeTemplateVersions(template.key)
    versions.value = response.data?.items || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载模板版本失败')
    versions.value = []
  } finally {
    versionsLoading.value = false
  }
}

async function rollback(version: AdminOrganizeTemplateVersion) {
  if (!selectedTemplate.value) return
  busyKey.value = `${selectedTemplate.value.key}:${version.version}`
  try {
    await rollbackAdminOrganizeTemplate(selectedTemplate.value.key, version.version, `回滚到 ${version.version}`)
    MessagePlugin.success(`已回滚到 ${version.version} 并发布新版本`)
    versionsVisible.value = false
    await loadTemplates()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '模板回滚失败')
  } finally {
    busyKey.value = ''
  }
}

onMounted(loadTemplates)

function defaultMarkdownTemplate() {
  return '# {{title}}\n\n> {{summary}}\n\n## 核心发现\n\n{{section_1}}\n\n## 下一步\n\n- [ ] {{section_2}}\n\n### 依据\n\n{{citations}}\n\n**标签：** {{tags}}'
}
</script>

<style scoped>
.organize-template-page { color: var(--td-text-color-primary); }
.organize-template-page__header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 24px; }
.organize-template-page__header h2 { margin: 6px 0 8px; font-size: 24px; }
.organize-template-page__header p { margin: 0; color: var(--td-text-color-secondary); }
.organize-template-eyebrow { color: var(--td-brand-color); font-size: 12px; font-weight: 600; letter-spacing: .08em; }
.organize-template-page__actions { display: flex; gap: 10px; }
.organize-template-panel { background: var(--td-bg-color-container); border: 1px solid var(--td-component-border); border-radius: 8px; overflow: hidden; }
.organize-template-toolbar { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.organize-template-toolbar .t-input { width: 280px; }
.organize-template-state { display: flex; justify-content: center; align-items: center; gap: 8px; min-height: 180px; color: var(--td-text-color-secondary); }
.organize-template-row { display: flex; gap: 16px; padding: 18px 20px; border-bottom: 1px solid var(--td-component-border); }
.organize-template-row:last-child { border-bottom: 0; }
.organize-template-row__icon { width: 36px; height: 36px; display: grid; place-items: center; color: var(--td-brand-color); background: var(--td-brand-color-light); border-radius: 8px; flex: 0 0 auto; }
.organize-template-row__main { min-width: 0; flex: 1; }
.organize-template-row__title { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.organize-template-row__title code { color: var(--td-text-color-secondary); font-size: 12px; }
.organize-template-row__main p { margin: 8px 0; color: var(--td-text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.organize-template-row__meta { display: flex; flex-wrap: wrap; gap: 14px; color: var(--td-text-color-placeholder); font-size: 12px; }
.organize-template-row__actions { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; justify-content: flex-end; }
.organize-template-pagination { padding: 14px 18px; display: flex; justify-content: flex-end; border-top: 1px solid var(--td-component-border); }
.organize-template-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.organize-template-form-help { margin: 8px 0 0; color: var(--td-text-color-secondary); font-size: 12px; line-height: 1.6; }
.organize-template-preview__meta { display: flex; gap: 20px; color: var(--td-text-color-secondary); font-size: 13px; margin-bottom: 14px; }
.organize-template-preview pre { margin: 14px 0 0; padding: 16px; min-height: 180px; white-space: pre-wrap; background: var(--td-bg-color-secondarycontainer); border-radius: 6px; line-height: 1.7; }
.organize-template-version-list { display: grid; gap: 10px; }
.organize-template-version-row { display: flex; justify-content: space-between; align-items: center; gap: 16px; padding: 14px 0; border-bottom: 1px solid var(--td-component-border); }
.organize-template-version-row:last-child { border-bottom: 0; }
.organize-template-version-row strong { margin-right: 12px; }
.organize-template-version-row span, .organize-template-version-row small { display: block; color: var(--td-text-color-secondary); margin-top: 5px; }
@media (max-width: 800px) {
  .organize-template-page__header, .organize-template-toolbar { flex-direction: column; }
  .organize-template-toolbar .t-input, .organize-template-toolbar .t-select { width: 100% !important; }
  .organize-template-row { flex-wrap: wrap; }
  .organize-template-row__actions { width: 100%; justify-content: flex-start; }
  .organize-template-form-grid { grid-template-columns: 1fr; gap: 0; }
}
</style>
