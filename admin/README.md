# 睿乐 Admin 工程

Admin 工程面向软件开发者、实施/运维和平台运营者，承接后台设置、空间治理、模型运行时、数据扩展、发布渠道、系统管理等管理类页面。主工程继续承载知识库使用、聊天、整理、服务工作台和个人日常操作。

## 技术栈

与主前端保持一致：

- Vue 3
- Vite
- TypeScript
- Pinia
- Vue Router
- Vue I18n
- TDesign Vue Next

## 本地开发

```bash
make dev-admin
```

或：

```bash
npm --prefix admin run dev
```

默认地址：

```text
http://localhost:8082/admin/
```

默认 API 代理目标：

```text
http://localhost:8080
```

可通过 `VITE_DEV_PROXY_TARGET` 或 `FRONTEND_BACKEND_URL` 覆盖。

## 验证

```bash
npm --prefix admin run type-check
npm --prefix admin run build
```

## 复用边界

第一版 admin 是独立工程，但通过 Vite alias 复用主前端源码：

- `@` 指向 `../frontend/src`
- `@admin` 指向 `./src`

这样可以先复用现有 API、auth store、i18n、TDesign 样式和 settings 组件，避免一次性搬迁所有模块。后续稳定后再把共享能力抽到 `packages/web-shared`。

Admin 不是园长或园所成员的日常产品入口。个人版、园长版、园所版和集团版是平台运营者在 admin 中识别、配置和治理的客户空间形态。

## 已落位模块

- 工作区：空间信息、成员管理、聊天历史、版本能力
- 资产：知识库占位、智能体、服务配置、共享空间
- 发布：IM 渠道、嵌入渠道、API Key
- 模型与数据：模型、Ollama、睿乐大脑云、向量库、解析引擎、存储后端、网络搜索、MCP 服务
- 系统：全局设置、运行队列
