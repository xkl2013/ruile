import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./doc-content.vue', import.meta.url), 'utf8');
const supportedTypesBlock = source.match(
  /const previewSupportedTypes = new Set\(\[([\s\S]*?)\]\);/,
)?.[1] || '';

test('knowledge document preview exposes supported video file types', () => {
  for (const extension of ['mp4', 'mov', 'avi', 'mkv', 'webm', 'wmv', 'flv']) {
    assert.match(
      supportedTypesBlock,
      new RegExp(`['"]${extension}['"]`),
      `missing video extension: ${extension}`,
    );
  }
});
