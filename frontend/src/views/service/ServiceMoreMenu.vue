<template>
  <div class="service-hub-more-menu-host">
    <t-dialog
      v-model:visible="infoDialogVisible"
      :header="infoDialogMode === 'name' ? '修改服务名称' : '修改服务描述'"
      width="520px"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: saving }"
      :cancel-btn="{ content: '取消' }"
      @confirm="saveServiceInfo"
    >
      <div class="service-info-dialog">
        <t-input
          v-if="infoDialogMode === 'name'"
          v-model="nameDraft"
          :maxlength="255"
          placeholder="服务名称"
        />
        <t-textarea
          v-else
          v-model="descriptionDraft"
          :maxlength="500"
          :autosize="{ minRows: 4, maxRows: 8 }"
          placeholder="说明这个服务负责什么、适合谁使用"
        />
        <p>
          {{ infoDialogMode === 'name'
            ? '名称会同步显示在服务列表与左侧导航。'
            : '描述会显示在服务列表卡片上，帮助成员快速判断服务边界。' }}
        </p>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="shareDialogVisible"
      header="分享服务"
      width="520px"
      :confirm-btn="{ content: '保存', theme: 'primary', loading: saving }"
      :cancel-btn="{ content: '取消' }"
      @confirm="saveVisibility"
    >
      <div class="service-share-dialog">
        <button
          v-for="option in shareOptions"
          :key="option.value"
          type="button"
          :class="{ selected: visibilityDraft === option.value }"
          @click="visibilityDraft = option.value"
        >
          <span>
            <strong>{{ option.label }}</strong>
            <small>{{ option.description }}</small>
          </span>
          <t-icon :name="visibilityDraft === option.value ? 'check-circle-filled' : 'circle'" />
        </button>
      </div>
    </t-dialog>

    <t-popup v-model="menuVisible" trigger="click" placement="bottom-right">
      <button type="button" class="service-hub-circle-button" title="更多操作" aria-label="更多操作">
        <t-icon name="more" />
      </button>
      <template #content>
        <div class="service-hub-space-menu">
          <template v-if="level === 'primary'">
            <button type="button" @click="openInfoDialog('name')">修改服务名称</button>
            <button type="button" @click="openInfoDialog('description')">修改服务描述</button>
            <button type="button" @click="openShareDialog">分享服务</button>
            <span class="service-hub-space-menu-divider" />
            <button type="button" class="has-trailing" @click="level = 'more'">
              更多
              <t-icon name="chevron-right" />
            </button>
          </template>
          <template v-else>
            <button type="button" class="has-leading" @click="level = 'primary'">
              <t-icon name="chevron-left" />
              更多
            </button>
            <span class="service-hub-space-menu-divider" />
            <button type="button" @click="navigate('overview')">服务概览</button>
            <button type="button" @click="navigate('activity')">动态</button>
            <button type="button" @click="navigate('settings/basic')">服务设置</button>
            <span class="service-hub-space-menu-divider" />
            <button type="button" @click="togglePause">
              {{ service.state === 'paused' ? '恢复服务' : '暂停服务' }}
            </button>
            <button type="button" @click="archiveService">
              {{ service.state === 'archived' ? '恢复归档服务' : '归档服务' }}
            </button>
            <button type="button" class="is-danger" @click="removeService">删除服务</button>
          </template>
        </div>
      </template>
    </t-popup>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  deleteServiceSpace,
  setServiceSpaceState,
  updateServiceSpace,
} from '@/api/service'
import type { ServiceRecord } from './serviceHubState'

type DetailSuffix = 'overview' | 'activity' | 'settings/basic'

const props = defineProps<{
  visible: boolean
  service: ServiceRecord
}>()
const emit = defineEmits<{
  'update:visible': [visible: boolean]
  navigate: [suffix: DetailSuffix]
  updated: [patch: Partial<ServiceRecord>]
  archived: [state: ServiceRecord['state']]
  removed: []
}>()

const level = ref<'primary' | 'more'>('primary')
const infoDialogVisible = ref(false)
const infoDialogMode = ref<'name' | 'description'>('name')
const nameDraft = ref('')
const descriptionDraft = ref('')
const shareDialogVisible = ref(false)
const visibilityDraft = ref<'private' | 'tenant'>('private')
const saving = ref(false)
const menuVisible = computed({
  get: () => props.visible,
  set: (visible) => emit('update:visible', visible),
})
const shareOptions = [
  { value: 'private' as const, label: '仅自己可见', description: '只有服务拥有者可以进入和查看内容' },
  { value: 'tenant' as const, label: '当前空间成员可见', description: '当前工作空间内的成员都可以发现并访问这个服务' },
]

const closeMenu = () => {
  menuVisible.value = false
  level.value = 'primary'
}
const openInfoDialog = (mode: 'name' | 'description') => {
  closeMenu()
  infoDialogMode.value = mode
  nameDraft.value = props.service.name
  descriptionDraft.value = props.service.description || ''
  infoDialogVisible.value = true
}
const openShareDialog = () => {
  closeMenu()
  visibilityDraft.value = props.service.visibility || 'private'
  shareDialogVisible.value = true
}
const saveServiceInfo = async () => {
  const name = nameDraft.value.trim()
  if (infoDialogMode.value === 'name' && !name) {
    MessagePlugin.warning('服务名称不能为空')
    return
  }
  saving.value = true
  try {
    const response = await updateServiceSpace(props.service.id, infoDialogMode.value === 'name'
      ? { name }
      : { description: descriptionDraft.value.trim() })
    if (response?.data) {
      emit('updated', {
        name: response.data.name,
        description: response.data.description || '',
      })
    }
    infoDialogVisible.value = false
    MessagePlugin.success('服务信息已更新')
  } catch (error) {
    console.error('[ServiceMoreMenu] Failed to update service info:', error)
    MessagePlugin.error('服务信息保存失败')
  } finally {
    saving.value = false
  }
}
const saveVisibility = async () => {
  saving.value = true
  try {
    const response = await updateServiceSpace(props.service.id, { visibility: visibilityDraft.value })
    emit('updated', { visibility: response?.data?.visibility || visibilityDraft.value })
    shareDialogVisible.value = false
    MessagePlugin.success('分享范围已更新')
  } catch (error) {
    console.error('[ServiceMoreMenu] Failed to update service visibility:', error)
    MessagePlugin.error('分享范围保存失败')
  } finally {
    saving.value = false
  }
}
const navigate = (suffix: DetailSuffix) => {
  closeMenu()
  emit('navigate', suffix)
}
const togglePause = async () => {
  closeMenu()
  const state = props.service.state === 'paused' ? 'active' : 'paused'
  try {
    await setServiceSpaceState(props.service.id, state)
    emit('updated', { state })
    MessagePlugin.success(state === 'paused' ? '服务已暂停' : '服务已恢复')
  } catch (error) {
    console.error('[ServiceMoreMenu] Failed to toggle service pause:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}
const archiveService = async () => {
  closeMenu()
  const state = props.service.state === 'archived' ? 'active' : 'archived'
  try {
    await setServiceSpaceState(props.service.id, state)
    emit('updated', { state })
    emit('archived', state)
    MessagePlugin.success(state === 'archived' ? '服务已归档' : '服务已恢复')
  } catch (error) {
    console.error('[ServiceMoreMenu] Failed to archive service:', error)
    MessagePlugin.error('服务状态更新失败')
  }
}
const removeService = async () => {
  closeMenu()
  try {
    await deleteServiceSpace(props.service.id)
    emit('removed')
    MessagePlugin.success('服务已删除')
  } catch (error) {
    console.error('[ServiceMoreMenu] Failed to delete service:', error)
    MessagePlugin.error('服务删除失败')
  }
}

watch(() => props.visible, (visible) => {
  if (!visible) level.value = 'primary'
})
</script>
