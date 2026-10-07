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

/** 正文里的 [M1] 来源标记 → 可点击小徽章（样式见 OrganizeMarkdownRenderer）。 */
const CITE_BADGE_PATTERN = /\[(M\d+)\](?!\()/g

const withCiteBadges = (markdown: string) =>
  markdown.replace(CITE_BADGE_PATTERN, (_match, id: string) =>
    `<span class="organize-cite-ref" data-cite="${id}">${id}</span>`,
  )

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
