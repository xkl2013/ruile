<template>
  <section class="public-kb-discover">
    <div class="public-kb-discover__toolbar">
      <div>
        <h3>知识库</h3>
        <p>订阅平台发布的知识库，订阅后可进入知识库查看、搜索和问答。</p>
      </div>
      <div class="public-kb-discover__filters">
        <t-input v-model="keyword" clearable placeholder="搜索知识库" @enter="loadPublications">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="viewMode" style="width: 128px" @change="loadPublications">
          <t-option value="all" label="全部知识库" />
          <t-option value="subscribed" label="我已订阅" />
        </t-select>
      </div>
    </div>

    <div v-if="loading" class="public-kb-discover__state">
      <t-loading size="medium" text="加载知识库中" />
    </div>
    <div v-else-if="!items.length" class="public-kb-discover__state">
      <t-icon name="folder-open" />
      <span>{{ viewMode === 'subscribed' ? '暂无已订阅知识库' : '暂无已发布知识库' }}</span>
    </div>
    <div v-else class="public-kb-grid">
      <article v-for="item in items" :key="item.id" class="public-kb-card">
        <div class="public-kb-card__icon">
          <KnowledgeBaseIcon :icon="item.icon" :icon-url="item.icon_url" :type="item.type || 'document'" size="large" />
        </div>
        <div class="public-kb-card__body">
          <div class="public-kb-card__title-row">
            <h4>{{ item.title }}</h4>
            <t-tag v-if="item.featured" theme="warning" variant="light" size="small">精选</t-tag>
          </div>
          <p>{{ item.description || '暂无简介' }}</p>
          <div class="public-kb-card__meta">
            <span>{{ item.category || '未分类' }}</span>
            <span>{{ item.knowledge_count || 0 }} 份资料</span>
            <span>{{ item.subscriber_count || 0 }} 人订阅</span>
          </div>
          <div class="public-kb-card__actions">
            <t-button
              v-if="item.is_subscribed"
              theme="primary"
              variant="outline"
              size="small"
              :loading="busyID === item.id"
              @click="toggleSubscription(item)"
            >
              已订阅
            </t-button>
            <t-button
              v-else
              theme="primary"
              size="small"
              :loading="busyID === item.id"
              @click="toggleSubscription(item)"
            >
              订阅
            </t-button>
            <t-button
              variant="text"
              size="small"
              :disabled="!item.is_subscribed"
              @click="openKnowledgeBase(item)"
            >
              查看知识库
            </t-button>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import {
  listMyPublicKnowledgeBaseSubscriptions,
  listPublicKnowledgeBases,
  subscribePublicKnowledgeBase,
  unsubscribePublicKnowledgeBase,
  type PublicKnowledgeBasePublication,
} from '@/api/public-knowledge-base'

const router = useRouter()
const items = ref<PublicKnowledgeBasePublication[]>([])
const keyword = ref('')
const viewMode = ref<'all' | 'subscribed'>('all')
const loading = ref(false)
const busyID = ref('')

async function loadPublications() {
  loading.value = true
  try {
    const response = viewMode.value === 'subscribed'
      ? await listMyPublicKnowledgeBaseSubscriptions()
      : await listPublicKnowledgeBases({ keyword: keyword.value.trim() })
    items.value = response.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库加载失败')
  } finally {
    loading.value = false
  }
}

async function toggleSubscription(item: PublicKnowledgeBasePublication) {
  if (busyID.value === item.id) return
  busyID.value = item.id
  try {
    if (item.is_subscribed) {
      await unsubscribePublicKnowledgeBase(item.id)
      item.is_subscribed = false
      item.subscriber_count = Math.max(0, item.subscriber_count - 1)
      MessagePlugin.success('已取消订阅')
      if (viewMode.value === 'subscribed') {
        items.value = items.value.filter((row) => row.id !== item.id)
      }
    } else {
      await subscribePublicKnowledgeBase(item.id)
      item.is_subscribed = true
      item.subscriber_count += 1
      MessagePlugin.success('订阅成功')
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || '订阅操作失败')
  } finally {
    busyID.value = ''
  }
}

function openKnowledgeBase(item: PublicKnowledgeBasePublication) {
  if (!item.is_subscribed) return
  void router.push(`/platform/knowledge-bases/${item.knowledge_base_id}`)
}

onMounted(() => {
  void loadPublications()
})
</script>

<style scoped>
.public-kb-discover {
  padding: 8px 0 24px;
  font-family: var(--app-font-family);
}

.public-kb-discover__toolbar {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
  margin-bottom: 18px;
}

.public-kb-discover__toolbar h3 {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 20px;
  font-weight: 600;
  line-height: 28px;
}

.public-kb-discover__toolbar p {
  margin: 6px 0 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 400;
  line-height: 20px;
}

.public-kb-discover__filters {
  display: flex;
  gap: 10px;
}

.public-kb-discover__filters .t-input {
  width: 240px;
}

.public-kb-grid {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 12px;
}

.public-kb-card {
  display: flex;
  gap: 14px;
  min-width: 0;
  min-height: 145px;
  box-sizing: border-box;
  padding: 16px;
  border: 1px solid var(--td-component-border);
  border-radius: 2px;
  background: var(--td-bg-color-container);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.public-kb-card:hover {
  border-color: var(--td-brand-color-focus);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.public-kb-card__icon {
  flex: 0 0 auto;
  padding-top: 1px;
}

.public-kb-card__body {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}

.public-kb-card__title-row {
  display: flex;
  align-items: center;
  min-height: 24px;
  gap: 8px;
}

.public-kb-card h4 {
  flex: 1;
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 16px;
  font-weight: 600;
  line-height: 24px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.public-kb-card p {
  display: -webkit-box;
  margin: 6px 0 8px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 400;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

.public-kb-card__meta {
  display: flex;
  gap: 12px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.public-kb-card__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 11px;
}

.public-kb-card__actions :deep(.t-button) {
  font-size: 12px;
  font-weight: 400;
}

.public-kb-discover__state {
  display: flex;
  min-height: 220px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-secondary);
}
@media (max-width: 900px) {
  .public-kb-discover__toolbar { flex-direction: column; }
  .public-kb-discover__filters { width: 100%; }
  .public-kb-discover__filters .t-input { flex: 1; width: auto; }
  .public-kb-grid { grid-template-columns: 1fr; }
}
</style>
