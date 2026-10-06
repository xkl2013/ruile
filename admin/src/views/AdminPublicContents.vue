<template>
  <section class="public-content-page">
    <div class="public-content-page__header">
      <div>
        <div class="public-content-eyebrow">平台内容目录</div>
        <h2>内容管理</h2>
        <p>维护小红书式图文内容和学习课程，审核通过后发布到发现模块。</p>
      </div>
    </div>

    <div class="public-content-summary">
      <article><span>全部内容</span><strong>{{ total }}</strong></article>
      <article><span>待审核</span><strong>{{ countByStatus('pending_review') }}</strong></article>
      <article><span>已发布</span><strong>{{ countByStatus('published') }}</strong></article>
      <article><span>课程内容</span><strong>{{ courseCount }}</strong></article>
    </div>

    <section class="public-content-panel">
      <div class="public-content-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索标题、摘要或系列名称" @enter="loadContents">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" style="width: 150px" @change="loadContents">
          <t-option value="all" label="全部状态" />
          <t-option value="pending_review" label="待审核" />
          <t-option value="published" label="已发布" />
          <t-option value="offline" label="已下架" />
          <t-option value="rejected" label="已驳回" />
          <t-option value="draft" label="草稿" />
        </t-select>
        <t-select v-model="typeFilter" style="width: 150px" @change="loadContents">
          <t-option value="all" label="全部类型" />
          <t-option value="post" label="图文内容" />
          <t-option value="course" label="学习课程" />
        </t-select>
      </div>

      <div v-if="loading" class="public-content-state"><t-loading size="small" /> 加载中</div>
      <div v-else-if="!contents.length" class="public-content-state">
        <t-icon name="folder-open" />
        <span>暂无符合条件的内容</span>
      </div>
      <div v-else class="public-content-table">
        <div v-for="item in contents" :key="item.id" class="public-content-row">
          <div class="public-content-row__icon">
            <t-icon :name="contentIcon(item)" />
          </div>
          <div class="public-content-row__main">
            <div class="public-content-row__title">
              <strong>{{ item.title }}</strong>
              <t-tag :theme="statusTheme(item.public_status)" variant="light" size="small">
                {{ statusLabel(item.public_status) }}
              </t-tag>
              <t-tag theme="default" variant="light-outline" size="small">
                {{ item.public_content_type === 'course' ? '课程' : '图文内容' }}
              </t-tag>
              <t-tag v-if="item.featured" theme="warning" variant="light" size="small">精选</t-tag>
              <t-tag v-if="item.recommendable" theme="success" variant="light-outline" size="small">参与推荐</t-tag>
            </div>
            <p>{{ item.source_summary || '暂无摘要' }}</p>
            <div class="public-content-row__meta">
              <span>{{ modalityLabel(item) }}</span>
              <span v-if="item.series_title">系列：{{ item.series_title }}{{ item.series_order ? ` · 第 ${item.series_order} 节` : '' }}</span>
              <span>创作者：{{ item.user_id || '未标记' }}</span>
            </div>
          </div>
          <div class="public-content-row__actions">
            <t-button variant="text" shape="square" title="编辑内容信息" @click="openEdit(item)">
              <template #icon><t-icon name="edit" /></template>
            </t-button>
            <t-button
              v-if="item.public_status === 'pending_review' || item.public_status === 'draft'"
              theme="primary"
              variant="outline"
              size="small"
              :loading="busyId === item.id"
              @click="publish(item)"
            >
              发布
            </t-button>
            <t-button
              v-if="item.public_status === 'pending_review'"
              theme="danger"
              variant="text"
              size="small"
              :loading="busyId === item.id"
              @click="reject(item)"
            >
              驳回
            </t-button>
            <t-button
              v-if="item.public_status === 'published'"
              theme="warning"
              variant="outline"
              size="small"
              :loading="busyId === item.id"
              @click="offline(item)"
            >
              下架
            </t-button>
          </div>
        </div>
      </div>
    </section>

    <t-dialog
      v-model:visible="editVisible"
      header="编辑内容信息"
      :confirm-btn="{ content: '保存', loading: saving }"
      @confirm="saveEdit"
    >
      <t-form v-if="editing" :data="editForm" label-align="top">
        <t-form-item label="标题">
          <t-input v-model="editForm.title" placeholder="用户看到的内容标题" />
        </t-form-item>
        <t-form-item label="内容类型">
          <t-select v-model="editForm.public_content_type">
            <t-option value="post" label="图文内容" />
            <t-option value="course" label="学习课程" />
          </t-select>
        </t-form-item>
        <t-form-item label="摘要">
          <t-textarea v-model="editForm.source_summary" :autosize="{ minRows: 3, maxRows: 6 }" placeholder="内容摘要" />
        </t-form-item>
        <div v-if="editForm.public_content_type === 'course'" class="course-fields">
          <t-form-item label="系列名称">
            <t-input v-model="editForm.series_title" placeholder="留空表示单篇课程；填写后可归入系列" />
          </t-form-item>
          <t-form-item label="系列节次">
            <t-input-number v-model="editForm.series_order" :min="1" :max="999" theme="normal" />
          </t-form-item>
        </div>
        <t-form-item label="发现展示">
          <div class="curation-options">
            <t-checkbox v-model="editForm.featured">加入精选</t-checkbox>
            <t-checkbox v-model="editForm.recommendable">参与推荐</t-checkbox>
          </div>
        </t-form-item>
        <t-form-item label="推荐排序">
          <t-input-number v-model="editForm.sort_order" :min="0" :max="9999" theme="normal" />
          <div class="form-help">数值越小越靠前，0 表示按更新时间排序。</div>
        </t-form-item>
        <t-form-item label="审核备注">
          <t-textarea v-model="editForm.review_note" :autosize="{ minRows: 2, maxRows: 4 }" placeholder="可记录驳回原因或发布说明" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listAdminPublicContents,
  offlineAdminPublicContent,
  publishAdminPublicContent,
  rejectAdminPublicContent,
  updateAdminPublicContent,
  type AdminPublicContent,
  type PublicContentStatus,
  type PublicContentType,
} from '@admin/api/public-content'

const contents = ref<AdminPublicContent[]>([])
const loading = ref(false)
const saving = ref(false)
const busyId = ref('')
const keyword = ref('')
const statusFilter = ref<PublicContentStatus | 'all'>('all')
const typeFilter = ref<PublicContentType | 'all'>('all')
const editVisible = ref(false)
const editing = ref<AdminPublicContent | null>(null)
const editForm = ref({
  title: '',
  source_summary: '',
  public_content_type: 'post' as PublicContentType,
  series_id: '',
  series_title: '',
  series_order: 0,
  review_note: '',
  featured: false,
  recommendable: true,
  sort_order: 0,
})

const total = ref(0)
const courseCount = computed(() => contents.value.filter((item) => item.public_content_type === 'course').length)

function countByStatus(status: PublicContentStatus) {
  return contents.value.filter((item) => item.public_status === status).length
}

function statusLabel(status: PublicContentStatus) {
  return ({
    draft: '草稿',
    pending_review: '待审核',
    published: '已发布',
    offline: '已下架',
    rejected: '已驳回',
  } as Record<PublicContentStatus, string>)[status] || status
}

function statusTheme(status: PublicContentStatus) {
  if (status === 'published') return 'success'
  if (status === 'pending_review') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

function contentIcon(item: AdminPublicContent) {
  const kind = String(item.metadata?.content_kind || item.output_type || '').toLowerCase()
  if (kind.includes('video')) return 'play-circle'
  if (kind.includes('audio')) return 'sound'
  return item.public_content_type === 'course' ? 'book-open' : 'file-word'
}

function modalityLabel(item: AdminPublicContent) {
  const kind = String(item.metadata?.content_kind_label || item.output_type || '').trim()
  return kind || '图文类'
}

async function loadContents() {
  loading.value = true
  try {
    const response = await listAdminPublicContents({
      status: statusFilter.value,
      contentType: typeFilter.value,
      keyword: keyword.value.trim(),
      page: 1,
      pageSize: 100,
    })
    contents.value = response.data?.items || []
    total.value = response.data?.total || 0
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载内容失败')
  } finally {
    loading.value = false
  }
}

function openEdit(item: AdminPublicContent) {
  editing.value = item
  editForm.value = {
    title: item.title || '',
    source_summary: item.source_summary || '',
    public_content_type: item.public_content_type || 'post',
    series_id: item.series_id || '',
    series_title: item.series_title || '',
    series_order: item.series_order || 0,
    review_note: item.review_note || '',
    featured: Boolean(item.featured),
    recommendable: item.recommendable !== false,
    sort_order: item.sort_order || 0,
  }
  editVisible.value = true
}

async function saveEdit() {
  if (!editing.value || !editForm.value.title.trim()) {
    MessagePlugin.warning('请输入标题')
    return
  }
  saving.value = true
  try {
    const seriesTitle = editForm.value.series_title.trim()
    await updateAdminPublicContent(editing.value.id, {
      title: editForm.value.title.trim(),
      source_summary: editForm.value.source_summary.trim(),
      public_content_type: editForm.value.public_content_type,
      series_id: seriesTitle ? (editForm.value.series_id || seriesTitle) : '',
      series_title: seriesTitle,
      series_order: seriesTitle ? editForm.value.series_order : 0,
      review_note: editForm.value.review_note.trim(),
      featured: editForm.value.featured,
      recommendable: editForm.value.recommendable,
      sort_order: editForm.value.sort_order,
    })
    MessagePlugin.success('内容信息已保存')
    editVisible.value = false
    await loadContents()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存内容失败')
  } finally {
    saving.value = false
  }
}

async function publish(item: AdminPublicContent) {
  busyId.value = item.id
  try {
    await publishAdminPublicContent(item.id)
    MessagePlugin.success('内容已发布')
    await loadContents()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发布失败')
  } finally {
    busyId.value = ''
  }
}

async function offline(item: AdminPublicContent) {
  busyId.value = item.id
  try {
    await offlineAdminPublicContent(item.id)
    MessagePlugin.success('内容已下架')
    await loadContents()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '下架失败')
  } finally {
    busyId.value = ''
  }
}

async function reject(item: AdminPublicContent) {
  busyId.value = item.id
  try {
    await rejectAdminPublicContent(item.id, item.review_note || '请补充内容信息后重新提交审核')
    MessagePlugin.success('内容已驳回')
    await loadContents()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '驳回失败')
  } finally {
    busyId.value = ''
  }
}

onMounted(() => {
  void loadContents()
})
</script>

<style scoped>
.public-content-page { padding: 4px 0 32px; }
.public-content-page__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 22px; }
.public-content-page__actions { display: flex; align-items: center; gap: 8px; }
.public-content-eyebrow { color: var(--td-text-color-secondary); font-size: 12px; margin-bottom: 6px; }
.public-content-page h2 { margin: 0; color: var(--td-text-color-primary); font-size: 24px; }
.public-content-page__header p { margin: 8px 0 0; color: var(--td-text-color-secondary); }
.public-content-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 18px; }
.public-content-summary article { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); padding: 14px 16px; }
.public-content-summary span { display: block; color: var(--td-text-color-secondary); font-size: 12px; }
.public-content-summary strong { display: block; margin-top: 6px; color: var(--td-text-color-primary); font-size: 22px; }
.public-content-panel { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); }
.public-content-toolbar { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.public-content-toolbar .t-input { width: min(380px, 100%); }
.public-content-table { padding: 0 16px; }
.public-content-row { display: flex; align-items: center; gap: 14px; padding: 16px 0; border-bottom: 1px solid var(--td-component-border); }
.public-content-row:last-child { border-bottom: 0; }
.public-content-row__icon { display: flex; align-items: center; justify-content: center; flex: 0 0 38px; height: 38px; color: var(--td-brand-color); background: var(--td-brand-color-light); border-radius: 6px; font-size: 20px; }
.public-content-row__main { min-width: 0; flex: 1; }
.public-content-row__title { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.public-content-row__title strong { color: var(--td-text-color-primary); font-size: 15px; }
.public-content-row__main p { margin: 6px 0; color: var(--td-text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.public-content-row__meta { display: flex; gap: 14px; color: var(--td-text-color-placeholder); font-size: 12px; flex-wrap: wrap; }
.public-content-row__actions { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; justify-content: flex-end; }
.public-content-state { min-height: 220px; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }
.course-fields { display: grid; grid-template-columns: minmax(0, 1fr) 120px; gap: 12px; }
.curation-options { display: flex; align-items: center; gap: 18px; flex-wrap: wrap; }
.form-help { margin-top: 6px; color: var(--td-text-color-placeholder); font-size: 12px; }
@media (max-width: 900px) {
  .public-content-page__header { flex-direction: column; }
  .public-content-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .public-content-toolbar { flex-wrap: wrap; }
  .public-content-row { align-items: flex-start; flex-wrap: wrap; }
  .public-content-row__actions { width: 100%; justify-content: flex-start; padding-left: 52px; }
}
</style>
