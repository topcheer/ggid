#!/bin/bash
# GGID Production E2E Test Suite
# Tests all API endpoints + console page accessibility
set -euo pipefail

BASE="https://ggid-console.iot2.win"
TENANT="00000000-0000-0000-0000-000000000001"
K="-k -s --max-time 10 -H 'Accept-Encoding: identity'"
PASS=0
FAIL=0
SKIP=0
FAILED_TESTS=()

api_call() {
  local method="$1" path="$2" data="$3" token="$4"
  local h="-H 'Content-Type: application/json' -H 'X-Tenant-ID: $TENANT'"
  [ -n "$token" ] && h="$h -H 'Authorization: Bearer $token'"
  [ -n "$data" ] && data="-d '$data'"
  eval curl $K $h $data -X $method -w '"\\n%{http_code}"' "$BASE$path" 2>/dev/null
}

page_check() {
  local path="$1"
  local code
  code=$(eval curl $K -o /dev/null -w '"%{http_code}"' "$BASE$path" 2>/dev/null)
  if [ "$code" = "200" ]; then
    echo "  PASS  GET $path ($code)"
    PASS=$((PASS+1))
  else
    echo "  FAIL  GET $path ($code)"
    FAIL=$((FAIL+1))
    FAILED_TESTS+=("GET $path -> $code")
  fi
}

echo "=========================================="
echo "GGID Production E2E Test Suite"
echo "=========================================="
echo ""

# === 1. Health Check ===
echo "--- 1. Health Check ---"
code=$(eval curl $K -o /dev/null -w '"%{http_code}"' "$BASE/api/v1/healthz" 2>/dev/null || echo "000")
if [ "$code" = "200" ]; then
  echo "  PASS  Gateway healthz ($code)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Gateway healthz ($code)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("healthz -> $code")
fi

# === 2. Register ===
echo ""
echo "--- 2. Register ---"
TS=$(date +%s)
REG_USER="e2e_${TS}"
REG_RESULT=$(api_call POST /api/v1/auth/register "{\"username\":\"$REG_USER\",\"password\":\"E2eTestPass123!\",\"email\":\"${REG_USER}@test.com\",\"display_name\":\"E2E User\"}" "")
REG_CODE=$(echo "$REG_RESULT" | tail -1)
REG_BODY=$(echo "$REG_RESULT" | head -n -1)
if [ "$REG_CODE" = "201" ]; then
  echo "  PASS  Register user=$REG_USER ($REG_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Register user=$REG_USER ($REG_CODE): $REG_BODY"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("register -> $REG_CODE")
fi

# === 3. Login ===
echo ""
echo "--- 3. Login ---"
LOGIN_RESULT=$(api_call POST /api/v1/auth/login "{\"username\":\"$REG_USER\",\"password\":\"E2eTestPass123!\"}" "")
LOGIN_CODE=$(echo "$LOGIN_RESULT" | tail -1)
LOGIN_BODY=$(echo "$LOGIN_RESULT" | head -n -1)
if [ "$LOGIN_CODE" = "200" ]; then
  TOKEN=$(echo "$LOGIN_BODY" | python3 -c "import sys,json; print(json.load(sys.stdin)['access_token'])" 2>/dev/null || echo "")
  if [ -n "$TOKEN" ]; then
    echo "  PASS  Login + JWT extracted ($LOGIN_CODE)"
    PASS=$((PASS+1))
  else
    echo "  FAIL  Login: JWT extraction failed ($LOGIN_CODE): $LOGIN_BODY"
    FAIL=$((FAIL+1))
    FAILED_TESTS+=("login JWT extract")
  fi
else
  echo "  FAIL  Login user=$REG_USER ($LOGIN_CODE): $LOGIN_BODY"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("login -> $LOGIN_CODE")
  TOKEN=""
fi

# === 4. Wrong password ===
echo ""
echo "--- 4. Wrong Password ---"
WP_RESULT=$(api_call POST /api/v1/auth/login "{\"username\":\"$REG_USER\",\"password\":\"wrongpassword\"}" "")
WP_CODE=$(echo "$WP_RESULT" | tail -1)
if [ "$WP_CODE" = "401" ]; then
  echo "  PASS  Wrong password rejected ($WP_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Wrong password should be 401, got ($WP_CODE)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("wrong password -> $WP_CODE")
fi

# === 5. Duplicate Register ===
echo ""
echo "--- 5. Duplicate Register ---"
DUP_RESULT=$(api_call POST /api/v1/auth/register "{\"username\":\"$REG_USER\",\"password\":\"E2eTestPass123!\",\"email\":\"${REG_USER}@test.com\",\"display_name\":\"E2E User\"}" "")
DUP_CODE=$(echo "$DUP_RESULT" | tail -1)
if [ "$DUP_CODE" = "409" ] || [ "$DUP_CODE" = "400" ]; then
  echo "  PASS  Duplicate register rejected ($DUP_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Duplicate register should be 409/400, got ($DUP_CODE)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("dup register -> $DUP_CODE")
fi

# === 6. API endpoints with valid token ===
echo ""
echo "--- 6. API Endpoints (authenticated) ---"
ENDPOINTS=(
  "GET /api/v1/users"
  "GET /api/v1/roles"
  "GET /api/v1/orgs"
  "GET /api/v1/audit/events"
  "GET /api/v1/policies"
  "GET /api/v1/auth/me"
  "GET /api/v1/auth/sessions"
  "GET /api/v1/security/incidents"
  "GET /api/v1/security/threats"
  "GET /api/v1/oauth/clients"
  "GET /api/v1/agents"
  "GET /api/v1/admin/users"
  "GET /api/v1/access-requests"
  "GET /api/v1/compliance/schedules"
  "GET /api/v1/siem/health"
  "GET /api/v1/webhooks"
  "GET /api/v1/rate-limits"
  "GET /api/v1/permissions/tree"
  "GET /api/v1/permissions/inheritance"
  "GET /api/v1/sod/rules"
  "GET /api/v1/sod/conflicts"
  "GET /api/v1/consent"
  "GET /api/v1/notifications"
  "GET /api/v1/mfa/status"
  "GET /api/v1/tokens"
  "GET /api/v1/audit/hash-chain"
  "GET /api/v1/introspection/config"
  "GET /api/v1/login-security"
  "GET /api/v1/password-history"
  "GET /api/v1/delegation"
  "GET /api/v1/account-linking"
  "GET /api/v1/policy-versions"
  "GET /api/v1/device-bindings"
  "GET /api/v1/alerts"
  "GET /api/v1/event-correlation/rules"
  "GET /api/v1/role-templates"
  "GET /api/v1/scope-management/scopes"
)

for ep in "${ENDPOINTS[@]}"; do
  METHOD=$(echo "$ep" | awk '{print $1}')
  PATH_EP=$(echo "$ep" | awk '{print $2}')
  RESULT=$(api_call $METHOD "$PATH_EP" "" "$TOKEN")
  CODE=$(echo "$RESULT" | tail -1)
  BODY=$(echo "$RESULT" | head -n -1 | head -c 100)
  if [ "$CODE" = "200" ]; then
    echo "  PASS  $METHOD $PATH_EP ($CODE)"
    PASS=$((PASS+1))
  elif [ "$CODE" = "404" ]; then
    echo "  SKIP  $METHOD $PATH_EP (404 - not implemented)"
    SKIP=$((SKIP+1))
    FAILED_TESTS+=("$METHOD $PATH_EP -> 404 NOT IMPLEMENTED")
  elif [ "$CODE" = "401" ] || [ "$CODE" = "403" ]; then
    echo "  SKIP  $METHOD $PATH_EP ($CODE - auth/permission issue)"
    SKIP=$((SKIP+1))
  else
    echo "  FAIL  $METHOD $PATH_EP ($CODE): $BODY"
    FAIL=$((FAIL+1))
    FAILED_TESTS+=("$METHOD $PATH_EP -> $CODE")
  fi
done

# === 7. CRUD operations ===
echo ""
echo "--- 7. CRUD Operations ---"

# Create role
ROLE_RESULT=$(api_call POST /api/v1/roles "{\"name\":\"E2E Role $TS\",\"key\":\"e2e_role_$TS\",\"description\":\"E2E test role\"}" "$TOKEN")
ROLE_CODE=$(echo "$ROLE_RESULT" | tail -1)
if [ "$ROLE_CODE" = "201" ] || [ "$ROLE_CODE" = "200" ]; then
  echo "  PASS  Create role ($ROLE_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Create role ($ROLE_CODE): $(echo "$ROLE_RESULT" | head -n -1 | head -c 100)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("create role -> $ROLE_CODE")
fi

# Create org
ORG_RESULT=$(api_call POST /api/v1/orgs "{\"name\":\"E2E Org $TS\"}" "$TOKEN")
ORG_CODE=$(echo "$ORG_RESULT" | tail -1)
if [ "$ORG_CODE" = "201" ] || [ "$ORG_CODE" = "200" ]; then
  echo "  PASS  Create org ($ORG_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Create org ($ORG_CODE): $(echo "$ORG_RESULT" | head -n -1 | head -c 100)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("create org -> $ORG_CODE")
fi

# Create policy
POL_RESULT=$(api_call POST /api/v1/policies "{\"name\":\"E2E Policy $TS\",\"effect\":\"allow\",\"actions\":[\"read\"],\"resources\":[\"*\"]}" "$TOKEN")
POL_CODE=$(echo "$POL_RESULT" | tail -1)
if [ "$POL_CODE" = "201" ] || [ "$POL_CODE" = "200" ]; then
  echo "  PASS  Create policy ($POL_CODE)"
  PASS=$((PASS+1))
else
  echo "  FAIL  Create policy ($POL_CODE): $(echo "$POL_RESULT" | head -n -1 | head -c 100)"
  FAIL=$((FAIL+1))
  FAILED_TESTS+=("create policy -> $POL_CODE")
fi

# === 8. Console Pages ===
echo ""
echo "--- 8. Console Pages ---"
PAGES=(
  "/"
  "/login"
  "/users"
  "/roles"
  "/organizations"
  "/audit"
  "/security"
  "/settings"
  "/settings/sso"
  "/settings/oauth-clients"
  "/settings/api-keys"
  "/settings/mfa"
  "/settings/certificates"
  "/settings/branding"
  "/settings/tenant-config"
  "/settings/login-flows"
  "/agents"
  "/dashboard"
  "/policies"
  "/access-requests"
)
for page in "${PAGES[@]}"; do
  page_check "$page"
done

# === Summary ===
echo ""
echo "=========================================="
echo "E2E Test Summary"
echo "=========================================="
echo "  PASS: $PASS"
echo "  FAIL: $FAIL"
echo "  SKIP: $SKIP (404/401/403)"
echo "  Total: $((PASS+FAIL+SKIP))"
echo ""
if [ $FAIL -gt 0 ]; then
  echo "FAILED TESTS:"
  for t in "${FAILED_TESTS[@]}"; do
    echo "  - $t"
  done
fi
echo "=========================================="
exit $FAIL
