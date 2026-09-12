#!/usr/bin/env bash
set -euo pipefail

# One-off cleanup for the documents left after the main migration:
#   少儿英语 / Janea英语/*  ->  幼儿园 / 英文篇/*
#
# Why this script exists:
# The official /knowledge/move API only accepts parse_status=completed.
# The first migration intentionally skipped failed/cancelled/draft records, so
# the source KB can still show folder counts. This script finishes the cleanup:
#   1) completed leftovers still use the official move API;
#   2) non-completed terminal records are reassigned in DB and optionally
#      submitted for reparse in the target KB.

BASE_URL="${BASE_URL:-https://ai.reallyedu.com}"
TOKEN="${TOKEN:-}"

SOURCE_KB_ID="${SOURCE_KB_ID:-7fc65db0-21c5-469e-bae6-85db99d00535}" # 少儿英语
TARGET_KB_ID="${TARGET_KB_ID:-32fe7427-99a9-4ac0-89ac-999b9af12697}" # 幼儿园
SOURCE_PREFIX="${SOURCE_PREFIX:-Janea英语}"
TARGET_PREFIX="${TARGET_PREFIX:-英文篇}"

MODE="${MODE:-reuse_vectors}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-WeKnora-postgres}"
POLL_INTERVAL_SECONDS="${POLL_INTERVAL_SECONDS:-3}"
POLL_TIMEOUT_SECONDS="${POLL_TIMEOUT_SECONDS:-1800}"
REPARSE_NON_COMPLETED="${REPARSE_NON_COMPLETED:-1}"
CONFIRM="${CONFIRM:-}"
FORCE_ACTIVE="${FORCE_ACTIVE:-0}"

log() { printf '[%s] %s\n' "$(date '+%F %T')" "$*"; }
die() { log "ERROR: $*"; exit 1; }

json_get() {
  local path="$1"
  python3 -c '
import json
import sys

path = sys.argv[1].split(".")
try:
    value = json.load(sys.stdin)
    for key in path:
        if not isinstance(value, dict):
            value = None
            break
        value = value.get(key)
    if isinstance(value, (str, int, float)):
        print(value)
except Exception:
    pass
' "$path"
}

api_post_json() {
  local url="$1"
  local body="$2"
  curl -sS -X POST \
    --connect-timeout 10 \
    --max-time 60 \
    "$url" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d "$body"
}

api_get() {
  local url="$1"
  curl -sS \
    --connect-timeout 10 \
    --max-time 60 \
    "$url" \
    -H "Authorization: Bearer $TOKEN"
}

psqlq() {
  docker exec -e PGPASSWORD="$DB_PASSWORD" -i "$POSTGRES_CONTAINER" \
    psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" "$@"
}

require_token() {
  [[ -n "$TOKEN" ]] || die "TOKEN is required. Example: TOKEN='Bearer ...' CONFIRM=YES bash $0"
}

BASE_URL="${BASE_URL%%/api/v1*}"
BASE_URL="${BASE_URL%/}"
TOKEN="${TOKEN#Bearer }"
TOKEN="${TOKEN#bearer }"

command -v python3 >/dev/null 2>&1 || die "python3 is required for JSON parsing"
[[ -f .env ]] || die ".env not found. Run this script from the production deploy directory."

set -a
# shellcheck disable=SC1091
source .env
set +a

: "${DB_USER:?DB_USER missing in .env}"
: "${DB_PASSWORD:?DB_PASSWORD missing in .env}"
: "${DB_NAME:?DB_NAME missing in .env}"

docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER" \
  || die "postgres container is not running: $POSTGRES_CONTAINER"

[[ "$SOURCE_KB_ID" != "$TARGET_KB_ID" ]] || die "source and target KB IDs are the same"

log "1/8 check knowledge bases"
psqlq -c "
select name, id, tenant_id, type, embedding_model_id, coalesce(vector_store_id, '') as vector_store_id
from knowledge_bases
where deleted_at is null
  and id in ('${SOURCE_KB_ID}', '${TARGET_KB_ID}')
order by name;
"

FOUND_KB_COUNT="$(psqlq -tAc "
select count(*)
from knowledge_bases
where deleted_at is null
  and id in ('${SOURCE_KB_ID}', '${TARGET_KB_ID}');
" | tr -d '[:space:]')"
[[ "$FOUND_KB_COUNT" == "2" ]] || die "source/target KB not found. Check SOURCE_KB_ID and TARGET_KB_ID"

log "2/8 preview remaining source records"
psqlq -c "
select parse_status, count(*) as docs
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$)
group by parse_status
order by docs desc, parse_status;
"

psqlq -c "
select id, parse_status, file_name
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$)
order by parse_status, file_name;
"

REMAINING_COUNT="$(psqlq -tAc "
select count(*)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '[:space:]')"

if [[ "${REMAINING_COUNT:-0}" == "0" ]]; then
  log "nothing left under ${SOURCE_PREFIX}; done"
  exit 0
fi

ACTIVE_COUNT="$(psqlq -tAc "
select count(*)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status in ('pending', 'processing', 'finalizing', 'deleting')
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '[:space:]')"

if [[ "${ACTIVE_COUNT:-0}" != "0" && "$FORCE_ACTIVE" != "1" ]]; then
  log "active records found; wait for them to finish or rerun with FORCE_ACTIVE=1 if you accept the risk"
  psqlq -c "
select id, parse_status, file_name
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status in ('pending', 'processing', 'finalizing', 'deleting')
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$)
order by file_name;
"
  exit 1
fi

COMPLETED_IDS_JSON="$(psqlq -tAc "
select coalesce(json_agg(id order by file_name), '[]'::json)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status = 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '\n')"

COMPLETED_COUNT="$(psqlq -tAc "
select count(*)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status = 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '[:space:]')"

NON_COMPLETED_IDS_JSON="$(psqlq -tAc "
select coalesce(json_agg(id order by file_name), '[]'::json)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status <> 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '\n')"

NON_COMPLETED_COUNT="$(psqlq -tAc "
select count(*)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status <> 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '[:space:]')"

log "remaining total: $REMAINING_COUNT"
log "completed leftovers for official move API: $COMPLETED_COUNT"
log "non-completed leftovers for DB reassignment: $NON_COMPLETED_COUNT"

if [[ "$CONFIRM" != "YES" ]]; then
  read -r -p "Type YES to move the remaining records: " CONFIRM
fi
[[ "$CONFIRM" == "YES" ]] || die "aborted"

if [[ "${COMPLETED_COUNT:-0}" != "0" ]]; then
  require_token
  log "3/8 move completed leftovers via official API"
  MOVE_BODY="$(cat <<JSON
{"source_kb_id":"$SOURCE_KB_ID","target_kb_id":"$TARGET_KB_ID","knowledge_ids":$COMPLETED_IDS_JSON,"mode":"$MODE"}
JSON
)"

  MOVE_RESP="$(api_post_json "$BASE_URL/api/v1/knowledge/move" "$MOVE_BODY")"
  echo "$MOVE_RESP"

  TASK_ID="$(printf '%s' "$MOVE_RESP" | json_get "data.task_id" | head -n 1)"
  [[ -n "$TASK_ID" ]] || die "move API did not return task_id. Response: $MOVE_RESP"

  log "4/8 poll move task: $TASK_ID"
  START="$(date +%s)"
  while true; do
    PROGRESS_RESP="$(api_get "$BASE_URL/api/v1/knowledge/move/progress/$TASK_ID")"
    echo "$PROGRESS_RESP"

    STATUS="$(printf '%s' "$PROGRESS_RESP" | json_get "data.status" | head -n 1)"
    FAILED="$(printf '%s' "$PROGRESS_RESP" | json_get "data.failed" | head -n 1)"
    [[ -z "$FAILED" ]] && FAILED="0"

    [[ "$STATUS" == "completed" ]] && break
    [[ "$STATUS" == "failed" ]] && die "move task failed"

    NOW="$(date +%s)"
    log "status=${STATUS:-unknown}; elapsed=$((NOW - START))s"
    (( NOW - START <= POLL_TIMEOUT_SECONDS )) || die "poll timeout"
    sleep "$POLL_INTERVAL_SECONDS"
  done

  [[ "$FAILED" == "0" ]] || die "official move task completed with failed=$FAILED; stop before DB cleanup"

  log "5/8 rewrite moved completed paths in target KB"
  psqlq <<SQL
BEGIN;
WITH moved(id) AS (
  SELECT jsonb_array_elements_text(\$json\$${COMPLETED_IDS_JSON}\$json\$::jsonb)
),
params AS (
  SELECT
    '${TARGET_KB_ID}'::varchar AS target_kb_id,
    \$\$${SOURCE_PREFIX}\$\$::text AS old_prefix,
    \$\$${TARGET_PREFIX}\$\$::text AS new_prefix
)
UPDATE knowledges k
SET file_name = p.new_prefix || substring(k.file_name from char_length(p.old_prefix) + 1),
    updated_at = now()
FROM moved m, params p
WHERE k.id = m.id
  AND k.deleted_at IS NULL
  AND k.knowledge_base_id = p.target_kb_id
  AND (k.file_name = p.old_prefix OR k.file_name LIKE p.old_prefix || '/%')
RETURNING k.id, k.parse_status, k.file_name;
COMMIT;
SQL
else
  log "3/8 no completed leftovers"
  log "4/8 skip official move poll"
  log "5/8 skip completed path rewrite"
fi

if [[ "${NON_COMPLETED_COUNT:-0}" != "0" ]]; then
  log "6/8 move non-completed terminal records by DB reassignment"
  psqlq <<SQL
BEGIN;
WITH moved AS (
  SELECT id, tenant_id
  FROM knowledges
  WHERE deleted_at IS NULL
    AND knowledge_base_id = '${SOURCE_KB_ID}'
    AND parse_status <> 'completed'
    AND (file_name = \$\$${SOURCE_PREFIX}\$\$ OR file_name LIKE \$\$${SOURCE_PREFIX}/%\$\$)
),
clear_tags AS (
  DELETE FROM knowledge_tag_relations ktr
  USING moved m
  WHERE ktr.knowledge_id = m.id
  RETURNING ktr.knowledge_id
),
move_chunks AS (
  UPDATE chunks c
  SET knowledge_base_id = '${TARGET_KB_ID}',
      updated_at = now()
  FROM moved m
  WHERE c.tenant_id = m.tenant_id
    AND c.knowledge_id = m.id
    AND c.deleted_at IS NULL
  RETURNING c.knowledge_id
),
params AS (
  SELECT
    '${TARGET_KB_ID}'::varchar AS target_kb_id,
    \$\$${SOURCE_PREFIX}\$\$::text AS old_prefix,
    \$\$${TARGET_PREFIX}\$\$::text AS new_prefix
)
UPDATE knowledges k
SET knowledge_base_id = p.target_kb_id,
    file_name = p.new_prefix || substring(k.file_name from char_length(p.old_prefix) + 1),
    updated_at = now()
FROM moved m, params p
WHERE k.id = m.id
RETURNING k.id, k.parse_status, k.file_name;
COMMIT;
SQL

  if [[ "$REPARSE_NON_COMPLETED" == "1" ]]; then
    require_token
    log "7/8 submit batch reparse for moved non-completed records"
    REPARSE_BODY="$(cat <<JSON
{"kb_id":"$TARGET_KB_ID","ids":$NON_COMPLETED_IDS_JSON}
JSON
)"
    REPARSE_RESP="$(api_post_json "$BASE_URL/api/v1/knowledge/batch-reparse" "$REPARSE_BODY")"
    echo "$REPARSE_RESP"
  else
    log "7/8 skip reparse because REPARSE_NON_COMPLETED=$REPARSE_NON_COMPLETED"
  fi
else
  log "6/8 no non-completed leftovers"
  log "7/8 skip reparse"
fi

log "8/8 final verification"
log "source remaining under ${SOURCE_PREFIX}:"
psqlq -c "
select parse_status, count(*) as docs
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$)
group by parse_status
order by docs desc, parse_status;
"

log "target records under ${TARGET_PREFIX}:"
psqlq -c "
select parse_status, count(*) as docs
from knowledges
where deleted_at is null
  and knowledge_base_id = '${TARGET_KB_ID}'
  and (file_name = \$\$${TARGET_PREFIX}\$\$ or file_name like \$\$${TARGET_PREFIX}/%\$\$)
group by parse_status
order by docs desc, parse_status;
"

log "done. Refresh the UI."
