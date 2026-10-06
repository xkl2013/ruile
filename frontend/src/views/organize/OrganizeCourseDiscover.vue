<template>
  <div v-if="loading" class="course-discover course-discover--loading">
    <t-loading size="medium" text="加载课程中" />
  </div>
  <div v-else-if="courses.length" class="course-discover" :class="{ 'course-discover--featured': props.variant === 'featured' }">
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
  <div v-else-if="props.variant === 'featured' && props.showEmpty" class="course-discover course-discover--empty">
    <t-icon name="book-open" />
    <span>暂无可推荐内容或课程</span>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import { getOrganizeCourseCover, getOrganizeCourses, type OrganizeCourse } from '@/api/organize'
import { discoverCategoryLabel } from './discoverCategories'

const props = withDefaults(defineProps<{
  variant?: 'feed' | 'featured'
  limit?: number
  showEmpty?: boolean
}>(), {
  variant: 'feed',
  limit: 6,
  showEmpty: false,
})

const router = useRouter()

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
      page_size: Math.max(1, props.limit),
      featured: props.variant === 'featured',
      recommendable: props.variant === 'feed' ? true : undefined,
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
.course-discover--empty {
  min-height: 120px;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}
.course-discover--empty .t-icon { font-size: 20px; }

.course-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  width: min(100%, 768px);
}

.course-card {
  display: grid;
  width: 100%;
  height: 91px;
  min-height: 91px;
  grid-template-columns: 56px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  box-sizing: border-box;
  padding: 16px;
  border: 1px solid rgba(0, 0, 0, 0.04);
  border-radius: 16px;
  background: var(--td-bg-color-container);
  box-shadow: none;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}
.course-card:hover {
  border-color: rgba(0, 0, 0, 0.1);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}
.course-card:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }

.course-card-cover {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 56px;
  height: 56px;
  padding: 0;
  box-sizing: border-box;
  border: 0.5px solid rgba(0, 0, 0, 0.04);
  border-radius: 8px;
  background-color: var(--td-bg-color-secondarycontainer);
  background-size: cover;
  background-position: center;
  color: var(--td-brand-color);
  font-size: 20px;
}
.course-card-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 8px;
}
.course-card-cover__kind { display: none; }

.course-card-body { min-width: 0; display: flex; flex-direction: column; justify-content: center; gap: 0; overflow: hidden; }
.course-card-title { display: flex; align-items: center; gap: 8px; min-width: 0; }
.course-card-title h3 {
  margin: 0;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--td-text-color-primary);
  font-size: 14px;
  font-weight: 400;
  line-height: 16px;
}
.course-source-badge {
  flex: 0 0 auto;
  padding: 1px 6px;
  border-radius: 5px;
  background: rgba(7, 192, 95, 0.1);
  color: var(--td-brand-color-7);
  font-size: 11px;
  line-height: 14px;
}
.course-card-meta {
  display: flex;
  gap: 2px;
  color: rgba(0, 0, 0, 0.44);
  font-size: 11px;
  line-height: 14px;
  flex-wrap: wrap;
}
.course-card-summary {
  margin: 0;
  margin-top: 4px;
  display: -webkit-box;
  -webkit-line-clamp: 1;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: rgba(0, 0, 0, 0.44);
  font-size: 11px;
  line-height: 14px;
}
.course-card-footer {
  display: none;
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
  .course-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; width: 100%; }
  .course-card { height: 132px; min-height: 132px; }
}
</style>
