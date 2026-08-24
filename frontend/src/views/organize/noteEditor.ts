const NOTE_TAG_STOP_WORDS = new Set([
  '我们', '你们', '他们', '她们', '它们', '自己', '这个', '那个', '这些', '那些',
  '一个', '一些', '可以', '因为', '所以', '然后', '如果', '但是', '而且', '已经',
  '正在', '还是', '还是说', '没有', '不是', '就是', '以及', '关于', '为了', '同时',
  '时候', '今天', '明天', '昨天', '内容', '记录', '笔记', '随笔', '标签', '智能',
  '发芽', '追加', '编辑', '页面', '详情', '正文', '标题', '事情', '感觉', '问题',
  '需要', '可能', '应该', '我们要', '自己会', '一直', '一次', '一种', '很多', '一些',
  '以及', '与', '和', '或', '在', '是', '有', '了', '就', '都', '也', '而',
  'of', 'the', 'and', 'for', 'with', 'this', 'that', 'from', 'into', 'your', 'you',
  'are', 'was', 'were', 'will', 'can', 'to', 'in', 'on', 'at', 'by', 'an', 'a',
])

const normalizeTagText = (value = '') => value
  .replace(/\s+/g, ' ')
  .trim()
  .replace(/^#+/g, '')
  .replace(/^@+/g, '')
  .trim()

const dedupeTags = (items: string[]) => {
  const seen = new Set<string>()
  const result: string[] = []
  for (const item of items) {
    const tag = normalizeTagText(item)
    if (!tag) continue
    const key = tag.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    result.push(tag)
  }
  return result
}

export const normalizeNoteTags = (value: unknown) => {
  if (Array.isArray(value)) {
    return dedupeTags(value.map((item) => String(item)))
  }
  if (typeof value === 'string') {
    return dedupeTags(value.split(/[，,;；\n]/g))
  }
  return []
}

const splitTextSegments = (value: string) => value
  .replace(/<[^>]+>/g, ' ')
  .replace(/&[a-z]+;/gi, ' ')
  .replace(/[^\p{L}\p{N}\u4e00-\u9fff]+/gu, ' ')
  .split(/\s+/)
  .map((item) => item.trim())
  .filter(Boolean)

const collectTextCandidates = (value: string, minLength: number, maxLength: number) => {
  const candidates: string[] = []
  for (const segment of splitTextSegments(value)) {
    if (/^[A-Za-z0-9]+$/.test(segment)) {
      if (segment.length >= minLength) candidates.push(segment)
      continue
    }

    if (!/^[\u4e00-\u9fff]+$/.test(segment)) continue

    const upper = Math.min(maxLength, segment.length)
    const lower = Math.min(minLength, upper)
    for (let len = lower; len <= upper; len += 1) {
      for (let i = 0; i + len <= segment.length; i += 1) {
        candidates.push(segment.slice(i, i + len))
      }
    }
  }
  return candidates
}

const isSmartTagCandidate = (value: string) => {
  const tag = normalizeTagText(value)
  if (!tag) return false
  if (/^\d+$/.test(tag)) return false
  if (tag.length > 12) return false

  const lower = tag.toLowerCase()
  if (NOTE_TAG_STOP_WORDS.has(lower)) return false

  for (const stopWord of NOTE_TAG_STOP_WORDS) {
    if (lower.includes(stopWord)) return false
  }

  return true
}

export const buildSmartNoteTags = (
  title: string,
  text: string,
  existingTags: string[] = [],
) => {
  const counts = new Map<string, number>()
  const push = (value: string, weight = 1) => {
    const tag = normalizeTagText(value)
    if (!isSmartTagCandidate(tag)) return
    counts.set(tag, (counts.get(tag) || 0) + weight)
  }

  collectTextCandidates(title, 2, 5).forEach((candidate) => push(candidate, 2))
  collectTextCandidates(text, 3, 5).forEach((candidate) => push(candidate, 1))

  const ordered = Array.from(counts.entries())
    .sort((left, right) => {
      if (right[1] !== left[1]) return right[1] - left[1]
      if (right[0].length !== left[0].length) return right[0].length - left[0].length
      return left[0].localeCompare(right[0], 'zh-Hans-CN')
    })
    .map(([tag]) => tag)

  const existing = new Set(normalizeNoteTags(existingTags).map((tag) => tag.toLowerCase()))
  return dedupeTags(ordered.filter((tag) => !existing.has(tag.toLowerCase()))).slice(0, 6)
}

export const mergeNoteMetadata = (metadata: Record<string, unknown> | undefined, tags: string[]) => {
  const next = { ...(metadata || {}) }
  next.tags = normalizeNoteTags(tags)
  return next
}
