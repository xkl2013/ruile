<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { getFileIcon } from '@/utils/files';

interface Tag {
  id: string;
  name: string;
  color?: string;
}

interface KnowledgeItem {
  id: string;
  file_name: string;
  file_type?: string;
  type?: string;
  tags?: Tag[];
  parse_status?: string;
  description?: string;
  error_message?: string;
}

defineProps<{
  items: KnowledgeItem[];
  canEdit: boolean;
}>();

const emit = defineEmits<{
  (e: 'open', item: KnowledgeItem): void;
  (e: 'tag-edit', item: KnowledgeItem): void;
  (e: 'filter-tag', tag: Tag): void;
}>();

const { t } = useI18n();

const getFileTone = (item: KnowledgeItem) => {
  const type = String(item.file_type || '').toLowerCase();
  if (type === 'pdf') return 'is-pdf';
  if (['doc', 'docx'].includes(type)) return 'is-word';
  if (['xls', 'xlsx', 'csv'].includes(type)) return 'is-sheet';
  if (['ppt', 'pptx'].includes(type)) return 'is-slide';
  if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp'].includes(type)) return 'is-image';
  if (item.type === 'url') return 'is-link';
  return 'is-default';
};

const getSummary = (item: KnowledgeItem) => {
  if (item.description?.trim()) return item.description.trim();
  if (item.parse_status === 'failed' || item.parse_status === 'cancelled') {
    return item.error_message?.trim() || t('knowledgeBase.fileModeSummaryFailed');
  }
  if (['pending', 'processing', 'finalizing'].includes(String(item.parse_status || ''))) {
    return t('knowledgeBase.fileModeSummaryProcessing');
  }
  if (item.parse_status === 'draft') return t('knowledgeBase.draftTip');
  return t('knowledgeBase.fileModeSummaryEmpty');
};
</script>

<template>
  <div class="document-file-feed">
    <article
      v-for="item in items"
      :key="item.id"
      class="document-file-row"
      role="button"
      tabindex="0"
      :title="getSummary(item)"
      @click="emit('open', item)"
      @keydown.enter.prevent="emit('open', item)"
      @keydown.space.prevent="emit('open', item)"
    >
      <span class="document-file-icon" :class="getFileTone(item)">
        <t-icon :name="getFileIcon(item)" size="17px" />
      </span>

      <div class="document-file-main">
        <div class="document-file-title-row">
          <strong class="document-file-name" :title="item.file_name">{{ item.file_name }}</strong>
          <div v-if="item.tags?.length" class="document-file-tags">
            <button
              v-for="tag in item.tags"
              :key="tag.id"
              type="button"
              class="document-file-tag"
              :title="tag.name"
              @click.stop="emit('filter-tag', tag)"
            >
              {{ tag.name }}
            </button>
          </div>
        </div>
        <p class="document-file-summary">{{ getSummary(item) }}</p>
      </div>

      <div class="document-file-actions" @click.stop>
        <t-tooltip :content="$t('knowledgeBase.viewDetails')" placement="top">
          <button
            type="button"
            class="document-file-action"
            :aria-label="$t('knowledgeBase.viewDetails')"
            @click="emit('open', item)"
          >
            <t-icon name="browse" size="16px" />
          </button>
        </t-tooltip>
        <t-tooltip v-if="canEdit" :content="$t('knowledgeBase.editTags')" placement="top">
          <button
            type="button"
            class="document-file-action"
            :aria-label="$t('knowledgeBase.editTags')"
            @click="emit('tag-edit', item)"
          >
            <t-icon name="discount" size="16px" />
          </button>
        </t-tooltip>
      </div>
    </article>
  </div>
</template>

<style scoped lang="less">
@keyframes document-file-feed-in {
  from {
    opacity: 0;
    transform: translateY(4px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.document-file-feed {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  animation: document-file-feed-in 0.24s ease-out;
}

.document-file-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  width: 100%;
  min-height: 66px;
  padding: 11px 14px;
  box-sizing: border-box;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  outline: none;
  transition:
    border-color 0.15s ease,
    background-color 0.15s ease,
    box-shadow 0.15s ease;

  &:hover,
  &:focus-visible,
  &:focus-within {
    border-color: color-mix(in srgb, var(--td-brand-color) 52%, var(--td-component-stroke));
    background: color-mix(in srgb, var(--td-brand-color) 2%, var(--td-bg-color-container));
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

    .document-file-actions {
      opacity: 1;
      pointer-events: auto;
    }
  }
}

.document-file-icon {
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  margin-top: 1px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);

  &.is-pdf {
    background: color-mix(in srgb, #e2574c 12%, var(--td-bg-color-container));
    color: #cf4036;
  }

  &.is-word {
    background: color-mix(in srgb, #3b78d8 12%, var(--td-bg-color-container));
    color: #2f6fca;
  }

  &.is-sheet {
    background: color-mix(in srgb, #1d9e75 12%, var(--td-bg-color-container));
    color: #178663;
  }

  &.is-slide {
    background: color-mix(in srgb, #d97706 12%, var(--td-bg-color-container));
    color: #c56b05;
  }

  &.is-image,
  &.is-link {
    background: color-mix(in srgb, #7f77dd 12%, var(--td-bg-color-container));
    color: #6d64ce;
  }
}

.document-file-main {
  flex: 1;
  min-width: 0;
}

.document-file-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.document-file-name {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  font-weight: 600;
  line-height: 22px;
}

.document-file-tags {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  overflow: hidden;
}

.document-file-tag {
  max-width: 112px;
  height: 20px;
  flex-shrink: 0;
  padding: 0 7px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: 11px;
  line-height: 18px;
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, background-color 0.15s ease;

  &:hover,
  &:focus-visible {
    border-color: color-mix(in srgb, var(--td-brand-color) 42%, var(--td-component-stroke));
    background: color-mix(in srgb, var(--td-brand-color) 7%, var(--td-bg-color-container));
    color: var(--td-brand-color);
    outline: none;
  }
}

.document-file-summary {
  margin: 1px 0 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 20px;
}

.document-file-actions {
  height: 30px;
  flex: 0 0 auto;
  display: flex;
  align-items: center;
  gap: 2px;
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s ease;
}

.document-file-action {
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 5px;
  padding: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;

  &:hover,
  &:focus-visible {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
    outline: none;
  }
}

@media (max-width: 720px) {
  .document-file-row {
    padding: 10px 12px;
  }

  .document-file-title-row {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }

  .document-file-tags {
    width: 100%;
  }

  .document-file-actions {
    opacity: 1;
    pointer-events: auto;
  }
}
</style>
