<template>
  <section class="discover-category-page">
    <div class="discover-category-page__header">
      <div>
        <div class="discover-category-eyebrow">整理模块 · 内容治理</div>
        <h2>发现栏目</h2>
        <p>维护发现页的内容栏目。停用后不会出现在新内容选择器和发现页导航中。</p>
      </div>
      <div class="discover-category-page__actions">
        <t-button variant="outline" :loading="loading" @click="loadCategories">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <t-button theme="primary" @click="openCreate">
          <template #icon><t-icon name="add" /></template>
          新建栏目
        </t-button>
      </div>
    </div>

    <section class="discover-category-panel">
      <div class="discover-category-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索栏目 Key、名称或说明" @enter="loadCategories">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="status" style="width: 140px" @change="loadCategories">
          <t-option value="all" label="全部状态" />
          <t-option value="enabled" label="已启用" />
          <t-option value="disabled" label="已停用" />
        </t-select>
        <t-button variant="outline" @click="loadCategories">查询</t-button>
      </div>

      <div v-if="loading && !categories.length" class="discover-category-state">
        <t-loading size="small" /> 加载中
      </div>
      <div v-else-if="!categories.length" class="discover-category-state">
        <t-icon name="folder-open" />
        <span>暂无发现栏目</span>
      </div>
      <div v-else class="discover-category-list">
        <article v-for="category in categories" :key="category.id" class="discover-category-row">
          <div class="discover-category-order">{{ category.sort_order }}</div>
          <div class="discover-category-main">
            <div class="discover-category-title">
              <strong>{{ category.label }}</strong>
              <code>{{ category.key }}</code>
              <t-tag :theme="statusTheme(category.status)" variant="light" size="small">
                {{ statusLabel(category.status) }}
              </t-tag>
            </div>
            <p>{{ category.description || '未填写栏目说明' }}</p>
            <small>更新于 {{ formatDate(category.updated_at) }}</small>
          </div>
          <div class="discover-category-actions">
            <t-button variant="text" size="small" @click="openEdit(category)">
              <template #icon><t-icon name="edit" /></template>
              编辑
            </t-button>
            <t-button
              v-if="category.status === 'enabled'"
              theme="warning"
              variant="outline"
              size="small"
              :loading="busyKey === category.key"
              @click="disable(category)"
            >停用</t-button>
          </div>
        </article>
      </div>

      <div v-if="total > pageSize" class="discover-category-pagination">
        <t-pagination
          v-model="page"
          :page-size="pageSize"
          :total="total"
          size="small"
          show-page-number
          @change="loadCategories"
        />
      </div>
    </section>

    <t-dialog
      v-model:visible="editorVisible"
      :header="editing ? '编辑发现栏目' : '新建发现栏目'"
      width="620px"
      :confirm-btn="{ content: '保存', loading: saving }"
      destroy-on-close
      @confirm="saveCategory"
    >
      <t-form :data="form" label-align="top">
        <div class="discover-category-form-grid">
          <t-form-item label="栏目 Key">
            <t-input v-model="form.key" :disabled="Boolean(editing)" placeholder="例如 teacher_research" />
          </t-form-item>
          <t-form-item label="栏目名称">
            <t-input v-model="form.label" placeholder="用户看到的栏目名称" />
          </t-form-item>
        </div>
        <t-form-item label="栏目说明">
          <t-textarea v-model="form.description" :autosize="{ minRows: 2, maxRows: 5 }" placeholder="说明栏目适合展示的内容" />
        </t-form-item>
        <t-form-item label="排序">
          <t-input-number v-model="form.sort_order" :min="0" :max="9999" theme="normal" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  createAdminOrganizeDiscoverCategory,
  disableAdminOrganizeDiscoverCategory,
  listAdminOrganizeDiscoverCategories,
  updateAdminOrganizeDiscoverCategory,
  type AdminOrganizeDiscoverCategory,
  type AdminOrganizeDiscoverCategoryStatus,
} from '@admin/api/organize'

const categories = ref<AdminOrganizeDiscoverCategory[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const keyword = ref('')
const status = ref<AdminOrganizeDiscoverCategoryStatus | 'all'>('all')
const loading = ref(false)
const saving = ref(false)
const busyKey = ref('')
const editorVisible = ref(false)
const editing = ref<AdminOrganizeDiscoverCategory | null>(null)

const form = reactive({
  key: '',
  label: '',
  description: '',
  sort_order: 0,
})

function statusLabel(value: AdminOrganizeDiscoverCategoryStatus) {
  return value === 'enabled' ? '已启用' : '已停用'
}

function statusTheme(value: AdminOrganizeDiscoverCategoryStatus) {
  return value === 'enabled' ? 'success' : 'warning'
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  return new Date(value).toLocaleString('zh-CN', { hour12: false })
}

async function loadCategories() {
  loading.value = true
  try {
    const response = await listAdminOrganizeDiscoverCategories({
      keyword: keyword.value.trim(),
      status: status.value,
      page: page.value,
      pageSize,
    })
    categories.value = response.data?.items || []
    total.value = response.data?.total || 0
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载发现栏目失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, { key: '', label: '', description: '', sort_order: 0 })
}

function openCreate() {
  editing.value = null
  resetForm()
  editorVisible.value = true
}

function openEdit(category: AdminOrganizeDiscoverCategory) {
  editing.value = category
  Object.assign(form, {
    key: category.key,
    label: category.label,
    description: category.description || '',
    sort_order: category.sort_order || 0,
  })
  editorVisible.value = true
}

async function saveCategory() {
  if (!form.key.trim() || !form.label.trim()) {
    MessagePlugin.warning('请填写栏目 Key 和栏目名称')
    return
  }
  saving.value = true
  try {
    const payload = {
      key: form.key.trim(),
      label: form.label.trim(),
      description: form.description.trim(),
      sort_order: form.sort_order || 0,
    }
    if (editing.value) {
      await updateAdminOrganizeDiscoverCategory(editing.value.key, payload)
    } else {
      await createAdminOrganizeDiscoverCategory(payload)
    }
    MessagePlugin.success(editing.value ? '发现栏目已保存' : '发现栏目已创建')
    editorVisible.value = false
    await loadCategories()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存发现栏目失败')
  } finally {
    saving.value = false
  }
}

function disable(category: AdminOrganizeDiscoverCategory) {
  const dialog = DialogPlugin.confirm({
    header: '停用发现栏目',
    body: `停用后，用户不能再将新内容归入“${category.label}”，已发布内容仍会保留。`,
    confirmBtn: { content: '停用', theme: 'danger' },
    cancelBtn: '取消',
    onConfirm: async () => {
      busyKey.value = category.key
      try {
        await disableAdminOrganizeDiscoverCategory(category.key)
        MessagePlugin.success('发现栏目已停用')
        await loadCategories()
        dialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '停用发现栏目失败')
      } finally {
        busyKey.value = ''
      }
    },
  })
}

onMounted(loadCategories)
</script>

<style scoped>
.discover-category-page { color: var(--td-text-color-primary); }
.discover-category-page__header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; margin-bottom: 24px; }
.discover-category-page__header h2 { margin: 6px 0 8px; font-size: 24px; }
.discover-category-page__header p { margin: 0; color: var(--td-text-color-secondary); }
.discover-category-eyebrow { color: var(--td-brand-color); font-size: 12px; font-weight: 600; letter-spacing: .08em; }
.discover-category-page__actions { display: flex; gap: 10px; }
.discover-category-panel { background: var(--td-bg-color-container); border: 1px solid var(--td-component-border); border-radius: 8px; overflow: hidden; }
.discover-category-toolbar { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.discover-category-toolbar .t-input { width: 300px; }
.discover-category-state { display: flex; justify-content: center; align-items: center; gap: 8px; min-height: 180px; color: var(--td-text-color-secondary); }
.discover-category-row { display: flex; align-items: center; gap: 16px; padding: 18px 20px; border-bottom: 1px solid var(--td-component-border); }
.discover-category-row:last-child { border-bottom: 0; }
.discover-category-order { width: 42px; color: var(--td-text-color-placeholder); font-variant-numeric: tabular-nums; text-align: center; }
.discover-category-main { min-width: 0; flex: 1; }
.discover-category-title { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.discover-category-title code { color: var(--td-text-color-secondary); font-size: 12px; }
.discover-category-main p { margin: 8px 0 4px; color: var(--td-text-color-secondary); }
.discover-category-main small { color: var(--td-text-color-placeholder); }
.discover-category-actions { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; justify-content: flex-end; }
.discover-category-pagination { padding: 14px 18px; display: flex; justify-content: flex-end; border-top: 1px solid var(--td-component-border); }
.discover-category-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
@media (max-width: 800px) {
  .discover-category-page__header, .discover-category-toolbar { flex-direction: column; }
  .discover-category-toolbar .t-input, .discover-category-toolbar .t-select { width: 100% !important; }
  .discover-category-row { flex-wrap: wrap; }
  .discover-category-actions { width: 100%; justify-content: flex-start; }
  .discover-category-form-grid { grid-template-columns: 1fr; gap: 0; }
}
</style>
