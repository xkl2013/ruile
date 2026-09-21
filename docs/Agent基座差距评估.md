# Agent 基座差距评估

## 睿乐大脑（ruile）对标 WorkBuddy 的「已具备 vs 缺口」逐项核算

编制：阿墨　｜　日期：2026 年 9 月 21 日

---

## 编制说明

**评估对象**：`/Users/admin/Desktop/workspace/ruile`（基于 WeKnora 改造的睿乐大脑，Go 1.26 + Vue 3.5 + 小程序 + Flutter + mcp-server）

**对标基准**：WorkBuddy（本次评估正在运行的那个环境，通用型 Agent 产品）

**方法**：反向记账。不正向列需求，而是先把对标产品的能力面拆成可枚举的元素，逐项回代码里找实现，把「已具备」按「从零开发要多少人日」折算出来，再列缺口估工，两者相比得占比。**所有结论以代码为证据，不采信 README 与文档**——本次盘点中已发现文档与代码不一致之处。

**工作量口径**：熟悉该代码库的前后端各 1 名工程师的净开发时间，不含测试、联调与排期损耗。给区间不给点值，区间宽度反映不确定性。

**范围界定**：只评 Agent 基座能力面——运行时内核、工具系统、技能与扩展生态、记忆与身份、自动化、交互层、多模态与产出、企业治理。**知识库本身的检索与解析深度不计入差距**，那是本项目的原生强项，不属于「追赶」范畴。

**盘点时点**：2026 年 9 月 21 日。所有行号对应此时代码状态，进入实施前建议复核。

---

## 一、结论

### 核心数字

| 项 | 数值 |
| 缺口工作量 | 288–510 人日 |
| 已具备工作量 | 325–462 人日 |
| 缺口占比 | 62%–157% |
| 中位占比 | 约 100% |

**一句话结论：这不是「补几个功能」，是「把已经付过的开发量，再付一遍」。**

### 四条支撑

**1. 零件级差距不大，装配级差距很大。**

后端该有的零件基本都在：ReAct 引擎、Docker 沙箱（`--cap-drop ALL` / `--network none` / `pids-limit`）、MCP 客户端与 OAuth 全套、审批门、专家包体系、多租户 RBAC、SSO、计费、Langfuse 全链路追踪。缺的不是零件，是**几类成体系的能力**——文件与终端工具、跨会话记忆、子 agent、自动化调度、多模态生成。这几类每一项都是「一条完整链路」，不是「一个功能点」。

**2. 两者不是一个物种，差距是结构性的。**

WorkBuddy 是「个人生产力 agent」：读写文件、跑终端、记长期记忆、派生 subagent、生成图片视频、把成果发布成在线链接。

ruile 是「企业知识库 agent 平台」：知识库检索与解析、专家包分发、IM 渠道接入、多租户与计费。

对照下来，**ruile 在 IM 渠道、计费体系、租户治理、知识库深度四项上反过来超过 WorkBuddy**——WorkBuddy 一个 IM 渠道都不接，也没有计费与租户概念。所以正确的问题不是「差多少」，而是「要不要往那个方向靠」。

**3. 最划算的投入不在功能，在可举证。**

治理五项（工具调用全量留痕、内置工具审批门、审批状态与决定落库、审计 action 扩展、默认值翻转）合计 **40–70 人日**，只占总缺口约 15%。但它是面向教培交付的**合规硬门槛**——agent 会碰家长与学生信息，出事时拿不出「谁让它干的」这条记录，损失的是机构信任。且审计表（`audit_log`，迁移 000044）与审批门都已存在，这五项属**接线**不属**造件**。

**4. 有一个不能忽略的乘数：技术债。**

前端 25 个 `.vue` 文件超过 1500 行、19 个超过 2000 行。其中 `WikiBrowser.vue` 6048 行、`AgentEditorModal.vue` 5928 行、`FAQEntryManager.vue` 5912 行、`Input-field.vue` 3720 行、`AgentStreamDisplay.vue` 3650 行。**而这个结构上加功能的边际成本会持续上涨**——下面分档里要改的交互层，恰好全落在最大的两个文件里。

---

## 二、现状盘点

### 2.1 运行时内核

| 能力 | 状态 | 证据与说明 |
| ReAct 主循环 | 已具备 | `agent/engine.go:386` 起；think→analyze→act→observe 四步，默认 20 轮 |
| 多工具并行执行 | 已具备 | `act.go:190` errgroup，需 ParallelToolCalls 且工具数 ≥2 |
| 上下文压缩 | 已具备 | `memory/consolidator.go:79` LLM 摘要 + `token/compress.go:19` 裁剪 |
| 流式输出 | 已具备 | `think.go:31` ChatStream |
| 模型降级与重试 | 已具备 | `const.go:22/26/29`，单次 LLM 120s、单工具 60s、retry 2 |
| 多模型分流 | 缺失 | 单 chatModel 注入（`engine.go:39`），无按任务难度路由 |
| 计划模式 | 缺失 | 引擎内无；仅专家工作流有 Planner 阶段且不等待确认 |
| 中断后重定向 | 缺失 | 只有 ctx cancel（`engine.go:388`），无「打断并改需求」入口 |
| 断点续跑 | 缺失 | 无 checkpoint/resume；仅工作流 ask_user 可 ResumeWaiting |
| 子 agent 派发 | 缺失 | 无任何 spawn 工具；`AgentRun.ParentRunID` 是专家追问链不是子 agent |
| 生命周期钩子 | 缺失 | 无用户可插的 before/after hook；`registry.go:66` WrapTool 是内部防劫持机制 |

### 2.2 工具系统

已具备 24 个内置工具（注册于 `agent_service.go:594-682`，名称常量见 `tools/definitions.go:9-34`），另加 MCP 动态注册。

| 能力 | 状态 | 证据与说明 |
| 知识库检索组 | 已具备 | `knowledge_search`、`grep_chunks`、`query_knowledge_graph`、`wiki_search`、`faq_snippet` |
| Wiki 读写组 | 已具备 | `wiki_read_page`、`wiki_write_page`、`wiki_replace_text`、`wiki_rename_page`、`wiki_delete_page` |
| Web 检索 | 已具备 | `tools/web_search.go`、`web_fetch.go`（含 chromedp headless 取页） |
| 任务清单 | 已具备 | `tools/todo_write.go:12`，前端 `PlanDisplay.vue` 有对应展示 |
| 命名空间隔离 | 已具备 | `mcp_{service}_{tool}`，与内置工具前缀隔离，重名 first-wins |
| 文件系统工具 | 缺失 | 无独立 Read/Write/Edit；`wiki_write_page` 只能操作知识库页 |
| 文件搜索 | 缺失 | 无 Glob/Grep；`grep_chunks` 是知识库分块检索，不是文件系统 |
| 终端执行 | 缺失 | `execute_skill_script.go:65` 只能跑技能目录白名单脚本 |
| 浏览器交互 | 缺失 | chromedp 只做导航取 HTML，无点击、表单、截图 |
| 子 agent 工具 | 缺失 | 无 |
| 绘图与可视化 | 缺失 | 全仓无 image_gen / widget 类实现 |
| 工具级权限 | 缺失 | 只有全局白名单（`definitions.go:74`），无风险等级 |

### 2.3 技能与扩展生态

| 能力 | 状态 | 证据与说明 |
| SKILL.md 加载 | 已具备 | `skills/loader.go:31`，含三级渐进披露（`skills/skill.go:37`） |
| 技能带可执行脚本 | 已具备 | `skill_execute.go:65` → sandbox，默认禁网 |
| MCP 客户端 | 已具备 | 支持 sse / http-streamable，stdio 三处硬禁用 |
| MCP OAuth | 已具备 | PKCE + state 单次消费 + AES-256-GCM 存储 + 按 principal 分键 |
| 专家包体系 | 已具备 | `types/expert_package.go:20-121`，含 import / publish / bind |
| 专家选用界面 | 已具备 | `frontend/src/views/chat/index.vue:1308` `runPublishedExpert` |
| 反向 MCP Server | 已具备 | `mcp-server/`，把自身 API 暴露为 MCP——**WorkBuddy 无此项** |
| 技能自动创建 | 缺失 | 无写回 SKILL.md 的通路 |
| 技能自我改进 | 缺失 | 无 |
| 技能市场与分发 | 缺失 | `router.go:1425` 仅注释「未来 upload」 |
| 专家包签名与版本门禁 | 缺失 | `PublishVersion` 只查 blocking 与 definitions 非空 |
| MCP 市场与内置种子 | 缺失 | `BUILTIN_MCP_SERVICES.md` 要求手工 SQL 插入，代码无种子 |

### 2.4 记忆与身份

| 能力 | 状态 | 证据与说明 |
| 会话内记忆 | 已具备 | `memory/consolidator.go:79` 单次 Execute 的 messages 压缩 |
| 多会话管理 | 已具备 | sessionID + `agent_history.go:65` |
| run 状态持久化 | 已具备 | AgentRun 表 + 事件序列（`types/agent_run.go:88`） |
| 跨会话记忆检索 | 缺失 | 无 recall / embedding 记忆索引，历史靠 DB 重建 |
| 长期项目记忆 | 缺失 | 无 MEMORY.md 类文件契约 |
| 每日工作日志 | 缺失 | 无 |
| 用户画像建模 | 缺失 | `ServiceMemoryExtraction` 是客服业务记忆，非使用者建模 |
| 人格文件 | 缺失 | 无 SOUL / IDENTITY / USER 类机制 |

### 2.5 自动化与渠道

| 能力 | 状态 | 证据与说明 |
| IM 渠道接入 | 已具备 | 企微 / 飞书 / 钉钉 / 微信 / QQ / Slack / Telegram / Mattermost 八路 |
| IM 入站触发 agent | 已具备 | `router.go:1596` `/im/callback/:channel_id` |
| cron 基础设施 | 已具备 | `datasource/scheduler.go:44`，但只用于数据源同步与卡死行清扫 |
| 定时任务 | 缺失 | 用户无法配置定时跑 agent，无 API 无前端入口 |
| 事件触发执行 | 缺失 | 事件总线全仓仅一处 Emit（`handler/session/qa.go:801`） |
| 出站 webhook | 缺失 | webhook 仅企微入站回调 |
| 工作流编排引擎 | 缺失 | 无 temporal/cadence；expert workflow 是硬编码交付流程 |

### 2.6 交互层

| 能力 | 状态 | 证据与说明 |
| 思考块 | 已具备 | `AgentStreamDisplay.vue:38-57`，可折叠、流式追加 |
| 工具调用三态 | 已具备 | `action-pending` / `action-error`，状态更新见 `useChatStreamHandler.ts:729` |
| 引用角标与来源抽屉 | 已具备 | `utils/citationMarkdown.ts:145-197` + `ChatReferencesDrawer.vue` |
| 审批交互卡片 | 已具备 | `ToolApprovalCard.vue:14-20`，可批准 / 拒绝 / 改参 |
| 产物预览面板 | 已具备 | `ExpertArtifactPanel.vue`，HTML/PDF/图/音视频 |
| 停止按钮 | 已具备 | `Input-field.vue:2670` |
| 多会话管理 | 已具备 | `menu.vue` 按日期分组、重命名、导出 markdown |
| @ 引用与附件 | 已具备 | `MentionSelector.vue` + `AttachmentUpload.vue`（9 种格式上传即解析） |
| 全局命令面板 | 已具备 | `GlobalCommandPalette.vue` ⌘K |
| 多语言 | 已具备 | zh-CN / en-US / ru-RU / ko-KR 四语 |
| 运行状态条 | 部分具备 | 有轮次 / 工具数 / 耗时，无 token 用量 |
| 工具参数展开 | 部分具备 | `ToolResultRenderer` 13 种 display_type，但 raw arguments 被刻意隐藏 |
| Office 在线预览 | 部分具备 | `document-preview.vue` 支持 docx/pptx/xlsx，但只服务 organize 模块，聊天产物仅能下载 |
| 子 agent 视图 | 缺失 | 全 src 无 subagent 命中 |
| 内联可视化 | 缺失 | 无流式 SVG / widget |
| 重新生成与编辑 | 缺失 | i18n 有 `chat.regenerate` 键但无组件引用 |
| 斜杠命令 | 缺失 | 仅 organize 编辑器有，聊天输入区无 |
| 语音输入 | 缺失 | 无 getUserMedia / SpeechRecognition；ASR 仅用于知识库音频解析 |

### 2.7 多模态与产出

| 能力 | 状态 | 证据与说明 |
| 图片理解 | 部分具备 | `engine.go:262/612` Images 字段 + `:690` VLM 描述 |
| 文生图 / 视频 / 3D | 缺失 | 全仓无实现 |
| Office 生成 | 缺失 | Go 侧无 docx/xlsx/pptx 实现；产物枚举有值但只产出 text/report/html |
| 表格只读分析 | 部分具备 | `data_analysis` 走 DuckDB，只读 |
| 在线链接发布 | 缺失 | 无 |

### 2.8 企业治理（本项目的优势区）

| 能力 | 状态 | 证据与说明 |
| Docker 沙箱 | 已具备 | 非 root、cap-drop ALL、network none、pids-limit 100、no-new-privileges |
| 命令校验器 | 已具备 | `sandbox/validator.go:52-164`，覆盖反弹 shell、参数注入、stdin 注入 |
| RBAC | 已具备 | owner / admin / contributor / viewer 四档（`types/tenant_member.go:22-32`） |
| SSO / OIDC | 已具备 | `config/config.go:334` + compose 内置 Dex |
| API Key | 已具备 | `types/tenant_api_key.go`，17 个能力档 |
| 计费 | 已具备 | pricing / plan / subscription / credit / points 全套——**WorkBuddy 无此项** |
| 租户隔离 | 已具备 | 全层 tenant_id |
| 审批门 | 部分具备 | 只覆盖 MCP 外部工具，内置写类工具无门 |
| 审计 | 部分具备 | 表结构与查询接口完整，但无 agent/tool 类 action |

### 2.9 已具备能力的反向折算（分母）

按「从零开发要多少人日」折算已具备部分：

| 后端模块 | 从零开发量 |
| Agent 引擎与提示词体系 | 25–35 |
| 工具系统 | 20–28 |
| 技能系统与沙箱执行 | 10–14 |
| Docker 沙箱与命令校验器 | 12–18 |
| MCP 客户端与 OAuth 全套 | 18–26 |
| 专家包体系 | 15–22 |
| 审批门 | 8–12 |
| 多租户 / RBAC / API Key / SSO | 18–25 |
| 会话与 run 持久化 | 12–16 |
| 计费体系 | 15–20 |
| IM 渠道接入（八路） | 20–30 |
| Langfuse 全链路观测 | 6–8 |
| 队列与限流 | 10–14 |
| 知识库工具化层 | 8–12 |
| **后端小计** | **197–280** |

| 前端模块 | 从零开发量 |
| 聊天交互与流式组件体系 | 40–55 |
| 设置 / 租户 / 成员 / 模型 / MCP 管理 | 30–40 |
| 服务与专家工作台 | 35–50 |
| 多语言四语 | 8–12 |
| 小程序与 Flutter 端 | 15–25 |
| **前端小计** | **128–182** |

**已具备合计：325–462 人日。**

### 2.10 工程成熟度信号

- 后端 Go 1.26，分层清晰（agent / sandbox / mcp / middleware / container / router）。
- 前端 Vue 3.5 + Pinia + TS 覆盖率 92%（163/177 个 `.vue` 含 `lang="ts"`），50 个单测文件。
- 部署成熟：Docker / Compose（30+ 服务）/ Helm 全套；支持 Ollama 与 vLLM 纯离线；Wails 桌面端；WeKnora-lite 单二进制零依赖。
- 观测完整：Langfuse 覆盖 `agent.execute` → `agent.round.N` → `agent.tool.*` 三层。
- 知识库底座强：10 种向量后端、10 种文档解析格式、混合检索 + rerank + FAQ + 知识图谱。

这四条说明一件事：**这个基座不是半成品，是一个已经成型的企业级产品。**

---

## 三、缺口清单与逐项成本

| 缺口 | 人日 | 为什么是这个量级 |
| 子 agent 派发与并行团队 | 25–45 | 引擎要改派生与生命周期、子 run 独立预算、权限只收窄不可扩大、留痕挂父 run，另需前端视图 |
| 跨会话记忆系统 | 20–35 | 存储 + 索引（FTS 或向量）+ 检索注入 + 自动写入策略 + 可见性界面 |
| 多模态生成 | 20–40 | 图 / 视频 / 3D 三类模型接入 + 产物存储 + 前端渲染，三件都要新做 |
| 技能自动创建与自我改进 | 20–35 | 要开写回 SKILL.md 的通路 + 质量门禁 + 冲突与版本处理 |
| 定时任务与自动化 | 18–30 | 任务模型 + CRUD API + 前端管理页 + 投递渠道 |
| Office 产出与在线预览 | 18–30 | 文档生成 + 聊天内预览（现有预览只服务 organize 模块） |
| 断点续跑 | 15–25 | 需 run 级状态快照与恢复语义，牵动会话模型 |
| 浏览器交互 | 12–22 | 现有 chromedp 只取 HTML；点击、表单、截图全是新做 |
| 技能与专家包签名门禁 | 12–20 | 哈希与发布者记录 + 发布前一致性校验 + 回滚机制 |
| 计划模式与权限模式 | 12–20 | 计划产物、确认协议、模式切换的状态机 |
| 预算闸 | 10–18 | token / 轮次 / 时长 / 并发四类配额的计量、告警与超限中断 |
| 重新生成与编辑消息 | 10–18 | 会话树要支持改写与分叉，牵动持久化结构 |
| MCP 市场与内置种子 | 10–16 | 目录模型 + 种子数据 + 管理端 |
| 提示注入防护升级 | 8–14 | 外部内容隔离声明 + 可疑模式检测 |
| 文件系统工具组 | 8–14 | 工具本身不难，难在与沙箱 / 白名单 / 展示类型的对接 |
| 内联可视化 | 8–15 | 流式渲染 + 沙箱化 + 明暗主题适配 |
| 发布为在线链接 | 8–15 | 多租户下的站点托管与鉴权 |
| 工具调用全量留痕 | 8–14 | 审计表与查询接口已在，属接线 |
| 内置工具审批门与落库 | 8–14 | 审批门已在，属接线 |
| 工具级权限与默认值翻转 | 8–14 | 三档风险等级 + 六处 default-open 反转 |
| 生命周期钩子 | 8–14 | 挂载点设计 + 用户配置 + 与审批审计的顺序约定 |
| Bash / 终端工具 | 6–12 | 工具本身简单，代价在强绑沙箱 + 强制审批 + 强制留痕 |
| 斜杠命令与语音输入 | 6–12 | 命令注册表 + ASR 接入 |
| **缺口合计** | **288–510** | — |

**缺口占比 = 288–510 / 325–462 = 62%–157%，中位约 100%。**

一个必须点出的对照：**本项目已有的聊天渲染层，代码量比对标产品的整个参照实现更大**。前端 `src` 共 20.0 万行，其中交互层相关组件（`AgentStreamDisplay` 3650 + `Input-field` 3720 + `chat/index` 2018 + `ToolResultRenderer` 与其 13 个渲染器）已远超对标形态本身的规模。意思是——**不是没做，是做在了另一条线上。**

---

## 四、分档成本结论

| 档位 | 人日 | 换来什么 |
| 档一 · 可交付 | 40–70 | 治理五项 + token 用量展示 + 参数展开。合规门槛过关，agent 行为可举证 |
| 档二 · 用着像 | 125–225 | 档一 + 文件工具组 + Bash + 跨会话记忆 + 身份人格层 + 定时任务 + 重新生成 + 斜杠命令 + 内联可视化 |
| 档三 · 完全对齐 | 288–510 | 全量对齐通用 Agent 能力面 |

**推荐档二。**

**档一的构成**：工具调用留痕 8–14、内置工具审批门 8–14、工具级权限与默认值翻转 8–14、预算闸 10–18、状态条补 token 与参数展开 5–10。合计 40–70 人日。

**推荐档二的理由**：档一只是「合规可交付」，使用者体感不到智能体变强——它解决的是「出事能举证」，不解决「好不好用」。到了档二，四条最直接的体感才出现：**能读本地文件、能记住上次的事、能自己定时跑、能重来一次**。这四条恰好也是教培场景里最常用的四件事。

**明确不推荐档三的理由**：档三比档二多出的 160–285 人日，买的是多模态生成、子 agent 并行、Office 产出、在线链接发布、MCP 市场。这些与「园所 / 教培知识库 agent」的定位关系不大，属于**因为对标产品的清单上有所以也要有**的过度投入。按上一份报告的原则：不要被「完整复刻」这个目标绑架——你的差异化在园所与教培的垂直知识，不在通用能力。

**分档边界说明**：档位的划分依据是「使用者能感知到什么」，不是「功能清单有多长」。所以档一与档二的分界是**体感线**，而不是**风险线**——治理五项虽然排在档一，但它是交付前置条件，不可推迟到档二。

---

## 五、风险与前置动作

**1. 技术债必须先还，且不能并进功能里算。**

25 个 `.vue` 超过 1500 行、19 个超过 2000 行。而档一档二要改的交互层，恰好全落在 `AgentStreamDisplay.vue`（3650 行）与 `Input-field.vue`（3720 行）这两个最大的文件里。**建议先拆这三个组件（含 `chat/index.vue` 2018 行）再动手**——否则每加一项功能的实际成本都会高于本报告估算。这一条决定了后续所有改动的成本曲线，所以单独立项，不计入上面的人日。

**2. 已确认一处数据链路断点。**

`reflection` 事件在后端 `types/chat.go:94` 已定义，前端 `useChatStreamHandler.ts:909` 只在过滤列表里提了一句，switch 无对应 case——**事件被静默丢弃**。这类断点要按「打通全链路」估工，不能按「做个 UI 组件」估。盘点中其他事件（thinking / tool_call / tool_result / references / approval）链路均完整。

**3. 已确认一处文档与代码矛盾。**

`docs/MCP功能使用说明.md` 明确写支持并推荐 Stdio，而代码在三处硬禁用（`mcp/client.go:216`、`manager.go:65`、`mcp_service.go:41`）。运维照文档配置会配不通，也可能在审计时成为「承诺未兑现」的证据。

**4. 多语言乘数。**

四种语言（zh-CN / en-US / ru-RU / ko-KR），每段新增 UI 文案要填 4 份。本报告人日按「不重复计语言」口径给，实施排期时需显式留出。

**5. 法律边界。**

交互形态属于通用设计模式，可以借鉴；代码、图标、视觉资产、文案不能照搬。这条在「照 WorkBuddy 的样子做」这类需求下，必须写进合同或交接说明。

**6. 建议动作顺序。**

1. 治理五项（合规门槛，且全是接线工作，风险最低）
2. 拆三个巨型组件（降低后续所有改动的成本）
3. 按档二逐项补能力（先记忆与文件工具，这两项的体感最强）

---

## 六、证据来源

**代码范围**：`internal/agent/`（含 `approval/`、`token/`、`tools/`、`memory/`、`skills/`）、`internal/sandbox/`、`internal/mcp/`、`internal/middleware/`、`internal/container/`、`internal/router/`、`internal/types/`、`internal/event/`、`internal/im/`、`internal/datasource/`、`internal/billing/`、`frontend/src/`（含 `views/chat/`、`components/chat/`、`composables/`、`api/`）、`miniprogram/`、`mobile/`、`mcp-server/`、`migrations/versioned/`、`go.mod`、`frontend/package.json`。

**文档范围**（仅作交叉核对，不作结论依据）：`docs/MCP功能使用说明.md`、`docs/BUILTIN_MCP_SERVICES.md`、`docs/agent-skills.md`、`docs/外部Agent包兼容与管理员发布路线.md`、`docs/worker-pool-governance.md`。

**方法**：先按「能力是否存在」通读代码并记录文件与行号，再按「默认值朝哪边倒」复核每一处开关，最后与文档描述交叉比对。

**边界说明**：本文档为评估报告，**未修改任何代码、迁移或前端**。人日估算为工程经验判断，非精确测量；区间中值可作为排期参考，不应作为报价依据。

---

## 七、一句话总结

**这个项目已经付过约 325–462 人日的开发量，追上 WorkBuddy 的通用能力面还要再付 288–510 人日——缺口约等于已具备量本身。但真正的问题不是缺口大小，而是方向：ruile 在企业治理、IM 渠道、计费与知识库深度上已经反超 WorkBuddy，缺的只是「个人生产力 agent」那一整套。花 40–70 人日把治理补上就能合规交付，花 125–225 人日能做出「用着像」的体感；剩下那 160–285 人日买的是与定位无关的通用能力，建议明确不投。**
