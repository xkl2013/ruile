import { defineAsyncComponent, defineComponent, h, type Component } from 'vue'
import { createRouter, createWebHistory, useRoute, type RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import AdminLayout from '@admin/layouts/AdminLayout.vue'
import AdminModulePage from '@admin/views/AdminModulePage.vue'
import AdminOverview from '@admin/views/AdminOverview.vue'
import AdminForbidden from '@admin/views/AdminForbidden.vue'
import AdminLoginBridge from '@admin/views/AdminLoginBridge.vue'
import AdminNoTenant from '@admin/views/AdminNoTenant.vue'
import AdminNotFound from '@admin/views/AdminNotFound.vue'
import AdminWorkspaceEditions from '@admin/views/AdminWorkspaceEditions.vue'
import type { AdminRole } from '@admin/config/navigation'
import { buildMainAppURL } from '@admin/utils/navigation'

type ComponentLoader = () => Promise<{ default: Component }>
type RoutePropsFactory = (route: ReturnType<typeof useRoute>) => Record<string, unknown>

function redirectToMainSettings(section: string) {
  const target = buildMainAppURL('/platform/settings')
  window.location.href = `${target}${target.includes('?') ? '&' : '?'}section=${section}`
  return false
}

function restoreStoredTokens(authStore: ReturnType<typeof useAuthStore>) {
  authStore.initFromStorage()

  const token = localStorage.getItem('weknora_token')
  if (token && !authStore.token) {
    authStore.setToken(token)
  }

  const refreshToken = localStorage.getItem('weknora_refresh_token')
  if (refreshToken && !authStore.refreshToken) {
    authStore.setRefreshToken(refreshToken)
  }
}

function createModulePage(
  loader: ComponentLoader,
  props?: Record<string, unknown> | RoutePropsFactory,
) {
  const AsyncModule = defineAsyncComponent(async () => (await loader()).default)
  return defineComponent({
    name: 'AdminWrappedModulePage',
    setup() {
      const route = useRoute()
      return () => h(
        AdminModulePage,
        null,
        {
          default: () => h(
            AsyncModule,
            typeof props === 'function' ? props(route) : props,
          ),
        },
      )
    },
  })
}

const moduleRoutes: RouteRecordRaw[] = [
  {
    path: 'workspaces/current/members',
    name: 'adminWorkspaceMembers',
    beforeEnter: () => redirectToMainSettings('members'),
    component: AdminNotFound,
    meta: { requiresTenant: false },
  },
  {
    path: 'workspaces/current/audit-log',
    beforeEnter: () => redirectToMainSettings('members'),
    component: AdminNotFound,
    meta: { requiresTenant: false },
  },
  {
    path: 'workspaces/current/chat-history',
    name: 'adminWorkspaceChatHistory',
    component: createModulePage(() => import('@/views/settings/ChatHistorySettings.vue')),
    meta: {
      navKey: 'workspace-chat-history',
      title: '聊天历史',
      description: '配置当前空间的聊天历史检索和索引策略。',
      minRole: 'admin',
    },
  },
  {
    path: 'workspaces/editions',
    name: 'adminWorkspaceEditions',
    component: AdminWorkspaceEditions,
    meta: {
      navKey: 'workspace-editions',
      title: '版本能力',
      description: '个人版与企业版的空间形态和能力边界。',
      requiresTenant: false,
    },
  },
  {
    path: 'knowledge-bases',
    name: 'adminKnowledgeBases',
    redirect: { name: 'adminKnowledgeBaseSettings' },
  },
  {
    path: 'runtime/knowledge-base',
    name: 'adminKnowledgeBaseSettings',
    component: () => import('@admin/views/AdminKnowledgeBaseDefaults.vue'),
    meta: {
      navKey: 'knowledge-base-settings',
      title: '知识库配置',
      description: '维护当前工作区知识库的模型、解析、索引和存储默认配置。',
      minRole: 'viewer',
    },
  },
  {
    path: 'knowledge-bases/settings',
    redirect: { name: 'adminKnowledgeBaseSettings' },
  },
  {
    path: 'knowledge-bases/:kbId/settings',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/basic',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/models',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/processing',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'processing' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/storage',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'storage' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/data-sources',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/sharing',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/directories',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/tags',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'agents',
    name: 'adminAgents',
    component: createModulePage(() => import('@/views/agent/AgentList.vue')),
    meta: {
      navKey: 'agents',
      title: '智能体',
      description: '配置空间智能体、工具、知识库和发布能力。',
      minRole: 'viewer',
    },
  },
  {
    path: 'agents/:agentId/edit',
    redirect: (to) => ({
      name: 'adminAgents',
      query: {
        ...to.query,
        edit: String(to.params.agentId),
        section: typeof to.query.section === 'string' ? to.query.section : 'basic',
      },
    }),
  },
  {
    path: 'service/profiles',
    name: 'adminServiceProfiles',
    component: () => import('@admin/views/AdminServiceProfiles.vue'),
    meta: {
      navKey: 'service-config',
      title: '服务配置',
      description: '基于员工分身描述管理服务能力。',
      minRole: 'admin',
    },
  },
  {
    path: 'spaces/organizations',
    name: 'adminOrganizations',
    beforeEnter: () => redirectToMainSettings('sharedSpace'),
    component: AdminNotFound,
    meta: { requiresTenant: false },
  },
  {
    path: 'models',
    name: 'adminModels',
    component: createModulePage(
      () => import('@/views/settings/ModelSettings.vue'),
      (route) => ({ initialType: route.query.type ?? null }),
    ),
    meta: {
      navKey: 'models',
      title: '模型',
      description: '管理模型供应商、模型凭据和调试。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'runtime/weknora-cloud',
    name: 'adminRuntimeWeKnoraCloud',
    component: createModulePage(() => import('@/views/settings/WeKnoraCloudSettings.vue')),
    meta: {
      navKey: 'runtime-weknora-cloud',
      title: '睿乐大脑云',
      description: '配置云端凭据并导入云端模型。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'data/vector-stores',
    name: 'adminDataVectorStores',
    component: createModulePage(() => import('@/views/settings/VectorStoreSettings.vue')),
    meta: {
      navKey: 'data-vector-stores',
      title: '向量库',
      description: '管理向量数据库连接、测试和默认能力。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'data/parser-engines',
    name: 'adminDataParserEngines',
    component: createModulePage(() => import('@/views/settings/ParserEngineSettings.vue')),
    meta: {
      navKey: 'data-parser-engines',
      title: '解析引擎',
      description: '管理文档解析引擎状态和空间级解析配置。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'data/storage-backends',
    name: 'adminDataStorageBackends',
    component: createModulePage(() => import('@/views/settings/StorageBackendSettings.vue')),
    meta: {
      navKey: 'data-storage-backends',
      title: '存储后端',
      description: '管理对象存储连接、测试和默认后端。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'extensions/web-search',
    name: 'adminExtensionsWebSearch',
    component: createModulePage(() => import('@/views/settings/WebSearchSettings.vue')),
    meta: {
      navKey: 'extensions-web-search',
      title: '网络搜索',
      description: '管理搜索 provider、凭据和连接测试。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'extensions/mcp-services',
    name: 'adminExtensionsMcp',
    component: createModulePage(() => import('@/views/settings/McpSettings.vue')),
    meta: {
      navKey: 'extensions-mcp',
      title: 'MCP 服务',
      description: '管理 MCP 服务定义、工具和策略。',
      requiresSystemAdmin: true,
    },
  },
  {
    path: 'system/settings',
    name: 'adminSystemSettings',
    component: createModulePage(() => import('@/views/system/SystemSettings.vue')),
    meta: {
      navKey: 'system-settings',
      title: '全局设置',
      description: '平台级设置、系统管理员和系统审计。',
      requiresSystemAdmin: true,
      requiresTenant: false,
    },
  },
  {
    path: 'system/admins',
    redirect: { name: 'adminSystemSettings' },
  },
  {
    path: 'system/users',
    redirect: { name: 'adminSystemSettings' },
  },
  {
    path: 'system/audit-log',
    redirect: { name: 'adminSystemSettings' },
  },
  {
    path: 'system/runtime-queues',
    name: 'adminSystemRuntimeQueues',
    component: createModulePage(() => import('@/views/system/RuntimeQueues.vue')),
    meta: {
      navKey: 'system-runtime-queues',
      title: '运行队列',
      description: '查看队列状态、任务列表和运行时模型负载。',
      requiresSystemAdmin: true,
      requiresTenant: false,
    },
  },
]

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'adminLogin',
    component: AdminLoginBridge,
    meta: {
      title: '登录 Admin',
      public: true,
      requiresTenant: false,
    },
  },
  {
    path: '/',
    component: AdminLayout,
    meta: {
      requiresTenant: false,
    },
    children: [
      {
        path: '',
        name: 'adminOverview',
        component: AdminOverview,
        meta: {
          navKey: 'overview',
          title: '系统概览',
          description: '查看系统运行状态、空间规模和后台能力。',
          requiresTenant: false,
        },
      },
      {
        path: 'no-tenant',
        name: 'adminNoTenant',
        component: AdminNoTenant,
        meta: {
          title: '需要工作空间',
          requiresTenant: false,
        },
      },
      {
        path: '403',
        name: 'adminForbidden',
        component: AdminForbidden,
        meta: {
          title: '无权限',
          requiresTenant: false,
        },
      },
      ...moduleRoutes,
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'adminNotFound',
    component: AdminNotFound,
    meta: {
      title: '页面不存在',
      public: true,
      requiresTenant: false,
    },
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior() {
    return { top: 0 }
  },
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true

  const authStore = useAuthStore()
  restoreStoredTokens(authStore)

  const hasToken = Boolean(localStorage.getItem('weknora_token') || authStore.token)
  if (!hasToken) {
    return {
      name: 'adminLogin',
      query: { redirect: to.fullPath },
    }
  }

  const hydrated = await authStore.refreshFromAuthMe()
  if (!hydrated) {
    return {
      name: 'adminLogin',
      query: { redirect: to.fullPath },
    }
  }

  if (to.meta.requiresSystemAdmin && !authStore.isSystemAdmin) {
    return {
      name: 'adminForbidden',
      query: { from: to.fullPath, reason: 'system_admin' },
    }
  }

  const minRole = to.meta.minRole as AdminRole | undefined
  if (minRole && !authStore.hasRole(minRole)) {
    return {
      name: 'adminForbidden',
      query: { from: to.fullPath, reason: minRole },
    }
  }

  const editionFeature = to.meta.editionFeature as string | undefined
  if (editionFeature && !authStore.hasEditionFeature(editionFeature)) {
    return {
      name: 'adminForbidden',
      query: { from: to.fullPath, reason: 'enterprise' },
    }
  }

  if (to.meta.requiresTenant !== false && !authStore.hasValidTenant) {
    return {
      name: 'adminNoTenant',
      query: { redirect: to.fullPath },
    }
  }

  return true
})

export default router
