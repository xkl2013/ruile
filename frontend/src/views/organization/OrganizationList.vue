<template>
  <div class="org-list-container">
    <div class="org-list-content">
      <div class="header" style="--wails-draggable: drag">
        <div class="header-title" style="--wails-draggable: drag">
          <div class="title-row" style="--wails-draggable: drag">
            <h2 style="--wails-draggable: drag">{{ $t('organization.title') }}</h2>
            <div class="header-actions" style="--wails-draggable: no-drag">
              <t-tooltip :content="canManageOrg ? $t('organization.createOrg') : noPermissionTip" placement="bottom">
                <t-button variant="text" theme="default" size="small" class="header-action-btn"
                  style="--wails-draggable: no-drag" :disabled="!canManageOrg" @click="handleCreateOrganization">
                  <template #icon><img src="@/assets/img/organization-green.svg" class="org-create-icon" alt=""
                      aria-hidden="true" /></template>
                </t-button>
              </t-tooltip>
            </div>
          </div>
          <p class="header-subtitle" style="--wails-draggable: drag">{{ $t('organization.subtitle') }}</p>
        </div>
      </div>
      <div class="org-list-main">
        <!-- 骨架屏占位 -->
        <div v-if="loading && displayOrganizations.length === 0" class="org-card-wrap">
          <div v-for="n in 4" :key="'skel-' + n" class="org-card org-card-skeleton">
            <div class="card-header">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '36px', height: '36px', type: 'circle' }, { width: '50%', height: '20px' }]]" />
            </div>
            <div style="flex:1;margin-top:12px">
              <t-skeleton animation="gradient"
                :row-col="[{ width: '100%', height: '14px' }, { width: '70%', height: '14px' }]" />
            </div>
            <div style="margin-top:auto">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '60px', height: '22px', type: 'rect' }, { width: '60px', height: '22px', type: 'rect' }]]" />
            </div>
          </div>
        </div>

        <!-- 卡片网格 -->
        <div v-if="displayOrganizations.length > 0" class="org-card-wrap">
          <template v-for="org in displayOrganizations" :key="org.id">
            <div class="org-card"
            :class="{ 'joined-org': !org.is_owner }" @click="handleCardClick(org)">
            <!-- 装饰：协作网络感图形 -->
            <div class="card-decoration">
              <svg class="card-deco-svg" width="56" height="40" viewBox="0 0 56 40" fill="none"
                xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
                <circle cx="10" cy="12" r="4" stroke="currentColor" stroke-width="1.5" fill="none" opacity="0.5" />
                <circle cx="28" cy="8" r="5" stroke="currentColor" stroke-width="1.8" fill="none" opacity="0.7" />
                <circle cx="46" cy="14" r="4" stroke="currentColor" stroke-width="1.5" fill="none" opacity="0.5" />
                <path d="M14 13 L24 10 M32 10 L42 13" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"
                  opacity="0.4" />
                <circle cx="28" cy="28" r="6" stroke="currentColor" stroke-width="1.2" fill="none" opacity="0.35" />
                <path d="M28 14 L28 22 M20 18 L26 24 M36 18 L30 24" stroke="currentColor" stroke-width="1"
                  stroke-linecap="round" opacity="0.3" />
              </svg>
            </div>

            <!-- 卡片头部 -->
            <div class="card-header">
              <div class="card-header-left">
                <div class="org-avatar">
                  <SpaceAvatar :name="org.name" :avatar="org.avatar" size="small" />
                </div>
                <div class="card-title-block">
                  <span class="card-title" :title="org.name">{{ org.name }}</span>
                </div>
              </div>
              <t-popup v-model="org.showMore" overlayClassName="card-more-popup"
                :on-visible-change="(visible: boolean) => onVisibleChange(visible, org)" trigger="click"
                destroy-on-close placement="bottom-right">
                <div class="more-wrap" @click.stop :class="{ 'active-more': org.showMore }">
                  <img class="more-icon" src="@/assets/img/more.png" alt="" />
                </div>
                <template #content>
                  <div class="popup-menu" @click.stop>
                    <div class="popup-menu-item" @click.stop="handleSettings(org)">
                      <t-icon class="menu-icon" name="setting" />
                      <span>{{ $t('organization.settings.editTitle') }}</span>
                    </div>
                    <div v-if="!org.is_owner" class="popup-menu-item delete" @click.stop="handleLeave(org)">
                      <t-icon class="menu-icon" name="logout" />
                      <span>{{ $t('organization.leave') }}</span>
                    </div>
                    <div v-if="org.is_owner && canManageOrg" class="popup-menu-item delete"
                      @click.stop="handleDelete(org)">
                      <t-icon class="menu-icon" name="delete" />
                      <span>{{ $t('common.delete') }}</span>
                    </div>
                  </div>
                </template>
              </t-popup>
            </div>

            <!-- 卡片内容 -->
            <div class="card-content">
              <div class="card-description">
                {{ org.description || $t('organization.noDescription') }}
              </div>
            </div>

            <!-- 卡片底部（与知识库卡片风格统一：小标签、无日期、智能体用主题色） -->
            <div class="card-bottom">
              <div class="bottom-left">
                <div class="feature-badges">
                  <t-tooltip :content="$t('organization.memberCount')" placement="top">
                    <div class="feature-badge stat-member">
                      <t-icon name="user" size="14px" />
                      <span class="badge-count">{{ org.member_count || 0 }}</span>
                    </div>
                  </t-tooltip>
                  <t-tooltip :content="$t('organization.invite.knowledgeBases')" placement="top">
                    <div class="feature-badge stat-kb">
                      <t-icon name="folder" size="14px" />
                      <span class="badge-count">{{ org.share_count ?? 0 }}</span>
                    </div>
                  </t-tooltip>
                  <t-tooltip :content="$t('organization.invite.agents')" placement="top">
                    <div class="feature-badge stat-agent">
                      <img src="@/assets/img/agent-green.svg" class="stat-agent-icon" alt="" aria-hidden="true" />
                      <span class="badge-count">{{ org.agent_share_count ?? 0 }}</span>
                    </div>
                  </t-tooltip>
                </div>
              </div>
              <div class="bottom-right">
                <div class="relation-role-tag" :class="org.is_owner ? 'owner' : (org.my_role || '')">
                  <t-icon :name="org.is_owner ? 'usergroup-add' : 'usergroup'" size="14px" />
                  <span>{{ org.is_owner ? $t('organization.owner') : (org.my_role ?
                    $t(`organization.role.${org.my_role}`) :
                    $t('organization.joinedByMe')) }}</span>
                </div>
              </div>
            </div>
          </div>
          </template>
        </div>

        <!-- 空状态（按筛选显示不同文案） -->
        <div v-else-if="!loading" class="empty-state">
          <img class="empty-img" src="@/assets/img/upload.svg" alt="">
          <span class="empty-txt">{{ $t('organization.empty') }}</span>
          <span class="empty-desc">{{ $t('organization.emptyDesc') }}</span>
          <div class="empty-state-actions">
            <t-tooltip :content="noPermissionTip" placement="top" :disabled="canManageOrg">
              <t-button class="org-create-btn" :disabled="!canManageOrg" @click="handleCreateOrganization">
                <template #icon><img src="@/assets/img/organization-green.svg" class="org-create-icon" alt=""
                    aria-hidden="true" /></template>
                {{ $t('organization.createOrg') }}
              </t-button>
            </t-tooltip>
          </div>
        </div>
      </div>
    </div>

    <!-- Organization Settings Modal (用于创建和编辑组织) -->
    <OrganizationSettingsModal :visible="showSettingsModal" :org-id="settingsOrgId" :mode="settingsMode"
      @update:visible="showSettingsModal = $event" @saved="handleSettingsSaved" />

    <!-- Delete Confirm Dialog -->
    <t-dialog v-model:visible="deleteVisible" dialogClassName="del-org-dialog" :closeBtn="false" :cancelBtn="null"
      :confirmBtn="null">
      <div class="circle-wrap">
        <div class="dialog-header">
          <img class="circle-img" src="@/assets/img/circle.png" alt="">
          <span class="circle-title">{{ $t('organization.deleteConfirmTitle') }}</span>
        </div>
        <span class="del-circle-txt">
          {{ $t('organization.deleteConfirmMessage', { name: deletingOrg?.name ?? '' }) }}
        </span>
        <div class="circle-btn">
          <span class="circle-btn-txt" @click="deleteVisible = false">{{ $t('common.cancel') }}</span>
          <span class="circle-btn-txt confirm" @click="confirmDelete">{{ $t('common.delete') }}</span>
        </div>
      </div>
    </t-dialog>

    <!-- Leave Confirm Dialog -->
    <t-dialog v-model:visible="leaveVisible" dialogClassName="del-org-dialog" :closeBtn="false" :cancelBtn="null"
      :confirmBtn="null">
      <div class="circle-wrap">
        <div class="dialog-header">
          <img class="circle-img" src="@/assets/img/circle.png" alt="">
          <span class="circle-title">{{ $t('organization.leaveConfirmTitle') }}</span>
        </div>
        <span class="del-circle-txt">
          {{ $t('organization.leaveConfirmMessage', { name: leavingOrg?.name ?? '' }) }}
        </span>
        <div class="circle-btn">
          <span class="circle-btn-txt" @click="leaveVisible = false">{{ $t('common.cancel') }}</span>
          <span class="circle-btn-txt confirm" @click="confirmLeave">{{ $t('organization.leave') }}</span>
        </div>
      </div>
    </t-dialog>

  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import { useOrganizationStore } from '@/stores/organization'
import { useAuthStore } from '@/stores/auth'
import type { Organization } from '@/api/organization'
import { useI18n } from 'vue-i18n'
import OrganizationSettingsModal from './OrganizationSettingsModal.vue'
import SpaceAvatar from '@/components/SpaceAvatar.vue'

interface OrgWithUI extends Organization {
  showMore?: boolean
}

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const orgStore = useOrganizationStore()
const authStore = useAuthStore()

// 后端 /api/v1/organizations 下的写操作（创建、管理员添加参与空间、改设置等）
// 在路由层都要求当前空间角色 ≥ admin。前端只用于 UI 渲染，安全边界仍在服务端。
const canManageOrg = computed(
  () => authStore.hasRole('admin') || authStore.canAccessAllTenants
)
const noPermissionTip = computed(() => t('organization.rbac.needTenantAdminTip'))

// State
const showSettingsModal = ref(false)
const settingsOrgId = ref('')
const settingsMode = ref<'create' | 'edit'>('edit')
const deleteVisible = ref(false)
const leaveVisible = ref(false)
const deletingOrg = ref<Organization | null>(null)
const leavingOrg = ref<Organization | null>(null)

// 监听菜单快捷操作事件
const handleOrganizationDialogEvent = ((event: CustomEvent<{ type: 'create' | 'join' }>) => {
  if (!canManageOrg.value) {
    MessagePlugin.warning(t('organization.rbac.cannotCreate'))
    return
  }
  if (event.detail?.type === 'create') {
    // 创建组织使用 SettingsModal
    settingsOrgId.value = ''
    settingsMode.value = 'create'
    showSettingsModal.value = true
  }
}) as EventListener

// Computed
const loading = computed(() => orgStore.loading)
const organizations = ref<OrgWithUI[]>([])
const displayOrganizations = computed(() => [...organizations.value].sort((a, b) => {
  if (a.is_owner === b.is_owner) return 0
  return a.is_owner ? -1 : 1
}))

// Watch store changes and update local organizations
watch(
  () => orgStore.organizations,
  (newOrgs) => {
    organizations.value = newOrgs.map(org => ({ ...org, showMore: false }))
  },
  { immediate: true }
)

// Methods
function getRoleTheme(role: string) {
  switch (role) {
    case 'admin': return 'primary'
    case 'editor': return 'warning'
    default: return 'default'
  }
}

const onVisibleChange = (visible: boolean, org: OrgWithUI) => {
  if (!visible) {
    org.showMore = false
  }
}

// 创建组织
function handleCreateOrganization() {
  if (!canManageOrg.value) {
    MessagePlugin.warning(t('organization.rbac.cannotCreate'))
    return
  }
  settingsOrgId.value = ''
  settingsMode.value = 'create'
  showSettingsModal.value = true
}

function handleCardClick(org: OrgWithUI) {
  // 如果弹窗正在显示，不触发设置
  if (org.showMore) {
    return
  }
  settingsOrgId.value = org.id
  settingsMode.value = 'edit'
  showSettingsModal.value = true
}

function handleSettingsSaved() {
  orgStore.fetchOrganizations()
}


function handleSettings(org: OrgWithUI) {
  org.showMore = false
  settingsOrgId.value = org.id
  settingsMode.value = 'edit'
  showSettingsModal.value = true
}

function handleLeave(org: OrgWithUI) {
  org.showMore = false
  leavingOrg.value = org
  leaveVisible.value = true
}

async function confirmLeave() {
  if (!leavingOrg.value) return
  const success = await orgStore.leave(leavingOrg.value.id)
  if (success) {
    MessagePlugin.success(t('organization.leaveSuccess'))
    leaveVisible.value = false
    leavingOrg.value = null
  } else {
    MessagePlugin.error(orgStore.error || t('organization.leaveFailed'))
  }
}

function handleDelete(org: OrgWithUI) {
  org.showMore = false
  deletingOrg.value = org
  deleteVisible.value = true
}

async function confirmDelete() {
  if (!deletingOrg.value) return
  if (!canManageOrg.value) {
    MessagePlugin.warning(t('organization.rbac.cannotManage'))
    return
  }
  const success = await orgStore.remove(deletingOrg.value.id)
  if (success) {
    MessagePlugin.success(t('organization.deleteSuccess'))
    deleteVisible.value = false
    deletingOrg.value = null
  } else {
    MessagePlugin.error(orgStore.error || t('organization.deleteFailed'))
  }
}

// Lifecycle
onMounted(async () => {
  orgStore.fetchOrganizations()
  window.addEventListener('openOrganizationDialog', handleOrganizationDialogEvent)

  // 检查 URL 中是否有 orgId，如果有则打开空间设置
  const orgId = route.query.orgId as string
  if (orgId) {
    settingsOrgId.value = orgId
    settingsMode.value = 'edit'
    showSettingsModal.value = true
    // 清除 URL 中的 orgId 参数，避免刷新时重复打开
    const newQuery = { ...route.query }
    delete newQuery.orgId
    router.replace({ path: route.path, query: newQuery })
  }
})

onUnmounted(() => {
  window.removeEventListener('openOrganizationDialog', handleOrganizationDialogEvent)
})
</script>

<style scoped lang="less">
.org-list-container {
  margin: 0 16px 0 0;
  height: 100%;
  box-sizing: border-box;
  flex: 1;
  display: flex;
  position: relative;
  min-height: 0;
}

.org-list-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 20px 28px 0 28px;
}

.org-list-main {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px 0;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  flex-shrink: 0;

  .header-title {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 24px;
    font-weight: 600;
    line-height: 32px;
  }
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.org-create-btn {
  background: var(--td-brand-color);
  border: none;
  color: var(--td-text-color-anti);
  font-weight: 500;
  box-shadow: 0 2px 8px rgba(7, 192, 95, 0.25);
  transition: all 0.25s ease;

  &:hover {
    background: var(--td-brand-color);
    box-shadow: 0 4px 14px rgba(7, 192, 95, 0.35);
  }

  .org-create-icon {
    width: 16px;
    height: 16px;
    filter: brightness(0) invert(1);
  }
}

.header-subtitle {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.header-action-btn {
  padding: 0 !important;
  min-width: 28px !important;
  width: 28px !important;
  height: 28px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  background: var(--td-bg-color-secondarycontainer) !important;
  border: 1px solid var(--td-component-stroke) !important;
  border-radius: 6px !important;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: background 0.2s, border-color 0.2s, color 0.2s;

  &:hover {
    background: var(--td-bg-color-secondarycontainer) !important;
    border-color: var(--td-component-stroke) !important;
    color: var(--td-text-color-primary);
  }

  :deep(.t-icon),
  :deep(.btn-icon-wrapper),
  :deep(.org-create-icon) {
    color: var(--td-brand-color);
  }

  :deep(.org-create-icon) {
    width: 16px;
    height: 16px;
  }
}

// Tab 切换样式（下划线式，与整体协作感一致）
.org-tabs {
  display: flex;
  align-items: center;
  gap: 28px;
  border-bottom: 1px solid var(--td-component-stroke);
  margin-bottom: 24px;

  .tab-item {
    padding: 12px 0;
    cursor: pointer;
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    user-select: none;
    position: relative;
    transition: color 0.2s ease;

    &:hover {
      color: var(--td-text-color-secondary);
    }

    &.active {
      color: var(--td-brand-color);
      font-weight: 500;

      &::after {
        content: '';
        position: absolute;
        bottom: -1px;
        left: 0;
        right: 0;
        height: 2px;
        background: var(--td-brand-color);
        border-radius: 1px;
      }
    }
  }
}

@keyframes contentFadeIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.org-card-wrap {
  display: grid;
  gap: 12px;
  grid-template-columns: 1fr;
  animation: contentFadeIn 0.32s ease-out;
}

// 共享空间分组标题——与 KB / Agent 列表口径完全一致（图标 + 名称 + 数量 + 折叠 chevron）。
.org-section-header {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 6px;
  // 整行只用来铺背景；点击靠子元素冒泡，避免点到标题右侧空白误折叠。
  pointer-events: none;

  & > * {
    pointer-events: auto;
  }
  position: sticky;
  top: 0;
  z-index: 5;
  background: var(--td-bg-color-container);
  box-shadow: 0 -8px 0 0 var(--td-bg-color-container),
    0 4px 0 0 var(--td-bg-color-container);
  padding: 6px 4px 6px 0;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  cursor: pointer;
  user-select: none;
  outline: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    box-shadow: 0 0 0 2px var(--td-brand-color-focus, rgba(0, 82, 217, 0.2));
  }

  .t-icon {
    color: inherit;
  }

  .org-section-toggle {
    margin-left: 4px;
    opacity: 0.7;
    transition: opacity 0.15s ease;
  }

  .org-section-count {
    margin-left: 2px;
    padding: 0 6px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 11px;
    line-height: 16px;
    font-weight: 500;
  }

  &:hover .org-section-toggle {
    opacity: 1;
  }
}

.org-card-skeleton {
  cursor: default;
  display: flex;
  flex-direction: column;
  height: 136px;
  min-height: 136px;
}

/* 与知识库 / 智能体列表统一：紧凑 + 1px 描边 */
.org-card {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  background: var(--td-bg-color-container);
  position: relative;
  cursor: pointer;
  transition: border-color 0.25s ease, box-shadow 0.25s ease, transform 0.2s ease;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  height: 136px;
  min-height: 136px;

  &::before {
    content: '';
    position: absolute;
    top: 0;
    right: 0;
    width: 120px;
    height: 80px;
    background: radial-gradient(ellipse 60% 50% at 100% 0%, rgba(7, 192, 95, 0.06) 0%, transparent 70%);
    pointer-events: none;
    z-index: 0;
  }

  &.joined-org {
    &:hover {
      border-color: rgba(7, 192, 95, 0.4);
      box-shadow: 0 4px 16px rgba(7, 192, 95, 0.08);
    }
  }

  &:hover {
    border-color: rgba(7, 192, 95, 0.5);
    box-shadow: 0 6px 20px rgba(7, 192, 95, 0.12);
  }

  .card-decoration {
    color: rgba(7, 192, 95, 0.35);
  }

  &:hover .card-decoration {
    color: rgba(7, 192, 95, 0.55);
  }

  .card-header {
    position: relative;
    z-index: 2;
    margin-bottom: 6px;
  }

  .card-title {
    font-size: 15px;
    line-height: 22px;
  }

  .card-content {
    position: relative;
    z-index: 1;
    margin-bottom: 6px;
  }

  .card-bottom {
    position: relative;
    z-index: 1;
    padding-top: 6px;
  }

  .card-description {
    font-size: 12px;
    line-height: 17px;
  }

  .more-wrap {
    width: 28px;
    height: 28px;
    border-radius: 8px;

    .more-icon {
      width: 16px;
      height: 16px;
    }
  }
}

// 卡片装饰：协作网络图形
.card-decoration {
  position: absolute;
  top: 8px;
  right: 14px;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 0;
  transition: color 0.3s ease;

  .card-deco-svg {
    display: block;
    width: 56px;
    height: 40px;
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  position: relative;
  z-index: 2;
}

.card-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

// 空间头像容器（SpaceAvatar 自带样式）
.org-avatar {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.card-title-block {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.card-title {
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
  letter-spacing: 0.01em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.more-wrap {
  display: flex;
  width: 28px;
  height: 28px;
  justify-content: center;
  align-items: center;
  border-radius: 8px;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s ease;
  opacity: 0;

  .org-card:hover & {
    opacity: 0.6;
  }

  &:hover {
    background: var(--td-bg-color-container-hover);
    opacity: 1 !important;
  }

  &.active-more {
    background: var(--td-bg-color-container-hover);
    opacity: 1 !important;
  }

  .more-icon {
    width: 16px;
    height: 16px;
  }
}

/* 与知识库卡片内容区一致 */
.card-content {
  flex: 1;
  min-height: 0;
  margin-bottom: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* 三个列表卡片统一：描述字体 */
.card-description {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding-top: 8px;
  border-top: .5px solid var(--td-component-stroke);
}

.bottom-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

// 与知识库卡片统一的底部标签：小尺寸、统一圆角
.feature-badges {
  display: flex;
  align-items: center;
  gap: 4px;
}

.feature-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 3px;
  height: 20px;
  padding: 0 5px;
  border-radius: 5px;
  font-size: 11px;
  font-weight: 500;
  font-family: var(--app-font-family);
  cursor: default;
  transition: background 0.2s ease;

  .t-icon {
    flex-shrink: 0;
  }

  .badge-count {
    line-height: 1;
  }

  &.stat-member {
    background: rgba(100, 116, 139, 0.08);
    color: var(--td-text-color-secondary);

    .t-icon {
      color: var(--td-text-color-secondary);
    }

    &:hover {
      background: rgba(100, 116, 139, 0.12);
    }
  }

  &.stat-kb {
    background: rgba(7, 192, 95, 0.08);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }

    &:hover {
      background: rgba(7, 192, 95, 0.12);
    }
  }

  &.stat-agent {
    background: rgba(124, 77, 255, 0.08);
    color: var(--td-brand-color);

    .stat-agent-icon {
      width: 14px;
      height: 14px;
      flex-shrink: 0;
      /* 将绿色 icon 着色为紫色，与标签统一 */
      filter: brightness(0) saturate(100%) invert(48%) sepia(79%) saturate(2476%) hue-rotate(236deg);
    }

    &:hover {
      background: rgba(124, 77, 255, 0.12);
    }
  }
}

// 右下角：创建者/角色 合并标签（带图标）
.bottom-right {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.relation-role-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 22px;
  padding: 0 6px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 500;
  font-family: var(--app-font-family);
  background: rgba(107, 114, 128, 0.08);
  color: var(--td-text-color-secondary);

  .t-icon {
    flex-shrink: 0;
    color: var(--td-text-color-secondary);
  }

  &.owner {
    background: rgba(124, 77, 255, 0.1);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.admin {
    background: rgba(7, 192, 95, 0.12);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.editor {
    background: rgba(7, 192, 95, 0.08);
    color: var(--td-brand-color);

    .t-icon {
      color: var(--td-brand-color);
    }
  }

  &.viewer {
    background: rgba(107, 114, 128, 0.08);
    color: var(--td-text-color-secondary);

    .t-icon {
      color: var(--td-text-color-secondary);
    }
  }
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 60px 20px;

  .empty-img {
    width: 162px;
    height: 162px;
    margin-bottom: 20px;
  }

  .empty-txt {
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: 16px;
    font-weight: 600;
    line-height: 26px;
    margin-bottom: 8px;
  }

  .empty-desc {
    color: var(--td-text-color-disabled);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    margin-bottom: 0;
  }

  .empty-state-actions {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-top: 20px;
  }
}

// 响应式布局
@media (min-width: 900px) {
  .org-card-wrap {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1250px) {
  .org-card-wrap {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (min-width: 1600px) {
  .org-card-wrap {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (min-width: 1900px) {
  .org-card-wrap {
    grid-template-columns: repeat(5, 1fr);
  }
}

@media (min-width: 2200px) {
  .org-card-wrap {
    grid-template-columns: repeat(6, 1fr);
  }
}

// 删除/离开确认对话框样式
:deep(.del-org-dialog) {
  padding: 0px !important;
  border-radius: 6px !important;

  .t-dialog__header {
    display: none;
  }

  .t-dialog__body {
    padding: 16px;
  }

  .t-dialog__footer {
    padding: 0;
  }
}

:deep(.t-dialog__position.t-dialog--top) {
  padding-top: 40vh !important;
}

.circle-wrap {
  .dialog-header {
    display: flex;
    align-items: center;
    margin-bottom: 8px;
  }

  .circle-img {
    width: 20px;
    height: 20px;
    margin-right: 8px;
  }

  .circle-title {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }

  .del-circle-txt {
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    display: inline-block;
    margin-left: 29px;
    margin-bottom: 21px;
  }

  .circle-btn {
    height: 22px;
    width: 100%;
    display: flex;
    justify-content: flex-end;
  }

  .circle-btn-txt {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    cursor: pointer;

    &:hover {
      opacity: 0.8;
    }
  }

  .confirm {
    color: var(--td-error-color);
    margin-left: 40px;

    &:hover {
      opacity: 0.8;
    }
  }
}
</style>

<style lang="less">
/* 下拉菜单样式已统一至 @/assets/dropdown-menu.less */

// 创建对话框样式优化
.create-org-dialog {
  .t-form-item__label {
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .t-input,
  .t-textarea {
    font-family: var(--app-font-family);
  }

}
</style>
