<template>
  <img
    :src="iconSrc"
    :class="['knowledge-base-scope-icon', `knowledge-base-scope-icon--${size}`]"
    alt=""
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import personalIcon from '@/assets/knowledge-base-scopes/personal.svg'
import enterpriseIcon from '@/assets/knowledge-base-scopes/enterprise.svg'
import subscribedIcon from '@/assets/knowledge-base-scopes/subscribed.svg'

type KnowledgeBaseScope = 'personal' | 'enterprise' | 'subscribed'

const props = withDefaults(defineProps<{
  scope: KnowledgeBaseScope
  size?: 'small' | 'medium'
}>(), {
  size: 'small',
})

const scopeIconSrc: Record<KnowledgeBaseScope, string> = {
  personal: personalIcon,
  enterprise: enterpriseIcon,
  subscribed: subscribedIcon,
}

const iconSrc = computed(() => scopeIconSrc[props.scope])
</script>

<style scoped lang="less">
.knowledge-base-scope-icon {
  display: block;
  flex-shrink: 0;
  object-fit: contain;

  &.knowledge-base-scope-icon--small {
    width: 16px;
    height: 16px;
  }

  &.knowledge-base-scope-icon--medium {
    width: 20px;
    height: 20px;
  }
}
</style>
