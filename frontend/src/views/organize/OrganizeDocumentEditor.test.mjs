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
