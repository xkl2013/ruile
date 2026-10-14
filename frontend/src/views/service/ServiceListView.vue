<template>
  <div class="service-hub-page">
    <section class="service-hub-list-view">
      <header class="service-hub-page-header">
        <div>
          <h1>{{ serviceHubState.mode === 'archived' ? '归档服务' : '服务' }}</h1>
          <p>每个服务一个空间，一个主理人调度多个 Agent</p>
        </div>
        <div v-if="serviceHubState.mode !== 'archived'" class="service-hub-page-actions">
          <t-button variant="outline" @click="showArchivedServices">查看归档服务</t-button>
          <t-button theme="primary" class="service-hub-primary-button" @click="openCreate">
            <template #icon><t-icon name="add" /></template>
            新建服务
          </t-button>
        </div>
        <t-button v-else variant="text" theme="primary" @click="showActiveServices">
          返回使用中的服务
        </t-button>
      </header>

      <div class="service-hub-list-main">
        <div class="service-hub-section-head">
          <span class="service-hub-section-title">
            {{ serviceHubState.mode === 'archived' ? '已归档' : '我的服务' }}
          </span>
          <div v-if="services.length > 0" class="service-hub-section-tools">
            <t-select v-model="sortMode" class="service-hub-sort" size="small" :options="sortOptions" />
            <t-input v-model="serviceQuery" class="service-hub-search" size="small" placeholder="搜索服务">
              <template #prefix-icon><t-icon name="search" /></template>
            </t-input>
          </div>
        </div>

        <template v-if="services.length > 0">
          <div v-if="filteredServices.length" class="service-hub-grid">
            <article
              v-for="service in filteredServices"
              :key="service.id"
              class="service-hub-card"
              @click="openWorkspace(service.id)"
            >
              <div class="service-hub-card-top">
                <div class="service-hub-card-icon"><t-icon name="folder" /></div>
                <strong>{{ service.customer?.familyName || service.name }}</strong>
                <span v-if="service.state !== 'active'" class="service-hub-card-state">
                  {{ serviceStateLabel(service.state) }}
                </span>
                <t-dropdown trigger="click" placement="bottom-right" @click.stop>
                  <button type="button" class="service-hub-more" aria-label="更多操作" @click.stop>
                    <t-icon name="more" />
                  </button>
                  <template #dropdown>
                    <t-dropdown-menu>
                      <t-dropdown-item @click="archiveService(service)">
                        {{ service.state === 'archived' ? '恢复服务' : '归档服务' }}
                      </t-dropdown-item>
                    </t-dropdown-menu>
                  </template>
                </t-dropdown>
              </div>
              <div class="service-hub-card-customer">
                学员 {{ service.customer?.student || '待补充' }}
                · {{ service.customer?.className || '班级待补充' }}
                · 主班 {{ service.customer?.teacher || '待补充' }}
              </div>
              <button
                v-if="service.recentActivity?.[0]"
                type="button"
                class="service-hub-card-activity"
                @click.stop="openActivity(service.id)"
              >
                <span>最近动态</span>
                <strong>{{ service.recentActivity[0].title }}</strong>
              </button>
              <button
                v-if="service.recentActivity?.find((item) => item.kind === 'reminder')"
                type="button"
                class="service-hub-card-activity is-reminder"
                @click.stop="openReminders(service.id)"
              >
                <span>新待办</span>
                <strong>{{ service.recentActivity.find((item) => item.kind === 'reminder')?.title }}</strong>
              </button>
              <div class="service-hub-card-meta">
                <span>更新于 {{ service.updatedLabel }}</span>
                <button type="button" @click.stop="openWorkspace(service.id)">
                  进入服务 <t-icon name="chevron-right" />
                </button>
              </div>
            </article>
          </div>
          <div v-else class="service-hub-no-result">
            <span>没有匹配“{{ serviceQuery.trim() }}”的服务</span>
            <button type="button" @click="serviceQuery = ''">清除筛选</button>
          </div>
        </template>
        <div v-else class="service-hub-empty-state">
          <div>
            <h2>{{ emptyTitle }}</h2>
            <p v-if="serviceHubState.mode === 'archived'">归档后的服务会保留历史内容，需要时可以恢复。</p>
            <p v-else-if="archivedServices.length">归档服务不会出现在工作导航中，可以进入归档列表恢复。</p>
            <p v-else>创建服务后，客户信息与工作方式由后台配置并自动带入。</p>
          </div>
          <t-button v-if="serviceHubState.mode === 'archived'" variant="outline" @click="showActiveServices">
            返回我的服务
          </t-button>
          <t-button v-else variant="outline" @click="openCreate">从空白创建</t-button>
        </div>
      </div>
    </section>

    <ServiceCreateDialog
      :visible="createDialogVisible"
      :source="createSource"
      :submitting="createSubmitting"
      @update:visible="setCreateDialogVisible"
      @submit="submitForm"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  createServiceSpace,
  importServiceOrganizeOutput,
  setServiceSpaceState,
} from '@/api/service'
import ServiceCreateDialog from './ServiceCreateDialog.vue'
import {
  getService,
  getSession,
  loadServiceHub,
  serviceHubState,
  type ServiceRecord,
} from './serviceHubState'
import {
  ensureServiceSession,
  serviceBasePath,
  serviceDetailPath,
  serviceStateLabel,
  serviceWorkspaceLocation,
} from './serviceNavigation'

type SortMode = 'recent' | 'created' | 'name'

const route = useRoute()
const router = useRouter()
const serviceQuery = ref('')
const sortMode = ref<SortMode>('recent')
const createDialogVisible = ref(false)
const createSubmitting = ref(false)
const sortOptions = [
  { label: '按最近活动', value: 'recent' },
  { label: '按创建时间', value: 'created' },
  { label: '按名称', value: 'name' },
]

const archivedServices = computed(() =>
  serviceHubState.services.filter((service) => service.state === 'archived'))
const services = computed(() => serviceHubState.mode === 'archived'
  ? archivedServices.value
  : serviceHubState.services.filter((service) => service.state !== 'archived'))
const filteredServices = computed(() => {
  const query = serviceQuery.value.trim().toLowerCase()
  const rows = services.value.filter((service) =>
    `${service.name} ${service.description}`.toLowerCase().includes(query))
  return [...rows].sort((a, b) => {
    if (sortMode.value === 'name') return a.name.localeCompare(b.name, 'zh-CN')
    if (sortMode.value === 'created') return b.createdAt - a.createdAt
    return b.updatedAt - a.updatedAt
  })
})
const emptyTitle = computed(() => {
  if (serviceHubState.mode === 'archived') return '还没有归档服务'
  return archivedServices.value.length ? '服务都已归档' : '还没有服务'
})
const createSource = computed(() => {
  const sourceId = typeof route.query.copy === 'string' ? route.query.copy : ''
  return sourceId ? getService(sourceId) || null : null
})

const openCreate = () => {
  createDialogVisible.value = true
}

const setCreateDialogVisible = (visible: boolean) => {
  createDialogVisible.value = visible
  if (visible) return
  if (
    route.query.create !== '1'
    && typeof route.query.copy !== 'string'
    && typeof route.query.source_type !== 'string'
    && typeof route.query.source_id !== 'string'
  ) return
  const query = { ...route.query }
  delete query.create
  delete query.copy
  delete query.source_type
  delete query.source_id
  void router.replace({ path: serviceBasePath(route), query })
}

const importPendingContextSource = async (serviceId: string) => {
  const sourceType = typeof route.query.source_type === 'string' ? route.query.source_type : ''
  const sourceId = typeof route.query.source_id === 'string' ? route.query.source_id : ''
  if (route.meta.mobileEntry || sourceType !== 'organize_output' || !sourceId) return
  try {
    await importServiceOrganizeOutput(serviceId, sourceId)
  } catch (error) {
    console.error('[ServiceListView] Failed to import pending context source:', error)
    MessagePlugin.error('服务已创建，但整理结果带入失败，请从整理详情页重试')
  }
}

const submitForm = async (payload: { name: string }) => {
  const name = payload.name.trim()
  if (!name || createSubmitting.value) return
  createSubmitting.value = true
  try {
    const response = await createServiceSpace({ name, activate: true })
    const serviceId = response?.data?.id
    if (!serviceId) throw new Error('missing service id')
    await loadServiceHub(true)
    const session = await ensureServiceSession(serviceId)
    await importPendingContextSource(serviceId)
    serviceHubState.activeServiceId = serviceId
    serviceHubState.activeSessionId = session.id
    createDialogVisible.value = false
    await router.replace(serviceWorkspaceLocation(route, serviceId, session.id))
    MessagePlugin.success('服务已创建')
  } catch (error) {
    console.error('[ServiceListView] Failed to create service:', error)
    const message = error instanceof Error ? error.message : ''
    MessagePlugin.error(message ? `服务创建失败：${message}` : '服务创建失败，请稍后重试')
  } finally {
    createSubmitting.value = false
  }
}

const openWorkspace = async (serviceId: string, panel?: 'reminders') => {
  await loadServiceHub()
  const service = getService(serviceId)
  if (!service) return
  const sessionId = typeof route.query.session === 'string' ? route.query.session : ''
  const routeSession = sessionId ? getSession(sessionId) : undefined
  const session = routeSession?.serviceId === serviceId
    ? routeSession
    : await ensureServiceSession(serviceId)
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = session.id
  await router.push(serviceWorkspaceLocation(
    route,
    serviceId,
    session.id,
    panel ? { panel } : {},
  ))
}

const openActivity = (serviceId: string) =>
  router.push(serviceDetailPath(route, serviceId, 'activity'))
const openReminders = (serviceId: string) => openWorkspace(serviceId, 'reminders')
const showArchivedServices = () => {
  serviceHubState.mode = 'archived'
  serviceQuery.value = ''
}
const showActiveServices = () => {
  serviceHubState.mode = 'data'
  serviceQuery.value = ''
}

const archiveService = async (service: ServiceRecord) => {
  const nextState = service.state === 'archived' ? 'active' : 'archived'
  try {
    await setServiceSpaceState(service.id, nextState)
    service.state = nextState
    MessagePlugin.success(nextState === 'archived' ? '服务已归档' : '服务已恢复')
  } catch (error) {
    console.error('[ServiceListView] Failed to change service state:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}

onMounted(() => loadServiceHub())

watch(
  () => [route.query.create, route.query.copy].join('|'),
  () => {
    if (route.query.create === '1' || typeof route.query.copy === 'string') {
      createDialogVisible.value = true
    }
  },
  { immediate: true },
)
</script>
