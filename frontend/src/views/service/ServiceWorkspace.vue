<template>
  <div class="service-page">
    <main class="service-main">
      <header class="service-header">
        <div class="service-heading">
          <h2>{{ pageMeta.title }}</h2>
          <p>{{ pageMeta.description }}</p>
        </div>

        <div v-if="activeView === 'messages'" class="service-header-tools">
          <div class="service-header-reminder">
            <span>服务提醒</span>
            <strong>{{ openReminderCount }} 条待处理</strong>
          </div>
          <button
            type="button"
            class="rail-icon-btn"
            :disabled="serviceMemoriesLoading"
            title="刷新服务提醒"
            aria-label="刷新服务提醒"
            @click="loadServiceMemories(true)"
          >
            <t-icon :name="serviceMemoriesLoading ? 'loading' : 'refresh'" />
          </button>
        </div>
        <div v-else-if="activeView === 'review'" class="service-header-tools">
          <div class="service-header-reminder">
            <span>日报生成</span>
            <strong>{{ reviewGenerationLabel }}</strong>
          </div>
          <button
            type="button"
            class="service-action-btn primary service-header-action"
            :disabled="serviceDailyReportsGenerating"
            @click="generateReviewReport"
          >
            <t-icon :name="serviceDailyReportsGenerating ? 'loading' : 'file-add'" />
            生成日报
          </button>
          <button
            type="button"
            class="rail-icon-btn"
            :disabled="serviceDailyReportsLoading"
            title="刷新日报"
            aria-label="刷新日报"
            @click="loadServiceDailyReports(true)"
          >
            <t-icon :name="serviceDailyReportsLoading ? 'loading' : 'refresh'" />
          </button>
        </div>
      </header>

      <div class="service-scroll">
        <section v-if="activeView === 'messages'" class="assistant-workspace">
          <aside class="assistant-rail" aria-label="服务提醒">
            <t-input
              v-model="keyword"
              class="rail-search"
              clearable
              placeholder="搜索服务提醒"
              size="small"
            >
              <template #prefix-icon>
                <t-icon name="search" />
              </template>
            </t-input>

            <div class="rail-summary">
              <span>{{ memoryExtractionSourceLabel }}</span>
              <em>{{ serviceMemoryRefreshLabel }}</em>
            </div>

            <div class="reminder-filter-tabs" role="tablist" aria-label="服务提醒筛选">
              <button
                v-for="tab in serviceReminderTabs"
                :key="tab.value"
                type="button"
                class="reminder-filter-tab"
                :class="{ active: serviceReminderFilter === tab.value }"
                role="tab"
                :aria-selected="serviceReminderFilter === tab.value"
                @click="serviceReminderFilter = tab.value"
              >
                <span>{{ tab.label }}</span>
                <small>{{ tab.count }}</small>
              </button>
            </div>

            <div class="rail-list" role="list">
              <button
                v-for="task in filteredTasks"
                :key="task.id"
                type="button"
                class="rail-item"
                :class="[
                  `priority-${task.priorityKey}`,
                  { active: activeTask.id === task.id, muted: isTaskClosed(task.id) },
                ]"
                @click="selectTask(task.id)"
              >
                <span class="rail-avatar" aria-hidden="true">{{ serviceTaskAvatarText(task) }}</span>
                <span class="rail-copy">
                  <span class="rail-title-row">
                    <strong>{{ task.title || task.customerName }}</strong>
                    <span class="rail-due">{{ task.dueText }}</span>
                  </span>
                  <em>{{ task.customerName }} · {{ task.riskLabel }} · {{ task.nextAction }}</em>
                  <span class="rail-tags">
                    <small :class="serviceReminderStatusClass(task)">{{ serviceReminderStatusLabel(task) }}</small>
                    <small>{{ task.stage }}</small>
                  </span>
                </span>
              </button>

              <div v-if="filteredTasks.length === 0" class="service-empty">{{ emptyConversationListText }}</div>
            </div>
          </aside>

          <article class="service-chat-main">
            <div v-if="hasActiveServiceTask" class="service-agent-shell">
              <section class="service-reminder-detail" aria-label="服务提醒详情">
                <div class="service-reminder-detail-head">
                  <div>
                    <span>当前服务提醒</span>
                    <h3>{{ activeTask.title || activeTask.customerName }}</h3>
                    <p>{{ activeTask.customerName }} · {{ activeTask.stage }} · {{ activeTask.dueText }}</p>
                  </div>
                  <em :class="serviceReminderStatusClass(activeTask)">{{ serviceReminderStatusLabel(activeTask) }}</em>
                </div>

                <div class="service-reminder-grid">
                  <section>
                    <h4>提醒原因</h4>
                    <p>{{ activeTask.assistReason }}</p>
                  </section>
                  <section>
                    <h4>下一步动作</h4>
                    <p>{{ activeTask.nextAction }}</p>
                  </section>
                  <section>
                    <h4>话术草稿</h4>
                    <p>{{ activeTask.replyDraft }}</p>
                  </section>
                </div>

                <div class="service-reminder-actions">
                  <button type="button" class="service-action-btn primary" @click="confirmActiveTask">
                    <t-icon name="check-circle" />
                    确认动作
                  </button>
                  <button type="button" class="service-action-btn" @click="snoozeActiveTask">
                    <t-icon name="time" />
                    稍后
                  </button>
                  <button type="button" class="service-action-btn" @click="ignoreActiveTask">
                    <t-icon name="close-circle" />
                    忽略
                  </button>
                </div>
              </section>

              <div class="service-agent-chat-shell">
                <ChatView
                  v-if="activeServiceChatSessionId"
                  :key="`${activeTask.id}:${activeServiceChatSessionId}`"
                  ref="serviceChatViewRef"
                  :session_id="activeServiceChatSessionId"
                  :agent-id="serviceAssistantAgentId"
                  :quoted-context="activeServiceAgentContext"
                  :embedded-input-placeholder="activeServiceAgentInputPlaceholder"
                  embedded-mode
                  @message-state-change="handleServiceChatMessageState"
                >
                  <template #empty-suggestions>
                    <div
                      v-if="shouldShowServiceAgentPrompts"
                      class="service-agent-empty-suggestions suggested-questions-container"
                      :class="{ 'has-questions': activePromptShortcuts.length > 0 || serviceAgentSuggestionsLoading }"
                      aria-label="便捷提问"
                    >
                      <div
                        v-if="serviceAgentSuggestionsLoading && activePromptShortcuts.length === 0"
                        class="suggested-questions-inner"
                      >
                        <div class="suggested-questions-title">
                          <t-skeleton animation="gradient" :row-col="[{ width: '120px', height: '14px' }]" />
                        </div>
                        <div class="suggested-questions-grid">
                          <div
                            v-for="n in SERVICE_ASSISTANT_SUGGESTION_LIMIT"
                            :key="`service-sq-skel-${n}`"
                            class="suggested-question-card sq-card-skeleton"
                          >
                            <t-skeleton
                              animation="gradient"
                              :row-col="[{ width: '100%', height: '14px', type: 'rect' }]"
                            />
                          </div>
                        </div>
                      </div>
                      <transition v-else appear name="sq-fade">
                        <div v-if="activePromptShortcuts.length > 0" class="suggested-questions-inner">
                          <div class="suggested-questions-title-row">
                            <p class="suggested-questions-caption">
                              <span class="suggested-questions-title">你可以这样问我</span>
                              <button
                                type="button"
                                class="suggested-questions-refresh"
                                :disabled="serviceAgentSuggestionsLoading"
                                title="换一批"
                                aria-label="换一批"
                                @click="refreshServiceAgentSuggestions"
                              >
                                <t-icon
                                  :name="serviceAgentSuggestionsLoading ? 'loading' : 'refresh'"
                                  :class="{ 'sq-refresh-spin': serviceAgentSuggestionsLoading }"
                                />
                              </button>
                            </p>
                          </div>
                          <div class="suggested-questions-grid">
                            <button
                              v-for="(prompt, index) in activePromptShortcuts"
                              :key="prompt"
                              type="button"
                              class="service-suggested-question suggested-question-card"
                              :style="{ transitionDelay: `${index * 50}ms` }"
                              @click="sendServiceAgentPrompt(prompt)"
                            >
                              <span class="suggested-question-text">{{ prompt }}</span>
                            </button>
                          </div>
                        </div>
                      </transition>
                    </div>
                  </template>
                </ChatView>
                <div v-else class="service-agent-state">
                  <t-icon :name="isActiveServiceChatLoading ? 'loading' : 'chat'" />
                  <span>
                    {{ activeServiceAgentStateText }}
                  </span>
                </div>
              </div>
            </div>
            <div v-else class="service-agent-state service-agent-state--empty">
              <t-icon name="search" />
              <span>{{ emptyConversationListText }}</span>
            </div>
          </article>

          <aside v-if="hasActiveServiceTask" class="customer-summary-panel" aria-label="服务工作档案">
            <div class="customer-summary-head">
              <span>服务助理</span>
              <div class="customer-summary-head-actions">
                <button type="button" class="customer-summary-status" @click="goToActiveCustomerSpace">
                  <t-icon name="folder" />
                  客户空间
                </button>
                <button type="button" class="customer-summary-status" @click="toggleTaskClosed(activeTask.id)">
                  <t-icon :name="isTaskClosed(activeTask.id) ? 'rollback' : 'check-circle'" />
                  {{ isTaskClosed(activeTask.id) ? '恢复' : '完成' }}
                </button>
              </div>
            </div>

            <div class="customer-summary-person">
              <div class="customer-summary-name-row">
                <h3>{{ activeTask.customerName }}</h3>
                <em :class="`priority-${activeTask.priorityKey}`">{{ activePriorityLabel }}</em>
              </div>
              <p>{{ activeStudentLabel }} · {{ activeTask.stage }} · {{ activeTask.channel }}</p>
            </div>

            <section class="assistant-result-card">
              <h4>客户摘要.md</h4>
              <p>{{ activeTask.summary }}</p>
              <div class="assistant-result-meta">
                <span>{{ serviceTaskSourceLabel(activeTask) }}</span>
                <span>{{ activeTask.lastMemoryLabel }}</span>
              </div>
            </section>

            <section class="assistant-result-card assistant-result-card--actions">
              <h4>未闭环事项.md</h4>
              <ul>
                <li v-for="item in activeTask.salesHighlights" :key="item">{{ item }}</li>
              </ul>
            </section>

            <dl class="customer-summary-facts">
              <div v-for="fact in activeServiceFacts" :key="fact.label">
                <dt>{{ fact.label }}</dt>
                <dd>{{ fact.value }}</dd>
              </div>
            </dl>

            <div class="customer-summary-actions">
              <button type="button" class="customer-summary-entry" @click="openCustomerDetail('followUp')">
                <span>
                  <strong>服务动作</strong>
                  <em>{{ activeTask.nextAction }}</em>
                </span>
                <t-icon name="chevron-right" />
              </button>

              <button type="button" class="customer-summary-entry" @click="openCustomerDetail('profile')">
                <span>
                  <strong>证据索引</strong>
                  <em>{{ serviceTaskSourceLabel(activeTask) }} · {{ activeTask.confidenceLabel }}</em>
                </span>
                <t-icon name="chevron-right" />
              </button>
            </div>

            <section class="customer-summary-draft">
              <h4>待确认动作</h4>
              <p>{{ activeTask.writeBackDraft }}</p>
            </section>

            <section class="customer-summary-source">
              <h4>客户空间</h4>
              <p>{{ activeCustomerSpaceHint }}</p>
              <button type="button" class="customer-summary-open-space" @click="goToActiveCustomerSpace">
                <span>查看客户空间</span>
                <t-icon name="chevron-right" />
              </button>
            </section>
          </aside>
          <aside v-else class="customer-summary-panel customer-summary-panel--empty" aria-label="服务工作档案">
            <div class="customer-summary-empty">
              <strong>暂无服务提醒</strong>
              <p>还没有收集到可生成提醒的客户服务记忆。</p>
            </div>
          </aside>
        </section>

        <section v-else class="review-page review-report-page">
          <div class="review-hero">
            <div class="review-hero-copy">
              <div class="review-heading">
                <OrganizeSproutIcon class="review-heading-icon" />
                <span>日报</span>
              </div>
              <p>汇总客户服务动作、沟通风险和下阶段建议，形成个人可回看的服务日报。</p>
            </div>
            <div class="review-hero-stats">
              <span><strong>{{ filteredReviewReports.length }}</strong> 份日报</span>
              <span><strong>{{ reviewTotalActions }}</strong> 个动作</span>
            </div>
          </div>

          <div class="review-month-list">
            <section v-for="group in reviewReportGroups" :key="group.key" class="review-month-group">
              <div class="review-month-header">
                <div class="review-month-heading">
                  <h3>{{ group.label }}</h3>
                  <div class="segmented-tabs review-range-tabs" role="tablist" aria-label="日报时间筛选">
                    <button
                      v-for="tab in reviewRangeTabs"
                      :key="tab.value"
                      type="button"
                      class="segmented-tab"
                      :class="{ 'segmented-tab--active': reviewRange === tab.value }"
                      role="tab"
                      :aria-selected="reviewRange === tab.value"
                      @click="setReviewRange(tab.value)"
                    >
                      {{ tab.label }}
                    </button>
                  </div>
                </div>
                <span>{{ filteredReviewReports.length }} 份日报</span>
              </div>

              <div v-if="group.reports.length" class="report-list review-report-list">
                <article
                  v-for="report in group.reports"
                  :key="report.id"
                  class="review-report-card report-card--editable"
                  role="button"
                  tabindex="0"
                  @click="openReviewPreview(report)"
                  @keydown.enter.self="openReviewPreview(report)"
                >
                  <div class="review-report-gutter" aria-hidden="true">
                    <div class="review-report-ribbon">
                      <span>睿乐</span>
                      <span>日报</span>
                    </div>
                  </div>

                  <div class="review-report-main">
                    <div class="review-report-title-row">
                      <h2>{{ report.title }}</h2>
                      <span class="type-badge" :class="`review-stage--${report.stageKey}`">{{ report.stage }}</span>
                    </div>
                    <p class="review-report-intro">{{ report.intro }}</p>

                    <div v-if="report.chips.length" class="report-chips">
                      <span v-for="chip in report.chips" :key="chip">{{ chip }}</span>
                    </div>

                    <div class="report-meta review-report-meta">
                      <span>{{ report.updated }}</span>
                      <span class="review-report-meta-separator">|</span>
                      <span>{{ report.actionCount }} 个动作</span>
                      <span>{{ report.customerCount }} 位客户</span>
                    </div>
                  </div>
                </article>
              </div>

              <div v-else class="service-empty">暂无日报</div>
            </section>
          </div>
        </section>
      </div>
    </main>

    <t-drawer
      v-model:visible="customerDetailVisible"
      class="customer-detail-drawer"
      :header="false"
      :footer="false"
      :close-btn="false"
      :size="'min(560px, 92vw)'"
      attach="body"
      placement="right"
    >
      <div class="customer-detail-header">
        <div>
          <span>服务客户摘要</span>
          <strong>{{ activeCustomerDetailTitle }}</strong>
        </div>
        <t-button
          variant="text"
          theme="default"
          size="small"
          class="customer-detail-close"
          aria-label="关闭服务详情"
          @click="closeCustomerDetail"
        >
          <template #icon><t-icon name="close" size="16px" /></template>
        </t-button>
      </div>

      <div class="customer-detail-body">
        <section class="customer-detail-profile">
          <h3>{{ activeTask.title || activeTask.customerName }}</h3>
          <p>{{ activeTask.customerName }} · {{ activeTask.channel }} · {{ activeStudentLabel }}</p>
        </section>

        <template v-if="customerDetailType === 'followUp'">
          <section class="customer-detail-section">
            <h4>处理顺序</h4>
            <ol class="customer-detail-steps">
              <li v-for="step in activeServiceSteps" :key="step">{{ step }}</li>
            </ol>
          </section>

          <section class="customer-detail-section">
            <h4>落地记录</h4>
            <p>{{ activeTask.writeBackDraft }}</p>
          </section>

          <section class="customer-detail-section customer-detail-section--warning">
            <h4>不要做</h4>
            <p>{{ activeTask.avoidAction }}</p>
          </section>

          <section class="customer-detail-section">
            <h4>可直接发送</h4>
            <blockquote>{{ activeTask.replyDraft }}</blockquote>
          </section>
        </template>

        <template v-else>
          <section class="customer-detail-section">
            <h4>服务事实</h4>
            <dl class="customer-detail-facts">
              <div v-for="fact in activeServiceFacts" :key="fact.label">
                <dt>{{ fact.label }}</dt>
                <dd>{{ fact.value }}</dd>
              </div>
            </dl>
          </section>

          <section class="customer-detail-section">
            <h4>本次判断</h4>
            <p>{{ activeTask.assistReason }}</p>
          </section>

          <section class="customer-detail-section">
            <h4>记忆证据</h4>
            <div class="customer-memory-evidence-list">
              <article
                v-for="memory in activeMemoryEvidence"
                :key="memory.id"
                class="customer-memory-evidence"
              >
                <strong>{{ memory.title }}</strong>
                <p>{{ memory.summary }}</p>
                <span>{{ memory.sourceLabel }} · {{ memory.occurredAtLabel }}</span>
              </article>
            </div>
          </section>

          <section class="customer-detail-section">
            <h4>服务信号</h4>
            <div class="customer-detail-tags">
              <span v-for="item in activeTask.memorySignals" :key="item">{{ item }}</span>
            </div>
          </section>
        </template>
      </div>
    </t-drawer>

    <t-drawer
      v-model:visible="reviewPreviewVisible"
      class="review-preview-drawer"
      :header="false"
      :footer="false"
      :close-btn="false"
      :size="'min(760px, 92vw)'"
      attach="body"
      placement="right"
    >
      <template v-if="activeReviewReport">
        <div class="review-preview-header">
          <div class="review-preview-header-copy">
            <div class="review-preview-eyebrow">日报</div>
            <div class="review-preview-title">{{ activeReviewReport.title }}</div>
          </div>
          <div class="review-preview-actions">
            <t-button
              variant="text"
              theme="default"
              size="small"
              class="review-preview-action"
              aria-label="关闭预览"
              @click="closeReviewPreview"
            >
              <template #icon><t-icon name="close" size="16px" /></template>
            </t-button>
          </div>
        </div>

        <div class="review-preview-page">
          <div class="review-preview-body">
            <div class="review-preview-meta">
              <span>{{ activeReviewReport.updated }}</span>
              <span>{{ activeReviewReport.actionCount }} 个动作</span>
              <span>{{ activeReviewReport.customerCount }} 位客户</span>
            </div>
            <h1>{{ activeReviewReport.title }}</h1>

            <div class="review-preview-content" v-html="activeReviewReport.renderedHtml" />
          </div>
        </div>
      </template>
    </t-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  findServiceMenuRoute,
  isServiceTab,
  type ServiceTab,
} from './serviceRoutes'
import { createSessions } from '@/api/chat'
import { getSuggestedQuestions, type SuggestedQuestion } from '@/api/agent'
import ChatView from '@/views/chat/index.vue'
import {
  buildSproutReportPreview,
  sproutReportContentForEditor,
  type SproutReportPreviewSection,
} from '../organize/sproutReport'
import OrganizeSproutIcon from '../organize/components/OrganizeSproutIcon.vue'
import {
  SERVICE_ASSISTANT_AGENT_ID,
  SERVICE_ASSISTANT_INPUT_PLACEHOLDER,
  SERVICE_ASSISTANT_SUGGESTION_LIMIT,
} from './serviceAgentConfig'
import {
  emptyServiceTask,
  type PriorityKey,
  type ServiceTask,
} from './serviceMemoryExtraction'
import {
  createServiceActionDraft,
  generateServiceDailyReport,
  getServiceBootstrap,
  listServiceDailyReports,
  mapServiceReminderToTask,
  refreshServiceModule,
  type ServiceDailyReportDTO,
  type ServiceWorkProfile as ServiceWorkProfileDTO,
  updateServiceReminderStatus,
} from '@/api/service'

type ReviewRange = 'week' | 'month'
type ReviewStageKey = 'formed' | 'expandable' | 'organizing'
type CustomerDetailType = 'followUp' | 'profile'
type ServiceReminderFilter = 'all' | 'lead' | 'customer' | 'schedule' | 'risk'

interface ServiceReviewStageInsight {
  id: string
  label: string
  count: number
  percent: number
  action: string
}

interface ServiceReviewRiskInsight {
  id: string
  label: string
  count: number
  description: string
  tasks: string[]
}

interface ServiceReviewActionRow {
  id: string
  customerName: string
  stage: string
  riskLabel: string
  nextAction: string
  dueText: string
  confidenceLabel: string
  evidenceLabel: string
  priorityKey: PriorityKey
}

interface ServiceReviewKnowledgeGap {
  id: string
  title: string
  reason: string
  action: string
}

interface ServiceReviewReport {
  id: string
  title: string
  content: string
  renderedHtml: string
  intro: string
  previewSections: SproutReportPreviewSection[]
  stage: string
  stageKey: ReviewStageKey
  range: ReviewRange
  updated: string
  updatedAt: string
  actionCount: number
  customerCount: number
  chips: string[]
}

interface ServiceFact {
  label: string
  value: string
}

type ServiceChatViewExpose = {
  triggerSend?: (question: string) => void
}

interface ServiceChatMessageState {
  sessionId?: string
  messageCount?: number
  hasMessages?: boolean
}

const route = useRoute()
const router = useRouter()

const activeView = computed<ServiceTab>({
  get() {
    const tab = route.meta.serviceTab
    return isServiceTab(tab) ? tab : 'messages'
  },
  set(value) {
    const nextRoute = findServiceMenuRoute(value)
    if (nextRoute && route.path !== nextRoute.path) {
      void router.push(nextRoute.path)
    }
  },
})

const keyword = ref('')
const serviceReminderFilter = ref<ServiceReminderFilter>('all')
const reviewRange = ref<ReviewRange>('week')
const reviewPreviewVisible = ref(false)
const activeReviewReport = ref<ServiceReviewReport | null>(null)
const customerDetailVisible = ref(false)
const customerDetailType = ref<CustomerDetailType>('followUp')
const activeTaskId = ref('')
const closedTaskIds = ref<string[]>([])
const ignoredTaskIds = ref<string[]>([])
const snoozedTaskIds = ref<string[]>([])
const serviceChatViewRef = ref<ServiceChatViewExpose | null>(null)
const serviceChatSessionIds = ref<Record<string, string>>({})
const serviceChatSessionLoadingId = ref('')
const serviceChatSessionError = ref('')
const serviceChatHasMessagesByTask = ref<Record<string, boolean>>({})
const serviceAgentSuggestions = ref<SuggestedQuestion[]>([])
const serviceAgentSuggestionsLoading = ref(false)
const serviceAgentSuggestionsLoaded = ref(false)
const backendServiceTasks = ref<ServiceTask[]>([])
const serviceWorkProfile = ref<ServiceWorkProfileDTO | null>(null)
const serviceBackendTotal = ref(0)
const serviceHasConfiguredProfile = ref(false)
const backendReviewReports = ref<ServiceReviewReport[]>([])
const serviceDailyReportsLoading = ref(false)
const serviceDailyReportsGenerating = ref(false)
const serviceDailyReportsLoaded = ref(false)
const serviceDailyReportsError = ref('')
const serviceMemoriesLoading = ref(false)
const serviceMemoriesLoaded = ref(false)
const serviceMemoriesError = ref('')
const serviceChatSessionRequests = new Map<string, Promise<string>>()
let serviceChatSessionRequestSeed = 0

const reviewRangeTabs: Array<{ label: string; value: ReviewRange }> = [
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' },
]

const activeServiceMemoryScope = computed(() => serviceWorkProfile.value?.memory_scope || '本人服务相关记忆')
const activeCustomerSpaceHint = computed(() => {
  if (!hasActiveServiceTask.value) return '暂无客户空间'
  return `${activeTask.value.customerName} 的摘要、跟进记录、未闭环事项和记忆证据`
})

const enrichServiceReviewReport = (
  item: Omit<ServiceReviewReport, 'renderedHtml' | 'intro' | 'previewSections'>,
): ServiceReviewReport => {
  const preview = buildSproutReportPreview(item.content, item.title)
  return {
    ...item,
    renderedHtml: sproutReportContentForEditor(item.content),
    intro: preview.intro || item.title,
    previewSections: preview.sections,
  }
}

const normalizeReviewReportRange = (range?: string): ReviewRange => (range === 'month' ? 'month' : 'week')

const formatReviewReportUpdated = (value?: string) => {
  if (!value) return '刚刚生成'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getMonth() + 1}月${date.getDate()}日 ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
}

const mapServiceDailyReportToReviewReport = (item: ServiceDailyReportDTO): ServiceReviewReport => enrichServiceReviewReport({
  id: item.id,
  title: item.title || '服务日报',
  content: item.content || item.summary || '# 服务日报\n\n暂无日报内容。',
  stage: item.stage || '已生成',
  stageKey: item.stage_key === 'organizing' ? 'organizing' : 'formed',
  range: normalizeReviewReportRange(item.range),
  updated: item.updated || formatReviewReportUpdated(item.updated_at),
  updatedAt: item.updated_at || item.created_at || new Date().toISOString(),
  actionCount: item.action_count || 0,
  customerCount: item.customer_count || 0,
  chips: item.chips || [],
})

const memoryDerivedTasks = computed(() => backendServiceTasks.value)
const serviceTasks = computed(() => memoryDerivedTasks.value)

const pageMeta = computed(() => {
  if (activeView.value === 'review') {
    return { title: '日报', description: '查看客户服务回顾、风险提醒和下阶段建议' }
  }
  return { title: '服务提醒', description: '从记忆笔记整理今天要服务谁、为什么提醒和下一步动作' }
})

const normalize = (value: string) => value.trim().toLowerCase()
const taskById = computed(() => new Map(serviceTasks.value.map((task) => [task.id, task])))
const taskStatus = (id: string) => taskById.value.get(id)?.status || ''
const isTaskClosed = (id: string) => closedTaskIds.value.includes(id) || ['confirmed', 'completed'].includes(taskStatus(id))
const isTaskIgnored = (id: string) => ignoredTaskIds.value.includes(id) || taskStatus(id) === 'ignored'
const isTaskSnoozed = (id: string) => snoozedTaskIds.value.includes(id) || taskStatus(id) === 'snoozed'
const visibleServiceTasks = computed(() => serviceTasks.value.filter((task) => !isTaskIgnored(task.id)))
const openTasks = computed(() => visibleServiceTasks.value.filter((task) => !isTaskClosed(task.id)))
const openReminderCount = computed(() => openTasks.value.length)
const hasActiveServiceTask = computed(() => visibleServiceTasks.value.length > 0)
const memoryExtractionSourceLabel = computed(() => {
  if (memoryDerivedTasks.value.length > 0) return '个人记忆 · 服务工作档案'
  if (serviceMemoriesLoading.value) return '正在整理服务记忆'
  return '无关记忆不进入服务提醒'
})
const serviceMemoryRefreshLabel = computed(() => {
  if (serviceMemoriesLoading.value) return '同步中'
  if (serviceMemoriesError.value) return '读取失败'
  if (serviceMemoriesLoaded.value && serviceBackendTotal.value > 0) return `${serviceBackendTotal.value} 条真实提醒`
  if (serviceMemoriesLoaded.value && serviceHasConfiguredProfile.value) return '暂无真实提醒'
  if (serviceMemoriesLoaded.value) return '暂无真实提醒'
  return '待同步'
})
const reviewGenerationLabel = computed(() => {
  if (serviceDailyReportsGenerating.value) return '生成中'
  if (serviceDailyReportsLoading.value) return '同步中'
  if (serviceDailyReportsError.value) return '读取失败'
  if (backendReviewReports.value.length > 0) return `${backendReviewReports.value.length} 份已生成`
  if (serviceDailyReportsLoaded.value) return '暂无已生成'
  return '待同步'
})

const priorityOrder: Record<PriorityKey, number> = { high: 0, medium: 1, low: 2 }
const reviewActiveTasks = computed(() => serviceTasks.value.filter((task) => !closedTaskIds.value.includes(task.id)))
const reviewActiveTaskCount = computed(() => serviceTasks.value.length)
const reviewHighRiskTasks = computed(() => serviceTasks.value.filter((task) =>
  task.priorityKey === 'high' || ['售后风险', '未闭环', '价格顾虑'].includes(task.riskLabel),
))
const reviewPendingWriteBackCount = computed(() => serviceTasks.value.filter((task) =>
  !closedTaskIds.value.includes(task.id) && task.writeBackStatus.includes('待'),
).length)

const reviewStageInsights = computed<ServiceReviewStageInsight[]>(() => {
  const total = Math.max(serviceTasks.value.length, 1)
  const groups = new Map<string, ServiceTask[]>()
  serviceTasks.value.forEach((task) => {
    const current = groups.get(task.stage) || []
    current.push(task)
    groups.set(task.stage, current)
  })

  return Array.from(groups.entries())
    .map(([stage, tasks]) => ({
      id: stage,
      label: stage,
      count: tasks.length,
      percent: Math.max(8, Math.round((tasks.length / total) * 100)),
      action: tasks[0]?.nextAction || '保持跟进节奏',
    }))
    .sort((a, b) => b.count - a.count)
})

const reviewRiskDescription = (riskLabel: string) => {
  if (riskLabel === '售后风险') return '先确认处理结果，再决定是否补充解释或升级沟通。'
  if (riskLabel === '价格顾虑') return '先补价值证明和孩子变化，再谈价格或优惠边界。'
  if (riskLabel === '适应焦虑') return '需要具体观察支撑，避免用泛泛安慰替代事实。'
  if (riskLabel === '续费窗口') return '先做阶段成长回顾，再进入续费判断。'
  if (riskLabel === '未闭环') return '需要补齐处理结果、家长反馈和下一步记录。'
  return '记忆证据不足，先补关键事实再生成判断。'
}

const reviewRiskInsights = computed<ServiceReviewRiskInsight[]>(() => {
  const groups = new Map<string, ServiceTask[]>()
  serviceTasks.value.forEach((task) => {
    const key = task.riskLabel || '待判断'
    const current = groups.get(key) || []
    current.push(task)
    groups.set(key, current)
  })

  const risks = Array.from(groups.entries())
    .map(([riskLabel, tasks]) => ({
      id: riskLabel,
      label: riskLabel,
      count: tasks.length,
      description: reviewRiskDescription(riskLabel),
      tasks: tasks.slice(0, 3).map((task) => task.customerName),
    }))
    .sort((a, b) => b.count - a.count)

  return risks.length > 0
    ? risks.slice(0, 4)
    : [{
      id: 'stable',
      label: '暂无明显风险',
      count: 0,
      description: '当前记忆没有抽到高频风险。',
      tasks: ['保持记录'],
    }]
})

const reviewActionRows = computed<ServiceReviewActionRow[]>(() => reviewActiveTasks.value
  .slice()
  .sort((a, b) => priorityOrder[a.priorityKey] - priorityOrder[b.priorityKey])
  .slice(0, 6)
  .map((task) => ({
    id: task.id,
    customerName: task.customerName,
    stage: task.stage,
    riskLabel: task.riskLabel,
    nextAction: task.nextAction,
    dueText: task.dueText,
    confidenceLabel: task.confidenceLabel,
    evidenceLabel: `${task.sourceMemoryCount} 条证据`,
    priorityKey: task.priorityKey,
  })))

const reviewKnowledgeGaps = computed<ServiceReviewKnowledgeGap[]>(() => {
  const tasks = serviceTasks.value
  const risks = new Set(tasks.map((task) => task.riskLabel))
  const stages = new Set(tasks.map((task) => task.stage))
  const gaps: ServiceReviewKnowledgeGap[] = []

  if (risks.has('价格顾虑')) {
    gaps.push({
      id: 'price-value',
      title: '价值证明素材',
      reason: '价格顾虑反复出现，单靠即时解释容易把对话推向砍价。',
      action: '沉淀孩子变化、课程目标、家长案例三类材料。',
    })
  }

  if (risks.has('适应焦虑')) {
    gaps.push({
      id: 'adaptation-observation',
      title: '适应期观察模板',
      reason: '家长焦虑需要具体观察记录支撑。',
      action: '整理入园前四周的观察维度和回访话术。',
    })
  }

  if (risks.has('售后风险') || risks.has('未闭环')) {
    gaps.push({
      id: 'after-sale-close',
      title: '售后闭环规则',
      reason: '餐食、午睡、请假等问题需要统一回应节奏。',
      action: '补齐先回应、再观察、再确认满意度的处理标准。',
    })
  }

  if (stages.has('续费服务')) {
    gaps.push({
      id: 'renewal-growth',
      title: '成长回顾模板',
      reason: '续费前更适合先回顾孩子变化，而不是直接提醒课次。',
      action: '沉淀阶段变化、当前目标、下阶段安排三段式模板。',
    })
  }

  if (tasks.some((task) => task.confidenceLabel === '待确认')) {
    gaps.push({
      id: 'memory-fields',
      title: '关键事实补录',
      reason: '部分线索只有单条记忆支撑，判断置信度不够。',
      action: '补客户、学员、决策人、下一步时间四个事实。',
    })
  }

  return gaps.length > 0
    ? gaps.slice(0, 4)
    : [{
      id: 'stable-playbook',
      title: '有效服务动作样本',
      reason: '当前风险较少，可以把已闭环动作沉淀成示范案例。',
      action: '选择 2 条完整服务记忆，整理成可复用话术。',
    }]
})

const reviewDiagnosis = computed(() => {
  if (reviewHighRiskTasks.value.length > 0) {
    return `当前 ${reviewHighRiskTasks.value.length} 条高风险线索需要先闭环，再生成日报。`
  }
  if (reviewPendingWriteBackCount.value > 0) {
    return `当前 ${reviewPendingWriteBackCount.value} 条动作待确认落地，适合先完成记录闭环。`
  }
  return '当前服务节奏稳定，可以把有效话术沉淀到公共知识库。'
})

const buildServiceReviewReportContent = () => {
  const rangeLabel = reviewRange.value === 'week' ? '本周' : '本月'
  const stageLines = reviewStageInsights.value
    .map((stage) => `- ${stage.label}：${stage.count} 条会话，建议动作：${stage.action}`)
    .join('\n')
  const riskLines = reviewRiskInsights.value
    .map((risk) => `- ${risk.label}：${risk.count} 条，${risk.description}`)
    .join('\n')
  const actionLines = reviewActionRows.value
    .map((action) => `- ${action.customerName}：${action.nextAction}（${action.dueText}，${action.confidenceLabel}）`)
    .join('\n')
  const gapLines = reviewKnowledgeGaps.value
    .map((gap) => `- ${gap.title}：${gap.action}`)
    .join('\n')

  return `# ${rangeLabel}日报

${reviewDiagnosis.value}

## 1、服务回顾

${stageLines || '- 暂无可回顾的客户阶段。'}

## 2、风险归因

${riskLines || '- 暂无明显风险。'}

## 3、行动闭环

${actionLines || '- 暂无待处理动作。'}

## 4、知识补齐

${gapLines || '- 暂无知识补齐建议。'}`
}

const currentReviewReport = computed<ServiceReviewReport>(() => enrichServiceReviewReport({
  id: `review-live-${reviewRange.value}`,
  title: `${reviewRange.value === 'week' ? '本周' : '本月'}记忆驱动日报`,
  content: buildServiceReviewReportContent(),
  stage: memoryDerivedTasks.value.length > 0 ? '记忆生成' : '等待记忆',
  stageKey: memoryDerivedTasks.value.length > 0 ? 'formed' : 'organizing',
  range: reviewRange.value,
  updated: serviceMemoriesLoading.value ? '同步中' : '当前梳理',
  updatedAt: new Date().toISOString(),
  actionCount: reviewActionRows.value.length,
  customerCount: reviewActiveTaskCount.value,
  chips: ['业务洞察', '风险归因', '行动闭环'],
}))

const reviewReports = computed(() => {
  if (serviceDailyReportsLoaded.value || backendReviewReports.value.length > 0) {
    return [currentReviewReport.value, ...backendReviewReports.value]
  }
  return [currentReviewReport.value]
})
const filteredReviewReports = computed(() => reviewReports.value.filter((report) => report.range === reviewRange.value))
const reviewTotalActions = computed(() => filteredReviewReports.value.reduce((total, report) => total + report.actionCount, 0))
const reviewReportGroups = computed(() => [{
  key: reviewRange.value,
  label: reviewRange.value === 'week' ? '本周' : '本月',
  reports: filteredReviewReports.value,
}])

const updateBackendTaskStatus = (id: string, status: string, writeBackStatus: string) => {
  backendServiceTasks.value = backendServiceTasks.value.map((task) => task.id === id
    ? { ...task, status, writeBackStatus }
    : task)
}

const toggleTaskClosed = async (id: string) => {
  const task = taskById.value.get(id)
  if (!task) return
  const nextStatus = isTaskClosed(id) ? 'pending' : 'confirmed'
  try {
    const response = await updateServiceReminderStatus(id, nextStatus)
    backendServiceTasks.value = backendServiceTasks.value.map((item) => item.id === id
      ? mapServiceReminderToTask(response.data)
      : item)
  } catch (error) {
    console.warn('[ServiceAgent] Failed to update service reminder status:', error)
    MessagePlugin.error('服务提醒状态更新失败')
    return
  }
  if (nextStatus === 'confirmed') {
    closedTaskIds.value = [...new Set([...closedTaskIds.value, id])]
  } else {
    closedTaskIds.value = closedTaskIds.value.filter((item) => item !== id)
  }
  ignoredTaskIds.value = ignoredTaskIds.value.filter((item) => item !== id)
  snoozedTaskIds.value = snoozedTaskIds.value.filter((item) => item !== id)
}

const confirmActiveTask = async () => {
  if (!activeTask.value.id) return
  try {
    await createServiceActionDraft(activeTask.value.id)
    updateBackendTaskStatus(activeTask.value.id, 'confirmed', '已确认动作')
  } catch (error) {
    console.warn('[ServiceAgent] Failed to create service action draft:', error)
    MessagePlugin.error('动作草稿确认失败')
    return
  }
  if (!isTaskClosed(activeTask.value.id)) {
    closedTaskIds.value = [...closedTaskIds.value, activeTask.value.id]
  }
  ignoredTaskIds.value = ignoredTaskIds.value.filter((item) => item !== activeTask.value.id)
  snoozedTaskIds.value = snoozedTaskIds.value.filter((item) => item !== activeTask.value.id)
  MessagePlugin.success('已确认动作草稿')
}

const snoozeActiveTask = async () => {
  if (!activeTask.value.id) return
  try {
    const response = await updateServiceReminderStatus(activeTask.value.id, 'snoozed')
    backendServiceTasks.value = backendServiceTasks.value.map((task) => task.id === activeTask.value.id
      ? mapServiceReminderToTask(response.data)
      : task)
  } catch (error) {
    console.warn('[ServiceAgent] Failed to snooze service reminder:', error)
    MessagePlugin.error('稍后处理失败')
    return
  }
  if (!isTaskSnoozed(activeTask.value.id)) {
    snoozedTaskIds.value = [...snoozedTaskIds.value, activeTask.value.id]
  }
  MessagePlugin.info('已标记为稍后处理')
}

const ignoreActiveTask = async () => {
  if (!activeTask.value.id) return
  try {
    const response = await updateServiceReminderStatus(activeTask.value.id, 'ignored')
    backendServiceTasks.value = backendServiceTasks.value.map((task) => task.id === activeTask.value.id
      ? mapServiceReminderToTask(response.data)
      : task)
  } catch (error) {
    console.warn('[ServiceAgent] Failed to ignore service reminder:', error)
    MessagePlugin.error('忽略失败')
    return
  }
  ignoredTaskIds.value = [...new Set([...ignoredTaskIds.value, activeTask.value.id])]
  closedTaskIds.value = closedTaskIds.value.filter((item) => item !== activeTask.value.id)
  snoozedTaskIds.value = snoozedTaskIds.value.filter((item) => item !== activeTask.value.id)
  MessagePlugin.info('已忽略该服务提醒')
}

const activeTask = computed<ServiceTask>(() => visibleServiceTasks.value.find((task) => task.id === activeTaskId.value) ?? visibleServiceTasks.value[0] ?? emptyServiceTask)
const serviceAssistantAgentId = SERVICE_ASSISTANT_AGENT_ID
const activeServiceChatSessionId = computed(() => serviceChatSessionIds.value[activeTask.value.id] || '')
const isActiveServiceChatLoading = computed(() => serviceChatSessionLoadingId.value === activeTask.value.id)
const activeServiceChatHasMessages = computed(() => Boolean(serviceChatHasMessagesByTask.value[activeTask.value.id]))
const activePromptShortcuts = computed(() => serviceAgentSuggestions.value.map((item) => item.question).filter(Boolean))
const shouldShowServiceAgentPrompts = computed(() =>
  !activeServiceChatHasMessages.value && (activePromptShortcuts.value.length > 0 || serviceAgentSuggestionsLoading.value),
)
const activeServiceAgentStateText = computed(() => {
  if (isActiveServiceChatLoading.value) return '正在准备服务助理'
  if (serviceChatSessionError.value) return serviceChatSessionError.value
  return '服务助理暂不可用'
})
const priorityLabelMap: Record<PriorityKey, string> = {
  high: '高优先',
  medium: '中优先',
  low: '低优先',
}
const activePriorityLabel = computed(() => priorityLabelMap[activeTask.value.priorityKey])
const activeStudentLabel = computed(() => (activeTask.value.studentName && activeTask.value.studentName !== '待补充' ? `学员 ${activeTask.value.studentName}` : '学员待补充'))
const activeMemoryEvidence = computed(() => activeTask.value.memoryEvidence)
const activeCustomerDetailTitle = computed(() => (customerDetailType.value === 'followUp' ? '服务动作' : '记忆证据'))
const serviceTaskSourceLabel = (task: ServiceTask) => `${task.sourceMemoryCount} 条记忆`
const activeServiceFacts = computed<ServiceFact[]>(() => {
  const task = activeTask.value
  return [
    { label: '来源', value: serviceTaskSourceLabel(task) },
    { label: '最近记忆', value: task.lastMemoryLabel },
    { label: '置信度', value: task.confidenceLabel },
    { label: '风险', value: task.riskLabel },
    { label: '决策人', value: task.decisionRole },
    { label: '渠道', value: task.channel },
    { label: '落地', value: task.writeBackStatus },
  ]
})
const activeServiceSteps = computed(() => {
  const task = activeTask.value
  return [
    `核对事实：${task.assistReason}`,
    `建议动作：${task.primaryAction}`,
    `落地记录：${task.writeBackDraft}`,
  ]
})
const activeServiceAgentContext = computed(() => buildServiceAgentContext(activeTask.value))
const activeServiceAgentInputPlaceholder = computed(() => SERVICE_ASSISTANT_INPUT_PLACEHOLDER)
const emptyConversationListText = computed(() => {
  if (serviceMemoriesLoading.value) return '正在整理服务提醒'
  if (serviceMemoriesError.value) return '记忆读取失败'
  if (visibleServiceTasks.value.length > 0) return '暂无匹配的服务提醒'
  if (serviceMemoriesLoaded.value) return '暂无可生成提醒的客户服务记忆'
  return '暂无服务提醒'
})

const serviceTaskAvatarText = (task: ServiceTask) => {
  const text = task.studentName && task.studentName !== '待补充' ? task.studentName : task.customerName || task.title || '服'
  return Array.from(text.trim())[0] || '服'
}

const serviceReminderStatusLabel = (task: ServiceTask) => {
  if (isTaskIgnored(task.id)) return '已忽略'
  if (isTaskClosed(task.id)) return '已确认'
  if (isTaskSnoozed(task.id)) return '稍后'
  if (serviceChatHasMessagesByTask.value[task.id]) return '已生成话术'
  if (task.priorityKey === 'high') return '待确认执行'
  return '待处理'
}

const serviceReminderStatusClass = (task: ServiceTask) => ({
  'reminder-status': true,
  'reminder-status--done': isTaskClosed(task.id),
  'reminder-status--muted': isTaskIgnored(task.id) || isTaskSnoozed(task.id),
  'reminder-status--warning': !isTaskClosed(task.id) && !isTaskIgnored(task.id) && task.priorityKey === 'high',
})

const taskMatchesReminderFilter = (task: ServiceTask, filter: ServiceReminderFilter) => {
  if (filter === 'all') return true
  const text = `${task.title} ${task.stage} ${task.riskLabel} ${task.nextAction} ${task.memorySignals.join(' ')}`
  if (filter === 'lead') return /线索|试听|体验课|咨询|报名|售前|邀约/.test(text)
  if (filter === 'schedule') return /排课|调课|请假|补课|试听安排|时间/.test(text)
  if (filter === 'risk') return /风险|投诉|售后|退款|退费|未闭环|不满|价格顾虑/.test(text)
  if (filter === 'customer') return /客户|家长|续费|回访|在园|服务|成长|反馈/.test(text)
  return true
}

const serviceReminderTabs = computed(() => {
  const tabs: Array<{ label: string; value: ServiceReminderFilter }> = [
    { label: '全部', value: 'all' },
    { label: '线索', value: 'lead' },
    { label: '服务', value: 'customer' },
    { label: '排课', value: 'schedule' },
    { label: '风险', value: 'risk' },
  ]
  return tabs.map((tab) => ({
    ...tab,
    count: visibleServiceTasks.value.filter((task) => taskMatchesReminderFilter(task, tab.value)).length,
  }))
})

const selectTask = (id: string) => {
  activeTaskId.value = id
}

const openCustomerDetail = (type: CustomerDetailType) => {
  customerDetailType.value = type
  customerDetailVisible.value = true
}

const goToActiveCustomerSpace = () => {
  const subjectId = activeTask.value.subjectId
  if (subjectId) {
    void router.push(`/platform/service/customers/${subjectId}`)
    return
  }
  void router.push('/platform/service/customers')
}

const closeCustomerDetail = () => {
  customerDetailVisible.value = false
}

const buildServiceAgentContext = (task: ServiceTask) => [
  '以下是从个人记忆收集到的服务提醒，并整理出的当前客户服务摘要，仅作为本轮对话上下文。不要逐字复述，除非用户明确要求。',
  '请把个人记忆视为事实层，把公共知识库视为课程、政策、话术和服务规则来源。事实不足时请指出需要补充哪条记忆。',
  '<service_customer_context>',
  `服务记忆范围：${activeServiceMemoryScope.value}`,
  `客户：${task.customerName}`,
  `学员：${task.studentName}`,
  `阶段：${task.stage}`,
  `渠道：${task.channel}`,
  '来源类型：个人记忆梳理',
  `来源记忆数：${serviceTaskSourceLabel(task)}`,
  `最近记忆：${task.lastMemoryLabel}`,
  `梳理置信度：${task.confidenceLabel}`,
  `事项：${task.title}`,
  `当前判断：${task.assistReason}`,
  `风险/关注点：${task.riskLabel}`,
  `决策角色：${task.decisionRole}`,
  `下一步：${task.nextAction}`,
  `处理策略：${task.primaryAction}`,
  `避免动作：${task.avoidAction}`,
  `服务信号：${task.memorySignals.join('、')}`,
  `待沉淀记录：${task.writeBackDraft}`,
  `建议话术：${task.replyDraft}`,
  '记忆证据：',
  ...task.memoryEvidence.map((memory, index) => `${index + 1}. ${memory.title}｜${memory.summary}｜${memory.sourceLabel}｜${memory.occurredAtLabel}`),
  '</service_customer_context>',
  '请围绕这个客户给出可执行、简洁、对服务人员有用的建议，并区分已由记忆支持的事实和需要确认的推断。',
].join('\n')

const buildServiceChatSessionTitle = (task: ServiceTask) => `${task.customerName} · ${task.title}`
const buildServiceChatSessionDescription = (task: ServiceTask) => `service-task:${task.id};agent:${serviceAssistantAgentId}`

const ensureServiceChatSession = async (task = activeTask.value) => {
  if (!task.id) return ''

  const existingSessionId = serviceChatSessionIds.value[task.id]
  if (existingSessionId) return existingSessionId

  const pendingRequest = serviceChatSessionRequests.get(task.id)
  if (pendingRequest) return pendingRequest

  const requestId = ++serviceChatSessionRequestSeed
  serviceChatSessionLoadingId.value = task.id
  serviceChatSessionError.value = ''

  const request = (async () => {
    try {
      const response = await createSessions({
        title: buildServiceChatSessionTitle(task),
        description: buildServiceChatSessionDescription(task),
      })
      const sessionId = response?.data?.id
      if (!sessionId) {
        throw new Error('missing session id')
      }
      serviceChatSessionIds.value = {
        ...serviceChatSessionIds.value,
        [task.id]: sessionId,
      }
      serviceChatHasMessagesByTask.value = {
        ...serviceChatHasMessagesByTask.value,
        [task.id]: false,
      }
      return sessionId
    } catch (error) {
      console.error('[ServiceAgent] Failed to create session:', error)
      serviceChatSessionError.value = '服务对话创建失败，请稍后重试'
      MessagePlugin.error(serviceChatSessionError.value)
      return ''
    } finally {
      serviceChatSessionRequests.delete(task.id)
      if (requestId === serviceChatSessionRequestSeed) {
        serviceChatSessionLoadingId.value = ''
      }
    }
  })()

  serviceChatSessionRequests.set(task.id, request)
  return request
}

const loadServiceMemories = async (force = false) => {
  if (serviceMemoriesLoading.value || (!force && serviceMemoriesLoaded.value)) return

  serviceMemoriesLoading.value = true
  serviceMemoriesError.value = ''
  try {
    const response = force ? await refreshServiceModule() : await getServiceBootstrap()
    const data = response?.data
    backendServiceTasks.value = (data?.reminders || []).map(mapServiceReminderToTask)
    serviceWorkProfile.value = data?.profile || null
    serviceBackendTotal.value = data?.total || backendServiceTasks.value.length
    serviceHasConfiguredProfile.value = Boolean(data?.profile?.id)
    closedTaskIds.value = backendServiceTasks.value
      .filter((task) => ['confirmed', 'completed'].includes(task.status || ''))
      .map((task) => task.id)
    ignoredTaskIds.value = backendServiceTasks.value
      .filter((task) => task.status === 'ignored')
      .map((task) => task.id)
    snoozedTaskIds.value = backendServiceTasks.value
      .filter((task) => task.status === 'snoozed')
      .map((task) => task.id)
  } catch (error) {
    console.warn('[ServiceAgent] Failed to load service reminders:', error)
    serviceMemoriesError.value = '服务提醒读取失败'
    backendServiceTasks.value = []
    serviceWorkProfile.value = null
    serviceBackendTotal.value = 0
    serviceHasConfiguredProfile.value = false
  } finally {
    serviceMemoriesLoaded.value = true
    serviceMemoriesLoading.value = false
  }
}

const serviceReportTimezone = () => Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'

const loadServiceDailyReports = async (force = false) => {
  if (serviceDailyReportsLoading.value || (!force && serviceDailyReportsLoaded.value)) return

  serviceDailyReportsLoading.value = true
  serviceDailyReportsError.value = ''
  try {
    const response = await listServiceDailyReports({
      range: reviewRange.value,
      page: 1,
      page_size: 20,
    })
    backendReviewReports.value = (response?.data?.items || []).map(mapServiceDailyReportToReviewReport)
  } catch (error) {
    console.warn('[ServiceAgent] Failed to load daily reports:', error)
    serviceDailyReportsError.value = '日报读取失败'
    backendReviewReports.value = []
  } finally {
    serviceDailyReportsLoaded.value = true
    serviceDailyReportsLoading.value = false
  }
}

const generateReviewReport = async () => {
  if (serviceDailyReportsGenerating.value) return

  serviceDailyReportsGenerating.value = true
  serviceDailyReportsError.value = ''
  try {
    const response = await generateServiceDailyReport({
      range: reviewRange.value,
      timezone: serviceReportTimezone(),
    })
    const report = mapServiceDailyReportToReviewReport(response.data)
    backendReviewReports.value = [
      report,
      ...backendReviewReports.value.filter((item) => item.id !== report.id),
    ]
    serviceDailyReportsLoaded.value = true
    openReviewPreview(report)
    MessagePlugin.success('日报已生成')
  } catch (error) {
    console.warn('[ServiceAgent] Failed to generate daily report:', error)
    serviceDailyReportsError.value = '日报生成失败'
    MessagePlugin.error('日报生成失败')
  } finally {
    serviceDailyReportsGenerating.value = false
  }
}

const loadServiceAgentSuggestions = async (force = false) => {
  if (serviceAgentSuggestionsLoading.value || (!force && serviceAgentSuggestionsLoaded.value)) return

  serviceAgentSuggestionsLoading.value = true
  try {
    const response = await getSuggestedQuestions(serviceAssistantAgentId, {
      limit: SERVICE_ASSISTANT_SUGGESTION_LIMIT,
    })
    serviceAgentSuggestions.value = response?.data?.questions ?? []
  } catch (error) {
    console.warn('[ServiceAgent] Failed to load suggested questions:', error)
    if (serviceAgentSuggestions.value.length === 0) {
      serviceAgentSuggestions.value = []
    }
  } finally {
    serviceAgentSuggestionsLoaded.value = true
    serviceAgentSuggestionsLoading.value = false
  }
}

const refreshServiceAgentSuggestions = () => {
  void loadServiceAgentSuggestions(true)
}

const handleServiceChatMessageState = (state: ServiceChatMessageState) => {
  const matchedTask = Object.entries(serviceChatSessionIds.value)
    .find(([, sessionId]) => sessionId === state.sessionId)
  const taskId = matchedTask?.[0] || activeTask.value.id
  serviceChatHasMessagesByTask.value = {
    ...serviceChatHasMessagesByTask.value,
    [taskId]: Boolean(state.hasMessages || (state.messageCount ?? 0) > 0),
  }
}

const sendServiceAgentPrompt = async (prompt: string) => {
  const query = prompt.trim()
  if (!query || !hasActiveServiceTask.value) return

  const sessionId = await ensureServiceChatSession(activeTask.value)
  if (!sessionId) return

  await nextTick()
  serviceChatViewRef.value?.triggerSend?.(query)
}

watch(
  () => [activeView.value, activeTask.value] as const,
  ([view]) => {
    if (view === 'messages' || view === 'review') {
      void loadServiceMemories()
    }

    if (view === 'review') {
      void loadServiceDailyReports()
    }

    if (view === 'messages') {
      void loadServiceAgentSuggestions()
      if (hasActiveServiceTask.value) {
        void ensureServiceChatSession(activeTask.value)
      }
    }
  },
  { immediate: true },
)

watch(
  visibleServiceTasks,
  (tasks) => {
    if (!tasks.length) return
    if (!tasks.some((task) => task.id === activeTaskId.value)) {
      activeTaskId.value = tasks[0]!.id
    }
  },
  { immediate: true },
)

const taskMatchesKeyword = (task: ServiceTask, q: string) => {
  if (!q) return true
  return [
    task.customerName,
    task.studentName,
    task.title,
    task.summary,
    task.stage,
    task.nextAction,
    task.channel,
    task.assistReason,
    task.riskLabel,
    task.avoidAction,
    task.writeBackDraft,
    task.lastMemoryLabel,
    task.confidenceLabel,
    ...task.contextItems,
    ...task.memorySignals,
    ...task.memoryEvidence.map((memory) => `${memory.title} ${memory.summary} ${memory.sourceLabel}`),
  ]
    .some((text) => normalize(text).includes(q))
}

const filteredTasks = computed(() => {
  const q = normalize(keyword.value)
  return visibleServiceTasks.value
    .filter((task) => taskMatchesReminderFilter(task, serviceReminderFilter.value))
    .filter((task) => taskMatchesKeyword(task, q))
})

const setReviewRange = (range: ReviewRange) => {
  reviewRange.value = range
  void loadServiceDailyReports(true)
}

const openReviewPreview = (report: ServiceReviewReport) => {
  activeReviewReport.value = report
  reviewPreviewVisible.value = true
}

const closeReviewPreview = () => {
  reviewPreviewVisible.value = false
}

</script>

<style scoped lang="less">
@import '../../components/css/suggested-questions.less';

.service-page {
  height: 100%;
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
  background: #f7f8fa;
  color: var(--td-text-color-primary);
}

.service-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  padding: 18px 0 0 24px;
  box-sizing: border-box;
}

.service-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 14px;
  padding-right: 24px;
}

.service-heading {
  min-width: 0;

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 20px;
    font-weight: 500;
    line-height: 30px;
    letter-spacing: 0;
  }

  p {
    margin: 2px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 20px;
  }
}

.service-header-tools {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 10px;
  flex: 0 0 auto;
  min-height: 34px;
  box-sizing: border-box;
}

.service-header-reminder {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  min-width: 0;
  gap: 1px;

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
    white-space: nowrap;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    white-space: nowrap;
  }
}

.service-scroll {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 24px 16px 0;
  box-sizing: border-box;
}

.assistant-workspace {
  display: grid;
  grid-template-columns: minmax(282px, 318px) minmax(420px, 1fr) minmax(236px, 270px);
  gap: 12px;
  align-items: stretch;
  flex: 1;
  width: 100%;
  max-width: 1320px;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.review-page {
  max-width: 1080px;
}

.review-panel {
  max-width: 860px;
}

.assistant-rail {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  border: 1px solid #e4e8ed;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
  overflow: hidden;
}

.rail-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid #e2e7ec;
  border-radius: 50%;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;

  &:hover:not(:disabled) {
    border-color: rgba(34, 101, 73, 0.28);
    background: rgba(34, 101, 73, 0.06);
    color: #236549;
  }

  &:disabled {
    cursor: default;
    opacity: 0.62;
  }
}

.rail-search {
  width: calc(100% - 24px);
  margin: 12px 12px 2px;

  :deep(.t-input) {
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
  }
}

.rail-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 22px;
  padding: 0 14px 4px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;

  span,
  em {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  span {
    color: var(--td-text-color-secondary);
  }

  em {
    flex: 0 0 auto;
    font-style: normal;
  }
}

.reminder-filter-tabs {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 4px;
  min-height: 32px;
  padding: 4px 12px 5px;
  overflow: hidden;
  box-sizing: border-box;
}

.reminder-filter-tab {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  min-height: 24px;
  padding: 2px 7px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 11px;
  line-height: 18px;
  white-space: nowrap;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;

  span {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  small {
    flex: 0 0 auto;
    color: var(--td-text-color-placeholder);
    font-size: 11px;
    line-height: 18px;
  }

  &:hover {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-primary);
  }

  &.active {
    border-color: transparent;
    background: #eaf2ee;
    color: #1c5a3f;
    font-weight: 600;
  }
}

.rail-list {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  gap: 0;
  padding: 4px 8px 8px;
}

.rail-item {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 70px;
  padding: 8px 9px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
  transition: background-color 0.16s ease, box-shadow 0.16s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.active {
    background: #f1f5f3;
    box-shadow: inset 3px 0 0 #236549;
  }

  &.muted {
    opacity: 0.58;
  }
}

.rail-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #8a99a8;
}

.rail-item.priority-high .rail-dot {
  background: #d54941;
}

.rail-item.priority-medium .rail-dot {
  background: #9a6b2f;
}

.rail-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: #e7eef8;
  color: #214a7a;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
}

.rail-item.priority-high .rail-avatar {
  background: rgba(213, 73, 65, 0.12);
  color: #b83f38;
}

.rail-item.priority-medium .rail-avatar {
  background: rgba(146, 94, 28, 0.12);
  color: #7a4d18;
}

.rail-item.priority-low .rail-avatar {
  background: rgba(34, 101, 73, 0.1);
  color: #236549;
}

.rail-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 2px;

  strong,
  em,
  small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
  }

  em {
    color: var(--td-text-color-secondary);
    font-style: normal;
    font-size: 12px;
    line-height: 18px;
  }

  small {
    flex: 0 0 auto;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    font-weight: 400;
    line-height: 18px;
  }
}

.rail-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  min-width: 0;

  strong {
    flex: 1;
    min-width: 0;
  }
}

.rail-due {
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
}

.rail-tags {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;

  small {
    display: inline-flex;
    align-items: center;
    flex: 0 1 auto;
    min-width: 0;
    max-width: 58%;
    min-height: 18px;
    padding: 0 6px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    line-height: 16px;
  }
}

.reminder-status {
  border: 1px solid rgba(34, 101, 73, 0.14);
  background: rgba(34, 101, 73, 0.08) !important;
  color: #236549 !important;
}

.reminder-status--warning {
  border-color: rgba(213, 73, 65, 0.16);
  background: rgba(213, 73, 65, 0.1) !important;
  color: #b83f38 !important;
}

.reminder-status--done {
  border-color: rgba(45, 116, 238, 0.16);
  background: rgba(45, 116, 238, 0.08) !important;
  color: #2365d6 !important;
}

.reminder-status--muted {
  border-color: var(--td-component-stroke);
  background: var(--td-bg-color-secondarycontainer) !important;
  color: var(--td-text-color-placeholder) !important;
}

.service-chat-main {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  box-sizing: border-box;
}

.service-agent-shell {
  display: flex;
  flex-direction: column;
  width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  border: 1px solid #e3e8ee;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
}

.service-reminder-detail {
  flex: 0 0 auto;
  padding: 16px 18px 14px;
  border-bottom: 1px solid #edf0f3;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
}

.service-reminder-detail-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  min-width: 0;

  div {
    min-width: 0;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  h3 {
    margin: 2px 0 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 17px;
    font-weight: 700;
    line-height: 26px;
    letter-spacing: 0;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  p {
    margin: 2px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  em {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    min-height: 24px;
    padding: 2px 8px;
    border-radius: 7px;
    font-style: normal;
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    box-sizing: border-box;
  }
}

.service-reminder-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  margin-top: 14px;
  border: 1px solid #edf0f3;
  border-radius: 8px;
  background: #fafbfc;
  overflow: hidden;

  section {
    min-width: 0;
    min-height: 82px;
    padding: 11px 12px;
    border-right: 1px solid #edf0f3;
    background: transparent;
    box-sizing: border-box;

    &:last-child {
      border-right: 0;
    }
  }

  h4 {
    margin: 0 0 6px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    letter-spacing: 0;
  }

  p {
    display: -webkit-box;
    margin: 0;
    max-height: 60px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 12px;
    line-height: 20px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
  }
}

.service-reminder-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
}

.service-action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 28px;
  padding: 3px 10px;
  border: 1px solid #dfe5e9;
  border-radius: 7px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;

  &:hover:not(:disabled) {
    border-color: rgba(45, 116, 238, 0.32);
    background: #f4f8ff;
    color: #2365d6;
  }

  &:disabled {
    cursor: default;
    opacity: 0.62;
  }

  &.primary {
    border-color: rgba(34, 101, 73, 0.28);
    background: #236549;
    color: #ffffff;

    &:hover:not(:disabled) {
      border-color: #1d543d;
      background: #1d543d;
      color: #ffffff;
    }
  }
}

.service-header-action {
  min-height: 30px;
}

.service-agent-chat-shell {
  display: flex;
  flex: 1;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  background: var(--td-bg-color-container);
}

.service-agent-chat-shell :deep(.chat) {
  display: flex;
  flex-direction: column;
  flex: 1;
  width: 100%;
  min-width: 0;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: var(--td-bg-color-container);
}

.service-agent-chat-shell :deep(.chat_scroll_box) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 18px 0;
}

.service-agent-chat-shell :deep(.msg_list) {
  max-width: 920px;
  min-height: 100%;
  gap: 14px;
}

.service-agent-empty-suggestions.suggested-questions-container {
  max-width: 920px;
  margin: 0 auto;
  padding: 12px 0 18px;
}

.sq-fade-enter-active,
.sq-fade-leave-active {
  transition: opacity 0.25s @suggested-ease;
}

.sq-fade-enter-from,
.sq-fade-leave-to {
  opacity: 0;
}

.service-suggested-question {
  border: 1px solid var(--td-component-stroke);
  font-family: var(--app-font-family);
  text-align: left;
}

.service-agent-chat-shell :deep(.input-container.is-embedded) {
  padding: 12px 16px 14px;
  border-top: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.service-agent-chat-shell :deep(.answers-input.is-embedded .rich-input-container) {
  border-radius: 8px;
  box-shadow: 0 10px 24px -24px rgba(17, 35, 62, 0.42);
}

.service-agent-state {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  gap: 8px;
  width: 100%;
  min-height: 0;
  padding: 24px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
  box-sizing: border-box;
}

.service-agent-state--empty {
  flex-direction: column;
  min-height: 420px;
  border: 1px dashed var(--td-component-border);
  border-radius: 8px;
  text-align: center;

  .t-icon {
    color: var(--td-text-color-placeholder);
    font-size: 22px;
  }
}

.customer-summary-panel {
  align-self: stretch;
  min-width: 0;
  min-height: 0;
  max-height: 100%;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 13px 14px;
  border: 1px solid #e3e8ee;
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-shadow: none;
  box-sizing: border-box;
}

.customer-summary-panel--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.customer-summary-empty {
  max-width: 190px;
  text-align: center;

  strong {
    display: block;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 22px;
  }

  p {
    margin: 6px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 20px;
  }
}

.customer-summary-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  span {
    color: var(--td-text-color-secondary);
    font-size: 13px;
    font-weight: 600;
    line-height: 22px;
  }
}

.customer-summary-head-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.customer-summary-status {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: 0 0 auto;
  gap: 5px;
  min-height: 26px;
  padding: 0 8px;
  border: 1px solid #e2e7ec;
  border-radius: 7px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
  transition: background-color 0.16s ease, border-color 0.16s ease, color 0.16s ease;

  &:hover {
    border-color: rgba(45, 116, 238, 0.32);
    background: #f4f8ff;
    color: #2365d6;
  }
}

.customer-summary-person {
  position: relative;
  margin-top: 10px;
  padding-bottom: 10px;
  border-bottom: 1px solid #edf0f3;

  p {
    margin: 3px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.customer-summary-name-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;

  h3 {
    flex: 1;
    min-width: 0;
    margin: 0;
    overflow: hidden;
    color: #0f1828;
    font-size: 16px;
    font-weight: 700;
    line-height: 26px;
    letter-spacing: 0;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  em {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 auto;
    min-height: 22px;
    padding: 1px 7px;
    border-radius: 6px;
    font-style: normal;
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    box-sizing: border-box;

    &.priority-high {
      background: rgba(213, 73, 65, 0.12);
      color: #b83f38;
    }

    &.priority-medium {
      background: rgba(154, 107, 47, 0.12);
      color: #785324;
    }

    &.priority-low {
      background: rgba(42, 156, 122, 0.12);
      color: #127354;
    }
  }
}

.customer-summary-facts {
  display: flex;
  flex-direction: column;
  gap: 0;
  margin: 8px 0 0;
  padding: 5px 0;
  border-bottom: 1px solid #edf0f3;

  div {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    min-width: 0;
    min-height: 28px;
    padding: 3px 0;
    box-sizing: border-box;
  }

  div:first-child {
    min-height: 30px;
    margin: 0 0 3px;
    padding: 5px 8px;
    border-radius: 7px;
    background: #f7f9fb;
  }

  dt {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }

  dd {
    margin: 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.assistant-result-card {
  margin-top: 8px;
  padding: 9px 0 0;
  border: 0;
  border-top: 1px solid #edf0f3;
  border-radius: 0;
  background: transparent;
  box-sizing: border-box;

  h4 {
    margin: 0 0 5px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    letter-spacing: 0;
  }

  p {
    display: -webkit-box;
    margin: 0;
    max-height: 56px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 12px;
    line-height: 20px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    position: relative;
    padding-left: 11px;
    color: var(--td-text-color-primary);
    font-size: 12px;
    line-height: 20px;

    &::before {
      position: absolute;
      top: 8px;
      left: 0;
      width: 5px;
      height: 5px;
      border-radius: 50%;
      background: #236549;
      content: '';
    }
  }
}

.assistant-result-card--actions {
  background: transparent;
}

.assistant-result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  margin-top: 7px;

  span {
    min-height: 20px;
    padding: 0 6px;
    border: 1px solid #dce7e1;
    border-radius: 6px;
    background: rgba(34, 101, 73, 0.05);
    color: #236549;
    font-size: 12px;
    line-height: 18px;
    box-sizing: border-box;
  }
}

.customer-summary-actions {
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-top: 7px;
}

.customer-summary-entry {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
  min-height: 40px;
  padding: 6px 7px;
  border: 0;
  border-radius: 7px;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  font-family: var(--app-font-family);
  text-align: left;
  box-sizing: border-box;
  transition: background-color 0.16s ease, border-color 0.16s ease, box-shadow 0.16s ease;

  &:hover {
    background: #f7faff;
    color: #2365d6;
    box-shadow: none;
  }

  span {
    display: flex;
    flex-direction: column;
    min-width: 0;
    gap: 3px;
  }

  strong,
  em {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
  }

  em {
    color: var(--td-text-color-secondary);
    font-style: normal;
    font-size: 12px;
    line-height: 18px;
  }
}

.customer-summary-draft {
  margin-top: 8px;
  padding: 9px;
  border: 0;
  border-radius: 7px;
  background: #f3f7f5;
  box-sizing: border-box;

  h4 {
    margin: 0 0 6px;
    color: #236549;
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    letter-spacing: 0;
  }

  p {
    display: -webkit-box;
    margin: 0;
    max-height: 56px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 12px;
    line-height: 20px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 3;
  }
}

.customer-summary-source {
  margin-top: 8px;
  padding-top: 9px;
  border-top: 1px solid #edf0f3;

  h4 {
    margin: 0 0 8px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
    letter-spacing: 0;
  }

  p {
    display: -webkit-box;
    margin: 0 0 8px;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 20px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
}

.customer-summary-open-space {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  min-height: 34px;
  padding: 0 10px;
  border: 1px solid #dce7e1;
  border-radius: 8px;
  background: #f7fbf8;
  color: #236549;
  cursor: pointer;
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 600;
  line-height: 18px;
  transition: background-color 0.16s ease, border-color 0.16s ease;

  &:hover {
    border-color: rgba(35, 101, 73, 0.34);
    background: #eef7f1;
  }
}

.customer-memory-evidence-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.customer-memory-evidence {
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  box-sizing: border-box;

  strong {
    display: block;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  p {
    display: -webkit-box;
    margin: 4px 0;
    max-height: 44px;
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 20px;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  span {
    display: block;
    overflow: hidden;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.service-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 96px;
  border: 1px dashed var(--td-component-border);
  border-radius: 8px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.review-report-page {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.review-hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 64px;
  padding: 8px 12px;
  border: 1px solid #e4dbcc;
  border-radius: 8px;
  background:
    linear-gradient(90deg, rgba(34, 101, 73, 0.06), transparent 38%),
    linear-gradient(135deg, rgba(164, 128, 57, 0.08), rgba(255, 255, 255, 0) 48%),
    #fffdf8;
  box-sizing: border-box;
}

.review-hero-copy {
  min-width: 0;

  p {
    max-width: 620px;
    margin: 4px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.review-heading {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--td-text-color-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
}

.review-heading-icon {
  width: 18px;
  height: 18px;
  color: #236549;
}

.review-hero-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(86px, 1fr));
  gap: 10px;
  flex: 0 0 auto;

  span {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-height: 42px;
    justify-content: center;
    padding: 4px 12px;
    border: 1px solid rgba(34, 101, 73, 0.12);
    border-radius: 8px;
    background: rgba(255, 255, 255, 0.72);
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
    box-sizing: border-box;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 12px;
    font-weight: 600;
    line-height: 18px;
  }
}

.review-month-list,
.review-month-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.review-month-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    letter-spacing: 0;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.review-month-heading {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  min-width: 0;
}

.segmented-tabs {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.segmented-tab {
  min-width: 56px;
  height: auto;
  padding: 0;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font-size: 12px;
  line-height: 18px;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &--active {
    color: var(--td-text-color-primary);
    font-weight: 600;
  }
}

.review-range-tabs {
  flex: 0 0 auto;
  gap: 28px;
}

.report-list {
  display: flex;
  flex-direction: column;
}

.report-card--editable {
  cursor: pointer;
}

.review-report-list {
  max-width: none;
  gap: 10px;
}

.review-report-card {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  min-height: 124px;
  overflow: hidden;
  border: 1px solid #e1d7c7;
  border-radius: 8px;
  background:
    linear-gradient(0deg, rgba(35, 31, 27, 0.018) 1px, transparent 1px),
    linear-gradient(90deg, rgba(35, 31, 27, 0.014) 1px, transparent 1px),
    #fffdf8;
  background-size: 22px 22px;
  box-shadow: 0 4px 14px rgba(38, 34, 29, 0.05);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;

  &:hover {
    border-color: rgba(34, 101, 73, 0.42);
    box-shadow: 0 10px 26px rgba(38, 34, 29, 0.1);
  }
}

.review-report-gutter {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100%;
  border-right: 1px solid #e6ddcf;
  background:
    linear-gradient(180deg, rgba(34, 101, 73, 0.08), rgba(164, 128, 57, 0.08)),
    #f8f2e7;
}

.review-report-ribbon {
  display: grid;
  place-items: center;
  width: 48px;
  height: 56px;
  border: 2px solid #20242a;
  background: rgba(255, 253, 248, 0.72);
  color: var(--td-text-color-primary);
  font-family: "Songti SC", "STSong", serif;
  font-size: 12px;
  line-height: 18px;
  letter-spacing: 0;
  text-align: center;
  box-shadow: inset 0 0 0 1px rgba(32, 36, 42, 0.12);

  span {
    display: block;
  }
}

.review-report-main {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 12px;

  h2 {
    display: -webkit-box;
    margin: 0;
    max-height: 18px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 18px;
    letter-spacing: 0;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 1;
  }
}

.review-report-title-row {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  min-width: 0;
  margin: 2px 0 6px;

  h2 {
    flex: 1;
    min-width: 0;
  }

  .type-badge {
    flex-shrink: 0;
    min-height: 20px;
    padding: 1px 7px;
    border-radius: 999px;
    font-size: 12px;
    line-height: 16px;
  }
}

.review-report-intro {
  display: -webkit-box;
  margin: 0;
  max-height: 54px;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}

.report-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 0;

  span {
    padding: 1px 7px;
    border-radius: 999px;
    background: rgba(34, 101, 73, 0.08);
    color: #236549;
    font-size: 12px;
    line-height: 18px;
  }
}

.review-report-main .report-chips {
  margin-top: auto;
  margin-bottom: 6px;
}

.report-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 8px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.review-report-meta {
  padding-top: 6px;
}

.review-report-meta-separator {
  color: var(--td-text-color-placeholder);
}

.type-badge.review-stage--formed {
  background: rgba(34, 101, 73, 0.1);
  color: #236549;
}

.type-badge.review-stage--expandable {
  background: rgba(146, 94, 28, 0.1);
  color: #7a4d18;
}

.type-badge.review-stage--organizing {
  background: rgba(35, 99, 148, 0.1);
  color: #1f5a86;
}

:deep(.customer-detail-drawer .t-drawer__body) {
  padding: 0;
  background: var(--td-bg-color-page);
}

.customer-detail-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 64px;
  padding: 12px 18px;
  border-bottom: 1px solid var(--td-component-stroke);
  background: rgba(255, 255, 255, 0.96);
  box-sizing: border-box;

  div {
    min-width: 0;
  }

  span {
    display: block;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  strong {
    display: block;
    margin-top: 2px;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.customer-detail-close {
  width: 30px !important;
  min-width: 30px !important;
  height: 30px !important;
  padding: 0 !important;
  border-radius: 6px !important;
}

.customer-detail-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  box-sizing: border-box;
}

.customer-detail-profile {
  padding: 0 0 14px;
  border-bottom: 1px solid var(--td-component-stroke);

  h3 {
    margin: 0;
    color: #0f1828;
    font-size: 22px;
    font-weight: 700;
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

.customer-detail-section {
  padding: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;

  h4 {
    margin: 0 0 8px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    letter-spacing: 0;
  }

  p {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 14px;
    line-height: 24px;
  }

  blockquote {
    margin: 0;
    padding: 12px 14px 12px 16px;
    border-left: 3px solid #2d7a52;
    border-radius: 0 7px 7px 0;
    background: rgba(34, 101, 73, 0.06);
    color: #394a42;
    font-size: 14px;
    line-height: 24px;
  }
}

.customer-detail-section--warning {
  border-color: rgba(213, 73, 65, 0.12);
  background: rgba(213, 73, 65, 0.06);
}

.customer-detail-steps {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding-left: 20px;

  li {
    color: var(--td-text-color-primary);
    font-size: 14px;
    line-height: 24px;
  }
}

.customer-detail-facts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 0;

  div {
    min-width: 0;
    padding: 9px 10px;
    border: 1px solid var(--td-component-stroke);
    border-radius: 7px;
    background: var(--td-bg-color-secondarycontainer);
    box-sizing: border-box;
  }

  dt {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 18px;
  }

  dd {
    margin: 2px 0 0;
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 13px;
    font-weight: 600;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.customer-detail-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;

  span {
    min-height: 24px;
    padding: 2px 8px;
    border: 1px solid var(--td-component-stroke);
    border-radius: 6px;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
    box-sizing: border-box;
  }
}

:deep(.review-preview-drawer .t-drawer__body) {
  padding: 0;
  background: #f5f0e7;
}

.review-preview-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 64px;
  padding: 12px 18px;
  border-bottom: 1px solid rgba(32, 36, 42, 0.1);
  background: rgba(255, 255, 255, 0.94);
  box-sizing: border-box;
}

.review-preview-header-copy {
  min-width: 0;
}

.review-preview-eyebrow {
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
}

.review-preview-title {
  overflow: hidden;
  color: var(--td-text-color-primary);
  font-size: 12px;
  line-height: 18px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.review-preview-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 0 0 auto;
}

.review-preview-action {
  width: 30px !important;
  min-width: 30px !important;
  height: 30px !important;
  padding: 0 !important;
  border-radius: 6px !important;
}

.review-preview-page {
  min-height: 100%;
  padding: 22px 26px 48px;
  box-sizing: border-box;
}

.review-preview-body {
  margin-top: 0;
  padding: 30px 34px 38px;
  border: 1px solid #ded2bf;
  border-radius: 8px;
  background: #fffdf8;
  color: var(--td-text-color-primary);
  box-shadow: 0 10px 26px rgba(38, 34, 29, 0.08);
  box-sizing: border-box;

  h1 {
    margin: 12px 0 18px;
    color: var(--td-text-color-primary);
    font-size: 18px;
    font-weight: 600;
    line-height: 28px;
    letter-spacing: 0;
  }
}

.review-preview-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
}

.review-preview-content {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 22px;

  :deep(h1) {
    display: none;
  }

  :deep(h2) {
    margin: 30px 0 12px;
    padding-top: 18px;
    border-top: 1px solid rgba(32, 36, 42, 0.12);
    color: var(--td-text-color-primary);
    font-size: 15px;
    font-weight: 600;
    line-height: 22px;
    letter-spacing: 0;
  }

  :deep(h3) {
    margin: 24px 0 10px;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
    letter-spacing: 0;
  }

  :deep(p) {
    margin: 10px 0;
  }

  :deep(> p:first-child) {
    margin: 0 0 18px;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 22px;
  }

  :deep(blockquote) {
    margin: 14px 0;
    padding: 11px 14px 11px 16px;
    border-left: 3px solid #2d7a52;
    border-radius: 0 6px 6px 0;
    background: rgba(34, 101, 73, 0.06);
    color: #4e5a52;
  }

  :deep(blockquote p) {
    margin: 4px 0;
  }

  :deep(strong) {
    color: var(--td-text-color-primary);
    font-weight: 600;
  }

  :deep(ul),
  :deep(ol) {
    margin: 10px 0 14px;
    padding-left: 22px;
  }

  :deep(li) {
    margin: 4px 0;
  }
}

@media (max-width: 1180px) {
  .assistant-workspace {
    grid-template-columns: minmax(230px, 280px) minmax(0, 1fr);
  }

  .customer-summary-panel {
    grid-column: 2;
  }

  .review-page,
  .review-panel {
    max-width: none;
  }

}

@media (max-width: 760px) {
  :global(.main:has(.service-page)) {
    min-width: 0;
  }

  :global(.main:has(.service-page) .aside_box) {
    display: none;
  }

  .service-page {
    height: auto;
    min-height: 100%;
    overflow-y: auto;
  }

  .service-main {
    min-height: 100%;
    overflow: visible;
    padding: 16px 0 16px 16px;
  }

  .service-header {
    align-items: stretch;
    flex-direction: column;
    padding-right: 16px;
  }

  .service-header-tools {
    align-self: flex-start;
    max-width: 100%;
    flex-wrap: wrap;
  }

  .service-header-reminder {
    align-items: flex-start;

    strong {
      white-space: normal;
    }
  }

  .service-scroll {
    flex: 0 0 auto;
    overflow: visible;
    padding-right: 16px;
  }

  .assistant-workspace {
    grid-template-columns: 1fr;
    align-items: start;
    height: auto;
    min-height: auto;
    overflow: visible;
  }

  .assistant-rail {
    min-height: 420px;
    max-height: min(560px, calc(100vh - 168px));
  }

  .service-chat-main {
    height: auto;
    min-height: 560px;
    overflow: visible;
  }

  .service-agent-shell {
    height: auto;
    min-height: 560px;
  }

  .service-agent-chat-shell {
    min-height: 320px;
  }

  .service-reminder-grid {
    grid-template-columns: 1fr;
  }

  .service-reminder-detail-head {
    flex-direction: column;
    align-items: flex-start;

    h3,
    p {
      white-space: normal;
    }
  }

  .customer-summary-panel {
    grid-column: auto;
    max-height: none;
    overflow: visible;
  }

  .customer-summary-facts,
  .customer-detail-facts {
    grid-template-columns: 1fr;
  }

  .review-hero,
  .review-month-header {
    align-items: flex-start;
    flex-direction: column;
  }

  .review-hero-stats {
    width: 100%;
  }

  .review-report-card {
    grid-template-columns: 64px minmax(0, 1fr);
  }

  .review-report-ribbon {
    width: 40px;
    height: 52px;
  }

  .review-preview-page {
    padding: 16px 14px 32px;
  }

  .review-preview-body {
    padding: 22px 18px 28px;
  }
}
</style>
