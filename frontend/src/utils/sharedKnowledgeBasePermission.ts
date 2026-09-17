export type SharedKnowledgeBasePermission = 'admin' | 'editor' | 'viewer'

interface SharedKnowledgeBaseRecord {
  knowledge_base?: { id: string } | null
  permission?: string | null
}

const permissionLevel: Record<SharedKnowledgeBasePermission, number> = {
  viewer: 1,
  editor: 2,
  admin: 3,
}

export function normalizeSharedKnowledgeBasePermission(
  permission?: string | null,
): SharedKnowledgeBasePermission | '' {
  return permission === 'viewer' || permission === 'editor' || permission === 'admin'
    ? permission
    : ''
}

export function highestSharedKnowledgeBase<T extends SharedKnowledgeBaseRecord>(
  records: readonly T[],
  kbId: string,
): T | null {
  let highest: T | null = null
  let highestLevel = 0
  for (const record of records) {
    if (record.knowledge_base?.id !== kbId) continue
    const permission = normalizeSharedKnowledgeBasePermission(record.permission)
    const level = permission ? permissionLevel[permission] : 0
    if (!highest || level > highestLevel) {
      highest = record
      highestLevel = level
    }
  }
  return highest
}
