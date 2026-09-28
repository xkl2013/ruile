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
  if (kbInfo.my_permission === 'viewer') return false

  // Public subscriptions and the legacy subscription source are viewer-only.
  // Keep this explicit even when a stale tenant role or owner flag is present;
  // the backend applies the same read-only boundary to upload and content
  // mutation routes.
  if (
    kbInfo?.access_source === 'shared_agent'
    || kbInfo?.access_source === 'subscription'
    || kbInfo?.access_source === 'public_subscription'
  ) {
    return false
  }

  // The detail API is the authoritative projection of the permission that
  // the mutation endpoints will enforce. Prefer it over the active tenant
  // role so a tenant Admin cannot accidentally see edit controls for a
  // viewer-only shared/subscribed KB.
  if (kbInfo.my_permission) {
    return kbInfo.my_permission === 'admin' || kbInfo.my_permission === 'editor'
  }

  if (isSharedKnowledgeBaseAccess(kbInfo, hasSharedRecord)) {
    const permission = effectiveKnowledgeBasePermission(kbInfo, sharedPermission)
    return permission === 'admin' || permission === 'editor'
  }

  // Access sources returned by the detail endpoint already represent a
  // successful server-side authorization decision. Legacy responses without
  // my_permission can still use these explicit source values safely.
  if (
    kbInfo.access_source === 'created'
    || kbInfo.access_source === 'tenant_admin'
    || kbInfo.access_source === 'system_admin'
    || kbInfo.access_source === 'api_key'
  ) {
    return true
  }

  // Missing access metadata is not enough to expose mutation controls. Keep
  // only the explicit creator/system-admin fallback for older cached payloads.
  return isOwner || isSystemAdmin
}
