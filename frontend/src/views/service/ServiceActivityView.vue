<template>
  <main class="service-activity-page">
    <header class="service-activity-header">
      <button type="button" class="service-activity-back" @click="backToWorkspace">
        <t-icon name="chevron-left" />
        返回
      </button>
      <div>
        <h1>动态</h1>
        <p>{{ service?.name || '服务' }}</p>
      </div>
    </header>

    <div class="service-activity-toolbar" role="tablist" aria-label="动态筛选">
      <button
        v-for="option in filterOptions"
        :key="option.value"
        type="button"
        :class="{ active: filter === option.value }"
        role="tab"
        :aria-selected="filter === option.value"
        @click="filter = option.value"
      >
        {{ option.label }}
      </button>
    </div>

    <div v-if="loading" class="service-activity-state">正在加载动态</div>
    <div v-else-if="error" class="service-activity-state is-error">{{ error }}</div>
    <div v-else-if="!filteredActivities.length" class="service-activity-state">
      还没有可展示的动态。开始一段工作后，这里会记录重点变化和服务动作。
    </div>
    <div v-else class="service-activity-groups">
      <section v-for="group in groupedActivities" :key="group.label">
        <h2>{{ group.label }}</h2>
        <div class="service-activity-list">
          <article v-for="item in group.items" :key="item.id" class="service-activity-item">
            <span :class="`service-activity-marker is-${item.kind}`" />
            <div class="service-activity-item-main">
              <div class="service-activity-item-title">
                <strong>{{ item.title }}</strong>
                <time>{{ displayActivityDate(item.createdAt) }}</time>
              </div>
              <p>{{ item.detail }}</p>
              <small>{{ item.source }} · 这一家的服务记录</small>
            </div>
          </article>
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getService,
  getFirstServiceSession,
  loadServiceHub,
} from './serviceHubState'
import {
  listServiceFacts,
  listServiceReminders,
  type ServiceReminder,
} from '@/api/service'
import {
  displayActivityDate,
  factToActivity,
  reminderToActivity,
  sortActivities,
  type ServiceActivityItem,
} from './servicePresentation'

type ActivityFilter = 'all' | 'customer' | 'service'

const route = useRoute()
const router = useRouter()
const serviceId = computed(() => String(route.params.serviceId || ''))
const service = computed(() => getService(serviceId.value))
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const loading = ref(true)
const error = ref('')
const filter = ref<ActivityFilter>('all')
const activities = ref<ServiceActivityItem[]>([])

const filterOptions = [
  { label: '全部', value: 'all' as const },
  { label: '这一家的变化', value: 'customer' as const },
  { label: '服务动作', value: 'service' as const },
]

const filteredActivities = computed(() => {
  if (filter.value === 'customer') return activities.value.filter((item) => item.kind === 'fact')
  if (filter.value === 'service') return activities.value.filter((item) => item.kind === 'reminder')
  return activities.value
})

const dayLabel = (value?: string) => {
  if (!value) return '最近'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '最近'
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  }).format(date)
}

const groupedActivities = computed(() => {
  const groups = new Map<string, ServiceActivityItem[]>()
  filteredActivities.value.forEach((item) => {
    const label = dayLabel(item.createdAt)
    const items = groups.get(label) || []
    items.push(item)
    groups.set(label, items)
  })
  return Array.from(groups, ([label, items]) => ({ label, items }))
})

const loadActivity = async () => {
  if (!serviceId.value) return
  loading.value = true
  error.value = ''
  try {
    await loadServiceHub()
    const [factsResponse, remindersResponse] = await Promise.all([
      listServiceFacts(serviceId.value, { page: 1, page_size: 100 }),
      listServiceReminders(serviceId.value, { page: 1, page_size: 100 }),
    ])
    activities.value = sortActivities([
      ...(factsResponse?.data?.items || []).map(factToActivity),
      ...(remindersResponse?.data?.items || []).map((item: ServiceReminder) => reminderToActivity(item)),
    ])
  } catch (loadError) {
    console.error('[ServiceActivityView] Failed to load service activity:', loadError)
    error.value = '服务动态暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const backToWorkspace = async () => {
  const session = getFirstServiceSession(serviceId.value)
  if (route.meta.mobileEntry) {
    await router.push({
      path: serviceBasePath.value,
      query: { service: serviceId.value, ...(session ? { session: session.id } : {}) },
    })
    return
  }
  await router.push(session
    ? `${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}/sessions/${encodeURIComponent(session.id)}`
    : `${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}`)
}

onMounted(loadActivity)
</script>

<style scoped lang="less">
.service-activity-page {
  width: 100%;
  min-width: 0;
  min-height: 100%;
  padding: 20px 28px 40px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-activity-header {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  max-width: 880px;
  min-height: 54px;
  gap: 14px;
  margin: 0 auto 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-activity-header h1,
.service-activity-header p {
  margin: 0;
}

.service-activity-header h1 {
  font-size: 20px;
  font-weight: 500;
  line-height: 28px;
}

.service-activity-header p {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-activity-back {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-activity-toolbar,
.service-activity-groups {
  max-width: 880px;
  margin: 0 auto;
}

.service-activity-toolbar {
  display: flex;
  gap: 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-activity-toolbar button {
  min-height: 36px;
  padding: 0 0 8px;
  border: 0;
  border-bottom: 2px solid transparent;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
}

.service-activity-toolbar button.active {
  border-bottom-color: var(--td-brand-color);
  color: var(--td-brand-color-7);
  font-weight: 500;
}

.service-activity-groups {
  padding-top: 24px;
}

.service-activity-groups section + section {
  margin-top: 28px;
}

.service-activity-groups h2 {
  margin: 0 0 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
}

.service-activity-list {
  display: flex;
  flex-direction: column;
}

.service-activity-item {
  display: grid;
  grid-template-columns: 10px minmax(0, 1fr);
  gap: 12px;
  padding: 0 0 18px;
}

.service-activity-item + .service-activity-item {
  padding-top: 18px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-activity-marker {
  width: 8px;
  height: 8px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-activity-marker.is-reminder {
  background: var(--td-warning-color);
}

.service-activity-item-title {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 14px;
}

.service-activity-item-title strong {
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
}

.service-activity-item-title time {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-activity-item p {
  margin: 5px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 21px;
}

.service-activity-item small {
  display: block;
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-activity-state {
  max-width: 880px;
  margin: 0 auto;
  padding: 80px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 21px;
  text-align: center;
}

.service-activity-state.is-error {
  color: var(--td-error-color);
}

@media (max-width: 680px) {
  .service-activity-page {
    padding: 16px;
  }

  .service-activity-item-title {
    align-items: flex-start;
    flex-direction: column;
    gap: 3px;
  }
}
</style>

