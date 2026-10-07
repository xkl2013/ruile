<!--cover
badge: 可行性核查
title: 招生顾问 Agent 可行性核查
subtitle: 先用代码确认记忆模块有什么，再决定能做哪些 Agent
meta: 项目=睿乐大脑 · 服务模块|版本=V1（基于代码核查）|核查对象=记忆模块真实能力|结论=10 个岗位中 4 个已具备
kpi: 2=真实可用链路|3=已预置招生模板|7=凭空想象的缺口
-->

> **后续定稿**：本文的「可做清单」已进一步收敛，最终只保留 3 个必用 Agent —— 见 [招生顾问必用3个Agent.md](./招生顾问必用3个Agent.md)。

# 招生顾问 Agent 可行性核查

> 文档状态：可行性核查（基于代码，非设计推演）
> 更新日期：2026-10-06
> 适用项目：ruile
> 核查对象：记忆模块（Organize）及其与 Agent 的全部连接点
> 取代关系：本文是对 [早幼教招生顾问Agent矩阵.md](./早幼教招生顾问Agent矩阵.md) 的能力校验版；该文保留业务链路分析与合规红线，**Agent 清单以本文为准**

---

## 0. 结论先行

上一版矩阵的 10 个岗位是按**业务链路**推导的，没有验证记忆模块实际提供了什么。核查后结论如下：

**1. 平台只有两条真实可用链路，能做的 Agent 必须落在其中：**

```
链路 A：  记忆 ──▶ 整理任务（模板驱动） ──▶ 成果（带 [M#] 证据引用）
链路 B：  成果 ──▶ 指派到服务 ──▶ 服务专家运行时读取（+ 知识库检索）
```

**2. 10 个岗位重新分级：**

| 分级 | 数量 | 说明 |
| --- | --- | --- |
| ✅ 已具备（3 个预置模板已在库） | 3 | 线索跟进清单、渠道效果周报、沟通复盘——**招生场景已有现成实现** |
| ✅ 只差一条模板（后台建即可，无需写代码） | 4 | 到访复盘、价格异议、体验课复盘、入园适应跟进 |
| 🔶 需先补通道 | 2 | 依赖「记忆 ↔ 服务对象」关联，该关联**当前不存在** |
| ❌ 不应做（凭空想象） | 2 | 渠道投放分析、招生漏斗复盘——需要 Ruile 明确不接的外部数据 |

**3. 最重要的一条**：能落地的 Agent **不是"会读记忆的 Agent"**，而是**"把记忆整理成结构化成果的整理模板"**。平台当前**没有任何工具**让 Agent 直接读取记忆。

---

## 1. 记忆模块真实有什么（逐条附代码位置）

### 1.1 数据模型

| 项 | 事实 | 位置 |
| --- | --- | --- |
| 记忆表 | `organize_memories` | `migrations/versioned/000072_organize_section.up.sql:4-22` |
| 字段 | `id, tenant_id, user_id, kind, title, content, source, occurred_at, duration_seconds, metadata(jsonb), created_at, updated_at, deleted_at` | 同上 |
| `kind` 枚举 | `note` / `record` / `audio` / `audio_card` | 同上 `:18-19`；常量 `internal/types/organize.go:15-18` |
| AI 摘要与标签 | **不是独立列**，运行时写进 `metadata` 的 `summary`、`tags` | `internal/application/service/organize_memory_audio.go:456-457` |
| 归属 | **只有 `tenant_id` + `user_id`** —— 不存在 `service_id` / `subject_id` / 客户关联列 | `000072_organize_section.up.sql:4-22` |
| 附件 | `organize_memory_attachments`（含 `transcript` 转写、`content`、`status`） | `migrations/versioned/001011_organize_memory_attachments.up.sql:3-27` |

### 1.2 AI 处理链路（录音进来自动做了什么）

入口 `ProcessMemoryTranscribe`（`internal/application/service/organize_memory_audio.go:343`）：

1. ffmpeg 归一化音频
2. ASR 转写（`asrModel.Transcribe`，`:417`）
3. 调用 `generateMemoryRecordingNoteAIResult`（`:431`）生成结构化结果
4. 输出 JSON：`{title, summary, tags[], note_markdown}`（提示词硬编码在 `internal/application/service/organize_upload.go:290-302`）
5. 回写记忆（`:447-467`）

> ⚠️ **没有自动场景分类**，也没有自动识别"这条记忆属于哪个家长/哪个服务"。

### 1.3 一个记录型记忆长什么样

```
[M1] 标题：李女士咨询托班
类型：audio
时间：2026-10-06 10:20
来源：语音记录
内容：（转写全文，采样上限 2200 字符）
```

格式来自 `buildOrganizeJobPrompt`（`internal/application/service/organize_workbench.go:904-952`）——**这是记忆唯一进入 LLM 的通道**。

---

## 2. 记忆模块**没有**什么（这才是关键，避免凭空想象）

| # | 想象中的能力 | 代码现实 | 证据 |
| --- | --- | --- | --- |
| 1 | Agent 有工具能读记忆 | **不存在**。工具清单里没有记忆类工具，只有知识库/数据/技能/Wiki | `internal/agent/tools/definitions.go:11-35`、`:46-69`；`config/builtin_agents.yaml:197-201` |
| 2 | 记忆按客户/服务筛选 | **不存在**。`ListMemories` 只支持 `kind` + 关键词（title/content/source） | `internal/application/repository/organize.go:157-163` |
| 3 | `memory_filter` 控制专家读哪些记忆 | **字段存而不读**。全仓仅 5 处引用：1 处定义 + 1 处透传赋值 + 3 处默认值初始化，**零消费点** | `internal/types/service_space.go:171,191-192,268`；`internal/application/service/service_space.go:3734` |
| 4 | 工作档案目录（`work_doc_directory`） | **字段存而不读**。同上模式，仅定义 + 存库 | `internal/types/service_space.go:169,266` |
| 5 | 记忆自动关联服务对象 | **不存在**。服务对象只能人工创建；`service_subject_profiles` 由 facts 物化，**不来自记忆** | `internal/application/service/service_space.go:2871`、`:566,606` |
| 6 | 记忆路由器（自动分类业务域） | **不存在**。`CreateMemory` 是纯 CRUD，无钩子 | `internal/application/service/organize.go:94-123` |
| 7 | 记忆自动生成服务提醒 | **不存在**。服务提醒是纯人工创建，`title` 必填 | `internal/application/service/service_reminder.go:47-115` |
| 8 | 提醒上挂记忆证据 | **字段已建但无写入方**：`SourceMemoryIDs` / `Confidence` / `MemorySignals` / `AgentDomain` | `internal/types/service_reminder.go:39,54-59`；CreateInput `:104-117` 不接受 |
| 9 | Agent 把结果写回记忆 | **不存在**。`agent_run.go` 全文无 memory 引用 | — |
| 10 | 证据链表 `agent_work_doc_memory_links` | **只有建表，代码零引用** | `migrations/versioned/000078_service_module.up.sql:125-147` |

> **归纳**：第 3、4、8、10 项属于典型的"**字段/表建好了，但代码没接线**"。设计意图清楚，但**今天不能当作已有能力用**。

---

## 3. 真实可用的两条链路

### 3.1 链路 A：记忆 → 整理任务 → 成果

**机制**（`internal/application/service/organize_workbench.go:904-952`）：
把「模板指令 + Markdown 骨架 + 章节顺序 + 记忆原文」拼成一个提示词交给模型，输出要求里写死三条：

1. 输出可直接阅读的中文 Markdown
2. 严格按预设章节顺序和层级组织
3. **每条关键结论必须用 `[M1]` 形式标注来源；没有依据的字段明确写"记录中未提供"**
4. 待办用 `- [ ]` 清单
5. 无可用记忆时：**"不得生成虚构内容"**，改为输出缺少输入的说明

**这就是"不凭空想象"的机制保障**——平台自己在提示词里堵死了编造。

**模板已实现且可后台维护**（这是最重要的落地支点）：

| 项 | 事实 | 位置 |
| --- | --- | --- |
| 模板存 DB | `organize_templates` + `organize_template_versions` | `migrations/versioned/000106_organize_workbench.up.sql:3-49` |
| 模板字段 | Key / Name / **Scene** / Description / OutputLabel / DefaultInstruction / **MarkdownTemplate** / **ExpertIDs** / Spec / Status | `internal/types/organize_workbench.go:55-79` |
| 后台管理 API | `/system/admin/organize/templates`：列表 / 创建 / 详情 / 更新 / **compile** / **preview** / **publish** / disable / versions / **rollback** | `internal/router/router.go:1220-1229`，守卫 `g.SystemAdmin()` |
| 用户侧读取 | `GET /organize/templates`、`/templates/scenes`、`/templates/:key` | `internal/router/router.go:1506-1508` |

### 3.2 链路 B：成果 → 指派到服务 → 服务专家

| 步骤 | 事实 | 位置 |
| --- | --- | --- |
| 成果表 | `organize_outputs`：Content / Title / **Citations** / **AssignedServiceID** / AssignmentStatus | `internal/types/organize.go:190-228`；指派列见 `000116_organize_service_assignment.up.sql` |
| 看引用 | `GET /organize/outputs/:id/citation` | `internal/router/router.go:1549` |
| 指派到服务 | `POST /organize/outputs/:id/assign` → 写 `service_context_sources` + 生成 `service_facts`（FactType=`organize_output`，带 title/summary/fields/citations/**memory_ids**） | `internal/handler/organize.go:682`；`internal/application/service/service_space.go:1662-1716`、`:1719-1760` |
| 专家运行时读到 | 拼装 `[服务空间事实]`（FactType/FactKey/SourceID/Value）+ `[服务空间整理来源]`（SourceTitle + 摘要 + SourceContent），有字节预算与截断 | `internal/application/service/service_space.go:3356-3420` |

> **关键含义**：服务专家能读到的，**只是"已经整理并指派过来"的成果**，不是全部记忆，也不是原始录音。

### 3.3 已有 3 个招生模板（现成可用的证据）

`migrations/versioned/000106_organize_workbench.up.sql:168-179`：

| 模板 key | 名称 | 场景 | 指令要点 | 章节 |
| --- | --- | --- | --- | --- |
| `lead_followup` | **线索跟进清单** | 招生增长 | 按线索状态分组，标注流失风险与下一步动作，**48 小时内回访置顶** | 线索分组 / 风险判断 / 下一步动作 |
| `channel_report` | **渠道效果周报** | 招生增长 | 呈现渠道、到访数、转化数和单位成本；**没有数据的字段明确标注缺失，不得编造** | 渠道表现 / 数据缺口 / 下周动作 |
| `parent_review` | **沟通复盘** | 家长服务 | 问题类型、处理方式、当前状态和跟进时间，**需园所回应的排最前** | 问题类型 / 处理情况 / 待回应事项 |

**招生场景不是从零开始**——"线索跟进清单"就是一线招生顾问最需要的那个东西，而且已经预置好了。

---

## 4. 重排后的 Agent 清单

### 4.1 ✅ 已具备（3 个，改配置即可用于招生）

| Agent | 载体 | 现状 |
| --- | --- | --- |
| **线索跟进** | 模板 `lead_followup` | 已发布（`status=enabled`, `published_version=v1`） |
| **渠道周报** | 模板 `channel_report` | 同上 |
| **家长沟通复盘** | 模板 `parent_review` | 同上（场景标"家长服务"，招生同样适用） |

### 4.2 ✅ 只差一条模板 —— 建议立即补（在后台建，无需写代码）

这 4 个都落在链路 A 上，机制与已有模板**完全同构**，只是换场景、换章节、换指令。

| # | 建议 key | 名称 | 场景 | 输出标签 | 建议章节 | 指令要点 |
| --- | --- | --- | --- | --- | --- | --- |
| 4 | `campus_visit_review` | **到访接待复盘** | 招生增长 | 到访复盘 | 家长关注点 / 现场已答 / 未决问题 / 下一步动作 | 从到访记录中提取家长真实关注点与遗留疑问；未决问题必须列出由谁在何时回应 |
| 5 | `pricing_objection` | **价格异议台账** | 招生增长 | 异议台账 | 异议类型 / 应对方式 / 家长反应 / 待审批优惠 | 汇总价格类沟通；**所有优惠口径必须来自记录，不得自行设计优惠方案** |
| 6 | `trial_review` | **体验课复盘** | 招生增长 | 体验复盘 | 环节回顾 / 孩子表现 / 家长反应 / 转化判断 | 按记录还原体验过程；转化判断只依据记录中家长的原话与行为 |
| 7 | `adaptation_followup` | **入园适应跟进** | 家长服务 | 适应跟进 | 逐日表现 / 家长诉求 / 情绪与作息 / 明日动作 | 把适应期零散记录整理成按日跟进表；**家长焦虑点单独成节**，不得淡化 |

> 这 4 条建完，招生链路从"获客记录 → 跟进 → 转化 → 适应"就全部有整理能力支撑了。
> 每条的落地成本 ≈ 在 admin 后台填一个表单 + 预览 + 发布。

### 4.3 🔶 需先补通道（2 个，说明补什么）

| Agent | 卡在哪 | 需要补什么 |
| --- | --- | --- |
| **线索管家**（按客户聚合线索） | `organize_memories` 没有 `subject_id`，记忆无法归到某个家长名下 | 需要"记忆 ↔ 服务对象"关联（功能设计中称为记忆路由器，`记忆笔记驱动多Agent工作助理设计.md` §8.1，**代码未实现**） |
| **通话/微信记录归集** | 记忆无来源客户字段，只能靠关键词碰运气 | 同上；或先靠人工在记忆标题里写家长名，用关键词检索兜底 |

> 在通道补齐前，**"按家长维度的记忆聚合"做不了**。这不是提示词能解决的问题，是数据模型问题。

### 4.4 ❌ 不应做（凭空想象）

| 上一版提的 Agent | 为什么不能做 |
| --- | --- |
| **渠道专员**（投放 ROI 分析） | 需要各渠道投放成本、曝光、点击等外部数据。Ruile 定位**明确不接外部 CRM/广告系统**（`记忆笔记驱动多Agent工作助理设计.md` §4.2）。模板 `channel_report` 只能整理**用户自己记下来的**渠道记录 |
| **招生复盘**（转化漏斗） | 需要结构化的线索阶段数据。当前线索只存在于自然语言记忆里，没有阶段字段，算不出漏斗 |
| **入园适应顾问**（主动推每日反馈） | "主动推送"需要自动生成服务提醒，而服务提醒是纯人工创建的。只能降级为 4.2 的 `adaptation_followup` **整理模板** |
| **任何"自动写回 CRM / 自动发消息"** | 平台明确不做（`记忆笔记驱动多Agent工作助理设计.md` §4.2：不允许 Agent 绕过用户确认直接写外部系统） |

---

## 5. 与上一版的差异对照

| 上一版 Agent | 核查结论 | 处置 |
| --- | --- | --- |
| ① 招生顾问（主岗） | 可作为**服务专家**存在，输入 = 已指派成果 + 知识库 | 保留，但需明确它读不到原始记忆 |
| ② 线索管家 | 需先补记忆↔服务对象关联 | 降级为 4.3 |
| ③ 渠道专员 | 需要外部投放数据 | ❌ 移出 |
| ④ 邀约专员 | 可作为整理模板或纯知识库对话 | 降级为 4.2 类 |
| ⑤ 家长问答 | 走知识库（`knowledge_search`），**与记忆无关** | 保留，但归属"知识库类"而非"记忆类" |
| ⑥ 到访接待 | 转成 `campus_visit_review` 整理模板 | 改写为 4.2 |
| ⑦ 体验课设计 | 转成 `trial_review` 整理模板 | 改写为 4.2 |
| ⑧ 价格方案 | 转成 `pricing_objection`，且**不得自行设计优惠** | 改写为 4.2 |
| ⑨ 入园适应顾问 | 转成 `adaptation_followup` | 改写为 4.2 |
| ⑩ 老带新专员 | 需要转介绍数据；可作整理模板 | 待定（下一批） |
| ⑪ 招生复盘 | 需要结构化数据 | ❌ 移出 |

**净结果**：**3 个已具备 + 4 个建模板即可 = 7 个真实可落地**，3 个移出，1 个待定。

---

## 6. 一个必须讲清的天花板

**所有整理类能力的天花板 = 用户录了多少记忆。**

链路 A 的输入只有记忆原文。记忆里没有的东西，模型只能写"记录中未提供"——这是平台**刻意**的设计（`organize_workbench.go:935-937`）。

所以对一线招生顾问的真实价值主张应该是：

> 「你录进来的东西，我帮你整理成能用的清单、台账和复盘」——**而不是**「我帮你分析你没告诉我的事」。

这句话也直接约束了产品宣传口径：**不能承诺"AI 自动帮你盯着客户"**，因为盯人需要自动生成提醒，而那部分没有实现。

---

## 7. 建议的下一步（按性价比排序）

| # | 动作 | 产出 | 成本 |
| --- | --- | --- | --- |
| 1 | 用现成 `lead_followup` 模板，拿真实招生记忆跑一遍 | 验证链路 A 在招生场景的**实际质量** | 极低，立即可做 |
| 2 | 在后台新建 4.2 的 4 条模板（到访/价格/体验课/适应） | 招生全链路整理能力补齐 | 低，后台建表单，无需写代码 |
| 3 | 把 `channel_report` 的输入口径写进培训材料 | 让顾问知道"渠道数据要自己录" | 低 |
| 4 | 出「记忆 ↔ 服务对象」关联方案（记忆路由器） | 解锁线索管家、按家长聚合 | 中，需要改数据模型 + 一个异步任务 |
| 5 | 明确服务专家的知识库清单（课程/价格/师资/制度） | 支撑家长问答类专家 | 中，取决于园所配合 |

**第 1 步建议今天就做**——它用零开发成本验证了整条链路的真实可用性。

---

## 附：核查方法说明

本文所有"不存在"结论，均通过以下方式确认，非推测：

1. 全仓检索字段/表名（不限扩展名），确认引用点数量与性质（定义 / 写入 / 消费）
2. 逐个读取声称消费该字段的函数体，确认是否有实际读取逻辑
3. 对照 `docs/记忆笔记驱动多Agent工作助理设计.md` 的设计条款，标注"文档有、代码无"
4. 工具能力以 `internal/agent/tools/definitions.go` 的常量表为准，不以文档描述为准
