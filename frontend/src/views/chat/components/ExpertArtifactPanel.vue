<template>
  <aside class="expert-artifact-panel" aria-label="专家产物预览">
    <header class="expert-artifact-panel__header">
      <div class="expert-artifact-panel__title-wrap">
        <div class="expert-artifact-panel__file-icon">{{ fileTypeLabel }}</div>
        <div class="expert-artifact-panel__title-copy">
          <strong>{{ artifact?.title || '专家产物' }}</strong>
          <small>{{ artifactFileName }}</small>
        </div>
      </div>
      <div class="expert-artifact-panel__header-actions">
        <button
          type="button"
          class="expert-artifact-panel__download"
          :disabled="downloading"
          title="下载产物"
          @click="downloadArtifact"
        >
          <t-icon :name="downloading ? 'loading' : 'download'" />
        </button>
        <button type="button" class="expert-artifact-panel__close" title="关闭预览" @click="emit('close')">
          <t-icon name="close" />
        </button>
      </div>
    </header>

    <div class="expert-artifact-panel__body">
      <div v-if="previewError" class="expert-artifact-panel__error" role="alert">
        <t-icon name="error-circle" />
        <div>
          <strong>产物预览失败</strong>
          <span>{{ previewError }}</span>
        </div>
        <button type="button" @click="loadPreview">重新加载</button>
      </div>

      <div v-if="loadingPreview" class="expert-artifact-panel__loading">
        正在加载产物...
      </div>

      <template v-else-if="previewKind === 'html' && textContent">
        <div class="expert-artifact-panel__badge">静态 HTML 文件 · 服务端预览</div>
        <iframe class="expert-artifact-panel__html" :srcdoc="textContent" sandbox="allow-same-origin" />
      </template>

      <template v-else-if="previewKind === 'image' && previewUrl">
        <div class="expert-artifact-panel__badge">图片产物</div>
        <img class="expert-artifact-panel__image" :src="previewUrl" :alt="artifact?.title || '专家图片产物'" />
      </template>

      <template v-else-if="previewKind === 'pdf' && previewUrl">
        <div class="expert-artifact-panel__badge">PDF 文档</div>
        <iframe class="expert-artifact-panel__document" :src="previewUrl" title="PDF 产物预览" />
      </template>

      <template v-else-if="previewKind === 'audio' && previewUrl">
        <div class="expert-artifact-panel__badge">音频产物</div>
        <audio class="expert-artifact-panel__media" :src="previewUrl" controls />
      </template>

      <template v-else-if="previewKind === 'video' && previewUrl">
        <div class="expert-artifact-panel__badge">视频产物</div>
        <video class="expert-artifact-panel__media" :src="previewUrl" controls />
      </template>

      <template v-else-if="previewKind === 'text' && textContent">
        <div class="expert-artifact-panel__badge">文本产物</div>
        <div
          v-if="isMarkdown"
          class="expert-artifact-panel__markdown markdown-content"
          v-html="renderedMarkdown"
        >
        </div>
        <pre v-else class="expert-artifact-panel__text">{{ textContent }}</pre>
      </template>

      <template v-else-if="report">
        <div class="expert-artifact-panel__badge">结构化报告</div>
        <h2>{{ report.title }}</h2>
        <p class="expert-artifact-panel__summary">{{ report.executive_summary }}</p>

        <section v-for="section in report.sections || []" :key="`${section.type}-${section.title}`"
          class="expert-artifact-section">
          <h3>{{ section.title }}</h3>
          <p v-if="section.content">{{ section.content }}</p>
          <ul v-if="section.items?.length">
            <li v-for="item in section.items" :key="item">{{ item }}</li>
          </ul>
        </section>
      </template>

      <div v-else class="expert-artifact-panel__empty">
        当前产物暂不支持内嵌预览，请点击右上角下载。
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import {
  downloadAgentArtifact,
  getAgentArtifactPreview,
  getAgentArtifactPreviewBlob,
} from '@/api/agent-run';
import {
  createChatMarkdownRenderer,
  renderChatMarkdown,
} from '@/utils/chatMarkdownRenderer';
import {
  safeMarkdownToHTML,
  sanitizeMarkdownHTML,
} from '@/utils/security';

const props = defineProps({
  artifact: {
    type: Object,
    default: null,
  },
  runId: {
    type: String,
    default: '',
  },
});

const emit = defineEmits(['close']);
const textContent = ref('');
const previewUrl = ref('');
const loadingPreview = ref(false);
const downloading = ref(false);
const previewError = ref('');

const artifactFileName = computed(() => (
  props.artifact?.original_name
  || (props.artifact?.mime_type === 'text/html' ? 'expert-report.html' : '')
  || props.artifact?.format
  || props.artifact?.kind
  || 'structured report'
));

const fileTypeLabel = computed(() => {
  if (previewKind.value === 'html') return 'HTML';
  if (previewKind.value === 'image') return 'IMG';
  if (previewKind.value === 'pdf') return 'PDF';
  if (isMarkdown.value) return 'MD';
  if (previewKind.value === 'text') return 'TXT';
  return String(props.artifact?.format || props.artifact?.kind || 'FILE').slice(0, 4).toUpperCase();
});

const mimeType = computed(() => String(props.artifact?.mime_type || '').toLowerCase());
const isMarkdown = computed(() => (
  mimeType.value === 'text/markdown'
  || /\.md$/i.test(artifactFileName.value)
  || ['md', 'markdown'].includes(String(props.artifact?.format || '').toLowerCase())
));
const previewKind = computed(() => {
  if (mimeType.value === 'text/html' || /\.html?$/i.test(artifactFileName.value)) return 'html';
  if (mimeType.value.startsWith('image/')) return 'image';
  if (mimeType.value === 'application/pdf' || /\.pdf$/i.test(artifactFileName.value)) return 'pdf';
  if (mimeType.value.startsWith('audio/')) return 'audio';
  if (mimeType.value.startsWith('video/')) return 'video';
  if (
    mimeType.value.startsWith('text/')
    || ['application/json', 'application/xml', 'application/javascript'].includes(mimeType.value)
    || /\.(md|markdown|txt|json|csv|xml|log)$/i.test(artifactFileName.value)
  ) return 'text';
  return '';
});
const markdownRenderer = createChatMarkdownRenderer();
const renderedMarkdown = computed(() => renderChatMarkdown(textContent.value, {
  renderer: markdownRenderer,
  escapeMarkdown: safeMarkdownToHTML,
  sanitizeHtml: sanitizeMarkdownHTML,
}));

const report = computed(() => {
  const content = props.artifact?.content;
  if (!content || props.artifact?.kind !== 'report') return null;
  if (content.format !== 'structured_report_v1') return null;
  return content;
});

const inlineTextContent = computed(() => {
  const content = props.artifact?.content;
  if (typeof content === 'string') return content;
  return content?.html || content?.text || content?.markdown || '';
});

const revokePreviewUrl = () => {
  if (previewUrl.value) {
    URL.revokeObjectURL(previewUrl.value);
    previewUrl.value = '';
  }
};

const loadPreview = async () => {
  textContent.value = '';
  previewError.value = '';
  revokePreviewUrl();
  const runId = String(props.runId || '').trim();
  const artifactId = String(props.artifact?.id || '').trim();
  if (!runId || !artifactId || !props.artifact?.resource_ref) {
    textContent.value = inlineTextContent.value;
    return;
  }
  loadingPreview.value = true;
  try {
    if (previewKind.value === 'html' || previewKind.value === 'text') {
      textContent.value = await getAgentArtifactPreview(runId, artifactId);
    } else {
      const blob = await getAgentArtifactPreviewBlob(runId, artifactId);
      previewUrl.value = URL.createObjectURL(blob);
    }
  } catch (error) {
    console.warn('[ExpertArtifact] failed to load server preview:', error);
    previewError.value = '无法加载在线预览，可以重新加载或使用右上角下载按钮。';
    textContent.value = inlineTextContent.value;
  } finally {
    loadingPreview.value = false;
  }
};

const downloadArtifact = async () => {
  const runId = String(props.runId || '').trim();
  const artifactId = String(props.artifact?.id || '').trim();
  if (!runId || !artifactId || downloading.value) return;
  downloading.value = true;
  try {
    const blob = await downloadAgentArtifact(runId, artifactId);
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = artifactFileName.value || 'agent-artifact';
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(url);
  } catch (error) {
    console.warn('[ExpertArtifact] failed to download artifact:', error);
  } finally {
    downloading.value = false;
  }
};

watch(
  () => [props.runId, props.artifact?.id, props.artifact?.resource_ref],
  () => { void loadPreview(); },
  { immediate: true },
);

onBeforeUnmount(revokePreviewUrl);
</script>

<style scoped lang="less">
@import '../../../components/css/chat-markdown.less';

.expert-artifact-panel {
  position: fixed;
  z-index: 12000;
  top: 0;
  right: 0;
  display: flex;
  width: min(42vw, 640px);
  height: 100vh;
  flex-direction: column;
  border-left: 1px solid var(--td-component-border);
  background: var(--td-bg-color-container);
  box-shadow: -8px 0 24px rgba(0, 0, 0, .08);
}

.expert-artifact-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 72px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--td-component-border);
}

.expert-artifact-panel__title-wrap {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 10px;
}

.expert-artifact-panel__file-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 7px;
  color: #fff;
  background: #3f80bd;
  font-size: 10px;
  font-weight: 700;
}

.expert-artifact-panel__title-copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.expert-artifact-panel__title-copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expert-artifact-panel__title-copy small {
  color: var(--td-text-color-placeholder);
}

.expert-artifact-panel__header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.expert-artifact-panel__download {
  display: inline-flex;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  padding: 0;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-container);
  cursor: pointer;
  font-size: 12px;
}

.expert-artifact-panel__download:hover:not(:disabled) {
  border-color: var(--td-brand-color);
  color: var(--td-brand-color);
}

.expert-artifact-panel__download:disabled {
  cursor: wait;
  opacity: .6;
}

.expert-artifact-panel__close {
  display: inline-flex;
  width: 28px;
  height: 28px;
  align-items: center;
  justify-content: center;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 16px;
}

.expert-artifact-panel__body {
  flex: 1;
  min-height: 0;
  padding: 28px;
  overflow: auto;
}

.expert-artifact-panel__loading {
  display: flex;
  min-height: 120px;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
}

.expert-artifact-panel__error {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  margin-bottom: 18px;
  padding: 12px 14px;
  border: 1px solid var(--td-error-color-3);
  border-radius: 8px;
  color: var(--td-error-color);
  background: var(--td-error-color-light);
}

.expert-artifact-panel__error > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.expert-artifact-panel__error span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.expert-artifact-panel__error button {
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid var(--td-error-color-3);
  border-radius: 6px;
  color: var(--td-error-color);
  background: var(--td-bg-color-container);
  cursor: pointer;
}

.expert-artifact-panel__badge {
  display: inline-flex;
  margin-bottom: 14px;
  padding: 5px 9px;
  border-radius: 999px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  font-size: 12px;
}

.expert-artifact-panel h2 {
  margin: 0 0 14px;
  font-size: 22px;
  line-height: 1.35;
}

.expert-artifact-panel__summary {
  margin: 0 0 22px;
  line-height: 1.7;
}

.expert-artifact-section {
  margin-top: 22px;
}

.expert-artifact-section h3 {
  margin: 0 0 8px;
  font-size: 16px;
}

.expert-artifact-section p {
  margin: 0;
  line-height: 1.7;
  white-space: pre-wrap;
}

.expert-artifact-section ul {
  margin: 8px 0 0;
  padding-left: 22px;
  line-height: 1.7;
}

.expert-artifact-panel__html {
  width: 100%;
  min-height: 720px;
  border: 0;
}

.expert-artifact-panel__image {
  display: block;
  width: 100%;
  max-height: 720px;
  object-fit: contain;
}

.expert-artifact-panel__document {
  width: 100%;
  min-height: 720px;
  border: 0;
}

.expert-artifact-panel__media {
  display: block;
  width: 100%;
  margin-top: 20px;
}

.expert-artifact-panel__text {
  margin: 0;
  padding: 14px;
  overflow: auto;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--td-bg-color-secondarycontainer);
  font: inherit;
  line-height: 1.65;
}

.expert-artifact-panel__markdown {
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 1.75;
}

.expert-artifact-panel__markdown :deep(> :first-child) {
  margin-top: 0;
}

.expert-artifact-panel__markdown :deep(.chat-markdown-table) {
  overflow-x: auto;
}

.expert-artifact-panel__empty {
  color: var(--td-text-color-secondary);
  line-height: 1.7;
}

@media (max-width: 900px) {
  .expert-artifact-panel {
    width: min(92vw, 640px);
  }
}
</style>
