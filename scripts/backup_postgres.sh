#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
ENV_FILE="${ENV_FILE:-${PROJECT_ROOT}/.env}"

if [[ -f "$ENV_FILE" ]]; then
    set -a
    # shellcheck disable=SC1090
    source "$ENV_FILE"
    set +a
fi

die() {
    printf 'ERROR: %s\n' "$*" >&2
    exit 1
}

DB_DRIVER="${DB_DRIVER:-postgres}"
DB_HOST="${DB_HOST:-postgres}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-}"
DB_PASSWORD="${DB_PASSWORD:-}"
DB_NAME="${DB_NAME:-}"
DB_SSLMODE="${DB_SSLMODE:-disable}"

BACKUP_DIR="${BACKUP_DIR:-${PROJECT_ROOT}/backups}"
BACKUP_PREFIX="${BACKUP_PREFIX:-postgres}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-0}"
POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-WeKnora-postgres}"
BACKUP_MODE="${BACKUP_MODE:-auto}"

[[ "$DB_DRIVER" == "postgres" ]] \
    || die "This script supports PostgreSQL only; current DB_DRIVER=${DB_DRIVER}"
[[ -n "$DB_USER" ]] || die "DB_USER is missing. Configure it in ${ENV_FILE}"
[[ -n "$DB_PASSWORD" ]] || die "DB_PASSWORD is missing. Configure it in ${ENV_FILE}"
[[ -n "$DB_NAME" ]] || die "DB_NAME is missing. Configure it in ${ENV_FILE}"
[[ "$BACKUP_MODE" == "auto" || "$BACKUP_MODE" == "container" || "$BACKUP_MODE" == "host" ]] \
    || die "BACKUP_MODE must be auto, container, or host"
[[ "$BACKUP_RETENTION_DAYS" =~ ^[0-9]+$ ]] \
    || die "BACKUP_RETENTION_DAYS must be a non-negative integer"

mkdir -p "$BACKUP_DIR"
umask 077

container_is_running() {
    command -v docker >/dev/null 2>&1 \
        && docker ps --format '{{.Names}}' | grep -Fxq "$POSTGRES_CONTAINER"
}

select_backup_mode() {
    case "$BACKUP_MODE" in
        container)
            command -v docker >/dev/null 2>&1 \
                || die "docker is required when BACKUP_MODE=container"
            container_is_running \
                || die "PostgreSQL container is not running: ${POSTGRES_CONTAINER}"
            printf '%s\n' "container"
            ;;
        host)
            command -v pg_dump >/dev/null 2>&1 \
                || die "pg_dump is required when BACKUP_MODE=host"
            printf '%s\n' "host"
            ;;
        auto)
            if container_is_running; then
                printf '%s\n' "container"
            else
                command -v pg_dump >/dev/null 2>&1 \
                    || die "No running PostgreSQL container found and pg_dump is not installed"
                printf '%s\n' "host"
            fi
            ;;
    esac
}

run_pg_dump() {
    local mode="$1"

    if [[ "$mode" == "container" ]]; then
        docker exec \
            "$POSTGRES_CONTAINER" \
            pg_dump \
            --username="$DB_USER" \
            --dbname="$DB_NAME" \
            --format=plain \
            --no-owner \
            --no-privileges
        return
    fi

    PGPASSWORD="$DB_PASSWORD" \
        PGSSLMODE="$DB_SSLMODE" \
        pg_dump \
        --host="$DB_HOST" \
        --port="$DB_PORT" \
        --username="$DB_USER" \
        --dbname="$DB_NAME" \
        --format=plain \
        --no-owner \
        --no-privileges
}

backup_mode="$(select_backup_mode)"
timestamp="$(date '+%Y-%m-%d_%H-%M-%S')"
backup_path="${BACKUP_DIR}/${BACKUP_PREFIX}-${timestamp}.sql.gz"
temporary_path="$(mktemp "${BACKUP_DIR}/.${BACKUP_PREFIX}-${timestamp}.XXXXXX")"

cleanup() {
    if [[ -n "${temporary_path:-}" && -e "$temporary_path" ]]; then
        rm -f "$temporary_path"
    fi
}
trap cleanup EXIT

printf 'Backing up PostgreSQL database "%s" via %s...\n' "$DB_NAME" "$backup_mode"
if ! run_pg_dump "$backup_mode" | gzip -c >"$temporary_path"; then
    die "pg_dump failed; no backup was created"
fi

[[ -s "$temporary_path" ]] || die "backup file is empty"
gzip -t "$temporary_path" || die "backup archive validation failed"

mv "$temporary_path" "$backup_path"
temporary_path=""

if (( BACKUP_RETENTION_DAYS > 0 )); then
    deleted_count=0
    while IFS= read -r -d '' old_backup; do
        rm -f "$old_backup"
        deleted_count=$((deleted_count + 1))
    done < <(
        find "$BACKUP_DIR" \
            -type f \
            -name "${BACKUP_PREFIX}-*.sql.gz" \
            -mtime +"$BACKUP_RETENTION_DAYS" \
            -print0
    )
    printf 'Deleted %d backup(s) older than %d day(s).\n' \
        "$deleted_count" "$BACKUP_RETENTION_DAYS"
fi

backup_size="$(wc -c <"$backup_path" | tr -d '[:space:]')"
printf 'Backup completed: %s (%s bytes)\n' "$backup_path" "$backup_size"
