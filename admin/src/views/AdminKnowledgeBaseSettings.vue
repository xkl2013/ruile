<template>
  <section class="admin-kb-settings">
    <div v-if="loading && !form" class="admin-kb-state">
      <t-loading size="small" />
      <span>正在加载知识库配置</span>
    </div>

    <div v-else-if="loadError" class="admin-kb-state admin-kb-state--error">
      <t-icon name="error-circle" />
      <strong>{{ loadError }}</strong>
      <t-button size="small" variant="outline" @click="loadKnowledgeBase(true)">重试</t-button>
    </div>

    <template v-else-if="form && kb">
      <header class="admin-kb-settings__hero">
        <div class="admin-kb-title">
          <t-button variant="text" size="small" @click="backToList">
            <template #icon><t-icon name="chevron-left" /></template>
            返回知识库
          </t-button>
          <div class="admin-kb-title__main">
            <KnowledgeBaseIcon
              :icon="form.icon"
              :icon-url="form.iconUrl"
              :type="form.type"
              size="large"
            />
            <span>
              <h2>{{ form.name || '未命名知识库' }}</h2>
              <p>
                <t-tag :theme="form.type === 'faq' ? 'primary' : 'success'" variant="light" size="small">
                  {{ form.type === 'faq' ? 'FAQ 库' : '文档库' }}
                </t-tag>
                <code>{{ kbId }}</code>
              </p>
            </span>
          </div>
        </div>
        <div class="admin-kb-title-actions">
          <t-button variant="outline" @click="openWorkspace">
            <template #icon><t-icon name="jump" /></template>
            打开工作台
          </t-button>
          <t-button
            theme="primary"
            :loading="saving"
            :disabled="!canSaveSettings"
            @click="handleSave"
          >
            <template #icon><t-icon name="save" /></template>
            保存配置
          </t-button>
        </div>
      </header>

      <t-alert
        v-if="!canSaveSettings"
        theme="warning"
        variant="light"
        message="当前账号只能查看该知识库后台配置；保存模型、解析、存储、目录和数据源需要空间 Admin 或系统管理员权限。"
      />

      <section class="admin-kb-settings__summary">
        <article>
          <span>配置状态</span>
          <strong>{{ isInitialized ? '已配置' : '待配置' }}</strong>
          <em>{{ indexingSummary }}</em>
        </article>
        <article>
          <span>内容量</span>
          <strong>{{ kb.type === 'faq' ? kb.chunk_count || 0 : kb.knowledge_count || 0 }}</strong>
          <em>{{ kb.type === 'faq' ? 'FAQ 条目' : '知识文件' }}</em>
        </article>
        <article>
          <span>存储后端</span>
          <strong>{{ form.storageBackendId ? '已绑定' : '默认' }}</strong>
          <em>{{ form.storageProvider || 'local' }}</em>
        </article>
        <article>
          <span>目录</span>
          <strong>{{ directoryRows.length }}</strong>
          <em>后台维护</em>
        </article>
      </section>

      <section class="admin-kb-settings__layout">
        <aside class="admin-kb-tabs" aria-label="知识库后台设置">
          <button
            v-for="tab in visibleTabs"
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

        <div class="admin-kb-settings__content">
          <section v-show="activeTab === 'basic'" class="admin-kb-section">
            <div class="admin-kb-section__heading">
              <h3>基础信息</h3>
              <p>维护知识库名称、描述、图标和索引启用范围。</p>
            </div>

            <t-form :data="form" label-align="top" @submit.prevent>
              <div class="admin-kb-basic-grid">
                <t-form-item label="知识库图标">
                  <t-popup
                    v-model:visible="iconPickerVisible"
                    trigger="click"
                    placement="bottom-left"
                    :overlay-style="{ padding: 0 }"
                    :overlay-inner-style="{ padding: 0 }"
                  >
                    <button type="button" class="admin-kb-icon-trigger">
                      <KnowledgeBaseIcon
                        :icon="form.icon"
                        :icon-url="form.iconUrl"
                        :type="form.type"
                        size="large"
                      />
                      <t-icon name="chevron-down" />
                    </button>
                    <template #content>
                      <div class="admin-kb-icon-picker" @click.stop>
                        <button
                          type="button"
                          class="admin-kb-icon-option"
                          :class="{ active: isUploadedKnowledgeBaseIcon }"
                          title="上传图标"
                          @click="triggerIconUpload"
                        >
                          <KnowledgeBaseIcon
                            v-if="isUploadedKnowledgeBaseIcon"
                            :icon="form.icon"
                            :icon-url="form.iconUrl"
                            :type="form.type"
                            size="large"
                          />
                          <t-icon v-else name="upload" />
                        </button>
                        <button
                          v-for="icon in kbIconOptions"
                          :key="icon"
                          type="button"
                          class="admin-kb-icon-option"
                          :class="{ active: form.icon === icon }"
                          :title="icon"
                          @click="selectKnowledgeBaseIcon(icon)"
                        >
                          <t-icon :name="icon" />
                        </button>
                        <input
                          ref="iconFileInputRef"
                          type="file"
                          class="admin-kb-icon-input"
                          accept="image/png,image/jpeg,image/webp,image/gif"
                          @change="handleIconFileChange"
                        />
                      </div>
                    </template>
                  </t-popup>
                </t-form-item>

                <t-form-item label="知识库类型">
                  <t-radio-group v-model="form.type" disabled>
                    <t-radio-button value="document">文档库</t-radio-button>
                    <t-radio-button value="faq">FAQ 库</t-radio-button>
                  </t-radio-group>
                </t-form-item>
              </div>

              <t-form-item label="知识库名称" name="name">
                <t-input v-model="form.name" :maxlength="50" clearable placeholder="请输入知识库名称" />
              </t-form-item>

              <t-form-item label="知识库描述" name="description">
                <t-textarea
                  v-model="form.description"
                  :maxlength="200"
                  :autosize="{ minRows: 3, maxRows: 6 }"
                  placeholder="说明该知识库服务的业务对象、资料边界和使用场景"
                />
              </t-form-item>

              <template v-if="form.type !== 'faq'">
                <div class="admin-kb-setting-block">
                  <div class="admin-kb-setting-block__title">
                    <span>
                      <strong>索引能力</strong>
                      <small>决定入库后生成哪些可检索资产；已有文件时变更需要重建索引。</small>
                    </span>
                    <t-tag v-if="isIndexingLocked" theme="warning" variant="light">已有文件</t-tag>
                  </div>
                  <div class="admin-kb-indexing-grid" :class="{ locked: isIndexingLocked }">
                    <label>
                      <t-switch v-model="form.indexingStrategy.vectorEnabled" :disabled="isIndexingLocked" />
                      <span>
                        <strong>向量检索</strong>
                        <small>语义召回，适合问答和摘要引用。</small>
                      </span>
                    </label>
                    <label>
                      <t-switch v-model="form.indexingStrategy.keywordEnabled" :disabled="isIndexingLocked" />
                      <span>
                        <strong>关键词检索</strong>
                        <small>精确词、术语和编号匹配。</small>
                      </span>
                    </label>
                    <label>
                      <t-switch v-model="form.indexingStrategy.wikiEnabled" :disabled="isIndexingLocked" />
                      <span>
                        <strong>Wiki 生成</strong>
                        <small>从资料中生成结构化知识页。</small>
                      </span>
                    </label>
                    <label>
                      <t-switch v-model="form.indexingStrategy.graphEnabled" :disabled="isIndexingLocked" />
                      <span>
                        <strong>知识图谱</strong>
                        <small>提取实体和关系。</small>
                      </span>
                    </label>
                  </div>
                </div>
              </template>

              <template v-else>
                <div class="admin-kb-setting-block">
                  <div class="admin-kb-setting-block__title">
                    <span>
                      <strong>FAQ 索引</strong>
                      <small>控制 FAQ 问题和答案的检索方式。</small>
                    </span>
                  </div>
                  <div class="admin-kb-form-grid">
                    <t-form-item label="索引内容">
                      <t-radio-group v-model="form.faqConfig.indexMode">
                        <t-radio-button value="question_only">仅问题</t-radio-button>
                        <t-radio-button value="question_answer">问题和答案</t-radio-button>
                      </t-radio-group>
                    </t-form-item>
                    <t-form-item label="问题索引方式">
                      <t-radio-group v-model="form.faqConfig.questionIndexMode">
                        <t-radio-button value="combined">合并索引</t-radio-button>
                        <t-radio-button value="separate">分开索引</t-radio-button>
                      </t-radio-group>
                    </t-form-item>
                  </div>
                </div>
              </template>
            </t-form>
          </section>

          <section v-show="activeTab === 'models'" class="admin-kb-section">
            <KBModelConfig
              :config="form.modelConfig"
              :has-files="hasFiles"
              :wiki-enabled="form.indexingStrategy?.wikiEnabled"
              :rag-enabled="form.indexingStrategy?.vectorEnabled || form.indexingStrategy?.keywordEnabled"
              :all-models="allModels"
              @update:config="handleModelConfigUpdate"
            />

            <div
              v-if="form.type !== 'faq' && form.indexingStrategy.wikiEnabled"
              class="admin-kb-setting-block admin-kb-setting-block--after"
            >
              <div class="admin-kb-setting-block__title">
                <span>
                  <strong>Wiki 提取参数</strong>
                  <small>用于控制 Wiki 生成粒度和页面生成指令。</small>
                </span>
              </div>
              <div class="admin-kb-form-grid">
                <t-form-item label="提取粒度">
                  <t-radio-group v-model="form.wikiConfig.extractionGranularity">
                    <t-radio-button value="focused">聚焦</t-radio-button>
                    <t-radio-button value="standard">标准</t-radio-button>
                    <t-radio-button value="exhaustive">详尽</t-radio-button>
                  </t-radio-group>
                </t-form-item>
                <t-form-item label="单次最大页面数">
                  <t-input-number v-model="form.wikiConfig.maxPagesPerIngest" :min="0" :max="500" />
                </t-form-item>
              </div>
              <t-form label-align="top" @submit.prevent>
                <t-form-item label="页面内容指令">
                  <t-textarea
                    v-model="form.wikiConfig.contentInstructions"
                    :maxlength="4000"
                    :autosize="{ minRows: 3, maxRows: 8 }"
                    placeholder="描述生成 Wiki 页面时需要遵循的内容边界"
                  />
                </t-form-item>
                <t-form-item label="提取指令">
                  <t-textarea
                    v-model="form.wikiConfig.extractionInstructions"
                    :maxlength="4000"
                    :autosize="{ minRows: 3, maxRows: 8 }"
                    placeholder="描述从原始资料抽取知识点的要求"
                  />
                </t-form-item>
              </t-form>
            </div>
          </section>

          <section v-show="activeTab === 'processing'" class="admin-kb-section">
            <template v-if="form.type === 'faq'">
              <div class="admin-kb-section__heading">
                <h3>FAQ 处理配置</h3>
                <p>FAQ 库只维护问答结构和标签，文档解析、分块、数据源不适用。</p>
              </div>
              <t-alert theme="info" variant="light" message="如需管理 FAQ 条目，请回到主工作台的知识库使用页。后台仅保留配置和治理项。" />
            </template>
            <template v-else>
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
                :rag-enabled="form.indexingStrategy?.vectorEnabled || form.indexingStrategy?.keywordEnabled"
                :all-models="allModels"
                :table-metadata-instructions="form.chunkingConfig.tableMetadataInstructions"
                @update:question-generation="handleQuestionGenerationUpdate"
                @update:table-metadata-instructions="handleTableMetadataInstructionsUpdate"
              />
            </template>
          </section>

          <section v-show="activeTab === 'multimodal'" class="admin-kb-section">
            <div class="admin-kb-section__heading">
              <h3>多模态和音频</h3>
              <p>配置图片、OCR 和音频转写能力，影响后续入库处理。</p>
            </div>

            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用图片理解</strong>
                  <small>对文档图片生成文字描述，供检索和问答使用。</small>
                </span>
                <t-switch v-model="form.multimodalConfig.enabled" @change="handleMultimodalToggle" />
              </div>
              <div v-if="form.multimodalConfig.enabled" class="admin-kb-form-grid">
                <t-form-item label="VLM 模型">
                  <ModelSelector
                    model-type="VLLM"
                    :selected-model-id="form.multimodalConfig.vllmModelId"
                    :all-models="allModels"
                    placeholder="选择图片理解模型"
                    @update:selected-model-id="(value: string) => { form.multimodalConfig.vllmModelId = value }"
                    @add-model="openModelSettings('vllm')"
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
              <t-form v-if="form.multimodalConfig.enabled" label-align="top" @submit.prevent>
                <t-form-item label="图片描述指令">
                  <t-textarea
                    v-model="form.multimodalConfig.customInstructions"
                    :maxlength="4000"
                    :autosize="{ minRows: 3, maxRows: 8 }"
                    placeholder="定义图片描述的细节、格式和排除项"
                  />
                </t-form-item>
              </t-form>
            </div>

            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用 OCR 兜底</strong>
                  <small>图片理解不可用或需提取图片文字时使用。</small>
                </span>
                <t-switch v-model="form.ocrConfig.enabled" @change="handleOCRToggle" />
              </div>
              <t-form-item v-if="form.ocrConfig.enabled" label="OCR 模型">
                <ModelSelector
                  model-type="OCR"
                  :selected-model-id="form.ocrConfig.modelId"
                  :all-models="allModels"
                  placeholder="选择 OCR 模型"
                  @update:selected-model-id="(value: string) => { form.ocrConfig.modelId = value }"
                  @add-model="openModelSettings('ocr')"
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
                    @add-model="openModelSettings('asr')"
                  />
                </t-form-item>
                <t-form-item label="语言">
                  <t-input v-model="form.asrConfig.language" clearable placeholder="可选，例如 zh、en" />
                </t-form-item>
              </div>
            </div>
          </section>

          <section v-show="activeTab === 'graph'" class="admin-kb-section">
            <div class="admin-kb-setting-block">
              <div class="admin-kb-switch-row">
                <span>
                  <strong>启用图谱索引</strong>
                  <small>开启后，后续入库可生成实体关系索引；已有文件变更后建议重建。</small>
                </span>
                <t-switch v-model="form.indexingStrategy.graphEnabled" :disabled="isIndexingLocked" />
              </div>
            </div>
            <GraphSettings
              :graph-extract="form.nodeExtractConfig"
              :model-id="form.modelConfig.llmModelId"
              :all-models="allModels"
              @update:graphExtract="handleNodeExtractUpdate"
            />
          </section>

          <section v-show="activeTab === 'storage'" class="admin-kb-section">
            <KBStorageSettings
              :storage-backend-id="form.storageBackendId"
              :storage-provider="form.storageProvider"
              :has-files="hasFiles"
              @update:storage-backend-id="handleStorageBackendUpdate"
              @update:storage-provider="handleStorageProviderUpdate"
            />
            <div class="admin-kb-settings-divider" />
            <KBVectorStoreSettings
              mode="edit"
              :bound-source="form.vectorStoreInfo.source"
              :bound-name="form.vectorStoreInfo.name"
              :bound-engine-type="form.vectorStoreInfo.engineType"
              :bound-status="form.vectorStoreInfo.status"
            />
          </section>

          <section v-show="activeTab === 'dataSources'" class="admin-kb-section">
            <DataSourceSettings :kb-id="kbId" />
          </section>

          <section v-show="activeTab === 'sharing'" class="admin-kb-section">
            <KBShareSettings :kb-id="kbId" :can-share="canShareKB" />
          </section>

          <section v-show="activeTab === 'directories'" class="admin-kb-section">
            <div class="admin-kb-section__heading admin-kb-section__heading--row">
              <span>
                <h3>目录配置</h3>
                <p>维护文档目录的显示名称和描述；不迁移文档浏览和文件预览。</p>
              </span>
              <t-button theme="primary" :disabled="!canSaveSettings" @click="openCreateDirectory">
                <template #icon><t-icon name="folder-add" /></template>
                新建目录
              </t-button>
            </div>

            <div class="admin-kb-setting-block">
              <t-form label-align="top" @submit.prevent>
                <t-form-item label="根目录描述">
                  <t-textarea
                    v-model="rootDirectoryDescription"
                    :maxlength="300"
                    :autosize="{ minRows: 3, maxRows: 5 }"
                    placeholder="说明根目录的资料范围"
                  />
                </t-form-item>
              </t-form>
              <div class="admin-kb-directory-actions">
                <t-button
                  variant="outline"
                  :loading="directorySaving"
                  :disabled="!canSaveSettings"
                  @click="saveRootDirectory"
                >
                  保存根目录描述
                </t-button>
              </div>
            </div>

            <div class="admin-kb-directory-table">
              <t-table
                row-key="path"
                :data="directoryRows"
                :columns="directoryColumns"
                table-layout="fixed"
                :hover="true"
              >
                <template #path="{ row }">
                  <code class="admin-kb-directory-path">{{ row.path }}</code>
                </template>
                <template #description="{ row }">
                  <span class="admin-kb-directory-desc" :title="row.description || ''">
                    {{ row.description || '暂无描述' }}
                  </span>
                </template>
                <template #operation="{ row }">
                  <div class="admin-kb-directory-row-actions">
                    <t-button
                      size="small"
                      variant="text"
                      :disabled="!canSaveSettings"
                      @click="openEditDirectory(row)"
                    >
                      编辑
                    </t-button>
                    <t-button
                      size="small"
                      variant="text"
                      theme="danger"
                      :disabled="!canSaveSettings"
                      @click="deleteDirectory(row)"
                    >
                      删除
                    </t-button>
                  </div>
                </template>
                <template #empty>
                  <div class="admin-kb-empty">
                    <t-icon name="folder-open" />
                    <span>暂无目录配置</span>
                  </div>
                </template>
              </t-table>
            </div>
          </section>

          <section v-show="activeTab === 'tags'" class="admin-kb-section">
            <div class="admin-kb-section__heading">
              <h3>标签管理</h3>
              <p>标签用于知识分类和检索过滤，后台只维护标签字典，不迁入知识内容编辑。</p>
            </div>
            <div class="admin-kb-setting-block admin-kb-tag-entry">
              <span>
                <strong>管理当前知识库标签</strong>
                <small>支持新增、改名和删除；删除会影响已打标知识的分类。</small>
              </span>
              <t-button theme="primary" :disabled="!canConfigureByOwnerOrAdmin" @click="tagDrawerVisible = true">
                <template #icon><t-icon name="discount" /></template>
                打开标签管理
              </t-button>
            </div>
          </section>
        </div>
      </section>

      <t-dialog
        v-model:visible="directoryDialogVisible"
        :header="directoryDialogMode === 'create' ? '新建目录' : '编辑目录'"
        width="460px"
        :confirm-btn="{ content: '保存', theme: 'primary', loading: directorySaving, disabled: !canSaveSettings }"
        :cancel-btn="{ content: '取消', disabled: directorySaving }"
        destroy-on-close
        @confirm="saveDirectoryDialog"
        @cancel="closeDirectoryDialog"
        @close="closeDirectoryDialog"
      >
        <t-form :data="directoryForm" label-align="top" @submit.prevent>
          <t-form-item v-if="directoryDialogMode === 'create'" label="父级目录">
            <t-select v-model="directoryForm.parentPath">
              <t-option value="" label="根目录" />
              <t-option
                v-for="dir in directoryRows"
                :key="dir.path"
                :value="dir.path"
                :label="dir.path"
              />
            </t-select>
          </t-form-item>
          <t-form-item label="目录名称">
            <t-input
              v-model="directoryForm.name"
              :maxlength="80"
              :disabled="directoryDialogMode === 'edit'"
              clearable
              placeholder="请输入目录名称"
              @enter="saveDirectoryDialog"
            />
          </t-form-item>
          <t-form-item label="目录描述">
            <t-textarea
              v-model="directoryForm.description"
              :maxlength="300"
              :autosize="{ minRows: 3, maxRows: 5 }"
              placeholder="说明该目录包含的资料范围"
            />
          </t-form-item>
        </t-form>
      </t-dialog>

      <KbTagManageDrawer
        v-model:visible="tagDrawerVisible"
        :kb-id="kbId"
        :is-faq="form.type === 'faq'"
      />
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  getKnowledgeBaseById,
  listKnowledgeFiles,
  rebuildKBIndex,
  updateKnowledgeBase,
  updateKnowledgeBaseDirectoryConfig,
  uploadKnowledgeBaseIcon,
  type KnowledgeBaseDirectoryConfigPayload,
  type KnowledgeBaseDirectoryNodePayload,
  type KnowledgeBaseDirectoryOrderPayload,
} from '@/api/knowledge-base'
import { updateKBConfig, type KBModelConfigRequest } from '@/api/initialization'
import { listModels } from '@/api/model'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import ModelSelector from '@/components/ModelSelector.vue'
import {
  cloneDefaultParserEngineRules,
  createDefaultKnowledgeBaseFormData,
  getDefaultKnowledgeBaseIcon,
  KNOWLEDGE_BASE_ICON_OPTIONS,
} from '@/config/knowledgeBaseDefaults'
import KBAdvancedSettings from '@/views/knowledge/settings/KBAdvancedSettings.vue'
import KBChunkingSettings from '@/views/knowledge/settings/KBChunkingSettings.vue'
import KBModelConfig from '@/views/knowledge/settings/KBModelConfig.vue'
import KBParserSettings from '@/views/knowledge/settings/KBParserSettings.vue'
import KBShareSettings from '@/views/knowledge/settings/KBShareSettings.vue'
import KBStorageSettings from '@/views/knowledge/settings/KBStorageSettings.vue'
import KBVectorStoreSettings from '@/views/knowledge/settings/KBVectorStoreSettings.vue'
import DataSourceSettings from '@/views/knowledge/settings/DataSourceSettings.vue'
import GraphSettings from '@/views/knowledge/settings/GraphSettings.vue'
import KbTagManageDrawer from '@/views/knowledge/components/KbTagManageDrawer.vue'
import { useAuthStore } from '@/stores/auth'
import { useChatResourcesStore } from '@/stores/chatResources'
import { openMainAppPath } from '@admin/utils/navigation'

type TabKey =
  | 'basic'
  | 'models'
  | 'processing'
  | 'multimodal'
  | 'graph'
  | 'storage'
  | 'dataSources'
  | 'sharing'
  | 'directories'
  | 'tags'

type DirectoryRow = {
  path: string
  name: string
  description: string
  parentPath: string
  createdAt: string
  updatedAt: string
}

type DirectoryOrder = {
  parentPath: string
  paths: string[]
}

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const chatResources = useChatResourcesStore()

const kbId = computed(() => String(route.params.kbId || ''))
const kb = ref<any>(null)
const form = ref<any>(null)
const allModels = ref<any[]>([])
const hasFiles = ref(false)
const loading = ref(false)
const saving = ref(false)
const loadError = ref('')
const iconPickerVisible = ref(false)
const iconFileInputRef = ref<HTMLInputElement | null>(null)
const tagDrawerVisible = ref(false)
const rootDirectoryDescription = ref('')
const directoryRows = ref<DirectoryRow[]>([])
const directoryOrders = ref<DirectoryOrder[]>([])
const directorySaving = ref(false)
const directoryDialogVisible = ref(false)
const directoryDialogMode = ref<'create' | 'edit'>('create')
const editingDirectoryPath = ref('')
const directoryForm = reactive({
  parentPath: '',
  name: '',
  description: '',
})
const activeTab = ref<TabKey>(normalizeTab(route.query.tab))
const initialIndexingStrategy = ref<any>(null)
const initialStorageBackendId = ref('')
const initialStorageProvider = ref('')

const kbIconOptions = KNOWLEDGE_BASE_ICON_OPTIONS
const KB_ICON_UPLOAD_MAX_BYTES = 5 * 1024 * 1024
const KB_ICON_IMAGE_SIZE = 96

const directoryColumns = [
  { colKey: 'path', title: '路径', minWidth: 220 },
  { colKey: 'name', title: '显示名称', width: 160 },
  { colKey: 'description', title: '描述', minWidth: 240 },
  { colKey: 'operation', title: '操作', width: 140, cell: 'operation' },
]

const canSaveSettings = computed(() => authStore.hasRole('admin') || authStore.isSystemAdmin)
const canConfigureByOwnerOrAdmin = computed(() => {
  if (authStore.isSystemAdmin || authStore.hasRole('admin')) return true
  const userId = authStore.user?.id || ''
  return Boolean(kb.value?.creator_id && userId && kb.value.creator_id === userId)
})
const canShareKB = computed(() => canConfigureByOwnerOrAdmin.value)
const isUploadedKnowledgeBaseIcon = computed(() => (form.value?.icon || '').trim().startsWith('image:'))

const isInitialized = computed(() => {
  if (!form.value) return false
  if (!form.value.modelConfig.llmModelId) return false
  const needsEmbedding =
    form.value.type !== 'faq' &&
    (form.value.indexingStrategy?.vectorEnabled || form.value.indexingStrategy?.keywordEnabled)
  return !needsEmbedding || Boolean(form.value.modelConfig.embeddingModelId)
})

const isIndexingLocked = computed(() => hasFiles.value)

const indexingSummary = computed(() => {
  if (!form.value) return '-'
  if (form.value.type === 'faq') return 'FAQ 问答索引'
  const strategy = form.value.indexingStrategy
  const items: string[] = []
  if (strategy?.vectorEnabled) items.push('向量')
  if (strategy?.keywordEnabled) items.push('关键词')
  if (strategy?.wikiEnabled) items.push('Wiki')
  if (strategy?.graphEnabled) items.push('图谱')
  return items.length ? items.join('、') : '未启用索引'
})

const visibleTabs = computed<Array<{ key: TabKey; label: string; desc: string; icon: string }>>(() => {
  const isFAQ = form.value?.type === 'faq'
  const tabs: Array<{ key: TabKey; label: string; desc: string; icon: string; hidden?: boolean }> = [
    { key: 'basic', label: '基础信息', desc: '名称、图标、索引', icon: 'info-circle' },
    { key: 'models', label: '模型绑定', desc: 'LLM、Embedding、Wiki', icon: 'control-platform' },
    { key: 'processing', label: '解析分块', desc: 'Parser、Chunking、高级项', icon: 'file-setting' },
    { key: 'multimodal', label: '多模态', desc: '图片、OCR、ASR', icon: 'image', hidden: isFAQ },
    { key: 'graph', label: '知识图谱', desc: '实体和关系提取', icon: 'chart-bubble', hidden: isFAQ },
    { key: 'storage', label: '存储向量', desc: '对象存储、向量库', icon: 'data-base', hidden: isFAQ },
    { key: 'dataSources', label: '数据源', desc: '同步、暂停、日志', icon: 'cloud-download', hidden: isFAQ },
    { key: 'sharing', label: '分享', desc: '组织共享权限', icon: 'share' },
    { key: 'directories', label: '目录', desc: '目录名称和描述', icon: 'folder-setting', hidden: isFAQ },
    { key: 'tags', label: '标签', desc: '标签字典', icon: 'discount' },
  ]
  return tabs.filter((tab) => !tab.hidden)
})

function normalizeTab(value: unknown): TabKey {
  const raw = Array.isArray(value) ? value[0] : value
  const key = String(raw || 'basic') as TabKey
  const valid: TabKey[] = [
    'basic',
    'models',
    'processing',
    'multimodal',
    'graph',
    'storage',
    'dataSources',
    'sharing',
    'directories',
    'tags',
  ]
  return valid.includes(key) ? key : 'basic'
}

function mapKBToForm(source: any) {
  const type = source?.type === 'faq' ? 'faq' : 'document'
  const defaults = createDefaultKnowledgeBaseFormData(type)
  const parserEngineRules = source?.chunking_config?.parser_engine_rules?.length
    ? source.chunking_config.parser_engine_rules
    : cloneDefaultParserEngineRules()

  return {
    ...defaults,
    type,
    icon: source?.icon || getDefaultKnowledgeBaseIcon(type),
    iconUrl: source?.icon_url || '',
    name: source?.name || '',
    description: source?.description || '',
    faqConfig: {
      indexMode: source?.faq_config?.index_mode || defaults.faqConfig.indexMode,
      questionIndexMode: source?.faq_config?.question_index_mode || defaults.faqConfig.questionIndexMode,
    },
    modelConfig: {
      llmModelId: source?.summary_model_id || '',
      embeddingModelId: source?.embedding_model_id || '',
      wikiSynthesisModelId: source?.wiki_config?.synthesis_model_id || '',
    },
    chunkingConfig: {
      ...defaults.chunkingConfig,
      chunkSize: source?.chunking_config?.chunk_size ?? defaults.chunkingConfig.chunkSize,
      chunkOverlap: source?.chunking_config?.chunk_overlap ?? defaults.chunkingConfig.chunkOverlap,
      separators: source?.chunking_config?.separators || defaults.chunkingConfig.separators,
      parserEngineRules,
      enableParentChild: source?.chunking_config?.enable_parent_child ?? defaults.chunkingConfig.enableParentChild,
      parentChunkSize: source?.chunking_config?.parent_chunk_size ?? defaults.chunkingConfig.parentChunkSize,
      childChunkSize: source?.chunking_config?.child_chunk_size ?? defaults.chunkingConfig.childChunkSize,
      strategy: source?.chunking_config?.strategy ?? '',
      tokenLimit: source?.chunking_config?.token_limit ?? 0,
      languages: source?.chunking_config?.languages || [],
      tableMetadataInstructions: source?.chunking_config?.table_metadata_instructions || '',
    },
    storageBackendId: source?.storage_backend_id || '',
    storageProvider: source?.storage_provider_config?.provider || source?.storage_config?.provider || 'local',
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
      nodes: (source?.extract_config?.nodes || []).map((node: any) => ({
        name: node.name,
        attributes: node.attributes || [],
      })),
      relations: source?.extract_config?.relations || [],
      customInstructions: source?.extract_config?.custom_instructions || '',
    },
    questionGenerationConfig: {
      enabled: source?.question_generation_config?.enabled ?? defaults.questionGenerationConfig.enabled,
      questionCount: source?.question_generation_config?.question_count ?? defaults.questionGenerationConfig.questionCount,
      customInstructions: source?.question_generation_config?.custom_instructions || '',
    },
    wikiConfig: {
      synthesisModelId: source?.wiki_config?.synthesis_model_id || '',
      maxPagesPerIngest: source?.wiki_config?.max_pages_per_ingest || 0,
      extractionGranularity:
        source?.wiki_config?.extraction_granularity === 'focused' ||
        source?.wiki_config?.extraction_granularity === 'exhaustive'
          ? source.wiki_config.extraction_granularity
          : 'standard',
      contentInstructions: source?.wiki_config?.content_instructions || '',
      extractionInstructions: source?.wiki_config?.extraction_instructions || '',
    },
    indexingStrategy: {
      vectorEnabled: source?.indexing_strategy?.vector_enabled ?? defaults.indexingStrategy.vectorEnabled,
      keywordEnabled: source?.indexing_strategy?.keyword_enabled ?? defaults.indexingStrategy.keywordEnabled,
      wikiEnabled: source?.indexing_strategy?.wiki_enabled ?? defaults.indexingStrategy.wikiEnabled,
      graphEnabled: source?.indexing_strategy?.graph_enabled ?? defaults.indexingStrategy.graphEnabled,
    },
    vectorStoreInfo: {
      source: source?.vector_store_source,
      name: source?.vector_store_name,
      engineType: source?.vector_store_engine_type,
      status: source?.vector_store_status,
    },
  }
}

async function loadModels() {
  try {
    allModels.value = await listModels()
  } catch {
    allModels.value = []
  }
}

async function loadKnowledgeBase(force = false) {
  if (!kbId.value) return
  loading.value = true
  loadError.value = ''
  try {
    const [kbRes, filesRes] = await Promise.all([
      getKnowledgeBaseById(kbId.value),
      listKnowledgeFiles(kbId.value, { page: 1, page_size: 1 }),
      loadModels(),
    ])
    const data = (kbRes as any)?.data
    if (!data) throw new Error('知识库不存在或无权访问')
    kb.value = data
    form.value = mapKBToForm(data)
    const fileTotal = Number((filesRes as any)?.total ?? (filesRes as any)?.data?.total ?? 0)
    hasFiles.value = fileTotal > 0
    initialIndexingStrategy.value = { ...form.value.indexingStrategy }
    initialStorageBackendId.value = form.value.storageBackendId || ''
    initialStorageProvider.value = form.value.storageProvider || ''
    applyDirectoryConfig(data.directory_config)
    if (force) MessagePlugin.success('知识库配置已刷新')
  } catch (error: any) {
    loadError.value = error?.message || '知识库配置加载失败'
  } finally {
    loading.value = false
  }
}

function validateForm(): boolean {
  if (!form.value) return false
  if (!canSaveSettings.value) {
    MessagePlugin.warning('当前账号无权保存知识库后台配置')
    return false
  }
  if (!form.value.name?.trim()) {
    activeTab.value = 'basic'
    MessagePlugin.warning('请输入知识库名称')
    return false
  }
  if (form.value.type !== 'faq') {
    const s = form.value.indexingStrategy
    if (!s.vectorEnabled && !s.keywordEnabled && !s.wikiEnabled && !s.graphEnabled) {
      activeTab.value = 'basic'
      MessagePlugin.warning('至少启用一种索引能力')
      return false
    }
    if ((s.vectorEnabled || s.keywordEnabled) && !form.value.modelConfig.embeddingModelId) {
      activeTab.value = 'models'
      MessagePlugin.warning('启用检索索引时必须选择 Embedding 模型')
      return false
    }
  }
  if (!form.value.modelConfig.llmModelId) {
    activeTab.value = 'models'
    MessagePlugin.warning('请选择总结/问答模型')
    return false
  }
  if (form.value.multimodalConfig.enabled && !form.value.multimodalConfig.vllmModelId) {
    activeTab.value = 'multimodal'
    MessagePlugin.warning('启用图片理解时必须选择 VLM 模型')
    return false
  }
  if (form.value.ocrConfig.enabled && !form.value.ocrConfig.modelId) {
    activeTab.value = 'multimodal'
    MessagePlugin.warning('启用 OCR 时必须选择 OCR 模型')
    return false
  }
  if (form.value.asrConfig.enabled && !form.value.asrConfig.modelId) {
    activeTab.value = 'multimodal'
    MessagePlugin.warning('启用音频转写时必须选择 ASR 模型')
    return false
  }
  return true
}

function buildKnowledgeBaseUpdateConfig() {
  const cfg: any = {}
  if (form.value.type === 'faq') {
    cfg.faq_config = {
      index_mode: form.value.faqConfig.indexMode || 'question_only',
      question_index_mode: form.value.faqConfig.questionIndexMode || 'separate',
    }
  } else {
    cfg.indexing_strategy = {
      vector_enabled: !!form.value.indexingStrategy.vectorEnabled,
      keyword_enabled: !!form.value.indexingStrategy.keywordEnabled,
      wiki_enabled: !!form.value.indexingStrategy.wikiEnabled,
      graph_enabled: !!form.value.indexingStrategy.graphEnabled,
    }
    cfg.wiki_config = {
      synthesis_model_id: form.value.modelConfig.wikiSynthesisModelId || '',
      max_pages_per_ingest: form.value.wikiConfig.maxPagesPerIngest || 0,
      extraction_granularity: form.value.wikiConfig.extractionGranularity || 'standard',
      content_instructions: form.value.wikiConfig.contentInstructions || '',
      extraction_instructions: form.value.wikiConfig.extractionInstructions || '',
    }
  }
  return cfg
}

function buildKBConfigRequest(): KBModelConfigRequest {
  const chunk = form.value.chunkingConfig
  return {
    llmModelId: form.value.modelConfig.llmModelId,
    embeddingModelId: form.value.modelConfig.embeddingModelId,
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
    documentSplitting: {
      chunkSize: chunk.chunkSize,
      chunkOverlap: chunk.chunkOverlap,
      separators: chunk.separators || [],
      parserEngineRules: chunk.parserEngineRules?.length ? chunk.parserEngineRules : undefined,
      enableParentChild: !!chunk.enableParentChild,
      parentChunkSize: chunk.parentChunkSize || 4096,
      childChunkSize: chunk.childChunkSize || 384,
      strategy: chunk.strategy ?? '',
      tokenLimit: chunk.tokenLimit ?? 0,
      languages: chunk.languages ?? [],
      tableMetadataInstructions: chunk.tableMetadataInstructions || '',
    },
    multimodal: {
      enabled: !!form.value.multimodalConfig.enabled,
    },
    storageBackendId: form.value.storageBackendId || '',
    storageProvider: form.value.storageProvider || 'local',
    nodeExtract: {
      enabled: !!form.value.nodeExtractConfig.enabled,
      text: form.value.nodeExtractConfig.text || '',
      tags: form.value.nodeExtractConfig.tags || [],
      nodes: form.value.nodeExtractConfig.nodes || [],
      relations: form.value.nodeExtractConfig.relations || [],
      customInstructions: form.value.nodeExtractConfig.customInstructions || '',
    },
    questionGeneration: {
      enabled: !!form.value.questionGenerationConfig.enabled,
      questionCount: form.value.questionGenerationConfig.questionCount || 3,
      customInstructions: form.value.questionGenerationConfig.customInstructions || '',
    },
  }
}

async function handleSave() {
  if (!validateForm()) return
  const storageChanged =
    initialStorageBackendId.value !== (form.value.storageBackendId || '') ||
    initialStorageProvider.value !== (form.value.storageProvider || '')
  if (hasFiles.value && storageChanged) {
    const dialog = DialogPlugin.confirm({
      header: '确认变更存储后端',
      body: '该知识库已有文件，变更存储后端可能需要迁移或重新处理历史文件。确认继续保存？',
      confirmBtn: { content: '继续保存', theme: 'primary' },
      cancelBtn: { content: '取消' },
      onConfirm: () => {
        dialog.destroy()
        void doSave()
      },
      onCancel: () => dialog.destroy(),
    })
    return
  }
  await doSave()
}

async function doSave() {
  if (!form.value || saving.value) return
  saving.value = true
  try {
    await updateKnowledgeBase(kbId.value, {
      name: form.value.name.trim(),
      description: form.value.description || '',
      icon: form.value.icon || getDefaultKnowledgeBaseIcon(form.value.type),
      config: buildKnowledgeBaseUpdateConfig(),
    })
    await updateKBConfig(kbId.value, buildKBConfigRequest())
    MessagePlugin.success('知识库配置已保存')
    chatResources.invalidateKnowledgeBaseDetail(kbId.value)
    chatResources.invalidate('knowledgeBases')
    const strategyChanged = hasIndexingStrategyChanged()
    await loadKnowledgeBase()
    if (strategyChanged && hasFiles.value) {
      confirmRebuildIndex()
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库配置保存失败')
  } finally {
    saving.value = false
  }
}

function hasIndexingStrategyChanged(): boolean {
  if (!initialIndexingStrategy.value || !form.value?.indexingStrategy) return false
  const prev = initialIndexingStrategy.value
  const current = form.value.indexingStrategy
  return (
    prev.vectorEnabled !== current.vectorEnabled ||
    prev.keywordEnabled !== current.keywordEnabled ||
    prev.wikiEnabled !== current.wikiEnabled ||
    prev.graphEnabled !== current.graphEnabled
  )
}

function confirmRebuildIndex() {
  const dialog = DialogPlugin.confirm({
    header: '重建索引',
    body: '索引能力已变更，是否立即重建该知识库的历史文件索引？',
    confirmBtn: { content: '重建索引', theme: 'primary' },
    cancelBtn: { content: '稍后处理' },
    onConfirm: async () => {
      dialog.destroy()
      try {
        const result: any = await rebuildKBIndex(kbId.value)
        MessagePlugin.success(`已提交重建任务，共 ${result?.data?.document_count ?? 0} 个文件`)
      } catch (error: any) {
        MessagePlugin.error(error?.message || '重建索引失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

function handleModelConfigUpdate(config: any) {
  form.value.modelConfig = { ...config }
}

function handleParserEngineRulesUpdate(rules: any[]) {
  form.value.chunkingConfig.parserEngineRules = rules?.length ? rules : undefined
}

function handleChunkingConfigUpdate(config: any) {
  form.value.chunkingConfig = {
    ...form.value.chunkingConfig,
    ...config,
  }
}

function handleTableMetadataInstructionsUpdate(value: string) {
  form.value.chunkingConfig.tableMetadataInstructions = value
}

function handleQuestionGenerationUpdate(config: any) {
  form.value.questionGenerationConfig = { ...config }
}

function handleNodeExtractUpdate(config: any) {
  form.value.nodeExtractConfig = { ...config }
}

function handleStorageBackendUpdate(value: string) {
  form.value.storageBackendId = value || ''
}

function handleStorageProviderUpdate(value: string) {
  form.value.storageProvider = value || form.value.storageProvider || 'local'
}

function handleMultimodalToggle() {
  if (!form.value.multimodalConfig.enabled) {
    form.value.multimodalConfig.vllmModelId = ''
  }
}

function handleOCRToggle() {
  if (!form.value.ocrConfig.enabled) {
    form.value.ocrConfig.modelId = ''
  }
}

function openModelSettings(type: string) {
  void router.push({ name: 'adminModels', query: { type } })
}

function selectKnowledgeBaseIcon(icon: string) {
  form.value.icon = icon
  form.value.iconUrl = ''
  iconPickerVisible.value = false
}

function triggerIconUpload() {
  iconFileInputRef.value?.click()
}

async function handleIconFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file || !form.value) return
  if (!file.type.startsWith('image/')) {
    MessagePlugin.warning('请上传图片文件')
    return
  }
  if (file.size > KB_ICON_UPLOAD_MAX_BYTES) {
    MessagePlugin.warning('图标不能超过 5MB')
    return
  }
  try {
    const imageBlob = await resizeKnowledgeBaseIconImage(file)
    const response: any = await uploadKnowledgeBaseIcon(imageBlob, kbId.value)
    const result = response?.data || response
    if (!result?.icon) throw new Error('missing uploaded icon')
    form.value.icon = result.icon
    form.value.iconUrl = result.url || ''
    iconPickerVisible.value = false
  } catch (error: any) {
    MessagePlugin.error(error?.message || '图标处理失败')
  }
}

function resizeKnowledgeBaseIconImage(file: File): Promise<Blob> {
  return new Promise((resolve, reject) => {
    const image = new Image()
    const url = URL.createObjectURL(file)
    image.onload = () => {
      try {
        const sourceWidth = image.naturalWidth || image.width
        const sourceHeight = image.naturalHeight || image.height
        const sourceSize = Math.min(sourceWidth, sourceHeight)
        const sourceX = Math.max(0, (sourceWidth - sourceSize) / 2)
        const sourceY = Math.max(0, (sourceHeight - sourceSize) / 2)
        const canvas = document.createElement('canvas')
        canvas.width = KB_ICON_IMAGE_SIZE
        canvas.height = KB_ICON_IMAGE_SIZE
        const ctx = canvas.getContext('2d')
        if (!ctx) throw new Error('canvas context unavailable')
        ctx.imageSmoothingEnabled = true
        ctx.imageSmoothingQuality = 'high'
        ctx.clearRect(0, 0, KB_ICON_IMAGE_SIZE, KB_ICON_IMAGE_SIZE)
        ctx.drawImage(
          image,
          sourceX,
          sourceY,
          sourceSize,
          sourceSize,
          0,
          0,
          KB_ICON_IMAGE_SIZE,
          KB_ICON_IMAGE_SIZE,
        )
        canvas.toBlob((blob) => {
          if (blob) resolve(blob)
          else reject(new Error('failed to encode image'))
        }, 'image/png')
      } catch (error) {
        reject(error)
      } finally {
        URL.revokeObjectURL(url)
      }
    }
    image.onerror = () => {
      URL.revokeObjectURL(url)
      reject(new Error('failed to load image'))
    }
    image.src = url
  })
}

function normalizeDirectoryPath(value: unknown): string {
  return String(value || '')
    .trim()
    .replace(/\\/g, '/')
    .replace(/^\/+|\/+$/g, '')
    .replace(/\/+/g, '/')
}

function getDirectoryParentPath(path: string): string {
  const normalized = normalizeDirectoryPath(path)
  const index = normalized.lastIndexOf('/')
  return index > 0 ? normalized.slice(0, index) : ''
}

function getDirectoryDisplayName(path: string): string {
  const normalized = normalizeDirectoryPath(path)
  return normalized.split('/').filter(Boolean).pop() || '根目录'
}

function normalizeDirectoryRows(items: any[]): DirectoryRow[] {
  const rows: DirectoryRow[] = []
  const seen = new Set<string>()
  for (const item of items || []) {
    const path = normalizeDirectoryPath(item?.path)
    if (!path || seen.has(path)) continue
    seen.add(path)
    rows.push({
      path,
      name: String(item?.name || getDirectoryDisplayName(path)).trim() || getDirectoryDisplayName(path),
      description: String(item?.description || '').trim(),
      parentPath: normalizeDirectoryPath(item?.parent_path ?? item?.parentPath ?? getDirectoryParentPath(path)),
      createdAt: String(item?.created_at || item?.createdAt || new Date().toISOString()),
      updatedAt: String(item?.updated_at || item?.updatedAt || item?.created_at || item?.createdAt || new Date().toISOString()),
    })
  }
  return rows.sort((a, b) => a.path.localeCompare(b.path, 'zh-CN'))
}

function normalizeDirectoryOrders(items: any[]): DirectoryOrder[] {
  return (items || []).map((item) => ({
    parentPath: normalizeDirectoryPath(item?.parent_path ?? item?.parentPath),
    paths: Array.isArray(item?.paths) ? item.paths.map(normalizeDirectoryPath).filter(Boolean) : [],
  })).filter((item) => item.paths.length > 0)
}

function applyDirectoryConfig(config?: KnowledgeBaseDirectoryConfigPayload | null) {
  rootDirectoryDescription.value = String(config?.root_description || '').trim()
  directoryRows.value = normalizeDirectoryRows(Array.isArray(config?.directories) ? config?.directories || [] : [])
  directoryOrders.value = normalizeDirectoryOrders(Array.isArray(config?.directory_orders) ? config?.directory_orders || [] : [])
}

function buildDirectoryPayload(rows = directoryRows.value): KnowledgeBaseDirectoryConfigPayload {
  return {
    root_description: rootDirectoryDescription.value.trim(),
    directories: rows.map((row): KnowledgeBaseDirectoryNodePayload => ({
      path: normalizeDirectoryPath(row.path),
      name: String(row.name || getDirectoryDisplayName(row.path)).trim() || getDirectoryDisplayName(row.path),
      description: String(row.description || '').trim(),
      parent_path: normalizeDirectoryPath(row.parentPath || getDirectoryParentPath(row.path)),
      created_at: row.createdAt || new Date().toISOString(),
      updated_at: row.updatedAt || row.createdAt || new Date().toISOString(),
    })),
    directory_orders: directoryOrders.value
      .map((order): KnowledgeBaseDirectoryOrderPayload => ({
        parent_path: normalizeDirectoryPath(order.parentPath),
        paths: order.paths.map(normalizeDirectoryPath).filter(Boolean),
      }))
      .filter((order) => order.paths.length > 0),
  }
}

async function persistDirectoryRows(rows = directoryRows.value) {
  if (!canSaveSettings.value || directorySaving.value) return
  directorySaving.value = true
  try {
    const payload = buildDirectoryPayload(rows)
    const result: any = await updateKnowledgeBaseDirectoryConfig(kbId.value, {
      directory_config: payload,
    })
    directoryRows.value = normalizeDirectoryRows(result?.data?.directory_config?.directories || payload.directories)
    directoryOrders.value = normalizeDirectoryOrders(result?.data?.directory_config?.directory_orders || payload.directory_orders || [])
    rootDirectoryDescription.value = String(result?.data?.directory_config?.root_description ?? payload.root_description)
    kb.value = {
      ...kb.value,
      directory_config: result?.data?.directory_config || payload,
    }
    MessagePlugin.success('目录配置已保存')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '目录配置保存失败')
  } finally {
    directorySaving.value = false
  }
}

function saveRootDirectory() {
  void persistDirectoryRows()
}

function openCreateDirectory() {
  directoryDialogMode.value = 'create'
  editingDirectoryPath.value = ''
  directoryForm.parentPath = ''
  directoryForm.name = ''
  directoryForm.description = ''
  directoryDialogVisible.value = true
  void nextTick()
}

function openEditDirectory(row: DirectoryRow) {
  directoryDialogMode.value = 'edit'
  editingDirectoryPath.value = row.path
  directoryForm.parentPath = row.parentPath
  directoryForm.name = row.name
  directoryForm.description = row.description
  directoryDialogVisible.value = true
}

function closeDirectoryDialog() {
  directoryDialogVisible.value = false
}

function directorySiblingNameExists(parentPath: string, name: string, excludePath = ''): boolean {
  const normalizedParent = normalizeDirectoryPath(parentPath)
  const normalizedName = name.trim()
  return directoryRows.value.some((row) =>
    row.path !== excludePath &&
    normalizeDirectoryPath(row.parentPath) === normalizedParent &&
    row.name.trim() === normalizedName
  )
}

function saveDirectoryDialog() {
  if (!canSaveSettings.value) return
  const name = directoryForm.name.trim()
  if (!name) {
    MessagePlugin.warning('请输入目录名称')
    return
  }
  if (/[\\/]/.test(name)) {
    MessagePlugin.warning('目录名称不能包含 / 或 \\')
    return
  }

  const now = new Date().toISOString()
  if (directoryDialogMode.value === 'edit') {
    const targetPath = editingDirectoryPath.value
    const nextRows = directoryRows.value.map((row) => {
      if (row.path !== targetPath) return row
      return {
        ...row,
        name,
        description: directoryForm.description.trim(),
        updatedAt: now,
      }
    })
    directoryRows.value = nextRows
    closeDirectoryDialog()
    void persistDirectoryRows(nextRows)
    return
  }

  const parentPath = normalizeDirectoryPath(directoryForm.parentPath)
  const path = parentPath ? `${parentPath}/${name}` : name
  if (directoryRows.value.some((row) => row.path === path) || directorySiblingNameExists(parentPath, name)) {
    MessagePlugin.warning('同级目录已存在')
    return
  }
  const nextRows = [
    ...directoryRows.value,
    {
      path,
      name,
      description: directoryForm.description.trim(),
      parentPath,
      createdAt: now,
      updatedAt: now,
    },
  ]
  directoryRows.value = normalizeDirectoryRows(nextRows)
  closeDirectoryDialog()
  void persistDirectoryRows(directoryRows.value)
}

function deleteDirectory(row: DirectoryRow) {
  if (!canSaveSettings.value) return
  const affected = directoryRows.value.filter((item) => item.path === row.path || item.path.startsWith(`${row.path}/`))
  const dialog = DialogPlugin.confirm({
    header: '删除目录',
    body: affected.length > 1
      ? `确认删除「${row.path}」及其 ${affected.length - 1} 个子目录配置？文件本身不会被删除。`
      : `确认删除「${row.path}」目录配置？文件本身不会被删除。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: { content: '取消' },
    onConfirm: () => {
      dialog.destroy()
      const nextRows = directoryRows.value.filter((item) => item.path !== row.path && !item.path.startsWith(`${row.path}/`))
      directoryOrders.value = directoryOrders.value
        .map((order) => ({
          ...order,
          paths: order.paths.filter((path) => path !== row.path && !path.startsWith(`${row.path}/`)),
        }))
        .filter((order) => order.paths.length > 0 && order.parentPath !== row.path && !order.parentPath.startsWith(`${row.path}/`))
      directoryRows.value = nextRows
      void persistDirectoryRows(nextRows)
    },
    onCancel: () => dialog.destroy(),
  })
}

function openWorkspace() {
  openMainAppPath(`/platform/knowledge-bases/${kbId.value}`)
}

function backToList() {
  void router.push({ name: 'adminKnowledgeBases' })
}

watch(activeTab, (tab) => {
  if (route.query.tab === tab) return
  void router.replace({ query: { ...route.query, tab } })
})

watch(() => route.query.tab, (tab) => {
  activeTab.value = normalizeTab(tab)
})

watch(visibleTabs, (tabs) => {
  if (!tabs.some((tab) => tab.key === activeTab.value)) {
    activeTab.value = 'basic'
  }
})

watch(kbId, () => {
  void loadKnowledgeBase()
})

onMounted(() => {
  void loadKnowledgeBase()
})
</script>

<style scoped lang="less">
.admin-kb-settings {
  display: grid;
  gap: 18px;
  width: min(100%, 1280px);
}

.admin-kb-state {
  display: flex;
  gap: 10px;
  align-items: center;
  justify-content: center;
  min-height: 280px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  color: var(--admin-text-secondary);
  box-shadow: var(--admin-shadow-sm);
}

.admin-kb-state--error {
  flex-direction: column;
  color: var(--admin-danger);

  strong {
    font-size: 15px;
  }
}

.admin-kb-settings__hero {
  display: flex;
  gap: 18px;
  align-items: flex-start;
  justify-content: space-between;
  min-width: 0;
  padding: 18px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-hero-bg), var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.admin-kb-title {
  display: grid;
  gap: 12px;
  min-width: 0;
}

.admin-kb-title__main {
  display: flex;
  gap: 12px;
  align-items: center;
  min-width: 0;

  span {
    display: grid;
    gap: 6px;
    min-width: 0;
  }

  h2 {
    margin: 0;
    overflow: hidden;
    color: var(--admin-text);
    font-size: 22px;
    font-weight: 650;
    line-height: 1.3;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  p {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
    margin: 0;
  }

  code {
    max-width: 460px;
    overflow: hidden;
    padding: 2px 7px;
    border: 1px solid rgba(219, 228, 231, 0.9);
    border-radius: 6px;
    background: var(--admin-code-bg);
    color: var(--admin-text-muted);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.admin-kb-title-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 10px;
  align-items: center;
}

.admin-kb-settings__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;

  article {
    position: relative;
    display: grid;
    gap: 5px;
    min-width: 0;
    overflow: hidden;
    padding: 15px 16px;
    border: 1px solid var(--admin-border);
    border-radius: 8px;
    background: var(--admin-surface);
    box-shadow: var(--admin-shadow-sm);

    &::before {
      position: absolute;
      top: 0;
      right: 0;
      left: 0;
      height: 3px;
      background: var(--admin-brand);
      content: '';
    }

    &:nth-child(2)::before {
      background: var(--admin-info);
    }

    &:nth-child(3)::before {
      background: #5f6f7a;
    }

    &:nth-child(4)::before {
      background: var(--admin-warning);
    }
  }

  span,
  em {
    overflow: hidden;
    color: var(--admin-text-secondary);
    font-size: 12px;
    font-style: normal;
    line-height: 1.4;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    overflow: hidden;
    color: var(--admin-text);
    font-size: 20px;
    font-weight: 650;
    line-height: 1.2;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.admin-kb-settings__layout {
  display: grid;
  grid-template-columns: 232px minmax(0, 1fr);
  gap: 14px;
  align-items: start;
}

.admin-kb-tabs {
  position: sticky;
  top: 88px;
  display: grid;
  gap: 6px;
  padding: 10px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.admin-kb-tabs button {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  min-height: 48px;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: 8px;
  background: transparent;
  color: var(--admin-text-secondary);
  cursor: pointer;
  text-align: left;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease, box-shadow 0.16s ease;

  .t-icon {
    color: var(--admin-text-muted);
  }

  &:hover {
    border-color: rgba(15, 122, 92, 0.16);
    background: var(--admin-surface-soft);
    color: var(--admin-brand-strong);
  }

  &.active {
    border-color: rgba(15, 122, 92, 0.18);
    background: var(--admin-brand-soft);
    color: var(--admin-brand-strong);
    box-shadow: inset 3px 0 0 var(--admin-brand);
  }

  &:hover .t-icon,
  &.active .t-icon {
    color: var(--admin-brand);
  }

  span {
    display: grid;
    gap: 2px;
    min-width: 0;
  }

  strong,
  small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    font-size: 14px;
    font-weight: 650;
    line-height: 1.3;
  }

  small {
    font-size: 12px;
    line-height: 1.3;
    opacity: 0.76;
  }
}

.admin-kb-settings__content {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.admin-kb-section {
  min-width: 0;
  padding: 22px 24px 26px;

  :deep(.t-form__label) {
    color: var(--admin-text-secondary);
    font-weight: 500;
  }
}

.admin-kb-section__heading {
  margin-bottom: 18px;

  h3 {
    margin: 0 0 5px;
    color: var(--admin-text);
    font-size: 17px;
    font-weight: 650;
    line-height: 1.35;
  }

  p {
    margin: 0;
    color: var(--admin-text-secondary);
    font-size: 13px;
    line-height: 1.5;
  }
}

.admin-kb-section__heading--row {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  justify-content: space-between;
}

.admin-kb-basic-grid,
.admin-kb-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.admin-kb-icon-trigger {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  min-width: 74px;
  height: 40px;
  padding: 0 10px;
  border: 1px solid var(--admin-border-strong);
  border-radius: 8px;
  background: var(--admin-surface);
  color: var(--admin-text-secondary);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, color 0.16s ease;

  &:hover {
    border-color: var(--admin-brand);
    color: var(--admin-brand);
    box-shadow: var(--admin-focus-ring);
  }
}

.admin-kb-icon-picker {
  display: grid;
  grid-template-columns: repeat(6, 36px);
  gap: 6px;
  padding: 10px;
  background: var(--admin-surface);
}

.admin-kb-icon-option {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  color: var(--admin-text-secondary);
  cursor: pointer;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;

  &:hover,
  &.active {
    border-color: var(--admin-brand);
    background: var(--admin-brand-soft);
    color: var(--admin-brand);
  }
}

.admin-kb-icon-input {
  display: none;
}

.admin-kb-setting-block {
  margin-top: 16px;
  padding: 16px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-soft);
}

.admin-kb-setting-block--after {
  margin-top: 22px;
}

.admin-kb-setting-block__title,
.admin-kb-switch-row,
.admin-kb-tag-entry {
  display: flex;
  gap: 14px;
  align-items: flex-start;
  justify-content: space-between;
}

.admin-kb-setting-block__title span,
.admin-kb-switch-row span,
.admin-kb-tag-entry span {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.admin-kb-setting-block__title strong,
.admin-kb-switch-row strong,
.admin-kb-tag-entry strong {
  color: var(--admin-text);
  font-size: 14px;
  font-weight: 650;
  line-height: 1.4;
}

.admin-kb-setting-block__title small,
.admin-kb-switch-row small,
.admin-kb-tag-entry small {
  color: var(--admin-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.admin-kb-indexing-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 14px;

  &.locked {
    opacity: 0.8;
  }
}

.admin-kb-indexing-grid label {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  min-width: 0;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  transition: border-color 0.16s ease, box-shadow 0.16s ease;

  &:hover {
    border-color: rgba(15, 122, 92, 0.26);
    box-shadow: var(--admin-shadow-sm);
  }
}

.admin-kb-settings-divider {
  height: 1px;
  margin: 22px 0;
  background: var(--admin-border);
}

.admin-kb-directory-actions {
  display: flex;
  justify-content: flex-end;
}

.admin-kb-directory-table {
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);

  :deep(.t-table__header th) {
    background: var(--admin-surface-soft);
    color: var(--admin-text-secondary);
    font-size: 12px;
    font-weight: 650;
  }

  :deep(.t-table__body td) {
    border-bottom-color: rgba(219, 228, 231, 0.74);
  }
}

.admin-kb-directory-path {
  display: inline-block;
  max-width: 100%;
  overflow: hidden;
  padding: 2px 6px;
  border-radius: 6px;
  background: var(--admin-surface-soft);
  color: var(--admin-text-secondary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-kb-directory-desc {
  display: block;
  overflow: hidden;
  color: var(--admin-text-secondary);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.admin-kb-directory-row-actions {
  display: inline-flex;
  gap: 4px;
  align-items: center;
}

.admin-kb-empty {
  display: inline-flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
  width: 100%;
  min-height: 122px;
  color: var(--admin-text-muted);
  font-size: 13px;
}

.admin-kb-tag-entry {
  align-items: center;
}

@media (max-width: 1080px) {
  .admin-kb-settings__hero,
  .admin-kb-title-actions {
    align-items: stretch;
    flex-direction: column;
  }

  .admin-kb-settings__summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .admin-kb-settings__layout {
    grid-template-columns: 1fr;
  }

  .admin-kb-tabs {
    position: static;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .admin-kb-settings__summary,
  .admin-kb-basic-grid,
  .admin-kb-form-grid,
  .admin-kb-indexing-grid,
  .admin-kb-tabs {
    grid-template-columns: 1fr;
  }

  .admin-kb-section {
    padding: 18px 16px 22px;
  }
}
</style>
