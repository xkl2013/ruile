<template>
  <section class="skill-admin">
    <div class="skill-admin__header">
      <div>
        <h2>全局 Skill 管理</h2>
        <p>管理平台内所有工作空间和用户可使用的自定义 Skill。</p>
      </div>
      <div class="skill-admin__actions">
        <t-button variant="outline" :loading="loading" @click="loadSkills">
          <template #icon><t-icon name="refresh" /></template>
          刷新
        </t-button>
        <label class="skill-upload" :class="{ 'skill-upload--disabled': uploading }">
          <t-icon name="add" />
          <span>导入 ZIP</span>
          <input
            type="file"
            accept=".zip,application/zip,application/x-zip-compressed"
            :disabled="uploading"
            @change="handleUpload"
          >
        </label>
      </div>
    </div>

    <t-alert theme="info" variant="light">
      ZIP 包必须包含一个带 YAML 头部的 SKILL.md。启用后的 Skill 会对所有工作空间生效。
    </t-alert>

    <t-alert v-if="errorMessage" theme="error" :message="errorMessage">
      <template #operation>
        <t-button size="small" variant="outline" @click="loadSkills">重试</t-button>
      </template>
    </t-alert>

    <section class="skill-admin__panel">
      <div v-if="loading && skills.length === 0" class="skill-empty">
        <t-loading size="small" />
        <span>正在读取全局 Skill...</span>
      </div>
      <div v-else-if="skills.length === 0" class="skill-empty">
        <t-icon name="inbox" />
        <span>暂无全局 Skill</span>
      </div>
      <div v-else class="skill-list">
        <article v-for="skill in skills" :key="skill.id" class="skill-row">
          <div class="skill-row__main">
            <div class="skill-row__title">
              <strong>{{ skill.name }}</strong>
              <t-tag :theme="skill.enabled ? 'success' : 'default'" variant="light" size="small">
                {{ skill.enabled ? '已启用' : '已停用' }}
              </t-tag>
            </div>
            <p>{{ skill.description || '未填写描述' }}</p>
            <small>更新于 {{ formatDate(skill.updated_at) }}</small>
          </div>
          <div class="skill-row__actions">
            <t-switch
              :value="skill.enabled"
              :loading="updatingId === skill.id"
              @change="(value: boolean) => toggleSkill(skill, value)"
            />
            <t-button variant="text" size="small" @click="openFiles(skill)">
              <template #icon><t-icon name="file-paste" /></template>
              文件
            </t-button>
            <t-button theme="danger" variant="text" size="small" @click="removeSkill(skill)">
              <template #icon><t-icon name="delete" /></template>
              删除
            </t-button>
          </div>
        </article>
      </div>
    </section>

    <t-dialog
      v-model:visible="filesVisible"
      :header="selectedSkill?.name || 'Skill 文件'"
      width="760px"
      destroy-on-close
    >
      <div class="file-viewer">
        <div class="file-list">
          <button
            v-for="filePath in files"
            :key="filePath"
            type="button"
            :class="['file-item', { active: filePath === selectedFile }]"
            @click="selectFile(filePath)"
          >
            <t-icon name="file-paste" />
            <span>{{ filePath }}</span>
          </button>
        </div>
        <pre v-if="fileContent !== null" class="file-content">{{ fileContent }}</pre>
        <t-empty v-else description="选择一个文件查看内容" />
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import {
  deleteGlobalSkill,
  listGlobalSkillFiles,
  listGlobalSkills,
  readGlobalSkillFile,
  setGlobalSkillEnabled,
  uploadGlobalSkill,
  type GlobalSkill,
} from '@admin/api/skill'

const skills = ref<GlobalSkill[]>([])
const loading = ref(false)
const uploading = ref(false)
const updatingId = ref('')
const errorMessage = ref('')
const filesVisible = ref(false)
const selectedSkill = ref<GlobalSkill | null>(null)
const files = ref<string[]>([])
const selectedFile = ref('')
const fileContent = ref<string | null>(null)

async function loadSkills() {
  if (loading.value) return
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await listGlobalSkills()
    skills.value = response?.data || []
  } catch (error: any) {
    errorMessage.value = error?.message || '全局 Skill 读取失败'
    skills.value = []
  } finally {
    loading.value = false
  }
}

async function handleUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  if (!file.name.toLowerCase().endsWith('.zip')) {
    MessagePlugin.warning('请选择 ZIP 格式的 Skill 包')
    return
  }
  uploading.value = true
  try {
    await uploadGlobalSkill(file)
    MessagePlugin.success('全局 Skill 导入成功')
    await loadSkills()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '全局 Skill 导入失败')
  } finally {
    uploading.value = false
  }
}

async function toggleSkill(skill: GlobalSkill, enabled: boolean) {
  updatingId.value = skill.id
  try {
    await setGlobalSkillEnabled(skill.id, enabled)
    skill.enabled = enabled
    MessagePlugin.success(enabled ? 'Skill 已对全员启用' : 'Skill 已对全员停用')
  } catch (error: any) {
    MessagePlugin.error(error?.message || 'Skill 状态更新失败')
  } finally {
    updatingId.value = ''
  }
}

function removeSkill(skill: GlobalSkill) {
  const dialog = DialogPlugin.confirm({
    header: '删除全局 Skill',
    body: `确认删除「${skill.name}」？删除后所有工作空间都不能再使用它。`,
    onConfirm: async () => {
      dialog.destroy()
      try {
        await deleteGlobalSkill(skill.id)
        MessagePlugin.success('全局 Skill 已删除')
        await loadSkills()
      } catch (error: any) {
        MessagePlugin.error(error?.message || '全局 Skill 删除失败')
      }
    },
    onCancel: () => dialog.destroy(),
  })
}

async function openFiles(skill: GlobalSkill) {
  selectedSkill.value = skill
  selectedFile.value = ''
  fileContent.value = null
  filesVisible.value = true
  try {
    const response = await listGlobalSkillFiles(skill.id)
    files.value = response?.data || []
    if (files.value.length > 0) await selectFile(files.value[0])
  } catch (error: any) {
    filesVisible.value = false
    MessagePlugin.error(error?.message || 'Skill 文件读取失败')
  }
}

async function selectFile(filePath: string) {
  if (!selectedSkill.value) return
  selectedFile.value = filePath
  try {
    fileContent.value = await readGlobalSkillFile(selectedSkill.value.id, filePath)
  } catch (error: any) {
    fileContent.value = null
    MessagePlugin.error(error?.message || 'Skill 文件内容读取失败')
  }
}

function formatDate(value?: string) {
  if (!value) return '未记录'
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

onMounted(() => {
  void loadSkills()
})
</script>

<style scoped>
.skill-admin {
  display: grid;
  gap: 16px;
  width: min(100%, 1180px);
}

.skill-admin__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.skill-admin__header h2 {
  margin: 0 0 6px;
  color: var(--admin-text);
  font-size: 22px;
  font-weight: 650;
  line-height: 1.35;
}

.skill-admin__header p {
  margin: 0;
  color: var(--admin-text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.skill-admin__actions {
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

.skill-upload {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 14px;
  color: #fff;
  background: var(--td-brand-color);
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
}

.skill-upload input {
  display: none;
}

.skill-upload--disabled {
  cursor: not-allowed;
  opacity: 0.6;
  pointer-events: none;
}

.skill-admin__panel {
  overflow: hidden;
  border: 1px solid var(--admin-border);
  border-radius: 8px;
  background: var(--admin-surface);
  box-shadow: var(--admin-shadow-sm);
}

.skill-empty {
  display: flex;
  min-height: 260px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--admin-text-secondary);
}

.skill-list {
  display: grid;
}

.skill-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 18px 20px;
  border-bottom: 1px solid var(--admin-border);
}

.skill-row:last-child {
  border-bottom: 0;
}

.skill-row__main {
  min-width: 0;
}

.skill-row__title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.skill-row__title strong {
  color: var(--admin-text);
  font-size: 15px;
}

.skill-row__main p {
  margin: 7px 0 4px;
  color: var(--admin-text-secondary);
  line-height: 1.5;
}

.skill-row__main small {
  color: var(--admin-text-tertiary);
}

.skill-row__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.file-viewer {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  gap: 14px;
  min-height: 380px;
}

.file-list {
  overflow-y: auto;
  border-right: 1px solid var(--admin-border);
}

.file-item {
  display: flex;
  width: 100%;
  align-items: center;
  gap: 7px;
  padding: 8px 10px;
  color: var(--admin-text-secondary);
  background: transparent;
  border: 0;
  border-radius: 4px;
  cursor: pointer;
  text-align: left;
}

.file-item span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-item:hover,
.file-item.active {
  color: var(--td-brand-color);
  background: var(--admin-surface-muted);
}

.file-content {
  max-height: 540px;
  margin: 0;
  overflow: auto;
  padding: 12px;
  color: var(--admin-text);
  background: var(--admin-surface-muted);
  border-radius: 4px;
  font: 12px/1.6 ui-monospace, SFMono-Regular, Menlo, monospace;
  white-space: pre-wrap;
  word-break: break-word;
}

@media (max-width: 760px) {
  .skill-admin__header,
  .skill-row {
    flex-direction: column;
    align-items: stretch;
  }

  .skill-admin__actions,
  .skill-row__actions {
    flex-wrap: wrap;
  }

  .file-viewer {
    grid-template-columns: 1fr;
  }

  .file-list {
    max-height: 150px;
    border-right: 0;
    border-bottom: 1px solid var(--admin-border);
  }
}
</style>
