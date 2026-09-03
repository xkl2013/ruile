export type AdminRole = 'viewer' | 'contributor' | 'admin' | 'owner'

export interface AdminNavItem {
  key: string
  label: string
  description: string
  icon: string
  path: string
  minRole?: AdminRole
  requiresSystemAdmin?: boolean
  requiresTenant?: boolean
}

export interface AdminNavGroup {
  key: string
  label: string
  items: AdminNavItem[]
}

export const ADMIN_NAV_GROUPS: AdminNavGroup[] = [
  {
    key: 'workspace',
    label: '企业空间',
    items: [
      {
        key: 'overview',
        label: '概览',
        description: '当前空间、权限和关键后台入口',
        icon: 'dashboard',
        path: '/',
        requiresTenant: false,
      },
      {
        key: 'workspace-overview',
        label: '空间信息',
        description: '空间资料、配额和状态',
        icon: 'home',
        path: '/workspaces/current/overview',
        minRole: 'viewer',
      },
      {
        key: 'workspace-members',
        label: '成员管理',
        description: '空间成员和角色管理',
        icon: 'usergroup',
        path: '/workspaces/current/members',
        minRole: 'viewer',
      },
      {
        key: 'workspace-chat-history',
        label: '聊天历史',
        description: '空间级聊天历史配置',
        icon: 'chat',
        path: '/workspaces/current/chat-history',
        minRole: 'admin',
      },
      {
        key: 'workspace-editions',
        label: '产品版本',
        description: '版本、空间形态和能力开关',
        icon: 'layers',
        path: '/workspaces/editions',
        minRole: 'viewer',
        requiresTenant: false,
      },
    ],
  },
  {
    key: 'assets',
    label: '资产治理',
    items: [
      {
        key: 'knowledge-bases',
        label: '知识库',
        description: '知识库资源和后台配置',
        icon: 'folder',
        path: '/knowledge-bases',
        minRole: 'viewer',
      },
      {
        key: 'agents',
        label: '智能体',
        description: '智能体配置和管理',
        icon: 'control-platform',
        path: '/agents',
        minRole: 'viewer',
      },
      {
        key: 'service-config',
        label: '服务配置',
        description: '员工分身与服务能力',
        icon: 'setting',
        path: '/service/profiles',
        minRole: 'admin',
      },
      {
        key: 'organizations',
        label: '共享空间',
        description: '组织、共享和跨空间协作',
        icon: 'usergroup',
        path: '/spaces/organizations',
        minRole: 'viewer',
      },
    ],
  },
  {
    key: 'publish',
    label: '渠道运营',
    items: [
      {
        key: 'publish-im',
        label: 'IM 渠道',
        description: '微信和 IM 接入渠道',
        icon: 'chat-bubble',
        path: '/publish/im',
        minRole: 'viewer',
      },
      {
        key: 'publish-embed',
        label: '嵌入渠道',
        description: '网页嵌入和公开访问渠道',
        icon: 'code',
        path: '/publish/embed',
        minRole: 'viewer',
      },
      {
        key: 'security-api-keys',
        label: 'API Key',
        description: '空间 API 凭证和 Principal',
        icon: 'secured',
        path: '/security/api-keys',
        minRole: 'owner',
      },
    ],
  },
  {
    key: 'runtime',
    label: '运行配置',
    items: [
      {
        key: 'models',
        label: '模型',
        description: '模型供应商、凭据和调试',
        icon: 'server',
        path: '/models',
        minRole: 'viewer',
      },
      {
        key: 'runtime-ollama',
        label: 'Ollama',
        description: '本地模型运行时',
        icon: 'terminal',
        path: '/runtime/ollama',
        minRole: 'admin',
      },
      {
        key: 'runtime-weknora-cloud',
        label: '睿乐大脑云',
        description: '云端模型凭据和模型导入',
        icon: 'cloud',
        path: '/runtime/weknora-cloud',
        minRole: 'admin',
      },
      {
        key: 'data-vector-stores',
        label: '向量库',
        description: '向量数据库连接',
        icon: 'data-base',
        path: '/data/vector-stores',
        minRole: 'viewer',
      },
      {
        key: 'data-parser-engines',
        label: '解析引擎',
        description: '文档解析和连接状态',
        icon: 'file-setting',
        path: '/data/parser-engines',
        minRole: 'viewer',
      },
      {
        key: 'data-storage-backends',
        label: '存储后端',
        description: '对象存储和默认后端',
        icon: 'folder-setting',
        path: '/data/storage-backends',
        minRole: 'viewer',
      },
      {
        key: 'extensions-web-search',
        label: '网络搜索',
        description: '搜索 provider 和凭据',
        icon: 'internet',
        path: '/extensions/web-search',
        minRole: 'viewer',
      },
      {
        key: 'extensions-mcp',
        label: 'MCP 服务',
        description: 'MCP 服务、工具和策略',
        icon: 'link',
        path: '/extensions/mcp-services',
        minRole: 'viewer',
      },
    ],
  },
  {
    key: 'system',
    label: '平台运维',
    items: [
      {
        key: 'system-settings',
        label: '全局设置',
        description: '平台级设置和管理员',
        icon: 'setting',
        path: '/system/settings',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
      {
        key: 'system-runtime-queues',
        label: '运行队列',
        description: '队列、任务和模型运行状态',
        icon: 'queue',
        path: '/system/runtime-queues',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
    ],
  },
]

export const ADMIN_NAV_ITEMS = ADMIN_NAV_GROUPS.flatMap((group) => group.items)
