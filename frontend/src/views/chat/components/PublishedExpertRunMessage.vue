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

    <section v-if="progressEvents.length" class="published-expert-progress" aria-live="polite">
      <div class="published-expert-progress__header">
        <span class="published-expert-progress__title">执行进度</span>
        <span class="published-expert-progress__phase">{{ phaseLabel }}</span>
      </div>
      <ol class="published-expert-progress__list">
        <li v-for="event in progressEvents" :key="event.sequence || event.id">
          <span class="published-expert-progress__dot" aria-hidden="true"></span>
          <span>{{ eventLabel(event) }}</span>
        </li>
      </ol>
    </section>

    <div v-if="message.run?.status === 'queued' || message.run?.status === 'running'" class="published-expert-message__loading">
      <span class="published-expert-spinner" aria-hidden="true"></span>
      <span>{{ phaseLabel }}</span>
    </div>

    <ExpertIntakePanel
      v-else-if="message.run?.status === 'waiting_input'"
      :interaction="message.run?.interaction"
      :submitted-answers="message.submittedAnswers"
      :submitted-summary="message.submittedAnswerSummary"
      :submitting="Boolean(message.submittingAnswers)"
      @submit="emit('submit-answers', $event)"
    />

    <div v-else-if="message.run?.status === 'failed' || message.run?.status === 'cancelled'"
      class="published-expert-message__error">
      {{ message.run?.error_message || '专家运行未完成' }}
    </div>

    <template v-else>
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

      <button
        v-for="(artifact, index) in artifacts"
        :key="`${artifact.title}-${index}`"
        type="button"
        class="published-expert-artifact"
        @click="emit('open-artifact', artifact)"
      >
        <span class="published-expert-artifact__icon" aria-hidden="true">▣</span>
        <span class="published-expert-artifact__copy">
          <strong>{{ artifact.title || '专家产物' }}</strong>
          <small>{{ artifact.format || artifact.kind || 'artifact' }}</small>
        </span>
        <span class="published-expert-artifact__open" aria-hidden="true">↗</span>
      </button>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue';
import ExpertIntakePanel from './ExpertIntakePanel.vue';

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits(['open-artifact', 'submit-answers']);

const artifacts = computed(() => props.message?.run?.result?.artifacts || []);

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

const progressEvents = computed(() => {
  const events = Array.isArray(props.message?.events) ? props.message.events : [];
  return events
    .filter((event) => [
      'AGENT_STEP_STARTED',
      'AGENT_STEP_FINISHED',
      'AGENT_STEP_ERROR',
      'QUALITY_UPDATED',
      'RUN_WAITING_INPUT',
      'RUN_ERROR',
    ].includes(event?.type))
    .slice(-6);
});

const eventLabel = (event) => {
  if (event?.message) return event.message;
  if (event?.type === 'QUALITY_UPDATED') {
    const score = event?.quality?.score;
    return typeof score === 'number' ? `质量审查完成，评分 ${score}` : '质量审查完成';
  }
  if (event?.type === 'RUN_WAITING_INPUT') return '等待补充关键信息';
  if (event?.type === 'RUN_ERROR') return event?.message || '执行失败';
  return phaseLabels[event?.phase] || '执行阶段已更新';
};

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
  padding: 3px 8px;
  border-radius: 999px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
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
  padding: 12px 14px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  color: var(--td-text-color-secondary);
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
  color: var(--td-error-color);
}

.published-expert-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--td-component-stroke);
  border-top-color: var(--td-brand-color);
  border-radius: 50%;
  animation: publishedExpertSpin .8s linear infinite;
}

.published-expert-card {
  display: grid;
  gap: 10px;
  padding: 14px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.published-expert-card__row {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  line-height: 1.55;
}

.published-expert-card__label {
  color: var(--td-text-color-placeholder);
}

.published-expert-artifact {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 10px;
  margin-top: 10px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-border);
  border-radius: 8px;
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-container);
  cursor: pointer;
  text-align: left;
}

.published-expert-artifact:hover {
  border-color: var(--td-brand-color);
  background: var(--td-bg-color-secondarycontainer);
}

.published-expert-artifact__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border-radius: 6px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
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
}

.published-expert-artifact__copy small {
  color: var(--td-text-color-placeholder);
}

.published-expert-artifact__open {
  color: var(--td-text-color-placeholder);
}

@keyframes publishedExpertSpin {
  to { transform: rotate(360deg); }
}
</style>
