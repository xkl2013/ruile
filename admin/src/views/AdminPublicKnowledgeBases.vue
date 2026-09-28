<template>
  <section class="public-kb-page">
    <div class="public-kb-page__header">
      <div>
        <div class="public-kb-eyebrow">平台内容目录</div>
        <h2>发布知识库</h2>
        <p>在当前工作空间创建知识库，上传资料并发布到用户的发现模块。</p>
      </div>
      <t-button theme="primary" @click="openCreateDialog">
        <template #icon><t-icon name="add" /></template>
        新建发布
      </t-button>
    </div>

    <div class="public-kb-summary">
      <article><span>全部</span><strong>{{ publications.length }}</strong></article>
      <article><span>草稿</span><strong>{{ countByStatus('draft') }}</strong></article>
      <article><span>已发布</span><strong>{{ countByStatus('published') }}</strong></article>
      <article><span>订阅用户</span><strong>{{ totalSubscribers }}</strong></article>
    </div>

    <section class="public-kb-panel">
      <div class="public-kb-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索标题、分类或描述" @enter="loadPublications">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" style="width: 140px" @change="loadPublications">
          <t-option value="all" label="全部状态" />
          <t-option value="draft" label="草稿" />
          <t-option value="published" label="已发布" />
          <t-option value="offline" label="已下架" />
        </t-select>
      </div>

      <div v-if="loading" class="public-kb-state"><t-loading size="small" /> 加载中</div>
      <div v-else-if="!publications.length" class="public-kb-state">
        <t-icon name="folder-open" />
        <span>暂无发布知识库</span>
      </div>
      <div v-else class="public-kb-table">
        <div v-for="item in publications" :key="item.id" class="public-kb-row">
          <KnowledgeBaseIcon :icon="item.icon" :icon-url="item.icon_url" :type="item.type || 'document'" size="large" />
          <div class="public-kb-row__main">
            <div class="public-kb-row__title">
              <strong>{{ item.title }}</strong>
              <t-tag :theme="statusTheme(item.status)" variant="light" size="small">{{ statusLabel(item.status) }}</t-tag>
            </div>
            <p>{{ item.description || '暂无描述' }}</p>
            <div class="public-kb-row__meta">
              <span>{{ item.category || '未分类' }}</span>
              <span>{{ item.knowledge_count || item.chunk_count || 0 }} 份内容</span>
              <span>{{ item.subscriber_count || 0 }} 位订阅</span>
            </div>
          </div>
          <div class="public-kb-row__actions">
            <t-button variant="text" shape="square" title="上传资料" @click="triggerUpload(item)">
              <template #icon><t-icon name="upload" /></template>
            </t-button>
            <t-button variant="text" shape="square" title="编辑知识库" @click="openKnowledgeBaseEditor(item)">
              <template #icon><t-icon name="edit" /></template>
            </t-button>
            <t-button variant="text" size="small" @click="openEditDialog(item)">发布信息</t-button>
            <t-button v-if="item.status !== 'published'" theme="primary" variant="outline" size="small" :loading="busyId === item.id" @click="publish(item)">
              发布
            </t-button>
            <t-button v-else theme="warning" variant="outline" size="small" :loading="busyId === item.id" @click="offline(item)">
              下架
            </t-button>
            <t-button variant="text" size="small" @click="openWorkspace(item.knowledge_base_id)">编辑内容</t-button>
          </div>
        </div>
      </div>
    </section>

    <KnowledgeBaseEditorModal
      :visible="editorVisible"
      :mode="editorMode"
      :kb-id="editorKbId || undefined"
      @update:visible="editorVisible = $event"
      @success="handleKnowledgeBaseEditorSuccess"
    />

    <input ref="uploadInput" type="file" hidden @change="handleFileChange" />

    <t-dialog v-model:visible="dialogVisible" :header="editing ? '编辑发布信息' : '补充发布信息'" :confirm-btn="{ content: '保存', loading: saving }" @confirm="saveDialog">
      <t-form :data="form" label-align="top">
        <t-form-item label="发布标题">
          <t-input v-model="form.title" placeholder="用户在发现模块看到的标题" />
        </t-form-item>
        <t-form-item label="分类">
          <t-input v-model="form.category" placeholder="例如：招生增长、教师成长" />
        </t-form-item>
        <t-form-item label="简介">
          <t-textarea v-model="form.description" :autosize="{ minRows: 3, maxRows: 6 }" placeholder="说明内容范围和适用场景" />
        </t-form-item>
      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import KnowledgeBaseEditorModal from '@/views/knowledge/KnowledgeBaseEditorModal.vue'
import { getKnowledgeBaseById, uploadKnowledgeFile } from '@/api/knowledge-base'
import { openMainAppPath } from '@admin/utils/navigation'
import {
  createAdminPublicKnowledgeBase,
  listAdminPublicKnowledgeBases,
  offlineAdminPublicKnowledgeBase,
  publishAdminPublicKnowledgeBase,
  updateAdminPublicKnowledgeBase,
  type PublicKnowledgeBasePublication,
  type PublicKnowledgeBaseStatus,
} from '@admin/api/public-knowledge-base'

const publications = ref<PublicKnowledgeBasePublication[]>([])
const loading = ref(false)
const saving = ref(false)
const busyId = ref('')
const keyword = ref('')
const statusFilter = ref<PublicKnowledgeBaseStatus | 'all'>('all')
const dialogVisible = ref(false)
const editing = ref<PublicKnowledgeBasePublication | null>(null)
const editorVisible = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editorKbId = ref('')
const pendingKnowledgeBaseId = ref('')
const uploadTarget = ref<PublicKnowledgeBasePublication | null>(null)
const uploadInput = ref<HTMLInputElement | null>(null)
const form = ref({ title: '', category: '', description: '' })

const totalSubscribers = computed(() => publications.value.reduce((total, item) => total + (item.subscriber_count || 0), 0))

function statusLabel(status: PublicKnowledgeBaseStatus) {
  return status === 'published' ? '已发布' : status === 'offline' ? '已下架' : '草稿'
}

function statusTheme(status: PublicKnowledgeBaseStatus) {
  return status === 'published' ? 'success' : status === 'offline' ? 'warning' : 'default'
}

function countByStatus(status: PublicKnowledgeBaseStatus) {
  return publications.value.filter((item) => item.status === status).length
}

async function loadPublications() {
  loading.value = true
  try {
    const response = await listAdminPublicKnowledgeBases({ status: statusFilter.value, keyword: keyword.value.trim() })
    publications.value = response.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载发布知识库失败')
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editing.value = null
  pendingKnowledgeBaseId.value = ''
  editorMode.value = 'create'
  editorKbId.value = ''
  editorVisible.value = true
}

function openEditDialog(item: PublicKnowledgeBasePublication) {
  editing.value = item
  form.value = {
    title: item.title,
    category: item.category || '',
    description: item.description || '',
  }
  dialogVisible.value = true
}

function openKnowledgeBaseEditor(item: PublicKnowledgeBasePublication) {
  editing.value = null
  pendingKnowledgeBaseId.value = ''
  editorMode.value = 'edit'
  editorKbId.value = item.knowledge_base_id
  editorVisible.value = true
}

async function handleKnowledgeBaseEditorSuccess(kbID: string) {
  if (editorMode.value === 'edit') {
    await loadPublications()
    return
  }

  try {
    const response: any = await getKnowledgeBaseById(kbID)
    const kb = response?.data || response
    if (!kb?.id) throw new Error('知识库创建成功，但未能读取知识库信息')

    pendingKnowledgeBaseId.value = kbID
    editing.value = null
    form.value = {
      title: kb.name || '',
      category: '',
      description: kb.description || '',
    }
    dialogVisible.value = true
  } catch (error: any) {
    MessagePlugin.error(error?.message || '读取知识库信息失败')
  }
}

async function saveDialog() {
  if (!form.value.title.trim()) {
    MessagePlugin.warning('请输入发布标题')
    return
  }
  saving.value = true
  try {
    if (editing.value) {
      await updateAdminPublicKnowledgeBase(editing.value.id, {
        title: form.value.title.trim(),
        category: form.value.category.trim(),
        description: form.value.description.trim(),
      })
      MessagePlugin.success('发布信息已保存')
    } else if (pendingKnowledgeBaseId.value) {
      await createAdminPublicKnowledgeBase({
        knowledge_base_id: pendingKnowledgeBaseId.value,
        title: form.value.title.trim(),
        category: form.value.category.trim(),
        description: form.value.description.trim(),
      })
      MessagePlugin.success('知识库草稿已创建')
    }
    dialogVisible.value = false
    pendingKnowledgeBaseId.value = ''
    await loadPublications()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function triggerUpload(item: PublicKnowledgeBasePublication) {
  uploadTarget.value = item
  uploadInput.value?.click()
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  const target = uploadTarget.value
  input.value = ''
  if (!file || !target) return
  busyId.value = target.id
  try {
    await uploadKnowledgeFile(target.knowledge_base_id, { file })
    MessagePlugin.success('资料已上传，解析完成后可发布')
    await loadPublications()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '资料上传失败')
  } finally {
    busyId.value = ''
    uploadTarget.value = null
  }
}

async function publish(item: PublicKnowledgeBasePublication) {
  busyId.value = item.id
  try {
    await publishAdminPublicKnowledgeBase(item.id)
    MessagePlugin.success('知识库已发布')
    await loadPublications()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发布失败，请确认资料已处理完成')
  } finally {
    busyId.value = ''
  }
}

async function offline(item: PublicKnowledgeBasePublication) {
  busyId.value = item.id
  try {
    await offlineAdminPublicKnowledgeBase(item.id)
    MessagePlugin.success('知识库已下架')
    await loadPublications()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '下架失败')
  } finally {
    busyId.value = ''
  }
}

function openWorkspace(kbID: string) {
  openMainAppPath(`/platform/knowledge-bases/${kbID}`)
}

onMounted(() => {
  void loadPublications()
})
</script>

<style scoped>
.public-kb-page { padding: 4px 0 32px; }
.public-kb-page__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 24px; margin-bottom: 22px; }
.public-kb-eyebrow { color: var(--td-text-color-secondary); font-size: 12px; letter-spacing: 0; margin-bottom: 6px; }
.public-kb-page h2 { margin: 0; color: var(--td-text-color-primary); font-size: 24px; }
.public-kb-page__header p { margin: 8px 0 0; color: var(--td-text-color-secondary); }
.public-kb-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 18px; }
.public-kb-summary article { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); padding: 14px 16px; }
.public-kb-summary span { display: block; color: var(--td-text-color-secondary); font-size: 12px; }
.public-kb-summary strong { display: block; margin-top: 6px; color: var(--td-text-color-primary); font-size: 22px; }
.public-kb-panel { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); }
.public-kb-toolbar { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.public-kb-toolbar .t-input { width: min(360px, 100%); }
.public-kb-table { padding: 0 16px; }
.public-kb-row { display: flex; align-items: center; gap: 14px; padding: 16px 0; border-bottom: 1px solid var(--td-component-border); }
.public-kb-row:last-child { border-bottom: 0; }
.public-kb-row__main { min-width: 0; flex: 1; }
.public-kb-row__title { display: flex; align-items: center; gap: 8px; }
.public-kb-row__title strong { color: var(--td-text-color-primary); font-size: 15px; }
.public-kb-row__main p { margin: 6px 0; color: var(--td-text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.public-kb-row__meta { display: flex; gap: 14px; color: var(--td-text-color-placeholder); font-size: 12px; }
.public-kb-row__actions { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; justify-content: flex-end; }
.public-kb-state { min-height: 220px; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }
@media (max-width: 900px) {
  .public-kb-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .public-kb-row { align-items: flex-start; flex-wrap: wrap; }
  .public-kb-row__actions { width: 100%; justify-content: flex-start; padding-left: 54px; }
}
</style>
