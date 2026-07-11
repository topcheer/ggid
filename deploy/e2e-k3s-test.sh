#!/bin/bash
# K3s E2E Test Suite — tests GGID via Ingress
# Usage: bash deploy/e2e-k3s-test.sh [GATEWAY_URL]
set -u

GATEWAY="${1:-https://ggid.iot2.win}"
TENANT_ID="00000000-0000-0000-0000-000000000001"
TS=$(date +%s)
# Use random password to avoid HIBP breach check triggering MFA
PASS_PWD="Xk9#$(openssl rand -hex 12)"
PASS=0
FAIL=0

echo "=== K3s E2E Test (via Ingress) ==="
echo "Gateway: $GATEWAY"
echo "Tenant:  $TENANT_ID"
echo "Run:     $TS"
echo ""

# 1. Gateway healthz
echo -n "1. Gateway healthz:        "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' "$GATEWAY/healthz")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 2. Register user
echo -n "2. Register user:          "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"username\":\"k3s-${TS}\",\"email\":\"k3s-${TS}@test.com\",\"password\":\"${PASS_PWD}\"}")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 3. Login + JWT
echo -n "3. Login + JWT:            "
LOGIN=$(curl -sk -X POST "$GATEWAY/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"username\":\"k3s-${TS}\",\"password\":\"${PASS_PWD}\"}")
JWT=$(echo "$LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null)
if [ -n "$JWT" ]; then echo "PASS (len=${#JWT})"; PASS=$((PASS+1)); else echo "FAIL (no JWT)"; FAIL=$((FAIL+1)); fi

# 4. 401 without JWT
echo -n "4. 401 without JWT:        "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/users" -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "401" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 5. List users with JWT
echo -n "5. List users (JWT):       "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/users" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 6. Create role
echo -n "6. Create role:            "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"name\":\"k3s-role-${TS}\",\"key\":\"k3s_role_${TS}\",\"description\":\"K3s test\",\"tenant_id\":\"$TENANT_ID\"}")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 7. List roles
echo -n "7. List roles:             "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' "$GATEWAY/api/v1/roles" \
  -H "Authorization: Bearer $JWT" \
  -H "X-Tenant-ID: $TENANT_ID")
if [ "$STATUS" = "200" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 8. Create org
echo -n "8. Create org:             "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/orgs" \
  -H "Authorization: Bearer $JWT" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"name\":\"k3s-org-${TS}\",\"description\":\"K3s test\",\"tenant_id\":\"$TENANT_ID\"}")
if [ "$STATUS" = "200" ] || [ "$STATUS" = "201" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 9. Wrong password
echo -n "9. Wrong password 401:     "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"username\":\"k3s-${TS}\",\"password\":\"WrongPassword123\"}")
if [ "$STATUS" = "401" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

# 10. Duplicate register
echo -n "10. Dup register 409:      "
STATUS=$(curl -sk -o /dev/null -w '%{http_code}' -X POST "$GATEWAY/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: $TENANT_ID" \
  -d "{\"username\":\"k3s-${TS}\",\"email\":\"k3s-${TS}@test.com\",\"password\":\"${PASS_PWD}\"}")
if [ "$STATUS" = "409" ]; then echo "PASS ($STATUS)"; PASS=$((PASS+1)); else echo "FAIL ($STATUS)"; FAIL=$((FAIL+1)); fi

echo ""
echo "================================"
echo "Results: $PASS PASS / $FAIL FAIL"
echo "================================"

[ "$FAIL" -eq 0 ] && exit 0 || exit 1
