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
          <img v-if="item.cover_url" :src="item.cover_url" :alt="item.title" />
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
  <div v-else-if="props.variant === 'featured'" class="course-discover course-discover--empty">
    <t-icon name="book-open" />
    <span>暂无可推荐内容或课程</span>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { useRouter } from 'vue-router'
import { getOrganizeCourses, type OrganizeCourse } from '@/api/organize'
import { discoverCategoryLabel } from './discoverCategories'

const props = withDefaults(defineProps<{
  variant?: 'feed' | 'featured'
  limit?: number
}>(), {
  variant: 'feed',
  limit: 6,
})

const router = useRouter()

const courses = ref<OrganizeCourse[]>([])
const loading = ref(true)

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

// Courses ship a cover only for platform-authored material; the rest get the
// same soft brand wash used by the recommendation cards.
function coverStyle(item: OrganizeCourse) {
  if (item.cover_url) return { backgroundImage: `url(${item.cover_url})` }
  return { background: 'linear-gradient(135deg, var(--td-brand-color-light) 0%, var(--td-bg-color-secondarycontainer) 100%)' }
}

async function loadCourses() {
  loading.value = true
  try {
    const response = await getOrganizeCourses({
      page: 1,
      page_size: Math.max(1, props.limit),
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || '课程加载失败')
    }
    courses.value = response.data.items || []
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

defineExpose({ reload: loadCourses })
</script>

<style scoped>
.course-discover { display: flex; flex-direction: column; gap: 12px; }
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

.course-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(360px, 1fr)); gap: 10px; }

.course-card {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  align-items: start;
  gap: 12px;
  min-height: 124px;
  padding: 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}
.course-card:hover { border-color: var(--td-brand-color); box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12); }
.course-card:focus-visible { outline: 2px solid var(--td-brand-color); outline-offset: 2px; }

.course-card-cover {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: 88px;
  height: 88px;
  padding: 10px;
  box-sizing: border-box;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background-color: var(--td-bg-color-secondarycontainer);
  background-size: cover;
  background-position: center;
  color: var(--td-text-color-primary);
  font-size: 26px;
}
.course-card-cover img { width: 100%; height: 100%; object-fit: cover; border-radius: 4px; }
.course-card-cover__kind { font-size: 11px; color: var(--td-text-color-secondary); }

.course-card-body { min-width: 0; display: flex; flex-direction: column; gap: 6px; }
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
}
.course-source-badge {
  flex: 0 0 auto;
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--td-brand-color-light);
  color: var(--td-brand-color);
  font-size: 11px;
  line-height: 18px;
}
.course-card-meta { display: flex; gap: 12px; color: var(--td-text-color-placeholder); font-size: 12px; flex-wrap: wrap; }
.course-card-summary {
  margin: 0;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.55;
}
.course-card-footer { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-top: 2px; }
.course-card-category {
  padding: 2px 8px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: 11px;
  line-height: 18px;
}
.course-card-date { color: var(--td-text-color-placeholder); font-size: 12px; }

@media (max-width: 720px) {
  .course-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
