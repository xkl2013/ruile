<template>
  <div class="organize-page">
    <main class="organize-main">
      <header class="organize-header">
        <div class="organize-header-title">
          <div class="organize-title-row">
            <h2>{{ activeMeta.title }}</h2>
          </div>
        </div>
        <div v-if="activeTab === 'memory'" class="organize-header-actions">
          <t-input v-model="keyword" class="organize-search" clearable placeholder="搜索">
            <template #prefix-icon>
              <t-icon name="search" />
            </template>
          </t-input>
        </div>
      </header>

      <div class="organize-scroll">
        <section v-if="activeTab === 'memory'" class="organize-section organize-section--memory">
          <div class="asset-summary" aria-label="记忆状态">
            <div class="section-heading">
              <t-icon name="folder" />
              <span>记忆状态（{{ allMemoryItems.length }} 条）</span>
              <t-icon name="info-circle" class="section-heading-info" />
            </div>
            <div class="asset-grid">
              <button
                v-for="status in memoryStatusCards"
                :key="status.key"
                type="button"
                class="asset-card"
                @click="openMemoryStatusList(status.key)"
              >
                <span class="asset-card-label">{{ status.label }}</span>
                <span class="asset-card-description">{{ status.description }}</span>
                <span class="asset-card-value">{{ status.count }} {{ status.unit }}</span>
              </button>
            </div>
          </div>

          <template v-if="activeMemoryAsset || activeMemoryStatus">
            <div class="memory-list-toolbar">
              <button type="button" class="memory-list-back" @click="openMemoryOverview">
                <t-icon name="chevron-left" />
                <span>记忆状态</span>
              </button>
              <div class="section-heading">
                <t-icon :name="activeMemoryFilterMeta.icon" />
                <span>{{ activeMemoryFilterMeta.label }}列表</span>
              </div>
            </div>
          </template>

          <div v-if="memoryLoading" class="organize-loading">
            <t-loading size="medium" text="加载记忆中" />
          </div>
          <div v-else class="memory-asset-list memory-overview-list">
            <section
              v-for="group in visibleMemoryListGroups"
              :key="group.date || activeMemoryAsset || 'all'"
              class="timeline-group"
            >
              <div v-if="group.showDate" class="timeline-date-row">
                <h2>{{ group.date }}</h2>
                <t-icon name="chevron-up" />
              </div>
              <article
                v-for="item in group.items"
                :key="item.id"
                class="memory-list-card output-card"
                :class="[
                  `memory-list-card--${memoryCardKind(item)}`,
                  { 'memory-list-card--editable': isEditableMemory(item) },
                ]"
                :role="isEditableMemory(item) ? 'button' : undefined"
                :tabindex="isEditableMemory(item) ? 0 : undefined"
                @click="openMemoryEditor(item)"
                @keydown.enter.self.prevent="openMemoryEditor(item)"
                @keydown.space.self.prevent="openMemoryEditor(item)"
              >
                <div class="memory-card-actions" @click.stop>
                  <t-popup
                    :visible="activeMemoryMenuId === item.id"
                    trigger="click"
                    overlayClassName="card-more-popup memory-card-menu-popup"
                    destroy-on-close
                    placement="bottom-right"
                    @visible-change="(visible: boolean) => handleMemoryMenuVisible(item.id, visible)"
                    @update:visible="(visible: boolean) => handleMemoryMenuVisible(item.id, visible)"
                  >
                    <button
                      type="button"
                      class="memory-card-more"
                      :class="{ 'is-active': activeMemoryMenuId === item.id }"
                      :aria-label="`打开 ${item.title || '无标题'} 操作菜单`"
                      @click.stop
                    >
                      <t-icon name="ellipsis" />
                    </button>
                    <template #content>
                      <div class="popup-menu memory-card-menu" @click.stop>
                        <button
                          v-if="isEditableMemory(item)"
                          type="button"
                          class="popup-menu-item memory-menu-item"
                          @click.stop="handleMemoryMenuAction(item, 'edit')"
                        >
                          <t-icon class="menu-icon" name="edit" />
                          <span>编辑</span>
                        </button>
                        <button
                          type="button"
                          class="popup-menu-item memory-menu-item"
                          :disabled="isMemoryOrganizeDisabled(item.id)"
                          @click.stop="handleMemoryMenuAction(item, 'organize')"
                        >
                          <t-icon class="menu-icon" name="layers" />
                          <span>{{ memoryOrganizeActionLabel(item) }}</span>
                        </button>
                        <button type="button" class="popup-menu-item delete memory-menu-item" @click.stop="handleMemoryMenuAction(item, 'delete')">
                          <t-icon class="menu-icon" name="delete" />
                          <span>删除</span>
                        </button>
                      </div>
                    </template>
                  </t-popup>
                </div>
                <div class="output-card-cover memory-card-cover" @click.stop="openMemoryEditor(item)">
                  <div class="output-card-cover-media memory-card-cover-media">
                    <t-icon :name="memoryTypeIcon(item)" />
                  </div>
                  <span class="output-kind-label memory-kind-label">{{ memoryKindLabel(item) }}</span>
                </div>
                <div class="output-card-body memory-list-card-main" @click.stop="openMemoryEditor(item)">
                  <h2>{{ item.title || '无标题' }}</h2>
                  <p v-if="memoryCardBodyText(item)" class="output-summary memory-card-content">{{ memoryCardBodyText(item) }}</p>
                  <div class="output-card-footer memory-card-footer">
                    <div class="discover-card-meta">
                      <span>{{ memoryCardTimeLabel(item, !group.showDate) }}</span>
                      <template v-if="memoryCardFooterInfo(item)">
                        <span class="discover-card-meta-separator">|</span>
                        <span>{{ memoryCardFooterInfo(item) }}</span>
                      </template>
                    </div>
                  </div>
                </div>
              </article>
            </section>
            <div v-if="visibleMemoryListEmpty" class="memory-list-empty">
              {{ memoryListEmptyText }}
            </div>
          </div>
        </section>

        <section v-else-if="activeTab === 'output'" class="organize-section organize-section--output">
          <div class="discover-mode-switch" role="tablist" aria-label="发现内容类型">
            <button
              type="button"
              :class="{ active: discoverMode === 'content' }"
              role="tab"
              :aria-selected="discoverMode === 'content'"
              @click="discoverMode = 'content'"
            >
              内容
            </button>
            <button
              type="button"
              :class="{ active: discoverMode === 'knowledge-base' }"
              role="tab"
              :aria-selected="discoverMode === 'knowledge-base'"
              @click="discoverMode = 'knowledge-base'"
            >
              知识库
            </button>
          </div>
          <PublicKnowledgeBaseDiscover v-if="discoverMode === 'knowledge-base'" />
          <div v-else class="discover-board">
            <section class="discover-featured-section">
              <div class="discover-section-head">
                <h3>精选</h3>
                <t-button
                  variant="text"
                  theme="default"
                  class="discover-refresh"
                  :disabled="discoverFeaturedLoading || featuredOutputs.length <= 1"
                  @click="rotateFeaturedOutputs"
                >
                  <template #icon><t-icon name="refresh" /></template>
                  换一换
                </t-button>
              </div>

              <div v-if="discoverFeaturedLoading" class="organize-loading discover-loading">
                <t-loading size="medium" text="加载精选中" />
              </div>
              <div v-else-if="featuredOutputs.length" class="discover-featured-grid">
                <article
                  v-for="item in featuredOutputs"
                  :key="`featured-${item.id}`"
                  class="output-card output-card--editable discover-card discover-card--featured"
                  :class="`output-card--${item.kind}`"
                  role="button"
                  tabindex="0"
                  @click="openOutputPreview(item)"
                  @keydown.enter.self="openOutputPreview(item)"
                >
                  <div
                    class="output-card-cover"
                    :class="`output-card-cover--${item.kind}`"
                    :style="outputCardCoverStyle(item)"
                  >
                    <div class="output-card-cover-media">
                      <template v-if="item.coverUrl">
                        <img :src="item.coverUrl" :alt="item.title" />
                      </template>
                      <template v-else>
                        <t-icon :name="item.icon" />
                      </template>
                    </div>
                    <span class="output-kind-label">{{ item.kindLabel }}</span>
                  </div>
                  <div class="output-card-body">
                    <div class="output-card-head">
                      <div class="output-card-actions" @click.stop>
                        <t-dropdown
                          v-if="canEditOutputItem(item)"
                          :options="getOutputStatusMenuOptions(item.statusKey)"
                          trigger="click"
                          placement="bottom-right"
                          attach="body"
                          @click="(action: any) => handleOutputStatusMenuClick(item, action)"
                        >
                          <button
                            type="button"
                            class="icon-button icon-button--more"
                            :aria-label="`更多操作 ${item.title}`"
                            @click.stop
                          >
                            <t-icon name="ellipsis" />
                          </button>
                        </t-dropdown>
                      </div>
                    </div>
                    <h2>{{ item.title }}</h2>
                    <div class="output-card-category output-card-category--content">
                      {{ item.contentTypeLabel }}<span v-if="item.seriesLabel"> · {{ item.seriesLabel }}</span>
                    </div>
                    <div v-if="item.categoryLabel" class="output-card-category">{{ item.categoryLabel }}</div>
                    <p class="output-summary">{{ item.summary }}</p>
                    <div class="output-card-footer">
                      <div class="discover-card-meta">
                        <span>{{ item.createdAtLabel }}</span>
                        <span class="discover-card-meta-separator">|</span>
                        <span>{{ '@' + outputCreatorDisplayName(item) }}</span>
                      </div>
                    </div>
                  </div>
                </article>
              </div>
              <OrganizeCourseDiscover
                v-if="!discoverFeaturedLoading"
                ref="featuredCourseDiscoverRef"
                variant="featured"
                :limit="FEATURED_OUTPUT_SIZE"
                :show-empty="showFeaturedCourseFallback"
              />
            </section>

            <section class="discover-tabs-section">
              <div class="discover-tabs-bar">
                <div class="discover-tabs-row">
                  <button
                    v-for="tab in discoverTabs"
                    :key="tab.value"
                    type="button"
                    class="discover-tab"
                    :class="{ 'discover-tab--active': discoverTab === tab.value }"
                    @click="setDiscoverTab(tab.value)"
                  >
                    {{ tab.label }}
                  </button>
                </div>
              </div>
            </section>

            <section class="discover-feed-section">
              <OrganizeCourseDiscover
                v-if="discoverTab === 'course'"
                ref="courseDiscoverRef"
              />
              <template v-else>
                <div v-if="discoverFeedLoading" class="organize-loading discover-loading">
                  <t-loading size="medium" text="加载发现中" />
                </div>
                <div v-else-if="paginatedOutputs.length" class="discover-feed-grid">
                  <article
                    v-for="item in paginatedOutputs"
                    :key="item.id"
                    class="output-card output-card--editable discover-card discover-card--feed"
                    :class="`output-card--${item.kind}`"
                    role="button"
                    tabindex="0"
                    @click="openOutputPreview(item)"
                    @keydown.enter.self="openOutputPreview(item)"
                  >
                    <div
                      class="output-card-cover"
                      :class="`output-card-cover--${item.kind}`"
                      :style="outputCardCoverStyle(item)"
                    >
                      <div class="output-card-cover-media">
                        <template v-if="item.coverUrl">
                          <img :src="item.coverUrl" :alt="item.title" />
                        </template>
                        <template v-else>
                          <t-icon :name="item.icon" />
                        </template>
                      </div>
                      <span class="output-kind-label">{{ item.kindLabel }}</span>
                    </div>
                    <div class="output-card-body">
                      <div class="output-card-head">
                        <div class="output-card-actions" @click.stop>
                          <t-dropdown
                            v-if="canEditOutputItem(item)"
                            :options="getOutputStatusMenuOptions(item.statusKey)"
                            trigger="click"
                            placement="bottom-right"
                            attach="body"
                            @click="(action: any) => handleOutputStatusMenuClick(item, action)"
                          >
                            <button
                              type="button"
                              class="icon-button icon-button--more"
                              :aria-label="`更多操作 ${item.title}`"
                              @click.stop
                            >
                              <t-icon name="ellipsis" />
                            </button>
                          </t-dropdown>
                        </div>
                      </div>
                      <h2>{{ item.title }}</h2>
                      <div class="output-card-category output-card-category--content">
                        {{ item.contentTypeLabel }}<span v-if="item.seriesLabel"> · {{ item.seriesLabel }}</span>
                      </div>
                      <div v-if="item.categoryLabel" class="output-card-category">{{ item.categoryLabel }}</div>
                      <p class="output-summary">{{ item.summary }}</p>
                      <div class="output-card-footer">
                        <div class="discover-card-meta">
                          <span>{{ item.createdAtLabel }}</span>
                          <span class="discover-card-meta-separator">|</span>
                          <span>{{ '@' + outputCreatorDisplayName(item) }}</span>
                        </div>
                      </div>
                    </div>
                  </article>
                </div>
                <div v-else class="output-empty">
                  {{ outputEmptyText }}
                </div>
              </template>
            </section>

            <div
              v-if="discoverTab !== 'course' && !discoverFeedLoading && discoverTotal > OUTPUT_PAGE_SIZE"
              class="output-pagination"
              aria-label="发现分页"
            >
              <t-pagination
                v-model="outputPage"
                :page-size="OUTPUT_PAGE_SIZE"
                :total="discoverTotal"
                size="small"
                show-page-number
                @change="handleDiscoverPageChange"
              />
            </div>
          </div>
        </section>

      </div>
    </main>

    <div v-if="activeTab === 'memory' || activeTab === 'output'" class="organize-fab-wrap">
      <t-dropdown
        v-if="activeTab === 'output'"
        :options="outputCreateOptions"
        trigger="click"
        placement="top-right"
        @click="handleOutputCreateAction"
      >
        <t-button
          class="organize-fab"
          theme="primary"
          shape="circle"
          :aria-label="activeMeta.actionLabel"
        >
          <template #icon><t-icon name="add" size="20px" /></template>
        </t-button>
      </t-dropdown>
      <template v-else>
        <t-dropdown
          :options="memoryCreateOptions"
          trigger="click"
          placement="top-right"
          :disabled="memoryImporting"
          @click="handleMemoryCreateAction"
        >
          <t-button
            class="organize-fab"
            theme="primary"
            shape="circle"
            :loading="memoryImporting"
            :aria-label="activeMeta.actionLabel"
          >
            <template #icon><t-icon name="add" size="20px" /></template>
          </t-button>
        </t-dropdown>
      </template>
      <input
        v-if="activeTab === 'memory'"
        ref="memoryImportInputRef"
        class="memory-import-input"
        type="file"
        multiple
        :accept="memoryImportAccept"
        @change="handleMemoryImportFileChange"
      />
    </div>

    <t-drawer
      v-model:visible="outputPreviewVisible"
      class="output-preview-drawer"
      :header="false"
      :footer="false"
      :close-btn="false"
      :size="'min(720px, 92vw)'"
      attach="body"
      placement="right"
    >
      <template v-if="activeOutputPreview">
        <div class="output-preview-header">
          <div class="output-preview-heading">
            <span class="output-preview-icon" :class="`output-preview-icon--${activeOutputPreview.kind}`">
              <t-icon :name="activeOutputPreview.icon" />
            </span>
            <div class="output-preview-title-block">
              <div class="output-preview-eyebrow">{{ activeOutputPreview.kindLabel }}</div>
              <div class="output-preview-title" :title="activeOutputPreview.title">{{ activeOutputPreview.title }}</div>
            </div>
          </div>
          <div class="output-preview-actions">
            <t-button
              v-if="canEditActiveOutputPreview"
              variant="text"
              theme="default"
              size="small"
              class="output-preview-action"
              aria-label="编辑发现"
              @click="editActiveOutputPreview"
            >
              <template #icon><t-icon name="edit-1" size="16px" /></template>
            </t-button>
            <t-button
              variant="text"
              theme="default"
              size="small"
              class="output-preview-action"
              aria-label="关闭预览"
              @click="closeOutputPreview"
            >
              <template #icon><t-icon name="close" size="16px" /></template>
            </t-button>
          </div>
        </div>

        <div class="output-preview-body">
          <section class="output-preview-section">
            <h4>摘要</h4>
            <p class="output-preview-summary">{{ activeOutputPreview.summary }}</p>
            <div v-if="activeOutputPreview.categoryLabel" class="output-preview-category">
              栏目：{{ activeOutputPreview.categoryLabel }}
            </div>
            <div class="output-preview-category">
              {{ activeOutputPreview.contentTypeLabel }}<span v-if="activeOutputPreview.seriesLabel">：{{ activeOutputPreview.seriesLabel }}</span>
            </div>
            <div v-if="activeOutputPreview.tags.length" class="output-preview-tags">
              <t-tag v-for="tag in activeOutputPreview.tags" :key="`preview-${activeOutputPreview.id}-${tag}`" size="small" variant="light-outline">
                {{ tag }}
              </t-tag>
            </div>
          </section>

          <section class="output-preview-section output-preview-content-section">
            <h4>源文件预览</h4>
            <div v-if="outputPreviewSourceUrl" class="output-preview-file-host">
              <DocumentPreview
                :source-url="outputPreviewSourceUrl"
                :file-type="outputPreviewFileTypeForPreview"
                :file-name="outputPreviewFileName"
                :active="outputPreviewVisible"
                fill-height
              />
            </div>
            <div v-else class="output-preview-file-empty">
              <t-icon name="file-unknown" />
              <span>暂无源文件</span>
            </div>
          </section>
        </div>
      </template>
    </t-drawer>

    <OrganizeOutputUploadDrawer
      v-model:visible="outputUploadVisible"
      :initial-kind="outputUploadInitialKind"
      @uploaded="handleOutputUploaded"
      @saved="handleOutputSaved"
    />

  </div>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref, watch } from 'vue'
import { DialogPlugin, Icon as TIcon, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import DocumentPreview from '@/components/document-preview.vue'
import {
  createOrganizeJob,
  deleteOrganizeMemory,
  deleteOrganizeOutput,
  getOrganizeDiscover,
  listOrganizeConfigs,
  listOrganizeJobs,
  listOrganizeMemories,
  listOrganizeOutputs,
  type OrganizeDiscoverTab,
  type OrganizeJob,
  type OrganizeJobStatus,
  type OrganizeMemory,
  type OrganizeMemoryKind,
  type OrganizeOutput,
  type OrganizeOutputStatus,
  updateOrganizeOutput,
  uploadOrganizeMemory,
} from '@/api/organize'
import { useAuthStore } from '@/stores/auth'
import {
  ORGANIZE_MEMORY_ASSET_ROUTES,
  ORGANIZE_MEMORY_STATUS_ROUTES,
  findMemoryAssetRoute,
  isMemoryAssetKey,
  isMemoryStatusKey,
  type MemoryAssetKey,
  type MemoryStatusKey,
} from './organizeRoutes'
import { saveOrganizeEditorDraft, type OrganizeEditorDraft } from './editorDraftStorage'
import {
  DISCOVER_CATEGORIES,
  discoverCategoryLabel,
  normalizeDiscoverCategory,
} from './discoverCategories'
import OrganizeOutputUploadDrawer from './components/OrganizeOutputUploadDrawer.vue'
import PublicKnowledgeBaseDiscover from './components/PublicKnowledgeBaseDiscover.vue'
import OrganizeCourseDiscover from './OrganizeCourseDiscover.vue'

type MemoryType = 'note' | 'record' | 'audio' | 'audio-card'
type MemoryOrganizationStatus = MemoryStatusKey
type OutputKind = 'all' | 'article' | 'video' | 'audio'
type OutputCreateKind = Exclude<OutputKind, 'all'>
type OutputStatusFilter = 'all' | OrganizeOutputStatus
type OutputViewMode = 'list' | 'grid'
type MemoryCreateAction = 'new-note' | 'import-file'

interface MemoryItem {
  id: string
  time: string
  type: MemoryType
  typeLabel: string
  title: string
  content: string
  summary?: string
  source?: string
  duration?: string
  occurredAt?: string
  durationSeconds?: number
  metadata?: Record<string, unknown>
  persisted: boolean
}

interface MemoryListItem extends MemoryItem {
  date: string
}

interface MemoryGroup {
  date: string
  items: MemoryItem[]
}

interface MemoryDisplayGroup {
  date: string
  showDate: boolean
  items: Array<MemoryItem | MemoryListItem>
}

type MemoryMenuAction = 'edit' | 'organize' | 'delete'

interface OutputItem {
  id: string
  title: string
  content: string
  type: string
  kind: Exclude<OutputKind, 'all'>
  kindLabel: string
  contentTypeLabel: string
  seriesLabel: string
  categoryLabel: string
  source: string
  summary: string
  updated: string
  createdAtLabel: string
  coverUrl?: string
  status: string
  statusKey: OrganizeOutputStatus
  statusLabel: string
  icon: string
  tags: string[]
  memoryIds: string[]
  creatorId?: string
  creatorName?: string
  creatorAvatar?: string
  subscribedByMe?: boolean
  metadata?: Record<string, unknown>
  persisted: boolean
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const keyword = ref('')
const memoryLoading = ref(true)
const memoryImporting = ref(false)
const memoryImportInputRef = ref<HTMLInputElement | null>(null)
const memoryOrganizationJobs = ref<OrganizeJob[]>([])
const memoryOrganizationOutputs = ref<OrganizeOutput[]>([])
const discoverFeaturedLoading = ref(true)
const discoverFeedLoading = ref(true)
const activeMemoryMenuId = ref('')
const organizingMemoryIds = ref<Set<string>>(new Set())
const outputUploadVisible = ref(false)
const outputUploadInitialKind = ref<OutputCreateKind>('article')
const outputPreviewVisible = ref(false)
const activeOutputPreview = ref<OutputItem | null>(null)
const outputStatusSaving = ref(false)
const initialDiscoverTab = typeof route.query.tab === 'string'
  ? route.query.tab
  : ''
const discoverTab = ref(initialDiscoverTab || 'recommended')
const discoverMode = ref<'content' | 'knowledge-base'>('content')
const discoverTabs = ref<OrganizeDiscoverTab[]>([
  { label: '推荐', value: 'recommended' },
  { label: '系列课程', value: 'course' },
  ...DISCOVER_CATEGORIES.map((category) => ({ label: category.label, value: category.key })),
])
const featuredRotation = ref(0)
const OUTPUT_PAGE_SIZE = 30
const FEATURED_OUTPUT_SIZE = 4
const outputPage = ref(1)
const editableOutputStatusOptions: Array<{ label: string; value: OrganizeOutputStatus }> = [
  { label: '草稿', value: 'draft' },
  { label: '待确认', value: 'review' },
  { label: '已发布', value: 'ready' },
  { label: '已归档', value: 'archived' },
]
const outputCreateOptions = [
  {
    content: '新建图文',
    value: 'article',
    prefixIcon: () => h(TIcon, { name: 'file-word', size: '16px' }),
  },
  {
    content: '新建视频',
    value: 'video',
    prefixIcon: () => h(TIcon, { name: 'play-circle', size: '16px' }),
  },
  {
    content: '新建音频',
    value: 'audio',
    prefixIcon: () => h(TIcon, { name: 'sound', size: '16px' }),
  },
]
const memoryCreateOptions = [
  {
    content: '新建笔记',
    value: 'new-note',
    prefixIcon: () => h(TIcon, { name: 'edit-1', size: '16px' }),
  },
  {
    content: '导入文件',
    value: 'import-file',
    prefixIcon: () => h(TIcon, { name: 'upload', size: '16px' }),
  },
]
const memoryImportAccept = [
  '.pdf',
  '.doc',
  '.docx',
  '.epub',
  '.mhtml',
  '.ppt',
  '.pptx',
  '.md',
  '.markdown',
  '.txt',
  '.csv',
  '.json',
  '.xlsx',
  '.xls',
  '.png',
  '.jpg',
  '.jpeg',
  '.gif',
  '.mp3',
  '.wav',
  '.m4a',
  '.flac',
  '.ogg',
  '.mp4',
  '.mov',
  '.avi',
  '.mkv',
  '.webm',
  '.wmv',
  '.flv',
].join(',')
const activeTab = computed<'memory' | 'output'>(() => {
  const tab = route.meta.organizeTab
  if (tab === 'discover') return 'output'
  return 'memory'
})

const activeMemoryAsset = computed<MemoryAssetKey | ''>(() => {
  if (activeTab.value !== 'memory') return ''
  const asset = route.meta.memoryAsset
  return isMemoryAssetKey(asset) ? asset : ''
})

const activeMemoryStatus = computed<MemoryStatusKey | ''>(() => {
  if (activeTab.value !== 'memory') return ''
  const status = route.query.status
  return isMemoryStatusKey(status) ? status : ''
})

const escapeHtml = (value: string) => {
  const node = document.createElement('div')
  node.textContent = value
  return node.innerHTML
}

const placeholderContent = (title: string) => `<p>${escapeHtml(title)}</p><p></p>`

const memoryGroups = ref<MemoryGroup[]>([])
const outputs = ref<OutputItem[]>([])
const featuredOutputs = ref<OutputItem[]>([])
const discoverTotal = ref(0)
let discoverFeedRequestSeq = 0
let discoverFeaturedRequestSeq = 0

const allMemoryItems = computed<MemoryListItem[]>(() => {
  return memoryGroups.value.flatMap((group) => group.items.map((item) => ({ ...item, date: group.date })))
})

const processingJobStatuses = new Set<OrganizeJobStatus>(['queued', 'running', 'repairing'])
const organizedJobStatuses = new Set<OrganizeJobStatus>(['completed', 'fallback'])

const memoryOrganizeActionLabel = (item: MemoryItem | MemoryListItem) => {
  if (isMemoryOrganizeCreating(item.id)) return '整理中...'
  const job = memoryOrganizationJobs.value
    .filter((candidate) => (candidate.memory_ids || []).includes(item.id))
    .sort((left, right) => new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime())[0]
  if (job && processingJobStatuses.has(job.status)) return '整理中...'
  if (job && organizedJobStatuses.has(job.status)) return '查看整理结果'
  return '整理'
}

const memoryOrganizationStatus = (memoryID: string): MemoryOrganizationStatus => {
  const jobs = memoryOrganizationJobs.value.filter((job) => (job.memory_ids || []).includes(memoryID))
  const hasProcessingJob = jobs.some((job) => processingJobStatuses.has(job.status))
  const hasOrganizedJob = jobs.some((job) => organizedJobStatuses.has(job.status))
  const hasOutput = memoryOrganizationOutputs.value.some((output) => (output.memory_ids || []).includes(memoryID))

  if (organizingMemoryIds.value.has(memoryID) || hasProcessingJob) return 'processing'
  if (hasOrganizedJob || hasOutput) return 'organized'
  return 'unorganized'
}

const memoryStatusCards = computed(() => {
  const items = allMemoryItems.value
  return ORGANIZE_MEMORY_STATUS_ROUTES.map((status) => ({
    ...status,
    count: items.filter((item) => memoryOrganizationStatus(item.id) === status.key).length,
  }))
})

const activeMemoryStatusMeta = computed(() => {
  return (
    memoryStatusCards.value.find((status) => status.key === activeMemoryStatus.value) ||
    ORGANIZE_MEMORY_STATUS_ROUTES[0]
  )
})

const activeMemoryAssetMeta = computed(() => {
  return ORGANIZE_MEMORY_ASSET_ROUTES.find((asset) => asset.key === activeMemoryAsset.value) || ORGANIZE_MEMORY_ASSET_ROUTES[0]
})

const activeMemoryFilterMeta = computed(() => {
  return activeMemoryStatus.value ? activeMemoryStatusMeta.value : activeMemoryAssetMeta.value
})

const openMemoryStatusList = async (status: MemoryStatusKey) => {
  await router.push({
    path: '/platform/organize/memory',
    query: { status },
  })
}

const openMemoryAssetList = async (asset: MemoryAssetKey) => {
  const nextRoute = findMemoryAssetRoute(asset)
  if (nextRoute && route.path !== nextRoute.path) await router.push(nextRoute.path)
}

const openMemoryOverview = async () => {
  const memoryPath = '/platform/organize/memory'
  if (route.path !== memoryPath || route.query.status) await router.push(memoryPath)
}

const activeMeta = computed(() => {
  if (activeTab.value === 'output') {
    return { title: '发现', actionIcon: 'file-add', actionLabel: '新建' }
  }
  if (activeMemoryAsset.value || activeMemoryStatus.value) {
    return { title: `${activeMemoryFilterMeta.value.label}列表`, actionIcon: 'add', actionLabel: '添加记忆' }
  }
  return { title: '记忆', actionIcon: 'add', actionLabel: '添加记忆' }
})

const currentUserId = computed(() => authStore.currentUserId || authStore.user?.id || '')

const filteredMemoryGroups = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return memoryGroups.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        const keywordMatched = !q || memorySearchText(item).includes(q)
        return keywordMatched
      }),
    }))
    .filter((group) => group.items.length > 0)
})

const filteredMemoryAssetItems = computed(() => {
  const asset = activeMemoryAsset.value
  const q = keyword.value.trim().toLowerCase()
  return allMemoryItems.value.filter((item) => {
    const assetMatched =
      asset === 'note'
        ? item.type === 'note' || item.type === 'record'
        : asset === 'audio'
          ? item.type === 'audio'
          : item.type === 'audio-card'
    return assetMatched && (!q || memorySearchText(item).includes(q))
  })
})

const filteredMemoryStatusItems = computed(() => {
  const status = activeMemoryStatus.value
  const q = keyword.value.trim().toLowerCase()
  return allMemoryItems.value.filter((item) => {
    return (
      status &&
      memoryOrganizationStatus(item.id) === status &&
      (!q || memorySearchText(item).includes(q))
    )
  })
})

const visibleMemoryListGroups = computed<MemoryDisplayGroup[]>(() => {
  if (activeMemoryStatus.value) {
    return [
      {
        date: activeMemoryStatus.value,
        showDate: false,
        items: filteredMemoryStatusItems.value,
      },
    ]
  }

  if (activeMemoryAsset.value) {
    return [
      {
        date: activeMemoryAsset.value,
        showDate: false,
        items: filteredMemoryAssetItems.value,
      },
    ]
  }

  return filteredMemoryGroups.value.map((group) => ({
    date: group.date,
    showDate: true,
    items: group.items,
  }))
})

const visibleMemoryListEmpty = computed(() => visibleMemoryListGroups.value.every((group) => group.items.length === 0))

const memoryListEmptyText = computed(() => {
  if (activeMemoryStatus.value) return `暂无${activeMemoryStatusMeta.value.label}记忆`
  return activeMemoryAsset.value ? `暂无${activeMemoryAssetMeta.value.label}` : '暂无记忆'
})

const activeDiscoverTabLabel = computed(() => {
  return discoverTabs.value.find((tab) => tab.value === discoverTab.value)?.label || '推荐'
})

const showFeaturedCourseFallback = computed(() => {
  return !discoverFeaturedLoading.value && featuredOutputs.value.length === 0
})

const courseDiscoverRef = ref<{ reload: () => Promise<void> } | null>(null)
const featuredCourseDiscoverRef = ref<{ reload: () => Promise<void> } | null>(null)

const setDiscoverTab = (tab: string) => {
  if (discoverTab.value === tab) return
  discoverTab.value = tab
  outputPage.value = 1
  void loadDiscoverFeedData({ tab, page: 1, resetPage: true }).catch(() => {
    MessagePlugin.warning('发现数据刷新失败')
  })
}

const rotateFeaturedOutputs = () => {
  const total = featuredOutputs.value.length
  if (total <= 1) return
  featuredRotation.value = (featuredRotation.value + 2) % total
  void loadDiscoverFeaturedData().catch(() => {
    MessagePlugin.warning('精选数据刷新失败')
  })
}

const handleDiscoverPageChange = (pageInfo: { current: number; pageSize: number }) => {
  outputPage.value = pageInfo.current
  void loadDiscoverFeedData({ tab: discoverTab.value, page: pageInfo.current, pageSize: pageInfo.pageSize }).catch(() => {
    MessagePlugin.warning('发现数据刷新失败')
  })
}

const outputEmptyText = computed(() => {
  const tabLabel = discoverTab.value === 'recommended' ? '' : activeDiscoverTabLabel.value
  return tabLabel ? `暂无${tabLabel}内容` : '暂无发现'
})

const outputPreviewFileName = computed(() => {
  const item = activeOutputPreview.value
  if (!item) return ''
  return asTrimmedString(item.metadata?.file_name) || item.title
})

const outputPreviewFileType = computed(() => {
  const item = activeOutputPreview.value
  if (!item) return ''
  return asTrimmedString(item.metadata?.file_type).toUpperCase()
})

const outputPreviewFileTypeForPreview = computed(() => {
  const explicitType = outputPreviewFileType.value.toLowerCase()
  if (explicitType) return explicitType
  const name = outputPreviewFileName.value
  const dotIndex = name.lastIndexOf('.')
  return dotIndex >= 0 ? name.slice(dotIndex + 1).toLowerCase() : ''
})

const outputPreviewSourcePath = computed(() => asTrimmedString(activeOutputPreview.value?.metadata?.file_path))

const outputPreviewSourceUrl = computed(() => {
  if (!outputPreviewSourcePath.value) return ''
  return `/files?${new URLSearchParams({ file_path: outputPreviewSourcePath.value }).toString()}`
})

const outputCardCoverStyle = (item: OutputItem) => {
  if (!item.coverUrl) return undefined
  return {
    backgroundImage: `linear-gradient(180deg, rgba(255, 255, 255, 0.16) 0%, rgba(255, 255, 255, 0.08) 100%), url(${JSON.stringify(item.coverUrl)})`,
  }
}

const canEditOutputItem = (item?: OutputItem | null) => {
  return Boolean(item && item.persisted && item.creatorId && currentUserId.value && item.creatorId === currentUserId.value)
}

const canEditActiveOutputPreview = computed(() => {
  return canEditOutputItem(activeOutputPreview.value)
})

const getOutputStatusMenuOptions = (currentStatus?: OrganizeOutputStatus) => {
  return [
    ...editableOutputStatusOptions.map((option) => ({
      content: option.label,
      value: option.value,
      disabled: option.value === currentStatus,
      prefixIcon: () =>
        h(TIcon, {
          name: option.value === currentStatus ? 'check-circle-filled' : 'check-circle',
          size: '16px',
        }),
    })),
    {
      content: '删除',
      value: 'delete',
      theme: 'error' as const,
      prefixIcon: () => h(TIcon, { name: 'delete', size: '16px' }),
    },
  ]
}

const formatDateLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '未知日期'
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const formatTimeLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--'
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}

const formatUpdatedLabel = (value: string) => `${formatDateLabel(value)} ${formatTimeLabel(value)}`

const formatDuration = (seconds?: number) => {
  const safeSeconds = Math.max(0, seconds || 0)
  const minutes = Math.floor(safeSeconds / 60)
  return `${String(minutes).padStart(2, '0')}:${String(safeSeconds % 60).padStart(2, '0')}`
}

const memoryTypeFromApi = (kind: OrganizeMemoryKind): MemoryType => (kind === 'audio_card' ? 'audio-card' : kind)
const memoryTypeToApi = (kind: MemoryType): OrganizeMemoryKind => (kind === 'audio-card' ? 'audio_card' : kind)

const memoryTypeLabel = (type: MemoryType) => {
  if (type === 'audio') return '录音'
  if (type === 'audio-card') return '工牌'
  return type === 'record' ? '记录' : '笔记'
}

const memoryReferenceKindLabel = (kind?: string) => {
  if (kind === 'audio') return '录音'
  if (kind === 'audio_card') return '工牌'
  if (kind === 'record') return '记录'
  return '笔记'
}

const statusLabel = (status: OrganizeOutputStatus) => ({ draft: '草稿', review: '评审中', ready: '可交付', archived: '已归档' })[status]
const outputKindLabelMap: Record<Exclude<OutputKind, 'all'>, string> = {
  article: '图文类',
  video: '视频类',
  audio: '音频类',
}
const outputKindIconMap: Record<Exclude<OutputKind, 'all'>, string> = {
  article: 'file-word',
  video: 'play-circle',
  audio: 'sound',
}

const asTrimmedString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const normalizeOneLineText = (value: string) => value.replace(/\s+/g, ' ').trim()

const compactText = (value: string, maxLength = 96) => {
  const text = normalizeOneLineText(value)
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

const outputCoverUrl = (item: OrganizeOutput) => {
  const metadata = item.metadata || {}
  return (
    asTrimmedString(metadata.cover_url) ||
    asTrimmedString(metadata.cover) ||
    asTrimmedString(metadata.thumbnail_url) ||
    asTrimmedString(metadata.thumbnail) ||
    asTrimmedString(metadata.poster_url) ||
    asTrimmedString(metadata.poster)
  )
}

const isTruthyFlag = (value: unknown) => value === true || value === 'true' || value === 1 || value === '1'

const outputKindFromValue = (value: unknown): Exclude<OutputKind, 'all'> => {
  const normalized = asTrimmedString(value).toLowerCase()
  if (normalized === 'video' || normalized === '视频类') return 'video'
  if (normalized === 'audio' || normalized === '音频类') return 'audio'
  return 'article'
}

const normalizeOutputTags = (value: unknown) => {
  if (Array.isArray(value)) {
    return value.map((item) => asTrimmedString(item)).filter(Boolean)
  }
  if (typeof value === 'string') {
    return value.split(/[，,;；\n]/).map((item) => item.trim()).filter(Boolean)
  }
  return []
}

const memorySearchText = (item: MemoryItem) => [
  item.title,
  item.typeLabel,
  item.summary,
  item.source,
  item.content,
].filter(Boolean).join(' ').toLowerCase()

const memoryTypeIcon = (item: MemoryItem) => {
  if (item.type === 'audio') return 'sound'
  if (item.type === 'audio-card') return 'file'
  return 'file-word'
}

const memoryCardKind = (item: MemoryItem) => {
  if (item.type === 'audio') return 'audio'
  if (item.type === 'audio-card') return 'audio-card'
  return 'note'
}

const memoryKindLabel = (item: MemoryItem) => {
  if (item.type === 'audio-card') return '工牌'
  return item.typeLabel || '笔记'
}

const memoryPlainText = (item: MemoryItem) => {
  const summary = asTrimmedString(item.summary)
  const content = contentText(item.content, item.title)
  return normalizeOneLineText(summary || content || item.title)
}

const memoryCardBodyText = (item: MemoryItem) => {
  const text = memoryPlainText(item)
  if (!text || text === item.title) return ''
  return compactText(text, item.type === 'audio' ? 140 : 180)
}

const memoryAudioDurationText = (item: MemoryItem) => {
  const fromSeconds = item.durationSeconds
  if (typeof fromSeconds === 'number' && Number.isFinite(fromSeconds)) {
    const safeSeconds = Math.max(0, Math.floor(fromSeconds))
    const minutes = Math.floor(safeSeconds / 60)
    const seconds = safeSeconds % 60
    return `${minutes}分${seconds}秒`
  }

  const duration = asTrimmedString(item.duration)
  const match = duration.match(/^(\d{1,2}):(\d{2})$/)
  if (match) {
    return `${Number(match[1])}分${Number(match[2])}秒`
  }
  return duration || '0分0秒'
}

const memoryCardTimeLabel = (item: MemoryItem | MemoryListItem, includeDate: boolean) => {
  const date = 'date' in item ? item.date : ''
  return includeDate && date ? `${date} ${item.time}` : item.time
}

const memoryCardFooterInfo = (item: MemoryItem) => {
  const parts: string[] = []
  if (item.source) parts.push(item.source)
  if (item.type === 'audio') parts.push(`录音时长 ${memoryAudioDurationText(item)}`)
  return parts.join(' · ')
}

const memoryDisplayTitle = (item: OrganizeMemory, type: MemoryType) => {
  if (type !== 'audio') return item.title
  const metadata = item.metadata || {}
  const explicitTitle =
    asTrimmedString(metadata.title) ||
    asTrimmedString(metadata.extracted_title) ||
    asTrimmedString(metadata.ai_title) ||
    asTrimmedString(metadata.summary_title)
  if (explicitTitle) return explicitTitle

  const derivedTitle = compactText(memorySummary(item), 24)
  if (derivedTitle) return derivedTitle

  return (
    item.title
  )
}

const memorySummary = (item: OrganizeMemory) => {
  const metadata = item.metadata || {}
  const explicitSummary =
    asTrimmedString(metadata.summary) ||
    asTrimmedString(metadata.source_summary) ||
    asTrimmedString(metadata.description) ||
    asTrimmedString(metadata.abstract)
  if (explicitSummary) return compactText(explicitSummary)

  const contentSummary = contentExcerpt(item.content, '')
  if (contentSummary) return contentSummary

  const transcript =
    asTrimmedString(metadata.transcript) ||
    asTrimmedString(metadata.transcription) ||
    asTrimmedString(metadata.asr_text)
  return transcript ? compactText(transcript) : ''
}

const currentUserDisplayName = () => authStore.user?.username || authStore.user?.email || '我'

const outputCreatorId = (item: OrganizeOutput) => {
  const metadata = item.metadata || {}
  return asTrimmedString(item.user_id)
    || asTrimmedString(metadata.creator_id)
    || asTrimmedString(metadata.created_by)
    || asTrimmedString(metadata.user_id)
}

const outputCreatorName = (item: OrganizeOutput) => {
  const metadata = item.metadata || {}
  const explicitName = asTrimmedString(item.creator_name)
    || asTrimmedString(metadata.creator_name)
    || asTrimmedString(metadata.creator_username)
    || asTrimmedString(metadata.author_name)
    || asTrimmedString(metadata.user_name)
  if (explicitName) return explicitName
  const creatorId = outputCreatorId(item)
  if (!creatorId || creatorId === currentUserId.value) return currentUserDisplayName()
  return '未知用户'
}

const outputCreatorAvatar = (item: OrganizeOutput) => {
  const metadata = item.metadata || {}
  const explicitAvatar = asTrimmedString(item.creator_avatar)
    || asTrimmedString(metadata.creator_avatar)
    || asTrimmedString(metadata.author_avatar)
    || asTrimmedString(metadata.user_avatar)
  if (explicitAvatar) return explicitAvatar
  const creatorId = outputCreatorId(item)
  if (!creatorId || creatorId === currentUserId.value) return authStore.user?.avatar || ''
  return ''
}

const isOutputSubscribedByMe = (item: OrganizeOutput) => {
  const metadata = item.metadata || {}
  return isTruthyFlag(item.is_subscribed)
    || isTruthyFlag(metadata.is_subscribed)
    || isTruthyFlag(metadata.subscribed)
    || isTruthyFlag(metadata.subscribed_by_me)
}

const creatorInitial = (name: string) => {
  const normalized = name.trim()
  return normalized ? normalized.slice(0, 1).toUpperCase() : '创'
}

const outputCreatorDisplayName = (item: OutputItem) => {
  if (item.creatorName) return item.creatorName
  return !item.creatorId || item.creatorId === currentUserId.value ? currentUserDisplayName() : '未知用户'
}

const outputCreatorDisplayAvatar = (item: OutputItem) => {
  if (item.creatorAvatar) return item.creatorAvatar
  return !item.creatorId || item.creatorId === currentUserId.value ? authStore.user?.avatar || '' : ''
}

const outputKindDisplayLabel = (kind: OutputKind) => {
  if (kind === 'all') return '全部'
  return outputKindLabelMap[kind]
}

const outputTotalPages = computed(() => Math.max(1, Math.ceil(discoverTotal.value / OUTPUT_PAGE_SIZE)))

const paginatedOutputs = computed(() => {
  return outputs.value
})

watch(discoverTotal, () => {
  if (outputPage.value > outputTotalPages.value) {
    outputPage.value = outputTotalPages.value
  }
  if (outputPage.value < 1) {
    outputPage.value = 1
  }
})

const mapMemory = (item: OrganizeMemory): MemoryListItem => {
  const type = memoryTypeFromApi(item.kind)
  return {
    id: item.id,
    date: formatDateLabel(item.occurred_at),
    time: formatTimeLabel(item.occurred_at),
    type,
    typeLabel: memoryTypeLabel(type),
    title: memoryDisplayTitle(item, type),
    content: item.content || placeholderContent(item.title),
    summary: type === 'audio' ? memorySummary(item) : undefined,
    source: item.source,
    duration: type === 'audio' ? formatDuration(item.duration_seconds) : undefined,
    occurredAt: item.occurred_at,
    durationSeconds: item.duration_seconds,
    metadata: item.metadata,
    persisted: true,
  }
}

const mapOutput = (item: OrganizeOutput): OutputItem => ({
  id: item.id,
  title: item.title,
  content: item.content || placeholderContent(item.title),
  type: item.output_type || '图文类',
  kind: outputKindFromValue(item.metadata?.content_kind || item.output_type || item.icon),
  kindLabel: outputKindDisplayLabel(outputKindFromValue(item.metadata?.content_kind || item.output_type || item.icon)),
  contentTypeLabel: item.public_content_type === 'course' ? '学习课程' : '图文内容',
  seriesLabel: item.series_title
    ? `${item.series_title}${item.series_order ? ` · 第 ${item.series_order} 节` : ''}`
    : '',
  categoryLabel: discoverCategoryLabel(item.metadata?.discover_category || item.metadata?.discover_category_label),
  source: item.memory_count ? `来自 ${item.memory_count} 条记忆` : '手动创建',
  summary: item.source_summary || asTrimmedString(item.metadata?.summary) || contentExcerpt(item.content, '暂无摘要'),
  updated: formatUpdatedLabel(item.updated_at),
  createdAtLabel: formatUpdatedLabel(item.created_at),
  coverUrl: outputCoverUrl(item),
  status: statusLabel(item.status),
  statusKey: item.status,
  statusLabel: statusLabel(item.status),
  icon: item.icon || outputKindIconMap[outputKindFromValue(item.metadata?.content_kind || item.output_type || item.icon)],
  tags: normalizeOutputTags(item.metadata?.tags),
  memoryIds: item.memory_ids || [],
  creatorId: outputCreatorId(item),
  creatorName: outputCreatorName(item),
  creatorAvatar: outputCreatorAvatar(item),
  subscribedByMe: isOutputSubscribedByMe(item),
  metadata: item.metadata,
  persisted: true,
})

const syncActiveOutputPreview = () => {
  if (!activeOutputPreview.value) return
  const nextPreview = [...outputs.value, ...featuredOutputs.value].find(
    (item) => item.id === activeOutputPreview.value?.id,
  )
  if (nextPreview) {
    activeOutputPreview.value = nextPreview
  } else {
    activeOutputPreview.value = null
    outputPreviewVisible.value = false
  }
}

const loadMemoryData = async () => {
  memoryLoading.value = true
  try {
    const response = await listOrganizeMemories({ page_size: 100 })
    if (!response.success || !response.data) {
      throw new Error(response.message || '记忆数据加载失败')
    }

    const nextGroups = groupMemoryItems(response.data.items.map(mapMemory))
    memoryGroups.value = nextGroups
    return nextGroups
  } finally {
    memoryLoading.value = false
  }
}

const loadMemoryOrganizationState = async () => {
  const [jobsResponse, outputsResponse] = await Promise.all([
    listOrganizeJobs({ page_size: 100 }),
    listOrganizeOutputs({ page_size: 100 }),
  ])
  if (!jobsResponse.success || !jobsResponse.data) {
    throw new Error(jobsResponse.message || '整理任务加载失败')
  }
  if (!outputsResponse.success || !outputsResponse.data) {
    throw new Error(outputsResponse.message || '整理产物加载失败')
  }
  memoryOrganizationJobs.value = jobsResponse.data.items
  memoryOrganizationOutputs.value = outputsResponse.data.items
}

// 顶部精选和底部分页列表分开拉取，切 tab 只刷新底部当前页。
const loadDiscoverFeaturedData = async () => {
  const requestSeq = ++discoverFeaturedRequestSeq
  discoverFeaturedLoading.value = true
  try {
    const response = await getOrganizeDiscover({
      tab: 'recommended',
      featured_offset: featuredRotation.value,
      page: 1,
      page_size: FEATURED_OUTPUT_SIZE,
    })

    if (requestSeq !== discoverFeaturedRequestSeq) return
    if (!response.success || !response.data) {
      throw new Error(response.message || '发现精选加载失败')
    }

    const data = response.data
    discoverTabs.value = data.tabs.length ? data.tabs : discoverTabs.value
    featuredOutputs.value = (data.featured_outputs || []).map(mapOutput)
    syncActiveOutputPreview()
  } finally {
    if (requestSeq === discoverFeaturedRequestSeq) {
      discoverFeaturedLoading.value = false
    }
  }
}

const loadDiscoverFeedData = async (options?: { tab?: string; page?: number; pageSize?: number; resetPage?: boolean }) => {
  const requestSeq = ++discoverFeedRequestSeq
  const tab = options?.tab ?? discoverTab.value
  const page = Math.max(1, options?.page ?? outputPage.value)
  const pageSize = Math.max(1, options?.pageSize ?? OUTPUT_PAGE_SIZE)
  discoverFeedLoading.value = true
  try {
    const response = await getOrganizeDiscover({
      tab,
      page,
      page_size: pageSize,
    })

    if (requestSeq !== discoverFeedRequestSeq) return
    if (!response.success || !response.data) {
      throw new Error(response.message || '发现数据加载失败')
    }

    const data = response.data
    discoverTabs.value = data.tabs.length ? data.tabs : discoverTabs.value
    outputs.value = (data.items || []).map(mapOutput)
    discoverTotal.value = data.total
    outputPage.value = data.page || page
    if (options?.resetPage) {
      outputPage.value = data.page || 1
    }

    syncActiveOutputPreview()
  } finally {
    if (requestSeq === discoverFeedRequestSeq) {
      discoverFeedLoading.value = false
    }
  }
}

const refreshDiscoverData = (options?: { resetPage?: boolean }) => {
  return Promise.allSettled([
    loadDiscoverFeaturedData(),
    loadDiscoverFeedData({
      tab: discoverTab.value,
      page: options?.resetPage ? 1 : outputPage.value,
      resetPage: options?.resetPage,
    }),
    discoverTab.value === 'course'
      ? courseDiscoverRef.value?.reload?.() || Promise.resolve()
      : Promise.resolve(),
    featuredCourseDiscoverRef.value?.reload?.() || Promise.resolve(),
  ]).then((results) => {
    if (results.some((result) => result.status === 'rejected')) {
      MessagePlugin.warning('发现数据刷新失败')
    }
  })
}

const groupMemoryItems = (items: MemoryListItem[]) => {
  const groups: MemoryGroup[] = []
  items.forEach(({ date, ...item }) => {
    let group = groups.find((candidate) => candidate.date === date)
    if (!group) {
      group = { date, items: [] }
      groups.push(group)
    }
    group.items.push(item)
  })
  return groups
}

const loadOrganizeData = async () => {
  const results = await Promise.allSettled([
    loadMemoryData(),
    loadMemoryOrganizationState(),
    loadDiscoverFeaturedData(),
    loadDiscoverFeedData({ tab: discoverTab.value, page: 1, resetPage: true }),
  ])

  const [memoryResult, memoryStateResult, featuredResult, feedResult] = results
  if (featuredResult.status === 'rejected' || feedResult.status === 'rejected') {
    MessagePlugin.warning('发现数据加载失败')
  }

  if (memoryResult.status === 'rejected') {
    MessagePlugin.warning('记忆数据加载失败')
  }
  if (memoryStateResult.status === 'rejected') {
    MessagePlugin.warning('记忆整理状态加载失败')
  }
}

const contentText = (html: string, fallback: string) => {
  if (!html) return fallback
  const body = new DOMParser().parseFromString(html, 'text/html').body
  const firstBlock = Array.from(body.children)[0]
  if (firstBlock?.tagName.toLowerCase() === 'h1') {
    firstBlock.remove()
  }
  const parsed = body.textContent?.trim() || ''
  return parsed || fallback
}

const contentExcerpt = (html: string, fallback: string) => {
  const parsed = contentText(html, fallback)
  return parsed.length > 96 ? `${parsed.slice(0, 96)}...` : parsed || fallback
}

type EditorDocumentType = 'memory' | 'output'

const editorPath = (documentType: EditorDocumentType, id: string) => {
  return `/platform/organize/editor/${documentType}/${encodeURIComponent(id)}`
}

const editorDraft = (item: MemoryItem | OutputItem | undefined): OrganizeEditorDraft | null => {
  if (!item || item.persisted) return null

  const draft: OrganizeEditorDraft = {
    title: item.title,
    content: item.content,
    metadata: item.metadata,
  }
  if ('statusKey' in item) {
    draft.output_type = item.type
    draft.status = item.statusKey
    draft.source_summary = item.summary
    draft.icon = item.icon
    draft.memory_ids = item.memoryIds
    draft.metadata = {
      ...draft.metadata,
      ...(item.categoryLabel
        ? {
            discover_category: normalizeDiscoverCategory(item.categoryLabel),
            discover_category_label: item.categoryLabel,
          }
        : {}),
      ...(item.tags.length ? { tags: item.tags } : {}),
    }
  } else {
    draft.kind = memoryTypeToApi(item.type)
    draft.duration_seconds = item.durationSeconds
    if (item.source) draft.source = item.source
  }
  return draft
}

const openDocumentEditor = async (
  documentType: EditorDocumentType,
  id = 'new',
  item?: MemoryItem | OutputItem,
) => {
  const draft = editorDraft(item)
  if (draft) {
    saveOrganizeEditorDraft(documentType, id, draft)
  }
  await router.push({ path: editorPath(documentType, id) })
}

const createActiveDocument = () => {
  if (activeTab.value === 'output') {
    void openDocumentEditor('output')
    return
  }
  void openDocumentEditor('memory')
}

const importMemoryFile = async (files: File[]) => {
  if (memoryImporting.value) return
  if (!files.length) return
  memoryImporting.value = true
  try {
    const response = await uploadOrganizeMemory(files)
    if (!response.success || !response.data) {
      throw new Error(response.message || '文件导入失败')
    }

    await loadMemoryData()
    MessagePlugin.success(files.length > 1 ? `已导入 ${files.length} 个文件，正在解析` : '已导入，正在解析')
    const imported = mapMemory(response.data)
    await openDocumentEditor('memory', imported.id, imported)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '文件导入失败')
  } finally {
    memoryImporting.value = false
  }
}

const handleMemoryCreateAction = (data: { value: string | number | boolean }) => {
  const action = String(data.value) as MemoryCreateAction
  if (action === 'new-note') {
    createActiveDocument()
    return
  }
  if (action === 'import-file') {
    memoryImportInputRef.value?.click()
  }
}

const handleMemoryImportFileChange = (event: Event) => {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (!files.length) return
  void importMemoryFile(files)
}

const isEditableMemory = (item: MemoryItem) =>
  item.type === 'note' || item.type === 'record' || item.type === 'audio' || item.type === 'audio-card'

const openMemoryEditor = (item: MemoryListItem | MemoryItem) => {
  if (!isEditableMemory(item)) return
  void openDocumentEditor('memory', item.id, item)
}

const handleMemoryMenuVisible = (id: string, visible: boolean) => {
  activeMemoryMenuId.value = visible ? id : ''
}

const closeMemoryMenu = () => {
  activeMemoryMenuId.value = ''
}

const removeMemoryFromGroups = (id: string) => {
  memoryGroups.value = memoryGroups.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => item.id !== id),
    }))
    .filter((group) => group.items.length > 0)
}

const deleteMemoryItem = (item: MemoryItem) => {
  const dialog = DialogPlugin.confirm({
    header: '删除记忆',
    body: `确认删除「${item.title || '无标题'}」？删除后无法恢复。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: { content: '取消' },
    onConfirm: async () => {
      try {
        if (item.persisted) {
          const response = await deleteOrganizeMemory(item.id)
          if (response?.success === false) {
            throw new Error(response.message || '删除失败')
          }
        }
        removeMemoryFromGroups(item.id)
        MessagePlugin.success('已删除')
        dialog.destroy()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '删除失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

const setMemoryOrganizeCreating = (id: string, creating: boolean) => {
  const next = new Set(organizingMemoryIds.value)
  if (creating) {
    next.add(id)
  } else {
    next.delete(id)
  }
  organizingMemoryIds.value = next
}

const isMemoryOrganizeCreating = (id: string) => organizingMemoryIds.value.has(id)

const latestMemoryOrganizeJob = (id: string) =>
  memoryOrganizationJobs.value
    .filter((job) => (job.memory_ids || []).includes(id))
    .sort((left, right) => new Date(right.updated_at).getTime() - new Date(left.updated_at).getTime())[0]

const isMemoryOrganizeDisabled = (id: string) => {
  if (isMemoryOrganizeCreating(id)) return true
  const job = latestMemoryOrganizeJob(id)
  return Boolean(job && processingJobStatuses.has(job.status))
}

const createOrganizeFromMemory = async (item: MemoryItem) => {
  if (isMemoryOrganizeDisabled(item.id)) return
  if (!item.persisted) {
    MessagePlugin.warning('请先保存记忆')
    return
  }

  const latestJob = latestMemoryOrganizeJob(item.id)
  if (latestJob && processingJobStatuses.has(latestJob.status)) {
    MessagePlugin.info('整理任务正在处理中')
    return
  }
  if (latestJob && organizedJobStatuses.has(latestJob.status) && latestJob.output_id) {
    await router.push({
      path: `/platform/organize/outputs/${encodeURIComponent(latestJob.output_id)}`,
      query: { from: 'memory', memoryId: item.id },
    })
    return
  }

  setMemoryOrganizeCreating(item.id, true)
  try {
    const configResponse = await listOrganizeConfigs({
      page: 1,
      page_size: 100,
      status: 'active',
    })
    if (!configResponse.success) {
      throw new Error(configResponse.message || '整理配置加载失败')
    }
    const config = configResponse.data?.items?.[0]
    if (!config) {
      MessagePlugin.warning('请先在整理工作台创建启用的整理配置')
      return
    }

    const response = await createOrganizeJob({
      config_id: config.id,
      memory_ids: [item.id],
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '整理任务创建失败')
    }

    memoryOrganizationJobs.value = [
      response.data,
      ...memoryOrganizationJobs.value.filter((job) => job.id !== response.data?.id),
    ]
    MessagePlugin.success(`已按「${config.name}」发起整理任务`)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '整理任务创建失败')
  } finally {
    setMemoryOrganizeCreating(item.id, false)
  }
}

const handleMemoryMenuAction = (item: MemoryItem, action: MemoryMenuAction) => {
  if (action === 'edit') {
    closeMemoryMenu()
    openMemoryEditor(item)
    return
  }
  if (action === 'organize') {
    closeMemoryMenu()
    void createOrganizeFromMemory(item)
    return
  }
  if (action === 'delete') {
    closeMemoryMenu()
    deleteMemoryItem(item)
  }
}

const openOutputPreview = (item: OutputItem) => {
  activeOutputPreview.value = item
  outputPreviewVisible.value = true
}

const closeOutputPreview = () => {
  outputPreviewVisible.value = false
}

const removeOutputFromLists = (id: string) => {
  outputs.value = outputs.value.filter((item) => item.id !== id)
  featuredOutputs.value = featuredOutputs.value.filter((item) => item.id !== id)
  if (activeOutputPreview.value?.id === id) {
    activeOutputPreview.value = null
    outputPreviewVisible.value = false
  }
}

const deleteOutputItem = (item: OutputItem) => {
  const dialog = DialogPlugin.confirm({
    header: '删除发现',
    body: `确认删除「${item.title || '无标题'}」？源文件也会一并删除。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: { content: '取消' },
    onConfirm: async () => {
      try {
        if (item.persisted) {
          const response = await deleteOrganizeOutput(item.id)
          if (response?.success === false) {
            throw new Error(response.message || '删除失败')
          }
        }
        removeOutputFromLists(item.id)
        MessagePlugin.success('已删除')
        dialog.destroy()
        void refreshDiscoverData({ resetPage: true })
      } catch (error: any) {
        MessagePlugin.error(error?.message || '删除失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

const editActiveOutputPreview = () => {
  const item = activeOutputPreview.value
  if (!item || !canEditActiveOutputPreview.value) return
  outputPreviewVisible.value = false
  void openDocumentEditor('output', item.id, item)
}

const buildOutputUpdateInput = (item: OutputItem, status: OrganizeOutputStatus) => ({
  title: item.title,
  output_type: item.type,
  content: item.content,
  source_summary: item.summary,
  status,
  icon: item.icon,
  memory_ids: item.memoryIds,
  metadata: item.metadata,
})

const updateOutputStatus = async (item: OutputItem, nextStatus: OrganizeOutputStatus) => {
  if (!canEditOutputItem(item) || item.statusKey === nextStatus || outputStatusSaving.value) return
  if (!editableOutputStatusOptions.some((option) => option.value === nextStatus)) return

  outputStatusSaving.value = true
  try {
    const response = await updateOrganizeOutput(item.id, buildOutputUpdateInput(item, nextStatus))
    if (response.success && response.data) {
      upsertOutputItem(response.data)
      void refreshDiscoverData({ resetPage: true })
      MessagePlugin.success('状态已更新')
    } else {
      MessagePlugin.error(response.message || '状态更新失败')
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '状态更新失败')
  } finally {
    outputStatusSaving.value = false
  }
}

const handleOutputStatusMenuClick = (item: OutputItem, action: { value: string | number | boolean }) => {
  if (String(action.value) === 'delete') {
    deleteOutputItem(item)
    return
  }
  void updateOutputStatus(item, String(action.value) as OrganizeOutputStatus)
}

const openOutputUploadDrawer = (kind: OutputCreateKind = 'article') => {
  outputUploadInitialKind.value = kind
  outputUploadVisible.value = true
}

const handleOutputCreateAction = (data: { value: string }) => {
  const kind = data.value === 'video' || data.value === 'audio' ? data.value : 'article'
  openOutputUploadDrawer(kind)
}

const upsertOutputItem = (item: OrganizeOutput) => {
  const next = mapOutput(item)
  const index = outputs.value.findIndex((candidate) => candidate.id === next.id)
  if (index !== -1) {
    outputs.value = outputs.value.map((candidate) => (candidate.id === next.id ? next : candidate))
  }
  const featuredIndex = featuredOutputs.value.findIndex((candidate) => candidate.id === next.id)
  if (featuredIndex !== -1) {
    featuredOutputs.value = featuredOutputs.value.map((candidate) => (candidate.id === next.id ? next : candidate))
  }
  if (activeOutputPreview.value?.id === next.id) {
    activeOutputPreview.value = next
  }
}

const handleOutputUploaded = (item: OrganizeOutput) => {
  upsertOutputItem(item)
  void refreshDiscoverData({ resetPage: true })
}

const handleOutputSaved = (item: OrganizeOutput) => {
  upsertOutputItem(item)
  void refreshDiscoverData({ resetPage: true })
  MessagePlugin.success('发现已更新')
}

onMounted(loadOrganizeData)
</script>

<style scoped lang="less">
.organize-page {
  margin: 0;
  height: 100%;
  box-sizing: border-box;
  flex: 1;
  display: flex;
  position: relative;
  min-height: 0;
  overflow: hidden;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.organize-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  padding: 20px 0 0 28px;
  box-sizing: border-box;
}

.organize-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  padding-right: 28px;
}

.organize-header-title {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.organize-title-row {
  display: flex;
  align-items: center;
  gap: 8px;

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 21px;
    font-weight: 500;
    line-height: 30px;
    letter-spacing: 0;
  }
}

.organize-fab-wrap {
  position: absolute;
  right: 28px;
  bottom: 28px;
  z-index: 5;
}

.organize-fab {
  width: 56px !important;
  height: 56px !important;
  min-width: 56px !important;
  padding: 0 !important;
  border: 0 !important;
  border-radius: 50% !important;
  background: linear-gradient(180deg, #16c65f 0%, #07c05f 100%) !important;
  box-shadow: 0 10px 24px rgba(7, 192, 95, 0.26) !important;
  color: #fff !important;

  &:hover {
    background: linear-gradient(180deg, #12b958 0%, #05b757 100%) !important;
    box-shadow: 0 12px 28px rgba(7, 192, 95, 0.3) !important;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
  }
}

.memory-import-input {
  display: none;
}

.organize-search {
  width: 260px;
}

.organize-header-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.organize-scroll {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 28px 8px 0;
}

.organize-section {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.organize-section--memory {
  max-width: none;
}

.section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.section-heading-info {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.asset-summary {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.asset-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 190px));
  gap: 10px;
}

.asset-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  width: 100%;
  max-width: 190px;
  min-height: 82px;
  padding: 8px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
  box-sizing: border-box;
}

button.asset-card {
  font: inherit;
  cursor: pointer;
  text-align: left;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }
}

.asset-card-label {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
}

.asset-card-description {
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.asset-card-value {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

@media (max-width: 720px) {
  .asset-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .asset-card {
    max-width: none;
  }
}

.content-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.output-board {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.output-filter-bar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px 12px;
  align-items: center;
  padding: 0 0 4px;
}

.output-filter-bar__filters {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  overflow-x: auto;
  flex-wrap: nowrap;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.15) transparent;

  &::-webkit-scrollbar {
    height: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background-color: rgba(0, 0, 0, 0.15);
    border-radius: 2px;
  }
}

.output-filter-bar__trailing {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.output-add-btn {
  flex-shrink: 0;
  height: 32px;
  min-width: 104px;
}

.output-add-btn--secondary {
  min-width: 104px;
}

.output-scope-tabs {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  height: 32px;
  padding: 2px;

  .segmented-tab {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    height: 26px;
    white-space: nowrap;
  }
}

.output-filter-field {
  width: 140px;
  flex-shrink: 0;
}

.output-filter-select {
  width: 100%;
}

.output-view-toggle {
  display: inline-flex;
  align-items: center;
  flex-shrink: 0;
  gap: 0;
  padding: 2px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.output-view-toggle-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 24px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &.active {
    background: var(--td-bg-color-container);
    color: var(--td-brand-color);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.08);
  }
}

.memory-list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  max-width: none;
}

.memory-list-back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 30px;
  padding: 0 8px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }
}

.memory-asset-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: none;
}

.memory-list-card {
  grid-template-columns: 88px minmax(0, 1fr);
}

.memory-list-card--editable,
.output-card--editable,
.report-card--editable {
  cursor: pointer;

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
  }
}

.memory-list-card-main {
  min-width: 0;
}

.memory-card-actions {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 3;
}

.memory-card-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #8a9099;
  cursor: pointer;

  &:hover,
  &.is-active {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
  }
}

.memory-card-cover {
  color: #0f8f52;
}

.memory-list-card--audio .memory-card-cover {
  color: #d87600;
}

.memory-list-card--audio-card .memory-card-cover {
  color: #2459d9;
}

.memory-card-footer {
  margin-top: auto;
}

:global(.memory-card-menu-popup .t-popup__content) {
  min-width: 228px;
  padding: 10px !important;
  border-radius: 8px !important;
}

.memory-card-menu {
  gap: 4px;
}

.memory-menu-item {
  width: 100%;
  min-height: 42px;
  border: 0;
  background: transparent;
  text-align: left;
  font: inherit;

  &:disabled {
    opacity: 0.56;
    cursor: progress;
  }
}

.memory-list-empty {
  padding: 24px 0;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  text-align: center;
}

.timeline-list,
.report-list {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.timeline-group {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 10px;
}

.timeline-date-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  grid-column: 1 / -1;
  padding: 2px 2px 0;
  color: var(--td-text-color-secondary);

  h2 {
    margin: 0;
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }
}

.memory-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 10px;
  min-height: 76px;
  padding: 14px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.memory-row-time {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.memory-row-body {
  min-width: 0;
}

.memory-row-meta,
.output-card-topline,
.report-topline {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-height: 24px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.type-badge {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 9px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  box-sizing: border-box;
}

.source-label {
  color: var(--td-text-color-placeholder);
}

.memory-row-title {
  margin-top: 8px;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.memory-row-summary {
  display: -webkit-box;
  max-width: 560px;
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  text-overflow: ellipsis;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.audio-card {
  display: grid;
  grid-template-columns: 16px minmax(104px, 160px) auto;
  align-items: center;
  gap: 8px;
  width: fit-content;
  max-width: 100%;
  min-height: 32px;
  margin-top: 8px;
  padding: 5px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  box-sizing: border-box;
}

.audio-wave {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;

  span {
    width: 2px;
    border-radius: 999px;
    background: var(--td-brand-color);
    opacity: 0.62;
  }
}

.audio-duration {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-weight: 400;
}

.segmented-tabs {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.segmented-tab {
  min-width: 52px;
  height: 24px;
  padding: 0 10px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  cursor: pointer;

  &--active {
    background: var(--td-bg-color-container);
    color: var(--td-text-color-primary);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  }
}

.discover-board {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.discover-mode-switch {
  display: inline-flex;
  gap: 2px;
  width: fit-content;
  padding: 3px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.discover-mode-switch button {
  min-width: 72px;
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.discover-mode-switch button.active {
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
}

.discover-featured-section,
.discover-tabs-section,
.discover-feed-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.discover-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }
}

.discover-refresh {
  flex: 0 0 auto;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.discover-featured-grid,
.discover-feed-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  width: min(100%, 768px);
}

.output-card.discover-card {
  width: 100%;
  height: 91px;
  min-height: 91px;
  box-sizing: border-box;
  grid-template-columns: 56px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 16px;
  border: 1px solid rgba(0, 0, 0, 0.04);
  border-radius: 16px;
  box-shadow: none;
}

.output-card.discover-card .output-card-cover {
  width: 56px;
  height: 56px;
  padding: 0;
  border: 0.5px solid rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  overflow: hidden;
  align-self: center;
}

.output-card.discover-card .output-card-cover-media {
  width: 56px;
  height: 56px;
  flex-basis: 56px;
  border-radius: 8px;
  font-size: 20px;
}

.output-card.discover-card .output-kind-label,
.output-card.discover-card .output-card-category {
  display: none;
}

.output-card.discover-card .output-card-body {
  justify-content: center;
}

.output-card.discover-card .output-card-body h2 {
  margin: 0;
  max-height: 16px;
  padding-right: 26px;
  font-size: 14px;
  font-weight: 400;
  line-height: 16px;
}

.output-card.discover-card .output-summary {
  margin: 4px 0 0;
  max-height: 14px;
  color: rgba(0, 0, 0, 0.44);
  font-size: 11px;
  line-height: 14px;
  -webkit-line-clamp: 1;
}

.output-card.discover-card .output-card-footer {
  margin-top: 8px;
}

.output-card.discover-card .discover-card-meta {
  gap: 2px;
  color: rgba(0, 0, 0, 0.44);
  font-size: 11px;
  line-height: 14px;
}

.discover-tabs-bar {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 16px;
  flex-wrap: wrap;
}

.discover-tabs-row {
  display: flex;
  align-items: center;
  gap: 28px;
  min-height: 40px;
  overflow-x: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.15) transparent;

  &::-webkit-scrollbar {
    height: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background-color: rgba(0, 0, 0, 0.15);
    border-radius: 2px;
  }
}

.discover-tab {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  cursor: pointer;
  white-space: nowrap;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &--active {
    color: var(--td-text-color-primary);
    font-weight: 400;
  }
}

.discover-card-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  white-space: nowrap;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.discover-card-meta-separator {
  color: var(--td-text-color-placeholder);
}

.output-card-category,
.output-preview-category {
  color: var(--td-brand-color-7);
  font-size: 12px;
  line-height: 18px;
}

.output-card-category {
  margin-top: 4px;
}

.output-preview-category {
  margin-top: 8px;
}

.output-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 14px;
}

.output-list-view {
  min-width: 0;
  overflow-x: auto;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
  background: var(--td-bg-color-container);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.output-list-header,
.output-list-row {
  display: grid;
  grid-template-columns:
    minmax(320px, 2.7fr)
    minmax(180px, 1.2fr)
    92px
    96px
    minmax(110px, 0.9fr)
    136px
    80px;
  align-items: center;
  column-gap: 0;
  min-width: 960px;
  padding: 0 16px;
}

.output-list-header {
  height: 40px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  border-radius: 8px 8px 0 0;
}

.output-list-body {
  display: flex;
  flex-direction: column;
}

.output-list-row {
  min-height: 68px;
  border-bottom: 1px solid var(--td-component-stroke);
  color: var(--td-text-color-primary);
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.2s ease;

  &:last-child {
    border-bottom: 0;
  }

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
  }
}

.output-cell {
  display: flex;
  align-items: center;
  min-width: 0;
  padding: 0 8px;

  &:first-child {
    padding-left: 0;
  }

  &:last-child {
    padding-right: 0;
  }
}

.output-cell-name {
  gap: 10px;
}

.output-cell-time,
.output-cell-actions {
  justify-content: flex-end;
}

.output-cell-actions {
  gap: 4px;
}

.output-cell-tags .output-tags {
  flex-wrap: nowrap;
  max-height: 24px;
  margin-bottom: 0;
}

.output-file-icon-wrap {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 18px;
}

.output-file-icon-wrap--video {
  background: rgba(37, 99, 235, 0.12);
  color: #2459d9;
}

.output-file-icon-wrap--audio {
  background: rgba(249, 115, 22, 0.14);
  color: #d87600;
}

.output-file-text {
  display: flex;
  flex-direction: column;
  gap: 3px;
  min-width: 0;
}

.output-file-name {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-file-desc {
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-muted,
.output-mono {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.output-row-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }
}

.output-empty {
  padding: 36px 0;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 20px;
  text-align: center;
}

.organize-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 168px;
  color: var(--td-text-color-secondary);
}

.discover-loading {
  min-height: 140px;
}

.output-pagination {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  min-height: 34px;
  padding: 4px 0 0;

  :deep(.t-pagination) {
    flex-wrap: wrap;
    justify-content: flex-end;
    row-gap: 8px;
  }
}

.output-card,
.report-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: rgba(0, 0, 0, 0.08);
    box-shadow: 0 6px 18px rgba(0, 0, 0, 0.06);
  }
}

.output-card {
  position: relative;
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  align-items: start;
  min-height: 124px;
  height: auto;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12);
  }
}

.output-card-cover {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 6px;
  width: auto;
  min-width: 0;
  width: 88px;
  height: 88px;
  padding: 10px;
  border-radius: 8px;
  box-sizing: border-box;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-secondarycontainer);
  background-size: cover;
  background-position: center;
  border: 1px solid var(--td-component-stroke);
  align-self: start;
}

.output-card-cover--article {
  background-color: var(--td-bg-color-secondarycontainer);
  color: #0f8f52;
}

.output-card-cover--video {
  background-color: var(--td-bg-color-secondarycontainer);
  color: #2459d9;
}

.output-card-cover--audio {
  background-color: var(--td-bg-color-secondarycontainer);
  color: #d87600;
}

.output-card-cover-media {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  flex: 0 0 44px;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: currentColor;
  font-size: 20px;

  img {
    width: 100%;
    height: 100%;
    border-radius: inherit;
    object-fit: cover;
  }
}

.output-card-cover-footer {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: auto;
}

.output-kind-label {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  min-height: 24px;
  padding: 2px 9px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  box-sizing: border-box;
}

.output-status-badge {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  white-space: nowrap;
}

.output-status-badge--draft {
  background: rgba(107, 114, 128, 0.12);
  color: #6b7280;
}

.output-status-badge--review {
  background: rgba(217, 119, 6, 0.14);
  color: #b45309;
}

.output-status-badge--ready {
  background: rgba(22, 163, 74, 0.14);
  color: #15803d;
}

.output-status-badge--archived {
  background: rgba(75, 85, 99, 0.12);
  color: #4b5563;
}

.output-card-topline {
  flex: 0 0 auto;
  gap: 8px;
  min-height: 16px;
  color: #8f8f8f;
  font-size: 11px;
  line-height: 16px;
}

.output-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  flex: 0 0 42px;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 20px;
}

.output-card-body,
.report-main {
  min-width: 0;
  flex: 1;

  h2 {
    margin: 10px 0 12px;
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 500;
    line-height: 23px;
    letter-spacing: 0;
  }
}

.output-card-body {
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  align-self: stretch;
  gap: 0;
  padding: 0;
  overflow: hidden;

  h2 {
    display: -webkit-box;
    flex: 0 0 auto;
    margin: 0 0 4px;
    max-height: 18px;
    padding-right: 34px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
  }
}

.output-card-head {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  gap: 12px;
  min-width: 0;
  margin: 0;
  pointer-events: none;
}

.output-card-source {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-summary {
  display: -webkit-box;
  margin: 0 0 6px;
  max-height: 36px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.output-card-footer {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  margin-top: auto;
  min-width: 0;
}

.output-meta,
.report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 14px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.output-meta {
  flex: 0 0 auto;
  justify-content: space-between;
  flex-wrap: nowrap;
  margin-top: auto;
  gap: 10px;
}

.output-creator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
}

.output-creator-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex: 0 0 18px;
  overflow: hidden;
  border-radius: 50%;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.output-creator-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-updated {
  flex: 0 0 auto;
}

.output-card-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transform: translateY(-2px);
  transition:
    opacity 0.16s ease,
    transform 0.16s ease,
    visibility 0s linear 0.16s;
}

.output-card:hover .output-card-actions,
.output-card:focus .output-card-actions,
.output-card:focus-within .output-card-actions {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
  transform: translateY(0);
  transition-delay: 0s;
}

.icon-button--more {
  flex: 0 0 auto;
}

:deep(.output-preview-drawer .t-drawer__body) {
  padding: 0;
  background: #f7f8fa;
}

.output-preview-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 68px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: rgba(255, 255, 255, 0.96);
  box-sizing: border-box;
}

.output-preview-heading {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.output-preview-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  flex: 0 0 36px;
  border-radius: 8px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 19px;
}

.output-preview-icon--video {
  background: rgba(37, 99, 235, 0.12);
  color: #2459d9;
}

.output-preview-icon--audio {
  background: rgba(249, 115, 22, 0.14);
  color: #d87600;
}

.output-preview-title-block {
  min-width: 0;
}

.output-preview-eyebrow {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.output-preview-title {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.output-preview-actions {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.output-preview-action {
  width: 28px;
  height: 28px;
  padding: 0 !important;
  border-radius: 6px;
}

.output-preview-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 18px 22px;
}

.output-preview-section {
  padding: 14px 16px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;

  h4 {
    margin: 0 0 12px;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 22px;
  }
}

.output-preview-summary {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;
}

.output-preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.output-preview-tags :deep(.t-tag) {
  border-radius: 999px;
}

.output-preview-content-section {
  min-height: 420px;
}

.output-preview-file-host {
  height: min(560px, calc(100vh - 310px));
  min-height: 360px;
}

.output-preview-file-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 240px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-placeholder);
  font-size: 13px;
}

.icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }
}

.report-list {
  max-width: 920px;
}

.report-card {
  align-items: center;
  justify-content: space-between;
}

.report-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0 0 8px;

  span {
    display: inline-flex;
    align-items: center;
    padding: 2px 4px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
    line-height: 14px;
  }
}

.report-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 104px;
  height: 34px;
  padding: 0 12px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  cursor: pointer;

  &:hover {
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
  }
}

.organize-section--sprout {
  max-width: 1080px;
}

.sprout-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 64px;
  padding: 8px 12px;
  border: 1px solid #e4dbcc;
  border-radius: 8px;
  background:
    linear-gradient(90deg, rgba(34, 101, 73, 0.06), transparent 38%),
    linear-gradient(135deg, rgba(164, 128, 57, 0.08), rgba(255, 255, 255, 0) 48%),
    #fffdf8;
  box-sizing: border-box;
}

.sprout-hero-copy {
  min-width: 0;

  p {
    max-width: 620px;
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
  }
}

.sprout-hero-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(86px, 1fr));
  gap: 10px;
  flex: 0 0 auto;

  span {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 42px;
    justify-content: center;
    padding: 4px 12px;
    border: 1px solid rgba(34, 101, 73, 0.12);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.72);
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    box-sizing: border-box;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
  }
}

.sprout-month-list,
.sprout-month-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.sprout-month-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
  }
}

.sprout-month-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-width: 0;
}

.sprout-range-tabs {
  flex: 0 0 auto;
  gap: 28px;
  padding: 0;
  border: 0;
  background: transparent;

  .segmented-tab {
    min-width: 56px;
    height: auto;
    padding: 0;
    border-radius: 0;
    color: var(--td-text-color-secondary);

    &:hover {
      color: var(--td-text-color-primary);
    }

    &--active {
      background: transparent;
      color: var(--td-text-color-primary);
      box-shadow: none;
    }
  }
}

.sprout-report-list {
  max-width: none;
  gap: 10px;
}

.sprout-report-card {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  min-height: 124px;
  overflow: hidden;
  border: 1px solid #e1d7c7;
  border-radius: 8px;
  background:
    linear-gradient(0deg, rgba(35, 31, 27, 0.018) 1px, transparent 1px),
    linear-gradient(90deg, rgba(35, 31, 27, 0.014) 1px, transparent 1px),
    #fffdf8;
  background-size: 22px 22px;
  box-shadow: 0 4px 14px rgba(38, 34, 29, 0.05);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: rgba(34, 101, 73, 0.42);
    box-shadow: 0 10px 26px rgba(38, 34, 29, 0.1);
  }
}

.sprout-report-gutter {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  border-right: 1px solid #e6ddcf;
  background:
    linear-gradient(180deg, rgba(34, 101, 73, 0.08), rgba(164, 128, 57, 0.08)),
    #f8f2e7;
}

.sprout-report-ribbon {
  display: grid;
  place-items: center;
  width: 48px;
  height: 56px;
  border: 2px solid #20242a;
  background: rgba(255, 253, 248, 0.72);
  color: var(--td-text-color-primary);
  font-family: "Songti SC", "STSong", serif;
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  letter-spacing: 0;
  text-align: center;
  box-shadow: inset 0 0 0 1px rgba(32, 36, 42, 0.12);

  span {
    display: block;
  }
}

.sprout-report-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 12px;

  h2 {
    display: -webkit-box;
    margin: 0;
    max-height: 18px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
  }
}

.sprout-report-title-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  margin: 2px 0 6px;

  h2 {
    flex: 1;
    min-width: 0;
  }

  .type-badge {
    flex-shrink: 0;
    min-height: 20px;
    padding: 1px 7px;
    line-height: 16px;
  }
}

.sprout-report-intro {
  display: -webkit-box;
  margin: 0;
  max-height: 54px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.sprout-report-main .report-chips {
  margin-top: auto;
  margin-bottom: 6px;
}

.sprout-report-meta {
  padding-top: 6px;
}

.sprout-report-meta-separator {
  color: var(--td-text-color-placeholder);
}

.type-badge.sprout-stage--formed {
  background: rgba(34, 101, 73, 0.1);
  color: #236549;
}

.type-badge.sprout-stage--expandable {
  background: rgba(146, 94, 28, 0.1);
  color: #7a4d18;
}

.type-badge.sprout-stage--organizing {
  background: rgba(35, 99, 148, 0.1);
  color: #1f5a86;
}

:deep(.sprout-preview-drawer .t-drawer__body) {
  padding: 0;
  background: #f5f0e7;
}

.sprout-preview-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 64px;
  padding: 12px 18px;
  border-bottom: 1px solid rgba(32, 36, 42, 0.1);
  background: rgba(255, 255, 255, 0.94);
  box-sizing: border-box;
}

.sprout-preview-header-copy {
  min-width: 0;
}

.sprout-preview-eyebrow {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.sprout-preview-title {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.sprout-preview-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.sprout-preview-action {
  width: 30px !important;
  min-width: 30px !important;
  height: 30px !important;
  padding: 0 !important;
  border-radius: 6px !important;
}

.sprout-preview-page {
  min-height: 100%;
  padding: 22px 26px 48px;
  box-sizing: border-box;
}

.sprout-preview-body {
  margin-top: 0;
  padding: 30px 34px 38px;
  border: 1px solid #ded2bf;
  border-radius: 8px;
  background: #fffdf8;
  color: var(--td-text-color-primary);
  box-shadow: 0 10px 26px rgba(38, 34, 29, 0.08);
  box-sizing: border-box;

  h1 {
    margin: 12px 0 18px;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }
}

.sprout-preview-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.sprout-preview-content {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;

  :deep(h1) {
    display: none;
  }

  :deep(h2) {
    margin: 30px 0 12px;
    padding-top: 18px;
    border-top: 1px solid rgba(32, 36, 42, 0.12);
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }

  :deep(h3) {
    margin: 24px 0 10px;
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
    letter-spacing: 0;
  }

  :deep(p) {
    margin: 10px 0;
  }

  :deep(> p:first-child) {
    margin: 0 0 18px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  :deep(blockquote) {
    margin: 14px 0;
    padding: 11px 14px 11px 16px;
    border-left: 3px solid #2d7a52;
    border-radius: 0 6px 6px 0;
    background: rgba(34, 101, 73, 0.06);
    color: #4e5a52;
  }

  :deep(blockquote p) {
    margin: 4px 0;
  }

  :deep(strong) {
    color: var(--td-text-color-primary);
    font-weight: 400;
  }

  :deep(ul),
  :deep(ol) {
    margin: 10px 0 14px;
    padding-left: 22px;
  }

  :deep(li) {
    margin: 4px 0;
  }
}

@media (max-width: 900px) {
  .organize-main {
    padding: 18px 0 0 18px;
  }

  .organize-header {
    margin-right: 18px;
    padding-right: 0;
  }

  .organize-scroll {
    padding-right: 18px;
  }

  .organize-fab-wrap {
    right: 18px;
    bottom: 18px;
  }

  .organize-fab {
    width: 52px !important;
    height: 52px !important;
    min-width: 52px !important;
  }

  .sprout-report-card {
    grid-template-columns: 88px minmax(0, 1fr);
  }

}

@media (max-width: 760px) {
  .organize-header,
  .content-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .organize-header-actions,
  .organize-search {
    width: 100%;
  }

  .output-grid {
    grid-template-columns: 1fr;
  }

  .discover-featured-grid,
  .discover-feed-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 8px;
    width: 100%;
  }

  .output-card.discover-card {
    height: 132px;
    min-height: 132px;
  }

  .timeline-group {
    grid-template-columns: 1fr;
  }

  .output-pagination,
  .sprout-pagination {
    justify-content: center;

    :deep(.t-pagination) {
      justify-content: center;
    }
  }

  .memory-row {
    grid-template-columns: 1fr;
  }

  .sprout-hero {
    align-items: stretch;
    flex-direction: column;
  }

  .sprout-hero-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    width: 100%;
  }

  .sprout-report-card {
    grid-template-columns: 1fr;
  }

  .sprout-report-gutter {
    min-height: 72px;
    border-right: 0;
    border-bottom: 1px solid #e6ddcf;
  }

  .sprout-report-ribbon {
    width: 72px;
    height: 42px;
    grid-template-columns: repeat(2, auto);
    gap: 4px;
    font-size: 12px;
    line-height: 18px;
  }

  .sprout-preview-page {
    padding: 16px 14px 36px;
  }

  .sprout-preview-body {
    padding: 24px 18px 30px;

    h1 {
      font-size: 12px;
      line-height: 18px;
    }
  }
}
</style>
