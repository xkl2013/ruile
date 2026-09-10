#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

command -v docker >/dev/null 2>&1 || {
  printf '[storage-migrate] ERROR: docker is not installed\n' >&2
  exit 1
}

docker compose run --rm --no-deps app /app/WeKnora-storage-migrate "$@"
