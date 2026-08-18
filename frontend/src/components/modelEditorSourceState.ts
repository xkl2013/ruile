export type ModelEditorSource = 'local' | 'remote'

export type ModelEditorType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'ocr' | 'asr'

export function shouldShowOllamaUnavailableTip(
  source: ModelEditorSource,
  modelType: ModelEditorType,
  ollamaServiceStatus: boolean | null,
): boolean {
  return source === 'local' && modelType !== 'rerank' && modelType !== 'ocr' && modelType !== 'asr' && ollamaServiceStatus === false
}
