# 阿里云 ECS 部署指南

本文档说明如何在阿里云 ECS 上部署睿乐大脑标准版。默认方案为单台 ECS + Docker Compose，适合个人、团队内网或中小规模私有化部署；如果需要多副本、高可用、统一 Ingress 和云盘编排，请参考文末 ACK/Helm 方案。

## 1. 部署架构

默认 `docker-compose.yml` 会启动 5 个核心常驻容器：

| 容器 | 作用 | 默认宿主机暴露 |
| --- | --- | --- |
| `WeKnora-frontend` | Vue UI + NGINX，反向代理 `/api` 和 `/files` | `80` |
| `WeKnora-app` | Go/Gin 后端 API、任务队列、模型调用 | `8080`，建议只绑定 `127.0.0.1` |
| `WeKnora-docreader` | Python 文档解析 gRPC 服务 | 不暴露到宿主机 |
| `WeKnora-postgres` | ParadeDB/PostgreSQL，含 pgvector/BM25 相关扩展 | 不暴露到宿主机 |
| `WeKnora-redis` | 流式输出、缓存、异步队列 | 不暴露到宿主机 |

可选能力通过 Compose profile 启用：

| Profile | 用途 | 命令 |
| --- | --- | --- |
| `neo4j` | 知识图谱 / GraphRAG | `docker compose --profile neo4j up -d` |
| `minio` | 自建对象存储 | `docker compose --profile minio up -d` |
| `langfuse` | 自建链路追踪 | `docker compose --profile langfuse up -d` |
| `searxng` | 自建搜索引擎 | `docker compose --profile searxng up -d` |
| `full` | 启动多数可选组件 | `docker compose --profile full up -d` |

## 2. ECS 规格建议

| 场景 | CPU / 内存 | 系统盘 | 说明 |
| --- | --- | --- | --- |
| 试用 / 小团队 | 4 核 8 GiB | 100 GiB ESSD | 默认 5 容器，文档量较少 |
| 生产推荐 | 4 核 16 GiB | 200 GiB ESSD | 文档解析、Embedding 并发更稳 |
| 全功能 / Langfuse / Neo4j | 8 核 32 GiB | 300 GiB+ ESSD | 同机运行图谱、可观测、大量索引 |

推荐镜像：Ubuntu Server 22.04 LTS / 24.04 LTS。下方安装命令按 Ubuntu 编写；使用 Alibaba Cloud Linux 时，请按系统对应包管理器安装 Docker 与 Docker Compose Plugin。

## 3. 阿里云前置配置

1. 创建 ECS，建议选择和用户、OSS、RDS 位于同一地域，降低访问延迟。
2. 安全组入方向只放行必要端口：

| 端口 | 来源 | 用途 |
| --- | --- | --- |
| `22/tcp` | 你的固定办公 IP | SSH 管理 |
| `80/tcp` | `0.0.0.0/0` | HTTP / 证书签发 |
| `443/tcp` | `0.0.0.0/0` | HTTPS |

不要对公网放行 `5432`、`6379`、`8080`、`9000`、`7474`、`7687` 等基础设施端口。需要排障时用 SSH 隧道或只绑定 `127.0.0.1`。

3. 如果使用中国大陆地域 ECS + 自有域名对公网服务，域名通常需要完成 ICP 备案。未备案前可先用公网 IP 测试，或使用非中国大陆地域。
4. 如果后续使用 HTTPS，先把域名 A 记录解析到 ECS 公网 IP。

## 4. 安装 Docker

SSH 到 ECS 后执行：

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl gnupg lsb-release git openssl

sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg \
  | sudo gpg --dearmor --yes -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://mirrors.aliyun.com/docker-ce/linux/ubuntu $(lsb_release -cs) stable" \
  | sudo tee /etc/apt/sources.list.d/docker.list >/dev/null

sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
sudo systemctl enable --now docker

docker --version
docker compose version
```

如果拉取 Docker Hub 镜像很慢，优先在阿里云容器镜像服务 ACR 中配置自有镜像加速或把所需镜像同步到自己的 ACR 命名空间，再调整 Compose 镜像地址。

## 5. 上传或拉取代码

### 方式 A：部署公开仓库版本

```bash
sudo mkdir -p /opt
sudo git clone https://github.com/Tencent/WeKnora.git /opt/WeKnora
sudo chown -R "$USER":"$USER" /opt/WeKnora
cd /opt/WeKnora

# 推荐固定版本，避免 latest/main 带来不可复现升级。
git checkout v0.7.0
```

### 方式 B：部署当前本地修改

在本地项目根目录执行：

```bash
rsync -az --delete \
  --exclude='.git' \
  --exclude='.env' \
  --exclude='.local-data' \
  --exclude='node_modules' \
  --exclude='frontend/node_modules' \
  ./ root@<ECS_PUBLIC_IP>:/opt/WeKnora/
```

然后在 ECS 上：

```bash
cd /opt/WeKnora
```

## 6. 配置 `.env`

```bash
cd /opt/WeKnora
cp .env.example .env
chmod 600 .env
```

生成并写入生产密钥：

```bash
genpw() { LC_ALL=C tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24; }
gen32() { openssl rand -hex 16; } # 32 个 ASCII 字符，满足 SYSTEM_AES_KEY 长度要求
set_env() {
  key="$1"
  val="$2"
  if grep -qE "^${key}=" .env; then
    sed -i "s|^${key}=.*|${key}=${val}|" .env
  else
    printf '%s=%s\n' "$key" "$val" >> .env
  fi
}

set_env WEKNORA_VERSION "v0.7.0"
set_env GIN_MODE "release"
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

# 默认不把 app 的 8080 暴露到公网；前端容器在 Docker 网络内访问 app。
set_env APP_PORT "127.0.0.1:8080"
```

注意：

- `SYSTEM_AES_KEY` 必须保持 32 字节。上线后不要随意更换，否则已加密的模型 Key、MCP 凭据、数据源凭据等无法解密。
- `DB_PASSWORD`、`REDIS_PASSWORD` 首次启动后不要改，除非同步修改对应持久化服务。
- `.env.example` 中的默认密钥只适合本地试用，生产必须替换。

### 首位管理员与注册策略

最简单流程：

1. 首次上线前允许注册：

```bash
set_env DISABLE_REGISTRATION "false"
```

2. 启动服务后访问 Web UI，注册第一个账号并创建空间。
3. 注册完成后关闭公开注册：

```bash
set_env DISABLE_REGISTRATION "true"
docker compose up -d app frontend
```

如果需要把某个已注册用户提升为系统管理员，可设置：

```bash
set_env WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL "admin@example.com"
docker compose up -d app
```

该变量只在“当前没有系统管理员”时生效；它不会自动创建用户。

## 7. 可选：使用阿里云 DashScope 作为默认模型

如果希望所有空间开箱即用，可通过内置模型 YAML 声明默认 LLM、Embedding、Rerank。

写入 `.env`：

```bash
set_env LLM_MODEL_NAME "qwen-plus"
set_env LLM_BASE_URL "https://dashscope.aliyuncs.com/compatible-mode/v1"
set_env LLM_API_KEY "sk-your-dashscope-api-key"
set_env LLM_PROVIDER "aliyun"

set_env EMBEDDING_MODEL_NAME "text-embedding-v3"
set_env EMBEDDING_BASE_URL "https://dashscope.aliyuncs.com/compatible-mode/v1"
set_env EMBEDDING_API_KEY "sk-your-dashscope-api-key"
set_env EMBEDDING_PROVIDER "aliyun"

set_env RERANK_MODEL_NAME "gte-rerank"
set_env RERANK_BASE_URL "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
set_env RERANK_API_KEY "sk-your-dashscope-api-key"
set_env RERANK_PROVIDER "aliyun"

set_env ASR_MODEL_NAME "qwen3-asr-flash"
set_env ASR_BASE_URL "https://dashscope.aliyuncs.com/compatible-mode/v1"
set_env ASR_API_KEY "sk-your-dashscope-api-key"
set_env ASR_PROVIDER "aliyun"
set_env ASR_LANGUAGE "zh"
set_env ASR_ENABLE_ITN "false"
```

创建 `config/builtin_models.yaml`：

```yaml
builtin_models:
  - id: builtin-aliyun-chat
    type: KnowledgeQA
    source: remote
    is_default: true
    name: ${LLM_MODEL_NAME}
    parameters:
      base_url: ${LLM_BASE_URL}
      api_key: ${LLM_API_KEY}
      provider: ${LLM_PROVIDER}

  - id: builtin-aliyun-embedding
    type: Embedding
    source: remote
    is_default: true
    name: ${EMBEDDING_MODEL_NAME}
    parameters:
      base_url: ${EMBEDDING_BASE_URL}
      api_key: ${EMBEDDING_API_KEY}
      provider: ${EMBEDDING_PROVIDER}
      embedding_parameters:
        dimension: 1024
        truncate_prompt_tokens: 0

  - id: builtin-aliyun-rerank
    type: Rerank
    source: remote
    name: ${RERANK_MODEL_NAME}
    parameters:
      base_url: ${RERANK_BASE_URL}
      api_key: ${RERANK_API_KEY}
      provider: ${RERANK_PROVIDER}

  - id: builtin-aliyun-asr
    type: ASR
    source: remote
    is_default: true
    name: ${ASR_MODEL_NAME}
    parameters:
      base_url: ${ASR_BASE_URL}
      api_key: ${ASR_API_KEY}
      provider: ${ASR_PROVIDER}
      extra_config:
        language: ${ASR_LANGUAGE}
        enable_itn: ${ASR_ENABLE_ITN}
```

创建 `docker-compose.override.yml`，让 app 容器挂载该配置：

```yaml
services:
  app:
    volumes:
      - ./config/builtin_models.yaml:/app/config/builtin_models.yaml:ro
```

重启 app 后验证：

```bash
docker compose up -d app
docker compose logs app | grep -E 'Built-in model|Built-in models'
```

也可以不预置 YAML，在 Web UI 的“设置 -> 模型管理”里手动添加模型。

## 8. 可选：使用阿里云 OSS 存储文件

默认 `STORAGE_TYPE=local` 会把上传文件保存到 Docker volume `data-files`。如果希望使用 OSS：

```bash
set_env STORAGE_TYPE "oss"
set_env OSS_ENDPOINT "https://oss-cn-hangzhou.aliyuncs.com"
set_env OSS_REGION "cn-hangzhou"
set_env OSS_ACCESS_KEY "your-access-key-id"
set_env OSS_SECRET_KEY "your-access-key-secret"
set_env OSS_BUCKET_NAME "your-bucket-name"
set_env OSS_PATH_PREFIX "weknora/"

docker compose up -d app
```

建议：

- Bucket 与 ECS 使用同一地域。
- AccessKey 使用最小权限 RAM 用户，不要使用主账号 AK。
- 面向飞书、企微、Slack 等 IM 渠道时，文件链接必须从公网可达；使用 OSS 比本地存储更适合这类场景。

## 9. 启动服务

使用官方预构建镜像：

```bash
cd /opt/WeKnora
docker compose pull
docker compose up -d --no-build
```

如果部署的是当前源码改动，需要自行构建镜像：

```bash
cd /opt/WeKnora
./scripts/build_frontend_dist.sh
docker compose build frontend app docreader sandbox
docker compose up -d
```

检查状态：

```bash
docker compose ps
curl -f http://127.0.0.1:8080/health
docker compose logs --tail=100 app
```

如果 `frontend` 的 `FRONTEND_PORT=80`，浏览器访问：

```text
http://<ECS_PUBLIC_IP>
```

## 10. 配置域名与 HTTPS

如果需要 HTTPS，建议由宿主机 NGINX 处理 TLS，Compose 前端只监听本机端口。

修改 `.env`：

```bash
set_env FRONTEND_PORT "127.0.0.1:18080"
set_env APP_EXTERNAL_URL "https://kb.example.com"
docker compose up -d frontend app
```

安装 NGINX 和 Certbot：

```bash
sudo apt-get install -y nginx certbot python3-certbot-nginx
```

创建 `/etc/nginx/sites-available/weknora.conf`：

```nginx
server {
    listen 80;
    server_name kb.example.com;
    client_max_body_size 100m;

    location / {
        proxy_pass http://127.0.0.1:18080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_buffering off;
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

启用站点并签发证书：

```bash
sudo ln -sf /etc/nginx/sites-available/weknora.conf /etc/nginx/sites-enabled/weknora.conf
sudo nginx -t
sudo systemctl reload nginx
sudo certbot --nginx -d kb.example.com
```

证书签发成功后访问：

```text
https://kb.example.com
```

## 11. 运维命令

查看容器：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f app
docker compose logs -f frontend
docker compose logs -f docreader
```

重启：

```bash
docker compose restart app frontend
```

停止：

```bash
docker compose down
```

不要在生产环境随意执行 `docker compose down -v`，它会删除数据库和文件 volume。

## 12. 备份与恢复

创建备份目录：

```bash
cd /opt/WeKnora
mkdir -p backups
```

备份 PostgreSQL：

```bash
docker exec -t WeKnora-postgres sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  | gzip > "backups/postgres-$(date +%F).sql.gz"
```

备份本地文件 volume：

```bash
DATA_VOLUME=$(docker inspect WeKnora-app --format '{{range .Mounts}}{{if eq .Destination "/data/files"}}{{.Name}}{{end}}{{end}}')
docker run --rm \
  -v "${DATA_VOLUME}:/data:ro" \
  -v "$PWD/backups:/backup" \
  alpine tar czf "/backup/data-files-$(date +%F).tgz" -C /data .
```

恢复 PostgreSQL 前先停应用容器：

```bash
docker compose stop app frontend docreader
gunzip -c backups/postgres-YYYY-MM-DD.sql.gz \
  | docker exec -i WeKnora-postgres sh -c 'psql -U "$POSTGRES_USER" "$POSTGRES_DB"'
docker compose up -d
```

如果使用 OSS，文件备份应优先通过 OSS 版本控制、生命周期、跨区域复制或定期 `ossutil` 同步完成。

## 13. 升级

升级前先备份 `.env`、数据库和文件 volume：

```bash
cd /opt/WeKnora
cp .env "backups/env-$(date +%F)"
docker exec -t WeKnora-postgres sh -c 'pg_dump -U "$POSTGRES_USER" "$POSTGRES_DB"' \
  | gzip > "backups/postgres-before-upgrade-$(date +%F).sql.gz"
```

切换版本并拉取镜像：

```bash
set_env WEKNORA_VERSION "v0.7.0"
git fetch --tags
git checkout v0.7.0
docker compose pull
docker compose up -d --no-build
```

应用启动时会自动执行数据库迁移。迁移失败时 UI 通常仍可访问，请先看日志：

```bash
docker compose logs --tail=300 app | grep -i -E 'migration|error|failed'
```

迁移排障参考：[`docs/migration-troubleshooting.md`](./migration-troubleshooting.md)。

## 14. 切换到阿里云托管组件

单机默认组件足够开始使用。需要更高可靠性时，可逐步替换：

| 组件 | 阿里云托管服务 | 关键配置 |
| --- | --- | --- |
| PostgreSQL / ParadeDB | RDS PostgreSQL | `DB_HOST`、`DB_PORT`、`DB_USER`、`DB_PASSWORD`、`DB_NAME` |
| Redis | Tair / 云数据库 Redis | `REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`，按需启用 TLS |
| 文件存储 | OSS | `STORAGE_TYPE=oss` 与 OSS 相关变量 |
| 入口流量 | SLB / ALB / WAF | 后端指向 ECS `80/443` 或 ACK Ingress |

使用 RDS PostgreSQL 前，必须确认目标实例支持项目迁移需要的扩展：`pg_trgm`、`vector`，以及默认 `RETRIEVE_DRIVER=postgres` 场景下使用的 ParadeDB `pg_search`。如果 RDS 不支持 `pg_search`，建议继续使用 Compose 内置 ParadeDB，或切换到 Qdrant、Milvus、OpenSearch、Doris 等外部检索引擎。

## 15. ACK / Helm 部署

当需要 Kubernetes 化部署时，使用仓库内 Helm Chart：

```bash
helm install weknora ./helm \
  --namespace weknora \
  --create-namespace \
  --set secrets.dbPassword="$(openssl rand -base64 24)" \
  --set secrets.redisPassword="$(openssl rand -base64 24)" \
  --set secrets.jwtSecret="$(openssl rand -base64 32)"
```

带 Ingress 的示例：

```bash
helm install weknora ./helm \
  --namespace weknora \
  --create-namespace \
  --set ingress.enabled=true \
  --set ingress.host=kb.example.com \
  --set ingress.tls.enabled=true \
  --set ingress.tls.secretName=weknora-tls \
  --set secrets.dbPassword="$(openssl rand -base64 24)" \
  --set secrets.redisPassword="$(openssl rand -base64 24)" \
  --set secrets.jwtSecret="$(openssl rand -base64 32)"
```

Helm 前置条件：Kubernetes 1.25+、Helm 3.10+、可用 PV provisioner、Ingress Controller。更多参数见 [`helm/README.md`](../helm/README.md)。

## 16. 常见问题

### 访问不到 Web UI

1. 检查安全组是否放行 `80/443`。
2. 检查前端容器端口：

```bash
docker compose ps frontend
```

3. 如果配置了宿主机 NGINX，检查 `FRONTEND_PORT` 是否是 `127.0.0.1:18080`，以及 NGINX 是否代理到该端口。

### 上传文件解析失败

查看 docreader 与 app 日志：

```bash
docker compose logs --tail=200 docreader
docker compose logs --tail=200 app
```

同时确认 `MAX_FILE_SIZE_MB` 是否满足上传文件大小，ECS 磁盘是否充足。

### 模型调用失败

1. 在 UI “设置 -> 模型管理”里测试模型。
2. 确认 ECS 能访问模型服务商地址。
3. 如果使用阿里云 DashScope，确认 `provider=aliyun`、Base URL 与 API Key 正确。

### 数据库迁移失败

查看：

```bash
docker compose logs --tail=300 app | grep -i migration
```

常见原因是 PostgreSQL 扩展缺失、权限不足、磁盘不足或 dirty migration state。参考 [`docs/migration-troubleshooting.md`](./migration-troubleshooting.md)。

### Docker Hub 拉取失败

可以选择：

1. 在网络可用的机器上提前 `docker pull`，再推送到自己的阿里云 ACR 仓库。
2. 修改 `docker-compose.override.yml` 中的 `image` 地址指向 ACR。
3. 使用源码构建，但仍需确保基础镜像、Go/Python/Node 依赖源可访问。

## 17. 参考资料

- [阿里云 ECS 文档](https://help.aliyun.com/zh/ecs/)
- [阿里云安全组文档](https://help.aliyun.com/zh/ecs/user-guide/security-groups)
- [阿里云 ICP 备案文档](https://help.aliyun.com/zh/icp-filing/)
- [Docker Engine Ubuntu 安装文档](https://docs.docker.com/engine/install/ubuntu/)
- [项目 Docker Compose 快速开始](../README_CN.md#-快速开始)
- [项目 Helm Chart 文档](../helm/README.md)
