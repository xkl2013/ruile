import { normalizeSharedKnowledgeBasePermission } from '../../utils/sharedKnowledgeBasePermission'

export type KnowledgeBasePermission = 'admin' | 'editor' | 'viewer' | ''

export interface KnowledgeBaseAccessInfo {
  access_source?: string
  my_permission?: string
}

interface KnowledgeBaseWriteOptions {
  permissionLoaded?: boolean
  kbInfo?: KnowledgeBaseAccessInfo | null
  sharedPermission?: string | null
  hasSharedRecord?: boolean
  isOwner?: boolean
  isTenantAdmin?: boolean
  isSystemAdmin?: boolean
}

export function effectiveKnowledgeBasePermission(
  kbInfo?: KnowledgeBaseAccessInfo | null,
  sharedPermission?: string | null,
): KnowledgeBasePermission {
  // The detail endpoint has already aggregated all eligible shared spaces.
  // A stale list grant must not override that current authorization result.
  if (kbInfo?.access_source === 'shared_space' || kbInfo?.access_source === 'shared_agent') {
    return normalizeSharedKnowledgeBasePermission(kbInfo.my_permission)
  }
  return normalizeSharedKnowledgeBasePermission(sharedPermission || kbInfo?.my_permission)
}

export function isSharedKnowledgeBaseAccess(
  kbInfo?: KnowledgeBaseAccessInfo | null,
  hasSharedRecord = false,
): boolean {
  return hasSharedRecord
    || kbInfo?.access_source === 'shared_space'
    || kbInfo?.access_source === 'shared_agent'
}

export function canWriteKnowledgeBase(options: KnowledgeBaseWriteOptions): boolean {
  const {
    permissionLoaded = true,
    kbInfo,
    sharedPermission,
    hasSharedRecord = false,
    isOwner = false,
    isTenantAdmin = false,
    isSystemAdmin = false,
  } = options

  if (!permissionLoaded || !kbInfo) return false
  if (kbInfo?.access_source === 'shared_agent') return false

  if (isSharedKnowledgeBaseAccess(kbInfo, hasSharedRecord)) {
    const permission = effectiveKnowledgeBasePermission(kbInfo, sharedPermission)
    return permission === 'admin' || permission === 'editor'
  }

  return isOwner || isTenantAdmin || isSystemAdmin
}
