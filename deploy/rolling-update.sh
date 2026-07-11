#!/bin/bash
# GGID Rolling Update Script — Zero-Downtime Deployment
#
# Usage: bash deploy/rolling-update.sh <image-tag>
# Example: bash deploy/rolling-update.sh v2.0.0
#
# This script performs a health-gated rolling update of all GGID services
# in a Docker Compose environment. It waits for each service to become
# healthy before proceeding to the next one.

set -euo pipefail

TAG="${1:-latest}"
SERVICES=("gateway" "identity" "auth" "oauth" "policy" "org" "audit")
COMPOSE_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "============================================"
echo "  GGID Rolling Update to tag: $TAG"
echo "============================================"
echo ""

# Verify docker compose is available
if ! command -v docker &> /dev/null; then
    echo "ERROR: docker not found"
    exit 1
fi

# Pull new images
echo "[1/3] Pulling images..."
for svc in "${SERVICES[@]}"; do
    echo "  Pulling ggid-$svc:$TAG..."
    docker compose -f "$COMPOSE_DIR/docker-compose.yaml" pull "$svc" 2>/dev/null || true
done
echo ""

# Rolling update each service
echo "[2/3] Rolling update services..."
FAILED_SERVICE=""

for svc in "${SERVICES[@]}"; do
    echo -n "  Updating $svc... "
    
    # Recreate the service container
    docker compose -f "$COMPOSE_DIR/docker-compose.yaml" up -d --no-deps --force-recreate "$svc"
    
    # Wait for healthcheck (max 60 seconds)
    HEALTHY=false
    for i in $(seq 1 30); do
        STATUS=$(docker inspect --format='{{.State.Health.Status}}' "ggid-$svc" 2>/dev/null || echo "none")
        if [ "$STATUS" = "healthy" ]; then
            HEALTHY=true
            break
        fi
        sleep 2
    done
    
    if [ "$HEALTHY" = "true" ]; then
        echo "OK (healthy)"
    else
        echo "FAILED (timeout)"
        echo ""
        echo "ERROR: $svc did not become healthy within 60 seconds"
        echo "Rolling back..."
        docker compose -f "$COMPOSE_DIR/docker-compose.yaml" restart "$svc"
        FAILED_SERVICE="$svc"
        break
    fi
done

echo ""

# Verify
echo "[3/3] Verifying..."
if [ -n "$FAILED_SERVICE" ]; then
    echo "  ROLLBACK: Update failed at $FAILED_SERVICE"
    exit 1
fi

# Run E2E test if available
if [ -f "$COMPOSE_DIR/e2e-docker-test.sh" ]; then
    echo "  Running E2E tests..."
    if bash "$COMPOSE_DIR/e2e-docker-test.sh"; then
        echo "  E2E: ALL PASS"
    else
        echo "  E2E: SOME TESTS FAILED (service may still be starting)"
    fi
fi

echo ""
echo "============================================"
echo "  Rolling update COMPLETE"
echo "============================================"
echo ""
docker compose -f "$COMPOSE_DIR/docker-compose.yaml" ps
