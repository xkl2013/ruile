<template>
  <div class="organize-page">
    <main class="organize-main">
      <header class="organize-header">
        <div class="organize-header-title">
          <div class="organize-title-row">
            <h2>{{ activeMeta.title }}</h2>
            <t-tooltip :content="activeMeta.actionLabel" placement="bottom">
              <t-button variant="text" theme="default" size="small" class="organize-header-action-btn">
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
              <span>记忆资产（31 条）</span>
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
              <article v-for="item in filteredMemoryAssetItems" :key="item.id" class="memory-list-card">
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
                <button type="button" class="icon-button" :aria-label="`打开 ${item.title}`">
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
                <article v-for="item in group.items" :key="item.id" class="memory-row">
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
            <article v-for="item in outputs" :key="item.id" class="output-card">
              <div class="output-card-icon">
                <t-icon :name="item.icon" />
              </div>
              <div class="output-card-body">
                <div class="output-card-topline">
                  <span class="type-badge">{{ item.type }}</span>
                  <span class="output-status" :class="`output-status--${item.statusKey}`">{{ item.status }}</span>
                </div>
                <h2>{{ item.title }}</h2>
                <div class="output-meta">
                  <span>{{ item.source }}</span>
                  <span>{{ item.updated }}</span>
                </div>
              </div>
              <button type="button" class="icon-button" :aria-label="`打开 ${item.title}`">
                <t-icon name="chevron-right" />
              </button>
            </article>
          </div>
        </section>

        <section v-else class="organize-section">
          <div class="report-list">
            <article v-for="report in sproutReports" :key="report.id" class="report-card">
              <div class="report-main">
                <div class="report-topline">
                  <span class="type-badge">{{ report.stage }}</span>
                  <span>{{ report.updated }}</span>
                </div>
                <h2>{{ report.title }}</h2>
                <div class="report-chips">
                  <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                </div>
                <div class="report-meta">
                  <span>{{ report.memoryCount }} 条记忆</span>
                  <span>{{ report.outputHint }}</span>
                </div>
              </div>
              <button type="button" class="report-action">
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
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ORGANIZE_MEMORY_ASSET_ROUTES,
  findMemoryAssetRoute,
  findOrganizeMenuRoute,
  isMemoryAssetKey,
  isOrganizeTab,
  type MemoryAssetKey,
  type OrganizeTab,
} from './organizeRoutes'

type MemoryType = 'note' | 'record' | 'audio'

interface MemoryItem {
  id: string
  time: string
  type: MemoryType
  typeLabel: string
  title: string
  source?: string
  duration?: string
}

interface MemoryListItem extends MemoryItem {
  date: string
}

interface MemoryGroup {
  date: string
  items: MemoryItem[]
}

const route = useRoute()
const router = useRouter()

const keyword = ref('')
const memoryFilter = ref<'all' | MemoryType>('all')

const activeTab = computed<OrganizeTab>(() => {
  const tab = route.meta.organizeTab
  return isOrganizeTab(tab) ? tab : 'memory'
})

const memoryAssets = ORGANIZE_MEMORY_ASSET_ROUTES

const activeMemoryAsset = computed<MemoryAssetKey | ''>(() => {
  if (activeTab.value !== 'memory') return ''
  const asset = route.meta.memoryAsset
  return isMemoryAssetKey(asset) ? asset : ''
})

const activeMemoryAssetMeta = computed(() => {
  return memoryAssets.find((asset) => asset.key === activeMemoryAsset.value) || memoryAssets[0]
})

const openMemoryAssetList = async (asset: MemoryAssetKey) => {
  const nextRoute = findMemoryAssetRoute(asset)
  if (nextRoute && route.path !== nextRoute.path) {
    await router.push(nextRoute.path)
  }
}

const openMemoryOverview = async () => {
  const memoryRoute = findOrganizeMenuRoute('memory')
  if (memoryRoute && route.path !== memoryRoute.path) {
    await router.push(memoryRoute.path)
  }
}

const activeMeta = computed(() => {
  if (activeTab.value === 'output') {
    return {
      title: '成果',
      countLabel: '12 份文档',
      updatedLabel: '今日更新 3 份',
      actionIcon: 'file-add',
      actionLabel: '新建成果',
    }
  }
  if (activeTab.value === 'sprout') {
    return {
      title: '发芽报告',
      countLabel: '6 份梳理',
      updatedLabel: '待完善 2 份',
      actionIcon: 'setting',
      actionLabel: '发芽报告设置',
    }
  }
  if (activeMemoryAsset.value) {
    return {
      title: `${activeMemoryAssetMeta.value.label}列表`,
      countLabel: `${activeMemoryAssetMeta.value.count} ${activeMemoryAssetMeta.value.unit}`,
      updatedLabel: '记忆资产',
      actionIcon: 'add',
      actionLabel: `新建${activeMemoryAssetMeta.value.label}`,
    }
  }
  return {
    title: '记忆',
    countLabel: '31 条记忆',
    updatedLabel: '今日新增 4 条',
    actionIcon: 'add',
    actionLabel: '新建记忆',
  }
})

const memoryFilterOptions = [
  { label: '全部', value: 'all' },
  { label: '笔记', value: 'note' },
  { label: '记录', value: 'record' },
  { label: '录音', value: 'audio' },
]

const memoryGroups: MemoryGroup[] = [
  {
    date: '6月29日',
    items: [
      { id: 'm1', time: '11:41', type: 'record', typeLabel: '记录', title: '创建了笔记电力行业相关企业分析及功率半导体产业链解读' },
      { id: 'm2', time: '11:27', type: 'record', typeLabel: '记录', title: '创建了笔记能源行业公司分析及中国电力结构探讨' },
      { id: 'm3', time: '11:13', type: 'record', typeLabel: '记录', title: '创建了笔记燃气轮机核心配件与光储行业分析' },
      { id: 'm4', time: '10:58', type: 'record', typeLabel: '记录', title: '创建了笔记燃气轮机行业分析及国内企业发展情况' },
      { id: 'm5', time: '10:22', type: 'audio', typeLabel: '录音', title: '客户访谈：设备更新预算与项目推进节奏', source: '会议录音', duration: '08:36' },
    ],
  },
  {
    date: '6月28日',
    items: [
      { id: 'm6', time: '18:05', type: 'note', typeLabel: '笔记', title: '整理储能项目投标材料中的常见技术指标', source: '手动输入' },
      { id: 'm7', time: '15:42', type: 'note', typeLabel: '笔记', title: '政策口径：新型电力系统与源网荷储协同', source: '网页摘录' },
      { id: 'm8', time: '09:18', type: 'audio', typeLabel: '录音', title: '内部同步：半导体设备国产替代机会', source: '语音记录', duration: '12:04' },
    ],
  },
]

const filteredMemoryGroups = computed(() => {
  const type = memoryFilter.value
  const q = keyword.value.trim().toLowerCase()
  return memoryGroups
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

const allMemoryItems = computed<MemoryListItem[]>(() => {
  return memoryGroups.flatMap((group) =>
    group.items.map((item) => ({
      ...item,
      date: group.date,
    })),
  )
})

const filteredMemoryAssetItems = computed(() => {
  const asset = activeMemoryAsset.value
  const q = keyword.value.trim().toLowerCase()

  return allMemoryItems.value.filter((item) => {
    const assetMatched =
      asset === 'note'
        ? item.type !== 'audio'
        : asset === 'audio' || asset === 'audio-card'
          ? item.type === 'audio'
          : false
    const keywordMatched = !q || item.title.toLowerCase().includes(q) || item.typeLabel.toLowerCase().includes(q)
    return assetMatched && keywordMatched
  })
})

const waveHeights = [10, 18, 14, 24, 12, 28, 18, 22]

const outputs = [
  {
    id: 'o1',
    title: '电力行业相关企业分析及功率半导体产业链解读',
    type: '分析报告',
    source: '来自 8 条记忆',
    updated: '今天 11:48',
    status: '可交付',
    statusKey: 'ready',
    icon: 'file-word',
  },
  {
    id: 'o2',
    title: '能源行业公司分析及中国电力结构探讨',
    type: '研究文档',
    source: '来自 6 条记忆',
    updated: '今天 11:31',
    status: '草稿',
    statusKey: 'draft',
    icon: 'file',
  },
  {
    id: 'o3',
    title: '燃气轮机核心配件供应链梳理',
    type: '方案材料',
    source: '来自 5 条记忆',
    updated: '昨天 19:20',
    status: '评审中',
    statusKey: 'review',
    icon: 'file-paste',
  },
  {
    id: 'o4',
    title: '光储行业重点企业与政策机会清单',
    type: '工作清单',
    source: '来自 9 条记忆',
    updated: '6月27日',
    status: '可交付',
    statusKey: 'ready',
    icon: 'file-excel',
  },
]

const sproutReports = [
  {
    id: 'r1',
    title: '功率半导体产业链的国产替代机会',
    stage: '可扩写',
    updated: '今天 12:10',
    memoryCount: 11,
    outputHint: '可生成分析报告',
    chips: ['产业链', '国产替代', '设备采购'],
  },
  {
    id: 'r2',
    title: '燃气轮机核心配件的供需缺口与企业机会',
    stage: '梳理中',
    updated: '今天 10:46',
    memoryCount: 7,
    outputHint: '缺少企业对比',
    chips: ['核心配件', '进口替代', '项目节奏'],
  },
  {
    id: 'r3',
    title: '新型电力系统政策口径下的储能选题',
    stage: '已成型',
    updated: '昨天 18:22',
    memoryCount: 9,
    outputHint: '已关联成果 2 份',
    chips: ['源网荷储', '储能', '政策'],
  },
]
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
