export type KnowledgeBaseScope = 'personal' | 'enterprise' | 'subscribed'

export type KnowledgeBaseScopeSource = {
  id?: string
  owner_type?: string
  access_source?: string
  list_category?: string
  isMine?: boolean
}

export function resolveKnowledgeBaseScope(
  kb: KnowledgeBaseScopeSource,
  fallback: KnowledgeBaseScope = 'personal',
): KnowledgeBaseScope {
  if (
    fallback === 'subscribed'
    || kb?.list_category === 'subscribed'
    || kb?.access_source === 'subscription'
  ) {
    return 'subscribed'
  }
  if (
    fallback === 'enterprise'
    || kb?.owner_type === 'organization'
    || kb?.access_source === 'shared_space'
    || kb?.access_source === 'shared_agent'
    || kb?.isMine === false
  ) {
    return 'enterprise'
  }
  return 'personal'
}
