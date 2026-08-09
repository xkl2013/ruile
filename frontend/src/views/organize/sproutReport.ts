import { marked } from 'marked'

import { sanitizeHTML, sanitizeMarkdownHTML, safeMarkdownToHTML } from '@/utils/security'

export interface SproutReportPreviewSection {
  number: string
  title: string
  seed: string
  body: string
  aha: string
}

export interface SproutReportPreview {
  intro: string
  sections: SproutReportPreviewSection[]
}

const GENERIC_REPORT_TITLES = new Set(['发芽报告', '发芽记录'])
let markedConfigured = false

const configureMarked = () => {
  if (markedConfigured) return
  marked.use({ gfm: true, breaks: true })
  markedConfigured = true
}

const decodeBasicEntities = (value: string) => value
  .replace(/&#39;/g, "'")
  .replace(/&#x27;/gi, "'")
  .replace(/&apos;/g, "'")
  .replace(/&#34;/g, '"')
  .replace(/&#x22;/gi, '"')
  .replace(/&quot;/g, '"')
  .replace(/&lt;/g, '<')
  .replace(/&gt;/g, '>')
  .replace(/&amp;/g, '&')

const htmlToReadableText = (value: string) => {
  if (!/<[a-z][\s\S]*>/i.test(value)) return value
  return decodeBasicEntities(value)
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/(p|div|section|article|li|blockquote)>/gi, '\n')
    .replace(/<h([1-6])[^>]*>/gi, (_match, level) => `\n${'#'.repeat(Number(level))} `)
    .replace(/<\/h[1-6]>/gi, '\n')
    .replace(/<[^>]+>/g, '')
}

const stripMarkdownMarkers = (value: string) => decodeBasicEntities(value)
  .replace(/!\[[^\]]*]\([^)]*\)/g, '')
  .replace(/\[([^\]]+)]\([^)]*\)/g, '$1')
  .replace(/[*_`~]/g, '')
  .replace(/^>+\s*/gm, '')
  .replace(/^#{1,6}\s*/gm, '')
  .replace(/^\s*[-*+]\s+/gm, '')
  .replace(/\s+/g, ' ')
  .trim()

const isHtmlDocument = (value: string) => /<\/?(h[1-6]|p|ul|ol|li|blockquote|div|table|article|section|br)\b/i.test(value)

const isMarkdownLike = (value: string) => {
  const text = htmlToReadableText(value)
  return /^#{1,6}\s+\S/m.test(text)
    || /^>\s+\S/m.test(text)
    || /^\s*[-*+]\s+\S/m.test(text)
    || /^\s*\d+[.、]\s+\S/m.test(text)
    || /\*\*[^*]+\*\*/.test(text)
}

export const normalizeSproutMarkdownSource = (value = '') => {
  const readable = htmlToReadableText(value)
    .replace(/\r\n?/g, '\n')
    .replace(/^\s*---\s*\n[\s\S]*?\n---\s*/, '')
    .replace(/[ \t]+---[ \t]+(?=#{1,6}\s*\d{1,2}[.、])/g, '\n\n')
    .replace(/[ \t]+(?=#{2,6}\s*\d{1,2}[.、]\s+)/g, '\n\n')
    .replace(/[ \t]+(?=>\s*\*\*(?:🌱|✨|Aha|种子|瞬间))/g, '\n')
    .replace(/^\s*---+\s*$/gm, '')
    .replace(/\n{3,}/g, '\n\n')
    .trim()
  return decodeBasicEntities(readable)
}

export const renderSproutReportHtml = (value = '') => {
  const source = value.trim()
  if (!source) return ''

  if (isHtmlDocument(source) && !isMarkdownLike(source)) {
    return sanitizeHTML(source)
  }

  configureMarked()
  const markdown = normalizeSproutMarkdownSource(source)
  const html = marked.parse(safeMarkdownToHTML(markdown), { gfm: true, breaks: true, async: false }) as string
  return sanitizeMarkdownHTML(html)
}

export const sproutReportContentForEditor = (value = '') => stripSproutReportHeading(renderSproutReportHtml(value))

const isSeedLabel = (line: string) => /^(?:🌱\s*)?种子[:：]?$/.test(stripMarkdownMarkers(line))
const isAhaLabel = (line: string) => /^(?:✨\s*)?(?:Aha\s*)?瞬间[:：]?$|^Aha\s*瞬间[:：]?$/i.test(stripMarkdownMarkers(line))

const cleanPreviewLine = (line: string) => stripMarkdownMarkers(line.replace(/^\s*>+\s?/, ''))

const compactText = (lines: string[], maxLength: number) => {
  const text = stripMarkdownMarkers(lines.join(' '))
  return text.length > maxLength ? `${text.slice(0, maxLength)}...` : text
}

const removeLeadingReportChrome = (lines: string[], title: string) => {
  const normalizedTitle = stripMarkdownMarkers(title)
  let started = false
  return lines.filter((line) => {
    const clean = cleanPreviewLine(line)
    if (!clean) return started
    if (!started && (GENERIC_REPORT_TITLES.has(clean) || clean === normalizedTitle)) return false
    if (!started && /^\d{1,2}\s*月\s*\d{1,2}\s*日\s*生成$/.test(clean)) return false
    started = true
    return true
  })
}

const parseSection = (number: string, title: string, body: string): SproutReportPreviewSection => {
  const lines = body.split('\n')
  const seedLines: string[] = []
  const ahaLines: string[] = []
  const bodyLines: string[] = []
  let bucket: 'seed' | 'aha' | 'body' = 'body'

  lines.forEach((line) => {
    const clean = cleanPreviewLine(line)
    if (!clean) return
    if (isSeedLabel(line)) {
      bucket = 'seed'
      return
    }
    if (isAhaLabel(line)) {
      bucket = 'aha'
      return
    }
    if (bucket === 'seed') seedLines.push(clean)
    else if (bucket === 'aha') ahaLines.push(clean)
    else bodyLines.push(clean)
  })

  return {
    number: number.padStart(2, '0'),
    title: cleanPreviewLine(title),
    seed: compactText(seedLines, 88),
    body: compactText(bodyLines, 110),
    aha: compactText(ahaLines, 88),
  }
}

export const buildSproutReportPreview = (value = '', title = ''): SproutReportPreview => {
  const markdown = normalizeSproutMarkdownSource(value)
  const lines = removeLeadingReportChrome(markdown.split('\n'), title)
  const normalized = lines.join('\n')
  const matches = Array.from(normalized.matchAll(/^#{2,6}\s*(\d{1,2})[.、]\s*(.+?)\s*$/gm))
  const introEnd = matches[0]?.index ?? normalized.length
  const intro = compactText(normalized.slice(0, introEnd).split('\n'), 150)

  const sections = matches.slice(0, 5).map((match, index) => {
    const bodyStart = (match.index || 0) + match[0].length
    const bodyEnd = matches[index + 1]?.index ?? normalized.length
    return parseSection(match[1], match[2], normalized.slice(bodyStart, bodyEnd))
  }).filter((section) => section.title)

  if (intro || sections.length) {
    return { intro, sections }
  }

  return {
    intro: compactText(markdown.split('\n'), 150),
    sections: [],
  }
}

export const stripSproutReportHeading = (html = '', title = '') => {
  const source = html.trim()
  if (!source) return source
  const normalizedTitle = stripMarkdownMarkers(title)
  const match = source.match(/^<h1[^>]*>([\s\S]*?)<\/h1>/i)
  if (!match) return source
  const heading = stripMarkdownMarkers(match[1])
  if (GENERIC_REPORT_TITLES.has(heading) || heading === normalizedTitle) {
    return source.slice(match[0].length).trim()
  }
  return source
}
