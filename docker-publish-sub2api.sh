#!/usr/bin/env sh

set -eu

IMAGE="${IMAGE:-ikun0x00/sub2api:latest}"
ACTION="${1:-publish}"
PLATFORMS="${PLATFORMS:-}"
BUILD_ARGS="${BUILD_ARGS:-}"

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$SCRIPT_DIR"

usage() {
  echo "Usage: $0 [publish|build|push|all|buildx]"
  echo
  echo "Defaults:"
  echo "  IMAGE=ikun0x00/sub2api:latest"
  echo "  ACTION=publish"
  echo "  publish builds for the current Docker host platform"
  echo
  echo "Examples:"
  echo "  $0"
  echo "  IMAGE=ikun0x00/sub2api:test $0"
  echo "  $0 build"
  echo "  $0 all"
  echo "  PLATFORMS=linux/amd64,linux/arm64 $0 buildx"
}

build_image() {
  echo "==> Building image: $IMAGE"
  # shellcheck disable=SC2086
  docker build $BUILD_ARGS -t "$IMAGE" .
}

push_image() {
  echo "==> Pushing image: $IMAGE"
  docker push "$IMAGE"
}

buildx_image() {
  if [ -z "$PLATFORMS" ]; then
    echo "PLATFORMS is required for buildx, for example:"
    echo "  PLATFORMS=linux/amd64,linux/arm64 $0 buildx"
    exit 1
  fi
  echo "==> Building and pushing multi-arch image: $IMAGE"
  echo "==> Platforms: $PLATFORMS"
  # shellcheck disable=SC2086
  docker buildx build $BUILD_ARGS --platform "$PLATFORMS" -t "$IMAGE" --push .
}

case "$ACTION" in
  publish)
    build_image
    push_image
    ;;
  build)
    build_image
    ;;
  push)
    push_image
    ;;
  all)
    build_image
    push_image
    ;;
  buildx)
    buildx_image
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    usage
    exit 1
    ;;
esac
