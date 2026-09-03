export type DiscoverCategoryKey =
  | 'admissions_growth'
  | 'parent_service'
  | 'event_planning'
  | 'kindergarten_operations'
  | 'team_leadership'
  | 'nutrition_food_education'
  | 'space_environment'
  | 'teacher_research'

export interface DiscoverCategory {
  key: DiscoverCategoryKey
  label: string
}

// Fixed first-version content columns from the content production manual.
export const DISCOVER_CATEGORIES: readonly DiscoverCategory[] = [
  { key: 'admissions_growth', label: '招生增长' },
  { key: 'parent_service', label: '家长服务' },
  { key: 'event_planning', label: '活动策划' },
  { key: 'kindergarten_operations', label: '园所运营' },
  { key: 'team_leadership', label: '团队运营/领导力' },
  { key: 'nutrition_food_education', label: '儿童营养与食育' },
  { key: 'space_environment', label: '空间设计 / 环创' },
  { key: 'teacher_research', label: '教师成长 / 教研' },
]

export const DISCOVER_CATEGORY_OPTIONS = DISCOVER_CATEGORIES.map((category) => ({
  label: category.label,
  value: category.key,
}))

const categoryAliases: Record<string, DiscoverCategoryKey> = {
  招生增长: 'admissions_growth',
  家长服务: 'parent_service',
  活动策划: 'event_planning',
  园所运营: 'kindergarten_operations',
  '团队运营/领导力': 'team_leadership',
  团队运营与园长领导力: 'team_leadership',
  '儿童营养与食育': 'nutrition_food_education',
  '空间设计 / 环创': 'space_environment',
  空间设计: 'space_environment',
  '教师成长 / 教研': 'teacher_research',
  教师成长: 'teacher_research',
}

export const normalizeDiscoverCategory = (value: unknown): DiscoverCategoryKey | '' => {
  const normalized = typeof value === 'string' ? value.trim() : ''
  if (!normalized) return ''
  if (DISCOVER_CATEGORIES.some((category) => category.key === normalized)) {
    return normalized as DiscoverCategoryKey
  }
  return categoryAliases[normalized] || ''
}

export const discoverCategoryLabel = (value: unknown) => {
  const key = normalizeDiscoverCategory(value)
  return DISCOVER_CATEGORIES.find((category) => category.key === key)?.label || ''
}
