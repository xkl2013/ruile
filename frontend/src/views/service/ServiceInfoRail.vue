<template>
  <aside class="service-info-rail">
    <section class="service-info-rail-section">
      <div class="service-info-rail-heading">
        <span>家长与学员</span>
        <t-icon name="user" />
      </div>
      <strong class="service-info-rail-family">{{ customer.familyName }}</strong>
      <dl class="service-info-rail-details">
        <div>
          <dt>学员</dt>
          <dd>{{ customer.student }}</dd>
        </div>
        <div>
          <dt>班级</dt>
          <dd>{{ customer.className }}</dd>
        </div>
        <div>
          <dt>主班</dt>
          <dd>{{ customer.teacher }}</dd>
        </div>
      </dl>
    </section>

    <section class="service-info-rail-section service-info-rail-activity">
      <div class="service-info-rail-heading">
        <span>最近动态</span>
        <button type="button" @click="$emit('open-activity')">查看全部 <t-icon name="chevron-right" /></button>
      </div>
      <div v-if="loading" class="service-info-rail-empty">正在加载动态</div>
      <div v-else-if="activities.length" class="service-info-rail-list">
        <article v-for="item in activities.slice(0, 3)" :key="item.id">
          <span :class="`service-info-rail-dot is-${item.kind}`" />
          <div>
            <strong>{{ item.title }}</strong>
            <p>{{ item.detail }}</p>
            <small>{{ displayActivityDate(item.createdAt) }} · {{ item.source }}</small>
          </div>
        </article>
      </div>
      <div v-else class="service-info-rail-empty">这个服务还没有新的动态</div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import {
  getServiceSubjectProfile,
  listServiceFacts,
  listServiceReminders,
  listServiceSubjects,
  type ServiceReminder,
  type ServiceSpaceProfile,
  type ServiceSubject,
} from '@/api/service'
import {
  buildCustomerInfo,
  displayActivityDate,
  factToActivity,
  reminderToActivity,
  sortActivities,
  type ServiceActivityItem,
  type ServiceCustomerInfo,
} from './servicePresentation'

const props = defineProps<{
  serviceId: string
  fallbackName?: string
}>()

defineEmits<{
  'open-activity': []
}>()

const loading = ref(true)
const customer = ref<ServiceCustomerInfo>(buildCustomerInfo(null, null, props.fallbackName || '当前家庭'))
const activities = ref<ServiceActivityItem[]>([])

const loadRail = async () => {
  if (!props.serviceId) return
  loading.value = true
  try {
    const [subjectsResponse, factsResponse, remindersResponse] = await Promise.all([
      listServiceSubjects(props.serviceId, { page: 1, page_size: 1 }),
      listServiceFacts(props.serviceId, { page: 1, page_size: 12 }),
      listServiceReminders(props.serviceId, { page: 1, page_size: 12 }),
    ])
    const subject = subjectsResponse?.data?.items?.[0] || null
    let profile: ServiceSpaceProfile | null = null
    if (subject) {
      const profileResponse = await getServiceSubjectProfile(props.serviceId, subject.id)
      profile = profileResponse?.data || null
    }
    customer.value = buildCustomerInfo(subject, profile, props.fallbackName || '当前家庭')
    const facts = factsResponse?.data?.items || []
    const reminders = remindersResponse?.data?.items || []
    activities.value = sortActivities([
      ...facts.map(factToActivity),
      ...reminders.map((item: ServiceReminder) => reminderToActivity(item)),
    ])
  } catch (error) {
    console.error('[ServiceInfoRail] Failed to load service context:', error)
    customer.value = buildCustomerInfo(null, null, props.fallbackName || '当前家庭')
    activities.value = []
  } finally {
    loading.value = false
  }
}

onMounted(loadRail)
watch(() => props.serviceId, loadRail)
</script>

<style scoped lang="less">
.service-info-rail {
  display: flex;
  width: 320px;
  min-width: 320px;
  flex: 0 0 320px;
  flex-direction: column;
  overflow-y: auto;
  border-left: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.service-info-rail-section {
  padding: 18px 18px 20px;
}

.service-info-rail-section + .service-info-rail-section {
  border-top: 1px solid var(--td-component-stroke);
}

.service-info-rail-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-info-rail-heading .t-icon {
  color: var(--td-brand-color);
}

.service-info-rail-heading button {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
}

.service-info-rail-family {
  display: block;
  margin-top: 8px;
  color: var(--td-text-color-primary);
  font-size: 18px;
  font-weight: 600;
  line-height: 26px;
}

.service-info-rail-details {
  display: grid;
  gap: 10px;
  margin: 16px 0 0;
}

.service-info-rail-details > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.service-info-rail-details dt {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-info-rail-details dd {
  max-width: 190px;
  margin: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-info-rail-list {
  display: flex;
  margin-top: 14px;
  flex-direction: column;
  gap: 14px;
}

.service-info-rail-list article {
  display: grid;
  grid-template-columns: 8px minmax(0, 1fr);
  gap: 9px;
}

.service-info-rail-dot {
  width: 7px;
  height: 7px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--td-brand-color);
}

.service-info-rail-dot.is-reminder {
  background: var(--td-warning-color);
}

.service-info-rail-list strong {
  display: block;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-info-rail-list p {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-info-rail-list small {
  display: block;
  margin-top: 4px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-info-rail-empty {
  padding: 26px 0 8px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 20px;
}

@media (max-width: 920px) {
  .service-info-rail {
    width: 280px;
    min-width: 280px;
    flex-basis: 280px;
  }
}

@media (max-width: 680px) {
  .service-info-rail {
    display: none;
  }
}
</style>
