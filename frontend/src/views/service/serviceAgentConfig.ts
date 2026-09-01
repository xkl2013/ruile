import { BUILTIN_SERVICE_ASSISTANT_ID } from '@/api/agent'

const configuredServiceAgentId = import.meta.env.VITE_SERVICE_ASSISTANT_AGENT_ID?.trim()

export const SERVICE_ASSISTANT_AGENT_ID = configuredServiceAgentId || BUILTIN_SERVICE_ASSISTANT_ID
export const SERVICE_ASSISTANT_AGENT_NAME = '服务提醒'
export const SERVICE_ASSISTANT_INPUT_PLACEHOLDER = '围绕当前服务提醒整理摘要、话术和下一步'
export const SERVICE_ASSISTANT_SUGGESTION_LIMIT = 4
