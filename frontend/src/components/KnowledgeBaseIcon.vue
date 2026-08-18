<template>
  <div class="kb-icon" :class="[`kb-icon--${size}`, `kb-icon--${variant}`]">
    <img v-if="resolvedImageSrc" :src="resolvedImageSrc" alt="" class="kb-icon-image" />
    <span v-else-if="isEmoji" class="kb-icon-emoji">{{ emojiChar }}</span>
    <t-icon v-else :name="resolvedIcon" class="kb-icon-symbol" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { getDefaultKnowledgeBaseIcon } from '@/config/knowledgeBaseDefaults'

const props = withDefaults(defineProps<{
  icon?: string
  iconUrl?: string
  type?: 'document' | 'faq' | string
  size?: 'small' | 'medium' | 'large'
}>(), {
  icon: '',
  iconUrl: '',
  type: 'document',
  size: 'medium',
})

const variant = computed(() => (props.type === 'faq' ? 'faq' : 'document'))
const normalizedIcon = computed(() => (props.icon || '').trim())
const resolvedImageSrc = computed(() => {
  if (!normalizedIcon.value.startsWith('image:')) return ''
  const explicitURL = (props.iconUrl || '').trim()
  if (explicitURL) return explicitURL
  const src = normalizedIcon.value.slice(6).trim()
  if (/^data:image\/(?:png|jpe?g|gif|webp);base64,/i.test(src)) return src
  if (/^https?:\/\//i.test(src) || src.startsWith('/')) return src
  return ''
})
const resolvedIcon = computed(() => {
  const value = normalizedIcon.value
  if (!value || value.startsWith('emoji:') || value.startsWith('image:')) {
    return getDefaultKnowledgeBaseIcon(props.type === 'faq' ? 'faq' : 'document')
  }
  return value
})

const isEmoji = computed(() => {
  const value = normalizedIcon.value
  return value.startsWith('emoji:') && value.length > 6
})
const emojiChar = computed(() => {
  if (!isEmoji.value) return ''
  return normalizedIcon.value.slice(6).trim()
})
</script>

<style scoped lang="less">
.kb-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  overflow: hidden;

  &.kb-icon--document {
    background: rgba(7, 192, 95, 0.08);
    color: var(--td-brand-color);
  }

  &.kb-icon--faq {
    background: rgba(0, 82, 217, 0.08);
    color: var(--td-brand-color);
  }

  &.kb-icon--small {
    width: 20px;
    height: 20px;

    .kb-icon-symbol {
      font-size: 12px;
    }

    .kb-icon-emoji {
      font-size: 12px;
    }
  }

  &.kb-icon--medium {
    width: 24px;
    height: 24px;

    .kb-icon-symbol {
      font-size: 14px;
    }

    .kb-icon-emoji {
      font-size: 14px;
    }
  }

  &.kb-icon--large {
    width: 32px;
    height: 32px;

    .kb-icon-symbol {
      font-size: 18px;
    }

    .kb-icon-emoji {
      font-size: 18px;
    }
  }
}

.kb-icon-emoji {
  line-height: 1;
  user-select: none;
}

.kb-icon-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
</style>
