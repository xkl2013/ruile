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
          v-if="isActionableMemory"
          theme="default"
          variant="outline"
          class="editor-organize-action"
          :class="`editor-organize-action--${memoryOrganizeHeaderState}`"
          :loading="memoryOrganizeCreating"
          :disabled="memoryOrganizeLoading || isActiveOrganizeJob(memoryOrganizeJob)"
          :aria-label="memoryOrganizeHeaderAriaLabel"
          @click="handleMemoryOrganizeHeaderAction"
        >
          <template #icon><t-icon name="layers" class="editor-organize-action-icon" /></template>
          {{ memoryOrganizeHeaderLabel }}
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
          <template v-if="isActionableMemory">
            <section class="memory-note-panel" aria-label="笔记详情">
              <div class="memory-note-header">
                <t-input v-model="title" class="memory-note-title-input" size="large" clearable :placeholder="`${memoryAssetLabel}标题`" />
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

              <section v-if="audioSourceCardVisible" class="memory-audio-panel memory-audio-panel--source" aria-label="录音播放">
                <div class="memory-audio-player" aria-label="录音播放">
                  <template v-if="audioPlayerUrl">
                    <audio class="memory-audio-native" :src="audioPlayerUrl" controls preload="metadata">
                      您的浏览器不支持音频播放
                    </audio>
                    <div class="memory-audio-transcript-chip">
                      <t-icon name="file-word" size="14px" />
                      <span>文稿</span>
                    </div>
                  </template>
                  <template v-else>
                    <div class="memory-audio-loading" aria-live="polite">
                      <t-loading v-if="audioPlayerLoading" size="small" />
                      <t-icon v-else name="error-circle" size="16px" />
                      <span>{{ audioPlayerLoading ? '加载音频中' : audioPlayerError || '音频暂不可播放' }}</span>
                    </div>
                    <div class="memory-audio-transcript-chip">
                      <t-icon name="file-word" size="14px" />
                      <span>文稿</span>
                    </div>
                  </template>
                </div>
              </section>

              <section v-if="memoryAttachments.length" class="memory-note-attachment-cards" aria-label="附件列表">
                <button
                  v-for="(attachment, index) in memoryAttachments"
                  :key="attachment.id"
                  type="button"
                  class="memory-note-source-card memory-note-attachment-card"
                  :aria-label="`预览附件 ${attachment.file_name}`"
                  @click="openMemoryAttachmentPreview(attachment)"
                >
                  <span class="memory-note-source-icon" aria-hidden="true">
                    <t-icon :name="memoryAttachmentIcon(attachment)" size="18px" />
                  </span>
                  <span class="memory-note-source-main">
                    <span class="memory-note-source-label">附件 {{ index + 1 }}</span>
                    <span class="memory-note-source-name">{{ attachment.file_name }}</span>
                  </span>
                  <span class="memory-note-source-meta">{{ memoryAttachmentKindLabel(attachment) }}</span>
                  <t-icon class="memory-note-source-arrow" name="file-view" size="16px" />
                </button>
              </section>

              <button
                v-if="!memoryAttachments.length && sourceFileCardVisible"
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
                  :class="{ 'is-active': noteActiveTab === 'result' }"
                  @click="noteActiveTab = 'result'"
                >
                  整理结果
                </button>
              </div>

              <div class="memory-note-tab-panels">
                <section v-show="noteActiveTab === 'content'" class="memory-note-panel-view">
                  <div
                    class="document-editor-shell document-editor-shell--memory-note"
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

                  <section v-if="memoryFileNotes.length" class="memory-file-notes" aria-label="文件解析笔记">
                    <div class="memory-file-notes__header">
                      <h2>附件解析笔记</h2>
                      <span
                        v-if="memoryAttachmentAggregateStatusLabel && memoryAttachmentAggregateStatusLabel !== '全部完成'"
                        class="memory-file-notes__status"
                      >
                        {{ memoryAttachmentAggregateStatusLabel }}
                      </span>
                    </div>

                    <div class="memory-file-note-list">
                      <article
                        v-for="fileNote in memoryFileNotes"
                        :key="fileNote.id"
                        class="memory-file-note"
                        :class="`memory-file-note--${fileNote.status}`"
                      >
                        <header class="memory-file-note__header">
                          <h3>[{{ fileNote.fileName }}]</h3>
                          <button
                            v-if="fileNote.canRetry"
                            type="button"
                            class="memory-file-note__retry"
                            :disabled="memoryAttachmentRetryingId === fileNote.id"
                            @click.stop="retryMemoryAttachment(fileNote.id)"
                          >
                            {{ memoryAttachmentRetryingId === fileNote.id ? '重试中' : '重新解析' }}
                          </button>
                        </header>

                        <template v-if="fileNote.status === 'completed'">
                          <section v-if="fileNote.summary" class="memory-file-note__section">
                            <h4>{{ fileNote.isTranscript ? '转写摘要' : '文件摘要' }}</h4>
                            <p>{{ fileNote.summary }}</p>
                          </section>
                          <section class="memory-file-note__section memory-file-note__section--body">
                            <h4>{{ fileNote.isTranscript ? '转写正文' : '正文内容' }}</h4>
                            <div
                              v-if="fileNote.renderedContent"
                              class="memory-file-note__content"
                              v-html="fileNote.renderedContent"
                            />
                            <p v-else class="memory-file-note__empty">该文件暂未生成可展示的解析内容。</p>
                          </section>
                        </template>
                        <p v-else class="memory-file-note__state">
                          {{ memoryFileNoteStateLabel(fileNote) }}
                        </p>
                      </article>
                    </div>
                  </section>
                </section>

                <section v-show="noteActiveTab === 'result'" class="memory-note-panel-view memory-note-panel-view--result">
                  <div v-if="memoryOrganizeLoading" class="memory-note-organize-loading">
                    <t-loading size="small" text="加载整理结果中" />
                  </div>
                  <article
                    v-else-if="memoryOrganizeCard"
                    class="memory-note-organize-card organize-result-card"
                    role="button"
                    tabindex="0"
                    @click="openMemoryOrganizeCard"
                    @keydown.enter.self="openMemoryOrganizeCard"
                  >
                    <div class="organize-result-gutter" aria-hidden="true">
                      <t-icon name="layers" size="20px" />
                    </div>

                    <div class="organize-result-main">
                      <div class="organize-result-title-row">
                        <h2>{{ memoryOrganizeCard.title }}</h2>
                        <span class="type-badge" :class="`organize-stage--${memoryOrganizeCard.stageKey}`">{{ memoryOrganizeCard.stage }}</span>
                      </div>
                      <p class="organize-result-intro">{{ memoryOrganizeCard.intro }}</p>

                      <div v-if="memoryOrganizeCard.chips.length" class="organize-result-chips">
                        <span v-for="chip in memoryOrganizeCard.chips" :key="chip">{{ chip }}</span>
                      </div>

                      <div class="report-meta organize-result-meta">
                        <span>{{ memoryOrganizeCard.updated }}</span>
                        <span
                          v-if="memoryOrganizeCard.referenceLabels.length || memoryOrganizeCard.sourceLabels.length"
                          class="organize-result-meta-separator"
                        >
                          |
                        </span>
                        <span v-for="label in memoryOrganizeCard.referenceLabels" :key="`note-organize-ref-${label}`">{{ label }}</span>
                        <span v-for="label in memoryOrganizeCard.sourceLabels" :key="`note-organize-source-${label}`">{{ label }}</span>
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
      :header="sourcePreviewFileName || '源文件预览'"
      :footer="false"
      :close-btn="false"
      size="min(860px, 92vw)"
      destroy-on-close
    >
      <section v-if="sourcePreviewVisible && sourcePreviewFileUrl" class="memory-source-preview-body">
        <DocumentPreview
          :source-url="sourcePreviewFileUrl"
          :file-type="sourcePreviewFileType"
          :file-name="sourcePreviewFileName || '源文件'"
          :active="sourcePreviewVisible"
          fill-height
        />
      </section>
    </t-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { marked } from 'marked'
import { useRoute, useRouter } from 'vue-router'
import { TiptapProEditor, type FeatureConfig, type TiptapProEditorExpose } from 'tiptap-ui-kit'
import 'tiptap-ui-kit/style.css'
import DocumentPreview from '@/components/document-preview.vue'
import { getDown } from '@/utils/request'
import { sanitizeHTML, sanitizeMarkdownHTML, safeMarkdownToHTML } from '@/utils/security'
import {
  createOrganizeJob,
  createOrganizeMemory,
  createOrganizeOutput,
  getOrganizeMemory,
  getOrganizeOutput,
  listOrganizeDiscoverCategories,
  listOrganizeConfigs,
  listOrganizeJobs,
  retryOrganizeMemoryAttachment,
  updateOrganizeMemory,
  updateOrganizeOutput,
  type OrganizeConfig,
  type OrganizeJob,
  type OrganizeMemory,
  type OrganizeMemoryAttachment,
  type OrganizeMemoryKind,
  type OrganizeOutput,
  type OrganizeOutputStatus,
} from '@/api/organize'
import {
  clearOrganizeEditorDraft,
  readOrganizeEditorDraft,
  type OrganizeEditorDraft,
} from './editorDraftStorage'
import {
  buildSmartNoteTags,
  mergeNoteMetadata,
  normalizeNoteTags,
} from './noteEditor'
import {
  DISCOVER_CATEGORY_OPTIONS,
  discoverCategoryLabel,
  normalizeDiscoverCategory,
} from './discoverCategories'

type OrganizeDocumentType = 'memory' | 'output'
type SaveState = 'idle' | 'saving' | 'saved' | 'waiting' | 'error'
type MemoryFileNoteStatus = OrganizeMemoryAttachment['status']

interface MemoryFileNote {
  id: string
  fileName: string
  mimeType: string
  status: MemoryFileNoteStatus
  content: string
  transcript: string
  summary: string
  renderedContent: string
  isTranscript: boolean
  canRetry: boolean
  errorMessage: string
  metadata: Record<string, unknown>
}

const AUTOSAVE_DELAY = 700

const route = useRoute()
const router = useRouter()

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
const memoryAttachments = ref<OrganizeMemoryAttachment[]>([])
const memoryTags = ref<string[]>([])
const memoryOccurredAt = ref('')
const memoryCreatedAt = ref('')
const memoryUpdatedAt = ref('')
const noteTagDraft = ref('')
const noteTagMenuVisible = ref(false)
const memoryOrganizeJob = ref<OrganizeJob | null>(null)
const memoryOrganizeCreating = ref(false)
const memoryOrganizeLoading = ref(false)
const memoryAttachmentRetryingId = ref('')
const noteActiveTab = ref<'content' | 'result'>('content')
const sourcePreviewVisible = ref(false)
const sourcePreviewAttachment = ref<OrganizeMemoryAttachment | null>(null)
const audioPlayerUrl = ref('')
const audioPlayerLoading = ref(false)
const audioPlayerError = ref('')
const outputDraft = ref<OrganizeOutput | null>(null)
const outputCategory = ref('')
const discoverCategoryOptions = ref<Array<{ label: string; value: string }>>([...DISCOVER_CATEGORY_OPTIONS])
let memoryOrganizeRequestSeq = 0
let audioPlayerRequestSeq = 0
let audioPlayerObjectUrl = ''

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
  return value === 'memory' || value === 'output' ? value : ''
})
const documentId = computed(() => readParam(route.params.id))
const activeDocumentId = computed(() => savedDocumentId.value || documentId.value)
const isCreate = computed(() => !savedDocumentId.value && documentId.value === 'new')

const memoryAssetLabel = computed(() => {
  if (memoryKind.value === 'audio') return '录音'
  if (memoryKind.value === 'audio_card') return '工牌'
  return '笔记'
})

const isAudioMemory = computed(
  () => documentType.value === 'memory' && (memoryKind.value === 'audio' || memoryKind.value === 'audio_card'),
)
const memoryTranscriptText = computed(() => {
  const metadata = memoryMetadata.value || {}
  const direct = asTrimmedString(metadata.raw_transcript) || asTrimmedString(metadata.transcript) || asTrimmedString(metadata.transcription)
  if (direct) return direct
  return ''
})
const memoryAttachmentStatusLabel = (status: OrganizeMemoryAttachment['status']) => {
  if (status === 'pending') return '等待解析'
  if (status === 'processing') return '解析中'
  if (status === 'completed') return '已完成'
  if (status === 'skipped') return '已跳过'
  return '解析失败'
}
const memoryAttachmentAggregateStatusLabel = computed(() => {
  const metadata = memoryMetadata.value || {}
  const status = asTrimmedString(metadata.attachment_status) || asTrimmedString(metadata.transcription_status)
  if (status === 'pending') return '等待解析'
  if (status === 'processing' || status === 'transcribing') return '解析中'
  if (status === 'partial') return '部分完成'
  if (status === 'completed') return '全部完成'
  if (status === 'failed') return '解析失败'
  return ''
})
const editorPlaceholder = computed(() => {
  if (isAudioMemory.value) return '录音转写内容'
  if (isMemoryDocument.value) return '输入正文'
  return '输入内容，或按“/”启用命令'
})

const typeLabel = computed(() => {
  if (documentType.value === 'output') return '发现文档'
  return memoryAssetLabel.value
})

const breadcrumbRootLabel = computed(() => {
  if (documentType.value === 'output') return '发现'
  return '记忆'
})

const breadcrumbItems = computed(() => {
  const root = breadcrumbRootLabel.value
  const section = String(typeLabel.value)
  return section && section !== root ? [root, section] : [root]
})

const isMemoryDocument = computed(() => documentType.value === 'memory')
const isActionableMemory = computed(() => isMemoryDocument.value)
const editorVersion = computed(() => (isMemoryDocument.value ? 'advanced' : 'basic'))

const defaultReturnTo = computed(() => {
  if (documentType.value === 'output') return '/platform/organize/output'
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
  return '记忆'
})

const escapeHtml = (value: string) => {
  const node = document.createElement('div')
  node.textContent = value
  return node.innerHTML
}

const normalizeTitle = (value = '') => value.trim().slice(0, 512)

const asTrimmedString = (value: unknown) => (typeof value === 'string' ? value.trim() : '')

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

const fileIconForType = (fileType = '') => {
  if (fileType === 'pdf') return 'file-pdf'
  if (['doc', 'docx'].includes(fileType)) return 'file-word'
  if (['xls', 'xlsx', 'csv'].includes(fileType)) return 'file-excel'
  if (['ppt', 'pptx'].includes(fileType)) return 'file-powerpoint'
  if (['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp', 'tiff', 'svg'].includes(fileType)) return 'image'
  if (['mp3', 'wav', 'm4a', 'flac', 'ogg'].includes(fileType)) return 'sound'
  if (['mp4', 'mov', 'webm', 'avi', 'mkv', 'wmv', 'flv'].includes(fileType)) return 'play-circle'
  return 'file'
}

const readableTextFromFileContent = (value = '') => {
  if (!value.trim()) return ''
  if (/<[a-z][\s\S]*>/i.test(value)) {
    return plainTextFromHtml(value)
  }
  return value
    .replace(/!\[[^\]]*]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)]\([^)]*\)/g, '$1')
    .replace(/^\s*#{1,6}\s+/gm, '')
    .replace(/^\s*[-*+]\s+/gm, '')
    .replace(/^\s*\d+[.、)]\s+/gm, '')
    .replace(/[*_`~]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}

const compactFileSummary = (value = '') => {
  const readable = readableTextFromFileContent(value)
  if (!readable) return ''
  if (readable.length <= 180) return readable
  const boundary = readable.lastIndexOf('。', 180)
  const end = boundary >= 80 ? boundary + 1 : 180
  return `${readable.slice(0, end)}...`
}

const metadataText = (metadata: Record<string, unknown> | undefined, keys: string[]) => {
  if (!metadata) return ''
  for (const key of keys) {
    const value = asTrimmedString(metadata[key])
    if (value) return value
  }
  return ''
}

const renderMemoryFileNoteContent = (value = '') => {
  const source = value.trim()
  if (!source) return ''
  if (/<\/?(h[1-6]|p|ul|ol|li|blockquote|div|table|article|section|br)\b/i.test(source)) {
    return sanitizeHTML(source)
  }
  const html = marked.parse(safeMarkdownToHTML(source), {
    gfm: true,
    breaks: true,
    async: false,
  }) as string
  return sanitizeMarkdownHTML(html)
}

const memoryFileNoteStatusFromMetadata = (hasContent: boolean): MemoryFileNoteStatus => {
  const metadata = memoryMetadata.value || {}
  const status = asTrimmedString(metadata.attachment_status) || asTrimmedString(metadata.transcription_status)
  if (status === 'pending') return 'pending'
  if (status === 'processing' || status === 'transcribing') return 'processing'
  if (status === 'failed') return 'failed'
  if (status === 'completed' || status === 'partial') return 'completed'
  return hasContent ? 'completed' : 'pending'
}

const memoryFileNotes = computed<MemoryFileNote[]>(() => {
  const attachments = memoryAttachments.value.map((attachment) => {
    const metadata = attachment.metadata || {}
    const transcript = asTrimmedString(attachment.transcript)
    const content = asTrimmedString(attachment.content)
    const body = transcript || content
    const attachmentFileType = fileExtension(attachment.file_name) || fileTypeFromMime(attachment.mime_type || '')
    const isTranscript = Boolean(transcript) || ['mp3', 'wav', 'm4a', 'flac', 'ogg', 'mp4', 'mov', 'webm'].includes(attachmentFileType)
    const summary = metadataText(metadata, ['summary', 'file_summary', 'summary_text', 'description']) ||
      (memoryAttachments.value.length === 1
        ? metadataText(memoryMetadata.value, ['summary', 'file_summary', 'summary_text'])
        : '') ||
      compactFileSummary(body)

    return {
      id: attachment.id,
      fileName: attachment.file_name || '未命名文件',
      mimeType: attachment.mime_type || '',
      status: attachment.status,
      content,
      transcript,
      summary,
      renderedContent: renderMemoryFileNoteContent(body),
      isTranscript,
      canRetry: attachment.status === 'failed' || attachment.status === 'skipped',
      errorMessage: attachment.error_message || '',
      metadata,
    }
  })

  if (attachments.length > 0) return attachments

  const legacyTranscript = memoryTranscriptText.value
  if (legacyTranscript) {
    return [{
      id: 'legacy-transcript',
      fileName: sourceFileName.value || title.value || '原始转写',
      mimeType: asTrimmedString(memoryMetadata.value?.mime_type),
      status: 'completed',
      content: legacyTranscript,
      transcript: legacyTranscript,
      summary: metadataText(memoryMetadata.value, ['summary', 'file_summary', 'summary_text']) ||
        compactFileSummary(legacyTranscript),
      renderedContent: renderMemoryFileNoteContent(legacyTranscript),
      isTranscript: true,
      canRetry: false,
      errorMessage: '',
      metadata: memoryMetadata.value || {},
    }]
  }

  const legacyFileContent = content.value.trim()
  const hasLegacySource = Boolean(
    sourceFilePath.value ||
    metadataText(memoryMetadata.value, ['file_name', 'fileName', 'original_name', 'originalName', 'filename']),
  )
  if (!hasLegacySource) return []

  const isPlaceholder = /文件已保存，等待解析/.test(readableTextFromFileContent(legacyFileContent))
  const displayContent = isPlaceholder ? '' : legacyFileContent
  const status = memoryFileNoteStatusFromMetadata(Boolean(displayContent))
  return [{
    id: 'legacy-source-file',
    fileName: sourceFileName.value || '源文件',
    mimeType: asTrimmedString(memoryMetadata.value?.mime_type) || asTrimmedString(memoryMetadata.value?.mimeType),
    status,
    content: displayContent,
    transcript: '',
    summary: metadataText(memoryMetadata.value, ['summary', 'file_summary', 'summary_text']) ||
      compactFileSummary(displayContent),
    renderedContent: renderMemoryFileNoteContent(displayContent),
    isTranscript: false,
    canRetry: false,
    errorMessage: '',
    metadata: memoryMetadata.value || {},
  }]
})

const memoryFileNoteStateLabel = (fileNote: MemoryFileNote) => {
  if (fileNote.status === 'pending') return '文件已保存，等待解析。'
  if (fileNote.status === 'processing') return '正在解析文件内容，请稍候。'
  if (fileNote.status === 'skipped') return fileNote.errorMessage || '该文件已跳过解析。'
  return fileNote.errorMessage || '文件解析失败，请重新解析。'
}

const isProviderFilePath = (value = '') => /^[a-z][a-z\d+.-]*:\/\//i.test(value)
  && !/^https?:\/\//i.test(value)
  && !value.startsWith('blob:')
  && !value.startsWith('data:')

const buildFileProxyUrl = (source = '') => `/files?${new URLSearchParams({ file_path: source }).toString()}`

const normalizeAuthenticatedFileProxyUrl = (value = '') => {
  const source = value.trim()
  if (!source) return ''
  if (source.startsWith('/api/v1/files?')) {
    return source.replace(/^\/api\/v1\/files(?=\?)/, '/files')
  }
  if (!/^https?:\/\//i.test(source)) return source

  try {
    const url = new URL(source)
    if (url.pathname === '/api/v1/files' && url.searchParams.has('file_path')) {
      url.pathname = '/files'
      return url.toString()
    }
  } catch {
    return source
  }
  return source
}

const playbackUrlFromSource = (value = '') => {
  const source = normalizeAuthenticatedFileProxyUrl(value)
  if (!source) return ''
  if (isProviderFilePath(source)) return buildFileProxyUrl(source)
  if (
    /^https?:\/\//i.test(source) ||
    source.startsWith('//') ||
    source.startsWith('/') ||
    source.startsWith('blob:') ||
    source.startsWith('data:')
  ) {
    return source
  }
  return buildFileProxyUrl(source)
}

const shouldHydrateAudioPlaybackUrl = (value = '') => {
  const source = normalizeAuthenticatedFileProxyUrl(value)
  if (!source) return false
  if (source.startsWith('/files?') && source.includes('file_path=')) return true
  if (!/^https?:\/\//i.test(source)) return false

  try {
    const url = new URL(source)
    const apiBase = (import.meta.env.VITE_API_BASE_URL || '').trim()
    const apiBaseOrigin = /^https?:\/\//i.test(apiBase) ? new URL(apiBase).origin : ''
    const sameTrustedOrigin = typeof window === 'undefined'
      ? Boolean(apiBaseOrigin)
      : url.origin === window.location.origin || Boolean(apiBaseOrigin && url.origin === apiBaseOrigin)
    return sameTrustedOrigin && url.pathname === '/files' && url.searchParams.has('file_path')
  } catch {
    return false
  }
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
  return fileIconForType(sourceFilePreviewType.value)
})

const sourceFilePreviewUrl = computed(() => {
  const source = sourceFilePath.value
  if (!source) return ''
  return playbackUrlFromSource(source)
})

const memoryAttachmentSourcePath = (attachment: OrganizeMemoryAttachment) => {
  const metadata = attachment.metadata || {}
  return (
    asTrimmedString(attachment.storage_path) ||
    asTrimmedString(attachment.storage_url) ||
    asTrimmedString(metadata.file_path) ||
    asTrimmedString(metadata.filePath) ||
    asTrimmedString(metadata.source_path) ||
    asTrimmedString(metadata.sourcePath) ||
    asTrimmedString(metadata.file_url) ||
    asTrimmedString(metadata.fileUrl) ||
    asTrimmedString(metadata.source_url) ||
    asTrimmedString(metadata.sourceUrl)
  )
}

const memoryAttachmentType = (attachment: OrganizeMemoryAttachment) =>
  fileExtension(attachment.file_name) || fileTypeFromMime(attachment.mime_type || '')

const memoryAttachmentKindLabel = (attachment: OrganizeMemoryAttachment) => {
  const fileType = memoryAttachmentType(attachment)
  return fileType ? fileType.toUpperCase() : '附件'
}

const memoryAttachmentIcon = (attachment: OrganizeMemoryAttachment) =>
  fileIconForType(memoryAttachmentType(attachment))

const sourcePreviewFileName = computed(() =>
  sourcePreviewAttachment.value?.file_name || sourceFileName.value,
)

const sourcePreviewFileType = computed(() => {
  if (sourcePreviewAttachment.value) {
    return memoryAttachmentType(sourcePreviewAttachment.value) || 'bin'
  }
  return sourceFilePreviewType.value
})

const sourcePreviewFileUrl = computed(() => {
  if (sourcePreviewAttachment.value) {
    const source = memoryAttachmentSourcePath(sourcePreviewAttachment.value)
    return source ? playbackUrlFromSource(source) : ''
  }
  return sourceFilePreviewUrl.value
})

const sourceFileCardVisible = computed(() => isMemoryDocument.value && Boolean(sourceFilePath.value))

const openSourceFilePreview = () => {
  sourcePreviewAttachment.value = null
  if (!sourcePreviewFileUrl.value) {
    MessagePlugin.warning('暂无源文件')
    return
  }
  sourcePreviewVisible.value = true
}

const openMemoryAttachmentPreview = (attachment: OrganizeMemoryAttachment) => {
  sourcePreviewAttachment.value = attachment
  if (!sourcePreviewFileUrl.value) {
    sourcePreviewAttachment.value = null
    MessagePlugin.warning('暂无附件预览')
    return
  }
  sourcePreviewVisible.value = true
}

const markDocumentDirty = () => {
  if (!editorReady.value || loading.value) return
  editRevision += 1
  saveError.value = ''
  saveState.value = 'idle'
  scheduleAutosave()
}

const currentEditor = () => editorRef.value?.getEditor() || null

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

const activeOrganizeJobStatuses: OrganizeJob['status'][] = ['queued', 'running', 'repairing']
const finishedOrganizeJobStatuses: OrganizeJob['status'][] = ['completed', 'fallback']

const isActiveOrganizeJob = (job?: OrganizeJob | null) =>
  Boolean(job && activeOrganizeJobStatuses.includes(job.status))

const isFinishedOrganizeJob = (job?: OrganizeJob | null) =>
  Boolean(job && finishedOrganizeJobStatuses.includes(job.status))

const organizeJobStageLabel = (job?: OrganizeJob | null) => {
  if (!job) return '整理'
  if (isActiveOrganizeJob(job)) return '整理中'
  if (job.status === 'failed') return '整理失败'
  if (job.status === 'canceled') return '已取消'
  if (isFinishedOrganizeJob(job)) return '已完成'
  return job.stage || '整理'
}

const loadMemoryOrganizeJob = async (memoryID: string, options?: { silent?: boolean }) => {
  if (!memoryID || memoryID === 'new') return
  const requestSeq = ++memoryOrganizeRequestSeq
  if (!options?.silent) memoryOrganizeLoading.value = true
  try {
    const response = await listOrganizeJobs({ page: 1, page_size: 100 })
    if (requestSeq !== memoryOrganizeRequestSeq) return
    if (!response.success || !response.data) {
      throw new Error(response.message || '整理任务加载失败')
    }
    const linkedJobs = response.data.items
      .filter((job) => (job.memory_ids || []).includes(memoryID))
      .sort((left, right) => {
        const leftTime = new Date(left.updated_at || left.created_at).getTime()
        const rightTime = new Date(right.updated_at || right.created_at).getTime()
        return rightTime - leftTime
      })
    memoryOrganizeJob.value = linkedJobs[0] || null
  } catch {
    if (!options?.silent) memoryOrganizeJob.value = null
  } finally {
    if (requestSeq === memoryOrganizeRequestSeq && !options?.silent) {
      memoryOrganizeLoading.value = false
    }
  }
}

const retryMemoryAttachment = async (attachmentID: string) => {
  const memoryID = activeDocumentId.value
  if (!memoryID || memoryID === 'new' || memoryAttachmentRetryingId.value) return
  memoryAttachmentRetryingId.value = attachmentID
  try {
    const response = await retryOrganizeMemoryAttachment(memoryID, attachmentID)
    if (!response.success || !response.data) {
      throw new Error(response.message || '附件重试失败')
    }
    memoryAttachments.value = response.data.attachments || []
    memoryMetadata.value = response.data.metadata
    MessagePlugin.success('已重新发起解析')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '附件重试失败')
  } finally {
    memoryAttachmentRetryingId.value = ''
  }
}

const scheduleMemoryOrganizeRefresh = (memoryID: string) => {
  const refresh = () => {
    void loadMemoryOrganizeJob(memoryID, { silent: true })
  }
  window.setTimeout(refresh, 1500)
  window.setTimeout(refresh, 5000)
  window.setTimeout(refresh, 10000)
}

const createMemoryOrganizeJob = async () => {
  if (!isMemoryDocument.value || memoryOrganizeCreating.value) return
  if (!savedDocumentId.value && documentId.value === 'new') {
    await saveDocument()
  }

  const memoryID = activeDocumentId.value
  if (!memoryID || memoryID === 'new') {
    MessagePlugin.warning('请先保存记忆')
    return
  }

  memoryOrganizeCreating.value = true
  try {
    const configResponse = await listOrganizeConfigs({ page: 1, page_size: 100, status: 'active' })
    if (!configResponse.success) {
      throw new Error(configResponse.message || '整理配置加载失败')
    }
    const config: OrganizeConfig | undefined = configResponse.data?.items?.[0]
    if (!config) {
      MessagePlugin.warning('请先在整理工作台创建启用的整理配置')
      return
    }

    const response = await createOrganizeJob({
      config_id: config.id,
      memory_ids: [memoryID],
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '整理任务创建失败')
  }
  memoryOrganizeJob.value = response.data
    noteActiveTab.value = 'result'
    MessagePlugin.success(`已按「${config.name}」发起整理任务`)
    scheduleMemoryOrganizeRefresh(memoryID)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '整理任务创建失败')
  } finally {
    memoryOrganizeCreating.value = false
  }
}

const memoryOrganizeHeaderState = computed(() => {
  const job = memoryOrganizeJob.value
  if (memoryOrganizeCreating.value || isActiveOrganizeJob(job)) return 'organizing'
  if (isFinishedOrganizeJob(job)) return 'formed'
  return 'idle'
})

const memoryOrganizeHeaderLabel = computed(() => {
  const state = memoryOrganizeHeaderState.value
  if (state === 'organizing') return '整理中'
  if (state === 'formed') return '查看整理结果'
  return '整理'
})

const memoryOrganizeHeaderAriaLabel = computed(() => {
  const state = memoryOrganizeHeaderState.value
  if (state === 'organizing') return '整理任务生成中'
  if (state === 'formed') return '查看整理结果'
  return '发起整理任务'
})

const memoryOrganizeTaskCard = computed(() => {
  const job = memoryOrganizeJob.value
  if (!job) return null
  const configName = typeof job.requirement?.config_name === 'string'
    ? job.requirement.config_name
    : '整理任务'
  const active = isActiveOrganizeJob(job)
  return {
    id: job.id,
    title: configName,
    intro: active
      ? job.summary || '整理任务正在处理这条记忆，完成后会生成整理结果。'
      : job.summary || (isFinishedOrganizeJob(job) ? '整理结果已生成，可进入结果页查看。' : '整理任务暂未完成。'),
    stage: organizeJobStageLabel(job),
    stageKey: active ? 'organizing' : isFinishedOrganizeJob(job) ? 'formed' : 'expandable',
    updated: job.updated_at ? formatUpdatedLabel(job.updated_at) : '刚刚 --:--',
    chips: [],
    referenceLabels: [`任务 ${job.id.slice(0, 8)}`],
    sourceLabels: [],
  }
})

const memoryOrganizeCard = computed(() => memoryOrganizeTaskCard.value)

const openMemoryOrganizeCard = () => {
  const outputID = memoryOrganizeJob.value?.output_id
  if (outputID) {
    void router.push({
      path: `/platform/organize/outputs/${encodeURIComponent(outputID)}`,
      query: { from: 'memory', memoryId: activeDocumentId.value },
    })
    return
  }
  noteActiveTab.value = 'result'
}

const handleMemoryOrganizeHeaderAction = () => {
  if (memoryOrganizeCreating.value || memoryOrganizeLoading.value || isActiveOrganizeJob(memoryOrganizeJob.value)) return
  if (isFinishedOrganizeJob(memoryOrganizeJob.value)) {
    openMemoryOrganizeCard()
    return
  }
  void createMemoryOrganizeJob()
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
    asTrimmedString(metadata.audio_file_url) ||
    asTrimmedString(metadata.audioFileUrl) ||
    asTrimmedString(metadata.audio_path) ||
    asTrimmedString(metadata.audioPath) ||
    asTrimmedString(metadata.audio_file_path) ||
    asTrimmedString(metadata.audioFilePath) ||
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
    asTrimmedString(metadata.file_path) ||
    asTrimmedString(metadata.storage_path) ||
    asTrimmedString(metadata.url)
  )
}

const audioSourceUrl = computed(() => {
  if (!isAudioMemory.value) return ''
  const metadata = memoryMetadata.value || {}
  const source = audioSourcePath(metadata) || sourceFilePath.value
  if (!source) return ''
  return playbackUrlFromSource(source)
})

const audioSourceCardVisible = computed(() => isAudioMemory.value && Boolean(audioSourceUrl.value))

const audioMimeType = computed(() => {
  const metadata = memoryMetadata.value || {}
  return (
    asTrimmedString(metadata.audio_mime_type) ||
    asTrimmedString(metadata.audioMimeType) ||
    asTrimmedString(metadata.mime_type) ||
    asTrimmedString(metadata.mimeType) ||
    (sourceFilePreviewType.value === 'wav' ? 'audio/wav' : '') ||
    (sourceFilePreviewType.value === 'm4a' ? 'audio/mp4' : '') ||
    (sourceFilePreviewType.value === 'flac' ? 'audio/flac' : '') ||
    (sourceFilePreviewType.value === 'ogg' ? 'audio/ogg' : '') ||
    (sourceFilePreviewType.value === 'mp3' ? 'audio/mpeg' : '')
  )
})

const revokeAudioPlayerObjectUrl = () => {
  if (!audioPlayerObjectUrl) return
  URL.revokeObjectURL(audioPlayerObjectUrl)
  audioPlayerObjectUrl = ''
}

const loadAudioPlayerUrl = async (sourceUrl: string) => {
  const requestSeq = ++audioPlayerRequestSeq
  revokeAudioPlayerObjectUrl()
  audioPlayerUrl.value = ''
  audioPlayerError.value = ''
  if (!sourceUrl) {
    audioPlayerLoading.value = false
    return
  }
  if (!shouldHydrateAudioPlaybackUrl(sourceUrl)) {
    audioPlayerLoading.value = false
    audioPlayerUrl.value = sourceUrl
    return
  }

  audioPlayerLoading.value = true
  try {
    const rawBlob = await getDown(sourceUrl)
    if (requestSeq !== audioPlayerRequestSeq) return
    const blob = rawBlob.type || !audioMimeType.value
      ? rawBlob
      : new Blob([rawBlob], { type: audioMimeType.value })
    const blobUrl = URL.createObjectURL(blob)
    audioPlayerObjectUrl = blobUrl
    audioPlayerUrl.value = blobUrl
  } catch {
    if (requestSeq !== audioPlayerRequestSeq) return
    audioPlayerError.value = '音频加载失败'
  } finally {
    if (requestSeq === audioPlayerRequestSeq) {
      audioPlayerLoading.value = false
    }
  }
}

const readQuery = (key: string) => readParam(route.query[key])

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

  if (kind) draft.kind = kind as OrganizeMemoryKind
  if (source) draft.source = source
  if (Number.isFinite(durationSeconds) && durationSeconds > 0) draft.duration_seconds = durationSeconds
  if (outputType) draft.output_type = outputType
  if (sourceSummary) draft.source_summary = sourceSummary
  if (status) draft.status = status as OrganizeOutputStatus
  if (icon) draft.icon = icon

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

const clearAutosaveTimer = () => {
  if (autosaveTimer) {
    clearTimeout(autosaveTimer)
    autosaveTimer = null
  }
}

const resetDraft = () => {
  const draft = readInitialDraft()
  const draftTitle = normalizeTitle(draft.title || extractTitleFromContent(draft.content))
  const draftContent = draft.content
  title.value = draftTitle
  memoryKind.value = draft.kind || 'note'
  memorySource.value = draft.source || '手动输入'
  memoryDurationSeconds.value = draft.duration_seconds || 0
  memoryMetadata.value = draft.metadata
  memoryAttachments.value = []
  memoryTags.value = noteTagsFromMetadata(draft.metadata)
  outputCategory.value = normalizeDiscoverCategory(
    draft.metadata?.discover_category || draft.metadata?.discover_category_label,
  ) || asTrimmedString(draft.metadata?.discover_category || draft.metadata?.discover_category_label)
  memoryOccurredAt.value = ''
  memoryCreatedAt.value = ''
  memoryUpdatedAt.value = ''
  noteTagDraft.value = ''
  noteTagMenuVisible.value = false
  memoryOrganizeJob.value = null
  memoryOrganizeLoading.value = false
  sourcePreviewVisible.value = false
  sourcePreviewAttachment.value = null
  memoryOrganizeRequestSeq += 1
  noteActiveTab.value = 'content'
  if (documentType.value === 'memory' && memoryKind.value === 'audio') {
    content.value = normalizeAudioMemoryContent(draftContent)
    title.value = draftTitle || audioMemoryFallbackTitle(content.value)
  } else if (isMemoryDocument.value) {
    content.value = memoryBodyContent(draftTitle, draftContent)
  } else {
    content.value = normalizeDocumentContent(draftTitle, draftContent)
  }
  outputDraft.value = outputFromDraft(draft)
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
      memoryAttachments.value = item.attachments || []
      memoryTags.value = noteTagsFromMetadata(item.metadata)
      memoryOccurredAt.value = item.occurred_at || item.created_at || item.updated_at || ''
      memoryCreatedAt.value = item.created_at || ''
      memoryUpdatedAt.value = item.updated_at || ''
      noteTagDraft.value = ''
      noteTagMenuVisible.value = false
      memoryOrganizeJob.value = null
      memoryOrganizeLoading.value = false
      sourcePreviewVisible.value = false
      sourcePreviewAttachment.value = null
      noteActiveTab.value = 'content'
      title.value = item.kind === 'audio' || item.kind === 'audio_card'
        ? audioMemoryDisplayTitle(item)
        : isMemoryDocument.value
          ? normalizeTitle(item.title || extractTitleFromContent(item.content) || plainTextFromHtml(item.content).slice(0, 80))
          : item.title
      content.value = item.kind === 'audio' || item.kind === 'audio_card'
        ? normalizeAudioMemoryContent(audioMemoryContentSource(item))
        : isMemoryDocument.value
          ? memoryBodyContent(item.title, item.content)
          : normalizeDocumentContent(item.title, item.content)
      if (isActionableMemory.value) {
        await loadMemoryOrganizeJob(item.id)
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
      ) || asTrimmedString(item.metadata?.discover_category || item.metadata?.discover_category_label)
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
    : currentType === 'memory'
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
          discover_category_label: discoverCategoryOptions.value.find((item) => item.value === outputCategory.value)?.label
            || discoverCategoryLabel(outputCategory.value)
            || outputCategory.value,
        },
      }
      const response = creating
        ? await createOrganizeOutput(input)
        : await updateOrganizeOutput(currentDocumentId, input)
      if (!response.success || !response.data) throw new Error(response.message || '发现保存失败')
      savedId = response.data.id
      outputDraft.value = response.data
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

const loadDiscoverCategories = async () => {
  try {
    const response = await listOrganizeDiscoverCategories()
    if (response.success && response.data?.length) {
      discoverCategoryOptions.value = response.data
        .filter((item) => item.status === 'enabled')
        .map((item) => ({ label: item.label, value: item.key }))
    }
  } catch {
    // Keep the built-in options when the public category endpoint is unavailable.
  }
}

onMounted(() => {
  void loadDiscoverCategories()
})

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
  audioSourceUrl,
  (sourceUrl) => {
    void loadAudioPlayerUrl(sourceUrl)
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  audioPlayerRequestSeq += 1
  revokeAudioPlayerObjectUrl()
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

.editor-organize-action {
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

.editor-organize-action:hover {
  border-color: rgba(34, 101, 73, 0.32);
  background: rgba(34, 101, 73, 0.04);
  color: #20242a;
}

.editor-organize-action :deep(.t-button__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 8px;
}

.editor-organize-action-icon {
  font-size: 18px;
  color: #22c55e;
}

.editor-organize-action--organizing .editor-organize-action-icon {
  color: #2eaadc;
}

.editor-organize-action--formed {
  border-color: rgba(34, 101, 73, 0.24);
  background: rgba(34, 101, 73, 0.06);
  color: #236549;
}

.editor-organize-action--formed:hover {
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

.memory-audio-panel--source {
  margin-bottom: 0;
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

.memory-audio-native {
  width: 100%;
  min-width: 0;
  grid-column: 1 / 3;
  height: 32px;
}

.memory-audio-loading {
  display: inline-flex;
  grid-column: 1 / 3;
  align-items: center;
  gap: 8px;
  min-width: 0;
  color: rgba(55, 53, 47, 0.58);
  font-size: 13px;
  line-height: 18px;
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

.memory-note-attachment-cards {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  width: min(860px, 100%);
}

.memory-note-attachment-card {
  width: 100%;
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

.memory-file-notes {
  margin-top: 16px;
  padding: 24px 22px 34px;
  border-radius: 0 0 8px 8px;
  background: #f1f1f1;
}

.memory-file-notes__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 42px;
}

.memory-file-notes__header h2 {
  margin: 0;
  color: #37352f;
  font-size: 28px;
  font-weight: 700;
  line-height: 1.25;
}

.memory-file-notes__status {
  flex: 0 0 auto;
  color: rgba(55, 53, 47, 0.48);
  font-size: 12px;
  line-height: 1.4;
}

.memory-file-note-list {
  display: flex;
  flex-direction: column;
  gap: 38px;
}

.memory-file-note {
  padding: 0;
}

.memory-file-note--failed,
.memory-file-note--skipped {
  color: #c9372c;
}

.memory-file-note__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}

.memory-file-note__header h3 {
  margin: 0;
  color: #37352f;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.35;
}

.memory-file-note__retry {
  flex: 0 0 auto;
  padding: 4px 0;
  border: 0;
  background: transparent;
  color: #2383c4;
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.memory-file-note__retry:disabled {
  cursor: default;
  opacity: 0.55;
}

.memory-file-note__section + .memory-file-note__section {
  margin-top: 26px;
}

.memory-file-note__section h4 {
  margin: 0 0 10px;
  color: #37352f;
  font-size: 24px;
  font-weight: 700;
  line-height: 1.35;
}

.memory-file-note__section > p {
  margin: 0;
  color: rgba(55, 53, 47, 0.78);
  font-size: 16px;
  line-height: 1.8;
}

.memory-file-note__content {
  color: rgba(55, 53, 47, 0.82);
  font-size: 16px;
  line-height: 1.8;
  overflow-wrap: anywhere;
}

.memory-file-note__content :deep(h1),
.memory-file-note__content :deep(h2),
.memory-file-note__content :deep(h3),
.memory-file-note__content :deep(h4) {
  margin: 14px 0 6px;
  color: #37352f;
  font-weight: 600;
  line-height: 1.45;
}

.memory-file-note__content :deep(h1) {
  font-size: 22px;
}

.memory-file-note__content :deep(h2) {
  font-size: 20px;
}

.memory-file-note__content :deep(h3),
.memory-file-note__content :deep(h4) {
  font-size: 18px;
}

.memory-file-note__content :deep(p) {
  margin: 0 0 10px;
}

.memory-file-note__content :deep(p:last-child) {
  margin-bottom: 0;
}

.memory-file-note__content :deep(ul),
.memory-file-note__content :deep(ol) {
  margin: 0 0 10px;
  padding-left: 22px;
}

.memory-file-note__content :deep(blockquote) {
  margin: 10px 0;
  padding-left: 12px;
  border-left: 3px solid rgba(46, 170, 220, 0.45);
  color: rgba(55, 53, 47, 0.66);
}

.memory-file-note__content :deep(pre) {
  margin: 10px 0;
  padding: 10px 12px;
  overflow-x: auto;
  border-radius: 6px;
  background: rgba(55, 53, 47, 0.06);
  font-size: 12px;
  line-height: 1.6;
}

.memory-file-note__content :deep(table) {
  width: 100%;
  margin: 10px 0;
  border-collapse: collapse;
  font-size: 13px;
}

.memory-file-note__content :deep(th),
.memory-file-note__content :deep(td) {
  padding: 7px 9px;
  border: 1px solid rgba(55, 53, 47, 0.12);
  text-align: left;
  vertical-align: top;
}

.memory-file-note__content :deep(th) {
  background: rgba(55, 53, 47, 0.04);
  font-weight: 600;
}

.memory-file-note__empty,
.memory-file-note__state {
  margin: 0;
  color: rgba(55, 53, 47, 0.52);
  font-size: 15px;
  line-height: 1.7;
}

.memory-file-note--failed .memory-file-note__state,
.memory-file-note--skipped .memory-file-note__state {
  color: #c9372c;
}

.memory-note-organize-loading {
  display: flex;
  align-items: center;
  min-height: 124px;
  padding-top: 4px;
}

.memory-note-organize-card {
  cursor: pointer;
}

.organize-result-card {
  display: grid;
  grid-template-columns: 48px minmax(0, 1fr);
  min-height: 124px;
  overflow: hidden;
  border: 1px solid rgba(55, 53, 47, 0.12);
  border-radius: 8px;
  background: #fff;
  box-shadow: 0 4px 14px rgba(55, 53, 47, 0.06);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover,
  &:focus {
    border-color: rgba(46, 170, 220, 0.42);
    box-shadow: 0 10px 26px rgba(55, 53, 47, 0.1);
    outline: none;
  }
}

.organize-result-gutter {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  border-right: 1px solid rgba(55, 53, 47, 0.08);
  background: rgba(46, 170, 220, 0.08);
  color: #2eaadc;
}

.organize-result-main {
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

.organize-result-title-row {
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

.organize-result-intro {
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

.organize-result-chips {
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

.organize-result-meta {
  padding-top: 6px;
}

.organize-result-meta-separator {
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

.type-badge.organize-stage--formed {
  background: rgba(34, 101, 73, 0.1);
  color: #236549;
}

.type-badge.organize-stage--expandable {
  background: rgba(146, 94, 28, 0.1);
  color: #7a4d18;
}

.type-badge.organize-stage--organizing {
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

  .memory-note-attachment-cards {
    grid-template-columns: 1fr;
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

  .memory-file-notes {
    padding-top: 20px;
  }

  .memory-file-notes__header {
    flex-direction: column;
    gap: 8px;
  }

  .memory-file-notes__status {
    align-self: flex-start;
  }

  .memory-file-note {
    padding: 14px 14px 16px;
  }

  .memory-file-note__header {
    align-items: flex-start;
  }

  .memory-file-note__retry {
    padding-inline: 7px;
  }

  .organize-result-card {
    grid-template-columns: 1fr;
  }

  .organize-result-gutter {
    min-height: 72px;
    border-right: 0;
    border-bottom: 1px solid rgba(55, 53, 47, 0.08);
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
