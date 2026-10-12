<template>
  <section class="runtime-page">
    <div class="runtime-page__header">
      <div>
        <div class="runtime-page__eyebrow">Agent · 运行时治理</div>
        <h2>Agent 运行时</h2>
        <p>维护当前工作区的 Sandbox 策略和 Skill 运行版本。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadAll">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-alert v-if="errorMessage" theme="error" :message="errorMessage" />
    <t-alert v-if="successMessage" theme="success" :message="successMessage" />

    <div class="runtime-grid">
      <section class="runtime-panel">
        <div class="runtime-panel__header">
          <div>
            <h3>Sandbox 策略</h3>
            <span>激活策略会影响后续 Agent Skill 执行。</span>
          </div>
          <t-button theme="primary" size="small" :loading="saving" @click="createConfig">
            <template #icon><t-icon name="add" /></template>
            新建
          </t-button>
        </div>

        <div class="runtime-form">
          <t-input v-model="draft.name" label="名称" placeholder="例如：Docker 默认运行时" />
          <t-select v-model="draft.mode" label="模式" :options="modeOptions" />
          <t-input
            v-if="draft.mode === 'docker'"
            v-model="draft.docker_image"
            label="Docker 镜像"
            placeholder="wechatopenai/weknora-sandbox:latest"
          />
          <t-input-number v-model="draft.timeout_seconds" label="超时（秒）" :min="1" :max="3600" />
          <div class="runtime-form__switches">
            <t-switch v-model="draft.enabled" label="启用" />
            <t-switch v-model="draft.fallback_enabled" label="允许降级" />
          </div>
        </div>

        <div v-if="loading && !configs.length" class="runtime-state">
          <t-loading size="small" /> 加载中
        </div>
        <div v-else-if="!configs.length" class="runtime-state">
          <t-icon name="inbox" /> 暂无 Sandbox 策略
        </div>
        <div v-else class="runtime-list">
          <article v-for="config in configs" :key="config.id" class="runtime-row">
            <div class="runtime-row__main">
              <div class="runtime-row__title">
                <strong>{{ config.name }}</strong>
                <t-tag v-if="config.active" theme="success" variant="light" size="small">当前激活</t-tag>
                <t-tag v-else theme="default" variant="light" size="small">{{ config.enabled ? '可用' : '已停用' }}</t-tag>
              </div>
              <div class="runtime-row__meta">
                <span>{{ config.mode }}</span>
                <span>{{ config.timeout_seconds }} 秒</span>
                <span v-if="config.mode === 'docker'">{{ config.docker_image }}</span>
              </div>
            </div>
            <div class="runtime-row__actions">
              <t-button variant="text" size="small" :loading="busyId === config.id" @click="checkConfig(config)">
                <template #icon><t-icon name="check-circle" /></template>
                检查
              </t-button>
              <t-button v-if="!config.active" variant="text" size="small" :loading="busyId === config.id" @click="activateConfig(config)">
                <template #icon><t-icon name="play-circle" /></template>
                激活
              </t-button>
              <t-button v-else variant="text" size="small" :loading="busyId === config.id" @click="deactivateConfig(config)">
                <template #icon><t-icon name="pause-circle" /></template>
                停用
              </t-button>
              <t-button v-if="!config.active" theme="danger" variant="text" size="small" :loading="busyId === config.id" @click="removeConfig(config)">
                <template #icon><t-icon name="delete" /></template>
                删除
              </t-button>
            </div>
          </article>
        </div>
      </section>

      <section class="runtime-panel">
        <div class="runtime-panel__header">
          <div>
            <h3>Skill 快照</h3>
            <span>快照固定当前 Skill bundle，支持恢复到历史版本。</span>
          </div>
          <t-button theme="primary" size="small" :loading="saving" @click="createSnapshot">
            <template #icon><t-icon name="add" /></template>
            创建快照
          </t-button>
        </div>

        <div v-if="loading && !snapshots.length" class="runtime-state">
          <t-loading size="small" /> 加载中
        </div>
        <div v-else-if="!snapshots.length" class="runtime-state">
          <t-icon name="history" /> 暂无 Skill 快照
        </div>
        <div v-else class="runtime-list">
          <article v-for="snapshot in snapshots" :key="snapshot.id" class="runtime-row">
            <div class="runtime-row__main">
              <div class="runtime-row__title">
                <strong>Generation {{ snapshot.generation }}</strong>
                <t-tag :theme="snapshot.state === 'active' ? 'success' : 'default'" variant="light" size="small">
                  {{ snapshot.state }}
                </t-tag>
              </div>
              <div class="runtime-row__meta">
                <span>{{ snapshot.id }}</span>
                <code>{{ snapshot.manifest_sha256.slice(0, 12) }}</code>
              </div>
            </div>
            <div class="runtime-row__actions">
              <t-button variant="text" size="small" :loading="busyId === snapshot.id" @click="verifySnapshot(snapshot)">
                <template #icon><t-icon name="check-circle" /></template>
                校验
              </t-button>
              <t-button v-if="snapshot.state !== 'active'" variant="text" size="small" :loading="busyId === snapshot.id" @click="activateSnapshot(snapshot)">
                <template #icon><t-icon name="play-circle" /></template>
                激活
              </t-button>
              <t-button v-if="snapshot.state !== 'active'" theme="warning" variant="text" size="small" :loading="busyId === snapshot.id" @click="rollbackSnapshot(snapshot)">
                <template #icon><t-icon name="rollback" /></template>
                回滚
              </t-button>
            </div>
          </article>
        </div>
      </section>

      <section class="runtime-panel runtime-panel--wide">
        <div class="runtime-panel__header">
          <div>
            <h3>成员 Agent 运行记录</h3>
            <span>查看成员 Agent 的独立执行状态、Token 用量和费用归因。</span>
          </div>
          <t-tag theme="default" variant="light" size="small">最近 50 条</t-tag>
        </div>

        <div v-if="loading && !delegationRuns.length" class="runtime-state">
          <t-loading size="small" /> 加载中
        </div>
        <div v-else-if="!delegationRuns.length" class="runtime-state">
          <t-icon name="history" /> 暂无成员 Agent 运行记录
        </div>
        <div v-else class="runtime-list">
          <article v-for="run in delegationRuns" :key="run.id" class="runtime-row">
            <div class="runtime-row__main">
              <div class="runtime-row__title">
                <strong>{{ run.member_agent_id }}</strong>
                <t-tag :theme="delegationRunTheme(run.status)" variant="light" size="small">
                  {{ delegationRunStatus(run.status) }}
                </t-tag>
              </div>
              <div class="runtime-row__meta">
                <span>{{ run.task || '未提供任务描述' }}</span>
                <span>{{ formatTokenCount(run.input_tokens + run.output_tokens) }} Token</span>
                <span>{{ formatPoints(run.billed_point_micros) }} 积分</span>
                <span>{{ formatDuration(run.duration_millis) }}</span>
                <span>{{ run.usage_ledger_count }} 次账单</span>
              </div>
              <div v-if="run.error_message" class="runtime-row__error">{{ run.error_message }}</div>
            </div>
            <div class="runtime-row__cost">
              <strong>{{ formatUSD(run.rated_cost_nanousd) }}</strong>
              <span>计费成本</span>
            </div>
          </article>
        </div>
      </section>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { DialogPlugin } from 'tdesign-vue-next'
import {
  activateSandboxConfig,
  activateSkillSnapshot,
  checkSandboxConfig,
  createSandboxConfig,
  createSkillSnapshot,
  deactivateSandboxConfig,
  deleteSandboxConfig,
  listSandboxConfigs,
  listDelegationRuns,
  listSkillSnapshots,
  rollbackSkillSnapshot,
  type SandboxCheck,
  type SandboxConfig,
  type DelegationRun,
  type SkillSnapshot,
  verifySkillSnapshot,
} from '@admin/api/agent-runtime'

const loading = ref(false)
const saving = ref(false)
const busyId = ref('')
const errorMessage = ref('')
const successMessage = ref('')
const configs = ref<SandboxConfig[]>([])
const snapshots = ref<SkillSnapshot[]>([])
const delegationRuns = ref<DelegationRun[]>([])
const modeOptions = [
  { label: 'Docker', value: 'docker' },
  { label: 'Local', value: 'local' },
  { label: '禁用', value: 'disabled' },
]
const draft = reactive<Partial<SandboxConfig>>({
  name: '',
  mode: 'docker',
  enabled: true,
  fallback_enabled: true,
  timeout_seconds: 60,
  docker_image: 'wechatopenai/weknora-sandbox:latest',
})

function clearNotice() {
  errorMessage.value = ''
  successMessage.value = ''
}

async function loadAll(clearNotices = true) {
  loading.value = true
  if (clearNotices) {
    clearNotice()
  }
  try {
    const [configResponse, snapshotResponse, delegationRunResponse] = await Promise.all([
      listSandboxConfigs(),
      listSkillSnapshots(),
      listDelegationRuns({ limit: 50 }),
    ])
    configs.value = configResponse?.data || []
    snapshots.value = snapshotResponse?.data || []
    delegationRuns.value = delegationRunResponse?.data || []
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取运行时配置失败'
  } finally {
    loading.value = false
  }
}

async function createConfig() {
  if (!draft.name?.trim()) {
    errorMessage.value = '请输入 Sandbox 策略名称'
    return
  }
  saving.value = true
  clearNotice()
  try {
    await createSandboxConfig(draft)
    draft.name = ''
    await loadAll(false)
    successMessage.value = 'Sandbox 策略已创建'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '创建 Sandbox 策略失败'
  } finally {
    saving.value = false
  }
}

async function runConfigAction(config: SandboxConfig, action: () => Promise<unknown>) {
  busyId.value = config.id
  clearNotice()
  try {
    await action()
    await loadAll(false)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Sandbox 操作失败'
  } finally {
    busyId.value = ''
  }
}

function activateConfig(config: SandboxConfig) {
  return runConfigAction(config, () => activateSandboxConfig(config.id))
}

function deactivateConfig(config: SandboxConfig) {
  return runConfigAction(config, () => deactivateSandboxConfig(config.id))
}

function checkConfig(config: SandboxConfig) {
  return runConfigAction(config, async () => {
    const response = await checkSandboxConfig(config.id)
    const result = response?.data as SandboxCheck
    successMessage.value = result.available
      ? `Sandbox 可用（${result.effective_mode}）`
      : result.message || 'Sandbox 不可用'
  })
}

function removeConfig(config: SandboxConfig) {
  const dialog = DialogPlugin.confirm({
    header: '删除 Sandbox 策略',
    body: `确认删除「${config.name}」？`,
    onConfirm: async () => {
      dialog.hide()
      await runConfigAction(config, () => deleteSandboxConfig(config.id))
    },
  })
}

async function createSnapshot() {
  saving.value = true
  clearNotice()
  try {
    await createSkillSnapshot()
    await loadAll(false)
    successMessage.value = 'Skill 快照已创建并激活'
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '创建 Skill 快照失败'
  } finally {
    saving.value = false
  }
}

async function runSnapshotAction(snapshot: SkillSnapshot, action: () => Promise<unknown>) {
  busyId.value = snapshot.id
  clearNotice()
  try {
    await action()
    await loadAll()
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Skill 快照操作失败'
  } finally {
    busyId.value = ''
  }
}

function activateSnapshot(snapshot: SkillSnapshot) {
  return runSnapshotAction(snapshot, () => activateSkillSnapshot(snapshot.id))
}

function rollbackSnapshot(snapshot: SkillSnapshot) {
  return runSnapshotAction(snapshot, () => rollbackSkillSnapshot(snapshot.id))
}

function verifySnapshot(snapshot: SkillSnapshot) {
  return runSnapshotAction(snapshot, async () => {
    const response = await verifySkillSnapshot(snapshot.id)
    successMessage.value = response?.data?.valid ? '快照校验通过' : '快照校验失败'
  })
}

function formatTokenCount(value: number) {
  return Math.max(0, Number(value) || 0).toLocaleString()
}

function formatPoints(value: number) {
  return (Math.max(0, Number(value) || 0) / 1_000_000).toLocaleString('zh-CN', {
    maximumFractionDigits: 4,
  })
}

function formatUSD(value: number) {
  return `$${(Math.max(0, Number(value) || 0) / 1_000_000_000).toFixed(6)}`
}

function formatDuration(value: number) {
  const milliseconds = Math.max(0, Number(value) || 0)
  return milliseconds < 1000 ? `${milliseconds} ms` : `${(milliseconds / 1000).toFixed(2)} s`
}

function delegationRunStatus(value: string) {
  if (value === 'succeeded') return '成功'
  if (value === 'failed') return '失败'
  if (value === 'running') return '运行中'
  return value || '未知'
}

function delegationRunTheme(value: string) {
  if (value === 'succeeded') return 'success'
  if (value === 'failed') return 'danger'
  if (value === 'running') return 'warning'
  return 'default'
}

onMounted(loadAll)
</script>

<style scoped>
.runtime-page { display: flex; flex-direction: column; gap: 18px; }
.runtime-page__header { display: flex; justify-content: space-between; gap: 24px; align-items: flex-start; }
.runtime-page__eyebrow { color: var(--td-brand-color); font-size: 12px; margin-bottom: 6px; }
.runtime-page__header h2 { margin: 0; font-size: 24px; }
.runtime-page__header p { margin: 8px 0 0; color: var(--td-text-color-secondary); }
.runtime-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; align-items: start; }
.runtime-panel { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); padding: 18px; }
.runtime-panel--wide { grid-column: 1 / -1; }
.runtime-panel__header { display: flex; justify-content: space-between; gap: 16px; align-items: flex-start; margin-bottom: 16px; }
.runtime-panel__header h3 { margin: 0; font-size: 16px; }
.runtime-panel__header span { display: block; margin-top: 6px; color: var(--td-text-color-secondary); font-size: 12px; }
.runtime-form { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; padding-bottom: 16px; border-bottom: 1px solid var(--td-component-border); }
.runtime-form__switches { display: flex; align-items: end; gap: 18px; padding-bottom: 6px; }
.runtime-list { display: flex; flex-direction: column; }
.runtime-row { display: flex; justify-content: space-between; gap: 16px; padding: 16px 0; border-bottom: 1px solid var(--td-component-border); }
.runtime-row:last-child { border-bottom: 0; }
.runtime-row__main { min-width: 0; }
.runtime-row__title { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.runtime-row__meta { display: flex; gap: 10px; flex-wrap: wrap; color: var(--td-text-color-secondary); font-size: 12px; margin-top: 8px; }
.runtime-row__actions { display: flex; align-items: center; flex-wrap: wrap; justify-content: flex-end; }
.runtime-row__cost { display: flex; flex-direction: column; align-items: flex-end; justify-content: center; min-width: 110px; }
.runtime-row__cost span, .runtime-row__error { color: var(--td-text-color-secondary); font-size: 12px; }
.runtime-row__error { color: var(--td-error-color); margin-top: 8px; white-space: pre-wrap; }
.runtime-state { min-height: 120px; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }
@media (max-width: 1100px) { .runtime-grid { grid-template-columns: 1fr; } }
@media (max-width: 700px) { .runtime-page__header, .runtime-row { flex-direction: column; } .runtime-row__actions { justify-content: flex-start; } .runtime-row__cost { align-items: flex-start; } .runtime-form { grid-template-columns: 1fr; } }
</style>
