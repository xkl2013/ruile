# 阿里云 ACR + ECS 部署指南

本文档说明如何把本项目构建成 Docker 镜像，推送到阿里云容器镜像服务 ACR，然后在阿里云 ECS 上通过拉取仓库代码和 ACR 镜像启动服务。

适用场景：

- 本地或 CI 负责构建镜像，ECS 只负责拉取镜像和运行。
- 需要固定镜像版本，避免 `latest` 导致版本不可追踪。
- 需要后续版本迭代时可升级、可回滚。

## 1. 镜像约定

当前 ACR 配置：

```text
ACR Registry: registry.cn-beijing.aliyuncs.com
ACR Namespace: rl-knowledge
```

业务镜像名称：

```text
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-app:<version>
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-docreader:<version>
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-ui:<version>
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:<version>
```

建议 `<version>` 使用 git commit 短 SHA、版本号或日期版本，例如：

```text
bfce07e7
v1.0.0
20260804-001
```

不要在生产部署中长期使用 `latest`。固定 tag 可以保证升级和回滚可控。

## 快速方式：纯镜像一键部署

如果 ECS 只需要直接拉取镜像运行，不需要拉仓库代码，可以使用本地脚本：

```text
scripts/deploy_aliyun_acr_image.local.sh
```

该脚本会在 ECS 上自动完成：

- 安装 Docker 和 Docker Compose Plugin。
- 创建部署目录 `/opt/weknora-image-deploy`。
- 生成 `.env`。
- 生成独立 `docker-compose.yml`。
- 登录阿里云 ACR。
- 从 ACR 拉取镜像并启动服务。

当前脚本使用的镜像全部来自 ACR：

```text
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-app:bfce07e7
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-docreader:bfce07e7
registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-ui:bfce07e7
registry.cn-beijing.aliyuncs.com/rl-knowledge/paradedb:v0.22.2-pg17
registry.cn-beijing.aliyuncs.com/rl-knowledge/redis:7.0-alpine
```

从本地把脚本传到 ECS 并执行：

```bash
scp scripts/deploy_aliyun_acr_image.local.sh root@<ECS_PUBLIC_IP>:/tmp/weknora-deploy.sh
ssh root@<ECS_PUBLIC_IP> 'bash /tmp/weknora-deploy.sh'
```

如果要部署新版本，传入新的镜像 tag：

```bash
scp scripts/deploy_aliyun_acr_image.local.sh root@<ECS_PUBLIC_IP>:/tmp/weknora-deploy.sh
ssh root@<ECS_PUBLIC_IP> 'WEKNORA_VERSION=<new-version> bash /tmp/weknora-deploy.sh'
```

部署完成后访问：

```text
http://<ECS_PUBLIC_IP>
```

在 ECS 上查看状态：

```bash
cd /opt/weknora-image-deploy
docker compose ps
docker compose logs -f app
```

注意：

- `scripts/deploy_aliyun_acr_image.local.sh` 内含 ACR 账号密码，已被 `.gitignore` 忽略，不应提交到远程仓库。
- 纯镜像部署默认关闭 Agent Skills sandbox，因为当前 app 镜像内没有 Docker CLI。核心知识库、前端、后端、文档解析、Postgres、Redis 不受影响。
- 如果后续需要启用 Agent Skills sandbox，应重新构建 app 镜像并加入 Docker CLI，或改为宿主机侧安全执行方案。

## 2. 本地构建并推送到 ACR

在本地项目根目录执行。

### 2.1 配置 ACR 登录信息

创建 `.acr.env`：

```bash
cat > .acr.env <<'EOF'
ACR_REGISTRY=registry.cn-beijing.aliyuncs.com
ACR_USERNAME=<ACR_USERNAME>
ACR_PASSWORD=<ACR_PASSWORD>
ACR_NAMESPACE=rl-knowledge

# ECS 通常是 x86_64 架构，建议构建 linux/amd64 镜像。
ACR_BUILD_PLATFORM=linux/amd64

# 如果在 Apple Silicon Mac 上构建，建议保留 linux/arm64 作为构建器平台。
ACR_BUILDER_PLATFORM=linux/arm64

# 构建时先把基础镜像同步到 ACR，减少 Docker Hub 网络问题。
USE_ACR_BASE_IMAGES=true
EOF

chmod 600 .acr.env
```

注意：

- `.acr.env` 包含密码，只应保存在本地或 CI Secret 中。
- 不要把 `.acr.env` 提交到 Git 仓库。

### 2.2 构建并推送

使用当前 git commit 短 SHA 作为镜像 tag：

```bash
./scripts/build_and_push_acr.sh
```

指定版本 tag：

```bash
./scripts/build_and_push_acr.sh --tag 20260804-001
```

如果镜像已经本地构建完成，只需要重新打 tag 并推送：

```bash
./scripts/build_and_push_acr.sh --skip-build --tag 20260804-001
```

脚本完成后会输出：

```text
Pushed tag: <version>

Use this on ECS:
  WEKNORA_VERSION=<version>

Image prefix:
  registry.cn-beijing.aliyuncs.com/rl-knowledge
```

记录这个 `<version>`，ECS 部署和后续升级都使用它。

## 3. ECS 安装 Docker

以下命令以 Ubuntu Server 为例。SSH 登录 ECS 后执行：

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg git openssl

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg \
  | sudo gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(. /etc/os-release && echo "$VERSION_CODENAME") stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker

docker --version
docker compose version
```

如果当前用户没有 Docker 权限，可以继续使用 `sudo docker ...`，或者把用户加入 `docker` 组后重新登录：

```bash
sudo usermod -aG docker "$USER"
```

## 4. ECS 拉取仓库代码

在 ECS 上执行：

```bash
sudo mkdir -p /opt
sudo git clone <GIT_REPO_URL> /opt/WeKnora
sudo chown -R "$USER":"$USER" /opt/WeKnora
cd /opt/WeKnora
```

如果要部署指定代码版本：

```bash
git fetch --tags
git checkout <GIT_REF>
```

要求：ECS 上的仓库代码应与本次镜像构建所用代码一致，至少 `docker-compose.yml`、配置文件和数据库迁移文件要匹配。

## 5. 配置运行环境

创建 `.env`：

```bash
cd /opt/WeKnora
cp .env.example .env
chmod 600 .env
```

写入基础环境变量：

```bash
set_env() {
  key="$1"
  val="$2"
  if grep -qE "^${key}=" .env; then
    sed -i "s|^${key}=.*|${key}=${val}|" .env
  else
    printf '%s=%s\n' "$key" "$val" >> .env
  fi
}

genpw() { LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24; }
gen32() { openssl rand -hex 16; }

set_env WEKNORA_VERSION "<version>"
set_env GIN_MODE "release"
set_env DB_USER "weknora"
set_env DB_NAME "weknora"
set_env DB_PASSWORD "$(genpw)"
set_env REDIS_PASSWORD "$(genpw)"
set_env JWT_SECRET "$(genpw)$(genpw)"
set_env SYSTEM_AES_KEY "$(gen32)"
set_env TENANT_AES_KEY "$(gen32)"
set_env DB_DRIVER "postgres"
set_env RETRIEVE_DRIVER "postgres"
set_env STORAGE_TYPE "local"
set_env STREAM_MANAGER_TYPE "redis"
set_env WEKNORA_LANGUAGE "zh-CN"

# 默认不把 app 的 8080 直接暴露到公网，前端容器会在 Docker 网络内访问 app。
set_env APP_PORT "127.0.0.1:8080"

# Agent Skills 使用的沙箱镜像。
set_env WEKNORA_SANDBOX_DOCKER_IMAGE "registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:<version>"
```

把 `<version>` 替换为第 2 步推送出来的 tag，例如：

```bash
set_env WEKNORA_VERSION "bfce07e7"
set_env WEKNORA_SANDBOX_DOCKER_IMAGE "registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:bfce07e7"
```

生产注意：

- `SYSTEM_AES_KEY` 和 `TENANT_AES_KEY` 首次上线后不要随意修改，否则已加密的模型 Key、凭据等可能无法解密。
- `DB_PASSWORD`、`REDIS_PASSWORD` 首次启动后不要随意修改，除非同步修改对应持久化服务。

## 6. 创建 ACR Compose 覆盖文件

默认 `docker-compose.yml` 使用 `wechatopenai/...` 镜像名。ECS 使用 ACR 镜像时，创建一个覆盖文件：

```bash
cat > docker-compose.acr.yml <<'YAML'
services:
  frontend:
    image: registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-ui:${WEKNORA_VERSION}
  app:
    image: registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-app:${WEKNORA_VERSION}
  docreader:
    image: registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-docreader:${WEKNORA_VERSION}
  sandbox:
    image: registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:${WEKNORA_VERSION}
YAML
```

## 7. 登录 ACR 并启动

登录 ACR：

```bash
read -rsp "ACR Password: " ACR_PASSWORD; echo
printf '%s' "$ACR_PASSWORD" | docker login \
  --username='<ACR_USERNAME>' \
  --password-stdin \
  registry.cn-beijing.aliyuncs.com
```

拉取镜像：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml pull
```

启动服务：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml up -d --no-build
```

`--no-build` 很重要：ECS 只运行已推送到 ACR 的镜像，不在云服务器上重新构建。

检查状态：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml ps
docker compose -f docker-compose.yml -f docker-compose.acr.yml logs --tail=100 app
curl -f http://127.0.0.1:8080/health
```

浏览器访问：

```text
http://<ECS_PUBLIC_IP>
```

## 8. 版本升级

每次版本迭代按以下流程执行。

### 8.1 本地或 CI 构建新版本

```bash
git pull
./scripts/build_and_push_acr.sh --tag <new-version>
```

### 8.2 ECS 切换代码和镜像版本

```bash
cd /opt/WeKnora
git fetch --all --tags
git checkout <new-git-ref>

set_env WEKNORA_VERSION "<new-version>"
set_env WEKNORA_SANDBOX_DOCKER_IMAGE "registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:<new-version>"

docker compose -f docker-compose.yml -f docker-compose.acr.yml pull
docker compose -f docker-compose.yml -f docker-compose.acr.yml up -d --no-build
```

应用启动时会自动执行数据库迁移。升级前建议先备份数据库和文件 volume。

## 9. 版本回滚

如果新版本启动失败，可以切回上一个镜像 tag：

```bash
cd /opt/WeKnora
git checkout <previous-git-ref>

set_env WEKNORA_VERSION "<previous-version>"
set_env WEKNORA_SANDBOX_DOCKER_IMAGE "registry.cn-beijing.aliyuncs.com/rl-knowledge/weknora-sandbox:<previous-version>"

docker compose -f docker-compose.yml -f docker-compose.acr.yml pull
docker compose -f docker-compose.yml -f docker-compose.acr.yml up -d --no-build
```

如果新版本已经执行了不可逆数据库迁移，仅回滚镜像可能不够，需要结合升级前的数据库备份恢复。

## 10. 常用运维命令

查看容器：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml ps
```

查看日志：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml logs -f app
docker compose -f docker-compose.yml -f docker-compose.acr.yml logs -f frontend
docker compose -f docker-compose.yml -f docker-compose.acr.yml logs -f docreader
```

重启服务：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml restart app frontend docreader
```

停止服务：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml down
```

不要在生产环境随意执行：

```bash
docker compose -f docker-compose.yml -f docker-compose.acr.yml down -v
```

`down -v` 会删除数据库和文件 volume。

## 11. 安全组建议

ECS 安全组入方向建议只开放：

| 端口 | 来源 | 用途 |
| --- | --- | --- |
| `22/tcp` | 运维固定 IP | SSH |
| `80/tcp` | `0.0.0.0/0` | HTTP |
| `443/tcp` | `0.0.0.0/0` | HTTPS |

不要对公网开放：

```text
5432 PostgreSQL
6379 Redis
8080 App API
50051 docreader gRPC
```

需要排障时优先使用 SSH 隧道或只绑定 `127.0.0.1`。

## 12. 成本和版本影响

本地或 CI 构建镜像不会产生 ECS 构建成本。ECS 侧主要成本来自：

- ECS 实例计算资源。
- 云盘存储。
- ACR 镜像存储。
- ECS 拉取镜像和公网访问产生的网络流量。

版本迭代不会影响旧版本镜像，只要使用不同 tag 推送即可。ECS 通过 `WEKNORA_VERSION` 选择运行哪个版本：

```bash
set_env WEKNORA_VERSION "bfce07e7"
docker compose -f docker-compose.yml -f docker-compose.acr.yml up -d --no-build
```

如果复用同一个 tag 推送新镜像，ECS 可能因为本地镜像缓存而继续运行旧镜像。因此生产环境建议每次发布使用新的不可变 tag。
