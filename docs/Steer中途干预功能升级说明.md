# Steer 中途干预功能升级说明

> 适用版本：当前睿乐实现（2026-10-11）
>
> 功能名称：Agent 运行中的用户消息中途干预（Steer）
>
> 适用对象：后端、前端和版本升级维护人员

## 1. 文档目的

本文记录当前已同步的 Steer 功能实现，作为后续从上游版本继续升级、冲突处理、联调和回滚的基线。

Steer 允许用户在 Agent 尚未完成时继续发送消息：

- `inject`：把消息注入当前 Agent 运行，在下一个 ReAct 轮次边界继续处理。
- `after`：把消息放入当前会话的待发送队列，当前运行完成后再开启下一轮。

这不是普通的“停止后重新发送”。当前运行的上下文、工具执行状态和历史消息需要继续保留。

## 2. 当前实现边界

当前版本的行为如下：

| 项目 | 当前行为 |
| --- | --- |
| 支持场景 | Agent 流式对话 |
| `inject` 消费时机 | Agent 下一次轮次开始前，或自然停止判断后发现有新消息时 |
| `after` 消费时机 | 当前运行完成后由前端自动提交下一轮 |
| 当前运行中的 LLM 调用 | 不会被强制中断；消息会等待当前调用或工具执行返回 |
| 最大排队数量 | 每个运行最多 10 条未消费消息 |
| 单条消息长度 | 最多 10000 个 Unicode 字符 |
| 附件 | 暂不支持中途干预时携带附件 |
| 消息持久化 | 注入前先创建普通 `user` 消息，原始内容落库 |
| 历史重放 | 通过 `user_message_injected` 和 Agent step 中的 `user_messages_before` 重建上下文分段 |
| 跨 API 实例 | Redis 模式通过 live-run 标记和独立队列支持 |
| 无活动运行 | 接口返回 `new_run`，前端降级为普通新消息发送 |

### 2.1 当前版本没有做的事情

当前版本没有由后端自动创建下一轮 Agent 运行。`after` 消息由浏览器在当前轮结束后调用普通发送接口提交。

后续如果上游增加“后端自动续跑”：

1. 必须先确认续跑的唯一责任方是后端还是前端。
2. 不能同时保留后端自动续跑和前端 `flushAfterSteerQueue()`，否则同一条消息可能发送两次。
3. 应保留 `status: new_run`、`already_injected` 等兼容状态，避免旧前端重复提交。

## 3. 代码清单

### 3.1 后端

| 文件 | 作用 |
| --- | --- |
| `internal/handler/session/steer.go` | Steer API、队列读写、幂等、消费状态和 live-run 查找 |
| `internal/types/steer.go` | Agent 与 Handler 之间的 `SteerSink` 接口、消息包装格式 |
| `internal/types/interfaces/stream_manager.go` | 独立 Steer 队列和 live-run 标记接口 |
| `internal/stream/memory_manager.go` | 本地内存模式的 Steer 队列和运行标记 |
| `internal/stream/redis_manager.go` | Redis 队列、Lua 去重、TTL 和跨实例运行标记 |
| `internal/agent/steer.go` | 在 Agent 轮次边界读取并持久化 Steer 消息 |
| `internal/agent/engine.go` | 注入时机、上下文追加、自然停止时的延迟注入处理 |
| `internal/agent/finalize.go` | 完成事件中保留已注入消息的历史关联 |
| `internal/application/service/session_agent_qa.go` | 将 `SteerSink` 注入 Agent Engine |
| `internal/handler/session/qa.go` | 创建 Steer sink、设置和清理 live-run 标记 |
| `internal/handler/session/agent_stream_handler.go` | 将注入确认写入可见 SSE 流 |
| `internal/event/event.go` | `EventUserMessageInjected` 事件类型 |
| `internal/event/event_data.go` | `UserMessageInjectedData` 事件载荷 |
| `internal/types/chat.go` | `steer` 和 `user_message_injected` 响应类型 |
| `internal/router/router.go` | 注册四个 Steer 路由 |

### 3.2 前端

| 文件 | 作用 |
| --- | --- |
| `frontend/src/api/chat/steer.ts` | Steer API 客户端和队列项类型 |
| `frontend/src/utils/steerId.ts` | 生成客户端 Steer ID |
| `frontend/src/utils/steerStreamFork.ts` | 乐观消息、消息确认、助手消息分段和历史重放 |
| `frontend/src/utils/chatSubmitShortcut.ts` | Enter、Cmd、Alt 快捷键语义 |
| `frontend/src/components/Input-field.vue` | 运行中输入、队列展示、立即注入、重试和删除 |
| `frontend/src/views/chat/index.vue` | 队列状态、API 调用、当前轮结束后的 `after` 刷新 |
| `frontend/src/composables/useChatStreamHandler.ts` | 处理 `user_message_injected` 并切分助手消息 |
| `frontend/src/i18n/locales/zh-CN.ts` | 中文提示文案 |
| `frontend/src/i18n/locales/en-US.ts` | 英文提示文案 |

## 4. 后端数据和事件模型

### 4.1 Steer 队列事件

Steer 不直接写入普通可见 SSE 流，而是写入与当前 assistant message 绑定的独立队列。

逻辑字段如下：

```text
session_id
assistant_message_id
steer_id
content
data.delivery              = "inject" | "after"
data.channel               = "web" ...
data.mentioned_items
data.consumed              = true | false
data.user_message_id       = 持久化后的普通 user 消息 ID
```

内存模式中，上述数据保存在 `MemoryStreamManager` 的 `steerEvents`。

Redis 模式默认使用以下 key：

```text
stream:events:<session_id>:<assistant_message_id>:steer
stream:events:<session_id>:live-run
```

如果配置了其他 Redis prefix，应将 `stream:events` 替换为实际 prefix。队列和 live-run 标记使用 StreamManager 的 TTL，默认 Redis TTL 为 24 小时。

### 4.2 消费流程

```text
浏览器提交 Steer
    |
    v
POST /sessions/:session_id/steer
    |
    v
独立 Steer 队列
    |
    +-- delivery=after  --------> 当前运行结束后由前端普通发送
    |
    +-- delivery=inject --------> Agent 下一轮边界 PollSteer
                                      |
                                      v
                            创建普通 user 消息
                                      |
                                      v
                           标记 consumed + user_message_id
                                      |
                                      v
                    追加 <steer_message> 到 Agent 上下文
                                      |
                                      v
                    EventUserMessageInjected -> SSE
                                      |
                                      v
                           前端切分助手消息展示
```

### 4.3 Agent 上下文格式

原始用户内容以普通 `user` 消息持久化。送入当前 Agent 上下文时，使用以下包装：

```text
<steer_message>
用户在运行中输入的内容
</steer_message>
<continue_task>
This is guidance for the task in progress. Apply it and continue unfinished work unless the user explicitly changes or cancels the task.
</continue_task>
```

这样可以区分“当前任务的继续指令”和新的独立会话问题。升级时不要直接修改历史中已持久化的原始 `user` 内容。

## 5. API 契约

接口前缀为 `/api/v1`。

### 5.1 提交消息

```http
POST /sessions/:session_id/steer
Content-Type: application/json
```

请求示例：

```json
{
  "query": "补充说明第二个方案的风险",
  "steer_id": "客户端生成的 UUID",
  "expected_assistant_message_id": "当前运行的 assistant message ID",
  "mentioned_items": [],
  "channel": "web",
  "delivery": "inject"
}
```

字段说明：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| `query` | 是 | 去除首尾空格后不能为空，最多 10000 个 Unicode 字符 |
| `steer_id` | 否 | 客户端 UUID。建议前端生成并重试时复用 |
| `expected_assistant_message_id` | 否 | 防止请求发送到已经切换的运行 |
| `mentioned_items` | 否 | 与普通聊天消息一致的引用项 |
| `channel` | 否 | 默认 `web` |
| `delivery` | 否 | `after` 或 `inject`，默认 `after` |

常见响应：

```json
{
  "success": true,
  "status": "queued",
  "steer_id": "uuid",
  "delivery": "inject",
  "assistant_message_id": "assistant-id"
}
```

`status` 语义：

| 状态 | 含义 | 前端处理 |
| --- | --- | --- |
| `queued` | 已进入队列 | 保留队列项，等待注入或下一轮发送 |
| `already_injected` | 相同 `steer_id` 已消费 | 移除队列项，但保留/转正已显示的消息 |
| `new_run` | 当前没有活动 Agent 运行 | 移除队列项并按普通消息发送 |

### 5.2 将排队消息立即注入

```http
POST /sessions/:session_id/steer/:steer_id/inject
```

该接口只将 `delivery` 更新为 `inject`。实际消费仍由 Agent 在下一个轮次边界完成。

### 5.3 查询队列

```http
GET /sessions/:id/steer
```

响应示例：

```json
{
  "success": true,
  "active": true,
  "assistant_message_id": "assistant-id",
  "items": [
    {
      "steer_id": "uuid",
      "content": "继续检查异常分支",
      "delivery": "after",
      "mentioned_items": []
    }
  ]
}
```

浏览器刷新或重新进入会话时，前端通过此接口恢复队列。

### 5.4 删除排队消息

```http
DELETE /sessions/:id/steer/:steer_id
```

常见响应状态：

| 状态 | 含义 |
| --- | --- |
| `deleted` | 成功删除尚未消费的队列项 |
| `already_injected` | 消息已经消费，不能再删除 |
| `gone` | 当前运行或队列已经不存在 |

## 6. 前端交互约定

### 6.1 快捷键

在 Agent 运行期间：

| 操作 | 行为 |
| --- | --- |
| Enter | 发送到 `after` 队列 |
| macOS Cmd+Enter | 发送到 `inject` |
| Windows/Linux Alt+Enter | 发送到 `inject` |
| Shift+Enter | 换行 |
| 输入框为空时 Enter | 注入队列中第一条可发送消息 |
| 输入法组合输入期间 Enter | 不触发 Steer |

非运行状态仍走普通聊天发送流程。

### 6.2 乐观展示和确认

`inject` 消息提交前会先在聊天记录中显示一条带 `_steerPending` 标记的用户消息。

服务端发出 `user_message_injected` 后，前端必须：

1. 根据 `steer_id` 或 `user_message_id` 找到消息。
2. 清除 `_steerPending`。
3. 关闭当前助手分段。
4. 在用户消息后创建或复用新的助手分段。
5. 将后续事件写入新的助手分段。
6. 从队列中移除已消费项。

不要仅通过消息数组尾部追加内容。中途干预可能发生在工具调用、答案流式输出或历史恢复过程中，需要使用 `steerStreamFork.ts` 中的分段逻辑。

### 6.3 `after` 队列

当前实现中，`after` 队列只在输入框中展示，当前轮完成后按顺序执行：

```text
找到第一条 delivery=after 且未失败的消息
    -> 删除服务端队列项
    -> 从前端队列移除
    -> 调用普通 sendMsg()
    -> 等待新一轮完成
    -> 递归处理下一条
```

删除服务端队列项是为了避免浏览器刷新后重复发送。若未来改成后端自动续跑，需要重新设计此处。

## 7. 从上游继续升级的推荐顺序

每次同步上游版本时，建议按以下顺序执行，不要先改前端。

### 第一步：确认上游变更范围

优先检查：

```bash
git diff <旧上游版本>..<新上游版本> -- \
  internal/handler/session/steer.go \
  internal/agent \
  internal/stream \
  internal/types \
  frontend/src/api/chat/steer.ts \
  frontend/src/utils/steerStreamFork.ts \
  frontend/src/components/Input-field.vue \
  frontend/src/views/chat/index.vue \
  frontend/src/composables/useChatStreamHandler.ts
```

同时搜索以下关键字：

```text
steer
user_message_injected
live-run
SteerSink
PendingSteerMessages
delivery
already_injected
```

### 第二步：先对齐后端契约

按以下顺序处理冲突：

1. `StreamManager` 接口。
2. 内存和 Redis 两种实现。
3. `SteerSink` 和 `AgentEngine`。
4. `EventUserMessageInjected` 及其数据结构。
5. Handler 和路由。
6. QA 生命周期中的 live-run 设置与清理。

接口变更完成后先运行：

```bash
go test -count=1 ./internal/agent ./internal/handler/session ./internal/application/service
```

### 第三步：再对齐前端数据流

按以下顺序处理冲突：

1. `frontend/src/api/chat/steer.ts`
2. `frontend/src/utils/steerId.ts`
3. `frontend/src/utils/steerStreamFork.ts`
4. `frontend/src/utils/chatSubmitShortcut.ts`
5. `Input-field.vue`
6. `views/chat/index.vue`
7. `useChatStreamHandler.ts`
8. 中英文 locale 文案

不要只解决 TypeScript 冲突。重点检查：

- 服务端 UUID 与客户端临时 UUID 是否能合并。
- `already_injected` 是否会造成重复气泡。
- `user_message_injected` 到达时是否能找到对应用户消息。
- 当前助手消息是否正确切分。
- 历史接口恢复后是否重复展开分段。
- `after` 队列是否被重复提交。

### 第四步：最后处理体验层

确认以下行为：

- 运行中输入框显示队列，而不是覆盖当前输入。
- 失败消息可以重试。
- 队列项可以删除。
- 第一条队列项可以通过按钮或快捷键立即注入。
- 当前输入非空时，立即发送按钮发送当前输入，而不是误操作队列项。
- 移动端不出现按钮文字溢出或队列遮挡输入框。

## 8. 部署顺序和兼容策略

### 8.1 推荐部署顺序

```text
1. 部署后端
2. 验证四个 Steer API 和 Redis key
3. 部署前端
4. 使用真实 Agent 运行验证 inject / after / reload
```

后端应先上线，因为新前端依赖新 API。旧前端连接新后端可以继续进行普通聊天；旧前端不会使用新的 Steer 能力。

### 8.2 混合版本注意事项

- 新后端产生的 `user_message_injected` 事件，旧前端应忽略未知事件或按普通事件降级。
- 新前端连接旧后端时，Steer API 可能返回 404；前端应保留普通发送兜底。
- 新旧后端共用 Redis 时，旧实例不会消费新的 `:steer` key，但不会破坏普通 SSE stream。
- 不要在升级期间删除 `stream:events:<session_id>:<assistant_message_id>:steer`，否则会丢失尚未消费的队列。
- 队列没有数据库迁移；Redis key 会按 TTL 自动过期。

## 9. 回滚方案

### 9.1 仅回滚前端

优先采用此方式：

1. 保留新后端。
2. 回滚前端到旧版本。
3. 等待现有 Steer 运行结束或队列 TTL 过期。
4. 再决定是否回滚后端。

新后端创建的普通 `user` 消息应保留，不要做数据库删除。旧前端可能无法还原分段，但不会影响已保存的会话内容。

### 9.2 回滚后端

只有确认没有正在使用的 Steer 队列后再执行：

1. 停止或等待 Agent 运行结束。
2. 检查 Redis 中是否仍有 `:steer` 和 `:live-run` key。
3. 先回滚前端，再回滚后端。
4. 不删除已经持久化的普通 user 消息。

如果必须立即回滚，遗留 Steer key 可以等待 TTL 过期。不要把 Steer 事件直接转换成普通 SSE 事件，避免旧版本误解析。

## 10. 验证清单

### 10.1 自动化验证

前端类型检查：

```bash
cd frontend
PATH=/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH \
  ./node_modules/.bin/vue-tsc --noEmit --pretty false
```

后端定向测试：

```bash
go test -count=1 \
  ./internal/agent \
  ./internal/handler/session \
  ./internal/application/service
```

前端完整测试：

```bash
cd frontend
PATH=/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/node/bin:$PATH \
  ./node_modules/.bin/tsx --test
```

格式检查：

```bash
git diff --check
```

全量前端测试如果存在既有失败，应记录失败用例名称和数量，不要把它们与 Steer 回归混为一谈。

### 10.2 手工验证

至少覆盖以下场景：

| 场景 | 预期 |
| --- | --- |
| Agent 运行中 Enter 发送 | 消息进入 `after` 队列 |
| Agent 运行中 macOS Cmd+Enter | 消息立即预览并在下一边界注入 |
| Agent 运行中 Windows/Linux Alt+Enter | 消息立即预览并在下一边界注入 |
| 注入期间正在执行工具 | 当前工具完成后继续处理 |
| 自然答案刚结束时注入 | 最多追加一次继续轮次，不丢消息 |
| 连续发送 10 条以上 | 第 11 条被拒绝 |
| 重复提交同一个 `steer_id` | 不创建重复 user 消息 |
| 注入后重复删除 | 返回 `already_injected`，不删除消息 |
| 浏览器刷新 | 队列可以恢复 |
| 当前运行完成后刷新 | `after` 队列只发送一次 |
| Redis 多实例 | 在非创建请求的实例上也能查询和提交队列 |
| 运行已结束后发送 | 返回 `new_run`，按普通消息发送 |
| 输入法组合输入 | Enter 不触发发送 |
| Shift+Enter | 只换行，不触发发送 |

## 11. 故障排查

### 11.1 API 返回 `new_run`

可能原因：

- Agent 已完成，live-run 标记已清理。
- Redis live-run key 已过期。
- 当前请求访问的 session 没有正在运行的 Agent。

处理方式：前端按普通消息调用 `sendMsg()`，不要继续向旧 assistant message 的 Steer 队列写入。

### 11. 队列显示存在，但 Agent 没有消费

检查顺序：

1. `stream:events:<session_id>:live-run` 是否存在。
2. live-run 中的 assistant message 是否仍未完成。
3. Steer key 是否使用了当前 Redis prefix。
4. 队列项 `delivery` 是否为 `inject`。
5. 队列项是否已经 `consumed=true`。
6. Agent 是否调用了 `SetSteerSink()`。
7. Agent 是否已经进入下一次轮次边界。

### 11.3 出现重复 user 消息

重点检查：

- 前端重试是否复用了同一个 `steer_id`。
- Redis `AppendSteerEvents` 是否仍使用原子 Lua 去重。
- `PersistSteerMessage` 是否在创建消息后成功写入 `consumed` 和 `user_message_id`。
- 重复请求是否正确返回 `already_injected`。

### 11.4 前端出现重复助手气泡

重点检查：

- 是否重复处理了 `user_message_injected`。
- 是否使用 `steerStreamFork.ts` 中的 `forkAfterInjectedUser()`。
- 历史恢复是否重复调用 `expandSteerForksInHistory()`。
- 是否把同一个 `assistant_message_id` 当成了多个独立持久化助手消息。

### 11.5 Redis 队列数据异常

可检查：

```bash
redis-cli GET "stream:events:<session_id>:live-run"
redis-cli LRANGE "stream:events:<session_id>:<assistant_message_id>:steer" 0 -1
```

生产环境执行前应确认 Redis ACL、DB 和 prefix 配置，避免误查其他环境。

## 12. 后续升级的兼容性要求

后续修改 Steer 时应保持以下约束：

1. `steer_id` 必须保持客户端可生成、服务端可复用的 UUID 语义。
2. 重试不能创建第二条逻辑相同的用户消息。
3. 已消费事件不能再次被删除或重新注入。
4. `delivery=after` 和 `delivery=inject` 的责任边界必须保持明确。
5. 原始 user 消息内容和模型上下文包装内容不能混用。
6. `user_message_injected` 必须包含 `steer_id`、`content`、`message_id` 和 `user_message_id`。
7. 运行标记清理必须按 assistant message ID 校验，不能误清理新一轮运行。
8. Redis 和内存 StreamManager 必须同时实现新增接口。
9. 前端历史重放必须兼容没有 Steer 事件的旧消息。
10. 后端上线顺序必须早于依赖新接口的前端上线。
