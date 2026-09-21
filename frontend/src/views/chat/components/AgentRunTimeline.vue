<template>
  <section v-if="eventRows.length" class="agent-run-timeline" :class="{ 'is-active': timeline.active }">
    <button type="button" class="agent-run-timeline__header" :aria-expanded="expanded" @click="toggleExpanded">
      <span class="agent-run-timeline__leading" aria-hidden="true">
        <t-icon :name="timeline.active ? 'loading' : timeline.failedCount ? 'close' : 'check'" />
      </span>
      <span class="agent-run-timeline__heading">
        <strong>{{ timeline.active ? activeLabel : '执行事件流' }}</strong>
        <small>{{ summaryLabel }}</small>
      </span>
      <span v-if="totalDurationLabel" class="agent-run-timeline__duration">{{ totalDurationLabel }}</span>
      <t-icon class="agent-run-timeline__chevron" :name="expanded ? 'chevron-up' : 'chevron-down'" />
    </button>

    <div v-if="expanded" class="agent-run-timeline__body">
      <article v-for="event in eventRows" :key="event.id" class="agent-run-event" :class="`is-${event.status}`">
        <component
          :is="event.details.length ? 'button' : 'div'"
          :type="event.details.length ? 'button' : undefined"
          class="agent-run-event__line"
          :class="{ 'is-expandable': event.details.length }"
          :aria-expanded="event.details.length ? isEventDetailsExpanded(event) : undefined"
          @click="event.details.length && toggleEventDetails(event)"
        >
          <span class="agent-run-event__icon" aria-hidden="true">
            <t-icon :name="event.status === 'running' ? 'loading' : event.status === 'failed' ? 'close' : 'check'" />
          </span>
          <span class="agent-run-event__copy">
            <span class="agent-run-event__title">
              <strong>{{ event.label }}</strong>
              <code>{{ event.type }}</code>
              <t-icon
                v-if="event.details.length"
                class="agent-run-event__details-chevron"
                :name="isEventDetailsExpanded(event) ? 'chevron-up' : 'chevron-down'"
                aria-hidden="true"
              />
            </span>
            <small v-if="event.summary">{{ event.summary }}</small>
          </span>
          <span v-if="event.durationMs" class="agent-run-event__duration">
            {{ formatAgentRunDuration(event.durationMs) }}
          </span>
        </component>
        <div v-if="event.details.length && isEventDetailsExpanded(event)" class="agent-run-event__details">
          <section
            v-for="(detail, detailIndex) in event.details"
            :key="`${event.id}-${detail.label}-${detailIndex}`"
            class="agent-run-event__detail"
            :class="{ 'is-error': detail.tone === 'error' }"
          >
            <span>{{ detail.label }}</span>
            <pre>{{ formatAgentRunEventValue(detail.value) }}</pre>
          </section>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import {
  agentRunPhaseLabel,
  buildAgentRunEventRows,
  buildAgentRunTimeline,
  formatAgentRunEventValue,
  formatAgentRunDuration,
} from '@/utils/agentRunTimeline';

const props = defineProps({
  events: {
    type: Array,
    default: () => [],
  },
  run: {
    type: Object,
    default: () => ({}),
  },
  steps: {
    type: Array,
    default: () => [],
  },
});

const expanded = ref(['queued', 'running', 'failed'].includes(props.run?.status));
const eventExpansionOverrides = ref(new Map());
const now = ref(Date.now());
let collapseTimer = 0;
const timer = window.setInterval(() => {
  if (['queued', 'running'].includes(props.run?.status)) now.value = Date.now();
}, 1000);

const timeline = computed(() => buildAgentRunTimeline(props.events, props.run, now.value));
const eventRows = computed(() => buildAgentRunEventRows(props.events, props.steps));
const hasTerminalEvent = computed(() => props.events.some((event) => (
  ['RUN_FINISHED', 'RUN_ERROR', 'RUN_CANCELLED'].includes(String(event?.type || ''))
)));
const executionActive = computed(() => (
  ['queued', 'running'].includes(props.run?.status) && !hasTerminalEvent.value
));
const activeLabel = computed(() => agentRunPhaseLabel(props.run?.phase));
const totalDurationLabel = computed(() => formatAgentRunDuration(timeline.value.totalDurationMs));
const summaryLabel = computed(() => {
  const eventCount = eventRows.value.length;
  if (timeline.value.active) return `${eventCount} 个事件 · ${timeline.value.completedCount} 个步骤已完成`;
  if (timeline.value.failedCount) return `${eventCount} 个事件 · ${timeline.value.failedCount} 个步骤失败`;
  return `${eventCount} 个事件 · ${timeline.value.completedCount} 个步骤已完成`;
});

const isEventDetailsExpanded = (event) => {
  if (eventExpansionOverrides.value.has(event.id)) {
    return eventExpansionOverrides.value.get(event.id);
  }
  return event.status !== 'succeeded';
};

const toggleEventDetails = (event) => {
  if (!event.details.length) return;
  const next = new Map(eventExpansionOverrides.value);
  next.set(event.id, !isEventDetailsExpanded(event));
  eventExpansionOverrides.value = next;
};

const toggleExpanded = () => {
  expanded.value = !expanded.value;
  if (collapseTimer) window.clearTimeout(collapseTimer);
};

watch(
  () => eventRows.value.map((event) => event.id),
  (eventIds) => {
    const allowedIds = new Set(eventIds);
    const next = new Map(
      [...eventExpansionOverrides.value].filter(([eventId]) => allowedIds.has(eventId)),
    );
    if (next.size !== eventExpansionOverrides.value.size) {
      eventExpansionOverrides.value = next;
    }
  },
  { immediate: true },
);

watch(
  [executionActive, hasTerminalEvent],
  ([active, ended], [previousActive, previousEnded] = []) => {
    if (collapseTimer) window.clearTimeout(collapseTimer);
    if (active) {
      expanded.value = true;
      return;
    }
    if (previousActive || (ended && !previousEnded)) {
      collapseTimer = window.setTimeout(() => {
        expanded.value = false;
      }, 400);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  window.clearInterval(timer);
  if (collapseTimer) window.clearTimeout(collapseTimer);
});
</script>

<style scoped lang="less">
.agent-run-timeline {
  margin: 4px 0 14px;
}

.agent-run-timeline__header {
  display: flex;
  width: fit-content;
  max-width: 100%;
  min-height: 30px;
  align-items: center;
  gap: 7px;
  padding: 3px 0;
  border: 0;
  color: var(--td-text-color-secondary);
  background: transparent;
  cursor: pointer;
  text-align: left;
}

.agent-run-timeline__header:hover .agent-run-timeline__heading strong {
  color: var(--td-text-color-primary);
}

.agent-run-timeline__leading {
  display: inline-flex;
  width: 15px;
  height: 15px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  color: var(--td-success-color);
  font-size: 14px;
}

.is-active .agent-run-timeline__leading {
  color: #4c785c;
}

.is-active .agent-run-timeline__leading :deep(svg) {
  animation: agentRunTimelineSpin .9s linear infinite;
}

.agent-run-timeline__heading {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 7px;
}

.agent-run-timeline__heading strong {
  color: var(--td-text-color-secondary);
  font-size: 12.5px;
  font-weight: 500;
  transition: color .15s ease;
}

.agent-run-timeline__heading small,
.agent-run-timeline__duration,
.agent-run-timeline__step-duration {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.agent-run-timeline__duration {
  margin-left: 2px;
  white-space: nowrap;
}

.agent-run-timeline__chevron {
  width: 14px;
  height: 14px;
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
}

.agent-run-timeline__body {
  margin: 3px 0 0 6px;
  padding: 2px 0 2px 14px;
  border-left: 1px solid color-mix(in srgb, var(--td-component-border) 74%, transparent);
}

.agent-run-event {
  padding: 5px 0 7px;
}

.agent-run-event + .agent-run-event {
  margin-top: 1px;
}

.agent-run-event__line {
  display: grid;
  box-sizing: border-box;
  width: 100%;
  grid-template-columns: 16px minmax(0, 1fr) auto;
  align-items: start;
  gap: 7px;
  padding: 0;
  border: 0;
  color: inherit;
  background: transparent;
  font: inherit;
  text-align: left;
}

.agent-run-event__line.is-expandable {
  cursor: pointer;
}

.agent-run-event__line.is-expandable:hover .agent-run-event__title strong {
  color: var(--td-text-color-primary);
}

.agent-run-event__icon {
  display: inline-flex;
  width: 15px;
  height: 20px;
  align-items: center;
  justify-content: center;
  color: var(--td-success-color);
  font-size: 12px;
}

.agent-run-event.is-running .agent-run-event__icon {
  color: #4c785c;
}

.agent-run-event.is-running .agent-run-event__icon :deep(svg) {
  animation: agentRunTimelineSpin .9s linear infinite;
}

.agent-run-event.is-failed .agent-run-event__icon,
.agent-run-event.is-failed .agent-run-event__title strong {
  color: var(--td-error-color);
}

.agent-run-event__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.agent-run-event__title {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 7px;
}

.agent-run-event__title strong {
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: 12.5px;
  font-weight: 500;
}

.agent-run-event__title code {
  overflow: hidden;
  padding: 1px 5px;
  border-radius: 4px;
  color: var(--td-text-color-placeholder);
  background: var(--td-bg-color-secondarycontainer);
  font-family: "SFMono-Regular", Consolas, monospace;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-run-event__details-chevron {
  width: 13px;
  height: 13px;
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
}

.agent-run-event__copy small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.agent-run-event__duration {
  padding-top: 2px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  white-space: nowrap;
}

.agent-run-event__details {
  display: grid;
  gap: 7px;
  margin: 6px 0 0 23px;
}

.agent-run-event__detail > span {
  display: block;
  margin-bottom: 3px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.agent-run-event__detail pre {
  overflow: auto;
  max-height: 280px;
  margin: 0;
  padding: 8px 10px;
  border: 0;
  border-radius: 6px;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  font-family: "SFMono-Regular", Consolas, monospace;
  font-size: 11px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.agent-run-event__detail.is-error pre {
  color: var(--td-error-color);
  background: var(--td-error-color-light);
}

@media (max-width: 640px) {
  .agent-run-timeline__heading small,
  .agent-run-event__title code {
    display: none;
  }

  .agent-run-event__details {
    margin-left: 0;
  }
}

@keyframes agentRunTimelineSpin {
  to { transform: rotate(360deg); }
}
</style>
