<template>
  <section class="admin-kb-defaults">
    <div v-if="loading" class="admin-kb-state">
      <t-loading size="small" />
      <span>正在加载知识库统一配置</span>
    </div>

    <template v-else>
      <header class="admin-kb-defaults__hero">
        <div>
          <h2>知识库配置</h2>
          <p>当前工作区只维护一份知识库高级默认配置；后续新建知识库自动继承，存量知识库需通过显式迁移变更。</p>
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
        message="当前账号只能查看统一配置；保存并应用需要空间 Admin 或系统管理员权限。"
      />

      <t-alert
        theme="info"
        variant="light"
        message="保存后仅更新当前工作区默认配置，后续新建知识库会自动继承；已有知识库保持不变。名称、描述和目录仍在主工作台维护。"
      />

      <section class="admin-kb-defaults__summary">
        <article>
          <span>当前工作区知识库</span>
          <strong>{{ knowledgeBaseCount }}</strong>
          <em>用于评估迁移影响范围</em>
        </article>
        <article>
          <span>当前配置状态</span>
          <strong>{{ isConfigured ? '完整' : '待完善' }}</strong>
          <em>模型与索引校验</em>
        </article>
        <article>
          <span>生效范围</span>
          <strong>统一</strong>
          <em>后续新建知识库</em>
        </article>
      </section>

      <section class="admin-kb-defaults__layout">
        <aside class="admin-kb-defaults__tabs" aria-label="知识库配置">
          <button
            v-for="tab in tabs"
            :key="tab.key"
            type="button"
            :class="{ active: activeTab === tab.key }"
            @click="activeTab = tab.key"
          >
            <t-icon :name="tab.icon" />
            <span>
              <strong>{{ tab.label }}</strong>
              <small>{{ tab.desc }}</small>
            </span>
          </button>
        </aside>

        <div class="admin-kb-defaults__content">
          <section v-show="activeTab === 'models'" class="admin-kb-section">
            <KBModelConfig
              :config="form.modelConfig"
              :has-files="false"
              :wiki-enabled="form.indexingStrategy.wikiEnabled"
              :rag-enabled="form.indexingStrategy.vectorEnabled || form.indexingStrategy.keywordEnabled"
              :all-models="allModels"
              @update:config="handleModelConfigUpdate"
            />
          </section>

          <section v-show="activeTab === 'processing'" class="admin-kb-section">
            <KBParserSettings
              :parser-engine-rules="form.chunkingConfig.parserEngineRules"
              @update:parser-engine-rules="handleParserEngineRulesUpdate"
            />
            <div class="admin-kb-settings-divider" />
            <KBChunkingSettings
              :config="form.chunkingConfig"
              @update:config="handleChunkingConfigUpdate"
            />
            <div class="admin-kb-settings-divider" />
            <KBAdvancedSettings
              :question-generation="form.questionGenerationConfig"
              :rag-enabled="form.indexingStrategy.vectorEnabled || form.indexingStrategy.keywordEnabled"
              :all-models="allModels"
              :table-metadata-instructions="form.chunkingConfig.tableMetadataInstructions"
              @update:question-generation="handleQuestionGenerationUpdate"
              @update:table-metadata-instructions="handleTableMetadataInstructionsUpdate"
            />
          </section>

          <section v-show="activeTab === 'multimodal'" class="admin-kb-section">
            <div class="admin-kb-section__heading">
              <h3>多模态和音频</h3>
              <p>统一配置图片理解、OCR 和音频转写模型，影响后续新建知识库的入库处理；存量知识库需显式迁移。</p>
            </div>

            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用图片理解</strong>
                  <small>对文档图片生成文字描述，供检索和问答使用。</small>
                </span>
                <t-switch v-model="form.multimodalConfig.enabled" />
              </div>
              <div v-if="form.multimodalConfig.enabled" class="admin-kb-form-grid">
                <t-form-item label="VLM 模型">
                  <ModelSelector
                    model-type="VLLM"
                    :selected-model-id="form.multimodalConfig.vllmModelId"
                    :all-models="allModels"
                    placeholder="选择图片理解模型"
                    @update:selected-model-id="(value: string) => { form.multimodalConfig.vllmModelId = value }"
                  />
                </t-form-item>
                <t-form-item label="描述语言">
                  <t-select v-model="form.multimodalConfig.descriptionLanguage" clearable placeholder="自动">
                    <t-option value="Chinese" label="中文" />
                    <t-option value="English" label="英文" />
                    <t-option value="Korean" label="韩文" />
                    <t-option value="Russian" label="俄文" />
                  </t-select>
                </t-form-item>
              </div>
              <t-form-item v-if="form.multimodalConfig.enabled" label="图片描述指令">
                <t-textarea
                  v-model="form.multimodalConfig.customInstructions"
                  :maxlength="4000"
                  :autosize="{ minRows: 3, maxRows: 8 }"
                  placeholder="定义图片描述的细节、格式和排除项"
                />
              </t-form-item>
            </div>

            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用 OCR 兜底</strong>
                  <small>图片理解不可用或需提取图片文字时使用。</small>
                </span>
                <t-switch v-model="form.ocrConfig.enabled" />
              </div>
              <t-form-item v-if="form.ocrConfig.enabled" label="OCR 模型">
                <ModelSelector
                  model-type="OCR"
                  :selected-model-id="form.ocrConfig.modelId"
                  :all-models="allModels"
                  placeholder="选择 OCR 模型"
                  @update:selected-model-id="(value: string) => { form.ocrConfig.modelId = value }"
                />
              </t-form-item>
            </div>

            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用音频转写</strong>
                  <small>处理录音、音视频文件时生成文本。</small>
                </span>
                <t-switch v-model="form.asrConfig.enabled" />
              </div>
              <div v-if="form.asrConfig.enabled" class="admin-kb-form-grid">
                <t-form-item label="ASR 模型">
                  <ModelSelector
                    model-type="ASR"
                    :selected-model-id="form.asrConfig.modelId"
                    :all-models="allModels"
                    placeholder="选择 ASR 模型"
                    @update:selected-model-id="(value: string) => { form.asrConfig.modelId = value }"
                  />
                </t-form-item>
                <t-form-item label="语言">
                  <t-input v-model="form.asrConfig.language" clearable placeholder="可选，例如 zh、en" />
                </t-form-item>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'graph'" class="admin-kb-section">
            <GraphSettings
              :graph-extract="form.nodeExtractConfig"
              :model-id="form.modelConfig.llmModelId"
              :all-models="allModels"
              @update:graph-extract="handleNodeExtractUpdate"
            />
          </section>

          <section v-show="activeTab === 'storage'" class="admin-kb-section">
            <KBStorageSettings
              :storage-backend-id="form.storageBackendId"
              :storage-provider="form.storageProvider"
              :has-files="false"
              @update:storage-backend-id="(value: string) => { form.storageBackendId = value }"
              @update:storage-provider="(value: string) => { form.storageProvider = value }"
            />
          </section>
        </div>
      </section>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  getKnowledgeBaseDefaults,
  listKnowledgeBases,
  updateKnowledgeBaseDefaults,
  type KnowledgeBaseDefaultsConfig,
} from '@/api/knowledge-base'
import { listModels } from '@/api/model'
import { createDefaultKnowledgeBaseFormData, cloneDefaultParserEngineRules } from '@/config/knowledgeBaseDefaults'
import { useAuthStore } from '@/stores/auth'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useRoute } from 'vue-router'
import KBModelConfig from '@/views/knowledge/settings/KBModelConfig.vue'
import KBParserSettings from '@/views/knowledge/settings/KBParserSettings.vue'
import KBChunkingSettings from '@/views/knowledge/settings/KBChunkingSettings.vue'
import KBAdvancedSettings from '@/views/knowledge/settings/KBAdvancedSettings.vue'
import KBStorageSettings from '@/views/knowledge/settings/KBStorageSettings.vue'
import GraphSettings from '@/views/knowledge/settings/GraphSettings.vue'
import ModelSelector from '@/components/ModelSelector.vue'

const route = useRoute()
const authStore = useAuthStore()
const chatResources = useChatResourcesStore()
const form = ref<any>(createDefaultKnowledgeBaseFormData('document'))
const allModels = ref<any[]>([])
const knowledgeBaseCount = ref(0)
const loading = ref(true)
const saving = ref(false)
const activeTab = ref(String(route.query.tab || 'models'))

const tabs = [
  { key: 'models', label: '模型绑定', desc: 'LLM、Embedding、Wiki', icon: 'control-platform' },
  { key: 'processing', label: '解析分块', desc: 'Parser、Chunking、高级项', icon: 'file-setting' },
  { key: 'multimodal', label: '多模态', desc: '图片、OCR、ASR', icon: 'image' },
  { key: 'graph', label: '知识图谱', desc: '实体和关系提取', icon: 'chart-bubble' },
  { key: 'storage', label: '存储绑定', desc: '对象存储实例', icon: 'data-base' },
]

const canSave = computed(() => authStore.hasRole('admin') || authStore.isSystemAdmin)
const isConfigured = computed(() => {
  const strategy = form.value.indexingStrategy
  return Boolean(
    form.value.modelConfig.llmModelId &&
    (!strategy.vectorEnabled && !strategy.keywordEnabled || form.value.modelConfig.embeddingModelId),
  )
})

function mapConfig(source: any) {
  const defaults = createDefaultKnowledgeBaseFormData('document')
  const chunk = source?.chunking_config || {}
  return {
    ...defaults,
    modelConfig: {
      llmModelId: source?.summary_model_id || '',
      embeddingModelId: source?.embedding_model_id || '',
      wikiSynthesisModelId: source?.wiki_config?.synthesis_model_id || '',
    },
    chunkingConfig: {
      ...defaults.chunkingConfig,
      ...chunk,
      separators: chunk.separators || defaults.chunkingConfig.separators,
      parserEngineRules: chunk.parser_engine_rules?.length
        ? chunk.parser_engine_rules
        : cloneDefaultParserEngineRules(),
    },
    storageBackendId: source?.storage_backend_id || '',
    storageProvider: source?.storage_provider || '',
    multimodalConfig: {
      enabled: !!source?.vlm_config?.enabled,
      vllmModelId: source?.vlm_config?.model_id || '',
      descriptionLanguage: source?.vlm_config?.description_language || '',
      customInstructions: source?.vlm_config?.custom_instructions || '',
    },
    ocrConfig: {
      enabled: !!source?.ocr_config?.enabled,
      modelId: source?.ocr_config?.model_id || '',
    },
    asrConfig: {
      enabled: !!source?.asr_config?.enabled,
      modelId: source?.asr_config?.model_id || '',
      language: source?.asr_config?.language || '',
    },
    nodeExtractConfig: {
      enabled: !!source?.extract_config?.enabled,
      text: source?.extract_config?.text || '',
      tags: source?.extract_config?.tags || [],
      nodes: source?.extract_config?.nodes || [],
      relations: source?.extract_config?.relations || [],
      customInstructions: source?.extract_config?.custom_instructions || '',
    },
    questionGenerationConfig: {
      enabled: source?.question_generation_config?.enabled ?? defaults.questionGenerationConfig.enabled,
      questionCount: source?.question_generation_config?.question_count ?? defaults.questionGenerationConfig.questionCount,
      customInstructions: source?.question_generation_config?.custom_instructions || '',
    },
    wikiConfig: {
      ...defaults.wikiConfig,
      ...(source?.wiki_config || {}),
      synthesisModelId: source?.wiki_config?.synthesis_model_id || '',
      maxPagesPerIngest: source?.wiki_config?.max_pages_per_ingest || 0,
      extractionGranularity: source?.wiki_config?.extraction_granularity || 'standard',
      contentInstructions: source?.wiki_config?.content_instructions || '',
      extractionInstructions: source?.wiki_config?.extraction_instructions || '',
    },
    indexingStrategy: {
      ...defaults.indexingStrategy,
      ...(source?.indexing_strategy || {}),
    },
  }
}

function buildPayload(): KnowledgeBaseDefaultsConfig {
  const chunk = form.value.chunkingConfig
  return {
    summary_model_id: form.value.modelConfig.llmModelId,
    embedding_model_id: form.value.modelConfig.embeddingModelId,
    vlm_config: {
      enabled: !!form.value.multimodalConfig.enabled,
      model_id: form.value.multimodalConfig.enabled ? form.value.multimodalConfig.vllmModelId || '' : '',
      description_language: form.value.multimodalConfig.descriptionLanguage || '',
      custom_instructions: form.value.multimodalConfig.customInstructions || '',
    },
    ocr_config: {
      enabled: !!form.value.ocrConfig.enabled,
      model_id: form.value.ocrConfig.enabled ? form.value.ocrConfig.modelId || '' : '',
    },
    asr_config: {
      enabled: !!form.value.asrConfig.enabled,
      model_id: form.value.asrConfig.enabled ? form.value.asrConfig.modelId || '' : '',
      language: form.value.asrConfig.language || '',
    },
    chunking_config: {
      chunk_size: chunk.chunkSize,
      chunk_overlap: chunk.chunkOverlap,
      separators: chunk.separators || [],
      parser_engine_rules: chunk.parserEngineRules || [],
      enable_parent_child: !!chunk.enableParentChild,
      parent_chunk_size: chunk.parentChunkSize || 4096,
      child_chunk_size: chunk.childChunkSize || 384,
      strategy: chunk.strategy || '',
      token_limit: chunk.tokenLimit || 0,
      languages: chunk.languages || [],
      table_metadata_instructions: chunk.tableMetadataInstructions || '',
    },
    storage_provider: form.value.storageProvider || '',
    storage_backend_id: form.value.storageBackendId || '',
    extract_config: {
      enabled: !!form.value.nodeExtractConfig.enabled,
      text: form.value.nodeExtractConfig.text || '',
      tags: form.value.nodeExtractConfig.tags || [],
      nodes: form.value.nodeExtractConfig.nodes || [],
      relations: form.value.nodeExtractConfig.relations || [],
      custom_instructions: form.value.nodeExtractConfig.customInstructions || '',
    },
    question_generation_config: {
      enabled: !!form.value.questionGenerationConfig.enabled,
      question_count: form.value.questionGenerationConfig.questionCount || 3,
      custom_instructions: form.value.questionGenerationConfig.customInstructions || '',
    },
    wiki_config: {
      synthesis_model_id: form.value.wikiConfig.synthesisModelId || '',
      max_pages_per_ingest: form.value.wikiConfig.maxPagesPerIngest || 0,
      extraction_granularity: form.value.wikiConfig.extractionGranularity || 'standard',
      content_instructions: form.value.wikiConfig.contentInstructions || '',
      extraction_instructions: form.value.wikiConfig.extractionInstructions || '',
    },
    indexing_strategy: {
      vector_enabled: !!form.value.indexingStrategy.vectorEnabled,
      keyword_enabled: !!form.value.indexingStrategy.keywordEnabled,
      wiki_enabled: !!form.value.indexingStrategy.wikiEnabled,
      graph_enabled: !!form.value.indexingStrategy.graphEnabled,
    },
  }
}

async function load() {
  loading.value = true
  try {
    const [configResponse, models, knowledgeBaseResponse] = await Promise.all([
      getKnowledgeBaseDefaults(),
      listModels(),
      listKnowledgeBases({ creator: 'all' }),
    ])
    const config = (configResponse as any)?.data
    form.value = mapConfig(config)
    allModels.value = Array.isArray(models) ? models : []
    const knowledgeBases = Array.isArray((knowledgeBaseResponse as any)?.data)
      ? (knowledgeBaseResponse as any).data
      : []
    knowledgeBaseCount.value = knowledgeBases.filter((kb: any) => !kb?.is_temporary).length
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库统一配置加载失败')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!canSave.value) return
  if (!form.value.modelConfig.llmModelId) {
    activeTab.value = 'models'
    MessagePlugin.warning('请选择总结/问答模型')
    return
  }
  if ((form.value.indexingStrategy.vectorEnabled || form.value.indexingStrategy.keywordEnabled) &&
    !form.value.modelConfig.embeddingModelId) {
    activeTab.value = 'models'
    MessagePlugin.warning('启用检索索引时必须选择 Embedding 模型')
    return
  }
  saving.value = true
  try {
    const response: any = await updateKnowledgeBaseDefaults(buildPayload())
    chatResources.invalidate('knowledgeBases')
    MessagePlugin.success('工作区默认配置已保存；已有知识库未修改')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库统一配置保存失败')
  } finally {
    saving.value = false
  }
}

function handleModelConfigUpdate(config: any) {
  form.value.modelConfig = { ...config }
}

function handleParserEngineRulesUpdate(rules: any[]) {
  form.value.chunkingConfig.parserEngineRules = rules || []
}

function handleChunkingConfigUpdate(config: any) {
  form.value.chunkingConfig = { ...form.value.chunkingConfig, ...config }
}

function handleQuestionGenerationUpdate(config: any) {
  form.value.questionGenerationConfig = { ...config }
}

function handleTableMetadataInstructionsUpdate(value: string) {
  form.value.chunkingConfig.tableMetadataInstructions = value
}

function handleNodeExtractUpdate(config: any) {
  form.value.nodeExtractConfig = { ...config }
}

onMounted(load)
</script>

<style scoped lang="less">
.admin-kb-defaults {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.admin-kb-defaults__hero {
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: flex-end;
}

.admin-kb-defaults__hero h2 {
  margin: 10px 0 6px;
  font-size: 24px;
}

.admin-kb-defaults__hero p {
  margin: 0;
  color: var(--td-text-color-secondary);
}

.admin-kb-defaults__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.admin-kb-defaults__summary article {
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.admin-kb-defaults__summary span,
.admin-kb-defaults__summary em {
  display: block;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-style: normal;
}

.admin-kb-defaults__summary strong {
  display: block;
  margin: 6px 0;
  font-size: 22px;
}

.admin-kb-defaults__layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 20px;
  min-height: 560px;
}

.admin-kb-defaults__tabs {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.admin-kb-defaults__tabs button {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  width: 100%;
  padding: 12px;
  border: 1px solid transparent;
  background: transparent;
  color: var(--td-text-color-secondary);
  text-align: left;
  cursor: pointer;
}

.admin-kb-defaults__tabs button.active {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color);
}

.admin-kb-defaults__tabs span {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.admin-kb-defaults__tabs small {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.admin-kb-defaults__content {
  min-width: 0;
  padding: 20px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.admin-kb-section {
  min-height: 520px;
}

.admin-kb-section__heading {
  margin-bottom: 20px;
}

.admin-kb-section__heading h3 {
  margin: 0 0 6px;
}

.admin-kb-section__heading p {
  margin: 0;
  color: var(--td-text-color-secondary);
}

.admin-kb-setting-block {
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.admin-kb-switch-row {
  display: flex;
  justify-content: space-between;
  gap: 20px;
}

.admin-kb-switch-row span {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.admin-kb-switch-row small {
  color: var(--td-text-color-secondary);
}

.admin-kb-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
  margin-top: 18px;
}

.admin-kb-settings-divider {
  height: 1px;
  margin: 20px 0;
  background: var(--td-component-stroke);
}

@media (max-width: 900px) {
  .admin-kb-defaults__hero,
  .admin-kb-defaults__layout {
    grid-template-columns: 1fr;
    display: grid;
  }

  .admin-kb-defaults__hero {
    align-items: start;
  }

  .admin-kb-defaults__summary,
  .admin-kb-form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
