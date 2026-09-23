<template>
  <div class="organize-product-page">
    <main v-if="config" class="organize-detail-scroll">
      <header class="organize-detail-head">
        <button type="button" class="organize-back-button" @click="backToHub">
          <t-icon name="chevron-left" />
          返回工作台
        </button>
        <div class="organize-detail-identity">
          <span class="organize-detail-icon">
            <t-icon :name="template?.icon || 'dashboard'" />
          </span>
          <div>
            <h2>{{ config.name }}</h2>
            <span>{{ template?.name || config.templateKey }} · {{ scheduleLabel }}</span>
          </div>
        </div>
        <div class="organize-detail-actions">
          <t-button theme="primary" :loading="running" @click="runNow">
            <template #icon><t-icon name="play-circle" /></template>
            立即整理
          </t-button>
        </div>
      </header>

      <section class="organize-detail-section">
        <div v-if="config.jobs.length" class="organize-timeline">
          <article
            v-for="job in config.jobs"
            :key="job.id"
            class="organize-timeline-item"
            :class="[`is-${job.state}`, { 'is-highlighted': job.id === route.query.job }]"
          >
            <div class="organize-timeline-when">
              <strong>{{ job.dateLabel }}</strong>
              <span>{{ job.timeLabel }}</span>
            </div>
            <div class="organize-timeline-node">
              <span />
            </div>
            <div class="organize-job-card">
              <div class="organize-job-status-row">
                <span class="organize-tag" :class="statusClass(job)">
                  {{ statusLabel(job) }}
                </span>
                <span v-if="job.id === route.query.job" class="organize-job-focus">深链定位 · {{ job.id }}</span>
              </div>
              <p>{{ job.summary }}</p>
              <div class="organize-job-meta">
                <span>{{ job.rangeLabel }}</span>
                <i />
                <span>模板版本 {{ job.templateVersion }}</span>
                <i v-if="job.conclusionCount != null" />
                <span v-if="job.conclusionCount != null">
                  {{ job.conclusionCount }} 结论 · {{ job.todoCount }} 待办
                </span>
                <button
                  v-if="job.outputId"
                  type="button"
                  class="organize-job-output-link"
                  @click="openOutput(job.outputId)"
                >
                  查看产出
                  <t-icon name="chevron-right" />
                </button>
                <button
                  v-if="isRetryable(job)"
                  type="button"
                  class="organize-job-output-link"
                  @click="retry(job.id)"
                >
                  重新执行
                </button>
                <button
                  v-if="isActive(job)"
                  type="button"
                  class="organize-job-output-link"
                  @click="cancel(job.id)"
                >
                  取消
                </button>
              </div>
              <div v-if="isActive(job)" class="organize-job-progress">
                <span :style="{ width: `${job.progress || 0}%` }" />
              </div>
            </div>
          </article>
        </div>

        <div v-else class="organize-detail-empty">
          <t-icon name="time" />
          <strong>还没有整理任务</strong>
          <span>该整理还没有生成任务。</span>
        </div>
      </section>
    </main>

    <main v-else-if="loading" class="organize-detail-scroll">
      <div class="organize-detail-empty">
        <t-loading size="small" />
        <strong>正在加载整理任务</strong>
      </div>
    </main>

    <main v-else class="organize-detail-scroll">
      <div class="organize-detail-empty">
        <t-icon name="error-circle" />
        <strong>整理不存在</strong>
        <t-button variant="outline" size="small" @click="backToHub">返回工作台</t-button>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  cancelOrganizeJob,
  getOrganizeConfig,
  listOrganizeConfigJobs,
  retryOrganizeJob,
  runOrganizeConfig,
} from '@/api/organize'
import {
  organizeScheduleLabels,
  toOrganizeConfig,
  toOrganizeJob,
  type OrganizeConfig,
  type OrganizeJob,
} from './organizeWorkbenchState'

const route = useRoute()
const router = useRouter()
const config = ref<OrganizeConfig | null>(null)
const loading = ref(true)
const running = ref(false)
let refreshTimer: number | undefined

const template = computed(() => config.value?.template)
const scheduleLabel = computed(() => (config.value ? organizeScheduleLabels[config.value.schedule] : ''))

const isActive = (job: OrganizeJob) =>
  ['queued', 'running', 'repairing'].includes(job.state)

const isRetryable = (job: OrganizeJob) =>
  ['failed', 'fallback'].includes(job.state)

const statusLabel = (job: OrganizeJob) => {
  if (isActive(job)) return job.stage || '进行中'
  if (job.state === 'failed') return '执行失败'
  if (job.state === 'canceled') return '已取消'
  if (job.state === 'fallback') return '基础结果'
  return job.fresh ? '刚生成' : '已完成'
}

const statusClass = (job: OrganizeJob) => {
  if (isActive(job)) return 'organize-tag--progress'
  if (job.state === 'failed' || job.state === 'canceled') return 'organize-tag--muted'
  return 'organize-tag--success'
}

const loadConfig = async (quiet = false) => {
  if (!quiet) loading.value = true
  const configId = String(route.params.configId || '')
  try {
    const [configResponse, jobResponse] = await Promise.all([
      getOrganizeConfig(configId),
      listOrganizeConfigJobs(configId, { page: 1, page_size: 100 }),
    ])
    const mapped = toOrganizeConfig(configResponse.data)
    mapped.jobs = (jobResponse.data?.items || []).map(toOrganizeJob)
    config.value = mapped
  } catch (error: any) {
    config.value = null
    if (!quiet) MessagePlugin.error(error?.message || '整理任务加载失败')
  } finally {
    loading.value = false
  }
}

const runNow = async () => {
  if (!config.value || running.value) return
  running.value = true
  try {
    const response = await runOrganizeConfig(config.value.id)
    MessagePlugin.success('整理任务已创建')
    await loadConfig(true)
    await router.replace({
      path: route.path,
      query: { ...route.query, job: response.data.id },
    })
  } catch (error: any) {
    MessagePlugin.error(error?.message || '整理任务创建失败')
  } finally {
    running.value = false
  }
}

const retry = async (jobId: string) => {
  try {
    const response = await retryOrganizeJob(jobId)
    MessagePlugin.success('已重新创建整理任务')
    await loadConfig(true)
    await router.replace({ path: route.path, query: { job: response.data.id } })
  } catch (error: any) {
    MessagePlugin.error(error?.message || '任务重试失败')
  }
}

const cancel = async (jobId: string) => {
  try {
    await cancelOrganizeJob(jobId)
    MessagePlugin.success('任务已取消')
    await loadConfig(true)
  } catch (error: any) {
    MessagePlugin.error(error?.message || '任务取消失败')
  }
}

const backToHub = async () => {
  await router.push('/platform/organize/hub')
}

const openOutput = async (outputId: string) => {
  if (!config.value) return
  await router.push({
    path: `/platform/organize/outputs/${encodeURIComponent(outputId)}`,
    query: { from: 'config', configId: config.value.id },
  })
}

onMounted(() => {
  void loadConfig()
  refreshTimer = window.setInterval(() => {
    if (config.value?.jobs.some(isActive)) void loadConfig(true)
  }, 3000)
})

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer)
})

watch(
  () => route.params.configId,
  () => void loadConfig(),
)
</script>

<style scoped lang="less">
.organize-product-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
}

.organize-detail-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 28px 42px 48px;
}

.organize-detail-head,
.organize-detail-section {
  max-width: 1040px;
  margin-right: auto;
  margin-left: auto;
}

.organize-detail-head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 18px;
  padding-bottom: 26px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.organize-detail-actions {
  display: flex;
  justify-content: flex-end;
}

.organize-back-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
  white-space: nowrap;
}

.organize-back-button:hover {
  color: var(--td-text-color-primary);
}

.organize-detail-identity {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

.organize-detail-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: 10px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 19px;
}

.organize-detail-identity h2 {
  margin: 0;
  overflow: hidden;
  font-size: 20px;
  line-height: 28px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-detail-identity span {
  display: block;
  margin-top: 3px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.organize-detail-section {
  padding-top: 22px;
}

.organize-timeline {
  display: flex;
  flex-direction: column;
}

.organize-timeline-item {
  display: grid;
  grid-template-columns: 100px 24px minmax(0, 1fr);
  min-height: 138px;
}

.organize-timeline-when {
  display: flex;
  padding-top: 16px;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  text-align: right;
}

.organize-timeline-when strong {
  color: var(--td-text-color-primary);
  font-weight: 600;
}

.organize-timeline-node {
  position: relative;
  display: flex;
  justify-content: center;
}

.organize-timeline-node::after {
  position: absolute;
  top: 25px;
  bottom: -10px;
  width: 1px;
  background: var(--td-component-stroke);
  content: '';
}

.organize-timeline-item:last-child .organize-timeline-node::after {
  display: none;
}

.organize-timeline-node span {
  position: relative;
  z-index: 1;
  width: 9px;
  height: 9px;
  margin-top: 20px;
  border: 2px solid var(--td-brand-color);
  border-radius: 50%;
  background: var(--td-bg-color-container);
}

.organize-timeline-item.is-running .organize-timeline-node span,
.organize-timeline-item.is-queued .organize-timeline-node span,
.organize-timeline-item.is-repairing .organize-timeline-node span {
  border-color: #d69a35;
  box-shadow: 0 0 0 4px #fff3dd;
}

.organize-timeline-item.is-failed .organize-timeline-node span,
.organize-timeline-item.is-canceled .organize-timeline-node span {
  border-color: var(--td-text-color-placeholder);
}

.organize-job-card {
  position: relative;
  margin: 0 0 14px 12px;
  padding: 15px 16px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 9px;
  background: var(--td-bg-color-container);
}

.organize-timeline-item.is-highlighted .organize-job-card {
  border-color: var(--td-brand-color);
  box-shadow: 0 0 0 3px var(--td-brand-color-light);
}

.organize-job-status-row,
.organize-job-meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
}

.organize-job-status-row {
  justify-content: space-between;
}

.organize-job-focus {
  color: var(--td-brand-color);
  font-size: 11px;
}

.organize-job-card > p {
  margin: 12px 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.organize-job-meta {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.organize-job-meta i {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: var(--td-text-color-placeholder);
}

.organize-job-output-link {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-left: auto;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.organize-job-progress {
  height: 3px;
  margin-top: 12px;
  overflow: hidden;
  border-radius: 2px;
  background: var(--td-bg-color-secondarycontainer);
}

.organize-job-progress span {
  display: block;
  height: 100%;
  background: var(--td-brand-color);
}

.organize-tag {
  display: inline-flex;
  align-items: center;
  min-height: 22px;
  padding: 0 7px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.organize-tag--success {
  border-color: #b7e1cf;
  background: #eef9f3;
  color: #23805a;
}

.organize-tag--progress {
  border-color: #f2d6a8;
  background: #fff7e8;
  color: #a46715;
}

.organize-tag--muted {
  color: var(--td-text-color-placeholder);
}

.organize-detail-empty {
  display: flex;
  min-height: 200px;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-secondary);
}

.organize-detail-empty :deep(.t-icon) {
  color: var(--td-text-color-placeholder);
  font-size: 26px;
}

.organize-detail-empty strong {
  color: var(--td-text-color-primary);
  font-size: 14px;
}

@media (max-width: 760px) {
  .organize-detail-scroll {
    padding: 24px 20px 36px;
  }

  .organize-detail-head {
    grid-template-columns: 1fr;
    align-items: flex-start;
    gap: 14px;
  }

  .organize-timeline-item {
    grid-template-columns: 72px 18px minmax(0, 1fr);
  }

  .organize-job-card {
    margin-left: 7px;
  }
}
</style>
