<template>
  <div class="published-expert-message">
    <div class="published-expert-message__header">
      <div>
        <div class="published-expert-message__eyebrow">已发布专家</div>
        <div class="published-expert-message__name">{{ message.expert?.display_name || '专家运行' }}</div>
      </div>
      <span
        class="published-expert-message__status"
        :class="message.submittedAnswers ? 'is-submitted' : `is-${message.run?.status || 'queued'}`"
      >
        {{ statusLabel }}
      </span>
    </div>

    <AgentRunTimeline :events="message.events" :run="message.run" :steps="message.steps" />

    <div v-if="message.run?.status === 'queued' || message.run?.status === 'running'" class="published-expert-message__loading">
      <span class="published-expert-spinner" aria-hidden="true"></span>
      <span class="published-expert-message__loading-copy">{{ phaseLabel }}</span>
      <button
        type="button"
        class="published-expert-message__cancel"
        :disabled="Boolean(message.cancelling)"
        @click="emit('cancel')"
      >
        {{ message.cancelling ? '正在取消' : '取消任务' }}
      </button>
    </div>

    <ExpertIntakePanel
      v-else-if="message.run?.status === 'waiting_input'"
      :interaction="message.run?.interaction"
      :submitted-answers="message.submittedAnswers"
      :submitted-summary="message.submittedAnswerSummary"
      :submitting="Boolean(message.submittingAnswers)"
      @submit="emit('submit-answers', $event)"
    />

    <section
      v-else-if="message.run?.status === 'failed' || message.run?.status === 'cancelled'"
      class="published-expert-message__error"
    >
      <span>{{ message.run?.error_message || '专家运行未完成' }}</span>
      <button
        v-if="message.run?.status === 'failed'"
        type="button"
        class="published-expert-message__retry"
        :disabled="Boolean(message.regeneratingAction)"
        @click="triggerRegeneration(retryAction)"
      >
        <t-icon :name="message.regeneratingAction ? 'loading' : 'refresh'" />
        {{ message.regeneratingAction ? '正在重试' : retryAction.label }}
      </button>
    </section>

    <template v-else>
      <div
        v-if="renderedResultDescription"
        class="published-expert-result-description markdown-content"
        v-html="renderedResultDescription"
      >
      </div>

      <div v-if="message.run?.result?.card" class="published-expert-card">
        <div class="published-expert-card__row">
          <span class="published-expert-card__label">标题</span>
          <strong>{{ message.run.result.card.title }}</strong>
        </div>
        <div class="published-expert-card__row">
          <span class="published-expert-card__label">摘要</span>
          <span>{{ message.run.result.card.summary }}</span>
        </div>
        <div class="published-expert-card__row">
          <span class="published-expert-card__label">下一步动作</span>
          <span>{{ message.run.result.card.next_action }}</span>
        </div>
      </div>

      <section v-if="artifacts.length" class="published-expert-artifacts">
        <div class="published-expert-artifacts__header">
          <strong>本轮产物</strong>
          <span>{{ artifacts.length }} 个文件</span>
        </div>
        <div class="published-expert-artifacts__grid">
          <button
            v-for="(artifact, index) in artifacts"
            :key="`${artifact.title}-${index}`"
            type="button"
            class="published-expert-artifact"
            :title="artifact.title || '专家产物'"
            @click="emit('open-artifact', artifact, message.run?.id || '')"
          >
            <span
              class="published-expert-artifact__icon"
              :class="`is-${artifactTone(artifact)}`"
              aria-hidden="true"
            >
              <t-icon :name="artifactIcon(artifact)" />
            </span>
            <span class="published-expert-artifact__copy">
              <strong>{{ artifact.title || '专家产物' }}</strong>
              <small>{{ artifactFileLabel(artifact) }}</small>
            </span>
            <t-icon class="published-expert-artifact__open" name="jump" />
          </button>
        </div>
      </section>

      <button
        v-if="message.run?.status === 'succeeded' && message.run?.parent_run_id"
        type="button"
        class="published-expert-version-diff"
        @click="emit('open-diff')"
      >
        <span>查看与上一版的差异</span>
        <t-icon name="chevron-right" />
      </button>

      <section v-if="message.run?.status === 'succeeded'" class="published-expert-follow-up">
        <div class="published-expert-follow-up__header">
          <strong>继续完善</strong>
          <span>基于当前结果生成新版本，原结果会保留</span>
        </div>
        <div class="published-expert-follow-up__actions">
          <button
            v-for="action in followUpActions"
            :key="action.key"
            type="button"
            :disabled="Boolean(message.regeneratingAction)"
            @click="triggerRegeneration(action)"
          >
            <span>{{ action.label }}</span>
            <t-icon :name="message.regeneratingAction === action.key ? 'loading' : 'chevron-right'" />
          </button>
        </div>
      </section>

      <section v-if="message.run?.status === 'succeeded'" class="published-expert-follow-up published-expert-explanation">
        <div class="published-expert-follow-up__header">
          <strong>了解结果</strong>
          <span>直接查看判断依据，不需要重新描述需求</span>
        </div>
        <div class="published-expert-follow-up__actions">
          <button
            v-for="action in explanationActions"
            :key="action.key"
            type="button"
            :disabled="Boolean(message.followingUpAction)"
            @click="triggerExplanation(action)"
          >
            <span>{{ action.label }}</span>
            <t-icon :name="message.followingUpAction === action.key ? 'loading' : 'chevron-right'" />
          </button>
        </div>
        <button
          type="button"
          class="published-expert-custom-follow-up-toggle"
          :disabled="Boolean(message.followingUpAction)"
          @click="customFollowUpOpen = !customFollowUpOpen"
        >
          <span>自定义追问</span>
          <t-icon :name="customFollowUpOpen ? 'chevron-up' : 'chevron-down'" />
        </button>
        <div v-if="customFollowUpOpen" class="published-expert-custom-follow-up">
          <textarea
            v-model="customFollowUpPrompt"
            class="published-expert-custom-follow-up__input"
            maxlength="500"
            rows="3"
            placeholder="有具体问题时再输入，也可以继续使用上面的选项"
            @keydown.meta.enter.prevent="submitCustomFollowUp"
            @keydown.ctrl.enter.prevent="submitCustomFollowUp"
          />
          <div class="published-expert-custom-follow-up__footer">
            <span>{{ customFollowUpPrompt.length }}/500</span>
            <button
              type="button"
              class="published-expert-custom-follow-up__submit"
              :disabled="!customFollowUpPrompt.trim() || Boolean(message.followingUpAction)"
              @click="submitCustomFollowUp"
            >
              <t-icon :name="message.followingUpAction === 'custom' ? 'loading' : 'arrow-right'" />
              发送追问
            </button>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import {
  createChatMarkdownRenderer,
  renderChatMarkdown,
} from '@/utils/chatMarkdownRenderer';
import {
  safeMarkdownToHTML,
  sanitizeMarkdownHTML,
} from '@/utils/security';
import AgentRunTimeline from './AgentRunTimeline.vue';
import ExpertIntakePanel from './ExpertIntakePanel.vue';

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits([
  'open-artifact',
  'submit-answers',
  'regenerate',
  'ask-follow-up',
  'cancel',
  'open-diff',
]);
const customFollowUpOpen = ref(false);
const customFollowUpPrompt = ref('');

const eventResultDescription = computed(() => (props.message?.events || [])
  .filter((event) => event?.type === 'TEXT_MESSAGE_CONTENT' && event?.delta)
  .map((event) => String(event.delta))
  .join('')
  .trim());
const resultDescription = computed(() => {
  const run = props.message?.run || {};
  const parts = [];
  const comparableText = (value) => String(value || '')
    .replace(/[*_`#>~-]/g, '')
    .replace(/\s+/g, ' ')
    .trim();
  const addPart = (value) => {
    const text = String(value || '').trim();
    if (!text) return;
    const normalized = comparableText(text);
    if (parts.some((part) => comparableText(part) === normalized)) return;
    parts.push(text);
  };
  const reportArtifact = (run?.result?.artifacts || []).find((artifact) => (
    artifact?.content?.format === 'structured_report_v1'
  ));
  const report = reportArtifact?.content;
  const sections = Array.isArray(report?.sections) ? report.sections : [];

  addPart(run?.result?.description);
  addPart(run?.result?.message);
  addPart(eventResultDescription.value);
  addPart(props.message?.liveSummary);
  addPart(run?.quality?.summary);
  addPart(report?.executive_summary);

  if (sections.length) {
    const sectionNames = sections
      .slice(0, 5)
      .map((section) => String(section?.title || '').trim())
      .filter(Boolean);
    addPart(
      `**本次产出范围**：共 ${sections.length} 个章节`
      + (sectionNames.length ? `，重点覆盖 ${sectionNames.join('、')} 等内容。` : '。'),
    );
  }

  if (!parts.length) {
    addPart(run?.result?.card?.summary);
    addPart(run?.result?.decision?.reason);
  }
  return parts.join('\n\n');
});
const markdownRenderer = createChatMarkdownRenderer();
const renderedResultDescription = computed(() => renderChatMarkdown(resultDescription.value, {
  renderer: markdownRenderer,
  escapeMarkdown: safeMarkdownToHTML,
  sanitizeHtml: sanitizeMarkdownHTML,
}));

const artifacts = computed(() => props.message?.run?.result?.artifacts || []);
const artifactType = (artifact) => {
  const mimeType = String(artifact?.mime_type || '').toLowerCase();
  const fileName = String(artifact?.original_name || '').toLowerCase();
  const format = String(artifact?.format || '').toLowerCase();
  if (mimeType === 'text/html' || fileName.endsWith('.html') || format === 'html') return 'html';
  if (mimeType === 'application/pdf' || fileName.endsWith('.pdf') || format === 'pdf') return 'pdf';
  if (mimeType.startsWith('image/')) return 'image';
  if (mimeType.startsWith('audio/')) return 'audio';
  if (mimeType.startsWith('video/')) return 'video';
  if (
    mimeType.includes('wordprocessingml')
    || mimeType === 'application/msword'
    || /\.(doc|docx)$/.test(fileName)
  ) return 'word';
  if (
    mimeType.includes('spreadsheetml')
    || mimeType === 'application/vnd.ms-excel'
    || /\.(xls|xlsx)$/.test(fileName)
  ) return 'sheet';
  if (
    mimeType.includes('presentationml')
    || mimeType === 'application/vnd.ms-powerpoint'
    || /\.(ppt|pptx)$/.test(fileName)
  ) return 'slides';
  if (
    mimeType === 'text/markdown'
    || fileName.endsWith('.md')
    || ['md', 'markdown'].includes(format)
  ) return 'markdown';
  if (mimeType.includes('json') || fileName.endsWith('.json') || format === 'json') return 'json';
  if (mimeType.includes('csv') || fileName.endsWith('.csv') || format === 'csv') return 'csv';
  if (mimeType.startsWith('text/')) return 'text';
  return 'file';
};
const artifactFileLabel = (artifact) => {
  const labels = {
    html: 'HTML 网页',
    pdf: 'PDF 文档',
    image: '图片',
    audio: '音频文件',
    video: '视频文件',
    word: 'Word 文档',
    sheet: '表格文件',
    slides: '演示文稿',
    markdown: 'Markdown 文档',
    json: 'JSON 数据',
    csv: 'CSV 数据',
    text: '文本文件',
    file: '文件产物',
  };
  const type = artifactType(artifact);
  const previewable = ['html', 'pdf', 'image', 'audio', 'video', 'markdown', 'json', 'csv', 'text'].includes(type);
  const details = [formatArtifactSize(artifact?.size_bytes), labels[type] || labels.file].filter(Boolean);
  details.push(previewable ? '可在线预览' : '点击查看');
  return details.join(' · ');
};
const formatArtifactSize = (value) => {
  const bytes = Number(value || 0);
  if (!Number.isFinite(bytes) || bytes <= 0) return '';
  if (bytes < 1024) return `${Math.round(bytes)} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(bytes >= 10240 ? 0 : 1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(bytes >= 10 * 1024 * 1024 ? 0 : 1)} MB`;
};
const artifactIcon = (artifact) => {
  const type = artifactType(artifact);
  if (type === 'html' || type === 'json') return 'code';
  if (type === 'image') return 'image';
  if (type === 'pdf') return 'file-pdf';
  if (type === 'audio') return 'sound';
  if (type === 'video') return 'video';
  return 'file';
};
const artifactTone = (artifact) => {
  const type = artifactType(artifact);
  if (type === 'pdf') return 'pdf';
  if (type === 'image') return 'image';
  if (type === 'audio' || type === 'video') return 'media';
  if (type === 'html' || type === 'json') return 'code';
  return 'document';
};
const followUpActions = [
  {
    key: 'execution_details',
    label: '补充执行细节',
    feedback: '在保留上一版有效内容的基础上，补充可直接执行的步骤、时间安排、责任分工和检查清单。',
  },
  {
    key: 'action_checklist',
    label: '整理成行动清单',
    feedback: '将上一版方案进一步整理为明确的行动清单，标注优先级、负责人角色、时间节点和完成标准。',
  },
  {
    key: 'concise_version',
    label: '调整得更简洁',
    feedback: '保留上一版的核心结论和必要依据，减少重复说明，生成更简洁、便于快速执行的新版本。',
  },
  {
    key: 'new_version',
    label: '重新生成一版',
    feedback: '基于原需求重新生成一版完整结果，保留已经确认的信息，并优化结构与可执行性。',
  },
];
const explanationActions = [
  {
    key: 'why_plan',
    label: '为什么这样安排',
    prompt: '请解释上一版方案为什么这样安排，优先说明核心判断、关键依据和主要取舍。',
  },
  {
    key: 'budget_logic',
    label: '预算是怎么算的',
    prompt: '请解释上一版方案中的预算或资源投入是如何估算的，并区分已知数据、合理假设和待确认项。',
  },
  {
    key: 'risk_logic',
    label: '安全风险怎么判断',
    prompt: '请解释上一版方案中的主要风险是如何判断的，以及哪些风险需要优先验证或设置兜底措施。',
  },
];
const retryAction = {
  key: 'retry',
  label: '重试本次任务',
  feedback: '重新执行原需求，保留已经确认的信息，并优先修复上一轮失败的问题。',
};

const triggerRegeneration = (action) => {
  if (!action || props.message?.regeneratingAction) return;
  emit('regenerate', action);
};

const triggerExplanation = (action) => {
  if (!action || props.message?.followingUpAction) return;
  emit('ask-follow-up', action);
};

const submitCustomFollowUp = () => {
  const prompt = customFollowUpPrompt.value.trim();
  if (!prompt || props.message?.followingUpAction) return;
  emit('ask-follow-up', {
    key: 'custom',
    label: '自定义追问',
    displayText: prompt,
    prompt,
  });
  customFollowUpPrompt.value = '';
  customFollowUpOpen.value = false;
};

const phaseLabels = {
  intake: '需求澄清',
  planning: '执行规划',
  drafting: '报告撰写',
  reviewing: '质量审查',
  revising: '报告修订',
  packaging: '结果整理',
  completed: '任务完成',
};

const phaseLabel = computed(() =>
  phaseLabels[props.message?.run?.phase] || (
    props.message?.run?.status === 'queued' ? '排队中' : '准备执行'
  ),
);

const statusLabel = computed(() => {
  if (props.message?.submittedAnswers) return '已提交';
  switch (props.message?.run?.status) {
    case 'queued': return '排队中';
    case 'running': return '生成中';
    case 'waiting_input': return '等待补充';
    case 'succeeded': return '已完成';
    case 'failed': return '失败';
    case 'cancelled': return '已取消';
    default: return '准备中';
  }
});
</script>

<style scoped lang="less">
@import '../../../components/css/chat-markdown.less';

.published-expert-message {
  width: min(680px, 100%);
  color: var(--td-text-color-primary);
  font-size: 14px;
}

.published-expert-message__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;
}

.published-expert-message__eyebrow {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.published-expert-message__name {
  margin-top: 2px;
  font-weight: 600;
}

.published-expert-message__status {
  flex-shrink: 0;
  padding: 3px 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.published-expert-message__status.is-succeeded {
  color: var(--td-success-color);
}

.published-expert-message__status.is-submitted {
  color: #4c785c;
}

.published-expert-message__status.is-failed,
.published-expert-message__status.is-cancelled {
  color: var(--td-error-color);
}

.published-expert-message__loading,
.published-expert-message__waiting,
.published-expert-message__error {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  color: var(--td-text-color-secondary);
}

.published-expert-message__loading-copy {
  flex: 1;
  min-width: 0;
}

.published-expert-message__cancel {
  min-height: 28px;
  flex: 0 0 auto;
  padding: 0 9px;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
}

.published-expert-message__cancel:hover:not(:disabled) {
  border-color: var(--td-error-color);
  color: var(--td-error-color);
}

.published-expert-message__cancel:disabled {
  cursor: wait;
  opacity: .6;
}

.published-expert-progress {
  margin-bottom: 10px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.published-expert-progress__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.published-expert-progress__title {
  font-weight: 600;
}

.published-expert-progress__phase {
  color: var(--td-brand-color);
  font-size: 12px;
}

.published-expert-progress__list {
  display: grid;
  gap: 6px;
  margin: 0;
  padding: 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  list-style: none;
}

.published-expert-progress__list li {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  line-height: 1.45;
}

.published-expert-progress__dot {
  width: 6px;
  height: 6px;
  flex: 0 0 auto;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.published-expert-message__error {
  justify-content: space-between;
  color: var(--td-error-color);
}

.published-expert-message__retry {
  display: inline-flex;
  min-height: 32px;
  flex: 0 0 auto;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  border: 1px solid var(--td-error-color-4);
  border-radius: 6px;
  color: var(--td-error-color);
  background: var(--td-bg-color-container);
  cursor: pointer;
}

.published-expert-message__retry:disabled {
  cursor: wait;
  opacity: .65;
}

.published-expert-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--td-component-stroke);
  border-top-color: var(--td-brand-color);
  border-radius: 50%;
  animation: publishedExpertSpin .8s linear infinite;
}

.published-expert-result-description {
  margin: 4px 0 14px;
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 1.75;
}

.published-expert-result-description :deep(> :first-child) {
  margin-top: 0;
}

.published-expert-result-description :deep(> :last-child) {
  margin-bottom: 0;
}

.published-expert-result-description :deep(.chat-markdown-table) {
  overflow-x: auto;
}

.published-expert-card {
  display: grid;
  gap: 8px;
  padding: 2px 0 4px;
}

.published-expert-card__row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  line-height: 1.55;
}

.published-expert-card__label {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.published-expert-artifacts {
  margin-top: 12px;
}

.published-expert-artifacts__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.published-expert-artifacts__header strong {
  font-size: 13px;
}

.published-expert-artifacts__header span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.published-expert-artifacts__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.published-expert-artifact {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 64px;
  min-width: 0;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid transparent;
  border-radius: 8px;
  outline: none;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-secondarycontainer);
  cursor: pointer;
  text-align: left;
  transition: background-color .16s ease, border-color .16s ease;
}

.published-expert-artifact:hover {
  border-color: var(--td-component-stroke);
  background: color-mix(
    in srgb,
    var(--td-bg-color-secondarycontainer) 92%,
    var(--td-text-color-primary)
  );
}

.published-expert-artifact:focus-visible {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 2px var(--td-brand-color-focus);
}

.published-expert-artifact__icon {
  display: inline-flex;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 7px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.published-expert-artifact__icon.is-code {
  color: var(--td-success-color);
  background: var(--td-success-color-light);
}

.published-expert-artifact__icon.is-pdf {
  color: var(--td-error-color);
  background: var(--td-error-color-light);
}

.published-expert-artifact__icon.is-image {
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.published-expert-artifact__icon.is-media {
  color: var(--td-warning-color);
  background: var(--td-warning-color-light);
}

.published-expert-artifact__icon :deep(svg) {
  width: 18px;
  height: 18px;
}

.published-expert-artifact__copy {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.published-expert-artifact__copy strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  line-height: 1.45;
}

.published-expert-artifact__copy small {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 1.4;
}

.published-expert-artifact__open {
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
}

.published-expert-artifact:hover .published-expert-artifact__open {
  color: var(--td-text-color-secondary);
}

.published-expert-version-diff {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 10px;
  padding: 0;
  border: 0;
  color: var(--td-brand-color);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
}

.published-expert-version-diff:hover {
  color: var(--td-brand-color-7);
}

.published-expert-follow-up {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid var(--td-component-border);
}

.published-expert-follow-up__header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}

.published-expert-follow-up__header strong {
  font-size: 13px;
}

.published-expert-follow-up__header span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  text-align: right;
}

.published-expert-follow-up__actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.published-expert-follow-up__actions button {
  display: flex;
  min-width: 0;
  min-height: 38px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid color-mix(in srgb, var(--td-component-border) 72%, transparent);
  border-radius: 6px;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  cursor: pointer;
  text-align: left;
}

.published-expert-follow-up__actions button:hover:not(:disabled) {
  border-color: var(--td-brand-color);
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.published-expert-follow-up__actions button:disabled {
  cursor: wait;
  opacity: .6;
}

.published-expert-follow-up__actions button span {
  overflow-wrap: anywhere;
}

.published-expert-custom-follow-up-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-top: 8px;
  padding: 0;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
}

.published-expert-custom-follow-up-toggle:hover:not(:disabled) {
  color: var(--td-brand-color);
}

.published-expert-custom-follow-up-toggle:disabled {
  cursor: not-allowed;
  opacity: .55;
}

.published-expert-custom-follow-up {
  margin-top: 8px;
  padding: 8px 0 0;
}

.published-expert-custom-follow-up__input {
  display: block;
  width: 100%;
  min-height: 72px;
  box-sizing: border-box;
  resize: vertical;
  padding: 8px 10px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-page);
  font: inherit;
  line-height: 1.5;
  outline: none;
}

.published-expert-custom-follow-up__input:focus {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 2px var(--td-brand-color-focus);
}

.published-expert-custom-follow-up__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.published-expert-custom-follow-up__submit {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  border: 0;
  border-radius: 6px;
  color: #fff;
  background: var(--td-brand-color);
  cursor: pointer;
  font-size: 12px;
}

.published-expert-custom-follow-up__submit:disabled {
  cursor: not-allowed;
  opacity: .5;
}

@keyframes publishedExpertSpin {
  to { transform: rotate(360deg); }
}

@media (max-width: 640px) {
  .published-expert-follow-up__header {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }

  .published-expert-follow-up__header span {
    text-align: left;
  }

  .published-expert-follow-up__actions {
    grid-template-columns: 1fr;
  }

  .published-expert-artifacts__grid {
    grid-template-columns: 1fr;
  }
}
</style>
