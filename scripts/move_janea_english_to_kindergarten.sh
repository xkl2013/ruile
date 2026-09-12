#!/usr/bin/env bash
set -euo pipefail

# One-off production repair:
#   少儿英语 / Janea英语/*  ->  幼儿园 / 英文篇/*
#
# Run from the production deploy directory that contains .env and the
# WeKnora-postgres container.

BASE_URL="${BASE_URL:-https://ai.reallyedu.com}"
TOKEN="${TOKEN:-}"
SOURCE_KB_NAME="${SOURCE_KB_NAME:-少儿英语}"
TARGET_KB_ID="${TARGET_KB_ID:-32fe7427-99a9-4ac0-89ac-999b9af12697}"
SOURCE_PREFIX="${SOURCE_PREFIX:-Janea英语}"
TARGET_PREFIX="${TARGET_PREFIX:-英文篇}"
MODE="${MODE:-reuse_vectors}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-WeKnora-postgres}"

log() { printf '[%s] %s\n' "$(date '+%F %T')" "$*"; }
die() { log "ERROR: $*"; exit 1; }

[[ -n "$TOKEN" ]] || die "TOKEN is required. Run: TOKEN='Bearer ...' ./jane.sh"
[[ -f .env ]] || die ".env not found. Run this from the production deploy directory."

BASE_URL="${BASE_URL%%/api/v1*}"
BASE_URL="${BASE_URL%/}"
TOKEN="${TOKEN#Bearer }"
TOKEN="${TOKEN#bearer }"

set -a
# shellcheck disable=SC1091
source .env
set +a

: "${DB_USER:?DB_USER missing in .env}"
: "${DB_PASSWORD:?DB_PASSWORD missing in .env}"
: "${DB_NAME:?DB_NAME missing in .env}"

docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER" \
  || die "postgres container is not running: $POSTGRES_CONTAINER"

psqlq() {
  docker exec -e PGPASSWORD="$DB_PASSWORD" -i "$POSTGRES_CONTAINER" \
    psql -v ON_ERROR_STOP=1 -U "$DB_USER" -d "$DB_NAME" "$@"
}

json_get() {
  local key="$1"
  tr -d '\n' | sed -n "s/.*\"$key\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p"
}

log "1/8 resolve source knowledge base"
SOURCE_KB_ID="$(psqlq -tAc "
select id
from knowledge_bases
where deleted_at is null
  and name = \$\$${SOURCE_KB_NAME}\$\$
order by created_at desc
limit 1;
" | tr -d '[:space:]')"

[[ -n "$SOURCE_KB_ID" ]] || die "source KB not found: $SOURCE_KB_NAME"
[[ "$SOURCE_KB_ID" != "$TARGET_KB_ID" ]] || die "source and target KB are the same"

TARGET_KB_NAME="$(psqlq -tAc "
select name
from knowledge_bases
where deleted_at is null
  and id = '${TARGET_KB_ID}'
limit 1;
" | tr -d '[:space:]')"

[[ -n "$TARGET_KB_NAME" ]] || die "target KB not found: $TARGET_KB_ID"

log "source: $SOURCE_KB_NAME ($SOURCE_KB_ID)"
log "target: $TARGET_KB_NAME ($TARGET_KB_ID)"
log "path: $SOURCE_PREFIX/* -> $TARGET_PREFIX/*"
log "mode: $MODE"

log "2/8 show source/target compatibility"
psqlq -c "
select name, id, type, embedding_model_id, coalesce(vector_store_id, '') as vector_store_id
from knowledge_bases
where id in ('${SOURCE_KB_ID}', '${TARGET_KB_ID}')
order by name;
"

log "3/8 preview documents to move"
psqlq -c "
select id, file_name, parse_status
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$)
order by file_name;
"

IDS_JSON="$(psqlq -tAc "
select coalesce(json_agg(id), '[]'::json)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status = 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '\n')"

MOVE_COUNT="$(psqlq -tAc "
select count(*)
from knowledges
where deleted_at is null
  and knowledge_base_id = '${SOURCE_KB_ID}'
  and parse_status = 'completed'
  and (file_name = \$\$${SOURCE_PREFIX}\$\$ or file_name like \$\$${SOURCE_PREFIX}/%\$\$);
" | tr -d '[:space:]')"

[[ "$MOVE_COUNT" != "0" && "$IDS_JSON" != "[]" ]] \
  || die "no completed documents found under $SOURCE_PREFIX"

log "completed documents to move: $MOVE_COUNT"
read -r -p "Type YES to execute: " CONFIRM
[[ "$CONFIRM" == "YES" ]] || die "aborted"

log "4/8 submit move task"
MOVE_BODY="$(cat <<JSON
{"source_kb_id":"$SOURCE_KB_ID","target_kb_id":"$TARGET_KB_ID","knowledge_ids":$IDS_JSON,"mode":"$MODE"}
JSON
)"

MOVE_RESP="$(curl -sS -X POST \
  --connect-timeout 10 --max-time 60 \
  "$BASE_URL/api/v1/knowledge/move" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "$MOVE_BODY")"
echo "$MOVE_RESP"

TASK_ID="$(printf '%s' "$MOVE_RESP" | json_get task_id | head -n 1)"
[[ -n "$TASK_ID" ]] || die "move API did not return task_id. If reuse_vectors is rejected, rerun with MODE=reparse"

log "5/8 poll move task: $TASK_ID"
START="$(date +%s)"
while true; do
  PROGRESS_RESP="$(curl -sS \
    --connect-timeout 10 --max-time 60 \
    "$BASE_URL/api/v1/knowledge/move/progress/$TASK_ID" \
    -H "Authorization: Bearer $TOKEN")"
  echo "$PROGRESS_RESP"

  STATUS="$(printf '%s' "$PROGRESS_RESP" | json_get status | head -n 1)"
  [[ "$STATUS" == "completed" ]] && break
  [[ "$STATUS" == "failed" ]] && die "move task failed"

  NOW="$(date +%s)"
  log "status=${STATUS:-unknown}; elapsed=$((NOW - START))s"
  (( NOW - START <= 1800 )) || die "poll timeout"
  sleep 3
done

log "6/8 rewrite display path in target KB"
psqlq <<SQL
BEGIN;
WITH moved(id) AS (
  SELECT jsonb_array_elements_text(\$\$${IDS_JSON}\$\$::jsonb)
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
RETURNING k.id, k.file_name;
COMMIT;
SQL

log "7/8 verify target files"
psqlq -c "
select file_name, parse_status
from knowledges
where deleted_at is null
  and knowledge_base_id = '${TARGET_KB_ID}'
  and (file_name = \$\$${TARGET_PREFIX}\$\$ or file_name like \$\$${TARGET_PREFIX}/%\$\$)
order by file_name;
"

log "8/8 done. Refresh the UI."
