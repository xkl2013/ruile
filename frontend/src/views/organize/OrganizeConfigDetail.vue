<template>
  <div class="organize-product-page">
    <main v-if="config" class="organize-detail-scroll">
      <header class="organize-detail-head">
        <div class="organize-detail-identity">
          <h2>{{ config.name }}</h2>
        </div>
        <div class="organize-detail-actions">
          <t-button theme="primary" :loading="running" @click="runNow">
            <template #icon><t-icon name="play-circle" /></template>
            立即整理
          </t-button>
        </div>
      </header>

      <section class="organize-detail-section">
        <div v-if="config.jobs.length" class="organize-result-list organize-timeline">
          <article
            v-for="job in config.jobs"
            :key="job.id"
            class="organize-result-item organize-timeline-item"
            :class="[`is-${job.state}`, { 'is-highlighted': job.id === route.query.job }]"
          >
            <div class="organize-timeline-when">
              <strong>{{ job.dateLabel }}</strong>
              <span>{{ job.timeLabel }}</span>
            </div>
            <div class="organize-timeline-node" aria-hidden="true">
              <span />
            </div>
            <div class="organize-timeline-content">
              <button
                v-if="outputForJob(job)"
                type="button"
                class="organize-result-card"
                :aria-label="`查看整理结果 ${resultCardTitle(outputForJob(job)!)}`"
                @click="openOutput(outputForJob(job)!.id, job.id)"
              >
                <span class="organize-result-card__body">
                  <span class="organize-result-card__title-row">
                    <strong>{{ resultCardTitle(outputForJob(job)!) }}</strong>
                    <span v-if="job.state === 'fallback'" class="organize-tag organize-tag--muted">基础结果</span>
                  </span>
                  <span class="organize-result-card__preview">
                    {{ outputForJob(job)!.preview || '整理结果已生成，点击查看完整内容。' }}
                  </span>
                  <span class="organize-result-card__footer">
                    {{ outputCreatedLabel(outputForJob(job)!) }}
                  </span>
                </span>
                <span class="organize-result-card__arrow" aria-hidden="true">
                  <t-icon name="chevron-right" />
                </span>
              </button>

              <div v-else class="organize-job-state-card">
                <div class="organize-job-status-row">
                  <span class="organize-tag" :class="statusClass(job)">{{ statusLabel(job) }}</span>
                </div>
                <strong>{{ config.name }}</strong>
                <p>{{ job.summary }}</p>
                <div class="organize-job-actions">
                  <button
                    v-if="job.outputId"
                    type="button"
                    class="organize-job-output-link"
                    @click="openOutput(job.outputId, job.id)"
                  >
                    查看结果
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
  listOrganizeOutputs,
  retryOrganizeJob,
  runOrganizeConfig,
  streamOrganizeJobEvents,
} from '@/api/organize'
import {
  ORGANIZE_CONFIG_UNREAD_EVENT,
  toOrganizeConfig,
  toOrganizeJob,
  toOrganizeOutput,
  type OrganizeConfig,
  type OrganizeJob,
  type OrganizeOutput,
} from './organizeWorkbenchState'

const route = useRoute()
const router = useRouter()
const config = ref<OrganizeConfig | null>(null)
const outputsByJobId = ref<Record<string, OrganizeOutput>>({})
const loading = ref(true)
const running = ref(false)
let eventController: AbortController | null = null

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

const outputForJob = (job: OrganizeJob) => outputsByJobId.value[job.id]

const resultCardTitle = (output: OrganizeOutput) =>
  output.title.replace(/\s*·\s*\d{4}-\d{2}-\d{2}\s*$/, '').trim() || output.title

const outputCreatedLabel = (output: OrganizeOutput) => {
  const date = new Date(output.createdAt)
  if (Number.isNaN(date.getTime())) return '整理结果已创建'
  const hours = String(date.getHours()).padStart(2, '0')
  const minutes = String(date.getMinutes()).padStart(2, '0')
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${hours}:${minutes} 创建 ${month}月${day}日`
}

const loadConfig = async (quiet = false) => {
  if (!quiet) loading.value = true
  const configId = String(route.params.configId || '')
  try {
    const [configResponse, jobResponse, outputResponse] = await Promise.all([
      getOrganizeConfig(configId),
      listOrganizeConfigJobs(configId, { page: 1, page_size: 100 }),
      listOrganizeOutputs({ config_id: configId, page: 1, page_size: 100 }),
    ])
    const mapped = toOrganizeConfig(configResponse.data)
    mapped.jobs = (jobResponse.data?.items || []).map(toOrganizeJob)
    outputsByJobId.value = Object.fromEntries(
      (outputResponse.data?.items || [])
        .map((item) => toOrganizeOutput(item))
        .filter((output) => output.jobId)
        .map((output) => [output.jobId, output]),
    )
    config.value = mapped
    window.dispatchEvent(new CustomEvent(ORGANIZE_CONFIG_UNREAD_EVENT, {
      detail: {
        configId: mapped.id,
        hasUnreadOutput: mapped.hasUnreadOutput,
      },
    }))
  } catch (error: any) {
    config.value = null
    outputsByJobId.value = {}
    if (!quiet) MessagePlugin.error(error?.message || '整理任务加载失败')
  } finally {
    loading.value = false
  }
}

const mergeStreamJob = (rawJob: Parameters<typeof toOrganizeJob>[0]) => {
  if (!config.value) return
  const mapped = toOrganizeJob(rawJob)
  const index = config.value.jobs.findIndex((job) => job.id === mapped.id)
  if (index < 0) {
    config.value.jobs = [mapped, ...config.value.jobs]
    return
  }
  config.value.jobs = config.value.jobs.map((job) => (job.id === mapped.id ? mapped : job))
}

const connectToJobStream = async () => {
  eventController?.abort()
  const activeJob = config.value?.jobs.find((job) => job.id === route.query.job) ||
    config.value?.jobs.find(isActive)
  if (!activeJob || !isActive(activeJob)) return

  const controller = new AbortController()
  eventController = controller
  try {
    await streamOrganizeJobEvents(activeJob.id, {
      signal: controller.signal,
      onJob: (job) => {
        mergeStreamJob(job)
        if (!isActive(toOrganizeJob(job))) {
          void loadConfig(true)
        }
      },
    })
  } catch (error: any) {
    if (controller.signal.aborted) return
    MessagePlugin.error(error?.message || '任务实时进度连接失败')
    await loadConfig(true)
  } finally {
    if (eventController === controller) eventController = null
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
    void connectToJobStream()
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
    void connectToJobStream()
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

const openOutput = async (outputId: string, jobId?: string) => {
  if (!config.value) return
  await router.push({
    path: `/platform/organize/outputs/${encodeURIComponent(outputId)}`,
    query: { from: 'config', configId: config.value.id, ...(jobId ? { job: jobId } : {}) },
  })
}

onMounted(() => {
  void loadConfig().then(() => connectToJobStream())
})

onBeforeUnmount(() => {
  eventController?.abort()
})

watch(
  () => route.params.configId,
  () => void loadConfig().then(() => connectToJobStream()),
)

watch(
  () => route.query.job,
  () => void connectToJobStream(),
)
</script>

<style scoped lang="less">
.organize-product-page {
  display: flex;
  width: 100%;
  height: 100%;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.organize-detail-scroll {
  box-sizing: border-box;
  flex: 1;
  width: 100%;
  align-self: stretch;
  min-height: 0;
  overflow-y: auto;
  padding: 26px 32px 56px;
}

.organize-detail-head,
.organize-detail-section {
  width: min(100%, 920px);
  margin-right: auto;
  margin-left: auto;
}

.organize-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
  min-height: 40px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.organize-detail-actions {
  display: flex;
  flex: none;
  justify-content: flex-end;
}

.organize-detail-actions :deep(.t-button) {
  min-width: 104px;
  height: 34px;
  padding: 0 15px;
  border-radius: 6px;
  box-shadow: 0 2px 7px rgba(7, 192, 95, 0.16);
  font-size: 13px;
  font-weight: 500;
}

.organize-detail-identity {
  display: flex;
  align-items: center;
  flex: 1;
  min-width: 0;
}

.organize-detail-identity h2 {
  margin: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 20px;
  font-weight: 600;
  line-height: 28px;
  letter-spacing: 0;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-detail-section {
  padding-top: 16px;
}

.organize-result-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.organize-result-item {
  min-width: 0;
}

.organize-timeline-item {
  display: grid;
  grid-template-columns: 76px 22px minmax(0, 1fr);
  align-items: stretch;
}

.organize-timeline-when {
  display: flex;
  align-items: flex-end;
  padding-top: 10px;
  flex-direction: column;
  gap: 1px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
  text-align: right;
}

.organize-timeline-when strong {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.organize-timeline-node {
  position: relative;
  display: flex;
  justify-content: center;
}

.organize-timeline-node::after {
  position: absolute;
  top: 19px;
  bottom: -1px;
  width: 1px;
  background: var(--td-component-stroke);
  content: '';
}

.organize-timeline-item:last-child .organize-timeline-node::after {
  display: none;
}

.organize-timeline-node > span {
  position: relative;
  z-index: 1;
  width: 8px;
  height: 8px;
  margin-top: 14px;
  border: 2px solid var(--td-brand-color);
  border-radius: 50%;
  background: var(--td-bg-color-container);
  box-shadow: 0 0 0 3px var(--td-brand-color-light);
}

.organize-timeline-item.is-running .organize-timeline-node > span,
.organize-timeline-item.is-queued .organize-timeline-node > span,
.organize-timeline-item.is-repairing .organize-timeline-node > span {
  border-color: var(--td-warning-color);
  box-shadow: 0 0 0 3px var(--td-warning-color-light);
}

.organize-timeline-item.is-failed .organize-timeline-node > span,
.organize-timeline-item.is-canceled .organize-timeline-node > span {
  border-color: var(--td-text-color-placeholder);
  box-shadow: 0 0 0 3px var(--td-bg-color-secondarycontainer);
}

.organize-timeline-content {
  min-width: 0;
  padding-bottom: 8px;
}

.organize-result-card,
.organize-job-state-card {
  box-sizing: border-box;
  width: 100%;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.025);
}

.organize-result-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 28px;
  align-items: center;
  gap: 16px;
  padding: 14px 13px 13px 18px;
  color: inherit;
  cursor: pointer;
  font: inherit;
  text-align: left;
  transition: border-color 0.18s ease, box-shadow 0.18s ease, transform 0.18s ease;
}

.organize-result-card:hover {
  border-color: var(--td-component-border);
  box-shadow: 0 6px 18px rgba(0, 0, 0, 0.055);
  transform: translateY(-1px);
}

.organize-result-card:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.organize-result-item.is-highlighted .organize-result-card,
.organize-result-item.is-highlighted .organize-job-state-card {
  border-color: var(--td-component-stroke);
  box-shadow: inset 3px 0 0 var(--td-brand-color), 0 2px 8px rgba(0, 0, 0, 0.035);
}

.organize-result-card__body {
  display: block;
  min-width: 0;
}

.organize-result-card__title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.organize-result-card__title-row strong {
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.organize-result-card__preview {
  display: -webkit-box;
  margin-top: 6px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.organize-result-card__footer {
  display: block;
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 16px;
}

.organize-result-card__arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  flex: none;
  border: 1px solid transparent;
  border-radius: 50%;
  color: var(--td-text-color-placeholder);
  transition: border-color 0.18s ease, background 0.18s ease, color 0.18s ease;
}

.organize-result-card:hover .organize-result-card__arrow {
  border-color: var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-brand-color);
}

.organize-job-state-card {
  padding: 14px 18px;
}

.organize-job-status-row,
.organize-job-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
}

.organize-job-status-row {
  justify-content: space-between;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.organize-job-state-card > strong {
  display: block;
  margin-top: 12px;
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 22px;
}

.organize-job-state-card > p {
  margin: 7px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.organize-job-actions {
  justify-content: flex-end;
  margin-top: 12px;
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
    padding: 20px 16px 40px;
  }

  .organize-detail-head {
    align-items: stretch;
    flex-wrap: wrap;
    gap: 10px 14px;
    min-height: 0;
    padding-bottom: 16px;
  }

  .organize-detail-identity {
    flex: 1 1 calc(100% - 126px);
  }

  .organize-timeline-item {
    grid-template-columns: 52px 18px minmax(0, 1fr);
  }

  .organize-timeline-when {
    padding-top: 9px;
    font-size: 10px;
    line-height: 14px;
  }

  .organize-timeline-when strong {
    font-size: 11px;
    line-height: 16px;
  }

  .organize-result-card {
    grid-template-columns: minmax(0, 1fr) 26px;
    gap: 10px;
    padding: 14px 11px 13px 14px;
  }

  .organize-result-card__preview {
    font-size: 13px;
    line-height: 20px;
    -webkit-line-clamp: 3;
  }
}
</style>
