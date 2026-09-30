<template>
  <div class="organize-product-page">
    <main v-if="course" class="organize-course-detail-scroll">
      <header class="organize-course-detail-head">
        <button type="button" class="organize-back-button" @click="close">
          <t-icon name="chevron-left" />
          返回发现
        </button>
        <div class="organize-output-detail-actions">
          <t-button variant="outline" size="small" @click="close">关闭</t-button>
        </div>
      </header>

      <article class="organize-course-hero">
        <div class="course-hero-cover" :style="coverStyle">
          <img v-if="course.cover_url" :src="course.cover_url" :alt="course.title" />
          <template v-else>
            <t-icon name="book-open" />
            <span class="course-hero-cover__kind">系列课程</span>
          </template>
        </div>
        <div class="course-hero-main">
          <span class="organize-output-eyebrow">{{ sourceLabel(course.source) }}</span>
          <h2>{{ course.title }}</h2>
          <div class="course-hero-meta">
            <span>{{ course.lesson_count }} 讲</span>
            <span v-if="course.teacher_name">
              讲师：{{ course.teacher_name }}{{ course.teacher_title ? ` · ${course.teacher_title}` : '' }}
            </span>
            <span v-if="categoryLabel(course.category)">{{ categoryLabel(course.category) }}</span>
            <span v-if="formatDate(course.updated_at)">更新于 {{ formatDate(course.updated_at) }}</span>
          </div>
          <p class="course-hero-summary">{{ course.summary || '暂无课程简介' }}</p>
          <div class="course-hero-actions">
            <t-button theme="primary" :disabled="!firstOpenLesson" @click="openLesson(firstOpenLesson)">
              <template #icon><t-icon name="play-circle" /></template>
              开始学习
            </t-button>
            <span class="course-hero-hint">{{ lessons.length }} 讲已上架</span>
          </div>
        </div>
      </article>

      <section class="organize-course-outline-block">
        <div class="course-outline-head">
          <b class="course-outline-title">课程大纲</b>
          <span class="course-outline-hint">{{ lessons.length }} 讲 · 点击任一一讲进入学习</span>
        </div>
        <div class="course-outline">
          <button
            v-for="(lesson, index) in lessons"
            :key="lesson.id"
            type="button"
            class="course-outline-row"
            :disabled="!lesson.available"
            @click="openLesson(lesson)"
          >
            <span class="course-outline-order">{{ index + 1 }}</span>
            <t-icon :name="kindIcon(lesson.lesson_type)" class="course-outline-icon" />
            <span class="course-outline-name" :title="lesson.title">{{ lesson.title }}</span>
            <span class="course-outline-kind">{{ kindLabel(lesson.lesson_type) }}</span>
            <span class="course-outline-state">{{ lesson.available ? '查看' : '未开放' }}</span>
          </button>
          <div v-if="!lessons.length" class="course-outline-empty">这门课还没有讲次</div>
        </div>
      </section>
    </main>

    <main v-else-if="loading" class="organize-course-detail-scroll">
      <div class="organize-detail-empty">
        <t-loading size="small" />
        <strong>正在加载课程</strong>
      </div>
    </main>

    <main v-else class="organize-course-detail-scroll">
      <div class="organize-detail-empty">
        <t-icon name="error-circle" />
        <strong>课程不存在或已下架</strong>
        <t-button variant="outline" size="small" @click="close">返回发现</t-button>
      </div>
    </main>

  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRoute, useRouter } from 'vue-router'
import { getOrganizeCourse, type OrganizeCourse, type OrganizeCourseLesson } from '@/api/organize'
import { DISCOVER_CATEGORIES, discoverCategoryLabel } from './discoverCategories'
import { ORGANIZE_ROUTE_NAMES } from './organizeRoutes'

const route = useRoute()
const router = useRouter()

const course = ref<OrganizeCourse | null>(null)
const loading = ref(true)

const lessons = computed<OrganizeCourseLesson[]>(() => course.value?.lessons || [])

// "开始学习" lands on the first lesson that is actually open, not blindly on
// index 0: an archived first lesson would dead-end the button.
const firstOpenLesson = computed<OrganizeCourseLesson | null>(
  () => lessons.value.find((lesson) => lesson.available) || null,
)

function sourceLabel(source?: string) {
  return source === 'creator' ? '创作者课程' : '官方精品课'
}

function categoryLabel(category?: string) {
  return discoverCategoryLabel(category) || ''
}

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

function formatDate(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

// Same treatment as the discover grid: platform courses ship a cover, the rest
// get the brand wash so a course page and its card look like one surface.
const coverStyle = computed(() => {
  if (course.value?.cover_url) return { backgroundImage: `url(${course.value.cover_url})` }
  return { background: 'linear-gradient(135deg, var(--td-brand-color-light) 0%, var(--td-bg-color-secondarycontainer) 100%)' }
})

async function loadCourse() {
  loading.value = true
  try {
    const response = await getOrganizeCourse(String(route.params.courseId || ''))
    if (!response.success || !response.data) {
      throw new Error(response.message || '课程详情加载失败')
    }
    course.value = response.data
  } catch (error: any) {
    course.value = null
    MessagePlugin.error(error?.message || '课程详情加载失败')
  } finally {
    loading.value = false
  }
}

// A lesson opens its own page rather than a drawer: learning a course means
// moving through it, and a drawer cannot carry 上一讲 / 下一讲 or keep the
// outline beside the body the way the lesson shell does.
function openLesson(lesson: OrganizeCourseLesson | null) {
  if (!lesson?.available) return
  void router.push({
    name: ORGANIZE_ROUTE_NAMES.lessonDetail,
    params: { courseId: String(route.params.courseId || ''), lessonId: lesson.id },
  })
}

// Returning to discover carries the tab along, otherwise the workspace would
// remount on 推荐 and the user would lose the course list they came from.
function close() {
  void router.push({ path: '/platform/organize/discover', query: { tab: 'recommended' } })
}

onMounted(() => {
  void loadCourse()
})

watch(
  () => route.params.courseId,
  () => void loadCourse(),
)
</script>

<style scoped lang="less">
.organize-product-page {
  display: flex;
  height: 100%;
  min-height: 0;
  flex-direction: column;
  background: var(--td-bg-color-container);
}

.organize-course-detail-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 42px 48px;
}

.organize-course-detail-head,
.organize-course-hero,
.organize-course-outline-block {
  max-width: 860px;
  margin-right: auto;
  margin-left: auto;
}

.organize-course-detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  margin-bottom: 18px;
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
  font-size: 13px;
}

.organize-back-button:hover {
  color: var(--td-brand-color);
}

.organize-output-detail-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.organize-course-hero {
  display: grid;
  grid-template-columns: 148px minmax(0, 1fr);
  gap: 18px;
  padding: 18px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.course-hero-cover {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 148px;
  height: 148px;
  padding: 14px;
  box-sizing: border-box;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background-color: var(--td-bg-color-secondarycontainer);
  background-size: cover;
  background-position: center;
  color: var(--td-text-color-primary);
  font-size: 40px;
}

.course-hero-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
}

.course-hero-cover__kind {
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.course-hero-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.course-hero-main h2 {
  margin: 0;
  color: var(--td-text-color-primary);
  font-size: 20px;
  font-weight: 600;
  line-height: 1.4;
}

.organize-output-eyebrow {
  align-self: flex-start;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 11px;
  line-height: 18px;
}

.course-hero-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.course-hero-summary {
  margin: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.organize-course-outline-block {
  margin-top: 18px;
}

.course-hero-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 14px;
}

.course-hero-hint {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.course-outline-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 8px;
}

.course-outline-title {
  margin-bottom: 0;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-weight: 600;
}

.course-outline-hint {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.course-outline {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.course-outline-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.course-outline-row:hover:not(:disabled) {
  border-color: var(--td-brand-color);
  background: var(--td-bg-color-container-hover);
}

.course-outline-row:disabled {
  cursor: not-allowed;
  color: var(--td-text-color-disabled);
}

.course-outline-order {
  flex: 0 0 26px;
  text-align: right;
  color: var(--td-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  font-size: 13px;
}

.course-outline-icon {
  flex: 0 0 auto;
  color: var(--td-brand-color);
}

.course-outline-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
}

.course-outline-kind,
.course-outline-state {
  flex: 0 0 auto;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.course-outline-empty {
  padding: 24px 0;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
  text-align: center;
}

.organize-detail-empty {
  display: flex;
  min-height: 320px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--td-text-color-placeholder);
}

@media (max-width: 720px) {
  .organize-course-detail-scroll {
    padding: 18px 16px 32px;
  }

  .organize-course-hero {
    grid-template-columns: minmax(0, 1fr);
  }

  .course-hero-cover {
    width: 100%;
    height: 160px;
  }
}
</style>
