#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEPLOY_DIR="${DEPLOY_DIR:-}"
COMPOSE_FILE="${COMPOSE_FILE:-}"

if [[ -z "$DEPLOY_DIR" ]]; then
  for candidate in "$SCRIPT_DIR" "$PROJECT_ROOT"; do
    if [[ -f "$candidate/docker-compose.yml" ||
      -f "$candidate/docker-compose.yaml" ||
      -f "$candidate/compose.yml" ||
      -f "$candidate/compose.yaml" ]]; then
      DEPLOY_DIR="$candidate"
      break
    fi
  done
fi

if [[ -z "$DEPLOY_DIR" ]]; then
  DEPLOY_DIR="$PROJECT_ROOT"
fi

cd "$DEPLOY_DIR"

command -v docker >/dev/null 2>&1 || {
  printf '[storage-migrate] ERROR: docker is not installed\n' >&2
  exit 1
}

if [[ -z "$COMPOSE_FILE" ]]; then
  for candidate in docker-compose.yml docker-compose.yaml compose.yml compose.yaml; do
    if [[ -f "$candidate" ]]; then
      COMPOSE_FILE="$candidate"
      break
    fi
  done
fi

if [[ -z "$COMPOSE_FILE" || ! -f "$COMPOSE_FILE" ]]; then
  printf '[storage-migrate] ERROR: compose file not found in %s\n' "$DEPLOY_DIR" >&2
  printf '[storage-migrate]        set COMPOSE_FILE=/path/to/docker-compose.yml and retry\n' >&2
  exit 1
fi

printf '[storage-migrate] using compose file: %s\n' "$COMPOSE_FILE"
exec docker compose -f "$COMPOSE_FILE" run --rm --no-deps app /app/WeKnora-storage-migrate "$@"
