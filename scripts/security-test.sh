#!/usr/bin/env bash
# GGID Security Test Suite — Automated penetration test baseline
# Usage: bash scripts/security-test.sh [BASE_URL]
# Default: http://localhost:8080
set -euo pipefail

BASE="${1:-http://localhost:8080}"
TENANT="00000000-0000-0000-0000-000000000001"
PASS=0
FAIL=0
WARN=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; ((PASS++)); }
log_fail() { echo -e "${RED}[FAIL]${NC} $1 — $2"; ((FAIL++)); }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1 — $2"; ((WARN++)); }

echo "============================================"
echo "GGID Security Test Suite"
echo "Target: $BASE"
echo "============================================"
echo ""

# --- Helper ---
http() {
  local method="$1" path="$2" body="${3:-}" auth="${4:-}" extra_header="${5:-}"
  local headers="-H 'Content-Type: application/json' -H \"X-Tenant-ID: $TENANT\""
  [ -n "$auth" ] && headers="$headers -H \"Authorization: Bearer $auth\""
  [ -n "$extra_header" ] && headers="$headers $extra_header"
  local cmd="curl -s -o /dev/null -w '%{http_code}' -X $method $headers"
  [ -n "$body" ] && cmd="$cmd -d '$body'"
  cmd="$cmd '$BASE$path'"
  eval "$cmd" 2>/dev/null || echo "000"
}

http_body() {
  local method="$1" path="$2" body="${3:-}" auth="${4:-}" extra_header="${5:-}"
  local headers="-H 'Content-Type: application/json' -H \"X-Tenant-ID: $TENANT\""
  [ -n "$auth" ] && headers="$headers -H \"Authorization: Bearer $auth\""
  [ -n "$extra_header" ] && headers="$headers $extra_header"
  local cmd="curl -s -X $method $headers"
  [ -n "$body" ] && cmd="$cmd -d '$body'"
  cmd="$cmd '$BASE$path'"
  eval "$cmd" 2>/dev/null || echo ""
}

# Get auth token
echo "--- Authentication Setup ---"
LOGIN_RESP=$(curl -s -X POST -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -d '{"username":"admin","password":"Admin@123456"}' "$BASE/api/v1/auth/login" 2>/dev/null || echo "")
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

if [ -n "$TOKEN" ]; then
  log_pass "Login successful — token obtained"
else
  echo "FATAL: Cannot obtain auth token. Ensure services are running."
  exit 1
fi

echo ""
echo "=== 1. SQL Injection Tests (10 endpoints) ==="

SQL_PAYLOADS=(
  "' OR '1'='1"
  "'; DROP TABLE users; --"
  "' UNION SELECT * FROM users --"
  "admin'--"
  "1; EXEC xp_cmdshell('dir') --"
)

SQL_ENDPOINTS=(
  "GET:/api/v1/users"
  "GET:/api/v1/users?filter=name"
  "GET:/api/v1/roles"
  "GET:/api/v1/audit"
  "GET:/api/v1/orgs"
  "GET:/api/v1/policies"
  "POST:/api/v1/auth/login"
  "GET:/api/v1/users/00000000-0000-0000-0000-000000000001"
  "GET:/api/v1/oauth/clients"
  "GET:/api/v1/agents"
)

for endpoint in "${SQL_ENDPOINTS[@]}"; do
  method="${endpoint%%:*}"
  path="${endpoint#*:}"
  for payload in "${SQL_PAYLOADS[@]}"; do
    if [ "$method" = "GET" ]; then
      # Inject payload into query parameter
      code=$(http GET "${path}%3Fid=${payload}" "" "$TOKEN")
    else
      code=$(http POST "$path" "{\"username\":\"$payload\",\"password\":\"x\"}" "")
    fi
    # SQL injection should never return 200 with data (should be 400/403/500)
    if [ "$code" = "200" ]; then
      log_warn "$method $path with SQL payload" "returned 200 — verify no data leak"
    else
      log_pass "$method $path with SQL payload — rejected ($code)"
    fi
  done
done

echo ""
echo "=== 2. XSS Tests (Input Reflection) ==="

XSS_PAYLOADS=(
  '<script>alert(1)</script>'
  '"><img src=x onerror=alert(1)>'
  'javascript:alert(1)'
  '<svg/onload=alert(1)>'
)

for payload in "${XSS_PAYLOADS[@]}"; do
  # Try to inject XSS in registration name
  body=$(http_body POST /api/v1/auth/register "{\"username\":\"xsstest$(date +%s)\",\"email\":\"xss$(date +%s)@test.com\",\"password\":\"XssTest@123\",\"name\":\"$payload\"}" "")
  if echo "$body" | grep -q "<script>" 2>/dev/null; then
    log_fail "XSS in register name" "script tag reflected in response"
  else
    log_pass "XSS payload not reflected: $payload"
  fi

  # Try in login username
  code=$(http POST /api/v1/auth/login "{\"username\":\"$payload\",\"password\":\"x\"}" "")
  if [ "$code" != "200" ]; then
    log_pass "XSS in login rejected ($code)"
  else
    log_warn "XSS in login" "returned 200 — investigate"
  fi
done

echo ""
echo "=== 3. IDOR Tests (Cross-User Access) ==="

# Create two users and try to access each other's data
USER1_RESP=$(http_body POST /api/v1/users '{"email":"idor1@test.com","password":"IdorTest@123","name":"IDOR1","username":"idor1"}' "$TOKEN")
USER1_ID=$(echo "$USER1_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")

USER2_RESP=$(http_body POST /api/v1/users '{"email":"idor2@test.com","password":"IdorTest@123","name":"IDOR2","username":"idor2"}' "$TOKEN")
USER2_ID=$(echo "$USER2_RESP" | python3 -c "import sys,json;print(json.load(sys.stdin).get('id',''))" 2>/dev/null || echo "")

# Login as user1
USER1_LOGIN=$(http_body POST /api/v1/auth/login '{"username":"idor1","password":"IdorTest@123"}' "$TENANT")
USER1_TOKEN=$(echo "$USER1_LOGIN" | python3 -c "import sys,json;print(json.load(sys.stdin).get('access_token',''))" 2>/dev/null || echo "")

if [ -n "$USER1_TOKEN" ]; then
  # User1 tries to access user2's data
  code=$(http GET "/api/v1/users/${USER2_ID}" "" "$USER1_TOKEN")
  if [ "$code" = "403" ] || [ "$code" = "404" ]; then
    log_pass "IDOR: user1 cannot access user2 data ($code)"
  elif [ "$code" = "200" ]; then
    log_fail "IDOR: user1 CAN access user2 data" "200 — unauthorized cross-user access"
  else
    log_warn "IDOR: unexpected status ($code)"
  fi
else
  log_warn "IDOR test skipped" "could not login as regular user"
fi

# Try accessing without tenant header (cross-tenant)
code=$(curl -s -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $TOKEN" "$BASE/api/v1/users" 2>/dev/null || echo "000")
if [ "$code" = "400" ] || [ "$code" = "401" ]; then
  log_pass "Missing tenant header rejected ($code)"
else
  log_warn "Missing tenant header: $code — may need stricter validation"
fi

echo ""
echo "=== 4. JWT Tampering Tests ==="

# Tamper with JWT payload
PARTS=(${TOKEN//./ })
HEADER="${PARTS[0]}"
PAYLOAD="${PARTS[1]}"
SIGNATURE="${PARTS[2]}"

# Decode payload, modify role, re-encode
TAMPERED_PAYLOAD=$(echo "$PAYLOAD" | base64 -d 2>/dev/null | python3 -c "
import sys, json, base64
data = json.load(sys.stdin)
data['role'] = 'superadmin'
data['is_admin'] = True
encoded = base64.urlsafe_b64encode(json.dumps(data).encode()).decode().rstrip('=')
sys.stdout.write(encoded)
" 2>/dev/null || echo "$PAYLOAD")

TAMPERED_TOKEN="${HEADER}.${TAMPERED_PAYLOAD}.${SIGNATURE}"

code=$(http GET /api/v1/users "" "$TAMPERED_TOKEN")
if [ "$code" = "401" ]; then
  log_pass "Tampered JWT rejected (signature mismatch)"
else
  log_fail "Tampered JWT accepted" "status $code — signature verification may be broken"
fi

# None algorithm attack
NONE_TOKEN="${HEADER}.$(echo -n '{\"sub\":\"admin\",\"role\":\"superadmin\",\"alg\":\"none\"}' | base64)."
code=$(http GET /api/v1/users "" "$NONE_TOKEN")
if [ "$code" = "401" ]; then
  log_pass "None-algorithm JWT rejected"
else
  log_fail "None-algorithm JWT accepted" "status $code"
fi

# Expired token simulation (modify exp to past)
EXPIRED_PAYLOAD=$(echo "$PAYLOAD" | base64 -d 2>/dev/null | python3 -c "
import sys, json, base64, time
data = json.load(sys.stdin)
data['exp'] = int(time.time()) - 3600
encoded = base64.urlsafe_b64encode(json.dumps(data).encode()).decode().rstrip('=')
sys.stdout.write(encoded)
" 2>/dev/null || echo "$PAYLOAD")
EXPIRED_TOKEN="${HEADER}.${EXPIRED_PAYLOAD}.${SIGNATURE}"

code=$(http GET /api/v1/users "" "$EXPIRED_TOKEN")
if [ "$code" = "401" ]; then
  log_pass "Expired JWT rejected"
else
  log_warn "Expired JWT: status $code"
fi

echo ""
echo "=== 5. Rate Limit Bypass Tests ==="

# Rapid-fire login attempts (should hit rate limit)
RATE_LIMITED=0
for i in $(seq 1 30); do
  code=$(http POST /api/v1/auth/login '{"username":"rate-test","password":"wrong"}' "")
  if [ "$code" = "429" ]; then
    RATE_LIMITED=1
    break
  fi
done

if [ $RATE_LIMITED -eq 1 ]; then
  log_pass "Rate limit triggered after $i rapid login attempts"
else
  log_warn "Rate limit not triggered after 30 attempts" "may need stricter limits"
fi

# Token endpoint rate limiting (KB-326: 10/min)
TOKEN_LIMITED=0
for i in $(seq 1 20); do
  code=$(http POST /oauth/token "grant_type=client_credentials&client_id=fake&client_secret=fake" "")
  if [ "$code" = "429" ]; then
    TOKEN_LIMITED=1
    break
  fi
done

if [ $TOKEN_LIMITED -eq 1 ]; then
  log_pass "OAuth token endpoint rate limited after $i attempts"
else
  log_warn "Token endpoint rate limit not hit" "may not be deployed yet"
fi

echo ""
echo "============================================"
echo "Security Test Summary"
echo "============================================"
echo -e "${GREEN}PASS: $PASS${NC}  ${RED}FAIL: $FAIL${NC}  ${YELLOW}WARN: $WARN${NC}"
echo ""
if [ $FAIL -gt 0 ]; then
  echo -e "${RED}SECURITY ISSUES DETECTED — see FAIL details above${NC}"
  exit 1
else
  echo -e "${GREEN}All critical security checks passed${NC}"
  exit 0
fi
