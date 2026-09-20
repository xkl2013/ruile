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
  editionFeature?: string
}

export interface AdminNavGroup {
  key: string
  label: string
  items: AdminNavItem[]
}

export const ADMIN_NAV_GROUPS: AdminNavGroup[] = [
  {
    key: 'workspace',
    label: '系统空间',
    items: [
      {
        key: 'overview',
        label: '概览',
        description: '系统运行状态、空间规模和关键后台入口',
        icon: 'dashboard',
        path: '/',
        requiresTenant: false,
      },
      {
        key: 'workspace-chat-history',
        label: '聊天历史',
        description: '当前工作区的聊天历史配置',
        icon: 'chat',
        path: '/workspaces/current/chat-history',
        minRole: 'admin',
      },
      {
        key: 'workspace-editions',
        label: '产品版本',
        description: '系统支持的版本、空间形态和能力开关',
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
        key: 'agents',
        label: '智能体',
        description: '统一维护平台内置智能体',
        icon: 'control-platform',
        path: '/agents',
        requiresSystemAdmin: true,
      },
      {
        key: 'service-config',
        label: '服务配置',
        description: '员工分身与服务能力',
        icon: 'setting',
        path: '/service/profiles',
        minRole: 'admin',
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
        description: '模型供应商、凭据、价格和调试',
        icon: 'server',
        path: '/models',
        requiresSystemAdmin: true,
      },
      {
        key: 'response-tier-settings',
        label: '回答档位',
        description: '全平台统一配置快速、均衡、极致模型',
        icon: 'layers',
        path: '/response-tiers',
        requiresSystemAdmin: true,
      },
      {
        key: 'knowledge-base-settings',
        label: '知识库配置',
        description: '当前工作区知识库的模型、解析、索引和存储配置',
        icon: 'file-setting',
        path: '/runtime/knowledge-base',
        minRole: 'viewer',
      },
      {
        key: 'data-vector-stores',
        label: '向量库',
        description: '向量数据库连接',
        icon: 'data-base',
        path: '/data/vector-stores',
        requiresSystemAdmin: true,
      },
      {
        key: 'data-parser-engines',
        label: '解析引擎',
        description: '文档解析和连接状态',
        icon: 'file-setting',
        path: '/data/parser-engines',
        requiresSystemAdmin: true,
      },
      {
        key: 'data-storage-backends',
        label: '存储后端',
        description: '对象存储和默认后端',
        icon: 'folder-setting',
        path: '/data/storage-backends',
        requiresSystemAdmin: true,
      },
      {
        key: 'extensions-web-search',
        label: '网络搜索',
        description: '搜索 provider 和凭据',
        icon: 'internet',
        path: '/extensions/web-search',
        requiresSystemAdmin: true,
      },
      {
        key: 'extensions-mcp',
        label: 'MCP 服务',
        description: 'MCP 服务、工具和策略',
        icon: 'link',
        path: '/extensions/mcp-services',
        requiresSystemAdmin: true,
      },
    ],
  },
  {
    key: 'membership',
    label: '会员',
    items: [
      {
        key: 'member-users',
        label: '用户管理',
        description: '查看系统下所有用户账号',
        icon: 'usergroup',
        path: '/members/users',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
      {
        key: 'member-enterprises',
        label: '企业管理',
        description: '查看系统下已开通的企业会员',
        icon: 'building-1',
        path: '/members/enterprises',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
      {
        key: 'member-billing',
        label: '订阅管理',
        description: '查看套餐、价格、积分账户和用量账本',
        icon: 'wallet',
        path: '/members/billing',
        requiresSystemAdmin: true,
        requiresTenant: false,
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
        key: 'system-enterprise-provisioning',
        label: '开通企业',
        description: '为指定用户手工开通企业并配置容量和积分',
        icon: 'usergroup-add',
        path: '/system/enterprise-provisioning',
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
