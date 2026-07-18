#!/bin/bash
# KB-328: Build all 8 GGID service images using prebuilt binary approach.
#
# This script:
#   1. Compiles all Go service binaries for linux/amd64
#   2. Builds individual Docker images using deploy/docker/Dockerfile.service
#   3. Tags and pushes to registry.iot2.win/ggid/<service>:latest
#
# Usage:
#   bash scripts/build-all-images.sh          # build + push
#   bash scripts/build-all-images.sh --no-push # build only, skip push
#   bash scripts/build-all-images.sh --console # also rebuild console

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

REGISTRY="${REGISTRY:-registry.iot2.win/ggid}"
TAG="${TAG:-latest}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH=true
BUILD_CONSOLE=false

for arg in "$@"; do
  case $arg in
    --no-push)  PUSH=false ;;
    --console)  BUILD_CONSOLE=true ;;
    *) echo "Unknown arg: $arg"; exit 1 ;;
  esac
done

# Go services (space-separated: name cmd_path)
GO_SERVICES=(
  "gateway services/gateway/cmd"
  "auth services/auth/cmd"
  "identity services/identity/cmd"
  "oauth services/oauth/cmd"
  "policy services/policy/cmd"
  "audit services/audit/cmd"
  "org services/org/cmd"
)

BIN_DIR="$ROOT_DIR/build/bin"
mkdir -p "$BIN_DIR"

# ── Step 1: Build all Go binaries ──────────────────────────────
echo "============================================"
echo "  Building Go binaries (linux/amd64, CGO=0)"
echo "============================================"

for entry in "${GO_SERVICES[@]}"; do
  # shellcheck disable=SC2086
  set -- $entry
  svc="$1"
  cmd_path="$2"
  echo -n "  Building $svc... "
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$BIN_DIR/$svc" "$ROOT_DIR/$cmd_path/"
  echo "done ($(du -h "$BIN_DIR/$svc" | cut -f1))"
done

echo ""

# ── Step 2: Build Docker images ────────────────────────────────
echo "============================================"
echo "  Building Docker images"
echo "============================================"

BUILT_IMAGES=()

for entry in "${GO_SERVICES[@]}"; do
  # shellcheck disable=SC2086
  set -- $entry
  svc="$1"
  echo -n "  Building $svc image... "
  docker build --platform "$PLATFORM" \
    -f deploy/docker/Dockerfile.service \
    --build-arg "BINARY=build/bin/$svc" \
    -t "$REGISTRY/$svc:$TAG" \
    "$ROOT_DIR" 2>&1 | tail -1
  BUILT_IMAGES+=("$svc")
done

# Console image (separate multi-stage build)
if [ "$BUILD_CONSOLE" = true ]; then
  echo -n "  Building console image... "
  docker build --platform "$PLATFORM" \
    -f console/Dockerfile \
    -t "$REGISTRY/console:$TAG" \
    "$ROOT_DIR" 2>&1 | tail -1
  BUILT_IMAGES+=("console")
fi

echo ""

# ── Step 3: Push images ─────────────────────────────────────────
if [ "$PUSH" = true ]; then
  echo "============================================"
  echo "  Pushing to $REGISTRY"
  echo "============================================"

  for svc in "${BUILT_IMAGES[@]}"; do
    echo -n "  Pushing $svc... "
    docker push "$REGISTRY/$svc:$TAG" 2>&1 | tail -1
  done

  echo ""
  echo "============================================"
  echo "  All ${#BUILT_IMAGES[@]} images pushed to $REGISTRY"
  echo "============================================"
else
  echo "============================================"
  echo "  Built ${#BUILT_IMAGES[@]} images (--no-push, skipping registry)"
  echo "============================================"
fi

# ── Summary ────────────────────────────────────────────────────
echo ""
echo "Images:"
for svc in "${BUILT_IMAGES[@]}"; do
  echo "  $REGISTRY/$svc:$TAG"
done
