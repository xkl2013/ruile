<template>
  <main class="service-subjects-page">
    <header class="service-subjects-header">
      <button type="button" class="service-subjects-back" @click="backToWorkspace">
        <t-icon name="chevron-left" />
        返回
      </button>
      <div>
        <h1>服务对象</h1>
        <p>{{ service?.name || '服务' }}</p>
      </div>
    </header>

    <div class="service-subjects-layout">
      <section class="service-subjects-create">
        <span>添加服务对象</span>
        <h2>这个服务主要在跟进谁</h2>
        <p>只需填写便于团队识别的名称，引用编号由系统自动生成。</p>
        <t-input
          v-model="draftName"
          :maxlength="255"
          :disabled="!canEdit"
          placeholder="例如：果果家、东苑分园、春季调研课题"
          @keydown.enter.prevent="createSubject"
        />
        <t-button block theme="primary" :loading="saving" :disabled="!canEdit" @click="createSubject">
          添加
        </t-button>
        <small v-if="!canEdit">当前角色可以查看服务对象，但不能修改。</small>
      </section>

      <section class="service-subjects-list">
        <div class="service-subjects-list-head">
          <div>
            <span>已添加</span>
            <h2>{{ subjects.length }} 个服务对象</h2>
          </div>
        </div>
        <div v-if="loading" class="service-subjects-state">正在加载服务对象</div>
        <div v-else-if="error" class="service-subjects-state is-error">{{ error }}</div>
        <div v-else-if="subjects.length" class="service-subjects-items">
          <article v-for="subject in subjects" :key="subject.id">
            <div class="service-subjects-avatar">{{ subject.display_name.slice(0, 1) }}</div>
            <div>
              <strong>{{ subject.display_name }}</strong>
              <small>{{ formatDate(subject.updated_at || subject.created_at) }}</small>
            </div>
            <button v-if="canEdit" type="button" aria-label="删除服务对象" @click="confirmDelete(subject)">
              <t-icon name="delete" />
            </button>
          </article>
        </div>
        <div v-else class="service-subjects-state">
          还没有服务对象。添加后，对话中沉淀的重点记录可以归到具体对象下。
        </div>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import {
  createServiceSubject,
  deleteServiceSubject,
  listServiceSubjects,
  type ServiceSubject,
} from '@/api/service'
import {
  getFirstServiceSession,
  getService,
  loadServiceHub,
} from './serviceHubState'

const route = useRoute()
const router = useRouter()
const serviceId = computed(() => String(route.params.serviceId || ''))
const service = computed(() => getService(serviceId.value))
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const canEdit = computed(() => ['拥有者', '管理员', '编辑者'].includes(service.value?.role || ''))
const subjects = ref<ServiceSubject[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const draftName = ref('')

const loadSubjects = async () => {
  if (!serviceId.value) return
  loading.value = true
  error.value = ''
  try {
    await loadServiceHub()
    const response = await listServiceSubjects(serviceId.value, { page: 1, page_size: 100 })
    subjects.value = response?.data?.items || []
  } catch (loadError) {
    console.error('[ServiceSubjectsView] Failed to load subjects:', loadError)
    error.value = '服务对象暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const createSubject = async () => {
  const displayName = draftName.value.trim()
  if (!canEdit.value || !displayName || saving.value) {
    if (!displayName) MessagePlugin.warning('请填写服务对象名称')
    return
  }
  saving.value = true
  try {
    const response = await createServiceSubject(serviceId.value, { display_name: displayName })
    if (response?.data) subjects.value = [response.data, ...subjects.value]
    draftName.value = ''
    MessagePlugin.success('服务对象已添加')
  } catch (createError) {
    console.error('[ServiceSubjectsView] Failed to create subject:', createError)
    MessagePlugin.error('服务对象添加失败')
  } finally {
    saving.value = false
  }
}

const confirmDelete = (subject: ServiceSubject) => {
  const dialog = DialogPlugin.confirm({
    header: '删除服务对象',
    body: `确定删除“${subject.display_name}”吗？已经沉淀的历史记录不会被改写。`,
    confirmBtn: { content: '删除', theme: 'danger' },
    cancelBtn: '取消',
    onConfirm: async () => {
      dialog.setConfirmLoading(true)
      try {
        await deleteServiceSubject(serviceId.value, subject.id)
        subjects.value = subjects.value.filter((item) => item.id !== subject.id)
        dialog.destroy()
        MessagePlugin.success('服务对象已删除')
      } catch (deleteError) {
        console.error('[ServiceSubjectsView] Failed to delete subject:', deleteError)
        dialog.setConfirmLoading(false)
        MessagePlugin.error('服务对象删除失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

const formatDate = (value?: string) => {
  if (!value) return '刚刚更新'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚更新'
  return `${date.getMonth() + 1} 月 ${date.getDate()} 日更新`
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

onMounted(loadSubjects)
</script>

<style scoped lang="less">
.service-subjects-page {
  width: 100%;
  min-width: 0;
  min-height: 100%;
  padding: 20px 28px 40px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-subjects-header {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  max-width: 1040px;
  min-height: 54px;
  gap: 14px;
  margin: 0 auto 22px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-subjects-header h1,
.service-subjects-header p,
.service-subjects-create h2,
.service-subjects-create p,
.service-subjects-list-head h2 {
  margin: 0;
}

.service-subjects-header h1 {
  font-size: 20px;
  font-weight: 500;
}

.service-subjects-header p,
.service-subjects-create > span,
.service-subjects-list-head span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-subjects-back {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-subjects-layout {
  display: grid;
  grid-template-columns: 300px minmax(0, 1fr);
  max-width: 1040px;
  gap: 18px;
  margin: 0 auto;
}

.service-subjects-create,
.service-subjects-list {
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-subjects-create {
  align-self: start;
}

.service-subjects-create h2,
.service-subjects-list-head h2 {
  margin-top: 3px;
  font-size: 15px;
  font-weight: 500;
}

.service-subjects-create p {
  margin: 10px 0 16px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 19px;
}

.service-subjects-create .t-button {
  margin-top: 10px;
}

.service-subjects-create > small {
  display: block;
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-subjects-items {
  display: grid;
  gap: 2px;
  margin-top: 14px;
}

.service-subjects-items article {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 30px;
  align-items: center;
  gap: 10px;
  padding: 10px 8px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-subjects-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 6px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 14px;
  font-weight: 600;
}

.service-subjects-items article > div:nth-child(2) {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-subjects-items strong {
  overflow: hidden;
  font-size: 13px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-subjects-items small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-subjects-items button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-subjects-items button:hover {
  background: var(--td-error-color-1);
  color: var(--td-error-color);
}

.service-subjects-state {
  padding: 48px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 21px;
  text-align: center;
}

.service-subjects-state.is-error {
  color: var(--td-error-color);
}

@media (max-width: 760px) {
  .service-subjects-page {
    padding: 14px 16px 28px;
  }

  .service-subjects-layout {
    grid-template-columns: 1fr;
  }
}
</style>
