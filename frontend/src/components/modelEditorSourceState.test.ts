import assert from 'node:assert/strict'
import test from 'node:test'

import { shouldShowOllamaUnavailableTip } from './modelEditorSourceState.ts'

test('hides Ollama unavailable tip while configuring a remote model', () => {
  assert.equal(shouldShowOllamaUnavailableTip('remote', 'chat', false), false)
})

test('shows Ollama unavailable tip only for local Ollama-backed models', () => {
  assert.equal(shouldShowOllamaUnavailableTip('local', 'chat', false), true)
  assert.equal(shouldShowOllamaUnavailableTip('local', 'embedding', false), true)
  assert.equal(shouldShowOllamaUnavailableTip('local', 'rerank', false), false)
  assert.equal(shouldShowOllamaUnavailableTip('local', 'asr', false), false)
})

test('does not show Ollama unavailable tip before status is known or when Ollama is available', () => {
  assert.equal(shouldShowOllamaUnavailableTip('local', 'chat', null), false)
  assert.equal(shouldShowOllamaUnavailableTip('local', 'chat', true), false)
})
