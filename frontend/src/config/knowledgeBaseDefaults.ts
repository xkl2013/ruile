export type KnowledgeBaseCreationType = 'document' | 'faq'
export type WikiExtractionGranularity = 'focused' | 'standard' | 'exhaustive'

export const KNOWLEDGE_BASE_ICON_OPTIONS = [
  'folder',
  'chat-bubble-help',
  'bookmark-add',
  'file-copy',
  'file-search',
  'data-base',
  'control-platform',
  'cloud',
  'share',
  'relation',
  'help-circle',
  'setting',
  'view-module',
  'file',
  'image',
  'sound',
] as const

const DEFAULT_KB_ICON_BY_TYPE: Record<KnowledgeBaseCreationType, string> = {
  document: 'folder',
  faq: 'chat-bubble-help',
}

export const getDefaultKnowledgeBaseIcon = (
  type: KnowledgeBaseCreationType = 'document',
) => DEFAULT_KB_ICON_BY_TYPE[type] || DEFAULT_KB_ICON_BY_TYPE.document

export const DEFAULT_KB_CHUNKING_PRESET = {
  chunkSize: 512,
  chunkOverlap: 80,
  enableParentChild: true,
} as const

export const DEFAULT_KB_PARSER_ENGINE_RULES = [
  { file_types: ['pptx', 'ppt'], engine: 'markitdown' },
] as const

export const cloneDefaultParserEngineRules = () =>
  DEFAULT_KB_PARSER_ENGINE_RULES.map((rule) => ({
    file_types: [...rule.file_types],
    engine: rule.engine,
  }))

export const WIKI_ONLY_KB_CHUNKING_PRESET = {
  chunkSize: 2048,
  chunkOverlap: 0,
  enableParentChild: false,
} as const

const DEFAULT_KB_SEPARATORS = ['\n\n', '\n', '。', '！', '？', ';', '；'] as const

export const createDefaultKnowledgeBaseFormData = (
  type: KnowledgeBaseCreationType = 'document',
) => ({
  type,
  icon: getDefaultKnowledgeBaseIcon(type),
  iconUrl: '',
  name: '',
  description: '',
  faqConfig: {
    indexMode: 'question_only',
    questionIndexMode: 'separate',
  },
  modelConfig: {
    llmModelId: '',
    embeddingModelId: '',
    wikiSynthesisModelId: '',
  },
  chunkingConfig: {
    ...DEFAULT_KB_CHUNKING_PRESET,
    separators: [...DEFAULT_KB_SEPARATORS],
    parserEngineRules: cloneDefaultParserEngineRules(),
    parentChunkSize: 4096,
    childChunkSize: 384,
    strategy: 'auto' as string,
    tokenLimit: 0,
    languages: [] as string[],
    tableMetadataInstructions: '',
  },
  storageBackendId: '' as string,
  storageProvider: '' as string,
  multimodalConfig: {
    enabled: false,
    vllmModelId: '',
    descriptionLanguage: '',
    customInstructions: '',
  },
  ocrConfig: {
    enabled: false,
    modelId: '',
  },
  asrConfig: {
    enabled: false,
    modelId: '',
    language: '',
  },
  nodeExtractConfig: {
    enabled: false,
    text: '',
    tags: [] as string[],
    nodes: [] as Array<{
      name: string
      attributes: string[]
    }>,
    relations: [] as Array<{
      node1: string
      node2: string
      type: string
    }>,
    customInstructions: '',
  },
  questionGenerationConfig: {
    enabled: true,
    questionCount: 3,
    customInstructions: '',
  },
  wikiConfig: {
    synthesisModelId: '',
    maxPagesPerIngest: 0,
    extractionGranularity: 'standard' as WikiExtractionGranularity,
    contentInstructions: '',
    extractionInstructions: '',
  },
  indexingStrategy: {
    vectorEnabled: true,
    keywordEnabled: true,
    wikiEnabled: false,
    graphEnabled: false,
  },
  vectorStoreId: '' as string,
  vectorStoreInfo: {
    source: undefined as string | undefined,
    name: undefined as string | undefined,
    engineType: undefined as string | undefined,
    status: undefined as string | undefined,
  },
})
