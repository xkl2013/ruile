import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const source = readFileSync(new URL('./OrganizeDocumentEditor.vue', import.meta.url), 'utf8')

test('document editor opens slash menu after leading whitespace', () => {
  assert.match(source, /@keydown\.capture="handleEditorKeydown"/)
  assert.ok(source.includes("event.key !== '/'"))
  assert.match(source, /\/\^\\s\+\$\/\.test\(textBeforeCursor\)/)
  assert.ok(source.includes("deleteRange({ from: $from.start(), to: $from.pos }).insertContent('/')"))
})

test('memory note editor exposes tag, font, color, and sprout controls', () => {
  assert.ok(source.includes('isNoteMemory'))
  assert.ok(source.includes('memory-note-tabs'))
  assert.ok(source.includes('+ 智能标签'))
  assert.ok(source.includes('createMemorySprout'))
  assert.ok(source.includes(':version="editorVersion"'))
})
