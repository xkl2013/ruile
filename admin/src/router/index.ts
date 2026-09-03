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

type ComponentLoader = () => Promise<{ default: Component }>
type RoutePropsFactory = (route: ReturnType<typeof useRoute>) => Record<string, unknown>

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
    path: 'workspaces/current/overview',
    name: 'adminWorkspaceOverview',
    component: createModulePage(() => import('@/views/settings/TenantInfo.vue')),
    meta: {
      navKey: 'workspace-overview',
      title: '空间信息',
      description: '查看当前空间资料、状态和存储配额。',
      minRole: 'viewer',
    },
  },
  {
    path: 'workspaces/current/members',
    name: 'adminWorkspaceMembers',
    component: createModulePage(() => import('@/views/settings/TenantMembers.vue'), {
      disableInvitations: true,
    }),
    meta: {
      navKey: 'workspace-members',
      title: '成员管理',
      description: '管理空间成员、角色和空间审计。',
      minRole: 'viewer',
    },
  },
  {
    path: 'workspaces/current/audit-log',
    redirect: { name: 'adminWorkspaceMembers' },
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
      description: '园长版、园所版和集团版的空间形态。',
      requiresTenant: false,
    },
  },
  {
    path: 'knowledge-bases',
    name: 'adminKnowledgeBases',
    component: () => import('@admin/views/AdminKnowledgeBases.vue'),
    meta: {
      navKey: 'knowledge-bases',
      title: '知识库资源管理',
      description: 'P2 范围：知识库创建、排序、分享、目录、数据源和处理配置。',
      minRole: 'viewer',
    },
  },
  {
    path: 'knowledge-bases/:kbId/settings',
    name: 'adminKnowledgeBaseSettings',
    component: () => import('@admin/views/AdminKnowledgeBaseSettings.vue'),
    meta: {
      navKey: 'knowledge-bases',
      title: '知识库设置',
      description: 'P2 范围：替代主工程的大型知识库设置弹窗。',
      minRole: 'viewer',
    },
  },
  {
    path: 'knowledge-bases/:kbId/basic',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'basic' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/models',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'models' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/processing',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'processing' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/storage',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'storage' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/data-sources',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'dataSources' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/sharing',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'sharing' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/directories',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'directories' },
    }),
  },
  {
    path: 'knowledge-bases/:kbId/tags',
    redirect: (to) => ({
      name: 'adminKnowledgeBaseSettings',
      params: to.params,
      query: { ...to.query, tab: 'tags' },
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
      description: '基于成员分身描述管理服务能力。',
      minRole: 'admin',
    },
  },
  {
    path: 'spaces/organizations',
    name: 'adminOrganizations',
    component: createModulePage(() => import('@/views/organization/OrganizationList.vue')),
    meta: {
      navKey: 'organizations',
      title: '共享空间',
      description: '管理共享空间、参与空间、加入申请和跨空间共享。',
      minRole: 'viewer',
    },
  },
  {
    path: 'publish/im',
    name: 'adminPublishIM',
    component: createModulePage(
      () => import('@/views/integrations/IntegrationSettingsSection.vue'),
      { tab: 'im' },
    ),
    meta: {
      navKey: 'publish-im',
      title: 'IM 渠道',
      description: '管理微信和 IM 接入渠道。',
      minRole: 'viewer',
    },
  },
  {
    path: 'publish/embed',
    name: 'adminPublishEmbed',
    component: createModulePage(
      () => import('@/views/integrations/IntegrationSettingsSection.vue'),
      { tab: 'embed' },
    ),
    meta: {
      navKey: 'publish-embed',
      title: '嵌入渠道',
      description: '管理网页嵌入渠道、预览和 token 轮换。',
      minRole: 'viewer',
    },
  },
  {
    path: 'security/api-keys',
    name: 'adminSecurityApiKeys',
    component: createModulePage(
      () => import('@/views/integrations/IntegrationSettingsSection.vue'),
      { tab: 'api' },
    ),
    meta: {
      navKey: 'security-api-keys',
      title: 'API Key',
      description: '管理空间 API Key、能力范围和 API Principal。',
      minRole: 'owner',
    },
  },
  {
    path: 'security/api-principal',
    redirect: { name: 'adminSecurityApiKeys' },
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
      minRole: 'viewer',
    },
  },
  {
    path: 'runtime/ollama',
    name: 'adminRuntimeOllama',
    component: createModulePage(() => import('@/views/settings/OllamaSettings.vue')),
    meta: {
      navKey: 'runtime-ollama',
      title: 'Ollama',
      description: '查看和配置本地模型运行时。',
      minRole: 'admin',
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
      minRole: 'admin',
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
      minRole: 'viewer',
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
      minRole: 'viewer',
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
      minRole: 'viewer',
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
      minRole: 'viewer',
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
      minRole: 'viewer',
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
          title: 'Admin 概览',
          description: '当前空间的后台入口和权限状态。',
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

  if (to.meta.requiresTenant !== false && !authStore.hasValidTenant) {
    return {
      name: 'adminNoTenant',
      query: { redirect: to.fullPath },
    }
  }

  return true
})

export default router
