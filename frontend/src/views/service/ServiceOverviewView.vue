<template>
  <main class="service-detail-page">
    <header class="service-detail-header">
      <button type="button" class="service-detail-back" @click="backToWorkspace">
        <t-icon name="chevron-left" />
        返回
      </button>
      <div>
        <h1>服务概览</h1>
        <p>{{ service?.name || '服务' }}</p>
      </div>
      <t-button
        variant="outline"
        size="small"
        :loading="loading"
        @click="refreshSummary"
      >
        <template #icon><t-icon name="refresh" /></template>
        刷新摘要
      </t-button>
    </header>

    <div v-if="loading && !profile && !summary" class="service-detail-state">正在加载服务概览</div>
    <div v-else-if="error" class="service-detail-state is-error">{{ error }}</div>
    <div v-else class="service-overview-layout">
      <section class="service-overview-summary">
        <span>当前进展</span>
        <h2>{{ summaryLead }}</h2>
        <div v-if="summary?.schema?.length" class="service-overview-sections">
          <article v-for="section in summary.schema" :key="section.key">
            <strong>{{ section.label }}</strong>
            <p>{{ summarySectionText(section.key) }}</p>
          </article>
        </div>
        <div v-else class="service-overview-empty">服务开展后，这里会根据真实工作资料生成进展摘要。</div>
      </section>

      <aside class="service-overview-side">
        <section class="service-overview-customer">
          <div class="service-detail-section-heading">
            <div>
              <span>家长与学员</span>
              <h2>{{ customer.familyName }}</h2>
            </div>
          </div>
          <dl>
            <div><dt>学员</dt><dd>{{ customer.student }}</dd></div>
            <div><dt>班级</dt><dd>{{ customer.className }}</dd></div>
            <div><dt>主班老师</dt><dd>{{ customer.teacher }}</dd></div>
          </dl>
        </section>

        <section class="service-overview-activity">
          <div class="service-detail-section-heading">
            <div>
              <span>最近动态</span>
              <h2>这一家的变化</h2>
            </div>
            <button type="button" @click="router.push(`${serviceBasePath}/${encodeURIComponent(serviceId)}/activity`)">
              查看全部
            </button>
          </div>
          <div v-if="recentActivity.length" class="service-overview-activity-list">
            <article v-for="item in recentActivity.slice(0, 4)" :key="item.id">
              <span :class="`service-overview-activity-dot is-${item.kind}`" />
              <div>
                <strong>{{ item.title }}</strong>
                <p>{{ item.detail }}</p>
                <small>{{ displayActivityDate(item.createdAt) }}</small>
              </div>
            </article>
          </div>
          <div v-else class="service-overview-empty">还没有新的服务动态。</div>
        </section>
      </aside>

      <section class="service-overview-records">
        <div class="service-detail-section-heading">
          <div>
            <span>重点记录</span>
            <h2>这个服务持续关注什么</h2>
          </div>
          <button type="button" @click="openSettings">调整</button>
        </div>
        <div v-if="profile?.schema?.length" class="service-overview-record-list">
          <div v-for="item in profile.schema" :key="item.key">
            <span>{{ item.label }}</span>
            <strong>{{ displayProfileValue(profile.values?.[item.key]) }}</strong>
          </div>
        </div>
        <div v-else class="service-overview-empty">还没有需要持续记录的内容。</div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  getServiceProfile,
  getServiceSummary,
  refreshServiceSummary,
  type ServiceSpaceProfile,
  type ServiceSpaceSummary,
} from '@/api/service'
import {
  getFirstServiceSession,
  getService,
  loadServiceHub,
} from './serviceHubState'
import {
  buildCustomerInfo,
  displayActivityDate,
  displayValue as displayProfileValue,
} from './servicePresentation'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const error = ref('')
const profile = ref<ServiceSpaceProfile | null>(null)
const summary = ref<ServiceSpaceSummary | null>(null)
const serviceId = computed(() => String(route.params.serviceId || ''))
const service = computed(() => getService(serviceId.value))
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const customer = computed(() =>
  service.value?.customer || buildCustomerInfo(null, null, service.value?.name || '当前家庭'))
const recentActivity = computed(() => service.value?.recentActivity || [])

const summaryLead = computed(() => {
  const labels = summary.value?.schema?.map((item) => item.label).filter(Boolean).slice(0, 3) || []
  if (!labels.length) return '先在会话里推进工作，摘要会随服务内容逐步形成。'
  return `按当前工作方式，这个服务重点关注${labels.join('、')}。`
})

const summarySectionText = (key: string) => {
  const value = summary.value?.sections?.[key]
  if (!value || typeof value !== 'object') return '等待相关资料沉淀后更新'
  const section = value as Record<string, unknown>
  if (typeof section.content === 'string' && section.content.trim()) return section.content
  if (typeof section.status === 'string' && section.status.trim()) return section.status
  return '等待相关资料沉淀后更新'
}

const loadOverview = async () => {
  if (!serviceId.value) return
  loading.value = true
  error.value = ''
  try {
    await loadServiceHub()
    const [profileResponse, summaryResponse] = await Promise.all([
      getServiceProfile(serviceId.value),
      getServiceSummary(serviceId.value),
    ])
    profile.value = profileResponse?.data || null
    summary.value = summaryResponse?.data || null
  } catch (loadError) {
    console.error('[ServiceOverviewView] Failed to load overview:', loadError)
    error.value = '服务概览暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const backToWorkspace = async () => {
  const session = getFirstServiceSession(serviceId.value)
  if (route.meta.mobileEntry) {
    await router.push({
      path: serviceBasePath.value,
      query: {
        service: serviceId.value,
        ...(session ? { session: session.id } : {}),
      },
    })
    return
  }
  await router.push(session
    ? `${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}/sessions/${encodeURIComponent(session.id)}`
    : `${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}`)
}

const openSettings = () =>
  router.push(`${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}/settings/records`)

const refreshSummary = async () => {
  if (!serviceId.value || loading.value) return
  loading.value = true
  try {
    const response = await refreshServiceSummary(serviceId.value)
    summary.value = response?.data || null
    MessagePlugin.success('服务摘要已刷新')
  } catch (refreshError) {
    console.error('[ServiceOverviewView] Failed to refresh summary:', refreshError)
    MessagePlugin.error('服务摘要刷新失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadOverview)
</script>

<style scoped lang="less">
.service-detail-page {
  width: 100%;
  min-width: 0;
  min-height: 100%;
  padding: 20px 28px 40px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-detail-header {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  max-width: 1040px;
  min-height: 54px;
  gap: 14px;
  margin: 0 auto 22px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-detail-header h1,
.service-detail-header p,
.service-overview-summary h2,
.service-detail-section-heading h2,
.service-overview-sections p {
  margin: 0;
}

.service-detail-header h1 {
  font-size: 20px;
  font-weight: 500;
  line-height: 28px;
}

.service-detail-header p {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-detail-back {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-overview-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) minmax(280px, 0.8fr);
  max-width: 1040px;
  gap: 18px;
  margin: 0 auto;
}

.service-overview-summary,
.service-overview-customer,
.service-overview-activity,
.service-overview-records {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-overview-summary {
  padding: 22px;
}

.service-overview-summary > span,
.service-detail-section-heading span {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-overview-summary > h2 {
  margin-top: 8px;
  font-size: 18px;
  font-weight: 500;
  line-height: 28px;
}

.service-overview-sections {
  display: grid;
  gap: 14px;
  margin-top: 22px;
}

.service-overview-sections article {
  padding-top: 14px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-overview-sections strong {
  font-size: 13px;
  font-weight: 500;
}

.service-overview-sections p {
  margin-top: 5px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 21px;
}

.service-overview-records {
  grid-column: 1 / -1;
  padding: 18px;
}

.service-overview-side {
  display: grid;
  align-content: start;
  gap: 18px;
}

.service-overview-customer,
.service-overview-activity {
  padding: 18px;
}

.service-overview-customer .service-detail-section-heading h2,
.service-overview-activity .service-detail-section-heading h2 {
  margin-top: 4px;
}

.service-overview-customer dl {
  display: grid;
  gap: 10px;
  margin: 18px 0 0;
}

.service-overview-customer dl > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.service-overview-customer dt {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-overview-customer dd {
  max-width: 180px;
  margin: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-overview-activity-list {
  display: grid;
  gap: 14px;
  margin-top: 18px;
}

.service-overview-activity-list article {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  gap: 8px;
}

.service-overview-activity-dot {
  width: 7px;
  height: 7px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-overview-activity-dot.is-reminder {
  background: var(--td-warning-color);
}

.service-overview-activity-list strong {
  display: block;
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-overview-activity-list p {
  margin: 2px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-overview-activity-list small {
  display: block;
  margin-top: 3px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-detail-section-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.service-detail-section-heading h2 {
  margin-top: 3px;
  font-size: 15px;
  font-weight: 500;
}

.service-detail-section-heading button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
}

.service-overview-record-list {
  display: grid;
  gap: 8px;
  margin-top: 16px;
}

.service-overview-record-list > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  padding: 9px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-overview-record-list span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-overview-record-list strong {
  font-size: 12px;
  font-weight: 500;
  text-align: right;
}

.service-overview-empty,
.service-detail-state {
  padding: 38px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 21px;
  text-align: center;
}

.service-detail-state {
  max-width: 1040px;
  margin: 0 auto;
}

.service-detail-state.is-error {
  color: var(--td-error-color);
}

@media (max-width: 760px) {
  .service-detail-page {
    padding: 14px 16px 28px;
  }

  .service-overview-layout {
    grid-template-columns: 1fr;
  }

  .service-overview-records {
    grid-column: auto;
  }

  .service-detail-header {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .service-detail-header > :last-child {
    grid-column: 1 / -1;
    justify-self: end;
    margin-bottom: 10px;
  }
}
</style>
