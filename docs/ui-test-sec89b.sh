#!/bin/bash
BASE="https://ggid.iot2.win"
TENANT="00000000-0000-0000-0000-000000000001"

LOGIN=$(curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/auth/login" \
  -d '{"username":"uitest9901","password":"TestPass1234!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access_token') or d.get('token',''))" 2>/dev/null)

echo "=== Section 8: Security Center (alternative paths) ==="

echo "--- 8.1a GET /api/v1/audit/risk-score ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/risk-score"
echo ""

echo "--- 8.1b GET /api/v1/auth/risk-score ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/auth/risk-score"
echo ""

echo "--- 8.2a GET /api/v1/audit/threats ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/threats"
echo ""

echo "--- 8.2b GET /api/v1/auth/threats ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/auth/threats"
echo ""

echo "--- 8.3a GET /api/v1/audit/anomalies ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/anomalies"
echo ""

echo "--- 8.4a GET /api/v1/auth/sessions ---"
curl -s -o /dev/null -w '%{http_code}' -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/auth/sessions"
echo ""

echo "--- 8.4b GET /api/v1/auth/session ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/auth/session" | head -c 500
echo ""

echo ""
echo "=== Section 9: AI Agents (more tests) ==="

echo "--- 9.3 POST /api/v1/agents/register ---"
curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/agents/register" \
  -d '{"name":"test-agent","type":"service","scopes":["read:users"]}' | head -c 500
echo ""

echo "--- 9.4 GET /api/v1/agents (after register) ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/agents" | head -c 500
echo ""

echo ""
echo "=== DONE ==="
