<template>
  <div class="organize-page">
    <main class="organize-main">
      <header class="organize-header">
        <div class="organize-header-title">
          <div class="organize-title-row">
            <h2>{{ activeMeta.title }}</h2>
          </div>
        </div>
        <div v-if="activeTab !== 'sprout'" class="organize-header-actions">
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
                <span class="asset-card-value"><strong>{{ asset.count }}</strong> {{ asset.unit }}</span>
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

        <section v-else-if="activeTab === 'output'" class="organize-section">
          <div class="content-toolbar">
            <div class="segmented-tabs" role="tablist" aria-label="成果筛选">
              <button
                v-for="tab in outputTabs"
                :key="tab.value"
                type="button"
                class="segmented-tab"
                :class="{ 'segmented-tab--active': outputScope === tab.value }"
                role="tab"
                :aria-selected="outputScope === tab.value"
                @click="setOutputScope(tab.value)"
              >
                {{ tab.label }}
              </button>
            </div>
          </div>

          <div class="output-grid">
            <article
              v-for="item in paginatedOutputs"
              :key="item.id"
              class="output-card output-card--editable"
              role="button"
              tabindex="0"
              @click="openOutputEditor(item)"
              @keydown.enter.self="openOutputEditor(item)"
            >
              <div class="output-card-icon">
                <t-icon :name="item.icon" />
              </div>
              <div class="output-card-body">
                <div class="output-card-topline">
                  <span class="type-badge">{{ item.type }}</span>
                </div>
                <h2>{{ item.title }}</h2>
                <p class="document-excerpt">{{ contentExcerpt(item.content, item.title) }}</p>
                <div class="output-meta">
                  <span class="output-creator" :title="`创建人：${outputCreatorDisplayName(item)}`">
                    <span class="output-creator-avatar">
                      <img v-if="outputCreatorDisplayAvatar(item)" :src="outputCreatorDisplayAvatar(item)" :alt="outputCreatorDisplayName(item)" />
                      <span v-else>{{ creatorInitial(outputCreatorDisplayName(item)) }}</span>
                    </span>
                    <span class="output-creator-name">{{ outputCreatorDisplayName(item) }}</span>
                  </span>
                  <span class="output-updated">{{ item.updated }}</span>
                </div>
              </div>
              <button type="button" class="icon-button" :aria-label="`编辑 ${item.title}`" @click.stop="openOutputEditor(item)">
                <t-icon name="chevron-right" />
              </button>
            </article>
          </div>
          <div v-if="filteredOutputs.length === 0" class="output-empty">
            {{ outputEmptyText }}
          </div>
          <div v-else class="output-pagination" aria-label="成果分页">
            <t-pagination
              v-model="outputPage"
              :page-size="OUTPUT_PAGE_SIZE"
              :total="filteredOutputs.length"
              size="small"
              show-page-number
            />
          </div>
        </section>

        <section v-else class="organize-section organize-section--sprout">
          <div class="sprout-hero">
            <div class="sprout-hero-copy">
              <div class="section-heading">
                <t-icon name="tree-list" />
                <span>发芽记录</span>
              </div>
              <p>把记忆里的碎片整理成可读的报告，先看结论，再看种子和 Aha 瞬间。</p>
            </div>
            <div class="sprout-hero-stats">
              <span><strong>{{ filteredSproutReports.length }}</strong> 份报告</span>
              <span><strong>{{ sproutReports.length }}</strong> 条总记录</span>
            </div>
          </div>

          <div class="sprout-month-list">
            <section v-for="group in sproutMonthGroups" :key="group.key" class="sprout-month-group">
              <div class="sprout-month-header">
                <div class="sprout-month-heading">
                  <h3>{{ group.label }}</h3>
                  <div class="segmented-tabs sprout-range-tabs" role="tablist" aria-label="发芽记录时间筛选">
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
                <span>{{ filteredSproutReports.length }} 份</span>
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
                      <span>发芽</span>
                      <span>报告</span>
                    </div>
                    <div class="sprout-report-date">{{ report.generatedLabel }}</div>
                  </div>

                  <div class="sprout-report-main">
                    <div class="report-topline">
                      <span class="type-badge" :class="`sprout-stage--${report.stageKey}`">{{ report.stage }}</span>
                      <span>{{ report.updated }}</span>
                    </div>
                    <h2>{{ report.title }}</h2>
                    <p class="sprout-report-intro">{{ report.intro }}</p>

                    <div v-if="report.previewSections.length" class="sprout-report-sections">
                      <div
                        v-for="section in report.previewSections.slice(0, 2)"
                        :key="`${report.id}-${section.number}`"
                        class="sprout-report-section"
                      >
                        <div class="sprout-report-section-head">
                          <span>{{ section.number }}</span>
                          <strong>{{ section.title }}</strong>
                        </div>
                        <p>{{ section.seed || section.body || section.aha }}</p>
                      </div>
                    </div>

                    <div v-if="report.chips.length" class="report-chips">
                      <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                    </div>

                    <div class="report-meta sprout-report-meta">
                      <span>{{ report.memoryCount }} 条记忆</span>
                      <span>{{ report.outputHint }}</span>
                    </div>
                  </div>

                  <button type="button" class="report-action" @click.stop="openSproutEditor(report)">
                    <t-icon name="edit-1" />
                    继续整理
                  </button>
                </article>
              </div>

              <div v-else class="output-empty">暂无发芽记录</div>
            </section>
          </div>
          <div v-if="filteredSproutReports.length > SPROUT_PAGE_SIZE" class="sprout-pagination" aria-label="发芽记录分页">
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
      <t-tooltip :content="activeMeta.actionLabel" placement="left">
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
            <div class="sprout-preview-eyebrow">发芽报告</div>
            <div class="sprout-preview-title">{{ activeSproutReport.title }}</div>
          </div>
          <div class="sprout-preview-actions">
            <t-button
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
          <div class="sprout-preview-cover" aria-hidden="true">
            <div class="sprout-preview-cover-mark">
              <span>发芽</span>
              <span>报告</span>
            </div>
            <div class="sprout-preview-cover-card">
              <span class="sprout-preview-cover-label">生成时间</span>
              <span class="sprout-preview-cover-date">{{ activeSproutReport.generatedLabel }}</span>
            </div>
          </div>

          <div class="sprout-preview-body">
            <div class="sprout-preview-meta">
              <span class="type-badge" :class="`sprout-stage--${activeSproutReport.stageKey}`">{{ activeSproutReport.stage }}</span>
              <span>{{ activeSproutReport.updated }}</span>
              <span>{{ activeSproutReport.memoryCount }} 条记忆</span>
            </div>
            <h1>{{ activeSproutReport.title }}</h1>

            <div class="sprout-preview-content" v-html="activeSproutReport.renderedHtml" />
          </div>
        </div>
      </template>
    </t-drawer>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
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

type MemoryType = 'note' | 'record' | 'audio' | 'audio-card'
type OutputScope = 'all' | 'mine' | 'subscribed'
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
  source: string
  updated: string
  status: string
  statusKey: OrganizeOutputStatus
  icon: string
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
  metadata?: Record<string, unknown>
  persisted: boolean
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const keyword = ref('')
const outputScope = ref<OutputScope>('all')
const sproutRange = ref<SproutRange>('3d')
const sproutPreviewVisible = ref(false)
const activeSproutReport = ref<SproutReportItem | null>(null)
const OUTPUT_PAGE_SIZE = 4
const outputPage = ref(1)
const SPROUT_PAGE_SIZE = 10
const sproutPage = ref(1)
const SPROUT_DAY_MS = 24 * 60 * 60 * 1000
const outputTabs: Array<{ label: string; value: OutputScope }> = [
  { label: '全部', value: 'all' },
  { label: '我创建的', value: 'mine' },
  { label: '我订阅的', value: 'subscribed' },
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
  { id: 'demo-o1', title: '电力行业相关企业分析及功率半导体产业链解读', content: '<h2>行业概览</h2><p>从电力设备需求出发，梳理功率半导体产业链及重点企业的竞争位置。</p>', type: '分析报告', source: '来自 8 条记忆', updated: '今天 11:48', status: '可交付', statusKey: 'ready', icon: 'file-word', memoryIds: [], persisted: false },
  { id: 'demo-o2', title: '能源行业公司分析及中国电力结构探讨', content: '<p>分析中国电力结构变化，并比较相关能源公司的业务布局。</p>', type: '研究文档', source: '来自 6 条记忆', updated: '今天 11:31', status: '草稿', statusKey: 'draft', icon: 'file', memoryIds: [], persisted: false },
  { id: 'demo-o3', title: '燃气轮机核心配件供应链梳理', content: '<p>梳理燃气轮机核心配件的供应商、交付周期和国产化进度。</p>', type: '方案材料', source: '来自 5 条记忆', updated: '昨天 19:20', status: '评审中', statusKey: 'review', icon: 'file-paste', memoryIds: [], persisted: false },
  { id: 'demo-o4', title: '光储行业重点企业与政策机会清单', content: '<p>汇总光伏、储能行业重点企业和近期政策机会。</p>', type: '工作清单', source: '来自 9 条记忆', updated: '6月27日', status: '可交付', statusKey: 'ready', icon: 'file-excel', memoryIds: [], persisted: false },
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
    title: '功率半导体产业链的国产替代机会',
    content: '从上游材料、晶圆制造、封装测试和下游应用四个环节，继续扩写国产替代机会。\n\n## 01. 上游材料与设备的确定性\n\n> **🌱 种子**\n> 记忆中多次提到国产替代、设备采购和交付节奏。\n\n国内供应链的短板正在从单点突破转向系统协同，材料、设备和封测能力需要一起观察。\n\n> **✨ Aha 瞬间**\n> 真正的机会不只来自缺口，而来自缺口被补上时的验证速度。',
    stage: '可扩写',
    stageKey: 'expandable',
    updated: '今天 12:10',
    memoryCount: 11,
    memoryIds: [],
    outputHint: '可生成分析报告',
    chips: ['产业链', '国产替代', '设备采购'],
    persisted: false,
  }),
  enrichSproutReport({
    id: 'demo-r2',
    title: '燃气轮机核心配件的供需缺口与企业机会',
    content: '当前素材已覆盖核心配件供需情况，仍需补充国内外企业对比。\n\n## 01. 核心配件的供需缺口\n\n> **🌱 种子**\n> 多条记录指向高温部件、长周期交付和进口替代难度。\n\n这类机会更像慢变量，需要把供应商资质、订单验证和产能爬坡放在同一个框架下看。\n\n> **✨ Aha 瞬间**\n> 产业突破的关键，不是单个参数领先，而是稳定交付能力被客户反复确认。',
    stage: '梳理中',
    stageKey: 'organizing',
    updated: '今天 10:46',
    memoryCount: 7,
    memoryIds: [],
    outputHint: '缺少企业对比',
    chips: ['核心配件', '进口替代', '项目节奏'],
    persisted: false,
  }),
  enrichSproutReport({
    id: 'demo-r3',
    title: '新型电力系统政策口径下的储能选题',
    content: '围绕源网荷储协同和储能商业模式形成后续研究选题。\n\n## 01. 从政策口径到商业模式\n\n> **🌱 种子**\n> 记忆中反复出现源网荷储协同、辅助服务和容量补偿。\n\n储能选题不能只看装机规模，更要拆分收益来源、消纳责任和调度规则。\n\n> **✨ Aha 瞬间**\n> 政策不是背景，而是现金流结构的一部分。',
    stage: '已成型',
    stageKey: 'formed',
    updated: '昨天 18:22',
    memoryCount: 9,
    memoryIds: [],
    outputHint: '已关联成果 2 份',
    chips: ['源网荷储', '储能', '政策'],
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
    return { title: '成果', actionIcon: 'file-add', actionLabel: '新建成果' }
  }
  if (activeTab.value === 'sprout') {
    return { title: '发芽记录', actionIcon: 'add', actionLabel: '新建发芽' }
  }
  if (activeMemoryAsset.value) {
    return { title: `${activeMemoryAssetMeta.value.label}列表`, actionIcon: 'add', actionLabel: '添加笔记' }
  }
  return { title: '记忆', actionIcon: 'add', actionLabel: '添加笔记' }
})

const currentUserId = computed(() => authStore.currentUserId || authStore.user?.id || '')

const setOutputScope = (scope: OutputScope) => {
  outputScope.value = scope
  outputPage.value = 1
}

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

const isCurrentUserOutput = (item: OutputItem) => {
  return !item.creatorId || !currentUserId.value || item.creatorId === currentUserId.value
}

const isOutputVisibleInScope = (item: OutputItem) => {
  if (outputScope.value === 'mine') return isCurrentUserOutput(item)
  if (outputScope.value === 'subscribed') return item.subscribedByMe === true
  return true
}

const filteredOutputs = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return outputs.value.filter((item) => {
    const scopeMatched = isOutputVisibleInScope(item)
    const keywordMatched = !q || `${item.title} ${item.type} ${outputCreatorDisplayName(item)}`.toLowerCase().includes(q)
    return scopeMatched && keywordMatched
  })
})

const outputEmptyText = computed(() => {
  if (outputScope.value === 'mine') return '暂无我创建的成果'
  if (outputScope.value === 'subscribed') return '暂无我订阅的成果'
  return '暂无成果'
})

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
  if (type === 'audio-card') return '录音卡'
  return type === 'record' ? '记录' : '笔记'
}

const statusLabel = (status: OrganizeOutputStatus) => ({ draft: '草稿', review: '评审中', ready: '可交付', archived: '已归档' })[status]
const stageLabel = (stage: OrganizeSproutStage) => ({ organizing: '梳理中', expandable: '可扩写', formed: '已成型' })[stage]

const asTrimmedString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const isTruthyFlag = (value: unknown) => value === true || value === 'true' || value === 1 || value === '1'

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

const outputTotalPages = computed(() => Math.max(1, Math.ceil(filteredOutputs.value.length / OUTPUT_PAGE_SIZE)))

const paginatedOutputs = computed(() => {
  const page = Math.min(Math.max(outputPage.value, 1), outputTotalPages.value)
  const start = (page - 1) * OUTPUT_PAGE_SIZE
  return filteredOutputs.value.slice(start, start + OUTPUT_PAGE_SIZE)
})

watch(keyword, () => {
  outputPage.value = 1
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
  type: item.output_type || '研究文档',
  source: item.source_summary || (item.memory_count ? `来自 ${item.memory_count} 条记忆` : '手动创建'),
  updated: formatUpdatedLabel(item.updated_at),
  status: statusLabel(item.status),
  statusKey: item.status,
  icon: item.icon || 'file-word',
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

const contentExcerpt = (html: string, fallback: string) => {
  if (!html) return fallback
  const body = new DOMParser().parseFromString(html, 'text/html').body
  const firstBlock = Array.from(body.children)[0]
  if (firstBlock?.tagName.toLowerCase() === 'h1') {
    firstBlock.remove()
  }
  const parsed = body.textContent?.trim() || ''
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
    draft.source_summary = item.source
    draft.icon = item.icon
    draft.memory_ids = item.memoryIds
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
  void openDocumentEditor(activeTab.value === 'output' ? 'output' : 'memory')
}

const isEditableMemory = (item: MemoryItem) => item.type === 'note' || item.type === 'record'

const openMemoryEditor = (item: MemoryListItem | MemoryItem) => {
  if (!isEditableMemory(item)) return
  void openDocumentEditor('memory', item.id, item)
}

const openOutputEditor = (item: OutputItem) => {
  void openDocumentEditor('output', item.id, item)
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
  font-size: 14px;
  font-weight: 500;
  line-height: 22px;
}

.section-heading-info {
  color: var(--td-text-color-placeholder);
  font-size: 15px;
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
  min-height: 76px;
  padding: 12px;
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
  font-size: 13px;
  font-weight: 500;
}

.asset-card-value {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;

  strong {
    font-size: 18px;
    font-weight: 600;
    line-height: 22px;
  }
}

.content-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
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
    font-size: 16px;
    font-weight: 500;
    line-height: 24px;
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
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
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
  font-weight: 500;
  line-height: 18px;
  box-sizing: border-box;
}

.source-label {
  color: var(--td-text-color-placeholder);
}

.memory-row-title {
  margin-top: 8px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;
  line-height: 22px;
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
  gap: 4px;
  padding: 4px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.segmented-tab {
  min-width: 58px;
  height: 30px;
  padding: 0 12px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;

  &--active {
    background: var(--td-bg-color-container);
    color: var(--td-text-color-primary);
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.06);
  }
}

.output-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
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
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12);
  }
}

.output-card {
  align-items: stretch;
  height: 196px;
  overflow: hidden;
}

.output-card-topline {
  flex: 0 0 auto;
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
  align-self: stretch;
  overflow: hidden;

  h2 {
    display: -webkit-box;
    flex: 0 0 auto;
    max-height: 46px;
    overflow: hidden;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.document-excerpt {
  display: -webkit-box;
  margin: -4px 0 12px;
  max-height: 40px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
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
  justify-content: flex-start;
  flex-wrap: nowrap;
  margin-top: auto;
}

.output-creator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  max-width: 150px;
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
  gap: 8px;
  margin: 0 0 12px;

  span {
    display: inline-flex;
    min-height: 24px;
    align-items: center;
    padding: 2px 9px;
    border-radius: 999px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 500;
    line-height: 18px;
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
  font-weight: 500;
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
  gap: 18px;
  min-height: 96px;
  padding: 18px 20px;
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
    margin: 8px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 21px;
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
    min-height: 54px;
    justify-content: center;
    padding: 8px 12px;
    border: 1px solid rgba(34, 101, 73, 0.12);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.72);
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
    box-sizing: border-box;
  }

  strong {
    color: #203126;
    font-size: 22px;
    line-height: 26px;
  }
}

.sprout-month-list,
.sprout-month-group {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.sprout-month-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 18px;
    font-weight: 600;
    line-height: 28px;
    letter-spacing: 0;
  }

  span {
    color: var(--td-text-color-placeholder);
    font-size: 13px;
    line-height: 20px;
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

  .segmented-tab {
    min-width: 66px;
  }
}

.sprout-report-list {
  max-width: none;
  gap: 14px;
}

.sprout-report-card {
  display: grid;
  grid-template-columns: 128px minmax(0, 1fr) auto;
  min-height: 230px;
  overflow: hidden;
  border: 1px solid #e1d7c7;
  border-radius: 8px;
  background:
    linear-gradient(0deg, rgba(35, 31, 27, 0.018) 1px, transparent 1px),
    linear-gradient(90deg, rgba(35, 31, 27, 0.014) 1px, transparent 1px),
    #fffdf8;
  background-size: 22px 22px;
  box-shadow: 0 4px 14px rgba(38, 34, 29, 0.05);
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;

  &:hover {
    border-color: rgba(34, 101, 73, 0.42);
    box-shadow: 0 10px 26px rgba(38, 34, 29, 0.1);
    transform: translateY(-1px);
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
  width: 74px;
  height: 88px;
  border: 2px solid #20242a;
  background: rgba(255, 253, 248, 0.72);
  color: #20242a;
  font-family: "Songti SC", "STSong", serif;
  font-size: 27px;
  font-weight: 700;
  line-height: 30px;
  letter-spacing: 0;
  text-align: center;
  box-shadow: inset 0 0 0 1px rgba(32, 36, 42, 0.12);

  span {
    display: block;
  }
}

.sprout-report-date {
  position: absolute;
  right: 12px;
  bottom: 14px;
  left: 12px;
  min-height: 24px;
  padding: 3px 6px;
  border: 1px solid rgba(32, 36, 42, 0.16);
  border-radius: 4px;
  background: rgba(255, 253, 248, 0.72);
  color: #4c463d;
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  text-align: center;
  box-sizing: border-box;
}

.sprout-report-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 18px 18px 16px;

  h2 {
    display: -webkit-box;
    margin: 10px 0 10px;
    max-height: 56px;
    overflow: hidden;
    color: #20242a;
    font-size: 19px;
    font-weight: 700;
    line-height: 28px;
    letter-spacing: 0;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.sprout-report-intro {
  display: -webkit-box;
  margin: 0;
  max-height: 48px;
  overflow: hidden;
  color: #4d5360;
  font-size: 14px;
  line-height: 24px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.sprout-report-sections {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.sprout-report-section {
  min-width: 0;
  min-height: 72px;
  padding: 10px 12px;
  border-left: 3px solid #2d7a52;
  border-radius: 6px;
  background: rgba(34, 101, 73, 0.06);
  box-sizing: border-box;

  p {
    display: -webkit-box;
    margin: 6px 0 0;
    max-height: 38px;
    overflow: hidden;
    color: #596165;
    font-size: 12px;
    line-height: 19px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.sprout-report-section-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    flex: 0 0 24px;
    border-radius: 50%;
    background: #20242a;
    color: #fff;
    font-size: 11px;
    font-weight: 700;
  }

  strong {
    min-width: 0;
    overflow: hidden;
    color: #20242a;
    font-size: 13px;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.sprout-report-meta {
  margin-top: auto;
  padding-top: 12px;
}

.sprout-report-card > .report-action {
  align-self: center;
  margin-right: 18px;
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
  color: #236549;
  font-size: 12px;
  font-weight: 700;
  line-height: 18px;
}

.sprout-preview-title {
  overflow: hidden;
  color: #20242a;
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
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

.sprout-preview-cover {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  min-height: 154px;
  padding: 22px;
  border: 1px solid #ded2bf;
  border-radius: 8px;
  background:
    linear-gradient(0deg, rgba(35, 31, 27, 0.022) 1px, transparent 1px),
    linear-gradient(90deg, rgba(35, 31, 27, 0.018) 1px, transparent 1px),
    #fffdf8;
  background-size: 20px 20px;
  box-sizing: border-box;
}

.sprout-preview-cover-mark {
  display: grid;
  place-items: center;
  width: 124px;
  height: 106px;
  border: 3px solid #20242a;
  color: #20242a;
  font-family: "Songti SC", "STSong", serif;
  font-size: 40px;
  font-weight: 800;
  line-height: 42px;
  letter-spacing: 0;
  text-align: center;
  box-shadow: inset 0 0 0 1px rgba(32, 36, 42, 0.14);

  span {
    display: block;
  }
}

.sprout-preview-cover-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 132px;
  padding: 12px 14px;
  border: 1px solid rgba(32, 36, 42, 0.14);
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.68);
}

.sprout-preview-cover-label {
  color: #81786a;
  font-size: 12px;
  line-height: 18px;
}

.sprout-preview-cover-date {
  color: #20242a;
  font-size: 14px;
  font-weight: 700;
  line-height: 20px;
}

.sprout-preview-body {
  margin-top: 18px;
  padding: 30px 34px 38px;
  border: 1px solid #ded2bf;
  border-radius: 8px;
  background: #fffdf8;
  color: #2a2f36;
  box-shadow: 0 10px 26px rgba(38, 34, 29, 0.08);
  box-sizing: border-box;

  h1 {
    margin: 12px 0 18px;
    color: #20242a;
    font-size: 28px;
    font-weight: 800;
    line-height: 36px;
    letter-spacing: 0;
  }
}

.sprout-preview-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  color: #6c7178;
  font-size: 12px;
  line-height: 18px;
}

.sprout-preview-content {
  color: #2f343b;
  font-size: 15px;
  line-height: 1.78;

  :deep(h1) {
    display: none;
  }

  :deep(h2) {
    margin: 30px 0 12px;
    padding-top: 18px;
    border-top: 1px solid rgba(32, 36, 42, 0.12);
    color: #20242a;
    font-size: 20px;
    font-weight: 800;
    line-height: 30px;
    letter-spacing: 0;
  }

  :deep(h3) {
    margin: 24px 0 10px;
    color: #20242a;
    font-size: 17px;
    font-weight: 700;
    line-height: 26px;
    letter-spacing: 0;
  }

  :deep(p) {
    margin: 10px 0;
  }

  :deep(> p:first-child) {
    margin: 0 0 18px;
    color: #4d5360;
    font-size: 16px;
    line-height: 1.82;
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
    color: #20242a;
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
    grid-template-columns: 108px minmax(0, 1fr);
  }

  .sprout-report-card > .report-action {
    grid-column: 2;
    justify-self: flex-start;
    align-self: flex-start;
    margin: 0 0 16px 18px;
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
    min-height: 96px;
    border-right: 0;
    border-bottom: 1px solid #e6ddcf;
  }

  .sprout-report-ribbon {
    width: 112px;
    height: 64px;
    grid-template-columns: repeat(2, auto);
    gap: 4px;
    font-size: 26px;
    line-height: 28px;
  }

  .sprout-report-date {
    right: 14px;
    bottom: 12px;
    left: auto;
    width: 116px;
  }

  .sprout-report-sections {
    grid-template-columns: 1fr;
  }

  .sprout-report-card > .report-action {
    grid-column: 1;
    margin: 0 18px 18px;
  }

  .sprout-preview-page {
    padding: 16px 14px 36px;
  }

  .sprout-preview-cover {
    align-items: flex-start;
    flex-direction: column;
  }

  .sprout-preview-body {
    padding: 24px 18px 30px;

    h1 {
      font-size: 24px;
      line-height: 32px;
    }
  }
}
</style>
