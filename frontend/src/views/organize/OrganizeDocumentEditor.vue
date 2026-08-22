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
              version="basic"
              theme-preset="notion"
              locale="zh-CN"
              :placeholder="editorPlaceholder"
              :features="editorFeatures"
            />
          </div>
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
import {
  clearOrganizeEditorDraft,
  readOrganizeEditorDraft,
  type OrganizeEditorDraft,
} from './editorDraftStorage'
import { sproutReportContentForEditor } from './sproutReport'

type OrganizeDocumentType = 'memory' | 'output' | 'sprout'
type SaveState = 'idle' | 'saving' | 'saved' | 'waiting' | 'error'

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
const outputDraft = ref<OrganizeOutput | null>(null)
const sproutDraft = ref<OrganizeSproutReport | null>(null)

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
const isCreate = computed(() => !savedDocumentId.value && (documentId.value === 'new' || documentId.value.startsWith('demo-')))

const memoryAssetLabel = computed(() => {
  if (memoryKind.value === 'audio') return '录音'
  if (memoryKind.value === 'audio_card') return '工牌'
  return '笔记'
})

const isAudioMemory = computed(() => documentType.value === 'memory' && memoryKind.value === 'audio')
const audioDurationLabel = computed(() => formatDuration(memoryDurationSeconds.value || 0))
const editorPlaceholder = computed(() => isAudioMemory.value ? '录音转写内容' : '输入内容，或按“/”启用命令')

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

const extractTitleFromContent = (html = '') => {
  const firstBlock = Array.from(parseHtmlBody(html).children)[0]
  if (firstBlock?.tagName.toLowerCase() !== 'h1') return ''
  return normalizeTitle(firstBlock.textContent || '')
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
  if (documentType.value === 'memory' && memoryKind.value === 'audio') {
    content.value = normalizeAudioMemoryContent(draftContent)
    title.value = draftTitle || audioMemoryFallbackTitle(content.value)
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
      title.value = item.kind === 'audio' ? audioMemoryDisplayTitle(item) : item.title
      content.value = item.kind === 'audio'
        ? normalizeAudioMemoryContent(audioMemoryContentSource(item))
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
        metadata: memoryMetadata.value,
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
  if (isAudioMemory.value) return
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
  if (!editorReady.value || loading.value || !isAudioMemory.value) return
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
  content: "无标题";
  float: left;
  height: 0;
  color: rgba(55, 53, 47, 0.2);
  pointer-events: none;
}

.document-editor-shell :deep(.ProseMirror p.is-empty::before) {
  content: "输入内容，或按“/”启用命令";
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

  .document-editor-shell :deep(.word-content-multi .ProseMirror) {
    min-height: calc(100vh - 190px);
    padding-bottom: 80px;
  }
}
</style>
