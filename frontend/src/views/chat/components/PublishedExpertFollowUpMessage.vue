<template>
  <div class="published-expert-follow-up-message">
    <div class="published-expert-follow-up-message__header">
      <div>
        <div class="published-expert-follow-up-message__eyebrow">专家追问</div>
        <div class="published-expert-follow-up-message__name">{{ message.expert?.display_name || '专家解释' }}</div>
      </div>
      <span class="published-expert-follow-up-message__status" :class="`is-${message.run?.status || 'queued'}`">
        {{ statusLabel }}
      </span>
    </div>

    <div v-if="message.run?.status === 'queued' || message.run?.status === 'running'" class="published-expert-follow-up-message__loading">
      <span class="published-expert-follow-up-message__spinner" aria-hidden="true"></span>
      <span>{{ message.run?.status === 'queued' ? '排队中' : '正在解释' }}</span>
    </div>
    <div v-else-if="message.run?.status === 'failed' || message.run?.status === 'cancelled'" class="published-expert-follow-up-message__error">
      {{ message.run?.error_message || '专家追问未完成' }}
    </div>
    <div
      v-else
      class="published-expert-follow-up-message__answer markdown-content"
      v-html="renderedAnswer"
    >
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import {
  createChatMarkdownRenderer,
  renderChatMarkdown,
} from '@/utils/chatMarkdownRenderer';
import {
  safeMarkdownToHTML,
  sanitizeMarkdownHTML,
} from '@/utils/security';

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
});

const answer = computed(() => String(props.message?.run?.result?.message || props.message?.liveSummary || '').trim());
const markdownRenderer = createChatMarkdownRenderer();
const renderedAnswer = computed(() => renderChatMarkdown(answer.value, {
  renderer: markdownRenderer,
  escapeMarkdown: safeMarkdownToHTML,
  sanitizeHtml: sanitizeMarkdownHTML,
}));

const statusLabel = computed(() => {
  switch (props.message?.run?.status) {
    case 'queued': return '排队中';
    case 'running': return '解释中';
    case 'succeeded': return '已完成';
    case 'failed': return '失败';
    case 'cancelled': return '已取消';
    default: return '准备中';
  }
});
</script>

<style scoped lang="less">
@import '../../../components/css/chat-markdown.less';

.published-expert-follow-up-message {
  width: min(680px, 100%);
  color: var(--td-text-color-primary);
  font-size: 14px;
}

.published-expert-follow-up-message__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 10px;
}

.published-expert-follow-up-message__eyebrow {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.published-expert-follow-up-message__name {
  margin-top: 2px;
  font-weight: 600;
}

.published-expert-follow-up-message__status {
  flex-shrink: 0;
  padding: 3px 8px;
  border-radius: 999px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  font-size: 12px;
}

.published-expert-follow-up-message__status.is-succeeded {
  color: var(--td-success-color);
}

.published-expert-follow-up-message__status.is-failed,
.published-expert-follow-up-message__status.is-cancelled {
  color: var(--td-error-color);
}

.published-expert-follow-up-message__loading,
.published-expert-follow-up-message__error,
.published-expert-follow-up-message__answer {
  padding: 12px 14px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
}

.published-expert-follow-up-message__loading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
}

.published-expert-follow-up-message__error {
  color: var(--td-error-color);
}

.published-expert-follow-up-message__answer {
  line-height: 1.65;
  background: var(--td-bg-color-container);
}

.published-expert-follow-up-message__answer :deep(.chat-markdown-table) {
  overflow-x: auto;
}

.published-expert-follow-up-message__spinner {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  border: 2px solid var(--td-component-stroke);
  border-top-color: var(--td-brand-color);
  border-radius: 50%;
  animation: publishedExpertFollowUpSpin .8s linear infinite;
}

@keyframes publishedExpertFollowUpSpin {
  to { transform: rotate(360deg); }
}
</style>
