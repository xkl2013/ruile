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
                <div class="memory-note-meta">
                  <span>{{ memoryAssetLabel }}</span>
                  <span>{{ memorySource }}</span>
                </div>
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
                    {{ tag }}
                    <button type="button" class="memory-note-tag-remove" :aria-label="`移除 ${tag}`" @click="removeMemoryTag(tag)">
                      <t-icon name="close" />
                    </button>
                  </t-tag>
                  <span v-if="!memoryTags.length" class="memory-note-tag-empty">还没有标签</span>
                </div>

                <div class="memory-note-tag-actions">
                  <t-popup
                    v-model:visible="noteTagMenuVisible"
                    trigger="click"
                    placement="bottom-left"
                    destroy-on-close
                    overlayClassName="memory-note-tag-popup"
                  >
                    <t-button variant="outline" theme="default" size="small">
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

                  <t-button variant="outline" theme="primary" size="small" @click="generateMemoryTags">+ 智能标签</t-button>
                  <t-button
                    theme="default"
                    variant="text"
                    size="small"
                    class="memory-note-sprout-action"
                    :loading="noteSproutCreating"
                    @click="createMemorySprout"
                  >
                    <template #icon><OrganizeSproutIcon class="memory-note-sprout-icon" /></template>
                    发芽
                  </t-button>
                </div>
              </div>

              <div class="memory-note-toolbar" aria-label="笔记格式工具">
                <div class="memory-note-toolbar-group">
                  <span class="memory-note-toolbar-label">字号</span>
                  <t-select
                    v-model="noteToolbarFontSize"
                    class="memory-note-font-select"
                    size="small"
                    clearable
                    :options="noteFontSizeOptions"
                    placeholder="默认"
                    @change="handleMemoryFontSizeChange"
                  />
                </div>
                <div class="memory-note-toolbar-group memory-note-toolbar-group--color">
                  <span class="memory-note-toolbar-label">颜色</span>
                  <div class="memory-note-color-row">
                    <button
                      v-for="color in noteColorOptions"
                      :key="color"
                      type="button"
                      class="memory-note-color-swatch"
                      :class="{ 'is-active': noteToolbarColor === color }"
                      :aria-label="color"
                      :title="color"
                      @click="applyMemoryTextColor(color)"
                    >
                      <span :style="{ backgroundColor: color }"></span>
                    </button>
                    <button type="button" class="memory-note-color-reset" @click="applyMemoryTextColor('')">清除</button>
                  </div>
                </div>
              </div>

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
                  :class="{ 'is-active': noteActiveTab === 'append' }"
                  @click="noteActiveTab = 'append'"
                >
                  追加笔记
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

                <section v-show="noteActiveTab === 'append'" class="memory-note-panel-view memory-note-panel-view--append">
                  <div class="memory-note-section-head">
                    <h3>追加笔记</h3>
                    <p>补充内容会追加到正文末尾。</p>
                  </div>
                  <t-textarea
                    v-model="appendNoteText"
                    class="memory-note-append-input"
                    :autosize="{ minRows: 6, maxRows: 12 }"
                    placeholder="在这里补充新内容，支持换行"
                  />
                  <div class="memory-note-panel-actions">
                    <t-button theme="primary" :disabled="!appendNoteText.trim()" @click="appendMemoryText">追加到正文</t-button>
                  </div>
                </section>

                <section v-show="noteActiveTab === 'sprout'" class="memory-note-panel-view memory-note-panel-view--sprout">
                  <div class="memory-note-section-head memory-note-section-head--sprout">
                    <div>
                      <h3>发芽</h3>
                      <p>把这条笔记整理成可继续复盘的发芽结果。</p>
                    </div>
                    <t-button theme="primary" :loading="noteSproutCreating" @click="createMemorySprout">
                      <template #icon><OrganizeSproutIcon class="memory-note-sprout-icon" /></template>
                      发芽
                    </t-button>
                  </div>
                  <div v-if="noteSproutReport" class="memory-note-sprout-result">
                    <div class="memory-note-sprout-meta">
                      <t-tag size="small" variant="light">{{ noteSproutStageLabel(noteSproutStatus || noteSproutReport.stage) }}</t-tag>
                      <span>{{ noteSproutReport.memory_count ?? 1 }} 条素材</span>
                    </div>
                    <h3>{{ noteSproutReport.title }}</h3>
                    <div class="memory-note-sprout-body" v-html="sproutReportContentForEditor(noteSproutReport.summary)"></div>
                  </div>
                  <div v-else class="memory-note-empty-state">
                    还没有发芽结果，点击按钮开始生成。
                  </div>
                </section>
              </div>
            </section>
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
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { TiptapProEditor, type FeatureConfig, type TiptapProEditorExpose } from 'tiptap-ui-kit'
import 'tiptap-ui-kit/style.css'
import {
  createOrganizeMemory,
  createOrganizeSproutReportFromMemory,
  createOrganizeOutput,
  createOrganizeSproutReport,
  getOrganizeMemory,
  getOrganizeOutput,
  getOrganizeSproutReport,
  updateOrganizeMemory,
  updateOrganizeOutput,
  updateOrganizeSproutReport,
  type OrganizeMemory,
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
import { sproutReportContentForEditor } from './sproutReport'
import {
  buildSmartNoteTags,
  mergeNoteMetadata,
  normalizeNoteTags,
} from './noteEditor'

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
const appendNoteText = ref('')
const noteTagDraft = ref('')
const noteTagMenuVisible = ref(false)
const noteSproutReport = ref<OrganizeSproutReport | null>(null)
const noteSproutCreating = ref(false)
const noteSproutStatus = ref<OrganizeSproutStage | ''>('')
const noteDefaultFontSize = '14px'
const noteToolbarFontSize = ref(noteDefaultFontSize)
const noteToolbarColor = ref('#37352f')
const noteActiveTab = ref<'content' | 'append' | 'sprout'>('content')
const outputDraft = ref<OrganizeOutput | null>(null)
const sproutDraft = ref<OrganizeSproutReport | null>(null)
const noteFontSizeOptions = [
  { label: '12px', value: '12px' },
  { label: '13px', value: '13px' },
  { label: '14px', value: '14px' },
  { label: '16px', value: '16px' },
  { label: '18px', value: '18px' },
  { label: '20px', value: '20px' },
]
const noteColorOptions = [
  '#37352f',
  '#787774',
  '#9b9a97',
  '#9c36b5',
  '#5f3dc4',
  '#2f9e44',
  '#1971c2',
  '#d9480f',
  '#c92a2a',
]

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
  if (isNoteMemory.value) return '输入正文，支持字号和颜色'
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

const applyMemoryFontSize = (fontSize: string) => {
  const editor = currentEditor()
  if (!editor) return
  const chain = editor.chain().focus() as any
  if (!fontSize) {
    chain.unsetFontSize().run()
  } else {
    chain.setFontSize(fontSize).run()
  }
  noteToolbarFontSize.value = fontSize || noteDefaultFontSize
}

const handleMemoryFontSizeChange = (fontSize: unknown) => {
  if (typeof fontSize === 'string' || typeof fontSize === 'number') {
    applyMemoryFontSize(String(fontSize))
    return
  }
  applyMemoryFontSize('')
}

const applyMemoryTextColor = (color: string) => {
  const editor = currentEditor()
  if (!editor) return
  const chain = editor.chain().focus() as any
  if (!color) {
    chain.unsetColor().run()
  } else {
    chain.setColor(color).run()
  }
  noteToolbarColor.value = color || '#37352f'
}

const appendMemoryText = () => {
  const text = appendNoteText.value.trim()
  if (!text) return
  const editor = currentEditor()
  if (!editor) return
  const html = text
    .split(/\n+/)
    .map((line) => `<p>${escapeHtml(line)}</p>`)
    .join('')
  editor.chain().focus('end').insertContent(html).run()
  appendNoteText.value = ''
  noteActiveTab.value = 'content'
}

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
  appendNoteText.value = ''
  noteTagDraft.value = ''
  noteTagMenuVisible.value = false
  noteSproutReport.value = null
  noteSproutStatus.value = ''
  noteToolbarFontSize.value = noteDefaultFontSize
  noteToolbarColor.value = '#37352f'
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
      appendNoteText.value = ''
      noteTagDraft.value = ''
      noteTagMenuVisible.value = false
      noteSproutReport.value = null
      noteSproutStatus.value = ''
      noteToolbarFontSize.value = noteDefaultFontSize
      noteToolbarColor.value = '#37352f'
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
    } else if (documentType.value === 'output') {
      const response = await getOrganizeOutput(documentId.value)
      if (!response.success || !response.data) throw new Error(response.message || '发现加载失败')
      const item = response.data
      title.value = item.title
      content.value = normalizeDocumentContent(item.title, item.content)
      outputDraft.value = item
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
      savedId = response.data.id
    } else if (currentType === 'output') {
      const draft = outputDraft.value
      const input = {
        title: normalizedTitle,
        content: html,
        output_type: draft?.output_type || readQuery('output_type') || '研究文档',
        source_summary: draft?.source_summary || readQuery('source_summary') || '手动创建',
        status: draft?.status || (readQuery('status') as OrganizeOutputStatus) || 'draft',
        icon: draft?.icon || readQuery('icon') || 'file-word',
        memory_ids: draft?.memory_ids || [],
        metadata: draft?.metadata,
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
  padding-top: 42px;
  box-sizing: border-box;
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
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
}

.memory-note-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  color: rgba(55, 53, 47, 0.52);
  font-size: 12px;
  line-height: 18px;
}

.memory-note-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.memory-note-tag-list {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.memory-note-tag-empty {
  color: rgba(55, 53, 47, 0.36);
  font-size: 12px;
  line-height: 18px;
}

.memory-note-tag-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}

.memory-note-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border-radius: 999px;
}

.memory-note-tag-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.memory-note-tag-remove:hover {
  background: rgba(55, 53, 47, 0.08);
}

.memory-note-sprout-action {
  padding-inline: 6px;
  color: rgba(55, 53, 47, 0.72);
}

.memory-note-sprout-icon {
  font-size: 14px;
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

.memory-note-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 18px 24px;
  padding-top: 6px;
  border-top: 1px solid rgba(55, 53, 47, 0.08);
}

.memory-note-toolbar-group {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  min-height: 34px;
}

.memory-note-toolbar-label {
  color: rgba(55, 53, 47, 0.56);
  font-size: 12px;
  line-height: 18px;
}

.memory-note-font-select {
  width: 104px;
}

.memory-note-color-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.memory-note-color-swatch {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  margin: 0;
  padding: 0;
  border: 1px solid rgba(55, 53, 47, 0.12);
  border-radius: 50%;
  background: #fff;
  cursor: pointer;
  box-sizing: border-box;
}

.memory-note-color-swatch span {
  width: 16px;
  height: 16px;
  border-radius: 50%;
}

.memory-note-color-swatch.is-active {
  box-shadow: 0 0 0 2px rgba(46, 170, 220, 0.18);
}

.memory-note-color-reset {
  border: 0;
  background: transparent;
  color: rgba(55, 53, 47, 0.58);
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}

.memory-note-color-reset:hover {
  color: #37352f;
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

.memory-note-section-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.memory-note-section-head h3 {
  margin: 0;
  color: #37352f;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.3;
}

.memory-note-section-head p {
  margin: 0;
  color: rgba(55, 53, 47, 0.56);
  font-size: 13px;
  line-height: 1.6;
}

.memory-note-section-head--sprout {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.memory-note-append-input {
  width: 100%;
}

.memory-note-append-input :deep(.t-textarea__inner) {
  min-height: 180px;
  border-color: rgba(55, 53, 47, 0.12);
  border-radius: 10px;
  resize: vertical;
  box-shadow: none;
}

.memory-note-panel-actions {
  display: flex;
  justify-content: flex-end;
}

.memory-note-empty-state {
  padding: 16px 0 4px;
  color: rgba(55, 53, 47, 0.48);
  font-size: 13px;
  line-height: 1.6;
}

.memory-note-sprout-result {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 4px;
  border-top: 1px solid rgba(55, 53, 47, 0.08);
}

.memory-note-sprout-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  color: rgba(55, 53, 47, 0.52);
  font-size: 12px;
  line-height: 18px;
}

.memory-note-sprout-result h3 {
  margin: 0;
  color: #37352f;
  font-size: 16px;
  font-weight: 600;
  line-height: 1.35;
}

.memory-note-sprout-body {
  max-height: 420px;
  overflow: auto;
  color: #37352f;
  font-size: 13px;
  line-height: 1.65;
}

.memory-note-sprout-body :deep(h1),
.memory-note-sprout-body :deep(h2),
.memory-note-sprout-body :deep(h3),
.memory-note-sprout-body :deep(p),
.memory-note-sprout-body :deep(ul),
.memory-note-sprout-body :deep(ol) {
  margin-top: 0;
}

.memory-note-sprout-body :deep(h1) {
  margin-bottom: 10px;
  font-size: 18px;
  line-height: 1.35;
}

.memory-note-sprout-body :deep(h2) {
  margin-bottom: 8px;
  font-size: 16px;
  line-height: 1.4;
}

.memory-note-sprout-body :deep(p),
.memory-note-sprout-body :deep(ul),
.memory-note-sprout-body :deep(ol) {
  margin-bottom: 10px;
}

.memory-note-sprout-body :deep(li) {
  margin-bottom: 6px;
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
  --editor-empty-placeholder: "输入正文，支持字号和颜色";
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
  }

  .organize-editor-main {
    padding: 0 18px 48px;
  }

  .document-page {
    padding-top: 28px;
  }

  .memory-note-panel {
    gap: 14px;
  }

  .memory-note-title-input :deep(.t-input__inner) {
    font-size: 24px;
  }

  .memory-note-tags {
    flex-direction: column;
  }

  .memory-note-tag-actions {
    width: 100%;
    justify-content: flex-start;
  }

  .memory-note-toolbar {
    gap: 12px 16px;
  }

  .memory-note-tabs {
    gap: 16px;
  }

  .memory-note-tab {
    padding-bottom: 12px;
    font-size: 14px;
  }

  .memory-note-section-head--sprout {
    flex-direction: column;
    align-items: flex-start;
  }

  .memory-note-sprout-body {
    max-height: 300px;
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
