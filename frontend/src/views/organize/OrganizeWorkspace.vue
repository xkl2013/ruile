<template>
  <div class="organize-page">
    <main class="organize-main">
      <header class="organize-header">
        <div class="organize-header-title">
          <div class="organize-title-row">
            <h2>{{ activeMeta.title }}</h2>
            <t-tooltip :content="activeMeta.actionLabel" placement="bottom">
              <t-button
                variant="text"
                theme="default"
                size="small"
                class="organize-header-action-btn"
                @click="handleHeaderAction"
              >
                <template #icon><t-icon :name="activeMeta.actionIcon" size="16px" /></template>
              </t-button>
            </t-tooltip>
          </div>
          <div class="organize-header-meta">
            <span>{{ activeMeta.countLabel }}</span>
            <span>{{ activeMeta.updatedLabel }}</span>
          </div>
        </div>
        <div class="organize-header-actions">
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
            <div class="timeline-toolbar">
              <div class="section-heading">
                <t-icon name="time" />
                <span>时间线</span>
              </div>
              <t-select v-model="memoryFilter" class="organize-filter" :options="memoryFilterOptions" />
            </div>

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
              <button type="button" class="segmented-tab segmented-tab--active">全部</button>
              <button type="button" class="segmented-tab">文档</button>
              <button type="button" class="segmented-tab">报告</button>
              <button type="button" class="segmented-tab">方案</button>
            </div>
            <t-button variant="outline">
              <template #icon><t-icon name="filter" /></template>
              筛选
            </t-button>
          </div>

          <div class="output-grid">
            <article
              v-for="item in filteredOutputs"
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
                  <span class="output-status" :class="`output-status--${item.statusKey}`">{{ item.status }}</span>
                </div>
                <h2>{{ item.title }}</h2>
                <p class="document-excerpt">{{ contentExcerpt(item.content, item.title) }}</p>
                <div class="output-meta">
                  <span>{{ item.source }}</span>
                  <span>{{ item.updated }}</span>
                </div>
              </div>
              <button type="button" class="icon-button" :aria-label="`编辑 ${item.title}`" @click.stop="openOutputEditor(item)">
                <t-icon name="chevron-right" />
              </button>
            </article>
          </div>
        </section>

        <section v-else class="organize-section">
          <div class="report-list">
            <article
              v-for="report in filteredSproutReports"
              :key="report.id"
              class="report-card report-card--editable"
              role="button"
              tabindex="0"
              @click="openSproutEditor(report)"
              @keydown.enter.self="openSproutEditor(report)"
            >
              <div class="report-main">
                <div class="report-topline">
                  <span class="type-badge">{{ report.stage }}</span>
                  <span>{{ report.updated }}</span>
                </div>
                <h2>{{ report.title }}</h2>
                <p class="document-excerpt">{{ contentExcerpt(report.content, report.title) }}</p>
                <div class="report-chips">
                  <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                </div>
                <div class="report-meta">
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
        </section>
      </div>
    </main>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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

type MemoryType = 'note' | 'record' | 'audio' | 'audio-card'

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
  metadata?: Record<string, unknown>
  persisted: boolean
}

interface SproutReportItem {
  id: string
  title: string
  content: string
  stage: string
  stageKey: OrganizeSproutStage
  updated: string
  memoryCount: number
  memoryIds: string[]
  outputHint: string
  chips: string[]
  metadata?: Record<string, unknown>
  persisted: boolean
}

const route = useRoute()
const router = useRouter()

const keyword = ref('')
const memoryFilter = ref<'all' | MemoryType>('all')

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

const fallbackSproutReports: SproutReportItem[] = [
  { id: 'demo-r1', title: '功率半导体产业链的国产替代机会', content: '<p>从上游材料、晶圆制造、封装测试和下游应用四个环节，继续扩写国产替代机会。</p>', stage: '可扩写', stageKey: 'expandable', updated: '今天 12:10', memoryCount: 11, memoryIds: [], outputHint: '可生成分析报告', chips: ['产业链', '国产替代', '设备采购'], persisted: false },
  { id: 'demo-r2', title: '燃气轮机核心配件的供需缺口与企业机会', content: '<p>当前素材已覆盖核心配件供需情况，仍需补充国内外企业对比。</p>', stage: '梳理中', stageKey: 'organizing', updated: '今天 10:46', memoryCount: 7, memoryIds: [], outputHint: '缺少企业对比', chips: ['核心配件', '进口替代', '项目节奏'], persisted: false },
  { id: 'demo-r3', title: '新型电力系统政策口径下的储能选题', content: '<p>围绕源网荷储协同和储能商业模式形成后续研究选题。</p>', stage: '已成型', stageKey: 'formed', updated: '昨天 18:22', memoryCount: 9, memoryIds: [], outputHint: '已关联成果 2 份', chips: ['源网荷储', '储能', '政策'], persisted: false },
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
    return { title: '成果', countLabel: `${outputs.value.length} 份文档`, updatedLabel: '点击内容即可编辑', actionIcon: 'file-add', actionLabel: '新建成果' }
  }
  if (activeTab.value === 'sprout') {
    return { title: '发芽', countLabel: `${sproutReports.value.length} 份梳理`, updatedLabel: '点击内容即可编辑', actionIcon: 'add', actionLabel: '新建发芽' }
  }
  if (activeMemoryAsset.value) {
    return { title: `${activeMemoryAssetMeta.value.label}列表`, countLabel: `${activeMemoryAssetMeta.value.count} ${activeMemoryAssetMeta.value.unit}`, updatedLabel: '记忆资产', actionIcon: 'add', actionLabel: '添加笔记' }
  }
  return { title: '记忆', countLabel: `${allMemoryItems.value.length} 条记忆`, updatedLabel: '笔记支持创建和编辑', actionIcon: 'add', actionLabel: '添加笔记' }
})

const memoryFilterOptions = [
  { label: '全部', value: 'all' },
  { label: '笔记', value: 'note' },
  { label: '记录', value: 'record' },
  { label: '录音', value: 'audio' },
]

const filteredMemoryGroups = computed(() => {
  const type = memoryFilter.value
  const q = keyword.value.trim().toLowerCase()
  return memoryGroups.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        const typeMatched = type === 'all' || item.type === type
        const keywordMatched = !q || item.title.toLowerCase().includes(q) || item.typeLabel.toLowerCase().includes(q)
        return typeMatched && keywordMatched
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
  return outputs.value.filter((item) => !q || `${item.title} ${item.type} ${item.source}`.toLowerCase().includes(q))
})

const filteredSproutReports = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  return sproutReports.value.filter((item) => !q || `${item.title} ${item.chips.join(' ')}`.toLowerCase().includes(q))
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
  metadata: item.metadata,
  persisted: true,
})

const mapSproutReport = (item: OrganizeSproutReport): SproutReportItem => ({
  id: item.id,
  title: item.title,
  content: item.summary || placeholderContent(item.title),
  stage: stageLabel(item.stage),
  stageKey: item.stage,
  updated: formatUpdatedLabel(item.updated_at),
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

const handleHeaderAction = () => {
  void openDocumentEditor(activeTab.value === 'output' ? 'output' : activeTab.value === 'sprout' ? 'sprout' : 'memory')
}

const isEditableMemory = (item: MemoryItem) => item.type === 'note' || item.type === 'record'

const openMemoryEditor = (item: MemoryListItem | MemoryItem) => {
  if (!isEditableMemory(item)) return
  void openDocumentEditor('memory', item.id, item)
}

const openOutputEditor = (item: OutputItem) => {
  void openDocumentEditor('output', item.id, item)
}

const openSproutEditor = (item: SproutReportItem) => {
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

.organize-header-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  margin-top: 0;
  color: var(--td-text-color-placeholder);
  font-family: var(--app-font-family);
  font-size: 13px;
  font-weight: 400;
  line-height: 19px;
}

.organize-header-action-btn {
  padding: 0 !important;
  min-width: 28px !important;
  width: 28px !important;
  height: 28px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  background: var(--td-bg-color-secondarycontainer) !important;
  border: 1px solid var(--td-component-stroke) !important;
  border-radius: 6px !important;
  color: var(--td-text-color-secondary);

  &:hover {
    color: var(--td-text-color-primary);
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

.timeline-toolbar,
.content-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.organize-filter {
  width: 128px;
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

.document-excerpt {
  display: -webkit-box;
  margin: -4px 0 12px;
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
  gap: 6px 14px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.output-status {
  margin-left: auto;
  font-weight: 500;

  &--ready {
    color: var(--td-success-color);
  }

  &--draft {
    color: var(--td-warning-color);
  }

  &--review {
    color: var(--td-brand-color);
  }
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
}

@media (max-width: 760px) {
  .organize-header,
  .content-toolbar,
  .timeline-toolbar {
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

  .memory-row {
    grid-template-columns: 1fr;
  }
}
</style>
