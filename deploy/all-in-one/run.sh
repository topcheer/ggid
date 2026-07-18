#!/bin/sh
# GGID All-in-One Docker launcher
# Usage: bash deploy/all-in-one/run.sh

set -e

IMAGE="ggid/ggid-all-in-one:latest"
NAME="ggid-all-in-one"

echo "=== GGID All-in-One Docker ==="

# Build if image doesn't exist
if ! docker image inspect "$IMAGE" >/dev/null 2>&1; then
    echo "Building image..."
    docker build -f deploy/all-in-one/Dockerfile -t "$IMAGE" .
fi

# Remove old container
docker rm -f "$NAME" 2>/dev/null || true

# Run with all service ports exposed (IPv4 only to avoid macOS IPv6 issues)
echo "Starting container..."
docker run -d \
    -p 127.0.0.1:8080:8080 \
    -p 127.0.0.1:3000:3000 \
    # Do NOT map backend ports to host — they bypass gateway authentication.
    # Only gateway (8080) and console (3000) are accessible externally.
    --name "$NAME" \
    "$IMAGE"

echo ""
echo "Waiting for services to start..."
sleep 20

# Health check
HEALTH=$(curl -s http://127.0.0.1:8080/healthz 2>/dev/null || echo "FAILED")
if echo "$HEALTH" | grep -q "ok"; then
    echo ""
    echo "=== GGID is running! ==="
    echo ""
    echo "  Console:   http://127.0.0.1:3000/"
    echo "  API:       http://127.0.0.1:8080/"
    echo ""
    echo "  Username:  admin"
    echo "  Password:  Password123!"
    echo "  Tenant:    00000000-0000-0000-0000-000000000001"
    echo ""
    echo "  Logs:      docker exec $NAME tail -f /var/log/supervisor/gateway-server.log"
    echo "  Stop:      docker rm -f $NAME"
else
    echo "WARNING: Services still starting up. Wait 10 more seconds then try http://127.0.0.1:3000/"
fi
