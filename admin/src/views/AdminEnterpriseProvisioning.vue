<template>
  <section class="enterprise-provisioning">
    <header class="enterprise-provisioning__header">
      <div>
        <div class="enterprise-provisioning__eyebrow">
          <t-icon name="usergroup-add" />
          平台运维
        </div>
        <h1>开通企业</h1>
        <p>
          支付功能暂未接入。系统管理员可为指定用户手工开通企业，用户原有个人工作区和数据保持不变。
        </p>
      </div>
      <t-tag theme="warning" variant="light">临时人工开通</t-tag>
    </header>

    <section class="enterprise-provisioning__toolbar" aria-label="用户检索">
      <t-input
        v-model="keyword"
        class="enterprise-provisioning__search"
        clearable
        placeholder="输入用户名、邮箱或手机号"
        @enter="searchUsers"
      >
        <template #prefix-icon>
          <t-icon name="search" />
        </template>
      </t-input>
      <t-button theme="primary" :loading="loading" @click="searchUsers">
        <template #icon>
          <t-icon name="search" />
        </template>
        搜索用户
      </t-button>
    </section>

    <section class="enterprise-provisioning__results">
      <div class="enterprise-provisioning__section-heading">
        <div>
          <h2>用户列表</h2>
          <span v-if="searched">共 {{ users.length }} 个结果</span>
          <span v-else>输入条件后检索用户</span>
        </div>
        <t-button variant="text" :loading="loading" @click="searchUsers">
          <template #icon>
            <t-icon name="refresh" />
          </template>
          刷新
        </t-button>
      </div>

      <div v-if="loading" class="enterprise-provisioning__state">
        <t-loading size="small" />
        <span>正在检索用户...</span>
      </div>
      <div v-else-if="searched && users.length === 0" class="enterprise-provisioning__state">
        <t-icon name="user-search" />
        <span>没有找到匹配用户</span>
      </div>
      <div v-else-if="!searched" class="enterprise-provisioning__state">
        <t-icon name="search" />
        <span>先搜索需要开通企业的用户</span>
      </div>
      <div v-else class="enterprise-provisioning__table-wrap">
        <table class="enterprise-provisioning__table">
          <thead>
            <tr>
              <th>用户</th>
              <th>个人工作区</th>
              <th>账号状态</th>
              <th class="enterprise-provisioning__action-column">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in users" :key="user.id">
              <td>
                <div class="enterprise-provisioning__user">
                  <t-avatar size="32px" :image="user.avatar">{{ user.username.slice(0, 1) }}</t-avatar>
                  <div>
                    <strong>{{ user.username || '未设置用户名' }}</strong>
                    <span>{{ user.email }}</span>
                  </div>
                </div>
              </td>
              <td>
                <span v-if="user.tenant_id" class="enterprise-provisioning__tenant-id">
                  #{{ user.tenant_id }}
                </span>
                <span v-else class="enterprise-provisioning__muted">未创建</span>
              </td>
              <td>
                <t-tag :theme="user.is_active ? 'success' : 'danger'" variant="light">
                  {{ user.is_active ? '正常' : '已停用' }}
                </t-tag>
                <t-tag v-if="user.is_system_admin" class="enterprise-provisioning__admin-tag" theme="primary" variant="light">
                  系统管理员
                </t-tag>
              </td>
              <td class="enterprise-provisioning__action-column">
                <t-button
                  size="small"
                  variant="outline"
                  :disabled="!user.is_active || !user.tenant_id"
                  @click="openProvisionDialog(user)"
                >
                  <template #icon>
                    <t-icon name="add" />
                  </template>
                  开通企业
                </t-button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <t-dialog
      v-model:visible="dialogVisible"
      header="开通企业"
      :confirm-btn="{ content: '确认开通', loading: submitting }"
      cancel-btn="取消"
      :on-confirm="submitProvisioning"
      @close="closeProvisionDialog"
    >
      <div v-if="selectedUser" class="enterprise-provisioning__dialog">
        <div class="enterprise-provisioning__target">
          <t-avatar size="36px" :image="selectedUser.avatar">{{ selectedUser.username.slice(0, 1) }}</t-avatar>
          <div>
            <strong>{{ selectedUser.username }}</strong>
            <span>{{ selectedUser.email }}</span>
          </div>
        </div>
        <t-alert
          theme="info"
          message="开通后会新增一个企业，并将该用户设为 Owner。用户的个人工作区不会被转换或删除。"
        />
        <t-form layout="vertical" class="enterprise-provisioning__form">
          <t-form-item label="企业名称" required>
            <t-input v-model="form.name" maxlength="128" placeholder="例如：华东销售中心" />
          </t-form-item>
          <t-form-item label="企业描述">
            <t-textarea
              v-model="form.description"
              :autosize="{ minRows: 3, maxRows: 5 }"
              maxlength="512"
              placeholder="可选，用于说明企业的业务或用途"
            />
          </t-form-item>
          <div class="enterprise-provisioning__form-grid">
            <t-form-item label="企业容量（GB）" required>
              <t-input-number
                v-model="form.storageQuotaGB"
                :min="1"
                :max="1048576"
                :step="1"
                theme="column"
              />
            </t-form-item>
            <t-form-item label="企业积分" required>
              <t-input-number
                v-model="form.enterpriseCredits"
                :min="0"
                :max="2147483647"
                :step="1"
                theme="column"
              />
            </t-form-item>
          </div>
        </t-form>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  provisionEnterpriseWorkspace,
  searchSystemUsers,
  type SystemUserSummary,
} from '@/api/system'

const keyword = ref('')
const users = ref<SystemUserSummary[]>([])
const loading = ref(false)
const searched = ref(false)
const dialogVisible = ref(false)
const submitting = ref(false)
const selectedUser = ref<SystemUserSummary | null>(null)
const form = reactive({
  name: '',
  description: '',
  storageQuotaGB: 100,
  enterpriseCredits: 100,
})

function getErrorMessage(error: unknown, fallback: string) {
  if (error && typeof error === 'object' && 'message' in error) {
    const message = (error as { message?: unknown }).message
    if (typeof message === 'string' && message.trim()) return message
  }
  return fallback
}

async function searchUsers() {
  loading.value = true
  try {
    const response = await searchSystemUsers({ keyword: keyword.value, limit: 50 })
    users.value = response.users || []
    searched.value = true
  } catch (error) {
    MessagePlugin.error(getErrorMessage(error, '用户检索失败'))
  } finally {
    loading.value = false
  }
}

function openProvisionDialog(user: SystemUserSummary) {
  selectedUser.value = user
  form.name = `${user.username || '用户'}的企业`
  form.description = ''
  form.storageQuotaGB = 100
  form.enterpriseCredits = 100
  dialogVisible.value = true
}

function closeProvisionDialog() {
  if (submitting.value) return
  dialogVisible.value = false
  selectedUser.value = null
}

async function submitProvisioning() {
  if (!selectedUser.value) return
  const name = form.name.trim()
  if (!name) {
    MessagePlugin.warning('请输入企业名称')
    return
  }
  if (!Number.isInteger(form.storageQuotaGB) || form.storageQuotaGB < 1) {
    MessagePlugin.warning('请输入大于 0 的企业容量')
    return
  }
  if (!Number.isInteger(form.enterpriseCredits) || form.enterpriseCredits < 0) {
    MessagePlugin.warning('请输入不小于 0 的企业积分')
    return
  }

  submitting.value = true
  try {
    const response = await provisionEnterpriseWorkspace({
      user_id: selectedUser.value.id,
      name,
      description: form.description.trim(),
      storage_quota_gb: form.storageQuotaGB,
      enterprise_credits: form.enterpriseCredits,
    })
    MessagePlugin.success(
      response.already_exists
        ? '该用户已有企业，已返回原企业'
        : '企业已开通',
    )
    dialogVisible.value = false
    selectedUser.value = null
    await searchUsers()
  } catch (error) {
    MessagePlugin.error(getErrorMessage(error, '企业开通失败'))
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.enterprise-provisioning {
  display: grid;
  gap: 18px;
  padding: 28px 30px 36px;
}

.enterprise-provisioning__header,
.enterprise-provisioning__toolbar,
.enterprise-provisioning__results {
  border: 1px solid var(--admin-border);
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.enterprise-provisioning__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
  padding: 24px 26px;
}

.enterprise-provisioning__eyebrow {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--admin-brand);
  font-size: 12px;
  font-weight: 650;
  letter-spacing: 0.04em;
  text-transform: uppercase;
}

.enterprise-provisioning h1,
.enterprise-provisioning h2,
.enterprise-provisioning p {
  margin: 0;
}

.enterprise-provisioning h1 {
  margin-top: 8px;
  color: var(--admin-text);
  font-size: 24px;
  line-height: 1.25;
}

.enterprise-provisioning__header p {
  max-width: 700px;
  margin-top: 8px;
  color: var(--admin-text-secondary);
  font-size: 14px;
  line-height: 1.7;
}

.enterprise-provisioning__toolbar {
  display: flex;
  gap: 10px;
  padding: 16px 18px;
}

.enterprise-provisioning__search {
  max-width: 520px;
}

.enterprise-provisioning__results {
  min-width: 0;
}

.enterprise-provisioning__section-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--admin-border);
}

.enterprise-provisioning__section-heading h2 {
  color: var(--admin-text);
  font-size: 16px;
  line-height: 1.4;
}

.enterprise-provisioning__section-heading span {
  display: block;
  margin-top: 3px;
  color: var(--admin-text-muted);
  font-size: 12px;
}

.enterprise-provisioning__state {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  gap: 9px;
  color: var(--admin-text-muted);
  font-size: 14px;
}

.enterprise-provisioning__table-wrap {
  overflow-x: auto;
}

.enterprise-provisioning__table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.enterprise-provisioning__table th,
.enterprise-provisioning__table td {
  padding: 14px 20px;
  border-bottom: 1px solid var(--admin-border);
  text-align: left;
  vertical-align: middle;
}

.enterprise-provisioning__table tr:last-child td {
  border-bottom: 0;
}

.enterprise-provisioning__table th {
  color: var(--admin-text-muted);
  font-size: 12px;
  font-weight: 600;
}

.enterprise-provisioning__table td {
  color: var(--admin-text-secondary);
  font-size: 13px;
}

.enterprise-provisioning__user,
.enterprise-provisioning__target {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.enterprise-provisioning__user > div,
.enterprise-provisioning__target > div {
  display: grid;
  min-width: 0;
  gap: 3px;
}

.enterprise-provisioning__user strong,
.enterprise-provisioning__target strong {
  overflow: hidden;
  color: var(--admin-text);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.enterprise-provisioning__user span,
.enterprise-provisioning__target span {
  overflow: hidden;
  color: var(--admin-text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.enterprise-provisioning__tenant-id {
  color: var(--admin-text);
  font-variant-numeric: tabular-nums;
}

.enterprise-provisioning__muted {
  color: var(--admin-text-muted);
}

.enterprise-provisioning__admin-tag {
  margin-left: 6px;
}

.enterprise-provisioning__action-column {
  width: 180px;
  text-align: right !important;
}

.enterprise-provisioning__dialog {
  display: grid;
  gap: 18px;
}

.enterprise-provisioning__form {
  display: grid;
  gap: 2px;
}

.enterprise-provisioning__form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.enterprise-provisioning__target {
  padding-bottom: 2px;
}

@media (max-width: 760px) {
  .enterprise-provisioning {
    padding: 18px 14px 28px;
  }

  .enterprise-provisioning__header {
    align-items: stretch;
    flex-direction: column;
    gap: 14px;
    padding: 20px;
  }

  .enterprise-provisioning__toolbar {
    flex-direction: column;
  }

  .enterprise-provisioning__search {
    max-width: none;
  }

  .enterprise-provisioning__table {
    min-width: 720px;
  }

  .enterprise-provisioning__form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
