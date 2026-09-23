<template>
  <section class="published-expert-route-choice" :class="{ 'is-complete': message.confirmed || message.cancelled }">
    <div class="published-expert-route-choice__header">
      <div>
        <div class="published-expert-route-choice__eyebrow">专家路由</div>
        <strong>{{ message.confirmed ? '已选择专家' : message.cancelled ? '已取消专家执行' : '请选择处理专家' }}</strong>
      </div>
      <span v-if="!message.confirmed && !message.cancelled" class="published-expert-route-choice__confidence">
        匹配度 {{ confidenceLabel }}
      </span>
    </div>

    <p v-if="message.routeDecision?.routing_reason && !message.confirmed && !message.cancelled"
      class="published-expert-route-choice__reason">
      {{ message.routeDecision.routing_reason }}
    </p>

    <div v-if="!message.confirmed && !message.cancelled" class="published-expert-route-choice__options">
      <button
        v-for="candidate in candidates"
        :key="candidate.definition_id"
        type="button"
        class="published-expert-route-choice__option"
        :disabled="message.confirming"
        @click="emit('select', candidate)"
      >
        <AgentAvatar
          class="published-expert-route-choice__avatar"
          :name="candidate.display_name || '专家'"
          :avatar="candidate.avatar"
          size="medium"
        />
        <span class="published-expert-route-choice__copy">
          <strong>{{ candidate.display_name || '未命名专家' }}</strong>
          <small>{{ candidate.description || candidate.domain || '已发布专家' }}</small>
        </span>
        <t-icon :name="message.confirming ? 'loading' : 'chevron-right'" />
      </button>
    </div>

    <button
      v-if="!message.confirmed && !message.cancelled"
      type="button"
      class="published-expert-route-choice__cancel"
      :disabled="message.confirming"
      @click="emit('cancel')"
    >
      暂不使用专家
    </button>
  </section>
</template>

<script setup>
import { computed } from 'vue';
import AgentAvatar from '@/components/AgentAvatar.vue';

const props = defineProps({
  message: {
    type: Object,
    required: true,
  },
});

const emit = defineEmits(['select', 'cancel']);

const candidates = computed(() => {
  const list = props.message?.routeDecision?.candidates;
  return Array.isArray(list) ? list.slice(0, 3) : [];
});

const confidenceLabel = computed(() => {
  const value = Number(props.message?.routeDecision?.confidence || 0);
  return `${Math.round(Math.max(0, Math.min(1, value)) * 100)}%`;
});
</script>

<style scoped>
.published-expert-route-choice {
  width: min(100%, 620px);
  margin: 10px 0 18px;
  padding: 2px 0;
  color: #1f2937;
}

.published-expert-route-choice__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.published-expert-route-choice__eyebrow {
  margin-bottom: 4px;
  color: #7b8492;
  font-size: 12px;
}

.published-expert-route-choice__header strong {
  font-size: 15px;
  font-weight: 600;
}

.published-expert-route-choice__confidence {
  color: #7b8492;
  font-size: 12px;
  white-space: nowrap;
}

.published-expert-route-choice__reason {
  margin: 10px 0 12px;
  color: #667085;
  font-size: 13px;
  line-height: 1.6;
}

.published-expert-route-choice__options {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.published-expert-route-choice__option {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 10px 11px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
  color: inherit;
  text-align: left;
  cursor: pointer;
  transition: border-color 0.16s ease, background 0.16s ease;
}

.published-expert-route-choice__option:hover:not(:disabled) {
  border-color: #16a765;
  background: #f5fbf7;
}

.published-expert-route-choice__option:disabled {
  cursor: wait;
  opacity: 0.65;
}

.published-expert-route-choice__avatar {
  flex: 0 0 28px;
  width: 28px;
  height: 28px;
  border-radius: 7px;
  box-shadow: none;
}

.published-expert-route-choice__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.published-expert-route-choice__copy strong,
.published-expert-route-choice__copy small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.published-expert-route-choice__copy strong {
  font-size: 13px;
  font-weight: 600;
}

.published-expert-route-choice__copy small {
  color: #8a93a1;
  font-size: 11px;
}

.published-expert-route-choice__option :deep(.t-icon) {
  flex: 0 0 auto;
  color: #9aa3af;
}

.published-expert-route-choice__cancel {
  margin-top: 10px;
  padding: 0;
  border: 0;
  background: transparent;
  color: #8a93a1;
  font-size: 12px;
  cursor: pointer;
}

.published-expert-route-choice__cancel:hover:not(:disabled) {
  color: #4b5563;
}

.published-expert-route-choice__cancel:disabled {
  cursor: wait;
  opacity: 0.6;
}

@media (max-width: 640px) {
  .published-expert-route-choice__options {
    grid-template-columns: 1fr;
  }
}
</style>
