<template>
  <section class="expert-admin">
    <div class="expert-admin__header">
      <div>
        <h2>专家维护</h2>
        <p>后台维护专家包、版本发布和运行测试。</p>
      </div>
      <div class="expert-admin__actions">
        <t-button variant="outline" :loading="loading" @click="loadExpertPage">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <t-button theme="primary" @click="openImportDialog">
          <template #icon><t-icon name="upload" /></template>
          导入专家包
        </t-button>
      </div>
    </div>

    <t-alert
      v-if="errorMessage"
      theme="error"
      :message="errorMessage"
      class="expert-admin__alert"
    >
      <template #operation>
        <t-button size="small" variant="outline" @click="loadExpertPage">重试</t-button>
      </template>
    </t-alert>

    <div class="expert-summary">
      <article>
        <span>专家包</span>
        <strong>{{ packages.length }}</strong>
        <em>后台维护</em>
      </article>
      <article>
        <span>版本</span>
        <strong>{{ versionCount }}</strong>
        <em>{{ publishedVersionCount }} 个已发布</em>
      </article>
      <article>
        <span>专家定义</span>
        <strong>{{ definitionCount }}</strong>
        <em>{{ publishedDefinitionCount }} 个已发布</em>
      </article>
      <article>
        <span>使用范围</span>
        <strong>服务空间</strong>
        <em>发布后可在服务空间使用</em>
      </article>
    </div>

    <section class="expert-admin__grid">
      <div class="expert-panel expert-panel--packages">
        <div class="expert-panel__title">
          <span>
            <strong>专家包</strong>
            <em>导入后先查看诊断，通过后发布版本。</em>
          </span>
        </div>

        <div v-if="loading && packages.length === 0" class="expert-empty">
          <t-loading size="small" />
          <span>正在读取专家包...</span>
        </div>
        <div v-else-if="packages.length === 0" class="expert-empty">
          <t-icon name="info-circle" />
          <span>暂无专家包</span>
        </div>

        <div v-else class="package-list">
          <article
            v-for="pkg in packages"
            :key="pkg.id"
            class="package-card"
            :class="{ 'package-card--active': selectedPackage?.id === pkg.id }"
            @click="selectPackage(pkg.id)"
          >
            <div class="package-card__head">
              <AgentAvatar
                class="package-card__icon"
                :name="pkg.display_name"
                :avatar="pkg.avatar"
                size="medium"
              />
              <span class="package-card__title">
                <strong>{{ pkg.display_name }}</strong>
                <em>{{ pkg.package_key }}</em>
              </span>
              <t-tag theme="primary" variant="light">{{ sourceLabel(pkg.source_format) }}</t-tag>
            </div>

            <p>{{ pkg.description || '未填写描述' }}</p>

            <div class="package-card__meta">
              <span>版本 {{ pkg.versions?.length || 0 }}</span>
              <span>专家 {{ packageDefinitionCount(pkg) }}</span>
              <span>更新 {{ formatDate(pkg.updated_at || pkg.created_at) }}</span>
            </div>

            <div class="version-list">
              <div v-for="version in sortedVersions(pkg)" :key="version.id" class="version-row">
                <div class="version-row__main">
                  <strong>v{{ version.version }}</strong>
                  <t-tag :theme="versionStateTheme(version.state)" variant="light">
                    {{ versionStateLabel(version.state) }}
                  </t-tag>
                  <t-tag v-if="diagnosticItems(version, 'blocking').length" theme="danger" variant="light">
                    阻断 {{ diagnosticItems(version, 'blocking').length }}
                  </t-tag>
                  <t-tag v-if="diagnosticItems(version, 'warnings').length" theme="warning" variant="light">
                    警告 {{ diagnosticItems(version, 'warnings').length }}
                  </t-tag>
                </div>
                <div class="version-row__actions">
                  <span>{{ version.definitions?.length || 0 }} 个专家</span>
                  <t-button
                    v-if="version.state !== 'published'"
                    size="small"
                    theme="primary"
                    variant="outline"
                    :loading="publishingVersionId === version.id"
                    :disabled="diagnosticItems(version, 'blocking').length > 0"
                    @click.stop="publishVersion(pkg.id, version)"
                  >
                    发布
                  </t-button>
                </div>
              </div>
            </div>
          </article>
        </div>
      </div>

      <aside class="expert-panel expert-panel--detail">
        <div v-if="!selectedPackage" class="expert-empty expert-empty--detail">
          <t-icon name="info-circle" />
          <span>选择一个专家包查看详情</span>
        </div>

        <template v-else>
          <div class="expert-panel__title">
            <div class="expert-panel__identity">
              <AgentAvatar
                :name="selectedPackage.display_name"
                :avatar="selectedPackage.avatar"
                size="medium"
              />
              <span>
                <strong>{{ selectedPackage.display_name }}</strong>
                <em>{{ selectedPackage.package_key }}</em>
              </span>
            </div>
            <t-tag theme="primary" variant="light">{{ sourceLabel(selectedPackage.source_format) }}</t-tag>
          </div>

          <dl class="package-detail">
            <div>
              <dt>来源</dt>
              <dd>{{ selectedPackage.source_uri || '未记录' }}</dd>
            </div>
            <div>
              <dt>许可</dt>
              <dd>{{ selectedPackage.license || '未记录' }}</dd>
            </div>
            <div>
              <dt>创建</dt>
              <dd>{{ formatDate(selectedPackage.created_at) }}</dd>
            </div>
          </dl>

          <section class="definition-section">
            <div class="definition-section__title">
              <strong>专家定义</strong>
              <em>只有已发布版本可以测试并在服务空间使用。</em>
            </div>

            <div v-if="selectedDefinitions.length === 0" class="expert-empty expert-empty--compact">
              <span>暂无专家定义</span>
            </div>
            <div v-else class="definition-list">
              <div
                v-for="entry in selectedDefinitions"
                :key="entry.definition.id"
                class="definition-row"
              >
                <div class="definition-row__head">
                  <span>
                    <strong>{{ entry.definition.display_name }}</strong>
                    <em>{{ entry.definition.agent_id }} · v{{ entry.version.version }}</em>
                  </span>
                  <div class="definition-row__actions">
                    <t-tag :theme="entry.version.state === 'published' ? 'success' : 'warning'" variant="light">
                      {{ versionStateLabel(entry.version.state) }}
                    </t-tag>
                    <t-button
                      v-if="entry.version.state === 'published'"
                      size="small"
                      variant="outline"
                      @click="openTestDialog(entry)"
                    >
                      <template #icon><t-icon name="play-circle" /></template>
                      测试
                    </t-button>
                  </div>
                </div>
                <p>{{ entry.definition.description || entry.package.description || '未填写描述' }}</p>
                <div class="definition-row__meta">
                  <span>领域 {{ entry.definition.domain || '未配置' }}</span>
                  <span>产物 {{ entry.definition.output_contract || '未配置' }}</span>
                  <span>迭代 {{ maxIterations(entry.definition) }}</span>
                </div>
                <div v-if="allowedTools(entry.definition).length" class="definition-tools">
                  <t-tag
                    v-for="tool in allowedTools(entry.definition)"
                    :key="tool"
                    size="small"
                    variant="light"
                  >
                    {{ tool }}
                  </t-tag>
                </div>
              </div>
            </div>
          </section>

        </template>
      </aside>
    </section>

    <t-dialog
      v-model:visible="importDialogVisible"
      header="导入专家包"
      width="760px"
      :confirm-btn="{ content: '导入', theme: 'primary', loading: importing, disabled: !selectedImportFile }"
      :cancel-btn="{ content: '取消', disabled: importing }"
      :close-on-overlay-click="!importing"
      destroy-on-close
      @confirm="submitImport"
    >
      <div class="import-dialog">
        <div class="archive-upload" :class="{ 'archive-upload--selected': selectedImportFile }">
          <input
            ref="archiveInputRef"
            type="file"
            accept=".zip,application/zip,application/x-zip-compressed"
            :disabled="importing"
            @change="handleArchiveFileChange"
          >
          <span class="archive-upload__icon">
            <t-icon name="upload" />
          </span>
          <span>
            <strong>{{ selectedImportFile?.name || '选择 WorkBuddy 专家包 ZIP' }}</strong>
            <em>{{ selectedImportFile ? formatFileSize(selectedImportFile.size) : '支持 .codebuddy-plugin/plugin.json、agents、skills、avatars 等目录结构' }}</em>
          </span>
          <t-button size="small" variant="outline" :disabled="importing" @click="chooseArchiveFile">
            选择文件
          </t-button>
        </div>
        <div v-if="importing || uploadProgress > 0" class="archive-upload__progress">
          <div><span :style="{ width: `${uploadProgress}%` }" /></div>
          <em>{{ uploadProgress }}%</em>
        </div>
        <div class="archive-upload__meta">
          <span>最大 50 MB</span>
          <span>脚本和 vendor 首版只保存并扫描，不执行</span>
          <span>发布前会展示阻断项和降级警告</span>
        </div>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="testDialogVisible"
      :header="testDialogTitle"
      width="900px"
      :footer="false"
      :close-on-overlay-click="!testingExpert && !submittingExpertAnswers"
      destroy-on-close
    >
      <div class="expert-test">
        <t-alert
          v-if="!modelLoading && chatModels.length === 0"
          theme="warning"
          message="当前工作区没有可用的对话模型，请先在系统设置中配置 KnowledgeQA 模型。"
        />

        <t-form :data="testForm" label-align="top" class="expert-test__form" @submit.prevent>
          <t-form-item label="执行模型">
            <t-select
              v-model="testForm.modelId"
              :options="modelOptions"
              :loading="modelLoading"
              :disabled="testingExpert || modelOptions.length === 0"
              placeholder="选择对话模型"
            />
          </t-form-item>
          <t-form-item label="测试问题">
            <t-textarea
              v-model="testForm.prompt"
              :disabled="testingExpert"
              :maxlength="12000"
              :autosize="{ minRows: 5, maxRows: 10 }"
              placeholder="输入一条完整、可验收的业务请求"
            />
          </t-form-item>
          <div class="expert-test__submit">
            <span>测试结果只保存在执行记录中，不会生成正式服务卡片。</span>
            <t-button
              theme="primary"
              :loading="testingExpert"
              :disabled="!canRunExpertTest"
              @click="runExpertTest"
            >
              <template #icon><t-icon name="play-circle" /></template>
              开始测试
            </t-button>
          </div>
        </t-form>

        <section v-if="testRun || testError" class="expert-test__result">
          <div class="expert-test__status">
            <span>
              <strong>执行结果</strong>
              <em v-if="testRun">运行 ID {{ compactId(testRun.id) }}</em>
            </span>
            <t-tag v-if="testRun" :theme="agentRunStatusTheme(testRun.status)" variant="light">
              {{ agentRunStatusLabel(testRun.status) }}
            </t-tag>
          </div>

          <t-alert v-if="testError" theme="error" :message="testError" />

          <section v-if="testInteraction" class="expert-test__intake">
            <div>
              <strong>专家需要补充信息</strong>
              <p>补齐这些关键条件后，将从规划阶段继续执行。</p>
            </div>
            <t-form label-align="top" @submit.prevent>
              <t-form-item
                v-for="question in testInteraction.questions"
                :key="question.id"
                :label="question.required ? `${question.label}（必填）` : question.label"
                :help="question.description"
              >
                <t-select
                  v-if="question.type === 'single_choice'"
                  :value="testAnswers[question.id] as string"
                  :options="questionOptions(question)"
                  :disabled="submittingExpertAnswers"
                  clearable
                  @change="setTestAnswer(question.id, $event)"
                />
                <t-select
                  v-else-if="question.type === 'multi_choice'"
                  :value="testAnswers[question.id] as string[]"
                  :options="questionOptions(question)"
                  :disabled="submittingExpertAnswers"
                  multiple
                  clearable
                  @change="setTestAnswer(question.id, $event)"
                />
                <t-date-picker
                  v-else-if="question.type === 'date'"
                  :value="testAnswers[question.id] as string"
                  :disabled="submittingExpertAnswers"
                  clearable
                  @change="setTestAnswer(question.id, $event)"
                />
                <t-input-number
                  v-else-if="question.type === 'number'"
                  :value="testAnswers[question.id] as number"
                  :disabled="submittingExpertAnswers"
                  theme="normal"
                  @change="setTestAnswer(question.id, $event)"
                />
                <t-input
                  v-else
                  :value="testAnswers[question.id] as string"
                  :disabled="submittingExpertAnswers"
                  clearable
                  @change="setTestAnswer(question.id, $event)"
                />
              </t-form-item>
            </t-form>
            <div class="expert-test__intake-actions">
              <span>答案会保存为本次运行的新输入版本。</span>
              <t-button
                theme="primary"
                :loading="submittingExpertAnswers"
                :disabled="!canSubmitExpertAnswers"
                @click="submitExpertAnswers"
              >
                继续执行
              </t-button>
            </div>
          </section>

          <section v-if="testSteps.length" class="expert-test__workflow">
            <div class="expert-test__section-title">
              <strong>执行步骤</strong>
              <em>{{ testRun?.phase ? agentRunPhaseLabel(testRun.phase) : '准备中' }}</em>
            </div>
            <div class="expert-test__steps">
              <article v-for="step in testSteps" :key="step.id">
                <span class="expert-test__step-index">{{ step.sequence }}</span>
                <span>
                  <strong>{{ agentRunPhaseLabel(step.step_type) }}</strong>
                  <em v-if="step.error">{{ step.error }}</em>
                  <em v-else>{{ step.model_id || '平台步骤' }}</em>
                </span>
                <t-tag :theme="agentRunStepTheme(step.status)" variant="light">
                  {{ agentRunStepLabel(step.status) }}
                </t-tag>
              </article>
            </div>
          </section>

          <section v-if="testQuality" class="expert-test__quality">
            <div class="expert-test__section-title">
              <span>
                <strong>质量门禁</strong>
                <em>{{ testQuality.summary || '已完成独立审查' }}</em>
              </span>
              <span class="expert-test__quality-score">{{ testQuality.score ?? 0 }}</span>
              <t-tag :theme="testQuality.passed ? 'success' : 'danger'" variant="light">
                {{ testQuality.passed ? '通过' : '未通过' }}
              </t-tag>
            </div>
            <div v-if="testRun?.status === 'succeeded' || testRun?.status === 'failed'" class="expert-test__quality-actions">
              <t-button
                variant="outline"
                :loading="regeneratingExpert"
                :disabled="regeneratingExpert || testingExpert || submittingExpertAnswers"
                @click="regenerateExpertTest"
              >
                <template #icon><t-icon name="refresh" /></template>
                按意见重新生成
              </t-button>
            </div>
            <div v-if="qualityDimensions.length" class="expert-test__dimensions">
              <span v-for="item in qualityDimensions" :key="item.label">
                {{ item.label }} {{ item.score }}
              </span>
            </div>
            <div v-if="testQuality.issues?.length" class="expert-test__issues">
              <article v-for="issue in testQuality.issues" :key="`${issue.code}-${issue.section || ''}`">
                <t-tag :theme="qualityIssueTheme(issue.severity)" variant="light" size="small">
                  {{ qualityIssueLabel(issue.severity) }}
                </t-tag>
                <span>
                  <strong>{{ issue.section || issue.code }}</strong>
                  <p>{{ issue.message }}</p>
                  <em>{{ issue.instruction }}</em>
                </span>
              </article>
            </div>
          </section>

          <template v-if="testRun?.result">
            <t-alert
              v-if="testValidationErrors.length"
              theme="warning"
              :message="`结果协议校验失败：${testValidationErrors.join('；')}`"
            />

            <div v-if="testRun.result.card" class="expert-test__card">
              <strong>服务卡片预览</strong>
              <dl>
                <div>
                  <dt>标题</dt>
                  <dd>{{ testRun.result.card.title }}</dd>
                </div>
                <div>
                  <dt>摘要</dt>
                  <dd>{{ testRun.result.card.summary }}</dd>
                </div>
                <div>
                  <dt>下一步动作</dt>
                  <dd>{{ testRun.result.card.next_action }}</dd>
                </div>
              </dl>
            </div>
            <t-alert
              v-else-if="testRun.status === 'succeeded'"
              theme="info"
              :message="testRun.result.decision?.reason || '专家判断本次不生成服务卡片。'"
            />

            <div v-if="primaryTestReport" class="expert-test__report">
              <div>
                <strong>{{ primaryTestReport.title }}</strong>
                <p>{{ primaryTestReport.executive_summary }}</p>
              </div>
              <section
                v-for="(section, index) in primaryTestReport.sections"
                :key="`${section.type}-${index}`"
              >
                <h4>{{ section.title }}</h4>
                <p v-if="section.content">{{ section.content }}</p>
                <ul v-if="section.items?.length">
                  <li v-for="item in section.items" :key="item">{{ item }}</li>
                </ul>
              </section>
            </div>
          </template>
        </section>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import AgentAvatar from '@/components/AgentAvatar.vue'
import { listModels, type ModelConfig } from '@/api/model'
import {
  importExpertPackageArchive,
  listExpertPackages,
  publishExpertPackageVersion,
  testExpertPackageAgent,
  type AgentDefinitionVersion,
  type ExpertPackage,
  type ExpertPackageVersion,
} from '@/api/expert-package'
import {
  getAgentRun,
  listAgentRunSteps,
  regenerateAgentRun,
  submitAgentRunAnswers,
  type AgentRun,
  type AgentRunQuality,
  type AgentRunStep,
} from '@/api/agent-run'
import type {
  ExpertIntakeInteraction,
  ExpertIntakeQuestion,
  StructuredReportV1,
} from '@/api/agent-run-types'

interface DefinitionEntry {
  package: ExpertPackage
  version: ExpertPackageVersion
  definition: AgentDefinitionVersion
}

interface ExpertTestForm {
  modelId: string
  prompt: string
}

const packages = ref<ExpertPackage[]>([])
const chatModels = ref<ModelConfig[]>([])
const selectedPackageId = ref('')
const loading = ref(false)
const errorMessage = ref('')
const importing = ref(false)
const importDialogVisible = ref(false)
const archiveInputRef = ref<HTMLInputElement | null>(null)
const selectedImportFile = ref<File | null>(null)
const uploadProgress = ref(0)
const publishingVersionId = ref('')
const modelLoading = ref(false)
const testDialogVisible = ref(false)
const testingExpert = ref(false)
const testDefinitionEntry = ref<DefinitionEntry | null>(null)
const testRun = ref<AgentRun | null>(null)
const testSteps = ref<AgentRunStep[]>([])
const testAnswers = ref<Record<string, unknown>>({})
const testError = ref('')
const submittingExpertAnswers = ref(false)
const regeneratingExpert = ref(false)
const testForm = ref<ExpertTestForm>({
  modelId: '',
  prompt: '',
})

const selectedPackage = computed(() => {
  return packages.value.find((pkg) => pkg.id === selectedPackageId.value) || packages.value[0] || null
})

const allDefinitionEntries = computed<DefinitionEntry[]>(() => (
  packages.value.flatMap((pkg) => sortedVersions(pkg).flatMap((version) => (
    (version.definitions || []).map((definition) => ({ package: pkg, version, definition }))
  )))
))

const selectedDefinitions = computed<DefinitionEntry[]>(() => {
  const pkg = selectedPackage.value
  if (!pkg) return []
  return sortedVersions(pkg).flatMap((version) => (
    (version.definitions || []).map((definition) => ({ package: pkg, version, definition }))
  ))
})

const versionCount = computed(() => (
  packages.value.reduce((total, pkg) => total + (pkg.versions?.length || 0), 0)
))

const publishedVersionCount = computed(() => (
  packages.value.reduce((total, pkg) => (
    total + (pkg.versions || []).filter((version) => version.state === 'published').length
  ), 0)
))

const definitionCount = computed(() => allDefinitionEntries.value.length)

const publishedDefinitionCount = computed(() => (
  allDefinitionEntries.value.filter((entry) => entry.version.state === 'published').length
))

const modelOptions = computed(() => (
  chatModels.value.map((model) => ({
    label: [
      model.display_name || model.name,
      model.is_default ? '默认' : '',
    ].filter(Boolean).join(' · '),
    value: model.id || '',
  })).filter((option) => option.value)
))

const canRunExpertTest = computed(() => (
  Boolean(
    testDefinitionEntry.value &&
    testForm.value.modelId &&
    testForm.value.prompt.trim() &&
    !testingExpert.value,
  )
))

const testDialogTitle = computed(() => {
  const entry = testDefinitionEntry.value
  return entry ? `测试专家 · ${entry.definition.display_name}` : '测试专家'
})

const testValidationErrors = computed(() => (
  testRun.value?.result?.validation?.errors || []
))

const testInteraction = computed<ExpertIntakeInteraction | null>(() => {
  if (testRun.value?.status !== 'waiting_input') return null
  const interaction = testRun.value.interaction
  if (!interaction || interaction.schema_version !== 'intake_request_v1' || !Array.isArray(interaction.questions)) {
    return null
  }
  return interaction as ExpertIntakeInteraction
})

const canSubmitExpertAnswers = computed(() => (
  Boolean(
    testInteraction.value &&
    !submittingExpertAnswers.value &&
    testInteraction.value.questions.every((question) => (
      !question.required || !expertAnswerIsBlank(testAnswers.value[question.id])
    )),
  )
))

const testQuality = computed<AgentRunQuality | null>(() => {
  const quality = testRun.value?.quality
  if (!quality || typeof quality.score !== 'number') return null
  return quality
})

const qualityDimensions = computed(() => (
  Object.entries(testQuality.value?.dimensions || {}).map(([label, score]) => ({ label, score }))
))

const primaryTestReport = computed<StructuredReportV1 | null>(() => {
  const artifact = testRun.value?.result?.artifacts?.find((item) => (
    item.role === 'primary' && item.kind === 'report'
  ))
  const content = artifact?.content
  if (!content || content.format !== 'structured_report_v1') return null
  return content as StructuredReportV1
})

async function loadExpertPage() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  await Promise.all([
    loadPackages(),
    loadChatModels(),
  ]).finally(() => {
    loading.value = false
  })
}

async function loadChatModels() {
  if (modelLoading.value) return
  modelLoading.value = true
  try {
    const models = await listModels('KnowledgeQA')
    chatModels.value = models.filter((model) => !model.status || model.status === 'active')
    normalizeTestModel()
  } catch (error) {
    console.warn('[AdminExpertPackages] Failed to load chat models:', error)
    chatModels.value = []
  } finally {
    modelLoading.value = false
  }
}

async function loadPackages() {
  try {
    const packageResponse = await listExpertPackages()
    packages.value = packageResponse?.data || []
    if (!selectedPackageId.value || !packages.value.some((pkg) => pkg.id === selectedPackageId.value)) {
      selectedPackageId.value = packages.value[0]?.id || ''
    }
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to load expert packages:', error)
    errorMessage.value = error?.message || '专家包读取失败'
    packages.value = []
  }
}

function selectPackage(id: string) {
  selectedPackageId.value = id
}

function openImportDialog() {
  selectedImportFile.value = null
  uploadProgress.value = 0
  importDialogVisible.value = true
}

function chooseArchiveFile() {
  archiveInputRef.value?.click()
}

function handleArchiveFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0] || null
  if (!file) {
    selectedImportFile.value = null
    return
  }
  const isZip = file.name.toLowerCase().endsWith('.zip')
  if (!isZip) {
    MessagePlugin.warning('请选择 .zip 专家包')
    input.value = ''
    selectedImportFile.value = null
    return
  }
  if (file.size > 50 * 1024 * 1024) {
    MessagePlugin.warning('专家包 ZIP 不能超过 50 MB')
    input.value = ''
    selectedImportFile.value = null
    return
  }
  selectedImportFile.value = file
  uploadProgress.value = 0
}

async function submitImport() {
  const file = selectedImportFile.value
  if (!file) {
    MessagePlugin.warning('请选择专家包 ZIP')
    return
  }

  importing.value = true
  uploadProgress.value = 0
  try {
    const response = await importExpertPackageArchive(file, (event) => {
      if (event.total) {
        uploadProgress.value = Math.min(99, Math.round(((event.loaded || 0) / event.total) * 100))
      }
    })
    uploadProgress.value = 100
    MessagePlugin.success('专家包已导入')
    importDialogVisible.value = false
    selectedImportFile.value = null
    if (archiveInputRef.value) archiveInputRef.value.value = ''
    selectedPackageId.value = response?.data?.package_id || selectedPackageId.value
    await loadPackages()
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to import expert package:', error)
    MessagePlugin.error(error?.message || '专家包导入失败')
  } finally {
    importing.value = false
  }
}

async function publishVersion(packageId: string, version: ExpertPackageVersion) {
  if (diagnosticItems(version, 'blocking').length > 0) {
    MessagePlugin.warning('该版本存在阻断诊断，暂不能发布')
    return
  }
  publishingVersionId.value = version.id
  try {
    await publishExpertPackageVersion(packageId, version.id)
    MessagePlugin.success('专家版本已发布')
    await loadPackages()
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to publish expert package:', error)
    MessagePlugin.error(error?.message || '专家版本发布失败')
  } finally {
    publishingVersionId.value = ''
  }
}

function openTestDialog(entry: DefinitionEntry) {
  testDefinitionEntry.value = entry
  testRun.value = null
  testSteps.value = []
  testAnswers.value = {}
  testError.value = ''
  testForm.value.prompt = ''
  normalizeTestModel()
  testDialogVisible.value = true
}

function normalizeTestModel() {
  const currentExists = chatModels.value.some((model) => model.id === testForm.value.modelId)
  if (currentExists) return
  testForm.value.modelId = (
    chatModels.value.find((model) => model.is_default)?.id ||
    chatModels.value[0]?.id ||
    ''
  )
}

async function runExpertTest() {
  const entry = testDefinitionEntry.value
  if (!entry || !canRunExpertTest.value) return

  testingExpert.value = true
  testRun.value = null
  testError.value = ''
  try {
    const response = await testExpertPackageAgent(entry.package.id, entry.definition.id, {
      prompt: testForm.value.prompt.trim(),
      model_id: testForm.value.modelId,
    })
    if (!response?.data) throw new Error(response?.message || '测试任务创建失败')
    testRun.value = response.data
    await pollExpertTestRun(response.data.id)
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Expert test failed:', error)
    testError.value = error?.message || '专家测试失败'
  } finally {
    testingExpert.value = false
  }
}

async function pollExpertTestRun(id: string) {
  const deadline = Date.now() + 5 * 60 * 1000
  while (Date.now() < deadline) {
    const [response, stepsResponse] = await Promise.all([
      getAgentRun(id),
      listAgentRunSteps(id),
    ])
    if (!response?.data) throw new Error(response?.message || '任务状态读取失败')
    testRun.value = response.data
    testSteps.value = stepsResponse?.data || []
    if (response.data.status === 'waiting_input') {
      initializeTestAnswers()
      MessagePlugin.info('专家需要补充关键信息')
      return
    }
    if (response.data.status === 'succeeded') {
      MessagePlugin.success('专家测试完成')
      return
    }
    if (response.data.status === 'failed' || response.data.status === 'cancelled') {
      throw new Error(response.data.error_message || '专家测试未完成')
    }
    await new Promise<void>((resolve) => window.setTimeout(resolve, 1200))
  }
  throw new Error('专家测试仍在执行，请稍后重新查看')
}

function initializeTestAnswers() {
  const answers = { ...testAnswers.value }
  for (const question of testInteraction.value?.questions || []) {
    if (Object.prototype.hasOwnProperty.call(answers, question.id)) continue
    answers[question.id] = question.type === 'multi_choice' ? [] : ''
  }
  testAnswers.value = answers
}

function setTestAnswer(id: string, value: unknown) {
  testAnswers.value = {
    ...testAnswers.value,
    [id]: value,
  }
}

function questionOptions(question: ExpertIntakeQuestion) {
  return (question.options || []).map((option) => ({ label: option, value: option }))
}

async function submitExpertAnswers() {
  const run = testRun.value
  if (!run || !canSubmitExpertAnswers.value) return
  submittingExpertAnswers.value = true
  testError.value = ''
  try {
    const response = await submitAgentRunAnswers(run.id, testAnswers.value)
    if (!response?.data) throw new Error(response?.message || '补充信息提交失败')
    testRun.value = response.data
    await pollExpertTestRun(run.id)
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to submit expert answers:', error)
    testError.value = error?.message || '补充信息提交失败'
  } finally {
    submittingExpertAnswers.value = false
  }
}

async function regenerateExpertTest() {
  const run = testRun.value
  if (!run || (run.status !== 'succeeded' && run.status !== 'failed')) return
  const feedback = window.prompt('请输入纠偏意见（可选）：', '')
  if (feedback === null) return
  regeneratingExpert.value = true
  testError.value = ''
  try {
    const response = await regenerateAgentRun(run.id, feedback)
    if (!response?.data) throw new Error(response?.message || '重新生成任务创建失败')
    testRun.value = response.data
    testSteps.value = []
    testAnswers.value = {}
    await pollExpertTestRun(response.data.id)
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to regenerate expert test:', error)
    testError.value = error?.message || '重新生成失败'
  } finally {
    regeneratingExpert.value = false
  }
}

function expertAnswerIsBlank(value: unknown) {
  if (value === null || value === undefined) return true
  if (typeof value === 'string') return value.trim() === ''
  if (Array.isArray(value)) return value.length === 0
  return false
}

function sortedVersions(pkg: ExpertPackage) {
  return [...(pkg.versions || [])].sort((a, b) => {
    const aTime = new Date(a.created_at || a.updated_at || '').getTime()
    const bTime = new Date(b.created_at || b.updated_at || '').getTime()
    if (!Number.isNaN(aTime) && !Number.isNaN(bTime) && aTime !== bTime) {
      return bTime - aTime
    }
    return String(b.version || '').localeCompare(String(a.version || ''))
  })
}

function packageDefinitionCount(pkg: ExpertPackage) {
  return (pkg.versions || []).reduce((total, version) => total + (version.definitions?.length || 0), 0)
}

function diagnosticItems(version: ExpertPackageVersion, key: 'blocking' | 'warnings') {
  const value = version.diagnostics?.[key]
  return Array.isArray(value) ? value.map(String).filter(Boolean) : []
}

function allowedTools(definition: AgentDefinitionVersion) {
  const value = definition.compiled_config?.allowed_tools
  return Array.isArray(value) ? value.map(String).filter(Boolean) : []
}

function maxIterations(definition: AgentDefinitionVersion) {
  const value = definition.compiled_config?.max_iterations
  return typeof value === 'number' ? value : '未配置'
}

function compactId(id: string) {
  return id.length > 10 ? `${id.slice(0, 6)}...${id.slice(-4)}` : id
}

function sourceLabel(source: string) {
  const labels: Record<string, string> = {
    workbuddy: 'WorkBuddy',
    'ruile-native': '睿乐',
    'custom-yaml': '自定义',
  }
  return labels[source] || source || '未知'
}

function versionStateLabel(state: string) {
  const labels: Record<string, string> = {
    testing: '测试中',
    published: '已发布',
    deprecated: '已废弃',
    archived: '已归档',
  }
  return labels[state] || state || '未知'
}

function versionStateTheme(state: string) {
  if (state === 'published') return 'success'
  if (state === 'testing') return 'warning'
  if (state === 'deprecated') return 'danger'
  return 'default'
}

function agentRunStatusLabel(status: string) {
  const labels: Record<string, string> = {
    queued: '排队中',
    running: '执行中',
    waiting_input: '等待补充',
    succeeded: '已完成',
    failed: '失败',
    cancelled: '已取消',
  }
  return labels[status] || status
}

function agentRunStatusTheme(status: string) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed' || status === 'cancelled') return 'danger'
  if (status === 'running') return 'primary'
  return 'warning'
}

function agentRunPhaseLabel(phase: string) {
  const labels: Record<string, string> = {
    intake: '需求澄清',
    planning: '执行规划',
    drafting: '生成初稿',
    reviewing: '质量审查',
    revising: '定向修订',
    packaging: '结果封装',
    completed: '执行完成',
  }
  return labels[phase] || phase
}

function agentRunStepLabel(status: string) {
  const labels: Record<string, string> = {
    running: '执行中',
    succeeded: '已完成',
    failed: '失败',
  }
  return labels[status] || status
}

function agentRunStepTheme(status: string) {
  if (status === 'succeeded') return 'success'
  if (status === 'failed') return 'danger'
  return 'primary'
}

function qualityIssueLabel(severity: string) {
  const labels: Record<string, string> = {
    warning: '提醒',
    error: '问题',
    red_line: '红线',
  }
  return labels[severity] || severity
}

function qualityIssueTheme(severity: string) {
  if (severity === 'red_line' || severity === 'error') return 'danger'
  return 'warning'
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function formatFileSize(size: number) {
  if (size >= 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`
  if (size >= 1024) return `${(size / 1024).toFixed(1)} KB`
  return `${size} B`
}

onMounted(() => {
  void loadExpertPage()
})
</script>

<style scoped>
.expert-admin {
  display: grid;
  gap: 18px;
  width: min(100%, 1320px);
}

.expert-admin__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 18px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-hero-bg), var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.expert-admin__header h2 {
  margin: 0 0 6px;
  color: var(--admin-text);
  font-size: 22px;
  font-weight: 650;
  line-height: 1.35;
}

.expert-admin__header p {
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.expert-admin__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 10px;
}

.expert-admin__alert {
  border-radius: 8px;
}

.expert-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.expert-summary article,
.expert-panel {
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.expert-summary article {
  display: grid;
  gap: 4px;
  padding: 16px;
}

.expert-summary span,
.package-card__title em,
.expert-panel__title em,
.definition-section__title em,
.definition-row__head em {
  color: var(--admin-text-secondary);
}

.expert-summary strong {
  color: var(--admin-text);
  font-size: 26px;
  font-weight: 680;
  line-height: 1.2;
}

.expert-summary em {
  color: var(--admin-text-tertiary);
  font-size: 12px;
  font-style: normal;
}

.expert-admin__grid {
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(360px, 0.85fr);
  gap: 16px;
  align-items: start;
}

.expert-panel {
  min-width: 0;
  padding: 16px;
}

.expert-panel--detail {
  position: sticky;
  top: 18px;
}

.expert-panel__title,
.package-card__head,
.definition-row__head {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: flex-start;
}

.expert-panel__title {
  margin-bottom: 14px;
}

.expert-panel__title span,
.definition-section__title,
.definition-row__head span {
  display: grid;
  gap: 4px;
}

.expert-panel__identity {
  display: flex;
  min-width: 0;
  align-items: center;
  gap: 10px;
}

.expert-panel__identity > span {
  min-width: 0;
}

.expert-panel__title strong,
.definition-section__title strong {
  color: var(--admin-text);
  font-size: 16px;
}

.package-list,
.definition-list {
  display: grid;
  gap: 12px;
}

.package-card {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-muted);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease, transform 0.16s ease;
}

.package-card:hover,
.package-card--active {
  border-color: var(--td-brand-color);
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}

.package-card:hover {
  transform: translateY(-1px);
}

.package-card__head {
  align-items: center;
}

.package-card__icon {
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  border-radius: 8px;
  box-shadow: none;
}

.package-card__title {
  display: grid;
  flex: 1;
  min-width: 0;
  gap: 3px;
}

.package-card__title strong,
 .definition-row__head strong {
  overflow: hidden;
  color: var(--admin-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.package-card__title em,
 .definition-row__head em {
  overflow: hidden;
  font-size: 12px;
  font-style: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.package-card p,
.definition-row p {
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.package-card__meta,
.definition-row__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  color: var(--admin-text-tertiary);
  font-size: 12px;
}

.package-card__meta span,
.definition-row__meta span {
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--admin-surface);
}

.version-list {
  display: grid;
  gap: 8px;
}

.version-row {
  display: grid;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
}

.version-row__main,
.version-row__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.version-row__main strong {
  color: var(--admin-text);
}

.version-row__actions {
  justify-content: space-between;
  color: var(--admin-text-tertiary);
  font-size: 12px;
}

.expert-empty {
  display: flex;
  min-height: 120px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: 1px dashed var(--admin-border);
  border-radius: 8px;
  color: var(--admin-text-secondary);
  background: var(--admin-surface-muted);
}

.expert-empty--detail {
  min-height: 360px;
}

.expert-empty--compact {
  min-height: 58px;
  font-size: 13px;
}

.package-detail {
  display: grid;
  gap: 10px;
  margin: 0 0 18px;
}

.package-detail div {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  align-items: baseline;
}

.package-detail dt {
  color: var(--admin-text-tertiary);
  font-size: 12px;
}

.package-detail dd {
  min-width: 0;
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--admin-text-secondary);
  font-size: 13px;
}

.definition-section {
  display: grid;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--admin-border);
}

.definition-row {
  display: grid;
  gap: 10px;
  padding: 12px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-muted);
}

.definition-tools {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.definition-row__actions {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 8px;
}

.import-dialog {
  display: grid;
  gap: 12px;
}

.expert-test {
  display: grid;
  gap: 18px;
}

.expert-test__form {
  display: grid;
  gap: 4px;
}

.expert-test__submit,
.expert-test__status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.expert-test__submit span,
.expert-test__status em {
  color: var(--admin-text-tertiary);
  font-size: 12px;
  font-style: normal;
}

.expert-test__result {
  display: grid;
  gap: 14px;
  padding-top: 16px;
  border-top: 1px solid var(--admin-border);
}

.expert-test__intake,
.expert-test__workflow,
.expert-test__quality {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-muted);
}

.expert-test__intake > div:first-child,
.expert-test__section-title,
.expert-test__section-title > span {
  display: grid;
  gap: 4px;
}

.expert-test__intake p,
.expert-test__intake-actions span,
.expert-test__section-title em,
.expert-test__steps em,
.expert-test__issues em {
  margin: 0;
  color: var(--admin-text-tertiary);
  font-size: 12px;
  font-style: normal;
  line-height: 1.6;
}

.expert-test__intake-actions,
.expert-test__section-title,
.expert-test__steps article,
.expert-test__issues article {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.expert-test__steps,
.expert-test__issues {
  display: grid;
  gap: 8px;
}

.expert-test__steps article,
.expert-test__issues article {
  padding: 10px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
}

.expert-test__steps article > span:nth-child(2),
.expert-test__issues article > span {
  display: grid;
  flex: 1;
  min-width: 0;
  gap: 3px;
}

.expert-test__step-index {
  display: inline-flex;
  width: 28px;
  height: 28px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
  font-size: 12px;
  font-weight: 650;
}

.expert-test__quality-score {
  margin-left: auto;
  color: var(--admin-text);
  font-size: 26px;
  font-weight: 680;
  line-height: 1;
}

.expert-test__dimensions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.expert-test__dimensions span {
  padding: 4px 8px;
  border-radius: 6px;
  color: var(--admin-text-secondary);
  background: var(--admin-surface);
  font-size: 12px;
}

.expert-test__issues article {
  align-items: flex-start;
  justify-content: flex-start;
}

.expert-test__issues p {
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.expert-test__status span {
  display: grid;
  gap: 3px;
}

.expert-test__card,
.expert-test__report {
  display: grid;
  gap: 12px;
  padding: 14px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-muted);
}

.expert-test__card > strong,
.expert-test__report strong {
  color: var(--admin-text);
  font-size: 15px;
}

.expert-test__card dl {
  display: grid;
  gap: 10px;
  margin: 0;
}

.expert-test__card dl div {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  gap: 12px;
}

.expert-test__card dt {
  color: var(--admin-text-tertiary);
  font-size: 12px;
}

.expert-test__card dd {
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.expert-test__report > div {
  display: grid;
  gap: 6px;
}

.expert-test__report p,
.expert-test__report li {
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.expert-test__report p {
  margin: 0;
  white-space: pre-wrap;
}

.expert-test__report section {
  padding-top: 10px;
  border-top: 1px solid var(--admin-border);
}

.expert-test__report h4 {
  margin: 0 0 6px;
  color: var(--admin-text);
  font-size: 13px;
}

.expert-test__report ul {
  display: grid;
  gap: 5px;
  margin: 0;
  padding-left: 20px;
}

.archive-upload {
  position: relative;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 14px;
  min-height: 128px;
  padding: 18px;
  border: 1px dashed var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-muted);
  color: var(--admin-text-secondary);
}

.archive-upload--selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.archive-upload input {
  position: absolute;
  inset: 0;
  width: 1px;
  height: 1px;
  opacity: 0;
  pointer-events: none;
}

.archive-upload__icon {
  display: inline-flex;
  width: 44px;
  height: 44px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--td-brand-color);
  background: var(--admin-surface);
}

.archive-upload span:nth-child(3) {
  display: grid;
  flex: 1;
  min-width: 0;
  gap: 5px;
}

.archive-upload strong {
  overflow: hidden;
  color: var(--admin-text);
  font-size: 15px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.archive-upload em,
.archive-upload__progress em,
.archive-upload__meta {
  color: var(--admin-text-tertiary);
  font-size: 13px;
  font-style: normal;
}

.archive-upload__progress {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 48px;
  gap: 10px;
  align-items: center;
}

.archive-upload__progress > div {
  height: 8px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--admin-surface-muted);
}

.archive-upload__progress span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--td-brand-color);
  transition: width 0.16s ease;
}

.archive-upload__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.archive-upload__meta span {
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--admin-surface-muted);
}

@media (max-width: 1080px) {
  .expert-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .expert-admin__grid {
    grid-template-columns: 1fr;
  }

  .expert-panel--detail {
    position: static;
  }
}

@media (max-width: 720px) {
  .expert-admin__header,
  .expert-panel__title,
  .package-card__head,
  .definition-row__head,
  .archive-upload {
    flex-direction: column;
    align-items: stretch;
  }

  .expert-summary {
    grid-template-columns: 1fr;
  }

  .expert-test__intake-actions,
  .expert-test__section-title,
  .expert-test__steps article,
  .expert-test__issues article {
    align-items: stretch;
  }

  .expert-test__intake-actions,
  .expert-test__section-title {
    flex-direction: column;
  }

  .expert-test__quality-score {
    margin-left: 0;
  }
}
</style>
