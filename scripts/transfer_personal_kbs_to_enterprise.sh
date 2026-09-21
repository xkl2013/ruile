#!/usr/bin/env bash
set -euo pipefail

SOURCE_PHONE=""
TARGET_PHONE=""
TARGET_TENANT_ID=""
KB_ID=""
APPLY=0
COMPOSE_FILE=""
POSTGRES_SERVICE=""
POSTGRES_CONTAINER=""

usage() {
  cat <<'EOF'
一次性把个人空间中的知识库迁移到另一个账号所属的企业空间。

默认仅预演并回滚；确认输出无误后再加 --apply。

用法:
  scripts/transfer_personal_kbs_to_enterprise.sh \
    --source-phone 13258978288 \
    --target-phone 13901156168

  scripts/transfer_personal_kbs_to_enterprise.sh \
    --source-phone 13258978288 \
    --target-phone 13901156168 \
    --apply

可选参数:
  --target-tenant-id ID   目标账号属于多个企业时，明确指定企业 tenant_id
  --kb-id UUID            只迁移一个知识库；默认迁移源个人空间下全部非临时知识库
  --compose-file FILE     指定 docker compose 文件
  --postgres-service NAME 指定 Compose 中的 PostgreSQL 服务名
  --postgres-container ID 指定运行中的 PostgreSQL 容器名或容器 ID
  --apply                 提交事务；不传时执行完整预演后 ROLLBACK
  -h, --help              显示帮助

执行要求:
  1. 先备份 PostgreSQL。
  2. 执行期间停止 app，避免上传、解析和索引任务并发写入。
  3. 迁移完成后启动 app，并验证知识库列表、文件预览和检索。
EOF
}

die() {
  echo "错误: $*" >&2
  exit 1
}

while (($# > 0)); do
  case "$1" in
    --source-phone)
      [[ $# -ge 2 ]] || die "--source-phone 缺少参数"
      SOURCE_PHONE="$2"
      shift 2
      ;;
    --target-phone)
      [[ $# -ge 2 ]] || die "--target-phone 缺少参数"
      TARGET_PHONE="$2"
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

[[ "$SOURCE_PHONE" =~ ^[0-9]{6,20}$ ]] || die "源手机号格式不正确"
[[ "$TARGET_PHONE" =~ ^[0-9]{6,20}$ ]] || die "目标手机号格式不正确"
[[ "$SOURCE_PHONE" != "$TARGET_PHONE" ]] || die "源账号和目标账号不能相同"
[[ -z "$TARGET_TENANT_ID" || "$TARGET_TENANT_ID" =~ ^[0-9]+$ ]] ||
  die "target tenant_id 必须是正整数"
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
    if ! grep -Fxq "$POSTGRES_SERVICE" <<<"$compose_services"; then
      die "docker compose 中未找到 PostgreSQL 服务: $POSTGRES_SERVICE"
    fi
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
        grep -Ei '^(WeKnora[-_]postgres|weknora[-_]postgres)([-_]dev)?$' |
        head -n 1 || true
    )"
    if [[ -z "$POSTGRES_CONTAINER" ]]; then
      POSTGRES_CONTAINER="$(
        docker ps --format '{{.Names}}' |
          grep -Ei 'postgres' |
          head -n 1 || true
      )"
    fi
    [[ -n "$POSTGRES_CONTAINER" ]] ||
      die "未找到运行中的 PostgreSQL 服务或容器；可用 --postgres-service 或 --postgres-container 指定"
    db_exec=(docker exec -i "$POSTGRES_CONTAINER")
  fi
fi

if ((APPLY == 1)); then
  END_TRANSACTION="COMMIT;"
  MODE_LABEL="正式迁移"
else
  END_TRANSACTION="ROLLBACK;"
  MODE_LABEL="预演（最终回滚）"
fi

echo "模式: $MODE_LABEL"
echo "源账号: $SOURCE_PHONE"
echo "目标账号: $TARGET_PHONE"
[[ -n "$TARGET_TENANT_ID" ]] && echo "目标企业 tenant_id: $TARGET_TENANT_ID"
[[ -n "$KB_ID" ]] && echo "指定知识库: $KB_ID"
echo

"${db_exec[@]}" \
  psql -X -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 -P pager=off \
  -v source_phone="$SOURCE_PHONE" \
  -v target_phone="$TARGET_PHONE" \
  -v target_tenant_id="$TARGET_TENANT_ID" \
  -v kb_id="$KB_ID" <<SQL
BEGIN;
SET LOCAL lock_timeout = '10s';
SET LOCAL statement_timeout = '30min';

CREATE TEMP TABLE _migration_args (
    source_phone TEXT NOT NULL,
    target_phone TEXT NOT NULL,
    requested_target_tenant_id BIGINT,
    requested_kb_id VARCHAR(36),
    source_user_id VARCHAR(36),
    target_user_id VARCHAR(36),
    source_tenant_id BIGINT,
    target_tenant_id BIGINT,
    moved_storage_bytes BIGINT NOT NULL DEFAULT 0
) ON COMMIT DROP;

INSERT INTO _migration_args (
    source_phone,
    target_phone,
    requested_target_tenant_id,
    requested_kb_id
) VALUES (
    :'source_phone',
    :'target_phone',
    NULLIF(:'target_tenant_id', '')::BIGINT,
    NULLIF(:'kb_id', '')::VARCHAR(36)
);

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    matched_count INTEGER;
BEGIN
    SELECT * INTO args FROM _migration_args;

    SELECT COUNT(*), MIN(id), MIN(tenant_id)
      INTO matched_count, args.source_user_id, args.source_tenant_id
      FROM users
     WHERE email = args.source_phone OR username = args.source_phone;
    IF matched_count <> 1 THEN
        RAISE EXCEPTION '源账号 % 匹配到 % 条用户记录，要求恰好 1 条',
            args.source_phone, matched_count;
    END IF;
    IF args.source_tenant_id IS NULL OR args.source_tenant_id = 0 THEN
        RAISE EXCEPTION '源账号 % 没有有效的默认个人空间', args.source_phone;
    END IF;
    IF EXISTS (
        SELECT 1 FROM tenants
         WHERE id = args.source_tenant_id
           AND space_type = 'organization'
           AND deleted_at IS NULL
    ) THEN
        RAISE EXCEPTION '源账号默认空间 % 已是企业空间，拒绝按个人空间迁移',
            args.source_tenant_id;
    END IF;

    SELECT COUNT(*), MIN(id)
      INTO matched_count, args.target_user_id
      FROM users
     WHERE (email = args.target_phone OR username = args.target_phone)
       AND deleted_at IS NULL
       AND is_active = TRUE;
    IF matched_count <> 1 THEN
        RAISE EXCEPTION '目标账号 % 匹配到 % 条有效用户记录，要求恰好 1 条',
            args.target_phone, matched_count;
    END IF;

    IF args.requested_target_tenant_id IS NOT NULL THEN
        SELECT COUNT(*), MIN(tm.tenant_id)
          INTO matched_count, args.target_tenant_id
          FROM tenant_members tm
          JOIN tenants t ON t.id = tm.tenant_id
         WHERE tm.user_id = args.target_user_id
           AND tm.tenant_id = args.requested_target_tenant_id
           AND tm.status = 'active'
           AND tm.deleted_at IS NULL
           AND t.deleted_at IS NULL
           AND (
                t.space_type = 'organization'
                OR EXISTS (
                    SELECT 1
                      FROM organizations o
                     WHERE o.owner_tenant_id = t.id
                       AND o.deleted_at IS NULL
                )
           );
    ELSE
        SELECT COUNT(*), MIN(tm.tenant_id)
          INTO matched_count, args.target_tenant_id
          FROM tenant_members tm
          JOIN tenants t ON t.id = tm.tenant_id
         WHERE tm.user_id = args.target_user_id
           AND tm.status = 'active'
           AND tm.deleted_at IS NULL
           AND t.deleted_at IS NULL
           AND (
                t.space_type = 'organization'
                OR EXISTS (
                    SELECT 1
                      FROM organizations o
                     WHERE o.owner_tenant_id = t.id
                       AND o.deleted_at IS NULL
                )
           );
    END IF;

    IF matched_count = 0 THEN
        RAISE EXCEPTION '目标账号 % 没有有效企业空间', args.target_phone;
    ELSIF matched_count > 1 THEN
        RAISE EXCEPTION '目标账号 % 属于 % 个企业空间，请使用 --target-tenant-id 指定',
            args.target_phone, matched_count;
    END IF;
    IF args.source_tenant_id = args.target_tenant_id THEN
        RAISE EXCEPTION '源空间和目标企业空间相同，无需迁移';
    END IF;

    UPDATE _migration_args
       SET source_user_id = args.source_user_id,
           target_user_id = args.target_user_id,
           source_tenant_id = args.source_tenant_id,
           target_tenant_id = args.target_tenant_id;
END
\$\$;

CREATE TEMP TABLE _kb_scope (
    id VARCHAR(36) PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO _kb_scope (id)
SELECT kb.id
  FROM knowledge_bases kb
  CROSS JOIN _migration_args args
 WHERE kb.tenant_id = args.source_tenant_id
   AND kb.deleted_at IS NULL
   AND kb.is_temporary = FALSE
   AND (
        args.requested_kb_id IS NULL
        OR kb.id = args.requested_kb_id
   );

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    kb_count INTEGER;
    processing_count INTEGER;
    moved_bytes BIGINT;
    target_used BIGINT;
    target_quota BIGINT;
BEGIN
    SELECT * INTO args FROM _migration_args;
    SELECT COUNT(*) INTO kb_count FROM _kb_scope;
    IF kb_count = 0 THEN
        RAISE EXCEPTION '源个人空间没有匹配的可迁移知识库';
    END IF;

    IF args.requested_kb_id IS NOT NULL AND kb_count <> 1 THEN
        RAISE EXCEPTION '指定知识库 % 不属于源个人空间', args.requested_kb_id;
    END IF;

    SELECT COUNT(*)
      INTO processing_count
      FROM knowledges
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND deleted_at IS NULL
       AND parse_status IN ('pending', 'processing', 'finalizing', 'deleting');
    IF processing_count > 0 THEN
        RAISE EXCEPTION '有 % 条知识正在处理，请等待任务结束并停止 app 后重试',
            processing_count;
    END IF;

    SELECT COALESCE(SUM(storage_size), 0)
      INTO moved_bytes
      FROM knowledges
     WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
       AND deleted_at IS NULL;

    SELECT storage_used, storage_quota
      INTO target_used, target_quota
      FROM tenants
     WHERE id = args.target_tenant_id
     FOR UPDATE;
    IF target_quota > 0 AND target_used + moved_bytes > target_quota THEN
        RAISE EXCEPTION
            '目标企业容量不足：当前 % 字节 + 待迁移 % 字节 > 配额 % 字节',
            target_used, moved_bytes, target_quota;
    END IF;

    UPDATE _migration_args SET moved_storage_bytes = moved_bytes;
END
\$\$;

LOCK TABLE knowledge_bases, knowledges, chunks IN SHARE ROW EXCLUSIVE MODE;

CREATE TEMP TABLE _storage_map (
    old_id VARCHAR(36) PRIMARY KEY,
    new_id VARCHAR(36) NOT NULL
) ON COMMIT DROP;

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    source_backend storage_backends%ROWTYPE;
    mapped_id VARCHAR(36);
    mapped_name VARCHAR(255);
BEGIN
    SELECT * INTO args FROM _migration_args;
    FOR source_backend IN
        SELECT DISTINCT sb.*
          FROM storage_backends sb
          JOIN knowledge_bases kb ON kb.storage_backend_id = sb.id
         WHERE kb.id IN (SELECT id FROM _kb_scope)
           AND sb.deleted_at IS NULL
    LOOP
        SELECT id
          INTO mapped_id
          FROM storage_backends
         WHERE tenant_id = args.target_tenant_id
           AND provider = source_backend.provider
           AND config = source_backend.config
           AND status = 'active'
           AND deleted_at IS NULL
         ORDER BY created_at, id
         LIMIT 1;

        IF mapped_id IS NULL THEN
            mapped_id := uuid_generate_v4()::TEXT;
            mapped_name := LEFT(
                source_backend.name || ' (迁移-' || LEFT(mapped_id, 8) || ')',
                255
            );
            INSERT INTO storage_backends (
                id, tenant_id, name, provider, config, source, status,
                legacy_alias, created_at, updated_at
            ) VALUES (
                mapped_id, args.target_tenant_id, mapped_name,
                source_backend.provider, source_backend.config,
                source_backend.source, source_backend.status,
                FALSE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
            );
        END IF;

        INSERT INTO _storage_map (old_id, new_id)
        VALUES (source_backend.id, mapped_id);
    END LOOP;
END
\$\$;

CREATE TEMP TABLE _vector_map (
    old_id VARCHAR(36) PRIMARY KEY,
    new_id VARCHAR(36) NOT NULL
) ON COMMIT DROP;

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    source_store vector_stores%ROWTYPE;
    mapped_id VARCHAR(36);
    mapped_name VARCHAR(255);
BEGIN
    SELECT * INTO args FROM _migration_args;
    FOR source_store IN
        SELECT DISTINCT vs.*
          FROM vector_stores vs
          JOIN knowledge_bases kb ON kb.vector_store_id = vs.id
         WHERE kb.id IN (SELECT id FROM _kb_scope)
           AND vs.deleted_at IS NULL
    LOOP
        SELECT id
          INTO mapped_id
          FROM vector_stores
         WHERE tenant_id = args.target_tenant_id
           AND engine_type = source_store.engine_type
           AND connection_config = source_store.connection_config
           AND index_config = source_store.index_config
           AND deleted_at IS NULL
         ORDER BY created_at, id
         LIMIT 1;

        IF mapped_id IS NULL THEN
            mapped_id := uuid_generate_v4()::TEXT;
            mapped_name := LEFT(
                source_store.name || ' (迁移-' || LEFT(mapped_id, 8) || ')',
                255
            );
            INSERT INTO vector_stores (
                id, name, engine_type, connection_config, index_config,
                tenant_id, created_at, updated_at
            ) VALUES (
                mapped_id, mapped_name, source_store.engine_type,
                source_store.connection_config, source_store.index_config,
                args.target_tenant_id, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
            );
        END IF;

        INSERT INTO _vector_map (old_id, new_id)
        VALUES (source_store.id, mapped_id);
    END LOOP;
END
\$\$;

CREATE TEMP TABLE _model_map (
    old_id VARCHAR(64) PRIMARY KEY,
    new_id VARCHAR(64) NOT NULL
) ON COMMIT DROP;

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    source_model models%ROWTYPE;
    mapped_id VARCHAR(64);
BEGIN
    SELECT * INTO args FROM _migration_args;
    FOR source_model IN
        SELECT m.*
          FROM models m
         WHERE m.tenant_id = args.source_tenant_id
           AND m.deleted_at IS NULL
    LOOP
        SELECT id
          INTO mapped_id
          FROM models
         WHERE tenant_id = args.target_tenant_id
           AND name = source_model.name
           AND type = source_model.type
           AND source = source_model.source
           AND parameters = source_model.parameters
           AND deleted_at IS NULL
         ORDER BY created_at, id
         LIMIT 1;

        IF mapped_id IS NULL THEN
            mapped_id := uuid_generate_v4()::TEXT;
            INSERT INTO models (
                id, tenant_id, name, display_name, type, source, description,
                parameters, is_default, status, is_builtin, managed_by,
                created_at, updated_at
            ) VALUES (
                mapped_id, args.target_tenant_id, source_model.name,
                source_model.display_name, source_model.type,
                source_model.source, source_model.description,
                source_model.parameters, FALSE, source_model.status,
                source_model.is_builtin, source_model.managed_by,
                CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
            );
        END IF;

        INSERT INTO _model_map (old_id, new_id)
        VALUES (source_model.id, mapped_id);
    END LOOP;
END
\$\$;

CREATE TEMP TABLE _knowledge_scope (
    id VARCHAR(36) PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO _knowledge_scope (id)
SELECT id
  FROM knowledges
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

CREATE TEMP TABLE _resource_scope (
    id VARCHAR(36) PRIMARY KEY
) ON COMMIT DROP;

INSERT INTO _resource_scope (id)
SELECT DISTINCT rb.resource_id
  FROM resource_bindings rb
 WHERE rb.owner_type = 'knowledge'
   AND rb.owner_id IN (SELECT id FROM _knowledge_scope)
ON CONFLICT DO NOTHING;

INSERT INTO _resource_scope (id)
SELECT DISTINCT r.id
  FROM knowledge_bases kb
  CROSS JOIN LATERAL regexp_matches(
      kb.icon,
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
 WHERE k.id IN (SELECT id FROM _knowledge_scope)
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
 WHERE c.knowledge_id IN (SELECT id FROM _knowledge_scope)
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

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    conflict_count INTEGER;
BEGIN
    SELECT * INTO args FROM _migration_args;

    SELECT COUNT(*)
      INTO conflict_count
      FROM resource_bindings rb
     WHERE rb.resource_id IN (SELECT id FROM _resource_scope)
       AND NOT (
            rb.owner_type = 'knowledge'
            AND rb.owner_id IN (SELECT id FROM _knowledge_scope)
       )
       AND NOT (
            rb.owner_type = 'knowledge_base'
            AND rb.owner_id IN (SELECT id FROM _kb_scope)
       );
    IF conflict_count > 0 THEN
        RAISE EXCEPTION
            '发现 % 条资源还被迁移范围外的对象引用，拒绝移动共享资源',
            conflict_count;
    END IF;

    SELECT COUNT(*)
      INTO conflict_count
      FROM resources source_resource
      JOIN resources target_resource
        ON target_resource.tenant_id = args.target_tenant_id
       AND target_resource.location_hash = source_resource.location_hash
       AND target_resource.deleted_at IS NULL
       AND target_resource.id <> source_resource.id
     WHERE source_resource.id IN (SELECT id FROM _resource_scope)
       AND source_resource.deleted_at IS NULL;
    IF conflict_count > 0 THEN
        RAISE EXCEPTION
            '目标企业已有 % 条相同物理位置的资源，请先人工合并资源记录',
            conflict_count;
    END IF;
END
\$\$;

-- Remove caller-specific shortcuts and detach old personal chat/channel
-- references. These rows do not grant ownership and should not cross spaces.
DELETE FROM user_kb_pins
 WHERE kb_id IN (SELECT id FROM _kb_scope);

DELETE FROM knowledge_base_subscriptions
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE sessions
   SET knowledge_base_id = NULL,
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE im_channels
   SET knowledge_base_id = '',
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

-- Rewrite concrete storage backend IDs while keeping the provider-side object
-- key unchanged.
DO \$\$
DECLARE
    mapping RECORD;
BEGIN
    FOR mapping IN SELECT * FROM _storage_map LOOP
        UPDATE knowledges
           SET file_path = CASE
               WHEN file_path LIKE 'storage://' || mapping.old_id || '/%'
               THEN 'storage://' || mapping.new_id || '/' ||
                    SUBSTRING(file_path FROM LENGTH('storage://' || mapping.old_id || '/') + 1)
               ELSE file_path
           END
         WHERE id IN (SELECT id FROM _knowledge_scope);

        UPDATE resources
           SET storage_backend_id = mapping.new_id,
               physical_path = CASE
                   WHEN physical_path LIKE 'storage://' || mapping.old_id || '/%'
                   THEN 'storage://' || mapping.new_id || '/' ||
                        SUBSTRING(physical_path FROM LENGTH('storage://' || mapping.old_id || '/') + 1)
                   ELSE physical_path
               END,
               updated_at = CURRENT_TIMESTAMP
         WHERE id IN (SELECT id FROM _resource_scope)
           AND storage_backend_id = mapping.old_id;
    END LOOP;
END
\$\$;

UPDATE resources
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE id IN (SELECT id FROM _resource_scope);

UPDATE resource_bindings
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args)
 WHERE resource_id IN (SELECT id FROM _resource_scope);

UPDATE sync_logs
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE data_source_id IN (
     SELECT id
       FROM data_sources
      WHERE knowledge_base_id IN (SELECT id FROM _kb_scope)
 );

UPDATE data_sources
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE wiki_log_entries
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args)
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE wiki_page_issues
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE wiki_pages
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE wiki_folders
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE knowledge_tags
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE chunks
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE knowledges
   SET tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE kb_shares
   SET source_tenant_id = (SELECT target_tenant_id FROM _migration_args),
       updated_at = CURRENT_TIMESTAMP
 WHERE knowledge_base_id IN (SELECT id FROM _kb_scope);

UPDATE knowledge_bases kb
   SET tenant_id = args.target_tenant_id,
       creator_id = args.target_user_id,
       storage_backend_id = COALESCE(
           (
               SELECT sm.new_id
                 FROM _storage_map sm
                WHERE sm.old_id = kb.storage_backend_id
           ),
           kb.storage_backend_id
       ),
       vector_store_id = COALESCE(
           (
               SELECT vm.new_id
                 FROM _vector_map vm
                WHERE vm.old_id = kb.vector_store_id
           ),
           kb.vector_store_id
       ),
       sort_order = 0,
       updated_at = CURRENT_TIMESTAMP
  FROM _migration_args args
 WHERE kb.id IN (SELECT id FROM _kb_scope);

DO \$\$
DECLARE
    mapping RECORD;
BEGIN
    FOR mapping IN SELECT * FROM _model_map LOOP
        UPDATE knowledge_bases
           SET embedding_model_id = CASE
                   WHEN embedding_model_id = mapping.old_id THEN mapping.new_id
                   ELSE embedding_model_id
               END,
               summary_model_id = CASE
                   WHEN summary_model_id = mapping.old_id THEN mapping.new_id
                   ELSE summary_model_id
               END,
               chunking_config = REPLACE(chunking_config::TEXT, mapping.old_id, mapping.new_id)::JSONB,
               image_processing_config = REPLACE(image_processing_config::TEXT, mapping.old_id, mapping.new_id)::JSONB,
               vlm_config = REPLACE(vlm_config::TEXT, mapping.old_id, mapping.new_id)::JSONB,
               ocr_config = REPLACE(ocr_config::TEXT, mapping.old_id, mapping.new_id)::JSONB,
               extract_config = CASE
                   WHEN extract_config IS NULL THEN NULL
                   ELSE REPLACE(extract_config::TEXT, mapping.old_id, mapping.new_id)::JSONB
               END,
               faq_config = CASE
                   WHEN faq_config IS NULL THEN NULL
                   ELSE REPLACE(faq_config::TEXT, mapping.old_id, mapping.new_id)::JSONB
               END,
               question_generation_config = CASE
                   WHEN question_generation_config IS NULL THEN NULL
                   ELSE REPLACE(question_generation_config::TEXT, mapping.old_id, mapping.new_id)::JSONB
               END,
               wiki_config = CASE
                   WHEN wiki_config IS NULL THEN NULL
                   ELSE REPLACE(wiki_config::TEXT, mapping.old_id, mapping.new_id)::JSONB
               END,
               updated_at = CURRENT_TIMESTAMP
         WHERE id IN (SELECT id FROM _kb_scope);

        UPDATE knowledges
           SET embedding_model_id = mapping.new_id,
               updated_at = CURRENT_TIMESTAMP
         WHERE id IN (SELECT id FROM _knowledge_scope)
           AND embedding_model_id = mapping.old_id;
    END LOOP;
END
\$\$;

UPDATE tenants source_tenant
   SET storage_used = GREATEST(
           0,
           source_tenant.storage_used - args.moved_storage_bytes
       ),
       updated_at = CURRENT_TIMESTAMP
  FROM _migration_args args
 WHERE source_tenant.id = args.source_tenant_id;

UPDATE tenants target_tenant
   SET storage_used = target_tenant.storage_used + args.moved_storage_bytes,
       updated_at = CURRENT_TIMESTAMP
  FROM _migration_args args
 WHERE target_tenant.id = args.target_tenant_id;

DO \$\$
DECLARE
    args _migration_args%ROWTYPE;
    invalid_count INTEGER;
BEGIN
    SELECT * INTO args FROM _migration_args;

    SELECT COUNT(*) INTO invalid_count
      FROM knowledge_bases
     WHERE id IN (SELECT id FROM _kb_scope)
       AND tenant_id <> args.target_tenant_id;
    IF invalid_count > 0 THEN
        RAISE EXCEPTION '迁移校验失败：仍有 % 个知识库不属于目标企业', invalid_count;
    END IF;

    SELECT COUNT(*) INTO invalid_count
      FROM knowledges
     WHERE id IN (SELECT id FROM _knowledge_scope)
       AND tenant_id <> args.target_tenant_id;
    IF invalid_count > 0 THEN
        RAISE EXCEPTION '迁移校验失败：仍有 % 条知识不属于目标企业', invalid_count;
    END IF;

    SELECT COUNT(*) INTO invalid_count
      FROM chunks
     WHERE knowledge_id IN (SELECT id FROM _knowledge_scope)
       AND tenant_id <> args.target_tenant_id;
    IF invalid_count > 0 THEN
        RAISE EXCEPTION '迁移校验失败：仍有 % 个分块不属于目标企业', invalid_count;
    END IF;
END
\$\$;

\echo ''
\echo '迁移结果预览'
SELECT
    args.source_phone,
    args.source_tenant_id,
    args.target_phone,
    args.target_tenant_id,
    args.moved_storage_bytes,
    COUNT(DISTINCT kb.id) AS knowledge_bases,
    COUNT(DISTINCT k.id) AS knowledges
  FROM _migration_args args
  JOIN _kb_scope scope ON TRUE
  JOIN knowledge_bases kb ON kb.id = scope.id
  LEFT JOIN knowledges k ON k.knowledge_base_id = kb.id
 GROUP BY
    args.source_phone,
    args.source_tenant_id,
    args.target_phone,
    args.target_tenant_id,
    args.moved_storage_bytes;

SELECT
    kb.id,
    kb.name,
    kb.type,
    kb.tenant_id,
    kb.creator_id,
    kb.storage_backend_id,
    kb.vector_store_id,
    COUNT(k.id) AS knowledge_count
  FROM knowledge_bases kb
  LEFT JOIN knowledges k ON k.knowledge_base_id = kb.id
 WHERE kb.id IN (SELECT id FROM _kb_scope)
 GROUP BY kb.id
 ORDER BY kb.name, kb.id;

SELECT 'storage_backend' AS dependency, old_id, new_id FROM _storage_map
UNION ALL
SELECT 'vector_store', old_id, new_id FROM _vector_map
UNION ALL
SELECT 'model', old_id, new_id FROM _model_map
ORDER BY dependency, old_id;

$END_TRANSACTION
SQL

if ((APPLY == 1)); then
  echo
  echo "迁移已提交。启动 app 后请验证知识库文件预览与检索。"
else
  echo
  echo "预演完成，数据库已回滚。确认输出后使用相同参数加 --apply。"
fi
