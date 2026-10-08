<template>
  <div class="organize-product-page">
    <main v-if="course && currentLesson && currentLesson.available" class="organize-lesson-scroll">
      <header class="organize-lesson-head">
        <button type="button" class="organize-back-button" @click="backToDiscover">
          <t-icon name="chevron-left" />
          返回发现
        </button>
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

          <div v-if="mediaLoading" class="lesson-player lesson-player--loading">
            <t-loading size="small" text="加载媒体" />
          </div>
          <div v-else-if="mediaPlayerUrl" class="lesson-player">
            <video
              v-if="isVideo"
              class="lesson-player-media"
              controls
              :src="mediaPlayerUrl"
              preload="metadata"
              ref="mediaElement"
              @error="handleMediaPlaybackError"
            />
            <audio
              v-else
              class="lesson-player-audio"
              controls
              :src="mediaPlayerUrl"
              preload="metadata"
              ref="mediaElement"
              @error="handleMediaPlaybackError"
            />
          </div>
          <div v-else-if="mediaError" class="lesson-body-empty">{{ mediaError }}</div>

          <div v-if="lecturerLabel" class="lesson-source">
            <span>讲师：</span>
            <span class="lesson-source-tag">
              <t-icon name="user" />
              {{ lecturerLabel }}
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
        <strong>{{ courseLoadError || (course ? '讲次不存在或未开放' : '课程不存在或已下架') }}</strong>
        <t-button variant="outline" size="small" @click="backToDiscover">返回发现</t-button>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  getOrganizeCourse,
  getOrganizeCourseLessonMediaURL,
  type OrganizeCourse,
  type OrganizeCourseLesson,
} from '@/api/organize'

const route = useRoute()
const router = useRouter()

const course = ref<OrganizeCourse | null>(null)
const loading = ref(true)
const courseLoadError = ref('')

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

const mediaPlayerUrl = ref('')
const mediaLoading = ref(false)
const mediaError = ref('')
const mediaElement = ref<HTMLMediaElement | null>(null)
let mediaRequestSeq = 0
let mediaDirectRetryAttempted = false
let courseAbortController: AbortController | null = null
let mediaAbortController: AbortController | null = null

const lecturerLabel = computed(() =>
  [course.value?.teacher_name, course.value?.teacher_title].filter((value) => value?.trim()).join(' · '),
)

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

function cancelNativeMediaRequest() {
  const element = mediaElement.value
  if (!element) return
  element.pause()
  element.removeAttribute('src')
  element.load()
}

function isDirectMediaUrl(url: string) {
  try {
    const parsed = new URL(url)
    if (parsed.protocol !== 'http:' && parsed.protocol !== 'https:') return false
    const normalizedPath = `/${parsed.pathname.replace(/^\/+/, '')}`
    return normalizedPath !== '/files'
      && normalizedPath !== '/api/v1/files/presigned'
      && !normalizedPath.startsWith('/r/')
      && !(
        normalizedPath.startsWith('/api/v1/organize/courses/')
        && normalizedPath.endsWith('/media')
      )
  } catch {
    return false
  }
}

function isRequestCanceled(error: any) {
  return error?.code === 'ERR_CANCELED' || error?.name === 'CanceledError' || error?.name === 'AbortError'
}

async function loadLessonMedia(isDirectRetry = false) {
  const requestSeq = ++mediaRequestSeq
  mediaAbortController?.abort()
  const controller = new AbortController()
  mediaAbortController = controller
  cancelNativeMediaRequest()
  mediaPlayerUrl.value = ''
  mediaError.value = ''
  if (!isDirectRetry) {
    mediaDirectRetryAttempted = false
  }
  if (
    !currentLesson.value?.available
    || !['video', 'audio'].includes(currentLesson.value.lesson_type)
  ) {
    mediaLoading.value = false
    mediaAbortController = null
    return
  }

  mediaLoading.value = true
  try {
    const response = await getOrganizeCourseLessonMediaURL(courseId.value, lessonId.value, {
      timeout: 0,
      signal: controller.signal,
    })
    if (requestSeq !== mediaRequestSeq) return
    const directUrl = response.success ? response.data?.url || '' : ''
    if (!isDirectMediaUrl(directUrl)) {
      throw new Error('对象存储未返回可用的媒体地址')
    }
    mediaPlayerUrl.value = directUrl
  } catch (error) {
    if (requestSeq !== mediaRequestSeq || isRequestCanceled(error)) return
    mediaError.value = '媒体直连地址获取失败，请稍后重试'
  } finally {
    if (requestSeq === mediaRequestSeq) {
      mediaLoading.value = false
    }
    if (mediaAbortController === controller) {
      mediaAbortController = null
    }
  }
}

function handleMediaPlaybackError() {
  if (!mediaPlayerUrl.value) return
  if (!mediaDirectRetryAttempted) {
    mediaDirectRetryAttempted = true
    void loadLessonMedia(true)
    return
  }
  cancelNativeMediaRequest()
  mediaPlayerUrl.value = ''
  mediaError.value = '浏览器无法读取对象存储媒体文件，请检查 OSS 域名和跨域配置'
}

async function loadCourse() {
  courseAbortController?.abort()
  const controller = new AbortController()
  courseAbortController = controller
  courseLoadError.value = ''
  loading.value = true
  try {
    const response = await getOrganizeCourse(courseId.value, {
      timeout: 0,
      signal: controller.signal,
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '课程加载失败')
    }
    course.value = response.data
    void loadLessonMedia()
  } catch (error: any) {
    if (isRequestCanceled(error)) return
    course.value = null
    courseLoadError.value = '课程加载失败，请稍后重试'
  } finally {
    if (courseAbortController === controller) {
      courseAbortController = null
      loading.value = false
    }
  }
}

// Moving between lessons swaps the URL in place: the course stays loaded, so
// the outline keeps its scroll position and only the media source changes.
function goTo(lesson: OrganizeCourseLesson | null) {
  if (!lesson || !lesson.available) return
  void router.replace({
    name: 'organizeLessonDetail',
    params: { courseId: courseId.value, lessonId: lesson.id },
  })
}

// Return to the recommendation stream because courses are now part of 推荐.
function backToDiscover() {
  void router.push({ path: '/platform/organize/discover', query: { tab: 'recommended' } })
}

watch(lessonId, () => {
  const scroll = document.querySelector('.organize-lesson-scroll')
  if (scroll) scroll.scrollTop = 0
  // Switching chapters only swaps the route and media source.
  void loadLessonMedia()
})

onMounted(() => {
  void loadCourse()
})

onBeforeUnmount(() => {
  mediaRequestSeq += 1
  cancelNativeMediaRequest()
  courseAbortController?.abort()
  mediaAbortController?.abort()
  courseAbortController = null
  mediaAbortController = null
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
  margin-bottom: 14px;
}

.organize-back-button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  font: inherit;
  font-size: 13px;
}

.organize-back-button:hover {
  color: var(--td-brand-color);
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
  margin: 16px 20px 0;
  padding: 10px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: #111318;
}

.lesson-player--loading {
  min-height: 120px;
  margin: 16px 20px 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--td-bg-color-secondarycontainer);
}

.lesson-player-media {
  display: block;
  width: 100%;
  aspect-ratio: 16 / 9;
  max-height: min(70vh, 560px);
  border-radius: 8px;
  background: #000;
  object-fit: contain;
}

.lesson-player-audio {
  display: block;
  width: 100%;
  height: 44px;
  accent-color: var(--td-brand-color);
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
