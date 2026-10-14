<template>
  <div class="service-hub-side-panel-body">
    <div v-if="loading" class="service-hub-panel-empty">正在加载服务待办。</div>
    <div v-else-if="error" class="service-hub-panel-empty service-hub-panel-error">{{ error }}</div>
    <template v-else>
      <section v-if="pendingProposals.length" class="service-reminder-group">
        <div class="service-reminder-group-head">
          <strong>要你确认</strong>
          <span>{{ pendingProposals.length }}</span>
        </div>
        <div class="service-proposal-list">
          <ServiceFactProposalCard
            v-for="proposal in pendingProposals"
            :key="proposal.id"
            :service-id="serviceId"
            :proposal="proposal"
            @resolved="removeResolvedProposal"
          />
        </div>
      </section>

      <div class="service-reminder-group-head">
        <strong>要你去做</strong>
        <span>{{ reminders.length }}</span>
      </div>
      <form class="service-hub-reminder-form" @submit.prevent="createReminder">
        <t-input v-model="titleDraft" size="small" placeholder="要跟进什么，例如：回访会员家庭 A" />
        <t-select
          v-model="assigneeDraft"
          size="small"
          multiple
          clearable
          :options="assigneeOptions"
          placeholder="选择负责人"
        />
        <t-input v-model="dueDraft" size="small" placeholder="时间，例如：本周五前" />
        <t-button type="submit" block theme="primary" size="small" :loading="saving">
          新建待办
        </t-button>
      </form>

      <div v-if="reminders.length" class="service-hub-reminder-list">
        <article v-for="reminder in reminders" :key="reminder.id" class="service-hub-reminder-item">
          <div class="service-hub-reminder-item-head">
            <strong>{{ reminder.title }}</strong>
            <span :class="`service-hub-reminder-priority is-${reminder.priority}`">
              {{ priorityLabel(reminder.priority) }}
            </span>
          </div>
          <p v-if="reminder.summary">{{ reminder.summary }}</p>
          <div class="service-hub-reminder-meta">
            <span>{{ statusLabel(reminder.status) }}</span>
            <span v-if="reminder.due_text">{{ reminder.due_text }}</span>
            <span v-if="reminder.parent_reminder_id">下级待办 · {{ reminder.depth || 1 }} 层</span>
          </div>
          <div class="service-hub-reminder-actions">
            <button type="button" @click="toggleCollaboration(reminder)">
              {{ selectedReminder?.id === reminder.id ? '收起详情' : '查看详情' }}
            </button>
            <t-dropdown
              v-if="nextStatuses(reminder).length"
              trigger="click"
              placement="bottom-left"
            >
              <button type="button">下一步 <t-icon name="chevron-down" /></button>
              <template #dropdown>
                <t-dropdown-menu>
                  <t-dropdown-item
                    v-for="nextStatus in nextStatuses(reminder)"
                    :key="nextStatus.id"
                    @click="changeStatus(reminder, nextStatus.status_key)"
                  >
                    {{ nextStatus.label }}
                  </t-dropdown-item>
                </t-dropdown-menu>
              </template>
            </t-dropdown>
            <button type="button" class="is-danger" @click="removeReminder(reminder)">删除</button>
          </div>

          <div v-if="selectedReminder?.id === reminder.id" class="service-reminder-inline">
            <div v-if="collaborationLoading" class="service-hub-panel-empty">正在加载待办详情</div>
            <template v-else>
              <section>
                <div class="service-reminder-inline-head">
                  <strong>负责人</strong>
                  <button type="button" :disabled="collaborationSaving" @click="saveAssignees">
                    保存
                  </button>
                </div>
                <t-select
                  v-model="selectedAssigneeDraft"
                  multiple
                  clearable
                  size="small"
                  :options="assigneeOptions"
                  placeholder="选择负责人"
                />
              </section>
              <section>
                <strong>评论</strong>
                <div v-if="comments.length" class="service-reminder-comment-list">
                  <article v-for="comment in comments" :key="comment.id">
                    <div>
                      <strong>{{ memberName(comment.user_id) }}</strong>
                      <small>{{ formatDate(comment.created_at) }}</small>
                    </div>
                    <p>{{ comment.content }}</p>
                  </article>
                </div>
                <div v-else class="service-reminder-inline-empty">还没有评论</div>
                <div class="service-reminder-comment-compose">
                  <t-textarea
                    v-model="commentDraft"
                    :autosize="{ minRows: 2, maxRows: 4 }"
                    placeholder="记录处理意见或协作结果"
                  />
                  <button type="button" :disabled="collaborationSaving" @click="submitComment">
                    添加评论
                  </button>
                </div>
              </section>
              <section>
                <strong>操作历史</strong>
                <div v-if="history.length" class="service-reminder-history-list">
                  <div v-for="item in history" :key="item.id">
                    <span>{{ item.action }}</span>
                    <small>{{ memberName(item.user_id) }} · {{ formatDate(item.created_at) }}</small>
                  </div>
                </div>
                <div v-else class="service-reminder-inline-empty">暂无操作历史</div>
              </section>
            </template>
          </div>
        </article>
      </div>
      <div v-else-if="!pendingProposals.length" class="service-hub-panel-empty">
        当前服务还没有待办。可以先记录一个需要跟进的动作。
      </div>
      <p class="service-reminder-status-note">
        待办按服务配置自动使用待处理、进行中、已完成和已取消等状态。
      </p>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import {
  addServiceReminderComment,
  createServiceReminder,
  deleteServiceReminder,
  listServiceMembers,
  listServiceFactProposals,
  listServiceReminderAssignees,
  listServiceReminderComments,
  listServiceReminderHistory,
  listServiceReminders,
  listServiceReminderStatuses,
  listServiceReminderStatusTransitions,
  replaceServiceReminderAssignees,
  updateServiceReminder,
  type ServiceReminder,
  type ServiceReminderComment,
  type ServiceReminderHistory,
  type ServiceReminderStatus,
  type ServiceReminderStatusTransition,
  type ServiceSpaceMember,
  type ServiceFactProposal,
} from '@/api/service'
import ServiceFactProposalCard from '@/views/chat/components/ServiceFactProposalCard.vue'

const props = defineProps<{ serviceId: string }>()
const emit = defineEmits<{ 'count-change': [count: number] }>()
const authStore = useAuthStore()
const reminders = ref<ServiceReminder[]>([])
const pendingProposals = ref<ServiceFactProposal[]>([])
const statuses = ref<ServiceReminderStatus[]>([])
const transitions = ref<ServiceReminderStatusTransition[]>([])
const serviceMembers = ref<ServiceSpaceMember[]>([])
const tenantMembers = ref<TenantMember[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')
const titleDraft = ref('')
const dueDraft = ref('')
const assigneeDraft = ref<string[]>([])
const selectedReminder = ref<ServiceReminder | null>(null)
const selectedAssigneeDraft = ref<string[]>([])
const comments = ref<ServiceReminderComment[]>([])
const history = ref<ServiceReminderHistory[]>([])
const commentDraft = ref('')
const collaborationLoading = ref(false)
const collaborationSaving = ref(false)

const statusMap = computed(() => new Map(statuses.value.map((item) => [item.id, item])))
const statusByKey = computed(() => new Map(statuses.value.map((item) => [item.status_key, item])))
const assigneeOptions = computed(() => {
  const directory = new Map(tenantMembers.value.map((member) => [member.user_id, member]))
  return serviceMembers.value
    .filter((member) => member.status === 'active')
    .map((member) => {
      const user = directory.get(member.user_id)
      return {
        value: member.user_id,
        label: user?.username || user?.email || '服务成员',
      }
    })
})

const resetCollaboration = () => {
  selectedReminder.value = null
  selectedAssigneeDraft.value = []
  comments.value = []
  history.value = []
  commentDraft.value = ''
}

const loadReminders = async () => {
  if (!props.serviceId) return
  loading.value = true
  error.value = ''
  resetCollaboration()
  try {
    const [reminderResponse, proposalResponse, statusResponse, transitionResponse, memberResponse] = await Promise.all([
      listServiceReminders(props.serviceId, { page: 1, page_size: 100 }),
      listServiceFactProposals(props.serviceId, {
        page: 1,
        page_size: 100,
        status: 'pending',
      }),
      listServiceReminderStatuses(props.serviceId),
      listServiceReminderStatusTransitions(props.serviceId),
      listServiceMembers(props.serviceId),
    ])
    reminders.value = reminderResponse?.data?.items || []
    pendingProposals.value = proposalResponse?.data?.items || []
    statuses.value = statusResponse?.data || []
    transitions.value = transitionResponse?.data || []
    serviceMembers.value = memberResponse?.data || []
    const tenantId = Number(authStore.currentTenantId || 0)
    tenantMembers.value = tenantId > 0 ? await fetchAllTenantMembers(tenantId) : []
  } catch (loadError) {
    console.error('[ServiceReminderPanel] Failed to load service reminders:', loadError)
    reminders.value = []
    pendingProposals.value = []
    statuses.value = []
    transitions.value = []
    serviceMembers.value = []
    tenantMembers.value = []
    error.value = '服务待办暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const statusLabel = (status: string) => statusByKey.value.get(status)?.label || status || '待处理'
const priorityLabel = (priority: ServiceReminder['priority']) => {
  if (priority === 'high') return '高'
  if (priority === 'low') return '低'
  return '普通'
}
const nextStatuses = (reminder: ServiceReminder) => {
  const current = statusByKey.value.get(reminder.status)
  if (!current) return []
  return transitions.value
    .filter((transition) => transition.from_status_id === current.id && transition.enabled)
    .map((transition) => statusMap.value.get(transition.to_status_id))
    .filter((status): status is ServiceReminderStatus => Boolean(status?.enabled))
}
const memberName = (userId: string) => {
  const member = tenantMembers.value.find((item) => item.user_id === userId)
  return member?.username || member?.email || '服务成员'
}
const formatDate = (value?: string) => {
  if (!value) return '刚刚带入'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚带入'
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date).replace('/', '-')
}

const createReminder = async () => {
  const title = titleDraft.value.trim()
  if (!props.serviceId || !title) {
    MessagePlugin.warning('请填写待办标题')
    return
  }
  saving.value = true
  try {
    const response = await createServiceReminder(props.serviceId, {
      title,
      priority: 'medium',
      due_text: dueDraft.value.trim(),
      assignee_user_ids: assigneeDraft.value,
    })
    if (response?.data) reminders.value = [response.data, ...reminders.value]
    titleDraft.value = ''
    dueDraft.value = ''
    assigneeDraft.value = []
    MessagePlugin.success('待办已创建')
  } catch (createError) {
    console.error('[ServiceReminderPanel] Failed to create service reminder:', createError)
    MessagePlugin.error('待办创建失败，请稍后重试')
  } finally {
    saving.value = false
  }
}

const changeStatus = async (reminder: ServiceReminder, status: string) => {
  if (!status || status === reminder.status) return
  try {
    const response = await updateServiceReminder(props.serviceId, reminder.id, { status })
    if (response?.data) {
      const index = reminders.value.findIndex((item) => item.id === reminder.id)
      if (index >= 0) reminders.value[index] = response.data
      if (selectedReminder.value?.id === reminder.id) selectedReminder.value = response.data
      MessagePlugin.success(`待办已更新为${statusLabel(status)}`)
    }
  } catch (updateError) {
    console.error('[ServiceReminderPanel] Failed to update service reminder:', updateError)
    MessagePlugin.error('待办状态更新失败，请检查状态流转规则')
  }
}

const removeReminder = async (reminder: ServiceReminder) => {
  try {
    await deleteServiceReminder(props.serviceId, reminder.id)
    reminders.value = reminders.value.filter((item) => item.id !== reminder.id)
    if (selectedReminder.value?.id === reminder.id) resetCollaboration()
    MessagePlugin.success('待办已删除')
  } catch (removeError) {
    console.error('[ServiceReminderPanel] Failed to delete service reminder:', removeError)
    MessagePlugin.error('待办删除失败，请稍后重试')
  }
}

const toggleCollaboration = async (reminder: ServiceReminder) => {
  if (selectedReminder.value?.id === reminder.id) {
    resetCollaboration()
    return
  }
  selectedReminder.value = reminder
  collaborationLoading.value = true
  commentDraft.value = ''
  try {
    const [assigneeResponse, commentResponse, historyResponse] = await Promise.all([
      listServiceReminderAssignees(props.serviceId, reminder.id),
      listServiceReminderComments(props.serviceId, reminder.id),
      listServiceReminderHistory(props.serviceId, reminder.id),
    ])
    selectedAssigneeDraft.value = (assigneeResponse?.data || []).map((item) => item.user_id)
    comments.value = commentResponse?.data || []
    history.value = historyResponse?.data || []
  } catch (loadError) {
    console.error('[ServiceReminderPanel] Failed to load reminder collaboration:', loadError)
    MessagePlugin.error('待办协作信息加载失败')
  } finally {
    collaborationLoading.value = false
  }
}

const saveAssignees = async () => {
  if (!selectedReminder.value) return
  collaborationSaving.value = true
  try {
    await replaceServiceReminderAssignees(
      props.serviceId,
      selectedReminder.value.id,
      selectedAssigneeDraft.value,
    )
    MessagePlugin.success('负责人已更新')
  } catch (saveError) {
    console.error('[ServiceReminderPanel] Failed to save reminder assignees:', saveError)
    MessagePlugin.error('负责人更新失败')
  } finally {
    collaborationSaving.value = false
  }
}

const submitComment = async () => {
  const content = commentDraft.value.trim()
  if (!selectedReminder.value || !content) {
    MessagePlugin.warning('请先填写评论内容')
    return
  }
  collaborationSaving.value = true
  try {
    const response = await addServiceReminderComment(
      props.serviceId,
      selectedReminder.value.id,
      content,
    )
    if (response?.data) comments.value.push(response.data)
    commentDraft.value = ''
    const historyResponse = await listServiceReminderHistory(
      props.serviceId,
      selectedReminder.value.id,
    )
    history.value = historyResponse?.data || history.value
    MessagePlugin.success('评论已添加')
  } catch (saveError) {
    console.error('[ServiceReminderPanel] Failed to add reminder comment:', saveError)
    MessagePlugin.error('评论添加失败')
  } finally {
    collaborationSaving.value = false
  }
}

const removeResolvedProposal = (proposal: ServiceFactProposal) => {
  pendingProposals.value = pendingProposals.value.filter((item) => item.id !== proposal.id)
}

const handleServiceDataUpdated = (event: Event) => {
  const detail = (event as CustomEvent).detail
  if (detail?.serviceId === props.serviceId) {
    void loadReminders()
  }
}

watch(() => props.serviceId, loadReminders, { immediate: true })
watch(
  [reminders, pendingProposals],
  ([items, proposals]) => emit('count-change', items.length + proposals.length),
  { deep: true, immediate: true },
)
onMounted(() => window.addEventListener('service-space-data-updated', handleServiceDataUpdated))
onUnmounted(() => window.removeEventListener('service-space-data-updated', handleServiceDataUpdated))
</script>
