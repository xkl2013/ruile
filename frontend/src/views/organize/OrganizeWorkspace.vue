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
          <div class="asset-summary" aria-label="记忆资产">
            <div class="section-heading">
              <t-icon name="folder" />
              <span>记忆资产（{{ allMemoryItems.length }} 条）</span>
              <t-icon name="info-circle" class="section-heading-info" />
            </div>
            <div class="asset-grid">
              <button
                v-for="asset in memoryAssets"
                :key="asset.key"
                type="button"
                class="asset-card"
                @click="openMemoryAssetList(asset.key)"
              >
                <span class="asset-card-label">{{ asset.label }}</span>
                <span class="asset-card-value">{{ asset.count }} {{ asset.unit }}</span>
              </button>
            </div>
          </div>

          <template v-if="activeMemoryAsset">
            <div class="memory-list-toolbar">
              <button type="button" class="memory-list-back" @click="openMemoryOverview">
                <t-icon name="chevron-left" />
                <span>记忆资产</span>
              </button>
              <div class="section-heading">
                <t-icon :name="activeMemoryAssetMeta.icon" />
                <span>{{ activeMemoryAssetMeta.label }}列表</span>
              </div>
            </div>

            <div class="memory-asset-list">
              <article
                v-for="item in filteredMemoryAssetItems"
                :key="item.id"
                class="memory-list-card"
                :class="{ 'memory-list-card--editable': isEditableMemory(item) }"
                :role="isEditableMemory(item) ? 'button' : undefined"
                :tabindex="isEditableMemory(item) ? 0 : undefined"
                @click="openMemoryEditor(item)"
                @keydown.enter.self="openMemoryEditor(item)"
              >
                <div class="memory-list-card-main">
                  <div class="memory-row-meta">
                    <span class="type-badge">{{ activeMemoryAssetMeta.itemTypeLabel }}</span>
                    <span>{{ item.date }} {{ item.time }}</span>
                    <span v-if="item.source" class="source-label">{{ item.source }}</span>
                  </div>
                  <h2>{{ item.title }}</h2>
                  <div v-if="item.type === 'audio'" class="audio-card">
                    <t-icon name="sound" />
                    <div class="audio-wave" aria-hidden="true">
                      <span
                        v-for="n in 18"
                        :key="n"
                        :style="{ height: `${waveHeights[(n - 1) % waveHeights.length]}px` }"
                      />
                    </div>
                    <span class="audio-duration">{{ item.duration }}</span>
                  </div>
                </div>
                <button
                  v-if="isEditableMemory(item)"
                  type="button"
                  class="icon-button"
                  :aria-label="`编辑 ${item.title}`"
                  @click.stop="openMemoryEditor(item)"
                >
                  <t-icon name="chevron-right" />
                </button>
              </article>
              <div v-if="filteredMemoryAssetItems.length === 0" class="memory-list-empty">
                暂无{{ activeMemoryAssetMeta.label }}
              </div>
            </div>
          </template>

          <template v-else>
            <div class="timeline-list">
              <section v-for="group in filteredMemoryGroups" :key="group.date" class="timeline-group">
                <div class="timeline-date-row">
                  <h2>{{ group.date }}</h2>
                  <t-icon name="chevron-up" />
                </div>
                <article
                  v-for="item in group.items"
                  :key="item.id"
                  class="memory-row"
                  :class="{ 'memory-row--editable': isEditableMemory(item) }"
                  :role="isEditableMemory(item) ? 'button' : undefined"
                  :tabindex="isEditableMemory(item) ? 0 : undefined"
                  @click="openMemoryEditor(item)"
                  @keydown.enter.self="openMemoryEditor(item)"
                >
                  <div class="memory-row-time">{{ item.time }}</div>
                  <div class="memory-row-body">
                    <div class="memory-row-meta">
                      <span class="type-badge">{{ item.typeLabel }}</span>
                      <span v-if="item.source" class="source-label">{{ item.source }}</span>
                    </div>
                    <div class="memory-row-title">{{ item.title }}</div>
                    <div v-if="item.type === 'audio'" class="audio-card">
                      <t-icon name="sound" />
                      <div class="audio-wave" aria-hidden="true">
                        <span
                          v-for="n in 18"
                          :key="n"
                          :style="{ height: `${waveHeights[(n - 1) % waveHeights.length]}px` }"
                        />
                      </div>
                      <span class="audio-duration">{{ item.duration }}</span>
                    </div>
                  </div>
                </article>
              </section>
            </div>
          </template>
        </section>

        <section v-else-if="activeTab === 'output'" class="organize-section organize-section--output">
          <div class="discover-board">
            <section class="discover-featured-section">
              <div class="discover-section-head">
                <h3>精选</h3>
                <t-button
                  variant="text"
                  theme="default"
                  class="discover-refresh"
                  :disabled="filteredOutputs.length <= 1"
                  @click="rotateFeaturedOutputs"
                >
                  <template #icon><t-icon name="refresh" /></template>
                  换一换
                </t-button>
              </div>

              <div v-if="featuredOutputs.length" class="discover-featured-grid">
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
                    @click="discoverTab = tab.value"
                  >
                    {{ tab.label }}
                  </button>
                </div>
              </div>
            </section>

            <section class="discover-feed-section">
              <div v-if="paginatedOutputs.length" class="discover-feed-grid">
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
            </section>

            <div v-if="filteredOutputs.length > OUTPUT_PAGE_SIZE" class="output-pagination" aria-label="发现分页">
              <t-pagination
                v-model="outputPage"
                :page-size="OUTPUT_PAGE_SIZE"
                :total="filteredOutputs.length"
                size="small"
                show-page-number
              />
            </div>
          </div>
        </section>

        <section v-else class="organize-section organize-section--sprout">
          <div class="sprout-hero">
            <div class="sprout-hero-copy">
              <div class="section-heading">
                <t-icon name="tree-list" />
                <span>经营复盘</span>
              </div>
              <p>沉淀招生、家长沟通和园务管理要点，生成园长可直接复盘和安排跟进的经营建议。</p>
            </div>
            <div class="sprout-hero-stats">
              <span><strong>{{ filteredSproutReports.length }}</strong> 份复盘</span>
              <span><strong>{{ sproutReports.length }}</strong> 条素材</span>
            </div>
          </div>

          <div class="sprout-month-list">
            <section v-for="group in sproutMonthGroups" :key="group.key" class="sprout-month-group">
              <div class="sprout-month-header">
                <div class="sprout-month-heading">
                  <h3>{{ group.label }}</h3>
                  <div class="segmented-tabs sprout-range-tabs" role="tablist" aria-label="经营复盘时间筛选">
                    <button
                      v-for="tab in sproutRangeTabs"
                      :key="tab.value"
                      type="button"
                      class="segmented-tab"
                      :class="{ 'segmented-tab--active': sproutRange === tab.value }"
                      role="tab"
                      :aria-selected="sproutRange === tab.value"
                      @click="setSproutRange(tab.value)"
                    >
                      {{ tab.label }}
                    </button>
                  </div>
                </div>
                <span>{{ filteredSproutReports.length }} 份复盘</span>
              </div>

              <div v-if="group.reports.length" class="report-list sprout-report-list">
                <article
                  v-for="report in group.reports"
                  :key="report.id"
                  class="sprout-report-card report-card--editable"
                  role="button"
                  tabindex="0"
                  @click="openSproutPreview(report)"
                  @keydown.enter.self="openSproutPreview(report)"
                >
                  <div class="sprout-report-gutter" aria-hidden="true">
                    <div class="sprout-report-ribbon">
                      <span>经营</span>
                      <span>复盘</span>
                    </div>
                  </div>

                  <div class="sprout-report-main">
                    <h2>{{ report.title }}</h2>
                    <p class="sprout-report-intro">{{ report.intro }}</p>

                    <div v-if="report.chips.length" class="report-chips">
                      <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                    </div>

                    <div class="report-meta sprout-report-meta">
                      <span>{{ report.updated }}</span>
                      <span class="sprout-report-meta-separator">|</span>
                      <span>@记忆</span>
                      <span>@上传文件</span>
                    </div>
                  </div>

                </article>
              </div>

              <div v-else class="output-empty">暂无经营复盘</div>
            </section>
          </div>
          <div v-if="filteredSproutReports.length > SPROUT_PAGE_SIZE" class="sprout-pagination" aria-label="经营复盘分页">
            <t-pagination
              v-model="sproutPage"
              :page-size="SPROUT_PAGE_SIZE"
              :total="filteredSproutReports.length"
              size="small"
              show-page-number
            />
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
      <t-tooltip v-else :content="activeMeta.actionLabel" placement="left">
        <t-button
          class="organize-fab"
          theme="primary"
          shape="circle"
          :aria-label="activeMeta.actionLabel"
          @click="createActiveDocument"
        >
          <template #icon><t-icon name="add" size="20px" /></template>
        </t-button>
      </t-tooltip>
    </div>

    <t-drawer
      v-model:visible="sproutPreviewVisible"
      class="sprout-preview-drawer"
      :header="false"
      :footer="false"
      :close-btn="false"
      :size="'min(760px, 92vw)'"
      attach="body"
      placement="right"
    >
      <template v-if="activeSproutReport">
        <div class="sprout-preview-header">
          <div class="sprout-preview-header-copy">
            <div class="sprout-preview-eyebrow">经营复盘</div>
            <div class="sprout-preview-title">{{ activeSproutReport.title }}</div>
          </div>
          <div class="sprout-preview-actions">
            <t-button
              v-if="canEditSproutReport(activeSproutReport)"
              variant="text"
              theme="default"
              size="small"
              class="sprout-preview-action"
              aria-label="编辑报告"
              @click="openSproutEditor(activeSproutReport)"
            >
              <template #icon><t-icon name="edit-1" size="16px" /></template>
            </t-button>
            <t-button
              variant="text"
              theme="default"
              size="small"
              class="sprout-preview-action"
              aria-label="关闭预览"
              @click="closeSproutPreview"
            >
              <template #icon><t-icon name="close" size="16px" /></template>
            </t-button>
          </div>
        </div>

        <div class="sprout-preview-page">
          <div class="sprout-preview-body">
            <div class="sprout-preview-meta">
              <span>{{ activeSproutReport.updated }}</span>
              <span>{{ activeSproutReport.memoryCount }} 条记忆</span>
            </div>
            <h1>{{ activeSproutReport.title }}</h1>

            <div class="sprout-preview-content" v-html="activeSproutReport.renderedHtml" />
          </div>
        </div>
      </template>
      </t-drawer>

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
import { Icon as TIcon, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import DocumentPreview from '@/components/document-preview.vue'
import {
  listOrganizeMemories,
  listOrganizeOutputs,
  listOrganizeSproutReports,
  type OrganizeMemory,
  type OrganizeMemoryKind,
  type OrganizeOutput,
  type OrganizeOutputStatus,
  type OrganizeSproutReport,
  type OrganizeSproutStage,
  updateOrganizeOutput,
} from '@/api/organize'
import { useAuthStore } from '@/stores/auth'
import {
  ORGANIZE_MEMORY_ASSET_ROUTES,
  findMemoryAssetRoute,
  findOrganizeMenuRoute,
  isMemoryAssetKey,
  isOrganizeTab,
  type MemoryAssetKey,
  type OrganizeTab,
} from './organizeRoutes'
import { saveOrganizeEditorDraft, type OrganizeEditorDraft } from './editorDraftStorage'
import {
  buildSproutReportPreview,
  sproutReportContentForEditor,
  type SproutReportPreviewSection,
} from './sproutReport'
import OrganizeOutputUploadDrawer from './components/OrganizeOutputUploadDrawer.vue'

type MemoryType = 'note' | 'record' | 'audio' | 'audio-card'
type OutputKind = 'all' | 'article' | 'video' | 'audio'
type OutputCreateKind = Exclude<OutputKind, 'all'>
type OutputStatusFilter = 'all' | OrganizeOutputStatus
type OutputViewMode = 'list' | 'grid'
type SproutRange = '3d' | '7d' | '1m'

interface MemoryItem {
  id: string
  time: string
  type: MemoryType
  typeLabel: string
  title: string
  content: string
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

interface OutputItem {
  id: string
  title: string
  content: string
  type: string
  kind: Exclude<OutputKind, 'all'>
  kindLabel: string
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

interface SproutReportItem {
  id: string
  title: string
  content: string
  renderedHtml: string
  intro: string
  previewSections: SproutReportPreviewSection[]
  stage: string
  stageKey: OrganizeSproutStage
  updated: string
  updatedAt: string
  generatedLabel: string
  memoryCount: number
  memoryIds: string[]
  outputHint: string
  chips: string[]
  creatorId?: string
  creatorName?: string
  creatorAvatar?: string
  metadata?: Record<string, unknown>
  persisted: boolean
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const keyword = ref('')
const outputUploadVisible = ref(false)
const outputUploadInitialKind = ref<OutputCreateKind>('article')
const outputPreviewVisible = ref(false)
const activeOutputPreview = ref<OutputItem | null>(null)
const outputStatusSaving = ref(false)
const discoverTab = ref('recommended')
const featuredRotation = ref(0)
const sproutRange = ref<SproutRange>('3d')
const sproutPreviewVisible = ref(false)
const activeSproutReport = ref<SproutReportItem | null>(null)
const OUTPUT_PAGE_SIZE = 10
const outputPage = ref(1)
const SPROUT_PAGE_SIZE = 10
const sproutPage = ref(1)
const SPROUT_DAY_MS = 24 * 60 * 60 * 1000
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
const sproutRangeTabs: Array<{ label: string; value: SproutRange; days: number }> = [
  { label: '近3天', value: '3d', days: 3 },
  { label: '近7天', value: '7d', days: 7 },
  { label: '近一月', value: '1m', days: 30 },
]

const activeTab = computed<OrganizeTab>(() => {
  const tab = route.meta.organizeTab
  return isOrganizeTab(tab) ? tab : 'memory'
})

const activeMemoryAsset = computed<MemoryAssetKey | ''>(() => {
  if (activeTab.value !== 'memory') return ''
  const asset = route.meta.memoryAsset
  return isMemoryAssetKey(asset) ? asset : ''
})

const escapeHtml = (value: string) => {
  const node = document.createElement('div')
  node.textContent = value
  return node.innerHTML
}

const placeholderContent = (title: string) => `<p>${escapeHtml(title)}</p><p></p>`

const fallbackMemoryGroups: MemoryGroup[] = [
  {
    date: '6月29日',
    items: [
      { id: 'demo-m1', time: '11:41', type: 'record', typeLabel: '记录', title: '创建了笔记电力行业相关企业分析及功率半导体产业链解读', content: '<p>围绕电力行业企业和功率半导体产业链，补充重点公司、供需关系及国产替代进展。</p>', persisted: false },
      { id: 'demo-m2', time: '11:27', type: 'record', typeLabel: '记录', title: '创建了笔记能源行业公司分析及中国电力结构探讨', content: '<p>整理能源行业公司基本面，以及中国电力结构变化带来的机会。</p>', persisted: false },
      { id: 'demo-m3', time: '11:13', type: 'record', typeLabel: '记录', title: '创建了笔记燃气轮机核心配件与光储行业分析', content: '<p>梳理燃气轮机核心配件、光伏和储能行业的关键数据。</p>', persisted: false },
      { id: 'demo-m4', time: '10:58', type: 'record', typeLabel: '记录', title: '创建了笔记燃气轮机行业分析及国内企业发展情况', content: '<p>记录燃气轮机市场格局和国内主要企业的发展情况。</p>', persisted: false },
      { id: 'demo-m5', time: '10:22', type: 'audio', typeLabel: '录音', title: '客户访谈：设备更新预算与项目推进节奏', content: '', source: '会议录音', duration: '08:36', persisted: false },
    ],
  },
  {
    date: '6月28日',
    items: [
      { id: 'demo-m6', time: '18:05', type: 'note', typeLabel: '笔记', title: '整理储能项目投标材料中的常见技术指标', content: '<p>整理储能项目投标材料中的效率、循环寿命、安全和并网指标。</p>', source: '手动输入', persisted: false },
      { id: 'demo-m7', time: '15:42', type: 'note', typeLabel: '笔记', title: '政策口径：新型电力系统与源网荷储协同', content: '<p>记录新型电力系统政策中的源网荷储协同要点。</p>', source: '网页摘录', persisted: false },
      { id: 'demo-m8', time: '09:18', type: 'audio', typeLabel: '录音', title: '内部同步：半导体设备国产替代机会', content: '', source: '语音记录', duration: '12:04', persisted: false },
    ],
  },
]

const memoryGroups = ref<MemoryGroup[]>(fallbackMemoryGroups)

const fallbackOutputs: OutputItem[] = [
  {
    id: 'demo-o1',
    title: '电力行业相关企业分析及功率半导体产业链解读',
    content: '<h2>行业概览</h2><p>从电力设备需求出发，梳理功率半导体产业链及重点企业的竞争位置。</p>',
    type: '图文类',
    kind: 'article',
    kindLabel: '图文类',
  source: '来自 8 条记忆',
  summary: '从电力设备需求出发，梳理功率半导体产业链及重点企业的竞争位置。',
  updated: '今天 11:48',
  createdAtLabel: '今天 11:48',
  status: '可交付',
  statusKey: 'ready',
  statusLabel: '可交付',
    icon: 'file-word',
    tags: ['产业链', '功率半导体'],
    memoryIds: [],
    persisted: false,
  },
  {
    id: 'demo-o2',
    title: '能源行业公司分析及中国电力结构探讨',
    content: '<p>分析中国电力结构变化，并比较相关能源公司的业务布局。</p>',
    type: '图文类',
    kind: 'article',
    kindLabel: '图文类',
  source: '来自 6 条记忆',
  summary: '分析中国电力结构变化，并比较相关能源公司的业务布局。',
  updated: '今天 11:31',
  createdAtLabel: '今天 11:31',
  status: '草稿',
  statusKey: 'draft',
  statusLabel: '草稿',
    icon: 'file-word',
    tags: ['能源结构', '业务布局'],
    memoryIds: [],
    persisted: false,
  },
  {
    id: 'demo-o3',
    title: '燃气轮机核心配件供应链梳理',
    content: '<p>梳理燃气轮机核心配件的供应商、交付周期和国产化进度。</p>',
    type: '图文类',
    kind: 'article',
    kindLabel: '图文类',
  source: '来自 5 条记忆',
  summary: '梳理燃气轮机核心配件的供应商、交付周期和国产化进度。',
  updated: '昨天 19:20',
  createdAtLabel: '昨天 19:20',
  status: '评审中',
  statusKey: 'review',
  statusLabel: '评审中',
    icon: 'file-word',
    tags: ['燃气轮机', '供应链'],
    memoryIds: [],
    persisted: false,
  },
  {
    id: 'demo-o4',
    title: '光储行业重点企业与政策机会清单',
    content: '<p>汇总光伏、储能行业重点企业和近期政策机会。</p>',
    type: '图文类',
    kind: 'article',
    kindLabel: '图文类',
  source: '来自 9 条记忆',
  summary: '汇总光伏、储能行业重点企业和近期政策机会。',
  updated: '6月27日',
  createdAtLabel: '6月27日',
  status: '可交付',
  statusKey: 'ready',
  statusLabel: '可交付',
    icon: 'file-word',
    tags: ['光伏', '储能'],
    memoryIds: [],
    persisted: false,
  },
]

const outputs = ref<OutputItem[]>(fallbackOutputs)

type SproutReportBaseItem = Omit<SproutReportItem, 'renderedHtml' | 'intro' | 'previewSections' | 'updatedAt' | 'generatedLabel'>

const formatGeneratedDateLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return `${String(date.getMonth() + 1).padStart(2, '0')}月${String(date.getDate()).padStart(2, '0')}日生成`
}

const enrichSproutReport = (item: SproutReportBaseItem & { updatedAt?: string }): SproutReportItem => {
  const preview = buildSproutReportPreview(item.content, item.title)
  return {
    ...item,
    renderedHtml: sproutReportContentForEditor(item.content),
    intro: preview.intro || item.title,
    previewSections: preview.sections,
    updatedAt: item.updatedAt || '',
    generatedLabel: (item.updatedAt && formatGeneratedDateLabel(item.updatedAt)) || item.updated,
  }
}

const fallbackSproutReports: SproutReportItem[] = [
  enrichSproutReport({
    id: 'demo-r1',
    title: '本周招生线索跟进复盘',
    content: '本周新增咨询集中来自老生转介绍和社区活动，家长最关心入园适应、师资稳定和托育时段。建议园长优先复盘线索跟进节奏，把高意向家庭安排到本周开放日。\n\n## 01. 高意向家庭需要更快分层\n\n> **🌱 种子**\n> 多条记录提到家长已经问到学位、费用和试听安排。\n\n将线索拆成已到访、待到访、观望三类，招生顾问当天完成回访，园长跟进关键家庭的疑虑。\n\n> **✨ Aha 瞬间**\n> 招生转化不是更多话术，而是把家长最担心的问题更早交给合适的人解决。',
    stage: '可扩写',
    stageKey: 'expandable',
    updated: '今天 12:10',
    memoryCount: 11,
    memoryIds: [],
    outputHint: '可生成招生复盘',
    chips: ['招生线索', '家长咨询', '开放日'],
    persisted: false,
  }),
  enrichSproutReport({
    id: 'demo-r2',
    title: '开放日到园体验优化建议',
    content: '近期到园家庭反馈集中在参观动线、课程展示和离园答疑三个环节。建议园长把开放日拆成接待、参观、体验、答疑四个节点，明确每位老师的讲解重点。\n\n## 01. 参观动线要服务家长决策\n\n> **🌱 种子**\n> 记录中多次出现家长询问安全、餐食、午睡和班级老师稳定性。\n\n开放日不只展示环境，更要让家长看到孩子一天如何被照顾，以及园所如何回应个性化问题。\n\n> **✨ Aha 瞬间**\n> 家长选择园所时买的是确定感，接待流程越清晰，成交后的信任成本越低。',
    stage: '梳理中',
    stageKey: 'organizing',
    updated: '今天 10:46',
    memoryCount: 7,
    memoryIds: [],
    outputHint: '可形成到园SOP',
    chips: ['开放日', '到园体验', '接待SOP'],
    persisted: false,
  }),
  enrichSproutReport({
    id: 'demo-r3',
    title: '老生续费与转介绍跟进计划',
    content: '本月家长沟通记录显示，续费犹豫主要来自成长反馈不够具体、升班安排说明不清晰。建议园长组织班主任梳理幼儿成长亮点，并设计老带新转介绍触达节奏。\n\n## 01. 续费沟通要先讲清成长证据\n\n> **🌱 种子**\n> 多条记录提到家长关注孩子表达能力、社交状态和生活习惯变化。\n\n续费前先让家长看到孩子的具体进步，再说明下阶段课程目标和班级安排。\n\n> **✨ Aha 瞬间**\n> 转介绍来自家长真实认可，续费沟通越具体，推荐意愿越容易被激活。',
    stage: '已成型',
    stageKey: 'formed',
    updated: '昨天 18:22',
    memoryCount: 9,
    memoryIds: [],
    outputHint: '可生成跟进清单',
    chips: ['续费', '转介绍', '成长反馈'],
    persisted: false,
  }),
]

const sproutReports = ref<SproutReportItem[]>(fallbackSproutReports)

const allMemoryItems = computed<MemoryListItem[]>(() => {
  return memoryGroups.value.flatMap((group) => group.items.map((item) => ({ ...item, date: group.date })))
})

const memoryAssets = computed(() => {
  const items = allMemoryItems.value
  return ORGANIZE_MEMORY_ASSET_ROUTES.map((asset) => ({
    ...asset,
    count:
      asset.key === 'note'
        ? items.filter((item) => item.type === 'note' || item.type === 'record').length
        : asset.key === 'audio'
          ? items.filter((item) => item.type === 'audio').length
          : items.filter((item) => item.type === 'audio-card').length,
  }))
})

const activeMemoryAssetMeta = computed(() => {
  return memoryAssets.value.find((asset) => asset.key === activeMemoryAsset.value) || memoryAssets.value[0]
})

const openMemoryAssetList = async (asset: MemoryAssetKey) => {
  const nextRoute = findMemoryAssetRoute(asset)
  if (nextRoute && route.path !== nextRoute.path) await router.push(nextRoute.path)
}

const openMemoryOverview = async () => {
  const memoryRoute = findOrganizeMenuRoute('memory')
  if (memoryRoute && route.path !== memoryRoute.path) await router.push(memoryRoute.path)
}

const activeMeta = computed(() => {
  if (activeTab.value === 'output') {
    return { title: '发现', actionIcon: 'file-add', actionLabel: '新建' }
  }
  if (activeTab.value === 'sprout') {
    return { title: '经营复盘', actionIcon: 'add', actionLabel: '新建复盘' }
  }
  if (activeMemoryAsset.value) {
    return { title: `${activeMemoryAssetMeta.value.label}列表`, actionIcon: 'add', actionLabel: '添加笔记' }
  }
  return { title: '记忆', actionIcon: 'add', actionLabel: '添加笔记' }
})

const currentUserId = computed(() => authStore.currentUserId || authStore.user?.id || '')

const setSproutRange = (range: SproutRange) => {
  sproutRange.value = range
  sproutPage.value = 1
}

const filteredMemoryGroups = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return memoryGroups.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        const keywordMatched = !q || item.title.toLowerCase().includes(q) || item.typeLabel.toLowerCase().includes(q)
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
    return assetMatched && (!q || item.title.toLowerCase().includes(q) || item.typeLabel.toLowerCase().includes(q))
  })
})

const filteredOutputs = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  const tab = discoverTab.value
  return outputs.value.filter((item) => {
    const tabMatched =
      tab === 'recommended'
        ? true
        : tab === 'article' || tab === 'video' || tab === 'audio'
          ? item.kind === tab
          : tab.startsWith('tag:')
            ? item.tags.includes(tab.slice(4))
            : true
    const keywordMatched = !q || [
      item.title,
      item.type,
      item.summary,
      item.source,
      item.tags.join(' '),
      outputCreatorDisplayName(item),
    ].join(' ').toLowerCase().includes(q)
    return tabMatched && keywordMatched
  })
})

const discoverTabs = computed(() => {
  const tagCounts = new Map<string, number>()
  outputs.value.forEach((item) => {
    item.tags.forEach((tag) => {
      tagCounts.set(tag, (tagCounts.get(tag) || 0) + 1)
    })
  })
  const topTags = Array.from(tagCounts.entries())
    .sort((a, b) => (b[1] - a[1]) || a[0].localeCompare(b[0], 'zh-CN'))
    .slice(0, 6)
    .map(([tag]) => ({ label: tag, value: `tag:${tag}` }))

  return [
    { label: '推荐', value: 'recommended' },
    { label: '图文类', value: 'article' },
    { label: '视频类', value: 'video' },
    { label: '音频类', value: 'audio' },
    ...topTags,
  ]
})

const activeDiscoverTabLabel = computed(() => {
  return discoverTabs.value.find((tab) => tab.value === discoverTab.value)?.label || '推荐'
})

const featuredOutputs = computed(() => {
  const items = filteredOutputs.value
  if (items.length === 0) return []
  const count = Math.min(4, items.length)
  const start = featuredRotation.value % items.length
  return Array.from({ length: count }, (_, index) => items[(start + index) % items.length])
})

const rotateFeaturedOutputs = () => {
  const total = filteredOutputs.value.length
  if (total <= 1) return
  featuredRotation.value = (featuredRotation.value + 2) % total
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

const canEditSproutReport = (item?: SproutReportItem | null) => {
  return Boolean(item && (!item.creatorId || !currentUserId.value || item.creatorId === currentUserId.value))
}

const canEditActiveOutputPreview = computed(() => {
  return canEditOutputItem(activeOutputPreview.value)
})

const getOutputStatusMenuOptions = (currentStatus?: OrganizeOutputStatus) => {
  return editableOutputStatusOptions.map((option) => ({
    content: option.label,
    value: option.value,
    disabled: option.value === currentStatus,
    prefixIcon: () =>
      h(TIcon, {
        name: option.value === currentStatus ? 'check-circle-filled' : 'check-circle',
        size: '16px',
      }),
  }))
}

const sproutReportTime = (report: SproutReportItem) => {
  const explicitDate = new Date(report.updatedAt)
  if (!Number.isNaN(explicitDate.getTime())) return explicitDate.getTime()

  const now = new Date()
  const labelDate = new Date(now)
  const timeMatch = report.updated.match(/(\d{1,2}):(\d{2})/)
  const applyTime = (date: Date) => {
    if (timeMatch) {
      date.setHours(Number(timeMatch[1]), Number(timeMatch[2]), 0, 0)
    } else {
      date.setHours(0, 0, 0, 0)
    }
    return date.getTime()
  }

  if (report.updated.includes('今天')) return applyTime(labelDate)
  if (report.updated.includes('昨天')) {
    labelDate.setDate(labelDate.getDate() - 1)
    return applyTime(labelDate)
  }

  const monthDayMatch = report.updated.match(/(\d{1,2})月(\d{1,2})日/)
  if (monthDayMatch) {
    const month = Number(monthDayMatch[1]) - 1
    const day = Number(monthDayMatch[2])
    const parsedDate = new Date(now.getFullYear(), month, day)
    if (parsedDate.getTime() > now.getTime()) {
      parsedDate.setFullYear(parsedDate.getFullYear() - 1)
    }
    return applyTime(parsedDate)
  }

  return Number.NaN
}

const filteredSproutReports = computed(() => {
  const activeRange = sproutRangeTabs.find((tab) => tab.value === sproutRange.value) || sproutRangeTabs[0]
  const now = new Date()
  const start = new Date(now.getTime() - (activeRange.days - 1) * SPROUT_DAY_MS)
  start.setHours(0, 0, 0, 0)
  const end = new Date(now)
  end.setHours(23, 59, 59, 999)
  return sproutReports.value.filter((report) => {
    const reportTime = sproutReportTime(report)
    return !Number.isNaN(reportTime) && reportTime >= start.getTime() && reportTime <= end.getTime()
  })
})

const sproutTotalPages = computed(() => Math.max(1, Math.ceil(filteredSproutReports.value.length / SPROUT_PAGE_SIZE)))

const paginatedSproutReports = computed(() => {
  const page = Math.min(Math.max(sproutPage.value, 1), sproutTotalPages.value)
  const start = (page - 1) * SPROUT_PAGE_SIZE
  return filteredSproutReports.value.slice(start, start + SPROUT_PAGE_SIZE)
})

const sproutMonthGroups = computed(() => {
  return [{ key: 'recent', label: '近期', reports: paginatedSproutReports.value }]
})

const waveHeights = [10, 18, 14, 24, 12, 28, 18, 22]

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

const statusLabel = (status: OrganizeOutputStatus) => ({ draft: '草稿', review: '评审中', ready: '可交付', archived: '已归档' })[status]
const stageLabel = (stage: OrganizeSproutStage) => ({ organizing: '梳理中', expandable: '可扩写', formed: '已成型' })[stage]
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

const sproutCreatorId = (item: OrganizeSproutReport) => {
  const metadata = item.metadata || {}
  return asTrimmedString(item.user_id)
    || asTrimmedString(metadata.creator_id)
    || asTrimmedString(metadata.created_by)
    || asTrimmedString(metadata.user_id)
}

const sproutCreatorName = (item: OrganizeSproutReport) => {
  const metadata = item.metadata || {}
  const explicitName = asTrimmedString(item.creator_name)
    || asTrimmedString(metadata.creator_name)
    || asTrimmedString(metadata.creator_username)
    || asTrimmedString(metadata.author_name)
    || asTrimmedString(metadata.user_name)
  if (explicitName) return explicitName
  const creatorId = sproutCreatorId(item)
  if (!creatorId || creatorId === currentUserId.value) return currentUserDisplayName()
  return '未知用户'
}

const sproutCreatorAvatar = (item: OrganizeSproutReport) => {
  const metadata = item.metadata || {}
  const explicitAvatar = asTrimmedString(item.creator_avatar)
    || asTrimmedString(metadata.creator_avatar)
    || asTrimmedString(metadata.author_avatar)
    || asTrimmedString(metadata.user_avatar)
  if (explicitAvatar) return explicitAvatar
  const creatorId = sproutCreatorId(item)
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

const outputTotalPages = computed(() => Math.max(1, Math.ceil(filteredOutputs.value.length / OUTPUT_PAGE_SIZE)))

const paginatedOutputs = computed(() => {
  const page = Math.min(Math.max(outputPage.value, 1), outputTotalPages.value)
  const start = (page - 1) * OUTPUT_PAGE_SIZE
  return filteredOutputs.value.slice(start, start + OUTPUT_PAGE_SIZE)
})

watch([keyword, discoverTab], () => {
  outputPage.value = 1
  featuredRotation.value = 0
})

watch(filteredOutputs, () => {
  if (outputPage.value > outputTotalPages.value) {
    outputPage.value = outputTotalPages.value
  }
})

watch(filteredSproutReports, () => {
  if (sproutPage.value > sproutTotalPages.value) {
    sproutPage.value = sproutTotalPages.value
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
    title: item.title,
    content: item.content || placeholderContent(item.title),
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

const mapSproutReport = (item: OrganizeSproutReport): SproutReportItem => enrichSproutReport({
  id: item.id,
  title: item.title,
  content: item.summary || placeholderContent(item.title),
  stage: stageLabel(item.stage),
  stageKey: item.stage,
  updated: formatUpdatedLabel(item.updated_at),
  updatedAt: item.updated_at,
  memoryCount: item.memory_count || 0,
  memoryIds: item.memory_ids || [],
  outputHint: item.output_hint || '可继续整理',
  chips: item.chips || [],
  creatorId: sproutCreatorId(item),
  creatorName: sproutCreatorName(item),
  creatorAvatar: sproutCreatorAvatar(item),
  metadata: item.metadata,
  persisted: true,
})

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
    listOrganizeMemories({ page_size: 100 }),
    listOrganizeOutputs({ page_size: 100 }),
    listOrganizeSproutReports({ page_size: 100 }),
  ])

  const [memoryResult, outputResult, sproutResult] = results
  if (memoryResult.status === 'fulfilled' && memoryResult.value.success && memoryResult.value.data.items.length) {
    memoryGroups.value = groupMemoryItems(memoryResult.value.data.items.map(mapMemory))
  }
  if (outputResult.status === 'fulfilled' && outputResult.value.success && outputResult.value.data.items.length) {
    outputs.value = outputResult.value.data.items.map(mapOutput)
  }
  if (sproutResult.status === 'fulfilled' && sproutResult.value.success && sproutResult.value.data.items.length) {
    sproutReports.value = sproutResult.value.data.items.map(mapSproutReport)
  }

  if (results.some((result) => result.status === 'rejected')) {
    MessagePlugin.warning('部分文档数据加载失败，当前展示示例内容')
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

type EditorDocumentType = 'memory' | 'output' | 'sprout'

const editorPath = (documentType: EditorDocumentType, id: string) => {
  return `/platform/organize/editor/${documentType}/${encodeURIComponent(id)}`
}

const editorDraft = (item: MemoryItem | OutputItem | SproutReportItem | undefined): OrganizeEditorDraft | null => {
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
    if (item.tags.length) draft.metadata = { ...draft.metadata, tags: item.tags }
  } else if ('stageKey' in item) {
    draft.stage = item.stageKey
    draft.output_hint = item.outputHint
    draft.chips = item.chips
    draft.memory_ids = item.memoryIds
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
  item?: MemoryItem | OutputItem | SproutReportItem,
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

const isEditableMemory = (item: MemoryItem) => item.type === 'note' || item.type === 'record'

const openMemoryEditor = (item: MemoryListItem | MemoryItem) => {
  if (!isEditableMemory(item)) return
  void openDocumentEditor('memory', item.id, item)
}

const openOutputPreview = (item: OutputItem) => {
  activeOutputPreview.value = item
  outputPreviewVisible.value = true
}

const closeOutputPreview = () => {
  outputPreviewVisible.value = false
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
  if (index === -1) {
    outputs.value = [next, ...outputs.value]
  } else {
    outputs.value = outputs.value.map((candidate) => (candidate.id === next.id ? next : candidate))
  }
  if (activeOutputPreview.value?.id === next.id) {
    activeOutputPreview.value = next
  }
  outputPage.value = 1
}

const handleOutputUploaded = (item: OrganizeOutput) => {
  upsertOutputItem(item)
}

const handleOutputSaved = (item: OrganizeOutput) => {
  upsertOutputItem(item)
  MessagePlugin.success('发现已更新')
}

const openSproutPreview = (item: SproutReportItem) => {
  activeSproutReport.value = item
  sproutPreviewVisible.value = true
}

const closeSproutPreview = () => {
  sproutPreviewVisible.value = false
}

const openSproutEditor = (item: SproutReportItem) => {
  sproutPreviewVisible.value = false
  void openDocumentEditor('sprout', item.id, item)
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
  grid-template-columns: repeat(auto-fit, minmax(128px, 160px));
  gap: 10px;
}

.asset-card {
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  width: 100%;
  max-width: 160px;
  min-height: 58px;
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

.asset-card-value {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
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
  max-width: 920px;
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
  gap: 10px;
  max-width: 920px;
}

.memory-list-card {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12);
  }
}

.memory-list-card--editable,
.memory-row--editable,
.output-card--editable,
.report-card--editable {
  cursor: pointer;

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
  }
}

.memory-list-card-main {
  flex: 1;
  min-width: 0;

  h2 {
    margin: 8px 0 0;
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 500;
    line-height: 23px;
    letter-spacing: 0;
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
  display: flex;
  flex-direction: column;
}

.timeline-date-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 0 12px;

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

.audio-card {
  display: grid;
  grid-template-columns: 20px minmax(140px, 260px) auto;
  align-items: center;
  gap: 12px;
  width: fit-content;
  max-width: 100%;
  margin-top: 14px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  box-sizing: border-box;
}

.audio-wave {
  display: flex;
  align-items: center;
  gap: 3px;
  min-width: 0;

  span {
    width: 3px;
    border-radius: 999px;
    background: var(--td-brand-color);
    opacity: 0.62;
  }
}

.audio-duration {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-weight: 500;
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
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 10px;
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

.output-pagination,
.sprout-pagination {
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

.sprout-pagination {
  padding-top: 8px;
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
    margin: 4px 0 6px;
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
