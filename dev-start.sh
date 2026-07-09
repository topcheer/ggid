#!/bin/bash
# GGID local dev startup script
set -e
cd "$(dirname "$0")"

echo "Starting GGID services..."

# Kill old processes
pkill -f "ggid-identity\|ggid-auth\|ggid-gateway\|ggid-policy\|ggid-org\|ggid-audit" 2>/dev/null || true
sleep 2

DB="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable"
REDIS="127.0.0.1:6379"
NATS="nats://127.0.0.1:4222"

# Identity
DATABASE_URL="$DB" /tmp/ggid-identity --http-addr=:8081 --grpc-addr=:50051 &
# Auth
DATABASE_URL="$DB" REDIS_ADDR="$REDIS" AUTH_HTTP_ADDR=":9001" \
  JWT_PRIVATE_KEY_PATH="configs/rsa_private.pem" JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
  /tmp/ggid-auth &
# Policy
DATABASE_URL="$DB" HTTP_ADDR=":8070" GRPC_ADDR=":50053" NATS_URL="$NATS" /tmp/ggid-policy &
# Org
DATABASE_URL="$DB" HTTP_ADDR=":8071" GRPC_ADDR=":50054" NATS_URL="$NATS" /tmp/ggid-org &
# Audit
DATABASE_URL="$DB" NATS_URL="$NATS" HTTP_ADDR=":8072" GRPC_ADDR=":50055" /tmp/ggid-audit &
sleep 5
# Gateway
GATEWAY_ADDR=":8080" /tmp/ggid-gateway &
sleep 3

echo "All services started. Checking health..."
for svc in "Gateway:8080" "Identity:8081" "Auth:9001" "Policy:8070" "Org:8071" "Audit:8072"; do
  name="${svc%%:*}"
  port="${svc##*:}"
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/healthz" 2>/dev/null || echo "000")
  echo "  $name (:$port): $code"
done
echo "Done."
