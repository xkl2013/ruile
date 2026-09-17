<template>
  <section class="expert-admin">
    <div class="expert-admin__header">
      <div>
        <h2>专家维护</h2>
        <p>后台维护专家包、版本发布和服务分身绑定。</p>
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
        <em>{{ publishedDefinitionCount }} 个可绑定</em>
      </article>
      <article>
        <span>服务绑定</span>
        <strong>{{ bindings.length }}</strong>
        <em>{{ enabledBindingCount }} 个启用</em>
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
              <span class="package-card__icon">
                <t-icon name="usergroup" />
              </span>
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
            <span>
              <strong>{{ selectedPackage.display_name }}</strong>
              <em>{{ selectedPackage.package_key }}</em>
            </span>
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
              <em>只有已发布版本可以绑定到服务分身。</em>
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
                <p>{{ entry.definition.description || '未填写描述' }}</p>
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

          <section class="binding-section">
            <div class="definition-section__title">
              <strong>绑定分身</strong>
              <em>绑定后服务队列可按分身匹配专家能力。</em>
            </div>

            <t-alert
              v-if="profileLoadFailed"
              theme="warning"
              message="员工分身读取失败，请确认当前账号有管理权限。"
              class="binding-alert"
            />

            <t-form :data="bindingForm" label-align="top" class="binding-form" @submit.prevent>
              <t-form-item label="员工分身">
                <t-select
                  v-model="bindingForm.profileId"
                  :options="profileOptions"
                  :loading="profileLoading"
                  :disabled="savingBinding || profileOptions.length === 0"
                  placeholder="选择员工分身"
                />
              </t-form-item>
              <t-form-item label="专家版本">
                <t-select
                  v-model="bindingForm.definitionId"
                  :options="definitionOptions"
                  :disabled="savingBinding || definitionOptions.length === 0"
                  placeholder="选择已发布专家"
                />
              </t-form-item>
              <div class="binding-form__row">
                <t-form-item label="服务领域">
                  <t-input
                    v-model="bindingForm.domain"
                    :disabled="savingBinding"
                    :maxlength="80"
                    clearable
                    placeholder="agent_domain"
                  />
                </t-form-item>
                <t-form-item label="启用">
                  <t-switch v-model="bindingForm.enabled" :disabled="savingBinding" />
                </t-form-item>
              </div>
              <t-button
                theme="primary"
                :loading="savingBinding"
                :disabled="!canSaveBinding"
                @click="saveBinding"
              >
                保存绑定
              </t-button>
            </t-form>

            <div class="binding-list">
              <div class="binding-list__title">当前包绑定</div>
              <div v-if="bindingsForSelected.length === 0" class="expert-empty expert-empty--compact">
                <span>暂无绑定</span>
              </div>
              <div v-else class="binding-row" v-for="binding in bindingsForSelected" :key="binding.id">
                <span>
                  <strong>{{ definitionName(binding.agent_definition_version_id) }}</strong>
                  <em>{{ profileName(binding.profile_id) }} · {{ binding.agent_domain }}</em>
                </span>
                <t-tag :theme="binding.enabled ? 'success' : 'default'" variant="light">
                  {{ binding.enabled ? '启用' : '停用' }}
                </t-tag>
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
      :close-on-overlay-click="!testingExpert"
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
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { listModels, type ModelConfig } from '@/api/model'
import {
  bindExpertPackageAgent,
  importExpertPackageArchive,
  listExpertPackageBindings,
  listExpertPackages,
  publishExpertPackageVersion,
  testExpertPackageAgent,
  type AgentBinding,
  type AgentDefinitionVersion,
  type ExpertPackage,
  type ExpertPackageVersion,
} from '@/api/expert-package'
import {
  getServiceAgentRun,
  listServiceWorkProfiles,
  type ServiceAgentRun,
  type ServiceWorkProfile,
  type StructuredReportV1,
} from '@/api/service'

interface DefinitionEntry {
  package: ExpertPackage
  version: ExpertPackageVersion
  definition: AgentDefinitionVersion
}

interface BindingForm {
  profileId: string
  definitionId: string
  domain: string
  enabled: boolean
}

interface ExpertTestForm {
  modelId: string
  prompt: string
}

const packages = ref<ExpertPackage[]>([])
const bindings = ref<AgentBinding[]>([])
const workProfiles = ref<ServiceWorkProfile[]>([])
const chatModels = ref<ModelConfig[]>([])
const selectedPackageId = ref('')
const loading = ref(false)
const errorMessage = ref('')
const profileLoading = ref(false)
const profileLoadFailed = ref(false)
const importing = ref(false)
const importDialogVisible = ref(false)
const archiveInputRef = ref<HTMLInputElement | null>(null)
const selectedImportFile = ref<File | null>(null)
const uploadProgress = ref(0)
const publishingVersionId = ref('')
const savingBinding = ref(false)
const modelLoading = ref(false)
const testDialogVisible = ref(false)
const testingExpert = ref(false)
const testDefinitionEntry = ref<DefinitionEntry | null>(null)
const testRun = ref<ServiceAgentRun | null>(null)
const testError = ref('')
const bindingForm = ref<BindingForm>({
  profileId: '',
  definitionId: '',
  domain: '',
  enabled: true,
})
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

const bindableDefinitions = computed<DefinitionEntry[]>(() => (
  selectedDefinitions.value.filter((entry) => entry.version.state === 'published')
))

const definitionsById = computed(() => {
  const map = new Map<string, DefinitionEntry>()
  allDefinitionEntries.value.forEach((entry) => map.set(entry.definition.id, entry))
  return map
})

const workProfilesById = computed(() => {
  const map = new Map<string, ServiceWorkProfile>()
  workProfiles.value.forEach((profile) => map.set(profile.id, profile))
  return map
})

const selectedDefinitionIdSet = computed(() => new Set(
  selectedDefinitions.value.map((entry) => entry.definition.id),
))

const bindingsForSelected = computed(() => (
  bindings.value.filter((binding) => selectedDefinitionIdSet.value.has(binding.agent_definition_version_id))
))

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

const enabledBindingCount = computed(() => bindings.value.filter((binding) => binding.enabled).length)

const profileOptions = computed(() => (
  workProfiles.value.map((profile) => ({
    label: [
      profile.name || '未命名分身',
      profile.default_profile ? '默认' : '',
      profile.enabled === false ? '未启用' : '',
    ].filter(Boolean).join(' · '),
    value: profile.id,
  }))
))

const definitionOptions = computed(() => (
  bindableDefinitions.value.map((entry) => ({
    label: `${entry.definition.display_name} · v${entry.version.version}`,
    value: entry.definition.id,
  }))
))

const canSaveBinding = computed(() => (
  Boolean(selectedPackage.value && bindingForm.value.profileId && bindingForm.value.definitionId && bindingForm.value.domain.trim())
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
    loadPackagesAndBindings(),
    loadWorkProfiles(),
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

async function loadPackagesAndBindings() {
  try {
    const [packageResponse, bindingResponse] = await Promise.all([
      listExpertPackages(),
      listExpertPackageBindings(),
    ])
    packages.value = packageResponse?.data || []
    bindings.value = bindingResponse?.data || []
    if (!selectedPackageId.value || !packages.value.some((pkg) => pkg.id === selectedPackageId.value)) {
      selectedPackageId.value = packages.value[0]?.id || ''
    }
    normalizeBindingForm()
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to load expert packages:', error)
    errorMessage.value = error?.message || '专家包读取失败'
    packages.value = []
    bindings.value = []
  }
}

async function loadWorkProfiles() {
  if (profileLoading.value) return
  profileLoading.value = true
  profileLoadFailed.value = false
  try {
    const response = await listServiceWorkProfiles()
    workProfiles.value = response?.data || []
    normalizeBindingForm()
  } catch (error) {
    console.warn('[AdminExpertPackages] Failed to load work profiles:', error)
    workProfiles.value = []
    profileLoadFailed.value = true
  } finally {
    profileLoading.value = false
  }
}

function selectPackage(id: string) {
  selectedPackageId.value = id
  normalizeBindingForm(true)
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
    await loadPackagesAndBindings()
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
    await loadPackagesAndBindings()
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to publish expert package:', error)
    MessagePlugin.error(error?.message || '专家版本发布失败')
  } finally {
    publishingVersionId.value = ''
  }
}

async function saveBinding() {
  const pkg = selectedPackage.value
  if (!pkg || !canSaveBinding.value) return

  savingBinding.value = true
  try {
    const response = await bindExpertPackageAgent(pkg.id, {
      profile_id: bindingForm.value.profileId,
      agent_definition_version_id: bindingForm.value.definitionId,
      agent_domain: bindingForm.value.domain.trim(),
      enabled: bindingForm.value.enabled,
    })
    const saved = response?.data
    if (saved) {
      bindings.value = bindings.value
        .filter((binding) => binding.id !== saved.id)
        .concat(saved)
    }
    MessagePlugin.success('专家绑定已保存')
    await loadPackagesAndBindings()
  } catch (error: any) {
    console.warn('[AdminExpertPackages] Failed to bind expert package agent:', error)
    MessagePlugin.error(error?.message || '专家绑定保存失败')
  } finally {
    savingBinding.value = false
  }
}

function openTestDialog(entry: DefinitionEntry) {
  testDefinitionEntry.value = entry
  testRun.value = null
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
    const response = await getServiceAgentRun(id)
    if (!response?.data) throw new Error(response?.message || '任务状态读取失败')
    testRun.value = response.data
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

function normalizeBindingForm(forceDefinition = false) {
  if (!bindingForm.value.profileId || !workProfiles.value.some((profile) => profile.id === bindingForm.value.profileId)) {
    bindingForm.value.profileId = preferredProfileId()
  }

  const hasDefinition = bindableDefinitions.value.some((entry) => entry.definition.id === bindingForm.value.definitionId)
  if (forceDefinition || !hasDefinition) {
    const first = bindableDefinitions.value[0]
    bindingForm.value.definitionId = first?.definition.id || ''
    bindingForm.value.domain = first?.definition.domain || ''
    return
  }

  syncBindingDomain()
}

function syncBindingDomain() {
  const current = bindableDefinitions.value.find((entry) => entry.definition.id === bindingForm.value.definitionId)
  if (current && !bindingForm.value.domain.trim()) {
    bindingForm.value.domain = current.definition.domain || ''
  }
}

function preferredProfileId() {
  return (
    workProfiles.value.find((profile) => profile.default_profile && profile.enabled !== false)?.id ||
    workProfiles.value.find((profile) => profile.enabled !== false)?.id ||
    workProfiles.value[0]?.id ||
    ''
  )
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

function definitionName(id: string) {
  const entry = definitionsById.value.get(id)
  if (!entry) return compactId(id)
  return `${entry.definition.display_name} · v${entry.version.version}`
}

function profileName(id: string) {
  const profile = workProfilesById.value.get(id)
  return profile?.name || compactId(id)
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

watch(() => bindingForm.value.definitionId, () => {
  bindingForm.value.domain = ''
  syncBindingDomain()
})

watch(bindableDefinitions, () => {
  normalizeBindingForm()
})

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
.definition-row__head em,
.binding-row em {
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
.definition-row__head,
.binding-row {
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
.binding-row span,
.definition-row__head span {
  display: grid;
  gap: 4px;
}

.expert-panel__title strong,
.definition-section__title strong {
  color: var(--admin-text);
  font-size: 16px;
}

.package-list,
.definition-list,
.binding-list {
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
  display: inline-flex;
  width: 36px;
  height: 36px;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--td-brand-color);
  background: var(--td-brand-color-light);
}

.package-card__title {
  display: grid;
  flex: 1;
  min-width: 0;
  gap: 3px;
}

.package-card__title strong,
.definition-row__head strong,
.binding-row strong {
  overflow: hidden;
  color: var(--admin-text);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.package-card__title em,
.definition-row__head em,
.binding-row em {
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

.definition-section,
.binding-section {
  display: grid;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid var(--admin-border);
}

.binding-section {
  margin-top: 18px;
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

.binding-form {
  display: grid;
  gap: 4px;
}

.binding-form__row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 96px;
  gap: 12px;
  align-items: start;
}

.binding-alert {
  border-radius: 8px;
}

.binding-list {
  padding-top: 12px;
}

.binding-list__title {
  color: var(--admin-text-tertiary);
  font-size: 12px;
}

.binding-row {
  padding: 10px 0;
  border-bottom: 1px solid var(--admin-border);
}

.binding-row:last-child {
  border-bottom: 0;
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
  .binding-row,
  .archive-upload {
    flex-direction: column;
    align-items: stretch;
  }

  .expert-summary {
    grid-template-columns: 1fr;
  }

  .binding-form__row {
    grid-template-columns: 1fr;
  }
}
</style>
