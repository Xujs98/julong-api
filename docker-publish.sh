#!/usr/bin/env bash

set -Eeuo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGE="${IMAGE:-qq1371446705/julong-api}"
PLATFORM="${PLATFORM:-linux/amd64}"
ACTION="${1:-publish}"
TAG="${2:-${TAG:-latest}}"

usage() {
  cat <<'EOF'
Usage:
  ./docker-publish.sh build [tag]    Build and load the image locally
  ./docker-publish.sh push [tag]     Push an existing local image
  ./docker-publish.sh publish [tag]  Build locally, verify, and push

Environment variables:
  IMAGE=qq1371446705/julong-api
  PLATFORM=linux/amd64
  TAG=latest
  NO_CACHE=1

Examples:
  ./docker-publish.sh build
  ./docker-publish.sh publish latest
  IMAGE=registry.example.com/team/julong-api ./docker-publish.sh publish 2026.09.08-1
EOF
}

if [[ ! "$TAG" =~ ^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$ ]]; then
  echo "Invalid Docker tag: $TAG" >&2
  exit 2
fi

case "$ACTION" in
  build | push | publish) ;;
  -h | --help | help)
    usage
    exit 0
    ;;
  *)
    usage >&2
    exit 2
    ;;
esac

command -v docker >/dev/null 2>&1 || {
  echo "Docker CLI was not found." >&2
  exit 1
}
docker info >/dev/null

IMAGE_REF="${IMAGE}:${TAG}"

build_image() {
  local build_args=(
    buildx build
    --platform "$PLATFORM"
    --load
    --tag "$IMAGE_REF"
  )
  if [[ "${NO_CACHE:-0}" == "1" ]]; then
    build_args+=(--no-cache)
  fi
  build_args+=("$ROOT_DIR")

  echo "Building $IMAGE_REF for $PLATFORM"
  docker "${build_args[@]}"

  local image_platform
  image_platform="$(docker image inspect "$IMAGE_REF" --format '{{.Os}}/{{.Architecture}}')"
  if [[ "$PLATFORM" != "$image_platform" ]]; then
    echo "Image platform mismatch: expected $PLATFORM, got $image_platform" >&2
    exit 1
  fi
  echo "Verified local image: $IMAGE_REF ($image_platform)"
}

push_image() {
  echo "Pushing $IMAGE_REF"
  docker push "$IMAGE_REF"
  echo "Published $IMAGE_REF"
}

case "$ACTION" in
  build)
    build_image
    ;;
  push)
    docker image inspect "$IMAGE_REF" >/dev/null
    push_image
    ;;
  publish)
    build_image
    push_image
    ;;
esac
