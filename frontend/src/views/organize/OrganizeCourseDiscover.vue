<template>
  <div v-if="loading" class="course-discover course-discover--loading">
    <t-loading size="medium" text="加载课程中" />
  </div>
  <div v-else-if="courses.length" class="course-discover">
    <div class="course-grid">
      <article
        v-for="item in courses"
        :key="item.id"
        class="course-card"
        role="button"
        tabindex="0"
        @click="openCourse(item)"
        @keydown.enter.self="openCourse(item)"
      >
        <div class="course-card-cover" :style="coverStyle(item)">
          <img
            v-if="hasCover(item)"
            :src="coverSource(item)"
            :alt="item.title"
            @error="handleCoverError(item)"
          />
          <template v-else>
            <t-icon name="book-open" />
            <span class="course-card-cover__kind">系列课程</span>
          </template>
        </div>
        <div class="course-card-body">
          <div class="course-card-title">
            <h3>{{ item.title }}</h3>
            <span class="course-source-badge">{{ sourceLabel(item.source) }}</span>
          </div>
          <div class="course-card-meta">
            <span>{{ item.lesson_count }} 讲</span>
            <span v-if="item.teacher_name">{{ item.teacher_name }}{{ item.teacher_title ? ` · ${item.teacher_title}` : '' }}</span>
          </div>
          <p class="course-card-summary">{{ item.summary || '暂无课程简介' }}</p>
          <div class="course-card-footer">
            <span v-if="categoryLabel(item.category)" class="course-card-category">{{ categoryLabel(item.category) }}</span>
            <span class="course-card-date">更新于 {{ formatDate(item.updated_at) }}</span>
          </div>
        </div>
      </article>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import { getOrganizeCourseCover, getOrganizeCourses, type OrganizeCourse } from '@/api/organize'
import { discoverCategoryLabel } from './discoverCategories'

const router = useRouter()

const PAGE_SIZE = 6

const courses = ref<OrganizeCourse[]>([])
const loading = ref(true)
const failedCoverIds = ref(new Set<string>())
const coverSources = ref(new Map<string, string>())

function sourceLabel(source?: string) {
  return source === 'creator' ? '创作者课程' : '官方精品课'
}

function categoryLabel(category?: string) {
  return discoverCategoryLabel(category) || ''
}

function formatDate(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function hasCover(item: OrganizeCourse) {
  return Boolean(coverSource(item) && !failedCoverIds.value.has(item.id))
}

function handleCoverError(item: OrganizeCourse) {
  if (failedCoverIds.value.has(item.id)) return
  failedCoverIds.value = new Set(failedCoverIds.value).add(item.id)
}

function coverSource(item: OrganizeCourse) {
  return coverSources.value.get(item.id) || ''
}

function coverStyle(item: OrganizeCourse) {
  if (hasCover(item)) return {}
  return { background: 'var(--td-brand-color-light)' }
}

function revokeCoverSources() {
  coverSources.value.forEach((source) => URL.revokeObjectURL(source))
  coverSources.value = new Map()
}

function setCoverSource(id: string, source: string) {
  coverSources.value = new Map(coverSources.value).set(id, source)
}

async function loadCoverSource(item: OrganizeCourse) {
  if (!item.cover_url) return
  if (!item.cover_url.startsWith('/api/')) {
    setCoverSource(item.id, item.cover_url)
    return
  }
  try {
    const blob = await getOrganizeCourseCover(item.cover_url)
    setCoverSource(item.id, URL.createObjectURL(blob))
  } catch {
    handleCoverError(item)
  }
}

async function loadCourses() {
  loading.value = true
  try {
    const response = await getOrganizeCourses({
      page: 1,
      page_size: PAGE_SIZE,
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '课程加载失败')
    }
    revokeCoverSources()
    courses.value = response.data.items || []
    failedCoverIds.value = new Set()
    void Promise.all(courses.value.map((item) => loadCoverSource(item)))
  } catch (error: any) {
    courses.value = []
    MessagePlugin.error(error?.message || '课程加载失败')
  } finally {
    loading.value = false
  }
}

function openCourse(item: OrganizeCourse) {
  void router.push(`/platform/organize/courses/${encodeURIComponent(item.id)}`)
}

onMounted(() => {
  void loadCourses()
})

onBeforeUnmount(revokeCoverSources)

defineExpose({ reload: loadCourses })
</script>

<style scoped>
.course-discover { display: flex; flex-direction: column; gap: 16px; }
.course-discover--loading { min-height: 120px; align-items: center; justify-content: center; }

.course-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(420px, 1fr)); gap: 14px; }

.course-card {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  align-items: start;
  gap: 16px;
  min-height: 128px;
  padding: 14px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}
.course-card:hover {
  border-color: rgba(7, 192, 95, 0.55);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}
.course-card:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }

.course-card-cover {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 92px;
  height: 92px;
  padding: 8px;
  box-sizing: border-box;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background-color: var(--td-bg-color-secondarycontainer);
  background-size: cover;
  background-position: center;
  color: var(--td-brand-color);
  font-size: 25px;
}
.course-card-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 5px;
}
.course-card-cover__kind {
  color: var(--td-brand-color-7);
  font-size: 11px;
  line-height: 16px;
}

.course-card-body { min-width: 0; display: flex; flex-direction: column; gap: 7px; }
.course-card-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.course-card-title h3 {
  margin: 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
}
.course-source-badge {
  flex: 0 0 auto;
  padding: 2px 9px;
  border-radius: 6px;
  background: rgba(7, 192, 95, 0.1);
  color: var(--td-brand-color-7);
  font-size: 11px;
  line-height: 17px;
}
.course-card-meta {
  display: flex;
  gap: 12px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
  line-height: 18px;
  flex-wrap: wrap;
}
.course-card-summary {
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 20px;
}
.course-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-top: 1px;
}
.course-card-category {
  padding: 2px 9px;
  border-radius: 6px;
  background: rgba(15, 23, 42, 0.05);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 17px;
}
.course-card-date { color: var(--td-text-color-placeholder); font-size: 12px; }

@media (max-width: 720px) {
  .course-grid { grid-template-columns: minmax(0, 1fr); gap: 12px; }
  .course-card { gap: 12px; padding: 12px; }
  .course-card-cover { width: 84px; height: 84px; }
}
</style>
