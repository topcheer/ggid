#!/bin/bash
BASE="https://ggid.iot2.win"
TENANT="00000000-0000-0000-0000-000000000001"

LOGIN=$(curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/auth/login" \
  -d '{"username":"uitest9901","password":"TestPass1234!"}')
TOKEN=$(echo "$LOGIN" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('access_token') or d.get('token',''))" 2>/dev/null)

echo "=== Security Center: 400 endpoints with params ==="

echo "--- 8.1 GET /api/v1/audit/risk-score?user_id=e6ae0dd2-5009-42e6-aba1-0c8fa4744608 ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/audit/risk-score?user_id=e6ae0dd2-5009-42e6-aba1-0c8fa4744608" | head -c 500
echo ""

echo "--- 8.4 GET /api/v1/auth/sessions?user_id=e6ae0dd2-5009-42e6-aba1-0c8fa4744608 ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/auth/sessions?user_id=e6ae0dd2-5009-42e6-aba1-0c8fa4744608" | head -c 500
echo ""

echo ""
echo "=== AI Agents: register with owner_user_id ==="

echo "--- 9.3 POST /api/v1/agents/register (with owner) ---"
curl -s -H 'Accept-Encoding: identity' -H 'Content-Type: application/json' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  -X POST "$BASE/api/v1/agents/register" \
  -d '{"name":"test-agent","type":"service","scopes":["read:users"],"owner_user_id":"e6ae0dd2-5009-42e6-aba1-0c8fa4744608"}' | head -c 500
echo ""

echo "--- 9.4 GET /api/v1/agents (after register) ---"
curl -s -H 'Accept-Encoding: identity' -H "Authorization: Bearer $TOKEN" -H "X-Tenant-ID: $TENANT" \
  "$BASE/api/v1/agents" | head -c 500
echo ""

echo ""
echo "=== DONE ==="
