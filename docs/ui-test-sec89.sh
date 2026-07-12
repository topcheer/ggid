#!/bin/bash
BASE="https://ggid.iot2.win"
TENANT="00000000-0000-0000-0000-000000000001"

LOGIN=$(curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/auth/login" \
  -d '{"username":"uitest9901","password":"TestPass1234!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access_token') or d.get('token',''))" 2>/dev/null)
echo "TOKEN: ${#TOKEN} chars"

echo ""
echo "=== Section 8: Security Center ==="

echo "--- 8.1 GET /api/v1/security/risk-score ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/security/risk-score" | head -c 500
echo ""

echo "--- 8.2 GET /api/v1/security/threats ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/security/threats" | head -c 500
echo ""

echo "--- 8.3 GET /api/v1/security/anomalies ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/security/anomalies" | head -c 500
echo ""

echo "--- 8.4 GET /api/v1/security/sessions ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/security/sessions" | head -c 500
echo ""

echo ""
echo "=== Section 9: AI Agents ==="

echo "--- 9.1 GET /api/v1/agents ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/agents" | head -c 500
echo ""

echo "--- 9.2 GET /api/v1/agents/list ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/agents/list" | head -c 500
echo ""

echo ""
echo "=== DONE ==="
