import type {
  ServiceFact,
  ServiceReminder,
  ServiceSpaceProfile,
  ServiceSubject,
} from '@/api/service'

export type ServiceCustomerInfo = {
  familyName: string
  student: string
  className: string
  teacher: string
}

export type ServiceActivityItem = {
  id: string
  kind: 'fact' | 'reminder'
  title: string
  detail: string
  source: string
  createdAt?: string
}

const textValue = (value: unknown) => {
  if (value === undefined || value === null || value === '') return ''
  if (Array.isArray(value)) return value.filter(Boolean).join('、')
  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    return String(record.value || record.content || record.text || '')
  }
  return String(value)
}

const pickValue = (
  values: Record<string, unknown> | undefined,
  keys: string[],
) => {
  if (!values) return ''
  for (const key of keys) {
    const value = textValue(values[key])
    if (value) return value
  }
  return ''
}

export const buildCustomerInfo = (
  subject?: ServiceSubject | null,
  profile?: ServiceSpaceProfile | null,
  fallbackName = '当前家庭',
): ServiceCustomerInfo => {
  const metadata = subject?.metadata || {}
  const values = profile?.values || {}
  const name = (keyList: string[]) => pickValue(values, keyList) || pickValue(metadata, keyList)

  return {
    familyName: subject?.display_name || name(['family_name', 'family', 'member_family']) || fallbackName,
    student: name(['student_name', 'child_name', 'learner_name', 'child', 'student', 'child_stage']) || '待补充',
    className: name(['class_name', 'class', 'course_class']) || '待补充',
    teacher: name(['teacher_name', 'homeroom_teacher', 'teacher', 'owner']) || '待补充',
  }
}

const formatFactValue = (fact: ServiceFact) =>
  textValue(fact.value?.value)
  || textValue(fact.value?.content)
  || textValue(fact.value?.text)
  || textValue(fact.value)
  || '已更新'

export const factToActivity = (fact: ServiceFact): ServiceActivityItem => ({
  id: `fact-${fact.id}`,
  kind: 'fact',
  title: fact.fact_key || '重点记录',
  detail: formatFactValue(fact),
  source: fact.source_type === 'organize_output' ? '整理记录' : '服务会话',
  createdAt: fact.created_at,
})

export const reminderToActivity = (reminder: ServiceReminder): ServiceActivityItem => ({
  id: `reminder-${reminder.id}`,
  kind: 'reminder',
  title: `待办：${reminder.title}`,
  detail: reminder.status || '待处理',
  source: '服务待办',
  createdAt: reminder.updated_at || reminder.created_at,
})

export const sortActivities = (items: ServiceActivityItem[]) =>
  [...items].sort((a, b) => {
    const left = a.createdAt ? Date.parse(a.createdAt) : 0
    const right = b.createdAt ? Date.parse(b.createdAt) : 0
    return right - left
  })

export const displayActivityDate = (value?: string) => {
  if (!value) return '刚刚'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '刚刚'
  return new Intl.DateTimeFormat('zh-CN', {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date)
}

export const displayValue = (value: unknown) => textValue(value) || '待补充'

