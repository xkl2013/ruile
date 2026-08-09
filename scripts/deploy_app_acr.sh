#!/usr/bin/env bash
# Build and push only the WeKnora app image to Alibaba Cloud ACR.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

ACR_ENV_FILE="${ACR_ENV_FILE:-$PROJECT_ROOT/.acr.env}"
if [[ -f "$ACR_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ACR_ENV_FILE"
  set +a
fi

ACR_REGISTRY="${ACR_REGISTRY:-registry.cn-beijing.aliyuncs.com}"
ACR_NAMESPACE="${ACR_NAMESPACE:-rl-knowledge}"
ACR_USERNAME="${ACR_USERNAME:-睿乐未来}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
ACR_BUILD_PLATFORM="${ACR_BUILD_PLATFORM:-linux/amd64}"
SKIP_BUILD=false
SKIP_LOGIN=false

usage() {
  cat <<EOF
Usage:
  $0 [options]

Options:
  --registry <registry>    ACR registry. Default: ${ACR_REGISTRY}
  --namespace <namespace>  ACR namespace. Default: ${ACR_NAMESPACE}
  --username <username>    ACR username. Default: ACR_USERNAME or configured default.
  --tag <tag>              Image tag. Default: ${IMAGE_TAG}
  --platform <platform>    Docker target platform. Default: ${ACR_BUILD_PLATFORM}
  --skip-build             Only tag/push an existing wechatopenai/weknora-app:latest.
  --skip-login             Do not run docker login.
  -h, --help               Show this help.

Environment:
  ACR_ENV_FILE             Env file to load. Default: .acr.env in project root.
  ACR_PASSWORD             ACR password. If unset, existing docker credentials are used.
  GO_BASE                  Optional Go builder image override.
  DEBIAN_BASE              Optional Debian runtime image override.
  APK_MIRROR_ARG           Debian apt mirror host. Default: mirrors.aliyun.com.
  GOPROXY                  Go proxy. Default: https://goproxy.cn,direct.
  GOSUMDB                  Go checksum DB. Default: off.

Image pushed:
  ${ACR_REGISTRY}/${ACR_NAMESPACE}/weknora-app:${IMAGE_TAG}
EOF
}

log() {
  printf '[deploy-app] %s\n' "$*"
}

fail() {
  printf '[deploy-app] ERROR: %s\n' "$*" >&2
  exit 1
}

target_arch() {
  case "$1" in
    linux/amd64) printf 'amd64' ;;
    linux/arm64) printf 'arm64' ;;
    *) printf '%s' "$1" | awk -F/ '{print $2}' ;;
  esac
}

builder_platform() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'linux/amd64' ;;
    aarch64|arm64) printf 'linux/arm64' ;;
    *) printf 'linux/amd64' ;;
  esac
}

platform_arch() {
  case "$1" in
    linux/amd64) printf 'amd64' ;;
    linux/arm64) printf 'arm64' ;;
    *) printf '%s' "$1" | awk -F/ '{print $2}' ;;
  esac
}

maybe_login() {
  if [[ "$SKIP_LOGIN" == true ]]; then
    log "skipping docker login"
    return 0
  fi

  if [[ -z "${ACR_PASSWORD:-}" ]]; then
    if [[ -t 0 ]]; then
      read -r -s -p "ACR password for ${ACR_USERNAME}@${ACR_REGISTRY}: " ACR_PASSWORD
      printf '\n'
    else
      log "ACR_PASSWORD is not set; using existing docker credentials"
      return 0
    fi
  fi

  log "logging in to ${ACR_REGISTRY}"
  printf '%s' "$ACR_PASSWORD" | docker login "$ACR_REGISTRY" --username "$ACR_USERNAME" --password-stdin
}

pick_go_base() {
  if [[ -n "${GO_BASE:-}" ]]; then
    return 0
  fi

  local build_arch
  build_arch="$(platform_arch "$(builder_platform)")"

  if docker image inspect "golang:1.26-bookworm-${build_arch}" >/dev/null 2>&1; then
    export GO_BASE="golang:1.26-bookworm-${build_arch}"
  elif docker image inspect "${ACR_REGISTRY}/${ACR_NAMESPACE}/base-golang:1.26-bookworm-${build_arch}" >/dev/null 2>&1; then
    export GO_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-golang:1.26-bookworm-${build_arch}"
  else
    export GO_BASE="golang:1.26-bookworm"
  fi
}

prepare_debian_base() {
  if [[ -n "${DEBIAN_BASE:-}" ]]; then
    return 0
  fi

  if [[ "${APP_USE_ALIYUN_DEBIAN_BASE:-true}" != "true" ]]; then
    export DEBIAN_BASE="debian:12.12-slim"
    return 0
  fi

  local arch
  local image
  arch="$(target_arch "$ACR_BUILD_PLATFORM")"
  image="weknora-local-base/debian:12.12-slim-aliyun-${arch}"
  if ! docker image inspect "$image" >/dev/null 2>&1; then
    log "creating local Debian base with Aliyun apt mirror: ${image}"
    docker build --progress=plain --platform "$ACR_BUILD_PLATFORM" -t "$image" - <<'EOF'
ARG DEBIAN_BASE=debian:12.12-slim
FROM ${DEBIAN_BASE}
RUN sed -i 's@deb.debian.org@mirrors.aliyun.com@g' /etc/apt/sources.list.d/debian.sources
EOF
  fi

  export DEBIAN_BASE="$image"
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --registry)
      ACR_REGISTRY="${2:-}"
      shift 2
      ;;
    --namespace)
      ACR_NAMESPACE="${2:-}"
      shift 2
      ;;
    --username)
      ACR_USERNAME="${2:-}"
      shift 2
      ;;
    --tag)
      IMAGE_TAG="${2:-}"
      shift 2
      ;;
    --platform)
      ACR_BUILD_PLATFORM="${2:-}"
      shift 2
      ;;
    --skip-build)
      SKIP_BUILD=true
      shift
      ;;
    --skip-login)
      SKIP_LOGIN=true
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

command -v docker >/dev/null 2>&1 || fail "docker is not installed"
docker info >/dev/null 2>&1 || fail "docker daemon is not running"
[[ -n "$ACR_REGISTRY" ]] || fail "--registry is required"
[[ -n "$ACR_NAMESPACE" ]] || fail "--namespace is required"

maybe_login

if [[ "$SKIP_BUILD" != true ]]; then
  pick_go_base
  prepare_debian_base

  export PLATFORM="$ACR_BUILD_PLATFORM"
  export TARGETARCH
  TARGETARCH="$(target_arch "$ACR_BUILD_PLATFORM")"
  export APK_MIRROR_ARG="${APK_MIRROR_ARG:-mirrors.aliyun.com}"
  export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
  export GOSUMDB="${GOSUMDB:-off}"

  log "building app image for ${PLATFORM}"
  "$PROJECT_ROOT/scripts/build_images.sh" --app
fi

local_image="wechatopenai/weknora-app:latest"
remote_image="${ACR_REGISTRY}/${ACR_NAMESPACE}/weknora-app:${IMAGE_TAG}"

log "tagging ${local_image} -> ${remote_image}"
docker tag "$local_image" "$remote_image"

log "pushing ${remote_image}"
docker push "$remote_image"

log "done: ${remote_image}"
