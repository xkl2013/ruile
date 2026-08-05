#!/usr/bin/env bash
# Build WeKnora Docker images and push them to Alibaba Cloud ACR.
#
# Usage:
#   ./scripts/build_and_push_acr.sh --namespace your-namespace
#
# Optional:
#   ACR_USERNAME='your-username' ./scripts/build_and_push_acr.sh --namespace your-namespace --tag v1

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

ACR_REGISTRY="${ACR_REGISTRY:-registry.cn-beijing.aliyuncs.com}"
ACR_USERNAME="${ACR_USERNAME:-睿乐未来}"
ACR_NAMESPACE="${ACR_NAMESPACE:-}"
IMAGE_TAG="${IMAGE_TAG:-}"
SKIP_BUILD=false
SKIP_LOGIN=false
ACR_ENV_FILE="${ACR_ENV_FILE:-$PROJECT_ROOT/.acr.env}"

if [[ -f "$ACR_ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ACR_ENV_FILE"
  set +a
fi

ACR_REGISTRY="${ACR_REGISTRY:-registry.cn-beijing.aliyuncs.com}"
ACR_USERNAME="${ACR_USERNAME:-睿乐未来}"
ACR_NAMESPACE="${ACR_NAMESPACE:-}"
IMAGE_TAG="${IMAGE_TAG:-}"
ACR_BUILD_PLATFORM="${ACR_BUILD_PLATFORM:-linux/amd64}"
ACR_BUILDER_PLATFORM="${ACR_BUILDER_PLATFORM:-}"
USE_ACR_BASE_IMAGES="${USE_ACR_BASE_IMAGES:-true}"

IMAGES=(
  "weknora-app"
  "weknora-docreader"
  "weknora-ui"
  "weknora-sandbox"
)

usage() {
  cat <<EOF
Usage:
  $0 --namespace <acr-namespace> [options]

Options:
  --registry <registry>    ACR registry. Default: ${ACR_REGISTRY}
  --namespace <namespace>  ACR namespace. Required unless ACR_NAMESPACE is set.
  --username <username>    ACR username. Default: ACR_USERNAME or configured default.
  --tag <tag>              Image tag. Default: git short SHA.
  --skip-build             Skip local image build and only tag/push existing images.
  --skip-login             Skip docker login.
  -h, --help               Show this help.

Environment:
  ACR_ENV_FILE             Env file to load. Default: .acr.env in project root.
  ACR_PASSWORD             ACR password. If omitted, the script prompts securely.
  ACR_REGISTRY             Same as --registry.
  ACR_NAMESPACE            Same as --namespace.
  ACR_USERNAME             Same as --username.
  IMAGE_TAG                Same as --tag.
  ACR_BUILD_PLATFORM        Docker target platform. Default: linux/amd64.
  ACR_BUILDER_PLATFORM      Native builder platform. Default: detected from uname.
  USE_ACR_BASE_IMAGES       Push local base images to ACR and build from ACR. Default: true.

Images pushed:
  ${ACR_REGISTRY}/<namespace>/weknora-app:<tag>
  ${ACR_REGISTRY}/<namespace>/weknora-docreader:<tag>
  ${ACR_REGISTRY}/<namespace>/weknora-ui:<tag>
  ${ACR_REGISTRY}/<namespace>/weknora-sandbox:<tag>
EOF
}

log() {
  printf '[acr] %s\n' "$*"
}

fail() {
  printf '[acr] ERROR: %s\n' "$*" >&2
  exit 1
}

target_arch() {
  case "$ACR_BUILD_PLATFORM" in
    linux/amd64) printf 'amd64' ;;
    linux/arm64) printf 'arm64' ;;
    *) printf '%s' "$ACR_BUILD_PLATFORM" | awk -F/ '{print $2}' ;;
  esac
}

builder_platform() {
  if [[ -n "$ACR_BUILDER_PLATFORM" ]]; then
    printf '%s' "$ACR_BUILDER_PLATFORM"
    return 0
  fi

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

first_existing_image() {
  local image
  for image in "$@"; do
    if docker image inspect "$image" >/dev/null 2>&1; then
      printf '%s' "$image"
      return 0
    fi
  done
  return 1
}

tag_and_push_base_image() {
  local source_image="$1"
  local remote_image="$2"

  log "tagging base ${source_image} -> ${remote_image}"
  docker tag "$source_image" "$remote_image"

  log "pushing base ${remote_image}"
  docker push "$remote_image"
}

prepare_acr_base_images() {
  local arch
  local build_arch
  local go_source
  local debian_source
  local docreader_python_source
  local nginx_source
  local sandbox_node_source
  local sandbox_python_source

  arch="$(target_arch)"
  build_arch="$(platform_arch "$(builder_platform)")"

  go_source="$(first_existing_image "golang:1.26-bookworm-${build_arch}" "golang:1.26-bookworm")" \
    || fail "missing local builder base image: golang:1.26-bookworm-${build_arch}"
  debian_source="$(first_existing_image "debian:12.12-slim" "debian:12.12-slim-${arch}")" \
    || fail "missing local base image: debian:12.12-slim"
  docreader_python_source="$(first_existing_image "python:3.10.18-bookworm" "python:3.10.18-bookworm-${arch}")" \
    || fail "missing local base image: python:3.10.18-bookworm"
  nginx_source="$(first_existing_image "nginx:stable-alpine" "nginx:stable-alpine-${arch}")" \
    || fail "missing local base image: nginx:stable-alpine"
  sandbox_node_source="$(first_existing_image "node:20-slim" "node:20-slim-${arch}")" \
    || fail "missing local base image: node:20-slim"
  sandbox_python_source="$(first_existing_image "python:3.11-slim" "python:3.11-slim-${arch}")" \
    || fail "missing local base image: python:3.11-slim"

  export GO_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-golang:1.26-bookworm-${build_arch}"
  export DEBIAN_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-debian:12.12-slim"
  export DOCREADER_PYTHON_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-python:3.10.18-bookworm"
  export NGINX_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-nginx:stable-alpine"
  export SANDBOX_NODE_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-node:20-slim"
  export SANDBOX_PYTHON_BASE="${ACR_REGISTRY}/${ACR_NAMESPACE}/base-python:3.11-slim"

  tag_and_push_base_image "$go_source" "$GO_BASE"
  tag_and_push_base_image "$debian_source" "$DEBIAN_BASE"
  tag_and_push_base_image "$docreader_python_source" "$DOCREADER_PYTHON_BASE"
  tag_and_push_base_image "$nginx_source" "$NGINX_BASE"
  tag_and_push_base_image "$sandbox_node_source" "$SANDBOX_NODE_BASE"
  tag_and_push_base_image "$sandbox_python_source" "$SANDBOX_PYTHON_BASE"
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
[[ -n "$ACR_USERNAME" ]] || fail "--username is required"

if [[ -z "$IMAGE_TAG" ]]; then
  IMAGE_TAG="$(git -C "$PROJECT_ROOT" rev-parse --short HEAD 2>/dev/null || true)"
  if [[ -z "$IMAGE_TAG" ]]; then
    IMAGE_TAG="$(date +%Y%m%d%H%M%S)"
  fi
fi

if [[ "$SKIP_LOGIN" != true ]]; then
  if [[ -z "${ACR_PASSWORD:-}" ]]; then
    read -r -s -p "ACR password for ${ACR_USERNAME}@${ACR_REGISTRY}: " ACR_PASSWORD
    printf '\n'
  fi

  log "logging in to ${ACR_REGISTRY}"
  printf '%s' "$ACR_PASSWORD" | docker login "$ACR_REGISTRY" --username "$ACR_USERNAME" --password-stdin
fi

if [[ "$SKIP_BUILD" != true ]]; then
  export PLATFORM="$ACR_BUILD_PLATFORM"
  export TARGETARCH="$(target_arch)"

  if [[ "$USE_ACR_BASE_IMAGES" == true ]]; then
    prepare_acr_base_images
  fi

  log "building local images"
  "$PROJECT_ROOT/scripts/build_images.sh" --all
fi

for image in "${IMAGES[@]}"; do
  local_image="wechatopenai/${image}:latest"
  remote_image="${ACR_REGISTRY}/${ACR_NAMESPACE}/${image}:${IMAGE_TAG}"

  log "tagging ${local_image} -> ${remote_image}"
  docker tag "$local_image" "$remote_image"

  log "pushing ${remote_image}"
  docker push "$remote_image"
done

cat <<EOF

Pushed tag: ${IMAGE_TAG}

Use this on ECS:
  WEKNORA_VERSION=${IMAGE_TAG}

Image prefix:
  ${ACR_REGISTRY}/${ACR_NAMESPACE}
EOF
