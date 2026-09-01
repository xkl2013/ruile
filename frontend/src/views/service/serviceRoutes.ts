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
export const SERVICE_MESSAGES_ROUTE_PATH = '/platform/messages'
export const SERVICE_REVIEW_ROUTE_PATH = '/platform/organize/daily'

export const SERVICE_ROUTE_NAMES = {
  messages: 'serviceMessages',
  review: 'serviceReview',
} as const

export const SERVICE_MENU_ROUTES: readonly ServiceMenuRouteItem[] = [
  {
    key: 'messages',
    label: '服务提醒',
    icon: 'chat',
    count: 4,
    routeName: SERVICE_ROUTE_NAMES.messages,
    path: SERVICE_MESSAGES_ROUTE_PATH,
  },
]

const SERVICE_ROUTE_ITEMS: readonly ServiceMenuRouteItem[] = [
  ...SERVICE_MENU_ROUTES,
  {
    key: 'review',
    label: '日报',
    icon: 'chart-line',
    count: 3,
    routeName: SERVICE_ROUTE_NAMES.review,
    path: SERVICE_REVIEW_ROUTE_PATH,
  },
]

export const isServiceTab = (value: unknown): value is ServiceTab => {
  return value === 'messages' || value === 'review'
}

export const findServiceMenuRoute = (tab: ServiceTab) => {
  return SERVICE_ROUTE_ITEMS.find((item) => item.key === tab)
}

export const resolveServiceRoutePath = (tab: unknown) => {
  if (isServiceTab(tab)) {
    return findServiceMenuRoute(tab)?.path || SERVICE_MESSAGES_ROUTE_PATH
  }

  return SERVICE_MESSAGES_ROUTE_PATH
}
