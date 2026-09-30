<template>
  <div class="organize-product-page">
    <main v-if="course && currentLesson && currentLesson.available" class="organize-lesson-scroll">
      <header class="organize-lesson-head">
        <button type="button" class="organize-back-button" @click="backToDiscover">
          <t-icon name="chevron-left" />
          返回发现
        </button>
        <t-button variant="outline" size="small" @click="backToCourse">课程详情</t-button>
        <span class="organize-lesson-crumb">
          发现 / 推荐 / {{ course.title }} / 第 {{ currentIndex + 1 }} 讲
        </span>
      </header>

      <div class="lesson-shell">
        <article class="lesson-main">
          <header class="lesson-main-head">
            <div class="lesson-course-title">{{ course.title }}</div>
            <h2>{{ currentLesson.title }}</h2>
            <div class="lesson-tags">
              <span class="lesson-tag">
                <t-icon :name="kindIcon(currentLesson.lesson_type)" />
                {{ kindLabel(currentLesson.lesson_type) }}
              </span>
              <span v-if="durationLabel" class="lesson-tag">{{ durationLabel }}</span>
              <span class="lesson-tag lesson-tag--progress">{{ currentIndex + 1 }} / {{ lessons.length }}</span>
            </div>
          </header>

          <div v-if="mediaUrl" class="lesson-player">
            <video v-if="isVideo" class="lesson-player-media" controls :src="mediaUrl" preload="metadata" />
            <audio v-else class="lesson-player-audio" controls :src="mediaUrl" preload="metadata" />
          </div>

          <div v-if="lessonBodyLoading" class="lesson-body-empty">
            <t-loading size="small" text="加载正文" />
          </div>
          <div v-else-if="lessonHtml" class="lesson-body markdown-content" v-html="lessonHtml" />
          <div v-else class="lesson-body-empty">本讲暂无正文</div>

          <div v-if="sourceFileName" class="lesson-source">
            <span>本讲来自 admin 上传文件夹中的：</span>
            <span class="lesson-source-tag">
              <t-icon name="book-open" />
              {{ sourceFileName }}
            </span>
          </div>

          <footer class="lesson-nav">
            <t-button variant="outline" :disabled="!prevLesson" @click="goTo(prevLesson)">
              <template #icon><t-icon name="chevron-left" /></template>
              上一讲
            </t-button>
            <span class="lesson-nav-mid">{{ currentIndex + 1 }} / {{ lessons.length }}</span>
            <t-button theme="primary" :disabled="!nextLesson" @click="goTo(nextLesson)">
              下一讲
              <template #icon><t-icon name="chevron-right" /></template>
            </t-button>
          </footer>
        </article>

        <aside class="lesson-aside">
          <div class="lesson-aside-head">
            <b>课程大纲</b>
            <span>{{ lessons.length }} 讲</span>
          </div>
          <div class="lesson-aside-list">
            <button
              v-for="(lesson, index) in lessons"
              :key="lesson.id"
              type="button"
              class="lesson-aside-row"
              :class="{ 'lesson-aside-row--current': lesson.id === lessonId }"
              :disabled="!lesson.available"
              @click="goTo(lesson)"
            >
              <span class="lesson-aside-order">{{ index + 1 }}</span>
              <span class="lesson-aside-name" :title="lesson.title">{{ lesson.title }}</span>
              <span class="lesson-aside-kind">{{ lesson.available ? kindLabel(lesson.lesson_type) : '未开放' }}</span>
            </button>
          </div>
        </aside>
      </div>
    </main>

    <main v-else-if="loading" class="organize-lesson-scroll">
      <div class="organize-detail-empty">
        <t-loading size="small" />
        <strong>正在加载讲次</strong>
      </div>
    </main>

    <main v-else class="organize-lesson-scroll">
      <div class="organize-detail-empty">
        <t-icon name="error-circle" />
        <strong>{{ course ? '讲次不存在或未开放' : '课程不存在或已下架' }}</strong>
        <t-button variant="outline" size="small" @click="backToDiscover">返回发现</t-button>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { getOrganizeCourse, getOrganizeCourseLessonContent, type OrganizeCourse, type OrganizeCourseLesson } from '@/api/organize'
import { renderSproutReportHtml } from './sproutReport'

const route = useRoute()
const router = useRouter()

const course = ref<OrganizeCourse | null>(null)
const loading = ref(true)

// The outline deliberately ships no bodies — a 14-lesson course used to move
// ~16 MB per load — so each chapter arrives in its own request.
const lessonContent = ref('')
const lessonBodyLoading = ref(false)

const courseId = computed(() => String(route.params.courseId || ''))
const lessonId = computed(() => String(route.params.lessonId || ''))

const lessons = computed<OrganizeCourseLesson[]>(() => course.value?.lessons || [])
const currentIndex = computed(() => lessons.value.findIndex((item) => item.id === lessonId.value))
const currentLesson = computed<OrganizeCourseLesson | null>(() =>
  currentIndex.value >= 0 ? lessons.value[currentIndex.value] : null,
)
const prevLesson = computed<OrganizeCourseLesson | null>(() =>
  currentIndex.value > 0 ? lessons.value[currentIndex.value - 1] : null,
)
const nextLesson = computed<OrganizeCourseLesson | null>(() =>
  currentIndex.value >= 0 && currentIndex.value < lessons.value.length - 1
    ? lessons.value[currentIndex.value + 1]
    : null,
)

const lessonHtml = computed(() => renderSproutReportHtml(lessonContent.value))

const isVideo = computed(() => currentLesson.value?.lesson_type === 'video')

// duration_seconds exists on the lesson but the upload pipeline does not measure
// it yet, so the label only appears once there is a real value to show rather
// than rendering "0 分钟" everywhere.
const durationLabel = computed(() => {
  const seconds = currentLesson.value?.duration_seconds || 0
  if (seconds <= 0) return ''
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟`
  return `${Math.floor(minutes / 60)} 小时 ${minutes % 60} 分钟`
})

const mediaUrl = computed(() => currentLesson.value?.media_url || '')

const sourceFileName = computed(() => currentLesson.value?.source_file_name || '')

function kindIcon(kind?: string) {
  if (kind === 'video') return 'play-circle'
  if (kind === 'audio') return 'sound'
  return 'file-word'
}

function kindLabel(kind?: string) {
  if (kind === 'video') return '视频'
  if (kind === 'audio') return '音频'
  return '图文'
}

async function loadLessonBody() {
  const lesson = currentLesson.value
  if (!lesson) {
    lessonContent.value = ''
    return
  }
  lessonBodyLoading.value = true
  try {
    const response = await getOrganizeCourseLessonContent(courseId.value, lesson.id)
    lessonContent.value = response.success ? response.data?.content || '' : ''
    if (!response.success) {
      throw new Error(response.message || '正文加载失败')
    }
  } catch (error: any) {
    lessonContent.value = ''
    MessagePlugin.error(error?.message || '正文加载失败')
  } finally {
    lessonBodyLoading.value = false
  }
}

async function loadCourse() {
  loading.value = true
  try {
    const response = await getOrganizeCourse(courseId.value)
    if (!response.success || !response.data) {
      throw new Error(response.message || '课程加载失败')
    }
    course.value = response.data
    await loadLessonBody()
  } catch (error: any) {
    course.value = null
    MessagePlugin.error(error?.message || '课程加载失败')
  } finally {
    loading.value = false
  }
}

// Moving between lessons swaps the URL in place: the course stays loaded, so
// the outline keeps its scroll position and only the body changes.
function goTo(lesson: OrganizeCourseLesson | null) {
  if (!lesson || !lesson.available) return
  void router.replace({
    name: 'organizeLessonDetail',
    params: { courseId: courseId.value, lessonId: lesson.id },
  })
}

function backToCourse() {
  void router.push({ name: 'organizeCourseDetail', params: { courseId: courseId.value } })
}

// Return to the recommendation stream because courses are now part of 推荐.
function backToDiscover() {
  void router.push({ path: '/platform/organize/discover', query: { tab: 'recommended' } })
}

watch(lessonId, () => {
  const scroll = document.querySelector('.organize-lesson-scroll')
  if (scroll) scroll.scrollTop = 0
  // Switching chapters only swaps the route; refetch just the body.
  void loadLessonBody()
})

onMounted(() => {
  void loadCourse()
})
</script>

<style scoped lang="less">
@import '../../components/css/chat-markdown.less';

.organize-lesson-scroll {
  flex: 1;
  min-height: 0;
  padding: 20px 24px 32px;
  overflow-y: auto;
}

.organize-lesson-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.organize-lesson-crumb {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lesson-shell {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 288px;
  gap: 16px;
  align-items: start;
}

.lesson-main {
  min-width: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.lesson-main-head {
  padding: 16px 20px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.lesson-course-title {
  margin-bottom: 5px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.lesson-main-head h2 {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 18px;
  line-height: 1.4;
}

.lesson-tags {
  display: flex;
  gap: 8px;
  margin-top: 9px;
  align-items: center;
}

.lesson-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.lesson-tag--progress {
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
}

.lesson-player {
  padding: 16px 20px 0;
}

.lesson-player-media {
  width: 100%;
  max-height: 420px;
  border-radius: 8px;
  background: #000;
}

.lesson-player-audio {
  width: 100%;
}

.lesson-body {
  padding: 18px 20px 4px;
}

.lesson-body-empty {
  padding: 28px 20px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  text-align: center;
}

.lesson-source {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin: 18px 20px 0;
  padding-top: 12px;
  border-top: 1px dashed var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.lesson-source-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 12px;
}

.lesson-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-top: 20px;
  padding: 14px 20px;
  border-top: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
}

.lesson-nav-mid {
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.lesson-aside {
  position: sticky;
  top: 20px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  overflow: hidden;
}

.lesson-aside-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 13px 15px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.lesson-aside-head b {
  color: var(--td-text-color-primary);
  font-size: 13px;
}

.lesson-aside-head span {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.lesson-aside-list {
  max-height: 520px;
  overflow-y: auto;
}

.lesson-aside-row {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 12px;
  border: none;
  border-bottom: 1px solid var(--td-component-stroke);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s ease;
}

.lesson-aside-row:last-child {
  border-bottom: none;
}

.lesson-aside-row:hover:not(:disabled) {
  background: var(--td-bg-color-container-hover);
}

.lesson-aside-row:disabled {
  cursor: not-allowed;
  color: var(--td-text-color-disabled);
}

.lesson-aside-row--current {
  background: var(--td-brand-color-light);
}

.lesson-aside-order {
  flex: 0 0 18px;
  height: 18px;
  border: 1px solid var(--td-component-border);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-placeholder);
  font-size: 10px;
}

.lesson-aside-row--current .lesson-aside-order {
  border-color: var(--td-brand-color);
  background: var(--td-brand-color);
  color: #fff;
}

.lesson-aside-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-size: 13px;
}

.lesson-aside-row--current .lesson-aside-name {
  color: var(--td-brand-color);
  font-weight: 600;
}

.lesson-aside-kind {
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

@media (max-width: 960px) {
  .lesson-shell {
    grid-template-columns: minmax(0, 1fr);
  }

  .lesson-aside {
    position: static;
  }
}
</style>
