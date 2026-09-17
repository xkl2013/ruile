<template>
  <div class="work-profile">
    <div class="section-header">
      <div class="section-title-row">
        <div>
          <h2>{{ $t('settings.workProfile.title') }}</h2>
          <p class="section-description">{{ $t('settings.workProfile.description') }}</p>
        </div>
        <div class="section-header-actions">
          <span v-if="currentTenantName" class="workspace-name">{{ currentTenantName }}</span>
          <template v-if="!editing">
            <t-button
              theme="default"
              variant="outline"
              size="small"
              class="header-edit-button"
              :title="$t('settings.workProfile.edit')"
              :aria-label="$t('settings.workProfile.edit')"
              @click="startEditing"
            >
              <template #icon>
                <t-icon name="edit" />
              </template>
              {{ $t('settings.workProfile.edit') }}
            </t-button>
          </template>
          <template v-else>
            <t-button
              theme="default"
              variant="outline"
              size="small"
              :disabled="saving || generating"
              @click="cancelEditing"
            >
              {{ $t('settings.workProfile.cancel') }}
            </t-button>
            <t-button
              theme="primary"
              size="small"
              :loading="saving"
              :disabled="!canSave"
              @click="saveProfile"
            >
              {{ $t('settings.workProfile.save') }}
            </t-button>
          </template>
        </div>
      </div>
    </div>

    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>{{ $t('tenant.loadingInfo') }}</span>
    </div>

    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadProfile">{{ $t('tenant.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <div v-else class="settings-group">
      <div class="setting-row">
        <div class="setting-info">
          <label>{{ $t('settings.workProfile.fieldLabel') }}</label>
          <p class="desc">{{ $t('settings.workProfile.fieldDescription') }}</p>
        </div>

        <div class="setting-control work-profile-control">
          <template v-if="!editing">
            <div
              class="description-display"
              :class="{ 'is-empty': !savedDescription }"
            >
              {{ savedDescription || $t('settings.workProfile.empty') }}
            </div>
          </template>

          <template v-else>
            <div class="ai-generation">
              <div class="ai-generation-header">
                <div>
                  <div class="ai-generation-label">{{ $t('settings.workProfile.promptLabel') }}</div>
                  <p class="ai-generation-description">{{ $t('settings.workProfile.promptDescription') }}</p>
                </div>
                <t-button
                  theme="primary"
                  size="small"
                  :loading="generating"
                  :disabled="!canGenerate"
                  @click="generateProfile"
                >
                  <template #icon>
                    <t-icon name="lightbulb" />
                  </template>
                  {{ $t('settings.workProfile.generate') }}
                </t-button>
              </div>
              <t-textarea
                v-model="prompt"
                :maxlength="1000"
                :autosize="{ minRows: 2, maxRows: 5 }"
                :placeholder="$t('settings.workProfile.promptPlaceholder')"
                :disabled="saving || generating"
                autofocus
                @keydown="handleKeydown"
              />
            </div>
            <t-textarea
              v-model="draftDescription"
              :maxlength="2000"
              :autosize="{ minRows: 5, maxRows: 10 }"
              :placeholder="$t('settings.workProfile.placeholder')"
              :disabled="saving || generating"
              @keydown="handleKeydown"
            />
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import {
  generateMyMemberProfile,
  getMyMemberProfile,
  updateMyMemberProfile,
} from '@/api/tenant/members'

const { t } = useI18n()
const authStore = useAuthStore()

const activeTenantId = computed(() => Number(authStore.effectiveTenantId || 0))
const currentTenantName = computed(() => authStore.currentTenantName || '')
const loading = ref(false)
const saving = ref(false)
const generating = ref(false)
const error = ref('')
const editing = ref(false)
const savedDescription = ref('')
const draftDescription = ref('')
const prompt = ref('')

const canSave = computed(() =>
  draftDescription.value.trim().length > 0 && !saving.value && !generating.value,
)
const canGenerate = computed(() =>
  prompt.value.trim().length > 0 && !saving.value && !generating.value,
)

async function loadProfile() {
  const tenantId = activeTenantId.value
  editing.value = false
  prompt.value = ''
  error.value = ''

  if (!tenantId) {
    savedDescription.value = ''
    draftDescription.value = ''
    error.value = t('settings.workProfile.noWorkspace')
    loading.value = false
    return
  }

  loading.value = true
  try {
    const resp = await getMyMemberProfile(tenantId)
    if (!resp.success || !resp.data) {
      error.value = resp.message || t('settings.workProfile.loadFailed')
      return
    }
    savedDescription.value = resp.data.work_profile_description?.trim() || ''
    draftDescription.value = savedDescription.value
  } catch (err: any) {
    error.value = err?.message || t('settings.workProfile.loadFailed')
  } finally {
    loading.value = false
  }
}

function startEditing() {
  draftDescription.value = savedDescription.value
  prompt.value = ''
  editing.value = true
}

function cancelEditing() {
  draftDescription.value = savedDescription.value
  prompt.value = ''
  editing.value = false
}

async function generateProfile() {
  const tenantId = activeTenantId.value
  const userPrompt = prompt.value.trim()
  if (!tenantId) {
    MessagePlugin.warning(t('settings.workProfile.noWorkspace'))
    return
  }
  if (!userPrompt) {
    MessagePlugin.warning(t('settings.workProfile.promptRequired'))
    return
  }

  generating.value = true
  try {
    const resp = await generateMyMemberProfile(tenantId, { prompt: userPrompt })
    const generatedDescription = resp.data?.description?.trim()
    if (!resp.success || !generatedDescription) {
      MessagePlugin.error(resp.message || t('settings.workProfile.generateFailed'))
      return
    }
    draftDescription.value = generatedDescription
    MessagePlugin.success(t('settings.workProfile.generated'))
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('settings.workProfile.generateFailed'))
  } finally {
    generating.value = false
  }
}

async function saveProfile() {
  const tenantId = activeTenantId.value
  const description = draftDescription.value.trim()
  if (!tenantId || !description) {
    MessagePlugin.warning(t('settings.workProfile.required'))
    return
  }

  saving.value = true
  try {
    const resp = await updateMyMemberProfile(tenantId, {
      work_profile_description: description,
    })
    if (!resp.success) {
      MessagePlugin.error(resp.message || t('settings.workProfile.saveFailed'))
      return
    }
    savedDescription.value = description
    draftDescription.value = description
    prompt.value = ''
    editing.value = false
    MessagePlugin.success(t('settings.workProfile.saved'))
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('settings.workProfile.saveFailed'))
  } finally {
    saving.value = false
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    cancelEditing()
    return
  }
  if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
    event.preventDefault()
    void saveProfile()
  }
}

watch(activeTenantId, loadProfile, { immediate: true })
</script>

<style lang="less" scoped>
.work-profile {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;
}

.section-title-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.section-header-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex: 0 0 auto;
  flex-wrap: wrap;
}

h2 {
  margin: 0 0 8px;
  color: var(--td-text-color-primary);
  font-size: 20px;
  font-weight: 600;
}

.section-description {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
  line-height: 1.5;
}

.workspace-name {
  flex: 0 0 auto;
  max-width: 220px;
  padding: 5px 10px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.loading-inline {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 40px 0;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.error-inline {
  padding: 20px 0;
}

.settings-group {
  display: flex;
  flex-direction: column;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 32px;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);
}

.setting-info {
  flex: 1 1 38%;
  min-width: 0;

  label {
    display: block;
    margin-bottom: 6px;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
  }

  .desc {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 1.6;
  }
}

.setting-control {
  flex: 1 1 62%;
  min-width: 0;
}

.work-profile-control {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 12px;
  max-width: 520px;
}

.description-display {
  min-height: 220px;
  padding: 12px 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  font-size: 14px;
  line-height: 1.65;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  overflow-y: auto;
}

.description-display.is-empty {
  color: var(--td-text-color-placeholder);
}

.ai-generation {
  padding: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
}

.ai-generation-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.ai-generation-label {
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 600;
}

.ai-generation-description {
  margin: 5px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 1.55;
}

.ai-generation :deep(.t-textarea) {
  margin-top: 12px;
}

@media (max-width: 720px) {
  .section-title-row,
  .setting-row {
    flex-direction: column;
    gap: 16px;
  }

  .section-header-actions,
  .workspace-name,
  .setting-control,
  .work-profile-control {
    width: 100%;
    max-width: none;
  }

  .section-header-actions {
    justify-content: flex-start;
  }

  .ai-generation-header {
    flex-direction: column;
  }
}
</style>
