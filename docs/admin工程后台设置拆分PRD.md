# Admin 工程后台设置拆分 PRD

文档日期：2026-09-01

适用仓库：`/Users/jamgogh/Desktop/agent/ruile`

执行修正：本阶段优先把 Agent 列表和 Agent 编辑配置迁入 `admin` 工程；知识库后台治理属于 P2 范围，不替代 Agent 编辑迁移目标。

## 1. 背景

当前主前端工程同时承载两类体验：

- 业务工作台：知识库使用、聊天、整理、服务工作流、个人日常操作。
- 后台管理：空间成员、模型、数据基础设施、发布渠道、系统管理、知识库管理配置等。

两类体验的目标用户、权限模型、导航密度和操作风险不同。继续放在同一个 `frontend` 工作台里，会导致普通用户入口变重，也会让高风险配置项分散在多个业务页面中。需要把后台设置和管理类模块独立为 `admin` 工程，让主工程聚焦一线使用，admin 工程聚焦配置、治理、运维和审计。

定位修正：`admin` 工程面向软件开发者、实施/运维和平台运营者，不直接面向园长、老师、招生顾问或园所成员。园长版、园所版等是 admin 中被管理的产品版本和客户空间形态，不是 admin 的最终用户入口。

## 2. 目标

1. 新建独立 `admin` 前端工程，承接后台设置、管理、治理类页面。
2. 主前端保留业务工作台、个人偏好、消费型页面和日常操作入口。
3. 第一阶段不拆后端服务和数据库，admin 工程复用现有 `/api/v1` 管理接口。
4. 明确系统管理员、空间 Owner、空间 Admin、普通成员之间的页面和接口边界。
5. 保留旧入口兼容跳转，避免历史链接和用户习惯直接失效。

## 3. 非目标

1. 不把后端拆成单独 admin 服务。
2. 不重做权限模型，仍以服务端鉴权为准，前端只负责入口和可见性。
3. 不把聊天、知识消费、服务工作台、整理工作台迁入 admin。
4. 不把睿乐做成 CRM 后台。admin 只承接配置和治理，不承接客户跟进工作流。
5. 不把服务模块做成用户自助 Agent 开关台。普通用户只在主工程使用“服务提醒”，工作分身和能力配置由 AI 工程师在 admin 中完成。
6. 不在本次拆分中重命名现有 API 路径，除非后续单独做 API 治理。

## 4. 当前现状摘要

### 4.1 现有主要入口

| 当前入口 | 当前承载 | 迁移结论 |
| --- | --- | --- |
| `/platform/settings` | 主设置页，含账号、空间、模型、集成、数据扩展、系统管理 | 拆分，大部分进入 admin |
| `/platform/tenant` | 重定向到 settings | 改为跳转 admin 的空间概览 |
| `/platform/agents` | 重定向到 settings 的智能体 section | 改为跳转 admin 的智能体管理 |
| `/platform/integrations` | 重定向到 settings 的集成 section | 改为跳转 admin 的发布与 API 集成 |
| `/platform/organizations` | 组织和共享空间管理 | 迁入 admin |
| `/platform/system/*` | 系统设置、管理员、队列兼容入口 | 迁入 admin system |
| 知识库设置弹窗 | 知识库模型、解析、存储、分享、数据源等 | 分阶段迁入 admin 的知识库资源管理 |

### 4.2 当前设置页模块

当前 `frontend/src/views/settings/Settings.vue` 已经聚合以下模块：

- 账号：`GeneralSettings`、`UserProfile`
- 空间：`TenantInfo`、`TenantMembers`、`ChatHistorySettings`、共享空间入口
- 智能体：`AgentList`
- 发布集成：IM、Embed、API 集成
- 模型与运行时：`ModelSettings`、`OllamaSettings`、`WeKnoraCloudSettings`
- 数据与扩展：`VectorStoreSettings`、`ParserEngineSettings`、`StorageBackendSettings`、`WebSearchSettings`、`McpSettings`
- 系统管理：`SystemSettings`、`RuntimeQueues`
- 平台信息：`SystemInfo`

### 4.3 当前权限边界

当前前端按角色做入口控制，但服务端仍是最终鉴权：

- `viewer`：可读部分空间信息、系统信息、模型列表、部分状态和日志。
- `admin`：可修改大多数空间级配置，例如成员、模型配置、向量库、解析器、存储、搜索、MCP、智能体、发布渠道。
- `owner`：拥有更高风险的空间级权限，例如 API Key、API Principal、空间更新和删除。
- `system_admin`：独立于空间角色，管理全局设置、系统管理员、运行队列、全局审计、跨空间能力。

admin 工程必须把这些边界显式反映到导航、路由守卫、页面按钮和错误页中。

## 5. 模块迁移结论

### 5.1 P0 必须迁入 admin 的模块

P0 是后台拆分的首批范围，原因是这些模块直接影响平台安全、全局运行、租户空间和凭证管理。

| 模块 | 当前前端位置 | 主要后端接口 | 目标 admin 路径 | 权限 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 系统全局设置 | `views/system/SystemSettings.vue` | `/api/v1/system/admin/settings` | `/admin/system/settings` | `system_admin` | 含 SSRF 白名单、注册模式、租户默认策略、worker 并发、模型并发等 |
| 系统管理员管理 | 旧 `/platform/system/admins` 兼容入口 | `/api/v1/system/admin/admins` | `/admin/system/admins` | `system_admin` | 含提升、撤销系统管理员 |
| 用户密码重置 | 系统管理员功能 | `/api/v1/system/admin/users/:id/reset-password` | `/admin/system/users` | `system_admin` | 建议和系统用户搜索列表合并 |
| 运行队列管理 | `views/system/RuntimeQueues.vue` | `/api/v1/system/admin/runtime-queues` | `/admin/system/runtime-queues` | `system_admin` | 含队列状态、任务列表、任务操作 |
| 系统审计 | 后端已有系统审计接口 | `/api/v1/system/admin/audit-log` | `/admin/system/audit-log` | `system_admin` | 第一阶段可先做列表和筛选 |
| 空间概览与编辑 | `TenantInfo.vue` | `/api/v1/tenants/:id` | `/admin/workspaces/current/overview` | `viewer` 读，`owner` 改 | 删除空间必须 owner 二次确认 |
| 成员管理 | `TenantMembers.vue` | `/api/v1/tenants/:id/members` | `/admin/workspaces/current/members` | `admin` 写，`viewer` 读 | 包含 suspend、reactivate、remove、role update |
| 空间邀请 | `TenantMembers.vue` 内部能力 | `/api/v1/tenants/:id/invitations` | `/admin/workspaces/current/invitations` | `admin` | 用户接受邀请仍留主工程 |
| 空间审计 | 成员和空间管理相关 | `/api/v1/tenants/:id/audit-log` | `/admin/workspaces/current/audit-log` | `admin` | 建议作为企业空间二级页 |
| API Key 管理 | `ApiIntegrationSettings.vue` | `/api/v1/tenants/:id/api-keys` | `/admin/security/api-keys` | `owner` | 涉及密钥创建、删除、测试 |
| API Principal 配置 | API 集成设置 | `/api/v1/tenants/:id/api-principal-config` | `/admin/security/api-principal` | `owner` | 与 API Key 放同一个安全模块 |
| 聊天历史配置 | `ChatHistorySettings.vue` | `/api/v1/tenants/kv/chat-history-config` | `/admin/workspaces/current/chat-history` | `admin` | 空间级行为配置，迁入 admin |

### 5.2 P1 迁入 admin 的模块

P1 是空间 Admin 常用配置，建议在 P0 shell 稳定后迁移。

| 模块 | 当前前端位置 | 主要后端接口 | 目标 admin 路径 | 权限 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 模型管理 | `ModelSettings.vue`、`ModelEditorDialog.vue`、`ModelDebugDrawer.vue` | `/api/v1/models` | `/admin/models` | `viewer` 读，`admin` 写 | 凭据更新保持脱敏展示 |
| Ollama 运行时 | `OllamaSettings.vue` | `/api/v1/initialization/ollama/*` | `/admin/runtime/ollama` | `viewer` 状态，`admin` 检查和下载 | 和模型管理同组 |
| 睿乐大脑云配置 | `WeKnoraCloudSettings.vue` | `/api/v1/weknoracloud/*` | `/admin/runtime/weknora-cloud` | `viewer` 状态，`admin` 保存凭据 | 凭据字段必须只写不回显 |
| 向量数据库 | `VectorStoreSettings.vue` | `/api/v1/vector-stores` | `/admin/data/vector-stores` | `viewer` 读，`admin` 写 | 含连接测试、默认选择 |
| 解析引擎 | `ParserEngineSettings.vue` | `/api/v1/system/parser-engines`、`/api/v1/tenants/kv/parser-engine-config` | `/admin/data/parser-engines` | `viewer` 读，`admin` 写 | 注意系统状态和空间配置是两个层级 |
| 存储后端 | `StorageBackendSettings.vue` | `/api/v1/storage-backends` | `/admin/data/storage-backends` | `viewer` 读，`admin` 写 | 当前 settings 使用该组件，旧 `StorageEngineSettings.vue` 需确认是否遗留 |
| 网络搜索 | `WebSearchSettings.vue` | `/api/v1/web-search-providers` | `/admin/extensions/web-search` | `viewer` 读，`admin` 写 | Provider 凭据脱敏 |
| MCP 服务 | `McpSettings.vue`、`McpServiceDialog.vue` | `/api/v1/mcp-services` | `/admin/extensions/mcp-services` | `viewer` 读，`admin` 写 | per-user OAuth 和 tool approval 不迁入 admin |
| 智能体配置 | `AgentList.vue`、`AgentEditorModal.vue` | `/api/v1/agents` | `/admin/agents` | `viewer` 读，`admin` 写 | 这里是智能体配置，不是聊天执行页 |
| 服务配置 | `admin/src/views/AdminServiceProfiles.vue`、`TenantMembers.vue` | `/api/v1/service/agent-templates`、`/api/v1/tenants/:id/members/:user_id/profile`、`/api/v1/tenants/:id/members/work-profile/suggest` | `/admin/service/profiles`、`/admin/workspaces/current/members` | `admin` 写，`viewer` 不开放 | 服务配置页保留服务项展示；成员分身描述在成员管理中作为必填字段维护，不再在服务配置页提供 per-member 分身表单 |
| IM 发布渠道 | `IMChannelPanel.vue` | `/api/v1/agents/:id/im-channels`、`/api/v1/im-channels` | `/admin/publish/im` | `viewer` 读，`admin` 写 | 微信二维码、启停等操作保留 admin |
| Embed 发布渠道 | `AgentEmbedChannelPanel.vue` | `/api/v1/agents/:id/embed-channels`、`/api/v1/embed-channels` | `/admin/publish/embed` | `viewer` 读，`admin` 写 | preview 可读，token rotate 需 admin |
| 组织和共享空间 | `OrganizationList.vue`、`OrganizationSettingsModal.vue`、`OrganizationEditorModal.vue` | `/api/v1/organizations` | `/admin/spaces/organizations` | `viewer` 读，`admin` 写 | 当前前端路由已要求 admin，迁移优先级高 |

### 5.3 P2 迁入 admin 的知识库资源管理模块

知识库设置和内容工作台耦合较深，建议作为 P2 单独拆。拆分时只迁移管理配置和资源治理，知识库消费和日常内容阅读仍留在主工程。

| 模块 | 当前前端位置 | 主要后端接口 | 目标 admin 路径 | 权限 | 备注 |
| --- | --- | --- | --- | --- | --- |
| 知识库创建、复制、删除 | `KnowledgeBaseList.vue`、`KnowledgeBaseMenu.vue` | `/api/v1/knowledge-bases` | `/admin/knowledge-bases` | `admin` 或创建者规则 | 删除必须二次确认 |
| 知识库排序 | 知识库列表 | `/api/v1/knowledge-bases/order` | `/admin/knowledge-bases/order` | `admin` 或 `system_admin` | 当前已是 admin-only 操作 |
| 知识库图标和基础信息 | `KnowledgeBaseEditorModal.vue` | `/api/v1/knowledge-bases/:id`、`/icon` | `/admin/knowledge-bases/:id/basic` | KB 写权限或 admin | 主工程仅保留展示 |
| 知识库模型绑定 | `KnowledgeBaseEditorModal.vue` | `/api/v1/knowledge-bases/:id/config` | `/admin/knowledge-bases/:id/models` | `KBAccessManage` 或 admin | 包含 LLM、embedding、rerank、ASR、OCR、VLM |
| 知识库解析和分块 | `KnowledgeBaseEditorModal.vue` | `/api/v1/knowledge-bases/:id/config` | `/admin/knowledge-bases/:id/processing` | `KBAccessManage` 或 admin | 包含 parser、chunking、多模态、图谱和高级项 |
| 知识库存储和向量库 | `KnowledgeBaseEditorModal.vue` | `/api/v1/knowledge-bases/:id/config` | `/admin/knowledge-bases/:id/storage` | `KBAccessManage` 或 admin | 使用 P1 的数据基础设施配置结果 |
| 知识库数据源 | `DataSourceSettings.vue`、`DataSourceEditorDialog.vue` | `/api/v1/datasource` | `/admin/knowledge-bases/:id/data-sources` | `viewer` 看日志，`admin` 写 | 同步、暂停、恢复、校验都属于后台操作 |
| 知识库分享 | `KBShareSettings` 相关能力 | 知识库分享接口、组织分享接口 | `/admin/knowledge-bases/:id/sharing` | 创建者、admin、system_admin | 区分客户私有记忆和公司公共知识库 |
| 知识库目录配置 | 知识库目录设置 | `/api/v1/knowledge-bases/:id/directory-config` | `/admin/knowledge-bases/:id/directories` | `KBAccessManage` 或 admin | 当前记忆中已有 admin 权限边界 |
| 知识库标签 | `KnowledgeTags` 相关能力 | `/api/v1/knowledge-bases/:id/tags` | `/admin/knowledge-bases/:id/tags` | 创建者、admin、KB 写权限 | 可与基础信息页合并 |

### 5.4 保留在主前端的模块

| 模块 | 保留原因 | 备注 |
| --- | --- | --- |
| 聊天、会话、知识库问答 | 一线使用场景 | 不进入 admin |
| 知识库阅读、文件预览、wiki、图谱浏览 | 内容消费场景 | 管理按钮跳转 admin |
| 整理工作台 | 业务生产场景 | 不属于后台设置 |
| 服务工作台和客户服务空间 | 服务执行场景 | 继续围绕“今天服务谁、怎么说、发什么、下一步做什么”；工作分身和能力配置迁入 admin |
| 个人资料和偏好 | 用户个人场景 | `UserProfile`、主题、语言、默认体验设置留主工程 |
| 邀请接受和个人通知 | 用户个人动作 | 邀请创建在 admin，接受在主工程 |
| MCP 用户 OAuth 授权 | 个人授权动作 | admin 只管 MCP 服务定义和策略 |
| Agent 工具审批 | 运行期用户决策 | 留在智能体或聊天工作流 |
| 只读系统信息 | 支持和诊断 | 可在主工程保留一个简化关于页，admin 可重复展示 |

## 6. Admin 信息架构

建议 admin 工程采用左侧导航，按治理对象组织，而不是照搬 settings section。

| 一级导航 | 二级页面 | 承载模块 |
| --- | --- | --- |
| 概览 | 工作区概览、运行状态 | 当前租户、关键配置健康状态、待处理告警 |
| 工作区 | 基础信息、成员、邀请、聊天历史、审计 | tenant 管理 |
| 知识库 | 知识库列表、基础信息、处理配置、数据源、分享、目录、标签 | P2 资源管理 |
| 智能体 | 智能体列表、编辑、调试配置、推荐问题 | agent 配置管理 |
| 服务交付 | 员工分身、服务能力、发布测试、配置审计 | AI 工程师为用户开通服务提醒能力 |
| 发布 | IM 渠道、嵌入渠道 | 外部触达和发布配置 |
| 安全 | API Key、API Principal、密钥使用记录 | 凭证和接口访问 |
| 模型与运行时 | 模型、Ollama、睿乐大脑云 | 模型供应商和运行配置 |
| 数据与扩展 | 向量库、解析引擎、存储、网络搜索、MCP 服务 | 基础设施连接 |
| 共享空间 | 组织列表、成员、共享关系 | organization 管理 |
| 系统管理 | 全局设置、系统管理员、运行队列、系统审计、用户管理 | `system_admin` 专属 |

## 7. 路由兼容策略

旧链接不要直接 404。主前端保留兼容路由或点击入口，跳转到 admin。

| 旧入口 | 新入口 |
| --- | --- |
| `/platform/settings?section=models` | `/admin/models` |
| `/platform/settings?section=ollama` | `/admin/runtime/ollama` |
| `/platform/settings?section=weknoracloud` | `/admin/runtime/weknora-cloud` |
| `/platform/settings?section=websearch` | `/admin/extensions/web-search` |
| `/platform/settings?section=mcp` | `/admin/extensions/mcp-services` |
| `/platform/settings?section=vectorstore` | `/admin/data/vector-stores` |
| `/platform/settings?section=parser` | `/admin/data/parser-engines` |
| `/platform/settings?section=storage` | `/admin/data/storage-backends` |
| `/platform/settings?section=members` | `/admin/workspaces/current/members` |
| `/platform/settings?section=tenant` | `/admin/workspaces/current/overview` |
| `/platform/settings?section=agents` | `/admin/agents` |
| `/platform/settings?section=integrations&tab=im` | `/admin/publish/im` |
| `/platform/settings?section=integrations&tab=embed` | `/admin/publish/embed` |
| `/platform/settings?section=integrations&tab=api` | `/admin/security/api-keys` |
| `/platform/settings?section=system-global` | `/admin/system/settings` |
| `/platform/settings?section=runtime-queues` | `/admin/system/runtime-queues` |
| `/platform/organizations` | `/admin/spaces/organizations` |
| 知识库设置弹窗 | `/admin/knowledge-bases/:kbId/settings` |

兼容跳转规则：

1. 已登录且有权限：直接跳 admin 对应页面。
2. 已登录但无权限：显示 admin 无权限页，并提供返回工作台。
3. 未登录：跳 admin 登录页，登录后回到目标页。
4. 主工程中的老设置页第一阶段可保留个人设置和跳转入口，第二阶段再移除已迁移页面。

## 8. 技术方案

### 8.1 工程结构

推荐新增：

```text
admin/
  package.json
  index.html
  vite.config.ts
  src/
    main.ts
    App.vue
    router/
    stores/
    layouts/
    views/
    modules/
packages/
  web-shared/
    src/
      api/
      auth/
      i18n/
      types/
      ui/
```

第一阶段如果时间紧，可以让 admin 通过 alias 临时复用 `frontend/src/api`、`frontend/src/stores/auth.ts` 和部分组件；但正式目标应该沉淀到 `packages/web-shared`，避免两个工程长期互相引用源码目录。

### 8.2 复用范围

必须复用：

- request 封装：JWT、刷新令牌、`X-Tenant-ID`、错误处理。
- auth store：当前用户、租户、角色、`isSystemAdmin`、`hasRole`。
- API 类型和接口函数：优先从 `frontend/src/api` 抽到 shared 包。
- i18n 文案基础能力。
- 通用 UI：空状态、错误页、危险操作确认、凭据输入、权限提示。

不建议直接复用：

- 主工作台布局、业务导航、聊天侧栏。
- 和业务页面状态强耦合的 store。
- 只为 settings 单页设计的 section 切换逻辑。

### 8.3 构建与部署

目标：

- 主工程仍构建为用户工作台。
- admin 工程构建为独立静态资源。
- 同源部署时，admin 访问路径为 `/admin/`。
- API 仍走 `/api/v1`。

部署要求：

1. Vite base 配置为 `/admin/`。
2. Nginx 或静态服务增加 `/admin/*` fallback 到 `/admin/index.html`。
3. Docker 构建增加 admin build 产物拷贝。
4. Makefile 增加本地 admin 启动命令，例如 `make dev-admin`。
5. CI 增加 `admin` 的 type-check 和 build。

### 8.4 权限与安全

1. 前端路由守卫只做体验控制，所有写操作仍必须依赖服务端鉴权。
2. admin 左侧导航按权限动态显示，但直接访问路由也要展示无权限页。
3. `system_admin` 页面不得因用户是空间 Owner 而开放。
4. Owner-only 功能必须单独标记，例如 API Key、API Principal、空间删除。
5. 所有密钥字段只允许写入和状态展示，不允许明文回显。
6. 危险操作必须二次确认，包括删除空间、删除知识库、撤销管理员、重置密码、删除密钥、运行队列任务操作。
7. admin 所有修改动作建议进入审计日志。

## 9. 功能需求

### 9.1 Admin Shell

- 支持独立登录页或复用主登录页。
- 支持当前空间选择。
- 支持按角色裁剪导航。
- 支持 403、404、加载失败和 token 过期处理。
- 支持从主工程跳入 admin 后保留目标路由。
- 支持回到工作台。

### 9.2 工作区管理

- 展示当前空间基础信息、创建者、当前角色、配额和关键状态。
- Owner 可编辑空间名称、描述、配置项。
- Owner 可删除空间，必须输入空间名称确认。
- Admin 可管理成员、邀请和成员状态。
- Viewer 可只读查看成员列表，是否展示取决于现有后端授权。
- Admin 可查看空间审计日志。

### 9.3 安全管理

- Owner 可创建、查看、删除 API Key。
- API Key 创建成功后只展示一次明文。
- 支持 API Key scope 展示和配置。
- 支持 API Principal 配置和测试。
- 所有凭据操作记录审计。

### 9.4 模型与运行时

- 支持模型列表、创建、编辑、删除、凭据更新、连接测试。
- 支持模型 debug 抽屉或独立页面。
- 支持 Ollama 状态、模型列表、检查、下载。
- 支持睿乐大脑云状态和凭据保存。
- 支持模型并发等系统级设置时，跳转到系统管理页面。

### 9.5 数据与扩展

- 支持向量数据库 provider 列表、创建、编辑、删除、测试。
- 支持解析引擎状态查看和空间默认解析配置。
- 支持存储后端列表、创建、编辑、删除、测试、设置默认。
- 支持网络搜索 provider 列表、创建、编辑、删除、测试。
- 支持 MCP 服务列表、创建、编辑、删除、连接测试、凭据、工具审批策略。
- 不承接 MCP 用户 OAuth 授权和运行期 tool approval。

### 9.6 智能体管理

- 支持智能体列表、创建、编辑、复制、删除。
- 支持占位变量、类型预设、推荐问题配置。
- 智能体执行、会话、服务使用入口留在主工程。
- 服务模块不在这里让工程师逐项给用户勾 Agent；用户级服务能力由“成员管理”的必填分身描述驱动，服务配置页保留服务项展示但不提供逐成员分身配置表单。

### 9.7 发布管理

- 支持 IM 渠道创建、编辑、启停、删除、二维码查看。
- 支持 Embed 渠道创建、编辑、删除、token 轮换、预览。
- 发布预览可读，变更类操作必须 admin。

### 9.8 共享空间

- 支持组织列表、创建、编辑、成员管理、共享关系管理。
- 支持从知识库或智能体管理页跳到相关组织共享设置。
- 共享关系变更必须进入审计。

### 9.9 知识库资源管理

- 支持知识库列表、创建、复制、删除、排序、图标管理。
- 支持基础信息、模型绑定、解析和分块、多模态、图谱、存储、数据源、分享、目录、标签。
- 主工程知识库详情页只保留使用和消费视图，管理按钮跳 admin。
- 对共享知识库必须严格沿用 `KBAccessWrite`、`KBAccessManage`、创建者、admin、system_admin 的既有边界。

### 9.10 系统管理

- 支持全局设置列表、编辑、重置和生效状态提示。
- 支持系统管理员列表、提升、撤销。
- 支持用户搜索和密码重置。
- 支持运行队列状态、任务筛选、任务操作。
- 支持系统审计日志。
- 支持租户默认配额和批量应用默认配额。

### 9.11 服务交付配置

服务交付配置是 AI 工程师的一对一交付后台，不是普通用户入口，也不是 CRM 后台。

目标：

- 读取成员管理中的必填分身描述，明确该成员的真实岗位、负责事项、不负责事项、服务对象、记忆范围、沟通风格和外部系统边界。
- 成员管理的操作列提供“配置员工分身”，可直接填写分身描述，也可输入岗位后由 AI 一键生成初稿。
- 服务配置页保留服务项/能力项展示，用于查看有哪些服务能力；不再维护 per-member 分身配置表单。
- 主工程“服务提醒”读取成员分身描述和服务规则生成建议，不把 Agent 开关暴露给普通用户。

页面主流程：

```text
成员管理选择目标成员
-> 点击操作列的“配置员工分身”
-> 录入岗位或直接填写分身描述
-> AI 一键生成分身描述初稿
-> 工程师确认服务对象、负责事项、不负责事项和记忆边界
-> 保存成员分身
-> 主工程服务提醒按成员分身描述生成建议
```

页面第一屏字段：

| 字段 | 说明 |
| --- | --- |
| 成员 | 成员管理列表中的目标成员 |
| 岗位 | 可选输入，例如园长、招生顾问、教务老师，用于 AI 一键生成 |
| 成员分身描述 | 必填，来源于成员管理，说明服务对象、负责事项、不负责事项、记忆范围和外部系统边界 |
| AI 一键录入 | 根据岗位生成分身描述初稿，工程师确认后保存 |

高级配置区：

- 暂不在服务配置页提供。
- 如后续需要公司公共知识库、Skill 或输出策略的高级绑定，应放在内部高级页，并继续以成员分身描述作为输入。

验收标准：

1. 成员管理列表展示分身描述预览，操作列提供“配置员工分身”按钮。
2. 工程师必须先在成员管理维护成员必填的分身描述；添加成员、定向邀请成员和编辑成员描述都必须填写。
3. 配置员工分身弹窗支持岗位输入和 AI 一键录入，生成结果进入分身描述 textarea，保存前可人工修改。
4. 服务配置页展示服务项/能力项，但不展示成员列表、目标成员、配置名称、发布状态、能力草稿或任何 per-member 分身配置表单。
5. 普通用户在主工程不看到分身描述维护入口、Agent 开关、能力方案 JSON。
6. 所有成员分身保存动作进入审计日志。
7. 页面文案避免“客户管理、商机管理、漏斗阶段”等 CRM 语言。

## 10. 研发拆分计划

### M0：范围确认和代码盘点

交付：

- 本 PRD。
- 当前 settings、system、organization、knowledge settings 模块清单。
- 旧入口到新入口映射。

验收：

- 产品和研发确认 P0、P1、P2 边界。
- 明确哪些主工程页面保留。

### M1：Admin 工程骨架

交付：

- 新增 `admin` Vite/Vue 工程。
- 新增 admin layout、router、auth guard、403/404 页面。
- 复用或抽取 request/auth/i18n。
- 增加 `make dev-admin` 和 admin build。

验收：

- 未登录访问 `/admin/*` 会跳登录。
- 已登录无权限访问受限页显示 403。
- 当前空间切换后 API 带正确 `X-Tenant-ID`。
- admin 工程可独立 type-check 和 build。

### M2：P0 后台基础模块迁移

交付：

- 系统设置、系统管理员、运行队列、系统审计。
- 空间概览、成员、邀请、空间审计。
- API Key、API Principal。
- 聊天历史配置。
- 主工程相关入口跳转 admin。

验收：

- `system_admin` 能访问系统管理，普通 Owner 不能访问。
- Tenant Admin 可管理成员，Owner-only 功能不会误开放。
- API Key 明文只展示一次。
- 旧 settings P0 深链可跳转到 admin。

### M3：P1 配置模块迁移

交付：

- 模型、Ollama、睿乐大脑云。
- 向量库、解析引擎、存储、网络搜索、MCP 服务。
- 智能体配置。
- 成员分身配置、AI 一键录入、发布测试和配置审计。
- IM 和 Embed 发布渠道。
- 组织和共享空间。

验收：

- 原 settings 中对应 section 均有 admin 新页面。
- 创建、编辑、删除、测试等写操作权限一致。
- 凭据字段没有明文回显。
- 成员管理可以维护必填分身描述，AI 一键录入可基于岗位生成初稿。
- 主工程不再展示这些后台配置表单。

### M4：P2 知识库资源管理迁移

交付：

- 知识库列表和资源管理。
- 知识库设置页替代现有大型弹窗。
- 数据源、分享、目录、标签管理。
- 主工程知识库详情页管理按钮跳 admin。

验收：

- 普通知识库使用者仍能在主工程阅读和问答。
- 管理者可在 admin 完成原弹窗内所有配置。
- 共享知识库权限不降级。
- 知识库删除、排序、分享变更有确认和审计。

### M5：主工程清理和兼容收口

交付：

- 主工程 settings 只保留账号和个人偏好。
- 移除已迁移后台表单和无用 section。
- 保留一段时间的兼容跳转和埋点。
- 更新 README、部署文档、开发文档。

验收：

- 主工程普通用户导航明显变轻。
- 历史链接不会出现空白页。
- admin 和主工程构建均通过。
- 权限冒烟测试覆盖 P0 和 P1 主路径。

## 11. 测试要求

### 11.1 前端测试

- admin router 权限守卫测试。
- 旧 settings section 到 admin 路径的映射测试。
- API Key、凭据输入、危险确认等高风险组件测试。
- 成员分身配置、AI 一键录入、保存校验和审计测试。
- 当前租户切换后请求头测试。
- 主工程入口清理后的冒烟测试。

### 11.2 后端测试

- 继续保证所有管理接口服务端鉴权。
- 覆盖 `system_admin` 与 tenant owner/admin 的交叉场景。
- 覆盖 API Key、tenant settings、model credentials、system settings、runtime queues 的 403 场景。
- 覆盖成员分身配置的 Admin+ 鉴权、空描述拒绝、AI 一键录入兜底返回和保存审计。

### 11.3 手工验收

- 用 viewer、admin、owner、system_admin 四类账号分别访问 admin。
- 检查直接访问受限 URL 不会绕过权限。
- 检查主工程业务路径不受影响。
- 检查 admin 在刷新、深链、登录回跳后状态正确。
- 检查所有凭据字段不会出现在列表、详情或日志中。

## 12. 风险和处理

| 风险 | 影响 | 处理 |
| --- | --- | --- |
| settings 单页组件和主工程布局强耦合 | 迁移成本高 | 先包一层 admin adapter，再逐步组件化 |
| auth store 是 UI 控制，容易被误认为安全边界 | 权限误判 | PRD 和代码注释明确服务端鉴权为准 |
| system_admin 与 tenant owner 混淆 | 高危权限误开放 | 路由 meta 和页面级能力检查分开 |
| 知识库设置弹窗过大 | P2 迁移容易拖慢 | P0/P1 先不碰内容消费，只迁资源管理 |
| 凭据字段回显 | 安全风险 | 所有 provider 统一用 masked credential 组件 |
| 服务配置滑向 CRM | admin 页面开始承接客户、线索、商机管理 | 服务配置只围绕成员分身、服务能力和权限边界，不建客户主数据 |
| AI 方案自动生效 | 错配能力或越权读取 | AI 一键录入只生成分身描述初稿，工程师保存后才作为服务能力判断输入 |
| 旧入口失效 | 用户困惑 | 至少保留一个版本周期兼容跳转 |
| 两个前端工程重复代码 | 维护成本高 | 抽 `packages/web-shared` 或先临时 alias 后收口 |

## 13. 待确认问题

1. admin 工程最终访问路径是否固定为 `/admin/`，还是需要独立域名。
2. 第一阶段是否允许旧 settings 中后台 section 只做跳转，不再保留原表单。
3. 知识库设置是否在 P2 一次性迁完，还是先把分享、目录、数据源等高风险项迁出。
4. 是否需要 admin 首页展示跨空间概览。若需要，只有 `system_admin` 可见。
5. 是否需要为 admin 增加独立审计埋点，记录页面级操作来源。
6. 成员分身描述和 AI 一键录入是否随服务模块作为 P1 必做，还是作为服务模块上线前的 P0.5。
7. AI 一键录入第一版使用规则模板、LLM，还是规则模板和 LLM 结合。

## 14. 最小可交付版本

如果要尽快落地，最小版本建议只做：

1. `admin` 工程骨架、登录态复用、当前空间选择、权限守卫。
2. 系统管理、成员管理、API Key、模型管理四个模块。
3. 若服务模块同步上线，成员管理增加“配置员工分身”和 AI 一键录入；服务配置页保留服务项展示。
4. 主工程旧入口跳转 admin。
5. admin 独立 build 和部署到 `/admin/`。

这个版本能先把最高风险、最不像业务工作台的后台能力移出主工程，同时不触碰知识库大弹窗和复杂内容页。

## 15. 产品版本与客户空间形态支持方案

### 15.1 核心结论

个人使用和组织使用不要拆成两套账号系统，也不要拆成两套后端。对外产品可以有个人版、园长版、园所版，但在内部实现上建议统一为：

- `User`：登录账号，代表一个自然人。
- `Tenant`：工作空间，代表资源、成员、权限、配额和账单边界。
- `Organization`：组织或共享空间，代表多个工作空间之间的协作和资源共享关系。
- `Edition`：产品版本或套餐，决定一个空间开放哪些能力。

对应到教育场景，admin 的操作对象是这些空间形态：

- 园长版：一个园长个人使用，本质是一个个人空间或个人增强空间，由运营者在 admin 中识别和治理。
- 园所版：一个幼儿园团队使用，本质是一个团队空间，由运营者在 admin 中管理版本、配额和能力。
- 集团版或联盟版：多个园所空间加入同一个组织，通过组织共享知识库、智能体和标准素材。

这样能保持权限模型稳定：账号永远是人，资源永远归空间，跨园所协作通过组织完成。

### 15.2 产品形态定义

| 产品形态 | 资源归属 | 典型客户 | 成员能力 | 适合场景 | 内部运营关注点 |
| --- | --- | --- | --- | --- | --- |
| 个人版 | 个人空间 | 老师、招生顾问、个人使用者 | 默认 1 人，可限制邀请 | 个人知识库、个人助理、个人资料整理 | 查看试用、配额、活跃度、升级线索 |
| 园长版 | 园长个人空间 | 园长、校长、负责人 | 默认 1 人，可升级为园所空间 | 园长个人沉淀招生话术、家长沟通、园所经营资料 | 管理园长身份、园所线索、套餐和升级 |
| 园所版 | 园所团队空间 | 园长、招生主管、老师、运营 | 支持成员、角色、邀请、审计 | 园所统一知识库、招生 SOP、服务话术、渠道发布 | 管理成员、配额、知识库、渠道、审计和续费 |
| 集团版 | 多园所组织 | 集团总部、区域负责人、多个园所 | 组织成员是园所空间，不是单个人 | 多园区模板、总部知识库下发、跨园共享 | 管理组织成员、总部模板、跨空间共享和统计 |

### 15.3 数据模型建议

当前 `TenantInfo` 已有 `name`、`description`、`business`、`storage_quota` 等字段，可以支撑基础空间使用。但为了长期支持版本化，不建议把所有差异塞进 `business` 字段。建议新增显式字段：

| 字段 | 建议值 | 说明 |
| --- | --- | --- |
| `workspace_type` | `personal`、`school`、`group_workspace` | 空间形态。个人、园所、集团工作空间 |
| `edition_code` | `personal`、`principal`、`school`、`group` | 产品版本。用于控制导航、套餐和能力 |
| `industry_code` | `kindergarten`、`general` | 行业类型。园所业务用 `kindergarten` |
| `member_limit` | 数字 | 成员上限。个人版为 1，园所版按套餐 |
| `plan_code` | `free`、`pro`、`team`、`enterprise` | 商业套餐，不等同于权限角色 |
| `school_profile` | JSON | 园所档案，例如城市、班级数、招生阶段、园所类型 |
| `enabled_modules` | JSON 或后端计算 | 当前空间可用模块，例如 `members`、`api_keys`、`publish_channels` |

短期可以先使用 `Tenant.business = kindergarten` 标记教育行业，用配置表或后端能力接口返回版本能力。中长期应做字段迁移，避免后续难以判断空间到底是个人、园长还是园所。

### 15.4 注册和开通流程

#### 新用户注册后

用户完成注册后进入空间开通页，不直接默认创建园所版。

建议提供三个动作：

1. 我个人使用
2. 我是园长，先自己用
3. 我代表园所团队使用
4. 我已有邀请码，加入园所或组织

其中前两个都创建个人型空间，但 `edition_code` 不同：

- 个人使用：`workspace_type=personal`，`edition_code=personal`
- 园长版：`workspace_type=personal`，`edition_code=principal`，`industry_code=kindergarten`
- 园所版：`workspace_type=school`，`edition_code=school`，`industry_code=kindergarten`

#### 园长版开通

需要填写：

- 园长姓名或账号昵称
- 园所名称，允许暂不绑定正式园所
- 所在城市
- 当前关注目标：招生、家长沟通、园所管理、教师培训、资料整理

系统创建一个园长个人空间，默认用户为 Owner。成员管理、组织管理、API Key 等高风险后台能力默认隐藏或按套餐开放。

#### 园所版开通

需要填写：

- 园所名称
- 城市和校区
- 园所规模，例如班级数、在园人数
- 主要使用目标，例如招生转化、家长服务、园务知识库、教师培训
- 是否邀请团队成员

系统创建园所团队空间，创建者为 Owner，并引导邀请招生主管、老师、运营等成员。

#### 加入已有园所

复用当前邀请能力：

- 园所 Admin 或 Owner 创建邀请。
- 被邀请人通过 `/me/invitations` 接受。
- 接受后进入对应园所空间。
- 用户仍可保留自己的个人空间，并通过空间切换器切换。

### 15.5 版本能力矩阵

| 能力 | 个人版 | 园长版 | 园所版 | 集团版 |
| --- | --- | --- | --- | --- |
| 个人知识库 | 支持 | 支持 | 支持 | 支持 |
| 园所公共知识库 | 不默认支持 | 支持个人维护 | 支持团队维护 | 支持总部模板和共享 |
| 成员邀请 | 不支持或仅升级后支持 | 不支持或仅升级后支持 | 支持 | 支持园所空间加入 |
| 角色权限 | 只有 Owner | 只有 Owner | Owner/Admin/Contributor/Viewer | 组织角色加空间角色 |
| 知识库分享 | 可分享给组织，需升级或开关 | 可分享给组织 | 支持 | 核心能力 |
| 后台智能体配置 | 支持轻量配置 | 支持招生和服务助理配置 | 支持团队智能体管理 | 支持模板化下发 |
| 员工分身配置 | 不默认开放 | AI 工程师可基于园长岗位生成招生/服务分身描述 | AI 工程师可按成员岗位配置分身描述 | 支持总部模板化分身描述 |
| IM/Embed 发布 | 可限制 | 可按套餐开放 | 支持 | 支持统一渠道策略 |
| API Key | 默认关闭 | 默认关闭或高级套餐 | Owner 可用 | 系统或总部管理员可用 |
| 审计日志 | 简化 | 简化 | 完整空间审计 | 组织和系统级审计 |
| 配额 | 小额度 | 中额度 | 按园所套餐 | 按集团套餐 |

### 15.6 Admin 工程里的内部运营视角

admin 工程不应该做成客户自助后台。它的使用者是软件开发者、实施/运维和平台运营者。`edition_code`、`workspace_type` 和能力开关的作用，是帮助内部人员判断“这个客户空间应该开放哪些能力、受哪些限制、需要什么运营动作”。

#### 个人版空间

显示：

- 空间基础信息
- 使用量和配额
- 试用状态
- 升级线索
- 资源列表，主要用于排障和客服支持

隐藏：

- 面向客户的成员管理体验
- 面向客户的组织治理体验
- 高风险 API Key 自助入口

#### 园长版空间

显示：

- 园长身份和园所线索
- 招生和服务能力配置状态
- 知识库和智能体配置状态
- 发布渠道开通状态
- 升级到园所版的运营动作

隐藏或弱化：

- 客户自助式多成员管理
- 面向客户的 API Key 创建入口
- 与该空间无关的系统级运维入口

#### 园所版空间

显示完整空间级治理能力：

- 园所空间概览
- 成员和角色
- 知识库资源管理
- 智能体管理
- 员工分身配置
- 发布渠道
- 数据源和模型配置
- 空间审计
- API Key，Owner 可见

#### 集团版组织

重点不是单个园所的成员管理，而是跨园所治理：

- 组织成员，也就是园所空间列表
- 总部知识库模板
- 共享到园所的智能体
- 加入申请和邀请
- 跨园所使用统计
- 总部管理员和区域管理员

### 15.7 园长版升级为园所版

必须支持平滑升级，建议提供两种路径。

#### 路径 A：原空间升级

把园长个人空间直接升级为园所空间：

- `workspace_type` 从 `personal` 改为 `school`
- `edition_code` 从 `principal` 改为 `school`
- 打开成员邀请、角色、审计和团队知识库能力
- 原知识库、智能体、文件和配置全部保留

适合：园长个人空间中的内容本来就是该园所资产。

#### 路径 B：创建园所空间并迁移资产

创建新的园所空间，再让园长选择迁移哪些资产：

- 选择知识库
- 选择智能体
- 选择话术、模板、数据源
- 可选择是否保留个人副本

适合：园长个人空间中混有私人资料、多个园所资料或试用内容。

推荐默认使用路径 B，因为它更容易保护个人私有内容。路径 A 作为“整空间升级”的高级选项。

### 15.8 组织和园所的关系

当前系统里的 `Organization` 更适合表示“多个空间的共享组织”，不是替代园所空间。

建议映射：

- 一个幼儿园 = 一个 `Tenant`，即园所空间。
- 一个园长个人版 = 一个 `Tenant`，即个人空间。
- 一个教育集团、加盟体系、区域联盟 = 一个 `Organization`。
- `Organization` 的成员是一个个园所空间或个人空间，不是直接把所有用户塞进同一个大空间。

这样做的好处：

- 园所数据天然隔离。
- 总部可以把知识库或智能体共享给多个园所。
- 每个园所仍能保留自己的招生资料、家长沟通记录和私有知识。
- 园所成员权限由各自空间管理，组织只管理共享关系和跨空间协作。

### 15.9 对本 PRD 的增量开发项

在 admin 工程拆分之外，需要补充以下开发项：

| 开发项 | 优先级 | 说明 |
| --- | --- | --- |
| 空间版本字段 | P0 | 新增 `workspace_type`、`edition_code`、`industry_code` 或等价字段 |
| 能力开关接口 | P0 | 后端返回当前空间可用模块、限制和套餐能力 |
| 开通页改造 | P0 | 主应用或运营开通页支持个人、园长、园所、加入已有空间 |
| Admin 菜单裁剪 | P0 | 按内部岗位、空间版本、权限、能力开关生成导航 |
| 员工分身与服务能力 | P1 | 成员管理维护必填分身描述；支持岗位输入和 AI 一键录入；Admin 服务配置不再展示单独配置表单 |
| 园长版升级流程 | P1 | 支持整空间升级或选择性迁移资产 |
| 园所档案 | P1 | 支持城市、校区、规模、招生阶段等行业字段 |
| 集团组织治理 | P2 | 支持总部模板、跨园所共享和统计 |
| 套餐和配额治理 | P2 | member limit、storage quota、渠道数、知识库数等限制 |

### 15.10 验收标准

1. 同一个账号可以同时拥有个人空间和园所空间，并能通过空间切换器切换。
2. 内部运营者查看个人版空间时，能看到试用、配额、资源和升级线索，但不会把成员管理包装成客户自助后台。
3. 内部运营者查看园长版空间时，能识别园长身份、园所线索、招生场景配置和升级路径。
4. 内部运营者查看园所版空间时，能治理成员、角色、审计、知识库、智能体、员工分身和发布渠道。
5. 园所版 Admin 不能执行 Owner-only 操作，例如删除空间和管理 API Principal。
6. 集团组织能邀请多个园所空间加入，并共享知识库或智能体。
7. 组织共享不会破坏园所私有数据隔离。
8. 所有版本差异都由后端能力和服务端鉴权兜底，不能只依赖前端隐藏菜单。
