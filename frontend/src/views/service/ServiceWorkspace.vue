<template>
  <div class="service-page">
    <main class="service-main">
      <header class="service-header">
        <div class="service-heading">
          <h2>{{ pageMeta.title }}</h2>
          <p>{{ pageMeta.description }}</p>
        </div>

        <t-input
          v-if="activeView === 'messages'"
          v-model="keyword"
          class="service-search"
          clearable
          placeholder="搜索客户"
        >
          <template #prefix-icon>
            <t-icon name="search" />
          </template>
        </t-input>
      </header>

      <div class="service-scroll">
        <section v-if="activeView === 'messages'" class="service-workspace">
          <aside class="simple-panel list-panel">
            <div class="panel-head">
              <div>
                <h3>客户列表</h3>
                <span>{{ filteredTasks.length }} 位客户</span>
              </div>
              <t-radio-group v-model="taskFilter" size="small" variant="default-filled">
                <t-radio-button value="all">全部</t-radio-button>
                <t-radio-button value="urgent">优先</t-radio-button>
              </t-radio-group>
            </div>

            <div class="simple-list" role="list">
              <button
                v-for="task in filteredTasks"
                :key="task.id"
                type="button"
                class="customer-row"
                :class="[
                  `priority-${task.priorityKey}`,
                  { active: activeTask.id === task.id, muted: isTaskClosed(task.id) },
                ]"
                @click="selectTask(task.id)"
              >
                <span class="customer-avatar">{{ task.avatar }}</span>
                <span class="customer-copy">
                  <span class="customer-title-line">
                    <strong>{{ task.customerName }}</strong>
                  </span>
                  <em>{{ task.requiredServiceAction }}</em>
                </span>
                <span class="customer-meta">
                  <span class="customer-time">{{ task.lastLabel }}</span>
                  <span v-if="!isTaskClosed(task.id) && task.unreadCount > 0" class="customer-unread">
                    {{ task.unreadCount }}
                  </span>
                </span>
              </button>

              <div v-if="filteredTasks.length === 0" class="service-empty">暂无匹配客户</div>
            </div>
          </aside>

          <article class="focus-panel">
            <div class="focus-head">
              <div>
                <span>客户信息 · {{ activeTask.stage }}</span>
                <h3>{{ activeTask.customerName }}</h3>
              </div>
              <t-button size="small" variant="outline" @click="toggleTaskClosed(activeTask.id)">
                <template #icon>
                  <t-icon :name="isTaskClosed(activeTask.id) ? 'rollback' : 'check-circle'" />
                </template>
                {{ isTaskClosed(activeTask.id) ? '恢复' : '闭环' }}
              </t-button>
            </div>

            <section class="next-card">
              <span>{{ activeTask.studentName }} · 下一步</span>
              <strong>{{ activeTask.nextAction }}</strong>
            </section>

            <section class="plain-section">
              <h3>当前事项</h3>
              <p>{{ activeTask.title }}：{{ activeTask.summary }}</p>
            </section>

            <section class="plain-section">
              <h3>AI 建议</h3>
              <p>{{ activeTask.advice }}</p>
            </section>

            <section class="plain-section">
              <h3>可直接发送</h3>
              <t-textarea
                v-model="composerText"
                class="reply-box"
                placeholder="编辑回复内容"
                :autosize="{ minRows: 3, maxRows: 5 }"
              />
              <div class="action-row">
                <t-button variant="text" @click="composerText = activeTask.replyDraft">
                  <template #icon><t-icon name="copy" /></template>
                  恢复建议
                </t-button>
                <t-button theme="primary">
                  <template #icon><t-icon name="send" /></template>
                  发送
                </t-button>
              </div>
            </section>

            <section class="plain-section">
              <h3>客户记忆</h3>
              <ul class="simple-bullets">
                <li v-for="memory in activeTask.memories" :key="memory">{{ memory }}</li>
              </ul>
            </section>

            <section class="plain-section">
              <h3>最近跟进</h3>
              <div class="simple-events">
                <article v-for="event in activeTask.events" :key="event.time + event.title">
                  <time>{{ event.time }}</time>
                  <div>
                    <strong>{{ event.title }}</strong>
                    <p>{{ event.content }}</p>
                  </div>
                </article>
              </div>
            </section>
          </article>
        </section>

        <section v-else class="review-page review-report-page">
          <div class="review-hero">
            <div class="review-hero-copy">
              <div class="review-heading">
                <OrganizeSproutIcon class="review-heading-icon" />
                <span>服务复盘报告</span>
              </div>
              <p>沉淀客户服务动作、沟通风险和下阶段建议，形成个人可回看的服务复盘。</p>
            </div>
            <div class="review-hero-stats">
              <span><strong>{{ filteredReviewReports.length }}</strong> 份复盘</span>
              <span><strong>{{ reviewTotalActions }}</strong> 个动作</span>
            </div>
          </div>

          <div class="review-month-list">
            <section v-for="group in reviewReportGroups" :key="group.key" class="review-month-group">
              <div class="review-month-header">
                <div class="review-month-heading">
                  <h3>{{ group.label }}</h3>
                  <div class="segmented-tabs review-range-tabs" role="tablist" aria-label="服务复盘时间筛选">
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
                <span>{{ filteredReviewReports.length }} 份复盘</span>
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
                      <span>服务</span>
                      <span>复盘</span>
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

              <div v-else class="service-empty">暂无服务复盘</div>
            </section>
          </div>
        </section>
      </div>
    </main>

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
            <div class="review-preview-eyebrow">服务复盘</div>
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
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  findServiceMenuRoute,
  isServiceTab,
  type ServiceTab,
} from './serviceRoutes'
import {
  buildSproutReportPreview,
  sproutReportContentForEditor,
  type SproutReportPreviewSection,
} from '../organize/sproutReport'
import OrganizeSproutIcon from '../organize/components/OrganizeSproutIcon.vue'

type TaskFilter = 'all' | 'urgent'
type ReviewRange = 'week' | 'month'
type PriorityKey = 'high' | 'medium' | 'low'
type ReviewStageKey = 'formed' | 'expandable' | 'organizing'

interface ServiceTask {
  id: string
  customerName: string
  studentName: string
  avatar: string
  title: string
  summary: string
  advice: string
  stage: string
  priorityKey: PriorityKey
  time: string
  lastLabel: string
  requiredServiceAction: string
  unreadCount: number
  nextAction: string
  memories: string[]
  events: Array<{ time: string; title: string; content: string }>
  replyDraft: string
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
const taskFilter = ref<TaskFilter>('all')
const reviewRange = ref<ReviewRange>('week')
const reviewPreviewVisible = ref(false)
const activeReviewReport = ref<ServiceReviewReport | null>(null)
const activeTaskId = ref('task-1')
const closedTaskIds = ref<string[]>(['task-4'])
const composerText = ref('')

const serviceTasks: ServiceTask[] = [
  {
    id: 'task-1',
    customerName: '林晨家长',
    studentName: '林晨',
    avatar: '林',
    title: '试听后回访',
    summary: '家长关注课程效果和孩子适应度。',
    advice: '先肯定孩子试听表现，再说明后续 4 周的适应目标，最后确认是否需要园长补充沟通。',
    stage: '售前试听',
    priorityKey: 'high',
    time: '09:20',
    lastLabel: '星期一',
    requiredServiceAction: '16:00 前微信回访，确认适应期安排',
    unreadCount: 11,
    nextAction: '今天 16:00 前完成微信回访',
    memories: ['家长希望孩子先适应集体环境，再考虑长期课程。', '孩子初次到园较依赖妈妈，但能跟随老师完成指令。', '家长对价格敏感，认可课程效果后接受度更高。'],
    events: [
      { time: '08-25', title: '完成试听', content: '参与感统活动 35 分钟，后半段状态明显放松。' },
      { time: '08-24', title: '线上咨询', content: '询问课程频次、费用和入园适应周期。' },
    ],
    replyDraft: '您好，林晨昨天试听时在活动里愿意尝试新环节，这一点很值得肯定。后续建议先建立入园安全感，再逐步提升参与度。我整理了一份适应期安排，发您参考一下？',
  },
  {
    id: 'task-2',
    customerName: '周予安家长',
    studentName: '周予安',
    avatar: '周',
    title: '餐食反馈未闭环',
    summary: '售后问题已超过 48 小时。',
    advice: '先确认这两天体验是否改善，再同步老师观察记录，避免解释过多。',
    stage: '在园服务',
    priorityKey: 'high',
    time: '10:05',
    lastLabel: '星期六',
    requiredServiceAction: '上午回访，确认餐食处理满意度',
    unreadCount: 3,
    nextAction: '上午回访餐食问题满意度',
    memories: ['家长重视生活照和细节反馈。', '孩子午餐摄入少时，家长会比较焦虑。', '之前对老师及时反馈的评价较好。'],
    events: [
      { time: '08-24', title: '餐食反馈', content: '家长反馈孩子回家后说午餐吃得少。' },
      { time: '08-26', title: '待回访', content: '尚未确认家长是否满意当前处理。' },
    ],
    replyDraft: '您好，这两天我们重点观察了予安的午餐情况，也做了进餐陪伴上的调整。想跟您确认一下孩子回家后的状态是否有改善，我也把这两天的观察发您看一下。',
  },
  {
    id: 'task-3',
    customerName: '陈屿家长',
    studentName: '陈屿',
    avatar: '陈',
    title: '准备续费前回顾',
    summary: '本期课剩余 3 次。',
    advice: '不要直接推续费，先把过去 6 周的变化整理成成长线，再讨论下一阶段目标。',
    stage: '续费服务',
    priorityKey: 'medium',
    time: '11:30',
    lastLabel: '08/18',
    requiredServiceAction: '生成 6 周成长回顾后再沟通',
    unreadCount: 16,
    nextAction: '生成 6 周成长回顾',
    memories: ['家长最关注表达主动性。', '近 4 次课堂反馈稳定。', '家庭时间主要在周末，适合周五晚上沟通。'],
    events: [
      { time: '08-19', title: '课后沟通', content: '老师反馈孩子表达意愿提升。' },
      { time: '08-26', title: '续费窗口', content: '剩余 3 次课，适合先做阶段回顾。' },
    ],
    replyDraft: '这段时间陈屿在课堂表达上有几个很明显的变化，我想先跟您同步一下，也听听您对下一阶段的关注点，我们再一起判断后面的安排。',
  },
  {
    id: 'task-4',
    customerName: '何嘉木家长',
    studentName: '何嘉木',
    avatar: '何',
    title: '公开课资料轻提醒',
    summary: '家长上次主动询问亲子活动。',
    advice: '以活动邀请为主，不直接索要转介绍；反馈积极后再补充简版资料。',
    stage: '转介绍',
    priorityKey: 'low',
    time: '昨天',
    lastLabel: '08/07',
    requiredServiceAction: '周五前轻提醒亲子公开课资料',
    unreadCount: 1,
    nextAction: '周五前轻提醒公开课资料',
    memories: ['家长愿意分享孩子课堂照片。', '对亲子活动参与度高。', '不喜欢过强销售感，适合自然邀请。'],
    events: [
      { time: '08-22', title: '活动咨询', content: '主动询问近期是否有亲子公开课。' },
      { time: '08-25', title: '发送资料', content: '已发送公开课安排，家长回复会看看。' },
    ],
    replyDraft: '本周有一场亲子公开课，内容比较轻松，如果您或朋友刚好感兴趣，我把活动安排发您参考。',
  },
]

const reviewRangeTabs: Array<{ label: string; value: ReviewRange }> = [
  { label: '本周', value: 'week' },
  { label: '本月', value: 'month' },
]

const serviceReviewReportSources: Array<Omit<ServiceReviewReport, 'renderedHtml' | 'intro' | 'previewSections'>> = [
  {
    id: 'review-week-1',
    title: '服务闭环稳定，售后风险需要更早提醒',
    content: `# 服务复盘报告

本周处理 18 个服务动作，14 个已闭环。售前试听回访较及时，在园售后问题的完整闭环偏慢，下周需要把首次回应控制在 24 小时内。

## 1、售前回访

### 观察

- 3 位试听家长均在 24 小时内完成首次回访。
- 价格顾虑和适应焦虑是本周最常见的问题。
- 有明确下一步动作的客户，后续回复更顺畅。

### 建议

先肯定孩子试听表现，再把 4 周适应目标讲清楚，最后确认是否需要园长补充沟通。不要一开始就进入价格解释。

## 2、售后问题

### 观察

- 餐食反馈类问题平均超过 36 小时才形成完整闭环。
- 家长更在意是否被及时回应，而不是一次性解释所有原因。

### 建议

售后问题先回应再解释：先确认感受和处理时间，再补充老师观察记录。

## 3、下周动作

- 周五前完成陈屿家长的续费前成长回顾。
- 把适应期高频回应整理到公共知识库。
- 餐食、午睡、生活照三类问题统一使用“先回应、再观察、再闭环”的节奏。`,
    stage: '已生成',
    stageKey: 'formed',
    range: 'week',
    updated: '今天 09:30',
    updatedAt: '2026-08-27T09:30:00+08:00',
    actionCount: 18,
    customerCount: 9,
    chips: ['售前回访', '售后闭环', '适应期'],
  },
  {
    id: 'review-week-2',
    title: '续费前回顾需要提前铺垫',
    content: `# 服务复盘报告

本周有 3 位客户进入续费前服务窗口，其中 1 位已经有明确沟通节点。直接提续费会增加压力，更适合先用成长记录打开对话。

## 1、客户变化

### 观察

- 陈屿近 4 次课堂反馈稳定，表达主动性提升明显。
- 家长最关注孩子是否能主动表达，不只是课程次数。

### 建议

用“过去变化、当前目标、下一阶段安排”三段式生成成长回顾，再约一次低压力沟通。

## 2、服务动作

- 周五晚上前发送阶段成长回顾。
- 沟通前准备 2 条课堂观察和 1 条家庭配合建议。
- 沟通后再判断是否进入续费确认，不在第一次回顾里强推。

## 3、风险

如果只提醒剩余课次，家长容易把对话理解为销售催促。需要把服务价值先讲完整。`,
    stage: '待跟进',
    stageKey: 'expandable',
    range: 'week',
    updated: '昨天 18:10',
    updatedAt: '2026-08-26T18:10:00+08:00',
    actionCount: 7,
    customerCount: 3,
    chips: ['续费服务', '成长回顾'],
  },
  {
    id: 'review-month-1',
    title: '客户上下文更完整，老客户触达可以更自然',
    content: `# 服务复盘报告

本月服务客户 42 位，售前到在园的衔接较顺。老客户转介绍触达偏少，适合用活动资料做轻量触达。

## 1、售前客户

### 观察

- 新增试听客户 9 位，主要顾虑集中在价格和适应期。
- 价格顾虑客户更需要看到阶段效果，适应焦虑客户更需要看到入园观察。

### 建议

把高意向客户分成两类服务：价格顾虑先发成果案例，适应焦虑先发入园观察资料。

## 2、在园客户

### 观察

- 售后问题多数来自餐食、午睡和生活反馈。
- 能持续记录客户记忆的客户，AI 生成建议更贴近真实场景。

### 建议

减少手动字段维护，把关键跟进沉淀成事件即可。每次服务动作只记录发生了什么、家长反馈和下一步。

## 3、老客户触达

### 观察

老客户转介绍触达偏少，且话术容易显得直接。

### 建议

补齐亲子活动和公开课资料，用自然邀请替代直接索要转介绍。`,
    stage: '已生成',
    stageKey: 'formed',
    range: 'month',
    updated: '08月27日生成',
    updatedAt: '2026-08-27T09:00:00+08:00',
    actionCount: 63,
    customerCount: 42,
    chips: ['客户上下文', '转介绍', '公共知识库'],
  },
  {
    id: 'review-month-2',
    title: '服务知识沉淀不足，高频回应仍依赖个人编辑',
    content: `# 服务复盘报告

本月反复出现的沟通问题已经比较清晰，但还没有完全沉淀成可复用资料，导致一线服务时仍需要重复编辑。

## 1、高频问题

- 试听后如何解释价格。
- 入园适应期如何回应家长焦虑。
- 餐食和午睡反馈如何形成闭环。

## 2、知识边界

客户空间只记录这个客户发生了什么。通用课程资料、价格解释、异议处理和活动物料应沉淀到公共知识库，不复制进每个客户。

## 3、下月建议

- 每周固定整理 3 条有效回应。
- 把售后闭环流程做成公共知识库资料。
- 保留个人客户记忆，减少为了系统完整性维护大量字段。`,
    stage: '已生成',
    stageKey: 'formed',
    range: 'month',
    updated: '08月24日生成',
    updatedAt: '2026-08-24T17:20:00+08:00',
    actionCount: 29,
    customerCount: 18,
    chips: ['知识沉淀', '服务边界'],
  },
]

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

const reviewReports = serviceReviewReportSources.map(enrichServiceReviewReport)

const pageMeta = computed(() => {
  if (activeView.value === 'review') {
    return { title: '复盘', description: '以报告形式查看自己的服务回顾和下阶段建议' }
  }
  return { title: '消息', description: '左侧选择客户，右侧查看客户信息和下一步建议' }
})

const normalize = (value: string) => value.trim().toLowerCase()
const isTaskClosed = (id: string) => closedTaskIds.value.includes(id)

const toggleTaskClosed = (id: string) => {
  closedTaskIds.value = isTaskClosed(id)
    ? closedTaskIds.value.filter((item) => item !== id)
    : [...closedTaskIds.value, id]
}

const activeTask = computed<ServiceTask>(() => serviceTasks.find((task) => task.id === activeTaskId.value) ?? serviceTasks[0]!)

const selectTask = (id: string) => {
  activeTaskId.value = id
  composerText.value = activeTask.value.replyDraft
}

const taskMatchesKeyword = (task: ServiceTask, q: string) => {
  if (!q) return true
  return [task.customerName, task.studentName, task.title, task.summary, task.stage, task.nextAction, task.requiredServiceAction]
    .some((text) => normalize(text).includes(q))
}

const filteredTasks = computed(() => {
  const q = normalize(keyword.value)
  return serviceTasks.filter((task) => {
    if (taskFilter.value === 'urgent' && task.priorityKey !== 'high') return false
    return taskMatchesKeyword(task, q)
  })
})

const filteredReviewReports = computed(() => reviewReports.filter((report) => report.range === reviewRange.value))
const reviewTotalActions = computed(() => filteredReviewReports.value.reduce((total, report) => total + report.actionCount, 0))
const reviewReportGroups = computed(() => [{
  key: reviewRange.value,
  label: reviewRange.value === 'week' ? '本周' : '本月',
  reports: filteredReviewReports.value,
}])

const setReviewRange = (range: ReviewRange) => {
  reviewRange.value = range
}

const openReviewPreview = (report: ServiceReviewReport) => {
  activeReviewReport.value = report
  reviewPreviewVisible.value = true
}

const closeReviewPreview = () => {
  reviewPreviewVisible.value = false
}

composerText.value = activeTask.value.replyDraft
</script>

<style scoped lang="less">
.service-page {
  height: 100%;
  flex: 1;
  display: flex;
  min-height: 0;
  overflow: hidden;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
}

.service-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
  padding: 20px 0 0 28px;
  box-sizing: border-box;
}

.service-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
  padding-right: 28px;
}

.service-heading {
  min-width: 0;

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 21px;
    font-weight: 500;
    line-height: 30px;
    letter-spacing: 0;
  }

  p {
    margin: 2px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 20px;
  }
}

.service-search {
  width: 280px;
  flex-shrink: 0;
}

.service-scroll {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0 28px 16px 0;
}

.service-workspace {
  display: grid;
  grid-template-columns: minmax(280px, 340px) minmax(0, 760px);
  gap: 16px;
  align-items: start;
}

.review-page {
  max-width: 1080px;
}

.simple-panel,
.focus-panel {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-sizing: border-box;
}

.simple-panel {
  padding: 12px;
}

.list-panel {
  overflow: hidden;
  padding: 0;
}

.focus-panel {
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 18px;
}

.review-panel {
  max-width: 860px;
}

.panel-head,
.focus-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.panel-head {
  margin-bottom: 0;
  padding: 9px 10px 8px;

  h3 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
  }

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }
}

.focus-head {
  padding-bottom: 14px;
  border-bottom: 1px solid var(--td-component-stroke);

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  h3 {
    margin: 2px 0 0;
    color: var(--td-text-color-primary);
    font-size: 18px;
    font-weight: 600;
    line-height: 26px;
    letter-spacing: 0;
  }
}

.simple-list {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.customer-row {
  display: grid;
  grid-template-columns: 42px minmax(0, 1fr) 40px;
  gap: 10px;
  align-items: center;
  width: 100%;
  min-height: 62px;
  padding: 8px 10px;
  border: 0;
  border-radius: 0;
  background: transparent;
  color: var(--td-text-color-primary);
  cursor: pointer;
  text-align: left;
  transition: background-color 0.16s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.active {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.muted {
    opacity: 0.72;
  }
}

.customer-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 42px;
  height: 42px;
  overflow: hidden;
  border-radius: 50%;
  background:
    radial-gradient(circle at 34% 30%, rgba(255, 255, 255, 0.78), transparent 24%),
    linear-gradient(135deg, #6ee7b7 0%, #07c05f 46%, #16834a 100%);
  color: #fff;
  font-size: 15px;
  font-weight: 600;
  line-height: 1;
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, 0.34);
}

.priority-high .customer-avatar {
  background:
    radial-gradient(circle at 34% 30%, rgba(255, 255, 255, 0.78), transparent 24%),
    linear-gradient(135deg, #ff8a70 0%, #ff4d2e 48%, #c43a28 100%);
}

.priority-medium .customer-avatar {
  background:
    radial-gradient(circle at 34% 30%, rgba(255, 255, 255, 0.78), transparent 24%),
    linear-gradient(135deg, #f8d56b 0%, #ed7b2f 48%, #b85f1f 100%);
}

.customer-copy {
  display: flex;
  flex-direction: column;
  min-width: 0;
  gap: 4px;

  em {
    overflow: hidden;
    color: var(--td-text-color-secondary);
    font-style: normal;
    font-size: 12px;
    line-height: 18px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  strong {
    overflow: hidden;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 20px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  em {
    font-size: 12px;
    line-height: 17px;
  }
}

.customer-title-line {
  display: flex;
  align-items: center;
  min-width: 0;
}

.customer-meta {
  display: flex;
  align-items: flex-end;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.customer-time {
  color: var(--td-text-color-placeholder);
  font-size: 11px;
  line-height: 16px;
  white-space: nowrap;
}

.customer-unread {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 19px;
  height: 19px;
  padding: 0 6px;
  border-radius: 999px;
  background: #ff4d2e;
  color: #fff;
  font-size: 11px;
  font-weight: 600;
  line-height: 19px;
  box-sizing: border-box;
}

.next-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 16px;
  padding: 14px;
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);

  span {
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }
}

.plain-section {
  margin-top: 18px;

  h3 {
    margin: 0 0 8px;
    color: var(--td-text-color-primary);
    font-size: 14px;
    font-weight: 600;
    line-height: 22px;
    letter-spacing: 0;
  }

  p {
    margin: 0;
    color: var(--td-text-color-secondary);
    font-size: 14px;
    line-height: 22px;
  }
}

.simple-bullets {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;

  li {
    color: var(--td-text-color-secondary);
    font-size: 13px;
    line-height: 21px;
  }

  strong {
    margin-right: 4px;
    color: var(--td-text-color-primary);
  }
}

.reply-box {
  width: 100%;
}

.action-row {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 10px;
}

.simple-events {
  display: flex;
  flex-direction: column;
  gap: 10px;

  article {
    display: grid;
    grid-template-columns: 56px minmax(0, 1fr);
    gap: 12px;
  }

  time {
    color: var(--td-text-color-placeholder);
    font-size: 12px;
    line-height: 20px;
  }

  strong {
    color: var(--td-text-color-primary);
    font-size: 13px;
    line-height: 20px;
  }

  p {
    margin: 2px 0 0;
    color: var(--td-text-color-secondary);
    font-size: 12px;
    line-height: 18px;
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
  .service-workspace {
    grid-template-columns: 1fr;
  }

  .review-page,
  .review-panel {
    max-width: none;
  }
}

@media (max-width: 760px) {
  .service-main {
    padding: 16px 0 0 16px;
  }

  .service-header {
    align-items: stretch;
    flex-direction: column;
    padding-right: 16px;
  }

  .service-search {
    width: 100%;
  }

  .service-scroll {
    padding-right: 16px;
  }

  .panel-head,
  .focus-head {
    align-items: flex-start;
    flex-direction: column;
  }

  .action-row {
    justify-content: flex-start;
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
