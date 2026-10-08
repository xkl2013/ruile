import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const source = readFileSync(new URL('./document-preview.vue', import.meta.url), 'utf8');
const detailSource = readFileSync(new URL('./doc-content.vue', import.meta.url), 'utf8');

test('knowledge preview requests a signed storage URL instead of the server preview stream', () => {
  assert.match(source, /getKnowledgePreviewUrl\(props\.knowledgeId\)/);
  assert.match(source, /fetch\(url,\s*\{[\s\S]*credentials:\s*'omit'/);
  assert.match(source, /preview\.directStorageAccessFailed/);
  assert.doesNotMatch(source, /previewKnowledgeFile/);
});

test('native browser previews use the signed URL without creating a blob', () => {
  assert.match(source, /\['pdf', 'image', 'audio', 'video'\]\.includes/);
  assert.match(source, /blobUrl\.value = directUrl/);
});

test('the detail audio player also uses the signed storage URL', () => {
  assert.match(detailSource, /getKnowledgePreviewUrl\(props\.details\.id\)/);
  assert.doesNotMatch(detailSource, /previewKnowledgeFile/);
});
