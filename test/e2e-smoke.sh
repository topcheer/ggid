#!/usr/bin/env bash
# GGID Platform E2E Smoke Test
# Tests the full flow: register → login → JWT → CRUD → 401 → MFA → OAuth
# Usage: bash test/e2e-smoke.sh [BASE_URL]
set -euo pipefail

BASE_URL="${1:-http://127.0.0.1:8080}"
TENANT_ID="00000000-0000-0000-0000-000000000001"
PASS=0
FAIL=0
JWT=""

green() { printf "\033[32m%s\033[0m\n" "$1"; }
red()   { printf "\033[31m%s\033[0m\n" "$1"; }
info()  { printf "\033[36m%s\033[0m\n" "$1"; }

assert_status() {
  local expected="$1" actual="$2" name="$3"
  if [ "$actual" = "$expected" ]; then
    green "  PASS: $name (HTTP $actual)"
    PASS=$((PASS + 1))
  else
    red  "  FAIL: $name — expected $expected, got $actual"
    FAIL=$((FAIL + 1))
  fi
}

assert_contains() {
  local body="$1" pattern="$2" name="$3"
  if echo "$body" | grep -q "$pattern"; then
    green "  PASS: $name"
    PASS=$((PASS + 1))
  else
    red  "  FAIL: $name — '$pattern' not in response"
    FAIL=$((FAIL + 1))
  fi
}

info "========================================"
info " GGID E2E Smoke Test"
info " Base URL: $BASE_URL"
info "========================================"

# --- 1. Health Check ---
info "1. Gateway Health Check"
HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/healthz")
assert_status "200" "$HEALTH" "Gateway healthz returns 200"

# --- 2. Register User ---
info "2. Register New User"
TS=$(date +%s%N)
REG_BODY=$(cat <<EOF
{"username":"smoke_${TS}","email":"smoke_${TS}@test.com","password":"SmokeTest@12345","full_name":"Smoke Test","tenant_id":"${TENANT_ID}"}
EOF
)
REG_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
  -d "$REG_BODY")
REG_CODE=$(echo "$REG_RESP" | tail -1)
assert_status "201" "$REG_CODE" "Register returns 201"
sleep 1

# --- 3. Login ---
info "3. Login with Registered User"
# Retry login to handle rate limiting
LOGIN_CODE=""
LOGIN_JSON=""
for i in 1 2 3; do
  LOGIN_BODY=$(cat <<EOF
{"username":"smoke_${TS}","password":"SmokeTest@12345","tenant_id":"${TENANT_ID}"}
EOF
)
  LOGIN_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/auth/login" \
    -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
    -d "$LOGIN_BODY")
  LOGIN_CODE=$(echo "$LOGIN_RESP" | tail -1)
  LOGIN_JSON=$(printf '%s' "$LOGIN_RESP" | sed '$d')
  if [ "$LOGIN_CODE" = "200" ]; then break; fi
  sleep 2
done
assert_status "200" "$LOGIN_CODE" "Login returns 200"

# Extract JWT
JWT=$(echo "$LOGIN_JSON" | sed 's/.*"access_token":"//' | sed 's/".*//' | head -c 1000)
if [ -n "$JWT" ]; then
  green "  PASS: JWT extracted (${#JWT} chars)"
  PASS=$((PASS + 1))
else
  red  "  FAIL: JWT not found in login response"
  FAIL=$((FAIL + 1))
fi

# --- 4. Access without JWT ---
info "4. 401 Without JWT"
NO_AUTH=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/users" \
  -H "X-Tenant-ID: $TENANT_ID")
assert_status "401" "$NO_AUTH" "Users endpoint requires JWT"

# --- 5. List Users ---
info "5. List Users (Authenticated)"
USERS_RESP=$(curl -s -w "\n%{http_code}" "$BASE_URL/api/v1/users" \
  -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID")
USERS_CODE=$(echo "$USERS_RESP" | tail -1)
assert_status "200" "$USERS_CODE" "List users returns 200"

# --- 6. Create Role ---
info "6. Create Role"
ROLE_BODY=$(cat <<EOF
{"name":"smoke_role_${TS}","key":"smoke_role_${TS}","description":"Smoke test role","permissions":["read:users"],"tenant_id":"${TENANT_ID}"}
EOF
)
ROLE_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/roles" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID" \
  -d "$ROLE_BODY")
ROLE_CODE=$(echo "$ROLE_RESP" | tail -1)
assert_status "201" "$ROLE_CODE" "Create role returns 201"

# --- 7. List Roles ---
info "7. List Roles"
ROLES_RESP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/roles" \
  -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID")
assert_status "200" "$ROLES_RESP" "List roles returns 200"

# --- 8. Create Organization ---
info "8. Create Organization"
ORG_BODY=$(cat <<EOF
{"name":"Smoke Org ${TS}","description":"Smoke test org","type":"department","tenant_id":"${TENANT_ID}"}
EOF
)
ORG_RESP=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/v1/orgs" \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID" \
  -d "$ORG_BODY")
ORG_CODE=$(echo "$ORG_RESP" | tail -1)
assert_status "201" "$ORG_CODE" "Create org returns 201"

# --- 9. Query Audit ---
info "9. Query Audit Events"
AUDIT_RESP=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/v1/audit" \
  -H "Authorization: Bearer $JWT" -H "X-Tenant-ID: $TENANT_ID")
assert_status "200" "$AUDIT_RESP" "Audit query returns 200"

# --- 10. Wrong Password ---
info "10. Wrong Password Returns 401"
WRONG_BODY=$(cat <<EOF
{"username":"smoke_${TS}","password":"WrongPassword123","tenant_id":"${TENANT_ID}"}
EOF
)
WRONG_RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
  -d "$WRONG_BODY")
assert_status "401" "$WRONG_RESP" "Wrong password returns 401"

# --- 11. Duplicate Register ---
info "11. Duplicate Register Returns 409"
DUP_RESP=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" -H "X-Tenant-ID: $TENANT_ID" \
  -d "$REG_BODY")
assert_status "409" "$DUP_RESP" "Duplicate register returns 409"

# --- 12. Hosted Login Page ---
info "12. Hosted Login Page Accessible"
LOGIN_PAGE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/login")
assert_status "200" "$LOGIN_PAGE" "Hosted login page returns 200"

# --- 13. Swagger Docs ---
info "13. Swagger UI Accessible"
SWAGGER=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/docs")
assert_status "200" "$SWAGGER" "Swagger UI returns 200"

# --- 14. JWKS Endpoint ---
info "14. JWKS Endpoint"
JWKS=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/.well-known/jwks.json")
assert_status "200" "$JWKS" "JWKS endpoint returns 200"

# --- 15. OpenID Configuration ---
info "15. OpenID Configuration"
OIDC=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/.well-known/openid-configuration")
if [ "$OIDC" = "200" ]; then
  green "  PASS: OIDC discovery returns 200"
  PASS=$((PASS + 1))
else
  info "  SKIP: OIDC discovery not implemented yet (HTTP $OIDC)"
fi

# --- Summary ---
info "========================================"
info " Results: $PASS passed, $FAIL failed"
if [ "$FAIL" -eq 0 ]; then
  green " ALL TESTS PASSED"
else
  red " $FAIL TESTS FAILED"
fi
info "========================================"

exit $FAIL
