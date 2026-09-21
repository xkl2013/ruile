<template>
  <aside class="expert-run-diff-panel" aria-label="专家运行版本差异">
    <header class="expert-run-diff-panel__header">
      <div>
        <strong>版本差异</strong>
        <small>{{ diff?.changed ? '当前版本与上一版存在变化' : '当前版本与上一版内容一致' }}</small>
      </div>
      <button type="button" title="关闭差异" @click="emit('close')">×</button>
    </header>

    <div v-if="loading" class="expert-run-diff-panel__loading">正在读取版本差异...</div>
    <div v-else class="expert-run-diff-panel__body">
      <section>
        <h3>上一版</h3>
        <pre>{{ diff?.before || '没有可比较的上一版内容' }}</pre>
      </section>
      <section>
        <h3>当前版</h3>
        <pre>{{ diff?.after || '当前版本没有可比较的正文' }}</pre>
      </section>
    </div>
  </aside>
</template>

<script setup>
defineProps({
  diff: {
    type: Object,
    default: null,
  },
  loading: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['close']);
</script>

<style scoped lang="less">
.expert-run-diff-panel {
  position: fixed;
  z-index: 12010;
  top: 0;
  right: 0;
  display: flex;
  width: min(46vw, 720px);
  height: 100vh;
  flex-direction: column;
  border-left: 1px solid var(--td-component-border);
  background: var(--td-bg-color-container);
  box-shadow: -8px 0 24px rgba(0, 0, 0, .08);
}

.expert-run-diff-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 72px;
  padding: 14px 18px;
  border-bottom: 1px solid var(--td-component-border);
}

.expert-run-diff-panel__header div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.expert-run-diff-panel__header small {
  color: var(--td-text-color-placeholder);
}

.expert-run-diff-panel__header button {
  width: 28px;
  height: 28px;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  font-size: 24px;
  line-height: 1;
}

.expert-run-diff-panel__loading {
  display: flex;
  min-height: 120px;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
}

.expert-run-diff-panel__body {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  flex: 1;
  min-height: 0;
}

.expert-run-diff-panel__body section {
  min-width: 0;
  padding: 18px;
  overflow: auto;
}

.expert-run-diff-panel__body section + section {
  border-left: 1px solid var(--td-component-border);
}

.expert-run-diff-panel__body h3 {
  margin: 0 0 10px;
  font-size: 13px;
}

.expert-run-diff-panel__body pre {
  margin: 0;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--td-text-color-secondary);
  font: inherit;
  line-height: 1.65;
}

@media (max-width: 900px) {
  .expert-run-diff-panel {
    width: min(92vw, 720px);
  }

  .expert-run-diff-panel__body {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .expert-run-diff-panel__body section {
    overflow: visible;
  }

  .expert-run-diff-panel__body section + section {
    border-top: 1px solid var(--td-component-border);
    border-left: 0;
  }
}
</style>
