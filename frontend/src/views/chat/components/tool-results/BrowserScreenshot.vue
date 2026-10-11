<template>
  <div class="browser-screenshot">
    <div class="browser-screenshot__meta">
      <span>{{ data.title || data.url || 'Browser screenshot' }}</span>
      <span v-if="data.size_bytes">{{ formatBytes(data.size_bytes) }}</span>
    </div>
    <img
      v-for="(image, index) in data.images || []"
      :key="`${image.slice(0, 24)}-${index}`"
      :src="image"
      alt="Browser screenshot"
      class="browser-screenshot__image"
    />
    <div v-if="!(data.images || []).length" class="browser-screenshot__empty">
      Screenshot unavailable
    </div>
  </div>
</template>

<script setup lang="ts">
import type { BrowserScreenshotData } from '@/types/tool-results'

defineProps<{ data: BrowserScreenshotData }>()

const formatBytes = (value: number) => {
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / (1024 * 1024)).toFixed(1)} MB`
}
</script>

<style scoped>
.browser-screenshot {
  display: grid;
  gap: 8px;
}

.browser-screenshot__meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.browser-screenshot__image {
  display: block;
  max-width: 100%;
  max-height: 520px;
  object-fit: contain;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.browser-screenshot__empty {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}
</style>
