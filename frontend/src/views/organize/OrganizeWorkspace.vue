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
                          :disabled="isMemorySproutCreating(item.id)"
                          @click.stop="handleMemoryMenuAction(item, 'sprout')"
                        >
                          <OrganizeSproutIcon class="menu-icon memory-sprout-icon" />
                          <span>{{ memorySproutActionLabel(item) }}</span>
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
          <div class="discover-board">
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
                暂无精选
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
                    @click="setDiscoverTab(tab.value)"
                  >
                    {{ tab.label }}
                  </button>
                </div>
              </div>
            </section>

            <section class="discover-feed-section">
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

            <div v-if="!discoverFeedLoading && discoverTotal > OUTPUT_PAGE_SIZE" class="output-pagination" aria-label="发现分页">
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

        <section v-else class="organize-section organize-section--sprout">
          <div class="sprout-hero">
            <div class="sprout-hero-copy">
              <div class="section-heading">
                <OrganizeSproutIcon class="section-heading-icon" />
                <span>经营复盘</span>
              </div>
              <p>沉淀招生、家长沟通和园务管理要点，生成园长可直接复盘和安排跟进的经营建议。</p>
            </div>
            <div class="sprout-hero-stats">
              <span><strong>{{ filteredSproutReports.length }}</strong> 份复盘</span>
              <span><strong>{{ sproutReports.length }}</strong> 条素材</span>
            </div>
          </div>

          <div v-if="sproutLoading" class="organize-loading">
            <t-loading size="medium" text="加载发芽中" />
          </div>
          <div v-else class="sprout-month-list">
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
                    <div class="sprout-report-title-row">
                      <h2>{{ report.title }}</h2>
                      <span class="type-badge" :class="`sprout-stage--${report.stageKey}`">{{ report.stage }}</span>
                    </div>
                    <p class="sprout-report-intro">{{ report.intro }}</p>

                    <div v-if="report.chips.length" class="report-chips">
                      <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                    </div>

                    <div class="report-meta sprout-report-meta">
                      <span>{{ report.updated }}</span>
                      <span v-if="sproutReportReferenceLabels(report).length || sproutReportSourceLabels(report).length" class="sprout-report-meta-separator">|</span>
                      <span v-for="label in sproutReportReferenceLabels(report)" :key="`${report.id}-${label}`">{{ label }}</span>
                      <span v-for="label in sproutReportSourceLabels(report)" :key="`${report.id}-${label}`">{{ label }}</span>
                    </div>
                  </div>

                </article>
              </div>

              <div v-else class="output-empty">暂无经营复盘</div>
            </section>
          </div>
          <div v-if="!sproutLoading && filteredSproutReports.length > SPROUT_PAGE_SIZE" class="sprout-pagination" aria-label="经营复盘分页">
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
        :accept="memoryImportAccept"
        @change="handleMemoryImportFileChange"
      />
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
import { DialogPlugin, Icon as TIcon, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import DocumentPreview from '@/components/document-preview.vue'
import {
  createOrganizeSproutReportFromMemory,
  deleteOrganizeMemory,
  getOrganizeDiscover,
  listOrganizeMemories,
  listOrganizeSproutReports,
  type OrganizeDiscoverTab,
  type OrganizeMemory,
  type OrganizeMemoryReference,
  type OrganizeMemoryKind,
  type OrganizeOutput,
  type OrganizeOutputStatus,
  type OrganizeSproutReport,
  type OrganizeSproutStage,
  updateOrganizeOutput,
  uploadOrganizeMemory,
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
import OrganizeSproutIcon from './components/OrganizeSproutIcon.vue'

type MemoryType = 'note' | 'record' | 'audio' | 'audio-card'
type OutputKind = 'all' | 'article' | 'video' | 'audio'
type OutputCreateKind = Exclude<OutputKind, 'all'>
type OutputStatusFilter = 'all' | OrganizeOutputStatus
type OutputViewMode = 'list' | 'grid'
type SproutRange = '3d' | '7d' | '1m'
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

type MemoryMenuAction = 'edit' | 'sprout' | 'delete'

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
  memoryRefs: OrganizeMemoryReference[]
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
const memoryLoading = ref(true)
const memoryImporting = ref(false)
const memoryImportInputRef = ref<HTMLInputElement | null>(null)
const discoverFeaturedLoading = ref(true)
const discoverFeedLoading = ref(true)
const sproutLoading = ref(true)
const activeMemoryMenuId = ref('')
const sproutingMemoryIds = ref<Set<string>>(new Set())
const outputUploadVisible = ref(false)
const outputUploadInitialKind = ref<OutputCreateKind>('article')
const outputPreviewVisible = ref(false)
const activeOutputPreview = ref<OutputItem | null>(null)
const outputStatusSaving = ref(false)
const discoverTab = ref('recommended')
const discoverTabs = ref<OrganizeDiscoverTab[]>([
  { label: '推荐', value: 'recommended' },
  { label: '图文类', value: 'article' },
  { label: '视频类', value: 'video' },
  { label: '音频类', value: 'audio' },
])
const featuredRotation = ref(0)
const sproutRange = ref<SproutRange>('3d')
const sproutPreviewVisible = ref(false)
const activeSproutReport = ref<SproutReportItem | null>(null)
const OUTPUT_PAGE_SIZE = 30
const FEATURED_OUTPUT_SIZE = 4
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

const memoryGroups = ref<MemoryGroup[]>([])
const outputs = ref<OutputItem[]>([])
const featuredOutputs = ref<OutputItem[]>([])
const discoverTotal = ref(0)
let discoverFeedRequestSeq = 0
let discoverFeaturedRequestSeq = 0

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

const sproutReports = ref<SproutReportItem[]>([])

const setMemorySproutCreating = (id: string, creating: boolean) => {
  const next = new Set(sproutingMemoryIds.value)
  if (creating) {
    next.add(id)
  } else {
    next.delete(id)
  }
  sproutingMemoryIds.value = next
}

const isMemorySproutCreating = (id: string) => sproutingMemoryIds.value.has(id)

const linkedMemorySproutReport = (item: MemoryItem | MemoryListItem) => {
  return sproutReports.value.find((report) => report.persisted && report.memoryIds.includes(item.id))
}

const memorySproutActionLabel = (item: MemoryItem | MemoryListItem) => {
  if (isMemorySproutCreating(item.id)) return '发芽中...'
  const report = linkedMemorySproutReport(item)
  if (report?.stageKey === 'organizing') return '发芽中...'
  if (report?.stageKey === 'formed') return '已发芽'
  return '发芽'
}

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

const visibleMemoryListGroups = computed<MemoryDisplayGroup[]>(() => {
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
  return activeMemoryAsset.value ? `暂无${activeMemoryAssetMeta.value.label}` : '暂无记忆'
})

const activeDiscoverTabLabel = computed(() => {
  return discoverTabs.value.find((tab) => tab.value === discoverTab.value)?.label || '推荐'
})

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

const memoryReferenceLabel = (ref: OrganizeMemoryReference) => {
  const title = asTrimmedString(ref.title) || '未命名'
  return `@创建了${memoryReferenceKindLabel(ref.kind)}${title}`
}

const memoryReferenceSourceLabel = (source?: string) => {
  const normalized = asTrimmedString(source)
  if (!normalized || normalized === '手动输入') return ''
  if (normalized === '文件导入') return '@上传文件'
  return `@${normalized}`
}

const sproutReportReferenceLabels = (report: Pick<SproutReportItem, 'memoryRefs' | 'memoryCount'>) => {
  const refs = report.memoryRefs || []
  if (!refs.length) return report.memoryCount > 0 ? [`@${report.memoryCount}条记忆`] : []
  return refs.slice(0, 2).map(memoryReferenceLabel)
}

const sproutReportSourceLabels = (report: Pick<SproutReportItem, 'memoryRefs'>) => {
  const labels = new Set<string>()
  for (const ref of report.memoryRefs || []) {
    const label = memoryReferenceSourceLabel(ref.source)
    if (label) labels.add(label)
  }
  return Array.from(labels).slice(0, 2)
}

const statusLabel = (status: OrganizeOutputStatus) => ({ draft: '草稿', review: '评审中', ready: '可交付', archived: '已归档' })[status]
const stageLabel = (stage: OrganizeSproutStage) => ({ organizing: '发芽中', expandable: '发芽', formed: '已发芽' })[stage]
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

const memorySproutRoleConfig = () => ({
  role: authStore.currentTenantRole || 'viewer',
  tenant_id: authStore.effectiveTenantId || authStore.currentTenantId || '',
  tenant_name: authStore.currentTenantName || '',
  user_id: currentUserId.value,
  user_name: currentUserDisplayName(),
})

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
  memoryRefs: item.memory_refs || [],
  outputHint: item.output_hint || '可继续整理',
  chips: item.chips || [],
  creatorId: sproutCreatorId(item),
  creatorName: sproutCreatorName(item),
  creatorAvatar: sproutCreatorAvatar(item),
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

const syncActiveSproutPreview = () => {
  if (!activeSproutReport.value) return
  const nextReport = sproutReports.value.find((item) => item.id === activeSproutReport.value?.id)
  if (nextReport) {
    activeSproutReport.value = nextReport
  } else {
    activeSproutReport.value = null
    sproutPreviewVisible.value = false
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

const loadSproutReportsData = async (options?: { silent?: boolean }) => {
  if (!options?.silent) {
    sproutLoading.value = true
  }
  try {
    const response = await listOrganizeSproutReports({ page_size: 100 })
    if (!response.success || !response.data) {
      throw new Error(response.message || '发芽数据加载失败')
    }

    const nextReports = response.data.items.map(mapSproutReport)
    sproutReports.value = nextReports
    syncActiveSproutPreview()
    return nextReports
  } finally {
    if (!options?.silent) {
      sproutLoading.value = false
    }
  }
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
    featuredOutputs.value = data.featured_outputs.map(mapOutput)
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
    outputs.value = data.items.map(mapOutput)
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
    loadDiscoverFeaturedData(),
    loadDiscoverFeedData({ tab: discoverTab.value, page: 1, resetPage: true }),
    loadSproutReportsData(),
  ])

  const [memoryResult, featuredResult, feedResult, sproutResult] = results
  if (featuredResult.status === 'rejected' || feedResult.status === 'rejected') {
    MessagePlugin.warning('发现数据加载失败')
  }

  if (memoryResult.status === 'rejected' || sproutResult.status === 'rejected') {
    MessagePlugin.warning('部分文档数据加载失败')
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

const importMemoryFile = async (file: File) => {
  if (memoryImporting.value) return
  memoryImporting.value = true
  try {
    const response = await uploadOrganizeMemory(file)
    if (!response.success || !response.data) {
      throw new Error(response.message || '文件导入失败')
    }

    await loadMemoryData()
    MessagePlugin.success('已导入为笔记')
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
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  void importMemoryFile(file)
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

const upsertSproutReportItem = (item: OrganizeSproutReport) => {
  const next = mapSproutReport(item)
  const index = sproutReports.value.findIndex((candidate) => candidate.id === next.id)
  if (index !== -1) {
    sproutReports.value = sproutReports.value.map((candidate) => (candidate.id === next.id ? next : candidate))
  } else {
    sproutReports.value = [next, ...sproutReports.value]
  }
  if (activeSproutReport.value?.id === next.id) {
    activeSproutReport.value = next
  }
  return next
}

const scheduleSproutReportRefresh = () => {
  const refresh = () => {
    void loadSproutReportsData({ silent: true }).catch(() => {
      // 创建后短轮询只用于同步异步生成结果，失败时保留当前“发芽中”状态。
    })
  }
  window.setTimeout(refresh, 1500)
  window.setTimeout(refresh, 5000)
}

const createSproutFromMemory = async (item: MemoryItem) => {
  const linkedReport = linkedMemorySproutReport(item)
  if (linkedReport) {
    return
  }
  if (isMemorySproutCreating(item.id)) return
  if (!item.persisted) {
    MessagePlugin.warning('该记忆暂不支持发芽')
    return
  }

  setMemorySproutCreating(item.id, true)
  try {
    const response = await createOrganizeSproutReportFromMemory({
      memory_id: item.id,
      role_config: memorySproutRoleConfig(),
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '发芽任务创建失败')
    }

    upsertSproutReportItem(response.data)
    MessagePlugin.success('发芽任务已创建')
    scheduleSproutReportRefresh()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发芽任务创建失败')
  } finally {
    setMemorySproutCreating(item.id, false)
  }
}

const handleMemoryMenuAction = (item: MemoryItem, action: MemoryMenuAction) => {
  if (action === 'edit') {
    closeMemoryMenu()
    openMemoryEditor(item)
    return
  }
  if (action === 'sprout') {
    void createSproutFromMemory(item)
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

.memory-sprout-icon {
  display: block;
  width: 16px;
  height: 16px;
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
