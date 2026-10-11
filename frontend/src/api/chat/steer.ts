import { del, get, post } from "../../utils/request";

export type SteerDelivery = 'inject' | 'after'

export type SteerQueueItem = {
  steer_id: string
  content: string
  delivery: SteerDelivery
  mentioned_items?: unknown[]
  promoting?: boolean
  awaitingIdleSend?: boolean
  pending?: boolean
  failed?: boolean
  expected_assistant_message_id?: string
  client_id?: string
}

export async function steerSession(
  session_id: string,
  query: string,
  mentionedItems: any[] = [],
  delivery: SteerDelivery = 'after',
  expectedAssistantMessageId?: string,
  steerId?: string,
) {
  return post(`/api/v1/sessions/${session_id}/steer`, {
    query,
    expected_assistant_message_id: expectedAssistantMessageId,
    steer_id: steerId,
    mentioned_items: mentionedItems,
    channel: 'web',
    delivery,
  });
}

export async function promoteSteerSession(session_id: string, steer_id: string) {
  return post(`/api/v1/sessions/${session_id}/steer/${steer_id}/inject`, {});
}

export async function listSteerSession(session_id: string) {
  return get(`/api/v1/sessions/${session_id}/steer`);
}

export async function removeSteerSession(session_id: string, steer_id: string) {
  return del(`/api/v1/sessions/${session_id}/steer/${steer_id}`);
}
