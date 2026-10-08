import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const dialogSource = readFileSync(
  new URL('./components/UploadConfirmDialog.vue', import.meta.url),
  'utf8',
);
const pageSource = readFileSync(new URL('./KnowledgeBase.vue', import.meta.url), 'utf8');

test('upload confirmation exposes a default-on parsing switch and upload-only result', () => {
  assert.match(dialogSource, /const parseEnabled = ref\(true\)/);
  assert.match(dialogSource, /<t-switch v-model="parseEnabled"/);
  assert.match(dialogSource, /parseEnabled: parseEnabled\.value/);
  assert.match(dialogSource, /confirmUploadOnly/);
});

test('file upload sends parse_enabled and omits process config when parsing is disabled', () => {
  assert.match(pageSource, /uploadData\.parse_enabled = options\.parseEnabled \?\? true/);
  assert.match(pageSource, /if \(uploadData\.parse_enabled && options\.processConfig\)/);
});
