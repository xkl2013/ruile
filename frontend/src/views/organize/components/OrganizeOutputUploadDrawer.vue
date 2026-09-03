<template>
  <t-drawer
    v-model:visible="visibleProxy"
    class="output-upload-drawer"
    :header="false"
    :footer="false"
    :close-btn="false"
    :size="'min(760px, 92vw)'"
    attach="body"
    placement="right"
  >
    <div class="output-upload-shell">
      <div class="output-upload-header">
        <div>
          <div class="output-upload-eyebrow">{{ drawerEyebrow }}</div>
          <div class="output-upload-title">{{ step === 'review' ? '确认 AI 结果' : '选择文件' }}</div>
          <div class="output-upload-desc">{{ drawerDescription }}</div>
        </div>
        <t-button variant="text" theme="default" shape="square" @click="close">
          <template #icon><t-icon name="close" /></template>
        </t-button>
      </div>

      <div v-if="step === 'pick'" class="output-upload-body">
        <button type="button" class="upload-dropzone" @click="openPicker">
          <t-icon name="upload" size="28px" />
          <strong>{{ dropzoneTitle }}</strong>
          <span>{{ acceptLabel }}</span>
        </button>
        <div class="upload-hint-row">
          <span>摘要和标签会在上传后自动生成</span>
          <span>AI 会自动识别分类，确认前可修改</span>
        </div>
      </div>

      <div v-else-if="step === 'processing'" class="output-upload-body output-upload-body--center">
        <t-loading size="small" />
        <div class="processing-text">AI 正在提取摘要和标签</div>
        <div class="processing-file">{{ currentFileName }}</div>
      </div>

      <div v-else class="output-upload-body">
        <div class="upload-file-summary">
          <div class="upload-file-name">{{ currentFileName }}</div>
          <div class="upload-file-meta">
            <t-tag size="small" variant="light">{{ kindLabel }}</t-tag>
            <t-tag size="small" theme="default" variant="light-outline">{{ statusLabel }}</t-tag>
            <span>{{ sourceLabel }}</span>
          </div>
        </div>

        <div class="upload-form-grid">
          <label class="upload-field upload-field--full">
            <span>标题</span>
            <t-input v-model="title" placeholder="请输入标题" />
          </label>

          <label class="upload-field upload-field--full">
            <span>摘要</span>
            <t-textarea v-model="summary" :autosize="{ minRows: 4, maxRows: 8 }" placeholder="摘要会自动生成，也可以继续调整" />
          </label>

          <label class="upload-field">
            <span>类型</span>
            <t-select v-model="kind" :options="kindOptions" />
          </label>

          <label class="upload-field">
            <span>栏目</span>
            <t-select v-model="category" :options="categoryOptions" placeholder="请选择发现栏目" />
          </label>

          <label class="upload-field">
            <span>状态</span>
            <t-select v-model="status" :options="statusOptions" />
          </label>

          <div class="upload-field upload-field--full">
            <span>标签</span>
            <div class="tag-editor">
              <t-tag
                v-for="tag in tags"
                :key="tag"
                class="tag-chip"
                size="small"
                variant="light"
              >
                {{ tag }}
                <button type="button" class="tag-remove" :aria-label="`移除 ${tag}`" @click="removeTag(tag)">
                  <t-icon name="close" />
                </button>
              </t-tag>
              <t-input
                v-model="tagInput"
                class="tag-input"
                placeholder="输入标签后回车"
                @keydown.enter.prevent="addTag"
              />
            </div>
          </div>

          <div class="upload-field upload-field--full">
            <span>AI 提示</span>
            <div class="upload-note">
              <span>上传完成后已生成摘要和标签</span>
              <span>确认发布后会进入发现广场</span>
            </div>
          </div>
        </div>
      </div>

      <div class="output-upload-footer">
        <t-button theme="default" variant="outline" @click="handleSecondary">
          {{ secondaryLabel }}
        </t-button>
        <t-button theme="primary" :loading="uploading || saving" :disabled="step === 'processing'" @click="handlePrimary">
          {{ primaryLabel }}
        </t-button>
      </div>
    </div>

    <input
      ref="fileInputRef"
      type="file"
      class="hidden-file-input"
      :accept="accept"
      @change="handleFileChange"
    />
  </t-drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  type OrganizeOutput,
  type OrganizeOutputInput,
  type OrganizeOutputStatus,
  updateOrganizeOutput,
  uploadOrganizeOutput,
} from '@/api/organize'
import {
  DISCOVER_CATEGORY_OPTIONS,
  discoverCategoryLabel,
  normalizeDiscoverCategory,
  type DiscoverCategoryKey,
} from '../discoverCategories'

type OutputKind = 'article' | 'video' | 'audio'
type UploadStep = 'pick' | 'processing' | 'review'

interface OutputTagMetadata {
  tags?: string[] | string | null
  content_kind?: string
  content_kind_label?: string
  file_name?: string
  file_path?: string
  file_type?: string
  mime_type?: string
  ai_status?: string
  ai_model_id?: string
  asr_model_id?: string
  transcript?: string
}

const props = defineProps<{
  visible: boolean
  initialKind?: OutputKind
}>()

const emit = defineEmits<{
  'update:visible': [visible: boolean]
  uploaded: [item: OrganizeOutput]
  saved: [item: OrganizeOutput]
}>()

const visibleProxy = computed({
  get: () => props.visible,
  set: (value: boolean) => emit('update:visible', value),
})

const acceptByKind: Record<OutputKind, string> = {
  article: '.doc,.docx,.pdf,.md,.markdown,.txt,.ppt,.pptx',
  video: '.mp4,.mov,.avi,.mkv,.webm,.wmv,.flv',
  audio: '.mp3,.m4a,.wav,.flac,.ogg,.aac',
}
const kindOptions = [
  { label: '图文类', value: 'article' },
  { label: '视频类', value: 'video' },
  { label: '音频类', value: 'audio' },
]
const statusOptions = [
  { label: '待确认', value: 'review' },
  { label: '草稿', value: 'draft' },
  { label: '已发布', value: 'ready' },
]
const categoryOptions = DISCOVER_CATEGORY_OPTIONS

const fileInputRef = ref<HTMLInputElement | null>(null)
const step = ref<UploadStep>('pick')
const uploading = ref(false)
const saving = ref(false)
const error = ref('')
const currentFile = ref<File | null>(null)
const draft = ref<OrganizeOutput | null>(null)
const title = ref('')
const summary = ref('')
const kind = ref<OutputKind>('article')
const status = ref<OrganizeOutputStatus>('review')
const category = ref<DiscoverCategoryKey | ''>('')
const tags = ref<string[]>([])
const tagInput = ref('')

const kindLabelMap: Record<OutputKind, string> = {
  article: '图文类',
  video: '视频类',
  audio: '音频类',
}

const statusLabelMap: Record<OrganizeOutputStatus, string> = {
  draft: '草稿',
  review: '待确认',
  ready: '已发布',
  archived: '已归档',
}

const currentFileName = computed(() => currentFile.value?.name || '未选择文件')
const kindLabel = computed(() => kindLabelMap[kind.value] || '图文类')
const statusLabel = computed(() => statusLabelMap[status.value] || '待确认')
const sourceLabel = computed(() => draft.value?.metadata?.file_path ? '已上传并解析' : '等待上传')
const accept = computed(() => acceptByKind[props.initialKind || 'article'] || Object.values(acceptByKind).join(','))
const drawerEyebrow = computed(() => `新建${kindLabelMap[props.initialKind || 'article'] || '发现'}`)
const drawerDescription = computed(() => `${kindLabelMap[props.initialKind || 'article'] || '发现'}上传后自动提取摘要、生成标签并识别分类。`)
const dropzoneTitle = computed(() => `选择${kindLabelMap[props.initialKind || 'article'] || '发现'}文件`)
const acceptLabel = computed(() => {
  if (props.initialKind === 'video') return '支持 MP4、MOV、AVI、MKV、WebM 等视频文件'
  if (props.initialKind === 'audio') return '支持 MP3、M4A、WAV、FLAC、OGG 等音频文件'
  return '支持 Word、PDF、Markdown、TXT、PPT 等图文文件'
})
const primaryLabel = computed(() => {
  if (step.value === 'pick') return '选择文件'
  if (step.value === 'processing') return '处理中'
  if (status.value === 'draft') return '保存草稿'
  if (status.value === 'review') return '保存待确认'
  return '确认发布'
})
const secondaryLabel = computed(() => {
  if (step.value === 'review') return '重新选择'
  return '关闭'
})

const resetState = () => {
  step.value = 'pick'
  uploading.value = false
  saving.value = false
  error.value = ''
  currentFile.value = null
  draft.value = null
  title.value = ''
  summary.value = ''
  kind.value = props.initialKind || 'article'
  status.value = 'review'
  category.value = ''
  tags.value = []
  tagInput.value = ''
}

const close = () => {
  visibleProxy.value = false
}

const openPicker = () => {
  fileInputRef.value?.click()
}

const extractKind = (item: OrganizeOutput): OutputKind => {
  const metadata = (item.metadata || {}) as OutputTagMetadata
  const raw = String(metadata.content_kind || item.output_type || '').trim().toLowerCase()
  if (raw === 'video' || raw === '视频类') return 'video'
  if (raw === 'audio' || raw === '音频类') return 'audio'
  return 'article'
}

const extractTags = (item: OrganizeOutput) => {
  const metadata = (item.metadata || {}) as OutputTagMetadata
  const raw = metadata.tags
  if (Array.isArray(raw)) {
    return raw.map((tag) => String(tag).trim()).filter(Boolean)
  }
  if (typeof raw === 'string' && raw.trim()) {
    return raw.split(/[，,;；\n]/).map((tag) => tag.trim()).filter(Boolean)
  }
  return []
}

const hydrateDraft = (item: OrganizeOutput) => {
  draft.value = item
  title.value = item.title || ''
  summary.value = item.source_summary || ''
  kind.value = extractKind(item)
  status.value = item.status || 'review'
  category.value = normalizeDiscoverCategory(item.metadata?.discover_category || item.metadata?.discover_category_label)
  tags.value = extractTags(item)
}

const addTag = () => {
  const next = tagInput.value.trim()
  if (!next) return
  if (!tags.value.some((tag) => tag.toLowerCase() === next.toLowerCase())) {
    tags.value = [...tags.value, next].slice(0, 8)
  }
  tagInput.value = ''
}

const removeTag = (tag: string) => {
  tags.value = tags.value.filter((item) => item !== tag)
}

const uploadFile = async (file: File) => {
  currentFile.value = file
  uploading.value = true
  error.value = ''
  step.value = 'processing'
  try {
    const response = await uploadOrganizeOutput(file)
    if (!response.success || !response.data) {
      throw new Error(response.message || '上传失败')
    }
    hydrateDraft(response.data)
    emit('uploaded', response.data)
    step.value = 'review'
  } catch (err: any) {
    error.value = err?.message || '上传失败'
    MessagePlugin.error(error.value)
    step.value = 'pick'
  } finally {
    uploading.value = false
  }
}

const handleFileChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  currentFile.value = file
  await uploadFile(file)
}

const buildMetadata = () => {
  const current = draft.value?.metadata || {}
  return {
    ...current,
    content_kind: kind.value,
    content_kind_label: kindLabel.value,
    discover_category: category.value,
    discover_category_label: discoverCategoryLabel(category.value),
    tags: [...tags.value],
    ai_status: 'confirmed',
  }
}

const handlePrimary = async () => {
  if (step.value === 'pick') {
    openPicker()
    return
  }
  if (!draft.value) return
  if (!category.value) {
    MessagePlugin.warning('请选择发现栏目')
    return
  }
  saving.value = true
  try {
    const input: OrganizeOutputInput = {
      title: title.value.trim() || draft.value.title,
      content: draft.value.content,
      output_type: kindLabel.value,
      source_summary: summary.value.trim() || draft.value.source_summary || '',
      status: status.value,
      icon: draft.value.icon || (kind.value === 'video' ? 'play-circle' : kind.value === 'audio' ? 'sound' : 'file-word'),
      memory_ids: draft.value.memory_ids || [],
      metadata: buildMetadata(),
    }
    const response = await updateOrganizeOutput(draft.value.id, input)
    if (!response.success || !response.data) {
      throw new Error(response.message || '发布失败')
    }
    emit('saved', response.data)
    close()
  } catch (err: any) {
    MessagePlugin.error(err?.message || '发布失败')
  } finally {
    saving.value = false
  }
}

const handleSecondary = () => {
  if (step.value === 'review') {
    step.value = 'pick'
    draft.value = null
    currentFile.value = null
    tags.value = []
    tagInput.value = ''
    title.value = ''
    summary.value = ''
    kind.value = props.initialKind || 'article'
    status.value = 'review'
    category.value = ''
    return
  }
  close()
}

watch(
  () => props.visible,
  (visible) => {
    if (visible) {
      resetState()
    } else {
      resetState()
    }
  },
)
</script>

<style scoped lang="less">
.output-upload-shell {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 18px 20px 20px;
  box-sizing: border-box;
}

.output-upload-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 18px;
}

.output-upload-eyebrow {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.output-upload-title {
  margin-top: 4px;
  color: var(--td-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
}

.output-upload-desc {
  margin-top: 4px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.output-upload-body {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.output-upload-body--center {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  min-height: 320px;
  color: var(--td-text-color-secondary);
}

.upload-dropzone {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-height: 260px;
  padding: 24px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;

  strong {
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 600;
  }

  span {
    font-size: 13px;
    line-height: 20px;
  }

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container-hover);
  }
}

.upload-hint-row {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-top: 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.upload-file-summary {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.upload-file-name {
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
}

.upload-file-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.upload-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 12px;
}

.upload-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;

  > span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.upload-field--full {
  grid-column: 1 / -1;
}

.tag-editor {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.tag-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.tag-remove {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.tag-input {
  flex: 1 1 180px;
  min-width: 140px;
}

.upload-note {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 14px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.output-upload-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
}

.processing-text {
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
}

.processing-file {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.hidden-file-input {
  display: none;
}

@media (max-width: 640px) {
  .upload-form-grid {
    grid-template-columns: 1fr;
  }

  .upload-hint-row,
  .output-upload-footer {
    flex-direction: column;
    align-items: stretch;
  }

  .output-upload-footer {
    margin-top: 12px;
  }
}
</style>
