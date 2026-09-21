import { fetchEventSource } from '@microsoft/fetch-event-source'
import { get, getDown, post } from '@/utils/request'
import { generateRandomString } from '@/utils/index'
import { getApiBaseUrl } from '@/utils/api-base'
import type {
  AgentRun,
  AgentRunEvent,
  AgentRunQuality,
  AgentRunStep,
} from './agent-run-types'

export type { AgentRun, AgentRunEvent, AgentRunQuality, AgentRunStep } from './agent-run-types'

interface AgentRunResponse<T> {
  success: boolean
  data: T
  message?: string
}

export interface AgentRunDiff {
  changed: boolean
  parent_run_id?: string
  run_id?: string
  before: string
  after: string
}

export function getAgentRun(id: string) {
  return get<AgentRunResponse<AgentRun>>(`/api/v1/agent-runs/${encodeURIComponent(id)}`)
}

export function listAgentThreadRuns(id: string) {
  return get<AgentRunResponse<AgentRun[]>>(`/api/v1/agent-runs/${encodeURIComponent(id)}/thread`)
}

export function getAgentRunDiff(id: string) {
  return get<AgentRunResponse<AgentRunDiff>>(`/api/v1/agent-runs/${encodeURIComponent(id)}/diff`)
}

export function getAgentRunQuality(id: string) {
  return get<AgentRunResponse<AgentRunQuality>>(`/api/v1/agent-runs/${encodeURIComponent(id)}/quality`)
}

export function listAgentRunSteps(id: string) {
  return get<AgentRunResponse<AgentRunStep[]>>(`/api/v1/agent-runs/${encodeURIComponent(id)}/steps`)
}

export function submitAgentRunAnswers(id: string, answers: Record<string, unknown>) {
  return post<AgentRunResponse<AgentRun>>(
    `/api/v1/agent-runs/${encodeURIComponent(id)}/answers`,
    { answers },
  )
}

export function regenerateAgentRun(id: string, feedback?: string) {
  return post<AgentRunResponse<AgentRun>>(
    `/api/v1/agent-runs/${encodeURIComponent(id)}/regenerate`,
    feedback?.trim() ? { feedback: feedback.trim() } : {},
  )
}

export function cancelAgentRun(id: string) {
  return post<AgentRunResponse<AgentRun>>(`/api/v1/agent-runs/${encodeURIComponent(id)}/cancel`)
}

export function getAgentArtifactPreview(runId: string, artifactId: string) {
  return get<string>(
    `/api/v1/agent-runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}/preview`,
    { responseType: 'text' },
  )
}

export function getAgentArtifactPreviewBlob(runId: string, artifactId: string) {
  return getDown(
    `/api/v1/agent-runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}/preview`,
  )
}

export function downloadAgentArtifact(runId: string, artifactId: string) {
  return getDown(
    `/api/v1/agent-runs/${encodeURIComponent(runId)}/artifacts/${encodeURIComponent(artifactId)}/download`,
  )
}

export async function streamAgentRunEvents(
  id: string,
  options: {
    afterSequence?: number
    signal?: AbortSignal
    onEvent: (event: AgentRunEvent) => void
  },
) {
  const token = localStorage.getItem('weknora_token')
  if (!token) throw new Error('登录状态已失效，请重新登录')

  const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id')
  const after = Math.max(0, Math.floor(options.afterSequence || 0))
  const url = `${getApiBaseUrl()}/api/v1/agent-runs/${encodeURIComponent(id)}/events?after=${after}`

  await fetchEventSource(url, {
    method: 'GET',
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: 'text/event-stream',
      'Accept-Language': localStorage.getItem('locale') || 'zh-CN',
      'X-Request-ID': generateRandomString(12),
      ...(selectedTenantId ? { 'X-Tenant-ID': selectedTenantId } : {}),
    },
    signal: options.signal,
    openWhenHidden: true,
    onopen: async (response) => {
      if (!response.ok) {
        throw new Error(`事件流连接失败（HTTP ${response.status}）`)
      }
    },
    onmessage: (message) => {
      if (!message.data) return
      const event = JSON.parse(message.data) as AgentRunEvent
      if (event.type !== 'PING') options.onEvent(event)
    },
    onclose: () => undefined,
    onerror: (error) => {
      throw error instanceof Error ? error : new Error('事件流连接失败')
    },
  })
}
