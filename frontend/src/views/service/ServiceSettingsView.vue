<template>
  <main class="service-settings-page">
    <ServiceExpertPickerDialog
      v-model:visible="expertPickerVisible"
      :experts="serviceExperts"
      :selected-ids="selectedExpertIds"
      :loading="serviceExpertsLoading"
      :error="serviceExpertsError"
      @confirm="selectedExpertIds = $event"
      @retry="loadServiceExperts(true)"
    />

    <header class="service-settings-header">
      <button type="button" class="service-settings-back" @click="backToWorkspace">
        <t-icon name="chevron-left" />
        返回
      </button>
      <div>
        <h1>服务设置</h1>
        <p>{{ service?.name || '服务' }}</p>
      </div>
    </header>

    <div class="service-settings-layout">
      <nav class="service-settings-nav" aria-label="服务设置分区">
        <span>设置</span>
        <button :class="{ active: section === 'basic' }" @click="openSection('basic')">基本信息</button>
        <span>工作方式</span>
        <button :class="{ active: section === 'records' }" @click="openSection('records')">重点记录</button>
        <button :class="{ active: section === 'sources' }" @click="openSection('sources')">资料</button>
        <span>共享</span>
        <button :class="{ active: section === 'members' }" @click="openSection('members')">成员</button>
      </nav>

      <section class="service-settings-content">
        <div v-if="loading" class="service-settings-state">正在加载服务设置</div>
        <div v-else-if="error" class="service-settings-state is-error">{{ error }}</div>

        <template v-else-if="section === 'basic'">
          <header class="service-settings-section-head">
            <div>
              <h2>基本信息</h2>
              <p>名称和说明帮助成员识别服务，工作指令决定专家如何开展工作。</p>
            </div>
            <t-button theme="primary" :loading="saving" :disabled="!canManage" @click="saveBasic">
              保存
            </t-button>
          </header>

          <div class="service-settings-form">
            <label>
              <span>服务名称</span>
              <t-input v-model="nameDraft" :disabled="!canManage" :maxlength="255" />
            </label>
            <label>
              <span>服务说明</span>
              <t-textarea
                v-model="descriptionDraft"
                :disabled="!canManage"
                :maxlength="500"
                :autosize="{ minRows: 3, maxRows: 6 }"
              />
            </label>
            <label>
              <span>工作指令</span>
              <t-textarea
                v-model="instructionDraft"
                :disabled="!canManage"
                :maxlength="4000"
                :autosize="{ minRows: 6, maxRows: 12 }"
              />
            </label>
          </div>

          <div class="service-settings-experts">
            <div class="service-settings-subhead">
              <div>
                <strong>专家</strong>
                <small>服务中的会话会从这些专家中选择合适的能力。</small>
              </div>
              <t-button v-if="canManage" variant="outline" size="small" @click="expertPickerVisible = true">
                调整专家
              </t-button>
            </div>
            <div class="service-settings-expert-list">
              <article v-for="expert in selectedExperts" :key="expert.id">
                <AgentAvatar :name="expert.name" :avatar="expert.avatar" size="small" />
                <div>
                  <strong>{{ expert.name }}</strong>
                  <small>{{ expert.description || '服务专家' }}</small>
                </div>
              </article>
            </div>
          </div>
        </template>

        <template v-else-if="section === 'records'">
          <header class="service-settings-section-head">
            <div>
              <h2>重点记录</h2>
              <p>选择这个服务需要持续关注的信息。名称会直接影响专家识别和归纳。</p>
            </div>
            <t-button theme="primary" :loading="saving" :disabled="!canEdit" @click="saveRecords">
              保存
            </t-button>
          </header>

          <div class="service-record-recommendations">
            <span>推荐</span>
            <button
              v-for="label in recordRecommendations"
              :key="label"
              type="button"
              :disabled="!canEdit || hasRecord(label)"
              @click="addRecord(label)"
            >
              <t-icon :name="hasRecord(label) ? 'check' : 'add'" />
              {{ label }}
            </button>
          </div>

          <div class="service-record-layout">
            <div class="service-record-list">
              <article v-for="item in recordSchema" :key="item.key">
                <t-input v-model="item.label" :disabled="!canEdit" />
                <span v-if="item.source === 'instruction'">指令</span>
                <button v-if="canEdit" type="button" aria-label="移除重点记录" @click="removeRecord(item.key)">
                  <t-icon name="close" />
                </button>
              </article>
              <div v-if="!recordSchema.length" class="service-settings-state">还没有重点记录，可以从上方推荐中添加。</div>
            </div>
            <aside class="service-record-preview">
              <span>服务对象预览</span>
              <strong>{{ previewSubjectName }}</strong>
              <div v-for="item in recordSchema.slice(0, 6)" :key="item.key">
                <span>{{ item.label }}</span>
                <em>待记录</em>
              </div>
            </aside>
          </div>
        </template>

        <template v-else-if="section === 'sources'">
          <header class="service-settings-section-head">
            <div>
              <h2>资料</h2>
              <p>这些整理结果会作为服务的背景资料，在会话中按需引用。</p>
            </div>
          </header>
          <div v-if="contextSources.length" class="service-source-list">
            <article v-for="source in contextSources" :key="source.id">
              <div>
                <strong>{{ source.source_title || '未命名整理结果' }}</strong>
                <small>{{ formatDate(source.created_at) }}</small>
              </div>
              <button v-if="canEdit" type="button" @click="removeSource(source.id)">移除</button>
            </article>
          </div>
          <div v-else class="service-settings-state">当前服务还没有带入资料。</div>
        </template>

        <template v-else>
          <header class="service-settings-section-head">
            <div>
              <h2>成员</h2>
              <p>已加入 {{ serviceMembers.length }} / {{ service?.memberLimit || 20 }}</p>
            </div>
          </header>

          <div class="service-member-list">
            <article v-for="member in serviceMembers" :key="member.user_id">
              <AgentAvatar :name="memberName(member.user_id)" size="small" />
              <div>
                <strong>{{ memberName(member.user_id) }}</strong>
                <small>{{ memberEmail(member.user_id) }}</small>
              </div>
              <span v-if="member.role === 'owner'" class="service-member-owner">拥有者</span>
              <t-select
                v-else
                :model-value="member.role"
                :options="memberRoleOptions"
                size="small"
                :disabled="!canManage"
                @change="changeMemberRole(member, $event)"
              />
              <button
                v-if="member.role !== 'owner' && canManage"
                type="button"
                aria-label="移除成员"
                @click="removeMember(member)"
              >
                <t-icon name="delete" />
              </button>
              <span v-else class="service-member-action-placeholder" />
            </article>
          </div>

          <div v-if="canManage" class="service-member-add">
            <t-select
              v-model="memberToAdd"
              filterable
              clearable
              :options="availableMemberOptions"
              placeholder="选择当前工作空间成员"
            />
            <t-select v-model="memberRoleToAdd" :options="memberRoleOptions" />
            <t-button theme="primary" :loading="memberSaving" :disabled="!memberToAdd" @click="addMember">
              加入服务
            </t-button>
          </div>
        </template>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import AgentAvatar from '@/components/AgentAvatar.vue'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import {
  addServiceMember,
  deleteServiceContextSource,
  getServiceProfile,
  listServiceContextSources,
  listServiceExperts as listBoundServiceExperts,
  listServiceMembers,
  removeServiceMember,
  replaceServiceExperts,
  updateServiceMemberRole,
  updateServiceProfile,
  updateServiceSpace,
  type ServiceContextSource,
  type ServiceSpaceMember,
  type ServiceSpaceProfile,
  type ServiceSpaceProfileField,
} from '@/api/service'
import ServiceExpertPickerDialog from './ServiceExpertPickerDialog.vue'
import {
  getFirstServiceSession,
  getService,
  getServiceExpert,
  loadServiceExperts,
  loadServiceHub,
  serviceExperts,
  serviceExpertsError,
  serviceExpertsLoading,
} from './serviceHubState'

type SettingsSection = 'basic' | 'records' | 'sources' | 'members'
type EditableMemberRole = 'admin' | 'editor' | 'viewer'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const serviceId = computed(() => String(route.params.serviceId || ''))
const service = computed(() => getService(serviceId.value))
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const section = computed<SettingsSection>(() => {
  const value = String(route.params.section || 'basic')
  return ['basic', 'records', 'sources', 'members'].includes(value)
    ? value as SettingsSection
    : 'basic'
})
const canManage = computed(() => service.value?.role === '拥有者' || service.value?.role === '管理员')
const canEdit = computed(() => canManage.value || service.value?.role === '编辑者')

const loading = ref(true)
const saving = ref(false)
const error = ref('')
const nameDraft = ref('')
const descriptionDraft = ref('')
const instructionDraft = ref('')
const expertPickerVisible = ref(false)
const selectedExpertIds = ref<string[]>([])
const profile = ref<ServiceSpaceProfile | null>(null)
const recordSchema = ref<ServiceSpaceProfileField[]>([])
const contextSources = ref<ServiceContextSource[]>([])
const serviceMembers = ref<ServiceSpaceMember[]>([])
const tenantMembers = ref<TenantMember[]>([])
const memberToAdd = ref('')
const memberRoleToAdd = ref<EditableMemberRole>('editor')
const memberSaving = ref(false)

const memberRoleOptions = [
  { label: '管理员', value: 'admin' },
  { label: '可编辑', value: 'editor' },
  { label: '只看', value: 'viewer' },
]

const selectedExperts = computed(() =>
  selectedExpertIds.value.map((id) => getServiceExpert(id) || {
    id,
    name: '服务专家',
    description: '已加入当前服务',
    domain: '',
    skills: [],
  }))

const recordRecommendations = computed(() => {
  switch (service.value?.spaceType) {
    case 'operations':
      return ['当前进展', '负责人', '风险', '下一步动作', '完成时间']
    case 'research':
      return ['研究问题', '关键发现', '事实依据', '结论', '下一步验证']
    default:
      return ['当前阶段', '核心诉求', '沟通记录', '风险信号', '下一步动作']
  }
})
const previewSubjectName = computed(() => {
  switch (service.value?.spaceType) {
    case 'operations':
      return '示例运营事项'
    case 'research':
      return '示例研究课题'
    default:
      return '示例服务对象'
  }
})

const memberDirectory = computed(() =>
  new Map(tenantMembers.value.map((member) => [member.user_id, member])))

const availableMemberOptions = computed(() => {
  const joined = new Set(serviceMembers.value.map((member) => member.user_id))
  return tenantMembers.value
    .filter((member) => member.status === 'active' && !joined.has(member.user_id))
    .map((member) => ({
      label: member.username || member.email,
      value: member.user_id,
    }))
})

const memberName = (userId: string) =>
  memberDirectory.value.get(userId)?.username || memberDirectory.value.get(userId)?.email || '服务成员'

const memberEmail = (userId: string) =>
  memberDirectory.value.get(userId)?.email || '当前服务成员'

const loadSettings = async () => {
  if (!serviceId.value) return
  loading.value = true
  error.value = ''
  try {
    await Promise.all([loadServiceHub(), loadServiceExperts()])
    const currentService = getService(serviceId.value)
    if (!currentService) throw new Error('service not found')
    nameDraft.value = currentService.name
    descriptionDraft.value = currentService.description || ''
    instructionDraft.value = currentService.instruction || ''
    const [profileResult, sourceResult, memberResult, expertResult] = await Promise.all([
      getServiceProfile(serviceId.value),
      listServiceContextSources(serviceId.value),
      listServiceMembers(serviceId.value),
      listBoundServiceExperts(serviceId.value),
    ])
    profile.value = profileResult?.data || null
    recordSchema.value = (profile.value?.schema || []).map((item) => ({ ...item }))
    contextSources.value = sourceResult?.data || []
    serviceMembers.value = memberResult?.data || []
    selectedExpertIds.value = (expertResult?.data || []).map((item) => item.expert_ref)
    const tenantId = Number(authStore.currentTenantId || 0)
    tenantMembers.value = tenantId > 0 ? await fetchAllTenantMembers(tenantId) : []
  } catch (loadError) {
    console.error('[ServiceSettingsView] Failed to load settings:', loadError)
    error.value = '服务设置暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const openSection = (nextSection: SettingsSection) =>
  router.push(`${serviceBasePath.value}/${encodeURIComponent(serviceId.value)}/settings/${nextSection}`)

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

const saveBasic = async () => {
  const currentService = service.value
  const name = nameDraft.value.trim()
  if (!currentService || !canManage.value || !name || saving.value) {
    if (!name) MessagePlugin.warning('服务名称不能为空')
    return
  }
  saving.value = true
  try {
    const [serviceResponse] = await Promise.all([
      updateServiceSpace(serviceId.value, {
        name,
        description: descriptionDraft.value.trim(),
        instruction: instructionDraft.value.trim(),
      }),
      replaceServiceExperts(serviceId.value, selectedExpertIds.value.map((id, index) => {
        const expert = getServiceExpert(id)
        return {
          expert_ref: id,
          expert_name: expert?.name || '服务专家',
          expert_domain: expert?.domain,
          enabled: true,
          display_order: index,
        }
      })),
    ])
    if (serviceResponse?.data) {
      currentService.name = serviceResponse.data.name
      currentService.description = serviceResponse.data.description || ''
      currentService.instruction = serviceResponse.data.instruction || ''
      currentService.expertIds = [...selectedExpertIds.value]
    }
    MessagePlugin.success('基本信息已保存')
  } catch (saveError) {
    console.error('[ServiceSettingsView] Failed to save basics:', saveError)
    MessagePlugin.error('基本信息保存失败')
  } finally {
    saving.value = false
  }
}

const recordKey = () => `record_${Date.now().toString(36)}_${recordSchema.value.length + 1}`
const hasRecord = (label: string) => recordSchema.value.some((item) => item.label === label)

const addRecord = (label: string) => {
  if (hasRecord(label)) return
  recordSchema.value.push({
    key: recordKey(),
    label,
    value_type: 'text',
    source: 'manual',
    required: false,
    sensitive: false,
    display_order: recordSchema.value.length + 1,
  })
}

const removeRecord = (key: string) => {
  recordSchema.value = recordSchema.value
    .filter((item) => item.key !== key)
    .map((item, index) => ({ ...item, display_order: index + 1 }))
}

const saveRecords = async () => {
  if (!canEdit.value || saving.value) return
  const schema = recordSchema.value
    .map((item, index) => ({ ...item, label: item.label.trim(), display_order: index + 1, required: false }))
    .filter((item) => item.label)
  saving.value = true
  try {
    const response = await updateServiceProfile(serviceId.value, profile.value?.values || {}, schema)
    profile.value = response?.data || null
    recordSchema.value = (profile.value?.schema || schema).map((item) => ({ ...item }))
    MessagePlugin.success('重点记录已保存')
  } catch (saveError) {
    console.error('[ServiceSettingsView] Failed to save records:', saveError)
    MessagePlugin.error('重点记录保存失败')
  } finally {
    saving.value = false
  }
}

const removeSource = async (sourceId: string) => {
  try {
    await deleteServiceContextSource(serviceId.value, sourceId)
    contextSources.value = contextSources.value.filter((item) => item.id !== sourceId)
    MessagePlugin.success('资料已移除')
  } catch (removeError) {
    console.error('[ServiceSettingsView] Failed to remove source:', removeError)
    MessagePlugin.error('资料移除失败')
  }
}

const addMember = async () => {
  if (!memberToAdd.value || !canManage.value || memberSaving.value) return
  memberSaving.value = true
  try {
    const response = await addServiceMember(serviceId.value, {
      user_id: memberToAdd.value,
      role: memberRoleToAdd.value,
    })
    if (response?.data) serviceMembers.value.push(response.data)
    memberToAdd.value = ''
    MessagePlugin.success('成员已加入服务')
  } catch (addError) {
    console.error('[ServiceSettingsView] Failed to add member:', addError)
    MessagePlugin.error('成员加入失败')
  } finally {
    memberSaving.value = false
  }
}

const changeMemberRole = async (member: ServiceSpaceMember, value: unknown) => {
  const role = String(value) as EditableMemberRole
  if (!canManage.value || role === member.role || !['admin', 'editor', 'viewer'].includes(role)) return
  const previousRole = member.role
  member.role = role
  try {
    await updateServiceMemberRole(serviceId.value, member.user_id, role)
    MessagePlugin.success('成员角色已更新')
  } catch (updateError) {
    member.role = previousRole
    console.error('[ServiceSettingsView] Failed to update member role:', updateError)
    MessagePlugin.error('成员角色更新失败')
  }
}

const removeMember = (member: ServiceSpaceMember) => {
  const dialog = DialogPlugin.confirm({
    header: '移出服务',
    body: `确定将“${memberName(member.user_id)}”移出这个服务吗？`,
    confirmBtn: { content: '移出', theme: 'danger' },
    cancelBtn: '取消',
    onConfirm: async () => {
      dialog.setConfirmLoading(true)
      try {
        await removeServiceMember(serviceId.value, member.user_id)
        serviceMembers.value = serviceMembers.value.filter((item) => item.user_id !== member.user_id)
        dialog.destroy()
        MessagePlugin.success('成员已移出服务')
      } catch (removeError) {
        console.error('[ServiceSettingsView] Failed to remove member:', removeError)
        dialog.setConfirmLoading(false)
        MessagePlugin.error('成员移出失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

const formatDate = (value?: string) => {
  if (!value) return '刚刚带入'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚带入'
  return `${date.getMonth() + 1} 月 ${date.getDate()} 日`
}

watch(serviceId, loadSettings)
onMounted(loadSettings)
</script>

<style scoped lang="less">
.service-settings-page {
  width: 100%;
  min-width: 0;
  min-height: 100%;
  padding: 20px 28px 40px;
  overflow-y: auto;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-settings-header {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  align-items: center;
  max-width: 1120px;
  min-height: 54px;
  gap: 14px;
  margin: 0 auto 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-settings-header h1,
.service-settings-header p,
.service-settings-section-head h2,
.service-settings-section-head p {
  margin: 0;
}

.service-settings-header h1 {
  font-size: 20px;
  font-weight: 500;
}

.service-settings-header p {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-settings-back {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-settings-layout {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
  max-width: 1120px;
  min-height: 520px;
  gap: 28px;
  margin: 0 auto;
}

.service-settings-nav {
  display: flex;
  align-self: start;
  flex-direction: column;
  gap: 3px;
  padding: 4px;
}

.service-settings-nav > span {
  margin: 14px 8px 4px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-settings-nav > button {
  min-height: 34px;
  padding: 0 10px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  text-align: left;
}

.service-settings-nav > button:hover,
.service-settings-nav > button.active {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-settings-nav > button.active {
  box-shadow: inset 3px 0 0 var(--td-brand-color);
}

.service-settings-content {
  min-width: 0;
  padding: 4px 0;
}

.service-settings-section-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 18px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-settings-section-head h2 {
  font-size: 18px;
  font-weight: 500;
}

.service-settings-section-head p {
  margin-top: 5px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 19px;
}

.service-settings-form {
  display: grid;
  gap: 18px;
  max-width: 720px;
  padding: 22px 0;
}

.service-settings-form label {
  display: grid;
  gap: 7px;
}

.service-settings-form label > span {
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-settings-experts {
  max-width: 720px;
  padding-top: 18px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-settings-subhead {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.service-settings-subhead > div {
  display: grid;
  gap: 3px;
}

.service-settings-subhead strong {
  font-size: 13px;
  font-weight: 500;
}

.service-settings-subhead small {
  color: var(--td-text-color-secondary);
  font-size: 11px;
}

.service-settings-expert-list {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-top: 12px;
}

.service-settings-expert-list article {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 9px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-settings-expert-list article > div {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.service-settings-expert-list strong,
.service-member-list strong,
.service-source-list strong {
  overflow: hidden;
  font-size: 12px;
  font-weight: 500;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-settings-expert-list small,
.service-member-list small,
.service-source-list small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-record-recommendations {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  padding: 18px 0;
}

.service-record-recommendations > span {
  margin-right: 4px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-record-recommendations > button {
  display: inline-flex;
  align-items: center;
  min-height: 28px;
  gap: 4px;
  padding: 4px 9px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 14px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
}

.service-record-recommendations > button:disabled {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  cursor: default;
}

.service-record-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 260px;
  align-items: start;
  gap: 22px;
}

.service-record-list {
  display: grid;
  gap: 8px;
}

.service-record-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto 28px;
  align-items: center;
  gap: 8px;
}

.service-record-list article > span {
  padding: 2px 6px;
  border-radius: 3px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 10px;
}

.service-record-list article > button,
.service-member-list article > button {
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

.service-record-list article > button:hover,
.service-member-list article > button:hover {
  background: var(--td-error-color-1);
  color: var(--td-error-color);
}

.service-record-preview {
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-record-preview > span {
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-record-preview > strong {
  display: block;
  margin: 5px 0 12px;
  font-size: 15px;
  font-weight: 500;
}

.service-record-preview > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  padding: 7px 0;
  border-top: 1px solid var(--td-component-stroke);
  font-size: 11px;
}

.service-record-preview em {
  color: var(--td-text-color-placeholder);
  font-style: normal;
}

.service-source-list,
.service-member-list {
  display: grid;
  gap: 2px;
  margin-top: 16px;
}

.service-source-list article,
.service-member-list article {
  display: grid;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 8px 6px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-source-list article {
  grid-template-columns: minmax(0, 1fr) auto;
}

.service-source-list article > div,
.service-member-list article > div {
  display: grid;
  min-width: 0;
  gap: 2px;
}

.service-source-list button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-error-color);
  cursor: pointer;
}

.service-member-list article {
  grid-template-columns: 32px minmax(0, 1fr) 120px 28px;
}

.service-member-owner {
  justify-self: end;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.service-member-action-placeholder {
  width: 28px;
}

.service-member-add {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 140px auto;
  gap: 8px;
  margin-top: 18px;
  padding: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 7px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-settings-state {
  padding: 54px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  line-height: 21px;
  text-align: center;
}

.service-settings-state.is-error {
  color: var(--td-error-color);
}

@media (max-width: 820px) {
  .service-settings-page {
    padding: 14px 16px 28px;
  }

  .service-settings-layout {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .service-settings-nav {
    flex-direction: row;
    overflow-x: auto;
    padding-bottom: 8px;
  }

  .service-settings-nav > span {
    display: none;
  }

  .service-settings-nav > button {
    flex: none;
  }

  .service-record-layout,
  .service-settings-expert-list {
    grid-template-columns: 1fr;
  }

  .service-member-add {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .service-settings-section-head {
    align-items: stretch;
    flex-direction: column;
  }

  .service-member-list article {
    grid-template-columns: 32px minmax(0, 1fr) 96px 28px;
  }
}
</style>
