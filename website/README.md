# 睿乐大脑官网 v2

面向早幼教领域的睿乐大脑产品官网。多页站点，贴近产品 TDesign 视觉（白底 + 品牌绿 `#07c05f` + logo 金点缀）。

## 页面结构

| 文件 | 内容 |
|------|------|
| `index.html` | 首页：Hero → WHY → 四大产品模块 → 8 大经营场景 → 一线采集与运营纠偏 → 角色赋能 → 落地流程 → 安全 → FAQ → CTA |
| `product.html` | 产品模块详解：知识库 / 整理与发现 / 服务空间 / 系列课程 / 智能体与多端触达 |
| `scenes.html` | 8 大园所经营场景（痛点-做法-产出）+ 一线采集与行为矫正 |
| `pricing.html` | 版本定价：个人版 / 企业版·团队 / 企业版·商业 + 三阶段演进 |
| `security.html` | 安全与部署：三种部署方式、六层安全控制、落地检查清单 |
| `privacy.html` / `terms.html` | 法务页（沿用共享样式） |
| `404.html` | 404 页（nginx error_page 引用） |
| `sitemap.xml` / `robots.txt` | SEO 基础设施 |

叙事为**双主线融合**：上半场讲园所经营场景与产品模块，下半场保留一线数据采集与运营行为矫正作为差异化亮点。

设计 token 在 `styles.css` `:root`（品牌绿取自 `frontend/src/assets/theme/theme.css` 的 `--td-brand-color-4`）。

## 预约演示表单

全站 19 处「预约演示」入口由 `main.js` 顶部的 `DEMO_FORM_URL` 常量统一管理：

- **留空**（当前状态）：点击后跳转首页 `#contact` 区块
- **填入飞书表单链接**（形如 `https://xxx.feishu.cn/share/base/form/...`）：全站「预约演示」自动改为新开表单页，无需逐页修改

## 本地运行

纯静态站，在 `website` 目录任选其一：

```bash
# 方式一：vite（frontend 依赖已装）
../frontend/node_modules/.bin/vite --host 127.0.0.1

# 方式二：python
python3 -m http.server 5180 --bind 127.0.0.1
```

## 构建与部署（对齐 admin 工程模式）

与 admin 相同的三段式：宿主机暂存产物 → nginx 镜像 → Makefile 串联。

```bash
# 1. 仅暂存静态产物到 website/dist（无打包器，纯拷贝 + 校验文件齐全）
./scripts/build_website_dist.sh        # 或 make build-website-dist

# 2. 构建生产镜像（内部会先执行步骤 1）
make docker-build-website              # 产物镜像 ruile-website:latest

# 3. 运行验证
docker run --rm -p 5181:80 ruile-website:latest
```

组成文件（对齐 `admin/`）：

| 文件 | 作用 |
|------|------|
| `website/Dockerfile` | `nginx:stable-alpine` + COPY dist（无需 entrypoint，无运行时配置） |
| `website/nginx.conf` | gzip、`/assets/` 一年 immutable、HTML no-cache、安全响应头、404 页 |
| `scripts/build_website_dist.sh` | 发布文件清单拷贝到 `website/dist`，缺文件即失败 |

`dist/`、`mock/` 为本地构建产物/设计源文件，均不入镜像发布清单。

