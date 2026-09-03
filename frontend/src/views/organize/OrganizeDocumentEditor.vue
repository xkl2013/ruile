<template>
  <div class="organize-editor-page theme-notion">
    <header class="organize-editor-header">
      <div class="editor-header-left">
        <button type="button" class="editor-back-button" :aria-label="`返回${returnLabel}`" @click="goBack">
          <t-icon name="chevron-left" />
          <span>返回</span>
        </button>

        <div class="editor-breadcrumbs" aria-label="文档位置">
          <template v-for="(item, index) in breadcrumbItems" :key="`${item}-${index}`">
            <span>{{ item }}</span>
            <t-icon name="chevron-right" />
          </template>
          <span class="editor-breadcrumb-current">{{ title || '无标题' }}</span>
        </div>
      </div>

      <div class="editor-page-actions">
        <t-button
          v-if="isNoteMemory"
          theme="default"
          variant="outline"
          class="editor-service-action"
          :loading="noteServiceExtracting"
          :disabled="saving || loading"
          @click="extractCurrentMemoryToService"
        >
          <template #icon><img src="@/assets/img/agent-green.svg" class="editor-service-action-icon" alt="" aria-hidden="true" /></template>
          提取服务
        </t-button>
        <t-button
          v-if="isNoteMemory"
          theme="default"
          variant="outline"
          class="editor-sprout-action"
          :class="`editor-sprout-action--${noteSproutHeaderState}`"
          :loading="noteSproutCreating"
          :disabled="noteSproutLoading"
          :aria-label="noteSproutHeaderAriaLabel"
          @click="handleMemorySproutHeaderAction"
        >
          <template #icon><OrganizeSproutIcon class="editor-sprout-action-icon" /></template>
          {{ noteSproutHeaderLabel }}
        </t-button>
        <span class="editor-save-state" :class="`editor-save-state--${saveState}`" aria-live="polite">{{ saveStateLabel }}</span>
      </div>
    </header>

    <main class="organize-editor-main">
      <div v-if="loading" class="editor-page-state">
        <t-loading size="small" />
        <span>正在加载文档</span>
      </div>

      <div v-else-if="loadError" class="editor-page-state editor-page-state--error">
        <t-icon name="error-circle" size="24px" />
        <span>{{ loadError }}</span>
        <t-button theme="primary" variant="outline" @click="goBack">返回列表</t-button>
      </div>

      <div v-else class="editor-page-content">
        <article class="document-page">
          <template v-if="isNoteMemory">
            <section class="memory-note-panel" aria-label="笔记详情">
              <div class="memory-note-header">
                <t-input v-model="title" class="memory-note-title-input" size="large" clearable placeholder="笔记标题" />
              </div>

              <div class="memory-note-tags">
                <div class="memory-note-tag-list">
                  <t-tag
                    v-for="tag in memoryTags"
                    :key="tag"
                    class="memory-note-tag"
                    size="small"
                    variant="light-outline"
                  >
                    <t-icon v-if="noteTagIcon(tag)" class="memory-note-tag-leading-icon" :name="noteTagIcon(tag)" size="14px" />
                    <span class="memory-note-tag-text">{{ tag }}</span>
                    <button type="button" class="memory-note-tag-remove" :aria-label="`移除 ${tag}`" @click="removeMemoryTag(tag)">
                      <t-icon name="close" />
                    </button>
                  </t-tag>
                </div>

                <div class="memory-note-tag-actions">
                  <t-popup
                    v-model:visible="noteTagMenuVisible"
                    trigger="click"
                    placement="bottom-left"
                    destroy-on-close
                    overlayClassName="memory-note-tag-popup"
                  >
                    <t-button class="memory-note-tag-action memory-note-tag-add-button" variant="outline" theme="default" size="small">
                      <template #icon><t-icon name="add" /></template>
                      添加标签
                    </t-button>
                    <template #content>
                      <div class="memory-note-tag-panel" @click.stop>
                        <t-input
                          v-model="noteTagDraft"
                          clearable
                          placeholder="输入标签后回车"
                          @keydown.enter.prevent="addMemoryTag()"
                        />
                        <div class="memory-note-tag-panel-actions">
                          <t-button theme="primary" size="small" @click="addMemoryTag()">添加</t-button>
                          <t-button theme="default" variant="outline" size="small" @click="noteTagMenuVisible = false">关闭</t-button>
                        </div>
                      </div>
                    </template>
                  </t-popup>

                  <t-button class="memory-note-tag-action memory-note-tag-smart-button" variant="outline" theme="default" size="small" @click="generateMemoryTags">
                    <template #icon><t-icon name="add" /></template>
                    智能标签
                  </t-button>
	                </div>
	              </div>

              <button
                v-if="sourceFileCardVisible"
                type="button"
                class="memory-note-source-card"
                :aria-label="`预览源文件 ${sourceFileName}`"
                @click="openSourceFilePreview"
              >
                <span class="memory-note-source-icon" aria-hidden="true">
                  <t-icon :name="sourceFileIcon" size="18px" />
                </span>
                <span class="memory-note-source-main">
                  <span class="memory-note-source-label">源文件</span>
                  <span class="memory-note-source-name">{{ sourceFileName }}</span>
                </span>
                <span v-if="sourceFileKindLabel" class="memory-note-source-meta">{{ sourceFileKindLabel }}</span>
                <t-icon class="memory-note-source-arrow" name="file-view" size="16px" />
              </button>

              <div class="memory-note-tabs" role="tablist" aria-label="笔记视图切换">
                <button
                  type="button"
                  class="memory-note-tab"
                  :class="{ 'is-active': noteActiveTab === 'content' }"
                  @click="noteActiveTab = 'content'"
                >
                  笔记内容
                </button>
                <button
                  type="button"
                  class="memory-note-tab"
                  :class="{ 'is-active': noteActiveTab === 'service' }"
                  @click="noteActiveTab = 'service'"
                >
                  服务
                </button>
                <button
                  type="button"
                  class="memory-note-tab"
                  :class="{ 'is-active': noteActiveTab === 'sprout' }"
                  @click="noteActiveTab = 'sprout'"
                >
                  发芽
                </button>
              </div>

              <div class="memory-note-tab-panels">
                <section v-show="noteActiveTab === 'content'" class="memory-note-panel-view">
                  <div
                    class="document-editor-shell document-editor-shell--memory-note"
                    @keydown.capture="handleEditorKeydown"
                  >
                    <TiptapProEditor
                      :key="editorKey"
                      ref="editorRef"
                      v-model="content"
                      :version="editorVersion"
                      theme-preset="notion"
                      locale="zh-CN"
                      :placeholder="editorPlaceholder"
                      :features="editorFeatures"
                    />
                  </div>
                </section>

                <section v-show="noteActiveTab === 'service'" class="memory-note-panel-view memory-note-panel-view--service">
                  <article v-if="noteServiceTask" class="memory-note-service-card">
                    <div class="memory-note-service-head">
                      <div class="memory-note-service-head-copy">
                        <span>客户摘要</span>
                        <div>
                          <h2>{{ noteServiceTask.customerName }}</h2>
                          <em :class="`priority-${noteServiceTask.priorityKey}`">
                            {{ noteServiceTask.stage }} · {{ noteServiceTask.confidenceLabel }}
                          </em>
                        </div>
                      </div>
                    </div>

                    <p class="memory-note-service-summary">{{ noteServiceTask.summary }}</p>

                    <dl class="memory-note-service-facts">
                      <div v-for="fact in noteServiceFacts" :key="fact.label">
                        <dt>{{ fact.label }}</dt>
                        <dd>{{ fact.value }}</dd>
                      </div>
                    </dl>

                    <section class="memory-note-service-section">
                      <h3>有利于销售的信息</h3>
                      <ul>
                        <li v-for="item in noteServiceTask.salesHighlights" :key="item">{{ item }}</li>
                      </ul>
                    </section>

                    <section class="memory-note-service-section">
                      <h3>建议动作</h3>
                      <p>{{ noteServiceTask.primaryAction }}</p>
                      <blockquote>{{ noteServiceTask.replyDraft }}</blockquote>
                    </section>

                    <section class="memory-note-service-section">
                      <h3>来源关联</h3>
                      <div class="memory-note-service-source-list">
                        <button
                          v-if="noteServiceSourceMemory"
                          type="button"
                          class="memory-note-service-source"
                          @click="noteActiveTab = 'content'"
                        >
                          <span class="memory-note-service-source-icon"><t-icon name="file" size="16px" /></span>
                          <span>
                            <strong>{{ noteServiceSourceMemory.title }}</strong>
                            <em>{{ noteServiceSourceMemory.sourceLabel }} · {{ noteServiceSourceMemory.dateLabel }} · {{ noteServiceSourceMemory.idLabel }}</em>
                          </span>
                        </button>
                        <button
                          v-if="sourceFileCardVisible"
                          type="button"
                          class="memory-note-service-source"
                          @click="openSourceFilePreview"
                        >
                          <span class="memory-note-service-source-icon"><t-icon :name="sourceFileIcon" size="16px" /></span>
                          <span>
                            <strong>{{ sourceFileName }}</strong>
                            <em>源文件 · {{ sourceFileKindLabel }}</em>
                          </span>
                        </button>
                      </div>
                    </section>
                  </article>

                  <div v-else class="memory-note-service-empty">
                    <t-icon name="search" size="24px" />
                    <strong>暂无服务资料</strong>
                    <p>这条记忆没有识别到客户身份和销售服务信号。</p>
                  </div>
                </section>

                <section v-show="noteActiveTab === 'sprout'" class="memory-note-panel-view memory-note-panel-view--sprout">
                  <div v-if="noteSproutLoading" class="memory-note-sprout-loading">
                    <t-loading size="small" text="加载发芽中" />
                  </div>
                  <article
                    v-else-if="noteSproutCard"
                    class="memory-note-sprout-card sprout-report-card"
                    role="button"
                    tabindex="0"
                    @click="openNoteSproutReport"
                    @keydown.enter.self="openNoteSproutReport"
                  >
                    <div class="sprout-report-gutter" aria-hidden="true">
                      <div class="sprout-report-ribbon">
                        <span>经营</span>
                        <span>复盘</span>
                      </div>
                    </div>

                    <div class="sprout-report-main">
                      <div class="sprout-report-title-row">
                        <h2>{{ noteSproutCard.title }}</h2>
                        <span class="type-badge" :class="`sprout-stage--${noteSproutCard.stageKey}`">{{ noteSproutCard.stage }}</span>
                      </div>
                      <p class="sprout-report-intro">{{ noteSproutCard.intro }}</p>

                      <div v-if="noteSproutCard.chips.length" class="report-chips">
                        <span v-for="chip in noteSproutCard.chips" :key="chip">{{ chip }}</span>
                      </div>

                      <div class="report-meta sprout-report-meta">
                        <span>{{ noteSproutCard.updated }}</span>
                        <span
                          v-if="noteSproutCard.referenceLabels.length || noteSproutCard.sourceLabels.length"
                          class="sprout-report-meta-separator"
                        >
                          |
                        </span>
                        <span v-for="label in noteSproutCard.referenceLabels" :key="`note-sprout-ref-${label}`">{{ label }}</span>
                        <span v-for="label in noteSproutCard.sourceLabels" :key="`note-sprout-source-${label}`">{{ label }}</span>
                      </div>
                    </div>
                  </article>
                </section>
              </div>
            </section>
          </template>

          <template v-else-if="documentType === 'output'">
            <section class="output-category-panel" aria-label="发现栏目">
              <div class="output-category-copy">
                <span class="output-category-eyebrow">内容归类</span>
                <strong>选择发现栏目</strong>
              </div>
              <t-select
                v-model="outputCategory"
                :options="discoverCategoryOptions"
                placeholder="请选择发现栏目"
                @change="markDocumentDirty"
              />
            </section>
            <div
              class="document-editor-shell"
              @keydown.capture="handleEditorKeydown"
            >
              <TiptapProEditor
                :key="editorKey"
                ref="editorRef"
                v-model="content"
                :version="editorVersion"
                theme-preset="notion"
                locale="zh-CN"
                :placeholder="editorPlaceholder"
                :features="editorFeatures"
              />
            </div>
          </template>

          <template v-else>
            <section v-if="isAudioMemory" class="memory-audio-panel" aria-label="录音详情">
              <t-input v-model="title" class="memory-audio-title-input" size="large" clearable placeholder="录音标题" />
              <div class="memory-audio-player" aria-label="录音播放">
                <template v-if="audioSourceUrl">
                  <audio class="memory-audio-native" :src="audioSourceUrl" controls preload="metadata"></audio>
                  <div class="memory-audio-transcript-chip">
                    <t-icon name="file-word" size="14px" />
                    <span>文稿</span>
                  </div>
                </template>
                <template v-else>
                  <button type="button" class="memory-audio-play" aria-label="播放录音">
                    <t-icon name="play-circle" size="18px" />
                  </button>
                  <div class="memory-audio-track-wrap">
                    <div class="memory-audio-track" aria-hidden="true">
                      <span class="memory-audio-thumb"></span>
                      <span class="memory-audio-progress"></span>
                    </div>
                    <div class="memory-audio-time-row">
                      <span>00:00</span>
                      <span>{{ audioDurationLabel }}</span>
                    </div>
                  </div>
                  <div class="memory-audio-transcript-chip">
                    <t-icon name="file-word" size="14px" />
                    <span>文稿</span>
                  </div>
                </template>
              </div>
            </section>
            <div
              class="document-editor-shell"
              :class="{ 'document-editor-shell--audio': isAudioMemory }"
              @keydown.capture="handleEditorKeydown"
            >
              <TiptapProEditor
                :key="editorKey"
                ref="editorRef"
                v-model="content"
                :version="editorVersion"
                theme-preset="notion"
                locale="zh-CN"
                :placeholder="editorPlaceholder"
                :features="editorFeatures"
              />
            </div>
          </template>
        </article>
      </div>
    </main>

    <t-drawer
      v-model:visible="sourcePreviewVisible"
      class="memory-source-preview-drawer"
      :header="sourceFileName || '源文件预览'"
      :footer="false"
      :close-btn="false"
      size="min(860px, 92vw)"
      destroy-on-close
    >
      <section v-if="sourcePreviewVisible && sourceFilePreviewUrl" class="memory-source-preview-body">
        <DocumentPreview
          :source-url="sourceFilePreviewUrl"
          :file-type="sourceFilePreviewType"
          :file-name="sourceFileName || '源文件'"
          :active="sourcePreviewVisible"
          fill-height
        />
      </section>
    </t-drawer>

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
      <template v-if="activeNoteSproutPreview">
        <div class="sprout-preview-header">
          <div class="sprout-preview-header-copy">
            <div class="sprout-preview-eyebrow">经营复盘</div>
            <div class="sprout-preview-title">{{ activeNoteSproutPreview.title }}</div>
          </div>
          <div class="sprout-preview-actions">
            <t-button
              variant="text"
              theme="default"
              size="small"
              class="sprout-preview-action"
              aria-label="编辑报告"
              @click="openNoteSproutEditor"
            >
              <template #icon><t-icon name="edit-1" size="16px" /></template>
            </t-button>
            <t-button
              variant="text"
              theme="default"
              size="small"
              class="sprout-preview-action"
              aria-label="关闭预览"
              @click="closeNoteSproutPreview"
            >
              <template #icon><t-icon name="close" size="16px" /></template>
            </t-button>
          </div>
        </div>

        <div class="sprout-preview-page">
          <div class="sprout-preview-body">
            <div class="sprout-preview-meta">
              <span>{{ activeNoteSproutPreview.updated }}</span>
              <span>{{ activeNoteSproutPreview.memoryCount }} 条记忆</span>
            </div>
            <h1>{{ activeNoteSproutPreview.title }}</h1>

            <div class="sprout-preview-content" v-html="activeNoteSproutPreview.renderedHtml" />
          </div>
        </div>
      </template>
    </t-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { TiptapProEditor, type FeatureConfig, type TiptapProEditorExpose } from 'tiptap-ui-kit'
import 'tiptap-ui-kit/style.css'
import DocumentPreview from '@/components/document-preview.vue'
import {
  createOrganizeMemory,
  createOrganizeSproutReportFromMemory,
  createOrganizeOutput,
  createOrganizeSproutReport,
  getOrganizeMemory,
  getOrganizeOutput,
  getOrganizeSproutReport,
  listOrganizeSproutReports,
  updateOrganizeMemory,
  updateOrganizeOutput,
  updateOrganizeSproutReport,
  type OrganizeMemory,
  type OrganizeMemoryReference,
  type OrganizeMemoryKind,
  type OrganizeOutput,
  type OrganizeOutputStatus,
  type OrganizeSproutReport,
  type OrganizeSproutStage,
} from '@/api/organize'
import { useAuthStore } from '@/stores/auth'
import {
  clearOrganizeEditorDraft,
  readOrganizeEditorDraft,
  type OrganizeEditorDraft,
} from './editorDraftStorage'
import { buildSproutReportPreview, sproutReportContentForEditor } from './sproutReport'
import {
  buildSmartNoteTags,
  mergeNoteMetadata,
  normalizeNoteTags,
} from './noteEditor'
import {
  buildServiceTaskFromMemory,
  formatMemoryDateLabel,
  type ServiceTask,
} from '../service/serviceMemoryExtraction'
import { extractServiceMemory } from '@/api/service'
import {
  DISCOVER_CATEGORY_OPTIONS,
  discoverCategoryLabel,
  normalizeDiscoverCategory,
  type DiscoverCategoryKey,
} from './discoverCategories'

type OrganizeDocumentType = 'memory' | 'output' | 'sprout'
type SaveState = 'idle' | 'saving' | 'saved' | 'waiting' | 'error'

const AUTOSAVE_DELAY = 700

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()

const title = ref('')
const emptyDocumentContent = '<h1></h1><p></p>'
const content = ref(emptyDocumentContent)
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const saveError = ref('')
const saveState = ref<SaveState>('idle')
const editorReady = ref(false)
const editorKey = ref(0)
const editorRef = ref<TiptapProEditorExpose | null>(null)
let autosaveTimer: ReturnType<typeof setTimeout> | null = null
let autosavePending = false
let editRevision = 0
let skipNextRouteLoad = false
const savedDocumentId = ref('')

const memoryKind = ref<OrganizeMemoryKind>('note')
const memorySource = ref('手动输入')
const memoryDurationSeconds = ref(0)
const memoryMetadata = ref<Record<string, unknown> | undefined>()
const memoryTags = ref<string[]>([])
const memoryOccurredAt = ref('')
const memoryCreatedAt = ref('')
const memoryUpdatedAt = ref('')
const noteTagDraft = ref('')
const noteTagMenuVisible = ref(false)
const noteSproutReport = ref<OrganizeSproutReport | null>(null)
const noteSproutCreating = ref(false)
const noteSproutLoading = ref(false)
const noteSproutStatus = ref<OrganizeSproutStage | ''>('')
const noteActiveTab = ref<'content' | 'service' | 'sprout'>('content')
const noteServiceExtracting = ref(false)
const sourcePreviewVisible = ref(false)
const sproutPreviewVisible = ref(false)
const outputDraft = ref<OrganizeOutput | null>(null)
const outputCategory = ref<DiscoverCategoryKey | ''>('')
const sproutDraft = ref<OrganizeSproutReport | null>(null)
const discoverCategoryOptions = DISCOVER_CATEGORY_OPTIONS
let noteSproutRequestSeq = 0

const editorFeatures: FeatureConfig = {
  headerNav: false,
  footerNav: false,
  floatingMenu: true,
  linkBubbleMenu: true,
  image: false,
  table: false,
  tableToolbar: false,
  slashCommand: true,
  dragHandle: true,
  dragHandleMenu: true,
  aiChat: false,
  aiSettings: false,
  collaboration: false,
}

const readParam = (value: unknown) => (typeof value === 'string' ? value : '')
const documentType = computed<OrganizeDocumentType | ''>(() => {
  const value = readParam(route.params.documentType)
  return value === 'memory' || value === 'output' || value === 'sprout' ? value : ''
})
const documentId = computed(() => readParam(route.params.id))
const activeDocumentId = computed(() => savedDocumentId.value || documentId.value)
const isCreate = computed(() => !savedDocumentId.value && documentId.value === 'new')

const memoryAssetLabel = computed(() => {
  if (memoryKind.value === 'audio') return '录音'
  if (memoryKind.value === 'audio_card') return '工牌'
  return '笔记'
})

const isAudioMemory = computed(() => documentType.value === 'memory' && memoryKind.value === 'audio')
const audioDurationLabel = computed(() => formatDuration(memoryDurationSeconds.value || 0))
const editorPlaceholder = computed(() => {
  if (isAudioMemory.value) return '录音转写内容'
  if (isNoteMemory.value) return '输入正文'
  return '输入内容，或按“/”启用命令'
})

const typeLabel = computed(() => {
  if (documentType.value === 'output') return '发现文档'
  if (documentType.value === 'sprout') return '经营复盘'
  return memoryAssetLabel.value
})

const breadcrumbRootLabel = computed(() => {
  if (documentType.value === 'output') return '发现'
  if (documentType.value === 'sprout') return '经营复盘'
  return '记忆'
})

const breadcrumbItems = computed(() => {
  const root = breadcrumbRootLabel.value
  const section = typeLabel.value
  return section && section !== root ? [root, section] : [root]
})

const isMemoryDocument = computed(() => documentType.value === 'memory')
const isNoteMemory = computed(() => isMemoryDocument.value && (memoryKind.value === 'note' || memoryKind.value === 'record'))
const editorVersion = computed(() => (isNoteMemory.value ? 'advanced' : 'basic'))

const defaultReturnTo = computed(() => {
  if (documentType.value === 'output') return '/platform/organize/output'
  if (documentType.value === 'sprout') return '/platform/organize/sprout'
  if (memoryKind.value === 'audio') return '/platform/organize/memory/audio'
  if (memoryKind.value === 'audio_card') return '/platform/organize/memory/audio-cards'
  return '/platform/organize/memory/notes'
})

const saveStateLabel = computed(() => {
  if (loading.value || !editorReady.value) return '加载中'
  if (saving.value || saveState.value === 'saving') return '正在保存'
  if (saveState.value === 'error') return saveError.value || '保存失败'
  if (saveState.value === 'waiting') return '等待内容'
  if (saveState.value === 'saved') return '已保存'
  return isCreate.value ? '输入后自动保存' : '有未保存修改'
})

const returnTo = computed(() => {
  const candidate = readParam(route.query.return)
  return candidate.startsWith('/platform/organize') ? candidate : defaultReturnTo.value
})

const returnLabel = computed(() => {
  if (returnTo.value.includes('/output')) return '发现'
  if (returnTo.value.includes('/sprout')) return '经营复盘'
  return '记忆'
})

const escapeHtml = (value: string) => {
  const node = document.createElement('div')
  node.textContent = value
  return node.innerHTML
}

const normalizeTitle = (value = '') => value.trim().slice(0, 512)

const asTrimmedString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

const formatDuration = (seconds?: number) => {
  const safeSeconds = Math.max(0, seconds || 0)
  const minutes = Math.floor(safeSeconds / 60)
  return `${String(minutes).padStart(2, '0')}:${String(safeSeconds % 60).padStart(2, '0')}`
}

const formatDateLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚'
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  if (diff < 60 * 60 * 1000) return '刚刚'
  if (diff < 24 * 60 * 60 * 1000) return '今天'
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

const formatTimeLabel = (value: string) => {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '--:--'
  return new Intl.DateTimeFormat('zh-CN', { hour: '2-digit', minute: '2-digit', hour12: false }).format(date)
}

const formatUpdatedLabel = (value: string) => `${formatDateLabel(value)} ${formatTimeLabel(value)}`

const parseHtmlBody = (html = '') => new DOMParser().parseFromString(html, 'text/html').body

const plainTextFromHtml = (html = '') => parseHtmlBody(html).textContent?.replace(/\s+/g, ' ').trim() || ''

const extractTitleFromContent = (html = '') => {
  const firstBlock = Array.from(parseHtmlBody(html).children)[0]
  if (firstBlock?.tagName.toLowerCase() !== 'h1') return ''
  return normalizeTitle(firstBlock.textContent || '')
}

const stripLeadingMemoryTitle = (html = '') => {
  const body = parseHtmlBody(html)
  const firstBlock = Array.from(body.children)[0]

  if (firstBlock?.tagName.toLowerCase() === 'h1') firstBlock.remove()

  const bodyHtml = body.innerHTML.trim()
  return bodyHtml || '<p></p>'
}

const normalizeDocumentContent = (documentTitle = '', html = '') => {
  const normalizedTitle = normalizeTitle(documentTitle)
  const body = parseHtmlBody(html)
  const firstBlock = Array.from(body.children)[0]

  if (firstBlock?.tagName.toLowerCase() === 'h1') {
    const headingTitle = normalizeTitle(firstBlock.textContent || '')
    if (normalizedTitle && headingTitle && headingTitle !== normalizedTitle) {
      return `<h1>${escapeHtml(normalizedTitle)}</h1>${body.innerHTML.trim() || '<p></p>'}`
    }
    if (!headingTitle && normalizedTitle) {
      firstBlock.textContent = normalizedTitle
    }
    return body.innerHTML.trim() || emptyDocumentContent
  }

  if (
    firstBlock?.tagName.toLowerCase() === 'p' &&
    normalizedTitle &&
    firstBlock.textContent?.trim() === normalizedTitle
  ) {
    const heading = body.ownerDocument.createElement('h1')
    heading.textContent = normalizedTitle
    body.replaceChild(heading, firstBlock)
    return body.innerHTML.trim() || `<h1>${escapeHtml(normalizedTitle)}</h1><p></p>`
  }

  const bodyHtml = body.innerHTML.trim()
  return `<h1>${escapeHtml(normalizedTitle)}</h1>${bodyHtml || '<p></p>'}`
}

const normalizeAudioMemoryContent = (html = '') => {
  const body = parseHtmlBody(html)
  const firstBlock = Array.from(body.children)[0]

  if (firstBlock?.tagName.toLowerCase() === 'h1') {
    firstBlock.remove()
  }

  if (body.children.length > 0) {
    return body.innerHTML.trim() || '<p></p>'
  }

  const text = body.textContent?.trim() || ''
  return text ? `<p>${escapeHtml(text)}</p>` : '<p></p>'
}

const memoryBodyContent = (documentTitle = '', html = '') => {
  if (isAudioMemory.value) return normalizeAudioMemoryContent(html)
  return stripLeadingMemoryTitle(html)
}

const noteTagsFromMetadata = (metadata?: Record<string, unknown>) => normalizeNoteTags(metadata?.tags)

const noteMetadataForSave = () => mergeNoteMetadata(memoryMetadata.value, memoryTags.value)

const noteTagIcon = (tag: string) => {
  const normalized = tag.trim().toLowerCase()
  if (normalized.includes('录音') || normalized.includes('音频') || normalized.includes('audio')) return 'microphone'
  return ''
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

const sproutReferenceLabels = (refs: OrganizeMemoryReference[] = [], memoryCount = 0) => {
  if (!refs.length) return memoryCount > 0 ? [`@${memoryCount}条记忆`] : []
  return refs.slice(0, 2).map(memoryReferenceLabel)
}

const sproutSourceLabels = (refs: OrganizeMemoryReference[] = []) => {
  const labels = new Set<string>()
  for (const ref of refs) {
    const label = memoryReferenceSourceLabel(ref.source)
    if (label) labels.add(label)
  }
  return Array.from(labels).slice(0, 2)
}

const fileBaseName = (value = '') => {
  const cleanValue = value.split(/[?#]/)[0].replace(/\\/g, '/')
  const segments = cleanValue.split('/').filter(Boolean)
  const baseName = segments[segments.length - 1] || cleanValue
  try {
    return decodeURIComponent(baseName)
  } catch {
    return baseName
  }
}

const fileExtension = (value = '') => {
  const baseName = fileBaseName(value)
  const dotIndex = baseName.lastIndexOf('.')
  return dotIndex >= 0 ? baseName.slice(dotIndex + 1).toLowerCase() : ''
}

const fileTypeFromMime = (value = '') => {
  const normalized = value.toLowerCase()
  if (normalized.includes('pdf')) return 'pdf'
  if (normalized.includes('wordprocessingml') || normalized.includes('msword')) return 'docx'
  if (normalized.includes('presentationml') || normalized.includes('powerpoint')) return 'pptx'
  if (normalized.includes('spreadsheetml') || normalized.includes('excel')) return 'xlsx'
  if (normalized.includes('markdown')) return 'md'
  if (normalized.startsWith('text/')) return 'txt'
  if (normalized.startsWith('image/')) return normalized.split('/')[1] || 'image'
  if (normalized.startsWith('audio/')) return normalized.split('/')[1] || 'audio'
  if (normalized.startsWith('video/')) return normalized.split('/')[1] || 'video'
  return ''
}

const sourceFilePath = computed(() => {
  const metadata = memoryMetadata.value || {}
  return (
    asTrimmedString(metadata.file_path) ||
    asTrimmedString(metadata.filePath) ||
    asTrimmedString(metadata.source_path) ||
    asTrimmedString(metadata.sourcePath) ||
    asTrimmedString(metadata.file_url) ||
    asTrimmedString(metadata.fileUrl) ||
    asTrimmedString(metadata.source_url) ||
    asTrimmedString(metadata.sourceUrl) ||
    asTrimmedString(metadata.preview_url) ||
    asTrimmedString(metadata.previewUrl)
  )
})

const sourceFileName = computed(() => {
  const metadata = memoryMetadata.value || {}
  return (
    asTrimmedString(metadata.file_name) ||
    asTrimmedString(metadata.fileName) ||
    asTrimmedString(metadata.original_name) ||
    asTrimmedString(metadata.originalName) ||
    asTrimmedString(metadata.filename) ||
    fileBaseName(sourceFilePath.value) ||
    title.value ||
    '源文件'
  )
})

const sourceFilePreviewType = computed(() => {
  const metadata = memoryMetadata.value || {}
  return (
    asTrimmedString(metadata.file_type).toLowerCase() ||
    asTrimmedString(metadata.fileType).toLowerCase() ||
    fileExtension(sourceFileName.value) ||
    fileExtension(sourceFilePath.value) ||
    fileTypeFromMime(asTrimmedString(metadata.mime_type) || asTrimmedString(metadata.mimeType)) ||
    'bin'
  )
})

const sourceFileKindLabel = computed(() => {
  const metadata = memoryMetadata.value || {}
  return (
    asTrimmedString(metadata.content_kind_label) ||
    asTrimmedString(metadata.contentKindLabel) ||
    asTrimmedString(metadata.mime_type) ||
    sourceFilePreviewType.value.toUpperCase()
  )
})

const sourceFileIcon = computed(() => {
  const fileType = sourceFilePreviewType.value
  if (fileType === 'pdf') return 'file-pdf'
  if (['doc', 'docx'].includes(fileType)) return 'file-word'
  if (['xls', 'xlsx', 'csv'].includes(fileType)) return 'file-excel'
  if (['ppt', 'pptx'].includes(fileType)) return 'file-powerpoint'
  if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'tiff', 'svg'].includes(fileType)) return 'image'
  if (['mp3', 'wav', 'm4a', 'flac', 'ogg'].includes(fileType)) return 'sound'
  if (['mp4', 'mov', 'webm', 'avi', 'mkv', 'wmv', 'flv'].includes(fileType)) return 'play-circle'
  return 'file'
})

const sourceFilePreviewUrl = computed(() => {
  const source = sourceFilePath.value
  if (!source) return ''
  if (/^https?:\/\//i.test(source) || source.startsWith('blob:') || source.startsWith('data:')) return source
  return `/files?${new URLSearchParams({ file_path: source }).toString()}`
})

const sourceFileCardVisible = computed(() => isNoteMemory.value && Boolean(sourceFilePath.value))

const openSourceFilePreview = () => {
  if (!sourceFilePreviewUrl.value) {
    MessagePlugin.warning('暂无源文件')
    return
  }
  sourcePreviewVisible.value = true
}

const currentNoteMemoryForService = computed<OrganizeMemory | null>(() => {
  if (!isNoteMemory.value) return null
  const memoryID = activeDocumentId.value
  if (!memoryID || memoryID === 'new') return null
  return {
    id: memoryID,
    kind: memoryKind.value,
    title: title.value,
    content: content.value,
    source: memorySource.value,
    occurred_at: memoryOccurredAt.value || memoryUpdatedAt.value || memoryCreatedAt.value,
    duration_seconds: memoryDurationSeconds.value,
    metadata: memoryMetadata.value,
    created_at: memoryCreatedAt.value,
    updated_at: memoryUpdatedAt.value || memoryOccurredAt.value || memoryCreatedAt.value,
  }
})

const noteServiceTask = computed<ServiceTask | null>(() => {
  const memory = currentNoteMemoryForService.value
  return memory ? buildServiceTaskFromMemory(memory) : null
})

const noteServiceSourceMemory = computed(() => {
  const memory = currentNoteMemoryForService.value
  if (!memory) return null
  return {
    title: memory.title || '未命名记忆',
    sourceLabel: memory.source || '个人记忆',
    dateLabel: formatMemoryDateLabel(memory.occurred_at || memory.updated_at || memory.created_at),
    idLabel: memory.id ? `#${memory.id.slice(0, 8)}` : '未保存',
  }
})

const noteServiceFacts = computed(() => {
  const task = noteServiceTask.value
  if (!task) return []
  return [
    { label: '阶段', value: task.stage },
    { label: '学员', value: task.studentName && task.studentName !== '待补充' ? task.studentName : '待补充' },
    { label: '风险', value: task.riskLabel },
    { label: '下一步', value: task.nextAction },
    { label: '来源', value: `${task.sourceMemoryCount} 条记忆` },
    { label: '置信度', value: task.confidenceLabel },
  ]
})

const serviceExtractionReasonMessage = (reason?: string) => {
  if (reason === 'profile_not_configured') return '服务提醒还未配置，请联系工程师开启'
  if (reason === 'agent_not_enabled') return '这类服务能力还未开启，请联系工程师配置'
  if (reason === 'memory_not_relevant') return '这条记忆缺少客户身份或服务信号'
  return '未生成服务提醒'
}

const serviceExtractionErrorMessage = (error: any) => {
  if (error?.status === 404) return '这条记忆不存在或无权限访问'
  if (typeof error?.message === 'string' && error.message) return error.message
  return '提取服务失败'
}

const extractCurrentMemoryToService = async () => {
  if (!isNoteMemory.value || noteServiceExtracting.value) return
  if (saving.value) {
    MessagePlugin.info('正在保存笔记，请稍后')
    return
  }

  if (isCreate.value || saveState.value !== 'saved') {
    await saveDocument()
  }

  const memoryID = activeDocumentId.value
  if (!memoryID || memoryID === 'new' || saveState.value === 'waiting' || saveState.value === 'error') {
    MessagePlugin.warning('请先保存笔记')
    return
  }

  noteServiceExtracting.value = true
  try {
    const response = await extractServiceMemory(memoryID)
    const data = response?.data
    if (!response.success || !data) {
      throw new Error(response.message || '提取服务失败')
    }
    if (!data.generated) {
      MessagePlugin.warning(serviceExtractionReasonMessage(data.reason))
      return
    }
    MessagePlugin.success('服务提醒已生成')
  } catch (error: any) {
    MessagePlugin.error(serviceExtractionErrorMessage(error))
  } finally {
    noteServiceExtracting.value = false
  }
}

const markDocumentDirty = () => {
  if (!editorReady.value || loading.value) return
  editRevision += 1
  saveError.value = ''
  saveState.value = 'idle'
  scheduleAutosave()
}

const currentEditor = () => editorRef.value?.getEditor() || null

const memorySproutRoleConfig = () => ({
  role: authStore.currentTenantRole || 'viewer',
  tenant_id: authStore.effectiveTenantId || authStore.currentTenantId || '',
  tenant_name: authStore.currentTenantName || '',
  user_id: authStore.currentUserId || '',
  user_name: authStore.user?.username || authStore.user?.email || '我',
})

const addMemoryTag = (tag = noteTagDraft.value) => {
  const normalized = normalizeNoteTags([tag])[0]
  if (!normalized) return
  memoryTags.value = normalizeNoteTags([...memoryTags.value, normalized])
  noteTagDraft.value = ''
  noteTagMenuVisible.value = false
  markDocumentDirty()
}

const removeMemoryTag = (tag: string) => {
  memoryTags.value = memoryTags.value.filter((item) => item.toLowerCase() !== tag.toLowerCase())
  markDocumentDirty()
}

const generateMemoryTags = () => {
  const editor = currentEditor()
  const editorText = editor?.getText() || plainTextFromHtml(content.value)
  const suggested = buildSmartNoteTags(title.value, editorText, memoryTags.value)
  if (!suggested.length) {
    MessagePlugin.info('暂无可用标签')
    return
  }
  memoryTags.value = normalizeNoteTags([...memoryTags.value, ...suggested])
  markDocumentDirty()
  MessagePlugin.success('已生成标签')
}

const createMemorySprout = async () => {
  if (!isMemoryDocument.value || noteSproutCreating.value) return
  if (!savedDocumentId.value && documentId.value === 'new') {
    await saveDocument()
  }

  const memoryID = activeDocumentId.value
  if (!memoryID || memoryID === 'new') {
    MessagePlugin.warning('请先保存笔记')
    return
  }

  noteSproutCreating.value = true
  try {
    const response = await createOrganizeSproutReportFromMemory({
      memory_id: memoryID,
      role_config: memorySproutRoleConfig(),
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '发芽任务创建失败')
    }
    noteSproutReport.value = response.data
    noteSproutStatus.value = response.data.stage
    noteActiveTab.value = 'sprout'
    MessagePlugin.success('发芽任务已创建')
    scheduleNoteSproutRefresh(memoryID)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发芽任务创建失败')
  } finally {
    noteSproutCreating.value = false
  }
}

const noteSproutStageLabel = (stage: OrganizeSproutStage | '' = '') => {
  const labels: Record<OrganizeSproutStage, string> = {
    organizing: '发芽中',
    expandable: '发芽',
    formed: '已发芽',
  }
  return stage ? labels[stage] : '发芽'
}

const noteSproutHeaderState = computed(() => {
  const stage = noteSproutReport.value?.stage || noteSproutStatus.value
  if (noteSproutCreating.value || stage === 'organizing') return 'organizing'
  if (noteSproutReport.value || stage === 'formed') return 'formed'
  return 'idle'
})

const noteSproutHeaderLabel = computed(() => {
  const state = noteSproutHeaderState.value
  if (state === 'organizing') return '发芽中'
  if (state === 'formed') return '已发芽'
  return '发芽'
})

const noteSproutHeaderAriaLabel = computed(() => {
  const state = noteSproutHeaderState.value
  if (state === 'organizing') return '发芽报告生成中'
  if (state === 'formed') return '查看发芽结果'
  return '生成发芽报告'
})

const noteSproutCard = computed(() => {
  const report = noteSproutReport.value
  if (!report) return null
  const preview = buildSproutReportPreview(report.summary || report.output_hint || report.title, report.title)
  const updatedAt = report.updated_at || report.created_at || ''
  const memoryRefs = report.memory_refs || []
  return {
    id: report.id,
    title: report.title || '未命名发芽',
    intro: preview.intro || report.output_hint || report.title,
    stage: noteSproutStageLabel(report.stage),
    stageKey: report.stage,
    updated: updatedAt ? formatUpdatedLabel(updatedAt) : '刚刚 --:--',
    chips: report.chips || [],
    referenceLabels: sproutReferenceLabels(memoryRefs, report.memory_count || 0),
    sourceLabels: sproutSourceLabels(memoryRefs),
  }
})

const activeNoteSproutPreview = computed(() => {
  const report = noteSproutReport.value
  if (!report) return null
  const updatedAt = report.updated_at || report.created_at || ''
  const contentSource = report.summary || report.output_hint || report.title
  return {
    id: report.id,
    title: report.title || '未命名发芽',
    updated: updatedAt ? formatUpdatedLabel(updatedAt) : '刚刚 --:--',
    memoryCount: report.memory_count || report.memory_ids?.length || 0,
    renderedHtml: sproutReportContentForEditor(contentSource),
  }
})

const loadLinkedMemorySproutReport = async (memoryID: string, options?: { silent?: boolean }) => {
  if (!memoryID || memoryID === 'new') return
  const requestSeq = ++noteSproutRequestSeq
  if (!options?.silent) noteSproutLoading.value = true
  try {
    const response = await listOrganizeSproutReports({ page_size: 10, memory_id: memoryID })
    if (requestSeq !== noteSproutRequestSeq) return
    if (!response.success || !response.data) {
      throw new Error(response.message || '发芽数据加载失败')
    }
    const linkedReport = response.data.items.find((report) => (report.memory_ids || []).includes(memoryID)) || response.data.items[0] || null
    noteSproutReport.value = linkedReport
    noteSproutStatus.value = linkedReport?.stage || ''
  } catch {
    if (!options?.silent) {
      noteSproutReport.value = null
      noteSproutStatus.value = ''
    }
  } finally {
    if (requestSeq === noteSproutRequestSeq && !options?.silent) {
      noteSproutLoading.value = false
    }
  }
}

const scheduleNoteSproutRefresh = (memoryID: string) => {
  const refresh = () => {
    void loadLinkedMemorySproutReport(memoryID, { silent: true })
  }
  window.setTimeout(refresh, 1500)
  window.setTimeout(refresh, 5000)
}

const openNoteSproutReport = () => {
  if (!activeNoteSproutPreview.value) return
  sproutPreviewVisible.value = true
}

const closeNoteSproutPreview = () => {
  sproutPreviewVisible.value = false
}

const openNoteSproutEditor = () => {
  const report = noteSproutReport.value
  if (!report?.id) return
  sproutPreviewVisible.value = false
  void router.push({
    path: `/platform/organize/editor/sprout/${encodeURIComponent(report.id)}`,
    query: { return: route.fullPath },
  })
}

const handleMemorySproutHeaderAction = () => {
  if (noteSproutCreating.value || noteSproutLoading.value) return
  if (noteSproutReport.value) {
    noteActiveTab.value = 'sprout'
    return
  }
  void createMemorySprout()
}

const audioMemoryFallbackTitle = (html = '') => {
  const body = parseHtmlBody(html)
  const text = body.textContent?.replace(/\s+/g, ' ').trim() || ''
  if (!text) return '录音记忆'
  return text.length > 24 ? `${text.slice(0, 24)}...` : text
}

const audioMemoryDisplayTitle = (item: OrganizeMemory) => {
  const metadata = item.metadata || {}
  return (
    asTrimmedString(metadata.title) ||
    asTrimmedString(metadata.extracted_title) ||
    asTrimmedString(metadata.ai_title) ||
    asTrimmedString(metadata.summary_title) ||
    item.title
  )
}

const audioMemoryContentSource = (item: OrganizeMemory) => {
  const metadata = item.metadata || {}
  return item.content ||
    asTrimmedString(metadata.transcript) ||
    asTrimmedString(metadata.transcription) ||
    asTrimmedString(metadata.asr_text)
}

const audioSourcePath = (metadata: Record<string, unknown>) => {
  return (
    asTrimmedString(metadata.audio_url) ||
    asTrimmedString(metadata.audioUrl) ||
    asTrimmedString(metadata.audio_path) ||
    asTrimmedString(metadata.audioPath) ||
    asTrimmedString(metadata.media_url) ||
    asTrimmedString(metadata.mediaUrl) ||
    asTrimmedString(metadata.media_path) ||
    asTrimmedString(metadata.mediaPath) ||
    asTrimmedString(metadata.file_url) ||
    asTrimmedString(metadata.fileUrl) ||
    asTrimmedString(metadata.source_url) ||
    asTrimmedString(metadata.sourceUrl) ||
    asTrimmedString(metadata.source_path) ||
    asTrimmedString(metadata.sourcePath) ||
    asTrimmedString(metadata.preview_url) ||
    asTrimmedString(metadata.previewUrl) ||
    asTrimmedString(metadata.file_path)
  )
}

const audioSourceUrl = computed(() => {
  if (!isAudioMemory.value) return ''
  const metadata = memoryMetadata.value || {}
  const source = audioSourcePath(metadata)
  if (!source) return ''
  if (/^https?:\/\//i.test(source) || source.startsWith('blob:') || source.startsWith('data:')) return source
  return `/files?${new URLSearchParams({ file_path: source }).toString()}`
})

const readQuery = (key: string) => readParam(route.query[key])

const readQueryList = (key: string) => readQuery(key).split(',').map((item) => item.trim()).filter(Boolean)

const readQueryDraft = (): OrganizeEditorDraft => {
  const draft: OrganizeEditorDraft = {
    title: readQuery('title') || undefined,
    content: readQuery('content') || undefined,
  }
  const kind = readQuery('kind')
  const source = readQuery('source')
  const durationSeconds = Number(readQuery('duration_seconds'))
  const outputType = readQuery('output_type')
  const sourceSummary = readQuery('source_summary')
  const status = readQuery('status')
  const icon = readQuery('icon')
  const stage = readQuery('stage')
  const outputHint = readQuery('output_hint')
  const chips = readQueryList('chips')

  if (kind) draft.kind = kind as OrganizeMemoryKind
  if (source) draft.source = source
  if (Number.isFinite(durationSeconds) && durationSeconds > 0) draft.duration_seconds = durationSeconds
  if (outputType) draft.output_type = outputType
  if (sourceSummary) draft.source_summary = sourceSummary
  if (status) draft.status = status as OrganizeOutputStatus
  if (icon) draft.icon = icon
  if (stage) draft.stage = stage as OrganizeSproutStage
  if (outputHint) draft.output_hint = outputHint
  if (chips.length) draft.chips = chips

  return draft
}

const readInitialDraft = () => {
  if (documentType.value && documentId.value) {
    const storedDraft = readOrganizeEditorDraft(documentType.value, documentId.value)
    if (storedDraft) return storedDraft
  }
  return readQueryDraft()
}

const outputFromDraft = (draft: OrganizeEditorDraft): OrganizeOutput | null => {
  const draftTitle = normalizeTitle(draft.title || extractTitleFromContent(draft.content))
  if (documentType.value !== 'output' || !draftTitle) return null
  return {
    id: documentId.value,
    title: draftTitle,
    content: normalizeDocumentContent(draftTitle, draft.content),
    output_type: draft.output_type || '研究文档',
    source_summary: draft.source_summary,
    status: draft.status || 'draft',
    icon: draft.icon || 'file-word',
    memory_ids: draft.memory_ids || [],
    metadata: draft.metadata,
    created_at: '',
    updated_at: '',
  }
}

const sproutFromDraft = (draft: OrganizeEditorDraft): OrganizeSproutReport | null => {
  const draftTitle = normalizeTitle(draft.title || extractTitleFromContent(draft.content))
  if (documentType.value !== 'sprout' || !draftTitle) return null
  const editableSummary = sproutReportContentForEditor(draft.content)
  return {
    id: documentId.value,
    title: draftTitle,
    summary: normalizeDocumentContent(draftTitle, editableSummary),
    stage: draft.stage || 'organizing',
    output_hint: draft.output_hint,
    chips: draft.chips || [],
    memory_count: draft.memory_ids?.length || 0,
    memory_ids: draft.memory_ids || [],
    metadata: draft.metadata,
    created_at: '',
    updated_at: '',
  }
}

const clearAutosaveTimer = () => {
  if (autosaveTimer) {
    clearTimeout(autosaveTimer)
    autosaveTimer = null
  }
}

const resetDraft = () => {
  const draft = readInitialDraft()
  const draftTitle = normalizeTitle(draft.title || extractTitleFromContent(draft.content))
  const draftContent = documentType.value === 'sprout'
    ? sproutReportContentForEditor(draft.content)
    : draft.content
  title.value = draftTitle
  memoryKind.value = draft.kind || 'note'
  memorySource.value = draft.source || '手动输入'
  memoryDurationSeconds.value = draft.duration_seconds || 0
  memoryMetadata.value = draft.metadata
  memoryTags.value = noteTagsFromMetadata(draft.metadata)
  outputCategory.value = normalizeDiscoverCategory(
    draft.metadata?.discover_category || draft.metadata?.discover_category_label,
  )
  memoryOccurredAt.value = ''
  memoryCreatedAt.value = ''
  memoryUpdatedAt.value = ''
  noteTagDraft.value = ''
  noteTagMenuVisible.value = false
  noteSproutReport.value = null
  noteSproutLoading.value = false
  noteSproutStatus.value = ''
  sourcePreviewVisible.value = false
  sproutPreviewVisible.value = false
  noteSproutRequestSeq += 1
  noteActiveTab.value = 'content'
  if (documentType.value === 'memory' && memoryKind.value === 'audio') {
    content.value = normalizeAudioMemoryContent(draftContent)
    title.value = draftTitle || audioMemoryFallbackTitle(content.value)
  } else if (isNoteMemory.value) {
    content.value = memoryBodyContent(draftTitle, draftContent)
  } else {
    content.value = normalizeDocumentContent(draftTitle, draftContent)
  }
  outputDraft.value = outputFromDraft(draft)
  sproutDraft.value = sproutFromDraft(draft)
}

const loadDocument = async () => {
  clearAutosaveTimer()
  editorReady.value = false
  saveError.value = ''
  editRevision += 1

  if (!documentType.value || !documentId.value) {
    loadError.value = '编辑地址无效'
    return
  }

  loadError.value = ''
  resetDraft()
  if (isCreate.value) {
    editorKey.value += 1
    await nextTick()
    editorReady.value = true
    saveState.value = 'idle'
    return
  }

  loading.value = true
  try {
    if (documentType.value === 'memory') {
      const response = await getOrganizeMemory(documentId.value)
      if (!response.success || !response.data) throw new Error(response.message || '笔记加载失败')
      const item = response.data
      memoryKind.value = item.kind
      memorySource.value = item.source || '手动输入'
      memoryDurationSeconds.value = item.duration_seconds || 0
      memoryMetadata.value = item.metadata
      memoryTags.value = noteTagsFromMetadata(item.metadata)
      memoryOccurredAt.value = item.occurred_at || item.created_at || item.updated_at || ''
      memoryCreatedAt.value = item.created_at || ''
      memoryUpdatedAt.value = item.updated_at || ''
      noteTagDraft.value = ''
      noteTagMenuVisible.value = false
      noteSproutReport.value = null
      noteSproutStatus.value = ''
      sourcePreviewVisible.value = false
      noteActiveTab.value = 'content'
      title.value = item.kind === 'audio'
        ? audioMemoryDisplayTitle(item)
        : isNoteMemory.value
          ? normalizeTitle(item.title || extractTitleFromContent(item.content) || plainTextFromHtml(item.content).slice(0, 80))
          : item.title
      content.value = item.kind === 'audio'
        ? normalizeAudioMemoryContent(audioMemoryContentSource(item))
        : isNoteMemory.value
          ? memoryBodyContent(item.title, item.content)
          : normalizeDocumentContent(item.title, item.content)
      if (isNoteMemory.value) {
        await loadLinkedMemorySproutReport(item.id)
      }
    } else if (documentType.value === 'output') {
      const response = await getOrganizeOutput(documentId.value)
      if (!response.success || !response.data) throw new Error(response.message || '发现加载失败')
      const item = response.data
      title.value = item.title
      content.value = normalizeDocumentContent(item.title, item.content)
      outputDraft.value = item
      outputCategory.value = normalizeDiscoverCategory(
        item.metadata?.discover_category || item.metadata?.discover_category_label,
      )
    } else {
      const response = await getOrganizeSproutReport(documentId.value)
      if (!response.success || !response.data) throw new Error(response.message || '发芽加载失败')
      const item = response.data
      title.value = item.title
      content.value = normalizeDocumentContent(item.title, sproutReportContentForEditor(item.summary))
      sproutDraft.value = item
    }
    editorKey.value += 1
    await nextTick()
    editorReady.value = true
    saveState.value = 'saved'
  } catch (error: any) {
    loadError.value = error?.message || '文档加载失败'
  } finally {
    loading.value = false
  }
}

const scheduleAutosave = () => {
  if (!editorReady.value || loading.value) return
  clearAutosaveTimer()
  autosaveTimer = setTimeout(() => {
    autosaveTimer = null
    void saveDocument()
  }, AUTOSAVE_DELAY)
}

const handleEditorKeydown = (event: KeyboardEvent) => {
  if (event.key !== '/' || event.isComposing || event.altKey || event.ctrlKey || event.metaKey) return
  const editor = editorRef.value?.getEditor()
  if (!editor || !editor.isEditable) return

  const { selection } = editor.state
  if (!selection.empty) return

  const { $from } = selection
  const textBeforeCursor = $from.parent.textBetween(0, $from.parentOffset, undefined, '\ufffc')
  if (!/^\s+$/.test(textBeforeCursor)) return

  event.preventDefault()
  editor.chain().focus().deleteRange({ from: $from.start(), to: $from.pos }).insertContent('/').run()
}

const saveDocument = async () => {
  if (!editorReady.value || loading.value) return
  if (saving.value) {
    autosavePending = true
    return
  }

  const currentType = documentType.value
  const currentDocumentId = activeDocumentId.value
  const currentMemoryKind = memoryKind.value
  const savingAudioMemory = currentType === 'memory' && currentMemoryKind === 'audio'
  const html = editorRef.value?.getHTML() || content.value
  const normalizedTitle = savingAudioMemory
    ? normalizeTitle(title.value || audioMemoryFallbackTitle(html))
    : currentType === 'memory' && isNoteMemory.value
      ? normalizeTitle(title.value || extractTitleFromContent(html) || plainTextFromHtml(html).slice(0, 80))
      : extractTitleFromContent(html)
  if (!normalizedTitle) {
    saveState.value = 'waiting'
    return
  }

  const revisionAtSave = editRevision
  const creating = isCreate.value
  saving.value = true
  title.value = normalizedTitle
  saveState.value = 'saving'
  saveError.value = ''

  try {
    let savedId = ''
    if (currentType === 'memory') {
      const memoryContent = savingAudioMemory ? normalizeAudioMemoryContent(html) : html
      const input = {
        kind: currentMemoryKind,
        title: normalizedTitle,
        content: memoryContent,
        source: memorySource.value,
        duration_seconds: memoryDurationSeconds.value,
        metadata: noteMetadataForSave(),
      }
      const response = creating
        ? await createOrganizeMemory(input)
        : await updateOrganizeMemory(currentDocumentId, input)
      if (!response.success || !response.data) throw new Error(response.message || '笔记保存失败')
      const savedMemory = response.data
      savedId = savedMemory.id
      memoryKind.value = savedMemory.kind
      memorySource.value = savedMemory.source || memorySource.value
      memoryDurationSeconds.value = savedMemory.duration_seconds || 0
      memoryMetadata.value = savedMemory.metadata
      memoryTags.value = noteTagsFromMetadata(savedMemory.metadata)
      memoryOccurredAt.value = savedMemory.occurred_at || ''
      memoryCreatedAt.value = savedMemory.created_at || ''
      memoryUpdatedAt.value = savedMemory.updated_at || ''
    } else if (currentType === 'output') {
      const draft = outputDraft.value
      if (!outputCategory.value) {
        saveState.value = 'waiting'
        saveError.value = '请选择发现栏目'
        return
      }
      const input = {
        title: normalizedTitle,
        content: html,
        output_type: draft?.output_type || readQuery('output_type') || '研究文档',
        source_summary: draft?.source_summary || readQuery('source_summary') || '手动创建',
        status: draft?.status || (readQuery('status') as OrganizeOutputStatus) || 'draft',
        icon: draft?.icon || readQuery('icon') || 'file-word',
        memory_ids: draft?.memory_ids || [],
        metadata: {
          ...draft?.metadata,
          discover_category: outputCategory.value,
          discover_category_label: discoverCategoryLabel(outputCategory.value),
        },
      }
      const response = creating
        ? await createOrganizeOutput(input)
        : await updateOrganizeOutput(currentDocumentId, input)
      if (!response.success || !response.data) throw new Error(response.message || '发现保存失败')
      savedId = response.data.id
      outputDraft.value = response.data
    } else if (currentType === 'sprout') {
      const draft = sproutDraft.value
      const input = {
        title: normalizedTitle,
        summary: html,
        stage: draft?.stage || (readQuery('stage') as OrganizeSproutStage) || 'organizing',
        output_hint: draft?.output_hint || readQuery('output_hint') || '可继续整理',
        chips: draft?.chips || readQueryList('chips'),
        memory_ids: draft?.memory_ids || [],
        metadata: draft?.metadata,
      }
      const response = creating
        ? await createOrganizeSproutReport(input)
        : await updateOrganizeSproutReport(currentDocumentId, input)
      if (!response.success || !response.data) throw new Error(response.message || '发芽保存失败')
      savedId = response.data.id
      sproutDraft.value = response.data
    }

    if (creating && savedId && currentType) {
      const draftDocumentId = documentId.value
      savedDocumentId.value = savedId
      clearOrganizeEditorDraft(currentType, draftDocumentId)
      skipNextRouteLoad = true
      await router.replace({
        path: `/platform/organize/editor/${currentType}/${encodeURIComponent(savedId)}`,
        query: {},
      })
    }

    if (editRevision === revisionAtSave) {
      saveState.value = 'saved'
    } else {
      scheduleAutosave()
    }
  } catch (error: any) {
    saveError.value = error?.message || '文档保存失败'
    saveState.value = 'error'
    MessagePlugin.error(saveError.value)
  } finally {
    saving.value = false
    if (autosavePending) {
      autosavePending = false
      scheduleAutosave()
    }
  }
}

const goBack = async () => {
  clearAutosaveTimer()
  if (editorReady.value && !saving.value && saveState.value !== 'saved') {
    await saveDocument()
  }
  await router.push(returnTo.value)
}

const syncTitleFromContent = () => {
  if (isMemoryDocument.value) return
  const nextTitle = extractTitleFromContent(editorRef.value?.getHTML() || content.value)
  if (nextTitle !== title.value) {
    title.value = nextTitle
  }
}

watch(content, () => {
  if (!editorReady.value || loading.value) return
  syncTitleFromContent()
  editRevision += 1
  saveError.value = ''
  saveState.value = 'idle'
  scheduleAutosave()
})

watch(title, () => {
  if (!editorReady.value || loading.value || !isMemoryDocument.value) return
  editRevision += 1
  saveError.value = ''
  saveState.value = 'idle'
  scheduleAutosave()
})

watch(
  () => `${route.params.documentType}:${route.params.id}`,
  () => {
    if (skipNextRouteLoad) {
      skipNextRouteLoad = false
      return
    }
    savedDocumentId.value = ''
    void loadDocument()
  },
  { immediate: true },
)
</script>

<style scoped lang="less">
.organize-editor-page {
  --document-content-max-width: 860px;
  --document-content-font: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI Variable Display", "Segoe UI", Helvetica, Arial, sans-serif;

  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: #fff;
  color: #37352f;
}

.organize-editor-header {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  min-height: 52px;
  padding: 8px 24px;
  border-bottom: 1px solid rgba(55, 53, 47, 0.09);
  background: rgba(255, 255, 255, 0.94);
  box-sizing: border-box;
}

.editor-header-left,
.editor-page-actions,
.editor-breadcrumbs {
  display: flex;
  align-items: center;
}

.editor-header-left {
  min-width: 0;
  gap: 12px;
}

.editor-back-button {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  flex: 0 0 auto;
  height: 32px;
  padding: 0 7px 0 3px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: rgba(55, 53, 47, 0.62);
  font: inherit;
  font-size: 13px;
  cursor: pointer;
}

.editor-back-button:hover {
  background: rgba(55, 53, 47, 0.08);
  color: #37352f;
}

.editor-breadcrumbs {
  min-width: 0;
  gap: 7px;
  overflow: hidden;
  color: rgba(55, 53, 47, 0.5);
  font-size: 13px;
  line-height: 20px;
  white-space: nowrap;
}

.editor-breadcrumbs :deep(.t-icon) {
  flex: 0 0 auto;
  color: rgba(55, 53, 47, 0.28);
  font-size: 12px;
}

.editor-breadcrumb-current {
  min-width: 0;
  overflow: hidden;
  color: rgba(55, 53, 47, 0.72);
  text-overflow: ellipsis;
}

.editor-page-actions {
  justify-content: flex-end;
  gap: 8px;
  flex: 0 0 auto;
}

.editor-service-action {
  height: 36px;
  padding: 0 14px;
  border-color: rgba(55, 53, 47, 0.14);
  border-radius: 8px;
  background: #fff;
  color: #20242a;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
}

.editor-service-action:hover {
  border-color: rgba(15, 118, 110, 0.28);
  background: rgba(15, 118, 110, 0.05);
  color: #20242a;
}

.editor-service-action :deep(.t-button__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
}

.editor-service-action-icon {
  display: block;
  width: 16px;
  height: 16px;
}

.editor-sprout-action {
  height: 36px;
  padding: 0 14px;
  border-color: rgba(55, 53, 47, 0.14);
  border-radius: 8px;
  background: #fff;
  color: #20242a;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
}

.editor-sprout-action:hover {
  border-color: rgba(34, 101, 73, 0.32);
  background: rgba(34, 101, 73, 0.04);
  color: #20242a;
}

.editor-sprout-action :deep(.t-button__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
}

.editor-sprout-action-icon {
  font-size: 18px;
  color: #22c55e;
}

.editor-sprout-action--organizing .editor-sprout-action-icon {
  color: #2eaadc;
}

.editor-sprout-action--formed {
  border-color: rgba(34, 101, 73, 0.24);
  background: rgba(34, 101, 73, 0.06);
  color: #236549;
}

.editor-sprout-action--formed:hover {
  border-color: rgba(34, 101, 73, 0.36);
  background: rgba(34, 101, 73, 0.1);
  color: #236549;
}

.editor-save-state {
  margin-right: 4px;
  color: rgba(55, 53, 47, 0.45);
  font-size: 12px;
  white-space: nowrap;
}

.editor-save-state--saving {
  color: #2eaadc;
}

.editor-save-state--saved {
  color: #4a8f5c;
}

.editor-save-state--error {
  color: var(--td-error-color);
}

.editor-save-state--waiting {
  color: rgba(55, 53, 47, 0.45);
}

.organize-editor-main {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 0 32px 80px;
  box-sizing: border-box;
}

.editor-page-content {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  min-height: 0;
}

.document-page {
  width: min(var(--document-content-max-width), 100%);
  min-width: 0;
  margin: 0 auto;
  padding-top: 0;
  box-sizing: border-box;
}

.output-category-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.output-category-copy {
  min-width: 0;
}

.output-category-eyebrow {
  display: block;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.output-category-copy strong {
  display: block;
  margin-top: 2px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 22px;
}

.output-category-panel :deep(.t-select) {
  flex: 0 0 260px;
}

.memory-audio-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 14px;
}

.memory-audio-title-input {
  width: 100%;
}

.memory-audio-title-input :deep(.t-input) {
  height: 42px;
  border-color: transparent;
  background: transparent;
  padding-inline: 0;
  box-shadow: none;
}

.memory-audio-title-input :deep(.t-input__inner) {
  color: #37352f;
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.memory-audio-player {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr) 58px;
  align-items: center;
  gap: 10px;
  min-height: 56px;
  padding: 10px 12px;
  border: 1px solid rgba(55, 53, 47, 0.06);
  border-radius: 12px;
  background: #f7f8fb;
  box-sizing: border-box;
}

.memory-audio-play {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border: 0;
  border-radius: 50%;
  background: #eef1f6;
  color: #464c57;
  cursor: pointer;
}

.memory-audio-native {
  width: 100%;
  min-width: 0;
  grid-column: 1 / 3;
  height: 32px;
}

.memory-audio-track-wrap {
  display: flex;
  flex-direction: column;
  gap: 7px;
  min-width: 0;
}

.memory-audio-track {
  position: relative;
  height: 4px;
  border-radius: 999px;
  background: #dde2eb;
}

.memory-audio-thumb {
  position: absolute;
  top: 50%;
  left: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #20242b;
  transform: translate(-2px, -50%);
}

.memory-audio-progress {
  position: absolute;
  inset: 0 auto 0 0;
  width: 18%;
  border-radius: 999px;
  background: #6e7684;
}

.memory-audio-time-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: rgba(55, 53, 47, 0.62);
  font-size: 12px;
  line-height: 16px;
}

.memory-audio-transcript-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 34px;
  border-left: 1px solid rgba(55, 53, 47, 0.08);
  color: rgba(55, 53, 47, 0.58);
  font-size: 12px;
  line-height: 18px;
}

.memory-note-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 6px;
}

.memory-note-header {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.memory-note-title-input {
  width: 100%;
}

.memory-note-title-input :deep(.t-input) {
  min-height: 58px;
  border-color: transparent;
  background: transparent;
  padding-inline: 0;
  box-shadow: none;
}

.memory-note-title-input :deep(.t-input__inner) {
  color: #37352f;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.2;
}

.memory-note-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
}

.memory-note-tag-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.memory-note-tag-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-start;
  gap: 10px;
}

.memory-note-tag {
  display: inline-flex;
  align-items: center;
  gap: 0;
  max-width: 180px;
  min-height: 30px;
  padding: 0 12px;
  border: 1px solid #dfe4ec;
  border-radius: 999px;
  background: #fbfcff;
  color: #667085;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
  box-sizing: border-box;
}

.memory-note-tag:hover {
  border-color: #d3d9e5;
  background: #fff;
  color: #5d687b;
}

.memory-note-tag-leading-icon {
  flex: 0 0 auto;
  margin-right: 6px;
  color: #667085;
}

.memory-note-tag-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.memory-note-tag :deep(.t-tag__text) {
  display: inline-flex;
  align-items: center;
  min-width: 0;
}

.memory-note-tag-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 0;
  height: 14px;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: #98a2b3;
  cursor: pointer;
  opacity: 0;
  overflow: hidden;
  pointer-events: none;
  transition:
    width 0.16s ease,
    margin-left 0.16s ease,
    opacity 0.16s ease,
    background 0.16s ease;
}

.memory-note-tag:hover .memory-note-tag-remove,
.memory-note-tag:focus-within .memory-note-tag-remove {
  width: 14px;
  margin-left: 6px;
  opacity: 1;
  pointer-events: auto;
}

.memory-note-tag-remove:hover {
  background: rgba(102, 112, 133, 0.12);
  color: #667085;
}

.memory-note-tag-action {
  min-width: 0;
  height: 30px;
  padding: 0 12px;
  border-color: #dfe4ec;
  border-radius: 999px;
  background: #fff;
  color: #667085;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.memory-note-tag-action:hover {
  border-color: #d3d9e5;
  background: #fbfcff;
  color: #5d687b;
}

.memory-note-tag-action :deep(.t-button__icon) {
  display: inline-flex;
  align-items: center;
  margin-right: 5px;
  color: inherit;
  font-size: 14px;
}

.memory-note-tag-action :deep(.t-button__text) {
  color: inherit;
  font-size: 13px;
  font-weight: 500;
  line-height: 18px;
}

.memory-note-tag-smart-button {
  border-color: #dbe5ff;
  background: #eef4ff;
  color: #5264d6;
}

.memory-note-tag-smart-button:hover {
  border-color: #cfdbff;
  background: #e8f0ff;
  color: #4859c7;
}

.memory-note-tag-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: 280px;
  padding: 14px;
  box-sizing: border-box;
}

.memory-note-tag-panel-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

:global(.memory-note-tag-popup .t-popup__content) {
  padding: 0;
  border-radius: 12px;
  box-shadow: 0 16px 40px rgba(15, 15, 15, 0.14);
  overflow: hidden;
}

.memory-note-source-card {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) auto 18px;
  align-items: center;
  gap: 12px;
  width: min(520px, 100%);
  min-height: 62px;
  margin: 0;
  padding: 10px 12px;
  border: 1px solid rgba(55, 53, 47, 0.12);
  border-radius: 8px;
  background: #fff;
  color: #37352f;
  text-align: left;
  cursor: pointer;
  box-shadow: 0 1px 2px rgba(15, 15, 15, 0.04);
  transition:
    border-color 0.16s ease,
    box-shadow 0.16s ease,
    transform 0.16s ease,
    background 0.16s ease;
}

.memory-note-source-card:hover,
.memory-note-source-card:focus-visible {
  border-color: rgba(55, 53, 47, 0.2);
  background: #fbfbfa;
  box-shadow: 0 6px 18px rgba(15, 15, 15, 0.08);
  transform: translateY(-1px);
  outline: none;
}

.memory-note-source-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: rgba(47, 179, 95, 0.1);
  color: #2fb35f;
}

.memory-note-source-main {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.memory-note-source-label {
  color: rgba(55, 53, 47, 0.52);
  font-size: 12px;
  line-height: 16px;
}

.memory-note-source-name {
  overflow: hidden;
  color: #37352f;
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.memory-note-source-meta {
  min-width: 0;
  max-width: 140px;
  overflow: hidden;
  color: rgba(55, 53, 47, 0.48);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.memory-note-source-arrow {
  color: rgba(55, 53, 47, 0.48);
}

:deep(.memory-source-preview-drawer .t-drawer__body) {
  padding: 0;
  background: #f7f8fa;
}

.memory-source-preview-body {
  height: calc(100vh - 56px);
  padding: 16px;
  box-sizing: border-box;
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

.memory-note-tabs {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 24px;
  border-bottom: 1px solid rgba(55, 53, 47, 0.1);
}

.memory-note-tab {
  position: relative;
  margin: 0;
  padding: 0 0 14px;
  border: 0;
  background: transparent;
  color: rgba(55, 53, 47, 0.5);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.2;
  cursor: pointer;
}

.memory-note-tab.is-active {
  color: #37352f;
}

.memory-note-tab.is-active::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 3px;
  border-radius: 999px;
  background: #37352f;
  content: '';
}

.memory-note-tab-panels {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 8px;
}

.memory-note-panel-view {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.memory-note-sprout-loading {
  display: flex;
  align-items: center;
  min-height: 124px;
  padding-top: 4px;
}

.memory-note-sprout-card {
  cursor: pointer;
}

.memory-note-service-card {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 20px;
  border: 1px solid rgba(55, 73, 65, 0.14);
  border-radius: 8px;
  background: #fbfcfb;
}

.memory-note-service-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
}

.memory-note-service-head-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 10px;

  > span {
    align-self: flex-start;
    padding: 4px 9px;
    border-radius: 999px;
    background: rgba(22, 93, 67, 0.08);
    color: #165d43;
    font-size: 12px;
    font-weight: 700;
    line-height: 1.4;
    white-space: nowrap;
  }

  h2 {
    margin: 0 0 6px;
    color: #26241f;
    font-size: 22px;
    font-weight: 700;
    line-height: 1.25;
  }

  em {
    color: rgba(55, 53, 47, 0.54);
    font-size: 13px;
    font-style: normal;
    line-height: 1.5;

    &.priority-high {
      color: #b7352c;
    }

    &.priority-medium {
      color: #8a5a10;
    }

    &.priority-low {
      color: #46605b;
    }
  }
}

.memory-note-service-summary {
  margin: 0;
  color: rgba(55, 53, 47, 0.78);
  font-size: 15px;
  line-height: 1.85;
}

.memory-note-service-facts {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin: 0;

  > div {
    min-width: 0;
    padding: 10px 12px;
    border: 1px solid rgba(55, 73, 65, 0.1);
    border-radius: 8px;
    background: #ffffff;
  }

  dt {
    margin: 0 0 4px;
    color: rgba(55, 53, 47, 0.45);
    font-size: 12px;
    line-height: 1.4;
  }

  dd {
    margin: 0;
    overflow: hidden;
    color: #37352f;
    font-size: 13px;
    font-weight: 600;
    line-height: 1.5;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.memory-note-service-section {
  display: flex;
  flex-direction: column;
  gap: 10px;

  h3 {
    margin: 0;
    color: #26241f;
    font-size: 15px;
    font-weight: 700;
    line-height: 1.4;
  }

  p {
    margin: 0;
    color: rgba(55, 53, 47, 0.76);
    font-size: 14px;
    line-height: 1.8;
  }

  ul {
    margin: 0;
    padding-left: 18px;
    color: rgba(55, 53, 47, 0.78);
    font-size: 14px;
    line-height: 1.8;
  }

  blockquote {
    margin: 0;
    padding: 10px 12px;
    border-left: 3px solid rgba(22, 93, 67, 0.22);
    border-radius: 0 8px 8px 0;
    background: rgba(22, 93, 67, 0.06);
    color: rgba(55, 53, 47, 0.82);
    font-size: 14px;
    line-height: 1.8;
  }
}

.memory-note-service-source-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.memory-note-service-source {
  display: grid;
  grid-template-columns: 34px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  min-width: 0;
  padding: 10px 12px;
  border: 1px solid rgba(55, 73, 65, 0.12);
  border-radius: 8px;
  background: #ffffff;
  text-align: left;
  cursor: pointer;

  &:hover {
    border-color: rgba(22, 93, 67, 0.28);
    background: #f8fbfa;
  }

  strong,
  em {
    display: block;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: #26241f;
    font-size: 13px;
    line-height: 1.45;
  }

  em {
    margin-top: 2px;
    color: rgba(55, 53, 47, 0.5);
    font-size: 12px;
    font-style: normal;
    line-height: 1.45;
  }
}

.memory-note-service-source-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: rgba(22, 93, 67, 0.08);
  color: #165d43;
}

.memory-note-service-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 180px;
  gap: 8px;
  border: 1px dashed rgba(55, 53, 47, 0.18);
  border-radius: 8px;
  background: #fcfcfb;
  color: rgba(55, 53, 47, 0.42);
  text-align: center;

  strong {
    color: #37352f;
    font-size: 15px;
    line-height: 1.5;
  }

  p {
    margin: 0;
    color: rgba(55, 53, 47, 0.52);
    font-size: 13px;
    line-height: 1.6;
  }
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

  &:hover,
  &:focus {
    border-color: rgba(34, 101, 73, 0.42);
    box-shadow: 0 10px 26px rgba(38, 34, 29, 0.1);
    outline: none;
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

.sprout-report-main .report-chips {
  margin-top: auto;
  margin-bottom: 6px;
}

.report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 14px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.sprout-report-meta {
  padding-top: 6px;
}

.sprout-report-meta-separator {
  color: var(--td-text-color-placeholder);
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

.document-editor-shell {
  position: relative;
  width: 100%;
  min-height: 560px;
  overflow: visible;
  background: transparent;
  --tiptap-primary: #2eaadc;
  --tiptap-primary-light: rgba(46, 170, 220, 0.12);
  --tiptap-border: rgba(55, 53, 47, 0.09);
  --tiptap-bg: #fff;
  --tiptap-bg-secondary: #fff;
  --tiptap-bg-hover: rgba(55, 53, 47, 0.08);
  --tiptap-text: #37352f;
  --tiptap-text-secondary: #787774;
  --tiptap-text-muted: #9b9a97;
  --tiptap-link: rgb(35, 131, 226);
  --editor-empty-placeholder: "输入内容，或按“/”启用命令";
  --editor-title-placeholder: "无标题";
}

.document-editor-shell--memory-note {
  min-height: 620px;
  --editor-empty-placeholder: "输入正文";
  --editor-title-placeholder: "标题";
}

.document-editor-shell :deep(.tiptap-pro-editor),
.document-editor-shell :deep(.word-document-container),
.document-editor-shell :deep(.document-pages),
.document-editor-shell :deep(.continuous-pages) {
  width: 100%;
  background: transparent;
}

.document-editor-shell :deep(.word-document-container) {
  padding: 0;
}

.document-editor-shell--memory-note :deep(.tiptap-pro-editor.word-mode) {
  height: auto;
  min-height: inherit;
  overflow: visible;
}

.document-editor-shell--memory-note :deep(.word-document-container) {
  flex: 0 1 auto;
  min-height: 0;
  overflow: visible;
  overscroll-behavior-y: auto;
}

.document-editor-shell--memory-note :deep(.document-pages) {
  flex: 0 0 auto;
}

.document-editor-shell :deep(.continuous-pages) {
  max-width: none;
  min-height: 100%;
  margin: 0;
  padding: 0;
  box-shadow: none;
  border-radius: 0;
}

.document-editor-shell :deep(.word-content-multi .ProseMirror) {
  min-height: calc(100vh - 180px);
  padding: 0 0 120px;
  color: #37352f;
  font-family: var(--document-content-font);
  font-size: 16px;
  line-height: 1.5;
}

.document-editor-shell--memory-note :deep(.word-content-multi .ProseMirror) {
  min-height: calc(100vh - 340px);
  font-size: 15px;
  line-height: 1.55;
}

.document-editor-shell--memory-note :deep(.ProseMirror p) {
  min-height: 28px;
  padding: 2px 0;
}

.document-editor-shell--memory-note :deep(.ProseMirror h1) {
  margin: 20px 0 6px;
  font-size: 26px;
}

.document-editor-shell--memory-note :deep(.ProseMirror h2) {
  margin: 18px 0 6px;
  font-size: 20px;
}

.document-editor-shell--memory-note :deep(.ProseMirror h3) {
  margin: 16px 0 6px;
  font-size: 18px;
}

.document-editor-shell :deep(.ProseMirror p),
.document-editor-shell :deep(.ProseMirror h1),
.document-editor-shell :deep(.ProseMirror h2),
.document-editor-shell :deep(.ProseMirror h3),
.document-editor-shell :deep(.ProseMirror blockquote),
.document-editor-shell :deep(.ProseMirror pre),
.document-editor-shell :deep(.ProseMirror ul),
.document-editor-shell :deep(.ProseMirror ol) {
  position: relative;
}

.document-editor-shell :deep(.ProseMirror p) {
  min-height: 30px;
  margin: 0;
  padding: 3px 0;
}

.document-editor-shell :deep(.ProseMirror h1) {
  margin: 24px 0 8px;
  padding: 3px 0;
  color: #37352f;
  font-family: var(--document-content-font);
  font-size: 30px;
  font-weight: 700;
  line-height: 1.25;
  letter-spacing: 0;
}

.document-editor-shell :deep(.ProseMirror > h1:first-child) {
  margin: 0 0 18px;
}

.document-editor-shell :deep(.ProseMirror > h1:first-child.is-empty::before) {
  content: var(--editor-title-placeholder);
  float: left;
  height: 0;
  color: rgba(55, 53, 47, 0.2);
  pointer-events: none;
}

.document-editor-shell :deep(.ProseMirror p.is-empty::before) {
  content: var(--editor-empty-placeholder);
  float: left;
  height: 0;
  color: rgba(55, 53, 47, 0.35);
  pointer-events: none;
}

.document-editor-shell :deep(.ProseMirror p.is-empty::after) {
  position: absolute;
  top: 3px;
  left: -72px;
  width: 56px;
  height: 24px;
  color: rgba(55, 53, 47, 0.35);
  font-size: 20px;
  line-height: 22px;
  letter-spacing: 2px;
  content: "+ ⋮⋮";
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.12s ease;
}

.document-editor-shell :deep(.ProseMirror-focused p.is-empty::after),
.document-editor-shell :deep(.ProseMirror p.is-empty:hover::after) {
  opacity: 1;
}

.document-editor-shell :deep(.drag-handle) {
  left: -72px;
  width: 58px;
  height: 26px;
  gap: 6px;
  color: rgba(55, 53, 47, 0.45);
}

.document-editor-shell :deep(.drag-handle::before) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 4px;
  color: rgba(55, 53, 47, 0.45);
  font-size: 22px;
  line-height: 22px;
  content: "+";
}

.document-editor-shell :deep(.drag-handle:hover::before),
.document-editor-shell :deep(.drag-handle.active::before) {
  background: rgba(55, 53, 47, 0.08);
  color: rgba(55, 53, 47, 0.72);
}

.document-editor-shell :deep(.drag-handle svg) {
  width: 20px;
  height: 20px;
  color: rgba(55, 53, 47, 0.38);
}

.document-editor-shell :deep(.drag-handle:hover) {
  background: transparent;
}

.document-editor-shell :deep(.drag-handle.active) {
  background: transparent;
}

.document-editor-shell :deep(.slash-command-menu),
.document-editor-shell :deep(.floating-menu) {
  border: 1px solid rgba(55, 53, 47, 0.08);
  border-radius: 6px;
  box-shadow: 0 8px 24px rgba(15, 15, 15, 0.12), 0 0 0 1px rgba(15, 15, 15, 0.03);
}

.editor-page-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-height: 300px;
  color: rgba(55, 53, 47, 0.52);
  font-size: 14px;
}

.editor-page-state--error {
  flex-direction: column;
  gap: 14px;
  color: var(--td-error-color);
}

@media (max-width: 760px) {
  .organize-editor-header {
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 8px;
    padding: 8px 16px;
  }

  .editor-header-left {
    width: 100%;
  }

  .editor-page-actions {
    width: 100%;
    justify-content: flex-end;
    flex-wrap: wrap;
    row-gap: 6px;
  }

  .organize-editor-main {
    padding: 0 18px 48px;
  }

  .document-page {
    padding-top: 0;
  }

  .output-category-panel {
    align-items: stretch;
    flex-direction: column;
    gap: 10px;
    padding: 12px;
  }

  .output-category-panel :deep(.t-select) {
    flex-basis: auto;
    width: 100%;
  }

  .memory-note-panel {
    gap: 14px;
  }

  .memory-note-title-input :deep(.t-input__inner) {
    font-size: 20px;
  }

  .memory-note-tags {
    flex-direction: column;
  }

  .memory-note-tag-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .memory-note-source-card {
    grid-template-columns: 34px minmax(0, 1fr) 18px;
    min-height: 58px;
  }

  .memory-note-source-meta {
    display: none;
  }

  .memory-note-tabs {
    gap: 16px;
  }

  .memory-note-tab {
    padding-bottom: 12px;
    font-size: 14px;
  }

  .memory-note-service-card {
    padding: 16px;
  }

  .memory-note-service-head {
    flex-direction: column;
    gap: 10px;

    h2 {
      font-size: 20px;
    }
  }

  .memory-note-service-facts,
  .memory-note-service-source-list {
    grid-template-columns: 1fr;
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

  .memory-note-tag-panel {
    width: min(280px, 78vw);
  }

  .memory-audio-player {
    grid-template-columns: 34px minmax(0, 1fr);
    padding: 10px;
  }

  .memory-audio-transcript-chip {
    grid-column: 1 / -1;
    min-height: 0;
    padding-top: 4px;
    border-left: 0;
    justify-content: flex-start;
  }

  .document-editor-shell {
    min-height: 460px;
  }

  .document-editor-shell--memory-note {
    min-height: 520px;
  }

  .document-editor-shell :deep(.word-content-multi .ProseMirror) {
    min-height: calc(100vh - 190px);
    padding-bottom: 80px;
  }

  .document-editor-shell--memory-note :deep(.word-content-multi .ProseMirror) {
    min-height: calc(100vh - 390px);
  }
}
</style>
