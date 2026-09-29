import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const component = readFileSync(new URL('./ModelSettings.vue', import.meta.url), 'utf8')

test('model pricing dialog loads the existing token price version', () => {
  assert.ok(component.includes('function nanoCNYToPrice'))
  assert.ok(component.includes('const existingPrice = modelPriceFor(model)'))
  assert.ok(component.includes('existingPrice?.pricing_mode === \'token\''))
  assert.ok(component.includes('priceDraft.inputCNY = nanoCNYToPrice(existingPrice.input_nanousd_per_m_tokens)'))
  assert.ok(component.includes('priceDraft.outputCNY = nanoCNYToPrice(existingPrice.output_nanousd_per_m_tokens)'))
})
