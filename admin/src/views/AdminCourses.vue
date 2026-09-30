<template>
  <section class="course-page">
    <div class="course-page__header">
      <div>
        <div class="course-eyebrow">平台内容目录</div>
        <h2>系列课程</h2>
        <p>上传一个文件夹即生成一门课程：系统按文件名前缀数字决定讲次顺序，逐个解析内容后一次性发布到发现模块。</p>
      </div>
      <t-button theme="primary" @click="openWizard">
        <template #icon><t-icon name="folder-add" /></template>
        上传文件夹建课
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
        <t-icon name="folder-open" />
        <span>还没有课程，点右上角上传一个文件夹试试</span>
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

    <!-- Folder-upload wizard: pick folder -> preview -> metadata -> result -->
    <t-dialog
      v-model:visible="wizardVisible"
      header="上传文件夹建课"
      :width="820"
      :footer="false"
      :close-on-overlay-click="false"
      @close="resetWizard"
    >
      <t-steps :options="stepOptions" :current="step" class="course-steps" />

      <div v-if="step === 0" class="course-step">
        <input
          ref="folderInput"
          type="file"
          multiple
          webkitdirectory
          directory
          class="course-hidden-input"
          @change="onFolderPicked"
        />
        <div class="course-dropzone" @click="pickFolder">
          <t-icon name="folder-add" class="course-dropzone__icon" />
          <strong>{{ pickedFiles.length ? pickedFolderName : '选择本地文件夹' }}</strong>
          <span v-if="pickedFiles.length">
            已读取 {{ pickedFiles.length }} 个文件，其中 {{ usableFiles.length }} 个可作为讲次
          </span>
          <span v-else>点此打开系统文件夹选择器，一次选中整门课的材料</span>
        </div>
        <p class="course-hint">
          建议把文件名写成「01_开课说明」「02_家长沟通」这样的形式，系统会按前缀数字排讲次。
          .DS_Store、__MACOSX 等系统文件会被自动忽略。
        </p>
      </div>

      <div v-else-if="step === 1" class="course-step">
        <div class="course-preview-head">
          <span>共 {{ usableFiles.length }} 讲将发布</span>
          <span v-if="skippedFiles.length" class="course-preview-skip">
            已忽略 {{ skippedFiles.length }} 个文件
          </span>
        </div>
        <div class="course-preview">
          <div v-for="(item, index) in usableFiles" :key="item.relativePath" class="course-preview-row">
            <span class="course-preview-row__order">{{ index + 1 }}</span>
            <t-icon :name="kindIcon(item.kind)" class="course-preview-row__icon" />
            <span class="course-preview-row__name" :title="item.relativePath">{{ item.name }}</span>
            <t-tag theme="default" variant="light-outline" size="small">{{ kindLabel(item.kind) }}</t-tag>
            <span class="course-preview-row__size">{{ formatSize(item.file.size) }}</span>
          </div>
          <div v-for="item in skippedFiles" :key="item.relativePath" class="course-preview-row course-preview-row--muted">
            <span class="course-preview-row__order">—</span>
            <t-icon name="close-circle" class="course-preview-row__icon" />
            <span class="course-preview-row__name" :title="item.relativePath">{{ item.name }}</span>
            <t-tag theme="default" variant="light" size="small">已忽略</t-tag>
            <span class="course-preview-row__size">{{ item.skipReason }}</span>
          </div>
        </div>
      </div>

      <div v-else-if="step === 2" class="course-step">
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
            <t-form-item label="讲师姓名">
              <t-input v-model="metaForm.teacherName" placeholder="选填" />
            </t-form-item>
            <t-form-item label="讲师头衔">
              <t-input v-model="metaForm.teacherTitle" placeholder="选填，例如 教学园长" />
            </t-form-item>
          </div>
          <t-form-item label="课程封面地址">
            <t-input v-model="metaForm.coverUrl" placeholder="选填，留空则使用首讲内容生成卡片" />
          </t-form-item>
        </t-form>

        <div v-if="uploading" class="course-progress">
          <t-progress :percentage="uploadPercent" :label="true" />
          <span>正在上传并解析 {{ usableFiles.length }} 个文件，视频文件耗时较长，请勿关闭页面</span>
        </div>
      </div>

      <div v-else class="course-step">
        <div class="course-done">
          <t-icon name="check-circle-filled" class="course-done__icon" />
          <strong>{{ result?.course?.title || '课程已创建' }}</strong>
          <span>
            已生成 {{ result?.course?.lesson_count || 0 }} 讲，状态为
            {{ statusLabel((result?.course?.public_status || 'published') as CourseStatus) }}
          </span>
        </div>
        <div v-if="result?.skipped?.length" class="course-done__skipped">
          <div class="course-done__skipped-title">以下文件未生成讲次</div>
          <div v-for="item in result.skipped" :key="item.file_name" class="course-preview-row course-preview-row--muted">
            <span class="course-preview-row__name" :title="item.file_name">{{ item.file_name }}</span>
            <span class="course-preview-row__size">{{ item.reason }}</span>
          </div>
        </div>
      </div>

      <div class="course-wizard-footer">
        <t-button v-if="step > 0 && step < 3" variant="outline" :disabled="uploading" @click="step -= 1">
          上一步
        </t-button>
        <span class="course-wizard-footer__spacer" />
        <t-button variant="text" :disabled="uploading" @click="closeWizard">
          {{ step === 3 ? '完成' : '取消' }}
        </t-button>
        <t-button v-if="step === 0" theme="primary" :disabled="!usableFiles.length" @click="step = 1">
          下一步
        </t-button>
        <t-button v-else-if="step === 1" theme="primary" @click="goToMeta">
          下一步
        </t-button>
        <t-button v-else-if="step === 2" theme="primary" :loading="uploading" @click="submit">
          开始上传（{{ usableFiles.length }} 讲）
        </t-button>
        <t-button v-else theme="primary" @click="closeWizard">完成</t-button>
      </div>
    </t-dialog>

    <t-dialog v-model:visible="detailVisible" header="课程详情" :width="720" :footer="false">
      <div v-if="detail" class="course-detail">
        <div class="course-detail__head">
          <strong>{{ detail.title }}</strong>
          <t-tag :theme="statusTheme(detail.public_status)" variant="light" size="small">
            {{ statusLabel(detail.public_status) }}
          </t-tag>
        </div>
        <p class="course-detail__summary">{{ detail.summary || '暂无课程简介' }}</p>
        <div class="course-detail__meta">
          <span>课程 ID：{{ detail.id }}</span>
          <span>来源：{{ sourceLabel(detail.source) }}</span>
          <span v-if="detail.teacher_name">讲师：{{ detail.teacher_name }}</span>
          <span v-if="detail.category">分类：{{ categoryLabel(detail.category) }}</span>
        </div>
        <div class="course-detail__lessons">
          <div class="course-detail__lessons-title">课程大纲（{{ detail.lessons?.length || 0 }} 讲）</div>
          <div v-for="(lesson, index) in detail.lessons || []" :key="lesson.id" class="course-preview-row">
            <span class="course-preview-row__order">{{ index + 1 }}</span>
            <t-icon :name="kindIcon(lesson.lesson_type)" class="course-preview-row__icon" />
            <span class="course-preview-row__name">{{ lesson.title }}</span>
            <t-tag theme="default" variant="light-outline" size="small">{{ kindLabel(lesson.lesson_type) }}</t-tag>
          </div>
        </div>
      </div>
    </t-dialog>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  deleteAdminCourse,
  getAdminCourse,
  listAdminCourses,
  offlineAdminCourse,
  publishAdminCourse,
  uploadAdminCourse,
  type AdminCourse,
  type AdminCourseSkippedFile,
  type AdminCourseStats,
  type AdminCourseUploadResult,
  type CourseLessonType,
  type CourseStatus,
} from '@admin/api/course'
import {
  DISCOVER_CATEGORY_OPTIONS,
  discoverCategoryLabel,
} from '@/views/organize/discoverCategories'

const VIDEO_EXTS = ['mp4', 'mov', 'm4v', 'avi', 'mkv', 'webm', 'flv', 'wmv', 'mpeg', 'mpg']
const AUDIO_EXTS = ['mp3', 'wav', 'm4a', 'aac', 'flac', 'ogg', 'wma', 'amr', 'opus']

interface PreviewItem {
  file: File
  relativePath: string
  name: string
  kind: CourseLessonType
  skipReason: string
}

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

const wizardVisible = ref(false)
const step = ref(0)
const folderInput = ref<HTMLInputElement | null>(null)
const pickedFiles = ref<PreviewItem[]>([])
const pickedFolderName = ref('')
const uploading = ref(false)
const uploadPercent = ref(0)
const result = ref<AdminCourseUploadResult | null>(null)
const metaForm = ref({
  title: '',
  summary: '',
  category: '',
  publicStatus: 'published' as CourseStatus,
  teacherName: '',
  teacherTitle: '',
  coverUrl: '',
})

const detailVisible = ref(false)
const detail = ref<AdminCourse | null>(null)

const stepOptions = [
  { title: '选择文件夹' },
  { title: '预览讲次' },
  { title: '填写课程信息' },
  { title: '发布完成' },
]

const usableFiles = computed(() => pickedFiles.value.filter((item) => !item.skipReason))
const skippedFiles = computed(() => pickedFiles.value.filter((item) => item.skipReason))
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
    detailVisible.value = true
  } catch (error: any) {
    MessagePlugin.error(error?.message || '加载课程详情失败')
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
  pickedFiles.value = []
  pickedFolderName.value = ''
  uploading.value = false
  uploadPercent.value = 0
  result.value = null
  metaForm.value = {
    title: '',
    summary: '',
    category: '',
    publicStatus: 'published',
    teacherName: '',
    teacherTitle: '',
    coverUrl: '',
  }
}

function pickFolder() {
  const input = folderInput.value
  if (!input) return
  // Vue renders the non-standard attributes, but setting them imperatively
  // keeps folder selection working on browsers that only honour the
  // prefixed/legacy spellings.
  input.setAttribute('webkitdirectory', '')
  input.setAttribute('directory', '')
  input.value = ''
  input.click()
}

function onFolderPicked(event: Event) {
  const input = event.target as HTMLInputElement
  const fileList = input.files
  if (!fileList || !fileList.length) return

  const items: PreviewItem[] = []
  for (let index = 0; index < fileList.length; index += 1) {
    const file = fileList[index]
    const relativePath = (file as File & { webkitRelativePath?: string }).webkitRelativePath || file.name
    const name = relativePath.split('/').pop() || file.name
    items.push({
      file,
      relativePath,
      name,
      kind: kindOf(name),
      skipReason: skipReasonFor(relativePath, name),
    })
  }

  pickedFiles.value = sortPreviewItems(items)
  const first = pickedFiles.value.find((item) => item.relativePath.includes('/'))
  pickedFolderName.value = first ? first.relativePath.split('/')[0] : ''
  if (!metaForm.value.title && pickedFolderName.value) {
    metaForm.value.title = pickedFolderName.value
  }

  if (!usableFiles.value.length) {
    MessagePlugin.warning('这个文件夹里没有可用的讲课文件，请换一个文件夹')
  }
}

/**
 * Mirrors the backend ordering rule: files whose name starts with a number are
 * ordered by that number first, everything else follows in name order. The
 * preview must match what will actually be published, so if this rule changes
 * the Go side has to change with it.
 */
function sortPreviewItems(items: PreviewItem[]) {
  return [...items].sort((left, right) => {
    const leftRank = rankOf(left.name)
    const rightRank = rankOf(right.name)
    if ((leftRank !== null) !== (rightRank !== null)) return leftRank !== null ? -1 : 1
    if (leftRank !== null && rightRank !== null && leftRank !== rightRank) return leftRank - rightRank
    const leftName = left.name.toLowerCase()
    const rightName = right.name.toLowerCase()
    if (leftName !== rightName) return leftName < rightName ? -1 : 1
    return 0
  })
}

function rankOf(name: string) {
  const match = /^(\d{1,4})([_\-.\s]|$)/.exec(name)
  return match ? Number(match[1]) : null
}

function kindOf(name: string): CourseLessonType {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  if (VIDEO_EXTS.includes(ext)) return 'video'
  if (AUDIO_EXTS.includes(ext)) return 'audio'
  return 'article'
}

function skipReasonFor(relativePath: string, name: string) {
  if (name.startsWith('.')) return '隐藏文件'
  const lower = relativePath.toLowerCase()
  if (lower.startsWith('__macosx/') || lower.includes('/__macosx/')) return '系统文件'
  return ''
}

function goToMeta() {
  if (!usableFiles.value.length) {
    MessagePlugin.warning('至少需要一个可用的讲课文件')
    return
  }
  step.value = 2
}

async function submit() {
  if (!metaForm.value.title.trim()) {
    MessagePlugin.warning('请填写课程名称')
    return
  }
  uploading.value = true
  uploadPercent.value = 0
  try {
    const response = await uploadAdminCourse(
      usableFiles.value.map((item) => ({ file: item.file, relativePath: item.relativePath })),
      {
        title: metaForm.value.title.trim(),
        summary: metaForm.value.summary.trim(),
        category: metaForm.value.category,
        coverUrl: metaForm.value.coverUrl.trim(),
        teacherName: metaForm.value.teacherName.trim(),
        teacherTitle: metaForm.value.teacherTitle.trim(),
        source: 'official',
        publicStatus: metaForm.value.publicStatus,
        folder: pickedFolderName.value,
      },
      (event: any) => {
        if (event?.total) {
          uploadPercent.value = Math.min(99, Math.round((event.loaded / event.total) * 100))
        }
      },
    )
    result.value = response.data || null
    uploadPercent.value = 100
    step.value = 3
    MessagePlugin.success('课程创建成功')
  } catch (error: any) {
    MessagePlugin.error(error?.message || '上传建课失败')
  } finally {
    uploading.value = false
  }
}

onMounted(() => {
  void loadCourses()
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
.course-progress { margin-top: 8px; display: flex; flex-direction: column; gap: 8px; }
.course-progress span { color: var(--td-text-color-secondary); font-size: 12px; }

.course-done { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 26px 20px 18px; text-align: center; }
.course-done__icon { font-size: 40px; color: var(--td-success-color); }
.course-done strong { color: var(--td-text-color-primary); font-size: 16px; }
.course-done span { color: var(--td-text-color-secondary); font-size: 13px; }
.course-done__skipped { margin-top: 8px; }
.course-done__skipped-title { margin-bottom: 8px; color: var(--td-text-color-secondary); font-size: 13px; }

.course-wizard-footer { display: flex; align-items: center; gap: 8px; margin-top: 22px; padding-top: 16px; border-top: 1px solid var(--td-component-border); }
.course-wizard-footer__spacer { flex: 1; }

.course-detail__head { display: flex; align-items: center; gap: 8px; }
.course-detail__head strong { color: var(--td-text-color-primary); font-size: 16px; }
.course-detail__summary { margin: 10px 0; color: var(--td-text-color-secondary); line-height: 1.7; }
.course-detail__meta { display: flex; gap: 14px; flex-wrap: wrap; color: var(--td-text-color-placeholder); font-size: 12px; margin-bottom: 16px; }
.course-detail__lessons-title { margin-bottom: 8px; color: var(--td-text-color-secondary); font-size: 13px; }
.course-detail__lessons .course-preview-row { border: 1px solid var(--td-component-border); border-bottom: 0; }
.course-detail__lessons .course-preview-row:last-child { border-bottom: 1px solid var(--td-component-border); }

@media (max-width: 900px) {
  .course-page__header { flex-direction: column; }
  .course-summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .course-toolbar { flex-wrap: wrap; }
  .course-row { align-items: flex-start; flex-wrap: wrap; }
  .course-row__actions { width: 100%; justify-content: flex-start; padding-left: 52px; }
  .course-form-grid { grid-template-columns: minmax(0, 1fr); }
}
</style>
