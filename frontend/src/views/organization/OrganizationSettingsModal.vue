<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="settings-overlay" @click.self="handleClose">
        <div class="settings-modal">
          <!-- 关闭按钮 -->
          <button class="close-btn" @click="handleClose" :aria-label="$t('common.close')">
            <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
              <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
            </svg>
          </button>

          <div class="settings-container">
            <!-- 左侧导航 -->
            <div class="settings-sidebar">
              <div class="sidebar-header">
                <h2 class="sidebar-title">{{ modalTitle }}</h2>
              </div>
              <div class="settings-nav">
                <div v-for="item in navItems" :key="item.key"
                  :class="['nav-item', { 'active': currentSection === item.key }]" @click="currentSection = item.key">
                  <img v-if="item.key === 'sharedAgents'"
                    :src="currentSection === 'sharedAgents' ? agentIconActiveSrc : agentIconSrc"
                    class="nav-icon nav-icon-img" alt="" aria-hidden="true" />
                  <t-icon v-else :name="item.icon" class="nav-icon" />
                  <span class="nav-label">{{ item.label }}</span>
                  <span
                    v-if="item.badge != null && (item.key === 'sharedKb' || item.key === 'sharedAgents' ? true : item.badge > 0)"
                    :class="['nav-item-badge', { 'nav-item-badge-count': item.key === 'sharedKb' || item.key === 'sharedAgents' }]">{{
                      item.badge }}</span>
                </div>
              </div>
            </div>

            <!-- 右侧内容区域 -->
            <div class="settings-content">
              <div class="content-wrapper">
                <!-- 组织管理员但空间角色不足，给出只读提示 -->
                <div v-if="showTenantRoleHint" class="tenant-role-hint">
                  <t-icon name="info-circle" size="16px" />
                  <span>{{ $t('organization.rbac.needTenantAdminTip') }}</span>
                </div>
                <!-- 基本信息 -->
                <div v-show="currentSection === 'basic'" class="section">
                  <div class="section-header">
                    <h2>{{ $t('organization.editor.basicTitle') }}</h2>
                    <p class="section-description">{{ $t('organization.editor.basicDesc') }}</p>
                  </div>

                  <div class="settings-group">
                    <!-- 空间名称与头像：一行展示，头像点击弹出 Emoji 选择 -->
                    <div class="setting-row">
                      <div class="setting-info">
                        <label>{{ $t('organization.name') }} <span class="required">*</span></label>
                        <p class="desc">{{ $t('organization.editor.nameTip') }}</p>
                      </div>
                      <div class="setting-control">
                        <div class="name-input-wrapper">
                          <t-popup v-model="avatarPopoverVisible" trigger="click" placement="bottom-left"
                            :disabled="!isAdmin" overlay-class-name="avatar-emoji-popover">
                            <div class="avatar-trigger-wrap">
                              <SpaceAvatar :name="formData.name || '?'" :avatar="formData.avatar" size="medium" />
                              <span v-if="isAdmin" class="avatar-change-hint">{{ $t('organization.avatar') }}</span>
                            </div>
                            <template #content>
                              <div class="avatar-popover-content" @click.stop>
                                <p class="avatar-popover-title">{{ $t('organization.avatarPickerHint') }}</p>
                                <div class="avatar-emoji-grid">
                                  <button v-for="emoji in avatarEmojiOptions" :key="emoji" type="button"
                                    class="avatar-emoji-btn"
                                    :class="{ 'is-selected': formData.avatar === 'emoji:' + emoji }"
                                    @click="selectAvatarEmoji(emoji)">
                                    {{ emoji }}
                                  </button>
                                </div>
                                <t-button v-if="formData.avatar" variant="text" size="small" class="avatar-clear-btn"
                                  @click="clearAvatarEmoji">
                                  {{ $t('organization.avatarClear') }}
                                </t-button>
                              </div>
                            </template>
                          </t-popup>
                          <t-input v-model="formData.name" :placeholder="$t('organization.namePlaceholder')"
                            :disabled="!isAdmin" class="name-input" />
                        </div>
                      </div>
                    </div>

                    <!-- 空间描述 -->
                    <div class="setting-row">
                      <div class="setting-info">
                        <label>{{ $t('organization.description') }}</label>
                        <p class="desc">{{ $t('organization.editor.descriptionTip') }}</p>
                      </div>
                      <div class="setting-control">
                        <t-textarea v-model="formData.description"
                          :placeholder="$t('organization.descriptionPlaceholder')"
                          :autosize="{ minRows: 3, maxRows: 6 }" :maxlength="500" :disabled="!isAdmin" />
                      </div>
                    </div>

                    <!-- 参与空间数量上限 (仅管理员可见) -->
                    <div v-if="isAdmin && orgId" class="setting-row setting-row-vertical">
                      <div class="setting-info full-width">
                        <label>{{ $t('organization.settings.memberLimit') }}</label>
                        <p class="desc">{{ $t('organization.settings.memberLimitDesc') }}</p>
                      </div>
                      <div class="setting-control full-width">
                        <div class="member-limit-input-row">
                          <t-input-number v-model="formData.member_limit" :min="0" :max="10000"
                            :placeholder="$t('organization.settings.memberLimitPlaceholder')" theme="normal"
                            style="width: 140px;" />
                          <span class="member-limit-hint">{{ $t('organization.settings.memberLimitHint', {
                            count:
                              orgInfo?.member_count
                              ?? 0
                          }) }}</span>
                        </div>
                      </div>
                    </div>


                  </div>
                </div>

                <!-- 参与空间管理 -->
                <div v-show="currentSection === 'members'" class="section">
                  <div class="section-header">
                    <h2>{{ $t('organization.manageMembers') }}</h2>
                    <p class="section-description">{{ $t('organization.settings.membersDesc') }}</p>
                  </div>

                  <div class="settings-group members-group">
                    <div class="members-header">
                      <div class="members-search">
                        <t-input v-model="memberSearchQuery" :placeholder="$t('common.search')" clearable>
                          <template #prefix-icon>
                            <t-icon name="search" />
                          </template>
                        </t-input>
                      </div>
                      <t-button v-if="isAdmin" variant="outline" size="small" @click="openAddMemberDialog">
                        <template #icon><t-icon name="user-add" /></template>
                        {{ $t('organization.addMember.button') }}
                      </t-button>
                    </div>

                    <t-loading :loading="membersLoading">
                      <div class="members-list">
                        <div v-for="member in filteredMembers" :key="member.id" class="member-item" :class="{
                          'is-owner': isOwnerMember(member),
                          'is-me': memberContainsCurrentUser(member)
                        }">
                          <div class="member-avatar" :class="{ 'is-me': memberContainsCurrentUser(member) }">
                            <img v-if="member.avatar" :src="member.avatar" alt="" />
                            <t-icon v-else name="user" size="20px" />
                          </div>
                          <div class="member-info">
                            <span class="member-name">
                              {{ memberPrimaryLabel(member) }}
                              <span v-if="memberContainsCurrentUser(member)" class="me-tag">{{ $t('common.me')
                              }}</span>
                            </span>
                            <span class="member-email">{{ memberSecondaryLabel(member) }}</span>
                          </div>
                          <div class="member-role">
                            <t-select v-if="isAdmin && !isOwnerMember(member)" v-model="member.role"
                              :options="roleOptions" size="small"
                              @change="(val: string) => handleRoleChange(member, val)" />
                            <t-tag v-else size="small" :theme="getRoleTheme(member.role)">
                              {{ $t(`organization.role.${member.role}`) }}
                              <span v-if="isOwnerMember(member)">({{ $t('organization.owner') }})</span>
                            </t-tag>
                          </div>
                          <div v-if="isAdmin && !isOwnerMember(member)" class="member-actions">
                            <t-popconfirm
                              :content="$t('organization.detail.removeMemberConfirm', { name: memberPrimaryLabel(member) })"
                              :confirm-btn="{ content: $t('common.confirm'), theme: 'danger' }"
                              :cancel-btn="{ content: $t('common.cancel') }"
                              placement="left"
                              @confirm="confirmRemoveMember(member)">
                              <t-button variant="text" theme="danger" size="small" @click.stop>
                                <t-icon name="delete" />
                              </t-button>
                            </t-popconfirm>
                          </div>
                        </div>
                        <div v-if="filteredMembers.length === 0" class="empty-members">
                          {{ $t('organization.noMembers') }}
                        </div>
                      </div>
                    </t-loading>
                  </div>
                </div>

                <!-- 共享知识库（独立侧边栏） -->
                <div v-show="currentSection === 'sharedKb'" class="section">
                  <div class="section-header">
                    <h2>{{ $t('organization.share.sharedKnowledgeBase') }}</h2>
                    <p class="section-description">{{ $t('organization.settings.sharedDesc') }}</p>
                    <p class="section-description permission-calc-hint">
                      <t-tooltip :content="$t('organization.settings.permissionCalcTip')" placement="top">
                        <span class="hint-inner">
                          <t-icon name="info-circle" size="14px" />
                          {{ $t('organization.settings.permissionCalcFormula') }}
                        </span>
                      </t-tooltip>
                    </p>
                  </div>
                  <div class="settings-group">
                    <t-loading :loading="sharesLoading">
                      <div v-if="sharedKnowledgeBases.length === 0 && !sharesLoading" class="empty-shared">
                        <div class="empty-icon">
                          <img src="@/assets/img/zhishiku.svg" class="empty-icon-kb" alt="" aria-hidden="true" />
                        </div>
                        <p class="empty-text">{{ $t('organization.settings.noSharedKB') }}</p>
                        <p class="empty-subtext">{{ $t('organization.settings.noSharedKBTip') }}</p>
                      </div>
                      <div v-else class="shared-list">
                        <div v-for="share in sharedKnowledgeBases" :key="share.id" class="shared-item"
                          @click="handleShareClick(share)">
                          <div class="shared-icon shared-icon-kb">
                            <img src="@/assets/img/zhishiku.svg" class="shared-icon-kb-img" alt="" aria-hidden="true" />
                          </div>
                          <div class="shared-info">
                            <span class="shared-name">{{ share.knowledge_base_name }}</span>
                            <div class="shared-meta">
                              <span v-if="share.shared_by_username" class="shared-by">
                                <t-icon name="user" size="12px" />
                                {{ share.shared_by_username }}
                              </span>
                              <span class="shared-time">
                                <t-icon name="time" size="12px" />
                                {{ formatDate(share.created_at) }}
                              </span>
                            </div>
                          </div>
                          <div class="shared-permissions">
                            <t-tooltip :content="$t('organization.settings.sharePermissionLabel')" placement="top">
                              <t-tag size="small" :theme="getPermissionTheme(share.permission)" variant="outline"
                                class="perm-tag">
                                {{ $t('organization.settings.sharePermissionLabel') }}: {{ (share.permission ===
                                  'editor' ||
                                  share.permission === 'admin') ? $t('organization.share.permissionEditable') :
                                  $t('organization.share.permissionReadonly') }}
                              </t-tag>
                            </t-tooltip>
                            <t-tooltip :content="$t('organization.settings.permissionCalcTip')" placement="top">
                              <t-tag size="small" :theme="getPermissionTheme(share.my_permission ?? share.permission)"
                                class="perm-tag">
                                {{ $t('organization.settings.myPermissionLabel') }}: {{ ((share.my_permission ??
                                  share.permission) ===
                                  'editor' || (share.my_permission ?? share.permission) === 'admin') ?
                                  $t('organization.share.permissionEditable') :
                                  $t('organization.share.permissionReadonly') }}
                              </t-tag>
                            </t-tooltip>
                          </div>
                          <t-popconfirm v-if="isAdmin"
                            :content="$t('organization.settings.removeShareConfirm', { name: share.knowledge_base_name || share.knowledge_base_id })"
                            :confirm-btn="{ content: $t('common.confirm'), theme: 'danger' }"
                            :cancel-btn="{ content: $t('common.cancel') }" @confirm="handleRemoveShare(share)">
                            <t-tooltip :content="$t('organization.settings.removeShareFromOrg')" placement="top">
                              <t-button variant="text" size="small" theme="danger" class="shared-remove-btn"
                                @click.stop>
                                <t-icon name="delete" size="16px" />
                              </t-button>
                            </t-tooltip>
                          </t-popconfirm>
                        </div>
                      </div>
                    </t-loading>
                  </div>
                </div>

                <!-- 共享智能体（独立侧边栏） -->
                <div v-show="currentSection === 'sharedAgents'" class="section">
                  <div class="section-header">
                    <h2>{{ $t('organization.settings.sharedAgents') }}</h2>
                    <p class="section-description">{{ $t('organization.settings.sharedAgentsDesc') }}</p>
                    <p class="section-description permission-calc-hint">
                      <t-tooltip :content="$t('organization.settings.sharedAgentsKbHint')" placement="top"
                        :show-delay="300">
                        <span class="hint-inner">
                          <t-icon name="info-circle" size="14px" />
                          {{ $t('organization.settings.sharedAgentsKbHintShort') }}
                        </span>
                      </t-tooltip>
                    </p>
                  </div>
                  <div class="settings-group">
                    <div v-if="sharedAgents.length === 0" class="empty-shared">
                      <div class="empty-icon">
                        <img src="@/assets/img/agent.svg" class="empty-icon-agent" alt="" aria-hidden="true" />
                      </div>
                      <p class="empty-text">{{ $t('organization.settings.noSharedAgents') }}</p>
                      <p class="empty-subtext">{{ $t('organization.settings.noSharedAgentsTip') }}</p>
                    </div>
                    <div v-else class="shared-list">
                      <div v-for="share in sharedAgents" :key="share.id" class="shared-item"
                        @mouseenter="onSharedAgentMouseEnter(share, $event)" @mousemove="onSharedAgentMouseMove($event)"
                        @mouseleave="onSharedAgentMouseLeave">
                        <div class="shared-icon shared-icon-agent-wrap">
                          <AgentAvatar :name="share.agent_name || share.agent_id" size="small" />
                        </div>
                        <div class="shared-info">
                          <span class="shared-name">{{ share.agent_name || share.agent_id }}</span>
                          <div class="shared-meta">
                            <span v-if="share.shared_by_username" class="shared-by"><t-icon name="user" size="12px" />{{
                              share.shared_by_username }}</span>
                            <span class="shared-time"><t-icon name="time" size="12px" />{{ formatDate(share.created_at)
                            }}</span>
                          </div>
                        </div>
                        <t-popconfirm v-if="isAdmin"
                          :content="$t('organization.settings.removeAgentShareConfirm', { name: share.agent_name || share.agent_id })"
                          :confirm-btn="{ content: $t('common.confirm'), theme: 'danger' }"
                          :cancel-btn="{ content: $t('common.cancel') }" @confirm="handleRemoveAgentShare(share)">
                          <t-button variant="text" size="small" theme="danger" class="shared-remove-btn"
                            @click.stop><t-icon name="delete" size="16px" /></t-button>
                        </t-popconfirm>
                      </div>
                    </div>
                  </div>
                </div>

              </div>

              <!-- 共享智能体 hover 跟随气泡 -->
              <Teleport to="body">
                <Transition name="agent-scope-popover-fade">
                  <div v-if="agentScopePopover" class="agent-scope-popover-follow" :style="agentScopePopoverStyle">
                    <div class="agent-scope-popover-card">
                      <div class="agent-scope-popover-name">{{ agentScopePopover.share.agent_name ||
                        agentScopePopover.share.agent_id
                      }}</div>
                      <div class="agent-scope-popover-meta">
                        <span v-if="agentScopePopover.share.shared_by_username" class="popover-meta-item">
                          <t-icon name="user" size="12px" /> {{ agentScopePopover.share.shared_by_username }}
                        </span>
                        <span class="popover-meta-item">
                          <t-icon name="time" size="12px" /> {{ formatDate(agentScopePopover.share.created_at) }}
                        </span>
                      </div>
                      <div class="agent-scope-popover-permission">
                        <span class="popover-label">{{ $t('organization.settings.sharePermissionLabel') }}</span>
                        <span class="popover-value">{{ $t('organization.share.permissionReadonly') }}</span>
                      </div>
                      <template v-if="getAgentScopeTags(agentScopePopover.share).length">
                        <div class="agent-scope-popover-divider" />
                        <div class="agent-scope-popover-section-title">{{ $t('agent.shareScope.title') }}</div>
                        <div v-for="(tag, idx) in getAgentScopeTags(agentScopePopover.share)" :key="idx"
                          class="agent-scope-popover-row">{{ tag }}</div>
                      </template>
                    </div>
                  </div>
                </Transition>
              </Teleport>

              <!-- 底部操作按钮 -->
              <div class="settings-footer">
                <t-button variant="outline" @click="handleClose">{{ $t('common.cancel') }}</t-button>
                <t-button v-if="isAdmin" theme="primary" :loading="submitting" @click="handleSave">
                  {{ isCreateMode ? $t('common.create') : $t('common.save') }}
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 添加参与成员弹窗（从成员管理数据选择，按成员本人授权） -->
    <t-dialog v-model:visible="showAddMemberDialog" :header="$t('organization.addMember.dialogTitle')"
      :confirm-btn="{ content: $t('organization.addMember.confirmBtn'), loading: addMemberSubmitting, disabled: selectedInviteCandidate == null }"
      :cancel-btn="$t('common.cancel')" @confirm="handleAddMember" @close="resetAddMemberDialog" width="420px">
      <div class="add-member-dialog">
        <div class="add-member-field">
          <label>{{ $t('organization.addMember.searchUser') }}</label>
          <t-select v-model="selectedInviteCandidateKey" :placeholder="$t('organization.addMember.searchPlaceholder')"
            filterable :filter="() => true" :loading="userSearchLoading" @search="handleUserSearch" clearable
            :options="userSearchOptions" @focus="loadDefaultUserList"
            @visible-change="handleUserSelectVisibleChange" />
          <p class="field-hint">{{ $t('organization.addMember.searchHint') }}</p>
        </div>

        <div class="add-member-field">
          <label>{{ $t('organization.addMember.selectRole') }}</label>
          <t-select v-model="addMemberRole" :options="addMemberRoleOptions"
            :placeholder="$t('organization.addMember.selectRole')" />
        </div>
      </div>
    </t-dialog>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  getOrganization,
  listMembers,
  updateOrganization,
  updateMemberRole,
  removeMember,
  listOrgShares,
  listOrgAgentShares,
  removeShare,
  removeAgentShare,
  searchUsersForInvite,
  inviteMember,
  type Organization,
  type OrganizationMember,
  type KnowledgeBaseShare,
  type AgentShareResponse,
  type UserSearchResult
} from '@/api/organization'
import { useOrganizationStore } from '@/stores/organization'
import { useAuthStore } from '@/stores/auth'
import SpaceAvatar from '@/components/SpaceAvatar.vue'
import AgentAvatar from '@/components/AgentAvatar.vue'
import agentIconSrc from '@/assets/img/agent.svg'
import agentIconActiveSrc from '@/assets/img/agent-green.svg'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const orgStore = useOrganizationStore()

interface Props {
  visible: boolean
  orgId?: string
  mode?: 'view' | 'edit' | 'create'
}

const props = withDefaults(defineProps<Props>(), {
  mode: 'view'
})

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'saved'): void
}>()

// State
const currentSection = ref('basic')
const orgInfo = ref<Organization | null>(null)
const members = ref<OrganizationMember[]>([])
const sharedKnowledgeBases = ref<KnowledgeBaseShare[]>([])
const sharedAgents = ref<AgentShareResponse[]>([])
const sharesLoading = ref(false)
const membersLoading = ref(false)
const memberSearchQuery = ref('')
const submitting = ref(false)

// 添加参与成员从成员管理数据选择；后端按 user_id 建立共享空间关系，tenant_id 只作为隐藏来源上下文。
const showAddMemberDialog = ref(false)
const addMemberSubmitting = ref(false)
const userSearchLoading = ref(false)
const userSearchResults = ref<UserSearchResult[]>([])
const userSearchBootstrapped = ref(false)
const selectedInviteCandidateKey = ref<string | null>(null)
const addMemberRole = ref<'admin' | 'editor' | 'viewer'>('viewer')

const formData = ref({
  name: '',
  description: '',
  avatar: '' as string,
  member_limit: 50 as number // 0 = unlimited
})

// 共享智能体 hover 跟随气泡
const agentScopePopover = ref<{ share: AgentShareResponse; x: number; y: number } | null>(null)
const agentScopePopoverTimer = ref<ReturnType<typeof setTimeout> | null>(null)
const POPOVER_OFFSET = 14
const POPOVER_DELAY = 200

// 空间头像可选 Emoji（方案三：Emoji 作为头像）
const avatarEmojiOptions = [
  '🚀', '📁', '👥', '🏢', '💡', '📚', '🌟', '🔧', '📌', '🎯',
  '📂', '🔒', '🌐', '⚡', '🎨', '📊', '🤝', '💼', '📧', '🏠',
  '🔑', '📈', '✨', '📋', '🌍', '💬', '🔔', '📦', '🎉', '🌈'
]
const avatarPopoverVisible = ref(false)

function selectAvatarEmoji(emoji: string) {
  formData.value.avatar = 'emoji:' + emoji
  avatarPopoverVisible.value = false
}
function clearAvatarEmoji() {
  formData.value.avatar = ''
  avatarPopoverVisible.value = false
}

// Computed
const isCreateMode = computed(() => props.mode === 'create')
const isEditMode = computed(() => props.mode === 'edit' || props.mode === 'create')
// 后端组织相关变更接口（保存设置、邀请、搜索用户、改/删成员、移除共享等）在路由层都要求当前空间角色 ≥ admin（见
// internal/router/router.go 的 RegisterOrganizationRoutes）。跨空间超管可绕过。
// 因此前端任何"管理类"入口必须同时满足：组织内是 admin/owner ∩ 当前空间 admin+。
const hasTenantAdmin = computed(
  () => authStore.hasRole('admin') || authStore.canAccessAllTenants
)
const isAdmin = computed(() => {
  if (isCreateMode.value) return hasTenantAdmin.value
  const orgAdmin = orgInfo.value?.my_role === 'admin' || orgInfo.value?.is_owner
  return !!orgAdmin && hasTenantAdmin.value
})

// 当用户在组织内是 admin/owner 但当前空间角色不足时，展示只读提示
const showTenantRoleHint = computed(() => {
  if (isCreateMode.value) return !hasTenantAdmin.value
  const orgAdmin = orgInfo.value?.my_role === 'admin' || orgInfo.value?.is_owner
  return !!orgAdmin && !hasTenantAdmin.value
})

// 添加参与成员时可选的角色
const addMemberRoleOptions = computed(() => [
  { label: t('organization.role.viewer'), value: 'viewer' },
  { label: t('organization.role.editor'), value: 'editor' },
  { label: t('organization.role.admin'), value: 'admin' },
])

const inviteCandidateUserId = (u: UserSearchResult): string =>
  u.user_id || u.id || u.representative_user_id || ''

const inviteCandidateKey = (u: UserSearchResult): string => {
  const userId = inviteCandidateUserId(u)
  return userId
}

const selectedInviteCandidate = computed(() => {
  const selectedKey = selectedInviteCandidateKey.value
  if (!selectedKey) return null
  return userSearchResults.value.find((u) => inviteCandidateKey(u) === selectedKey) || null
})

const userSearchOptions = computed(() =>
  userSearchResults.value.flatMap((u) => {
    const userId = inviteCandidateUserId(u)
    const value = inviteCandidateKey(u)
    if (!userId || !value) return []
    const username = u.username || u.representative_username || ''
    const phone = u.phone || u.representative_phone || u.email || u.representative_email || ''
    const userLabel = username || phone || userId
    const contact = phone && phone !== userLabel ? ` / ${phone}` : ''
    const joined = u.is_already_member ? ` (${t('organization.addMember.alreadyInSpace')})` : ''
    return [{
      label: `${userLabel}${contact}${joined}`,
      value,
      disabled: !!u.is_already_member || !u.tenant_id,
    }]
  })
)

const modalTitle = computed(() => {
  if (isCreateMode.value) return t('organization.createOrg')
  return t('organization.settings.editTitle')
})

const navItems = computed(() => {
  const items: { key: string; icon: string; label: string; badge?: number }[] = [
    { key: 'basic', icon: 'info-circle', label: t('organization.editor.navBasic') },
  ]
  // 只有在编辑已有组织时才显示参与成员和共享资源
  if (props.orgId && !isCreateMode.value) {
    items.push({ key: 'members', icon: 'user', label: t('organization.manageMembers') })
    items.push({
      key: 'sharedKb',
      icon: 'folder-open',
      label: t('organization.share.sharedKnowledgeBase'),
      badge: sharedKnowledgeBases.value.length
    })
    items.push({
      key: 'sharedAgents',
      icon: 'control-platform',
      label: t('organization.settings.sharedAgents'),
      badge: sharedAgents.value.length
    })
  }
  return items
})

const roleOptions = computed(() => [
  { label: t('organization.role.admin'), value: 'admin' },
  { label: t('organization.role.editor'), value: 'editor' },
  { label: t('organization.role.viewer'), value: 'viewer' }
])

const filteredMembers = computed(() => {
  const query = memberSearchQuery.value.toLowerCase()
  if (!query) return members.value
  return members.value.filter((m) =>
    (m.username || '').toLowerCase().includes(query) ||
    (m.phone || '').toLowerCase().includes(query) ||
    (m.email || '').toLowerCase().includes(query) ||
    (m.representative_user_id || m.user_id || '').toLowerCase().includes(query)
  )
})

const memberPrimaryLabel = (m: OrganizationMember): string => {
  return m.username || m.phone || m.email || m.representative_user_id || m.user_id || `member#${m.id}`
}

const memberSecondaryLabel = (m: OrganizationMember): string => {
  const primary = memberPrimaryLabel(m)
  const contact = m.phone || m.email || ''
  if (contact && contact !== primary) return contact
  const accountID = m.representative_user_id || m.user_id || ''
  return accountID && accountID !== primary ? accountID : ''
}

const memberContainsCurrentUser = (m: OrganizationMember): boolean => {
  const currentUserId = authStore.currentUserId
  if (!currentUserId) return false
  return m.user_id === currentUserId || m.representative_user_id === currentUserId
}

const isOwnerMember = (member: OrganizationMember): boolean => {
  return member.user_id === orgInfo.value?.owner_id || member.representative_user_id === orgInfo.value?.owner_id
}

// Methods
const handleClose = () => {
  emit('update:visible', false)
}

const fetchOrgDetail = async () => {
  if (!props.orgId) return
  try {
    const res = await getOrganization(props.orgId)
    if (res.success && res.data) {
      orgInfo.value = res.data
      const memberLimit = res.data.member_limit
      formData.value = {
        name: res.data.name,
        description: res.data.description || '',
        avatar: res.data.avatar || '',
        member_limit: typeof memberLimit === 'number' && memberLimit >= 0 ? memberLimit : 50
      }
    }
  } catch (error) {
    console.error('Failed to fetch org:', error)
  }
}

const fetchMembers = async () => {
  if (!props.orgId) return
  membersLoading.value = true
  try {
    const res = await listMembers(props.orgId)
    if (res.success && res.data) {
      members.value = res.data.members || []
    }
  } catch (error) {
    console.error('Failed to fetch members:', error)
  } finally {
    membersLoading.value = false
  }
}

const fetchSharedKBs = async () => {
  if (!props.orgId) return
  sharesLoading.value = true
  try {
    const [kbRes, agentRes] = await Promise.all([
      listOrgShares(props.orgId),
      listOrgAgentShares(props.orgId)
    ])
    if (kbRes.success && kbRes.data) {
      sharedKnowledgeBases.value = kbRes.data.shares || []
    } else {
      sharedKnowledgeBases.value = []
    }
    if (agentRes.success && agentRes.data) {
      sharedAgents.value = agentRes.data.shares || []
    } else {
      sharedAgents.value = []
    }
  } catch (error) {
    console.error('Failed to fetch shared resources:', error)
    sharedKnowledgeBases.value = []
    sharedAgents.value = []
  } finally {
    sharesLoading.value = false
  }
}

const handleSave = async () => {
  if (!formData.value.name.trim()) {
    MessagePlugin.warning(t('organization.nameRequired'))
    currentSection.value = 'basic'
    return
  }

  submitting.value = true
  try {
    if (isCreateMode.value) {
      // 创建模式
      const result = await orgStore.create(
        formData.value.name.trim(),
        formData.value.description.trim()
      )
      if (result) {
        MessagePlugin.success(t('organization.createSuccess'))
        emit('saved')
        handleClose()
      } else {
        MessagePlugin.error(orgStore.error || t('organization.createFailed'))
      }
    } else {
      // 编辑模式
      if (!props.orgId) return
      const res = await updateOrganization(props.orgId, {
        name: formData.value.name.trim(),
        description: formData.value.description.trim(),
        avatar: formData.value.avatar || undefined,
        member_limit: formData.value.member_limit
      })
      if (res.success) {
        MessagePlugin.success(t('common.saveSuccess'))
        emit('saved')
        handleClose()
      } else {
        MessagePlugin.error(res.message || t('common.saveFailed'))
      }
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.saveFailed'))
  } finally {
    submitting.value = false
  }
}

const handleRoleChange = async (member: OrganizationMember, newRole: string) => {
  if (!props.orgId) return
  try {
    const res = await updateMemberRole(props.orgId, member.id, {
      role: newRole as 'admin' | 'editor' | 'viewer'
    })
    if (res.success) {
      MessagePlugin.success(t('organization.roleUpdated'))
    } else {
      MessagePlugin.error(res.message || t('organization.roleUpdateFailed'))
      fetchMembers()
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('organization.roleUpdateFailed'))
    fetchMembers()
  }
}

const confirmRemoveMember = async (member: OrganizationMember) => {
  if (!props.orgId) return

  try {
    const res = await removeMember(props.orgId, member.id)
    if (res.success) {
      MessagePlugin.success(t('organization.memberRemoved'))
      fetchMembers()
    } else {
      MessagePlugin.error(res.message || t('organization.memberRemoveFailed'))
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('organization.memberRemoveFailed'))
  }
}

// 添加参与成员：唤起成员管理候选列表，并支持用户名 / 手机号模糊搜索。
let userSearchTimer: ReturnType<typeof setTimeout> | null = null
const fetchUserInviteCandidates = async (query = '') => {
  if (!props.orgId) return
  userSearchLoading.value = true
  try {
    const res = await searchUsersForInvite(props.orgId, query, 20)
    if (res.success && res.data) {
      userSearchResults.value = res.data
    }
  } catch (error) {
    console.error('Failed to search member candidates:', error)
  } finally {
    userSearchLoading.value = false
  }
}

const loadDefaultUserList = () => {
  if (userSearchLoading.value || userSearchBootstrapped.value) return
  userSearchBootstrapped.value = true
  fetchUserInviteCandidates()
}

const handleUserSelectVisibleChange = (visible: boolean) => {
  if (visible) {
    loadDefaultUserList()
  }
}

const openAddMemberDialog = () => {
  showAddMemberDialog.value = true
  loadDefaultUserList()
}

const handleUserSearch = (query: string) => {
  if (userSearchTimer) {
    clearTimeout(userSearchTimer)
  }
  userSearchTimer = setTimeout(async () => {
    const keyword = query.trim()
    await fetchUserInviteCandidates(keyword.length >= 2 ? keyword : '')
  }, 300)
}

// 添加参与成员：提交选中的成员行，后端按该成员本人建立共享空间关系。
const handleAddMember = async () => {
  const candidate = selectedInviteCandidate.value
  const representativeUserId = candidate ? inviteCandidateUserId(candidate) : ''
  if (!props.orgId || !candidate?.tenant_id || !representativeUserId) return

  addMemberSubmitting.value = true
  try {
    const res = await inviteMember(props.orgId, {
      tenant_id: candidate.tenant_id,
      representative_user_id: representativeUserId,
      user_id: representativeUserId,
      role: addMemberRole.value,
    })
    if (res.success) {
      MessagePlugin.success(t('organization.addMember.success'))
      showAddMemberDialog.value = false
      resetAddMemberDialog()
      fetchMembers() // 刷新成员列表
    } else {
      MessagePlugin.error(res.message || t('organization.addMember.failed'))
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('organization.addMember.failed'))
  } finally {
    addMemberSubmitting.value = false
  }
}

// 重置添加参与成员弹窗
const resetAddMemberDialog = () => {
  selectedInviteCandidateKey.value = null
  addMemberRole.value = 'viewer'
  userSearchResults.value = []
  userSearchBootstrapped.value = false
}

const handleShareClick = (share: KnowledgeBaseShare) => {
  handleClose()
  router.push(`/platform/knowledge-bases/${share.knowledge_base_id}`)
}

const handleRemoveShare = async (share: KnowledgeBaseShare) => {
  if (!props.orgId) return
  try {
    const res = await removeShare(share.knowledge_base_id, share.id)
    if (res.success) {
      MessagePlugin.success(t('organization.settings.removeShareSuccess'))
      sharedKnowledgeBases.value = sharedKnowledgeBases.value.filter(s => s.id !== share.id)
    } else {
      MessagePlugin.error(res.message || t('organization.settings.removeShareFailed'))
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('organization.settings.removeShareFailed'))
  }
}

const handleRemoveAgentShare = async (share: AgentShareResponse) => {
  if (!props.orgId) return
  try {
    const res = await removeAgentShare(share.agent_id, share.id)
    if (res.success) {
      MessagePlugin.success(t('organization.settings.removeShareSuccess'))
      sharedAgents.value = sharedAgents.value.filter(s => s.id !== share.id)
    } else {
      MessagePlugin.error(res.message || t('organization.settings.removeShareFailed'))
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('organization.settings.removeShareFailed'))
  }
}

const formatDate = (dateStr: string) => {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** 共享智能体能力范围标签（知识库、网络搜索、MCP） */
function getAgentScopeTags(share: AgentShareResponse): string[] {
  const tags: string[] = []
  if (share.scope_kb !== undefined && share.scope_kb !== '') {
    const kbText = share.scope_kb === 'all'
      ? t('agent.shareScope.kbAll')
      : share.scope_kb === 'selected' && (share.scope_kb_count ?? 0) > 0
        ? t('agent.shareScope.kbSelected', { count: share.scope_kb_count })
        : t('agent.shareScope.kbNone')
    tags.push(`${t('agent.shareScope.knowledgeBase')}：${kbText}`)
  }
  if (share.scope_web_search !== undefined) {
    tags.push(`${t('agent.shareScope.webSearch')}：${share.scope_web_search ? t('agent.shareScope.enabled') : t('agent.shareScope.disabled')}`)
  }
  if (share.scope_mcp !== undefined && share.scope_mcp !== '') {
    const mcpText = share.scope_mcp === 'all'
      ? t('agent.shareScope.mcpAll')
      : share.scope_mcp === 'selected' && (share.scope_mcp_count ?? 0) > 0
        ? t('agent.shareScope.mcpSelected', { count: share.scope_mcp_count })
        : t('agent.shareScope.mcpNone')
    tags.push(`${t('agent.shareScope.mcp')}：${mcpText}`)
  }
  return tags
}

function onSharedAgentMouseEnter(share: AgentShareResponse, e: MouseEvent) {
  agentScopePopoverTimer.value = setTimeout(() => {
    agentScopePopover.value = { share, x: e.clientX, y: e.clientY }
    agentScopePopoverTimer.value = null
  }, POPOVER_DELAY)
}

function onSharedAgentMouseMove(e: MouseEvent) {
  if (agentScopePopover.value) {
    agentScopePopover.value = { ...agentScopePopover.value, x: e.clientX, y: e.clientY }
  }
}

function onSharedAgentMouseLeave() {
  if (agentScopePopoverTimer.value) {
    clearTimeout(agentScopePopoverTimer.value)
    agentScopePopoverTimer.value = null
  }
  agentScopePopover.value = null
}

const agentScopePopoverStyle = computed(() => {
  if (!agentScopePopover.value) return {}
  const { x, y } = agentScopePopover.value
  const popoverWidth = 240
  const popoverHeight = 180
  let left = x + POPOVER_OFFSET
  let top = y + POPOVER_OFFSET
  const rightEdge = window.innerWidth - popoverWidth - 12
  const bottomEdge = window.innerHeight - popoverHeight - 12
  if (left > rightEdge) left = rightEdge
  if (left < 12) left = 12
  if (top > bottomEdge) top = bottomEdge
  if (top < 12) top = 12
  return { left: `${left}px`, top: `${top}px` }
})

const getRoleTheme = (role: string) => {
  switch (role) {
    case 'admin': return 'primary'
    case 'editor': return 'warning'
    case 'viewer': return 'default'
    default: return 'default'
  }
}

const getPermissionTheme = (permission: string) => {
  switch (permission) {
    case 'admin': return 'primary'
    case 'editor': return 'warning'
    case 'viewer': return 'default'
    default: return 'default'
  }
}

// Watch
watch(() => props.visible, (newVal) => {
  if (newVal) {
    currentSection.value = 'basic'
    memberSearchQuery.value = ''
    if (props.mode === 'create') {
      // 创建模式：重置表单
      formData.value = { name: '', description: '', avatar: '', member_limit: 50 }
      orgInfo.value = null
      members.value = []
      sharedKnowledgeBases.value = []
    } else if (props.orgId) {
      fetchOrgDetail()
      fetchMembers()
      fetchSharedKBs()
    }
  } else {
    if (agentScopePopoverTimer.value) {
      clearTimeout(agentScopePopoverTimer.value)
      agentScopePopoverTimer.value = null
    }
    agentScopePopover.value = null
  }
})
</script>

<style scoped lang="less">
@primary-color: var(--td-brand-color);
@primary-light: var(--td-brand-color-light);
@primary-lighter: var(--td-component-stroke);
@primary-hover: var(--td-brand-color-active);

.settings-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 2000;
  backdrop-filter: blur(4px);
}

.settings-modal {
  position: relative;
  width: 90vw;
  max-width: 1100px;
  height: 85vh;
  max-height: 750px;
  background: var(--td-bg-color-container);
  border-radius: 16px;
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.04),
    0 4px 6px -1px rgba(15, 23, 42, 0.06),
    0 12px 24px -4px rgba(15, 23, 42, 0.1),
    0 24px 48px -8px rgba(15, 23, 42, 0.12);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.close-btn {
  position: absolute;
  top: 20px;
  right: 20px;
  width: 36px;
  height: 36px;
  border: none;
  background: var(--td-bg-color-container-hover);
  border-radius: 10px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
  transition: background 0.2s ease, color 0.2s ease, transform 0.15s ease;
  z-index: 10;

  &:hover {
    background: rgba(0, 0, 0, 0.08);
    color: var(--td-text-color-primary);
    transform: scale(1.02);
  }

  &:active {
    transform: scale(0.98);
  }
}

.settings-container {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.settings-sidebar {
  width: 200px;
  background: var(--td-bg-color-settings-modal);
  border-right: 1px solid var(--td-component-stroke);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;

  .sidebar-header {
    padding: 26px 20px;
    border-bottom: 1px solid var(--td-component-stroke);

    .sidebar-title {
      margin: 0;
      font-family: var(--app-font-family);
      font-size: 18px;
      font-weight: 600;
      color: var(--td-text-color-primary);
      letter-spacing: -0.02em;
    }
  }

  .settings-nav {
    flex: 1;
    padding: 12px 8px;
    overflow-y: auto;

    .nav-item {
      display: flex;
      align-items: center;
      padding: 12px 14px;
      margin-bottom: 4px;
      border-radius: 10px;
      cursor: pointer;
      transition: background 0.2s ease, color 0.2s ease;
      font-family: var(--app-font-family);
      font-size: 14px;
      color: var(--td-text-color-secondary);
      font-weight: 500;

      .nav-icon {
        margin-right: 10px;
        font-size: 18px;
        flex-shrink: 0;
        display: flex;
        align-items: center;
        justify-content: center;
        color: inherit;
        transition: color 0.2s;

        &.nav-icon-img {
          width: 18px;
          height: 18px;
        }
      }

      .nav-label {
        flex: 1;
        min-width: 0;
      }

      .nav-item-badge {
        min-width: 20px;
        height: 20px;
        padding: 0 6px;
        border-radius: 10px;
        background: rgba(250, 173, 20, 0.18);
        color: var(--td-warning-color-active);
        font-size: 12px;
        font-weight: 600;
        line-height: 20px;
        text-align: center;
        flex-shrink: 0;

        &.nav-item-badge-count {
          background: rgba(0, 0, 0, 0.06);
          color: var(--td-text-color-secondary);
          font-weight: 500;
        }
      }

      &:hover {
        background: var(--td-bg-color-container-hover);
        color: var(--td-text-color-primary);
      }

      &.active {
        background: var(--td-bg-color-secondarycontainer);
        color: var(--td-brand-color);
        font-weight: 600;
      }
    }
  }
}

.settings-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}

.content-wrapper {
  flex: 1;
  overflow-y: auto;
  padding: 24px 32px;
}

.tenant-role-hint {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  margin-bottom: 16px;
  padding: 10px 12px;
  background: var(--td-warning-color-light);
  border: 1px solid var(--td-warning-color-focus);
  border-radius: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--td-warning-color-active);

  .t-icon {
    flex-shrink: 0;
    margin-top: 2px;
  }
}

.section {
  .section-header {
    margin-bottom: 20px;

    h2 {
      margin: 0 0 8px 0;
      font-family: var(--app-font-family);
      font-size: 16px;
      font-weight: 600;
      color: var(--td-text-color-primary);
    }

    .section-description {
      margin: 0;
      font-family: var(--app-font-family);
      font-size: 14px;
      color: var(--td-text-color-placeholder);
      line-height: 22px;
    }

    .permission-calc-hint {
      margin-top: 6px;

      .hint-inner {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        cursor: help;
        color: var(--td-text-color-secondary);
        font-size: 13px;
      }
    }
  }
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:first-child {
    padding-top: 0;
  }

  &:last-child {
    border-bottom: none;
  }

  .setting-info {
    flex: 1;
    max-width: 45%;
    padding-right: 20px;

    &.full-width {
      max-width: 100%;
      padding-right: 0;
    }

    label {
      display: block;
      font-size: 14px;
      font-weight: 600;
      color: var(--td-text-color-primary);
      margin-bottom: 4px;

      .required {
        color: var(--td-error-color);
        margin-left: 2px;
      }
    }

    .desc {
      font-size: 13px;
      color: var(--td-text-color-secondary);
      margin: 0;
      line-height: 1.5;
    }
  }

  .setting-control {
    flex: 1;
    max-width: 50%;
    min-width: 0;

    &.full-width {
      max-width: 100%;
    }
  }

  &.setting-row-vertical {
    flex-direction: column;
    gap: 12px;
  }
}

.avatar-trigger-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  cursor: pointer;
  flex-shrink: 0;
  padding: 4px;
  border-radius: 12px;
  transition: background 0.2s ease;
}

.avatar-trigger-wrap:hover {
  background: var(--td-bg-color-container-hover);
}

.avatar-change-hint {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
  line-height: 1.2;
}

.name-input-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-input-wrapper .name-input {
  flex: 1;
  min-width: 0;
}

/* 头像 Emoji 弹层内容 */
.avatar-popover-content {
  padding: 12px;
  min-width: 260px;
}

.avatar-popover-title {
  margin: 0 0 10px 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.4;
}

.avatar-popover-content .avatar-emoji-grid {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-width: 280px;
}

.avatar-popover-content .avatar-emoji-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  font-size: 18px;
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.avatar-popover-content .avatar-emoji-btn:hover {
  border-color: var(--td-brand-color);
  background: rgba(7, 192, 95, 0.06);
}

.avatar-popover-content .avatar-emoji-btn.is-selected {
  border-color: var(--td-brand-color);
  background: rgba(7, 192, 95, 0.12);
}

.avatar-popover-content .avatar-clear-btn {
  margin-top: 10px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.avatar-popover-content .avatar-clear-btn:hover {
  color: var(--td-brand-color-active);
}

.member-limit-input-row {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;

  .member-limit-hint {
    font-size: 12px;
    color: var(--td-text-color-secondary);
  }
}

// Members
.members-header {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
  align-items: center;

  .members-search {
    flex: 1;
  }
}

.members-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;

  .member-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--td-bg-color-container);
    border-radius: 8px;
    transition: background 0.2s;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }

    &.is-me {
      border: 1px solid @primary-color;
      background: rgba(7, 192, 95, 0.04);
    }

    .member-avatar {
      width: 36px;
      height: 36px;
      border-radius: 50%;
      background: var(--td-bg-color-secondarycontainer);
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
      color: var(--td-text-color-secondary);

      &.is-me {
        background: rgba(7, 192, 95, 0.15);
        color: @primary-color;
        box-shadow: 0 0 0 2px @primary-color;
      }

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }
    }

    .member-info {
      flex: 1;
      min-width: 0;

      .member-name {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        font-weight: 500;
        color: var(--td-text-color-primary);

        .me-tag {
          display: inline-flex;
          align-items: center;
          padding: 0 5px;
          height: 16px;
          background: @primary-color;
          color: var(--td-text-color-anti);
          border-radius: 3px;
          font-size: 10px;
          font-weight: 500;
          flex-shrink: 0;
        }
      }

      .member-email {
        display: block;
        font-size: 12px;
        color: var(--td-text-color-secondary);
      }

      .member-tenant-users {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-top: 8px;
      }

      .member-tenant-user {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        max-width: 160px;
        height: 22px;
        padding: 0 7px;
        border-radius: 4px;
        background: var(--td-bg-color-secondarycontainer);
        color: var(--td-text-color-secondary);
        font-size: 12px;
        line-height: 22px;
      }

      .member-tenant-user-label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .member-role {
      flex-shrink: 0;
    }

    .member-actions {
      flex-shrink: 0;
    }
  }

  .empty-members {
    padding: 32px;
    text-align: center;
    color: var(--td-text-color-secondary);
    font-size: 14px;
  }
}

// Shared KBs
.empty-shared {
  padding: 48px 24px;
  text-align: center;

  .empty-icon {
    width: 80px;
    height: 80px;
    margin: 0 auto 16px;
    border-radius: 50%;
    background: var(--td-bg-color-container);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--td-text-color-placeholder);

    .empty-icon-agent {
      width: 48px;
      height: 48px;
    }

    .empty-icon-kb {
      width: 48px;
      height: 48px;
    }
  }

  .empty-text {
    font-size: 14px;
    color: var(--td-text-color-secondary);
    margin: 0 0 8px;
  }

  .empty-subtext {
    font-size: 12px;
    color: var(--td-text-color-secondary);
    margin: 0;
  }

  &.small {
    padding: 24px 16px;

    .empty-text {
      margin: 0;
    }
  }
}

.shared-subsection {
  margin-top: 24px;
  padding-top: 24px;
  border-top: 1px solid var(--td-component-stroke);

  .shared-subtitle {
    font-size: 14px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 12px 0;
  }
}

.shared-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 400px;
  overflow-y: auto;

  .shared-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--td-bg-color-container);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--td-brand-color-light);
    }

    .shared-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      padding: 0 8px;
      height: 26px;
      border-radius: 6px;
      gap: 4px;

      &.type-document {
        background: rgba(7, 192, 95, 0.08);
        color: var(--td-brand-color-active);
      }

      &.type-faq {
        background: rgba(0, 82, 217, 0.08);
        color: var(--td-brand-color);
      }

      & .shared-icon-org {
        background: rgba(7, 192, 95, 0.08);
        color: var(--td-brand-color-active);
      }

      &.shared-icon-agent-wrap {
        padding: 0;
        height: auto;
        background: transparent;
      }

      &.shared-icon-kb {
        background: rgba(7, 192, 95, 0.08);
        color: var(--td-brand-color-active);
      }

      .shared-icon-kb-img {
        width: 20px;
        height: 20px;
        flex-shrink: 0;
      }

      .shared-icon-agent {
        width: 20px;
        height: 20px;
        flex-shrink: 0;
      }

      .org-icon-img {
        width: 18px;
        height: 18px;
        flex-shrink: 0;
      }

      .badge-count {
        font-size: 12px;
        font-weight: 500;
      }
    }

    .shared-info {
      flex: 1;
      min-width: 0;

      .shared-name {
        display: block;
        font-size: 14px;
        font-weight: 500;
        color: var(--td-text-color-primary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        margin-bottom: 4px;
      }

      .shared-meta {
        display: flex;
        align-items: center;
        gap: 12px;
        font-size: 12px;
        color: var(--td-text-color-secondary);

        .shared-by,
        .shared-time {
          display: flex;
          align-items: center;
          gap: 4px;
        }
      }

      .shared-desc {
        display: block;
        font-size: 12px;
        color: var(--td-text-color-secondary);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .shared-permissions {
      display: flex;
      flex-direction: column;
      align-items: flex-end;
      gap: 6px;
      flex-shrink: 0;
      margin-left: auto;

      .perm-tag {
        white-space: nowrap;
      }
    }

    .shared-remove-btn {
      flex-shrink: 0;
      margin-left: 4px;
    }
  }
}

.settings-footer {
  padding: 20px 32px;
  border-top: 1px solid rgba(0, 0, 0, 0.06);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
  background: var(--td-bg-color-container);
}

// Transitions
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.35s cubic-bezier(0.4, 0, 0.2, 1);

  .settings-modal {
    transition: transform 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
  }
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;

  .settings-modal {
    transform: scale(0.92) translateY(-8px);
  }
}

.modal-enter-to,
.modal-leave-from {
  .settings-modal {
    transform: scale(1) translateY(0);
  }
}

.add-member-dialog {
  .add-member-field {
    margin-bottom: 20px;

    &:last-child {
      margin-bottom: 0;
    }

    label {
      display: block;
      margin-bottom: 8px;
      font-size: 14px;
      font-weight: 500;
      color: var(--td-text-color-primary);
    }

    .t-select {
      width: 100%;
    }

    .field-hint {
      margin: 6px 0 0;
      font-size: 12px;
      color: var(--td-text-color-secondary);
    }
  }
}
</style>

<style lang="less">
/* 共享智能体 hover 跟随气泡（Teleport 到 body） */
.agent-scope-popover-follow {
  position: fixed;
  z-index: 10000;
  pointer-events: none;
}

.agent-scope-popover-card {
  min-width: 220px;
  max-width: 280px;
  padding: 14px 16px;
  background: var(--td-bg-color-container);
  border-radius: 10px;
  box-shadow: var(--td-shadow-3), 0 2px 8px rgba(0, 0, 0, 0.06);
  border: 1px solid var(--td-component-stroke);
}

.agent-scope-popover-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin-bottom: 8px;
  line-height: 1.3;
  padding-right: 8px;
}

.agent-scope-popover-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
  margin-bottom: 10px;

  .popover-meta-item {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
}

.agent-scope-popover-permission {
  font-size: 12px;
  margin-bottom: 10px;

  .popover-label {
    color: var(--td-text-color-secondary);
    margin-right: 6px;
  }

  .popover-value {
    color: var(--td-text-color-primary);
    font-weight: 500;
  }
}

.agent-scope-popover-divider {
  height: 1px;
  background: var(--td-bg-color-secondarycontainer);
  margin: 10px 0;
}

.agent-scope-popover-section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  margin-bottom: 8px;
}

.agent-scope-popover-row {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.7;
  padding: 2px 0;
}

.agent-scope-popover-fade-enter-active,
.agent-scope-popover-fade-leave-active {
  transition: opacity 0.12s ease;
}

.agent-scope-popover-fade-enter-from,
.agent-scope-popover-fade-leave-to {
  opacity: 0;
}
</style>
