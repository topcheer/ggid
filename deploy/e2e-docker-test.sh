#!/bin/bash
# Docker Compose E2E Test Suite
# Tests the full GGID IAM platform through the API Gateway
set -e

GATEWAY="${GATEWAY_URL:-http://localhost:8080}"
TENANT_ID="00000000-0000-0000-0000-000000000001"
PASS=0
FAIL=0

echo "=== Docker Compose E2E Test ==="
echo "Gateway: $GATEWAY"
echo ""

# 1. Gateway healthz
echo -n "1. Gateway healthz:        "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/healthz")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 2. Register with tenant
echo -n "2. Register user:          "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"email":"e2e@docker.test","password":"TestPass123!","name":"E2E Docker Test"}')
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 3. Login + JWT
echo -n "3. Login + JWT:            "
LOGIN=$(curl -s -X POST "$GATEWAY/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"email":"e2e@docker.test","password":"TestPass123!"}')
JWT=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)
if [ -n "$JWT" ]; then echo "PASS (len=${#JWT})"; PASS=$((PASS+1)); else echo "FAIL (no JWT)"; FAIL=$((FAIL+1)); fi

# 4. 401 without JWT
echo -n "4. 401 without JWT:        "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/users")
if [ "$STATUS" = "401" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 5. List users with JWT
echo -n "5. List users (JWT):       "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/users" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 6. Create role
echo -n "6. Create role:            "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"name":"e2e-role","description":"E2E test role"}')
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 7. List roles
echo -n "7. List roles:             "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 8. Create org
echo -n "8. Create org:             "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/orgs" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Content-Type: application/json" \
  -d '{"name":"E2E Corp","slug":"e2e-corp"}')
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 9. Audit query
echo -n "9. Audit query:            "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/audit?limit=5" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 10. Wrong password rejected
echo -n "10. Wrong password 401:    "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d '{"email":"e2e@docker.test","password":"wrong"}')
if [ "$STATUS" = "401" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 11. OAuth healthz (direct)
echo -n "11. OAuth healthz:         "
STATUS=$(curl -s -o /dev/null -w '%{http_code}' "$GATEWAY/oauth/healthz" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -H "Authorization: Bearer $JWT")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

echo ""
echo "================================"
echo "Results: $PASS PASS / $FAIL FAIL"
echo "================================"
[ "$FAIL" -eq 0 ]
