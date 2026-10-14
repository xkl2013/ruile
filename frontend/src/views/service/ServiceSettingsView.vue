<template>
  <main class="service-settings-page">
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
              <p>名称和一句话说明帮助成员识别服务，客户信息由后台维护。</p>
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
          </div>

          <div class="service-settings-customer">
            <div class="service-settings-subhead">
              <div>
                <strong>家长与学员</strong>
                <small>客户绑定与重点信息由后台配置，这里只读。</small>
              </div>
            </div>
            <div class="service-settings-customer-grid">
              <div><span>家庭</span><strong>{{ customerInfo.familyName }}</strong></div>
              <div><span>学员</span><strong>{{ customerInfo.student }}</strong></div>
              <div><span>班级</span><strong>{{ customerInfo.className }}</strong></div>
              <div><span>主班老师</span><strong>{{ customerInfo.teacher }}</strong></div>
            </div>
          </div>
        </template>

        <template v-else-if="section === 'records'">
          <header class="service-settings-section-head">
            <div>
              <h2>重点记录</h2>
              <p>选择这个服务需要持续关注的信息，名称会直接影响服务摘要和归纳。</p>
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
              <span>家长与学员预览</span>
              <strong>{{ customerInfo.familyName }}</strong>
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
  listServiceMembers,
  removeServiceMember,
  updateServiceMemberRole,
  updateServiceProfile,
  updateServiceSpace,
  type ServiceContextSource,
  type ServiceSpaceMember,
  type ServiceSpaceProfile,
  type ServiceSpaceProfileField,
} from '@/api/service'
import {
  getFirstServiceSession,
  getService,
  loadServiceHub,
} from './serviceHubState'
import { buildCustomerInfo } from './servicePresentation'

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

const customerInfo = computed(() =>
  service.value?.customer || buildCustomerInfo(null, null, service.value?.name || '当前家庭'))

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
    await loadServiceHub()
    const currentService = getService(serviceId.value)
    if (!currentService) throw new Error('service not found')
    nameDraft.value = currentService.name
    descriptionDraft.value = currentService.description || ''
    const [profileResult, sourceResult, memberResult] = await Promise.all([
      getServiceProfile(serviceId.value),
      listServiceContextSources(serviceId.value),
      listServiceMembers(serviceId.value),
    ])
    profile.value = profileResult?.data || null
    recordSchema.value = (profile.value?.schema || []).map((item) => ({ ...item }))
    contextSources.value = sourceResult?.data || []
    serviceMembers.value = memberResult?.data || []
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
    const serviceResponse = await updateServiceSpace(serviceId.value, {
      name,
      description: descriptionDraft.value.trim(),
    })
    if (serviceResponse?.data) {
      currentService.name = serviceResponse.data.name
      currentService.description = serviceResponse.data.description || ''
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

<style scoped lang="less" src="./serviceSettings.less"></style>
