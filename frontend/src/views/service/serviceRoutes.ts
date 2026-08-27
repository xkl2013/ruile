export type ServiceTab = 'messages' | 'review'

export interface ServiceMenuRouteItem {
  key: ServiceTab
  label: string
  icon: string
  count: number
  routeName: string
  path: string
}

export const SERVICE_ROUTE_BASE_PATH = '/platform/service'

export const SERVICE_ROUTE_NAMES = {
  messages: 'serviceMessages',
  review: 'serviceReview',
} as const

export const SERVICE_MENU_ROUTES: readonly ServiceMenuRouteItem[] = [
  {
    key: 'messages',
    label: '消息',
    icon: 'chat',
    count: 4,
    routeName: SERVICE_ROUTE_NAMES.messages,
    path: `${SERVICE_ROUTE_BASE_PATH}/messages`,
  },
  {
    key: 'review',
    label: '复盘',
    icon: 'chart-line',
    count: 3,
    routeName: SERVICE_ROUTE_NAMES.review,
    path: `${SERVICE_ROUTE_BASE_PATH}/review`,
  },
]

export const isServiceTab = (value: unknown): value is ServiceTab => {
  return value === 'messages' || value === 'review'
}

export const findServiceMenuRoute = (tab: ServiceTab) => {
  return SERVICE_MENU_ROUTES.find((item) => item.key === tab)
}

export const resolveServiceRoutePath = (tab: unknown) => {
  if (isServiceTab(tab)) {
    return findServiceMenuRoute(tab)?.path || `${SERVICE_ROUTE_BASE_PATH}/messages`
  }

  return `${SERVICE_ROUTE_BASE_PATH}/messages`
}
