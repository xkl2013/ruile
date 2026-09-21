#!/usr/bin/env bash
set -euo pipefail

SOURCE_TENANT_ID=""
TARGET_TENANT_ID=""
KB_ID=""
APPLY=0
COMPOSE_FILE=""
POSTGRES_SERVICE=""
POSTGRES_CONTAINER=""

usage() {
  cat <<'EOF'
一次性清洗知识库迁移后仍指向源租户的历史存储路径。

默认仅预演并回滚；确认输出无误后再加 --apply。

使用前提:
  1. 先完成物理文件/对象从源租户路径到目标租户路径的复制或移动。
  2. 确认目标知识库已经属于目标租户。
  3. 执行期间停止 app、worker 和上传/解析任务。

用法:
  scripts/clean_migrated_kb_paths.sh \
    --source-tenant-id 10000 \
    --target-tenant-id 10003 \
    --kb-id 00000000-0000-0000-0000-000000000000

可选参数:
  --kb-id UUID            只清洗一个知识库；默认清洗目标租户下全部知识库
  --compose-file FILE     指定 docker compose 文件
  --postgres-service NAME 指定 Compose 中的 PostgreSQL 服务名
  --postgres-container ID 指定运行中的 PostgreSQL 容器名或容器 ID
  --apply                 提交事务；不传时执行完整预演后 ROLLBACK
  -h, --help              显示帮助
EOF
}

die() {
  echo "错误: $*" >&2
  exit 1
}

while (($# > 0)); do
  case "$1" in
    --source-tenant-id)
      [[ $# -ge 2 ]] || die "--source-tenant-id 缺少参数"
      SOURCE_TENANT_ID="$2"
      shift 2
      ;;
    --target-tenant-id)
      [[ $# -ge 2 ]] || die "--target-tenant-id 缺少参数"
      TARGET_TENANT_ID="$2"
      shift 2
      ;;
    --kb-id)
      [[ $# -ge 2 ]] || die "--kb-id 缺少参数"
      KB_ID="$2"
      shift 2
      ;;
    --compose-file)
      [[ $# -ge 2 ]] || die "--compose-file 缺少参数"
      COMPOSE_FILE="$2"
      shift 2
      ;;
    --postgres-service)
      [[ $# -ge 2 ]] || die "--postgres-service 缺少参数"
      POSTGRES_SERVICE="$2"
      shift 2
      ;;
    --postgres-container)
      [[ $# -ge 2 ]] || die "--postgres-container 缺少参数"
      POSTGRES_CONTAINER="$2"
      shift 2
      ;;
    --apply)
      APPLY=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "未知参数: $1"
      ;;
  esac
done

[[ "$SOURCE_TENANT_ID" =~ ^[1-9][0-9]*$ ]] || die "源 tenant_id 格式不正确"
[[ "$TARGET_TENANT_ID" =~ ^[1-9][0-9]*$ ]] || die "目标 tenant_id 格式不正确"
[[ "$SOURCE_TENANT_ID" != "$TARGET_TENANT_ID" ]] || die "源和目标 tenant_id 不能相同"
[[ -z "$KB_ID" || "$KB_ID" =~ ^[0-9A-Fa-f-]{36}$ ]] ||
  die "knowledge base ID 格式不正确"

command -v docker >/dev/null 2>&1 || die "未找到 docker"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-WeKnora}"

compose=(docker compose)
if [[ -n "$COMPOSE_FILE" ]]; then
  compose+=(-f "$COMPOSE_FILE")
fi

db_exec=()
if [[ -n "$POSTGRES_CONTAINER" ]]; then
  docker inspect "$POSTGRES_CONTAINER" >/dev/null 2>&1 ||
    die "未找到 PostgreSQL 容器: $POSTGRES_CONTAINER"
  db_exec=(docker exec -i "$POSTGRES_CONTAINER")
else
  compose_services="$("${compose[@]}" config --services 2>/dev/null || true)"
  if [[ -n "$POSTGRES_SERVICE" ]]; then
    grep -Fxq "$POSTGRES_SERVICE" <<<"$compose_services" ||
      die "docker compose 中未找到 PostgreSQL 服务: $POSTGRES_SERVICE"
  else
    for candidate in postgres db database; do
      if grep -Fxq "$candidate" <<<"$compose_services"; then
        POSTGRES_SERVICE="$candidate"
        break
      fi
    done
  fi

  if [[ -n "$POSTGRES_SERVICE" ]]; then
    postgres_container_id="$("${compose[@]}" ps -q "$POSTGRES_SERVICE" 2>/dev/null || true)"
    if [[ -n "$postgres_container_id" ]]; then
      db_exec=("${compose[@]}" exec -T "$POSTGRES_SERVICE")
    fi
  fi

  if ((${#db_exec[@]} == 0)); then
    POSTGRES_CONTAINER="$(
      docker ps --format '{{.Names}}' |
        grep -Ei 'postgres' |
        head -n 1 || true
    )"
    [[ -n "$POSTGRES_CONTAINER" ]] ||
      die "未找到运行中的 PostgreSQL 服务或容器；可用 --postgres-service 或 --postgres-container 指定"
    db_exec=(docker exec -i "$POSTGRES_CONTAINER")
  fi
fi

if ((APPLY == 1)); then
  END_TRANSACTION="COMMIT;"
  MODE_LABEL="正式清洗"
else
  END_TRANSACTION="ROLLBACK;"
  MODE_LABEL="预演（最终回滚）"
fi

echo "模式: $MODE_LABEL"
echo "源 tenant_id: $SOURCE_TENANT_ID"
echo "目标 tenant_id: $TARGET_TENANT_ID"
[[ -n "$KB_ID" ]] && echo "指定知识库: $KB_ID"
echo

"${db_exec[@]}" \
  psql -X -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -P pager=off \
  -v source_tenant_id="$SOURCE_TENANT_ID" \
  -v target_tenant_id="$TARGET_TENANT_ID" \
  -v kb_id="$KB_ID" <<SQL
BEGIN;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '30min';
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TEMP TABLE _path_args (
    source_tenant_id BIGINT NOT NULL,
    target_tenant_id BIGINT NOT NULL,
    requested_kb_id VARCHAR(36)
) ON COMMIT DROP;

INSERT INTO _path_args (
    source_tenant_id,
    target_tenant_id,
    requested_kb_id
) VALUES (
    :'source_tenant_id'::BIGINT,
    :'target_tenant_id'::BIGINT,
    NULLIF(:'kb_id', '')
);

CREATE TEMP TABLE _kb_scope (
    id VARCHAR(36) PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO _kb_scope (id)
SELECT id
  FROM knowledge_bases
 WHERE tenant_id = (SELECT target_tenant_id FROM _path_args)
   AND deleted_at IS NULL
   AND is_temporary = FALSE
   AND (
        (SELECT requested_kb_id FROM _path_args) IS NULL
        OR id = (SELECT requested_kb_id FROM _path_args)
   );

DO \$\$
DECLARE
    kb_count INTEGER;
    invalid_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO kb_count FROM _kb_scope;
    IF kb_count = 0 THEN
        RAISE EXCEPTION '目标租户没有匹配的可清洗知识库';
    END IF;

    SELECT COUNT(*)
      INTO invalid_count
      FROM knowledge_bases
     WHERE id IN (SELECT id FROM _kb_scope)
       AND tenant_id <> (SELECT target_tenant_id FROM _path_args);
    IF invalid_count > 0 THEN
        RAISE EXCEPTION '知识库范围校验失败：仍有 % 个知识库不属于目标租户', invalid_count;
    END IF;
END
\$\$;

CREATE TEMP TABLE _path_rewrite_stats (
    table_name TEXT NOT NULL,
    row_count BIGINT NOT NULL
) ON COMMIT DROP;

CREATE OR REPLACE FUNCTION pg_temp.rewrite_provider_tenant_paths(
    value TEXT,
    source_tenant TEXT,
    target_tenant TEXT
) RETURNS TEXT
LANGUAGE plpgsql
AS \$\$
DECLARE
    scheme TEXT;
BEGIN
    IF value IS NULL OR value = '' THEN
        RETURN value;
    END IF;

    -- The tenant segment may appear directly after the scheme or after a
    -- bucket/prefix, depending on the storage provider and historical format.
    FOREACH scheme IN ARRAY ARRAY['local', 'minio', 'cos', 'tos', 's3', 'oss', 'ks3', 'obs']
    LOOP
        value := regexp_replace(
            value,
            '(' || scheme || '://[^[:space:]<>"]*)/' || source_tenant || '/',
            '\1/' || target_tenant || '/',
            'g'
        );
        value := replace(
            value,
            scheme || '://' || source_tenant || '/',
            scheme || '://' || target_tenant || '/'
        );
    END LOOP;
    RETURN value;
END
\$\$;

-- Refuse to silently rewrite a KB that is still owned by the source tenant.
DO \$\$
DECLARE
    invalid_count INTEGER;
    source_tenant BIGINT;
BEGIN
    SELECT source_tenant_id INTO source_tenant FROM _path_args;

    SELECT COUNT(*)
      INTO invalid_count
      FROM knowledge_bases
     WHERE id IN (SELECT id FROM _kb_scope)
       AND tenant_id = source_tenant;
    IF invalid_count > 0 THEN
        RAISE EXCEPTION
            '知识库仍属于源租户，请先完成知识库归属迁移：% 个',
            invalid_count;
    END IF;
END
\$\$;

CREATE TEMP TABLE _resource_scope (
    id VARCHAR(36) PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO _resource_scope (id)
SELECT DISTINCT rb.resource_id
  FROM resource_bindings rb
 WHERE rb.owner_type = 'knowledge'
   AND rb.owner_id IN (
       SELECT id
         FROM knowledges
        WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
   )
ON CONFLICT DO NOTHING;

INSERT INTO _resource_scope (id)
SELECT DISTINCT r.id
  FROM knowledge_bases kb
  CROSS JOIN LATERAL regexp_matches(
      COALESCE(kb.icon, ''),
      'resource://([0-9A-Za-z_-]{22})',
      'g'
  ) match
  JOIN resources r ON r.handle = match[1]
 WHERE kb.id IN (SELECT id FROM _kb_scope)
ON CONFLICT DO NOTHING;

INSERT INTO _resource_scope (id)
SELECT DISTINCT r.id
  FROM knowledges k
  CROSS JOIN LATERAL regexp_matches(
      COALESCE(k.file_path, '') || ' ' || COALESCE(k.metadata::TEXT, ''),
      'resource://([0-9A-Za-z_-]{22})',
      'g'
  ) match
  JOIN resources r ON r.handle = match[1]
 WHERE k.knowledge_base_id IN (SELECT id FROM _kb_scope)
ON CONFLICT DO NOTHING;

INSERT INTO _resource_scope (id)
SELECT DISTINCT r.id
  FROM chunks c
  CROSS JOIN LATERAL regexp_matches(
      COALESCE(c.content, '') || ' ' || COALESCE(c.image_info, ''),
      'resource://([0-9A-Za-z_-]{22})',
      'g'
  ) match
  JOIN resources r ON r.handle = match[1]
 WHERE c.knowledge_base_id IN (SELECT id FROM _kb_scope)
ON CONFLICT DO NOTHING;

INSERT INTO _resource_scope (id)
SELECT DISTINCT r.id
  FROM wiki_pages wp
  CROSS JOIN LATERAL regexp_matches(
      COALESCE(wp.content, '') || ' ' || COALESCE(wp.page_metadata::TEXT, ''),
      'resource://([0-9A-Za-z_-]{22})',
      'g'
  ) match
  JOIN resources r ON r.handle = match[1]
 WHERE wp.knowledge_base_id IN (SELECT id FROM _kb_scope)
ON CONFLICT DO NOTHING;

WITH updated AS (
    UPDATE knowledge_bases
       SET icon = pg_temp.rewrite_provider_tenant_paths(
           icon,
           (SELECT source_tenant_id::TEXT FROM _path_args),
           (SELECT target_tenant_id::TEXT FROM _path_args)
       ),
           updated_at = CURRENT_TIMESTAMP
     WHERE id IN (SELECT id FROM _kb_scope)
       AND icon IS NOT NULL
       AND icon <> pg_temp.rewrite_provider_tenant_paths(
           icon,
           (SELECT source_tenant_id::TEXT FROM _path_args),
           (SELECT target_tenant_id::TEXT FROM _path_args)
       )
    RETURNING 1
)
INSERT INTO _path_rewrite_stats VALUES ('knowledge_bases.icon', (SELECT COUNT(*) FROM updated));

WITH updated AS (
    UPDATE knowledges
       SET file_path = pg_temp.rewrite_provider_tenant_paths(
           file_path,
           (SELECT source_tenant_id::TEXT FROM _path_args),
           (SELECT target_tenant_id::TEXT FROM _path_args)
       ),
           metadata = CASE
               WHEN metadata IS NULL THEN NULL
               ELSE pg_temp.rewrite_provider_tenant_paths(
                   metadata::TEXT,
                   (SELECT source_tenant_id::TEXT FROM _path_args),
                   (SELECT target_tenant_id::TEXT FROM _path_args)
               )::JSONB
           END,
           updated_at = CURRENT_TIMESTAMP
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            file_path IS NOT NULL
            OR metadata IS NOT NULL
       )
    RETURNING 1
)
INSERT INTO _path_rewrite_stats VALUES ('knowledges', (SELECT COUNT(*) FROM updated));

WITH updated AS (
    UPDATE chunks
       SET content = pg_temp.rewrite_provider_tenant_paths(
               content,
               (SELECT source_tenant_id::TEXT FROM _path_args),
               (SELECT target_tenant_id::TEXT FROM _path_args)
           ),
           image_info = pg_temp.rewrite_provider_tenant_paths(
               image_info,
               (SELECT source_tenant_id::TEXT FROM _path_args),
               (SELECT target_tenant_id::TEXT FROM _path_args)
           )
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            content IS NOT NULL
            OR image_info IS NOT NULL
       )
    RETURNING 1
)
INSERT INTO _path_rewrite_stats VALUES ('chunks', (SELECT COUNT(*) FROM updated));

WITH updated AS (
    UPDATE wiki_pages
       SET content = pg_temp.rewrite_provider_tenant_paths(
               content,
               (SELECT source_tenant_id::TEXT FROM _path_args),
               (SELECT target_tenant_id::TEXT FROM _path_args)
           ),
           page_metadata = CASE
               WHEN page_metadata IS NULL THEN NULL
               ELSE pg_temp.rewrite_provider_tenant_paths(
                   page_metadata::TEXT,
                   (SELECT source_tenant_id::TEXT FROM _path_args),
                   (SELECT target_tenant_id::TEXT FROM _path_args)
               )::JSONB
           END,
           updated_at = CURRENT_TIMESTAMP
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            content IS NOT NULL
            OR page_metadata IS NOT NULL
       )
    RETURNING 1
)
INSERT INTO _path_rewrite_stats VALUES ('wiki_pages', (SELECT COUNT(*) FROM updated));

WITH updated AS (
    UPDATE resources
       SET physical_path = pg_temp.rewrite_provider_tenant_paths(
               physical_path,
               (SELECT source_tenant_id::TEXT FROM _path_args),
               (SELECT target_tenant_id::TEXT FROM _path_args)
           ),
           location_hash = encode(
               digest(
                   pg_temp.rewrite_provider_tenant_paths(
                       physical_path,
                       (SELECT source_tenant_id::TEXT FROM _path_args),
                       (SELECT target_tenant_id::TEXT FROM _path_args)
                   ),
                   'sha256'
               ),
               'hex'
           ),
           updated_at = CURRENT_TIMESTAMP
     WHERE id IN (
         SELECT id FROM _resource_scope
     )
       AND physical_path IS NOT NULL
    RETURNING 1
)
INSERT INTO _path_rewrite_stats VALUES ('resources', (SELECT COUNT(*) FROM updated));

\echo ''
\echo '清洗结果预览'
SELECT table_name, row_count
  FROM _path_rewrite_stats
 ORDER BY table_name;

DO \$\$
DECLARE
    remaining_count BIGINT;
    source_tenant TEXT;
BEGIN
    SELECT source_tenant_id::TEXT INTO source_tenant FROM _path_args;

    SELECT COUNT(*)
      INTO remaining_count
      FROM knowledges
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            file_path LIKE '%local://' || source_tenant || '/%'
            OR file_path LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR file_path LIKE '%obs://%' || '/' || source_tenant || '/%'
       );
    IF remaining_count > 0 THEN
        RAISE EXCEPTION
            '清洗校验失败：knowledges.file_path 仍有 % 条源租户路径',
            remaining_count;
    END IF;

    SELECT COUNT(*)
      INTO remaining_count
      FROM resources r
     WHERE r.id IN (SELECT id FROM _resource_scope)
       AND (
            r.physical_path LIKE '%local://' || source_tenant || '/%'
            OR r.physical_path LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR r.physical_path LIKE '%obs://%' || '/' || source_tenant || '/%'
       );
    IF remaining_count > 0 THEN
        RAISE EXCEPTION
            '清洗校验失败：resources.physical_path 仍有 % 条源租户路径',
            remaining_count;
    END IF;

    SELECT COUNT(*)
      INTO remaining_count
      FROM chunks
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            content LIKE '%local://' || source_tenant || '/%'
            OR content LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR content LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR content LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR content LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR content LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR content LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR content LIKE '%obs://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%local://' || source_tenant || '/%'
            OR image_info LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR image_info LIKE '%obs://%' || '/' || source_tenant || '/%'
       );
    IF remaining_count > 0 THEN
        RAISE EXCEPTION
            '清洗校验失败：chunks 中仍有 % 条源租户路径',
            remaining_count;
    END IF;

    SELECT COUNT(*)
      INTO remaining_count
      FROM wiki_pages
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND (
            content LIKE '%local://' || source_tenant || '/%'
            OR content LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR content LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR content LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR content LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR content LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR content LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR content LIKE '%obs://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%local://' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%minio://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%cos://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%tos://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%s3://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%oss://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%ks3://%' || '/' || source_tenant || '/%'
            OR page_metadata::TEXT LIKE '%obs://%' || '/' || source_tenant || '/%'
       );
    IF remaining_count > 0 THEN
        RAISE EXCEPTION
            '清洗校验失败：wiki_pages 中仍有 % 条源租户路径',
            remaining_count;
    END IF;
END
\$\$;

$END_TRANSACTION
SQL

if ((APPLY == 1)); then
  echo
  echo "历史存储路径清洗已提交。请启动 app 后验证原文件预览、图片和检索。"
else
  echo
  echo "预演完成，数据库已回滚。确认输出后使用相同参数加 --apply。"
fi
