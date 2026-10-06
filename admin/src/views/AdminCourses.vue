<template>
  <section class="course-page">
    <div class="course-page__header">
      <div>
        <div class="course-eyebrow">平台内容目录</div>
        <h2>系列课程</h2>
        <p>先创建课程信息，再逐集添加课程内容；每次只上传一份课程文件。</p>
      </div>
      <t-button theme="primary" @click="openWizard">
        <template #icon><t-icon name="add" /></template>
        创建课程
      </t-button>
    </div>

    <div class="course-summary">
      <article><span>全部课程</span><strong>{{ stats.total }}</strong></article>
      <article><span>已发布</span><strong>{{ countByStatus('published') }}</strong></article>
      <article><span>已下架</span><strong>{{ countByStatus('offline') }}</strong></article>
      <article><span>课次合计</span><strong>{{ lessonTotal }}</strong></article>
    </div>

    <section class="course-panel">
      <div class="course-toolbar">
        <t-input v-model="keyword" clearable placeholder="搜索课程标题、摘要或讲师" @enter="resetAndLoad">
          <template #prefix-icon><t-icon name="search" /></template>
        </t-input>
        <t-select v-model="statusFilter" style="width: 150px" @change="resetAndLoad">
          <t-option value="all" label="全部状态" />
          <t-option value="published" label="已发布" />
          <t-option value="draft" label="草稿" />
          <t-option value="offline" label="已下架" />
        </t-select>
        <t-select v-model="categoryFilter" style="width: 180px" @change="resetAndLoad">
          <t-option value="" label="全部分类" />
          <t-option v-for="item in DISCOVER_CATEGORY_OPTIONS" :key="item.value" :value="item.value" :label="item.label" />
        </t-select>
      </div>

      <div v-if="loading" class="course-state"><t-loading size="small" /> 加载中</div>
      <div v-else-if="!courses.length" class="course-state">
        <t-icon name="book-open" />
        <span>还没有课程，点右上角先创建课程信息</span>
      </div>
      <div v-else class="course-list">
        <div v-for="item in courses" :key="item.id" class="course-row">
          <div class="course-row__icon"><t-icon name="book-open" /></div>
          <div class="course-row__main">
            <div class="course-row__title">
              <strong>{{ item.title }}</strong>
              <t-tag :theme="statusTheme(item.public_status)" variant="light" size="small">
                {{ statusLabel(item.public_status) }}
              </t-tag>
              <t-tag theme="default" variant="light-outline" size="small">
                {{ sourceLabel(item.source) }}
              </t-tag>
              <t-tag theme="default" variant="light-outline" size="small">
                {{ visibilityLabel(item.visibility_scope) }}
              </t-tag>
              <t-tag v-if="item.featured" theme="warning" variant="light" size="small">精选</t-tag>
              <t-tag v-if="item.recommendable" theme="success" variant="light-outline" size="small">参与推荐</t-tag>
            </div>
            <p>{{ item.summary || '暂无课程简介' }}</p>
            <div class="course-row__meta">
              <span>{{ item.lesson_count }} 讲</span>
              <span v-if="item.teacher_name">讲师：{{ item.teacher_name }}</span>
              <span v-if="item.category">{{ categoryLabel(item.category) }}</span>
              <span>{{ formatDate(item.updated_at) }}</span>
            </div>
          </div>
          <div class="course-row__actions">
            <t-button variant="outline" size="small" @click="openLessonWizard(item)">添加内容</t-button>
            <t-button variant="text" size="small" @click="openDetail(item)">查看</t-button>
            <t-button
              v-if="item.public_status !== 'published'"
              theme="primary"
              variant="outline"
              size="small"
              :loading="busyId === item.id"
              @click="publish(item)"
            >发布</t-button>
            <t-button
              v-if="item.public_status === 'published'"
              theme="warning"
              variant="outline"
              size="small"
              :loading="busyId === item.id"
              @click="offline(item)"
            >下架</t-button>
            <t-button
              theme="danger"
              variant="text"
              size="small"
              :loading="busyId === item.id"
              @click="remove(item)"
            >删除</t-button>
          </div>
        </div>
      </div>
      <div v-if="!loading && total > pageSize" class="course-pagination">
        <t-pagination
          v-model="page"
          :page-size="pageSize"
          :total="total"
          size="small"
          show-page-number
          @change="loadCourses"
        />
      </div>
    </section>

    <!-- Course creation: metadata first, then one lesson per request -->
    <t-dialog
      v-model:visible="wizardVisible"
      header="创建系列课程"
      :width="820"
      :footer="false"
      :close-on-overlay-click="false"
      @close="resetWizard"
    >
      <t-steps :options="stepOptions" :current="step" class="course-steps" />

      <div v-if="step === 0" class="course-step">
        <t-form :data="metaForm" label-align="top" class="course-form">
          <t-form-item label="课程名称">
            <t-input v-model="metaForm.title" placeholder="例如：园所招生话术实操" />
          </t-form-item>
          <t-form-item label="课程简介">
            <t-textarea
              v-model="metaForm.summary"
              :autosize="{ minRows: 3, maxRows: 5 }"
              placeholder="这门课帮学习者解决什么问题"
            />
          </t-form-item>
          <div class="course-form-grid">
            <t-form-item label="经营场景分类">
              <t-select v-model="metaForm.category" placeholder="选择分类">
                <t-option value="" label="暂不分类" />
                <t-option
                  v-for="item in DISCOVER_CATEGORY_OPTIONS"
                  :key="item.value"
                  :value="item.value"
                  :label="item.label"
                />
              </t-select>
            </t-form-item>
            <t-form-item label="发布状态">
              <t-select v-model="metaForm.publicStatus">
                <t-option value="published" label="立即发布到发现页" />
                <t-option value="draft" label="先存草稿" />
              </t-select>
            </t-form-item>
          </div>
          <div class="course-form-grid">
            <t-form-item label="可见范围">
              <t-select v-model="metaForm.visibilityScope">
                <t-option value="system" label="系统公开（所有已登录用户）" />
                <t-option value="shared_space" label="指定共享空间可见" />
                <t-option value="private" label="私有（创建空间可见）" />
              </t-select>
            </t-form-item>
            <t-form-item v-if="metaForm.visibilityScope === 'shared_space'" label="共享空间">
              <t-select
                v-model="metaForm.sharedSpaceIds"
                multiple
                filterable
                :loading="organizationLoading"
                placeholder="选择一个或多个共享空间"
              >
                <t-option
                  v-for="organization in organizations"
                  :key="organization.id"
                  :value="organization.id"
                  :label="organization.name"
                />
              </t-select>
            </t-form-item>
          </div>
          <div class="course-form-grid">
            <t-form-item label="讲师姓名">
              <t-input v-model="metaForm.teacherName" placeholder="选填" />
            </t-form-item>
            <t-form-item label="讲师头衔">
              <t-input v-model="metaForm.teacherTitle" placeholder="选填，例如 教学园长" />
            </t-form-item>
          </div>
          <t-form-item label="课程封面">
            <div class="course-cover-upload">
              <input
                ref="coverInput"
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif"
                class="course-hidden-input"
                @change="onCoverPicked"
              />
              <button type="button" class="course-cover-upload__box" @click="pickCover">
                <img v-if="coverPreviewUrl" :src="coverPreviewUrl" alt="课程封面预览" />
                <span v-else class="course-cover-upload__empty">
                  <t-icon name="upload" />
                  <span>上传课程封面</span>
                </span>
              </button>
              <div class="course-cover-upload__meta">
                <span>{{ coverFileName || '选填，留空则使用首讲内容生成卡片' }}</span>
                <t-button
                  v-if="coverPreviewUrl"
                  size="small"
                  variant="text"
                  theme="danger"
                  @click="clearCover"
                >
                  移除
                </t-button>
              </div>
            </div>
          </t-form-item>
        </t-form>
      </div>

      <div v-else class="course-step">
        <div class="course-lesson-head">
          <div>
            <strong>{{ courseDraft?.title }}</strong>
            <span>{{ courseDraft?.lesson_count || 0 }} 讲</span>
          </div>
          <t-tag :theme="statusTheme(courseDraft?.public_status || 'draft')" variant="light" size="small">
            {{ statusLabel(courseDraft?.public_status || 'draft') }}
          </t-tag>
        </div>
        <input
          ref="lessonInput"
          type="file"
          class="course-hidden-input"
          @change="onLessonPicked"
        />
        <div class="course-lesson-picker" @click="pickLesson">
          <t-icon name="upload" class="course-dropzone__icon" />
          <strong>{{ lessonFileName || '选择一集课程文件' }}</strong>
          <span>{{ lessonFile ? formatSize(lessonFile.size) : '每次选择一个文件并上传' }}</span>
        </div>
        <div v-if="lessonFile" class="course-lesson-selected">
          <span>{{ lessonFileName }}</span>
          <t-button
            theme="primary"
            :loading="uploadingLesson"
            @click="uploadLesson"
          >
            上传这一集
          </t-button>
        </div>
        <div v-if="uploadingLesson" class="course-progress">
          <t-progress :percentage="uploadPercent" :label="true" />
          <span>正在上传并解析当前课程文件，请勿关闭页面</span>
        </div>
        <div class="course-preview-head">
          <span>已添加 {{ lessonItems.length }} 讲</span>
        </div>
        <div v-if="lessonItems.length" class="course-preview">
          <div v-for="(lesson, index) in lessonItems" :key="lesson.id" class="course-preview-row">
            <span class="course-preview-row__order">{{ index + 1 }}</span>
            <t-icon :name="kindIcon(lesson.lesson_type)" class="course-preview-row__icon" />
            <span class="course-preview-row__name">{{ lesson.title }}</span>
            <t-tag theme="default" variant="light-outline" size="small">{{ kindLabel(lesson.lesson_type) }}</t-tag>
          </div>
        </div>
        <div v-else class="course-empty-lessons">课程还没有内容，选择文件后上传第一集</div>
      </div>

      <div class="course-wizard-footer">
        <t-button v-if="step === 1" variant="outline" :disabled="uploadingLesson" @click="step = 0">
          上一步
        </t-button>
        <span class="course-wizard-footer__spacer" />
        <t-button variant="text" :disabled="creatingCourse || uploadingLesson" @click="closeWizard">
          {{ step === 0 ? '取消' : '完成' }}
        </t-button>
        <t-button v-if="step === 0" theme="primary" :loading="creatingCourse" @click="createCourse">
          创建课程
        </t-button>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="detailVisible" header="课程详情" :width="820" :footer="false">
      <div v-if="detail" class="course-detail">
        <div class="course-detail__head">
          <div class="course-detail__heading">
            <strong>{{ detail.title }}</strong>
            <t-tag :theme="statusTheme(detail.public_status)" variant="light" size="small">
              {{ statusLabel(detail.public_status) }}
            </t-tag>
          </div>
          <t-button theme="primary" variant="outline" size="small" @click="addLessonFromDetail">
            <template #icon><t-icon name="add" /></template>
            添加一讲
          </t-button>
        </div>
        <p class="course-detail__summary">{{ detail.summary || '暂无课程简介' }}</p>
        <div class="course-detail__meta">
          <span>课程 ID：{{ detail.id }}</span>
          <span>来源：{{ sourceLabel(detail.source) }}</span>
          <span v-if="detail.teacher_name">讲师：{{ detail.teacher_name }}</span>
          <span v-if="detail.category">分类：{{ categoryLabel(detail.category) }}</span>
        </div>
        <div class="course-detail__visibility">
          <div class="course-detail__visibility-field">
            <span>可见范围</span>
            <t-select v-model="detailVisibilityScope" style="width: 240px">
              <t-option value="system" label="系统公开（所有已登录用户）" />
              <t-option value="shared_space" label="指定共享空间可见" />
              <t-option value="private" label="私有（创建空间可见）" />
            </t-select>
          </div>
          <div v-if="detailVisibilityScope === 'shared_space'" class="course-detail__visibility-field">
            <span>共享空间</span>
            <t-select
              v-model="detailSharedSpaceIds"
              multiple
              filterable
              :loading="organizationLoading"
              style="min-width: 240px; flex: 1"
              placeholder="选择一个或多个共享空间"
            >
              <t-option
                v-for="organization in organizations"
                :key="organization.id"
                :value="organization.id"
                :label="organization.name"
              />
            </t-select>
          </div>
          <div class="course-detail__visibility-actions">
            <t-button
              theme="primary"
              variant="outline"
              size="small"
              :loading="savingVisibility"
              @click="saveVisibility"
            >
              保存可见范围
            </t-button>
          </div>
        </div>
        <div class="course-detail__discovery">
          <div class="course-detail__discovery-fields">
            <t-checkbox v-model="detailFeatured">加入精选</t-checkbox>
            <t-checkbox v-model="detailRecommendable">参与推荐</t-checkbox>
            <label class="course-detail__sort-field">
              <span>推荐排序</span>
              <t-input-number v-model="detailSortOrder" :min="0" :max="9999" theme="normal" />
            </label>
          </div>
          <div class="course-detail__discovery-footer">
            <span>数值越小越靠前，0 表示按更新时间排序。</span>
            <t-button
              theme="primary"
              variant="outline"
              size="small"
              :loading="savingDiscovery"
              @click="saveDiscovery"
            >
              保存发现展示
            </t-button>
          </div>
        </div>
        <div class="course-detail__lessons">
          <div class="course-detail__lessons-title">课程大纲（{{ detail.lessons?.length || 0 }} 讲）</div>
          <div v-for="(lesson, index) in detail.lessons || []" :key="lesson.id" class="course-preview-row course-detail__lesson-row">
            <span class="course-preview-row__order">{{ index + 1 }}</span>
            <t-icon :name="kindIcon(lesson.lesson_type)" class="course-preview-row__icon" />
            <span class="course-preview-row__name">{{ lesson.title }}</span>
            <t-tag theme="default" variant="light-outline" size="small">{{ kindLabel(lesson.lesson_type) }}</t-tag>
            <div class="course-detail__lesson-actions">
              <t-button
                variant="text"
                shape="square"
                title="编辑讲次"
                @click="openLessonEdit(lesson)"
              >
                <template #icon><t-icon name="edit" /></template>
              </t-button>
              <t-button
                variant="text"
                shape="square"
                theme="danger"
                title="删除讲次"
                :loading="deletingLessonId === lesson.id"
                @click="deleteLesson(lesson, index)"
              >
                <template #icon><t-icon name="delete" /></template>
              </t-button>
            </div>
          </div>
          <div v-if="!detail.lessons?.length" class="course-empty-lessons">
            课程还没有内容，点击右上角添加第一讲
          </div>
        </div>
      </div>
    </t-dialog>

    <t-dialog
      v-model:visible="lessonEditVisible"
      header="编辑课程内容"
      :width="520"
      :confirm-btn="{ content: '保存', loading: savingLesson }"
      @confirm="saveLessonEdit"
      @close="resetLessonEdit"
    >
      <t-form v-if="editingLesson" :data="lessonEditForm" label-align="top">
        <t-form-item label="本讲名称">
          <t-input v-model="lessonEditForm.title" placeholder="学习者看到的讲次名称" />
        </t-form-item>
        <t-form-item label="课程描述">
          <t-textarea
            v-model="lessonEditForm.description"
            :autosize="{ minRows: 4, maxRows: 7 }"
            placeholder="补充这一讲的学习重点或内容简介"
          />
        </t-form-item>
        <div class="course-edit-meta">
          <span>内容类型：{{ kindLabel(editingLesson.lesson_type) }}</span>
          <span>原始文件不会被替换，如需替换请删除后重新上传。</span>
        </div>
      </t-form>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  createAdminCourse,
  deleteAdminCourseLesson,
  deleteAdminCourse,
  getAdminCourse,
  listAdminCourseOrganizations,
  listAdminCourses,
  offlineAdminCourse,
  publishAdminCourse,
  updateAdminCourseDiscovery,
  updateAdminCourseVisibility,
  updateAdminCourseLesson,
  uploadAdminCourse,
  type AdminCourse,
  type AdminCourseLesson,
  type AdminCourseOrganization,
  type AdminCourseStats,
  type CourseStatus,
  type CourseVisibilityScope,
} from '@admin/api/course'
import {
  DISCOVER_CATEGORY_OPTIONS,
  discoverCategoryLabel,
} from '@/views/organize/discoverCategories'

const courses = ref<AdminCourse[]>([])
const loading = ref(false)
const busyId = ref('')
const keyword = ref('')
const statusFilter = ref<CourseStatus | 'all'>('all')
const categoryFilter = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = 20
const stats = ref<AdminCourseStats>({ total: 0, published: 0, offline: 0, lesson_total: 0 })
const organizations = ref<AdminCourseOrganization[]>([])
const organizationLoading = ref(false)

const wizardVisible = ref(false)
const step = ref(0)
const coverInput = ref<HTMLInputElement | null>(null)
const lessonInput = ref<HTMLInputElement | null>(null)
const coverFile = ref<File | null>(null)
const coverPreviewUrl = ref('')
const coverFileName = ref('')
const lessonFile = ref<File | null>(null)
const lessonFileName = ref('')
const lessonItems = ref<AdminCourseLesson[]>([])
const courseDraft = ref<AdminCourse | null>(null)
const creatingCourse = ref(false)
const uploadingLesson = ref(false)
const uploadPercent = ref(0)
const metaForm = ref({
  title: '',
  summary: '',
  category: '',
  publicStatus: 'published' as CourseStatus,
  visibilityScope: 'system' as CourseVisibilityScope,
  sharedSpaceIds: [] as string[],
  teacherName: '',
  teacherTitle: '',
  coverUrl: '',
})

const detailVisible = ref(false)
const detail = ref<AdminCourse | null>(null)
const lessonEditVisible = ref(false)
const editingLesson = ref<AdminCourseLesson | null>(null)
const lessonEditForm = ref({ title: '', description: '' })
const savingLesson = ref(false)
const deletingLessonId = ref('')
const detailVisibilityScope = ref<CourseVisibilityScope>('system')
const detailSharedSpaceIds = ref<string[]>([])
const savingVisibility = ref(false)
const detailFeatured = ref(false)
const detailRecommendable = ref(true)
const detailSortOrder = ref(0)
const savingDiscovery = ref(false)

const stepOptions = [
  { title: '填写课程信息' },
  { title: '添加课程内容' },
]

const lessonTotal = computed(() => stats.value.lesson_total)

function countByStatus(status: CourseStatus) {
  if (status === 'published') return stats.value.published
  if (status === 'offline') return stats.value.offline
  return courses.value.filter((item) => item.public_status === status).length
}

function statusLabel(status: CourseStatus) {
  return ({
    draft: '草稿',
    pending_review: '待审核',
    published: '已发布',
    offline: '已下架',
    rejected: '已驳回',
  } as Record<CourseStatus, string>)[status] || status
}

function statusTheme(status: CourseStatus) {
  if (status === 'published') return 'success'
  if (status === 'pending_review') return 'warning'
  if (status === 'rejected') return 'danger'
  return 'default'
}

function sourceLabel(source?: string) {
  return source === 'creator' ? '创作者课程' : '官方精品课'
}

function visibilityLabel(scope?: CourseVisibilityScope) {
  return ({
    system: '系统公开',
    shared_space: '共享空间',
    private: '私有',
  } as Record<CourseVisibilityScope, string>)[scope || 'system']
    || '系统公开'
}

function categoryLabel(category?: string) {
  return discoverCategoryLabel(category) || category || '未分类'
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

function formatSize(bytes: number) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit += 1
  }
  return `${value >= 100 || unit === 0 ? Math.round(value) : value.toFixed(1)} ${units[unit]}`
}

function formatDate(value?: string) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

/* ---------------------------------------------------------------- loading -- */

async function loadCourses() {
  loading.value = true
  try {
    const response = await listAdminCourses({
      status: statusFilter.value,
      category: categoryFilter.value,
      keyword: keyword.value.trim(),
      page: page.value,
      pageSize,
    })
    courses.value = response.data?.items || []
    total.value = response.data?.total || 0
    page.value = response.data?.page || page.value
    stats.value = response.data?.stats || stats.value
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载课程失败')
  } finally {
    loading.value = false
  }
}

function resetAndLoad() {
  page.value = 1
  void loadCourses()
}

async function publish(item: AdminCourse) {
  busyId.value = item.id
  try {
    await publishAdminCourse(item.id)
    MessagePlugin.success('课程已发布，讲义同步上线')
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '发布失败')
  } finally {
    busyId.value = ''
  }
}

async function offline(item: AdminCourse) {
  busyId.value = item.id
  try {
    await offlineAdminCourse(item.id)
    MessagePlugin.success('课程已下架，讲义同步下线')
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '下架失败')
  } finally {
    busyId.value = ''
  }
}

async function remove(item: AdminCourse) {
  const confirmed = await new Promise<boolean>((resolve) => {
    // eslint-disable-next-line no-alert
    resolve(window.confirm(`删除课程「${item.title}」？课程内的讲义会一并从课程中移除，此操作不可撤销。`))
  })
  if (!confirmed) return
  busyId.value = item.id
  try {
    await deleteAdminCourse(item.id)
    MessagePlugin.success('课程已删除')
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '删除失败')
  } finally {
    busyId.value = ''
  }
}

async function openDetail(item: AdminCourse) {
  try {
    const response = await getAdminCourse(item.id)
    detail.value = response.data || null
    detailVisibilityScope.value = detail.value?.visibility_scope || 'system'
    detailSharedSpaceIds.value = [...(detail.value?.shared_space_ids || [])]
    detailFeatured.value = Boolean(detail.value?.featured)
    detailRecommendable.value = detail.value?.recommendable !== false
    detailSortOrder.value = detail.value?.sort_order || 0
    detailVisible.value = true
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载课程详情失败')
  }
}

async function saveDiscovery() {
  if (!detail.value) return
  savingDiscovery.value = true
  try {
    const response = await updateAdminCourseDiscovery(detail.value.id, {
      featured: detailFeatured.value,
      recommendable: detailRecommendable.value,
      sortOrder: detailSortOrder.value,
    })
    detail.value = response.data || detail.value
    detailFeatured.value = Boolean(detail.value.featured)
    detailRecommendable.value = detail.value.recommendable !== false
    detailSortOrder.value = detail.value.sort_order || 0
    MessagePlugin.success('课程发现展示已保存')
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存发现展示失败')
  } finally {
    savingDiscovery.value = false
  }
}

async function saveVisibility() {
  if (!detail.value) return
  if (detailVisibilityScope.value === 'shared_space' && !detailSharedSpaceIds.value.length) {
    MessagePlugin.warning('请选择至少一个共享空间')
    return
  }
  savingVisibility.value = true
  try {
    const response = await updateAdminCourseVisibility(detail.value.id, {
      visibilityScope: detailVisibilityScope.value,
      sharedSpaceIds: detailVisibilityScope.value === 'shared_space'
        ? detailSharedSpaceIds.value
        : [],
    })
    detail.value = response.data || detail.value
    detailVisibilityScope.value = detail.value.visibility_scope || 'system'
    detailSharedSpaceIds.value = [...(detail.value.shared_space_ids || [])]
    MessagePlugin.success('课程可见范围已保存')
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存可见范围失败')
  } finally {
    savingVisibility.value = false
  }
}

function addLessonFromDetail() {
  if (!detail.value) return
  const course = detail.value
  detailVisible.value = false
  void openLessonWizard(course)
}

function openLessonEdit(lesson: AdminCourseLesson) {
  editingLesson.value = lesson
  lessonEditForm.value = {
    title: lesson.title || '',
    description: lesson.output?.source_summary || '',
  }
  lessonEditVisible.value = true
}

function resetLessonEdit() {
  editingLesson.value = null
  lessonEditForm.value = { title: '', description: '' }
  savingLesson.value = false
}

async function saveLessonEdit() {
  if (!detail.value || !editingLesson.value) return
  const title = lessonEditForm.value.title.trim()
  if (!title) {
    MessagePlugin.warning('请输入课程名称')
    return
  }

  savingLesson.value = true
  try {
    await updateAdminCourseLesson(detail.value.id, editingLesson.value.id, {
      title,
      description: lessonEditForm.value.description.trim(),
    })
    MessagePlugin.success('课程内容已保存')
    lessonEditVisible.value = false
    const response = await getAdminCourse(detail.value.id)
    detail.value = response.data || detail.value
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '保存课程内容失败')
  } finally {
    savingLesson.value = false
  }
}

async function deleteLesson(lesson: AdminCourseLesson, index: number) {
  if (!detail.value) return
  // eslint-disable-next-line no-alert
  const confirmed = window.confirm(`删除第 ${index + 1} 讲「${lesson.title}」？视频文件和这一讲的内容也会一并删除。`)
  if (!confirmed) return

  deletingLessonId.value = lesson.id
  try {
    await deleteAdminCourseLesson(detail.value.id, lesson.id)
    MessagePlugin.success('课程内容已删除')
    const response = await getAdminCourse(detail.value.id)
    detail.value = response.data || detail.value
    await loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '删除课程内容失败')
  } finally {
    deletingLessonId.value = ''
  }
}

async function openLessonWizard(item: AdminCourse) {
  try {
    const response = await getAdminCourse(item.id)
    courseDraft.value = response.data || item
    lessonItems.value = response.data?.lessons || []
    step.value = 1
    clearLesson()
    clearCover()
    wizardVisible.value = true
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载课程内容失败')
  }
}

/* ---------------------------------------------------------------- wizard --- */

function openWizard() {
  resetWizard()
  wizardVisible.value = true
}

function closeWizard() {
  wizardVisible.value = false
  resetWizard()
  void loadCourses()
}

function resetWizard() {
  step.value = 0
  courseDraft.value = null
  lessonItems.value = []
  clearLesson()
  clearCover()
  creatingCourse.value = false
  uploadingLesson.value = false
  uploadPercent.value = 0
  metaForm.value = {
    title: '',
    summary: '',
    category: '',
    publicStatus: 'published',
    visibilityScope: 'system',
    sharedSpaceIds: [],
    teacherName: '',
    teacherTitle: '',
    coverUrl: '',
  }
}

function pickCover() {
  const input = coverInput.value
  if (!input) return
  input.value = ''
  input.click()
}

function onCoverPicked(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const ext = file.name.split('.').pop()?.toLowerCase() || ''
  const imageExts = new Set(['png', 'jpg', 'jpeg', 'webp', 'gif'])
  if (!file.type.startsWith('image/') && !imageExts.has(ext)) {
    MessagePlugin.warning('请选择图片文件作为课程封面')
    input.value = ''
    return
  }
  if (coverPreviewUrl.value) {
    URL.revokeObjectURL(coverPreviewUrl.value)
  }
  coverFile.value = file
  coverFileName.value = file.name
  coverPreviewUrl.value = URL.createObjectURL(file)
}

function clearCover() {
  if (coverPreviewUrl.value) {
    URL.revokeObjectURL(coverPreviewUrl.value)
  }
  coverFile.value = null
  coverPreviewUrl.value = ''
  coverFileName.value = ''
  if (coverInput.value) {
    coverInput.value.value = ''
  }
}

function pickLesson() {
  const input = lessonInput.value
  if (!input) return
  input.value = ''
  input.click()
}

function onLessonPicked(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  lessonFile.value = file
  lessonFileName.value = file.name
}

function clearLesson() {
  lessonFile.value = null
  lessonFileName.value = ''
  if (lessonInput.value) {
    lessonInput.value.value = ''
  }
}

async function createCourse() {
  if (!metaForm.value.title.trim()) {
    MessagePlugin.warning('请填写课程名称')
    return
  }
  if (metaForm.value.visibilityScope === 'shared_space' && !metaForm.value.sharedSpaceIds.length) {
    MessagePlugin.warning('请选择至少一个共享空间')
    return
  }

  creatingCourse.value = true
  uploadPercent.value = 0
  try {
    const response = await createAdminCourse({
      title: metaForm.value.title.trim(),
      summary: metaForm.value.summary.trim(),
      category: metaForm.value.category,
      coverUrl: metaForm.value.coverUrl.trim(),
      coverImage: coverFile.value,
      teacherName: metaForm.value.teacherName.trim(),
      teacherTitle: metaForm.value.teacherTitle.trim(),
      source: 'official',
      publicStatus: metaForm.value.publicStatus,
      visibilityScope: metaForm.value.visibilityScope,
      sharedSpaceIds: metaForm.value.sharedSpaceIds,
    })
    courseDraft.value = response.data
    lessonItems.value = response.data?.lessons || []
    step.value = 1
    MessagePlugin.success('课程信息已创建，现在可以逐集上传')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '创建课程失败')
  } finally {
    creatingCourse.value = false
  }
}

async function uploadLesson() {
  if (!courseDraft.value) {
    MessagePlugin.warning('请先创建课程信息')
    return
  }
  if (!lessonFile.value) {
    MessagePlugin.warning('请选择一集课程文件')
    return
  }

  uploadingLesson.value = true
  uploadPercent.value = 0
  try {
    const file = lessonFile.value
    const response = await uploadAdminCourse(
      [{ file, relativePath: file.name }],
      {
        title: courseDraft.value.title,
        summary: courseDraft.value.summary,
        category: courseDraft.value.category,
        courseId: courseDraft.value.id,
        teacherName: courseDraft.value.teacher_name,
        teacherTitle: courseDraft.value.teacher_title,
        source: courseDraft.value.source,
        publicStatus: courseDraft.value.public_status,
        visibilityScope: courseDraft.value.visibility_scope,
        sharedSpaceIds: courseDraft.value.shared_space_ids,
      },
      (event: any) => {
        uploadPercent.value = event?.total
          ? Math.min(99, Math.round((event.loaded / event.total) * 100))
          : 0
      },
    )
    const nextCourse = response.data?.course
    if (nextCourse) {
      const appendedLesson = nextCourse.lessons?.[0]
      courseDraft.value = { ...courseDraft.value, ...nextCourse }
      if (
        appendedLesson
        && !lessonItems.value.some((item) => item.id === appendedLesson.id)
      ) {
        lessonItems.value.push(appendedLesson)
      }
    }
    clearLesson()
    uploadPercent.value = 100
    MessagePlugin.success('本集上传成功')
    void loadCourses()
  } catch (error: any) {
    MessagePlugin.error(error?.message || '本集上传失败')
  } finally {
    uploadingLesson.value = false
  }
}

async function loadOrganizations() {
  organizationLoading.value = true
  try {
    const response = await listAdminCourseOrganizations()
    organizations.value = response.data?.items || []
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载共享空间失败')
  } finally {
    organizationLoading.value = false
  }
}

onMounted(() => {
  void loadCourses()
  void loadOrganizations()
})

onBeforeUnmount(() => {
  clearCover()
  clearLesson()
})
</script>

<style scoped>
.course-page { padding: 4px 0 32px; }
.course-page__header { display: flex; align-items: flex-start; justify-content: space-between; gap: 20px; margin-bottom: 22px; }
.course-eyebrow { color: var(--td-text-color-secondary); font-size: 12px; margin-bottom: 6px; }
.course-page h2 { margin: 0; color: var(--td-text-color-primary); font-size: 24px; }
.course-page__header p { margin: 8px 0 0; color: var(--td-text-color-secondary); max-width: 640px; line-height: 1.6; }

.course-summary { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; margin-bottom: 18px; }
.course-summary article { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); padding: 14px 16px; }
.course-summary span { display: block; color: var(--td-text-color-secondary); font-size: 12px; }
.course-summary strong { display: block; margin-top: 6px; color: var(--td-text-color-primary); font-size: 22px; }

.course-panel { border: 1px solid var(--td-component-border); background: var(--td-bg-color-container); }
.course-toolbar { display: flex; gap: 12px; padding: 16px; border-bottom: 1px solid var(--td-component-border); }
.course-toolbar .t-input { width: min(360px, 100%); }
.course-list { padding: 0 16px; }
.course-pagination { display: flex; justify-content: flex-end; padding: 12px 16px 16px; }
.course-row { display: flex; align-items: center; gap: 14px; padding: 16px 0; border-bottom: 1px solid var(--td-component-border); }
.course-row:last-child { border-bottom: 0; }
.course-row__icon { display: flex; align-items: center; justify-content: center; flex: 0 0 38px; height: 38px; color: var(--td-brand-color); background: var(--td-brand-color-light); border-radius: 6px; font-size: 20px; }
.course-row__main { min-width: 0; flex: 1; }
.course-row__title { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.course-row__title strong { color: var(--td-text-color-primary); font-size: 15px; }
.course-row__main p { margin: 6px 0; color: var(--td-text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.course-row__meta { display: flex; gap: 14px; color: var(--td-text-color-placeholder); font-size: 12px; flex-wrap: wrap; }
.course-row__actions { display: flex; align-items: center; gap: 4px; flex-wrap: wrap; justify-content: flex-end; }
.course-state { min-height: 220px; display: flex; align-items: center; justify-content: center; gap: 8px; color: var(--td-text-color-secondary); }

.course-steps { margin-bottom: 22px; }
.course-step { min-height: 260px; }
.course-hidden-input { display: none; }
.course-dropzone { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; padding: 38px 20px; border: 1px dashed var(--td-component-border); background: var(--td-bg-color-container-hover); cursor: pointer; text-align: center; }
.course-dropzone:hover { border-color: var(--td-brand-color); }
.course-dropzone__icon { font-size: 34px; color: var(--td-brand-color); }
.course-dropzone strong { color: var(--td-text-color-primary); font-size: 15px; }
.course-dropzone span { color: var(--td-text-color-secondary); font-size: 12px; }
.course-hint { margin: 14px 0 0; color: var(--td-text-color-placeholder); font-size: 12px; line-height: 1.7; }

.course-preview-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px; color: var(--td-text-color-secondary); font-size: 13px; }
.course-preview-skip { color: var(--td-warning-color); }
.course-preview { border: 1px solid var(--td-component-border); max-height: 340px; overflow-y: auto; }
.course-preview-row { display: flex; align-items: center; gap: 10px; padding: 10px 12px; border-bottom: 1px solid var(--td-component-border); font-size: 13px; }
.course-preview-row:last-child { border-bottom: 0; }
.course-preview-row--muted { color: var(--td-text-color-placeholder); background: var(--td-bg-color-container-hover); }
.course-preview-row__order { flex: 0 0 28px; color: var(--td-text-color-placeholder); text-align: right; font-variant-numeric: tabular-nums; }
.course-preview-row__icon { flex: 0 0 auto; color: var(--td-brand-color); }
.course-preview-row__name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--td-text-color-primary); }
.course-preview-row--muted .course-preview-row__name { color: var(--td-text-color-placeholder); }
.course-preview-row__size { flex: 0 0 auto; color: var(--td-text-color-placeholder); font-size: 12px; }

.course-form { max-width: 100%; }
.course-form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 16px; }
.course-cover-upload { display: flex; align-items: center; gap: 12px; }
.course-cover-upload__box {
  width: 152px;
  aspect-ratio: 16 / 9;
  border: 1px dashed var(--td-component-border);
  background: var(--td-bg-color-container-hover);
  border-radius: 6px;
  padding: 0;
  overflow: hidden;
  cursor: pointer;
}
.course-cover-upload__box:hover { border-color: var(--td-brand-color); }
.course-cover-upload__box img { display: block; width: 100%; height: 100%; object-fit: cover; }
.course-cover-upload__empty {
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--td-text-color-secondary);
  font-size: 12px;
}
.course-cover-upload__empty .t-icon { color: var(--td-brand-color); font-size: 22px; }
.course-cover-upload__meta { min-width: 0; display: flex; align-items: center; gap: 8px; color: var(--td-text-color-placeholder); font-size: 12px; }
.course-cover-upload__meta > span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.course-progress { margin-top: 8px; display: flex; flex-direction: column; gap: 8px; }
.course-progress span { color: var(--td-text-color-secondary); font-size: 12px; }
.course-lesson-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 18px; }
.course-lesson-head > div { display: flex; align-items: baseline; gap: 10px; min-width: 0; }
.course-lesson-head strong { overflow: hidden; color: var(--td-text-color-primary); font-size: 16px; text-overflow: ellipsis; white-space: nowrap; }
.course-lesson-head span { color: var(--td-text-color-secondary); font-size: 13px; }
.course-lesson-picker { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; min-height: 148px; padding: 24px 20px; border: 1px dashed var(--td-component-border); background: var(--td-bg-color-container-hover); cursor: pointer; text-align: center; }
.course-lesson-picker:hover { border-color: var(--td-brand-color); }
.course-lesson-picker .course-dropzone__icon { font-size: 30px; }
.course-lesson-picker strong { max-width: 100%; overflow: hidden; color: var(--td-text-color-primary); font-size: 15px; text-overflow: ellipsis; white-space: nowrap; }
.course-lesson-picker span { color: var(--td-text-color-secondary); font-size: 12px; }
.course-lesson-selected { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-top: 12px; padding: 10px 12px; border: 1px solid var(--td-component-border); }
.course-lesson-selected > span { min-width: 0; overflow: hidden; color: var(--td-text-color-primary); font-size: 13px; text-overflow: ellipsis; white-space: nowrap; }
.course-empty-lessons { padding: 28px 12px; color: var(--td-text-color-placeholder); font-size: 13px; text-align: center; }

.course-done { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 26px 20px 18px; text-align: center; }
.course-done__icon { font-size: 40px; color: var(--td-success-color); }
.course-done strong { color: var(--td-text-color-primary); font-size: 16px; }
.course-done span { color: var(--td-text-color-secondary); font-size: 13px; }
.course-done__skipped { margin-top: 8px; }
.course-done__skipped-title { margin-bottom: 8px; color: var(--td-text-color-secondary); font-size: 13px; }

.course-wizard-footer { display: flex; align-items: center; gap: 8px; margin-top: 22px; padding-top: 16px; border-top: 1px solid var(--td-component-border); }
.course-wizard-footer__spacer { flex: 1; }

.course-detail__head { display: flex; align-items: center; justify-content: space-between; gap: 16px; }
.course-detail__heading { display: flex; align-items: center; gap: 8px; min-width: 0; }
.course-detail__heading strong { overflow: hidden; color: var(--td-text-color-primary); font-size: 16px; text-overflow: ellipsis; white-space: nowrap; }
.course-detail__summary { margin: 10px 0; color: var(--td-text-color-secondary); line-height: 1.7; }
.course-detail__meta { display: flex; gap: 14px; flex-wrap: wrap; color: var(--td-text-color-placeholder); font-size: 12px; margin-bottom: 16px; }
.course-detail__visibility { display: flex; flex-direction: column; gap: 10px; margin-bottom: 18px; padding: 12px; border: 1px solid var(--td-component-border); background: var(--td-bg-color-container-hover); }
.course-detail__visibility-field { display: flex; align-items: center; gap: 12px; }
.course-detail__visibility-field > span { flex: 0 0 64px; color: var(--td-text-color-secondary); font-size: 12px; }
.course-detail__visibility-actions { display: flex; justify-content: flex-end; }
.course-detail__discovery { display: flex; flex-direction: column; gap: 10px; margin-bottom: 18px; padding: 12px; border: 1px solid var(--td-component-border); background: var(--td-bg-color-container-hover); }
.course-detail__discovery-fields { display: flex; align-items: center; gap: 18px; flex-wrap: wrap; }
.course-detail__sort-field { display: inline-flex; align-items: center; gap: 8px; color: var(--td-text-color-secondary); font-size: 12px; }
.course-detail__sort-field .t-input-number { width: 120px; }
.course-detail__discovery-footer { display: flex; align-items: center; justify-content: space-between; gap: 12px; color: var(--td-text-color-placeholder); font-size: 12px; }
.course-detail__lessons-title { margin-bottom: 8px; color: var(--td-text-color-secondary); font-size: 13px; }
.course-detail__lessons .course-preview-row { border: 1px solid var(--td-component-border); border-bottom: 0; }
.course-detail__lessons .course-preview-row:last-child { border-bottom: 1px solid var(--td-component-border); }
.course-detail__lesson-row { min-height: 46px; }
.course-detail__lesson-actions { display: flex; align-items: center; gap: 2px; margin-left: 4px; }
.course-edit-meta { display: flex; flex-direction: column; gap: 6px; color: var(--td-text-color-placeholder); font-size: 12px; line-height: 1.5; }

@media (max-width: 900px) {
  .course-page__header { flex-direction: column; }
  .course-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .course-toolbar { flex-wrap: wrap; }
  .course-row { align-items: flex-start; flex-wrap: wrap; }
  .course-row__actions { width: 100%; justify-content: flex-start; padding-left: 52px; }
  .course-form-grid { grid-template-columns: minmax(0, 1fr); }
  .course-detail__visibility-field { align-items: flex-start; flex-direction: column; gap: 6px; }
  .course-detail__discovery-footer { align-items: flex-start; flex-direction: column; }
}
</style>
