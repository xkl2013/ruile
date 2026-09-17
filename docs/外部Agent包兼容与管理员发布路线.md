# 外部 Agent 包兼容与管理员发布路线

> 文档状态：技术实施附录  
> 更新日期：2026-09-16  
> 适用项目：ruile  
> 主文档：[可扩展专家 Agent 服务平台架构设计](./服务Agent与卡片报告架构设计.md)  
> 本文范围：说明外部 Agent 包如何由后台管理员导入、适配、评测、发布和回退；普通用户不具备导入或安装 Agent 的权限

## 1. 核心结论

ruile 已具备 Agent 执行的主要底座，不需要重新建设 Agent 引擎。但当前不能把任意外部 Agent 包直接复制到项目中运行，也不应开放普通用户上传或安装 Agent。

正确定位是：

```text
后台管理员维护 Agent
  -> 多来源 Agent 包导入
  -> Source Adapter 转换
  -> ruile Expert Package
  -> 安全扫描与能力检查
  -> Agent Markdown Compiler
  -> Agent/Skill Registry
  -> Agent Evaluation
  -> 管理员发布
  -> 绑定服务领域/工作画像
  -> 普通用户使用
```

WorkBuddy 只是其中一种可适配来源。后续还可以适配其他开源 Agent 包、企业内部 Agent 包和 ruile 原生 Agent 包。平台内部只面向统一的 `ruile Expert Package`、统一权限、统一输出和统一产物模型。

## 2. 权限边界

| 角色 | 能力 | 限制 |
| --- | --- | --- |
| 平台管理员 | 导入 Agent 包、解析、扫描、编译、评测、发布、停用、回退 | 不能绕过安全扫描和发布门禁 |
| 租户/工作空间管理员 | 启用已发布 Agent、绑定工作画像、设置可见范围 | 不能上传外部包或修改 Agent 定义 |
| 普通用户 | 使用已启用 Agent、生成卡片和产物、追问、纠偏、保存结果 | 不能导入、安装、编辑 Agent 或扩大工具权限 |

如果当前后台暂时只有系统管理员角色，可以先由系统管理员承担前两类职责。数据模型仍应按上述边界设计，避免后续权限拆分时推翻架构。

## 3. 多来源适配模型

不要把 WorkBuddy manifest 或任何外部格式作为 ruile 的永久内部模型。推荐三层结构：

```mermaid
flowchart LR
    A["外部 Agent 包"] --> B["Source Adapter"]
    B --> C["ruile Expert Package"]
    C --> D["Agent Markdown Compiler"]
    D --> E["Agent/Skill Registry"]
    E --> F["管理员发布"]
    F --> G["用户使用"]
```

### 3.1 支持来源

首批可以支持：

- ruile 原生 Agent 包。
- WorkBuddy 风格 Agent 包。
- 企业内部 Agent 包。
- 后续其他开源社区 Agent 包。

所有来源都必须转换为同一个内部模型：

```yaml
api_version: ruile.ai/v1alpha1
kind: ExpertPackage
metadata:
  id: consulting-partners
  version: 3.0.0
  display_name: 战略咨询顾问
  source_format: workbuddy
  source_uri: admin-upload
  license: unknown

spec:
  agents:
    - id: consulting-partner
      definition: agents/consulting-partner.md
      output_contract: agent_result_v1
      skills:
        - hypothesis-framing
      capabilities:
        required:
          - knowledge.read
        optional:
          - web.search
          - artifact.publish

  skills:
    - id: hypothesis-framing
      path: skills/hypothesis-framing
      type: instruction_skill

  evals:
    cases: evals/cases.json
    rules: evals/rules.json

  permissions:
    network: none
    external_write: false
```

### 3.2 Source Adapter

每种来源一个 Adapter：

| Adapter | 输入 | 输出 |
| --- | --- | --- |
| `ruile-native` | ruile 原生包 | 标准 `ExpertPackage` |
| `workbuddy` | WorkBuddy 目录或 ZIP | 标准 `ExpertPackage` |
| `custom-yaml` | 企业自定义 YAML/Markdown | 标准 `ExpertPackage` |
| 后续 Adapter | 其他社区格式 | 标准 `ExpertPackage` |

Adapter 只负责源格式解析和字段映射，不负责生产发布，不负责扩大权限。

## 4. 与 WorkBuddy 的关系

WorkBuddy 仍然是一个重要参考来源，但不绑定平台架构。

### 4.1 可低成本迁移

- `agents/*.md` 中的角色、职责、边界和工作流。
- 说明型 `SKILL.md`。
- references、模板和方法论资料。
- Agent 名称、描述、头像和快捷问题。
- `maxTurns` 到 ruile 最大迭代次数的映射。
- 卡片、报告和产物的内容规范。

### 4.2 需要运行时增强

- Python、Node 或 Shell 脚本。
- 依赖多级 references、scripts 和 vendor 的复杂 Skill。
- 需要读取输入文件并生成交付物的专家。
- 需要生成 HTML、PDF、表格、图片或演示文稿的专家。

### 4.3 需要专用适配器

- WorkBuddy 私有工具。
- 特定联网数据源。
- PPTX 专用编译器。
- 本地 Office 编辑桥接。
- 本地桌面应用或浏览器自动化。

这些不能通过复制 Markdown 获得，必须转成 Capability，再由 ruile 的 MCP、连接器或专用工具实现。

## 5. 必须开发的组件

### 5.1 Admin Agent Package Importer

这是后台管理组件，不是用户功能。

支持输入：

- 管理员上传本地目录。
- ZIP/TAR 包。
- Git 仓库地址和指定版本。
- 后台登记的受信来源。

不支持：

- 普通用户上传 Agent 包。
- 普通用户安装外部 Skill。
- 未审核包直接进入生产执行。
- 公开 Agent 市场。

导入器必须完成：

- 防止压缩包路径穿越。
- 限制文件数量、单文件大小和总体积。
- 计算单文件和整包哈希。
- 识别脚本、二进制文件和危险链接。
- 读取来源、作者、许可证和版本。
- 保留原始包，不直接修改源文件。
- 可选验证发布者签名并生成软件物料清单。
- 生成兼容性问题列表。

### 5.2 Source Adapter Registry

根据包结构选择 Adapter：

```text
.codebuddy-plugin/plugin.json -> workbuddy
ruile-package.yaml            -> ruile-native
agent-package.yaml            -> custom-yaml
```

解析结果需要标记：

- 可直接转换字段。
- 需要 Capability 映射字段。
- 当前平台不支持字段。
- 缺失文件、缺失 Skill、路径越界。
- 需要管理员确认的权限扩大。

### 5.3 Agent Markdown Compiler

编译器将内部标准 Agent 定义转换为 ruile 可执行配置。

职责：

- 将 Markdown 正文编译为系统提示词或提示词片段。
- 将最大轮次、模型参数和输出协议映射到执行配置。
- 将 Skills 绑定为带包名和版本的稳定引用。
- 将来源平台工具名转换为 Capability。
- 合并平台固定安全提示。
- 检查正文引用的文件和 Skills。
- 检查超长提示词、冲突规则和危险权限。
- 生成定义哈希。
- 注册不可变版本。

编译器不负责调用模型，也不判断 Agent 质量。质量由评测、灰度和用户反馈闭环控制。

### 5.4 Agent/Skill Registry

需要支持：

- Agent 和 Skill 不可变版本。
- 包级命名空间。
- 同名 Skill 不互相覆盖。
- Agent 固定引用具体 Skill 版本。
- `draft`、`testing`、`approved`、`published`、`deprecated`、`archived` 生命周期。
- stable、canary 等发布通道。
- 停用和回退。
- 历史卡片继续指向当时版本。

推荐引用格式：

```text
consulting-partners@3.0.0/consulting-partner
consulting-partners@3.0.0/hypothesis-framing
```

### 5.5 Capability Resolver

不要把外部工具名直接写入 ruile 的 `AllowedTools`。先转换为平台无关 Capability：

```text
knowledge.read
web.search
web.fetch
filesystem.read
filesystem.write
script.python
script.node
network.http
artifact.publish
report.render_html
report.export_pdf
mcp.invoke
office.pptx.create
office.pptx.edit
```

再映射到项目实现：

```text
knowledge.read
  -> knowledge_search / grep_chunks

script.python
  -> sandbox + Python runtime

report.render_html
  -> ruile 报告渲染服务
```

兼容结果：

- `supported`：必需能力全部满足。
- `degraded`：可运行，但部分可选能力或产物不可用。
- `blocked`：关键能力缺失，不能发布。

### 5.6 Agent Evaluation

外部包中的 `evals` 可以作为导入来源，但必须转换为 ruile 评测模型。

支持断言：

- 是否应生成服务卡片。
- 输出是否通过 Schema。
- 是否命中关键事实。
- 是否生成指定类型产物。
- 是否包含证据引用。
- 是否调用或禁止调用某些工具。
- `must_hit`、`must_not`、人工判分和模型辅助判分。

发布门禁：

- 核心安全用例失败时禁止发布。
- 输出协议失败时禁止发布。
- 权限扩大必须管理员重新确认。
- 新版本先灰度，再扩大范围。

## 6. 管理员发布流程

```text
上传或登记 Agent 包
  -> 校验来源和包结构
  -> 安全扫描
  -> Source Adapter 解析
  -> 转换为 ruile Expert Package
  -> 编译 Agent 和 Skill
  -> Capability 检查
  -> 运行评测
  -> 管理员确认权限
  -> 发布版本
  -> 绑定服务领域/工作画像
  -> 普通用户使用
```

### 6.1 发布前报告

管理员至少需要看到：

- 包来源、作者、许可证和版本。
- 包含哪些 Agent、Skills 和资源。
- 是否包含脚本、二进制或 vendor 文件。
- 申请哪些 Capability。
- 需要哪些 MCP、网络域名和凭证。
- 支持和不支持哪些产物类型。
- 哪些能力会降级。
- 评测通过率和阻断项。

### 6.2 升级差异

升级前展示：

- Agent Markdown 变化。
- Skills 增删和版本变化。
- 新增工具、权限和网络访问。
- 新增脚本或二进制文件。
- 输出协议和产物策略变化。
- 新旧版本评测差异。

权限扩大必须重新确认，不能静默升级。

## 7. 用户侧使用模型

普通用户看不到导入流程。用户侧只看到：

- 已启用 Agent 名称和说明。
- 服务卡片和报告。
- 产物查看、保存和分享入口。
- 追问、纠偏和重新生成入口。

用户不能：

- 上传 Agent 包。
- 安装外部 Skills。
- 修改 Agent Markdown。
- 修改工具权限。
- 选择未发布版本。
- 让 Agent 获取额外系统权限。

## 8. 开发优先级

### P0：AgentRun 执行队列

先完成统一运行入口：

1. `AgentRun` 数据模型。
2. `agent` 队列和 Worker。
3. 日报异步生成。
4. 记忆异步生成服务卡片。
5. 前端轮询运行状态。
6. 基础幂等、失败和取消。

P0 不依赖外部 Agent 包导入。

### P1：管理员 Registry 与发布中心

1. Agent/Skill Registry。
2. Agent 生命周期。
3. 管理员发布、停用和回退。
4. Agent 与工作画像、服务领域绑定。
5. 基础 Agent Evaluation。
6. `agent_result_v1` 输出校验。

完成后，内置 Agent 也走同一发布治理模型。

### P2：多来源 Agent 包导入

1. Admin Agent Package Importer。
2. Source Adapter Registry。
3. ruile-native Adapter。
4. WorkBuddy Adapter。
5. 兼容性报告。
6. 安全扫描和权限确认。

完成后，管理员可以接入不同来源 Agent 包，但普通用户仍不能导入。

### P3：通用产物与脚本

1. `AgentArtifact` 和版本。
2. AgentRun 级沙箱工作区。
3. Python/Node 运行时。
4. Skill 依赖解析。
5. `/output` 产物回收。
6. HTML/PDF 渲染和预览。

### P4：联网、凭证和专用能力

1. Capability 到 MCP/连接器的自动绑定。
2. 网络白名单。
3. Secrets Broker。
4. OAuth 和凭证安装检查。
5. PPTX、Office、本地应用等专用适配器。

### P5：完整质量与运营治理

1. 完整发布门禁。
2. 版本差异、灰度和回退。
3. 运行质量与成本仪表盘。
4. 自动故障诊断。
5. Git 或后台登记源更新检查。
6. 异常版本自动停用。

## 9. 兼容等级

| 等级 | 内容 | 支持策略 |
| --- | --- | --- |
| L0 | 只有 Agent Markdown | 优先完整支持 |
| L1 | Agent Markdown + 说明型 Skills | 优先完整支持 |
| L2 | 包含离线 Python/Node 脚本 | 增强沙箱后支持 |
| L3 | 需要联网、MCP、凭证或 OAuth | 受控网络和 Capability 后支持 |
| L4 | 需要 PPT、Office、本地应用或专用 UI | 按需求开发专用适配器 |

优先支持 L0-L2。大量 Agent 的核心价值来自提示词、方法论和参考资料，这部分接入成本最低，也最适合先验证业务价值。

## 10. 验收标准

P0 完成时：

- 用户生成日报和服务卡片时立即获得 `run_id`。
- 任务在 `agent` 队列中执行。
- 前端展示 queued、running、succeeded、failed。
- 成功后刷新现有日报或服务卡片。
- 重复提交不会重复生成多份结果。

P1 完成时：

- 管理员可以维护 Agent 版本。
- 只有 `published` 版本可被普通用户调用。
- Agent 可以绑定到工作画像和服务领域。
- 新版本评测失败时不能发布。
- 回退不影响历史卡片和报告。

P2 完成时：

- 管理员可以导入至少一种外部 Agent 包。
- Source Adapter 转换为 ruile Expert Package。
- 系统展示支持、降级和阻断项。
- 普通用户不能导入或安装 Agent。

P3-P5 完成时：

- 脚本只在隔离沙箱中执行。
- 生成文件自动进入统一产物管理。
- 网络和凭证只能通过获批 Capability 使用。
- 运营人员可以灰度、回退、停用和诊断 Agent 版本。

## 11. 最终原则

不要为每个外部 Agent 单独编写 Go 业务代码。平台只为新的通用 Capability 开发一次适配器，所有声明该能力的 Agent 复用。

目标接入流程是：

```text
管理员导入
-> Source Adapter 解析
-> 安全与兼容检查
-> 编译
-> 评测
-> 发布
-> 绑定
-> 用户使用
```

而不是：

```text
普通用户上传 Agent
-> 直接执行外部脚本
-> 运行中发现缺依赖或越权
```

这个权限边界是降低长期运营成本和安全风险的关键。
