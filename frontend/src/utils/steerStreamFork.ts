import { markRaw } from 'vue'

export type ChatMessage = Record<string, any>

export function previewSteerMessage(
  list: ChatMessage[],
  item: { steer_id: string; content: string; mentioned_items?: unknown[] },
): ChatMessage {
  const existing = list.find(m => m.role === 'user' && m.steer_id === item.steer_id)
  if (existing) return existing
  const message: ChatMessage = {
    id: `steer-user-${item.steer_id}`,
    steer_id: item.steer_id,
    role: 'user',
    content: item.content,
    mentioned_items: item.mentioned_items || [],
    isSteer: true,
    is_completed: true,
    _steerPending: true,
  }
  list.push(message)
  return message
}

export function discardSteerPreview(list: ChatMessage[], steerId: string): void {
  const index = list.findIndex(m => m.steer_id === steerId && m._steerPending)
  if (index >= 0) list.splice(index, 1)
}

export function persistedAssistantId(message: ChatMessage | undefined): string {
  if (!message) return ''
  if (typeof message.assistant_message_id === 'string' && message.assistant_message_id) {
    return message.assistant_message_id
  }
  return typeof message.id === 'string' ? message.id : ''
}

export function reconcileSteerMessageId(
  list: ChatMessage[],
  clientId: string,
  serverId: string,
): ChatMessage | undefined {
  const preview = list.find(m => m.role === 'user' && m.steer_id === clientId)
  const received = list.find(m => m.role === 'user' && m.steer_id === serverId)
  if (preview && received && preview !== received && preview._steerPending) {
    if (Array.isArray(preview.mentioned_items) && preview.mentioned_items.length) {
      received.mentioned_items = preview.mentioned_items
    }
    list.splice(list.indexOf(preview), 1)
    return received
  }
  if (preview) preview.steer_id = serverId
  return received || preview
}

export function sealAssistantSegment(message: ChatMessage): void {
  message.thinking = false
  message.is_completed = true
  message.steerForked = true
  const stream = message.agentEventStream
  if (!Array.isArray(stream)) return
  for (const raw of stream) {
    const event = raw as ChatMessage
    if (event.type === 'thinking' || event.type === 'answer') {
      event.done = true
      event.thinking = false
    }
  }
}

export function forkAfterInjectedUser(
  list: ChatMessage[],
  sourceAssistant: ChatMessage,
  userMessage: ChatMessage,
  steerId?: string,
): ChatMessage {
  userMessage.isSteer = true
  delete userMessage._steerPending
  delete userMessage._steerFailed

  const existingIdx = list.indexOf(userMessage)
  if (existingIdx > 0) {
    const before = list[existingIdx - 1]
    const after = list[existingIdx + 1]
    if (
      before?.role === 'assistant' &&
      before.request_id === sourceAssistant.request_id &&
      after?.role === 'assistant' &&
      after.request_id === sourceAssistant.request_id
    ) {
      sealAssistantSegment(before)
      if (after._steerReplayPending) {
        delete after._steerReplayPending
        after.is_completed = false
        after.steerForked = false
      }
      return after
    }
  }

  sealAssistantSegment(sourceAssistant)
  let userIdx = list.indexOf(userMessage)
  let sourceIdx = list.indexOf(sourceAssistant)
  if (userIdx < 0) {
    const insertAt = sourceIdx >= 0 ? sourceIdx + 1 : list.length
    list.splice(insertAt, 0, userMessage)
    userIdx = insertAt
  } else if (sourceIdx >= 0 && userIdx !== sourceIdx + 1) {
    list.splice(userIdx, 1)
    if (userIdx < sourceIdx) sourceIdx -= 1
    list.splice(sourceIdx + 1, 0, userMessage)
    userIdx = sourceIdx + 1
  }

  const afterUser = list[userIdx + 1]
  if (
    afterUser?.role === 'assistant' &&
    !afterUser.is_completed &&
    afterUser.request_id === sourceAssistant.request_id
  ) {
    return afterUser
  }

  const persistedId = sourceAssistant.assistant_message_id || sourceAssistant.id || ''
  const continuation: ChatMessage = {
    id: `steer-cont-${steerId || String(Date.now())}`,
    assistant_message_id: persistedId,
    request_id: sourceAssistant.request_id,
    role: 'assistant',
    content: '',
    isAgentMode: true,
    isRagMode: sourceAssistant.isRagMode,
    is_completed: false,
    hideContent: true,
    agentEventStream: markRaw([]),
    _eventMap: new Map(),
    _pendingToolCalls: new Map(),
    knowledge_references: [],
  }
  list.splice(userIdx + 1, 0, continuation)
  return continuation
}

function eventTime(event: ChatMessage): number {
  if (typeof event.timestamp === 'number' && Number.isFinite(event.timestamp)) return event.timestamp
  if (typeof event.timestamp === 'string') {
    const parsed = Date.parse(event.timestamp)
    return Number.isNaN(parsed) ? 0 : parsed
  }
  return 0
}

function isTrailingTurnEvent(event: ChatMessage): boolean {
  if (event.type === 'agent_complete' || event.type === 'stop') return true
  return event.type === 'answer' && event.done === true && !event.superseded && !event.intermediate_answer
}

export function expandSteerForksInHistory(messages: ChatMessage[]): ChatMessage[] {
  const out: ChatMessage[] = []
  for (let i = 0; i < messages.length; i++) {
    const item = messages[i]
    if (item.role !== 'assistant' || !item.request_id || item.steerForked) {
      out.push(item)
      continue
    }
    const persistedID = persistedAssistantId(item)
    if (persistedID && item.id !== persistedID) {
      out.push(item)
      continue
    }
    const forks: ChatMessage[] = []
    let j = i + 1
    while (j < messages.length && messages[j].role === 'user' && messages[j].request_id === item.request_id) {
      forks.push(messages[j])
      j++
    }
    if (!forks.length) {
      out.push(item)
      continue
    }

    const stream = Array.isArray(item.agentEventStream) ? item.agentEventStream : []
    const boundaryIDs = stream.filter((e: ChatMessage) => e.type === 'user_message_injected')
      .map((e: ChatMessage) => e.user_message_id)
    const hasBoundaries = boundaryIDs.length > 0
    if (hasBoundaries) {
      forks.sort((a, b) => {
        const ai = boundaryIDs.indexOf(a.id)
        const bi = boundaryIDs.indexOf(b.id)
        return (ai < 0 ? Infinity : ai) - (bi < 0 ? Infinity : bi)
      })
    }
    const cuts = forks.map((user) => {
      const parsed = Date.parse(String(user.created_at || ''))
      return Number.isNaN(parsed) ? 0 : parsed
    })
    const buckets: ChatMessage[][] = Array.from({ length: forks.length + 1 }, () => [])
    const trailing: ChatMessage[] = []
    let segmentIndex = 0
    for (const event of stream) {
      if (event.type === 'user_message_injected') {
        const index = forks.findIndex(user => user.id === event.user_message_id)
        if (index >= 0) segmentIndex = index + 1
        continue
      }
      if (isTrailingTurnEvent(event)) {
        trailing.push(event)
        continue
      }
      let target = hasBoundaries ? segmentIndex : 0
      const timestamp = eventTime(event)
      while (!hasBoundaries && target < cuts.length && timestamp >= cuts[target]) target++
      buckets[target].push(event)
    }
    buckets[buckets.length - 1].push(...trailing)

    const persistedId = persistedAssistantId(item)
    const original = { ...item }
    for (let segmentIndex = 0; segmentIndex < buckets.length; segmentIndex++) {
      const last = segmentIndex === buckets.length - 1
      const segment = segmentIndex === 0
        ? item
        : { ...original, id: `${persistedId}:steer:${segmentIndex}`, assistant_message_id: persistedId }
      segment.agentEventStream = markRaw(buckets[segmentIndex])
      if (last) {
        segment.content = original.content
        segment.is_completed = original.is_completed
        segment.steerForked = original.steerForked
      } else {
        segment.steerForked = true
        segment.is_completed = true
        segment.content = buckets[segmentIndex]
          .filter(e => e.type === 'answer' && !e.superseded)
          .map(e => e.content || '').join('')
        segment.knowledge_references = []
      }
      out.push(segment)
      if (segmentIndex < forks.length) {
        forks[segmentIndex].isSteer = true
        out.push(forks[segmentIndex])
      }
    }
    i = j - 1
  }
  return out
}

export function steerStepEvents(step: ChatMessage): ChatMessage[] {
  if (!Array.isArray(step.user_messages_before)) return []
  return step.user_messages_before.map((id: unknown) => ({
    type: 'user_message_injected',
    user_message_id: id,
  }))
}

export function isAssistantTurnComplete(message: ChatMessage | undefined): boolean {
  if (!message || message.steerForked) return false
  return Boolean(message.is_completed) || (
    Array.isArray(message.agentEventStream) &&
    message.agentEventStream.some((event: ChatMessage) =>
      event.type === 'agent_complete' || event.type === 'stop',
    )
  )
}
export function resetSteerTurnForReplay(list: ChatMessage[], requestId: string): ChatMessage | undefined {
  const segments = list.filter(m => m.role === 'assistant' && m.request_id === requestId)
  if (!segments.some(m => m.steerForked)) return undefined
  for (const segment of segments) {
    segment.content = ''
    segment.agentEventStream = []
    segment._eventMap = new Map()
    segment._pendingToolCalls = new Map()
    segment.is_completed = true
    segment._steerReplayPending = true
  }
  segments[0].is_completed = false
  segments[0].steerForked = false
  delete segments[0]._steerReplayPending
  return segments[0]
}
