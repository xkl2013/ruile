# 知识库个人、组织与内部共享空间 PRD

文档日期：2026-09-12

适用仓库：`/Users/jamgogh/Desktop/agent/ruile`

状态：产品需求草案，已补充个人升级企业路径、存储管理、知识库默认配置与基于当前工程的改版规划；Token/积分模块列为后续版本，本期不开发

## 0. 方案总览

本方案把知识库归属拆成三层：

```text
账号 User
  - 个人空间
      - 个人知识库：仅本人可见，不可共享、不可发布、不可被订阅
  - 组织空间
      - 组织员工
      - 组织知识库
      - 组织内部共享空间
          - 共享空间成员：从本组织员工中选择
          - 共享空间知识库：从本组织知识库中选择
```

登录账号在知识库首页只需要理解三个入口：

```text
我创建的
共享给我的
我订阅的
```

关键产品结论：

- 个人知识库永远是私有资产。
- 个人升级企业不是把个人空间改成组织空间，而是新增一个组织空间。
- 组织内部共享空间只负责组织内知识分发，不解决集团、多组织、跨组织共享。
- 订阅只是个人快捷入口，不是权限来源。
- 存储管理按个人空间和组织空间分别计算，订阅和共享空间不复制文件也不占用额外存储。
- 本期不开发 Token/积分账户、计费、扣减和额度管理；模型调用继续沿用现有技术链路，不增加产品级余额拦截。
- 知识库创建保持极简：当前选中的空间决定归属，用户只填写名称和描述，其余配置由后台默认配置自动注入。
- 所有最终权限以后端统一计算为准，前端只展示权限结果。

## 1. 背景

当前知识库能力以「空间」为主要归属单位，已具备空间成员、角色权限、知识库创建者、共享空间等基础能力。但在产品表达上，个人使用、组织内部协作、共享空间、订阅等概念边界还不够清晰，容易出现以下问题：

- 个人知识库是否可以分享、发布、被订阅，缺少明确规则。
- 组织员工创建的知识库到底属于个人还是组织，界面和权限表达不够直观。
- 多人协作时，用户希望按业务小组、项目、岗位创建「共享空间」，而不是让组织内所有员工默认看到所有组织知识库。
- 登录后的知识库列表需要从「空间下全部知识库」升级为以账号为中心的三类入口：我创建的、共享给我的、我订阅的。

本 PRD 重新定义知识库在个人、组织、组织内部共享空间中的归属、权限、列表与订阅规则。

## 2. 目标

1. 支持个人用户创建自己的私有知识库。
2. 明确个人知识库不可共享、不可发布、不可加入共享空间、不可被他人订阅。
3. 支持组织内创建组织知识库。
4. 支持组织内创建共享空间，并将组织员工、组织知识库加入共享空间。
5. 已加入共享空间的员工可以查看共享空间内的知识库。
6. 登录账号可以统一查看：
   - 我创建的
   - 共享给我的
   - 我订阅的
7. 订阅作为个人快捷入口，不作为权限来源。
8. 后端提供统一权限判断，前端仅展示后端返回的可见性和能力。
9. 支持个人账号升级为企业/组织账号，但保留个人空间和个人知识库的私有边界。
10. 增加存储管理：展示个人/组织空间用量和配额，组织可管理存储实例和默认存储，知识库自动绑定默认存储。
11. 简化知识库创建流程：用户只输入知识库名称和描述，后台自动应用默认配置。

## 3. 非目标

1. 第一版不引入集团、总部、子组织、多级组织树。
2. 第一版不做跨组织共享空间；共享空间仅限单个组织内部使用。
3. 第一版不支持个人知识库共享给任何人。
4. 第一版不做知识库公开市场、公开发现页、付费订阅。
5. 第一版不复制被订阅知识库内容，订阅只保存引用关系。
6. 第一版不重构文档解析、向量索引、底层存储后端实现、模型配置链路。
7. 第一版不要求替换现有所有历史共享能力，可分阶段兼容迁移。
8. 第一版不把个人空间原地转换成组织空间。
9. 第一版不自动迁移、共享或接管任何个人知识库。
10. 第一版不做跨空间存储池、集团级统一存储池和跨组织成本分摊。
11. 本期不开发 Token/积分账户、计价、消费、员工限额、充值、退款、对账和余额拦截。
12. Token/积分相关能力保留为后续独立版本，不纳入本期验收和交付计划。
13. 第一版不向普通用户开放知识库模型、解析、切片、向量、检索、存储绑定等高级配置。
14. 第一版不在知识库创建表单中提供配置项选择；高级配置只在后台默认配置或后台治理流程中维护。

## 4. 术语定义

| 术语 | 定义 |
| --- | --- |
| 个人知识库 | 用户在个人空间创建的知识库，仅创建者本人可见和管理 |
| 组织知识库 | 用户在组织空间创建的知识库，归属于组织 |
| 组织员工 | 被加入组织的账号，拥有组织内角色 |
| 共享空间 | 组织内部的知识分发容器，可加入组织员工和组织知识库 |
| 空间成员 | 被加入某个共享空间的组织员工 |
| 空间知识库 | 被加入某个共享空间的组织知识库 |
| 订阅 | 用户对自己已有访问权的知识库建立个人快捷入口 |
| 访问权 | 由创建者、组织角色、共享空间成员关系等规则计算得到的最终权限 |
| 升级企业 | 个人账号创建一个新的组织空间，并成为该组织 Owner |
| 存储配额 | 空间可使用的最大存储量 |
| 存储用量 | 空间已消耗的文件、解析文本、索引估算等存储量 |
| 存储实例 | 一个空间下配置的对象/文件存储后端，例如 local、MinIO、OSS、S3 |
| Token usage | 模型供应商返回的技术用量字段，仅用于现有模型响应和内部观测，本期不转化为产品积分 |
| Token/积分管理 | 后续版本能力，本期不建立积分账户、流水、计价或额度模型 |
| 知识库默认配置 | 后台为新知识库自动注入的模型、解析、切片、向量、检索和存储策略 |
| 配置版本 | 创建知识库时使用的后台默认配置版本，用于追踪和后续迁移 |

### 4.1 产品概念与现有技术概念映射

| 产品概念 | 当前技术概念建议 |
| --- | --- |
| 个人空间 | `tenant` 的一种类型，`space_type = personal` |
| 组织空间 | `tenant` 的一种类型，`space_type = organization` |
| 组织员工 | `tenant_members` |
| 组织知识库 | `knowledge_bases.tenant_id = 组织空间 ID` |
| 创建者 | `knowledge_bases.creator_id` |
| 组织内部共享空间 | 新增 `shared_spaces`，隶属于一个组织空间 |
| 订阅 | 新增 `knowledge_base_subscriptions` |
| 存储配额/用量 | 复用 `tenants.storage_quota`、`tenants.storage_used` |
| 存储实例 | 复用 `storage_backends`、`tenants.default_storage_backend_id` |
| 知识库存储绑定 | 后台自动写入 `knowledge_bases.storage_backend_id`，不作为创建表单输入 |
| Token/积分管理 | 后续版本规划，本期不新增账户、流水、计价和员工限额表 |

说明：现有 `organizations / kb_shares` 更偏跨空间共享关系。第一版的「组织内部共享空间」建议作为组织内协作能力单独建模，避免和跨空间协作概念混用。

## 5. 核心规则

### 5.1 个人知识库规则

- 个人知识库只能由创建者本人查看、检索、问答、上传、编辑、删除。
- 个人知识库不能加入共享空间。
- 个人知识库不能分享给组织员工。
- 个人知识库不能发布或允许订阅。
- 个人知识库不能被组织管理员查看或接管。
- 个人知识库删除只影响创建者本人。

### 5.2 组织知识库规则

- 组织知识库必须属于某一个组织空间。
- 组织员工可按组织角色创建知识库，默认创建者拥有管理权限。
- 组织管理员可管理组织内全部组织知识库。
- 组织知识库可加入一个或多个组织内部共享空间。
- 员工只有在满足以下任一条件时可以查看组织知识库：
  - 自己创建；
  - 是组织管理员；
  - 被加入包含该知识库的共享空间；
  - 已订阅，且当前仍有访问权。

### 5.3 共享空间规则

- 共享空间隶属于单个组织。
- 共享空间只能添加本组织员工。
- 共享空间只能添加本组织知识库。
- 共享空间成员默认可查看空间内知识库。
- 第一版共享空间知识库权限只做 `viewer`；后续可扩展 `editor`。
- 员工被移出共享空间后，立即失去通过该共享空间获得的知识库访问权。
- 知识库从共享空间移除后，空间成员立即失去通过该空间获得的访问权。

### 5.4 订阅规则

- 用户只能订阅自己当前有权访问的知识库。
- 订阅不改变知识库权限。
- 订阅不复制知识库内容、索引、文件、标签和配置。
- 如果用户后续失去访问权，订阅记录可以保留，但「我订阅的」列表中应隐藏或显示为无权限状态。
- 个人知识库不可被他人订阅。

### 5.5 个人升级企业规则

- 个人升级企业的本质是创建一个新的组织空间。
- 原个人空间继续保留。
- 原个人知识库继续保持私有，不自动进入组织空间。
- 当前账号自动成为新组织空间的 Owner。
- 升级完成后，用户可以在个人空间和组织空间之间切换。
- 用户可选择把个人知识库复制到组织空间，但默认不复制。
- 第一版只支持复制到组织空间，不支持直接转移个人知识库所有权。
- 复制后的组织知识库是一个新知识库，归属组织空间；原个人知识库仍归属个人空间。

### 5.6 存储管理规则

- 个人空间和组织空间分别计算存储用量和配额。
- 个人知识库消耗个人空间存储配额。
- 组织知识库消耗组织空间存储配额。
- 共享空间不拥有知识库文件，不单独计算存储用量；共享空间内知识库的存储仍计入知识库所属组织空间。
- 订阅不复制知识库内容，不增加订阅者个人空间或组织空间的存储用量。
- 个人升级企业时，原个人空间存储用量不转入组织空间。
- 将个人知识库复制到组织空间时，新组织知识库产生的文件和索引消耗组织空间配额。本路线在 V3 提供完整内容复制；仅复制配置不作为企业升级完成标准。
- 组织管理员可管理组织空间存储实例和默认存储实例。
- 个人版默认只展示存储用量和配额；是否允许个人配置自定义存储实例由部署形态决定。SaaS 场景建议不开放，私有化部署可复用现有空间 Admin+ 存储管理能力。
- 存储配额不足时，禁止继续上传、解析或复制会增加存储占用的内容，并给出明确提示。

### 5.7 Token/积分管理范围

- Token/积分账户、计价、消费、余额、员工限额和后台发放不属于本期开发范围。
- 本期不新增产品级积分扣减，也不因余额或额度限制阻断问答、解析、导入等已有能力。
- 现有模型返回的 token usage、日志和 Langfuse 观测能力保持现状，仅作为技术数据，不作为用户可见积分余额。
- 后续如开发积分模块，应单独确定个人/组织扣费主体、计价规则、流水和员工限额，不在本期实现中预留业务逻辑。

### 5.8 知识库创建与默认配置规则

- 创建知识库时，当前登录账号正在使用的空间决定知识库归属：
  - 当前为个人空间时，创建个人知识库；
  - 当前为组织空间时，创建组织知识库。
- 创建表单只包含：
  - 知识库名称；
  - 知识库描述。
- 创建请求不得由前端传入 `tenant_id`、`owner_type`、`storage_backend_id`、模型 ID、解析配置、切片配置或向量库配置。
- 后端从登录上下文解析当前空间，并校验当前用户是否有在该空间创建知识库的权限。
- 后端创建知识库时自动注入默认配置，至少包括：
  - 知识库类型和默认图标；
  - 文档解析和抽取配置；
  - 切片配置；
  - embedding、summary、VLM、OCR、ASR 等模型配置；
  - 向量存储和检索策略；
  - 默认存储实例；
  - 默认访问权限。
- 普通用户在创建完成后不需要继续配置知识库，即可上传内容、检索和问答。
- 普通用户不能在知识库设置页修改上述高级配置；知识库设置页只保留名称、描述、内容管理、共享空间和订阅等与业务协作有关的入口。
- 后台默认配置按“全局默认配置 -> 空间/组织默认配置 -> 知识库创建时解析”的顺序生效；如无空间级覆盖，则使用全局默认配置。
- 默认配置在创建时解析并记录配置版本。后续后台修改默认配置只影响新建知识库，不自动改变已有知识库，避免线上知识库行为突然变化。
- 已有知识库确需调整高级配置时，由后台管理员执行配置迁移或重新绑定，不向普通用户开放。

## 6. 用户角色与权限

### 6.1 组织角色

| 角色 | 说明 | 关键权限 |
| --- | --- | --- |
| Owner | 组织所有者 | 组织全部管理能力、Owner 转移、删除组织 |
| Admin | 组织管理员 | 管理员工、组织知识库、共享空间 |
| Contributor | 普通贡献者 | 创建和维护自己创建的组织知识库 |
| Viewer | 只读成员 | 查看授权给自己的知识库 |

### 6.2 共享空间角色

| 角色 | 说明 | 关键权限 |
| --- | --- | --- |
| Admin | 共享空间管理员 | 管理该共享空间成员和空间知识库 |
| Viewer | 共享空间成员 | 查看该共享空间内知识库 |

第一版不引入共享空间 `Editor`。如需允许共享空间成员维护内容，可在后续版本增加 `Editor`，并在知识库加入共享空间时配置 `viewer/editor`。

### 6.3 操作权限矩阵

| 操作 | 个人知识库创建者 | 组织 KB 创建者 | 组织 Admin/Owner | 共享空间 Admin | 共享空间 Viewer |
| --- | --- | --- | --- | --- | --- |
| 修改知识库名称和描述 | 是 | 是 | 是 | 否 | 否 |
| 查看个人知识库 | 是 | 否 | 否 | 否 | 否 |
| 编辑个人知识库 | 是 | 否 | 否 | 否 | 否 |
| 创建组织知识库 | 否 | 按组织角色 | 是 | 否 | 否 |
| 查看自己创建的组织知识库 | 否 | 是 | 是 | 视共享关系 | 视共享关系 |
| 编辑自己创建的组织知识库 | 否 | 是 | 是 | 否 | 否 |
| 删除组织知识库 | 否 | 是 | 是 | 否 | 否 |
| 创建共享空间 | 否 | 否 | 是 | 否 | 否 |
| 添加员工到共享空间 | 否 | 否 | 是 | 是 | 否 |
| 添加知识库到共享空间 | 否 | 自己创建的组织知识库 | 是 | 需同时有 KB 管理权 | 否 |
| 查看共享空间知识库 | 否 | 如被加入空间 | 是 | 是 | 是 |
| 订阅知识库 | 自己无需订阅 | 有访问权即可 | 有访问权即可 | 有访问权即可 | 有访问权即可 |
| 查看存储用量 | 是，仅个人空间 | 是，仅可见空间 | 是 | 否 | 否 |
| 管理组织存储实例 | 否 | 否 | 是 | 否 | 否 |
| 普通用户修改知识库高级配置 | 否 | 否 | 否 | 否 | 否 |

说明：后台默认配置不属于组织员工权限矩阵，由独立的系统/后台管理员能力维护；组织 Owner/Admin 只管理组织存储实例、员工和共享空间。Token/积分权限不在本期设计。

## 7. 用户故事

### 7.1 个人用户创建私有知识库

作为个人用户，我希望创建只属于自己的知识库，用于沉淀个人资料、学习笔记或工作材料，并确保组织管理员或其他员工不可见。

验收标准：

- 个人空间下创建的知识库标记为个人知识库。
- 设置页不出现共享空间、发布、允许订阅等入口。
- 其他账号通过 URL 访问返回无权限或不存在。

### 7.2 组织管理员创建共享空间

作为组织管理员，我希望为某个业务小组创建共享空间，例如「招生资料共享空间」，并把相关员工加入其中。

验收标准：

- 管理员可以创建、编辑、删除共享空间。
- 管理员可以从组织员工中选择成员加入共享空间。
- 非本组织员工不可被加入。

### 7.3 将组织知识库加入共享空间

作为组织管理员或知识库创建者，我希望把组织知识库加入共享空间，让空间成员可以查看。

验收标准：

- 只能选择本组织知识库。
- 个人知识库不可出现在可加入列表。
- 被加入空间的员工可在「共享给我的」看到该知识库。

### 7.4 员工查看共享给自己的知识库

作为组织员工，我希望只看到和我相关的共享知识库，而不是组织内所有知识库。

验收标准：

- 员工登录后，「共享给我的」只展示自己所在共享空间内的知识库。
- 员工被移出共享空间后，该知识库从「共享给我的」消失。

### 7.5 员工订阅共享知识库

作为员工，我希望订阅常用的共享知识库，之后可以在「我订阅的」快速访问。

验收标准：

- 只有有访问权的知识库显示订阅入口。
- 订阅后出现在「我订阅的」。
- 失去访问权后不可继续打开订阅知识库。

### 7.6 个人用户升级为企业

作为个人用户，我希望在业务扩大后创建企业/组织空间，邀请员工协作，同时保留个人空间中的私有知识库。

验收标准：

- 升级后系统新增组织空间，当前用户成为 Owner。
- 原个人空间继续存在。
- 原个人知识库不自动共享给企业员工。
- 用户可选择把个人知识库复制到组织空间。
- 复制后的组织知识库可加入共享空间。

### 7.7 组织管理员管理存储

作为组织管理员，我希望查看组织存储用量、配置存储实例、设置默认存储，并让新建知识库自动使用组织默认存储。

验收标准：

- 管理员可以查看组织总配额、已用量、剩余量和使用率。
- 管理员可以管理组织存储实例。
- 管理员可以设置组织默认存储实例。
- 新建组织知识库默认使用组织默认存储实例。
- 创建组织知识库时不需要选择存储实例。
- 配额不足时，上传、复制和解析流程给出明确阻断提示。

### 7.8 个人用户查看存储

作为个人用户，我希望看到个人空间的存储用量和配额，理解哪些个人知识库占用了空间。

验收标准：

- 个人空间可以展示总配额、已用量、剩余量和使用率。
- 个人知识库卡片或详情可展示存储占用。
- 个人订阅的组织知识库不计入个人空间用量。

### 7.9 Token/积分模块（后续版本）

Token/积分账户、计价、消费明细、员工限额、充值和后台发放将在后续版本单独设计和开发，本期不作为用户故事、功能需求或验收项。

### 7.10 用户创建知识库

作为个人用户或组织员工，我希望快速创建知识库，不需要理解模型、解析、向量和存储配置。

验收标准：

- 用户打开创建弹窗后只看到名称和描述两个字段。
- 当前选中的个人空间或组织空间自动作为知识库归属，不要求用户再次选择空间。
- 创建成功后，知识库可以直接进入内容上传和问答流程。
- 创建表单不展示模型、解析、切片、向量、检索和存储配置。
- 后台默认配置变更不会影响已有知识库。

## 8. 功能需求

### 8.1 知识库归属

FR-001：创建知识库时，后端必须根据当前登录上下文确定创建位置。

- 当前活动空间为个人空间：创建个人知识库。
- 当前活动空间为组织空间：创建组织知识库。
- 前端不提交可任意指定的 `tenant_id`，避免越权创建。

FR-002：知识库响应中应返回归属信息。

建议字段：

```json
{
  "owner_type": "personal",
  "tenant_id": 10000,
  "creator_id": "user-id",
  "created_scope": "personal",
  "access_source": "created"
}
```

FR-003：个人知识库必须在后端禁止共享、发布、被订阅。

FR-004：创建知识库请求只允许提交名称和描述。

```json
{
  "name": "招生资料",
  "description": "招生话术、活动方案和常见问答"
}
```

以下字段不得作为普通用户创建请求的一部分：

- `tenant_id`；
- `owner_type`；
- `storage_backend_id`；
- `vector_store_id`；
- `embedding_model_id`；
- `summary_model_id`；
- `chunking_config`；
- `extract_config`；
- `indexing_strategy`；
- OCR、VLM、ASR 及其他模型配置。

FR-005：后端创建知识库时自动应用后台默认配置。

默认配置至少覆盖：

- 知识库类型和默认图标；
- 文档解析、抽取和切片；
- embedding、summary、VLM、OCR、ASR 模型；
- 向量存储和检索策略；
- 默认存储实例；
- 默认访问权限。

FR-006：知识库创建成功后不要求用户继续完成配置。

- 创建成功后直接进入知识库内容页。
- 用户可以立即上传文件、导入内容、检索和问答。
- 普通用户设置页只提供名称、描述、内容管理、共享空间和订阅等业务入口。
- 高级配置由后台默认配置或后台迁移流程维护。

FR-007：后台默认配置变更只影响新建知识库。

- 创建知识库时记录使用的默认配置版本。
- 已有知识库继续使用创建时解析的配置。
- 后台管理员如需变更已有知识库，必须显式发起迁移并记录审计。

### 8.2 我的知识库列表

FR-010：新增统一列表接口，返回当前账号维度的三类知识库。

```text
GET /api/v1/knowledge-bases/my
```

响应示例：

```json
{
  "success": true,
  "data": {
    "created": [],
    "shared": [],
    "subscribed": []
  }
}
```

FR-011：列表分类规则。

| 分类 | 后端规则 |
| --- | --- |
| 我创建的 | `knowledge_bases.creator_id = 当前用户` |
| 共享给我的 | 当前用户是某共享空间成员，且该共享空间包含该知识库 |
| 我订阅的 | 当前用户有订阅记录，且当前仍有知识库访问权 |

FR-012：同一知识库出现在多个分类时，前端按分类独立展示，但卡片 ID 保持一致，不复制数据。

FR-013：订阅列表必须再次校验访问权，不允许仅凭订阅记录访问。

### 8.3 共享空间管理

FR-020：组织管理员可创建共享空间。

字段：

- 名称；
- 描述；
- 创建人；
- 所属组织。

FR-021：组织管理员和共享空间管理员可编辑共享空间名称和描述。

FR-022：组织管理员可删除共享空间。

删除影响：

- 删除共享空间成员关系；
- 删除共享空间知识库关系；
- 不删除知识库本体；
- 不删除订阅记录，但订阅访问权会失效。

### 8.4 共享空间成员

FR-030：共享空间只能添加本组织员工。

FR-031：添加成员时可设置角色。

第一版角色：

- `admin`
- `viewer`

FR-032：移除成员后，该成员立即失去通过该共享空间获得的知识库访问权。

FR-033：组织员工被移出组织或停用时，应自动失去所有共享空间访问权。

### 8.5 共享空间知识库

FR-040：共享空间只能添加本组织知识库。

FR-041：个人知识库不可加入共享空间。

FR-042：同一知识库可以加入多个共享空间。

FR-043：从共享空间移除知识库时，不删除知识库本体。

FR-044：第一版共享空间内知识库权限固定为只读。

### 8.6 订阅

FR-050：有访问权的用户可以订阅知识库。

FR-051：个人知识库不允许被其他用户订阅。

FR-052：订阅后，知识库出现在「我订阅的」。

FR-053：取消订阅后，从「我订阅的」移除。

FR-054：失去访问权后，订阅记录不可继续提供访问能力。

### 8.7 权限判断

FR-060：后端新增统一知识库访问解析方法。

建议方法职责：

```text
ResolveKnowledgeBaseAccess(ctx, userID, tenantID, kbID)
```

返回：

```text
permission: none / viewer / editor / manager
access_source: created / organization_admin / shared_space / subscription
effective_tenant_id
```

FR-061：所有知识库详情、文档列表、搜索、预览、问答、上传、编辑、删除、订阅接口都必须调用统一权限判断。

FR-062：前端 `canEdit/canShare/canSubscribe` 只使用后端返回结果，不自行推导最终权限。

### 8.8 个人升级企业

FR-070：个人账号可以发起升级企业流程。

升级请求至少包含：

- 组织名称；
- 行业；
- 规模；
- 联系人信息，默认当前账号。

FR-071：升级企业时创建新的组织空间。

新组织空间字段：

```text
space_type = organization
```

FR-072：系统为当前用户写入组织成员关系。

组织 Owner 以 `tenant_members.role=owner` 为唯一权威来源，不额外维护 `tenants.owner_user_id`。

```text
tenant_members.user_id = 当前用户
tenant_members.tenant_id = 新组织空间 ID
tenant_members.role = owner
tenant_members.status = active
```

FR-073：升级流程不得修改原个人知识库的归属和权限。

FR-074：升级流程支持将个人知识库复制到组织空间。

复制规则：

- 创建新的 `knowledge_bases` 行；
- 新知识库 `tenant_id` 指向组织空间；
- 新知识库 `creator_id` 仍为当前用户；
- 不复制共享关系；
- 不复制订阅关系；
- 文件、解析结果和索引复制策略复用现有知识库复制能力，以异步任务执行；仅复制设置不作为 V3 的交付结果。

FR-075：升级完成后，前端应引导用户进入组织空间，并明确提示「个人空间和个人知识库仍然私有」。

### 8.9 存储管理

FR-080：系统应展示空间级存储用量。

展示字段：

- 总配额；
- 已用量；
- 剩余量；
- 使用率；
- 超配额状态；
- 按知识库聚合的用量。

FR-081：个人空间只展示个人存储用量和配额。

- 个人知识库计入个人空间。
- 订阅知识库不计入个人空间。
- 个人知识库复制到组织空间后，新组织知识库计入组织空间。

FR-082：组织空间由组织 Admin/Owner 管理存储实例。

管理内容复用现有存储实例能力：

- 创建存储实例；
- 测试连通性；
- 更新凭据；
- 设置默认存储实例；
- 禁用或删除未使用的存储实例。

FR-083：知识库创建时由后端自动完成存储绑定。

- 个人知识库自动使用个人空间的默认存储策略。
- 组织知识库自动使用组织空间的默认存储实例。
- 创建请求不允许传入 `storage_backend_id`。
- 普通用户不能在知识库创建页或设置页选择、修改存储实例。
- 个人知识库不能绑定组织空间存储实例。
- 组织知识库不能绑定个人空间存储实例。
- 后台管理员可通过默认配置或迁移流程调整已有知识库的存储绑定。

FR-084：共享空间和订阅不产生独立存储占用。

- 共享空间只保存成员关系和知识库关系。
- 订阅只保存用户和知识库关系。
- 共享和订阅不复制文件、切片、索引或资源绑定。

FR-085：存储配额必须在会增加占用的操作前后校验。

至少包括：

- 上传文件；
- URL 抓取入库；
- 文本手动入库；
- 重新解析；
- 知识库复制到组织空间；
- 批量导入。

FR-086：配额不足时应返回可识别错误。

建议错误：

```text
Storage quota exceeded
```

前端提示：

```text
当前空间存储容量不足，请清理知识库文件或联系管理员扩容。
```

### 8.10 Token/积分管理（后续版本，不纳入本期）

以下内容仅作为后续版本设计草案，本期不开发、不建表、不接入现有问答、解析、导入和评测流程，也不作为本期验收标准。

FR-090：系统应为空间建立积分账户。

账户维度：

- 个人空间积分账户；
- 组织空间积分账户。

展示字段：

- 当前余额；
- 冻结积分；
- 可用余额；
- 本月消耗；
- 低余额阈值；
- 账户状态。

FR-091：系统应记录积分流水。

流水类型：

- `grant`：系统发放；
- `recharge`：充值入账；
- `consume`：模型调用消耗；
- `refund`：失败或取消后退回；
- `adjust`：管理员手工调整；
- `expire`：积分过期；
- `reserve`：异步任务预冻结；
- `release`：释放未消耗冻结额。

流水至少记录：

- 空间 ID；
- 操作用户；
- 知识库 ID；
- 会话 ID 或任务 ID；
- 模型 ID；
- 操作类型；
- prompt tokens；
- completion tokens；
- cached tokens；
- 原始用量；
- 换算后的积分；
- 计价规则版本；
- 扣减前后余额；
- 幂等键。

FR-092：系统应维护模型计价规则。

计价维度：

- 模型供应商；
- 模型 ID；
- 能力类型：chat、embedding、rerank、vlm、ocr、asr、summary、suggestion、evaluation；
- 输入 token 单价；
- 输出 token 单价；
- 缓存 token 单价；
- 图片、页数、音频分钟、文档数等非 token 计价单位；
- 生效时间；
- 状态。

FR-093：系统应按知识库归属确定计费主体。

规则：

- 个人知识库：计费主体为个人空间。
- 组织知识库：计费主体为组织空间。
- 共享空间知识库：计费主体为知识库所属组织空间。
- 订阅知识库：计费主体仍为知识库所属空间。
- 复制后的组织知识库：计费主体为目标组织空间。

FR-094：以下操作应纳入积分管理。

至少包括：

- 知识库问答；
- 流式问答；
- 检索增强中的 query 改写、总结、引用整理；
- 文档上传后的 embedding；
- OCR、VLM 图片理解、ASR 音频转写；
- rerank；
- FAQ、问题建议、摘要、标签、Wiki 生成；
- 评测任务；
- 智能体调用知识库产生的模型调用。

FR-095：系统应在产生模型成本前做余额和限额校验。

- 同步问答：发起前按请求和模型预估最低可用积分，不足时直接拒绝。
- 异步解析：入队前按文件大小、页数、音频时长或历史均值预估并冻结积分。
- 任务完成后按实际 usage 结算，多冻结部分释放，不足部分补扣或标记待补扣。
- 重试任务必须复用幂等键，避免重复扣费。

FR-096：员工积分限额应支持组织级配置。

配置范围：

- 组织默认日限额；
- 组织默认月限额；
- 单个员工日限额；
- 单个员工月限额；
- 是否允许超额后继续只读访问。

说明：

- 员工限额只限制员工消耗组织积分的能力，不改变知识库访问权。
- Owner/Admin 可不受默认员工限额限制，但仍受组织余额限制。
- 被移出组织后，不再能消耗该组织积分。

FR-097：积分不足时应返回可识别错误。

建议错误：

```text
Credit balance insufficient
```

员工限额不足建议错误：

```text
Member credit limit exceeded
```

前端提示：

```text
当前空间积分不足，请充值或联系管理员增加额度。
```

```text
你已达到本组织的积分使用上限，请联系组织管理员调整额度。
```

FR-098：用量来源应明确标识。

取值：

- `provider_reported`：供应商返回的准确 usage；
- `estimated`：系统按文本长度、页数、图片数、音频时长估算；
- `manual`：人工调整；
- `imported`：从外部计费或观测系统导入。

FR-099：积分统计应支持多维聚合。

聚合维度：

- 空间；
- 用户；
- 知识库；
- 共享空间；
- 模型；
- 能力类型；
- 日期；
- 任务；
- 会话。

## 9. 数据模型

### 9.1 tenants 增补字段

```sql
ALTER TABLE tenants
ADD COLUMN IF NOT EXISTS space_type VARCHAR(20);
```

取值：

- `personal`
- `organization`
- `legacy`

说明：

- `NULL` 或 `legacy` 表示存量空间尚未完成类型确认，继续使用兼容逻辑。
- 不把所有存量租户设置为 `organization`，避免个人知识库因迁移被组织管理员或同空间成员看到。
- 组织 Owner 继续从 `tenant_members.role=owner` 读取，不新增 `owner_user_id`。

### 9.2 knowledge_bases 增补字段

```sql
ALTER TABLE knowledge_bases
ADD COLUMN IF NOT EXISTS config_source VARCHAR(20),
ADD COLUMN IF NOT EXISTS config_version VARCHAR(64);
```

说明：

- 不新增 `owner_type` 作为第二个归属事实，知识库归属通过 `knowledge_bases.tenant_id -> tenants.space_type` 判断。
- 响应层可以返回计算得到的 `owner_type`，但不得将其作为独立可修改字段持久化。
- `knowledge_bases` 中已有的模型、解析、切片、向量和存储字段继续作为运行时配置载体，但这些字段由后台默认配置解析流程写入，不由普通用户在创建请求中传入。
- `storage_backend_id`、`vector_store_id` 等字段属于内部实现字段；创建接口只返回配置来源和版本，不要求前端理解具体配置值。

建议补充内部配置追踪字段：

```sql
ALTER TABLE knowledge_bases
ADD COLUMN IF NOT EXISTS config_source VARCHAR(32) NOT NULL DEFAULT 'backend_default',
ADD COLUMN IF NOT EXISTS config_version VARCHAR(64) NOT NULL DEFAULT 'default';
```

取值建议：

- `config_source = backend_default`：由后台默认配置注入；
- `config_source = migrated`：由后台迁移流程调整；
- `config_version`：记录创建或迁移时使用的配置版本。

### 9.3 shared_spaces

```sql
CREATE TABLE shared_spaces (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

索引：

```sql
CREATE INDEX idx_shared_spaces_tenant_id ON shared_spaces(tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_shared_spaces_created_by ON shared_spaces(created_by) WHERE deleted_at IS NULL;
```

### 9.4 shared_space_members

```sql
CREATE TABLE shared_space_members (
    id VARCHAR(36) PRIMARY KEY,
    shared_space_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    role VARCHAR(20) NOT NULL DEFAULT 'viewer',
    added_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

约束：

```sql
CREATE UNIQUE INDEX uq_shared_space_members_space_user
ON shared_space_members(shared_space_id, user_id)
WHERE deleted_at IS NULL;
```

### 9.5 shared_space_knowledge_bases

```sql
CREATE TABLE shared_space_knowledge_bases (
    id VARCHAR(36) PRIMARY KEY,
    shared_space_id VARCHAR(36) NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    permission VARCHAR(20) NOT NULL DEFAULT 'viewer',
    added_by VARCHAR(36) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

约束：

```sql
CREATE UNIQUE INDEX uq_shared_space_kbs_space_kb
ON shared_space_knowledge_bases(shared_space_id, knowledge_base_id)
WHERE deleted_at IS NULL;
```

### 9.6 knowledge_base_subscriptions

```sql
CREATE TABLE knowledge_base_subscriptions (
    id VARCHAR(36) PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    knowledge_base_id VARCHAR(36) NOT NULL,
    source VARCHAR(32) NOT NULL DEFAULT 'shared_space',
    source_id VARCHAR(36) NOT NULL DEFAULT '',
    subscribed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
```

约束：

```sql
CREATE UNIQUE INDEX uq_kb_subscriptions_user_kb
ON knowledge_base_subscriptions(user_id, knowledge_base_id)
WHERE deleted_at IS NULL;
```

### 9.7 存储管理复用字段

本 PRD 不新增独立存储池表，优先复用当前工程已有字段和表：

| 对象 | 字段/表 | 用途 |
| --- | --- | --- |
| 空间配额 | `tenants.storage_quota` | 当前空间最大可用存储量 |
| 空间已用量 | `tenants.storage_used` | 当前空间已用存储量 |
| 默认存储实例 | `tenants.default_storage_backend_id` | 未显式绑定时的新知识库默认存储实例 |
| 存储实例 | `storage_backends` | 当前空间可用的对象/文件存储实例 |
| 知识库存储绑定 | `knowledge_bases.storage_backend_id` | 知识库绑定的具体存储实例 |
| 知识条目存储占用 | `knowledges.storage_size` | 单条知识内容估算或实际占用 |

需要补充的查询能力：

- 按空间聚合存储用量；
- 按知识库聚合存储用量；
- 按存储实例聚合知识库数量和用量；
- 识别超配额空间。

说明：

- 共享空间不需要存储字段。
- 订阅表不需要存储字段。
- 个人升级企业不迁移 `storage_used`，复制知识库时由复制流程在目标组织空间重新计量。

### 9.8 后台默认配置

后台默认配置不要求普通用户在创建知识库时选择。第一版建议优先复用现有 `system_settings`、模型默认标记、组织默认存储和检索引擎配置；如果配置来源较多、需要版本化，再新增统一配置档案表：

```sql
CREATE TABLE knowledge_base_config_profiles (
    id VARCHAR(36) PRIMARY KEY,
    scope_type VARCHAR(20) NOT NULL DEFAULT 'global',
    tenant_id BIGINT NOT NULL DEFAULT 0,
    knowledge_base_type VARCHAR(32) NOT NULL DEFAULT 'document',
    version VARCHAR(64) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

配置内容至少覆盖：

- 默认知识库类型和图标；
- 文档解析、抽取、切片；
- embedding、summary、VLM、OCR、ASR；
- 向量存储和检索策略；
- 默认存储实例；

配置解析优先级：

```text
全局默认配置
  -> 组织/空间默认配置
  -> 按知识库类型匹配
  -> 创建时生成最终知识库配置
```

### 9.9 Token/积分管理表（后续版本，不纳入本期）

以下表结构仅供后续版本评估，本期不新增 `credit_accounts`、`credit_transactions`、`credit_pricing_rules`、`member_credit_limits` 或相关聚合表。

新增 `credit_accounts`：

```sql
CREATE TABLE credit_accounts (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    balance BIGINT NOT NULL DEFAULT 0,
    frozen_balance BIGINT NOT NULL DEFAULT 0,
    monthly_grant BIGINT NOT NULL DEFAULT 0,
    low_balance_threshold BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

约束：

```sql
CREATE UNIQUE INDEX uq_credit_accounts_tenant
ON credit_accounts(tenant_id);
```

字段说明：

| 字段 | 说明 |
| --- | --- |
| `account_type` | `personal` 或 `organization`，与 `tenants.space_type` 保持一致 |
| `balance` | 当前总余额，整数最小积分单位 |
| `frozen_balance` | 已预冻结但尚未结算的积分 |
| `monthly_grant` | 每月自动发放额度，第一版可为 0 |
| `low_balance_threshold` | 低余额提醒阈值 |
| `status` | `active/suspended` |

新增 `credit_transactions`：

```sql
CREATE TABLE credit_transactions (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    account_id VARCHAR(36) NOT NULL,
    user_id VARCHAR(36) NOT NULL DEFAULT '',
    actor_id VARCHAR(36) NOT NULL DEFAULT '',
    transaction_type VARCHAR(32) NOT NULL,
    amount BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    frozen_after BIGINT NOT NULL DEFAULT 0,
    operation_type VARCHAR(32) NOT NULL DEFAULT '',
    resource_type VARCHAR(32) NOT NULL DEFAULT '',
    resource_id VARCHAR(64) NOT NULL DEFAULT '',
    knowledge_base_id VARCHAR(36) NOT NULL DEFAULT '',
    session_id VARCHAR(36) NOT NULL DEFAULT '',
    task_id VARCHAR(64) NOT NULL DEFAULT '',
    model_id VARCHAR(64) NOT NULL DEFAULT '',
    pricing_rule_id VARCHAR(36) NOT NULL DEFAULT '',
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    raw_units BIGINT NOT NULL DEFAULT 0,
    billing_source VARCHAR(32) NOT NULL DEFAULT 'provider_reported',
    idempotency_key VARCHAR(128) NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

索引：

```sql
CREATE UNIQUE INDEX uq_credit_transactions_idempotency
ON credit_transactions(tenant_id, idempotency_key);

CREATE INDEX idx_credit_transactions_tenant_created
ON credit_transactions(tenant_id, created_at DESC);

CREATE INDEX idx_credit_transactions_user_created
ON credit_transactions(tenant_id, user_id, created_at DESC);

CREATE INDEX idx_credit_transactions_kb_created
ON credit_transactions(tenant_id, knowledge_base_id, created_at DESC);
```

新增 `credit_pricing_rules`：

```sql
CREATE TABLE credit_pricing_rules (
    id VARCHAR(36) PRIMARY KEY,
    provider VARCHAR(64) NOT NULL DEFAULT '',
    model_id VARCHAR(64) NOT NULL,
    capability VARCHAR(32) NOT NULL,
    input_token_rate BIGINT NOT NULL DEFAULT 0,
    output_token_rate BIGINT NOT NULL DEFAULT 0,
    cached_token_rate BIGINT NOT NULL DEFAULT 0,
    unit_type VARCHAR(32) NOT NULL DEFAULT 'token',
    unit_rate BIGINT NOT NULL DEFAULT 0,
    effective_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

说明：

- token 类模型使用 `input_token_rate/output_token_rate/cached_token_rate`。
- OCR、ASR、VLM、rerank 等可使用 `unit_type/unit_rate` 补充按页、分钟、图片、文档数计价。
- 计价规则变更不回写历史流水，历史流水保留当时的 `pricing_rule_id` 和金额。

新增 `member_credit_limits`：

```sql
CREATE TABLE member_credit_limits (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL,
    daily_limit BIGINT NOT NULL DEFAULT 0,
    monthly_limit BIGINT NOT NULL DEFAULT 0,
    daily_used BIGINT NOT NULL DEFAULT 0,
    monthly_used BIGINT NOT NULL DEFAULT 0,
    daily_reset_at TIMESTAMP WITH TIME ZONE,
    monthly_reset_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

约束：

```sql
CREATE UNIQUE INDEX uq_member_credit_limits_tenant_user
ON member_credit_limits(tenant_id, user_id);
```

可选聚合表：

```sql
CREATE TABLE credit_usage_daily (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    user_id VARCHAR(36) NOT NULL DEFAULT '',
    knowledge_base_id VARCHAR(36) NOT NULL DEFAULT '',
    model_id VARCHAR(64) NOT NULL DEFAULT '',
    operation_type VARCHAR(32) NOT NULL DEFAULT '',
    usage_date DATE NOT NULL,
    prompt_tokens BIGINT NOT NULL DEFAULT 0,
    completion_tokens BIGINT NOT NULL DEFAULT 0,
    cached_tokens BIGINT NOT NULL DEFAULT 0,
    credits_consumed BIGINT NOT NULL DEFAULT 0
);
```

说明：

- 第一版可以先只写流水，再通过 SQL 聚合展示。
- 当数据量增长后再引入 `credit_usage_daily` 做报表加速。

## 10. API 设计

### 10.1 创建知识库

创建知识库：

```text
POST /api/v1/knowledge-bases
```

创建请求：

```json
{
  "name": "招生资料",
  "description": "招生话术、活动方案和常见问答"
}
```

创建规则：

- 当前活动空间从登录上下文读取，不由请求体传入。
- 后端根据当前活动空间判断创建个人知识库或组织知识库。
- 后端自动注入后台默认配置。
- 前端不需要传递模型、解析、切片、向量或存储字段。

创建响应应至少返回：

```json
{
  "success": true,
  "data": {
    "id": "kb-id",
    "name": "招生资料",
    "description": "招生话术、活动方案和常见问答",
    "tenant_id": 10001,
    "owner_type": "organization",
    "config_source": "backend_default",
    "config_version": "org-default-v1"
  }
}
```

用户侧创建接口不接受以下字段：

```text
tenant_id
owner_type
storage_backend_id
vector_store_id
embedding_model_id
summary_model_id
chunking_config
extract_config
indexing_strategy
```

### 10.2 我的知识库列表

```text
GET /api/v1/knowledge-bases/my
```

查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| include_counts | bool | 是否返回文档数、分块数 |
| include_inaccessible_subscriptions | bool | 是否返回已失效订阅 |

### 10.3 共享空间

```text
GET    /api/v1/shared-spaces
POST   /api/v1/shared-spaces
GET    /api/v1/shared-spaces/:id
PUT    /api/v1/shared-spaces/:id
DELETE /api/v1/shared-spaces/:id
```

创建请求：

```json
{
  "name": "招生资料共享空间",
  "description": "面向招生顾问共享招生话术、活动方案和常见问答"
}
```

### 10.4 共享空间成员

```text
GET    /api/v1/shared-spaces/:id/members
POST   /api/v1/shared-spaces/:id/members
PUT    /api/v1/shared-spaces/:id/members/:member_id
DELETE /api/v1/shared-spaces/:id/members/:member_id
```

添加成员请求：

```json
{
  "user_id": "user-id",
  "role": "viewer"
}
```

### 10.5 共享空间知识库

```text
GET    /api/v1/shared-spaces/:id/knowledge-bases
POST   /api/v1/shared-spaces/:id/knowledge-bases
DELETE /api/v1/shared-spaces/:id/knowledge-bases/:kb_id
```

添加知识库请求：

```json
{
  "knowledge_base_id": "kb-id",
  "permission": "viewer"
}
```

### 10.6 订阅

```text
POST   /api/v1/knowledge-bases/:id/subscribe
DELETE /api/v1/knowledge-bases/:id/subscribe
GET    /api/v1/knowledge-bases/subscriptions
```

订阅响应：

```json
{
  "success": true,
  "data": {
    "knowledge_base_id": "kb-id",
    "subscribed": true
  }
}
```

### 10.7 个人升级企业

```text
POST /api/v1/enterprise-upgrade/preview
POST /api/v1/enterprise-upgrade
POST /api/v1/knowledge-bases/:id/copy-to-organization
```

`preview` 响应示例：

```json
{
  "success": true,
  "data": {
    "can_upgrade": true,
    "personal_space": {
      "id": 10000,
      "name": "我的个人空间"
    },
    "personal_knowledge_bases": []
  }
}
```

`enterprise-upgrade` 请求示例：

```json
{
  "name": "睿乐教育",
  "industry": "教培",
  "size": "11-50"
}
```

`copy-to-organization` 请求示例：

```json
{
  "target_tenant_id": 10001,
  "target_name": "招生资料知识库",
  "copy_mode": "settings_only"
}
```

`copy_mode` 第一版建议只支持：

- `settings_only`：复制知识库配置，不复制文档和索引；
- 后续再扩展 `full_copy`：复制文档、资源绑定和索引重建任务。

### 10.8 存储管理

复用现有存储实例接口：

```text
GET    /api/v1/storage-backends/types
POST   /api/v1/storage-backends/test
POST   /api/v1/storage-backends
GET    /api/v1/storage-backends
GET    /api/v1/storage-backends/:id
PUT    /api/v1/storage-backends/:id
DELETE /api/v1/storage-backends/:id
POST   /api/v1/storage-backends/:id/test
PUT    /api/v1/storage-backends/:id/default
```

建议新增空间用量接口：

```text
GET /api/v1/storage/usage
GET /api/v1/storage/usage/knowledge-bases
GET /api/v1/storage/usage/backends
```

`GET /api/v1/storage/usage` 响应示例：

```json
{
  "success": true,
  "data": {
    "tenant_id": 10001,
    "space_type": "organization",
    "quota_bytes": 10737418240,
    "used_bytes": 2147483648,
    "remaining_bytes": 8589934592,
    "usage_ratio": 0.2,
    "exceeded": false
  }
}
```

`GET /api/v1/storage/usage/knowledge-bases` 响应示例：

```json
{
  "success": true,
  "data": [
    {
      "knowledge_base_id": "kb-id",
      "name": "招生资料知识库",
      "owner_type": "organization",
      "storage_backend_id": "backend-id",
      "storage_bytes": 104857600
    }
  ]
}
```

知识库创建和复制接口不要求普通用户传递存储字段：

- 创建知识库时，后端自动使用当前空间默认存储策略。
- 复制个人知识库到组织空间时，后端自动使用目标组织默认存储实例。
- `storage_backend_id` 只作为后端内部解析结果和运维迁移字段，不出现在普通用户创建表单中。

### 10.9 后台默认配置

后台默认配置仅供系统/后台管理员维护，普通用户无访问入口。

```text
GET  /api/v1/system/admin/knowledge-base-defaults
GET  /api/v1/system/admin/knowledge-base-defaults/:scope
PUT  /api/v1/system/admin/knowledge-base-defaults/:scope
POST /api/v1/system/admin/knowledge-base-defaults/validate
POST /api/v1/system/admin/knowledge-base-defaults/preview
```

支持的 `scope`：

- `global`：平台全局默认；
- `tenant/:id`：组织或空间默认；
- `type/:knowledge_base_type`：按知识库类型的默认。

后台配置更新规则：

- 保存前校验模型、解析、向量和存储配置是否完整可用；
- 保存后生成新的 `config_version`；
- 新建知识库使用最新有效版本；
- 已有知识库不自动重写；
- 需要修改已有知识库时，必须通过显式迁移接口并产生审计记录。

### 10.10 Token/积分管理 API（后续版本，不纳入本期）

以下接口仅作为后续版本草案，本期不开发、不注册路由、不增加前端调用。

面向当前登录空间的积分接口：

```text
GET /api/v1/credits/account
GET /api/v1/credits/usage
GET /api/v1/credits/transactions
GET /api/v1/credits/pricing
POST /api/v1/credits/estimate
```

`GET /api/v1/credits/account` 响应示例：

```json
{
  "success": true,
  "data": {
    "tenant_id": 10001,
    "space_type": "organization",
    "balance": 2000000,
    "frozen_balance": 100000,
    "available_balance": 1900000,
    "monthly_consumed": 350000,
    "low_balance_threshold": 200000,
    "status": "active"
  }
}
```

`GET /api/v1/credits/usage` 查询参数：

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| start_date | string | 开始日期 |
| end_date | string | 结束日期 |
| group_by | string | `day/user/knowledge_base/model/operation` |
| user_id | string | 组织管理员可按员工筛选 |
| knowledge_base_id | string | 按知识库筛选 |
| operation_type | string | 按操作类型筛选 |

响应示例：

```json
{
  "success": true,
  "data": {
    "total_consumed": 350000,
    "items": [
      {
        "group": "chat",
        "prompt_tokens": 120000,
        "completion_tokens": 30000,
        "cached_tokens": 20000,
        "credits_consumed": 180000
      }
    ]
  }
}
```

`GET /api/v1/credits/transactions` 响应示例：

```json
{
  "success": true,
  "data": [
    {
      "id": "tx-id",
      "transaction_type": "consume",
      "amount": -1200,
      "balance_after": 1998800,
      "operation_type": "chat",
      "knowledge_base_id": "kb-id",
      "model_id": "gpt-5-mini",
      "prompt_tokens": 1000,
      "completion_tokens": 200,
      "billing_source": "provider_reported",
      "created_at": "2026-09-12T10:00:00Z"
    }
  ]
}
```

`POST /api/v1/credits/estimate` 请求示例：

```json
{
  "operation_type": "knowledge_import",
  "knowledge_base_id": "kb-id",
  "model_id": "embedding-model-id",
  "input": {
    "file_count": 3,
    "estimated_pages": 120,
    "estimated_chars": 300000
  }
}
```

响应示例：

```json
{
  "success": true,
  "data": {
    "billing_tenant_id": 10001,
    "estimated_credits": 56000,
    "available_balance": 1900000,
    "can_run": true
  }
}
```

组织管理员接口：

```text
GET /api/v1/credits/member-limits
PUT /api/v1/credits/member-limits/:user_id
PUT /api/v1/credits/settings
```

`PUT /api/v1/credits/member-limits/:user_id` 请求示例：

```json
{
  "daily_limit": 50000,
  "monthly_limit": 1000000,
  "status": "active"
}
```

系统管理员接口：

```text
GET  /api/v1/system/admin/credit-pricing-rules
POST /api/v1/system/admin/credit-pricing-rules
PUT  /api/v1/system/admin/credit-pricing-rules/:id
POST /api/v1/system/admin/tenants/:id/credits/grant
POST /api/v1/system/admin/tenants/:id/credits/adjust
GET  /api/v1/system/admin/tenants/:id/credits/transactions
```

`grant` 请求示例：

```json
{
  "amount": 1000000,
  "reason": "enterprise trial initial grant"
}
```

内部服务接口不直接暴露为公网 API，建议封装为 `CreditService`：

```text
ResolveBillingSubject(ctx, knowledgeBaseIDs, operationType)
Estimate(ctx, request)
Authorize(ctx, request)
Reserve(ctx, request)
Commit(ctx, usage)
Refund(ctx, reservationID, reason)
Release(ctx, reservationID)
```

## 11. 前端设计

### 11.1 知识库首页

一级结构：

```text
知识库
  - 我创建的
  - 共享给我的
  - 我订阅的
```

卡片信息：

- 知识库名称；
- 类型：文档 / FAQ / Wiki；
- 来源：
  - 个人知识库；
  - 组织知识库；
  - 来自共享空间：xxx；
- 权限：
  - 可管理；
  - 可编辑；
  - 仅查看；
- 是否已订阅；
- 文档数量、解析状态。

### 11.2 个人知识库设置

个人知识库创建入口：

```text
创建知识库
  - 名称
  - 描述
  - 创建
```

当前正在使用的个人空间自动作为知识库归属，不显示空间选择和高级配置。

个人知识库设置页不展示：

- 共享空间；
- 分享；
- 发布；
- 允许订阅；
- 成员权限。

个人知识库设置页仅保留：

- 名称；
- 描述；
- 内容管理；
- 删除知识库。

### 11.3 组织知识库设置

组织知识库创建入口：

```text
创建知识库
  - 名称
  - 描述
  - 创建
```

当前正在使用的组织空间自动作为知识库归属。创建后直接进入内容管理，不要求继续配置模型、解析、切片、向量或存储。

组织知识库设置页展示：

- 基础信息；
- 文档管理；
- 加入共享空间；
- 已加入共享空间列表；
- 订阅状态。

组织知识库设置页不展示：

- 模型选择；
- 解析配置；
- 切片配置；
- 向量存储选择；
- 存储实例选择；

### 11.4 共享空间管理页

入口建议放在组织管理下：

```text
组织管理
  - 员工
  - 共享空间
      - 空间列表
      - 空间成员
      - 空间知识库
```

共享空间详情页：

- 基础信息；
- 成员列表；
- 知识库列表；
- 操作日志。

### 11.5 订阅交互

订阅按钮展示规则：

- 当前用户有查看权限；
- 不是个人知识库；
- 当前知识库未被当前用户订阅。

取消订阅按钮展示规则：

- 当前用户已经订阅该知识库。

### 11.6 个人升级企业向导

升级入口：

- 个人空间顶部；
- 账号菜单；
- 新用户引导页。

向导步骤：

```text
1. 创建组织
   - 组织名称
   - 行业
   - 规模

2. 管理员设置
   - 当前账号默认为 Owner
   - 提示后续可邀请成员

3. 处理个人知识库
   - 默认全部保留在个人空间
   - 可勾选复制到组织空间

4. 完成
   - 进入组织空间
   - 明确提示个人知识库仍然私有
```

### 11.7 存储管理页面

个人空间存储页：

```text
个人空间
  - 当前配额
  - 已用 / 剩余 / 使用率
  - 按个人知识库展示用量
  - 订阅知识库不计入用量的提示
```

组织空间存储页：

```text
组织管理
  - 存储概览
      - 总配额
      - 已用 / 剩余 / 使用率
      - 超配额提醒
  - 存储实例
      - 实例列表
      - 新建实例
      - 测试连接
      - 设置默认实例
  - 知识库用量
      - 按知识库排序
      - 按存储实例筛选
```

知识库创建/设置页：

- 创建表单只包含名称和描述。
- 当前活动空间自动决定知识库归属。
- 存储实例由后台默认配置自动绑定。
- 模型、解析、切片、向量、检索和存储配置由后台默认配置自动注入。
- 配额不足时，上传按钮和复制动作应展示明确阻断原因。

### 11.8 Token/积分管理页面（后续版本，不纳入本期）

以下页面仅作为后续版本草案，本期不开发、不加入主前端或 Admin 导航。

个人空间积分页：

```text
个人空间
  - Token 积分概览
      - 当前余额
      - 本月消耗
      - 冻结积分
      - 低余额提醒
  - 消耗明细
      - 按个人知识库
      - 按模型
      - 按操作类型
      - 最近流水
```

组织空间积分页：

```text
组织管理
  - Token 积分概览
      - 当前余额
      - 可用余额
      - 本月消耗
      - 低余额提醒
  - 用量分析
      - 按员工
      - 按知识库
      - 按模型
      - 按共享空间
  - 员工限额
      - 默认日/月限额
      - 单个员工日/月限额
      - 超额状态
  - 流水记录
      - 消耗
      - 发放
      - 调整
      - 退款
```

知识库首页和问答页：

- 知识库卡片可显示本月积分消耗。
- 个人知识库显示“消耗个人积分”。
- 组织共享或订阅知识库显示“消耗组织积分”。
- 低余额时在问答输入框、上传入口、解析入口展示提示。
- 积分不足时，禁用会产生模型成本的操作，并保留只读浏览能力。

系统 admin 积分管理页：

```text
系统管理
  - 空间积分账户
      - 搜索个人/组织空间
      - 查看余额和消耗
      - 发放积分
      - 手工调整
  - 模型计价规则
      - 模型列表
      - 能力类型
      - 输入/输出/缓存 token 单价
      - 非 token 计价单位
      - 生效状态
  - 积分审计
      - 人工调整记录
      - 异常扣费记录
      - 余额不足记录
```

## 12. 权限与异常处理

| 场景 | 处理 |
| --- | --- |
| 个人 KB 尝试加入共享空间 | 403，提示个人知识库不可共享 |
| 非组织员工加入共享空间 | 400 或 403，提示只能添加本组织员工 |
| 添加外组织 KB 到共享空间 | 403，提示只能添加本组织知识库 |
| 失去访问权后打开订阅 KB | 403，提示当前无访问权限 |
| 共享空间被删除后访问其 KB | 不通过该空间授权；若无其他授权则 403 |
| 重复订阅 | 幂等返回成功 |
| 重复添加共享空间成员 | 幂等返回已有成员或 409，产品上建议提示已存在 |
| 重复添加空间知识库 | 幂等返回已有关系或 409，产品上建议提示已存在 |
| 存储配额不足 | 400 或 403，提示当前空间存储容量不足 |
| 选择其他空间存储实例 | 403，提示存储实例不可用 |
| 默认存储实例被禁用 | 阻止新建知识库或回退到系统默认策略，并提示管理员处理 |

## 13. 迁移与兼容

### 13.1 存量知识库归属

迁移策略：

- 已有 `knowledge_bases.creator_id` 的知识库继续保留创建者。
- 若知识库所属空间是组织空间，则视为组织知识库。
- 若引入个人空间，需要为每个用户创建或补齐个人空间。
- 存量空间先按成员数、Owner、历史共享和 API Key 使用情况进行分类；无法确认的空间保留 `legacy` 兼容态，不强行标记为个人或组织。

### 13.2 现有共享能力兼容

当前系统已有跨空间共享能力，第一版建议：

- 保留现有跨空间共享接口，不在本 PRD 中扩大使用范围。
- 新增组织内部共享空间能力时，使用独立路由和表，避免和旧 `organizations / kb_shares` 语义冲突。
- 后续如要统一，可做一轮产品命名治理：跨空间共享空间、组织内部共享空间、组织空间三者分开命名。

### 13.3 API 兼容

- 保留 `GET /api/v1/knowledge-bases` 当前语义，避免破坏旧前端和 SDK。
- 新增 `GET /api/v1/knowledge-bases/my` 承担新列表。
- 旧列表页迁移完成后，再评估是否把默认列表切到账号维度。

### 13.4 个人升级企业兼容

- 升级企业不修改原个人空间 ID。
- 升级企业不修改原个人知识库的 `tenant_id`、`creator_id`、文件、索引、订阅关系。
- 新组织空间复用现有 `tenants` 和 `tenant_members`。
- 原个人账号获得一个新的组织成员身份，因此登录响应中的 memberships 会增加一条组织空间记录。
- 现有空间切换器可直接用于个人空间和组织空间切换。
- 若当前部署没有个人空间概念，第一版可以把“升级企业”限定为新注册个人账号之后的流程；存量空间先保持 `legacy` 兼容态，后续由管理员确认类型。

### 13.5 存储管理兼容

- 保留现有 `/storage-backends` API，不重命名。
- 保留现有 `tenants.storage_quota`、`tenants.storage_used` 语义。
- 保留现有 `knowledge_bases.storage_backend_id` 绑定逻辑。
- 新增存储用量接口只做聚合展示，不改变已有上传、解析和存储写入流程。
- 升级企业创建的新组织空间使用当前系统默认空间配额和默认存储策略。
- 新建知识库和复制个人知识库到组织空间时，均自动使用目标空间默认存储实例，不允许普通用户显式选择其他存储实例。
- 存量知识库已有的 `storage_backend_id` 继续生效；后台管理员可通过迁移流程调整。

### 13.6 知识库默认配置兼容

- 保留现有 `knowledge_bases` 中的模型、解析、切片、向量和索引字段，继续作为运行时配置载体。
- 新建知识库时由 `KnowledgeBaseDefaultsService` 自动填充这些字段。
- 现有知识库不因本次改版被重置配置。
- 旧接口如果仍允许提交高级配置，第一阶段可以继续兼容读取，但新前端不再发送；后续应在 handler DTO 中移除或忽略普通用户字段。
- 后台默认配置变更只作用于新建知识库。
- 对已有知识库的配置迁移必须显式执行，并记录配置版本、操作者、迁移前后摘要和审计日志。
- 创建接口返回 `config_source`、`config_version`，不要求前端展示具体模型和存储配置。

### 13.7 Token/积分管理兼容（后续版本）

以下兼容策略暂不执行。本期不创建积分账户、不迁移历史 token usage，也不启用 report_only 或 enforce 模式。

- 存量空间迁移时为每个 `tenants` 行补齐一条 `credit_accounts`。
- 存量空间默认初始积分由部署策略决定：SaaS 可按套餐发放，私有化部署可默认给较大额度或关闭硬阻断。
- 历史模型调用 usage 不强制生成扣费流水，避免补扣争议。
- 已有 `messages`、`message_suggestion_sets` 中的 token 字段可用于历史统计参考，但不回写为真实扣费。
- Langfuse 中已有成本观测可作为运营分析参考，不作为积分余额的唯一事实来源。
- 新增积分流水后，以 `credit_transactions` 为余额事实来源。
- 部署初期可开启 `credit_enforcement = report_only`，只记录和展示消耗，不阻断请求；稳定后切换为 `enforce`。
- 个人升级企业创建的新组织空间需要同步创建组织积分账户。
- 个人积分和组织积分不自动合并；如果运营需要转赠，应走系统管理员手工 `grant/adjust` 并产生审计。

## 14. 埋点与审计

建议记录以下事件：

| 事件 | 说明 |
| --- | --- |
| `shared_space.created` | 创建共享空间 |
| `shared_space.updated` | 更新共享空间 |
| `shared_space.deleted` | 删除共享空间 |
| `shared_space.member_added` | 添加共享空间成员 |
| `shared_space.member_removed` | 移除共享空间成员 |
| `shared_space.member_role_changed` | 修改共享空间成员角色 |
| `shared_space.kb_added` | 添加知识库到共享空间 |
| `shared_space.kb_removed` | 从共享空间移除知识库 |
| `knowledge_base.subscribed` | 订阅知识库 |
| `knowledge_base.unsubscribed` | 取消订阅 |
| `knowledge_base.access_denied` | 知识库访问被拒绝 |
| `storage_backend.created` | 创建存储实例 |
| `storage_backend.updated` | 更新存储实例 |
| `storage_backend.deleted` | 删除存储实例 |
| `storage_backend.default_changed` | 修改默认存储实例 |
| `storage.quota_exceeded` | 存储配额不足导致操作被拒绝 |
| `knowledge_base.created_with_defaults` | 使用后台默认配置创建知识库 |
| `knowledge_base.config_migrated` | 后台迁移已有知识库配置 |
| `knowledge_base.default_config_changed` | 修改后台默认配置 |

审计要求：

- 组织管理员可以查看本组织共享空间变更记录。
- 系统管理员可以跨组织审计。
- 普通员工不看到审计日志。
- 知识库默认配置变更和已有知识库配置迁移必须记录配置版本和操作者。
- Token/积分审计待后续版本单独设计。

## 15. 验收标准

### 15.1 个人知识库

- 个人用户可以创建个人知识库。
- 个人知识库只有本人可见。
- 个人知识库没有共享空间入口。
- 后端拒绝个人知识库共享和被订阅。

### 15.2 组织知识库

- 组织成员可按角色创建组织知识库。
- 组织管理员可以管理组织内知识库。
- 普通员工不能看到未授权的组织知识库。

### 15.3 共享空间

- 组织管理员可以创建共享空间。
- 可以把本组织员工加入共享空间。
- 可以把本组织知识库加入共享空间。
- 共享空间成员可以在「共享给我的」看到空间知识库。
- 移除成员或知识库后，可见性立即失效。

### 15.4 订阅

- 有访问权的员工可以订阅共享知识库。
- 订阅后知识库出现在「我订阅的」。
- 取消订阅后不再出现。
- 失去访问权后，订阅不能继续提供访问能力。

### 15.5 列表

- 登录账号可以看到三类列表：我创建的、共享给我的、我订阅的。
- 同一知识库在不同分类中展示一致的名称、图标、类型和状态。
- 列表接口返回的权限字段与实际操作权限一致。

### 15.6 个人升级企业

- 个人账号可以创建组织空间。
- 当前账号自动成为新组织 Owner。
- 个人空间仍在空间切换器中可见。
- 个人知识库未被自动共享给组织。
- 勾选复制的个人知识库在组织空间中生成新的组织知识库。
- 复制后的组织知识库可以加入共享空间。

### 15.7 存储管理

- 个人空间可查看存储配额和已用量。
- 组织管理员可查看组织存储配额和已用量。
- 组织管理员可创建、测试、更新、删除存储实例。
- 组织管理员可设置默认存储实例。
- 新组织知识库默认绑定组织默认存储实例。
- 共享空间和订阅不增加额外存储用量。
- 配额不足时，上传、解析、复制等新增存储占用的操作会被阻断。

### 15.8 Token/积分管理（后续版本，不纳入本期验收）

以下验收标准暂不执行，待后续版本单独立项。

- 个人空间可查看积分余额、本月消耗和最近流水。
- 组织管理员可查看组织积分余额、员工用量、知识库用量和模型用量。
- 个人知识库问答、解析、embedding 等操作消耗个人积分。
- 组织知识库问答、解析、embedding 等操作消耗组织积分。
- 员工通过共享空间或订阅入口访问组织知识库时，消耗组织积分。
- 共享空间和订阅不创建独立积分账户。
- 个人升级企业后，个人积分账户保留，新组织空间拥有独立积分账户。
- 组织管理员可设置员工日/月积分限额。
- 积分余额不足或员工限额不足时，会产生模型成本的操作被阻断。
- 模型调用成功后产生积分流水，且重复请求不会重复扣费。
- 供应商未返回 usage 时，系统按估算用量扣费并标记 `billing_source = estimated`。

### 15.9 知识库创建与默认配置

- 创建知识库表单只展示名称和描述。
- 当前活动空间自动决定知识库归属。
- 创建请求不接受 `tenant_id`、模型、解析、切片、向量和存储配置。
- 创建成功后，后台自动写入默认配置和配置版本。
- 创建完成后用户可以直接进入内容上传、检索和问答流程。
- 个人和组织知识库设置页均不展示高级配置入口。
- 后台默认配置变更不影响已有知识库。

## 16. 四版本实施路线

本方案使用 4 个版本完成设计和开发。每个版本都应具备独立发布、独立验收和独立回滚能力；版本之间通过新增数据结构和兼容 API 逐步演进，不通过一次性大迁移切换线上知识库。

| 版本 | 产品目标 | 主要上线内容 | 线上策略 |
| --- | --- | --- | --- |
| V1 | 兼容基础与极简创建 | 空间类型兼容字段、统一权限解析、默认配置注入、创建接口收敛为名称和描述 | 新表和新字段先灰度，旧列表和旧共享能力保持不变 |
| V2 | 组织内部协作 | 内部共享空间、成员/知识库管理、订阅、三分类知识库列表 | 不转换历史跨空间共享，先对新组织或灰度租户启用 |
| V3 | 企业化与存储治理 | 个人升级企业、知识库复制、空间存储用量、配额提示、Admin 管理 | 升级新增组织空间，复制生成新知识库，原个人数据不变 |
| V4 | 后台治理与存量收尾 | 默认配置后台管理、存量配置迁移、审计、批量操作、失效订阅和旧入口收敛 | 所有存量变更显式执行，观察期后再关闭旧 UI，保留旧 API |

四个版本均不开发 Token/积分账户、计价、扣减、充值、额度和余额拦截。本期只保留现有技术 token usage 和观测链路。

## 17. 基于当前工程的改版规划

本节中的 P0/P1/P2/P3 是研发实施子任务，不是对外发布版本。对外版本和上线边界以 V1/V2/V3/V4 为准；一个版本可以包含多个 P 子任务，但不得跨版本提前开启未验收能力。

### 17.1 当前工程可复用能力

当前工程已经具备多项基础能力，可以复用而不是重建：

| 能力 | 当前落点 | 复用方式 |
| --- | --- | --- |
| 空间实体 | `internal/types/tenant.go`、`tenants` 表 | 增补可空的 `space_type`；Owner 继续以 `tenant_members.role=owner` 为准 |
| 空间成员和角色 | `internal/types/tenant_member.go`、`tenant_members` 表 | 继续作为组织员工和组织角色来源 |
| 知识库归属 | `internal/types/knowledgebase.go`、`knowledge_bases.tenant_id` | 继续表示知识库所属空间 |
| 创建者归属 | `knowledge_bases.creator_id` | 继续支撑“我创建的”和创建者管理权 |
| 知识库列表与详情 | `internal/handler/knowledgebase.go` | 新增账号维度列表，并逐步替换散落权限判断 |
| 路由注册 | `internal/router/router.go` | 新增 shared-spaces、subscriptions、enterprise-upgrade 路由组 |
| 组织/共享空间旧能力 | `internal/types/organization.go`、`internal/handler/organization.go` | 暂保留为跨空间共享能力，不直接承载组织内部共享空间 |
| 主前端知识库页 | `frontend/src/views/knowledge/KnowledgeBaseList.vue` | 改成三分类视图 |
| 主前端资源缓存 | `frontend/src/stores/chatResources.ts` | 增加 my knowledge bases 缓存 |
| 组织管理 UI | `admin/src/views/organization/OrganizationList.vue`、`frontend/src/views/organization/*` | 可借鉴成员选择和空间管理交互 |
| Admin 知识库治理 | `admin/src/views/AdminKnowledgeBases.vue`、`admin/src/views/AdminKnowledgeBaseSettings.vue` | 组织知识库管理和共享空间管理优先放 admin |
| 存储配额 | `internal/types/tenant.go`、`internal/application/repository/tenant.go` | 复用 `storage_quota/storage_used` 和 `AdjustStorageUsed` |
| 存储实例 | `internal/types/storagebackend.go`、`internal/application/service/storagebackend.go`、`docs/api/storage-backend.md` | 复用现有 StorageBackend API |
| 存储前端 | `frontend/src/views/settings/StorageBackendSettings.vue`、`frontend/src/views/settings/TenantInfo.vue`、`admin/src/views/settings/StorageBackendSettings.vue` | 复用存储实例管理和空间用量展示 |
| 知识库默认配置 | `internal/types/knowledgebase.go`、`internal/application/service/knowledgebase.go` | 复用现有知识库配置字段，由后台默认配置解析器在创建时自动注入 |
| 知识库存储绑定 | `knowledge_bases.storage_backend_id`、`admin/src/views/AdminKnowledgeBaseSettings.vue` | 后端自动绑定当前空间默认存储；普通用户不选择存储实例 |
| 模型 Token usage 与观测 | `internal/types/chat.go`、`internal/tracing/langfuse/*`、`internal/models/*/langfuse_wrapper.go` | 保持现有技术用量记录和观测链路，本期不转化为产品积分 |
| 模型配置与能力 | `internal/application/service/model.go`、`internal/types/model.go` | 复用现有模型、解析和检索能力配置 |

### 17.2 当前工程需要避免的复用误区

1. 不建议直接复用现有 `organizations / kb_shares` 做组织内部共享空间。该模型当前偏跨空间共享，直接承载组织内部共享会让“组织空间”和“共享空间”概念混淆。
2. 不建议只在前端隐藏个人知识库共享入口。个人知识库不可共享必须在后端路由和 service 层强校验。
3. 不建议继续把权限散落在 `validateAndGetKnowledgeBase`、`callerCanViewTenantKnowledgeBase`、前端 `canEdit` 中重复判断。需要抽一个统一访问解析器。
4. 不建议升级企业时修改个人空间类型。应新增组织空间。
5. 不建议为共享空间和订阅新增独立存储池。共享空间和订阅都应引用原知识库，不复制存储。
6. 不建议把知识库高级配置继续放在普通用户创建或设置流程中。创建流程只收集名称和描述，默认配置统一由后端解析。

### 17.3 后端改造计划

#### 后端 P0：迁移与类型

新增迁移，建议命名为下一版本号，例如：

```text
migrations/versioned/000082_personal_org_shared_spaces.up.sql
migrations/versioned/000082_personal_org_shared_spaces.down.sql
```

迁移内容：

- `tenants.space_type`
- `knowledge_bases.config_source`
- `knowledge_bases.config_version`
- 可选：`knowledge_base_config_profiles`
- `shared_spaces`
- `shared_space_members`
- `shared_space_knowledge_bases`
- `knowledge_base_subscriptions`

存储管理不需要新增核心存储表，优先复用：

- `tenants.storage_quota`
- `tenants.storage_used`
- `tenants.default_storage_backend_id`
- `knowledge_bases.storage_backend_id`
- `knowledges.storage_size`
- `knowledge_bases` 现有模型、解析、切片、索引和向量配置字段

类型文件：

```text
internal/types/tenant.go
internal/types/knowledgebase.go
internal/types/shared_space.go        # 新增
internal/types/knowledge_subscription.go  # 新增
internal/types/knowledgebase_config_profile.go # 新增
```

接口文件：

```text
internal/types/interfaces/shared_space.go       # 新增
internal/types/interfaces/knowledge_subscription.go  # 新增
```

#### 后端 P1：Repository 和 Service

新增 repository：

```text
internal/application/repository/shared_space.go
internal/application/repository/knowledge_subscription.go
```

新增 service：

```text
internal/application/service/shared_space.go
internal/application/service/knowledge_subscription.go
internal/application/service/knowledgebase_access.go
internal/application/service/enterprise_upgrade.go
internal/application/service/storage_usage.go
internal/application/service/knowledgebase_defaults.go
```

`knowledgebase_access.go` 建议提供统一方法：

```text
ResolveKnowledgeBaseAccess(ctx, kbID) (*KnowledgeBaseAccess, error)
ListMyKnowledgeBases(ctx) (*MyKnowledgeBaseList, error)
```

返回结构建议包含：

```text
permission: viewer/editor/manager
access_source: created/organization_admin/shared_space/subscription
owner_type: personal/organization
shared_space_id
is_subscribed
effective_tenant_id
```

`knowledgebase_defaults.go` 建议提供默认配置解析：

```text
ResolveDefaults(ctx, tenantID, knowledgeBaseType) (*KnowledgeBaseDefaults, error)
ApplyDefaults(ctx, knowledgeBase, defaults) error
GetConfigVersion(ctx, tenantID) (string, error)
```

默认配置解析器负责：

- 读取全局默认配置；
- 读取组织/空间默认配置；
- 按知识库类型合并配置；
- 解析默认模型、解析、切片、向量、检索和存储策略；
- 将配置写入知识库现有配置字段；
- 记录 `config_source` 和 `config_version`；
- 阻止普通用户通过 DTO 覆盖默认配置。

模型调用说明：

- 本期不修改现有 chat、embedding、OCR、VLM、ASR、rerank 和评测调用链。
- 现有模型返回的 token usage 继续按当前技术链路处理，不接入积分账户、流水或余额校验。

#### 后端 P2：Handler 与 Router

新增 handler：

```text
internal/handler/shared_space.go
internal/handler/knowledge_subscription.go
internal/handler/enterprise_upgrade.go
internal/handler/storage_usage.go
internal/handler/knowledgebase_defaults.go
```

修改现有 handler：

```text
internal/handler/knowledgebase.go
internal/handler/knowledge.go
```

重点修改：

- `ListKnowledgeBases` 保持旧接口兼容。
- 新增 `ListMyKnowledgeBases`。
- `validateAndGetKnowledgeBase` 逐步改为调用 `ResolveKnowledgeBaseAccess`。
- 知识库创建 handler 只接收名称和描述，并从当前活动空间解析 `tenant_id`。
- 知识库创建 service 调用 `ResolveDefaults` 和 `ApplyDefaults`，不接受用户传入的高级配置。
- 文档上传、列表、预览、搜索、删除、目录配置、FAQ、Wiki、标签等入口统一使用访问解析结果。
- 个人知识库在分享、订阅、加入共享空间路由中强制拒绝。

修改路由：

```text
internal/router/router.go
```

新增路由组：

```text
/api/v1/shared-spaces
/api/v1/knowledge-bases
/api/v1/knowledge-bases/my
/api/v1/knowledge-bases/:id/subscribe
/api/v1/knowledge-bases/subscriptions
/api/v1/enterprise-upgrade
/api/v1/storage/usage
/api/v1/system/admin/knowledge-base-defaults
```

#### 后端 P3：容器注入

需要在依赖注入中注册新增 repository、service、handler。

重点检查：

```text
internal/container/container.go
```

要确保新增服务拿得到：

- DB；
- TenantService；
- TenantMemberService；
- KnowledgeBaseService；
- UserService；
- AuditLogService。
- StorageBackendService；
- StorageBackendResolver。
- ModelService；
- KnowledgeBaseDefaultsService；

### 17.4 前端改造计划

#### 主前端 P0：API 和 Store

新增 API：

```text
frontend/src/api/shared-space/index.ts
frontend/src/api/knowledge-subscription/index.ts
frontend/src/api/enterprise-upgrade/index.ts
frontend/src/api/storage-usage/index.ts
```

修改：

```text
frontend/src/api/knowledge-base/index.ts
frontend/src/stores/chatResources.ts
frontend/src/stores/auth.ts
```

`chatResources` 增加：

- `myKnowledgeBases.created`
- `myKnowledgeBases.shared`
- `myKnowledgeBases.subscribed`
- `ensureMyKnowledgeBases(force)`
- `invalidate('myKnowledgeBases')`
- `storageUsage`
- `ensureStorageUsage(force)`

知识库创建 API 和 Store：

```text
frontend/src/api/knowledge-base/index.ts
frontend/src/views/knowledge/KnowledgeBaseEditorModal.vue
```

创建请求只发送：

```text
name
description
```

当前活动空间从 `authStore` 或空间上下文读取，前端不拼接或覆盖 `tenant_id`、模型、解析和存储配置。

#### 主前端 P1：知识库首页

重点文件：

```text
frontend/src/views/knowledge/KnowledgeBaseList.vue
frontend/src/views/knowledge/kbListMerge.ts
frontend/src/components/KnowledgeBaseMenu.vue
frontend/src/components/ResourceOriginBadge.vue
```

改造内容：

- 知识库首页改为三分类。
- 卡片展示 `owner_type`、`access_source`、`is_subscribed`。
- 个人知识库隐藏共享、发布、加入共享空间、订阅相关入口。
- 共享给我的知识库展示所属共享空间。
- 我订阅的知识库展示来源和失效状态。

#### 主前端 P2：升级企业向导

新增页面或弹窗：

```text
frontend/src/views/enterprise/EnterpriseUpgradeWizard.vue
```

接入入口：

```text
frontend/src/components/TenantSelector.vue
frontend/src/views/settings/TenantInfo.vue
frontend/src/stores/menu.ts
```

改造内容：

- 创建组织空间；
- 当前账号成为 Owner；
- 复制个人知识库到组织空间；
- 完成后刷新 auth memberships 和空间切换器。

#### 主前端 P3：个人存储展示

重点文件：

```text
frontend/src/views/settings/TenantInfo.vue
frontend/src/views/knowledge/KnowledgeBaseList.vue
frontend/src/views/knowledge/KnowledgeBase.vue
```

改造内容：

- 个人空间显示存储用量和配额。
- 知识库卡片或详情展示存储占用。
- 订阅知识库标记为“不占用个人存储”。
- 配额不足错误统一展示。

#### 主前端：Token/积分模块延期

本期不新增积分页面、余额、流水、扣费提示或额度管理；相关能力保留为后续版本独立模块。

#### Admin P1：共享空间管理

建议优先放在 admin 工程：

```text
admin/src/views/shared-space/SharedSpaceList.vue
admin/src/views/shared-space/SharedSpaceDetail.vue
admin/src/router/index.ts
admin/src/config/navigation.ts
```

功能：

- 共享空间列表；
- 创建/编辑/删除共享空间；
- 添加/移除空间成员；
- 添加/移除空间知识库；
- 查看审计记录。

#### Admin P2：知识库设置联动

重点文件：

```text
admin/src/views/AdminKnowledgeBaseSettings.vue
frontend/src/views/knowledge/KnowledgeBaseEditorModal.vue
frontend/src/views/knowledge/settings/KBShareSettings.vue
```

改造内容：

- 组织知识库设置页增加“内部共享空间”。
- 个人知识库不显示共享设置。
- 现有 `KBShareSettings` 保留给跨空间共享，不混入内部共享空间。
- 移除普通用户可见的解析配置、模型配置、向量存储和存储实例选择。
- 知识库设置页只保留名称、描述、内容管理和内部共享关系。

#### Admin P3：组织存储管理

重点文件：

```text
admin/src/views/settings/StorageBackendSettings.vue
admin/src/views/AdminKnowledgeBaseSettings.vue
admin/src/router/index.ts
admin/src/config/navigation.ts
```

改造内容：

- 在组织管理或数据管理中展示存储概览。
- 复用存储实例列表、测试连接、默认实例设置。
- 在后台知识库治理页展示当前绑定实例和配置版本，仅供管理员排查。
- 创建组织知识库时自动使用组织默认存储实例，不提供选择控件。
- 配额不足时提示组织管理员扩容或清理。
- 原 `KBStorageSettings.vue` 如保留，只能改为只读状态展示，不提供普通用户修改入口。

#### Admin：Token/积分模块延期

本期不新增组织积分管理、计价规则、员工限额、发放和审计页面；相关能力保留为后续版本独立模块。

#### Admin P4：知识库默认配置管理

重点文件：

```text
admin/src/views/settings/KnowledgeBaseDefaults.vue
admin/src/api/knowledge-base-defaults/index.ts
admin/src/router/index.ts
admin/src/config/navigation.ts
```

改造内容：

- 系统管理员维护全局知识库默认配置。
- 按组织或知识库类型维护覆盖配置。
- 保存前执行完整性校验和连通性校验。
- 展示当前生效版本和影响范围。
- 支持预览“新建知识库将使用的配置”。
- 支持对已有知识库发起显式配置迁移。
- 普通用户和组织普通管理员不进入该页面。

### 17.5 测试计划

后端建议新增测试：

```text
internal/application/service/shared_space_test.go
internal/application/service/knowledge_subscription_test.go
internal/application/service/knowledgebase_access_test.go
internal/application/service/storage_usage_test.go
internal/application/service/knowledgebase_defaults_test.go
internal/handler/shared_space_test.go
internal/handler/enterprise_upgrade_test.go
internal/handler/storage_usage_test.go
internal/handler/knowledgebase_create_test.go
internal/handler/knowledgebase_defaults_test.go
internal/router/shared_space_rbac_test.go
```

核心用例：

- 个人知识库不可加入共享空间。
- 个人知识库不可被他人订阅。
- 组织知识库加入共享空间后，空间成员可读。
- 移除共享空间成员后访问立即失效。
- 移除共享空间知识库后访问立即失效。
- 订阅不授予访问权。
- 升级企业不改变个人知识库归属。
- 复制到组织空间后可加入共享空间。
- 共享空间和订阅不增加存储用量。
- 个人知识库复制到组织空间后消耗组织空间配额。
- 配额不足时上传和复制被拒绝。
- 创建知识库只接收名称和描述。
- 当前活动个人空间/组织空间正确决定知识库归属。
- 后端自动写入默认模型、解析、向量和存储配置。
- 普通用户传入高级配置字段时被忽略或拒绝。
- 后台默认配置变更不影响已有知识库。
- 默认配置保存前执行完整性校验。
- 新建知识库使用最新有效配置版本。
- 普通用户无权访问后台默认配置接口。

前端建议新增测试：

```text
frontend/src/views/knowledge/KnowledgeBaseList.test.mjs
frontend/src/components/KnowledgeBaseMenu.test.mjs
frontend/src/stores/chatResources.test.ts
frontend/src/views/knowledge/KnowledgeBaseEditorModal.test.ts
frontend/src/api/storage-usage/storageUsage.test.ts
admin/src/views/settings/KnowledgeBaseDefaults.test.ts
```

验证：

- 三分类渲染；
- 个人知识库不显示共享入口；
- 订阅状态按钮；
- 来源徽章；
- 共享空间失效态。
- 存储用量展示；
- 配额不足错误提示。
- 创建弹窗只展示名称和描述；
- 创建成功后直接进入知识库内容页；
- 不展示模型、解析和存储配置控件。

### 17.6 验证命令

建议每个阶段至少执行：

```bash
go test ./internal/application/service ./internal/handler ./internal/router
npm --prefix frontend run type-check
npm --prefix admin run type-check
npm --prefix frontend test -- src/components/KnowledgeBaseMenu.test.mjs
npm --prefix admin test -- src/views/settings/KnowledgeBaseDefaults.test.ts
git diff --check
```

如涉及迁移：

```bash
go test ./internal/database ./internal/container
```

如涉及知识库列表 UI：

```bash
npm --prefix frontend test -- src/views/knowledge/KnowledgeBaseList.test.mjs
```

### 17.7 推荐交付顺序

| 版本 | 目标 | 后端/数据 | 主前端 | Admin/运营 | 存量迁移边界 |
| --- | --- | --- | --- | --- | --- |
| V1 | 兼容基础与极简创建 | 迁移、默认配置解析器、统一访问解析器、创建 DTO、feature flags | 创建弹窗只提交名称和描述；旧列表保持不变 | 先不开放复杂治理页面，提供必要的诊断数据 | 只做盘点和元数据回填，不改归属、不改共享关系、不移动文件 |
| V2 | 组织内部共享与订阅 | shared-spaces、subscriptions、账号维度聚合列表、权限扩展 | 三分类列表、共享来源、订阅/取消订阅、空间知识库入口 | 共享空间基本 CRUD、成员和知识库管理 | 不自动迁移 `organizations/kb_shares`，新能力仅写新表 |
| V3 | 企业升级与存储治理 | enterprise upgrade、复制任务、storage usage、配额校验、审计任务状态 | 升级企业、空间切换、复制知识库、用量和配额提示 | 组织存储概览、默认实例展示、共享空间治理增强 | 新建组织和复制任务可产生新数据，原个人空间完全保留 |
| V4 | 默认配置治理与存量收尾 | 配置版本、显式迁移、editor、批量操作、失效订阅、旧 API 兼容 | 失效态和错误态完善，旧入口收敛 | 默认配置管理、存量空间确认、配置迁移、审计报表 | 只迁移管理员明确选择的空间/知识库，观察期后再关闭旧 UI |

#### 17.7.1 V1：兼容基础与极简创建

**版本目标**

完成目标模型的底层兼容能力，但不改变线上用户现有的知识库列表和历史跨空间共享体验。V1 的核心是把后续能力所需的边界先固定下来。

**后端和数据开发**

- 新增可空的 `tenants.space_type`，取值为 `personal`、`organization`、`legacy`；存量不确定空间保持 `legacy`。
- 新增 `knowledge_bases.config_source`、`knowledge_bases.config_version`，存量知识库标记为 `legacy`，不重写其运行配置。
- 建立 `shared_spaces`、`shared_space_members`、`shared_space_knowledge_bases`、`knowledge_base_subscriptions` 表和索引，但暂不对全量用户开放写入。
- 新增统一 `ResolveKnowledgeBaseAccess`，先复现当前创建者、租户 RBAC、历史 `kb_shares` 的访问结果，再预留新共享空间和订阅来源。
- 新增 `CreateKnowledgeBaseRequest`，只允许名称和描述；`tenant_id`、模型、解析、切片、向量、存储等字段由后端上下文和默认配置生成。
- 兼容旧客户端传入的高级字段：第一阶段忽略并记录审计或 debug 信息，不能让旧客户端绕过默认配置。
- 新增默认配置解析服务，创建知识库时自动填充现有模型、解析、切片、向量、检索和存储字段。
- 增加版本开关：默认关闭新空间规则、共享空间读写和个人知识库新增共享拦截。

**前端和 Admin 开发**

- 主前端创建弹窗只保留名称和描述，并确认实际请求 payload 不再发送高级配置。
- 知识库列表暂不切换三分类，继续使用当前 `/knowledge-bases`。
- Admin 暂不开放完整默认配置治理页，只提供必要的配置完整性检查和错误提示。

**线上迁移与影响**

- 先生成租户、成员、知识库、历史分享、资源目录和存储绑定盘点报告。
- 只执行 DDL、索引和安全元数据回填，不创建新的共享关系。
- 不修改已有 `tenant_id`、`creator_id`、`vector_store_id`、`storage_backend_id` 和物理文件路径。
- 旧 API、旧前端和历史 `organizations/kb_shares` 继续可用。

**V1 退出标准**

- 旧知识库列表、详情、搜索、问答、上传和历史共享回归通过。
- 新建知识库只需名称和描述即可直接使用。
- 存量知识库的配置和物理存储没有被改写。
- 新旧权限解析结果对账一致。
- 出现问题时可以关闭 feature flags 恢复旧流程。

#### 17.7.2 V2：组织内部共享与订阅

**版本目标**

正式上线组织内部共享空间和账号维度的知识库入口，使员工能够查看组织分配给自己的共享知识库，并通过订阅建立个人快捷入口。

**后端和数据开发**

- 实现共享空间 CRUD。
- 实现共享空间成员添加、移除和角色校验；第一版空间成员权限固定为 `viewer`。
- 实现组织知识库加入和移除共享空间。
- 强制校验共享空间、空间成员和知识库属于同一组织空间。
- 个人知识库禁止新增共享、加入组织内部共享空间和被其他用户订阅。
- 实现 `/api/v1/knowledge-bases/my`，返回“我创建的、共享给我的、我订阅的”三类结果。
- 实现订阅、取消订阅和失权校验；订阅不参与授权。
- 三类列表同时兼容新的内部共享关系和历史 `kb_shares`，结果中必须标识来源。
- 处理共享成员移除、知识库移除和订阅失权后的即时权限失效。

**前端和 Admin 开发**

- 主前端知识库首页增加三分类入口和来源标识。
- 组织知识库卡片提供订阅/取消订阅；个人知识库不显示共享和订阅入口。
- 共享空间内的知识库显示“共享给我”来源，不复制知识库内容。
- Admin 增加共享空间列表、创建/编辑、成员管理和知识库管理。
- 保留当前跨空间共享页面，产品名称明确为“跨空间共享”，避免与内部共享空间混淆。

**线上迁移与影响**

- 不从现有 `organizations`、`organization_tenant_members`、`kb_shares` 自动生成新的内部共享空间。
- 新功能只写入新表，历史共享继续由原模型提供访问。
- 新列表上线前先进行双读对账：旧 `/knowledge-bases` 与新聚合接口对比“当前用户已有访问权的知识库”。
- 共享空间和订阅不复制文件、索引、资源绑定，不增加存储用量。

**V2 退出标准**

- 组织管理员可以创建共享空间、添加员工、添加组织知识库。
- 被加入共享空间的员工可以查看共享知识库。
- 移除员工或知识库后访问立即失效。
- 订阅只改变“我订阅的”入口，不扩大权限。
- 个人知识库所有新增共享入口在后端被拒绝。
- 历史 `kb_shares` 访问不丢失。

#### 17.7.3 V3：企业升级与存储治理

**版本目标**

完成个人账号升级企业、组织空间初始化、知识库复制，以及个人/组织空间的存储用量和配额管理。

**后端和数据开发**

- 新增企业升级接口和异步任务状态。
- 升级企业只创建新的组织租户，并写入当前账号的 `tenant_members.role=owner`。
- 原个人租户及其知识库、文件、索引、订阅和历史共享关系保持不变。
- 复制知识库使用独立任务，目标知识库生成新的 ID、`tenant_id` 和资源记录。
- 复制前估算原始文件、解析内容、索引和任务空间，校验目标组织配额。
- 复制默认不带入历史共享关系、订阅、收藏和 Pin 状态。
- 增加空间级用量、剩余配额、使用率和按知识库聚合的查询接口。
- 上传、解析、重建索引、批量导入和知识库复制统一执行配额校验。

**前端和 Admin 开发**

- 增加“升级企业”流程：组织名称、基本信息、创建结果和空间切换。
- 增加个人空间到组织空间的切换提示，明确原个人知识库不会自动共享。
- 知识库提供“复制到组织空间”入口，展示预计用量、目标空间剩余配额和任务进度。
- 个人/组织空间展示存储用量和配额不足提示。
- Admin 复用现有存储后端管理能力，增加组织存储概览、默认实例和配额展示。

**线上迁移与影响**

- 升级企业不执行 `UPDATE knowledge_bases SET tenant_id=...`，也不改变原知识库的物理路径。
- 复制失败只清理复制任务创建的目标资源，原个人知识库不回滚、不删除、不改变权限。
- 目标组织的存储用量包含复制产生的新文件、解析数据和索引；共享和订阅仍不增加用量。
- 本地存储到 OSS 的迁移作为独立运维任务执行，不和企业升级事务绑定。
- 对象存储迁移必须先部署包含最新迁移程序的镜像，先 dry-run，再 `--execute`，并保留本地源文件作为回退路径。

**V3 退出标准**

- 一个个人账号可以成功创建组织空间并完成空间切换。
- 原个人知识库在升级前后归属、访问和文件内容一致。
- 至少一个知识库能够完整复制到组织空间，并完成文档、分块、索引和访问校验。
- 配额不足时上传、解析和复制可以被明确拒绝。
- 存储用量能解释文件、解析和索引增加来源。

#### 17.7.4 V4：后台治理与存量收尾

**版本目标**

完成默认配置治理、存量空间确认、存量知识库显式迁移和产品收尾，使新旧模型长期共存时具备可管理、可审计、可回滚能力。

**后端和数据开发**

- Admin 管理全局默认配置、空间级覆盖、配置版本和生效时间。
- 默认配置发布前执行模型、解析、向量、存储和连通性完整性校验。
- 支持按单个知识库、指定空间和批量预览执行配置迁移。
- 对 `vector_store_id`、存储实例和索引重建等高风险变更使用异步任务，不在普通保存设置时隐式执行。
- 增加审计记录：共享空间变更、成员变更、订阅、企业升级、知识库复制、默认配置变更和配置迁移。
- 支持共享空间 `editor`、批量成员/知识库操作和失效订阅清理。
- 增加资源、存储实例、文件类型和知识库维度的用量细分。

**前端和 Admin 开发**

- 完善失效订阅、失去访问权、复制失败、配额不足和配置迁移中的状态展示。
- Admin 增加默认配置管理、配置预览、影响范围、迁移任务和审计查询。
- 对确认完成迁移的租户隐藏旧共享空间或旧入口，但保留旧 API 和数据库字段。
- 前端不展示 Token/积分入口，也不增加积分余额或消费提示。

**线上迁移与影响**

- 只有管理员明确选择的空间和知识库才执行存量迁移。
- `legacy` 空间不自动变成个人或组织；必须完成数据核对后再确认类型。
- 历史 `kb_shares` 不因新模型上线自动撤销。
- 默认配置变更只影响新建知识库；已有知识库必须显式发起迁移。
- 观察期内保留旧 API、旧表、旧字段和源存储，确认无误后再关闭旧 UI。

**V4 退出标准**

- 后台可以查看默认配置版本、影响范围、迁移进度和审计记录。
- 已确认的个人空间不能新增共享、内部共享空间或他人订阅。
- 组织共享空间支持完整的成员和知识库治理。
- 存量知识库迁移前后配置、内容、索引、权限和存储用量可对账。
- 旧模型关闭后仍可通过兼容 API 回滚到历史列表和权限逻辑。

### 17.8 当前工程实现基线（2026-09-12 核对）

本节以当前仓库代码、路由和数据库迁移为准，用于区分“已经存在的能力”和“本次改版需要新增的能力”。后续实施不能把规划中的表、接口或页面当成当前线上已经具备的能力。

| 能力 | 当前工程状态 | 改版处理 |
| --- | --- | --- |
| 租户/空间 | 已有 `tenants`，当前实际承担工作空间和数据隔离职责 | 继续复用 `tenant`，新增个人/组织空间标识时采用兼容字段，不改写已有 `tenant_id` |
| 组织成员与 RBAC | 已有 `tenant_members`、`viewer/contributor/admin/owner` 角色和知识库 `creator_id` | 继续作为组织成员和基础权限来源，新增共享空间权限时在其上叠加，不复制一套组织角色 |
| 创建者权限 | 已有创建者字段和 `KBAccessRead/Write` 等访问判断 | 保留 `creator_id`，统一扩展访问解析器，避免前端或单个 handler 自行判断 |
| 当前 `organizations` | 已有 `organizations`、`organization_tenant_members`、`kb_shares`，语义是跨租户/跨空间共享 | 不直接改造成组织内部共享空间；继续兼容现有共享关系，产品文案可命名为“跨空间共享” |
| 组织成员按账号管理 | `000081_org_members_by_user` 已将现有组织成员关系收敛到具体账号 | 可以复用其中的账号权限经验，但新组织内部共享空间仍建议独立建模 |
| 存储后端 | 已有 `storage_backends`、租户默认存储、知识库存储绑定和资源目录 | 继续复用；共享、订阅不移动文件，个人升级企业复制内容时才产生新的存储用量 |
| 资源目录 | 已有 `resources/resource_bindings/resource_access_grants`，资源路径和存储后端绑定已经存在 | 迁移只补充空间归属和权限校验，不直接修改物理路径 |
| 知识库创建 | 前端创建弹窗已经隐藏大部分高级配置，但仍会提交完整配置对象；后端也接受完整知识库 JSON | 先在后端增加创建 DTO 白名单和默认配置注入，再逐步移除前端旧字段 |
| 知识库高级配置 | Admin 和知识库设置页仍可配置模型、解析、切片、向量、存储等高级项 | 普通创建流程不再暴露；后台默认配置和已有知识库配置分离管理 |
| “我创建的/共享给我的/我订阅的” | 当前已有个人创建、组织共享、收藏/最近使用等列表能力，但没有完整的订阅模型 | 新增订阅表和聚合查询，保留原有 `/knowledge-bases` 兼容返回 |
| 组织内部共享空间 | 当前没有 `shared_spaces`、空间成员和空间知识库关系表 | 新增独立表和 API，不从现有 `organizations` 数据自动转换 |
| Token/积分 | 当前没有本期所需的产品级积分账户和消费模型 | 本期不新增、不迁移、不改变现有技术 token usage |
| 存储迁移 | 已有本地/对象存储迁移工具和知识库范围迁移能力 | 与产品空间改版分开发布，先做预览和校验，再执行对象迁移 |

当前路由和代码还有两个实施约束：

1. `GET /api/v1/knowledge-bases` 仍然是现有主列表入口，改版不能直接删除或改变其基本返回结构；新三分类可以先通过新增聚合接口实现，再由前端逐步切换。
2. 当前知识库分享链路存在“所有者/管理员校验”和共享权限校验的先后关系，新增或调整共享编辑权限时必须覆盖普通成员、共享空间成员、组织管理员和 API Key 四类身份，避免出现权限已授予但在前置 middleware 被 403 的情况。

## 18. 风险与待确认

| 风险 | 说明 | 建议 |
| --- | --- | --- |
| 新旧共享空间概念混淆 | 当前已有跨空间共享说明，新需求是组织内部共享空间 | 产品文案中明确区分组织空间、内部共享空间、跨空间共享 |
| 权限重复计算 | 前端、handler、middleware 各算一遍容易不一致 | 后端统一 `ResolveKnowledgeBaseAccess` |
| 订阅被误解为授权 | 用户可能认为订阅后永久可访问 | UI 文案说明订阅只是快捷入口 |
| 存量空间类型迁移 | 老空间无法仅凭一列数据准确判断个人/组织，且部分空间可能已有历史跨空间共享 | 先标记为 `legacy` 或保持兼容态；只对判断明确的空间回填，禁止一刀切改成个人或组织 |
| 多组织集团需求扩展 | 后续可能需要跨组织共享 | 第一版不做，未来用跨空间共享或共享空间分组扩展 |
| 升级企业误伤个人隐私 | 自动迁移或共享个人知识库会破坏隐私承诺 | 升级只新增组织空间，个人知识库默认不动 |
| 现有 `organizations` 继续存在 | 新旧共享空间能力并存时产品命名可能混乱 | admin 中把现有能力命名为跨空间共享，内部能力命名为组织内部共享空间 |
| 存储用量口径不一致 | 文件、解析文本、向量索引估算可能不是同一口径 | 第一版沿用当前 `storage_used/storage_size` 口径，后续再细分 |
| 复制知识库成本不可见 | 用户复制个人 KB 到组织时可能意外占用组织配额 | 复制前预估用量并提示目标组织剩余容量 |
| Token/积分延期 | 本期不做产品级余额、计价和额度控制，后续接入范围可能扩大 | 后续单独立项，补充数据模型、计价方案、兼容策略和验收标准 |
| 默认配置变更影响线上知识库 | 后台修改默认模型或解析配置可能改变已有知识库行为 | 默认配置按版本生效，只影响新建知识库；已有知识库必须显式迁移 |
| 创建接口被高级字段绕过 | 旧客户端可能继续提交模型、存储或解析参数 | 后端 DTO 白名单只接受名称和描述，并在 service 层二次过滤 |
| 默认配置不完整导致知识库不可用 | 某类知识库缺少模型、向量或存储默认值 | 发布默认配置前做完整性校验，创建前拒绝不完整配置并提示后台管理员 |

## 19. 开放问题

1. 新用户注册后是否默认创建个人空间？
2. 组织员工创建知识库时，默认创建到个人空间还是当前选中的组织空间？
3. 组织 Contributor 是否默认可以创建组织知识库，还是需要组织开关控制？
4. 共享空间管理员是否必须是组织 Admin，还是可以由组织 Admin 指派普通员工担任？
5. 「我创建的」是否需要按个人知识库和组织知识库再做二级分组？
6. 失效订阅是否显示在列表中并提示无权限，还是直接隐藏？
7. 个人升级企业后，是否允许立即邀请员工，还是先进入组织空间后再邀请？
8. 本路线 V3 采用完整内容复制并重建必要索引；仅复制配置不作为企业升级交付能力。
9. 个人版 SaaS 是否允许配置自定义存储实例？
10. 组织空间的存储配额由系统管理员设置，还是组织 Owner 可自行购买/扩容？
11. 存储用量是否需要拆分为原始文件、解析文本、向量索引、临时文件四类？
12. Token/积分模块后续是否独立立项，个人和组织是否采用统一计价模型？
13. 后台默认配置按全局、组织、套餐还是部署环境分层维护？
14. 默认配置变更后，已有知识库是否允许后台批量迁移？
15. 创建知识库后是否允许后台管理员修改名称和描述之外的高级配置，还是只能重建知识库？

## 20. 基于当前工程的线上影响与迁移方案

### 20.1 总体迁移原则

本次改版必须按照“新增模型、兼容读取、逐步切换、可回滚”的方式实施，不做直接改表语义或批量改归属的迁移。

1. 不修改线上知识库已有的 `tenant_id`、`creator_id`、`vector_store_id` 和物理存储路径。
2. 不把当前 `organizations/kb_shares` 直接转换成新的组织内部共享空间。
3. 不因为个人升级企业而自动移动、复制或共享个人知识库。
4. 不因为后台默认配置变化而批量覆盖已有知识库配置。
5. 新表、新字段先以可空或兼容状态发布，数据校验完成后再打开新行为。
6. 旧 API、旧前端和旧客户端在灰度期间仍然可以访问原有知识库。
7. 所有涉及文件、解析文本、向量索引的复制或迁移都必须有任务记录、进度、校验结果和失败重试。
8. 数据库迁移、产品空间迁移和对象存储迁移拆成独立发布单元，不能在一次上线中同时改变三套数据边界。

### 20.2 当前模型到目标模型的映射

| 当前对象 | 当前实际语义 | 目标处理 | 是否迁移存量数据 |
| --- | --- | --- | --- |
| `tenants` | 工作空间和数据隔离单元 | 继续作为个人空间或组织空间的承载对象，增加兼容性的空间类型标识 | 只回填标识，不改 ID |
| `tenant_members` | 空间成员和 RBAC | 继续作为组织员工和基础角色来源 | 不迁移 |
| `knowledge_bases.tenant_id` | 知识库所属空间 | 继续作为最终归属和配额计费边界 | 不改写 |
| `knowledge_bases.creator_id` | 知识库创建者 | 继续用于个人可见和创建者管理权限 | 不改写；空值保持兼容 |
| `organizations` | 跨租户/跨空间协作容器 | 保留为旧的跨空间共享能力，必要时在产品界面改名 | 不转换 |
| `kb_shares` | 知识库与跨空间共享关系 | 保留，继续提供历史共享访问 | 不删除、不批量撤销 |
| 新增 `shared_spaces` | 组织内部共享空间 | 只关联一个组织空间，成员和知识库必须来自同一组织空间 | 新表，无历史自动导入 |
| 新增 `shared_space_members` | 共享空间内的具体账号 | 只允许加入组织成员 | 新表，无历史自动导入 |
| 新增 `shared_space_knowledge_bases` | 共享空间内的组织知识库 | 只允许加入组织知识库 | 新表，无历史自动导入 |
| 新增 `knowledge_base_subscriptions` | 用户快捷入口 | 只保留用户和知识库引用，不复制内容和权限 | 新表，不从收藏自动推断 |
| `storage_backends/resources` | 存储实例和物理资源 | 继续复用，增加新访问解析规则 | 不搬迁物理文件 |
| 知识库高级配置字段 | 每个知识库当前独立保存的配置 | 标记为历史配置，新增知识库使用后台默认配置 | 不批量覆盖 |

### 20.3 数据库改造建议

当前仓库的版本化迁移已经包含到 `000081`，实际开发时应在发布分支的最新迁移之后创建下一可用版本，并同步维护 SQLite 初始化结构。不能只修改 PostgreSQL 版本化迁移而遗漏 `migrations/sqlite/000000_init.up.sql`。

建议新增以下结构：

```text
tenants
  space_type              nullable: personal | organization | legacy

shared_spaces
  id
  tenant_id               只能指向组织空间
  name
  description
  status
  created_by
  created_at
  updated_at

shared_space_members
  shared_space_id
  user_id
  role                    第一版固定 viewer，保留后续 editor 扩展
  status
  created_at
  updated_at
  unique(shared_space_id, user_id)

shared_space_knowledge_bases
  shared_space_id
  knowledge_base_id
  created_by
  created_at
  unique(shared_space_id, knowledge_base_id)

knowledge_base_subscriptions
  user_id
  knowledge_base_id
  status
  created_at
  updated_at
  unique(user_id, knowledge_base_id)

knowledge_bases
  config_source            nullable: legacy | default
  config_version           nullable
```

字段设计要求：

- `space_type` 在数据库内部保留 `legacy`，但产品接口只向用户暴露“个人空间”和“组织空间”两种正式类型。
- 不新增第二个“知识库所有者 ID”字段，空间所有权继续以 `tenant_id` 为准；组织 Owner 继续以 `tenant_members.role=owner` 为准，避免产生两个权威来源。
- `shared_spaces` 必须通过数据库约束或 service 层校验保证 `shared_space.tenant_id = knowledge_bases.tenant_id`，禁止把个人知识库加入组织内部共享空间。
- `knowledge_base_subscriptions` 不参与授权判断，访问权限必须由统一的 `ResolveKnowledgeBaseAccess` 计算。
- `config_source/config_version` 仅用于追踪知识库创建时使用的默认配置版本，不代表要求重新应用配置。
- 本次不删除历史表、不删除旧字段、不重命名 `organizations` 和 `organization_tenant_members`，为回滚和线上排查保留原始数据。

### 20.4 存量数据预检查

正式迁移前先生成只读盘点报告，报告必须按租户和知识库输出数量，不允许只看总数。

必查指标：

| 检查项 | 目的 |
| --- | --- |
| 租户总数、活跃成员数、Owner 数 | 判断个人、组织和异常空间 |
| `knowledge_bases` 总数、`creator_id` 为空数量 | 识别 API Key 或历史创建数据 |
| 每个租户的知识库数和创建者数 | 判断是否可能为个人空间 |
| 活跃 `kb_shares`、组织成员、共享 Agent 数 | 识别已有跨空间协作关系 |
| 知识库的文档、分块、FAQ、Wiki、标签、数据源数量 | 迁移前后完整性校验 |
| `storage_backend_id`、旧 `storage_provider_config`、`cos_config` 使用情况 | 确认物理存储位置，避免错误绑定新存储 |
| `resources/resource_bindings` 数量、文件大小和物理路径 | 识别资源孤儿和存储迁移范围 |
| 正在解析、导入、重建索引的任务 | 避免在异步任务中途复制或迁移 |
| 当前 API Key、合成用户和真实用户访问情况 | 保持旧客户端和系统任务可用 |

盘点报告必须保存快照时间、数据库版本、应用版本和查询条件，作为上线后的对账基线。

### 20.5 存量空间类型回填

空间类型是本次最容易误伤隐私的字段，不采用“全部老空间默认为组织”的方案。建议增加三种内部状态：

1. `personal`：只有一个活跃真实用户，且该用户是 Owner，没有需要组织协作解释的成员关系；如果存在历史跨空间共享，则标记为 `legacy`，不直接判定为个人。
2. `organization`：存在多个活跃真实成员，或已经存在明确的组织管理员、组织内部协作关系。
3. `legacy`：无法仅凭现有数据判断，或者包含历史共享、API Key-only、多个历史 Owner 等复杂情况。

回填规则：

- 只回填判断明确的空间。
- `legacy` 空间在兼容模式下继续沿用当前列表和权限逻辑。
- 未确认空间不启用“个人知识库禁止新共享”的强约束，避免历史数据被突然拒绝访问。
- 管理员在后台确认空间类型后，才启用对应的新产品规则。
- 不根据知识库名称、描述或当前前端选中状态推断空间类型。

### 20.6 知识库存量迁移

已有知识库不做物理迁移，只做元数据兼容和访问规则扩展。

#### 20.6.1 保持不变的字段

以下字段不得在本次空间改版中批量更新：

- `knowledge_bases.id`
- `knowledge_bases.tenant_id`
- `knowledge_bases.creator_id`
- `knowledge_bases.vector_store_id`
- 已有 `storage_backend_id`
- 文档、分块、FAQ、Wiki、标签、数据源和索引记录
- `resources` 和 `resource_bindings` 的物理路径

#### 20.6.2 历史共享兼容

- 已存在的 `kb_shares` 继续有效，不能因为租户被标记为 `personal` 就自动撤销。
- 新的个人空间知识库禁止新增共享、加入内部共享空间和被他人订阅。
- 对已经存在历史共享关系的个人候选空间，标记为 `legacy_shared` 的兼容状态；访问保留到用户主动撤销或管理员确认迁移策略。
- 新的“共享给我的”列表应同时读取历史 `kb_shares` 和新的 `shared_space_knowledge_bases`，并显示来源类型。

#### 20.6.3 默认配置兼容

- 存量知识库的模型、解析、切片、向量和存储配置全部视为历史配置，不因新增默认配置而改变。
- `config_source` 可回填为 `legacy`，`config_version` 可以为空。
- 新建知识库才使用当前有效的后台默认配置。
- 存量配置迁移必须是后台显式操作，支持单个知识库、指定空间和批量预览三种方式。
- 涉及 `vector_store_id`、存储实例或索引重建的迁移，必须单独创建异步任务并提供失败回滚，不允许在普通保存设置时隐式执行。

### 20.7 个人升级企业的线上迁移

个人升级企业不修改原个人租户，而是创建一个新的组织租户：

1. 校验当前账号拥有个人空间，且不存在同名组织或未完成的升级任务。
2. 创建新的组织 `tenant`，设置 `space_type=organization`。
3. 在 `tenant_members` 写入当前账号的 `owner` 成员关系。
4. 使用现有组织初始化逻辑写入默认存储配额、默认存储实例和基础系统配置。
5. 原个人租户、个人知识库、文件、索引、订阅和历史访问关系保持不变。
6. 页面提示用户选择“仅创建组织空间”或“复制指定知识库”。
7. 复制操作使用独立的异步升级任务，不直接更新原知识库的 `tenant_id`。
8. 复制前计算原始文件、解析内容、索引和预计任务占用，确认目标组织配额足够后才开始。
9. 复制完成后，为目标组织创建新的知识库 ID；原知识库和目标知识库分别管理。
10. 默认不复制原知识库的跨空间分享、共享空间关系、订阅、个人收藏和 Pin 状态，由用户在组织空间中重新设置。

复制失败时只清理本次升级任务创建的目标资源，原个人知识库不回滚、不删除、不改变权限。升级任务需要记录源知识库 ID、目标知识库 ID、资源数量、成功数、失败数和最终校验结果。

### 20.8 存储与线上文件迁移边界

产品空间改版和对象存储迁移必须分开处理。

#### 20.8.1 空间改版不移动文件

- 改变 `space_type` 不改变存储后端。
- 新增共享空间或订阅不复制文件，不增加存储用量。
- 个人知识库复制到组织空间会创建新的资源和索引，新增用量计入组织空间。
- 组织默认存储实例只对新建知识库生效；不能直接把已有知识库的物理路径改指向新实例。
- `storage_backend_id` 为空的历史知识库继续按历史存储配置读取，完成核对后再补充绑定元数据。

#### 20.8.2 本地存储到对象存储

当前仓库已经有知识库范围的存储迁移命令和脚本。线上执行必须遵循：

1. 先重新构建并部署包含最新迁移程序的应用镜像；只修改本地命令文件不会改变线上行为。
2. 先以 `knowledge` 范围执行预览，按租户、知识库和资源路径检查 `planned` 数量。
3. 预览结果确认后再执行迁移；源文件在校验完成前保留，不做删除。
4. 迁移后校验资源数量、文件大小、抽样下载、图片预览、文档解析和知识库问答。
5. 迁移失败时将访问流量切回原路径，保留源文件和迁移记录，禁止直接清理源端文件。

对象存储迁移的成功标准是“新路径可读且旧路径仍可回退”，不是只看迁移命令返回成功。

### 20.9 发布与灰度顺序

建议拆为以下发布批次：

| 批次 | 发布内容 | 默认状态 |
| --- | --- | --- |
| R0 | 盘点脚本、数据快照、迁移前检查和回滚开关 | 开启只读检查 |
| R1 | 新增字段、新表、新索引、审计字段 | 新功能关闭 |
| R2 | 后端统一访问解析器，兼容读取历史共享和新共享空间 | 保持旧列表 |
| R3 | 新增内部共享空间 API、订阅 API、存储概览 API | 仅内部测试租户开启 |
| R4 | 新建知识库仅提交名称和描述，后端默认配置注入 | 仅新建请求开启，旧编辑不变 |
| R5 | 个人/组织空间规则和个人知识库新增共享拦截 | 仅确认过类型的空间开启 |
| R6 | 前端三分类列表、空间切换、企业升级和复制任务 | 小范围灰度 |
| R7 | Admin 批量治理、配置迁移和存量空间确认 | 管理员按需执行 |
| R8 | 观察期结束后关闭旧 UI 入口，保留旧 API 和数据库兼容字段 | 仅在确认无回滚需求后执行 |

建议使用以下开关，名称可以按项目现有配置体系调整：

```text
knowledge_space_v2_enabled
knowledge_space_v2_read_merge_enabled
knowledge_base_simple_create_enabled
knowledge_base_default_config_enabled
personal_knowledge_base_share_block_enabled
knowledge_base_enterprise_copy_enabled
```

灰度开关必须支持按租户、按用户和按请求来源配置，不能只提供全局开关。

### 20.10 线上影响矩阵

| 业务动作 | 对存量知识库的影响 | 处理方式 |
| --- | --- | --- |
| 打开知识库首页 | 可能新增订阅和共享空间聚合查询 | 保留旧列表接口，先双读对账，再切换前端 |
| 查看/搜索/问答 | 访问判断增加共享空间和订阅关系解析 | 订阅只作为入口，最终仍校验实际访问权 |
| 上传、删除、解析 | 受新的统一权限解析器影响 | 重点验证创建者、组织 Admin、共享 viewer/editor、API Key |
| 创建知识库 | 新请求不再允许用户选择高级配置 | 后端兼容接收旧字段，但忽略或审计，不让旧客户端绕过默认配置 |
| 编辑存量知识库 | 不应被新默认配置覆盖 | 保留现有配置编辑权限，按权限逐步收敛到后台治理 |
| 加入内部共享空间 | 只允许组织知识库 | 校验知识库租户和共享空间租户一致 |
| 个人升级企业 | 新增组织空间和可选复制任务 | 原个人空间不变，复制为新知识库 |
| 订阅/取消订阅 | 只影响个人列表 | 不新增文件、索引和存储用量 |
| 修改存储默认实例 | 只影响后续新建知识库 | 已有知识库继续使用原绑定 |
| 对象存储迁移 | 影响文件读取和资源访问 | 单独执行、预览、校验和回退 |

### 20.11 验证、对账和回滚

#### 20.11.1 数据对账

上线后至少执行以下对账：

- 租户数、知识库数、文档数、分块数、FAQ 数、Wiki 数、标签数、数据源数前后一致。
- 每个知识库的 `tenant_id`、`creator_id`、`vector_store_id` 和存储绑定未被意外改变。
- 历史 `kb_shares` 数量和活跃状态不减少，除非有明确的用户撤销记录。
- 新共享空间成员和知识库关系不存在跨组织、跨租户绑定。
- 订阅记录中的用户和知识库均存在，失权订阅不会被误当作授权。
- `storage_used`、资源数量和文件大小在复制或对象存储迁移后可以解释。

#### 20.11.2 权限矩阵

至少验证：

- 个人空间 Owner；
- 组织 Owner、Admin、Contributor、Viewer；
- 共享空间成员和被移除成员；
- 组织知识库创建者；
- 组织管理员访问其他员工创建的知识库；
- 历史跨空间共享 viewer/editor；
- API Key 和系统合成用户；
- 个人知识库新增共享、加入共享空间、被订阅的拒绝结果。

#### 20.11.3 回滚策略

- 数据库：只关闭新功能开关，保留新增表和字段，不执行破坏性 down migration。
- 前端：回退到原知识库列表和原创建弹窗，旧接口继续工作。
- 权限：关闭新共享空间解析，恢复历史 `kb_shares` 和当前租户 RBAC 逻辑。
- 企业升级：删除未完成的目标组织和本次任务产生的目标资源，原个人空间不做任何修改。
- 配置：关闭默认配置注入不影响已有知识库；新建知识库回到兼容默认值前必须确认后台配置完整。
- 存储：源文件和旧路径保留，迁移失败时切回旧路径；在校验窗口结束前不得清理源端。

### 20.12 本次改版完成标准

本次改版只有同时满足以下条件，才可以认为线上迁移完成：

1. 新旧列表、访问权限和存储用量完成至少一个完整观察周期的对账。
2. 已确认的个人空间新建知识库无法被共享、加入内部共享空间或被他人订阅。
3. 组织内部共享空间只能访问同一组织的组织知识库。
4. 历史跨空间共享关系没有因为新模型上线而意外失效。
5. 个人升级企业不会修改原个人空间和个人知识库。
6. 复制知识库的目标资源、索引、配额和失败重试可追踪。
7. 新建知识库只需名称和描述，后台默认配置完整生效；已有知识库配置保持不变。
8. 存储迁移具备预览、执行、校验和回退路径，未验证的源文件没有被删除。
9. Token/积分相关能力没有被错误带入本期数据库、接口、页面和权限判断。
