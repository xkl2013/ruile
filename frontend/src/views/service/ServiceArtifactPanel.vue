<template>
  <div class="service-hub-side-panel-body">
    <div v-if="loading" class="service-hub-panel-empty">正在加载服务文件。</div>
    <div v-else-if="error" class="service-hub-panel-empty service-hub-panel-error">{{ error }}</div>
    <button
      v-for="artifact in artifacts"
      :key="artifact.id"
      type="button"
      class="service-hub-side-artifact"
      :class="{ active: selectedArtifact?.id === artifact.id }"
      @click="selectedArtifact = artifact"
    >
      <span class="service-hub-artifact-icon">{{ artifact.format || artifact.kind }}</span>
      <span>
        <strong>{{ artifact.title || artifact.original_name || '未命名文件' }}</strong>
        <small>v{{ artifact.version }} · {{ artifact.lifecycle }}</small>
      </span>
    </button>
    <div v-if="!artifacts.length && !loading" class="service-hub-panel-empty">当前服务还没有文件。</div>
    <div v-if="selectedArtifact" class="service-hub-artifact-detail">
      <strong>{{ selectedArtifact.title || selectedArtifact.original_name || '未命名文件' }}</strong>
      <p>{{ selectedArtifact.summary || '该文件暂无摘要。' }}</p>
      <div class="service-hub-artifact-links">
        <a :href="artifactStreamUrl(selectedArtifact, 'preview')" target="_blank" rel="noreferrer">预览</a>
        <a
          v-if="selectedArtifact.downloadable"
          :href="artifactStreamUrl(selectedArtifact, 'download')"
          target="_blank"
          rel="noreferrer"
        >
          下载
        </a>
        <button
          v-if="selectedArtifact.lifecycle === 'temporary'"
          type="button"
          @click="changeLifecycle(selectedArtifact, 'saved')"
        >
          保存
        </button>
        <button
          v-if="selectedArtifact.lifecycle !== 'archived'"
          type="button"
          class="is-danger"
          @click="changeLifecycle(selectedArtifact, 'archived')"
        >
          归档
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  listServiceArtifacts,
  updateServiceArtifactLifecycle,
  type ServiceArtifact,
} from '@/api/service'

const props = defineProps<{ serviceId: string }>()
const emit = defineEmits<{ 'count-change': [count: number] }>()
const artifacts = ref<ServiceArtifact[]>([])
const selectedArtifact = ref<ServiceArtifact | null>(null)
const loading = ref(false)
const error = ref('')

const artifactStreamUrl = (artifact: ServiceArtifact, mode: 'preview' | 'download') =>
  `/api/v1/services/${encodeURIComponent(artifact.service_id)}/artifacts/${encodeURIComponent(artifact.artifact_id)}/${mode}`

const createIdempotencyKey = () =>
  typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `artifact-${Date.now()}-${Math.random().toString(16).slice(2)}`

const loadArtifacts = async () => {
  if (!props.serviceId) return
  loading.value = true
  error.value = ''
  try {
    const response = await listServiceArtifacts(props.serviceId, { page: 1, page_size: 100 })
    artifacts.value = response?.data?.items || []
    selectedArtifact.value = artifacts.value[0] || null
  } catch (loadError) {
    console.error('[ServiceArtifactPanel] Failed to load service artifacts:', loadError)
    artifacts.value = []
    selectedArtifact.value = null
    error.value = '服务文件暂不可用，请稍后重试'
  } finally {
    loading.value = false
  }
}

const changeLifecycle = async (
  artifact: ServiceArtifact,
  lifecycle: ServiceArtifact['lifecycle'],
) => {
  try {
    const response = await updateServiceArtifactLifecycle(
      artifact.service_id,
      artifact.artifact_id,
      { lifecycle, idempotency_key: createIdempotencyKey() },
    )
    if (!response?.data) return
    const index = artifacts.value.findIndex((item) => item.id === response.data.id)
    if (index >= 0) artifacts.value[index] = response.data
    selectedArtifact.value = response.data
    MessagePlugin.success(lifecycle === 'archived' ? '文件已归档' : '文件已保存')
  } catch (updateError) {
    console.error('[ServiceArtifactPanel] Failed to update artifact lifecycle:', updateError)
    MessagePlugin.error('文件状态更新失败，请稍后重试')
  }
}

watch(() => props.serviceId, loadArtifacts, { immediate: true })
watch(artifacts, (items) => emit('count-change', items.length), { deep: true, immediate: true })
</script>
