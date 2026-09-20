#!/usr/bin/env bash
# Read-only release checks for a production build.
#
# This script does not build, push, migrate, or restart any service.
#
# Usage:
#   ./scripts/release_preflight.sh
#   ./scripts/release_preflight.sh --tests
#   ./scripts/release_preflight.sh --allow-dirty --tag dirty-20260919

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

ALLOW_DIRTY=false
RUN_TESTS=false
TAG=""

usage() {
  cat <<'EOF'
Usage:
  scripts/release_preflight.sh [options]

Options:
  --tests         Run Go tests and frontend/admin type checks.
  --allow-dirty   Allow uncommitted changes. Only use for a temporary package.
  --tag <tag>     Validate and print the image tag to use.
  -h, --help      Show this help.

The script only checks the local release state. It never pushes images,
changes the database, or restarts services.
EOF
}

log() {
  printf '[release-check] %s\n' "$*"
}

warn() {
  printf '[release-check] WARNING: %s\n' "$*" >&2
}

fail() {
  printf '[release-check] ERROR: %s\n' "$*" >&2
  exit 1
}

require_command() {
  command -v "$1" >/dev/null 2>&1 || fail "missing command: $1"
}

validate_tag() {
  [[ "$1" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]] \
    || fail "invalid image tag: $1"
}

detect_compose() {
  if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1 && docker-compose version >/dev/null 2>&1; then
    COMPOSE=(docker-compose)
  else
    fail "Docker Compose is not available"
  fi
}

run_tests() {
  log "running Go tests"
  go test ./...

  for app in frontend admin; do
    [[ -f "$PROJECT_ROOT/$app/package.json" ]] || continue
    [[ -x "$PROJECT_ROOT/frontend/node_modules/.bin/vue-tsc" ]] \
      || fail "$app dependencies are missing; run npm --prefix frontend ci first"
    log "running $app type check"
    npm --prefix "$app" run type-check
  done
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --allow-dirty)
      ALLOW_DIRTY=true
      shift
      ;;
    --tests)
      RUN_TESTS=true
      shift
      ;;
    --tag)
      [[ $# -ge 2 ]] || fail "--tag requires a value"
      TAG="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      fail "unknown option: $1"
      ;;
  esac
done

cd "$PROJECT_ROOT"
require_command git
require_command docker
require_command npm

if [[ -z "$TAG" ]]; then
  TAG="$(git rev-parse --short HEAD)"
fi
validate_tag "$TAG"

if [[ -n "$(git status --porcelain)" ]]; then
  if [[ "$ALLOW_DIRTY" != true ]]; then
    git status --short
    fail "working tree is dirty; commit the release first or explicitly use --allow-dirty"
  fi
  warn "building from uncommitted changes; tag this package explicitly and do not use it as a rollback target"
fi

log "checking whitespace errors"
git diff --check

log "checking Docker daemon and Compose"
docker info >/dev/null 2>&1 || fail "Docker daemon is not running"
detect_compose

if [[ -f "$PROJECT_ROOT/.env" ]]; then
  log "validating docker compose configuration"
  "${COMPOSE[@]}" config --quiet
else
  warn ".env does not exist; skipped docker compose config validation"
fi

[[ -f "$PROJECT_ROOT/.env.example" ]] || fail ".env.example is missing"
[[ -f "$PROJECT_ROOT/frontend/package-lock.json" ]] || fail "frontend/package-lock.json is missing"

if [[ "$RUN_TESTS" == true ]]; then
  require_command go
  run_tests
fi

log "preflight passed"
printf '\n'
printf 'Recommended immutable image tag: %s\n' "$TAG"
printf 'Build and push:\n'
printf '  ./scripts/build_and_push_acr.sh --tag %s\n' "$TAG"
printf '\n'
printf 'ECS deployment value:\n'
printf '  WEKNORA_VERSION=%s\n' "$TAG"
