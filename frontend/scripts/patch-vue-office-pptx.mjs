import { readdir, readFile, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const SMART_ART_MARKER = 'smart-chart-diagram';
const SMART_ART_WINDOW_SIZE = 1600;
const APPEND_CHILD_PATTERN = /([A-Za-z_$][\w$]*)\.appendChild\(([A-Za-z_$][\w$]*)\)/;

export function patchVueOfficePptxBundle(source) {
  const markerIndex = source.indexOf(SMART_ART_MARKER);
  if (markerIndex < 0) {
    return { source, status: 'not-applicable' };
  }

  const windowEnd = Math.min(source.length, markerIndex + SMART_ART_WINDOW_SIZE);
  const smartArtBlock = source.slice(markerIndex, windowEnd);
  const appendMatch = smartArtBlock.match(APPEND_CHILD_PATTERN);
  if (!appendMatch || appendMatch.index === undefined) {
    return { source, status: 'missing-target' };
  }

  const [appendExpression, parentName, childName] = appendMatch;
  const guardedExpression = `${childName}&&${appendExpression}`;
  if (smartArtBlock.includes(guardedExpression)) {
    return { source, status: 'already-patched' };
  }

  const appendIndex = markerIndex + appendMatch.index;
  const patchedSource = [
    source.slice(0, appendIndex),
    guardedExpression,
    source.slice(appendIndex + appendExpression.length),
  ].join('');
  return { source: patchedSource, status: 'patched' };
}

async function listBundleFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true });
  const files = [];

  for (const entry of entries) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      files.push(...await listBundleFiles(entryPath));
      continue;
    }
    if (entry.isFile() && /\.(?:m?js)$/.test(entry.name)) {
      files.push(entryPath);
    }
  }
  return files;
}

export async function patchInstalledVueOfficePptx() {
  const require = createRequire(import.meta.url);
  const packageJsonPath = require.resolve('@vue-office/pptx/package.json');
  const packageDirectory = path.dirname(packageJsonPath);
  const bundleFiles = await listBundleFiles(path.join(packageDirectory, 'lib'));

  let patchedCount = 0;
  let alreadyPatchedCount = 0;
  for (const bundleFile of bundleFiles) {
    const source = await readFile(bundleFile, 'utf8');
    const result = patchVueOfficePptxBundle(source);
    if (result.status === 'patched') {
      await writeFile(bundleFile, result.source);
      patchedCount += 1;
    } else if (result.status === 'already-patched') {
      alreadyPatchedCount += 1;
    }
  }

  if (patchedCount === 0 && alreadyPatchedCount === 0) {
    throw new Error('Unable to locate the @vue-office/pptx SmartArt appendChild target');
  }

  return { patchedCount, alreadyPatchedCount };
}

const invokedDirectly = process.argv[1]
  && import.meta.url === pathToFileURL(fileURLToPath(pathToFileURL(process.argv[1]))).href;

if (invokedDirectly) {
  const result = await patchInstalledVueOfficePptx();
  console.log(
    `[postinstall] @vue-office/pptx SmartArt patch: ${result.patchedCount} patched, `
      + `${result.alreadyPatchedCount} already patched`,
  );
}
