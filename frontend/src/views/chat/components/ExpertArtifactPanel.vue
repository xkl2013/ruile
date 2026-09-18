<template>
  <aside class="expert-artifact-panel" aria-label="专家产物预览">
    <header class="expert-artifact-panel__header">
      <div class="expert-artifact-panel__title-wrap">
        <div class="expert-artifact-panel__file-icon">DOC</div>
        <div class="expert-artifact-panel__title-copy">
          <strong>{{ artifact?.title || '专家产物' }}</strong>
          <small>{{ artifact?.format || artifact?.kind || 'structured report' }}</small>
        </div>
      </div>
      <button type="button" class="expert-artifact-panel__close" title="关闭预览" @click="emit('close')">×</button>
    </header>

    <div class="expert-artifact-panel__body">
      <template v-if="report">
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

      <template v-else-if="artifact?.kind === 'html' && htmlContent">
        <iframe class="expert-artifact-panel__html" :srcdoc="htmlContent" sandbox="allow-same-origin" />
      </template>

      <div v-else class="expert-artifact-panel__empty">
        当前产物暂不支持内嵌预览，请先查看产物卡片内容。
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue';

const props = defineProps({
  artifact: {
    type: Object,
    default: null,
  },
});

const emit = defineEmits(['close']);

const report = computed(() => {
  const content = props.artifact?.content;
  if (!content || props.artifact?.kind !== 'report') return null;
  if (content.format !== 'structured_report_v1') return null;
  return content;
});

const htmlContent = computed(() => {
  const content = props.artifact?.content;
  return typeof content === 'string' ? content : content?.html || '';
});
</script>

<style scoped lang="less">
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

.expert-artifact-panel__close {
  width: 28px;
  height: 28px;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 24px;
  line-height: 1;
}

.expert-artifact-panel__body {
  flex: 1;
  min-height: 0;
  padding: 28px;
  overflow: auto;
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
