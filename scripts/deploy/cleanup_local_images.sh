#!/usr/bin/env bash

# Remove local tags after a successful registry push without touching volumes.
cleanup_pushed_images() {
  if [[ "${CLEAN_LOCAL_IMAGES:-true}" != "true" ]]; then
    printf '[docker-cleanup] keeping local images (CLEAN_LOCAL_IMAGES=%s)\n' "${CLEAN_LOCAL_IMAGES:-}"
    return 0
  fi

  local image
  for image in "$@"; do
    [[ -n "$image" ]] || continue
    if ! docker image inspect "$image" >/dev/null 2>&1; then
      continue
    fi

    if docker image rm "$image" >/dev/null 2>&1; then
      printf '[docker-cleanup] removed local image tag: %s\n' "$image"
    else
      printf '[docker-cleanup] warning: could not remove %s; it may be used by a container\n' "$image" >&2
    fi
  done

  # Remove unreferenced layers left by previous builds. This does not remove
  # containers, named volumes, or images that still have a tag.
  if [[ "${CLEAN_DANGLING_IMAGES:-true}" == "true" ]]; then
    docker image prune -f >/dev/null 2>&1 || {
      printf '[docker-cleanup] warning: dangling image cleanup failed\n' >&2
    }
  fi
}
