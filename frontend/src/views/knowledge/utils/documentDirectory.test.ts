import assert from 'node:assert/strict';
import test from 'node:test';
import { getURLDisplayPath, getUploadDisplayFileName } from './documentDirectory';

test('places a single uploaded file in the selected directory', () => {
  assert.equal(
    getUploadDisplayFileName('report.pdf', '时代出生'),
    '时代出生/report.pdf',
  );
});

test('preserves a folder upload hierarchy beneath the selected directory', () => {
  assert.equal(
    getUploadDisplayFileName('report.pdf', '时代出生', 'local-folder/quarterly/report.pdf'),
    '时代出生/quarterly/report.pdf',
  );
});

test('does not use the source folder name for a top-level folder upload', () => {
  assert.equal(
    getUploadDisplayFileName('report.pdf', '', 'local-folder/report.pdf'),
    'report.pdf',
  );
});

test('places an imported web page in the selected directory', () => {
  assert.equal(
    getURLDisplayPath('https://example.com/guides/getting-started?from=menu', '时代出生'),
    '时代出生/getting-started',
  );
});

test('does not set a display path for a root-level web import', () => {
  assert.equal(getURLDisplayPath('https://example.com/', ''), '');
});
