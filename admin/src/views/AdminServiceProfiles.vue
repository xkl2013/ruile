<template>
  <section class="service-config-page">
    <div class="service-config-page__header">
      <div>
        <h2>服务配置</h2>
        <p>查看当前工作区的服务空间及其运行状态。专家、知识库和 Skill 在具体服务空间中配置。</p>
      </div>
      <t-button variant="outline" :loading="loading" @click="loadServiceSpaces">
        <template #icon><t-icon name="refresh" /></template>
        刷新
      </t-button>
    </div>

    <t-alert v-if="errorMessage" theme="error" :message="errorMessage">
      <template #operation>
        <t-button size="small" variant="outline" @click="loadServiceSpaces">重试</t-button>
      </template>
    </t-alert>

    <div class="service-config-summary">
      <article>
        <span>服务空间</span>
        <strong>{{ services.length }}</strong>
        <em>当前工作区</em>
      </article>
      <article>
        <span>运行中</span>
        <strong>{{ activeCount }}</strong>
        <em>已启用服务</em>
      </article>
      <article>
        <span>暂停或草稿</span>
        <strong>{{ pausedOrDraftCount }}</strong>
        <em>待配置服务</em>
      </article>
    </div>

    <section class="service-config-panel">
      <div class="panel-title">
        <span>
          <strong>服务空间</strong>
          <em>每个空间独立保存会话、专家配置和产出物。</em>
        </span>
      </div>

      <div v-if="loading && services.length === 0" class="service-empty">
        <t-loading size="small" />
        <span>正在读取服务空间...</span>
      </div>
      <div v-else-if="services.length === 0" class="service-empty">
        <t-icon name="info-circle" />
        <span>暂无服务空间</span>
      </div>
      <div v-else class="service-list">
        <article v-for="service in services" :key="service.id" class="service-card">
          <div class="service-card__head">
            <span class="service-card__icon">
              <t-icon name="service" />
            </span>
            <span class="service-card__title">
              <strong>{{ service.name }}</strong>
              <em>{{ service.template_key || '自定义服务' }}</em>
            </span>
            <t-tag :theme="stateTheme(service.state)" variant="light">
              {{ stateLabel(service.state) }}
            </t-tag>
          </div>

          <p>{{ service.description || '未填写服务描述' }}</p>

          <dl>
            <div>
              <dt>专家</dt>
              <dd>在服务空间中配置</dd>
            </div>
            <div>
              <dt>知识库</dt>
              <dd>{{ service.knowledge_base_ids?.length || 0 }} 个</dd>
            </div>
            <div>
              <dt>更新时间</dt>
              <dd>{{ formatDate(service.updated_at || service.created_at) }}</dd>
            </div>
          </dl>

          <div class="service-card__actions">
            <t-button
              v-if="service.state === 'active'"
              size="small"
              variant="outline"
              :loading="updatingId === service.id"
              @click="changeState(service, 'paused')"
            >
              暂停
            </t-button>
            <t-button
              v-else-if="service.state === 'paused'"
              size="small"
              theme="primary"
              variant="outline"
              :loading="updatingId === service.id"
              @click="changeState(service, 'active')"
            >
              启用
            </t-button>
          </div>
        </article>
      </div>
    </section>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listServiceSpaces,
  setServiceSpaceState,
  type ServiceSpace,
  type ServiceSpaceState,
} from '@/api/service'

const services = ref<ServiceSpace[]>([])
const loading = ref(false)
const updatingId = ref('')
const errorMessage = ref('')

const activeCount = computed(() => services.value.filter((service) => service.state === 'active').length)
const pausedOrDraftCount = computed(() => (
  services.value.filter((service) => service.state === 'paused' || service.state === 'draft').length
))

async function loadServiceSpaces() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await listServiceSpaces({ include_archived: false })
    services.value = response?.data || []
  } catch (error: any) {
    console.warn('[AdminServiceProfiles] Failed to load service spaces:', error)
    errorMessage.value = error?.message || '服务空间读取失败'
    services.value = []
  } finally {
    loading.value = false
  }
}

async function changeState(service: ServiceSpace, state: ServiceSpaceState) {
  updatingId.value = service.id
  try {
    const response = await setServiceSpaceState(service.id, state)
    if (response?.data) {
      services.value = services.value.map((item) => item.id === service.id ? response.data : item)
    }
    MessagePlugin.success(state === 'active' ? '服务空间已启用' : '服务空间已暂停')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '服务空间状态更新失败')
  } finally {
    updatingId.value = ''
  }
}

function stateLabel(state: ServiceSpaceState) {
  const labels: Record<ServiceSpaceState, string> = {
    draft: '草稿',
    active: '运行中',
    paused: '已暂停',
    archived: '已归档',
  }
  return labels[state] || state
}

function stateTheme(state: ServiceSpaceState) {
  if (state === 'active') return 'success'
  if (state === 'paused') return 'warning'
  if (state === 'archived') return 'default'
  return 'primary'
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

onMounted(() => {
  void loadServiceSpaces()
})
</script>

<style scoped>
.service-config-page {
  display: grid;
  gap: 18px;
  width: min(100%, 1280px);
}

.service-config-page__header {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 18px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.service-config-page__header h2 {
  margin: 0 0 6px;
  color: var(--admin-text);
  font-size: 22px;
  font-weight: 650;
  line-height: 1.35;
}

.service-config-page__header p {
  max-width: 760px;
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 14px;
  line-height: 1.65;
}

.service-config-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}

.service-config-summary article {
  display: grid;
  gap: 5px;
  min-width: 0;
  padding: 15px 16px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.service-config-summary span,
.service-config-summary em {
  color: var(--admin-text-secondary);
  font-size: 12px;
  font-style: normal;
}

.service-config-summary strong {
  color: var(--admin-text);
  font-size: 19px;
  font-weight: 650;
}

.service-config-panel {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.panel-title {
  display: flex;
  padding: 17px 18px;
  border-bottom: 1px solid var(--admin-border);
}

.panel-title span {
  display: grid;
  gap: 4px;
}

.panel-title strong {
  color: var(--admin-text);
  font-size: 16px;
  font-weight: 650;
}

.panel-title em {
  color: var(--admin-text-secondary);
  font-size: 13px;
  font-style: normal;
}

.service-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  padding: 18px;
}

.service-card {
  display: grid;
  gap: 13px;
  min-width: 0;
  padding: 16px;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface-soft);
}

.service-card__head {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 11px;
  align-items: center;
}

.service-card__icon {
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  border: 1px solid rgba(15, 122, 92, 0.16);
  border-radius: 8px;
  background: var(--admin-brand-soft);
  color: var(--admin-brand);
}

.service-card__title {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.service-card__title strong,
.service-card__title em {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-card__title strong {
  color: var(--admin-text);
  font-size: 15px;
  font-weight: 650;
}

.service-card__title em {
  color: var(--admin-text-secondary);
  font-size: 12px;
  font-style: normal;
}

.service-card p {
  min-height: 42px;
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 13px;
  line-height: 1.65;
}

.service-card dl {
  display: grid;
  gap: 0;
  margin: 0;
  overflow: hidden;
  border: 1px solid rgba(219, 228, 231, 0.88);
  border-radius: 8px;
  background: var(--admin-surface);
}

.service-card dl div {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: 10px;
  padding: 8px 10px;
  border-top: 1px solid rgba(219, 228, 231, 0.72);
}

.service-card dl div:first-child {
  border-top: 0;
}

.service-card dt,
.service-card dd {
  margin: 0;
  font-size: 13px;
  line-height: 1.5;
}

.service-card dt {
  color: var(--admin-text-muted);
}

.service-card dd {
  min-width: 0;
  overflow: hidden;
  color: var(--admin-text);
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-card__actions {
  display: flex;
  justify-content: flex-end;
}

.service-empty {
  display: flex;
  min-height: 180px;
  align-items: center;
  justify-content: center;
  gap: 10px;
  color: var(--admin-text-secondary);
  font-size: 13px;
}

@media (max-width: 900px) {
  .service-config-page__header {
    display: grid;
  }

  .service-config-summary,
  .service-list {
    grid-template-columns: 1fr;
  }
}
</style>
