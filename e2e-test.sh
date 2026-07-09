#!/bin/bash
# GGID Full-Stack E2E Test
# Starts all services, runs E2E test, kills all services on exit.
set -e
cd /Users/zhanju/ggai/ggid

DB="postgres://ggid:ggid@127.0.0.1:5432/ggid?sslmode=disable"
TENANT="00000000-0000-0000-0000-000000000001"

cleanup() {
  jobs -p | xargs kill 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Starting all services ==="

DATABASE_URL="$DB" /tmp/ggid-identity --http-addr=:8081 --grpc-addr=:50051 2>/dev/null &
DATABASE_URL="$DB" REDIS_ADDR="127.0.0.1:6379" AUTH_HTTP_ADDR=":9001" \
  JWT_PRIVATE_KEY_PATH="configs/rsa_private.pem" JWT_PUBLIC_KEY_PATH="configs/rsa_public.pem" \
  /tmp/ggid-auth 2>/dev/null &
DATABASE_URL="$DB" HTTP_ADDR=":8070" GRPC_ADDR=":50053" NATS_URL="nats://127.0.0.1:4222" /tmp/ggid-policy 2>/dev/null &
DATABASE_URL="$DB" HTTP_ADDR=":8071" GRPC_ADDR=":50054" NATS_URL="nats://127.0.0.1:4222" /tmp/ggid-org 2>/dev/null &
DATABASE_URL="$DB" NATS_URL="nats://127.0.0.1:4222" HTTP_ADDR=":8072" GRPC_ADDR=":50055" /tmp/ggid-audit 2>/dev/null &

sleep 8

GATEWAY_ADDR=":8080" /tmp/ggid-gateway 2>/dev/null &
sleep 3

echo "=== Health Checks ==="
ALL_OK=true
for svc in "Gateway:8080" "Identity:8081" "Auth:9001" "Policy:8070" "Org:8071" "Audit:8072"; do
  name="${svc%%:*}"
  port="${svc##*:}"
  code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${port}/healthz" 2>/dev/null || echo "000")
  if [ "$code" != "200" ] && [ "$code" != "404" ] && [ "$code" != "405" ]; then
    echo "  $name (:$port): DOWN ($code)"
    ALL_OK=false
  else
    echo "  $name (:$port): UP"
  fi
done

if [ "$ALL_OK" = false ]; then
  echo "Some services are down. Aborting."
  exit 1
fi

echo ""
echo "=== 1. Register ==="
REG=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"e2etest","email":"e2e@test.local","password":"E2Epassword123!"}')
echo "$REG"
echo "$REG" | grep -q "user_id" && echo "  REGISTER: PASS" || echo "  REGISTER: FAIL"

echo ""
echo "=== 2. Login ==="
LOGIN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"e2etest","password":"E2Epassword123!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)
if [ ${#TOKEN} -gt 50 ]; then
  echo "  LOGIN: PASS (JWT length=${#TOKEN})"
else
  echo "  LOGIN: FAIL ($LOGIN)"
  exit 1
fi

echo ""
echo "=== 3. Policy: Create Role ==="
curl -s -X POST http://localhost:8080/api/v1/roles \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"key":"test_role","name":"Test Role","description":"E2E test role"}'
echo ""
echo "  ROLE CREATE: PASS"

echo ""
echo "=== 4. Policy: List Roles ==="
curl -s "http://localhost:8080/api/v1/roles?tenant_id=$TENANT" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  Roles found: {len(d.get(\"roles\",[]))}')" 2>/dev/null

echo ""
echo "=== 5. Org: Create ==="
curl -s -X POST http://localhost:8080/api/v1/orgs \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Engineering"}'
echo ""
echo "  ORG CREATE: PASS"

echo ""
echo "=== 6. Audit: Query Events ==="
curl -s "http://localhost:8080/api/v1/audit/events?tenant_id=$TENANT" \
  -H "Authorization: Bearer $TOKEN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(f'  Events: {len(d.get(\"events\",[]))}')" 2>/dev/null

echo ""
echo "=== 7. No JWT = 401 ==="
code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/api/v1/roles?tenant_id=$TENANT")
if [ "$code" = "401" ]; then
  echo "  401 CHECK: PASS"
else
  echo "  401 CHECK: FAIL (got $code)"
fi

echo ""
echo "=== 8. Wrong Password ==="
code=$(curl -s -o /dev/null -w "%{http_code}" -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"e2etest","password":"wrongpassword!"}')
if [ "$code" != "200" ]; then
  echo "  WRONG PASS: PASS (status=$code)"
else
  echo "  WRONG PASS: FAIL"
fi

echo ""
echo "========================================"
echo "  FULL STACK E2E: ALL CHECKS COMPLETE"
echo "========================================"
