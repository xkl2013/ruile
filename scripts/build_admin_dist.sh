#!/usr/bin/env bash
# Build Admin static assets for Docker / release packaging.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

if [ -z "${VITE_FRONTEND_COMMIT:-}" ]; then
	# shellcheck source=/dev/null
	eval "$("$PROJECT_ROOT/scripts/get_version.sh" env)"
	export VITE_FRONTEND_COMMIT="${COMMIT_ID:-unknown}"
fi

export VITE_IS_DOCKER="${VITE_IS_DOCKER:-true}"
export VITE_ADMIN_BASE="${VITE_ADMIN_BASE:-/admin/}"
export VITE_ADMIN_API_BASE_URL="${VITE_ADMIN_API_BASE_URL:-/}"

cd "$PROJECT_ROOT/frontend"
npm ci

if [ ! -e "$PROJECT_ROOT/admin/node_modules" ] && [ ! -L "$PROJECT_ROOT/admin/node_modules" ]; then
	ln -s ../frontend/node_modules "$PROJECT_ROOT/admin/node_modules"
fi

cd "$PROJECT_ROOT/admin"
npm run build
