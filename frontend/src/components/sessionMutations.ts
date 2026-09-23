import {
  clearSessionMessages,
  delSession,
  pinSession,
  unpinSession,
  updateSession,
} from '@/api/chat'

export const SESSION_MUTATION_EVENT = 'weknora:session-mutation'

export interface SessionMutationPatch {
  title?: string
  is_pinned?: boolean
  pinned_at?: string | null
}

export interface SessionMutationDetail {
  sessionId: string
  patch?: SessionMutationPatch
  messagesCleared?: boolean
  removed?: boolean
}

export function notifySessionMutation(detail: SessionMutationDetail): void {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<SessionMutationDetail>(SESSION_MUTATION_EVENT, { detail }))
}

function ensureSuccess(response: any): any {
  if (!response?.success) {
    throw new Error(response?.message || 'session mutation failed')
  }
  return response
}

export async function renameSession(
  sessionId: string,
  title: string,
  description = '',
  serviceId = '',
): Promise<any> {
  const response = ensureSuccess(await updateSession(sessionId, { title, description }, serviceId || undefined))
  const nextTitle = response.data?.title || title
  notifySessionMutation({ sessionId, patch: { title: nextTitle } })
  return response.data
}

export async function setSessionPinned(sessionId: string, pinned: boolean, serviceId = ''): Promise<void> {
  const response = ensureSuccess(
    pinned
      ? await pinSession(sessionId, serviceId || undefined)
      : await unpinSession(sessionId, serviceId || undefined),
  )
  notifySessionMutation({
    sessionId,
    patch: {
      is_pinned: pinned,
      pinned_at: pinned ? new Date().toISOString() : null,
    },
  })
}

export async function clearSession(sessionId: string, serviceId = ''): Promise<void> {
  ensureSuccess(await clearSessionMessages(sessionId, serviceId || undefined))
  notifySessionMutation({ sessionId, messagesCleared: true })
}

export async function removeSession(sessionId: string, serviceId = ''): Promise<void> {
  ensureSuccess(await delSession(sessionId, serviceId || undefined))
  notifySessionMutation({ sessionId, removed: true })
}
