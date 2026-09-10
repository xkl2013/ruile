#!/usr/bin/env bash
# Enable Aliyun OSS storage for an existing Docker Compose deployment.

set -euo pipefail

DEPLOY_DIR="${DEPLOY_DIR:-/opt/weknora-image-deploy}"
OSS_ENDPOINT="${OSS_ENDPOINT:-https://oss-cn-beijing.aliyuncs.com}"
OSS_REGION="${OSS_REGION:-cn-beijing}"
OSS_BUCKET_NAME="${OSS_BUCKET_NAME:-rl-knowledge}"
OSS_PATH_PREFIX="${OSS_PATH_PREFIX:-weknora/}"
PULL_APP=false
DRY_RUN=false

DOCKER=(docker)

usage() {
  cat <<EOF
Usage:
  $0 [options]

Options:
  --deploy-dir <dir>  Compose deployment directory. Default: ${DEPLOY_DIR}
  --endpoint <url>    Aliyun OSS endpoint. Default: ${OSS_ENDPOINT}
  --region <region>   Aliyun OSS region. Default: ${OSS_REGION}
  --bucket <bucket>   Aliyun OSS bucket. Default: ${OSS_BUCKET_NAME}
  --prefix <prefix>   Aliyun OSS object prefix. Default: ${OSS_PATH_PREFIX}
  --pull-app          Pull the configured app image before recreating the container.
  --dry-run           Update files in a temp copy and print the resulting non-secret config.
  -h, --help          Show this help.

Environment:
  OSS_ACCESS_KEY      Required unless already present in DEPLOY_DIR/.env.
  OSS_SECRET_KEY      Required unless already present in DEPLOY_DIR/.env.
  OSS_TEMP_BUCKET_NAME Optional temp OSS bucket.
  OSS_TEMP_REGION      Optional temp OSS region.
  WEKNORA_STORAGE_FORCE_ENV_DEFAULT Defaults to true; switches legacy env/local KB bindings to env/oss on startup.

Example:
  OSS_ACCESS_KEY='...' OSS_SECRET_KEY='...' $0 --pull-app
EOF
}

log() {
  printf '[oss-storage] %s\n' "$*"
}

fail() {
  printf '[oss-storage] ERROR: %s\n' "$*" >&2
  exit 1
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --deploy-dir)
      DEPLOY_DIR="${2:-}"
      shift 2
      ;;
    --endpoint)
      OSS_ENDPOINT="${2:-}"
      shift 2
      ;;
    --region)
      OSS_REGION="${2:-}"
      shift 2
      ;;
    --bucket)
      OSS_BUCKET_NAME="${2:-}"
      shift 2
      ;;
    --prefix)
      OSS_PATH_PREFIX="${2:-}"
      shift 2
      ;;
    --pull-app)
      PULL_APP=true
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
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

require_docker() {
  command -v docker >/dev/null 2>&1 || fail "docker is not installed"
  if docker info >/dev/null 2>&1; then
    DOCKER=(docker)
  elif [[ "$(id -u)" -ne 0 ]] && command -v sudo >/dev/null 2>&1 && sudo -n docker info >/dev/null 2>&1; then
    DOCKER=(sudo docker)
  else
    fail "cannot access docker daemon; run as root or with a docker-enabled user"
  fi
  "${DOCKER[@]}" compose version >/dev/null 2>&1 || fail "docker compose plugin is not available"
}

env_file_value() {
  local key="$1"
  [[ -f .env ]] || return 0
  awk -F= -v key="$key" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' .env
}

set_env() {
  local key="$1"
  local value="$2"
  local tmp
  tmp="$(mktemp)"
  if [[ -f .env ]]; then
    grep -v -E "^${key}=" .env >"$tmp" || true
  fi
  printf '%s=%s\n' "$key" "$value" >>"$tmp"
  cat "$tmp" >.env
  rm -f "$tmp"
}

resolve_secret() {
  local key="$1"
  local prompt="$2"
  local value="${!key:-}"
  if [[ -z "$value" ]]; then
    value="$(env_file_value "$key")"
  fi
  if [[ -z "$value" && -t 0 ]]; then
    read -r -s -p "${prompt}: " value
    printf '\n'
  fi
  [[ -n "$value" ]] || fail "$key is required"
  printf '%s' "$value"
}

normalize_prefix() {
  local prefix="$1"
  prefix="${prefix#/}"
  if [[ -n "$prefix" && "$prefix" != */ ]]; then
    prefix="${prefix}/"
  fi
  printf '%s' "$prefix"
}

print_effective_env() {
  awk -F= '
    /^(STORAGE_TYPE|WEKNORA_STORAGE_FORCE_ENV_DEFAULT|OSS_ENDPOINT|OSS_REGION|OSS_BUCKET_NAME|OSS_PATH_PREFIX|OSS_TEMP_BUCKET_NAME|OSS_TEMP_REGION)=/ {
      print $1 "=" $2
    }
    /^(OSS_ACCESS_KEY|OSS_SECRET_KEY)=/ {
      if ($2 != "") print $1 "=set"; else print $1 "=missing"
    }
  ' .env
}

print_compose_storage_env() {
  local compose_file="$1"
  awk '
    /STORAGE_TYPE=|WEKNORA_STORAGE_FORCE_ENV_DEFAULT=|OSS_ENDPOINT=|OSS_REGION=|OSS_ACCESS_KEY=|OSS_SECRET_KEY=|OSS_BUCKET_NAME=|OSS_PATH_PREFIX=|OSS_TEMP_BUCKET_NAME=|OSS_TEMP_REGION=/ {
      line = $0
      sub(/OSS_ACCESS_KEY=.*/, "OSS_ACCESS_KEY=***", line)
      sub(/OSS_SECRET_KEY=.*/, "OSS_SECRET_KEY=***", line)
      print line
    }
  ' "$compose_file"
}

patch_compose_file() {
  local compose_file=""
  local candidate
  for candidate in docker-compose.yml compose.yml compose.yaml; do
    if [[ -f "$candidate" ]]; then
      compose_file="$candidate"
      break
    fi
  done
  [[ -n "$compose_file" ]] || fail "no compose file found in $DEPLOY_DIR"

  local backup
  backup="${compose_file}.bak.$(date +%F-%H%M%S)"
  cp "$compose_file" "$backup"
  log "backed up $compose_file to $DEPLOY_DIR/$backup"

  local has_force_env has_oss_endpoint has_oss_region has_oss_access has_oss_secret
  local has_oss_bucket has_oss_prefix has_oss_temp_bucket has_oss_temp_region
  grep -q 'WEKNORA_STORAGE_FORCE_ENV_DEFAULT=' "$compose_file" && has_force_env=true || has_force_env=false
  grep -q 'OSS_ENDPOINT=' "$compose_file" && has_oss_endpoint=true || has_oss_endpoint=false
  grep -q 'OSS_REGION=' "$compose_file" && has_oss_region=true || has_oss_region=false
  grep -q 'OSS_ACCESS_KEY=' "$compose_file" && has_oss_access=true || has_oss_access=false
  grep -q 'OSS_SECRET_KEY=' "$compose_file" && has_oss_secret=true || has_oss_secret=false
  grep -q 'OSS_BUCKET_NAME=' "$compose_file" && has_oss_bucket=true || has_oss_bucket=false
  grep -q 'OSS_PATH_PREFIX=' "$compose_file" && has_oss_prefix=true || has_oss_prefix=false
  grep -q 'OSS_TEMP_BUCKET_NAME=' "$compose_file" && has_oss_temp_bucket=true || has_oss_temp_bucket=false
  grep -q 'OSS_TEMP_REGION=' "$compose_file" && has_oss_temp_region=true || has_oss_temp_region=false

  local tmp
  tmp="$(mktemp)"
  awk \
    -v has_force_env="$has_force_env" \
    -v has_oss_endpoint="$has_oss_endpoint" \
    -v has_oss_region="$has_oss_region" \
    -v has_oss_access="$has_oss_access" \
    -v has_oss_secret="$has_oss_secret" \
    -v has_oss_bucket="$has_oss_bucket" \
    -v has_oss_prefix="$has_oss_prefix" \
    -v has_oss_temp_bucket="$has_oss_temp_bucket" \
    -v has_oss_temp_region="$has_oss_temp_region" '
    {
      if ($0 ~ /^[[:space:]]*-[[:space:]]*STORAGE_TYPE=\$\{STORAGE_TYPE:-[^}]*\}/) {
        sub(/\$\{STORAGE_TYPE:-[^}]*\}/, "${STORAGE_TYPE:-oss}")
      }
      print
      if (!inserted && $0 ~ /^[[:space:]]*-[[:space:]]*LOCAL_STORAGE_BASE_DIR=/) {
        if (has_force_env != "true") print "      - WEKNORA_STORAGE_FORCE_ENV_DEFAULT=${WEKNORA_STORAGE_FORCE_ENV_DEFAULT:-true}"
        if (has_oss_endpoint != "true") print "      - OSS_ENDPOINT=${OSS_ENDPOINT:-}"
        if (has_oss_region != "true") print "      - OSS_REGION=${OSS_REGION:-}"
        if (has_oss_access != "true") print "      - OSS_ACCESS_KEY=${OSS_ACCESS_KEY:-}"
        if (has_oss_secret != "true") print "      - OSS_SECRET_KEY=${OSS_SECRET_KEY:-}"
        if (has_oss_bucket != "true") print "      - OSS_BUCKET_NAME=${OSS_BUCKET_NAME:-}"
        if (has_oss_prefix != "true") print "      - OSS_PATH_PREFIX=${OSS_PATH_PREFIX:-weknora/}"
        if (has_oss_temp_bucket != "true") print "      - OSS_TEMP_BUCKET_NAME=${OSS_TEMP_BUCKET_NAME:-}"
        if (has_oss_temp_region != "true") print "      - OSS_TEMP_REGION=${OSS_TEMP_REGION:-}"
        inserted = 1
      }
    }
  ' "$compose_file" >"$tmp"
  cat "$tmp" >"$compose_file"
  rm -f "$tmp"
  log "effective compose storage environment:"
  print_compose_storage_env "$compose_file"
}

wait_for_app() {
  local status
  for _ in $(seq 1 45); do
    status="$("${DOCKER[@]}" inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' WeKnora-app 2>/dev/null || true)"
    if [[ "$status" == "healthy" || "$status" == "running" ]]; then
      return 0
    fi
    sleep 2
  done
  "${DOCKER[@]}" logs --tail=100 WeKnora-app || true
  fail "WeKnora-app did not become healthy/running in time"
}

[[ -n "$DEPLOY_DIR" ]] || fail "--deploy-dir is required"
[[ -n "$OSS_ENDPOINT" ]] || fail "--endpoint is required"
[[ -n "$OSS_REGION" ]] || fail "--region is required"
[[ -n "$OSS_BUCKET_NAME" ]] || fail "--bucket is required"

if [[ "$DRY_RUN" == true ]]; then
  tmp_dir="$(mktemp -d)"
  if [[ -d "$DEPLOY_DIR" ]]; then
    cp -R "$DEPLOY_DIR"/. "$tmp_dir"/
  fi
  DEPLOY_DIR="$tmp_dir"
  log "dry run uses temp deploy dir: $DEPLOY_DIR"
else
  [[ -d "$DEPLOY_DIR" ]] || fail "deployment directory does not exist: $DEPLOY_DIR"
  require_docker
fi

cd "$DEPLOY_DIR"
[[ -f docker-compose.yml || -f compose.yml || -f compose.yaml ]] || fail "no compose file found in $DEPLOY_DIR"
touch .env
chmod 600 .env

backup=".env.bak.$(date +%F-%H%M%S)"
cp .env "$backup"
log "backed up .env to $DEPLOY_DIR/$backup"

OSS_ACCESS_KEY_VALUE="$(resolve_secret OSS_ACCESS_KEY "OSS access key")"
OSS_SECRET_KEY_VALUE="$(resolve_secret OSS_SECRET_KEY "OSS secret key")"
OSS_PATH_PREFIX="$(normalize_prefix "$OSS_PATH_PREFIX")"

set_env STORAGE_TYPE "oss"
set_env WEKNORA_STORAGE_FORCE_ENV_DEFAULT "${WEKNORA_STORAGE_FORCE_ENV_DEFAULT:-true}"
set_env OSS_ENDPOINT "$OSS_ENDPOINT"
set_env OSS_REGION "$OSS_REGION"
set_env OSS_ACCESS_KEY "$OSS_ACCESS_KEY_VALUE"
set_env OSS_SECRET_KEY "$OSS_SECRET_KEY_VALUE"
set_env OSS_BUCKET_NAME "$OSS_BUCKET_NAME"
set_env OSS_PATH_PREFIX "$OSS_PATH_PREFIX"
set_env OSS_TEMP_BUCKET_NAME "${OSS_TEMP_BUCKET_NAME:-$(env_file_value OSS_TEMP_BUCKET_NAME)}"
set_env OSS_TEMP_REGION "${OSS_TEMP_REGION:-$(env_file_value OSS_TEMP_REGION)}"

log "effective OSS config:"
print_effective_env

patch_compose_file

if [[ "$DRY_RUN" == true ]]; then
  log "dry run complete"
  exit 0
fi

"${DOCKER[@]}" compose config >/dev/null

if [[ "$PULL_APP" == true ]]; then
  log "pulling app image"
  "${DOCKER[@]}" compose pull app
fi

log "recreating app container"
"${DOCKER[@]}" compose up -d --force-recreate app
wait_for_app

log "app container environment:"
"${DOCKER[@]}" exec WeKnora-app sh -lc '
echo STORAGE_TYPE=$STORAGE_TYPE
echo WEKNORA_STORAGE_FORCE_ENV_DEFAULT=$WEKNORA_STORAGE_FORCE_ENV_DEFAULT
echo OSS_ENDPOINT=$OSS_ENDPOINT
echo OSS_REGION=$OSS_REGION
echo OSS_BUCKET_NAME=$OSS_BUCKET_NAME
echo OSS_PATH_PREFIX=$OSS_PATH_PREFIX
[ -n "$OSS_ACCESS_KEY" ] && echo OSS_ACCESS_KEY=set || echo OSS_ACCESS_KEY=missing
[ -n "$OSS_SECRET_KEY" ] && echo OSS_SECRET_KEY=set || echo OSS_SECRET_KEY=missing
'

log "done; new uploads will use OSS when the selected workspace/knowledge-base storage backend resolves to the env OSS backend"
