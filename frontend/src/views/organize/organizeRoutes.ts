export type OrganizeTab = 'memory' | 'output' | 'sprout'
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
  memory: 'organizeMemory',
  output: 'organizeOutput',
  sprout: 'organizeSprout',
  editor: 'organizeEditor',
  memoryNotes: 'organizeMemoryNotes',
  memoryAudio: 'organizeMemoryAudio',
  memoryAudioCards: 'organizeMemoryAudioCards',
} as const

export const ORGANIZE_MENU_ROUTES: readonly OrganizeMenuRouteItem[] = [
  {
    key: 'memory',
    label: '记忆',
    icon: 'folder',
    count: 31,
    routeName: ORGANIZE_ROUTE_NAMES.memory,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/memory`,
  },
  {
    key: 'output',
    label: '发现',
    icon: 'star',
    count: 12,
    routeName: ORGANIZE_ROUTE_NAMES.output,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/output`,
  },
  {
    key: 'sprout',
    label: '发芽',
    icon: 'tree-list',
    count: 6,
    routeName: ORGANIZE_ROUTE_NAMES.sprout,
    path: `${ORGANIZE_ROUTE_BASE_PATH}/sprout`,
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
  return value === 'memory' || value === 'output' || value === 'sprout'
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

  if (isOrganizeTab(tab)) {
    return findOrganizeMenuRoute(tab)?.path || `${ORGANIZE_ROUTE_BASE_PATH}/memory`
  }

  return `${ORGANIZE_ROUTE_BASE_PATH}/memory`
}
