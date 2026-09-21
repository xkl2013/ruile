<template>
  <section
    class="expert-intake-panel"
    :class="{ 'is-submitted': submitted, 'is-submitting': submitting }"
  >
    <header class="expert-intake-panel__header">
      <span class="expert-intake-panel__icon" aria-hidden="true">
        <svg
          v-if="submitted"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2.1"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M20 6 9 17l-5-5" />
        </svg>
        <svg
          v-else
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.9"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path d="M8 3H6a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V5a2 2 0 0 0-2-2h-2" />
          <rect x="8" y="2" width="8" height="4" rx="1" />
          <path d="M8 12h.01M12 12h.01M16 12h.01M8 16h.01M12 16h.01M16 16h.01" />
        </svg>
      </span>

      <div class="expert-intake-panel__heading">
        <strong>{{ submitted ? '已提交' : title }}</strong>
        <span v-if="submitted" class="expert-intake-panel__summary">
          {{ submittedSummary }}
        </span>
        <span v-else>{{ hint }}</span>
      </div>

      <span class="expert-intake-panel__state">
        {{ submitted ? '已确认' : submitting ? '正在提交' : `${questions.length} 题 · 等你填` }}
      </span>
    </header>

    <div class="expert-intake-panel__questions">
      <div
        v-for="(question, index) in questions"
        :key="question.id || index"
        class="expert-intake-question"
      >
        <div class="expert-intake-question__header">
          <span class="expert-intake-question__chip">{{ index + 1 }}</span>
          <div class="expert-intake-question__copy">
            <span class="expert-intake-question__label">
              {{ question.label || `问题 ${index + 1}` }}
            </span>
            <span v-if="question.description" class="expert-intake-question__description">
              {{ question.description }}
            </span>
          </div>
          <span v-if="question.required" class="expert-intake-question__required">必填</span>
          <span v-if="isMultiChoice(question)" class="expert-intake-question__multi">可多选</span>
        </div>

        <div v-if="isChoiceQuestion(question)" class="expert-intake-options">
          <button
            v-for="option in optionsFor(question)"
            :key="option.value"
            type="button"
            class="expert-intake-option"
            :class="{ 'is-selected': isSelected(question, option.value) }"
            :disabled="submitting || submitted"
            :aria-pressed="isSelected(question, option.value)"
            @click="toggleOption(question, option.value)"
          >
            <span
              class="expert-intake-option__mark"
              :class="{
                'is-multiple': isMultiChoice(question),
                'is-selected': isSelected(question, option.value),
              }"
            >
              <svg
                v-if="isMultiChoice(question) && isSelected(question, option.value)"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="3"
                stroke-linecap="round"
                stroke-linejoin="round"
              >
                <path d="M20 6 9 17l-5-5" />
              </svg>
              <i v-else />
            </span>
            <span class="expert-intake-option__copy">
              <span class="expert-intake-option__label">{{ option.label }}</span>
              <span v-if="option.description" class="expert-intake-option__description">
                {{ option.description }}
              </span>
            </span>
          </button>

          <span v-if="!optionsFor(question).length" class="expert-intake-question__empty">
            当前问题没有预置选项，请填写补充说明。
          </span>
        </div>

        <textarea
          v-if="normalizedType(question) === 'text'"
          v-model="answers[question.id]"
          class="expert-intake-input expert-intake-textarea"
          :disabled="submitting || submitted"
          :placeholder="question.description || '请输入补充信息'"
          rows="2"
        />
        <input
          v-else-if="normalizedType(question) === 'date'"
          v-model="answers[question.id]"
          class="expert-intake-input"
          type="date"
          :disabled="submitting || submitted"
        />
        <input
          v-else-if="normalizedType(question) === 'number'"
          v-model="answers[question.id]"
          class="expert-intake-input"
          type="number"
          :disabled="submitting || submitted"
          placeholder="请输入数字"
        />
      </div>
    </div>

    <footer class="expert-intake-panel__actions">
      <span v-if="submitted" class="expert-intake-panel__nudge">
        已将补充信息提交，专家正在继续执行
      </span>
      <span v-else-if="missing.length" class="expert-intake-panel__nudge">
        还差 <b>{{ missing.join('、') }}</b>
      </span>
      <span v-else class="expert-intake-panel__nudge">填完一起提交</span>

      <button
        v-if="!submitted"
        type="button"
        class="expert-intake-panel__submit"
        :disabled="submitting || Boolean(missing.length)"
        @click="submit"
      >
        {{ submitting ? '提交中…' : '提交' }}
      </button>
    </footer>
  </section>
</template>

<script setup>
import { computed, reactive, watch } from 'vue';

const props = defineProps({
  interaction: {
    type: Object,
    default: () => ({}),
  },
  submittedAnswers: {
    type: Object,
    default: null,
  },
  submittedSummary: {
    type: String,
    default: '',
  },
  submitting: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['submit']);
const answers = reactive({});

const submitted = computed(() => Boolean(
  props.submittedAnswers && Object.keys(props.submittedAnswers).length,
));

const questions = computed(() => {
  const list = props.interaction?.questions;
  return Array.isArray(list) ? list : [];
});

const title = computed(() => props.interaction?.title || '需要你补充一下');
const hint = computed(() =>
  props.interaction?.hint || '补充这些信息后，我会继续生成结果。',
);

const normalizedType = (question) => {
  const type = String(question?.type || 'text').toLowerCase();
  if (['enum', 'select', 'choice'].includes(type)) return 'single_choice';
  if (['array', 'multi_enum', 'multi-select', 'multiselect'].includes(type)) return 'multi_choice';
  return type;
};

const isChoiceQuestion = (question) => {
  const type = normalizedType(question);
  return type === 'single_choice' || type === 'multi_choice';
};

const isMultiChoice = (question) => normalizedType(question) === 'multi_choice';

const optionsFor = (question) => {
  if (!Array.isArray(question?.options)) return [];
  return question.options
    .map((option) => {
      if (typeof option === 'string') {
        return { value: option, label: option, description: '' };
      }
      const value = option?.value ?? option?.label ?? '';
      return {
        value: String(value),
        label: option?.label ?? String(value),
        description: option?.description ?? option?.desc ?? '',
      };
    })
    .filter((option) => option.value !== '');
};

const isBlank = (value) => {
  if (Array.isArray(value)) return value.length === 0;
  return value === undefined || value === null || String(value).trim() === '';
};

const missing = computed(() =>
  questions.value
    .filter((question) => question.required && isBlank(answers[question.id]))
    .map((question, index) => question.label || `问题 ${index + 1}`),
);

const isSelected = (question, optionValue) => {
  const value = submitted.value ? props.submittedAnswers?.[question.id] : answers[question.id];
  return isMultiChoice(question)
    ? Array.isArray(value) && value.includes(optionValue)
    : value === optionValue;
};

const toggleOption = (question, optionValue) => {
  if (props.submitting || submitted.value) return;

  if (isMultiChoice(question)) {
    const current = Array.isArray(answers[question.id]) ? answers[question.id] : [];
    answers[question.id] = current.includes(optionValue)
      ? current.filter((item) => item !== optionValue)
      : [...current, optionValue];
    return;
  }

  answers[question.id] = optionValue;
  if (questions.value.length === 1 && question.required) submit();
};

const submit = () => {
  if (props.submitting || missing.value.length) return;

  const payload = {};
  questions.value.forEach((question) => {
    const value = answers[question.id];
    if (!isBlank(value)) payload[question.id] = value;
  });
  if (Object.keys(payload).length > 0) emit('submit', payload);
};

watch(
  () => props.interaction,
  () => {
    Object.keys(answers).forEach((key) => delete answers[key]);
  },
  { deep: true },
);
</script>

<style scoped lang="less">
.expert-intake-panel {
  width: min(720px, 100%);
  box-sizing: border-box;
  margin: 8px 0;
  padding: 10px 12px 11px;
  border: 1px solid rgba(63, 107, 79, .28);
  border-radius: 10px;
  background: #fbfcfa;
  color: var(--td-text-color-primary);
  font-size: 12.5px;
}

.expert-intake-panel.is-submitted {
  border-color: rgba(0, 0, 0, .07);
  background: #fafaf8;
}

.expert-intake-panel.is-submitting {
  opacity: .82;
}

.expert-intake-panel__header {
  display: flex;
  align-items: center;
  gap: 7px;
  color: #3f6b4f;
}

.is-submitted .expert-intake-panel__header {
  color: #8a8a85;
}

.expert-intake-panel__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 15px;
  height: 15px;
  flex: 0 0 15px;
}

.expert-intake-panel__icon svg {
  width: 14px;
  height: 14px;
}

.expert-intake-panel__heading {
  display: flex;
  min-width: 0;
  flex: 1;
  align-items: baseline;
  gap: 8px;
}

.expert-intake-panel__heading strong {
  flex: 0 0 auto;
  color: #3f6b4f;
  font-weight: 600;
}

.is-submitted .expert-intake-panel__heading strong {
  color: #8a8a85;
}

.expert-intake-panel__heading > span {
  min-width: 0;
  overflow: hidden;
  color: #a0a09a;
  line-height: 1.5;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expert-intake-panel__summary {
  color: #3f6b4f !important;
}

.expert-intake-panel__state {
  flex: 0 0 auto;
  margin-left: auto;
  color: #a8a8a2;
  font-size: 11.5px;
  white-space: nowrap;
}

.expert-intake-panel__questions {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 10px;
}

.expert-intake-panel.is-submitted .expert-intake-panel__questions {
  gap: 9px;
}

.expert-intake-question__header {
  display: flex;
  align-items: baseline;
  gap: 7px;
  margin-bottom: 7px;
}

.expert-intake-question__chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  padding: 3px 7px;
  border-radius: 999px;
  background: rgba(63, 107, 79, .1);
  color: #3f6b4f;
  font-size: 10.5px;
  line-height: 1;
}

.is-submitted .expert-intake-question__chip {
  background: rgba(0, 0, 0, .05);
  color: #8a8a85;
}

.expert-intake-question__copy {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 2px;
}

.expert-intake-question__label {
  color: #1f1f1d;
  line-height: 1.6;
}

.expert-intake-question__description {
  color: #a0a09a;
  font-size: 11.5px;
  line-height: 1.5;
}

.expert-intake-question__required,
.expert-intake-question__multi {
  flex: 0 0 auto;
  font-size: 10.5px;
}

.expert-intake-question__required {
  padding: 2px 5px;
  border-radius: 4px;
  background: #fff3e7;
  color: #9b5b24;
}

.expert-intake-question__multi {
  padding: 1px 5px;
  border: 1px solid rgba(0, 0, 0, .12);
  border-radius: 4px;
  color: #a0a09a;
}

.expert-intake-options {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.expert-intake-option {
  display: flex;
  align-items: flex-start;
  width: 100%;
  min-height: 38px;
  box-sizing: border-box;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid rgba(0, 0, 0, .09);
  border-radius: 8px;
  color: #1f1f1d;
  background: #fff;
  cursor: pointer;
  font: inherit;
  font-size: 12.5px;
  text-align: left;
  transition: border-color .15s, background .15s, transform .08s;
}

.expert-intake-option:not(:disabled):hover {
  border-color: rgba(63, 107, 79, .5);
  background: #f6faf7;
}

.expert-intake-option:not(:disabled):active {
  transform: scale(.995);
}

.expert-intake-option:disabled {
  cursor: default;
}

.expert-intake-option:disabled:not(.is-selected) {
  opacity: .5;
  background: #fbfbf9;
}

.expert-intake-option.is-selected {
  border-color: rgba(63, 107, 79, .5);
  background: #f2f7f3;
}

.expert-intake-option__mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  flex: 0 0 14px;
  box-sizing: border-box;
  margin-top: 2px;
  border: 1.5px solid #c9c9c3;
  border-radius: 50%;
  color: #fff;
  transition: border-color .15s, background .15s;
}

.expert-intake-option__mark.is-multiple {
  border-radius: 4px;
}

.expert-intake-option__mark:not(.is-selected) i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: transparent;
  transition: background .15s;
}

.expert-intake-option:not(:disabled):hover .expert-intake-option__mark {
  border-color: #3f6b4f;
}

.expert-intake-option:not(:disabled):hover .expert-intake-option__mark:not(.is-multiple) i {
  background: rgba(63, 107, 79, .35);
}

.expert-intake-option__mark.is-selected:not(.is-multiple) {
  border-width: 2px;
  border-color: #3f6b4f;
}

.expert-intake-option__mark.is-selected:not(.is-multiple) i {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #3f6b4f;
}

.expert-intake-option__mark.is-selected.is-multiple {
  border-color: #3f6b4f;
  background: #3f6b4f;
}

.expert-intake-option__mark svg {
  width: 9px;
  height: 9px;
}

.expert-intake-option__copy {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.expert-intake-option__label {
  line-height: 1.5;
}

.expert-intake-option__description {
  color: #a0a09a;
  font-size: 11.5px;
  line-height: 1.5;
}

.expert-intake-input {
  width: 100%;
  box-sizing: border-box;
  min-height: 38px;
  padding: 8px 10px;
  border: 1px solid rgba(0, 0, 0, .12);
  border-radius: 8px;
  outline: none;
  color: #1f1f1d;
  background: #fff;
  font: inherit;
  font-size: 12.5px;
}

.expert-intake-input:focus {
  border-color: rgba(63, 107, 79, .5);
  box-shadow: 0 0 0 2px rgba(63, 107, 79, .1);
}

.expert-intake-textarea {
  resize: vertical;
  line-height: 1.6;
}

.expert-intake-question__empty {
  padding: 4px 2px;
  color: #a8a8a2;
  font-size: 11.5px;
}

.expert-intake-panel__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 37px;
  margin-top: 12px;
  padding-top: 9px;
  border-top: 1px solid rgba(0, 0, 0, .08);
}

.expert-intake-panel__nudge {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: #a0a09a;
  font-size: 11.5px;
  line-height: 1.5;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.expert-intake-panel__nudge b {
  color: #8d6a4e;
  font-weight: 500;
}

.expert-intake-panel__submit {
  min-width: 64px;
  min-height: 32px;
  padding: 6px 13px;
  border: 0;
  border-radius: 6px;
  color: #fff;
  background: #3f6b4f;
  cursor: pointer;
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
}

.expert-intake-panel__submit:disabled {
  cursor: not-allowed;
  opacity: .45;
}

@media (max-width: 640px) {
  .expert-intake-panel__heading {
    align-items: flex-start;
    flex-direction: column;
    gap: 2px;
  }

  .expert-intake-panel__heading > span {
    white-space: normal;
  }

  .expert-intake-panel__state {
    align-self: flex-start;
  }

  .expert-intake-panel__actions {
    align-items: stretch;
    flex-direction: column;
  }

  .expert-intake-panel__nudge {
    white-space: normal;
  }

  .expert-intake-panel__submit {
    width: 100%;
  }
}
</style>
