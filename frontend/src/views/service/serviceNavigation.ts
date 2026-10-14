import type {
  LocationQueryRaw,
  RouteLocationNormalizedLoaded,
  RouteLocationRaw,
} from 'vue-router'
import {
  createServiceSession,
  getFirstServiceSession,
  type ServiceRecord,
  type ServiceSession,
} from './serviceHubState'

const serviceSessionRequests = new Map<string, Promise<ServiceSession>>()

export const serviceBasePath = (route: RouteLocationNormalizedLoaded) =>
  route.meta.mobileEntry ? '/mobile/service' : '/platform/service'

export const serviceWorkspaceLocation = (
  route: RouteLocationNormalizedLoaded,
  serviceId: string,
  sessionId?: string,
  query: LocationQueryRaw = {},
): RouteLocationRaw => {
  const basePath = serviceBasePath(route)
  if (route.meta.mobileEntry) {
    return {
      path: basePath,
      query: {
        service: serviceId,
        ...(sessionId ? { session: sessionId } : {}),
        ...query,
      },
    }
  }
  const servicePath = `${basePath}/${encodeURIComponent(serviceId)}`
  return {
    path: sessionId
      ? `${servicePath}/sessions/${encodeURIComponent(sessionId)}`
      : servicePath,
    query,
  }
}

export const serviceDetailPath = (
  route: RouteLocationNormalizedLoaded,
  serviceId: string,
  suffix: 'overview' | 'activity' | 'settings/basic',
) => `${serviceBasePath(route)}/${encodeURIComponent(serviceId)}/${suffix}`

export const ensureServiceSession = async (serviceId: string) => {
  const existing = getFirstServiceSession(serviceId)
  if (existing) return existing
  const pending = serviceSessionRequests.get(serviceId)
  if (pending) return pending

  const request = createServiceSession(serviceId, { title: '新的会话' })
    .finally(() => serviceSessionRequests.delete(serviceId))
  serviceSessionRequests.set(serviceId, request)
  return request
}

export const serviceStateLabel = (state?: ServiceRecord['state']) => {
  switch (state) {
    case 'active':
      return '进行中'
    case 'paused':
      return '已暂停'
    case 'archived':
      return '已归档'
    case 'draft':
      return '草稿'
    default:
      return '服务'
  }
}
