import { marked } from 'marked'

import { sanitizeHTML, sanitizeMarkdownHTML, safeMarkdownToHTML } from '@/utils/security'

export type OrganizeMarkdownNormalizer = (value: string) => string

let markedConfigured = false

const configureMarked = () => {
  if (markedConfigured) return
  marked.use({ gfm: true, breaks: true })
  markedConfigured = true
}

export const decodeBasicEntities = (value: string) => value
  .replace(/&#39;/g, "'")
  .replace(/&#x27;/gi, "'")
  .replace(/&apos;/g, "'")
  .replace(/&#34;/g, '"')
  .replace(/&#x22;/gi, '"')
  .replace(/&quot;/g, '"')
  .replace(/&lt;/g, '<')
  .replace(/&gt;/g, '>')
  .replace(/&amp;/g, '&')

export const htmlToReadableText = (value: string) => {
  if (!/<[a-z][\s\S]*>/i.test(value)) return value
  return decodeBasicEntities(value)
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/(p|div|section|article|li|blockquote)>/gi, '\n')
    .replace(/<h([1-6])[^>]*>/gi, (_match, level) => `\n${'#'.repeat(Number(level))} `)
    .replace(/<\/h[1-6]>/gi, '\n')
    .replace(/<[^>]+>/g, '')
}

const isHtmlDocument = (value: string) => /<\/?(h[1-6]|p|ul|ol|li|blockquote|div|table|article|section|br)\b/i.test(value)

const isMarkdownLike = (value: string) => {
  const text = htmlToReadableText(value)
  return /^#{1,6}\s+\S/m.test(text)
    || /^>\s+\S/m.test(text)
    || /^\s*[-*+]\s+\S/m.test(text)
    || /^\s*\d+[.、]\s+\S/m.test(text)
    || /\*\*[^*]+\*\*/.test(text)
}

export interface RenderOrganizeMarkdownOptions {
  normalize?: OrganizeMarkdownNormalizer
}

/**
 * 正文里的来源标记 → 可点击小徽章（样式见 OrganizeMarkdownRenderer）。
 * 批次合并会产生 [M1-批次1] 这类带批次后缀的标记，同一句常连着出现好几个，
 * 这里统一归一成 [M1] 徽章并去掉重复，避免正文里留一串标记乱码。
 */
const CITE_RUN_PATTERN = /(?:\[M\d+(?:-[^\]\s]+)?\](?!\()[ \t]*){2,}/g
const CITE_PATTERN = /\[(M\d+)(?:-[^\]\s]+)?\](?!\()/g

const citeBadge = (id: string) =>
  `<span class="organize-cite-ref" data-cite="${id}">${id}</span>`

const withCiteBadges = (markdown: string) =>
  markdown
    .replace(CITE_RUN_PATTERN, (run) => {
      const ids = [...new Set([...run.matchAll(/\[(M\d+)/g)].map((match) => match[1]))]
      return ids.map(citeBadge).join('')
    })
    .replace(CITE_PATTERN, (_match, id: string) => citeBadge(id))

/**
 * Render model-produced Markdown while keeping legacy HTML output readable and safe.
 * The optional normalizer is reserved for legacy content with known formatting noise.
 */
export const renderOrganizeMarkdown = (
  value = '',
  options: RenderOrganizeMarkdownOptions = {},
) => {
  const source = value.trim()
  if (!source) return ''

  if (isHtmlDocument(source) && !isMarkdownLike(source)) {
    return sanitizeHTML(source)
  }

  configureMarked()
  const markdown = (options.normalize ? options.normalize(source) : source.replace(/\r\n?/g, '\n')).trim()
  const html = marked.parse(withCiteBadges(safeMarkdownToHTML(markdown)), {
    gfm: true,
    breaks: true,
    async: false,
  }) as string
  return sanitizeMarkdownHTML(html)
}
