<template>
  <section class="public-creator-page">
    <div class="public-creator-page__header">
      <div>
        <div class="public-creator-eyebrow">创作者目录</div>
        <h2>创作者管理</h2>
        <p>仅展示已被管理员标记为创作者的用户，审核其知识库和内容后统一发布到发现模块。</p>
      </div>
      <div class="public-creator-page__header-actions">
        <t-button theme="primary" @click="openAddCreatorDialog">
          <template #icon><t-icon name="user-add" /></template>
          添加创作者
        </t-button>
      </div>
    </div>

    <div class="public-creator-summary">
      <article>
        <span>创作者总数</span>
        <strong>{{ total }}</strong>
      </article>
      <article>
        <span>待发布创作者</span>
        <strong>{{ pendingCreatorCount }}</strong>
      </article>
      <article>
        <span>待发布知识库</span>
        <strong>{{ pendingKnowledgeBaseCount }}</strong>
      </article>
      <article>
        <span>待发布内容</span>
        <strong>{{ pendingContentCount }}</strong>
      </article>
    </div>

    <section class="public-creator-panel">
      <div class="public-creator-toolbar">
        <t-input
          v-model="keyword"
          clearable
          placeholder="搜索创作者、用户名或邮箱"
          @enter="loadCreators"
        >
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" style="width: 150px" @change="loadCreators">
          <t-option value="all" label="全部创作者" />
          <t-option value="pending" label="有待发布资产" />
          <t-option value="published" label="已有发布资产" />
        </t-select>
      </div>

      <div v-if="loading" class="public-creator-state">
        <t-loading size="small" /> 加载中
      </div>
      <div v-else-if="!creators.length" class="public-creator-state">
        <t-icon name="usergroup" />
        <span>{{ keyword || statusFilter !== 'all' ? '暂无符合条件的创作者' : '暂无已标记的创作者，请先在用户管理中设置' }}</span>
      </div>
      <div v-else class="public-creator-list">
        <div v-for="creator in creators" :key="creator.id" class="public-creator-row">
          <t-avatar size="42px" :image="creator.avatar">
            {{ creatorInitial(creator) }}
          </t-avatar>
          <div class="public-creator-row__main">
            <div class="public-creator-row__title">
              <strong>{{ creator.display_name }}</strong>
              <t-tag v-if="creator.pending_knowledge_base_count + creator.pending_content_count > 0" theme="warning" variant="light" size="small">
                待发布
              </t-tag>
              <t-tag v-if="creator.published_knowledge_base_count + creator.published_content_count > 0" theme="success" variant="light" size="small">
                已有发布
              </t-tag>
            </div>
            <p>{{ creator.username }}<span v-if="creator.email"> · {{ creator.email }}</span></p>
            <div class="public-creator-row__meta">
              <span>知识库 {{ creator.knowledge_base_count }}</span>
              <span>内容 {{ creator.content_count }}</span>
              <span v-if="creator.pending_knowledge_base_count">待发布知识库 {{ creator.pending_knowledge_base_count }}</span>
              <span v-if="creator.pending_content_count">待发布内容 {{ creator.pending_content_count }}</span>
            </div>
          </div>
          <t-button variant="text" shape="square" title="查看创作资产" @click="openCreator(creator.id)">
            <template #icon><t-icon name="chevron-right" /></template>
          </t-button>
        </div>
      </div>
    </section>

    <t-drawer
      v-model:visible="drawerVisible"
      header="创作者详情"
      size="760px"
      :footer="false"
      @close="closeDrawer"
    >
      <div v-if="detailLoading" class="public-creator-state public-creator-state--drawer">
        <t-loading size="small" /> 加载中
      </div>
      <div v-else-if="detail" class="public-creator-detail">
        <div class="public-creator-detail__identity">
          <t-avatar size="52px" :image="detail.avatar">
            {{ creatorInitial(detail) }}
          </t-avatar>
          <div>
            <h3>{{ detail.display_name }}</h3>
            <p>{{ detail.username }}<span v-if="detail.email"> · {{ detail.email }}</span></p>
          </div>
          <div class="public-creator-detail__actions">
            <t-button
              v-if="hasPendingAssets"
              theme="primary"
              :loading="actionLoading === 'publish'"
              :disabled="!hasPublishableAssets"
              :title="hasPublishableAssets ? '' : '当前没有满足发布条件的资产'"
              @click="publishCreator"
            >
              <template #icon><t-icon name="upload" /></template>
              {{ hasPublishableAssets ? '发布全部' : '暂无可发布资产' }}
            </t-button>
            <t-button
              v-if="hasPublishedAssets"
              theme="warning"
              variant="outline"
              :loading="actionLoading === 'offline'"
              @click="offlineCreator"
            >
              <template #icon><t-icon name="download" /></template>
              全部下架
            </t-button>
          </div>
        </div>

        <div class="public-creator-detail__summary">
          <span>知识库 {{ detail.knowledge_base_count }}</span>
          <span>内容 {{ detail.content_count }}</span>
          <span>已发布 {{ detail.published_knowledge_base_count + detail.published_content_count }}</span>
          <span>待发布 {{ detail.pending_knowledge_base_count + detail.pending_content_count }}</span>
        </div>

        <section class="public-creator-assets">
          <div class="public-creator-assets__heading">
            <h4>知识库</h4>
            <span>{{ detail.knowledge_bases.length }} 个</span>
          </div>
          <div v-if="!detail.knowledge_bases.length" class="public-creator-assets__empty">暂无知识库</div>
          <div v-else class="public-creator-assets__list">
            <div v-for="item in detail.knowledge_bases" :key="item.id" class="public-creator-asset">
              <div class="public-creator-asset__icon">
                <t-icon name="book-open" />
              </div>
              <div class="public-creator-asset__main">
                <div class="public-creator-asset__title">
                  <strong>{{ item.name }}</strong>
                  <t-tag :theme="knowledgeBaseStatusTheme(item.publication_status)" variant="light" size="small">
                    {{ knowledgeBaseStatusLabel(item.publication_status) }}
                  </t-tag>
                </div>
                <p>{{ item.description || '暂无简介' }}</p>
                <div class="public-creator-asset__meta">
                  <span>{{ item.knowledge_count }} 条知识</span>
                  <span v-if="item.is_processing">处理中{{ item.processing_count ? ` · ${item.processing_count} 项` : '' }}</span>
                  <span v-else>已完成处理</span>
                  <span v-if="item.publish_block_reason" class="public-creator-asset__warning">
                    {{ publishBlockReasonLabel(item.publish_block_reason) }}
                  </span>
                </div>
              </div>
              <div v-if="item.publication_status !== 'published'" class="public-creator-asset__actions">
                <t-button
                  theme="primary"
                  variant="outline"
                  size="small"
                  :loading="knowledgeBaseActionId === item.id"
                  :disabled="!item.can_publish"
                  :title="item.publish_block_reason ? publishBlockReasonLabel(item.publish_block_reason) : ''"
                  @click="publishKnowledgeBase(item)"
                >
                  <template #icon><t-icon name="upload" /></template>
                  {{ item.can_publish ? (item.publication_status === 'offline' ? '重新发布' : '发布') : '暂不可发布' }}
                </t-button>
              </div>
            </div>
          </div>
        </section>

        <section class="public-creator-assets">
          <div class="public-creator-assets__heading">
            <h4>内容</h4>
            <span>{{ detail.contents.length }} 个</span>
          </div>
          <div v-if="!detail.contents.length" class="public-creator-assets__empty">暂无公开提交内容</div>
          <div v-else class="public-creator-assets__list">
            <div v-for="item in detail.contents" :key="item.id" class="public-creator-asset">
              <div class="public-creator-asset__icon">
                <t-icon :name="contentIcon(item)" />
              </div>
              <div class="public-creator-asset__main">
                <div class="public-creator-asset__title">
                  <strong>{{ item.title }}</strong>
                  <t-tag :theme="contentStatusTheme(item.public_status)" variant="light" size="small">
                    {{ contentStatusLabel(item.public_status) }}
                  </t-tag>
                  <t-tag theme="default" variant="light-outline" size="small">
                    {{ item.public_content_type === 'course' ? '课程' : '图文' }}
                  </t-tag>
                </div>
                <p>{{ item.source_summary || '暂无摘要' }}</p>
                <div class="public-creator-asset__meta">
                  <span>{{ modalityLabel(item) }}</span>
                  <span v-if="item.series_title">系列：{{ item.series_title }}{{ item.series_order ? ` · 第 ${item.series_order} 节` : '' }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
    </t-drawer>

    <t-dialog
      v-model:visible="addCreatorVisible"
      header="添加创作者"
      width="640px"
      :footer="false"
      destroy-on-close
      @close="closeAddCreatorDialog"
    >
      <div class="public-creator-add">
        <p class="public-creator-add__hint">选择用户并授予创作者身份，授权后该用户的知识库和内容会进入审核目录。</p>
        <div class="public-creator-add__toolbar">
          <t-input
            v-model="creatorSearchKeyword"
            clearable
            placeholder="搜索用户名、邮箱或手机号"
            @enter="loadCreatorCandidates"
          >
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
          <t-button variant="outline" :loading="creatorCandidatesLoading" @click="loadCreatorCandidates">
            <template #icon><t-icon name="search" /></template>
            搜索
          </t-button>
        </div>

        <div v-if="creatorCandidatesLoading" class="public-creator-add__state">
          <t-loading size="small" /> 加载用户中
        </div>
        <div v-else-if="!creatorCandidates.length" class="public-creator-add__state">
          <t-icon name="user-search" />
          <span>{{ creatorSearchKeyword ? '没有找到可添加的用户' : '暂无可添加的用户' }}</span>
        </div>
        <div v-else class="public-creator-candidate-list">
          <div v-for="user in creatorCandidates" :key="user.id" class="public-creator-candidate">
            <t-avatar size="38px" :image="user.avatar">
              {{ (user.username || user.email || '?').slice(0, 1).toUpperCase() }}
            </t-avatar>
            <div class="public-creator-candidate__main">
              <strong>{{ user.username || '未设置用户名' }}</strong>
              <span>{{ user.email || '未设置邮箱' }}</span>
            </div>
            <t-button
              size="small"
              theme="primary"
              :loading="creatorMutationID === user.id"
              :disabled="creatorMutationID !== ''"
              @click="addCreator(user)"
            >
              <template #icon><t-icon name="user-add" /></template>
              添加
            </t-button>
          </div>
        </div>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  searchSystemUsers,
  setSystemUserCreator,
  type SystemUserSummary,
} from '@/api/system'
import {
  getAdminPublicCreator,
  listAdminPublicCreators,
  offlineAdminPublicCreator,
  publishAdminPublicCreatorKnowledgeBase,
  publishAdminPublicCreator,
  type AdminPublicCreatorContent,
  type AdminPublicCreatorDetail,
  type AdminPublicCreatorKnowledgeBase,
  type AdminPublicCreatorSummary,
  type PublicCreatorStatus,
} from '@admin/api/public-creator'

const creators = ref<AdminPublicCreatorSummary[]>([])
const total = ref(0)
const loading = ref(false)
const keyword = ref('')
const statusFilter = ref<PublicCreatorStatus>('all')
const drawerVisible = ref(false)
const detailLoading = ref(false)
const actionLoading = ref<'publish' | 'offline' | ''>('')
const knowledgeBaseActionId = ref('')
const detail = ref<AdminPublicCreatorDetail | null>(null)
const addCreatorVisible = ref(false)
const creatorSearchKeyword = ref('')
const creatorCandidates = ref<SystemUserSummary[]>([])
const creatorCandidatesLoading = ref(false)
const creatorMutationID = ref('')

const pendingCreatorCount = computed(() => creators.value.filter((item) => hasPending(item)).length)
const pendingKnowledgeBaseCount = computed(() => creators.value.reduce((sum, item) => sum + item.pending_knowledge_base_count, 0))
const pendingContentCount = computed(() => creators.value.reduce((sum, item) => sum + item.pending_content_count, 0))
const hasPendingAssets = computed(() => Boolean(detail.value && detail.value.pending_knowledge_base_count + detail.value.pending_content_count > 0))
const hasPublishableAssets = computed(() => Boolean(
  detail.value
  && (
    detail.value.knowledge_bases.some((item) => item.publication_status !== 'published' && item.can_publish)
    || detail.value.contents.some((item) => item.public_status !== 'published')
  ),
))
const hasPublishedAssets = computed(() => Boolean(detail.value && detail.value.published_knowledge_base_count + detail.value.published_content_count > 0))

function hasPending(item: AdminPublicCreatorSummary) {
  return item.pending_knowledge_base_count + item.pending_content_count > 0
}

function creatorInitial(item: AdminPublicCreatorSummary | AdminPublicCreatorDetail) {
  return (item.display_name || item.username || item.email || '?').slice(0, 1).toUpperCase()
}

function knowledgeBaseStatusLabel(status: string) {
  return ({
    unpublished: '未进入目录',
    draft: '待发布',
    published: '已发布',
    offline: '已下架',
  } as Record<string, string>)[status] || status
}

function knowledgeBaseStatusTheme(status: string) {
  if (status === 'published') return 'success'
  if (status === 'offline') return 'default'
  return 'warning'
}

function contentStatusLabel(status: string) {
  return ({
    draft: '草稿',
    pending_review: '待审核',
    published: '已发布',
    offline: '已下架',
    rejected: '已驳回',
  } as Record<string, string>)[status] || status || '待处理'
}

function contentStatusTheme(status: string) {
  if (status === 'published') return 'success'
  if (status === 'pending_review' || status === 'draft') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

function contentIcon(item: AdminPublicCreatorContent) {
  const kind = String(item.metadata?.content_kind || item.output_type || '').toLowerCase()
  if (kind.includes('video')) return 'play-circle'
  if (kind.includes('audio')) return 'sound'
  return item.public_content_type === 'course' ? 'book-open' : 'file-word'
}

function modalityLabel(item: AdminPublicCreatorContent) {
  const kind = String(item.metadata?.content_kind_label || item.output_type || '').trim()
  return kind || '图文类'
}

function publishBlockReasonLabel(reason?: string) {
  return ({
    title_required: '请先补充发布标题',
    content_required: '请先添加至少一条知识内容',
    processing: '内容仍在处理中，请稍后再发布',
    knowledge_base_missing: '知识库信息不完整',
  } as Record<string, string>)[reason || ''] || '当前不满足发布条件'
}

function publishFailureMessage(message?: string) {
  const text = String(message || '')
  if (text.includes('published knowledge base requires a title and at least one content item')) {
    return '知识库发布前需要有发布标题，并且至少包含一条知识内容。'
  }
  if (text.includes('knowledge base is still processing')) {
    return '知识库内容仍在处理中，请稍后再发布。'
  }
  return text || '知识库发布失败'
}

async function loadCreators() {
  loading.value = true
  try {
    const response = await listAdminPublicCreators({
      status: statusFilter.value,
      keyword: keyword.value.trim(),
      page: 1,
      pageSize: 100,
    })
    creators.value = response.data?.items || []
    total.value = response.data?.total || 0
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载创作者失败')
  } finally {
    loading.value = false
  }
}

async function loadCreatorCandidates() {
  creatorCandidatesLoading.value = true
  try {
    const response = await searchSystemUsers({
      keyword: creatorSearchKeyword.value.trim(),
      limit: 50,
    })
    creatorCandidates.value = (response.users || []).filter((user) => !user.is_creator)
  } catch (error: any) {
    creatorCandidates.value = []
    MessagePlugin.error(error?.message || '加载可添加用户失败')
  } finally {
    creatorCandidatesLoading.value = false
  }
}

function openAddCreatorDialog() {
  addCreatorVisible.value = true
  creatorSearchKeyword.value = ''
  void loadCreatorCandidates()
}

function closeAddCreatorDialog() {
  creatorCandidates.value = []
  creatorSearchKeyword.value = ''
  creatorMutationID.value = ''
}

function addCreator(user: SystemUserSummary) {
  if (creatorMutationID.value) return
  const dialog = DialogPlugin.confirm({
    header: '确认添加创作者',
    body: `确认将「${user.username || user.email || '该用户'}」设置为创作者？`,
    confirmBtn: { content: '确认添加', theme: 'primary' },
    cancelBtn: { content: '取消' },
    onConfirm: async () => {
      dialog.destroy()
      creatorMutationID.value = user.id
      try {
        await setSystemUserCreator(user.id, true)
        MessagePlugin.success('已添加创作者')
        await Promise.all([loadCreators(), loadCreatorCandidates()])
      } catch (error: any) {
        MessagePlugin.error(error?.message || '添加创作者失败')
      } finally {
        creatorMutationID.value = ''
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

async function openCreator(id: string) {
  drawerVisible.value = true
  detailLoading.value = true
  detail.value = null
  try {
    const response = await getAdminPublicCreator(id)
    detail.value = response.data
  } catch (error: any) {
    drawerVisible.value = false
    MessagePlugin.error(error?.message || '读取创作者资产失败')
  } finally {
    detailLoading.value = false
  }
}

function closeDrawer() {
  detail.value = null
}

async function reloadDetail() {
  if (!detail.value) return
  const response = await getAdminPublicCreator(detail.value.id)
  detail.value = response.data
  await loadCreators()
}

async function publishKnowledgeBase(item: AdminPublicCreatorKnowledgeBase) {
  if (!detail.value) return
  if (!item.can_publish) {
    MessagePlugin.warning(publishBlockReasonLabel(item.publish_block_reason))
    return
  }
  knowledgeBaseActionId.value = item.id
  try {
    const response = await publishAdminPublicCreatorKnowledgeBase(detail.value.id, item.id)
    const result = response.data
    if (result.failures.length) {
      MessagePlugin.warning(publishFailureMessage(result.failures[0]?.message))
    } else if (result.knowledge_bases_published > 0) {
      MessagePlugin.success('知识库已发布')
    } else {
      MessagePlugin.info('知识库无需重复发布')
    }
    await reloadDetail()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库发布失败')
  } finally {
    knowledgeBaseActionId.value = ''
  }
}

async function publishCreator() {
  if (!detail.value) return
  if (!hasPublishableAssets.value) {
    MessagePlugin.warning('当前没有满足发布条件的资产，请先补充知识库内容或发布内容')
    return
  }
  actionLoading.value = 'publish'
  try {
    const response = await publishAdminPublicCreator(detail.value.id)
    const result = response.data
    if (result.failures.length) {
      MessagePlugin.warning(`已完成部分发布，${result.failures.length} 项资产未发布`)
    } else {
      MessagePlugin.success('创作者资产已全部发布')
    }
    await reloadDetail()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发布创作者资产失败')
  } finally {
    actionLoading.value = ''
  }
}

async function offlineCreator() {
  if (!detail.value) return
  actionLoading.value = 'offline'
  try {
    const response = await offlineAdminPublicCreator(detail.value.id)
    const result = response.data
    if (result.failures.length) {
      MessagePlugin.warning(`已完成部分下架，${result.failures.length} 项资产未下架`)
    } else {
      MessagePlugin.success('创作者资产已全部下架')
    }
    await reloadDetail()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '下架创作者资产失败')
  } finally {
    actionLoading.value = ''
  }
}

onMounted(() => {
  void loadCreators()
})
</script>

<style scoped>
.public-creator-page { padding: 4px 0 32px; }
.public-creator-page__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 22px; }
.public-creator-page__header-actions { flex: 0 0 auto; }
.public-creator-eyebrow { color: var(--td-text-color-secondary); font-size: 12px; margin-bottom: 6px; }
.public-creator-page h2 { margin: 0; color: var(--td-text-color-primary); font-size: 24px; }
.public-creator-page__header p { margin: 8px 0 0; color: var(--td-text-color-secondary); }
.public-creator-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 18px; }
.public-creator-summary article { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); padding: 14px 16px; }
.public-creator-summary span { display: block; color: var(--td-text-color-secondary); font-size: 12px; }
.public-creator-summary strong { display: block; margin-top: 8px; color: var(--td-text-color-primary); font-size: 24px; line-height: 1; }
.public-creator-panel { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); }
.public-creator-toolbar { display: flex; gap: 10px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.public-creator-toolbar .t-input { flex: 1; }
.public-creator-state { display: flex; min-height: 220px; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }
.public-creator-state > .t-icon { font-size: 26px; }
.public-creator-state--drawer { min-height: 280px; }
.public-creator-list { display: flex; flex-direction: column; }
.public-creator-row { display: flex; align-items: center; gap: 12px; min-height: 80px; padding: 14px 16px; border-bottom: 1px solid var(--td-component-border); }
.public-creator-row:last-child { border-bottom: 0; }
.public-creator-row__main { min-width: 0; flex: 1; }
.public-creator-row__title, .public-creator-asset__title { display: flex; align-items: center; flex-wrap: wrap; gap: 6px; }
.public-creator-row__title strong { color: var(--td-text-color-primary); }
.public-creator-row__main p, .public-creator-asset__main p { margin: 4px 0 0; color: var(--td-text-color-secondary); font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.public-creator-row__meta, .public-creator-asset__meta { display: flex; flex-wrap: wrap; gap: 14px; margin-top: 8px; color: var(--td-text-color-placeholder); font-size: 12px; }
.public-creator-detail { padding-bottom: 24px; }
.public-creator-detail__identity { display: flex; align-items: center; gap: 12px; padding-bottom: 18px; border-bottom: 1px solid var(--td-component-border); }
.public-creator-detail__identity h3 { margin: 0; font-size: 20px; color: var(--td-text-color-primary); }
.public-creator-detail__identity p { margin: 4px 0 0; color: var(--td-text-color-secondary); font-size: 13px; }
.public-creator-detail__actions { display: flex; flex-wrap: wrap; gap: 8px; margin-left: auto; }
.public-creator-detail__summary { display: flex; flex-wrap: wrap; gap: 18px; padding: 14px 0 6px; color: var(--td-text-color-secondary); font-size: 13px; }
.public-creator-assets { margin-top: 22px; }
.public-creator-assets__heading { display: flex; align-items: baseline; justify-content: space-between; margin-bottom: 10px; }
.public-creator-assets__heading h4 { margin: 0; color: var(--td-text-color-primary); font-size: 16px; }
.public-creator-assets__heading span { color: var(--td-text-color-placeholder); font-size: 12px; }
.public-creator-assets__list { border: 1px solid var(--td-component-border); }
.public-creator-assets__empty { padding: 18px; color: var(--td-text-color-placeholder); font-size: 13px; }
.public-creator-asset { display: flex; gap: 12px; padding: 14px; border-bottom: 1px solid var(--td-component-border); }
.public-creator-asset:last-child { border-bottom: 0; }
.public-creator-asset__icon { display: flex; width: 34px; height: 34px; align-items: center; justify-content: center; flex: 0 0 34px; color: var(--td-brand-color); background: var(--td-brand-color-light); }
.public-creator-asset__main { min-width: 0; flex: 1; }
.public-creator-asset__title strong { max-width: 100%; color: var(--td-text-color-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.public-creator-asset__actions { display: flex; align-items: center; flex: 0 0 auto; }
.public-creator-asset__warning { color: var(--td-warning-color); }
.public-creator-add__hint { margin: 0 0 14px; color: var(--td-text-color-secondary); font-size: 13px; line-height: 1.5; }
.public-creator-add__toolbar { display: flex; gap: 10px; margin-bottom: 14px; }
.public-creator-add__toolbar .t-input { flex: 1; }
.public-creator-add__state { display: flex; min-height: 160px; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }
.public-creator-candidate-list { border: 1px solid var(--td-component-border); }
.public-creator-candidate { display: flex; align-items: center; gap: 10px; padding: 12px; border-bottom: 1px solid var(--td-component-border); }
.public-creator-candidate:last-child { border-bottom: 0; }
.public-creator-candidate__main { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 3px; }
.public-creator-candidate__main strong { color: var(--td-text-color-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.public-creator-candidate__main span { color: var(--td-text-color-secondary); font-size: 12px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
@media (max-width: 760px) {
  .public-creator-page__header { flex-direction: column; }
  .public-creator-page__header-actions { width: 100%; }
  .public-creator-page__header-actions .t-button { width: 100%; }
  .public-creator-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .public-creator-toolbar { flex-direction: column; }
  .public-creator-toolbar .t-select { width: 100% !important; }
  .public-creator-detail__identity { align-items: flex-start; flex-wrap: wrap; }
  .public-creator-detail__actions { width: 100%; margin-left: 64px; }
  .public-creator-add__toolbar { flex-direction: column; }
}
</style>
