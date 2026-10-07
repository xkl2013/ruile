import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./document-preview.vue', import.meta.url), 'utf8');

test('reading mode removes the DOCX frame and uses the available preview width', () => {
  assert.ok(source.includes('readingMode?: boolean'));
  assert.ok(source.includes("'reading-mode': readingMode"));
  assert.ok(source.includes(':deep(.docx-preview-wrapper-wrapper)'));
  assert.ok(source.includes('padding: 0 !important'));
  assert.ok(source.includes('background: transparent !important'));
  assert.ok(source.includes(':deep(.docx-preview-wrapper-wrapper > section.docx-preview-wrapper)'));
  assert.ok(source.includes('padding: 32px 40px 40px !important'));
  assert.ok(source.includes('box-shadow: none !important'));
});
