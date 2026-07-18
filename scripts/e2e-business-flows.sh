#!/bin/bash
# scripts/e2e-business-flows.sh — End-to-end API business flow verification.
# Usage: bash scripts/e2e-business-flows.sh [GGID_URL] [TENANT_ID]
# Each step outputs PASS/FAIL + response time.

set -euo pipefail

GGID="${1:-https://ggid.iot2.win}"
TENANT="${2:-00000000-0000-0000-0000-000000000001}"
ADMIN_USER="${ADMIN_USER:-admin}"
ADMIN_PASS="${ADMIN_PASS:-Admin@123456}"

PASS=0
FAIL=0
RESULTS=""

# Colors (disabled if not a TTY)
if [ -t 1 ]; then
  GREEN="\033[32m"; RED="\033[31m"; YELLOW="\033[33m"; NC="\033[0m"
else
  GREEN=""; RED=""; YELLOW=""; NC=""
fi

log_pass() {
  PASS=$((PASS + 1))
  RESULTS="${RESULTS}\n✅ PASS | $1 | ${2}ms"
  echo -e "${GREEN}✅ PASS${NC} | $1 | ${2}ms"
}

log_fail() {
  FAIL=$((FAIL + 1))
  RESULTS="${RESULTS}\n❌ FAIL | $1 | ${2}ms | $3"
  echo -e "${RED}❌ FAIL${NC} | $1 | ${2}ms | $3"
}

# Helper: timed curl POST
tpost() {
  local url="$1" data="$2" auth="$3"
  local start=$(python3 -c 'import time;print(int(time.time()*1000))')
  local resp
  if [ -n "$auth" ]; then
    resp=$(curl -s -w "\n%{http_code}" -X POST "$url" \
      -H "Authorization: Bearer $auth" \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d "$data" 2>/dev/null)
  else
    resp=$(curl -s -w "\n%{http_code}" -X POST "$url" \
      -H "Content-Type: application/json" \
      -H "X-Tenant-ID: $TENANT" \
      -d "$data" 2>/dev/null)
  fi
  local end=$(python3 -c 'import time;print(int(time.time()*1000))')
  local code=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  local elapsed=$((end - start))
  echo "$code|$elapsed|$body"
}

# Helper: timed curl GET
tget() {
  local url="$1" auth="$2"
  local start=$(python3 -c 'import time;print(int(time.time()*1000))')
  local resp
  if [ -n "$auth" ]; then
    resp=$(curl -s -w "\n%{http_code}" "$url" \
      -H "Authorization: Bearer $auth" \
      -H "X-Tenant-ID: $TENANT" 2>/dev/null)
  else
    resp=$(curl -s -w "\n%{http_code}" "$url" \
      -H "X-Tenant-ID: $TENANT" 2>/dev/null)
  fi
  local end=$(python3 -c 'import time;print(int(time.time()*1000))')
  local code=$(echo "$resp" | tail -1)
  local body=$(echo "$resp" | sed '$d')
  local elapsed=$((end - start))
  echo "$code|$elapsed|$body"
}

echo "============================================"
echo "GGID E2E Business Flows"
echo "URL: $GGID"
echo "Tenant: $TENANT"
echo "============================================"
echo ""

# === Step 1: Login ===
echo "--- Step 1: Login ---"
RESP=$(tpost "$GGID/api/v1/auth/login" "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" "")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)
TOKEN=$(echo "$BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

if [ "$CODE" = "200" ] && [ -n "$TOKEN" ]; then
  log_pass "1. Login (admin)" "$MS"
else
  log_fail "1. Login (admin)" "$MS" "code=$CODE token=$([ -n "$TOKEN" ] && echo yes || echo no)"
  echo "FATAL: Cannot continue without token"
  exit 1
fi

# === Step 2: Create User ===
echo "--- Step 2: Create User ---"
E2E_TS=$(date +%s)
RESP=$(tpost "$GGID/api/v1/users" "{\"email\":\"e2e_${E2E_TS}@test.com\",\"password\":\"E2eScript@123\",\"name\":\"E2E Script\",\"username\":\"e2e_${E2E_TS}\"}" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)
USER_ID=$(echo "$BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")

if [ "$CODE" = "200" ] || [ "$CODE" = "201" ]; then
  log_pass "2. Create User" "$MS"
else
  log_fail "2. Create User" "$MS" "code=$CODE body=$(echo $BODY | head -c 100)"
  USER_ID=""
fi

# === Step 3: List Users ===
echo "--- Step 3: List Users ---"
RESP=$(tget "$GGID/api/v1/users?limit=5" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)
TOTAL=$(echo "$BODY" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('total',len(d.get('users',[]))))" 2>/dev/null || echo "0")

if [ "$CODE" = "200" ] && [ "$TOTAL" != "0" ]; then
  log_pass "3. List Users (total=$TOTAL)" "$MS"
else
  log_fail "3. List Users" "$MS" "code=$CODE total=$TOTAL"
fi

# === Step 4: Assign Role ===
echo "--- Step 4: Assign Role ---"
if [ -n "$USER_ID" ]; then
  # Get viewer role ID
  ROLE_RESP=$(tget "$GGID/api/v1/roles" "$TOKEN")
  ROLE_BODY=$(echo "$ROLE_RESP" | cut -d'|' -f3-)
  ROLE_ID=$(echo "$ROLE_BODY" | python3 -c "import sys,json;roles=json.load(sys.stdin)['roles'];print([r['id'] for r in roles if r['key']=='viewer'][0])" 2>/dev/null || echo "")

  if [ -n "$ROLE_ID" ]; then
    RESP=$(tpost "$GGID/api/v1/roles/assign" "{\"user_id\":\"$USER_ID\",\"role_id\":\"$ROLE_ID\"}" "$TOKEN")
    CODE=$(echo "$RESP" | cut -d'|' -f1)
    MS=$(echo "$RESP" | cut -d'|' -f2)
    BODY=$(echo "$RESP" | cut -d'|' -f3-)

    if [ "$CODE" = "200" ] || [ "$CODE" = "201" ]; then
      log_pass "4. Assign Role (viewer)" "$MS"
    else
      log_fail "4. Assign Role" "$MS" "code=$CODE body=$(echo $BODY | head -c 100)"
    fi
  else
    log_fail "4. Assign Role" "0" "viewer role not found"
  fi
else
  log_fail "4. Assign Role" "0" "no user_id"
fi

# === Step 5: Check Permission ===
echo "--- Step 5: Check Permission ---"
RESP=$(tpost "$GGID/api/v1/policies/check" "{\"user_id\":\"${USER_ID:-00000000-0000-0000-0000-000000000000}\",\"resource\":\"users\",\"action\":\"read\"}" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)

if [ "$CODE" = "200" ]; then
  log_pass "5. Check Permission" "$MS"
else
  log_fail "5. Check Permission" "$MS" "code=$CODE body=$(echo $BODY | head -c 80)"
fi

# === Step 6: Create OAuth Client ===
echo "--- Step 6: Create OAuth Client ---"
RESP=$(tpost "$GGID/api/v1/oauth/clients" '{"client_name":"E2E Script Client","redirect_uris":["http://localhost:3000/cb"],"grant_types":["authorization_code","client_credentials"],"response_types":["code"]}' "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)
CLIENT_ID=$(echo "$BODY" | python3 -c "import sys,json;d=json.load(sys.stdin);c=d.get('Client',d);print(c.get('client_id',c.get('ClientID','')))" 2>/dev/null || echo "")

if [ "$CODE" = "200" ] || [ "$CODE" = "201" ]; then
  log_pass "6. Create OAuth Client (id=${CLIENT_ID:0:20})" "$MS"
else
  log_fail "6. Create OAuth Client" "$MS" "code=$CODE body=$(echo $BODY | head -c 100)"
fi

# === Step 7: Client Credentials Token ===
echo "--- Step 7: Client Credentials Token ---"
CLIENT_SECRET=$(echo "$BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('ClientSecret',''))" 2>/dev/null || echo "")

if [ -n "$CLIENT_ID" ] && [ -n "$CLIENT_SECRET" ]; then
  START=$(python3 -c 'import time;print(int(time.time()*1000))')
  RAW=$(curl -s -o /tmp/e2e_token_resp.txt -w "%{http_code}" -X POST "$GGID/api/v1/oauth/token" \
    -H "Content-Type: application/x-www-form-urlencoded" \
    -H "X-Tenant-ID: $TENANT" \
    -d "grant_type=client_credentials&client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&scope=users:read" 2>/dev/null)
  END=$(python3 -c 'import time;print(int(time.time()*1000))')
  CODE="$RAW"
  MS=$((END - START))
  BODY=$(cat /tmp/e2e_token_resp.txt 2>/dev/null)
  MACHINE_TOKEN=$(echo "$BODY" | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

  if [ "$CODE" = "200" ] && [ -n "$MACHINE_TOKEN" ]; then
    log_pass "7. Client Credentials Token" "$MS"
  else
    log_fail "7. Client Credentials Token" "$MS" "code=$CODE body=$(echo $BODY | head -c 80)"
  fi
else
  log_fail "7. Client Credentials Token" "0" "missing client_id or secret"
fi

# === Step 8: Query Audit Events ===
echo "--- Step 8: Query Audit Events ---"
RESP=$(tget "$GGID/api/v1/audit/events?limit=5" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)
EVENT_COUNT=$(echo "$BODY" | python3 -c "import sys,json;d=json.load(sys.stdin);print(d.get('total',len(d.get('events',[]))))" 2>/dev/null || echo "0")

if [ "$CODE" = "200" ]; then
  log_pass "8. Query Audit Events (count=$EVENT_COUNT)" "$MS"
else
  log_fail "8. Query Audit Events" "$MS" "code=$CODE"
fi

# === Step 9: Create Webhook ===
echo "--- Step 9: Create Webhook ---"
RESP=$(tpost "$GGID/api/v1/webhooks" '{"url":"https://httpbin.org/post","events":["user.created"]}' "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)
BODY=$(echo "$RESP" | cut -d'|' -f3-)

if [ "$CODE" = "200" ] || [ "$CODE" = "201" ]; then
  log_pass "9. Create Webhook" "$MS"
else
  log_fail "9. Create Webhook" "$MS" "code=$CODE body=$(echo $BODY | head -c 80)"
fi

# === Step 10: Audit Export ===
echo "--- Step 10: Audit Export ---"
RESP=$(tget "$GGID/api/v1/audit/export" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)

if [ "$CODE" = "200" ]; then
  log_pass "10. Audit Export" "$MS"
else
  log_fail "10. Audit Export" "$MS" "code=$CODE"
fi

# === Step 11: List Sessions ===
echo "--- Step 11: List Sessions ---"
RESP=$(tget "$GGID/api/v1/auth/sessions" "$TOKEN")
CODE=$(echo "$RESP" | cut -d'|' -f1)
MS=$(echo "$RESP" | cut -d'|' -f2)

if [ "$CODE" = "200" ]; then
  log_pass "11. List Sessions" "$MS"
else
  log_fail "11. List Sessions" "$MS" "code=$CODE"
fi

# === Summary ===
echo ""
echo "============================================"
echo "E2E Results: PASS=$PASS FAIL=$FAIL"
echo "============================================"

# Write report
REPORT="docs/test/e2e-business-flows-report.md"
mkdir -p docs/test
cat > "$REPORT" << EOF
# E2E Business Flows Report

**Date:** $(date -u '+%Y-%m-%d %H:%M:%S UTC')
**URL:** ${GGID}

## Results

| Status | Count |
|--------|-------|
| ✅ PASS | ${PASS} |
| ❌ FAIL | ${FAIL} |
| **Total** | **$((PASS + FAIL))** |

## Steps

| # | Step | Result | Latency |
|---|------|--------|---------|${RESULTS}

## Conclusion

$([ "$FAIL" -eq 0 ] && echo "All business flows passed. Platform is production-ready." || echo "${FAIL} flow(s) failed. See details above.")
EOF

echo "Report: $REPORT"
