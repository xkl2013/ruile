<template>
  <section class="public-kb-discover">
    <div v-if="loading" class="public-kb-discover__state">
      <t-loading size="medium" text="加载知识库中" />
    </div>
    <template v-else>
      <div v-if="!items.length" class="public-kb-discover__state">
        <t-icon name="folder-open" />
        <span>暂无已发布知识库</span>
      </div>

      <template v-else>
        <section v-if="visibleFeaturedItems.length" class="public-kb-section">
          <div class="public-kb-section-head">
            <h3>精选</h3>
            <button
              type="button"
              class="public-kb-refresh"
              :disabled="featuredItems.length <= FEATURED_PAGE_SIZE"
              @click="rotateFeaturedItems"
            >
              <t-icon name="refresh" />
              <span>换一换</span>
            </button>
          </div>
          <div class="public-kb-grid">
            <article
              v-for="item in visibleFeaturedItems"
              :key="`featured-${item.id}`"
              class="output-card output-card--editable discover-card public-kb-card"
              :class="{ 'is-subscribed': item.is_subscribed }"
              :tabindex="item.is_subscribed ? 0 : undefined"
              @click="handleCardClick(item)"
              @keydown.enter.self.prevent="handleCardClick(item)"
            >
              <div class="public-kb-card__image">
                <KnowledgeBaseIcon :icon="item.icon" :icon-url="item.icon_url" :type="item.type || 'document'" size="large" />
              </div>
              <div class="output-card-body">
                <div class="output-card-head">
                  <div class="output-card-actions" @click.stop>
                    <t-dropdown
                      :options="knowledgeBaseActionOptions(item)"
                      trigger="click"
                      placement="bottom-right"
                      attach="body"
                      @click="handleKnowledgeBaseAction(item, $event)"
                    >
                      <button
                        type="button"
                        class="icon-button icon-button--more"
                        :aria-label="`更多操作 ${item.title}`"
                        :disabled="busyID === item.id"
                        @click.stop
                      >
                        <t-icon name="ellipsis" />
                      </button>
                    </t-dropdown>
                  </div>
                </div>
                <h2>{{ item.title }}</h2>
                <p class="output-summary">{{ item.description || '暂无简介' }}</p>
                <div class="output-card-footer">
                  <div class="discover-card-meta">
                    <span>{{ formatCount(item.subscriber_count) }}人已订阅</span>
                    <span class="discover-card-meta-separator">|</span>
                    <span>{{ formatCount(contentCount(item)) }}个内容</span>
                    <span class="discover-card-meta-separator">|</span>
                    <span>@{{ publisherLabel(item) }}</span>
                  </div>
                </div>
              </div>
            </article>
          </div>
        </section>

        <section v-if="recommendedItems.length" class="public-kb-section">
          <div class="public-kb-section-head">
            <h3 class="public-kb-section-title--recommended">推荐</h3>
          </div>
          <div class="public-kb-grid">
            <article
              v-for="item in recommendedItems"
              :key="`recommended-${item.id}`"
              class="output-card output-card--editable discover-card public-kb-card"
              :class="{ 'is-subscribed': item.is_subscribed }"
              :tabindex="item.is_subscribed ? 0 : undefined"
              @click="handleCardClick(item)"
              @keydown.enter.self.prevent="handleCardClick(item)"
            >
              <div class="public-kb-card__image">
                <KnowledgeBaseIcon :icon="item.icon" :icon-url="item.icon_url" :type="item.type || 'document'" size="large" />
              </div>
              <div class="output-card-body">
                <div class="output-card-head">
                  <div class="output-card-actions" @click.stop>
                    <t-dropdown
                      :options="knowledgeBaseActionOptions(item)"
                      trigger="click"
                      placement="bottom-right"
                      attach="body"
                      @click="handleKnowledgeBaseAction(item, $event)"
                    >
                      <button
                        type="button"
                        class="icon-button icon-button--more"
                        :aria-label="`更多操作 ${item.title}`"
                        :disabled="busyID === item.id"
                        @click.stop
                      >
                        <t-icon name="ellipsis" />
                      </button>
                    </t-dropdown>
                  </div>
                </div>
                <h2>{{ item.title }}</h2>
                <p class="output-summary">{{ item.description || '暂无简介' }}</p>
                <div class="output-card-footer">
                  <div class="discover-card-meta">
                    <span>{{ formatCount(item.subscriber_count) }}人已订阅</span>
                    <span class="discover-card-meta-separator">|</span>
                    <span>{{ formatCount(contentCount(item)) }}个内容</span>
                    <span class="discover-card-meta-separator">|</span>
                    <span>@{{ publisherLabel(item) }}</span>
                  </div>
                </div>
              </div>
            </article>
          </div>
        </section>
      </template>
    </template>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import KnowledgeBaseIcon from '@/components/KnowledgeBaseIcon.vue'
import {
  listPublicKnowledgeBases,
  subscribePublicKnowledgeBase,
  unsubscribePublicKnowledgeBase,
  type PublicKnowledgeBasePublication,
} from '@/api/public-knowledge-base'

const router = useRouter()
const items = ref<PublicKnowledgeBasePublication[]>([])
const loading = ref(false)
const busyID = ref('')
const featuredOffset = ref(0)
const FEATURED_PAGE_SIZE = 4

const featuredItems = computed(() => items.value.filter((item) => item.featured))
const recommendedItems = computed(() => items.value.filter((item) => !item.featured))
const visibleFeaturedItems = computed(() => {
  const list = featuredItems.value
  if (list.length <= FEATURED_PAGE_SIZE) return list
  return Array.from({ length: FEATURED_PAGE_SIZE }, (_, index) =>
    list[(featuredOffset.value + index) % list.length],
  )
})

async function loadPublications() {
  loading.value = true
  featuredOffset.value = 0
  try {
    const response = await listPublicKnowledgeBases()
    items.value = response.data || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '知识库加载失败')
  } finally {
    loading.value = false
  }
}

function rotateFeaturedItems() {
  if (featuredItems.value.length <= FEATURED_PAGE_SIZE) return
  featuredOffset.value = (featuredOffset.value + FEATURED_PAGE_SIZE) % featuredItems.value.length
}

function contentCount(item: PublicKnowledgeBasePublication) {
  return item.type === 'faq' ? item.chunk_count || 0 : item.knowledge_count || 0
}

function formatCount(value: number | undefined) {
  const count = Number(value || 0)
  if (count >= 10000) {
    const formatted = (count / 10000).toFixed(count >= 100000 ? 0 : 1).replace(/\.0$/, '')
    return `${formatted}万`
  }
  return String(count)
}

function publisherLabel(item: PublicKnowledgeBasePublication) {
  return item.name || item.category || '平台知识库'
}

function knowledgeBaseActionOptions(item: PublicKnowledgeBasePublication) {
  if (!item.is_subscribed) {
    return [{ content: '订阅', value: 'subscribe' }]
  }
  return [
    { content: '查看知识库', value: 'open' },
    { content: '取消订阅', value: 'unsubscribe', theme: 'error' as const },
  ]
}

function handleKnowledgeBaseAction(
  item: PublicKnowledgeBasePublication,
  action: { value?: string | number | boolean },
) {
  switch (String(action.value)) {
    case 'open':
      openKnowledgeBase(item)
      break
    case 'subscribe':
    case 'unsubscribe':
      void toggleSubscription(item)
      break
    default:
      break
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

function handleCardClick(item: PublicKnowledgeBasePublication) {
  if (item.is_subscribed) openKnowledgeBase(item)
}

onMounted(() => {
  void loadPublications()
})
</script>

<style scoped>
.public-kb-discover {
  padding: 4px 0 32px;
  font-family: var(--app-font-family);
}

.public-kb-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.public-kb-section + .public-kb-section {
  margin-top: 42px;
}

.public-kb-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.public-kb-section-head h3 {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 20px;
  font-weight: 600;
  line-height: 28px;
}

.public-kb-section-head h3.public-kb-section-title--recommended {
  color: var(--td-text-color-primary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.public-kb-refresh {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 16px;
  line-height: 24px;
  cursor: pointer;
}

.public-kb-refresh:hover {
  color: var(--td-text-color-primary);
}

.public-kb-refresh:disabled {
  cursor: default;
  opacity: 0.45;
}

.public-kb-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px 16px;
}

.public-kb-card {
  position: relative;
  display: flex;
  align-items: start;
  gap: 12px;
  min-width: 0;
  min-height: 90px;
  height: auto;
  box-sizing: border-box;
  padding: 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 12px;
  background: var(--td-bg-color-container);
  cursor: default;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.public-kb-card:hover {
  border-color: var(--td-brand-color);
  box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12);
}

.public-kb-card.is-subscribed {
  cursor: pointer;
}

.public-kb-card.is-subscribed:focus-visible {
  outline: 2px solid var(--td-brand-color);
  outline-offset: 2px;
}

.public-kb-card__image {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  flex: 0 0 56px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  overflow: hidden;
}

.public-kb-card__image :deep(.kb-icon--large) {
  width: 56px;
  height: 56px;
  border: 0;
  border-radius: 8px;
}

.public-kb-card__image :deep(.kb-icon--large .kb-icon-symbol),
.public-kb-card__image :deep(.kb-icon--large .kb-icon-emoji) {
  font-size: 28px;
}

.output-card-body {
  display: flex;
  flex-direction: column;
  align-self: stretch;
  min-width: 0;
  padding: 0;
  overflow: hidden;
}

.output-card-body h2 {
  display: -webkit-box;
  flex: 0 0 auto;
  margin: 0 0 4px;
  max-height: 18px;
  padding-right: 34px;
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 400;
  line-height: 18px;
  letter-spacing: 0;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 1;
}

.output-card-head {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 2;
  display: flex;
  align-items: flex-start;
  justify-content: flex-end;
  gap: 12px;
  min-width: 0;
  margin: 0;
  pointer-events: none;
}

.output-card-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  opacity: 0;
  visibility: hidden;
  pointer-events: none;
  transform: translateY(-2px);
  transition: opacity 0.16s ease, transform 0.16s ease, visibility 0s linear 0.16s;
}

.public-kb-card:hover .output-card-actions,
.public-kb-card:focus .output-card-actions,
.public-kb-card:focus-within .output-card-actions {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
  transform: translateY(0);
  transition-delay: 0s;
}

.public-kb-card:not(.is-subscribed) .output-card-actions {
  opacity: 1;
  visibility: visible;
  pointer-events: auto;
  transform: translateY(0);
}

.icon-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  flex: 0 0 30px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
}

.icon-button:hover {
  background: var(--td-bg-color-container-hover);
  color: var(--td-text-color-primary);
}

.icon-button:disabled {
  cursor: default;
  opacity: 0.5;
}

.output-summary {
  display: -webkit-box;
  margin: 0 0 6px;
  max-height: 18px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
}

.output-card-footer {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  margin-top: auto;
  min-width: 0;
}

.discover-card-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
  white-space: nowrap;
}

.discover-card-meta span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.discover-card-meta-separator {
  color: var(--td-text-color-placeholder);
}

.public-kb-discover__state {
  display: flex;
  min-height: 280px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  flex-direction: column;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

@media (max-width: 900px) {
  .public-kb-grid {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .public-kb-card {
    gap: 10px;
    padding: 12px;
  }

  .public-kb-card__image {
    width: 48px;
    height: 48px;
    flex-basis: 48px;
  }

  .public-kb-card__image :deep(.kb-icon--large) {
    width: 48px;
    height: 48px;
  }
}
</style>
