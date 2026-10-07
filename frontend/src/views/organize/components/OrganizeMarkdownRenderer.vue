<template>
  <div
    class="organize-markdown-renderer"
    :class="`organize-markdown-renderer--${profile}`"
    v-html="renderedHtml"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { renderOrganizeMarkdown, type RenderOrganizeMarkdownOptions } from '../organizeMarkdown'

type OrganizeMarkdownProfile = 'report' | 'checklist'

const props = withDefaults(
  defineProps<{
    content?: string
    profile?: OrganizeMarkdownProfile
  }>(),
  {
    content: '',
    profile: 'report',
  },
)

const renderOptions: RenderOrganizeMarkdownOptions = {}
const renderedHtml = computed(() => renderOrganizeMarkdown(props.content, renderOptions))
</script>

<style scoped lang="less">
@import '../../../components/css/chat-markdown.less';

.organize-markdown-renderer {
  padding-top: 8px;
  .chat-markdown-typography();
}

.organize-markdown-renderer--checklist :deep(input[type='checkbox']) {
  margin-right: 6px;
  accent-color: var(--td-brand-color);
}

.organize-markdown-renderer--report {
  max-width: 880px;
  color: var(--td-text-color-primary);
}

.organize-markdown-renderer--report :deep(h1) {
  margin: 10px 0 18px;
  font-size: 26px;
  line-height: 1.3;
  font-weight: 650;
}

.organize-markdown-renderer--report :deep(h2) {
  margin: 28px 0 10px;
  padding-left: 12px;
  border-left: 3px solid var(--td-brand-color);
  font-size: 18px;
  line-height: 1.5;
  font-weight: 600;
}

.organize-markdown-renderer--report :deep(h3) {
  margin: 20px 0 8px;
  font-size: 15px;
  line-height: 1.5;
  font-weight: 600;
}

.organize-markdown-renderer--report :deep(blockquote) {
  margin: 12px 0 18px;
  padding: 10px 14px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  border-left: 3px solid var(--td-brand-color);
  border-radius: 0 6px 6px 0;
}

.organize-markdown-renderer--report :deep(table) {
  width: 100%;
  margin: 14px 0 20px;
  border-collapse: collapse;
  font-size: 13px;
}

.organize-markdown-renderer--report :deep(th),
.organize-markdown-renderer--report :deep(td) {
  padding: 9px 10px;
  text-align: left;
  border: 1px solid var(--td-component-border);
  vertical-align: top;
}

.organize-markdown-renderer--report :deep(th) {
  color: var(--td-text-color-primary);
  background: var(--td-bg-color-secondarycontainer);
  font-weight: 600;
}

.organize-markdown-renderer--report :deep(hr) {
  margin: 22px 0;
  border: 0;
  border-top: 1px solid var(--td-component-border);
}

.organize-markdown-renderer--report :deep(input[type='checkbox']) {
  margin-right: 6px;
  accent-color: var(--td-brand-color);
}

/* [M1] 来源徽章：弱化为可点击小角标 */
.organize-markdown-renderer--report :deep(.organize-cite-ref),
.organize-markdown-renderer--checklist :deep(.organize-cite-ref) {
  display: inline-block;
  margin: 0 2px;
  padding: 0 5px;
  border-radius: 4px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 10px;
  font-weight: 600;
  line-height: 15px;
  vertical-align: 2px;
  cursor: pointer;
  user-select: none;
}

/* 表格弱化：细边框 + 斑马纹 */
.organize-markdown-renderer--report :deep(th),
.organize-markdown-renderer--report :deep(td) {
  border-color: var(--td-component-stroke);
}

.organize-markdown-renderer--report :deep(tbody tr:nth-child(even) td) {
  background: var(--td-bg-color-secondarycontainer);
}
</style>
