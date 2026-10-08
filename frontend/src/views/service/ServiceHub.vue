<template>
  <div class="service-hub-page">
    <t-dialog
      v-model:visible="serviceInfoDialogVisible"
      :header="serviceInfoDialogMode === 'name' ? '修改服务名称' : '修改服务描述'"
      width="520px"
      :confirm-btn="{
        content: '保存',
        theme: 'primary',
        loading: serviceInfoSaving,
      }"
      :cancel-btn="{ content: '取消' }"
      @confirm="saveServiceInfo"
    >
      <div class="service-info-dialog">
        <t-input
          v-if="serviceInfoDialogMode === 'name'"
          v-model="serviceInfoNameDraft"
          :maxlength="255"
          placeholder="服务名称"
        />
        <t-textarea
          v-else
          v-model="serviceInfoDescriptionDraft"
          :maxlength="500"
          :autosize="{ minRows: 4, maxRows: 8 }"
          placeholder="说明这个服务负责什么、适合谁使用"
        />
        <p>
          {{ serviceInfoDialogMode === 'name'
            ? '名称会同步显示在服务列表与左侧导航。'
            : '描述会显示在服务列表卡片上，帮助成员快速判断服务边界。' }}
        </p>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="shareDialogVisible"
      header="分享服务"
      width="520px"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: serviceInfoSaving }"
      :cancel-btn="{ content: '取消' }"
      @confirm="saveServiceVisibility"
    >
      <div class="service-share-dialog">
        <button
          v-for="option in shareOptions"
          :key="option.value"
          type="button"
          :class="{ selected: shareVisibilityDraft === option.value }"
          @click="shareVisibilityDraft = option.value"
        >
          <span>
            <strong>{{ option.label }}</strong>
            <small>{{ option.description }}</small>
          </span>
          <t-icon :name="shareVisibilityDraft === option.value ? 'check-circle-filled' : 'circle'" />
        </button>
      </div>
    </t-dialog>


    <section v-if="view === 'list'" class="service-hub-list-view">
      <header class="service-hub-page-header">
        <div>
          <h1>{{ serviceHubState.mode === 'archived' ? '归档服务' : '服务' }}</h1>
          <p>每个服务一个空间，专家在里面干活</p>
        </div>
        <t-button
          v-if="serviceHubState.mode !== 'archived'"
          theme="primary"
          class="service-hub-primary-button"
          @click="openCreate()"
        >
          <template #icon><t-icon name="add" /></template>
          新建服务
        </t-button>
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
                <div class="service-hub-card-icon">
                  <t-icon :name="templateFor(service)?.icon || 'folder'" />
                </div>
                <strong>{{ service.name }}</strong>
                <t-dropdown trigger="click" placement="bottom-right" @click.stop>
                  <button type="button" class="service-hub-more" aria-label="更多操作" @click.stop>
                    <t-icon name="more" />
                  </button>
                  <template #dropdown>
                    <t-dropdown-menu>
                      <t-dropdown-item @click="openCreate(service)">复制配置创建</t-dropdown-item>
                      <t-dropdown-item @click="archiveService(service)">
                        {{ service.state === 'archived' ? '恢复服务' : '归档服务' }}
                      </t-dropdown-item>
                    </t-dropdown-menu>
                  </template>
                </t-dropdown>
              </div>
              <div class="service-hub-card-tags">
                <span class="service-hub-tag">{{ templateFor(service)?.name || '自定义服务' }}</span>
                <span class="service-hub-role">{{ service.role }}</span>
              </div>
              <p class="service-hub-card-description">{{ service.description || '还没有填写服务描述' }}</p>
              <div class="service-hub-card-meta">
                <span>{{ service.updatedLabel }}</span>
                <span>{{ service.members }} 位成员</span>
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
            <h2>{{ serviceHubState.mode === 'archived' ? '还没有归档服务' : archivedServices.length ? '服务都已归档' : '还没有服务' }}</h2>
            <p v-if="serviceHubState.mode === 'archived'">归档后的服务会保留历史内容，需要时可以恢复。</p>
            <p v-else-if="archivedServices.length">归档服务不会出现在工作导航中，可以进入归档列表恢复。</p>
            <p v-else>选一个模板开始，配置会自动带入；也可以从空白服务自己设置。</p>
          </div>
          <t-button v-if="serviceHubState.mode === 'archived'" variant="outline" @click="showActiveServices">返回我的服务</t-button>
          <t-button v-else-if="archivedServices.length" variant="outline" @click="showArchivedServices">查看归档服务</t-button>
          <t-button v-else variant="outline" @click="openCreate()">从空白创建</t-button>
        </div>

        <div v-if="serviceHubState.mode !== 'archived'" class="service-hub-section-head service-hub-template-head">
          <span class="service-hub-section-title">从模板创建</span>
          <t-input v-model="templateQuery" class="service-hub-search" size="small" placeholder="搜索模板">
            <template #prefix-icon><t-icon name="search" /></template>
          </t-input>
        </div>
        <div v-if="serviceHubState.mode !== 'archived'" class="service-hub-template-groups">
          <section v-for="group in filteredTemplateGroups" :key="group.type" class="service-hub-template-group">
            <div class="service-hub-template-group-label">{{ group.type }}</div>
            <div class="service-hub-grid">
              <button
                v-for="template in group.items"
                :key="template.id"
                type="button"
                class="service-hub-template-card"
                @click="openCreate(template)"
              >
                <div class="service-hub-template-top">
                  <div class="service-hub-template-icon"><t-icon :name="template.icon" /></div>
                  <strong>{{ template.name }}</strong>
                </div>
                <p>{{ template.description }}</p>
                <span>{{ template.subjectLabel || '服务主体' }} · 创建后可调整</span>
              </button>
            </div>
          </section>
        </div>
      </div>
    </section>

    <section v-else-if="view === 'create'" class="service-hub-create-view">
      <ServiceCreateDialog
        :visible="true"
        display-mode="page"
        :source="createSource"
        @update:visible="cancelCreate"
        @submit="submitForm"
      />
    </section>

    <section v-else class="service-hub-workspace-view">
      <header class="service-hub-space-head">
        <button type="button" class="service-hub-back-button" @click="backToList">
          <t-icon name="chevron-left" />
          返回
        </button>
        <span class="service-hub-space-identity">
          <span class="service-hub-space-icon" aria-hidden="true">
            <t-icon :name="templateFor(activeService)?.icon || 'folder'" />
          </span>
          <span class="service-hub-space-name">{{ activeSession?.title || '新的会话' }}</span>
        </span>
        <span class="service-hub-space-template">{{ activeService?.name }}</span>
        <div class="service-hub-space-actions">
          <button
            v-for="tool in headTools"
            :key="tool.key"
            type="button"
            class="service-hub-head-tool"
            :class="{ active: panel === tool.key }"
            @click="panel = panel === tool.key ? '' : tool.key"
          >
            <t-icon :name="tool.icon" />
            <span>{{ tool.label }}</span>
            <small>{{ tool.count }}</small>
          </button>
          <span v-if="activeService?.state !== 'active'" class="service-hub-active-state">
            {{ serviceStateLabel(activeService?.state) }}
          </span>
          <t-popup v-model="serviceMenuVisible" trigger="click" placement="bottom-right">
            <button type="button" class="service-hub-circle-button" title="更多操作" aria-label="更多操作">
              <t-icon name="more" />
            </button>
            <template #content>
              <div class="service-hub-space-menu">
                <template v-if="serviceMenuLevel === 'primary'">
                  <button type="button" @click="openServiceInfoDialog('name')">修改服务名称</button>
                  <button type="button" @click="openServiceInfoDialog('description')">修改服务描述</button>
                  <button type="button" @click="openShareDialog">分享服务</button>
                  <span class="service-hub-space-menu-divider" />
                  <button type="button" class="has-trailing" @click="serviceMenuLevel = 'more'">
                    更多
                    <t-icon name="chevron-right" />
                  </button>
                </template>
                <template v-else>
                  <button type="button" class="has-leading" @click="serviceMenuLevel = 'primary'">
                    <t-icon name="chevron-left" />
                    更多
                  </button>
                  <span class="service-hub-space-menu-divider" />
                  <button type="button" @click="openServicePage('overview')">服务概览</button>
                  <button type="button" @click="openServicePage('subjects')">服务对象</button>
                  <button type="button" disabled>动态</button>
                  <button type="button" @click="openServicePage('settings/basic')">服务设置</button>
                  <span class="service-hub-space-menu-divider" />
                  <button type="button" @click="toggleActiveServicePause">
                    {{ activeService?.state === 'paused' ? '恢复服务' : '暂停服务' }}
                  </button>
                  <button type="button" @click="archiveService(activeService)">
                    {{ activeService?.state === 'archived' ? '恢复归档服务' : '归档服务' }}
                  </button>
                  <button type="button" class="is-danger" @click="deleteActiveService">删除服务</button>
                </template>
              </div>
            </template>
          </t-popup>
        </div>
      </header>

      <div class="service-hub-workbench">
        <article class="service-hub-chat">
          <header class="service-hub-chat-head">
            <strong>{{ activeSession?.title || '开始一段新的工作' }}</strong>
            <span>{{ activeSession?.expert || '服务助理' }}</span>
          </header>
          <div class="service-hub-chat-body">
            <ChatView
              v-if="activeChatSessionId"
              :key="`${activeSession?.id}:${activeChatSessionId}`"
              ref="serviceChatViewRef"
              :session_id="activeChatSessionId"
              :service-id="activeService?.id || ''"
              :agent-id="serviceChatAgentId"
              :kb-ids="activeServiceKnowledgeBaseIds"
              embedded-input-placeholder="围绕当前服务整理摘要、话术和下一步"
              embedded-mode
              hosted-mode
            >
              <template #empty-suggestions>
                <div class="service-hub-chat-prompts">
                  <strong>从一段工作开始</strong>
                  <p>把要整理、分析或推进的事情告诉专家，工作过程与产物会持续沉淀在服务空间。</p>
                  <button
                    v-for="prompt in serviceChatPrompts"
                    :key="prompt"
                    type="button"
                    @click="sendServiceChatPrompt(prompt)"
                  >
                    {{ prompt }}
                  </button>
                </div>
              </template>
            </ChatView>
            <div v-else class="service-hub-chat-state">
              <t-icon :name="activeChatSessionLoading ? 'loading' : 'chat'" :class="{ 'is-loading': activeChatSessionLoading }" />
              <span>{{ activeChatSessionLoading ? '正在准备会话' : activeChatSessionError || '会话暂不可用' }}</span>
              <t-button v-if="activeChatSessionError" variant="text" theme="primary" size="small" @click="retryActiveChatSession">
                重试
              </t-button>
            </div>

            <aside v-if="panel" class="service-hub-side-panel">
              <header>
                <strong>{{ panelTitle }}</strong>
                <button type="button" class="service-hub-side-panel-close" aria-label="关闭面板" @click="panel = ''"><t-icon name="close" /></button>
              </header>
              <div class="service-hub-side-panel-body">
                <template v-if="panel === 'reminders'">
                  <div v-if="remindersLoading" class="service-hub-panel-empty">正在加载服务待办。</div>
                  <div v-else-if="remindersError" class="service-hub-panel-empty service-hub-panel-error">
                    {{ remindersError }}
                  </div>
                  <template v-else>
                    <form class="service-hub-reminder-form" @submit.prevent="createActiveReminder">
                      <t-input v-model="reminderDraftTitle" size="small" placeholder="要跟进什么，例如：回访会员家庭 A" />
                      <t-select
                        v-model="reminderDraftAssignees"
                        size="small"
                        multiple
                        clearable
                        :options="reminderAssigneeOptions"
                        placeholder="选择负责人"
                      />
                      <t-input v-model="reminderDraftDueText" size="small" placeholder="时间，例如：本周五前" />
                      <t-button type="submit" block theme="primary" size="small" :loading="remindersSaving">
                        新建待办
                      </t-button>
                    </form>
                    <div v-if="serviceReminders.length" class="service-hub-reminder-list">
                      <article v-for="reminder in serviceReminders" :key="reminder.id" class="service-hub-reminder-item">
                        <div class="service-hub-reminder-item-head">
                          <strong>{{ reminder.title }}</strong>
                          <span :class="`service-hub-reminder-priority is-${reminder.priority}`">
                            {{ reminderPriorityLabel(reminder.priority) }}
                          </span>
                        </div>
                        <p v-if="reminder.summary">{{ reminder.summary }}</p>
                        <div class="service-hub-reminder-meta">
                          <span>{{ reminderStatusLabel(reminder.status) }}</span>
                          <span v-if="reminder.due_text">{{ reminder.due_text }}</span>
                          <span v-if="reminder.parent_reminder_id">下级待办 · {{ reminder.depth || 1 }} 层</span>
                        </div>
                        <div class="service-hub-reminder-actions">
                          <button type="button" @click="openReminderCollaboration(reminder)">
                            {{ selectedReminder?.id === reminder.id ? '收起详情' : '查看详情' }}
                          </button>
                          <t-dropdown
                            v-if="nextReminderStatuses(reminder).length"
                            trigger="click"
                            placement="bottom-left"
                          >
                            <button type="button">下一步 <t-icon name="chevron-down" /></button>
                            <template #dropdown>
                              <t-dropdown-menu>
                                <t-dropdown-item
                                  v-for="nextStatus in nextReminderStatuses(reminder)"
                                  :key="nextStatus.id"
                                  @click="changeReminderStatus(reminder, nextStatus.status_key)"
                                >
                                  {{ nextStatus.label }}
                                </t-dropdown-item>
                              </t-dropdown-menu>
                            </template>
                          </t-dropdown>
                          <button type="button" class="is-danger" @click="removeActiveReminder(reminder)">删除</button>
                        </div>
                        <div
                          v-if="selectedReminder?.id === reminder.id"
                          class="service-reminder-inline"
                        >
                          <div v-if="reminderCollaborationLoading" class="service-hub-panel-empty">
                            正在加载待办详情
                          </div>
                          <template v-else>
                            <section>
                              <div class="service-reminder-inline-head">
                                <strong>负责人</strong>
                                <button type="button" :disabled="reminderCollaborationSaving" @click="saveReminderAssignees">
                                  保存
                                </button>
                              </div>
                              <t-select
                                v-model="reminderAssigneeDraft"
                                multiple
                                clearable
                                size="small"
                                :options="reminderAssigneeOptions"
                                placeholder="选择负责人"
                              />
                            </section>
                            <section>
                              <strong>评论</strong>
                              <div v-if="reminderComments.length" class="service-reminder-comment-list">
                                <article v-for="comment in reminderComments" :key="comment.id">
                                  <div>
                                    <strong>{{ serviceMemberName(comment.user_id) }}</strong>
                                    <small>{{ formatSourceDate(comment.created_at) }}</small>
                                  </div>
                                  <p>{{ comment.content }}</p>
                                </article>
                              </div>
                              <div v-else class="service-reminder-inline-empty">还没有评论</div>
                              <div class="service-reminder-comment-compose">
                                <t-textarea
                                  v-model="reminderCommentDraft"
                                  :autosize="{ minRows: 2, maxRows: 4 }"
                                  placeholder="记录处理意见或协作结果"
                                />
                                <button type="button" :disabled="reminderCollaborationSaving" @click="submitReminderComment">
                                  添加评论
                                </button>
                              </div>
                            </section>
                            <section>
                              <strong>操作历史</strong>
                              <div v-if="reminderHistory.length" class="service-reminder-history-list">
                                <div v-for="history in reminderHistory" :key="history.id">
                                  <span>{{ history.action }}</span>
                                  <small>{{ serviceMemberName(history.user_id) }} · {{ formatSourceDate(history.created_at) }}</small>
                                </div>
                              </div>
                              <div v-else class="service-reminder-inline-empty">暂无操作历史</div>
                            </section>
                          </template>
                        </div>
                      </article>
                    </div>
                    <div v-else class="service-hub-panel-empty">当前服务还没有待办。可以先记录一个需要跟进的动作。</div>
                    <p class="service-reminder-status-note">
                      待办按工作指令自动使用待处理、进行中、已完成和已取消等状态。
                    </p>
                  </template>
                </template>
                <template v-else>
                  <div v-if="artifactsLoading" class="service-hub-panel-empty">正在加载服务产物。</div>
                  <div v-else-if="artifactsError" class="service-hub-panel-empty service-hub-panel-error">{{ artifactsError }}</div>
                  <button
                    v-for="artifact in activeArtifacts"
                    :key="artifact.id"
                    type="button"
                    class="service-hub-side-artifact"
                    :class="{ active: selectedArtifact?.id === artifact.id }"
                    @click="selectedArtifact = artifact"
                  >
                    <span class="service-hub-artifact-icon">{{ artifact.format || artifact.kind }}</span>
                    <span>
                      <strong>{{ artifact.title || artifact.original_name || '未命名产物' }}</strong>
                      <small>v{{ artifact.version }} · {{ artifact.lifecycle }}</small>
                    </span>
                  </button>
                  <div v-if="!activeArtifacts.length && !artifactsLoading" class="service-hub-panel-empty">当前服务还没有产物。</div>
                  <div v-if="selectedArtifact" class="service-hub-artifact-detail">
                    <strong>{{ selectedArtifact.title || selectedArtifact.original_name || '未命名产物' }}</strong>
                    <p>{{ selectedArtifact.summary || '该产物暂无摘要。' }}</p>
                    <div class="service-hub-artifact-links">
                      <a :href="artifactStreamUrl(selectedArtifact, 'preview')" target="_blank" rel="noreferrer">预览</a>
                      <a v-if="selectedArtifact.downloadable" :href="artifactStreamUrl(selectedArtifact, 'download')" target="_blank" rel="noreferrer">下载</a>
                      <button
                        v-if="selectedArtifact.lifecycle === 'temporary'"
                        type="button"
                        @click="changeArtifactLifecycle(selectedArtifact, 'saved')"
                      >
                        保存
                      </button>
                      <button
                        v-if="selectedArtifact.lifecycle !== 'archived'"
                        type="button"
                        class="is-danger"
                        @click="changeArtifactLifecycle(selectedArtifact, 'archived')"
                      >
                        归档
                      </button>
                    </div>
                  </div>
                </template>
              </div>
            </aside>
          </div>
        </article>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import ChatView from '@/views/chat/index.vue'
import { BUILTIN_SMART_REASONING_ID } from '@/api/agent'
import { fetchAllTenantMembers, type TenantMember } from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import ServiceCreateDialog from './ServiceCreateDialog.vue'
import {
  applyServiceTemplate,
  createServiceSpace,
  importServiceOrganizeOutput,
  createServiceReminder,
  deleteServiceReminder,
  listServiceReminderAssignees,
  listServiceReminderComments,
  listServiceReminderHistory,
  addServiceReminderComment,
  replaceServiceReminderAssignees,
  listServiceReminderStatusTransitions,
  listServiceReminderStatuses,
  listServiceReminders,
  listServiceArtifacts,
  listServiceMembers,
  updateServiceArtifactLifecycle,
  updateServiceSpace,
  deleteServiceSpace,
  updateServiceReminder,
  setServiceSpaceState,
  type ServiceReminder,
  type ServiceReminderAssignee,
  type ServiceReminderComment,
  type ServiceReminderHistory,
  type ServiceReminderStatus,
  type ServiceReminderStatusTransition,
  type ServiceSpaceMember,
  type ServiceArtifact as ApiServiceArtifact,
} from '@/api/service'
import {
  SESSION_MUTATION_EVENT,
  type SessionMutationDetail,
} from '@/components/sessionMutations'
import {
  createServiceSession,
  getFirstServiceSession,
  getService,
  getServiceExpert,
  getServiceTemplate,
  getSession,
  loadServiceHub,
  serviceHubState,
  serviceTemplates,
  type ServiceRecord,
  type ServiceSession,
  type ServiceTemplate,
} from './serviceHubState'

type HubView = 'list' | 'create' | 'workspace'
type HubPanel = '' | 'artifacts' | 'reminders'
type SortMode = 'recent' | 'created' | 'name'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const serviceBasePath = computed(() => route.meta.mobileEntry ? '/mobile/service' : '/platform/service')
const serviceChatAgentId = BUILTIN_SMART_REASONING_ID

const view = ref<HubView>('list')
const serviceQuery = ref('')
const templateQuery = ref('')
const sortMode = ref<SortMode>('recent')
const createSource = ref<ServiceTemplate | ServiceRecord | null>(null)
const pendingContextSourceId = ref('')
const pendingContextSourceType = ref('')
const serviceInfoDialogVisible = ref(false)
const serviceInfoDialogMode = ref<'name' | 'description'>('name')
const serviceInfoNameDraft = ref('')
const serviceInfoDescriptionDraft = ref('')
const serviceInfoSaving = ref(false)
const shareDialogVisible = ref(false)
const shareVisibilityDraft = ref<'private' | 'tenant'>('private')
const selectedArtifact = ref<ApiServiceArtifact | null>(null)
const serviceArtifactsList = ref<ApiServiceArtifact[]>([])
const artifactsLoading = ref(false)
const artifactsError = ref('')
const panel = ref<HubPanel>('')
type ServiceChatViewExpose = {
  triggerSend?: (question: string) => void
}
const serviceChatViewRef = ref<ServiceChatViewExpose | null>(null)
const activeChatSessionLoadingId = ref('')
const activeChatSessionError = ref('')
const serviceReminders = ref<ServiceReminder[]>([])
const reminderStatuses = ref<ServiceReminderStatus[]>([])
const reminderTransitions = ref<ServiceReminderStatusTransition[]>([])
const remindersLoading = ref(false)
const remindersSaving = ref(false)
const remindersError = ref('')
const reminderDraftTitle = ref('')
const reminderDraftDueText = ref('')
const reminderDraftAssignees = ref<string[]>([])
const serviceMembers = ref<ServiceSpaceMember[]>([])
const tenantMembers = ref<TenantMember[]>([])
const selectedReminder = ref<ServiceReminder | null>(null)
const reminderAssignees = ref<ServiceReminderAssignee[]>([])
const reminderComments = ref<ServiceReminderComment[]>([])
const reminderHistory = ref<ServiceReminderHistory[]>([])
const reminderCollaborationLoading = ref(false)
const reminderCommentDraft = ref('')
const reminderAssigneeDraft = ref<string[]>([])
const reminderCollaborationSaving = ref(false)
const serviceMenuVisible = ref(false)
const serviceMenuLevel = ref<'primary' | 'more'>('primary')

const sortOptions = [
  { label: '按最近活动', value: 'recent' },
  { label: '按创建时间', value: 'created' },
  { label: '按名称', value: 'name' },
]
const shareOptions = [
  {
    value: 'private' as const,
    label: '仅自己可见',
    description: '只有服务拥有者可以进入和查看内容',
  },
  {
    value: 'tenant' as const,
    label: '当前空间成员可见',
    description: '当前工作空间内的成员都可以发现并访问这个服务',
  },
]

const activeService = computed(() => getService(serviceHubState.activeServiceId))
const activeSession = computed(() => getSession(serviceHubState.activeSessionId))
const activeChatSessionId = computed(() => activeSession.value?.chatSessionId || '')
const activeChatSessionLoading = computed(() => activeChatSessionLoadingId.value === activeSession.value?.id)
const activeServiceKnowledgeBaseIds = computed(() => activeService.value?.knowledgeBaseIds || [])
const archivedServices = computed(() =>
  serviceHubState.services.filter((service) => service.state === 'archived'),
)
const services = computed(() => {
  if (serviceHubState.mode === 'archived') return serviceHubState.services.filter((service) => service.state === 'archived')
  return serviceHubState.services.filter((service) => service.state !== 'archived')
})
const filteredServices = computed(() => {
  const query = serviceQuery.value.trim().toLowerCase()
  const rows = services.value.filter((service) => `${service.name} ${service.description}`.toLowerCase().includes(query))
  return [...rows].sort((a, b) => {
    if (sortMode.value === 'name') return a.name.localeCompare(b.name, 'zh-CN')
    if (sortMode.value === 'created') return b.createdAt - a.createdAt
    return b.updatedAt - a.updatedAt
  })
})
const filteredTemplateGroups = computed(() => {
  const query = templateQuery.value.trim().toLowerCase()
  const groups = new Map<string, ServiceTemplate[]>()
  serviceTemplates
    .filter((template) => `${template.name} ${template.description} ${template.spaceTypeLabel || ''} ${template.subjectLabel || ''}`
      .toLowerCase()
      .includes(query))
    .forEach((template) => {
      const type = template.spaceTypeLabel || '其他'
      const items = groups.get(type) || []
      items.push(template)
      groups.set(type, items)
    })
  return Array.from(groups, ([type, items]) => ({ type, items }))
})
const serviceChatPrompts = computed(() => {
  const service = activeService.value
  const template = templateFor(service)
  const subject = template?.subjectLabel || '当前工作'
  const instructionFocus = (service?.instruction || '')
    .split(/[。；\n]/)
    .map((item) => item.trim())
    .find((item) => item.length >= 6 && item.length <= 34)
  return [
    `先帮我梳理${subject}现在最需要推进的事项`,
    instructionFocus
      ? `按“${instructionFocus}”检查已有资料并给出下一步`
      : '检查已有资料，指出风险、缺口和下一步动作',
    '把这次工作整理成一份可以直接执行的清单',
  ]
})
const activeArtifacts = computed(() => {
  return serviceArtifactsList.value
})
const reminderStatusMap = computed(() => new Map(reminderStatuses.value.map((item) => [item.id, item])))
const reminderStatusByKey = computed(() => new Map(reminderStatuses.value.map((item) => [item.status_key, item])))
const templateFor = (service: ServiceRecord | undefined) => getServiceTemplate(service)
const panelTitle = computed(() => panel.value === 'reminders' ? '服务待办' : '产物')
const headTools = computed(() => [
  { key: 'artifacts' as const, label: '产物', icon: 'file', count: activeArtifacts.value.length },
  { key: 'reminders' as const, label: '待办', icon: 'check-circle', count: serviceReminders.value.length },
])
const reminderAssigneeOptions = computed(() => {
  const byUserId = new Map(tenantMembers.value.map((member) => [member.user_id, member]))
  return serviceMembers.value
    .filter((member) => member.status === 'active')
    .map((member) => {
      const directoryMember = byUserId.get(member.user_id)
      return {
        value: member.user_id,
        label: directoryMember?.username || directoryMember?.email || '服务成员',
      }
    })
})
const serviceMemberName = (userId: string) => {
  const member = tenantMembers.value.find((item) => item.user_id === userId)
  return member?.username || member?.email || '服务成员'
}
const serviceStateLabel = (state?: ServiceRecord['state']) => {
  switch (state) {
    case 'active':
      return '进行中'
    case 'paused':
      return '已暂停'
    case 'archived':
      return '已归档'
    case 'draft':
      return '草稿'
    default:
      return '服务'
  }
}

const artifactStreamUrl = (artifact: ApiServiceArtifact, mode: 'preview' | 'download') =>
  `/api/v1/services/${encodeURIComponent(artifact.service_id)}/artifacts/${encodeURIComponent(artifact.artifact_id)}/${mode}`

const createLocalIdempotencyKey = () =>
  typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `artifact-${Date.now()}-${Math.random().toString(16).slice(2)}`

const loadArtifacts = async (serviceId: string) => {
  artifactsLoading.value = true
  artifactsError.value = ''
  try {
    const response = await listServiceArtifacts(serviceId, { page: 1, page_size: 100 })
    serviceArtifactsList.value = response?.data?.items || []
    selectedArtifact.value = serviceArtifactsList.value[0] || null
  } catch (error) {
    console.error('[ServiceHub] Failed to load service artifacts:', error)
    serviceArtifactsList.value = []
    selectedArtifact.value = null
    artifactsError.value = '服务产物暂不可用，请稍后重试'
  } finally {
    artifactsLoading.value = false
  }
}

const changeArtifactLifecycle = async (
  artifact: ApiServiceArtifact,
  lifecycle: ApiServiceArtifact['lifecycle'],
) => {
  try {
    const response = await updateServiceArtifactLifecycle(artifact.service_id, artifact.artifact_id, {
      lifecycle,
      idempotency_key: createLocalIdempotencyKey(),
    })
    if (response?.data) {
      const index = serviceArtifactsList.value.findIndex((item) => item.id === response.data.id)
      if (index >= 0) serviceArtifactsList.value[index] = response.data
      selectedArtifact.value = response.data
      MessagePlugin.success(lifecycle === 'archived' ? '产物已归档' : '产物已保存')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to update artifact lifecycle:', error)
    MessagePlugin.error('产物状态更新失败，请稍后重试')
  }
}

const closeServiceMenu = () => {
  serviceMenuVisible.value = false
  serviceMenuLevel.value = 'primary'
}

const openServicePage = async (suffix: 'overview' | 'subjects' | 'settings/basic') => {
  const serviceId = activeService.value?.id
  if (!serviceId) return
  closeServiceMenu()
  panel.value = ''
  await router.push(`${serviceBasePath.value}/${encodeURIComponent(serviceId)}/${suffix}`)
}

const openServiceInfoDialog = (mode: 'name' | 'description') => {
  const service = activeService.value
  if (!service) return
  closeServiceMenu()
  serviceInfoDialogMode.value = mode
  serviceInfoNameDraft.value = service.name
  serviceInfoDescriptionDraft.value = service.description || ''
  serviceInfoDialogVisible.value = true
}

const saveServiceInfo = async () => {
  const service = activeService.value
  if (!service || serviceInfoSaving.value) return
  const name = serviceInfoNameDraft.value.trim()
  if (serviceInfoDialogMode.value === 'name' && !name) {
    MessagePlugin.warning('服务名称不能为空')
    return
  }
  serviceInfoSaving.value = true
  try {
    const response = await updateServiceSpace(service.id, serviceInfoDialogMode.value === 'name'
      ? { name }
      : { description: serviceInfoDescriptionDraft.value.trim() })
    if (response?.data) {
      service.name = response.data.name
      service.description = response.data.description || ''
    }
    serviceInfoDialogVisible.value = false
    MessagePlugin.success('服务信息已更新')
  } catch (error) {
    console.error('[ServiceHub] Failed to update service info:', error)
    MessagePlugin.error('服务信息保存失败')
  } finally {
    serviceInfoSaving.value = false
  }
}

const openShareDialog = () => {
  if (!activeService.value) return
  closeServiceMenu()
  shareVisibilityDraft.value = activeService.value.visibility || 'private'
  shareDialogVisible.value = true
}

const saveServiceVisibility = async () => {
  const service = activeService.value
  if (!service || serviceInfoSaving.value) return
  serviceInfoSaving.value = true
  try {
    const response = await updateServiceSpace(service.id, { visibility: shareVisibilityDraft.value })
    service.visibility = response?.data?.visibility || shareVisibilityDraft.value
    shareDialogVisible.value = false
    MessagePlugin.success('分享范围已更新')
  } catch (error) {
    console.error('[ServiceHub] Failed to update service visibility:', error)
    MessagePlugin.error('分享范围保存失败')
  } finally {
    serviceInfoSaving.value = false
  }
}

const serviceWorkspacePath = (serviceId: string, sessionId?: string) => {
  if (route.meta.mobileEntry) return serviceBasePath.value
  const servicePath = `${serviceBasePath.value}/${encodeURIComponent(serviceId)}`
  return sessionId
    ? `${servicePath}/sessions/${encodeURIComponent(sessionId)}`
    : servicePath
}

const openCreate = async (source?: ServiceTemplate | ServiceRecord | null) => {
  createSource.value = source || null
  view.value = 'create'
  if (route.meta.mobileEntry) {
    await router.push({
      path: serviceBasePath.value,
      query: source
        ? ('templateId' in source ? { copy: source.id } : { template: source.id })
        : { create: '1' },
    })
    return
  }
  await router.push({
    path: `${serviceBasePath.value}/new`,
    query: source
      ? ('templateId' in source ? { copy: source.id } : { template: source.id })
      : {},
  })
}

const cancelCreate = async () => {
  createSource.value = null
  await router.push(serviceBasePath.value)
}

const showArchivedServices = () => {
  serviceHubState.mode = 'archived'
  serviceQuery.value = ''
}

const showActiveServices = () => {
  serviceHubState.mode = 'data'
  serviceQuery.value = ''
}

const createIdempotencyKey = () => {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `service-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

const openWorkspace = async (serviceId: string, sessionId?: string) => {
  await loadServiceHub()
  const service = getService(serviceId)
  if (!service) return
  serviceHubState.activeServiceId = serviceId
  const session = sessionId
    ? getSession(sessionId)
    : getFirstServiceSession(serviceId)
  const active = session || await ensureServiceSession(serviceId)
  if (!active) return
  serviceHubState.activeSessionId = active.id
  view.value = 'workspace'
  await router.replace({
    path: serviceWorkspacePath(serviceHubState.activeServiceId, serviceHubState.activeSessionId),
    query: route.meta.mobileEntry
      ? { service: serviceHubState.activeServiceId, session: serviceHubState.activeSessionId }
      : {},
  })
}

const loadReminders = async (serviceId: string) => {
  remindersLoading.value = true
  remindersError.value = ''
  try {
    const [reminderResponse, statusResponse, transitionResponse, memberResponse] = await Promise.all([
      listServiceReminders(serviceId, { page: 1, page_size: 100 }),
      listServiceReminderStatuses(serviceId),
      listServiceReminderStatusTransitions(serviceId),
      listServiceMembers(serviceId),
    ])
    serviceReminders.value = reminderResponse?.data?.items || []
    reminderStatuses.value = statusResponse?.data || []
    reminderTransitions.value = transitionResponse?.data || []
    serviceMembers.value = memberResponse?.data || []
    const tenantId = Number(authStore.currentTenantId || 0)
    tenantMembers.value = tenantId > 0 ? await fetchAllTenantMembers(tenantId) : []
  } catch (error) {
    console.error('[ServiceHub] Failed to load service reminders:', error)
    serviceReminders.value = []
    reminderStatuses.value = []
    reminderTransitions.value = []
    serviceMembers.value = []
    tenantMembers.value = []
    remindersError.value = '服务待办暂不可用，请稍后重试'
  } finally {
    remindersLoading.value = false
  }
}

const reminderStatusLabel = (status: string) =>
  reminderStatusByKey.value.get(status)?.label || status || '待处理'

const reminderPriorityLabel = (priority: ServiceReminder['priority']) => {
  switch (priority) {
    case 'high':
      return '高'
    case 'low':
      return '低'
    default:
      return '普通'
  }
}

const nextReminderStatuses = (reminder: ServiceReminder) => {
  const current = reminderStatusByKey.value.get(reminder.status)
  if (!current) return []
  return reminderTransitions.value
    .filter((transition) => transition.from_status_id === current.id && transition.enabled)
    .map((transition) => reminderStatusMap.value.get(transition.to_status_id))
    .filter((status): status is ServiceReminderStatus => Boolean(status && status.enabled))
}

const createActiveReminder = async () => {
  const serviceId = activeService.value?.id
  const title = reminderDraftTitle.value.trim()
  if (!serviceId || !title) {
    MessagePlugin.warning('请填写待办标题')
    return
  }
  remindersSaving.value = true
  try {
    const response = await createServiceReminder(serviceId, {
      title,
      priority: 'medium',
      due_text: reminderDraftDueText.value.trim(),
      assignee_user_ids: reminderDraftAssignees.value,
    })
    if (response?.data) {
      serviceReminders.value = [response.data, ...serviceReminders.value]
      reminderDraftTitle.value = ''
      reminderDraftDueText.value = ''
      reminderDraftAssignees.value = []
      MessagePlugin.success('待办已创建')
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to create service reminder:', error)
    MessagePlugin.error('待办创建失败，请稍后重试')
  } finally {
    remindersSaving.value = false
  }
}

const changeReminderStatus = async (reminder: ServiceReminder, status: string) => {
  const serviceId = activeService.value?.id
  if (!serviceId || !status || status === reminder.status) return
  try {
    const response = await updateServiceReminder(serviceId, reminder.id, { status })
    if (response?.data) {
      const index = serviceReminders.value.findIndex((item) => item.id === reminder.id)
      if (index >= 0) serviceReminders.value[index] = response.data
      if (selectedReminder.value?.id === reminder.id) selectedReminder.value = response.data
      MessagePlugin.success(`待办已更新为${reminderStatusLabel(status)}`)
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to update service reminder:', error)
    MessagePlugin.error('待办状态更新失败，请检查状态流转规则')
  }
}

const removeActiveReminder = async (reminder: ServiceReminder) => {
  const serviceId = activeService.value?.id
  if (!serviceId) return
  try {
    await deleteServiceReminder(serviceId, reminder.id)
    serviceReminders.value = serviceReminders.value.filter((item) => item.id !== reminder.id)
    if (selectedReminder.value?.id === reminder.id) {
      selectedReminder.value = null
      reminderAssignees.value = []
      reminderComments.value = []
      reminderHistory.value = []
      reminderAssigneeDraft.value = []
    }
    MessagePlugin.success('待办已删除')
  } catch (error) {
    console.error('[ServiceHub] Failed to delete service reminder:', error)
    MessagePlugin.error('待办删除失败，请稍后重试')
  }
}

const openReminderCollaboration = async (reminder: ServiceReminder) => {
  const serviceId = activeService.value?.id
  if (!serviceId) return
  if (selectedReminder.value?.id === reminder.id) {
    selectedReminder.value = null
    reminderAssignees.value = []
    reminderComments.value = []
    reminderHistory.value = []
    reminderAssigneeDraft.value = []
    reminderCommentDraft.value = ''
    return
  }
  selectedReminder.value = reminder
  reminderCollaborationLoading.value = true
  reminderCommentDraft.value = ''
  try {
    const [assignees, comments, history] = await Promise.all([
      listServiceReminderAssignees(serviceId, reminder.id),
      listServiceReminderComments(serviceId, reminder.id),
      listServiceReminderHistory(serviceId, reminder.id),
    ])
    reminderAssignees.value = assignees?.data || []
    reminderComments.value = comments?.data || []
    reminderHistory.value = history?.data || []
    reminderAssigneeDraft.value = reminderAssignees.value.map((item) => item.user_id)
  } catch (error) {
    console.error('[ServiceHub] Failed to load reminder collaboration:', error)
    MessagePlugin.error('待办协作信息加载失败')
  } finally {
    reminderCollaborationLoading.value = false
  }
}

const saveReminderAssignees = async () => {
  const serviceId = activeService.value?.id
  const reminderId = selectedReminder.value?.id
  if (!serviceId || !reminderId) return
  reminderCollaborationSaving.value = true
  try {
    const response = await replaceServiceReminderAssignees(
      serviceId,
      reminderId,
      reminderAssigneeDraft.value,
    )
    reminderAssignees.value = response?.data || []
    MessagePlugin.success('负责人已更新')
  } catch (error) {
    console.error('[ServiceHub] Failed to save reminder assignees:', error)
    MessagePlugin.error('负责人更新失败')
  } finally {
    reminderCollaborationSaving.value = false
  }
}

const submitReminderComment = async () => {
  const serviceId = activeService.value?.id
  const reminderId = selectedReminder.value?.id
  const content = reminderCommentDraft.value.trim()
  if (!serviceId || !reminderId || !content) {
    MessagePlugin.warning('请先填写评论内容')
    return
  }
  reminderCollaborationSaving.value = true
  try {
    const response = await addServiceReminderComment(serviceId, reminderId, content)
    if (response?.data) reminderComments.value.push(response.data)
    reminderCommentDraft.value = ''
    const history = await listServiceReminderHistory(serviceId, reminderId)
    reminderHistory.value = history?.data || reminderHistory.value
    MessagePlugin.success('评论已添加')
  } catch (error) {
    console.error('[ServiceHub] Failed to add reminder comment:', error)
    MessagePlugin.error('评论添加失败')
  } finally {
    reminderCollaborationSaving.value = false
  }
}

const formatSourceDate = (value?: string) => {
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

const toggleActiveServicePause = async () => {
  const service = activeService.value
  if (!service) return
  closeServiceMenu()
  const nextState = service.state === 'paused' ? 'active' : 'paused'
  try {
    await setServiceSpaceState(service.id, nextState)
    service.state = nextState
    MessagePlugin.success(nextState === 'paused' ? '服务已暂停' : '服务已恢复')
  } catch (error) {
    console.error('[ServiceHub] Failed to toggle service pause:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}

const deleteActiveService = async () => {
  const service = activeService.value
  if (!service) return
  closeServiceMenu()
  try {
    await deleteServiceSpace(service.id)
    await loadServiceHub(true)
    MessagePlugin.success('服务已删除')
    await backToList()
  } catch (error) {
    console.error('[ServiceHub] Failed to delete service:', error)
    MessagePlugin.error('服务删除失败')
  }
}

const backToList = async () => {
  closeServiceMenu()
  view.value = 'list'
  panel.value = ''
  selectedArtifact.value = null
  serviceHubState.activeServiceId = ''
  serviceHubState.activeSessionId = ''
  await router.push(serviceBasePath.value)
}

const syncFromRoute = async () => {
  const routeName = String(route.name || '')
  const routeService = typeof route.params.serviceId === 'string' ? route.params.serviceId : ''
  const routeSessionId = typeof route.params.sessionId === 'string' ? route.params.sessionId : ''
  const queryService = routeService || (typeof route.query.service === 'string' ? route.query.service : '')
  const querySession = routeSessionId || (typeof route.query.session === 'string' ? route.query.session : '')
  const queryCreate = route.query.create === '1'
  const queryTemplate = typeof route.query.template === 'string' ? route.query.template : ''
  const queryCopy = typeof route.query.copy === 'string' ? route.query.copy : ''
  const querySourceType = typeof route.query.source_type === 'string' ? route.query.source_type : ''
  const querySourceId = typeof route.query.source_id === 'string' ? route.query.source_id : ''
  const isCreateRoute = routeName === 'serviceCreate' || queryCreate || Boolean(queryTemplate || queryCopy)
  if (isCreateRoute) view.value = 'create'

  await loadServiceHub()
  if (isCreateRoute) {
    createSource.value = queryCopy
      ? getService(queryCopy) || null
      : serviceTemplates.find((template) => template.id === queryTemplate) || null
    if (querySourceType === 'organize_output' && querySourceId) {
      pendingContextSourceType.value = querySourceType
      pendingContextSourceId.value = querySourceId
    }
    return
  }
  if (queryService && getService(queryService)) {
    serviceHubState.activeServiceId = queryService
    const routeSession = querySession ? getSession(querySession) : undefined
    const session = routeSession?.serviceId === queryService
      ? routeSession
      : getFirstServiceSession(queryService) || await ensureServiceSession(queryService)
    if (!session) return
    serviceHubState.activeSessionId = session.id
    view.value = 'workspace'
    serviceHubState.expandedServices = Object.fromEntries(
      serviceHubState.services.map((service) => [service.id, service.id === queryService]),
    )
    if (!route.meta.mobileEntry && !routeService) {
      await router.replace({
        path: serviceWorkspacePath(queryService, session.id),
      })
    }
    return
  }
  createSource.value = null
  view.value = 'list'
}

type CreateServicePayload = {
  name: string
  description: string
  instruction: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
}

const serviceExpertsForPayload = (payload: CreateServicePayload) => payload.expertIds.length
  ? payload.expertIds.map((expertId, index) => {
      const expert = getServiceExpert(expertId)
      return {
        expert_ref: expertId,
        expert_name: expert?.name || expertId,
        expert_domain: expert?.domain,
        display_order: index,
        enabled: true,
      }
    })
  : [{
      expert_ref: BUILTIN_SMART_REASONING_ID,
      expert_name: '服务助理',
      display_order: 0,
      enabled: true,
    }]

const importPendingContextSource = async (serviceId: string) => {
  if (
    route.meta.mobileEntry
    || pendingContextSourceType.value !== 'organize_output'
    || !pendingContextSourceId.value
  ) {
    return false
  }
  try {
    await importServiceOrganizeOutput(serviceId, pendingContextSourceId.value)
    pendingContextSourceType.value = ''
    pendingContextSourceId.value = ''
    return true
  } catch (error) {
    console.error('[ServiceHub] Failed to import pending context source:', error)
    MessagePlugin.error('服务已创建，但整理结果带入失败，请从整理详情页重试')
    return false
  }
}

const openCreatedService = async (serviceId: string, message = '服务已创建') => {
  await loadServiceHub(true)
  const session = await ensureServiceSession(serviceId)
  await importPendingContextSource(serviceId)
  serviceHubState.activeServiceId = serviceId
  serviceHubState.activeSessionId = session.id
  createSource.value = null
  view.value = 'workspace'
  await router.replace({
    path: serviceWorkspacePath(serviceId, session.id),
    query: route.meta.mobileEntry ? { service: serviceId, session: session.id } : {},
  })
  MessagePlugin.success(message)
}

const createServiceDirectly = async (payload: CreateServicePayload) => {
  const response = await createServiceSpace({
    name: payload.name.trim(),
    description: payload.description.trim(),
    instruction: payload.instruction.trim(),
    template_key: payload.templateId || undefined,
    knowledge_base_ids: payload.knowledgeBaseIds,
    selected_skills: [],
    activate: true,
    experts: serviceExpertsForPayload(payload),
  })
  const serviceId = response?.data?.id
  if (!serviceId) throw new Error('missing service id')
  await openCreatedService(serviceId)
}

const persistService = async (payload: CreateServicePayload) => {
  const name = payload.name.trim()
  if (!name) {
    MessagePlugin.error('请输入服务名称')
    return
  }
  const normalizedPayload = { ...payload, name }
  try {
    const selectedTemplate = serviceTemplates.find((template) => template.id === payload.templateId)
    if (!selectedTemplate?.autoApply) {
      await createServiceDirectly(normalizedPayload)
      return
    }

    const response = await applyServiceTemplate(selectedTemplate.id, {
      name,
      description: payload.description.trim(),
      instruction: payload.instruction.trim(),
      template_version: 1,
      knowledge_base_ids: payload.knowledgeBaseIds,
      experts: serviceExpertsForPayload(payload),
      idempotency_key: createIdempotencyKey(),
    })
    const serviceId = response?.data?.id
    if (!serviceId) throw new Error('missing service id')
    await openCreatedService(serviceId, '已按审核模板创建服务')
  } catch (error) {
    console.error('[ServiceHub] Failed to create service:', error)
    const message = error && typeof error === 'object' && 'message' in error
      ? String((error as { message?: unknown }).message || '')
      : error instanceof Error
        ? error.message
        : ''
    MessagePlugin.error(message ? `服务创建失败：${message}` : '服务创建失败，请稍后重试')
  }
}

const submitForm = async (payload: {
  name: string
  description: string
  instruction: string
  templateId: string
  expertIds: string[]
  knowledgeBaseIds: string[]
}) => {
  await persistService(payload as CreateServicePayload)
}

const serviceSessionRequests = new Map<string, Promise<ServiceSession>>()

const ensureServiceSession = async (serviceId: string) => {
  const existing = getFirstServiceSession(serviceId)
  if (existing) return existing
  const pending = serviceSessionRequests.get(serviceId)
  if (pending) return pending
  const request = createServiceSession(serviceId, {
    title: '开始一段新的工作',
  }).finally(() => {
    serviceSessionRequests.delete(serviceId)
  })
  serviceSessionRequests.set(serviceId, request)
  return request
}

const archiveService = async (service: ServiceRecord | undefined, navigate = view.value === 'workspace') => {
  if (!service) return
  closeServiceMenu()
  const nextState = service.state === 'archived' ? 'active' : 'archived'
  try {
    await setServiceSpaceState(service.id, nextState)
    service.state = nextState
    MessagePlugin.success(nextState === 'archived' ? '服务已归档' : '服务已恢复')
    if (navigate || (view.value === 'workspace' && nextState === 'archived')) {
      await backToList()
    }
  } catch (error) {
    console.error('[ServiceHub] Failed to change service state:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}

const chatSessionRequests = new Map<string, Promise<string>>()

const ensureChatSession = async (session = activeSession.value) => {
  if (!session) return ''
  if (session.chatSessionId) return session.chatSessionId
  const pending = chatSessionRequests.get(session.id)
  if (pending) return pending

  activeChatSessionLoadingId.value = session.id
  activeChatSessionError.value = ''
  const request = (async () => {
    try {
      const chatSessionId = session.chatSessionId || session.id
      session.chatSessionId = chatSessionId
      return session.chatSessionId
    } catch (error) {
      console.error('[ServiceHub] Failed to create chat session:', error)
      activeChatSessionError.value = '会话创建失败，请稍后重试'
      return ''
    } finally {
      chatSessionRequests.delete(session.id)
      if (activeChatSessionLoadingId.value === session.id) {
        activeChatSessionLoadingId.value = ''
      }
    }
  })()
  chatSessionRequests.set(session.id, request)
  return request
}

const retryActiveChatSession = () => {
  activeChatSessionError.value = ''
  void ensureChatSession()
}

const sendServiceChatPrompt = (prompt: string) => {
  serviceChatViewRef.value?.triggerSend?.(prompt)
}

const handleServiceSessionMutation = (event: Event) => {
  const detail = (event as CustomEvent<SessionMutationDetail>).detail
  if (!detail?.sessionId) return
  const session = serviceHubState.sessions.find((item) => item.chatSessionId === detail.sessionId)
  if (!session) return

  if (detail.patch?.title) {
    session.title = detail.patch.title
  }
  if (typeof detail.patch?.is_pinned === 'boolean') {
    session.pinned = detail.patch.is_pinned
  }
  if (detail.removed) {
    session.chatSessionId = undefined
    if (session.id === serviceHubState.activeSessionId) {
      void ensureChatSession(session)
    }
  }
}

watch(panel, (value) => {
  if (value === 'reminders' && activeService.value?.id) {
    void loadReminders(activeService.value.id)
  }
  if (value === 'artifacts' && activeService.value?.id) {
    void loadArtifacts(activeService.value.id)
  }
})

watch(
  () => activeService.value?.id,
  (serviceId) => {
    serviceReminders.value = []
    reminderStatuses.value = []
    reminderTransitions.value = []
    remindersError.value = ''
    selectedReminder.value = null
    reminderAssignees.value = []
    reminderComments.value = []
    reminderHistory.value = []
    reminderAssigneeDraft.value = []
    reminderCommentDraft.value = ''
    serviceArtifactsList.value = []
    selectedArtifact.value = null
    artifactsError.value = ''
    if (serviceId) {
      void loadReminders(serviceId)
      void loadArtifacts(serviceId)
    }
  },
)

onMounted(async () => {
  window.addEventListener(SESSION_MUTATION_EVENT, handleServiceSessionMutation)
  await loadServiceHub()
  await syncFromRoute()
})
onUnmounted(() => {
  window.removeEventListener(SESSION_MUTATION_EVENT, handleServiceSessionMutation)
})

watch(
  () => [
    route.name,
    route.params.serviceId,
    route.params.sessionId,
    route.query.service,
    route.query.session,
    route.query.create,
    route.query.template,
    route.query.copy,
    route.query.source_type,
    route.query.source_id,
  ],
  () => syncFromRoute(),
  { immediate: true },
)

watch(
  activeSession,
  (session) => {
    if (session) void ensureChatSession(session)
  },
  { immediate: true },
)

watch(serviceMenuVisible, (visible) => {
  if (!visible) serviceMenuLevel.value = 'primary'
})
</script>

<style scoped lang="less">
.service-hub-page {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-hub-list-view,
.service-hub-create-view,
.service-hub-workspace-view {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  box-sizing: border-box;
}

.service-hub-list-view,
.service-hub-create-view {
  overflow-y: auto;
  padding: 20px 28px 32px;
}

.service-hub-list-view {
  padding-right: 28px;
}

.service-hub-create-view {
  padding: 0;
}

.service-info-dialog > p {
  margin: 10px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 19px;
}

.service-share-dialog {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-share-dialog > button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 16px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.service-share-dialog > button:hover,
.service-share-dialog > button.selected {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
}

.service-share-dialog > button > span {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 3px;
}

.service-share-dialog strong {
  font-size: 13px;
  font-weight: 500;
}

.service-share-dialog small {
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.service-share-dialog .t-icon {
  flex: none;
  color: var(--td-brand-color);
  font-size: 18px;
}

.service-hub-page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  min-height: 60px;
  margin-bottom: 16px;

  h1 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 21px;
    font-weight: 500;
    line-height: 30px;
    letter-spacing: 0;
  }

  p {
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.service-hub-primary-button {
  background: var(--td-brand-color);
  border: 0;
  color: var(--td-text-color-anti);
}

.service-hub-primary-button:hover {
  background: var(--td-brand-color-hover);
}

.service-hub-list-main {
  min-width: 0;
}

.service-hub-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 10px;
}

.service-hub-section-title {
  font-size: 14px;
  font-weight: 500;
  line-height: 20px;
}

.service-hub-section-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-sort {
  width: 132px;
}

.service-hub-search {
  width: 180px;
}

.service-hub-template-head {
  margin-top: 26px;
}

.service-hub-template-groups {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.service-hub-template-group-label {
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 500;
}

.service-hub-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.service-hub-card,
.service-hub-template-card {
  min-width: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  cursor: pointer;
  transition: border-color 0.16s ease, box-shadow 0.16s ease;
}

.service-hub-card {
  min-height: 137px;
  padding: 10px 11px;
}

.service-hub-card:hover,
.service-hub-template-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: 0 3px 12px rgba(0, 0, 0, 0.04);
}

.service-hub-card-top,
.service-hub-template-top {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 7px;
}

.service-hub-card-top strong,
.service-hub-template-top strong {
  min-width: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 500;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-card-icon,
.service-hub-template-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-card-icon {
  width: 24px;
  height: 24px;
}

.service-hub-template-icon {
  width: 22px;
  height: 22px;
}

.service-hub-more {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  width: 24px;
  height: 24px;
  margin-left: auto;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-more:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.service-hub-card-tags {
  display: flex;
  align-items: center;
  gap: 5px;
  margin-top: 8px;
}

.service-hub-tag,
.service-hub-role {
  display: inline-flex;
  align-items: center;
  min-height: 18px;
  padding: 1px 5px;
  border-radius: 3px;
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.service-hub-tag {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-role {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
}

.service-hub-card-description {
  display: -webkit-box;
  min-height: 34px;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.service-hub-card-meta {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-template-card {
  display: flex;
  width: 100%;
  min-height: 108px;
  padding: 11px 12px;
  flex-direction: column;
  font: inherit;
  text-align: left;
}

.service-hub-template-card p {
  margin: 7px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
}

.service-hub-template-card > span {
  display: block;
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
}

.service-hub-no-result {
  padding: 32px 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  text-align: center;
}

.service-hub-empty-state {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 20px;
  margin-top: 6px;
  padding: 20px 22px;
  border: 1px dashed var(--td-component-border);
  border-radius: 12px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-empty-state h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-empty-state p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;
}

.service-hub-create-view {
  max-width: none;
}

.service-hub-create-header {
  margin-bottom: 22px;
}

.service-hub-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-hub-form-panel {
  padding: 20px 22px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
}

.service-hub-field {
  margin-bottom: 16px;
}

.service-hub-field:last-child {
  margin-bottom: 0;
}

.service-hub-field > label {
  display: block;
  margin-bottom: 6px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}

.service-hub-field > label em {
  color: var(--td-error-color);
  font-style: normal;
}

.service-hub-count {
  margin-top: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-align: right;
}

.service-hub-error {
  display: block;
  margin-top: 5px;
  color: var(--td-error-color);
  font-size: 12px;
}

.service-hub-field-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 12px;
}

.service-hub-field-head h2 {
  margin: 0;
  font-size: 14px;
  font-weight: 500;
}

.service-hub-field-head p {
  margin: 4px 0 0;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-picker {
  margin-bottom: 12px;
  padding: 8px;
  border: 1px solid var(--td-component-border);
  border-radius: 6px;
}

.service-hub-picker-option,
.service-hub-choice-row {
  display: flex;
  align-items: center;
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: var(--app-font-family);
  text-align: left;
}

.service-hub-picker-option {
  justify-content: space-between;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 6px;
}

.service-hub-picker-option:hover,
.service-hub-picker-option.selected,
.service-hub-choice-row:hover,
.service-hub-choice-row.selected {
  background: var(--td-brand-color-1);
}

.service-hub-picker-option > span,
.service-hub-choice-row > span:last-child {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;
}

.service-hub-picker-option strong,
.service-hub-choice-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-picker-option small,
.service-hub-choice-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 17px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-list,
.service-hub-choice-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-hub-expert-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-expert-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 12px;
  font-weight: 600;
}

.service-hub-expert-row > div:nth-child(2) {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-expert-row strong {
  font-size: 13px;
  font-weight: 500;
}

.service-hub-expert-row small {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-expert-row button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-form-empty {
  padding: 10px 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-choice-row {
  gap: 10px;
  padding: 9px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
}

.service-hub-choice-check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  flex: none;
  border: 1px solid var(--td-component-border);
  border-radius: 3px;
  color: transparent;
}

.service-hub-choice-row.selected .service-hub-choice-check {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: var(--td-text-color-anti);
}

.service-hub-form-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 20px;
}

.service-hub-form-footer > div {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-form-footer span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.service-hub-workspace-view {
  padding: 0 24px 18px 28px;
  overflow: hidden;
}

.service-hub-space-head {
  display: flex;
  align-items: center;
  flex: none;
  min-width: 0;
  min-height: 64px;
  gap: 10px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-back-button {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 4px;
  min-height: 30px;
  padding: 5px 8px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  white-space: nowrap;
}

.service-hub-back-button:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-back-button .t-icon {
  font-size: 16px;
}

.service-hub-space-identity {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  gap: 8px;
}

.service-hub-space-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 7px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-space-name {
  max-width: 220px;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-space-template {
  max-width: 180px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-space-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  min-width: 0;
  margin-left: auto;
  gap: 6px;
}

.service-hub-head-tool {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  white-space: nowrap;
}

.service-hub-head-tool:hover,
.service-hub-head-tool.active {
  border-color: var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-brand-color-7);
}

.service-hub-head-tool .t-icon {
  font-size: 15px;
}

.service-hub-head-tool small {
  min-width: 16px;
  padding: 0 3px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 16px;
  text-align: center;
}

.service-hub-head-tool.active small {
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-active-state {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: 12px;
  background: var(--td-warning-color-1);
  color: var(--td-warning-color-7);
  font-size: 11px;
  line-height: 18px;
  white-space: nowrap;
}

.service-hub-active-state::before {
  width: 6px;
  height: 6px;
  margin-right: 5px;
  border-radius: 50%;
  background: currentColor;
  content: '';
}

.service-hub-circle-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  line-height: 1;
}

.service-hub-circle-button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-space-menu {
  display: flex;
  width: 184px;
  padding: 6px;
  flex-direction: column;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  box-shadow: var(--td-shadow-2);
}

.service-hub-space-menu button {
  display: flex;
  align-items: center;
  width: 100%;
  min-height: 32px;
  gap: 7px;
  padding: 5px 8px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  text-align: left;
}

.service-hub-space-menu button:hover {
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-space-menu button:disabled {
  color: var(--td-text-color-disabled);
  cursor: not-allowed;
}

.service-hub-space-menu button:disabled:hover {
  background: transparent;
}

.service-hub-space-menu button.is-danger {
  color: var(--td-error-color);
}

.service-hub-space-menu button.has-trailing {
  justify-content: space-between;
}

.service-hub-space-menu button.has-leading .t-icon {
  flex: none;
}

.service-hub-space-menu-divider {
  height: 1px;
  margin: 5px 3px;
  background: var(--td-component-stroke);
}

.service-hub-workspace-view > .service-hub-space-head + .service-hub-workbench {
  min-height: 0;
  margin-top: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.service-hub-workbench {
  display: flex;
  min-width: 0;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.service-hub-chat {
  display: flex;
  position: relative;
  flex: 1;
  min-width: 0;
  min-height: 0;
  flex-direction: column;
  overflow: hidden;
}

.service-hub-chat-head {
  display: flex;
  align-items: center;
  flex: none;
  min-height: 52px;
  gap: 8px;
  padding: 0 16px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-chat-head strong {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-chat-head span {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
}

.service-hub-chat-body {
  display: flex;
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.service-hub-chat-prompts {
  display: flex;
  max-width: 640px;
  margin: 84px auto 0;
  padding: 0 12px;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

.service-hub-chat-prompts > strong {
  color: var(--td-text-color-primary);
  font-size: 22px;
  font-weight: 600;
  line-height: 30px;
}

.service-hub-chat-prompts > p {
  max-width: 560px;
  margin: 0 0 4px;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 22px;
}

.service-hub-chat-prompts > button {
  display: inline-flex;
  align-items: center;
  min-height: 34px;
  padding: 6px 11px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  line-height: 20px;
  text-align: left;
}

.service-hub-chat-prompts > button:hover {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
}

.service-hub-chat-body :deep(.chat) {
  width: 100%;
  max-width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  padding: 0;
}

.service-hub-chat-body :deep(.chat_scroll_box) {
  padding: 12px 18px 0;
}

.service-hub-chat-body :deep(.msg_list) {
  max-width: 960px;
}

.service-hub-side-panel {
  display: flex;
  position: absolute;
  z-index: 12;
  top: 0;
  right: 0;
  bottom: 0;
  width: min(360px, calc(100% - 24px));
  min-width: 280px;
  flex-direction: column;
  border-left: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  box-shadow: -10px 0 24px rgba(0, 0, 0, 0.08);
}

.service-hub-side-panel > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 48px;
  padding: 0 14px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-hub-side-panel > header > strong {
  margin-right: auto;
  font-size: 13px;
  font-weight: 500;
}

.service-hub-side-panel-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-side-panel-close:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-primary);
}

.service-hub-side-panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
}

.service-hub-side-artifact {
  display: flex;
  align-items: center;
  width: 100%;
  gap: 8px;
  padding: 9px 8px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
}

.service-hub-side-artifact:hover,
.service-hub-side-artifact.active {
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-side-artifact > span:last-child {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
}

.service-hub-side-artifact strong,
.service-hub-side-artifact small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-side-artifact strong {
  font-size: 12px;
  font-weight: 500;
}

.service-hub-side-artifact small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-hub-artifact-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  flex: none;
  border-radius: 4px;
  background: var(--td-brand-color-1);
  color: var(--td-brand-color-7);
  font-size: 10px;
}

.service-hub-artifact-preview {
  margin: 10px 0 0;
  padding: 10px;
  overflow: auto;
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font: inherit;
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
}

.service-hub-context-source {
  padding: 11px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
}

.service-hub-context-source.is-highlighted {
  border-color: var(--td-brand-color-4);
  background: var(--td-brand-color-1);
  box-shadow: inset 3px 0 0 var(--td-brand-color);
}

.service-hub-context-source + .service-hub-context-source {
  margin-top: 8px;
}

.service-hub-context-source-head {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.service-hub-context-source-head strong {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-context-source-badge {
  flex: none;
  color: var(--td-brand-color-7);
  font-size: 10px;
  line-height: 18px;
}

.service-hub-context-source-remove {
  display: inline-flex;
  width: 22px;
  height: 22px;
  align-items: center;
  justify-content: center;
  flex: none;
  padding: 0;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.service-hub-context-source-remove:hover {
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-error-color);
}

.service-hub-context-source p {
  display: -webkit-box;
  margin: 7px 0 0;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.service-hub-context-source-meta {
  display: flex;
  gap: 8px;
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-context-source-actions {
  display: flex;
  gap: 10px;
  margin-top: 9px;
}

.service-hub-context-source-actions button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-hub-context-source-actions button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-reminder-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-reminder-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.service-hub-reminder-item {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
}

.service-hub-reminder-item-head,
.service-hub-reminder-meta,
.service-hub-reminder-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.service-hub-reminder-item-head {
  justify-content: space-between;
}

.service-hub-reminder-item-head strong {
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-reminder-item p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
  overflow-wrap: anywhere;
}

.service-hub-reminder-priority {
  flex: none;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-reminder-priority.is-high {
  color: var(--td-error-color);
}

.service-hub-reminder-priority.is-low {
  color: var(--td-text-color-placeholder);
}

.service-hub-reminder-meta {
  margin-top: 7px;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-hub-reminder-actions {
  flex-wrap: wrap;
  margin-top: 9px;
}

.service-hub-reminder-actions button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-hub-reminder-actions button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-reminder-actions button.is-danger {
  color: var(--td-error-color);
}

.service-reminder-inline {
  display: flex;
  margin-top: 10px;
  padding-top: 10px;
  flex-direction: column;
  gap: 14px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-reminder-inline > section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-reminder-inline > section > strong,
.service-reminder-inline-head > strong {
  color: var(--td-text-color-primary);
  font-size: 11px;
  font-weight: 600;
  line-height: 18px;
}

.service-reminder-inline-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.service-reminder-inline-head button,
.service-reminder-comment-compose button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-reminder-inline-head button:disabled,
.service-reminder-comment-compose button:disabled {
  color: var(--td-text-color-disabled);
  cursor: not-allowed;
}

.service-reminder-inline-empty {
  padding: 8px 0;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.service-reminder-status-note {
  margin: 12px 2px 0;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  line-height: 16px;
}

.service-reminder-collaboration {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.service-reminder-collaboration-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.service-reminder-collaboration-head > div {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 4px;
}

.service-reminder-collaboration-head strong {
  color: var(--td-text-color-primary);
  font-size: 15px;
  line-height: 22px;
  overflow-wrap: anywhere;
}

.service-reminder-collaboration-head span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-reminder-collaboration-section {
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.service-reminder-collaboration-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.service-reminder-collaboration-section-head strong,
.service-reminder-collaboration-section > strong {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
}

.service-reminder-collaboration-section button,
.service-reminder-comment-compose button,
.service-hub-status-add button,
.service-hub-status-row button,
.service-hub-artifact-links button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
}

.service-reminder-token-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.service-reminder-token-list span {
  padding: 3px 7px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 16px;
}

.service-reminder-comment-list,
.service-reminder-history-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-reminder-comment-list article,
.service-reminder-history-list > div {
  padding: 9px 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-reminder-comment-list article > div,
.service-reminder-history-list > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.service-reminder-comment-list article strong,
.service-reminder-history-list span {
  color: var(--td-text-color-primary);
  font-size: 11px;
  font-weight: 600;
}

.service-reminder-comment-list article small,
.service-reminder-history-list small {
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.service-reminder-comment-list article p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.service-reminder-comment-compose {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 7px;
}

.service-hub-settings-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.service-hub-settings-form > label {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.service-hub-settings-form > label > span,
.service-hub-settings-divider {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-settings-divider {
  padding-top: 4px;
  border-top: 1px solid var(--td-component-stroke);
}

.service-hub-status-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.service-hub-status-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto auto;
  align-items: center;
  gap: 7px;
}

.service-hub-status-row > span {
  max-width: 130px;
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-status-row > span.is-disabled {
  text-decoration: line-through;
}

.service-hub-status-row button.is-danger,
.service-hub-artifact-links button.is-danger {
  color: var(--td-error-color);
}

.service-hub-profile-field-list {
  display: flex;
  flex-direction: column;
  gap: 7px;
}

.service-hub-profile-field-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(80px, 0.8fr) auto auto;
  align-items: center;
  gap: 7px;
}

.service-hub-profile-field-row > span {
  overflow: hidden;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-profile-field-required {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 10px;
  white-space: nowrap;
}

.service-hub-profile-field-row button,
.service-hub-profile-field-add button {
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-brand-color);
  cursor: pointer;
  font: inherit;
  font-size: 11px;
  white-space: nowrap;
}

.service-hub-profile-field-row button.is-danger {
  color: var(--td-error-color);
}

.service-hub-profile-field-add {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 7px;
}

.service-hub-status-add {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 7px;
}

.service-hub-status-add .t-select,
.service-hub-status-add button {
  min-width: 0;
}

.service-hub-status-add button {
  min-height: 28px;
  padding: 0 8px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
}

.service-hub-artifact-detail {
  display: flex;
  flex-direction: column;
  gap: 7px;
  margin-top: 12px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-artifact-detail > strong {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-artifact-detail > p {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.service-hub-artifact-links {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.service-hub-artifact-links a {
  color: var(--td-brand-color);
  font-size: 11px;
  text-decoration: none;
}

.service-hub-artifact-links a:hover,
.service-reminder-collaboration-section button:hover,
.service-reminder-comment-compose button:hover,
.service-hub-status-add button:hover,
.service-hub-status-row button:hover,
.service-hub-artifact-links button:hover {
  color: var(--td-brand-color-7);
}

.service-hub-subject-form {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
}

.service-hub-subject-form-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 8px;
}

.service-hub-subject-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 12px;
}

.service-hub-subject-item {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
  background: var(--td-bg-color-container);
}

.service-hub-subject-item-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 8px;
}

.service-hub-subject-item-head strong {
  min-width: 0;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  overflow-wrap: anywhere;
}

.service-hub-subject-item-head span,
.service-hub-subject-item small {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-subject-item-head span {
  flex: none;
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-hub-subject-item small {
  display: block;
  margin-top: 4px;
  overflow-wrap: anywhere;
}

.service-hub-panel-empty {
  padding: 28px 10px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 20px;
  text-align: center;
}

.service-hub-panel-error {
  color: var(--td-error-color);
}

.service-hub-planning-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.service-hub-planning-row,
.service-hub-planning-section {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 5px;
}

.service-hub-planning-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.service-hub-planning-row span,
.service-hub-planning-section span {
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}

.service-hub-planning-row strong {
  max-width: 58%;
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 500;
  line-height: 18px;
  text-align: right;
  word-break: break-word;
}

.service-hub-planning-section > div {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px;
}

.service-hub-planning-section > div > strong {
  font-size: 12px;
  font-weight: 500;
}

.service-hub-planning-section p {
  margin: 8px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 19px;
}

.service-hub-chat-state {
  display: flex;
  flex: 1;
  align-items: center;
  justify-content: center;
  flex-direction: column;
  gap: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.service-hub-chat-state .t-icon {
  font-size: 24px;
}

.service-hub-chat-state .is-loading {
  animation: service-hub-spin 0.9s linear infinite;
}

@keyframes service-hub-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 1180px) {
  .service-hub-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 920px) {
  .service-hub-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .service-hub-space-head {
    flex-wrap: wrap;
    padding: 8px 0;
  }

  .service-hub-space-actions {
    width: 100%;
    margin-left: 0;
  }

  .service-hub-workspace-view > .service-hub-space-head + .service-hub-workbench {
    margin-top: 10px;
  }
}

@media (max-width: 680px) {
  .service-hub-list-view,
  .service-hub-workspace-view {
    padding: 16px;
  }

  .service-hub-create-view {
    padding: 0;
  }

  .service-hub-grid {
    grid-template-columns: 1fr;
  }

  .service-hub-space-template {
    display: none;
  }

  .service-hub-space-actions {
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: 2px;
  }

  .service-hub-head-tool {
    flex: none;
  }

  .service-hub-chat-prompts {
    margin-top: 48px;
  }

  .service-hub-chat-prompts > strong {
    font-size: 18px;
    line-height: 26px;
  }

  .service-hub-side-panel {
    top: auto;
    left: 0;
    width: 100%;
    min-width: 0;
    height: min(68vh, 560px);
    border-top: 1px solid var(--td-component-stroke);
    border-left: 0;
    box-shadow: 0 -10px 24px rgba(0, 0, 0, 0.1);
  }

  .service-hub-section-head,
  .service-hub-section-tools,
  .service-hub-form-footer {
    align-items: stretch;
    flex-direction: column;
  }

  .service-hub-search,
  .service-hub-sort {
    width: 100%;
  }

  .service-hub-form-footer > div {
    justify-content: flex-end;
  }
}
</style>
