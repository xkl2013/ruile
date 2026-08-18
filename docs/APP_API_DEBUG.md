# App 调试接口文档

本文档面向移动 App、小程序、H5 客户端调试，按实际接入链路整理常用接口、鉴权方式、请求示例和排障要点。完整 OpenAPI 以运行时 Swagger 为准：开发环境启动后访问 `http://localhost:8080/swagger/index.html`。

## 基础约定

| 项目 | 说明 |
| --- | --- |
| 本地服务 | `http://localhost:8080` |
| API 前缀 | `/api/v1` |
| 健康检查 | `GET /health`，无需鉴权 |
| Swagger | `GET /swagger/index.html`，仅 `GIN_MODE != release` 时启用 |
| 默认格式 | JSON：`Content-Type: application/json` |
| 文件上传 | `multipart/form-data` |
| 聊天响应 | Server-Sent Events：`Content-Type: text/event-stream` |

建议每个请求都带 `X-Request-ID`，便于服务端日志定位：

```http
X-Request-ID: app-20260816-0001
```

## 鉴权

App 调试优先使用空间 API Key：

```http
X-API-Key: sk-xxxxx
```

如果 App 走账号登录，则先调用 `/api/v1/auth/login` 获取 `token`，后续请求带：

```http
Authorization: Bearer <token>
```

`X-Tenant-ID` 主要用于 JWT 多空间切换。API Key 已绑定空间，普通 App 调试不要额外传 `X-Tenant-ID`。

### API Key 能力建议

| 场景 | 最小能力 |
| --- | --- |
| 查询空间、知识库、知识内容 | `retrieve` |
| 创建会话、问答、读取本会话消息 | `chat` |
| URL/文件/手工知识导入 | `ingest` |
| App 内创建、改名、删除知识库 | `manage_kbs` |
| 读取智能体列表或详情 | `read_agents`；如果已有 `chat` 也可读 |
| 搜索全空间历史对话 | `message_history` |
| 省事调试 | `full_access` |

如果 API Key 配了 `knowledge_base_ids` 白名单，只能访问白名单内知识库；空白名单表示当前空间全部知识库。

## 推荐调试链路

1. 检查服务：`GET /health`
2. 配置鉴权：保存 `baseUrl` 和 `apiKey`
3. 拉取空间：`GET /api/v1/tenants`
4. 拉取知识库：`GET /api/v1/knowledge-bases`
5. 可选导入知识：`POST /api/v1/knowledge-bases/{kb_id}/knowledge/url`
6. 轮询解析状态：`GET /api/v1/knowledge/{knowledge_id}`
7. 创建会话：`POST /api/v1/sessions`
8. 发起问答：`POST /api/v1/knowledge-chat/{session_id}`
9. 解析 SSE，拼接 `response_type=answer` 的 `content`
10. 拉取历史：`GET /api/v1/messages/{session_id}/load`

## 核心接口

### 1. 服务健康检查

```bash
curl 'http://localhost:8080/health'
```

响应：

```json
{"status":"ok"}
```

### 2. 获取当前用户或 API Key 身份

```bash
curl 'http://localhost:8080/api/v1/auth/me' \
  -H 'X-API-Key: sk-xxxxx'
```

常用于确认当前凭证解析出的用户和空间上下文。JWT 登录后也可以用 `Authorization: Bearer <token>` 调用。

### 3. 获取空间列表

```bash
curl 'http://localhost:8080/api/v1/tenants' \
  -H 'X-API-Key: sk-xxxxx'
```

响应中 `data` 为当前凭证可见的空间列表。App 一般只需要展示当前空间名称，API Key 模式下通常只有一个空间。

### 4. 获取知识库列表

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases' \
  -H 'X-API-Key: sk-xxxxx'
```

常用字段：

| 字段 | 说明 |
| --- | --- |
| `id` | 知识库 ID，后续问答和导入使用 |
| `name` | 知识库名称 |
| `type` | `document` 或 `faq` |
| `description` | 描述 |
| `knowledge_count` | 知识条数 |
| `chunk_count` | 分块数 |
| `processing_count` | 正在解析的知识数 |
| `is_pinned` | 当前用户是否置顶 |

### 5. 获取知识库详情

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001' \
  -H 'X-API-Key: sk-xxxxx'
```

详情包含知识库配置、模型配置、存储配置和统计字段。调试“能看到列表但不能问答”时，先检查知识库是否有已完成解析的内容。

### 6. URL 导入知识

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/url' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://github.com/Tencent/WeKnora",
    "enable_multimodel": false,
    "channel": "app"
  }'
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `url` | string | 是 | 要导入的网页或远程文件 URL |
| `file_name` | string | 否 | 指定后按远程文件模式处理 |
| `file_type` | string | 否 | 如 `pdf`、`docx` |
| `enable_multimodel` | boolean | 否 | 是否启用多模态解析 |
| `title` | string | 否 | 自定义标题 |
| `tag_id` | string | 否 | 标签 ID |
| `channel` | string | 否 | 来源标识，App 可传 `app`、`miniprogram` |

成功后返回 `data.id`，解析通常异步进行，初始 `parse_status` 多为 `processing`。

### 7. 文件上传导入知识

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/file' \
  -H 'X-API-Key: sk-xxxxx' \
  -F 'file=@/path/to/manual.pdf' \
  -F 'enable_multimodel=false' \
  -F 'channel=app'
```

不要手动设置 `Content-Type: application/json`。使用 `-F` 时 curl 会自动生成 multipart boundary。

### 8. 手工 Markdown 导入

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/manual' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "产品使用指南",
    "content": "# 产品使用指南\n\n这里是正文。",
    "status": "published",
    "channel": "app"
  }'
```

### 9. 查询知识解析状态

```bash
curl 'http://localhost:8080/api/v1/knowledge/knowledge-0001' \
  -H 'X-API-Key: sk-xxxxx'
```

关键状态：

| 状态 | 含义 |
| --- | --- |
| `pending` | 等待解析 |
| `processing` | 解析、分块、向量化中 |
| `finalizing` | 主解析完成，仍在生成摘要、问题或图谱 |
| `completed` | 可用于检索和问答 |
| `failed` | 解析失败，查看 `error_message` |
| `cancelled` | 已取消，可重新解析 |

也可以按知识库分页查询：

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge?page=1&page_size=20' \
  -H 'X-API-Key: sk-xxxxx'
```

### 10. 创建会话

```bash
curl 'http://localhost:8080/api/v1/sessions' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "App 调试会话",
    "description": "移动端接口联调"
  }'
```

响应中的 `data.id` 是 `session_id`，问答接口必须使用它。

### 11. 知识库问答

```bash
curl -N 'http://localhost:8080/api/v1/knowledge-chat/session-0001' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "请总结这个知识库的核心内容",
    "knowledge_base_ids": ["kb-00000001"],
    "channel": "app"
  }'
```

请求体：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `query` | string | 是 | 用户问题 |
| `knowledge_base_ids` | string[] | 否 | 指定检索的知识库 |
| `knowledge_ids` | string[] | 否 | 限定具体知识文件 |
| `agent_id` | string | 否 | 使用指定智能体，如 `builtin-quick-answer` |
| `summary_model_id` | string | 否 | 覆盖默认生成模型 |
| `disable_title` | boolean | 否 | 是否禁用自动会话标题 |
| `images` | object[] | 否 | 图片输入，字段为 `data`，值为 base64 data URI |
| `channel` | string | 否 | 来源标识 |

SSE 响应示例：

```text
event: message
data: {"id":"req-1","response_type":"references","content":"","done":false,"knowledge_references":[...]}

event: message
data: {"id":"req-1","response_type":"answer","content":"知识库主要包含...","done":false,"assistant_message_id":"assistant-message-id"}

event: message
data: {"id":"req-1","response_type":"answer","content":"","done":true}
```

App 侧处理建议：

| `response_type` | 处理方式 |
| --- | --- |
| `references` | 保存引用来源，用于展示出处 |
| `answer` | 按顺序拼接 `content` |
| `complete` | 生成流程结束 |
| `session_title` | 更新会话标题 |
| `error` | 停止生成并展示错误 |
| `tool_call` / `tool_result` / `thinking` | Agent 模式下可展示过程，也可忽略 |

### 12. Agent 问答

```bash
curl -N 'http://localhost:8080/api/v1/agent-chat/session-0001' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "结合知识库和网络搜索回答这个问题",
    "agent_enabled": true,
    "agent_id": "builtin-smart-reasoning",
    "web_search_enabled": true,
    "knowledge_base_ids": ["kb-00000001"],
    "channel": "app"
  }'
```

Agent 模式会额外返回 `thinking`、`tool_call`、`tool_result`、`reflection` 等事件。

### 13. 停止生成

```bash
curl 'http://localhost:8080/api/v1/sessions/session-0001/stop' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{"message_id":"assistant-message-id"}'
```

`message_id` 使用 SSE 中的 `assistant_message_id`；如果 App 没有记录，可先用消息列表接口查最近一条 `role=assistant` 且 `is_completed=false` 的消息 ID。

### 14. 拉取会话消息

```bash
curl 'http://localhost:8080/api/v1/messages/session-0001/load?limit=20' \
  -H 'X-API-Key: sk-xxxxx'
```

分页向前拉取时传 `before_time`，值为当前列表最早一条消息的 `created_at`：

```bash
curl 'http://localhost:8080/api/v1/messages/session-0001/load?limit=20&before_time=2026-08-16T10%3A00%3A00%2B08%3A00' \
  -H 'X-API-Key: sk-xxxxx'
```

### 15. 推荐问题

回答完成后可让服务端生成后续推荐问题：

```bash
curl 'http://localhost:8080/api/v1/sessions/session-0001/messages/assistant-message-id/suggestions' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{"regenerate": false}'
```

读取结果：

```bash
curl 'http://localhost:8080/api/v1/sessions/session-0001/messages/assistant-message-id/suggestions' \
  -H 'X-API-Key: sk-xxxxx'
```

点击、曝光、关闭等事件上报：

```bash
curl 'http://localhost:8080/api/v1/sessions/session-0001/suggestion-events' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "suggestion_set_id": "set-001",
    "question_id": "q-001",
    "event_type": "click"
  }'
```

### 16. 混合检索

不生成回答，只返回命中的分块：

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/hybrid-search' \
  -H 'X-API-Key: sk-xxxxx' \
  -H 'Content-Type: application/json' \
  -d '{
    "query_text": "产品如何安装？",
    "vector_threshold": 0.3,
    "keyword_threshold": 0.1,
    "match_count": 10
  }'
```

常用于调试“检索不到”还是“模型回答不理想”。

## SSE 解析参考

微信小程序当前示例客户端使用 `wx.request` 获取完整响应文本后解析 SSE。核心逻辑是按空行分隔事件，再解析 `data:` 行：

```js
function collectAnswerFromSSE(raw) {
  return raw
    .split(/\n\n+/)
    .map(block => block.split(/\n/).find(line => line.startsWith("data:")))
    .filter(Boolean)
    .map(line => {
      try {
        return JSON.parse(line.replace(/^data:\s*/, ""));
      } catch {
        return null;
      }
    })
    .filter(item => item && item.response_type === "answer")
    .map(item => item.content || "")
    .join("");
}
```

真机 App 如果支持流式读取，可以边读边按同样规则处理，不必等请求结束。

## 文件访问

| 接口 | 鉴权 | 说明 |
| --- | --- | --- |
| `GET /files?file_path=...` | JWT；拒绝 API Key | 当前空间文件代理 |
| `GET /api/v1/knowledge-bases/{kb_id}/files?file_path=...` | JWT；拒绝 API Key | 访问共享知识库内图片等资源 |
| `GET /api/v1/files/presigned?...` | 签名 URL，无需登录 | IM 等无法加鉴权头的客户端访问图片 |
| `GET /r/{token}` | 短期授权 token | 资源授权链接 |

普通 App 调试知识库问答一般不需要直接调用文件代理；展示回答引用里的图片时按服务端返回 URL 访问。

## 嵌入渠道接口

如果 App 内嵌的是公开 Web Chat，而不是直接用 API Key，可走 embed 流程：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/embed/{channel_id}/exchange` | 用发布 token 换嵌入会话凭证 |
| `GET` | `/api/v1/embed/{channel_id}/config` | 获取嵌入配置 |
| `POST` | `/api/v1/embed/{channel_id}/sessions` | 创建嵌入会话 |
| `POST` | `/api/v1/embed/{channel_id}/knowledge-chat/{session_id}` | 嵌入知识库问答 |
| `POST` | `/api/v1/embed/{channel_id}/agent-chat/{session_id}` | 嵌入 Agent 问答 |
| `GET` | `/api/v1/embed/{channel_id}/messages/{session_id}/load` | 拉取嵌入会话消息 |

embed 接口使用发布 token 和 `X-Embed-Session`，不使用普通 `X-API-Key`。

## 常见错误

| HTTP | 常见原因 | 排查 |
| --- | --- | --- |
| 400 | JSON 格式错误、缺字段、multipart 头错误、URL 被 SSRF 校验拦截 | 检查 `Content-Type` 和请求体 |
| 401 | 缺少鉴权、API Key 错误、JWT 过期 | 先调 `/api/v1/auth/me` |
| 403 | API Key 能力不足、知识库不在白名单、JWT 角色不足 | 检查 key 的 `capabilities` 和 `knowledge_base_ids` |
| 404 | ID 不存在或当前空间不可见 | 确认 `kb_id`、`session_id`、`knowledge_id` |
| 409 | 文件或 URL 重复导入 | 使用已存在知识或删除后重试 |
| 429 | 公开接口限流 | 降低重试频率 |
| 500 | 服务端依赖异常 | 用 `X-Request-ID` 查服务端日志 |

JWT 用户无空间时会返回：

```json
{
  "error": "Workspace required",
  "code": "TENANT_REQUIRED"
}
```

## 全量接口入口

常用模块文档：

| 模块 | 文档 |
| --- | --- |
| 认证 | [`docs/api/auth.md`](./api/auth.md) |
| 空间 | [`docs/api/tenant.md`](./api/tenant.md) |
| 知识库 | [`docs/api/knowledge-base.md`](./api/knowledge-base.md) |
| 知识导入 | [`docs/api/knowledge.md`](./api/knowledge.md) |
| 会话 | [`docs/api/session.md`](./api/session.md) |
| 聊天 | [`docs/api/chat.md`](./api/chat.md) |
| 消息 | [`docs/api/message.md`](./api/message.md) |
| 智能体 | [`docs/api/agent.md`](./api/agent.md) |
| FAQ | [`docs/api/faq.md`](./api/faq.md) |
| 标签 | [`docs/api/tag.md`](./api/tag.md) |
| 系统、模型、MCP、数据源、IM | [`docs/api/README.md`](./api/README.md) |

运行服务后，最完整的字段 schema 和所有 200+ 端点以 Swagger 为准。
