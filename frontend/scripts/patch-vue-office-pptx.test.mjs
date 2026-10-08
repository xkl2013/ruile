import assert from 'node:assert/strict';
import test from 'node:test';

import { patchVueOfficePptxBundle } from './patch-vue-office-pptx.mjs';

test('guards unsupported SmartArt children in readable bundles', () => {
  const source = `
    node.className = "smart-chart-diagram";
    child instanceof Shape ? rendered = renderShape(child) : child instanceof Text && (rendered = renderText(child));
    node.appendChild(rendered);
  `;

  const result = patchVueOfficePptxBundle(source);

  assert.equal(result.status, 'patched');
  assert.match(result.source, /rendered&&node\.appendChild\(rendered\)/);
});

test('guards unsupported SmartArt children in minified bundles', () => {
  const source = 'n.className="smart-chart-diagram",y instanceof A?_=B(y):y instanceof C&&(_=D(y)),n.appendChild(_)}return n';

  const result = patchVueOfficePptxBundle(source);

  assert.equal(result.status, 'patched');
  assert.match(result.source, /_&&n\.appendChild\(_\)/);
});

test('is idempotent', () => {
  const source = 'n.className="smart-chart-diagram",y instanceof A?_=B(y):y instanceof C&&(_=D(y)),_&&n.appendChild(_)}return n';

  const result = patchVueOfficePptxBundle(source);

  assert.equal(result.status, 'already-patched');
  assert.equal(result.source, source);
});
