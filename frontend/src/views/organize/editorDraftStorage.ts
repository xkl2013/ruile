import type {
  OrganizeMemoryKind,
  OrganizeOutputStatus,
  OrganizeSproutStage,
} from '@/api/organize'

export type OrganizeEditorDraftType = 'memory' | 'output' | 'sprout'

export interface OrganizeEditorDraft {
  title?: string
  content?: string
  kind?: OrganizeMemoryKind
  source?: string
  duration_seconds?: number
  output_type?: string
  source_summary?: string
  status?: OrganizeOutputStatus
  icon?: string
  stage?: OrganizeSproutStage
  output_hint?: string
  chips?: string[]
  memory_ids?: string[]
  metadata?: Record<string, unknown>
}

const STORAGE_PREFIX = 'weknora_organize_editor_draft:'

const canUseSessionStorage = () => {
  try {
    return typeof window !== 'undefined' && typeof window.sessionStorage !== 'undefined'
  } catch {
    return false
  }
}

const draftStorageKey = (documentType: OrganizeEditorDraftType, id: string) => {
  return `${STORAGE_PREFIX}${documentType}:${id}`
}

export const saveOrganizeEditorDraft = (
  documentType: OrganizeEditorDraftType,
  id: string,
  draft: OrganizeEditorDraft,
) => {
  if (!id || !canUseSessionStorage()) return
  try {
    window.sessionStorage.setItem(draftStorageKey(documentType, id), JSON.stringify(draft))
  } catch {
    // sessionStorage may be disabled or full; the editor can still open as a blank draft.
  }
}

export const readOrganizeEditorDraft = (
  documentType: OrganizeEditorDraftType,
  id: string,
): OrganizeEditorDraft | null => {
  if (!id || !canUseSessionStorage()) return null
  try {
    const raw = window.sessionStorage.getItem(draftStorageKey(documentType, id))
    if (!raw) return null
    const parsed = JSON.parse(raw)
    return parsed && typeof parsed === 'object' ? parsed as OrganizeEditorDraft : null
  } catch {
    return null
  }
}

export const clearOrganizeEditorDraft = (documentType: OrganizeEditorDraftType, id: string) => {
  if (!id || !canUseSessionStorage()) return
  try {
    window.sessionStorage.removeItem(draftStorageKey(documentType, id))
  } catch {
    // Ignore storage failures.
  }
}
