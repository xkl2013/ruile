export type AdminRole = 'viewer' | 'contributor' | 'admin' | 'owner'

export interface AdminNavItem {
  key: string
  label: string
  description: string
  icon: string
  path: string
  children?: AdminNavItem[]
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
        key: 'skill-catalog',
        label: '全局 Skill',
        description: '管理所有工作空间可用的 Skill',
        icon: 'code',
        path: '/skills',
        requiresSystemAdmin: true,
      },
      {
        key: 'organize-templates',
        label: '整理模板',
        description: '维护平台整理模板、版本和试跑结果',
        icon: 'file-paste',
        path: '/organize/templates',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
      {
        key: 'discover-categories',
        label: '发现栏目',
        description: '维护发现页栏目和内容归类',
        icon: 'folder',
        path: '/organize/discover-categories',
        requiresSystemAdmin: true,
        requiresTenant: false,
      },
      {
        key: 'content-management',
        label: '内容管理',
        description: '管理用户可订阅的知识库和发布内容',
        icon: 'book-open',
        path: '/public-knowledge-bases',
        requiresSystemAdmin: true,
        children: [
          {
            key: 'public-knowledge-bases',
            label: '发布知识库',
            description: '创建、上传并发布用户可订阅的知识库',
            icon: 'book-open',
            path: '/public-knowledge-bases',
            requiresSystemAdmin: true,
          },
          {
            key: 'public-contents',
            label: '发布内容',
            description: '维护已发布知识库中的资料和内容',
            icon: 'file-setting',
            path: '/public-contents',
            requiresSystemAdmin: true,
          },
          {
            key: 'courses',
            label: '系列课程',
            description: '上传文件夹建课并发布到发现模块',
            icon: 'book-open',
            path: '/courses',
            requiresSystemAdmin: true,
          },
          {
            key: 'public-creators',
            label: '创作者管理',
            description: '查看创作者创建的知识库和内容并统一发布',
            icon: 'usergroup',
            path: '/public-creators',
            requiresSystemAdmin: true,
          },
        ],
      },
      {
        key: 'expert-packages',
        label: '专家',
        description: '专家包导入、发布和运行测试',
        icon: 'usergroup',
        path: '/experts',
        minRole: 'admin',
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

function flattenNavItems(items: AdminNavItem[]): AdminNavItem[] {
  return items.flatMap((item) => item.children?.length ? flattenNavItems(item.children) : [item])
}

export const ADMIN_NAV_ITEMS = ADMIN_NAV_GROUPS.flatMap((group) => flattenNavItems(group.items))
