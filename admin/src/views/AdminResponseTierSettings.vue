<template>
  <section class="admin-response-tiers">
    <div v-if="loading" class="admin-response-tiers__state">
      <t-loading size="small" />
      <span>正在加载回答档位配置</span>
    </div>

    <template v-else>
      <header class="admin-response-tiers__hero">
        <div>
          <h2>回答档位</h2>
          <p>为全平台所有工作区和智能体统一配置快速、均衡、极致模型。工作区无需重复配置。</p>
        </div>
        <t-button
          theme="primary"
          :loading="saving"
          :disabled="!canSave"
          @click="handleSave"
        >
          <template #icon><t-icon name="save" /></template>
          保存配置
        </t-button>
      </header>

      <t-alert
        v-if="!canSave"
        theme="warning"
        variant="light"
        message="当前账号只能查看配置；保存需要系统管理员权限。"
      />

      <section class="admin-response-tiers__policy">
        <div class="admin-response-tiers__policy-copy">
          <strong>启用回答档位</strong>
          <span>启用后，所有工作区均使用这里绑定的模型和 Think 策略。</span>
        </div>
        <t-switch v-model="form.enabled" :disabled="!canSave" />
        <div class="admin-response-tiers__default">
          <label for="response-tier-default">默认档位</label>
          <t-select
            id="response-tier-default"
            v-model="form.default_tier"
            :disabled="!canSave"
            style="width: 160px"
          >
            <t-option value="fast" label="快速" />
            <t-option value="balanced" label="均衡" />
            <t-option value="ultimate" label="极致" />
          </t-select>
        </div>
      </section>

      <t-alert
        v-if="!allModels.length"
        theme="warning"
        variant="light"
        message="当前没有可用的对话模型，请先在“模型”中完成配置。"
      />

      <section class="admin-response-tiers__grid">
        <article
          v-for="tier in tiers"
          :key="tier.key"
          class="admin-response-tier"
        >
          <header class="admin-response-tier__header">
            <div>
              <h3>{{ tier.label }}</h3>
              <p>{{ tier.description }}</p>
            </div>
            <t-tag :theme="tier.theme" variant="light">{{ tier.key === form.default_tier ? '默认' : '可选' }}</t-tag>
          </header>

          <div class="admin-response-tier__fields">
            <t-form-item label="聊天模型">
              <ModelSelector
                model-type="KnowledgeQA"
                :selected-model-id="form[tier.key].model_id"
                :all-models="allModels"
                :disabled="!canSave"
                placeholder="选择对话模型"
                @update:selected-model-id="(value: string) => { form[tier.key].model_id = value }"
              />
            </t-form-item>

            <t-form-item label="Think 策略">
              <t-select
                :value="thinkingMode(form[tier.key])"
                :disabled="!canSave"
                @change="(value: string) => setThinking(form[tier.key], value as ThinkingMode)"
              >
                <t-option value="default" label="跟随模型默认" />
                <t-option value="on" label="开启 Think" />
                <t-option value="off" label="关闭 Think" />
              </t-select>
            </t-form-item>
          </div>
        </article>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { listModels, type ModelConfig } from '@/api/model'
import {
  defaultResponseTierConfig,
  getSystemResponseTierConfig,
  updateSystemResponseTierConfig,
  type ResponseTierConfig,
  type ResponseTierProfile,
} from '@/api/response-tier'
import ModelSelector from '@/components/ModelSelector.vue'
import { useAuthStore } from '@/stores/auth'

type TierKey = 'fast' | 'balanced' | 'ultimate'
type ThinkingMode = 'default' | 'on' | 'off'

const authStore = useAuthStore()
const loading = ref(true)
const saving = ref(false)
const allModels = ref<ModelConfig[]>([])
const form = ref<ResponseTierConfig>(cloneConfig(defaultResponseTierConfig))

const tiers: Array<{
  key: TierKey
  label: string
  description: string
  theme: 'primary' | 'success' | 'warning'
}> = [
  {
    key: 'fast',
    label: '快速',
    description: '优先响应速度，适合简单问答和高频操作。',
    theme: 'success',
  },
  {
    key: 'balanced',
    label: '均衡',
    description: '在响应速度、成本和回答质量之间保持平衡。',
    theme: 'primary',
  },
  {
    key: 'ultimate',
    label: '极致',
    description: '优先回答质量和复杂问题处理能力。',
    theme: 'warning',
  },
]

const canSave = computed(() => authStore.isSystemAdmin)

function cloneConfig(config: ResponseTierConfig): ResponseTierConfig {
  return {
    enabled: Boolean(config.enabled),
    default_tier: config.default_tier || 'balanced',
    fast: { model_id: config.fast?.model_id || '', thinking: config.fast?.thinking ?? null },
    balanced: { model_id: config.balanced?.model_id || '', thinking: config.balanced?.thinking ?? null },
    ultimate: { model_id: config.ultimate?.model_id || '', thinking: config.ultimate?.thinking ?? null },
  }
}

function thinkingMode(profile: ResponseTierProfile): ThinkingMode {
  if (profile.thinking === true) return 'on'
  if (profile.thinking === false) return 'off'
  return 'default'
}

function setThinking(profile: ResponseTierProfile, mode: ThinkingMode) {
  profile.thinking = mode === 'default' ? null : mode === 'on'
}

async function load() {
  loading.value = true
  try {
    const [configResult, modelsResult] = await Promise.allSettled([
      getSystemResponseTierConfig(),
      listModels('KnowledgeQA'),
    ])

    if (configResult.status === 'fulfilled') {
      form.value = cloneConfig(configResult.value)
    } else {
      MessagePlugin.error(configResult.reason?.message || '回答档位配置加载失败')
    }

    if (modelsResult.status === 'fulfilled') {
      allModels.value = Array.isArray(modelsResult.value) ? modelsResult.value : []
    } else {
      allModels.value = []
      MessagePlugin.error(modelsResult.reason?.message || '对话模型加载失败')
    }
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!canSave.value) return
  if (
    form.value.enabled &&
    (!form.value.fast.model_id || !form.value.balanced.model_id || !form.value.ultimate.model_id)
  ) {
    MessagePlugin.warning('启用回答档位时，快速、均衡、极致都必须绑定模型')
    return
  }

  saving.value = true
  try {
    const saved = await updateSystemResponseTierConfig(form.value)
    form.value = cloneConfig(saved)
    MessagePlugin.success('回答档位配置已保存')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '回答档位配置保存失败')
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped lang="less">
.admin-response-tiers {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.admin-response-tiers__state {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 240px;
  color: var(--td-text-color-secondary);
}

.admin-response-tiers__hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: flex-end;
}

.admin-response-tiers__hero h2 {
  margin: 10px 0 6px;
  font-size: 24px;
}

.admin-response-tiers__hero p {
  max-width: 760px;
  margin: 0;
  color: var(--td-text-color-secondary);
}

.admin-response-tiers__policy {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 20px;
  align-items: center;
  padding: 18px 20px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.admin-response-tiers__policy-copy {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.admin-response-tiers__policy-copy span,
.admin-response-tiers__default label {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.admin-response-tiers__default {
  display: flex;
  align-items: center;
  gap: 10px;
}

.admin-response-tiers__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.admin-response-tier {
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.admin-response-tier__header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
  margin-bottom: 20px;
}

.admin-response-tier__header h3 {
  margin: 0 0 6px;
  font-size: 18px;
}

.admin-response-tier__header p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.admin-response-tier__fields {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

@media (max-width: 900px) {
  .admin-response-tiers__hero,
  .admin-response-tiers__policy {
    align-items: flex-start;
  }

  .admin-response-tiers__hero {
    flex-direction: column;
  }

  .admin-response-tiers__policy {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .admin-response-tiers__default {
    grid-column: 1 / -1;
  }

  .admin-response-tiers__grid {
    grid-template-columns: 1fr;
  }
}
</style>
