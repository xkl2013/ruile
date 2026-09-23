export type OrganizeTab = 'hub' | 'mine' | 'memory' | 'discover'
export type LegacyOrganizeTab = 'memory' | 'output' | 'sprout' | 'discover'
export type MemoryAssetKey = 'note' | 'audio' | 'audio-card'

export interface OrganizeMenuRouteItem {
  key: OrganizeTab
  label: string
  icon: string
  count: number
  routeName: string
  path: string
}

export interface OrganizeMemoryAssetRouteItem {
  key: MemoryAssetKey
  label: string
  icon: string
  count: number
  unit: string
  itemTypeLabel: string
  routeName: string
  path: string
}

export const ORGANIZE_ROUTE_BASE_PATH = '/platform/organize'

export const ORGANIZE_ROUTE_NAMES = {
  hub: 'organizeHub',
  memory: 'organizeMemory',
  mine: 'organizeMine',
  discover: 'organizeDiscover',
  configDetail: 'organizeConfigDetail',
  outputDetail: 'organizeOutputDetail',
  editor: 'organizeEditor',
  memoryNotes: 'organizeMemoryNotes',
  memoryAudio: 'organizeMemoryAudio',
  memoryAudioCards: 'organizeMemoryAudioCards',
} as const

export const ORGANIZE_MENU_ROUTES: readonly OrganizeMenuRouteItem[] = [
  {
    key: 'hub',
    label: '工作台',
    icon: 'dashboard',
    count: 0,
    routeName: ORGANIZE_ROUTE_NAMES.hub,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/hub`,
  },
  {
    key: 'mine',
    label: '我的整理',
    icon: 'task',
    count: 0,
    routeName: ORGANIZE_ROUTE_NAMES.mine,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/mine`,
  },
  {
    key: 'memory',
    label: '记忆',
    icon: 'folder',
    count: 0,
    routeName: ORGANIZE_ROUTE_NAMES.memory,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/memory`,
  },
  {
    key: 'discover',
    label: '发现',
    icon: 'browse',
    count: 0,
    routeName: ORGANIZE_ROUTE_NAMES.discover,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/discover`,
  },
]

export const ORGANIZE_MEMORY_ASSET_ROUTES: readonly OrganizeMemoryAssetRouteItem[] = [
  {
    key: 'note',
    label: '笔记',
    icon: 'folder',
    count: 29,
    unit: '条',
    itemTypeLabel: '笔记',
    routeName: ORGANIZE_ROUTE_NAMES.memoryNotes,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/memory/notes`,
  },
  {
    key: 'audio',
    label: '录音',
    icon: 'sound',
    count: 2,
    unit: '条',
    itemTypeLabel: '录音',
    routeName: ORGANIZE_ROUTE_NAMES.memoryAudio,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/memory/audio`,
  },
  {
    key: 'audio-card',
    label: '工牌',
    icon: 'file',
    count: 2,
    unit: '条',
    itemTypeLabel: '工牌',
    routeName: ORGANIZE_ROUTE_NAMES.memoryAudioCards,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/memory/audio-cards`,
  },
]

export const isOrganizeTab = (value: unknown): value is OrganizeTab => {
  return value === 'hub' || value === 'mine'
}

export const isMemoryAssetKey = (value: unknown): value is MemoryAssetKey => {
  return value === 'note' || value === 'audio' || value === 'audio-card'
}

export const findOrganizeMenuRoute = (tab: OrganizeTab) => {
  return ORGANIZE_MENU_ROUTES.find((item) => item.key === tab)
}

export const findMemoryAssetRoute = (asset: MemoryAssetKey) => {
  return ORGANIZE_MEMORY_ASSET_ROUTES.find((item) => item.key === asset)
}

export const resolveOrganizeRoutePath = (tab: unknown, asset?: unknown) => {
  if (tab === 'memory') {
    if (isMemoryAssetKey(asset)) {
      return findMemoryAssetRoute(asset)?.path || `${ORGANIZE_ROUTE_BASE_PATH}/memory`
    }
    return `${ORGANIZE_ROUTE_BASE_PATH}/memory`
  }

  if (tab === 'output') return `${ORGANIZE_ROUTE_BASE_PATH}/discover`
  if (tab === 'sprout') return `${ORGANIZE_ROUTE_BASE_PATH}/mine`

  if (isOrganizeTab(tab)) {
    return findOrganizeMenuRoute(tab)?.path || `${ORGANIZE_ROUTE_BASE_PATH}/hub`
  }

  return `${ORGANIZE_ROUTE_BASE_PATH}/hub`
}
