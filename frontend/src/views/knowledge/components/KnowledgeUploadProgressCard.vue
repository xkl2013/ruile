<template>
  <section
    v-if="state"
    class="knowledge-upload-progress"
    :class="`is-${state.status}`"
    role="status"
    aria-live="polite"
  >
    <header class="knowledge-upload-progress__header">
      <h2 class="knowledge-upload-progress__title">
        {{ t('knowledgeBase.uploadProgress.title') }}
      </h2>
      <span
        class="knowledge-upload-progress__add"
        aria-hidden="true"
      >
        <t-icon name="upload" size="22px" />
      </span>
    </header>

    <div class="knowledge-upload-progress__body">
      <div class="knowledge-upload-progress__file">
        <span class="knowledge-upload-progress__file-icon" aria-hidden="true">
          <t-icon name="folder" size="24px" />
        </span>
        <div class="knowledge-upload-progress__file-copy">
          <strong :title="state.title">{{ state.title }}</strong>
          <span>
            {{ state.processed }}/{{ state.total }}{{ t('knowledgeBase.uploadProgress.items') }}
          </span>
        </div>
      </div>

      <div
        class="knowledge-upload-progress__track"
        role="progressbar"
        :aria-valuenow="progressPercent"
        aria-valuemin="0"
        aria-valuemax="100"
      >
        <span :style="{ width: `${progressPercent}%` }" />
      </div>

      <p v-if="state.failed > 0" class="knowledge-upload-progress__error">
        {{ t('knowledgeBase.uploadProgress.failed', { count: state.failed }) }}
      </p>
    </div>

    <footer class="knowledge-upload-progress__footer">
      <button
        type="button"
        class="knowledge-upload-progress__action"
        :aria-label="state.status === 'paused'
          ? t('knowledgeBase.uploadProgress.resume')
          : t('knowledgeBase.uploadProgress.pause')"
        :title="state.status === 'paused'
          ? t('knowledgeBase.uploadProgress.resume')
          : t('knowledgeBase.uploadProgress.pause')"
        :disabled="state.status !== 'uploading' && state.status !== 'paused'"
        @click="state.status === 'paused' ? emit('resume') : emit('pause')"
      >
        <t-icon :name="state.status === 'paused' ? 'play-circle' : 'pause-circle'" size="28px" />
      </button>

      <span class="knowledge-upload-progress__divider" aria-hidden="true" />

      <button
        type="button"
        class="knowledge-upload-progress__action knowledge-upload-progress__action--danger"
        :aria-label="t('knowledgeBase.uploadProgress.cancel')"
        :title="t('knowledgeBase.uploadProgress.cancel')"
        @click="emit('cancel')"
      >
        <t-icon name="delete" size="28px" />
      </button>
    </footer>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { KnowledgeUploadProgressState } from './knowledgeUploadProgress'

const props = defineProps<{
  state: KnowledgeUploadProgressState | null
}>()

const emit = defineEmits<{
  pause: []
  resume: []
  cancel: []
}>()

const { t } = useI18n()

const progressPercent = computed(() => {
  const state = props.state
  if (!state || state.total <= 0) return 0
  return Math.min(100, Math.max(0, Math.round(
    ((state.processed + state.currentProgress / 100) / state.total) * 100,
  )))
})
</script>

<style scoped lang="less">
.knowledge-upload-progress {
  position: absolute;
  z-index: 12;
  right: 20px;
  bottom: 20px;
  display: flex;
  flex-direction: column;
  width: min(420px, calc(100% - 32px));
  height: 320px;
  min-height: 260px;
  max-height: calc(100% - 32px);
  overflow: hidden;
  border: 1px solid color-mix(in srgb, var(--td-component-stroke) 72%, transparent);
  border-radius: 20px;
  background: var(--td-bg-color-container);
  box-shadow: 0 12px 32px rgba(0, 0, 0, 0.12);
  color: var(--td-text-color-primary);

  &.is-paused {
    .knowledge-upload-progress__track span {
      background: var(--td-text-color-secondary);
    }
  }

  &.is-failed {
    .knowledge-upload-progress__track span {
      background: var(--td-error-color);
    }
  }

  &.is-cancelled {
    opacity: 0.86;
  }
}

.knowledge-upload-progress__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  padding: 20px 22px 10px;
}

.knowledge-upload-progress__title {
  margin: 0;
  font-size: 22px;
  font-weight: 600;
  line-height: 1.25;
}

.knowledge-upload-progress__add {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 6px;
  color: var(--td-text-color-primary);
}

.knowledge-upload-progress__body {
  flex: 1;
  min-height: 0;
  padding: 8px 24px 20px;
}

.knowledge-upload-progress__file {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  min-width: 0;
}

.knowledge-upload-progress__file-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  color: #b9e7d6;
}

.knowledge-upload-progress__file-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 10px;

  strong,
  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    font-size: 18px;
    font-weight: 500;
    line-height: 1.25;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 16px;
    line-height: 1.25;
  }
}

.knowledge-upload-progress__track {
  height: 3px;
  margin-top: 14px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--td-bg-color-secondarycontainer);

  span {
    display: block;
    height: 100%;
    min-width: 2px;
    border-radius: inherit;
    background: var(--td-success-color);
    transition: width 0.18s ease, background-color 0.18s ease;
  }
}

.knowledge-upload-progress__error {
  margin: 10px 0 0;
  color: var(--td-error-color);
  font-size: 13px;
}

.knowledge-upload-progress__footer {
  display: grid;
  grid-template-columns: 1fr 1px 1fr;
  align-items: center;
  flex-shrink: 0;
  min-height: 64px;
  padding: 0 28px;
  border-top: 1px solid var(--td-component-stroke);
}

.knowledge-upload-progress__action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  justify-self: center;
  width: 44px;
  height: 44px;
  padding: 0;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;

  &:hover:not(:disabled),
  &:focus-visible:not(:disabled) {
    color: var(--td-brand-color);
    outline: none;
  }

  &:disabled {
    color: var(--td-text-color-disabled);
    cursor: default;
  }

  &--danger:hover:not(:disabled),
  &--danger:focus-visible:not(:disabled) {
    color: var(--td-error-color);
  }
}

.knowledge-upload-progress__divider {
  width: 1px;
  height: 28px;
  background: var(--td-component-stroke);
}

@media (max-width: 720px) {
  .knowledge-upload-progress {
    right: 16px;
    bottom: 16px;
    width: calc(100% - 32px);
    height: 300px;
    min-height: 250px;
    max-height: calc(100% - 32px);
    border-radius: 18px;
  }

  .knowledge-upload-progress__header {
    padding: 18px 18px 8px;
  }

  .knowledge-upload-progress__title {
    font-size: 20px;
  }

  .knowledge-upload-progress__body {
    padding: 8px 18px 18px;
  }

  .knowledge-upload-progress__file-copy {
    strong {
      font-size: 17px;
    }

    span {
      font-size: 15px;
    }
  }

  .knowledge-upload-progress__footer {
    min-height: 60px;
    padding: 0 20px;
  }
}
</style>
